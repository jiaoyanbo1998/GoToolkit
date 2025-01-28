package delayQueue

import (
	"GoToolkit/loggerx"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

// 获取单个任务（Lua脚本实现原子操作）
//go:embed task.lua
var luaScript string

// Queue 延迟队列主体
type Queue struct {
	redis    redis.UniversalClient // 支持多种Redis部署模式的客户端
	queueKey string                // Redis键前缀

	// 配置参数
	pollInterval   time.Duration  // 轮询间隔
	handlerTimeout time.Duration  // 处理超时时间
	concurrency    int            // 并发数
	logger         loggerx.Logger // 日志记录器

	ctx     context.Context    // 上下文
	cancel  context.CancelFunc // 取消函数
	stopped chan struct{}      // 停止信号
}

// NewQueue 创建新队列实例
func NewQueue(rdb redis.UniversalClient, queueName string,
	logger loggerx.Logger, opts ...Option) *Queue {
	// 默认配置
	q := &Queue{
		redis:        rdb,
		queueKey:     queueName,
		pollInterval: time.Second,
		concurrency:  10,
		logger:       logger,
		stopped:      make(chan struct{}),
	}
	// 自定义配置
	for _, opt := range opts {
		opt(q)
	}
	// 初始化上下文和取消函数
	q.ctx, q.cancel = context.WithCancel(context.Background())
	return q
}

// Option 配置选项类型
type Option func(*Queue)

// WithPollInterval 设置轮询间隔
func WithPollInterval(d time.Duration) Option {
	return func(q *Queue) {
		q.pollInterval = d
	}
}

// WithHandlerTimeout 设置处理超时时间
func WithHandlerTimeout(d time.Duration) Option {
	return func(q *Queue) {
		q.handlerTimeout = d
	}
}

// WithConcurrency 设置并发数
func WithConcurrency(n int) Option {
	return func(q *Queue) {
		q.concurrency = n
	}
}

// WithLogger 设置自定义日志
func WithLogger(logger loggerx.Logger) Option {
	return func(q *Queue) {
		q.logger = logger
	}
}

// Add 添加延迟任务
func (q *Queue) Add(ctx context.Context, payload interface{}, delay time.Duration) error {
	// 生成，唯一任务ID
	taskID := uuid.New().String()
	// json序列化
	taskData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}
	// TxPipeline 事务管道，允许多个命令一起执行，减少网络开销
	pipe := q.redis.TxPipeline()
	// Zset延迟队列，score任务的延迟时间，member任务ID
	pipe.ZAdd(ctx, q.queueKey+":delayed", redis.Z{
		Score:  float64(time.Now().Add(delay).Unix()), // 延迟时间的 Unix 时间戳
		Member: taskID,
	})
	// Hash存储任务数据
	pipe.HSet(ctx, q.queueKey+":tasks", taskID, taskData)
	// 执行事务
	_, err = pipe.Exec(ctx)
	return err
}

// Start 启动消费者协程
func (q *Queue) Start(handler func(context.Context, []byte) error) {
	go q.run(handler)
}

// Stop 优雅停止
func (q *Queue) Stop() {
	q.cancel() // 取消上下文
	<-q.stopped
}

// run 核心运行逻辑
func (q *Queue) run(handler func(context.Context, []byte) error) {
	// 关闭通道
	defer close(q.stopped)
	// 启动轮询任务
	sem := make(chan struct{}, q.concurrency)
	ticker := time.NewTicker(q.pollInterval)
	defer ticker.Stop()
	// 循环处理任务
	for {
		select {
		case <-q.ctx.Done(): // 上下文取消
			q.logger.Info("结束：", loggerx.String("queueKey", q.queueKey))
			return
		case <-ticker.C: // 轮询间隔
			q.processBatch(sem, handler)
		}
	}
}

// processBatch 处理批次任务
func (q *Queue) processBatch(sem chan struct{}, handler func(context.Context, []byte) error) {
	ctx := context.Background() // 上下文
	now := time.Now().Unix()    // 当前时间戳
	// 循环处理任务
	for {
		select {
		case sem <- struct{}{}: // 信号量控制并发数
			// 获取单个任务
			taskID, err := q.fetchTask(ctx, now)
			if err != nil {
				<-sem // 释放信号量
				if err == redis.Nil {
					return
				}
				q.logger.Error("Fetch task error: %v", loggerx.Error(err))
				return
			}
			// 处理任务
			go q.handleTask(ctx, taskID, handler, sem)
		default:
			return
		}
	}
}

// fetchTask 获取单个任务
func (q *Queue) fetchTask(ctx context.Context, now int64) (string, error) {
	// 执行Lua脚本
	val, err := redis.NewScript(luaScript).Run(ctx, q.redis,
		[]string{q.queueKey + ":delayed"}, now).Text()
	// redis.Nil 表示没有任务
	if err != nil && err != redis.Nil {
		return "", fmt.Errorf("lua script error: %w", err)
	}
	return val, nil
}

// handleTask 处理单个任务
func (q *Queue) handleTask(
	ctx context.Context,
	taskID string,
	handler func(context.Context, []byte) error,
	sem chan struct{},
) {
	// 释放信号量
	defer func() {
		<-sem
	}()
	// 设置处理超时时间
	ctx, cancel := context.WithTimeout(ctx, q.handlerTimeout)
	defer cancel()

	// 获取任务数据
	data, err := q.redis.HGet(ctx, q.queueKey+":tasks", taskID).Bytes()
	if err != nil {
		q.logger.Error("Get task data error: %v", loggerx.Error(err))
		return
	}

	// 执行用户处理逻辑
	err = handler(ctx, data)
	if err != nil {
		q.logger.Error("error: %v",
			loggerx.Error(err),
			loggerx.String("Handle task %s", taskID))
		return
	}

	// 清理任务数据
	if _, err := q.redis.HDel(ctx, q.queueKey+":tasks", taskID).Result(); err != nil {
		q.logger.Error("error: %v",
			loggerx.Error(err),
			loggerx.String("Delete task %s", taskID))
	}
}
