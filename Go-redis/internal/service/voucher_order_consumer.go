package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-redis/internal/constant"
	"go-redis/internal/dto"
	"go-redis/internal/model"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// 阻塞队列读信息
func (s *voucherService) StartOrderConsumer(ctx context.Context) {
	// 阻塞读取队列消息，没有消息时会等待
	for {
		select {
		case <-ctx.Done():
			log.Printf("---start order consumer---")
			return
		default:
		}

		//消费者右弹
		//BRPop返回值 1.对列名，2.弹出的值
		result, err := s.rdb.BRPop(ctx, 2*time.Second, constant.SeckillOrderQueueKey).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			log.Printf("consume seckill order failed:%v", err)
			continue
		}

		if len(result) != 2 {
			continue
		}

		if err := s.handleOrderMessage(ctx, result[1]); err != nil {
			log.Printf("handle seckill order failed: %v", err)
		}
	}
}

// 拆消息、验消息、转交处理
func (s *voucherService) handleOrderMessage(ctx context.Context, raw string) error {
	var msg dto.SeckillOrderMessage

	// 把队列里的 JSON 解析出来
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return fmt.Errorf("unmarshal seckill order message failed, raw=%s: %w", raw, err)
	}

	// 基本的消息合法性校验，避免脏消息直接进数据库
	if msg.OrderID <= 0 || msg.UserID == 0 || msg.VoucherID == 0 {
		return fmt.Errorf(
			"invalid seckill order message, orderId=%d userId=%d voucherId=%d raw=%s",
			msg.OrderID, msg.UserID, msg.VoucherID, raw,
		)
	}

	// 转交异步落库逻辑
	if err := s.createVoucherOrderAsync(ctx, msg); err != nil {
		return fmt.Errorf(
			"process seckill order message failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}

	return nil
}

// 实现落库
func (s *voucherService) createVoucherOrderAsync(ctx context.Context, msg dto.SeckillOrderMessage) error {
	//数据库校验
	count, err := s.repo.CountByUserIdAndVoucherId(ctx, msg.UserID, msg.VoucherID)
	if err != nil {
		return fmt.Errorf(
			"check existing order failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}
	if count > 0 {
		log.Printf(
			"skip duplicated order message, orderId=%d userId=%d voucherId=%d",
			msg.OrderID, msg.UserID, msg.VoucherID,
		)
		return nil
	}

	//数据库扣减库存
	if err := s.repo.DeductStock(ctx, msg.VoucherID); err != nil {
		return fmt.Errorf(
			"deduct stock failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}

	order := &model.VoucherOrder{
		ID:        msg.OrderID,
		UserID:    msg.UserID,
		VoucherID: msg.VoucherID,
	}
	//将创建的订单写入数据库
	if err := s.repo.CreateVoucherOrder(ctx, order); err != nil {
		return fmt.Errorf(
			"create voucher order failed, orderId=%d userId=%d voucherId=%d: %w",
			msg.OrderID, msg.UserID, msg.VoucherID, err,
		)
	}

	return nil
}
