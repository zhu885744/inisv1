package main

import (
	"fmt"
	api "inis/app/api/route"
	dev "inis/app/dev/route"
	index "inis/app/index/route"
	inis "inis/app/inis/route"
	"inis/app/middleware"
	socket "inis/app/socket/route"
	"inis/app/timer"
	app "inis/config"
	_ "inis/docs"

	"github.com/fsnotify/fsnotify"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title inis API
// @version 1.0
// @description 基于Gin框架和Gorm二次封装的Go语言Web开发框架
// @termsOfService http://swagger.io/terms/

// @contact.name 不语
// @contact.url https://github.com/zhu885744/inisv1
// @contact.email 2776686748@qq.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 
// @BasePath /api/
// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {

	// 监听服务
	watch()
	// 运行服务
	run()

	// 静默运行 - 不显示控制台
	// go build -ldflags -H=windowsgui 或 bee pack -ba="-ldflags -H=windowsgui"
	// 压缩二进制包 - https://github.com/upx/upx/releases
	// upx -9 -o inis unti
}

func run() {
	// 允许跨域
	app.Gin.Use(middleware.Cors(), middleware.Install())

	// Swagger路由配置
	app.Gin.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 注册路由
	app.Use(api.Route, dev.Route, index.Route, inis.Route, socket.Route)
	// 运行服务
	app.Run(func() {
		timer.Run()
	})
}

// 监听配置文件变化
func watch() {

	app.AppToml.Viper.WatchConfig()
	// 配置文件变化时，重新初始化配置文件
	app.AppToml.Viper.OnConfigChange(func(event fsnotify.Event) {

		// 关闭服务
		if app.Server != nil {
			// 关闭服务
			err := app.Server.Shutdown(nil)
			if err != nil {
				fmt.Println("关闭服务发生错误: ", err)
				return
			}
		}

		watch()
		// 重新初始化驱动
		app.InitApp()
		// 重新运行服务
		run()
	})
}
