package controller

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * Feed - 聚合流（关注 / 推荐 / 热门）· 只读接口
 *
 * 三条流的口径：
 *   follow     我关注的人（user_follows.status = 1）发布的动态，按发布时间倒序
 *   recommend  近 N 天公开动态里「我没关注、也不是我自己」的内容，按热度倒序
 *   hot        近 N 天全站公开动态，按热度倒序
 *
 * 热度 = 点赞×1 + 评论×2 + 收藏×3（权重见常量）。
 * 只对候选池（≤ feedHotWindow 条）做三次 IN 查询后在内存里算，不做「每条动态一次聚合」的 N+1。
 *
 * 可见性（隐私）：
 *   **固定 audit = 1 且 status = 1** —— 只出对外公开的动态，
 *   不提供「看自己草稿 / 待审核」的入口（那是「我的动态」的职责，入口见 moments.go 的 isSelfQuery）。
 *   作者信息沿用 model.Moments.AfterFind 的字段白名单 + SanitizeAuthor 清洗，
 *   不会带出权限组（result.auth）、封禁记录等管理侧字段。
 *
 * 防滥用：
 *   1. 必须登录（权限种子见 model/auth-rules.go：path=all&type=login）
 *   2. limit / page 双上限钳制 —— 既挡「一次拖库」，也挡「深翻页批量抓取」
 *   3. Redis 缓存 60s；hot 与「我是谁」无关 → 全站共享 key，
 *      follow / recommend 的 key 带 uid，绝不会串号串流
 *   4. 通用 QPS 限流由 middleware/qps.go 统一负责，这里不重复实现
 */
type Feed struct {
	base
}

const (
	// feedMaxLimit - 单页最多返回条数（与前端分页大小上限一致）
	feedMaxLimit = 20
	// feedMaxPage - 允许翻到的最深页（挡深翻页抓取：最多 feedMaxPage × feedMaxLimit 条）
	feedMaxPage = 30
	// feedDefaultLimit - 默认单页条数
	feedDefaultLimit = 10
	// feedCacheTTL - 缓存秒数（注意传 int64：facade.Cache 只认 int64 的过期参数）
	feedCacheTTL = int64(60)
	// feedHotWindow - 热门 / 推荐的候选池大小（按发布时间倒序截取，保证新内容一定在内）
	feedHotWindow = 300
	// feedHotDays - 热门 / 推荐只统计最近多少天
	feedHotDays = 30
	// 热度权重
	feedScoreLike    = 1
	feedScoreComment = 2
	feedScoreCollect = 3
)

// feedAllowTypes - 允许的流类型
var feedAllowTypes = []any{"follow", "recommend", "hot"}

func (this *Feed) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"all": this.all,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// 聚合流是只读数据，不提供任何写入入口
func (this *Feed) IPOST(ctx *gin.Context) { this.readonly(ctx) }
func (this *Feed) IPUT(ctx *gin.Context)  { this.readonly(ctx) }
func (this *Feed) IDEL(ctx *gin.Context)  { this.readonly(ctx) }

func (this *Feed) readonly(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "聚合流为只读接口，不支持该操作！"), 405)
}

func (this *Feed) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// all - 聚合流：type=follow｜recommend｜hot
func (this *Feed) all(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{
		"type":  "follow",
		"page":  1,
		"limit": feedDefaultLimit,
	})

	feedType := strings.ToLower(strings.TrimSpace(cast.ToString(params["type"])))
	if !utils.In.Array(feedType, feedAllowTypes) {
		this.json(ctx, nil, facade.Lang(ctx, "type 只支持 follow / recommend / hot！"), 400)
		return
	}

	// 分页钳制：上限而不是报错，避免前端传大值时直接 400 影响体验
	page := cast.ToInt(params["page"])
	if page <= 0 {
		page = 1
	}
	if page > feedMaxPage {
		page = feedMaxPage
	}

	limit := cast.ToInt(params["limit"])
	if limit <= 0 {
		limit = feedDefaultLimit
	}
	if limit > feedMaxLimit {
		limit = feedMaxLimit
	}

	// follow / recommend 与「我关注了谁」有关 → key 必须带 uid，否则会串流
	cacheName := fmt.Sprintf("feed:%d:%s:%d:%d", uid, feedType, page, limit)
	if feedType == "hot" {
		// hot 的结果与「我关注了谁」无关，但**与「我拉黑了谁」有关**
		// （hotFeed 里会双向过滤拉黑对象）。所以不能直接用全站共享 key：
		// A 拉黑 B 之后，过滤过的结果会被缓存给所有人 —— 要么大家都看不到 B，
		// 要么反过来 A 仍能看到 B（取决于谁的请求先填充缓存）。
		// 这里把「拉黑集合」做成指纹拼进 key：没拉黑过人（绝大多数）的用户仍共享同一份缓存，
		// 有拉黑的按集合区分，正确性与命中率兼顾。
		cacheName = fmt.Sprintf("feed:hot:%d:%d:%s", page, limit, this.blockedFingerprint(uid))
	}

	if facade.Cache.Has(cacheName) {
		if cached, ok := facade.Cache.Get(cacheName).(map[string]any); ok && len(cached) > 0 {
			this.json(ctx, cached, facade.Lang(ctx, "数据请求成功！"), 200)
			return
		}
	}

	var data []map[string]any
	var total int

	switch feedType {
	case "follow":
		data, total = this.followFeed(uid, page, limit)
	case "recommend":
		data, total = this.hotFeed(uid, page, limit, true)
	default:
		data, total = this.hotFeed(uid, page, limit, false)
	}

	result := gin.H{
		"data":  data,
		"count": total,
		"page":  math.Ceil(float64(total) / float64(limit)),
		"type":  feedType,
	}

	go facade.Cache.Set(cacheName, result, feedCacheTTL)

	this.json(ctx, result, facade.Lang(ctx, "数据请求成功！"), 200)
}

// baseQuery - 公开可见的动态查询（已发布 + 已审核通过）
//
// 软删除条件由 soft_delete 插件自动附加（delete_time = 0），无需手写。
func (this *Feed) baseQuery() *facade.ModelStruct {
	return facade.DB.Model(&[]model.Moments{}).
		Where("status", 1).
		Where("audit", 1)
}

// followFeed - 关注流：我关注的人发布的动态，按发布时间倒序
func (this *Feed) followFeed(uid, page, limit int) ([]map[string]any, int) {

	following := this.followingUids(uid)
	if len(following) == 0 {
		return []map[string]any{}, 0
	}

	query := this.baseQuery().WhereIn("uid", "IN", following)

	if blocked := this.blockedUids(uid); len(blocked) > 0 {
		query = query.Where("uid", "NOT IN", blocked)
	}

	count, _ := query.Count()
	total := cast.ToInt(count)
	if total <= 0 {
		return []map[string]any{}, 0
	}

	items, _ := query.Limit(limit).Page(page).Order("create_time desc").Select()
	if items == nil {
		items = []map[string]any{}
	}

	return items, total
}

// hotFeed - 热门 / 推荐：近 N 天公开动态做候选池 → 热度排序 → 内存分页
//
// recommend = true 时排除「我自己 + 我已关注的人」（推荐的意义就是没看过的内容）。
func (this *Feed) hotFeed(uid, page, limit int, recommend bool) ([]map[string]any, int) {

	since := time.Now().AddDate(0, 0, -feedHotDays).Unix()

	query := this.baseQuery().Where("create_time", ">=", since)

	if recommend {
		exclude := append([]any{uid}, this.followingUids(uid)...)
		query = query.Where("uid", "NOT IN", exclude)
	}
	if blocked := this.blockedUids(uid); len(blocked) > 0 {
		query = query.Where("uid", "NOT IN", blocked)
	}

	// 候选池按时间倒序截取：宁可漏掉很久以前的高分老帖，也不漏新内容
	candidates, _ := query.Limit(feedHotWindow).Order("create_time desc").Select()
	if len(candidates) == 0 {
		return []map[string]any{}, 0
	}

	score := this.hotScore(candidates)

	sort.SliceStable(candidates, func(i, j int) bool {
		left := score[cast.ToInt(candidates[i]["id"])]
		right := score[cast.ToInt(candidates[j]["id"])]
		if left != right {
			return left > right
		}
		// 同分按时间倒序，保证顺序稳定（SliceStable + 时间兜底，翻页不会出现重复/漏项）
		return cast.ToInt64(candidates[i]["create_time"]) > cast.ToInt64(candidates[j]["create_time"])
	})

	total := len(candidates)
	start := (page - 1) * limit
	if start >= total {
		return []map[string]any{}, total
	}

	end := start + limit
	if end > total {
		end = total
	}

	return candidates[start:end], total
}

// hotScore - 热度分：点赞×1 + 评论×2 + 收藏×3
//
// 只对候选池的 id 做三次 IN 查询 + 内存计数，避免逐条聚合造成 N+1。
// 评论只算「可见」的（audit=1），与前台实际能看到的一致。
func (this *Feed) hotScore(items []map[string]any) map[int]int {

	score := make(map[int]int, len(items))
	if len(items) == 0 {
		return score
	}

	ids := make([]any, 0, len(items))
	for _, item := range items {
		ids = append(ids, cast.ToInt(item["id"]))
	}

	// 动态的 target_type 历史上有 moment / moments 两种写法，都算上。
	// 表结构见 model/user-likes.go、model/user-collects.go（uniqueIndex: uk_uid_target）
	if column, err := facade.DB.Model(&[]model.UserLikes{}).
		Where("target_type", "IN", []any{"moment", "moments"}).
		Where("target_id", "IN", ids).
		Column("target_id"); err == nil {
		for _, id := range cast.ToIntSlice(column) {
			score[id] += feedScoreLike
		}
	}

	if column, err := facade.DB.Model(&[]model.UserCollects{}).
		Where("target_type", "IN", []any{"moment", "moments"}).
		Where("target_id", "IN", ids).
		Column("target_id"); err == nil {
		for _, id := range cast.ToIntSlice(column) {
			score[id] += feedScoreCollect
		}
	}

	if column, err := facade.DB.Model(&[]model.Comment{}).
		Where("bind_type", "moments").
		Where("bind_id", "IN", ids).
		Where("audit", 1).
		Column("bind_id"); err == nil {
		for _, id := range cast.ToIntSlice(column) {
			score[id] += feedScoreComment
		}
	}

	return score
}

// followingUids - 我关注的人（status = 1；0 表示已取消关注）
//
// 顺手排掉自己：重复关注 / 互关数据异常时，流里不该出现自己的动态。
func (this *Feed) followingUids(uid int) []any {

	column, err := facade.DB.Model(&[]model.UserFollows{}).
		Where("uid", uid).
		Where("status", 1).
		Column("follow_uid")
	if err != nil {
		return []any{}
	}

	result := make([]any, 0)
	for _, id := range cast.ToIntSlice(column) {
		if id <= 0 || id == uid {
			continue
		}
		result = append(result, id)
	}

	return result
}

// blockedFingerprint - 拉黑集合的缓存指纹（进热门流缓存 key，见 all() 里的说明）
//
// 没有拉黑任何人时返回 "0"，所有普通用户共享同一份热门缓存；
// 有拉黑时按集合内容取模哈希（同集合的人仍能共享），集合变化会自然换 key，
// 旧缓存 60 秒后自动过期，不需要额外清理。
func (this *Feed) blockedFingerprint(uid int) string {

	blocked := this.blockedUids(uid)
	if len(blocked) == 0 {
		return "0"
	}

	ids := make([]int, 0, len(blocked))
	for _, id := range blocked {
		ids = append(ids, cast.ToInt(id))
	}
	sort.Ints(ids)

	sum := fnv.New32a()
	for _, id := range ids {
		_, _ = sum.Write([]byte(strconv.Itoa(id) + ","))
	}

	return strconv.FormatUint(uint64(sum.Sum32()), 36)
}

// blockedUids - 我拉黑的人 + 拉黑我的人（双向过滤）
//
// 语义统一走 model.UserBlocks.BlockedUids（双向）：拉黑我的人的内容同样不给我看，
// 避免出现「我拉黑了 TA，TA 却还能出现在我的流里」这种一半生效的状态。
func (this *Feed) blockedUids(uid int) []any {

	if uid <= 0 {
		return []any{}
	}

	ids := (&model.UserBlocks{}).BlockedUids(uid)
	if len(ids) == 0 {
		return []any{}
	}

	result := make([]any, 0, len(ids))
	for _, id := range ids {
		result = append(result, id)
	}

	return result
}
