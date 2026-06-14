package cache

import (
	"context"
	_ "embed"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

//go:embed lua/unlock.lua
var unlockLua string
var unlockScript = redis.NewScript(unlockLua)

// 可重入锁脚本
//
//go:embed lua/reentrant_lock.lua
var reentrantLockLua string
var reentrantLockScript = redis.NewScript(reentrantLockLua)

//go:embed lua/reentrant_unlock.lua
var reentrantUnlockLua string
var reentrantUnlockScript = redis.NewScript(reentrantUnlockLua)

//go:embed lua/reentrant_renew.lua
var reentrantRenewLua string
var reentrantRenewScript = redis.NewScript(reentrantRenewLua)

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

	mu             sync.Mutex
	watchdogCancel context.CancelFunc
}

func NewReentrantLockFromContext(
	ctx context.Context,
	rdb *redis.Client,
	key string,
	leaseTTL time.Duration,
) (*ReentrantLock, context.Context, error) {
	ctx = WithLockOwner(ctx)
	owner, ok := LockOwnerFromContext(ctx)
	if !ok || owner == "" {
		return nil, ctx, errors.New("lock owner is empty")
	}

	lock, err := NewReentrantLock(rdb, key, owner, leaseTTL)
	if err != nil {
		return nil, ctx, err
	}

	return lock, ctx, nil
}

func NewReentrantLock(rdb *redis.Client, key string, owner string, leaseTTL time.Duration) (*ReentrantLock, error) {
	if leaseTTL <= 0 {
		leaseTTL = 5 * time.Second
	}

	//context的owner不能为空
	if owner == "" {
		return nil, errors.New("lock owner is empty")
	}

	return &ReentrantLock{
		rdb:      rdb,
		key:      key,
		owner:    owner,
		leaseTTL: leaseTTL,
	}, nil
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

	if res != 1 {
		return false, nil
	}

	// 成功拿锁或成功重入后，确保看门狗在运行。
	l.startWatchdog()
	return true, nil
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

	// res == 0 说明已经完全释放，锁 key 已删除。
	if res == 0 {
		l.stopWatchdog()
	}

	return nil
}

type lockOwnerContextKey struct{}

// 在上下文中添加owner
func WithLockOwner(ctx context.Context) context.Context {
	//确保 ctx 不为 nil
	if ctx == nil {
		ctx = context.Background()
	}

	//幂等性检查：如果当前 context 已经有了标识符，直接返回原 context
	if _, ok := LockOwnerFromContext(ctx); ok {
		return ctx
	}

	//在ctx中使用uuid生成owner标识
	return context.WithValue(ctx, lockOwnerContextKey{}, "req-lock-owner:"+uuid.NewString())
}

// 在上下文中获取owner
func LockOwnerFromContext(ctx context.Context) (string, bool) {
	//从 context 中取出值，并进行类型断言（Type Assertion）
	owner, ok := ctx.Value(lockOwnerContextKey{}).(string)

	//// 验证值是否存在且不为空
	if !ok || owner == "" {
		return "", false
	}

	return owner, true
}

func (l *ReentrantLock) startWatchdog() {
	// 1. 加本地锁，保护结构体字段并发访问安全
	l.mu.Lock()
	defer l.mu.Unlock()

	// 2. 幂等检查：如果看门狗已在运行（cancel 函数不为空），则直接返回
	if l.watchdogCancel != nil {
		return
	}

	// 3. 创建可取消的上下文，并将 cancel 函数保存到结构体，供解锁时停止协程
	watchdogCtx, cancel := context.WithCancel(context.Background())
	l.watchdogCancel = cancel

	// 4. 设置续期间隔：通常为租约时间的 1/3，留出足够的网络容错时间
	interval := l.leaseTTL / 3
	if interval <= 0 {
		interval = time.Second // 兜底逻辑：最小间隔 1 秒
	}

	// 5. 启动后台协程，执行具体的 Redis 续期循环
	go l.watchdogLoop(watchdogCtx, interval)
}

func (l *ReentrantLock) watchdogLoop(ctx context.Context, interval time.Duration) {
	// 创建定时器，按照传入的间隔时间触发续期
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		// 如果收到取消信号（解锁时触发），则停止看门狗并退出协程
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 每次定时器触发时，创建一个带 2 秒超时保护的上下文执行续期操作
			renewCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			renewed, err := l.renew(renewCtx)
			cancel() // 及时取消 renewCtx，释放资源

			// 如果续期过程中发生错误，或者 Redis 返回续期失败（例如锁已被他人抢占），则停止看门狗
			if err != nil || !renewed {
				return
			}
		}
	}
}

func (l *ReentrantLock) renew(ctx context.Context) (bool, error) {
	res, err := reentrantRenewScript.Run(
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

func (l *ReentrantLock) stopWatchdog() {
	// 1. 加本地锁，确保在停止看门狗时，不会有其他协程同时在操作这个锁的状态
	l.mu.Lock()
	defer l.mu.Unlock()

	// 2. 检查看门狗是否已经在运行，如果没在运行则直接返回
	if l.watchdogCancel == nil {
		return
	}

	// 3. 调用 context 的 cancel 函数，向后台 watchdogLoop 协程发送停止信号
	l.watchdogCancel()

	// 4. 将 cancel 函数置为空，既能释放资源，也方便后续重新启动看门狗（幂等性）
	l.watchdogCancel = nil
}
