package middleware

import (
	"net/netip"
	"strings"

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
	// 本机 / 内网地址永远放行：
	// 没有配置可信代理时，ClientIP 拿到的可能是 Nginx / 容器的内网地址，
	// 一旦把这个地址计入限流，整站访客会共享同一个配额，最终一起被拉黑。
	if isLocalOrPrivateIP(ip) {
		return true
	}

	whitelist := getWhitelist()
	return utils.InArray[string](ip, whitelist)
}

// isLocalOrPrivateIP - 判断是否为本机 / 内网地址
func isLocalOrPrivateIP(ip string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsUnspecified() {
		return true
	}

	// 运营商级 NAT（100.64.0.0/10）：netip 的 IsPrivate 不包含这一段，
	// 但容器 / 部分云厂商的内网地址会落在这里
	if addr.Is4() {
		octets := addr.As4()
		if octets[0] == 100 && octets[1] >= 64 && octets[1] <= 127 {
			return true
		}
	}

	return false
}
