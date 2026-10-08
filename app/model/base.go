package model

import (
	"fmt"
	"inis/app/facade"
	"strings"
	"sync"
	"time"

	"github.com/jasonlvhit/gocron"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 公共常量
const (
	adminGroupId = 1
)

// task - 定时任务（每秒一次）：安装完成的那一刻接管数据库初始化，装完不用重启服务
func task() {
	// 未完成安装时什么都不做。以前这里只看 install.lock，只拷一个二进制的部署
	// （既没有锁也没有数据库配置）会被判成「已安装」，立刻去连空数据库并 panic。
	// 口径见 facade.Installed：安装锁已解除且 database.toml 已生成。
	if !facade.Installed() {
		return
	}

	gocron.Remove(task)

	facade.WatchDB(true)
	if cast.ToBool(facade.NewToml(facade.TomlDb).Get("mysql.migrate")) {
		go InitTable()
	}
}

// initTableMutex - 初始化数据库表的进程内串行锁
//
// 有两路会触发初始化：包 init()（已安装时随启动即跑）与 gocron 的 task()（安装完成那一刻跑）。
// 两者可能在同一秒内先后触发，而各表的数据初始化都是「先查数量、再插入」的形式，
// 并发执行会插出重复的默认数据（典型症状：后台出现两条一模一样的「默认分类」）。
// 这里把整轮初始化串起来，第二个调用者会等前一轮跑完，从而看到刚插进去的数据。
var initTableMutex sync.Mutex

// InitTable - 初始化数据库表
//
// 注意：各 Init* 里的「数据初始化」必须同步完成（不要另起 go），
// 否则下面的等待与超时覆盖不到，别的初始化流程仍可能与它重复插入。
func InitTable() {
	initTableMutex.Lock()
	defer initTableMutex.Unlock()

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
		{"UserOauth", InitUserOauth},
		{"AuthGroup", InitAuthGroup},
		{"Pages", InitPages},
		{"Level", InitLevel},
		{"EXP", InitEXP},
		{"Checkin", InitCheckin},
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
		// Decoration 必须排在 Goods 之后：首次运行会为默认装扮生成关联的商品记录
		{"Decoration", InitDecoration},
		{"IntegralCard", InitIntegralCard},
		{"RewardCard", InitRewardCard},
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

	// 尚未完成安装时跳过数据库初始化，避免空 DSN 直接 panic（连安装向导都打不开）。
	// 判定口径见 facade.Installed：安装锁已解除且数据库配置已生成才算装好。
	if !facade.Installed() {
		return
	}

	facade.WatchDB(true)
	if cast.ToBool(facade.NewToml(facade.TomlDb).Get("mysql.migrate")) {
		go InitTable()
	}
}

// DomainTemp1 - 域名模板替换（查询时）
//
// 存储相关模板只有两种：{{cos}}（腾讯云 COS）与 {{localhost}}（当前站点域名）。
func DomainTemp1() (replace map[string]any) {
	toml := facade.NewToml(facade.TomlStorage)
	replace = make(map[string]any)

	// COS：配置了自定义域名（CDN）就用它，否则用默认域名 https://<bucket>-<appid>.cos.<region>.myqcloud.com
	domain := cast.ToString(toml.Get("cos.domain"))
	if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
		replace["{{cos}}"] = domain
	} else {
		replace["{{cos}}"] = facade.COSDomain()
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

	if !utils.Is.Empty(toml.Get("cos.domain")) {
		replace[cast.ToString(toml.Get("cos.domain"))] = "{{cos}}"
	}

	localhost := facade.Var.Get("domain")
	if !utils.Is.Empty(localhost) {
		replace[cast.ToString(localhost)] = "{{localhost}}"
	}
	if !utils.Is.Empty(facade.Cache.Get("domain")) {
		replace[cast.ToString(facade.Cache.Get("domain"))] = "{{localhost}}"
	}

	// COS 默认域名同样走 facade.COSDomain()（桶名归一化 + 地域回退）
	replace[facade.COSDomain()] = "{{cos}}"

	return replace
}

// ReplaceDomainColumns - 替换 Column() 查询结果里的域名模板（查询时口径）
//
// 背景：Column() 内部走 Scan 到 []map[string]any，不会实例化模型结构体，
// 因此模型上的 AfterFind 钩子不会被触发，而头像 / 附件地址里的 {{cos}}、
// {{localhost}} 等存储模板正是在 AfterFind 里还原成真实域名的。
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
