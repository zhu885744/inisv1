package model

import (
	"inis/app/facade"
)

// DocsHistory - 文档历史快照
//
// 每次保存（create / update / rollback）都会写一条，正文全量存储，
// 后台可查看历史列表、对比差异并回滚到任意版本。
type DocsHistory struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	DocsId     int    `gorm:"type:int(32); comment:文档ID;" json:"docs_id"`
	Uid        int    `gorm:"type:int(32); comment:操作人ID; default:0;" json:"uid"`
	Version    string `gorm:"size:32; comment:版本; default:'latest';" json:"version"`
	Title      string `gorm:"size:256; comment:标题; default:Null;" json:"title"`
	Content    string `gorm:"type:longtext; comment:Markdown 原文;" json:"content"`
	Summary    string `gorm:"size:512; comment:变更摘要; default:Null;" json:"summary"`
	CreateTime int64  `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

// InitDocsHistory - 初始化 DocsHistory 表
func InitDocsHistory() {
	if err := facade.DB.Drive().AutoMigrate(&DocsHistory{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "DocsHistory表迁移失败")
	}
}
