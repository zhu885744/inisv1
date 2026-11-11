package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"inis/app/facade"
	"net"
	"net/http"
	"os"
	debugs "runtime/debug"
	"strings"
	"time"
)

// requestLog 请求日志快照
//
// gin 的 *gin.Context 是池化复用的：handler 返回后对象会被归还、下一个请求接着用。
// 因此异步写日志前必须把需要的值全部取出来，绝不能把 ctx 交给 goroutine ——
// 否则异步读 ctx 里的 params map 会和下一个请求并发读写同一个 map（直接 fatal）。
type requestLog struct {
	params    map[string]any
	path      string
	method    string
	ip        string
	userAgent string
	errors    string
}

// GinLogger - 接收gin框架默认的日志
func GinLogger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 在进入业务处理前取时间，日志里的 cost 才是真实请求耗时
		start := time.Now()
		ctx.Next()

		// 快照必须在当前 goroutine 内完成（见 requestLog 注释）
		snapshot := requestLog{
			path:      ctx.Request.URL.Path,
			method:    ctx.Request.Method,
			ip:        ctx.ClientIP(),
			userAgent: ctx.Request.UserAgent(),
			errors:    ctx.Errors.ByType(gin.ErrorTypePrivate).String(),
		}
		if params, ok := ctx.Get("params"); ok {
			snapshot.params = cast.ToStringMap(params)
		}

		go logRequest(snapshot, start)
	}
}

// GinRecovery - recover掉项目可能出现的panic，并使用zap记录相关日志
func GinRecovery(debug ...bool) gin.HandlerFunc {
	if len(debug) == 0 {
		debug = append(debug, false)
	}
	if !debug[0] {
		return func(ctx *gin.Context) {
			ctx.Next()
		}
	}
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var broken bool
				if network, ok := err.(*net.OpError); ok {
					if system, ok := network.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(system.Error()), "broken pipe") || strings.Contains(strings.ToLower(system.Error()), "connection reset by peer") {
							broken = true
						}
					}
				}

				// 请求导出同样要脱敏：Authorization（登录会话 id）/ Cookie 不允许进日志
				request := dumpRequestSanitized(ctx)
				if broken {
					facade.Log.Error(map[string]any{
						"path":    ctx.Request.URL.Path,
						"request": request,
					}, "middleware")
					ctx.Error(err.(error))
					ctx.Abort()
					return
				}

				facade.Log.Error(map[string]any{
					"path":    ctx.Request.URL.Path,
					"error":   err,
					"stack":   string(debugs.Stack()),
					"request": request,
				}, "[Recovery from panic]")

				var stack []string
				for i, item := range strings.Split(string(debugs.Stack()), "\n") {
					if i > 0 && len(item) > 0 {
						stack = append(stack, strings.TrimSpace(item))
					}
				}

				if ctx.Request.URL.Path == "/" {
					ctx.Next()
					return
				}

				ctx.JSON(http.StatusInternalServerError, gin.H{
					"code": http.StatusInternalServerError,
					"msg":  err.(error).Error(),
					"data": stack,
				})
				ctx.Abort()
				return
			}
		}()
		ctx.Next()
	}
}

// logRequest - 记录请求日志（start 由中间件在 ctx.Next() 之前取得）
func logRequest(snapshot requestLog, start time.Time) {

	path := snapshot.path
	method := snapshot.method

	if strings.Contains(path, ".") || path == "/" {
		return
	}

	if cast.ToBool(facade.LogToml.Get("on", true)) {
		facade.Log.Info(map[string]any{
			"path":   path,
			"method": method,
			// 脱敏后落盘：密码 / 密钥 / 验证码整值打码，手机号、邮箱、账号做掩码。
			// 日志文件曾被明文记录「账号 + 密码」，属于可直接撞库的泄露，详见 redact.go
			"params":     SanitizeLogParams(snapshot.params),
			"ip":         snapshot.ip,
			"user-agent": snapshot.userAgent,
			"errors":     snapshot.errors,
			"cost":       time.Since(start).String(),
		}, "middleware")
	}
}
