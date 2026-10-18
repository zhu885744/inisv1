package config

import (
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

// registerPprof 注册运行时性能分析路由（仅 app.debug = true 时启用）
//
// 常用命令：
//
//	go tool pprof http://127.0.0.1:8642/debug/pprof/heap     # 内存占用来源
//	go tool pprof http://127.0.0.1:8642/debug/pprof/profile  # 30 秒 CPU 采样
//	go tool pprof http://127.0.0.1:8642/debug/pprof/goroutine # goroutine 泄漏
//	go tool pprof --alloc_space http://127.0.0.1:8642/debug/pprof/heap  # 累计分配量
//
// 注意：pprof 会暴露进程内部信息，生产环境请保持 app.debug = false。
func registerPprof() {
	Gin.GET("/debug/pprof/", gin.WrapF(pprof.Index))
	Gin.GET("/debug/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	Gin.GET("/debug/pprof/profile", gin.WrapF(pprof.Profile))
	Gin.GET("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	Gin.POST("/debug/pprof/symbol", gin.WrapF(pprof.Symbol))
	Gin.GET("/debug/pprof/trace", gin.WrapF(pprof.Trace))

	// 其余预置 profile（allocs / block / goroutine / heap / mutex / threadcreate）
	Gin.GET("/debug/pprof/:name", func(ctx *gin.Context) {
		pprof.Handler(ctx.Param("name")).ServeHTTP(ctx.Writer, ctx.Request)
	})
}
