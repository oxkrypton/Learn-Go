package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/infrastructure/cache"
	"go-redis/internal/model"
	"go-redis/internal/pkg/bizerr"
	"go-redis/internal/repository"
	"go-redis/internal/script"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
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
	db   *gorm.DB
}

func NewVoucherService(repo repository.VoucherRepository, rdb *redis.Client, db *gorm.DB) VoucherService {
	return &voucherService{repo: repo, rdb: rdb, db: db}
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
			return errors.New("seckill voucher requires beginTime and endTime")
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

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherId uint64, userId uint64) (int64, error) {
	voucher, err := s.repo.QueryVoucherById(ctx, voucherId)
	if err != nil {
		return 0, fmt.Errorf("query voucher failed: %w", err)
	}
	if voucher == nil {
		return 0, bizerr.New("voucher not found")
	}

	seckill, err := s.repo.QuerySeckillVoucherById(ctx, voucherId)
	if err != nil {
		return 0, fmt.Errorf("query seckill voucher failed: %w", err)
	}
	if seckill == nil {
		return 0, bizerr.New("this is not a seckill voucher")
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

	ret, err := script.RunSeckillLua(ctx, s.rdb, voucherId, userId)
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
		UserID:    userId,
		VoucherID: voucherId,
	}
	if err := s.enqueueSeckillOrder(ctx, msg); err != nil {
		return 0, fmt.Errorf("enqueue order failed: %w", err)
	}

	return orderID, nil
}

func (s *voucherService) enqueueSeckillOrder(ctx context.Context, msg dto.SeckillOrderMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return s.rdb.LPush(ctx, constant.SeckillOrderQueueKey, body).Err()
}

func (s *voucherService) StartOrderConsumer(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("stop order consumer")
			return
		default:
		}

		result, err := s.rdb.BRPop(ctx, 2*time.Second, constant.SeckillOrderQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			if ctx.Err() != nil {
				log.Printf("stop order consumer")
				return
			}
			log.Printf("consume seckill order failed: %v", err)
			continue
		}

		if len(result) != 2 {
			continue
		}

		if err := s.handleOrderMessage(ctx, result[1]); err != nil {
			log.Printf("handle seckill order failed: %v", err)
		}
	}
}

func (s *voucherService) handleOrderMessage(ctx context.Context, raw string) error {
	var msg dto.SeckillOrderMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return fmt.Errorf("unmarshal seckill order message failed, raw=%s: %w", raw, err)
	}

	if msg.OrderID <= 0 || msg.UserID == 0 || msg.VoucherID == 0 {
		return fmt.Errorf(
			"invalid seckill order message, orderId=%d userId=%d voucherId=%d raw=%s",
			msg.OrderID, msg.UserID, msg.VoucherID, raw,
		)
	}

	if err := s.createVoucherOrderAsync(ctx, msg); err != nil {
		return fmt.Errorf(
			"process seckill order message failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}

	return nil
}

func (s *voucherService) createVoucherOrderAsync(ctx context.Context, msg dto.SeckillOrderMessage) error {
	count, err := s.repo.CountByUserIdAndVoucherId(ctx, msg.UserID, msg.VoucherID)
	if err != nil {
		return fmt.Errorf(
			"check existing order failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}
	if count > 0 {
		log.Printf(
			"skip duplicated order message, orderId=%d userId=%d voucherId=%d",
			msg.OrderID, msg.UserID, msg.VoucherID,
		)
		return nil
	}

	if err := s.repo.DeductStock(ctx, msg.VoucherID); err != nil {
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
	if err := s.repo.CreateVoucherOrder(ctx, order); err != nil {
		return fmt.Errorf(
			"create voucher order failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}

	return nil
}
