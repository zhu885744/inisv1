package model

import (
	"errors"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

const (
	TagFollowStatusOn  = 1 // 关注中
	TagFollowStatusOff = 0 // 已取消
)

// TagFollows - 话题关注
//
// 标签即话题（见 tags.go 的 ModeratorUid / Recommend 与 topics.go 的解析逻辑），
// 关注关系与 user_follows 一样用 status 软开关（不删行），这样：
//   - 取关再关注不会撞唯一索引 uk_tag_follows_pair；
//   - 关注数可以按 status=1 直接 COUNT。
type TagFollows struct {
	Id     int `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid    int `gorm:"type:int(32); comment:用户ID; default:0; index; uniqueIndex:uk_tag_follows_pair,priority:1;" json:"uid"`
	TagId  int `gorm:"type:int(32); comment:话题ID; default:0; index; uniqueIndex:uk_tag_follows_pair,priority:2;" json:"tag_id"`
	Status int `gorm:"type:tinyint(1); comment:状态 1-关注 0-取消; default:1;" json:"status"`
	// 以下为公共字段
	Description string                `gorm:"comment:描述; default:Null;" json:"description"`
	Json        any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text        any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result      any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime  int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime  int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime  soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitTagFollows - 初始化TagFollows表
func InitTagFollows() {
	if err := facade.DB.Drive().AutoMigrate(&TagFollows{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "TagFollows表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *TagFollows) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// Follow - 关注话题（幂等：重复关注不报错、不产生第二条记录）
func (this *TagFollows) Follow(uid, tagId int) error {

	if uid <= 0 || tagId <= 0 {
		return errors.New("参数不完整！")
	}

	// 已有记录（含被软删的）：恢复 + 置为关注中，避免唯一索引冲突
	if id := this.findId(uid, tagId); id > 0 {
		return this.activate(id)
	}

	_, err := facade.DB.Model(&TagFollows{}).Create(&TagFollows{
		Uid:    uid,
		TagId:  tagId,
		Status: TagFollowStatusOn,
	})
	if err == nil {
		return nil
	}

	// 并发（连点 / 重试）时可能已被另一次请求插入：再找一次，按「已关注」处理，
	// 不把 MySQL 的 1062 原文抛给用户
	if id := this.findId(uid, tagId); id > 0 {
		return this.activate(id)
	}

	return errors.New("关注失败，请稍后重试！")
}

// Unfollow - 取消关注（幂等）
func (this *TagFollows) Unfollow(uid, tagId int) error {

	if uid <= 0 || tagId <= 0 {
		return errors.New("参数不完整！")
	}

	_, err := facade.DB.Model(&TagFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"tag_id", "=", tagId},
	}).Update(map[string]any{"status": TagFollowStatusOff})

	return err
}

// IsFollowing - 是否已关注（只算关注中的）
func (this *TagFollows) IsFollowing(uid, tagId int) bool {

	if uid <= 0 || tagId <= 0 {
		return false
	}

	exist, _ := facade.DB.Model(&TagFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"tag_id", "=", tagId},
		[]any{"status", "=", TagFollowStatusOn},
	}).Exist()

	return exist
}

// FollowingTagIds - 我关注的话题 id（最近关注的在前）
func (this *TagFollows) FollowingTagIds(uid int) []int {

	if uid <= 0 {
		return []int{}
	}

	column, _ := facade.DB.Model(&TagFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"status", "=", TagFollowStatusOn},
	}).Order("update_time desc").Column("tag_id")

	return uniqueIds(cast.ToIntSlice(column))
}

// FilterFollowingTagIds - 批量判断一批话题里哪些已被 uid 关注（列表标记用）
func (this *TagFollows) FilterFollowingTagIds(uid int, tagIds []int) map[int]bool {

	result := make(map[int]bool)
	if uid <= 0 || len(tagIds) == 0 {
		return result
	}

	rows, _ := facade.DB.Model(&[]TagFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"status", "=", TagFollowStatusOn},
		[]any{"tag_id", "IN", tagIds},
	}).Select()

	for _, row := range rows {
		result[cast.ToInt(row["tag_id"])] = true
	}

	return result
}

// GetFollowingCount - 该用户关注的话题数
func (this *TagFollows) GetFollowingCount(uid int) int64 {
	count, _ := facade.DB.Model(&TagFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"status", "=", TagFollowStatusOn},
	}).Count()
	return count
}

// GetFollowersCount - 该话题的关注数
func (this *TagFollows) GetFollowersCount(tagId int) int64 {
	count, _ := facade.DB.Model(&TagFollows{}).Where([]any{
		[]any{"tag_id", "=", tagId},
		[]any{"status", "=", TagFollowStatusOn},
	}).Count()
	return count
}

// CountByTagIds - 批量统计每个话题的关注数（热度榜一次算完，避免 N+1）
func (this *TagFollows) CountByTagIds() map[string]int {

	result := make(map[string]int)

	column, err := facade.DB.Model(&TagFollows{}).Where("status", TagFollowStatusOn).Column("tag_id")
	if err != nil || utils.Is.Empty(column) {
		return result
	}

	for _, id := range cast.ToStringSlice(column) {
		if utils.Is.Empty(id) {
			continue
		}
		result[id]++
	}

	return result
}

// findId - 查 (uid, tag_id) 这条记录的 id（**包含回收站里的**，0 表示没有）
//
// 用 Select() 取 map，而不是 Find(&struct) + 原始 SQL：facade 的 Where 是按列名解析的
// （非法列名会直接 return this 丢弃条件），传 "uid = ? AND tag_id = ?" 会被丢掉，
// 查询条件静默失效后就会走到 Create 撞唯一索引。详见 user-blocks.go 的同名方法。
func (this *TagFollows) findId(uid, tagId int) int {

	rows, _ := facade.DB.Model(&[]TagFollows{}).WithTrashed().Where([]any{
		[]any{"uid", "=", uid},
		[]any{"tag_id", "=", tagId},
	}).Select()

	if len(rows) == 0 {
		return 0
	}

	return cast.ToInt(rows[0]["id"])
}

// activate - 把（可能已在回收站的）关注记录恢复为「关注中」
func (this *TagFollows) activate(id int) error {

	// 软删过的先恢复，否则下面的 Update 找不到这一行
	if _, err := facade.DB.Model(&TagFollows{}).Restore(id); err != nil {
		return err
	}

	_, err := facade.DB.Model(&TagFollows{}).Where("id", id).Update(map[string]any{"status": TagFollowStatusOn})

	return err
}
