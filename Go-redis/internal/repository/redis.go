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
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	return RDB.Ping(context.Background()).Err()
}
