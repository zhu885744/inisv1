package middleware

import (
	"time"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/middleware"
	"inis/app/model"
)

var cacheIpBlackPrefix = "[GET][ip-black][column]"

// getBlacklist 获取黑名单列表（带缓存）
func getBlacklist() []string {
	cacheName := cacheIpBlackPrefix
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringSlice(facade.Cache.Get(cacheName))
	}

	list := facade.DB.Model(&model.IpBlack{}).Column("ip")
	column := cast.ToStringSlice(utils.ArrayEmpty(utils.ArrayUnique(cast.ToStringSlice(list))))

	if cacheState {
		go facade.Cache.Set(cacheName, column)
	}

	return column
}

// IsIpBanned 检查IP是否在封禁期内（考虑白名单）
func IsIpBanned(ip string) bool {
	// 白名单IP不受限制
	if middleware.IsIpWhitelisted(ip) {
		return false
	}

	// 查询黑名单记录
	var ipBlack model.IpBlack
	facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Scan(&ipBlack)

	if ipBlack.Id == 0 {
		return false
	}

	return ipBlack.IsBanned()
}

// IpBlack - IP黑名单
func IpBlack() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()

		// 白名单IP不受限制
		if middleware.IsIpWhitelisted(ip) {
			ctx.Next()
			return
		}

		// 检查是否在封禁期内
		var ipBlack model.IpBlack
		facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Scan(&ipBlack)

		if ipBlack.Id > 0 && ipBlack.IsBanned() {
			// 计算剩余时间
			remaining := ""
			if !ipBlack.IsPermanent && ipBlack.ExpireTime > 0 {
				remainingSec := ipBlack.ExpireTime - time.Now().Unix()
				if remainingSec > 0 {
					hours := remainingSec / 3600
					mins := (remainingSec % 3600) / 60
					if hours > 0 {
						remaining = facade.Lang(ctx, "，剩余 %d 小时 %d 分钟", hours, mins)
					} else {
						remaining = facade.Lang(ctx, "，剩余 %d 分钟", mins)
					}
				}
			} else if ipBlack.IsPermanent {
				remaining = facade.Lang(ctx, "，永久封禁")
			}

			msg := facade.Lang(ctx, "IP已被列入黑名单") + remaining
			ctx.JSON(200, gin.H{"code": 406, "msg": msg, "data": nil})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
