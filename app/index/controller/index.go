package controller

import (
	"inis/app/facade"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

// Index - 首页控制器
func Index(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/html; charset=utf-8")

	htmlContent, err := os.ReadFile("public/index.html")
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "读取index.html文件失败")
		ctx.String(200, "页面模板（public/index.html）读取失败，请联系管理员进行处理！")
		return
	}

	if !utils.File().Exist("install.lock") {
		ctx.String(200, string(htmlContent))
		return
	}

	ctx.String(200, string(htmlContent))
}
