package controller

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Moments struct {
	base
}

const (
	momentsAllowFields = "content,images,location,json,text,publish_time,status"
	momentsAllowQuery  = "id,top,views"
)

var momentsAllowFieldsSlice = []any{"content", "images", "location", "json", "text", "publish_time", "status"}
var momentsAllowQuerySlice = []any{"id", "top", "views"}

func (this *Moments) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

// isSelfQuery - 判断当前查询是否为"查询自己的动态"（where.uid 等于当前登录用户）
// 用户在「我的动态」中需要看到自己的草稿与待审核内容，
// 因此这种情况下不再强制追加 audit=1 的过滤条件。
func (this *Moments) isSelfQuery(ctx *gin.Context, params map[string]any) bool {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		return false
	}

	var where map[string]any
	switch val := params["where"].(type) {
	case map[string]any:
		where = val
	case string:
		if !utils.Is.Empty(val) {
			_ = json.Unmarshal([]byte(val), &where)
		}
	}

	if utils.Is.Empty(where) {
		return false
	}

	val := where["uid"]

	// 支持 where.uid = ["=", 123] 这类数组形式
	if arr, ok := val.([]any); ok {
		for _, v := range arr {
			if !utils.Is.Empty(v) && cast.ToInt(v) == uid {
				return true
			}
		}
		return false
	}

	return cast.ToInt(val) == uid
}

func (this *Moments) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Moments) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Moments) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Moments) processFieldValue(val any) any {
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

func (this *Moments) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":           this.one,
		"all":           this.all,
		"sum":           this.sum,
		"min":           this.min,
		"max":           this.max,
		"rand":          this.rand,
		"count":         this.count,
		"column":        this.column,
		"comment":       this.comment,
		"comment_count": this.commentCount,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Moments) IPOST(ctx *gin.Context) {
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

	go this.delCache()
}

func (this *Moments) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
		"set_top": this.setTop,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Moments) IDEL(ctx *gin.Context) {
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

	go this.delCache()
}

func (this *Moments) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Moments) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "moments"})
}

func (this *Moments) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Moments{}

	for key, val := range params {
		if utils.In.Array(key, momentsAllowQuerySlice) {
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

		if !this.meta.root(ctx) {
			query = query.Where("audit", 1)
		}

		item, _ := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	// 更新动态浏览量
	go this.updateViews(ctx, cast.ToStringMap(data))

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Moments) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "top desc, create_time desc",
	})

	table := model.Moments{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Moments

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 非管理员默认只能看已审核内容；但查询"自己的动态"时放开，
	// 以便用户在「我的动态」中管理草稿与待审核内容。
	// 这里用 permit()（与权限中间件同一口径，见 meta.permit 注释）而不是 root()：
	// 站点常把管理员放在「非 root 分组」里只勾选权限点，用 root() 会误判为普通用户
	// 而强制追加 audit=1，导致后台「待审核」筛选（where.audit=0）永远查不到数据。
	if !this.meta.permit(ctx) && !this.isSelfQuery(ctx, params) {
		query = query.Where("audit", 1)
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

func (this *Moments) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.Moments{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Moments{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	items, _ := mold.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *Moments) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Moments) create(ctx *gin.Context) {
	params := this.params(ctx)
	var err error

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Moments{Uid: uid, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}
	allowFields := append([]any{}, momentsAllowFieldsSlice...)
	root := this.meta.root(ctx)
	if root {
		// reason（驳回原因）与 audit 一样只允许管理员写，普通作者不能自己填 / 清
		allowFields = append(allowFields, "audit", "top", "reason")
	}

	status := cast.ToInt(params["status"])

	if status == 0 {
		utils.Struct.Set(&table, "Audit", 1)
		utils.Struct.Set(&table, "Status", 0)
		utils.Struct.Set(&table, "PublishTime", 0)
	} else {
		// 是否开启审核：配置存放在 config.json.audit（与文章/页面口径一致）。
		// 此前误读顶层 config["audit"]（该列不存在，恒为 false），导致审核开关永远失效、
		// 发布即通过审核，后台「待审核」自然没有任何数据。
		audit := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
		utils.Struct.Set(&table, "Audit", cast.ToInt(!audit))
		// 状态固定为「已发布」：待审核 ≠ 草稿（原来复用 !audit，会把待审核动态写成草稿，混进草稿筛选）
		utils.Struct.Set(&table, "Status", 1)

		if publishTime, ok := params["publish_time"]; ok && cast.ToInt64(publishTime) > 0 {
			utils.Struct.Set(&table, "PublishTime", cast.ToInt64(publishTime))
		} else {
			utils.Struct.Set(&table, "PublishTime", time.Now().Unix())
		}
	}

	for key, val := range params {
		if utils.In.Array(key, allowFields) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 待审核：通知管理员去审核（邮件受「系统设置 → 邮件通知」的 moments.pending 控制，站内信始终发）
	// audit：0 待审核 / 1 通过 / 2 未通过；草稿（status=0）audit 恒为 1，不会误发
	if cast.ToInt(table.Audit) == 0 {
		go notifyMomentsAudit(uid, table.Id, table.Content, cast.ToInt(table.Audit), table.Reason)
	}

	if status == 0 {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "草稿保存成功！"), 200)
	} else {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
		go (&model.EXP{}).Add(model.EXP{
			Type:        "moments",
			Uid:         uid,
			BindId:      table.Id,
			BindType:    "moments",
			Description: "发布动态奖励",
		})
		go (&model.Integral{}).Add(model.Integral{
			Uid:  uid,
			Type: "moments",
		})
	}
}

func (this *Moments) update(ctx *gin.Context) {
	params := this.params(ctx)
	var err error

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.Moments{}
	async := utils.Async[map[string]any]()
	allowFields := append([]any{}, momentsAllowFieldsSlice...)
	root := this.meta.root(ctx)
	if root {
		// reason（驳回原因）与 audit 一样只允许管理员写
		allowFields = append(allowFields, "audit", "top", "reason")
	}

	status := cast.ToInt(params["status"])

	// 原文状态：用于判断是否「首次发布」，避免每次编辑都把审核状态重置为待审核
	prev, _ := facade.DB.Model(&model.Moments{}).WithTrashed().Where("id", params["id"]).Find()

	if status == 0 {
		async.Set("audit", 1)
		async.Set("status", 0)
	} else {
		async.Set("status", 1)
		// 审核开关读取 config.json.audit（与文章/页面口径一致，此前误读顶层 audit 恒为 false）
		auditSwitch := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
		if !auditSwitch {
			async.Set("audit", 1)
		} else if cast.ToInt(prev["status"]) == 0 || cast.ToInt(prev["audit"]) == 0 {
			// 仅「首次发布」（原为草稿 / 尚未审核）进入待审核，已审核过的编辑不再打回
			async.Set("audit", 0)
		}
		if publishTime, ok := params["publish_time"]; ok && cast.ToInt64(publishTime) > 0 {
			async.Set("publish_time", cast.ToInt64(publishTime))
		}
	}

	for key, val := range params {
		if utils.In.Array(key, allowFields) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	async.Set("last_update", time.Now().Unix())

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	if !this.meta.root(ctx) {
		itemData, _ := item.Find()
		if cast.ToInt(itemData["uid"]) != this.user(ctx).Id {
			this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
			return
		}
	}

	// 取一次更新内容：Update 与「审核状态变化」判定共用
	payload := async.Result()
	// 审核通过时清空驳回原因：作者不该继续看到已经过期的原因
	if cast.ToInt(payload["audit"]) == 1 {
		payload["reason"] = ""
	}
	_, err = item.Scan(&table).Update(payload)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 审核状态变化时通知（邮件 + 站内信），普通编辑保存（audit 未变）不打扰任何人
	// audit：0 待审核 / 1 通过 / 2 未通过；后台「批量审核」在前端是逐条调 update，
	// 这里按条判定状态是否变化，天然不会重复发
	//
	// 注意：必须判断 payload 里**是否真的带了 audit** —— 作者编辑自己「已审核过」的动态时
	// audit 不会进 payload（保持原状态），此时 cast.ToInt(nil) 会得到 0，
	// 不加这个判断就会把「未通过(2)」误判成「变成待审核(0)」，给管理员发一条假通知。
	if rawAudit, ok := payload["audit"]; ok {
		if nowAudit := cast.ToInt(rawAudit); nowAudit != cast.ToInt(prev["audit"]) {
			content := cast.ToString(payload["content"])
			if utils.Is.Empty(content) {
				content = cast.ToString(prev["content"])
			}
			go notifyMomentsAudit(cast.ToInt(prev["uid"]), table.Id, content, nowAudit, cast.ToString(payload["reason"]))
		}
	}

	if status == 0 {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "草稿保存成功！"), 200)
	} else {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
	}
}

// momentsSummary 动态正文摘要（通知文案用）
//
// 复用 search.go 的 cleanSearchText：去掉 HTML 标签、图片 / 链接语法、Markdown 标记并折叠空白，
// 否则正文里的 <img src="..."> 会原样进到通知标题里。截断 30 字（与评论通知口径一致）。
func momentsSummary(content string) string {
	text := cleanSearchText(content)
	if text == "" {
		return "（无正文）"
	}
	if len([]rune(text)) > 30 {
		text = string([]rune(text)[:30]) + "..."
	}
	return text
}

// notifyMomentsAudit 动态审核通知（邮件 + 站内信），audit 为「变更后」的状态
//
//	0 待审核 → 通知管理员去审核（邮件开关 moments.pending，站内信始终发）
//	1 通过   → 通知作者（邮件开关 moments.passed，站内信始终发）
//	2 未通过 → 通知作者（邮件开关 moments.rejected，站内信始终发），并带上管理员填写的驳回原因
//
// 文案带动态正文摘要（动态没有标题，「内容」直接用摘要代入）；
// 站内信 bind_type 固定为 moments、bind_id 为动态 id，消息中心点击可跳转到动态页。
//
// 调用方负责判断「审核状态是否真的变了」（见 create / update），
// 因此反复编辑、批量审核都不会重复打扰。所有发送都在子协程里进行，不阻塞响应。
func notifyMomentsAudit(uid, bindId int, content string, audit int, reason string) {
	if uid <= 0 {
		return
	}

	summary := momentsSummary(content)
	now := model.MailNotifyTime()
	reason = strings.TrimSpace(reason)

	// 邮件正文：与其它场景保持一致的「字段名：值」风格
	lines := []string{"动态：" + summary, "时间：" + now}
	if audit == 2 && reason != "" {
		lines = append(lines, "驳回原因："+reason)
	}

	// 站内信正文：驳回原因放最前面（消息列表内容区只有 2 行高度，原因比时间更需要被看到）
	notifyContent := "时间：" + now
	if audit == 2 && reason != "" {
		notifyContent = "驳回原因：" + reason + " · " + notifyContent
	}

	switch audit {
	case 0:
		title := "您有新的动态「" + summary + "」待审核"
		account, nickname := model.MailNotifyUserIdentity(uid)
		model.MailNotifyAdmin("moments.pending", title, append(model.MailNotifyUserInfo(uid), lines...)...)
		model.NotifyAdmins(uid, model.NotificationTypeMoments, title,
			"作者："+nickname+"（"+account+"） · "+notifyContent, "moments", bindId)
	case 1:
		title := "您的动态「" + summary + "」审核已通过"
		model.MailNotifyUser(uid, "moments.passed", title, lines...)
		model.CreateUserNotify(uid, 0, model.NotificationTypeMoments, title, notifyContent, "moments", bindId)
	case 2:
		title := "您的动态「" + summary + "」审核未通过"
		model.MailNotifyUser(uid, "moments.rejected", title, lines...)
		model.CreateUserNotify(uid, 0, model.NotificationTypeMoments, title, notifyContent, "moments", bindId)
	}
}

func (this *Moments) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Moments{}), params)
	query = this.buildQuery(query, params)

	// 与 all 保持一致：管理员（含非 root 分组）可统计全部状态，
	// 否则后台「待审核」计数会被强制 audit=1 抹平为 0
	if !this.meta.permit(ctx) {
		query = query.Where("audit", 1)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Moments) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Moments{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}

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

func (this *Moments) sum(ctx *gin.Context) {
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

func (this *Moments) min(ctx *gin.Context) {
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

func (this *Moments) max(ctx *gin.Context) {
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

func (this *Moments) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Moments{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}

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

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Moments) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Moments{})
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

func (this *Moments) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Moments{}).WithTrashed()
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

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

func (this *Moments) clear(ctx *gin.Context) {
	table := model.Moments{}
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

func (this *Moments) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Moments{}).OnlyTrashed().WhereIn("id", ids)
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.Moments{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *Moments) comment(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	bindId := cast.ToInt(params["bind_id"])
	if bindId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	query = query.Where("bind_type", "moments").Where("bind_id", bindId)
	count, _ := query.Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item, _ := query.Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

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

func (this *Moments) commentCount(ctx *gin.Context) {
	params := this.params(ctx)
	bindId := cast.ToInt(params["bind_id"])

	query := facade.DB.Model(&model.Comment{})
	query = query.Where("bind_type", "moments")

	if bindId > 0 {
		query = query.Where("bind_id", bindId)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Moments) setTop(ctx *gin.Context) {
	params := this.params(ctx)

	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	isTop := cast.ToInt(params["top"])

	// 第一步：过滤出真实存在的动态ID（含已软删），并丢弃原查询链避免状态污染
	checkQuery := facade.DB.Model(&model.Moments{}).WithTrashed().WhereIn("id", ids)
	columnData, _ := checkQuery.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 第二步：使用全新的查询链执行更新，确保 WHERE 生效
	// 直接使用 map 传入 Updates(map)，GORM 对 map 会写入零值（top=0 也能落库）
	updateData := map[string]any{
		"top":         isTop,
		"last_update": time.Now().Unix(),
	}
	updateQuery := facade.DB.Model(&model.Moments{}).WithTrashed().WhereIn("id", ids)
	tx, err := updateQuery.Update(updateData)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "置顶设置失败！"), 400)
		return
	}

	if tx != nil && tx.RowsAffected == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "置顶设置未生效，无记录被更新！"), 204)
		return
	}

	topLabel := "取消置顶"
	if isTop == 1 {
		topLabel = "置顶设置"
	}

	this.json(ctx, gin.H{"ids": ids, "top": isTop}, facade.Lang(ctx, topLabel+"成功！"), 200)
}

func (this *Moments) updateViews(ctx *gin.Context, data map[string]any) {
	if utils.Is.Empty(data["id"]) {
		return
	}

	ip := ctx.ClientIP()
	userAgent := ctx.Request.UserAgent()
	momentID := cast.ToString(data["id"])

	deviceKey := ip + userAgent
	md5Hash := md5.Sum([]byte(deviceKey))
	cacheKey := "moments_views_cd:" + momentID + ":" + fmt.Sprintf("%x", md5Hash)

	if facade.Cache.Has(cacheKey) {
		return
	}

	facade.DB.Model(&model.Moments{}).Where("id", data["id"]).Inc("views", 1)
	facade.Cache.Set(cacheKey, true, 86400)
}

func (this *Moments) config(ctx *gin.Context) (result map[string]any) {
	cacheName := "[GET]config[MOMENTS]"

	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	result, _ = facade.DB.Model(&model.Config{}).Where("key", "MOMENTS").Find()
	go facade.Cache.Set(cacheName, result)

	return result
}
