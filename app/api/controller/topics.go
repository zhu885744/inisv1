package controller

import (
	"fmt"
	"strings"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// Topics - 话题（Hashtag）控制器
//
// 与后台「标签管理」（/api/tags/*）是同一张表：后台管 CRUD 与主持人 / 精选，
// 这里只放面向用户的话题能力 —— 话题榜、话题内内容聚合、关注 / 取关、联想候选。
//
// @Summary 话题API
// @Description 话题（Hashtag）：#话题# 聚合页、关注、热度榜
// @Tags Topics
type Topics struct {
	// 继承
	base
}

// 话题榜缓存时长（秒）
//
// ⚠ 必须带过期时间：关注 / 发布内容时会主动清，但漏清一处也不能变成「永久缓存」
// （配置缓存不带过期时间，正是「后台关掉内容审核不生效」那类问题的根源）。
const topicsBoardCacheExpire = 60

// 话题联想候选的上限（编辑器 # 面板一次取回后本地过滤）
const topicCandidateLimitForPicker = 500

// IGET - 获取话题数据
// @Summary 获取话题数据
// @Description 话题榜 / 话题详情 / 话题内内容 / 联想候选 / 我关注的话题
// @Tags Topics
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(board, detail, content, candidates, mine)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/topics/{method} [get]
func (this *Topics) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"board":      this.board,
		"detail":     this.detail,
		"content":    this.content,
		"candidates": this.candidates,
		"mine":       this.mine,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - 话题写操作
// @Summary 话题写操作
// @Description 关注 / 取关 / 主持人编辑话题资料
// @Tags Topics
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(follow, unfollow, moderator-update)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/topics/{method} [post]
func (this *Topics) IPOST(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"follow":           this.follow,
		"unfollow":         this.unfollow,
		"moderator-update": this.moderatorUpdate,
		// 主持人能力：话题内置顶 / 移出话题 / 删除内容
		"pin":            this.pin,
		"unpin":          this.unpin,
		"remove-content": this.removeContent,
		"delete-content": this.deleteContent,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - 更新话题
// @Summary 更新话题
// @Description 取关（与 user-follows 一致，PUT / POST 都支持）
// @Tags Topics
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(unfollow)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/topics/{method} [put]
func (this *Topics) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"unfollow": this.unfollow,
		"unpin":    this.unpin,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - 话题删除
// @Summary 话题删除
// @Description 话题删除走后台接口（/api/tags/remove），这里不开放
// @Tags Topics
// @Accept json
// @Produce json
// @Success 403 {object} map[string]interface{} "响应信息"
// @Router /api/topics/{method} [delete]
func (this *Topics) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "话题删除请到后台「话题管理」操作！"), 403)
}

// INDEX - 话题索引接口
// @Summary 话题索引接口
// @Description 话题模块的索引接口（无实际功能）
// @Tags Topics
// @Accept json
// @Produce json
// @Success 202 {object} map[string]interface{} "响应信息"
// @Router /api/topics/index [get]
func (this *Topics) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// ---------- 读接口 ----------

// board - 话题榜（按热度：内容量 + 关注数加权）
//
// 参数：keyword（按名称模糊，做联想）、recommend=1（只看精选）、limit
func (this *Topics) board(ctx *gin.Context) {

	params := this.params(ctx)

	keyword := strings.TrimSpace(cast.ToString(params["keyword"]))
	onlyRecommend := cast.ToBool(params["recommend"])
	limit := cast.ToInt(params["limit"])
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	cacheName := fmt.Sprintf("[GET]topics[board][kw=%v][recommend=%v][limit=%v]", keyword, onlyRecommend, limit)

	var base []map[string]any
	if facade.Cache.Has(cacheName) {
		base = toMapList(facade.Cache.Get(cacheName))
	} else {
		base = (&model.Topic{}).Board(keyword, onlyRecommend, limit)
		// 带过期时间（见 topicsBoardCacheExpire 的注释）
		facade.Cache.Set(cacheName, base, topicsBoardCacheExpire)
	}

	this.json(ctx, this.withFollowing(ctx, base), facade.Lang(ctx, "数据请求成功！"), 200)
}

// candidates - 话题联想候选（编辑器里输入 # 时用）
//
// 与 @提及 的候选一样要求登录：未登录不提示话题，避免被当成公开的用户/话题爬取接口。
func (this *Topics) candidates(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	keyword := strings.TrimSpace(cast.ToString(params["keyword"]))

	// 编辑器里的 # 联想面板一次取回后在本地过滤，所以要给足候选（默认 50，最多 500）
	limit := cast.ToInt(params["limit"])
	if limit <= 0 || limit > topicCandidateLimitForPicker {
		limit = 50
	}

	base := (&model.Topic{}).Board(keyword, false, limit)

	this.json(ctx, this.withFollowing(ctx, base), facade.Lang(ctx, "数据请求成功！"), 200)
}

// detail - 话题详情（支持 id 或名称访问，与前台 /topic/:key 一致）
func (this *Topics) detail(ctx *gin.Context) {

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return // findTag 里已响应（参数缺失 400 / 话题不存在 204）
	}

	tagId := cast.ToInt(item["id"])

	articleCount := this.countContent(&[]model.Article{}, tagId)
	momentCount := this.countContent(&[]model.Moments{}, tagId)
	followCount := cast.ToInt((&model.TagFollows{}).GetFollowersCount(tagId))

	data := map[string]any{
		"id":            item["id"],
		"name":          item["name"],
		"description":   item["description"],
		"avatar":        item["avatar"],
		"moderator_uid": item["moderator_uid"],
		"recommend":     item["recommend"],
		"create_time":   item["create_time"],
		"article_count": articleCount,
		"moment_count":  momentCount,
		"follow_count":  followCount,
		"heat":          (&model.Topic{}).Heat(articleCount, momentCount, followCount),
		"moderator":     this.moderator(cast.ToInt(item["moderator_uid"])),
		"is_following":  false,
	}

	if uid := this.meta.user(ctx).Id; uid > 0 {
		data["is_following"] = (&model.TagFollows{}).IsFollowing(uid, tagId)
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// content - 话题内的内容聚合（动态 / 文章分页）
//
// 参数：id 或 key（话题 id / 名称）、type（moments 默认 / article）、sort（new 默认 / hot / top）、page、limit
func (this *Topics) content(ctx *gin.Context) {

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return // findTag 里已响应（参数缺失 400 / 话题不存在 204）
	}

	tagId := cast.ToInt(item["id"])
	contentType := strings.ToLower(cast.ToString(params["type"]))
	sortType := strings.ToLower(cast.ToString(params["sort"]))
	page := cast.ToInt(params["page"])
	if page <= 0 {
		page = 1
	}
	limit := this.meta.limit(ctx)
	like := "%|" + cast.ToString(tagId) + "|%"

	var data []map[string]any
	var count int64
	pinned := []int{}

	if contentType == "article" {
		// 文章：只算审核通过的（与文章列表口径一致）
		order := "create_time desc"
		if sortType == "hot" {
			order = "views desc, create_time desc"
		}
		pinned = (&model.TopicPins{}).PinnedIds(tagId, model.TopicTargetArticle)
		query := facade.DB.Model(&[]model.Article{}).Where("audit", 1).Like("tags", like)
		count, _ = query.Count()
		// 置顶优先的表达式是内部拼装的（id 来自数据库的 int），走 OrderRaw 跳过白名单
		data, _ = query.Limit(limit).Page(page).OrderRaw(withPinnedOrder(order, pinned)).Select()
	} else {
		// 动态：只算已发布 + 审核通过的
		order := "create_time desc"
		switch sortType {
		case "hot":
			order = "views desc, create_time desc"
		case "top":
			order = "top desc, create_time desc"
		}
		pinned = (&model.TopicPins{}).PinnedIds(tagId, model.TopicTargetMoments)
		query := facade.DB.Model(&[]model.Moments{}).Where("audit", 1).Where("status", 1).Like("tags", like)
		count, _ = query.Count()
		data, _ = query.Limit(limit).Page(page).OrderRaw(withPinnedOrder(order, pinned)).Select()
	}

	// 置顶标记：前端据此显示「置顶」徽标与主持人的取消置顶按钮
	data = markPinned(data, pinned)

	// 可见范围：话题聚合页同样按条脱敏（登录可见 / 付费可见的内容不回传正文）
	bindType := "moments"
	if contentType == "article" {
		bindType = "article"
	}
	if redacted, ok := applyScopeData(ctx, data, bindType).([]map[string]any); ok {
		data = redacted
	}

	if utils.Is.Empty(data) {
		this.json(ctx, gin.H{"data": []any{}, "count": 0, "page": 0}, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  page,
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// mine - 我关注的话题
func (this *Topics) mine(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	ids := (&model.TagFollows{}).FollowingTagIds(uid)
	if len(ids) == 0 {
		this.json(ctx, []any{}, facade.Lang(ctx, "无数据！"), 200)
		return
	}

	rows, _ := facade.DB.Model(&[]model.Tags{}).WhereIn("id", ids).Select()

	// 按「最近关注」的顺序返回（FollowingTagIds 已是该顺序）
	byId := make(map[int]map[string]any, len(rows))
	for _, row := range rows {
		byId[cast.ToInt(row["id"])] = row
	}

	articles, moments := (&model.Topic{}).Counts()
	follows := (&model.TagFollows{}).CountByTagIds()

	data := []map[string]any{}
	for _, id := range ids {
		row, ok := byId[id]
		if !ok {
			continue // 话题已被删除
		}
		key := cast.ToString(id)
		item := map[string]any{
			"id":            row["id"],
			"name":          row["name"],
			"description":   row["description"],
			"avatar":        row["avatar"],
			"moderator_uid": row["moderator_uid"],
			"recommend":     row["recommend"],
			"article_count": articles[key],
			"moment_count":  moments[key],
			"follow_count":  follows[key],
			"is_following":  true,
		}
		item["heat"] = (&model.Topic{}).Heat(articles[key], moments[key], follows[key])
		data = append(data, item)
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// ---------- 写接口 ----------

// follow - 关注话题
func (this *Topics) follow(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return // findTag 里已响应（参数缺失 400 / 话题不存在 204）
	}

	tagId := cast.ToInt(item["id"])

	if err := (&model.TagFollows{}).Follow(uid, tagId); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{
		"id":           tagId,
		"is_following": true,
		"follow_count": (&model.TagFollows{}).GetFollowersCount(tagId),
	}, facade.Lang(ctx, "关注成功！"), 200)
}

// unfollow - 取消关注话题
func (this *Topics) unfollow(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return // findTag 里已响应（参数缺失 400 / 话题不存在 204）
	}

	tagId := cast.ToInt(item["id"])

	if err := (&model.TagFollows{}).Unfollow(uid, tagId); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{
		"id":           tagId,
		"is_following": false,
		"follow_count": (&model.TagFollows{}).GetFollowersCount(tagId),
	}, facade.Lang(ctx, "已取消关注！"), 200)
}

// moderatorUpdate - 话题主持人编辑自己的话题资料（简介 / 封面）
//
// 权限：话题的主持人本人，或系统管理员。
// 只能改 description / avatar —— 话题名不允许改（正文里的 #[名称](/topic/1) 会跟着变味），
// 精选、主持人改派等仍走后台「标签管理」。
func (this *Topics) moderatorUpdate(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return // findTag 里已响应（参数缺失 400 / 话题不存在 204）
	}

	tagId := cast.ToInt(item["id"])

	if !this.meta.root(ctx) && cast.ToInt(item["moderator_uid"]) != uid {
		this.json(ctx, nil, facade.Lang(ctx, "只有话题主持人可以编辑话题资料！"), 403)
		return
	}

	update := map[string]any{}
	allow := []any{"description", "avatar"}

	for key, val := range params {
		if !utils.In.Array(key, allow) {
			continue
		}
		text := cast.ToString(val)
		if facade.Comm.DetectXSS(text) {
			this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
			return
		}
		if key == "avatar" {
			// 链接类字段不转义 &，否则带参数的图片地址会失效
			update[key] = facade.Comm.SanitizeURL(text)
			continue
		}
		update[key] = facade.Comm.SanitizeHTML(text)
	}

	if len(update) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "没有可更新的内容！"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.Tags{}).Where("id", tagId).Update(update); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{"id": tagId}, facade.Lang(ctx, "更新成功！"), 200)
}

// ---------- 主持人能力（置顶 / 移出话题 / 删除内容） ----------

// pin - 在话题内置顶一条内容
func (this *Topics) pin(ctx *gin.Context) {
	this.togglePin(ctx, true)
}

// unpin - 取消话题内置顶
func (this *Topics) unpin(ctx *gin.Context) {
	this.togglePin(ctx, false)
}

// togglePin - 置顶 / 取消置顶的共用逻辑
//
// 只允许话题主持人（或系统管理员）操作，且内容必须真的在这个话题里
// —— 否则主持人就能靠猜 id 去置顶任意内容。
func (this *Topics) togglePin(ctx *gin.Context, on bool) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return
	}

	tagId := cast.ToInt(item["id"])
	if !this.assertModerator(ctx, item, uid) {
		return
	}

	contentType := normalizeContentType(cast.ToString(params["type"]))
	targetId := cast.ToInt(params["target_id"])

	if targetId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	if !this.contentInTopic(tagId, contentType, targetId) {
		this.json(ctx, nil, facade.Lang(ctx, "该内容不在这个话题里！"), 400)
		return
	}

	var err error
	if on {
		err = (&model.TopicPins{}).Pin(tagId, contentType, targetId, uid)
	} else {
		err = (&model.TopicPins{}).Unpin(tagId, contentType, targetId)
	}

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{
		"target_id": targetId,
		"is_pinned": on,
	}, facade.Lang(ctx, "操作成功！"), 200)
}

// removeContent - 把一条内容移出话题（**不删除内容本身**，可再次编辑加回来）
func (this *Topics) removeContent(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return
	}

	tagId := cast.ToInt(item["id"])
	if !this.assertModerator(ctx, item, uid) {
		return
	}

	contentType := normalizeContentType(cast.ToString(params["type"]))
	targetId := cast.ToInt(params["target_id"])

	if targetId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	current := this.readTargetTags(contentType, targetId)
	ids := (&model.Topic{}).TagsFieldIds(current)

	rest := []int{}
	found := false
	for _, id := range ids {
		if id == tagId {
			found = true
			continue
		}
		rest = append(rest, id)
	}

	if !found {
		this.json(ctx, nil, facade.Lang(ctx, "该内容不在这个话题里！"), 400)
		return
	}

	// 用 map 更新：话题被摘完后要写成空串（map 更新不会跳过零值）
	if err := this.updateTargetTags(contentType, targetId, (&model.Topic{}).BuildTagsField(rest)); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 顺手清掉它在各话题里的置顶，避免「已移出但还排在置顶」
	(&model.TopicPins{}).ClearTarget(contentType, targetId)
	go this.delCache()

	this.json(ctx, gin.H{"target_id": targetId}, facade.Lang(ctx, "已移出话题！"), 200)
}

// deleteContent - 删除话题内的内容（软删除，可到后台回收站恢复）
//
// 与 removeContent 的区别：这是真的删内容，属于主持人权限里最重的操作，
// 只对「确实在本话题里的内容」生效，并记一条告警日志便于事后追溯。
func (this *Topics) deleteContent(ctx *gin.Context) {

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	item := this.findTag(ctx, params)
	if utils.Is.Empty(item) {
		return
	}

	tagId := cast.ToInt(item["id"])
	if !this.assertModerator(ctx, item, uid) {
		return
	}

	contentType := normalizeContentType(cast.ToString(params["type"]))
	targetId := cast.ToInt(params["target_id"])

	if targetId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	if !this.contentInTopic(tagId, contentType, targetId) {
		this.json(ctx, nil, facade.Lang(ctx, "该内容不在这个话题里！"), 400)
		return
	}

	var err error
	if contentType == model.TopicTargetArticle {
		_, err = facade.DB.Model(&model.Article{}).Delete(targetId)
	} else {
		_, err = facade.DB.Model(&model.Moments{}).Delete(targetId)
	}

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	(&model.TopicPins{}).ClearTarget(contentType, targetId)
	go this.delCache()

	facade.Log.Warn(map[string]any{
		"uid": uid, "topic_id": tagId, "type": contentType, "target_id": targetId,
	}, "话题主持人删除话题内内容")

	this.json(ctx, gin.H{"target_id": targetId}, facade.Lang(ctx, "已删除！"), 200)
}

// ---------- 内部工具 ----------

// withPinnedOrder - 把话题内置顶并进 SQL 排序（置顶的排前面，其余照常）
//
// 用 ORDER BY (id IN (...)) DESC 而不是「先查置顶再查其余」：后者会打乱分页偏移，
// 翻页时出现重复 / 漏项。id 全部来自数据库（int），不含用户输入。
func withPinnedOrder(order string, pinned []int) string {

	ids := []string{}
	for _, id := range pinned {
		if id > 0 {
			ids = append(ids, cast.ToString(id))
		}
	}

	if len(ids) == 0 {
		return order
	}

	return fmt.Sprintf("(id IN (%v)) DESC, %v", strings.Join(ids, ","), order)
}

// markPinned - 给列表补 is_pinned 标记
func markPinned(list []map[string]any, pinned []int) []map[string]any {

	if len(list) == 0 {
		return list
	}

	set := make(map[int]bool, len(pinned))
	for _, id := range pinned {
		set[id] = true
	}

	for _, row := range list {
		row["is_pinned"] = set[cast.ToInt(row["id"])]
	}

	return list
}

// assertModerator - 校验当前用户是不是该话题的主持人（管理员放行）
func (this *Topics) assertModerator(ctx *gin.Context, item map[string]any, uid int) bool {

	if this.meta.root(ctx) {
		return true
	}

	if uid > 0 && cast.ToInt(item["moderator_uid"]) == uid {
		return true
	}

	this.json(ctx, nil, facade.Lang(ctx, "只有话题主持人可以操作！"), 403)

	return false
}

// contentInTopic - 内容是否真的关联着该话题（主持人操作的前置校验）
func (this *Topics) contentInTopic(tagId int, contentType string, targetId int) bool {

	for _, id := range (&model.Topic{}).TagsFieldIds(this.readTargetTags(contentType, targetId)) {
		if id == tagId {
			return true
		}
	}

	return false
}

// readTargetTags - 读内容的 tags 字段（|1|2|3|）
func (this *Topics) readTargetTags(contentType string, targetId int) string {

	var row map[string]any

	if contentType == model.TopicTargetArticle {
		row, _ = facade.DB.Model(&model.Article{}).Find(targetId)
	} else {
		row, _ = facade.DB.Model(&model.Moments{}).Find(targetId)
	}

	return cast.ToString(row["tags"])
}

// updateTargetTags - 写内容的 tags 字段
func (this *Topics) updateTargetTags(contentType string, targetId int, tags string) error {

	target := targetModel(contentType)
	_, err := facade.DB.Model(target).Where("id", targetId).Update(map[string]any{"tags": tags})

	return err
}

// targetModel - 内容类型 → 模型（用于 update / delete）
func targetModel(contentType string) any {
	if contentType == model.TopicTargetArticle {
		return &model.Article{}
	}
	return &model.Moments{}
}

// normalizeContentType - 内容类型归一化（默认动态）
func normalizeContentType(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == model.TopicTargetArticle {
		return model.TopicTargetArticle
	}
	return model.TopicTargetMoments
}

// delCache - 话题相关缓存（榜单 / 详情）全部失效
func (this *Topics) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "topics"})
	facade.Cache.DelTags([]any{"[GET]", "tags"})
}

// findTag - 按 id 或名称定位话题（两者都没有时直接响应 400）
func (this *Topics) findTag(ctx *gin.Context, params map[string]any) map[string]any {

	tagId := cast.ToInt(params["id"])
	key := strings.TrimSpace(cast.ToString(params["key"]))

	query := facade.DB.Model(&model.Tags{})

	if tagId > 0 {
		query = query.Where("id", tagId)
	} else if key != "" {
		// 与前台 /topic/:key 一致：允许用 id 或名称访问，名称重名时取 id 最小者
		if id := cast.ToInt(key); id > 0 {
			query = query.Where("id", id)
		} else {
			query = query.Where("name", key).Order("id asc")
		}
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return nil
	}

	item, _ := query.Find()

	// 不存在就在这里响应，调用方只负责 return，避免重复写响应
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "话题不存在！"), 204)
	}

	return item
}

// countContent - 统计某个话题关联的内容量（文章 / 动态，口径与后台标签管理一致）
func (this *Topics) countContent(table any, tagId int) int {

	query := facade.DB.Model(table).Where("audit", 1)
	if _, ok := table.(*[]model.Moments); ok {
		query = query.Where("status", 1)
	}

	count, _ := query.Like("tags", "%|"+cast.ToString(tagId)+"|%").Count()

	return cast.ToInt(count)
}

// moderator - 话题主持人信息（公开字段，与作者信息一样要清洗管理侧字段）
func (this *Topics) moderator(uid int) any {

	if uid <= 0 {
		return nil
	}

	user, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return nil
	}

	author := utils.Map.WithField(user, []string{"id", "nickname", "avatar", "description", "title"})
	model.SanitizeAuthor(author)

	return author
}

// toMapList - 把缓存里取回的数据还原成 []map[string]any
//
// 缓存后端不同返回类型也不同：内存缓存直接给原对象，Redis / 文件缓存是 JSON 往返后的 []any，
// 所以两种形态都要兼容，不能靠类型断言硬取。
func toMapList(value any) []map[string]any {

	switch data := value.(type) {
	case []map[string]any:
		return data
	case []any:
		result := make([]map[string]any, 0, len(data))
		for _, item := range data {
			if row, ok := item.(map[string]any); ok {
				result = append(result, row)
			}
		}
		return result
	case string:
		if data == "" {
			return []map[string]any{}
		}
		return toMapList(utils.Json.Decode(data))
	}

	return []map[string]any{}
}

// withFollowing - 给榜单 / 候选列表补上「当前用户是否已关注」
//
// 榜单是带缓存且可能被多个请求共享的，这里复制一份再写 is_following，避免污染缓存对象。
func (this *Topics) withFollowing(ctx *gin.Context, base []map[string]any) []map[string]any {

	uid := this.meta.user(ctx).Id

	ids := make([]int, 0, len(base))
	for _, item := range base {
		ids = append(ids, cast.ToInt(item["id"]))
	}

	following := map[int]bool{}
	if uid > 0 {
		following = (&model.TagFollows{}).FilterFollowingTagIds(uid, ids)
	}

	rows := make([]map[string]any, 0, len(base))
	for _, item := range base {
		row := make(map[string]any, len(item)+1)
		for key, val := range item {
			row[key] = val
		}
		row["is_following"] = following[cast.ToInt(item["id"])]
		rows = append(rows, row)
	}

	return rows
}
