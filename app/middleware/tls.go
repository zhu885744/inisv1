package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unrolled/secure"
	"github.com/unti-io/go-utils/utils"
	"strings"
)

// Tls - HTTPS处理中间件
func Tls() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		port := func() string {
			item := utils.Env().Get("app.port", ":8642")
			result := cast.ToString(item)

			if !strings.Contains(result, ":") {
				result = ":" + result
			}
			return result
		}

		secureMiddleware := secure.New(secure.Options{
			SSLRedirect: true,
			SSLHost:     port(),
		})
		err := secureMiddleware.Process(ctx.Writer, ctx.Request)

		if err != nil {
			return
		}

		ctx.Next()
	}
}
