package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type Upgrade struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Upgrade) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"list":   this.list,
		"detail": this.detail,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Upgrade) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"theme":  this.theme,
		"system": this.system,
		"create": this.create,
		"update": this.update,
		"delete": this.delete,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - PUT请求本体
func (this *Upgrade) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - DELETE请求本体
func (this *Upgrade) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - GET请求本体
func (this *Upgrade) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// theme 主题升级
func (this *Upgrade) theme(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"files":         []string{"static"},
		"frontend_path": "/", // 前端部署路径，默认根目录
	})

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "主题版本ID"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// user 转 map
	json := cast.ToStringMap(utils.Json.Decode(utils.Json.Encode(user)))
	body := map[string]any{
		"ip":    this.get(ctx, "ip"),
		"id":    params["id"],
		"agent": this.header(ctx, "User-Agent"),
		"json":  utils.Map.WithField(json, []string{"id", "nickname", "email", "phone"}),
	}

	// 先尝试从本地数据库获取版本信息
	var upgrade model.Upgrade
	err := facade.DB.Drive().First(&upgrade, params["id"]).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "版本不存在！"), 404)
		return
	}

	// 检查本地版本的下载地址
	if utils.Is.Empty(upgrade.Url) {
		// 如果本地没有下载地址，尝试从远程服务器获取
		curl := utils.Curl(utils.CurlRequest{
			Body:    body,
			Method:  "GET",
			Headers: facade.Comm.Signature(body),
			Url:     facade.Uri + "/sn/theme-version/download",
		}).Send()

		if curl.Error != nil {
			this.json(ctx, nil, facade.Lang(ctx, "远程服务器错误：%v", curl.Error.Error()), 500)
			return
		}

		if cast.ToInt(curl.Json["code"]) != 200 {
			this.json(ctx, curl.Json["data"], curl.Json["msg"], curl.Json["code"])
			return
		}

		download := cast.ToStringMap(curl.Json["data"])
		// 检查 url 是否合法
		_, err = url.ParseRequestURI(cast.ToString(download["url"]))
		if err != nil {
			this.json(ctx, nil, facade.Lang(ctx, "%s 不合法！", "下载地址"), 400)
			return
		}
		// 使用远程服务器返回的下载地址
		upgrade.Url = cast.ToString(download["url"])
	}

	// 检查 url 是否合法
	_, err = url.ParseRequestURI(cast.ToString(upgrade.Url))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不合法！", "下载地址"), 400)
		return
	}

	// 确定前端部署路径
	frontendPath := cast.ToString(params["frontend_path"])
	if frontendPath == "" {
		frontendPath = "/"
	}

	// 下载文件
	zipPath := frontendPath + "theme.zip"
	item := utils.File().Download(cast.ToString(upgrade.Url), zipPath)
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "下载失败：%v", item.Error.Error()), 400)
		return
	}

	// 批量删除文件和目录
	for _, path := range utils.Unity.Keys(params["files"]) {
		utils.File().Remove(frontendPath + cast.ToString(path))
	}

	// 解压文件
	item = utils.File().Dir(frontendPath).Name("theme.zip").UnZip()
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "解压失败：%v", item.Error.Error()), 400)
		return
	}

	// 删除压缩包
	utils.File().Remove(zipPath)

	// 返回详细信息
	this.json(ctx, map[string]any{
		"frontend_path": frontendPath,
		"version":       upgrade.Version,
		"message":       "主题升级成功！",
		"cdn_purge":     true, // 可选：是否需要清除CDN缓存
	}, facade.Lang(ctx, "升级成功！"), 200)
}

// system 系统升级
func (this *Upgrade) system(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)
	// 压缩包名称
	name := "system.zip"

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "系统版本ID"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// user 转 map
	json := cast.ToStringMap(utils.Json.Decode(utils.Json.Encode(user)))
	body := map[string]any{
		"ip":    this.get(ctx, "ip"),
		"id":    params["id"],
		"agent": this.header(ctx, "User-Agent"),
		"json":  utils.Map.WithField(json, []string{"id", "nickname", "email", "phone"}),
	}

	// 获取系统版本的下载地址
	curl := utils.Curl(utils.CurlRequest{
		Url:     facade.Uri + "/sn/system-version/download",
		Method:  "GET",
		Headers: facade.Comm.Signature(body),
		Body:    body,
	}).Send()

	if curl.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "远程服务器错误：%v", curl.Error.Error()), 500)
		return
	}

	if cast.ToInt(curl.Json["code"]) != 200 {
		this.json(ctx, curl.Json["data"], curl.Json["msg"], curl.Json["code"])
		return
	}

	download := cast.ToStringMap(curl.Json["data"])

	// 检查 url 是否合法
	_, err := url.ParseRequestURI(cast.ToString(download["url"]))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不合法！", "下载地址"), 400)
		return
	}

	item := utils.File().Download(cast.ToString(download["url"]), name)
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "下载失败：%v", item.Error.Error()), 400)
		return
	}

	// 解压文件
	item = utils.File().Dir("./").Name(name).UnZip()
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "解压失败：%v", item.Error.Error()), 400)
		return
	}

	// 删除压缩包
	utils.File().Remove(name)

	this.json(ctx, nil, facade.Lang(ctx, "升级成功！"), 200)

	// 杀死进程
	go this.kill(ctx)
}

// kill 杀死进程
func (this *Upgrade) kill(ctx *gin.Context) {

	time.Sleep(5 * time.Second)

	// 根据操作系统选择不同的命令
	var cmd *exec.Cmd
	cmd = exec.Command("taskkill", "/F", "/PID", cast.ToString(utils.Get.Pid()))
	// 守护进程
	// nohup /www/wwwroot/inis.cn/inis 1>/dev/null 2>&1 &

	// 执行命令
	err := cmd.Run()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "关闭进程失败：%v", err.Error()), 400)
		os.Exit(1)
		return
	}
}

// create 创建版本
func (this *Upgrade) create(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["version"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "版本号"), 400)
		return
	}

	if utils.Is.Empty(params["type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "类型"), 400)
		return
	}

	if utils.Is.Empty(params["content"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "更新内容"), 400)
		return
	}

	if utils.Is.Empty(params["url"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "更新地址"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 检查类型是否合法
	if cast.ToString(params["type"]) != "app" && cast.ToString(params["type"]) != "theme" {
		this.json(ctx, nil, facade.Lang(ctx, "类型必须为 app 或 theme！"), 400)
		return
	}

	// 创建版本
	upgrade := model.Upgrade{
		Version: cast.ToString(params["version"]),
		Type:    cast.ToString(params["type"]),
		Content: cast.ToString(params["content"]),
		Url:     cast.ToString(params["url"]),
		Status:  cast.ToInt(params["status"]),
		Json:    params["json"],
		Text:    params["text"],
	}

	// 保存到数据库
	err := facade.DB.Drive().Create(&upgrade).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "创建版本失败：%v", err.Error()), 500)
		return
	}

	this.json(ctx, upgrade, facade.Lang(ctx, "创建版本成功！"), 200)
}

// update 更新版本
func (this *Upgrade) update(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "版本ID"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 查找版本
	var upgrade model.Upgrade
	err := facade.DB.Drive().First(&upgrade, params["id"]).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "版本不存在！"), 404)
		return
	}

	// 更新字段
	if !utils.Is.Empty(params["version"]) {
		upgrade.Version = cast.ToString(params["version"])
	}
	if !utils.Is.Empty(params["type"]) {
		// 检查类型是否合法
		if cast.ToString(params["type"]) != "app" && cast.ToString(params["type"]) != "theme" {
			this.json(ctx, nil, facade.Lang(ctx, "类型必须为 app 或 theme！"), 400)
			return
		}
		upgrade.Type = cast.ToString(params["type"])
	}
	if !utils.Is.Empty(params["content"]) {
		upgrade.Content = cast.ToString(params["content"])
	}
	if !utils.Is.Empty(params["url"]) {
		upgrade.Url = cast.ToString(params["url"])
	}
	if !utils.Is.Empty(params["status"]) {
		upgrade.Status = cast.ToInt(params["status"])
	}
	if !utils.Is.Empty(params["json"]) {
		upgrade.Json = params["json"]
	}
	if !utils.Is.Empty(params["text"]) {
		upgrade.Text = params["text"]
	}

	// 保存到数据库
	err = facade.DB.Drive().Save(&upgrade).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "更新版本失败：%v", err.Error()), 500)
		return
	}

	this.json(ctx, upgrade, facade.Lang(ctx, "更新版本成功！"), 200)
}

// delete 删除版本
func (this *Upgrade) delete(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "版本ID"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 查找版本
	var upgrade model.Upgrade
	err := facade.DB.Drive().First(&upgrade, params["id"]).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "版本不存在！"), 404)
		return
	}

	// 删除版本
	err = facade.DB.Drive().Delete(&upgrade).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除版本失败：%v", err.Error()), 500)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "删除版本成功！"), 200)
}

// list 获取版本列表
func (this *Upgrade) list(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 构建查询
	query := facade.DB.Drive().Model(&model.Upgrade{})

	// 按类型筛选
	if !utils.Is.Empty(params["type"]) {
		query = query.Where("type = ?", cast.ToString(params["type"]))
	}

	// 按状态筛选
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status = ?", cast.ToInt(params["status"]))
	}

	// 按版本号搜索
	if !utils.Is.Empty(params["version"]) {
		query = query.Where("version like ?", "%"+cast.ToString(params["version"])+"%")
	}

	// 分页
	page := cast.ToInt(params["page"])
	limit := cast.ToInt(params["limit"])
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	offset := (page - 1) * limit

	// 排序
	sort := cast.ToString(params["sort"])
	order := cast.ToString(params["order"])
	if sort == "" {
		sort = "create_time"
	}
	if order == "" {
		order = "desc"
	}

	// 执行查询
	var upgrades []model.Upgrade
	var total int64

	// 获取总数
	query.Count(&total)

	// 获取列表
	err := query.Order(sort + " " + order).Limit(limit).Offset(offset).Find(&upgrades).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取版本列表失败：%v", err.Error()), 500)
		return
	}

	// 返回结果
	this.json(ctx, map[string]any{
		"list":  upgrades,
		"total": total,
		"page":  page,
		"limit": limit,
	}, facade.Lang(ctx, "获取版本列表成功！"), 200)
}

// detail 获取版本详情
func (this *Upgrade) detail(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "版本ID"), 400)
		return
	}

	user := this.meta.user(ctx)
	if user.Id == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	// 查找版本
	var upgrade model.Upgrade
	err := facade.DB.Drive().First(&upgrade, params["id"]).Error
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "版本不存在！"), 404)
		return
	}

	this.json(ctx, upgrade, facade.Lang(ctx, "获取版本详情成功！"), 200)
}