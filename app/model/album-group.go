package model

import (
	"inis/app/facade"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type AlbumGroup struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int    `gorm:"type:int(32); default:0; comment:用户ID;" json:"uid"`
	Name        string `gorm:"type:varchar(32); comment:昵称; default:Null;" json:"name"`
	Description string `gorm:"type:varchar(256); comment:描述; default:Null;" json:"description"`
	Covers      string `gorm:"type:text; comment:封面; default:Null;" json:"covers"`
	Remark      string `gorm:"type:varchar(256); comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitAlbumGroup - 初始化AlbumGroup表
func InitAlbumGroup() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&AlbumGroup{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "AlbumGroup表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *AlbumGroup) AfterFind(tx *gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}