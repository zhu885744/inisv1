package facade

/**
 * 登录会话（Session）—— 服务端会话，令牌本体不携带任何声明
 *
 * 项目已彻底移除 JWT，登录态完全由这里的服务端会话承担：
 *   - 令牌 = 32 字节随机数的 base64url（sid），不含 uid / 密码哈希，泄露也解析不出任何信息；
 *   - 会话数据存缓存（生产必须是 Redis）：session:{sid} → {uid, scope, hash, ver, ip, agent, created_at, last_seen, expire_at}；
 *   - 会话分作用域（scope）：user = 前台登录态，admin = 后台管理会话。
 *     两者 cookie 不同、有效期不同、互不通用，后台会话必须重新输密码才能签发；
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
	// DefaultAdminSessionExpire 默认后台管理会话有效期：2 小时
	// （后台会话可动配置与用户数据，有效期刻意远短于前台，过期需重新输密码）
	DefaultAdminSessionExpire = "2 * 60 * 60"
	// DefaultAdminTokenName 后台管理会话的默认 cookie 名（与前台 token 分开）
	DefaultAdminTokenName = "INIS_ADMIN_TOKEN"
	// sessionTouchInterval 滑动续期的最小间隔（秒）
	sessionTouchInterval = int64(3600)
	// sessionVerTTL 版本号保留时长：足够长，避免版本号过期后「旧会话复活」
	sessionVerTTL = 10 * 365 * 24 * time.Hour
)

// 会话作用域：前台会话与后台管理会话完全独立
//
// 为什么要分：前台登录态（INIS_LOGIN_TOKEN）是长期挂在浏览器上的，
// 一旦被 XSS / 共享电脑 / 代理日志拿到，就等于拿到了后台入口。
// 后台另建一条会话（独立 cookie + scope=admin），
// 只有「有后台页面权限」的账号重新输入密码才能签发，并且可以单独撤销 / 单独过期。
const (
	// SessionScopeUser 前台会话
	SessionScopeUser = "user"
	// SessionScopeAdmin 后台管理会话
	SessionScopeAdmin = "admin"
)

// ErrSessionInvalid 会话不存在 / 已过期 / 已被踢下线
var ErrSessionInvalid = errors.New("会话已失效，请重新登录")

// ErrSessionStore 会话存储不可用（缓存写入失败）
var ErrSessionStore = errors.New("会话存储不可用")

// ErrAdminSessionRequired 后台管理会话缺失 / 已失效，需要重新验证密码
//
// 与 ErrSessionInvalid 分开：前台会话过期只需重新登录，
// 后台会话过期要再输一次密码（见 controller/comm.go 的 adminLogin）。
var ErrAdminSessionRequired = errors.New("进入后台需要重新验证密码")

// Session 一条服务端会话
type Session struct {
	Uid       int    `json:"uid"`        // 用户 id
	Scope     string `json:"scope"`      // 会话作用域：user 前台 / admin 后台
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
		"scope":      this.Scope,
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
	Scope string // 会话作用域：user 前台 / admin 后台
	Hash  string // 登录时的密码哈希（调用方应据此校验密码是否已变更）
	Valid int64  // 剩余有效期（秒）
	Sid   string // 会话 id
}

var sessionWarnOnce sync.Once

// normalizeScope 归一化会话作用域
//
// 空值按前台处理：升级前已落缓存的会话没有 scope 字段，
// 不归一化会全部因为「作用域不匹配」被判失效，导致所有在线用户被踢下线。
func normalizeScope(scope string) string {
	if strings.TrimSpace(scope) == SessionScopeAdmin {
		return SessionScopeAdmin
	}
	return SessionScopeUser
}

// AdminTokenName 后台管理会话的 cookie 名
func AdminTokenName() string {
	name := strings.TrimSpace(cast.ToString(AppToml.Get("app.admin_token_name", "")))
	if utils.Is.Empty(name) {
		name = DefaultAdminTokenName
	}
	return name
}

// SessionExpireFor 指定作用域的会话有效期（秒）
//
// 后台读取 crypt.toml 的 session.admin_expire（默认 2 小时），未配置时回落到前台有效期。
func SessionExpireFor(scope string) int64 {
	if normalizeScope(scope) != SessionScopeAdmin {
		return SessionExpire()
	}

	raw := cast.ToString(CryptToml.Get("session.admin_expire", ""))
	if utils.Is.Empty(raw) {
		raw = DefaultAdminSessionExpire
	}

	expire := cast.ToInt64(utils.Calc(raw))
	if expire <= 0 {
		expire = cast.ToInt64(utils.Calc(DefaultAdminSessionExpire))
	}
	return expire
}

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

// SessionLogin 登录成功后创建「前台会话」，返回 (sid, 有效期秒数, error)
//
// hash 传 utils.Hash.Sum32(数据库里的密码哈希)：改密后该值变化，旧会话在下次校验时失效。
func SessionLogin(uid int, hash, ip, agent string) (string, int64, error) {
	return SessionLoginScope(SessionScopeUser, uid, hash, ip, agent)
}

// SessionLoginScope 按作用域创建会话，返回 (sid, 有效期秒数, error)
//
// scope=admin 的会话是后台专用的另一条登录态（独立 cookie + 独立有效期），
// 只能通过「有后台页面权限的账号重新输入密码」签发（见 controller/comm.go 的 adminLogin）。
func SessionLoginScope(scope string, uid int, hash, ip, agent string) (string, int64, error) {
	scope = normalizeScope(scope)

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

	expire := SessionExpireFor(scope)
	now := time.Now().Unix()
	session := Session{
		Uid:       uid,
		Scope:     scope,
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
		Scope:     normalizeScope(cast.ToString(data["scope"])),
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

	// 按会话自身的作用域续期：后台会话必须继续用「短有效期」，
	// 用前台有效期续期会让后台会话悄悄变成 15 天
	expire := SessionExpireFor(session.Scope)
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

	// 用缓存的原子自增：多实例同时踢同一个人时不会算出同一个版本号（旧实现是读改写）
	version, ok := Cache.Incr(sessionVerKey(uid))
	if !ok {
		return false
	}

	// Incr 不会带 TTL，这里补上；每次踢人顺带续期，不会因为版本号过期导致旧会话「复活」
	Cache.Expire(sessionVerKey(uid), sessionVerTTL)

	Log.Info(map[string]any{"uid": uid, "session_version": version}, "已失效该用户的全部登录会话")
	return true
}

// AuthenticateToken 校验「前台会话」凭证（只认服务端会话）
//
// 返回的身份信息只做「你是谁」，用户状态（冻结 / 封禁）与密码哈希校验由调用方完成。
func AuthenticateToken(token string) (AuthIdentity, error) {
	return AuthenticateSession(token, SessionScopeUser)
}

// AuthenticateSession 校验指定作用域的会话凭证
//
// 作用域必须严格匹配：前台会话（cookie 长期挂在浏览器上）不能当作后台会话使用，
// 后台会话也不用于前台接口 —— 两条登录态的泄露面相互独立。
// 作用域不匹配时**不销毁**该会话（它本身是合法的，只是用错了地方），
// 否则前台随手带上后台 cookie 就会把后台会话烧掉。
func AuthenticateSession(token, scope string) (AuthIdentity, error) {
	scope = normalizeScope(scope)

	token = strings.TrimSpace(token)
	if utils.Is.Empty(token) {
		return AuthIdentity{}, ErrSessionInvalid
	}

	session, ok := SessionGet(token)
	if !ok {
		return AuthIdentity{}, ErrSessionInvalid
	}

	if session.Scope != scope {
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
		Scope: session.Scope,
		Hash:  session.Hash,
		Valid: valid,
		Sid:   token,
	}, nil
}

// SessionCookieExpire Cookie 的 Max-Age 取值（秒）
func SessionCookieExpire() int {
	return SessionCookieExpireFor(SessionScopeUser)
}

// SessionCookieExpireFor 指定作用域的 Cookie Max-Age（秒）
func SessionCookieExpireFor(scope string) int {
	expire := SessionExpireFor(scope)
	if expire <= 0 {
		expire = cast.ToInt64(utils.Calc(DefaultSessionExpire))
	}
	return int(expire)
}
