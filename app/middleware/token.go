package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"

	"inis/app/facade"
)

// Token常量
const (
	defaultTokenValue = "0147."
)

// Token - 简单token验证中间件
func Token() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		auth := ctx.Request.Header.Get("Authorization")
		if utils.Is.Empty(auth) {
			auth = ctx.Query("token")
		}

		if utils.Is.Empty(auth) {
			ctx.JSON(200, gin.H{"data": nil, "code": 401, "msg": "未授权"})
			ctx.Abort()
			return
		}

		// 从配置读取 token（可配置），使用恒定时间比较防止时序攻击
		token := cast.ToString(facade.AppToml.Get("app.token", ""))
		if token == "" {
			token = defaultTokenValue
		}

		if subtle.ConstantTimeCompare([]byte(auth), []byte(token)) != 1 {
			ctx.JSON(200, gin.H{"data": nil, "code": 403, "msg": "无权限"})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
