package service

import (
	"context"
	"go-redis/internal/dto"
	"go-redis/internal/model"
	"go-redis/internal/repository"
)

type VoucherService interface {
	QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]dto.VoucherDTO, error)
	AddVoucher(ctx context.Context, voucher *model.Voucher) error
}

type voucherService struct {
	repo repository.VoucherRepository
}

func NewVoucherService(repo repository.VoucherRepository) VoucherService {
	return &voucherService{repo: repo}
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

func (s *voucherService) AddVoucher(ctx context.Context, voucher *model.Voucher) error {
	return s.repo.CreateVoucher(ctx, voucher)
}
