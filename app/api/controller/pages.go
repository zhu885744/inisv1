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

// Pages - 页面管理控制器
// @Summary 页面管理API
// @Description 提供页面相关的CRUD操作及数据统计功能
// @Tags Pages
type Pages struct {
	// 继承
	base
}

// IGET - 获取页面数据
// @Summary 获取页面数据
// @Description 根据不同方法获取页面相关数据
// @Tags Pages
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(one, all, sum, min, max, rand, count, column)
// @Param id query int false "页面ID"
// @Param key query string false "页面标识"
// @Param where query string false "查询条件"
// @Param order query string false "排序方式"
// @Param field query string false "字段过滤"
// @Param page query int false "页码"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/pages/{method} [get]
func (this *Pages) IGET(ctx *gin.Context) {
	// 转小写
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

// IPOST - 创建/保存页面
// @Summary 创建/保存页面
// @Description 创建新页面或保存页面数据（包含创建和更新）
// @Tags Pages
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(save, create)
// @Param key formData string true "页面标识"
// @Param title formData string true "页面标题"
// @Param content formData string true "页面内容"
// @Param remark formData string false "页面备注"
// @Param tags formData string false "页面标签"
// @Param editor formData string false "编辑器类型"
// @Param json formData string false "JSON数据"
// @Param text formData string false "文本数据"
// @Success 200 {object} map[string]interface{} "成功响应，包含页面ID"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/pages/{method} [post]
func (this *Pages) IPOST(ctx *gin.Context) {
    method := strings.ToLower(ctx.Param("method"))
    allow := map[string]any{
        "save":   this.save,
        "create": this.create,
        "update": this.update,
    }
    err := this.call(allow, method, ctx)
    if err != nil {
        this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
        return
    }
    go this.delCache() // 新增删除缓存方法
}

// IPUT - 更新/恢复页面数据
// @Summary 更新/恢复页面数据
// @Description 根据不同方法更新或恢复页面相关数据
// @Tags Pages
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(update, restore)
// @Param id formData int true "页面ID"
// @Param key formData string false "页面标识"
// @Param title formData string false "页面标题"
// @Param content formData string false "页面内容"
// @Param remark formData string false "页面备注"
// @Param tags formData string false "页面标签"
// @Param editor formData string false "编辑器类型"
// @Param json formData string false "JSON数据"
// @Param text formData string false "文本数据"
// @Success 200 {object} map[string]interface{} "成功响应，包含页面ID"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/pages/{method} [put]
func (this *Pages) IPUT(ctx *gin.Context) {
	// 转小写
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

	// 删除缓存
	go this.delCache()
}

// IDEL - 删除页面数据
// @Summary 删除页面数据
// @Description 根据不同方法删除页面数据（支持软删除、硬删除和清空回收站）
// @Tags Pages
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(remove, delete, clear)
// @Param ids formData string true "页面ID列表，逗号分隔"
// @Success 200 {object} map[string]interface{} "成功响应，包含删除的ID列表"
// @Failure 401 {object} map[string]interface{} "未登录"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/pages/{method} [delete]
func (this *Pages) IDEL(ctx *gin.Context) {
	// 转小写
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

	// 删除缓存
	go this.delCache()
}

func (this *Pages) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// 删除缓存
func (this *Pages) delCache() {
	// 删除缓存
	facade.Cache.DelTags([]any{"[GET]", "pages"})
}

// one 获取指定数据
func (this *Pages) one(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx)

	// 表数据结构体
	table := model.Pages{}
	// 允许查询的字段
	allow := []any{"id", "key"}
	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		mold := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
		mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])
		item := mold.Where(table).Find()

		// 排除字段
		data = facade.Comm.WithField(item, params["field"])

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	// 更新用户经验
	go func() {
		user := this.meta.user(ctx)
		// 用户未登录
		if user.Id == 0 {
			return
		}
		item := cast.ToStringMap(data)
		// 数据不存在
		if utils.Is.Empty(item) {
			return
		}
		_ = (&model.EXP{}).Add(model.EXP{
			Uid:      user.Id,
			Type:     "visit",
			BindId:   cast.ToInt(item["id"]),
			BindType: "page",
		})
	}()

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all 获取全部数据
func (this *Pages) all(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	// 表数据结构体
	table := model.Pages{}
	// 允许查询的字段
	var allow []any
	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Pages
	mold := facade.DB.Model(&result).WithoutField("content").OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
	mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])
	count := mold.Where(table).Count()

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		// 从数据库中获取数据
		item := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()

		// 排除字段
		data = utils.ArrayMapWithField(item, params["field"])

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

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

// rand 随机获取
func (this *Pages) rand(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 限制最大数量
	limit := this.meta.limit(ctx)

	// 排除的 id 列表
	except := utils.Unity.Ids(params["except"])

	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	item := facade.DB.Model(&model.Pages{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		item = item.Where("id", "NOT IN", except)
	}

	// 从全部的 id 中随机选取指定数量的 id
	ids := utils.Rand.Slice(utils.Unity.Ids(item.Column("id")), limit)

	// 查询条件
	mold := facade.DB.Model(&[]model.Pages{}).Where("id", "IN", ids).WithoutField("content")
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed).IWhere(params["where"]).IOr(params["or"])
	mold.ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// 查询并打乱顺序
	data := utils.Array.MapWithField(utils.Rand.MapSlice(mold.Select()), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

// save 保存数据 - 包含创建和更新
func (this *Pages) save(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create 创建数据
func (this *Pages) create(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)
	// 验证器
	err := validator.NewValid("pages", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	uid := this.meta.user(ctx).Id
	if uid == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 处理发布时间：优先使用传入的publish_time，否则使用当前时间
	publishTime := time.Now().Unix()
	if pt, ok := params["publish_time"]; ok && cast.ToInt64(pt) > 0 {
		publishTime = cast.ToInt64(pt)
	}

	// 表数据结构体（新增PublishTime字段）
	table := model.Pages{
		Uid:         uid,
		CreateTime:  time.Now().Unix(),
		UpdateTime:  time.Now().Unix(),
		LastUpdate:  time.Now().Unix(),
		PublishTime: publishTime, // 赋值发布时间
	}
	
	// 允许的字段添加publish_time
	allow := []any{"key", "title", "content", "remark", "tags", "editor", "json", "text", "publish_time"}

	// 越权 - 增加可选字段
	if this.meta.root(ctx) {
		allow = append(allow, "audit")
	}

	// 是否开启了审核
	audit := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
	// 更新审核状态
	utils.Struct.Set(&table, "audit", cast.ToInt(!audit))

	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			switch utils.Get.Type(val) {
			case "map":
				val = utils.Json.Encode(val)
			case "2d slice":
				val = utils.Json.Encode(val)
			case "slice":
				val = strings.Join(cast.ToStringSlice(val), ",")
			}
			utils.Struct.Set(&table, key, val)
		}
	}

	// 添加数据
	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新数据
func (this *Pages) update(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
        this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
        return
    }

	// 验证器
	err := validator.NewValid("pages", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 表数据结构体
	table := model.Pages{}
	// 允许更新的字段添加publish_time
	allow := []any{"key", "title", "content", "remark", "tags", "editor", "json", "text", "publish_time"}
	async := utils.Async[map[string]any]()

	// 处理发布时间（如果有传入则更新）
	if pt, ok := params["publish_time"]; ok && cast.ToInt64(pt) > 0 {
		async.Set("publish_time", cast.ToInt64(pt))
	}

	// 越权 - 增加可选字段
	if this.meta.root(ctx) {
		allow = append(allow, "audit")
	}

	// 是否开启了审核
	audit := cast.ToBool(cast.ToStringMap(this.config(ctx)["json"])["audit"])
	// 更新审核状态
	utils.Struct.Set(&table, "audit", cast.ToInt(!audit))

	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			switch utils.Get.Type(val) {
			case "map":
				val = utils.Json.Encode(val)
			case "2d slice":
				val = utils.Json.Encode(val)
			case "slice":
				val = strings.Join(cast.ToStringSlice(val), ",")
			}
			async.Set(key, val)
		}
	}

	// 更新时间
	async.Set("last_update", time.Now().Unix())

	// 更新数据
	tx := facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// count 统计数据
func (this *Pages) count(ctx *gin.Context) {

	// 表数据结构体
	table := model.Pages{}
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table)
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	this.json(ctx, item.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

// sum 求和
func (this *Pages) sum(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Pages
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"])).Order(params["order"])
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// id 数组 - 参数归一化
	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		item.WhereIn("id", ids)
	}

	// field 数组 - 参数归一化
	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		result := make(map[string]any)

		for _, val := range fields {
			result[cast.ToString(val)] = item.Sum(val)
		}

		// 从数据库中获取数据 - 排除字段
		data = result

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// min 求最小值
func (this *Pages) min(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Pages
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"])).Order(params["order"])
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// id 数组 - 参数归一化
	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		item.WhereIn("id", ids)
	}

	// field 数组 - 参数归一化
	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		result := make(map[string]any)

		for _, val := range fields {
			result[cast.ToString(val)] = item.Min(val)
		}

		// 从数据库中获取数据 - 排除字段
		data = result

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// max 求最大值
func (this *Pages) max(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Pages
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"])).Order(params["order"])
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// id 数组 - 参数归一化
	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		item.WhereIn("id", ids)
	}

	// field 数组 - 参数归一化
	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		result := make(map[string]any)

		for _, val := range fields {
			result[cast.ToString(val)] = item.Max(val)
		}

		// 从数据库中获取数据 - 排除字段
		data = result

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// column 获取单列数据
func (this *Pages) column(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table []model.Pages
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"])).Order(params["order"])
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// id 数组 - 参数归一化
	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		item.WhereIn("id", ids)
	}

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		// 从数据库中获取数据 - 排除字段
		data = utils.ArrayMapWithField(item.Select(), params["field"])

		// 缓存数据
		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// remove 软删除
func (this *Pages) remove(ctx *gin.Context) {

	// 表数据结构体
	table := model.Pages{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table)

	// 得到允许操作的 id 数组
	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 软删除
	tx := item.Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// delete 真实删除
func (this *Pages) delete(ctx *gin.Context) {

	// 表数据结构体
	table := model.Pages{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table).WithTrashed()

	// 得到允许操作的 id 数组
	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 真实删除
	tx := item.Force().Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// clear 清空回收站
func (this *Pages) clear(ctx *gin.Context) {

	// 表数据结构体
	table := model.Pages{}

	item := facade.DB.Model(&table).OnlyTrashed()

	ids := utils.Unity.Ids(item.Column("id"))

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 找到所有软删除的数据
	tx := item.Force().Delete()

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

// restore 恢复数据
func (this *Pages) restore(ctx *gin.Context) {

	// 表数据结构体
	table := model.Pages{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table).OnlyTrashed().WhereIn("id", ids)

	// 得到允许操作的 id 数组
	ids = utils.Unity.Ids(item.Column("id"))

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 还原数据
	tx := facade.DB.Model(&table).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// 获取配置
func (this *Pages) config(ctx *gin.Context) (result map[string]any) {

	// 是否允许注册
	cacheName := "[GET]config[PAGE]"

	// 如果缓存中存在，则直接使用缓存中的数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		return cast.ToStringMap(facade.Cache.Get(cacheName))
	}

	// 不存在则查询数据库
	result = facade.DB.Model(&model.Config{}).Where("key", "PAGE").Find()
	// 写入缓存
	go facade.Cache.Set(cacheName, result)

	return result
}
