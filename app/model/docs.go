package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// Docs - 文档页面
//
// 与文章的区别：
//   - 归属于目录（Gid），同一版本内按 Sort 排序；
//   - 正文只存 Markdown 原文，由前端（markdown-it）渲染，服务端不再缓存 HTML；
//   - Status：0 草稿 / 1 已发布（文档不走审核流程）；
//   - 支持多版本（Version），/docs 默认展示配置里的默认版本。
type Docs struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Gid     int    `gorm:"type:int(32); comment:目录ID; default:0;" json:"gid"`
	Uid     int    `gorm:"type:int(32); comment:作者ID; default:0;" json:"uid"`
	Version string `gorm:"size:32; comment:版本; default:'latest';" json:"version"`
	Slug    string `gorm:"size:128; comment:页面标识; default:Null;" json:"slug"`
	Title   string `gorm:"size:256; comment:标题; default:Null;" json:"title"`
	Content string `gorm:"type:longtext; comment:Markdown;" json:"content"`
	Sort    int    `gorm:"type:int(32); comment:排序; default:0;" json:"sort"`
	// 状态：0 草稿 / 1 已发布
	Status int `gorm:"type:int(12); comment:状态 0-草稿 1-发布;" json:"status"`
	// 可见性：1-公开 2-登录可见 3-指定角色组
	Visible int    `gorm:"type:int(12); comment:可见性 1-公开 2-登录 3-角色组; default:1;" json:"visible"`
	RoleIds string `gorm:"size:256; comment:可见角色组ID，逗号分隔; default:Null;" json:"role_ids"`
	Views   int    `gorm:"type:int(32); comment:浏览量; default:0;" json:"views"`
	// 文档反馈统计（“这篇文档有帮助吗”）
	Useful  int `gorm:"type:int(32); comment:有帮助数; default:0;" json:"useful"`
	Useless int `gorm:"type:int(32); comment:无帮助数; default:0;" json:"useless"`
	// SEO
	SeoTitle       string `gorm:"size:256; comment:SEO标题; default:Null;" json:"seo_title"`
	SeoKeywords    string `gorm:"size:512; comment:SEO关键词; default:Null;" json:"seo_keywords"`
	SeoDescription string `gorm:"size:512; comment:SEO描述; default:Null;" json:"seo_description"`
	// 以下为公共字段
	Json        any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text        any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result      any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime  int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	PublishTime int64                 `gorm:"comment:发布时间; default:0;" json:"publish_time"`
	UpdateTime  int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime  soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitDocs - 初始化 Docs 表
func InitDocs() {
	if err := facade.DB.Drive().AutoMigrate(&Docs{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Docs表迁移失败")
		return
	}

	// 文档已下线审核流程：历史 status=2（待审核）统一转为草稿，
	// 避免这些记录在前台变成「永远不可见」的孤儿数据。
	facade.DB.Model(&Docs{}).Where("status", 2).UpdateColumn("status", 0)

	// 早期版本在服务端渲染并缓存 HTML / TOC，现已改为前端渲染；
	// 迁移时把这两列删掉（HasColumn 判断，重复执行无副作用）。
	migrator := facade.DB.Drive().Migrator()

	if migrator.HasColumn(&Docs{}, "content_html") {
		if err := migrator.DropColumn(&Docs{}, "content_html"); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "删除 docs.content_html 列失败")
		}
	}
	if migrator.HasColumn(&Docs{}, "toc") {
		if err := migrator.DropColumn(&Docs{}, "toc"); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "删除 docs.toc 列失败")
		}
	}
}

// AfterFind - 查询 Hook
func (this *Docs) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}
