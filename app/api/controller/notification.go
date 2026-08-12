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

type Notification struct {
	base
}

const (
	notificationAllowFields = "uid,from_uid,type,title,content,bind_id,bind_type,is_read"
	notificationAllowQuery  = "id,uid,from_uid,type,bind_id,bind_type,is_read"
)

var notificationAllowFieldsSlice = []any{"uid", "from_uid", "type", "title", "content", "bind_id", "bind_type", "is_read"}
var notificationAllowQuerySlice = []any{"id", "uid", "from_uid", "type", "bind_id", "bind_type", "is_read"}

func (this *Notification) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Notification) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Notification) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Notification) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Notification) IGET(ctx *gin.Context) {
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
		"unread-count": this.unreadCount,
		"list":         this.list,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Notification) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":        this.save,
		"create":      this.create,
		"send-system": this.sendSystem,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Notification) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":     this.update,
		"restore":    this.restore,
		"read":       this.read,
		"read-all":   this.readAll,
		"read-batch": this.readBatch,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Notification) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove":     this.remove,
		"delete":     this.delete,
		"clear":      this.clear,
		"remove-all": this.removeAll,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Notification) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Notification) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "notification"})
}

func (this *Notification) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Notification{}

	for key, val := range params {
		if utils.In.Array(key, notificationAllowQuerySlice) {
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
		query = query.Where("uid", this.user(ctx).Id)

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

func (this *Notification) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Notification{}
	for key, val := range params {
		if utils.In.Array(key, notificationAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Notification

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	query = query.Where("uid", this.user(ctx).Id)

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

func (this *Notification) list(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	typ := cast.ToString(params["type"])
	isRead := cast.ToInt(params["is_read"])
	if params["is_read"] == nil {
		isRead = -1
	}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	data, count := (&model.Notification{}).GetNotifications(uid, typ, isRead, page, limit, cast.ToString(params["order"]))

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

func (this *Notification) unreadCount(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	count := (&model.Notification{}).GetUnreadCount(uid)

	this.json(ctx, gin.H{"count": count}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Notification) read(ctx *gin.Context) {
	params := this.params(ctx)
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := (&model.Notification{}).MarkRead(uid, cast.ToInt(params["id"]))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "标记已读失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"id": params["id"]}, facade.Lang(ctx, "标记已读成功！"), 200)
}

func (this *Notification) readAll(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	err := (&model.Notification{}).MarkAllRead(uid)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "全部标记已读失败！"), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "全部标记已读成功！"), 200)
}

func (this *Notification) readBatch(ctx *gin.Context) {
	params := this.params(ctx)
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	_, err := facade.DB.Model(&model.Notification{}).
		WhereIn("id", ids).
		Where("uid", uid).
		Update(map[string]any{"is_read": 1})

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "批量标记已读失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "批量标记已读成功！"), 200)
}

func (this *Notification) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Notification) create(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 && cast.ToInt(params["uid"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Notification{Uid: cast.ToInt(params["uid"])}
	if table.Uid == 0 {
		table.Uid = uid
	}

	for key, val := range params {
		if utils.In.Array(key, notificationAllowFieldsSlice) {
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

func (this *Notification) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.Notification{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, notificationAllowFieldsSlice) {
			async.Set(key, val)
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	_, err := item.Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Notification) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Notification{}), params)
	query = this.buildQuery(query, params)
	query = query.Where("uid", this.user(ctx).Id)

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Notification) sum(ctx *gin.Context) {
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

func (this *Notification) min(ctx *gin.Context) {
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

func (this *Notification) max(ctx *gin.Context) {
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

func (this *Notification) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Notification{}), params)
	query = this.buildQuery(query, params).Order(params["order"])
	query = query.Where("uid", this.user(ctx).Id)

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

func (this *Notification) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.Notification{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	query = query.Where("uid", this.user(ctx).Id)

	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Notification{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)
	mold = mold.Where("uid", this.user(ctx).Id)

	items, _ := mold.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *Notification) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Notification{}), params)
	query = this.buildQuery(query, params).Order(params["order"])
	query = query.Where("uid", this.user(ctx).Id)

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

func (this *Notification) remove(ctx *gin.Context) {
	params := this.params(ctx)
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Notification{}).Where("uid", uid)
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

func (this *Notification) delete(ctx *gin.Context) {
	params := this.params(ctx)
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Notification{}).WithTrashed().Where("uid", uid)
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

func (this *Notification) removeAll(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	typ := cast.ToString(params["type"])

	item := facade.DB.Model(&model.Notification{}).Where("uid", uid)
	if typ != "" {
		item = item.Where("type", typ)
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

func (this *Notification) clear(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	item := facade.DB.Model(&model.Notification{}).OnlyTrashed().Where("uid", uid)

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

func (this *Notification) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Notification{}).OnlyTrashed().WhereIn("id", ids)

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.Notification{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// sendSystem 系统消息推送 - 管理员向指定用户发送系统通知
func (this *Notification) sendSystem(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 检查管理员权限
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无操作权限！"), 403)
		return
	}

	params := this.params(ctx)

	targetType := cast.ToString(params["target_type"]) // all | partial | single
	title := cast.ToString(params["title"])
	content := cast.ToString(params["content"])
	sendEmail := cast.ToBool(params["send_email"])
	asSystem := cast.ToBool(params["as_system"])
	userIds := utils.Unity.Ids(params["user_ids"])

	// 参数校验
	if utils.Is.Empty(title) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "通知标题"), 400)
		return
	}
	if utils.Is.Empty(content) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "通知内容"), 400)
		return
	}
	if utils.Is.Empty(targetType) {
		targetType = "all"
	}

	// 获取目标用户列表
	var targetUsers []any
	switch targetType {
	case "all":
		// 获取所有用户ID
		ids, _ := facade.DB.Model(&model.Users{}).Column("id")
		targetUsers = utils.Unity.Keys(ids)
	case "partial", "single":
		if utils.Is.Empty(userIds) {
			this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "目标用户"), 400)
			return
		}
		targetUsers = userIds
	default:
		this.json(ctx, nil, facade.Lang(ctx, "无效的目标类型！"), 400)
		return
	}

	if utils.Is.Empty(targetUsers) {
		this.json(ctx, nil, facade.Lang(ctx, "没有可发送的目标用户！"), 204)
		return
	}

	// 构造通知标题和内容
	var notifTitle, notifContent string
	if asSystem {
		notifTitle = "【系统消息】" + title
		notifContent = content
	} else {
		// 获取管理员昵称
		adminInfo, _ := facade.DB.Model(&model.Users{}).Where("id", uid).Find()
		adminNickname := cast.ToString(cast.ToStringMap(adminInfo)["nickname"])
		notifTitle = title
		notifContent = adminNickname + " 发送了一条系统通知：" + content
	}

	// 批量发送通知
	successCount := 0

	for _, targetUid := range cast.ToSlice(targetUsers) {
		targetId := cast.ToInt(targetUid)
		if targetId <= 0 {
			continue
		}

		_, err := (&model.Notification{}).CreateNotification(
			targetId,
			uid,
			"system",
			notifTitle,
			notifContent,
			"",
			0,
		)

		if err != nil {
			facade.Log.Error(map[string]any{"error": err, "uid": targetId}, "系统消息推送失败")
			continue
		}

		successCount++

		// 如果需要发送邮件
		if sendEmail {
			go func(uid int, t, c string) {
				defer func() {
					if r := recover(); r != nil {
						facade.Log.Error(map[string]any{"error": r}, "系统邮件通知协程错误")
					}
				}()

				userInfo, _ := facade.DB.Model(&model.Users{}).Find(uid)
				if utils.Is.Empty(userInfo) {
					return
				}

				email := cast.ToString(cast.ToStringMap(userInfo)["email"])
				if utils.Is.Empty(email) {
					return
				}

				// 使用现有邮件发送机制
				commentInfo := map[string]any{
					"email":   email,
					"subject": t,
					"content": c,
				}
				facade.SMS.SendCommentNotify(email, commentInfo)
			}(targetId, notifTitle, notifContent)
		}
	}

	this.json(ctx, gin.H{
		"total":   len(cast.ToSlice(targetUsers)),
		"success": successCount,
	}, facade.Lang(ctx, "推送完成！"), 200)
}
