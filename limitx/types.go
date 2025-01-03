package limitx

import "context"

// Limiter 限流器
type Limiter interface {
	// key 要限流的对象
	// bool 返回true表示限流，返回false表示不限流
	Limit(ctx context.Context, key string) (bool, error)
}
