package model

import (
	"errors"
	"inis/app/facade"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type UserLikes struct {
	Id         int                   `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int                   `gorm:"type:int(32); comment:用户ID;" json:"uid"`
	TargetType string                `gorm:"type:varchar(32); comment:目标类型(article/page/moment/comment/user);" json:"target_type"`
	TargetId   int                   `gorm:"type:int(32); comment:目标ID;" json:"target_id"`
	Status     int                   `gorm:"type:int(12); default:1; comment:状态(1:已点赞,0:已取消);" json:"status"`
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func (this *UserLikes) TableName() string {
	return "user_likes"
}

func InitUserLikes() {
	err := facade.DB.Drive().AutoMigrate(&UserLikes{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserLikes表迁移失败")
		return
	}
	facade.DB.Drive().Exec("ALTER TABLE user_likes ADD UNIQUE INDEX uk_uid_target (uid, target_type, target_id)")
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
		*result = utils.Map.WithField(user, []string{"id", "nickname", "avatar", "description"})
	}
}

func (this *UserLikes) AfterCreate(tx *gorm.DB) (err error) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		this.handleLikeExp()
	}()
	wg.Wait()
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
	case "moment":
		var moment Moments
		facade.DB.Model(&Moments{}).Where("id", this.TargetId).Find(&moment)
		authorId = moment.Uid
		expType = "article-like"
	case "comment":
		var comment Comment
		facade.DB.Model(&Comment{}).Where("id", this.TargetId).Find(&comment)
		authorId = comment.Uid
		expType = "comment-like"
	case "user":
		authorId = this.TargetId
		expType = "user-like"
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

	var exist []UserLikes
	facade.DB.Model(&exist).
		WithTrashed().
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Select()

	if len(exist) > 0 {
		if exist[0].Status == 1 && exist[0].DeleteTime == 0 {
			return errors.New("已经点赞过了")
		}
		facade.DB.Model(&UserLikes{}).Restore(exist[0].Id)
		_, err = facade.DB.Model(&UserLikes{}).
			Where("id", exist[0].Id).
			UpdateColumn("status", 1)

		if err == nil && targetType == "moment" {
			facade.DB.Model(&Moments{}).
				Where("id", targetId).
				UpdateColumn("likes", gorm.Expr("likes + 1"))
		}
		return
	}

	if targetType == "user" {
		if !this.CheckDailyLimit(uid) {
			return errors.New("每日点赞次数已达上限")
		}
	}

	_, err = facade.DB.Model(&UserLikes{}).Create(&UserLikes{
		Uid:        uid,
		TargetType: targetType,
		TargetId:   targetId,
		Status:     1,
	})

	if err == nil && targetType == "moment" {
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

	_, err = facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Where("target_type", targetType).
		Where("target_id", targetId).
		UpdateColumn("status", 0)

	if err == nil && targetType == "moment" {
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
		Where("status", 1).
		Count()
	return count > 0
}

func (this *UserLikes) GetLikesByUid(uid int, targetType string) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]UserLikes{}).
		Where("uid", uid).
		Where("status", 1)

	if targetType != "" {
		query = query.Where("target_type", targetType)
	}

	count, _ := query.Count()
	data, _ := query.Order("create_time DESC").Select()
	return data, count
}

func (this *UserLikes) GetLikesCount(targetId int, targetType string) int64 {
	if targetType == "moment" {
		var moment Moments
		facade.DB.Model(&Moments{}).
			Where("id", targetId).
			Find(&moment)
		return moment.Likes
	}

	count, _ := facade.DB.Model(&UserLikes{}).
		Where("target_type", targetType).
		Where("target_id", targetId).
		Where("status", 1).
		Count()
	return count
}

func (this *UserLikes) GetReceivedLikesCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserLikes{}).
		Where("target_type", "user").
		Where("target_id", uid).
		Where("status", 1).
		Count()
	return count
}

func (this *UserLikes) CheckDailyLimit(uid int) bool {
	limit := int64(this.GetDailyLimit())
	count := this.GetDailyCount(uid)
	return count < limit
}

func (this *UserLikes) GetDailyLimit() int {
	expConfig := GetExpConfig()
	if config, ok := expConfig["user-like"]; ok {
		return cast.ToInt(config["daily_limit"])
	}
	return 10
}

func (this *UserLikes) GetDailyCount(uid int) int64 {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	count, _ := facade.DB.Model(&UserLikes{}).
		Where("uid", uid).
		Where("target_type", "user").
		Where("status", 1).
		Where("create_time >= ?", startOfDay).
		Count()
	return count
}

func (this *UserLikes) GetDailyInfo(uid int) facade.H {
	return facade.H{
		"limit":  this.GetDailyLimit(),
		"count":  this.GetDailyCount(uid),
		"remain": this.GetDailyLimit() - int(this.GetDailyCount(uid)),
	}
}

func (this *UserLikes) GetLikesCounts(targetType string, targetIds []int) map[int]int64 {
	result := make(map[int]int64)
	if len(targetIds) == 0 || targetType == "" {
		return result
	}

	var counts []struct {
		TargetId int   `json:"target_id"`
		Count    int64 `json:"count"`
	}

	facade.DB.Drive().Model(&UserLikes{}).
		Select("target_id, COUNT(*) as count").
		Where("target_type = ?", targetType).
		Where("target_id IN ?", targetIds).
		Where("status = 1").
		Group("target_id").
		Scan(&counts)

	for _, item := range counts {
		result[item.TargetId] = item.Count
	}

	return result
}
