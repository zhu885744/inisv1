package controller

import (
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
	goodsAllowFields = "title,description,cover,price,stock,status,type,deliver_type,deliver_content,cards,json,text"
	goodsAllowQuery  = "id"
)

var goodsAllowFieldsSlice = []any{"title", "description", "cover", "price", "stock", "status", "type", "deliver_type", "deliver_content", "cards", "json", "text"}
var goodsAllowQuerySlice = []any{"id"}

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
		"one":       this.one,
		"all":       this.all,
		"orders":    this.orders,
		"orders-all": this.ordersAll,
		"count":     this.count,
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
	if this.meta.root(ctx) {
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
	this.sanitizeGoods(ctx, data)

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all - 商品列表（公开仅展示上架商品，管理员可通过 status 参数查全部）
func (this *Goods) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Goods

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)

	// 普通用户仅展示上架商品；管理员默认查看全部，可传 status 参数过滤
	if this.meta.root(ctx) {
		if !utils.Is.Empty(params["status"]) {
			query = query.Where("status", params["status"])
		}
	} else {
		query = query.Where("status", model.GoodsStatusOn)
	}

	count, _ := query.Count()
	items, _ := query.Order(params["order"]).Limit(limit).Page(page).Select()
	data = utils.ArrayMapWithField(items, params["field"])
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

// count - 商品数量
func (this *Goods) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.buildQuery(facade.DB.Model(&model.Goods{}), params)
	if !this.meta.root(ctx) {
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

// ordersAll - 全部订单列表（管理员专用，支持 uid/status 过滤）
func (this *Goods) ordersAll(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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

// orderStatus - 更新订单状态（管理员：0待发货 1已发货 2已完成）
func (this *Goods) orderStatus(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if status < model.OrderStatusPending || status > model.OrderStatusCompleted {
		this.json(ctx, nil, facade.Lang(ctx, "无效的订单状态！"), 400)
		return
	}

	update := map[string]any{"status": status}
	if !utils.Is.Empty(logistics) {
		update["logistics"] = logistics
	}

	_, err := facade.DB.Model(&model.GoodsOrder{}).Where("id", orderId).Update(update)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": orderId, "status": status}, facade.Lang(ctx, "更新成功！"), 200)
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

	this.json(ctx, gin.H{
		"order_id":        order.Id,
		"price":           order.Price,
		"integral":        model.IntegralBalance(user.Id),
		"status":          order.Status,
		"deliver_content": order.DeliverContent,
	}, facade.Lang(ctx, "购买成功！"), 200)
}

// save - 保存商品（管理员）
func (this *Goods) save(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
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
