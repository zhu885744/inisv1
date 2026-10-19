package model

import (
	"strings"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// DefaultDocsProjectSlug - 内置默认项目的别名。
//
// 沿用历史数据的 version='latest'：迁移时把存量文档挂到这个项目下，
// 老链接 /docs/start/install 这类不带项目名的地址仍能命中。
const DefaultDocsProjectSlug = "latest"

// DocsProject - 文档项目
//
// 一个项目 = 一套完全独立的文档站，对外通过 /docs/:slug 访问，slug 即 URL 里的项目名。
//
// 设计要点：项目本身不存文档，只负责「命名空间 + 展示信息」：
//   - 文档与目录仍存docs / docs_group 表，用它们的 version 字段存本表的 Slug；
//   - 因此新增项目不需要动任何文档表结构，也不用改现有接口的 version 参数；
//   - 项目的 CommentEnabled 控制该项目下所有文档页能否评论（对应后台「启用评论」）。
type DocsProject struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name        string `gorm:"size:128; comment:项目名称; default:Null;" json:"name"`
	Slug        string `gorm:"size:64; index; comment:项目别名，即URL里的项目名; default:Null;" json:"slug"`
	Description string `gorm:"size:512; comment:描述; default:Null;" json:"description"`
	Icon        string `gorm:"size:64; comment:图标; default:Null;" json:"icon"`
	Sort        int    `gorm:"type:int(32); comment:排序; default:0;" json:"sort"`
	Status      int    `gorm:"type:int(12); comment:状态 1-启用 0-禁用; default:1;" json:"status"`
	// 可见性：1-公开 2-登录可见 3-指定角色组（控制项目能否在前台项目列表里出现）
	Visible int    `gorm:"type:int(12); comment:可见性 1-公开 2-登录 3-角色组; default:1;" json:"visible"`
	RoleIds string `gorm:"size:256; comment:可见角色组ID，逗号分隔; default:Null;" json:"role_ids"`
	// 启用评论：0-禁止 1-允许（作用于该项目下的所有文档页）
	CommentEnabled int `gorm:"type:int(12); comment:启用评论 0-禁止 1-允许; default:1;" json:"comment_enabled"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitDocsProject - 初始化 DocsProject 表
func InitDocsProject() {
	if err := facade.DB.Drive().AutoMigrate(&DocsProject{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "DocsProject表迁移失败")
		return
	}

	migrateDocsProjects()
}

// migrateDocsProjects - 存量兼容（幂等，可重复执行）
//
//  1. 表为空时建一个 slug=latest 的默认项目；
//  2. docs / docs_group 里出现过的 version 若还没有对应项目，自动补建（沿用原 version 名），
//     这样历史版本的数据不会因为没有项目记录而在前台消失。
func migrateDocsProjects() {
	exist, _ := facade.DB.Model(&DocsProject{}).Exist()
	if !exist {
		_, _ = facade.DB.Model(&DocsProject{}).Create(&DocsProject{
			Name:           "默认文档",
			Slug:           DefaultDocsProjectSlug,
			Description:    "系统内置文档项目",
			Icon:           "bi bi-journal-bookmark",
			Sort:           1,
			Status:         1,
			Visible:        1,
			CommentEnabled: 1,
		})
	}

	rows, _ := facade.DB.Model(&DocsProject{}).Column("slug")
	known := make(map[string]bool)
	for _, item := range cast.ToStringSlice(rows) {
		known[strings.TrimSpace(item)] = true
	}

	// 收集文档 / 目录里出现过但尚未建项目的 version
	missing := make(map[string]bool)
	for _, target := range []any{&Docs{}, &DocsGroup{}} {
		list, _ := facade.DB.Model(target).Column("version")
		for _, item := range cast.ToStringSlice(list) {
			item = strings.TrimSpace(item)
			if item != "" && !known[item] {
				missing[item] = true
			}
		}
	}

	for slug := range missing {
		_, _ = facade.DB.Model(&DocsProject{}).Create(&DocsProject{
			Name:           slug,
			Slug:           slug,
			Description:    "由历史数据自动创建",
			Icon:           "bi bi-journal-bookmark",
			Status:         1,
			Visible:        1,
			CommentEnabled: 1,
		})
	}
}

// DocsProjectBySlug - 按别名查项目，查不到返回零值
func DocsProjectBySlug(slug string) DocsProject {
	var item DocsProject
	if strings.TrimSpace(slug) == "" {
		return item
	}
	facade.DB.Model(&DocsProject{}).Where("slug", slug).Scan(&item)
	return item
}

// DocsProjectCommentEnabled - 该项目是否允许评论。
// 项目不存在（脏数据 / 历史 version）时按「允许」处理，避免老内容莫名其妙没有评论区。
func DocsProjectCommentEnabled(slug string) bool {
	item := DocsProjectBySlug(slug)
	if item.Id == 0 {
		return true
	}
	return item.CommentEnabled == 1
}

// AfterFind - 查询 Hook
func (this *DocsProject) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}