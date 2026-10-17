package facade

/**
 * 登录会话（Session）—— 服务端会话，令牌本体不携带任何声明
 *
 * 项目已彻底移除 JWT，登录态完全由这里的服务端会话承担：
 *   - 令牌 = 32 字节随机数的 base64url（sid），不含 uid / 密码哈希，泄露也解析不出任何信息；
 *   - 会话数据存缓存（生产必须是 Redis）：session:{sid} → {uid, hash, ver, ip, agent, created_at, last_seen, expire_at}；
 *   - 退出登录 / 改密 / 封禁 / 管理员强制下线 = 删除 session 或把用户会话版本号 +1，立即生效；
 *   - 滑动续期：请求间隔超过 1 小时才回写一次，避免每个请求都写 Redis。
 *
 * 为什么用「版本号」而不是列举会话：
 *   CacheInterface 没有 Set 结构（SADD），列举某个用户的全部 sid 需要额外的集合；
 *   版本号方案下「踢掉某用户全部会话」只是一次 INCR 语义（这里用读改写实现），
 *   会话在下次校验时版本不匹配即失效并惰性删除，不需要遍历。
 *
 * 部署要求：多实例必须使用 Redis 缓存（cache.toml 的 default = "redis"），
 *   文件 / 内存缓存无法跨实例共享会话；检测到非 Redis 时首次登录会打 warn 日志。
 */

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	// SessionKeyPrefix 会话键前缀（redis 里实际还会再叠加 cache.prefix）
	SessionKeyPrefix = "session:"
	// SessionVerPrefix 用户会话版本号键前缀
	SessionVerPrefix = "session:ver:"
	// DefaultSessionExpire 默认会话有效期：15 天（与旧 JWT 保持一致）
	DefaultSessionExpire = "15 * 24 * 60 * 60"
	// sessionTouchInterval 滑动续期的最小间隔（秒）
	sessionTouchInterval = int64(3600)
	// sessionVerTTL 版本号保留时长：足够长，避免版本号过期后「旧会话复活」
	sessionVerTTL = 10 * 365 * 24 * time.Hour
)

// ErrSessionInvalid 会话不存在 / 已过期 / 已被踢下线
var ErrSessionInvalid = errors.New("会话已失效，请重新登录")

// ErrSessionStore 会话存储不可用（缓存写入失败）
var ErrSessionStore = errors.New("会话存储不可用")

// Session 一条服务端会话
type Session struct {
	Uid       int    `json:"uid"`        // 用户 id
	Hash      string `json:"hash"`       // 登录时的密码哈希（改密后旧会话立即失效）
	Ver       int    `json:"ver"`        // 创建时的用户会话版本号（强制下线时 +1）
	Ip        string `json:"ip"`         // 登录 IP（仅记录，不强制校验）
	Agent     string `json:"agent"`      // 登录设备（UA，仅记录）
	CreatedAt int64  `json:"created_at"` // 创建时间
	LastSeen  int64  `json:"last_seen"`  // 最近一次请求时间（滑动续期用）
	ExpireAt  int64  `json:"expire_at"`  // 过期时间（秒级时间戳）
}

// toMap 会话落缓存的统一结构
//
// 显式转 map 而不是直接缓存结构体：缓存驱动最终走 JSON 编解码，
// 字段名可能按 Go 字段名（Uid）而不是 json tag（uid）落盘，
// 读取时按固定 key 取会拿不到值，表现为「登录成功但立刻失效」。
func (this Session) toMap() map[string]any {
	return map[string]any{
		"uid":        this.Uid,
		"hash":       this.Hash,
		"ver":        this.Ver,
		"ip":         this.Ip,
		"agent":      this.Agent,
		"created_at": this.CreatedAt,
		"last_seen":  this.LastSeen,
		"expire_at":  this.ExpireAt,
	}
}

// AuthIdentity 会话校验结果
type AuthIdentity struct {
	Uid   int    // 用户 id
	Hash  string // 登录时的密码哈希（调用方应据此校验密码是否已变更）
	Valid int64  // 剩余有效期（秒）
	Sid   string // 会话 id
}

var sessionWarnOnce sync.Once

// SessionExpire 会话有效期（秒）
//
// 读取 crypt.toml 的 session.expire；老站点的配置文件里只有 jwt.expire（JWT 时代遗留），
// 兼容读取一次，等管理员在后台保存过一次配置后会自动改写成 [session]。
func SessionExpire() int64 {
	raw := cast.ToString(CryptToml.Get("session.expire", ""))
	if utils.Is.Empty(raw) {
		raw = cast.ToString(CryptToml.Get("jwt.expire", DefaultSessionExpire))
	}

	expire := cast.ToInt64(utils.Calc(raw))
	if expire <= 0 {
		expire = cast.ToInt64(utils.Calc(DefaultSessionExpire))
	}
	return expire
}

func sessionKey(sid string) string {
	return SessionKeyPrefix + sid
}

func sessionVerKey(uid int) string {
	return SessionVerPrefix + cast.ToString(uid)
}

// newSessionID 生成会话 id：32 字节 crypto/rand → base64url（43 个字符，无填充）
func newSessionID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// clipAgent 截断设备信息，避免超长 UA 把会话撑大（按字符截，不产生乱码）
func clipAgent(agent string) string {
	agent = strings.TrimSpace(agent)
	runes := []rune(agent)
	if len(runes) > 255 {
		return string(runes[:255])
	}
	return agent
}

// SessionVersion 读取用户当前会话版本号（不存在时为 0）
func SessionVersion(uid int) int {
	if uid <= 0 || Cache == nil {
		return 0
	}
	return cast.ToInt(Cache.Get(sessionVerKey(uid)))
}

// SessionLogin 登录成功后创建会话，返回 (sid, 有效期秒数, error)
//
// hash 传 utils.Hash.Sum32(数据库里的密码哈希)：改密后该值变化，旧会话在下次校验时失效。
func SessionLogin(uid int, hash, ip, agent string) (string, int64, error) {
	if uid <= 0 {
		return "", 0, ErrSessionInvalid
	}
	// 缓存未初始化（配置异常 / 驱动不可用）时不能签发：否则登录成功但下一次请求立刻失效
	if Cache == nil {
		Log.Error(map[string]any{"uid": uid}, "会话存储不可用：缓存未初始化，请检查 cache.toml")
		return "", 0, ErrSessionStore
	}

	sessionWarnOnce.Do(func() {
		if cast.ToString(CacheToml.Get("default")) != CacheModeRedis {
			Log.Warn(map[string]any{
				"cache_driver": cast.ToString(CacheToml.Get("default")),
			}, "当前缓存驱动不是 redis：登录会话无法跨实例共享，多实例部署请把 cache.toml 的 default 改为 redis")
		}
	})

	sid, err := newSessionID()
	if err != nil {
		Log.Error(map[string]any{"error": err.Error(), "uid": uid}, "生成会话 id 失败")
		return "", 0, ErrSessionStore
	}

	expire := SessionExpire()
	now := time.Now().Unix()
	session := Session{
		Uid:       uid,
		Hash:      hash,
		Ver:       SessionVersion(uid),
		Ip:        ip,
		Agent:     clipAgent(agent),
		CreatedAt: now,
		LastSeen:  now,
		ExpireAt:  now + expire,
	}

	if !Cache.Set(sessionKey(sid), session.toMap(), time.Duration(expire)*time.Second) {
		Log.Error(map[string]any{"uid": uid}, "会话写入失败：请检查缓存（redis）是否可用")
		return "", 0, ErrSessionStore
	}

	return sid, expire, nil
}

// SessionGet 读取会话（不存在 / 结构异常时 ok=false，不区分原因）
func SessionGet(sid string) (Session, bool) {
	sid = strings.TrimSpace(sid)
	if utils.Is.Empty(sid) || Cache == nil {
		return Session{}, false
	}

	item := Cache.Get(sessionKey(sid))
	if utils.Is.Empty(item) {
		return Session{}, false
	}

	data := cast.ToStringMap(item)
	session := Session{
		Uid:       cast.ToInt(data["uid"]),
		Hash:      cast.ToString(data["hash"]),
		Ver:       cast.ToInt(data["ver"]),
		Ip:        cast.ToString(data["ip"]),
		Agent:     cast.ToString(data["agent"]),
		CreatedAt: cast.ToInt64(data["created_at"]),
		LastSeen:  cast.ToInt64(data["last_seen"]),
		ExpireAt:  cast.ToInt64(data["expire_at"]),
	}
	if session.Uid <= 0 {
		return Session{}, false
	}

	return session, true
}

// SessionTouch 滑动续期：距上次活跃超过 1 小时才回写，降低 Redis 写压力
func SessionTouch(sid string, session Session) {
	now := time.Now().Unix()
	if session.LastSeen > 0 && now-session.LastSeen < sessionTouchInterval {
		return
	}

	expire := SessionExpire()
	session.LastSeen = now
	session.ExpireAt = now + expire

	Cache.Set(sessionKey(sid), session.toMap(), time.Duration(expire)*time.Second)
}

// SessionDestroy 退出登录 / 清理指定会话
func SessionDestroy(sid string) bool {
	sid = strings.TrimSpace(sid)
	if utils.Is.Empty(sid) || Cache == nil {
		return false
	}
	return Cache.Del(sessionKey(sid))
}

// SessionDestroyUser 踢掉某用户的全部会话（改密、封禁、管理员强制下线）
//
// 实现为版本号 +1：所有旧会话在下次校验时版本不匹配而失效，
// 无需遍历该用户的所有 sid（CacheInterface 没有集合操作）。
func SessionDestroyUser(uid int) bool {
	if uid <= 0 || Cache == nil {
		return false
	}

	version := SessionVersion(uid) + 1
	ok := Cache.Set(sessionVerKey(uid), version, sessionVerTTL)
	Log.Info(map[string]any{"uid": uid, "session_version": version}, "已失效该用户的全部登录会话")
	return ok
}

// AuthenticateToken 校验登录凭证（只认服务端会话）
//
// 返回的身份信息只做「你是谁」，用户状态（冻结 / 封禁）与密码哈希校验由调用方完成。
func AuthenticateToken(token string) (AuthIdentity, error) {
	token = strings.TrimSpace(token)
	if utils.Is.Empty(token) {
		return AuthIdentity{}, ErrSessionInvalid
	}

	session, ok := SessionGet(token)
	if !ok {
		return AuthIdentity{}, ErrSessionInvalid
	}

	now := time.Now().Unix()

	// 过期（Redis 自动过期之外的兜底，兼容 file / ram 驱动）
	if session.ExpireAt > 0 && now >= session.ExpireAt {
		SessionDestroy(token)
		return AuthIdentity{}, ErrSessionInvalid
	}

	// 版本号不一致 = 已被踢下线 / 改密 / 封禁
	if session.Ver != SessionVersion(session.Uid) {
		SessionDestroy(token)
		return AuthIdentity{}, ErrSessionInvalid
	}

	SessionTouch(token, session)

	valid := session.ExpireAt - now
	if valid <= 0 {
		valid = 1
	}

	return AuthIdentity{
		Uid:   session.Uid,
		Hash:  session.Hash,
		Valid: valid,
		Sid:   token,
	}, nil
}

// SessionCookieExpire Cookie 的 Max-Age 取值（秒）
func SessionCookieExpire() int {
	expire := SessionExpire()
	if expire <= 0 {
		expire = cast.ToInt64(utils.Calc(DefaultSessionExpire))
	}
	return int(expire)
}
