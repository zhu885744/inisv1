package model

import (
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
)

// ============================== 滑块验证配置（后台可配） ==============================
//
// 配置存在 config 表的一条记录里（key = SYSTEM_CAPTCHA，json 存各场景开关），
// 由后台「系统配置 → 安全 → 安全验证（滑块）」维护，路径：/api/config/save。
//
// 口径：
//   - 默认只开「发送验证码」防刷（同一手机号/邮箱 5 分钟内重复发送要过滑块）；
//   - 登录 / 注册 / 改密 / 改绑 / 注销这些敏感操作默认**关闭**，
//     把「是否牺牲一点顺畅度换安全」交给站长按站点情况决定；
//   - 修改后立即生效：config/save 会清掉 config[SYSTEM_CAPTCHA] 缓存。

// CaptchaConfigKey - 滑块验证配置在 config 表的键
const CaptchaConfigKey = "SYSTEM_CAPTCHA"

// 滑块验证场景
const (
	// CaptchaSceneSendCode 发送验证码：同一目标 5 分钟内的第二次发送要求滑块
	CaptchaSceneSendCode = "send_code"
	// CaptchaSceneLogin 密码登录
	CaptchaSceneLogin = "login"
	// CaptchaSceneCodeLogin 验证码登录（提交验证码时）
	CaptchaSceneCodeLogin = "code_login"
	// CaptchaSceneRegister 注册（提交验证码时）
	CaptchaSceneRegister = "register"
	// CaptchaSceneResetPassword 重置 / 找回密码（提交新密码时）
	CaptchaSceneResetPassword = "reset_password"
	// CaptchaSceneChangeContact 修改邮箱 / 手机号（提交验证码时）
	CaptchaSceneChangeContact = "change_contact"
	// CaptchaSceneDestroy 注销账户（提交验证码时）
	CaptchaSceneDestroy = "destroy"
)

// captchaSceneList - 全部场景（顺序与后台展示一致）
var captchaSceneList = []string{
	CaptchaSceneSendCode,
	CaptchaSceneLogin,
	CaptchaSceneCodeLogin,
	CaptchaSceneRegister,
	CaptchaSceneResetPassword,
	CaptchaSceneChangeContact,
	CaptchaSceneDestroy,
}

// defaultCaptchaSettings - 默认配置：仅「发送验证码」防刷开启
func defaultCaptchaSettings() facade.H {
	settings := facade.H{}
	for _, scene := range captchaSceneList {
		settings[scene] = 0
	}
	settings[CaptchaSceneSendCode] = 1
	return settings
}

// CaptchaSettings - 读取滑块验证配置（缓存优先，缺失项用默认值兜底）
//
// 缓存名与 config/save 清理的名字保持一致（config[KEY]），否则后台改完要等重启才生效。
func CaptchaSettings() facade.H {
	settings := defaultCaptchaSettings()

	cacheName := "config[" + CaptchaConfigKey + "]"

	if facade.Cache.Has(cacheName) {
		if data, ok := facade.Cache.Get(cacheName).(map[string]any); ok {
			return mergeCaptchaSettings(settings, data)
		}
		if data, ok := facade.Cache.Get(cacheName).(facade.H); ok {
			return mergeCaptchaSettings(settings, data)
		}
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", CaptchaConfigKey).Find()
	if !utils.Is.Empty(item) {
		if data, ok := item["json"].(map[string]any); ok {
			settings = mergeCaptchaSettings(settings, data)
			facade.Cache.Set(cacheName, settings)
		}
	}

	return settings
}

// CaptchaSceneEnabled - 指定场景是否需要滑块验证
func CaptchaSceneEnabled(scene string) bool {
	if utils.Is.Empty(scene) {
		return false
	}
	return cast.ToInt(CaptchaSettings()[scene]) == 1
}

// mergeCaptchaSettings - 用配置值覆盖默认值（只认已知场景，避免脏数据影响判断）
func mergeCaptchaSettings(base facade.H, override map[string]any) facade.H {
	for _, scene := range captchaSceneList {
		value, ok := override[scene]
		if !ok {
			continue
		}
		base[scene] = captchaSwitchValue(value)
	}
	return base
}

// captchaSwitchValue - 开关值归一化为 0/1
//
// 兼容三种来源：前端下拉的 "0"/"1" 字符串、数字、以及手写 JSON 时的布尔值。
func captchaSwitchValue(value any) int {
	if flag, ok := value.(bool); ok {
		if flag {
			return 1
		}
		return 0
	}
	if cast.ToInt(value) == 1 {
		return 1
	}
	return 0
}
