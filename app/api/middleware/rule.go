package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"sync"
)

const cacheRulePrefix = "rule[%v][%v]"

// getUserFromContext 从上下文获取用户信息
func getUserFromContext(ctx *gin.Context) model.Users {
	var table model.Users
	keys := utils.Struct.Keys(&table)

	if user, ok := ctx.Get("user"); ok {
		for key, val := range cast.ToStringMap(user) {
			if utils.InArray[string](key, keys) && !utils.Is.Empty(val) {
				utils.Struct.Set(&table, key, val)
			}
		}
	}
	return table
}

// getRuleFromCache 从缓存或数据库获取规则
func getRuleFromCache(ctx *gin.Context) map[string]any {
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))
	path := strings.ReplaceAll(ctx.Request.URL.Path, "/", ".")
	cacheName := fmt.Sprintf(cacheRulePrefix, strings.ToUpper(ctx.Request.Method), path)

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	var table model.AuthRules
	result, _ := facade.DB.Model(&table).Where([]any{
		[]any{"route", "=", ctx.Request.URL.Path},
		[]any{"method", "=", strings.ToUpper(ctx.Request.Method)},
	}).Find()

	if !utils.Is.Empty(result) && cacheState {
		go facade.Cache.Set(cacheName, result, 0)
	}

	return result
}

// isCommonRoute 判断是否为公共路由
func isCommonRoute(ruleType string) bool {
	return ruleType == "common"
}

// publicRoutes - 无需登录即可访问的接口（与 createAuthRules 里标记 type=common 的公开接口一致）
//
// 为什么还要硬编码一份：规则是从 auth_rules 表查的，而刚装完（或规则尚未落库/缓存为空）时
// 查不到这一行，中间件就会按「需要登录」处理 —— 结果是**登录接口自己要先登录**，
// 带着旧 token 的用户永远登不进来（表现为登录接口返回 401、点「登录」毫无反应）。
// 这里对公开接口兜一层，保证鉴权链路本身可用；权限收敛仍以 auth_rules 为准。
var publicRoutes = map[string]bool{
	"POST /api/comm/login":            true,
	"POST /api/comm/sign-code":        true, // 验证码登录（邮箱/手机号 + 验证码）
	"POST /api/comm/register":         true,
	"POST /api/comm/check-token":      true,
	"POST /api/comm/reset-password":   true,
	"POST /api/comm/logout":           true,
	"DELETE /api/comm/logout":         true,
}

// isPublicRoute 判断是否为公开接口（与规则表无关的兜底）
func isPublicRoute(ctx *gin.Context) bool {
	key := strings.ToUpper(ctx.Request.Method) + " " + ctx.Request.URL.Path
	return publicRoutes[key]
}

// isLoginRoute 判断是否为登录路由
func isLoginRoute(ruleType string) bool {
	return ruleType == "login"
}

// Rule - 规则校验中间件
func Rule() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		async := sync.WaitGroup{}
		async.Add(2)

		var user model.Users
		go func(async *sync.WaitGroup) {
			defer async.Done()
			user = getUserFromContext(ctx)
		}(&async)

		var rule map[string]any
		go func(async *sync.WaitGroup) {
			defer async.Done()
			rule = getRuleFromCache(ctx)
			ctx.Set("route", rule)
		}(&async)

		async.Wait()

		ruleType := cast.ToString(rule["type"])

		// 公共接口（type=common）：无论是否携带 token、token 是否有效，一律放行。
		// 这是"匿名请求携带旧/跨实例 cookie 时不被 401 拖累"的关键。
		// isPublicRoute 是针对「规则还没落库」窗口期的兜底（见其注释）。
		if isCommonRoute(ruleType) || isPublicRoute(ctx) {
			ctx.Next()
			return
		}

		// 以下分支均要求有效登录身份：

		// 1) token 解析失败/用户失效（Jwt() 暂存的错误）→ 401，
		//    同时通过 abortWithError 清除客户端无效 cookie，实现"自愈"：
		//    下一次请求即为匿名，可重新登录获取有效 token。
		if jwtErr, ok := ctx.Get(jwtErrorKey); ok {
			err, _ := jwtErr.(error)
			abortWithError(ctx, getTokenName(), 401, jwtErrorMessage(ctx, err))
			return
		}

		// 2) 未登录（无 token）→ 401
		//
		// auth: "guest" 用于把「没登录」和「登录态异常」区分开：
		// 两者都是 401，但前端只应对后者弹「登录状态异常，需要自行清除 cookie」，
		// 否则刚装完还没登录、或旧 token 指向的用户已不存在时，会凭空弹一个吓人的框。
		// 见 Mellow/src/api/request.js 的 isGuestAuth。
		if user.Id == 0 {
			ctx.JSON(200, gin.H{"code": 401, "msg": facade.Lang(ctx, "请先登录！"), "data": nil, "auth": "guest"})
			ctx.Abort()
			return
		}

		// 3) 仅要求登录的接口（type=login）——已登录即放行
		if isLoginRoute(ruleType) {
			ctx.Next()
			return
		}

		// 4) 默认（type=default / 未标注）：需具备对应权限点
		rules := (&model.Users{}).Rules(user.Id)
		name := fmt.Sprintf("[%v][%v]", strings.ToUpper(ctx.Request.Method), ctx.Request.URL.Path)

		if !utils.InArray[any](name, rules) {
			ctx.JSON(200, gin.H{"code": 403, "msg": facade.Lang(ctx, "无权限！"), "data": nil})
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
