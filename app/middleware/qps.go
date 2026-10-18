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
	defaultPointSpeed  = 120
	defaultGlobalSpeed = 600

	// qpsWarnDedupWindow - 同一 IP 的超限记录去重窗口
	//
	// 一次页面加载可能并发几十个请求，全部被拒时不应该记几十条超限：
	// 旧版（进程内 rate.Limiter，100/s 补充）几乎不会触发，掩盖了这个问题；
	// 改成按秒严格计数后，一次刷新的并发就够把 count=3 的阈值击穿。
	qpsWarnDedupWindow = 10 * time.Second

	// qpsWarnMinCount - 封禁阈值的兜底下限
	//
	// 老库里的默认值是「1 小时 3 次」，那是配合旧限流实现的；手机 App 上线后
	// 同一出口 IP（NAT）会承载大量正常请求，封禁只应针对持续攻击流量，
	// 所以即使管理员没改过配置，也至少按 30 次算。
	qpsWarnMinCount int64 = 30

	// qpsWarnDefaultWindow - 统计窗口解析失败时的兜底值
	qpsWarnDefaultWindow = time.Hour

	// qpsWarnCountKey - 窗口内超限次数（按 IP，带 TTL，封禁成功后清零）
	qpsWarnCountKey = "qps:warn:count:%s"
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

// qpsWarnDedup - 同一 IP 在去重窗口内只处理一次超限
//
// 返回 true 表示本次需要记录/判定，false 表示窗口内已经处理过，直接跳过。
// 没有缓存时退化为「每次都处理」，与改造前行为一致（不会更差）。
func qpsWarnDedup(ip string, now time.Time) bool {
	if facade.Cache == nil {
		return true
	}

	bucket := now.Unix() / int64(qpsWarnDedupWindow/time.Second)
	key := fmt.Sprintf("qps:warn:dedup:%s:%d", ip, bucket)

	// TTL 略大于窗口：保证同一桶在整个窗口内都能命中
	return facade.Cache.SetNX(key, 1, qpsWarnDedupWindow+5*time.Second)
}

// qpsWarnCount - 统计窗口内该 IP 的超限次数（去重后）
//
// 计数放在缓存里而不是每次都 COUNT qps_warn：
//   - 历史记录不会把新窗口的计数顶高（旧实现封禁后记录仍在，解封即二次封禁、等级一路升级）；
//   - 高频超限时避免每次都对告警表做范围统计。
//
// 缓存不可用时回落到数据库统计，保证单实例 / 缓存故障时功能不失效。
func qpsWarnCount(ip string, window time.Duration) int64 {
	if facade.Cache != nil {
		key := fmt.Sprintf(qpsWarnCountKey, ip)
		count, ok := facade.Cache.Incr(key)
		if ok {
			if count == 1 && !facade.Cache.Expire(key, window) {
				// 过期时间没设上：这个键会永久累积，之后所有超限都会直接命中封禁阈值。
				// 宁可丢掉本次计数，也不能留下一个只增不减的键。
				facade.Cache.Del(key)
			}
			return count
		}
	}

	since := time.Now().Add(-window).Unix()
	count, _ := facade.DB.Model(&model.QpsWarn{}).Where("ip", ip).Where("create_time", ">", since).Count()

	return count
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

	// 去重：同一 IP 在窗口内只判定一次，避免并发风暴一次写出几十条告警
	if !qpsWarnDedup(ip, time.Now()) {
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

	window := cast.ToDuration(utils.Calc(config["second"])) * time.Second
	if window <= 0 {
		window = qpsWarnDefaultWindow
	}

	// 阈值兜底：老库默认「1 小时 3 次」是配合旧版 rate.Limiter 的，
	// 在按秒严格计数后一次正常访问就能击穿，这里至少按 10 次算。
	limit := cast.ToInt64(config["count"])
	if limit < qpsWarnMinCount {
		limit = qpsWarnMinCount
	}

	count := qpsWarnCount(ip, window)

	// 达到封禁阈值
	if count >= limit {
		// 查询是否已在黑名单中
		var existingBan model.IpBlack
		facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Scan(&existingBan)

		// 已在封禁期内：保持原判，不再升级等级
		// （否则封禁期内的每次判定都会 GetNextBanLevel，一路升到永久）
		if existingBan.Id > 0 && existingBan.IsBanned() {
			return
		}

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

		// 清零窗口计数：本次封禁已生效，解封后应重新累计，
		// 否则上一个窗口的计数会让下一次超限立刻二次封禁并升级等级。
		if facade.Cache != nil {
			facade.Cache.Del(fmt.Sprintf(qpsWarnCountKey, ip))
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
//
// 走统一邮件通知（SYSTEM_MAIL_NOTIFY）的 security.qps_ban 场景：
// 场景开关、管理员收件邮箱、发送队列与失败重试都由邮件模块处理，
// 这里只负责组织正文。原来的 SYSTEM_QPS_NOTIFY（邮件/Webhook）从未实现，
// 只有两条 TODO 日志，已下线。
func sendBanNotification(ctx *gin.Context, ip string, level int, duration int64, isPermanent bool) {
	durationStr := "永久"
	if !isPermanent {
		if duration >= 24*7 {
			durationStr = fmt.Sprintf("%d天", duration/24)
		} else {
			durationStr = fmt.Sprintf("%d小时", duration)
		}
	}

	lines := []string{
		"IP：" + ip,
		fmt.Sprintf("封禁等级：%d 级", level),
		"封禁时长：" + durationStr,
		"原因：触发 QPS 警告上限",
		"时间：" + time.Now().Format("2006-01-02 15:04:05"),
	}

	if agent := strings.TrimSpace(ctx.GetHeader("User-Agent")); agent != "" {
		lines = append(lines, "User-Agent："+agent)
	}

	if path := ctx.Request.URL.Path; path != "" {
		lines = append(lines, "触发路径："+path)
	}

	model.MailNotifyAdmin("security.qps_ban", "IP 已被自动封禁", lines...)
}
