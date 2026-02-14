package repository

import (
	"context"
	"errors"
	"go-redis/internal/model"

	"gorm.io/gorm"
)

// VoucherRepository 定义优惠券及秒杀相关接口
type VoucherRepository interface {
	// QueryVouchersByShopId 查询店铺下的优惠券列表
	QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]model.Voucher, error)
	// QueryVoucherById 查询优惠券基础信息
	QueryVoucherById(ctx context.Context, id uint64) (*model.Voucher, error)
	// QuerySeckillVoucherById 查询秒杀优惠券详情(库存、时间)
	QuerySeckillVoucherById(ctx context.Context, voucherId uint64) (*model.SeckillVoucher, error)
	
	// DeductStock 扣减库存 (核心：CAS乐观锁，确保 stock > 0)
	DeductStock(ctx context.Context, voucherId uint64) error
	
	// CreateVoucherOrder 创建秒杀订单
	CreateVoucherOrder(ctx context.Context, order *model.VoucherOrder) error
	// CountByUserIdAndVoucherId 查询用户是否已购买过该券 (用于一人一单校验)
	CountByUserIdAndVoucherId(ctx context.Context, userId uint64, voucherId uint64) (int64, error)
}

type voucherRepository struct {
	db *gorm.DB
}

func NewVoucherRepository(db *gorm.DB) VoucherRepository {
	return &voucherRepository{db: db}
}

func (r *voucherRepository) QueryVouchersByShopId(ctx context.Context, shopId uint64) ([]model.Voucher, error) {
	var vouchers []model.Voucher
	// 查询该店铺下所有上架(status=1)的优惠券
	err := r.db.WithContext(ctx).
		Where("shop_id = ? AND status = 1", shopId).
		Find(&vouchers).Error
	return vouchers, err
}

func (r *voucherRepository) QueryVoucherById(ctx context.Context, id uint64) (*model.Voucher, error) {
	var voucher model.Voucher
	err := r.db.WithContext(ctx).First(&voucher, id).Error
	if err != nil {
		return nil, err
	}
	return &voucher, nil
}

func (r *voucherRepository) QuerySeckillVoucherById(ctx context.Context, voucherId uint64) (*model.SeckillVoucher, error) {
	var seckillVoucher model.SeckillVoucher
	// 对应 tb_seckill_voucher 表
	err := r.db.WithContext(ctx).
		Where("voucher_id = ?", voucherId).
		First(&seckillVoucher).Error
	if err != nil {
		return nil, err
	}
	return &seckillVoucher, nil
}

// DeductStock 扣减库存
// 对应 SQL: UPDATE tb_seckill_voucher SET stock = stock - 1 WHERE voucher_id = ? AND stock > 0
func (r *voucherRepository) DeductStock(ctx context.Context, voucherId uint64) error {
	result := r.db.WithContext(ctx).
		Model(&model.SeckillVoucher{}).
		Where("voucher_id = ? AND stock > 0", voucherId).
		Update("stock", gorm.Expr("stock - 1"))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient stock or voucher not found")
	}
	return nil
}

func (r *voucherRepository) CreateVoucherOrder(ctx context.Context, order *model.VoucherOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *voucherRepository) CountByUserIdAndVoucherId(ctx context.Context, userId uint64, voucherId uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.VoucherOrder{}).
		Where("user_id = ? AND voucher_id = ?", userId, voucherId).
		Count(&count).Error
	return count, err
}