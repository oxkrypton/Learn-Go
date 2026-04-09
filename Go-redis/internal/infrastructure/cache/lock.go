package cache

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/unlock.lua
var unlockLua string

var unlockScript = redis.NewScript(unlockLua)

// TryLock 尝试获取互斥锁（非阻塞）
func TryLock(rdb *redis.Client, ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return rdb.SetNX(ctx, key, value, ttl).Result()
}

// Unlock 释放互斥锁
func Unlock(rdb *redis.Client, ctx context.Context, key string, value string) error {
	_, err := unlockScript.Run(ctx, rdb, []string{key}, value).Result()
	if err == redis.Nil {
		return nil
	}
	return err
}
