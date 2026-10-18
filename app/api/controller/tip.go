package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// Tip - 积分打赏
//
// 接口：
//   GET  /api/tip/summary?bind_type=&bind_id=   某内容的打赏概览（次数 / 积分）+ 我的余额
//   GET  /api/tip/stats?uid=                    某用户收到的打赏统计（用户主页）
//   GET  /api/tip/list?bind_type=&bind_id=      某内容的打赏记录（公开）
//   GET  /api/tip/list?box=received|sent        我收到 / 我发出的打赏（需登录）
//   POST /api/tip/send                          打赏（需登录）
type Tip struct {
	base
}

const tipAllowFields = "bind_type,bind_id,value,message,to_uid"

var tipAllowFieldsSlice = []any{"bind_type", "bind_id", "value", "message", "to_uid"}

func (this *Tip) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"summary": this.summary,
		"stats":   this.stats,
		"list":    this.list,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Tip) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"send": this.send,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Tip) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "方法调用错误！"), 405)
}

func (this *Tip) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "打赏记录不支持删除！"), 405)
}

func (this *Tip) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// summary - 内容的打赏概览
//
// 公开接口：访客也能看到「已获打赏 N 次 / M 积分」，只有登录用户才返回自己的余额。
func (this *Tip) summary(ctx *gin.Context) {
	params := this.params(ctx)

	bindType := strings.ToLower(cast.ToString(params["bind_type"]))
	bindId := cast.ToInt(params["bind_id"])

	if utils.Is.Empty(bindType) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_type"), 400)
		return
	}

	data := model.TipSummaryOf(bindType, bindId)

	// 我的余额（未登录为 0）
	balance := 0
	if uid := this.user(ctx).Id; uid > 0 {
		balance = model.IntegralBalance(uid)
	}
	data["balance"] = balance
	data["min"] = model.TipMinValue
	data["max"] = model.TipMaxValue
	data["message_max"] = model.TipMessageMaxLength

	this.json(ctx, data, facade.Lang(ctx, "查询成功！"), 200)
}

// stats - 用户收到的打赏统计（用户主页展示）
func (this *Tip) stats(ctx *gin.Context) {
	params := this.params(ctx)

	uid := cast.ToInt(params["uid"])
	if uid <= 0 {
		uid = this.user(ctx).Id
	}
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请指定用户！"), 400)
		return
	}

	this.json(ctx, model.TipUserStats(uid), facade.Lang(ctx, "查询成功！"), 200)
}

// list - 打赏记录
//
// 两种用法：
//  1. 传 bind_type + bind_id：查看某条内容收到的打赏（公开，留言原样展示）；
//  2. 传 box=received|sent：查看自己的收/支记录（需登录），查看他人时隐藏留言。
func (this *Tip) list(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "id desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	uid := this.user(ctx).Id

	bindType := strings.ToLower(cast.ToString(params["bind_type"]))
	bindId := cast.ToInt(params["bind_id"])
	box := strings.ToLower(cast.ToString(params["box"]))

	query := facade.DB.Model(&[]model.Tip{})

	// 是否展示留言：内容维度公开；用户维度只有本人（或管理员）能看
	showMessage := true

	switch {
	case !utils.Is.Empty(bindType) && bindId > 0:
		query = query.Where("bind_type", bindType).Where("bind_id", bindId)

	case uid > 0 && (box == "sent" || box == "give"):
		query = query.Where("uid", uid)

	case uid > 0:
		target := cast.ToInt(params["uid"])
		if target <= 0 {
			target = uid
		}
		if target != uid && !this.meta.root(ctx) {
			showMessage = false
		}
		query = query.Where("to_uid", target)

	default:
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	count, _ := query.Count()
	items, _ := query.Order(params["order"]).Limit(limit).Page(page).Select()

	this.json(ctx, gin.H{
		"data":  this.formatTips(items, showMessage),
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// formatTips - 打赏记录补全用户信息（昵称 / 头像）
//
// 一次把涉及的 uid 全查出来，避免每条记录都查一遍用户表。
func (this *Tip) formatTips(items []map[string]any, showMessage bool) []facade.H {
	result := make([]facade.H, 0, len(items))
	if len(items) == 0 {
		return result
	}

	ids := make([]int, 0, len(items)*2)
	seen := make(map[int]bool, len(items)*2)
	for _, item := range items {
		for _, key := range []string{"uid", "to_uid"} {
			id := cast.ToInt(item[key])
			if id > 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	users := make(map[int]map[string]any, len(ids))
	rows, _ := facade.DB.Model(&[]model.Users{}).WhereIn("id", ids).Select()
	for _, row := range rows {
		users[cast.ToInt(row["id"])] = row
	}

	for _, item := range items {
		uid := cast.ToInt(item["uid"])
		toUid := cast.ToInt(item["to_uid"])

		message := ""
		if showMessage {
			message = cast.ToString(item["message"])
		}

		result = append(result, facade.H{
			"id":        cast.ToInt(item["id"]),
			"uid":       uid,
			"to_uid":    toUid,
			"value":     cast.ToInt(item["value"]),
			"bind_type": cast.ToString(item["bind_type"]),
			"bind_id":   cast.ToInt(item["bind_id"]),
			"message":   message,
			"time":      cast.ToInt64(item["create_time"]),
			"from": facade.H{
				"id":       uid,
				"nickname": cast.ToString(users[uid]["nickname"]),
				"avatar":   cast.ToString(users[uid]["avatar"]),
			},
			"to": facade.H{
				"id":       toUid,
				"nickname": cast.ToString(users[toUid]["nickname"]),
				"avatar":   cast.ToString(users[toUid]["avatar"]),
			},
		})
	}

	return result
}

// send - 打赏
func (this *Tip) send(ctx *gin.Context) {
	params := this.params(ctx)

	if err := validator.NewValid("tip", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	uid := this.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Tip{}
	for key, val := range params {
		if utils.In.Array(key, tipAllowFieldsSlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	data, err := table.Send(uid, cast.ToInt(params["to_uid"]), table.Value, table.BindType, table.BindId, table.Message)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "打赏成功，感谢支持！"), 200)
}
