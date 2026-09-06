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
		if isCommonRoute(ruleType) {
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
		if user.Id == 0 {
			ctx.JSON(200, gin.H{"code": 401, "msg": facade.Lang(ctx, "请先登录！"), "data": nil})
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
