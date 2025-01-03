package gormx

import (
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
	"time"
)

type Callbacks struct {
	// 展示数据的百分位（比如99线，999线，平均数，中位数）
	summaryVec *prometheus.SummaryVec
	// Prometheus指标的命名空间名字
	Namespace string
	// Prometheus指标的子系统名字
	Subsystem string
	// Prometheus指标的名字
	Name string
	// Prometheus指标的帮助文本，解释这个指标有什么用
	Help string
	// Prometheus实例ID
	InstanceId string
}

func (c *Callbacks) before() func(db *gorm.DB) {
	Before := func(db *gorm.DB) {
		start := time.Now()
		// 记录开始时间
		db.Set("start_time", start)
	}
	return Before
}
func (c *Callbacks) after(typeStr string) func(db *gorm.DB) {
	After := func(db *gorm.DB) {
		// 获取开始时间
		val, _ := db.Get("start_time")
		// 转换为time.Time类型
		startTime, ok := val.(time.Time)
		if !ok {
			// 系统错误
			return
		}
		// 获取表名
		table := db.Statement.Table
		// 如果表名为空，则设置为unknown
		if table == "" {
			table = "unknown"
		}
		// 统计耗时，上报prometheus
		// time.Since(startTime)，计算startTime到now的时间间隔
		c.summaryVec.WithLabelValues(typeStr, table).
			// 被观测的值（数据库执行耗时）
			Observe(float64(time.Since(startTime).Milliseconds()))
	}
	return After
}

func (c *Callbacks) RegisterAll(db *gorm.DB) error {
	// 创建一个summary，用来统数据库语句执行计耗时
	summaryVec := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: c.Namespace,
		Subsystem: c.Subsystem,
		Name:      c.Name,
		Help:      c.Help,
		Objectives: map[float64]float64{
			0.5:  0.01,
			0.75: 0.001,
			0.8:  0.0001,
			0.9:  0.00001,
			0.99: 0.000001,
		},
		ConstLabels: map[string]string{
			"db_name":     db.Name(),
			"instance_id": c.InstanceId,
		},
	}, []string{"type", "table"})
	prometheus.MustRegister(summaryVec)
	c.summaryVec = summaryVec

	// create
	err := db.Callback().Create().Before("*").
		Register("prometheus_create_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Create().After("*").
		Register("prometheus_create_after", c.after("create"))
	if err != nil {
		panic(err)
	}

	// 修改
	err = db.Callback().Update().Before("*").
		Register("prometheus_update_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Update().After("*").
		Register("prometheus_update_after", c.after("update"))
	if err != nil {
		panic(err)
	}

	// 删除
	err = db.Callback().Delete().Before("*").
		Register("prometheus_delete_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Delete().After("*").
		Register("prometheus_delete_after", c.after("delete"))
	if err != nil {
		panic(err)
	}

	// 查询
	err = db.Callback().Query().Before("*").
		Register("prometheus_query_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Query().After("*").
		Register("prometheus_query_after", c.after("query"))
	if err != nil {
		panic(err)
	}

	// 原始的sql查询
	err = db.Callback().Raw().Before("*").
		Register("prometheus_raw_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Raw().After("*").
		Register("prometheus_raw_after", c.after("raw"))
	if err != nil {
		panic(err)
	}

	// 返回一条语句
	// row := db.Raw("SELECT name FROM users WHERE id = ?", 1).Row()
	err = db.Callback().Row().Before("*").
		Register("prometheus_row_before", c.before())
	if err != nil {
		panic(err)
	}
	err = db.Callback().Row().After("*").
		Register("prometheus_row_after", c.after("row"))
	if err != nil {
		panic(err)
	}

	return nil
}
