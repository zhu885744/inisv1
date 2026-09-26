package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Comm struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Comm) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Comm) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"login":            this.login,
		"register":         this.register,
		"check-token":      this.checkToken,
		"reset-password":   this.resetPassword,
		"logout":           this.logout,
		"verify-email":     this.verifyEmail,    // 邮箱验证（注册验证方式为 email 时使用）
		"send-verify-mail": this.sendVerifyMail, // 重发注册验证邮件
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - PUT请求本体
func (this *Comm) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - DELETE请求本体
func (this *Comm) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"logout": this.logout,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - GET请求本体
func (this *Comm) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// 登录
func (this *Comm) login(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}
	// 请求参数
	params := this.params(ctx, map[string]any{
		"source": "default",
	})
	// 请求头信息
	headers := this.headers(ctx)

	if utils.Is.Empty(params["account"]) {
		this.json(ctx, nil, facade.Lang(ctx, "请提交帐号（或邮箱和手机号）！"), 400)
		return
	}

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "请提交密码！"), 400)
		return
	}

	// 正则表达式，匹配通过空格分割的两个16位任意字符 `^(\w{16}) (\w{16})$`
	reg := regexp.MustCompile(`^([\w+]{16})\D+([\w+]{16})$`)
	match := reg.FindStringSubmatch(cast.ToString(headers["X-Gorgon"]))

	// 密文解密
	if match != nil {

		cipher := utils.AES(match[1], match[2])

		// 只要有一个为空，就不是我们要的数据
		if utils.Is.Empty(headers["X-Khronos"]) || utils.Is.Empty(headers["X-Argus"]) {
			this.json(ctx, nil, facade.Lang(ctx, "账号或密码错误！"), 400)
			return
		}

		decode := cipher.Decrypt([]byte(cast.ToString(headers["X-Argus"])))
		if decode.Error != nil {
			this.json(ctx, nil, facade.Lang(ctx, "账号或密码错误！"), 400)
			return
		}

		// 解密后的数据
		text := cast.ToStringMap(utils.Json.Decode(decode.Text))

		if utils.Is.Empty(text["account"]) || utils.Is.Empty(text["password"]) || utils.Is.Empty(text["unix"]) {
			this.json(ctx, nil, facade.Lang(ctx, "账号或密码错误！"), 400)
			return
		}

		// 验证时间戳
		if cast.ToString(text["unix"]) != cast.ToString(headers["X-Khronos"]) {
			this.json(ctx, nil, facade.Lang(ctx, "账号或密码错误！"), 400)
			return
		}

		// 1、当前时间戳 - 提交的时间戳 > 60秒 = 过期
		// 2、如果结果为负数，说明提交的时间戳大于当前时间戳，也是过期
		diff := time.Now().Unix() - cast.ToInt64(text["unix"])
		if diff > 60 || diff < -60 {
			this.json(ctx, gin.H{
				"diff": diff,
				"unix": text["unix"],
				"now":  time.Now().Unix(),
			}, facade.Lang(ctx, "账号或密码错误！"), 400)
			return
		}

		// 赋值
		params["account"] = text["account"]
		params["password"] = text["password"]
	}

	// 查询用户是否存在
	item, _ := facade.DB.Model(&table).Or([]any{
		[]any{"email", "=", params["account"]},
		[]any{"phone", "=", params["account"]},
		[]any{"account", "=", params["account"]},
	}).Where("source", params["source"]).Find()

	// 来源未命中时回退为不限定 source 再查一次：
	// 注册来源可能不是 default（如前台主题注册时写入 source=mellow），
	// 若登录仍按 default 过滤，这类账号将永远无法密码登录。
	// 账号/邮箱/手机号在保存时已有全局唯一校验，回退查询不会出现一对多歧义。
	if utils.Is.Empty(item) {
		item, _ = facade.DB.Model(&table).Or([]any{
			[]any{"email", "=", params["account"]},
			[]any{"phone", "=", params["account"]},
			[]any{"account", "=", params["account"]},
		}).Find()
	}

	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "账户不存在！"), 400)
		return
	}

	// 检查账号是否被冻结（使用 status 字段，0为正常，1为冻结）
	if table.Status == model.UserStatusFrozen {
		this.json(ctx, nil, facade.Lang(ctx, "当前账号已被冻结，请联系管理员！"), 403)
		return
	}

	// 检查账号是否处于「注册待审核」状态（开启人工审核注册的后台才会出现）
	if table.Status == model.UserStatusAudit {
		this.json(ctx, nil, facade.Lang(ctx, "账号正在审核中，请等待管理员审核通过后再登录！"), 403)
		return
	}

	// 检查邮箱是否完成验证（仅对「注册时写入 email_verified=0」的账号生效，
	// 未写入该标记的历史账号不受影响，避免开启邮箱验证后把老用户全部拦在门外）
	if setting := model.RegisterSettings(); setting.VerifyMode == model.RegisterVerifyEmail {
		if !table.EmailVerified() {
			this.json(ctx, nil, facade.Lang(ctx, "请先完成邮箱验证（验证邮件已发送至您的注册邮箱）！"), 403)
			return
		}
	}

	// 检查账号是否处于封禁状态（限制登录）
	if table.Restrictions&model.BanTypeLogin != 0 && table.CurrentBanId > 0 {
		banRecord, _ := facade.DB.Model(&model.UserBanRecords{}).Find(table.CurrentBanId)
		if !utils.Is.Empty(banRecord) {
			banMap := cast.ToStringMap(banRecord)
			if cast.ToInt(banMap["status"]) == model.BanStatusActive {
				reason := cast.ToString(banMap["reason"])
				duration := cast.ToInt(banMap["duration"])
				expiresAt := cast.ToInt64(banMap["expires_at"])

				msg := fmt.Sprintf("您的账号已被封禁！原因：%s", reason)
				if duration > 0 {
					remainingDays := (expiresAt - time.Now().Unix()) / 86400
					if remainingDays > 0 {
						msg += fmt.Sprintf("，剩余 %d 天", remainingDays)
					} else {
						msg += "，将于今日解封"
					}
				} else {
					msg = fmt.Sprintf("您的账号已被永久封禁！原因：%s", reason)
				}
				this.json(ctx, nil, facade.Lang(ctx, msg), 403)
				return
			}
		}
	}

	if utils.Is.Empty(table.Password) {
		this.json(ctx, nil, facade.Lang(ctx, "该帐号未设置密码，请切换登录方式！"), 400)
		return
	}

	// 密码校验
	if utils.Password.Verify(table.Password, params["password"]) == false {
		this.json(ctx, nil, facade.Lang(ctx, "密码错误！"), 400)
		return
	}

	jwt := facade.Jwt().Create(facade.H{
		"uid":  table.Id,
		"hash": utils.Hash.Sum32(table.Password),
	})

	// 删除 item 中的密码
	delete(item, "password")
	// 更新用户登录时间
	item["login_time"] = time.Now().Unix()
	facade.DB.Model(&table).Where("id", table.Id).Update(map[string]any{
		"login_time": item["login_time"],
	})

	result := map[string]any{
		"user":       item,
		"token":      jwt.Text,
		"valid_time": jwt.Valid, // 登录会话有效期（秒）
	}

	// 往客户端写入cookie - 存储登录token
	setToken(ctx, jwt.Text)
	// 登录增加经验
	go this.loginExp(item["id"])

	// 登录成功后，异步创建“账号登录通知”，记录账号/昵称/时间/IP/设备
	go func(uid int, account, nickname, ip, ua string) {
		notification := new(model.Notification)
		if _, e := notification.CreateLoginNotification(uid, account, nickname, ip, ua); e != nil {
			facade.Log.Error(map[string]any{"error": e.Error(), "uid": uid}, "发送账号登录通知失败")
		}
	}(cast.ToInt(item["id"]), cast.ToString(params["account"]), cast.ToString(item["nickname"]), ctx.ClientIP(), ctx.Request.UserAgent())

	this.json(ctx, result, facade.Lang(ctx, "登录成功！"), 200)
}

// 注册
func (this *Comm) register(ctx *gin.Context) {

	if !cast.ToBool(this.signInConfig(ctx)["value"]) {
		this.json(ctx, nil, "管理员关闭了注册功能！", 403)
		return
	}

	// 表数据结构体
	table := model.Users{}
	// 请求参数
	params := this.params(ctx, map[string]any{
		"source": "default",
	})

	// 验证器
	err := validator.NewValid("users", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if utils.Is.Empty(params["social"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "social"), 400)
		return
	}

	var social string
	social = utils.Ternary(utils.Is.Email(params["social"]), "email", social)
	social = utils.Ternary(utils.Is.Phone(params["social"]), "phone", social)

	if utils.Is.Empty(social) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 格式不正确！", "social"), 400)
		return
	}

	// ===== 注册扩展设置（/admin/system?tab=register）=====
	setting := model.RegisterSettings()

	// 邮箱域名限制：放在发送验证码之前，避免白白消耗一条短信/邮件
	if social == "email" {
		if err := model.CheckEmailDomain(setting, cast.ToString(params["social"])); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
			return
		}
	}

	// 开启「Email 验证」后必须用邮箱注册（手机号没有可验证的邮箱地址）
	if setting.VerifyMode == model.RegisterVerifyEmail && social != "email" {
		this.json(ctx, nil, facade.Lang(ctx, "本站已开启邮箱验证，请使用邮箱注册！"), 400)
		return
	}

	// 判断是否已经注册
	ok, _ := facade.DB.Model(&table).WithTrashed().Where([]any{
		[]any{"source", "=", params["source"]},
		[]any{social, "=", params["social"]},
	}).Exist()
	// 已注册
	if ok {
		switch social {
		case "email":
			this.json(ctx, nil, facade.Lang(ctx, "该邮箱已经注册！"), 400)
			return
		case "phone":
			this.json(ctx, nil, facade.Lang(ctx, "该手机号已经注册！"), 400)
			return
		}
	}

	if !utils.Is.Empty(params["account"]) {
		// 判断账号是否已经注册
		ok, _ := facade.DB.Model(&table).WithTrashed().Where([]any{
			[]any{"source", "=", params["source"]},
			[]any{"account", "=", params["account"]},
		}).Exist()
		if ok {
			this.json(ctx, nil, facade.Lang(ctx, "该帐号已经注册！"), 400)
			return
		}
	}

	cacheName := fmt.Sprintf("[register][%v=%v]", social, params["social"])

	// 验证码为空 - 发送验证码
	if utils.Is.Empty(params["code"]) {

		drives := cast.ToStringMap(facade.SMSToml.Get("drive"))
		drive := utils.Ternary(social == "email", "email", "sms")

		if utils.Is.Empty(drives[drive]) {
			this.json(ctx, nil, facade.Lang(ctx, "发送验证码失败！管理员未开启短信服务！"), 400)
			return
		}

		// 本地频控检查
		frequencyCacheName := fmt.Sprintf("frequency-%v-%v", drive, params["social"])
		dailyLimitCacheName := fmt.Sprintf("daily-limit-%v-%v", drive, params["social"])

		// 检查发送间隔（60秒）
		lastSendTime := facade.Cache.Get(frequencyCacheName)
		if !utils.Is.Empty(lastSendTime) {
			if time.Now().Unix()-cast.ToInt64(lastSendTime) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		// 检查每日发送限制（10次）
		dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
		if dailyCount >= 10 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		sms := facade.NewSMS(drives[drive]).VerifyCode(params["social"])
		if sms.Error != nil {
			// 处理阿里云频控错误
			if drive == "sms" && (strings.Contains(sms.Error.Error(), "check frequency failed") || strings.Contains(sms.Error.Error(), "FREQUENCY_FAIL")) {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请稍后再试！"), 400)
				return
			}
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}
		// 缓存验证码 - 5分钟
		facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
		// 缓存发送时间 - 60秒
		go facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*60)
		// 缓存每日发送次数 - 24小时
		go facade.Cache.Set(dailyLimitCacheName, dailyCount+1, time.Hour*24)
		this.json(ctx, nil, facade.Lang(ctx, "验证码发送成功！"), 201)
		return
	}

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "密码"), 400)
		return
	}

	// 获取缓存里面的验证码
	cacheCode := facade.Cache.Get(cacheName)

	if cast.ToString(params["code"]) != cacheCode {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	// 允许存储的字段
	allow := []any{"account", "password", "email", "phone", "nickname", "avatar", "description", "source"}
	// 动态给结构体赋值
	for key, val := range params {
		// 加密密码
		if key == "password" {
			val = utils.Password.Create(params["password"])
		} else if utils.Get.Type(val) == "string" {
			// 检测是否包含XSS攻击
			if key == "account" || key == "nickname" || key == "avatar" || key == "description" {
				if facade.Comm.DetectXSS(cast.ToString(val)) {
					this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
					return
				}
				// 头像等链接类字段使用 SanitizeURL，避免 URL 中的 & 被转义为 &amp;
				if key == "avatar" {
					val = facade.Comm.SanitizeURL(cast.ToString(val))
				} else {
					val = facade.Comm.SanitizeHTML(cast.ToString(val))
				}
			}
		}
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}
	utils.Struct.Set(&table, social, params["social"])

	// 未传入昵称时随机生成（如：用户_123456）
	if utils.Is.Empty(table.Nickname) {
		utils.Struct.Set(&table, "nickname", fmt.Sprintf("用户_%v", utils.Rand.Number(6)))
	}

	// 未传入头像时随机设置默认头像
	if utils.Is.Empty(table.Avatar) {
		utils.Struct.Set(&table, "avatar", fmt.Sprintf("https://img.zhuxu.asia/tx/%d.png", utils.Rand.Int(1, 5)))
	}

	// 未传入账号时，随机生成「小写字母 + 数字」的组合账号（无位数限制）
	// 借助 account 唯一索引，冲突时自动重新生成
	if utils.Is.Empty(table.Account) {
		const maxRetry = 20
		for i := 0; i < maxRetry; i++ {
			// 小写字母 + 数字随机值
			account := utils.Rand.String(10, "abcdefghijklmnopqrstuvwxyz0123456789")
			utils.Struct.Set(&table, "account", account)
			// 设置登录时间
			utils.Struct.Set(&table, "login_time", time.Now().Unix())
			// 创建用户
			_, err = facade.DB.Model(&table).Create(&table)
			if err == nil {
				break
			}
			// 非唯一索引冲突（Error 1062）则重试，其他错误直接返回
			if !strings.Contains(cast.ToString(err.Error()), "1062") {
				this.json(ctx, nil, err.Error(), 400)
				return
			}
		}
		if err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "注册失败，请稍后重试！"), 400)
			return
		}
	} else {
		// 设置登录时间
		utils.Struct.Set(&table, "login_time", time.Now().Unix())
		// 创建用户
		_, err = facade.DB.Model(&table).Create(&table)
		if err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	}

	// 删除验证码
	go facade.Cache.Del(cacheName)

	// 默认权限组：同步执行（必须在返回前落库，否则前端拿到 token 后立刻校验登录态，
	// 可能先把「空的权限缓存」写进 Cache（该缓存无过期时间），导致默认权限长期不生效）
	this.auth(table.Id)

	// 删除密码
	table.Password = ""

	// ===== 注册验证方式分流 =====
	switch setting.VerifyMode {
	case model.RegisterVerifyManual:
		// 人工审核：账号置为「待审核」，管理员在后台通过后才能登录
		if _, err := facade.DB.Model(&model.Users{}).Where("id", table.Id).
			UpdateColumn("status", model.UserStatusAudit); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "uid": table.Id}, "写入待审核状态失败")
		} else {
			table.Status = model.UserStatusAudit

			// 有新用户等待人工审核：通知管理员（开关见「系统设置 → 邮件通知」的 user.pending）
			go model.MailNotifyAdmin("user.pending", "有新用户等待审核", append(
				model.MailNotifyUserInfo(table.Id),
				"邮箱："+cast.ToString(table.Email),
				"时间："+model.MailNotifyTime(),
			)...)
		}

		this.json(ctx, gin.H{
			"user":       table,
			"need_audit": true,
		}, facade.Lang(ctx, "注册成功，请等待管理员审核通过后再登录！"), 200)
		return

	case model.RegisterVerifyEmail:
		// 邮箱验证：标记未验证并发送验证邮件，验证通过后才能登录
		if err := model.MarkEmailUnverified(table.Id); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "uid": table.Id}, "写入邮箱未验证标记失败")
		}
		if err := model.SendRegisterVerifyMail(table.Id, cast.ToString(table.Email), this.baseURL(ctx)); err != nil {
			// 邮件发送失败不阻断注册（账号已创建），但要把原因告知前端，便于用户重发
			this.json(ctx, gin.H{
				"user":        table,
				"need_verify": true,
				"email":       table.Email,
				"mail_error":  err.Error(),
			}, facade.Lang(ctx, "注册成功，但验证邮件发送失败，请稍后在登录页重新发送！"), 200)
			return
		}

		this.json(ctx, gin.H{
			"user":        table,
			"need_verify": true,
			"email":       table.Email,
		}, facade.Lang(ctx, "注册成功，请前往邮箱完成验证后登录！"), 200)
		return
	}

	jwt := facade.Jwt().Create(facade.H{
		"uid":  table.Id,
		"hash": utils.Hash.Sum32(table.Password),
	})

	result := map[string]any{
		"user":       table,
		"token":      jwt.Text,
		"valid_time": jwt.Valid, // 登录会话有效期（秒）
	}

	// 往客户端写入cookie - 存储登录token
	setToken(ctx, jwt.Text)
	// 登录增加经验
	go this.loginExp(table.Id)
	// 注册欢迎消息 / 欢迎邮件（按后台开关执行，内部异步）
	model.SendWelcome(table.Id, table.Account, table.Nickname, cast.ToString(table.Email))

	this.json(ctx, result, facade.Lang(ctx, "注册成功！"), 200)
}

// baseURL - 拼出前台站点根地址，用于邮件里的验证链接
// 优先取 config/app.toml 的 app.domain（反向代理场景），否则回退到当前请求的 scheme + host
func (this *Comm) baseURL(ctx *gin.Context) string {

	domain := strings.TrimSpace(cast.ToString(facade.AppToml.Get("app.domain", "")))
	if !utils.Is.Empty(domain) {
		if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
			domain = "https://" + domain
		}
		return strings.TrimRight(domain, "/")
	}

	scheme := "http"
	if ctx.Request.TLS != nil || strings.EqualFold(ctx.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}

	return fmt.Sprintf("%v://%v", scheme, ctx.Request.Host)
}

// verifyEmail - 邮箱验证（注册验证方式为 email 时，用户点击邮件链接后调用）
// 参数：token（邮件里的验证 token）
func (this *Comm) verifyEmail(ctx *gin.Context) {

	params := this.params(ctx)
	token := cast.ToString(params["token"])

	uid := model.ConsumeMailToken(token)
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "验证链接无效或已过期，请重新发送验证邮件！"), 400)
		return
	}

	if err := model.MarkEmailVerified(uid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "邮箱验证失败，请稍后重试！"), 400)
		return
	}

	// 验证通过后补发注册欢迎消息 / 欢迎邮件
	user, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if !utils.Is.Empty(user) {
		model.SendWelcome(uid, cast.ToString(user["account"]), cast.ToString(user["nickname"]), cast.ToString(user["email"]))
	}

	this.json(ctx, gin.H{"uid": uid}, facade.Lang(ctx, "邮箱验证成功，请登录！"), 200)
}

// sendVerifyMail - 重发注册验证邮件
// 参数：email（注册邮箱）；仅对「已注册且邮箱未验证」的账号发送，避免被当作探测接口
func (this *Comm) sendVerifyMail(ctx *gin.Context) {

	params := this.params(ctx)
	email := strings.TrimSpace(cast.ToString(params["email"]))

	if !utils.Is.Email(email) {
		this.json(ctx, nil, facade.Lang(ctx, "邮箱格式不正确！"), 400)
		return
	}

	setting := model.RegisterSettings()
	if setting.VerifyMode != model.RegisterVerifyEmail {
		this.json(ctx, nil, facade.Lang(ctx, "当前未开启邮箱验证！"), 400)
		return
	}

	user, _ := facade.DB.Model(&model.Users{}).Where("email", email).Find()
	if utils.Is.Empty(user) {
		// 不暴露账号是否存在，统一提示「已发送」
		this.json(ctx, nil, facade.Lang(ctx, "如果该邮箱已注册，验证邮件将发送到您的邮箱！"), 200)
		return
	}

	uid := cast.ToInt(user["id"])
	if model.IsEmailVerified(user["json"]) {
		this.json(ctx, nil, facade.Lang(ctx, "该邮箱已完成验证，请直接登录！"), 400)
		return
	}

	if err := model.SendRegisterVerifyMail(uid, email, this.baseURL(ctx)); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "验证邮件已发送，请注意查收！"), 200)
}

// 忘记密码
func (this *Comm) resetPassword(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"source": "default",
	})

	// social 不能为空
	if utils.Is.Empty(params["social"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "手机号/邮箱"), 400)
		return
	}

	// 验证器
	err := validator.NewValid("users", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 检查类型，邮箱或者手机号
	var socialType string
	if utils.Is.Email(params["social"]) {
		socialType = "email"
	} else if utils.Is.Phone(params["social"]) {
		socialType = "phone"
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "请输入正确的手机号或邮箱！"), 400)
		return
	}

	// 找回密码（不传 user，让 password 函数自己查询）
	this.password(ctx, socialType)
}

// 忘记密码
func (this *Comm) password(ctx *gin.Context, socialType string) {

	// 请求参数
	params := this.params(ctx)

	drives := cast.ToStringMap(facade.SMSToml.Get("drive"))

	// 获取用户提交的联系方式
	social := cast.ToString(params["social"])

	// 确定模式和驱动
	var mode string
	var drive string

	if socialType == "email" {
		mode = "email"
		drive = cast.ToString(drives["email"])
	} else if socialType == "phone" {
		mode = "sms"
		drive = cast.ToString(drives["sms"])
	}

	// 驱动不可用
	if utils.Is.Empty(drive) {
		if mode == "email" {
			this.json(ctx, nil, facade.Lang(ctx, "管理员未开启邮箱服务，无法发送验证码！"), 400)
		} else {
			this.json(ctx, nil, facade.Lang(ctx, "管理员未开启短信服务，无法发送验证码！"), 400)
		}
		return
	}

	// 通过 social 查询用户
	var user map[string]any
	table := model.Users{}
	user, _ = facade.DB.Model(&table).Where(socialType, social).Find()

	if utils.Is.Empty(user) {
		if mode == "email" {
			this.json(ctx, nil, facade.Lang(ctx, "该邮箱未注册！"), 400)
		} else {
			this.json(ctx, nil, facade.Lang(ctx, "该手机号未注册！"), 400)
		}
		return
	}

	// 缓存名称 - 使用 mode 而非具体驱动名，确保发送和验证时 key 一致
	cacheName := fmt.Sprintf("[reset-password][%v=%v]", mode, social)

	// 验证码为空 - 发送验证码
	if utils.Is.Empty(params["code"]) {

		// 本地频控检查（仅短信模式）
		if drive == "sms" {
			frequencyCacheName := fmt.Sprintf("frequency-%v-%v", drive, social)
			dailyLimitCacheName := fmt.Sprintf("daily-limit-%v-%v", drive, social)

			// 检查发送间隔（60秒）
			lastSendTime := facade.Cache.Get(frequencyCacheName)
			if !utils.Is.Empty(lastSendTime) {
				if time.Now().Unix()-cast.ToInt64(lastSendTime) < 60 {
					this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
					return
				}
			}

			// 检查每日发送限制（10次）
			dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
			if dailyCount >= 10 {
				this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
				return
			}

			sms := facade.NewSMS(drive).VerifyCode(social)
			if sms.Error != nil {
				// 处理阿里云频控错误
				if strings.Contains(sms.Error.Error(), "check frequency failed") || strings.Contains(sms.Error.Error(), "FREQUENCY_FAIL") {
					this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请稍后再试！"), 400)
					return
				}
				this.json(ctx, nil, sms.Error.Error(), 400)
				return
			}
			// 缓存验证码 - 5分钟
			facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
			// 缓存发送时间 - 60秒
			go facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*60)
			// 缓存每日发送次数 - 24小时
			go facade.Cache.Set(dailyLimitCacheName, dailyCount+1, time.Hour*24)

			msg := fmt.Sprintf("验证码发送至您的%v：%s，请注意查收！", utils.Ternary(mode == "email", "邮箱", "手机"), social)
			this.json(ctx, nil, facade.Lang(ctx, msg), 201)
			return
		}

		sms := facade.NewSMS(drive).VerifyCode(social)
		if sms.Error != nil {
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}
		// 缓存验证码 - 5分钟
		facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)

		msg := fmt.Sprintf("验证码发送至您的%v：%s，请注意查收！", utils.Ternary(mode == "email", "邮箱", "手机"), social)
		this.json(ctx, nil, facade.Lang(ctx, msg), 201)
		return
	}

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "密码"), 400)
		return
	}

	// 获取缓存里面的验证码
	cacheCode := facade.Cache.Get(cacheName)

	if cast.ToString(params["code"]) != cast.ToString(cacheCode) {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	// 加密密码
	password := utils.Password.Create(params["password"])

	// 更新密码
	_, err := facade.DB.Model(&model.Users{}).Where("id", user["id"]).UpdateColumn("password", password)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 删除验证码
	go facade.Cache.Del(cacheName)

	this.json(ctx, nil, facade.Lang(ctx, "密码重置成功！"), 200)
}

// 校验token
func (this *Comm) checkToken(ctx *gin.Context) {

	params := this.params(ctx)

	tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))

	var token string
	if !utils.Is.Empty(ctx.Request.Header.Get("Authorization")) {
		token = ctx.Request.Header.Get("Authorization")
	} else {
		token, _ = ctx.Cookie(tokenName)
	}

	if utils.Is.Empty(token) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "Authorization"), 412)
		return
	}

	// 解析token
	jwt := facade.Jwt().Parse(token)
	if jwt.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "%s 无效！", "Authorization"), 400)
		return
	}

	// 表数据结构体
	table := model.Users{}
	// 查询用户
	item, _ := facade.DB.Model(&table).Where("id", jwt.Data["uid"]).Find()
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 204)
		return
	}

	// token 有效时长
	valid := jwt.Valid

	if cast.ToBool(params["renew"]) {
		jwt = facade.Jwt().Create(facade.H{
			"uid":  table.Id,
			"hash": utils.Hash.Sum32(table.Password),
		})
		token = jwt.Text
		valid = cast.ToInt64(utils.Calc(facade.AppToml.Get("jwt.expire", facade.DefaultJwtExpire)))
		// 往客户端写入cookie - 存储登录token
		setToken(ctx, token)
	}

	delete(item, "password")

	this.json(ctx, gin.H{
		"user":       item,
		"token":      token,
		"valid_time": valid,
	}, facade.Lang(ctx, facade.Lang(ctx, "合法的token！")), 200)
}

// 退出登录
func (this *Comm) logout(ctx *gin.Context) {

	host := ctx.Request.Host
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))

	// 清除 cookie：domain 必须与登录时 setToken 的 domain 一致，否则无法清除
	// 同时兼容带域名和空域名两种写法
	ctx.SetCookie(tokenName, "", -1, "/", host, false, false)
	ctx.SetCookie(tokenName, "", -1, "/", "", false, false)

	this.json(ctx, nil, facade.Lang(ctx, "退出成功！"), 200)
}

// 设置登录token到客户的cookie中
func setToken(ctx *gin.Context, token any) {

	expire := cast.ToInt(utils.Calc(facade.CryptToml.Get("jwt.expire", facade.DefaultJwtExpire)))
	tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))

	// domain 传空：写入 host-only cookie（不带 Domain 属性）。
	// 关键：必须与 abortWithError 中的清除方式保持一致——浏览器按 name+domain+path
	// 三元组匹配 cookie，若写入带 Domain=host 而清除时不带，二者是不同条目，
	// 将导致 401 时 cookie 永远无法被清除（旧 token 持续污染后续请求）。
	ctx.SetCookie(tokenName, cast.ToString(token), expire, "/", "", false, false)
}

// 获取注册配置
func (this *Comm) signInConfig(ctx *gin.Context) (result map[string]any) {

	// 是否允许注册
	cacheName := "[GET]config[ALLOW_REGISTER]"

	// 如果缓存中存在，则直接使用缓存中的数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	// 不存在则查询数据库
	result, _ = facade.DB.Model(&model.Config{}).Where("key", "ALLOW_REGISTER").Find()
	// 写入缓存
	go facade.Cache.Set(cacheName, result)

	return result
}

// 登录增加经验值
func (this *Comm) loginExp(uid any) {
	_ = (&model.EXP{}).Add(model.EXP{
		Type:        "login",
		Uid:         cast.ToInt(uid),
		Description: "登录奖励！",
	})
	// 登录同时赚取积分
	_ = (&model.Integral{}).Add(model.Integral{
		Type: "login",
		Uid:  cast.ToInt(uid),
	})
}

// 添加默认权限
// 注册后为新用户分配「默认权限组」：分组 ID 来自 ALLOW_REGISTER 配置的 text 字段
// （形如 "|1|2|" / "1,2"，由 utils.Unity.Ids 按数字提取，空值表示不分配任何权限组）
func (this *Comm) auth(uid any) {

	// 获取注册配置
	config, _ := facade.DB.Model(&model.Config{}).Where("key", "ALLOW_REGISTER").Find()
	// 配置不存在 - 跳过
	if utils.Is.Empty(config) {
		return
	}

	// 默认权限
	ids := utils.Unity.Ids(config["text"])

	// 是否真正写入了权限组（用于判断要不要清权限缓存）
	changed := false

	for _, id := range ids {
		// 查找权限分组数据
		item, _ := facade.DB.Model(&model.AuthGroup{}).WithTrashed().Where("id", id).Find()
		// 分组不存在 - 跳过（注意是 continue：某个 ID 失效不应中断后续分组）
		if utils.Is.Empty(item) {
			continue
		}
		uids := utils.Unity.Ids(item["uids"])
		// 如果分组中没有该用户
		if !utils.In.Array(uid, uids) {
			uids = append(uids, uid)
			if _, err := facade.DB.Model(&model.AuthGroup{}).Where("id", id).Update(map[string]any{
				"uids": fmt.Sprintf("|%v|", strings.Join(cast.ToStringSlice(utils.ArrayUnique(utils.ArrayEmpty(uids))), "|")),
			}); err == nil {
				changed = true
			}
		}
	}

	// 权限变化后清掉该用户的权限缓存（key 形如 user[uid][rule-group]，无过期时间）
	if changed {
		go facade.Cache.DelTags(fmt.Sprintf("user[%v]", uid))
	}
}
