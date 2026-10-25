package controller

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
)

type Docs struct {
	base
}

var docsAllowFields = []string{
	"gid", "version", "slug", "title", "content", "sort", "status",
	"visible", "role_ids", "seo_title", "seo_keywords", "seo_description", "json", "text",
}

func (this *Docs) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":            this.one,
		"all":            this.all,
		"tree":           this.tree,
		"count":          this.count,
		"search":         this.search,
		"history":        this.history,
		"feedback-stats": this.feedbackStats,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Docs) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":     this.save,
		"create":   this.create,
		"update":   this.update,
		"remove":   this.remove,
		"restore":  this.restore,
		"delete":   this.delete,
		"sort":     this.sort,
		"move":     this.move,
		"rollback": this.rollback,
		"feedback": this.feedback,
		"import":   this.importDocs,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Docs) IPUT(ctx *gin.Context) {
	this.IPOST(ctx)
}

func (this *Docs) IDEL(ctx *gin.Context) {
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

func (this *Docs) INDEX(ctx *gin.Context) {
	this.json(ctx, gin.H{"route": "/api/docs/:method"}, facade.Lang(ctx, "文档接口"), 200)
}

/* ------------------------------ 查询 ------------------------------ */

func (this *Docs) one(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})

	var item model.Docs
	query := facade.DB.Model(&model.Docs{})

	// 按 id 查时不要带 version：id 本身全局唯一，而 version 有默认值 'latest'，
	// 带上会让非 latest 项目的文档（后台编辑、前台直链）全部查不出来。
	// 按 slug 查才需要 version，因为 slug 只在项目内唯一。
	if utils.Is.Empty(params["id"]) {
		query = query.Where("version", params["version"])
	}

	if !utils.Is.Empty(params["id"]) {
		query = query.Where("id", params["id"])
	} else if !utils.Is.Empty(params["slug"]) {
		// 路径 /docs/:version/:groupSlug/:pageSlug：先按目录 slug 定位 gid，再按页面 slug 查询
		if !utils.Is.Empty(params["group"]) {
			groupSlug := cast.ToString(params["group"])

			// "-" 是「未分类 / 一级根」的占位目录（见 docs-group/tree 的虚拟根节点）
			if groupSlug == "-" || groupSlug == "root" {
				query = query.Where("gid", 0)
			} else {
				var group model.DocsGroup
				facade.DB.Model(&model.DocsGroup{}).
					Where("version", params["version"]).
					Where("slug", groupSlug).
					Scan(&group)

				if group.Id == 0 {
					this.json(ctx, nil, facade.Lang(ctx, "目录不存在！"), 404)
					return
				}
				query = query.Where("gid", group.Id)
			}
		}
		query = query.Where("slug", params["slug"])
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id 或 slug"), 400)
		return
	}

	query.Scan(&item)

	if item.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	// 后台预览：带 admin=1 可以看草稿 / 待审核 / 受限文档，但必须真的具备文档管理权限
	//
	// 以前这里只看 admin 这个参数本身，等于把未发布内容的读取权交给了任何会改 URL 的人
	// （匿名请求 /api/docs/one?id=x&admin=1 就能拿到草稿）。
	isAdminPreview := cast.ToBool(params["admin"])
	if isAdminPreview && !this.canPreview(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权预览该文档！"), 403)
		return
	}

	// 前台只返回已发布内容
	if !isAdminPreview && item.Status != 1 {
		this.json(ctx, nil, facade.Lang(ctx, "文档不存在或未发布！"), 404)
		return
	}

	// 可见性校验：受限文档未登录时返回 401，前端据此跳登录并回跳
	if !isAdminPreview && !this.canView(ctx, &item) {
		this.json(ctx, nil, facade.Lang(ctx, "请登录后查看！"), 401)
		return
	}

	// 浏览量：与文章一致，同一 IP + UA 每天只计一次（后台预览不计）
	if !isAdminPreview {
		go this.updateViews(ctx, item.Id)
	}

	this.json(ctx, this.buildDetail(item), facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *Docs) all(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})
	limit := this.meta.limit(ctx)

	// admin=1 会让列表跳过 status 过滤（含草稿 / 待审核），必须真的具备文档管理权限，
	// 否则匿名带上参数就能把未发布文档整页拉走
	if cast.ToBool(params["admin"]) && !this.canPreview(ctx) {
		params["admin"] = false
	}

	query := facade.DB.Model(&[]model.Docs{}).
		Where("version", params["version"]).
		Order(utils.Ternary[string](utils.Is.Empty(params["order"]), "sort asc, id asc", cast.ToString(params["order"]))).
		Limit(limit).
		Page(this.pageValue(params))

	// 回收站：只看已软删除的文档（删除后前台不可见，可在回收站恢复或彻底删除）
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}

	// gid=0 表示「一级根 / 未分类」，需要精确过滤（不能用 IsEmpty 判断，0 会被当成未传）
	if value, ok := params["gid"]; ok && cast.ToString(value) != "" {
		query = query.Where("gid", cast.ToInt(value))
	}
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	} else if !cast.ToBool(params["admin"]) {
		query = query.Where("status", 1)
	}

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

func (this *Docs) tree(ctx *gin.Context) {
	group := &DocsGroup{}
	group.tree(ctx)
}

func (this *Docs) count(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})

	query := facade.DB.Model(&model.Docs{}).Where("version", params["version"])
	if value, ok := params["gid"]; ok && cast.ToString(value) != "" {
		query = query.Where("gid", cast.ToInt(value))
	}
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", params["status"])
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "数据请求成功！"), 200)
}

// search - 站内文档搜索（标题 + 正文），支持版本 / 目录范围过滤
//
// 标题和正文分两次查询再按 id 合并：ModelStruct 的 OR 组合行为依赖驱动，
// 分开查最稳，文档量级（百篇）也完全够用。
func (this *Docs) search(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest"})
	keyword := strings.TrimSpace(cast.ToString(params["keyword"]))
	limit := this.meta.limit(ctx)
	page := this.pageValue(params)

	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "keyword"), 400)
		return
	}

	like := "%" + keyword + "%"

	baseQuery := func(target any) *facade.ModelStruct {
		query := facade.DB.Model(target).
			Where("version", params["version"]).
			Where("status", 1)
		if !utils.Is.Empty(params["gid"]) {
			query = query.Where("gid", params["gid"])
		}
		return query
	}

	var byTitle []model.Docs
	baseQuery(&byTitle).Where("title", "LIKE", like).Select()

	var byContent []model.Docs
	baseQuery(&byContent).Where("content", "LIKE", like).Select()

	seen := make(map[int]bool, len(byTitle)+len(byContent))
	merged := make([]model.Docs, 0, len(byTitle)+len(byContent))

	for _, list := range [][]model.Docs{byTitle, byContent} {
		for _, item := range list {
			if seen[item.Id] {
				continue
			}
			seen[item.Id] = true
			merged = append(merged, item)
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Sort != merged[j].Sort {
			return merged[i].Sort < merged[j].Sort
		}
		return merged[i].Id < merged[j].Id
	})

	if len(merged) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "没有找到相关文档！"), 204)
		return
	}

	// 内存分页，避免再次查询
	start := (page - 1) * limit
	if start > len(merged) {
		start = len(merged)
	}
	end := start + limit
	if end > len(merged) {
		end = len(merged)
	}

	result := make([]gin.H, 0, end-start)
	for _, item := range merged[start:end] {
		result = append(result, gin.H{
			"id":          item.Id,
			"gid":         item.Gid,
			"version":     item.Version,
			"slug":        item.Slug,
			"title":       item.Title,
			"group_slug":  this.groupSlug(item.Gid),
			"update_time": item.UpdateTime,
			"path":        "/docs/" + item.Version + "/" + this.groupSlug(item.Gid) + "/" + item.Slug,
		})
	}

	this.json(ctx, gin.H{"data": result, "count": len(merged)}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// history - 某篇文档的历史快照
func (this *Docs) history(ctx *gin.Context) {
	params := this.params(ctx)

	docsId := cast.ToInt(params["docs_id"])
	if docsId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "docs_id"), 400)
		return
	}

	query := facade.DB.Model(&[]model.DocsHistory{}).
		Where("docs_id", docsId).
		Order("id desc").
		Limit(this.meta.limit(ctx)).
		Page(this.pageValue(params))

	items, _ := query.Select()
	data := utils.ArrayMapWithField(items, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// feedbackStats - 后台查看某篇文档的反馈统计
func (this *Docs) feedbackStats(ctx *gin.Context) {
	params := this.params(ctx)
	docsId := cast.ToInt(params["docs_id"])

	var item model.Docs
	query := facade.DB.Model(&model.Docs{})
	if docsId > 0 {
		query = query.Where("id", docsId)
	}
	query.Scan(&item)

	if item.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "文档不存在！"), 404)
		return
	}

	this.json(ctx, gin.H{
		"docs_id": item.Id,
		"title":   item.Title,
		"useful":  item.Useful,
		"useless": item.Useless,
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

/* ------------------------------ 写入 ------------------------------ */

func (this *Docs) save(ctx *gin.Context) {
	params := this.params(ctx)

	// 前端可能漏传 id（编辑时按 create 提交），此时按 version + gid + slug
	// 找回同一条记录再走更新，避免「编辑自己的文档却提示标识已存在」。
	if cast.ToInt(params["id"]) <= 0 {
		version := cast.ToString(params["version"])
		gid := cast.ToInt(params["gid"])
		slug := cast.ToString(params["slug"])

		if slug != "" {
			var exist model.Docs
			facade.DB.Model(&model.Docs{}).
				Where("version", version).
				Where("gid", gid).
				Where("slug", slug).
				Scan(&exist)

			if exist.Id > 0 {
				params["id"] = exist.Id
			}
		}
	}

	if cast.ToInt(params["id"]) <= 0 {
		this.create(ctx)
		return
	}
	this.update(ctx)
}

func (this *Docs) create(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"version": "latest", "status": 1, "visible": 1})

	if err := validator.NewValid("docs", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 目录必须存在
	gid := cast.ToInt(params["gid"])
	if gid > 0 {
		if exist, _ := facade.DB.Model(&model.DocsGroup{}).Where("id", gid).Exist(); !exist {
			this.json(ctx, nil, facade.Lang(ctx, "所属目录不存在！"), 400)
			return
		}
	}

	// 最后一道兜底：同版本 + 同目录 + 同 slug 已存在时转为更新。
	// 前端编辑时若漏传 id（或 id 被序列化成 0），save 会按 create 进来，
	// 这里再查一次，避免「编辑自己的文档却提示标识已存在」。
	if slug := strings.TrimSpace(cast.ToString(params["slug"])); slug != "" {
		var exist model.Docs
		facade.DB.Model(&model.Docs{}).
			Where("version", cast.ToString(params["version"])).
			Where("gid", cast.ToInt(params["gid"])).
			Where("slug", slug).
			Scan(&exist)

		if exist.Id > 0 {
			params["id"] = exist.Id
			ctx.Set("params", params)
			this.update(ctx)
			return
		}
	}

	table := model.Docs{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}

	for key, val := range params {
		if utils.InArray(key, docsAllowFields) {
			utils.Struct.Set(&table, key, this.processDocsValue(val))
		}
	}

	if table.Sort == 0 {
		table.Sort = this.nextSort(table.Version, table.Gid)
	}

	if table.Status == 1 {
		table.PublishTime = time.Now().Unix()
	}

	if _, err := facade.DB.Model(&table).Create(&table); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.snapshot(&table, "创建文档")

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *Docs) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	if err := validator.NewValid("docs", params); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	id := cast.ToInt(params["id"])

	var old model.Docs
	facade.DB.Model(&model.Docs{}).Where("id", id).Scan(&old)
	if old.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "文档不存在！"), 404)
		return
	}

	data := make(map[string]any, len(docsAllowFields))
	for key, val := range params {
		if utils.InArray(key, docsAllowFields) {
			data[key] = this.processDocsValue(val)
		}
	}
	data["update_time"] = time.Now().Unix()

	// 首次发布时补发布时间
	if cast.ToInt(data["status"]) == 1 && old.PublishTime == 0 {
		data["publish_time"] = time.Now().Unix()
	}

	if _, err := facade.DB.Model(&model.Docs{}).Where("id", id).Update(data); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// slug 变更时记录 301（旧路径 → 新路径）
	if slug, ok := data["slug"]; ok && cast.ToString(slug) != old.Slug {
		this.recordRedirect(&old, cast.ToString(slug), cast.ToInt(data["gid"]))
	}

	var latest model.Docs
	facade.DB.Model(&model.Docs{}).Where("id", id).Scan(&latest)
	this.snapshot(&latest, "更新文档")

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Docs) sort(ctx *gin.Context) {
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
		if _, err := facade.DB.Model(&model.Docs{}).Where("id", id).Update(map[string]any{
			"sort":        cast.ToInt(item["sort"]),
			"update_time": time.Now().Unix(),
		}); err == nil {
			updated++
		}
	}

	this.json(ctx, gin.H{"updated": updated}, facade.Lang(ctx, "排序更新成功！"), 200)
}

func (this *Docs) move(ctx *gin.Context) {
	params := this.params(ctx)

	id := cast.ToInt(params["id"])
	if id <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	gid := cast.ToInt(params["gid"])
	if gid > 0 {
		if exist, _ := facade.DB.Model(&model.DocsGroup{}).Where("id", gid).Exist(); !exist {
			this.json(ctx, nil, facade.Lang(ctx, "目标目录不存在！"), 400)
			return
		}
	}

	data := map[string]any{"gid": gid, "update_time": time.Now().Unix()}
	if !utils.Is.Empty(params["sort"]) {
		data["sort"] = cast.ToInt(params["sort"])
	} else {
		var old model.Docs
		facade.DB.Model(&model.Docs{}).Where("id", id).Scan(&old)
		data["sort"] = this.nextSort(old.Version, gid)
	}

	if _, err := facade.DB.Model(&model.Docs{}).Where("id", id).Update(data); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": id, "gid": gid}, facade.Lang(ctx, "移动成功！"), 200)
}

func (this *Docs) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.Docs{}).Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Docs) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 彻底删除时一并清掉历史与反馈，避免孤儿数据
	facade.DB.Model(&model.DocsHistory{}).Where("docs_id", "IN", ids).Force().Delete()
	facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", "IN", ids).Force().Delete()

	if _, err := facade.DB.Model(&model.Docs{}).Force().Where("id", "IN", ids).Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "彻底删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "彻底删除成功！"), 200)
}

func (this *Docs) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.Docs{}).OnlyTrashed().Where("id", "IN", ids).Update(map[string]any{"delete_time": 0}); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// clear - 清空回收站
//
// 传了 version 就只清该项目下的，不传清全部（后台项目页会带上当前项目，
// 避免在 A 项目里点「清空」把 B 项目的回收站也清了）。
func (this *Docs) clear(ctx *gin.Context) {
	params := this.params(ctx)

	query := facade.DB.Model(&model.Docs{}).OnlyTrashed()
	if !utils.Is.Empty(params["version"]) {
		query = query.Where("version", params["version"])
	}

	columnData, _ := query.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 彻底删除时一并清掉历史与反馈，避免孤儿数据（与 delete 保持一致）
	facade.DB.Model(&model.DocsHistory{}).Where("docs_id", "IN", ids).Force().Delete()
	facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", "IN", ids).Force().Delete()

	if _, err := facade.DB.Model(&model.Docs{}).OnlyTrashed().Where("id", "IN", ids).Force().Delete(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "回收站已清空！"), 200)
}

// rollback - 回滚到指定历史快照
func (this *Docs) rollback(ctx *gin.Context) {
	params := this.params(ctx)

	historyId := cast.ToInt(params["history_id"])
	if historyId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "history_id"), 400)
		return
	}

	var history model.DocsHistory
	facade.DB.Model(&model.DocsHistory{}).Where("id", historyId).Scan(&history)
	if history.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "历史记录不存在！"), 404)
		return
	}

	_, err := facade.DB.Model(&model.Docs{}).Where("id", history.DocsId).Update(map[string]any{
		"title":       history.Title,
		"content":     history.Content,
		"update_time": time.Now().Unix(),
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	var latest model.Docs
	facade.DB.Model(&model.Docs{}).Where("id", history.DocsId).Scan(&latest)
	this.snapshot(&latest, "回滚到历史 #"+cast.ToString(historyId))

	this.json(ctx, gin.H{"id": history.DocsId}, facade.Lang(ctx, "回滚成功！"), 200)
}

// feedback - “这篇文档有帮助吗”
func (this *Docs) feedback(ctx *gin.Context) {
	params := this.params(ctx)

	docsId := cast.ToInt(params["docs_id"])
	useful := cast.ToInt(params["useful"])

	if docsId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "docs_id"), 400)
		return
	}
	if useful != 0 && useful != 1 {
		this.json(ctx, nil, facade.Lang(ctx, "参数不合法！"), 400)
		return
	}

	if exist, _ := facade.DB.Model(&model.Docs{}).Where("id", docsId).Exist(); !exist {
		this.json(ctx, nil, facade.Lang(ctx, "文档不存在！"), 404)
		return
	}

	uid := this.uid(ctx)
	ip := ctx.ClientIP()

	// 同一用户（登录按 uid / 未登录按 IP）只保留一条反馈
	query := facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", docsId)
	if uid > 0 {
		query = query.Where("uid", uid)
	} else {
		query = query.Where("uid", 0).Where("ip", ip)
	}

	var old model.DocsFeedback
	query.Scan(&old)

	if old.Id > 0 {
		if old.Useful == useful {
			this.json(ctx, gin.H{"useful": useful}, facade.Lang(ctx, "已记录！"), 200)
			return
		}
		facade.DB.Model(&model.DocsFeedback{}).Where("id", old.Id).Update(map[string]any{"useful": useful})
	} else {
		item := model.DocsFeedback{DocsId: docsId, Uid: uid, Useful: useful, Ip: ip}
		facade.DB.Model(&item).Create(&item)
	}

	this.refreshFeedbackCount(docsId)

	this.json(ctx, gin.H{"useful": useful}, facade.Lang(ctx, "感谢反馈！"), 200)
}

/* ------------------------------ 工具 ------------------------------ */

// updateViews - 浏览量 +1（同一 IP + UA 每天只计一次）
//
// 与文章（app/api/controller/article.go 的 updateViews）保持同一套口径：
// 用 IP + UA 的 md5 做访客标识，缓存 24 小时，命中即跳过。
func (this *Docs) updateViews(ctx *gin.Context, id int) {
	if id <= 0 {
		return
	}

	deviceKey := ctx.ClientIP() + ctx.Request.UserAgent()
	hash := md5.Sum([]byte(deviceKey))
	cacheKey := fmt.Sprintf("docs_views_cd:%d:%x", id, hash)

	if facade.Cache.Has(cacheKey) {
		return
	}

	if _, err := facade.DB.Model(&model.Docs{}).Where("id", id).Inc("views", 1); err != nil {
		return
	}

	facade.Cache.Set(cacheKey, true, 86400)
}

// pageValue - 取页码（缺省 1）
func (this *Docs) pageValue(params map[string]any) int {
	page := cast.ToInt(params["page"])
	if page <= 0 {
		return 1
	}
	return page
}

func (this *Docs) processDocsValue(val any) any {
	switch utils.Get.Type(val) {
	case "map", "2d slice":
		return utils.Json.Encode(val)
	case "slice":
		return strings.Join(cast.ToStringSlice(val), ",")
	}
	return val
}

// snapshot - 写历史快照
func (this *Docs) snapshot(item *model.Docs, summary string) {
	history := model.DocsHistory{
		DocsId:  item.Id,
		Uid:     item.Uid,
		Version: item.Version,
		Title:   item.Title,
		Content: item.Content,
		Summary: summary,
	}
	if _, err := facade.DB.Model(&history).Create(&history); err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "docs_id": item.Id}, "写入文档历史失败")
	}
}

// recordRedirect - 记录 slug 变更的 301
func (this *Docs) recordRedirect(old *model.Docs, newSlug string, newGid int) {
	if utils.Is.Empty(old.Slug) || utils.Is.Empty(newSlug) {
		return
	}

	oldGroup := this.groupSlug(old.Gid)
	newGroup := this.groupSlug(newGid)

	redirect := model.DocsRedirect{
		Version: old.Version,
		OldPath: "/docs/" + old.Version + "/" + oldGroup + "/" + old.Slug,
		NewPath: "/docs/" + old.Version + "/" + newGroup + "/" + newSlug,
	}
	facade.DB.Model(&redirect).Create(&redirect)
}

func (this *Docs) groupSlug(gid int) string {
	if gid <= 0 {
		return "-"
	}
	var group model.DocsGroup
	facade.DB.Model(&model.DocsGroup{}).Where("id", gid).Scan(&group)
	if utils.Is.Empty(group.Slug) {
		return cast.ToString(gid)
	}
	return group.Slug
}

func (this *Docs) nextSort(version string, gid int) int {
	var item model.Docs
	facade.DB.Model(&model.Docs{}).
		Where("version", version).
		Where("gid", gid).
		Order("sort desc").
		Scan(&item)

	if item.Id == 0 {
		return 1
	}
	return item.Sort + 1
}

// buildDetail - 详情响应：正文 + TOC + 面包屑 + 上一篇 / 下一篇
func (this *Docs) buildDetail(item model.Docs) gin.H {
	// 面包屑：从所属目录一路向上
	breadcrumb := make([]gin.H, 0, 4)
	current := item.Gid
	for depth := 0; depth < 16 && current > 0; depth++ {
		var group model.DocsGroup
		facade.DB.Model(&model.DocsGroup{}).Where("id", current).Scan(&group)
		if group.Id == 0 {
			break
		}
		breadcrumb = append([]gin.H{{"id": group.Id, "name": group.Name, "slug": group.Slug}}, breadcrumb...)
		current = group.Pid
	}

	prev, next := this.prevNext(item)

	return gin.H{
		"id":              item.Id,
		"gid":             item.Gid,
		"version":         item.Version,
		"slug":            item.Slug,
		"title":           item.Title,
		"content":         item.Content,
		"sort":            item.Sort,
		"status":          item.Status,
		"visible":         item.Visible,
		"views":           item.Views,
		"useful":          item.Useful,
		"useless":         item.Useless,
		"seo_title":       item.SeoTitle,
		"seo_keywords":    item.SeoKeywords,
		"seo_description": item.SeoDescription,
		"create_time":     item.CreateTime,
		"update_time":     item.UpdateTime,
		"publish_time":    item.PublishTime,
		"breadcrumb":      breadcrumb,
		"prev":            prev,
		"next":            next,
		"uid":             item.Uid,
		"comment":         this.docsComment(item),
		"path":            "/docs/" + item.Version + "/" + this.groupSlug(item.Gid) + "/" + item.Slug,
	}
}

// docsComment - 文档页评论开关与数量
//
// 优先级：全局 COMMENT.allow < 模块总开关 DOCS.comment < 项目 comment_enabled。
// 口径与文章一致（见 model/article.go 的 comment）：allow 控制能否发表，
// show 控制整个评论区是否展示，count 只统计审核通过的评论。
func (this *Docs) docsComment(item model.Docs) gin.H {
	project := model.DocsProjectBySlug(item.Version)

	// 项目缺失（脏数据 / 历史 version）时按允许处理，避免老内容莫名没有评论区
	allow := 1
	if project.Id > 0 && project.CommentEnabled != 1 {
		allow = 2
	}
	show := 1

	module := cast.ToStringMap(docsCommentConfig("docs", "comment"))
	if cast.ToInt(module["allow"]) == 2 {
		allow = 2
	}
	if cast.ToInt(module["show"]) == 2 {
		show = 2
	}

	// 全局 COMMENT.allow 是总闸，关掉后全部禁止
	if cast.ToInt(docsCommentConfig("comment")["allow"]) == 2 {
		allow = 2
	}

	count, _ := facade.DB.Model(&model.Comment{}).
		Where("bind_type", "docs").
		Where("bind_id", item.Id).
		Where("audit", AuditPassed).
		Count()

	return gin.H{"allow": allow, "show": show, "count": count}
}

// docsCommentConfig - 读评论相关配置，行为对齐 (*Comment).config
//
// 用法：
//   - docsCommentConfig("comment")      取全局 COMMENT 那一层 json
//   - docsCommentConfig("docs","comment") 取 DOCS 模块 json 里的 comment
func docsCommentConfig(key ...any) map[string]any {
	configKey := "DOCS"
	if len(key) > 0 && cast.ToString(key[0]) == "comment" {
		configKey = "COMMENT"
	}

	cacheName := "config[" + configKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	var config any
	if cacheState && facade.Cache.Has(cacheName) {
		config = facade.Cache.Get(cacheName)
	} else {
		config, _ = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
		if cacheState {
			go facade.Cache.Set(cacheName, config)
		}
	}

	json := cast.ToStringMap(cast.ToStringMap(config)["json"])

	if len(key) > 1 {
		return cast.ToStringMap(json[cast.ToString(key[1])])
	}

	if len(key) > 0 {
		return cast.ToStringMap(json[cast.ToString(key[0])])
	}

	return json
}

// prevNext - 按 sort 顺序取上一篇 / 下一篇（同版本、已发布）
func (this *Docs) prevNext(item model.Docs) (gin.H, gin.H) {
	query := func(compare string) map[string]any {
		var row model.Docs
		q := facade.DB.Model(&model.Docs{}).
			Where("version", item.Version).
			Where("status", 1)

		if compare == "prev" {
			q = q.Where("sort", "<", item.Sort).Order("sort desc, id desc")
		} else {
			q = q.Where("sort", ">", item.Sort).Order("sort asc, id asc")
		}

		q.Scan(&row)
		if row.Id == 0 {
			return nil
		}

		groupSlug := this.groupSlug(row.Gid)
		return map[string]any{
			"id":    row.Id,
			"title": row.Title,
			"slug":  row.Slug,
			"path":  "/docs/" + row.Version + "/" + groupSlug + "/" + row.Slug,
		}
	}

	prev := query("prev")
	next := query("next")

	return gin.H(prev), gin.H(next)
}

// canPreview - 是否允许后台预览（草稿 / 待审核 / 受限文档）
func (this *Docs) canPreview(ctx *gin.Context) bool {
	return docsCanPreview(ctx)
}

// docsCanPreview - 是否具备文档后台预览权限。
//
// 判定口径：
//  1. permit：与权限中间件一致（root 分组，或当前路由已被授予权限点）
//  2. 兜底：拥有任意 /api/docs/ 管理权限点（部分站点把管理员放在非 root 分组，
//     只勾了文档管理，此时访问 docs/one 这种 common 路由不会命中第 1 条）
//
// 抽成包级函数是为了让 docs-group 的 tree 也复用：那边同样有 admin=1 绕过
// status 过滤的写法，匿名带上参数就能看到未启用的目录与草稿。
func docsCanPreview(ctx *gin.Context) bool {
	helper := meta{}

	if helper.permit(ctx) {
		return true
	}

	for _, item := range helper.rules(ctx) {
		if strings.Contains(cast.ToString(item), "/api/docs/") {
			return true
		}
	}

	return false
}

// canView - 可见性校验：1 公开 / 2 登录 / 3 指定角色组
func (this *Docs) canView(ctx *gin.Context, item *model.Docs) bool {
	return docsCanView(ctx, item)
}

// docsCanView - 文档可见性校验。
//
// 抽成包级函数是为了让评论接口（controller/comment.go 的 create）也能复用，
// 避免匿名用户对「登录可见 / 指定角色组」的文档发评论。
func docsCanView(ctx *gin.Context, item *model.Docs) bool {
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

func (this *Docs) uid(ctx *gin.Context) int {
	if user, ok := ctx.Get("user"); ok && user != nil {
		return cast.ToInt(cast.ToStringMap(user)["id"])
	}
	return 0
}

func (this *Docs) refreshFeedbackCount(docsId int) {
	useful, _ := facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", docsId).Where("useful", 1).Count()
	useless, _ := facade.DB.Model(&model.DocsFeedback{}).Where("docs_id", docsId).Where("useful", 0).Count()

	facade.DB.Model(&model.Docs{}).Where("id", docsId).Update(map[string]any{
		"useful":  useful,
		"useless": useless,
	})
}
