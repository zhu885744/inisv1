package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 勋章
//
// 两类使用场景：
//   - 后台：勋章定义的增删改查（save/create/update/remove/delete/restore/clear），
//     以及给用户「手动发放 / 收回」（grant/revoke）、查看谁获得了（records）；
//   - 前台：我的勋章进度（mine，登录）、某用户已获得的勋章（held，公开）。
//
// 自动发放（发布文章、注册满 N 天等）不在这里触发：条件判定见 model/badge.go
// 的 BadgeAutoGrant（发布动作后异步调用）与 app/timer/badge.go（每小时兜底扫描）。

const badgeAllowFields = "name,description,icon,image,color,condition,condition_value,category_id,rarity,start_time,end_time,expire_days,sort,status,remark,json,text"

var badgeAllowFieldsSlice = []any{
	"name", "description", "icon", "image", "color",
	"condition", "condition_value", "category_id",
	"rarity", "start_time", "end_time", "expire_days",
	"sort", "status", "remark", "json", "text",
}

type Badge struct {
	base
}

func (this *Badge) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Badge) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Badge) processFieldValue(val any) any {
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

func (this *Badge) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "badge"})
}

// IGET - GET请求本体
func (this *Badge) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":        this.one,
		"all":        this.all,
		"count":      this.count,
		"column":     this.column,
		"conditions": this.conditions,
		"records":    this.records,
		"mine":       this.mine,
		"held":       this.held,
		"wall":       this.wall,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Badge) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":   this.save,
		"create": this.create,
		"scan":   this.scan,
		"wear":   this.wear,
		"unwear": this.unwear,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

// IPUT - PUT请求本体
func (this *Badge) IPUT(ctx *gin.Context) {
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

// IDEL - DELETE请求本体
func (this *Badge) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove": this.remove,
		"delete": this.delete,
		"clear":  this.clear,
		// 收回勋章（也可以走 PUT revoke，两者等价）
		"revoke": this.revoke,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	go this.delCache()
}

// INDEX - 根路径
func (this *Badge) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// one 获取指定勋章
func (this *Badge) one(ctx *gin.Context) {
	params := this.params(ctx)
	table := model.Badge{}

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

// all 勋章列表（后台）
func (this *Badge) all(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "sort desc, id asc",
	})

	table := model.Badge{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Badge

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
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, msg, code)
}

// count 统计
func (this *Badge) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.buildQuery(facade.DB.Model(&model.Badge{}), params)
	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

// column 单列查询
func (this *Badge) column(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Badge{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// conditions 条件清单（后台勋章表单的下拉）
func (this *Badge) conditions(ctx *gin.Context) {
	this.json(ctx, model.BadgeConditionList, facade.Lang(ctx, "数据请求成功！"), 200)
}

// checkCategory - 分类必须存在（0 表示未分类，允许）
func (this *Badge) checkCategory(ctx *gin.Context, params map[string]any) bool {
	id := cast.ToInt(params["category_id"])
	if id <= 0 {
		return true
	}
	exist, _ := facade.DB.Model(&model.BadgeCategory{}).Where("id", id).Exist()
	if !exist {
		this.json(ctx, nil, facade.Lang(ctx, "分类不存在！"), 400)
		return false
	}
	return true
}

// save 保存（有 id 走更新）
func (this *Badge) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create 创建勋章
func (this *Badge) create(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "name"), 400)
		return
	}

	if err := validator.NewValid("badge", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	condition := cast.ToString(params["condition"])
	if !model.BadgeConditionKeyValid(condition) {
		this.json(ctx, nil, facade.Lang(ctx, "条件类型不合法！"), 400)
		return
	}

	if !this.checkCategory(ctx, params) {
		return
	}

	if !model.BadgeRarityKeyValid(cast.ToString(params["rarity"])) {
		this.json(ctx, nil, facade.Lang(ctx, "稀有度不合法！"), 400)
		return
	}

	now := time.Now().Unix()
	table := model.Badge{CreateTime: now, UpdateTime: now, Status: 1}

	for key, val := range params {
		if utils.In.Array(key, badgeAllowFieldsSlice) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	// 手动勋章不需要阈值
	if table.Condition == model.BadgeConditionManual {
		table.ConditionValue = 0
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新勋章
func (this *Badge) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	if err := validator.NewValid("badge", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if condition, exist := params["condition"]; exist && !model.BadgeConditionKeyValid(cast.ToString(condition)) {
		this.json(ctx, nil, facade.Lang(ctx, "条件类型不合法！"), 400)
		return
	}

	if _, exist := params["category_id"]; exist && !this.checkCategory(ctx, params) {
		return
	}

	if rarity, exist := params["rarity"]; exist && !model.BadgeRarityKeyValid(cast.ToString(rarity)) {
		this.json(ctx, nil, facade.Lang(ctx, "稀有度不合法！"), 400)
		return
	}

	table := model.Badge{}
	async := utils.Async[map[string]any]()

	for key, val := range params {
		if utils.In.Array(key, badgeAllowFieldsSlice) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	// 改成手动勋章时清掉阈值，避免出现「手动 + 阈值」的矛盾数据
	if cast.ToString(params["condition"]) == model.BadgeConditionManual {
		async.Set("condition_value", 0)
	}

	if _, err := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result()); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// remove 软删除
func (this *Badge) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.Badge{})
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

// delete 彻底删除
func (this *Badge) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.Badge{}).WithTrashed()
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

// clear 清空回收站
func (this *Badge) clear(ctx *gin.Context) {
	query := facade.DB.Model(&model.Badge{}).OnlyTrashed()
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

// restore 恢复
func (this *Badge) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.Badge{}).OnlyTrashed().WhereIn("id", ids)
	columnData, _ := query.Column("id")
	ids = utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	if _, err := facade.DB.Model(&model.Badge{}).OnlyTrashed().Restore(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// grant 手动发放
//
// 参数：uids（用户 ID，支持 1,2,3 或数组）、badge_ids（勋章 ID）、remark
func (this *Badge) grant(ctx *gin.Context) {

	// 双重保险：权限点由中间件把关，这里再挡一次「非管理员直接调接口」
	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx)
	uids := utils.Unity.Ids(params["uids"])
	badgeIds := utils.Unity.Ids(params["badge_ids"])

	if utils.Is.Empty(uids) || utils.Is.Empty(badgeIds) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uids / badge_ids"), 400)
		return
	}

	operator := this.user(ctx).Id
	success, failed := 0, 0

	for _, uid := range uids {
		for _, badgeId := range badgeIds {
			ok, err := model.GrantUserBadge(cast.ToInt(uid), cast.ToInt(badgeId), operator, "manual", cast.ToString(params["remark"]))
			if err != nil {
				failed++
				continue
			}
			if ok {
				success++
			}
		}
	}

	this.json(ctx, gin.H{"success": success, "failed": failed}, facade.Lang(ctx, "发放完成！"), 200)
}

// revoke 收回
//
// 参数：uids、badge_ids（两者都传时按笛卡尔积收回）
func (this *Badge) revoke(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx)
	uids := utils.Unity.Ids(params["uids"])
	badgeIds := utils.Unity.Ids(params["badge_ids"])

	if utils.Is.Empty(uids) || utils.Is.Empty(badgeIds) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uids / badge_ids"), 400)
		return
	}

	success, failed := 0, 0

	for _, uid := range uids {
		for _, badgeId := range badgeIds {
			ok, err := model.RevokeUserBadge(cast.ToInt(uid), cast.ToInt(badgeId))
			if err != nil || !ok {
				failed++
				continue
			}
			success++
		}
	}

	this.json(ctx, gin.H{"success": success, "failed": failed}, facade.Lang(ctx, "收回完成！"), 200)
}

// records 持有记录（后台：谁在什么时候获得了哪枚勋章）
func (this *Badge) records(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	query := facade.DB.Model(&[]model.UserBadge{})
	if !utils.Is.Empty(params["badge_id"]) {
		query = query.Where("badge_id", params["badge_id"])
	}
	if !utils.Is.Empty(params["uid"]) {
		query = query.Where("uid", params["uid"])
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}

	count, _ := query.Count()
	items, _ := query.Limit(limit).Page(page).Order(params["order"]).Select()

	rows := make([]map[string]any, 0, len(items))
	if len(items) > 0 {
		uids := make([]int, 0, len(items))
		badgeIds := make([]int, 0, len(items))
		for _, item := range items {
			uids = append(uids, cast.ToInt(item["uid"]))
			badgeIds = append(badgeIds, cast.ToInt(item["badge_id"]))
		}

		// 批量补用户昵称 / 头像，避免逐条查库
		userIndex := make(map[int]map[string]any)
		var users []model.Users
		facade.DB.Model(&users).WhereIn("id", uids).Select()
		for _, user := range users {
			userIndex[user.Id] = map[string]any{
				"id":       user.Id,
				"nickname": user.Nickname,
				"account":  user.Account,
				"avatar":   user.Avatar,
			}
		}

		badgeIndex := model.BadgeNamesByIds(badgeIds)

		for _, item := range items {
			uid := cast.ToInt(item["uid"])
			badgeId := cast.ToInt(item["badge_id"])
			user := userIndex[uid]
			if user == nil {
				user = map[string]any{"id": uid}
			}
			rows = append(rows, map[string]any{
				"id":           cast.ToInt(item["id"]),
				"uid":          uid,
				"badge_id":     badgeId,
				"badge_name":   badgeIndex[badgeId],
				"source":       cast.ToString(item["source"]),
				"operator_uid": cast.ToInt(item["operator_uid"]),
				"remark":       cast.ToString(item["remark"]),
				"create_time":  cast.ToInt64(item["create_time"]),
				"user":         user,
			})
		}
	}

	code := 200
	msg := facade.Lang(ctx, "数据请求成功！")
	if len(rows) == 0 {
		code = 204
		msg = facade.Lang(ctx, "无数据！")
	}

	this.json(ctx, gin.H{
		"data":  rows,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, msg, code)
}

// mine 我的勋章：全部启用的勋章 + 是否获得 + 进度
func (this *Badge) mine(ctx *gin.Context) {
	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 进来先按当前数据补判一次：否则「条件早就满足、但勋章功能上线后才打开这个页面」的用户
	// 会一直看到 10/10 却没有勋章，得等整点的定时任务（或下一次发布动作）才补发。
	// 单用户判定只有几条 COUNT，代价很小；失败也不影响下面的进度展示。
	// 返回值就是「本次新获得」的勋章，前端据此播一次获得动效（只在真的新获得时非空）。
	granted := model.BadgeAutoGrant(uid)

	list := model.BadgeProgress(uid)

	newBadges := make([]gin.H, 0, len(granted))
	for _, item := range granted {
		newBadges = append(newBadges, gin.H{
			"id":          item.Id,
			"name":        item.Name,
			"description": item.Description,
			"icon":        item.Icon,
			"image":       item.Image,
			"color":       item.Color,
			"rarity":      item.Rarity,
			"rarity_name": model.BadgeRarityName(item.Rarity),
			"category_id": item.CategoryId,
		})
	}

	held := 0
	for _, item := range list {
		if cast.ToBool(item["held"]) {
			held++
		}
	}

	// 按分类分组：前台勋章墙直接按这个顺序渲染，省一次请求也避免「分类名 / 排序」两处口径不一致。
	// 空分类不下发（没有勋章的分组在用户端只是一个空壳），「未分类」永远排最后。
	totalByCat := make(map[int]int)
	heldByCat := make(map[int]int)
	for _, item := range list {
		cid := cast.ToInt(item["category_id"])
		totalByCat[cid]++
		if cast.ToBool(item["held"]) {
			heldByCat[cid]++
		}
	}

	categories := make([]gin.H, 0, len(totalByCat))
	for _, cat := range model.BadgeCategoryEnabled() {
		if totalByCat[cat.Id] == 0 {
			continue
		}
		categories = append(categories, gin.H{
			"id":          cat.Id,
			"name":        cat.Name,
			"description": cat.Description,
			"icon":        cat.Icon,
			"color":       cat.Color,
			"cover":       cat.Cover,
			"total":       totalByCat[cat.Id],
			"held":        heldByCat[cat.Id],
		})
	}
	if totalByCat[0] > 0 {
		categories = append(categories, gin.H{
			"id":          0,
			"name":        "未分类",
			"description": "还没有归类的勋章",
			"icon":        "bi bi-collection",
			"color":       "",
			"cover":       "",
			"total":       totalByCat[0],
			"held":        heldByCat[0],
		})
	}

	this.json(ctx, gin.H{
		"data":       list,
		"count":      len(list),
		"held":       held,
		"categories": categories,
		"wearing":    model.WearingBadge(uid),
		"granted":    newBadges,
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// wear 佩戴勋章（登录用户，只能佩戴自己已获得的）
func (this *Badge) wear(ctx *gin.Context) {
	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	badgeId := cast.ToInt(this.params(ctx)["badge_id"])
	if err := model.WearBadge(uid, badgeId); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"wearing": model.WearingBadge(uid)}, facade.Lang(ctx, "佩戴成功！"), 200)
}

// unwear 卸下佩戴的勋章
func (this *Badge) unwear(ctx *gin.Context) {
	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if err := model.UnwearBadge(uid); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "已卸下！"), 200)
}

// wall 勋章馆 / 佩戴榜（公开）
//
// 未登录走 5 分钟缓存（排行榜不需要秒级实时，勋章馆又是个静态味道很重的页面）；
// 登录用户带「我是否已获得」，单独计算，不共用缓存。
func (this *Badge) wall(ctx *gin.Context) {

	if uid := this.user(ctx).Id; uid > 0 {
		this.json(ctx, model.BadgeWall(uid), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	const cacheName = "badge-wall"
	if facade.Cache.Has(cacheName) {
		this.json(ctx, facade.Cache.Get(cacheName), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	data := model.BadgeWall(0)
	go facade.Cache.Set(cacheName, data, 300)
	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// held 指定用户已获得的勋章（公开，用于用户主页）
func (this *Badge) held(ctx *gin.Context) {
	params := this.params(ctx)

	uid := cast.ToInt(params["uid"])
	if uid <= 0 {
		uid = this.user(ctx).Id
	}
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	this.json(ctx, model.BadgeHeldByUser(uid), facade.Lang(ctx, "数据请求成功！"), 200)
}

// scan 手动触发一次自动判定（后台调试 / 补发历史用户）
func (this *Badge) scan(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx)
	uid := cast.ToInt(params["uid"])

	if uid > 0 {
		granted := model.BadgeAutoGrant(uid)
		names := make([]string, 0, len(granted))
		for _, item := range granted {
			names = append(names, item.Name)
		}
		this.json(ctx, gin.H{"uid": uid, "granted": names}, facade.Lang(ctx, "判定完成！"), 200)
		return
	}

	model.BadgeScan()
	this.json(ctx, nil, facade.Lang(ctx, "已触发全站扫描！"), 200)
}
