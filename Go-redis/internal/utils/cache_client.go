package utils

import (
	"context"
	"encoding/json"
	"errors"
	"go-redis/internal/constant"
	"go-redis/internal/model"
	"strconv"
	"time"

	"log"

	"github.com/redis/go-redis/v9"
)

// 普通缓存写入
// Set 将任意对象序列化为 JSON 并存储到 string 类型的 key 中，设置 TTL 过期时间
func Set(rdb *redis.Client, ctx context.Context, key string, value any, ttl time.Duration) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return rdb.Set(ctx, key, jsonBytes, ttl).Err()
}

// 逻辑过期缓存写入
// SetWithLogicalExpire 将对象序列化为 JSON，包装在 RedisData 中并设置逻辑过期时间，写入 Redis（TTL=0 永不过期）
func SetWithLogicalExpire(rdb *redis.Client, ctx context.Context, key string, value any, logicalTTL time.Duration) error {
	// 1. 将 Shop 序列化为 JSON（作为 RedisData.Data）
	JSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// 2. 构建 RedisData，设置逻辑过期时间 = 当前时间 + expireSeconds
	redisData := model.RedisData{
		Data:       JSON,
		ExpireTime: time.Now().Add(logicalTTL),
	}

	// 3. 序列化整个 RedisData
	dataJSON, err := json.Marshal(redisData)
	if err != nil {
		return err
	}

	// 4. 写入 Redis，**不设置 TTL**（永不过期）
	return rdb.Set(ctx, key, dataJSON, 0).Err() // TTL=0 表示永不过期

}

// 缓存穿透防护查询
// QueryWithPassThrough 根据 key 查询缓存，缓存未命中时调用 dbFallback 回源，
// 利用缓存空值方式解决缓存穿透问题
// 参数说明：
//	keyPrefix: key 前缀，如 "cache:shop:"
//	id:        业务 ID
//	ttl:       正常数据的缓存过期时间
//	dbFallback: 缓存未命中时的数据库查询函数
func QueryWithPassThrough[T any](
	rdb *redis.Client,
	ctx context.Context,
	keyPrefix string,
	id uint64,
	ttl time.Duration,
	dbFallback func(ctx context.Context, id uint64) (*T, error),
) (*T, error) {
	//1.拼接 key
	key := keyPrefix + strconv.FormatUint(id, 10)

	//2.rdb.Get → 命中时判断空值 ("") → 非空则 json.Unmarshal 返回
	val, err := rdb.Get(ctx, key).Result()
	if err == nil {
		// 命中空值 → 返回 nil（穿透防护）
		if val == "" {
			return nil, nil
		}
		//命中有效数据 → 反序列化返回
		var result T
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	//3.未命中 → 调用 dbFallback(ctx, id)
	data, err := dbFallback(ctx, id)
	if err != nil {
		return nil, err
	}

	//4.数据库返回 nil → 缓存空值 "" + 短 TTL
	if data == nil {
		rdb.Set(ctx, key, "", RandomizeTTL(constant.CacheNilTTL, 3*time.Minute))
		return nil, nil
	}

	//5.数据库返回数据 → json.Marshal + rdb.Set + 正常 TTL
	Set(rdb,ctx,key,data,RandomizeTTL(ttl, 5*time.Minute))

	//6.返回结果
	return data, nil
}

// 逻辑过期防击穿查询
// QueryWithLogicalExpire 根据 key 查询缓存，利用逻辑过期解决缓存击穿
func QueryWithLogicalExpire[T any](
	rdb *redis.Client,
	ctx context.Context,
	keyPrefix string,
	id uint64,
	logicalTTL time.Duration,
	lockKeyPrefix string,
	dbFallback func(ctx context.Context, id uint64) (*T, error),
) (*T, error) {
	// 1. 拼接 key
	key := keyPrefix + strconv.FormatUint(id, 10)
	// 2. 查缓存
	val, err := rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil // 未预热，直接返回
	}
	if err != nil {
		return nil, err
	}
	// 3. 反序列化外层 RedisData
	var redisData model.RedisData
	if err := json.Unmarshal([]byte(val), &redisData); err != nil {
		return nil, err
	}
	// 4. 反序列化内层业务数据
	var result T
	if err := json.Unmarshal(redisData.Data, &result); err != nil {
		return nil, err
	}
	// 5. 未过期 → 直接返回
	if redisData.ExpireTime.After(time.Now()) {
		return &result, nil
	}
	// 6. 已逻辑过期 → 尝试获取锁
	lockKey := lockKeyPrefix + strconv.FormatUint(id, 10)
	locked, err := TryLock(rdb, ctx, lockKey)
	if err != nil || !locked {
		// 获取锁失败或出错 → 降级返回旧数据
		return &result, nil
	}
	// 7. 获取锁成功 → 异步重建缓存
	go func() {
		defer Unlock(rdb, context.Background(), lockKey)
		newData, err := dbFallback(context.Background(), id)
		if err != nil {
			log.Printf("cache rebuild error: %v", err)
			return
		}
		if newData == nil {
			return
		}
		// 用 SetWithLogicalExpire 写入（复用方法2）
		if err := SetWithLogicalExpire(rdb, context.Background(), key, newData, logicalTTL); err != nil {
			log.Printf("cache rebuild marshal error: %v", err)
		}
	}()
	// 8. 立即返回旧数据
	return &result, nil
}

// tryLock 尝试获取互斥锁（非阻塞）
// 使用 Redis SETNX 实现，key 格式: lock:shop:{id}
func TryLock(rdb *redis.Client, ctx context.Context, key string) (bool, error) {
	return rdb.SetNX(ctx, key, "1", time.Duration(constant.LockShopTTL)*time.Second).Result()
}

// unlock释放互斥锁
func Unlock(rdb *redis.Client, ctx context.Context, key string) {
	rdb.Del(ctx, key)
}