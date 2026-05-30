package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"golang.org/x/time/rate"
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"sync"
	"time"
)

// QPS常量
const (
	qpsPointCacheName   = "config[SYSTEM_QPS]"
	qpsBlockCacheName   = "config[SYSTEM_QPS_BLOCK]"
	defaultPointSpeed   = 10
	defaultGlobalSpeed  = 50
	qpsWarnInterval     = 10 * time.Millisecond
)

// QoSPoint - 单接口限流器
var QoSPoint = make(map[string]*rate.Limiter)

// QoSGlobal - 全局接口限流器
var QoSGlobal = make(map[string]*rate.Limiter)

// qpsMutex - 互斥锁
var qpsMutex = &sync.Mutex{}

// QpsPoint - 单接口限流器
func QpsPoint() gin.HandlerFunc {
	go qpsDelete()
	go qpsReset()

	return func(ctx *gin.Context) {
		var config map[string]any

		cacheState := cast.ToBool(facade.CacheToml.Get("open"))

		if cacheState && facade.Cache.Has(qpsPointCacheName) {
			config = cast.ToStringMap(facade.Cache.Get(qpsPointCacheName))
		} else {
			config = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS").Find()
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

		qpsMutex.Lock()
		limit := QoSPoint[key]
		if limit == nil {
			limit = rate.NewLimiter(rate.Every(qpsWarnInterval), speed)
			QoSPoint[key] = limit
		}
		qpsMutex.Unlock()

		if !limit.Allow() {
			go QpsWarn(ctx)
			ctx.AbortWithStatusJSON(200, gin.H{"code": 429, "msg": facade.Lang(ctx, "请求过于频繁！"), "data": nil})
			return
		}

		ctx.Next()
	}
}

// QpsGlobal - 全局接口限流器
func QpsGlobal() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var config map[string]any

		cacheState := cast.ToBool(facade.CacheToml.Get("open"))

		if cacheState && facade.Cache.Has(qpsPointCacheName) {
			config = cast.ToStringMap(facade.Cache.Get(qpsPointCacheName))
		} else {
			config = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS").Find()
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
		qpsMutex.Lock()
		limit := QoSGlobal[ip]
		if limit == nil {
			limit = rate.NewLimiter(rate.Every(qpsWarnInterval), speed)
			QoSGlobal[ip] = limit
		}
		qpsMutex.Unlock()

		if !limit.Allow() {
			go QpsWarn(ctx)
			ctx.AbortWithStatusJSON(200, gin.H{"code": 429, "msg": facade.Lang(ctx, "请求过于频繁！"), "data": nil})
			return
		}

		ctx.Next()
	}
}

// qpsDelete - 监控QPSPoint和QoSGlobal的协程
func qpsDelete() {
	for {
		time.Sleep(time.Second)
		qpsMutex.Lock()
		for key, item := range QoSPoint {
			if item.Allow() {
				delete(QoSPoint, key)
			}
		}
		for key, item := range QoSGlobal {
			if item.Allow() {
				delete(QoSGlobal, key)
			}
		}
		qpsMutex.Unlock()
	}
}

// qpsReset - 重置QPSPoint和QoSGlobal的协程
func qpsReset() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		qpsMutex.Lock()
		if len(QoSPoint) == 0 {
			for key := range QoSPoint {
				delete(QoSPoint, key)
			}
		}
		if len(QoSGlobal) == 0 {
			for key := range QoSGlobal {
				delete(QoSGlobal, key)
			}
		}
		qpsMutex.Unlock()
	}
}

// QpsWarn - QPS警告
func QpsWarn(ctx *gin.Context) {
	var QpsBlock map[string]any

	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(qpsBlockCacheName) {
		QpsBlock = cast.ToStringMap(facade.Cache.Get(qpsBlockCacheName))
	} else {
		QpsBlock = facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_QPS_BLOCK").Find()
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
	unix := time.Now().Add(-cast.ToDuration(utils.Calc(config["second"]))*time.Second).Unix()
	count := facade.DB.Model(&model.QpsWarn{}).Where("ip", ctx.ClientIP()).Where("create_time", ">", unix).Count()

	if count >= cast.ToInt64(config["count"]) {
		ip := ctx.ClientIP()
		facade.DB.Model(&model.IpBlack{}).Where("ip", ip).Save(&model.IpBlack{
			Ip:     ip,
			Agent:  ctx.GetHeader("User-Agent"),
			Cause:  "触发QPS警告上限，自动拉黑！",
		})
		return
	}

	tx := facade.DB.Model(&model.QpsWarn{}).Create(&model.QpsWarn{
		Ip:     ctx.ClientIP(),
		Agent:  ctx.GetHeader("User-Agent"),
		Path:   ctx.Request.URL.Path,
		Method: strings.ToUpper(ctx.Request.Method),
	})

	if tx.Error != nil {
		facade.Log.Error(map[string]any{
			"error":     tx.Error.Error(),
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "QPS警告写入失败！")
	}
}
