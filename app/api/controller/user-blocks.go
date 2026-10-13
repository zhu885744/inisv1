package controller

import (
	"math"
	"strings"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * UserBlocks - 拉黑（黑名单）
 *
 * 语义是**双向**的（见 model/user-blocks.go）：
 *   我拉黑 TA 后，我的聚合流里不再出现 TA 的内容，TA 的流里也不再出现我的；
 *   双方不能互相发私信（私信上线后走 model.UserBlocks.CanInteract）。
 *
 * 接口：
 *   GET  user-blocks/mine    我的拉黑列表（含被拉黑用户的公开信息）
 *   GET  user-blocks/status  我是否已拉黑某人（界面按钮状态）
 *   POST user-blocks/block   拉黑（同时解除双方的关注关系）
 *   PUT  user-blocks/unblock 解除拉黑
 *
 * 隐私：拉黑列表只对本人可见；对外（Feed / 私信）只体现为「内容不出现 / 不能互动」，
 * 不向被拉黑方暴露「你被谁拉黑了」。
 */
type UserBlocks struct {
	base
}

const (
	// userBlocksMaxPage - 拉黑列表最深页码（防深翻页）
	userBlocksMaxPage = 50
	// userBlocksDefaultLimit - 默认每页条数
	userBlocksDefaultLimit = 20
)

func (this *UserBlocks) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"mine":   this.mine,
		"status": this.status,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserBlocks) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"block": this.block,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserBlocks) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"unblock": this.unblock,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserBlocks) IDEL(ctx *gin.Context) { this.readonly(ctx) }

func (this *UserBlocks) readonly(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "拉黑关系请用拉黑 / 解除拉黑接口，不支持该操作！"), 405)
}

func (this *UserBlocks) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// mine - 我的拉黑列表
func (this *UserBlocks) mine(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{"page": 1, "limit": userBlocksDefaultLimit})

	page := cast.ToInt(params["page"])
	if page <= 0 {
		page = 1
	}
	if page > userBlocksMaxPage {
		page = userBlocksMaxPage
	}

	limit := cast.ToInt(params["limit"])
	if limit <= 0 {
		limit = userBlocksDefaultLimit
	}
	if limit > 50 {
		limit = 50
	}

	data, count := (&model.UserBlocks{}).GetBlocked(uid, page, limit)

	list := make([]map[string]any, 0, len(data))
	for _, item := range data {
		blockUid := cast.ToInt(item["block_uid"])
		if blockUid <= 0 {
			continue
		}
		// reason 只在本人列表里出现，不随内容一起对外
		item["block_uid"] = blockUid
		item["user"] = this.userBrief(blockUid)
		list = append(list, item)
	}

	this.json(ctx, gin.H{
		"data":  list,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// status - 我是否已拉黑某人（界面按钮状态用）
func (this *UserBlocks) status(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	blockUid := cast.ToInt(params["block_uid"])
	if blockUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "block_uid"), 400)
		return
	}

	blocked := (&model.UserBlocks{}).IsBlocked(uid, blockUid)

	this.json(ctx, gin.H{"blocked": blocked}, facade.Lang(ctx, "查询成功！"), 200)
}

// block - 拉黑
//
// 同时解除双方的关注关系：拉黑后仍显示「已关注」会让人以为关系还在，
// 而且 Feed 里既然不再出现对方内容，关注关系就没有意义了。
func (this *UserBlocks) block(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	blockUid := cast.ToInt(params["block_uid"])
	if blockUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "block_uid"), 400)
		return
	}
	if blockUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能拉黑自己！"), 400)
		return
	}

	// 与封禁体系保持一致：系统管理员（uid=1）不接受被拉黑，避免站点运营被个人设置影响
	if blockUid == 1 {
		this.json(ctx, nil, facade.Lang(ctx, "不能拉黑系统管理员！"), 403)
		return
	}

	target, _ := facade.DB.Model(&model.Users{}).Find(blockUid)
	if utils.Is.Empty(target) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	reason := strings.TrimSpace(cast.ToString(params["reason"]))
	if len([]rune(reason)) > 120 {
		reason = string([]rune(reason)[:120])
	}

	follows := &model.UserFollows{}
	blocks := &model.UserBlocks{}

	err := blocks.Block(uid, blockUid, reason)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 解除双方的关注关系（单向或互关都一并清掉）
	_ = follows.Unfollow(uid, blockUid)
	_ = follows.Unfollow(blockUid, uid)

	// Feed / 大 V 主页等都缓存过数据，拉黑后要把双方的缓存清掉，
	// 否则「拉黑后仍能看到对方内容」会持续到缓存过期（Feed 缓存 60s）
	go this.delFeedCache(uid, blockUid)

	this.json(ctx, gin.H{"block_uid": blockUid}, facade.Lang(ctx, "已拉黑，对方的内容不会再出现在你的动态流里！"), 200)
}

// unblock - 解除拉黑
func (this *UserBlocks) unblock(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	blockUid := cast.ToInt(params["block_uid"])
	if blockUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "block_uid"), 400)
		return
	}

	if err := (&model.UserBlocks{}).Unblock(uid, blockUid); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delFeedCache(uid, blockUid)

	// 解除拉黑不自动恢复关注关系：那是对方的意愿，不该由解除拉黑代替
	this.json(ctx, gin.H{"block_uid": blockUid}, facade.Lang(ctx, "已解除拉黑！"), 200)
}

// delFeedCache - 清掉双方的聚合流缓存（拉黑 / 解除拉黑后立即生效）
func (this *UserBlocks) delFeedCache(uids ...int) {
	// 缓存名格式见 controller/feed.go：feed:{uid}:{type}:{page}:{limit}
	// 这里按前缀批量删，避免把 page/limit 组合写死在两处
	prefixes := make([]any, 0, len(uids)*3)
	for _, uid := range uids {
		if uid <= 0 {
			continue
		}
		prefixes = append(prefixes, "feed:"+cast.ToString(uid)+":")
	}
	facade.Cache.DelPrefix(prefixes...)
}

// userBrief - 拉黑列表里展示的用户公开信息（字段白名单 + 隐私清洗）
func (this *UserBlocks) userBrief(uid int) map[string]any {

	item, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(item) {
		return map[string]any{"id": uid}
	}

	user := utils.Map.WithField(item, []string{"id", "nickname", "avatar", "description", "title"})
	model.SanitizeAuthor(user)

	return user
}
