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

	//6.数据库查到了 → json.Marshal 序列化后 s.rdb.Set 存入 Redis
	jsonBytes, err := json.Marshal(shopTypes)
	if err != nil {
		return nil, err
	}
	s.rdb.Set(ctx, key, jsonBytes, utils.RandomizeTTL(constant.CacheShopTypeListTTL, 5*time.Minute))

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
	exists, err := s.rdb.Do(ctx, "BF.EXISTS", constant.BloomFilterShopIdsKey, idStr).Bool()
	if err != nil {
		//Redis bloom 出错时，降级放行（不拦截）,避免全局故障
		log.Printf("bloom filter check error: %v", err)
	} else if !exists {
		//布隆过滤器未命中，直接返回 nil，不查库
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

	//4.构建锁的key
	lockKey := constant.LockShopKey + strconv.FormatUint(id, 10)

	//5.尝试获取互斥锁
	const maxRetries = 20
	locked := false
	for i := 0; i < maxRetries; i++ {
		locked, err = s.tryLock(ctx, lockKey)
		if err != nil {
			return nil, err
		}
		if locked {
			break
		}
		// 未获取到锁，休眠后重试
		time.Sleep(50 * time.Millisecond)

		//再次查询缓存（可能在等锁期间已被其他请求写入）
		val, err = s.rdb.Get(ctx, key).Result()
		if err == nil {
			//判断是否是空值缓存
			if val == "" {
				return nil, nil
			}
			var shop model.Shop
			//缓存命中：用 json.Unmarshal 反序列化 → return &shop, nil
			if err := json.Unmarshal([]byte(val), &shop); err != nil {
				return nil, err
			}
			return &shop, nil
		}
	}

	//6.重试耗尽仍未获取锁 → 返回错误
	if !locked {
		return nil, errors.New("failed to acquire lock for shop query")
	}

	//7.获取到锁 → 确保最终释放锁
	defer s.unlock(ctx, lockKey)

	//8.双重检查：再次查询缓存（可能在等锁期间已被其他请求写入）
	val, err = s.rdb.Get(ctx, key).Result()
	if err == nil {
		//判断是否是空值缓存
		if val == "" {
			return nil, nil
		}
		var shop model.Shop
		//缓存命中：用 json.Unmarshal 反序列化 → return &shop, nil
		if err := json.Unmarshal([]byte(val), &shop); err != nil {
			return nil, err
		}
		return &shop, nil
	}

	//9.缓存未命中 → 查数据库：s.repo.QueryShopById(ctx, id)
	shop, err := s.repo.QueryShopById(ctx, id)
	if err != nil {
		return nil, err
	}

	//10.数据库也没找到（shop == nil）,缓存空值（防止缓存穿透）
	if shop == nil {
		s.rdb.Set(ctx, key, "", utils.RandomizeTTL(constant.CacheNilTTL, 5*time.Minute))
		return nil, nil
	}

	//11.数据库找到了：将json序列化后存入redis
	jsonBytes, err := json.Marshal(shop)
	if err != nil {
		return nil, err
	}
	s.rdb.Set(ctx, key, jsonBytes, utils.RandomizeTTL(constant.CacheShopTTL, 5*time.Minute))

	//12.返回数据(锁会在defer中自动释放)
	return shop, nil
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

// tryLock 尝试获取互斥锁（非阻塞）
// 使用 Redis SETNX 实现，key 格式: lock:shop:{id}
func (s *shopService) tryLock(ctx context.Context, key string) (bool, error) {
	result, err := s.rdb.SetNX(ctx, key, "1", time.Duration(constant.LockShopTTL)*time.Second).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

// unlock释放互斥锁
func (s *shopService) unlock(ctx context.Context, key string) {
	s.rdb.Del(ctx, key)
}

func (s *shopService) SaveShopToRedis(ctx context.Context, id uint64, expireSeconds int64) error {
	// 1. 查询数据库
	shop, err := s.repo.QueryShopById(ctx, id)
	if err != nil {
		return err
	}
	if shop == nil {
		return errors.New("shop not found")
	}

	// 2. 将 Shop 序列化为 JSON（作为 RedisData.Data）
	shopJSON, err := json.Marshal(shop)
	if err != nil {
		return err
	}

	// 3. 构建 RedisData，设置逻辑过期时间 = 当前时间 + expireSeconds
	redisData := model.RedisData{
		Data:       shopJSON,
		ExpireTime: time.Now().Add(time.Duration(expireSeconds) * time.Second),
	}

	// 4. 序列化整个 RedisData
	dataJSON, err := json.Marshal(redisData)
	if err != nil {
		return err
	}

	// 5. 写入 Redis，**不设置 TTL**（永不过期）
	key := constant.CacheHotShopKey + strconv.FormatUint(id, 10)
	s.rdb.Set(ctx, key, dataJSON, 0) // TTL=0 表示永不过期

	return nil
}

func (s *shopService) QueryHotShopById(ctx context.Context, id uint64) (*model.Shop, error) {
	//1.拼接key
	key := constant.CacheHotShopKey + strconv.FormatUint(id, 10)

	// 2. 从 Redis 查询
	val, err := s.rdb.Get(ctx, key).Result()

	// 3. 缓存未命中 → 直接返回 nil（热点 key 理应存在，不存在说明未预热或数据不存在
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// 4. 命中 → 反序列化外层 RedisData
	var redisData model.RedisData
	if err := json.Unmarshal([]byte(val), &redisData); err != nil {
		return nil, err
	}

	// 5. 反序列化内层 Shop 数据
	var shop model.Shop
	if err := json.Unmarshal(redisData.Data, &shop); err != nil {
		return nil, err
	}

	// 6. 判断是否逻辑过期
	if redisData.ExpireTime.After(time.Now()) {
		// 未过期 → 直接返回
		return &shop, nil
	}
	// 7. 已逻辑过期 → 尝试获取互斥锁
	lockKey := constant.LockShopKey + strconv.FormatUint(id, 10)
	locked, err := s.tryLock(ctx, lockKey)
	if err != nil {
		// 获取锁出错，降级返回旧数据
		return &shop, nil
	}

	// 8. 未获取到锁 → 说明有其他 goroutine 正在重建，直接返回旧数据
	if !locked {
		return &shop, nil
	}

	// 9. 获取到锁 → 开启独立 goroutine 异步重建缓存
	go func() {
		defer s.unlock(context.Background(), lockKey)

		// 9a. 查询数据库（用 Background context，因为原请求可能已结束）
		newShop, err := s.repo.QueryShopById(context.Background(), id)
		if err != nil {
			log.Printf("hot shop cache rebuild error: %v", err)
			return
		}
		if newShop == nil {
			return
		}

		// 9b. 序列化 Shop
		shopJSON, err := json.Marshal(newShop)
		if err != nil {
			log.Printf("hot shop cache marshal error: %v", err)
			return
		}

		// 9c. 构建新的 RedisData（重新设置逻辑过期时间）
		newRedisData := model.RedisData{
			Data:       shopJSON,
			ExpireTime: time.Now().Add(30 * time.Minute),
		}

		// 9d. 写入 Redis（永不过期）
		dataJSON, err := json.Marshal(newRedisData)
		if err != nil {
			log.Printf("hot shop cache marshal error:%v", err)
			return
		}
		s.rdb.Set(context.Background(), key, dataJSON, 0)
	}()

	// 10. 当前请求立即返回旧数据（不等待 goroutine 完成）
	return &shop, nil
}
