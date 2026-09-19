package model

import (
	"errors"
	"inis/app/facade"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// IntegralCacheKey - 积分规则缓存键
const IntegralCacheKey = "SYSTEM_INTEGRAL_RULES"

// 积分流水类型常量
const (
	IntegralTypeCheckIn       = "check-in"       // 每日签到
	IntegralTypeLogin         = "login"          // 每日登录
	IntegralTypeArticleCreate = "article-create" // 发布文章
	IntegralTypeComment       = "comment"        // 发表评论
	IntegralTypeMoments       = "moments"        // 发布动态
	IntegralTypeShare         = "share"          // 分享内容
	IntegralTypeBuy           = "buy"            // 商城兑换（消耗）
	IntegralTypeRefund        = "refund"         // 订单取消/退款返还（获得）
	IntegralTypeGive          = "give"           // 管理员调整
)

// defaultIntegralRules - 默认积分任务规则（作为配置缺失时的兜底）
// value: 单次积分；daily_limit: 每日可完成次数（0 = 不限制）；icon: 前端展示图标
func defaultIntegralRules() map[string]facade.H {
	return map[string]facade.H{
		IntegralTypeCheckIn:       {"name": "每日签到", "value": 5, "daily_limit": 1, "icon": "bi-calendar-check"},
		IntegralTypeLogin:         {"name": "每日登录", "value": 2, "daily_limit": 1, "icon": "bi-box-arrow-in-right"},
		IntegralTypeArticleCreate: {"name": "发布文章", "value": 10, "daily_limit": 5, "icon": "bi-file-earmark-text"},
		IntegralTypeComment:       {"name": "发表评论", "value": 2, "daily_limit": 10, "icon": "bi-chat-dots"},
		IntegralTypeMoments:       {"name": "发布动态", "value": 20, "daily_limit": 1, "icon": "bi-lightning"},
		IntegralTypeShare:         {"name": "分享内容", "value": 2, "daily_limit": 3, "icon": "bi-share"},
	}
}

// GetIntegralConfig - 获取积分任务规则（缓存优先）
func GetIntegralConfig() map[string]facade.H {
	// 默认规则（始终作为兜底）
	defaultConfig := defaultIntegralRules()

	// 优先从缓存获取
	if facade.Cache.Has(IntegralCacheKey) {
		if data, ok := facade.Cache.Get(IntegralCacheKey).(map[string]facade.H); ok {
			for k, v := range defaultConfig {
				if _, exists := data[k]; !exists {
					data[k] = v
				}
			}
			return data
		}
	}

	// 从数据库获取
	item, _ := facade.DB.Model(&Config{}).Where("key", IntegralCacheKey).Find()
	if !utils.Is.Empty(item) {
		if jsonData, ok := item["json"].(map[string]any); ok {
			config := make(map[string]facade.H)
			for k, v := range jsonData {
				if vMap, ok := v.(map[string]any); ok {
					config[k] = facade.H(vMap)
				}
			}
			for k, v := range defaultConfig {
				if _, exists := config[k]; !exists {
					config[k] = v
				}
			}
			facade.Cache.Set(IntegralCacheKey, config)
			return config
		}
	}

	return defaultConfig
}

// Integral - 积分流水表
type Integral struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid         int    `gorm:"type:int(32); index; comment:用户ID;" json:"uid"`
	Value       int    `gorm:"type:int(32); comment:积分值（正=获得 负=消耗）; default:0;" json:"value"`
	Type        string `gorm:"comment:类型; default:'default';" json:"type"`
	Description string `gorm:"comment:描述; default:Null;" json:"description"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitIntegral - 初始化积分表
func InitIntegral() {
	err := facade.DB.Drive().AutoMigrate(&Integral{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Integral表迁移失败")
		return
	}
	// 创建索引（流水查询以 uid + create_time 为主）
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_integral_uid ON inis_integral(uid)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_integral_type ON inis_integral(type)")
	facade.DB.Drive().Exec("CREATE INDEX IF NOT EXISTS idx_integral_create_time ON inis_integral(create_time)")
}

// AfterFind - 查询Hook
func (this *Integral) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// IntegralBalance - 查询用户积分余额
func IntegralBalance(uid int) int {
	user, _ := facade.DB.Model(&Users{}).Where("id", uid).Find()
	return cast.ToInt(user["integral"])
}

// integralTodayStart - 今日 0 点时间戳
func integralTodayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// IntegralSummary - 积分概览：余额 + 累计收支 + 今日收支
func IntegralSummary(uid int) facade.H {
	result := facade.H{
		"integral":      IntegralBalance(uid),
		"total_income":  0,
		"total_expense": 0,
		"today_income":  0,
		"today_expense": 0,
	}

	// 累计收支
	var total struct {
		Income  int
		Expense int
	}
	facade.DB.Drive().Raw(
		"SELECT COALESCE(SUM(CASE WHEN value > 0 THEN value ELSE 0 END), 0) AS income, "+
			"COALESCE(SUM(CASE WHEN value < 0 THEN -value ELSE 0 END), 0) AS expense "+
			"FROM inis_integral WHERE uid = ? AND (delete_time IS NULL OR delete_time = 0)",
		uid,
	).Scan(&total)

	result["total_income"] = total.Income
	result["total_expense"] = total.Expense

	// 今日收支
	var today struct {
		Income  int
		Expense int
	}
	facade.DB.Drive().Raw(
		"SELECT COALESCE(SUM(CASE WHEN value > 0 THEN value ELSE 0 END), 0) AS income, "+
			"COALESCE(SUM(CASE WHEN value < 0 THEN -value ELSE 0 END), 0) AS expense "+
			"FROM inis_integral WHERE uid = ? AND create_time >= ? AND (delete_time IS NULL OR delete_time = 0)",
		uid, integralTodayStart().Unix(),
	).Scan(&today)

	result["today_income"] = today.Income
	result["today_expense"] = today.Expense

	return result
}

// IntegralTodayCount - 今日某类型流水的条数与累计值
func IntegralTodayCount(uid int, typ string) (count int, total int) {
	var row struct {
		Count int
		Total int
	}
	facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS count, COALESCE(SUM(value), 0) AS total FROM inis_integral "+
			"WHERE uid = ? AND type = ? AND create_time >= ? AND (delete_time IS NULL OR delete_time = 0)",
		uid, typ, integralTodayStart().Unix(),
	).Scan(&row)
	return row.Count, row.Total
}

// IntegralTasks - 今日积分任务进度（登录用户）
// 返回：任务列表（含今日完成次数、进度、可获积分）+ 今日合计 + 连签信息
func IntegralTasks(uid int) facade.H {
	config := GetIntegralConfig()
	today := integralTodayStart()

	// 一次性查出今日各类型完成情况
	var rows []map[string]any
	facade.DB.Drive().Raw(
		"SELECT type, COUNT(id) AS count, COALESCE(SUM(value), 0) AS total FROM inis_integral "+
			"WHERE uid = ? AND create_time >= ? AND (delete_time IS NULL OR delete_time = 0) GROUP BY type",
		uid, today.Unix(),
	).Scan(&rows)

	countMap := make(map[string]int)
	valueMap := make(map[string]int)
	for _, row := range rows {
		key := cast.ToString(row["type"])
		countMap[key] = cast.ToInt(row["count"])
		valueMap[key] = cast.ToInt(row["total"])
	}

	list := make([]facade.H, 0, len(config))
	todayIncome := 0
	doneCount := 0

	for key, rule := range config {
		limit := cast.ToInt(rule["daily_limit"])
		count := countMap[key]
		income := valueMap[key]
		done := limit > 0 && count >= limit
		if done {
			doneCount++
		}
		if income > 0 {
			todayIncome += income
		}

		// 进度百分比（不限制的任务固定 100%）
		progress := 100
		if limit > 0 {
			progress = int(float64(count) / float64(limit) * 100)
			if progress > 100 {
				progress = 100
			}
		}

		// 未完成的任务展示"还能拿多少分"
		remain := 0
		if limit == 0 {
			remain = cast.ToInt(rule["value"])
		} else if count < limit {
			remain = (limit - count) * cast.ToInt(rule["value"])
		}

		list = append(list, facade.H{
			"type":         key,
			"name":         rule["name"],
			"icon":         rule["icon"],
			"value":        cast.ToInt(rule["value"]),
			"daily_limit":  limit,
			"today_count":  count,
			"today_income": income,
			"progress":     progress,
			"done":         done,
			"remain":       remain,
		})
	}

	// 连签信息（与签到任务共用 exp 表，复用经验侧连续签到计算）
	todayTime := time.Now()
	if _, ok := countMap[IntegralTypeCheckIn]; ok {
		// 今日已签到：连续天数含今天
	} else {
		todayTime = todayTime.AddDate(0, 0, -1)
	}
	streak := CheckInStreak(uid, time.Date(todayTime.Year(), todayTime.Month(), todayTime.Day(), 0, 0, 0, 0, todayTime.Location()))

	return facade.H{
		"list":         list,
		"today_income": todayIncome,
		"done_count":   doneCount,
		"total_count":  len(list),
		"streak":       streak,
	}
}

// IntegralRank - 积分排行榜
// by: earned（累计获得，默认）/ balance（当前余额）
// 返回: { list: [...], my_rank, my_value, by }
func IntegralRank(by string, limit int, currentUid int) facade.H {
	if by != "balance" {
		by = "earned"
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	list := make([]facade.H, 0, limit)
	myRank := 0
	myValue := 0

	fillUser := func(uid int) facade.H {
		user, _ := facade.DB.Model(&Users{}).Where("id", uid).Find()
		return facade.H{
			"id":          uid,
			"nickname":    cast.ToString(user["nickname"]),
			"avatar":      cast.ToString(user["avatar"]),
			"description": cast.ToString(user["description"]),
			"title":       cast.ToString(user["title"]),
		}
	}

	if by == "balance" {
		var rows []map[string]any
		facade.DB.Drive().Raw(
			"SELECT id, integral FROM inis_users WHERE integral > 0 AND (delete_time IS NULL OR delete_time = 0) "+
				"ORDER BY integral DESC, id ASC LIMIT ?", limit,
		).Scan(&rows)

		for index, row := range rows {
			uid := cast.ToInt(row["id"])
			item := fillUser(uid)
			item["rank"] = index + 1
			item["value"] = cast.ToInt(row["integral"])
			item["is_me"] = uid == currentUid
			list = append(list, item)
		}

		if currentUid > 0 {
			var mine struct{ Integral int }
			facade.DB.Drive().Raw("SELECT integral FROM inis_users WHERE id = ?", currentUid).Scan(&mine)
			myValue = mine.Integral

			var ahead int64
			facade.DB.Drive().Raw(
				"SELECT COUNT(*) FROM inis_users WHERE integral > ? AND (delete_time IS NULL OR delete_time = 0)", myValue,
			).Scan(&ahead)
			if myValue > 0 {
				myRank = int(ahead) + 1
			}
		}
	} else {
		var rows []map[string]any
		facade.DB.Drive().Raw(
			"SELECT uid, SUM(value) AS total FROM inis_integral WHERE value > 0 AND (delete_time IS NULL OR delete_time = 0) "+
				"GROUP BY uid ORDER BY total DESC, uid ASC LIMIT ?", limit,
		).Scan(&rows)

		for index, row := range rows {
			uid := cast.ToInt(row["uid"])
			item := fillUser(uid)
			item["rank"] = index + 1
			item["value"] = cast.ToInt(row["total"])
			item["is_me"] = uid == currentUid
			list = append(list, item)
		}

		if currentUid > 0 {
			var mine struct{ Total int }
			facade.DB.Drive().Raw(
				"SELECT COALESCE(SUM(value), 0) AS total FROM inis_integral WHERE uid = ? AND value > 0 AND (delete_time IS NULL OR delete_time = 0)",
				currentUid,
			).Scan(&mine)
			myValue = mine.Total

			if myValue > 0 {
				var ahead int64
				facade.DB.Drive().Raw(
					"SELECT COUNT(*) FROM (SELECT uid, SUM(value) AS total FROM inis_integral "+
						"WHERE value > 0 AND (delete_time IS NULL OR delete_time = 0) GROUP BY uid HAVING SUM(value) > ?) AS t",
					myValue,
				).Scan(&ahead)
				myRank = int(ahead) + 1
			}
		}
	}

	return facade.H{
		"list":     list,
		"by":       by,
		"my_rank":  myRank,
		"my_value": myValue,
	}
}

// Add - 增加/扣除积分
// 任务类型（规则配置内）自动从配置取值并校验每日限制；give/buy 等类型需显式传入 value
func (this *Integral) Add(table Integral) (err error) {

	if table.Uid == 0 {
		return errors.New("请先登录！")
	}

	config := GetIntegralConfig()

	// 确定积分值：任务类型未显式传值时从规则配置读取
	value := table.Value
	rule, isTask := config[table.Type]
	if value == 0 && isTask {
		value = cast.ToInt(rule["value"])
	}
	if value == 0 {
		return errors.New("积分值不能为0！")
	}

	// 任务奖励（获得积分）：校验每日限制（daily_limit <= 0 表示不限制）
	if isTask && value > 0 {
		if dailyLimit := cast.ToInt(rule["daily_limit"]); dailyLimit > 0 {
			count, _ := facade.DB.Model(&Integral{}).Where([]any{
				[]any{"uid", "=", table.Uid},
				[]any{"type", "=", table.Type},
				[]any{"create_time", ">=", integralTodayStart().Unix()},
			}).Count()

			if count >= int64(dailyLimit) {
				return errors.New("今日奖励已达上限！")
			}
		}
	}

	// 消耗积分：校验余额是否充足
	if value < 0 {
		if IntegralBalance(table.Uid) < -value {
			return errors.New("积分不足！")
		}
	}

	table.Value = value
	if utils.Is.Empty(table.Description) {
		if isTask {
			table.Description = cast.ToString(rule["name"])
		} else if table.Type == IntegralTypeRefund {
			table.Description = "订单退款返还"
		} else if value > 0 {
			table.Description = "积分调整"
		} else {
			table.Description = "积分消耗"
		}
	}

	// 余额快照：便于对账（调用方未显式传入 Json 时写入）
	if utils.Is.Empty(table.Json) {
		table.Json = utils.Json.Encode(map[string]any{
			"balance_after": IntegralBalance(table.Uid) + value,
		})
	}

	_, err = facade.DB.Model(&Integral{}).Create(&table)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "type": table.Type, "uid": table.Uid}, "积分记录创建失败")
		return err
	}

	// 更新用户积分余额
	_, _ = facade.DB.Model(&Users{}).Where("id", table.Uid).Inc("integral", value)

	facade.Log.Info(map[string]any{"uid": table.Uid, "type": table.Type, "value": value}, "积分变动成功")
	return nil
}
