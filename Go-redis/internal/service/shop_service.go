package service

import (
	"context"
	"go-redis/internal/model"
	"go-redis/internal/repository"
)

// ShopService 商铺相关业务逻辑接口
type ShopService interface {
	// QueryShopTypeList 查询所有商铺类型列表
	QueryShopTypeList(ctx context.Context) ([]model.ShopType, error)
	// QueryShopsByType 根据商铺类型分页查询商铺列表
	QueryShopsByType(ctx context.Context, typeId uint64, current int) ([]model.Shop, error)
}

type shopService struct {
	repo repository.ShopRepository
}

// NewShopService 构造函数：注入 ShopRepo
func NewShopService(repo repository.ShopRepository) ShopService {
	return &shopService{repo: repo}
}

func (s *shopService) QueryShopTypeList(ctx context.Context) ([]model.ShopType, error) {
	// 核心逻辑：目前直接查数据库，未来可在此处加 Redis 缓存逻辑，提升性能
	return s.repo.QueryShopTypes(ctx)
}

// QueryShopsByType 根据商铺类型分页查询（每页5条）
func (s *shopService) QueryShopsByType(ctx context.Context, typeId uint64, current int) ([]model.Shop, error) {
	size := 5
	return s.repo.QueryShopsByType(ctx, typeId, current, size)
}
