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

//go:embed lua/compensate_seckill.lua
var compensateSeckillLua string
var compensateSeckillScript = redis.NewScript(compensateSeckillLua)

func RunSeckillLua(ctx context.Context, rdb *redis.Client, voucherId, userId uint64) (int64, error) {
	stockKey := constant.SeckillStockKey + strconv.FormatUint(voucherId, 10)
	orderKey := constant.SeckillOrderKey + strconv.FormatUint(voucherId, 10)

	return seckillScript.Run(ctx, rdb, []string{stockKey, orderKey}, userId).Int64()
}

func RunCompensateSeckillLua(ctx context.Context, rdb *redis.Client, voucherId, userId uint64) error {
	stockKey := constant.SeckillStockKey + strconv.FormatUint(voucherId, 10)
	orderKey := constant.SeckillOrderKey + strconv.FormatUint(voucherId, 10)

	return compensateSeckillScript.Run(ctx, rdb, []string{stockKey, orderKey}, userId).Err()
}
