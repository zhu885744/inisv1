package model

import (
	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Config struct {
	Id     int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Key    string `gorm:"size:32; comment:唯一键; default:Null;" json:"key"`
	Value  string `gorm:"type:text; comment:值; default:Null;" json:"value"`
	Remark string `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitConfig - 初始化Config表
func InitConfig() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&Config{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Config表迁移失败")
		return
	}

	configs := []Config{
		{Key: "SYSTEM_API_KEY", Value: "0", Remark: "API KEY验证"},
		{Key: "SYSTEM_QPS", Value: "1", Json: utils.Json.Encode(facade.H{
			"point": 15, "global": 50,
		}), Remark: "接口限流器（QPS）"},
		{Key: "SYSTEM_QPS_BLOCK", Value: "0", Json: utils.Json.Encode(facade.H{
			"count": 3, "second": "60 * 60",
		}), Remark: "满足QPS阈值后自动拦截"},
		{Key: "SYSTEM_PAGE_LIMIT", Value: "1", Text: "50", Remark: "限制分页查询单次最大数据量"},
		{Key: "ALLOW_REGISTER", Value: "1", Remark: "是否允许用户自行注册"},
		{Key: "PAGE", Json: utils.Json.Encode(facade.H{
			"editor": "tinymce", "comment": facade.H{"allow": 1, "show": 1}, "audit": 1,
		}), Remark: "页面配置"},
		{Key: "ARTICLE", Json: utils.Json.Encode(facade.H{
			"editor": "tinymce", "comment": facade.H{"allow": 1, "show": 1}, "audit": 1,
		}), Remark: "主题配置"},
		{Key: "COMMENT", Json: utils.Json.Encode(facade.H{
			"rate_limit":       facade.H{"enabled": 1, "max_count": 5, "time_window": 60},
			"max_length":       500,
			"require_chinese":  1,
			"sensitive_filter": 1,
			"sensitive_words":  []string{"色情", "广告", "开户"},
			"email_notify":     facade.H{"enabled": 1, "retry_count": 3, "retry_interval": 5},
		}), Remark: "评论配置"},
	}

	for _, item := range configs {
		if facade.DB.Model(&Config{}).Where("key", item.Key).Exist() {
			continue
		}
		facade.DB.Model(&item).Create(&item)
	}
}

// AfterFind - 查询Hook
func (this *Config) AfterFind(*gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}
