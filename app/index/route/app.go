package route

import (
	"github.com/gin-gonic/gin"
	"github.com/jasonlvhit/gocron"
	"github.com/radovskyb/watcher"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/index/controller"
	"runtime/debug"
	"sync"
	"time"
)

var (
	isWatching bool
	watchLock  sync.Mutex
	// 监听 themes 和 admin 目录的 index.html
	watchTargets = []string{
		"public/themes/index.html",
		"public/admin/index.html",
	}
)

func Route(Gin *gin.Engine) {
	// 拦截异常
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{
				"error":     err,
				"stack":     string(debug.Stack()),
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line,
			}, "路由初始化发生错误")
		}
	}()

	// 配置静态资源路由
	Gin.Static("/themes", "public/themes")
	Gin.Static("/admin", "public/admin")

	// 启动监听（加锁防止重复）
	watchLock.Lock()
	if !isWatching {
		go watch(Gin)
	}
	watchLock.Unlock()

	// 注册首页路由
	Gin.GET("/", controller.Index)
}

// watch - 监听 themes 和 admin 目录的 index.html 文件变化
func watch(Gin *gin.Engine) {
	watchLock.Lock()
	isWatching = true
	watchLock.Unlock()

	item := watcher.New()
	defer item.Close()

	// 添加上所有监听目标
	for _, target := range watchTargets {
		if err := item.Add(target); err != nil {
			facade.Log.Error(map[string]any{
				"error":  err,
				"target": target,
				"stack":  string(debug.Stack()),
			}, "添加监听文件失败")
		}
	}

	// 定时器：文件不存在时重试监听（修正 gocron 赋值）
	timer := func() {
		// 旧版本 gocron.Do() 仅返回 error，无需接收两个值
		err := gocron.Every(1).Seconds().Do(checkFiles(Gin))
		if err != nil {
			facade.Log.Error(map[string]any{"error": err}, "启动监听重试定时器失败")
		}
	}

	// 处理文件变化事件
	go func() {
		for {
			select {
			case event := <-item.Event:
				if event.Op == watcher.Write {
					facade.Log.Info(map[string]any{
						"path": event.Path,
						"op":   event.Op.String(),
					}, "文件发生变化")

					// 若使用 Gin 模板引擎，可重新加载对应模板；静态文件则省略
					switch event.Path {
					case "public/themes/index.html":
						Gin.LoadHTMLGlob("public/themes/index.html")
					case "public/admin/index.html":
						Gin.LoadHTMLGlob("public/admin/index.html")
					}
				}
			case <-item.Error:
				// 修正：Warn 方法传入 map[string]any + 字符串描述
				facade.Log.Warn(map[string]any{}, "监听发生错误，启动重试定时器")
				timer()
			case <-item.Closed:
				// 修正：Warn 方法传入 map[string]any + 字符串描述
				facade.Log.Warn(map[string]any{}, "监听被关闭，启动重试定时器")
				timer()
			}
		}
	}()

	// 启动监听
	if err := item.Start(time.Second); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "启动监听失败，启动重试定时器")
		timer()
	}
}

// checkFiles - 检查监听文件是否存在，并重试监听
func checkFiles(Gin *gin.Engine) func() {
	return func() {
		watchLock.Lock()
		defer watchLock.Unlock()

		// 检查是否有至少一个监听文件存在
		hasExistFile := false
		for _, target := range watchTargets {
			if utils.File().Exist(target) {
				hasExistFile = true
				break
			}
		}

		if hasExistFile && !isWatching {
			go watch(Gin)
			// 停止定时器（旧版本 gocron 直接 Remove 任务）
			gocron.Remove(checkFiles(Gin))
		} else if !hasExistFile && isWatching {
			isWatching = false
		}
	}
}