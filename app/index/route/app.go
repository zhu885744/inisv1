package route

import (
	"github.com/gin-gonic/gin"
	"github.com/radovskyb/watcher"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/index/controller"
	"runtime/debug"
	"sync"
	"time"
)

// 全局唯一的监听控制器，管理所有监听资源
type TemplateWatcher struct {
	ginEngine  *gin.Engine
	watcher    *watcher.Watcher
	ticker     *time.Ticker
	isRunning  bool
	mu         sync.RWMutex
}

var globalWatcher *TemplateWatcher

// 初始化监听（全局仅初始化一次）
func initWatcher(Gin *gin.Engine) {
	if globalWatcher == nil {
		globalWatcher = &TemplateWatcher{
			ginEngine: Gin,
			watcher:   watcher.New(),
			ticker:    time.NewTicker(1 * time.Second), // 原生定时器，替代gocron
		}
	}
}

func Route(Gin *gin.Engine) {
	// 拦截全局panic
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{
				"error":     err,
				"stack":     string(debug.Stack()),
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line,
			}, "首页模板渲染发生错误")
		}
	}()

	// 初始化模板监听
	initWatcher(Gin)
	// 初始加载模板
	loadTemplateIfExist(Gin)
	// 启动监听逻辑（非阻塞）
	go globalWatcher.start()

	// 模板状态接口（供前端轮询）
	Gin.GET("/api/template-status", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"ok":     utils.File().Exist("public/index.html"),
			"msg":    "模板状态实时检测",
			"reload": utils.File().Exist("public/index.html"),
		})
	})

	// 首页路由：无侵入式兜底
    Gin.GET("/", func(ctx *gin.Context) {
	    templatePath := "public/index.html"
	    if !utils.File().Exist(templatePath) {
	    	ctx.String(200, `页面模板（public/index.html）不存在，请联系管理员进行处理！`)
	    	return
	    }
    	// 模板存在，执行原有控制器逻辑
	    controller.Index(ctx)
    })
}

// 仅当文件存在时加载模板
func loadTemplateIfExist(Gin *gin.Engine) {
	templatePath := "public/index.html"
	if !utils.File().Exist(templatePath) {
		facade.Log.Warn(map[string]any{"file": templatePath}, "模板文件不存在，暂不加载")
		return
	}
	// 加载模板
	Gin.LoadHTMLFiles(templatePath)
	facade.Log.Info(map[string]any{"file": templatePath}, "模板文件加载成功")
}

// 启动模板监听（核心逻辑）
func (tw *TemplateWatcher) start() {
	tw.mu.Lock()
	if tw.isRunning {
		tw.mu.Unlock()
		return // 避免重复启动
	}
	tw.isRunning = true
	tw.mu.Unlock()

	templatePath := "public/index.html"
	defer func() {
		// 退出时清理资源
		tw.mu.Lock()
		tw.isRunning = false
		tw.mu.Unlock()
		tw.watcher.Close()
		tw.ticker.Stop()
		facade.Log.Info(map[string]any{}, "模板监听器已停止")
	}()

	// 尝试添加文件监听
	if utils.File().Exist(templatePath) {
		if err := tw.watcher.Add(templatePath); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "添加模板文件监听失败")
			// 监听失败，启动定时器兜底检查
			tw.runTickerCheck()
			return
		}
		facade.Log.Info(map[string]any{"file": templatePath}, "模板文件监听已启动")
	} else {
		// 文件不存在，直接启动定时器检查
		tw.runTickerCheck()
		return
	}

	// 处理监听器事件
	for {
		select {
		case event, ok := <-tw.watcher.Event:
			if !ok {
				return // 通道关闭，退出
			}
			// 文件修改/创建时重新加载模板
			if event.Op == watcher.Write || event.Op == watcher.Create {
				loadTemplateIfExist(tw.ginEngine)
			}
			// 文件删除时，启动定时器检查
			if event.Op == watcher.Remove {
				facade.Log.Warn(map[string]any{"file": templatePath}, "模板文件被删除，启动定时检查")
				tw.runTickerCheck()
				return
			}
		case err, ok := <-tw.watcher.Error:
			if !ok {
				return
			}
			// 区分错误类型，友好日志
			if err.Error() == "error: watched file or folder deleted" {
				facade.Log.Warn(map[string]any{"error": err}, "模板文件被删除，启动定时检查")
			} else {
				facade.Log.Error(map[string]any{"error": err}, "模板监听器非预期错误")
			}
			tw.runTickerCheck()
			return
		case <-tw.watcher.Closed:
			facade.Log.Warn(map[string]any{}, "模板监听器被关闭，启动定时检查")
			tw.runTickerCheck()
			return
		}
	}
}

// 定时检查文件是否存在，恢复后重启监听
func (tw *TemplateWatcher) runTickerCheck() {
	facade.Log.Info(map[string]any{}, "启动模板文件定时检查（每秒）")
	for range tw.ticker.C {
		templatePath := "public/index.html"
		if utils.File().Exist(templatePath) {
			// 文件恢复，重启监听
			facade.Log.Info(map[string]any{"file": templatePath}, "检测到模板文件恢复，重启监听器")
			loadTemplateIfExist(tw.ginEngine)
			go tw.start() // 重启监听
			return        // 退出定时器
		}
	}
}