package repository

import (
	"context"
	"fmt"
	"go-redis/internal/config"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() error {
	cfg := config.GlobalConfig.Redis

	RDB = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.MaxActive,
		MinIdleConns: cfg.MaxIdle,
	})

	if err := RDB.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("redis connect failed: %v", err)
	}

	return nil
}
