package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"math"
	"regexp"
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

// buildSearchQuery 构建搜索查询
func (this *Search) buildSearchQuery(keyword string, searchFields []string, auditCondition string) (string, map[string]any) {
	searchTerm := "%" + keyword + "%"
	var conditions []string
	var args []any

	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}

	query := strings.Join(conditions, " OR ")
	if auditCondition != "" {
		query = "(" + query + ") AND " + auditCondition
	}

	return query, map[string]any{
		"term": searchTerm,
		"args": args,
	}
}

// highlightKeyword 关键词高亮（大小写不敏感）
func (this *Search) highlightKeyword(text string, keyword string) string {
	if text == "" || keyword == "" {
		return text
	}
	// 转义正则特殊字符；(?i) 让英文关键词大小写都能命中
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(keyword))
	return re.ReplaceAllString(text, `<mark>$0</mark>`)
}

// 搜索结果「内容预览」用的清理规则（见 cleanSearchText）
var (
	searchHTMLTagRe      = regexp.MustCompile(`<[^>]*>`)
	searchMarkdownImgRe  = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	searchMarkdownLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	searchMarkdownLineRe = regexp.MustCompile(`(?m)^\s{0,3}(#{1,6}|>|[-*+]|\d+\.)\s+`)
	// 代码块围栏（含 ```go 这种语言标记）
	searchMarkdownFenceRe = regexp.MustCompile("(?m)^\\s*```.*$")
)

// cleanSearchText - 把正文整理成一段可读纯文本（用于搜索结果的摘要 / 内容预览）
//
// 只做轻量清理：去 HTML 标签、图片与链接语法、代码块围栏、行首 Markdown 标记，最后折叠空白。
// 目的是让「搜索结果里那一段话」看起来像正常文本，而不是一串标记。
func cleanSearchText(text string) string {
	if text == "" {
		return ""
	}

	text = searchHTMLTagRe.ReplaceAllString(text, " ")
	text = searchMarkdownImgRe.ReplaceAllString(text, " ")
	text = searchMarkdownLinkRe.ReplaceAllString(text, "$1")
	text = searchMarkdownFenceRe.ReplaceAllString(text, " ")
	text = searchMarkdownLineRe.ReplaceAllString(text, "")

	return strings.TrimSpace(strings.Join(strings.Fields(text), " "))
}

// snippetKeyword - 从正文里截取「包含关键词」的一段作为内容预览
//
// 返回的片段已做纯文本化处理，关键词用 <mark> 包裹（前端直接 v-html 展示即可）：
//   - 以关键词为锚点，前后各保留 before / after 个字符（按 rune 切，不会截断中文）；
//   - 截断处补省略号；
//   - 正文里找不到关键词（例如命中标题 / 标签）时，从开头截取。
func (this *Search) snippetKeyword(content string, keyword string, before, after int) string {
	text := cleanSearchText(content)
	if text == "" {
		return ""
	}
	if before <= 0 {
		before = 40
	}
	if after <= 0 {
		after = 100
	}

	runes := []rune(text)
	keyRunes := []rune(keyword)

	// 用字节索引定位，再换算成 rune 索引，避免把多字节字符切坏
	index := -1
	if keyword != "" {
		if byteIndex := strings.Index(strings.ToLower(text), strings.ToLower(keyword)); byteIndex >= 0 {
			index = len([]rune(text[:byteIndex]))
		}
	}
	if index < 0 {
		index = 0
	}

	start := index - before
	if start < 0 {
		start = 0
	}
	end := index + len(keyRunes) + after
	if end > len(runes) {
		end = len(runes)
	}

	snippet := this.highlightKeyword(string(runes[start:end]), keyword)
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}

	return snippet
}

// highlightResult 高亮搜索结果中的关键词
func (this *Search) highlightResult(data []map[string]any, keyword string) []map[string]any {
	if keyword == "" {
		return data
	}

	// 需要高亮的字段
	highlightFields := []string{"title", "content", "abstract", "description", "name", "nickname"}

	for _, item := range data {
		for _, field := range highlightFields {
			if val, ok := item[field].(string); ok {
				item[field] = this.highlightKeyword(val, keyword)
			}
		}
	}
	return data
}

// maskEmail 邮箱脱敏
func (this *Search) maskEmail(email string) string {
	if email == "" {
		return ""
	}

	// 按 @ 分割邮箱
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	// 如果用户名长度 <= 3，只保留第一位
	if len(username) <= 3 {
		return username[:1] + "***@" + domain
	}

	// 保留前两位和最后一位，中间用 *** 替换
	return username[:2] + "***" + username[len(username)-1:] + "@" + domain
}

// processSearchResult 处理搜索结果
func (this *Search) processSearchResult(items any, count int64, limit int, searchType string, keyword ...string) map[string]interface{} {
	var data []map[string]any

	// 内容预览要从正文里截，这里先把关键词取出来
	word := ""
	if len(keyword) > 0 {
		word = keyword[0]
	}

	switch v := items.(type) {
	case []model.Article:
		for _, article := range v {
			// snippet 为「内容预览」：摘要为空时前端会退化成展示这段正文片段
			data = append(data, map[string]any{
				"id":          article.Id,
				"title":       article.Title,
				"covers":      article.Covers,
				"abstract":    article.Abstract,
				"snippet":     this.snippetKeyword(article.Content, word, 40, 100),
				"create_time": article.CreateTime,
				"tags":        article.Tags,
				"views":       article.Views,
				"audit":       article.Audit,
			})
		}
	case []model.Pages:
		for _, page := range v {
			data = append(data, map[string]any{
				"id":          page.Id,
				"key":         page.Key,
				"title":       page.Title,
				"snippet":     this.snippetKeyword(page.Content, word, 40, 100),
				"create_time": page.CreateTime,
				"views":       page.Views,
				"audit":       page.Audit,
			})
		}
	case []model.Tags:
		for _, tag := range v {
			data = append(data, map[string]any{
				"id":          tag.Id,
				"name":        tag.Name,
				"avatar":      tag.Avatar,
				"description": tag.Description,
			})
		}
	case []model.Users:
		for _, user := range v {
			data = append(data, map[string]any{
				"id":          user.Id,
				"nickname":    user.Nickname,
				"avatar":      user.Avatar,
				"description": user.Description,
				"title":       user.Title,
				"email":       this.maskEmail(user.Email),
			})
		}
	case []model.Links:
		for _, link := range v {
			data = append(data, map[string]any{
				"id":          link.Id,
				"nickname":    link.Nickname,
				"avatar":      link.Avatar,
				"description": link.Description,
				"url":         link.Url,
				"audit":       link.Audit,
			})
		}
	case []model.Moments:
		for _, moment := range v {
			data = append(data, map[string]any{
				"id":          moment.Id,
				"content":     moment.Content,
				"images":      moment.Images,
				"location":    moment.Location,
				"create_time": moment.CreateTime,
				"audit":       moment.Audit,
				"status":      moment.Status,
			})
		}
	}

	// 高亮关键词
	if len(keyword) > 0 && keyword[0] != "" {
		data = this.highlightResult(data, keyword[0])
	}

	return map[string]interface{}{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
		"type":  searchType,
	}
}

// IGET - GET请求本体
func (this *Search) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"article": this.article,
		"pages":   this.pages,
		"tags":    this.tags,
		"users":   this.users,
		"links":   this.links,
		"moments": this.moments,
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
			"global":  "/api/search/all?keyword=关键词",
			"article": "/api/search/article?keyword=关键词",
			"pages":   "/api/search/pages?keyword=关键词",
			"tags":    "/api/search/tags?keyword=关键词",
			"users":   "/api/search/users?keyword=关键词",
			"links":   "/api/search/links?keyword=关键词",
			"moments": "/api/search/moments?keyword=关键词",
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
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	// 执行实际的文章搜索
	result := this.searchArticle(keyword, page, limit, fields)
	result["keyword"] = keyword

	// 如果没有搜索到结果，返回空数据而不是测试数据
	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// pages - 页面搜索
func (this *Search) pages(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	result := this.searchPages(keyword, page, limit, fields)
	result["keyword"] = keyword

	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// tags - 标签搜索
func (this *Search) tags(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	result := this.searchTags(keyword, page, limit, fields)
	result["keyword"] = keyword

	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// users - 用户搜索
func (this *Search) users(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	result := this.searchUsers(keyword, page, limit, fields)
	result["keyword"] = keyword

	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// links - 友链搜索
func (this *Search) links(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	result := this.searchLinks(keyword, page, limit, fields)
	result["keyword"] = keyword

	if result["count"] == int64(0) {
		result["data"] = []map[string]any{}
	}

	this.json(ctx, result, facade.Lang(ctx, "搜索成功！"), 200)
}

// moments - 动态搜索
func (this *Search) moments(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"keyword": "",
		"page":    1,
		"limit":   10,
		"fields":  "",
	})

	keyword := cast.ToString(params["keyword"])
	if keyword == "" {
		this.json(ctx, nil, facade.Lang(ctx, "搜索关键词不能为空！"), 400)
		return
	}

	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	fields := cast.ToString(params["fields"])

	result := this.searchMoments(keyword, page, limit, fields)
	result["keyword"] = keyword

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
func (this *Search) searchArticle(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	// 使用数据库级别的 LIKE 查询，提高性能
	db := facade.DB.Drive()

	// 构建搜索字段
	searchFields := []string{"title", "content", "abstract", "tags"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	// 构建搜索查询
	var conditions []string
	var args []any
	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}
	args = append(args, 1) // audit = 1

	var articles []model.Article
	query := db.Model(&articles).Where("("+strings.Join(conditions, " OR ")+") AND audit = ?", args...)

	// 统计总数
	var count int64
	query.Count(&count)

	// 分页查询
	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&articles)

	return this.processSearchResult(articles, count, limit, "article", keyword)
}

// searchPages - 搜索独立页面
func (this *Search) searchPages(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	db := facade.DB.Drive()

	searchFields := []string{"title", "content", "key"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	var conditions []string
	var args []any
	for _, field := range searchFields {
		if field == "key" {
			conditions = append(conditions, "`key` LIKE ?")
		} else {
			conditions = append(conditions, field+" LIKE ?")
		}
		args = append(args, searchTerm)
	}
	args = append(args, 1)

	var pages []model.Pages
	query := db.Model(&pages).Where("("+strings.Join(conditions, " OR ")+") AND audit = ?", args...)

	var count int64
	query.Count(&count)

	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&pages)

	return this.processSearchResult(pages, count, limit, "pages", keyword)
}

// searchTags - 搜索标签
func (this *Search) searchTags(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	db := facade.DB.Drive()

	searchFields := []string{"name", "description"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	var conditions []string
	var args []any
	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}

	var tags []model.Tags
	query := db.Model(&tags).Where("("+strings.Join(conditions, " OR ")+")", args...)

	var count int64
	query.Count(&count)

	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&tags)

	return this.processSearchResult(tags, count, limit, "tags", keyword)
}

// searchUsers - 搜索用户
func (this *Search) searchUsers(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	db := facade.DB.Drive()

	searchFields := []string{"nickname", "email", "description", "title"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	var conditions []string
	var args []any
	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}
	args = append(args, 0)

	var users []model.Users
	query := db.Model(&users).Where("("+strings.Join(conditions, " OR ")+") AND status = ?", args...)

	var count int64
	query.Count(&count)

	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&users)

	return this.processSearchResult(users, count, limit, "users", keyword)
}

// searchLinks - 搜索友链
func (this *Search) searchLinks(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	db := facade.DB.Drive()

	searchFields := []string{"nickname", "description", "url"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	var conditions []string
	var args []any
	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}
	args = append(args, 1)

	var links []model.Links
	query := db.Model(&links).Where("("+strings.Join(conditions, " OR ")+") AND audit = ?", args...)

	var count int64
	query.Count(&count)

	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&links)

	return this.processSearchResult(links, count, limit, "links", keyword)
}

// searchMoments - 搜索动态
func (this *Search) searchMoments(keyword string, page, limit int, fields ...string) map[string]interface{} {
	searchTerm := "%" + keyword + "%"

	db := facade.DB.Drive()

	searchFields := []string{"content", "location"}
	if len(fields) > 0 && fields[0] != "" {
		searchFields = strings.Split(fields[0], ",")
	}

	var conditions []string
	var args []any
	for _, field := range searchFields {
		conditions = append(conditions, field+" LIKE ?")
		args = append(args, searchTerm)
	}
	args = append(args, 1)

	var moments []model.Moments
	query := db.Model(&moments).Where("("+strings.Join(conditions, " OR ")+") AND audit = ?", args...)

	var count int64
	query.Count(&count)

	offset := (page - 1) * limit
	query.Limit(limit).Offset(offset).Order("create_time desc").Find(&moments)

	return this.processSearchResult(moments, count, limit, "moments", keyword)
}

// searchAll - 全局搜索
func (this *Search) searchAll(keyword string, page, limit int) map[string]interface{} {
	perTypeLimit := limit / 6
	if perTypeLimit < 1 {
		perTypeLimit = 1
	}

	articleResult := this.searchArticle(keyword, page, perTypeLimit)
	pagesResult := this.searchPages(keyword, page, perTypeLimit)
	tagsResult := this.searchTags(keyword, page, perTypeLimit)
	usersResult := this.searchUsers(keyword, page, perTypeLimit)
	linksResult := this.searchLinks(keyword, page, perTypeLimit)
	momentsResult := this.searchMoments(keyword, page, perTypeLimit)

	return map[string]interface{}{
		"article": articleResult,
		"pages":   pagesResult,
		"tags":    tagsResult,
		"users":   usersResult,
		"links":   linksResult,
		"moments": momentsResult,
		"total":   cast.ToInt64(articleResult["count"]) + cast.ToInt64(pagesResult["count"]) + cast.ToInt64(tagsResult["count"]) + cast.ToInt64(usersResult["count"]) + cast.ToInt64(linksResult["count"]) + cast.ToInt64(momentsResult["count"]),
		"type":    "all",
	}
}
