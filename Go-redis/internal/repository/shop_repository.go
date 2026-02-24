package repository

import (
	"context"
	"errors"
	"go-redis/internal/model"

	"gorm.io/gorm"
)

// 定义店铺及店铺类型的数据访问接口
type ShopRepository interface {
	//根据id查询店铺
	QueryShopById(ctx context.Context, id uint64) (*model.Shop, error)
	//查询所有店铺类型(用于首页分类展示)
	QueryShopTypes(ctx context.Context) ([]model.ShopType, error)
	//根据类型分页查询店铺
	QueryShopsByType(ctx context.Context, typeId uint64, current int, size int) ([]model.Shop, error)
	//更新店铺信息
	UpdateShop(ctx context.Context, shop *model.Shop) error
}

type shopRepository struct {
	db *gorm.DB
}

// NewShopRepository 构造函数
func NewShopRepository(db *gorm.DB) ShopRepository {
	return &shopRepository{
		db: db,
	}
}

// QueryShopById 实现根据ID查询店铺
func (r *shopRepository) QueryShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	var shop model.Shop
	// 对应 tb_shop 表
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&shop).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 没找到不报错，返回 nil
		}
		return nil, err
	}
	return &shop, nil
}

// QueryShopTypes 实现查询所有店铺类型
func (r *shopRepository) QueryShopTypes(ctx context.Context) ([]model.ShopType, error) {
	var types []model.ShopType
	// 对应 tb_shop_type 表，按 sort 字段排序
	if err := r.db.WithContext(ctx).Order("sort ASC").Find(&types).Error; err != nil {
		return nil, err
	}
	return types, nil
}

// QueryShopsByType 实现根据类型分页查询店铺
func (r *shopRepository) QueryShopsByType(ctx context.Context, typeId uint64, current int, size int) ([]model.Shop, error) {
	var shops []model.Shop
	offset := (current - 1) * size
	// 对应 tb_shop 表，根据 type_id 过滤
	err := r.db.WithContext(ctx).
		Where("type_id = ?", typeId).
		Order("id ASC"). // 默认按ID排序，后续可改为按距离或评分
		Limit(size).
		Offset(offset).
		Find(&shops).Error
	if err != nil {
		return nil, err
	}
	return shops, nil
}

// UpdateShop 实现更新店铺信息
func (r *shopRepository) UpdateShop(ctx context.Context, shop *model.Shop) error {
	// 只更新非零值字段
	return r.db.WithContext(ctx).Model(shop).Updates(shop).Error
}
