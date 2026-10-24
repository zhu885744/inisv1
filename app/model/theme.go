package model

import (
	"fmt"
	"hash/fnv"
	"inis/app/facade"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 主题配色（设计 token）
//
// 背景：Mellow 的配色全部走 CSS 变量（见 Mellow/src/assets/styles.css 的 :root 与
// :root[data-theme='dark']），后台只要覆盖这些变量就能换肤，不必改前端代码。
//
// 分层：
//  1. themeTokens —— 可编辑的设计 token，带类型（纯色 / 可渐变 / 长度 / 阴影）；
//  2. 语义别名（--bg / --text / --primary …）在 styles.css 里用 var() 引用基础 token，
//     改基础色即可整体跟随；
//  3. 派生色（--accent-* / --gold-* / --bg-overlay / --switch-knob）由基础色按固定
//     透明度算出来，避免管理员逐项填 rgba；
//  4. extras —— 自由 CSS 变量（高自定义），直接追加到两套模式里。
//
// 存储：config 表的 SYSTEM_THEME 一项（json 字段），与其它系统配置一致。
// 生效：/theme-custom.css 由本文件实时生成（config/app.go 注册路由），
// 保存后立即重建进程内缓存，无需重启。
const ThemeConfigKey = "SYSTEM_THEME"

// defaultThemePreset - 默认预设（与 styles.css 的「暖纸 · 朱砂」一致）
const defaultThemePreset = "cinnabar"

// token 类型
const (
	themeTypeColor  = "color"  // 纯色（文字、描边、品牌色）
	themeTypePaint  = "paint"  // 纯色或渐变（背景类）
	themeTypeLength = "length" // 长度（圆角、间距）
	themeTypeShadow = "shadow" // 阴影
)

// ThemeToken - 一个可编辑的设计 token
type ThemeToken struct {
	// Key CSS 变量名后缀（实际变量为 --Key）
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
	Type  string `json:"type"`
	Desc  string `json:"desc,omitempty"`
	Unit  string `json:"unit,omitempty"`
}

// ThemePreset - 内置主题预设
type ThemePreset struct {
	Key   string            `json:"key"`
	Name  string            `json:"name"`
	Desc  string            `json:"desc"`
	Light map[string]string `json:"light"`
	Dark  map[string]string `json:"dark"`
}

// ThemeSettings - 已保存的主题配色
//
// Light / Dark 存「解析后的完整取值」（预设 + 覆盖），
// 这样 CSS 生成与前端回显都不需要再关心预设叠加逻辑。
// Extras 是自由 CSS 变量（key 不带 --），浅色/深色共用。
type ThemeSettings struct {
	Preset string            `json:"preset"`
	Light  map[string]string `json:"light"`
	Dark   map[string]string `json:"dark"`
	Extras map[string]string `json:"extras"`
}

// themeGroups - token 分组（前端按此顺序渲染编辑区）
var themeGroups = []facade.H{
	{"key": "surface", "label": "底面与描边", "desc": "页面底色、卡片、输入框的分层，以及分隔线（支持渐变）"},
	{"key": "text", "label": "文字层级", "desc": "标题、正文、辅助文字与占位信息"},
	{"key": "brand", "label": "品牌与语义", "desc": "主色（按钮 / 链接 / 强调）与成功、警告色"},
	{"key": "gradient", "label": "渐变", "desc": "默认关闭（none）；配置后叠加在纯色之上，不改动原有主色"},
	{"key": "shape", "label": "圆角", "desc": "控件 10 / 卡片 14 / 弹层 18；可用下方「统一圆角」批量调整"},
	{"key": "depth", "label": "阴影", "desc": "浮层投影；深色模式默认无阴影，靠表面提亮区分层级"},
}

// themeTokens - 可编辑的设计 token 表
var themeTokens = []ThemeToken{
	{Key: "paper-0", Label: "卡片底", Group: "surface", Type: themeTypePaint, Desc: "卡片、弹层等最亮表面"},
	{Key: "paper-1", Label: "页面底", Group: "surface", Type: themeTypePaint, Desc: "整页背景"},
	{Key: "paper-2", Label: "输入 / 浮层", Group: "surface", Type: themeTypePaint, Desc: "输入框、代码块、悬浮元素"},
	{Key: "line", Label: "描边", Group: "surface", Type: themeTypeColor, Desc: "主分隔线与控件描边"},
	{Key: "line-soft", Label: "浅描边", Group: "surface", Type: themeTypeColor, Desc: "卡片边、虚线等更弱的分隔"},

	{Key: "ink-900", Label: "标题 / 正文", Group: "text", Type: themeTypeColor, Desc: "最高对比度文字"},
	{Key: "ink-700", Label: "次级正文", Group: "text", Type: themeTypeColor},
	{Key: "ink-500", Label: "辅助文字", Group: "text", Type: themeTypeColor, Desc: "说明、时间、标签"},
	{Key: "ink-300", Label: "占位 / 弱信息", Group: "text", Type: themeTypeColor, Desc: "placeholder、禁用态"},

	{Key: "cinnabar", Label: "品牌主色", Group: "brand", Type: themeTypeColor, Desc: "主按钮、链接、强调；透明度阶梯由此派生"},
	{Key: "cinnabar-deep", Label: "主色按下 / 链接", Group: "brand", Type: themeTypeColor, Desc: "浅色下更暗，深色下更亮"},
	{Key: "cinnabar-soft", Label: "主色浅调", Group: "brand", Type: themeTypeColor, Desc: "圆点、引用边、弱强调"},
	{Key: "bamboo", Label: "成功色", Group: "brand", Type: themeTypeColor, Desc: "成功提示、版块标识"},
	{Key: "bamboo-deep", Label: "成功色深调", Group: "brand", Type: themeTypeColor},
	{Key: "warning", Label: "警告色", Group: "brand", Type: themeTypeColor, Desc: "警告提示；积分 / 收藏等暖金阶梯由此派生"},

	{Key: "gradient-brand", Label: "品牌渐变", Group: "gradient", Type: themeTypePaint, Desc: "主按钮背景；默认 none，配置后叠加在品牌色上"},
	{Key: "gradient-page", Label: "页面渐变", Group: "gradient", Type: themeTypePaint, Desc: "整页背景；默认 none，配置后叠加在页面底色上"},

	{Key: "radius-xs", Label: "迷你圆角", Group: "shape", Type: themeTypeLength, Unit: "px", Desc: "标签内小按钮"},
	{Key: "radius-sm", Label: "小圆角", Group: "shape", Type: themeTypeLength, Unit: "px", Desc: "按钮、输入框"},
	{Key: "radius", Label: "基础圆角", Group: "shape", Type: themeTypeLength, Unit: "px", Desc: "控件默认"},
	{Key: "radius-lg", Label: "卡片圆角", Group: "shape", Type: themeTypeLength, Unit: "px", Desc: "卡片、面板"},
	{Key: "radius-xl", Label: "弹层圆角", Group: "shape", Type: themeTypeLength, Unit: "px", Desc: "弹窗、浮层"},

	{Key: "shadow-sm", Label: "轻微阴影", Group: "depth", Type: themeTypeShadow, Desc: "卡片默认投影"},
	{Key: "shadow", Label: "悬浮阴影", Group: "depth", Type: themeTypeShadow, Desc: "卡片 hover / 下拉"},
	{Key: "shadow-md", Label: "浮层阴影", Group: "depth", Type: themeTypeShadow, Desc: "弹窗、抽屉"},
}

// themeTokenDefault - 非颜色 token 的默认值（各预设共用）
//
// 颜色 token 的默认值来自预设；圆角 / 阴影 / 渐变与配色无关，统一放在这里，
// 避免在 5 套预设里重复 13 份。
var themeTokenDefault = map[string]struct {
	Light string
	Dark  string
}{
	"gradient-brand": {"none", "none"},
	"gradient-page":  {"none", "none"},

	"radius-xs": {"6px", "6px"},
	"radius-sm": {"8px", "8px"},
	"radius":    {"10px", "10px"},
	"radius-lg": {"14px", "14px"},
	"radius-xl": {"18px", "18px"},

	"shadow-sm": {"0 1px 2px rgba(50, 46, 38, 0.04)", "none"},
	"shadow":    {"0 6px 20px rgba(50, 46, 38, 0.07)", "none"},
	"shadow-md": {"0 14px 34px rgba(50, 46, 38, 0.1)", "none"},
}

// themePresets - 内置预设：每套都给出浅色 / 深色完整取值
var themePresets = []ThemePreset{
	{
		Key:  "cinnabar",
		Name: "朱砂暖纸",
		Desc: "暖纸为底、墨色为文、朱砂为印（默认）",
		Light: map[string]string{
			"paper-0": "#fbf9f5", "paper-1": "#f6f1e8", "paper-2": "#efe8da",
			"line": "#e2d9c8", "line-soft": "#ede6d8",
			"ink-900": "#322e26", "ink-700": "#554e42", "ink-500": "#7b7361", "ink-300": "#a79e8b",
			"cinnabar": "#c7482a", "cinnabar-deep": "#a5371c", "cinnabar-soft": "#e0a08a",
			"bamboo": "#2e7a5c", "bamboo-deep": "#24624a", "warning": "#c08a2e",
		},
		Dark: map[string]string{
			"paper-0": "#221f1b", "paper-1": "#1a1815", "paper-2": "#2b2723",
			"line": "#3b352c", "line-soft": "#302b25",
			"ink-900": "#ece6da", "ink-700": "#c8c1b2", "ink-500": "#9b9384", "ink-300": "#726b5e",
			"cinnabar": "#e2684a", "cinnabar-deep": "#ee8365", "cinnabar-soft": "#b5563a",
			"bamboo": "#5aa884", "bamboo-deep": "#7cc0a0", "warning": "#e2b967",
		},
	},
	{
		Key:  "bamboo",
		Name: "竹青素笺",
		Desc: "素白为底、竹青为印，清爽克制",
		Light: map[string]string{
			"paper-0": "#fbfbf7", "paper-1": "#f4f7f2", "paper-2": "#e8efe7",
			"line": "#d3e0d3", "line-soft": "#e2ebdf",
			"ink-900": "#24302a", "ink-700": "#3f5147", "ink-500": "#67796e", "ink-300": "#9aab9e",
			"cinnabar": "#2f7b5b", "cinnabar-deep": "#22603f", "cinnabar-soft": "#a8cdb8",
			"bamboo": "#2f7b5b", "bamboo-deep": "#22603f", "warning": "#b8862f",
		},
		Dark: map[string]string{
			"paper-0": "#1d2420", "paper-1": "#151b17", "paper-2": "#26302a",
			"line": "#33453a", "line-soft": "#293830",
			"ink-900": "#e4efe6", "ink-700": "#bccfc1", "ink-500": "#8da394", "ink-300": "#66796c",
			"cinnabar": "#5fb98c", "cinnabar-deep": "#7fd0a5", "cinnabar-soft": "#3f7f62",
			"bamboo": "#5fb98c", "bamboo-deep": "#7fd0a5", "warning": "#d8b264",
		},
	},
	{
		Key:  "indigo",
		Name: "靛青书卷",
		Desc: "微冷纸色配靛蓝，适合阅读型站点",
		Light: map[string]string{
			"paper-0": "#fbfbfd", "paper-1": "#f4f5fa", "paper-2": "#e9ebf4",
			"line": "#d6dbeb", "line-soft": "#e4e8f2",
			"ink-900": "#232838", "ink-700": "#414963", "ink-500": "#6b7489", "ink-300": "#9aa2b5",
			"cinnabar": "#3a5ba0", "cinnabar-deep": "#2b4478", "cinnabar-soft": "#a7b7d8",
			"bamboo": "#2e7a5c", "bamboo-deep": "#24624a", "warning": "#c08a2e",
		},
		Dark: map[string]string{
			"paper-0": "#1d1f27", "paper-1": "#16181f", "paper-2": "#262a35",
			"line": "#363c4c", "line-soft": "#2b303c",
			"ink-900": "#e6e8f0", "ink-700": "#c1c6d6", "ink-500": "#9099ae", "ink-300": "#6b7385",
			"cinnabar": "#7d9ce0", "cinnabar-deep": "#a0b9ee", "cinnabar-soft": "#4e6aa8",
			"bamboo": "#5aa884", "bamboo-deep": "#7cc0a0", "warning": "#d8b264",
		},
	},
	{
		Key:  "plum",
		Name: "梅子笺",
		Desc: "藕粉纸色配梅子红，柔和偏暖",
		Light: map[string]string{
			"paper-0": "#fdfbfb", "paper-1": "#faf4f5", "paper-2": "#f1e7e9",
			"line": "#e5d3d7", "line-soft": "#efe2e4",
			"ink-900": "#33262a", "ink-700": "#55424a", "ink-500": "#7d6a71", "ink-300": "#a9979d",
			"cinnabar": "#a63d5c", "cinnabar-deep": "#852c46", "cinnabar-soft": "#d9a8b7",
			"bamboo": "#2e7a5c", "bamboo-deep": "#24624a", "warning": "#c08a2e",
		},
		Dark: map[string]string{
			"paper-0": "#251d20", "paper-1": "#1c1518", "paper-2": "#2f262a",
			"line": "#43353b", "line-soft": "#352a2f",
			"ink-900": "#f0e3e7", "ink-700": "#d0bcc3", "ink-500": "#a38d95", "ink-300": "#7b676f",
			"cinnabar": "#d0708c", "cinnabar-deep": "#e592a9", "cinnabar-soft": "#a04f68",
			"bamboo": "#5aa884", "bamboo-deep": "#7cc0a0", "warning": "#d8b264",
		},
	},
	{
		Key:  "graphite",
		Name: "石墨黑白",
		Desc: "近乎无彩，适合以内容图片为主的站点",
		Light: map[string]string{
			"paper-0": "#ffffff", "paper-1": "#f7f7f8", "paper-2": "#ededf0",
			"line": "#dcdcdf", "line-soft": "#e8e8ec",
			"ink-900": "#1f1f23", "ink-700": "#3f3f46", "ink-500": "#6b6b74", "ink-300": "#9c9ca6",
			"cinnabar": "#4b4b55", "cinnabar-deep": "#33333a", "cinnabar-soft": "#b9b9c2",
			"bamboo": "#3f7a5a", "bamboo-deep": "#2f5f45", "warning": "#b08430",
		},
		Dark: map[string]string{
			"paper-0": "#1e1e21", "paper-1": "#161618", "paper-2": "#28282c",
			"line": "#38383d", "line-soft": "#2c2c30",
			"ink-900": "#ececef", "ink-700": "#c6c6cc", "ink-500": "#94949c", "ink-300": "#6e6e77",
			"cinnabar": "#b6b6c2", "cinnabar-deep": "#d6d6e0", "cinnabar-soft": "#71717c",
			"bamboo": "#5aa884", "bamboo-deep": "#7cc0a0", "warning": "#d8b264",
		},
	},
}

// themeState - 进程内主题缓存（读多写少，读写锁保护）
var themeState = struct {
	sync.RWMutex
	settings ThemeSettings
	css      string
	etag     string
}{
	settings: defaultThemeSettings(),
}

func init() {
	// 包初始化时先用默认预设生成一份 CSS：数据库尚未就绪（未安装 / DB 未连接）时
	// /theme-custom.css 也能正常返回，不会 500。
	themeState.css, themeState.etag = buildThemeCSS(themeState.settings)
}

// ThemeTokens - token 表（副本，避免调用方改动内部数据）
func ThemeTokens() []ThemeToken {
	return append([]ThemeToken(nil), themeTokens...)
}

// ThemeGroups - token 分组
func ThemeGroups() []facade.H {
	return append([]facade.H(nil), themeGroups...)
}

// ThemePresets - 内置预设（深拷贝）
func ThemePresets() []ThemePreset {
	result := make([]ThemePreset, 0, len(themePresets))
	for _, item := range themePresets {
		result = append(result, ThemePreset{
			Key:   item.Key,
			Name:  item.Name,
			Desc:  item.Desc,
			Light: themeClone(item.Light),
			Dark:  themeClone(item.Dark),
		})
	}
	return result
}

// defaultThemeSettings - 默认配色（默认预设的浅色 / 深色完整取值）
func defaultThemeSettings() ThemeSettings {
	preset := themePreset(defaultThemePreset)
	settings := ThemeSettings{
		Preset: preset.Key,
		Light:  themeClone(preset.Light),
		Dark:   themeClone(preset.Dark),
		Extras: map[string]string{},
	}
	themeFillDefaults(&settings)
	return settings
}

// themePreset - 按 key 取预设，不存在时回退默认预设
func themePreset(key string) ThemePreset {
	for _, item := range themePresets {
		if item.Key == key {
			return item
		}
	}
	for _, item := range themePresets {
		if item.Key == defaultThemePreset {
			return item
		}
	}
	return themePresets[0]
}

func themeClone(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

// themeFillDefaults - 给缺失的 token 补默认值（兼容旧版本保存的配置）
func themeFillDefaults(settings *ThemeSettings) {
	if settings.Light == nil {
		settings.Light = map[string]string{}
	}
	if settings.Dark == nil {
		settings.Dark = map[string]string{}
	}
	if settings.Extras == nil {
		settings.Extras = map[string]string{}
	}

	for _, token := range themeTokens {
		if settings.Light[token.Key] == "" {
			settings.Light[token.Key] = themeTokenValue(nil, token, false)
		}
		if settings.Dark[token.Key] == "" {
			settings.Dark[token.Key] = themeTokenValue(nil, token, true)
		}
	}
}

// themeTokenValue - 取 token 值，缺失时回退：非颜色 token 默认值 → 默认预设
func themeTokenValue(values map[string]string, token ThemeToken, dark bool) string {
	if value := strings.TrimSpace(values[token.Key]); value != "" {
		return value
	}
	if item, ok := themeTokenDefault[token.Key]; ok {
		if dark {
			return item.Dark
		}
		return item.Light
	}
	preset := themePreset(defaultThemePreset)
	if dark {
		return preset.Dark[token.Key]
	}
	return preset.Light[token.Key]
}

// ThemeCurrent - 当前生效的配色（副本）
func ThemeCurrent() ThemeSettings {
	themeState.RLock()
	defer themeState.RUnlock()
	return ThemeSettings{
		Preset: themeState.settings.Preset,
		Light:  themeClone(themeState.settings.Light),
		Dark:   themeClone(themeState.settings.Dark),
		Extras: themeClone(themeState.settings.Extras),
	}
}

// ThemeCSS - 当前生效的 CSS 与内容指纹
func ThemeCSS() (string, string) {
	themeState.RLock()
	defer themeState.RUnlock()
	return themeState.css, themeState.etag
}

// ThemeVersion - 主题样式表版本（内容指纹，用于给 /theme-custom.css 加 ?v=）
func ThemeVersion() string {
	themeState.RLock()
	defer themeState.RUnlock()
	return strings.Trim(themeState.etag, `"`)
}

// ThemePayload - 后台「主题配色」页需要的全部数据
func ThemePayload() facade.H {
	settings := ThemeCurrent()
	themeState.RLock()
	css := themeState.css
	themeState.RUnlock()

	return facade.H{
		"settings": settings,
		"defaults": defaultThemeSettings(),
		"tokens":   ThemeTokens(),
		"groups":   ThemeGroups(),
		"presets":  ThemePresets(),
		"css":      css,
	}
}

// ReloadTheme - 从数据库重新加载主题配色（启动 / 配置热更新时调用）
func ReloadTheme() {
	settings := defaultThemeSettings()

	if facade.Installed() {
		if stored, ok := themeStoredSettings(); ok {
			settings = stored
		}
	}

	themeApply(settings)
}

// themeStoredSettings - 从 config 表读取已保存的配色
//
// 为什么不直接用 Model(&Config{}).Find()：Find 的实现是「先扫进 Config 结构体，
// 再把结构体编码成 JSON 解码回来」，其中 json 字段（any 类型）在不同驱动下可能是
// 原始文本、[]byte 或已被上层解码的对象；形态判断错一层就会把整份配置读成空，
// 表现正是「保存成功后又变回预设值」。
//
// 这里直接 Scan 出 json 列的原始文本再解析，形态确定。
func themeStoredSettings() (ThemeSettings, bool) {
	// 路径一：直接取 json 列的原始文本（最确定）
	var payload string
	if err := facade.DB.Drive().
		Model(&Config{}).
		Select("json").
		Where("key = ?", ThemeConfigKey).
		Limit(1).
		Scan(&payload).Error; err == nil && strings.TrimSpace(payload) != "" {
		if data := ParseConfigJson(payload); themeSettingsPayload(data) {
			return normalizeThemeSettings(data["preset"], data["light"], data["dark"], data["extras"]), true
		}
	}

	// 路径二：走 Find，并兼容 json 字段可能是 map / 字符串 / []byte 的差异
	item, _ := facade.DB.Model(&Config{}).Where("key", ThemeConfigKey).Find()
	if data := themeDecodeStored(item["json"]); themeSettingsPayload(data) {
		return normalizeThemeSettings(data["preset"], data["light"], data["dark"], data["extras"]), true
	}

	return ThemeSettings{}, false
}

// themeDecodeStored - 兼容 Find 返回的多种 json 字段形态
func themeDecodeStored(raw any) map[string]any {
	switch value := raw.(type) {
	case map[string]any:
		return value
	case map[string]string:
		result := make(map[string]any, len(value))
		for key, item := range value {
			result[key] = item
		}
		return result
	case []byte:
		return ParseConfigJson(string(value))
	case string:
		return ParseConfigJson(value)
	}
	return nil
}

// themeSettingsPayload - 确认解析出来的是主题配置（而不是别的配置 / 空结构）
func themeSettingsPayload(data map[string]any) bool {
	if len(data) == 0 {
		return false
	}
	_, hasLight := data["light"]
	_, hasDark := data["dark"]
	_, hasPreset := data["preset"]
	return hasLight || hasDark || hasPreset
}

// themeApply - 把配色写入进程内缓存并重建样式表
func themeApply(settings ThemeSettings) {
	themeState.Lock()
	themeState.settings = settings
	themeState.css, themeState.etag = buildThemeCSS(settings)
	themeState.Unlock()
}

// SaveTheme - 校验并保存主题配色，成功后立即生效
func SaveTheme(settings ThemeSettings) (ThemeSettings, error) {
	// 未完成安装时没有可写的配置表（保存接口本身也需要登录，这里是双保险）
	if !facade.Installed() {
		return ThemeSettings{}, fmt.Errorf("站点尚未完成安装，无法保存主题配色")
	}

	normalized := normalizeThemeSettings(settings.Preset, settings.Light, settings.Dark, settings.Extras)

	// 用 map 而不是直接编码结构体：与项目里其它配置写入保持一致，
	// 也避免不同 JSON 库对结构体指针 / 空 map 的处理差异
	payload := utils.Json.Encode(map[string]any{
		"preset": normalized.Preset,
		"light":  normalized.Light,
		"dark":   normalized.Dark,
		"extras": normalized.Extras,
	})

	exist, _ := facade.DB.Model(&Config{}).WithTrashed().Where("key", ThemeConfigKey).Exist()

	var err error
	if exist {
		// 带上 delete_time：该记录若曾被「系统配置」软删除，保存主题时顺手恢复，
		// 否则更新的是回收站里的行，读取（默认过滤 delete_time=0）仍拿不到新值。
		_, err = facade.DB.Model(&Config{}).WithTrashed().Where("key", ThemeConfigKey).Update(map[string]any{
			"json":        payload,
			"delete_time": 0,
		})
	} else {
		_, err = facade.DB.Model(&Config{}).Create(&Config{
			Key:    ThemeConfigKey,
			Json:   payload,
			Remark: "主题配色（设计 token，浅色/深色两套）",
		})
	}

	if err != nil {
		return ThemeSettings{}, err
	}

	// 读配置的接口有缓存（config[KEY] / [GET]config[KEY]），清掉避免回显旧值
	facade.Cache.Del("config[" + ThemeConfigKey + "]")
	facade.Cache.Del("[GET]config[" + ThemeConfigKey + "]")

	// 写库成功后直接用归一化结果刷新内存：
	// 不再「写完立刻回读」——回读形态一旦不稳定，就会出现「保存后变回预设值」。
	// 重启后由 ReloadTheme 从库里读回（读取路径已改为直接取原始文本）。
	themeApply(normalized)

	return normalized, nil
}

// ResetTheme - 恢复默认配色
func ResetTheme() (ThemeSettings, error) {
	return SaveTheme(defaultThemeSettings())
}

// normalizeThemeSettings - 以预设为基准，逐项套用合法覆盖值
//
// 任何非法值（空、超长、含 ; { } 等可能破坏 CSS 的字符）都会被丢弃并回退到预设值，
// 保证最终生成的样式表一定可用。
func normalizeThemeSettings(presetKey any, light any, dark any, extras any) ThemeSettings {
	preset := themePreset(strings.TrimSpace(cast.ToString(presetKey)))

	result := ThemeSettings{
		Preset: preset.Key,
		Light:  themeClone(preset.Light),
		Dark:   themeClone(preset.Dark),
		Extras: themeExtras(extras),
	}

	themeApplyOverrides(result.Light, themeOverrides(light))
	themeApplyOverrides(result.Dark, themeOverrides(dark))
	themeFillDefaults(&result)

	return result
}

// themeOverrides - 把「提交上来的覆盖值」统一成 map[string]any
//
// 不能直接用 cast.ToStringMap：cast v1.10 的 toMapE 只做类型断言，没有反射兜底，
// map[string]string（本函数上游 settings.Light 的类型）会落到 default 分支返回空 map。
// 表现就是：请求里 15 个颜色都在，归一化后全被丢弃，保存完又变回预设值。
func themeOverrides(value any) map[string]any {
	switch item := value.(type) {
	case map[string]any:
		return item
	case map[string]string:
		result := make(map[string]any, len(item))
		for key, val := range item {
			result[key] = val
		}
		return result
	case string:
		return ParseConfigJson(item)
	}
	return nil
}

func themeApplyOverrides(target map[string]string, overrides map[string]any) {
	if len(overrides) == 0 {
		return
	}
	for _, token := range themeTokens {
		value := strings.TrimSpace(cast.ToString(overrides[token.Key]))
		if value == "" {
			continue
		}
		if !themeValueValid(token.Type, value) {
			continue
		}
		target[token.Key] = value
	}
}

// themeExtras - 规范化自由 CSS 变量（key 去 -- 前缀、校验、去重）
func themeExtras(value any) map[string]string {
	result := map[string]string{}
	for key, val := range themeOverrides(value) {
		name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(cast.ToString(key))), "--")
		name = strings.TrimPrefix(name, "--")
		if !themeExtraKeyValid(name) {
			continue
		}
		if themeTokenExists(name) {
			continue
		}
		text := strings.TrimSpace(cast.ToString(val))
		if !themeExtraValueValid(text) {
			continue
		}
		result[name] = text
	}
	return result
}

var themeExtraKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)

func themeExtraKeyValid(key string) bool {
	return themeExtraKeyPattern.MatchString(key)
}

func themeTokenExists(key string) bool {
	for _, token := range themeTokens {
		if token.Key == key {
			return true
		}
	}
	return false
}

func themeExtraValueValid(value string) bool {
	if value == "" || len(value) > 200 {
		return false
	}
	if strings.ContainsAny(value, ";{}<>\"'\\") {
		return false
	}
	return !strings.Contains(strings.ToLower(value), "url(")
}

// themeValueValid - 按 token 类型校验取值
func themeValueValid(tokenType string, value string) bool {
	if value == "" || len(value) > 200 {
		return false
	}
	if strings.ContainsAny(value, ";{}<>\"'\\") {
		return false
	}

	switch tokenType {
	case themeTypePaint:
		return value == "none" || themeColorValid(value) || themeGradientValid(value)
	case themeTypeLength:
		return themeLengthValid(value)
	case themeTypeShadow:
		return themeShadowValid(value)
	default:
		return themeColorValid(value)
	}
}

// themeColorValid - 颜色值校验（十六进制 / rgb() / rgba() / hsl() / 颜色关键字）
//
// 只做「不会破坏样式表」的白名单校验，不追求完整解析 CSS 颜色：
// 分母是后台管理员，真正的风险是填入 `}` 之类的字符把后面的样式截断。
func themeColorValid(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	if strings.ContainsAny(value, ";{}<>\"'\\") {
		return false
	}

	lower := strings.ToLower(value)

	if strings.HasPrefix(lower, "#") {
		hex := strings.TrimPrefix(lower, "#")
		if len(hex) != 3 && len(hex) != 6 && len(hex) != 8 {
			return false
		}
		_, err := strconv.ParseUint(hex, 16, 64)
		return err == nil
	}

	for _, prefix := range []string{"rgb(", "rgba(", "hsl(", "hsla("} {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ")") {
			return true
		}
	}

	// CSS 颜色关键字（如 transparent / currentColor）
	for _, char := range lower {
		if (char < 'a' || char > 'z') && char != '-' {
			return false
		}
	}
	return lower != ""
}

// themeGradientValid - 渐变校验（只接受 gradient 函数，禁止 url()）
func themeGradientValid(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))

	allowed := []string{
		"linear-gradient(", "radial-gradient(", "conic-gradient(",
		"repeating-linear-gradient(", "repeating-radial-gradient(",
	}

	matched := false
	for _, prefix := range allowed {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ")") {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}

	return !strings.Contains(lower, "url(")
}

var themeLengthPattern = regexp.MustCompile(`^(0|-?[0-9]+(\.[0-9]+)?(px|rem|em|%|vh|vw|vmin|vmax|pt|ch))$`)

func themeLengthValid(value string) bool {
	return themeLengthPattern.MatchString(strings.TrimSpace(strings.ToLower(value)))
}

// themeShadowValid - 阴影校验
//
// 支持 none、单段 / 多段阴影；每段至少要有 2 个长度值（x y），颜色段可选，
// 允许 inset。只做结构上的宽松校验，重点是挡住 ; { } 这类会截断样式表的字符。
func themeShadowValid(value string) bool {
	lower := strings.TrimSpace(strings.ToLower(value))
	if lower == "none" {
		return true
	}
	if strings.Contains(lower, "url(") {
		return false
	}

	// 按顶层逗号分段：rgba(0, 0, 0, 0.12) 内部的逗号不能切开
	for _, part := range themeSplitTopLevel(lower) {
		lengths := 0
		for _, field := range strings.Fields(part) {
			if field == "inset" {
				continue
			}
			if themeLengthValid(field) {
				lengths++
			}
		}
		if lengths < 2 {
			return false
		}
	}

	return true
}

// themeSplitTopLevel - 按顶层逗号拆分（忽略括号内的逗号）
func themeSplitTopLevel(text string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, char := range text {
		switch char {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				if item := strings.TrimSpace(current.String()); item != "" {
					result = append(result, item)
				}
				current.Reset()
				continue
			}
		}
		current.WriteRune(char)
	}

	if item := strings.TrimSpace(current.String()); item != "" {
		result = append(result, item)
	}

	return result
}

// buildThemeCSS - 生成覆盖 styles.css 的样式表（浅色 + 深色）
func buildThemeCSS(settings ThemeSettings) (string, string) {
	var builder strings.Builder

	builder.WriteString("/* 主题配色：由后台「主题配色」生成，请勿手工编辑 */\n")
	builder.WriteString("/* 变量含义见 Mellow/src/assets/styles.css 的 :root 区块 */\n\n")

	// 为什么用 :root:root 而不是 :root：
	// 主题产物里的 styles.css 也定义了 :root，两者特异性相同，谁后加载谁生效；
	// 而 <link> 在 head 里的位置会随构建 / 模板变化。把本文件的特异性顶到 (0,2,0)，
	// 就能与加载顺序无关地稳定覆盖默认值。深色块再多一个属性选择器，保证压过浅色块。
	builder.WriteString(themeBlock(":root:root", settings, false))
	builder.WriteString("\n")
	builder.WriteString(themeBlock(":root:root[data-theme='dark']", settings, true))

	css := builder.String()

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(css))
	etag := `"` + strconv.FormatUint(uint64(hash.Sum32()), 16) + `"`

	return css, etag
}

// themeBlock - 生成一个作用域的变量块
func themeBlock(selector string, settings ThemeSettings, dark bool) string {
	var builder strings.Builder

	values := settings.Light
	preset := themePreset(settings.Preset)
	if dark {
		values = settings.Dark
	}

	builder.WriteString(selector)
	builder.WriteString(" {\n")

	for _, token := range themeTokens {
		fmt.Fprintf(&builder, "  --%s: %s;\n", token.Key, themeTokenValue(values, token, dark))
	}

	// 主色透明度阶梯：按钮浅底、焦点环、高亮都从这里取。
	// 主色被填成渐变 / 非法值时，退回预设主色算 rgba（渐变没法直接取 RGB）。
	primary := themeTokenValue(values, themeTokenByKey("cinnabar"), dark)
	primaryFallback := preset.Light["cinnabar"]
	if dark {
		primaryFallback = preset.Dark["cinnabar"]
	}
	alpha := []string{"0.06", "0.12", "0.2", "0.38"}
	if dark {
		alpha = []string{"0.1", "0.18", "0.3", "0.45"}
	}
	for index, name := range []string{"accent-wash", "accent-soft", "accent-ring", "accent-glow"} {
		fmt.Fprintf(&builder, "  --%s: %s;\n", name, themeRGBA(primary, alpha[index], primaryFallback))
	}

	// 暖金阶梯：积分 / 收藏等辅助强调，取警告色
	warning := themeTokenValue(values, themeTokenByKey("warning"), dark)
	warningFallback := preset.Light["warning"]
	if dark {
		warningFallback = preset.Dark["warning"]
	}
	goldAlpha := []string{"0.1", "0.16", "0.32"}
	if dark {
		goldAlpha = []string{"0.12", "0.18", "0.34"}
	}
	for index, name := range []string{"gold-wash", "gold-soft", "gold-line"} {
		fmt.Fprintf(&builder, "  --%s: %s;\n", name, themeRGBA(warning, goldAlpha[index], warningFallback))
	}

	// 半透明浮层底：浅色取卡片底，深色取页面底（与 styles.css 原口径一致）
	overlayKey := "paper-0"
	if dark {
		overlayKey = "paper-1"
	}
	overlay := themeTokenValue(values, themeTokenByKey(overlayKey), dark)
	overlayFallback := preset.Light[overlayKey]
	if dark {
		overlayFallback = preset.Dark[overlayKey]
	}
	fmt.Fprintf(&builder, "  --bg-overlay: %s;\n", themeRGBA(overlay, "0.86", overlayFallback))

	// 深色下的开关滑块：用最高对比度文字色，避免纯白滑块与暖色底脱节
	if dark {
		knob := themeTokenValue(values, themeTokenByKey("ink-900"), true)
		fmt.Fprintf(&builder, "  --switch-knob: %s;\n", knob)
	}

	// 自由 CSS 变量：排序后输出，保证 ETag 稳定
	keys := make([]string, 0, len(settings.Extras))
	for key := range settings.Extras {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&builder, "  --%s: %s;\n", key, settings.Extras[key])
	}

	builder.WriteString("}\n")

	return builder.String()
}

// themeTokenByKey - 按 key 找 token（找不到返回零值，themeTokenValue 会回退默认预设）
func themeTokenByKey(key string) ThemeToken {
	for _, token := range themeTokens {
		if token.Key == key {
			return token
		}
	}
	return ThemeToken{Key: key}
}

// themeRGBA - 把颜色转成带透明度的 rgba()；解析失败时用 fallback 再试，最后给中性灰
func themeRGBA(value string, alpha string, fallback string) string {
	if red, green, blue, ok := themeParseHex(value); ok {
		return fmt.Sprintf("rgba(%d, %d, %d, %s)", red, green, blue, alpha)
	}
	if red, green, blue, ok := themeParseHex(fallback); ok {
		return fmt.Sprintf("rgba(%d, %d, %d, %s)", red, green, blue, alpha)
	}
	return "rgba(128, 128, 128, " + alpha + ")"
}

// themeParseHex - 解析 #rgb / #rrggbb / #rrggbbaa 为 RGB
func themeParseHex(value string) (int, int, int, bool) {
	text := strings.TrimSpace(strings.ToLower(value))
	if !strings.HasPrefix(text, "#") {
		return 0, 0, 0, false
	}

	hex := strings.TrimPrefix(text, "#")
	if len(hex) == 8 {
		hex = hex[:6]
	}
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 0, 0, 0, false
	}

	number, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}

	return int(number >> 16 & 0xff), int(number >> 8 & 0xff), int(number & 0xff), true
}

// InitTheme - 初始化主题配色
//
// 只往 config 表补一条默认记录（已有记录不动），然后加载到内存。
// 由 InitTable 调用，此时数据库已就绪。
func InitTheme() {
	if err := facade.DB.Drive().AutoMigrate(&Config{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Config表迁移失败（主题配色）")
		return
	}

	exist, _ := facade.DB.Model(&Config{}).WithTrashed().Where("key", ThemeConfigKey).Exist()
	if !exist {
		settings := defaultThemeSettings()
		if _, err := facade.DB.Model(&Config{}).Create(&Config{
			Key: ThemeConfigKey,
			Json: utils.Json.Encode(map[string]any{
				"preset": settings.Preset,
				"light":  settings.Light,
				"dark":   settings.Dark,
				"extras": settings.Extras,
			}),
			Remark: "主题配色（设计 token，浅色/深色两套）",
		}); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error()}, "写入默认主题配色失败")
		}
	}

	ReloadTheme()
}
