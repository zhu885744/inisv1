package middleware

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LocalOnly - 仅允许本机（loopback）访问的中间件
// 用于保护 /dev/ 等本地运维接口，防止外网直达
func LocalOnly() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		parsed := net.ParseIP(ip)
		if parsed == nil || !parsed.IsLoopback() {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "禁止访问！",
				"data": nil,
			})
			return
		}
		ctx.Next()
	}
}
