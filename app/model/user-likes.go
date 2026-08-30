package model

import (
	"errors"
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

type UserLikes struct {
	Id         int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int    `gorm:"type:int(32); comment:用户ID; uniqueIndex:uk_uid_target;" json:"uid"`
	TargetType string `gorm:"type:varchar(32); comment:目标类型(article/page/moment/comment/user); uniqueIndex:uk_uid_target;" json:"target_type"`
	TargetId   int    `gorm:"type:int(32); comment:目标ID; uniqueIndex:uk_uid_target;" json:"target_id"`
	Json       any    `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any    `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any    `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64  `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

func InitUserLikes() {
	err := facade.DB.Drive().AutoMigrate(&UserLikes{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserLikes表迁移失败")
		return
	}
}

func (this *UserLikes) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func (this *UserLikes) result() (result map[string]any) {
	var author any
	wg := sync.WaitGroup{}
	wg.Add(1)

	go this.author(&wg, &author)

	wg.Wait()

	return map[string]any{
		"author": author,
	}
}

func (this *UserLikes) author(wg *sync.WaitGroup, result *any) {
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
	case "comment":
		var comment Comment
		facade.DB.Model(&Comment{}).Where("id", this.TargetId).Find(&comment)
		uid = comment.Uid
	case "user":
		uid = this.TargetId
	default:
		return
	}

	if uid > 0 {
		user, _ := facade.DB.Model(&Users{}).Find(uid)
		*result = utils.Map.WithField(user, []string{"id", "nickname", "avatar", "description", "json"})
	}
}

func (this *UserLikes) AfterCreate(tx *gorm.DB) (err error) {
	return
}

func (this *UserLikes) handleLikeExp() {
	var authorId int
	var expType string

	switch this.TargetType {
	case "article":
		var article Article
		facade.DB.Model(&Article{}).Where("id", this.TargetId).Find(&article)
		authorId = article.Uid
		expType = "article-like"
	case "page":
		var page Pages
		facade.DB.Model(&Pages{}).Where("id", this.TargetId).Find(&page)
		authorId = page.Uid
		expType = "article-like"
	case "moments":
		var moment Moments
		facade.DB.Model(&Moments{}).Where("id", this.TargetId).Find(&moment)
		authorId = moment.Uid
		expType = "article-like"
	case "comment":
		var comment Comment
		facade.DB.Model(&Comment{}).Where("id", this.TargetId).Find(&comment)
		authorId = comment.Uid
		expType = "comment-like"
	default:
		return
	}

	if authorId > 0 && authorId != this.Uid {
		(&EXP{}).Add(EXP{
			Uid:         authorId,
			Type:        expType,
			BindType:    this.TargetType,
			BindId:      this.TargetId,
			Description: expType + "奖励",
		})
	}
}

func (this *UserLikes) Like(uid, targetId int, targetType string) (err error) {
	if uid <= 0 || targetId <= 0 || targetType == "" {
		return errors.New("参数错误")
	}

	exists, _ := facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()

	if exists > 0 {
		return errors.New("已经点赞过了")
	}

	tx, err := facade.DB.Model(&UserLikes{}).Create(&UserLikes{
		Uid:        uid,
		TargetType: targetType,
		TargetId:   targetId,
	})

	if err == nil && tx.RowsAffected > 0 && targetType == "moments" {
		facade.DB.Model(&Moments{}).
			Where("id", targetId).
			UpdateColumn("likes", gorm.Expr("likes + 1"))
	}

	return
}

func (this *UserLikes) Unlike(uid, targetId int, targetType string) (err error) {
	if uid <= 0 || targetId <= 0 || targetType == "" {
		return errors.New("参数错误")
	}

	tx, err := facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Delete()

	if err == nil && tx.RowsAffected > 0 && targetType == "moments" {
		facade.DB.Model(&Moments{}).
			Where("id", targetId).
			UpdateColumn("likes", gorm.Expr("GREATEST(likes - 1, 0)"))
	}

	return
}

func (this *UserLikes) IsLiked(uid, targetId int, targetType string) bool {
	count, _ := facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()
	return count > 0
}

func (this *UserLikes) GetLikesByUid(uid int, targetType string) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]UserLikes{}).
		Where("uid", uid)

	if targetType != "" {
		query = query.Where("target_type", targetType)
	}

	count, _ := query.Count()
	data, _ := query.Order("create_time DESC").Select()
	return data, count
}

func (this *UserLikes) GetLikesCount(targetId int, targetType string) int64 {
	count, _ := facade.DB.Model(&UserLikes{}).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Count()
	return count
}

func (this *UserLikes) GetUserLikesCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Count()
	return count
}

func (this *UserLikes) GetLikesCounts(targetType string, targetIds []int) map[int]int64 {
	result := make(map[int]int64)
	if len(targetIds) == 0 || targetType == "" {
		return result
	}

	var counts []struct {
		TargetId int   `gorm:"column:target_id"`
		Count    int64 `gorm:"column:count"`
	}

	facade.DB.Drive().Model(&UserLikes{}).
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
