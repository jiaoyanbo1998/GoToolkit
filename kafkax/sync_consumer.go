package kafkax

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// KafkaSyncConsumer 异步消费
type KafkaSyncConsumer[T any] struct {
	reader  *kafka.Reader // kafka读取器
	timeout time.Duration // 超时时间
	// 参数1：从kafka消费者中，接收到的消息
	// 参数2：消息中存储的值
	handler func(val T) error // 处理函数
}

// Option 是一个函数类型，用于设置 KafkaSyncConsumer 的选项
type Option func(*kafka.ReaderConfig)

// NewKafkaConsumer 创建一个kafka消费者
func NewKafkaConsumer[T any](brokers []string, groupId, topic string,
	timeout time.Duration, handler func(val T) error, opts ...Option) *KafkaSyncConsumer[T] {
	// 默认配置
	config := kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupId,
		Topic:    topic,
		MinBytes: 10e3, // 默认 最小字节数 10KB
		MaxBytes: 10e6, // 默认 最大字节数 10MB
	}
	// 用户自定义配置
	for _, opt := range opts {
		opt(&config)
	}
	// 创建 KafkaSyncConsumer
	ksc := &KafkaSyncConsumer[T]{
		reader:  kafka.NewReader(config),
		timeout: timeout,
		handler: handler,
	}
	// 启动一个goroutine，读取数据
	go ksc.ReadMsg()
	return ksc
}

// WithMinBytes 设置每次读取的最小字节数
func WithMinBytes(minBytes int) Option {
	return func(cfg *kafka.ReaderConfig) {
		cfg.MinBytes = minBytes
	}
}

// WithMaxBytes 设置每次读取的最大字节数
func WithMaxBytes(maxBytes int) Option {
	return func(cfg *kafka.ReaderConfig) {
		cfg.MaxBytes = maxBytes
	}
}

// ReadMsg 从kafka中读取数据
func (kc *KafkaSyncConsumer[T]) ReadMsg() {
	// 关闭kafka读取器
	defer kc.reader.Close()
	// channel，1个容量
	sigChan := make(chan os.Signal, 1)
	// syscall.SIGINT 用户在终端按下Ctrl+C，表示中断程序
	// syscall.SIGTERM 系统或管理员发送，表示程序终止
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// 无限循环
	for {
		select {
		// 监听到终止信号，退出循环
		case <-sigChan:
			zap.L().Info("接收到终止信号，停止消费")
			return
		default:
			// 读取消息
			ctx, cancel := context.WithTimeout(context.Background(), kc.timeout)
			// 从kafka中读取消息
			message, err := kc.reader.ReadMessage(ctx)
			if err != nil {
				if err == context.DeadlineExceeded {
					zap.L().Warn("读取消息超时")
					continue
				}
				zap.L().Error("kafka读取数据失败", zap.Error(err))
				continue
			}
			fmt.Printf("消息在offset %d: %s = %s\n", message.Offset,
				string(message.Key), string(message.Value))
			// 调用处理函数
			// json反序列化
			var val T
			err = json.Unmarshal(message.Value, &val)
			if err != nil {
				zap.L().Error("消息反序列化失败", zap.Error(err))
				continue
			}
			err = kc.handler(val)
			if err != nil {
				zap.L().Error("调用处理接口失败", zap.Error(err))
				continue
			}
			// 标记消费过的消息
			err = kc.reader.CommitMessages(ctx, message)
			if err != nil {
				zap.L().Error("标记消息失败", zap.Error(err))
			}
			cancel()
		}
	}
}

func (kc *KafkaSyncConsumer[T]) Stop() {
	// 关闭kafka读取器
	kc.reader.Close()
}
