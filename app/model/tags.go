package model

import (
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"inis/app/facade"
)

type Tags struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name        string `gorm:"size:32; comment:话题名; default:Null;" json:"name"`
	Description string `gorm:"comment:描述; default:Null;" json:"description"`
	Avatar      string `gorm:"size:256; comment:封面; default:Null;" json:"avatar"`
	// 话题（Hashtag）扩展：**标签即话题**（解析见 topics.go），以下字段只在话题场景使用
	// 注意：这里没有 status 字段 —— 话题下架直接复用软删除（delete_time），
	// 避免 AutoMigrate 给存量标签补默认值时为 0 而被误判成「已下架」
	ModeratorUid int `gorm:"type:int(32); comment:话题主持人用户ID; default:0;" json:"moderator_uid"`
	Recommend    int `gorm:"type:tinyint(1); comment:精选 1-是 0-否; default:0;" json:"recommend"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitTags - 初始化Tags表
func InitTags() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&Tags{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Tags表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *Tags) AfterFind(tx *gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}