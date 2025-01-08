package kafkax

import (
	"context"
	"errors"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"time"
)

// KafkaProducer kafka异步生产者
type KafkaProducer struct {
	writer    *kafka.Writer // kafka写入器
	closeChan chan struct{} // 发送关闭信号的管道
}

// ProducerOption 是一个函数类型，用户自定义配置
type ProducerOption func(*kafka.Writer)

// WithTimeout 设置超时时间，控制writer.WriteMessages()方法的超时时间
func WithTimeout(timeout time.Duration) ProducerOption {
	return func(writer *kafka.Writer) {
		writer.WriteTimeout = timeout
	}
}

// WithAsync 是否开启异步发送，控制writer.WriteMessages()方法是否开启异步发送
func WithAsync(isAsync bool) ProducerOption {
	return func(writer *kafka.Writer) {
		writer.Async = isAsync
	}
}

// WithBatchSize 设置批量大小
func WithBatchSize(batchSize int) ProducerOption {
	return func(writer *kafka.Writer) {
		writer.BatchSize = batchSize
	}
}

// NewKafkaProducer 初始化kafka-go的Producer
func NewKafkaProducer(addr []string, topic string, opts ...ProducerOption) *KafkaProducer {
	// 创建写入器
	writer := &kafka.Writer{
		Addr:  kafka.TCP(addr...), // kafka地址
		Topic: topic,
	}
	// 用户自定义配置
	for _, opt := range opts {
		opt(writer)
	}
	// 创建KafkaProducer
	kap := &KafkaProducer{
		writer: writer,
		// 发送关闭信号的管道，只要向这个管道发送信号，就会关闭writer
		closeChan: make(chan struct{}),
	}
	return kap
}

// Send 异步发送Message
func (kap *KafkaProducer) Send(message kafka.Message) {
	go func() {
		var err error
		// 重试 3 次
		const retry = 3
		for i := 0; i < retry; i++ {
			select {
			case <-kap.closeChan:
				// 收到关闭信号，停止发送
				zap.L().Info("收到关闭信号，停止发送消息",
					zap.String("data", string(message.Value)),
					zap.String("topic", message.Topic))
				return
			default:
				// 发送消息
				err = kap.writer.WriteMessages(context.Background(), message)
				// 记录发送的Message
				zap.L().Info("发送消息",
					zap.String("data", string(message.Value)),
					zap.String("topic", message.Topic))
				// 发送消息成功，退出循环
				if err == nil {
					break
				}
				// 超时错误 DeadlineExceeded，继续循环
				if errors.Is(err, context.DeadlineExceeded) {
					time.Sleep(time.Millisecond * 250 * time.Duration(i+1)) // 指数退避
					continue
				}
				// 其他错误，记录Message
				zap.L().Error("向 Kafka 发送数据失败", zap.Error(err),
					zap.String("data", string(message.Value)),
					zap.String("topic", message.Topic))
			}
		}
		// 如果重试失败，记录错误Message（可以进一步处理，如存储到数据库）
		if err != nil {
			zap.L().Error("发送Message失败，已重试 3 次",
				zap.String("data", string(message.Value)),
				zap.String("topic", message.Topic))
		}
	}()
}

// Stop 停止 Kafka 生产者，优雅关闭
func (kap *KafkaProducer) Stop() error {
	close(kap.closeChan) // 发送关闭信号
	err := kap.writer.Close()
	if err != nil {
		zap.L().Error("关闭 Kafka writer 失败", zap.Error(err))
		return err
	}
	return nil
}
