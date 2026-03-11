package service

import (
	"context"
	"encoding/json"
	"errors"
	"go-redis/internal/constant"
	"go-redis/internal/model"
	"go-redis/internal/repository"
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
	// UpdateShop 更新商铺信息，并删除对应的Redis缓存（Cache Aside模式）
	UpdateShop(ctx context.Context, shop *model.Shop) error
}

type shopService struct {
	repo repository.ShopRepository
	rdb  *redis.Client
	//布隆过滤器
	bf *bloom.BloomFilter
}

// NewShopService 构造函数：注入 ShopRepo
func NewShopService(repo repository.ShopRepository, rdb *redis.Client) ShopService {
	//1.创建布隆过滤器(预估100，000条数据，误判率1%)
	bf := bloom.NewWithEstimates(100000, 0.01)

	//2.从数据库加载所有已有的Shop ID到布隆过滤器
	ids, err := repo.QueryAllShopIds(context.Background())
	if err == nil {
		for _, id := range ids {
			idStr := strconv.FormatUint(id, 10)
			bf.AddString(idStr)
		}
	} else {
		log.Printf("Failed to init bloom filter: %v", err)

	}
	return &shopService{
		repo: repo,
		rdb:  rdb,
		bf:   bf,
	}
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

	//6.数据库查到了 → json.Marshal 序列化后 s.rdb.Set 存入 Redis
	jsonBytes, err := json.Marshal(shopTypes)
	if err != nil {
		return nil, err
	}
	s.rdb.Set(ctx, key, jsonBytes, constant.CacheShopTypeListTTL*time.Minute)

	//7.返回结果
	return shopTypes, nil

}

// QueryShopsByType 根据商铺类型分页查询（每页5条）
func (s *shopService) QueryShopsByType(ctx context.Context, typeId uint64, current int) ([]model.Shop, error) {
	size := 5
	return s.repo.QueryShopsByType(ctx, typeId, current, size)
}

func (s *shopService) QueryShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	//先过布隆过滤器
	idStr := strconv.FormatUint(id, 10)
	if !s.bf.TestString(idStr) {
		//布隆过滤器不存在，返回nil
		return nil, nil
	}
	// 1. 拼接 key：constant.CacheShopKey + strconv.FormatUint(id, 10)
	key := constant.CacheShopKey + strconv.FormatUint(id, 10)

	// 2. 从 Redis 查询：s.rdb.Get(ctx, key)
	val, err := s.rdb.Get(ctx, key).Result()

	// 3. 判断是否命中（err == nil 表示命中）
	if err == nil {
		//判断是否是空值缓存
		if val == "" {
			return nil, nil
		}
		//缓存命中：用 json.Unmarshal 反序列化 → return &shop, nil
		var shop model.Shop
		if err := json.Unmarshal([]byte(val), &shop); err != nil {
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
		s.rdb.Set(ctx, key, "", constant.CacheNilTTL*time.Minute)
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
