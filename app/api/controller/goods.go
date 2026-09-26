package controller

import (
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

const (
	goodsAllowFields = "title,description,cover,price,stock,status,type,deliver_type,deliver_content,cards,category,limit_per_user,min_exp,start_time,end_time,sort,json,text"
	goodsAllowQuery  = "id"
)

var goodsAllowFieldsSlice = []any{
	"title", "description", "cover", "price", "stock", "status", "type", "deliver_type", "deliver_content", "cards",
	"category", "limit_per_user", "min_exp", "start_time", "end_time", "sort", "json", "text",
}
var goodsAllowQuerySlice = []any{"id"}

// goodsOrderAllow - 商品列表排序白名单（防止 SQL 注入）
var goodsOrderAllow = []any{
	"sort desc, id desc", "sort asc, id desc",
	"create_time desc", "create_time asc",
	"price asc", "price desc",
	"stock desc", "stock asc",
	"sold desc", "id desc", "id asc",
}

type Goods struct {
	base
}

func (this *Goods) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Goods) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Goods) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":        this.one,
		"all":        this.all,
		"count":      this.count,
		"categories": this.categories,
		"orders":     this.orders,
		"orders-all": this.ordersAll,
		"order-one":  this.orderOne,
		"my-stats":   this.myStats,
		"stats":      this.stats,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Goods) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"buy":    this.buy,
		"save":   this.save,
		"create": this.create,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Goods) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":       this.update,
		"restore":      this.restore,
		"order-status": this.orderStatus,
		"cancel-order": this.cancelOrder,
		"receive":      this.receive,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Goods) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove": this.remove,
		"delete": this.delete,
		"clear":  this.clear,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Goods) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Goods) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "goods"})
}

func (this *Goods) processFieldValue(val any) any {
	switch utils.Get.Type(val) {
	case "map":
		return utils.Json.Encode(val)
	case "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

// sanitizeGoods - 非管理员隐藏敏感字段（卡密池、文本发货内容）
func (this *Goods) sanitizeGoods(ctx *gin.Context, data any) {
	if this.meta.permit(ctx) {
		return
	}
	sanitize := func(m map[string]any) {
		delete(m, "cards")
		delete(m, "deliver_content")
	}
	switch v := data.(type) {
	case map[string]any:
		sanitize(v)
	case []map[string]any:
		for _, m := range v {
			sanitize(m)
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				sanitize(m)
			}
		}
	}
}

// goodsOrder - 排序白名单校验
func (this *Goods) goodsOrder(order string) string {
	order = strings.ToLower(strings.TrimSpace(order))
	if utils.Is.Empty(order) {
		return "sort desc, id desc"
	}
	if utils.In.Array(order, goodsOrderAllow) {
		return order
	}
	return "sort desc, id desc"
}

// toGoodsSlice - 把 WithField 结果统一为 []map[string]any
func (this *Goods) toGoodsSlice(data any) []map[string]any {
	switch v := data.(type) {
	case []map[string]any:
		return v
	case []any:
		list := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				list = append(list, m)
			}
		}
		return list
	case map[string]any:
		if len(v) == 0 {
			return nil
		}
		return []map[string]any{v}
	}
	return nil
}

// decorateBuyState - 为商品列表补充「可兑换状态」：限购剩余 / 是否可兑换 / 不可兑换原因
// 只做一次用户与订单聚合查询，避免逐条查询造成 N+1
func (this *Goods) decorateBuyState(ctx *gin.Context, items []map[string]any) {
	uid := this.user(ctx).Id
	userExp, userIntegral := 0, 0
	boughtMap := make(map[int]int)

	if uid > 0 {
		user, _ := facade.DB.Model(&model.Users{}).Where("id", uid).Find()
		userExp = cast.ToInt(user["exp"])
		userIntegral = cast.ToInt(user["integral"])

		var rows []map[string]any
		if err := facade.DB.Drive().Raw(
			"SELECT goods_id, COUNT(id) AS count FROM inis_goods_order "+
				"WHERE uid = ? AND status != ? AND (delete_time IS NULL OR delete_time = 0) GROUP BY goods_id",
			uid, model.OrderStatusCanceled,
		).Scan(&rows).Error; err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "uid": uid}, "统计用户已兑换数量失败")
		}

		for _, row := range rows {
			boughtMap[cast.ToInt(row["goods_id"])] = cast.ToInt(row["count"])
		}
	}

	now := time.Now().Unix()

	for _, item := range items {
		price := cast.ToInt(item["price"])
		stock := cast.ToInt(item["stock"])
		limit := cast.ToInt(item["limit_per_user"])
		minExp := cast.ToInt(item["min_exp"])
		start := cast.ToInt64(item["start_time"])
		end := cast.ToInt64(item["end_time"])

		mine := boughtMap[cast.ToInt(item["id"])]

		// 限购剩余（-1 表示不限购）
		remain := -1
		if limit > 0 {
			remain = limit - mine
			if remain < 0 {
				remain = 0
			}
		}

		canBuy, reason := true, ""
		switch {
		case stock <= 0:
			canBuy, reason = false, "已售罄"
		case start > 0 && now < start:
			canBuy, reason = false, "未开始"
		case end > 0 && now > end:
			canBuy, reason = false, "已结束"
		case uid == 0:
			// 未登录：商品维度校验通过，是否可兑换交由前端登录判断
		case minExp > 0 && userExp < minExp:
			canBuy, reason = false, fmt.Sprintf("需经验 %d", minExp)
		case limit > 0 && mine >= limit:
			canBuy, reason = false, "已达限购"
		case userIntegral < price:
			canBuy, reason = false, "积分不足"
		}

		item["my_bought"] = mine
		item["limit_remain"] = remain
		item["can_buy"] = canBuy
		item["buy_reason"] = reason
	}
}

// one - 商品详情
func (this *Goods) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Goods{}

	for key, val := range params {
		if utils.In.Array(key, goodsAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	query := this.withTrashOptions(facade.DB.Model(&table), params)
	query = this.buildQuery(query, params)
	item, _ := query.Where(table).Find()
	data = facade.Comm.WithField(item, params["field"])

	if !utils.Is.Empty(data) {
		// 补充兑换状态（详情页据此展示按钮文案）
		if list := this.toGoodsSlice(data); len(list) > 0 {
			this.decorateBuyState(ctx, list)
		}
		this.sanitizeGoods(ctx, data)
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all - 商品列表（公开仅展示上架商品，管理员可通过 status 参数查全部）
// 支持 category 分类筛选、keyword 关键词搜索
func (this *Goods) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "sort desc, id desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Goods

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 普通用户仅展示上架商品；管理员默认查看全部，可传 status 参数过滤
	if this.meta.permit(ctx) {
		if !utils.Is.Empty(params["status"]) {
			query = query.Where("status", params["status"])
		}
	} else {
		query = query.Where("status", model.GoodsStatusOn)
	}

	// 分类筛选
	if category := cast.ToString(params["category"]); !utils.Is.Empty(category) {
		query = query.Where("category", category)
	}

	// 关键词搜索（商品名称）
	if keyword := cast.ToString(params["keyword"]); !utils.Is.Empty(keyword) {
		query = query.Like("title", "%"+keyword+"%")
	}

	count, _ := query.Count()
	items, _ := query.Order(this.goodsOrder(cast.ToString(params["order"]))).Limit(limit).Page(page).Select()
	data = utils.ArrayMapWithField(items, params["field"])

	if list := this.toGoodsSlice(data); len(list) > 0 {
		this.decorateBuyState(ctx, list)
	}
	this.sanitizeGoods(ctx, data)

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

// categories - 商品分类聚合（公开，仅统计上架商品）
func (this *Goods) categories(ctx *gin.Context) {
	params := this.params(ctx)

	var rows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT category, COUNT(id) AS count FROM inis_goods "+
			"WHERE status = ? AND (delete_time IS NULL OR delete_time = 0) AND category IS NOT NULL AND category != '' "+
			"GROUP BY category ORDER BY count DESC",
		model.GoodsStatusOn,
	).Scan(&rows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "商品分类聚合失败")
	}

	list := make([]facade.H, 0, len(rows)+1)
	// 首个为「全部」（count 为上架商品总数）
	var totalRows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS total FROM inis_goods WHERE status = ? AND (delete_time IS NULL OR delete_time = 0)",
		model.GoodsStatusOn,
	).Scan(&totalRows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "商品总数统计失败")
	}
	var total int
	if len(totalRows) > 0 {
		total = cast.ToInt(totalRows[0]["total"])
	}

	list = append(list, facade.H{"category": "", "name": "全部", "count": total})
	for _, row := range rows {
		category := cast.ToString(row["category"])
		list = append(list, facade.H{
			"category": category,
			"name":     category,
			"count":    cast.ToInt(row["count"]),
		})
	}

	// 允许通过配置覆盖分类展示名：goods.category-alias = {"vip":"会员"}
	if alias, ok := params["alias"].(map[string]any); ok && len(alias) > 0 {
		for index, item := range list {
			if name, ok := alias[cast.ToString(item["category"])]; ok {
				list[index]["name"] = cast.ToString(name)
			}
		}
	}

	this.json(ctx, list, facade.Lang(ctx, "查询成功！"), 200)
}

// count - 商品数量
func (this *Goods) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.buildQuery(facade.DB.Model(&model.Goods{}), params)
	if !this.meta.permit(ctx) {
		query = query.Where("status", model.GoodsStatusOn)
	}
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

// orders - 我的订单列表（仅当前用户自己的订单）
func (this *Goods) orders(ctx *gin.Context) {
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
	var result []model.GoodsOrder

	query := facade.DB.Model(&result).Where("uid", user.Id)
	// 状态筛选（支持 0/1/2/3）
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}
	if !utils.Is.Empty(params["goods_id"]) {
		query = query.Where("goods_id", params["goods_id"])
	}

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

// orderOne - 订单详情（仅本人或管理员）
func (this *Goods) orderOne(ctx *gin.Context) {
	params := this.params(ctx)
	orderId := cast.ToInt(params["id"])

	if orderId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	var order model.GoodsOrder
	if err := facade.DB.Drive().Where("id = ?", orderId).First(&order).Error; err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "订单不存在！"), 204)
		return
	}

	if !this.meta.permit(ctx) && order.Uid != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	this.json(ctx, order, facade.Lang(ctx, "查询成功！"), 200)
}

// ordersAll - 全部订单列表（管理员专用，支持 uid/status 过滤）
func (this *Goods) ordersAll(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.GoodsOrder

	query := facade.DB.Model(&result)
	if !utils.Is.Empty(params["uid"]) {
		query = query.Where("uid", params["uid"])
	}
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}
	if !utils.Is.Empty(params["goods_id"]) {
		query = query.Where("goods_id", params["goods_id"])
	}

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

// myStats - 我的兑换统计（登录）
func (this *Goods) myStats(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 用 map + cast 取值：SUM() 在 MySQL 下返回 DECIMAL，直接 Scan 到结构体整型字段
	// 在部分驱动/场景下会取值失败并被静默忽略（表现为全部为 0），这里统一走 map。
	var rows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS total, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS pending, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS shipped, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS finished, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS canceled, "+
			"COALESCE(SUM(CASE WHEN status != ? THEN price ELSE 0 END), 0) AS spent "+
			"FROM inis_goods_order WHERE uid = ? AND (delete_time IS NULL OR delete_time = 0)",
		model.OrderStatusPending, model.OrderStatusShipped, model.OrderStatusCompleted, model.OrderStatusCanceled,
		model.OrderStatusCanceled, user.Id,
	).Scan(&rows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "uid": user.Id}, "我的兑换统计查询失败")
		this.json(ctx, nil, facade.Lang(ctx, "统计查询失败：%v", err.Error()), 400)
		return
	}

	row := map[string]any{}
	if len(rows) > 0 {
		row = rows[0]
	}

	this.json(ctx, gin.H{
		"order_total":    cast.ToInt(row["total"]),
		"pending":        cast.ToInt(row["pending"]),
		"shipped":        cast.ToInt(row["shipped"]),
		"completed":      cast.ToInt(row["finished"]),
		"canceled":       cast.ToInt(row["canceled"]),
		"integral_spent": cast.ToInt(row["spent"]),
		"integral":       model.IntegralBalance(user.Id),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// stats - 商城统计（管理员）
func (this *Goods) stats(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	// 商品统计（用 map + cast 取值，避免 SUM(DECIMAL) 到结构体整型的静默取值失败）
	var goodsRows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS total, "+
			"COALESCE(SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END), 0) AS on_sale, "+
			"COALESCE(SUM(CASE WHEN status = 0 THEN 1 ELSE 0 END), 0) AS off_sale, "+
			"COALESCE(SUM(CASE WHEN status = 1 AND stock <= 5 THEN 1 ELSE 0 END), 0) AS stock_warn "+
			"FROM inis_goods WHERE (delete_time IS NULL OR delete_time = 0)",
	).Scan(&goodsRows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "商城统计-商品统计失败")
		this.json(ctx, nil, facade.Lang(ctx, "统计查询失败：%v", err.Error()), 400)
		return
	}

	var orderRows []map[string]any
	if err := facade.DB.Drive().Raw(
		"SELECT COUNT(id) AS total, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS pending, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS shipped, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS finished, "+
			"COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS canceled, "+
			"COALESCE(SUM(CASE WHEN status != ? THEN price ELSE 0 END), 0) AS spent, "+
			"COALESCE(SUM(refund), 0) AS refund "+
			"FROM inis_goods_order WHERE (delete_time IS NULL OR delete_time = 0)",
		model.OrderStatusPending, model.OrderStatusShipped, model.OrderStatusCompleted,
		model.OrderStatusCanceled, model.OrderStatusCanceled,
	).Scan(&orderRows).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "商城统计-订单统计失败")
		this.json(ctx, nil, facade.Lang(ctx, "统计查询失败：%v", err.Error()), 400)
		return
	}

	goodsRow, orderRow := map[string]any{}, map[string]any{}
	if len(goodsRows) > 0 {
		goodsRow = goodsRows[0]
	}
	if len(orderRows) > 0 {
		orderRow = orderRows[0]
	}

	spent := cast.ToInt(orderRow["spent"])
	refund := cast.ToInt(orderRow["refund"])

	this.json(ctx, gin.H{
		"goods": gin.H{
			"total":      cast.ToInt(goodsRow["total"]),
			"on":         cast.ToInt(goodsRow["on_sale"]),
			"off":        cast.ToInt(goodsRow["off_sale"]),
			"stock_warn": cast.ToInt(goodsRow["stock_warn"]),
		},
		"order": gin.H{
			"total":     cast.ToInt(orderRow["total"]),
			"pending":   cast.ToInt(orderRow["pending"]),
			"shipped":   cast.ToInt(orderRow["shipped"]),
			"completed": cast.ToInt(orderRow["finished"]),
			"canceled":  cast.ToInt(orderRow["canceled"]),
		},
		"integral": gin.H{
			"spent":  spent,
			"refund": refund,
			"net":    spent - refund,
		},
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// orderStatus - 更新订单状态（管理员：0待发货 1已发货 2已完成 3已取消）
// 置为 3（已取消）时会自动退还积分并回滚库存
func (this *Goods) orderStatus(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	orderId := cast.ToInt(params["id"])
	status := cast.ToInt(params["status"])
	logistics := cast.ToString(params["logistics"])

	if orderId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}
	if status < model.OrderStatusPending || status > model.OrderStatusCanceled {
		this.json(ctx, nil, facade.Lang(ctx, "无效的订单状态！"), 400)
		return
	}

	// 取消订单：走退款流程（退还积分 + 回滚库存）
	if status == model.OrderStatusCanceled {
		order, err := (&model.GoodsOrder{}).CancelOrder(0, orderId, true)
		if err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}

		// 取消并退款：通知买家（开关见「系统设置 → 邮件通知」的 order.canceled）
		go model.MailNotifyUser(order.Uid, "order.canceled", "您的订单已取消并退还积分",
			"商品："+order.GoodsTitle,
			"订单号："+order.OrderNo,
			"退还积分："+cast.ToString(order.Refund),
			"时间："+model.MailNotifyTime(),
		)

		this.json(ctx, gin.H{"id": orderId, "status": status}, facade.Lang(ctx, "订单已取消并退还积分！"), 200)
		return
	}

	update := map[string]any{"status": status}
	if !utils.Is.Empty(logistics) {
		update["logistics"] = logistics
	}
	if status == model.OrderStatusCompleted {
		update["finish_time"] = time.Now().Unix()
	}

	_, err := facade.DB.Model(&model.GoodsOrder{}).Where("id", orderId).Update(update)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 发货：通知买家（开关见「系统设置 → 邮件通知」的 order.shipped）
	if status == model.OrderStatusShipped {
		order, _ := facade.DB.Model(&model.GoodsOrder{}).Find(orderId)
		if !utils.Is.Empty(order) {
			go model.MailNotifyUser(cast.ToInt(order["uid"]), "order.shipped", "您的订单已发货",
				"商品："+cast.ToString(order["goods_title"]),
				"订单号："+cast.ToString(order["order_no"]),
				"物流："+logistics,
				"时间："+model.MailNotifyTime(),
			)
		}
	}

	this.json(ctx, gin.H{"id": orderId, "status": status}, facade.Lang(ctx, "更新成功！"), 200)
}

// cancelOrder - 取消订单（用户取消自己的待发货订单，积分原路退还）
func (this *Goods) cancelOrder(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	orderId := cast.ToInt(params["id"])
	if orderId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	order, err := (&model.GoodsOrder{}).CancelOrder(user.Id, orderId, this.meta.permit(ctx))
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{
		"id":       order.Id,
		"status":   order.Status,
		"refund":   order.Refund,
		"integral": model.IntegralBalance(order.Uid),
	}, facade.Lang(ctx, "订单已取消，积分已退还！"), 200)
}

// receive - 确认收货（已发货 → 已完成）
func (this *Goods) receive(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	orderId := cast.ToInt(params["id"])
	if orderId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	order, err := (&model.GoodsOrder{}).ReceiveOrder(user.Id, orderId)
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"id": order.Id, "status": order.Status}, facade.Lang(ctx, "确认收货成功！"), 200)
}

// buy - 购买商品
func (this *Goods) buy(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	goodsId := cast.ToInt(params["goods_id"])
	if goodsId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "goods_id"), 400)
		return
	}

	order, err := (&model.Goods{}).Buy(user.Id, goodsId, cast.ToString(params["address"]))
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	// 返回限购剩余，便于前端即时刷新按钮状态
	limitRemain := -1
	goods := &model.Goods{}
	if findErr := facade.DB.Drive().Where("id = ?", goodsId).First(goods).Error; findErr == nil {
		_, _, limitRemain = (&model.Goods{}).CanBuy(user.Id, goods)
	}

	this.json(ctx, gin.H{
		"order_id":        order.Id,
		"order_no":        order.OrderNo,
		"price":           order.Price,
		"integral":        model.IntegralBalance(user.Id),
		"status":          order.Status,
		"deliver_content": order.DeliverContent,
		"limit_remain":    limitRemain,
	}, facade.Lang(ctx, "购买成功！"), 200)
}

// save - 保存商品（管理员）
func (this *Goods) save(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create - 创建商品（管理员）
func (this *Goods) create(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	now := time.Now().Unix()
	table := model.Goods{CreateTime: now, UpdateTime: now}

	for key, val := range params {
		if utils.In.Array(key, goodsAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	_, err := facade.DB.Model(&table).Create(&table)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update - 更新商品（管理员）
func (this *Goods) update(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	table := model.Goods{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, goodsAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	_, err := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// remove - 软删除商品（管理员）
func (this *Goods) remove(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Goods{})
	columnData, _ := query.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := query.Delete(ids)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// delete - 彻底删除商品（管理员）
func (this *Goods) delete(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Goods{}).WithTrashed()
	columnData, _ := query.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := query.Force().Delete(ids)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// clear - 清空回收站（管理员）
func (this *Goods) clear(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	query := facade.DB.Model(&model.Goods{}).OnlyTrashed()
	columnData, _ := query.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := query.Force().Delete()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

// restore - 恢复商品（管理员）
func (this *Goods) restore(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Goods{}).OnlyTrashed().WhereIn("id", ids)
	columnData, _ := query.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	_, err := facade.DB.Model(&model.Goods{}).OnlyTrashed().Restore(ids)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}
