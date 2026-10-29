package model

import (
	"inis/app/facade"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

/**
 * UserCertification - 用户认证（职业 / 兴趣 / 机构）
 *
 * 参考 B 站、抖音的「职业认证 / 兴趣认证」：用户提交资料与证明材料 → 后台审核 →
 * 通过后在其个人主页昵称旁展示认证图标，并可查看认证名称、领域与说明。
 *
 * 设计要点：
 *   1. **一人一条**（uid 唯一索引）：认证是账号属性，不是可累积的凭证；
 *      被驳回后带着原资料修改重提，通过后再次提交则重新进入审核队列。
 *   2. **敏感信息隔离**：证明材料、联系方式、公司全称只在后台接口返回，
 *      前台公开接口（one）只给认证名称 / 类型 / 领域 / 说明这类展示型字段。
 *   3. **审核留痕**：handler_uid / reason / handle_time，便于审计与告知用户驳回原因。
 *   4. **有效期**：expire_time=0 表示永久；过期后用户主页不再展示，需重新申请。
 *
 * 状态流转（Status）：
 *   0 待审核 → 1 已通过
 *            → 2 已驳回（用户可修改后重新提交，回到 0）
 */
type UserCertification struct {
	Id int `gorm:"type:int(32); comment:主键;" json:"id"`
	// Uid - 申请人（一人一条，唯一索引）
	Uid int `gorm:"type:int(32); uniqueIndex:uk_cert_uid; comment:申请人ID;" json:"uid"`
	// CertType - 1 职业认证 / 2 兴趣认证 / 3 机构认证
	CertType int `gorm:"type:int(12); default:1; comment:认证类型 1职业 2兴趣 3机构;" json:"cert_type"`
	// Category - 认证领域（如「编程开发」「摄影」），用于展示与筛选
	Category string `gorm:"type:varchar(32); comment:认证领域;" json:"category"`
	// Title - 认证名称（展示在昵称后的「认证身份」，如「前端工程师」「独立摄影师」）
	Title string `gorm:"type:varchar(32); comment:认证名称;" json:"title"`
	// Company / Position - 公司 / 职位（职业、机构认证必填，兴趣认证可空）
	Company  string `gorm:"type:varchar(64); default:Null; comment:公司/组织;" json:"company"`
	Position string `gorm:"type:varchar(32); default:Null; comment:职位/身份;" json:"position"`
	// Description - 认证说明（**后台填写**，对外展示；用户不能自己写，避免主页变成广告位）
	Description string `gorm:"type:varchar(255); default:Null; comment:认证说明（后台填写）;" json:"description"`
	// Contact - 联系方式（仅后台可见，用于核实资料）
	Contact string `gorm:"type:varchar(64); default:Null; comment:联系方式（仅后台可见）;" json:"contact"`
	// Images - 证明材料图片（逗号分隔 URL，存储口径与举报证据一致）
	Images string `gorm:"type:text; comment:证明材料图片;" json:"images"`
	// Data - 自定义字段值（JSON：{字段key: 值}），字段定义见 model.CertType.Fields
	Data string `gorm:"type:longtext; comment:自定义字段值(JSON);" json:"data"`
	// Status - 0 待审核 / 1 已通过 / 2 已驳回
	Status int `gorm:"type:int(12); index:idx_cert_status; default:0; comment:状态 0待审 1通过 2驳回;" json:"status"`
	// Reason - 驳回原因（用户可见）
	Reason     string `gorm:"type:varchar(255); default:Null; comment:驳回原因;" json:"reason"`
	HandlerUid int    `gorm:"type:int(32); default:0; comment:审核人ID;" json:"handler_uid"`
	HandleTime int64  `gorm:"default:0; comment:审核时间;" json:"handle_time"`
	// ExpireTime - 认证有效期（0=永久）
	ExpireTime int64 `gorm:"default:0; comment:认证有效期 0=永久;" json:"expire_time"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// 认证状态
const (
	CertPending  = 0 // 待审核
	CertPassed   = 1 // 已通过
	CertRejected = 2 // 已驳回
)

// 认证类型
const (
	CertTypeCareer   = 1 // 职业认证
	CertTypeInterest = 2 // 兴趣认证
	CertTypeOrg      = 3 // 机构认证
)

// CertTypes - 内置类型文案（**仅兜底**：正常情况以 cert-type 表的后台配置为准）
//
// 用途只有两个：首次安装播种默认类型、历史记录的类型被删除后仍能显示文案。
var CertTypes = map[int]string{
	CertTypeCareer:   "职业认证",
	CertTypeInterest: "兴趣认证",
	CertTypeOrg:      "机构认证",
}

// CertCategories - 认证领域候选（前台下拉 / 后台筛选用，选填但仍建议统一取值）
var CertCategories = []string{
	"编程开发", "科技互联网", "设计与创意", "音乐", "影视", "游戏", "动漫二次元",
	"摄影", "教育", "医疗健康", "金融财经", "媒体传播", "体育", "美食",
	"旅行", "生活方式", "企业机构", "其他",
}

// CertStatusText - 状态文案
func CertStatusText(status int) string {
	switch status {
	case CertPending:
		return "待审核"
	case CertPassed:
		return "已通过"
	case CertRejected:
		return "已驳回"
	}
	return "未知"
}

// CertTypeText - 类型文案
func CertTypeText(certType int) string {
	if text, ok := CertTypes[certType]; ok {
		return text
	}
	return "未知类型"
}

// CertCategoryText - 领域文案（领域是自由文本，仅做去空处理）
func CertCategoryText(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return "其他"
	}
	return category
}

func InitUserCertification() {
	if err := facade.DB.Drive().AutoMigrate(&UserCertification{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserCertification表迁移失败")
		return
	}

	// 索引显式补一次：AutoMigrate 在老表上不一定补齐（同 badge / checkin 的处理）
	facade.DB.Drive().Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_cert_uid ON inis_user_certification(uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_cert_status ON inis_user_certification(status)")
}

// GetByUid - 取某用户的认证记录（含未通过）
func (this *UserCertification) GetByUid(uid int) map[string]any {
	if uid <= 0 {
		return nil
	}
	item, _ := facade.DB.Model(&UserCertification{}).Where("uid", uid).Find()
	if utils.Is.Empty(item) {
		return nil
	}
	return cast.ToStringMap(item)
}

// CertSummary - 用户主页展示用的认证摘要（仅「已通过且未过期」）
//
// 只返回展示型字段：联系方式、证明材料、公司全称等敏感信息不出前台。
// 未认证 / 审核中 / 已过期均返回 nil，前端按「无认证」渲染。
func CertSummary(uid int) map[string]any {
	if uid <= 0 {
		return nil
	}

	item, _ := facade.DB.Model(&UserCertification{}).
		Where("uid", uid).
		Where("status", CertPassed).
		Find()
	if utils.Is.Empty(item) {
		return nil
	}

	record := cast.ToStringMap(item)
	if expire := cast.ToInt64(record["expire_time"]); expire > 0 && time.Now().Unix() > expire {
		return nil
	}

	certType := cast.ToInt(record["cert_type"])
	data := CertDataMap(record["data"])

	// 一次查询拿到类型名与字段定义：主页是高频接口，别在这里做两次查表
	typeText := CertTypeText(certType)
	color := ""
	fields := []CertField{}
	if config := CertTypeFind(certType); config != nil {
		if name := cast.ToString(config["name"]); name != "" {
			typeText = name
		}
		color = cast.ToString(config["color"])
		fields, _ = config["fields"].([]CertField)
	}

	// 认证名称是主页标识的文案：老记录 / 后台删掉了 title 字段时回落到类型名，
	// 否则前端 VerifiedBadge 会因为没有 title 而整个不渲染
	title := cast.ToString(record["title"])
	if title == "" {
		title = typeText
	}

	return map[string]any{
		"id":        cast.ToInt(record["id"]),
		"uid":       uid,
		"cert_type": certType,
		"type_text": typeText,
		// 认证标识配色：昵称旁的盾牌与简介下的认证行都读这个值，
		// 避免两处各写一套色值导致同一认证出现两种颜色
		"color": color,
		// 注意：认证领域（category）不在公开摘要里返回。
		// 个人主页的认证区默认只展示认证名称 + 后台填写的认证说明；
		// 领域仅用于申请分类与后台筛选（mine / all 仍会返回）。
		"title":       title,
		"description": cast.ToString(record["description"]),
		"expire_time": cast.ToInt64(record["expire_time"]),
		"handle_time": cast.ToInt64(record["handle_time"]),
		// 后台标记为「公开展示」的自定义字段（公司 / 职位 / 作品链接等），
		// 认证说明由后台填写，其余敏感项（联系方式、证明材料）不出前台
		"fields": CertPublicFields(fields, record, data),
	}
}

// CertBadgeCacheTTL - 头像角标摘要的缓存时间（秒）
//
// 一条列表响应可能内嵌几十个作者，逐个回表会变成 N 次查询；认证是低频变更
// （审核通过 / 过期），缓存 60 秒足够，后台审核通过后最迟一分钟前台可见。
const CertBadgeCacheTTL = 60

// CertBadgeSummary - 头像右下角角标用的极简认证摘要（title / 类型 / 配色）
//
// 与 CertSummary 的区别：不返回 description / fields，也不去查自定义字段配置，
// 只为「头像角上一个盾牌 + hover 文案」服务，因此可以按作者逐个取（带缓存）。
// 未认证 / 审核中 / 已过期返回 nil；空结果同样进缓存，避免每次列表都回表。
func CertBadgeSummary(uid int) map[string]any {
	if uid <= 0 {
		return nil
	}

	cacheName := "author-cert-badge-" + cast.ToString(uid)
	switch cached := facade.Cache.Get(cacheName).(type) {
	case map[string]any:
		if len(cached) == 0 {
			return nil
		}
		return cached
	case facade.H:
		if len(cached) == 0 {
			return nil
		}
		return cached
	}

	badge := map[string]any{}

	item, _ := facade.DB.Model(&UserCertification{}).
		Where("uid", uid).
		Where("status", CertPassed).
		Find()

	if !utils.Is.Empty(item) {
		record := cast.ToStringMap(item)
		expire := cast.ToInt64(record["expire_time"])

		if expire == 0 || time.Now().Unix() <= expire {
			certType := cast.ToInt(record["cert_type"])
			typeText := CertTypeText(certType)
			color := ""
			if config := CertTypeFind(certType); config != nil {
				if name := cast.ToString(config["name"]); name != "" {
					typeText = name
				}
				color = cast.ToString(config["color"])
			}

			// 认证名称缺失时回落到类型名，否则前端 VerifiedBadge 会因为没 title 而不渲染
			title := cast.ToString(record["title"])
			if title == "" {
				title = typeText
			}

			badge = map[string]any{
				"uid":       uid,
				"cert_type": certType,
				"type_text": typeText,
				"color":     color,
				"title":     title,
			}
		}
	}

	facade.Cache.Set(cacheName, badge, CertBadgeCacheTTL)

	if len(badge) == 0 {
		return nil
	}
	return badge
}

// CertBadgeFlush - 清掉某用户的头像角标缓存（认证变更后立即生效）
//
// CertBadgeSummary 是给列表接口用的 60 秒短缓存；授予 / 审核 / 修改 / 收回之后
// 必须让它立刻失效，否则最长一分钟内主页还画着旧盾牌（收回后尤其不能拖）。
func CertBadgeFlush(uid int) {
	if uid <= 0 {
		return
	}
	facade.Cache.Del("author-cert-badge-" + cast.ToString(uid))
}

// CertDataMap - 自定义字段值（JSON 文本 / map 都兼容，脏数据返回空 map）
func CertDataMap(raw any) map[string]any {
	switch value := raw.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return value
	case []byte:
		return cast.ToStringMap(utils.Json.Decode(string(value)))
	}

	text := strings.TrimSpace(cast.ToString(raw))
	if text == "" {
		return map[string]any{}
	}

	return cast.ToStringMap(utils.Json.Decode(text))
}

// certLegacyColumns - 升级前提交的老记录只有这些独立列，字段值缺失时按同名列回落
var certLegacyColumns = map[string]bool{
	"title": true, "category": true, "company": true,
	"position": true, "contact": true, "description": true, "images": true,
}

// CertLegacyColumn - 该字段键是否存在对应的老独立列（用于老数据回落）
func CertLegacyColumn(key string) bool {
	return certLegacyColumns[key]
}

// CertPublicFields - 前台可展示的字段（Public 且非 AdminOnly，空值略过）
//
// 逐项返回 {key,label,type,value,text}，前端按顺序渲染即可；
// 老记录（data 为空）回落到同名独立列，避免升级后主页突然变空白。
func CertPublicFields(fields []CertField, record map[string]any, data map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(fields))

	for _, field := range fields {
		if !field.Public || field.AdminOnly {
			continue
		}
		// 内置键由独立列提供展示口径（认证名称 / 领域已在摘要里），这里只补自定义项
		if field.Key == CertKeyTitle || field.Key == CertKeyCategory {
			continue
		}

		value := data[field.Key]
		if utils.Is.Empty(value) && record != nil && certLegacyColumns[field.Key] {
			value = record[field.Key]
		}
		text := CertValueText(field, value)
		if text == "" {
			continue
		}

		result = append(result, map[string]any{
			"key":   field.Key,
			"label": field.Label,
			"type":  field.Type,
			"value": value,
			"text":  text,
		})
	}

	return result
}

// CertValueText - 字段值的展示文本（图片 / 多选统一成可读串）
func CertValueText(field CertField, value any) string {
	if utils.Is.Empty(value) {
		return ""
	}

	switch field.Type {
	case CertFieldImage:
		// 注意：字符串必须按逗号拆，不能用 cast.ToStringSlice ——
		// 那个函数对 string 是按空白切的，会把 "a.png, b.png" 拆成 "a.png," 与 "b.png"
		var list []string
		switch item := value.(type) {
		case string:
			list = strings.Split(item, ",")
		case []string:
			list = item
		default:
			list = cast.ToStringSlice(value)
		}

		clean := make([]string, 0, len(list))
		for _, item := range list {
			if item = strings.TrimSpace(item); item != "" {
				clean = append(clean, item)
			}
		}
		return strings.Join(clean, ",")
	case CertFieldNumber:
		return cast.ToString(value)
	}

	if list, ok := value.([]any); ok {
		return strings.Join(cast.ToStringSlice(list), " / ")
	}

	return strings.TrimSpace(cast.ToString(value))
}

// PendingCount - 待审核认证数（后台角标用）
func (this *UserCertification) PendingCount() int64 {
	count, _ := facade.DB.Model(&UserCertification{}).Where("status", CertPending).Count()
	return count
}

// AfterFind - 查询Hook
func (this *UserCertification) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
