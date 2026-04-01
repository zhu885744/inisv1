package controller

import (
	"inis/app/facade"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

func Index(ctx *gin.Context) {

	ctx.Header("Content-Type", "text/html; charset=utf-8")

	// 直接读取最新的HTML文件内容
	htmlContent, err := os.ReadFile("public/index.html")
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "读取index.html文件失败")
		ctx.String(200, `页面模板（public/index.html）读取失败，请联系管理员进行处理！`)
		return
	}

	// 检查运行目录是否存在 install.lock 文件
	if utils.File().Exist("install.lock") {
		// 直接返回HTML内容
		ctx.String(200, string(htmlContent))
		return
	}

	// 直接返回HTML内容
	ctx.String(200, string(htmlContent))
}
