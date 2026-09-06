package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"time"

	JWTLIB "github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	cacheUserPrefix  = "user[%v]"
	tokenNameKey     = "app.token_name"
	defaultTokenName = "INIS_LOGIN_TOKEN"
	// jwtErrorKey - Jwt() 解析失败时暂存错误的上下文键。
	// Rule() 依据接口类型决定是否因此拒绝请求：
	// 公共接口（type=common）不受坏 token 影响，需登录的接口才会 401。
	jwtErrorKey = "jwt_error"
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

// tokenFingerprint - 提取 token 指纹（签发时间/签发者/签名尾段），用于定位失败 token 的来源。
// 通过 iat（签发时间）可判断 token 是何时被哪个实例签发的：
// 若 iat 早于本实例密钥的启用时间，则为旧密钥/其他实例签发的残留 token。
func tokenFingerprint(token string) map[string]any {
	result := map[string]any{"token_length": len(token)}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		result["format"] = "malformed（非三段式，可能携带了 Bearer 前缀或脏数据）"
		result["token_head"] = token[:min(16, len(token))]
		return result
	}

	// 记录签名末段，同一密钥签出的 token 签名不同，但可用于跨日志比对
	sig := parts[2]
	result["sig_tail"] = sig[max(0, len(sig)-8):]

	// 解码 payload，取签发时间与签发者
	if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
		var claims struct {
			Iss string `json:"iss"`
			Sub string `json:"sub"`
			Iat int64  `json:"iat"`
		}
		if err := json.Unmarshal(payload, &claims); err == nil {
			if claims.Iat > 0 {
				result["iat"] = time.Unix(claims.Iat, 0).Format("2006-01-02 15:04:05")
			}
			result["iss"] = claims.Iss
			result["sub"] = claims.Sub
		}
	}
	return result
}

// jwtErrorMessage 根据 JWT 真实错误类型生成用户提示，
// 并将非过期原因记录日志。此前所有解析错误都被统一显示为"登录已过期"，
// 导致签名密钥变化（signature is invalid）等关键问题被掩盖、无法定位。
func jwtErrorMessage(ctx *gin.Context, err error) string {
	if err == nil {
		return facade.Lang(ctx, "禁止非法操作！")
	}
	if errors.Is(err, JWTLIB.ErrTokenExpired) {
		return facade.Lang(ctx, "登录已过期，请重新登录！")
	}

	return facade.Lang(ctx, "登录状态异常，请重新登录！")
}

// logJwtFailure - 记录 JWT 校验失败的详细上下文（含 token 指纹），便于定位 token 来源
func logJwtFailure(ctx *gin.Context, token string, err error) {
	facade.Log.Warn(map[string]any{
		"error":  err.Error(),
		"token":  tokenFingerprint(token),
		"ip":     ctx.ClientIP(),
		"path":   ctx.Request.URL.Path,
		"origin": ctx.Request.Header.Get("Origin"),
		"referer": ctx.Request.Header.Get("Referer"),
	}, "JWT 校验失败（非过期原因）：若 iat 早于本实例密钥启用时间或签名与最近签发的 token 不一致，说明该 token 由其他后端实例或旧密钥签发")
}

// getUserInfoWithCache 获取用户信息（带缓存逻辑）
func getUserInfoWithCache(uid any, jwtValid int64) (map[string]any, error) {
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
		go facade.Cache.Set(cacheName, user, time.Duration(jwtValid)*time.Second)
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
				if cast.ToInt(banMap["status"]) == model.BanStatusActive {
					reason := cast.ToString(banMap["reason"])
					duration := cast.ToInt(banMap["duration"])
					expiresAt := cast.ToInt64(banMap["expires_at"])
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

// Jwt - JWT 校验中间件（容错模式）
//
// 语义设计：本中间件只负责「身份识别」，不负责「访问控制」。
//   - 无 token           → 按匿名放行（是否允许由 Rule() 依据接口类型决定）
//   - token 有效         → 校验用户存在/状态后写入 ctx["user"]
//   - token 无效（坏/过期/他实例签发）→ 不中断请求，
//     仅记录日志并把错误暂存 ctx[jwtErrorKey]，交给 Rule() 决策：
//     公共接口（type=common）即使携带坏 token 也正常返回；
//     真正需要登录的接口才会 401，并顺带清除客户端无效 cookie。
//
// 此前的实现是「有 token 就必须有效，否则 401」，导致携带旧/跨实例 cookie
// 的匿名访问（含公共 GET 与登录接口本身）全部被拒，表现为"无论是否登录，
// 所有接口都返回登录状态异常"，用户连重新登录的入口都没有。
func Jwt() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := getTokenFromHeaderOrCookie(ctx, getTokenName())

		if utils.Is.Empty(token) {
			ctx.Next()
			return
		}

		jwtResult := facade.Jwt().Parse(token)
		if jwtResult.Error != nil {
			// 记录失败上下文（含 token 指纹，便于定位是否由他实例/旧密钥签发）
			logJwtFailure(ctx, token, jwtResult.Error)
			ctx.Set(jwtErrorKey, jwtResult.Error)
			ctx.Next()
			return
		}

		user, err := getUserInfoWithCache(jwtResult.Data["uid"], jwtResult.Valid)
		if err != nil {
			ctx.Set(jwtErrorKey, err)
			ctx.Next()
			return
		}

		if err := validateUserStatus(user); err != nil {
			ctx.Set(jwtErrorKey, err)
			ctx.Next()
			return
		}

		ctx.Set("user", user)
		ctx.Next()
	}
}
