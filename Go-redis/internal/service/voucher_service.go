package service

import (
	"context"
	"go-redis/internal/model"
	"go-redis/internal/repository"
)

type VoucherService interface {
	QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]model.Voucher, error)
}

type voucherService struct {
	repo repository.VoucherRepository
}

func NewVoucherService(repo repository.VoucherRepository) VoucherService {
	return &voucherService{repo: repo}
}

func (s *voucherService) QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]model.Voucher, error) {
	return s.repo.QueryVouchersByShopId(ctx, shopId)
}
