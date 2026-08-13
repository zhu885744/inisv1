package model

import (
	"fmt"
	"inis/app/facade"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/soft_delete"
)

type Notification struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); comment:接收用户ID 0表示广播通知(推送给全体用户);" json:"uid"`
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
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_uid ON inis_notification(uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_type ON inis_notification(type)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON inis_notification(is_read)")
}

// NotificationRead 广播通知的用户状态表
// 说明：广播通知(uid=0)在 notifications 表只存一条记录，
// 每个用户对该广播通知的"已读/隐藏"状态记录在此表，避免为每个用户创建记录。
type NotificationRead struct {
	Id             int   `gorm:"type:int(32); comment:主键;" json:"id"`
	NotificationId int   `gorm:"type:int(32); index:idx_notification_reads_nid; uniqueIndex:uk_notification_reads_nid_uid,priority:1; comment:通知ID;" json:"notification_id"`
	Uid            int   `gorm:"type:int(32); uniqueIndex:uk_notification_reads_nid_uid,priority:2; comment:用户ID;" json:"uid"`
	IsRead         int   `gorm:"type:int(2); default:0; comment:是否已读 0未读 1已读;" json:"is_read"`
	IsDeleted      int   `gorm:"type:int(2); default:0; comment:是否隐藏 0否 1是;" json:"is_deleted"`
	CreateTime     int64 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime     int64 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
}

func InitNotificationRead() {
	err := facade.DB.Drive().AutoMigrate(&NotificationRead{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "NotificationRead表迁移失败")
		return
	}
	// 同一用户对同一条广播通知仅保留一条状态记录（唯一索引在 AutoMigrate 中创建）
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_notification_reads_uid ON inis_notification_read(uid)")
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

// CreateBroadcastNotification 创建广播通知（uid=0 表示推送给全体用户，仅创建一条记录）
func (this *Notification) CreateBroadcastNotification(fromUid int, typ, title, content, bindType string, bindId int) (*Notification, error) {
	notif := &Notification{
		Uid:      0,
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
		facade.Log.Error(map[string]any{"error": err}, "创建广播通知失败")
		return nil, err
	}

	return notif, nil
}

// CreateLoginNotification 创建“账号登录通知”（系统消息）
// 在用户登录成功后调用，记录登录账号、时间、IP、设备等信息，提醒用户确认是否为本人操作。
func (this *Notification) CreateLoginNotification(uid int, account, ip, ua string) (*Notification, error) {
	now := time.Now()
	loginTime := now.Format("2006-01-02 15:04:05")

	// 去除 UA 中的换行，避免注入
	ua = strings.ReplaceAll(ua, "\n", " ")
	ua = strings.ReplaceAll(ua, "\r", " ")

	// 解析为简洁的设备描述（如：Windows 10 Chrome 144）
	device := parseDevice(ua)

	content := "请确认你的登录信息，并确保是你本人操作。\n\n" +
		"登录账号：" + account + "\n" +
		"登录时间：" + loginTime + "\n" +
		"登录IP：" + ip + "\n" +
		"登录设备：" + device + "\n\n" +
		"如非本人操作，请尽快修改密码。"

	notif := &Notification{
		Uid:      uid,
		FromUid:  uid,
		Type:     "system",
		Title:    "账号登录通知",
		Content:  content,
		BindType: "user",
		BindId:   uid,
		IsRead:   0,
		Json: utils.Json.Encode(map[string]any{
			"account":    account,
			"login_time": loginTime,
			"ip":         ip,
			"device":     device,
		}),
	}

	_, err := facade.DB.Model(&Notification{}).Create(notif)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": uid}, "创建账号登录通知失败")
		return nil, err
	}

	return notif, nil
}

// parseDevice 从 User-Agent 中解析出简洁的设备描述
// 例如：Windows 10 Chrome 144 / Android Chrome 145 / iPhone Safari 17
func parseDevice(ua string) string {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return "未知设备"
	}

	var os, browser string

	// 操作系统
	switch {
	case strings.Contains(ua, "Windows NT 10.0"):
		os = "Windows 10"
	case strings.Contains(ua, "Windows NT 6.3"):
		os = "Windows 8.1"
	case strings.Contains(ua, "Windows NT 6.2"):
		os = "Windows 8"
	case strings.Contains(ua, "Windows NT 6.1"):
		os = "Windows 7"
	case strings.Contains(ua, "Windows"):
		os = "Windows"
	case strings.Contains(ua, "Android"):
		os = "Android"
	case strings.Contains(ua, "iPhone"), strings.Contains(ua, "iPad"), strings.Contains(ua, "iPod"):
		os = "iOS"
	case strings.Contains(ua, "Mac OS X"), strings.Contains(ua, "Macintosh"):
		os = "macOS"
	case strings.Contains(ua, "Linux"):
		os = "Linux"
	}

	// 浏览器
	switch {
	case strings.Contains(ua, "Edg/"):
		browser = "Edge"
	case strings.Contains(ua, "OPR/"), strings.Contains(ua, "Opera"):
		browser = "Opera"
	case strings.Contains(ua, "Firefox/"):
		browser = "Firefox"
	case strings.Contains(ua, "Chrome/"):
		browser = "Chrome"
	case strings.Contains(ua, "Safari/") && strings.Contains(ua, "Version/"):
		browser = "Safari"
	}

	// 浏览器主版本号
	version := ""
	for _, prefix := range []string{"Edg/", "OPR/", "Firefox/", "Chrome/"} {
		if idx := strings.Index(ua, prefix); idx >= 0 {
			rest := ua[idx+len(prefix):]
			if end := strings.IndexAny(rest, " .;)"); end > 0 {
				version = rest[:end]
			} else {
				version = rest
			}
			break
		}
	}
	if version == "" {
		if idx := strings.Index(ua, "Version/"); idx >= 0 {
			rest := ua[idx+len("Version/"):]
			if end := strings.IndexAny(rest, " .;)"); end > 0 {
				version = rest[:end]
			} else {
				version = rest
			}
		}
	}

	parts := []string{}
	if os != "" {
		parts = append(parts, os)
	}
	if browser != "" {
		parts = append(parts, browser)
	}
	if version != "" {
		parts = append(parts, version)
	}

	if len(parts) == 0 {
		return "未知设备"
	}

	return strings.Join(parts, " ")
}

// MarkBroadcastRead 标记广播通知为已读（记录该用户的已读状态）
func (this *Notification) MarkBroadcastRead(id, uid int) error {
	return facade.DB.Drive().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}, {Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]any{"is_read": 1, "update_time": time.Now().Unix()}),
	}).Create(&NotificationRead{
		NotificationId: id,
		Uid:            uid,
		IsRead:         1,
	}).Error
}

// HideBroadcast 隐藏广播通知（用户删除后不再展示）
func (this *Notification) HideBroadcast(id, uid int) error {
	return facade.DB.Drive().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}, {Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]any{"is_deleted": 1, "update_time": time.Now().Unix()}),
	}).Create(&NotificationRead{
		NotificationId: id,
		Uid:            uid,
		IsDeleted:      1,
	}).Error
}

// GetUnreadCount 获取用户未读通知数量（包含广播通知）
func (this *Notification) GetUnreadCount(uid int) int64 {
	var count int64
	// 广播通知(uid=0)：未读 = 该用户没有已读(notification_reads.is_read=1)且没有隐藏(notification_reads.is_deleted=1)的状态记录
	sql := "SELECT COUNT(*) FROM inis_notification n " +
		"LEFT JOIN inis_notification_read nr ON nr.notification_id = n.id AND nr.uid = ? " +
		"WHERE (n.uid = ? OR n.uid = 0) AND (n.delete_time IS NULL OR n.delete_time = 0) " +
		"AND (n.uid != 0 AND n.is_read = 0 " +
		"OR n.uid = 0 AND (nr.id IS NULL OR (nr.is_read = 0 AND nr.is_deleted = 0)))"
	facade.DB.Drive().Raw(sql, uid, uid).Scan(&count)
	return count
}

// MarkRead 标记单条通知为已读（支持广播通知）
func (this *Notification) MarkRead(uid, id int) error {
	info, _ := facade.DB.Model(&Notification{}).Where("id", id).Find()
	if utils.Is.Empty(info) {
		return fmt.Errorf("通知不存在")
	}

	// 广播通知：记录该用户的已读状态
	if cast.ToInt(info["uid"]) == 0 {
		return this.MarkBroadcastRead(id, uid)
	}

	// 个人通知：仅能操作自己的通知
	if cast.ToInt(info["uid"]) != uid {
		return nil
	}

	_, err := facade.DB.Model(&Notification{}).
		Where("id", id).
		Where("uid", uid).
		Update(map[string]any{"is_read": 1})
	return err
}

// MarkAllRead 标记用户所有通知为已读（包含广播通知）
func (this *Notification) MarkAllRead(uid int) error {
	// 个人通知
	if _, err := facade.DB.Model(&Notification{}).
		Where("uid", uid).
		Where("is_read", 0).
		Update(map[string]any{"is_read": 1}); err != nil {
		return err
	}

	// 广播通知：批量写入已读状态
	ids, _ := facade.DB.Model(&[]Notification{}).Where("uid", 0).Column("id")

	var records []NotificationRead
	for _, id := range utils.Unity.Ids(ids) {
		records = append(records, NotificationRead{
			NotificationId: cast.ToInt(id),
			Uid:            uid,
			IsRead:         1,
		})
	}

	if utils.Is.Empty(records) {
		return nil
	}

	return facade.DB.Drive().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "notification_id"}, {Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]any{"is_read": 1, "update_time": time.Now().Unix()}),
	}).Create(&records).Error
}

// GetNotifications 获取用户通知列表（包含广播通知）
// 广播通知(uid=0)只存一条记录，通过 notification_reads 表计算每个用户的已读/隐藏状态
func (this *Notification) GetNotifications(uid int, typ string, isRead int, page, limit int, order string) ([]map[string]any, int64) {

	where := ""
	args := []any{uid, uid}

	// 类型过滤
	if typ != "" {
		where += " AND n.type = ?"
		args = append(args, typ)
	}

	// 已读/未读过滤（广播通知的已读状态在 notification_reads 表）
	if isRead >= 0 {
		where += " AND (n.uid != 0 AND n.is_read = ? OR n.uid = 0 AND COALESCE(nr.is_read, 0) = ?)"
		args = append(args, isRead, isRead)
	}

	// 排序字段白名单，防止 SQL 注入
	if !utils.In.Array(strings.ToLower(order), []any{"create_time desc", "create_time asc", "id desc", "id asc", "is_read desc", "is_read asc"}) {
		order = "create_time desc"
	}

	common := "FROM inis_notification n " +
		"LEFT JOIN inis_notification_read nr ON nr.notification_id = n.id AND nr.uid = ? " +
		"WHERE (n.uid = ? OR n.uid = 0) AND (n.delete_time IS NULL OR n.delete_time = 0) " +
		"AND (n.uid != 0 OR nr.id IS NULL OR nr.is_deleted = 0)"

	// 统计总数
	var count int64
	facade.DB.Drive().Raw("SELECT COUNT(*) "+common+where, args...).Scan(&count)

	// 分页列表
	// 注意：facade.DB.Model().Query() 存在缺陷——Query 内部调用 this.model.Raw() 但未将返回值
	// 赋回 this.model，且后续 Select() 走的是 Find()，而 GORM 的 Find() 不会执行 Raw SQL，
	// 导致精心构造的 JOIN/WHERE SQL 完全不生效，接口始终返回空数据。
	// 这里直接使用 Drive().Raw().Scan() 执行原生 SQL，与上方 count 查询的写法保持一致。
	fields := "SELECT n.id, n.uid, n.from_uid, n.type, n.title, n.content, n.bind_id, n.bind_type, " +
		"CASE WHEN n.uid = 0 THEN COALESCE(nr.is_read, 0) ELSE n.is_read END AS is_read, " +
		"n.`json`, n.`text`, n.create_time, n.update_time, n.delete_time "

	offset := max(0, (page-1)*limit)
	listArgs := append(args, limit, offset)
	sql := fields + common + where + fmt.Sprintf(" ORDER BY n.%s LIMIT ? OFFSET ?", order)

	var data []map[string]any
	if err := facade.DB.Drive().Raw(sql, listArgs...).Scan(&data).Error; err != nil {
		facade.Log.Error(map[string]any{
			"err":  err.Error(),
			"sql":  sql,
			"args": listArgs,
			"uid":  uid,
		}, "GetNotifications-SQL执行失败")
	}

	return data, count
}
