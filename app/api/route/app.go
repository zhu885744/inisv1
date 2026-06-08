package route

import (
	"fmt"
	"inis/app/api/controller"
	middle "inis/app/api/middleware"
	global "inis/app/middleware"

	"github.com/gin-gonic/gin"
)

// 默认路由配置
var defaultMiddleware = []gin.HandlerFunc{
	middle.IpBlack(),
	global.QpsPoint(),
	global.QpsGlobal(),
	global.Params(),
	middle.Jwt(),
	middle.Rule(),
	middle.ApiKey(),
}

// 所有可用的控制器
var controllers = map[string]controller.ApiInterface{
	"exp":           &controller.EXP{},
	"test":          &controller.Test{},
	"comm":          &controller.Comm{},
	"toml":          &controller.Toml{},
	"file":          &controller.File{},
	"tags":          &controller.Tags{},
	"pages":         &controller.Pages{},
	"users":         &controller.Users{},
	"oauth":         &controller.OAuth{},
	"links":         &controller.Links{},
	"proxy":         &controller.Proxy{},
	"level":         &controller.Level{},
	"banner":        &controller.Banner{},
	"config":        &controller.Config{},
	"upgrade":       &controller.Upgrade{},
	"article":       &controller.Article{},
	"comment":       &controller.Comment{},
	"placard":       &controller.Placard{},
	"api-keys":      &controller.ApiKeys{},
	"ip-black":      &controller.IpBlack{},
	"ip-white":      &controller.IpWhite{},
	"qps-warn":      &controller.QpsWarn{},
	"auth-group":    &controller.AuthGroup{},
	"auth-pages":    &controller.AuthPages{},
	"auth-rules":    &controller.AuthRules{},
	"links-group":   &controller.LinksGroup{},
	"article-group": &controller.ArticleGroup{},
	"search":        &controller.Search{},
	"rss":           &controller.Rss{},
}

// registerRoutes 注册路由
func registerRoutes(group gin.IRoutes, controllers map[string]controller.ApiInterface) {
	for key, item := range controllers {
		group.Any(key, item.INDEX)
		group.GET(fmt.Sprintf("%s/:method", key), item.IGET)
		group.PUT(fmt.Sprintf("%s/:method", key), item.IPUT)
		group.POST(fmt.Sprintf("%s/:method", key), item.IPOST)
		group.DELETE(fmt.Sprintf("%s/:method", key), item.IDEL)
	}
}

// Route - 路由配置
// @Summary API路由入口
// @Description 配置所有API路由及其中间件
// @Tags Route
func Route(Gin *gin.Engine) {
	group := Gin.Group("/api/", defaultMiddleware...)
	registerRoutes(group, controllers)
}
