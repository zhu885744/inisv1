package main

import (
	"context"
	"fmt"
	"time"
	api "inis/app/api/route"
	dev "inis/app/dev/route"
	index "inis/app/index/route"
	"inis/app/middleware"
	socket "inis/app/socket/route"
	"inis/app/timer"
	app "inis/config"

	"github.com/fsnotify/fsnotify"
)

// main - 主入口函数
func main() {
	watch()
	run()
}

// run - 运行服务
func run() {
	app.Gin.Use(middleware.Cors(), middleware.Install())
	app.Use(api.Route, dev.Route, index.Route, socket.Route)
	app.Run(func() {
		timer.Run()
	})
}

// watch - 监听配置文件变化
func watch() {
	app.AppToml.Viper.WatchConfig()
	app.AppToml.Viper.OnConfigChange(func(event fsnotify.Event) {
		shutdownServer()
		watch()
		app.InitApp()
		run()
	})
}

// shutdownServer - 关闭服务
func shutdownServer() {
	if app.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Server.Shutdown(ctx); err != nil {
			fmt.Println("关闭服务发生错误: ", err)
		}
	}
}
