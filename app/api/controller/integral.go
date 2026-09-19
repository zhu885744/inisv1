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

type Integral struct {
	base
}

func (this *Integral) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"status": this.status,
		"all":    this.all,
		"rules":  this.rules,
		"tasks":  this.tasks,
		"rank":   this.rank,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Integral) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"give": this.give,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Integral) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "方法调用错误！"), 405)
}

func (this *Integral) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "方法调用错误！"), 405)
}

func (this *Integral) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Integral) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "integral"})
}

// status - 查询当前用户积分概览（余额 + 累计收支 + 今日收支）
func (this *Integral) status(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	this.json(ctx, model.IntegralSummary(user.Id), facade.Lang(ctx, "查询成功！"), 200)
}

// rules - 获取任务规则列表（哪些行为能赚积分）
func (this *Integral) rules(ctx *gin.Context) {
	config := model.GetIntegralConfig()
	result := make([]facade.H, 0, len(config))
	for key, rule := range config {
		icon := cast.ToString(rule["icon"])
		if utils.Is.Empty(icon) {
			icon = "bi-coin"
		}
		result = append(result, facade.H{
			"type":        key,
			"name":        rule["name"],
			"value":       cast.ToInt(rule["value"]),
			"daily_limit": cast.ToInt(rule["daily_limit"]),
			"icon":        icon,
		})
	}
	this.json(ctx, result, facade.Lang(ctx, "查询成功！"), 200)
}

// tasks - 今日积分任务进度（登录用户）
// 返回各任务今日完成次数 / 进度 / 还能获得多少积分，以及今日合计收益与连续签到天数
func (this *Integral) tasks(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	this.json(ctx, model.IntegralTasks(user.Id), facade.Lang(ctx, "查询成功！"), 200)
}

// rank - 积分排行榜（公开接口）
// by: earned（累计获得，默认）/ balance（当前余额）；limit: 默认 20，最大 100
func (this *Integral) rank(ctx *gin.Context) {
	params := this.params(ctx)

	by := cast.ToString(params["by"])
	if utils.Is.Empty(by) {
		by = "earned"
	}
	limit := cast.ToInt(params["limit"])
	if limit <= 0 {
		limit = 20
	}

	data := model.IntegralRank(by, limit, this.user(ctx).Id)

	this.json(ctx, data, facade.Lang(ctx, "查询成功！"), 200)
}

// integralFilter - 积分流水筛选条件（列表与统计共用，保证口径一致）
// 支持：type 类型 / direction 收支方向 / start、end 时间范围 / keyword 描述关键词
func (this *Integral) integralFilter(params map[string]any, uid int) (where string, args []any) {
	conditions := []string{"uid = ?", "(delete_time IS NULL OR delete_time = 0)"}
	args = []any{uid}

	if typ := cast.ToString(params["type"]); !utils.Is.Empty(typ) {
		conditions = append(conditions, "type = ?")
		args = append(args, typ)
	}

	switch cast.ToString(params["direction"]) {
	case "income":
		conditions = append(conditions, "value > 0")
	case "expense":
		conditions = append(conditions, "value < 0")
	}

	if start := cast.ToInt64(params["start"]); start > 0 {
		conditions = append(conditions, "create_time >= ?")
		args = append(args, start)
	}
	if end := cast.ToInt64(params["end"]); end > 0 {
		conditions = append(conditions, "create_time <= ?")
		args = append(args, end)
	}
	if keyword := cast.ToString(params["keyword"]); !utils.Is.Empty(keyword) {
		conditions = append(conditions, "description LIKE ?")
		args = append(args, "%"+keyword+"%")
	}

	return strings.Join(conditions, " AND "), args
}

// all - 积分流水列表（当前登录用户）
// 支持按类型 / 收支方向 / 时间范围 / 关键词筛选，并返回筛选区间内的收支合计
func (this *Integral) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	where, args := this.integralFilter(params, user.Id)

	// 排序白名单，防止 SQL 注入
	order := strings.ToLower(cast.ToString(params["order"]))
	if !utils.In.Array(order, []any{"create_time desc", "create_time asc", "value desc", "value asc", "id desc", "id asc"}) {
		order = "create_time desc"
	}

	// 区间收支合计
	var summary struct {
		Income  int
		Expense int
	}
	facade.DB.Drive().Raw(
		"SELECT COALESCE(SUM(CASE WHEN value > 0 THEN value ELSE 0 END), 0) AS income, "+
			"COALESCE(SUM(CASE WHEN value < 0 THEN -value ELSE 0 END), 0) AS expense "+
			"FROM inis_integral WHERE "+where, args...,
	).Scan(&summary)

	// 总数
	var count int64
	facade.DB.Drive().Raw("SELECT COUNT(*) FROM inis_integral WHERE "+where, args...).Scan(&count)

	// 分页数据（用原生 SQL 保证与统计口径完全一致）
	offset := max(0, (page-1)*limit)
	listArgs := append(append([]any{}, args...), limit, offset)
	var items []map[string]any
	facade.DB.Drive().Raw(
		"SELECT id, uid, value, type, description, `json`, create_time, update_time "+
			"FROM inis_integral WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?", listArgs...,
	).Scan(&items)

	for _, item := range items {
		item["json"] = utils.Json.Decode(item["json"])
	}
	data = items

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
		"summary": gin.H{
			"income":  summary.Income,
			"expense": summary.Expense,
			"net":     summary.Income - summary.Expense,
		},
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// give - 管理员调整积分（正=发放 负=扣除）
// 支持单个 uid 或 uids 数组批量发放
func (this *Integral) give(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	value := cast.ToInt(params["value"])
	description := cast.ToString(params["description"])

	if value == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "积分值不能为0！"), 400)
		return
	}

	// 兼容单个 uid 与批量 uids
	rawIds := utils.Unity.Ids(params["uids"])
	if utils.Is.Empty(rawIds) {
		rawIds = utils.Unity.Ids(params["uid"])
	}
	if utils.Is.Empty(rawIds) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	// 统一转为 int 数组
	uids := make([]int, 0, len(rawIds))
	for _, item := range rawIds {
		uids = append(uids, cast.ToInt(item))
	}

	// 单个用户：保持原有返回结构（并附带最新余额）
	if len(uids) == 1 {
		targetUid := uids[0]
		userItem, _ := facade.DB.Model(&model.Users{}).Where("id", targetUid).Find()
		if utils.Is.Empty(userItem) {
			this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 400)
			return
		}

		err := (&model.Integral{}).Add(model.Integral{
			Uid:         targetUid,
			Value:       value,
			Type:        model.IntegralTypeGive,
			Description: description,
		})
		if err != nil {
			this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
			return
		}

		this.json(ctx, gin.H{
			"uid":      targetUid,
			"value":    value,
			"integral": model.IntegralBalance(targetUid),
		}, facade.Lang(ctx, "调整成功！"), 200)
		return
	}

	// 批量发放：逐个处理，失败不影响其它用户
	type result struct {
		Uid     int    `json:"uid"`
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Balance int    `json:"balance"`
	}

	list := make([]result, 0, len(uids))
	success := 0

	for _, targetUid := range uids {
		item := result{Uid: targetUid}

		if targetUid <= 0 {
			item.Msg = "无效的用户ID"
			list = append(list, item)
			continue
		}

		userItem, _ := facade.DB.Model(&model.Users{}).Where("id", targetUid).Find()
		if utils.Is.Empty(userItem) {
			item.Msg = "用户不存在"
			list = append(list, item)
			continue
		}

		err := (&model.Integral{}).Add(model.Integral{
			Uid:         targetUid,
			Value:       value,
			Type:        model.IntegralTypeGive,
			Description: description,
		})
		if err != nil {
			item.Msg = err.Error()
			list = append(list, item)
			continue
		}

		item.Success = true
		item.Msg = "成功"
		item.Balance = model.IntegralBalance(targetUid)
		success++
		list = append(list, item)
	}

	this.json(ctx, gin.H{
		"total":   len(uids),
		"success": success,
		"failed":  len(uids) - success,
		"list":    list,
	}, facade.Lang(ctx, "批量调整完成！"), 200)
}
