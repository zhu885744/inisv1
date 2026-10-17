package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ========================= 敏感操作统一身份验证框架 =========================
//
// 适用场景：修改密码 / 修改邮箱 / 修改手机号 / 注销账户。
// 这些操作一旦被他人执行就不可逆（账号被夺、联系方式被换、账号被清空），
// 所以都不能只凭「登录态」放行 —— 必须再证明一次「操作者是本人」。
//
// 原先 email / phone / destroy 各写了一套「发验证码 + 校验验证码 + 频控」，
// 三份代码的缓存 key、频控口径、错误文案都要各自维护，改一处要同步三处。
// 这里抽成四个原语，各接口共用（`account` 改账号的接口当前未开放，开放时直接接入即可）：
//
//	sendCode        发送验证码（场景 / 频道隔离的缓存 key + 60 秒与每日 10 次频控）
//	checkCode       校验验证码（校验通过立即删除防重放，错试 5 次作废）
//	verifyPassword  校验当前密码
//	verifyIdentity  身份验证三选一：password > email_code > phone_code
//
// 各接口的用法：
//
//	email / phone     verifyIdentity（当前密码或已绑定联系方式验证码）+ 新联系方式的 code
//	change-password   verifyIdentity（老用户用旧密码，第三方登录用户用邮箱/手机验证码）
//	destroy           checkCode（发到已绑定联系方式）+ verifyPassword —— 保留双因子
//
// 约定：本文件里的 `fail` 会**立即写响应**并返回 error，
// 调用方拿到 error 直接 return 即可，不需要再自己 this.json。

const (
	// 验证码场景：参与缓存 key，避免「改邮箱的验证码」被拿去「改密码」
	codeSceneIdentity    = "identity"     // 身份验证码（发到已绑定的邮箱 / 手机号）
	codeSceneChangeEmail = "change-email" // 新邮箱的验证码（证明新邮箱归你所有）
	codeSceneChangePhone = "change-phone" // 新手机号的验证码
	codeSceneDestroy     = "destroy"      // 注销账户的验证码

	// 验证码频道：同时决定用哪个驱动（drive.email / drive.sms）
	channelEmail = "email"
	channelSMS   = "sms"

	// 验证码：有效期 5 分钟、60 秒内不可重发、每日最多 10 条
	codeExpire        = 5 * time.Minute
	codeResendGap     = 60 // 秒
	codeDailyLimit    = 10
	codeDailyLimitTTL = 24 * time.Hour

	// 验证码最多可试错 5 次，超过即作废：
	// 6 位数字码在 5 分钟窗口内若不限次，可被暴力枚举
	codeMaxAttempt = 5

	// 新密码长度：下限与前端表单一致（6 位），上限 32 位
	// （bcrypt 只取前 72 字节，过长的密码会被静默截断，不如直接拦住）
	passwordMinLen = 6
	passwordMaxLen = 32
)

// weakPasswords - 高频弱密码（以小写比较）
//
// 这些口令在各类泄露字典里排名最前，撞库成功率极高，直接在设置环节拒掉。
var weakPasswords = map[string]bool{
	"123456":     true,
	"1234567":    true,
	"12345678":   true,
	"123456789":  true,
	"1234567890": true,
	"password":   true,
	"password1":  true,
	"qwerty":     true,
	"qwerty123":  true,
	"abc123":     true,
	"111111":     true,
	"000000":     true,
	"iloveyou":   true,
	"admin123":   true,
	"a123456":    true,
}

// codeCacheKey - 验证码缓存 key：场景 + 频道 + 联系方式 三段隔离
func codeCacheKey(scene, channel, contact string) string {
	return fmt.Sprintf("[code][%v][%v=%v]", scene, channel, contact)
}

// codeAttemptKey - 验证码错试计数 key（与验证码同生命周期）
func codeAttemptKey(cacheName string) string {
	return "code-error-" + cacheName
}

// fail - 统一失败响应：立即写响应并返回 error，供调用方 `return this.fail(...)` 使用
func (this *Users) fail(ctx *gin.Context, message string, status ...int) error {

	code := 400
	if len(status) > 0 {
		code = status[0]
	}

	this.json(ctx, nil, facade.Lang(ctx, message), code)

	return errors.New(message)
}

// sendCode - 统一发送验证码
//
//	scene   场景（隔离缓存 key）
//	channel 频道（email / sms）
//	contact 目标联系方式
//
// 无论成功还是失败都会自行响应客户端（201 已发送 / 400 失败），调用方直接 return 即可。
// 「验证码为空 = 发送验证码」是这几个接口的既有约定：同一个接口传 code 与否决定发码还是校验。
func (this *Users) sendCode(ctx *gin.Context, scene, channel, contact string) {

	if utils.Is.Empty(contact) {
		this.fail(ctx, "联系方式不能为空！")
		return
	}

	// 驱动：邮箱 / 短信服务是否已开启
	drive := cast.ToString(facade.SMSToml.Get(fmt.Sprintf("drive.%v", channel)))
	if utils.Is.Empty(drive) {
		if channel == channelEmail {
			this.fail(ctx, "管理员未开启邮箱服务，无法发送验证码！")
		} else {
			this.fail(ctx, "管理员未开启短信服务，无法发送验证码！")
		}
		return
	}

	cacheName := codeCacheKey(scene, channel, contact)
	frequencyCacheName := "frequency-" + cacheName
	dailyLimitCacheName := "daily-limit-" + cacheName

	// 频控①：60 秒内不可重发
	if last := facade.Cache.Get(frequencyCacheName); !utils.Is.Empty(last) {
		if time.Now().Unix()-cast.ToInt64(last) < codeResendGap {
			this.fail(ctx, "发送过于频繁，请60秒后再试！")
			return
		}
	}

	// 频控②：每日发送上限
	dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
	if dailyCount >= codeDailyLimit {
		this.fail(ctx, "今日发送验证码次数已达上限，请明日再试！")
		return
	}

	sms := facade.NewSMS(drive).VerifyCode(contact)
	if sms.Error != nil {
		// 阿里云等服务端也有频控，把驱动原文转成更友好的提示
		if strings.Contains(sms.Error.Error(), "check frequency failed") || strings.Contains(sms.Error.Error(), "FREQUENCY_FAIL") {
			this.fail(ctx, "发送过于频繁，请稍后再试！")
			return
		}
		this.fail(ctx, sms.Error.Error())
		return
	}

	// 同步写缓存：旧实现用了 go（异步），响应 201 时验证码可能还没落缓存，
	// 用户立刻提交会拿到「验证码错误」
	facade.Cache.Set(cacheName, sms.VerifyCode, codeExpire)
	facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*time.Duration(codeResendGap))
	facade.Cache.Set(dailyLimitCacheName, dailyCount+1, codeDailyLimitTTL)

	// 重发即重置错试计数：上一轮的失败次数不该影响新验证码
	facade.Cache.Del(codeAttemptKey(cacheName))

	// 提示里只出现脱敏后的联系方式，避免日志 / 页面留存明文
	if channel == channelEmail {
		this.json(ctx, nil, facade.Lang(ctx, fmt.Sprintf("验证码发送至您的邮箱：%s，请注意查收！", facade.Comm.MaskEmail(contact))), 201)
	} else {
		this.json(ctx, nil, facade.Lang(ctx, fmt.Sprintf("验证码发送至您的手机：%s，请注意查收！", facade.Comm.MaskPhone(contact))), 201)
	}
}

// checkCode - 统一校验验证码
//
// 校验通过立即删除（防重放）；错试达到 codeMaxAttempt 次则作废验证码，必须重新获取。
// 返回 error 表示未通过（并且已经响应过），调用方直接 return。
func (this *Users) checkCode(ctx *gin.Context, scene, channel, contact, code string) error {

	if utils.Is.Empty(code) {
		return this.fail(ctx, "请输入验证码！")
	}

	cacheName := codeCacheKey(scene, channel, contact)
	attemptKey := codeAttemptKey(cacheName)

	cacheCode := facade.Cache.Get(cacheName)
	if utils.Is.Empty(cacheCode) {
		return this.fail(ctx, "验证码已过期，请重新获取！")
	}

	if cast.ToString(code) != cast.ToString(cacheCode) {

		attempt := cast.ToInt(facade.Cache.Get(attemptKey)) + 1
		facade.Cache.Set(attemptKey, attempt, codeExpire)

		if attempt >= codeMaxAttempt {
			facade.Cache.Del(cacheName)
			facade.Cache.Del(attemptKey)
			return this.fail(ctx, "验证码错误次数过多，请重新获取！")
		}

		return this.fail(ctx, "验证码错误！")
	}

	// 一次性使用
	facade.Cache.Del(cacheName)
	facade.Cache.Del(attemptKey)

	return nil
}

// readPassword - 单独读取当前用户的密码哈希
//
// 不能走 users/one 那套查询：那里 WithoutField("password") 脱敏过，
// 拿不到原值就没法比对旧密码。
func (this *Users) readPassword(uid int) (string, error) {

	item, err := facade.DB.Model(&model.Users{}).Where("id", uid).Find()
	if err != nil {
		return "", err
	}

	return cast.ToString(item["password"]), nil
}

// verifyPassword - 校验当前密码；返回 error 表示未通过（并已响应）
func (this *Users) verifyPassword(ctx *gin.Context, user model.Users, password string) error {

	if utils.Is.Empty(password) {
		return this.fail(ctx, "请输入当前密码！")
	}

	hash, err := this.readPassword(user.Id)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": user.Id}, "读取用户密码失败")
		return this.fail(ctx, "系统繁忙，请稍后重试！")
	}

	// 第三方登录注册（或管理员建号时未设密码）的账号可能从未设置过密码
	if utils.Is.Empty(hash) {
		return this.fail(ctx, "当前账号未设置密码，请改用邮箱或手机验证码完成验证！")
	}

	if !utils.Password.Verify(hash, password) {
		return this.fail(ctx, "密码错误！")
	}

	return nil
}

// verifyIdentity - 统一身份验证：密码 / 邮箱验证码 / 手机验证码 三选一
//
// 从 params 按优先级取凭证：
//
//	password   —— 当前密码
//	email_code —— 发到「已绑定邮箱」的验证码
//	phone_code —— 发到「已绑定手机号」的验证码
//
// 一个都没传视为未验证 —— 敏感操作必须留一种可验证的凭证，不允许「登录即可改」。
// 返回 error 表示未通过（并已响应），调用方直接 return。
func (this *Users) verifyIdentity(ctx *gin.Context, user model.Users, params map[string]any) error {

	if !utils.Is.Empty(params["password"]) {
		return this.verifyPassword(ctx, user, cast.ToString(params["password"]))
	}

	if !utils.Is.Empty(params["email_code"]) {
		if utils.Is.Empty(user.Email) {
			return this.fail(ctx, "当前账号未绑定邮箱，请改用其它方式验证！")
		}
		return this.checkCode(ctx, codeSceneIdentity, channelEmail, user.Email, cast.ToString(params["email_code"]))
	}

	if !utils.Is.Empty(params["phone_code"]) {
		if utils.Is.Empty(user.Phone) {
			return this.fail(ctx, "当前账号未绑定手机号，请改用其它方式验证！")
		}
		return this.checkCode(ctx, codeSceneIdentity, channelSMS, user.Phone, cast.ToString(params["phone_code"]))
	}

	return this.fail(ctx, "请先完成身份验证（当前密码 或 邮箱/手机验证码）！")
}

// verifyIdentityFor - 改绑联系方式用的身份验证（按账号实际情况决定是否强制）
//
// 与 verifyIdentity 的区别：只有「可验证的凭证」存在时才强制验证 ——
//
//	有密码          → 必须给 password（或已绑定联系方式的验证码）
//	没密码、绑了联系方式 → 必须给对应联系方式的验证码
//	都没有           → 没有可验证的凭证，只能凭登录态 + 新联系方式验证码
//
// 最后一种是为了不把「纯第三方登录注册、未绑任何联系方式」的账号锁死：
// 它们没有密码、没有联系方式 → 永远过不了身份验证 → 永远绑不了邮箱 / 手机号
// → 也永远设不了密码（死锁）。这类账号的登录凭证在第三方平台手里，风险由其担保。
func (this *Users) verifyIdentityFor(ctx *gin.Context, user model.Users, params map[string]any) error {

	hash, err := this.readPassword(user.Id)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": user.Id}, "读取用户密码失败")
		return this.fail(ctx, "系统繁忙，请稍后重试！")
	}

	if utils.Is.Empty(hash) && utils.Is.Empty(user.Email) && utils.Is.Empty(user.Phone) {
		return nil
	}

	return this.verifyIdentity(ctx, user, params)
}

// passwordWeakMessage - 新密码强度校验，返回空字符串表示通过
//
// 「修改密码」（本文件）与「忘记密码 / 重置密码」（comm.resetPassword）共用同一套口径，
// 避免出现「登录时能设的密码，忘记密码时却设不了」这类不一致。
func passwordWeakMessage(password string) string {

	if utils.Is.Empty(password) {
		return "请输入新密码！"
	}

	// 按字符数（而不是字节数）计长度：中文等宽字符不会被算成 3 倍
	length := len([]rune(password))

	if length < passwordMinLen {
		return fmt.Sprintf("密码长度不能少于 %v 位！", passwordMinLen)
	}
	if length > passwordMaxLen {
		return fmt.Sprintf("密码长度不能超过 %v 位！", passwordMaxLen)
	}
	if strings.ContainsAny(password, " \t\r\n") {
		return "密码不能包含空格！"
	}
	if weakPasswords[strings.ToLower(password)] {
		return "该密码过于常见，请更换更复杂的密码！"
	}

	// 纯数字最容易被撞库：6 位数字只有 100 万种组合
	if _, err := strconv.Atoi(password); err == nil {
		return "密码不能是纯数字，请混合字母或符号！"
	}

	return ""
}

// sendVerifyCode 发送身份验证码
//
// PUT /api/users/verify-code —— 把验证码发到**当前登录用户已绑定**的邮箱 / 手机号，
// 供「修改密码 / 注销账户」等敏感操作做二次验证（也可用于没有旧密码的第三方登录账号）。
// 参数 mode：email / sms，不传时优先邮箱，未绑邮箱则用手机号。
func (this *Users) sendVerifyCode(ctx *gin.Context) {

	params := this.params(ctx)

	user := this.meta.user(ctx)
	// 即便中间件校验过登录，这里二次校验：接口权限若被改动，uid 是强制的，会造成意外
	if user.Id == 0 {
		this.fail(ctx, "请先登录！", 401)
		return
	}

	channel := cast.ToString(params["mode"])
	if channel != channelEmail && channel != channelSMS {
		if !utils.Is.Empty(user.Email) {
			channel = channelEmail
		} else {
			channel = channelSMS
		}
	}

	contact := user.Email
	if channel == channelSMS {
		contact = user.Phone
	}

	if utils.Is.Empty(contact) {
		if channel == channelEmail {
			this.fail(ctx, "当前账号未绑定邮箱！")
		} else {
			this.fail(ctx, "当前账号未绑定手机号！")
		}
		return
	}

	this.sendCode(ctx, codeSceneIdentity, channel, contact)
}

// changePassword 修改密码
//
// PUT /api/users/change-password —— 登录态 + 身份验证。
//
// 与「忘记密码 / 重置密码」（comm/reset-password：免登录 + 验证码）的分工：
//   - 本接口必须登录，且要先通过身份验证（当前密码，或已绑定联系方式的验证码）
//   - 从未设过密码的账号（第三方登录注册）没有旧密码可校验，
//     用邮箱 / 手机验证码通过身份验证后即可设置密码
//
// 参数：
//
//	password     新密码（必填）
//	old_password 当前密码  ┐
//	email_code   邮箱验证码 ├ 三选一（见 verifyIdentity），都没传则拒绝
//	phone_code   手机验证码 ┘
func (this *Users) changePassword(ctx *gin.Context) {

	params := this.params(ctx)

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.fail(ctx, "请先登录！", 401)
		return
	}

	// 身份验证：前端把「当前密码」叫 old_password，框架里统一叫 password，
	// 这里显式组装凭证，避免与新密码（password）撞字段
	proof := map[string]any{
		"password":   params["old_password"],
		"email_code": params["email_code"],
		"phone_code": params["phone_code"],
	}
	if err := this.verifyIdentity(ctx, user, proof); err != nil {
		return
	}

	password := cast.ToString(params["password"])

	// 新密码强度校验：前端有强度条，但接口可以被直接调用，后端必须再拦一次
	if message := passwordWeakMessage(password); message != "" {
		this.fail(ctx, message)
		return
	}

	// 与当前密码相同：拒绝 —— 避免「改了个寂寞」，也让用户察觉自己复用了旧密码
	hash, _ := this.readPassword(user.Id)
	if !utils.Is.Empty(hash) && utils.Password.Verify(hash, password) {
		this.fail(ctx, "新密码不能与当前密码相同！")
		return
	}

	hashed := utils.Password.Create(password)

	if _, err := facade.DB.Model(&model.Users{}).Where("id", user.Id).UpdateColumn("password", hashed); err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": user.Id}, "修改密码失败")
		this.fail(ctx, "修改失败，请稍后重试！")
		return
	}

	// 清资料缓存（口径与 users/update 一致）
	facade.Cache.Del(fmt.Sprintf("user[%v]", user.Id))

	// 改密后该账号的全部旧会话立即失效（其他设备强制下线）；
	// 当前设备续签一个新会话，避免用户刚改完密码就被踢回登录页
	facade.SessionDestroyUser(user.Id)

	var token string
	var valid int64
	if sid, expire, err := facade.SessionLogin(user.Id, utils.Hash.Sum32(hashed), ctx.ClientIP(), ctx.Request.UserAgent()); err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "uid": user.Id}, "改密后签发新会话失败")
	} else {
		token, valid = sid, expire
		setToken(ctx, sid)
	}

	// 审计日志只记 user_id：密码相关操作不落任何凭证明文
	facade.Log.Info(map[string]any{"user_id": user.Id}, "用户修改密码")

	this.json(ctx, gin.H{"id": user.Id, "token": token, "valid_time": valid}, facade.Lang(ctx, "密码修改成功！"), 200)
}
