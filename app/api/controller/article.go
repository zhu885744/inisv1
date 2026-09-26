package controller

import (
	"crypto/md5"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Article struct {
	base
}

const (
	articleAllowFields = "title,abstract,content,covers,tags,group,editor,remark,json,text,publish_time,status"
	articleAllowQuery  = "id"
)

var articleAllowFieldsSlice = []any{"title", "abstract", "content", "covers", "tags", "group", "editor", "remark", "json", "text", "publish_time", "status"}
var articleAllowQuerySlice = []any{"id"}

func (this *Article) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Article) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Article) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Article) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Article) processFieldValue(val any) any {
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

func (this *Article) IGET(ctx *gin.Context) {
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
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Article) IPOST(ctx *gin.Context) {
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

func (this *Article) IPUT(ctx *gin.Context) {
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

	go this.delCache()
}

func (this *Article) IDEL(ctx *gin.Context) {
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

func (this *Article) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Article) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "article"})
	// 文章变动会影响标签的引用文章数（tags 接口的 article_count），同步清理标签缓存
	facade.Cache.DelTags([]any{"[GET]", "tags"})
}

func (this *Article) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Article{}

	for key, val := range params {
		if utils.In.Array(key, articleAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	// 非管理员：判断是否为「作者本人查看」（本人可预览自己的待审核 / 未通过文章）
	uid := this.user(ctx).Id
	self := false
	if !this.meta.root(ctx) && uid > 0 {
		row, _ := facade.DB.Model(&model.Article{}).WithTrashed().Where("id", table.Id).Find()
		self = cast.ToInt(row["uid"]) == uid
	}

	cacheName := this.cache.name(ctx)
	// 作者本人看到的结果可能包含未审核内容，与公共缓存不一致，因此不读也不写缓存
	if !self {
		if cached, ok := this.getFromCache(ctx, cacheName); ok {
			msg[1] = "（来自缓存）"
			data = cached
		}
	}

	if utils.Is.Empty(data) {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)

		// 非管理员只能看到已审核通过的文章，但作者本人可查看自己的全部状态
		if !this.meta.root(ctx) && !self {
			query = query.Where("audit", 1)
		}

		item, _ := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		if !self {
			this.setCache(ctx, cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	// 更新用户经验
	go this.updateExp(ctx, cast.ToStringMap(data))
	// 更新文章浏览量
	go this.updateViews(ctx, cast.ToStringMap(data))

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Article) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Article{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Article

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 非管理员默认只返回「已审核通过」的文章；
	// 但查询条件已限定为本人文章时不过滤审核状态，
	// 这样作者能在「我的文章」里看到自己的待审核 / 未通过内容
	if !this.meta.root(ctx) && !this.isSelfQuery(ctx, params) {
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

func (this *Article) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.Article{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Article{}).Where("id", "IN", ids)
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

func (this *Article) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Article) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("article", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Article{Uid: uid, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}
	allowFields := append([]any{}, articleAllowFieldsSlice...)
	root := this.meta.root(ctx)
	if root {
		allowFields = append(allowFields, "top", "audit")
	}

	// 获取状态：0-草稿，1-发布
	status := cast.ToInt(params["status"])

	// 如果是草稿，跳过审核检查，不设置发布时间
	if status == 0 {
		utils.Struct.Set(&table, "Audit", 1)
		utils.Struct.Set(&table, "Status", 0)
		utils.Struct.Set(&table, "PublishTime", 0)
	} else {
		// 是否开启了审核
		audit := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
		utils.Struct.Set(&table, "Audit", cast.ToInt(!audit))
		utils.Struct.Set(&table, "Status", 1)

		// 处理 publish_time，若未传则默认使用当前时间
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

	// 待审核：通知管理员去审核（开关见「系统设置 → 邮件通知」的 article.pending）
	// audit：0 待审核 / 1 通过 / 2 未通过
	if table.Audit == 0 {
		go model.MailNotifyAdmin("article.pending", "有新的文章待审核", append(
			model.MailNotifyUserInfo(uid),
			"标题："+table.Title,
			"时间："+model.MailNotifyTime(),
		)...)
	}

	// 发布文章时触发经验值与积分
	if status == 1 {
		go func() {
			(&model.EXP{}).Add(model.EXP{
				Uid:         uid,
				Type:        "article-create",
				BindType:    "article",
				BindId:      table.Id,
				Description: "发布文章奖励",
			})
			_ = (&model.Integral{}).Add(model.Integral{
				Uid:  uid,
				Type: "article-create",
			})
		}()
	}

	if status == 0 {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "草稿保存成功！"), 200)
	} else {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
	}
}

func (this *Article) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("article", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.Article{}
	async := utils.Async[map[string]any]()
	allowFields := append([]any{}, articleAllowFieldsSlice...)
	root := this.meta.root(ctx)
	if root {
		allowFields = append(allowFields, "top", "audit")
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	findResult, _ := item.Find()
	if !root && cast.ToInt(findResult["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	// 获取状态：0-草稿，1-发布
	status := cast.ToInt(params["status"])
	// 原文状态：用于判断是否「首次发布」，避免每次编辑都把审核状态重置为待审核
	prevStatus := cast.ToInt(findResult["status"])
	prevAudit := cast.ToInt(findResult["audit"])

	if status == 0 {
		// 草稿：跳过审核，不设置发布时间
		async.Set("audit", 1)
		async.Set("status", 0)
	} else {
		async.Set("status", 1)

		// 审核规则：未开启审核 → 直接通过；
		// 开启审核时，只有「首次发布」（原状态为草稿 / 尚未审核过）才进入待审核，
		// 已审核过的文章再次编辑保存不会重置审核状态（审核状态可由管理员在编辑页修改）
		auditSwitch := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
		if !auditSwitch {
			async.Set("audit", 1)
		} else if prevStatus == 0 || prevAudit == 0 {
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

	// 取一次更新内容：Update 与「审核状态变化」判定共用
	payload := async.Result()
	_, err = item.Scan(&table).Update(payload)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 审核状态变化时邮件通知（开关见「系统设置 → 邮件通知」）
	// audit：0 待审核 / 1 通过 / 2 未通过；只在状态真正变化时发，普通编辑不打扰
	notifyAuditChange(cast.ToInt(findResult["uid"]), "article",
		cast.ToString(findResult["title"]), prevAudit, payload["audit"])

	if status == 0 {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "草稿保存成功！"), 200)
	} else {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
	}
}

// isSelfQuery - 查询条件是否限定为「当前登录用户自己的文章」（与 moments / links 同名同义）
// 作者在「我的文章」中需要看到自己的草稿与待审核 / 未通过内容，因此不再强制追加 audit=1。
// 注意：这里要求 where.uid 有且只有当前用户（比 moments 的「包含即可」更严格），
// 避免 uid 传数组时把他人未审核的文章一并带出来。
func (this *Article) isSelfQuery(ctx *gin.Context, params map[string]any) (ok bool) {

	uid := this.user(ctx).Id
	if uid == 0 {
		return false
	}

	// where 支持 JSON 字符串（前端）与 map 两种形式
	where := cast.ToStringMap(params["where"])
	if len(where) == 0 {
		return false
	}

	value, exist := where["uid"]
	if !exist {
		return false
	}

	switch item := value.(type) {
	case []any:
		ids := cast.ToIntSlice(item)
		return len(ids) == 1 && ids[0] == uid
	case []int:
		return len(item) == 1 && item[0] == uid
	default:
		return cast.ToInt(value) == uid
	}
}

func (this *Article) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Article{}), params)
	query = this.buildQuery(query, params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Article) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Article{}), params)
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

func (this *Article) sum(ctx *gin.Context) {
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

func (this *Article) min(ctx *gin.Context) {
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

func (this *Article) max(ctx *gin.Context) {
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

func (this *Article) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Article{}), params)
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

func (this *Article) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Article{})
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

func (this *Article) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Article{}).WithTrashed()
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

func (this *Article) clear(ctx *gin.Context) {
	table := model.Article{}
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

func (this *Article) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Article{}).OnlyTrashed().WhereIn("id", ids)
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.Article{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *Article) updateExp(ctx *gin.Context, data map[string]any) {
	user := this.meta.user(ctx)
	if user.Id == 0 || utils.Is.Empty(data) {
		return
	}
	_ = (&model.EXP{}).Add(model.EXP{
		Uid:      user.Id,
		Type:     "visit",
		BindId:   cast.ToInt(data["id"]),
		BindType: "article",
	})
}

func (this *Article) updateViews(ctx *gin.Context, data map[string]any) {
	if utils.Is.Empty(data["id"]) {
		return
	}

	ip := ctx.ClientIP()
	userAgent := ctx.Request.UserAgent()
	articleID := cast.ToString(data["id"])

	deviceKey := ip + userAgent
	md5Hash := md5.Sum([]byte(deviceKey))
	cacheKey := "article_views_cd:" + articleID + ":" + fmt.Sprintf("%x", md5Hash)

	if facade.Cache.Has(cacheKey) {
		return
	}

	facade.DB.Model(&model.Article{}).Where("id", data["id"]).Inc("views", 1)
	facade.Cache.Set(cacheKey, true, 86400)
}

func (this *Article) config(ctx *gin.Context) (result map[string]any) {
	cacheName := "[GET]config[ARTICLE]"

	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	result, _ = facade.DB.Model(&model.Config{}).Where("key", "ARTICLE").Find()
	go facade.Cache.Set(cacheName, result)

	return result
}
