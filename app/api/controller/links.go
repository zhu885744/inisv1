package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Links struct {
	base
}

const (
	linksAllowFields = "nickname,description,url,avatar,target,group,json,text"
	linksAllowQuery  = "id"
)

var linksAllowFieldsSlice = []any{"nickname", "description", "url", "avatar", "target", "group", "json", "text"}
var linksAllowQuerySlice = []any{"id"}

func (this *Links) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Links) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Links) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Links) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Links) processFieldValue(val any) any {
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

func (this *Links) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Links{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	fields := utils.Unity.Keys(params["field"])
	if utils.Is.Empty(fields) {
		return nil, ""
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		result := make(map[string]any)
		for _, val := range fields {
			result[cast.ToString(val)] = aggFunc(query, cast.ToString(val))
		}
		data = result
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		msg[0] = "数据请求成功！"
	}

	return data, facade.Lang(ctx, strings.Join(msg, ""))
}

func (this *Links) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"sum":    this.sum,
		"min":    this.min,
		"max":    this.max,
		"rand":   this.rand,
		"count":  this.count,
		"column": this.column,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Links) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
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

func (this *Links) IPUT(ctx *gin.Context) {
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

func (this *Links) IDEL(ctx *gin.Context) {
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

func (this *Links) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

func (this *Links) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "links"})
}

func (this *Links) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"status": false,
	})

	table := model.Links{}

	for key, val := range params {
		if utils.In.Array(key, linksAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)
		if !this.meta.root(ctx) {
			query = query.Where("audit", 1)
		}
		item := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	checkStatus := cast.ToBool(params["status"])
	if checkStatus && !utils.Is.Empty(data) {
		data = this.checkSingleLinkStatus(ctx, data)
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Links) checkSingleLinkStatus(ctx *gin.Context, data any) any {
	item, ok := data.(map[string]any)
	if !ok {
		return data
	}

	url := cast.ToString(item["url"])
	if url == "" {
		item["online"] = false
		item["responseTime"] = 0
		return item
	}

	online, responseTime := this.checkURLStatus(url)
	item["online"] = online
	item["responseTime"] = responseTime

	return item
}

func (this *Links) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":   1,
		"order":  "create_time desc",
		"status": false,
	})

	table := model.Links{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Links

	query := this.withTrashOptions(facade.DB.Model(&result), params)
	query = this.buildQuery(query, params)
	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}
	count := query.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	checkStatus := cast.ToBool(params["status"])
	if checkStatus && !utils.Is.Empty(data) {
		if checkedItems, ok := this.checkLinksStatus(ctx, data).([]any); ok {
			data = checkedItems
		}
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Links) checkLinksStatus(ctx *gin.Context, data any) any {
	items, ok := data.([]any)
	if !ok {
		return data
	}

	numLinks := len(items)
	if numLinks == 0 {
		return data
	}

	statusMap := make(map[int]map[string]any, numLinks)
	mu := sync.Mutex{}

	cacheKey := "links_status_cache"
	if facade.Cache.Has(cacheKey) {
		cachedData := facade.Cache.Get(cacheKey)
		if cachedData != nil {
			if cachedStatus, ok := cachedData.(map[string]any); ok {
				for i, item := range items {
					if linkMap, linkOk := item.(map[string]any); linkOk {
						url := cast.ToString(linkMap["url"])
						if statusData, exists := cachedStatus[url]; exists {
							if statusMap, statusOk := statusData.(map[string]any); statusOk {
								if online, ok := statusMap["online"]; ok {
									linkMap["online"] = online
								}
								if responseTime, ok := statusMap["responseTime"]; ok {
									linkMap["responseTime"] = responseTime
								}
								continue
							}
						}
					}
					statusMap[i] = map[string]any{"online": false, "responseTime": 0}
				}
				go this.updateLinksStatusCache(items)
				return items
			}
		}
	}

	var wg sync.WaitGroup
	for i, item := range items {
		link, linkOk := item.(map[string]any)
		if !linkOk {
			continue
		}

		wg.Add(1)
		go func(index int, linkMap map[string]any) {
			defer wg.Done()

			url := cast.ToString(linkMap["url"])

			if url == "" {
				mu.Lock()
				statusMap[index] = map[string]any{
					"online":       false,
					"responseTime": 0,
				}
				mu.Unlock()
				return
			}

			online, responseTime := this.checkURLStatus(url)

			mu.Lock()
			statusMap[index] = map[string]any{
				"online":       online,
				"responseTime": responseTime,
			}
			mu.Unlock()
		}(i, link)
	}

	wg.Wait()

	this.cacheLinksStatus(items, statusMap)

	for i := range items {
		if status, ok := statusMap[i]; ok {
			if linkMap, ok := items[i].(map[string]any); ok {
				linkMap["online"] = status["online"]
				linkMap["responseTime"] = status["responseTime"]
			}
		}
	}

	return items
}

func (this *Links) cacheLinksStatus(items []any, statusMap map[int]map[string]any) {
	cachedStatus := make(map[string]any)
	for i, item := range items {
		if linkMap, ok := item.(map[string]any); ok {
			url := cast.ToString(linkMap["url"])
			if status, ok := statusMap[i]; ok {
				cachedStatus[url] = status
			}
		}
	}
	facade.Cache.Set("links_status_cache", cachedStatus, 300)
}

func (this *Links) updateLinksStatusCache(items []any) {
	statusMap := make(map[int]map[string]any)
	mu := sync.Mutex{}

	var wg sync.WaitGroup

	for i, item := range items {
		link, linkOk := item.(map[string]any)
		if !linkOk {
			continue
		}

		wg.Add(1)
		go func(index int, linkMap map[string]any) {
			defer wg.Done()

			url := cast.ToString(linkMap["url"])
			if url == "" {
				mu.Lock()
				statusMap[index] = map[string]any{"online": false, "responseTime": 0}
				mu.Unlock()
				return
			}

			online, responseTime := this.checkURLStatus(url)
			mu.Lock()
			statusMap[index] = map[string]any{"online": online, "responseTime": responseTime}
			mu.Unlock()
		}(i, link)
	}

	wg.Wait()
	this.cacheLinksStatus(items, statusMap)
}

func (this *Links) checkURLStatus(urlStr string) (bool, int) {
	urlStr = strings.TrimSpace(urlStr)
	urlStr = strings.Trim(urlStr, "`")
	urlStr = strings.TrimSpace(urlStr)

	if urlStr == "" {
		return false, 0
	}

	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	return this.tryCheckURL(urlStr, client, 1)
}

func (this *Links) tryCheckURL(urlStr string, client *http.Client, maxRetries int) (bool, int) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		start := time.Now()

		req, err := http.NewRequest("HEAD", urlStr, nil)
		if err != nil {
			req, err = http.NewRequest("GET", urlStr, nil)
			if err != nil {
				return false, 0
			}
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Connection", "close")

		resp, err := client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				if strings.HasPrefix(urlStr, "https://") {
					urlStr = "http://" + strings.TrimPrefix(urlStr, "https://")
				} else if strings.HasPrefix(urlStr, "http://") {
					urlStr = "https://" + strings.TrimPrefix(urlStr, "http://")
				}
				continue
			}
			return false, 0
		}
		defer resp.Body.Close()

		duration := time.Since(start)

		if this.isValidStatusCode(resp.StatusCode) {
			return true, int(duration.Milliseconds())
		}

		if attempt < maxRetries {
			if strings.HasPrefix(urlStr, "https://") {
				urlStr = "http://" + strings.TrimPrefix(urlStr, "https://")
			} else if strings.HasPrefix(urlStr, "http://") {
				urlStr = "https://" + strings.TrimPrefix(urlStr, "http://")
			}
			continue
		}

		return false, int(duration.Milliseconds())
	}

	return false, 0
}

func (this *Links) isValidStatusCode(statusCode int) bool {
	switch statusCode {
	case 200, 201, 202, 203, 204, 205, 206:
		return true
	case 301, 302, 303, 304, 307, 308:
		return true
	case 401, 403:
		return true
	case 404:
		return true
	case 500, 502, 503, 504:
		return false
	default:
		return statusCode >= 200 && statusCode < 500
	}
}

func (this *Links) rand(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{
		"status": false,
	})

	limit := this.meta.limit(ctx)
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.Links{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}
	if !this.meta.root(ctx) {
		query = query.Where("audit", 1)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Links{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	data := utils.Array.MapWithField(utils.Rand.MapSlice(mold.Select()), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	checkStatus := cast.ToBool(params["status"])
	if checkStatus && !utils.Is.Empty(data) {
		if checkedItems, ok := this.checkLinksStatus(ctx, data).([]any); ok {
			data = checkedItems
		}
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

func (this *Links) save(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Links) create(ctx *gin.Context) {
	params := this.params(ctx)
	err := validator.NewValid("links", params)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Links{Uid: uid, CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}
	allow := linksAllowFieldsSlice

	if this.meta.root(ctx) {
		allow = append(allow, "audit", "remark")
	}

	for key, val := range params {
		if utils.Get.Type(val) == "string" {
			if key == "nickname" || key == "description" || key == "url" || key == "avatar" || key == "remark" || key == "text" {
				if facade.Comm.DetectXSS(cast.ToString(val)) {
					this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
					return
				}
				val = facade.Comm.SanitizeHTML(cast.ToString(val))
			}
		}
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, this.processFieldValue(val))
		}
	}

	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *Links) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	err := validator.NewValid("links", params)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	table := model.Links{}
	async := utils.Async[map[string]any]()
	root := this.meta.root(ctx)
	allow := linksAllowFieldsSlice

	if root {
		allow = append(allow, "audit", "remark")
	}

	for key, val := range params {
		if utils.Get.Type(val) == "string" {
			if key == "nickname" || key == "description" || key == "url" || key == "avatar" || key == "remark" || key == "text" {
				if facade.Comm.DetectXSS(cast.ToString(val)) {
					this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
					return
				}
				val = facade.Comm.SanitizeHTML(cast.ToString(val))
			}
		}
		if utils.In.Array(key, allow) {
			async.Set(key, this.processFieldValue(val))
		}
	}

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	if !root && cast.ToInt(item.Find()["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	tx := item.Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Links) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := facade.DB.Model(&model.Links{})
	query = this.buildQuery(query, params)
	this.json(ctx, query.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Links) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Sum(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Links) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Min(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Links) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Max(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Links) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&[]model.Links{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		data = utils.ArrayMapWithField(query.Select(), params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Links) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Links{})

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := item.Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Links) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Links{}).WithTrashed()

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := item.Force().Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Links) clear(ctx *gin.Context) {
	table := model.Links{}
	item := facade.DB.Model(&table).OnlyTrashed()

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids := utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := item.Force().Delete()

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *Links) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Links{}).OnlyTrashed().WhereIn("id", ids)

	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.Links{}).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}