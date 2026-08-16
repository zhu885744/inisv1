package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"

	"inis/app/facade"
)

// CORS常量配置
const (
	defaultCorsMaxAge                = 1800
	defaultCorsAllowCredentials      = true
	defaultCorsAllowedMethods        = "GET, POST, PATCH, PUT, DELETE, OPTIONS"
	defaultCorsAllowedHeaders        = "X-Khronos, X-Gorgon, X-Argus, X-Ss-Stub, Token, Authorization, i-api-key, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-CSRF-TOKEN, X-Requested-With"
	defaultCorsExposedHeaders        = "Content-Type, Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers"
	defaultCorsDefaultOrigin         = "http://localhost:8642"
	defaultCorsAllowedOriginsDefault = "http://localhost:3000,http://127.0.0.1:3000"
)

// Cors - 跨域中间件
func Cors() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		header := ctx.Writer.Header()

		enabled := utils.Env().Get("system.cors.enabled", facade.AppToml.Get("system.cors.enabled", true)).(bool)
		if !enabled {
			ctx.Next()
			return
		}

		allowedOriginsStr := utils.Env().Get("system.cors.allowed_origins", facade.AppToml.Get("system.cors.allowed_origins", defaultCorsAllowedOriginsDefault)).(string)
		allowedOrigins := make(map[string]bool)
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins[origin] = true
			}
		}

		maxAge := cast.ToInt(utils.Env().Get("system.cors.max_age", facade.AppToml.Get("system.cors.max_age", defaultCorsMaxAge)))
		allowCredentials := cast.ToBool(utils.Env().Get("system.cors.allow_credentials", facade.AppToml.Get("system.cors.allow_credentials", defaultCorsAllowCredentials)))
		allowedMethods := cast.ToString(utils.Env().Get("system.cors.allowed_methods", facade.AppToml.Get("system.cors.allowed_methods", defaultCorsAllowedMethods)))
		allowedHeaders := cast.ToString(utils.Env().Get("system.cors.allowed_headers", facade.AppToml.Get("system.cors.allowed_headers", defaultCorsAllowedHeaders)))
		exposedHeaders := cast.ToString(utils.Env().Get("system.cors.exposed_headers", facade.AppToml.Get("system.cors.exposed_headers", defaultCorsExposedHeaders)))
		defaultOrigin := cast.ToString(utils.Env().Get("system.cors.default_origin", facade.AppToml.Get("system.cors.default_origin", defaultCorsDefaultOrigin)))

		header.Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
		header.Set("Access-Control-Allow-Credentials", strconv.FormatBool(allowCredentials))
		header.Set("Content-Type", "application/json; charset=utf-8")
		header.Set("Access-Control-Allow-Methods", strings.TrimSpace(allowedMethods))
		header.Set("Access-Control-Allow-Headers", strings.TrimSpace(allowedHeaders))
		header.Set("Access-Control-Expose-Headers", strings.TrimSpace(exposedHeaders))

		origin := ctx.Request.Header.Get("Origin")
		// 仅当来源在允许列表中时才回显该来源；否则不设置跨域头（浏览器将阻止跨域读取）
		if origin != "" {
			if allowedOrigins[origin] {
				header.Set("Access-Control-Allow-Origin", origin)
				// 非通配符来源时，允许携带凭证
				header.Set("Vary", "Origin")
			} else {
				// 非白名单来源：不放行跨域，OPTIONS 预检直接拒绝
				if strings.ToUpper(ctx.Request.Method) == "OPTIONS" {
					ctx.AbortWithStatus(http.StatusForbidden)
					return
				}
				ctx.Next()
				return
			}
		} else {
			// 同源/非跨域请求（无 Origin 头），回退到默认来源
			header.Set("Access-Control-Allow-Origin", defaultOrigin)
		}

		if strings.ToUpper(ctx.Request.Method) == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
