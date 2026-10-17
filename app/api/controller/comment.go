package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Comment struct {
	base
}

const (
	// 评论只有 1 公开 / 2 登录可见，没有付费可见，因此不接受 scope_integral
	commentAllowFields = "pid,content,images,bind_id,bind_type,editor,json,text,scope"
	commentAllowQuery  = "id,pid,bind_id,bind_type,editor"
)

var commentAllowFieldsSlice = []any{"pid", "content", "images", "bind_id", "bind_type", "editor", "json", "text", "scope"}
var commentAllowQuerySlice = []any{"id", "pid", "bind_id", "bind_type", "editor"}

func (this *Comment) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Comment) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Comment) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Comment) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Comment) processFieldValue(val any) any {
	switch utils.Get.Type(val) {
	case "map":
		return utils.Json.Encode(val)
	case "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

func (this *Comment) maskItem(item map[string]any) {
	if ip, ok := item["ip"].(string); ok && ip != "" {
		item["ip"] = facade.Comm.MaskIP(ip)
	}
	if agent, ok := item["agent"].(string); ok && agent != "" {
		item["agent"] = facade.Comm.MaskUA(agent)
	}
	if replies, ok := item["replies"]; ok {
		switch v := replies.(type) {
		case []map[string]any:
			for _, r := range v {
				this.maskItem(r)
			}
		case []any:
			for _, r := range v {
				if rm, ok := r.(map[string]any); ok {
					this.maskItem(rm)
				}
			}
		}
	}
}

func (this *Comment) maskCommentData(ctx *gin.Context, data any) any {
	if data == nil || this.meta.root(ctx) {
		return data
	}
	switch v := data.(type) {
	case map[string]any:
		this.maskItem(v)
		return v
	case []map[string]any:
		for i := range v {
			this.maskItem(v[i])
		}
		return v
	case []any:
		for i := range v {
			if m, ok := v[i].(map[string]any); ok {
				this.maskItem(m)
				v[i] = m
			}
		}
		return v
	}
	return data
}

func (this *Comment) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Comment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	fields := utils.Unity.Keys(params["field"])
	if utils.Is.Empty(fields) {
		return nil, ""
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		result := make(map[string]any)
		for _, val := range fields {
			result[cast.ToString(val)] = aggFunc(query, cast.ToString(val))
		}
		data = result
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		msg[0] = "数据请求成功！"
	}

	return data, facade.Lang(ctx, strings.Join(msg, ""))
}

func (this *Comment) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"sum":    this.sum,
		"min":    this.min,
		"max":    this.max,
		"rand":   this.rand,
		"count":  this.count,
		"column": this.column,
		"flat":   this.flat,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Comment) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":   this.save,
		"create": this.create,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove": this.remove,
		"delete": this.delete,
		"clear":  this.clear,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Comment) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "comment"})
}

func (this *Comment) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Comment{}

	for key, val := range params {
		if utils.In.Array(key, commentAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)
		item, _ := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论同样支持登录可见 / 付费可见，无权查看时清空正文
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Comment{}
	for key, val := range params {
		if utils.In.Array(key, commentAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 非管理员只看「审核通过」的评论（审核未通过 / 待审核的评论对前台不可见，口径同文章 / 动态）
	if !this.meta.root(ctx) {
		query = query.Where("audit", AuditPassed)
	}

	count, _ := query.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item, _ := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论列表同样按条脱敏（付费评论只保留占位信息）
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	mold := facade.DB.Model(&[]model.Comment{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		mold.Where("id", "NOT IN", except)
	}
	mold = this.buildQuery(mold, params)

	item, _ := mold.Order("RAND()").Limit(limit).Select()
	data := this.maskCommentData(ctx, utils.ArrayMapWithField(item, params["field"]))
	// 可见范围：随机获取的评论同样脱敏
	data = applyScopeData(ctx, data, "comment")

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *Comment) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Comment) create(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"bind_type": "article",
	})
	err := validator.NewValid("comment", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 评论开关优先级：全局 COMMENT.allow < 模块总开关（ARTICLE/MOMENTS/PAGE）< 内容自身 json.comment
	// 内容 / 模块取值：0 继承、1 允许/显示、2 禁止/隐藏；全局 COMMENT.allow：0 关闭、1 开启
	var comment map[string]any
	var moduleComment map[string]any

	switch params["bind_type"] {
	case "article":
		article, _ := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(article) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(article["json"])["comment"])
		moduleComment = this.config("article", "comment")
	case "page":
		page, _ := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(page) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(page["json"])["comment"])
		moduleComment = this.config("page", "comment")
	case "moments":
		moments, _ := facade.DB.Model(&model.Moments{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(moments) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的动态！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(moments["json"])["comment"])
		moduleComment = this.config("moments", "comment")
	default:
		comment = this.config("comment")
	}

	// 模块总开关为「禁止 / 隐藏」时，覆盖内容自身的设置
	if cast.ToInt(moduleComment["allow"]) == 2 {
		comment["allow"] = 2
	}
	if cast.ToInt(moduleComment["show"]) == 2 {
		comment["show"] = 2
	}

	// 内容未设置时继承模块总开关
	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = moduleComment["allow"]
	}
	if cast.ToInt(comment["show"]) == 0 {
		comment["show"] = moduleComment["show"]
	}

	// 模块总开关未配置时，回落到全局评论开关（COMMENT.allow：1 开启 / 0 关闭）
	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = this.config("comment")["allow"]
	}

	if cast.ToInt(comment["allow"]) == 2 || cast.ToInt(comment["allow"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "评论功能已关闭！"), 400)
		return
	}
	// 隐藏评论时整块不展示，同样不允许发表
	if cast.ToInt(comment["show"]) == 2 {
		this.json(ctx, nil, facade.Lang(ctx, "评论功能已关闭！"), 400)
		return
	}

	commentConfig := this.config("comment")

	rateLimit := cast.ToStringMap(commentConfig["rate_limit"])
	if cast.ToInt(rateLimit["enabled"]) == 1 {
		maxCount := cast.ToInt(rateLimit["max_count"])
		timeWindow := cast.ToInt(rateLimit["time_window"])

		now := time.Now().Unix()
		timeWindowKey := now / int64(timeWindow)
		cacheKey := fmt.Sprintf("comment:rate_limit:%d:%s:%d", user.Id, ctx.ClientIP(), timeWindowKey)

		currentCount := 0
		cacheValue := facade.Cache.Get(cacheKey)
		if cacheValue != nil {
			currentCount = cast.ToInt(cacheValue)
		}

		if currentCount >= maxCount {
			this.json(ctx, nil, facade.Lang(ctx, "评论过于频繁，请稍后再试！"), 429)
			return
		}

		facade.Cache.Set(cacheKey, currentCount+1, time.Duration(timeWindow)*time.Second)
	}

	maxLength := cast.ToInt(commentConfig["max_length"])
	if maxLength > 0 && utf8.RuneCountInString(cast.ToString(params["content"])) > maxLength {
		this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
		return
	}

	if cast.ToInt(commentConfig["require_chinese"]) == 1 {
		hasChinese := false
		for _, r := range cast.ToString(params["content"]) {
			if unicode.Is(unicode.Scripts["Han"], r) {
				hasChinese = true
				break
			}
		}
		if !hasChinese {
			this.json(ctx, nil, facade.Lang(ctx, "评论内容必须包含中文！"), 400)
			return
		}
	}

	if cast.ToInt(commentConfig["sensitive_filter"]) == 1 {
		sensitiveWords := cast.ToStringSlice(commentConfig["sensitive_words"])
		content := cast.ToString(params["content"])

		for _, word := range sensitiveWords {
			if strings.Contains(content, word) {
				this.json(ctx, nil, facade.Lang(ctx, "评论内容包含敏感词，请修改后重试！"), 400)
				return
			}
		}
	}

	agent := this.header(ctx, "User-Agent")
	if len(agent) > 511 {
		agent = agent[:511]
	}
	table := model.Comment{
		Uid:        user.Id,
		Agent:      agent,
		Ip:         ctx.ClientIP(),
		CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix(),
	}

	// 审核：关闭审核 → 直接「通过」；开启审核 → 「待审核」（口径见 audit.go，开关在「系统设置 → 评论配置」）。
	// 管理员自己发的评论直接通过（不必审核自己）；管理员在 payload 里显式指定 audit 时，
	// 下面遍历参数时会覆盖这里的判定（allow 里 audit 只对 root 开放）。
	// ⚠ 必须直接赋值：Struct.Set 按小写 tag（json:"audit"）匹配，传 "Audit" 会静默失效
	table.Audit = auditForCreate(contentAuditSwitch("COMMENT") && !this.meta.root(ctx), false)

	for key, val := range params {
		if utils.In.Array(key, commentAllowFieldsSlice) {
			switch utils.Get.Type(val) {
			case "string":
				if key == "content" || key == "text" {
					if facade.Comm.DetectXSS(cast.ToString(val)) {
						this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
						return
					}
					val = facade.Comm.SanitizeHTML(cast.ToString(val))
				}
			}
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	// 评论不支持付费可见：scope=3 一律降级为「登录可见」
	// （旧前端 / 历史数据里可能带 3，统一在这里收敛，避免评论出现积分墙）
	table.Scope = model.ScopeForBind("comment", table.Scope)

	// @ 提及：只匹配作者**已关注**的人，把 @昵称 规范化成 @[昵称](/author/uid)（前端渲染成可点击的链接）。
	// 规范化会让内容变长（@张三 → @[张三](/author/12)），所以长度要按转换后的结果再校验一次，
	// 避免突破 Comment.Content 的 varchar 上限导致插入失败。
	mentionTargets := []model.MentionTarget{}
	table.Content, mentionTargets = (&model.Mention{}).Normalize(table.Content, user.Id)

	// 话题：#话题# 规范化为 #[话题名](/topic/12)，前端渲染成可点击链接。
	// 评论表没有 tags 字段，所以只做链接、不参与话题内容聚合（见 model/topics.go）。
	// 同样放在长度校验之前：规范化会让内容变长。
	table.Content, _ = (&model.Topic{}).Parse(table.Content)

	if maxLength > 0 && utf8.RuneCountInString(table.Content) > maxLength {
		this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
		return
	}

	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "更新用户经验协程发生错误")
			}
		}()
		_ = (&model.EXP{}).Add(model.EXP{
			Uid:         user.Id,
			Type:        "comment-create",
			BindId:      table.Id,
			BindType:    table.BindType,
			Description: "发表评论奖励",
		})
		_ = (&model.Integral{}).Add(model.Integral{
			Uid:  user.Id,
			Type: "comment",
		})
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "发送评论邮件通知协程发生错误")
			}
		}()

		// 评论 / 回复的邮件开关已统一到「系统设置 → 邮件通知」：
		// 总开关 + 场景开关（comment.notify / comment.reply），分批投递与失败重试由邮箱队列负责；
		// 这里只组装邮件内容并交给对应场景发送（model.MailNotifyComment / MailNotifyReply）
		if model.MailNotifyEnabled("") {

			userInfo, _ := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			userEmail := cast.ToString(cast.ToStringMap(userInfo)["email"])
			facade.Log.Info(map[string]any{"user_id": user.Id, "user_email": facade.Comm.MaskEmail(userEmail), "user_name": cast.ToString(cast.ToStringMap(userInfo)["name"])}, "获取评论用户信息")

			title := ""
			bindLabel := ""
			switch table.BindType {
			case "article":
				article, _ := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					title = cast.ToString(cast.ToStringMap(article)["title"])
				}
				bindLabel = "文章"
			case "page":
				page, _ := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					title = cast.ToString(cast.ToStringMap(page)["title"])
				}
				bindLabel = "页面"
			case "moments":
				moment, _ := facade.DB.Model(&model.Moments{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(moment) {
					content := cast.ToString(cast.ToStringMap(moment)["content"])
					runes := []rune(content)
					if len(runes) > 50 {
						title = string(runes[:50]) + "..."
					} else {
						title = content
					}
				}
				bindLabel = "动态"
			}

			createdAt := time.Now().Format("2006-01-02 15:04:05")
			facade.Log.Info(map[string]any{"local_time": createdAt}, "获取服务器本地时间")

			commentInfo := map[string]any{
				"content":      table.Content,
				"created_at":   createdAt,
				"author_name":  cast.ToString(cast.ToStringMap(userInfo)["nickname"]),
				"author_email": userEmail,
				// 评论者账号：邮件模板里的 ${author_account}（不传会原样留下占位符）
				"author_account": commenterAccount(cast.ToStringMap(userInfo)),
				"ip":             table.Ip,
				"bind_type":      table.BindType,
				"bind_id":        table.BindId,
				"bind_label":     bindLabel,
				"title":          title,
			}
			facade.Log.Info(map[string]any{"comment_id": table.Id, "bind_type": table.BindType, "bind_id": table.BindId}, "评论通知处理")

			var authorEmail string
			switch table.BindType {
			case "article":
				article, _ := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					authorId := cast.ToInt(cast.ToStringMap(article)["uid"])
					author, _ := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"article_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取文章作者信息")
				} else {
					facade.Log.Warn(map[string]any{"article_id": table.BindId}, "文章不存在，跳过发送给作者")
				}
			case "page":
				page, _ := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					authorId := cast.ToInt(cast.ToStringMap(page)["uid"])
					author, _ := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"page_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取页面作者信息")
				} else {
					facade.Log.Warn(map[string]any{"page_id": table.BindId}, "页面不存在，跳过发送给作者")
				}
			case "moments":
				moment, _ := facade.DB.Model(&model.Moments{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(moment) {
					authorId := cast.ToInt(cast.ToStringMap(moment)["uid"])
					author, _ := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"moment_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取动态作者信息")
				} else {
					facade.Log.Warn(map[string]any{"moment_id": table.BindId}, "动态不存在，跳过发送给作者")
				}
			default:
				facade.Log.Warn(map[string]any{"bind_type": table.BindType}, "未知绑定类型，跳过发送给作者")
			}

			// 评论通知：发给内容作者（场景 comment.notify）
			// 作者邮箱无效、或评论者本人就是作者时不发（与原先的跳过规则一致）
			if utils.Is.Email(authorEmail) && authorEmail != userEmail {
				model.MailNotifyComment(authorEmail, commentInfo)
			}

			// 回复通知：发给被回复的人（场景 comment.reply）
			// 回复自己、或被回复人没有邮箱时不发
			if table.Pid > 0 {
				parentComment, _ := facade.DB.Model(&model.Comment{}).Where("id", table.Pid).Find()
				if !utils.Is.Empty(parentComment) {
					parentUid := cast.ToInt(cast.ToStringMap(parentComment)["uid"])
					parentUser, _ := facade.DB.Model(&model.Users{}).Where("id", parentUid).Find()
					parentEmail := cast.ToString(cast.ToStringMap(parentUser)["email"])

					if utils.Is.Email(parentEmail) && parentUid != user.Id {
						model.MailNotifyReply(parentEmail, commentInfo)
					}
				}
			}
		} else {
			facade.Log.Info(nil, "邮件通知总开关未开启，跳过评论邮件通知")
		}
	}()

	// 创建通知 - 新回复通知给文章/页面/动态作者
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "创建回复通知协程发生错误")
			}
		}()

		var authorId int
		var bindTitle string

		switch table.BindType {
		case "article":
			article, _ := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
			if !utils.Is.Empty(article) {
				authorId = cast.ToInt(cast.ToStringMap(article)["uid"])
				bindTitle = cast.ToString(cast.ToStringMap(article)["title"])
			}
		case "page":
			page, _ := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
			if !utils.Is.Empty(page) {
				authorId = cast.ToInt(cast.ToStringMap(page)["uid"])
				bindTitle = cast.ToString(cast.ToStringMap(page)["title"])
			}
		case "moments":
			moment, _ := facade.DB.Model(&model.Moments{}).Where("id", table.BindId).Find()
			if !utils.Is.Empty(moment) {
				authorId = cast.ToInt(cast.ToStringMap(moment)["uid"])
				content := cast.ToString(cast.ToStringMap(moment)["content"])
				if len([]rune(content)) > 30 {
					bindTitle = string([]rune(content)[:30]) + "..."
				} else {
					bindTitle = content
				}
			}
		}

		if authorId > 0 && authorId != user.Id {
			title := "收到新回复"
			notifUserInfo, _ := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			notifContent := cast.ToString(cast.ToStringMap(notifUserInfo)["nickname"]) + " 回复了你"
			if bindTitle != "" {
				notifContent += "的" + bindTitle
			}

			_, _ = (&model.Notification{}).CreateNotification(
				authorId, user.Id, "comment", title, notifContent, table.BindType, table.BindId,
			)
		}
	}()

	// @ 提及通知：只发给「被 @ 到的人」（内容作者那条已由上面的逻辑发出）。
	// 评论待审核（audit=0）时不发：评论可能不被展示，通知过去也看不到内容。
	if cast.ToInt(table.Audit) == 1 {
		go (&model.Mention{}).NotifyNew(mentionTargets, user.Id, "评论", table.BindType, table.BindId, table.Content, "")
	}

	// 返回本次写入的审核状态：前台据此决定提示语（开启审核时提示「审核通过后显示」）
	// 并插入一条「待审核」占位 —— 待审核的评论不在列表接口的返回里（非管理员只查 audit=1）
	this.json(ctx, gin.H{
		"id":      table.Id,
		"audit":   table.Audit,
		"pending": table.Audit == AuditPending,
	}, facade.Lang(ctx, "创建成功！"), 200)
}

// commenterAccount 评论者账号（评论邮件模板里的 ${author_account}，空时用昵称 / 「—」兜底）
func commenterAccount(commenter map[string]any) string {
	if account := strings.TrimSpace(cast.ToString(commenter["account"])); !utils.Is.Empty(account) {
		return account
	}
	if nickname := strings.TrimSpace(cast.ToString(commenter["nickname"])); !utils.Is.Empty(nickname) {
		return nickname
	}
	return "—"
}

func (this *Comment) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("comment", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.Comment{}
	allow := []any{"content", "images", "editor", "json", "text"}
	async := utils.Async[map[string]any]()

	root := this.meta.root(ctx)
	if root {
		// audit / reason 只对管理员开放：这是后台列表「通过 / 驳回」的操作入口
		allow = append(allow, "pid", "bind_id", "bind_type", "audit", "reason")
	}

	for key, val := range params {
		if utils.In.Array(key, allow) {
			switch utils.Get.Type(val) {
			case "string":
				if key == "content" || key == "text" {
					if facade.Comm.DetectXSS(cast.ToString(val)) {
						this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
						return
					}
					val = facade.Comm.SanitizeHTML(cast.ToString(val))
				}
			}
			async.Set(key, this.processFieldValue(val))
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	itemData, _ := item.Find()
	if !root && cast.ToInt(itemData["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	payload := async.Result()

	// @ 提及：编辑评论时同样规范化（匹配范围仍是评论作者已关注的人），
	// 否则手动输入的 @昵称 永远变不成链接
	if _, ok := payload["content"]; ok {
		payload["content"], _ = (&model.Mention{}).Normalize(cast.ToString(payload["content"]), cast.ToInt(itemData["uid"]))
		// 话题：与新建一致，只做链接不做聚合（评论没有 tags 字段）
		payload["content"], _ = (&model.Topic{}).Parse(cast.ToString(payload["content"]))
	}

	_, err = item.Scan(&table).Update(payload)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 评论审核通过时补发 @ 提及通知：待审核阶段故意不发（评论不展示，点进去看不到），
	// 所以「待审核 → 通过」这一刻必须补 —— 这就是后台「通过」按钮这条路径。
	// 只在审核状态真正发生变化时补，普通编辑（audit 未进 payload）不会重复打扰。
	if rawAudit, ok := payload["audit"]; ok {
		if nowAudit := cast.ToInt(rawAudit); nowAudit == AuditPassed && nowAudit != cast.ToInt(itemData["audit"]) {
			go model.NotifyMentionsOnCommentApprove(cast.ToInt(params["id"]))
		}
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Comment) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Comment{}), params)
	query = this.buildQuery(query, params)

	// 非管理员只统计「审核通过」的评论：保证前台显示的评论数与列表一致
	if !this.meta.root(ctx) {
		query = query.Where("audit", AuditPassed)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Comment) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Sum(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Comment) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Min(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Comment) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Max(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Comment) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Comment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		items, _ := query.Select()
		data = utils.ArrayMapWithField(items, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：column 也可能取到 content，同样按条脱敏
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{})

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Comment) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{}).WithTrashed()

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Force().Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Comment) clear(ctx *gin.Context) {
	table := model.Comment{}
	item := facade.DB.Model(&table).OnlyTrashed()

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *Comment) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{}).OnlyTrashed().WhereIn("id", ids)

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.Comment{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *Comment) replies(pid any, ctx *gin.Context) (ids []int) {
	params := this.params(ctx)

	var result []model.Comment
	mold := facade.DB.Model(&result).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
	mold.Where("pid", pid).Column("id")
	mold.Select()

	for _, val := range result {
		ids = append(ids, val.Id)
		ids = append(ids, this.replies(val.Id, ctx)...)
	}

	return ids
}

func (this *Comment) flat(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":      1,
		"bind_type": "article",
		"order":     "create_time desc",
	})

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "bind_id 不能为空！"), 400)
		return
	}

	table := model.Comment{}
	for key, val := range params {
		if key == "bind_id" || key == "bind_type" {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment

	mold := facade.DB.Model(&result).Where("pid", 0)
	mold = this.withTrashOptions(mold, params)
	mold = this.buildQuery(mold, params)
	count, _ := mold.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item, _ := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])

		list := cast.ToSlice(data)
		for key, val := range list {
			ids := this.replies(cast.ToStringMap(val)["id"], ctx)
			replies, _ := facade.DB.Model(&[]model.Comment{}).WhereIn("id", ids).Order("create_time asc").Select()
			cast.ToStringMap(list[key])["replies"] = utils.ArrayMapWithField(replies, params["field"])
		}

		data = list
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论树（含回复）逐层脱敏
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) config(key ...any) (json map[string]any) {
	var config map[string]any
	configKey := "ARTICLE"

	isCommentConfig := false
	if len(key) > 0 && cast.ToString(key[0]) == "comment" {
		configKey = "COMMENT"
		isCommentConfig = true
	} else if len(key) > 0 && cast.ToString(key[0]) == "moments" {
		configKey = "MOMENTS"
	} else if len(key) > 0 && cast.ToString(key[0]) == "page" {
		// 独立页面此前会落进默认的 ARTICLE，导致继承文章配置
		configKey = "PAGE"
	} else if len(key) > 0 && cast.ToString(key[0]) == "article" {
		configKey = "ARTICLE"
	}

	cacheName := "config[" + configKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if isCommentConfig {
		config, _ = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
		if cacheState {
			go facade.Cache.Set(cacheName, config)
		}
	} else {
		if cacheState && facade.Cache.Has(cacheName) {
			config = cast.ToStringMap(facade.Cache.Get(cacheName))
		} else {
			config, _ = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
			if cacheState {
				go facade.Cache.Set(cacheName, config)
			}
		}
	}

	if isCommentConfig {
		return cast.ToStringMap(config["json"])
	}

	if len(key) > 1 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[1])])
	}

	if len(key) > 0 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[0])])
	}

	return cast.ToStringMap(config["json"])
}
