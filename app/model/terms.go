package model

import (
	"strings"

	"inis/app/facade"

	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// 术语迁移：聊天室 → 群聊
//
// 背景：产品口径统一把「聊天室」改叫「群聊」，代码与前端文案都已替换。
// 只改代码不够：已安装的库里还有三处旧术语是**存进数据库**的，改代码不会让它们变：
//   - auth_rules.name：接口权限规则名（后台「权限规则」列表直接展示，如「聊天室列表」）
//   - auth_pages.name：后台菜单 / 页面名（后台「页面管理」里可改名，属于系统 UI）
//   - config.remark：配置项备注（后台「网站配置」里的说明文字）
//
// 明确不动：群名称 / 群公告 / 聊天消息 / 通知历史 —— 这些是用户内容，
// 不是系统术语，替换等于篡改用户数据。
//
// 幂等：按 LIKE 捞出含旧术语的行，逐行算新值再写回，跑多少次结果都一样。
// 调用点：EnsureAuthRules（安装迁移 + 每次启动的维护任务，已有分布式锁与 panic 保护）。
func RenameChatRoomTerms() {

	if facade.DB == nil || !facade.Installed() {
		return
	}

	db := facade.DB.Drive()
	if db == nil {
		return
	}

	renamed := renameLegacyTerm(db, &AuthRules{}, "name")
	renamed += renameLegacyTerm(db, &AuthPages{}, "name")
	renamed += renameLegacyTerm(db, &Config{}, "remark")

	if renamed <= 0 || facade.Log == nil {
		return
	}

	facade.Log.Info(map[string]any{"rows": renamed}, "已把系统术语「聊天室」统一为「群聊」")

	// 规则名带缓存：中间件按 hash 缓存规则、后台列表按标签 rule 缓存，
	// 不清掉的话后台还会继续显示旧名称，直到缓存被动过期。
	facade.Cache.DelTags([]any{"rule"})
}

// renameLegacyTerm - 把某张表某一列里的「聊天室」替换成「群聊」，返回实际更新的行数
//
// 走 map 查询而不是结构体：只需要 id 与目标列，不必关心各表其它字段的差异；
// WithTrashed 让回收站里的记录也一起纠正，避免恢复后又是旧术语。
func renameLegacyTerm(db *gorm.DB, table any, column string) int {

	var rows []map[string]any
	if err := db.Model(table).Unscoped().
		Select("id", column).
		Where(column+" LIKE ?", "%聊天室%").
		Find(&rows).Error; err != nil {
		return 0
	}

	updated := 0
	for _, row := range rows {
		id := cast.ToInt(row["id"])
		old := cast.ToString(row[column])
		if id <= 0 || old == "" {
			continue
		}

		next := strings.ReplaceAll(old, "聊天室", "群聊")
		if next == old {
			continue
		}

		if err := db.Model(table).Unscoped().Where("id = ?", id).Update(column, next).Error; err != nil {
			continue
		}
		updated++
	}

	return updated
}
