package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Params常量
const (
	contentTypeJSON           = "application/json"
	contentTypeFormURLEncoded = "application/x-www-form-urlencoded"
	contentTypeMultipartForm  = "multipart/form-data"
	// maxMultipartFormMemory multipart 表单在内存里的上限，超出部分由 net/http 落到临时文件
	//
	// 原来是 32MB：一个 30MB 的图片上传会整块留在堆上，加上下面那次重复读取，
	// 峰值内存是文件大小的 2~3 倍。降到 8MB 后大文件走磁盘，小图仍然在内存里。
	maxMultipartFormMemory = 8 << 20
	storageConfigFile      = "config/storage.toml"
	domainCacheKey         = "domain"

	// maxBodyBytes 非 multipart 请求体上限
	//
	// Params() 会把整个 body 读进内存，不设上限时一个超大 JSON body
	// 就能把进程内存打满（未登录接口也可触发）。16MB 足够覆盖正常的
	// 表单 / JSON 请求；文件上传走 multipart 分支，不受此限制。
	maxBodyBytes = 16 << 20

	// maxMultipartBytes multipart 请求的硬上限（防磁盘打满的兜底）
	//
	// 单文件大小由附件配置（storage.toml 的 attachment.max_file_size）约束，
	// 这里只挡住「无上限写盘」这种最坏情况，阈值刻意放宽到 1GB。
	maxMultipartBytes = 1 << 30
)

// domainVarPattern storage.toml 模板里的 ${xxx} 占位符
var domainVarPattern = regexp.MustCompile(`\${(.+?)}`)

// Params - 参数处理中间件
func Params() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 这三个值都很便宜，同步算即可。
		// 旧实现是 go domain(ctx) / go clientIP(ctx) / go port(ctx)：
		// 每个请求 3~4 个 goroutine（domain 内部还再开一个），既浪费栈内存，
		// 又和后续 handler 读 ctx 形成数据竞争（goroutine 的写入可能晚于 handler 的读取）。
		clientIP(ctx)
		port(ctx)
		domain(ctx)

		params := make(map[string]any)
		contentType := ctx.GetHeader("Content-Type")

		// multipart 交给 net/http 自己流式解析，不再整体读进内存。
		//
		// 旧实现：io.ReadAll 把整个请求体（含文件字节）读进堆 -> string(body) 再复制一份
		// -> multipart.ReadForm(0) 又完整解析一次（文件部分同样在内存）。
		// 现在只解析表单字段到 params，文件本身留给上传控制器用 ctx.MultipartForm()，
		// 超过内存阈值的部分由 net/http 自动落临时文件。
		if strings.Contains(contentType, contentTypeMultipartForm) {
			ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxMultipartBytes)
			if err := ctx.Request.ParseMultipartForm(maxMultipartFormMemory); err == nil && ctx.Request.MultipartForm != nil {
				values := url.Values{}
				for key, list := range ctx.Request.MultipartForm.Value {
					for _, value := range list {
						values.Add(key, value)
					}
				}
				mergeParams(params, parseFormValues(values))
			}

			mergeParams(params, parseQuery(ctx.Request.URL.Query()))
			ctx.Set("params", params)
			return
		}

		// 上限由 MaxBytesReader 保证：超限时 ReadAll 返回错误，
		// 直接终止请求，绝不能用半截 body 继续解析出「看起来正常」的参数
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxBodyBytes)
		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusOK, gin.H{
				"code": http.StatusRequestEntityTooLarge,
				"msg":  facade.Lang(ctx, "请求内容过大！"),
				"data": nil,
			})
			return
		}

		if len(body) > 0 {
			switch {
			case strings.Contains(contentType, contentTypeJSON):
				item := cast.ToStringMap(utils.Json.Decode(body))
				// 兜底：有些客户端会把表单体配上 JSON 头发出来
				// （axios 对 URLSearchParams 用的是「不覆盖实例默认头」，而实例默认头恰好是 application/json），
				// 此时 JSON 解不出来，若直接放过会导致「全部参数静默丢失」，表现为「xxx 不能为空！」。
				// 解不出 JSON 就退一步按表单解析。
				if utils.Is.Empty(item) {
					item = parseFormBody(body)
				}
				mergeParams(params, item)

			case strings.Contains(contentType, contentTypeFormURLEncoded):
				mergeParams(params, parseFormBody(body))

			default:
				// 未声明 Content-Type：先按 JSON 试，解不出来再按表单试（与原兜底逻辑一致）
				item := cast.ToStringMap(utils.Json.Decode(body))
				if utils.Is.Empty(item) {
					item = parseFormBody(body)
				}
				mergeParams(params, item)
			}
		}

		mergeParams(params, parseQuery(ctx.Request.URL.Query()))

		// 归还请求体，保证后续 handler 还能读（bytes.NewReader 不复制底层数组）
		ctx.Request.Body = io.NopCloser(bytes.NewReader(body))
		ctx.Set("params", params)
	}
}

// parseFormBody 解析 urlencoded 请求体
func parseFormBody(body []byte) map[string]any {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil
	}
	return parseFormValues(values)
}

// parseQuery 解析 URL query
func parseQuery(values url.Values) map[string]any {
	if len(values) == 0 {
		return nil
	}
	return parseFormValues(values)
}

// parseFormValues 统一走 go-utils 的类型转换（"1,2,3" -> 切片、JSON 字符串 -> 结构等）
func parseFormValues(values url.Values) map[string]any {
	if len(values) == 0 {
		return nil
	}
	return utils.Parse.Params(utils.Parse.ParamsBefore(values))
}

// mergeParams 把 src 合并进 dst
func mergeParams(dst map[string]any, src map[string]any) {
	for key, val := range src {
		dst[key] = val
	}
}

// hostPattern 合法主机名：域名 / IPv4（可带端口）/ [IPv6]（可带端口）
var hostPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?(:\d{1,5})?$|^\[[0-9a-fA-F:.]+\](:\d{1,5})?$`)

// sanitizeHost 校验请求里的 Host
//
// 只放行主机名字符集，挡掉带换行 / 引号 / 路径的畸形值 —— 这些值会进全局域名
// 缓存，甚至被写进 storage.toml，不能原样落盘。
func sanitizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > 260 || !hostPattern.MatchString(host) {
		return ""
	}
	return host
}

// remoteIsLocal 请求是否来自本机 / 内网（与 ApiKey 的本站来源判定同一口径）
func remoteIsLocal(ctx *gin.Context) bool {
	raw, _, err := net.SplitHostPort(ctx.Request.RemoteAddr)
	if err != nil {
		raw = ctx.Request.RemoteAddr
	}

	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// requestHost 取本次请求的主机名（已校验）
//
// X-Host 只在本机 / 内网反向代理发起时才采信：反代会把 Host 改写成后端地址，
// 真实域名放在 X-Host 里；公网直连的请求带 X-Host 一律忽略，防伪造。
func requestHost(ctx *gin.Context) string {
	if remoteIsLocal(ctx) {
		if host := sanitizeHost(ctx.GetHeader("X-Host")); host != "" {
			return host
		}
	}
	return sanitizeHost(ctx.Request.Host)
}

// splitHostPort 拆分 host[:port]，兼容 IPv6 的 [::1]:8080
func splitHostPort(value string) (host, port string) {
	if value == "" {
		return "", ""
	}
	if h, p, err := net.SplitHostPort(value); err == nil {
		return h, p
	}
	return value, ""
}

// stripScheme 去掉 URL 的 http(s):// 前缀
func stripScheme(value string) string {
	value = strings.TrimPrefix(value, "https://")
	return strings.TrimPrefix(value, "http://")
}

// configuredSiteURL 站点已配置的权威域名（storage.toml 的 local.domain）
//
// 返回带 scheme 的完整地址（如 https://example.com），未配置时返回空串。
func configuredSiteURL() string {
	value := strings.TrimSpace(cast.ToString(facade.StorageToml.Get("local.domain")))
	if value == "" {
		return ""
	}

	value = strings.TrimRight(value, "/")
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return ""
	}

	return value
}

// isLocalHostName 主机名是否指向本机 / 内网（localhost、私有 IP）
//
// 这类值只可能出现在「本地安装、还没换正式域名」的阶段，不能当作权威域名。
func isLocalHostName(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	return isLocalOrPrivateIP(host)
}

// rememberDomain 域名变化时同步全局变量与缓存（值没变就什么都不做）
func rememberDomain(ctx *gin.Context, value string) string {
	ctx.Set(domainCacheKey, value)

	if cast.ToString(facade.Var.Get(domainCacheKey)) == value {
		return value
	}

	facade.Var.Set(domainCacheKey, value)

	if cast.ToBool(facade.CacheToml.Get("open")) {
		facade.Cache.Set(domainCacheKey, value, 0)
	}

	return value
}

// domain - 获取域名
func domain(ctx *gin.Context) (result string) {

	host, port := splitHostPort(requestHost(ctx))
	configured := configuredSiteURL()

	// 站点已配置「正式域名」时，只有请求的主机名与配置一致才采信请求
	//（允许端口 / 协议不同，给反代、本地调试留空间）；主机名不一致一律用配置值。
	//
	// 这是 Host 头投毒的关键修复点。原实现无条件用 Host / X-Host 拼域名，
	// 任何人发一个 `Host: evil.com` 就能改写全局域名缓存与 storage.toml，
	// 之后所有 {{localhost}} 资源地址、邮件里的链接都会指向攻击者的域名。
	//
	// 配置值仍是 localhost / 127.0.0.1 / 内网 IP 时不做权威判定：
	// 那是「安装时自动学到的临时值」，必须允许后续用真实域名自愈。
	configuredHost, configuredPort := splitHostPort(stripScheme(configured))
	authoritative := configured != "" && !isLocalHostName(configuredHost)

	if authoritative {
		if host == "" || !strings.EqualFold(host, configuredHost) {
			return rememberDomain(ctx, configured)
		}
		if port == "" {
			port = configuredPort
		}
	}

	if host == "" {
		host = "localhost"
	}

	scheme := "http"
	if strings.HasPrefix(configured, "https://") && authoritative {
		scheme = "https"
	}
	if remoteIsLocal(ctx) {
		if header := strings.ToLower(strings.TrimSpace(ctx.GetHeader("X-Scheme"))); header == "https" || header == "http" {
			scheme = header
		}
	}
	if port == "443" {
		scheme = "https"
	}

	result = scheme + "://" + host
	if port != "" && port != "80" && port != "443" {
		result += ":" + port
	}

	// 只有「可信来源」或「与已配置域名一致」时才把结果写进全局变量与缓存：
	//   - 已配置正式域名：同主机请求，写进去也是同一个值；
	//   - 本机 / 内网反代：首次安装、反代转发都属于这类，需要学习域名；
	//   - 公网直连且还没有配置：只用在本请求的 ctx 里，绝不改全局
	//     —— 否则任意请求都能用一个伪造的 Host 覆盖全站域名（缓存投毒）。
	if authoritative || remoteIsLocal(ctx) {
		rememberDomain(ctx, result)
	} else {
		ctx.Set(domainCacheKey, result)
	}

	// 只有「还没有权威域名」且来源可信时，才把学到的域名写进 storage.toml。
	if !authoritative && remoteIsLocal(ctx) {
		saveStorageDomain(result)
	}

	return
}

// clientIP - 获取客户端IP
//
// 统一走 gin 的 ClientIP：它只在「请求来自可信代理」（app.toml 的
// app.trusted_proxies）时才采信 X-Forwarded-For / X-Real-IP，与 IpBlack / QPS
// 的风控口径一致。原实现无条件采信这两个头，任何客户端都能伪造 IP ——
// 既能绕过限流与封禁，也能把别人的 IP 顶进黑名单。
func clientIP(ctx *gin.Context) (result string) {
	result = ctx.ClientIP()
	ctx.Set("ip", result)
	return result
}

// port - 获取端口号
func port(ctx *gin.Context) (result int) {
	_, requestPort := splitHostPort(requestHost(ctx))
	if requestPort != "" {
		result = cast.ToInt(requestPort)
	}

	// 请求里没带端口时，用已配置域名的端口（与 domain() 的口径保持一致）
	if result == 0 {
		if _, configuredPort := splitHostPort(stripScheme(configuredSiteURL())); configuredPort != "" {
			result = cast.ToInt(configuredPort)
		}
	}

	if result == 0 {
		switch {
		case strings.HasPrefix(configuredSiteURL(), "https://"):
			result = 443
		case remoteIsLocal(ctx) && ctx.GetHeader("X-Scheme") == "https":
			result = 443
		default:
			result = 80
		}
	}

	ctx.Set("port", result)

	// 同样只在变化时写全局变量：这个值每个请求都会算一遍，但几乎从不改变
	if cast.ToInt(facade.Var.Get("port")) != result {
		facade.Var.Set("port", result)
	}

	return result
}

// saveStorageDomain - 保存存储域名
func saveStorageDomain(domain any) {
	local := facade.StorageToml.Get("local.domain")
	if !utils.Is.Empty(local) {
		return
	}

	temp := facade.TempStorage
	temp = utils.Replace(temp, map[string]any{
		"${local.domain}": domain,
	})

	matches := domainVarPattern.FindAllStringSubmatch(temp, -1)

	for _, match := range matches {
		temp = strings.Replace(temp, match[0], cast.ToString(facade.StorageToml.Get(match[1])), -1)
	}

	utils.File().Save(strings.NewReader(temp), storageConfigFile)
}
