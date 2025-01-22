package saramax

import (
	"GoToolkit/loggerx"
	"context"
	"encoding/json"
	"github.com/IBM/sarama"
	"time"
)

// BatchHandler 处理"批量消息"的处理器
type BatchHandler[T any] struct {
	// 日志记录器
	logger loggerx.Logger
	// 消息处理函数
	// 	  func Consume(msg []*sarama.ConsumerMessage, evt []ReadEvent) error {}
	//    	 参数1：从kafka消费者中，接收到的消息
	//	     参数2：消息中存储的值
	fn            func(msg []*sarama.ConsumerMessage, ts []T) error
	batchSize     int
	batchDuration time.Duration
}

// Option 是修改 BatchHandler 的配置项类型
type Option[T any] func(*BatchHandler[T])

// NewBatchHandler 使用构造函数，创建处理"批量消息"的处理器
func NewBatchHandler[T any](l loggerx.Logger,
	fn func(msg []*sarama.ConsumerMessage, ts []T) error,
	options ...Option[T]) *BatchHandler[T] {
	// 默认配置
	handler := &BatchHandler[T]{
		logger:        l,
		fn:            fn,
		batchSize:     10,              // 默认批处理大小
		batchDuration: 5 * time.Second, // 默认批处理持续时间
	}

	// 应用所有的选项
	for _, option := range options {
		option(handler)
	}

	return handler
}

// WithBatchSize 设置批处理的大小
func WithBatchSize[T any](size int) Option[T] {
	return func(b *BatchHandler[T]) {
		b.batchSize = size
	}
}

// WithBatchDuration 设置批处理的持续时间
func WithBatchDuration[T any](duration time.Duration) Option[T] {
	return func(b *BatchHandler[T]) {
		b.batchDuration = duration
	}
}

// ConsumeClaim 消费"批量消息"
//    session 消费者组的会话（从和Kafka建立连接到断开连接之间的一段时间）
//    claim  消费者组，消费的分区
func (b *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	// 获取分区中的消息
	messages := claim.Messages()
	// 一批处理几个消息
	batchSize := b.batchSize
	// 循环处理批量消息
	for {
		// 创建有过期时间的上下文，防止因为一批消息凑不够，就一直等待
		ctx, cancel := context.WithTimeout(context.Background(), b.batchDuration)
		// 存储被消费过的消息
		msg := make([]*sarama.ConsumerMessage, 0, batchSize)
		// 存储消息的值
		ts := make([]T, 0, batchSize)
		done := false
		// 处理size个消息
		for i := 0; i < batchSize; i++ {
			select {
			// 上下文到期
			case <-ctx.Done():
				done = true
			// 读取messages管道中传递的消息
			case message, ok := <-messages:
				if !ok {
					// 消费者被关闭
					cancel()
					return nil
				}
				var t T
				// json反序列化
				err := json.Unmarshal(message.Value, &t)
				if err != nil {
					b.logger.Error("消息反序列化失败", loggerx.Error(err))
					// 这个消息不做处理，继续处理下一个消息
					continue
				}
				msg = append(msg, message)
				ts = append(ts, t)
			}
			// 上下文到期结束，避免长时间等待凑够一批
			if done {
				break
			}
		}
		// 取消上下文
		cancel()
		// 没有一条消息，执行下一次循环
		if len(msg) == 0 {
			continue
		}
		// 调用批量处理函数
		err := b.fn(msg, ts)
		if err != nil {
			b.logger.Error("调用批量处理接口失败", loggerx.Error(err))
		}
		// 标记这些消息已被消费
		for _, m := range msg {
			session.MarkMessage(m, "")
		}
	}
}

// Setup 消费组会话开始前执行初始化操作
func (b *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 消费者组会话结束后执行清理工作
func (b *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}
