package model

import (
	"errors"
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type UserFollows struct {
	Id          int                   `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int                   `gorm:"type:int(32); comment:用户ID;" json:"uid"`
	FollowUid   int                   `gorm:"type:int(32); comment:关注的用户ID;" json:"follow_uid"`
	Status      int                   `gorm:"type:int(12); default:1; comment:状态（1关注 0取消）;" json:"status"`
	Description string                `gorm:"comment:描述; default:Null;" json:"description"`
	Json        any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text        any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result      any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime  int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime  int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime  soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitUserFollows() {
	err := facade.DB.Drive().AutoMigrate(&UserFollows{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "UserFollows表迁移失败")
		return
	}
}

func (this *UserFollows) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func (this *UserFollows) result() (result map[string]any) {
	var followerUser any
	var followeeUser any
	wg := sync.WaitGroup{}
	wg.Add(2)

	go this.followUser(&wg, &followerUser, this.Uid)
	go this.followUser(&wg, &followeeUser, this.FollowUid)

	wg.Wait()

	return map[string]any{
		"follower": followerUser,
		"followee": followeeUser,
	}
}

func (this *UserFollows) followUser(wg *sync.WaitGroup, result *any, uid int) {
	defer wg.Done()

	user := make(map[string]any)
	allow := []string{"id", "nickname", "avatar", "description", "json", "result", "title", "exp"}
	item, _ := facade.DB.Model(&Users{}).Find(uid)

	if !utils.Is.Empty(item) {
		user = utils.Map.WithField(item, allow)
	}

	*result = user
}

func (this *UserFollows) Follow(uid, followUid int) error {
	if uid == followUid {
		return errors.New("不能关注自己！")
	}

	var exist []UserFollows
	facade.DB.Model(&exist).
		WithTrashed().
		Where([]any{
			[]any{"uid", "=", uid},
			[]any{"follow_uid", "=", followUid},
		}).Select()

	if len(exist) > 0 {
		facade.DB.Model(&UserFollows{}).Restore(exist[0].Id)
		_, err := facade.DB.Model(&UserFollows{}).
			Where("id", exist[0].Id).
			UpdateColumn("status", 1)
		return err
	}

	_, err := facade.DB.Model(&UserFollows{}).Create(&UserFollows{
		Uid:       uid,
		FollowUid: followUid,
		Status:    1,
	})

	return err
}

func (this *UserFollows) Unfollow(uid, followUid int) error {
	_, err := facade.DB.Model(&UserFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"follow_uid", "=", followUid},
	}).Update(map[string]any{"status": 0})
	return err
}

func (this *UserFollows) GetFollowing(uid int, page, limit int) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]UserFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"status", "=", 1},
	})
	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("create_time desc").Select()
	return data, count
}

func (this *UserFollows) GetFollowers(followUid int, page, limit int) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]UserFollows{}).Where([]any{
		[]any{"follow_uid", "=", followUid},
		[]any{"status", "=", 1},
	})
	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("create_time desc").Select()
	return data, count
}

func (this *UserFollows) IsFollowing(uid, followUid int) bool {
	exist, _ := facade.DB.Model(&UserFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"follow_uid", "=", followUid},
		[]any{"status", "=", 1},
	}).Exist()
	return exist
}

func (this *UserFollows) GetFollowingCount(uid int) int64 {
	count, _ := facade.DB.Model(&UserFollows{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"status", "=", 1},
	}).Count()
	return count
}

func (this *UserFollows) GetFollowersCount(followUid int) int64 {
	count, _ := facade.DB.Model(&UserFollows{}).Where([]any{
		[]any{"follow_uid", "=", followUid},
		[]any{"status", "=", 1},
	}).Count()
	return count
}

func (this *UserFollows) GetFollowsCounts(targetType string, targetIds []int) map[int]int64 {
	result := make(map[int]int64)
	if len(targetIds) == 0 || targetType == "" {
		return result
	}

	var counts []struct {
		TargetId int   `gorm:"column:target_id"`
		Count    int64 `gorm:"column:count"`
	}

	var field string
	if targetType == "following" {
		field = "uid"
	} else {
		field = "follow_uid"
	}

	facade.DB.Drive().Model(&UserFollows{}).
		Select(field+" as target_id, COUNT(*) as count").
		Where(field+" IN ?", targetIds).
		Where("status = 1").
		Group(field).
		Scan(&counts)

	for _, item := range counts {
		result[item.TargetId] = item.Count
	}

	return result
}
