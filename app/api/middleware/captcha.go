package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
)

// ============================== 验证码防刷 / 敏感操作滑块 ==============================
//
// 两层能力，开关都在后台「系统配置 → 安全 → 安全验证（滑块）」（config 表 SYSTEM_CAPTCHA）：
//
//  1. 发送验证码防刷（默认开启，不可按接口单独配）：同一手机号 / 邮箱在
//     sendCodeCaptchaWindow 内再次请求发送验证码时，必须先通过一次滑块；
//  2. 敏感操作滑块（默认关闭，按场景配置）：登录 / 验证码登录 / 注册 / 重置密码 /
//     改绑邮箱手机号 / 注销账户 —— 打开后每次提交都要先过滑块。
//
// 与前端的分工：任一路径判定为「需要滑块」时返回 code=428 + { need_captcha, captcha }，
// 前端拦截器（Mellow/src/api/request.js）统一弹出滑块并在通过后**自动重放原请求**，
// 因此各业务接口本身不需要做任何判断。

// sendCodeRoutes - 会「发送短信 / 邮件验证码」的接口（METHOD + PATH）
//
// 判定口径：参数里没有 code 的请求 = 发送阶段。
var sendCodeRoutes = map[string]bool{
	"POST /api/comm/sign-code":      true,
	"POST /api/comm/register":       true,
	"POST /api/comm/reset-password": true,
	"PUT /api/users/email":          true,
	"PUT /api/users/phone":          true,
	"DELETE /api/users/destroy":     true,
}

// captchaSceneRoutes - 敏感操作 → 场景开关（METHOD + PATH）
//
// 说明：
//   - login 本身不带 code，命中即校验；
//   - 其余路径同时是「发送验证码」入口，只有带 code 时才算「提交阶段」，
//     走场景开关；不带 code 的发送阶段由上面的窗口规则处理。
var captchaSceneRoutes = map[string]string{
	"POST /api/comm/login":          model.CaptchaSceneLogin,
	"POST /api/comm/sign-code":      model.CaptchaSceneCodeLogin,
	"POST /api/comm/register":       model.CaptchaSceneRegister,
	"POST /api/comm/reset-password": model.CaptchaSceneResetPassword,
	"PUT /api/users/email":          model.CaptchaSceneChangeContact,
	"PUT /api/users/phone":          model.CaptchaSceneChangeContact,
	"DELETE /api/users/destroy":     model.CaptchaSceneDestroy,
}

const (
	// sendCodeCaptchaWindow - 同一手机号/邮箱在这个窗口内的第二次（及以后）发送请求需要先过滑块
	sendCodeCaptchaWindow = 5 * time.Minute
	// sendCodeMemory - 发送时间的保留时长（大于窗口，保证窗口判断可靠）
	sendCodeMemory = 30 * time.Minute
)

// CaptchaGuard - 验证码防刷闸门（挂在 /api 中间件链的最后）
func CaptchaGuard() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		routeKey := ctx.Request.Method + " " + ctx.Request.URL.Path

		params, _ := ctx.Get("params")
		values := cast.ToStringMap(params)

		// 带 code = 提交验证码阶段（校验登录 / 改绑 / 注销 / 注册）
		hasCode := !utils.Is.Empty(values["code"])

		// ---------- 1. 敏感操作的场景开关 ----------
		if scene, ok := captchaSceneRoutes[routeKey]; ok {
			// sign-code / register / reset-password / email / phone / destroy 只在提交阶段拦，
			// 发送阶段交给下面的窗口规则（判定口径不同，不能混）
			isSubmitStage := scene == model.CaptchaSceneLogin || hasCode
			if isSubmitStage && model.CaptchaSceneEnabled(scene) {
				// 复用式校验：密码输错重试、验证码填错重试时不必反复滑
				if !requireCaptcha(ctx, values, true) {
					return
				}
			}
		}

		// ---------- 2. 发送验证码的窗口防刷 ----------
		if !hasCode && sendCodeRoutes[routeKey] && model.CaptchaSceneEnabled(model.CaptchaSceneSendCode) {
			if !sendCodeWindowGuard(ctx, values) {
				return
			}
		}

		ctx.Next()
	}
}

// requireCaptcha - 要求通过滑块验证（内部直接响应 428 并中断，返回 false）
//
// reusable=true 时同一张票据在有效期内可重复使用（适合「提交后可能重试」的场景）
func requireCaptcha(ctx *gin.Context, values map[string]any, reusable bool) bool {
	token := cast.ToString(values["captcha_token"])
	value := cast.ToInt(values["captcha_value"])

	passed := false
	if reusable {
		passed = model.VerifySliderCaptchaReusable(token, value)
	} else {
		// 一次性：每次发送验证码都必须重新滑（否则滑一次就能无限触发发送）
		passed = model.VerifySliderCaptcha(token, value)
	}

	if passed {
		return true
	}

	abortNeedCaptcha(ctx)
	return false
}

// sendCodeWindowGuard - 发送验证码的窗口防刷
//
// 首次请求（或距上次超过窗口）直接放行并记录时间；窗口内的后续请求必须过滑块。
func sendCodeWindowGuard(ctx *gin.Context, values map[string]any) bool {
	target, kind := sendCodeTarget(ctx, values)
	if utils.Is.Empty(target) {
		// 目标缺失（未登录 / 未提交联系方式）：交给业务层返回参数错误
		return true
	}

	key := fmt.Sprintf("send-code-guard-%v-%v", kind, target)
	last := facade.Cache.Get(key)
	withinWindow := !utils.Is.Empty(last) &&
		time.Now().Unix()-cast.ToInt64(last) < int64(sendCodeCaptchaWindow.Seconds())

	if withinWindow && !requireCaptcha(ctx, values, false) {
		return false
	}

	// 记录本次发送时间，作为下一次「5 分钟窗口」的起点
	facade.Cache.Set(key, time.Now().Unix(), sendCodeMemory)
	return true
}

// abortNeedCaptcha - 返回「需要滑块」并附上题目
//
// 题目随响应一起下发，前端无需再请求一次；数据结构见 model.IssueSliderCaptcha。
func abortNeedCaptcha(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"code": 428,
		"msg":  facade.Lang(ctx, "请先完成滑块验证！"),
		"data": gin.H{
			"need_captcha": true,
			"captcha":      model.IssueSliderCaptcha(ctx.ClientIP()),
		},
	})
	ctx.Abort()
}

// sendCodeTarget - 识别「验证码发往哪个手机号 / 邮箱」
//
// 优先取请求参数（注册、验证码登录、找回密码、改绑都是显式提交目标）；
// 注销账户发的是当前登录用户已绑定的联系方式（参数里没有），因此回退到 ctx 里的登录用户。
func sendCodeTarget(ctx *gin.Context, values map[string]any) (target string, kind string) {
	for _, key := range []string{"social", "email", "phone", "mobile", "contact"} {
		if value := normalizeContact(cast.ToString(values[key])); !utils.Is.Empty(value) {
			return value, kindOfContact(value)
		}
	}

	user := getUserFromContext(ctx)
	for _, value := range []string{user.Email, user.Phone} {
		if normalized := normalizeContact(value); !utils.Is.Empty(normalized) {
			return normalized, kindOfContact(normalized)
		}
	}

	return "", ""
}

// normalizeContact - 归一化联系方式：邮箱转小写、手机号去空格与连字符，
// 否则「180 0730 0738」「ABC@x.com」这类写法可以绕过同一个目标的限流。
func normalizeContact(value string) string {
	if utils.Is.Empty(value) {
		return ""
	}
	value = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(value), " ", ""), "-", "")
	return strings.ToLower(value)
}

// kindOfContact - 区分邮箱 / 手机号，避免两者的缓存键互相撞车
func kindOfContact(value string) string {
	if utils.Is.Email(value) {
		return "email"
	}
	if utils.Is.Phone(value) {
		return "phone"
	}
	return "contact"
}
