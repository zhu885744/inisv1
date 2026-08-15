package route

import (
	"github.com/gin-gonic/gin"
	"inis/app/dev/controller"
	global "inis/app/middleware"
)

// 路由组前缀
const (
	apiPrefix    = "/dev/"
)

// 中间件配置
// 注意：/dev/install 为安装引导接口，需允许外网访问以完成安装流程；
// 其安全性由 Install 中间件控制（已安装后 /dev/install 会返回 412 禁止访问）
var installDevMiddleware = []gin.HandlerFunc{
	global.Params(),
}

// info 路由仅做参数解析，敏感方法（system/device/renew/kill）在控制器内部校验本机访问
var infoDevMiddleware = []gin.HandlerFunc{
	global.Params(),
}

// registerDevRoutes 注册开发路由
func registerDevRoutes(group *gin.RouterGroup, controllers map[string]controller.ApiInterface) {
	for key, item := range controllers {
		group.Any(key, item.INDEX)
		group.GET(key+"/:method", item.IGET)
		group.PUT(key+"/:method", item.IPUT)
		group.POST(key+"/:method", item.IPOST)
		group.DELETE(key+"/:method", item.IDEL)
	}
}

// Route - 路由配置
func Route(Gin *gin.Engine) {
	// install 接口：仅本机可访问
	installGroup := Gin.Group(apiPrefix, installDevMiddleware...)
	registerDevRoutes(installGroup, map[string]controller.ApiInterface{
		"install": &controller.Install{},
	})

	// info 接口：time/version 公开，敏感方法内部校验本机
	infoGroup := Gin.Group(apiPrefix, infoDevMiddleware...)
	registerDevRoutes(infoGroup, map[string]controller.ApiInterface{
		"info": &controller.Info{},
	})
}
