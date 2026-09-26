package controller

/**
 * 消息通知（/api/notification）
 *
 * 权限模型：
 * - 查询/统计类接口（one / all / count / rand / column / sum|min|max）统一走 applyScope：
 *   普通用户仅自己的通知；管理员（root）为自己 + 广播通知（uid=0）；显式 uid 再做精确过滤
 * - 写操作（create / update / remove / delete）带归属校验：普通用户只能操作自己的通知，
 *   且 create / update 不允许指定或修改 uid / from_uid（防止伪造广播或给他人投递）
 * - 管理端视角（scope=admin，仅对 root 生效）：
 *   查询不做 uid 限制、写操作不限归属，供后台「消息管理」查看 / 维护全站通知（含发给指定用户的记录）；
 *   restore / update / remove / delete 都支持该视角，非 root 传入会被忽略（回落到默认范围）
 *
 * 广播通知（uid=0）：
 * - 只存一条共享记录，用户的「已读 / 隐藏」状态在 notification_read 表
 * - read-all 只写「未读且未隐藏」的广播；用户删除广播 = 隐藏自己，管理员删除 = 全体撤回
 *
 * 批量上限（notificationBatchLimit = 200）：
 * - read-batch / remove 直接校验入参数量
 * - remove-all 单次处理上限，返回 { cleared, remaining }，前端按 remaining 继续清理
 *
 * 缓存：GET 接口按「方法 + 路径 + 参数 + uid」缓存（cacheKey），并按标签 [GET]notification 失效
 */

import (
	"fmt"
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

	// notificationBatchLimit 批量操作单次处理的 ID 上限
	// read-batch 直接限制入参数量；remove-all 单次最多清理这么多条，
	// 返回 cleared / remaining，前端可提示用户再次点击（避免一条请求构造超长 IN 子句）。
	notificationBatchLimit = 200
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

// delCache 清理通知相关查询缓存
// 缓存关闭时直接跳过；清理失败（如没有命中的缓存键）记录一条 Warn，便于排查「改了数据列表不刷新」
func (this *Notification) delCache() {
	if !cast.ToBool(facade.CacheToml.Get("open")) {
		return
	}

	if !facade.Cache.DelTags([]any{"[GET]", "notification"}) {
		facade.Log.Warn(map[string]any{"tag": "[GET]notification"}, "清理通知缓存未命中或失败")
	}
}

// cacheKey 通知查询的缓存键
//
// 必须带上用户维度：base.cache.name() 只由「方法 + 路径 + 参数」决定，
// 不同用户用同样的参数请求会命中同一份缓存，导致互相看到对方的消息。
func (this *Notification) cacheKey(ctx *gin.Context) string {
	return fmt.Sprintf("%v&uid=%v", this.cache.name(ctx), this.meta.user(ctx).Id)
}

// tooManyIds 批量操作的 ID 数量校验（超出上限时写入错误响应并返回 true）
func (this *Notification) tooManyIds(ctx *gin.Context, count int) bool {
	if count <= notificationBatchLimit {
		return false
	}
	this.json(ctx, nil, facade.Lang(ctx, "一次最多处理 %v 条，请分批操作！", notificationBatchLimit), 400)
	return true
}

// adminScope 是否以「管理端视角」操作：仅 root 且显式声明 scope=admin
//
// 后台「消息管理」需要查看 / 维护全站通知（含发给指定用户的记录），
// 而默认范围（自己 + 广播）看不到这些数据，所以用一个显式开关放行：
// 非 root 即使传了 scope=admin 也会被忽略（回落到默认范围），避免普通用户越权。
func (this *Notification) adminScope(ctx *gin.Context, params map[string]any) bool {
	return cast.ToString(params["scope"]) == "admin" && this.meta.root(ctx)
}

// applyScope 统一通知查询的可见范围（one / all / count / rand / column / sum|min|max 共用）
//
// 规则与主题端 notification/list 一致：
//   - 普通用户：仅自己的通知（uid = 当前用户）
//   - 管理员（root）：自己的通知 + 广播通知（uid = 0），便于查看/统计公告
//   - 管理员（root）且 scope=admin：不做 uid 限制，可查看/统计全站通知（后台消息管理用）
//
// 说明：广播通知对用户的「已读 / 隐藏」状态在 notification_read 表中，
// 这里只控制可见性；隐藏过滤由 list / unread-count / MarkAllRead 的 JOIN 完成。
func (this *Notification) applyScope(ctx *gin.Context, query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if this.adminScope(ctx, params) {
		return query
	}
	if this.meta.root(ctx) {
		return query.Where("uid", "IN", []any{0, this.meta.user(ctx).Id})
	}
	return query.Where("uid", this.meta.user(ctx).Id)
}

func (this *Notification) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Notification{}

	for key, val := range params {
		// 空字符串不参与结构体过滤：is_read="" 会被 cast 成 0，误判为「只看未读」
		if utils.In.Array(key, notificationAllowQuerySlice) && !utils.Is.Empty(val) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cacheKey(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)
		query = this.applyScope(ctx, query, params)

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
		// 空字符串不参与结构体过滤：is_read="" 会被 cast 成 0，误判为「只看未读」
		if utils.In.Array(key, notificationAllowQuerySlice) && !utils.Is.Empty(val) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Notification

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	// 可见范围：普通用户仅自己；管理员可同时查看广播通知（uid=0）；
	// 管理员且 scope=admin 时不做 uid 限制（后台消息管理查看全站通知）
	query = this.applyScope(ctx, query, params)

	// 显式指定 uid 参数时精确过滤（如"系统公告" uid=0；零值无法通过结构体过滤）
	if uidParam, ok := params["uid"]; ok && !utils.Is.Empty(uidParam) {
		query = query.Where("uid", cast.ToInt(uidParam))
	}

	count, _ := query.Where(table).Count()

	cacheName := this.cacheKey(ctx)
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

	// is_read：不传 / 传空串 / 传 null 都表示「全部」，仅显式传 0 / 1 时才过滤
	// （原实现把 "" 交给 cast.ToInt 得到 0，会被误判成「只看未读」）
	isRead := -1
	if raw, ok := params["is_read"]; ok && !utils.Is.Empty(raw) {
		isRead = cast.ToInt(raw)
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
		// 通知不存在 / 不属于当前用户时，直接回传模型层的提示，便于定位越权操作
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
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

	// 限制单次批量数量：ids 直接来自请求，不加限制会构造超长 IN 子句 / 逐条 upsert
	if this.tooManyIds(ctx, len(ids)) {
		return
	}

	// 分离广播通知与个人通知（广播通知 uid=0，个人通知必须属于当前用户）
	var broadcastIds, personalIds []int
	items, _ := facade.DB.Model(&[]model.Notification{}).WhereIn("id", ids).Select()
	for _, item := range items {
		switch {
		case cast.ToInt(item["uid"]) == 0:
			broadcastIds = append(broadcastIds, cast.ToInt(item["id"]))
		case cast.ToInt(item["uid"]) == uid:
			personalIds = append(personalIds, cast.ToInt(item["id"]))
		}
	}

	// 个人通知：直接更新已读状态
	if !utils.Is.Empty(personalIds) {
		_, err := facade.DB.Model(&model.Notification{}).
			WhereIn("id", personalIds).
			Where("uid", uid).
			Update(map[string]any{"is_read": 1})

		if err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "批量标记已读失败！"), 400)
			return
		}
	}

	// 广播通知：写入该用户的已读状态
	for _, nid := range broadcastIds {
		if err := (&model.Notification{}).MarkBroadcastRead(nid, uid); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "批量标记已读失败！"), 400)
			return
		}
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

// create 创建通知
//
// 权限约束：
//   - 登录用户只能给自己创建（uid / from_uid 一律由服务端填充），
//     否则普通用户可以伪造 uid=0 的「广播通知」或给任意用户投递消息；
//   - 管理员（root）可以指定 uid / from_uid（uid=0 即广播）。
func (this *Notification) create(ctx *gin.Context) {
	params := this.params(ctx)

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	root := this.meta.root(ctx)
	table := model.Notification{}

	for key, val := range params {
		if !utils.In.Array(key, notificationAllowFieldsSlice) {
			continue
		}
		// 非管理员：忽略投递对象字段，只能给自己创建
		if !root && (key == "uid" || key == "from_uid") {
			continue
		}
		utils.Struct.Set(&table, key, val)
	}

	// 兜底：非管理员强制归属自己；管理员未指定接收人（uid=0）时同样落到自己，
	// 需要发广播请使用 send-system（有独立的管理员校验）
	if !root || table.Uid == 0 {
		table.Uid = uid
	}
	if !root {
		table.FromUid = uid
	}

	_, err := facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新通知
//
// 权限约束：
//   - 普通用户：只能更新自己的通知（uid = 自己），且不能改 uid / from_uid
//     （否则可以把别人的通知「搬」到自己名下，或把消息投递到任意用户）；
//   - 管理员（root）：可更新自己的与广播通知（uid IN (0, 自己)），允许改投递字段。
func (this *Notification) update(ctx *gin.Context) {
	params := this.params(ctx)
	uid := this.meta.user(ctx).Id

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	root := this.meta.root(ctx)
	table := model.Notification{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if !utils.In.Array(key, notificationAllowFieldsSlice) {
			continue
		}
		// 非管理员：不允许改投递对象
		if !root && (key == "uid" || key == "from_uid") {
			continue
		}
		async.Set(key, val)
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])
	// 越权防护：普通用户仅能改自己的；管理员可改自己的与广播；
	// 管理端视角（scope=admin，仅 root）可改任意用户的通知（如发给指定用户的系统消息）
	switch {
	case this.adminScope(ctx, params):
	case root:
		item = item.Where("uid", "IN", []any{0, uid})
	default:
		item = item.Where("uid", uid)
	}

	tx, err := item.Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 没有命中任何记录：说明该通知不属于当前用户或不存在
	if tx != nil && tx.RowsAffected == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Notification) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Notification{}), params)
	query = this.buildQuery(query, params)
	query = this.applyScope(ctx, query, params)

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
	query = this.applyScope(ctx, query, params)

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		return nil, ""
	}

	cacheName := this.cacheKey(ctx)
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
	query = this.applyScope(ctx, query, params)

	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Notification{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)
	mold = this.applyScope(ctx, mold, params)

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
	query = this.applyScope(ctx, query, params)

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	cacheName := this.cacheKey(ctx)
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

	// 数量上限：广播通知是「逐条写隐藏状态」，不限制会被拿来放大请求
	if this.tooManyIds(ctx, len(ids)) {
		return
	}

	// 管理端视角：可撤回任意用户的通知（后台消息管理）
	admin := this.adminScope(ctx, params)

	// 分离广播通知与个人通知（广播通知 uid=0 对所有用户可见，删除即对该用户隐藏）
	var broadcastIds, personalIds []int
	items, _ := facade.DB.Model(&[]model.Notification{}).WhereIn("id", ids).Select()
	for _, item := range items {
		switch {
		case cast.ToInt(item["uid"]) == 0:
			broadcastIds = append(broadcastIds, cast.ToInt(item["id"]))
		case admin || cast.ToInt(item["uid"]) == uid:
			personalIds = append(personalIds, cast.ToInt(item["id"]))
		}
	}

	if utils.Is.Empty(broadcastIds) && utils.Is.Empty(personalIds) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 个人通知：软删除（管理端视角不限归属）
	if !utils.Is.Empty(personalIds) {
		query := facade.DB.Model(&model.Notification{}).WhereIn("id", personalIds)
		if !admin {
			query = query.Where("uid", uid)
		}

		if _, err := query.Delete(personalIds); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
			return
		}
	}

	// 广播通知：管理员删除则撤下公告（软删除，全体不可见）；普通用户删除仅隐藏自己
	if this.meta.root(ctx) && !utils.Is.Empty(broadcastIds) {
		if _, err := facade.DB.Model(&model.Notification{}).WhereIn("id", broadcastIds).Delete(broadcastIds); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
			return
		}
	} else {
		for _, nid := range broadcastIds {
			if err := (&model.Notification{}).HideBroadcast(nid, uid); err != nil {
				facade.Log.Error(map[string]any{"error": err, "id": nid}, "隐藏广播通知失败")
			}
		}
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

	// 管理员可彻底删除广播通知（uid=0，撤回公告），普通用户仅限自己的通知；
	// 管理端视角（scope=admin，仅 root）可彻底删除任意用户的通知
	item := facade.DB.Model(&model.Notification{}).WithTrashed()
	switch {
	case this.adminScope(ctx, params):
	case this.meta.root(ctx):
		item = item.Where("uid", "IN", []any{0, uid})
	default:
		item = item.Where("uid", uid)
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

// removeAll 清空当前用户的消息（可按类型 / 仅已读）
//
// 单次处理上限 notificationBatchLimit：
// 个人通知按条件软删除、广播通知写「隐藏」状态，均在内存里拿 id 列表后批量执行。
// 不加限制时，消息量大的账号会生成超长 IN 子句并占用大量内存。
// 返回 cleared（本次处理条数）与 remaining（剩余可见条数），前端可提示再次点击。
func (this *Notification) removeAll(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	typ := cast.ToString(params["type"])

	// is_read：仅显式传 1 时只清空「已读」通知（用于「清空已读消息」）
	onlyRead := false
	if raw, ok := params["is_read"]; ok && !utils.Is.Empty(raw) {
		onlyRead = cast.ToInt(raw) == 1
	}

	// 个人通知的查询条件（每次重建，避免复用同一个 ModelStruct 时条件叠加）
	personalQuery := func() *facade.ModelStruct {
		item := facade.DB.Model(&model.Notification{}).Where("uid", uid)
		if typ != "" {
			item = item.Where("type", typ)
		}
		if onlyRead {
			item = item.Where("is_read", 1)
		}
		return item
	}

	// ---------- 个人通知：软删除（单次上限） ----------
	columnData, _ := personalQuery().Limit(notificationBatchLimit).Column("id")
	ids := utils.Unity.Ids(columnData)

	cleared := 0
	if !utils.Is.Empty(ids) {
		if _, err := personalQuery().WhereIn("id", ids).Delete(ids); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
			return
		}
		cleared += len(ids)
	}

	// ---------- 广播通知：对该用户隐藏（不删除共享记录） ----------
	// 仅处理该用户「可见」的广播；清空已读时只处理其中该用户已读的
	bcIds := this.broadcastTargets(uid, typ, onlyRead)
	remainingBroadcast := len(bcIds)
	if len(bcIds) > notificationBatchLimit {
		bcIds = bcIds[:notificationBatchLimit]
	}

	for _, nid := range bcIds {
		if err := (&model.Notification{}).HideBroadcast(nid, uid); err != nil {
			facade.Log.Error(map[string]any{"error": err, "id": nid}, "隐藏广播通知失败")
		}
	}
	cleared += len(bcIds)
	remainingBroadcast -= len(bcIds)

	if cleared == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 剩余可见条数（个人 + 广播），用于前端提示「点一次清不完」
	personalRemain, _ := personalQuery().Count()
	remaining := cast.ToInt(personalRemain) + remainingBroadcast

	this.json(ctx, gin.H{
		"ids":       ids,
		"cleared":   cleared,
		"remaining": remaining,
	}, facade.Lang(ctx, "清空成功！"), 200)
}

// broadcastTargets 广播通知的处理清单（返回 ID 列表，按 id 倒序）
// 可见 = 未被软删除 且 该用户未隐藏；onlyRead=true 时再要求该用户已读
func (this *Notification) broadcastTargets(uid int, typ string, onlyRead bool) []int {

	visible := (&model.Notification{}).VisibleBroadcastIds(uid, typ, 0)
	if !onlyRead {
		return visible
	}

	// 该用户已读的广播（已读状态记录在 notification_read 表）
	readData, _ := facade.DB.Model(&model.NotificationRead{}).
		Where("uid", uid).
		Where("is_read", 1).
		Column("notification_id")

	readSet := make(map[int]bool)
	for _, id := range utils.Unity.Ids(readData) {
		readSet[cast.ToInt(id)] = true
	}

	result := make([]int, 0, len(visible))
	for _, id := range visible {
		if readSet[id] {
			result = append(result, id)
		}
	}

	return result
}

func (this *Notification) clear(ctx *gin.Context) {
	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 管理员清空回收站时同时清理广播通知（uid=0），普通用户仅限自己的通知
	item := facade.DB.Model(&model.Notification{}).OnlyTrashed()
	if this.meta.root(ctx) {
		item = item.Where("uid", "IN", []any{0, uid})
	} else {
		item = item.Where("uid", uid)
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

func (this *Notification) restore(ctx *gin.Context) {
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

	// 归属校验：普通用户只能恢复自己的通知（含广播），
	// 管理员可恢复自己的与广播，管理端视角（scope=admin，仅 root）可恢复任意用户的通知
	item := facade.DB.Model(&model.Notification{}).OnlyTrashed().WhereIn("id", ids)
	switch {
	case this.adminScope(ctx, params):
	case this.meta.root(ctx):
		item = item.Where("uid", "IN", []any{0, uid})
	default:
		item = item.Where("uid", uid)
	}

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

	// 构造通知标题和内容
	var notifTitle, notifContent string
	if asSystem {
		// 标题前缀与后台「消息管理」的命名保持一致（群发短消息 / 短消息）
		notifTitle = "【短消息】" + title
		notifContent = content
	} else {
		// 获取管理员昵称（未设置昵称时回退「管理员」，避免出现「 发送了一条短消息」这种空主语）
		adminInfo, _ := facade.DB.Model(&model.Users{}).Where("id", uid).Find()
		adminNickname := strings.TrimSpace(cast.ToString(cast.ToStringMap(adminInfo)["nickname"]))
		if utils.Is.Empty(adminNickname) {
			adminNickname = "管理员"
		}
		notifTitle = title
		notifContent = adminNickname + " 发送了一条短消息：" + content
	}

	// 全量推送：广播模式，仅创建一条 uid=0 的记录，全体用户通过列表接口可见
	// 不向百万级用户逐条创建记录，也不发送邮件（无法向海量用户发邮件）
	if targetType == "all" {
		broadcast, err := (&model.Notification{}).CreateBroadcastNotification(uid, "system", notifTitle, notifContent, "", 0)
		if err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "广播推送失败！"), 400)
			return
		}
		// 发送人（管理员）自己标记为已读，避免后台角标出现未读
		_ = (&model.Notification{}).MarkBroadcastRead(broadcast.Id, uid)

		this.json(ctx, gin.H{
			"broadcast": true,
			"id":        broadcast.Id,
			"total":     1,
			"success":   1,
		}, facade.Lang(ctx, "广播成功！全体用户可见"), 200)
		return
	}

	// 部分用户 / 单个用户：正常为每个用户创建记录
	if utils.Is.Empty(userIds) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "目标用户"), 400)
		return
	}

	// 批量发送通知
	successCount := 0

	for _, targetUid := range cast.ToSlice(userIds) {
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
			go func(targetUid int, t, c string) {
				defer func() {
					if r := recover(); r != nil {
						facade.Log.Error(map[string]any{"error": r}, "系统消息邮件通知协程错误")
					}
				}()

				userInfo, _ := facade.DB.Model(&model.Users{}).Find(targetUid)
				if utils.Is.Empty(userInfo) {
					return
				}

				user := cast.ToStringMap(userInfo)
				email := cast.ToString(user["email"])
				if utils.Is.Empty(email) {
					return
				}

				// 用「用户消息通知」模板（facade.SendMessageNotify）：
				// 不要用 SendCommentNotify —— 那是评论通知模板，会渲染「评论者 / 评论 IP」等评论字段；
				// 也刻意不走 facade.SMS：短信驱动发不了邮件，站点配了短信驱动时会静默失败
				// 收件人的账号 / 昵称一并带上，用户能一眼确认这封邮件发给的是哪个账号
				response := facade.SendMessageNotify(email, map[string]any{
					"title":    t,
					"content":  c,
					"account":  cast.ToString(user["account"]),
					"nickname": cast.ToString(user["nickname"]),
				})

				if response != nil && response.Error != nil {
					facade.Log.Error(map[string]any{
						"error":     response.Error,
						"uid":       targetUid,
						"recipient": facade.Comm.MaskEmail(email),
					}, "系统消息邮件通知发送失败")
				}
			}(targetId, notifTitle, notifContent)
		}
	}

	this.json(ctx, gin.H{
		"total":   len(cast.ToSlice(userIds)),
		"success": successCount,
	}, facade.Lang(ctx, "推送完成！"), 200)
}
