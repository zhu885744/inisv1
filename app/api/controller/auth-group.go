package controller

import (
	"fmt"
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

type AuthGroup struct {
	base
}

const (
	authGroupAllowFields = "name,key,rules,uids,root,pages,remark,json,text"
	authGroupAllowQuery  = "id"
)

var authGroupAllowFieldsSlice = []any{"name", "key", "rules", "uids", "root", "pages", "remark", "json", "text"}
var authGroupAllowQuerySlice = []any{"id"}

func (this *AuthGroup) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *AuthGroup) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *AuthGroup) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *AuthGroup) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *AuthGroup) processFieldValue(val any) any {
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

func (this *AuthGroup) anyToIntSlice(idsAny []any) []int {
	ids := make([]int, 0, len(idsAny))
	for _, id := range idsAny {
		ids = append(ids, cast.ToInt(id))
	}
	return ids
}

func (this *AuthGroup) isSystemAdminGroup(ids []int) (bool, []int) {
	systemGroupIds, _ := facade.DB.Model(&model.AuthGroup{}).
		Where("id", "=", 1).
		WhereIn("id", ids).
		Column("id")

	systemIdsAny := utils.Unity.Ids(systemGroupIds)
	systemIds := this.anyToIntSlice(systemIdsAny)

	if len(systemIds) > 0 {
		return true, systemIds
	}
	return false, nil
}

func (this *AuthGroup) IGET(ctx *gin.Context) {
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

func (this *AuthGroup) IPOST(ctx *gin.Context) {
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

func (this *AuthGroup) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
		"uids":    this.uids,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *AuthGroup) IDEL(ctx *gin.Context) {
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

func (this *AuthGroup) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *AuthGroup) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "auth-group"})
}

func (this *AuthGroup) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.AuthGroup{}

	for key, val := range params {
		if utils.In.Array(key, authGroupAllowQuerySlice) {
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

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *AuthGroup) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.AuthGroup{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.AuthGroup

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
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

func (this *AuthGroup) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.AuthGroup{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.AuthGroup{}).Where("id", "IN", ids)
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

func (this *AuthGroup) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *AuthGroup) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("auth-group", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.AuthGroup{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.In.Array(key, authGroupAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *AuthGroup) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("auth-group", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.AuthGroup{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, authGroupAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *AuthGroup) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.AuthGroup{}), params)
	query = this.buildQuery(query, params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *AuthGroup) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.AuthGroup{}), params)
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

func (this *AuthGroup) sum(ctx *gin.Context) {
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

func (this *AuthGroup) min(ctx *gin.Context) {
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

func (this *AuthGroup) max(ctx *gin.Context) {
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

func (this *AuthGroup) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.AuthGroup{}), params)
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

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *AuthGroup) remove(ctx *gin.Context) {
	params := this.params(ctx)
	idsAny := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(idsAny) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	ids := this.anyToIntSlice(idsAny)

	isSys, sysIds := this.isSystemAdminGroup(ids)
	if isSys {
		this.json(ctx, nil, facade.Lang(ctx, "禁止删除系统管理员分组！", sysIds), 403)
		return
	}

	item := facade.DB.Model(&[]model.AuthGroup{}).Where("default", "!=", 1)

	columnData, _ := item.WhereIn("id", ids).Column("id")
	allowIdsAny := utils.Unity.Ids(columnData)
	allowIds := this.anyToIntSlice(allowIdsAny)

	if utils.Is.Empty(allowIds) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Delete(allowIds)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": allowIds}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *AuthGroup) delete(ctx *gin.Context) {
	params := this.params(ctx)
	idsAny := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(idsAny) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	ids := this.anyToIntSlice(idsAny)

	isSys, sysIds := this.isSystemAdminGroup(ids)
	if isSys {
		this.json(ctx, nil, facade.Lang(ctx, "禁止删除系统管理员分组！", sysIds), 403)
		return
	}

	item := facade.DB.Model(&[]model.AuthGroup{}).WithTrashed().Where("default", "!=", 1)

	allowIdsAny := utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))
	allowIds := this.anyToIntSlice(allowIdsAny)

	if utils.Is.Empty(allowIds) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Force().Delete(allowIds)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": allowIds}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *AuthGroup) clear(ctx *gin.Context) {
	table := model.AuthGroup{}
	item := facade.DB.Model(&table).OnlyTrashed()

	columnData, _ := item.Column("id")
	idsAny := utils.Unity.Ids(columnData)
	ids := this.anyToIntSlice(idsAny)

	isSys, sysIds := this.isSystemAdminGroup(ids)
	if isSys {
		this.json(ctx, nil, facade.Lang(ctx, "禁止清空系统管理员分组 %v ！", sysIds), 403)
		return
	}

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.WhereIn("id", ids).Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *AuthGroup) restore(ctx *gin.Context) {
	params := this.params(ctx)
	idsAny := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(idsAny) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	ids := this.anyToIntSlice(idsAny)
	item := facade.DB.Model(&model.AuthGroup{}).OnlyTrashed().WhereIn("id", ids)

	columnData, _ := item.Column("id")
	allowIdsAny := utils.Unity.Ids(columnData)
	allowIds := this.anyToIntSlice(allowIdsAny)

	if utils.Is.Empty(allowIds) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.AuthGroup{}).OnlyTrashed().Restore(allowIds)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": allowIds}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *AuthGroup) uids(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	go func() {
		ids := utils.Unity.Ids(params["ids"])

		cull := facade.DB.Model(&[]model.AuthGroup{}).WithTrashed()

		if !utils.Is.Empty(ids) {
			cull.Where("id", "not in", ids)
		}

		cullItems, _ := cull.Select()
		for _, item := range cullItems {
			uids := cast.ToIntSlice(utils.ArrayUnique(utils.ArrayEmpty(strings.Split(cast.ToString(item["uids"]), "|"))))
			if utils.InArray[int](cast.ToInt(params["uid"]), uids) {
				for key, val := range uids {
					if val == cast.ToInt(params["uid"]) {
						uids = append(uids[:key], uids[key+1:]...)
					}
				}
			}
			var result string
			if len(uids) > 0 {
				result = fmt.Sprintf("|%v|", strings.Join(cast.ToStringSlice(uids), "|"))
			}
			facade.DB.Model(&model.AuthGroup{}).WithTrashed().Where("id", item["id"]).Update(map[string]any{
				"uids": result,
			})
		}

		addItems, _ := facade.DB.Model(&[]model.AuthGroup{}).WithTrashed().Where("id", "in", ids).Select()

		for _, item := range addItems {
			uids := strings.Split(cast.ToString(item["uids"]), "|")
			uids = append(uids, cast.ToString(params["uid"]))
			var result string
			if len(uids) > 0 {
				result = fmt.Sprintf("|%v|", strings.Join(cast.ToStringSlice(utils.ArrayUnique(utils.ArrayEmpty(uids))), "|"))
			}
			facade.DB.Model(&model.AuthGroup{}).WithTrashed().Where("id", item["id"]).Update(map[string]any{
				"uids": result,
			})
		}
	}()

	go func() {
		facade.Cache.DelTags(fmt.Sprintf("user[%v]", params["uid"]))
	}()

	this.json(ctx, nil, facade.Lang(ctx, "更新成功！"), 200)
}
