package facade

import (
	"html"
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
	eventRegex     = regexp.MustCompile(`(?i)\bon\w+\s*=\s*(?:["']|)[\s\S]*?`)
	jsUrlRegex     = regexp.MustCompile(`(?i)javascript:\s*[\s\S]*`)
	iframeRegex    = regexp.MustCompile(`(?i)<iframe\b[^>]*>[\s\S]*?</iframe>`)
	objectRegex    = regexp.MustCompile(`(?i)<object\b[^>]*>[\s\S]*?</object>`)
	embedRegex     = regexp.MustCompile(`(?i)<embed\b[^>]*>`)
	evalRegex      = regexp.MustCompile(`(?i)\beval\s*\(\s*[\s\S]*?\)`)
	timerRegex     = regexp.MustCompile(`(?i)\b(setTimeout|setInterval)\s*\(\s*[\s\S]*?\)`)
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

// SanitizeHTML 彻底防御 XSS（核心方法）
func (c *CommStruct) SanitizeHTML(input string) string {
	if input == "" {
		return ""
	}
	// 1. 专业 XSS 过滤
	clean := htmlSanitizer.Sanitize(input)
	// 2. 双重转义，彻底杜绝执行
	clean = html.EscapeString(clean)
	// 3. 清理空白
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
		timerRegex.MatchString(input) {
		return true
	}

	return false
}

// decodeHTMLEntities 完整 HTML 实体解码（防编码绕过）
func (c *CommStruct) decodeHTMLEntities(input string) string {
	// 基础实体
	replacements := map[string]string{
		"&lt;":   "<",
		"&gt;":   ">",
		"&amp;":  "&",
		"&quot;": "\"",
		"&apos;": "'",
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
