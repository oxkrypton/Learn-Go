package repository_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"go-redis/internal/config"
	"go-redis/internal/infrastructure/database"
	"go-redis/internal/model"
	"go-redis/internal/repository"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestVoucherRepositoryWithTxRollsBackStockWhenCreateOrderFails(t *testing.T) {
	t.Helper()

	db := openRepositoryTestDB(t)
	repo := repository.NewVoucherRepository(db)
	ctx := context.Background()

	const (
		voucherID      uint64 = 13
		existingOrder  int64  = 910000000000001
		conflictOrder  int64  = existingOrder
		existingUserID uint64 = 41001
		testUserID     uint64 = 41002
	)

	originalStock := mustQueryStock(t, db, voucherID)
	t.Cleanup(func() {
		mustDeleteOrdersByIDs(t, db, existingOrder)
		mustDeleteOrdersByUsers(t, db, voucherID, existingUserID, testUserID)
		mustSetStock(t, db, voucherID, originalStock)
	})

	// 先把相关脏数据清掉，再把库存压成 1，方便精确观察事务有没有回滚。
	mustDeleteOrdersByIDs(t, db, existingOrder)
	mustDeleteOrdersByUsers(t, db, voucherID, existingUserID, testUserID)
	mustSetStock(t, db, voucherID, 1)

	// 先插入一条固定主键的订单，后面再用同样的 id 下单，强行制造插入失败。
	mustInsertOrder(t, db, &model.VoucherOrder{
		ID:        existingOrder,
		UserID:    existingUserID,
		VoucherID: voucherID,
	})

	err := repo.WithTx(ctx, func(txRepo repository.VoucherRepository) error {
		if err := txRepo.DeductStock(ctx, voucherID); err != nil {
			return err
		}

		return txRepo.CreateVoucherOrder(ctx, &model.VoucherOrder{
			ID:        conflictOrder,
			UserID:    testUserID,
			VoucherID: voucherID,
		})
	})
	if err == nil {
		t.Fatal("expected CreateVoucherOrder to fail, but transaction returned nil")
	}

	if got := mustQueryStock(t, db, voucherID); got != 1 {
		t.Fatalf("stock should roll back to 1 after failed insert, got %d", got)
	}

	if got := mustCountOrdersByUser(t, db, voucherID, testUserID); got != 0 {
		t.Fatalf("failed transaction should not create test user's order, got %d rows", got)
	}
}

func openRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := loadIntegrationConfig(t)
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

	return db
}

func loadIntegrationConfig(t *testing.T) *config.Config {
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

func mustQueryStock(t *testing.T, db *gorm.DB, voucherID uint64) int32 {
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

func mustSetStock(t *testing.T, db *gorm.DB, voucherID uint64, stock int32) {
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

func mustInsertOrder(t *testing.T, db *gorm.DB, order *model.VoucherOrder) {
	t.Helper()

	if err := db.WithContext(context.Background()).Create(order).Error; err != nil {
		t.Fatalf("insert order %+v failed: %v", order, err)
	}
}

func mustDeleteOrdersByIDs(t *testing.T, db *gorm.DB, orderIDs ...int64) {
	t.Helper()

	if len(orderIDs) == 0 {
		return
	}

	if err := db.WithContext(context.Background()).
		Where("id IN ?", orderIDs).
		Delete(&model.VoucherOrder{}).Error; err != nil {
		t.Fatalf("delete orders by id failed: %v", err)
	}
}

func mustDeleteOrdersByUsers(t *testing.T, db *gorm.DB, voucherID uint64, userIDs ...uint64) {
	t.Helper()

	if len(userIDs) == 0 {
		return
	}

	if err := db.WithContext(context.Background()).
		Where("voucher_id = ? AND user_id IN ?", voucherID, userIDs).
		Delete(&model.VoucherOrder{}).Error; err != nil {
		t.Fatalf("delete orders by user failed: %v", err)
	}
}

func mustCountOrdersByUser(t *testing.T, db *gorm.DB, voucherID uint64, userID uint64) int64 {
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
