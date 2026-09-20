package controller

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 卡密兑换防刷参数：单位时间内失败次数达到上限后锁定，避免暴力枚举卡密
const (
	integralRedeemFailKey = "integral:card:redeem:fail:%d" // 失败计数缓存键
	integralRedeemLockKey = "integral:card:redeem:lock:%d" // 锁定标记缓存键
	integralRedeemMaxFail = 10                             // 允许的最大连续失败次数
	integralRedeemLockTTL = 600                            // 计数/锁定有效期（秒），即 10 分钟

	// integralCardExportLimit 单次导出未使用卡密的数量上限（防止一次性拉取过多明文卡密）
	integralCardExportLimit = 10000
)

type Integral struct {
	base
}

func (this *Integral) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"status":      this.status,
		"all":         this.all,
		"rules":       this.rules,
		"tasks":       this.tasks,
		"rank":        this.rank,
		"card-all":    this.cardAll,
		"card-stats":  this.cardStats,
		"card-export": this.cardExport,
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
		"give":          this.give,
		"card-generate": this.cardGenerate,
		"card-redeem":   this.cardRedeem,
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
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"card-remove": this.cardRemove,
		"card-delete": this.cardDelete,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
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

		// 积分变动后发送消息通知（仅管理员调整积分触发）
		balance := model.IntegralBalance(targetUid)
		model.IntegralGiveNotify(targetUid, value, balance, description)

		this.json(ctx, gin.H{
			"uid":      targetUid,
			"value":    value,
			"integral": balance,
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
		// 积分变动后发送消息通知（仅管理员调整积分触发）
		model.IntegralGiveNotify(targetUid, value, item.Balance, description)
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

// integralCardExpire - 解析卡密有效期参数
// 支持两种传参：
//  1. expire_time：秒级时间戳，0 表示永久有效；
//  2. expire：日期字符串（2006-01-02 或 2006-01-02 15:04:05），日期格式默认到当天 23:59:59 失效。
func integralCardExpire(params map[string]any) (int64, error) {
	if expireTime := cast.ToInt64(params["expire_time"]); expireTime > 0 {
		return expireTime, nil
	}

	expire := strings.TrimSpace(cast.ToString(params["expire"]))
	if utils.Is.Empty(expire) {
		return 0, nil
	}

	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02"} {
		parsed, err := time.ParseInLocation(layout, expire, time.Local)
		if err != nil {
			continue
		}
		// 仅日期时，有效期覆盖到当天结束
		if layout == "2006-01-02" {
			parsed = parsed.Add(24*time.Hour - time.Second)
		}
		return parsed.Unix(), nil
	}

	return 0, errors.New("卡密有效期格式不正确！")
}

// cardGenerate - 生成卡密（管理员）
// 参数：value 积分面额；count 生成数量（1~1000，默认1）；length 卡密长度（8~64，默认16）；
// expire_time/expire 有效期（0 或留空=永久）；remark 备注
func (this *Integral) cardGenerate(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	value := cast.ToInt(params["value"])
	count := cast.ToInt(params["count"])
	if count <= 0 {
		count = 1
	}
	length := cast.ToInt(params["length"])

	expireTime, err := integralCardExpire(params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	list, batch, err := model.GenerateIntegralCards(value, count, length, expireTime, cast.ToString(params["remark"]))
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 卡密明文仅在此接口返回一次，用于管理员导出下发
	cards := make([]string, 0, len(list))
	for _, item := range list {
		cards = append(cards, item.Card)
	}

	this.json(ctx, gin.H{
		"batch":       batch,
		"count":       len(list),
		"value":       value,
		"expire_time": expireTime,
		"cards":       cards,
		"list":        list,
	}, facade.Lang(ctx, "生成成功！"), 200)
}

// redeemLocked - 卡密兑换是否被锁定（超过失败次数上限）
func (this *Integral) redeemLocked(uid int) bool {
	return facade.Cache.Has(fmt.Sprintf(integralRedeemLockKey, uid))
}

// redeemFail - 记录一次兑换失败，达到上限后锁定
func (this *Integral) redeemFail(uid int) {
	key := fmt.Sprintf(integralRedeemFailKey, uid)
	count := cast.ToInt(facade.Cache.Get(key)) + 1
	facade.Cache.Set(key, count, integralRedeemLockTTL)

	if count >= integralRedeemMaxFail {
		facade.Cache.Set(fmt.Sprintf(integralRedeemLockKey, uid), 1, integralRedeemLockTTL)
	}
}

// redeemReset - 兑换成功后清除失败计数
func (this *Integral) redeemReset(uid int) {
	facade.Cache.Del(fmt.Sprintf(integralRedeemFailKey, uid))
}

// cardRedeem - 卡密兑换积分（登录用户）
func (this *Integral) cardRedeem(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 防暴力枚举：连续失败过多则临时锁定
	if this.redeemLocked(user.Id) {
		this.json(ctx, nil, facade.Lang(ctx, "尝试过于频繁，请 %d 分钟后再试！", integralRedeemLockTTL/60), 429)
		return
	}

	params := this.params(ctx)
	card := cast.ToString(params["card"])
	if utils.Is.Empty(strings.TrimSpace(card)) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "card"), 400)
		return
	}

	result, err := model.RedeemIntegralCard(user.Id, card)
	if err != nil {
		this.redeemFail(user.Id)
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.redeemReset(user.Id)

	this.json(ctx, result, facade.Lang(ctx, "兑换成功！"), 200)
}

// appendCardUser - 为卡密列表补充使用者昵称（一次性批量查询，避免 N+1）
// 未使用的卡密（uid <= 0）昵称置空，前端展示为占位符
func (this *Integral) appendCardUser(items []map[string]any) []map[string]any {
	uids := make([]int, 0)
	seen := make(map[int]struct{})

	for _, item := range items {
		uid := cast.ToInt(item["uid"])
		if uid <= 0 {
			item["nickname"] = ""
			continue
		}
		if _, exist := seen[uid]; exist {
			continue
		}
		seen[uid] = struct{}{}
		uids = append(uids, uid)
	}

	if len(uids) == 0 {
		return items
	}

	var users []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT id, nickname FROM inis_users WHERE id IN ?", uids,
	).Scan(&users).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "uids": uids}, "卡密使用者昵称查询失败")
		return items
	}

	nicknameMap := make(map[int]string, len(users))
	for _, user := range users {
		nicknameMap[cast.ToInt(user["id"])] = cast.ToString(user["nickname"])
	}

	for _, item := range items {
		if uid := cast.ToInt(item["uid"]); uid > 0 {
			item["nickname"] = nicknameMap[uid]
		}
	}

	return items
}

// cardAll - 卡密列表（管理员）
// 支持 status（0未使用 1已使用）、batch（批次）、uid（使用者）、value（面额）、
// keyword（卡密模糊）、expired（1已过期 / 0未过期含永久）筛选
func (this *Integral) cardAll(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "id desc",
	})
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	// 构造筛选条件（全部使用参数绑定，避免 SQL 注入）
	conditions := []string{"(delete_time IS NULL OR delete_time = 0)"}
	args := make([]any, 0)
	now := time.Now().Unix()

	if !utils.Is.Empty(params["status"]) {
		conditions = append(conditions, "status = ?")
		args = append(args, cast.ToInt(params["status"]))
	}
	if batch := cast.ToString(params["batch"]); !utils.Is.Empty(batch) {
		conditions = append(conditions, "batch = ?")
		args = append(args, batch)
	}
	if uid := cast.ToInt(params["uid"]); uid > 0 {
		conditions = append(conditions, "uid = ?")
		args = append(args, uid)
	}
	if value := cast.ToInt(params["value"]); value > 0 {
		conditions = append(conditions, "value = ?")
		args = append(args, value)
	}
	if keyword := strings.TrimSpace(cast.ToString(params["keyword"])); !utils.Is.Empty(keyword) {
		conditions = append(conditions, "card LIKE ?")
		args = append(args, "%"+strings.ToUpper(keyword)+"%")
	}
	// expired: 1 仅已过期 / 0 仅未过期（含永久）
	switch cast.ToString(params["expired"]) {
	case "1":
		conditions = append(conditions, "expire_time > 0 AND expire_time < ?")
		args = append(args, now)
	case "0":
		conditions = append(conditions, "(expire_time = 0 OR expire_time >= ?)")
		args = append(args, now)
	}

	where := strings.Join(conditions, " AND ")

	// 排序白名单，防止 SQL 注入
	order := strings.ToLower(strings.TrimSpace(cast.ToString(params["order"])))
	if !utils.In.Array(order, []any{
		"id desc", "id asc", "value desc", "value asc",
		"create_time desc", "create_time asc", "use_time desc", "use_time asc",
	}) {
		order = "id desc"
	}

	var count int64
	facade.DB.Drive().Raw("SELECT COUNT(*) FROM inis_integral_card WHERE "+where, args...).Scan(&count)

	offset := max(0, (page-1)*limit)
	listArgs := append(append([]any{}, args...), limit, offset)

	var items []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT * FROM inis_integral_card WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?", listArgs...,
	).Scan(&items).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "卡密列表查询失败")
		this.json(ctx, nil, facade.Lang(ctx, "查询失败！"), 400)
		return
	}
	// 补充使用者昵称（避免管理端直接展示裸露的 uid）
	items = this.appendCardUser(items)
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

// cardStats - 卡密统计（管理员）
// 返回：总数 / 未使用 / 已使用 / 已过期 / 累计发放积分 / 已兑换积分
func (this *Integral) cardStats(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	now := time.Now().Unix()

	var rows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS total, "+
			"COALESCE(SUM(CASE WHEN status = 0 AND (expire_time = 0 OR expire_time >= ?) THEN 1 ELSE 0 END), 0) AS unused, "+
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS used, "+
			"COALESCE(SUM(CASE WHEN status = 0 AND expire_time > 0 AND expire_time < ? THEN 1 ELSE 0 END), 0) AS expired, "+
			"COALESCE(SUM(value), 0) AS value_total, "+
			"COALESCE(SUM(CASE WHEN status = 1 THEN value ELSE 0 END), 0) AS value_used "+
			"FROM inis_integral_card WHERE (delete_time IS NULL OR delete_time = 0)",
		now, now,
	).Scan(&rows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "卡密统计查询失败")
		this.json(ctx, nil, facade.Lang(ctx, "统计查询失败！"), 400)
		return
	}

	row := map[string]any{}
	if len(rows) > 0 {
		row = rows[0]
	}

	this.json(ctx, gin.H{
		"total":       cast.ToInt(row["total"]),
		"unused":      cast.ToInt(row["unused"]),
		"used":        cast.ToInt(row["used"]),
		"expired":     cast.ToInt(row["expired"]),
		"value_total": cast.ToInt(row["value_total"]),
		"value_used":  cast.ToInt(row["value_used"]),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// cardExport - 导出未使用卡密（管理员，用于批量复制/下发）
// 仅导出「未使用且未过期」的卡密；支持 batch / value / keyword 可选筛选；
// 单次最多导出 integralCardExportLimit 张，超出时截断并通过 truncated 标记。
func (this *Integral) cardExport(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	now := time.Now().Unix()

	// 固定条件：未软删除 + 未使用 + 未过期
	conditions := []string{
		"(delete_time IS NULL OR delete_time = 0)",
		"status = ?",
		"(expire_time = 0 OR expire_time >= ?)",
	}
	args := []any{model.IntegralCardStatusUnused, now}

	if batch := strings.TrimSpace(cast.ToString(params["batch"])); !utils.Is.Empty(batch) {
		conditions = append(conditions, "batch = ?")
		args = append(args, batch)
	}
	if value := cast.ToInt(params["value"]); value > 0 {
		conditions = append(conditions, "value = ?")
		args = append(args, value)
	}
	if keyword := strings.TrimSpace(cast.ToString(params["keyword"])); !utils.Is.Empty(keyword) {
		conditions = append(conditions, "card LIKE ?")
		args = append(args, "%"+strings.ToUpper(keyword)+"%")
	}

	where := strings.Join(conditions, " AND ")

	// 多查一条用于判断是否被截断
	var cards []string
	listArgs := append(append([]any{}, args...), integralCardExportLimit+1)
	if err := facade.DB.Drive().Raw(
		"SELECT card FROM inis_integral_card WHERE "+where+" ORDER BY id ASC LIMIT ?", listArgs...,
	).Scan(&cards).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "导出未使用卡密失败")
		this.json(ctx, nil, facade.Lang(ctx, "导出失败！"), 400)
		return
	}

	truncated := len(cards) > integralCardExportLimit
	if truncated {
		cards = cards[:integralCardExportLimit]
	}

	// 卡密明文导出属敏感操作，记录操作日志便于审计
	facade.Log.Info(map[string]any{
		"user_id":   this.user(ctx).Id,
		"count":     len(cards),
		"truncated": truncated,
	}, "管理员导出未使用卡密")

	this.json(ctx, gin.H{
		"count":     len(cards),
		"truncated": truncated,
		"cards":     cards,
	}, facade.Lang(ctx, "导出成功！"), 200)
}

// cardRemove - 软删除卡密（管理员）
func (this *Integral) cardRemove(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.IntegralCard{})
	columnData, _ := query.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := query.Delete(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// cardDelete - 彻底删除卡密（管理员）
func (this *Integral) cardDelete(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.IntegralCard{}).WithTrashed()
	columnData, _ := query.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := query.Force().Delete(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}
