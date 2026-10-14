package model

import (
	"errors"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 话题内置顶的内容类型
const (
	TopicTargetMoments = "moments"
	TopicTargetArticle = "article"
)

// TopicPins - 话题内的置顶
//
// 与内容自身的 top 字段区分开：内容只能全局置顶一次，
// 而一条内容可能同时属于多个话题，每个话题的主持人各自决定在自己话题里置顶哪几条，
// 所以这里是一张 (话题, 内容) 的关系表。
type TopicPins struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	TagId      int    `gorm:"type:int(32); comment:话题ID; default:0; index; uniqueIndex:uk_topic_pins_target,priority:1;" json:"tag_id"`
	TargetType string `gorm:"size:32; comment:内容类型 moments/article; default:Null; uniqueIndex:uk_topic_pins_target,priority:2;" json:"target_type"`
	TargetId   int    `gorm:"type:int(32); comment:内容ID; default:0; uniqueIndex:uk_topic_pins_target,priority:3;" json:"target_id"`
	Uid        int    `gorm:"type:int(32); comment:操作人; default:0;" json:"uid"`
	// 以下为公共字段
	Description string                `gorm:"comment:描述; default:Null;" json:"description"`
	Json        any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text        any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result      any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime  int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime  int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime  soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitTopicPins - 初始化TopicPins表
func InitTopicPins() {
	if err := facade.DB.Drive().AutoMigrate(&TopicPins{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "TopicPins表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *TopicPins) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// Pin - 在话题内置顶一条内容（幂等）
func (this *TopicPins) Pin(tagId int, targetType string, targetId, uid int) error {

	if tagId <= 0 || targetId <= 0 {
		return errors.New("参数不完整！")
	}
	if targetType != TopicTargetMoments && targetType != TopicTargetArticle {
		return errors.New("内容类型不支持！")
	}

	// 已有记录（含被软删的）：恢复，避免撞唯一索引（同 TagFollows.Pin 的处理）
	if id := this.findId(tagId, targetType, targetId); id > 0 {
		if _, err := facade.DB.Model(&TopicPins{}).Restore(id); err != nil {
			return err
		}
		return nil
	}

	_, err := facade.DB.Model(&TopicPins{}).Create(&TopicPins{
		TagId:      tagId,
		TargetType: targetType,
		TargetId:   targetId,
		Uid:        uid,
	})
	if err == nil {
		return nil
	}

	// 并发（连点）：再找一次，找到就按「已置顶」处理
	if id := this.findId(tagId, targetType, targetId); id > 0 {
		_, err = facade.DB.Model(&TopicPins{}).Restore(id)
		return err
	}

	return errors.New("置顶失败，请稍后重试！")
}

// Unpin - 取消话题内置顶（幂等）
func (this *TopicPins) Unpin(tagId int, targetType string, targetId int) error {

	if tagId <= 0 || targetId <= 0 {
		return errors.New("参数不完整！")
	}

	_, err := facade.DB.Model(&TopicPins{}).Where([]any{
		[]any{"tag_id", "=", tagId},
		[]any{"target_type", "=", targetType},
		[]any{"target_id", "=", targetId},
	}).Delete()

	return err
}

// PinnedIds - 该话题内置顶的内容 id（最近置顶的在前）
func (this *TopicPins) PinnedIds(tagId int, targetType string) []int {

	if tagId <= 0 {
		return []int{}
	}

	column, _ := facade.DB.Model(&TopicPins{}).Where([]any{
		[]any{"tag_id", "=", tagId},
		[]any{"target_type", "=", targetType},
	}).Order("create_time desc").Column("target_id")

	return uniqueIds(cast.ToIntSlice(column))
}

// FilterPinned - 批量判断一批内容里哪些在该话题内被置顶（列表标记用）
func (this *TopicPins) FilterPinned(tagId int, targetType string, targetIds []int) map[int]bool {

	result := make(map[int]bool)
	if tagId <= 0 || len(targetIds) == 0 {
		return result
	}

	rows, _ := facade.DB.Model(&[]TopicPins{}).Where([]any{
		[]any{"tag_id", "=", tagId},
		[]any{"target_type", "=", targetType},
		[]any{"target_id", "IN", targetIds},
	}).Select()

	for _, row := range rows {
		result[cast.ToInt(row["target_id"])] = true
	}

	return result
}

// ClearTarget - 清掉某条内容在所有话题里的置顶（内容被移出话题 / 删除时调用）
func (this *TopicPins) ClearTarget(targetType string, targetId int) {
	if targetId <= 0 {
		return
	}
	_, _ = facade.DB.Model(&TopicPins{}).Where([]any{
		[]any{"target_type", "=", targetType},
		[]any{"target_id", "=", targetId},
	}).Delete()
}

// findId - 查 (话题, 类型, 内容) 这条记录的 id（含回收站，0 表示没有）
func (this *TopicPins) findId(tagId int, targetType string, targetId int) int {

	rows, _ := facade.DB.Model(&[]TopicPins{}).WithTrashed().Where([]any{
		[]any{"tag_id", "=", tagId},
		[]any{"target_type", "=", targetType},
		[]any{"target_id", "=", targetId},
	}).Select()

	if len(rows) == 0 {
		return 0
	}

	return cast.ToInt(rows[0]["id"])
}
