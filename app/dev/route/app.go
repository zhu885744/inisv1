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
var defaultDevMiddleware = []gin.HandlerFunc{
	global.Params(),
}

// 控制器映射
var devControllers = map[string]controller.ApiInterface{
	"info":    &controller.Info{},
	"install": &controller.Install{},
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
	devGroup := Gin.Group(apiPrefix, defaultDevMiddleware...)
	registerDevRoutes(devGroup, devControllers)
}
