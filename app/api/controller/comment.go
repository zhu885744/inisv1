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
	commentAllowFields = "pid,content,bind_id,bind_type,editor,json,text"
	commentAllowQuery  = "id,pid,bind_id,bind_type,editor"
)

var commentAllowFieldsSlice = []any{"pid", "content", "bind_id", "bind_type", "editor", "json", "text"}
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
		item := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

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
	count := query.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

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

	item := mold.Order("RAND()").Limit(limit).Select()
	data := this.maskCommentData(ctx, utils.ArrayMapWithField(item, params["field"]))

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

	var comment map[string]any
	switch params["bind_type"] {
	case "article":
		article := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(article) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(article["json"])["comment"])
	case "page":
		page := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(page) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(page["json"])["comment"])
	default:
		comment = this.config("comment")
	}

	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = this.config("comment")["allow"]
	}

	if cast.ToInt(comment["allow"]) == 0 {
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

	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "更新用户经验协程发生错误")
			}
		}()
		_ = (&model.EXP{}).Add(model.EXP{
			Uid:      user.Id,
			Type:     "comment",
			BindId:   table.BindId,
			BindType: table.BindType,
		})
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "发送评论邮件通知协程发生错误")
			}
		}()

		commentConfig := this.config("comment")
		facade.Log.Info(map[string]any{"config": commentConfig}, "加载评论配置")

		emailNotify := cast.ToStringMap(commentConfig["email_notify"])
		facade.Log.Info(map[string]any{"email_notify": emailNotify}, "加载邮件通知配置")

		if cast.ToInt(emailNotify["enabled"]) == 1 {
			facade.Log.Info(nil, "邮件通知功能已开启")

			retryCount := cast.ToInt(emailNotify["retry_count"])
			retryInterval := cast.ToInt(emailNotify["retry_interval"])
			facade.Log.Info(map[string]any{"retry_count": retryCount, "retry_interval": retryInterval}, "邮件发送重试配置")

			userInfo := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			userEmail := cast.ToString(cast.ToStringMap(userInfo)["email"])
			facade.Log.Info(map[string]any{"user_id": user.Id, "user_email": facade.Comm.MaskEmail(userEmail), "user_name": cast.ToString(cast.ToStringMap(userInfo)["name"])}, "获取评论用户信息")

			title := ""
			switch table.BindType {
			case "article":
				article := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					title = cast.ToString(cast.ToStringMap(article)["title"])
				}
			case "page":
				page := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					title = cast.ToString(cast.ToStringMap(page)["title"])
				}
			}

			createdAt := time.Now().Format("2006-01-02 15:04:05")
			facade.Log.Info(map[string]any{"local_time": createdAt}, "获取服务器本地时间")

			commentInfo := map[string]any{
				"content":      table.Content,
				"created_at":   createdAt,
				"author_name":  cast.ToString(cast.ToStringMap(userInfo)["nickname"]),
				"author_email": userEmail,
				"ip":           table.Ip,
				"bind_type":    table.BindType,
				"bind_id":      table.BindId,
				"title":        title,
			}
			facade.Log.Info(map[string]any{"comment_id": table.Id, "bind_type": table.BindType, "bind_id": table.BindId}, "评论通知处理")

			var authorEmail string
			switch table.BindType {
			case "article":
				article := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					authorId := cast.ToInt(cast.ToStringMap(article)["uid"])
					author := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"article_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取文章作者信息")
				} else {
					facade.Log.Warn(map[string]any{"article_id": table.BindId}, "文章不存在，跳过发送给作者")
				}
			case "page":
				page := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					authorId := cast.ToInt(cast.ToStringMap(page)["uid"])
					author := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"page_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取页面作者信息")
				} else {
					facade.Log.Warn(map[string]any{"page_id": table.BindId}, "页面不存在，跳过发送给作者")
				}
			default:
				facade.Log.Warn(map[string]any{"bind_type": table.BindType}, "未知绑定类型，跳过发送给作者")
			}

			if utils.Is.Email(authorEmail) && authorEmail != userEmail {
				facade.Log.Info(map[string]any{"recipient": facade.Comm.MaskEmail(authorEmail)}, "开始发送邮件给作者")
				for i := 0; i <= retryCount; i++ {
					sms := facade.NewSMS("email")
					response := sms.SendCommentNotify(authorEmail, commentInfo)
					if response.Error == nil {
						facade.Log.Info(map[string]any{"recipient": facade.Comm.MaskEmail(authorEmail)}, "邮件发送给作者成功")
						break
					}
					if i < retryCount {
						facade.Log.Warn(map[string]any{"error": response.Error, "retry": i + 1}, "邮件发送给作者失败，准备重试")
						time.Sleep(time.Duration(retryInterval) * time.Second)
					} else {
						facade.Log.Error(map[string]any{"error": response.Error, "recipient": facade.Comm.MaskEmail(authorEmail)}, "发送文章作者邮件通知失败")
					}
				}
			} else {
				facade.Log.Warn(map[string]any{"author_email": facade.Comm.MaskEmail(authorEmail), "user_email": facade.Comm.MaskEmail(userEmail)}, "作者邮箱无效或重复，跳过发送")
			}

			if table.Pid > 0 {
				facade.Log.Info(map[string]any{"pid": table.Pid}, "检测到回复评论，准备通知被回复用户")
				parentComment := facade.DB.Model(&model.Comment{}).Where("id", table.Pid).Find()
				if !utils.Is.Empty(parentComment) {
					parentUid := cast.ToInt(cast.ToStringMap(parentComment)["uid"])
					parentUser := facade.DB.Model(&model.Users{}).Where("id", parentUid).Find()
					parentEmail := cast.ToString(cast.ToStringMap(parentUser)["email"])
					facade.Log.Info(map[string]any{"parent_comment_id": table.Pid, "parent_user_id": parentUid, "parent_email": facade.Comm.MaskEmail(parentEmail)}, "获取被回复用户信息")

					if utils.Is.Email(parentEmail) && parentUid != user.Id {
						facade.Log.Info(map[string]any{"recipient": facade.Comm.MaskEmail(parentEmail)}, "开始发送邮件给被回复用户")
						for i := 0; i <= retryCount; i++ {
							sms := facade.NewSMS("email")
							response := sms.SendReplyNotify(parentEmail, commentInfo)
							if response.Error == nil {
								facade.Log.Info(map[string]any{"recipient": facade.Comm.MaskEmail(parentEmail)}, "邮件发送给被回复用户成功")
								break
							}
							if i < retryCount {
								facade.Log.Warn(map[string]any{"error": response.Error, "retry": i + 1}, "邮件发送给被回复用户失败，准备重试")
								time.Sleep(time.Duration(retryInterval) * time.Second)
							} else {
								facade.Log.Error(map[string]any{"error": response.Error, "recipient": facade.Comm.MaskEmail(parentEmail)}, "发送评论回复邮件通知失败")
							}
						}
					} else {
						facade.Log.Warn(map[string]any{"parent_email": facade.Comm.MaskEmail(parentEmail), "parent_uid": parentUid, "user_id": user.Id}, "被回复用户邮箱无效或为评论者本人，跳过发送")
					}
				} else {
					facade.Log.Warn(map[string]any{"pid": table.Pid}, "父评论不存在，跳过发送给被回复用户")
				}
			} else {
				facade.Log.Info(map[string]any{"pid": table.Pid}, "不是回复评论，跳过发送给被回复用户")
			}
		} else {
			facade.Log.Info(nil, "邮件通知功能未开启")
		}
	}()

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
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
	allow := []any{"content", "editor", "json", "text"}
	async := utils.Async[map[string]any]()

	root := this.meta.root(ctx)
	if root {
		allow = append(allow, "pid", "bind_id", "bind_type")
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

	if !root && cast.ToInt(item.Find()["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	tx := item.Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Comment) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Comment{}), params)
	query = this.buildQuery(query, params)
	this.json(ctx, query.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Comment) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Sum(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Comment) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Min(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Comment) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Max(field)
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
		data = utils.ArrayMapWithField(query.Select(), params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

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

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := item.Delete(ids)

	if tx.Error != nil {
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

	tx := item.Force().Delete(ids)

	if tx.Error != nil {
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

	ids := utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := item.Force().Delete()

	if tx.Error != nil {
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

	ids = utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.Comment{}).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
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
	count := mold.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])

		list := cast.ToSlice(data)
		for key, val := range list {
			ids := this.replies(cast.ToStringMap(val)["id"], ctx)
			replies := facade.DB.Model(&[]model.Comment{}).WhereIn("id", ids).Order("create_time asc").Select()
			cast.ToStringMap(list[key])["replies"] = utils.ArrayMapWithField(replies, params["field"])
		}

		data = list
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

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
	}

	cacheName := "config[" + configKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if isCommentConfig {
		config = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
		if cacheState {
			go facade.Cache.Set(cacheName, config)
		}
	} else {
		if cacheState && facade.Cache.Has(cacheName) {
			config = cast.ToStringMap(facade.Cache.Get(cacheName))
		} else {
			config = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
			if cacheState {
				go facade.Cache.Set(cacheName, config)
			}
		}
	}

	if isCommentConfig {
		return cast.ToStringMap(config["json"])
	}

	if len(key) > 0 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[0])])
	}

	return cast.ToStringMap(config["json"])
}
