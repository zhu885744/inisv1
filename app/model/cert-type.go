package model

import (
	"encoding/json"
	"inis/app/facade"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

/**
 * CertType - 认证类型配置（后台可自定义）
 *
 * 一个类型 = 一套认证表单：类型名称 / 图标 / 认证说明（后台填写，前台展示给申请者）/
 * 认证领域候选 / 需要申请者填写的字段列表。
 *
 * 为什么做成数据而不是常量：
 *   1. 站点主题各异，「职业 / 兴趣 / 机构」只是默认播种，运营可以改成「企业认证 / 创作者认证 / 媒体认证」；
 *   2. 每个类型需要的信息不同（职业要公司职位、创作者要作品链接、机构要营业执照），
 *      字段必须能按类型单独配置，否则只能在代码里堆 if；
 *   3. 「认证说明」是对外展示的口径，必须由后台写死，不能让申请者自己编（避免主页出现营销话术）。
 *
 * 存储口径：
 *   - categories / fields / conditions 都存 JSON 文本（MySQL longtext），读取时解析；
 *   - 认证领域只是一个字符串数组，顺序即展示顺序（仅用于申请筛选，不在个人主页展示）；
 *   - 字段定义见 CertField，key 是用户提交值的存储键；
 *   - 申请条件见 CertConditions，全部为 0 表示无门槛。
 *
 * 兼容性：
 *   - 老数据（inis_user_certification.cert_type = 1/2/3）依赖下面播种的默认三条记录；
 *   - 某条类型被删除后，历史记录的类型名回退到内置 CertTypes（见 CertTypeName）。
 */
type CertType struct {
	Id int `gorm:"type:int(32); comment:主键;" json:"id"`
	// Name - 类型名称（前台下拉 / 主页标识文案）
	Name string `gorm:"type:varchar(32); comment:类型名称;" json:"name"`
	// Icon - Bootstrap Icons 类名（如 bi-briefcase），仅用于展示
	Icon string `gorm:"type:varchar(64); default:Null; comment:图标;" json:"icon"`
	// Color - 认证标识配色（#rgb / #rrggbb / #rrggbbaa），空表示用前端内置兜底色
	//
	// 放在类型上而不是写死在前端：昵称旁的盾牌和简介下的认证行必须同色，
	// 而且自定义类型（id ≥ 4）没有「职业 / 兴趣 / 机构」的语义，只能由运营指定。
	Color string `gorm:"type:varchar(16); default:Null; comment:标识配色;" json:"color"`
	// Intro - 一句话简介（申请页选类型时的副文案，可空）
	Intro string `gorm:"type:varchar(255); default:Null; comment:一句话简介;" json:"intro"`
	// Notice - 认证说明（后台填写，前台原样展示给申请者，不支持用户填写）
	Notice string `gorm:"type:text; comment:认证说明（后台填写）;" json:"notice"`
	// Categories - 认证领域候选（JSON 字符串数组，顺序即展示顺序）
	Categories string `gorm:"type:longtext; comment:认证领域（JSON 数组）;" json:"categories"`
	// Fields - 申请字段定义（JSON 数组，见 CertField）
	Fields string `gorm:"type:longtext; comment:申请字段（JSON 数组）;" json:"fields"`
	// Conditions - 申请条件（JSON，见 CertConditions）：粉丝数 / 文章数 / 动态数 / 注册天数
	//
	// 为什么是固定结构而不是任意表达式：
	//   条件只能取站点里真实存在的统计口径（关注表、文章表、动态表、注册时间），
	//   做成可配置表达式会引入 SQL 拼装与性能风险，运营也不需要那个自由度。
	Conditions string `gorm:"type:longtext; comment:申请条件（JSON）;" json:"conditions"`
	// Sort - 排序（小的在前）
	Sort int `gorm:"type:int(12); default:0; comment:排序;" json:"sort"`
	// Enable - 1 启用 / 0 停用（停用后前台不再可选，已有记录照常展示）
	Enable int `gorm:"type:int(12); default:1; index:idx_cert_type_enable; comment:是否启用;" json:"enable"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// 申请字段类型（与前端渲染器一一对应，新增类型必须同时补前端渲染）
const (
	CertFieldText     = "text"     // 单行文本
	CertFieldTextarea = "textarea" // 多行文本
	CertFieldNumber   = "number"   // 数字
	CertFieldSelect   = "select"   // 下拉（选项来自字段自身的 options）
	CertFieldDate     = "date"     // 日期（YYYY-MM-DD）
	CertFieldLink     = "link"     // 链接（http/https）
	CertFieldImage    = "image"    // 图片（走附件上传，存 URL 数组）
)

// CertFieldTypes - 合法字段类型（校验用）
var CertFieldTypes = []string{
	CertFieldText, CertFieldTextarea, CertFieldNumber, CertFieldSelect,
	CertFieldDate, CertFieldLink, CertFieldImage,
}

// 内置字段键：有特殊语义，写回记录的独立列（其余键进 data JSON）
const (
	CertKeyTitle    = "title"    // 认证名称 → inis_user_certification.title（主页展示）
	CertKeyCategory = "category" // 认证领域 → .category（筛选 / 展示）
	CertKeyImages   = "images"   // 证明材料 → .images（后台核对）
)

// CertSystemKeys - 内置键说明（后台编辑器里提示用）
var CertSystemKeys = map[string]string{
	CertKeyTitle:    "认证名称：展示在用户主页昵称旁，写入记录 title 列",
	CertKeyCategory: "认证领域：写入记录 category 列，可用于后台筛选",
	CertKeyImages:   "证明材料：图片会上传到附件模块，写入记录 images 列",
}

// certColorPattern - 合法颜色：3 / 6 / 8 位十六进制（#rgba / #rrggbb / #rrggbbaa）
var certColorPattern = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)

// NormalizeCertColor - 规范化配色，非法值返回空串（表示「用默认色」）
//
// 只接受十六进制颜色：颜色会直接写进前端 style，放开任意字符串等于给自己留一个 CSS 注入点。
func NormalizeCertColor(value any) string {
	color := strings.TrimSpace(cast.ToString(value))
	if !certColorPattern.MatchString(color) {
		return ""
	}

	return strings.ToLower(color)
}

// CertField - 单个申请字段的定义
type CertField struct {
	// Key - 存储键（英文 / 数字 / 下划线），同类型内唯一
	Key string `json:"key"`
	// Label - 展示名称
	Label string `json:"label"`
	// Type - 见 CertFieldText 等常量
	Type string `json:"type"`
	// Required - 是否必填
	Required bool `json:"required"`
	// Placeholder / Hint - 输入提示 / 字段下方说明
	Placeholder string `json:"placeholder"`
	Hint        string `json:"hint"`
	// Max - 最大字符数（0 表示按类型默认值）
	Max int `json:"max"`
	// Options - select 的候选值
	Options []string `json:"options"`
	// Multiple - select 是否多选
	Multiple bool `json:"multiple"`
	// MaxCount - image 最多几张（0 表示按默认 6 张）
	MaxCount int `json:"max_count"`
	// Public - 是否公开展示（他人可在主页看到该项；admin_only 优先）
	Public bool `json:"public"`
	// AdminOnly - 仅后台可见（如联系方式），不进公开接口
	AdminOnly bool `json:"admin_only"`
}

// 字段长度兜底（字段未配置 max 时按类型取）
const (
	certFieldTextMax     = 64
	certFieldTextareaMax = 500
	certFieldLinkMax     = 512
	certFieldImageMax    = 6
)

// CertConditions - 认证申请门槛
//
// 四项都取站点里已有的真实统计口径，0 表示不限：
//   - fans     粉丝数（user_follows.status=1，按 follow_uid 统计）
//   - articles 已发布文章数（audit=1 且 status=1，草稿与待审核不算）
//   - moments  已发布动态数（同口径）
//   - days     注册天数（users.create_time 到现在的自然天数）
//
// 只统计「对外公开」的内容，避免用户靠刷未过审的草稿/待审稿凑数量。
type CertConditions struct {
	Fans     int `json:"fans"`
	Articles int `json:"articles"`
	Moments  int `json:"moments"`
	Days     int `json:"days"`
}

// 条件上限：防止运营手滑填出天文数字导致永远无法申请（1 亿已远超实际站点规模）
const certConditionMax = 100000000

// certConditionLabels - 条件的展示名（错误提示 / 前台进度条共用同一套文案）
var certConditionLabels = map[string]string{
	"fans":     "粉丝数",
	"articles": "发布文章",
	"moments":  "发布动态",
	"days":     "注册天数",
}

// Empty - 没有任何门槛
func (this CertConditions) Empty() bool {
	return this.Fans <= 0 && this.Articles <= 0 && this.Moments <= 0 && this.Days <= 0
}

// CertConditionsFromAny - 任意入参（map / JSON 字符串 / nil）→ 规范化条件
func CertConditionsFromAny(value any) CertConditions {
	if value == nil {
		return CertConditions{}
	}

	result := CertConditions{}

	switch item := value.(type) {
	case CertConditions:
		result = item
	case map[string]any:
		result = CertConditions{
			Fans:     cast.ToInt(item["fans"]),
			Articles: cast.ToInt(item["articles"]),
			Moments:  cast.ToInt(item["moments"]),
			Days:     cast.ToInt(item["days"]),
		}
	case string:
		text := strings.TrimSpace(item)
		if text == "" {
			return result
		}
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			return CertConditions{}
		}
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return CertConditions{}
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return CertConditions{}
		}
	}

	return result.normalize()
}

// normalize - 负数归零、超上限截断
func (this CertConditions) normalize() CertConditions {
	fix := func(value int) int {
		if value < 0 {
			return 0
		}
		if value > certConditionMax {
			return certConditionMax
		}
		return value
	}

	return CertConditions{
		Fans:     fix(this.Fans),
		Articles: fix(this.Articles),
		Moments:  fix(this.Moments),
		Days:     fix(this.Days),
	}
}

// Check - 判断是否达标，返回未达标项（已达标时第二返回值为空切片）
//
// 未达标项结构：{key,label,need,current,short}
//   - need    门槛值
//   - current 当前值
//   - short   还差多少（need - current），前端直接拿来做进度与文案
func (this CertConditions) Check(stats map[string]int) (bool, []map[string]any) {

	missing := make([]map[string]any, 0)

	items := []struct {
		key  string
		need int
	}{
		{"fans", this.Fans},
		{"articles", this.Articles},
		{"moments", this.Moments},
		{"days", this.Days},
	}

	for _, item := range items {
		if item.need <= 0 {
			continue
		}

		current := cast.ToInt(stats[item.key])
		if current >= item.need {
			continue
		}

		missing = append(missing, map[string]any{
			"key":     item.key,
			"label":   certConditionLabels[item.key],
			"need":    item.need,
			"current": current,
			"short":   item.need - current,
		})
	}

	return len(missing) == 0, missing
}

// CheckMessage - 未达标项的合并文案（直接给用户看）
func (this CertConditions) CheckMessage(missing []map[string]any) string {
	if len(missing) == 0 {
		return ""
	}

	parts := make([]string, 0, len(missing))
	for _, item := range missing {
		parts = append(parts, cast.ToString(item["label"])+"需达到 "+
			cast.ToString(item["need"])+"（当前 "+cast.ToString(item["current"])+"）")
	}

	return "申请条件未满足：" + strings.Join(parts, "；") + "。"
}

// 认证条件统计的缓存
//
// why：申请页每打开一次就要 3 次 count（粉丝 / 文章 / 动态）+ 1 次用户查询，
// 而这三个数字分钟级变化即可接受；「还差多少粉丝」这种提示晚 3 分钟不影响判断。
// 所有会改变计数的动作（关注、取关、内容发布 / 删除 / 审核、类型配置改动）都会主动清。
const (
	// certStatsCacheKey - 缓存 key 基础名
	certStatsCacheKey = "cert-user-stats"
	// certStatsGenKey - 「代际」版本号：整体失效时不遍历删 key，而是把版本号推进
	//
	// 为什么不用 DelPrefix：redis 驱动会把前缀当 glob（`[GET]` 里的方括号是字符类），
	// ram 驱动的 DelPrefix 与写入时的内部前缀口径也不一致；靠字符串前缀删 key 不可靠。
	// 推进版本号只依赖最基础的 Set/Get，三种驱动语义一致，旧 key 到 TTL 自然消失。
	certStatsGenKey = "cert-user-stats-gen"
	// certStatsTTL - 缓存秒数（3 分钟）
	certStatsTTL = int64(180)
)

// certStatsGen - 当前代际（未初始化视为 1）
func certStatsGen() int64 {
	if value := facade.Cache.Get(certStatsGenKey); value != nil {
		if gen := cast.ToInt64(value); gen > 0 {
			return gen
		}
	}

	return 1
}

func certStatsCacheName(uid int) string {
	return certStatsCacheKey + "-" + cast.ToString(certStatsGen()) + "-" + cast.ToString(uid)
}

// CertStatsFlush - 清掉指定用户的统计缓存
//
// 关注 / 取关只会改变被关注者的粉丝数，按 uid 精确清即可；异步执行，不阻塞主流程。
func CertStatsFlush(uids ...int) {
	for _, uid := range uids {
		if uid > 0 {
			go facade.Cache.Del(certStatsCacheName(uid))
		}
	}
}

// CertStatsFlushAll - 让所有用户的统计缓存整体失效
//
// 内容的新增 / 删除 / 审核通过分散在 article、moments 的多个方法里，
// 逐个回溯作者容易漏；而这份缓存重建只有 3 次 count，宁可整体失效一层，
// 也不要让用户看到过期的「还差多少」。
//
// 用毫秒时间戳而不是纳秒：file 驱动会把值 JSON 化再读回，
// 超过 2^53 的整数在 float64 往返时会丢精度。
func CertStatsFlushAll() {
	go facade.Cache.Set(certStatsGenKey, time.Now().UnixMilli(), 86400)
}

// CertUserStats - 用户当前的条件统计值（带缓存）
//
// 申请页进度展示与提交校验共用，保证「页面显示达标」与「后端允许提交」口径一致。
func CertUserStats(uid int) map[string]int {

	if uid <= 0 {
		return map[string]int{"fans": 0, "articles": 0, "moments": 0, "days": 0}
	}

	cacheName := certStatsCacheName(uid)
	if cached := facade.Cache.Get(cacheName); cached != nil {
		// file / ram 驱动取回来的是 JSON 解码后的 map[string]any，需要转一次
		if value, ok := cached.(map[string]int); ok && len(value) > 0 {
			return value
		}
		if value := cast.ToStringMapInt(cached); len(value) > 0 {
			return value
		}
	}

	result := map[string]int{"fans": 0, "articles": 0, "moments": 0, "days": 0}

	result["fans"] = int((&UserFollows{}).GetFollowersCount(uid))
	result["articles"] = certPublishedCount[Article](uid)
	result["moments"] = certPublishedCount[Moments](uid)

	if user, _ := facade.DB.Model(&Users{}).Find(uid); !utils.Is.Empty(user) {
		if createTime := cast.ToInt64(cast.ToStringMap(user)["create_time"]); createTime > 0 {
			if days := (time.Now().Unix() - createTime) / 86400; days > 0 {
				result["days"] = int(days)
			}
		}
	}

	_ = facade.Cache.Set(cacheName, result, certStatsTTL)

	return result
}

// certPublishedCount - 某用户**对外公开**的内容数（动态 / 文章通用：audit=1 且 status=1）
func certPublishedCount[T any](uid int) int {
	count, _ := facade.DB.Model(&[]T{}).
		Where("uid", uid).
		Where("audit", 1).
		Where("status", 1).
		Count()
	return int(count)
}

// InitCertType - 建表 + 播种默认类型
func InitCertType() {
	if err := facade.DB.Drive().AutoMigrate(&CertType{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "CertType表迁移失败")
		return
	}

	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_cert_type_enable ON inis_cert_type(enable)")

	count, _ := facade.DB.Model(&CertType{}).Count()
	if count > 0 {
		return
	}

	// 首次运行：播种与原内置枚举完全一致的三条，保证老体验不变
	for _, item := range defaultCertTypes() {
		table := CertType{
			Id:         item.Id,
			Name:       item.Name,
			Icon:       item.Icon,
			Color:      item.Color,
			Intro:      item.Intro,
			Notice:     item.Notice,
			Categories: utils.Json.Encode(item.Categories),
			Fields:     utils.Json.Encode(item.Fields),
			Conditions: utils.Json.Encode(item.Conditions),
			Sort:       item.Id,
			Enable:     1,
		}
		if _, err := facade.DB.Model(&CertType{}).Create(&table); err != nil {
			facade.Log.Error(map[string]any{"error": err, "name": item.Name}, "认证类型播种失败")
		}
	}
}

// certTypeSeed - 播种用的中间结构（领域 / 字段还是切片，写库前再编码成 JSON）
type certTypeSeed struct {
	Id         int
	Name       string
	Icon       string
	Color      string
	Intro      string
	Notice     string
	Categories []string
	Fields     []CertField
	Conditions CertConditions
}

// defaultCertTypes - 默认认证类型（字段口径与旧版硬编码表单一致）
func defaultCertTypes() []certTypeSeed {
	contact := CertField{
		Key: "contact", Label: "联系方式", Type: CertFieldText, Max: 64,
		Placeholder: "邮箱 / 微信 / 手机号，任选其一", Hint: "仅管理员可见，用于核实资料", AdminOnly: true,
	}
	images := CertField{
		Key: CertKeyImages, Label: "证明材料", Type: CertFieldImage, Required: true, MaxCount: certFieldImageMax,
		Placeholder: "", Hint: "工牌、在职证明、作品链接截图等，最多 6 张",
	}
	title := CertField{
		Key: CertKeyTitle, Label: "认证名称", Type: CertFieldText, Required: true, Max: 32,
		Placeholder: "不超过 32 字", Hint: "展示在昵称旁的身份，如「前端工程师」「独立摄影师」", Public: true,
	}
	// 领域与公司 / 职位默认不公开：个人主页的认证区默认只展示「认证名称 + 认证说明」
	category := CertField{
		Key: CertKeyCategory, Label: "认证领域", Type: CertFieldSelect, Required: true, Public: false,
		Hint: "用于分类与后台筛选，不在个人主页展示",
	}
	company := CertField{
		Key: "company", Label: "公司 / 组织", Type: CertFieldText, Required: true, Max: 64,
		Placeholder: "如：某某科技有限公司", Public: false,
	}
	position := CertField{
		Key: "position", Label: "职位 / 身份", Type: CertFieldText, Required: true, Max: 32,
		Placeholder: "如：前端工程师", Public: false,
	}

	return []certTypeSeed{
		{
			Id:     1,
			Name:   "职业认证",
			Icon:   "bi bi-briefcase",
			Color:  "#e8a020",
			Intro:  "在职身份认证",
			Notice: "在职身份认证，需填写公司 / 组织与职位，适合开发者、设计师、编辑等。审核通过后认证标识会展示在你的个人主页昵称旁。",
			Categories: []string{
				"编程开发", "科技互联网", "设计与创意", "教育", "医疗健康",
				"金融财经", "媒体传播", "企业机构", "其他",
			},
			Fields: []CertField{title, category, company, position, contact, images},
		},
		{
			Id:     2,
			Name:   "兴趣认证",
			Icon:   "bi bi-palette",
			Color:  "#2f8fe8",
			Intro:  "垂直领域创作者",
			Notice: "兴趣领域认证，适合摄影、音乐、游戏等垂直领域的活跃创作者。公司 / 职位不是必填项，但请上传能证明你在该领域活跃的材料。",
			Categories: []string{
				"摄影", "音乐", "影视", "游戏", "动漫二次元",
				"美食", "旅行", "体育", "生活方式", "其他",
			},
			Fields: []CertField{title, category, contact, images},
		},
		{
			Id:     3,
			Name:   "机构认证",
			Icon:   "bi bi-building",
			Color:  "#2fa96b",
			Intro:  "企业 / 组织 / 团队",
			Notice: "企业 / 组织 / 团队认证，需填写组织名称与联系人身份，并上传营业执照或加盖公章的证明。认证主体须与材料一致。",
			Categories: []string{
				"企业机构", "教育", "医疗健康", "金融财经", "媒体传播", "科技互联网", "其他",
			},
			Fields: []CertField{title, category, company, position, contact, images},
		},
	}
}

// parseCertFields - JSON → 字段定义（脏数据一律过滤，不因为一条配置写坏而整表报错）
func parseCertFields(raw string) []CertField {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	var list []CertField
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return nil
	}

	return normalizeCertFields(list)
}

// normalizeCertFields - 清洗字段定义：去空、类型兜底、key 去重
func normalizeCertFields(list []CertField) []CertField {
	result := make([]CertField, 0, len(list))
	seen := make(map[string]bool, len(list))

	for _, item := range list {
		item.Key = strings.TrimSpace(item.Key)
		item.Label = strings.TrimSpace(item.Label)
		if item.Key == "" || seen[item.Key] {
			continue
		}
		if !utils.InArray(item.Type, CertFieldTypes) {
			item.Type = CertFieldText
		}
		if item.Label == "" {
			item.Label = item.Key
		}
		if item.Type == CertFieldImage {
			if item.MaxCount <= 0 {
				item.MaxCount = certFieldImageMax
			}
			if item.MaxCount > 12 {
				item.MaxCount = 12
			}
		}
		if item.Type == CertFieldSelect {
			options := make([]string, 0, len(item.Options))
			for _, option := range item.Options {
				if option = strings.TrimSpace(option); option != "" {
					options = append(options, option)
				}
			}
			item.Options = options
		} else {
			// 一律给空数组而不是 nil：前端拿到 null 还要再兜一次，接口直接输出 [] 更省事
			item.Options = []string{}
			item.Multiple = false
		}
		seen[item.Key] = true
		result = append(result, item)
	}

	return result
}

// parseCertCategories - JSON（或逗号分隔）→ 领域列表
func parseCertCategories(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}

	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		// 兼容手工在数据库里写成「摄影,音乐,游戏」
		list = strings.Split(raw, ",")
	}

	result := make([]string, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}

	return result
}

// CertTypeList - 类型列表（onlyEnable=true 只返回启用的，供前台下拉使用）
func CertTypeList(onlyEnable bool) []map[string]any {
	query := facade.DB.Model(&[]CertType{})
	if onlyEnable {
		query = query.Where("enable", 1)
	}

	items, _ := query.Order("sort asc, id asc").Select()

	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		list = append(list, CertTypeOutput(item))
	}

	return list
}

// CertTypeFind - 取单个类型（不存在返回 nil）
func CertTypeFind(id int) map[string]any {
	if id <= 0 {
		return nil
	}

	item, _ := facade.DB.Model(&CertType{}).Where("id", id).Find()
	if utils.Is.Empty(item) {
		return nil
	}

	return CertTypeOutput(cast.ToStringMap(item))
}

// CertTypeOutput - 统一输出（JSON 字段解析成数组，前端不再二次解析）
func CertTypeOutput(item map[string]any) map[string]any {
	if utils.Is.Empty(item) {
		return map[string]any{}
	}

	fields := parseCertFields(cast.ToString(item["fields"]))
	if fields == nil {
		fields = []CertField{}
	}
	// 领域候选项动态补进 select 字段，避免后台改完领域忘了同步字段选项
	categories := parseCertCategories(cast.ToString(item["categories"]))
	for index, field := range fields {
		if field.Key == CertKeyCategory && field.Type == CertFieldSelect && len(field.Options) == 0 {
			fields[index].Options = categories
		}
	}

	return map[string]any{
		"id":          cast.ToInt(item["id"]),
		"name":        cast.ToString(item["name"]),
		"icon":        cast.ToString(item["icon"]),
		"color":       NormalizeCertColor(item["color"]),
		"intro":       cast.ToString(item["intro"]),
		"notice":      cast.ToString(item["notice"]),
		"categories":  categories,
		"fields":      fields,
		"conditions":  CertConditionsFromAny(item["conditions"]),
		"sort":        cast.ToInt(item["sort"]),
		"enable":      cast.ToInt(item["enable"]),
		"create_time": cast.ToInt64(item["create_time"]),
		"update_time": cast.ToInt64(item["update_time"]),
	}
}

// CertFieldsFromAny - 任意入参（JSON 数组 / JSON 字符串）→ 规范化字段定义
//
// 后台表单拿到的是 []map[string]any，手工调接口时可能是 JSON 字符串，
// 统一走一次 JSON 往返再清洗，避免控制器里重复写类型判断。
func CertFieldsFromAny(value any) []CertField {
	if value == nil {
		return nil
	}

	if text, ok := value.(string); ok {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		var list []CertField
		if err := json.Unmarshal([]byte(text), &list); err != nil {
			return nil
		}
		return normalizeCertFields(list)
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}

	var list []CertField
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}

	return normalizeCertFields(list)
}

// CertCategoriesFromAny - 任意入参 → 领域列表（数组 / "摄影,音乐" 都收）
func CertCategoriesFromAny(value any) []string {
	if value == nil {
		return []string{}
	}

	switch item := value.(type) {
	case string:
		return parseCertCategories(item)
	case []string:
		return parseCertCategories(utils.Json.Encode(item))
	case []any:
		return parseCertCategories(utils.Json.Encode(item))
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return []string{}
	}

	return parseCertCategories(string(raw))
}

// CertTypeName - 类型名（类型被删除时回退到内置枚举，保证历史记录有文案）
func CertTypeName(id int) string {
	if item := CertTypeFind(id); item != nil {
		if name := cast.ToString(item["name"]); name != "" {
			return name
		}
	}

	if text, ok := CertTypes[id]; ok {
		return text
	}

	return "未知类型"
}

// CertTypeConditions - 取类型的申请条件（类型不存在时返回零值，即无门槛）
func CertTypeConditions(id int) CertConditions {
	item := CertTypeFind(id)
	if item == nil {
		return CertConditions{}
	}

	conditions, _ := item["conditions"].(CertConditions)

	return conditions
}

// CertTypeSchema - 取类型用于「提交校验」的三要素
func CertTypeSchema(id int) (fields []CertField, categories []string, name string) {
	item := CertTypeFind(id)
	if item == nil {
		return nil, nil, ""
	}

	fields, _ = item["fields"].([]CertField)
	categories, _ = item["categories"].([]string)

	return fields, categories, cast.ToString(item["name"])
}

// CertFieldMax - 字段的实际长度上限
func CertFieldMax(field CertField) int {
	if field.Max > 0 {
		return field.Max
	}

	switch field.Type {
	case CertFieldTextarea:
		return certFieldTextareaMax
	case CertFieldLink:
		return certFieldLinkMax
	case CertFieldText, CertFieldNumber, CertFieldDate, CertFieldSelect:
		return certFieldTextMax
	}

	return certFieldTextMax
}

// CertFieldOptions - select 字段候选（领域字段跟随类型的领域列表）
func CertFieldOptions(field CertField, categories []string) []string {
	if field.Key == CertKeyCategory && len(field.Options) == 0 {
		return categories
	}
	if len(field.Options) == 0 {
		return nil
	}

	// 保持后台配置的顺序，不做排序：选项顺序本身就是运营的表达
	options := make([]string, len(field.Options))
	copy(options, field.Options)

	return options
}

// AfterFind - 查询Hook
func (this *CertType) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
