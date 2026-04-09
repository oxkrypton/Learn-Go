package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"go-redis/internal/constant"
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
	JSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	redisData := RedisData{
		Data:       JSON,
		ExpireTime: time.Now().Add(logicalTTL),
	}

	dataJSON, err := json.Marshal(redisData)
	if err != nil {
		return err
	}

	return rdb.Set(ctx, key, dataJSON, 0).Err()
}

// 缓存穿透防护查询
// QueryWithPassThrough 根据 key 查询缓存，缓存未命中时调用 dbFallback 回源，
// 利用缓存空值方式解决缓存穿透问题
func QueryWithPassThrough[T any](
	rdb *redis.Client,
	ctx context.Context,
	keyPrefix string,
	id uint64,
	ttl time.Duration,
	dbFallback func(ctx context.Context, id uint64) (*T, error),
) (*T, error) {
	key := keyPrefix + strconv.FormatUint(id, 10)

	val, err := rdb.Get(ctx, key).Result()
	if err == nil {
		if val == "" {
			return nil, nil
		}

		var result T
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	if !errors.Is(err, redis.Nil) {
		return nil, err
	}

	data, err := dbFallback(ctx, id)
	if err != nil {
		return nil, err
	}

	if data == nil {
		if err := rdb.Set(ctx, key, "", RandomizeTTL(constant.CacheNilTTL, 3*time.Minute)).Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	if err := Set(rdb, ctx, key, data, RandomizeTTL(ttl, 5*time.Minute)); err != nil {
		return nil, err
	}

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
	key := keyPrefix + strconv.FormatUint(id, 10)
	val, err := rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var redisData RedisData
	if err := json.Unmarshal([]byte(val), &redisData); err != nil {
		return nil, err
	}

	var result T
	if err := json.Unmarshal(redisData.Data, &result); err != nil {
		return nil, err
	}

	if redisData.ExpireTime.After(time.Now()) {
		return &result, nil
	}

	lockKey := lockKeyPrefix + strconv.FormatUint(id, 10)
	lockValue := uuid.NewString()
	locked, err := TryLock(rdb, ctx, lockKey, lockValue, time.Duration(constant.LockShopTTL)*time.Second)
	if err != nil || !locked {
		return &result, nil
	}

	go func(lockKey, lockValue, key string, id uint64) {
		defer func() {
			if err := Unlock(rdb, context.Background(), lockKey, lockValue); err != nil {
				log.Printf("cache unlock error: %v", err)
			}
		}()

		newData, err := dbFallback(context.Background(), id)
		if err != nil {
			log.Printf("cache rebuild error: %v", err)
			return
		}
		if newData == nil {
			return
		}
		if err := SetWithLogicalExpire(rdb, context.Background(), key, newData, logicalTTL); err != nil {
			log.Printf("cache rebuild marshal error: %v", err)
		}
	}(lockKey, lockValue, key, id)

	return &result, nil
}

// RandomizeTTL 在 baseTTL 的基础上添加随机抖动，防止缓存雪崩
func RandomizeTTL(baseTTL, maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return baseTTL
	}

	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return baseTTL + jitter
}
