package controller

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"inis/app/validator"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Comment struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Comment) IGET(ctx *gin.Context) {

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
		"flat":   this.flat,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Comment) IPOST(ctx *gin.Context) {

	// 转小写
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

	// 删除缓存
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

// IPUT - PUT请求本体
func (this *Comment) IPUT(ctx *gin.Context) {
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

// IDEL - DELETE请求本体
func (this *Comment) IDEL(ctx *gin.Context) {
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "删除缓存协程发生错误")
			}
		}()
		this.delCache()
	}()
}

// INDEX - GET请求本体
func (this *Comment) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// 删除缓存
func (this *Comment) delCache() {
	// 删除缓存
	facade.Cache.DelTags([]any{"[GET]", "comment"})
}

// one 获取指定数据
func (this *Comment) one(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx)

	// 表数据结构体
	table := model.Comment{}
	// 允许查询的字段
	allow := []any{"id"}
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

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// all 获取全部数据
func (this *Comment) all(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	// 表数据结构体
	table := model.Comment{}
	// 允许查询的字段
	allow := []any{"pid", "bind_id", "bind_type", "editor"}
	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment
	mold := facade.DB.Model(&result).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
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
func (this *Comment) rand(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 限制最大数量
	limit := this.meta.limit(ctx)

	// 排除的 id 列表
	except := utils.Unity.Ids(params["except"])

	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	// 直接使用数据库的随机查询功能，提高性能
	mold := facade.DB.Model(&[]model.Comment{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		mold.Where("id", "NOT IN", except)
	}
	mold.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	// 使用 ORDER BY RAND() 进行随机查询
	item := mold.Order("RAND()").Limit(limit).Select()

	// 排除字段
	data := utils.ArrayMapWithField(item, params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

// save 保存数据 - 包含创建和更新
func (this *Comment) save(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

// create 创建数据
func (this *Comment) create(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"bind_type": "article",
	})
	// 验证器
	err := validator.NewValid("comment", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bind_id"), 400)
		return
	}

	user := this.meta.user(ctx)
	// 即便中间件已经校验过登录了，这里还进行二次校验是未了防止接口权限被改，而 uid 又是强制的，从而导致的意外情况
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 评论配置
	var comment map[string]any

	// 从数据库里面找一下存不存在这个类型的数据
	switch params["bind_type"] {
	case "article":
		article := facade.DB.Model(&model.Article{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(article) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的文章！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(article["json"])["comment"])
	case "page":
		page := facade.DB.Model(&model.Pages{}).Where("id", params["bind_id"]).Find()
		if utils.Is.Empty(page) {
			this.json(ctx, nil, facade.Lang(ctx, "不存在的页面！"), 400)
			return
		}
		comment = cast.ToStringMap(cast.ToStringMap(page["json"])["comment"])
	default:
		// 对于其他类型，使用全局评论配置
		comment = this.config("comment")
	}

	// 允许评论选项继承了父级配置
	if cast.ToInt(comment["allow"]) == 0 {
		comment["allow"] = this.config("comment")["allow"]
	}

	// 评论开关
	if cast.ToInt(comment["allow"]) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "评论功能已关闭！"), 400)
		return
	}

	// 获取评论配置
	commentConfig := this.config("comment")

	// 1. 评论速率限制
	rateLimit := cast.ToStringMap(commentConfig["rate_limit"])
	if cast.ToInt(rateLimit["enabled"]) == 1 {
		maxCount := cast.ToInt(rateLimit["max_count"])
		timeWindow := cast.ToInt(rateLimit["time_window"])

		// 缓存键：用户ID + IP + 当前时间窗口
		now := time.Now().Unix()
		timeWindowKey := now / int64(timeWindow)
		cacheKey := fmt.Sprintf("comment:rate_limit:%d:%s:%d", user.Id, ctx.ClientIP(), timeWindowKey)

		// 获取当前评论次数
		currentCount := 0
		if facade.Cache.Has(cacheKey) {
			currentCount = cast.ToInt(facade.Cache.Get(cacheKey))
		}

		if currentCount >= maxCount {
			this.json(ctx, nil, facade.Lang(ctx, "评论过于频繁，请稍后再试！"), 429)
			return
		}

		// 增加评论次数并设置缓存
		facade.Cache.Set(cacheKey, currentCount+1, time.Duration(timeWindow)*time.Second)
	}

	// 2. 最大字数限制
	maxLength := cast.ToInt(commentConfig["max_length"])
	if maxLength > 0 && len(cast.ToString(params["content"])) > maxLength {
		this.json(ctx, nil, facade.Lang(ctx, "评论内容过长，最多允许%d个字符！", maxLength), 400)
		return
	}

	// 3. 评论必须包含中文
	if cast.ToInt(commentConfig["require_chinese"]) == 1 {
		hasChinese := false
		for _, r := range cast.ToString(params["content"]) {
			if unicode.Is(unicode.Scripts["Han"], r) {
				hasChinese = true
				break
			}
		}
		if !hasChinese {
			this.json(ctx, nil, facade.Lang(ctx, "评论内容必须包含中文！"), 400)
			return
		}
	}

	// 4. 敏感词过滤
	if cast.ToInt(commentConfig["sensitive_filter"]) == 1 {
		sensitiveWords := cast.ToStringSlice(commentConfig["sensitive_words"])
		content := cast.ToString(params["content"])

		for _, word := range sensitiveWords {
			if strings.Contains(content, word) {
				this.json(ctx, nil, facade.Lang(ctx, "评论内容包含敏感词，请修改后重试！"), 400)
				return
			}
		}
	}

	// 表数据结构体
	agent := this.header(ctx, "User-Agent")
	if len(agent) > 511 {
		agent = agent[:511] // 截断到511字符，防止超出数据库字段长度
	}
	table := model.Comment{
		Uid:        user.Id,
		Agent:      agent,
		Ip:         ctx.ClientIP(), // 使用真实的客户端IP，防止伪造
		CreateTime: time.Now().Unix(), UpdateTime: time.Now().Unix(),
	}
	allow := []any{"pid", "content", "bind_id", "bind_type", "editor", "json", "text"}

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

	// 更新用户经验
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "更新用户经验协程发生错误")
			}
		}()
		_ = (&model.EXP{}).Add(model.EXP{
			Uid:      user.Id,
			Type:     "comment",
			BindId:   table.BindId,
			BindType: table.BindType,
		})
	}()

	// 发送评论邮件通知
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "发送评论邮件通知协程发生错误")
			}
		}()

		// 获取评论配置
		commentConfig := this.config("comment")
		facade.Log.Info(map[string]any{"config": commentConfig}, "加载评论配置")

		emailNotify := cast.ToStringMap(commentConfig["email_notify"])
		facade.Log.Info(map[string]any{"email_notify": emailNotify}, "加载邮件通知配置")

		// 检查邮件通知是否开启
		if cast.ToInt(emailNotify["enabled"]) == 1 {
			facade.Log.Info(nil, "邮件通知功能已开启")

			// 重试机制参数
			retryCount := cast.ToInt(emailNotify["retry_count"])
			retryInterval := cast.ToInt(emailNotify["retry_interval"])
			facade.Log.Info(map[string]any{"retry_count": retryCount, "retry_interval": retryInterval}, "邮件发送重试配置")

			// 获取用户信息
			userInfo := facade.DB.Model(&model.Users{}).Where("id", user.Id).Find()
			userEmail := cast.ToString(cast.ToStringMap(userInfo)["email"])
			facade.Log.Info(map[string]any{"user_id": user.Id, "user_email": userEmail, "user_name": cast.ToString(cast.ToStringMap(userInfo)["name"])}, "获取评论用户信息")

			// 获取文章或页面标题
			title := ""
			switch table.BindType {
			case "article":
				article := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					title = cast.ToString(cast.ToStringMap(article)["title"])
				}
			case "page":
				page := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					title = cast.ToString(cast.ToStringMap(page)["title"])
				}
			}

			// 获取服务器本地时区的当前时间
			createdAt := time.Now().Format("2006-01-02 15:04:05")
			facade.Log.Info(map[string]any{"local_time": createdAt}, "获取服务器本地时间")

			// 评论信息
			commentInfo := map[string]any{
				"content":      table.Content,
				"created_at":   createdAt,
				"author_name":  cast.ToString(cast.ToStringMap(userInfo)["nickname"]),
				"author_email": userEmail,
				"ip":           table.Ip,
				"bind_type":    table.BindType,
				"bind_id":      table.BindId,
				"title":        title,
			}
			facade.Log.Info(map[string]any{"comment_id": table.Id, "bind_type": table.BindType, "bind_id": table.BindId}, "评论通知处理")

			// 发送邮件通知给文章作者
			var authorEmail string
			switch table.BindType {
			case "article":
				// 获取文章信息
				article := facade.DB.Model(&model.Article{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(article) {
					authorId := cast.ToInt(cast.ToStringMap(article)["uid"])
					// 获取作者信息
					author := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"article_id": table.BindId, "author_id": authorId, "author_email": authorEmail}, "获取文章作者信息")
				} else {
					facade.Log.Warn(map[string]any{"article_id": table.BindId}, "文章不存在，跳过发送给作者")
				}
			case "page":
				// 获取页面信息
				page := facade.DB.Model(&model.Pages{}).Where("id", table.BindId).Find()
				if !utils.Is.Empty(page) {
					authorId := cast.ToInt(cast.ToStringMap(page)["uid"])
					// 获取作者信息
					author := facade.DB.Model(&model.Users{}).Where("id", authorId).Find()
					authorEmail = cast.ToString(cast.ToStringMap(author)["email"])
					facade.Log.Info(map[string]any{"page_id": table.BindId, "author_id": authorId, "author_email": authorEmail}, "获取页面作者信息")
				} else {
					facade.Log.Warn(map[string]any{"page_id": table.BindId}, "页面不存在，跳过发送给作者")
				}
			default:
				facade.Log.Warn(map[string]any{"bind_type": table.BindType}, "未知绑定类型，跳过发送给作者")
			}

			// 发送通知给文章作者，避免重复通知
			if utils.Is.Email(authorEmail) && authorEmail != userEmail {
				facade.Log.Info(map[string]any{"recipient": authorEmail}, "开始发送邮件给作者")
				for i := 0; i <= retryCount; i++ {
					sms := facade.NewSMS("email")
					response := sms.SendCommentNotify(authorEmail, commentInfo)
					if response.Error == nil {
						facade.Log.Info(map[string]any{"recipient": authorEmail}, "邮件发送给作者成功")
						break
					}
					if i < retryCount {
						facade.Log.Warn(map[string]any{"error": response.Error, "retry": i + 1}, "邮件发送给作者失败，准备重试")
						time.Sleep(time.Duration(retryInterval) * time.Second)
					} else {
						facade.Log.Error(map[string]any{"error": response.Error, "recipient": authorEmail}, "发送文章作者邮件通知失败")
					}
				}
			} else {
				facade.Log.Warn(map[string]any{"author_email": authorEmail, "user_email": userEmail}, "作者邮箱无效或重复，跳过发送")
			}

			// 如果是回复评论，还需要通知被回复的用户
			if table.Pid > 0 {
				facade.Log.Info(map[string]any{"pid": table.Pid}, "检测到回复评论，准备通知被回复用户")
				// 获取父评论信息
				parentComment := facade.DB.Model(&model.Comment{}).Where("id", table.Pid).Find()
				if !utils.Is.Empty(parentComment) {
					parentUid := cast.ToInt(cast.ToStringMap(parentComment)["uid"])
					// 获取被回复用户信息
					parentUser := facade.DB.Model(&model.Users{}).Where("id", parentUid).Find()
					parentEmail := cast.ToString(cast.ToStringMap(parentUser)["email"])
					facade.Log.Info(map[string]any{"parent_comment_id": table.Pid, "parent_user_id": parentUid, "parent_email": parentEmail}, "获取被回复用户信息")

					if utils.Is.Email(parentEmail) && parentUid != user.Id {
						facade.Log.Info(map[string]any{"recipient": parentEmail}, "开始发送邮件给被回复用户")
						// 发送回复通知
						for i := 0; i <= retryCount; i++ {
							sms := facade.NewSMS("email")
							response := sms.SendReplyNotify(parentEmail, commentInfo)
							if response.Error == nil {
								facade.Log.Info(map[string]any{"recipient": parentEmail}, "邮件发送给被回复用户成功")
								break
							}
							if i < retryCount {
								facade.Log.Warn(map[string]any{"error": response.Error, "retry": i + 1}, "邮件发送给被回复用户失败，准备重试")
								time.Sleep(time.Duration(retryInterval) * time.Second)
							} else {
								facade.Log.Error(map[string]any{"error": response.Error, "recipient": parentEmail}, "发送评论回复邮件通知失败")
							}
						}
					} else {
						facade.Log.Warn(map[string]any{"parent_email": parentEmail, "parent_uid": parentUid, "user_id": user.Id}, "被回复用户邮箱无效或为评论者本人，跳过发送")
					}
				} else {
					facade.Log.Warn(map[string]any{"pid": table.Pid}, "父评论不存在，跳过发送给被回复用户")
				}
			} else {
				facade.Log.Info(map[string]any{"pid": table.Pid}, "不是回复评论，跳过发送给被回复用户")
			}
		} else {
			facade.Log.Info(nil, "邮件通知功能未开启")
		}
	}()

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update 更新数据
func (this *Comment) update(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	// 验证器
	err := validator.NewValid("comment", params)

	// 参数校验不通过
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 表数据结构体
	table := model.Comment{}
	allow := []any{"content", "editor", "json", "text"}
	async := utils.Async[map[string]any]()

	root := this.meta.root(ctx)

	// 越权 - 增加可选字段
	if root {
		allow = append(allow, "pid", "bind_id", "bind_type")
	}

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

	item := facade.DB.Model(&table).WithTrashed().Where("id", params["id"])

	// 越权 - 既没有管理权限，也不是自己的数据
	if !root && cast.ToInt(item.Find()["uid"]) != this.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	// 更新数据 - Scan() 解析结构体，防止 table 拿不到数据
	tx := item.Scan(&table).Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id}, facade.Lang(ctx, "更新成功！"), 200)
}

// count 统计数据
func (this *Comment) count(ctx *gin.Context) {

	// 表数据结构体
	table := model.Comment{}
	// 获取请求参数
	params := this.params(ctx)

	item := facade.DB.Model(&table).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
	item.IWhere(params["where"]).IOr(params["or"]).ILike(params["like"]).INot(params["not"]).INull(params["null"]).INotNull(params["notNull"])

	this.json(ctx, item.Count(), facade.Lang(ctx, "查询成功！"), 200)
}

// sum 求和
func (this *Comment) sum(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Comment
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
func (this *Comment) min(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Comment
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
func (this *Comment) max(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table model.Comment
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
func (this *Comment) column(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 表数据结构体
	var table []model.Comment
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
func (this *Comment) remove(ctx *gin.Context) {

	// 表数据结构体
	table := model.Comment{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table)

	// 越权 - 既没有管理权限，只能删除自己的数据
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

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
func (this *Comment) delete(ctx *gin.Context) {

	// 表数据结构体
	table := model.Comment{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table).WithTrashed()

	// 越权 - 既没有管理权限，只能删除自己的数据
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

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
func (this *Comment) clear(ctx *gin.Context) {

	// 表数据结构体
	table := model.Comment{}

	item := facade.DB.Model(&table).OnlyTrashed()

	// 越权 - 既没有管理权限，只能删除自己的数据
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

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
func (this *Comment) restore(ctx *gin.Context) {

	// 表数据结构体
	table := model.Comment{}
	// 获取请求参数
	params := this.params(ctx)

	// id 数组 - 参数归一化
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&table).OnlyTrashed().WhereIn("id", ids)

	// 越权 - 既没有管理权限，只能删除自己的数据
	if !this.meta.root(ctx) {
		item.Where("uid", this.user(ctx).Id)
	}

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

// replies 递归获取子评论的 id 列表
func (this *Comment) replies(pid any, ctx *gin.Context) (ids []int) {

	// 获取请求参数
	params := this.params(ctx)

	var result []model.Comment
	mold := facade.DB.Model(&result).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
	mold.Where("pid", pid).Column("id")
	mold.Select()

	for _, val := range result {
		ids = append(ids, val.Id)
		ids = append(ids, this.replies(val.Id, ctx)...)
	}

	return ids
}

// flat 扁平化数据
func (this *Comment) flat(ctx *gin.Context) {

	code := 204
	msg := []string{"无数据！", ""}
	var data any

	// 获取请求参数
	params := this.params(ctx, map[string]any{
		"page":      1,
		"bind_type": "article",
		"order":     "create_time desc",
	})

	if utils.Is.Empty(params["bind_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "bind_id 不能为空！"), 400)
		return
	}

	// 表数据结构体
	table := model.Comment{}
	// 允许查询的字段
	allow := []any{"bind_id", "bind_type"}
	// 动态给结构体赋值
	for key, val := range params {
		// 防止恶意传入字段
		if utils.In.Array(key, allow) {
			utils.Struct.Set(&table, key, val)
		}
	}

	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	var result []model.Comment
	mold := facade.DB.Model(&result).Where("pid", 0).OnlyTrashed(cast.ToBool(params["onlyTrashed"])).WithTrashed(cast.ToBool(params["withTrashed"]))
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

		list := cast.ToSlice(data)

		// 根据 pid 递归获取子评论的 id 列表
		for key, val := range list {

			ids := this.replies(cast.ToStringMap(val)["id"], ctx)
			replies := facade.DB.Model(&[]model.Comment{}).WhereIn("id", ids).Order("create_time asc").Select()

			// 排除字段
			cast.ToStringMap(list[key])["replies"] = utils.ArrayMapWithField(replies, params["field"])
		}

		data = list

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

// config 配置
func (this *Comment) config(key ...any) (json map[string]any) {

	var config map[string]any
	configKey := "ARTICLE"

	// 如果请求的是评论配置，使用 COMMENT 配置
	isCommentConfig := false
	if len(key) > 0 && cast.ToString(key[0]) == "comment" {
		configKey = "COMMENT"
		isCommentConfig = true
	}

	// 缓存名称
	cacheName := "config[" + configKey + "]"
	// 是否开启了缓存
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	// 检查缓存是否存在
	if cacheState && facade.Cache.Has(cacheName) {

		config = cast.ToStringMap(facade.Cache.Get(cacheName))

	} else {

		config = facade.DB.Model(&model.Config{}).Where("key", configKey).Find()
		// 存储到缓存中
		if cacheState {
			go facade.Cache.Set(cacheName, config)
		}
	}

	// 如果是评论配置，直接返回整个 json 配置
	if isCommentConfig {
		return cast.ToStringMap(config["json"])
	}

	// 其他配置，返回指定键的值
	if len(key) > 0 {
		return cast.ToStringMap(cast.ToStringMap(config["json"])[cast.ToString(key[0])])
	}

	return cast.ToStringMap(config["json"])
}
