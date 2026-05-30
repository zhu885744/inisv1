package controller

import (
	"errors"
	"inis/app/facade"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type base struct{}

type ApiInterface interface {
	IGET(ctx *gin.Context)
	IPUT(ctx *gin.Context)
	IDEL(ctx *gin.Context)
	IPOST(ctx *gin.Context)
	INDEX(ctx *gin.Context)
}

// 运行时信息缓存
type runtimeInfoStruct struct {
	GOOS         string
	GOARCH       string
	GOROOT       string
	NumCPU       int
	NumGoroutine int
}

var runtimeInfo = getRuntimeInfo()

// getRuntimeInfo 获取运行时信息（缓存）
func getRuntimeInfo() runtimeInfoStruct {
	return runtimeInfoStruct{
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		GOROOT:       runtime.GOROOT(),
		NumCPU:       runtime.NumCPU(),
		NumGoroutine: runtime.NumGoroutine(),
	}
}

const (
	DefaultSuccessCode             = 200
	DefaultErrorCode               = 400
	DefaultUnauthorizedCode        = 401
	DefaultForbiddenCode           = 403
	DefaultNotFoundCode            = 404
	DefaultMethodNotAllowedCode    = 405
	DefaultInternalServerErrorCode = 500
)

// 通用响应码常量
var (
	successMessage     = "成功！"
	errorMessage       = "失败！"
	defaultResponseMsg = "好的！"
)

// json 统一响应
func (this base) json(ctx *gin.Context, data, msg, code any) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": cast.ToInt(code),
		"msg":  cast.ToString(msg),
		"data": data,
	})
}

// success 成功响应
func (this base) success(ctx *gin.Context, data any, msg ...string) {
	message := successMessage
	if len(msg) > 0 && !utils.Is.Empty(msg[0]) {
		message = facade.Lang(ctx, msg[0])
	}
	this.json(ctx, data, message, DefaultSuccessCode)
}

// error 错误响应
func (this base) error(ctx *gin.Context, msg any, code ...int) {
	httpCode := DefaultErrorCode
	if len(code) > 0 {
		httpCode = code[0]
	}
	this.json(ctx, nil, msg, httpCode)
}

// setToken 设置登录token到客户的cookie中
func (this base) setToken(ctx *gin.Context, token any) {
	host := ctx.Request.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}
	expire := cast.ToInt(facade.AppToml.Get("jwt.expire", "7200"))
	tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))
	ctx.SetCookie(tokenName, cast.ToString(token), expire, "/", host, false, false)
}

// Call 方法调用 - 资源路由本体
func (this base) call(allow map[string]any, name string, params ...any) (err error) {
	if empty := utils.Is.Empty(allow); empty {
		return errors.New("allow is empty")
	}
	if _, ok := allow[name]; !ok {
		return errors.New(name + " is not in allow")
	}
	method := reflect.ValueOf(allow[name])
	if len(params) != method.Type().NumIn() {
		return errors.New("输入参数的数量不匹配！")
	}
	in := make([]reflect.Value, len(params))
	for key, val := range params {
		in[key] = reflect.ValueOf(val)
	}
	method.Call(in)
	return nil
}

// handleHTTPMethod 处理HTTP方法的通用逻辑
func (this base) handleHTTPMethod(ctx *gin.Context, allow map[string]any) {
	method := strings.ToLower(ctx.Param("method"))
	if utils.Is.Empty(allow) {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：allow is empty"), DefaultMethodNotAllowedCode)
		return
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), DefaultMethodNotAllowedCode)
		return
	}
}

// 获取单个参数
func (this base) param(ctx *gin.Context, key string, def ...any) any {
	var value map[string]any
	if empty := utils.Is.Empty(def); !empty {
		value = map[string]any{key: def[0]}
	} else {
		value = map[string]any{key: nil}
	}
	params := this.params(ctx, value)
	return params[key]
}

// params 获取全部参数 , def map[string]any
func (this base) params(ctx *gin.Context, def ...map[string]any) (result map[string]any) {
	params, ok := ctx.Get("params")
	result = utils.Ternary(ok, cast.ToStringMap(params), make(map[string]any))
	if empty := utils.Is.Empty(def); !empty {
		for key, val := range def[0] {
			if _, ok := result[key]; !ok {
				result[key] = val
			}
		}
	}
	return
}

// getBool 获取布尔参数
func (this base) getBool(ctx *gin.Context, key string, def ...bool) bool {
	defaultValue := false
	if len(def) > 0 {
		defaultValue = def[0]
	}
	return cast.ToBool(this.param(ctx, key, defaultValue))
}

// getString 获取字符串参数
func (this base) getString(ctx *gin.Context, key string, def ...string) string {
	defaultValue := ""
	if len(def) > 0 {
		defaultValue = def[0]
	}
	return cast.ToString(this.param(ctx, key, defaultValue))
}

// getInt 获取整数参数
func (this base) getInt(ctx *gin.Context, key string, def ...int) int {
	defaultValue := 0
	if len(def) > 0 {
		defaultValue = def[0]
	}
	return cast.ToInt(this.param(ctx, key, defaultValue))
}

// getInt64 获取64位整数参数
func (this base) getInt64(ctx *gin.Context, key string, def ...int64) int64 {
	defaultValue := int64(0)
	if len(def) > 0 {
		defaultValue = def[0]
	}
	return cast.ToInt64(this.param(ctx, key, defaultValue))
}

// 获取单个请求头信息
func (this base) header(ctx *gin.Context, key string, def ...any) (result string) {
	result = ctx.GetHeader(key)
	if empty := utils.Is.Empty(result); empty {
		if !utils.Is.Empty(def) {
			result = def[0].(string)
		}
	}
	return
}

// 获取全部请求头信息
func (this base) headers(ctx *gin.Context) (result map[string]any) {
	result = make(map[string]any)
	for key, val := range ctx.Request.Header {
		result[key] = val[0]
	}
	return
}

// 获取 *gin.Context.Get() 中的值
func (this base) get(ctx *gin.Context, key any, def ...any) (value any) {
	if item, exist := ctx.Get(cast.ToString(key)); exist {
		value = item
	} else {
		if empty := utils.Is.Empty(def); !empty {
			value = def[0]
		}
	}
	return
}

// validateRequired 验证必填参数
func (this base) validateRequired(ctx *gin.Context, params map[string]any, fields ...string) bool {
	for _, field := range fields {
		if utils.Is.Empty(params[field]) {
			this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", field), DefaultErrorCode)
			return false
		}
	}
	return true
}

// validateRequiredParams 验证必填参数（直接从上下文获取）
func (this base) validateRequiredParams(ctx *gin.Context, fields ...string) (map[string]any, bool) {
	params := this.params(ctx)
	return params, this.validateRequired(ctx, params, fields...)
}

// getSystemInfo 获取系统信息
func (this base) getSystemInfo(ctx *gin.Context) map[string]any {
	return map[string]any{
		"GOOS":         runtimeInfo.GOOS,
		"GOARCH":       runtimeInfo.GOARCH,
		"GOROOT":       runtimeInfo.GOROOT,
		"NumCPU":       runtimeInfo.NumCPU,
		"NumGoroutine": runtimeInfo.NumGoroutine,
		"go":           utils.Version.Go(),
		"inis":         facade.Version,
		"agent":        this.header(ctx, "User-Agent"),
	}
}

// getCurrentTime 获取当前时间信息
func (this base) getCurrentTime() map[string]any {
	return map[string]any{
		"unix": time.Now().Unix(),
		"date": time.Now().Format("2006-01-02 15:04:05"),
	}
}
