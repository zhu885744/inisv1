package facade

import (
	"errors"
	"net"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/shirou/gopsutil/mem"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 全局安全策略（单例，高性能）
var htmlSanitizer = bluemonday.UGCPolicy()

// XSS 检测正则（预编译，提升性能）
var (
	scriptRegex    = regexp.MustCompile(`(?i)<script\b[^>]*>[\s\S]*?</script>`)
	eventRegex     = regexp.MustCompile(`(?i)\bon(click|mouse|key|load|error|submit|change|focus|blur|scroll|resize|select|abort|afterprint|beforeprint|beforeunload|canplay|canplaythrough|cuechange|dblclick|drag|dragend|dragenter|dragleave|dragover|dragstart|drop|durationchange|emptied|ended|error|focusin|focusout|hashchange|input|invalid|keydown|keypress|keyup|load|loadeddata|loadedmetadata|loadstart|message|mousedown|mousemove|mouseout|mouseover|mouseup|mousewheel|offline|online|open|pagehide|pageshow|play|playing|popstate|progress|ratechange|readystatechange|reset|resize|scroll|seeked|seeking|select|show|stalled|storage|submit|suspend|timeupdate|toggle|touchcancel|touchend|touchmove|touchstart|transitionend|unload|volumechange|waiting|wheel)\b[^=]*=\s*(?:"[^"]*"|'[^']*'|[^'"<>]+)`)
	jsUrlRegex     = regexp.MustCompile(`(?i)javascript:\s*[\s\S]*`)
	iframeRegex    = regexp.MustCompile(`(?i)<iframe\b[^>]*>[\s\S]*?</iframe>`)
	objectRegex    = regexp.MustCompile(`(?i)<object\b[^>]*>[\s\S]*?</object>`)
	embedRegex     = regexp.MustCompile(`(?i)<embed\b[^>]*>`)
	evalRegex      = regexp.MustCompile(`(?i)\beval\s*\(\s*[\s\S]*?\)`)
	timerRegex     = regexp.MustCompile(`(?i)\b(setTimeout|setInterval)\s*\(\s*[\s\S]*?\)`)
	styleRegex     = regexp.MustCompile(`(?i)<style\b[^>]*>[\s\S]*?</style>`)
	svgRegex       = regexp.MustCompile(`(?i)<svg\b[^>]*>[\s\S]*?</svg>`)
	imgRegex       = regexp.MustCompile(`(?i)<img\b[^>]*on\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^'"<>]+)`)
	linkRegex      = regexp.MustCompile(`(?i)<link\b[^>]*href\s*=\s*["']?javascript:`)
	dataUrlRegex   = regexp.MustCompile(`(?i)data:text/html[^>]*`)
	base64Regex    = regexp.MustCompile(`(?i)base64[^>]*<[^>]+>`)
	entityRegex    = regexp.MustCompile(`(?i)&[a-z0-9]+;`)
	hexEntityRegex = regexp.MustCompile(`&#x([0-9a-fA-F]+);`)
	decEntityRegex = regexp.MustCompile(`&#([0-9]+);`)
)

type CommStruct struct{}

// 全局单例
var Comm = &CommStruct{}

// Sn 获取机器唯一序列号
func (c *CommStruct) Sn() string {
	mac := utils.Get.Mac()
	machineID, err := machineid.ID()
	if err != nil {
		machineID = mac
	}
	return utils.Hash.Token(machineID, 32, mac)
}

// Device 上报设备信息
func (c *CommStruct) Device() *utils.CurlResponse {
	var memoryInfo string
	vm, err := mem.VirtualMemory()
	if err == nil {
		memory := map[string]any{
			"free":  vm.Free,
			"used":  vm.Used,
			"total": vm.Total,
		}
		memoryInfo = utils.Json.String(memory)
	}

	body := map[string]any{
		"sn":     c.Sn(),
		"mac":    utils.Get.Mac(),
		"port":   map[string]any{"run": Var.Get("port"), "real": AppToml.Get("app.port")},
		"memory": memoryInfo,
		"domain": Var.Get("domain"),
		"goos":   runtime.GOOS,
		"goarch": runtime.GOARCH,
		"cpu":    runtime.NumCPU(),
	}

	unix := time.Now().Unix()
	headers := c.generateSecureHeaders(body, unix)

	return utils.Curl(utils.CurlRequest{
		Method:  "POST",
		Url:     Uri + "/dev/device/record",
		Body:    body,
		Headers: headers,
	}).Send()
}

// Signature 生成接口签名
func (c *CommStruct) Signature(params map[string]any) map[string]any {
	port := AppToml.Get("app.port")
	unix := time.Now().Unix()

	encryptData := map[string]any{
		"sn":   c.Sn(),
		"port": port,
		"mac":  utils.Get.Mac(),
	}

	key := utils.Hash.Token(port, 16, "AesKey")
	iv := utils.Hash.Token(unix, 16, "AesIv")
	argus := utils.AES(key, iv).Encrypt(utils.Json.Encode(encryptData)).Text

	gorgon := cast.ToString(port) + utils.Hash.Token(c.Sn(), 48, unix)
	stub := strings.ToUpper(utils.Hash.Token(utils.Map.ToURL(params), 32, unix))

	return map[string]any{
		"X-Khronos": unix,
		"X-Argus":   argus,
		"X-Gorgon":  gorgon,
		"X-SS-STUB": stub,
	}
}

// WithField 白名单保留字段
func (c *CommStruct) WithField(data map[string]any, field any) map[string]any {
	keys := cast.ToStringSlice(utils.Unity.Keys(field))
	if utils.Is.Empty(keys) {
		return data
	}
	return utils.Map.WithField(data, keys)
}

// MaskEmail 邮箱脱敏处理
func (c *CommStruct) MaskEmail(email string) string {
	if email == "" || !utils.Is.Email(email) {
		return email
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	prefix := parts[0]
	domain := parts[1]

	prefixLen := len(prefix)
	if prefixLen <= 2 {
		return email
	}

	maskedPrefix := prefix[:2] + strings.Repeat("*", prefixLen-2)
	return maskedPrefix + "@" + domain
}

// MaskPhone 手机号脱敏处理
func (c *CommStruct) MaskPhone(phone string) string {
	if phone == "" {
		return phone
	}
	phone = strings.ReplaceAll(strings.ReplaceAll(phone, " ", ""), "-", "")
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskIP IP地址脱敏处理
func (c *CommStruct) MaskIP(ip string) string {
	if ip == "" {
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return parts[0] + "." + parts[1] + ".*.*"
}

// MaskSecret 密钥/密码脱敏处理（保留首尾各 4 位，中间隐藏）
func (c *CommStruct) MaskSecret(secret string) string {
	if secret == "" {
		return secret
	}

	runes := []rune(secret)
	length := len(runes)

	// 长度不足 8 位时，全部隐藏，避免泄露短密钥（如简单密码）
	if length <= 8 {
		return strings.Repeat("*", length)
	}

	// 保留前 4 位和后 4 位
	return string(runes[:4]) + "****" + string(runes[length-4:])
}

// MaskUA User-Agent脱敏处理
func (c *CommStruct) MaskUA(ua string) string {
	if ua == "" {
		return ua
	}
	if len(ua) <= 50 {
		return ua
	}
	return ua[:50] + "..."
}

// SanitizeHTML 彻底防御 XSS（核心方法）
func (c *CommStruct) SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}
	// 专业 XSS 过滤（使用 bluemonday）
	clean := htmlSanitizer.Sanitize(input)
	// 清理空白
	return strings.TrimSpace(clean)
}

// DetectXSS 高精度 XSS 检测（防绕过）
func (c *CommStruct) DetectXSS(input string) bool {
	if input == "" {
		return false
	}

	// 先解码所有实体编码（防编码绕过）
	input = c.decodeHTMLEntities(input)

	// 高危特征检测
	if scriptRegex.MatchString(input) ||
		eventRegex.MatchString(input) ||
		jsUrlRegex.MatchString(input) ||
		iframeRegex.MatchString(input) ||
		objectRegex.MatchString(input) ||
		embedRegex.MatchString(input) ||
		evalRegex.MatchString(input) ||
		timerRegex.MatchString(input) ||
		styleRegex.MatchString(input) ||
		svgRegex.MatchString(input) ||
		imgRegex.MatchString(input) ||
		linkRegex.MatchString(input) ||
		dataUrlRegex.MatchString(input) ||
		base64Regex.MatchString(input) {
		return true
	}

	return false
}

// decodeHTMLEntities 完整 HTML 实体解码（防编码绕过）
func (c *CommStruct) decodeHTMLEntities(input string) string {
	// 基础实体
	replacements := map[string]string{
		"&lt;":     "<",
		"&gt;":     ">",
		"&amp;":    "&",
		"&quot;":   "\"",
		"&apos;":   "'",
		"&nbsp;":   " ",
		"&iexcl;":  "¡",
		"&cent;":   "¢",
		"&pound;":  "£",
		"&curren;": "¤",
		"&yen;":    "¥",
		"&brvbar;": "¦",
		"&sect;":   "§",
		"&uml;":    "¨",
		"&copy;":   "©",
		"&ordf;":   "ª",
		"&laquo;":  "«",
		"&not;":    "¬",
		"&reg;":    "®",
		"&macr;":   "¯",
		"&deg;":    "°",
		"&plusmn;": "±",
		"&sup2;":   "²",
		"&sup3;":   "³",
		"&acute;":  "´",
		"&micro;":  "µ",
		"&para;":   "¶",
		"&middot;": "·",
		"&cedil;":  "¸",
		"&sup1;":   "¹",
		"&ordm;":   "º",
		"&raquo;":  "»",
		"&frac14;": "¼",
		"&frac12;": "½",
		"&frac34;": "¾",
		"&iquest;": "¿",
	}
	for k, v := range replacements {
		input = strings.ReplaceAll(input, k, v)
	}

	// 十六进制实体
	input = hexEntityRegex.ReplaceAllStringFunc(input, func(m string) string {
		hex := m[3 : len(m)-1]
		if val, err := strconv.ParseUint(hex, 16, 32); err == nil {
			return string(rune(val))
		}
		return m
	})

	// 十进制实体
	input = decEntityRegex.ReplaceAllStringFunc(input, func(m string) string {
		dec := m[2 : len(m)-1]
		if val, err := strconv.Atoi(dec); err == nil {
			return string(rune(val))
		}
		return m
	})

	return input
}

// generateSecureHeaders 统一生成安全请求头
func (c *CommStruct) generateSecureHeaders(body map[string]any, unix int64) map[string]any {
	sn := cast.ToString(body["sn"])
	mac := cast.ToString(body["mac"])

	key := utils.Hash.Token(sn, 16, Token)
	iv := utils.Hash.Token(mac, 16, Token)
	aes := utils.AES(key, iv)

	argus := aes.Encrypt(utils.Json.Encode(body)).Text
	gorgon := "8642" + utils.Hash.Token(sn, 48, unix)
	stub := strings.ToUpper(utils.Hash.Token(utils.Map.ToURL(body), 32, unix))

	return map[string]any{
		"X-Khronos": unix,
		"X-Argus":   argus,
		"X-Gorgon":  gorgon,
		"X-SS-STUB": stub,
	}
}

// IsPrivateIP 判断 IP 是否为内网/回环/链路本地/保留地址，用于 SSRF 防护
func (c *CommStruct) IsPrivateIP(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		// 解析失败（可能是 IPv6 带 zone 等），保守起见视为内网拦截
		return true
	}

	// IPv4 内网/保留地址段
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.IsLoopback() ||
			ip4.IsPrivate() ||
			ip4.IsLinkLocalUnicast() ||
			ip4.IsLinkLocalMulticast() ||
			ip4.IsUnspecified() ||
			ip4.IsMulticast()
	}

	// IPv6 内网/回环/链路本地
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

// IsSafeOutboundURL SSRF 防护：校验目标 URL 是否允许对外发起请求
// 仅允许 http/https 协议，且目标主机不能解析到内网地址
func (c *CommStruct) IsSafeOutboundURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if utils.Is.Empty(rawURL) {
		return errors.New("目标地址不能为空！")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("目标地址格式错误！")
	}

	// 协议白名单
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("仅允许 http/https 协议！")
	}

	// 主机名不能为空
	host := u.Hostname()
	if utils.Is.Empty(host) {
		return errors.New("目标地址缺少主机名！")
	}

	// 解析主机名对应的所有 IP（防止 DNS rebinding 与十六进制/整数形式的 IP 绕过）
	ips, err := net.LookupIP(host)
	if err != nil {
		return errors.New("目标主机无法解析！")
	}

	for _, ip := range ips {
		if c.IsPrivateIP(ip.String()) {
			return errors.New("禁止访问内网地址！")
		}
	}

	return nil
}
