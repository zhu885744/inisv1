package controller

import (
	"fmt"
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

type Users struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Users) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":       this.one,
		"all":       this.all,
		"sum":       this.sum,
		"min":       this.min,
		"max":       this.max,
		"rand":      this.rand,
		"count":     this.count,
		"column":    this.column,
		"blackroom": this.blackroom,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Users) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":          this.save,
		"create":        this.create,
		"appeal":        this.appeal,
		"appeal-public": this.appealPublic,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	// 删除缓存
	go this.delCache()
}

// IPUT - PUT请求本体
func (this *Users) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":        this.update,
		"restore":       this.restore,
		"email":         this.email,
		"phone":         this.phone,
		"status":        this.status,
		"ban":           this.ban,
		"unban":         this.unban,
		"appeal-handle": this.appealHandle,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	// 删除缓存
	go this.delCache()
}

// IDEL - DELETE请求本体
func (this *Users) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove":  this.remove,
		"delete":  this.delete,
		"clear":   this.clear,
		"destroy": this.destroy,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}

	// 删除缓存
	go this.delCache()
}

// INDEX - GET请求本体
func (this *Users) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// 删除缓存
func (this *Users) delCache() {
	// 删除缓存
	facade.Cache.DelTags([]any{"[GET]", "users"})
}

// one 获取指定数据
func (this *Users) one(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx)

	// 表数据结构体
	table := model.Users{}
	// 允许查询的字段
	allow := []any{"id", "email"}
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

		mold.WithoutField("password")

		user := this.user(ctx)
		isAdmin := this.meta.root(ctx)
		isOwnData := table.Id == user.Id && user.Id != 0

		// 从数据库中获取数据
		item, _ := mold.Where(table).Find()

		// 非管理员查看他人数据时，对敏感字段进行脱敏处理
		if !isAdmin && !isOwnData && !utils.Is.Empty(item) {
			itemMap := cast.ToStringMap(item)
			if email, ok := itemMap["email"].(string); ok && email != "" {
				itemMap["email"] = facade.Comm.MaskEmail(email)
			}
			if phone, ok := itemMap["phone"].(string); ok && phone != "" {
				itemMap["phone"] = facade.Comm.MaskPhone(phone)
			}
			if account, ok := itemMap["account"].(string); ok && account != "" {
				accountLen := len(account)
				if accountLen > 2 {
					itemMap["account"] = account[:2] + strings.Repeat("*", accountLen-2)
				}
			}
			item = itemMap
		}

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

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all 获取全部数据
func (this *Users) all(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	// 表数据结构体
	table := model.Users{}
	// 允许查询的字段
	allow := []any{"source"}
	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Users
	mold := facade.DB.Model(&result).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
	mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])
	count, _ := mold.Where(table).Count()

	cacheName := this.cache.name(ctx)
	// 开启了缓存 并且 缓存中有数据
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {

		// 从缓存中获取数据
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)

	} else {

		mold.WithoutField("password")

		isAdmin := this.meta.root(ctx)

		// 从数据库中获取数据
		item, _ := mold.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()

		// 排除字段
		data = utils.ArrayMapWithField(item, params["field"])

		// 非管理员查看列表时，对数据进行脱敏处理
		if !isAdmin {
			dataList := cast.ToSlice(data)
			for i, val := range dataList {
				dataMap := cast.ToStringMap(val)
				if email, ok := dataMap["email"].(string); ok && email != "" {
					dataMap["email"] = facade.Comm.MaskEmail(email)
				}
				if phone, ok := dataMap["phone"].(string); ok && phone != "" {
					dataMap["phone"] = facade.Comm.MaskPhone(phone)
				}
				if account, ok := dataMap["account"].(string); ok && account != "" {
					accountLen := len(account)
					if accountLen > 2 {
						dataMap["account"] = account[:2] + strings.Repeat("*", accountLen-2)
					}
				}
				dataList[i] = dataMap
			}
			data = dataList
		}

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
func (this *Users) rand(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 限制最大数量
	limit := this.meta.limit(ctx)

	// 排除的 id 列表
	except := utils.Unity.Ids(params["except"])

	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	item := facade.DB.Model(&model.Users{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		item = item.Where("id", "NOT IN", except)
	}

	// 从全部的 id 中随机选取指定数量的 id
	ids := utils.Rand.Slice(utils.Unity.Ids(item.Column("id")), limit)

	// 查询条件
	mold := facade.DB.Model(&[]model.Users{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed).IWhere(params["where"]).IOr(params["or"])
	mold.ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])
	mold.WithoutField("password")

	// 越权 - 没有管理权限
	if !this.meta.root(ctx) {
		mold.WithoutField("account", "email", "phone")
	}

	// 查询并打乱顺序
	items, _ := mold.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "数据请求成功！"), 200)
}

// save 保存数据 - 包含创建和更新
func (this *Users) save(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create 创建数据
func (this *Users) create(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)
	// 验证器
	err := validator.NewValid("users", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 表数据结构体
	table := model.Users{CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix()}
	allow := []any{"account", "password", "nickname", "email", "phone", "avatar", "description", "source", "remark", "title", "gender", "json", "text", "status"}

	if utils.Is.Empty(params["email"]) {
		this.json(ctx, nil, facade.Lang(ctx, "邮箱不能为空！"), 400)
		return
	}

	// 动态给结构体赋值
	for key, val := range params {
		// 加密密码
		if key == "password" {
			val = utils.Password.Create(params["password"])
		} else if utils.Get.Type(val) == "string" {
			// 检测是否包含XSS攻击
			if key == "account" || key == "nickname" || key == "avatar" || key == "description" || key == "remark" || key == "title" || key == "text" {
				if facade.Comm.DetectXSS(cast.ToString(val)) {
					this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
					return
				}
				val = facade.Comm.SanitizeHTML(cast.ToString(val))
			}
		}
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

	// 创建用户
	_, err = facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新数据
func (this *Users) update(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)
	var err error

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	// 验证器
	err = validator.NewValid("users", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 表数据结构体
	table := model.Users{}
	allow := []any{"id", "account", "password", "nickname", "avatar", "title", "description", "gender", "json", "text", "status"}
	async := utils.Async[map[string]any]()

	root := this.meta.root(ctx)
	// 越权 - 增加可选字段
	if root {
		allow = append(allow, "source", "remark", "email", "phone")
	}

	// 动态给结构体赋值
	for key, val := range params {
		// 加密密码
		if key == "password" {
			// 密码为空时不更新此项
			if utils.Is.Empty(val) {
				continue
			}
			val = utils.Password.Create(params["password"])
		} else if utils.Get.Type(val) == "string" {
			// 检测是否包含XSS攻击
			if key == "account" || key == "nickname" || key == "avatar" || key == "description" || key == "remark" || key == "title" || key == "text" {
				if facade.Comm.DetectXSS(cast.ToString(val)) {
					this.json(ctx, nil, facade.Lang(ctx, "内容包含恶意代码，禁止提交！"), 400)
					return
				}
				val = facade.Comm.SanitizeHTML(cast.ToString(val))
			}
		}
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

	// 越权 - 既没有管理权限，也不是自己的数据
	if !root && cast.ToInt(params["id"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	// 更新用户
	_, err = facade.DB.Model(&table).WithTrashed().Where("id", params["id"]).Scan(&table).Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 删除缓存
	facade.Cache.Del(fmt.Sprintf("user[%v]", params["id"]))

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// status 修改用户状态
func (this *Users) status(ctx *gin.Context) {
	// 获取请求参数
	params := this.params(ctx)

	// 验证ID和状态参数
	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}
	if utils.Is.Empty(params["status"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "status"), 400)
		return
	}

	// 验证权限 - 只有管理员可以修改状态
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限修改用户状态！"), 403)
		return
	}

	// 禁止修改系统管理员状态
	userId := cast.ToInt(params["id"])
	if userId == 1 {
		this.json(ctx, nil, facade.Lang(ctx, "禁止修改系统管理员状态！"), 403)
		return
	}

	// 验证状态值是否合法
	status := cast.ToInt(params["status"])
	if status != 0 && status != 1 {
		this.json(ctx, nil, facade.Lang(ctx, "状态值必须为0或1！"), 400)
		return
	}

	// 更新状态
	table := model.Users{}
	tx, err := facade.DB.Model(&table).Where("id", userId).UpdateColumn("status", status)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 检查是否有数据被更新
	if tx.RowsAffected == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "未找到用户或状态未变更！"), 204)
		return
	}

	// 删除缓存
	facade.Cache.Del(fmt.Sprintf("user[%v]", userId))

	this.json(ctx, gin.H{"id": userId, "status": status}, facade.Lang(ctx, "状态更新成功！"), 200)
}

// count 统计数据
func (this *Users) count(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table)
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	count, _ := item.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

// sum 求和
func (this *Users) sum(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}

	// 使用聚合查询
	data, cacheMsg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Order(ctx.Request.URL.Query()["order"]).Sum(field)
		return result
	})
	msg[1] = cacheMsg

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// min 求最小值
func (this *Users) min(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}

	// 使用聚合查询
	data, cacheMsg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Order(ctx.Request.URL.Query()["order"]).Min(field)
		return result
	})
	msg[1] = cacheMsg

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// max 求最大值
func (this *Users) max(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}

	// 使用聚合查询
	data, cacheMsg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Order(ctx.Request.URL.Query()["order"]).Max(field)
		return result
	})
	msg[1] = cacheMsg

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// column 获取单列数据
func (this *Users) column(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx)

	// 从缓存或数据库获取数据
	data, msg[1] = this.getFromCache(ctx, params, func() any {
		item := this.buildQuery(ctx, &model.Users{}, params)
		item.WithoutField("password")
		if !this.meta.root(ctx) {
			item.WithoutField("account", "email", "phone")
		}
		items, _ := item.Select()
		return utils.ArrayMapWithField(items, params["field"])
	})

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// aggregateQuery 聚合查询（sum/min/max）
func (this *Users) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {

	// 获取请求参数
	params := this.params(ctx)

	// field 数组 - 参数归一化
	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return nil, ""
	}

	// 从缓存或数据库获取数据
	return this.getFromCache(ctx, params, func() any {
		query := this.buildQuery(ctx, &model.Users{}, params)
		result := make(map[string]any)
		for _, val := range fields {
			result[cast.ToString(val)] = aggFunc(query, cast.ToString(val))
		}
		return result
	})
}

// getFromCache 通用缓存处理
func (this *Users) getFromCache(ctx *gin.Context, params map[string]any, fetchFunc func() any) (any, string) {
	msg := ""
	cacheName := this.cache.name(ctx)

	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		msg = "（来自缓存）"
		return facade.Cache.Get(cacheName), msg
	}

	data := fetchFunc()
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data)
	}
	return data, msg
}

// buildQuery 构建通用查询
func (this *Users) buildQuery(ctx *gin.Context, table any, params map[string]any) *facade.ModelStruct {
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])
	mold := facade.DB.Model(table).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// id 数组 - 参数归一化
	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		mold.WhereIn("id", ids)
	}

	return mold
}

// remove 软删除
func (this *Users) remove(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 检查是否为系统管理员
	if utils.In.Array(1, ids) {
		this.json(ctx, nil, facade.Lang(ctx, "禁止删除系统管理员账户！"), 403)
		return
	}

	if utils.In.Array(this.meta.user(ctx).Id, ids) {
		this.json(ctx, nil, facade.Lang(ctx, "不能删除自己！"), 400)
		return
	}

	item := facade.DB.Model(&table)

	// 得到允许操作的 id 数组
	columnData, _ := item.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 软删除
	_, err := item.Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// delete 真实删除
func (this *Users) delete(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 检查是否为系统管理员
	if utils.In.Array(1, ids) {
		this.json(ctx, nil, facade.Lang(ctx, "禁止删除系统管理员账户！"), 403)
		return
	}

	if utils.In.Array(this.meta.user(ctx).Id, ids) {
		this.json(ctx, nil, facade.Lang(ctx, "不能删除自己！"), 400)
		return
	}

	item := facade.DB.Model(&table).WithTrashed()

	// 得到允许操作的 id 数组
	columnData, _ := item.WhereIn("id", ids).Column("id")
	ids = utils.Unity.Ids(columnData)

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 真实删除
	_, err := item.Force().Delete(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// clear 清空回收站
func (this *Users) clear(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}

	item := facade.DB.Model(&table).OnlyTrashed()

	// 检查回收站中是否包含系统管理员的账户
	hasAdmin, _ := item.Where("id", 1).Exist()
	if hasAdmin {
		this.json(ctx, nil, facade.Lang(ctx, "回收站中包含系统管理员账户，禁止清空！"), 403)
		return
	}

	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 找到所有软删除的数据
	_, err := item.Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
}

// restore 恢复数据
func (this *Users) restore(ctx *gin.Context) {

	// 表数据结构体
	table := model.Users{}
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
	columnData, _ := item.Column("id")
	ids = utils.Unity.Ids(columnData)

	// 无可操作数据
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 还原数据
	_, err := facade.DB.Model(&table).OnlyTrashed().Restore(ids)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}

// email 修改邮箱
func (this *Users) email(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)
	var err error

	if utils.Is.Empty(params["email"]) {
		this.json(ctx, nil, facade.Lang(ctx, "邮箱不能为空！"), 400)
		return
	}

	user := this.meta.user(ctx)
	// 即便中间件已经校验过登录了，这里还进行二次校验是未了防止接口权限被改，而 uid 又是强制的，从而导致的意外情况
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 驱动
	drive := cast.ToString(facade.SMSToml.Get("drive.email"))

	if utils.Is.Empty(drive) {
		this.json(ctx, nil, facade.Lang(ctx, "管理员未开启邮箱服务，无法发送验证码！"), 400)
		return
	}

	// 从数据库里面找一下这个邮箱是否已经存在
	exist, _ := facade.DB.Model(&model.Users{}).Where("email", params["email"]).Where("id", "!=", user.Id).Exist()
	if exist {
		this.json(ctx, nil, facade.Lang(ctx, "该邮箱已绑定其它账号！"), 400)
		return
	}

	// 缓存名称
	cacheName := fmt.Sprintf("%v-%v", drive, params["email"])

	// 验证码为空，发送验证码
	if utils.Is.Empty(params["code"]) {
		// 本地频控检查
		frequencyCacheName := fmt.Sprintf("frequency-%v-%v", drive, params["email"])
		dailyLimitCacheName := fmt.Sprintf("daily-limit-%v-%v", drive, params["email"])

		// 检查发送间隔（60秒）
		lastSendTime := facade.Cache.Get(frequencyCacheName)
		if !utils.Is.Empty(lastSendTime) {
			if time.Now().Unix()-cast.ToInt64(lastSendTime) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		// 检查每日发送限制（10次）
		dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
		if dailyCount >= 10 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		sms := facade.NewSMS(drive).VerifyCode(params["email"])
		if sms.Error != nil {
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}

		// 缓存验证码 - 5分钟
		go facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
		// 缓存发送时间 - 60秒
		go facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*60)
		// 缓存每日发送次数 - 24小时
		go facade.Cache.Set(dailyLimitCacheName, dailyCount+1, time.Hour*24)

		msg := fmt.Sprintf("验证码发送至您的邮箱：%s，请注意查收！", facade.Comm.MaskEmail(cast.ToString(params["email"])))
		this.json(ctx, nil, facade.Lang(ctx, msg), 201)
		return
	}

	// 获取缓存里面的验证码
	cacheCode := facade.Cache.Get(cacheName)

	if cast.ToString(params["code"]) != cast.ToString(cacheCode) {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	// 更新邮箱
	_, err = facade.DB.Model(&model.Users{}).Where("id", user.Id).UpdateColumn("email", params["email"])
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 删除验证码
	go facade.Cache.Del(cacheName)

	this.json(ctx, gin.H{"id": user.Id}, facade.Lang(ctx, "修改成功！"), 200)
}

// phone 修改手机号
func (this *Users) phone(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)
	var err error

	if utils.Is.Empty(params["phone"]) {
		this.json(ctx, nil, facade.Lang(ctx, "手机号不能为空！"), 400)
		return
	}

	user := this.meta.user(ctx)
	// 即便中间件已经校验过登录了，这里还进行二次校验是未了防止接口权限被改，而 uid 又是强制的，从而导致的意外情况
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 驱动
	drive := cast.ToString(facade.SMSToml.Get("drive.sms"))

	if utils.Is.Empty(drive) {
		this.json(ctx, nil, facade.Lang(ctx, "管理员未开启短信服务，无法发送验证码！"), 400)
		return
	}

	// 从数据库里面找一下这个手机号是否已经存在
	exist, _ := facade.DB.Model(&model.Users{}).Where("phone", params["phone"]).Where("id", "!=", user.Id).Exist()
	if exist {
		this.json(ctx, nil, facade.Lang(ctx, "该手机号已绑定其它账号！"), 400)
		return
	}

	// 缓存名称
	cacheName := fmt.Sprintf("%v-%v", drive, params["phone"])

	// 验证码为空，发送验证码
	if utils.Is.Empty(params["code"]) {
		// 本地频控检查
		frequencyCacheName := fmt.Sprintf("frequency-%v-%v", drive, params["phone"])
		dailyLimitCacheName := fmt.Sprintf("daily-limit-%v-%v", drive, params["phone"])

		// 检查发送间隔（60秒）
		lastSendTime := facade.Cache.Get(frequencyCacheName)
		if !utils.Is.Empty(lastSendTime) {
			if time.Now().Unix()-cast.ToInt64(lastSendTime) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		// 检查每日发送限制（10次）
		dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
		if dailyCount >= 10 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		sms := facade.NewSMS(drive).VerifyCode(params["phone"])
		if sms.Error != nil {
			// 处理阿里云频控错误
			if strings.Contains(sms.Error.Error(), "check frequency failed") || strings.Contains(sms.Error.Error(), "FREQUENCY_FAIL") {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请稍后再试！"), 400)
				return
			}
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}

		// 缓存验证码 - 5分钟
		go facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
		// 缓存发送时间 - 60秒
		go facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*60)
		// 缓存每日发送次数 - 24小时
		go facade.Cache.Set(dailyLimitCacheName, dailyCount+1, time.Hour*24)

		msg := fmt.Sprintf("验证码发送至您的手机：%s，请注意查收！", facade.Comm.MaskPhone(cast.ToString(params["phone"])))
		this.json(ctx, nil, facade.Lang(ctx, msg), 201)
		return
	}

	// 获取缓存里面的验证码
	cacheCode := facade.Cache.Get(cacheName)

	if cast.ToString(params["code"]) != cast.ToString(cacheCode) {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	// 更新手机号
	_, err = facade.DB.Model(&model.Users{}).Where("id", user.Id).UpdateColumn("phone", params["phone"])
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 删除验证码
	go facade.Cache.Del(cacheName)

	this.json(ctx, gin.H{"id": user.Id}, facade.Lang(ctx, "修改成功！"), 200)
}

// 注销 - 邮箱、手机号
func (this *Users) destroy(ctx *gin.Context) {

	table := model.Users{}
	var err error
	params := this.params(ctx, map[string]any{
		"source": "default",
	})

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	if user.Id == 1 {
		this.json(ctx, nil, facade.Lang(ctx, "禁止注销系统管理员账户！"), 403)
		return
	}

	var social string
	social = utils.Ternary(utils.Is.Email(user.Email), "email", social)
	social = utils.Ternary(utils.Is.Phone(user.Phone), "phone", social)

	if utils.Is.Empty(social) {
		this.json(ctx, nil, facade.Lang(ctx, "您未绑定手机或邮箱，无法验证注销安全性！"), 400)
		return
	}

	var contact string
	if social == "email" {
		contact = user.Email
	} else {
		contact = user.Phone
	}

	cacheName := fmt.Sprintf("[login][%v=%v]", social, contact)

	if utils.Is.Empty(params["code"]) {
		drive := utils.Ternary(social == "email", "email", "sms")

		frequencyCacheName := fmt.Sprintf("destroy-frequency-%v-%v", drive, contact)
		dailyLimitCacheName := fmt.Sprintf("destroy-daily-limit-%v-%v", drive, contact)

		lastSendTime := facade.Cache.Get(frequencyCacheName)
		if !utils.Is.Empty(lastSendTime) {
			if time.Now().Unix()-cast.ToInt64(lastSendTime) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		dailyCount := cast.ToInt(facade.Cache.Get(dailyLimitCacheName))
		if dailyCount >= 10 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		sms := facade.NewSMS(drive).VerifyCode(contact)
		if sms.Error != nil {
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}

		go facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
		go facade.Cache.Set(frequencyCacheName, time.Now().Unix(), time.Second*60)
		go facade.Cache.Set(dailyLimitCacheName, dailyCount+1, time.Hour*24)
		this.json(ctx, nil, facade.Lang(ctx, "验证码发送成功！"), 201)
		return
	}

	cacheCode := facade.Cache.Get(cacheName)
	if cast.ToString(params["code"]) != cast.ToString(cacheCode) {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	go facade.Cache.Del(cacheName)

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "请输入当前密码以确认注销！"), 400)
		return
	}

	userRecord, _ := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
	if utils.Is.Empty(userRecord) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	userPassword := cast.ToString(userRecord["password"])
	if utils.Password.Verify(userPassword, params["password"]) == false {
		this.json(ctx, nil, facade.Lang(ctx, "密码错误！"), 400)
		return
	}

	(&model.Users{}).Destroy(user.Id)

	randomPassword := utils.Rand.String(32, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	randomPasswordHash := utils.Password.Create(randomPassword)
	_, err = facade.DB.Model(&model.Users{}).Where("id", user.Id).UpdateColumn("password", randomPasswordHash)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	facade.Cache.Del(fmt.Sprintf("user[%v]", user.Id))

	_, err = facade.DB.Model(&table).Force().Delete(user.Id)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	ctx.SetCookie(cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN")), "", -1, "/", "", false, false)

	facade.Log.Info(map[string]any{"user_id": user.Id, "email": user.Email, "phone": user.Phone}, "用户注销账户")

	this.json(ctx, nil, facade.Lang(ctx, "注销成功！"), 200)
}

// ========================= 封禁系统 =========================

// ban 管理员封禁用户
func (this *Users) ban(ctx *gin.Context) {

	params := this.params(ctx)

	// 权限检查 - 仅管理员
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	if utils.Is.Empty(params["uid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	uid := cast.ToInt(params["uid"])
	if uid == 1 {
		this.json(ctx, nil, facade.Lang(ctx, "禁止封禁系统管理员！"), 403)
		return
	}

	// 检查目标用户是否存在
	targetUser, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(targetUser) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}
	targetMap := cast.ToStringMap(targetUser)

	// 检查是否已在封禁中
	if cast.ToInt(targetMap["current_ban_id"]) > 0 {
		existRecord, _ := facade.DB.Model(&model.UserBanRecords{}).Find(cast.ToInt(targetMap["current_ban_id"]))
		if !utils.Is.Empty(existRecord) {
			if cast.ToInt(cast.ToStringMap(existRecord)["status"]) == model.BanStatusActive {
				this.json(ctx, nil, facade.Lang(ctx, "该用户已在封禁中，请先解封后再操作！"), 400)
				return
			}
		}
	}

	// 封禁类型位掩码，默认全封禁
	banType := cast.ToInt(params["ban_type"])
	if banType <= 0 || banType > 31 {
		banType = model.BanTypeAll
	}

	// 时长：0=永久，其他按天数；若未提供且非永久，启用手动指定或自动梯度
	duration := cast.ToInt(params["duration"])
	autoGradient := cast.ToBool(params["auto_gradient"])

	if duration == 0 && !autoGradient && !utils.Is.Empty(params["duration"]) {
		// 明确指定了 0 = 永久
	} else if autoGradient || utils.Is.Empty(params["duration"]) {
		// 自动梯度封禁：根据累计封禁次数决定时长
		// 首次1天 二次7天 三次15天 四次30天 五次及以上永久
		banCount := cast.ToInt(targetMap["ban_count"])
		switch banCount {
		case 0:
			duration = 1 // 首次1天
		case 1:
			duration = 7 // 二次7天
		case 2:
			duration = 15 // 三次15天
		case 3:
			duration = 30 // 四次30天
		default:
			duration = 0 // 五次及以上永久
		}
	}

	// 封禁原因
	reason := cast.ToString(params["reason"])
	if utils.Is.Empty(reason) {
		reason = "违反社区规定"
	}

	// 证据
	evidence := cast.ToString(params["evidence"])

	// 操作人信息
	operator := this.user(ctx)
	operatorIp := ctx.ClientIP()
	operatorUa := ctx.Request.UserAgent()

	now := time.Now().Unix()
	violationNum := cast.ToInt(targetMap["ban_count"]) + 1

	// 计算过期时间
	var expiresAt int64
	if duration > 0 {
		expiresAt = now + int64(duration)*86400
	}

	// 创建封禁记录
	record := model.UserBanRecords{
		Uid:          uid,
		OperatorId:   operator.Id,
		BanType:      banType,
		Reason:       reason,
		Evidence:     evidence,
		Duration:     duration,
		BanTime:      now,
		ExpiresAt:    expiresAt,
		ViolationNum: violationNum,
		Status:       model.BanStatusActive,
		OperatorIp:   operatorIp,
		OperatorUa:   operatorUa,
		CreateTime:   now,
		UpdateTime:   now,
	}

	// 事务：创建记录 + 更新用户状态
	_, err := facade.DB.Model(&record).Create(&record)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 更新用户封禁信息
	tokenName := cast.ToString(facade.AppToml.Get("app.token_name", "INIS_LOGIN_TOKEN"))
	_, err = facade.DB.Model(&model.Users{}).Where("id", uid).Update(map[string]any{
		"ban_count":      violationNum,
		"current_ban_id": record.Id,
		"last_ban_at":    now,
		"restrictions":   banType,
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 清除目标用户的缓存，强制下线
	facade.Cache.Del(fmt.Sprintf("user[%v]", uid))
	facade.Cache.DelPrefix(fmt.Sprintf("[token][%v]", tokenName))

	// 删除列表缓存
	go this.delCache()

	// 审计日志
	facade.Log.Info(map[string]any{
		"uid":           uid,
		"operator_id":   operator.Id,
		"ban_type":      banType,
		"reason":        reason,
		"duration":      duration,
		"expires_at":    expiresAt,
		"violation_num": violationNum,
		"operator_ip":   operatorIp,
		"operator_ua":   operatorUa,
	}, "管理员封禁用户")

	this.json(ctx, gin.H{"id": record.Id}, facade.Lang(ctx, "封禁成功！"), 200)
}

// unban 管理员解封用户
func (this *Users) unban(ctx *gin.Context) {

	params := this.params(ctx)

	// 权限检查 - 仅管理员
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	uid := cast.ToInt(params["uid"])
	recordId := cast.ToInt(params["record_id"])

	if uid == 0 && recordId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid 或 record_id"), 400)
		return
	}

	// 如果提供了 record_id，直接查找记录
	if recordId > 0 {
		this.unbanByRecordId(ctx, recordId)
		return
	}

	// 根据 uid 查找当前生效的封禁记录
	targetUser, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(targetUser) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}
	targetMap := cast.ToStringMap(targetUser)
	currentBanId := cast.ToInt(targetMap["current_ban_id"])

	if currentBanId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该用户当前未被封禁！"), 400)
		return
	}

	this.unbanByRecordId(ctx, currentBanId)
}

// unbanByRecordId 根据封禁记录ID解封
func (this *Users) unbanByRecordId(ctx *gin.Context, recordId int) {

	operator := this.user(ctx)
	now := time.Now().Unix()

	// 查找封禁记录
	record, _ := facade.DB.Model(&model.UserBanRecords{}).Find(recordId)
	if utils.Is.Empty(record) {
		this.json(ctx, nil, facade.Lang(ctx, "封禁记录不存在！"), 404)
		return
	}
	recordMap := cast.ToStringMap(record)

	if cast.ToInt(recordMap["status"]) != model.BanStatusActive {
		this.json(ctx, nil, facade.Lang(ctx, "该封禁记录非生效中状态！"), 400)
		return
	}

	uid := cast.ToInt(recordMap["uid"])

	// 更新封禁记录状态
	_, err := facade.DB.Model(&model.UserBanRecords{}).Where("id", recordId).Update(map[string]any{
		"status":     model.BanStatusRevoked,
		"unban_time": now,
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 恢复用户状态
	_, err = facade.DB.Model(&model.Users{}).Where("id", uid).Update(map[string]any{
		"current_ban_id": 0,
		"restrictions":   0,
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 清除用户缓存
	facade.Cache.Del(fmt.Sprintf("user[%v]", uid))
	go this.delCache()

	// 审计日志
	facade.Log.Info(map[string]any{
		"uid":         uid,
		"record_id":   recordId,
		"operator_id": operator.Id,
		"operator_ip": ctx.ClientIP(),
		"operator_ua": ctx.Request.UserAgent(),
	}, "管理员解封用户")

	this.json(ctx, gin.H{"id": recordId}, facade.Lang(ctx, "解封成功！"), 200)
}

// blackroom 小黑屋公示 - 公开接口
func (this *Users) blackroom(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	var records []model.UserBanRecords
	mold := facade.DB.Model(&records)
	mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	cacheName := this.cache.name(ctx)
	if this.cache.enable(ctx) && facade.Cache.Has(cacheName) {
		msg[1] = "（来自缓存）"
		data = facade.Cache.Get(cacheName)
	} else {
		count, _ := mold.Count()
		items, _ := mold.Order(params["order"]).Limit(limit).Page(page).Select()

		// 脱敏处理：用户昵称脱敏
		dataList := cast.ToSlice(items)
		for i, val := range dataList {
			itemMap := cast.ToStringMap(val)
			// 对封禁用户昵称脱敏
			if result, ok := itemMap["result"].(map[string]any); ok {
				if user, ok := result["user"].(map[string]any); ok {
					if nickname, ok := user["nickname"].(string); ok && nickname != "" {
						runes := []rune(nickname)
						if len(runes) > 2 {
							user["nickname"] = string(runes[0]) + "***" + string(runes[len(runes)-1])
						} else if len(runes) > 1 {
							user["nickname"] = string(runes[0]) + "*"
						}
						result["user"] = user
						itemMap["result"] = result
						dataList[i] = itemMap
					}
				}
			}
		}

		data = gin.H{
			"data":  dataList,
			"count": count,
			"page":  math.Ceil(float64(count) / float64(limit)),
		}

		if this.cache.enable(ctx) {
			go facade.Cache.Set(cacheName, data)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	// 如果 data 是 gin.H，直接返回
	if h, ok := data.(gin.H); ok {
		this.json(ctx, h, facade.Lang(ctx, strings.Join(msg, "")), code)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// appeal 用户申诉
func (this *Users) appeal(ctx *gin.Context) {

	params := this.params(ctx)
	user := this.user(ctx)

	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	recordId := cast.ToInt(params["record_id"])
	appealContent := cast.ToString(params["content"])

	if recordId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "record_id"), 400)
		return
	}

	if utils.Is.Empty(appealContent) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "申诉内容"), 400)
		return
	}

	// 检查封禁记录是否存在且属于当前用户
	record, _ := facade.DB.Model(&model.UserBanRecords{}).Find(recordId)
	if utils.Is.Empty(record) {
		this.json(ctx, nil, facade.Lang(ctx, "封禁记录不存在！"), 404)
		return
	}
	recordMap := cast.ToStringMap(record)

	if cast.ToInt(recordMap["uid"]) != user.Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权操作此封禁记录！"), 403)
		return
	}

	// 只有生效中的封禁才能申诉
	if cast.ToInt(recordMap["status"]) != model.BanStatusActive {
		this.json(ctx, nil, facade.Lang(ctx, "当前封禁状态不允许申诉！"), 400)
		return
	}

	// 五次及以上违规（永久封禁）禁止申诉
	if cast.ToInt(recordMap["violation_num"]) >= 5 {
		this.json(ctx, nil, facade.Lang(ctx, "该封禁为永久封禁且违规次数已达上限，禁止申诉！"), 403)
		return
	}

	// 更新封禁记录为申诉中
	now := time.Now().Unix()
	_, err := facade.DB.Model(&model.UserBanRecords{}).Where("id", recordId).Update(map[string]any{
		"status":         model.BanStatusAppealed,
		"appeal_content": appealContent,
		"appeal_time":    now,
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 清除缓存
	facade.Cache.Del(fmt.Sprintf("user[%v]", user.Id))
	go this.delCache()

	// 审计日志
	facade.Log.Info(map[string]any{
		"uid":       user.Id,
		"record_id": recordId,
		"content":   appealContent,
	}, "用户提交封禁申诉")

	this.json(ctx, gin.H{"id": recordId}, facade.Lang(ctx, "申诉已提交，请耐心等待管理员审核！"), 200)
}

// appealHandle 管理员处理申诉（通过或驳回）
func (this *Users) appealHandle(ctx *gin.Context) {

	params := this.params(ctx)

	// 权限检查 - 仅管理员
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	recordId := cast.ToInt(params["record_id"])
	action := cast.ToString(params["action"]) // "approve" | "reject"
	reply := cast.ToString(params["reply"])   // 申诉回复（驳回时必填）

	if recordId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "record_id"), 400)
		return
	}

	if action != "approve" && action != "reject" {
		this.json(ctx, nil, facade.Lang(ctx, "action 必须为 approve 或 reject！"), 400)
		return
	}

	if action == "reject" && utils.Is.Empty(reply) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "驳回理由"), 400)
		return
	}

	// 查找封禁记录
	record, _ := facade.DB.Model(&model.UserBanRecords{}).Find(recordId)
	if utils.Is.Empty(record) {
		this.json(ctx, nil, facade.Lang(ctx, "封禁记录不存在！"), 404)
		return
	}
	recordMap := cast.ToStringMap(record)

	// 只有申诉中状态才能处理
	if cast.ToInt(recordMap["status"]) != model.BanStatusAppealed {
		this.json(ctx, nil, facade.Lang(ctx, "当前封禁记录非申诉中状态！"), 400)
		return
	}

	now := time.Now().Unix()
	operator := this.user(ctx)
	uid := cast.ToInt(recordMap["uid"])

	if action == "approve" {
		// 申诉通过：更新记录状态 + 恢复用户
		_, err := facade.DB.Model(&model.UserBanRecords{}).Where("id", recordId).Update(map[string]any{
			"status":            model.BanStatusAppealApproved,
			"unban_time":        now,
			"appeal_reply":      reply,
			"appeal_reply_time": now,
		})
		if err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}

		// 恢复用户状态
		_, err = facade.DB.Model(&model.Users{}).Where("id", uid).Update(map[string]any{
			"current_ban_id": 0,
			"restrictions":   0,
		})
		if err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}

		// 清除缓存
		facade.Cache.Del(fmt.Sprintf("user[%v]", uid))
		go this.delCache()

		// 审计日志
		facade.Log.Info(map[string]any{
			"uid":         uid,
			"record_id":   recordId,
			"operator_id": operator.Id,
			"action":      "approve",
			"reply":       reply,
		}, "管理员通过申诉")

		this.json(ctx, gin.H{"id": recordId}, facade.Lang(ctx, "申诉已通过，用户已解封！"), 200)

	} else {
		// 申诉驳回：仅更新记录状态，封禁继续生效
		_, err := facade.DB.Model(&model.UserBanRecords{}).Where("id", recordId).Update(map[string]any{
			"status":            model.BanStatusAppealRejected,
			"appeal_reply":      reply,
			"appeal_reply_time": now,
		})
		if err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}

		// 审计日志
		facade.Log.Info(map[string]any{
			"uid":         uid,
			"record_id":   recordId,
			"operator_id": operator.Id,
			"action":      "reject",
			"reply":       reply,
		}, "管理员驳回申诉")

		this.json(ctx, gin.H{"id": recordId}, facade.Lang(ctx, "申诉已驳回！"), 200)
	}
}

// appealPublic 公开申诉接口（被封禁用户无需登录即可申诉）
// 两阶段流程：
//   1. 不传 code：发送验证码到账号绑定的邮箱/手机号
//   2. 传 code + content：验证验证码并提交申诉
func (this *Users) appealPublic(ctx *gin.Context) {

	params := this.params(ctx)
	account := cast.ToString(params["account"])

	if utils.Is.Empty(account) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "账号"), 400)
		return
	}

	// 通过账号查找用户（支持邮箱、手机号、账号名）
	table := model.Users{}
	item, _ := facade.DB.Model(&table).Or([]any{
		[]any{"email", "=", account},
		[]any{"phone", "=", account},
		[]any{"account", "=", account},
	}).Find()

	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "账号与封禁记录不匹配！"), 403)
		return
	}
	userMap := cast.ToStringMap(item)
	uid := cast.ToInt(userMap["id"])

	// 查找用户当前生效的封禁记录
	if cast.ToInt(userMap["current_ban_id"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该账号当前未被封禁！"), 400)
		return
	}

	banRecord, _ := facade.DB.Model(&model.UserBanRecords{}).Find(cast.ToInt(userMap["current_ban_id"]))
	if utils.Is.Empty(banRecord) {
		this.json(ctx, nil, facade.Lang(ctx, "封禁记录不存在！"), 404)
		return
	}
	banMap := cast.ToStringMap(banRecord)
	recordId := cast.ToInt(banMap["id"])

	// 只有生效中的封禁才能申诉
	if cast.ToInt(banMap["status"]) != model.BanStatusActive {
		this.json(ctx, nil, facade.Lang(ctx, "当前封禁状态不允许申诉！"), 400)
		return
	}

	// 五次及以上禁止申诉
	if cast.ToInt(banMap["violation_num"]) >= 5 {
		this.json(ctx, nil, facade.Lang(ctx, "该封禁为永久封禁且违规次数已达上限，禁止申诉！"), 403)
		return
	}

	// 确定联系方式（优先邮箱，其次手机号）
	var social string
	var contact string
	userEmail := cast.ToString(userMap["email"])
	userPhone := cast.ToString(userMap["phone"])
	if utils.Is.Email(userEmail) {
		social = "email"
		contact = userEmail
	} else if utils.Is.Phone(userPhone) {
		social = "phone"
		contact = userPhone
	} else {
		this.json(ctx, nil, facade.Lang(ctx, "该账号未绑定邮箱或手机号，无法验证身份！"), 400)
		return
	}

	drive := utils.Ternary(social == "email", "email", "sms")
	drives := cast.ToStringMap(facade.SMSToml.Get("drive"))
	if utils.Is.Empty(drives[drive]) {
		this.json(ctx, nil, facade.Lang(ctx, "管理员未开启%v服务，无法发送验证码！", utils.Ternary(social == "email", "邮箱", "短信")), 400)
		return
	}

	clientIP := ctx.ClientIP()

	// 缓存键名
	cacheName := fmt.Sprintf("[appeal][%v=%v]", social, contact)
	freqAccount := fmt.Sprintf("appeal-freq-account-%v", contact)
	freqIP := fmt.Sprintf("appeal-freq-ip-%v", clientIP)
	dailyAccount := fmt.Sprintf("appeal-daily-account-%v", contact)
	dailyIP := fmt.Sprintf("appeal-daily-ip-%v", clientIP)
	submitAccount := fmt.Sprintf("appeal-submit-account-%v", contact)
	submitIP := fmt.Sprintf("appeal-submit-ip-%v", clientIP)
	codeErrKey := fmt.Sprintf("appeal-code-err-%v", contact)

	// ========== 阶段一：发送验证码 ==========
	if utils.Is.Empty(params["code"]) {

		// 账号维度发送间隔（60秒）
		if lastSend := facade.Cache.Get(freqAccount); !utils.Is.Empty(lastSend) {
			if time.Now().Unix()-cast.ToInt64(lastSend) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		// IP 维度发送间隔（60秒）
		if lastSend := facade.Cache.Get(freqIP); !utils.Is.Empty(lastSend) {
			if time.Now().Unix()-cast.ToInt64(lastSend) < 60 {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请60秒后再试！"), 400)
				return
			}
		}

		// 账号每日发送上限（5次）
		if cast.ToInt(facade.Cache.Get(dailyAccount)) >= 5 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		// IP 每日发送上限（10次）
		if cast.ToInt(facade.Cache.Get(dailyIP)) >= 10 {
			this.json(ctx, nil, facade.Lang(ctx, "今日发送验证码次数已达上限，请明日再试！"), 400)
			return
		}

		sms := facade.NewSMS(drives[drive]).VerifyCode(contact)
		if sms.Error != nil {
			if drive == "sms" && (strings.Contains(sms.Error.Error(), "check frequency failed") || strings.Contains(sms.Error.Error(), "FREQUENCY_FAIL")) {
				this.json(ctx, nil, facade.Lang(ctx, "发送过于频繁，请稍后再试！"), 400)
				return
			}
			this.json(ctx, nil, sms.Error.Error(), 400)
			return
		}

		// 缓存验证码 - 5分钟
		facade.Cache.Set(cacheName, sms.VerifyCode, 5*time.Minute)
		// 清除验证码错误计数
		facade.Cache.Del(codeErrKey)
		// 发送间隔 - 60秒
		go facade.Cache.Set(freqAccount, time.Now().Unix(), time.Second*60)
		go facade.Cache.Set(freqIP, time.Now().Unix(), time.Second*60)
		// 每日计数 - 24小时
		go facade.Cache.Set(dailyAccount, cast.ToInt(facade.Cache.Get(dailyAccount))+1, time.Hour*24)
		go facade.Cache.Set(dailyIP, cast.ToInt(facade.Cache.Get(dailyIP))+1, time.Hour*24)

		var contactMasked string
		if social == "email" {
			contactMasked = facade.Comm.MaskEmail(contact)
		} else {
			contactMasked = facade.Comm.MaskPhone(contact)
		}
		msg := fmt.Sprintf("验证码已发送至您的%v：%s，请注意查收！",
			utils.Ternary(social == "email", "邮箱", "手机"), contactMasked)
		this.json(ctx, gin.H{
			"contact_type":   social,
			"contact_masked": contactMasked,
		}, facade.Lang(ctx, msg), 201)
		return
	}

	// ========== 阶段二：验证并提交申诉 ==========
	appealContent := cast.ToString(params["content"])
	if utils.Is.Empty(appealContent) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "申诉内容"), 400)
		return
	}

	// 验证码错误次数上限（5次后需重新发送）
	if cast.ToInt(facade.Cache.Get(codeErrKey)) >= 5 {
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误次数过多，请重新发送验证码！"), 400)
		return
	}

	// 验证验证码
	cacheCode := facade.Cache.Get(cacheName)
	if cast.ToString(params["code"]) != cast.ToString(cacheCode) {
		go facade.Cache.Set(codeErrKey, cast.ToInt(facade.Cache.Get(codeErrKey))+1, 5*time.Minute)
		this.json(ctx, nil, facade.Lang(ctx, "验证码错误！"), 400)
		return
	}

	// 提交频率限制（账号维度，每小时3次）
	if cast.ToInt(facade.Cache.Get(submitAccount)) >= 3 {
		this.json(ctx, nil, facade.Lang(ctx, "申诉提交过于频繁，请稍后再试！"), 400)
		return
	}

	// 提交频率限制（IP维度，每小时5次）
	if cast.ToInt(facade.Cache.Get(submitIP)) >= 5 {
		this.json(ctx, nil, facade.Lang(ctx, "申诉提交过于频繁，请稍后再试！"), 400)
		return
	}

	// 通过 status 条件更新限制每条封禁记录仅可申诉一次
	now := time.Now().Unix()
	tx, err := facade.DB.Model(&model.UserBanRecords{}).
		Where("id", recordId).
		Where("status", model.BanStatusActive).
		Update(map[string]any{
			"status":         model.BanStatusAppealed,
			"appeal_content": appealContent,
			"appeal_time":    now,
		})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}
	if tx.RowsAffected == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该封禁记录已被处理，无法重复申诉！"), 400)
		return
	}

	// 清理验证码相关缓存
	go facade.Cache.Del(cacheName)
	go facade.Cache.Del(codeErrKey)

	// 更新提交频率
	go facade.Cache.Set(submitAccount, cast.ToInt(facade.Cache.Get(submitAccount))+1, time.Hour)
	go facade.Cache.Set(submitIP, cast.ToInt(facade.Cache.Get(submitIP))+1, time.Hour)

	// 清除用户缓存
	facade.Cache.Del(fmt.Sprintf("user[%v]", uid))
	go this.delCache()

	// 审计日志
	facade.Log.Info(map[string]any{
		"uid":       uid,
		"record_id": recordId,
		"account":   account,
		"content":   appealContent,
		"ip":        clientIP,
	}, "被封禁用户通过公开接口提交申诉")

	this.json(ctx, gin.H{"id": recordId}, facade.Lang(ctx, "申诉已提交，请耐心等待管理员审核！"), 200)
}
