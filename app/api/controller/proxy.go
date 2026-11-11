package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"strings"
)

type Proxy struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Proxy) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Proxy) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - PUT请求本体
func (this *Proxy) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - DELETE请求本体
func (this *Proxy) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - GET请求本体
func (this *Proxy) INDEX(ctx *gin.Context) {

	params := this.params(ctx)
	filter := []string{"i-url", "i-type"}

	// SSRF 防护：校验目标地址，禁止访问内网与非 http/https 协议
	targetURL := cast.ToString(params["i-url"])
	if err := facade.Comm.IsSafeOutboundURL(targetURL); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// // 创建 CookieJar
	// jar, _ := cookiejar.New(nil)
	//
	// client := &http.Client{
	// 	Jar: jar,
	// }
	//
	// // cookie 数据不为空
	// if !utils.Is.Empty(ctx.Request.Cookies()) {
	// 	client.Jar.SetCookies(ctx.Request.URL, ctx.Request.Cookies())
	// }

	item := utils.Curl(utils.CurlRequest{
		Client:  proxyHTTPClient(),
		Url:     cast.ToString(params["i-url"]),
		Method:  strings.ToUpper(ctx.Request.Method),
		Headers: this.forwardHeaders(ctx),
		Data: 	 this.data(ctx, filter),
		Query:   this.query(ctx, filter),
	}).Send()

	if item.Error != nil {
		ctx.JSON(500, item.Error.Error())
		return
	}

	switch strings.ToUpper(cast.ToString(params["i-type"])) {
	case "JSON":
		ctx.JSON(200, item.Json)
	case "TEXT":
		ctx.JSON(200, item.Text)
	case "BYTE":
		ctx.JSON(200, item.Byte)
	default:
		ctx.JSON(200, item.Json)
	}
}

// proxyAllowedHeaders - 允许转发给目标站点的请求头白名单
//
// 只转发「描述内容格式」的头。Cookie / Authorization / X-Forwarded-* / Host
// 一律不转发：目标地址由调用方控制，把本站用户的会话凭证送过去
// 等于让任何有该接口权限的账号把别人的登录态导出到自己的服务器。
var proxyAllowedHeaders = []string{
	"Content-Type",
	"Content-Length",
	"Accept",
	"Accept-Language",
	"Accept-Encoding",
	"User-Agent",
}

// forwardHeaders 按白名单挑出可转发的请求头
func (this *Proxy) forwardHeaders(ctx *gin.Context) map[string]any {
	result := make(map[string]any, len(proxyAllowedHeaders))
	for _, key := range proxyAllowedHeaders {
		if value := strings.TrimSpace(ctx.GetHeader(key)); value != "" {
			result[key] = value
		}
	}
	return result
}

// proxyHTTPClient - 代理出站客户端：逐跳校验重定向目标
//
// IsSafeOutboundURL 只能校验首跳 URL。目标站返回 302 指向 127.0.0.1:8642
// 或 169.254.169.254 时，默认 client 会直接跟过去 —— 这是一条完整的 SSRF 绕过链路。
// 这里接管重定向，每一跳都重新校验，并限制最大跳数。
func proxyHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("重定向次数过多！")
			}
			return facade.Comm.IsSafeOutboundURL(req.URL.String())
		},
	}
}

// query - 获取 query 位置的参数
func (this *Proxy) query(ctx *gin.Context, filter []string) (result map[string]any) {

	result = make(map[string]any)

	// query 数据不为空
	if !utils.Is.Empty(ctx.Request.URL.Query()) {
		item := utils.Parse.Params(utils.Parse.ParamsBefore(ctx.Request.URL.Query()))
		// 合并 params 参数
		for key, val := range item {
			if !utils.InArray(key, filter) {
				result[key] = val
			}
		}
	}

	return
}

// data - 获取 body 位置的参数
func (this *Proxy) data(ctx *gin.Context, filter []string) (result map[string]any) {

	result = make(map[string]any)

	// 只获取 key
	for key := range this.query(ctx, filter) {
		filter = append(filter, key)
	}

	for key, val := range this.params(ctx) {
		if !utils.InArray(key, filter) {
			result[key] = val
		}
	}

	return
}