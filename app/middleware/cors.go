package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"

	"inis/app/facade"
)

// Cors 跨域中间件（从配置文件和环境变量读取配置）
func Cors() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 核心：直接操作Writer.Header，确保CORS头不被后续逻辑覆盖
		header := ctx.Writer.Header()

		// 从配置文件和环境变量读取CORS配置
		// 环境变量优先于配置文件
		enabled := utils.Env().Get("system.cors.enabled", facade.AppToml.Get("system.cors.enabled", true)).(bool)
		if !enabled {
			ctx.Next()
			return
		}

		// 读取允许的源列表
		allowedOriginsStr := utils.Env().Get("system.cors.allowed_origins", facade.AppToml.Get("system.cors.allowed_origins", "https://zhuxu.asia,http://localhost:3000,http://127.0.0.1:3000")).(string)
		allowedOrigins := make(map[string]bool)
		for _, origin := range strings.Split(allowedOriginsStr, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				allowedOrigins[origin] = true
			}
		}

		// 读取其他CORS配置
		maxAge := int(utils.Env().Get("system.cors.max_age", facade.AppToml.Get("system.cors.max_age", 1800)).(int64))
		allowCredentials := utils.Env().Get("system.cors.allow_credentials", facade.AppToml.Get("system.cors.allow_credentials", true)).(bool)
		allowedMethods := utils.Env().Get("system.cors.allowed_methods", facade.AppToml.Get("system.cors.allowed_methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")).(string)
		allowedHeaders := utils.Env().Get("system.cors.allowed_headers", facade.AppToml.Get("system.cors.allowed_headers", "X-Khronos, X-Gorgon, X-Argus, X-Ss-Stub, Token, Authorization, i-api-key, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-CSRF-TOKEN, X-Requested-With")).(string)
		exposedHeaders := utils.Env().Get("system.cors.exposed_headers", facade.AppToml.Get("system.cors.exposed_headers", "Content-Type, Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")).(string)
		defaultOrigin := utils.Env().Get("system.cors.default_origin", facade.AppToml.Get("system.cors.default_origin", "https://cs.zhuxu.asia")).(string)

		// 1. 基础CORS配置（适配前后端跨域核心需求）
		header.Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
		header.Set("Access-Control-Allow-Credentials", strconv.FormatBool(allowCredentials))
		header.Set("Content-Type", "application/json; charset=utf-8")
		header.Set("Access-Control-Allow-Methods", strings.TrimSpace(allowedMethods))
		header.Set("Access-Control-Allow-Headers", strings.TrimSpace(allowedHeaders))
		header.Set("Access-Control-Expose-Headers", strings.TrimSpace(exposedHeaders))

		// 2. 精准匹配Origin（仅允许配置的源）
		origin := ctx.Request.Header.Get("Origin")
		if origin != "" {
			if allowedOrigins[origin] {
				// 允许的Origin：直接返回请求的Origin
				header.Set("Access-Control-Allow-Origin", origin)
			} else {
				// 非法Origin：返回默认域名（阻断跨域，保证安全）
				header.Set("Access-Control-Allow-Origin", defaultOrigin)
			}
		} else {
			// 无Origin：返回默认域名
			header.Set("Access-Control-Allow-Origin", defaultOrigin)
		}

		// 3. 处理OPTIONS预检请求（前端复杂请求必过的环节）
		if strings.ToUpper(ctx.Request.Method) == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return // 明确终止，避免执行后续逻辑
		}

		// 放行正常请求
		ctx.Next()
	}
}
