package database

import (
	"context"
	"fmt"

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
