package middleware

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

var (
	cacheConfigPrefix = "config"
	cacheApiKeyPrefix = "[GET]/api/api-keys/column"
)

// getConfigValue 获取配置值（带缓存）
func getConfigValue(key string) any {
	cacheName := cacheConfigPrefix + "[" + key + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return facade.Cache.Get(cacheName)
	}

	item := facade.DB.Model(&model.Config{}).Where("key", key).Find()
	value := item["value"]

	if cacheState {
		go facade.Cache.Set(cacheName, value)
	}

	return value
}

// getApiKeys 获取所有API密钥（带缓存）
func getApiKeys() []string {
	cacheName := cacheApiKeyPrefix + "[value]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringSlice(facade.Cache.Get(cacheName))
	}

	keys := cast.ToStringSlice(facade.DB.Model(&model.ApiKeys{}).Column("value"))

	if cacheState {
		go facade.Cache.Set(cacheName, keys)
	}

	return keys
}

// isPublicPath 判断是否为公开路径
func isPublicPath(path string) bool {
	publicPaths := []any{"/api/file/rand"}
	return utils.In.Array(path, publicPaths)
}

// ApiKey - 安全校验中间件
func ApiKey() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		open := cast.ToBool(getConfigValue("SYSTEM_API_KEY"))

		if !open {
			ctx.Next()
			return
		}

		var key string
		if !utils.Is.Empty(ctx.Request.Header.Get("i-api-key")) {
			key = ctx.Request.Header.Get("i-api-key")
		} else {
			key, _ = ctx.GetQuery("i-api-key")
		}

		if utils.Is.Empty(key) && !isPublicPath(ctx.Request.URL.Path) {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "禁止非法操作！"), "data": nil})
			ctx.Abort()
			return
		}

		keys := getApiKeys()

		if !utils.InArray(key, keys) && !isPublicPath(ctx.Request.URL.Path) {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "禁止非法操作！"), "data": nil})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
