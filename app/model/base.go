package model

import (
	"fmt"
	"inis/app/facade"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

func task() {
	// 如果存在安装锁，表示还没进行初始化安装，不进行自动迁移
	if !utils.File().Exist("install.lock") {
		// 结束任务
		gocron.Remove(task)
		// 初始化数据库
		facade.WatchDB(true)
		// 检查是否开启自动迁移
		if cast.ToBool(facade.NewToml(facade.TomlDb).Get("mysql.migrate")) {
			go InitTable()
		}
	}
}

// InitTable - 初始化数据库表 - 自动迁移
func InitTable() {

	allow := []struct {
		name string
		fn   func()
	}{{
		"ApiKeys", InitApiKeys},
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
		// 为每个表的初始化添加超时控制
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

		// 等待初始化完成，最多等待30秒
		select {
		case <-done:
			// 初始化完成
		case <-time.After(30 * time.Second):
			// 超时
			facade.Log.Error(map[string]any{}, fmt.Sprintf("初始化%s表超时", item.name))
		}
	}
}

func init() {
	if err := gocron.Every(1).Second().Do(task); err != nil {
		return
	}
	// 启动调度器
	gocron.Start()
}

// DomainTemp1 域名模板替换
func DomainTemp1() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)
	storage := []string{"oss", "cos", "kodo"}
	// 模板变量替换
	for _, val := range storage {
		// 优先使用配置文件中的域名
		if !utils.Is.Empty(toml.Get(val + ".domain")) {
			replace["{{"+val+"}}"] = cast.ToString(toml.Get(val + ".domain"))
			continue
		}
		// 如果配置文件中没有域名，则使用默认域名
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
	// 本地域名
	localhost := facade.Var.Get("domain")
	if !utils.Is.Empty(localhost) {
		replace["{{localhost}}"] = cast.ToString(localhost)
	}
	if !utils.Is.Empty(facade.Cache.Get("domain")) {
		replace["{{localhost}}"] = cast.ToString(facade.Cache.Get("domain"))
	}

	return replace
}

// DomainTemp2 域名模板替换
func DomainTemp2() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)
	storage := []string{"oss", "cos", "kodo"}
	// 拼接自定义域名
	for _, val := range storage {
		if !utils.Is.Empty(toml.Get(val + ".domain")) {
			replace[cast.ToString(toml.Get(val+".domain"))] = "{{" + val + "}}"
		}
	}
	// 拼接本地域名
	localhost := facade.Var.Get("domain")
	if !utils.Is.Empty(localhost) {
		replace[cast.ToString(localhost)] = "{{localhost}}"
	}
	if !utils.Is.Empty(facade.Cache.Get("domain")) {
		replace[cast.ToString(facade.Cache.Get("domain"))] = "{{localhost}}"
	}

	// 拼接 oss 域名
	oss := fmt.Sprintf("https://%s.%s",
		cast.ToString(toml.Get("oss.bucket")),
		cast.ToString(toml.Get("oss.endpoint")),
	)
	// 拼接 cos 域名
	cos := fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com",
		cast.ToString(toml.Get("cos.bucket")),
		cast.ToString(toml.Get("cos.app_id")),
		cast.ToString(toml.Get("cos.region")),
	)
	replace[oss] = "{{oss}}"
	replace[cos] = "{{cos}}"

	return replace
}
