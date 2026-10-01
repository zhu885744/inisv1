package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// Checkin - 每日签到（独立模块）
//
// 签到原先寄生在经验模块（/api/exp/check-in*），现在独立成 /api/checkin/*：
//
//	GET  /api/checkin/status     签到状态（连签 / 周期 / 今日可得 / 里程碑 / 月进度 / 补签）
//	GET  /api/checkin/calendar   签到日历（按月）
//	GET  /api/checkin/rank       签到排行榜（公开）
//	GET  /api/checkin/rules      签到规则（公开：开关 / 周期 / 里程碑 / 资产清单）
//	POST /api/checkin/sign       签到
//	POST /api/checkin/makeup     补签（消耗积分，可在后台配置）
//
// 奖励不再只有「经验 + 积分」两种，而是由后台「签到」页的配置驱动（多资产 / 随机区间 /
// 触发概率 / 周期奖励 / 月全勤 …），发放走 model 的奖励引擎（app/model/reward.go）。
type Checkin struct {
	base
}

func (this *Checkin) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"status":   this.status,
		"calendar": this.calendar,
		"rank":     this.rank,
		"rules":    this.rules,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Checkin) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"sign":       this.sign,
		"makeup":     this.makeup,
		"card-stock": this.cardStock,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Checkin) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "方法调用错误！"), 405)
}

func (this *Checkin) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "方法调用错误！"), 405)
}

func (this *Checkin) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Checkin) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "checkin"})
}

// status - 签到状态（需登录）
//
// 未签到：返回「签到后」的预期奖励（含概率 / 区间）；已签到：返回今天的实际明细。
func (this *Checkin) status(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	this.json(ctx, model.CheckinStatus(user.Id), facade.Lang(ctx, "查询成功！"), 200)
}

// calendar - 签到日历（需登录）
//
// 参数：year 年、month 月（不传为当前月）
func (this *Checkin) calendar(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	this.json(ctx, model.CheckinCalendar(
		user.Id,
		cast.ToInt(params["year"]),
		cast.ToInt(params["month"]),
	), facade.Lang(ctx, "查询成功！"), 200)
}

// rank - 签到排行榜（公开）
//
// 参数：start / end 时间戳（默认本月）、limit 条数
func (this *Checkin) rank(ctx *gin.Context) {
	params := this.params(ctx)

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	if !utils.Is.Empty(params["start"]) {
		start = time.Unix(cast.ToInt64(params["start"]), 0)
	}
	if !utils.Is.Empty(params["end"]) {
		end = time.Unix(cast.ToInt64(params["end"]), 0)
	}

	user := this.user(ctx)

	this.json(ctx, model.CheckinRank(
		start.Unix(),
		end.Unix(),
		this.meta.limit(ctx),
		user.Id,
	), facade.Lang(ctx, "数据请求成功！"), 200)
}

// rules - 签到规则（公开）
func (this *Checkin) rules(ctx *gin.Context) {
	this.json(ctx, model.CheckinRules(), facade.Lang(ctx, "查询成功！"), 200)
}

// cardStock - 卡密库存查询（管理员）
//
// 后台「签到设置」页用它显示每个卡密奖励还剩多少张：传该奖励项当前填写的卡密内容（codes），
// 返回 { total, issued, remain }；顺带把新填的卡密补进库存（幂等），
// 这样 total 就等于管理员填写的数量（不用自己数），remain 是还能发几张（发完即失效）。
func (this *Checkin) cardStock(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	codes := model.ParseCardCodes(params["codes"])

	if len(codes) > model.RewardCardMaxCount {
		this.json(ctx, nil, facade.Lang(ctx, "单个奖励最多配置 %d 张卡密！", model.RewardCardMaxCount), 400)
		return
	}

	this.json(ctx, model.RewardCardStock(codes), facade.Lang(ctx, "查询成功！"), 200)
}

// sign - 签到（需登录）
func (this *Checkin) sign(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	result, err := model.CheckinDo(user.Id, ctx.ClientIP())
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "签到成功！"), 200)
}

// makeup - 补签（需登录）
//
// 参数：date 补签日期（yyyymmdd），不传则补昨天
func (this *Checkin) makeup(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	result, err := model.CheckinMakeup(user.Id, cast.ToInt(params["date"]), ctx.ClientIP())
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "补签成功！"), 200)
}
