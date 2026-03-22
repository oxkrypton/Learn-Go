package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-redis/internal/constant"
	"go-redis/internal/model"
	"go-redis/internal/repository"
	"go-redis/internal/utils"
	"log"
	"strconv"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
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
	// CreateShop 创建店铺
	CreateShop(ctx context.Context, shop *model.Shop) error
	// UpdateShop 更新商铺信息，并删除对应的Redis缓存（Cache Aside模式）
	UpdateShop(ctx context.Context, shop *model.Shop) error
	// SaveShopToRedis 预热热点商铺数据到 Redis（逻辑过期方案）
	SaveShopToRedis(ctx context.Context, id uint64, expireSeconds int64) error
	// QueryHotShopById 查询热点商铺（逻辑过期方案，防止缓存击穿）
	QueryHotShopById(ctx context.Context, id uint64) (*model.Shop, error)
}

type shopService struct {
	repo repository.ShopRepository
	rdb  *redis.Client
	//布隆过滤器
	bf *bloom.BloomFilter
}

// NewShopService 构造函数：注入 ShopRepo
func NewShopService(repo repository.ShopRepository, rdb *redis.Client) (ShopService, error) {
	//用redis bloom filter初始化
	//1.创建布隆过滤器(如果不存在)
	_, err := rdb.Do(context.Background(), "BF.RESERVE", constant.BloomFilterShopIdsKey, "0.01", "100000").Result()
	if err != nil {
		// "ERR item exists" 说明已经创建过，不是错误
		if err.Error() != "ERR item exists" {
			return nil, fmt.Errorf("failed to reserve bloom filter: %w", err)
		}
	}

	//2.从数据库加载所有已有的Shop ID到redis布隆过滤器
	ids, err := repo.QueryAllShopIds(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load shop ids for bloom filter: %w", err)
	}
	for _, id := range ids {
		idStr := strconv.FormatUint(id, 10)
		rdb.Do(context.Background(), "BF.ADD", constant.BloomFilterShopIdsKey, idStr)
	}

	return &shopService{repo: repo, rdb: rdb}, nil
}

func (s *shopService) QueryShopTypeList(ctx context.Context) ([]model.ShopType, error) {
	//1.拼接key:constant.CacheShopListKey
	key := constant.CacheShopTypeListKey

	//2.从 Redis 查询 → s.rdb.Get(ctx, key)
	val, err := s.rdb.Get(ctx, key).Result()

	//3.缓存命中 → 反序列化。注意这里是 []model.ShopType（切片），不是单个对象，所以：
	//var shopTypes []model.ShopType
	//json.Unmarshal([]byte(val), &shopTypes)
	if err == nil {
		var shopTypes []model.ShopType
		if err := json.Unmarshal([]byte(val), &shopTypes); err != nil {
			return nil, err
		}
		return shopTypes, nil
	}

	//4.Redis 出错（非 redis.Nil）→ 向上传递错误
	if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	//5.缓存未命中 → 查数据库：s.repo.QueryShopTypes(ctx)
	shopTypes, err := s.repo.QueryShopTypes(ctx)
	if err != nil {
		return nil, err
	}

	//6.写入缓存
	utils.Set(s.rdb, ctx, key, shopTypes, utils.RandomizeTTL(constant.CacheShopTypeListTTL, 5*time.Minute))

	//7.返回结果
	return shopTypes, nil

}

// QueryShopsByType 根据商铺类型分页查询（每页5条）
func (s *shopService) QueryShopsByType(ctx context.Context, typeId uint64, current int) ([]model.Shop, error) {
	size := 5
	return s.repo.QueryShopsByType(ctx, typeId, current, size)
}

// QueryShopById 根据ID查询商铺（带缓存）
func (s *shopService) QueryShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	//先过布隆过滤器
	idStr := strconv.FormatUint(id, 10)
	exists, err := s.rdb.Do(ctx, "BF.EXISTS", constant.BloomFilterShopIdsKey, idStr).Bool()
	if err != nil {
		//Redis bloom 出错时，降级放行（不拦截）,避免全局故障
		log.Printf("bloom filter check error: %v", err)
	} else if !exists {
		//布隆过滤器未命中，直接返回 nil，不查库
		return nil, nil
	}

	// 方法3：缓存空值防穿透
	return utils.QueryWithPassThrough(
		s.rdb, ctx,
		constant.CacheShopKey,
		id,
		utils.RandomizeTTL(constant.CacheShopTTL, 5*time.Minute),
		func(ctx context.Context, id uint64) (*model.Shop, error) {
			return s.repo.QueryShopById(ctx, id)
		},
	)
}

// QueryHotShopById 查询热点商铺（逻辑过期方案，防止缓存击穿）
func (s *shopService) QueryHotShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	//先过布隆过滤器
	idStr := strconv.FormatUint(id, 10)
	exists, err := s.rdb.Do(ctx, "BF.EXISTS", constant.BloomFilterShopIdsKey, idStr).Bool()
	if err != nil {
		//Redis bloom 出错时，降级放行（不拦截）,避免全局故障
		log.Printf("bloom filter check error: %v", err)
	} else if !exists {
		//布隆过滤器未命中，直接返回 nil，不查库
		return nil, nil
	}

	return utils.QueryWithLogicalExpire(
		s.rdb,
		ctx,
		constant.CacheHotShopKey,
		id,
		30*time.Minute,
		constant.LockShopKey,
		func(ctx context.Context, id uint64) (*model.Shop, error) {
			return s.repo.QueryShopById(ctx, id)
		},
	)
}

// CreateShop 创建店铺
func (s *shopService) CreateShop(ctx context.Context, shop *model.Shop) error {
	//1.先写入数据库
	if err := s.repo.CreateShop(ctx, shop); err != nil {
		return err
	}
	//2.写库成功后，将新id加入redis bloom filter
	idStr := strconv.FormatUint(shop.ID, 10)
	s.rdb.Do(ctx, "BF.ADD", constant.BloomFilterShopIdsKey, idStr)
	return nil
}

// UpdateShop 更新商铺信息并删除缓存
// 采用 Cache Aside 模式：先更新数据库，再删除缓存
// 这样下次查询时会从数据库读取最新数据并重新写入缓存
func (s *shopService) UpdateShop(ctx context.Context, shop *model.Shop) error {
	// 1. 先更新数据库
	if err := s.repo.UpdateShop(ctx, shop); err != nil {
		return err
	}

	// 2. 再删除缓存，确保下次读取时能拿到最新数据
	key := constant.CacheShopKey + strconv.FormatUint(shop.ID, 10)
	s.rdb.Del(ctx, key)

	return nil
}

func (s *shopService) SaveShopToRedis(ctx context.Context, id uint64, expireSeconds int64) error {
	// 查询数据库
	shop, err := s.repo.QueryShopById(ctx, id)
	if err != nil {
		return err
	}
	if shop == nil {
		return errors.New("shop not found")
	}

	// 写入 Redis，**不设置 TTL**（永不过期）
	key := constant.CacheHotShopKey + strconv.FormatUint(id, 10)
	utils.SetWithLogicalExpire(s.rdb, ctx, key, shop, time.Duration(expireSeconds)*time.Second)

	return nil
}
