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

type Config struct {
	base
}

const (
	configAllowFields = "key,value,remark,json,text"
	configAllowQuery  = "key"
)

var configAllowFieldsSlice = []any{"key", "value", "remark", "json", "text"}
var configAllowQuerySlice = []any{"key"}

func (this *Config) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Config) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Config) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Config) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Config) processFieldValue(val any) any {
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

func (this *Config) applyRootFilter(ctx *gin.Context, query *facade.ModelStruct) *facade.ModelStruct {
	if !this.meta.root(ctx) {
		query = query.Not("key", "LIKE", "SYSTEM_%")
	}
	return query
}

func (this *Config) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"count":  this.count,
		"column": this.column,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Config) IPOST(ctx *gin.Context) {
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

func (this *Config) IPUT(ctx *gin.Context) {
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

func (this *Config) IDEL(ctx *gin.Context) {
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

func (this *Config) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Config) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "config"})
	facade.Cache.DelTags([]any{"[GET]", "[?]"})
}

func (this *Config) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)

	if utils.Is.Empty(params["key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "key"), 400)
		return
	}

	table := model.Config{}

	for key, val := range params {
		if utils.In.Array(key, configAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.applyRootFilter(ctx, query)
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

func (this *Config) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Config{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Config

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	query = this.applyRootFilter(ctx, query)
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

func (this *Config) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "key"), 400)
		return
	}

	ok, _ := facade.DB.Model(&model.Config{}).WithTrashed().Where("key", params["key"]).Exist()
	if !ok {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Config) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("config", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.Config{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.In.Array(key, configAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{
		"id":  table.Id,
		"key": table.Key,
	}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *Config) update(ctx *gin.Context) {
	params := this.params(ctx)

	err := validator.NewValid("config", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.Config{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, configAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).WithTrashed().Where("key", params["key"]).Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.watch()

	this.json(ctx, gin.H{
		"id":  table.Id,
		"key": table.Key,
	}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Config) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Config{}), params)
	query = this.buildQuery(query, params)
	query = this.applyRootFilter(ctx, query)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Config) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Config{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	keys := utils.Unity.Keys(params["keys"])
	if !utils.Is.Empty(keys) {
		query = query.WhereIn("key", keys)
	}

	cacheName := this.cache.name(ctx)
	if !this.meta.root(ctx) {
		query = query.Not("key", "LIKE", "SYSTEM_%")
		cacheName += "&root"
	}

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

func (this *Config) remove(ctx *gin.Context) {
	params := this.params(ctx)
	keys := utils.Unity.Keys(params["keys"])

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "keys"), 400)
		return
	}

	item := facade.DB.Model(&[]model.Config{})
	columnData, _ := item.WhereIn("key", keys).Column("key")
	keys = utils.Unity.Keys(columnData)

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.WhereIn("key", keys).Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"keys": keys}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Config) delete(ctx *gin.Context) {
	params := this.params(ctx)
	keys := utils.Unity.Keys(params["keys"])

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "keys"), 400)
		return
	}

	item := facade.DB.Model(&[]model.Config{}).WithTrashed()
	columnData, _ := item.WhereIn("key", keys).Column("key")
	keys = utils.Unity.Keys(columnData)

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.WhereIn("key", keys).Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"keys": keys}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Config) clear(ctx *gin.Context) {
	table := model.Config{}
	item := facade.DB.Model(&table).OnlyTrashed()

	keys := utils.Unity.Keys(item.Column("key"))

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"keys": keys}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *Config) restore(ctx *gin.Context) {
	params := this.params(ctx)
	keys := utils.Unity.Keys(params["keys"])

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "keys"), 400)
		return
	}

	item := facade.DB.Model(&[]model.Config{}).OnlyTrashed().WhereIn("key", keys)
	columnData, _ := item.Column("key")
	keys = utils.Unity.Keys(columnData)

	if utils.Is.Empty(keys) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&[]model.Config{}).OnlyTrashed().WhereIn("key", keys).Restore()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"keys": keys}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *Config) watch() {
	item, _ := facade.DB.Model(&model.Config{}).Where("key", "SYSTEM_API_KEY").Find()
	if cast.ToInt(item["value"]) == 1 {

		ApiKeys := facade.DB.Model(&model.ApiKeys{})
		count, _ := ApiKeys.Count()
		if count == 0 {
			UUID := uuid.New().String()
			UUID = strings.Replace(UUID, "-", "", -1)
			ApiKeys.Create(&model.ApiKeys{
				Value:  strings.ToUpper(UUID),
				Remark: "检测到您开启了API_KEY功能，但无可用密钥，系统贴心的为您创建了一个密钥！",
			})
		}
	}
}
