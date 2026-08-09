package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 封禁状态常量
const (
	BanStatusActive   = 0 // 生效中
	BanStatusExpired  = 1 // 已解封（到期自动）
	BanStatusRevoked  = 2 // 已撤销（管理员手动）
	BanStatusAppealed = 3 // 申诉中
	BanStatusAppealApproved = 4 // 申诉通过
	BanStatusAppealRejected = 5 // 申诉驳回
)

// 封禁类型位掩码常量
const (
	BanTypeLogin       = 1 << 0 // 限制登录
	BanTypeContent     = 1 << 1 // 限制发表内容
	BanTypeComment     = 1 << 2 // 限制评论
	BanTypeUpload      = 1 << 3 // 限制上传
	BanTypeInteraction = 1 << 4 // 限制互动（点赞、收藏、关注）
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

// UserBanRecords 用户封禁记录表
type UserBanRecords struct {
	Id           int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid          int    `gorm:"type:int(32); index; comment:被封禁用户ID;" json:"uid"`
	OperatorId   int    `gorm:"type:int(32); comment:操作人ID;" json:"operator_id"`
	BanType      int    `gorm:"type:int(32); default:31; comment:封禁类型位掩码（默认全封禁）;" json:"ban_type"`
	Reason       string `gorm:"size:512; comment:封禁原因;" json:"reason"`
	Evidence     string `gorm:"size:1024; comment:封禁证据;" json:"evidence"`
	Duration     int    `gorm:"type:int(32); default:0; comment:封禁时长（天），0=永久;" json:"duration"`
	BanTime      int64  `gorm:"comment:封禁时间;" json:"ban_time"`
	ExpiresAt    int64  `gorm:"comment:解封时间;" json:"expires_at"`
	UnbanTime    int64  `gorm:"comment:实际解封时间;" json:"unban_time"`
	ViolationNum int    `gorm:"type:int(32); default:1; comment:违规次数;" json:"violation_num"`
	Status       int    `gorm:"tinyint; default:0; comment:封禁状态（0生效中 1已解封 2已撤销 3申诉中 4申诉通过 5申诉驳回）;" json:"status"`
	AppealContent string    `gorm:"size:1024; comment:申诉内容;" json:"appeal_content"`
	AppealTime    int64     `gorm:"comment:申诉时间;" json:"appeal_time"`
	AppealReply   string    `gorm:"size:1024; comment:申诉回复;" json:"appeal_reply"`
	AppealReplyTime int64   `gorm:"comment:申诉回复时间;" json:"appeal_reply_time"`
	OperatorIp     string   `gorm:"size:64; comment:操作人IP;" json:"operator_ip"`
	OperatorUa     string   `gorm:"size:512; comment:操作人UserAgent;" json:"operator_ua"`
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
