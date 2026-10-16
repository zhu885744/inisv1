package controller

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ============================== 修改账号（用户名） ==============================
//
// 账号（users.account）既是登录凭证（登录按 account / email / phone 三字段匹配），
// 也是对外展示的身份标识，改完还会占掉别人的登录名 —— 属于高风险且不可逆的操作。
//
// 所以「能不能改、多久能改一次、改之前要验证什么」全部交给后台配置
// （config 表 SYSTEM_ACCOUNT_CHANGE，见 model/account-change.go）：
//
//	enabled         总开关（默认关闭）
//	cooldown_days   冷却天数（默认 30：改完 30 天内不可再改）
//	require_contact 是否要求账号已绑定手机号或邮箱（默认要求）
//	require_verify  是否要求身份验证（默认要求）
//
// 接口：
//
//	GET  /api/users/account-rule   读取规则 + 我自己的冷却状态（前台据此决定是否显示入口）
//	PUT  /api/users/change-account 提交修改
//
// 身份验证复用统一框架（users-identity.go 的 verifyIdentity）：
// 当前密码 / 邮箱验证码 / 手机验证码三选一，发送验证码见 users/verify-code。
//
// 参数：
//
//	account    新账号（4-20 位，字母 / 数字 / 下划线）
//	password   当前密码   ┐
//	email_code 邮箱验证码 ├ 身份验证三选一（require_verify 关闭时不需要）
//	phone_code 手机验证码 ┘

// accountPattern - 账号格式：字母 / 数字 / 下划线 + 长度限制
var accountPattern = regexp.MustCompile(fmt.Sprintf(`^[A-Za-z0-9_]{%v,%v}$`, model.AccountMinLength, model.AccountMaxLength))

// accountFormatMessage - 账号格式校验，返回空字符串表示通过
//
// 只允许字母 / 数字 / 下划线：邮箱与手机号都含其它字符，
// 这样新账号不可能与「他人邮箱 / 手机号」长得一样（登录按三个字段匹配）。
func accountFormatMessage(account string) string {

	if utils.Is.Empty(account) {
		return "请输入新账号！"
	}

	if !accountPattern.MatchString(account) {
		return fmt.Sprintf("账号为 %v-%v 位字母、数字或下划线！", model.AccountMinLength, model.AccountMaxLength)
	}

	return ""
}

// accountCooldownRemaining - 距离下次可修改还剩几天（0 = 现在就能改）
func accountCooldownRemaining(setting model.AccountChangeSetting, lastTime int64) int {

	if setting.CooldownDays <= 0 || lastTime <= 0 {
		return 0
	}

	left := lastTime + int64(setting.CooldownDays)*86400 - time.Now().Unix()
	if left <= 0 {
		return 0
	}

	// 向上取整：还剩 1 秒也算「还需 1 天」
	return int((left + 86399) / 86400)
}

// accountCooldownMessage - 冷却提示文案
func accountCooldownMessage(cooldownDays, remaining int) string {
	return fmt.Sprintf("账号修改后 %v 天内不可再次修改，还需 %v 天！", cooldownDays, remaining)
}

// accountRuleData - 组装规则数据（含当前用户自身的冷却状态）
//
// isRoot：超级管理员不受冷却限制（便于运维与排障），与 changeAccount 的判断保持一致
func (this *Users) accountRuleData(user model.Users, isRoot bool) map[string]any {

	setting := model.AccountChangeConfig()

	hasContact := !utils.Is.Empty(user.Email) || !utils.Is.Empty(user.Phone)

	remaining := 0
	nextTime := int64(0)
	if !isRoot {
		remaining = accountCooldownRemaining(setting, user.AccountUpdateTime)
		if remaining > 0 {
			nextTime = user.AccountUpdateTime + int64(setting.CooldownDays)*86400
		}
	}

	// 综合判定：开关 → 联系方式要求 → 冷却（身份验证在提交时才校验）
	canChange := setting.Enabled
	message := ""
	switch {
	case !setting.Enabled:
		message = "管理员未开放账号修改功能"
	case setting.RequireContact && !hasContact:
		canChange = false
		message = "请先绑定手机号或邮箱再修改账号"
	case remaining > 0:
		canChange = false
		message = fmt.Sprintf("账号修改后 %v 天内不可再次修改，还需 %v 天", setting.CooldownDays, remaining)
	}

	return map[string]any{
		"enabled":         setting.Enabled,
		"cooldown_days":   setting.CooldownDays,
		"require_contact": setting.RequireContact,
		"require_verify":  setting.RequireVerify,
		"has_contact":     hasContact,
		"can_change":      canChange,
		"next_time":       nextTime,
		"remaining_days":  remaining,
		"min_length":      model.AccountMinLength,
		"max_length":      model.AccountMaxLength,
		"message":         message,
	}
}

// accountRule 读取账号修改规则（GET）
//
// 前台在「账号安全」页加载时调用：未开放 / 未绑联系方式 / 仍在冷却期时不显示修改入口，
// 直接把 message 展示给用户，避免用户点进去才被拒。
func (this *Users) accountRule(ctx *gin.Context) {

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.fail(ctx, "请先登录！", 401)
		return
	}

	this.json(ctx, this.accountRuleData(user, this.meta.root(ctx)), facade.Lang(ctx, "数据请求成功！"), 200)
}

// changeAccount 修改账号（PUT）
//
// 校验顺序与提示文案都刻意「不泄露信息」：账号唯一性失败时统一回「已被占用」，
// 不区分是被账号 / 邮箱 / 手机号占用（否则等于送了一个账号枚举接口）。
func (this *Users) changeAccount(ctx *gin.Context) {

	params := this.params(ctx)

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.fail(ctx, "请先登录！", 401)
		return
	}

	setting := model.AccountChangeConfig()

	// ① 总开关：默认关闭，只有管理员显式开启后才可用
	if !setting.Enabled {
		this.fail(ctx, "管理员未开放账号修改功能！", 403)
		return
	}

	// ② 新账号格式
	account := strings.TrimSpace(cast.ToString(params["account"]))
	if message := accountFormatMessage(account); message != "" {
		this.fail(ctx, message)
		return
	}

	// ③ 与原账号相同
	if account == user.Account {
		this.fail(ctx, "新账号不能与当前账号相同！")
		return
	}

	// ④ 联系方式要求：账号本身不是联系方式，出了问题必须能联系到本人
	if setting.RequireContact && utils.Is.Empty(user.Email) && utils.Is.Empty(user.Phone) {
		this.fail(ctx, "请先绑定手机号或邮箱再修改账号！")
		return
	}

	// ⑤ 冷却：超级管理员豁免（便于运维 / 排障）
	isRoot := this.meta.root(ctx)
	if !isRoot {
		if remaining := accountCooldownRemaining(setting, user.AccountUpdateTime); remaining > 0 {
			this.fail(ctx, accountCooldownMessage(setting.CooldownDays, remaining))
			return
		}
	}

	// ⑥ 身份验证（复用统一框架：当前密码 / 邮箱验证码 / 手机验证码三选一）
	if setting.RequireVerify {
		if err := this.verifyIdentity(ctx, user, params); err != nil {
			return
		}
	}

	// ⑦ 唯一性：既不能占用他人账号，也不能与他人邮箱 / 手机号重名
	// （登录按 account / email / phone 匹配，重名会让一次登录指向两个账号）
	conflict := false
	for _, field := range []string{"account", "email", "phone"} {
		exist, _ := facade.DB.Model(&model.Users{}).Where(field, account).Where("id", "!=", user.Id).Exist()
		if exist {
			conflict = true
			break
		}
	}
	if conflict {
		this.fail(ctx, "该账号已被占用，请更换其它账号！")
		return
	}

	// ⑧ 写库：账号 + 本次修改时间（下次冷却的依据）
	now := time.Now().Unix()
	if _, err := facade.DB.Model(&model.Users{}).Where("id", user.Id).Update(map[string]any{
		"account":             account,
		"account_update_time": now,
	}); err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": user.Id}, "修改账号失败")
		this.fail(ctx, "修改失败，请稍后重试！")
		return
	}

	// 清资料缓存（口径与 users/update、users/change-password 一致）
	facade.Cache.Del(fmt.Sprintf("user[%v]", user.Id))

	// 审计日志：只记 uid 与新账号（旧账号无需留痕，也不落任何凭证）
	facade.Log.Info(map[string]any{"user_id": user.Id, "account": account}, "用户修改账号")

	this.json(ctx, gin.H{
		"id":                  user.Id,
		"account":             account,
		"account_update_time": now,
	}, facade.Lang(ctx, "账号修改成功！"), 200)
}
