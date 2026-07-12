package middleware

import (
	"inis/app/facade"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func FileLimit(maxSize int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maxSize)
		ctx.Next()
	}
}

func FileSizeLimit() gin.HandlerFunc {
	defaultMaxSize := int64(50 * 1024 * 1024)
	maxSize := cast.ToInt64(facade.AppToml.Get("attachment.max_size"))
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return FileLimit(maxSize)
}