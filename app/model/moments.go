package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Moments struct {
	Id          int                   `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int                   `gorm:"type:int(32); comment:用户ID; default:0;" json:"uid"`
	Content     string                `gorm:"type:longtext; comment:内容; default:Null;" json:"content"`
	Images      string                `gorm:"type:text; comment:图片; default:Null;" json:"images"`
	Location    string                `gorm:"size:256; comment:位置; default:Null;" json:"location"`
	Audit       int                   `gorm:"type:int(12); comment:审核; default:0;" json:"audit"`
	Status      int                   `gorm:"type:int(12); comment:状态 0-草稿 1-发布; default:1;" json:"status"`
	LastUpdate  int64                 `gorm:"comment:最后更新时间; default:0;" json:"last_update"`
	Json        any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text        any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result      any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime  int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	PublishTime int64                 `gorm:"comment:发布时间; default:0;" json:"publish_time"`
	UpdateTime  int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime  soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitMoments() {
	err := facade.DB.Drive().AutoMigrate(&Moments{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Moments表迁移失败")
		return
	}
}

func (this *Moments) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func (this *Moments) result() (result map[string]any) {
	author := make(map[string]any)
	allow := []string{"id", "nickname", "avatar", "description", "result", "title"}
	user := facade.DB.Model(&Users{}).Find(this.Uid)

	if !utils.Is.Empty(user) {
		author = utils.Map.WithField(user, allow)
	}

	return map[string]any{
		"author": author,
	}
}
