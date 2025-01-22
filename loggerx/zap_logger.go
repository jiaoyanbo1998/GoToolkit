package loggerx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

// ZapLogger Logger接口的适配类 --- zap框架实现
type ZapLogger struct {
	zapLogger *zap.Logger // zap.Logger zap框架的核心日志记录器
}

// DefaultConfig 默认配置
type DefaultConfig struct {
	Filename   string        // 日志文件路径
	MaxSize    int           // 日志文件最大体积（MB）
	MaxAge     int           // 日志文件最大保存天数
	MaxBackups int           // 最大保留的日志文件数量
	BufferSize int           // 缓冲区大小
	LogLevel   zapcore.Level // 日志级别
}

// Options 配置选项函数类型
type Options func(*DefaultConfig)

// WithFilename 设置日志文件路径
func WithFilename(filename string) Options {
	return func(config *DefaultConfig) {
		config.Filename = filename
	}
}

// WithMaxSize 设置日志文件最大体积（MB）
func WithMaxSize(maxSize int) Options {
	return func(config *DefaultConfig) {
		config.MaxSize = maxSize
	}
}

// WithMaxAge 设置日志文件最大保存天数
func WithMaxAge(maxAge int) Options {
	return func(config *DefaultConfig) {
		config.MaxAge = maxAge
	}
}

// WithMaxBackups 设置最大保留的日志文件数量
func WithMaxBackups(maxBackups int) Options {
	return func(config *DefaultConfig) {
		config.MaxBackups = maxBackups
	}
}

// WithBufferSize 设置缓冲区大小
func WithBufferSize(bufferSize int) Options {
	return func(config *DefaultConfig) {
		config.BufferSize = bufferSize
	}
}

// WithLogLevel 设置日志级别
func WithLogLevel(logLevel zapcore.Level) Options {
	return func(config *DefaultConfig) {
		config.LogLevel = logLevel
	}
}

// NewDefaultConfig 创建默认配置
func NewDefaultConfig(opts ...Options) *DefaultConfig {
	// 默认配置
	config := &DefaultConfig{
		Filename:   "./log/lifeLog.log", // 日志文件路径
		MaxSize:    10,                  // 日志文件最大体积（MB）
		MaxAge:     30,                  // 日志文件最大保存天数
		MaxBackups: 5,                   // 最大保留的日志文件数量
		BufferSize: 4096,                // 缓冲区大小
		LogLevel:   zapcore.DebugLevel,  // 日志级别
	}
	// 用户自定义配置选项
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// NewZapLogger 使用构造函数初始化ZapLogger
func NewZapLogger(config *DefaultConfig) *ZapLogger {
	// 确保日志文件目录存在
	err := ensureLogFileDir(config.Filename)
	if err != nil {
		panic(err)
	}
	// 构建日志核心组件，支持同时输出到文件和控制台
	core := zapcore.NewTee(
		// 输出到控制台
		zapcore.NewCore(
			getConsoleEncoder(),        // 控制台日志编码器
			zapcore.AddSync(os.Stdout), // 输出到控制台
			config.LogLevel,            // 日志级别
		),
		// 输出到文件
		zapcore.NewCore(
			getJSONEncoder(),         // json格式日志编码器
			getLogWriterSync(config), // 输出到文件
			config.LogLevel,          // 日志级别
		),
	)
	// 创建日志记录器
	logger := zap.New(core, zap.AddCaller()) // 启用调用者信息
	return &ZapLogger{
		zapLogger: logger,
	}
}

// ensureLogFileDir 确保日志文件目录存在
func ensureLogFileDir(filename string) error {
	// 获取日志文件目录
	dir := filepath.Dir(filename)
	// 创建目录
	return os.MkdirAll(dir, os.ModePerm)
}

// getConsoleEncoder 控制台日志编码器
func getConsoleEncoder() zapcore.Encoder {
	// 开发模式
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	// 设置时间格式
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	// 小写带颜色的日志级别
	encoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// getJSONEncoder json日志编码器
func getJSONEncoder() zapcore.Encoder {
	// 开发模式
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	// 设置时间键为"time"
	encoderConfig.TimeKey = "time"
	// 设置时间格式
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	// 小写日志级别
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// getLogWriter 获取，同步日志记录器
func getLogWriter(config *DefaultConfig) zapcore.WriteSyncer {
	// 分片
	lumberLogger := &lumberjack.Logger{
		Filename:   config.Filename,   // 日志文件路径
		MaxSize:    config.MaxSize,    // 日志文件最大体积（MB）
		MaxAge:     config.MaxAge,     // 日志文件最大保存天数
		MaxBackups: config.MaxBackups, // 最大保留的日志文件数量
	}
	return zapcore.AddSync(lumberLogger)
}

// getLogWriterSync 获取，异步日志记录器
func getLogWriterSync(config *DefaultConfig) *zapcore.BufferedWriteSyncer {
	// 分片
	lumberLogger := &lumberjack.Logger{
		Filename:   config.Filename,   // 日志文件路径
		MaxSize:    config.MaxSize,    // 日志文件最大体积（MB）
		MaxAge:     config.MaxAge,     // 日志文件最大保存天数
		MaxBackups: config.MaxBackups, // 最大保留的日志文件数量
	}
	// BufferedWriteSyncer 异步日志记录器
	return &zapcore.BufferedWriteSyncer{
		WS:   zapcore.AddSync(lumberLogger),
		Size: config.BufferSize,
	}
}

// toZapField 将[]Field转为zap.Field
func (z *ZapLogger) toZapField(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	return zapFields
}

// Debug 记录Debug级别的日志
func (z *ZapLogger) Debug(msg string, args ...Field) {
	z.zapLogger.Debug(msg, z.toZapField(args)...)
}

// Info 记录Info级别的日志
func (z *ZapLogger) Info(msg string, args ...Field) {
	z.zapLogger.Info(msg, z.toZapField(args)...)
}

// Warn 记录Warn级别的日志
func (z *ZapLogger) Warn(msg string, args ...Field) {
	z.zapLogger.Warn(msg, z.toZapField(args)...)
}

// Error 记录Error级别的日志
func (z *ZapLogger) Error(msg string, args ...Field) {
	z.zapLogger.Error(msg, z.toZapField(args)...)
}
