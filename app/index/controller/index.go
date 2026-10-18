package controller

import (
	"inis/app/facade"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 首页模板缓存
//
// 旧实现每个请求都 os.ReadFile("public/index.html") 再 string() 复制一次：
// 刷新一次页面就是一次磁盘读 + 一次整文件分配，而文件只在部署时才会变。
// 这里改成「mtime 变了才重读」，既保留热更新能力，又不再每请求读盘。
var (
	indexHTMLMu   sync.RWMutex
	indexHTMLData []byte
	indexHTMLMod  time.Time
)

// loadIndexHTML 读取首页模板（带 mtime 校验的进程内缓存）
func loadIndexHTML() ([]byte, error) {
	info, err := os.Stat("public/index.html")
	if err != nil {
		return nil, err
	}

	indexHTMLMu.RLock()
	cached, mod := indexHTMLData, indexHTMLMod
	indexHTMLMu.RUnlock()

	if cached != nil && mod.Equal(info.ModTime()) {
		return cached, nil
	}

	content, err := os.ReadFile("public/index.html")
	if err != nil {
		return nil, err
	}

	indexHTMLMu.Lock()
	indexHTMLData = content
	indexHTMLMod = info.ModTime()
	indexHTMLMu.Unlock()

	return content, nil
}

// Index - 首页控制器
func Index(ctx *gin.Context) {
	content, err := loadIndexHTML()
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "读取index.html文件失败")
		ctx.String(200, "页面模板（public/index.html）读取失败，请联系管理员进行处理！")
		return
	}

	// 直接写字节，不再 string(content) 复制一份
	ctx.Data(200, "text/html; charset=utf-8", content)
}
