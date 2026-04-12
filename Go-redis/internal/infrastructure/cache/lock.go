package cache

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/unlock.lua
var unlockLua string
var unlockScript = redis.NewScript(unlockLua)

// 可重入锁脚本
//go:embed lua/reentrant_lock.lua
var reentrantLockLua string
var reentrantLockScript = redis.NewScript(reentrantLockLua)

//go:embed lua/reentrant_unlock.lua
var reentrantUnlockLua string
var reentrantUnlockScript = redis.NewScript(reentrantUnlockLua)

var ErrLockNotHeld = errors.New("lock not held by current owner")

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

// 可重入锁结构体
type ReentrantLock struct {
	rdb      *redis.Client
	key      string
	owner    string
	leaseTTL time.Duration
}

func NewReentrantLock(rdb *redis.Client, key string, leaseTTL time.Duration) *ReentrantLock {
	if leaseTTL <= 0 {
		leaseTTL = 5 * time.Second
	}

	return &ReentrantLock{
		rdb:      rdb,
		key:      key,
		owner:    uuid.NewString(),
		leaseTTL: leaseTTL,
	}
}

func (l *ReentrantLock) TryLock(ctx context.Context) (bool, error) {
	res, err := reentrantLockScript.Run(
		ctx,
		l.rdb,
		[]string{l.key},
		l.owner,
		l.leaseTTL.Milliseconds(),
	).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

func (l *ReentrantLock) Unlock(ctx context.Context) error {
	res, err := reentrantUnlockScript.Run(
		ctx,
		l.rdb,
		[]string{l.key},
		l.owner,
		l.leaseTTL.Milliseconds(),
	).Int()
	if err != nil {
		return err
	}

	if res == -1 {
		return ErrLockNotHeld
	}

	return nil
}
