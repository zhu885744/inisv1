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

// status - 查询当前用户积分余额
func (this *Integral) status(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	this.json(ctx, gin.H{
		"integral": model.IntegralBalance(user.Id),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// rules - 获取任务规则列表（哪些行为能赚积分）
func (this *Integral) rules(ctx *gin.Context) {
	config := model.GetIntegralConfig()
	result := make([]facade.H, 0, len(config))
	for key, rule := range config {
		result = append(result, facade.H{
			"type":        key,
			"name":        rule["name"],
			"value":       rule["value"],
			"daily_limit": rule["daily_limit"],
		})
	}
	this.json(ctx, result, facade.Lang(ctx, "查询成功！"), 200)
}

// all - 积分流水列表（当前登录用户）
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
	var result []model.Integral

	query := facade.DB.Model(&result).Where("uid", user.Id)
	count, _ := query.Count()
	items, _ := query.Order(params["order"]).Limit(limit).Page(page).Select()
	data = utils.ArrayMapWithField(items, params["field"])

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

// give - 管理员调整积分（正=发放 负=扣除）
func (this *Integral) give(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx)
	targetUid := cast.ToInt(params["uid"])
	value := cast.ToInt(params["value"])

	if targetUid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}
	if value == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "积分值不能为0！"), 400)
		return
	}

	userItem, _ := facade.DB.Model(&model.Users{}).Where("id", targetUid).Find()
	if utils.Is.Empty(userItem) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 400)
		return
	}

	err := (&model.Integral{}).Add(model.Integral{
		Uid:         targetUid,
		Value:       value,
		Type:        "give",
		Description: cast.ToString(params["description"]),
	})

	if err != nil {
		this.json(ctx, gin.H{"value": 0}, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"uid": targetUid, "value": value}, facade.Lang(ctx, "调整成功！"), 200)
}
