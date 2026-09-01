package model

import (
	"errors"
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

type UserCollects struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int    `gorm:"type:int(32); comment:用户ID; uniqueIndex:uk_uid_target;" json:"uid"`
	TargetType string `gorm:"type:varchar(32); comment:目标类型(article/page/moment); uniqueIndex:uk_uid_target;" json:"target_type"`
	TargetId   int    `gorm:"type:int(32); comment:目标ID; uniqueIndex:uk_uid_target;" json:"target_id"`
	Json       any    `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any    `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any    `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64  `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

func InitUserCollects() {
	err := facade.DB.Drive().AutoMigrate(&UserCollects{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserCollects表迁移失败")
		return
	}
}

func (this *UserCollects) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func (this *UserCollects) result() (result map[string]any) {
	var author any
	wg := sync.WaitGroup{}
	wg.Add(1)

	go this.author(&wg, &author)

	wg.Wait()

	return map[string]any{
		"author": author,
	}
}

func (this *UserCollects) author(wg *sync.WaitGroup, result *any) {
	defer wg.Done()

	var uid int
	switch this.TargetType {
	case "article":
		var article Article
		facade.DB.Model(&Article{}).Where("id", this.TargetId).Find(&article)
		uid = article.Uid
	case "page":
		var page Pages
		facade.DB.Model(&Pages{}).Where("id", this.TargetId).Find(&page)
		uid = page.Uid
	case "moment":
		var moment Moments
		facade.DB.Model(&Moments{}).Where("id", this.TargetId).Find(&moment)
		uid = moment.Uid
	default:
		return
	}

	if uid > 0 {
		user, _ := facade.DB.Model(&Users{}).Find(uid)
		*result = utils.Map.WithField(user, []string{"id", "nickname", "avatar", "description", "json"})
	}
}

func (this *UserCollects) AfterCreate(tx *gorm.DB) (err error) {
	return
}

func (this *UserCollects) handleCollectExp() {
	var authorId int

	switch this.TargetType {
	case "article":
		var article Article
		facade.DB.Model(&Article{}).Where("id", this.TargetId).Find(&article)
		authorId = article.Uid
	case "page":
		var page Pages
		facade.DB.Model(&Pages{}).Where("id", this.TargetId).Find(&page)
		authorId = page.Uid
	case "moments":
		var moment Moments
		facade.DB.Model(&Moments{}).Where("id", this.TargetId).Find(&moment)
		authorId = moment.Uid
	default:
		return
	}

	if authorId > 0 && authorId != this.Uid {
		(&EXP{}).Add(EXP{
			Uid:         authorId,
			Type:        "article-collect",
			BindType:    this.TargetType,
			BindId:      this.TargetId,
			Description: "内容被收藏奖励",
		})
	}
}

func (this *UserCollects) Collect(uid, targetId int, targetType string) (err error) {
	if uid <= 0 || targetId <= 0 || targetType == "" {
		return errors.New("参数错误")
	}

	if targetType == "user" {
		return errors.New("不支持收藏用户")
	}

	exists, _ := facade.DB.Model(&UserCollects{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()

	if exists > 0 {
		return errors.New("已经收藏过了")
	}

	_, err = facade.DB.Model(&UserCollects{}).Create(&UserCollects{
		Uid:        uid,
		TargetType: targetType,
		TargetId:   targetId,
	})

	return
}

func (this *UserCollects) Uncollect(uid, targetId int, targetType string) (err error) {
	if uid <= 0 || targetId <= 0 || targetType == "" {
		return errors.New("参数错误")
	}

	_, err = facade.DB.Model(&UserCollects{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Delete()

	return
}

func (this *UserCollects) IsCollected(uid, targetId int, targetType string) bool {
	count, _ := facade.DB.Model(&UserCollects{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()
	return count > 0
}

func (this *UserCollects) GetCollectsByUid(uid int, targetType string) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]UserCollects{}).
		Where("uid", uid)

	if targetType != "" {
		query = query.Where("target_type", targetType)
	}

	count, _ := query.Count()
	data, _ := query.Order("create_time DESC").Select()
	return data, count
}

func (this *UserCollects) GetCollectsCount(targetId int, targetType string) int64 {
	count, _ := facade.DB.Model(&UserCollects{}).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()
	return count
}

func (this *UserCollects) GetReceivedCollectsCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserCollects{}).
		Where("target_type", "user").
		Where("target_id", uid).
		Count()
	return count
}

func (this *UserCollects) GetUserCollectsCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserCollects{}).
		Where("uid", uid).
		Count()
	return count
}

func (this *UserCollects) GetCollectsCounts(targetType string, targetIds []int) map[int]int64 {
	result := make(map[int]int64)
	if len(targetIds) == 0 || targetType == "" {
		return result
	}

	var counts []struct {
		TargetId int   `gorm:"column:target_id"`
		Count    int64 `gorm:"column:count"`
	}

	facade.DB.Drive().Model(&UserCollects{}).
		Select("target_id, COUNT(*) as count").
		Where("target_type = ?", targetType).
		Where("target_id IN ?", targetIds).
		Group("target_id").
		Scan(&counts)

	for _, item := range counts {
		result[item.TargetId] = item.Count
	}

	return result
}
