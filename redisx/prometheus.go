package redisx

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"net"
	"strconv"
	"time"
)

type PrometheusRedisHook struct {
	svc *prometheus.SummaryVec
}

func NewPrometheusRedisHook(opts prometheus.SummaryOpts) *PrometheusRedisHook {
	summaryVec := prometheus.NewSummaryVec(opts,
		// 变量标签：在指标的生命周期内，标签是不会改变的
		//   cmd：redis命令，get，set，.....
		//   key_exist：key是否存在
		//  	        key_exist == false代表key不存在，缓存未命中，
		//			    key_exist == true代表key存在，缓存命中
		[]string{"cmd", "key_exist"})
	// 将创建的summaryVec注册到Prometheus监控系统中，这样Prometheus就可以采集到这些数据
	prometheus.MustRegister(summaryVec)
	return &PrometheusRedisHook{
		svc: summaryVec,
	}
}

// DialHook 服务器和redis建立连接时，执行的钩子函数
func (p *PrometheusRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		// 执行下一个钩子函数
		return next(ctx, network, addr)
	}
}

// ProcessHook 执行redis命令前/后，执行的钩子函数
//		redis命令，get，set，.....
func (p *PrometheusRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// 在redis命令执行前，执行一些操作
		// 记录开始时间
		start := time.Now()
		var err error
		// key是否存在
		var keyExist bool
		defer func() {
			// redis命令，执行了多久
			// 计算从开始，到now，经过的时间间隔
			duration := time.Since(start).Milliseconds()
			// 判断缓存是否命中
			if err == redis.Nil {
				// key不存在，缓存未命中
				keyExist = false
			} else {
				// key存在，缓存命中
				keyExist = true
			}
			// 更新监控指标
			//    cmd.Name() 获取命令名称
			//    strconv.FormatBool(key_exist) 转换为字符串
			p.svc.WithLabelValues(cmd.Name(), strconv.FormatBool(keyExist)).
				// redis命令执行的时间
				Observe(float64(duration))
		}()
		// 执行下一个钩子函数
		err = next(ctx, cmd)
		// 在redis命令执行结束后，执行一些操作
		return err
	}
}

// ProcessPipelineHook 执行redis管道命令前/后，执行的钩子函数
func (p *PrometheusRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// 执行下一个钩子函数
		return next(ctx, cmds)
	}
}
