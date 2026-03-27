package utils

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

const beginTimestamp int64 = 1640995200 // 2022-01-01 00:00:00 UTC
const countBits = 32

// NextID 生成全局唯一ID: 时间戳(31位) + 计数器(32位)
func NewxtID(ctx context.Context, rdb *redis.Client, keyPrefix string) (int64, error) {
	now := time.Now()
	timestamp := now.Unix() - beginTimestamp

	// 按天拼 key，方便统计每天的订单量
	date := now.Format("2006:01:02")
	key := fmt.Sprintf("icr:%s:%s", keyPrefix, date)

	//Redis INCR 自增
	count, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	// 拼接: 时间戳左移32位 | 计数器
	return (timestamp << countBits) | count, nil
}
