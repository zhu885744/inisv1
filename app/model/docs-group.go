package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// DocsGroup - 文档目录
//
// 多级目录通过 Pid 自关联实现（无限层级）：
//   - 同一版本下，同一父级内的 Slug 唯一；
//   - Sort 升序排列，值小的在前，支持拖拽后批量更新；
//   - Status：1 启用 / 0 禁用（禁用后前台不展示，子目录与页面一并隐藏）。
type DocsGroup struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Pid         int    `gorm:"type:int(32); comment:父级ID; default:0;" json:"pid"`
	Version     string `gorm:"size:32; comment:版本; default:'latest';" json:"version"`
	Slug        string `gorm:"size:128; comment:目录标识; default:Null;" json:"slug"`
	Name        string `gorm:"size:128; comment:目录名称; default:Null;" json:"name"`
	Description string `gorm:"size:512; comment:描述; default:Null;" json:"description"`
	Icon        string `gorm:"size:64; comment:图标; default:Null;" json:"icon"`
	Sort        int    `gorm:"type:int(32); comment:排序; default:0;" json:"sort"`
	Status      int    `gorm:"type:int(12); comment:状态 1-启用 0-禁用; default:1;" json:"status"`
	// 可见性：1-公开 2-登录可见 3-指定角色组
	Visible int    `gorm:"type:int(12); comment:可见性 1-公开 2-登录 3-角色组; default:1;" json:"visible"`
	RoleIds string `gorm:"size:256; comment:可见角色组ID，逗号分隔; default:Null;" json:"role_ids"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitDocsGroup - 初始化 DocsGroup 表
func InitDocsGroup() {
	if err := facade.DB.Drive().AutoMigrate(&DocsGroup{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "DocsGroup表迁移失败")
	}
}

// AfterFind - 查询 Hook
func (this *DocsGroup) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
