package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"inis/app/facade"
	"inis/app/middleware"
)

// IsIpBanned 检查IP是否在封禁期内（考虑白名单）
func IsIpBanned(ip string) bool {
	// 白名单IP不受限制
	if middleware.IsIpWhitelisted(ip) {
		return false
	}

	info, ok := middleware.IpBanIndex()[ip]
	return ok && info.Banned(time.Now().Unix())
}

// IpBlack - IP黑名单
//
// 判定走进程内快照（app/middleware/ip-ban.go），每个请求零 DB 查询。
// 此前每个请求都要查一次 ip_black 表，而这张表几分钟才变一次。
func IpBlack() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()

		// 白名单IP不受限制
		if middleware.IsIpWhitelisted(ip) {
			ctx.Next()
			return
		}

		now := time.Now().Unix()

		info, ok := middleware.IpBanIndex()[ip]
		if !ok || !info.Banned(now) {
			ctx.Next()
			return
		}

		// 计算剩余时间
		remaining := ""
		if info.Permanent {
			remaining = facade.Lang(ctx, "，永久封禁")
		} else if left := info.ExpireAt - now; left > 0 {
			hours := left / 3600
			mins := (left % 3600) / 60
			if hours > 0 {
				remaining = facade.Lang(ctx, "，剩余 %d 小时 %d 分钟", hours, mins)
			} else {
				remaining = facade.Lang(ctx, "，剩余 %d 分钟", mins)
			}
		}

		msg := facade.Lang(ctx, "IP已被列入黑名单") + remaining
		ctx.JSON(200, gin.H{"code": 406, "msg": msg, "data": nil})
		ctx.Abort()
	}
}
