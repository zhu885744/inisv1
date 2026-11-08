package controller

import (
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
)

/**
 * ChatModeration - 聊天审核（私信 / 公共聊天室）
 *
 * 数据来自 model.ChatModeration：消息命中词库时留下的「命中片段」，
 * 只包含命中词前后各 40 字，不含整条消息，后台据此判断是否违规。
 *
 * 只做「查看 + 标记处理」：禁言、封禁、撤回各自有专门的接口（room / users），
 * 这里不重复实现，避免同一动作有三套口径。
 */
type ChatModeration struct {
	base
}

func (this *ChatModeration) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"])
}

// IGET - 列表 / 数量
func (this *ChatModeration) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"all":   this.all,
		"count": this.count,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IPOST - 不支持手动新增：记录由消息发送时自动留证
func (this *ChatModeration) IPOST(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "聊天审核记录由发送消息自动生成，不支持手动新增！"), 405)
}

// IPUT - 标记已处理 / 恢复待处理
func (this *ChatModeration) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"handle": this.handle,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IDEL - 删除审核记录
func (this *ChatModeration) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove": this.remove,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *ChatModeration) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// scopedQuery - 列表与统计共用的筛选条件
//
// 常用筛选项走显式条件（前端不用拼 where，也避免字段名写错被当成未知列）；
// 其余仍支持通用的 where / like。
func (this *ChatModeration) scopedQuery(params map[string]any, dest any) *facade.ModelStruct {

	// dest 必须是切片指针（&[]model.ChatModeration{}）：
	// facade.ModelStruct.Select() 会把结果 Find 到 dest 里，传单个结构体时只会得到一行、
	// JSON 也是对象而不是数组，再经 ArrayMapWithField 就被 cast.ToSlice 吞成 nil
	// —— 表现为「count 有值、data 为 null」。
	query := this.buildQuery(facade.DB.Model(dest), params)

	if scene := strings.TrimSpace(cast.ToString(params["scene"])); scene != "" {
		query = query.Where("scene", scene)
	}
	if action := strings.TrimSpace(cast.ToString(params["action"])); action != "" {
		query = query.Where("action", action)
	}
	// handled 允许 0：用「参数存在」判断，不能用 Is.Empty
	if raw, ok := params["handled"]; ok && raw != nil && !utils.Is.Empty(cast.ToString(raw)) {
		query = query.Where("handled", cast.ToInt(raw))
	}
	if uid := cast.ToInt(params["uid"]); uid > 0 {
		query = query.Where("uid", uid)
	}
	if word := strings.TrimSpace(cast.ToString(params["word"])); word != "" {
		query = query.Where("word", "like", "%"+word+"%")
	}

	return query
}

// all - 分页列表
func (this *ChatModeration) all(ctx *gin.Context) {

	params := this.params(ctx, map[string]any{"page": 1})

	page := cast.ToInt(params["page"])
	if page <= 0 {
		page = 1
	}
	limit := this.meta.limit(ctx)
	if limit > 100 {
		limit = 100
	}

	// 列表与统计各自建一次查询：Count 与 Select 复用同一个 ModelStruct 时，
	// gorm 的 Statement 会被上一次执行改写过，容易出现「统计对了、列表空了」
	var rows []model.ChatModeration
	count, _ := this.scopedQuery(params, &rows).Count()
	items, _ := this.scopedQuery(params, &rows).Order("id desc").Limit(limit).Page(page).Select()

	list := utils.ArrayMapWithField(items, params["field"])

	this.attachUserInfo(list)

	this.json(ctx, gin.H{
		"data":  list,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// count - 数量（配合前端角标 / 待处理提醒）
func (this *ChatModeration) count(ctx *gin.Context) {
	params := this.params(ctx)
	count, _ := this.scopedQuery(params, &[]model.ChatModeration{}).Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

// handle - 标记已处理（handled=1）/ 恢复待处理（handled=0）
func (this *ChatModeration) handle(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	handled := 1
	if _, ok := params["handled"]; ok {
		handled = cast.ToInt(params["handled"])
	}

	payload := map[string]any{
		"handled":      handled,
		"handled_uid":  this.user(ctx).Id,
		"handled_time": time.Now().Unix(),
	}
	if handled == 0 {
		// 撤销处理：处理痕迹一起清掉，避免列表里显示「已处理」却又是待处理状态
		payload["handled_uid"] = 0
		payload["handled_time"] = 0
	}
	if remark, ok := params["remark"]; ok {
		payload["remark"] = cast.ToString(remark)
	}

	if _, err := facade.DB.Model(&model.ChatModeration{}).WhereIn("id", ids).Update(payload); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "处理失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids, "handled": handled}, facade.Lang(ctx, "处理成功！"), 200)
}

// remove - 删除记录（证据过期或确认无误）
func (this *ChatModeration) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.ChatModeration{}).WhereIn("id", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// attachUserInfo - 补发送者 / 接收者的昵称头像（列表里只显示 #uid 没法判断是谁）
func (this *ChatModeration) attachUserInfo(list []any) {

	if len(list) == 0 {
		return
	}

	rows := make([]map[string]any, 0, len(list))
	ids := make([]int, 0, len(list)*2)

	for _, raw := range list {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, item)
		for _, key := range []string{"uid", "to_uid"} {
			if uid := cast.ToInt(item[key]); uid > 0 {
				ids = append(ids, uid)
			}
		}
	}
	if len(ids) == 0 {
		return
	}

	columns, _ := facade.DB.Model(&model.Users{}).WhereIn("id", ids).Column("id", "account", "nickname", "avatar")

	users := make(map[int]map[string]any, len(ids))
	for _, raw := range cast.ToSlice(columns) {
		user := cast.ToStringMap(raw)
		users[cast.ToInt(user["id"])] = user
	}

	for _, item := range rows {
		if user, ok := users[cast.ToInt(item["uid"])]; ok {
			item["nickname"] = user["nickname"]
			item["avatar"] = user["avatar"]
			item["account"] = user["account"]
		}
		if toUid := cast.ToInt(item["to_uid"]); toUid > 0 {
			if user, ok := users[toUid]; ok {
				item["to_nickname"] = user["nickname"]
			}
		}
	}
}
