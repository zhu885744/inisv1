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
	result := facade.DB.Model(&table).Where([]any{
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

		if isCommonRoute(cast.ToString(rule["type"])) {
			ctx.Next()
			return
		}

		if isLoginRoute(cast.ToString(rule["type"])) {
			if user.Id == 0 {
				ctx.JSON(200, gin.H{"code": 401, "msg": facade.Lang(ctx, "请先登录！"), "data": nil})
				ctx.Abort()
				return
			}
			ctx.Next()
			return
		}

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
