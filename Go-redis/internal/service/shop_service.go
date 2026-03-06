package service

import (
	"context"
	"encoding/json"
	"errors"
	"go-redis/internal/constant"
	"go-redis/internal/model"
	"go-redis/internal/repository"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// ShopService 商铺相关业务逻辑接口
type ShopService interface {
	// QueryShopTypeList 查询所有商铺类型列表
	QueryShopTypeList(ctx context.Context) ([]model.ShopType, error)
	// QueryShopsByType 根据商铺类型分页查询商铺列表
	QueryShopsByType(ctx context.Context, typeId uint64, current int) ([]model.Shop, error)
	// QueryShopById 根据ID查询商铺（带缓存）
	QueryShopById(ctx context.Context, id uint64) (*model.Shop, error)
}

type shopService struct {
	repo repository.ShopRepository
	rdb  *redis.Client
}

// NewShopService 构造函数：注入 ShopRepo
func NewShopService(repo repository.ShopRepository, rdb *redis.Client) ShopService {
	return &shopService{repo: repo, rdb: rdb}
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

func (s *shopService) QueryShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	// 1. 拼接 key：constant.CacheShopKey + strconv.FormatUint(id, 10)
	key := constant.CacheShopKey + strconv.FormatUint(id, 10)

	// 2. 从 Redis 查询：s.rdb.Get(ctx, key)
	Val, err := s.rdb.Get(ctx, key).Result()

	// 3. 判断是否命中（err == nil 表示命中）
	if err == nil {
		//缓存命中：用 json.Unmarshal 反序列化 → return &shop, nil
		var shop model.Shop
		if err := json.Unmarshal([]byte(Val), &shop); err != nil {
			return nil, err
		}
		return &shop, nil
	}

	if !errors.Is(err, redis.Nil) {
		// Redis 出错（非"key不存在"），向上传递
		return nil, err
	}

	//缓存未命中（err == redis.Nil）：查数据库
	shop, err := s.repo.QueryShopById(ctx, id)
	if err != nil {
		return nil, err
	}

	//数据库也没找到（shop == nil）：return nil, nil（Handler 层处理404）
	if shop == nil {
		return nil, nil
	}

	//数据库找到了：将json序列化后存入redis
	jsonBytes, err := json.Marshal(shop)
	if err != nil {
		return nil, err
	}
	s.rdb.Set(ctx, key, jsonBytes, constant.CacheShopTTL*time.Minute)

	return shop, nil
}
