package loggerx

import "time"

// Logger 日志接口
type Logger interface {
	// Debug 记录Debug级别的日志
	Debug(msg string, args ...Field)
	// Info 记录Info级别的日志
	Info(msg string, args ...Field)
	// Warn 记录Warn级别的日志
	Warn(msg string, args ...Field)
	// Error 记录Error级别的日志
	Error(msg string, args ...Field)
}

// Field 日志字段
type Field struct {
	Key   string
	Value any
}

// String 构造Field
func String(key string, val string) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

// Error 构造Field
func Error(err error) Field {
	return Field{
		Key:   "error",
		Value: err,
	}
}

// Int 构造Field
func Int(key string, value int) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Int64 构造Field
func Int64(key string, value int64) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Float32 构造Field
func Float32(key string, value float32) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Float64 构造Field
func Float64(key string, value float64) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Bool 构造Field
func Bool(key string, value bool) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Time 构造Field
func Time(key string, value time.Time) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}

// Any 构造Field
func Any(key string, value any) Field {
	return Field{
		Key:   key,
		Value: value,
	}
}
