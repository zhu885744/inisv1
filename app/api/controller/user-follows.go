package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"math"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type UserFollows struct {
	base
}

const (
	userFollowsAllowFields = "follow_uid,status,description"
	userFollowsAllowQuery  = "id,uid,follow_uid,status"
)

var userFollowsAllowFieldsSlice = []any{"follow_uid", "status", "description"}
var userFollowsAllowQuerySlice = []any{"id", "uid", "follow_uid", "status"}

func (this *UserFollows) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *UserFollows) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *UserFollows) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *UserFollows) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *UserFollows) IGET(ctx *gin.Context) {
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
		"following":    this.following,
		"followers":    this.followers,
		"is-following": this.isFollowing,
		"counts":       this.counts,
		// 主页社交化：互关状态 / 可能认识的人 / 社交数据卡片
		"relation": this.relation,
		"suggest":  this.suggest,
		"stats":    this.stats,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *UserFollows) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":     this.save,
		"create":   this.create,
		"follow":   this.follow,
		"unfollow": this.unfollow,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserFollows) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":   this.update,
		"restore":  this.restore,
		"unfollow": this.unfollow,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *UserFollows) IDEL(ctx *gin.Context) {
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

func (this *UserFollows) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *UserFollows) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "user-follows"})
}

func (this *UserFollows) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.UserFollows{}

	for key, val := range params {
		if utils.In.Array(key, userFollowsAllowQuerySlice) {
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

func (this *UserFollows) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.UserFollows{}
	for key, val := range params {
		if utils.In.Array(key, userFollowsAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.UserFollows

	query := this.withTrashOptions(facade.DB.Model(&result), params)
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

func (this *UserFollows) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.UserFollows{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.UserFollows{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
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

func (this *UserFollows) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *UserFollows) create(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["follow_uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "follow_uid"), 400)
		return
	}

	followUid := cast.ToInt(params["follow_uid"])
	if uid == followUid {
		this.json(ctx, nil, facade.Lang(ctx, "不能关注自己！"), 400)
		return
	}

	// 走模型层的关注逻辑：同一个 (uid, follow_uid) 复用已有记录（含软删除），
	// 不直接 Create，避免重复关注插入多行
	id, err := (&model.UserFollows{}).FollowWith(uid, followUid, cast.ToString(params["description"]))

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *UserFollows) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.UserFollows{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, userFollowsAllowFieldsSlice) {
			async.Set(key, val)
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

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

func (this *UserFollows) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.UserFollows{}), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uid", this.user(ctx).Id)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserFollows) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.UserFollows{}), params)
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

func (this *UserFollows) sum(ctx *gin.Context) {
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

func (this *UserFollows) min(ctx *gin.Context) {
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

func (this *UserFollows) max(ctx *gin.Context) {
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

func (this *UserFollows) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.UserFollows{}), params)
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

func (this *UserFollows) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserFollows{})
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

func (this *UserFollows) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserFollows{}).WithTrashed()
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

func (this *UserFollows) clear(ctx *gin.Context) {
	table := model.UserFollows{}
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

func (this *UserFollows) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.UserFollows{}).OnlyTrashed().WhereIn("id", ids)
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.UserFollows{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *UserFollows) follow(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["follow_uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "follow_uid"), 400)
		return
	}

	followUid := cast.ToInt(params["follow_uid"])

	err := (&model.UserFollows{}).Follow(uid, followUid)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 创建通知 - 被关注通知
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "创建关注通知协程发生错误")
			}
		}()

		userInfo, _ := facade.DB.Model(&model.Users{}).Find(uid)
		fromNickname := cast.ToString(cast.ToStringMap(userInfo)["nickname"])

		_, _ = (&model.Notification{}).CreateNotification(
			followUid, uid, "follow", "有新关注",
			fromNickname+" 关注了你",
			"user", uid,
		)
	}()

	// 关注关系变了 → 清掉双方的社交数据卡片缓存（粉丝 / 关注数）
	this.invalidateSocialStats(uid, followUid)
	// 粉丝数是认证申请条件之一，被关注者的条件进度缓存要立刻失效
	model.CertStatsFlush(followUid)

	this.json(ctx, gin.H{"follow_uid": followUid}, facade.Lang(ctx, "关注成功！"), 200)
}

func (this *UserFollows) unfollow(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["follow_uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "follow_uid"), 400)
		return
	}

	followUid := cast.ToInt(params["follow_uid"])

	err := (&model.UserFollows{}).Unfollow(uid, followUid)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 关注关系变了 → 清掉双方的社交数据卡片缓存（粉丝 / 关注数）
	this.invalidateSocialStats(uid, followUid)
	// 同上：取消关注同样会改变被关注者的粉丝数
	model.CertStatsFlush(followUid)

	this.json(ctx, gin.H{"follow_uid": followUid}, facade.Lang(ctx, "取消关注成功！"), 200)
}

func (this *UserFollows) following(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page": 1,
	})

	currentUid := this.meta.user(ctx).Id
	// 公共接口：任何人都能通过 uid 查询对应用户数据。
	// 已登录用户未传 uid（或 uid 等于自己）时查自己；未登录用户必须传 uid。
	var targetUid int
	if !utils.Is.Empty(params["uid"]) {
		targetUid = cast.ToInt(params["uid"])
	} else if currentUid > 0 {
		targetUid = currentUid
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "请指定要查询的用户（uid）！"), 400)
		return
	}

	// 查看他人关注列表时的隐私校验
	if targetUid != currentUid {
		privacy := model.GetUserPrivacy(targetUid)
		if privacy.Follows == "none" || privacy.Follows == "followers" {
			this.json(ctx, gin.H{"data": []any{}, "count": 0, "page": 0, "private": true},
				facade.Lang(ctx, "对方设置了私密，无法查看！"), 200)
			return
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	data, count := (&model.UserFollows{}).GetFollowing(targetUid, page, limit)
	// 互关标识：批量补 is_following / is_follower / mutual（未登录时全部为 false）
	data = this.attachRelation(data, currentUid, "following")

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserFollows) followers(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page": 1,
	})

	currentUid := this.meta.user(ctx).Id
	// 公共接口：任何人都能通过 uid 查询对应用户数据。
	// 已登录用户未传 uid（或 uid 等于自己）时查自己；未登录用户必须传 uid。
	var targetUid int
	if !utils.Is.Empty(params["uid"]) {
		targetUid = cast.ToInt(params["uid"])
	} else if currentUid > 0 {
		targetUid = currentUid
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "请指定要查询的用户（uid）！"), 400)
		return
	}

	// 查看他人粉丝列表时的隐私校验
	if targetUid != currentUid {
		privacy := model.GetUserPrivacy(targetUid)
		if privacy.Follows == "none" || privacy.Follows == "following" {
			this.json(ctx, gin.H{"data": []any{}, "count": 0, "page": 0, "private": true},
				facade.Lang(ctx, "对方设置了私密，无法查看！"), 200)
			return
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	data, count := (&model.UserFollows{}).GetFollowers(targetUid, page, limit)
	// 互关标识：批量补 is_following / is_follower / mutual（未登录时全部为 false）
	data = this.attachRelation(data, currentUid, "followers")

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserFollows) isFollowing(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["follow_uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "follow_uid"), 400)
		return
	}

	followUid := cast.ToInt(params["follow_uid"])

	isFollowing := (&model.UserFollows{}).IsFollowing(uid, followUid)

	this.json(ctx, gin.H{"is_following": isFollowing}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *UserFollows) counts(ctx *gin.Context) {
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

	counts := (&model.UserFollows{}).GetFollowsCounts(targetType, targetIds)

	this.json(ctx, gin.H{"counts": counts}, facade.Lang(ctx, "查询成功！"), 200)
}

// ============================== 主页社交化 ==============================

const (
	// socialStatsTTL - 社交数据卡片缓存秒数（获赞数要跨表统计，避免每次刷新都算一遍）
	socialStatsTTL = int64(300)
	// socialLikesIdLimit - 统计获赞时单类内容最多取多少条 id（防超大量内容把 IN 撑爆）
	socialLikesIdLimit = 2000
	// suggestLimit - 「可能认识的人」最多返回几个
	suggestLimit = 6
)

// relation - 我与目标用户的关注关系（需登录）
//
// 返回 {is_following, is_follower, mutual}，主页按钮据此显示四种状态：
//
//	未关注 → 关注      已关注（对方没关注我）→ 已关注
//	对方关注我 → 回关   互相关注 → 互关
func (this *UserFollows) relation(ctx *gin.Context) {

	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	followUid := cast.ToInt(params["follow_uid"])
	if followUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "follow_uid"), 400)
		return
	}

	follows := &model.UserFollows{}
	isFollowing := follows.IsFollowing(uid, followUid)
	// 看自己主页时两个方向都按 false 处理，避免按钮显示成「互关」
	isFollower := followUid != uid && follows.IsFollowing(followUid, uid)

	this.json(ctx, gin.H{
		"is_following": isFollowing,
		"is_follower":  isFollower,
		"mutual":       isFollowing && isFollower,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// suggest - 可能认识的人（共同关注二度推荐，需登录）
//
// 算法（不引入新表，全部基于现有 user_follows）：
//  1. 取我关注的人 A
//  2. 取 A 们关注的人 B（二度人脉）
//  3. 排除：我自己、我已经关注的人
//  4. 按「共同关注数」降序，最多 suggestLimit 个
//
// 隐私：只返回公开的作者字段（白名单 + SanitizeAuthor），不暴露任何关注关系明细，
// 前端只展示「你们有 N 个共同关注」这一条聚合信息。
func (this *UserFollows) suggest(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	follows := &model.UserFollows{}

	myFollowing := follows.FollowingUids(uid)
	if len(myFollowing) == 0 {
		this.json(ctx, gin.H{"list": []any{}}, facade.Lang(ctx, "关注一些人之后，这里会推荐你可能认识的人！"), 200)
		return
	}

	mine := make(map[int]bool, len(myFollowing))
	for _, id := range myFollowing {
		mine[id] = true
	}

	// 二度人脉：我关注的人都在关注谁（一次 IN 查询，只取 follow_uid 列）
	column, err := facade.DB.Model(&[]model.UserFollows{}).
		Where("uid", "IN", userFollowsAnyIds(myFollowing)).
		Where("status", 1).
		Column("follow_uid")
	if err != nil {
		this.json(ctx, gin.H{"list": []any{}}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	// 统计共同关注数（自己、已关注的人直接跳过）
	weight := make(map[int]int)
	for _, id := range cast.ToIntSlice(column) {
		if id <= 0 || id == uid || mine[id] {
			continue
		}
		weight[id]++
	}

	if len(weight) == 0 {
		this.json(ctx, gin.H{"list": []any{}}, facade.Lang(ctx, "暂时没有可推荐的人！"), 200)
		return
	}

	ids := make([]int, 0, len(weight))
	for id := range weight {
		ids = append(ids, id)
	}
	// 共同关注多的优先；同分按 uid 升序，保证每次顺序稳定（推荐结果不会跳来跳去）
	sort.Slice(ids, func(i, j int) bool {
		if weight[ids[i]] != weight[ids[j]] {
			return weight[ids[i]] > weight[ids[j]]
		}
		return ids[i] < ids[j]
	})
	if len(ids) > suggestLimit {
		ids = ids[:suggestLimit]
	}

	list := make([]map[string]any, 0, len(ids))
	for _, id := range ids {

		item, _ := facade.DB.Model(&model.Users{}).Find(id)
		if utils.Is.Empty(item) {
			continue
		}

		// 与 moments / 文章内嵌作者同一套白名单，剥掉权限组、封禁管理字段
		user := utils.Map.WithField(item, []string{"id", "nickname", "avatar", "description", "json", "result", "title"})
		model.SanitizeAuthor(user)

		user["mutual_count"] = weight[id]
		list = append(list, user)
	}

	this.json(ctx, gin.H{"list": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// stats - 社交数据卡片：关注 / 粉丝 / 动态 / 文章 / 获赞
//
// 都是主页本来就会展示的公开数字，所以 type=common（未登录也能看别人的主页）。
// 结果缓存 5 分钟；关注 / 取关时调用 invalidateSocialStats 主动清掉双方缓存。
func (this *UserFollows) stats(ctx *gin.Context) {

	params := this.params(ctx)

	targetUid := cast.ToInt(params["uid"])
	if targetUid <= 0 {
		targetUid = this.meta.user(ctx).Id
	}
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	cacheName := socialStatsCacheName(targetUid)
	if facade.Cache.Has(cacheName) {
		if cached, ok := facade.Cache.Get(cacheName).(map[string]any); ok && len(cached) > 0 {
			this.json(ctx, cached, facade.Lang(ctx, "查询成功！"), 200)
			return
		}
	}

	follows := &model.UserFollows{}

	result := gin.H{
		"follow":  follows.GetFollowingCount(targetUid),
		"fans":    follows.GetFollowersCount(targetUid),
		"moment":  socialContentCount[model.Moments](targetUid),
		"article": socialContentCount[model.Article](targetUid),
		"likes":   this.likesReceived(targetUid),
	}

	go facade.Cache.Set(cacheName, result, socialStatsTTL)

	this.json(ctx, result, facade.Lang(ctx, "查询成功！"), 200)
}

// socialStatsCacheName - 社交数据卡片缓存名（保留 [GET] 前缀，便于通用标签清理）
func socialStatsCacheName(uid int) string {
	return "[GET]user-follows-stats[" + cast.ToString(uid) + "]"
}

// invalidateSocialStats - 关注关系变化后清掉相关用户的数据卡片缓存
func (this *UserFollows) invalidateSocialStats(uids ...int) {
	for _, uid := range uids {
		if uid > 0 {
			go facade.Cache.Del(socialStatsCacheName(uid))
		}
	}
}

// socialContentCount - 某用户**对外公开**的内容数（动态 / 文章通用：audit=1 且 status=1）
func socialContentCount[T any](uid int) int64 {
	count, _ := facade.DB.Model(&[]T{}).
		Where("uid", uid).
		Where("audit", 1).
		Where("status", 1).
		Count()
	return count
}

// likesReceived - 获赞数：我发布的公开内容累计被点赞次数
//
// 分两步：先取公开内容的 id（动态 / 文章各自封顶 socialLikesIdLimit 条），
// 再按 target_type + target_id IN (...) 统计 —— 不用 JOIN / 子查询，
// 避免依赖表前缀（前缀由 config 决定，写死在 SQL 里不可靠）。
func (this *UserFollows) likesReceived(uid int) int64 {

	ids := make([]any, 0)
	kinds := make([]any, 0)

	for _, item := range []struct {
		target any
		kind   []any
	}{
		{&[]model.Moments{}, []any{"moment", "moments"}},
		{&[]model.Article{}, []any{"article", "articles"}},
	} {
		column, err := facade.DB.Model(item.target).
			Where("uid", uid).
			Where("audit", 1).
			Where("status", 1).
			Order("id desc").
			Limit(socialLikesIdLimit).
			Column("id")
		if err != nil {
			continue
		}

		slice := cast.ToIntSlice(column)
		if len(slice) == 0 {
			continue
		}

		kinds = append(kinds, item.kind...)
		for _, id := range slice {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return 0
	}

	count, _ := facade.DB.Model(&[]model.UserLikes{}).
		Where("target_type", "IN", kinds).
		Where("target_id", "IN", ids).
		Count()

	return count
}

// attachRelation - 给关注 / 粉丝列表批量补互关标识
//
// kind = "following"：列表项是 TA 关注的人，对方 uid 在 follow_uid
// kind = "followers"：列表项是 TA 的粉丝，对方 uid 在 uid
// 未登录（myUid = 0）时不计算，字段保持 false —— 前端据此只展示列表、不展示关系状态。
// 关键：一次 IN 查询批量算完（只处理当前这一页的 uid），不做逐条 Exist。
func (this *UserFollows) attachRelation(data []map[string]any, myUid int, kind string) []map[string]any {

	if len(data) == 0 {
		return data
	}

	uids := make([]int, 0, len(data))
	for _, item := range data {
		uids = append(uids, followsListItemUid(item, kind))
	}

	follows := &model.UserFollows{}
	iFollow := follows.FilterFollowingUids(myUid, uids) // 我关注了谁
	followMe := follows.FilterFollowerUids(myUid, uids) // 谁关注了我

	for _, item := range data {
		uid := followsListItemUid(item, kind)
		if uid <= 0 {
			continue
		}
		item["is_following"] = iFollow[uid]
		item["is_follower"] = followMe[uid]
		item["mutual"] = iFollow[uid] && followMe[uid]
	}

	return data
}

// followsListItemUid - 取关注 / 粉丝列表项里「对方」的 uid
func followsListItemUid(item map[string]any, kind string) int {
	if kind == "followers" {
		return cast.ToInt(item["uid"])
	}
	return cast.ToInt(item["follow_uid"])
}

// userFollowsAnyIds - []int → []any（facade 的 IN 条件需要 any 切片）
func userFollowsAnyIds(ids []int) []any {
	result := make([]any, 0, len(ids))
	for _, id := range ids {
		result = append(result, id)
	}
	return result
}
