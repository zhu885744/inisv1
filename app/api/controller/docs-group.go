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

type DocsGroup struct {
	base
}

var docsGroupAllowFields = []string{
	"pid", "version", "slug", "name", "description", "icon",
	"sort", "status", "visible", "role_ids", "json", "text",
}

func (this *DocsGroup) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":      this.one,
		"all":      this.all,
		"tree":     this.tree,
		"count":    this.count,
		"versions": this.versions,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *DocsGroup) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":         this.save,
		"create":       this.create,
		"update":       this.update,
		"remove":       this.remove,
		"restore":      this.restore,
		"delete":       this.delete,
		"sort":         this.sort,
		"move":         this.move,
		"copy-version": this.copyVersion,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *DocsGroup) IPUT(ctx *gin.Context) {
	this.IPOST(ctx)
}

func (this *DocsGroup) IDEL(ctx *gin.Context) {
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

func (this *DocsGroup) INDEX(ctx *gin.Context) {
	this.json(ctx, gin.H{"route": "/api/docs-group/:method"}, facade.Lang(ctx, "文档目录接口"), 200)
}

/* ------------------------------ 查询 ------------------------------ */

func (this *DocsGroup) one(ctx *gin.Context) {
	params := this.params(ctx)

	var item model.DocsGroup
	query := facade.DB.Model(&model.DocsGroup{})

	if !utils.Is.Empty(params["id"]) {
		query = query.Where("id", params["id"])
	}
	if !utils.Is.Empty(params["slug"]) {
		query = query.Where("slug", params["slug"])
	}
	if !utils.Is.Empty(params["version"]) {
		query = query.Where("version", params["version"])
	}

	query.Scan(&item)

	if item.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, item, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *DocsGroup) all(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})
	limit := this.meta.limit(ctx)

	query := facade.DB.Model(&[]model.DocsGroup{}).
		Where("version", params["version"]).
		Order(utils.Ternary[string](utils.Is.Empty(params["order"]), "sort asc, id asc", cast.ToString(params["order"]))).
		Limit(limit).
		Page(this.pageValue(params))

	if !utils.Is.Empty(params["pid"]) {
		query = query.Where("pid", params["pid"])
	}
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

type docsGroupNode struct {
	model.DocsGroup
	Children []*docsGroupNode `json:"children"`
	Docs     []model.Docs     `json:"docs"`
}

func (this *DocsGroup) tree(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})
	version := cast.ToString(params["version"])

	var groups []model.DocsGroup
	query := facade.DB.Model(&groups).
		Where("version", version).
		Order("sort asc, id asc")

	// 前台只展示启用中的目录；后台传 admin=1 时不过滤
	if !cast.ToBool(params["admin"]) {
		query = query.Where("status", 1)
	}
	query.Select()

	// 一次查询后内存建树，避免逐层递归查库
	nodes := make(map[int]*docsGroupNode, len(groups))
	for i := range groups {
		nodes[groups[i].Id] = &docsGroupNode{
			DocsGroup: groups[i],
			Children:  make([]*docsGroupNode, 0),
			Docs:      make([]model.Docs, 0),
		}
	}

	roots := make([]*docsGroupNode, 0, len(groups))
	for i := range groups {
		node := nodes[groups[i].Id]
		if groups[i].Pid != 0 && groups[i].Pid != groups[i].Id {
			if parent, ok := nodes[groups[i].Pid]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	// with_docs=1：把页面挂到所属目录上，左侧树一次请求即可渲染完整结构
	if cast.ToBool(params["with_docs"]) {
		var docs []model.Docs
		query := facade.DB.Model(&docs).Where("version", version).Order("sort asc, id asc")
		if !cast.ToBool(params["admin"]) {
			query = query.Where("status", 1)
		}
		query.Select()

		rootDocs := make([]model.Docs, 0)

		for i := range docs {
			if node, ok := nodes[docs[i].Gid]; ok {
				node.Docs = append(node.Docs, docs[i])
				continue
			}
			// 没有目录（或目录已删）的文档挂到虚拟根节点，保证「一级根」也能显示
			if docs[i].Gid == 0 {
				rootDocs = append(rootDocs, docs[i])
			}
		}

		if len(rootDocs) > 0 {
			roots = append([]*docsGroupNode{{
				DocsGroup: model.DocsGroup{Id: 0, Version: version, Name: "未分类", Status: 1, Visible: 1},
				Children:  make([]*docsGroupNode, 0),
				Docs:      rootDocs,
			}}, roots...)
		}
	}

	this.json(ctx, roots, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *DocsGroup) count(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})

	query := facade.DB.Model(&model.DocsGroup{}).Where("version", params["version"])
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *DocsGroup) versions(ctx *gin.Context) {
	rows, _ := facade.DB.Model(&model.DocsGroup{}).Column("version")
	list := cast.ToStringSlice(rows)

	// 去重 + 保证 latest 存在
	seen := make(map[string]bool, len(list)+1)
	result := make([]string, 0, len(list)+1)
	for _, item := range append([]string{"latest"}, list...) {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}

	this.json(ctx, result, facade.Lang(ctx, "数据请求成功！"), 200)
}

/* ------------------------------ 写入 ------------------------------ */

func (this *DocsGroup) save(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
		return
	}
	this.update(ctx)
}

func (this *DocsGroup) create(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest", "status": 1, "visible": 1})

	if err := validator.NewValid("docs-group", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// pid 必须指向已存在的目录
	if pid := cast.ToInt(params["pid"]); pid != 0 {
		if exist, _ := facade.DB.Model(&model.DocsGroup{}).Where("id", pid).Exist(); !exist {
			this.json(ctx, nil, facade.Lang(ctx, "父级目录不存在！"), 400)
			return
		}
	}

	table := model.DocsGroup{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.InArray(key, docsGroupAllowFields) {
			utils.Struct.Set(&table, key, this.processValue(val))
		}
	}

	// 未指定排序时排到同级末尾
	if table.Sort == 0 {
		table.Sort = this.nextSort(table.Version, table.Pid)
	}

	// slug 唯一性：同一项目下、同一父级内不能重复
	if !utils.Is.Empty(table.Slug) {
		exist, _ := facade.DB.Model(&model.DocsGroup{}).
			WithTrashed().
			Where("version", table.Version).
			Where("pid", table.Pid).
			Where("slug", table.Slug).
			Exist()

		if exist {
			this.json(ctx, nil, facade.Lang(ctx, "目录标识 %s 已存在！", table.Slug), 400)
			return
		}
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *DocsGroup) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	if err := validator.NewValid("docs-group", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	id := cast.ToInt(params["id"])

	var old model.DocsGroup
	facade.DB.Model(&model.DocsGroup{}).Where("id", id).Scan(&old)
	if old.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "目录不存在！"), 404)
		return
	}

	// 改父级时不能移到自己或自己的子目录下（否则树会成环）
	if !utils.Is.Empty(params["pid"]) {
		pid := cast.ToInt(params["pid"])
		if pid != 0 && (pid == id || this.isDescendant(id, pid)) {
			this.json(ctx, nil, facade.Lang(ctx, "不能移动到自身或子目录下！"), 400)
			return
		}
	}

	data := make(map[string]any, len(docsGroupAllowFields))
	for key, val := range params {
		if utils.InArray(key, docsGroupAllowFields) {
			data[key] = this.processValue(val)
		}
	}
	data["update_time"] = time.Now().Unix()

	// slug 唯一性：自己改自己的时候要排除自身。
	//
	// 这里不能用模型的 AfterSave 校验 —— 那条历史上带了 `id != this.Id` 排队，
	// 但 update 是用 map 提交的，而 `id` 不在 docsGroupAllowFields 白名单里，
	// 导致 map 里没有 id、this.Id 为 0，自己反而撞了自己，报「目录标识已存在」。
	if !utils.Is.Empty(data["slug"]) {
		targetVersion := cast.ToString(params["version"])
		if targetVersion == "" {
			targetVersion = old.Version
		}
		targetPid := cast.ToInt(params["pid"])

		exist, _ := facade.DB.Model(&model.DocsGroup{}).
			WithTrashed().
			Where("id", "!=", id).
			Where("version", targetVersion).
			Where("pid", targetPid).
			Where("slug", data["slug"]).
			Exist()

		if exist {
			this.json(ctx, nil, facade.Lang(ctx, "目录标识 %s 已存在！", cast.ToString(data["slug"])), 400)
			return
		}
	}

	if _, err := facade.DB.Model(&model.DocsGroup{}).Where("id", id).Update(data); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *DocsGroup) sort(ctx *gin.Context) {
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
		if _, err := facade.DB.Model(&model.DocsGroup{}).Where("id", id).Update(map[string]any{
			"sort":        cast.ToInt(item["sort"]),
			"update_time": time.Now().Unix(),
		}); err == nil {
			updated++
		}
	}

	this.json(ctx, gin.H{"updated": updated}, facade.Lang(ctx, "排序更新成功！"), 200)
}

func (this *DocsGroup) move(ctx *gin.Context) {
	params := this.params(ctx)

	id := cast.ToInt(params["id"])
	if id <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	pid := cast.ToInt(params["pid"])
	if pid != 0 && (pid == id || this.isDescendant(id, pid)) {
		this.json(ctx, nil, facade.Lang(ctx, "不能移动到自身或子目录下！"), 400)
		return
	}

	data := map[string]any{"pid": pid, "update_time": time.Now().Unix()}
	if !utils.Is.Empty(params["sort"]) {
		data["sort"] = cast.ToInt(params["sort"])
	}

	if _, err := facade.DB.Model(&model.DocsGroup{}).Where("id", id).Update(data); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": id, "pid": pid}, facade.Lang(ctx, "移动成功！"), 200)
}

func (this *DocsGroup) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 有子目录或页面时拒绝删除，避免出现孤儿数据
	childCount, _ := facade.DB.Model(&model.DocsGroup{}).Where("pid", "IN", ids).Count()
	docsCount, _ := facade.DB.Model(&model.Docs{}).Where("gid", "IN", ids).Count()

	if childCount > 0 || docsCount > 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该目录下还有子目录或文档，请先清空！"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.DocsGroup{}).Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *DocsGroup) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.DocsGroup{}).Force().Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "彻底删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "彻底删除成功！"), 200)
}

func (this *DocsGroup) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.DocsGroup{}).OnlyTrashed().Where("id", "IN", ids).Update(map[string]any{"delete_time": 0}); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

/* ------------------------------ 工具 ------------------------------ */

// pageValue - 取页码（缺省 1）
func (this *DocsGroup) pageValue(params map[string]any) int {
	page := cast.ToInt(params["page"])
	if page <= 0 {
		return 1
	}
	return page
}

// processValue - 把 map / slice 统一转成可入库的字符串
func (this *DocsGroup) processValue(val any) any {
	switch utils.Get.Type(val) {
	case "map", "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

// nextSort - 同级目录的下一个排序值
func (this *DocsGroup) nextSort(version string, pid int) int {
	var item model.DocsGroup
	facade.DB.Model(&model.DocsGroup{}).
		Where("version", version).
		Where("pid", pid).
		Order("sort desc").
		Scan(&item)

	if item.Id == 0 {
		return 1
	}
	return item.Sort + 1
}

// isDescendant - 判断 childID 是否在 parentID 的子树中（防止移动成环）
func (this *DocsGroup) isDescendant(parentID, childID int) bool {
	if parentID <= 0 || childID <= 0 {
		return false
	}

	visited := make(map[int]bool)
	current := childID

	for depth := 0; depth < 64 && current > 0; depth++ {
		if visited[current] {
			return true
		}
		visited[current] = true

		var item model.DocsGroup
		facade.DB.Model(&model.DocsGroup{}).Where("id", current).Scan(&item)
		if item.Id == 0 {
			return false
		}
		if item.Pid == parentID {
			return true
		}
		current = item.Pid
	}

	return false
}

// copyVersion - 把某个版本的目录与文档整体复制到新版本
func (this *DocsGroup) copyVersion(ctx *gin.Context) {
	params := this.params(ctx)
	from := cast.ToString(params["from"])
	to := cast.ToString(params["to"])

	if from == "" || to == "" || from == to {
		this.json(ctx, nil, facade.Lang(ctx, "版本参数不合法！"), 400)
		return
	}

	if exist, _ := facade.DB.Model(&model.DocsGroup{}).Where("version", to).Exist(); exist {
		this.json(ctx, nil, facade.Lang(ctx, "目标版本已存在！"), 400)
		return
	}

	var groups []model.DocsGroup
	facade.DB.Model(&groups).Where("version", from).Order("sort asc, id asc").Select()

	// 旧 id → 新 id，便于把文档挂到复制后的目录
	idMap := make(map[int]int, len(groups))
	for i := range groups {
		source := groups[i]
		item := model.DocsGroup{
			Pid:         0,
			Version:     to,
			Slug:        source.Slug,
			Name:        source.Name,
			Description: source.Description,
			Icon:        source.Icon,
			Sort:        source.Sort,
			Status:      source.Status,
			Visible:     source.Visible,
			RoleIds:     source.RoleIds,
		}
		if _, err := facade.DB.Model(&item).Create(&item); err == nil {
			idMap[source.Id] = item.Id
		}
	}

	// 第二遍修正父子关系（复制过程中父 id 已拿到）
	for i := range groups {
		if newId, ok := idMap[groups[i].Id]; ok && groups[i].Pid != 0 {
			if newPid, ok := idMap[groups[i].Pid]; ok {
				facade.DB.Model(&model.DocsGroup{}).Where("id", newId).Update(map[string]any{"pid": newPid})
			}
		}
	}

	var docs []model.Docs
	facade.DB.Model(&docs).Where("version", from).Select()

	copied := 0
	for i := range docs {
		source := docs[i]
		gid := 0
		if v, ok := idMap[source.Gid]; ok {
			gid = v
		}
		item := model.Docs{
			Gid:            gid,
			Uid:            source.Uid,
			Version:        to,
			Slug:           source.Slug,
			Title:          source.Title,
			Content:        source.Content,
			Sort:           source.Sort,
			Status:         source.Status,
			Visible:        source.Visible,
			RoleIds:        source.RoleIds,
			SeoTitle:       source.SeoTitle,
			SeoKeywords:    source.SeoKeywords,
			SeoDescription: source.SeoDescription,
			PublishTime:    source.PublishTime,
		}
		if _, err := facade.DB.Model(&item).Create(&item); err == nil {
			copied++
		}
	}

	this.json(ctx, gin.H{"groups": len(idMap), "docs": copied}, facade.Lang(ctx, "版本复制成功！"), 200)
}
