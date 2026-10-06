package middleware

import (
	"fmt"
	"net/http/httputil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
)

// ============================== 日志 / 请求导出脱敏 ==============================
//
// 背景（P0 安全问题）：请求日志中间件会把 POST body 解析出的参数原样写进
// runtime/logs，于是「账号 + 明文密码」以及注册 / 验证码接口的手机号字段
// （social）都以明文落盘。日志文件一旦被越权读取或误传，等于直接泄露用户凭据。
//
// 处理口径（与 model/privacy.go 的分级脱敏保持一致）：
//   - 凭据类字段（password / token / secret / 验证码…）：整值替换为 ***，
//     一个字符都不保留 —— 密码、密钥没有任何「保留前后几位」的正当性；
//   - PII 类字段（phone / email / account / social…）：按内容掩码，
//     手机号 → 180****0738，邮箱 → zh***@qq.com，其它 → 首尾各留一位；
//   - 递归处理嵌套 map / slice（params 里可能存在 where 之类的嵌套对象），
//     但限制深度与元素数量，避免异常构造的参数把日志写入拖慢。
//
// 注意：这里只影响「写日志」，业务逻辑拿到的仍是原始参数。

// logSensitiveKeywords - 命中即整值打码的字段关键字（按子串匹配，大小写不敏感）
//
// 用子串而不是全等，是因为同一个含义会出现多种命名：
// old_password / new_password / confirmPassword / userPwd / db_password …
var logSensitiveKeywords = []string{
	"password", "passwd", "pwd",
	"token", "secret", "privatekey", "private_key", "apikey", "api_key",
	"credential", "authorization", "signature",
	"captcha", "verifycode", "verify_code", "vcode", "smscode", "sms_code",
}

// logSensitiveExactKeys - 需要整值打码的「短字段名」（全等匹配，避免误伤）
//
// code：短信 / 邮箱验证码，泄漏后可在有效期内被重放。
// auth：部分接口用它传裸 token。
var logSensitiveExactKeys = []string{
	"code", "auth", "sign", "key", "pwd",
}

// logPIIKeywords - 命中即按内容掩码的字段关键字（手机号 / 邮箱 / 账号等）
var logPIIKeywords = []string{
	"phone", "mobile", "telephone", "tel",
	"email", "mail",
	"account", "userid", "user_id", "social", "contact",
}

const (
	// 递归深度上限（params 正常只有 1~2 层）
	logRedactMaxDepth = 6
	// 单层 map / slice 处理元素上限，防止超大 body 撑爆日志
	logRedactMaxItems = 50
)

// SanitizeLogParams - 脱敏请求参数（供日志与异常导出使用）
//
// 返回一份新的结构，不修改调用方传入的原始参数（业务逻辑继续用原值）。
func SanitizeLogParams(params any) any {
	return redactValue("", params, 0)
}

// redactValue - 递归脱敏
func redactValue(key string, value any, depth int) any {
	if value == nil {
		return nil
	}
	if depth > logRedactMaxDepth {
		return "..."
	}

	// 敏感字段直接整值打码，不再往下钻（避免 password 是个对象时漏掉）
	if isSensitiveKey(key) {
		return "***"
	}

	switch val := value.(type) {

	case map[string]any:
		result := make(map[string]any, len(val))
		index := 0
		for childKey, childVal := range val {
			if index >= logRedactMaxItems {
				result["_truncated"] = "..."
				break
			}
			result[childKey] = redactValue(childKey, childVal, depth+1)
			index++
		}
		return result

	case []any:
		result := make([]any, 0, len(val))
		for index, childVal := range val {
			if index >= logRedactMaxItems {
				result = append(result, "...")
				break
			}
			result = append(result, redactValue(key, childVal, depth+1))
		}
		return result

	case []string:
		result := make([]any, 0, len(val))
		for index, childVal := range val {
			if index >= logRedactMaxItems {
				result = append(result, "...")
				break
			}
			result = append(result, redactValue(key, childVal, depth+1))
		}
		return result

	case string:
		// PII 字段按内容掩码；其余字符串原样返回
		if isPIIKey(key) {
			return maskPII(val)
		}
		return val

	default:
		// 数字 / 布尔等：PII 字段也可能是数字（如纯数字手机号），一并按字符串掩码
		if isPIIKey(key) {
			return maskPII(castToText(val))
		}
		return val
	}
}

// isSensitiveKey - 是否为凭据类字段
func isSensitiveKey(key string) bool {
	name := normalizeKey(key)
	if name == "" {
		return false
	}
	for _, item := range logSensitiveExactKeys {
		if name == item {
			return true
		}
	}
	for _, item := range logSensitiveKeywords {
		if strings.Contains(name, item) {
			return true
		}
	}
	return false
}

// isPIIKey - 是否为 PII（个人信息）字段
func isPIIKey(key string) bool {
	name := normalizeKey(key)
	if name == "" {
		return false
	}
	for _, item := range logPIIKeywords {
		if strings.Contains(name, item) {
			return true
		}
	}
	return false
}

// normalizeKey - 字段名归一化：小写 + 去掉分隔符，便于子串匹配
//
// old_password / oldPassword / old-password 归一化后都是 oldpassword。
func normalizeKey(key string) string {
	if key == "" {
		return ""
	}
	name := strings.ToLower(strings.TrimSpace(key))
	name = strings.NewReplacer("_", "", "-", "", " ", "", ".", "").Replace(name)
	return name
}

// maskPII - 按内容掩码：手机号 / 邮箱用框架统一规则，其它保留首尾各一位
func maskPII(value string) string {
	if value == "" {
		return value
	}
	if facade.Comm.ValidPhone(value) {
		return facade.Comm.MaskPhone(value)
	}
	if utils.Is.Email(value) {
		return facade.Comm.MaskEmail(value)
	}
	return maskPartial(value)
}

// maskPartial - 通用部分掩码：保留首尾字符（单字符则全掩码）
func maskPartial(value string) string {
	runes := []rune(value)
	switch {
	case len(runes) > 2:
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	case len(runes) == 2:
		return string(runes[0]) + "*"
	case len(runes) == 1:
		return "*"
	}
	return value
}

// castToText - 非字符串值转文本（仅用于 PII 掩码）
func castToText(value any) string {
	switch val := value.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	}
	return fmt.Sprintf("%v", value)
}

// dumpRequestSanitized - 导出请求（仅请求行 + 头部）时抹掉凭据类头部
//
// panic 时的 request 导出原本会把 Authorization（裸 JWT）与 Cookie 一起写进
// 错误日志，等价于把可用登录凭据落盘；这里复制一份请求再抹掉这些头部。
func dumpRequestSanitized(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}

	clone := ctx.Request.Clone(ctx.Request.Context())
	clone.Header = ctx.Request.Header.Clone()

	for _, key := range []string{"Authorization", "Cookie", "X-Token", "X-Api-Key"} {
		if clone.Header.Get(key) == "" {
			continue
		}
		clone.Header.Set(key, "***")
	}

	// 第二个参数 false：不导出 body（body 可能含密码等敏感内容）
	dump, err := httputil.DumpRequest(clone, false)
	if err != nil {
		return ""
	}
	return string(dump)
}
