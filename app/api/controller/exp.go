package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type EXP struct {
	base
}

const (
	expAllowFields = "value,type,description,json,text"
	expAllowQuery  = "id"
)

var expAllowFieldsSlice = []any{"value", "type", "description", "json", "text"}
var expAllowQuerySlice = []any{"id"}

func (this *EXP) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *EXP) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *EXP) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *EXP) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *EXP) processFieldValue(val any) any {
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

func (this *EXP) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.EXP{}), params)
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

func (this *EXP) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":             this.one,
		"all":             this.all,
		"sum":             this.sum,
		"min":             this.min,
		"max":             this.max,
		"rand":            this.rand,
		"count":           this.count,
		"column":          this.column,
		"active":          this.active,
		"check-in-status": this.checkInStatus,
		"check-in-rank":   this.checkInRank,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *EXP) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":     this.save,
		"create":   this.create,
		"like":     this.like,
		"share":    this.share,
		"collect":  this.collect,
		"check-in": this.checkIn,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *EXP) IPUT(ctx *gin.Context) {
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

func (this *EXP) IDEL(ctx *gin.Context) {
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

func (this *EXP) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *EXP) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "exp"})
}

func (this *EXP) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.EXP{}

	for key, val := range params {
		if utils.In.Array(key, expAllowQuerySlice) {
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

func (this *EXP) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.EXP{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.EXP

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

func (this *EXP) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.EXP{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.EXP{}).Where("id", "IN", ids)
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

func (this *EXP) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *EXP) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("exp", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.EXP{Uid: uid, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.In.Array(key, expAllowFieldsSlice) {
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

func (this *EXP) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("exp", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.EXP{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, expAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	itemData, _ := item.Find()
	if !this.meta.root(ctx) && cast.ToInt(itemData["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	_, err = item.Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *EXP) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := facade.DB.Model(&model.EXP{})
	query = this.buildQuery(query, params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *EXP) sum(ctx *gin.Context) {
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

func (this *EXP) min(ctx *gin.Context) {
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

func (this *EXP) max(ctx *gin.Context) {
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

func (this *EXP) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.EXP{}), params)
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

func (this *EXP) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.EXP{})

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

func (this *EXP) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.EXP{}).WithTrashed()

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

func (this *EXP) clear(ctx *gin.Context) {
	table := model.EXP{}
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

func (this *EXP) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.EXP{}).OnlyTrashed().WhereIn("id", ids)

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.EXP{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *EXP) checkInStatus(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	checked, _ := facade.DB.Model(&model.EXP{}).Where([]any{
		[]any{"uid", "=", user.Id},
		[]any{"type", "=", "check-in"},
		[]any{"create_time", ">=", today.Unix()},
	}).Exist()

	var value int
	var checkInTime int64

	if checked {
		item, _ := facade.DB.Model(&model.EXP{}).Where([]any{
			[]any{"uid", "=", user.Id},
			[]any{"type", "=", "check-in"},
			[]any{"create_time", ">=", today.Unix()},
		}).Order("create_time desc").Find()
		value = cast.ToInt(item["value"])
		checkInTime = cast.ToInt64(item["create_time"])
	}

	streak := 0
	if checked {
		for i := 0; i < 365; i++ {
			dayStart := today.AddDate(0, 0, -i)
			dayEnd := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
			has, _ := facade.DB.Model(&model.EXP{}).Where([]any{
				[]any{"uid", "=", user.Id},
				[]any{"type", "=", "check-in"},
				[]any{"create_time", ">=", dayStart.Unix()},
				[]any{"create_time", "<=", dayEnd.Unix()},
			}).Exist()
			if has {
				streak++
			} else {
				break
			}
		}
	}

	this.json(ctx, gin.H{
		"checked":       checked,
		"value":         value,
		"check_in_time": checkInTime,
		"streak":        streak,
		"today":         today.Unix(),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *EXP) checkInRank(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)

	now := time.Now()
	year, month, _ := now.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	if !utils.Is.Empty(params["start"]) {
		start = time.Unix(cast.ToInt64(params["start"]), 0)
	}
	if !utils.Is.Empty(params["end"]) {
		end = time.Unix(cast.ToInt64(params["end"]), 0)
	}

	var table []model.EXP

	sql := "SELECT uid, COUNT(id) as check_in_count, SUM(value) AS total_exp FROM inis_exp WHERE type = 'check-in' AND create_time >= ? AND create_time <= ? GROUP BY uid ORDER BY check_in_count DESC, total_exp DESC LIMIT ?"
	total, _ := facade.DB.Model(&table).Query(sql, start.Unix(), end.Unix(), this.meta.limit(ctx)).Column("uid", "check_in_count", "total_exp")
	list := cast.ToSlice(total)

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		result := make([]any, len(list))

		wg := sync.WaitGroup{}

		for key, val := range list {
			wg.Add(1)
			go func(key int, val any) {
				defer wg.Done()
				value := cast.ToStringMap(val)
				field := []string{"id", "nickname", "avatar", "description", "title", "gender", "result"}
				author, _ := facade.DB.Model(&model.Users{}).Where("id", value["uid"]).Find()
				item := facade.Comm.WithField(author, field)
				item["check_in_count"] = cast.ToInt(value["check_in_count"])
				item["total_exp"] = cast.ToInt(value["total_exp"])
				item["rank"] = key + 1
				result[key] = item
			}(key, val)
		}

		wg.Wait()

		data = result
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *EXP) checkIn(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	err := (&model.EXP{}).Add(model.EXP{
		Uid:  user.Id,
		Type: "check-in",
	})

	if err != nil {
		this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"value": 10}, facade.Lang(ctx, "签到成功！"), 200)
}

func (this *EXP) share(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"bind_type": "article",
	})

	allow := []any{"article", "page", "moments"}

	if !utils.In.Array(params["bind_type"], allow) {
		this.json(ctx, nil, facade.Lang(ctx, "不存在的分享类型！"), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	switch params["bind_type"] {
	case "article":
		exist, _ := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
	case "page":
		exist, _ := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
	case "moments":
		exist, _ := facade.DB.Model(&model.Moments{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的动态！"), 400)
			return
		}
	}

	err := (&model.EXP{}).Add(model.EXP{
		Type:        "share",
		Uid:         user.Id,
		BindId:      cast.ToInt(params["bind_id"]),
		BindType:    cast.ToString(params["bind_type"]),
		Description: cast.ToString(params["description"]),
	})

	if err != nil {
		this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"value": 1}, facade.Lang(ctx, "分享成功！"), 200)
}

func (this *EXP) collect(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"state":     1,
		"bind_type": "article",
	})

	if !utils.InArray(cast.ToInt(params["state"]), []int{0, 1}) {
		this.json(ctx, nil, facade.Lang(ctx, "state 只能是 0 或 1"), 400)
		return
	}

	allow := []any{"article", "page", "moments"}

	if !utils.In.Array(params["bind_type"], allow) {
		this.json(ctx, nil, facade.Lang(ctx, "不存在的收藏类型！"), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	switch params["bind_type"] {
	case "article":
		exist, _ := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
	case "page":
		exist, _ := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
	case "moments":
		exist, _ := facade.DB.Model(&model.Moments{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的动态！"), 400)
			return
		}
	}

	item, _ := facade.DB.Model(&model.EXP{}).Where([]any{
		[]any{"uid", "=", user.Id},
		[]any{"type", "=", "collect"},
		[]any{"bind_id", "=", params["bind_id"]},
		[]any{"bind_type", "=", params["bind_type"]},
	}).Find()

	if !utils.Is.Empty(item) {
		if cast.ToInt(params["state"]) == 0 {
			_, err := facade.DB.Model(&model.EXP{}).Where(item["id"]).Update(map[string]any{
				"state": 0,
			})
			if err != nil {
				this.json(ctx, nil, err.Error(), 400)
				return
			}
			this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "取消收藏成功！"), 200)
			return
		}

		if cast.ToInt(params["state"]) == 1 && cast.ToInt(item["state"]) == 1 {
			this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "已经收藏过了！"), 400)
			return
		}

		_, err := facade.DB.Model(&model.EXP{}).Where(item["id"]).Update(map[string]any{
			"state": 1,
		})
		if err != nil {
			this.json(ctx, gin.H{"value": 0}, err.Error(), 400)
			return
		}

		this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "收藏成功！"), 200)
		return
	}

	if cast.ToInt(params["state"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "您未收藏该内容！"), 400)
		return
	}

	err := (&model.EXP{}).Add(model.EXP{
		Type:        "collect",
		Uid:         user.Id,
		BindId:      cast.ToInt(params["bind_id"]),
		BindType:    cast.ToString(params["bind_type"]),
		Description: cast.ToString(params["description"]),
	})

	if err != nil {
		this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"value": 1}, facade.Lang(ctx, "收藏成功！"), 200)
}

func (this *EXP) like(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"state":     1,
		"bind_type": "article",
	})

	if !utils.InArray(cast.ToInt(params["state"]), []int{0, 1}) {
		this.json(ctx, nil, facade.Lang(ctx, "state 只能是 0 或 1"), 400)
		return
	}

	allow := []any{"article", "page", "comment", "moments"}

	if !utils.In.Array(params["bind_type"], allow) {
		this.json(ctx, nil, facade.Lang(ctx, "不存在的点赞类型！"), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	switch params["bind_type"] {
	case "article":
		exist, _ := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
	case "page":
		exist, _ := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
	case "comment":
		exist, _ := facade.DB.Model(&model.Comment{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的评论！"), 400)
			return
		}
	case "moments":
		exist, _ := facade.DB.Model(&model.Moments{}).Where("id", params["bind_id"]).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的动态！"), 400)
			return
		}
	}

	item, _ := facade.DB.Model(&model.EXP{}).Where([]any{
		[]any{"uid", "=", user.Id},
		[]any{"type", "=", "like"},
		[]any{"bind_id", "=", params["bind_id"]},
		[]any{"bind_type", "=", params["bind_type"]},
	}).Find()

	if !utils.Is.Empty(item) {
		if cast.ToInt(params["state"]) == 0 {
			_, err := facade.DB.Model(&model.EXP{}).Where(item["id"]).Update(map[string]any{
				"state": 0,
			})
			if err != nil {
				this.json(ctx, nil, err.Error(), 400)
				return
			}
			this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "点踩成功！"), 200)
			return
		}

		if cast.ToInt(params["state"]) == 1 && cast.ToInt(item["state"]) == 1 {
			this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "已经点过赞啦！"), 400)
			return
		}

		_, err := facade.DB.Model(&model.EXP{}).Where(item["id"]).Update(map[string]any{
			"state": 1,
		})
		if err != nil {
			this.json(ctx, gin.H{"value": 0}, err.Error(), 400)
			return
		}

		this.json(ctx, gin.H{"value": 0}, facade.Lang(ctx, "点赞成功！"), 200)
		return
	}

	msg := "点赞"
	if cast.ToInt(params["state"]) == 0 {
		msg = "点踩"
	}

	err := (&model.EXP{}).Add(model.EXP{
		Type:        "like",
		Uid:         user.Id,
		State:       cast.ToInt(params["state"]),
		BindId:      cast.ToInt(params["bind_id"]),
		BindType:    cast.ToString(params["bind_type"]),
		Description: utils.Default(cast.ToString(params["description"]), msg+"奖励"),
	})

	if err != nil {
		this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"value": 1}, facade.Lang(ctx, msg+"成功！"), 200)
}

func (this *EXP) active(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)

	now := time.Now()
	year, month, _ := now.Date()
	start := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	if utils.Is.Empty(params["start"]) {
		params["start"] = start.Unix()
	}
	if utils.Is.Empty(params["end"]) {
		params["end"] = end.Unix()
	}

	var table []model.EXP

	sql := "SELECT uid, SUM(value) AS total, COUNT(id) as number FROM inis_exp WHERE create_time >= ? AND create_time <= ? GROUP BY uid ORDER BY SUM(value) DESC LIMIT ?"
	total, _ := facade.DB.Model(&table).Query(sql, params["start"], params["end"], this.meta.limit(ctx)).Column("uid", "total", "number")
	list := cast.ToSlice(total)

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		result := make([]any, len(list))

		wg := sync.WaitGroup{}

		for key, val := range list {
			wg.Add(1)
			go func(key int, val any) {
				defer wg.Done()
				value := cast.ToStringMap(val)
				field := []string{"id", "nickname", "avatar", "description", "login_time", "title", "gender", "result"}
				author, _ := facade.DB.Model(&model.Users{}).Where("id", value["uid"]).Find()
				item := facade.Comm.WithField(author, field)
				item["exp"] = cast.ToInt(value["total"])
				item["count"] = value["number"]
				result[key] = item
			}(key, val)
		}

		wg.Wait()

		data = result
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}
