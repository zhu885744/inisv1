package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Search - 搜索控制器
// @Summary 搜索管理API
// @Description 提供高效的全局搜索功能，支持多表搜索
// @Tags Search
type Search struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Search) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"article": this.article,
		"pages":   this.pages,
		"tags":    this.tags,
		"all":     this.all,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Search) IPOST(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "搜索控制器不支持POST请求"), 405)
}

// IPUT - PUT请求本体
func (this *Search) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "搜索控制器不支持PUT请求"), 405)
}

// IDEL - DELETE请求本体
func (this *Search) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "搜索控制器不支持DELETE请求"), 405)
}

// INDEX - 搜索首页
func (this *Search) INDEX(ctx *gin.Context) {
	this.json(ctx, map[string]interface{}{
		"message": "搜索控制器首页",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"global":  "/api/search/index?keyword=关键词",
			"article": "/api/search/article?keyword=关键词",
			"pages":   "/api/search/pages?keyword=关键词",
			"tags":    "/api/search/tags?keyword=关键词",
		},
	}, facade.Lang(ctx, "搜索控制器首页"), 200)
}

// article - 文章搜索
func (this *Search) article(ctx *gin.Context) {
	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])

	// 执行实际的文章搜索
	result := this.searchArticle(keyword, page, limit)
	result["keyword"] = keyword

	// 如果没有搜索到结果，返回空数据而不是测试数据
	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// pages - 页面搜索
func (this *Search) pages(ctx *gin.Context) {
	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])

	// 执行实际的页面搜索
	result := this.searchPages(keyword, page, limit)
	result["keyword"] = keyword

	// 如果没有搜索到结果，返回空数据
	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// tags - 标签搜索
func (this *Search) tags(ctx *gin.Context) {
	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])

	// 执行实际的标签搜索
	result := this.searchTags(keyword, page, limit)
	result["keyword"] = keyword

	// 如果没有搜索到结果，返回空数据
	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// all - 全局搜索
func (this *Search) all(ctx *gin.Context) {
	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])

	// 执行实际的全局搜索
	result := this.searchAll(keyword, page, limit)
	result["keyword"] = keyword

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// searchArticle - 搜索文章
func (this *Search) searchArticle(keyword string, page, limit int) map[string]interface{} {
	// 构建搜索条件
	searchTerm := "%" + keyword + "%"

	// 使用数据库级别的 LIKE 查询，提高性能
	// 注意：直接使用底层的 GORM 连接来构建复杂查询
	db := facade.DB.Drive()

	// 构建搜索查询
	var articles []model.Article
	query := db.Model(&articles).Where("(title LIKE ? OR content LIKE ? OR abstract LIKE ? OR tags LIKE ?) AND audit = ?", searchTerm, searchTerm, searchTerm, searchTerm, 1)

	// 统计总数
	var count int64
	query.Count(&count)

	// 分页查询
	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&articles)

	// 处理返回字段
	var data []map[string]any
	for _, article := range articles {
		item := map[string]any{
			"id":          article.Id,
			"title":       article.Title,
			"covers":      article.Covers,
			"abstract":    article.Abstract,
			"create_time": article.CreateTime,
			"tags":        article.Tags,
			"views":       article.Views,
			"audit":       article.Audit,
		}
		data = append(data, item)
	}

	return map[string]interface{}{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
		"type":  "article",
	}
}

// searchPages - 搜索独立页面
func (this *Search) searchPages(keyword string, page, limit int) map[string]interface{} {
	// 构建搜索条件
	searchTerm := "%" + keyword + "%"

	// 使用数据库级别的 LIKE 查询，提高性能
	// 注意：直接使用底层的 GORM 连接来构建复杂查询
	db := facade.DB.Drive()

	// 构建搜索查询
	var pages []model.Pages
	query := db.Model(&pages).Where("(title LIKE ? OR content LIKE ? OR `key` LIKE ?) AND audit = ?", searchTerm, searchTerm, searchTerm, 1)

	// 统计总数
	var count int64
	query.Count(&count)

	// 分页查询
	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&pages)

	// 处理返回字段
	var data []map[string]any
	for _, page := range pages {
		item := map[string]any{
			"id":          page.Id,
			"key":         page.Key,
			"title":       page.Title,
			"create_time": page.CreateTime,
			"views":       page.Views,
			"audit":       page.Audit,
		}
		data = append(data, item)
	}

	return map[string]interface{}{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
		"type":  "pages",
	}
}

// searchTags - 搜索标签
func (this *Search) searchTags(keyword string, page, limit int) map[string]interface{} {
	// 构建搜索条件
	searchTerm := "%" + keyword + "%"

	// 使用数据库级别的 LIKE 查询，提高性能
	// 注意：直接使用底层的 GORM 连接来构建复杂查询
	db := facade.DB.Drive()

	// 构建搜索查询
	var tags []model.Tags
	query := db.Model(&tags).Where("(name LIKE ? OR description LIKE ?)", searchTerm, searchTerm)

	// 统计总数
	var count int64
	query.Count(&count)

	// 分页查询
	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&tags)

	// 处理返回字段
	var data []map[string]any
	for _, tag := range tags {
		item := map[string]any{
			"id":          tag.Id,
			"name":        tag.Name,
			"avatar":      tag.Avatar,
			"description": tag.Description,
		}
		data = append(data, item)
	}

	return map[string]interface{}{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
		"type":  "tags",
	}
}

// searchAll - 全局搜索
func (this *Search) searchAll(keyword string, page, limit int) map[string]interface{} {
	// 构建搜索条件
	searchTerm := "%" + keyword + "%"

	// 使用底层的 GORM 连接
	db := facade.DB.Drive()

	// 搜索文章 - 只搜索审核通过的文章
	var articles []model.Article
	articleQuery := db.Model(&articles).Where("(title LIKE ? OR content LIKE ? OR abstract LIKE ? OR tags LIKE ?) AND audit = ?", searchTerm, searchTerm, searchTerm, searchTerm, 1)
	var articleCount int64
	articleQuery.Count(&articleCount)
	articleQuery.Limit(limit / 3).Order("create_time desc").Find(&articles)

	// 处理文章数据
	var articleData []map[string]any
	for _, article := range articles {
		item := map[string]any{
			"id":          article.Id,
			"title":       article.Title,
			"covers":      article.Covers,
			"abstract":    article.Abstract,
			"create_time": article.CreateTime,
			"tags":        article.Tags,
			"views":       article.Views,
			"audit":       article.Audit,
		}
		articleData = append(articleData, item)
	}

	// 搜索独立页面
	var pages []model.Pages
	pagesQuery := db.Model(&pages).Where("(title LIKE ? OR content LIKE ? OR abstract LIKE ?)", searchTerm, searchTerm, searchTerm)
	var pagesCount int64
	pagesQuery.Count(&pagesCount)
	pagesQuery.Limit(limit / 3).Order("create_time desc").Find(&pages)

	// 处理页面数据
	var pagesData []map[string]any
	for _, page := range pages {
		item := map[string]any{
			"id":          page.Id,
			"key":         page.Key,
			"title":       page.Title,
			"create_time": page.CreateTime,
			"views":       page.Views,
		}
		pagesData = append(pagesData, item)
	}

	// 搜索标签
	var tags []model.Tags
	tagsQuery := db.Model(&tags).Where("(name LIKE ? OR description LIKE ?)", searchTerm, searchTerm)
	var tagsCount int64
	tagsQuery.Count(&tagsCount)
	tagsQuery.Limit(limit / 3).Order("create_time desc").Find(&tags)

	// 处理标签数据
	var tagsData []map[string]any
	for _, tag := range tags {
		item := map[string]any{
			"id":          tag.Id,
			"name":        tag.Name,
			"avatar":      tag.Avatar,
			"description": tag.Description,
		}
		tagsData = append(tagsData, item)
	}

	return map[string]interface{}{
		"article": map[string]interface{}{
			"data":  articleData,
			"count": articleCount,
		},
		"pages": map[string]interface{}{
			"data":  pagesData,
			"count": pagesCount,
		},
		"tags": map[string]interface{}{
			"data":  tagsData,
			"count": tagsCount,
		},
		"total": articleCount + pagesCount + tagsCount,
		"type":  "all",
	}
}
