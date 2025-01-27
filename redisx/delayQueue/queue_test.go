package delayQueue

import (
	"GoToolkit/loggerx"
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

// 模拟任务处理的 handler
func taskHandler(ctx context.Context, data []byte) error {
	fmt.Printf("Processing task: %s\n", string(data))
	return nil
}

func TestQueue(t *testing.T) {
	// 配置 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // 替换为你的 Redis 地址
	})
	defer rdb.Close()

	// 创建队列实例
	config := loggerx.NewDefaultConfig()
	logger := loggerx.NewZapLogger(config)
	q := NewQueue(rdb, "test_queue", logger,
		WithPollInterval(500*time.Millisecond), // 轮询间隔设置为 500 毫秒
		WithConcurrency(5),                     // 设置并发数为 5
	)

	// 启动消费者协程
	q.Start(taskHandler)

	// 模拟向队列添加任务
	for i := 0; i < 10; i++ {
		payload := fmt.Sprintf("Task #%d", i+1)
		if err := q.Add(context.Background(), payload, 2*time.Second); err != nil {
			t.Errorf("Failed to add task: %v", err)
		} else {
			fmt.Printf("Task #%d added to queue\n", i+1)
		}
	}
	// 停止队列
	q.Stop()
}
