package model

import (
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Notification struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); comment:接收用户ID;" json:"uid"`
	FromUid  int    `gorm:"type:int(32); comment:触发用户ID;" json:"from_uid"`
	Type     string `gorm:"type:varchar(32); comment:通知类型(comment/like/follow/system);" json:"type"`
	Title    string `gorm:"type:varchar(256); comment:通知标题;" json:"title"`
	Content  string `gorm:"type:varchar(1024); comment:通知内容;" json:"content"`
	BindId   int    `gorm:"type:int(32); comment:关联实体ID; default:0;" json:"bind_id"`
	BindType string `gorm:"type:varchar(32); comment:关联实体类型;" json:"bind_type"`
	IsRead   int    `gorm:"type:int(2); default:0; comment:是否已读 0未读 1已读;" json:"is_read"`
	// 公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitNotification() {
	err := facade.DB.Drive().AutoMigrate(&Notification{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Notification表迁移失败")
		return
	}
	// 创建索引
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_uid ON notifications(uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read)")
}

func (this *Notification) AfterFind(tx *gorm.DB) (err error) {
	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func (this *Notification) result() (result map[string]any) {
	var fromUser any
	wg := sync.WaitGroup{}
	wg.Add(1)

	go this.fromUserSync(&wg, &fromUser)

	wg.Wait()

	return map[string]any{
		"from_user": fromUser,
	}
}

func (this *Notification) fromUserSync(wg *sync.WaitGroup, result *any) {
	defer wg.Done()

	if this.FromUid > 0 {
		user, _ := facade.DB.Model(&Users{}).Find(this.FromUid)
		*result = utils.Map.WithField(user, []string{"id", "nickname", "avatar", "description", "title"})
	}
}

// CreateNotification 创建通知并推送WebSocket
func (this *Notification) CreateNotification(uid, fromUid int, typ, title, content, bindType string, bindId int) (*Notification, error) {
	notif := &Notification{
		Uid:      uid,
		FromUid:  fromUid,
		Type:     typ,
		Title:    title,
		Content:  content,
		BindId:   bindId,
		BindType: bindType,
		IsRead:   0,
	}

	_, err := facade.DB.Model(&Notification{}).Create(notif)

	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "创建通知失败")
		return nil, err
	}

	return notif, nil
}

// GetUnreadCount 获取用户未读通知数量
func (this *Notification) GetUnreadCount(uid int) int64 {
	count, _ := facade.DB.Model(&Notification{}).
		Where("uid", uid).
		Where("is_read", 0).
		Count()
	return count
}

// MarkRead 标记单条通知为已读
func (this *Notification) MarkRead(uid, id int) error {
	_, err := facade.DB.Model(&Notification{}).
		Where("id", id).
		Where("uid", uid).
		Update(map[string]any{"is_read": 1})
	return err
}

// MarkAllRead 标记用户所有通知为已读
func (this *Notification) MarkAllRead(uid int) error {
	_, err := facade.DB.Model(&Notification{}).
		Where("uid", uid).
		Where("is_read", 0).
		Update(map[string]any{"is_read": 1})
	return err
}

// GetNotifications 获取用户通知列表
func (this *Notification) GetNotifications(uid int, typ string, isRead int, page, limit int, order string) ([]map[string]any, int64) {
	query := facade.DB.Model(&[]Notification{}).Where("uid", uid)

	if typ != "" {
		query = query.Where("type", typ)
	}
	if isRead >= 0 {
		query = query.Where("is_read", isRead)
	}

	count, _ := query.Count()

	if order == "" {
		order = "create_time desc"
	}

	data, _ := query.Limit(limit).Page(page).Order(order).Select()

	return data, count
}
