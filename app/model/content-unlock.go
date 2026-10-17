package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

// 内容可见范围（文章 / 动态 / 独立页面 / 评论共用一套口径）
//
//	1 公开       所有用户（含访客）均可见
//	2 登录可见   访客只能看到「登录后查看」的提示
//	3 付费可见   登录用户消耗积分兑换后可见
//	4 仅作者可见 只有被评论内容的作者（及评论者本人、管理员）可见
//
// 文章 / 动态 / 独立页面的字段为 scope（范围）+ scope_integral（兑换所需积分），
// 评论只有 scope —— 评论不支持付费可见，见 ScopePaySupported；
// 也只有评论支持「仅作者可见」，见 ScopeAuthorSupported。
// 历史数据 / 未设置（0）一律按「公开」处理。
const (
	ScopePublic = 1
	ScopeLogin  = 2
	ScopePay    = 3
	ScopeAuthor = 4
)

// 访问状态（接口回传给前端的 scope_state）
//
//	""      可以查看完整内容
//	login   需要登录
//	pay     需要消耗积分兑换
//	author  仅作者可见（评论）：非作者只能看到占位提示
const (
	ScopeStateAllow  = ""
	ScopeStateLogin  = "login"
	ScopeStatePay    = "pay"
	ScopeStateAuthor = "author"
)

// scopeBindTypes - 支持可见范围的绑定类型
var scopeBindTypes = map[string]bool{
	"article": true,
	"moments": true,
	"page":    true,
	"comment": true,
}

// ScopeNormalize - 归一化可见范围：0 / 非法值按「公开」处理
func ScopeNormalize(scope int) int {
	switch scope {
	case ScopeLogin, ScopePay, ScopeAuthor:
		return scope
	default:
		return ScopePublic
	}
}

// ScopeBindTypeValid - 是否为支持的绑定类型
func ScopeBindTypeValid(bindType string) bool {
	return scopeBindTypes[bindType]
}

// ScopePaySupported - 该内容类型是否支持「付费可见」
//
// 评论只支持 公开 / 登录可见 / 仅作者可见：评论是讨论的一部分，按条收费体验很差，
// 且容易变成刷积分的手段，因此不对评论开放积分兑换。
func ScopePaySupported(bindType string) bool {
	return bindType != "comment"
}

// ScopeAuthorSupported - 该内容类型是否支持「仅作者可见」
//
// 只对评论开放：文章 / 动态 / 独立页面的可见范围面向「读者群体」，
// 「仅作者可见」等价于不发布，没有实际意义。
func ScopeAuthorSupported(bindType string) bool {
	return bindType == "comment"
}

// ScopeForBind - 按内容类型归一化可见范围
//
// 评论不支持付费可见：历史数据或旧前端传来的 3 一律降级为「登录可见」，
// 保证「评论永远不会要求积分」；
// 其它类型不支持「仅作者可见」，非评论类型传来的 4 一律按「公开」处理，
// 避免出现「谁都不能看」的孤儿内容。
func ScopeForBind(bindType string, scope int) int {
	scope = ScopeNormalize(scope)
	if scope == ScopePay && !ScopePaySupported(bindType) {
		return ScopeLogin
	}
	if scope == ScopeAuthor && !ScopeAuthorSupported(bindType) {
		return ScopePublic
	}
	return scope
}

// ScopeCommentContentOwner - 评论所属内容的作者 uid（「仅作者可见」的判定依据）
//
// 评论的 bind_type / bind_id 指向被评论的文章 / 动态 / 独立页面，
// 该内容的作者才是「仅作者可见」里的「作者」；内容不存在（或已被彻底删除）时返回 0。
func ScopeCommentContentOwner(bindType string, bindId int) int {
	if bindId <= 0 {
		return 0
	}

	var table any
	switch bindType {
	case "article":
		table = &Article{}
	case "moments":
		table = &Moments{}
	case "page":
		table = &Pages{}
	default:
		return 0
	}

	item, _ := facade.DB.Model(table).Where("id", bindId).Find()
	return cast.ToInt(item["uid"])
}

// ContentUnlock - 付费可见内容的解锁记录
//
// 一个用户对一条内容只记录一次（(uid, bind_type, bind_id) 唯一索引），
// 重复兑换直接命中已有记录、不会重复扣积分。
type ContentUnlock struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); uniqueIndex:uk_content_unlock,priority:1; comment:解锁用户ID; default:0;" json:"uid"`
	BindType string `gorm:"size:32; uniqueIndex:uk_content_unlock,priority:2; comment:绑定类型; default:'article';" json:"bind_type"`
	BindId   int    `gorm:"type:int(32); uniqueIndex:uk_content_unlock,priority:3; comment:绑定内容ID; default:0;" json:"bind_id"`
	Integral int    `gorm:"type:int(32); comment:消耗积分; default:0;" json:"integral"`
	// 以下为公共字段
	Json       any   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
}

// InitContentUnlock - 初始化可见范围解锁表
func InitContentUnlock() {
	err := facade.DB.Drive().AutoMigrate(&ContentUnlock{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ContentUnlock表迁移失败")
	}
}

// ScopeUnlockedBatch - 批量查询已解锁的内容 id（列表接口用，避免 N+1 查询）
func ScopeUnlockedBatch(uid int, bindType string, ids []int) map[int]bool {
	result := make(map[int]bool)

	if uid <= 0 || len(ids) == 0 || !ScopeBindTypeValid(bindType) {
		return result
	}

	rows, _ := facade.DB.Model(&[]ContentUnlock{}).
		Where("uid", uid).
		Where("bind_type", bindType).
		WhereIn("bind_id", ids).
		Column("bind_id")

	for _, id := range cast.ToIntSlice(rows) {
		result[id] = true
	}

	return result
}

// ScopeContentInfo - 读取内容的可见范围信息（作者 / 范围 / 积分 / 展示标题）
func ScopeContentInfo(bindType string, bindId int) (owner int, scope int, integral int, title string, err error) {
	if !ScopeBindTypeValid(bindType) {
		return 0, 0, 0, "", errors.New("不支持的内容类型！")
	}
	if bindId <= 0 {
		return 0, 0, 0, "", errors.New("内容ID不能为空！")
	}

	var table any
	switch bindType {
	case "article":
		table = &Article{}
	case "moments":
		table = &Moments{}
	case "page":
		table = &Pages{}
	case "comment":
		table = &Comment{}
	default:
		return 0, 0, 0, "", errors.New("不支持的内容类型！")
	}

	item, _ := facade.DB.Model(table).WithTrashed().Where("id", bindId).Find()
	if len(item) == 0 {
		return 0, 0, 0, "", errors.New("内容不存在！")
	}

	owner = cast.ToInt(item["uid"])
	// 按内容类型归一化：评论不支持付费可见（3 降级为登录可见，积分视为 0）
	scope = ScopeForBind(bindType, cast.ToInt(item["scope"]))
	if ScopePaySupported(bindType) {
		integral = cast.ToInt(item["scope_integral"])
	}
	title = scopeContentTitle(item)

	return owner, scope, integral, title, nil
}

// scopeTagPattern - 去掉 HTML 标签与表情标记（通知文案是纯文本）
var scopeTagPattern = regexp.MustCompile(`<[^>]*>|\[emoji:[^\]]+\]`)

// scopeContentTitle - 内容在通知里的展示标题
//
// 文章 / 独立页面用 title；动态没有标题，取正文前 40 个字（去掉标签与表情标记）。
func scopeContentTitle(item map[string]any) string {
	if title := strings.TrimSpace(cast.ToString(item["title"])); !utils.Is.Empty(title) {
		return title
	}

	text := strings.Join(strings.Fields(scopeTagPattern.ReplaceAllString(cast.ToString(item["content"]), " ")), " ")
	runes := []rune(text)
	if len(runes) > 40 {
		return string(runes[:40]) + "…"
	}

	return text
}

// Pay - 使用积分兑换内容的可见权限
//
// 规则：
//   - 只有「付费可见」的内容需要兑换，其它范围直接返回错误；
//   - 作者本人 / 管理员在接口层直接被放行，不会走到这里；
//   - 已解锁的内容幂等返回，不会重复扣积分；
//   - 扣除的积分转给内容作者（作者不存在时视为平台回收），双方各记一条流水；
//   - 扣款使用带余额条件的原子更新，余额不足整体回滚。
func (this *ContentUnlock) Pay(uid int, bindType string, bindId int) (result facade.H, err error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	owner, scope, integral, title, err := ScopeContentInfo(bindType, bindId)
	if err != nil {
		return nil, err
	}

	// 按内容类型归一化：评论不支持付费可见（历史数据里的 3 会降级为登录可见，
	// 从而走下面的「无需付费查看」分支，杜绝评论被兑换）
	scope = ScopeForBind(bindType, scope)
	if scope != ScopePay {
		return nil, errors.New("该内容无需付费查看！")
	}
	if owner == uid {
		return nil, errors.New("这是您自己的内容，无需兑换！")
	}
	if integral <= 0 {
		return nil, errors.New("该内容未设置兑换积分！")
	}

	// 本次是否真的新解锁（幂等命中时不重复发通知）
	unlockedNow := false

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {

		// 1. 已解锁则直接返回（幂等，不重复扣分）
		var unlocked int64
		if err := tx.Model(&ContentUnlock{}).
			Where("uid", uid).
			Where("bind_type", bindType).
			Where("bind_id", bindId).
			Count(&unlocked).Error; err != nil {
			return err
		}
		if unlocked > 0 {
			return nil
		}

		// 2. 扣款：带余额条件，避免并发下扣成负数
		update := tx.Model(&Users{}).
			Where("id", uid).
			Where("integral >= ?", integral).
			UpdateColumn("integral", gorm.Expr("integral - ?", integral))
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errors.New("积分不足！")
		}

		// 3. 解锁记录
		unlock := ContentUnlock{
			Uid:      uid,
			BindType: bindType,
			BindId:   bindId,
			Integral: integral,
		}
		if err := tx.Create(&unlock).Error; err != nil {
			return err
		}
		unlockedNow = true

		// 4. 积分流水：兑换者支出
		if err := tx.Create(&Integral{
			Uid:         uid,
			Value:       -integral,
			Type:        IntegralTypeUnlock,
			Description: fmt.Sprintf("兑换内容可见权限（%s #%d）", scopeBindTypeName(bindType), bindId),
			Json: utils.Json.Encode(map[string]any{
				"bind_type": bindType,
				"bind_id":   bindId,
			}),
		}).Error; err != nil {
			return err
		}

		// 5. 积分流水：作者收入（作者不存在时视为平台回收）
		if owner > 0 {
			if err := tx.Model(&Users{}).Where("id", owner).
				UpdateColumn("integral", gorm.Expr("integral + ?", integral)).Error; err != nil {
				return err
			}
			if err := tx.Create(&Integral{
				Uid:         owner,
				Value:       integral,
				Type:        IntegralTypeUnlock,
				Description: fmt.Sprintf("内容被兑换可见权限（%s #%d）", scopeBindTypeName(bindType), bindId),
				Json: utils.Json.Encode(map[string]any{
					"bind_type": bindType,
					"bind_id":   bindId,
					"from_uid":  uid,
				}),
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 通知作者：事务外异步发送，通知落库失败不影响兑换结果与响应速度
	if unlockedNow {
		go SendUnlockNotify(owner, uid, bindType, bindId, title, integral)
	}

	return facade.H{
		"bind_type": bindType,
		"bind_id":   bindId,
		"integral":  integral,
		"balance":   IntegralBalance(uid),
		"unlocked":  true,
		"time":      time.Now().Unix(),
	}, nil
}

// SendUnlockNotify - 「内容被兑换」通知作者
//
// 只在真正发生兑换时调用（幂等命中不重复发）。通知标题类似：
// 「小朱 兑换了你的文章《标题》内容查看权限」，点击跳到被兑换的内容。
func SendUnlockNotify(owner, fromUid int, bindType string, bindId int, contentTitle string, integral int) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "owner": owner, "bind_id": bindId}, "发送兑换通知时发生panic")
		}
	}()

	if owner <= 0 || owner == fromUid {
		return
	}

	// 兑换者昵称（拿不到就退化成一个可读的占位）
	nickname := ""
	if user, _ := facade.DB.Model(&Users{}).Find(fromUid); !utils.Is.Empty(user) {
		nickname = strings.TrimSpace(cast.ToString(user["nickname"]))
	}
	if utils.Is.Empty(nickname) {
		nickname = fmt.Sprintf("用户#%d", fromUid)
	}

	typeName := scopeBindTypeName(bindType)
	subject := typeName + "内容"
	if !utils.Is.Empty(contentTitle) {
		// 标题过长时截断，避免撑爆 notification.title(varchar 256)
		runes := []rune(contentTitle)
		if len(runes) > 60 {
			contentTitle = string(runes[:60]) + "…"
		}
		subject = fmt.Sprintf("%s《%s》", typeName, contentTitle)
	}

	notifyTitle := fmt.Sprintf("%s 兑换了你的%s内容查看权限", nickname, subject)
	content := fmt.Sprintf("对方消耗 %d 积分兑换了%s的查看权限，积分已转入你的账户。", integral, subject)

	CreateUserNotify(owner, fromUid, NotificationTypeUnlock, notifyTitle, content, bindType, bindId)
}

// scopeBindTypeName - 绑定类型的中文名（用于流水描述与提示文案）
func scopeBindTypeName(bindType string) string {
	switch bindType {
	case "article":
		return "文章"
	case "moments":
		return "动态"
	case "page":
		return "独立页面"
	case "comment":
		return "评论"
	default:
		return bindType
	}
}
