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

type UserLikes struct {
	base
}

const (
	userLikesAllowFields = "target_type,target_id"
	userLikesAllowQuery  = "id,uid,target_type,target_id"
)

var userLikesAllowFieldsSlice = []any{"target_type", "target_id"}
var userLikesAllowQuerySlice = []any{"id", "uid", "target_type", "target_id"}

func (this *UserLikes) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *UserLikes) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *UserLikes) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *UserLikes) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":      this.one,
		"all":      this.all,
		"sum":      this.sum,
		"min":      this.min,
		"max":      this.max,
		"rand":     this.rand,
		"count":    this.count,
		"column":   this.column,
		"is-liked": this.isLiked,
		"likes":    this.likes,
		"counts":   this.counts,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *UserLikes) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":   this.save,
		"create": this.create,
		"like":   this.like,
		"unlike": this.unlike,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserLikes) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update": this.update,
		"unlike": this.unlike,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserLikes) IDEL(ctx *gin.Context) {
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

func (this *UserLikes) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *UserLikes) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "user-likes"})
}

func (this *UserLikes) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.UserLikes{}

	for key, val := range params {
		if utils.In.Array(key, userLikesAllowQuerySlice) {
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

func (this *UserLikes) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.UserLikes{}
	for key, val := range params {
		if utils.In.Array(key, userLikesAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.UserLikes

	// 查看他人点赞：支持 uid 参数，并按隐私设置拦截
	currentUid := this.user(ctx).Id
	targetUid := currentUid
	if !utils.Is.Empty(params["uid"]) {
		targetUid = cast.ToInt(params["uid"])
	}
	if targetUid != currentUid && !this.meta.root(ctx) {
		privacy := model.GetUserPrivacy(targetUid)
		if privacy.Likes != 1 {
			this.json(ctx, gin.H{"data": []any{}, "count": 0, "page": 0, "private": true},
				facade.Lang(ctx, "对方设置了私密，无法查看！"), 200)
			return
		}
	}

	query := facade.DB.Model(&result)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uid", targetUid)
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

func (this *UserLikes) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])

	query := facade.DB.Model(&model.UserLikes{})

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.UserLikes{}).Where("id", "IN", ids)
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

func (this *UserLikes) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *UserLikes) create(ctx *gin.Context) {
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

	table := model.UserLikes{Uid: uid}
	for key, val := range params {
		if utils.In.Array(key, userLikesAllowFieldsSlice) {
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

func (this *UserLikes) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.UserLikes{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, userLikesAllowFieldsSlice) {
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

func (this *UserLikes) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := facade.DB.Model(&model.UserLikes{})
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserLikes) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := facade.DB.Model(&model.UserLikes{})
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

func (this *UserLikes) sum(ctx *gin.Context) {
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

func (this *UserLikes) min(ctx *gin.Context) {
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

func (this *UserLikes) max(ctx *gin.Context) {
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

func (this *UserLikes) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := facade.DB.Model(&[]model.UserLikes{})
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

func (this *UserLikes) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserLikes{})
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

func (this *UserLikes) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserLikes{})
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

func (this *UserLikes) clear(ctx *gin.Context) {
	item := facade.DB.Model(&model.UserLikes{})

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

func (this *UserLikes) like(ctx *gin.Context) {
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

	err := (&model.UserLikes{}).Like(uid, targetId, targetType)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "点赞经验值协程发生错误")
			}
		}()

		var authorId int
		var expType string

		switch targetType {
		case "article":
			article, _ := facade.DB.Model(&model.Article{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(article)["uid"])
			expType = "article-like"
		case "page":
			page, _ := facade.DB.Model(&model.Pages{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(page)["uid"])
			expType = "article-like"
		case "moments":
			moment, _ := facade.DB.Model(&model.Moments{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(moment)["uid"])
			expType = "article-like"
		case "comment":
			comment, _ := facade.DB.Model(&model.Comment{}).Where("id", targetId).Find()
			authorId = cast.ToInt(cast.ToStringMap(comment)["uid"])
			expType = "comment-like"
		default:
			return
		}

		if authorId > 0 && authorId != uid {
			_ = (&model.EXP{}).Add(model.EXP{
				Uid:      authorId,
				Type:     expType,
				BindType: targetType,
				BindId:   targetId,
			})
		}
	}()

	// 创建点赞通知
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "创建点赞通知协程发生错误")
			}
		}()

		var authorId int
		var bindType, title, content string
		var bindId int

		switch targetType {
		case "article":
			item, _ := facade.DB.Model(&model.Article{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			title = cast.ToString(itemMap["title"])
			bindType = "article"
			bindId = targetId
		case "page":
			item, _ := facade.DB.Model(&model.Pages{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			title = cast.ToString(itemMap["title"])
			bindType = "page"
			bindId = targetId
		case "moments":
			item, _ := facade.DB.Model(&model.Moments{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			content = cast.ToString(itemMap["content"])
			bindType = "moments"
			bindId = targetId
		case "comment":
			item, _ := facade.DB.Model(&model.Comment{}).Where("id", targetId).Find()
			itemMap := cast.ToStringMap(item)
			authorId = cast.ToInt(itemMap["uid"])
			content = cast.ToString(itemMap["content"])
			bindType = "comment"
			bindId = targetId
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
				notifTitle = "文章被点赞"
				notifContent = fromNickname + " 赞了你的文章「" + title + "」"
			} else {
				notifTitle = "获得新点赞"
				notifContent = fromNickname + " 赞了你的文章"
			}
		case "page":
			if !utils.Is.Empty(title) {
				title = truncateSafe(title, 20)
				notifTitle = "页面被点赞"
				notifContent = fromNickname + " 赞了你的页面「" + title + "」"
			} else {
				notifTitle = "获得新点赞"
				notifContent = fromNickname + " 赞了你的页面"
			}
		case "moments":
			if !utils.Is.Empty(content) {
				content = truncateSafe(content, 30)
				notifTitle = "动态被点赞"
				notifContent = fromNickname + " 赞了你的动态「" + content + "」"
			} else {
				notifTitle = "获得新点赞"
				notifContent = fromNickname + " 赞了你的动态"
			}
		case "comment":
			if !utils.Is.Empty(content) {
				content = truncateSafe(content, 30)
				notifTitle = "评论被点赞"
				notifContent = fromNickname + " 赞了你的评论「" + content + "」"
			} else {
				notifTitle = "获得新点赞"
				notifContent = fromNickname + " 赞了你的评论"
			}
		}

		_, _ = (&model.Notification{}).CreateNotification(
			authorId, uid, "like", notifTitle, notifContent, bindType, bindId,
		)
	}()

	this.json(ctx, gin.H{
		"target_type": targetType,
		"target_id":   targetId,
	}, facade.Lang(ctx, "点赞成功！"), 200)
}

func (this *UserLikes) unlike(ctx *gin.Context) {
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

	err := (&model.UserLikes{}).Unlike(uid, targetId, targetType)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{
		"target_type": targetType,
		"target_id":   targetId,
	}, facade.Lang(ctx, "取消点赞成功！"), 200)
}

func (this *UserLikes) isLiked(ctx *gin.Context) {
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
	isLiked := false
	if uid > 0 {
		isLiked = (&model.UserLikes{}).IsLiked(uid, targetId, targetType)
	}
	count := (&model.UserLikes{}).GetLikesCount(targetId, targetType)

	this.json(ctx, gin.H{
		"is_liked": isLiked,
		"count":    count,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserLikes) likes(ctx *gin.Context) {
	params := this.params(ctx)

	currentUid := this.meta.user(ctx).Id
	// 公共接口：任何人都能通过 uid 查询对应用户数据。
	// 已登录用户未传 uid（或 uid 等于自己）时查自己；未登录用户必须传 uid。
	var targetUid int
	uidParam := cast.ToInt(params["uid"])
	if uidParam > 0 {
		targetUid = uidParam
	} else if currentUid > 0 {
		targetUid = currentUid
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "请指定要查询的用户（uid）！"), 400)
		return
	}

	// 查看他人点赞时按隐私设置拦截
	if targetUid != currentUid && !this.meta.root(ctx) {
		privacy := model.GetUserPrivacy(targetUid)
		if privacy.Likes != 1 {
			this.json(ctx, gin.H{"list": []any{}, "count": 0, "private": true},
				facade.Lang(ctx, "对方设置了私密，无法查看！"), 200)
			return
		}
	}

	targetType := cast.ToString(params["target_type"])
	data, count := (&model.UserLikes{}).GetLikesByUid(targetUid, targetType)

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, gin.H{"list": utils.ArrayMapWithField(data, params["field"]), "count": count}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserLikes) counts(ctx *gin.Context) {
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
	// target_ids 可能是逗号分隔字符串（如 "1,2,3"），cast.ToIntSlice 在 v1.10 无法解析字符串，
	// 需先经 utils.Unity.Ids 归一化（项目内标准做法），否则批量查询会返回空结果
	targetIds := cast.ToIntSlice(utils.Unity.Ids(params["target_ids"]))

	var counts map[int]int64

	if targetType == "user_likes" {
		counts = make(map[int]int64)
		for _, uid := range targetIds {
			counts[uid] = (&model.UserLikes{}).GetUserLikesCount(uid)
		}
	} else {
		counts = (&model.UserLikes{}).GetLikesCounts(targetType, targetIds)
	}

	this.json(ctx, gin.H{"counts": counts}, facade.Lang(ctx, "查询成功！"), 200)
}

// truncateSafe 按字符(rune)数截断字符串，超长时追加省略号
// 供 user-likes.go / user-collects.go 生成通知内容时使用
func truncateSafe(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
