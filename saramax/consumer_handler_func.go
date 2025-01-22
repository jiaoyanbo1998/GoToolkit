package saramax

import (
	"GoToolkit/loggerx"
	"encoding/json"
	"github.com/IBM/sarama"
)

// Handler 泛型结构体，用于处理kafka消费者信息
type Handler[T any] struct {
	logger loggerx.Logger
	// 消费处理函数
	fn func(msg *sarama.ConsumerMessage, t T) error
}

// NewHandler 使用构造方法，创建一个Handler
func NewHandler[T any](l loggerx.Logger,
	fn func(msg *sarama.ConsumerMessage, t T) error) *Handler[T] {
	return &Handler[T]{
		logger: l,
		fn:     fn,
	}
}

// ConsumeClaim 消费Kafka消息
// 参数：session 消费者组会话
// 参数：claim 消费者组的分区
func (h *Handler[T]) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	// 获取分区中的所有消息
	messages := claim.Messages()
	// 循环处理消息
	for message := range messages {
		// json序列化
		var t T
		err := json.Unmarshal(message.Value, &t)
		if err != nil {
			// json序列化失败，记录日志
			h.logger.Error("反序列化消息体失败",
				loggerx.String("topic", message.Topic),
				loggerx.Int("partition", int(message.Partition)),
				loggerx.Int("offset", int(message.Offset)),
				loggerx.Error(err))
			// 不中断，继续下一个
			continue
		}
		// 调用处理函数
		err = h.fn(message, t)
		if err != nil {
			h.logger.Error("处理消息失败",
				loggerx.String("topic", message.Topic),
				loggerx.Int("partition", int(message.Partition)),
				loggerx.Int("offset", int(message.Offset)),
				loggerx.Error(err))
		} else {
			// 标记消息，已被消费
			session.MarkMessage(message, "")
		}
	}
	return nil
}

// Setup 消费者组会话开始前进行初始化工作
func (h *Handler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 消费者组会话结束后进行清理工作
func (h *Handler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}
