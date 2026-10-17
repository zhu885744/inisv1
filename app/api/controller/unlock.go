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

// Unlock - 内容可见范围（登录可见 / 付费可见）相关接口
//
//	GET  /api/unlock/check  查询某条内容的可见状态（是否需要登录 / 需要多少积分 / 是否已兑换）
//	GET  /api/unlock/list   我的兑换记录
//	POST /api/unlock/pay    使用积分兑换可见权限
type Unlock struct {
	base
}

func (this *Unlock) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"check": this.check,
		"list":  this.list,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Unlock) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"pay": this.pay,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Unlock) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Unlock) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Unlock) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// check - 查询内容可见状态
//
// 参数：bind_type（article / moments / page / comment）、bind_id
// 返回：scope（1 公开 / 2 登录可见 / 3 付费可见）、scope_integral（所需积分）、
// scope_state（"" 可查看 / login 需登录 / pay 需兑换）、can_view（当前是否可查看完整内容）
func (this *Unlock) check(ctx *gin.Context) {
	params := this.params(ctx)

	bindType := cast.ToString(params["bind_type"])
	bindId := cast.ToInt(params["bind_id"])

	if !model.ScopeBindTypeValid(bindType) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的内容类型！"), 400)
		return
	}
	if bindId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	owner, scope, integral, _, err := model.ScopeContentInfo(bindType, bindId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	state := newScopeGuard(ctx).access(owner, scope, bindId, bindType, nil)

	// 评论不支持付费可见：3 降级为登录可见，且不再下发兑换积分
	scope = model.ScopeForBind(bindType, scope)
	if !model.ScopePaySupported(bindType) {
		integral = 0
	}

	this.json(ctx, gin.H{
		"bind_type":      bindType,
		"bind_id":        bindId,
		"scope":          scope,
		"scope_integral": integral,
		"scope_state":    state,
		"can_view":       state == model.ScopeStateAllow,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// pay - 使用积分兑换内容可见权限
//
// 参数：bind_type、bind_id
func (this *Unlock) pay(ctx *gin.Context) {
	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	bindType := cast.ToString(params["bind_type"])
	bindId := cast.ToInt(params["bind_id"])

	if !model.ScopeBindTypeValid(bindType) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的内容类型！"), 400)
		return
	}
	if bindId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}
	// 评论只支持 公开 / 登录可见，不参与积分兑换
	if !model.ScopePaySupported(bindType) {
		this.json(ctx, nil, facade.Lang(ctx, "该内容不支持积分兑换！"), 202)
		return
	}

	result, err := (&model.ContentUnlock{}).Pay(user.Id, bindType, bindId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "兑换成功！"), 200)
}

// list - 我的兑换记录
//
// 参数：page、limit、bind_type（可选）、order（可选，默认 create_time desc）
func (this *Unlock) list(ctx *gin.Context) {
	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	bindType := cast.ToString(params["bind_type"])
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	query := facade.DB.Model(&[]model.ContentUnlock{}).Where("uid", user.Id)
	if !utils.Is.Empty(bindType) {
		if !model.ScopeBindTypeValid(bindType) {
			this.json(ctx, nil, facade.Lang(ctx, "不支持的内容类型！"), 400)
			return
		}
		query = query.Where("bind_type", bindType)
	}

	count, _ := query.Count()
	items, _ := query.Limit(limit).Page(page).Order(params["order"]).Select()

	this.json(ctx, gin.H{
		"data":  items,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}
