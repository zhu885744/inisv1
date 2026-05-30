package middleware

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	UserStatusNormal  = 0
	UserStatusFrozen = 1
)

const (
	cacheUserPrefix = "user[%v]"
	tokenNameKey    = "app.token_name"
	defaultTokenName = "INIS_LOGIN_TOKEN"
)

var getTokenName = func() string {
	return cast.ToString(facade.AppToml.Get(tokenNameKey, defaultTokenName))
}

// getTokenFromHeaderOrCookie 从请求头或 Cookie 中获取 token
func getTokenFromHeaderOrCookie(ctx *gin.Context, tokenName string) string {
	if authHeader := ctx.Request.Header.Get("Authorization"); !utils.Is.Empty(authHeader) {
		return authHeader
	}
	token, _ := ctx.Cookie(tokenName)
	return token
}

// handleJwtError 处理 JWT 解析错误
func handleJwtError(ctx *gin.Context, tokenName string, jwtResult facade.JwtResponse, defaultMsg string) {
	msg := utils.Ternary(
		jwtResult.Valid == 0,
		facade.Lang(ctx, "登录已过期，请重新登录！"),
		utils.Ternary(utils.Is.Empty(jwtResult.Error), defaultMsg, jwtResult.Error.Error()),
	)
	abortWithError(ctx, tokenName, 401, msg)
}

// getUserInfoWithCache 获取用户信息（带缓存逻辑）
func getUserInfoWithCache(uid any, jwtValid int64) (map[string]any, error) {
	cacheName := fmt.Sprintf(cacheUserPrefix, uid)
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName)), nil
	}

	user := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return nil, fmt.Errorf("用户不存在！")
	}

	if cacheState {
		go facade.Cache.Set(cacheName, user, time.Duration(jwtValid)*time.Second)
	}

	return user, nil
}

// validateUserStatus 验证用户状态
func validateUserStatus(user map[string]any) error {
	userStatus := cast.ToInt(user["status"])
	if userStatus == UserStatusFrozen {
		return fmt.Errorf("账号已被冻结，请联系管理员！")
	}
	return nil
}

// validatePasswordHash 验证密码哈希是否匹配
func validatePasswordHash(jwtHash, userPassword any) error {
	if utils.Hash.Sum32(userPassword) != jwtHash {
		return fmt.Errorf("登录已过期，请重新登录！")
	}
	return nil
}

// abortWithError 统一错误响应处理
func abortWithError(ctx *gin.Context, tokenName string, code int, msg string) {
	ctx.SetCookie(tokenName, "", -1, "/", "", false, false)
	ctx.JSON(200, gin.H{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
	ctx.Abort()
}

// Jwt - JWT 中间件
func Jwt() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenName := getTokenName()
		token := getTokenFromHeaderOrCookie(ctx, tokenName)

		if utils.Is.Empty(token) {
			ctx.Next()
			return
		}

		jwtResult := facade.Jwt().Parse(token)
		if jwtResult.Error != nil {
			handleJwtError(ctx, tokenName, jwtResult, "禁止非法操作！")
			return
		}

		user, err := getUserInfoWithCache(jwtResult.Data["uid"], jwtResult.Valid)
		if err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		if err := validateUserStatus(user); err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		if err := validatePasswordHash(jwtResult.Data["hash"], user["password"]); err != nil {
			abortWithError(ctx, tokenName, 401, err.Error())
			return
		}

		ctx.Set("user", user)
		ctx.Next()
	}
}
