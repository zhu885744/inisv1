package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"
)

type QpsWarn struct {
	base
}

const (
	qpsWarnAllowFields = "ip,json,text"
	qpsWarnAllowQuery  = "id"
)

var qpsWarnAllowFieldsSlice = []any{"ip", "json", "text"}
var qpsWarnAllowQuerySlice = []any{"id"}

func (this *QpsWarn) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *QpsWarn) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *QpsWarn) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *QpsWarn) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *QpsWarn) processFieldValue(val any) any {
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

func (this *QpsWarn) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.QpsWarn{}), params)
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

// IGET - 获取QPS告警数据
func (this *QpsWarn) IGET(ctx *gin.Context) {
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

// IPOST - 创建/保存QPS告警
func (this *QpsWarn) IPOST(ctx *gin.Context) {
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
	go this.delCache() // 新增删除缓存方法
}

// IPUT - 更新/恢复QPS告警数据
func (this *QpsWarn) IPUT(ctx *gin.Context) {
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

// IDEL - 删除QPS告警数据
func (this *QpsWarn) IDEL(ctx *gin.Context) {
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

func (this *QpsWarn) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// 删除缓存
func (this *QpsWarn) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "qps-warn"})
}

// one 获取指定数据
func (this *QpsWarn) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.QpsWarn{}

	for key, val := range params {
		if utils.In.Array(key, qpsWarnAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		mold := this.withTrashOptions(facade.DB.Model(&table), params)
		mold = this.buildQuery(mold, params)
		item := mold.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all 获取全部数据
func (this *QpsWarn) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.QpsWarn{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.QpsWarn

	mold := this.withTrashOptions(facade.DB.Model(&result), params)
	mold = this.buildQuery(mold, params)
	count := mold.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
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

// rand 随机获取
func (this *QpsWarn) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	item := facade.DB.Model(&model.QpsWarn{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		item = item.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(item.Column("id")), limit)

	mold := facade.DB.Model(&[]model.QpsWarn{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	data := utils.Array.MapWithField(utils.Rand.MapSlice(mold.Select()), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

// save 保存数据 - 包含创建和更新
func (this *QpsWarn) save(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create 创建数据
func (this *QpsWarn) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("qps-warn", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.QpsWarn{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.In.Array(key, qpsWarnAllowFieldsSlice) {
			if key == "value" {
				val = strings.ToUpper(cast.ToString(val))
			}
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新数据
func (this *QpsWarn) update(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("qps-warn", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.QpsWarn{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, qpsWarnAllowFieldsSlice) {
			if key == "value" {
				val = strings.ToUpper(cast.ToString(val))
			}
			async.Set(key, this.processFieldValue(val))
		}
	}

	tx := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// count 统计数据
func (this *QpsWarn) count(ctx *gin.Context) {
	params := this.params(ctx)
	item := facade.DB.Model(&model.QpsWarn{})
	item = this.buildQuery(item, params)
	this.json(ctx, item.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

// sum 求和
func (this *QpsWarn) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Sum(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

// min 求最小值
func (this *QpsWarn) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Min(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

// max 求最大值
func (this *QpsWarn) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Max(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

// column 获取单列数据
func (this *QpsWarn) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.QpsWarn{}), params)
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

// remove 软删除
func (this *QpsWarn) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.QpsWarn{})
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

// delete 真实删除
func (this *QpsWarn) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.QpsWarn{}).WithTrashed()
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

// clear 清空回收站
func (this *QpsWarn) clear(ctx *gin.Context) {
	table := model.QpsWarn{}
	item := facade.DB.Model(&table).OnlyTrashed()

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

// restore 恢复数据
func (this *QpsWarn) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.QpsWarn{}).OnlyTrashed().WhereIn("id", ids)
	ids = utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.QpsWarn{}).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}
