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

// decorationAllowFields - 管理员可写字段白名单
// stock / min_exp 是装扮商城自身的售卖字段（不再同步到商品记录）
// 注：limit_per_user（每人限购）已废弃、不再生效，故不在白名单内（列与 json 字段保留兼容老数据）
const decorationAllowFields = "type,name,description,preview,payload,price_type,price,rarity,duration,unlock,category,status,sort,is_default,stock,min_exp,json,text"

var decorationAllowFieldsSlice = []any{
	"type", "name", "description", "preview", "payload", "price_type", "price", "rarity",
	"duration", "unlock", "category", "status", "sort", "is_default",
	"stock", "min_exp", "json", "text",
}

var decorationAllowQuerySlice = []any{"id", "type", "status"}

// decorationOrderAllow - 排序白名单（防止 SQL 注入）
var decorationOrderAllow = []any{
	"sort desc, id desc", "sort asc, id asc",
	"create_time desc", "create_time asc",
	"price asc", "price desc",
	"id desc", "id asc",
}

type Decoration struct {
	base
}

func (this *Decoration) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":     this.one,
		"all":     this.all,
		"shop":    this.shop,
		"types":   this.types,
		"mine":    this.mine,
		"wearing": this.wearing,
		// 注意：不能叫 user，会与基类的 this.user(ctx) 取当前登录用户的方法重名
		"user-decorations": this.userDecorations,
		"count":            this.count,
		"config":  this.config,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Decoration) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"wear":   this.wear,
		"unwear": this.unwear,
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

func (this *Decoration) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
		"grant":   this.grant,
		"revoke":  this.revoke,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *Decoration) IDEL(ctx *gin.Context) {
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

func (this *Decoration) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Decoration) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "decoration"})
}

func (this *Decoration) processFieldValue(val any) any {
	switch utils.Get.Type(val) {
	case "map", "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

// decorationOrder - 排序白名单校验
func (this *Decoration) decorationOrder(order string) string {
	order = strings.ToLower(strings.TrimSpace(order))
	if utils.Is.Empty(order) {
		return "sort desc, id desc"
	}
	if utils.In.Array(order, decorationOrderAllow) {
		return order
	}
	return "sort desc, id desc"
}

// ============================== 读取 ==============================

// types - 装扮类型元数据（公开）
//
// 前端据此动态渲染商城分区与装扮编辑表单：新增装扮类型时前端无需改代码。
func (this *Decoration) types(ctx *gin.Context) {
	this.json(ctx, gin.H{
		"types":  model.DecorationTypeMeta(),
		"config": model.GetDecorationConfig(),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// config - 装扮配置（公开，前端据此决定是否展示入口）
func (this *Decoration) config(ctx *gin.Context) {
	this.json(ctx, model.GetDecorationConfig(), facade.Lang(ctx, "查询成功！"), 200)
}

// shop - 装扮商城列表（公开，仅上架，价格 / 库存 / 限购取自装扮自身）
func (this *Decoration) shop(ctx *gin.Context) {
	params := this.params(ctx)

	if !model.DecorationEnabled() {
		this.json(ctx, nil, facade.Lang(ctx, "装扮商城未开启！"), 200)
		return
	}

	// price_type：free 只看免费 / paid 只看付费（= 非免费），空=全部
	// 注：装扮已不做「每人限购」，因此这里不再需要传入当前用户
	list := model.ShopDecorations(
		cast.ToString(params["type"]),
		cast.ToString(params["price_type"]),
	)

	this.json(ctx, gin.H{
		"data":  list,
		"count": len(list),
		"types": model.DecorationTypeMeta(),
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// one - 装扮详情
func (this *Decoration) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Decoration{}

	for key, val := range params {
		if utils.In.Array(key, decorationAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	item, _ := facade.DB.Model(&table).Where(table).Find()
	data = facade.Comm.WithField(item, params["field"])

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all - 装扮列表（管理员查全部，普通用户仅上架）
func (this *Decoration) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "sort desc, id desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Decoration

	query := facade.DB.Model(&result)
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}

	if this.meta.permit(ctx) {
		if !utils.Is.Empty(params["status"]) {
			query = query.Where("status", params["status"])
		}
	} else {
		query = query.Where("status", model.DecorationStatusOn)
	}

	if typ := cast.ToString(params["type"]); !utils.Is.Empty(typ) {
		query = query.Where("type", typ)
	}
	// 价格筛选：free 只看免费 / paid 只看付费（= 非免费），空=全部
	query = model.FilterDecorationPrice(query, cast.ToString(params["price_type"]))
	if keyword := cast.ToString(params["keyword"]); !utils.Is.Empty(keyword) {
		query = query.Like("name", "%"+keyword+"%")
	}

	count, _ := query.Count()
	items, _ := query.Order(this.decorationOrder(cast.ToString(params["order"]))).Limit(limit).Page(page).Select()
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

// count - 装扮统计（管理员）
func (this *Decoration) count(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}
	this.json(ctx, model.DecorationCount(), facade.Lang(ctx, "查询成功！"), 200)
}

// mine - 我的装扮（已拥有 + 系统默认），按类型分组
func (this *Decoration) mine(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	this.json(ctx, model.MyDecorations(user.Id), facade.Lang(ctx, "查询成功！"), 200)
}

// wearing - 我当前佩戴的装扮
func (this *Decoration) wearing(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	this.json(ctx, model.WearingDecorations(user.Id), facade.Lang(ctx, "查询成功！"), 200)
}

// userDecorations - 查看指定用户拥有的装扮（管理员）
//
// 与 mine 的区别：mine 只能看自己；这里按 uid 查任意用户，用于后台排查
// 「这个用户有哪些装扮 / 为什么佩戴的头衔没生效 / 装扮是否已过期」。
// 返回结构与 mine 一致：{ user, types, wearing, list }。
func (this *Decoration) userDecorations(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	uid := cast.ToInt(this.params(ctx)["uid"])
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	// 只回传必要字段：避免把 password 等敏感信息一起带给前端
	info := facade.H{}
	if user, _ := facade.DB.Model(&model.Users{}).Where("id", uid).Find(); !utils.Is.Empty(user) {
		for _, key := range []string{"id", "account", "nickname", "avatar"} {
			info[key] = user[key]
		}
	}
	if len(info) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	data := model.MyDecorations(uid)
	data["user"] = info

	this.json(ctx, data, facade.Lang(ctx, "查询成功！"), 200)
}

// ============================== 佩戴 ==============================

// wear - 佩戴装扮
//
// text: 纯样式类头衔允许用户自填文字（装扮自带文字时忽略）
func (this *Decoration) wear(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	decorationId := cast.ToInt(params["decoration_id"])
	if decorationId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "decoration_id"), 400)
		return
	}

	wearing, err := model.WearDecoration(user.Id, decorationId, cast.ToString(params["text"]))
	if err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"wearing": wearing}, facade.Lang(ctx, "佩戴成功！"), 200)
}

// unwear - 卸下某类型的装扮
func (this *Decoration) unwear(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	typ := cast.ToString(params["type"])
	if utils.Is.Empty(typ) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "type"), 400)
		return
	}

	if err := model.UnwearDecoration(user.Id, typ); err != nil {
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, gin.H{"wearing": model.WearingDecorations(user.Id)}, facade.Lang(ctx, "已卸下！"), 200)
}

// ============================== 管理端 ==============================

// save - 保存装扮（管理员）
func (this *Decoration) save(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
		return
	}
	this.update(ctx)
}

// create - 新增装扮（管理员）
func (this *Decoration) create(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	now := time.Now().Unix()
	table := model.Decoration{CreateTime: now, UpdateTime: now}

	for key, val := range params {
		if utils.In.Array(key, decorationAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	if utils.Is.Empty(table.Type) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "type"), 400)
		return
	}
	if !model.DecorationTypeSupported(table.Type) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的装扮类型：%s", table.Type), 400)
		return
	}
	if utils.Is.Empty(table.Name) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "name"), 400)
		return
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update - 更新装扮（管理员）
func (this *Decoration) update(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	async := utils.Async[map[string]any]()
	for key, val := range params {
		if utils.In.Array(key, decorationAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	table := model.Decoration{}
	if _, err := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result()); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": params["id"]}, facade.Lang(ctx, "更新成功！"), 200)
}

// buy - 兑换装扮（登录用户）
//
// 装扮商城的独立下单入口：只扣用户积分并发放装扮，不经过积分商城的商品 / 订单体系
// （没有 goods / goods_order 记录）。兑换痕迹见积分流水与「我的装扮」。
func (this *Decoration) buy(ctx *gin.Context) {
	user := this.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)
	decorationId := cast.ToInt(params["decoration_id"])
	if decorationId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "decoration_id"), 400)
		return
	}

	result, err := model.BuyDecoration(user.Id, decorationId)
	if err != nil {
		// 业务性拒绝（积分不足 / 已售罄 / 已达限购 / 已下架）用 202，由前端直接提示
		this.json(ctx, nil, err.Error(), 202)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "兑换成功！"), 200)
}

// grant - 发放装扮（管理员，支持批量用户 / 批量装扮）
func (this *Decoration) grant(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	uids := utils.Unity.Ids(params["uids"])
	decorationIds := utils.Unity.Ids(params["decoration_ids"])

	if utils.Is.Empty(uids) || utils.Is.Empty(decorationIds) {
		this.json(ctx, nil, facade.Lang(ctx, "uids 与 decoration_ids 不能为空！"), 400)
		return
	}

	duration := cast.ToInt64(params["duration"])
	source := cast.ToString(params["source"])
	if utils.Is.Empty(source) {
		source = model.DecorationSourceAdmin
	}

	success, failed := 0, 0
	for _, uid := range uids {
		for _, decorationId := range decorationIds {
			if err := model.GrantDecoration(cast.ToInt(uid), cast.ToInt(decorationId), source, duration); err != nil {
				failed++
				continue
			}
			success++
		}
	}

	this.json(ctx, gin.H{"success": success, "failed": failed}, facade.Lang(ctx, "发放完成！"), 200)
}

// revoke - 回收装扮（管理员，支持批量用户 / 批量装扮）
func (this *Decoration) revoke(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	uids := utils.Unity.Ids(params["uids"])
	decorationIds := utils.Unity.Ids(params["decoration_ids"])

	if utils.Is.Empty(uids) || utils.Is.Empty(decorationIds) {
		this.json(ctx, nil, facade.Lang(ctx, "uids 与 decoration_ids 不能为空！"), 400)
		return
	}

	success, failed := 0, 0
	for _, uid := range uids {
		for _, decorationId := range decorationIds {
			if err := model.RevokeDecoration(cast.ToInt(uid), cast.ToInt(decorationId)); err != nil {
				failed++
				continue
			}
			success++
		}
	}

	this.json(ctx, gin.H{"success": success, "failed": failed}, facade.Lang(ctx, "回收完成！"), 200)
}

// remove - 软删除装扮（管理员）
func (this *Decoration) remove(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Decoration{})
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

	// 关联商品一并下架，避免商城出现「点进去发现装扮已删」的脏数据
	_ = facade.DB.Drive().Model(&model.Goods{}).Where("decoration_id IN ?", ids).
		UpdateColumns(map[string]any{"status": model.GoodsStatusOff}).Error

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// delete - 彻底删除装扮（管理员）
func (this *Decoration) delete(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Decoration{}).WithTrashed()
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

	// 用户已拥有的记录一并清理（含正在佩戴的），避免装扮无处引用
	_ = facade.DB.Drive().Unscoped().Where("decoration_id IN ?", ids).Delete(&model.UserDecoration{}).Error
	_ = facade.DB.Drive().Model(&model.Goods{}).Where("decoration_id IN ?", ids).
		UpdateColumns(map[string]any{"status": model.GoodsStatusOff}).Error

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// restore - 恢复装扮（管理员）
func (this *Decoration) restore(ctx *gin.Context) {
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

	query := facade.DB.Model(&model.Decoration{}).OnlyTrashed().WhereIn("id", ids)
	columnData, _ := query.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := facade.DB.Model(&model.Decoration{}).OnlyTrashed().Restore(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// clear - 清空装扮回收站（管理员）
func (this *Decoration) clear(ctx *gin.Context) {
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	query := facade.DB.Model(&model.Decoration{}).OnlyTrashed()
	columnData, _ := query.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := query.Force().Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}
