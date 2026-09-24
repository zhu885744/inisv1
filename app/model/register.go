package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
 */

// 邮箱域名限制模式
const (
	RegisterDomainOff       = "off"       // 关闭
	RegisterDomainWhitelist = "whitelist" // 白名单：仅名单内域名可注册
	RegisterDomainBlacklist = "blacklist" // 黑名单：名单内域名禁止注册
)

// 注册验证方式
const (
	RegisterVerifyNone   = "none"   // 无：直接注册成功
	RegisterVerifyEmail  = "email"  // Email 验证：发验证邮件，验证通过后才能登录
	RegisterVerifyManual = "manual" // 人工审核：账号先进入待审核，管理员通过后才能登录
)

// 邮箱验证邮件有效期
const RegisterMailTokenExpire = 24 * time.Hour

// 邮箱验证相关缓存键
const (
	registerMailTokenKey = "[register][email-verify][%v]" // token → uid
	registerMailLimitKey = "[register][email-verify-limit][%v]"
)

// RegisterSetting - 注册扩展设置
type RegisterSetting struct {
	// 邮箱域名限制：off / whitelist / blacklist
	EmailDomainMode string `json:"email_domain_mode"`
	// 允许注册的邮箱域名（如 qq.com，保存时已去掉 @ 前缀）
	EmailWhitelist []string `json:"email_whitelist"`
	// 禁止注册的邮箱域名
	EmailBlacklist []string `json:"email_blacklist"`
	// 注册验证方式：none / email / manual
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
	switch mode := cast.ToString(jsonMap["verify_mode"]); mode {
	case RegisterVerifyEmail, RegisterVerifyManual:
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

// MarkEmailUnverified - 标记邮箱未验证（注册时按「Email 验证」模式写入）
func MarkEmailUnverified(uid any) error {
	return UpdateUserJson(uid, map[string]any{UserJsonEmailVerified: 0})
}

// MarkEmailVerified - 标记邮箱已验证
func MarkEmailVerified(uid any) error {
	return UpdateUserJson(uid, map[string]any{UserJsonEmailVerified: 1})
}

// CreateMailToken - 生成邮箱验证 token（24 小时有效，可重发覆盖）
func CreateMailToken(uid int) string {
	token := utils.Rand.String(32, "abcdefghijklmnopqrstuvwxyz0123456789")
	facade.Cache.Set(fmt.Sprintf(registerMailTokenKey, token), uid, RegisterMailTokenExpire)
	return token
}

// ConsumeMailToken - 消费邮箱验证 token：有效则返回 uid 并立即失效（一次性）
func ConsumeMailToken(token string) int {
	if utils.Is.Empty(token) {
		return 0
	}
	key := fmt.Sprintf(registerMailTokenKey, token)
	uid := cast.ToInt(facade.Cache.Get(key))
	if uid > 0 {
		facade.Cache.Del(key)
	}
	return uid
}

// mailLimit - 发信频控：同一账号 60 秒内只允许发送一次验证邮件
func mailLimit(uid int) error {
	key := fmt.Sprintf(registerMailLimitKey, uid)
	if !utils.Is.Empty(facade.Cache.Get(key)) {
		return errors.New("验证邮件发送过于频繁，请稍后再试！")
	}
	facade.Cache.Set(key, 1, time.Minute)
	return nil
}

// SendRegisterVerifyMail - 发送注册邮箱验证邮件（链接指向前台 /auth/verify?token=xxx）
func SendRegisterVerifyMail(uid int, email string, baseURL string) error {

	if !utils.Is.Email(email) {
		return errors.New("邮箱格式不正确，无法发送验证邮件！")
	}

	if err := mailLimit(uid); err != nil {
		return err
	}

	token := CreateMailToken(uid)
	site := SiteTitle()
	link := fmt.Sprintf("%v/auth/verify?token=%v", strings.TrimRight(baseURL, "/"), token)

	subject := fmt.Sprintf("%v：请验证您的注册邮箱", site)
	content := fmt.Sprintf(
		"您好：\n\n感谢注册 %v，请点击下面的链接完成邮箱验证：\n%v\n\n链接 24 小时内有效；如果不是您本人的操作，忽略本邮件即可。",
		site, link,
	)

	if response := facade.SendMail(email, subject, content); response != nil && response.Error != nil {
		facade.Log.Error(map[string]any{"error": response.Error.Error(), "uid": uid}, "发送注册验证邮件失败")
		return errors.New("验证邮件发送失败，请联系管理员检查邮件服务配置！")
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

// SendWelcome - 注册成功（或审核通过 / 邮箱验证通过）后发送欢迎消息与欢迎邮件
// 两个开关都关闭时不产生任何请求
func SendWelcome(uid int, nickname string, email string) {

	setting := RegisterSettings()
	if !setting.WelcomeMessage && !setting.WelcomeEmail {
		return
	}

	site := SiteTitle()
	if utils.Is.Empty(nickname) {
		nickname = "朋友"
	}

	title := fmt.Sprintf("欢迎加入 %v", site)
	content := fmt.Sprintf("%v，您好：\n\n感谢您注册 %v，祝您使用愉快！如有疑问可通过站内联系方式联系我们。", nickname, site)

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
