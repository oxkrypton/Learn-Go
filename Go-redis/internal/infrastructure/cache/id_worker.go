package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const beginTimestamp int64 = 1640995200 // 2022-01-01 00:00:00 UTC
const countBits = 32

// NextID 生成全局唯一ID: 时间戳(31位) + 计数器(32位)
func NextID(ctx context.Context, rdb *redis.Client, keyPrefix string) (int64, error) {
	now := time.Now()
	timestamp := now.Unix() - beginTimestamp

	date := now.Format("2006:01:02")
	key := fmt.Sprintf("icr:%s:%s", keyPrefix, date)

	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return (timestamp << countBits) | count, nil
}
