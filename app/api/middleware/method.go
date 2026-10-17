package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
)

var requireAuthMethods = []any{"POST", "PUT", "DELETE", "PATCH"}

// requiresAuthentication 判断请求方法是否需要认证
func requiresAuthentication(method string) bool {
	return utils.In.Array(method, requireAuthMethods)
}

// setContextUser 设置用户到上下文
func setContextUser(ctx *gin.Context, user map[string]any) {
	ctx.Set("user", user)
}

// Method - 请求类型校验中间件
func Method() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !requiresAuthentication(ctx.Request.Method) {
			ctx.Next()
			return
		}

		tokenName := getTokenName()
		token := getTokenFromHeaderOrCookie(ctx, tokenName)

		result := gin.H{"code": 401, "msg": facade.Lang(ctx, "禁止非法操作！"), "data": nil}

		if utils.Is.Empty(token) {
			ctx.JSON(200, result)
			ctx.Abort()
			return
		}

		user, err := authenticateToken(token)
		if err != nil {
			if !errors.Is(err, facade.ErrSessionInvalid) {
				logAuthFailure(ctx, token, err)
			}
			abortWithError(ctx, tokenName, 401, authErrorMessage(ctx, err))
			return
		}

		setContextUser(ctx, user)
		ctx.Next()
	}
}
