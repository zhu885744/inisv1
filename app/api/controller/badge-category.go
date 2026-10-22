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

// 勋章分类
//
// 后台：分类 CRUD（创建 / 改名 / 换图标配色 / 排序 / 停用 / 删除），
// 以及把勋章批量挪到某个分类（move）。
//
// 前台不需要单独接口：`badge/mine` 已经按分类把勋章分组返回（见 controller/badge.go 的 mine），
// 少一次请求也少一处口径不一致。

const badgeCategoryAllowFields = "name,description,icon,color,cover,sort,status,json,text"

var badgeCategoryAllowFieldsSlice = []any{
	"name", "description", "icon", "color", "cover", "sort", "status", "json", "text",
}

type BadgeCategory struct {
	base
}

func (this *BadgeCategory) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *BadgeCategory) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *BadgeCategory) processFieldValue(val any) any {
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

// delCache - 分类变化后清掉勋章相关缓存（勋章列表 / 进度都带 badge 标签）
func (this *BadgeCategory) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "badge", "badge-category"})
}

func (this *BadgeCategory) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"count":  this.count,
		"column": this.column,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *BadgeCategory) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":   this.save,
		"create": this.create,
		// move：把一批勋章挪到指定分类（后台分类页的批量操作）
		"move": this.move,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *BadgeCategory) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

func (this *BadgeCategory) IDEL(ctx *gin.Context) {
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

func (this *BadgeCategory) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// one 单个分类
func (this *BadgeCategory) one(ctx *gin.Context) {
	params := this.params(ctx)
	table := model.BadgeCategory{}

	if !utils.Is.Empty(params["id"]) {
		table.Id = cast.ToInt(params["id"])
	}

	query := this.withTrashOptions(facade.DB.Model(&table), params)
	query = this.buildQuery(query, params)
	item, _ := query.Where(table).Find()

	data := facade.Comm.WithField(item, params["field"])
	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// all 分类列表（后台）
//
// 除列表外一并返回 counts（category_id → 勋章数量，key=0 是未分类），
// 后台侧栏要显示「成长勋章 3」这类计数，省一次聚合请求。
func (this *BadgeCategory) all(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "sort desc, id asc",
	})

	table := model.BadgeCategory{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.BadgeCategory

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	count, _ := query.Where(table).Count()

	item, _ := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
	data := utils.ArrayMapWithField(item, params["field"])

	code := 200
	msg := facade.Lang(ctx, "数据请求成功！")
	if utils.Is.Empty(data) {
		code = 204
		msg = facade.Lang(ctx, "无数据！")
	}

	this.json(ctx, gin.H{
		"data":   data,
		"count":  count,
		"page":   math.Ceil(float64(count) / float64(limit)),
		"counts": model.BadgeCategoryCounts(),
	}, msg, code)
}

func (this *BadgeCategory) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.BadgeCategory{}), params)
	query = this.buildQuery(query, params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *BadgeCategory) column(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.BadgeCategory{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *BadgeCategory) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// checkName - 分类名必填 + 不重名（重名会让后台侧栏和前台分组彻底分不清）
func (this *BadgeCategory) checkName(ctx *gin.Context, params map[string]any) bool {

	name := strings.TrimSpace(cast.ToString(params["name"]))
	if utils.Is.Empty(name) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "name"), 400)
		return false
	}

	query := facade.DB.Model(&model.BadgeCategory{}).Where("name", name)
	if id := cast.ToInt(params["id"]); id > 0 {
		query = query.Where("id", "!=", id)
	}
	if exist, _ := query.WithTrashed().Exist(); exist {
		this.json(ctx, nil, facade.Lang(ctx, "分类名称已存在！"), 400)
		return false
	}

	return true
}

func (this *BadgeCategory) create(ctx *gin.Context) {
	params := this.params(ctx)

	if !this.checkName(ctx, params) {
		return
	}

	now := time.Now().Unix()
	table := model.BadgeCategory{CreateTime: now, UpdateTime: now, Status: 1, Icon: "bi bi-collection"}

	for key, val := range params {
		if utils.In.Array(key, badgeCategoryAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *BadgeCategory) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	// 只改排序 / 状态时不校验名称，避免后台拖一下顺序就被「名称不能为空」拦住
	if _, exist := params["name"]; exist {
		if !this.checkName(ctx, params) {
			return
		}
	}

	table := model.BadgeCategory{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, badgeCategoryAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	if _, err := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result()); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// move 批量把勋章挪到某分类（category_id = 0 表示挪回「未分类」）
func (this *BadgeCategory) move(ctx *gin.Context) {
	params := this.params(ctx)

	ids := utils.Unity.Ids(params["ids"])
	target := cast.ToInt(params["category_id"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if target > 0 {
		exist, _ := facade.DB.Model(&model.BadgeCategory{}).Where("id", target).Exist()
		if !exist {
			this.json(ctx, nil, facade.Lang(ctx, "目标分类不存在！"), 400)
			return
		}
	}

	tx := facade.DB.Drive().Model(&model.Badge{}).
		Where("id IN ?", ids).
		UpdateColumn("category_id", target)
	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "移动失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids, "category_id": target}, facade.Lang(ctx, "移动成功！"), 200)
}

// remove 软删除分类
//
// 分类下还有勋章时默认拒绝，避免「删个分类顺手把勋章弄丢了」；
// 前端确认后会带 force=1 再来一次，此时把勋章退化为未分类（不删勋章、不删持有记录）。
func (this *BadgeCategory) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.BadgeCategory{})
	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	used, _ := facade.DB.Model(&model.Badge{}).WhereIn("category_id", ids).Count()
	if used > 0 && !cast.ToBool(params["force"]) {
		this.json(ctx, gin.H{"badges": used},
			facade.Lang(ctx, "该分类下还有 %d 枚勋章，删除后它们会变成「未分类」", used), 400)
		return
	}

	if used > 0 {
		facade.DB.Drive().Model(&model.Badge{}).
			Where("category_id IN ?", ids).
			UpdateColumn("category_id", 0)
	}

	if _, err := item.Delete(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids, "badges": used}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *BadgeCategory) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.BadgeCategory{}).WithTrashed()
	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := item.Force().Delete(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *BadgeCategory) clear(ctx *gin.Context) {
	item := facade.DB.Model(&model.BadgeCategory{}).OnlyTrashed()

	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := item.Force().Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *BadgeCategory) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.BadgeCategory{}).OnlyTrashed().WhereIn("id", ids)
	ids = utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := facade.DB.Model(&model.BadgeCategory{}).OnlyTrashed().Restore(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}
