package middleware

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// QPS常量
const (
	qpsPointCacheName  = "config[SYSTEM_QPS]"
	qpsBlockCacheName  = "config[SYSTEM_QPS_BLOCK]"
	defaultPointSpeed  = 10
	defaultGlobalSpeed = 50
)

// qpsOnce - 确保后台协程（IP 自动解封）只启动一次
var qpsOnce = &sync.Once{}

// qpsCacheDown 熔断截止时间（unix 秒）
//
// 限流在**每个请求**上都要访问缓存；如果 Redis 挂了，每次请求都要等一次连接超时，
// 整站会被限流中间件拖死。这里失败一次就熔断 30 秒，期间直接放行。
var qpsCacheDown atomic.Int64

// qpsWindowKey 限流计数的键：按「秒」分桶
//
// 用秒级时间戳分桶而不是「一个键 + TTL」，好处是窗口边界固定、不需要重置逻辑，
// 旧桶由缓存过期自然回收（见 qpsAllow 里的 Expire）。
func qpsWindowKey(scope, id string, now time.Time) string {
	return fmt.Sprintf("qps:%s:%d:%s", scope, now.Unix(), id)
}

// qpsAllow 判断本次请求是否放行（每 scope+id 每秒最多 speed 次）
//
// 旧实现用的是进程内 rate.Limiter：多实例部署时实际阈值被放大 N 倍，
// 进程重启后所有桶清零，等于限流可以被「重启 / 轮询实例」绕过。
// 现在改成缓存的原子自增（redis 下跨实例共享），语义是「每 scope+id 每秒 speed 次」。
//
// 缓存不可用时直接放行：限流是保护性功能，不能因为缓存故障把整站打成 429。
func qpsAllow(scope, id string, speed int, now time.Time) bool {
	if speed <= 0 || facade.Cache == nil {
		return true
	}

	// 缓存刚失败过：熔断期内不再尝试（见 qpsCacheDown）
	if qpsCacheDown.Load() > now.Unix() {
		return true
	}

	key := qpsWindowKey(scope, id, now)
	count, ok := facade.Cache.Incr(key)
	if !ok {
		qpsCacheDown.Store(now.Unix() + 30)
		return true
	}

	// 第一个请求负责给计数键设置过期时间；留 2 秒让窗口内的后续请求都能读到同一个桶
	if count == 1 {
		facade.Cache.Expire(key, 2*time.Second)
	}

	return count <= int64(speed)
}

// QpsPoint - 单接口限流器
func QpsPoint() gin.HandlerFunc {
	qpsOnce.Do(func() {
		go qpsAutoUnban()
	})

	return func(ctx *gin.Context) {
		var config map[string]any

		cacheState := cast.ToBool(facade.CacheToml.Get("open"))

		if cacheState && facade.Cache.Has(qpsPointCacheName) {
			config = cast.ToStringMap(facade.Cache.Get(qpsPointCacheName))
		} else {
			config, _ = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS").Find()
			if cacheState {
				go facade.Cache.Set(qpsPointCacheName, config)
			}
		}

		if !cast.ToBool(config["value"]) {
			ctx.Next()
			return
		}

		speed := cast.ToInt(cast.ToStringMap(config["json"])["point"])
		speed = utils.Ternary[int](utils.Is.Empty(speed), defaultPointSpeed, speed)

		ip := ctx.ClientIP()
		path := ctx.Request.URL.Path
		method := ctx.Request.Method
		key := fmt.Sprintf("ip=%s&path=%s&method=%s", ip, path, method)

		if !qpsAllow("point", key, speed, time.Now()) {
			go QpsWarn(ctx)
			ctx.AbortWithStatusJSON(200, gin.H{"code": 429, "msg": facade.Lang(ctx, "请求过于频繁！"), "data": nil})
			return
		}

		ctx.Next()
	}
}

func QpsGlobal() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var config map[string]any

		cacheState := cast.ToBool(facade.CacheToml.Get("open"))

		if cacheState && facade.Cache.Has(qpsPointCacheName) {
			config = cast.ToStringMap(facade.Cache.Get(qpsPointCacheName))
		} else {
			config, _ = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS").Find()
			if cacheState {
				go facade.Cache.Set(qpsPointCacheName, config)
			}
		}

		if !cast.ToBool(config["value"]) {
			ctx.Next()
			return
		}

		speed := cast.ToInt(cast.ToStringMap(config["json"])["global"])
		speed = utils.Ternary[int](utils.Is.Empty(speed), defaultGlobalSpeed, speed)

		ip := ctx.ClientIP()
		if !qpsAllow("global", ip, speed, time.Now()) {
			go QpsWarn(ctx)
			ctx.AbortWithStatusJSON(200, gin.H{"code": 429, "msg": facade.Lang(ctx, "请求过于频繁！"), "data": nil})
			return
		}

		ctx.Next()
	}
}

// qpsAutoUnban - 自动解封协程
func qpsAutoUnban() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		if facade.DB == nil {
			continue
		}

		// 多实例只需一个实例扫描：删除本身是幂等的，但没必要 N 个实例同时扫全表
		if facade.Cache != nil && !facade.Cache.SetNX("qps:auto_unban", time.Now().Unix(), 4*time.Minute) {
			continue
		}

		var expiredIPs []model.IpBlack
		now := time.Now().Unix()
		facade.DB.Model(&model.IpBlack{}).
			Where("is_permanent = ?", false).
			Where("expire_time > 0").
			Where("expire_time < ?", now).
			Scan(&expiredIPs)

		for _, ipBlack := range expiredIPs {
			// 按主键删除：Delete 的参数必须是 id 或 id 切片。
			// 之前传的是 &model.IpBlack{}，会拼成 `id = <struct>` 这种非法条件导致删除失败，
			// 行删不掉 → 每 5 分钟又会被扫出来「重复解封」一次。
			if _, err := facade.DB.Model(&model.IpBlack{}).Delete([]int{ipBlack.Id}); err != nil {
				facade.Log.Error(map[string]any{
					"error": err.Error(),
					"id":    ipBlack.Id,
					"ip":    ipBlack.Ip,
				}, "IP自动解封失败")
				continue
			}
			facade.Log.Info(map[string]any{
				"ip":    ipBlack.Ip,
				"level": ipBlack.Level,
			}, "IP自动解封")
		}

		cacheState := cast.ToBool(facade.CacheToml.Get("open"))
		if cacheState {
			facade.Cache.Del("[GET][ip-black][column]")
		}
	}
}

// QpsWarn - QPS警告（支持分级封禁）
func QpsWarn(ctx *gin.Context) {
	ip := ctx.ClientIP()

	// 白名单IP不受限制
	if IsIpWhitelisted(ip) {
		return
	}

	var QpsBlock map[string]any

	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(qpsBlockCacheName) {
		QpsBlock = cast.ToStringMap(facade.Cache.Get(qpsBlockCacheName))
	} else {
		QpsBlockData, _ := facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS_BLOCK").Find()
		QpsBlock = QpsBlockData
		if cacheState {
			go facade.Cache.Set(qpsBlockCacheName, QpsBlock)
		}
	}

	if utils.Is.Empty(QpsBlock) {
		return
	}

	if !cast.ToBool(QpsBlock["value"]) {
		return
	}

	config := cast.ToStringMap(QpsBlock["json"])
	unix := time.Now().Add(-cast.ToDuration(utils.Calc(config["second"])) * time.Second).Unix()
	count, _ := facade.DB.Model(&model.QpsWarn{}).Where("ip", ip).Where("create_time", ">", unix).Count()

	// 达到封禁阈值
	if count >= cast.ToInt64(config["count"]) {
		// 查询是否已在黑名单中
		var existingBan model.IpBlack
		facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Scan(&existingBan)

		var newLevel int
		var cause string

		if existingBan.Id == 0 {
			// 新封禁，从一级开始
			newLevel = model.BanLevel1
			cause = "触发QPS警告上限，自动拉黑！"
		} else {
			// 已存在，升级封禁等级
			newLevel = model.GetNextBanLevel(existingBan.Level)
			cause = fmt.Sprintf("再次触发QPS警告上限，封禁等级升级为%d级！", newLevel)
		}

		// 创建或更新封禁记录
		ipBlack := &model.IpBlack{
			Ip:             ip,
			Agent:          ctx.GetHeader("User-Agent"),
			Cause:          cause,
			ViolationCount: existingBan.ViolationCount + 1,
		}
		ipBlack.CalculateExpireTime(newLevel)

		if existingBan.Id > 0 {
			// 更新现有记录
			facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Update(map[string]any{
				"level":           ipBlack.Level,
				"duration":        ipBlack.Duration,
				"expire_time":     ipBlack.ExpireTime,
				"is_permanent":    ipBlack.IsPermanent,
				"violation_count": ipBlack.ViolationCount,
				"cause":           ipBlack.Cause,
				"update_time":     time.Now().Unix(),
			})
		} else {
			// 创建新记录
			facade.DB.Model(&model.IpBlack{}).Create(ipBlack)
		}

		// 清除缓存
		if cacheState {
			facade.Cache.Del("[GET][ip-black][column]")
		}

		// 发送封禁通知
		go sendBanNotification(ctx, ip, newLevel, ipBlack.Duration, ipBlack.IsPermanent)

		return
	}

	_, err := facade.DB.Model(&model.QpsWarn{}).Create(&model.QpsWarn{
		Ip:     ip,
		Agent:  ctx.GetHeader("User-Agent"),
		Path:   ctx.Request.URL.Path,
		Method: strings.ToUpper(ctx.Request.Method),
	})

	if err != nil {
		facade.Log.Error(map[string]any{
			"error":     err.Error(),
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "QPS警告写入失败！")
	}
}

// sendBanNotification - 发送封禁通知
func sendBanNotification(ctx *gin.Context, ip string, level int, duration int64, isPermanent bool) {
	// 获取通知配置
	var notifyConfig map[string]any
	notifyCacheName := "config[SYSTEM_QPS_NOTIFY]"

	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(notifyCacheName) {
		notifyConfig = cast.ToStringMap(facade.Cache.Get(notifyCacheName))
	} else {
		notifyConfig, _ = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS_NOTIFY").Find()
		if cacheState {
			go facade.Cache.Set(notifyCacheName, notifyConfig)
		}
	}

	if utils.Is.Empty(notifyConfig) || !cast.ToBool(notifyConfig["value"]) {
		return
	}

	config := cast.ToStringMap(notifyConfig["json"])

	// 构建通知内容
	durationStr := "永久"
	if !isPermanent {
		if duration >= 24*7 {
			durationStr = fmt.Sprintf("%d天", duration/24)
		} else if duration >= 24 {
			durationStr = fmt.Sprintf("%d小时", duration)
		} else {
			durationStr = fmt.Sprintf("%d小时", duration)
		}
	}

	message := fmt.Sprintf("IP %s 已被封禁\n封禁等级: %d级\n封禁时长: %s\n原因: 触发QPS警告上限", ip, level, durationStr)

	// 发送邮件通知
	if !utils.Is.Empty(config["email"]) {
		// TODO: 实现邮件发送
		facade.Log.Info(map[string]any{
			"email":   config["email"],
			"message": message,
		}, "QPS封禁邮件通知")
	}

	// 发送Webhook通知
	if !utils.Is.Empty(config["webhook"]) {
		// TODO: 实现Webhook发送
		facade.Log.Info(map[string]any{
			"webhook": config["webhook"],
			"message": message,
		}, "QPS封禁Webhook通知")
	}
}
