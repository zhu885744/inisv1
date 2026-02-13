package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Upgrade struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Version string `gorm:"type:varchar(32); comment:版本号;" json:"version"`
	Type    string `gorm:"type:varchar(32); comment:类型;" json:"type"` // type: app（程序更新）、theme（主题更新）
	Content string `gorm:"type:longtext; comment:更新内容;" json:"content"`
	Url     string `gorm:"type:varchar(256); comment:更新地址;" json:"url"`
	Status  int    `gorm:"type:int(32); comment:状态;" json:"status"` // status: 0（禁用）、1（启用）
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitUpgrade - 初始化Upgrade表
func InitUpgrade() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&Upgrade{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Upgrade表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *Upgrade) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// result - 返回结果
func (this *Upgrade) result() (result map[string]any) {
	return map[string]any{}
}