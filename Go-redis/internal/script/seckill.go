package script

import (
	"context"
	_ "embed"
	"go-redis/internal/constant"
	"strconv"

	"github.com/redis/go-redis/v9"
)

//go:embed lua/seckill.lua
var seckillLua string
var seckillScript = redis.NewScript(seckillLua)

func RunSeckillLua(ctx context.Context, rdb *redis.Client, voucherId, userId uint64) (int64, error) {
	stockKey := constant.SeckillStockKey + strconv.FormatUint(voucherId, 10)
	orderKey := constant.SeckillOrderKey + strconv.FormatUint(voucherId, 10)

	// 在 Redis 里原子完成库存判断和一人一单判断
	return seckillScript.Run(ctx, rdb, []string{stockKey, orderKey}, userId).Int64()
}
