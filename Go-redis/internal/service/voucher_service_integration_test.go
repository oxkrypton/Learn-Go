package service_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go-redis/internal/config"
	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/infrastructure/database"
	"go-redis/internal/model"
	"go-redis/internal/repository"
	"go-redis/internal/service"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestVoucherServiceConsumerIgnoresDuplicateMessage(t *testing.T) {
	db, rdb := openServiceTestDeps(t)
	svc := service.NewVoucherService(repository.NewVoucherRepository(db), rdb)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		voucherID uint64 = 13
		userID    uint64 = 42001
		orderID   int64  = 920000000000001
	)

	originalStock := mustQueryServiceStock(t, db, voucherID)
	t.Cleanup(func() {
		mustClearQueue(t, rdb)
		mustDeleteServiceOrders(t, db, voucherID, userID)
		mustSetServiceStock(t, db, voucherID, originalStock)
	})

	// 这个场景只想验证“同一条消息重复投递两次”不会被重复扣库存。
	mustClearQueue(t, rdb)
	mustDeleteServiceOrders(t, db, voucherID, userID)
	mustSetServiceStock(t, db, voucherID, 1)

	go svc.StartOrderConsumer(ctx)

	raw := mustMarshalMessage(t, dto.SeckillOrderMessage{
		OrderID:   orderID,
		UserID:    userID,
		VoucherID: voucherID,
	})

	if err := rdb.LPush(ctx, constant.SeckillOrderQueueKey, raw, raw).Err(); err != nil {
		t.Fatalf("push duplicate messages failed: %v", err)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		return mustQueueLen(t, rdb) == 0 && mustCountServiceOrders(t, db, voucherID, userID) == 1
	})

	if got := mustQueryServiceStock(t, db, voucherID); got != 0 {
		t.Fatalf("duplicate message should deduct stock only once, got stock=%d", got)
	}

	if got := mustCountServiceOrders(t, db, voucherID, userID); got != 1 {
		t.Fatalf("duplicate message should create exactly one order, got %d", got)
	}
}

func TestVoucherServiceConsumerKeepsStockWhenOrderAlreadyExists(t *testing.T) {
	db, rdb := openServiceTestDeps(t)
	svc := service.NewVoucherService(repository.NewVoucherRepository(db), rdb)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const (
		voucherID uint64 = 13
		userID    uint64 = 42002
		orderID   int64  = 920000000000002
	)

	originalStock := mustQueryServiceStock(t, db, voucherID)
	t.Cleanup(func() {
		mustClearQueue(t, rdb)
		mustDeleteServiceOrders(t, db, voucherID, userID)
		mustSetServiceStock(t, db, voucherID, originalStock)
	})

	// 先放一条现有订单，再重放同一条消息，预期消费者直接跳过，不改库存。
	mustClearQueue(t, rdb)
	mustDeleteServiceOrders(t, db, voucherID, userID)
	mustSetServiceStock(t, db, voucherID, 1)
	mustInsertServiceOrder(t, db, &model.VoucherOrder{
		ID:        orderID,
		UserID:    userID,
		VoucherID: voucherID,
	})

	go svc.StartOrderConsumer(ctx)

	raw := mustMarshalMessage(t, dto.SeckillOrderMessage{
		OrderID:   orderID,
		UserID:    userID,
		VoucherID: voucherID,
	})
	if err := rdb.LPush(ctx, constant.SeckillOrderQueueKey, raw).Err(); err != nil {
		t.Fatalf("push replay message failed: %v", err)
	}

	waitForCondition(t, 5*time.Second, func() bool {
		return mustQueueLen(t, rdb) == 0
	})

	if got := mustQueryServiceStock(t, db, voucherID); got != 1 {
		t.Fatalf("existing order replay should not change stock, got stock=%d", got)
	}

	if got := mustCountServiceOrders(t, db, voucherID, userID); got != 1 {
		t.Fatalf("existing order replay should keep exactly one order, got %d", got)
	}
}

func openServiceTestDeps(t *testing.T) (*gorm.DB, *redis.Client) {
	t.Helper()

	cfg := loadServiceIntegrationConfig(t)

	db, err := database.InitMysql(cfg.MySQL)
	if err != nil {
		t.Skipf("skip integration test because mysql is unavailable: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB failed: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	rdb, err := database.InitRedis(cfg.Redis)
	if err != nil {
		t.Skipf("skip integration test because redis is unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	return db, rdb
}

func loadServiceIntegrationConfig(t *testing.T) *config.Config {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current test file failed")
	}

	configPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "config", "config.yaml")
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("read config file %s failed: %v", configPath, err)
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		t.Fatalf("unmarshal config failed: %v", err)
	}

	return &cfg
}

func mustMarshalMessage(t *testing.T, msg dto.SeckillOrderMessage) string {
	t.Helper()

	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal message failed: %v", err)
	}

	return string(body)
}

func mustClearQueue(t *testing.T, rdb *redis.Client) {
	t.Helper()

	if err := rdb.Del(context.Background(), constant.SeckillOrderQueueKey).Err(); err != nil {
		t.Fatalf("clear queue failed: %v", err)
	}
}

func mustQueueLen(t *testing.T, rdb *redis.Client) int64 {
	t.Helper()

	size, err := rdb.LLen(context.Background(), constant.SeckillOrderQueueKey).Result()
	if err != nil {
		t.Fatalf("query queue length failed: %v", err)
	}

	return size
}

func mustSetServiceStock(t *testing.T, db *gorm.DB, voucherID uint64, stock int32) {
	t.Helper()

	result := db.WithContext(context.Background()).
		Model(&model.SeckillVoucher{}).
		Where("voucher_id = ?", voucherID).
		Updates(map[string]any{
			"stock":       stock,
			"update_time": time.Now(),
		})
	if result.Error != nil {
		t.Fatalf("set stock failed: %v", result.Error)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("set stock affected %d rows, want 1", result.RowsAffected)
	}
}

func mustQueryServiceStock(t *testing.T, db *gorm.DB, voucherID uint64) int32 {
	t.Helper()

	var stock int32
	if err := db.WithContext(context.Background()).
		Model(&model.SeckillVoucher{}).
		Select("stock").
		Where("voucher_id = ?", voucherID).
		Scan(&stock).Error; err != nil {
		t.Fatalf("query stock failed: %v", err)
	}

	return stock
}

func mustDeleteServiceOrders(t *testing.T, db *gorm.DB, voucherID uint64, userID uint64) {
	t.Helper()

	if err := db.WithContext(context.Background()).
		Where("voucher_id = ? AND user_id = ?", voucherID, userID).
		Delete(&model.VoucherOrder{}).Error; err != nil {
		t.Fatalf("delete test orders failed: %v", err)
	}
}

func mustInsertServiceOrder(t *testing.T, db *gorm.DB, order *model.VoucherOrder) {
	t.Helper()

	if err := db.WithContext(context.Background()).Create(order).Error; err != nil {
		t.Fatalf("insert existing order failed: %v", err)
	}
}

func mustCountServiceOrders(t *testing.T, db *gorm.DB, voucherID uint64, userID uint64) int64 {
	t.Helper()

	var count int64
	if err := db.WithContext(context.Background()).
		Model(&model.VoucherOrder{}).
		Where("voucher_id = ? AND user_id = ?", voucherID, userID).
		Count(&count).Error; err != nil {
		t.Fatalf("count orders failed: %v", err)
	}

	return count
}

func waitForCondition(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("condition was not satisfied before timeout")
}
