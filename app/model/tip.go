package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"strings"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// ============================== 积分打赏 ==============================
//
// 打赏 = 用户把自己的积分转给内容作者 / 站点用户，一次操作产生：
//  1. 一条 tips 记录（谁打赏谁、多少分、打赏的什么内容、留言）；
//  2. 两条积分流水：打赏者 -N（type=tip）、接收者 +N（type=tip-income）；
//  3. 一条站内通知给接收者。
//
// 与「内容兑换」（content-unlock.go）的区别：兑换是买权限、有幂等要求，打赏是一次性赠予，
// 允许对同一内容重复打赏；但同样用「带余额条件的原子更新」扣款，避免并发下扣成负数。

// 打赏绑定类型
const (
	TipBindArticle = "article" // 打赏文章作者
	TipBindMoments = "moments" // 打赏动态作者
	TipBindComment = "comment" // 打赏评论作者
	TipBindPage    = "page"    // 打赏独立页面作者
	TipBindUser    = "user"    // 打赏用户（用户主页入口），bind_id 即被 打赏用户 uid
)

// 打赏约束
const (
	TipMinValue         = 1     // 单次最少积分
	TipMaxValue         = 100000 // 单次最多积分（防误操作 / 防刷）
	TipMessageMaxLength = 100   // 留言最大字数（按 rune 计）
)

// Tip - 积分打赏记录
type Tip struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); index; comment:打赏者ID;" json:"uid"`
	ToUid    int    `gorm:"type:int(32); index; comment:接收者ID;" json:"to_uid"`
	Value    int    `gorm:"type:int(32); comment:打赏积分; default:0;" json:"value"`
	BindType string `gorm:"size:32; index:idx_tip_bind,priority:1; comment:绑定类型;" json:"bind_type"`
	BindId   int    `gorm:"type:int(32); index:idx_tip_bind,priority:2; comment:绑定内容ID;" json:"bind_id"`
	Message  string `gorm:"size:256; comment:打赏留言;" json:"message"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitTip - 初始化打赏表
func InitTip() {
	if err := facade.DB.Drive().AutoMigrate(&Tip{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Tip表迁移失败")
		return
	}

	// 查询维度：某内容的打赏列表 / 某用户收到或发出的打赏
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_tip_to_uid ON inis_tips(to_uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_tip_create_time ON inis_tips(create_time)")
}

// TipTarget - 解析打赏对象，返回接收者 uid 与展示标题
//
// 内容类打赏以「内容作者」为接收者，避免前端传一个别人的 uid 就能把钱打给别人；
// 用户类打赏（用户主页入口）bind_id 就是接收者 uid。
func TipTarget(bindType string, bindId int) (owner int, title string, err error) {
	if bindId <= 0 {
		return 0, "", errors.New("打赏对象不能为空！")
	}

	switch bindType {
	case TipBindUser:
		user, _ := facade.DB.Model(&Users{}).Find(bindId)
		if utils.Is.Empty(user) {
			return 0, "", errors.New("用户不存在！")
		}
		return bindId, strings.TrimSpace(cast.ToString(user["nickname"])), nil
	case TipBindArticle, TipBindMoments, TipBindComment, TipBindPage:
		owner, _, _, title, err = ScopeContentInfo(bindType, bindId)
		return owner, title, err
	default:
		return 0, "", errors.New("不支持的打赏类型！")
	}
}

// TipSummaryOf - 某个内容的打赏概览（次数 + 积分合计）
func TipSummaryOf(bindType string, bindId int) facade.H {
	var row struct {
		Count int
		Total int
	}

	facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS count, COALESCE(SUM(value), 0) AS total FROM inis_tips "+
			"WHERE bind_type = ? AND bind_id = ? AND (delete_time IS NULL OR delete_time = 0)",
		bindType, bindId,
	).Scan(&row)

	return facade.H{
		"bind_type": bindType,
		"bind_id":   bindId,
		"count":     row.Count,
		"integral":  row.Total,
	}
}

// TipUserStats - 用户收到的打赏统计（用户主页展示）
//
// recent 只带最近的几条，用于「最近打赏」头像墙；不返回分页列表（列表走 tip/list）。
func TipUserStats(uid int) facade.H {
	result := facade.H{
		"uid":       uid,
		"count":     0,
		"integral":  0,
		"tippers":   0, // 打赏过的人数（去重）
		"recent":    []facade.H{},
	}

	if uid <= 0 {
		return result
	}

	var row struct {
		Count   int
		Total   int
		Tippers int
	}
	facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS count, COALESCE(SUM(value), 0) AS total, COUNT(DISTINCT uid) AS tippers "+
			"FROM inis_tips WHERE to_uid = ? AND (delete_time IS NULL OR delete_time = 0)",
		uid,
	).Scan(&row)

	result["count"] = row.Count
	result["integral"] = row.Total
	result["tippers"] = row.Tippers

	items, _ := facade.DB.Model(&[]Tip{}).
		Where("to_uid", uid).
		Order("id desc").
		Limit(8).
		Select()

	recent := make([]facade.H, 0, len(items))
	for _, item := range items {
		uid := cast.ToInt(item["uid"])
		user, _ := facade.DB.Model(&Users{}).Find(uid)
		recent = append(recent, facade.H{
			"id":       cast.ToInt(item["id"]),
			"uid":      uid,
			"nickname": cast.ToString(user["nickname"]),
			"avatar":   cast.ToString(user["avatar"]),
			"value":    cast.ToInt(item["value"]),
			"message":  cast.ToString(item["message"]),
			"time":     cast.ToInt64(item["create_time"]),
		})
	}
	result["recent"] = recent

	return result
}

// Send - 发送一次打赏（扣打赏者、加接收者、落记录，全部在一个事务里）
//
// 并发安全：扣款用 `integral >= value` 的条件更新，余额不足时 RowsAffected = 0 直接回滚，
// 不会出现「并发打赏把余额扣成负数」。
func (this *Tip) Send(uid, toUid, value int, bindType string, bindId int, message string) (result facade.H, err error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	value = cast.ToInt(value)
	if value < TipMinValue {
		return nil, fmt.Errorf("打赏积分不能少于 %d 分！", TipMinValue)
	}
	if value > TipMaxValue {
		return nil, fmt.Errorf("单次打赏不能超过 %d 分！", TipMaxValue)
	}

	message = strings.TrimSpace(message)
	if len([]rune(message)) > TipMessageMaxLength {
		return nil, fmt.Errorf("打赏留言不能超过 %d 个字！", TipMessageMaxLength)
	}

	bindType = strings.ToLower(strings.TrimSpace(bindType))
	if utils.Is.Empty(bindType) {
		bindType = TipBindUser
	}

	owner, title, err := TipTarget(bindType, bindId)
	if err != nil {
		return nil, err
	}

	// 前端没传 to_uid 时按「打赏对象」自动补全；传了就必须与对象作者一致
	if toUid <= 0 {
		toUid = owner
	}
	if toUid != owner {
		return nil, errors.New("接收者与打赏对象不一致！")
	}
	if toUid == uid {
		return nil, errors.New("不能打赏自己！")
	}

	// 接收者昵称（通知与流水描述里用）
	nickname := strings.TrimSpace(cast.ToString(mustUser(toUid)["nickname"]))
	if utils.Is.Empty(nickname) {
		nickname = fmt.Sprintf("用户#%d", toUid)
	}

	record := Tip{
		Uid:      uid,
		ToUid:    toUid,
		Value:    value,
		BindType: bindType,
		BindId:   bindId,
		Message:  message,
	}

	subject := tipSubject(bindType, title, nickname)

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 1. 扣打赏者积分（带余额条件，防并发扣成负数）
		update := tx.Model(&Users{}).
			Where("id", uid).
			Where("integral >= ?", value).
			UpdateColumn("integral", gorm.Expr("integral - ?", value))
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errors.New("积分不足！")
		}

		// 2. 打赏记录
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// 3. 打赏者支出流水
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       -value,
			Type:        IntegralTypeTip,
			Description: "打赏" + subject,
			Json: utils.Json.Encode(map[string]any{
				"bind_type": bindType,
				"bind_id":   bindId,
				"to_uid":    toUid,
				"tip_id":    record.Id,
			}),
		}).Error; err != nil {
			return err
		}

		// 4. 接收者入账
		if err := tx.Model(&Users{}).Where("id", toUid).
			UpdateColumn("integral", gorm.Expr("integral + ?", value)).Error; err != nil {
			return err
		}

		// 5. 接收者收入流水
		if err := tx.Create(&Integral{
			Uid:         toUid,
			Value:       value,
			Type:        IntegralTypeTipIncome,
			Description: "收到打赏（" + subject + "）",
			Json: utils.Json.Encode(map[string]any{
				"bind_type": bindType,
				"bind_id":   bindId,
				"from_uid":  uid,
				"tip_id":    record.Id,
			}),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 通知接收者：事务外异步，通知落库失败不影响打赏结果
	go SendTipNotify(toUid, uid, bindType, bindId, subject, value, message)

	return facade.H{
		"id":         record.Id,
		"uid":        uid,
		"to_uid":     toUid,
		"value":      value,
		"bind_type":  bindType,
		"bind_id":    bindId,
		"balance":    IntegralBalance(uid),
		"to_balance": IntegralBalance(toUid),
	}, nil
}

// mustUser - 取用户行（取不到时返回空 map，避免调用方判空）
func mustUser(uid int) map[string]any {
	user, _ := facade.DB.Model(&Users{}).Find(uid)
	return user
}

// tipSubject - 打赏对象的中文描述，如「文章《xxx》」「用户 小朱」
func tipSubject(bindType string, title string, nickname string) string {
	runes := []rune(strings.TrimSpace(title))
	if len(runes) > 30 {
		title = string(runes[:30]) + "…"
	}

	switch bindType {
	case TipBindUser:
		if utils.Is.Empty(nickname) {
			return "用户"
		}
		return "用户 " + nickname
	default:
		name := scopeBindTypeName(bindType)
		if utils.Is.Empty(title) {
			return name
		}
		return fmt.Sprintf("%s《%s》", name, title)
	}
}

// SendTipNotify - 「收到打赏」站内通知
func SendTipNotify(owner, fromUid int, bindType string, bindId int, subject string, value int, message string) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "owner": owner}, "发送打赏通知时发生panic")
		}
	}()

	if owner <= 0 || owner == fromUid || value <= 0 {
		return
	}

	nickname := strings.TrimSpace(cast.ToString(mustUser(fromUid)["nickname"]))
	if utils.Is.Empty(nickname) {
		nickname = fmt.Sprintf("用户#%d", fromUid)
	}

	// 留言是用户输入，截断后拼进通知正文（通知正文是纯文本展示）
	text := strings.TrimSpace(message)
	if runes := []rune(text); len(runes) > 50 {
		text = string(runes[:50]) + "…"
	}

	title := "收到打赏"
	content := fmt.Sprintf("%s 打赏了你 %d 积分（%s），积分已存入你的账户。", nickname, value, subject)
	if !utils.Is.Empty(text) {
		content += "留言：" + text
	}

	CreateUserNotify(owner, fromUid, NotificationTypeTip, title, content, bindType, bindId)
}
