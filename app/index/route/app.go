package route

import (
	"inis/app/facade"
	"inis/app/index/controller"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

func Route(Gin *gin.Engine) {
	// 拦截全局panic
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{
				"error":     err,
				"stack":     string(debug.Stack()),
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line,
			}, "首页模板渲染发生错误")
		}
	}()

	// 模板状态接口（供前端轮询）
	Gin.GET("/api/template-status", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"ok":     utils.File().Exist("public/index.html"),
			"msg":    "模板状态实时检测",
			"reload": utils.File().Exist("public/index.html"),
		})
	})

	// 首页路由：无侵入式兜底
	Gin.GET("/", func(ctx *gin.Context) {
		templatePath := "public/index.html"
		if !utils.File().Exist(templatePath) {
			ctx.String(200, `页面模板（public/index.html）不存在，请联系管理员进行处理！`)
			return
		}
		// 模板存在，执行原有控制器逻辑
		controller.Index(ctx)
	})
}
