package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

// Install常量
const (
	installLockFile     = "install.lock"
	installCookieName   = "install"
	installCookieExpire = 3
	installRedirectURL  = "/install.html"
	devInstallPath      = "/dev/install"
	apiPathPrefix       = "/api/"
)

// Install - 安装引导中间件
func Install() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		method := strings.ToUpper(ctx.Request.Method)

		if utils.File().Exist(installLockFile) {
			if (path == "/" || path == "/install") && method == "GET" {
				if ok, _ := ctx.Cookie(installCookieName); !utils.Is.Empty(ok) {
					ctx.Next()
					return
				}

				ctx.SetCookie(installCookieName, "1", installCookieExpire, "/", "", false, true)
				ctx.Redirect(301, installRedirectURL)
				ctx.Abort()
				return
			}

			if strings.HasPrefix(path, apiPathPrefix) {
				ctx.JSON(200, map[string]any{"code": 412, "msg": "安装引导未完成，禁止访问！", "data": nil})
				ctx.Abort()
				return
			}
		} else {
			if strings.HasPrefix(path, devInstallPath) {
				ctx.JSON(200, map[string]any{"code": 412, "msg": "程序已完成安装，禁止访问！", "data": nil})
				ctx.Abort()
				return
			}

			if path == installRedirectURL {
				ctx.Redirect(301, "/")
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}
