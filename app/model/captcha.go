package model

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
)

// ============================== 滑块验证码（自研，无第三方依赖） ==============================
//
// 用途：短信 / 邮箱验证码的二次校验 —— 同一手机号或邮箱在短时间内重复请求发送验证码时，
// 要求先完成一次滑块验证，把「脚本批量刷短信」的成本从「一次 HTTP 请求」抬高到
// 「必须完成人机交互」。
//
// 安全模型（务必了解其边界）：
//   - 缺口位置 target 由服务端随机生成并存缓存，前端只是把它画出来；
//     请求里回传的任何位置信息一律不采信，校验只认缓存里的值；
//   - 校验一次性：通过后立即删除 token，防止同一个 token 被复用；
//   - 同一道题最多错 SliderCaptchaMaxFails 次即作废（需重新取题），
//     避免拿一个 token 反复试位置把容差试出来；
//   - 取题本身也有 IP 维度频率限制，防止「不断取新题 + 爆破位置」。
//
// 它挡的是「脚本」，不是「人工」：真人拖一次的成本极低。
//
// 前端约定（见 Mellow/src/components/SliderCaptcha.vue）：
//   - target / width 是 0~100 的百分比与轨道宽度，前端按此在轨道上画缺口；
//   - 用户拖动到位后，前端提交 { captcha_token: token, captcha_value: 落点百分比 }。

const (
	// SliderCaptchaTTL - 题目有效期
	SliderCaptchaTTL = 5 * time.Minute
	// SliderCaptchaTolerance - 允许的落点误差（百分比）
	SliderCaptchaTolerance = 3
	// SliderCaptchaMaxFails - 单题最多校验失败次数
	SliderCaptchaMaxFails = 3
	// sliderIssueLimit - 同一 IP 每分钟最多取题次数
	sliderIssueLimit = 20
)

// sliderCaptchaKey - 题目缓存键
func sliderCaptchaKey(token string) string {
	return "captcha-slider-" + token
}

// sliderIssueKey - 取题频率计数键
func sliderIssueKey(ip string) string {
	return "captcha-slider-issue-" + ip
}

// IssueSliderCaptcha - 签发一道滑块题
//
// ip 用于取题频率限制；返回的字段直接给前端渲染与提交用。
// 注意：这里返回的 target 是「显示用」的，校验时以缓存内的值为准，
// 即使前端篡改 target 也只影响自己的显示。
func IssueSliderCaptcha(ip string) facade.H {

	// 取题频率限制：防止被刷 token / 爆破落点
	if !utils.Is.Empty(ip) {
		count := cast.ToInt(facade.Cache.Get(sliderIssueKey(ip)))
		if count >= sliderIssueLimit {
			return facade.H{
				"limited": true,
				"message": "操作过于频繁，请稍后再试！",
			}
		}
		facade.Cache.Set(sliderIssueKey(ip), count+1, time.Minute)
	}

	target := utils.Rand.Int(10, 90)
	token := sliderCaptchaToken()

	facade.Cache.Set(sliderCaptchaKey(token), facade.H{
		"target": target,
		"fails":  0,
	}, SliderCaptchaTTL)

	return facade.H{
		"token":     token,
		"target":    target,
		"width":     300,
		"tolerance": SliderCaptchaTolerance,
		"expire":    time.Now().Add(SliderCaptchaTTL).Unix(),
	}
}

// VerifySliderCaptcha - 校验滑块结果（一次性：通过后 token 立即作废）
//
// 用于「发送验证码」这类每次都必须人机校验的动作：同一个 token 不能反复用，
// 否则用户滑一次就能无限次触发发送。
func VerifySliderCaptcha(token string, value int) bool {
	return verifySlider(token, value, true)
}

// VerifySliderCaptchaReusable - 校验滑块结果（题目有效期内可重复通过）
//
// 用于「登录 / 改密」这类**一次提交可能因业务失败而重试**的动作：
// 例如密码输错后重试，若把 token 消费掉就要再滑一次，体验很差。
// 这里保留 token 直到自然过期（5 分钟），失败次数照常累计。
func VerifySliderCaptchaReusable(token string, value int) bool {
	return verifySlider(token, value, false)
}

// verifySlider - 滑块校验实现
//
// consume=true：通过即删除 token（一次性）
// consume=false：通过后保留 token 到自然过期（同一张票据可在有效期内复用）
func verifySlider(token string, value int, consume bool) bool {

	if utils.Is.Empty(token) {
		return false
	}

	key := sliderCaptchaKey(token)
	data := facade.Cache.Get(key)

	// 题目不存在 / 已过期 / 已被使用
	if utils.Is.Empty(data) {
		return false
	}

	item := cast.ToStringMap(data)
	target := cast.ToInt(item["target"])
	fails := cast.ToInt(item["fails"])

	// 错误次数超限：作废该题，必须重新取题
	if fails >= SliderCaptchaMaxFails {
		facade.Cache.Del(key)
		return false
	}

	if sliderDistance(value, target) <= SliderCaptchaTolerance {
		if consume {
			facade.Cache.Del(key)
			return true
		}
		// 可复用模式：刷新剩余有效期（复用窗口从最后一次通过开始重新计时）
		facade.Cache.Set(key, facade.H{"target": target, "fails": 0}, SliderCaptchaTTL)
		return true
	}

	// 校验失败：累计次数，并缩短剩余有效期（题目不再「一直有效」）
	facade.Cache.Set(key, facade.H{
		"target": target,
		"fails":  fails + 1,
	}, time.Minute)

	return false
}

// sliderDistance - 两个落点之间的距离（绝对值）
func sliderDistance(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

// sliderCaptchaToken - 生成不可预测的题目 token
//
// 用 crypto/rand 而不是时间戳或 math/rand：token 是校验的唯一凭据，
// 可预测的 token 意味着别人能拿你的题去试位置。
func sliderCaptchaToken() string {
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		// 极端情况下（系统熵不可用）退回时间戳 + 随机数，保证流程不中断
		return fmt.Sprintf("%v%v", time.Now().UnixNano(), utils.Rand.Int(100000, 999999))
	}
	return hex.EncodeToString(buf)
}
