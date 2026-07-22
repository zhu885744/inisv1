package controller

import (
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

type IpBlack struct {
	base
}

const (
	ipBlackAllowFields = "ip,cause,remark,json,text,level,duration,expire_time,is_permanent,violation_count"
	ipBlackAllowQuery  = "id"
)

var ipBlackAllowFieldsSlice = []any{"ip", "cause", "remark", "json", "text", "level", "duration", "expire_time", "is_permanent", "violation_count"}
var ipBlackAllowQuerySlice = []any{"id"}

func (this *IpBlack) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *IpBlack) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *IpBlack) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *IpBlack) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *IpBlack) processFieldValue(val any) any {
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

func (this *IpBlack) maskIPData(ctx *gin.Context, data any) any {
	if data == nil || this.meta.root(ctx) {
		return data
	}
	switch v := data.(type) {
	case map[string]any:
		if ip, ok := v["ip"]; ok {
			v["ip"] = facade.Comm.MaskIP(cast.ToString(ip))
		}
		return v
	case []map[string]any:
		for i, item := range v {
			if ip, ok := item["ip"]; ok {
				v[i]["ip"] = facade.Comm.MaskIP(cast.ToString(ip))
			}
		}
		return v
	case []any:
		for i, item := range v {
			if m, ok := item.(map[string]any); ok {
				if ip, ok2 := m["ip"]; ok2 {
					m["ip"] = facade.Comm.MaskIP(cast.ToString(ip))
				}
				v[i] = m
			}
		}
		return v
	}
	return data
}

func (this *IpBlack) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.IpBlack{}), params)
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

func (this *IpBlack) IGET(ctx *gin.Context) {
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

func (this *IpBlack) IPOST(ctx *gin.Context) {
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

func (this *IpBlack) IPUT(ctx *gin.Context) {
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

func (this *IpBlack) IDEL(ctx *gin.Context) {
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

func (this *IpBlack) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *IpBlack) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "ip-black"})
}

func (this *IpBlack) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.IpBlack{}

	for key, val := range params {
		if utils.In.Array(key, ipBlackAllowQuerySlice) {
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

	this.json(ctx, this.maskIPData(ctx, data), facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *IpBlack) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.IpBlack{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.IpBlack

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
		"data":  this.maskIPData(ctx, data),
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *IpBlack) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.IpBlack{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.IpBlack{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	items, _ := mold.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, this.maskIPData(ctx, data), facade.Lang(ctx, "好的！"), 200)
}

func (this *IpBlack) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *IpBlack) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("ip-black", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.IpBlack{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	// 设置默认封禁参数（如果用户未指定）
	// 默认 level = 1，1 小时封禁；如果指定了 is_permanent = true，则永久封禁
	userLevel := cast.ToInt(params["level"])
	userIsPermanent := cast.ToBool(params["is_permanent"])
	userDuration := cast.ToInt64(params["duration"])
	userExpireTime := cast.ToInt64(params["expire_time"])

	if userLevel == 0 && !userIsPermanent {
		userLevel = 1 // 默认1级（1小时）
	}

	if userLevel > 0 {
		table.Level = userLevel
	}
	if userIsPermanent {
		table.IsPermanent = true
		table.Duration = 0
		table.ExpireTime = 0
	} else {
		table.IsPermanent = false
		// 计算封禁时长
		if userDuration > 0 {
			table.Duration = userDuration
		} else if userLevel > 0 {
			// 根据等级设置默认时长
			switch userLevel {
			case 1:
				table.Duration = 1
			case 2:
				table.Duration = 24
			case 3:
				table.Duration = 24 * 7
			default:
				table.Duration = 1
			}
		}
		// 计算解封时间
		if userExpireTime > 0 {
			table.ExpireTime = userExpireTime
		} else if table.Duration > 0 {
			table.ExpireTime = time.Now().Unix() + table.Duration*3600
		}
	}

	for key, val := range params {
		if utils.In.Array(key, ipBlackAllowFieldsSlice) {
			if key == "ip" {
				val = strings.ToUpper(cast.ToString(val))
			}
			// 已经处理过这些字段了，跳过
			if key == "level" || key == "duration" || key == "expire_time" || key == "is_permanent" {
				continue
			}
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

func (this *IpBlack) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("ip-black", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.IpBlack{}
	async := utils.Async[map[string]any]()

	// 先查询现有记录，用于计算新的封禁参数
	facade.DB.Model(&model.IpBlack{}).WithTrashed().Where("id", params["id"]).Scan(&table)

	// 处理封禁等级和期限参数
	userLevel := cast.ToInt(params["level"])
	userIsPermanent := cast.ToBool(params["is_permanent"])
	userDuration := cast.ToInt64(params["duration"])
	userExpireTime := cast.ToInt64(params["expire_time"])

	// 如果提交了 level 或 is_permanent，则重新计算
	if userLevel > 0 || userIsPermanent {
		newLevel := userLevel
		if userLevel == 0 {
			newLevel = table.Level
		}
		if userLevel > 0 {
			async.Set("level", newLevel)
		}

		if userIsPermanent {
			async.Set("is_permanent", true)
			async.Set("duration", 0)
			async.Set("expire_time", 0)
		} else {
			async.Set("is_permanent", false)
			// 计算封禁时长
			if userDuration > 0 {
				async.Set("duration", userDuration)
			} else if newLevel > 0 {
				// 根据等级设置默认时长
				var duration int64
				switch newLevel {
				case 1:
					duration = 1
				case 2:
					duration = 24
				case 3:
					duration = 24 * 7
				default:
					duration = 1
				}
				async.Set("duration", duration)
			} else if table.Duration > 0 {
				async.Set("duration", table.Duration)
			}

			// 计算解封时间
			if userExpireTime > 0 {
				async.Set("expire_time", userExpireTime)
			} else {
				// 从当前时间开始计算
				var duration int64
				if userDuration > 0 {
					duration = userDuration
				} else if newLevel > 0 {
					switch newLevel {
					case 1:
						duration = 1
					case 2:
						duration = 24
					case 3:
						duration = 24 * 7
					default:
						duration = 1
					}
				} else if table.Duration > 0 {
					duration = table.Duration
				}
				if duration > 0 {
					async.Set("expire_time", time.Now().Unix()+duration*3600)
				}
			}
		}
	} else if userDuration > 0 || userExpireTime > 0 {
		// 只修改了时长或过期时间，未修改等级
		if userDuration > 0 {
			async.Set("duration", userDuration)
		}
		if userExpireTime > 0 {
			async.Set("expire_time", userExpireTime)
		} else if userDuration > 0 {
			async.Set("expire_time", time.Now().Unix()+userDuration*3600)
		}
		async.Set("is_permanent", false)
	}

	// 处理其他字段
	for key, val := range params {
		if utils.In.Array(key, ipBlackAllowFieldsSlice) {
			// 这些字段已经在上面处理过了
			if key == "level" || key == "duration" || key == "expire_time" || key == "is_permanent" {
				continue
			}
			if key == "ip" {
				val = strings.ToUpper(cast.ToString(val))
			}
			async.Set(key, this.processFieldValue(val))
		}
	}

	_, err = facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *IpBlack) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.IpBlack{}), params)
	query = this.buildQuery(query, params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *IpBlack) sum(ctx *gin.Context) {
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

func (this *IpBlack) min(ctx *gin.Context) {
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

func (this *IpBlack) max(ctx *gin.Context) {
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

func (this *IpBlack) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.IpBlack{}), params)
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

	this.json(ctx, this.maskIPData(ctx, data), facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *IpBlack) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.IpBlack{})
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

func (this *IpBlack) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.IpBlack{}).WithTrashed()
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

func (this *IpBlack) clear(ctx *gin.Context) {
	table := model.IpBlack{}
	item := facade.DB.Model(&table).OnlyTrashed()

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

func (this *IpBlack) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.IpBlack{}).OnlyTrashed().WhereIn("id", ids)
	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.IpBlack{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}
