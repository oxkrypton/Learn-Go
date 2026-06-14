package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go-redis/internal/config"
	"go-redis/internal/dto"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type SeckillOrderQueue interface {
	Publish(ctx context.Context, msg dto.SeckillOrderMessage) error
	Consume(ctx context.Context, handler func(context.Context, dto.SeckillOrderMessage) error) error
	Close() error
}

type NATSSeckillOrderQueue struct {
	nc       *nats.Conn
	js       jetstream.JetStream
	consumer jetstream.Consumer
	stream   string
	subject  string
}

func NewNATSSeckillOrderQueue(ctx context.Context, cfg config.NATSConfig) (*NATSSeckillOrderQueue, error) {
	// 连接 NATS 时保留重连能力
	nc, err := nats.Connect(
		cfg.URL,
		nats.Name("go-redis-seckill-order-queue"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect failed: %w", err)
	}

	// 创建新版 JetStream 客户端
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("jetstream new failed: %w", err)
	}

	duplicateWindow := time.Duration(cfg.DuplicateWindowSeconds) * time.Second
	// 自动创建 stream
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       cfg.Stream,
		// 绑定秒杀订单 subject
		Subjects:   []string{cfg.Subject},
		Retention:  jetstream.WorkQueuePolicy,
		// 保证 NATS 重启后消息仍可恢复
		Storage:    jetstream.FileStorage,
		// 用于发布去重窗口
		Duplicates: duplicateWindow,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create or update stream failed: %w", err)
	}

	ackWait := time.Duration(cfg.AckWaitSeconds) * time.Second
	// 自动创建 durable consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       cfg.Consumer,
		// 只有明确确认后，消息才算处理完成
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       ackWait,
		// 达到最大投递次数后不再继续投递
		MaxDeliver:    cfg.MaxDeliver,
		FilterSubject: cfg.Subject,
		MaxAckPending: 100,
		// 业务处理失败后，不确认消息，让 JetStream 按这个节奏重新投递
		BackOff: []time.Duration{
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
			30 * time.Second,
		},
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create or update consumer failed: %w", err)
	}

	return &NATSSeckillOrderQueue{
		nc:       nc,
		js:       js,
		consumer: consumer,
		stream:   cfg.Stream,
		subject:  cfg.Subject,
	}, nil
}

// 发布订单消息
func (q *NATSSeckillOrderQueue) Publish(ctx context.Context, msg dto.SeckillOrderMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	msgID := fmt.Sprintf("seckill-order:%d", msg.OrderID)
	_, err = q.js.Publish(
		ctx,
		q.subject,
		body,
		jetstream.WithMsgID(msgID),
		jetstream.WithExpectStream(q.stream),
	)
	if err != nil {
		return fmt.Errorf("publish seckill order message failed: %w", err)
	}

	return nil
}

func (q *NATSSeckillOrderQueue) Consume(ctx context.Context, handler func(context.Context, dto.SeckillOrderMessage) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := q.consumer.Fetch(1, jetstream.FetchMaxWait(2*time.Second))
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("fetch seckill order message failed: %v", err)
			continue
		}

		for msg := range msgs.Messages() {
			q.handleMsg(msg, handler)
		}
	}
}

func (q *NATSSeckillOrderQueue) handleMsg(msg jetstream.Msg, handler func(context.Context, dto.SeckillOrderMessage) error) {
	var orderMsg dto.SeckillOrderMessage
	if err := json.Unmarshal(msg.Data(), &orderMsg); err != nil {
		// JSON 解析失败
		log.Printf("unmarshal seckill order message failed: %v", err)
		if err := msg.Term(); err != nil {
			log.Printf("term invalid seckill order message failed: %v", err)
		}
		return
	}

	if orderMsg.OrderID <= 0 || orderMsg.UserID == 0 || orderMsg.VoucherID == 0 {
		log.Printf("invalid seckill order message, orderId=%d userId=%d voucherId=%d", orderMsg.OrderID, orderMsg.UserID, orderMsg.VoucherID)
		if err := msg.Term(); err != nil {
			log.Printf("term invalid seckill order message failed: %v", err)
		}
		return
	}

	// 业务处理失败：打印日志，不确认消息，也不 Nak()，让 JetStream 根据 BackOff 重投。
	if err := handler(context.Background(), orderMsg); err != nil {
		log.Printf("handle nats seckill order message failed: %v", err)
		return
	}

	ackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 业务处理成功
	if err := msg.DoubleAck(ackCtx); err != nil {
		log.Printf("ack seckill order message failed: %v", err)
	}
}

func (q *NATSSeckillOrderQueue) Close() error {
	if q.nc == nil {
		return nil
	}
	return q.nc.Drain()
}
