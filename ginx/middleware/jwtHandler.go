package middleware

import (
	"GoToolkit/loggerx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

type JWTHandler struct {
	logger loggerx.Logger
}

func NewJWTHandler(l loggerx.Logger) *JWTHandler {
	return &JWTHandler{
		logger: l,
	}
}

type UserClaims struct {
	Id       int64  // 用户id
	NickName string // 用户名
	jwt.RegisteredClaims
	SessionId string
}

func (jwtHandler *JWTHandler) GetUserInfo(ctx *gin.Context) (UserClaims, bool) {
	userClaims, exists := ctx.Get("userClaims")
	if !exists {
		jwtHandler.logger.Error("用户信息不存在")
		// 返回false，表示用户信息不存在
		return UserClaims{}, false
	}
	userInfo, ok := userClaims.(UserClaims)
	if !ok {
		ctx.JSON(http.StatusOK, Result[string]{
			Code: 500,
			Msg:  "系统错误",
			Data: "error",
		})
		jwtHandler.logger.Error("用户信息断言失败")
		return UserClaims{}, false
	}
	// 返回true，表示用户信息存在
	return userInfo, true
}

// GetTokenString 获取加密后的token
func (jwtHandler *JWTHandler) GetTokenString(ctx *gin.Context) (string, bool) {
	author := ctx.GetHeader("Authorization")
	splitN := strings.SplitN(author, " ", 2)
	if len(splitN) != 2 || splitN[0] != "Bearer" {
		return "", false
	}
	return splitN[1], true
}

func (jwtHandler *JWTHandler) SetJwt(ctx *gin.Context, userClaims UserClaims, flag bool) {
	uid := uuid.New().String()
	userClaims.SessionId = uid
	// 初始化长短token
	if flag {
		jwtHandler.SetLongJwt(ctx, userClaims, time.Now().Add(time.Hour*24*7))
		jwtHandler.SetShortJwt(ctx, userClaims, time.Now().Add(time.Minute*10))
	}
	// 短token续约
	jwtHandler.SetShortJwt(ctx, userClaims, time.Now().Add(time.Minute*10))
}
func (jwtHandler *JWTHandler) SetLongJwt(ctx *gin.Context, userClaims UserClaims, t time.Time) bool {
	// 设置过期时间
	userClaims.ExpiresAt = jwt.NewNumericDate(t)
	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, userClaims)
	// token加密
	tokenString, err := token.SignedString([]byte("A1q22s5Od1Y8m2v51s2u5B20R5sN10F1"))
	if err != nil {
		ctx.JSON(http.StatusOK, Result[string]{
			Code: 500,
			Msg:  "系统错误",
			Data: "error",
		})
		jwtHandler.logger.Error("长token设置失败，setLongJwt方法",
			loggerx.Error(err))
		return false
	}
	ctx.Header("jwt-long-token", tokenString)
	return true
}

func (jwtHandler *JWTHandler) SetShortJwt(ctx *gin.Context, userClaims UserClaims, t time.Time) bool {
	// 设置过期时间
	userClaims.ExpiresAt = jwt.NewNumericDate(t)
	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, userClaims)
	// token加密
	tokenString, err := token.SignedString([]byte("A1q22s5Od1Y8m2v51s2u5B20R5sN10F1"))
	if err != nil {
		ctx.JSON(http.StatusOK, Result[string]{
			Code: 500,
			Msg:  "系统错误",
			Data: "error",
		})
		jwtHandler.logger.Error("短token设置失败，setShortJwt方法",
			loggerx.Error(err))
		return false
	}
	ctx.Header("jwt-short-token", tokenString)
	return true
}
