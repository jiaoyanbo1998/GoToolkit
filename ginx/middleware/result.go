package middleware

// Result 封装统一的响应结果
type Result[T any] struct {
	Code int    // 状态码
	Msg  string // 状态码描述
	Data T      // 传递给前端的数据
}
