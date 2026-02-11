package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type EXP struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int    `gorm:"type:int(32); comment:用户ID;" json:"uid"`
	Value       int    `gorm:"type:int(32); comment:经验值; default:0;" json:"value"`
	Type        string `gorm:"comment:类型; default:'default';" json:"type"`
	BindType    string `gorm:"comment:绑定类型; default:'default';" json:"bind_type"`
	BindId      int    `gorm:"type:int(32); comment:绑定ID; default:0;" json:"bind_id"`
	State       int    `gorm:"type:int(32); comment:状态; default:1;" json:"state"`
	Description string `gorm:"comment:描述; default:Null;" json:"description"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitEXP - 初始化EXP表
func InitEXP() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&EXP{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "EXP表迁移失败")
		return
	}
}

// AfterFind - 查询Hook
func (this *EXP) AfterFind(tx *gorm.DB) (err error) {

	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// Add - 增加经验值
func (this *EXP) Add(table EXP) (err error) {

	// 拦截异常
	defer func() {
		if bug := recover(); bug != nil {
			facade.Log.Error(map[string]any{"error": bug}, "增加经验值失败")

		}
	}()

	if table.Uid == 0 {
		return errors.New("请先登录！")
	}

	limit := map[string][]any{
		"like":     {"点赞", 1, 10}, // 点赞 - 每天10次，一次1经验值
		"collect":  {"收藏", 1, 10}, // 收藏 - 每天10次，一次1经验值
		"visit":    {"访问", 1, 10}, // 访问 - 每天10次，一次1经验值
		"share":    {"分享", 1, 10}, // 分享 - 每天10次，一次1经验值
		"login":    {"登录", 5, 1},  // 登录 - 每天1次，一次5经验值
		"comment":  {"评论", 1, 10}, // 评论 - 每天10次，一次1经验值
		"check-in": {"签到", 10, 1}, // 签到 - 每天1次，一次10经验值
	}

	// 检查 limit[table.Type] 是否存在
	if _, ok := limit[table.Type]; !ok {
		return errors.New("未知的经验值类型！")
	}

	// 今天开始的时间戳
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 检查是否可以增加经验值
	var canAddExp bool = true

	// 根据操作类型进行不同的处理
	switch table.Type {
	case "check-in", "login":
		// 对于签到和登录类型，需要特殊处理，因为它们是每天只能一次的操作
		// 检查是否已经操作过
		count := facade.DB.Model(&EXP{}).Where([]any{
			[]any{"uid", "=", table.Uid},
			[]any{"type", "=", table.Type},
			[]any{"create_time", ">=", today.Unix()},
		}).Count()

		if count >= cast.ToInt64(limit[table.Type][2]) {
			// 已经操作过，返回错误信息
			switch table.Type {
			case "check-in":
				return errors.New("今天已经签到过了！")
			case "login":
				return errors.New("今天获取过登录经验值！")
			default:
				return errors.New("未知的经验值类型！")
			}
		}
	case "like", "collect", "share", "comment":
		// 对于基于对象的操作，检查是否已经对该对象操作过
		exist := facade.DB.Model(&EXP{}).Where([]any{
			[]any{"uid", "=", table.Uid},
			[]any{"type", "=", table.Type},
			[]any{"bind_id", "=", table.BindId},
			[]any{"bind_type", "=", table.BindType},
			[]any{"create_time", ">=", today.Unix()},
		}).Exist()

		if exist {
			// 已经对该对象操作过，不增加经验值，但操作可以继续
			canAddExp = false
		}
	default:
		// 对于其他类型，检查总次数是否达到限制
		count := facade.DB.Model(&EXP{}).Where([]any{
			[]any{"uid", "=", table.Uid},
			[]any{"type", "=", table.Type},
			[]any{"create_time", ">=", today.Unix()},
		}).Count()

		if count >= cast.ToInt64(limit[table.Type][2]) {
			// 达到限制，不增加经验值
			canAddExp = false
		}
	}

	// 如果可以增加经验值
	if canAddExp {
		// 每次增加的经验值
		table.Value = cast.ToInt(limit[table.Type][1])
		if utils.Is.Empty(table.Description) {
			table.Description = fmt.Sprintf("%s奖励", limit[table.Type][0])
		}

		// 添加经验日志
		tx := facade.DB.Model(&EXP{}).Create(&table)

		if tx.Error != nil {
			return tx.Error
		}

		// 增加经验值
		facade.DB.Model(&Users{}).Where("id", table.Uid).Inc("exp", table.Value)
	}

	// 对于基于对象的操作和其他类型，即使不增加经验值，也返回nil，这样操作可以继续
	return nil
}

// result - 返回结果
func (this *EXP) result() (result map[string]any) {

	var author any
	wg := sync.WaitGroup{}
	wg.Add(1)

	go this.author(&wg, &author)

	wg.Wait()

	return map[string]any{
		"author": author,
	}
}

// tags - 标签
func (this *EXP) author(wg *sync.WaitGroup, result *any) {

	defer wg.Done()

	// 作者信息
	author := make(map[string]any)
	allow := []string{"id", "nickname", "avatar", "description", "result", "title"}
	user := facade.DB.Model(&Users{}).Find(this.Uid)

	if !utils.Is.Empty(user) {
		author = utils.Map.WithField(user, allow)
	}

	*result = author
}
