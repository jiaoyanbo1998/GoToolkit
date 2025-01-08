package kafkax

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"time"
)

// KafkaAsyncBatchConsumer 异步批量消费者
type KafkaAsyncBatchConsumer[T any] struct {
	reader    *kafka.Reader // kafka读取器
	timeout   time.Duration // 超时时间
	batchSize int           // 每一批的大小
	// 参数1：从kafka消费者中，接收到的消息
	// 参数2：消息中存储的值
	handler func(vals []T) error // 处理函数
}

// BatchConsumerOption 是一个函数类型，用于设置KafkaAsyncBatchConsumer的选项
type BatchConsumerOption func(*kafka.ReaderConfig)

// NewKafkaAsyncBatchConsumer 创建一个kafka异步批量消费者
func NewKafkaAsyncBatchConsumer[T any](
	brokers []string, groupId, topic string,
	timeout time.Duration, batchSize int,
	handler func(vals []T) error, opts ...BatchConsumerOption) *KafkaAsyncBatchConsumer[T] {
	// 默认配置
	config := kafka.ReaderConfig{
		Brokers: brokers,
		GroupID: groupId,
		Topic:   topic,
	}
	// 用户自定义配置
	for _, opt := range opts {
		opt(&config)
	}
	// 创建 KafkaAsyncBatchConsumer
	kac := &KafkaAsyncBatchConsumer[T]{
		reader:    kafka.NewReader(config),
		timeout:   timeout,
		batchSize: batchSize,
		handler:   handler,
	}
	return kac
}

// WithMinBytes 设置每次读取的最小字节数
func (kac *KafkaAsyncBatchConsumer[T]) WithMinBytes(minBytes int) BatchConsumerOption {
	return func(cfg *kafka.ReaderConfig) {
		cfg.MinBytes = minBytes
	}
}

// WithMaxBytes 设置每次读取的最大字节数
func (kac *KafkaAsyncBatchConsumer[T]) WithMaxBytes(maxBytes int) BatchConsumerOption {
	return func(cfg *kafka.ReaderConfig) {
		cfg.MaxBytes = maxBytes
	}
}

// ReadAndProcessMsg 读取并处理消息
func (kac *KafkaAsyncBatchConsumer[T]) ReadAndProcessMsg() {
	// 无限循环
	for {
		// 创建有过期时间的上下文，防止因为一批消息凑不够，就一直等待
		ctx, cancel := context.WithTimeout(context.Background(), kac.timeout)
		defer cancel()

		// 存储被消费过的消息
		messages := make([]kafka.Message, 0, kac.batchSize)
		// 存储消息的值
		vals := make([]T, 0, kac.batchSize)

		// 读取消息，直到凑过一批，或超时
		for i := 0; i < kac.batchSize; i++ {
			// 读取kafka中的消息
			message, err := kac.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.DeadlineExceeded {
					zap.L().Warn("读取消息超时")
					break
				}
				zap.L().Error("kafka读取数据失败", zap.Error(err))
				continue
			}

			// json反序列化
			var val T
			err = json.Unmarshal(message.Value, &val)
			if err != nil {
				zap.L().Error("消息反序列化失败", zap.Error(err))
				// 这个消息不做处理，继续处理下一个消息
				continue
			}

			// 读取到的kafka中的消息
			messages = append(messages, message)
			// 消息中的数据
			vals = append(vals, val)
		}

		// 如果没有读取到任何消息，继续下一次循环
		if len(messages) == 0 {
			continue
		}

		// 调用处理函数处理消息批次
		err := kac.handler(vals)
		if err != nil {
			zap.L().Error("调用批量处理接口失败", zap.Error(err))
			continue
		}

		// 批量提交已经消费的消息
		err = kac.reader.CommitMessages(ctx, messages...)
		if err != nil {
			zap.L().Error("标记消息失败", zap.Error(err))
		} else {
			for _, message := range messages {
				zap.L().Info("消息已消费",
					zap.String("topic", message.Topic),
					zap.Int64("offset", message.Offset))
			}
		}
	}
}

func (kac *KafkaAsyncBatchConsumer[T]) Stop() {
	// 关闭kafka读取器
	kac.reader.Close()
}
