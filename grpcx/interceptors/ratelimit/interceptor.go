package ratelimit

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"webook-rpc/pkg/limitx"
	"webook-rpc/pkg/loggerx"
)

// Interceptor 拦截器
type Interceptor struct {
	// 限流器
	limiter     limitx.Limiter // 整个服务的限流
	key         string
	logger      loggerx.Logger
	serviceName string // 服务名
}

func NewInterceptor(limiter limitx.Limiter, key string, logger loggerx.Logger,
	serviceName string) *Interceptor {
	return &Interceptor{
		limiter:     limiter,
		key:         key,
		logger:      logger,
		serviceName: serviceName,
	}
}

// BuildServerInterceptor 对服务端限流
func (i *Interceptor) BuildServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {
		limit, err := i.limiter.Limit(ctx, i.key)
		if err != nil {
			// 保守法，拒绝请求
			// codes.ResourceExhausted 服务端资源不足
			i.logger.Error("限流器创建失败", loggerx.Error(err),
				loggerx.String("method:", "Interceptor:BuildServerInterceptor"))
			return nil, status.Errorf(codes.ResourceExhausted, "触发限流")
			// 激进策略
			// return handler(ctx, req)
		}
		if limit == true {
			i.logger.Warn("触发限流，可能有人在攻击你的系统",
				loggerx.String("method:", "Interceptor:BuildServerInterceptor"))
			return nil, status.Errorf(codes.ResourceExhausted, "触发限流")
		}
		// 执行下一个拦截器，或者是真实的业务代码
		return handler(ctx, req)
	}
}

// BuildClientInterceptor 对客户端限流
func (i *Interceptor) BuildClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
		limit, err := i.limiter.Limit(ctx, i.key)
		if err != nil {
			// 保守策略
			i.logger.Error("限流器创建失败", loggerx.Error(err))
			return status.Errorf(codes.ResourceExhausted, "触发限流")
			// 激进策略
			// return invoker(ctx, method, req, reply, cc, opts...)
		}
		// 触发限流
		if limit == true {
			return status.Errorf(codes.ResourceExhausted, "触发限流")
		}
		// 执行下一个拦截器，或者是真实的业务代码
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// BuildServerInterceptorService 对某个具体的服务限流
func (i *Interceptor) BuildServerInterceptorService() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (resp any, err error) {
		if strings.HasPrefix(info.FullMethod, "/"+i.serviceName+"/") {
			limit, er := i.limiter.Limit(ctx, "limiter:service"+i.serviceName)
			if er != nil {
				// 保守法，拒绝请求
				// codes.ResourceExhausted 服务端资源不足
				i.logger.Error("限流器创建失败", loggerx.Error(err),
					loggerx.String("method:", "Interceptor:BuildServerInterceptor"))
				return nil, status.Errorf(codes.ResourceExhausted, "触发限流")
				// 激进策略
				// return handler(ctx, req)
			}
			if limit == true {
				i.logger.Warn("触发限流，可能有人在攻击你的系统",
					loggerx.String("method:", "Interceptor:BuildServerInterceptor"))
				return nil, status.Errorf(codes.ResourceExhausted, "触发限流")
			}
		}
		// 执行下一个拦截器，或者是真实的业务代码
		return handler(ctx, req)
	}
}
