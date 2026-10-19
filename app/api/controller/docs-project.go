package controller

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
)

type DocsProject struct {
	base
}

var docsProjectAllowFields = []string{
	"name", "slug", "description", "icon", "sort",
	"status", "visible", "role_ids", "comment_enabled", "json", "text",
}

func (this *DocsProject) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":   this.one,
		"all":   this.all,
		"list":  this.list,
		"count": this.count,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *DocsProject) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":    this.save,
		"create":  this.create,
		"update":  this.update,
		"remove":  this.remove,
		"restore": this.restore,
		"delete":  this.delete,
		"sort":    this.sort,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *DocsProject) IPUT(ctx *gin.Context) {
	this.IPOST(ctx)
}

func (this *DocsProject) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove":  this.remove,
		"delete":  this.delete,
		"restore": this.restore,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *DocsProject) INDEX(ctx *gin.Context) {
	this.json(ctx, gin.H{"route": "/api/docs-project/:method"}, facade.Lang(ctx, "文档项目接口"), 200)
}

/* ------------------------------ 查询 ------------------------------ */

func (this *DocsProject) one(ctx *gin.Context) {
	params := this.params(ctx)

	var item model.DocsProject
	query := facade.DB.Model(&model.DocsProject{})

	if !utils.Is.Empty(params["id"]) {
		query = query.Where("id", params["id"])
	}
	if !utils.Is.Empty(params["slug"]) {
		query = query.Where("slug", params["slug"])
	}

	query.Scan(&item)

	if item.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, item, facade.Lang(ctx, "数据请求成功！"), 200)
}

// all - 后台项目列表（带草稿 / 禁用项）
func (this *DocsProject) all(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)

	query := facade.DB.Model(&[]model.DocsProject{}).
		Order(utils.Ternary[string](utils.Is.Empty(params["order"]), "sort asc, id asc", cast.ToString(params["order"]))).
		Limit(limit).
		Page(this.pageValue(params))

	// 回收站：只看已软删除的项目
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}

	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}
	if !utils.Is.Empty(params["keyword"]) {
		like := "%" + cast.ToString(params["keyword"]) + "%"
		query = query.Where("name", "LIKE", like)
	}

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// list - 前台项目列表：只返回启用中且当前访客可见的项目，并附带文档数
func (this *DocsProject) list(ctx *gin.Context) {
	limit := this.meta.limit(ctx)

	var items []model.DocsProject
	facade.DB.Model(&items).
		Where("status", 1).
		Order("sort asc, id asc").
		Limit(limit).
		Select()

	visible := make([]model.DocsProject, 0, len(items))
	for i := range items {
		if this.canView(ctx, &items[i]) {
			visible = append(visible, items[i])
		}
	}

	result := make([]gin.H, 0, len(visible))
	if len(visible) == 0 {
		this.json(ctx, result, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	// 项目数量很少（个位数），逐个统计比 group by 更省事，也避开了 DB 包装层的限制
	counts := make(map[string]int, len(visible))
	for i := range visible {
		count, _ := facade.DB.Model(&model.Docs{}).
			Where("version", visible[i].Slug).
			Where("status", 1).
			Count()
		counts[visible[i].Slug] = cast.ToInt(count)
	}

	for i := range visible {
		result = append(result, gin.H{
			"id":              visible[i].Id,
			"name":            visible[i].Name,
			"slug":            visible[i].Slug,
			"description":     visible[i].Description,
			"icon":            visible[i].Icon,
			"sort":            visible[i].Sort,
			"comment_enabled": visible[i].CommentEnabled,
			"docs_count":      counts[visible[i].Slug],
			"path":            "/docs/" + visible[i].Slug,
			"update_time":     visible[i].UpdateTime,
		})
	}

	this.json(ctx, result, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *DocsProject) count(ctx *gin.Context) {
	params := this.params(ctx)

	query := facade.DB.Model(&model.DocsProject{})
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "数据请求成功！"), 200)
}

/* ------------------------------ 写入 ------------------------------ */

func (this *DocsProject) save(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
		return
	}
	this.update(ctx)
}

func (this *DocsProject) create(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"status": 1, "visible": 1, "comment_enabled": 1,
	})

	if err := validator.NewValid("docs-project", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.DocsProject{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.InArray(key, docsProjectAllowFields) {
			utils.Struct.Set(&table, key, this.processValue(val))
		}
	}

	// 未指定排序时排到末尾
	if table.Sort == 0 {
		table.Sort = this.nextSort()
	}

	// 别名全局唯一：它既是 URL 里的项目名，也是 docs.version，撞了会互相覆盖。
	// 校验放在这里而不是模型的 AfterSave —— 那条历史的进去重带了 `id != this.Id`，
	// 而 update 是用 map 提交的、id 不在白名单里，会导致 this.Id 为 0、自己撞自己
	if !utils.Is.Empty(table.Slug) {
		exist, _ := facade.DB.Model(&model.DocsProject{}).
			WithTrashed().
			Where("slug", table.Slug).
			Exist()

		if exist {
			this.json(ctx, nil, facade.Lang(ctx, "项目别名 %s 已存在！", table.Slug), 400)
			return
		}
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id, "slug": table.Slug}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *DocsProject) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	if err := validator.NewValid("docs-project", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	id := cast.ToInt(params["id"])

	var old model.DocsProject
	facade.DB.Model(&model.DocsProject{}).Where("id", id).Scan(&old)
	if old.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "项目不存在！"), 404)
		return
	}

	data := make(map[string]any, len(docsProjectAllowFields))
	for key, val := range params {
		if utils.InArray(key, docsProjectAllowFields) {
			data[key] = this.processValue(val)
		}
	}
	data["update_time"] = time.Now().Unix()

	// 改了别名就同步把 docs / docs_group 里的 version 一起改，
	// 否则老链接会指向一个不存在的项目
	renamed := false
	if slug := cast.ToString(params["slug"]); slug != "" && slug != old.Slug {
		renamed = true
	}

	if renamed {
		exist, _ := facade.DB.Model(&model.DocsProject{}).
			WithTrashed().
			Where("id", "!=", id).
			Where("slug", cast.ToString(params["slug"])).
			Exist()

		if exist {
			this.json(ctx, nil, facade.Lang(ctx, "项目别名 %s 已存在！", cast.ToString(params["slug"])), 400)
			return
		}
	}

	if _, err := facade.DB.Model(&model.DocsProject{}).Where("id", id).Update(data); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if renamed {
		newSlug := cast.ToString(params["slug"])
		stamp := time.Now().Unix()

		facade.DB.Model(&model.Docs{}).Where("version", old.Slug).Update(map[string]any{
			"version": newSlug, "update_time": stamp,
		})
		facade.DB.Model(&model.DocsGroup{}).Where("version", old.Slug).Update(map[string]any{
			"version": newSlug, "update_time": stamp,
		})
		// 历史快照 / 301 记录同步改名，避免回滚与跳转失效
		facade.DB.Model(&model.DocsRedirect{}).Where("old_path", "LIKE", "/docs/"+old.Slug+"/%").Update(map[string]any{
			"old_path": "/docs/" + newSlug + "%",
		})
	}

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *DocsProject) sort(ctx *gin.Context) {
	params := this.params(ctx)
	items := cast.ToSlice(params["items"])

	if len(items) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "items"), 400)
		return
	}

	updated := 0
	for _, raw := range items {
		item := cast.ToStringMap(raw)
		id := cast.ToInt(item["id"])
		if id <= 0 {
			continue
		}
		if _, err := facade.DB.Model(&model.DocsProject{}).Where("id", id).Update(map[string]any{
			"sort":        cast.ToInt(item["sort"]),
			"update_time": time.Now().Unix(),
		}); err == nil {
			updated++
		}
	}

	this.json(ctx, gin.H{"updated": updated}, facade.Lang(ctx, "排序更新成功！"), 200)
}

func (this *DocsProject) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 还有文档或目录时拒绝删除，避免孤儿数据
	docsCount, _ := facade.DB.Model(&model.Docs{}).Where("version", "IN", this.slugsByIds(ids)).Count()
	groupCount, _ := facade.DB.Model(&model.DocsGroup{}).Where("version", "IN", this.slugsByIds(ids)).Count()

	// 注意：上面的统计走默认作用域，已在「文档回收站」里的文档不计入 ——
	// 所以项目可能带着一批软删除文档被删掉，彻底删除时要连它们一起清理，
	// 否则那些文档会指向一个不存在的 version，变成谁也看不到的孤儿数据。

	if docsCount > 0 || groupCount > 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该项目下还有文档或目录，请先清空！"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.DocsProject{}).Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *DocsProject) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	suffix := this.purgeContents(ids)

	if _, err := facade.DB.Model(&model.DocsProject{}).Force().Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "彻底删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids, "purged": suffix}, facade.Lang(ctx, "彻底删除成功！"), 200)
}

// purgeContents - 连同该项目的目录与文档一起彻底删除
//
// 项目别名就是 docs.version。如果项目下还残留在「文档回收站」里的文档
// （上面的 remove 统计走默认作用域，看不见它们），只删项目会让这些文档
// 指向一个不存在的 version，变成谁也访问不到的孤儿数据。
//
// 返回被清理的 version 列表，便于前端提示清理范围。
func (this *DocsProject) purgeContents(ids []any) []string {
	slugs := this.slugsByIds(ids)
	if len(slugs) == 0 {
		return nil
	}

	// 先取文档 id，历史与反馈要按 docs_id 清理
	columnData, _ := facade.DB.Model(&model.Docs{}).WithTrashed().Where("version", "IN", slugs).Column("id")
	docsIds := utils.Unity.Ids(columnData)

	if len(docsIds) > 0 {
		facade.DB.Model(&model.DocsHistory{}).Where("docs_id", "IN", docsIds).Force().Delete()
		facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", "IN", docsIds).Force().Delete()
	}

	facade.DB.Model(&model.Docs{}).WithTrashed().Where("version", "IN", slugs).Force().Delete()
	facade.DB.Model(&model.DocsGroup{}).WithTrashed().Where("version", "IN", slugs).Force().Delete()

	return slugs
}

func (this *DocsProject) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.DocsProject{}).OnlyTrashed().Where("id", "IN", ids).Update(map[string]any{"delete_time": 0}); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// clear - 清空项目回收站
func (this *DocsProject) clear(ctx *gin.Context) {
	columnData, _ := facade.DB.Model(&model.DocsProject{}).OnlyTrashed().Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	cleaned := this.purgeContents(ids)

	if _, err := facade.DB.Model(&model.DocsProject{}).OnlyTrashed().Where("id", "IN", ids).Force().Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids, "purged": cleaned}, facade.Lang(ctx, "回收站已清空！"), 200)
}

/* ------------------------------ 工具 ------------------------------ */

// pageValue - 取页码（缺省 1）
func (this *DocsProject) pageValue(params map[string]any) int {
	page := cast.ToInt(params["page"])
	if page <= 0 {
		return 1
	}
	return page
}

// processValue - 把 map / slice 统一转成可入库的字符串
func (this *DocsProject) processValue(val any) any {
	switch utils.Get.Type(val) {
	case "map", "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

// nextSort - 下一个项目排序值
func (this *DocsProject) nextSort() int {
	var item model.DocsProject
	facade.DB.Model(&model.DocsProject{}).Order("sort desc").Scan(&item)

	if item.Id == 0 {
		return 1
	}
	return item.Sort + 1
}

// slugsByIds - 按主键批量取别名
func (this *DocsProject) slugsByIds(ids []any) []string {
	rows, _ := facade.DB.Model(&model.DocsProject{}).Where("id", "IN", ids).Column("slug")
	return cast.ToStringSlice(rows)
}

// canView - 项目可见性：1 公开 / 2 登录 / 3 指定角色组
func (this *DocsProject) canView(ctx *gin.Context, item *model.DocsProject) bool {
	if item.Visible == 0 || item.Visible == 1 {
		return true
	}

	user, ok := ctx.Get("user")
	if !ok || user == nil {
		return false
	}

	if item.Visible == 2 {
		return true
	}

	if item.Visible == 3 {
		roleIds := cast.ToStringSlice(strings.Split(item.RoleIds, ","))
		if len(roleIds) == 0 {
			return true
		}

		userMap := cast.ToStringMap(user)
		auth := cast.ToStringMap(userMap["result"])
		group := cast.ToStringMap(auth["auth"])
		groupIds := cast.ToStringSlice(cast.ToStringMap(group["group"])["ids"])

		for _, roleId := range roleIds {
			if utils.InArray(roleId, groupIds) {
				return true
			}
		}
	}

	return false
}