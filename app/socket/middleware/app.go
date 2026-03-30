package middleware

import (
	"crypto/md5"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// App - socket 中间件
func App(ctx *gin.Context) {
	// 获取用户信息
	user, exists := ctx.Get("user")

	if exists {
		// 登录用户，使用用户 ID 作为唯一标识
		uid := cast.ToInt(user.(map[string]any)["id"])
		ctx.Set("client_id", fmt.Sprintf("user_%d", uid))
	} else {
		// 访客用户，生成基于浏览器和 IP 的唯一标识
		clientId := generateGuestId(ctx)
		ctx.Set("client_id", fmt.Sprintf("guest_%s", clientId))
	}

	ctx.Next()
}

// generateGuestId - 生成访客唯一 ID
func generateGuestId(ctx *gin.Context) string {
	// 收集访客信息
	userAgent := ctx.Request.UserAgent()
	ip := ctx.ClientIP()
	host := ctx.Request.Host
	accept := ctx.Request.Header.Get("Accept")
	acceptLanguage := ctx.Request.Header.Get("Accept-Language")

	// 组合信息生成唯一 ID
	info := fmt.Sprintf("%s%s%s%s%s", userAgent, ip, host, accept, acceptLanguage)
	hash := md5.Sum([]byte(info))

	return fmt.Sprintf("%x", hash)
}
