package controller

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// 续期
//
// 续期档位不是全站统一的：每枚勋章 / 每件装扮在自己的 renew 字段里配置
// 「多少天 / 多少积分」，在后台的勋章、装扮编辑表单里维护（见 model/renew-plan.go）。
//
// 因此这里只有两个前台接口：
//   - POST renew/badge      { badge_id, days }
//   - POST renew/decoration { decoration_id, days }
//
// 只传 days 不传价格：价格由服务端从商品自己的档位里取，前端改不了价。
// 档位列表随「我的勋章」「我的装扮」一起下发（badge/mine、decoration/mine 的 renew_options），
// 不额外多一次价格请求。

type Renew struct {
	base
}

func (this *Renew) IGET(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "续期档位随勋章 / 装扮列表下发，无独立查询接口！"), 405)
}

func (this *Renew) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "续期档位在勋章 / 装扮的编辑表单里维护，无独立更新接口！"), 405)
}

func (this *Renew) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "续期档位在勋章 / 装扮的编辑表单里维护，无独立删除接口！"), 405)
}

func (this *Renew) IPOST(ctx *gin.Context) {
	method := ctx.Param("method")

	allow := map[string]any{
		// 续期下单：勋章 / 装扮各一个入口
		"badge":      this.badge,
		"decoration": this.decoration,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Renew) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// badge 勋章续期
func (this *Renew) badge(ctx *gin.Context) {

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	badgeId := cast.ToInt(params["badge_id"])
	if badgeId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "badge_id"), 400)
		return
	}

	days := cast.ToInt(params["days"])
	if days <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "days"), 400)
		return
	}

	result, err := model.RenewBadge(user.Id, badgeId, days)
	if err != nil {
		// 业务性拒绝（积分不足 / 未持有 / 永久资产 / 档位不存在）用 202，由前端直接提示
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "续期成功！"), 200)
}

// decoration 装扮续期
func (this *Renew) decoration(ctx *gin.Context) {

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	decorationId := cast.ToInt(params["decoration_id"])
	if decorationId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "decoration_id"), 400)
		return
	}

	days := cast.ToInt(params["days"])
	if days <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "days"), 400)
		return
	}

	result, err := model.RenewDecoration(user.Id, decorationId, days)
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "续期成功！"), 200)
}
