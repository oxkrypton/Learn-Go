package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"go-redis/internal/config"
)

var rdb *redis.Client

func InitRedis(cfg config.RedisConfig) (*redis.Client, error) {
	rdb = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.MaxActive,
		MinIdleConns: cfg.MaxIdle,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis connect failed: %v", err)
	}

	return rdb, nil
}

func InitMutilRedis(cfgs []config.RedisConfig) ([]*redis.Client, error) {
	if len(cfgs) == 0 {
		return nil, fmt.Errorf("redis configs is empty")
	}

	clients := make([]*redis.Client, 0, len(cfgs))

	for i, cfg := range cfgs {
		client := redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.MaxActive,
			MinIdleConns: cfg.MaxIdle,
		})

		// ping健康检查，两秒超时直接失败
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := client.Ping(ctx).Err()
		cancel()

		if err != nil {
			_ = client.Close() // 关闭当前连接失败的客户端

			for _, c := range clients {
				_ = c.Close() // 依次关闭之前已经成功创建并放入切片的客户端
			}

			return nil, fmt.Errorf("redis node %d connect failed (%s:%d): %w", i, cfg.Host, cfg.Port, err)
		}

		clients = append(clients, client)
	}

	return clients, nil
}
