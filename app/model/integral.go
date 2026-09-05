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

// GetIntegralConfig - 获取积分规则（缓存优先）
func GetIntegralConfig() map[string]facade.H {
	// 默认规则（始终作为兜底）
	defaultConfig := map[string]facade.H{
		"check-in":       {"name": "每日签到", "value": 5, "daily_limit": 1},
		"login":          {"name": "每日登录", "value": 2, "daily_limit": 1},
		"article-create": {"name": "发布文章", "value": 10, "daily_limit": 5},
		"comment":        {"name": "发表评论", "value": 2, "daily_limit": 10},
		"moments":        {"name": "发布动态", "value": 20, "daily_limit": 1},
	}

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

// Add - 增加/扣除积分
// 任务类型（规则配置内）自动从配置取值并校验每日限制；give/buy 等类型需显式传入 value
func (this *Integral) Add(table Integral) (err error) {

	if table.Uid == 0 {
		return errors.New("请先登录！")
	}

	config := GetIntegralConfig()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 确定积分值：任务类型未显式传值时从规则配置读取
	value := table.Value
	rule, isTask := config[table.Type]
	if value == 0 && isTask {
		value = cast.ToInt(rule["value"])
	}
	if value == 0 {
		return errors.New("积分值不能为0！")
	}

	// 任务奖励（获得积分）：校验每日限制
	if isTask && value > 0 {
		count, _ := facade.DB.Model(&Integral{}).Where([]any{
			[]any{"uid", "=", table.Uid},
			[]any{"type", "=", table.Type},
			[]any{"create_time", ">=", today.Unix()},
		}).Count()

		if count >= cast.ToInt64(rule["daily_limit"]) {
			return errors.New("今日奖励已达上限！")
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
		} else if value > 0 {
			table.Description = "积分调整"
		} else {
			table.Description = "积分消耗"
		}
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
