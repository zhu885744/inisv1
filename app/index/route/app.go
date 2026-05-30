package route

import (
	"inis/app/facade"
	"inis/app/index/controller"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

// 路由常量
const (
	templateStatusPath = "/api/template-status"
	indexPath          = "/"
	installLockFile    = "install.lock"
	templateFile       = "public/index.html"
)

// Route - 路由配置
func Route(Gin *gin.Engine) {
	defer handlePanic()

	Gin.GET(templateStatusPath, func(ctx *gin.Context) {
		templateExists := utils.File().Exist(templateFile)
		ctx.JSON(200, gin.H{
			"ok":     templateExists,
			"msg":    "模板状态实时检测",
			"reload": templateExists,
		})
	})

	Gin.GET(indexPath, func(ctx *gin.Context) {
		if !utils.File().Exist(templateFile) {
			ctx.String(200, "页面模板（public/index.html）不存在，请联系管理员进行处理！")
			return
		}
		controller.Index(ctx)
	})
}

// handlePanic - 处理全局panic
func handlePanic() {
	if err := recover(); err != nil {
		facade.Log.Error(map[string]any{
			"error":     err,
			"stack":     string(debug.Stack()),
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "首页模板渲染发生错误")
	}
}
