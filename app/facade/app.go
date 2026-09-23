package facade

import (
	"github.com/fsnotify/fsnotify"
	"github.com/unti-io/go-utils/utils"
)

type H map[string]any

const (
	// ConfigPath - 配置目录
	ConfigPath = "config"
	// ModeToml - 配置文件格式
	ModeToml = "toml"
	// ConfigNameApp - 应用配置文件名
	ConfigNameApp = "app"
)   
   
// AppToml - APP配置文件
var AppToml *utils.ViperResponse

// initAppToml - 初始化App配置文件
func initAppToml() {
	item := utils.Viper(utils.ViperModel{
		Path:    ConfigPath,
		Mode:    ModeToml,
		Name:    ConfigNameApp,
		Content: utils.Replace(TempApp, nil),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line}, 
			"App配置初始化错误")
	}

	AppToml = &item
}

// init - 初始化函数
func init() {
	// 本文件是包内最早执行的 init（文件名字典序），而 log.go 的初始化排在其后；
	// 这里在配置异常时会输出日志，故先确保日志组件就绪
	ensureLogReady()

	initAppToml()
	initApp()

	WatchConfigChange(AppToml, initApp)
}

// initApp - 初始化应用
func initApp() {

}

// WatchConfigChange - 监听配置文件变化（导出函数，供包内其他文件使用）
func WatchConfigChange(config *utils.ViperResponse, callback func()) {
	if config == nil || config.Viper == nil {
		return
	}
	config.Viper.WatchConfig()
	config.Viper.OnConfigChange(func(event fsnotify.Event) {
		callback()
	})
}