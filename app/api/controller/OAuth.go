package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"

	"inis/app/facade"
	"inis/app/model"
)

// ============================== 第三方登录（QQ / GitHub / Gitee） ==============================
//
// 分流规则（同一个入口，靠「是否已登录 + 是否已绑定」判断）：
//
//	1. 登录：第三方账号已绑定过本站账号 → 直接发 token 登录；
//	2. 绑定：第三方账号未绑定 + 已登录 → 直接绑到当前账号；
//	3. 二选一：第三方账号未绑定 + 未登录 → 发一张短票据，回调页让用户自己选：
//	     - 创建新账号：POST /api/oauth/register（票据，受注册开关与「允许第三方建号」约束）
//	     - 绑定已有账号：POST /api/oauth/bind-account（票据 + 账号密码，口径与密码登录一致）
//
// 前端拿到的结果分三种：
//
//	{user, token, valid_time, bind:true, register:true}  → 新建账号并登录成功
//	{user, token, valid_time, bind:true}                 → 登录成功 / 绑定成功
//	{need_bind:true, need_choice:true, ticket, ...}      → 未登录且未绑定，需用户在回调页选择
//
// 关于「先登录再绑定」：旧实现在这种场景会把用户赶去登录页，登录后再用票据自动绑定
// （票据消费见 /api/oauth/bind 的 key 分支）。新实现在回调页直接给出选择，不必绕登录页；
// 旧路径仍然保留可用 —— 老前端、以及选择页刷新（票据已存 sessionStorage）后仍能完成绑定。
//
// 为什么不在这里做「浏览器整跳转发号」：token 落在 URL 上容易被 Referer / 日志带走，
// 而且主题的前后端常常不同域。这里统一走「前端拿 code → 后端换身份 → 后端发 token」，
// AppSecret 只留在服务端，前端只需要拼授权页地址（见 config 接口）。
//
// 关于 state：oauth 中间件无会话，服务端无法校验；state 由前端生成并比对（sessionStorage），
// 属于标准做法，服务端只透传不回存。
//
// 配置：config 表的 SYSTEM_OAUTH（见 model/oauth.go）；绑定关系：inis_user_oauth。

type OAuth struct {
	// 继承
	base
}

// 第三方接口请求的超时与代理见 model.OauthTimeout / model.OauthProxy（后台可配），
// 因为自建服务器常因出网受限连不上 github.com，硬编码超时解决不了问题。
// 另：utils.Curl 默认的 http.Client 没有超时，慢响应会把连接一直占着。

// oauthBindTicketPrefix - 「先登录再绑定」票据的缓存前缀
const oauthBindTicketPrefix = "oauth-bind-ticket:"

// oauthBindTicketExpire - 票据有效期：从第三方跳回来到用户完成登录，10 分钟足够
const oauthBindTicketExpire = 10 * time.Minute

// oauthIdentity - 第三方平台返回的身份信息
type oauthIdentity struct {
	Platform string
	Openid   string
	Unionid  string
	Nickname string
	Avatar   string
}

// IGET - GET请求本体
func (this *OAuth) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"config": this.config,
		"mine":   this.mine,
		"qq":     this.qq,
		"github": this.github,
		"gitee":  this.gitee,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *OAuth) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"qq":     this.qq,
		"github": this.github,
		"gitee":  this.gitee,
		"bind":   this.bind,
		"unbind": this.unbind,
		// 未绑定时回调页的两个选项（都靠票据证明第三方身份，无需登录态）
		"register":     this.register,
		"bind-account": this.bindAccount,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - PUT请求本体
func (this *OAuth) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"bind": this.bind,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - DELETE请求本体
func (this *OAuth) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"unbind": this.unbind,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - GET请求本体
func (this *OAuth) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// ============================== 对外接口 ==============================

// config - 第三方登录配置（公开）
//
// 只回「前端拼授权地址需要的字段」：开关、AppID、回调地址、授权页、scope。
// AppSecret 绝不出网 —— 换 token 由服务端做。
func (this *OAuth) config(ctx *gin.Context) {

	result := facade.H{}
	platforms := []facade.H{}

	for _, platform := range model.OauthPlatformList() {

		setting := model.OauthPlatformConfig(platform)
		available, _ := model.OauthPlatformAvailable(platform)
		authorize := model.OauthPlatformAuthorize(platform)

		item := facade.H{
			"platform":  platform,
			"name":      model.OauthPlatformName(platform),
			"enable":    utils.Ternary(available, 1, 0),
			"app_id":    cast.ToString(setting["app_id"]),
			"redirect":  cast.ToString(setting["redirect"]),
			"authorize": authorize["url"],
			"scope":     authorize["scope"],
		}

		result[platform] = item
		platforms = append(platforms, item)
	}

	// 未绑定时是否允许自动注册：前端据此提示「将为你创建新账号」还是「请先登录再绑定」
	result["auto_register"] = utils.Ternary(model.OauthAutoRegister(), 1, 0)
	result["platforms"] = platforms

	this.json(ctx, result, facade.Lang(ctx, "查询成功！"), 200)
}

// qq - QQ登录（登录 / 注册 / 绑定，见文件头说明）
func (this *OAuth) qq(ctx *gin.Context) {
	this.handle(ctx, model.OauthPlatformQQ)
}

// github - GitHub登录
func (this *OAuth) github(ctx *gin.Context) {
	this.handle(ctx, model.OauthPlatformGithub)
}

// gitee - Gitee登录
func (this *OAuth) gitee(ctx *gin.Context) {
	this.handle(ctx, model.OauthPlatformGitee)
}

// mine - 我绑定的第三方账号（含还能绑哪些平台，供「账号安全」页渲染）
func (this *OAuth) mine(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 管理员可以指定 uid 查看任意用户的绑定（后台「用户详情」弹窗用）；普通用户只能看自己
	if target := cast.ToInt(this.params(ctx)["uid"]); target > 0 && target != uid {
		if !this.meta.permit(ctx) {
			this.json(ctx, nil, facade.Lang(ctx, "无权限查看其它用户的第三方绑定！"), 403)
			return
		}
		uid = target
	}

	list := []facade.H{}
	bound := map[string]bool{}

	for _, item := range model.UserOauthList(uid) {
		platform := cast.ToString(item["platform"])
		bound[platform] = true
		list = append(list, facade.H{
			"id":          item["id"],
			"platform":    platform,
			"name":        model.OauthPlatformName(platform),
			"nickname":    item["nickname"],
			"avatar":      item["avatar"],
			"create_time": item["create_time"],
		})
	}

	// 还能绑定的平台：已开启 + 必填项齐全 + 当前账号未绑定
	available := []facade.H{}
	for _, platform := range model.OauthPlatformList() {

		if bound[platform] {
			continue
		}

		if ok, _ := model.OauthPlatformAvailable(platform); !ok {
			continue
		}

		setting := model.OauthPlatformConfig(platform)
		authorize := model.OauthPlatformAuthorize(platform)

		available = append(available, facade.H{
			"platform":  platform,
			"name":      model.OauthPlatformName(platform),
			"app_id":    cast.ToString(setting["app_id"]),
			"redirect":  cast.ToString(setting["redirect"]),
			"authorize": authorize["url"],
			"scope":     authorize["scope"],
		})
	}

	this.json(ctx, gin.H{
		"list":      list,
		"available": available,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// bind - 绑定第三方账号（需登录）
//
// 两种入参：
//   - key：未登录时尝试社交登录拿到的票据（选择页刷新后、或老前端引导登录后的补绑）
//   - platform + code：在账号安全页点「绑定」跳转第三方后回调带回，属于「已登录直接绑」
func (this *OAuth) bind(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	// 方式一：票据（登录页跳第三方 → 未绑定 → 返回 ticket → 用户登录后带着 ticket 回来绑定）
	if key := cast.ToString(params["key"]); !utils.Is.Empty(key) {

		cacheKey := oauthBindTicketPrefix + key
		identity := oauthCachedTicket(facade.Cache.Get(cacheKey))
		if identity == nil {
			this.json(ctx, nil, facade.Lang(ctx, "绑定信息已过期，请重新发起第三方登录！"), 400)
			return
		}

		item := oauthIdentity{
			Platform: cast.ToString(identity["platform"]),
			Openid:   cast.ToString(identity["openid"]),
			Unionid:  cast.ToString(identity["unionid"]),
			Nickname: cast.ToString(identity["nickname"]),
			Avatar:   cast.ToString(identity["avatar"]),
		}

		if !this.bindTo(ctx, uid, item) {
			return
		}

		consumeTicket(cast.ToString(params["key"]))
		this.json(ctx, gin.H{"bind": true, "uid": uid, "platform": item.Platform}, facade.Lang(ctx, "绑定成功！"), 200)
		return
	}

	// 方式二：平台 + code（已登录状态下直接发起第三方授权）
	platform := strings.ToLower(cast.ToString(params["platform"]))
	code := cast.ToString(params["code"])

	if utils.Is.Empty(platform) || utils.Is.Empty(code) {
		this.json(ctx, nil, facade.Lang(ctx, "请提交 key 或 platform + code！"), 400)
		return
	}

	item, msg := this.identity(ctx, platform, code)
	if !utils.Is.Empty(msg) {
		this.json(ctx, nil, msg, 400)
		return
	}

	if !this.bindTo(ctx, uid, item) {
		return
	}

	this.json(ctx, gin.H{"bind": true, "uid": uid, "platform": platform}, facade.Lang(ctx, "绑定成功！"), 200)
}

// unbind - 解绑第三方账号（需登录）
func (this *OAuth) unbind(ctx *gin.Context) {

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	platform := strings.ToLower(cast.ToString(this.params(ctx)["platform"]))
	if !model.OauthPlatformValid(platform) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的第三方登录方式！"), 400)
		return
	}

	// 别把用户锁在门外：解绑后必须还有别的登录手段
	// （密码为空说明从未设过密码；邮箱/手机号为空说明也没法用验证码登录）
	hasPassword := !utils.Is.Empty(user.Password)
	hasContact := !utils.Is.Empty(user.Email) || !utils.Is.Empty(user.Phone)
	otherBind := model.UserOauthBoundCount(user.Id) - 1

	if !hasPassword && !hasContact && otherBind <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "解绑后你将无法登录本站，请先设置密码或绑定其它登录方式！"), 400)
		return
	}

	if err := model.UnbindUserOauth(user.Id, platform); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, gin.H{"bind": false, "platform": platform}, facade.Lang(ctx, "解绑成功！"), 200)
}

// register - 用票据创建新账号、绑定并登录（未绑定时用户在回调页选择「创建新账号」）
//
// 与「已登录用户去账号安全页绑定」不同：这里没有任何登录态，身份完全由票据证明。
// 双开关：站点注册总开关（ALLOW_REGISTER）+ 后台「允许第三方账号创建新账号」（auto_register）。
func (this *OAuth) register(ctx *gin.Context) {

	params := this.params(ctx)

	identity, msg := this.ticketIdentity(ctx, params)
	if !utils.Is.Empty(msg) {
		this.json(ctx, nil, msg, 400)
		return
	}

	// 管理员关了注册：不建号（否则等于绕开了注册开关）
	if !allowRegister() {
		this.json(ctx, nil, facade.Lang(ctx, "管理员关闭了注册功能，请先注册账号后再绑定！"), 400)
		return
	}

	if !model.OauthAutoRegister() {
		this.json(ctx, nil, facade.Lang(ctx, "管理员未开放「第三方账号创建新账号」，请改用已有账号绑定！"), 400)
		return
	}

	// 票据里的第三方账号可能已被别人绑走（用户开了两个页面 / 票据被他人拿到）
	if binding := model.FindUserOauth(identity.Platform, identity.Openid); !utils.Is.Empty(binding) && cast.ToInt(binding["delete_time"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该第三方账号已绑定其它用户！"), 400)
		return
	}

	newUid, needAudit, msg := this.registerUser(ctx, identity)
	if !utils.Is.Empty(msg) {
		this.json(ctx, nil, msg, 400)
		return
	}

	// 建号即绑定：无论是否需要人工审核都先绑上，审核通过后直接点第三方登录就能进
	if !this.bindTo(ctx, newUid, identity) {
		return
	}

	consumeTicket(cast.ToString(params["key"]))

	// 人工审核：与注册流程一致，此时不发 token
	if needAudit {
		table := model.Users{}
		item, _ := facade.DB.Model(&table).Where("id", newUid).Find()
		this.json(ctx, gin.H{
			"user":       item,
			"need_audit": true,
			"bind":       true,
			"register":   true,
		}, facade.Lang(ctx, "注册成功，请等待管理员审核通过后再登录！"), 200)
		return
	}

	this.loginResult(ctx, newUid, facade.H{"bind": true, "register": true, "platform": identity.Platform})
}

// bindAccount - 用「已有账号 + 密码」把未绑定的第三方账号绑上去并登录
//
// 未登录用户点第三方登录、而该第三方账号还没绑定时，回调页可以让用户直接输账号密码绑定，
// 不必先跳登录页再回来。账号口径与密码登录完全一致（account / email / phone，
// 先按 source=default 查、未命中再放开 source 回查一次），风控同样走 assertLoginAllowed。
func (this *OAuth) bindAccount(ctx *gin.Context) {

	params := this.params(ctx)

	identity, msg := this.ticketIdentity(ctx, params)
	if !utils.Is.Empty(msg) {
		this.json(ctx, nil, msg, 400)
		return
	}

	account := strings.TrimSpace(cast.ToString(params["account"]))
	password := cast.ToString(params["password"])
	if utils.Is.Empty(account) || utils.Is.Empty(password) {
		this.json(ctx, nil, facade.Lang(ctx, "请提交账号（或邮箱、手机号）和密码！"), 400)
		return
	}

	table := model.Users{}
	item, _ := facade.DB.Model(&table).Or([]any{
		[]any{"email", "=", account},
		[]any{"phone", "=", account},
		[]any{"account", "=", account},
	}).Where("source", "default").Find()

	// 来源未命中时回退为不限定 source 再查一次（与密码登录同一处理，
	// 否则前台注册（source=mellow）等来源的账号会「账户不存在」）
	if utils.Is.Empty(item) {
		item, _ = facade.DB.Model(&table).Or([]any{
			[]any{"email", "=", account},
			[]any{"phone", "=", account},
			[]any{"account", "=", account},
		}).Find()
	}

	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "账户不存在！"), 400)
		return
	}

	// 登录前置校验（冻结 / 待审核 / 封禁）—— 与密码登录共用同一口径
	if !assertLoginAllowed(ctx, table) {
		return
	}

	if utils.Is.Empty(table.Password) {
		this.json(ctx, nil, facade.Lang(ctx, "该帐号未设置密码，请先登录后再到「账号安全」绑定第三方账号！"), 400)
		return
	}

	if !utils.Password.Verify(table.Password, password) {
		this.json(ctx, nil, facade.Lang(ctx, "密码错误！"), 400)
		return
	}

	// 该第三方账号已绑给别人 → 拒绝，不做「偷偷切账号」
	if binding := model.FindUserOauth(identity.Platform, identity.Openid); !utils.Is.Empty(binding) && cast.ToInt(binding["delete_time"]) == 0 {
		if boundUid := cast.ToInt(binding["uid"]); boundUid != table.Id {
			this.json(ctx, nil, facade.Lang(ctx, "该第三方账号已绑定其它用户！"), 400)
			return
		}
	}

	if !this.bindTo(ctx, table.Id, identity) {
		return
	}

	consumeTicket(cast.ToString(params["key"]))

	this.loginResult(ctx, table.Id, facade.H{"bind": true, "platform": identity.Platform})
}

// ============================== 核心流程 ==============================

// handle - 登录 / 注册 / 绑定 的统一入口
func (this *OAuth) handle(ctx *gin.Context, platform string) {

	params := this.params(ctx)

	code := cast.ToString(params["code"])
	if utils.Is.Empty(code) {
		this.json(ctx, nil, facade.Lang(ctx, "缺少 code 参数！"), 400)
		return
	}

	identity, msg := this.identity(ctx, platform, code)
	if !utils.Is.Empty(msg) {
		this.json(ctx, nil, msg, 400)
		return
	}

	uid := this.meta.user(ctx).Id

	binding := model.FindUserOauth(platform, identity.Openid)
	if !utils.Is.Empty(binding) && cast.ToInt(binding["delete_time"]) == 0 {

		boundUid := cast.ToInt(binding["uid"])

		// 已登录，但这个第三方账号绑在别人身上：直接拒绝，不做「偷偷切账号」这种事
		if uid > 0 && boundUid != uid {
			this.json(ctx, nil, facade.Lang(ctx, "该第三方账号已绑定其它用户！"), 400)
			return
		}

		// 未登录 → 直接登录
		if uid == 0 {
			this.loginResult(ctx, boundUid, facade.H{"bind": true, "platform": platform})
			return
		}

		// 已登录且就是自己 → 顺手刷新第三方昵称头像
		if !this.bindTo(ctx, uid, identity) {
			return
		}
		this.json(ctx, gin.H{"bind": true, "uid": uid, "platform": platform}, facade.Lang(ctx, "该第三方账号已绑定当前账号！"), 200)
		return
	}

	// 未绑定 + 已登录 → 直接绑到当前账号
	if uid > 0 {
		if !this.bindTo(ctx, uid, identity) {
			return
		}
		this.json(ctx, gin.H{"bind": true, "uid": uid, "platform": platform}, facade.Lang(ctx, "绑定成功！"), 200)
		return
	}

	// 未绑定 + 未登录 → 发一张短票据，由用户在回调页自己选：
	//   - 创建新账号（POST oauth/register，受注册总开关与「允许第三方建号」约束）
	//   - 用已有账号密码绑定（POST oauth/bind-account）
	// 旧实现这里是「静默建号」或「提示先登录再绑定」：前者不给用户选择（已有账号的人会被建成
	// 第二个号），后者把用户挡在门外。改成二选一后，两条路都不需要先跳登录页。
	this.needChoice(ctx, identity)
}

// bindTo - 把第三方身份绑定到指定用户（失败时已写出响应，返回 false）
func (this *OAuth) bindTo(ctx *gin.Context, uid int, identity oauthIdentity) bool {

	_, err := model.BindUserOauth(uid, identity.Platform, identity.Openid, identity.Unionid, identity.Nickname, identity.Avatar)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return false
	}

	return true
}

// needChoice - 未登录且未绑定：发一张短票据，前端在回调页让用户二选一
//
//	创建新账号   → POST /api/oauth/register（仅传 key）
//	绑定已有账号 → POST /api/oauth/bind-account（key + account + password）
//
// can_register 由「站点注册总开关」与「后台允许第三方建号（auto_register）」共同决定：
// 为 0 时前端只展示「绑定已有账号」。
// need_bind 保留给老前端（它会引导去登录页，登录后用同一张票据自动绑定）。
func (this *OAuth) needChoice(ctx *gin.Context, identity oauthIdentity) {

	key := oauthNewTicket(identity)

	this.json(ctx, gin.H{
		"need_bind":    true, // 兼容旧前端：未绑定、需用户处理
		"need_choice":  true, // 新前端：展示「创建新账号 / 绑定已有账号」选择页
		"ticket":       key,
		"platform":     identity.Platform,
		"name":         model.OauthPlatformName(identity.Platform),
		"nickname":     identity.Nickname,
		"avatar":       identity.Avatar,
		"can_register": allowRegister() && model.OauthAutoRegister(),
	}, facade.Lang(ctx, "该第三方账号还未绑定本站账号，请选择创建新账号或绑定已有账号"), 200)
}

// oauthNewTicket - 生成一张绑定票据（10 分钟），缓存第三方身份供后续步骤使用
func oauthNewTicket(identity oauthIdentity) string {

	key := utils.Rand.String(32, "abcdefghijklmnopqrstuvwxyz0123456789")

	facade.Cache.Set(oauthBindTicketPrefix+key, facade.H{
		"platform": identity.Platform,
		"openid":   identity.Openid,
		"unionid":  identity.Unionid,
		"nickname": identity.Nickname,
		"avatar":   identity.Avatar,
	}, oauthBindTicketExpire)

	return key
}

// ticketIdentity - 从请求参数取票据并还原第三方身份（失败时返回提示文案）
//
// 票据是「未登录用户」在此流程里唯一可用的身份证明：它由上一步第三方回调换取，
// 10 分钟有效、成功后一次性作废（见 consumeTicket），因此 register / bindAccount
// 不需要登录态也不会被伪造。
func (this *OAuth) ticketIdentity(ctx *gin.Context, params map[string]any) (oauthIdentity, string) {

	key := strings.TrimSpace(cast.ToString(params["key"]))
	if utils.Is.Empty(key) {
		return oauthIdentity{}, facade.Lang(ctx, "缺少绑定票据 key！")
	}

	identity := oauthCachedTicket(facade.Cache.Get(oauthBindTicketPrefix + key))
	if identity == nil {
		return oauthIdentity{}, facade.Lang(ctx, "绑定信息已过期，请重新发起第三方登录！")
	}

	return oauthIdentity{
		Platform: cast.ToString(identity["platform"]),
		Openid:   cast.ToString(identity["openid"]),
		Unionid:  cast.ToString(identity["unionid"]),
		Nickname: cast.ToString(identity["nickname"]),
		Avatar:   cast.ToString(identity["avatar"]),
	}, ""
}

// consumeTicket - 票据一次性使用：建号 / 绑定成功后立即删除，避免重放
func consumeTicket(key string) {
	if utils.Is.Empty(key) {
		return
	}
	go facade.Cache.Del(oauthBindTicketPrefix + key)
}

// allowRegister - 站点是否开放注册（config 表 ALLOW_REGISTER，与注册流程同一处判断）
func allowRegister() bool {
	item, _ := facade.DB.Model(&model.Config{}).Where("key", "ALLOW_REGISTER").Find()
	return !utils.Is.Empty(item) && cast.ToBool(item["value"])
}

// loginResult - 发号登录（与密码登录、验证码登录同一套：状态校验 + 脱敏 + 登录奖励 + 登录通知）
func (this *OAuth) loginResult(ctx *gin.Context, uid int, extra facade.H) bool {

	table := model.Users{}
	item, _ := facade.DB.Model(&table).Where("id", uid).Find()
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 204)
		return false
	}

	// 登录前置校验（冻结 / 待审核 / 限制登录）—— 与密码登录共用一处口径，
	// 避免出现「密码登录被拦、第三方登录能进」的绕过
	if !assertLoginAllowed(ctx, table) {
		return false
	}

	jwt := facade.Jwt().Create(facade.H{
		"uid":  table.Id,
		"hash": utils.Hash.Sum32(table.Password),
	})

	// 分级脱敏：登录返回的是「本人」数据（保留账号/邮箱/手机号，移除密码与管理员备注）
	this.meta.privacyUserAs(item, this.meta.privacyLevel(ctx), cast.ToInt(item["id"]))

	item["login_time"] = time.Now().Unix()
	facade.DB.Model(&table).Where("id", table.Id).Update(map[string]any{
		"login_time": item["login_time"],
	})

	result := facade.H{
		"user":       item,
		"token":      jwt.Text,
		"valid_time": jwt.Valid, // 登录会话有效期（秒）
	}
	for key, value := range extra {
		result[key] = value
	}

	// 往客户端写入cookie - 存储登录token
	setToken(ctx, jwt.Text)
	// 登录增加经验 / 积分
	go loginReward(table.Id)

	// 账号登录通知（与密码登录一致）
	go func(uid int, account, nickname, ip, ua string) {
		notification := new(model.Notification)
		if _, e := notification.CreateLoginNotification(uid, account, nickname, ip, ua); e != nil {
			facade.Log.Error(map[string]any{"error": e.Error(), "uid": uid}, "发送账号登录通知失败")
		}
	}(table.Id, table.Account, table.Nickname, ctx.ClientIP(), ctx.Request.UserAgent())

	this.json(ctx, result, facade.Lang(ctx, "登录成功！"), 200)
	return true
}

// registerUser - 用第三方资料创建新账号
//
//	返回值：用户ID / 是否需要人工审核 / 错误信息（空串表示成功）
//
// 不写第三方邮箱手机号：这两个字段在本站是唯一索引，也用于登录与找回密码，
// 直接回写可能与已有账号冲突（甚至把别人的邮箱占掉）。需要邮箱请走「联系方式」页的验证码流程。
func (this *OAuth) registerUser(ctx *gin.Context, identity oauthIdentity) (int, bool, string) {

	setting := model.RegisterSettings()

	table := model.Users{}
	utils.Struct.Set(&table, "nickname", oauthNickname(identity.Nickname))
	utils.Struct.Set(&table, "avatar", oauthAvatar(identity.Avatar))
	utils.Struct.Set(&table, "source", "oauth")
	utils.Struct.Set(&table, "login_time", time.Now().Unix())

	// 账号：随机生成「小写字母 + 数字」，借 account 唯一索引防撞（与注册流程同一套办法）
	const maxRetry = 20
	var err error

	for i := 0; i < maxRetry; i++ {
		utils.Struct.Set(&table, "account", utils.Rand.String(10, "abcdefghijklmnopqrstuvwxyz0123456789"))
		if _, err = facade.DB.Model(&table).Create(&table); err == nil {
			break
		}
		// 非唯一索引冲突（Error 1062）则重试，其他错误直接返回
		if !strings.Contains(cast.ToString(err.Error()), "1062") {
			facade.Log.Error(map[string]any{"error": err.Error(), "platform": identity.Platform}, "第三方登录建号失败")
			return 0, false, facade.Lang(ctx, "创建账号失败，请稍后重试！")
		}
	}
	if err != nil {
		return 0, false, facade.Lang(ctx, "创建账号失败，请稍后重试！")
	}

	// 默认权限组（与注册流程一致）
	assignDefaultAuthGroups(table.Id)

	// 人工审核：置为待审核，管理员通过后才能登录（与注册流程一致）
	if setting.VerifyMode == model.RegisterVerifyManual {

		if _, err := facade.DB.Model(&model.Users{}).Where("id", table.Id).
			UpdateColumn("status", model.UserStatusAudit); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "uid": table.Id}, "写入待审核状态失败")
		} else {
			table.Status = model.UserStatusAudit

			go model.MailNotifyAdmin("user.pending", "有新用户等待审核", append(
				model.MailNotifyUserInfo(table.Id),
				"注册方式："+model.OauthPlatformName(identity.Platform),
				"时间："+model.MailNotifyTime(),
			)...)
		}

		return table.Id, true, ""
	}

	// 注册欢迎消息 / 欢迎邮件（按后台开关执行，内部异步）
	go model.SendWelcome(table.Id, table.Account, table.Nickname, cast.ToString(table.Email))

	return table.Id, false, ""
}

// ============================== 第三方身份获取 ==============================

// identity - 用 code 换取第三方身份（平台无关的统一入口）
//
//	返回值：身份信息 / 错误信息（空串表示成功）
func (this *OAuth) identity(ctx *gin.Context, platform, code string) (oauthIdentity, string) {

	result := oauthIdentity{Platform: platform}

	if !model.OauthPlatformValid(platform) {
		return result, facade.Lang(ctx, "不支持的第三方登录方式！")
	}

	// 平台未开启 / 参数没配齐：直接把原因说清楚，别让前端只看到「获取失败」
	if ok, msg := model.OauthPlatformAvailable(platform); !ok {
		return result, facade.Lang(ctx, msg)
	}

	switch platform {
	case model.OauthPlatformQQ:
		return this.identityByQQ(ctx, code)
	case model.OauthPlatformGithub:
		return this.identityByGithub(ctx, code)
	case model.OauthPlatformGitee:
		return this.identityByGitee(ctx, code)
	}

	return result, facade.Lang(ctx, "不支持的第三方登录方式！")
}

// identityByQQ - QQ：换 token → 换 openid/unionid → 取用户信息
func (this *OAuth) identityByQQ(ctx *gin.Context, code string) (result oauthIdentity, msg string) {

	result.Platform = model.OauthPlatformQQ

	setting := model.OauthPlatformConfig(model.OauthPlatformQQ)
	appId := cast.ToString(setting["app_id"])

	// 1) code 换 access_token
	token := oauthRequest(utils.CurlRequest{
		Method: "GET",
		Url:    "https://graph.qq.com/oauth2.0/token",
		Query: map[string]any{
			"grant_type":    "authorization_code",
			"client_id":     appId,
			"client_secret": cast.ToString(setting["app_key"]),
			"code":          code,
			"redirect_uri":  cast.ToString(setting["redirect"]),
			"fmt":           "json",
		},
	})
	if token.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 access_token", token.Error)
	}

	accessToken := cast.ToString(token.Json["access_token"])
	if utils.Is.Empty(accessToken) {
		return result, oauthErrorText(token, "获取 access_token 失败")
	}

	// 2) 换 openid / unionid（unionid=1 时多返回 unionid，用于同主体多应用打通）
	me := oauthRequest(utils.CurlRequest{
		Method: "GET",
		Url:    "https://graph.qq.com/oauth2.0/me",
		Query: map[string]any{
			"access_token": accessToken,
			"unionid":      "1",
			"fmt":          "json",
		},
	})
	if me.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 QQ 用户标识", me.Error)
	}

	result.Openid = cast.ToString(me.Json["openid"])
	result.Unionid = cast.ToString(me.Json["unionid"])
	if utils.Is.Empty(result.Openid) {
		return result, oauthErrorText(me, "获取 QQ 用户标识失败")
	}

	// 3) 取用户信息（需要 access_token + app_id + openid 三件套）
	user := oauthRequest(utils.CurlRequest{
		Method: "GET",
		Url:    "https://graph.qq.com/user/get_user_info",
		Query: map[string]any{
			"access_token":       accessToken,
			"oauth_consumer_key": appId,
			"openid":             result.Openid,
			"fmt":                "json",
		},
	})
	if user.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 QQ 用户信息", user.Error)
	}
	// ret 非 0 表示失败（如未开通权限、access_token 失效）
	if cast.ToInt(user.Json["ret"]) != 0 {
		return result, oauthErrorText(user, "获取 QQ 用户信息失败")
	}

	result.Nickname = cast.ToString(user.Json["nickname"])
	// 100x100 头像优先，其次 40x40、老字段
	result.Avatar = oauthFirstString(user.Json, "figureurl_qq_2", "figureurl_qq_1", "figureurl_2")

	return result, ""
}

// identityByGithub - GitHub：换 token → 取 /user
func (this *OAuth) identityByGithub(ctx *gin.Context, code string) (result oauthIdentity, msg string) {

	result.Platform = model.OauthPlatformGithub

	setting := model.OauthPlatformConfig(model.OauthPlatformGithub)

	// 1) code 换 access_token（form 表单提交；accept=json 才会回 JSON，否则回 form-urlencoded）
	token := oauthRequest(utils.CurlRequest{
		Method: "POST",
		Url:    "https://github.com/login/oauth/access_token",
		Headers: map[string]any{
			"Content-Type": "application/x-www-form-urlencoded",
			"accept":       "application/json",
		},
		Data: map[string]any{
			"client_id":     cast.ToString(setting["app_id"]),
			"client_secret": cast.ToString(setting["app_key"]),
			"code":          code,
		},
	})
	if token.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 access_token", token.Error)
	}

	accessToken := cast.ToString(token.Json["access_token"])
	if utils.Is.Empty(accessToken) {
		return result, oauthErrorText(token, "获取 access_token 失败")
	}

	// 2) 取用户信息（GitHub 强制要求 User-Agent）
	user := oauthRequest(utils.CurlRequest{
		Method: "GET",
		Url:    "https://api.github.com/user",
		Headers: map[string]any{
			"Authorization": "Bearer " + accessToken,
			"accept":        "application/vnd.github+json",
		},
	})
	if user.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 GitHub 用户信息", user.Error)
	}

	// GitHub 的用户唯一标识是数字 id（login 可以改名，不能作为绑定依据）
	result.Openid = cast.ToString(user.Json["id"])
	if utils.Is.Empty(result.Openid) {
		return result, oauthErrorText(user, "获取 GitHub 用户信息失败")
	}

	result.Nickname = utils.Default(cast.ToString(user.Json["name"]), cast.ToString(user.Json["login"]))
	result.Avatar = cast.ToString(user.Json["avatar_url"])

	return result, ""
}

// identityByGitee - Gitee：换 token → 取 /user
func (this *OAuth) identityByGitee(ctx *gin.Context, code string) (result oauthIdentity, msg string) {

	result.Platform = model.OauthPlatformGitee

	setting := model.OauthPlatformConfig(model.OauthPlatformGitee)

	// 1) code 换 access_token（form 表单提交，redirect_uri 必须与授权时一致）
	token := oauthRequest(utils.CurlRequest{
		Method: "POST",
		Url:    "https://gitee.com/oauth/token",
		Headers: map[string]any{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Data: map[string]any{
			"grant_type":    "authorization_code",
			"code":          code,
			"client_id":     cast.ToString(setting["app_id"]),
			"client_secret": cast.ToString(setting["app_key"]),
			"redirect_uri":  cast.ToString(setting["redirect"]),
		},
	})
	if token.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 access_token", token.Error)
	}

	accessToken := cast.ToString(token.Json["access_token"])
	if utils.Is.Empty(accessToken) {
		return result, oauthErrorText(token, "获取 access_token 失败")
	}

	// 2) 取用户信息
	user := oauthRequest(utils.CurlRequest{
		Method: "GET",
		Url:    "https://gitee.com/api/v5/user",
		Query: map[string]any{
			"access_token": accessToken,
		},
	})
	if user.Error != nil {
		return result, oauthLangError(ctx, result.Platform, "获取 Gitee 用户信息", user.Error)
	}

	result.Openid = cast.ToString(user.Json["id"])
	if utils.Is.Empty(result.Openid) {
		return result, oauthErrorText(user, "获取 Gitee 用户信息失败")
	}

	result.Nickname = utils.Default(cast.ToString(user.Json["name"]), cast.ToString(user.Json["login"]))
	result.Avatar = cast.ToString(user.Json["avatar_url"])

	return result, ""
}

// ============================== 工具函数 ==============================

// oauthHttpClient - 第三方请求的 http.Client：统一超时 + 可选代理
//
// 超时与代理都来自后台「系统配置 → 第三方登录」——
// 自建服务器常因出网受限连接不上 github.com（表现为 context deadline exceeded），
// 这时要么放通服务器出网，要么在这里给第三方请求配一个 HTTP/SOCKS5 代理。
func oauthHttpClient() *http.Client {

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if proxy := model.OauthProxy(); !utils.Is.Empty(proxy) {
		if parsed, err := url.Parse(proxy); err == nil && !utils.Is.Empty(parsed.Host) {
			transport.Proxy = http.ProxyURL(parsed)
		} else {
			facade.Log.Warn(map[string]any{"proxy": proxy}, "第三方登录代理地址无法解析，已回退为直连")
		}
	}

	return &http.Client{
		Timeout:   model.OauthTimeout(),
		Transport: transport,
	}
}

// oauthLangError - 把网络层错误翻译成「能照着做」的提示
//
// 出网受限时原始错误只有一句 context deadline exceeded，
// 站长看不出该改什么；这里按错误类型给出可操作的建议。
// 原始错误作为参数传入 facade.Lang（而不是拼进模板），避免错误文本里的 % 被当成占位符。
func oauthLangError(ctx *gin.Context, platform, action string, err error) string {

	text := ""
	if err != nil {
		text = err.Error()
	}
	name := model.OauthPlatformName(platform)

	switch {
	case strings.Contains(text, "Client.Timeout") || strings.Contains(text, "deadline exceeded"):
		return facade.Lang(ctx, "无法连接 %v（%v）：服务器出网超时。请检查服务器能否访问外网，或在后台「系统配置 → 第三方登录」中配置 HTTP 代理（原始错误：%v）", name, action, text)
	case strings.Contains(text, "no such host"):
		return facade.Lang(ctx, "无法连接 %v（%v）：域名解析失败，请检查服务器 DNS 设置（原始错误：%v）", name, action, text)
	case strings.Contains(text, "connection refused") || strings.Contains(text, "connection reset") || strings.Contains(text, "EOF"):
		return facade.Lang(ctx, "无法连接 %v（%v）：连接被拒绝或中断，多为网络限制所致，可尝试配置 HTTP 代理（原始错误：%v）", name, action, text)
	}

	return facade.Lang(ctx, "调用 %v 接口失败（%v）：%v", name, action, text)
}

// oauthRequest - 统一的第三方请求
//
//   - 固定请求头：User-Agent（GitHub 会直接拒绝无 UA 的请求）+ accept: application/json；
//   - 固定超时：utils.Curl 默认的 http.Client 没有超时；
//   - 注意：utils.Curl 默认 Content-Type 是 application/json，
//     需要 form 提交时必须显式传 application/x-www-form-urlencoded（否则参数进不了表单）。
func oauthRequest(request utils.CurlRequest) *utils.CurlResponse {

	if utils.Is.Empty(request.Method) {
		request.Method = "GET"
	}
	if request.Headers == nil {
		request.Headers = map[string]any{}
	}
	if _, ok := request.Headers["User-Agent"]; !ok {
		request.Headers["User-Agent"] = "inis-oauth"
	}
	if _, ok := request.Headers["accept"]; !ok {
		request.Headers["accept"] = "application/json"
	}
	if request.Client == nil {
		request.Client = oauthHttpClient()
	}

	return utils.Curl(request).Send()
}

// oauthErrorText - 从三方响应里挑一条可读的错误信息（各平台字段名不统一）
func oauthErrorText(response *utils.CurlResponse, fallback string) string {
	for _, key := range []string{"error_description", "error", "msg", "message"} {
		if text := cast.ToString(response.Json[key]); !utils.Is.Empty(text) {
			return fmt.Sprintf("%v：%v", fallback, text)
		}
	}
	return fallback
}

// oauthFirstString - 依次取第一个非空字段（三方头像字段名与大小档位不一）
func oauthFirstString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := cast.ToString(data[key]); !utils.Is.Empty(value) {
			return value
		}
	}
	return ""
}

// oauthNickname - 第三方昵称归一化：去标签 + 去空白 + 截断（users.nickname 是 size:32）
//
// 不做「含恶意代码就整单拒绝」：昵称来自第三方，站长没法要求对方改，
// 这里按纯文本清洗、超长截断，避免一串脏数据把注册流程卡死。
func oauthNickname(nickname string) string {

	nickname = strings.TrimSpace(facade.Comm.SanitizeHTML(strings.TrimSpace(nickname)))
	nickname = strings.TrimSpace(nickname)

	if utils.Is.Empty(nickname) {
		return fmt.Sprintf("用户_%v", utils.Rand.Number(6))
	}

	// 按「字符」截断，避免把多字节字符截坏
	runes := []rune(nickname)
	if len(runes) > 32 {
		nickname = string(runes[:32])
	}

	return nickname
}

// oauthAvatar - 第三方头像归一化（含 & 的地址要 SanitizeURL，否则会被转义成 &amp; 导致破图）
func oauthAvatar(avatar string) string {

	if utils.Is.Empty(avatar) {
		return fmt.Sprintf("https://img.zhuxu.asia/tx/%d.png", utils.Rand.Int(1, 5))
	}

	return facade.Comm.SanitizeURL(avatar)
}

// oauthCachedTicket - 读取绑定票据（缓存实现不同，类型可能是 facade.H 或 map[string]any）
func oauthCachedTicket(value any) facade.H {
	switch item := value.(type) {
	case facade.H:
		return item
	case map[string]any:
		return facade.H(item)
	default:
		return nil
	}
}
