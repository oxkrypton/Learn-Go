package service

import (
	"context"
	"errors"
	"fmt"
	"go-redis/internal/dto"
	"go-redis/internal/model"
	"go-redis/internal/repository"
	"go-redis/internal/utils"
	"time"

	"github.com/redis/go-redis/v9"
)

type VoucherService interface {
	QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]dto.VoucherDTO, error)
	AddVoucher(ctx context.Context, v *dto.VoucherDTO) error
	//SeckillVoucher 秒杀下单
	SeckillVoucher(ctx context.Context, voucherId uint64, userId uint64) (int64, error)
}

type voucherService struct {
	repo repository.VoucherRepository
	rdb  *redis.Client
}

func NewVoucherService(repo repository.VoucherRepository, rdb *redis.Client) VoucherService {
	return &voucherService{repo: repo, rdb: rdb}
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
		if v.Type == 1 { //秒杀券，查秒杀信息
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
	// 1. 写入 tb_voucher
	voucher := &model.Voucher{
		ShopID: v.ShopID, Title: v.Title, SubTitle: v.SubTitle,
		Rules: v.Rules, PayValue: v.PayValue, ActualValue: v.ActualValue,
		Type: v.Type, Status: v.Status,
	}
	if err := s.repo.CreateVoucher(ctx, voucher); err != nil {
		return err
	}
	v.ID = voucher.ID // 回写自增ID

	// 2. 若为秒杀券(type==1)，额外写入 tb_seckill_voucher
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
	}
	return nil
}

func (s *voucherService) SeckillVoucher(ctx context.Context, voucherId uint64, userId uint64) (int64, error) {
	//1.查询优惠券基础信息
	voucher, err := s.repo.QueryVoucherById(ctx, voucherId)
	if err != nil {
		return 0, fmt.Errorf("Fail to query voucher:%w", err)
	}
	if voucher == nil {
		return 0, errors.New("voucher not found")
	}

	//2.查询秒杀券信息 (库存、开始/结束时间)
	seckill, err := s.repo.QuerySeckillVoucherById(ctx, voucherId)
	if err != nil {
		return 0, fmt.Errorf("Fail to query seckill_info:%w", err)
	}
	if seckill == nil {
		return 0, errors.New("This is not a seckill voucher")
	}

	//3.判断秒杀是否开始
	now := time.Now()
	if now.Before(seckill.BeginTime) {
		return 0, errors.New("seckill is not started")
	}
	// 判断秒杀是否已经结束
	if now.After(seckill.EndTime) {
		return 0, errors.New("seckill is ended")
	}

	//4.判断库存是否充足
	if seckill.Stock < 1 {
		return 0, errors.New("stock is not enough")
	}

	//5.扣减库存 (CAS乐观锁: stock > 0)
	if err := s.repo.DeductStock(ctx, voucherId); err != nil {
		return 0, fmt.Errorf("fail to DeductStock:%w", err)
	}

	//6.创建订单 — 使用 Redis 全局唯一ID
	orderId, err := utils.NextID(ctx, s.rdb, "order")
	if err != nil {
		return 0, fmt.Errorf("fail to NextID:%w", err)
	}

	order := &model.VoucherOrder{
		ID:        orderId,
		UserID:    userId,
		VoucherID: voucherId,
	}
	if err := s.repo.CreateVoucherOrder(ctx, order); err != nil {
		return 0, fmt.Errorf("Fail to create order:%w", err)
	}

	//7.返回订单ID
	return orderId, nil
}
