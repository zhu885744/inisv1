package model

import (
	"fmt"
	"inis/app/facade"

	"github.com/jasonlvhit/gocron"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"time"
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
		{"Upgrade", InitUpgrade},
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
}

// DomainTemp1 - 域名模板替换（查询时）
func DomainTemp1() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)
	storage := []string{"oss", "cos", "kodo"}

	for _, val := range storage {
		if !utils.Is.Empty(toml.Get(val + ".domain")) {
			replace["{{"+val+"}}"] = cast.ToString(toml.Get(val + ".domain"))
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
