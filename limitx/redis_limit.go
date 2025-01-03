package limitx

import (
	_ "embed"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
	"time"
)

//go:embed lua/slide_window.lua
var luaSlideWindow string

// RedisSlidingWindowLimiter redis滑动窗口限流
type RedisSlidingWindowLimiter struct {
	cmd      redis.Cmdable // redis客户端
	interval time.Duration // 滑动窗口的大小，时间间隔
	rate     int           // 阈值，允许的请求数量
	// interval内允许rate个请求，比如10s内允许100个请求
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration,
	rate int) *RedisSlidingWindowLimiter {
	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		interval: interval,
		rate:     rate,
	}
}
func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	return r.cmd.Eval(ctx, // 上下文，用于控制请求的生命周期
		luaSlideWindow,            // lua脚本
		[]string{key},             // KEY数组
		r.interval.Milliseconds(), // ARGV数组，第一个元素，时间间隔的毫秒数
		r.rate,                    // ARGV数组，第二个元素，允许的请求数量
		time.Now().UnixMilli()). // 当前时间
		Bool()
}
