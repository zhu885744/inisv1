package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ============================== 账号（用户名）修改规则 ==============================
//
// 配置存在 config 表的一条记录里（key = SYSTEM_ACCOUNT_CHANGE）：
//   - value：总开关（"0" 关闭 / "1" 开启）
//   - json ：冷却天数、是否要求已绑定联系方式、是否要求身份验证
//
// 后台在「系统设置 → 账号与登录 → 账号修改」里调整（Mellow/src/views/admin/SystemConfig.vue），
// 前台读取规则与提交改账号见 app/api/controller/users-account.go。
//
// 为什么默认关闭：账号既是登录凭证（登录按 account / email / phone 匹配）又是对外身份标识，
// 改完还会占掉别人的登录名，被他人操作则不可逆 —— 开不开放交给站点自己决定。
const AccountChangeConfigKey = "SYSTEM_ACCOUNT_CHANGE"

// 账号长度（与前端 CONST.ACCOUNT_MIN / ACCOUNT_MAX 一致）
const (
	AccountMinLength = 4
	AccountMaxLength = 20
)

// AccountCooldownDefault - 默认冷却天数（改完 30 天内不可再改）
const AccountCooldownDefault = 30

// AccountChangeSetting - 账号修改规则（已归一化，可直接使用）
type AccountChangeSetting struct {
	Enabled        bool // 总开关
	CooldownDays   int  // 冷却天数（0 = 不限制）
	RequireContact bool // 是否要求账号已绑定手机号或邮箱
	RequireVerify  bool // 是否要求身份验证（当前密码 / 邮箱或手机验证码）
}

// defaultAccountChangeConfig - 默认配置（InitConfig 播种 / 后台表单兜底共用）
func defaultAccountChangeConfig() facade.H {
	return facade.H{
		// enabled 与 value 同步写：value 沿用其它开关的 "0"/"1" 口径，json.enabled 便于整体读取
		"enabled":         0,
		"cooldown_days":   AccountCooldownDefault,
		"require_contact": 1,
		"require_verify":  1,
	}
}

// AccountChangeConfig - 读取账号修改规则（逐项兜底）
//
// 老库升级后 json 里可能还没有新字段，缺什么按默认值处理，与后台表单的默认值保持一致；
// 与 register.go 一样直接读库（配置量极小、且改动后要立即生效，不额外做缓存）。
func AccountChangeConfig() AccountChangeSetting {

	setting := AccountChangeSetting{
		Enabled:        false,
		CooldownDays:   AccountCooldownDefault,
		RequireContact: true,
		RequireVerify:  true,
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", AccountChangeConfigKey).Find()
	if utils.Is.Empty(item) {
		return setting
	}

	json := asStringMap(item["json"])
	if len(json) == 0 {
		if raw := cast.ToString(item["json"]); !utils.Is.Empty(raw) {
			json = asStringMap(utils.Json.Decode(raw))
		}
	}

	// 总开关：json.enabled 优先（后台保存时两处同步写，取 json 更贴近表单所见）
	if _, exist := json["enabled"]; exist {
		setting.Enabled = cast.ToBool(json["enabled"])
	} else {
		setting.Enabled = cast.ToBool(item["value"])
	}

	if _, exist := json["cooldown_days"]; exist {
		setting.CooldownDays = cast.ToInt(json["cooldown_days"])
	}
	if _, exist := json["require_contact"]; exist {
		setting.RequireContact = cast.ToBool(json["require_contact"])
	}
	if _, exist := json["require_verify"]; exist {
		setting.RequireVerify = cast.ToBool(json["require_verify"])
	}

	// 负数没有意义（后台已限制 0-365，这里再兜一次）
	if setting.CooldownDays < 0 {
		setting.CooldownDays = 0
	}

	return setting
}
