package model

import (
	"inis/app/facade"
)

// DocsRedirect - 文档 slug 变更后的 301 记录
//
// 页面地址形如 /docs/:version/:groupSlug/:pageSlug，
// 修改 slug（目录或页面）时写入旧路径 → 新路径，前台按旧链接访问时 301 到新地址。
type DocsRedirect struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Version    string `gorm:"size:32; comment:版本; default:'latest';" json:"version"`
	OldPath    string `gorm:"size:512; comment:旧路径;" json:"old_path"`
	NewPath    string `gorm:"size:512; comment:新路径;" json:"new_path"`
	CreateTime int64  `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

// InitDocsRedirect - 初始化 DocsRedirect 表
func InitDocsRedirect() {
	if err := facade.DB.Drive().AutoMigrate(&DocsRedirect{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "DocsRedirect表迁移失败")
	}
}
