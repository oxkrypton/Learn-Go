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
