package cache

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// BloomClient 布隆过滤器客户端接口
// 抽象出接口，使业务层不依赖具体实现，便于测试和替换
type BloomClient interface {
	Reserve(ctx context.Context, key string, errorRate float64, capacity int64) error
	Add(ctx context.Context, key string, item string) error
	AddBatch(ctx context.Context, key string, items []string) error
	Exists(ctx context.Context, key string, item string) (bool, error)
}

// RedisBloomClient 基于 Redis RedisBloom 模块的布隆过滤器实现
type RedisBloomClient struct {
	rdb *redis.Client
}

// NewRedisBloomClient 构造函数
func NewRedisBloomClient(rdb *redis.Client) *RedisBloomClient {
	return &RedisBloomClient{rdb: rdb}
}

func (bc *RedisBloomClient) Reserve(ctx context.Context, key string, errorRate float64, capacity int64) error {
	_, err := bc.rdb.Do(ctx, "BF.RESERVE", key, fmt.Sprintf("%f", errorRate), capacity).Result()
	if err != nil {
		if err.Error() == "ERR item exists" {
			return nil
		}
		return fmt.Errorf("bloom filter reserve failed:%w", err)
	}
	return nil
}

func (bc *RedisBloomClient) Add(ctx context.Context, key string, item string) error {
	_, err := bc.rdb.Do(ctx, "BF.ADD", key, item).Result()
	if err != nil {
		return fmt.Errorf("bloom filter add failed: %w", err)
	}
	return nil
}

// AddBatch 使用 Redis Pipeline 批量添加，避免 N 次网络往返
func (bc *RedisBloomClient) AddBatch(ctx context.Context, key string, items []string) error {
	if len(items) == 0 {
		return nil
	}

	pipe := bc.rdb.Pipeline()
	for _, item := range items {
		pipe.Do(ctx, "BF.ADD", key, item)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("bloom filter add batch failed: %w", err)
	}
	return nil
}

func (bc *RedisBloomClient) Exists(ctx context.Context, key string, item string) (bool, error) {
	exists, err := bc.rdb.Do(ctx, "BF.EXISTS", key, item).Bool()
	if err != nil {
		return false, fmt.Errorf("bloom filter exists check failed: %w", err)
	}
	return exists, nil
}

// BloomCheck 业务层便捷方法：检查 ID 是否在布隆过滤器中
// 出错时降级放行（返回 true），避免全局故障
func BloomCheck(bc BloomClient, ctx context.Context, key string, id uint64) bool {
	idStr := strconv.FormatUint(id, 10)
	exists, err := bc.Exists(ctx, key, idStr)
	if err != nil {
		log.Printf("bloom filter check error: %v", err)
		return true
	}
	return exists
}
