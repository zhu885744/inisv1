package middleware

import (
	"fmt"
	"inis/app/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// routeRestrictionMap 路由 -> 限制类型映射
var routeRestrictionMap = map[string]int{
	// 发表内容
	"POST /api/article/create":   model.BanTypeContent,
	"POST /api/article/save":     model.BanTypeContent,
	"POST /api/moments/create":   model.BanTypeContent,
	"POST /api/moments/save":     model.BanTypeContent,
	"PUT /api/article/update":    model.BanTypeContent,
	"PUT /api/moments/update":    model.BanTypeContent,
	// 评论
	"POST /api/comment/create": model.BanTypeComment,
	"POST /api/comment/save":   model.BanTypeComment,
	// 上传
	"POST /api/attachment/upload": model.BanTypeUpload,
	// 互动（点赞、收藏、关注）
	"POST /api/user-likes/like":      model.BanTypeInteraction,
	"POST /api/user-collects/collect": model.BanTypeInteraction,
	"POST /api/user-follows/follow":  model.BanTypeInteraction,
	"PUT /api/user-likes/like":       model.BanTypeInteraction,
	"PUT /api/user-collects/collect":  model.BanTypeInteraction,
	"PUT /api/user-follows/follow":   model.BanTypeInteraction,
}

// getRestrictionForRoute 根据当前路由获取所需的限制类型
func getRestrictionForRoute(ctx *gin.Context) (restrictionBit int, hasRestriction bool) {
	method := strings.ToUpper(ctx.Request.Method)
	path := ctx.Request.URL.Path

	// 精确匹配
	key := fmt.Sprintf("%s %s", method, path)
	if bit, ok := routeRestrictionMap[key]; ok {
		return bit, true
	}

	// 模糊匹配（PUT /api/user-likes/:method 等）
	for routePattern, bit := range routeRestrictionMap {
		parts := strings.SplitN(routePattern, " ", 2)
		if len(parts) != 2 {
			continue
		}
		routeMethod := parts[0]
		routePath := parts[1]

		if method != routeMethod {
			continue
		}

		// 路径前缀匹配（如 PUT /api/article/ 匹配 PUT /api/article/update）
		if strings.HasPrefix(path, routePath+"/") || path == routePath {
			return bit, true
		}

		// 路径前缀匹配（更宽松的匹配）
		routePrefix := strings.TrimSuffix(routePath, "/create")
		routePrefix = strings.TrimSuffix(routePrefix, "/save")
		routePrefix = strings.TrimSuffix(routePrefix, "/update")
		if strings.HasPrefix(path, routePrefix+"/") || path == routePrefix+"/create" || path == routePrefix+"/save" || path == routePrefix+"/update" {
			return bit, true
		}
	}

	return 0, false
}

// getUserFromCtx 从上下文中获取用户信息 map
func getUserFromCtx(ctx *gin.Context) map[string]any {
	if user, ok := ctx.Get("user"); ok {
		return cast.ToStringMap(user)
	}
	return nil
}

// Restriction 细粒度权限限制中间件
// 检查用户是否因封禁被限制了特定操作
func Restriction() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		user := getUserFromCtx(ctx)
		if utils.Is.Empty(user) {
			ctx.Next()
			return
		}

		restrictionBit, hasRestriction := getRestrictionForRoute(ctx)
		if !hasRestriction {
			ctx.Next()
			return
		}

		// 检查用户限制位掩码
		userRestrictions := cast.ToInt(user["restrictions"])
		if userRestrictions&restrictionBit != 0 {
			// 确认封禁记录仍处于生效状态
			currentBanId := cast.ToInt(user["current_ban_id"])
			if currentBanId > 0 {
				// 快速检查：如果 restrictions 不为 0 且有 ban_id，认为限制有效
				// 避免每次请求都查数据库
				ctx.JSON(200, gin.H{
					"code": 403,
					"msg":  "您当前的账号操作受到限制，如有疑问请提交申诉！",
					"data": nil,
				})
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}

// CheckUserRestriction 供Controller内直接调用的限制检查函数
// 返回 true 表示受限
func CheckUserRestriction(ctx *gin.Context, restrictionBit int) bool {
	user := getUserFromCtx(ctx)
	if utils.Is.Empty(user) {
		return false
	}

	userRestrictions := cast.ToInt(user["restrictions"])
	if userRestrictions&restrictionBit != 0 {
		currentBanId := cast.ToInt(user["current_ban_id"])
		if currentBanId > 0 {
			return true
		}
	}
	return false
}
