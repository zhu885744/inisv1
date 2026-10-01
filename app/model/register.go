package model

import (
	"errors"
	"fmt"
	"strings"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * 注册扩展设置
 *
 * 存放位置：config 表 ALLOW_REGISTER 记录（同一条记录承载三部分数据）
 * - value：是否允许自行注册（"0"/"1"）
 * - text ：新用户默认权限组 ID 列表，如 "1,2"
 * - json ：本文件定义的扩展设置（邮箱域名限制 / 注册验证方式 / 欢迎消息开关）
 *
 * 前端入口：/admin/system?tab=register
 *
 * 关于「邮箱验证」：本站注册本身就要求邮箱/手机验证码（见 comm/register），
 * 邮箱所有权在注册那一步已经证明，因此不再提供「注册后再点邮件链接验证」的二次验证。
 */

// 邮箱域名限制模式
const (
	RegisterDomainOff       = "off"       // 关闭
	RegisterDomainWhitelist = "whitelist" // 白名单：仅名单内域名可注册
	RegisterDomainBlacklist = "blacklist" // 黑名单：名单内域名禁止注册
)

// 注册验证方式
const (
	RegisterVerifyNone   = "none"   // 无：注册完成即登录
	RegisterVerifyManual = "manual" // 人工审核：账号先进入待审核，管理员通过后才能登录
)

// RegisterSetting - 注册扩展设置
type RegisterSetting struct {
	// 邮箱域名限制：off / whitelist / blacklist
	EmailDomainMode string `json:"email_domain_mode"`
	// 允许注册的邮箱域名（如 qq.com，保存时已去掉 @ 前缀）
	EmailWhitelist []string `json:"email_whitelist"`
	// 禁止注册的邮箱域名
	EmailBlacklist []string `json:"email_blacklist"`
	// 注册验证方式：none / manual
	// 历史配置里可能残留 "email"（已废弃），RegisterSettings 会将其降级为 none
	VerifyMode string `json:"verify_mode"`
	// 注册成功后发送站内欢迎消息
	WelcomeMessage bool `json:"welcome_message"`
	// 注册成功后发送欢迎邮件
	WelcomeEmail bool `json:"welcome_email"`
}

// RegisterSettings - 读取注册扩展设置（字段缺失时用默认值兜底）
func RegisterSettings() RegisterSetting {

	setting := RegisterSetting{
		EmailDomainMode: RegisterDomainOff,
		EmailWhitelist:  []string{},
		EmailBlacklist:  []string{},
		VerifyMode:      RegisterVerifyNone,
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", "ALLOW_REGISTER").Find()
	if utils.Is.Empty(item) {
		return setting
	}

	jsonMap := ParseConfigJson(item["json"])

	switch mode := cast.ToString(jsonMap["email_domain_mode"]); mode {
	case RegisterDomainWhitelist, RegisterDomainBlacklist:
		setting.EmailDomainMode = mode
	}
	// 只认 manual：历史配置里残留的 "email"（已废弃的邮箱验证模式）会被降级为 none
	if mode := cast.ToString(jsonMap["verify_mode"]); mode == RegisterVerifyManual {
		setting.VerifyMode = mode
	}

	setting.EmailWhitelist = NormalizeDomains(jsonMap["email_whitelist"])
	setting.EmailBlacklist = NormalizeDomains(jsonMap["email_blacklist"])
	setting.WelcomeMessage = cast.ToBool(jsonMap["welcome_message"])
	setting.WelcomeEmail = cast.ToBool(jsonMap["welcome_email"])

	return setting
}

// ParseConfigJson - 解析 config 的 json 字段（兼容「JSON 字符串」与「已解码的 map」两种形态）
func ParseConfigJson(raw any) map[string]any {
	switch val := raw.(type) {
	case map[string]any:
		return val
	case string:
		if decoded := utils.Json.Decode(val); decoded != nil {
			if item, ok := decoded.(map[string]any); ok {
				return item
			}
		}
	default:
		if decoded := utils.Json.Decode(cast.ToString(raw)); decoded != nil {
			if item, ok := decoded.(map[string]any); ok {
				return item
			}
		}
	}
	return map[string]any{}
}

// NormalizeDomains - 域名列表归一化：支持数组 / 多行文本 / 逗号分隔，去掉 @ 前缀并统一小写去重
func NormalizeDomains(raw any) []string {

	list := make([]string, 0)

	switch val := raw.(type) {
	case []string:
		list = append(list, val...)
	case []any:
		for _, item := range val {
			list = append(list, cast.ToString(item))
		}
	case string:
		list = append(list, strings.FieldsFunc(val, func(r rune) bool {
			return r == '\n' || r == '\r' || r == ',' || r == '，' || r == ';' || r == '；' || r == ' ' || r == '\t'
		})...)
	default:
		if !utils.Is.Empty(raw) {
			list = append(list, cast.ToString(raw))
		}
	}

	result := make([]string, 0, len(list))
	exist := map[string]bool{}
	for _, item := range list {
		domain := strings.ToLower(strings.TrimSpace(item))
		domain = strings.TrimPrefix(domain, "@")
		domain = strings.Trim(domain, ".")
		if domain == "" || exist[domain] {
			continue
		}
		exist[domain] = true
		result = append(result, domain)
	}

	return result
}

// EmailDomainOf - 取邮箱的域名部分（小写，无 @ 前缀）
func EmailDomainOf(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	index := strings.LastIndex(email, "@")
	if index < 0 || index == len(email)-1 {
		return ""
	}
	return email[index+1:]
}

// matchDomain - 域名匹配：完全一致或为其子域（允许 a.qq.com 命中名单里的 qq.com）
func matchDomain(domain string, rule string) bool {
	if domain == rule {
		return true
	}
	return strings.HasSuffix(domain, "."+rule)
}

// CheckEmailDomain - 邮箱域名限制校验
// 模式为「关闭」或地址不是邮箱时直接放行；命中限制时返回错误信息
func CheckEmailDomain(setting RegisterSetting, email string) error {

	mode := setting.EmailDomainMode
	if mode != RegisterDomainWhitelist && mode != RegisterDomainBlacklist {
		return nil
	}

	domain := EmailDomainOf(email)
	if domain == "" {
		return nil
	}

	switch mode {
	case RegisterDomainWhitelist:
		for _, rule := range setting.EmailWhitelist {
			if matchDomain(domain, rule) {
				return nil
			}
		}
		return errors.New("该邮箱域名不在允许注册的名单内！")
	case RegisterDomainBlacklist:
		for _, rule := range setting.EmailBlacklist {
			if matchDomain(domain, rule) {
				return errors.New("该邮箱域名已被禁止注册！")
			}
		}
	}

	return nil
}

// SiteTitle - 站点标题（取前台「网站设置」中的 title，缺省用「本站」）
func SiteTitle() string {
	item, _ := facade.DB.Model(&Config{}).Where("key", "Mellow_functions").Find()
	if utils.Is.Empty(item) {
		return "本站"
	}
	title := cast.ToString(ParseConfigJson(item["json"])["title"])
	if utils.Is.Empty(title) {
		return "本站"
	}
	return title
}

// SendWelcome - 注册成功（或人工审核通过）后发送欢迎消息与欢迎邮件
// 两个开关都关闭时不产生任何请求
//
// account / nickname 会写进正文：涉及用户的邮件统一带上「账号 / 昵称」，
// 方便用户确认这封邮件属于哪个账号。
func SendWelcome(uid int, account string, nickname string, email string) {

	setting := RegisterSettings()
	if !setting.WelcomeMessage && !setting.WelcomeEmail {
		return
	}

	site := SiteTitle()
	if utils.Is.Empty(nickname) {
		nickname = "朋友"
	}
	if utils.Is.Empty(account) {
		account = "—"
	}

	title := fmt.Sprintf("欢迎加入 %v", site)
	content := fmt.Sprintf("%v，您好：\n\n您的账号：%v\n\n感谢您注册 %v，祝您使用愉快！如有疑问可通过站内联系方式联系我们。", nickname, account, site)

	if setting.WelcomeMessage {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					facade.Log.Error(map[string]any{"error": err}, "发送注册欢迎消息时发生panic")
				}
			}()

			note := &Notification{}
			if _, err := note.CreateNotification(uid, 0, NotificationTypeSystem, title, content, "", 0); err != nil {
				facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid}, "发送注册欢迎消息失败")
			}
		}()
	}

	if setting.WelcomeEmail && utils.Is.Email(email) {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					facade.Log.Error(map[string]any{"error": err}, "发送注册欢迎邮件时发生panic")
				}
			}()

			if response := facade.SendMail(email, title, content); response != nil && response.Error != nil {
				facade.Log.Error(map[string]any{"error": response.Error.Error(), "uid": uid}, "发送注册欢迎邮件失败")
			}
		}()
	}
}
