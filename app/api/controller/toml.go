package controller

import (
	"context"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	toml "github.com/BurntSushi/toml"
	AliYunClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	AliYunUtil "github.com/alibabacloud-go/openapi-util/service"
	AliYunUtilV2 "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	TencentCloud "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/unti-io/go-utils/utils"
	"gopkg.in/gomail.v2"
)

// 预编译正则表达式
var tomlVarRegex = regexp.MustCompile(`\${(.+?)}`)

// retentionDaysPattern 通知保留天数配置行（每次保存配置都会用到，包级编译一次）
var retentionDaysPattern = regexp.MustCompile(`(?m)^retention_days\s*=.*$`)

// Toml - 配置管理控制器
// @Summary 配置管理API
// @Description 提供系统配置相关的API接口，包括短信、缓存、加密、存储等配置管理
// @Tags Config
type Toml struct {
	// 继承
	base
}

// replaceTomlVars 替换模板变量
func (this *Toml) replaceTomlVars(temp string, tomlData map[string]any) string {
	matches := tomlVarRegex.FindAllStringSubmatch(temp, -1)
	for _, match := range matches {
		placeholder := match[0] // ${file.path}
		keyPath := match[1]     // file.path

		// 支持嵌套 key，如 "file.path" -> tomlData["file"]["path"]
		value := this.getNestedValue(tomlData, keyPath)
		temp = strings.Replace(temp, placeholder, cast.ToString(value), -1)
	}
	return temp
}

// 敏感字段名集合（密钥、密码等，返回给前端时需脱敏）
var sensitiveTomlFields = map[string]bool{
	"password":          true, // mysql/redis/email 密码
	"access_key_secret": true, // 阿里云 AccessKey Secret
	"secret_key":        true, // 腾讯云/七牛云 SecretKey
	"access_key":        true, // 七牛云 AccessKey
	"access_key_id":     true, // AccessKey ID
	"secret_id":         true, // 腾讯云 SecretId
	"key":               true, // JWT 签名密钥（仅出现在 crypt 配置的 jwt 段）
}

// restoreSecretParams 还原脱敏占位值：若前端回传的是脱敏后的密钥（含 **** 或全 *），
// 则用当前配置中的原值替换，避免把脱敏值写回配置导致密钥损坏
func (this *Toml) restoreSecretParams(params map[string]any, keys []string, current map[string]any) {
	for _, key := range keys {
		val := cast.ToString(params[key])
		// 脱敏占位特征：包含 **** 或（非空且全由 * 组成）
		if strings.Contains(val, "****") || (val != "" && strings.Trim(val, "*") == "") {
			if original, ok := current[key]; ok && original != nil {
				params[key] = original
			} else {
				delete(params, key)
			}
		}
	}
}

// maskSensitiveFields 递归脱敏 map/slice 中的敏感字段
//
// 管理员（permit 判定，与权限中间件 Rule 同口径）**不脱敏**：
// 后台「系统配置」页需要看到并核对真实的密钥/密码，否则表单只能回填 **** 占位串。
// 其它调用方（非管理员）仍按 sensitiveTomlFields 脱敏。
func (this *Toml) maskSensitiveFields(ctx *gin.Context, data any) any {

	// 管理员直返原值
	if ctx != nil && this.meta.permit(ctx) {
		return data
	}

	switch val := data.(type) {
	case map[string]any:
		for key, value := range val {
			if sensitiveTomlFields[strings.ToLower(key)] {
				val[key] = facade.Comm.MaskSecret(cast.ToString(value))
			} else {
				val[key] = this.maskSensitiveFields(ctx, value)
			}
		}
		return val
	case []any:
		for i, value := range val {
			val[i] = this.maskSensitiveFields(ctx, value)
		}
		return val
	default:
		return data
	}
}

// getNestedValue 从嵌套 map 中获取值，支持 "file.path" 这样的路径
func (this *Toml) getNestedValue(data map[string]any, keyPath string) any {
	parts := strings.Split(keyPath, ".")
	var current any = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			if val, ok := v[part]; ok {
				current = val
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return current
}

// storageConfigToReplaceMap 将存储配置转换为 replaceMap
func (this *Toml) storageConfigToReplaceMap() map[string]any {
	result := make(map[string]any)
	data := facade.StorageToml.Result

	if v, ok := data["default"]; ok {
		result["${default}"] = v
	}

	if local, ok := data["local"].(map[string]any); ok {
		if v, ok := local["domain"]; ok {
			result["${local.domain}"] = v
		}
		if v, ok := local["path"]; ok {
			result["${local.path}"] = v
		}
	}
	// 命名规则：配置里缺项（老配置文件没有这两个键）时用默认值补上，
	// 否则整份重建模板会把它们写成空串
	result["${local.dir_rule}"] = facade.StorageRule("local", facade.StorageRuleDir)
	result["${local.file_rule}"] = facade.StorageRule("local", facade.StorageRuleFile)

	if cos, ok := data["cos"].(map[string]any); ok {
		if v, ok := cos["secret_id"]; ok {
			result["${cos.secret_id}"] = v
		}
		if v, ok := cos["secret_key"]; ok {
			result["${cos.secret_key}"] = v
		}
		if v, ok := cos["app_id"]; ok {
			result["${cos.app_id}"] = v
		}
		if v, ok := cos["bucket"]; ok {
			result["${cos.bucket}"] = v
		}
		if v, ok := cos["region"]; ok {
			result["${cos.region}"] = v
		}
		if v, ok := cos["domain"]; ok {
			result["${cos.domain}"] = v
		}
		if v, ok := cos["path"]; ok {
			result["${cos.path}"] = v
		}
	}
	result["${cos.dir_rule}"] = facade.StorageRule("cos", facade.StorageRuleDir)
	result["${cos.file_rule}"] = facade.StorageRule("cos", facade.StorageRuleFile)

	if attachment, ok := data["attachment"].(map[string]any); ok {
		if v, ok := attachment["allow_extensions"]; ok {
			result["${attachment.allow_extensions}"] = v
		}
		if v, ok := attachment["max_file_size"]; ok {
			result["${attachment.max_file_size}"] = v
		}
		if v, ok := attachment["concurrent_limit"]; ok {
			result["${attachment.concurrent_limit}"] = v
		}
	}

	return result
}

// validateStorageRules - 校验上传命名规则（dir_rule / file_rule）
//
// 只允许 facade.StorageRulePlaceholders 里列出的占位符。未知占位符直接报 400：
// 一旦写进配置，上传会生成带 "{xx}" 的目录，用户很难自己发现是配置写错了。
//
// label 用于提示语（如「本地存储」「腾讯云 COS」），rules 可以是扁平参数
// （storage-local / storage-cos）或统一保存里的局部 map（storage 的 local / cos）。
func (this *Toml) validateStorageRules(ctx *gin.Context, label string, rules map[string]any) bool {

	for _, kind := range []string{facade.StorageRuleDir, facade.StorageRuleFile} {

		rule := cast.ToString(rules[kind])
		if utils.Is.Empty(rule) {
			// 留空表示用默认规则（见 facade.StorageRule）
			continue
		}

		if unknown := facade.UnknownStorageRulePlaceholders(rule); len(unknown) > 0 {
			this.json(ctx, nil, facade.Lang(ctx, "%s 的 %s 里 %s 不是可用占位符！", label, kind, strings.Join(unknown, "、")), 400)
			return false
		}
	}

	return true
}

// saveTomlConfig 保存配置文件
//
// 写完必须把对应的内存配置也换成新值，否则前端保存后立刻回读会拿到旧快照
// （各配置的 Result 是首次 Read 时拍下的，WatchConfig 只刷新 Get，不会刷新 Result）。
//
// 两个重载都用同步调用：前端 saveTomlItem 是「保存成功后马上 GET 回填」，
// 用 go 会和紧接着的回读请求赛跑，表现为偶发地仍然显示旧配置。
func (this *Toml) saveTomlConfig(ctx *gin.Context, content string, path string, successMsg string) {
	item := utils.File().Save(strings.NewReader(content), path)
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "修改失败！"), 400)
		return
	}

	switch path {
	case "config/storage.toml":
		facade.ReloadStorageToml()
	case "config/sms.toml":
		facade.ReloadSMSToml()
	}

	this.json(ctx, nil, facade.Lang(ctx, successMsg), 200)
}

// IGET - 获取配置信息
// @Summary 获取配置信息
// @Description 根据不同方法获取系统配置相关数据
func (this *Toml) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"log":             this.getLog,
		"sms":             this.getSMS,
		"cache":           this.getCache,
		"crypt":           this.getCrypt,
		"storage":         this.getStorage,
		"notification":    this.getNotification,
		"sensitive":       this.getSensitive,
		"sensitive-stats": this.sensitiveStats,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - 测试配置服务
// @Summary 测试配置服务
// @Description 测试系统配置相关服务（短信、Redis、存储等）
func (this *Toml) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"test-sms-email":                this.testSMSEmail,
		"test-sms-aliyun":               this.testSMSAliyun,
		"test-sms-aliyun-number-verify": this.testSMSAliYunNumberVerify,
		"test-sms-tencent":              this.testSMSTencent,
		"test-redis":                    this.testRedis,
		"test-cos":                      this.testCOS,
		"sensitive-test":                this.testSensitive,
		"sensitive-rollback":            this.rollbackSensitive,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - 更新配置信息
// @Summary 更新配置信息
// @Description 更新系统配置相关数据（短信、加密、缓存、存储等）
func (this *Toml) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"sms":                      this.putSMS,
		"sms-email":                this.putSMSEmail,
		"sms-email-queue":          this.putSMSEmailQueue,
		"sms-aliyun":               this.putSMSAliyun,
		"sms-aliyun-number-verify": this.putSMSAliYunNumberVerify,
		"sms-tencent":              this.putSMSTencent,
		"sms-drive":                this.putSMSDrive,
		"crypt-session":            this.putCryptSession,
		"cache-default":            this.putCacheDefault,
		"cache-redis":              this.putCacheRedis,
		"cache-file":               this.putCacheFile,
		"cache-ram":                this.putCacheRam,
		"storage":                  this.putStorage,
		"storage-default":          this.putStorageDefault,
		"storage-local":            this.putStorageLocal,
		"storage-cos":              this.putStorageCOS,
		"storage-attachment":       this.putStorageAttachment,
		"notification":             this.putNotification,
		// 源码原文与可视化表单共用：按载荷（content / rules）自动分派
		"sensitive": this.putSensitive,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - 删除配置信息
// @Summary 删除配置信息
// @Description 删除系统配置相关数据（当前暂不支持）
func (this *Toml) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - 配置管理首页
// @Summary 配置管理首页
// @Description 配置管理控制器首页（没什么用）
func (this *Toml) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// getSMS - 获取SMS服务配置
func (this *Toml) getSMS(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 允许的查询范围 - 新增 aliyun_number_verify
	field := []any{"email", "aliyun", "aliyun_number_verify", "tencent"}

	item := facade.SMSToml
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "SMS配置文件异常！"), 400)
		return
	}

	// 获取全部
	if utils.Is.Empty(params["name"]) {
		// 逐段 Get 而不是直接返回 item.Result：
		// Result 是首次 Read 时拍下的快照，配置文件热更新不会刷新它，
		// 直接返回会导致「保存成功但回读还是旧值」。Get 走底层 viper，是实时的。
		result := make(map[string]any, len(item.Result))
		for key, fallback := range item.Result {
			if value := item.Get(key); value != nil {
				result[key] = value
			} else {
				result[key] = fallback
			}
		}

		this.json(ctx, this.maskSensitiveFields(ctx, result), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	result := cast.ToStringMap(item.Get(cast.ToString(params["name"])))
	result["drive"] = item.Get("drive")

	// 获取指定（脱敏敏感字段）
	this.json(ctx, this.maskSensitiveFields(ctx, result), facade.Lang(ctx, "数据请求成功！"), 200)
}

// putSMS - 修改SMS服务配置
func (this *Toml) putSMS(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "配置名称不能为空！"), 400)
		return
	}

	// 允许的修改范围 - 新增 aliyun_number_verify / email_queue
	field := []any{"default", "email", "email_queue", "aliyun", "aliyun_number_verify", "tencent"}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的修改范围！"), 400)
		return
	}

	switch params["name"] {
	case "drive":
		this.putSMSDrive(ctx)
	case "email":
		this.putSMSEmail(ctx)
	case "email_queue":
		this.putSMSEmailQueue(ctx)
	case "aliyun":
		this.putSMSAliyun(ctx)
	case "aliyun_number_verify": // 新增分支
		this.putSMSAliYunNumberVerify(ctx)
	case "tencent":
		this.putSMSTencent(ctx)
	default:
		this.putSMSDrive(ctx)
	}
}

// getCache - 获取缓存服务配置
func (this *Toml) getCache(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 允许的查询范围
	field := []any{"redis", "file", "ram"}

	item := facade.CacheToml
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "缓存配置文件异常！"), 400)
		return
	}

	// 获取全部
	if utils.Is.Empty(params["name"]) {
		// 同 getSMS：Result 是启动时的快照，热更新不会刷新它，这里用 Get 实时取值
		result := make(map[string]any, len(item.Result))
		for key, fallback := range item.Result {
			if value := item.Get(key); value != nil {
				result[key] = value
			} else {
				result[key] = fallback
			}
		}

		this.json(ctx, this.maskSensitiveFields(ctx, result), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	result := cast.ToStringMap(item.Get(cast.ToString(params["name"])))
	result["open"] = cast.ToBool(item.Get("open"))
	result["default"] = item.Get("default")

	// 获取指定（脱敏敏感字段）
	this.json(ctx, this.maskSensitiveFields(ctx, result), facade.Lang(ctx, "数据请求成功！"), 200)
}

// getCrypt - 获取加密服务配置
func (this *Toml) getCrypt(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 允许的查询范围（JWT 已移除，只剩登录会话配置；老配置里仍可能是 [jwt] 段）
	field := []any{"session", "jwt"}

	item := facade.CryptToml
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "加密配置文件异常！"), 400)
		return
	}

	// 兼容老配置：crypt.toml 里只有 [jwt] 时，把 expire 映射到 session 返回，
	// 避免后台「登录会话」页在管理员保存前显示为空
	result := cast.ToStringMap(item.Get("session"))
	if utils.Is.Empty(result["expire"]) {
		if legacy := cast.ToStringMap(item.Get("jwt")); !utils.Is.Empty(legacy["expire"]) {
			result["expire"] = legacy["expire"]
		}
	}
	if utils.Is.Empty(result["expire"]) {
		result["expire"] = facade.DefaultSessionExpire
	}

	payload := gin.H{"session": result}

	// 获取全部
	if utils.Is.Empty(params["name"]) {
		this.json(ctx, payload, facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	// 指定查询（name=jwt 是旧前端的写法，同样返回会话配置）
	this.json(ctx, payload, facade.Lang(ctx, "数据请求成功！"), 200)
}

// applyStorageRules - 给存储配置补上命名规则的生效值（dir_rule / file_rule）
//
// 老配置文件里没有这两个键，GET 时按默认值补上，接口返回的就是实际生效的规则
// （见 facade.StorageRule），前端表单和调用方都不用各自猜默认值。
// driver 为空表示整份配置（local / cos 都补），否则只补指定分组。
func (this *Toml) applyStorageRules(data map[string]any, driver string) map[string]any {

	result := make(map[string]any, len(data))
	for key, val := range data {
		result[key] = val
	}

	drivers := []string{"local", "cos"}
	if driver != "" {
		drivers = []string{driver}
	}

	for _, name := range drivers {
		section := cast.ToStringMap(result[name])
		copied := make(map[string]any, len(section)+2)
		for key, val := range section {
			copied[key] = val
		}
		copied[facade.StorageRuleDir] = facade.StorageRule(name, facade.StorageRuleDir)
		copied[facade.StorageRuleFile] = facade.StorageRule(name, facade.StorageRuleFile)
		result[name] = copied
	}

	return result
}

// getStorage - 获取存储服务配置
func (this *Toml) getStorage(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 允许的查询范围（仅保留本地存储 local 与腾讯云 COS）
	field := []any{"local", "cos", "attachment"}

	item := facade.StorageToml
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "存储配置文件异常！"), 400)
		return
	}

	// 获取全部
	if utils.Is.Empty(params["name"]) {
		this.json(ctx, this.applyStorageRules(item.Result, ""), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	// 获取全部（脱敏敏感字段）
	if utils.Is.Empty(params["name"]) {
		this.json(ctx, this.maskSensitiveFields(ctx, this.applyStorageRules(item.Result, "")), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	name := cast.ToString(params["name"])
	result := cast.ToStringMap(item.Get(name))
	result["default"] = item.Get("default")

	// 单个驱动（local / cos）时补上命名规则的生效值；attachment 分组不涉及
	if name == "local" || name == "cos" {
		result = this.applyStorageRules(result, name)
	}

	// 获取指定（脱敏敏感字段）
	this.json(ctx, this.maskSensitiveFields(ctx, result), facade.Lang(ctx, "数据请求成功！"), 200)
}

// getStorage - 获取日志服务配置
func (this *Toml) getLog(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 允许的查询范围
	var field []any

	item := facade.LogToml
	if item.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "日志配置文件异常！"), 400)
		return
	}

	// 获取全部（脱敏敏感字段）
	if utils.Is.Empty(params["name"]) {
		this.json(ctx, this.maskSensitiveFields(ctx, item.Result), facade.Lang(ctx, "数据请求成功！"), 200)
		return
	}

	if !utils.In.Array(params["name"], field) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许的查询范围！"), 400)
		return
	}

	// 获取指定
	this.json(ctx, item.Get(cast.ToString(params["name"])), facade.Lang(ctx, "数据请求成功！"), 200)
}

// putSMSDrive - 修改SMS驱动配置
func (this *Toml) putSMSDrive(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"default": "email",
	})

	opts := make(map[string]any)
	allow := []any{"email", "sms", "default"}

	for key, value := range params {
		if !utils.In.Array(key, allow) {
			continue
		}
		opts[fmt.Sprintf("${drive.%s}", key)] = value
	}

	temp := facade.TempSMS
	temp = utils.Replace(temp, opts)
	temp = this.replaceTomlVars(temp, this.smsTomlData())

	this.saveTomlConfig(ctx, temp, "config/sms.toml", "修改成功！")
}

// smsTomlData 当前 sms.toml 的可用于模板替换的数据副本
//
// 模板 TempSMS 里 [email] 段既有服务信息也有发件队列参数，配置里缺项时会替换成空值，
// 写出 `batch_size = ` 这种非法 TOML（老配置文件没有队列项）。
// 因此这里复制一份配置，并用 MailQueueDefaultValues 补齐 [email] 段的缺项。
func (this *Toml) smsTomlData() map[string]any {

	data := make(map[string]any, len(facade.SMSToml.Result))
	for key, val := range facade.SMSToml.Result {
		data[key] = val
	}

	email := make(map[string]any)
	for key, val := range cast.ToStringMap(data["email"]) {
		email[key] = val
	}
	for key, val := range facade.MailQueueDefaultValues() {
		if utils.Is.Empty(email[key]) {
			email[key] = val
		}
	}
	data["email"] = email

	return data
}

// putSMSEmailQueue - 修改发件队列配置（config/sms.toml 的 [email] 段）
//
// 对应字段（与 app/facade/mail_queue.go 的 MailQueueDefaultValues 一一对应）：
// batch_size / batch_interval / retry_delay / max_attempts / send_timeout / verify_wait / queue_size
//
// 全部可选：只提交需要修改的字段，未提交的保持原值（配置里缺项时用默认值补齐）；
// 每个字段都有取值范围（见 facade.MailQueueLimits），越界直接返回 400。
// 保存后由配置监听（initSMS → mailQueue.reload）热更新，无需重启。
func (this *Toml) putSMSEmailQueue(ctx *gin.Context) {

	params := this.params(ctx)
	tomlData := this.smsTomlData()
	email := cast.ToStringMap(tomlData["email"])

	limits := facade.MailQueueLimits()
	for key, limit := range limits {
		raw, ok := params[key]
		// 注意：这里只跳过「未提交 / 空串」，0 是合法值（如 verify_wait = 0 表示纯异步）
		if !ok || cast.ToString(raw) == "" {
			continue
		}

		if !utils.Is.Number(raw) {
			this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", key), 400)
			return
		}

		value := cast.ToInt(raw)
		if value < limit[0] || value > limit[1] {
			this.json(ctx, nil, facade.Lang(ctx, "%s 取值范围 %d ~ %d！", key, limit[0], limit[1]), 400)
			return
		}

		email[key] = value
	}

	tomlData["email"] = email

	// 整份文件按模板重建（未提交的字段沿用当前配置）
	temp := this.replaceTomlVars(facade.TempSMS, tomlData)

	this.saveTomlConfig(ctx, temp, "config/sms.toml", "发件队列配置已保存！")
}

// putSMSEmail - 修改SMS邮箱配置
func (this *Toml) putSMSEmail(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（避免把脱敏密钥写回配置）
	this.restoreSecretParams(params, []string{"password"}, cast.ToStringMap(facade.SMSToml.Get("email")))

	if utils.Is.Empty(params["host"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "host"), 400)
		return
	}

	if utils.Is.Empty(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "port"), 400)
		return
	}

	if !utils.Is.Number(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "port"), 400)
		return
	}

	if utils.Is.Empty(params["account"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "account"), 400)
		return
	}

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "password"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	temp := facade.TempSMS
	temp = utils.Replace(temp, map[string]any{
		"${email.host}":      params["host"],
		"${email.port}":      cast.ToInt(params["port"]),
		"${email.account}":   params["account"],
		"${email.password}":  params["password"],
		"${email.nickname}":  params["nickname"],
		"${email.sign_name}": params["sign_name"],
	})
	temp = this.replaceTomlVars(temp, this.smsTomlData())

	this.saveTomlConfig(ctx, temp, "config/sms.toml", "修改成功！")
}

// putSMSAliyun - 修改阿里云短信服务配置
func (this *Toml) putSMSAliyun(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（避免把脱敏密钥写回配置）
	this.restoreSecretParams(params, []string{"access_key_secret"}, cast.ToStringMap(facade.SMSToml.Get("aliyun")))

	if utils.Is.Empty(params["access_key_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_id"), 400)
		return
	}

	if utils.Is.Empty(params["access_key_secret"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_secret"), 400)
		return
	}

	if utils.Is.Empty(params["endpoint"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "endpoint"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["verify_code"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "verify_code"), 400)
		return
	}

	temp := facade.TempSMS
	temp = utils.Replace(temp, map[string]any{
		"${aliyun.access_key_id}":     params["access_key_id"],
		"${aliyun.access_key_secret}": params["access_key_secret"],
		"${aliyun.endpoint}":          params["endpoint"],
		"${aliyun.sign_name}":         params["sign_name"],
		"${aliyun.verify_code}":       params["verify_code"],
	})
	temp = this.replaceTomlVars(temp, this.smsTomlData())

	this.saveTomlConfig(ctx, temp, "config/sms.toml", "修改成功！")
}

// putSMSAliYunNumberVerify - 修改阿里云号码验证配置
func (this *Toml) putSMSAliYunNumberVerify(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（避免把脱敏密钥写回配置）
	this.restoreSecretParams(params, []string{"access_key_secret"}, cast.ToStringMap(facade.SMSToml.Get("aliyun_number_verify")))

	// 必传参数校验
	if utils.Is.Empty(params["access_key_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_id"), 400)
		return
	}

	if utils.Is.Empty(params["access_key_secret"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_secret"), 400)
		return
	}

	if utils.Is.Empty(params["endpoint"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "endpoint"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["template_code"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "template_code"), 400)
		return
	}

	// 替换配置模板中的变量
	temp := facade.TempSMS
	temp = utils.Replace(temp, map[string]any{
		"${aliyun_number_verify.access_key_id}":     params["access_key_id"],
		"${aliyun_number_verify.access_key_secret}": params["access_key_secret"],
		"${aliyun_number_verify.endpoint}":          params["endpoint"],
		"${aliyun_number_verify.sign_name}":         params["sign_name"],
		"${aliyun_number_verify.template_code}":     params["template_code"],
	})
	temp = this.replaceTomlVars(temp, this.smsTomlData())

	// 保存配置文件
	this.saveTomlConfig(ctx, temp, "config/sms.toml", "修改成功！")
}

// putSMSTencent - 修改腾讯云短信服务配置
func (this *Toml) putSMSTencent(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（避免把脱敏密钥写回配置）
	this.restoreSecretParams(params, []string{"secret_key", "secret_id"}, cast.ToStringMap(facade.SMSToml.Get("tencent")))

	if utils.Is.Empty(params["secret_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_id"), 400)
		return
	}

	if utils.Is.Empty(params["secret_key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_key"), 400)
		return
	}

	if utils.Is.Empty(params["endpoint"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "endpoint"), 400)
		return
	}

	if utils.Is.Empty(params["sms_sdk_app_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sms_sdk_app_id"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["verify_code"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "verify_code"), 400)
		return
	}

	if utils.Is.Empty(params["region"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "region"), 400)
		return
	}

	temp := facade.TempSMS
	temp = utils.Replace(temp, map[string]any{
		"${tencent.secret_id}":      params["secret_id"],
		"${tencent.secret_key}":     params["secret_key"],
		"${tencent.endpoint}":       params["endpoint"],
		"${tencent.sms_sdk_app_id}": params["sms_sdk_app_id"],
		"${tencent.sign_name}":      params["sign_name"],
		"${tencent.verify_code}":    params["verify_code"],
		"${tencent.region}":         params["region"],
	})
	temp = this.replaceTomlVars(temp, this.smsTomlData())

	this.saveTomlConfig(ctx, temp, "config/sms.toml", "修改成功！")
}

// testEmail - 测试邮件服务
func (this *Toml) testSMSEmail(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（非管理员看到的密码是 ****，需还原成真实值再测试）
	this.restoreSecretParams(params, []string{"password"}, cast.ToStringMap(facade.SMSToml.Get("email")))

	if utils.Is.Empty(params["host"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "host"), 400)
		return
	}

	if utils.Is.Empty(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "port"), 400)
		return
	}

	if !utils.Is.Number(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "port"), 400)
		return
	}

	if utils.Is.Empty(params["account"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "account"), 400)
		return
	}

	if utils.Is.Empty(params["password"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "password"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["email"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "email"), 400)
		return
	}

	if !utils.Is.Email(params["email"]) {
		this.json(ctx, nil, facade.Lang(ctx, "邮箱格式不正确！"), 400)
		return
	}

	client := gomail.NewDialer(
		cast.ToString(params["host"]),
		cast.ToInt(params["port"]),
		cast.ToString(params["account"]),
		cast.ToString(params["password"]),
	)

	item := gomail.NewMessage()
	nickname := cast.ToString(params["nickname"])
	account := cast.ToString(params["account"])
	// 显示名支持中文：必须走 SetAddressHeader，详见 facade.SetMailFrom 的注释
	facade.SetMailFrom(item, nickname, account)
	// 发送给多个用户
	item.SetHeader("To", cast.ToString(params["email"]))
	// 设置邮件主题
	item.SetHeader("Subject", cast.ToString(params["sign_name"]))
	// 设置邮件正文
	item.SetBody("text/html", "当您收到此封邮件时，说明您的邮件服务配置正确！")

	// 发送邮件
	err := client.DialAndSend(item)

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试邮件发送失败！"), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "测试邮件发送成功！"), 200)
}

// testSMSAliyun - 发送阿里云测试短信
func (this *Toml) testSMSAliyun(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（非管理员看到的密钥是 ****，需还原成真实值再测试）
	this.restoreSecretParams(params, []string{"access_key_secret"}, cast.ToStringMap(facade.SMSToml.Get("aliyun")))

	if utils.Is.Empty(params["access_key_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_id"), 400)
		return
	}

	if utils.Is.Empty(params["access_key_secret"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_secret"), 400)
		return
	}

	if utils.Is.Empty(params["endpoint"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "endpoint"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["verify_code"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "verify_code"), 400)
		return
	}

	if utils.Is.Empty(params["phone"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "phone"), 400)
		return
	}

	if !utils.Is.Phone(params["phone"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 格式不正确！", "phone"), 400)
		return
	}

	client, err := AliYunClient.NewClient(&AliYunClient.Config{
		// 访问的域名
		Endpoint: tea.String(cast.ToString(params["endpoint"])),
		// 必填，您的 AccessKey ID
		AccessKeyId: tea.String(cast.ToString(params["access_key_id"])),
		// 必填，您的 AccessKey Secret
		AccessKeySecret: tea.String(cast.ToString(params["access_key_secret"])),
	})

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	query := map[string]any{
		// 必填，接收短信的手机号码
		"PhoneNumbers": tea.String(cast.ToString(params["phone"])),
		// 必填，短信签名名称
		"SignName": tea.String(cast.ToString(params["sign_name"])),
		// 必填，短信模板ID
		"TemplateCode": tea.String(cast.ToString(params["verify_code"])),
	}

	query["TemplateParam"] = tea.String(utils.Json.Encode(map[string]any{
		"code": 6666,
	}))

	runtime := &AliYunUtilV2.RuntimeOptions{}
	request := &AliYunClient.OpenApiRequest{
		Query: AliYunUtil.Query(query),
	}

	// 返回值为 Map 类型，可从 Map 中获得三类数据：响应体 body、响应头 headers、HTTP 返回的状态码 statusCode
	result, err := client.CallApi((&facade.AliYunSMS{}).ApiInfo(), request, runtime)
	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" {
		this.json(ctx, cast.ToString(body["Message"]), facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "测试短信发送成功！"), 200)
}

// testSMSAliYunNumberVerify - 测试阿里云号码验证服务
func (this *Toml) testSMSAliYunNumberVerify(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（非管理员看到的密钥是 ****，需还原成真实值再测试）
	this.restoreSecretParams(params, []string{"access_key_secret"}, cast.ToStringMap(facade.SMSToml.Get("aliyun_number_verify")))

	// 必传参数校验
	if utils.Is.Empty(params["access_key_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_id"), 400)
		return
	}

	if utils.Is.Empty(params["access_key_secret"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "access_key_secret"), 400)
		return
	}

	if utils.Is.Empty(params["endpoint"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "endpoint"), 400)
		return
	}

	if utils.Is.Empty(params["sign_name"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "sign_name"), 400)
		return
	}

	if utils.Is.Empty(params["template_code"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "template_code"), 400)
		return
	}

	if utils.Is.Empty(params["phone"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "phone"), 400)
		return
	}

	if !utils.Is.Phone(params["phone"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 格式不正确！", "phone"), 400)
		return
	}

	// 创建阿里云客户端
	client, err := AliYunClient.NewClient(&AliYunClient.Config{
		Endpoint:        tea.String(cast.ToString(params["endpoint"])),
		AccessKeyId:     tea.String(cast.ToString(params["access_key_id"])),
		AccessKeySecret: tea.String(cast.ToString(params["access_key_secret"])),
	})

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试号码验证失败！"), 400)
		return
	}

	// 组装发送验证码请求参数
	reqParams := map[string]any{
		"SchemeName":       tea.String("测试方案"),
		"CountryCode":      tea.String("86"),
		"PhoneNumber":      tea.String(cast.ToString(params["phone"])),
		"SignName":         tea.String(cast.ToString(params["sign_name"])),
		"TemplateCode":     tea.String(cast.ToString(params["template_code"])),
		"TemplateParam":    tea.String(`{"code":"##code##","min":"5"}`),
		"CodeLength":       tea.Int64(6),
		"ValidTime":        tea.Int64(300),
		"ReturnVerifyCode": tea.Bool(true), // 返回验证码便于测试
	}

	// 发送请求
	runtime := &AliYunUtilV2.RuntimeOptions{}
	request := &AliYunClient.OpenApiRequest{
		Query: AliYunUtil.Query(reqParams),
	}

	// 调用SendSmsVerifyCode接口
	result, err := client.CallApi((&facade.AliYunNumberVerify{}).SendSmsVerifyCodeApiInfo(), request, runtime)
	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试号码验证失败！"), 400)
		return
	}

	// 处理响应结果
	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" || !cast.ToBool(body["Success"]) {
		errMsg := cast.ToString(body["Message"])
		if utils.Is.Empty(errMsg) {
			errMsg = "发送验证码失败"
		}
		this.json(ctx, errMsg, facade.Lang(ctx, "测试号码验证失败！"), 400)
		return
	}

	// 解析返回的验证码（便于测试）
	model := cast.ToStringMap(body["Model"])
	verifyCode := cast.ToString(model["VerifyCode"])

	// 返回成功结果，包含验证码便于测试核验功能
	this.json(ctx, gin.H{
		"verify_code": verifyCode,
		"message":     "测试验证码发送成功",
	}, facade.Lang(ctx, "测试号码验证成功！"), 200)
}

// testSMSTencent - 发送腾讯云测试短信
func (this *Toml) testSMSTencent(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（非管理员看到的密钥是 ****，需还原成真实值再测试）
	this.restoreSecretParams(params, []string{"secret_key", "secret_id"}, cast.ToStringMap(facade.SMSToml.Get("tencent")))

	credential := common.NewCredential(
		cast.ToString(params["secret_id"]),
		cast.ToString(params["secret_key"]),
	)
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = cast.ToString(params["endpoint"])
	client, err := TencentCloud.NewClient(
		credential,
		cast.ToString(params["region"]),
		clientProfile,
	)

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	// 实例化一个请求对象,每个接口都会对应一个request对象
	request := TencentCloud.NewSendSmsRequest()

	request.PhoneNumberSet = common.StringPtrs([]string{cast.ToString(params["phone"])})
	request.SmsSdkAppId = common.StringPtr(cast.ToString(params["sms_sdk_app_id"]))
	request.SignName = common.StringPtr(cast.ToString(params["sign_name"]))
	request.TemplateId = common.StringPtr(cast.ToString(params["verify_code"]))
	request.TemplateParamSet = common.StringPtrs([]string{"6666"})

	item, err := client.SendSms(request)

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	if item.Response == nil {
		this.json(ctx, "response is nil", facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	if len(item.Response.SendStatusSet) == 0 {
		this.json(ctx, "response send status set is nil", facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	if *item.Response.SendStatusSet[0].Code != "Ok" {
		this.json(ctx, item.Response.SendStatusSet[0].Message, facade.Lang(ctx, "测试短信发送失败！"), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "测试短信发送成功！"), 200)
}

// putCryptSession - 修改登录会话配置
//
// JWT 已彻底移除，crypt.toml 只剩 [session] expire；
// 保存时会用新模板重写文件，老站点里的 [jwt] 段随之清理掉。
func (this *Toml) putCryptSession(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["expire"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "expire"), 400)
		return
	}

	// 有效期必须是可计算的表达式且大于 0（否则会话一创建就过期）
	if cast.ToInt64(utils.Calc(cast.ToString(params["expire"]))) <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 格式不正确！", "expire"), 400)
		return
	}

	temp := utils.Replace(facade.TempCrypt, map[string]any{
		"${session.expire}": params["expire"],
	})
	temp = this.replaceTomlVars(temp, facade.CryptToml.Result)

	this.saveTomlConfig(ctx, temp, "config/crypt.toml", "修改成功！")
}

// putCacheRedis - 修改Redis缓存配置
func (this *Toml) putCacheRedis(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"database": 0,
		"host":     "localhost",
		"port":     6379,
		"prefix":   "inis:",
		"expire":   "2 * 60 * 60",
	})

	// 还原脱敏占位值（避免把脱敏密码写回配置）
	this.restoreSecretParams(params, []string{"password"}, cast.ToStringMap(facade.CacheToml.Get("redis")))

	if !utils.Is.Number(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "port"), 400)
		return
	}

	if !utils.Is.Number(params["database"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "database"), 400)
		return
	}

	temp := facade.TempCache
	temp = utils.Replace(temp, map[string]any{
		"${redis.host}":     params["host"],
		"${redis.port}":     params["port"],
		"${redis.database}": params["database"],
		"${redis.password}": params["password"],
		"${redis.prefix}":   params["prefix"],
		"${redis.expire}":   params["expire"],
		"${open}":           cast.ToBool(facade.CacheToml.Get("open")),
	})
	temp = this.replaceTomlVars(temp, facade.CacheToml.Result)

	this.saveTomlConfig(ctx, temp, "config/cache.toml", "修改成功！")
}

// putCacheFile - 修改File缓存配置
func (this *Toml) putCacheFile(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"path":   "runtime/cache",
		"prefix": "inis_",
		"expire": "2 * 60 * 60",
	})

	temp := facade.TempCache
	temp = utils.Replace(temp, map[string]any{
		"${file.path}":   params["path"],
		"${file.prefix}": params["prefix"],
		"${file.expire}": params["expire"],
		"${open}":        cast.ToBool(facade.CacheToml.Get("open")),
	})
	temp = this.replaceTomlVars(temp, facade.CacheToml.Result)

	this.saveTomlConfig(ctx, temp, "config/cache.toml", "修改成功！")
}

// putCacheRam - 修改Ram缓存配置
func (this *Toml) putCacheRam(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"expire": "2 * 60 * 60",
	})

	temp := facade.TempCache
	temp = utils.Replace(temp, map[string]any{
		"${ram.expire}": params["expire"],
		"${open}":       cast.ToBool(facade.CacheToml.Get("open")),
	})
	temp = this.replaceTomlVars(temp, facade.CacheToml.Result)

	this.saveTomlConfig(ctx, temp, "config/cache.toml", "修改成功！")
}

// testRedis - 测试Redis连接
func (this *Toml) testRedis(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"database": 0,
		"host":     "localhost",
		"port":     6379,
	})

	if !utils.Is.Number(params["port"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "port"), 400)
		return
	}

	if !utils.Is.Number(params["database"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 只能是数字！", "database"), 400)
		return
	}

	// 创建Redis连接客户端
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", params["host"], cast.ToInt(params["port"])),
		DB:       cast.ToInt(params["database"]),
		Password: cast.ToString(params["password"]),
	})

	// Ping Redis
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试Redis连接失败！"), 400)
		return
	}

	this.json(ctx, pong, facade.Lang(ctx, "测试Redis连接成功！"), 200)
}

// putCacheDefault - 修改缓存默认服务类型
func (this *Toml) putCacheDefault(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"value": "file",
		"open":  "false",
	})

	allow := []any{"redis", "file", "ram"}

	if !utils.In.Array(params["value"], allow) {
		this.json(ctx, nil, facade.Lang(ctx, "value 只允许是 redis、file、ram ！"), 400)
		return
	}

	temp := facade.TempCache
	temp = utils.Replace(temp, map[string]any{
		"${open}":    utils.Ternary(cast.ToBool(params["open"]), "true", "false"),
		"${default}": params["value"],
	})
	temp = this.replaceTomlVars(temp, facade.CacheToml.Result)

	this.saveTomlConfig(ctx, temp, "config/cache.toml", "修改成功！")
}

// putStorageDefault - 修改存储默认服务类型
func (this *Toml) putStorageDefault(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["value"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "value"), 400)
		return
	}

	allow := []any{"local", "cos"}

	if !utils.In.Array(params["value"], allow) {
		this.json(ctx, nil, facade.Lang(ctx, "value 只允许是 local、cos 其中一个！"), 400)
		return
	}

	replaceMap := this.storageConfigToReplaceMap()
	replaceMap["${default}"] = params["value"]

	temp := facade.TempStorage
	temp = utils.Replace(temp, replaceMap)

	this.saveTomlConfig(ctx, temp, "config/storage.toml", "修改成功！")
}

// putStorageLocal - 修改本地存储配置
func (this *Toml) putStorageLocal(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx, map[string]any{
		"domain": this.get(ctx, "domain"),
	})

	replaceMap := this.storageConfigToReplaceMap()
	if v, ok := params["domain"]; ok {
		replaceMap["${local.domain}"] = v
	}
	if v, ok := params["path"]; ok {
		replaceMap["${local.path}"] = v
	}

	// 上传命名规则（目录结构 / 文件名）：未知占位符直接拒绝
	if !this.validateStorageRules(ctx, "本地存储", params) {
		return
	}
	for _, kind := range []string{facade.StorageRuleDir, facade.StorageRuleFile} {
		if v, ok := params[kind]; ok {
			replaceMap["${local."+kind+"}"] = v
		}
	}

	temp := facade.TempStorage
	temp = utils.Replace(temp, replaceMap)

	this.saveTomlConfig(ctx, temp, "config/storage.toml", "修改成功！")
}

// testCOS - 测试COS连接
func (this *Toml) testCOS(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["secret_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_id"), 400)
		return
	}

	if utils.Is.Empty(params["secret_key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_key"), 400)
		return
	}

	if utils.Is.Empty(params["app_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "app_id"), 400)
		return
	}

	if utils.Is.Empty(params["bucket"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bucket"), 400)
		return
	}

	if utils.Is.Empty(params["region"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "region"), 400)
		return
	}

	appId := cast.ToString(params["app_id"])
	secretId := cast.ToString(params["secret_id"])
	secretKey := cast.ToString(params["secret_key"])
	bucket := cast.ToString(params["bucket"])
	region := cast.ToString(params["region"])

	// 桶名归一化：bucket 可填裸桶名（自动补 -app_id），也可填控制台复制的全名
	BucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com",
		facade.COSBucketNameWith(bucket, appId),
		region,
	))
	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试COS连接失败！"), 400)
		return
	}

	client := cos.NewClient(&cos.BaseURL{
		BucketURL: BucketURL,
	}, &http.Client{
		// 设置超时时间
		Timeout: 100 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretId,
			SecretKey: secretKey,
		},
	})

	// 查询存储桶
	exist, err := client.Bucket.IsExist(context.Background())

	if err != nil {
		this.json(ctx, err.Error(), facade.Lang(ctx, "测试COS连接失败！"), 400)
		return
	}

	if !exist {
		this.json(ctx, nil, facade.Lang(ctx, "Bucket 不存在！"), 400)
		return
	}

	this.json(ctx, nil, facade.Lang(ctx, "测试COS连接成功！"), 200)
}

// putStorageCOS - 修改COS存储配置
func (this *Toml) putStorageCOS(ctx *gin.Context) {

	// 请求参数
	params := this.params(ctx)

	// 还原脱敏占位值（避免把脱敏密钥写回配置）
	this.restoreSecretParams(params, []string{"secret_key", "secret_id"}, cast.ToStringMap(facade.StorageToml.Get("cos")))

	if utils.Is.Empty(params["secret_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_id"), 400)
		return
	}

	if utils.Is.Empty(params["secret_key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "secret_key"), 400)
		return
	}

	if utils.Is.Empty(params["app_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "app_id"), 400)
		return
	}

	if utils.Is.Empty(params["bucket"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "bucket"), 400)
		return
	}

	if utils.Is.Empty(params["region"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "region"), 400)
		return
	}

	replaceMap := this.storageConfigToReplaceMap()
	replaceMap["${cos.secret_id}"] = cast.ToString(params["secret_id"])
	replaceMap["${cos.secret_key}"] = cast.ToString(params["secret_key"])
	replaceMap["${cos.app_id}"] = cast.ToString(params["app_id"])
	replaceMap["${cos.bucket}"] = cast.ToString(params["bucket"])
	replaceMap["${cos.region}"] = cast.ToString(params["region"])
	if v, ok := params["domain"]; ok {
		replaceMap["${cos.domain}"] = cast.ToString(v)
	}
	if v, ok := params["path"]; ok {
		replaceMap["${cos.path}"] = cast.ToString(v)
	}

	// 上传命名规则（目录结构 / 文件名）：未知占位符直接拒绝
	if !this.validateStorageRules(ctx, "腾讯云 COS", params) {
		return
	}
	for _, kind := range []string{facade.StorageRuleDir, facade.StorageRuleFile} {
		if v, ok := params[kind]; ok {
			replaceMap["${cos."+kind+"}"] = v
		}
	}

	temp := facade.TempStorage
	temp = utils.Replace(temp, replaceMap)

	this.saveTomlConfig(ctx, temp, "config/storage.toml", "修改成功！")
}

// putStorageAttachment - 修改附件配置
func (this *Toml) putStorageAttachment(ctx *gin.Context) {

	params := this.params(ctx)

	// 只有这三项真正生效（原先的 limit_per_minute 等 5 个「每时段上限」后端从未校验，已移除）
	allowFields := map[string]string{
		"allow_extensions": "${attachment.allow_extensions}",
		"max_file_size":    "${attachment.max_file_size}",
		"concurrent_limit": "${attachment.concurrent_limit}",
	}

	replaceMap := this.storageConfigToReplaceMap()
	hasUpdate := false
	for field, placeholder := range allowFields {
		if val, ok := params[field]; ok {
			replaceMap[placeholder] = val
			hasUpdate = true
		}
	}

	if !hasUpdate {
		this.json(ctx, nil, facade.Lang(ctx, "请提供要修改的配置参数！"), 400)
		return
	}

	temp := facade.TempStorage
	temp = utils.Replace(temp, replaceMap)

	this.saveTomlConfig(ctx, temp, "config/storage.toml", "修改成功！")

	go func() {
		facade.InitAttachmentConfig()
	}()
}

// putStorage - 统一修改存储配置
func (this *Toml) putStorage(ctx *gin.Context) {
	params := this.params(ctx)

	replaceMap := this.storageConfigToReplaceMap()

	if val, ok := params["default"]; ok {
		allow := []any{"local", "cos"}
		if !utils.In.Array(val, allow) {
			this.json(ctx, nil, facade.Lang(ctx, "default 只允许是 local、cos 其中一个！"), 400)
			return
		}
		replaceMap["${default}"] = val
	}

	if local, ok := params["local"].(map[string]any); ok {
		if v, ok := local["domain"]; ok {
			replaceMap["${local.domain}"] = v
		}
		if v, ok := local["path"]; ok {
			replaceMap["${local.path}"] = v
		}
		// 上传命名规则（目录结构 / 文件名）：未知占位符直接拒绝
		if !this.validateStorageRules(ctx, "本地存储", local) {
			return
		}
		for _, kind := range []string{facade.StorageRuleDir, facade.StorageRuleFile} {
			if v, ok := local[kind]; ok {
				replaceMap["${local."+kind+"}"] = v
			}
		}
	}

	if cos, ok := params["cos"].(map[string]any); ok {
		if v, ok := cos["secret_id"]; ok {
			replaceMap["${cos.secret_id}"] = v
		}
		if v, ok := cos["secret_key"]; ok {
			replaceMap["${cos.secret_key}"] = v
		}
		if v, ok := cos["app_id"]; ok {
			replaceMap["${cos.app_id}"] = v
		}
		if v, ok := cos["bucket"]; ok {
			replaceMap["${cos.bucket}"] = v
		}
		if v, ok := cos["region"]; ok {
			replaceMap["${cos.region}"] = v
		}
		if v, ok := cos["domain"]; ok {
			replaceMap["${cos.domain}"] = v
		}
		if v, ok := cos["path"]; ok {
			replaceMap["${cos.path}"] = v
		}
		// 上传命名规则（目录结构 / 文件名）：未知占位符直接拒绝
		if !this.validateStorageRules(ctx, "腾讯云 COS", cos) {
			return
		}
		for _, kind := range []string{facade.StorageRuleDir, facade.StorageRuleFile} {
			if v, ok := cos[kind]; ok {
				replaceMap["${cos."+kind+"}"] = v
			}
		}
	}

	if attachment, ok := params["attachment"].(map[string]any); ok {
		if v, ok := attachment["allow_extensions"]; ok {
			replaceMap["${attachment.allow_extensions}"] = v
		}
		if v, ok := attachment["max_file_size"]; ok {
			replaceMap["${attachment.max_file_size}"] = v
		}
		if v, ok := attachment["concurrent_limit"]; ok {
			replaceMap["${attachment.concurrent_limit}"] = v
		}
	}

	temp := facade.TempStorage
	temp = utils.Replace(temp, replaceMap)

	this.saveTomlConfig(ctx, temp, "config/storage.toml", "修改成功！")
}

// getNotification - 获取通知配置
func (this *Toml) getNotification(ctx *gin.Context) {
	retentionDays := cast.ToInt(facade.AppToml.Get("notification.retention_days", 30))
	this.json(ctx, map[string]any{
		"retention_days": retentionDays,
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}

// putNotification - 修改通知配置（保留天数）
func (this *Toml) putNotification(ctx *gin.Context) {
	params := this.params(ctx)

	retentionDays := cast.ToInt(params["retention_days"])
	if retentionDays <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 必须大于 0！", "retention_days"), 400)
		return
	}

	const appTomlPath = "config/app.toml"

	// 读取现有配置（文件不存在时用模板兜底）
	content := facade.TempApp
	if utils.File().Exist(appTomlPath) {
		if bytes := utils.File().Byte(appTomlPath).Byte; len(bytes) > 0 {
			content = string(bytes)
		}
	}

	newValue := fmt.Sprintf("retention_days = %d", retentionDays)

	switch {
	case strings.Contains(content, "retention_days"):
		// 已有 retention_days 键，替换其值
		content = retentionDaysPattern.ReplaceAllString(content, newValue)
	case strings.Contains(content, "[notification]"):
		// 有 [notification] 段但无 retention_days，在段名后插入
		content = strings.Replace(content, "[notification]", fmt.Sprintf("[notification]\n%s", newValue), 1)
	default:
		// 无 [notification] 段，追加到文件末尾
		content = strings.TrimRight(content, "\n") + fmt.Sprintf("\n\n[notification]\n%s\n", newValue)
	}

	this.saveTomlConfig(ctx, content, appTomlPath, "修改成功！")
}

// getSensitive - 读取敏感词库（解析结果 + 文件原文）
//
// 后台用原文编辑：规则带 action / scenes / level / match，结构化表单会很快失控，
// 直接编辑 TOML 反而更透明；解析结果同时返回，便于前端展示概览。
func (this *Toml) getSensitive(ctx *gin.Context) {
	this.sensitiveResult(ctx, facade.Lang(ctx, "数据请求成功！"))
}

// putSensitive - 保存敏感词库（同一接口，两种提交方式）
//
//   - { content }：TOML 原文，保留手写注释（后台「TOML 源码」模式）；
//   - { enabled, default_action, rules, whitelist }：结构化数据，
//     服务端按模板生成规范 TOML（后台「可视化配置」模式）。
//
// 两条路径共用同一套字段校验、备份、版本自增与热更新，权限点也只需要一个。
func (this *Toml) putSensitive(ctx *gin.Context) {

	params := this.params(ctx)

	if content := strings.TrimRight(cast.ToString(params["content"]), "\r\n"); strings.TrimSpace(content) != "" {
		this.putSensitiveSource(ctx, content)
		return
	}

	// 表单模式至少要带一个结构化字段：空载荷不能当成「清空词库」执行
	_, hasRules := params["rules"]
	_, hasWhitelist := params["whitelist"]
	_, hasEnabled := params["enabled"]
	_, hasAction := params["default_action"]
	if !hasRules && !hasWhitelist && !hasEnabled && !hasAction {
		this.json(ctx, nil, facade.Lang(ctx, "配置内容不能为空！"), 400)
		return
	}

	this.putSensitiveForm(ctx, params)
}

// putSensitiveSource - 保存 TOML 原文（源码模式）
//
// 保存前用 TOML 解析器校验语法，避免写入后服务启动失败；
// 旧文件先备份为 sensitive.toml.bak，保存后同步重载（不依赖文件监听时序）。
func (this *Toml) putSensitiveSource(ctx *gin.Context, content string) {

	var parsed map[string]any
	if _, err := toml.Decode(content, &parsed); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "TOML 语法错误：%v", err.Error()), 400)
		return
	}
	if err := validateSensitiveRules(parsed); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "%v", err.Error()), 400)
		return
	}

	if !this.saveSensitiveContent(ctx, content) {
		return
	}

	this.sensitiveResult(ctx, facade.Lang(ctx, "保存成功！"))
}

// sensitiveResult - 统一的敏感词库响应（解析结果 + 文件原文）
func (this *Toml) sensitiveResult(ctx *gin.Context, msg string) {
	data := (&model.Sensitive{}).Config()
	data["content"] = facade.SensitiveFileContent()
	this.json(ctx, data, msg, 200)
}

// saveSensitiveContent - 备份旧文件 → 版本自增 → 写盘 → 重载
//
// 源码保存与可视化表单保存共用，保证两条路径的回滚与版本行为一致。
func (this *Toml) saveSensitiveContent(ctx *gin.Context, content string) bool {

	// 版本号自增：命中审计记录 version，才能回溯「某次误伤是哪一版规则判的」
	content = bumpSensitiveVersion(content)

	// 备份旧文件：单步回滚足够覆盖「保存后发现大面积误伤」的场景
	if err := facade.BackupSensitiveToml(); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "备份失败，已取消保存！"), 400)
		return false
	}

	if err := facade.SaveSensitiveToml(content); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "保存失败！"), 400)
		return false
	}

	return true
}

// sensitiveRuleForm - 可视化表单里的一条规则（与 TOML 的 [[rules]] 同构）
type sensitiveRuleForm struct {
	Word   string
	Action string
	Scenes []string
	Level  string
	Match  string
}

// putSensitiveForm - 可视化表单保存词库
//
// 表单模式按模板重新生成 TOML（源码里的手写注释不会保留），
// 需要保留注释请用「TOML 源码」模式；校验 / 备份 / 版本自增 / 热更新与源码保存一致。
func (this *Toml) putSensitiveForm(ctx *gin.Context, params map[string]any) {

	defaultAction := strings.ToLower(strings.TrimSpace(cast.ToString(params["default_action"])))
	if defaultAction == "" {
		defaultAction = model.ActionReview
	}
	switch defaultAction {
	case model.ActionBlock, model.ActionLimit, model.ActionReview, model.ActionWarn, model.ActionLog:
	default:
		this.json(ctx, nil, facade.Lang(ctx, "缺省动作不支持：%s", defaultAction), 400)
		return
	}

	rules := parseSensitiveFormRows(params["rules"])
	whitelist := parseSensitiveFormRows(params["whitelist"])

	// 复用源码保存的字段校验：表单同样不允许写错的 action / level / scene / 超长词条
	if err := validateSensitiveRules(map[string]any{
		"rules":     sensitiveFormRows(rules),
		"whitelist": sensitiveFormRows(whitelist),
	}); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "%v", err.Error()), 400)
		return
	}

	// 版本号沿用当前文件（随后统一自增），避免版本链断开
	version := "1"
	if current, ok := sensitiveVersionNumber(facade.SensitiveFileContent()); ok {
		version = strconv.Itoa(current)
	}

	content := renderSensitiveToml(cast.ToBool(params["enabled"]), defaultAction, version, rules, whitelist)

	if !this.saveSensitiveContent(ctx, content) {
		return
	}

	this.sensitiveResult(ctx, facade.Lang(ctx, "保存成功！"))
}

// parseSensitiveFormRows - 把表单提交的数组转成内部结构
func parseSensitiveFormRows(value any) []sensitiveRuleForm {

	rows := cast.ToSlice(value)
	if len(rows) == 0 {
		return nil
	}

	result := make([]sensitiveRuleForm, 0, len(rows))
	for _, item := range rows {
		row := cast.ToStringMap(item)

		scenes := make([]string, 0, 5)
		for _, scene := range cast.ToStringSlice(row["scenes"]) {
			scene = strings.ToLower(strings.TrimSpace(scene))
			if scene != "" {
				scenes = append(scenes, scene)
			}
		}

		result = append(result, sensitiveRuleForm{
			Word:   strings.TrimSpace(cast.ToString(row["word"])),
			Action: strings.ToLower(strings.TrimSpace(cast.ToString(row["action"]))),
			Scenes: scenes,
			Level:  strings.ToLower(strings.TrimSpace(cast.ToString(row["level"]))),
			Match:  strings.ToLower(strings.TrimSpace(cast.ToString(row["match"]))),
		})
	}

	return result
}

// sensitiveFormRows - 转成 validateSensitiveRules 认识的 []map[string]any
func sensitiveFormRows(rules []sensitiveRuleForm) []map[string]any {

	rows := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		rows = append(rows, map[string]any{
			"word":   rule.Word,
			"action": rule.Action,
			"scenes": rule.Scenes,
			"level":  rule.Level,
			"match":  rule.Match,
		})
	}

	return rows
}

// renderSensitiveToml - 把表单结构渲染成带注释的 TOML 原文
//
// match = contains 与 level = medium 是默认值，省略后解析结果等价，文件也更短。
func renderSensitiveToml(enabled bool, defaultAction, version string, rules, whitelist []sensitiveRuleForm) string {

	var builder strings.Builder

	builder.WriteString("# ======== 全站敏感词库 ========\n")
	builder.WriteString("# 由后台「敏感词库 → 可视化配置」生成；需要手写注释请切到「TOML 源码」模式。\n")
	builder.WriteString("# 规则字段：word 词条 / action 处置 / scenes 场景 / level 等级 / match 匹配方式\n\n")

	builder.WriteString("# 总开关\n")
	builder.WriteString(fmt.Sprintf("enabled = %t\n\n", enabled))

	builder.WriteString("# 规则缺省动作（规则未写 action 时使用）\n")
	builder.WriteString(fmt.Sprintf("default_action = %s\n\n", strconv.Quote(defaultAction)))

	builder.WriteString("# 规则版本（每次保存自动 +1，命中审计据此回溯误伤）\n")
	builder.WriteString(fmt.Sprintf("version = %s\n", strconv.Quote(version)))

	if len(whitelist) > 0 {
		builder.WriteString("\n# ======== 白名单（命中即放行） ========\n")
		for _, rule := range whitelist {
			builder.WriteString("\n[[whitelist]]\n")
			builder.WriteString(fmt.Sprintf("word = %s\n", strconv.Quote(rule.Word)))
			if len(rule.Scenes) > 0 {
				builder.WriteString(fmt.Sprintf("scenes = %s\n", quoteTomlArray(rule.Scenes)))
			}
			if rule.Match != "" && rule.Match != "contains" {
				builder.WriteString(fmt.Sprintf("match = %s\n", strconv.Quote(rule.Match)))
			}
		}
	}

	if len(rules) > 0 {
		builder.WriteString("\n# ======== 词条规则 ========\n")
		for _, rule := range rules {
			builder.WriteString("\n[[rules]]\n")
			builder.WriteString(fmt.Sprintf("word = %s\n", strconv.Quote(rule.Word)))
			if rule.Action != "" {
				builder.WriteString(fmt.Sprintf("action = %s\n", strconv.Quote(rule.Action)))
			}
			if len(rule.Scenes) > 0 {
				builder.WriteString(fmt.Sprintf("scenes = %s\n", quoteTomlArray(rule.Scenes)))
			}
			if rule.Level != "" && rule.Level != "medium" {
				builder.WriteString(fmt.Sprintf("level = %s\n", strconv.Quote(rule.Level)))
			}
			if rule.Match != "" && rule.Match != "contains" {
				builder.WriteString(fmt.Sprintf("match = %s\n", strconv.Quote(rule.Match)))
			}
		}
	}

	return strings.TrimRight(builder.String(), "\n")
}

// quoteTomlArray - 字符串数组字面量
func quoteTomlArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// sensitiveVersionPattern - 顶层 version = "1" 配置行
var sensitiveVersionPattern = regexp.MustCompile(`(?m)^version\s*=\s*"?(?P<number>\d+)"?\s*$`)

// sensitiveVersionNumber - 解析数字版本号；缺失或非数字返回 false
//
// 版本号允许管理员手工维护成日期 / v1 之类，这类值不参与自动递增。
func sensitiveVersionNumber(content string) (int, bool) {

	matches := sensitiveVersionPattern.FindStringSubmatch(content)
	if len(matches) != 2 {
		return 0, false
	}

	version, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	return version, true
}

// setSensitiveVersion - 把配置里的版本号改写成指定值
func setSensitiveVersion(content string, version int) string {
	return sensitiveVersionPattern.ReplaceAllString(content, fmt.Sprintf(`version = "%d"`, version))
}

// bumpSensitiveVersion - 保存时版本号自增；非数字版本保持原样（由管理员手工维护）
func bumpSensitiveVersion(content string) string {

	version, ok := sensitiveVersionNumber(content)
	if !ok {
		return content
	}

	return setSensitiveVersion(content, version+1)
}

// rollbackVersion - 回滚时的版本号
//
// 回滚不能让版本号倒退：v1 保存成 v2 后回滚到 v1，若原样写回 v1，
// 下次保存又会自增成 v2，审计里同一个 v2 就对应了两份不同内容，误伤回溯会指错版本。
// 取 max(备份版本, 回滚前版本) + 1，保证版本号单调递增。
func rollbackVersion(backup string, current string) string {

	version, ok := sensitiveVersionNumber(backup)
	if !ok {
		return backup
	}

	if currentVersion, ok := sensitiveVersionNumber(current); ok && currentVersion > version {
		version = currentVersion
	}

	return setSensitiveVersion(backup, version+1)
}

// validateSensitiveRules - 保存前检查规则字段
//
// 只做「明显写错」的拦截：缺 word、action / match / scenes 拼错。
// 拼错的 action 如果直接放进去，运行时会静默回落到 default_action，
// 管理员会以为规则生效了 —— 这类静默失败比 400 更危险。
func validateSensitiveRules(parsed map[string]any) error {

	allowedActions := map[string]bool{
		model.ActionBlock: true, model.ActionLimit: true, model.ActionReview: true,
		model.ActionWarn: true, model.ActionLog: true,
		// 白名单的结构与规则同构，常被顺手写成 action = "allow"；
		// 运行时 normalizeAction 本来就接受它，校验口径不能更严
		model.ActionAllow: true,
	}
	allowedMatch := map[string]bool{"contains": true, "exact": true}
	allowedScenes := map[string]bool{
		model.SceneComment: true, model.SceneArticle: true, model.SceneMoment: true,
		model.SceneDM: true, model.SceneChat: true,
	}
	allowedLevels := map[string]bool{
		model.LevelHigh: true, model.LevelMedium: true, model.LevelLow: true,
	}

	for _, key := range []string{"rules", "whitelist"} {
		raw, ok := parsed[key]
		if !ok || raw == nil {
			continue
		}

		rows, ok := raw.([]map[string]any)
		if !ok {
			return fmt.Errorf("%s 必须是数组（[[%s]]）", key, key)
		}

		for index, row := range rows {
			position := fmt.Sprintf("%s[%d]", key, index+1)

			word := strings.TrimSpace(cast.ToString(row["word"]))
			if word == "" {
				return fmt.Errorf("%s 缺少 word", position)
			}
			if len([]rune(word)) > model.SensitiveWordMaxRunes {
				return fmt.Errorf("%s 的 word 超过 %d 个字符，超长规则运行时会忽略", position, model.SensitiveWordMaxRunes)
			}
			if action := strings.ToLower(strings.TrimSpace(cast.ToString(row["action"]))); action != "" && !allowedActions[action] {
				return fmt.Errorf("%s 的 action 不支持：%s", position, action)
			}
			if match := strings.ToLower(strings.TrimSpace(cast.ToString(row["match"]))); match != "" && !allowedMatch[match] {
				return fmt.Errorf("%s 的 match 不支持：%s", position, match)
			}
			if level := strings.ToLower(strings.TrimSpace(cast.ToString(row["level"]))); level != "" && !allowedLevels[level] {
				return fmt.Errorf("%s 的 level 不支持：%s", position, level)
			}
			for _, scene := range cast.ToStringSlice(row["scenes"]) {
				scene = strings.ToLower(strings.TrimSpace(scene))
				if scene != "" && !allowedScenes[scene] {
					return fmt.Errorf("%s 的 scene 不支持：%s", position, scene)
				}
			}
		}
	}

	return nil
}

// rollbackSensitive - 回滚到上一次保存前的词库
//
// 备份同样会被加载进内存：先校验、再推进版本号，最后写回并重载，
// 避免「回滚后整份词库失效」和「同一个版本号对应两份内容」两种静默故障。
func (this *Toml) rollbackSensitive(ctx *gin.Context) {

	backup, ok := facade.SensitiveBackupContent()
	if !ok {
		this.json(ctx, nil, facade.Lang(ctx, "没有可回滚的备份！"), 400)
		return
	}

	var parsed map[string]any
	if _, err := toml.Decode(backup, &parsed); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "备份文件语法错误，无法回滚：%v", err.Error()), 400)
		return
	}
	if err := validateSensitiveRules(parsed); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "备份文件规则不合法，无法回滚：%v", err.Error()), 400)
		return
	}

	content := rollbackVersion(backup, facade.SensitiveFileContent())
	if err := facade.SaveSensitiveToml(content); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "回滚失败！"), 400)
		return
	}

	this.sensitiveResult(ctx, facade.Lang(ctx, "已回滚到上一次保存前的版本！"))
}

// testSensitive - 用当前词库试跑一条文本（不写命中审计）
func (this *Toml) testSensitive(ctx *gin.Context) {

	params := this.params(ctx)

	scene := strings.ToLower(strings.TrimSpace(cast.ToString(params["scene"])))
	switch scene {
	case model.SceneComment, model.SceneArticle, model.SceneMoment, model.SceneDM, model.SceneChat:
	default:
		this.json(ctx, nil, facade.Lang(ctx, "不支持的场景！"), 400)
		return
	}

	text := cast.ToString(params["text"])
	decision := (&model.Sensitive{}).Decide(scene, 0, text)

	this.json(ctx, gin.H{
		"scene":       scene,
		"text_length": len([]rune(text)),
		"decision":    decision,
	}, facade.Lang(ctx, "测试完成！"), 200)
}

// sensitiveStats - 最近 7 天命中统计（按场景 + 动作）
func (this *Toml) sensitiveStats(ctx *gin.Context) {
	this.json(ctx, gin.H{
		"days":  7,
		"items": model.SensitiveHitStats(7),
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}
