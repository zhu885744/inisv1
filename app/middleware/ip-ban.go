package middleware

import (
	"sync"
	"time"

	"github.com/spf13/cast"

	"inis/app/facade"
	"inis/app/model"
)

/**
 * IP 封禁名单的进程内快照
 *
 * 为什么需要：IpBlack 中间件挂在每个 /api 请求上，原实现每个请求都要
 * `SELECT * FROM ip_black WHERE ip = ?` —— 一次页面加载几十个请求就是几十次查询，
 * 而这张表通常只有几十到几百行、几分钟才变一次。
 *
 * 这里按 30 秒 TTL 在进程内缓存「仍在封禁期内」的记录：
 *   - 每次请求只剩一次 map 查找，零 DB 开销；
 *   - 自动封禁（QpsWarn）写库后调用 ResetIpBanCache()，立即生效；
 *   - 后台手动封禁最多延迟 30 秒，对风控没有实际影响。
 */

const (
	// ipBanCacheTTL 封禁名单快照有效期
	ipBanCacheTTL = 30 * time.Second
)

// IpBan 一条封禁记录的精简形态
type IpBan struct {
	Permanent bool  // 是否永久封禁
	ExpireAt  int64 // 解封时间（秒级时间戳，Permanent=false 时有效）
}

// Banned 指定时刻是否仍在封禁期内
func (this IpBan) Banned(now int64) bool {
	if this.Permanent {
		return true
	}
	return this.ExpireAt > 0 && now < this.ExpireAt
}

var (
	ipBanMu     sync.RWMutex
	ipBanIndex  map[string]IpBan
	ipBanLoaded time.Time
)

// IpBanIndex 取封禁索引（进程内快照 → 数据库）
func IpBanIndex() map[string]IpBan {
	ipBanMu.RLock()
	if ipBanIndex != nil && time.Since(ipBanLoaded) < ipBanCacheTTL {
		index := ipBanIndex
		ipBanMu.RUnlock()
		return index
	}
	ipBanMu.RUnlock()

	ipBanMu.Lock()
	defer ipBanMu.Unlock()

	// 双检：等锁期间可能已有其它请求刷新过
	if ipBanIndex != nil && time.Since(ipBanLoaded) < ipBanCacheTTL {
		return ipBanIndex
	}

	items, err := facade.DB.Model(&[]model.IpBlack{}).Select()
	if err != nil {
		// 查询失败：沿用旧快照。旧快照为空时返回空表（等于放行），
		// 绝不能因为数据库抖动把全站用户判成被封禁。
		if ipBanIndex != nil {
			return ipBanIndex
		}
		return map[string]IpBan{}
	}

	now := time.Now().Unix()
	index := make(map[string]IpBan, len(items))

	for _, item := range items {
		ip := cast.ToString(item["ip"])
		if ip == "" {
			continue
		}

		info := IpBan{
			Permanent: cast.ToBool(item["is_permanent"]),
			ExpireAt:  cast.ToInt64(item["expire_time"]),
		}

		// 已过期的记录不进索引（自动解封协程还没跑到也不影响判定）
		if info.Banned(now) {
			index[ip] = info
		}
	}

	ipBanIndex = index
	ipBanLoaded = time.Now()

	return index
}

// ResetIpBanCache 清空封禁快照（封禁 / 解封后调用，保证立即生效）
func ResetIpBanCache() {
	ipBanMu.Lock()
	ipBanIndex = nil
	ipBanLoaded = time.Time{}
	ipBanMu.Unlock()
}
