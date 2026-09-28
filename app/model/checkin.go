package model

import (
	"errors"
	"inis/app/facade"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// ============================== 每日签到 ==============================
//
// 签到原先寄生在经验模块里：配置挂在 SYSTEM_EXP_RULES 的 check-in 项、记录写在 inis_exp 表、
// 奖励只有「经验 + 积分」两种，连续加成 / 里程碑也只能靠配置里的两个字段凑。
//
// 现在它独立成一个模块：
//   - 配置：独立的 SYSTEM_CHECKIN_RULES（开关 / 重置时间 / 基础奖励 / 周期奖励 / 连签加成 /
//     里程碑 / 月全勤 / 随机奖励 / 补签 / 文案池，想加就加，不用改代码）；
//   - 记录：独立的 inis_checkin 表（一天一行，(uid,date) 唯一索引，天然防重复签到），
//     连签天数不再靠「逐天回溯经验流水」算，一次查询即可；
//   - 奖励：交给奖励引擎（reward.go）发放，奖励项由配置描述，支持多资产、随机区间、
//     触发概率，扩展新资产只需注册一次。

// CheckinCacheKey - 签到配置缓存键（独立于经验 / 积分规则）
const CheckinCacheKey = "SYSTEM_CHECKIN_RULES"

// 签到来源
const (
	CheckinSourceSign   = 1 // 正常签到
	CheckinSourceMakeup = 2 // 补签
)

// 奖励来源分组（用于前端分类展示与明细记录）
const (
	CheckinGroupBase      = "base"      // 基础奖励
	CheckinGroupCycle     = "cycle"     // 周期奖励（第 N 天）
	CheckinGroupStreak    = "streak"    // 连续签到加成
	CheckinGroupMilestone = "milestone" // 里程碑（连续天数）
	CheckinGroupMonthly   = "monthly"   // 月累计（全勤）
	CheckinGroupRandom    = "random"    // 随机 / 概率奖励
	CheckinGroupMakeup    = "makeup"    // 补签奖励
)

// Checkin - 签到记录（一天一行）
//
// Date 是「签到日」键（yyyymmdd，按配置的 reset_hour 折算），与 uid 组成唯一索引：
// 并发重复签到会被数据库挡住，不需要额外的分布式锁。
type Checkin struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid      int    `gorm:"type:int(32); uniqueIndex:idx_checkin_uid_date; comment:用户ID;" json:"uid"`
	Date     int    `gorm:"type:int(32); uniqueIndex:idx_checkin_uid_date; index; comment:签到日(yyyymmdd，按重置时间折算);" json:"date"`
	Days     int    `gorm:"type:int(32); comment:签到后的连续天数; default:0;" json:"days"`
	Cycle    int    `gorm:"type:int(32); comment:周期内第几天; default:0;" json:"cycle"`
	Month    int    `gorm:"type:int(32); index; comment:所属月份(yyyymm); default:0;" json:"month"`
	Source   int    `gorm:"type:int(32); comment:来源（1正常 2补签）; default:1;" json:"source"`
	Exp      int    `gorm:"type:int(32); comment:本次经验合计; default:0;" json:"exp"`
	Integral int    `gorm:"type:int(32); comment:本次积分合计; default:0;" json:"integral"`
	Tips     string `gorm:"comment:签到文案; default:Null;" json:"tips"`
	Ip       string `gorm:"size:64; comment:签到IP; default:Null;" json:"ip"`
	// 以下为公共字段
	Rewards    any                   `gorm:"type:longtext; comment:奖励明细;" json:"rewards"`
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitCheckin - 初始化签到表
func InitCheckin() {
	if err := facade.DB.Drive().AutoMigrate(&Checkin{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Checkin表迁移失败")
		return
	}
	// 补索引：AutoMigrate 在已有表上不一定补全，这里显式再建一次（IF NOT EXISTS 幂等）
	facade.DB.Drive().Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_checkin_uid_date ON inis_checkin(uid, date)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_checkin_month ON inis_checkin(month)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_checkin_create_time ON inis_checkin(create_time)")
}

// AfterFind - 查询Hook
func (this *Checkin) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	this.Rewards = utils.Json.Decode(this.Rewards)
	return
}

// ============================== 配置 ==============================

// defaultCheckinConfig - 签到默认配置
//
// 结构说明（全部可在后台「签到」页里改）：
//
//	enabled     总开关（0 关闭，其它值开启）
//	name        名称，用于文案与流水描述
//	reset_hour  每日重置时间点（0-23）：设为 4 表示凌晨 4 点前算前一天，方便熬夜用户
//	notice      签到成功后是否发站内信（0/1）
//	tips        随机文案池
//	base        基础奖励（每次签到都有）：奖励项数组
//	streak      连续签到加成：{enabled, asset, per_day(每连续一天 +N), max(封顶)}
//	cycle       周期奖励：{enabled, loop(是否循环), days:[{label, rewards:[奖励项]}]}
//	milestones  里程碑：{day, label, rewards} 的数组（连续天数刚好达到时发放）
//	monthly     月累计：{day, label, rewards} 的数组（当月签到天数刚好达到时发放）
//	random      随机奖励：奖励项数组，可带 chance（概率，单位 %）与 min/max（随机区间）
//	makeup      补签：{enabled, days(可补最近几天), limit(每月次数), asset/cost(消耗), rewards(可选)}
func defaultCheckinConfig() facade.H {
	return facade.H{
		"enabled":    1,
		"name":       "每日签到",
		"reset_hour": 0,
		"notice":     0,
		"tips": []any{
			"签到成功，今天也要元气满满～",
			"滴，签到卡！新的一天开始啦",
			"坚持的人运气不会太差，明天见",
			"签到成功，离目标又近了一步",
			"连续签到有惊喜，别忘了明天再来",
		},
		"base": []any{
			facade.H{"asset": "exp", "value": 10},
			facade.H{"asset": "integral", "value": 5},
		},
		"streak": facade.H{
			"enabled": 1,
			"asset":   "exp",
			"per_day": 2,
			"max":     50,
		},
		"cycle": facade.H{
			"enabled": 1,
			"loop":    1,
			"days": []any{
				facade.H{"label": "第 1 天"},
				facade.H{"label": "第 2 天"},
				facade.H{"label": "第 3 天", "rewards": []any{facade.H{"asset": "integral", "value": 5}}},
				facade.H{"label": "第 4 天"},
				facade.H{"label": "第 5 天"},
				facade.H{"label": "第 6 天"},
				facade.H{"label": "第 7 天", "rewards": []any{facade.H{"asset": "integral", "value": 30, "label": "周期礼包"}}},
			},
		},
		"milestones": []any{
			facade.H{"day": 7, "label": "连签一周", "rewards": []any{
				facade.H{"asset": "exp", "value": 50},
				facade.H{"asset": "integral", "value": 20},
			}},
			facade.H{"day": 15, "label": "半月坚持", "rewards": []any{
				facade.H{"asset": "exp", "value": 100},
				facade.H{"asset": "integral", "value": 50},
			}},
			facade.H{"day": 30, "label": "月度全勤", "rewards": []any{
				facade.H{"asset": "exp", "value": 200},
				facade.H{"asset": "integral", "value": 100},
			}},
		},
		"monthly": []any{
			facade.H{"day": 20, "label": "月签满 20 天", "rewards": []any{
				facade.H{"asset": "integral", "value": 50},
			}},
			facade.H{"day": 28, "label": "当月全勤", "rewards": []any{
				facade.H{"asset": "integral", "value": 200},
				// 卡密奖励示例：从卡密池取一张面额 50 的卡密发给用户；
				// 池子为空时（fallback=integral）自动改发 50 积分，不会让签到失败
				facade.H{"asset": "card", "value": 50, "fallback": "integral", "label": "全勤卡密"},
			}},
		},
		"random": []any{
			facade.H{"asset": "integral", "value": 20, "chance": 10, "label": "幸运奖励"},
		},
		"makeup": facade.H{
			"enabled": 1,
			"days":    7,
			"limit":   3,
			"asset":   "integral",
			"cost":    20,
		},
	}
}

// GetCheckinConfig - 获取签到配置（缓存优先，缺项用默认值补全）
func GetCheckinConfig() facade.H {
	defaults := defaultCheckinConfig()

	// 优先从缓存获取
	if config, ok := checkinConfigFromCache(); ok {
		return mergeCheckinDefaults(config, defaults)
	}

	// 从数据库获取
	item, _ := facade.DB.Model(&Config{}).Where("key", CheckinCacheKey).Find()
	if !utils.Is.Empty(item) {
		if jsonData, ok := item["json"].(map[string]any); ok {
			config := facade.H(jsonData)
			mergeCheckinDefaults(config, defaults)
			facade.Cache.Set(CheckinCacheKey, config)
			return config
		}
	}

	return defaults
}

// checkinConfigFromCache - 缓存里取签到配置（兼容两种存放形态）
func checkinConfigFromCache() (facade.H, bool) {
	switch data := facade.Cache.Get(CheckinCacheKey).(type) {
	case facade.H:
		return data, true
	case map[string]any:
		return facade.H(data), true
	}
	return nil, false
}

// mergeCheckinDefaults - 缺项补全（老配置里没有的键用默认值）
//
// 只做「一层 + 已知子对象」的补全：base / milestones / monthly / random / tips 这类数组以
// 配置为准（用户可能故意清空），而 streak / cycle / makeup 这些子对象逐键补全。
func mergeCheckinDefaults(config facade.H, defaults facade.H) facade.H {
	if config == nil {
		return defaults
	}

	for key, value := range defaults {
		current, exist := config[key]
		if !exist || current == nil {
			config[key] = value
			continue
		}

		// 子对象（streak / cycle / makeup）：逐键补全
		sub := asStringMap(current)
		def := asStringMap(value)
		if len(sub) == 0 || len(def) == 0 {
			continue
		}
		for k, v := range def {
			if _, ok := sub[k]; !ok {
				sub[k] = v
			}
		}
		config[key] = sub
	}

	return config
}

// CheckinEnabled - 签到功能是否开启
func CheckinEnabled(cfg facade.H) bool {
	return cast.ToInt(cfg["enabled"]) != 0
}

// CheckinName - 签到名称（流水描述用）
func CheckinName(cfg facade.H) string {
	name := cast.ToString(cfg["name"])
	if utils.Is.Empty(name) {
		return "每日签到"
	}
	return name
}

// CheckinBaseIntegral - 基础奖励里的积分合计（供积分任务页展示「签到可得多少积分」）
func CheckinBaseIntegral(cfg facade.H) int {
	total := 0
	for _, item := range ParseRewardItems(cfg["base"], CheckinGroupBase) {
		if item.Asset == "integral" && item.Value > 0 {
			total += item.Value
		}
	}
	return total
}

// ============================== 时间与「签到日」 ==============================

// CheckinDayTime - 把任意时间折算成「所属签到日」的 0 点
//
// reset_hour 支持「跨天」：例如设为 4，则 4 点之前仍算前一天，方便熬夜的用户。
func CheckinDayTime(t time.Time) time.Time {
	cfg := GetCheckinConfig()
	if hour := cast.ToInt(cfg["reset_hour"]); hour > 0 && hour < 24 && t.Hour() < hour {
		t = t.AddDate(0, 0, -1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CheckinDateKey - 时间 → 签到日键（yyyymmdd）
func CheckinDateKey(t time.Time) int {
	day := CheckinDayTime(t)
	return day.Year()*10000 + int(day.Month())*100 + day.Day()
}

// CheckinMonthKey - 时间 → 月份键（yyyymm）
func CheckinMonthKey(t time.Time) int {
	day := CheckinDayTime(t)
	return day.Year()*100 + int(day.Month())
}

// CheckinDateKeyToTime - 签到日键 → 当天 0 点时间
func CheckinDateKeyToTime(key int) time.Time {
	year := key / 10000
	month := (key % 10000) / 100
	day := key % 100
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Now().Location())
}

// CheckinDateKeyText - 签到日键 → 文本（2026-09-27）
func CheckinDateKeyText(key int) string {
	return CheckinDateKeyToTime(key).Format("2006-01-02")
}

// ============================== 记录查询 ==============================

// CheckinRecord - 取某一天的签到记录
func CheckinRecord(uid int, date int) (facade.H, bool) {
	if uid <= 0 || date <= 0 {
		return nil, false
	}
	item, _ := facade.DB.Model(&Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"date", "=", date},
	}).Find()
	return item, !utils.Is.Empty(item)
}

// CheckinTodayChecked - 今天（按重置时间折算）是否已签到
func CheckinTodayChecked(uid int) bool {
	_, exist := CheckinRecord(uid, CheckinDateKey(time.Now()))
	return exist
}

// CheckinCurrentStreak - 当前生效的连续签到天数
//
// 今天已签到：含今天；今天未签到：展示截至昨天的连续天数（未签到时为 0）。
func CheckinCurrentStreak(uid int) int {
	day := CheckinDayTime(time.Now())
	if _, exist := CheckinRecord(uid, CheckinDateKey(day)); exist {
		return CheckinStreakOf(uid, day)
	}
	return CheckinStreakOf(uid, day.AddDate(0, 0, -1))
}

// CheckinMonthDays - 某月（yyyymm）的签到天数
func CheckinMonthDays(uid int, month int) int {
	if uid <= 0 || month <= 0 {
		return 0
	}
	count, _ := facade.DB.Model(&Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"month", "=", month},
	}).Count()
	return int(count)
}

// CheckinMonthMakeupCount - 某月的补签次数
func CheckinMonthMakeupCount(uid int, month int) int {
	if uid <= 0 || month <= 0 {
		return 0
	}
	count, _ := facade.DB.Model(&Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"month", "=", month},
		[]any{"source", "=", CheckinSourceMakeup},
	}).Count()
	return int(count)
}

// CheckinMonthRecords - 某月的全部签到记录（date → 记录）
func CheckinMonthRecords(uid int, month int) map[int]facade.H {
	result := make(map[int]facade.H)
	if uid <= 0 || month <= 0 {
		return result
	}
	rows, _ := facade.DB.Model(&[]Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"month", "=", month},
	}).Order("date asc").Select()

	for _, row := range rows {
		result[cast.ToInt(row["date"])] = row
	}
	return result
}

// CheckinStreakOf - 从 end（含）往前回溯的连续签到天数
//
// 一次查询取回近一年多的签到日，再逐日回溯，避免「每天一次数据库查询」的老写法。
func CheckinStreakOf(uid int, end time.Time) int {
	if uid <= 0 {
		return 0
	}

	day := CheckinDayTime(end)
	from := day.AddDate(0, 0, -365)

	rows, _ := facade.DB.Model(&Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"date", ">=", CheckinDateKey(from)},
	}).Column("date")

	signed := make(map[int]bool)
	for _, value := range cast.ToIntSlice(rows) {
		signed[value] = true
	}

	streak := 0
	for i := 0; i < 366; i++ {
		if !signed[CheckinDateKey(day)] {
			break
		}
		streak++
		day = day.AddDate(0, 0, -1)
	}

	return streak
}

// ============================== 奖励计算 ==============================

// CheckinRewardContext - 奖励计算的上下文
//
// 里程碑与月累计都按「跨越式」判定（before < 档位 <= after），而不是「刚好等于」：
// 正常签到两者等价；补签把断掉的连签补起来时，可能一次跨越多个档位，用等于判断就会漏发。
// 传入的 Prev* 为 0 时按「本次只增加一天」处理（after - 1）。
type CheckinRewardContext struct {
	Streak        int  // 本次签到后的连续天数
	PrevStreak    int  // 本次之前的连续天数
	CycleDay      int  // 处于周期的第几天
	MonthDays     int  // 本次签到后的当月累计天数
	PrevMonthDays int  // 本次之前的当月累计天数
	Makeup        bool // 是否补签（补签默认只发基础奖励，里程碑 / 月累计另算）
}

// CheckinRewards - 计算一次签到能拿到的奖励项（不结算随机性，因此可用于「签到前预览」）
//
// 返回：items 扁平奖励项（含 chance / min / max，交给 RollRewardItems 结算）；
// detail 按来源分组的明细（base / cycle / streak / milestone / monthly / random）。
func CheckinRewards(cfg facade.H, ctx CheckinRewardContext) (items []RewardItem, detail facade.H) {
	detail = facade.H{}
	items = make([]RewardItem, 0)

	// 补签：这里只算「补签奖励」（默认基础奖励），不叠加周期 / 连签加成；
	// 补签把断掉的连签补起来时可能跨越的里程碑 / 月档位，由 CheckinMakeup 按跨越口径单独补发
	if ctx.Makeup {
		rewards := ParseRewardItems(asStringMap(cfg["makeup"])["rewards"], CheckinGroupMakeup)
		if len(rewards) == 0 {
			rewards = ParseRewardItems(cfg["base"], CheckinGroupMakeup)
		}
		detail[CheckinGroupMakeup] = rewardItemMaps(rewards)
		return append(items, rewards...), detail
	}

	// 1. 基础奖励
	base := ParseRewardItems(cfg["base"], CheckinGroupBase)
	if len(base) > 0 {
		detail[CheckinGroupBase] = rewardItemMaps(base)
		items = append(items, base...)
	}

	// 2. 周期奖励（第 N 天）
	if _, cycleItems, _ := CheckinCycle(cfg, ctx.CycleDay); len(cycleItems) > 0 {
		detail[CheckinGroupCycle] = rewardItemMaps(cycleItems)
		items = append(items, cycleItems...)
	}

	// 3. 连续签到加成
	if streak := CheckinStreakBonus(cfg, ctx.Streak); streak.Asset != "" && streak.Value > 0 {
		detail[CheckinGroupStreak] = rewardItemMaps([]RewardItem{streak})
		items = append(items, streak)
	}

	// 4. 里程碑（连续天数跨越档位）
	if milestone := CheckinMilestoneItems(cfg, ctx.PrevStreak, ctx.Streak); len(milestone) > 0 {
		detail[CheckinGroupMilestone] = rewardItemMaps(milestone)
		items = append(items, milestone...)
	}

	// 5. 月累计（当月天数跨越档位）
	if monthly := CheckinMonthlyItems(cfg, ctx.PrevMonthDays, ctx.MonthDays); len(monthly) > 0 {
		detail[CheckinGroupMonthly] = rewardItemMaps(monthly)
		items = append(items, monthly...)
	}

	// 6. 随机奖励
	random := ParseRewardItems(cfg["random"], CheckinGroupRandom)
	if len(random) > 0 {
		detail[CheckinGroupRandom] = rewardItemMaps(random)
		items = append(items, random...)
	}

	return
}

// rewardItemMaps - 奖励项列表 → 展示结构列表
func rewardItemMaps(items []RewardItem) []facade.H {
	result := make([]facade.H, 0, len(items))
	for _, item := range items {
		result = append(result, RewardItemToMap(item))
	}
	return result
}

// rewardMaps - 把 JSON 解出来的奖励明细（[]any）统一成 []facade.H
func rewardMaps(list []any) []facade.H {
	result := make([]facade.H, 0, len(list))
	for _, raw := range list {
		if item := asStringMap(raw); len(item) > 0 {
			result = append(result, item)
		}
	}
	return result
}

// CheckinCycle - 计算某个连续天数对应的周期进度
//
// 返回：day 周期内第几天（1 起，未开启周期时为 0）、items 当天额外奖励、label 展示标签。
func CheckinCycle(cfg facade.H, streak int) (day int, items []RewardItem, label string) {
	cycle := asStringMap(cfg["cycle"])
	if cast.ToInt(cycle["enabled"]) == 0 {
		return 0, nil, ""
	}

	days := cast.ToSlice(cycle["days"])
	if len(days) == 0 || streak <= 0 {
		return 0, nil, ""
	}

	if cast.ToInt(cycle["loop"]) == 1 {
		day = (streak-1)%len(days) + 1
	} else {
		day = streak
		if day > len(days) {
			day = len(days)
		}
	}

	entry := asStringMap(days[day-1])
	label = cast.ToString(entry["label"])
	items = ParseRewardItems(entry["rewards"], CheckinGroupCycle)
	return
}

// CheckinCycleTable - 周期奖励全表（前端展示「第 1~N 天」的格子）
func CheckinCycleTable(cfg facade.H) []facade.H {
	cycle := asStringMap(cfg["cycle"])
	if cast.ToInt(cycle["enabled"]) == 0 {
		return []facade.H{}
	}

	days := cast.ToSlice(cycle["days"])
	result := make([]facade.H, 0, len(days))
	for index, raw := range days {
		entry := asStringMap(raw)
		result = append(result, facade.H{
			"day":     index + 1,
			"label":   cast.ToString(entry["label"]),
			"rewards": rewardItemMaps(ParseRewardItems(entry["rewards"], CheckinGroupCycle)),
		})
	}
	return result
}

// CheckinStreakBonus - 连续签到加成
func CheckinStreakBonus(cfg facade.H, streak int) RewardItem {
	bonus := asStringMap(cfg["streak"])
	if cast.ToInt(bonus["enabled"]) == 0 || streak <= 0 {
		return RewardItem{}
	}

	asset := cast.ToString(bonus["asset"])
	if utils.Is.Empty(asset) {
		asset = "exp"
	}

	value := streak * cast.ToInt(bonus["per_day"])
	if max := cast.ToInt(bonus["max"]); max > 0 && value > max {
		value = max
	}
	if value <= 0 {
		return RewardItem{}
	}

	return RewardItem{
		Asset: asset,
		Value: value,
		Label: "连续签到加成",
		Group: CheckinGroupStreak,
		Text:  "已连续 " + cast.ToString(streak) + " 天",
	}
}

// checkinRewardEntries - 解析「{day,label,rewards}」这类阶梯配置（里程碑 / 月累计共用）
func checkinRewardEntries(raw any) []facade.H {
	result := make([]facade.H, 0)

	for _, item := range cast.ToSlice(raw) {
		entry := asStringMap(item)
		day := cast.ToInt(entry["day"])
		if day <= 0 {
			continue
		}
		result = append(result, facade.H{
			"day":     day,
			"label":   cast.ToString(entry["label"]),
			"rewards": entry["rewards"],
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		return cast.ToInt(result[i]["day"]) < cast.ToInt(result[j]["day"])
	})

	return result
}

// CheckinMilestones - 里程碑列表（升序，带奖励项与达成状态）
func CheckinMilestones(cfg facade.H, streak int) []facade.H {
	result := make([]facade.H, 0)

	for _, entry := range checkinRewardEntries(cfg["milestones"]) {
		day := cast.ToInt(entry["day"])
		reached := streak >= day
		result = append(result, facade.H{
			"day":       day,
			"label":     entry["label"],
			"rewards":   rewardItemMaps(ParseRewardItems(entry["rewards"], CheckinGroupMilestone)),
			"reached":   reached,
			"days_left": utils.Ternary(reached, 0, day-streak),
		})
	}

	return result
}

// CheckinMilestoneItems - 本次「跨越」到的里程碑奖励项
//
// 判定口径：before < 档位天数 <= after。
//   - 正常签到：before = 截至昨天的连签、after = before + 1，等价于「刚好达到」；
//   - 补签补起断掉的连签：可能一次跨越多个档位，这里会一并发放，不会漏。
func CheckinMilestoneItems(cfg facade.H, before int, after int) []RewardItem {
	if after <= 0 {
		return nil
	}
	if before < 0 {
		before = 0
	}

	result := make([]RewardItem, 0)
	for _, entry := range checkinRewardEntries(cfg["milestones"]) {
		day := cast.ToInt(entry["day"])
		if day <= before || day > after {
			continue
		}
		result = append(result, ParseRewardItems(entry["rewards"], CheckinGroupMilestone)...)
	}

	return result
}

// CheckinCrossedMilestones - 本次跨越了哪些里程碑（返回档位列表，用于文案说明）
func CheckinCrossedMilestones(cfg facade.H, before int, after int) []int {
	result := make([]int, 0)

	for _, entry := range checkinRewardEntries(cfg["milestones"]) {
		day := cast.ToInt(entry["day"])
		if day > before && day <= after {
			result = append(result, day)
		}
	}

	return result
}

// CheckinNextMilestone - 下一个尚未达成的里程碑
func CheckinNextMilestone(cfg facade.H, streak int) facade.H {
	for _, item := range CheckinMilestones(cfg, streak) {
		if !cast.ToBool(item["reached"]) {
			return item
		}
	}
	return nil
}

// CheckinMonthly - 月累计奖励列表（升序，带达成状态）
func CheckinMonthly(cfg facade.H, monthDays int) []facade.H {
	result := make([]facade.H, 0)

	for _, entry := range checkinRewardEntries(cfg["monthly"]) {
		day := cast.ToInt(entry["day"])
		reached := monthDays >= day
		result = append(result, facade.H{
			"day":       day,
			"label":     entry["label"],
			"rewards":   rewardItemMaps(ParseRewardItems(entry["rewards"], CheckinGroupMonthly)),
			"reached":   reached,
			"days_left": utils.Ternary(reached, 0, day-monthDays),
		})
	}

	return result
}

// CheckinMonthlyItems - 本次「跨越」到的月累计档位奖励项
//
// 判定口径同里程碑：before < 档位天数 <= after（补签使当月天数增加时同样适用）。
func CheckinMonthlyItems(cfg facade.H, before int, after int) []RewardItem {
	if after <= 0 {
		return nil
	}
	if before < 0 {
		before = 0
	}

	result := make([]RewardItem, 0)
	for _, entry := range checkinRewardEntries(cfg["monthly"]) {
		day := cast.ToInt(entry["day"])
		if day <= before || day > after {
			continue
		}
		result = append(result, ParseRewardItems(entry["rewards"], CheckinGroupMonthly)...)
	}

	return result
}

// CheckinCrossedMonthly - 本次跨越了哪些月档位（返回档位列表，用于文案说明）
func CheckinCrossedMonthly(cfg facade.H, before int, after int) []int {
	result := make([]int, 0)

	for _, entry := range checkinRewardEntries(cfg["monthly"]) {
		day := cast.ToInt(entry["day"])
		if day > before && day <= after {
			result = append(result, day)
		}
	}

	return result
}

// PickCheckinTips - 随机取一条签到文案
func PickCheckinTips(cfg facade.H) string {
	tips := cast.ToStringSlice(cfg["tips"])
	if len(tips) == 0 {
		return "签到成功！"
	}
	return tips[utils.Rand.Int(len(tips), 0)]
}

// ============================== 状态 / 日历 / 排行 / 规则 ==============================

// CheckinStatus - 签到页所需的全部状态（已签到则为实际明细，未签到则为预期奖励）
func CheckinStatus(uid int) facade.H {
	cfg := GetCheckinConfig()
	now := time.Now()
	today := CheckinDayTime(now)
	todayKey := CheckinDateKey(today)
	monthKey := CheckinMonthKey(today)

	record, checked := CheckinRecord(uid, todayKey)

	// 连续天数：已签到含今天，未签到则展示截至昨天的连续天数
	streak := 0
	if checked {
		streak = CheckinStreakOf(uid, today)
	} else {
		streak = CheckinStreakOf(uid, today.AddDate(0, 0, -1))
	}

	monthDays := CheckinMonthDays(uid, monthKey)

	// 未签到时，用「签到后」的天数做预期展示
	previewStreak, previewMonth := streak, monthDays
	if !checked {
		previewStreak++
		previewMonth++
	}

	cycleDay, _, cycleLabel := CheckinCycle(cfg, previewStreak)
	items, detail := CheckinRewards(cfg, CheckinRewardContext{
		Streak:        previewStreak,
		PrevStreak:    streak,
		CycleDay:      cycleDay,
		MonthDays:     previewMonth,
		PrevMonthDays: monthDays,
	})

	// 「签到可得」只统计必得的固定奖励；概率奖励（chance）与随机区间（min/max）单独返回，
	// 否则用户会误以为必得（例如把 10% 概率的 20 积分算进合计里）。
	sureItems, variableItems := SplitRewardItems(items)
	total, flat := GroupRewardItems(sureItems)
	chanceTotal, chanceFlat := GroupRewardItems(variableItems)

	// 已签到：明细改用记录里的「实际发放结果」（含卡密明文等发放信息），
	// 未签到：上面的计算结果就是「签到可得」的预期
	if checked {
		if rewards := checkinRecordRewards(record); len(rewards) > 0 {
			if list := rewardMaps(cast.ToSlice(rewards["items"])); len(list) > 0 {
				flat = list
			}
			if group := asStringMap(rewards["detail"]); len(group) > 0 {
				detail = group
			}
			if summary := asStringMap(rewards["total"]); len(summary) > 0 {
				total = summary
			}
		}
		// 已发放的都是实际到手的奖励，不存在「概率」一说
		chanceTotal = facade.H{}
		chanceFlat = []facade.H{}
	}

	result := facade.H{
		"enabled":      CheckinEnabled(cfg),
		"name":         CheckinName(cfg),
		"checked":      checked,
		"today":        todayKey,
		"today_text":   CheckinDateKeyText(todayKey),
		"server_time":  now.Unix(),
		"reset_hour":   cast.ToInt(cfg["reset_hour"]),
		"streak":       streak,
		"days":         streak,
		"preview_days": previewStreak,
		"cycle_day":    cycleDay,
		"cycle_label":  cycleLabel,
		"cycle_days":   CheckinCycleTable(cfg),
		"month":        monthKey,
		"month_days":   monthDays,
		"rewards":      detail,
		"items":        flat,
		"total":        total,
		// 概率 / 区间奖励：前端用「另有 10% 概率获得 20 积分」这类文案展示，不要混进 total
		"chance_items": chanceFlat,
		"chance_total": chanceTotal,
		"milestones":   CheckinMilestones(cfg, streak),
		"monthly":      CheckinMonthly(cfg, monthDays),
		"assets":       RewardAssetOptions(),
		"makeup":       CheckinMakeupInfo(uid, cfg),
	}

	result["next_milestone"] = CheckinNextMilestone(cfg, streak)
	result["next_monthly"] = CheckinNextMonthly(cfg, monthDays)

	// 今日获得的卡密（发放时才写入明文，未签到或没配卡密时为空数组）
	result["cards"] = []facade.H{}

	if checked {
		result["check_in_time"] = cast.ToInt64(record["create_time"])
		result["record"] = map[string]any{
			"date":     record["date"],
			"days":     record["days"],
			"cycle":    record["cycle"],
			"source":   record["source"],
			"exp":      record["exp"],
			"integral": record["integral"],
			"tips":     record["tips"],
			"rewards":  checkinRecordRewards(record),
		}
		result["cards"] = checkinRecordCards(record)
	} else {
		result["check_in_time"] = 0
		result["tips"] = PickCheckinTips(cfg)
	}

	return result
}

// checkinRecordRewards - 取签到记录里的奖励明细
//
// 记录是 JSON 字符串存的，Select()/Find() 取出来可能是字符串也可能已被解码，两种都兼容。
func checkinRecordRewards(record facade.H) facade.H {
	if rewards := asStringMap(record["rewards"]); len(rewards) > 0 {
		return rewards
	}
	if raw := cast.ToString(record["rewards"]); !utils.Is.Empty(raw) {
		return asStringMap(utils.Json.Decode(raw))
	}
	return facade.H{}
}

// checkinRecordCards - 从签到记录里取出当天发放的卡密（明细里的 extra）
func checkinRecordCards(record facade.H) []facade.H {
	items := make([]facade.H, 0)

	for _, raw := range cast.ToSlice(checkinRecordRewards(record)["items"]) {
		if item := asStringMap(raw); len(item) > 0 {
			items = append(items, item)
		}
	}

	return checkinCardsOf(items)
}

// checkinCardsOf - 从发放明细里挑出卡密奖励（卡密明文藏在 extra 里）
//
// 同一个方法既服务于「刚发放完的响应」，也服务于「读取历史记录的展示」。
func checkinCardsOf(items []facade.H) []facade.H {
	result := make([]facade.H, 0)

	for _, item := range items {
		if cast.ToString(item["asset"]) != RewardCardAssetKey {
			continue
		}

		extra := asStringMap(item["extra"])
		result = append(result, facade.H{
			"card":           cast.ToString(extra["card"]),
			"value":          cast.ToInt(extra["value"]),
			"card_missing":   cast.ToBool(extra["card_missing"]),
			"fallback":       cast.ToString(extra["fallback"]),
			"fallback_value": cast.ToInt(extra["fallback_value"]),
			"granted_at":     cast.ToInt64(extra["granted_at"]),
		})
	}

	return result
}

// CheckinNextMonthly - 下一个尚未达成的月累计档位
func CheckinNextMonthly(cfg facade.H, monthDays int) facade.H {
	for _, item := range CheckinMonthly(cfg, monthDays) {
		if !cast.ToBool(item["reached"]) {
			return item
		}
	}
	return nil
}

// CheckinMakeupInfo - 补签信息（开关 / 消耗 / 剩余次数 / 可补签日期）
func CheckinMakeupInfo(uid int, cfg facade.H) facade.H {
	mk := asStringMap(cfg["makeup"])
	enabled := cast.ToInt(mk["enabled"]) != 0
	days := cast.ToInt(mk["days"])
	limit := cast.ToInt(mk["limit"])
	asset := cast.ToString(mk["asset"])
	if utils.Is.Empty(asset) {
		asset = "integral"
	}
	cost := cast.ToInt(mk["cost"])

	monthKey := CheckinMonthKey(time.Now())
	used := CheckinMonthMakeupCount(uid, monthKey)

	result := facade.H{
		"enabled": enabled,
		"days":    days,
		"limit":   limit,
		"used":    used,
		"remain":  utils.Ternary(limit > 0, limit-used, 0),
		"asset":   asset,
		"cost":    cost,
		"dates":   []facade.H{},
	}
	if !enabled || uid <= 0 || days <= 0 {
		return result
	}

	// 可补签日期：最近 N 天内、不是今天、且当天没有记录（一次查询取回区间记录）
	today := CheckinDayTime(time.Now())
	from := today.AddDate(0, 0, -days)
	rows, _ := facade.DB.Model(&Checkin{}).Where([]any{
		[]any{"uid", "=", uid},
		[]any{"date", ">=", CheckinDateKey(from)},
	}).Column("date")

	signed := make(map[int]bool)
	for _, value := range cast.ToIntSlice(rows) {
		signed[value] = true
	}

	dates := make([]facade.H, 0, days)
	for i := days; i >= 1; i-- {
		day := today.AddDate(0, 0, -i)
		key := CheckinDateKey(day)
		if signed[key] {
			continue
		}
		dates = append(dates, facade.H{
			"date": key,
			"text": CheckinDateKeyText(key),
			"time": day.Unix(),
			"cost": cost,
		})
	}
	result["dates"] = dates

	return result
}

// CheckinCalendar - 签到日历（某个月的每日状态 + 连签 + 月累计进度）
func CheckinCalendar(uid int, year int, month int) facade.H {
	cfg := GetCheckinConfig()
	now := time.Now()
	if year <= 0 {
		year = now.Year()
	}
	if month <= 0 {
		month = int(now.Month())
	}

	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	monthKey := year*100 + month

	records := CheckinMonthRecords(uid, monthKey)

	daysInMonth := end.Day()
	days := make([]facade.H, 0, daysInMonth)
	for d := 1; d <= daysInMonth; d++ {
		key := year*10000 + month*100 + d
		item := facade.H{
			"day":      d,
			"checked":  false,
			"source":   0,
			"exp":      0,
			"integral": 0,
		}
		if record, exist := records[key]; exist {
			item["checked"] = true
			item["source"] = record["source"]
			item["exp"] = record["exp"]
			item["integral"] = record["integral"]
		}
		days = append(days, item)
	}

	// 连续天数：当月看今天 / 昨天，历史月份看月末
	streak := 0
	today := 0
	if year == now.Year() && month == int(now.Month()) {
		today = CheckinDayTime(now).Day()
		if _, exist := records[year*10000+month*100+today]; exist {
			streak = CheckinStreakOf(uid, CheckinDayTime(now))
		} else {
			streak = CheckinStreakOf(uid, CheckinDayTime(now).AddDate(0, 0, -1))
		}
	} else {
		streak = CheckinStreakOf(uid, end)
	}

	monthDays := len(records)

	return facade.H{
		"year":         year,
		"month":        month,
		"days":         days,
		"streak":       streak,
		"total":        monthDays,
		"today":        today,
		"month_days":   monthDays,
		"monthly":      CheckinMonthly(cfg, monthDays),
		"next_monthly": CheckinNextMonthly(cfg, monthDays),
	}
}

// CheckinRank - 签到排行榜（按时间范围内的签到次数 / 经验 / 积分）
func CheckinRank(start int64, end int64, limit int, uid int) facade.H {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	list := make([]facade.H, 0, limit)
	myRank, myCount := 0, 0

	var rows []map[string]any
	facade.DB.Drive().Raw(
		"SELECT uid, COUNT(id) AS check_in_count, COALESCE(SUM(exp), 0) AS total_exp, "+
			"COALESCE(SUM(integral), 0) AS total_integral FROM inis_checkin "+
			"WHERE create_time >= ? AND create_time <= ? AND (delete_time IS NULL OR delete_time = 0) "+
			"GROUP BY uid ORDER BY check_in_count DESC, total_exp DESC, uid ASC LIMIT ?",
		start, end, limit,
	).Scan(&rows)

	// 批量取用户信息（一次查询，避免 N+1）
	uids := make([]any, 0, len(rows))
	for _, row := range rows {
		uids = append(uids, cast.ToInt(row["uid"]))
	}

	users := make(map[int]map[string]any)
	if len(uids) > 0 {
		items, _ := facade.DB.Model(&[]Users{}).WhereIn("id", uids).Select()
		for _, item := range items {
			users[cast.ToInt(item["id"])] = item
		}
	}

	for index, row := range rows {
		user := users[cast.ToInt(row["uid"])]
		list = append(list, facade.H{
			"id":             cast.ToInt(row["uid"]),
			"nickname":       cast.ToString(user["nickname"]),
			"avatar":         cast.ToString(user["avatar"]),
			"description":    cast.ToString(user["description"]),
			"title":          cast.ToString(user["title"]),
			"rank":           index + 1,
			"check_in_count": cast.ToInt(row["check_in_count"]),
			"total_exp":      cast.ToInt(row["total_exp"]),
			"total_integral": cast.ToInt(row["total_integral"]),
			"is_me":          cast.ToInt(row["uid"]) == uid,
		})
	}

	// 我的排名（不在前 limit 名时也能显示）
	if uid > 0 {
		var mine struct{ Count int }
		facade.DB.Drive().Raw(
			"SELECT COUNT(id) AS count FROM inis_checkin WHERE uid = ? AND create_time >= ? AND create_time <= ? "+
				"AND (delete_time IS NULL OR delete_time = 0)",
			uid, start, end,
		).Scan(&mine)
		myCount = mine.Count

		if myCount > 0 {
			var ahead int64
			facade.DB.Drive().Raw(
				"SELECT COUNT(*) FROM (SELECT uid, COUNT(id) AS count FROM inis_checkin "+
					"WHERE create_time >= ? AND create_time <= ? AND (delete_time IS NULL OR delete_time = 0) "+
					"GROUP BY uid HAVING COUNT(id) > ?) AS t",
				start, end, myCount,
			).Scan(&ahead)
			myRank = int(ahead) + 1
		}
	}

	return facade.H{
		"list":     list,
		"my_rank":  myRank,
		"my_count": myCount,
	}
}

// CheckinRules - 签到公开规则（未登录也能看：开关 / 周期 / 里程碑 / 资产清单 / 文案池）
func CheckinRules() facade.H {
	cfg := GetCheckinConfig()
	cycle := asStringMap(cfg["cycle"])
	makeup := asStringMap(cfg["makeup"])

	return facade.H{
		"enabled":    CheckinEnabled(cfg),
		"name":       CheckinName(cfg),
		"reset_hour": cast.ToInt(cfg["reset_hour"]),
		"tips":       cast.ToStringSlice(cfg["tips"]),
		"assets":     RewardAssetOptions(),
		"base":       rewardItemMaps(ParseRewardItems(cfg["base"], CheckinGroupBase)),
		"cycle":      facade.H{"enabled": cast.ToInt(cycle["enabled"]) != 0, "loop": cast.ToInt(cycle["loop"]), "days": CheckinCycleTable(cfg)},
		"streak":     asStringMap(cfg["streak"]),
		"milestones": CheckinMilestones(cfg, 0),
		"monthly":    CheckinMonthly(cfg, 0),
		"random":     rewardItemMaps(ParseRewardItems(cfg["random"], CheckinGroupRandom)),
		"makeup": facade.H{
			"enabled": cast.ToInt(makeup["enabled"]) != 0,
			"days":    cast.ToInt(makeup["days"]),
			"limit":   cast.ToInt(makeup["limit"]),
			"cost":    cast.ToInt(makeup["cost"]),
			"asset":   cast.ToString(makeup["asset"]),
		},
	}
}

// ============================== 签到 / 补签 ==============================

// CheckinDo - 执行签到
//
// 流程：算连签 → 算奖励 → 事务内「写签到记录 + 发奖励」，任何一步失败整体回滚。
// 重复签到由 (uid, date) 唯一索引兜底，并发请求只会成功一个。
func CheckinDo(uid int, ip string) (result facade.H, err error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	cfg := GetCheckinConfig()
	if !CheckinEnabled(cfg) {
		return nil, errors.New("签到功能已关闭！")
	}

	now := time.Now()
	today := CheckinDayTime(now)
	dateKey := CheckinDateKey(today)
	monthKey := CheckinMonthKey(today)

	if _, exist := CheckinRecord(uid, dateKey); exist {
		return nil, errors.New("今天已经签到过了！")
	}

	// 连签天数：昨天往前连续 + 今天（里程碑 / 月累计按「跨越」判定，此处等价于刚好达到）
	prevStreak := CheckinStreakOf(uid, today.AddDate(0, 0, -1))
	prevMonthDays := CheckinMonthDays(uid, monthKey)
	streak := prevStreak + 1
	monthDays := prevMonthDays + 1
	cycleDay, _, _ := CheckinCycle(cfg, streak)

	items, detail := CheckinRewards(cfg, CheckinRewardContext{
		Streak:        streak,
		PrevStreak:    prevStreak,
		CycleDay:      cycleDay,
		MonthDays:     monthDays,
		PrevMonthDays: prevMonthDays,
	})
	rolled := RollRewardItems(items)
	total, granted := GroupRewardItems(rolled)

	tips := PickCheckinTips(cfg)
	record := Checkin{
		Uid:      uid,
		Date:     dateKey,
		Days:     streak,
		Cycle:    cycleDay,
		Month:    monthKey,
		Source:   CheckinSourceSign,
		Exp:      cast.ToInt(total["exp"]),
		Integral: cast.ToInt(total["integral"]),
		Tips:     tips,
		Ip:       ip,
		Rewards: utils.Json.Encode(facade.H{
			"items":  granted,
			"total":  total,
			"detail": detail,
		}),
	}

	// 发放结果里的附加信息（卡密明文、降级说明等）只有真正发放时才知道，这里回填
	var grantedItems []facade.H

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		grantResult, err := GrantRewardsTx(tx, uid, rolled, facade.H{
			"type":        "check-in",
			"description": CheckinName(cfg),
			"bind_type":   "checkin",
			"bind_id":     record.Id,
			"json": facade.H{
				"source": CheckinSourceSign,
				"streak": streak,
				"cycle":  cycleDay,
			},
		})
		if err != nil {
			return err
		}

		if list, ok := grantResult["items"].([]facade.H); ok {
			grantedItems = list
		}
		if len(grantedItems) == 0 {
			return nil
		}

		// 用「实际发放明细」覆盖记录里的奖励信息（含卡密等发放结果）
		record.Rewards = utils.Json.Encode(facade.H{
			"items":  grantedItems,
			"total":  total,
			"detail": detail,
		})
		return tx.Model(&Checkin{}).Where("id", record.Id).
			Update("rewards", record.Rewards).Error
	})

	if err != nil {
		// 并发下可能已被其它请求抢先签到
		if _, exist := CheckinRecord(uid, dateKey); exist {
			return nil, errors.New("今天已经签到过了！")
		}
		facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid}, "签到失败")
		return nil, errors.New("签到失败，请稍后重试！")
	}

	if len(grantedItems) > 0 {
		granted = grantedItems
	}

	// 站内信（可选，默认关闭）：拿到卡密时一并带上，方便用户去兑换
	if cast.ToInt(cfg["notice"]) == 1 {
		go func() {
			content := "今日签到成功，获得：" + checkinGrantText(granted) +
				"。已连续签到 " + cast.ToString(streak) + " 天。"
			_, _ = (&Notification{}).CreateNotification(uid, 0, NotificationTypeSystem, CheckinName(cfg), content, "", 0)
		}()
	}

	return facade.H{
		"checked":    true,
		"today":      dateKey,
		"days":       streak,
		"streak":     streak,
		"cycle_day":  cycleDay,
		"month_days": monthDays,
		"items":      granted,
		"rewards":    detail,
		"total":      total,
		"cards":      checkinCardsOf(granted),
		// 本次跨越到的档位（前端提示「达成连签 7 天里程碑」）
		"crossed": facade.H{
			"streak":  CheckinCrossedMilestones(cfg, prevStreak, streak),
			"monthly": CheckinCrossedMonthly(cfg, prevMonthDays, monthDays),
		},
		"tips":           tips,
		"milestone":      detail[CheckinGroupMilestone],
		"next_milestone": CheckinNextMilestone(cfg, streak),
	}, nil
}

// CheckinMakeup - 补签（消耗积分）
//
// 发放口径：补签奖励（默认基础奖励）+ 若补签把断掉的连签补起来，按「跨越」口径补发里程碑 / 月档位奖励
// （周期奖励与连签加成不补，避免刷奖励）。
//
// 约束：只能补最近 makeup.days 天内、没有签到过的日期；每月最多 makeup.limit 次；
// 每次消耗 makeup.cost（资产由 makeup.asset 指定，默认积分）。
func CheckinMakeup(uid int, date int, ip string) (result facade.H, err error) {
	if uid <= 0 {
		return nil, errors.New("请先登录！")
	}

	cfg := GetCheckinConfig()
	if !CheckinEnabled(cfg) {
		return nil, errors.New("签到功能已关闭！")
	}

	mk := asStringMap(cfg["makeup"])
	if cast.ToInt(mk["enabled"]) == 0 {
		return nil, errors.New("补签功能已关闭！")
	}

	today := CheckinDayTime(time.Now())
	if date <= 0 {
		date = CheckinDateKey(today.AddDate(0, 0, -1))
	}
	if date >= CheckinDateKey(today) {
		return nil, errors.New("只能补签今天之前的日期！")
	}

	days := cast.ToInt(mk["days"])
	if days <= 0 {
		days = 7
	}
	if min := CheckinDateKey(today.AddDate(0, 0, -days)); date < min {
		return nil, errors.New("超出可补签范围！")
	}

	if _, exist := CheckinRecord(uid, date); exist {
		return nil, errors.New("该日期已经签到过了！")
	}

	monthKey := date / 100
	limit := cast.ToInt(mk["limit"])
	if limit > 0 && CheckinMonthMakeupCount(uid, monthKey) >= limit {
		return nil, errors.New("本月补签次数已用完！")
	}

	// 奖励：补签奖励（默认基础奖励）
	items, detail := CheckinRewards(cfg, CheckinRewardContext{Makeup: true})
	rolled := RollRewardItems(items)
	total, granted := GroupRewardItems(rolled)

	// 消耗：作为一笔「负向奖励」交给同一套引擎处理（余额不足会返回明确错误）
	costAsset := cast.ToString(mk["asset"])
	if utils.Is.Empty(costAsset) {
		costAsset = "integral"
	}
	cost := cast.ToInt(mk["cost"])
	if cost > 0 {
		rolled = append(rolled, RewardItem{
			Asset: costAsset,
			Value: -cost,
			Label: "补签消耗",
			Group: CheckinGroupMakeup,
		})
	}

	tips := PickCheckinTips(cfg)
	makeupTime := CheckinDateKeyToTime(date)
	record := Checkin{
		Uid:      uid,
		Date:     date,
		Cycle:    0,
		Month:    monthKey,
		Source:   CheckinSourceMakeup,
		Exp:      cast.ToInt(total["exp"]),
		Integral: cast.ToInt(total["integral"]),
		Tips:     tips,
		Ip:       ip,
		Rewards: utils.Json.Encode(facade.H{
			"items":  granted,
			"total":  total,
			"detail": detail,
			"cost":   facade.H{costAsset: cost},
		}),
	}

	// 补签当天的连签天数：插入后（同一事务内查不到未提交的数据，故提交后再回填）
	var grantedItems []facade.H

	err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		grantResult, err := GrantRewardsTx(tx, uid, rolled, facade.H{
			"type":        "check-in",
			"description": CheckinName(cfg) + "（补签）",
			"bind_type":   "checkin",
			"bind_id":     record.Id,
			"json": facade.H{
				"source": CheckinSourceMakeup,
				"date":   date,
			},
		})
		if err != nil {
			return err
		}

		if list, ok := grantResult["items"].([]facade.H); ok {
			grantedItems = list
		}
		if len(grantedItems) == 0 {
			return nil
		}

		record.Rewards = utils.Json.Encode(facade.H{
			"items":  grantedItems,
			"total":  total,
			"detail": detail,
			"cost":   facade.H{costAsset: cost},
		})
		return tx.Model(&Checkin{}).Where("id", record.Id).
			Update("rewards", record.Rewards).Error
	})

	if err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid, "date": date}, "补签失败")
		if _, exist := CheckinRecord(uid, date); exist {
			return nil, errors.New("该日期已经签到过了！")
		}
		return nil, err
	}

	if len(grantedItems) > 0 {
		granted = grantedItems
	}

	// 补签可能把断掉的连签补起来，从而「跨越」里程碑或月档位：
	// 按跨越口径（before < 档位 <= after）补发一次，既不会漏也不会重复。
	streakBefore := CheckinStreakOf(uid, makeupTime.AddDate(0, 0, -1))
	streakAfter := CheckinCurrentStreak(uid)
	monthAfter := CheckinMonthDays(uid, monthKey)
	monthBefore := monthAfter - 1

	crossed := facade.H{
		"streak":  CheckinCrossedMilestones(cfg, streakBefore, streakAfter),
		"monthly": CheckinCrossedMonthly(cfg, monthBefore, monthAfter),
	}

	crossItems := append(
		CheckinMilestoneItems(cfg, streakBefore, streakAfter),
		CheckinMonthlyItems(cfg, monthBefore, monthAfter)...,
	)

	if len(crossItems) > 0 {
		crossRolled := RollRewardItems(crossItems)
		crossTotal, crossItemsMap := GroupRewardItems(crossRolled)

		grantResult, grantErr := GrantRewards(uid, crossRolled, facade.H{
			"type":        "check-in",
			"description": CheckinName(cfg) + "（补签达成）",
			"bind_type":   "checkin",
			"bind_id":     record.Id,
			"json": facade.H{
				"source": CheckinSourceMakeup,
				"date":   date,
				"streak": streakAfter,
			},
		})
		if grantErr != nil {
			// 补签本身已成功，这里失败只记日志不回滚（用户下次签到不会重复发放）
			facade.Log.Error(map[string]any{"error": grantErr.Error(), "uid": uid, "date": date}, "补签跨越奖励发放失败")
		} else {
			if list, ok := grantResult["items"].([]facade.H); ok && len(list) > 0 {
				granted = append(granted, list...)

				for asset, value := range crossTotal {
					total[asset] = cast.ToInt(total[asset]) + cast.ToInt(value)
				}
				for _, item := range crossItemsMap {
					group := cast.ToString(item["group"])
					list, _ := detail[group].([]facade.H)
					detail[group] = append(list, item)
				}

				record.Rewards = utils.Json.Encode(facade.H{
					"items":  granted,
					"total":  total,
					"detail": detail,
					"cost":   facade.H{costAsset: cost},
				})
				facade.DB.Model(&Checkin{}).Where("id", record.Id).
					Update("rewards", record.Rewards)
			}
		}
	}

	// 回填连签天数（补签可能把断掉的连签串起来）
	daysAfter := CheckinStreakOf(uid, makeupTime)
	if daysAfter > 0 && daysAfter != record.Days {
		facade.DB.Model(&Checkin{}).Where([]any{
			[]any{"uid", "=", uid},
			[]any{"date", "=", date},
		}).Update("days", daysAfter)
	}

	return facade.H{
		"date":       date,
		"date_text":  CheckinDateKeyText(date),
		"days":       daysAfter,
		"items":      granted,
		"rewards":    detail,
		"total":      total,
		"cards":      checkinCardsOf(granted),
		"crossed":    crossed,
		"cost":       facade.H{costAsset: cost},
		"tips":       tips,
		"month_days": CheckinMonthDays(uid, monthKey),
		"makeup":     CheckinMakeupInfo(uid, cfg),
	}, nil
}

// checkinGroupNames - 奖励来源的中文名（文案里说明「这些奖励是怎么来的」用）
var checkinGroupNames = map[string]string{
	CheckinGroupBase:      "基础",
	CheckinGroupCycle:     "周期",
	CheckinGroupStreak:    "连签加成",
	CheckinGroupMilestone: "里程碑",
	CheckinGroupMonthly:   "月度",
	CheckinGroupRandom:    "随机",
	CheckinGroupMakeup:    "补签",
}

// checkinGrantText - 把发放明细整理成一句人话（站内信 / 文案用）
//
// 规则：
//  1. 同一种奖励合并成合计：「4 经验值、2 积分」，不会再出现「2 经验值、2 经验值」；
//  2. 来源多于一种时，再用括号说明构成：「（基础 2 经验值 + 2 积分 · 连签加成 +2 经验值）」；
//  3. 卡密按张数展示，并带上明文与兑换入口（它的 value 是面额，不能拼成「50 张」）。
func checkinGrantText(items []facade.H) string {
	if len(items) == 0 {
		return "无"
	}

	assetOrder := make([]string, 0)
	assetValue := facade.H{}
	assetUnit := facade.H{} // asset → 单位（以配置里的资产清单为准）

	groupOrder := make([]string, 0)
	groupValue := facade.H{} // group → []string（构成说明）

	cardCount := 0
	codes := make([]string, 0)

	for _, item := range items {
		asset := cast.ToString(item["asset"])
		group := cast.ToString(item["group"])

		if asset == RewardCardAssetKey {
			cardCount++
			extra := asStringMap(item["extra"])
			if code := cast.ToString(extra["card"]); !utils.Is.Empty(code) {
				codes = append(codes, code+"（面额 "+cast.ToString(extra["value"])+" 积分）")
			}
			continue
		}

		value := cast.ToInt(item["value"])
		if _, exist := assetValue[asset]; !exist {
			assetOrder = append(assetOrder, asset)
		}
		assetValue[asset] = cast.ToInt(assetValue[asset]) + value
		if cast.ToString(assetUnit[asset]) == "" {
			assetUnit[asset] = cast.ToString(item["unit"])
		}

		if utils.Is.Empty(group) {
			continue
		}
		// 基础奖励写原值，额外来源（连签加成 / 里程碑 …）加「+」更直观
		prefix := "+"
		if group == CheckinGroupBase {
			prefix = ""
		}
		list, exist := groupValue[group].([]string)
		if !exist {
			groupOrder = append(groupOrder, group)
		}
		groupValue[group] = append(list, prefix+cast.ToString(value)+" "+cast.ToString(item["unit"]))
	}

	total := make([]string, 0, len(assetOrder)+1)
	for _, asset := range assetOrder {
		total = append(total, cast.ToString(assetValue[asset])+" "+unitOfAsset(asset, cast.ToString(assetUnit[asset])))
	}
	if cardCount > 0 {
		total = append(total, cast.ToString(cardCount)+" 张卡密")
	}

	result := strings.Join(total, "、")
	if utils.Is.Empty(result) {
		result = "无"
	}

	// 来源多于一种时写清楚构成，避免用户以为发错了
	if len(groupOrder) > 1 {
		breakdown := make([]string, 0, len(groupOrder))
		for _, group := range groupOrder {
			list, _ := groupValue[group].([]string)
			name := checkinGroupNames[group]
			if utils.Is.Empty(name) {
				name = group
			}
			breakdown = append(breakdown, name+" "+strings.Join(list, " + "))
		}
		result += "（" + strings.Join(breakdown, " · ") + "）"
	}

	if len(codes) > 0 {
		result += "。卡密：" + strings.Join(codes, "；") + "，到「我的积分 → 卡密兑换」兑换"
	}

	return result
}

// unitOfAsset - 资产单位（优先取注册表里的单位，兜底用奖励项自带的单位 / 资产标识）
func unitOfAsset(asset string, fallback string) string {
	if item, exist := RewardAssetOf(asset); exist && !utils.Is.Empty(item.Unit) {
		return item.Unit
	}
	if !utils.Is.Empty(fallback) {
		return fallback
	}
	return asset
}
