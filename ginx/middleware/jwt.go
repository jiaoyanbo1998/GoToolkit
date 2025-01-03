package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"net/http"
	"webook-grpc/pkg/loggerx"
)

type JwtMiddlewareBuilder struct {
	paths  []string
	logger loggerx.Logger
	cmd    redis.Cmdable
	JWTHandler
}

func NewJwtMiddlewareBuilder(logger loggerx.Logger, cmd redis.Cmdable) *JwtMiddlewareBuilder {
	return &JwtMiddlewareBuilder{
		logger: logger,
		cmd:    cmd,
	}
}

func (j *JwtMiddlewareBuilder) IgnorePath(path string) *JwtMiddlewareBuilder {
	j.paths = append(j.paths, path)
	return j
}
func (j *JwtMiddlewareBuilder) Builder() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 检查当前路由是否需要登录校验
		// 1.不需要登录校验
		for _, path := range j.paths {
			if path == ctx.Request.URL.Path {
				j.logger.Info("当前路径不需要登录校验",
					loggerx.String("path", ctx.Request.URL.Path))
				return
			}
		}
		// 2.需要登录校验
		// 获取请求头信息
		tokenString, ok := j.GetTokenString(ctx)
		if !ok {
			ctx.JSON(http.StatusUnauthorized, Result[string]{
				Code: 401,
				Msg:  "系统错误",
				Data: "error",
			})
			j.logger.Error("jwt-token获取失败，没有传入jwt，或jwt被篡改，"+
				"middleware包下的jwt方法",
				loggerx.String("path", ctx.Request.URL.Path),
				loggerx.String("token", tokenString))
			ctx.Abort()
			return
		}
		userClaims := UserClaims{}
		// 解析token
		token, err := jwt.ParseWithClaims(tokenString, &userClaims, func(token *jwt.Token) (interface{}, error) {
			return []byte("A1q22s5Od1Y8m2v51s2u5B20R5sN10F1"), nil
		})
		// 校验token是否合法
		// token.Valid == false token非法
		// token.Valid == true token合法
		if err != nil || token.Valid == false || token == nil {
			ctx.JSON(http.StatusUnauthorized, Result[string]{
				Code: 401,
				Msg:  "系统错误",
				Data: "error",
			})
			j.logger.Error("jwt-token非法",
				loggerx.String("path", ctx.Request.URL.Path),
				loggerx.Error(err))
			ctx.Abort()
			return
		}
		// 检查用户是否已经退出
		key := fmt.Sprintf("logout:sessionId:%s", userClaims.SessionId)
		// 判断key是否存在，存在返回1，不存在返回0
		//	存在，代表用户已经退出
		exists, _ := j.cmd.Exists(ctx, key).Result()
		if exists == 1 {
			ctx.JSON(http.StatusUnauthorized, Result[string]{
				Code: 500,
				Msg:  "系统错误",
				Data: "error",
			})
			j.logger.Error("用户已经退出",
				loggerx.String("path", ctx.Request.URL.Path),
				loggerx.String("sessionId", userClaims.SessionId))
			ctx.Abort()
			return
		}
		// 将用户信息保存到上下文
		ctx.Set("userClaims", userClaims)
	}
}
