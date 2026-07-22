package middleware

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

var cacheIpWhitePrefix = "[GET][ip-white][column]"

// getWhitelist 获取白名单列表（带缓存）
func getWhitelist() []string {
	cacheName := cacheIpWhitePrefix
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringSlice(facade.Cache.Get(cacheName))
	}

	list, _ := facade.DB.Model(&model.IpWhite{}).Column("ip")
	column := cast.ToStringSlice(utils.ArrayEmpty(utils.ArrayUnique(cast.ToStringSlice(list))))

	if cacheState {
		go facade.Cache.Set(cacheName, column)
	}

	return column
}

// IsIpWhitelisted 检查IP是否在白名单中
func IsIpWhitelisted(ip string) bool {
	whitelist := getWhitelist()
	return utils.InArray[string](ip, whitelist)
}
