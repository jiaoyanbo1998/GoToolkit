package metric

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

type MiddlewareBuilder struct {
	Namespace  string
	Subsystem  string
	Name       string
	Help       string
	InstanceId string
}

func NewMiddlewareBuilder(Namespace, Subsystem, Name, Help,
	InstanceId string) *MiddlewareBuilder {
	return &MiddlewareBuilder{
		Namespace:  Namespace,
		Subsystem:  Subsystem,
		Name:       Name,
		Help:       Help,
		InstanceId: InstanceId,
	}
}

// BuildGinHttpResponseInfo 统计HTTP请求的响应信息
func (m *MiddlewareBuilder) BuildGinHttpResponseInfo() gin.HandlerFunc {
	// 1.统计http请求的响应时间
	// SummaryVec和Summary的区别
	//	 SummaryVec：可以根据变动标签进行分类
	//	 Summary：不可以
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: m.Namespace,               // 命名空间
		Subsystem: m.Subsystem,               // 子系统
		Name:      m.Name + "_response_time", // 指标名称
		Help:      m.Help,                    // 指标描述
		// 常量标签：在指标的生命周期内，标签是不会改变的
		ConstLabels: map[string]string{
			// 实例ID，使用id来区分不同实例
			"instance_id": m.InstanceId,
		},
		// 性能指标：如果实际性能超过这些指标，就会报警
		Objectives: map[float64]float64{
			0.5:  0.05,   // 0.5 == 50%的请求，0.05 == 误差
			0.7:  0.02,   // 0.7 == 70%的请求，0.02 == 误差
			0.9:  0.001,  // 0.9 == 90%的请求，0.001 == 误差
			0.99: 0.0001, // 0.99 == 99%的请求，0.0001 == 误差
		},
		// 变动标签
		//	  method：http请求方式，要监控的请求方式（get，post，delete...）
		//	  pattern：http请求路由，要监控的请求路由
		//	  status：http请求的状态码，标记请求的状态，200成功，404资源不存在，500服务端内部错误
		//    注意：method，pattern和status的笛卡尔积数量不能太大，否则会占用过多内存，造成内存泄露
	}, []string{"method", "pattern", "status"})
	// 注册"采集指标"，告诉prometheus要采集这些指标
	prometheus.MustRegister(summaryVec) // 当Namespace+Subsystem+Name重复，此处会panic
	// 2.统计当前正在执行的http请求的数量
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_active_count",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceId,
		},
	})
	// 注册指标
	prometheus.MustRegister(gauge)
	// 3.统一监控错误码
	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: m.Namespace,
		Subsystem: m.Subsystem,
		Name:      m.Name + "_error_code",
		Help:      m.Help,
		ConstLabels: map[string]string{
			"instance_id": m.InstanceId,
		},
	}, []string{"method", "code"})
	// 注册指标
	prometheus.MustRegister(counterVec)
	return func(ctx *gin.Context) {
		// 记录请求开始的时间
		start := time.Now()
		// 请求数量+1
		gauge.Inc()
		// 即使出现panic，也会执行defer语句
		defer func() {
			// 请求数量-1
			gauge.Dec()
			// 计算请求开始到当前时间的持续时间
			duration := time.Since(start)
			// 获取HTTP请求方法
			method := ctx.Request.Method
			// 获取请求路径
			pattern := ctx.FullPath()
			// 请求路径未找到，返回unknown
			if pattern == "" {
				pattern = "unknown"
			}
			// 获取HTTP请求的响应码
			status := strconv.Itoa(ctx.Writer.Status())
			// 添加"采集指标"
			// 统计请求的响应时间
			summaryVec.WithLabelValues(method, pattern, status).
				Observe(float64(duration.Milliseconds()))
			// 统计错误码
			if ctx.Writer.Status() != 200 {
				counterVec.WithLabelValues(method, status).Inc()
			}
		}()
		// 最终会执行到业务中
		ctx.Next()
	}
}
