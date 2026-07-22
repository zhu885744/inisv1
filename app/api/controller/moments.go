package controller

import (
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
	momentsAllowQuery  = "id"
)

var momentsAllowFieldsSlice = []any{"content", "images", "location", "json", "text", "publish_time", "status"}
var momentsAllowQuerySlice = []any{"id"}

func (this *Moments) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
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

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Moments) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Moments{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Moments

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
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
		allowFields = append(allowFields, "audit")
	}

	status := cast.ToInt(params["status"])

	if status == 0 {
		utils.Struct.Set(&table, "Audit", 1)
		utils.Struct.Set(&table, "Status", 0)
		utils.Struct.Set(&table, "PublishTime", 0)
	} else {
		audit := cast.ToBool(this.config(ctx)["audit"])
		utils.Struct.Set(&table, "Audit", cast.ToInt(!audit))
		utils.Struct.Set(&table, "Status", cast.ToInt(!audit))

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
		allowFields = append(allowFields, "audit")
	}

	status := cast.ToInt(params["status"])

	if status == 0 {
		async.Set("audit", 1)
		async.Set("status", 0)
	} else {
		audit := cast.ToBool(this.config(ctx)["audit"])
		async.Set("audit", cast.ToInt(!audit))
		async.Set("status", cast.ToInt(!audit))
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

	_, err = item.Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if status == 0 {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "草稿保存成功！"), 200)
	} else {
		this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
	}
}

func (this *Moments) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Moments{}), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
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

func (this *Moments) config(ctx *gin.Context) (result map[string]any) {
	cacheName := "[GET]config[MOMENTS]"

	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	result, _ = facade.DB.Model(&model.Config{}).Where("key", "MOMENTS").Find()
	go facade.Cache.Set(cacheName, result)

	return result
}
