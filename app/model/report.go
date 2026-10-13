package model

import (
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

/**
 * Report - 内容 / 用户举报
 *
 * 设计要点：
 *   1. **存内容快照**：被举报的动态 / 评论随时可能被作者删除或撤回，
 *      只存 target_id 的话管理员点进去只能看到 404，无法判断。快照在提交时抓取一次。
 *   2. **防重复 + 防刷**：同一用户对同一对象只允许一条「待处理」举报；
 *      每天提交数量有上限（见 controller 的限流常量）。
 *   3. **处理动作留痕**：handler_uid / handle_remark / handle_time / remove_content，
 *      便于审计「谁在什么时候怎么处理的」。
 *
 * 状态流转（Status）：
 *   0 待处理 → 1 已处理（属实，可选同时移除内容 / 后续可一键封禁）
 *            → 2 已驳回（不属实）
 *   举报人可在「待处理」阶段撤回（Delete 软删）。
 */
type Report struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int    `gorm:"type:int(32); index:idx_report_uid; comment:举报人ID;" json:"uid"`
	TargetType string `gorm:"type:varchar(32); index:idx_report_target; comment:举报对象类型(moment/article/comment/user);" json:"target_type"`
	TargetId   int    `gorm:"type:int(32); index:idx_report_target; comment:举报对象ID;" json:"target_id"`
	TargetUid  int    `gorm:"type:int(32); index; comment:被举报内容的作者ID;" json:"target_uid"`
	// Snapshot - 提交时抓取的内容快照（超长截断），保证原内容被删后仍可复核
	Snapshot string `gorm:"type:varchar(1024); comment:内容快照;" json:"snapshot"`
	// Images - 举报证据图片（逗号分隔的 URL，数量上限与清洗规则见 controller.reportImages）
	// 存储口径与评论图片一致（逗号分隔字符串），前台/后台统一用 reportOutput 转成数组输出
	Images string `gorm:"type:text; comment:证据图片; default:Null;" json:"images"`
	Reason   string `gorm:"type:varchar(32); comment:举报原因分类;" json:"reason"`
	Detail   string `gorm:"type:varchar(512); comment:补充说明; default:Null;" json:"detail"`
	// Status - 0 待处理 / 1 已处理 / 2 已驳回
	Status       int    `gorm:"type:int(12); index:idx_report_status; default:0; comment:状态 0待处理 1已处理 2已驳回;" json:"status"`
	HandlerUid   int    `gorm:"type:int(32); comment:处理人ID;" json:"handler_uid"`
	HandleRemark string `gorm:"type:varchar(512); comment:处理备注; default:Null;" json:"handle_remark"`
	HandleTime   int64  `gorm:"comment:处理时间; default:0;" json:"handle_time"`
	// RemoveContent - 处理时是否同时移除了被举报内容（1 是 0 否），仅作审计留痕
	RemoveContent int `gorm:"type:int(12); default:0; comment:处理时是否移除内容;" json:"remove_content"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// 举报状态
const (
	ReportPending  = 0 // 待处理
	ReportHandled  = 1 // 已处理（属实）
	ReportRejected = 2 // 已驳回（不属实）
)

// ReportStatusText - 状态文案（后台列表 / 站内通知共用）
func ReportStatusText(status int) string {
	switch status {
	case ReportPending:
		return "待处理"
	case ReportHandled:
		return "已处理"
	case ReportRejected:
		return "已驳回"
	}
	return "未知"
}

// ReportTargets - 允许举报的对象类型
// key 为存储值，value 为文案（前台选择 / 后台展示共用）
var ReportTargets = map[string]string{
	"moment":  "动态",
	"article": "文章",
	"comment": "评论",
	"user":    "用户",
}

// ReportReasons - 举报原因分类（与前端 ReportDialog 的选项一一对应）
var ReportReasons = map[string]string{
	"spam":       "垃圾广告 / 刷屏",
	"abuse":      "辱骂 / 人身攻击",
	"porn":       "色情低俗",
	"illegal":    "违法违规",
	"politics":   "敏感政治",
	"infringe":   "侵权 / 抄袭",
	"harass":     "骚扰 / 恶意私信",
	"other":      "其他",
}

// ReportTargetText - 对象类型文案
func ReportTargetText(targetType string) string {
	if text, ok := ReportTargets[targetType]; ok {
		return text
	}
	return targetType
}

// ReportReasonText - 举报原因文案
func ReportReasonText(reason string) string {
	if text, ok := ReportReasons[reason]; ok {
		return text
	}
	return reason
}

func InitReport() {
	if err := facade.DB.Drive().AutoMigrate(&Report{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Report表迁移失败")
	}
}

// Create - 提交举报
//
// 调用方（controller）需先做完参数校验与防刷判断，这里只负责落库。
func (this *Report) Create(item *Report) (*Report, error) {
	_, err := facade.DB.Model(&Report{}).Create(item)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "提交举报失败")
		return nil, err
	}
	return item, nil
}

// PendingExists - 该用户对同一对象是否已有「待处理」的举报（防重复提交）
func (this *Report) PendingExists(uid int, targetType string, targetId int) bool {
	if uid <= 0 || targetId <= 0 {
		return false
	}
	exist, _ := facade.DB.Model(&Report{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"target_type", "=", targetType},
		[]any{"target_id", "=", targetId},
		[]any{"status", "=", ReportPending},
	}).Exist()
	return exist
}

// CountTodayByUid - 该用户今天提交的举报数（防刷：每天上限见 controller）
func (this *Report) CountTodayByUid(uid int) int64 {
	if uid <= 0 {
		return 0
	}
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	count, _ := facade.DB.Model(&Report{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"create_time", ">=", todayStart},
	}).Count()
	return count
}

// PendingCount - 待处理举报总数（后台角标用）
func (this *Report) PendingCount() int64 {
	count, _ := facade.DB.Model(&Report{}).Where("status", ReportPending).Count()
	return count
}

// GetMine - 我提交的举报（分页，前台「我的举报」列表）
func (this *Report) GetMine(uid int, page, limit int) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]Report{}).Where("uid", uid)
	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("create_time desc").Select()
	return data, count
}

// AfterFind - 查询Hook
func (this *Report) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
