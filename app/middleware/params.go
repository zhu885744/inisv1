package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"io"
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

		body, _ := io.ReadAll(ctx.Request.Body)

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

// domain - 获取域名
func domain(ctx *gin.Context) (result string) {
	host := ctx.Request.Header.Get("X-Host")
	host = utils.Ternary[string](utils.Is.Empty(host), ctx.Request.Host, host)

	info := []string{"localhost", "80"}
	if strings.Contains(host, ":") {
		info = strings.Split(host, ":")
	} else {
		info[0] = host
	}

	scheme := ctx.Request.Header.Get("X-Scheme")
	if utils.Is.Empty(scheme) {
		scheme = utils.Ternary[string](cast.ToInt(info[1]) == 443, "https", "http")
	}

	result = scheme + "://" + info[0]
	if !utils.InArray[int](cast.ToInt(info[1]), []int{80, 443}) {
		result += ":" + info[1]
	}

	ctx.Set(domainCacheKey, result)

	// 只在域名变化时写缓存 / 全局变量 / storage.toml。
	//
	// 旧实现是每个请求都 go func(){ Cache.Set(...); Var.Set(...); saveStorageDomain(...) }：
	// 一次页面几十个静态请求就是几十次缓存写；local.domain 为空时还会每个请求重写一次
	// storage.toml（几十次磁盘写）。这些值只跟 Host 有关，没必要每请求都刷新。
	if cast.ToString(facade.Var.Get(domainCacheKey)) == result {
		return
	}

	facade.Var.Set(domainCacheKey, result)

	if cast.ToBool(facade.CacheToml.Get("open")) {
		facade.Cache.Set(domainCacheKey, result, 0)
	}

	saveStorageDomain(result)

	return
}

// clientIP - 获取客户端IP
func clientIP(ctx *gin.Context) (result string) {
	result = ctx.Request.Header.Get("X-Real-IP")
	if utils.Is.Empty(result) {
		result = ctx.Request.Header.Get("X-Forwarded-For")
	}
	if utils.Is.Empty(result) {
		result = ctx.ClientIP()
	}

	ctx.Set("ip", result)
	return result
}

// port - 获取端口号
func port(ctx *gin.Context) (result int) {
	host := ctx.Request.Header.Get("X-Host")
	host = utils.Ternary[string](utils.Is.Empty(host), ctx.Request.Host, host)

	if strings.Contains(host, ":") {
		info := strings.Split(host, ":")
		result = cast.ToInt(info[1])
	}

	if utils.Is.Empty(result) {
		result = utils.Ternary[int](ctx.Request.Header.Get("X-Scheme") == "https", 443, 80)
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
