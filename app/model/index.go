package model

import "inis/app/facade"

/**
 * 索引补齐工具
 *
 * 为什么不用 `CREATE INDEX IF NOT EXISTS`：那是 MariaDB 的语法，MySQL 会直接
 * 报语法错误；而 `Exec` 的返回值不检查时错误被静默吞掉，结果就是「代码里写了
 * 建索引、线上却没有索引」（通知 / 积分表就踩过这个坑，表现为数据量上来后
 * 列表与未读数统计全表扫）。
 *
 * 这里改成「先查 information_schema、确认不存在再建」，既能重复执行，
 * 也能把失败原因写进日志，不至于再次静默失败。
 *
 * 只用于 AutoMigrate 无法表达的复合索引 / 存量表补索引；
 * 单列索引仍优先写在结构体 tag 上（见 goods.go 的说明）。
 */

// EnsureIndex - 幂等地确保某个索引存在
//
// 参数：
//   - table：表名（如 inis_notification）
//   - name：索引名（用于查重，必须与 ddl 里的一致）
//   - ddl：完整建索引语句（如 CREATE INDEX idx_x ON inis_notification(uid, create_time)）
func EnsureIndex(table, name, ddl string) {

	if table == "" || name == "" || ddl == "" {
		return
	}

	var count int64
	err := facade.DB.Drive().Raw(
		"SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?",
		table, name,
	).Scan(&count).Error

	// 查得到且已存在：什么都不做（初始化每次启动都会跑，必须幂等）
	if err == nil && count > 0 {
		return
	}

	// 查询失败时不阻断建索引：最坏情况是遇到「索引已存在」的 1061，
	// 记一条日志即可，不影响启动
	if err := facade.DB.Drive().Exec(ddl).Error; err != nil {
		facade.Log.Error(map[string]any{
			"table": table,
			"index": name,
			"error": err.Error(),
		}, "创建索引失败")
	}
}
