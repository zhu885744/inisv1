package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
	"time"
)

// 定义用户状态常量，提高代码可读性和可维护性
const (
	UserStatusNormal = 0 // 正常状态
	UserStatusFrozen = 1 // 冻结状态
)

// Jwt - JWT 中间件
func Jwt() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))
		
		// 统一获取 token 的逻辑
		token := getTokenFromHeaderOrCookie(ctx, tokenName)
		
		// 无 token 直接放行（由后续中间件或接口处理权限）
		if utils.Is.Empty(token) {
			ctx.Next()
			return
		}

		// 解析 JWT
		jwtResult := facade.Jwt().Parse(token)
		if jwtResult.Error != nil {
			handleJwtError(ctx, tokenName, jwtResult, "禁止非法操作！")
			return
		}

		// 获取用户信息（支持缓存）
		user, err := getUserInfo(jwtResult.Data["uid"], jwtResult.Valid)
		if err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		// 验证用户状态
		if err := validateUserStatus(user); err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		// 验证密码哈希
		if err := validatePasswordHash(jwtResult.Data["hash"], user["password"]); err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		// 将用户信息存入上下文
		ctx.Set("user", user)
		ctx.Next()
	}
}

// 从请求头或 Cookie 中获取 token
func getTokenFromHeaderOrCookie(ctx *gin.Context, tokenName string) string {
	if authHeader := ctx.Request.Header.Get("Authorization"); !utils.Is.Empty(authHeader) {
		return authHeader
	}
	token, _ := ctx.Cookie(tokenName)
	return token
}

// 处理 JWT 解析错误
func handleJwtError(ctx *gin.Context, tokenName string, jwtResult facade.JwtResponse, defaultMsg string) {
	msg := utils.Ternary(
		jwtResult.Valid == 0,
		facade.Lang(ctx, "登录已过期，请重新登录！"),
		utils.Ternary(utils.Is.Empty(jwtResult.Error), defaultMsg, jwtResult.Error.Error()),
	)
	abortWithError(ctx, tokenName, 401, msg)
}

// 获取用户信息（带缓存逻辑）
func getUserInfo(uid any, jwtValid int64) (map[string]any, error) {
	cacheName := fmt.Sprintf("user[%v]", uid)
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	// 优先从缓存获取
	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName)), nil
	}

	// 缓存未命中，从数据库获取
	user := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return nil, fmt.Errorf("用户不存在！")
	}

	// 写入缓存（异步）
	if cacheState {
		go facade.Cache.Set(cacheName, user, time.Duration(jwtValid)*time.Second)
	}

	return user, nil
}

// 验证用户状态
func validateUserStatus(user map[string]any) error {
	userStatus := cast.ToInt(user["status"])
	if userStatus == UserStatusFrozen {
		return fmt.Errorf("账号已被冻结，请联系管理员！")
	}
	return nil
}

// 验证密码哈希是否匹配
func validatePasswordHash(jwtHash, userPassword any) error {
	if utils.Hash.Sum32(userPassword) != jwtHash {
		return fmt.Errorf("登录已过期，请重新登录！")
	}
	return nil
}

// 统一错误响应处理
func abortWithError(ctx *gin.Context, tokenName string, code int, msg string) {
	// 清除无效 token
	ctx.SetCookie(tokenName, "", -1, "/", "", false, false)
	// 返回错误响应
	ctx.JSON(200, gin.H{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
	ctx.Abort()
}