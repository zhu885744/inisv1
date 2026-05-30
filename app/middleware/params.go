package middleware

import (
	"bytes"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Params常量
const (
	contentTypeJSON            = "application/json"
	contentTypeFormURLEncoded  = "application/x-www-form-urlencoded"
	contentTypeMultipartForm   = "multipart/form-data"
	maxMultipartFormMemory     = 32 << 20
	storageConfigFile         = "config/storage.toml"
	domainCacheKey           = "domain"
)

// Params - 参数处理中间件
func Params() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		go domain(ctx)
		go clientIP(ctx)
		go port(ctx)

		method := ctx.Request.Method
		params := make(map[string]any)

		body, _ := io.ReadAll(ctx.Request.Body)

		content := map[string]any{
			"type":  ctx.GetHeader("Content-Type"),
			"body":  string(body),
			"form":  ctx.Request.Form,
			"query": ctx.Request.URL.Query(),
		}

		if utils.Is.Empty(content["type"]) {
			if method == "GET" || method == "DELETE" {
				content["type"] = contentTypeFormURLEncoded
			} else {
				if !utils.Is.Empty(content["body"]) {
					content["type"] = contentTypeJSON
				} else if !utils.Is.Empty(content["query"]) && !utils.Is.Empty(content["form"]) {
					content["type"] = contentTypeFormURLEncoded
				}
			}
		}

		if utils.In.Array(method, []any{"POST", "PUT", "DELETE", "PATCH"}) {
			if err := ctx.Request.ParseMultipartForm(maxMultipartFormMemory); err != nil {
				if !errors.Is(err, http.ErrNotMultipart) {
				}
			}
		}

		contentType := cast.ToString(content["type"])

		if !utils.Is.Empty(content["body"]) {
			var item map[string]any

			switch {
			case strings.Contains(contentType, contentTypeJSON):
				item = cast.ToStringMap(utils.Json.Decode(string(body)))
				if !utils.Is.Empty(item) {
					for key, val := range item {
						params[key] = val
					}
				}

			case strings.Contains(contentType, contentTypeFormURLEncoded):
				values, err := url.ParseQuery(string(body))
				if err != nil {
					break
				}
				item = utils.Parse.Params(utils.Parse.ParamsBefore(values))
				if !utils.Is.Empty(item) {
					for key, val := range item {
						params[key] = val
					}
				}

			case strings.Contains(contentType, contentTypeMultipartForm):
				bodyBuffer := bytes.NewBufferString(string(body))
				boundary := strings.Split(contentType, "boundary=")

				if len(boundary) != 2 {
					break
				}

				multipartReader := multipart.NewReader(bodyBuffer, boundary[1])
				formData, err := multipartReader.ReadForm(0)
				if err != nil {
					break
				}

				values := url.Values{}
				for key, valuesList := range formData.Value {
					for _, value := range valuesList {
						values.Add(key, value)
					}
				}

				item = utils.Parse.Params(utils.Parse.ParamsBefore(values))
				if !utils.Is.Empty(item) {
					for key, val := range item {
						params[key] = val
					}
				}
			}
		}

		if !utils.Is.Empty(content["query"]) {
			item := utils.Parse.Params(utils.Parse.ParamsBefore(ctx.Request.URL.Query()))
			for key, val := range item {
				params[key] = val
			}
		}
		ctx.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		ctx.Set("params", params)
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

	go func() {
		if cast.ToBool(facade.CacheToml.Get("open")) {
			facade.Cache.Set(domainCacheKey, result, 0)
		}
		ctx.Set(domainCacheKey, result)
		facade.Var.Set(domainCacheKey, result)
		go saveStorageDomain(result)
	}()

	return result
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
	facade.Var.Set("port", result)
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

	reg := regexp.MustCompile(`\${(.+?)}`)
	matches := reg.FindAllStringSubmatch(temp, -1)

	for _, match := range matches {
		temp = strings.Replace(temp, match[0], cast.ToString(facade.StorageToml.Get(match[1])), -1)
	}

	utils.File().Save(strings.NewReader(temp), storageConfigFile)
}
