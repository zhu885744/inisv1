package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	apiKeysAllowFields = "value,remark,json,text"
	apiKeysAllowQuery  = "id"
)

var apiKeysAllowFieldsSlice = []any{"value", "remark", "json", "text"}
var apiKeysAllowQuerySlice = []any{"id"}

type ApiKeys struct {
	base
}

func (this *ApiKeys) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *ApiKeys) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *ApiKeys) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *ApiKeys) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *ApiKeys) processFieldValue(key string, val any) any {
	if key == "value" {
		val = strings.ToUpper(cast.ToString(val))
	}
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

func (this *ApiKeys) generateAPIKey() string {
	UUID := uuid.New().String()
	UUID = strings.Replace(UUID, "-", "", -1)
	return strings.ToUpper(UUID)
}

func (this *ApiKeys) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.ApiKeys{}), params)
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

func (this *ApiKeys) IGET(ctx *gin.Context) {
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

func (this *ApiKeys) IPOST(ctx *gin.Context) {
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

func (this *ApiKeys) IPUT(ctx *gin.Context) {
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

func (this *ApiKeys) IDEL(ctx *gin.Context) {
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

func (this *ApiKeys) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *ApiKeys) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "api-keys"})
}

func (this *ApiKeys) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.ApiKeys{}

	for key, val := range params {
		if utils.In.Array(key, apiKeysAllowQuerySlice) {
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

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *ApiKeys) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.ApiKeys{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.ApiKeys

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

func (this *ApiKeys) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.ApiKeys{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.ApiKeys{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	data := utils.Array.MapWithField(utils.Rand.MapSlice(mold.Select()), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *ApiKeys) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *ApiKeys) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("api-keys", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	now := time.Now().Unix()
	table := model.ApiKeys{CreateTime: now, UpdateTime: now}

	for key, val := range params {
		if utils.In.Array(key, apiKeysAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(key, val))
		}
	}

	if utils.Is.Empty(table.Value) {
		table.Value = this.generateAPIKey()
	}

	if facade.DB.Model(&table).Where("value", table.Value).Exist() {
		this.json(ctx, nil, facade.Lang(ctx, "%s 已经存在！", table.Value), 400)
		return
	}

	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *ApiKeys) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("api-keys", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.ApiKeys{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, apiKeysAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(key, val))
		}
	}

	if utils.Is.Empty(async.Get("value")) {
		async.Set("value", this.generateAPIKey())
	}

	keyVal := cast.ToString(async.Get("value"))
	item := facade.DB.Model(&table).Where("value", keyVal).Find()

	if !utils.Is.Empty(item) && item["value"] != keyVal {
		this.json(ctx, nil, facade.Lang(ctx, "%s 已经存在！", keyVal), 400)
		return
	}

	tx := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *ApiKeys) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.buildQuery(facade.DB.Model(&model.ApiKeys{}), params)
	query = this.withTrashOptions(query, params)
	this.json(ctx, query.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

func (this *ApiKeys) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Sum(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *ApiKeys) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Min(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *ApiKeys) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Max(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *ApiKeys) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.ApiKeys{}), params)
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

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *ApiKeys) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.ApiKeys{})
	ids = utils.Unity.Ids(query.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := query.Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	if facade.DB.Model(&model.ApiKeys{}).Count() == 0 {
		item := facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_API_KEY")
		if cast.ToInt(item.Find()["value"]) == 1 {
			res := item.Update(map[string]any{
				"value": 0,
			})
			if res.Error == nil {
				go facade.Cache.DelTags("SYSTEM_API_KEY")
				this.json(ctx, nil, facade.Lang(ctx, "删除成功！<br>同时检测到您开启了API_KEY，但无密钥可用。<br>系统已为您自动关闭API_KEY功能！"), 200)
				return
			}
		}
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *ApiKeys) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.ApiKeys{}).WithTrashed()
	ids = utils.Unity.Ids(query.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := query.Force().Delete(ids)
	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *ApiKeys) clear(ctx *gin.Context) {
	query := facade.DB.Model(&model.ApiKeys{}).OnlyTrashed()
	ids := utils.Unity.Ids(query.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := query.Force().Delete()
	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *ApiKeys) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.ApiKeys{}).OnlyTrashed().WhereIn("id", ids)
	ids = utils.Unity.Ids(query.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.ApiKeys{}).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}
