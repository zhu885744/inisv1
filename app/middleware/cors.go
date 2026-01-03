package middleware

import (
	"github.com/gin-gonic/gin"   // 引入Gin核心包
	"net/http"                   // 引入HTTP标准库
	"strings"                    // 引入字符串处理库
)

// Cors 跨域中间件
func Cors() gin.HandlerFunc {
	// 返回一个符合Gin中间件规范的函数
	return func(ctx *gin.Context) {
		// 设置跨域相关响应头
		// 预检请求的缓存时间（1800秒=30分钟），减少OPTIONS请求次数
		ctx.Header("Access-Control-Max-Age", "1800")
		// 允许所有源访问（生产环境不建议用*，建议动态匹配前端域名）
		ctx.Header("Access-Control-Allow-Origin", "*")
		// 注释掉了允许携带凭证（如Cookie），如果开启，Allow-Origin不能为*
		// ctx.Header("Access-Control-Allow-Credentials", "true")
		// 设置响应内容类型为JSON，编码UTF-8
		ctx.Header("Content-Type", "application/json; charset=utf-8")
		// 允许的HTTP请求方法
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS, PATCH")
		// 允许前端携带的自定义请求头（如Token、Authorization等）
		ctx.Header("Access-Control-Allow-Headers", "X-Khronos, X-Gorgon, X-Argus, X-Ss-Stub, Token, Authorization, i-api-key, Content-Type, If-Match, If-Modified-Since, If-None-Match, If-Unmodified-Since, X-CSRF-TOKEN, X-Requested-With")
		// 允许前端访问的响应头（扩展默认可访问的响应头）
		ctx.Header("Access-Control-Expose-Headers", "Content-Type, Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")

		// 处理OPTIONS预检请求
		// 判断当前请求方法是否为OPTIONS（前端跨域复杂请求会先发送OPTIONS预检）
		if strings.ToUpper(ctx.Request.Method) == "OPTIONS" {
			// 直接返回204 No Content，终止后续中间件执行
			ctx.AbortWithStatus(http.StatusNoContent)
		}

		// 放行正常请求，执行后续中间件/处理器
		ctx.Next()
	}
}