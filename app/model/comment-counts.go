package model

import (
	"inis/app/facade"
)

// CommentCounts - 按 bind_id 批量统计评论数（含二级及更深的回复）
//
// 口径与控制器 comment/count 保持一致：只统计审核通过（audit=1）的评论，
// 保证前台展示数与实际列表一致。
//
// 为什么单独抽出来：列表页过去「每条动态各查一次 count」，一页 20 条就是 20 次往返；
// 这里用一次 GROUP BY 拿到整页的数量，调用方再按 bind_id 取用。
func CommentCounts(bindType string, bindIds []int) map[int]int64 {
	result := make(map[int]int64)
	if bindType == "" || len(bindIds) == 0 {
		return result
	}

	var rows []struct {
		BindId int   `gorm:"column:bind_id"`
		Count  int64 `gorm:"column:count"`
	}

	facade.DB.Drive().Model(&Comment{}).
		Select("bind_id, COUNT(*) as count").
		Where("bind_type = ?", bindType).
		Where("bind_id IN ?", bindIds).
		Where("audit = ?", 1).
		Group("bind_id").
		Scan(&rows)

	for _, row := range rows {
		result[row.BindId] = row.Count
	}

	return result
}
