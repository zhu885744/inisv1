package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// 勋章条件常量
//
// 条件（Condition）决定「怎么获得」：
//   - manual：只能后台手动发放（不参与自动判定）；
//   - 其余为自动条件，ConditionValue 是阈值，由 BadgeAutoGrant / 定时任务判定。
//
// 判定口径统一放在 badgeValue() 里，前后端展示的进度也来自同一个函数，
// 避免「页面显示 9/10 但已经发放」这类不一致。
const (
	BadgeConditionManual        = "manual"           // 仅手动发放
	BadgeConditionRegisterDays  = "register_days"    // 注册天数
	BadgeConditionArticle       = "article_count"    // 已发布文章数
	BadgeConditionArticleViews  = "article_views"    // 文章总阅读量
	BadgeConditionMoments       = "moments_count"    // 已发布动态数
	BadgeConditionComment       = "comment_count"    // 评论数
	BadgeConditionLikeReceived  = "like_received"    // 内容被点赞数（文章 / 动态 / 评论）
	BadgeConditionCollectRecv   = "collect_received" // 内容被收藏数（文章 / 动态 / 独立页）
	BadgeConditionFans          = "fans_count"       // 粉丝数
	BadgeConditionDocs          = "docs_count"       // 已发布文档数
	BadgeConditionCheckin       = "checkin_days"     // 累计签到天数
	BadgeConditionCheckinStreak = "checkin_streak"   // 连续签到（历史最长）
	BadgeConditionExp           = "exp"              // 经验值
	BadgeConditionIntegral      = "integral"         // 积分余额
	BadgeConditionBadgeCount    = "badge_count"      // 已获得勋章数（集邮向）
	BadgeConditionDecoration    = "decoration_count" // 拥有的装扮数
	BadgeConditionGoods         = "goods_count"      // 兑换的商品件数
	BadgeConditionTipReceived   = "tip_received"     // 收到的打赏次数
)

// BadgeConditionList - 条件清单（后台下拉用，前端 api/badge.js 的 CONDITION 文案与此对齐）
//
// name 是「不带阈值」的短语，拼接顺序统一为 `名称 阈值单位`（见 BadgeConditionText），
// 这样后台下拉、勋章说明、进度条文案三处口径一致。
// 顺序 = 后台下拉顺序，按主题分组：内容产出 → 内容认可 → 活跃 → 消费。
var BadgeConditionList = []map[string]any{
	{"key": BadgeConditionManual, "name": "手动发放", "unit": ""},
	// 内容产出
	{"key": BadgeConditionArticle, "name": "发布文章", "unit": "篇"},
	{"key": BadgeConditionArticleViews, "name": "文章阅读量", "unit": "次"},
	{"key": BadgeConditionMoments, "name": "发布动态", "unit": "条"},
	{"key": BadgeConditionComment, "name": "发表评论", "unit": "条"},
	{"key": BadgeConditionDocs, "name": "发布文档", "unit": "篇"},
	// 内容认可
	{"key": BadgeConditionLikeReceived, "name": "获赞", "unit": "次"},
	{"key": BadgeConditionCollectRecv, "name": "被收藏", "unit": "次"},
	{"key": BadgeConditionFans, "name": "粉丝数", "unit": "人"},
	// 活跃度
	{"key": BadgeConditionRegisterDays, "name": "注册满", "unit": "天"},
	{"key": BadgeConditionCheckin, "name": "累计签到", "unit": "天"},
	{"key": BadgeConditionCheckinStreak, "name": "连续签到", "unit": "天"},
	{"key": BadgeConditionExp, "name": "经验值达到", "unit": "EXP"},
	{"key": BadgeConditionIntegral, "name": "积分达到", "unit": "积分"},
	// 收集 / 消费
	{"key": BadgeConditionBadgeCount, "name": "获得勋章", "unit": "枚"},
	{"key": BadgeConditionDecoration, "name": "拥有装扮", "unit": "个"},
	{"key": BadgeConditionGoods, "name": "兑换商品", "unit": "件"},
	{"key": BadgeConditionTipReceived, "name": "收到打赏", "unit": "次"},
}

// BadgeConditionName - 条件的中文名（用于自动生成描述）
func BadgeConditionName(key string) string {
	for _, item := range BadgeConditionList {
		if cast.ToString(item["key"]) == key {
			return cast.ToString(item["name"])
		}
	}
	return key
}

// BadgeConditionUnit - 条件的单位
func BadgeConditionUnit(key string) string {
	for _, item := range BadgeConditionList {
		if cast.ToString(item["key"]) == key {
			return cast.ToString(item["unit"])
		}
	}
	return ""
}

// 稀有度（只影响展示：描边 / 光效 / 标签，不参与判定）
const (
	BadgeRarityCommon    = "common"
	BadgeRarityRare      = "rare"
	BadgeRarityEpic      = "epic"
	BadgeRarityLegendary = "legendary"
)

// BadgeRarityList - 稀有度清单（后台下拉 / 前台文案共用）
var BadgeRarityList = []map[string]any{
	{"key": BadgeRarityCommon, "name": "普通"},
	{"key": BadgeRarityRare, "name": "稀有"},
	{"key": BadgeRarityEpic, "name": "史诗"},
	{"key": BadgeRarityLegendary, "name": "传说"},
}

// BadgeRarityName - 稀有度中文名
func BadgeRarityName(key string) string {
	for _, item := range BadgeRarityList {
		if cast.ToString(item["key"]) == key {
			return cast.ToString(item["name"])
		}
	}
	return "普通"
}

// BadgeRarityKeyValid - 稀有度 key 是否合法（空值按普通处理）
func BadgeRarityKeyValid(key string) bool {
	if utils.Is.Empty(key) {
		return true
	}
	for _, item := range BadgeRarityList {
		if cast.ToString(item["key"]) == key {
			return true
		}
	}
	return false
}

// BadgeInWindow - 当前是否处于「可获取」时间窗内（0 表示该端不限）
//
// 窗口只约束「能不能再拿到」：已经发出去的记录不受影响，
// 这样限定勋章过期后仍然是用户墙上的纪念品，而不是凭空消失。
func BadgeInWindow(badge Badge, now int64) bool {
	if badge.StartTime > 0 && now < badge.StartTime {
		return false
	}
	if badge.EndTime > 0 && now > badge.EndTime {
		return false
	}
	return true
}

// BadgeExpired - 持有记录是否已过期（expireTime = 0 表示永久）
func BadgeExpired(expireTime, now int64) bool {
	return expireTime > 0 && now > expireTime
}

// Badge - 勋章定义
type Badge struct {
	Id             int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name           string `gorm:"size:64; comment:勋章名称;" json:"name"`
	Description    string `gorm:"comment:勋章说明; default:Null;" json:"description"`
	Icon           string `gorm:"size:64; comment:图标（Bootstrap Icons 类名）; default:'bi bi-award';" json:"icon"`
	Image          string `gorm:"comment:勋章图片（优先于图标）; default:Null;" json:"image"`
	Color          string `gorm:"size:32; comment:主题色; default:Null;" json:"color"`
	Condition      string `gorm:"size:32; comment:获得条件; default:'manual';" json:"condition"`
	ConditionValue int    `gorm:"type:int(32); comment:条件阈值; default:0;" json:"condition_value"`
	// CategoryId 勋章分类（见 BadgeCategory），0 = 未分类（历史数据兜底）
	CategoryId int `gorm:"type:int(32); index:idx_badge_category; comment:分类ID; default:0;" json:"category_id"`
	// 稀有度：common / rare / epic / legendary（只影响展示，不影响判定）
	Rarity string `gorm:"size:16; comment:稀有度; default:'common';" json:"rarity"`
	// 限时获取窗口（unix 秒，0 = 不限）：窗口外不再自动发放，但已经获得的不受影响
	StartTime int64 `gorm:"type:int(32); comment:可获取开始时间; default:0;" json:"start_time"`
	EndTime   int64 `gorm:"type:int(32); comment:可获取结束时间; default:0;" json:"end_time"`
	// 获得后有效天数（0 = 永久）：>0 时发放瞬间算出 UserBadge.ExpireTime
	ExpireDays int    `gorm:"type:int(32); comment:获得后有效天数（0=永久）; default:0;" json:"expire_days"`
	Sort       int    `gorm:"type:int(32); comment:排序（越大越前）; default:0;" json:"sort"`
	Status     int    `gorm:"tinyint; index; comment:状态（1 启用 0 停用）; default:1;" json:"status"`
	Remark     string `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// UserBadge - 用户持有的勋章
//
// 注意：这里**不加 (uid, badge_id) 唯一索引** —— 记录是软删除的，
// 收回后再发放会与历史行冲突（同一对 uid+badge_id 会有多条历史）。
// 幂等性由 GrantUserBadge 里「先查是否持有」保证。
type UserBadge struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int    `gorm:"type:int(32); index:idx_user_badge_uid; comment:用户ID;" json:"uid"`
	BadgeId     int    `gorm:"type:int(32); index:idx_user_badge_badge; comment:勋章ID;" json:"badge_id"`
	Source      string `gorm:"size:16; comment:来源（auto 自动 / manual 手动）; default:'manual';" json:"source"`
	OperatorUid int    `gorm:"type:int(32); comment:发放人（手动发放时记录）; default:0;" json:"operator_uid"`
	Remark      string `gorm:"comment:备注; default:Null;" json:"remark"`
	// ExpireTime 过期时间（unix 秒，0 = 永久）：发放时按勋章 ExpireDays 算好落库，
	// 之后改勋章的有效天数不会追溯已有记录（和「改了条件不追溯」同一口径）
	ExpireTime int64 `gorm:"type:int(32); comment:过期时间（0=永久）; default:0;" json:"expire_time"`
	// ExpireNotified 是否已发过「即将到期」提醒（0 未发 / 1 已发），避免每小时重复提醒
	ExpireNotified int `gorm:"type:tinyint; comment:到期提醒是否已发; default:0;" json:"expire_notified"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:获得时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitBadge - 初始化勋章表
func InitBadge() {
	if err := facade.DB.Drive().AutoMigrate(&Badge{}, &UserBadge{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Badge表迁移失败")
		return
	}

	// 索引显式补一次：AutoMigrate 在老表上不一定补齐（同 checkin 的处理）
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_user_badge_uid ON inis_user_badge(uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_user_badge_badge ON inis_user_badge(badge_id)")

	// 初始化数据：必须同步完成，InitTable 的等待/超时才覆盖得到
	initBadgeData()
}

// AfterFind - 查询Hook
func (this *Badge) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	this.Result = this.result()
	return
}

// result - 派生信息：给前端直接用的「条件文案」
func (this *Badge) result() map[string]any {

	desc := this.Description

	// 未填说明时，按条件自动生成「发布文章 1篇」这类文案
	if utils.Is.Empty(desc) && this.Condition != BadgeConditionManual {
		desc = BadgeConditionText(this.Condition, this.ConditionValue)
	}

	return map[string]any{
		"condition_name": BadgeConditionName(this.Condition),
		"condition_unit": BadgeConditionUnit(this.Condition),
		"condition_text": desc,
		"auto":           this.Condition != BadgeConditionManual && this.ConditionValue > 0,
		"rarity":         this.Rarity,
		"rarity_name":    BadgeRarityName(this.Rarity),
		// limited：有时限（领取窗口 / 持有有效期任一存在）
		"limited":     this.StartTime > 0 || this.EndTime > 0 || this.ExpireDays > 0,
		"expire_days": this.ExpireDays,
	}
}

// UserBadge AfterFind
func (this *UserBadge) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// initBadgeData - 初始化勋章数据（只在表为空时播种，不覆盖管理员改过的内容）
func initBadgeData() {

	count, _ := facade.DB.Model(&Badge{}).Count()
	if count != 0 {
		return
	}

	array := []Badge{
		{
			Name: "初出茅庐", Icon: "bi bi-pencil-square", Color: "#4f7cff",
			Description: "发布第一篇文章", Condition: BadgeConditionArticle, ConditionValue: 1, Sort: 100,
		},
		{
			Name: "笔耕不辍", Icon: "bi bi-journal-richtext", Color: "#7c5cff",
			Description: "累计发布 10 篇文章", Condition: BadgeConditionArticle, ConditionValue: 10, Sort: 90,
		},
		{
			Name: "初次发声", Icon: "bi bi-chat-square-heart", Color: "#22b8a6",
			Description: "发布第一条动态", Condition: BadgeConditionMoments, ConditionValue: 1, Sort: 80,
		},
		{
			Name: "热心评论", Icon: "bi bi-chat-left-text", Color: "#f0a020",
			Description: "累计发表 10 条评论", Condition: BadgeConditionComment, ConditionValue: 10, Sort: 70,
		},
		{
			Name: "常驻居民", Icon: "bi bi-house-heart", Color: "#3ba55c",
			Description: "注册满 30 天", Condition: BadgeConditionRegisterDays, ConditionValue: 30, Sort: 60,
		},
		{
			Name: "元老", Icon: "bi bi-award", Color: "#d4a017",
			Description: "注册满 365 天", Condition: BadgeConditionRegisterDays, ConditionValue: 365, Sort: 50,
		},
	}

	for _, item := range array {
		table := item
		facade.DB.Model(&Badge{}).Create(&table)
	}
}

/* ============================================================
 * 发放 / 收回
 * ============================================================ */

// HeldBadgeIds - 某用户当前持有的勋章 ID 集合
func HeldBadgeIds(uid int) map[int]bool {
	result := make(map[int]bool)
	if uid <= 0 {
		return result
	}
	rows, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Column("badge_id")
	for _, id := range utils.Unity.Ids(rows) {
		result[cast.ToInt(id)] = true
	}
	return result
}

// GrantUserBadge - 发放勋章（幂等：已持有则直接返回 false, nil）
//
// source 用 auto / manual；手动发放时 operatorUid 记操作人，自动发放传 0。
func GrantUserBadge(uid, badgeId, operatorUid int, source, remark string) (bool, error) {

	if uid <= 0 || badgeId <= 0 {
		return false, fmt.Errorf("用户或勋章不能为空")
	}

	// 勋章必须存在且未被删除（回收站里的不允许再发放）
	badgeRow, _ := facade.DB.Model(&Badge{}).Where("id", badgeId).Find()
	if utils.Is.Empty(badgeRow) {
		return false, fmt.Errorf("勋章不存在")
	}

	// 已持有：幂等返回
	held, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Exist()
	if held {
		return false, nil
	}

	if utils.Is.Empty(source) {
		source = "manual"
	}

	// 限定勋章：发放瞬间把过期时间算好落库，之后改勋章的有效天数不追溯
	expireTime := int64(0)
	if days := cast.ToInt(badgeRow["expire_days"]); days > 0 {
		expireTime = time.Now().Unix() + int64(days)*86400
	}

	table := UserBadge{
		Uid:         uid,
		BadgeId:     badgeId,
		Source:      source,
		OperatorUid: operatorUid,
		Remark:      remark,
		ExpireTime:  expireTime,
	}

	if _, err := facade.DB.Model(&UserBadge{}).Create(&table); err != nil {
		return false, err
	}

	// 勋章列表带缓存，发放后清掉
	delBadgeCache(uid)

	// 站内通知：自动获得的勋章没有任何「眼前发生的事」可参照，不通知用户根本不会知道
	notifyBadgeGranted(uid, badgeId, badgeRow, source, expireTime)

	return true, nil
}

// notifyBadgeGranted - 发放勋章后的站内通知
//
// type=badge、bind_type=badge、bind_id=勋章 ID：前台「消息」页据此显示勋章图标，
// 点击跳到「我的勋章」（见 Mellow/src/utils/notification.js）。
// 文案按来源区分：管理员发放 / 条件自动解锁 / 其它（活动、等级等）；
// 限定勋章会带上剩余有效期，避免用户拿到就放到过期。
func notifyBadgeGranted(uid, badgeId int, badgeRow map[string]any, source string, expireTime int64) {

	name := cast.ToString(badgeRow["name"])
	if utils.Is.Empty(name) {
		name = "新勋章"
	}

	title := "获得新勋章"
	content := fmt.Sprintf("你获得了勋章「%s」，快去「我的勋章」佩戴展示吧。", name)

	switch source {
	case "manual":
		content = fmt.Sprintf("管理员向你发放了勋章「%s」，快去「我的勋章」佩戴展示吧。", name)
	case "auto":
		if desc := cast.ToString(badgeRow["description"]); !utils.Is.Empty(desc) {
			content = fmt.Sprintf("你达成了「%s」，「%s」已自动发放，快去佩戴展示吧。", desc, name)
		} else {
			content = fmt.Sprintf("你达成了获得条件，「%s」已自动发放，快去佩戴展示吧。", name)
		}
	}

	// 上限取整：剩 1 小时也显示「1 天」，别写成 0 天
	if expireTime > 0 {
		days := int((expireTime - time.Now().Unix() + 86399) / 86400)
		if days < 1 {
			days = 1
		}
		content += fmt.Sprintf("（限定勋章，%d 天后过期）", days)
	}

	SendBadgeNotify(uid, badgeId, title, content)
}

// RevokeUserBadge - 收回勋章（软删除，保留发放历史）
func RevokeUserBadge(uid, badgeId int) (bool, error) {

	if uid <= 0 || badgeId <= 0 {
		return false, fmt.Errorf("用户或勋章不能为空")
	}

	exist, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Exist()
	if !exist {
		return false, fmt.Errorf("该用户未获得这枚勋章")
	}

	if _, err := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Delete(); err != nil {
		return false, err
	}

	// 收回的正好是佩戴中的那枚：佩戴位一并清掉，不然主页还挂着一枚已经没有的勋章
	clearWearingBadge(uid, badgeId)

	delBadgeCache(uid)

	// 收回也给个通知：主页上突然少一枚勋章，用户需要知道原因
	if name := BadgeNamesByIds([]int{badgeId})[badgeId]; !utils.Is.Empty(name) {
		SendBadgeNotify(uid, badgeId, "勋章已被收回",
			fmt.Sprintf("你的勋章「%s」已被管理员收回，如有疑问请联系站点管理员。", name))
	}

	return true, nil
}

func delBadgeCache(uid int) {
	go func() {
		defer func() { _ = recover() }()
		facade.Cache.DelTags([]any{"badge"})
		if uid > 0 {
			facade.Cache.Del(fmt.Sprintf("badge[%d]", uid))
		}
	}()
}

/* ============================================================
 * 条件判定
 * ============================================================ */

// badgeStats - 一次请求内复用的统计数据（避免每个条件都查一次库）
type badgeStats struct {
	uid      int
	user     map[string]any
	loaded   map[string]int
	register int64
}

func newBadgeStats(uid int) *badgeStats {
	return &badgeStats{uid: uid, loaded: make(map[string]int)}
}

func (this *badgeStats) userRow() map[string]any {
	if this.user != nil {
		return this.user
	}
	row, _ := facade.DB.Model(&Users{}).Where("id", this.uid).Find()
	if row == nil {
		row = map[string]any{}
	}
	this.user = row
	return row
}

// value - 取某个条件当前的实际值（进度）
func (this *badgeStats) value(condition string) int {

	if val, exist := this.loaded[condition]; exist {
		return val
	}

	var result int

	switch condition {
	case BadgeConditionRegisterDays:
		created := cast.ToInt64(this.userRow()["create_time"])
		if created > 0 {
			days := (time.Now().Unix() - created) / 86400
			if days > 0 {
				result = int(days)
			}
		}
	case BadgeConditionExp:
		result = cast.ToInt(this.userRow()["exp"])
	case BadgeConditionIntegral:
		result = cast.ToInt(this.userRow()["integral"])
	case BadgeConditionArticle:
		count, _ := facade.DB.Model(&Article{}).
			Where("uid", this.uid).Where("status", 1).Where("audit", 1).Count()
		result = int(count)
	case BadgeConditionMoments:
		count, _ := facade.DB.Model(&Moments{}).
			Where("uid", this.uid).Where("status", 1).Where("audit", 1).Count()
		result = int(count)
	case BadgeConditionComment:
		count, _ := facade.DB.Model(&Comment{}).Where("uid", this.uid).Count()
		result = int(count)
	case BadgeConditionCheckin:
		count, _ := facade.DB.Model(&Checkin{}).Where("uid", this.uid).Count()
		result = int(count)
	case BadgeConditionCheckinStreak:
		result = badgeCheckinStreak(this.uid)
	case BadgeConditionFans:
		count, _ := facade.DB.Model(&UserFollows{}).
			Where("follow_uid", this.uid).Where("status", 1).Count()
		result = int(count)
	case BadgeConditionLikeReceived:
		result = badgeLikesReceived(this.uid)
	case BadgeConditionCollectRecv:
		result = badgeCollectsReceived(this.uid)
	case BadgeConditionArticleViews:
		result = badgeArticleViews(this.uid)
	case BadgeConditionDocs:
		count, _ := facade.DB.Model(&Docs{}).Where("uid", this.uid).Where("status", 1).Count()
		result = int(count)
	case BadgeConditionBadgeCount:
		count, _ := facade.DB.Model(&UserBadge{}).Where("uid", this.uid).Count()
		result = int(count)
	case BadgeConditionDecoration:
		// status：0 已拥有 / 1 佩戴中 / 2 已过期，过期的不算「拥有」
		count, _ := facade.DB.Model(&UserDecoration{}).Where("uid", this.uid).Where("status <> 2").Count()
		result = int(count)
	case BadgeConditionGoods:
		// status：0 待发货 / 1 已发货 / 2 已完成 / 3 已取消，取消的不算
		count, _ := facade.DB.Model(&GoodsOrder{}).Where("uid", this.uid).Where("status <> 3").Count()
		result = int(count)
	case BadgeConditionTipReceived:
		count, _ := facade.DB.Model(&Tip{}).Where("to_uid", this.uid).Count()
		result = int(count)
	}

	this.loaded[condition] = result
	return result
}

/* ============================================================
 * 新增条件的统计辅助
 *
 * 口径统一约定：
 *   - 「内容被认可」类（获赞 / 被收藏）只统计别人对「我发布的内容」的互动；
 *   - 文章 / 动态一律只算已发布且审核通过的（与文章数、动态数条件保持一致）；
 *   - 过期装扮、已取消订单不算成果，其余都按累计值算。
 * ============================================================ */

// badgeTableName - 模型对应的真实表名
//
// 不要硬编码表名：框架可能给表加前缀或改命名策略，这里让 gorm 自己解析一次。
func badgeTableName(model any) string {
	if facade.DB == nil || model == nil {
		return ""
	}
	stmt := &gorm.Statement{DB: facade.DB.Drive()}
	if err := stmt.Parse(model); err != nil {
		return ""
	}
	return stmt.Schema.Table
}

// badgeRelationCount - 「我的内容」被点赞 / 收藏的次数
//
// user_likes / user_collects 只存 target_type + target_id，得按类型 join 到内容表，
// 再按内容作者过滤，才能得到「别人给我的内容点了多少次赞」。
func badgeRelationCount(relation any, targetType string, model any, uid int) int {
	if uid <= 0 || targetType == "" {
		return 0
	}
	relTable := badgeTableName(relation)
	targetTable := badgeTableName(model)
	if relTable == "" || targetTable == "" {
		return 0
	}
	var total int64
	facade.DB.Drive().Model(relation).
		Joins("JOIN "+targetTable+" ON "+targetTable+".id = "+relTable+".target_id").
		Where(relTable+".target_type = ?", targetType).
		Where(targetTable+".uid = ?", uid).
		// 被删除的内容不再计入（join 进来的表不会自动带软删除条件）
		Where(targetTable+".delete_time", 0).
		Count(&total)
	return int(total)
}

// badgeLikesReceived - 文章 / 动态 / 评论被点赞的总次数
// 注意 moments 的 target_type 历史上两种写法都有（moment / moments），两个都算上
func badgeLikesReceived(uid int) int {
	return badgeRelationCount(&UserLikes{}, "article", &Article{}, uid) +
		badgeRelationCount(&UserLikes{}, "moment", &Moments{}, uid) +
		badgeRelationCount(&UserLikes{}, "moments", &Moments{}, uid) +
		badgeRelationCount(&UserLikes{}, "comment", &Comment{}, uid)
}

// badgeCollectsReceived - 文章 / 动态 / 独立页被收藏的总次数
func badgeCollectsReceived(uid int) int {
	return badgeRelationCount(&UserCollects{}, "article", &Article{}, uid) +
		badgeRelationCount(&UserCollects{}, "moment", &Moments{}, uid) +
		badgeRelationCount(&UserCollects{}, "moments", &Moments{}, uid) +
		badgeRelationCount(&UserCollects{}, "page", &Pages{}, uid)
}

// badgeArticleViews - 已发布文章的累计阅读量
func badgeArticleViews(uid int) int {
	type row struct {
		Total int64 `gorm:"column:total"`
	}
	var data row
	facade.DB.Drive().Model(&Article{}).
		Select("COALESCE(SUM(views), 0) AS total").
		Where("uid", uid).Where("status", 1).Where("audit", 1).
		Scan(&data)
	return int(data.Total)
}

// badgeCheckinStreak - 历史最长连续签到天数
//
// 用 MAX(checkin.days) 而不是当前连续天数：勋章一旦获得就是永久纪念，
// 「连续签到 30 天」不该因为断签而变回未获得（进度条要往前走）。
func badgeCheckinStreak(uid int) int {
	type row struct {
		MaxDays int `gorm:"column:max_days"`
	}
	var data row
	facade.DB.Drive().Model(&Checkin{}).
		Select("COALESCE(MAX(days), 0) AS max_days").
		Where("uid", uid).
		Scan(&data)
	return data.MaxDays
}

// BadgeAutoGrant - 判定并发放该用户所有「已满足条件」的勋章
//
// 返回本次新获得的勋章（用于前端提示 / 日志）；无变化时返回空切片。
// 调用时机：发布文章 / 动态 / 评论之后（异步），以及定时任务兜底。
func BadgeAutoGrant(uid int) []Badge {

	if uid <= 0 {
		return nil
	}

	// 维护任务：任何异常都不能影响业务主流程
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "uid": uid}, "勋章自动判定发生panic")
		}
	}()

	var list []Badge
	facade.DB.Model(&list).Where("status", 1).Select()

	if len(list) == 0 {
		return nil
	}

	held := HeldBadgeIds(uid)
	stats := newBadgeStats(uid)
	granted := make([]Badge, 0, 2)
	now := time.Now().Unix()

	for _, badge := range list {
		if held[badge.Id] {
			continue
		}
		if badge.Condition == BadgeConditionManual || badge.ConditionValue <= 0 {
			continue
		}
		// 限时勋章：领取窗口外不再自动发放（手动发放不走这里，管理员仍可补）
		if !BadgeInWindow(badge, now) {
			continue
		}
		if stats.value(badge.Condition) < badge.ConditionValue {
			continue
		}
		ok, err := GrantUserBadge(uid, badge.Id, 0, "auto", "")
		if err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid, "badge": badge.Id}, "自动发放勋章失败")
			continue
		}
		if ok {
			granted = append(granted, badge)
		}
	}

	return granted
}

// BadgeProgress - 某用户的勋章进度（前台「勋章墙」用）
//
// 返回：全部启用勋章 + 是否已获得 + 当前进度值 / 目标值。
// 已获得的排在前面（同组内按 Sort desc、获得时间 desc）。
func BadgeProgress(uid int) []map[string]any {

	var list []Badge
	facade.DB.Model(&list).Where("status", 1).Select()

	if len(list) == 0 {
		return []map[string]any{}
	}

	now := time.Now().Unix()
	heldAt := make(map[int]int64)
	expireAt := make(map[int]int64)
	if uid > 0 {
		var records []UserBadge
		facade.DB.Model(&records).Where("uid", uid).Select()
		for _, item := range records {
			heldAt[item.BadgeId] = item.CreateTime
			expireAt[item.BadgeId] = item.ExpireTime
		}
	}

	stats := newBadgeStats(uid)
	result := make([]map[string]any, 0, len(list))

	for _, badge := range list {
		current := 0
		target := badge.ConditionValue
		if badge.Condition != BadgeConditionManual && uid > 0 {
			current = stats.value(badge.Condition)
		} else if badge.Condition == BadgeConditionManual {
			// 手动勋章没有进度概念，进度条按「已获得 / 未获得」展示
			target = 1
			if _, ok := heldAt[badge.Id]; ok {
				current = 1
			}
		} else if _, ok := heldAt[badge.Id]; ok {
			current = 1
		}

		// 已获得的直接按满进度展示
		at, held := heldAt[badge.Id]
		if held && target > 0 {
			current = target
		}

		expire := expireAt[badge.Id]
		expired := held && BadgeExpired(expire, now)
		// available=false 表示「现在拿不到」：领取窗口已关闭（或还没开始）
		available := BadgeInWindow(badge, now) && !expired

		result = append(result, map[string]any{
			"id":              badge.Id,
			"name":            badge.Name,
			"description":     badge.Description,
			"icon":            badge.Icon,
			"image":           badge.Image,
			"color":           badge.Color,
			"condition":       badge.Condition,
			"condition_value": badge.ConditionValue,
			"category_id":     badge.CategoryId,
			"condition_name":  BadgeConditionName(badge.Condition),
			"condition_unit":  BadgeConditionUnit(badge.Condition),
			"rarity":          badge.Rarity,
			"rarity_name":     BadgeRarityName(badge.Rarity),
			"start_time":      badge.StartTime,
			"end_time":        badge.EndTime,
			"expire_days":     badge.ExpireDays,
			"expire_time":     expire,
			"expired":         expired,
			"available":       available,
			"sort":            badge.Sort,
			"held":            held,
			"held_at":         at,
			"current":         current,
			"target":          target,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		left, right := result[i], result[j]
		lh, rh := cast.ToBool(left["held"]), cast.ToBool(right["held"])
		if lh != rh {
			return lh
		}
		if lh && rh {
			return cast.ToInt64(left["held_at"]) > cast.ToInt64(right["held_at"])
		}
		return cast.ToInt(left["sort"]) > cast.ToInt(right["sort"])
	})

	return result
}

// BadgeHeldByUser - 某用户已获得的勋章（展示用，附带获得时间）
func BadgeHeldByUser(uid int) []map[string]any {

	if uid <= 0 {
		return []map[string]any{}
	}

	var records []UserBadge
	facade.DB.Model(&records).Where("uid", uid).Order("create_time desc").Select()
	if len(records) == 0 {
		return []map[string]any{}
	}

	badgeIds := make([]int, 0, len(records))
	for _, item := range records {
		badgeIds = append(badgeIds, item.BadgeId)
	}

	var badges []Badge
	facade.DB.Model(&badges).WhereIn("id", badgeIds).Select()

	index := make(map[int]Badge, len(badges))
	for _, item := range badges {
		index[item.Id] = item
	}

	now := time.Now().Unix()
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		badge, exist := index[record.BadgeId]
		if !exist {
			continue
		}
		result = append(result, map[string]any{
			"id":              badge.Id,
			"name":            badge.Name,
			"description":     badge.Description,
			"icon":            badge.Icon,
			"image":           badge.Image,
			"color":           badge.Color,
			"condition":       badge.Condition,
			"condition_name":  BadgeConditionName(badge.Condition),
			"category_id":     badge.CategoryId,
			"rarity":          badge.Rarity,
			"rarity_name":     BadgeRarityName(badge.Rarity),
			"source":          record.Source,
			"held_at":         record.CreateTime,
			"expire_time":     record.ExpireTime,
			"expired":         BadgeExpired(record.ExpireTime, now),
			"condition_value": badge.ConditionValue,
		})
	}

	return result
}

/* ============================================================
 * 定时扫描（时间类条件兜底）
 * ============================================================ */

const badgeScanLimit = 200

// BadgeScan - 扫描并补发自动勋章
//
// 主要解决「时间类条件」：注册满 N 天这类事件没有触发点，只能靠定时检查。
// 数量类条件平时由发布动作触发，这里也一并扫一遍，防止「先有数据、后建勋章」漏发。
// 单轮每枚勋章最多处理 badgeScanLimit 个用户，避免大站一次性扫库。
func BadgeScan() {

	if facade.DB == nil {
		return
	}

	var list []Badge
	facade.DB.Model(&list).Where("status", 1).Select()

	total := 0
	now := time.Now().Unix()
	for _, badge := range list {
		if badge.Condition == BadgeConditionManual || badge.ConditionValue <= 0 {
			continue
		}
		// 限时勋章过了领取窗口就不再补发
		if !BadgeInWindow(badge, now) {
			continue
		}
		uids := badgeCandidates(badge)
		for _, uid := range uids {
			ok, err := GrantUserBadge(uid, badge.Id, 0, "auto", "")
			if err != nil {
				continue
			}
			if ok {
				total++
			}
		}
	}

	if total > 0 {
		facade.Log.Info(map[string]any{"granted": total}, "定时补发勋章完成")
	}

	// 顺带把「已过期 / 已不持有」的佩戴位清掉，不然主页会一直挂着一枚过期勋章
	BadgeWearSweep()
}

// BadgeWearSweep - 清理失效的佩戴位（过期的、勋章被收回的、勋章被删/停用的）
//
// 佩戴信息是冗余在 users.json 里的，写的时候没法预知以后的变化，
// 只能靠定时任务（每小时，随 BadgeScan 一起跑）对齐一次。
func BadgeWearSweep() {

	if facade.DB == nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "清理勋章佩戴位时发生panic")
		}
	}()

	// 没有任何「会过期」的勋章时直接跳过：这一扫是 users 的 LIKE 全表扫描，
	// 绝大多数站点用的是永久勋章，没必要每小时白扫一遍
	limited, _ := facade.DB.Model(&Badge{}).Where("expire_days", ">", 0).Count()
	if limited == 0 {
		return
	}

	// 只扫「json 里提到 wearing_badge」的用户：没有佩戴位的绝大多数用户直接跳过
	columnData, _ := facade.DB.Model(&Users{}).Where("json LIKE ?", "%"+BadgeWearingKey+"%").Column("id")
	uids := utils.Unity.Ids(columnData)

	now := time.Now().Unix()
	cleared := 0

	for _, raw := range uids {
		uid := cast.ToInt(raw)
		if uid <= 0 {
			continue
		}

		item, ok := GetUserJson(uid)[BadgeWearingKey].(map[string]any)
		if !ok {
			continue
		}
		badgeId := cast.ToInt(item["id"])
		if badgeId <= 0 {
			_ = UpdateUserJson(uid, map[string]any{BadgeWearingKey: nil})
			cleared++
			continue
		}

		record, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Find()
		expired := utils.Is.Empty(record) || BadgeExpired(cast.ToInt64(record["expire_time"]), now)

		if !expired {
			exist, _ := facade.DB.Model(&Badge{}).Where("id", badgeId).Where("status", 1).Exist()
			expired = !exist
		}

		if expired {
			_ = UpdateUserJson(uid, map[string]any{BadgeWearingKey: nil})
			cleared++
		}
	}

	if cleared > 0 {
		facade.Log.Info(map[string]any{"cleared": cleared}, "已清理失效的勋章佩戴位")
	}
}

// badgeCandidates - 找出「满足条件但还没持有」的用户（单轮上限 badgeScanLimit）
func badgeCandidates(badge Badge) []int {

	// 排除已持有的用户：子查询取 user_badge.uid，外层列名由调用方给（users 表是 id）
	heldQuery := func(query *gorm.DB, column string) *gorm.DB {
		sub := facade.DB.Drive().Model(&UserBadge{}).Select("uid").Where("badge_id", badge.Id)
		return query.Where(column+" NOT IN (?)", sub)
	}

	target := badge.ConditionValue
	uids := make([]int, 0, badgeScanLimit)

	switch badge.Condition {
	case BadgeConditionRegisterDays:
		deadline := time.Now().Unix() - int64(target)*86400
		query := facade.DB.Drive().Model(&Users{}).Where("create_time", "<=", deadline).Where("create_time", ">", 0)
		query = heldQuery(query, "id")
		_ = query.Order("id asc").Limit(badgeScanLimit).Pluck("id", &uids)

	case BadgeConditionExp:
		query := facade.DB.Drive().Model(&Users{}).Where("exp", ">=", target)
		query = heldQuery(query, "id")
		_ = query.Order("id asc").Limit(badgeScanLimit).Pluck("id", &uids)

	case BadgeConditionIntegral:
		query := facade.DB.Drive().Model(&Users{}).Where("integral", ">=", target)
		query = heldQuery(query, "id")
		_ = query.Order("id asc").Limit(badgeScanLimit).Pluck("id", &uids)

	case BadgeConditionArticle, BadgeConditionMoments, BadgeConditionComment, BadgeConditionCheckin:
		uids = badgeCountCandidates(badge.Condition, target, badge.Id)

	case BadgeConditionFans:
		// 粉丝数：user_follows 按「被关注者」聚合（status=1 才算仍然关注）
		uids = badgeAggCandidates(&UserFollows{}, "follow_uid", "COUNT(*)", target, badge.Id, "status = 1")

	case BadgeConditionArticleViews:
		// 阅读量是求和，不是计数
		uids = badgeAggCandidates(&Article{}, "uid", "SUM(views)", target, badge.Id, "status = 1", "audit = 1")

	case BadgeConditionDocs:
		uids = badgeAggCandidates(&Docs{}, "uid", "COUNT(*)", target, badge.Id, "status = 1")

	case BadgeConditionCheckinStreak:
		uids = badgeAggCandidates(&Checkin{}, "uid", "MAX(days)", target, badge.Id)

	case BadgeConditionBadgeCount:
		uids = badgeAggCandidates(&UserBadge{}, "uid", "COUNT(*)", target, badge.Id)

	case BadgeConditionDecoration:
		uids = badgeAggCandidates(&UserDecoration{}, "uid", "COUNT(*)", target, badge.Id, "status <> 2")

	case BadgeConditionGoods:
		uids = badgeAggCandidates(&GoodsOrder{}, "uid", "COUNT(*)", target, badge.Id, "status <> 3")

	case BadgeConditionTipReceived:
		uids = badgeAggCandidates(&Tip{}, "to_uid", "COUNT(*)", target, badge.Id)

	case BadgeConditionLikeReceived, BadgeConditionCollectRecv:
		uids = badgeRelationCandidates(badge.Condition, target, badge.Id)
	}

	return uids
}

// badgeUidRow - 聚合查询里统一把用户列别名成 uid
type badgeUidRow struct {
	Uid int `gorm:"column:uid"`
}

// badgeAggCandidates - 聚合类条件的候选用户：按 uidColumn 分组，聚合值 >= 阈值
//
// agg 可以是 COUNT(*) 也可以是 SUM(views) / MAX(days)：三种条件都是「同一个用户的一列数据攒够就发」。
func badgeAggCandidates(model any, uidColumn, agg string, target, badgeId int, filters ...string) []int {

	query := facade.DB.Drive().Model(model).
		Select(uidColumn+" AS uid").
		Group(uidColumn).
		Having(agg+" >= ?", target)

	for _, filter := range filters {
		query = query.Where(filter)
	}

	// 已持有的直接排除，省掉一轮无效判定
	held := facade.DB.Drive().Model(&UserBadge{}).Select("uid").Where("badge_id", badgeId)
	query = query.Where(uidColumn+" NOT IN (?)", held).Order(uidColumn + " asc").Limit(badgeScanLimit)

	var rows []badgeUidRow
	if err := query.Find(&rows).Error; err != nil {
		return nil
	}

	result := make([]int, 0, len(rows))
	for _, item := range rows {
		if item.Uid > 0 {
			result = append(result, item.Uid)
		}
	}
	return result
}

// badgeRelationCandidates - 「获赞 / 被收藏」类条件的候选用户
//
// 这两类条件的数据在关系表里，没有直接的 uid，只能按目标类型 join 到内容表，
// 再按内容作者聚合。各类型分别查一次后合并去重（同一用户可能多类内容都达标）。
func badgeRelationCandidates(condition string, target, badgeId int) []int {

	type targetSpec struct {
		relation any
		targetTy string
		model    any
	}

	var specs []targetSpec
	switch condition {
	case BadgeConditionLikeReceived:
		specs = []targetSpec{
			{&UserLikes{}, "article", &Article{}},
			{&UserLikes{}, "moment", &Moments{}},
			{&UserLikes{}, "moments", &Moments{}},
			{&UserLikes{}, "comment", &Comment{}},
		}
	case BadgeConditionCollectRecv:
		specs = []targetSpec{
			{&UserCollects{}, "article", &Article{}},
			{&UserCollects{}, "moment", &Moments{}},
			{&UserCollects{}, "moments", &Moments{}},
			{&UserCollects{}, "page", &Pages{}},
		}
	default:
		return nil
	}

	seen := make(map[int]bool)
	result := make([]int, 0, badgeScanLimit)

	for _, spec := range specs {
		relTable := badgeTableName(spec.relation)
		targetTable := badgeTableName(spec.model)
		if relTable == "" || targetTable == "" {
			continue
		}

		var rows []badgeUidRow
		err := facade.DB.Drive().Model(spec.relation).
			Select(targetTable+".uid AS uid").
			Joins("JOIN "+targetTable+" ON "+targetTable+".id = "+relTable+".target_id").
			Where(relTable+".target_type = ?", spec.targetTy).
			Group(targetTable+".uid").
			Having("COUNT(*) >= ?", target).
			Order(targetTable + ".uid asc").
			Limit(badgeScanLimit).
			Find(&rows).Error
		if err != nil {
			continue
		}

		for _, item := range rows {
			if item.Uid <= 0 || seen[item.Uid] || len(result) >= badgeScanLimit {
				continue
			}
			seen[item.Uid] = true
			result = append(result, item.Uid)
		}
	}

	return result
}

// badgeCountCandidates - 计数类条件：GROUP BY uid HAVING count >= 阈值
func badgeCountCandidates(condition string, target, badgeId int) []int {

	var model any
	switch condition {
	case BadgeConditionArticle:
		model = &Article{}
	case BadgeConditionMoments:
		model = &Moments{}
	case BadgeConditionComment:
		model = &Comment{}
	case BadgeConditionCheckin:
		model = &Checkin{}
	default:
		return nil
	}

	type row struct {
		Uid int
	}

	var rows []row
	query := facade.DB.Drive().Model(model).Select("uid").Group("uid").Having("COUNT(*) >= ?", target)

	// 文章 / 动态只统计「已发布且审核通过」的，与 badgeValue 的口径一致
	if condition == BadgeConditionArticle || condition == BadgeConditionMoments {
		query = query.Where("status", 1).Where("audit", 1)
	}

	held := facade.DB.Drive().Model(&UserBadge{}).Select("uid").Where("badge_id", badgeId)
	query = query.Where("uid NOT IN (?)", held).Order("uid asc").Limit(badgeScanLimit)

	if err := query.Find(&rows).Error; err != nil {
		return nil
	}

	result := make([]int, 0, len(rows))
	for _, item := range rows {
		result = append(result, item.Uid)
	}
	return result
}

// BadgeConditionText - 拼接条件文案（列表页/详情页可复用），如「发布文章 3篇」
func BadgeConditionText(condition string, value int) string {
	if condition == BadgeConditionManual {
		return "手动发放"
	}
	return strings.TrimSpace(fmt.Sprintf("%s %d%s", BadgeConditionName(condition), value, BadgeConditionUnit(condition)))
}

// BadgeNamesByIds - 批量取勋章名称（后台「持有记录」列表用）
func BadgeNamesByIds(ids []int) map[int]string {
	result := make(map[int]string)
	if len(ids) == 0 {
		return result
	}
	var list []Badge
	facade.DB.Model(&list).WithTrashed().WhereIn("id", ids).Select()
	for _, item := range list {
		result[item.Id] = item.Name
	}
	return result
}

/* ============================================================
 * 佩戴（users.json.wearing_badge）
 * ============================================================ */

// BadgeWearingKey - 佩戴的勋章在 users.json 里的键名
const BadgeWearingKey = "wearing_badge"

// BadgeRenderInfo - 佩戴位需要的冗余字段
//
// 为什么把整份信息冗余进 users.json：评论 / 文章 / 动态的作者信息里只带 users.json 整块，
// 如果这里只存一个勋章 id，前端每渲染一个昵称都得再查一次勋章详情（列表页就是 N+1 请求）。
// 佩戴时把渲染字段一起写进去，和「装扮把头像框参数回写 json」是同一套思路。
func BadgeRenderInfo(item map[string]any) map[string]any {
	return map[string]any{
		"id":          cast.ToInt(item["id"]),
		"name":        cast.ToString(item["name"]),
		"description": cast.ToString(item["description"]),
		"icon":        cast.ToString(item["icon"]),
		"image":       cast.ToString(item["image"]),
		"color":       cast.ToString(item["color"]),
		"rarity":      cast.ToString(item["rarity"]),
	}
}

// WearingBadge - 用户当前佩戴的勋章（未佩戴返回 nil）
//
// 读的时候兜底校验一次：勋章被删除 / 停用后，佩戴位自动视为空，
// 避免旧冗余数据在用户主页继续显示一枚已经不存在的勋章。
func WearingBadge(uid int) map[string]any {
	if uid <= 0 {
		return nil
	}

	item, ok := GetUserJson(uid)[BadgeWearingKey].(map[string]any)
	if !ok {
		return nil
	}

	id := cast.ToInt(item["id"])
	if id <= 0 {
		return nil
	}

	exist, _ := facade.DB.Model(&Badge{}).Where("id", id).Where("status", 1).Exist()
	if !exist {
		return nil
	}

	return item
}

// WearBadge - 佩戴一枚勋章（同时只戴一枚，重复调用等于换一枚）
func WearBadge(uid, badgeId int) error {

	if uid <= 0 {
		return errors.New("请先登录！")
	}
	if badgeId <= 0 {
		return errors.New("请选择勋章！")
	}

	record, _ := facade.DB.Model(&UserBadge{}).Where("uid", uid).Where("badge_id", badgeId).Find()
	if utils.Is.Empty(record) {
		return errors.New("你还没有获得这枚勋章！")
	}
	if expire := cast.ToInt64(record["expire_time"]); expire > 0 && time.Now().Unix() > expire {
		return errors.New("这枚勋章已过期，不能再佩戴")
	}

	item, _ := facade.DB.Model(&Badge{}).Where("id", badgeId).Find()
	if utils.Is.Empty(item) {
		return errors.New("勋章不存在！")
	}
	if cast.ToInt(item["status"]) != 1 {
		return errors.New("勋章已停用！")
	}

	if err := UpdateUserJson(uid, map[string]any{BadgeWearingKey: BadgeRenderInfo(item)}); err != nil {
		return err
	}

	// 勋章馆（佩戴榜）带缓存，佩戴变化后要让它失效
	delBadgeCache(uid)
	return nil
}

// UnwearBadge - 卸下佩戴的勋章
func UnwearBadge(uid int) error {
	if uid <= 0 {
		return errors.New("请先登录！")
	}

	if err := UpdateUserJson(uid, map[string]any{BadgeWearingKey: nil}); err != nil {
		return err
	}

	delBadgeCache(uid)
	return nil
}

// clearWearingBadge - 收回勋章时清掉佩戴位（佩戴的不是这枚就什么都不做）
func clearWearingBadge(uid, badgeId int) {
	item, ok := GetUserJson(uid)[BadgeWearingKey].(map[string]any)
	if !ok || cast.ToInt(item["id"]) != badgeId {
		return
	}
	_ = UpdateUserJson(uid, map[string]any{BadgeWearingKey: nil})
	delBadgeCache(uid)
}

// badgeConditionKeyValid - 条件 key 是否合法（校验器用）
func BadgeConditionKeyValid(key string) bool {
	if utils.Is.Empty(key) {
		return true
	}
	// 直接以 BadgeConditionList 为准：新增条件只改清单一处，不必再同步第二份白名单
	for _, item := range BadgeConditionList {
		if cast.ToString(item["key"]) == key {
			return true
		}
	}
	return false
}

/* ============================================================
 * 到期提醒（限定勋章）
 * ============================================================ */

// BadgeExpireWarnDays - 限定勋章到期前几天提醒
const BadgeExpireWarnDays = 3

// SendBadgeNotify - 勋章相关的站内通知（bind_type=badge，点击跳「我的勋章」）
func SendBadgeNotify(uid, badgeId int, title, content string) {
	if uid <= 0 || utils.Is.Empty(title) {
		return
	}
	CreateUserNotify(uid, 0, NotificationTypeBadge, title, content, "badge", badgeId)
}

// BadgeExpireWarn - 给「N 天内即将过期」的限定勋章发一次提醒
//
// 只提醒一次：靠 UserBadge.ExpireNotified 标记，否则每小时跑一次会变成连发。
// 重新发放会产生新记录（标记为 0），所以「续期」后还能再提醒一次，符合预期。
func BadgeExpireWarn() {

	if facade.DB == nil {
		return
	}

	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "发送勋章到期提醒时发生panic")
		}
	}()

	now := time.Now().Unix()
	deadline := now + int64(BadgeExpireWarnDays)*86400

	var rows []UserBadge
	facade.DB.Model(&rows).
		Where("expire_time", ">", now).
		Where("expire_time", "<=", deadline).
		Where("expire_notified", 0).
		Order("expire_time asc").
		Limit(500).
		Select()

	if len(rows) == 0 {
		return
	}

	ids := make([]int, 0, len(rows))
	for _, item := range rows {
		ids = append(ids, item.BadgeId)
	}
	names := BadgeNamesByIds(ids)

	sent := 0
	for _, row := range rows {
		name := names[row.BadgeId]
		if utils.Is.Empty(name) {
			name = "限定勋章"
		}

		// 向上取整：剩 1 小时也算「1 天」，别显示成 0 天
		days := int((row.ExpireTime - now + 86399) / 86400)
		title := "限定勋章即将到期"
		content := fmt.Sprintf("你的勋章「%s」将在 %d 天后过期，过期后不能再佩戴，记得及时留念。", name, days)

		SendBadgeNotify(row.Uid, row.BadgeId, title, content)

		if _, err := facade.DB.Model(&UserBadge{}).Where("id", row.Id).UpdateColumn("expire_notified", 1); err != nil {
			facade.Log.Warn(map[string]any{"error": err.Error(), "id": row.Id}, "标记勋章到期提醒失败")
			continue
		}
		sent++
	}

	if sent > 0 {
		facade.Log.Info(map[string]any{"sent": sent}, "已发送限定勋章到期提醒")
	}
}

/* ============================================================
 * 勋章馆 / 佩戴榜（公开）
 * ============================================================ */

// BadgeHeldCounts - 每枚勋章的持有人数
func BadgeHeldCounts() map[int]int {
	result := make(map[int]int)
	type row struct {
		BadgeId int
		Total   int
	}
	var rows []row
	facade.DB.Drive().Model(&UserBadge{}).
		Select("badge_id, COUNT(*) as total").
		Group("badge_id").
		Scan(&rows)
	for _, item := range rows {
		result[item.BadgeId] = item.Total
	}
	return result
}

// BadgeWearers - 正在佩戴各勋章的用户（每枚最多 6 个，用于「佩戴榜」）
//
// 佩戴位存在 users.json 里，只能按 json LIKE 扫；限制最多 1000 个用户 + 只取 id/昵称/头像/json，
// 保证这个公开接口不会变成全表重扫。数据量大的站点可以再加缓存。
func BadgeWearers() map[int][]map[string]any {

	result := make(map[int][]map[string]any)
	if facade.DB == nil {
		return result
	}

	type wearRow struct {
		Id       int
		Nickname string
		Avatar   string
		Json     string
	}

	var rows []wearRow
	facade.DB.Drive().Model(&Users{}).
		Select("id, nickname, avatar, json").
		Where("json LIKE ?", "%"+BadgeWearingKey+"%").
		Order("id desc").
		Limit(1000).
		Scan(&rows)

	const maxWearers = 6

	// 头像里存的是 {{cos}} / {{localhost}} 这类域名模板，展开动作本来在 Users.AfterFind 里；
	// 这里为了性能走的是 Scan 到临时结构体，不会触发钩子，必须手动展开，
	// 否则接口会把「{{cos}}/2026/xxx.jpeg」原样返回给前端（见 model/base.go:207 的说明）。
	replace := DomainTemp1()

	for _, row := range rows {
		jsonData, _ := utils.Json.Decode(row.Json).(map[string]any)
		if jsonData == nil {
			continue
		}
		item, ok := jsonData[BadgeWearingKey].(map[string]any)
		if !ok {
			continue
		}
		badgeId := cast.ToInt(item["id"])
		if badgeId <= 0 || len(result[badgeId]) >= maxWearers {
			continue
		}
		avatar := row.Avatar
		if !utils.Is.Empty(avatar) && !utils.Is.Empty(replace) {
			avatar = utils.Replace(avatar, replace)
		}
		result[badgeId] = append(result[badgeId], map[string]any{
			"uid":      row.Id,
			"nickname": row.Nickname,
			"avatar":   avatar,
		})
	}

	return result
}

// BadgeWall - 勋章馆 / 佩戴榜数据（公开，uid > 0 时附带「我是否已获得」）
func BadgeWall(uid int) facade.H {

	result := facade.H{
		"categories": make([]map[string]any, 0),
		"data":       make([]map[string]any, 0),
		"stats":      facade.H{},
	}

	if facade.DB == nil {
		return result
	}

	var list []Badge
	facade.DB.Model(&list).Where("status", 1).Select()

	rank := map[string]int{
		BadgeRarityCommon:    1,
		BadgeRarityRare:      2,
		BadgeRarityEpic:      3,
		BadgeRarityLegendary: 4,
	}

	// 荣誉墙的核心排序：稀有度优先，其次后台排序
	sort.SliceStable(list, func(i, j int) bool {
		left, right := rank[list[i].Rarity], rank[list[j].Rarity]
		if left != right {
			return left > right
		}
		if list[i].Sort != list[j].Sort {
			return list[i].Sort > list[j].Sort
		}
		return list[i].Id < list[j].Id
	})

	heldCounts := BadgeHeldCounts()
	wearers := BadgeWearers()
	held := HeldBadgeIds(uid)

	now := time.Now().Unix()
	catTotal := make(map[int]int)
	totalHeld := 0
	totalWearers := 0

	data := make([]map[string]any, 0, len(list))
	for _, badge := range list {
		catTotal[badge.CategoryId]++
		heldCount := heldCounts[badge.Id]
		wearing := wearers[badge.Id]
		totalHeld += heldCount
		totalWearers += len(wearing)

		data = append(data, map[string]any{
			"id":              badge.Id,
			"name":            badge.Name,
			"description":     badge.Description,
			"icon":            badge.Icon,
			"image":           badge.Image,
			"color":           badge.Color,
			"condition":       badge.Condition,
			"condition_value": badge.ConditionValue,
			"condition_name":  BadgeConditionName(badge.Condition),
			"condition_unit":  BadgeConditionUnit(badge.Condition),
			"category_id":     badge.CategoryId,
			"rarity":          badge.Rarity,
			"rarity_name":     BadgeRarityName(badge.Rarity),
			"start_time":      badge.StartTime,
			"end_time":        badge.EndTime,
			"expire_days":     badge.ExpireDays,
			"available":       BadgeInWindow(badge, now),
			"held":            held[badge.Id],
			"held_count":      heldCount,
			"wear_count":      len(wearing),
			"wearers":         wearing,
		})
	}

	categories := make([]map[string]any, 0)
	for _, cat := range BadgeCategoryEnabled() {
		if catTotal[cat.Id] == 0 {
			continue
		}
		categories = append(categories, map[string]any{
			"id":          cat.Id,
			"name":        cat.Name,
			"description": cat.Description,
			"icon":        cat.Icon,
			"color":       cat.Color,
			"cover":       cat.Cover,
			"total":       catTotal[cat.Id],
		})
	}
	if catTotal[0] > 0 {
		categories = append(categories, map[string]any{
			"id":          0,
			"name":        "未分类",
			"description": "",
			"icon":        "bi bi-collection",
			"color":       "",
			"cover":       "",
			"total":       catTotal[0],
		})
	}

	result["categories"] = categories
	result["data"] = data
	result["stats"] = facade.H{
		"badges":  len(list),
		"held":    totalHeld,
		"wearers": totalWearers,
	}

	return result
}
