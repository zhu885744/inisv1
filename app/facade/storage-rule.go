package facade

/**
 * 上传命名规则（目录结构 + 文件名）—— 对应 config/storage.toml 的 [local] / [cos] 段
 *
 * 本地存储与腾讯云 COS 都支持自定义「目录命名规则」与「文件命名规则」，
 * 后台在「系统设置 → 存储 → 本地存储 / 腾讯云 COS」里可视化修改
 * （PUT /api/toml/storage-local、/api/toml/storage-cos、/api/toml/storage）。
 *
 * 可用占位符（见 StorageRulePlaceholders，后台表单里的提示与本文档同源）：
 *
 *	{Y}             年份(2026)                    {md5}            32 位随机 md5
 *	{y}             两位数年份(26)                {md5-16}         16 位随机 md5（32 位的中段）
 *	{m}             月份(09)                      {str-random-16}  16 位随机字符串
 *	{d}             当月的第几号(26)              {str-random-10}  10 位随机字符串
 *	{timestamp}     时间戳(秒)                    {filename}       文件原始名称（不含扩展名）
 *	{uniqid}        唯一字符串                    {uid}            用户 ID，游客为 0
 *
 * 例（默认值）：
 *
 *	dir_rule  = {Y}-{m}/{d}                 → 2026-09/26
 *	file_rule = {timestamp}{str-random-10}  → 1758888888123abc7def
 *	最终对象键 = path 前缀 / 目录 / 文件名 + 扩展名
 *	          = storage/2026-09/26/1758888888123abc7def.jpg
 *
 * 说明：
 *   - 目录规则留空 = 用默认规则；想让文件直接放在前缀目录下（不要子目录），
 *     把目录规则设为 `/`（或 `.`）即可（语法上会被清成空目录）；
 *   - 同一次上传里目录与文件名共用一组占位符值，{timestamp} / {md5} / {str-random-*}
 *     在两处保持一致（{md5} 与 {md5-16} 同源、{str-random-16} 与 {str-random-10} 同源）；
 *   - 规则结果会被清洗：统一斜杠、剔除 . / .. 等目录穿越片段、去掉非法字符并截断超长段。
 */

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 命名规则类型（config/storage.toml 里的字段名）
const (
	StorageRuleDir  = "dir_rule"
	StorageRuleFile = "file_rule"
)

// 默认命名规则：目录结构与历史行为一致（年-月/日），
// 文件名用「秒级时间戳 + 10 位随机串」保证唯一（避免同秒同名的文件互相覆盖）
const (
	DefaultStorageDirRule  = "{Y}-{m}/{d}"
	DefaultStorageFileRule = "{timestamp}{str-random-10}"
)

// StorageRulePlaceholders 支持的占位符与说明（后台提示 / API 文档 / 校验共用一份口径）
var StorageRulePlaceholders = []map[string]string{
	{"token": "{Y}", "desc": "年份，如 2026"},
	{"token": "{y}", "desc": "两位数年份，如 26"},
	{"token": "{m}", "desc": "月份，如 09"},
	{"token": "{d}", "desc": "当月的第几号，如 26"},
	{"token": "{timestamp}", "desc": "时间戳（秒）"},
	{"token": "{uniqid}", "desc": "唯一字符串（微秒时间戳 + 随机）"},
	{"token": "{md5}", "desc": "32 位随机 md5"},
	{"token": "{md5-16}", "desc": "16 位随机 md5（32 位的中段）"},
	{"token": "{str-random-16}", "desc": "16 位随机字符串"},
	{"token": "{str-random-10}", "desc": "10 位随机字符串"},
	{"token": "{filename}", "desc": "文件原始名称（不含扩展名）"},
	{"token": "{uid}", "desc": "用户 ID，游客为 0"},
}

// storageRuleTokens 支持的占位符（校验用，顺序与 StorageRulePlaceholders 一致）
var storageRuleTokens = []string{
	"{Y}", "{y}", "{m}", "{d}", "{timestamp}", "{uniqid}",
	"{md5}", "{md5-16}", "{str-random-16}", "{str-random-10}", "{filename}", "{uid}",
}

// storageRuleTokenPattern 匹配规则里的 {xxx} 片段（不认识的一律当未知占位符）
var storageRuleTokenPattern = regexp.MustCompile(`\{[^{}]*\}`)

// storageRuleIllegal 各平台的文件名非法字符（Windows 保留字符 + 控制字符）
var storageRuleIllegal = regexp.MustCompile(`[<>:"|?*\x00-\x1F]`)

// storageRuleMaxSegment 单段（目录名 / 文件名）最大长度，避免拼出超长路径
const storageRuleMaxSegment = 100

// storageRuleCharset 随机串字符集：只用小写 + 数字，避免大小写不敏感的文件系统 / CDN 出问题
const storageRuleCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

// StorageRuleContext 生成命名规则时的上下文
type StorageRuleContext struct {
	Uid      int    // 用户 ID，游客为 0
	Filename string // 文件原始名称（不含扩展名，调用方已清洗）
	Ext      string // 扩展名（不含点，小写）
}

// StorageRuleValues 一次上传生成的一组占位符值
//
// 目录规则与文件命名规则共用同一组值（都由 NewStorageRuleValues 生成），
// 保证同一次上传里 {timestamp} / {md5} / {str-random-*} 在目录与文件名中一致。
type StorageRuleValues map[string]string

// NewStorageRuleValues 生成占位符值
func NewStorageRuleValues(ctx StorageRuleContext) StorageRuleValues {

	now := time.Now()

	// md5 与 str-random 各生成一次：{md5} / {md5-16} 同源，{str-random-16} / {str-random-10} 同源
	digest := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v-%v", now.UnixNano(), utils.Rand.String(16, storageRuleCharset)))))
	random := utils.Rand.String(16, storageRuleCharset)

	return StorageRuleValues{
		"{Y}":             now.Format("2006"),
		"{y}":             now.Format("06"),
		"{m}":             now.Format("01"),
		"{d}":             now.Format("02"),
		"{timestamp}":     cast.ToString(now.Unix()),
		"{uniqid}":        fmt.Sprintf("%x%s", now.UnixMicro(), utils.Rand.String(4, storageRuleCharset)),
		"{md5}":           digest,
		"{md5-16}":        digest[8:24],
		"{str-random-16}": random,
		"{str-random-10}": random[:10],
		"{filename}":      ctx.Filename,
		"{uid}":           cast.ToString(ctx.Uid),
	}
}

// Apply 用占位符值套用命名规则，返回清洗后的目录 / 文件名片段
//
// 未知占位符（如 {xx}）会替换成空串：配置保存时已校验过一遍，这里只是兜底。
func (this StorageRuleValues) Apply(rule string) string {

	rule = strings.TrimSpace(rule)
	if rule == "" {
		return ""
	}

	result := storageRuleTokenPattern.ReplaceAllStringFunc(rule, func(token string) string {
		return this[token]
	})

	return sanitizeRuleSegment(result)
}

// UnknownStorageRulePlaceholders 返回规则里不受支持的占位符（保存配置时校验用）
//
// 例：dir_rule = "{Y}/{mm}" → ["{mm}"]
func UnknownStorageRulePlaceholders(rule string) []string {

	supported := make(map[string]struct{}, len(storageRuleTokens))
	for _, token := range storageRuleTokens {
		supported[token] = struct{}{}
	}

	seen := make(map[string]struct{})
	unknown := make([]string, 0)

	for _, token := range storageRuleTokenPattern.FindAllString(rule, -1) {
		if _, ok := supported[token]; ok {
			continue
		}
		// 同一个错误占位符只报一次（如 "{xx}/{xx}"）
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		unknown = append(unknown, token)
	}

	return unknown
}

// StorageRule 读取某个驱动（local / cos）的命名规则，配置缺项或留空时回退默认值
func StorageRule(driver string, kind string) string {

	if StorageToml != nil {
		if value := strings.TrimSpace(cast.ToString(StorageToml.Get(driver + "." + kind))); value != "" {
			return value
		}
	}

	if kind == StorageRuleFile {
		return DefaultStorageFileRule
	}

	return DefaultStorageDirRule
}

// StorageObjectKey 组装对象键：各段之间用 / 连接，空段自动忽略
//
// 前缀（path）来自配置、目录与文件名来自命名规则，都可能带有首尾斜杠或
// 空段（如 cos.path 为空、目录规则设为 "/"），统一在这里规范化，
// 避免生成 "/2026-09/26/x.jpg" 这类键（拼上 CDN 域名会出现 // 导致 404）。
func StorageObjectKey(parts ...string) string {

	segments := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.ReplaceAll(part, "\\", "/")
		for _, segment := range strings.Split(part, "/") {
			segment = strings.TrimSpace(segment)
			// 丢掉空段与 . / .. 这类会改变目录层级的片段
			if segment == "" || segment == "." || segment == ".." {
				continue
			}
			segments = append(segments, segment)
		}
	}

	return strings.Join(segments, "/")
}

// StorageNameWithExt 拼出最终文件名：规则结果 + 扩展名
//
// 规则被清成空（如 file_rule 只写了非法字符）时用毫秒时间戳兜底，保证文件名不为空。
func StorageNameWithExt(name string, ext string) string {

	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")

	if name = strings.TrimSpace(name); name == "" {
		name = cast.ToString(time.Now().UnixNano() / 1e6)
	}

	if ext == "" {
		return name
	}

	return name + "." + ext
}

// sanitizeRuleSegment 清洗命名规则生成的结果
//
// 规则来自后台配置，还可能带上用户上传的文件名，必须按不可信输入处理：
//   - 反斜杠统一成 /，去掉重复斜杠与首尾斜杠；
//   - 丢弃 . / .. 片段，防止写出 public/ 之外的路径；
//   - 去掉 <>:"|?* 与控制字符（Windows 落盘、COS 对象键都不允许）；
//   - 单段截断到 storageRuleMaxSegment，避免超长路径。
func sanitizeRuleSegment(segment string) string {

	segment = strings.ReplaceAll(segment, "\\", "/")
	segment = storageRuleIllegal.ReplaceAllString(segment, "_")

	parts := make([]string, 0)

	for _, part := range strings.Split(segment, "/") {

		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			continue
		}

		// 结尾的点在 Windows 上会被静默去掉（如 "photo." 落盘成 "photo"），
		// 与数据库里记录的名字对不上，这里统一清掉
		part = strings.TrimRight(part, ".")

		if part == "" {
			continue
		}

		if len([]rune(part)) > storageRuleMaxSegment {
			part = strings.TrimRight(string([]rune(part)[:storageRuleMaxSegment]), ".")
		}

		if part != "" {
			parts = append(parts, part)
		}
	}

	return strings.Join(parts, "/")
}
