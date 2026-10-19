package model

import (
	"inis/app/facade"
)

// DocsFeedback - 文档反馈（“这篇文档有帮助吗”）
//
// 同一用户（登录按 uid，未登录按 IP + 文档）只记一次，重复提交覆盖原值。
type DocsFeedback struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	DocsId     int    `gorm:"type:int(32); comment:文档ID;" json:"docs_id"`
	Uid        int    `gorm:"type:int(32); comment:用户ID; default:0;" json:"uid"`
	Useful     int    `gorm:"type:int(12); comment:是否有帮助 1-是 0-否; default:1;" json:"useful"`
	Ip         string `gorm:"size:64; comment:IP;" json:"ip"`
	CreateTime int64  `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

// InitDocsFeedback - 初始化 DocsFeedback 表
func InitDocsFeedback() {
	if err := facade.DB.Drive().AutoMigrate(&DocsFeedback{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "DocsFeedback表迁移失败")
	}
}
