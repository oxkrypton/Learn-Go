package service

import (
	"context"
	"errors"
	"fmt"
	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/infrastructure/cache"
	"go-redis/internal/infrastructure/mq"
	"go-redis/internal/model"
	"go-redis/internal/pkg/bizerr"
	"go-redis/internal/repository"
	"go-redis/internal/script"
	"log"
	"strconv"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type VoucherService interface {
	QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]dto.VoucherDTO, error)
	AddVoucher(ctx context.Context, v *dto.VoucherDTO) error
	SeckillVoucher(ctx context.Context, voucherId uint64, userId uint64) (int64, error)
	StartOrderConsumer(ctx context.Context)
}

type voucherService struct {
	repo repository.VoucherRepository
	rdb  *redis.Client
	mq   mq.SeckillOrderQueue
}

func NewVoucherService(repo repository.VoucherRepository, rdb *redis.Client, mq mq.SeckillOrderQueue) VoucherService {
	return &voucherService{repo: repo, rdb: rdb, mq: mq}
}

func (s *voucherService) QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]dto.VoucherDTO, error) {
	vouchers, err := s.repo.QueryVouchersByShopId(ctx, shopId)
	if err != nil {
		return nil, err
	}

	result := make([]dto.VoucherDTO, 0, len(vouchers))
	for _, v := range vouchers {
		d := dto.VoucherDTO{
			ID: v.ID, ShopID: v.ShopID, Title: v.Title,
			SubTitle: v.SubTitle, Rules: v.Rules,
			PayValue: v.PayValue, ActualValue: v.ActualValue,
			Type: v.Type, Status: v.Status,
		}
		if v.Type == 1 {
			sk, err := s.repo.QuerySeckillVoucherById(ctx, v.ID)
			if err == nil && sk != nil {
				d.Stock = sk.Stock
				d.BeginTime = &sk.BeginTime
				d.EndTime = &sk.EndTime
			}
		}
		result = append(result, d)
	}
	return result, nil
}

func (s *voucherService) AddVoucher(ctx context.Context, v *dto.VoucherDTO) error {
	voucher := &model.Voucher{
		ShopID: v.ShopID, Title: v.Title, SubTitle: v.SubTitle,
		Rules: v.Rules, PayValue: v.PayValue, ActualValue: v.ActualValue,
		Type: v.Type, Status: v.Status,
	}
	if err := s.repo.CreateVoucher(ctx, voucher); err != nil {
		return err
	}
	v.ID = voucher.ID

	if v.Type == 1 {
		if v.BeginTime == nil || v.EndTime == nil {
			return bizerr.New("seckill voucher requires begin time and end time")
		}
		sv := &model.SeckillVoucher{
			VoucherID: voucher.ID,
			Stock:     v.Stock,
			BeginTime: *v.BeginTime,
			EndTime:   *v.EndTime,
		}
		if err := s.repo.CreateSeckillVoucher(ctx, sv); err != nil {
			return err
		}

		stockKey := constant.SeckillStockKey + strconv.FormatUint(voucher.ID, 10)
		if err := cache.Set(s.rdb, ctx, stockKey, v.Stock, 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherID uint64, userID uint64) (int64, error) {
	voucher, err := s.repo.QueryVoucherById(ctx, voucherID)
	if err != nil {
		return 0, fmt.Errorf("query voucher failed: %w", err)
	}
	if voucher == nil {
		return 0, bizerr.New("voucher not found")
	}

	seckill, err := s.repo.QuerySeckillVoucherById(ctx, voucherID)
	if err != nil {
		return 0, fmt.Errorf("query seckill voucher failed: %w", err)
	}
	if seckill == nil {
		return 0, bizerr.New("voucher is not a seckill voucher")
	}

	now := time.Now()
	if now.Before(seckill.BeginTime) {
		return 0, bizerr.New("seckill is not started")
	}
	if now.After(seckill.EndTime) {
		return 0, bizerr.New("seckill is ended")
	}

	orderID, err := cache.NextID(ctx, s.rdb, "order")
	if err != nil {
		return 0, fmt.Errorf("generate order id failed: %w", err)
	}

	ret, err := script.RunSeckillLua(ctx, s.rdb, voucherID, userID)
	if err != nil {
		return 0, fmt.Errorf("run lua failed: %w", err)
	}

	switch ret {
	case 1:
		return 0, bizerr.New("stock is not enough")
	case 2:
		return 0, bizerr.New("each user can only order once")
	}

	msg := dto.SeckillOrderMessage{
		OrderID:   orderID,
		UserID:    userID,
		VoucherID: voucherID,
	}

	if err := s.enqueueSeckillOrder(ctx, msg); err != nil {
		return 0, fmt.Errorf("enqueue order failed: %w", err)
	}

	return orderID, nil
}

func (s *voucherService) enqueueSeckillOrder(ctx context.Context, msg dto.SeckillOrderMessage) error {
	if err := s.mq.Publish(ctx, msg); err != nil {
		if compensateErr := s.compensateSeckillPreDeduct(ctx, msg); compensateErr != nil {
			log.Printf(
				"compensate seckill pre-deduct failed, orderId=%d userId=%d voucherId=%d err=%v",
				msg.OrderID, msg.UserID, msg.VoucherID, compensateErr,
			)
		}
		return err
	}

	return nil
}

func (s *voucherService) compensateSeckillPreDeduct(ctx context.Context, msg dto.SeckillOrderMessage) error {
	return script.RunCompensateSeckillLua(ctx, s.rdb, msg.VoucherID, msg.UserID)
}

// 消费者事务
func (s *voucherService) StartOrderConsumer(ctx context.Context) {
	if err := s.mq.Consume(ctx, s.handleOrderMessage); err != nil {
		if ctx.Err() != nil {
			log.Printf("stop order consumer!")
			return
		}
		log.Printf("consume seckill order failed: %v", err)
	}
}

func (s *voucherService) handleOrderMessage(ctx context.Context, msg dto.SeckillOrderMessage) error {
	if msg.OrderID <= 0 || msg.UserID == 0 || msg.VoucherID == 0 {
		return fmt.Errorf(
			"invalid seckill order message, orderId=%d userId=%d voucherId=%d",
			msg.OrderID,
			msg.UserID,
			msg.VoucherID,
		)
	}
	if err := s.AsyncCreateVoucherOrder(ctx, msg); err != nil {
		return fmt.Errorf(
			"process seckill order message failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID,
			msg.UserID,
			msg.VoucherID,
			err,
		)
	}

	return nil
}

// 查重-扣减库存-订单落库
func (s *voucherService) AsyncCreateVoucherOrder(ctx context.Context, msg dto.SeckillOrderMessage) error {
	//使用事务化将三次数据库操作汇总，原子化
	err := s.repo.WithTx(ctx, func(txRepo repository.VoucherRepository) error {
		//根据userid、voucherid查询订单
		count, err := txRepo.CountByUserIdAndVoucherId(ctx, msg.UserID, msg.VoucherID)
		if err != nil {
			return fmt.Errorf(
				"check existing order failed, orderId=%d userId=%d voucherId=%d: %w",
				msg.OrderID, msg.UserID, msg.VoucherID, err,
			)
		}
		if count > 0 {
			return nil
		}

		if err := txRepo.DeductStock(ctx, msg.VoucherID); err != nil {
			return fmt.Errorf(
				"deduct stock failed, orderId=%d userId=%d voucherId=%d: %w",
				msg.OrderID, msg.UserID, msg.VoucherID, err,
			)
		}

		order := &model.VoucherOrder{
			ID:        msg.OrderID,
			UserID:    msg.UserID,
			VoucherID: msg.VoucherID,
		}
		if err := txRepo.CreateVoucherOrder(ctx, order); err != nil {
			//判断是否唯一索引冲突
			var mysqlErr *gomysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				return nil
			}
			return fmt.Errorf(
				"create voucher order failed, orderId=%d userId=%d voucherId=%d: %w",
				msg.OrderID, msg.UserID, msg.VoucherID, err,
			)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("create voucher order async failed: %w", err)
	}
	return nil
}
