package model

import (
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 封禁状态常量
const (
	BanStatusActive         = 0 // 生效中
	BanStatusExpired        = 1 // 已解封（到期自动）
	BanStatusRevoked        = 2 // 已撤销（管理员手动）
	BanStatusAppealed       = 3 // 申诉中
	BanStatusAppealApproved = 4 // 申诉通过
	BanStatusAppealRejected = 5 // 申诉驳回
)

// BanStatusRestricted 该封禁记录是否让用户继续处于「被封禁」状态
//
// **申诉中(3) 与 申诉驳回(5) 都算**：提交申诉、申诉被驳回都不会解除封禁，
// 只有 已解封(1)、已撤销(2)、申诉通过(4) 才是真正恢复。
//
// 判定口径统一走这里 —— 历史上前台/登录/中间件各处都只认「生效中(0)」，
// 结果用户一提交申诉，封禁标识与登录限制就凭空消失了。
func BanStatusRestricted(status int) bool {
	switch status {
	case BanStatusActive, BanStatusAppealed, BanStatusAppealRejected:
		return true
	}
	return false
}

// BanStatusText 封禁记录状态文案（提示语 / 站内消息 / 后台展示共用）
func BanStatusText(status int) string {
	switch status {
	case BanStatusActive:
		return "封禁中"
	case BanStatusExpired:
		return "已解封"
	case BanStatusRevoked:
		return "已撤销"
	case BanStatusAppealed:
		return "申诉中"
	case BanStatusAppealApproved:
		return "申诉通过"
	case BanStatusAppealRejected:
		return "申诉驳回"
	}
	return "未知"
}

// 封禁类型位掩码常量
const (
	BanTypeLogin       = 1 << 0                                                                              // 限制登录
	BanTypeContent     = 1 << 1                                                                              // 限制发表内容
	BanTypeComment     = 1 << 2                                                                              // 限制评论
	BanTypeUpload      = 1 << 3                                                                              // 限制上传
	BanTypeInteraction = 1 << 4                                                                              // 限制互动（点赞、收藏、关注）
	BanTypeAll         = BanTypeLogin | BanTypeContent | BanTypeComment | BanTypeUpload | BanTypeInteraction // 全面封禁
)

// BanTypeMap 封禁类型中文映射
var BanTypeMap = map[int]string{
	BanTypeLogin:       "限制登录",
	BanTypeContent:     "限制发表内容",
	BanTypeComment:     "限制评论",
	BanTypeUpload:      "限制上传",
	BanTypeInteraction: "限制互动",
}

// BanTypeOrder 位掩码的展示顺序（直接遍历 map 会让文案顺序随机）
var BanTypeOrder = []int{BanTypeLogin, BanTypeContent, BanTypeComment, BanTypeUpload, BanTypeInteraction}

// BanTypeText 把封禁类型位掩码转成中文文案（如「限制登录、限制评论」）
func BanTypeText(banType int) string {
	if banType == 0 {
		return "无限制"
	}
	if banType == BanTypeAll {
		return "全面封禁"
	}

	text := ""
	for _, bit := range BanTypeOrder {
		if banType&bit == 0 {
			continue
		}
		if text != "" {
			text += "、"
		}
		text += BanTypeMap[bit]
	}

	if text == "" {
		return "账号限制"
	}

	return text
}

// BanInfoLines 封禁信息文案（邮件通知与站内消息共用，避免两处口径不一致）
//
//	原因：违反社区规定
//	限制权限：全面封禁
//	到期时间：2026-09-26 17:20:17（永久封禁时为「永久」）
//	冻结时间：2026-09-25 17:20:17
func BanInfoLines(banType int, reason string, expiresAt int64, banAt int64) []string {

	expireText := "永久"
	if expiresAt > 0 {
		expireText = time.Unix(expiresAt, 0).Format("2006-01-02 15:04:05")
	}

	return []string{
		"原因：" + reason,
		"限制权限：" + BanTypeText(banType),
		"到期时间：" + expireText,
		"冻结时间：" + time.Unix(banAt, 0).Format("2006-01-02 15:04:05"),
	}
}

// UserBanRecords 用户封禁记录表
type UserBanRecords struct {
	Id              int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid             int    `gorm:"type:int(32); index; comment:被封禁用户ID;" json:"uid"`
	OperatorId      int    `gorm:"type:int(32); comment:操作人ID;" json:"operator_id"`
	BanType         int    `gorm:"type:int(32); default:31; comment:封禁类型位掩码（默认全封禁）;" json:"ban_type"`
	Reason          string `gorm:"size:512; comment:封禁原因;" json:"reason"`
	Evidence        string `gorm:"size:1024; comment:封禁证据;" json:"evidence"`
	Duration        int    `gorm:"type:int(32); default:0; comment:封禁时长（天），0=永久;" json:"duration"`
	BanTime         int64  `gorm:"comment:封禁时间;" json:"ban_time"`
	ExpiresAt       int64  `gorm:"comment:解封时间;" json:"expires_at"`
	UnbanTime       int64  `gorm:"comment:实际解封时间;" json:"unban_time"`
	ViolationNum    int    `gorm:"type:int(32); default:1; comment:违规次数;" json:"violation_num"`
	Status          int    `gorm:"tinyint; default:0; comment:封禁状态（0生效中 1已解封 2已撤销 3申诉中 4申诉通过 5申诉驳回）;" json:"status"`
	DeleteContent   int    `gorm:"tinyint; default:0; comment:是否删除用户全部内容（0否 1是）;" json:"delete_content"`
	BanAppeal       int    `gorm:"tinyint; default:0; comment:是否禁止申诉（0允许 1禁止）;" json:"ban_appeal"`
	FreezeUser      int    `gorm:"tinyint; default:0; comment:是否冻结用户（0正常 1冻结）;" json:"freeze_user"`
	AppealContent   string `gorm:"size:1024; comment:申诉内容;" json:"appeal_content"`
	AppealTime      int64  `gorm:"comment:申诉时间;" json:"appeal_time"`
	AppealReply     string `gorm:"size:1024; comment:申诉回复;" json:"appeal_reply"`
	AppealReplyTime int64  `gorm:"comment:申诉回复时间;" json:"appeal_reply_time"`
	OperatorIp      string `gorm:"size:64; comment:操作人IP;" json:"operator_ip"`
	OperatorUa      string `gorm:"size:512; comment:操作人UserAgent;" json:"operator_ua"`
	// 公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitUserBanRecords - 初始化UserBanRecords表
func InitUserBanRecords() {
	err := facade.DB.Drive().AutoMigrate(&UserBanRecords{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserBanRecords表迁移失败")
		return
	}
}

// AfterFind - 查询后的钩子
func (this *UserBanRecords) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// result - 返回结果
func (this *UserBanRecords) result() (result map[string]any) {
	result = make(map[string]any)

	// 封禁用户信息（脱敏）
	if this.Uid > 0 {
		user, _ := facade.DB.Model(&Users{}).Field("id", "nickname", "avatar", "account").Find(this.Uid)
		if !utils.Is.Empty(user) {
			result["user"] = user
		}
	}

	// 操作人信息
	if this.OperatorId > 0 {
		operator, _ := facade.DB.Model(&Users{}).Field("id", "nickname", "avatar").Find(this.OperatorId)
		if !utils.Is.Empty(operator) {
			result["operator"] = operator
		}
	}

	// 解析封禁类型为可读名称
	banTypes := []map[string]any{}
	for bit, name := range BanTypeMap {
		if this.BanType&bit != 0 {
			banTypes = append(banTypes, map[string]any{
				"bit":  bit,
				"name": name,
			})
		}
	}
	result["ban_types"] = banTypes

	return
}
