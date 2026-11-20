package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"math/rand"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Comment struct {
	base
}

const (
	// 单条评论最多携带的图片数（前端同样限制，这里兜底防绕过）
	maxCommentImages = 9
	// 单条评论最多 @ 的人数（防止一条评论触发上千条站内信）
	maxCommentMentions = 10
	// 评论树最大展示层级，同时作为脏数据成环的熔断（每层一次查询，8 层已足够覆盖真实回复链）
	maxCommentDepth = 8
	// 评论批量审核一次最多处理多少条
	maxCommentBatch = 200
)

// 可写入字段（create / update 白名单）；images 在写入前走 sanitizeCommentImages 单独校验
var commentAllowFieldsSlice = []any{"pid", "content", "images", "bind_id", "bind_type", "editor", "json", "text", "scope"}

// 可用于「按字段筛选」的字段
var commentAllowQuerySlice = []any{"id", "pid", "uid", "bind_id", "bind_type", "editor", "audit", "scope", "create_time", "update_time"}

// 评论支持的绑定类型：未知类型一律拒绝，防止往评论表塞任意 bind_type / bind_id 的脏数据
var commentBindTypes = []any{"article", "page", "moments", "docs"}

// commentWhereKeys - 允许出现在 where / or / not / null / notNull 里的字段
//
// 取评论表自己的列：接口返回的 ip / agent 已经脱敏，但 where[ip][like]=10.% 仍然
// 可以靠「有没有结果」做布尔盲探，把脱敏后的 IP 前缀猜出来。
var commentWhereKeys = func() map[string]bool {
	keys := utils.Struct.Keys(&model.Comment{})
	set := make(map[string]bool, len(keys))
	for _, key := range keys {
		set[cast.ToString(key)] = true
	}
	return set
}()

// commentFilterWhere - 过滤条件字段白名单（非表字段一律丢弃）
func commentFilterWhere(raw any) map[string]any {
	list := cast.ToStringMap(raw)
	if len(list) == 0 {
		return nil
	}
	result := make(map[string]any, len(list))
	for key, val := range list {
		if commentWhereKeys[key] {
			result[key] = val
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (this *Comment) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(commentFilterWhere(params["where"])).
		IOr(commentFilterWhere(params["or"])).
		ILike(params["like"]).
		INot(commentFilterWhere(params["not"])).
		INull(commentFilterWhere(params["null"])).
		INotNull(commentFilterWhere(params["notNull"]))
}

func (this *Comment) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Comment) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

// setCache - 写缓存
//
// 这里必须同步写：以前用 go 异步写，会出现「刚查完的请求立刻再查一次仍然打库」，
// 而且与写操作结尾的 delCache 协程存在竞态（delCache 先跑完、setCache 后落盘的旧数据会长期驻留）。
func (this *Comment) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		facade.Cache.Set(cacheName, data)
	}
}

// sanitizeCommentImages - 校验并规范化评论配图
//
// 只做「结构 / 数量 / 协议」校验，不做域名白名单（站点可能配 COS 与本地多套域名，
// 白名单会误伤），但必须挡掉 javascript: / data: / vbscript: 这类能进 <img src> 的伪协议。
//
// 允许三种形态：{{cos}} / {{localhost}} 域名模板（见 model.DomainTemp）、站内相对路径、http(s) 绝对地址。
// 每一项可以是「封面」或「封面::视频」（实况图），两段分别校验。
func sanitizeCommentImages(raw string) (string, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return "", true
	}

	parts := strings.Split(text, ",")
	list := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))

	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}

		for _, one := range strings.Split(item, "::") {
			url := strings.TrimSpace(one)
			if url == "" || strings.ContainsAny(url, " \t\r\n\"'<>") {
				return "", false
			}
			lower := strings.ToLower(url)
			switch {
			case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
			case strings.HasPrefix(url, "{{"), strings.HasPrefix(url, "/"):
			default:
				return "", false
			}
		}

		if seen[item] {
			continue
		}
		seen[item] = true
		list = append(list, item)
		if len(list) > maxCommentImages {
			return "", false
		}
	}

	return strings.Join(list, ","), true
}

// commentContentExceeds - 正文是否超过站点配置的上限（maxLength <= 0 表示不限制）
//
// @ 与话题的规范化会把内容写长（@张三 → @[张三](/author/12)），
// 所以 create / update 都必须在规范化**之后**再校验一次，否则会撞 varchar(1024) 上限。
func commentContentExceeds(content string, maxLength int) bool {
	return maxLength > 0 && utf8.RuneCountInString(content) > maxLength
}

// commentBindTarget - 评论目标内容的存在性 / 发布状态 / 可见性校验
//
// 四种绑定类型走同一套规则（文档此前单独实现过，这里统一）：
//   - 内容必须存在，否则可以对着任意 id 评论，还能借「不存在 / 评论成功」的差异枚举未发布内容；
//   - 作者本人与管理员可以评论自己的草稿（联调 / 后台处理）；
//   - 其他人只能评论「已发布且审核通过」的内容；
//   - 文档额外过 docsCanView（登录可见 / 指定角色组）。
//
// 返回的 item 供调用方复用 uid / title，errMsg 非空时调用方直接返回 400 / 401。
func (this *Comment) commentBindTarget(ctx *gin.Context, bindType string, bindId int) (map[string]any, string) {
	if bindId <= 0 {
		return nil, facade.Lang(ctx, "%s 不能为空！", "bind_id")
	}

	var table any
	switch bindType {
	case "article":
		table = &model.Article{}
	case "page":
		table = &model.Pages{}
	case "moments":
		table = &model.Moments{}
	case "docs":
		table = &model.Docs{}
	default:
		return nil, facade.Lang(ctx, "不支持的评论目标类型！")
	}

	item, _ := facade.DB.Model(table).Where("id", bindId).Find()
	data := cast.ToStringMap(item)
	if len(data) == 0 {
		return nil, facade.Lang(ctx, "评论目标不存在！")
	}

	// 文档：不区分作者，未发布 / 无权查看一律拒绝（与改造前一致）
	if bindType == "docs" {
		var docs model.Docs
		facade.DB.Model(&model.Docs{}).Where("id", bindId).Scan(&docs)
		if docs.Status != 1 {
			return nil, facade.Lang(ctx, "文档未发布！")
		}
		if !docsCanView(ctx, &docs) {
			return nil, facade.Lang(ctx, "请先登录后查看！")
		}
		return data, ""
	}

	// 作者本人 / 管理员：草稿与未过审内容也允许评论
	uid := cast.ToInt(data["uid"])
	if user := this.meta.user(ctx); user.Id > 0 && (uid == user.Id) {
		return data, ""
	}
	if this.canAuditAll(ctx) {
		return data, ""
	}

	// Pages 没有 status 字段（页面创建即发布），只有 audit
	if bindType != "page" && cast.ToInt(data["status"]) != 1 {
		return nil, facade.Lang(ctx, "内容未发布，暂时无法评论！")
	}
	if cast.ToInt(data["audit"]) != AuditPassed {
		return nil, facade.Lang(ctx, "内容未通过审核，暂时无法评论！")
	}

	return data, ""
}

// commentDescendantIds - 收集给定评论的全部子孙 id（BFS，按层批量查，最多 maxCommentDepth 层）
//
// 用于删除 / 恢复时的级联处理：只删父评论会让子回复的 pid 指向一条不存在的记录，
// 而 flat 只取 pid=0 的顶级评论，这些子回复从此永远不可达。
func (this *Comment) commentDescendantIds(rootIds []any, withTrashed, onlyTrashed bool) []any {
	all := make([]any, 0, len(rootIds)*4)
	if len(rootIds) == 0 {
		return all
	}

	visited := make(map[int]bool, len(rootIds))
	level := make([]int, 0, len(rootIds))

	for _, raw := range rootIds {
		id := cast.ToInt(raw)
		if id <= 0 || visited[id] {
			continue
		}
		visited[id] = true
		level = append(level, id)
	}
	for depth := 0; depth < maxCommentDepth && len(level) > 0; depth++ {
		raw, _ := facade.DB.Model(&model.Comment{}).
			OnlyTrashed(onlyTrashed).
			WithTrashed(withTrashed).
			WhereIn("pid", level).
			Column("id")

		next := make([]int, 0, len(level)*2)
		for _, item := range cast.ToSlice(raw) {
			var id int
			switch value := item.(type) {
			case map[string]any:
				id = cast.ToInt(value["id"])
			default:
				id = cast.ToInt(value)
			}
			if id <= 0 || visited[id] {
				continue
			}
			visited[id] = true
			next = append(next, id)
			all = append(all, id)
		}
		level = next
	}

	return all
}

// commentReplyTree - 一次取出这批父评论的全部子孙回复，按 pid 分组返回
//
// 旧实现是「每个父评论递归一次 replies()」，而 replies() 每层一条 SQL：
// 一页 20 条评论 × 3 层回复就是 60+ 次查询，再叠加 20 次 WhereIn。
// 这里改成按层 WhereIn 批量查，整页固定只要「实际层数」次查询，并用 visited 熔断脏数据的环。
func (this *Comment) commentReplyTree(rootIds []int, field any) map[int][]map[string]any {
	result := make(map[int][]map[string]any)
	if len(rootIds) == 0 {
		return result
	}

	nodes := make(map[int]map[string]any)
	children := make(map[int][]int)
	visited := make(map[int]bool, len(rootIds)*2)
	for _, id := range rootIds {
		visited[id] = true
	}

	level := rootIds
	for depth := 0; depth < maxCommentDepth && len(level) > 0; depth++ {
		raw, _ := facade.DB.Model(&model.Comment{}).
			WhereIn("pid", level).
			Where("audit", AuditPassed).
			Order("create_time asc").
			Select()

		rows := cast.ToSlice(raw)
		projected := cast.ToSlice(utils.ArrayMapWithField(raw, field))

		next := make([]int, 0, len(level)*2)
		for index, item := range rows {
			row := cast.ToStringMap(item)
			id := cast.ToInt(row["id"])
			pid := cast.ToInt(row["pid"])
			if id <= 0 || visited[id] {
				continue
			}
			visited[id] = true

			node := row
			if index < len(projected) {
				if mapped, ok := projected[index].(map[string]any); ok && len(mapped) > 0 {
					node = mapped
				}
			}

			nodes[id] = node
			children[pid] = append(children[pid], id)
			next = append(next, id)
		}
		level = next
	}

	var build func(pid int) []map[string]any
	build = func(pid int) []map[string]any {
		list := make([]map[string]any, 0, len(children[pid]))
		for _, id := range children[pid] {
			node := nodes[id]
			if node == nil {
				continue
			}
			if sub := build(id); len(sub) > 0 {
				node["replies"] = sub
			}
			list = append(list, node)
		}
		return list
	}

	for _, id := range rootIds {
		if sub := build(id); len(sub) > 0 {
			result[id] = sub
		}
	}

	return result
}

// canAuditAll - 是否可见全部审核状态的评论
//
// 只有管理员（root 分组，或被授予该接口权限点的后台用户）能看「待审核 / 未通过」；
// 普通用户与访客只能看 audit=1 的评论。
func (this *Comment) canAuditAll(ctx *gin.Context) bool {
	return this.meta.permit(ctx)
}

// withAuditFilter - 非管理员只保留「审核通过」的评论
func (this *Comment) withAuditFilter(query *facade.ModelStruct, ctx *gin.Context) *facade.ModelStruct {
	if this.canAuditAll(ctx) {
		return query
	}
	return query.Where("audit", AuditPassed)
}

// auditCacheName - 缓存名带上「审核可见性」视角
//
// 缓存名只由请求参数生成、不区分身份：若管理员先请求，会把「含待审核」的结果写进缓存，
// 随后游客命中同一 key 就能看到待审核评论。这里把视角拼进 key，两种视角各自独立缓存。
func (this *Comment) auditCacheName(ctx *gin.Context) string {
	if this.canAuditAll(ctx) {
		return this.cache.name(ctx) + "&audit=all"
	}
	return this.cache.name(ctx) + "&audit=passed"
}

func (this *Comment) processFieldValue(val any) any {
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

func (this *Comment) maskItem(item map[string]any) {
	if ip, ok := item["ip"].(string); ok && ip != "" {
		item["ip"] = facade.Comm.MaskIP(ip)
	}
	if agent, ok := item["agent"].(string); ok && agent != "" {
		item["agent"] = facade.Comm.MaskUA(agent)
	}
	if replies, ok := item["replies"]; ok {
		switch v := replies.(type) {
		case []map[string]any:
			for _, r := range v {
				this.maskItem(r)
			}
		case []any:
			for _, r := range v {
				if rm, ok := r.(map[string]any); ok {
					this.maskItem(rm)
				}
			}
		}
	}
}

func (this *Comment) maskCommentData(ctx *gin.Context, data any) any {
	// 口径与 canAuditAll 保持一致：有「评论管理」权限点的后台用户也应看到原始 IP / UA
	if data == nil || this.canAuditAll(ctx) {
		return data
	}
	switch v := data.(type) {
	case map[string]any:
		this.maskItem(v)
		return v
	case []map[string]any:
		for i := range v {
			this.maskItem(v[i])
		}
		return v
	case []any:
		for i := range v {
			if m, ok := v[i].(map[string]any); ok {
				this.maskItem(m)
				v[i] = m
			}
		}
		return v
	}
	return data
}

func (this *Comment) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Comment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	// 验证：聚合统计同样不能把待审核评论算进去
	query = this.withAuditFilter(query, ctx)

	fields := utils.Unity.Keys(params["field"])
	if utils.Is.Empty(fields) {
		return nil, ""
	}

	cacheName := this.auditCacheName(ctx)
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

func (this *Comment) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"sum":    this.sum,
		"min":    this.min,
		"max":    this.max,
		"rand":   this.rand,
		"count":  this.count,
		"counts": this.counts,
		"column": this.column,
		"flat":   this.flat,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Comment) IPOST(ctx *gin.Context) {
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) IPUT(ctx *gin.Context) {
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) IDEL(ctx *gin.Context) {
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

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

func (this *Comment) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Comment) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "comment"})
}

func (this *Comment) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Comment{}

	for key, val := range params {
		if utils.In.Array(key, commentAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.auditCacheName(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)
		// 单条查询同样要过审核：否则知道 id 就能直接读到待审核 / 未通过的评论
		query = this.withAuditFilter(query, ctx)
		item, _ := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论同样支持登录可见 / 付费可见，无权查看时清空正文
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Comment{}
	for key, val := range params {
		if utils.In.Array(key, commentAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 非管理员只看「审核通过」的评论（审核未通过 / 待审核的评论对前台不可见，口径同文章 / 动态）
	query = this.withAuditFilter(query, ctx)

	// count 与列表一起缓存：否则缓存命中时仍要走一次 COUNT 聚合，列表缓存等于白做
	// v=2 是「列表 + count」的新结构，旧 key 里的纯列表数据不再命中
	cacheName := this.auditCacheName(ctx) + "&v=2"
	count := int64(0)

	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		if cache, valid := cached.(map[string]any); valid {
			data = cache["data"]
			count = cast.ToInt64(cache["count"])
		} else {
			data = cached
		}
	}

	if data == nil {
		count, _ = query.Where(table).Count()
		item, _ := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, map[string]any{"data": data, "count": count})
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论列表同样按条脱敏（付费评论只保留占位信息）
	data = applyScopeData(ctx, data, "comment")

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

func (this *Comment) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	mold := facade.DB.Model(&[]model.Comment{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		mold.Where("id", "NOT IN", except)
	}
	mold = this.buildQuery(mold, params)
	mold = this.withAuditFilter(mold, ctx)

	// 随机取数：不用 ORDER BY RAND()（大表上等于全表扫 + 临时表排序），
	// 改为「先取总数、随机起始页」，代价固定为两次范围扫描。
	var item []map[string]any
	mold = mold.Limit(limit)
	if total, _ := mold.Count(); total > int64(limit) {
		page := rand.Intn(int(total)/limit) + 1
		item, _ = mold.Page(page).Select()
	} else {
		item, _ = mold.Select()
	}

	data := this.maskCommentData(ctx, utils.ArrayMapWithField(item, params["field"]))
	// 可见范围：随机获取的评论同样脱敏
	data = applyScopeData(ctx, data, "comment")

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *Comment) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// commentSummary - 评论正文摘要（审核通知的邮件 / 站内信文案用）
func commentSummary(content string) string {
	text := cleanSearchText(content)
	if utils.Is.Empty(text) {
		return "（无正文）"
	}

	runes := []rune(text)
	if len(runes) > 30 {
		return string(runes[:30]) + "..."
	}
	return text
}

// commentBindTitle - 评论所属内容的展示名（邮件 / 站内信里标明评论来源）
//
// 文章 / 页面取标题，动态没有标题则用正文摘要（与评论通知邮件模板的口径一致）。
func commentBindTitle(bindType string, bindId int) string {
	if bindId <= 0 {
		return ""
	}

	var table any
	switch bindType {
	case "article":
		table = &model.Article{}
	case "page":
		table = &model.Pages{}
	case "moments":
		table = &model.Moments{}
	case "docs":
		table = &model.Docs{}
	default:
		return ""
	}

	item, _ := facade.DB.Model(table).Where("id", bindId).Find()
	data := cast.ToStringMap(item)

	if title := strings.TrimSpace(cast.ToString(data["title"])); !utils.Is.Empty(title) {
		return title
	}

	return commentSummary(cast.ToString(data["content"]))
}

// notifyCommentAudit - 评论审核状态变化时发通知（邮件 + 站内信）
//
// 与文章 / 动态 / 页面共用一套规则（见 controller/mail-notify.go）：
//   - 进入待审核 → 通知管理员（comment.pending），站内信正文带上评论者身份；
//   - 审核通过 → 通知评论者（comment.passed）；
//   - 审核未通过 → 通知评论者（comment.rejected），并带上驳回原因。
//
// 站内信固定发，邮件受「系统设置 → 邮件通知 → 内容审核」的场景开关控制；
// 跳转目标用被评论内容的 bind_type / bind_id，点消息直接回到内容页。
func notifyCommentAudit(uid int, content string, audit int, reason, bindType string, bindId int) {
	if uid <= 0 {
		return
	}

	summary := commentSummary(content)
	source := commentBindTitle(bindType, bindId)
	nowText := model.MailNotifyTime()
	reason = strings.TrimSpace(reason)

	lines := []string{"评论：" + summary}
	if !utils.Is.Empty(source) {
		lines = append(lines, "来源："+source)
	}
	lines = append(lines, "时间："+nowText)
	if audit == AuditReject && !utils.Is.Empty(reason) {
		lines = append(lines, "驳回原因："+reason)
	}

	// 站内信正文：驳回原因放最前面（消息列表内容区只有 2 行高度），跳转目标为被评论内容
	notifyContent := "时间：" + nowText
	if !utils.Is.Empty(source) {
		notifyContent = "来源：" + source + " · " + notifyContent
	}
	if audit == AuditReject && !utils.Is.Empty(reason) {
		notifyContent = "驳回原因：" + reason + " · " + notifyContent
	}

	switch audit {
	case AuditPending:
		title := "有新的评论待审核"
		account, nickname := model.MailNotifyUserIdentity(uid)
		model.MailNotifyAdmin("comment.pending", title, append(model.MailNotifyUserInfo(uid), lines...)...)
		model.NotifyAdmins(uid, model.NotificationTypeComment, title,
			"评论者："+nickname+"（"+account+"） · "+notifyContent, bindType, bindId)
	case AuditPassed:
		title := "您的评论已通过审核"
		model.MailNotifyUser(uid, "comment.passed", title, lines...)
		model.CreateUserNotify(uid, 0, model.NotificationTypeComment, title, notifyContent, bindType, bindId)
	case AuditReject:
		title := "您的评论未通过审核"
		model.MailNotifyUser(uid, "comment.rejected", title, lines...)
		model.CreateUserNotify(uid, 0, model.NotificationTypeComment, title, notifyContent, bindType, bindId)
	}
}

func (this *Comment) create(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"bind_type": "article",
	})
	err := validator.NewValid("comment", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 绑定类型白名单：未知类型不查内容直接入库，会给评论表塞进任意 bind_type / bind_id 的脏数据
	bindType := cast.ToString(params["bind_type"])
	if !utils.In.Array(bindType, commentBindTypes) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的评论目标类型！"), 400)
		return
	}

	bindId := cast.ToInt(params["bind_id"])

	// 目标内容一次校验 + 一次查询：后面的邮件 / 站内信通知都复用这里的 title / uid，
	// 改造前同一条文章在 create 里会被查 4 次（校验一次、邮件取标题一次、邮件取作者一次、通知一次）
	bindTarget, bindErr := this.commentBindTarget(ctx, bindType, bindId)
	if bindErr != "" {
		this.json(ctx, nil, bindErr, 400)
		return
	}

	// 评论开关优先级：全局 COMMENT.allow < 模块总开关（ARTICLE/MOMENTS/PAGE）< 内容自身 json.comment
	// 内容 / 模块取值：0 继承、1 允许/显示、2 禁止/隐藏；全局 COMMENT.allow：0 关闭、1 开启
	comment := cast.ToStringMap(cast.ToStringMap(bindTarget["json"])["comment"])
	if comment == nil {
		comment = map[string]any{}
	}
	moduleComment := this.config(bindType, "comment")

	if bindType == "docs" {
		// 项目级开关直接落成 allow/show，不走「0 继承」那套：
		// 项目禁用评论就是 2，启用就是 1，优先级低于模块总开关
		comment = facade.H{"allow": 1, "show": 1}
		if !model.DocsProjectCommentEnabled(cast.ToString(bindTarget["version"])) {
			comment["allow"] = 2
		}
	}

	// 模块总开关为「禁止 / 隐藏」时，覆盖内容自身的设置
	if cast.ToInt(moduleComment["allow"]) == 2 {
		comment["allow"] = 2
	}
	if cast.ToInt(moduleComment["show"]) == 2 {
		comment["show"] = 2
	}

	// 内容未设置时继承模块总开关
	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = moduleComment["allow"]
	}
	if cast.ToInt(comment["show"]) == 0 {
		comment["show"] = moduleComment["show"]
	}

	// 模块总开关未配置时，回落到全局评论开关（COMMENT.allow：1 开启 / 0 关闭）
	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = this.config("comment")["allow"]
	}

	if cast.ToInt(comment["allow"]) == 2 || cast.ToInt(comment["allow"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "评论功能已关闭！"), 400)
		return
	}
	// 隐藏评论时整块不展示，同样不允许发表
	if cast.ToInt(comment["show"]) == 2 {
		this.json(ctx, nil, facade.Lang(ctx, "评论功能已关闭！"), 400)
		return
	}

	commentConfig := this.config("comment")

	// 评论限流：固定窗口计数，用缓存的原子自增
	//
	// 旧实现是 Get → 算 +1 → Set：并发请求会读到同一个旧值，等于没限流；
	// key 里带 ctx.ClientIP() 也不合适 —— Gin 的 ClientIP 取 X-Forwarded-For，
	// 伪造请求头就能换一个计数桶，所以只按 uid 计。
	rateLimit := cast.ToStringMap(commentConfig["rate_limit"])
	if cast.ToInt(rateLimit["enabled"]) == 1 {
		maxCount := cast.ToInt(rateLimit["max_count"])
		timeWindow := cast.ToInt(rateLimit["time_window"])

		if maxCount > 0 {
			if timeWindow <= 0 {
				timeWindow = 60
			}

			windowKey := time.Now().Unix() / int64(timeWindow)
			cacheKey := fmt.Sprintf("comment:rate_limit:%d:%d", user.Id, windowKey)

			current, ok := facade.Cache.Incr(cacheKey, 1)
			if !ok {
				// 缓存不可用时放行：限流是附加保护，不能反过来把正常评论挡死
				current = 0
			}
			if current == 1 {
				// 窗口内的第一次计数挂上过期时间，窗口过去后自动清零
				facade.Cache.Expire(cacheKey, time.Duration(timeWindow)*time.Second)
			}

			if current > int64(maxCount) {
				// 被拒绝的这次不占额度，否则窗口内每失败一次都会少一次机会
				facade.Cache.Incr(cacheKey, -1)
				this.json(ctx, nil, facade.Lang(ctx, "评论过于频繁，请稍后再试！"), 429)
				return
			}
		}
	}

	maxLength := cast.ToInt(commentConfig["max_length"])
	if commentContentExceeds(cast.ToString(params["content"]), maxLength) {
		this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
		return
	}

	if cast.ToInt(commentConfig["require_chinese"]) == 1 {
		hasChinese := false
		for _, r := range cast.ToString(params["content"]) {
			if unicode.Is(unicode.Scripts["Han"], r) {
				hasChinese = true
				break
			}
		}
		if !hasChinese {
			this.json(ctx, nil, facade.Lang(ctx, "评论内容必须包含中文！"), 400)
			return
		}
	}

	agent := this.header(ctx, "User-Agent")
	if len(agent) > 511 {
		agent = agent[:511]
	}
	table := model.Comment{
		Uid:        user.Id,
		Agent:      agent,
		Ip:         ctx.ClientIP(),
		CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix(),
	}

	// 审核：关闭审核 → 直接「通过」；开启审核 → 「待审核」（口径见 audit.go，开关在「系统设置 → 评论配置」）。
	// 管理员自己发的评论直接通过（不必审核自己）；管理员在 payload 里显式指定 audit 时，
	// 下面遍历参数时会覆盖这里的判定（allow 里 audit 只对 root 开放）。
	// ⚠ 必须直接赋值：Struct.Set 按小写 tag（json:"audit"）匹配，传 "Audit" 会静默失效
	// 是否开启本地审核：只查一次库，后面「要不要再调云端机审」也用它
	commentAuditSwitch := contentAuditSwitch("COMMENT")
	table.Audit = auditForCreate(commentAuditSwitch && !this.canAuditAll(ctx), false)

	for key, val := range params {
		if utils.In.Array(key, commentAllowFieldsSlice) {
			switch utils.Get.Type(val) {
			case "string":
				if key == "content" || key == "text" {
					if facade.Comm.DetectXSS(cast.ToString(val)) {
						this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
						return
					}
					val = facade.Comm.SanitizeHTML(cast.ToString(val))
				}
			}
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	// 配图：数量 / 协议校验（images 是 text 字段，前端限制可以绕过）
	images, imagesValid := sanitizeCommentImages(table.Images)
	if !imagesValid {
		this.json(ctx, nil, facade.Lang(ctx, "配图格式不正确或数量超出限制！"), 400)
		return
	}
	table.Images = images

	// 敏感词：评论高频，按规则动作处理 —— block / limit 拒绝，review 进待审核，
	// warn / log 放行（warn 的提示随成功消息返回，命中审计由 model 层统一记录）。
	sensitive := (&model.Sensitive{}).Decide(model.SceneComment, user.Id, table.Content)
	if sensitive.Action != model.ActionAllow {
		if sensitive.Blocked() {
			this.json(ctx, nil, facade.Lang(ctx, sensitive.UserMessage()), 400)
			return
		}
		if sensitive.Action == model.ActionReview && !this.canAuditAll(ctx) {
			table.Audit = AuditPending
		}
	}

	// 评论不支持付费可见：scope=3 一律降级为「登录可见」
	// （旧前端 / 历史数据里可能带 3，统一在这里收敛，避免评论出现积分墙）
	table.Scope = model.ScopeForBind("comment", table.Scope)

	// @ 提及：只匹配作者**已关注**的人，把 @昵称 规范化成 @[昵称](/author/uid)（前端渲染成可点击的链接）。
	// 规范化会让内容变长（@张三 → @[张三](/author/12)），所以长度要按转换后的结果再校验一次，
	// 避免突破 Comment.Content 的 varchar 上限导致插入失败。
	mentionTargets := []model.MentionTarget{}
	table.Content, mentionTargets = (&model.Mention{}).Normalize(table.Content, user.Id)

	// 通知对象同样要限流：候选来自「我关注的人」（上限 500），一条评论能把关注列表 @ 满，
	// 于是几百条站内信 + 邮件在创建时被一次性打出去
	if len(mentionTargets) > maxCommentMentions {
		mentionTargets = mentionTargets[:maxCommentMentions]
	}

	// 话题：#话题# 规范化为 #[话题名](/topic/12)，前端渲染成可点击链接。
	// 评论表没有 tags 字段，所以只做链接、不参与话题内容聚合（见 model/topics.go）。
	// 同样放在长度校验之前：规范化会让内容变长。
	table.Content, _ = (&model.Topic{}).Parse(table.Content)

	if commentContentExceeds(table.Content, maxLength) {
		this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
		return
	}

	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 云端文本审核（百度）：评论高频，走异步队列消费，结果命中 block / review 时回写 audit。
	// 只有「本地审核关闭」时才需要机审兜底：开启审核时评论本来就会进人工队列
	if !commentAuditSwitch {
		model.SubmitCloudAudit(model.SceneComment, user.Id, "comment", table.Id, table.Content)
	}

	// 审核通知：评论进入待审核时提醒管理员（与文章 / 动态同一套规则）
	// 站内信始终发，邮件受「系统设置 → 邮件通知 → 内容审核 → 评论待审核」控制
	if cast.ToInt(table.Audit) == AuditPending {
		go notifyCommentAudit(user.Id, table.Content, AuditPending, "", table.BindType, table.BindId)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "更新用户经验协程发生错误")
			}
		}()
		_ = (&model.EXP{}).Add(model.EXP{
			Uid:         user.Id,
			Type:        "comment-create",
			BindId:      table.Id,
			BindType:    table.BindType,
			Description: "发表评论奖励",
		})
		_ = (&model.Integral{}).Add(model.Integral{
			Uid:  user.Id,
			Type: "comment",
		})

		// 勋章：发表评论后按条件判定（如「累计 10 条评论」）
		model.BadgeAutoGrant(user.Id)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "发送评论邮件通知协程发生错误")
			}
		}()

		// 评论 / 回复的邮件开关已统一到「系统设置 → 邮件通知」：
		// 总开关 + 场景开关（comment.notify / comment.reply），分批投递与失败重试由邮箱队列负责；
		// 这里只组装邮件内容并交给对应场景发送（model.MailNotifyComment / MailNotifyReply）
		if model.MailNotifyEnabled("") {

			userInfo, _ := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			userEmail := cast.ToString(cast.ToStringMap(userInfo)["email"])
			facade.Log.Info(map[string]any{"user_id": user.Id, "user_email": facade.Comm.MaskEmail(userEmail), "user_name": cast.ToString(cast.ToStringMap(userInfo)["name"])}, "获取评论用户信息")

			// 标题直接复用 create 开头解析好的 bindTarget，不再按绑定类型重复查一次内容表
			bindLabel := map[string]string{
				"article": "文章",
				"page":    "页面",
				"moments": "动态",
				"docs":    "文档",
			}[table.BindType]

			title := strings.TrimSpace(cast.ToString(bindTarget["title"]))
			if title == "" {
				title = commentSummary(cast.ToString(bindTarget["content"]))
			}

			createdAt := time.Now().Format("2006-01-02 15:04:05")
			facade.Log.Info(map[string]any{"local_time": createdAt}, "获取服务器本地时间")

			commentInfo := map[string]any{
				"content":      table.Content,
				"created_at":   createdAt,
				"author_name":  cast.ToString(cast.ToStringMap(userInfo)["nickname"]),
				"author_email": userEmail,
				// 评论者账号：邮件模板里的 ${author_account}（不传会原样留下占位符）
				"author_account": commenterAccount(cast.ToStringMap(userInfo)),
				"ip":             table.Ip,
				"bind_type":      table.BindType,
				"bind_id":        table.BindId,
				"bind_label":     bindLabel,
				"title":          title,
			}
			facade.Log.Info(map[string]any{"comment_id": table.Id, "bind_type": table.BindType, "bind_id": table.BindId}, "评论通知处理")

			// 内容作者：bindTarget 已带 uid，这里只需要一次用户查询
			var authorEmail string
			if authorId := cast.ToInt(bindTarget["uid"]); authorId > 0 {
				author, _ := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
				authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
				facade.Log.Info(map[string]any{"bind_id": table.BindId, "author_id": authorId, "author_email": facade.Comm.MaskEmail(authorEmail)}, "获取内容作者信息")
			}

			// 评论通知：发给内容作者（场景 comment.notify）
			// 作者邮箱无效、或评论者本人就是作者时不发（与原先的跳过规则一致）
			if utils.Is.Email(authorEmail) && authorEmail != userEmail {
				model.MailNotifyComment(authorEmail, commentInfo)
			}

			// 回复通知：发给被回复的人（场景 comment.reply）
			// 回复自己、或被回复人没有邮箱时不发；
			// 「仅作者可见」的评论只应让内容作者看到，不能让被回复的其他人拿到正文
			if table.Pid > 0 && cast.ToInt(table.Scope) != model.ScopeAuthor {
				parentComment, _ := facade.DB.Model(&model.Comment{}).Where("id", table.Pid).Find()
				if !utils.Is.Empty(parentComment) {
					parentUid := cast.ToInt(cast.ToStringMap(parentComment)["uid"])
					parentUser, _ := facade.DB.Model(&model.Users{}).Where("id", parentUid).Find()
					parentEmail := cast.ToString(cast.ToStringMap(parentUser)["email"])

					if utils.Is.Email(parentEmail) && parentUid != user.Id {
						model.MailNotifyReply(parentEmail, commentInfo)
					}
				}
			}
		} else {
			facade.Log.Info(nil, "邮件通知总开关未开启，跳过评论邮件通知")
		}
	}()

	// 创建通知 - 新回复通知给文章/页面/动态作者
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "创建回复通知协程发生错误")
			}
		}()

		// 内容作者与标题都来自 create 开头解析好的 bindTarget，不再重复查内容表
		authorId := cast.ToInt(bindTarget["uid"])
		bindTitle := strings.TrimSpace(cast.ToString(bindTarget["title"]))
		if bindTitle == "" {
			bindTitle = commentSummary(cast.ToString(bindTarget["content"]))
		}

		if authorId > 0 && authorId != user.Id {
			title := "收到新回复"
			notifUserInfo, _ := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			notifContent := cast.ToString(cast.ToStringMap(notifUserInfo)["nickname"]) + " 回复了你"
			if bindTitle != "" {
				notifContent += "的" + bindTitle
			}

			_, _ = (&model.Notification{}).CreateNotification(
				authorId, user.Id, "comment", title, notifContent, table.BindType, table.BindId,
			)
		}
	}()

	// @ 提及通知：只发给「被 @ 到的人」（内容作者那条已由上面的逻辑发出）。
	// 评论待审核（audit=0）时不发：评论可能不被展示，通知过去也看不到内容；
	// 「仅作者可见」同样不发 —— 被 @ 的人看不到正文，通知只会泄露内容。
	if cast.ToInt(table.Audit) == 1 && cast.ToInt(table.Scope) != model.ScopeAuthor {
		go (&model.Mention{}).NotifyNew(mentionTargets, user.Id, "评论", table.BindType, table.BindId, table.Content, "")
	}

	// 返回本次写入的审核状态：前台据此决定提示语（开启审核时提示「审核通过后显示」）
	// 并插入一条「待审核」占位 —— 待审核的评论不在列表接口的返回里（非管理员只查 audit=1）
	msg := facade.Lang(ctx, "创建成功！")
	if notice := sensitive.NoticeMessage(); notice != "" {
		msg += "（" + notice + "）"
	}

	this.json(ctx, gin.H{
		"id":      table.Id,
		"audit":   table.Audit,
		"pending": table.Audit == AuditPending,
	}, msg, 200)
}

// commenterAccount 评论者账号（评论邮件模板里的 ${author_account}，空时用昵称 / 「—」兜底）
func commenterAccount(commenter map[string]any) string {
	if account := strings.TrimSpace(cast.ToString(commenter["account"])); !utils.Is.Empty(account) {
		return account
	}
	if nickname := strings.TrimSpace(cast.ToString(commenter["nickname"])); !utils.Is.Empty(nickname) {
		return nickname
	}
	return "—"
}

func (this *Comment) update(ctx *gin.Context) {
	params := this.params(ctx)

	// 批量审核：后台列表勾选后一次提交 { ids: [...], audit: 1, reason: "" }
	//
	// 不单独开 audit-batch 接口 —— 批量只是「把同一个 audit 应用到一组 id」，
	// 单开接口要在 auth-rules.go 里多登记一条规则（漏登记会让所有人 403），
	// 前端也要多记一个地址，而 update 本来就是同一个语义。
	if batchIds := utils.Unity.Ids(params["ids"]); len(batchIds) > 0 {
		this.auditBatch(ctx, batchIds)
		return
	}

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	// 后台「通过 / 驳回」只提交 {id, audit, reason}，不带 content；
	// 而 comment 验证器把 content 标成了 required（发表评论必填），
	// 无条件校验会把这两个按钮直接打成 400「content 不能为空！」。
	// 因此只有真的提交了正文（编辑评论）时才跑完整校验。
	var err error
	if _, ok := params["content"]; ok {
		if err = validator.NewValid("comment", params); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	}

	// 口径统一用 canAuditAll（= meta.permit）：只认 root() 会把「有评论管理权限点、
	// 但分组不是 root」的后台用户判成普通用户，界面上能看见待审核列表却点不动审核按钮
	admin := this.canAuditAll(ctx)
	table := model.Comment{}
	allow := []any{"content", "images", "editor", "json", "text"}
	async := utils.Async[map[string]any]()

	// audit / reason 只对管理员开放：这是后台列表「通过 / 驳回」的操作入口。
	// 无权限却提交了这两个字段时显式 403 —— 以前是被 allow 过滤掉、然后照样返回「更新成功」。
	_, hasAudit := params["audit"]
	_, hasReason := params["reason"]
	if !admin && (hasAudit || hasReason) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}
	if admin {
		allow = append(allow, "pid", "bind_id", "bind_type", "audit", "reason")
	}

	for key, val := range params {
		if utils.In.Array(key, allow) {
			switch utils.Get.Type(val) {
			case "string":
				if key == "content" || key == "text" {
					if facade.Comm.DetectXSS(cast.ToString(val)) {
						this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
						return
					}
					val = facade.Comm.SanitizeHTML(cast.ToString(val))
				}
			}
			async.Set(key, this.processFieldValue(val))
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	itemData, _ := item.Find()
	if len(itemData) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "评论不存在！"), 404)
		return
	}

	// 未登录时 uid 也是 0，不能靠「0 == 0」命中历史数据里 uid=0 的评论，所以先判登录态
	user := this.user(ctx)
	if !admin && (user.Id == 0 || cast.ToInt(itemData["uid"]) != user.Id) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	payload := async.Result()

	// 审核通过时清掉上一次的驳回原因，避免列表里残留旧原因
	if cast.ToInt(payload["audit"]) == AuditPassed {
		payload["reason"] = ""
	}

	// 配图：与新建同一套数量 / 协议校验（images 直通 text 字段，前端限制可绕过）
	if raw, ok := payload["images"]; ok {
		images, valid := sanitizeCommentImages(cast.ToString(raw))
		if !valid {
			this.json(ctx, nil, facade.Lang(ctx, "配图格式不正确或数量超出限制！"), 400)
			return
		}
		payload["images"] = images
	}

	// @ 提及：编辑评论时同样规范化（匹配范围仍是评论作者已关注的人），
	// 否则手动输入的 @昵称 永远变不成链接
	// sensitive 在块外声明：warn 的提示要拼进成功消息
	var sensitive model.SensitiveDecision
	if _, ok := payload["content"]; ok {
		payload["content"], _ = (&model.Mention{}).Normalize(cast.ToString(payload["content"]), cast.ToInt(itemData["uid"]))
		// 话题：与新建一致，只做链接不做聚合（评论没有 tags 字段）
		payload["content"], _ = (&model.Topic{}).Parse(cast.ToString(payload["content"]))

		// 规范化会让内容变长（@张三 → @[张三](/author/12)），长度要按转换后的结果再校验一次，
		// 否则编辑评论会撞 Comment.Content 的 varchar(1024) 上限
		maxLength := cast.ToInt(this.config("comment")["max_length"])
		if commentContentExceeds(cast.ToString(payload["content"]), maxLength) {
			this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
			return
		}

		// 敏感词：编辑评论同样按场景判定；命中 review 的编辑重新进入待审核
		sensitive = (&model.Sensitive{}).Decide(model.SceneComment, cast.ToInt(itemData["uid"]), cast.ToString(payload["content"]))
		if sensitive.Blocked() {
			this.json(ctx, nil, facade.Lang(ctx, sensitive.UserMessage()), 400)
			return
		}
		if sensitive.Action == model.ActionReview && !admin {
			payload["audit"] = AuditPending
		}

		// 非管理员修改正文 → 重新进待审核（站点开启评论审核时）
		//
		// 否则「先发一条正常评论通过审核、再编辑成违规内容」可以完全绕过人工审核。
		// 只在正文真的变化时降级，避免前端原样重提交把已通过的评论白白打回。
		if !admin && contentAuditSwitch("COMMENT") && cast.ToString(payload["content"]) != cast.ToString(itemData["content"]) {
			payload["audit"] = AuditPending
		}
	}

	_, err = item.Scan(&table).Update(payload)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 审核状态变化 → 发通知（邮件 + 站内信）；普通编辑（audit 未进 payload）不打扰任何人。
	// 评论审核通过时还要补发 @ 提及通知：待审核阶段故意不发（评论不展示，点进去看不到），
	// 所以「待审核 → 通过」这一刻必须补 —— 这就是后台「通过」按钮这条路径。
	if rawAudit, ok := payload["audit"]; ok {
		nowAudit := cast.ToInt(rawAudit)
		if prevAudit := cast.ToInt(itemData["audit"]); nowAudit != prevAudit {
			// 后台点「通过 / 驳回」时通常只传 audit（+ reason），正文沿用库里的
			content := cast.ToString(payload["content"])
			if utils.Is.Empty(content) {
				content = cast.ToString(itemData["content"])
			}

			go notifyCommentAudit(
				cast.ToInt(itemData["uid"]), content, nowAudit,
				cast.ToString(payload["reason"]),
				cast.ToString(itemData["bind_type"]), cast.ToInt(itemData["bind_id"]),
			)

			if nowAudit == AuditPassed {
				go model.NotifyMentionsOnCommentApprove(cast.ToInt(params["id"]))
			}
		}
	}

	msg := facade.Lang(ctx, "更新成功！")
	if notice := sensitive.NoticeMessage(); notice != "" {
		msg += "（" + notice + "）"
	}

	this.json(ctx, gin.H{"id": table.Id}, msg, 200)
}

// auditBatch - 批量审核评论（update 带 ids 时的分支，后台列表勾选后一次提交）
//
// 一条条 PUT update 的话，50 条待审核就是 50 次请求 + 50 次全量 delCache；
// 这里合并成一次。只做「状态真的变化」的那部分通知，避免重复点「通过」时
// 给评论者发一堆重复站内信。
//
// 只处理 audit / reason，不接受正文等字段 —— 批量改内容没有业务含义。
func (this *Comment) auditBatch(ctx *gin.Context, ids []any) {
	if !this.canAuditAll(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx)
	audit := cast.ToInt(params["audit"])
	reason := strings.TrimSpace(cast.ToString(params["reason"]))

	if audit != AuditPending && audit != AuditPassed && audit != AuditReject {
		this.json(ctx, nil, facade.Lang(ctx, "审核状态不正确！"), 400)
		return
	}

	// 上限保护：一次勾选上千条会把一条请求拖成长事务
	if len(ids) > maxCommentBatch {
		ids = ids[:maxCommentBatch]
	}

	raw, _ := facade.DB.Model(&model.Comment{}).WithTrashed().WhereIn("id", ids).Select()
	rows := cast.ToSlice(raw)

	if len(rows) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	payload := map[string]any{"audit": audit}
	if audit == AuditPassed {
		// 通过时清掉历史驳回原因，避免列表残留旧原因
		payload["reason"] = ""
	} else if reason != "" {
		payload["reason"] = reason
	}

	done := make([]int, 0, len(rows))

	for _, item := range rows {
		row := cast.ToStringMap(item)
		id := cast.ToInt(row["id"])
		if id <= 0 {
			continue
		}

		if _, err := facade.DB.Model(&model.Comment{}).Where("id", id).Update(payload); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "id": id}, "批量审核：更新评论失败")
			continue
		}

		done = append(done, id)

		if cast.ToInt(row["audit"]) == audit {
			continue
		}

		go notifyCommentAudit(
			cast.ToInt(row["uid"]), cast.ToString(row["content"]), audit,
			cast.ToString(payload["reason"]),
			cast.ToString(row["bind_type"]), cast.ToInt(row["bind_id"]),
		)

		if audit == AuditPassed {
			go model.NotifyMentionsOnCommentApprove(id)
		}
	}

	this.json(ctx, gin.H{"ids": done, "count": len(done)}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Comment) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Comment{}), params)
	query = this.buildQuery(query, params)

	// 非管理员只统计「审核通过」的评论：保证前台显示的评论数与列表一致。
	// 口径必须与 all 的 withAuditFilter（canAuditAll）完全一致，否则后台列表条数对不上
	if !this.canAuditAll(ctx) {
		query = query.Where("audit", AuditPassed)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

// counts 批量评论数（列表页专用）
//
// 一次拿到整页内容的评论数，替代前端对每条动态各调一次 count。
// 口径与 count 一致：只统计审核通过的评论（含二级及更深的回复）。
func (this *Comment) counts(ctx *gin.Context) {
	params := this.params(ctx)

	bindType := cast.ToString(params["bind_type"])
	if bindType == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_type"), 400)
		return
	}
	if !utils.In.Array(bindType, commentBindTypes) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的评论目标类型！"), 400)
		return
	}

	bindIds := cast.ToIntSlice(utils.Unity.Ids(params["bind_ids"]))
	if len(bindIds) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_ids"), 400)
		return
	}

	// 上限保护：批量统计不能变成全表 COUNT
	const maxCounts = 200
	if len(bindIds) > maxCounts {
		bindIds = bindIds[:maxCounts]
	}

	this.json(ctx, gin.H{"counts": model.CommentCounts(bindType, bindIds)}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Comment) sum(ctx *gin.Context) {
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

func (this *Comment) min(ctx *gin.Context) {
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

func (this *Comment) max(ctx *gin.Context) {
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

func (this *Comment) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Comment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	// 取列可能拿到 content，同样按审核状态过滤
	query = this.withAuditFilter(query, ctx)

	cacheName := this.auditCacheName(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		items, _ := query.Select()
		data = utils.ArrayMapWithField(items, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：column 也可能取到 content，同样按条脱敏
	data = applyScopeData(ctx, data, "comment")

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Comment) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{})

	if !this.canAuditAll(ctx) {
		user := this.user(ctx)
		if user.Id == 0 {
			this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
			return
		}
		item.Where("uid", user.Id)
	}

	columnData, _ := item.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 级联子回复：只删父评论会让子回复的 pid 指向一条不存在的记录，
	// 而 flat 只取 pid=0 的顶级评论，这些回复从此永远不可达。
	// 子孙的删除不再带 uid 条件 —— 评论者删自己的评论时，别人在其下的回复同样要一起走。
	children := this.commentDescendantIds(ids, false, false)

	_, err := facade.DB.Model(&model.Comment{}).WhereIn("id", ids).Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	if len(children) > 0 {
		facade.DB.Model(&model.Comment{}).WhereIn("id", children).Delete(children)
		ids = append(ids, children...)
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Comment) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{}).WithTrashed()

	if !this.canAuditAll(ctx) {
		user := this.user(ctx)
		if user.Id == 0 {
			this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
			return
		}
		item.Where("uid", user.Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 物理删除同样级联子孙（含回收站里的），否则会留下再也无法访问的孤立回复
	children := this.commentDescendantIds(ids, true, false)

	_, err := facade.DB.Model(&model.Comment{}).WithTrashed().WhereIn("id", ids).Force().Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	if len(children) > 0 {
		facade.DB.Model(&model.Comment{}).WithTrashed().WhereIn("id", children).Force().Delete(children)
		ids = append(ids, children...)
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Comment) clear(ctx *gin.Context) {
	table := model.Comment{}
	item := facade.DB.Model(&table).OnlyTrashed()

	admin := this.canAuditAll(ctx)
	if !admin {
		user := this.user(ctx)
		if user.Id == 0 {
			this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
			return
		}
		item.Where("uid", user.Id)
	}

	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 管理员清空整个回收站；普通用户只清自己的，但要连带清掉自己评论下的子孙，
	// 否则那些回复会留在回收站里、永远恢复不回来（父评论已经没了）
	if admin {
		if _, err := item.Force().Delete(); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
			return
		}
	} else {
		ids = append(ids, this.commentDescendantIds(ids, true, true)...)
		if _, err := facade.DB.Model(&table).WithTrashed().WhereIn("id", ids).Force().Delete(ids); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
			return
		}
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *Comment) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Comment{}).OnlyTrashed().WhereIn("id", ids)

	if !this.canAuditAll(ctx) {
		user := this.user(ctx)
		if user.Id == 0 {
			this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
			return
		}
		item.Where("uid", user.Id)
	}

	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 恢复父评论时一并恢复它名下的子孙（未删除的 id 对 Restore 是 no-op），
	// 否则回复会留在回收站里，父评论恢复后依然看不到它们
	ids = append(ids, this.commentDescendantIds(ids, true, false)...)

	_, err := facade.DB.Model(&model.Comment{}).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

func (this *Comment) flat(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":      1,
		"bind_type": "article",
		"order":     "create_time desc",
	})

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "bind_id 不能为空！"), 400)
		return
	}

	// 与 create / counts 同一套白名单：避免用一个不存在的 bind_type 反复探测
	if !utils.In.Array(cast.ToString(params["bind_type"]), commentBindTypes) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的评论目标类型！"), 400)
		return
	}

	table := model.Comment{}
	for key, val := range params {
		if key == "bind_id" || key == "bind_type" {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment

	mold := facade.DB.Model(&result).Where("pid", 0)
	mold = this.withTrashOptions(mold, params)
	mold = this.buildQuery(mold, params)
	// 前台评论树只返回「审核通过」的评论：待审核 / 未通过的评论对任何前台访问者
	// （含作者本人与管理员）都不展示，避免把审核中内容当成已发布内容显示。
	// 刚提交的评论由前端本地「待审核」占位提示，审核通过后重新拉取才会出现。
	mold = mold.Where("audit", AuditPassed)

	// key 带 audit=passed 后缀：让加入过滤前写入的旧缓存（可能含待审核评论）自然失效；
	// v=2 是「评论树 + count」的新结构，旧 key 里的纯列表数据不再命中
	cacheName := this.cache.name(ctx) + "&audit=passed&v=2"
	count := int64(0)

	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		if cache, valid := cached.(map[string]any); valid {
			data = cache["data"]
			count = cast.ToInt64(cache["count"])
		} else {
			data = cached
		}
	}

	if data == nil {
		count, _ = mold.Where(table).Count()
		item, _ := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		list := cast.ToSlice(utils.ArrayMapWithField(item, params["field"]))

		// 一次取出本页所有父评论的子孙回复（旧的 replies() 是「每条父评论 × 每层一次 SQL」）
		roots := make([]int, 0, len(list))
		for _, val := range list {
			if id := cast.ToInt(cast.ToStringMap(val)["id"]); id > 0 {
				roots = append(roots, id)
			}
		}
		tree := this.commentReplyTree(roots, params["field"])
		for index := range list {
			node := cast.ToStringMap(list[index])
			if replies, exist := tree[cast.ToInt(node["id"])]; exist {
				node["replies"] = replies
			}
		}

		data = list
		this.setCache(ctx, cacheName, map[string]any{"data": data, "count": count})
	}

	data = this.maskCommentData(ctx, data)

	// 可见范围：评论树（含回复）逐层脱敏
	data = applyScopeData(ctx, data, "comment")

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

func (this *Comment) config(key ...any) (json map[string]any) {
	var config map[string]any
	configKey := "ARTICLE"

	isCommentConfig := false
	if len(key) > 0 && cast.ToString(key[0]) == "comment" {
		configKey = "COMMENT"
		isCommentConfig = true
	} else if len(key) > 0 && cast.ToString(key[0]) == "moments" {
		configKey = "MOMENTS"
	} else if len(key) > 0 && cast.ToString(key[0]) == "page" {
		// 独立页面此前会落进默认的 ARTICLE，导致继承文章配置
		configKey = "PAGE"
	} else if len(key) > 0 && cast.ToString(key[0]) == "article" {
		configKey = "ARTICLE"
	} else if len(key) > 0 && cast.ToString(key[0]) == "docs" {
		// 文档站模块总开关，见 app/model/config.go 的 DOCS 配置项
		configKey = "DOCS"
	}

	cacheName := "config[" + configKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if isCommentConfig {
		config, _ = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
		if cacheState {
			go facade.Cache.Set(cacheName, config)
		}
	} else {
		if cacheState && facade.Cache.Has(cacheName) {
			config = cast.ToStringMap(facade.Cache.Get(cacheName))
		} else {
			config, _ = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
			if cacheState {
				go facade.Cache.Set(cacheName, config)
			}
		}
	}

	if isCommentConfig {
		return cast.ToStringMap(config["json"])
	}

	if len(key) > 1 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[1])])
	}

	if len(key) > 0 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[0])])
	}

	return cast.ToStringMap(config["json"])
}
