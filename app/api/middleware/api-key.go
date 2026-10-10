package middleware

import (
	"inis/app/facade"
	"inis/app/model"
	"net"
	"net/url"
	"strings"
	"time"

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

	item, _ := facade.DB.Model(&model.Config{}).Where("key", key).Find()
	value := item["value"]

	if cacheState {
		go facade.Cache.Set(cacheName, value)
	}

	return value
}

// getApiKeys - 取全部未删除密钥的索引：value(大写) → { id, status, expire_time }
//
// 缓存里存的是**全部**密钥及其状态 / 有效期，而不是"当前可用的那批" —— 有效期随时间变化，
// 若把"可用列表"缓存下来，某把密钥到期后仍会被放行。可用性在每次请求时现算（内存判断，很便宜）。
// 停用 / 删除 / 改有效期都会走 api-keys 的写接口 → 控制器 delCache 会按 `[GET]`、`api-keys`
// 两个标签清掉这里的键（子串匹配，键名里同时含这两段）。
func getApiKeys() map[string]facade.H {
	cacheName := cacheApiKeyPrefix + "[index]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		// 内存缓存里就是原类型；Redis / 文件缓存经过 JSON 往返后是 map[string]any，两种都兼容
		if data, ok := facade.Cache.Get(cacheName).(map[string]facade.H); ok {
			return data
		}
		if raw, ok := facade.Cache.Get(cacheName).(map[string]any); ok {
			result := make(map[string]facade.H, len(raw))
			for value, item := range raw {
				result[value] = cast.ToStringMap(item)
			}
			return result
		}
	}

	items, _ := facade.DB.Model(&[]model.ApiKeys{}).Select()
	index := make(map[string]facade.H, len(items))
	for _, item := range items {
		value := cast.ToString(item["value"])
		if utils.Is.Empty(value) {
			continue
		}
		index[value] = facade.H{
			"id":          cast.ToInt(item["id"]),
			"status":      cast.ToInt(item["status"]),
			"expire_time": cast.ToInt64(item["expire_time"]),
		}
	}

	if cacheState {
		go facade.Cache.Set(cacheName, index)
	}

	return index
}

// isPublicPath 判断是否为公开路径
func isPublicPath(path string) bool {
	publicPaths := []any{"/api/file/rand"}
	return utils.In.Array(path, publicPaths)
}

// ============================== 本站来源判定 ==============================

// isLocalRequest - 请求是否来自「本站自己」
//
// 用途：开启密钥校验后，内嵌在二进制里的 Mellow 前端**不带 i-api-key**，
// 若不区分来源，开关一开浏览器端会全部 403（连后台都进不去）。命中任一即视为本站来源：
//
//	1) TCP 对端为本机 / 内网（127.0.0.1、::1、10.x、172.16-31.x、192.168.x）——
//	   后端只监听内网端口 + 反代转发时这条**可靠**（源 IP 伪造不了）；
//	2) Sec-Fetch-Site: same-origin（现代浏览器的同源指示）；
//	3) Origin / Referer 的 host 与本次请求的 Host 相同（浏览器同源请求）。
//
// ⚠️ 2、3 两条能被脚本伪造（curl -H 'Origin: …'），它们挡的是扫描器与无脑爬虫，
// **不是安全边界**。需要强边界时请让后端只监听内网端口，由反代注入 i-api-key。
//
// 这里用 RemoteAddr 而不是 ctx.ClientIP()：后者会采用 X-Forwarded-For，
// 在直连暴露的部署里可以被伪造成 127.0.0.1 从而绕过校验。
func isLocalRequest(ctx *gin.Context) bool {
	if localAddress(hostOnly(ctx.Request.RemoteAddr)) {
		return true
	}

	if strings.EqualFold(strings.TrimSpace(ctx.Request.Header.Get("Sec-Fetch-Site")), "same-origin") {
		return true
	}

	host := hostOnly(ctx.Request.Host)
	if utils.Is.Empty(host) {
		return false
	}

	for _, raw := range []string{
		ctx.Request.Header.Get("Origin"),
		ctx.Request.Header.Get("Referer"),
	} {
		if utils.Is.Empty(raw) {
			continue
		}
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if strings.EqualFold(hostOnly(parsed.Host), host) {
			return true
		}
	}

	return false
}

// hostOnly - 去掉 host 里的端口，兼容 IPv6 的 [::1]:8080
func hostOnly(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if value, _, err := net.SplitHostPort(host); err == nil {
		return value
	}
	return strings.Trim(host, "[]")
}

// localAddress - 是否为本机 / 内网 / 链路本地地址
func localAddress(host string) bool {
	ip := net.ParseIP(strings.TrimSpace(host))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// ApiKey - 安全校验中间件
//
// 校验顺序：开关关闭 → 全部放行；公开路径 → 放行；本站来源（SYSTEM_API_KEY_LOCAL，
// 默认开）→ 放行；其余必须带合法密钥。
func ApiKey() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		open := cast.ToBool(getConfigValue("SYSTEM_API_KEY"))

		if !open {
			ctx.Next()
			return
		}

		// 公开路径（如 /api/file/rand）不校验密钥
		if isPublicPath(ctx.Request.URL.Path) {
			ctx.Next()
			return
		}

		// 本站来源放行：站内前端（打包进二进制的 Mellow）照常工作，外部调用仍必须带密钥。
		// 关掉 SYSTEM_API_KEY_LOCAL 即回到「所有请求都必须带密钥」。
		if cast.ToBool(getConfigValue("SYSTEM_API_KEY_LOCAL")) && isLocalRequest(ctx) {
			ctx.Next()
			return
		}

		var key string
		if !utils.Is.Empty(ctx.Request.Header.Get("i-api-key")) {
			key = ctx.Request.Header.Get("i-api-key")
		} else {
			key, _ = ctx.GetQuery("i-api-key")
		}

		if utils.Is.Empty(key) {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "禁止非法操作！"), "data": nil})
			ctx.Abort()
			return
		}

		// 密钥值统一大写存储，这里同样转大写后精确匹配（避免调用方贴成小写被拒）
		item, exist := getApiKeys()[strings.ToUpper(key)]
		if !exist {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "禁止非法操作！"), "data": nil})
			ctx.Abort()
			return
		}

		// 停用 / 过期：给出明确原因便于调用方自查（密钥本身已核对通过，不构成信息泄露）
		if cast.ToInt(item["status"]) != model.ApiKeyStatusOn {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "该密钥已停用！"), "data": nil})
			ctx.Abort()
			return
		}
		if expire := cast.ToInt64(item["expire_time"]); expire > 0 && time.Now().Unix() > expire {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "该密钥已过期！"), "data": nil})
			ctx.Abort()
			return
		}

		// 记录用量（异步：累计调用次数 + 最后使用时间，不影响本次请求）
		go model.ApiKeyTouch(cast.ToInt(item["id"]))

		ctx.Next()
	}
}
