package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	cacheUserPrefix  = "user[%v]"
	tokenNameKey     = "app.token_name"
	defaultTokenName = "INIS_LOGIN_TOKEN"
	// authErrorKey - Auth() 校验失败时暂存错误的上下文键。
	// Rule() 依据接口类型决定是否因此拒绝请求：
	// 公共接口（type=common）不受无效凭证影响，需登录的接口才会 401。
	authErrorKey = "auth_error"
)

var getTokenName = func() string {
	return cast.ToString(facade.AppToml.Get(tokenNameKey, defaultTokenName))
}

// getTokenFromHeaderOrCookie 从请求头或 Cookie 中获取登录凭证（服务端会话 id）
func getTokenFromHeaderOrCookie(ctx *gin.Context, tokenName string) string {
	if authHeader := ctx.Request.Header.Get("Authorization"); !utils.Is.Empty(authHeader) {
		return authHeader
	}
	token, _ := ctx.Cookie(tokenName)
	return token
}

// tokenFingerprint - 提取凭证指纹（长度 + 哈希前 6 字节）
//
// 会话 id 本身就是凭证，日志里不能出现明文片段，用哈希保证同一 token 的日志可关联但不泄露。
func tokenFingerprint(token string) map[string]any {
	sum := sha256.Sum256([]byte(token))
	return map[string]any{
		"token_length": len(token),
		"token_hash":   hex.EncodeToString(sum[:6]),
	}
}

// authErrorMessage 根据凭证错误类型生成用户提示
//
// 会话不存在 / 已过期 / 被踢下线 / 改密后失效都属于「需要重新登录」；
// 其它异常（用户被冻结、封禁、密码校验不通过）原样返回具体原因，便于用户自助排查。
func authErrorMessage(ctx *gin.Context, err error) string {
	if err == nil {
		return facade.Lang(ctx, "禁止非法操作！")
	}
	if errors.Is(err, facade.ErrSessionInvalid) {
		return facade.Lang(ctx, "登录已过期，请重新登录！")
	}

	return err.Error()
}

// logAuthFailure - 记录凭证校验失败的上下文（含载荷指纹），便于排查会话异常
func logAuthFailure(ctx *gin.Context, token string, err error) {
	facade.Log.Warn(map[string]any{
		"error":   err.Error(),
		"token":   tokenFingerprint(token),
		"ip":      ctx.ClientIP(),
		"path":    ctx.Request.URL.Path,
		"origin":  ctx.Request.Header.Get("Origin"),
		"referer": ctx.Request.Header.Get("Referer"),
	}, "登录凭证校验失败（非会话过期原因）")
}

// getUserInfoWithCache 获取用户信息（带缓存逻辑）
func getUserInfoWithCache(uid any, valid int64) (map[string]any, error) {
	cacheName := fmt.Sprintf(cacheUserPrefix, uid)
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName)), nil
	}

	user, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return nil, fmt.Errorf("用户不存在！")
	}

	if cacheState {
		go facade.Cache.Set(cacheName, user, time.Duration(valid)*time.Second)
	}

	return user, nil
}

// validateUserStatus 验证用户状态（冻结、封禁登录限制）
func validateUserStatus(user map[string]any) error {
	userStatus := cast.ToInt(user["status"])
	if userStatus == model.UserStatusFrozen {
		return fmt.Errorf("账号已被冻结，请联系管理员！")
	}

	// 检查封禁限制中是否包含登录限制
	restrictions := cast.ToInt(user["restrictions"])
	if restrictions&model.BanTypeLogin != 0 {
		currentBanId := cast.ToInt(user["current_ban_id"])
		if currentBanId > 0 {
			banRecord, _ := facade.DB.Model(&model.UserBanRecords{}).Find(currentBanId)
			if !utils.Is.Empty(banRecord) {
				banMap := cast.ToStringMap(banRecord)
				// 申诉中 / 申诉驳回期间封禁继续生效（只有真正解封才放行）
				if model.BanStatusRestricted(cast.ToInt(banMap["status"])) {
					status := cast.ToInt(banMap["status"])
					reason := cast.ToString(banMap["reason"])
					duration := cast.ToInt(banMap["duration"])
					expiresAt := cast.ToInt64(banMap["expires_at"])

					// 申诉相关状态单独说明，避免用户以为申诉后就自动解封了
					if status == model.BanStatusAppealed {
						return fmt.Errorf("您的账号仍处于封禁状态（申诉审核中）！原因：%s", reason)
					}
					if status == model.BanStatusAppealRejected {
						return fmt.Errorf("您的账号仍处于封禁状态（申诉未通过）！原因：%s", reason)
					}

					if duration > 0 {
						remainingDays := (expiresAt - time.Now().Unix()) / 86400
						if remainingDays > 0 {
							return fmt.Errorf("您的账号已被封禁！原因：%s，剩余 %d 天", reason, remainingDays)
						}
						return fmt.Errorf("您的账号已被封禁！原因：%s，将于今日解封", reason)
					}
					return fmt.Errorf("您的账号已被永久封禁！原因：%s", reason)
				}
			}
		}
	}

	return nil
}

// validatePasswordHash 验证密码哈希是否匹配
func validatePasswordHash(hash, userPassword any) error {
	if utils.Hash.Sum32(userPassword) != hash {
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

// authenticateToken - 校验登录凭证并取回用户
//
// 凭证是服务端会话 id（缓存里的 session:{sid}）。取到会话后统一做：
// 密码哈希校验（改密 / 重置密码后旧会话立即失效）与状态校验（冻结 / 封禁）。
func authenticateToken(token string) (map[string]any, error) {
	identity, err := facade.AuthenticateToken(token)
	if err != nil {
		return nil, err
	}

	user, err := getUserInfoWithCache(identity.Uid, identity.Valid)
	if err != nil {
		return nil, err
	}

	if err := validatePasswordHash(identity.Hash, user["password"]); err != nil {
		return nil, err
	}

	if err := validateUserStatus(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Auth - 登录凭证校验中间件（容错模式）
//
// 语义设计：本中间件只负责「身份识别」，不负责「访问控制」。
//   - 无 token           → 按匿名放行（是否允许由 Rule() 依据接口类型决定）
//   - token 有效         → 校验用户存在/状态后写入 ctx["user"]
//   - token 无效（过期 / 被踢下线 / 坏数据）→ 不中断请求，
//     仅记录日志并把错误暂存 ctx[authErrorKey]，交给 Rule() 决策：
//     公共接口（type=common）即使携带坏 token 也正常返回；
//     真正需要登录的接口才会 401，并顺带清除客户端无效 cookie。
//
// 注意：项目已完全移除 JWT，凭证是 Redis 里的服务端会话 id（见 facade/session.go），
// 旧 JWT token 不再被接受，升级后所有用户需要重新登录一次。
func Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := getTokenFromHeaderOrCookie(ctx, getTokenName())

		if utils.Is.Empty(token) {
			ctx.Next()
			return
		}

		user, err := authenticateToken(token)
		if err != nil {
			// 会话过期 / 被踢属于正常现象，不刷 warn 日志；其它异常保留排查上下文
			if !errors.Is(err, facade.ErrSessionInvalid) {
				logAuthFailure(ctx, token, err)
			}
			ctx.Set(authErrorKey, err)
			ctx.Next()
			return
		}

		ctx.Set("user", user)
		ctx.Next()
	}
}
