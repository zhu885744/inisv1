package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type UserCollects struct {
	base
}

const (
	userCollectsAllowFields = "target_type,target_id"
	userCollectsAllowQuery  = "id,uid,target_type,target_id"
)

var userCollectsAllowFieldsSlice = []any{"target_type", "target_id"}
var userCollectsAllowQuerySlice = []any{"id", "uid", "target_type", "target_id"}

func (this *UserCollects) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *UserCollects) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *UserCollects) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *UserCollects) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":          this.one,
		"all":          this.all,
		"sum":          this.sum,
		"min":          this.min,
		"max":          this.max,
		"rand":         this.rand,
		"count":        this.count,
		"column":       this.column,
		"is-collected": this.isCollected,
		"collects":     this.collects,
		"counts":       this.counts,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *UserCollects) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":      this.save,
		"create":    this.create,
		"collect":   this.collect,
		"uncollect": this.uncollect,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserCollects) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":    this.update,
		"uncollect": this.uncollect,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserCollects) IDEL(ctx *gin.Context) {
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

func (this *UserCollects) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *UserCollects) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "user-collects"})
}

func (this *UserCollects) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.UserCollects{}

	for key, val := range params {
		if utils.In.Array(key, userCollectsAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := facade.DB.Model(&table)
		query = this.buildQuery(query, params)

		if !this.meta.root(ctx) {
			query = query.Where("uid", this.user(ctx).Id)
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

func (this *UserCollects) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "created_at desc",
	})

	table := model.UserCollects{}
	for key, val := range params {
		if utils.In.Array(key, userCollectsAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.UserCollects

	query := facade.DB.Model(&result)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
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

func (this *UserCollects) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])

	query := facade.DB.Model(&model.UserCollects{})

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.UserCollects{}).Where("id", "IN", ids)
	mold = this.buildQuery(mold, params)

	if !this.meta.root(ctx) {
		mold = mold.Where("uid", this.user(ctx).Id)
	}

	items, _ := mold.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *UserCollects) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *UserCollects) create(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}

	if utils.Is.Empty(params["target_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	targetType := cast.ToString(params["target_type"])
	if targetType == "user" {
		this.json(ctx, nil, facade.Lang(ctx, "不支持收藏用户！"), 400)
		return
	}

	table := model.UserCollects{Uid: uid}
	for key, val := range params {
		if utils.In.Array(key, userCollectsAllowFieldsSlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	_, err := facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *UserCollects) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.UserCollects{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, userCollectsAllowFieldsSlice) {
			async.Set(key, val)
		}
	}

	item := facade.DB.Model(&table).Where("id", params["id"])

	findResult, _ := item.Find()
	if !this.meta.root(ctx) && cast.ToInt(findResult["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	_, err := item.Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *UserCollects) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := facade.DB.Model(&model.UserCollects{})
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserCollects) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := facade.DB.Model(&model.UserCollects{})
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
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

func (this *UserCollects) sum(ctx *gin.Context) {
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

func (this *UserCollects) min(ctx *gin.Context) {
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

func (this *UserCollects) max(ctx *gin.Context) {
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

func (this *UserCollects) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := facade.DB.Model(&[]model.UserCollects{})
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
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

func (this *UserCollects) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserCollects{})
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

func (this *UserCollects) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserCollects{})
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

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

func (this *UserCollects) clear(ctx *gin.Context) {
	table := model.UserCollects{}
	item := facade.DB.Model(&table)

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := item.Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *UserCollects) collect(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}

	if utils.Is.Empty(params["target_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	targetType := cast.ToString(params["target_type"])
	targetId := cast.ToInt(params["target_id"])

	err := (&model.UserCollects{}).Collect(uid, targetId, targetType)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "收藏经验值协程发生错误")
			}
		}()

		var authorId int

		switch targetType {
		case "article":
			article, _ := facade.DB.Model(&model.Article{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(article)["uid"])
		case "page":
			page, _ := facade.DB.Model(&model.Pages{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(page)["uid"])
		case "moments":
			moment, _ := facade.DB.Model(&model.Moments{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(moment)["uid"])
		default:
			return
		}

		if authorId > 0 && authorId != uid {
			_ = (&model.EXP{}).Add(model.EXP{
				Uid:      authorId,
				Type:     "article-collect",
				BindType: targetType,
				BindId:   targetId,
			})
		}
	}()

	// 创建收藏通知
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "创建收藏通知协程发生错误")
			}
		}()

		var authorId int
		var title, content string

		switch targetType {
		case "article":
			item, _ := facade.DB.Model(&model.Article{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			title = cast.ToString(itemMap["title"])
		case "page":
			item, _ := facade.DB.Model(&model.Pages{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			title = cast.ToString(itemMap["title"])
		case "moments":
			item, _ := facade.DB.Model(&model.Moments{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			content = cast.ToString(itemMap["content"])
		default:
			return
		}

		if authorId <= 0 || authorId == uid {
			return
		}

		userInfo, _ := facade.DB.Model(&model.Users{}).Find(uid)
		fromNickname := cast.ToString(cast.ToStringMap(userInfo)["nickname"])

		var notifTitle, notifContent string
		switch targetType {
		case "article":
			if !utils.Is.Empty(title) {
				title = truncateSafe(title, 20)
				notifTitle = "文章被收藏"
				notifContent = fromNickname + " 收藏了你的文章「" + title + "」"
			} else {
				notifTitle = "获得新收藏"
				notifContent = fromNickname + " 收藏了你的文章"
			}
		case "page":
			if !utils.Is.Empty(title) {
				title = truncateSafe(title, 20)
				notifTitle = "页面被收藏"
				notifContent = fromNickname + " 收藏了你的页面「" + title + "」"
			} else {
				notifTitle = "获得新收藏"
				notifContent = fromNickname + " 收藏了你的页面"
			}
		case "moments":
			if !utils.Is.Empty(content) {
				content = truncateSafe(content, 30)
				notifTitle = "动态被收藏"
				notifContent = fromNickname + " 收藏了你的动态「" + content + "」"
			} else {
				notifTitle = "获得新收藏"
				notifContent = fromNickname + " 收藏了你的动态"
			}
		}

		_, _ = (&model.Notification{}).CreateNotification(
			authorId, uid, "collect", notifTitle, notifContent, targetType, targetId,
		)
	}()

	this.json(ctx, gin.H{
		"target_type": targetType,
		"target_id":   targetId,
	}, facade.Lang(ctx, "收藏成功！"), 200)
}

func (this *UserCollects) uncollect(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}

	if utils.Is.Empty(params["target_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	targetType := cast.ToString(params["target_type"])
	targetId := cast.ToInt(params["target_id"])

	err := (&model.UserCollects{}).Uncollect(uid, targetId, targetType)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{
		"target_type": targetType,
		"target_id":   targetId,
	}, facade.Lang(ctx, "取消收藏成功！"), 200)
}

func (this *UserCollects) isCollected(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}

	if utils.Is.Empty(params["target_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	targetType := cast.ToString(params["target_type"])
	targetId := cast.ToInt(params["target_id"])

	uid := this.meta.user(ctx).Id
	isCollected := false
	if uid > 0 {
		isCollected = (&model.UserCollects{}).IsCollected(uid, targetId, targetType)
	}
	count := (&model.UserCollects{}).GetCollectsCount(targetId, targetType)

	this.json(ctx, gin.H{
		"is_collected": isCollected,
		"count":        count,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserCollects) collects(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetType := cast.ToString(params["target_type"])
	data, count := (&model.UserCollects{}).GetCollectsByUid(uid, targetType)

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, gin.H{"list": utils.ArrayMapWithField(data, params["field"]), "count": count}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserCollects) counts(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}

	if utils.Is.Empty(params["target_ids"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_ids"), 400)
		return
	}

	targetType := cast.ToString(params["target_type"])
	targetIds := cast.ToIntSlice(params["target_ids"])

	var counts map[int]int64

	if targetType == "user_collects" {
		counts = make(map[int]int64)
		for _, uid := range targetIds {
			counts[uid] = (&model.UserCollects{}).GetUserCollectsCount(uid)
		}
	} else {
		counts = (&model.UserCollects{}).GetCollectsCounts(targetType, targetIds)
	}

	this.json(ctx, gin.H{"counts": counts}, facade.Lang(ctx, "查询成功！"), 200)
}
