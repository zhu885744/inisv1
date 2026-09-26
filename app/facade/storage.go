package facade

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cast"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/unti-io/go-utils/utils"
)

func init() {

	// 初始化配置文件
	initStorageToml()
	// 初始化存储
	initStorage()

	// 监听配置文件变化
	StorageToml.Viper.WatchConfig()
	// 配置文件变化时，重新初始化配置文件
	StorageToml.Viper.OnConfigChange(func(event fsnotify.Event) {
		initStorage()
	})
}

const (
	// StorageModeLocal - 本地存储
	StorageModeLocal = "local"
	// StorageModeCOS - 腾讯云 COS 存储
	StorageModeCOS = "cos"
)

// NewStorage - 创建Storage实例
/**
 * @param mode 驱动模式：local / cos（其它值回退本地存储）
 * @return StorageInterface
 * @example：
 * 1. storage := facade.NewStorage("cos")
 * 2. storage := facade.NewStorage(facade.StorageModeCOS)
 */
func NewStorage(mode any) StorageInterface {
	switch strings.ToLower(cast.ToString(mode)) {
	case StorageModeLocal:
		Storage = LocalStorage
	case StorageModeCOS:
		Storage = COS
	default:
		Storage = LocalStorage
	}
	return Storage
}

// StorageToml - 存储配置文件
var StorageToml *utils.ViperResponse

// initStorageToml - 初始化存储配置文件
func initStorageToml() {
	item := utils.Viper(utils.ViperModel{
		Path: "config",
		Mode: "toml",
		Name: "storage",
		Content: utils.Replace(TempStorage, map[string]any{
			"${default}": "local",
			// 本地域名留空：full_url 存相对路径 /storage/xxx，直接由站点静态服务提供；
			// 之前默认成 "storage" 会拼出 storage/storage/xxx 这种无效地址
			"${local.domain}":                "",
			"${local.path}":                  "storage",
			"${local.dir_rule}":              DefaultStorageDirRule,
			"${local.file_rule}":             DefaultStorageFileRule,
			"${cos.app_id}":                  "",
			"${cos.secret_id}":               "",
			"${cos.secret_key}":              "",
			"${cos.bucket}":                  "inis-cos",
			"${cos.region}":                  "ap-guangzhou",
			"${cos.domain}":    "",
			"${cos.path}":      "inis",
			"${cos.dir_rule}":  DefaultStorageDirRule,
			"${cos.file_rule}": DefaultStorageFileRule,
			"${attachment.allow_extensions}": "jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md",
			"${attachment.max_file_size}":    51200,
			"${attachment.concurrent_limit}": 5,
		}),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "存储配置初始化错误")
		return
	}

	StorageToml = &item
}

// ReloadStorageToml - 重新读取存储配置文件
func ReloadStorageToml() {
	item := utils.Viper(utils.ViperModel{
		Path: "config",
		Mode: "toml",
		Name: "storage",
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "重新读取存储配置文件错误")
		return
	}

	StorageToml = &item
	initStorage()
}

// 初始化存储（仅本地存储 local 与腾讯云 COS 两种驱动）
func initStorage() {

	// 腾讯云 COS 对象存储
	COS = &COSStruct{}
	COS.init()

	// 本地存储
	LocalStorage = &LocalStorageStruct{}

	switch cast.ToString(StorageToml.Get("default")) {
	case StorageModeCOS:
		Storage = COS
	default:
		Storage = LocalStorage
	}

	// 初始化附件配置
	InitAttachmentConfig()
}

// Storage - Storage实例
/**
 * @return StorageInterface
 * @example：
 * ctx := facade.StorageRuleContext{Uid: int(uid), Filename: "photo", Ext: "jpg"}
 * storage := facade.Storage.Upload(facade.Storage.Path(ctx), bytes) // 对象键已含扩展名
 */
var Storage StorageInterface
var LocalStorage *LocalStorageStruct
var COS *COSStruct

// =================================== 附件配置 - 开始 ===================================

// AttachmentConfig - 附件配置
//
// 只有这三项真正参与上传校验（见 attachment.go 的 uploadSingleFile / checkUploadLimit）；
// 历史上还留过 limit_per_minute / limit_per_hour / limit_per_day / limit_per_week /
// limit_per_month 五个「每时段上传上限」，但从未实现校验逻辑，已移除。
type AttachmentConfig struct {
	AllowExtensions []string // 允许的文件扩展名
	MaxFileSize     int64    // 单个文件最大大小（KB）
	ConcurrentLimit int      // 并发上传限制
}

// AttachmentConfigInstance - 附件配置实例
var AttachmentConfigInstance *AttachmentConfig

// InitAttachmentConfig - 初始化附件配置
func InitAttachmentConfig() {
	AttachmentConfigInstance = &AttachmentConfig{
		AllowExtensions: parseExtensions(cast.ToString(StorageToml.Get("attachment.allow_extensions"))),
		MaxFileSize:     cast.ToInt64(StorageToml.Get("attachment.max_file_size")),
		ConcurrentLimit: cast.ToInt(StorageToml.Get("attachment.concurrent_limit")),
	}
}

// parseExtensions - 解析扩展名字符串
func parseExtensions(extensions string) []string {
	if extensions == "" {
		return []string{}
	}
	parts := strings.Split(extensions, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(strings.ToLower(part))
	}
	return parts
}

// IsExtensionAllowed - 检查扩展名是否允许
func (this *AttachmentConfig) IsExtensionAllowed(ext string) bool {
	if len(this.AllowExtensions) == 0 {
		return true
	}
	ext = strings.ToLower(ext)
	for _, allowed := range this.AllowExtensions {
		if allowed == ext {
			return true
		}
	}
	return false
}

// GetMaxFileSizeBytes - 获取最大文件大小（字节）
func (this *AttachmentConfig) GetMaxFileSizeBytes() int64 {
	return this.MaxFileSize * 1024
}

type StorageResponse struct {
	Error  error
	Path   string
	Domain string
}

type StorageInterface interface {
	Upload(key string, reader io.Reader) *StorageResponse
	Delete(key string) error
	DeleteMulti(keys []string) error
	// Path 按「目录命名规则 + 文件命名规则」生成对象键（已含扩展名）
	Path(ctx StorageRuleContext) string
}

// =================================== 存储公共工具 ===================================

// storageDeleteBatch 单次批量删除的对象数上限
// COS 的 DeleteMulti 单请求最多 1000 个对象（见官方文档），这里统一按 500 分批，留出余量
const storageDeleteBatch = 500

// NormalizeStorageKey 清理对象键：去空白、去前导斜杠
//
// 历史上前缀（如 cos.path）为空时会生成 "/2026-09/26/xxx" 这种键：
// 云存储里会多出一层空目录，拼上 CDN 域名后还会出现 "//" 导致 404。
// 上传与删除两侧统一走这里，保证键完全一致。
func NormalizeStorageKey(key string) string {
	return strings.TrimLeft(strings.TrimSpace(key), "/")
}

// StorageKey 生成对象键：前缀 + 目录 + 文件名（各段自动清理多余斜杠）
//
// 目录与文件名由命名规则生成（见 storage-rule.go），这里只做拼装。
// 保留该名字供历史调用方使用，新代码可直接用 StorageObjectKey。
func StorageKey(prefix, dir, name string) string {
	return StorageObjectKey(prefix, dir, name)
}

// cosToml 读取 COS 配置项（配置文件未就绪时返回空，避免空指针）
func cosToml(key string) string {
	if StorageToml == nil {
		return ""
	}
	return cast.ToString(StorageToml.Get(key))
}

// COSBucketNameWith 归一化「桶名 + app_id」（表单参数或配置值都走这里）
//
// 兼容两种写法，避免拼出 inis-cos-1250000000-1250000000.cos... 这类错误域名：
//   - bucket = "inis-cos"、app_id = "1250000000"        → inis-cos-1250000000
//   - bucket = "inis-cos-1250000000"（控制台复制的全名） → 原样使用
func COSBucketNameWith(bucket, appId string) string {
	bucket = strings.TrimSpace(bucket)
	appId = strings.TrimSpace(appId)

	if bucket == "" {
		return ""
	}
	if appId == "" || strings.HasSuffix(bucket, "-"+appId) {
		return bucket
	}

	return bucket + "-" + appId
}

// COSBucketName 归一化配置里的 COS 桶名
func COSBucketName() string {
	return COSBucketNameWith(cosToml("cos.bucket"), cosToml("cos.app_id"))
}

// COSRegion COS 地域（未配置时回退广州）
func COSRegion() string {
	region := strings.TrimSpace(cosToml("cos.region"))
	if region == "" {
		return "ap-guangzhou"
	}
	return region
}

// COSDomain COS 默认访问域名
//
// 未配置自定义域名时，上传（写 full_url 模板）与查询（展开 {{cos}}）都用它，
// 两侧口径必须一致，否则附件地址会出现「存的域名和显示的域名不一样」。
func COSDomain() string {
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com", COSBucketName(), COSRegion())
}

// =================================== 本地存储存储 - 开始 ===================================

// LocalStorageStruct 本地存储
type LocalStorageStruct struct{}

// Upload - 上传文件
func (this *LocalStorageStruct) Upload(path string, reader io.Reader) (result *StorageResponse) {

	result = &StorageResponse{}

	item := utils.File().Save(reader, path)

	if item.Error != nil {
		result.Error = item.Error
		return
	}

	// 对外路径去掉 public/ 前缀（web 根目录是 public，由站点静态服务映射）
	// 用 TrimPrefix 而不是 Replace：目录名里若含 "public"（如 public-files），
	// Replace 会把中间那段一起删掉，导致 save_path 与实际文件对不上
	result.Path = "/" + strings.Trim(strings.TrimPrefix(path, "public"), "/")

	// 本地域名：留空表示用相对路径（/storage/xxx），也可以填 https://cdn.xxx.com
	domain := strings.TrimRight(strings.TrimSpace(cast.ToString(StorageToml.Get("local.domain"))), "/")
	localPath := strings.Trim(strings.TrimSpace(cast.ToString(StorageToml.Get("local.path"))), "/")

	// 兼容历史默认值：模板里 domain 与 path 都曾是 "storage"，会拼出 storage/storage/xxx。
	// 这种「不含点的短名 + 与 path 相同」的配置一律按未配置处理，避免附件地址失效。
	if domain != "" && domain == localPath && !strings.Contains(domain, ".") {
		domain = ""
	}

	result.Domain = domain

	return
}

// Path - 本地存储位置 - 按命名规则生成文件路径（public 目录下的相对路径）
//
// 目录与文件名都来自 config/storage.toml 的 [local] 段（dir_rule / file_rule，
// 见 storage-rule.go），local.path 是固定前缀；例如：
//
//	storage + {Y}-{m}/{d} + {timestamp}{str-random-10} + .jpg
//	→ public/storage/2026-09/26/1758888888123abc7def.jpg
func (this *LocalStorageStruct) Path(ctx StorageRuleContext) string {

	values := NewStorageRuleValues(ctx)

	dir := values.Apply(StorageRule("local", StorageRuleDir))
	name := values.Apply(StorageRule("local", StorageRuleFile))

	// local.path 为空时不带前导斜杠，避免拼出 "public//2026-09/…"
	return "public/" + StorageObjectKey(
		cast.ToString(StorageToml.Get("local.path")),
		dir,
		StorageNameWithExt(name, ctx.Ext),
	)
}

// Delete - 删除文件
func (this *LocalStorageStruct) Delete(key string) error {
	path := strings.TrimPrefix(key, "/")
	if !strings.HasPrefix(path, "public/") {
		path = "public/" + path
	}

	// 路径规范化并校验，防止目录穿越（如 ../../）删除 public 目录之外的任意文件
	clean := filepath.Clean(path)
	if clean != "public" && !strings.HasPrefix(clean, "public"+string(filepath.Separator)) {
		return errors.New("非法的文件路径！")
	}

	err := os.Remove(clean)
	// 文件本来就不存在视为删除成功（幂等）：避免「已删过的文件」在清空回收站时
	// 被记为失败，导致对应记录一直留在回收站里
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}

// DeleteMulti - 批量删除文件
func (this *LocalStorageStruct) DeleteMulti(keys []string) error {
	for _, key := range keys {
		if err := this.Delete(key); err != nil {
			Log.Error(map[string]any{
				"error": err,
				"key":   key,
			}, "本地存储批量删除文件失败")
		}
	}
	return nil
}

// ================================== 腾讯云对象存储 - 开始 ==================================

// COSStruct 腾讯云对象存储
type COSStruct struct {
	Client *cos.Client

	// bucketReady 是否已确认存储桶存在（确认过就不再每次上传都查一次桶）
	bucketReady bool
	bucketMu    sync.Mutex
}

// init 初始化 腾讯云对象存储
func (this *COSStruct) init() {

	secretId := strings.TrimSpace(cast.ToString(StorageToml.Get("cos.secret_id")))
	secretKey := strings.TrimSpace(cast.ToString(StorageToml.Get("cos.secret_key")))
	bucket := COSBucketName()

	if bucket == "" {
		Log.Warn(map[string]any{"bucket": cast.ToString(StorageToml.Get("cos.bucket"))}, "COS Bucket 未配置，腾讯云 COS 不可用")
		return
	}
	if secretId == "" || secretKey == "" {
		Log.Warn(map[string]any{"bucket": bucket}, "COS SecretId / SecretKey 未配置，腾讯云 COS 不可用")
	}

	// 桶名统一由 COSBucketName() 归一化（自动补 -appid），
	// 与 model.DomainTemp1 / DomainTemp2 的 {{cos}} 口径保持一致
	cosUrl, err := url.Parse(COSDomain())
	if err != nil {
		Log.Error(map[string]any{
			"error":     err.Error(),
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "COS URL 解析错误")
		return
	}

	this.Client = cos.NewClient(&cos.BaseURL{
		BucketURL: cosUrl,
	}, &http.Client{
		// 设置超时时间
		Timeout: 100 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretId,
			SecretKey: secretKey,
		},
	})
}

// Object - 获取 ObjectService
//
// 只返回对象服务本身：查询 / 创建存储桶是上传时才需要的动作，
// 删除、读取等操作不应顺带做 IsExist + CreateBucket（既慢，又可能在桶名写错时误建桶）。
func (this *COSStruct) Object() *cos.ObjectService {

	if this.Client == nil {
		return nil
	}

	return this.Client.Object
}

// ensureBucket 确保存储桶存在（仅上传时调用；默认公共读私有写）
func (this *COSStruct) ensureBucket() {

	if this.Client == nil {
		return
	}

	// 已经确认过就不再重复查询（首次上传才做一次 IsExist）
	this.bucketMu.Lock()
	ready := this.bucketReady
	this.bucketMu.Unlock()
	if ready {
		return
	}

	exist, err := this.Client.Bucket.IsExist(context.Background())
	if err != nil {
		// 密钥错误 / 桶名或地域不对都会走到这里：只告警、不建桶，避免掩盖配置问题
		Log.Error(map[string]any{
			"error":  err,
			"bucket": COSBucketName(),
			"region": COSRegion(),
		}, "COS Bucket 查询失败（请检查 SecretId / SecretKey / 桶名 / 地域）")
		return
	}
	if exist {
		this.markBucketReady()
		return
	}

	if _, err := this.Client.Bucket.Put(context.Background(), &cos.BucketPutOptions{
		XCosACL: "public-read",
	}); err != nil {
		Log.Error(map[string]any{
			"error":     err,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "COS Bucket 创建失败")
		return
	}

	Log.Info(map[string]any{"bucket": COSBucketName(), "region": COSRegion()}, "COS Bucket 已自动创建（公共读私有写）")
	this.markBucketReady()
}

// markBucketReady 标记存储桶已确认存在
func (this *COSStruct) markBucketReady() {
	this.bucketMu.Lock()
	this.bucketReady = true
	this.bucketMu.Unlock()
}

// cosUploadBody 归一化上传 body
//
// 返回可直接交给 SDK 的 reader、字节数（0 = 未知）与清理函数。
// COS SDK 只在能取到长度时（*os.File / *bytes.Reader / *bytes.Buffer / *strings.Reader）
// 使用带 Content-Length 的常规 PUT，其它包装类型（如附件上传用的 io.TeeReader）
// 会退化成 chunked 传输，部分网络 / CDN 环境会失败，因此这里统一中转到临时文件。
func cosUploadBody(reader io.Reader) (body io.Reader, size int64, cleanup func(), err error) {

	cleanup = func() {}

	switch item := reader.(type) {
	case *os.File:
		if info, statErr := item.Stat(); statErr == nil {
			return item, info.Size(), cleanup, nil
		}
	case *bytes.Reader:
		return item, item.Size(), cleanup, nil
	case *bytes.Buffer:
		return item, int64(item.Len()), cleanup, nil
	case *strings.Reader:
		return item, item.Size(), cleanup, nil
	}

	tmp, tmpErr := os.CreateTemp("", "inis-cos-upload-*")
	if tmpErr != nil {
		return nil, 0, cleanup, fmt.Errorf("创建临时文件失败：%v", tmpErr)
	}

	cleanup = func() {
		name := tmp.Name()
		tmp.Close()
		os.Remove(name)
	}

	size, err = io.Copy(tmp, reader)
	if err != nil {
		cleanup()
		return nil, 0, func() {}, fmt.Errorf("读取上传内容失败：%v", err)
	}

	if _, err = tmp.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, 0, func() {}, fmt.Errorf("重置临时文件指针失败：%v", err)
	}

	return tmp, size, cleanup, nil
}

// Upload - 上传文件
func (this *COSStruct) Upload(key string, reader io.Reader) (result *StorageResponse) {

	result = &StorageResponse{}

	object := this.Object()
	if object == nil {
		result.Error = errors.New("腾讯云 COS 未初始化，请检查 bucket / app_id / secret_id / secret_key / region")
		return
	}

	key = NormalizeStorageKey(key)
	if key == "" {
		result.Error = errors.New("对象键不能为空！")
		return
	}

	// 上传时才确认存储桶存在（删除等操作不建桶）
	this.ensureBucket()

	body, size, cleanup, err := cosUploadBody(reader)
	defer cleanup()
	if err != nil {
		result.Error = err
		return
	}

	// 单个对象设为公共读，配合桶的 public-read，避免私有桶导致图片 403
	options := &cos.ObjectPutOptions{
		ACLHeaderOptions: &cos.ACLHeaderOptions{XCosACL: "public-read"},
	}
	if size > 0 {
		options.ContentLength = size
	}

	if _, err := object.Put(context.Background(), key, body, options); err != nil {
		result.Error = err
		return
	}

	domain := cast.ToString(StorageToml.Get("cos.domain"))
	if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
		result.Domain = strings.TrimRight(domain, "/")
	} else {
		result.Domain = "{{cos}}"
	}

	result.Path = "/" + key

	return
}

// Path - COS存储位置 - 按命名规则生成对象键
//
// 目录与文件名都来自 config/storage.toml 的 [cos] 段（dir_rule / file_rule，
// 见 storage-rule.go），cos.path 是固定前缀；例如：
//
//	inis + {Y}-{m}/{d} + {timestamp}{str-random-10} + .jpg
//	→ inis/2026-09/26/1758888888123abc7def.jpg
func (this *COSStruct) Path(ctx StorageRuleContext) string {

	values := NewStorageRuleValues(ctx)

	dir := values.Apply(StorageRule("cos", StorageRuleDir))
	name := values.Apply(StorageRule("cos", StorageRuleFile))

	// cos.path 为空时不带前导斜杠（避免生成 "/2026-09/…" 这种键，拼 CDN 域名会出现 //）
	return StorageObjectKey(
		cast.ToString(StorageToml.Get("cos.path")),
		dir,
		StorageNameWithExt(name, ctx.Ext),
	)
}

// Delete - 删除单个对象
//
// DELETE Object 本身是幂等的：对象不存在时 COS 返回 204，不算失败。
func (this *COSStruct) Delete(key string) error {

	object := this.Object()
	if object == nil {
		return errors.New("腾讯云 COS 未初始化，请检查 bucket / app_id / secret_id / secret_key / region")
	}

	key = NormalizeStorageKey(key)
	if key == "" {
		return errors.New("对象键不能为空！")
	}

	response, err := object.Delete(context.Background(), key)
	// 对象不存在（404）视为删除成功，避免重复清理时报错
	if err != nil && response != nil && response.StatusCode == http.StatusNotFound {
		return nil
	}

	return err
}

// DeleteMulti - 批量删除对象（单请求最多 1000 个，这里按 500 分批）
//
// Quiet 模式只返回失败对象；若存在失败对象则返回错误（调用方据此保留对应记录），
// 避免「以为删掉了、其实还在」的静默失败。
func (this *COSStruct) DeleteMulti(keys []string) error {

	object := this.Object()
	if object == nil {
		return errors.New("腾讯云 COS 未初始化，请检查 bucket / app_id / secret_id / secret_key / region")
	}

	list := make([]string, 0, len(keys))
	for _, key := range keys {
		if key = NormalizeStorageKey(key); key != "" {
			list = append(list, key)
		}
	}
	if len(list) == 0 {
		return nil
	}

	var failed []string

	for start := 0; start < len(list); start += storageDeleteBatch {
		end := start + storageDeleteBatch
		if end > len(list) {
			end = len(list)
		}
		batch := list[start:end]

		objects := make([]cos.Object, 0, len(batch))
		for _, key := range batch {
			objects = append(objects, cos.Object{Key: key})
		}

		result, _, err := object.DeleteMulti(context.Background(), &cos.ObjectDeleteMultiOptions{
			Quiet:   true,
			Objects: objects,
		})

		if err != nil {
			failed = append(failed, batch...)
			Log.Error(map[string]any{"error": err.Error(), "count": len(batch), "first": batch[0]}, "COS 批量删除对象失败")
			continue
		}

		// Quiet 模式下 Errors 只包含失败的 Key
		if result != nil {
			for _, item := range result.Errors {
				failed = append(failed, item.Key)
				Log.Error(map[string]any{
					"key":     item.Key,
					"code":    item.Code,
					"message": item.Message,
				}, "COS 删除对象失败")
			}
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("COS 有 %d/%d 个对象删除失败（首个：%s）", len(failed), len(list), failed[0])
	}

	return nil
}
