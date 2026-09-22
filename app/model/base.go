package model

import (
	"fmt"
	"inis/app/facade"
	"strings"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 公共常量
const (
	installLockFile = "install.lock"
	adminGroupId    = 1
)

// task - 定时任务
func task() {
	if !utils.File().Exist(installLockFile) {
		gocron.Remove(task)
		facade.WatchDB(true)
		if cast.ToBool(facade.NewToml(facade.TomlDb).Get("mysql.migrate")) {
			go InitTable()
		}
	}
}

// InitTable - 初始化数据库表
func InitTable() {
	allow := []struct {
		name string
		fn   func()
	}{
		{"ApiKeys", InitApiKeys},
		{"Article", InitArticle},
		{"ArticleGroup", InitArticleGroup},
		{"AuthPages", InitAuthPages},
		{"AuthRules", InitAuthRules},
		{"Banner", InitBanner},
		{"Comment", InitComment},
		{"Config", InitConfig},
		{"Links", InitLinks},
		{"LinksGroup", InitLinksGroup},
		{"Placard", InitPlacard},
		{"Tags", InitTags},
		{"Users", InitUsers},
		{"AuthGroup", InitAuthGroup},
		{"Pages", InitPages},
		{"Level", InitLevel},
		{"EXP", InitEXP},
		{"QpsWarn", InitQpsWarn},
		{"IpBlack", InitIpBlack},
		{"IpWhite", InitIpWhite},
		{"Moments", InitMoments},
		{"Attachment", InitAttachment},
		{"UserLikes", InitUserLikes},
		{"UserCollects", InitUserCollects},
		{"UserFollows", InitUserFollows},
		{"UserBanRecords", InitUserBanRecords},
		{"Notification", InitNotification},
		{"NotificationRead", InitNotificationRead},
		{"Integral", InitIntegral},
		{"Goods", InitGoods},
		{"IntegralCard", InitIntegralCard},
	}

	for _, item := range allow {
		done := make(chan struct{})
		go func(name string, fn func()) {
			defer func() {
				if err := recover(); err != nil {
					facade.Log.Error(map[string]any{
						"error": err,
					}, fmt.Sprintf("初始化%s表时发生错误", name))
				}
				close(done)
			}()

			facade.Log.Info(map[string]any{}, fmt.Sprintf("开始初始化%s表", name))
			fn()
			facade.Log.Info(map[string]any{}, fmt.Sprintf("初始化%s表完成", name))
		}(item.name, item.fn)

		select {
		case <-done:
		case <-time.After(30 * time.Second):
			facade.Log.Error(map[string]any{}, fmt.Sprintf("初始化%s表超时", item.name))
		}
	}
}

func init() {
	if err := gocron.Every(1).Second().Do(task); err != nil {
		return
	}
	gocron.Start()

	// 未安装（存在安装锁）时跳过数据库初始化，避免空配置导致连接失败 panic
	if utils.File().Exist(installLockFile) {
		return
	}

	facade.WatchDB(true)
	if cast.ToBool(facade.NewToml(facade.TomlDb).Get("mysql.migrate")) {
		go InitTable()
	}
}

// DomainTemp1 - 域名模板替换（查询时）
func DomainTemp1() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)
	storage := []string{"oss", "cos", "kodo"}

	for _, val := range storage {
		domain := cast.ToString(toml.Get(val + ".domain"))
		if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
			replace["{{"+val+"}}"] = domain
			continue
		}
		if utils.In.Array(val, []any{"oss", "cos"}) {
			if val == "oss" {
				replace["{{"+val+"}}"] = fmt.Sprintf("https://%s.%s",
					cast.ToString(toml.Get("oss.bucket")),
					cast.ToString(toml.Get("oss.endpoint")),
				)
			}
			if val == "cos" {
				replace["{{"+val+"}}"] = fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com",
					cast.ToString(toml.Get("cos.bucket")),
					cast.ToString(toml.Get("cos.app_id")),
					cast.ToString(toml.Get("cos.region")),
				)
			}
		}
	}

	localhost := facade.Var.Get("domain")
	if !utils.Is.Empty(localhost) {
		replace["{{localhost}}"] = cast.ToString(localhost)
	}
	if !utils.Is.Empty(facade.Cache.Get("domain")) {
		replace["{{localhost}}"] = cast.ToString(facade.Cache.Get("domain"))
	}

	return replace
}

// DomainTemp2 - 域名模板替换（保存时）
func DomainTemp2() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)
	storage := []string{"oss", "cos", "kodo"}

	for _, val := range storage {
		if !utils.Is.Empty(toml.Get(val + ".domain")) {
			replace[cast.ToString(toml.Get(val+".domain"))] = "{{" + val + "}}"
		}
	}

	localhost := facade.Var.Get("domain")
	if !utils.Is.Empty(localhost) {
		replace[cast.ToString(localhost)] = "{{localhost}}"
	}
	if !utils.Is.Empty(facade.Cache.Get("domain")) {
		replace[cast.ToString(facade.Cache.Get("domain"))] = "{{localhost}}"
	}

	oss := fmt.Sprintf("https://%s.%s",
		cast.ToString(toml.Get("oss.bucket")),
		cast.ToString(toml.Get("oss.endpoint")),
	)
	cos := fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com",
		cast.ToString(toml.Get("cos.bucket")),
		cast.ToString(toml.Get("cos.app_id")),
		cast.ToString(toml.Get("cos.region")),
	)
	replace[oss] = "{{oss}}"
	replace[cos] = "{{cos}}"

	return replace
}

// ReplaceDomainColumns - 替换 Column() 查询结果里的域名模板（查询时口径）
//
// 背景：Column() 内部走 Scan 到 []map[string]any，不会实例化模型结构体，
// 因此模型上的 AfterFind 钩子不会被触发，而头像 / 附件地址里的 {{cos}}、{{oss}}、
// {{kodo}}、{{localhost}} 等存储模板正是在 AfterFind 里还原成真实域名的。
// 凡是用 Column() 取回了这类字段（如 user 的 avatar），都需要调用本函数补一次替换，
// 否则接口会把模板原样返回给前端，导致图片无法显示。
//
// @param rows   Column() 的返回值（[]map[string]any）
// @param fields 需要替换的字段名，如 "avatar"
func ReplaceDomainColumns(rows any, fields ...string) []map[string]any {

	replace := DomainTemp1()
	if utils.Is.Empty(replace) || len(fields) == 0 {
		return castToColumnSlice(rows)
	}

	result := make([]map[string]any, 0)
	for _, item := range cast.ToSlice(rows) {
		row := cast.ToStringMap(item)
		for _, field := range fields {
			value := cast.ToString(row[field])
			if utils.Is.Empty(value) {
				continue
			}
			row[field] = utils.Replace(value, replace)
		}
		result = append(result, row)
	}

	return result
}

// castToColumnSlice - 把 Column() 的返回值统一成 []map[string]any（不做替换时的兜底）
func castToColumnSlice(rows any) []map[string]any {
	result := make([]map[string]any, 0)
	for _, item := range cast.ToSlice(rows) {
		result = append(result, cast.ToStringMap(item))
	}
	return result
}
