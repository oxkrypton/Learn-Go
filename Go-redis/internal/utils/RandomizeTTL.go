package utils

import (
	"math/rand"
	"time"
)

// RandomizeTTL 在 baseTTL 的基础上添加随机抖动，防止缓存雪崩
// 例如 baseTTL=30min, maxJitter=5min → 实际 TTL 在 [30min, 35min) 之间
func RandomizeTTL(baseTTL, maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return baseTTL
	}
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return baseTTL + jitter
}
