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
		// 开启密钥校验后，本站来源（同源浏览器请求 / 本机·内网直连）是否免密钥 —— **默认开**：
		// 内嵌在二进制里的 Mellow 前端不带 i-api-key，若不放行会让站内（含后台）全部 403。
		// 判定见 app/api/middleware/api-key.go 的 isLocalRequest；关掉即「所有请求都必须带密钥」
		{Key: "SYSTEM_API_KEY_LOCAL", Value: "1", Remark: "API KEY放行本站同源/内网请求"},
		{Key: "SYSTEM_QPS", Value: "1", Json: utils.Json.Encode(facade.H{
			"point": 15, "global": 50,
		}), Remark: "接口限流器（QPS）"},
		{Key: "SYSTEM_QPS_BLOCK", Value: "0", Json: utils.Json.Encode(facade.H{
			"count": 3, "second": "60 * 60",
		}), Remark: "满足QPS阈值后自动拦截"},
		{Key: "SYSTEM_QPS_NOTIFY", Value: "0", Json: utils.Json.Encode(facade.H{
			"email":   "",
			"webhook": "",
		}), Remark: "QPS自动封禁通知（邮件/Webhook）"},
		{Key: "SYSTEM_MAIL_NOTIFY", Value: "1", Json: utils.Json.Encode(MailNotifyDefaultConfig()), Remark: "统一邮件通知（各场景开关 + 管理员收件邮箱）"},
		{Key: "SYSTEM_PAGE_LIMIT", Value: "1", Text: "50", Remark: "限制分页查询单次最大数据量"},
		{Key: "ALLOW_REGISTER", Value: "1", Json: utils.Json.Encode(facade.H{
			// 邮箱域名限制：off 关闭 / whitelist 白名单 / blacklist 黑名单
			"email_domain_mode": "off",
			"email_whitelist":   []string{},
			"email_blacklist":   []string{},
			// 注册验证方式：none 直接注册 / manual 人工审核（历史配置里的 email 已废弃）
			"verify_mode": "none",
			// 注册成功后是否发送站内欢迎消息 / 欢迎邮件
			"welcome_message": 0,
			"welcome_email":   0,
			// 注：本记录的 text 字段另存「新用户默认权限组」ID 列表，见 model/register.go
		}), Remark: "是否允许用户自行注册 + 注册扩展设置（域名限制/验证方式/欢迎消息）"},
		{Key: "PAGE", Json: utils.Json.Encode(facade.H{
			"editor": "tinymce", "comment": facade.H{"allow": 1, "show": 1}, "audit": 1,
		}), Remark: "独立页面配置"},
		{Key: "ARTICLE", Json: utils.Json.Encode(facade.H{
			"editor": "tinymce", "comment": facade.H{"allow": 1, "show": 1}, "audit": 1,
		}), Remark: "文章配置"},
		{Key: "MOMENTS", Json: utils.Json.Encode(facade.H{
			"editor": "tinymce", "comment": facade.H{"allow": 1, "show": 1}, "audit": 1,
		}), Remark: "动态配置"},
		{Key: "COMMENT", Json: utils.Json.Encode(facade.H{
			"allow":            1,
			"rate_limit":       facade.H{"enabled": 1, "max_count": 5, "time_window": 60},
			"max_length":       500,
			"require_chinese":  1,
			"sensitive_filter": 1,
			"sensitive_words":  []string{"色情", "广告", "开户"},
			// 注：评论 / 回复的邮件开关已并入「统一邮件通知」（SYSTEM_MAIL_NOTIFY 的
			// comment.notify / comment.reply 场景），此处不再保留 email_notify
		}), Remark: "评论配置"},
		{Key: "SYSTEM_EXP_RULES", Json: utils.Json.Encode(facade.H{
			"like":    facade.H{"name": "点赞", "value": 1, "daily_limit": 10},
			"collect": facade.H{"name": "收藏", "value": 1, "daily_limit": 10},
			"visit":   facade.H{"name": "访问", "value": 1, "daily_limit": 10},
			"share":   facade.H{"name": "分享", "value": 1, "daily_limit": 10},
			"login":   facade.H{"name": "登录", "value": 5, "daily_limit": 1},
			"comment": facade.H{"name": "评论", "value": 1, "daily_limit": 10},
			// 注：签到规则已独立成 SYSTEM_CHECKIN_RULES（model/checkin.go），不再挂在经验规则里
			"moments":         facade.H{"name": "发布动态", "value": 50, "daily_limit": 1},
			"article-create":  facade.H{"name": "发布文章", "value": 5, "daily_limit": 10},
			"article-like":    facade.H{"name": "内容获赞", "value": 5, "daily_limit": 10},
			"article-collect": facade.H{"name": "内容被收藏", "value": 5, "daily_limit": 10},
			"comment-create":  facade.H{"name": "发表评论", "value": 5, "daily_limit": 10},
			"comment-like":    facade.H{"name": "评论获赞", "value": 5, "daily_limit": 10},
		}), Remark: "经验值规则配置"},
		{Key: "SYSTEM_INTEGRAL_RULES", Json: utils.Json.Encode(facade.H{
			"login":          facade.H{"name": "每日登录", "value": 2, "daily_limit": 1},
			"article-create": facade.H{"name": "发布文章", "value": 10, "daily_limit": 5},
			"comment":        facade.H{"name": "发表评论", "value": 2, "daily_limit": 10},
			"moments":        facade.H{"name": "发布动态", "value": 20, "daily_limit": 1},
		}), Remark: "积分规则配置"},
		// 签到配置（独立于经验 / 积分规则）：奖励项由奖励引擎统一发放，见 model/checkin.go
		{Key: CheckinCacheKey, Json: utils.Json.Encode(defaultCheckinConfig()), Remark: "每日签到配置"},
		// 装扮配置（头像框 / 头衔商城，见 model/decoration.go）
		{Key: DecorationConfigKey, Json: utils.Json.Encode(defaultDecorationConfig()), Remark: "装扮商城配置"},
	}

	for _, item := range configs {
		exist, _ := facade.DB.Model(&Config{}).Where("key", item.Key).Exist()
		if exist {
			continue
		}
		_, _ = facade.DB.Model(&item).Create(&item)
	}

	// 兼容旧库：清理历史配置里残留的签到规则（签到已独立到 SYSTEM_CHECKIN_RULES）
	cleanupLegacyCheckinRules()
}

// cleanupLegacyCheckinRules - 删除经验 / 积分规则里残留的 `check-in` 配置（幂等）
//
// 背景：签到原先挂在 SYSTEM_EXP_RULES 与 SYSTEM_INTEGRAL_RULES 里，独立成
// SYSTEM_CHECKIN_RULES 之后，这两份历史 JSON 里的 check-in 已经不再生效。
// 读取时虽然会忽略它，但后台规则页以「原始 JSON」为基底保存，会把它一直带下去，
// 因此在启动迁移时顺手删掉，让配置表保持干净。
func cleanupLegacyCheckinRules() {

	for _, key := range []string{ExpCacheKey, IntegralCacheKey} {

		item, _ := facade.DB.Model(&Config{}).Where("key", key).Find()
		if utils.Is.Empty(item) {
			continue
		}

		// json 可能是已解码的对象，也可能是原始字符串，两种都兼容
		jsonData := asStringMap(item["json"])
		if len(jsonData) == 0 {
			if raw := cast.ToString(item["json"]); !utils.Is.Empty(raw) {
				jsonData = asStringMap(utils.Json.Decode(raw))
			}
		}
		if len(jsonData) == 0 {
			continue
		}

		if _, exist := jsonData[IntegralTypeCheckIn]; !exist {
			continue
		}

		delete(jsonData, IntegralTypeCheckIn)

		if _, err := facade.DB.Model(&Config{}).Where("key", key).Update(map[string]any{
			"json": utils.Json.Encode(jsonData),
		}); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "key": key}, "清理签到规则失败")
			continue
		}

		// 清缓存，避免旧的（含 check-in 的）配置被继续读到
		facade.Cache.Del(key)
		facade.Log.Info(map[string]any{"key": key}, "已清理规则配置里残留的 check-in（签到已独立到 SYSTEM_CHECKIN_RULES）")
	}
}

// AfterFind - 查询Hook
func (this *Config) AfterFind(*gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}
