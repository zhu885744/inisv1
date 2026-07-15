package facade

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/fsnotify/fsnotify"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/storage"
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
	// StorageModeOSS - OSS存储
	StorageModeOSS = "oss"
	// StorageModeCOS - COS存储
	StorageModeCOS = "cos"
	// StorageModeKODO - KODO存储
	StorageModeKODO = "kodo"
)

// NewStorage - 创建Storage实例
/**
 * @param mode 驱动模式
 * @return StorageInterface
 * @example：
 * 1. storage := facade.NewStorage("oss")
 * 2. storage := facade.NewStorage(facade.StorageModeOSS)
 */
func NewStorage(mode any) StorageInterface {
	switch strings.ToLower(cast.ToString(mode)) {
	case StorageModeLocal:
		Storage = LocalStorage
	case StorageModeOSS:
		Storage = OSS
	case StorageModeCOS:
		Storage = COS
	case StorageModeKODO:
		Storage = KODO
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
			"${default}":                     "local",
			"${local.domain}":                "storage",
			"${local.path}":                  "storage",
			"${oss.access_key_id}":           "",
			"${oss.access_key_secret}":       "",
			"${oss.endpoint}":                "",
			"${oss.bucket}":                  "inis-oss",
			"${oss.domain}":                  "",
			"${oss.path}":                    "inis",
			"${cos.app_id}":                  "",
			"${cos.secret_id}":               "",
			"${cos.secret_key}":              "",
			"${cos.bucket}":                  "inis-cos",
			"${cos.region}":                  "ap-guangzhou",
			"${cos.domain}":                  "",
			"${cos.path}":                    "inis",
			"${kodo.access_key}":             "",
			"${kodo.secret_key}":             "",
			"${kodo.bucket}":                 "inis-kodo",
			"${kodo.region}":                 "z2",
			"${kodo.domain}":                 "",
			"${attachment.allow_extensions}": "jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md",
			"${attachment.max_file_size}":    51200,
			"${attachment.concurrent_limit}": 5,
			"${attachment.limit_per_minute}": 60,
			"${attachment.limit_per_hour}":   500,
			"${attachment.limit_per_day}":    1000,
			"${attachment.limit_per_week}":   5000,
			"${attachment.limit_per_month}":  20000,
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

// 初始化缓存
func initStorage() {

	// OSS 对象存储
	OSS = &OSSStruct{}
	OSS.init()

	// COS 对象存储
	COS = &COSStruct{}
	COS.init()
	// 初始化COS Bucket
	COS.Object()

	// KODO 对象存储
	KODO = &KODOStruct{}
	KODO.init()

	// 本地存储
	LocalStorage = &LocalStorageStruct{}

	switch cast.ToString(StorageToml.Get("default")) {
	case "local":
		Storage = LocalStorage
	case "oss":
		Storage = OSS
	case "cos":
		Storage = COS
	case "kodo":
		Storage = KODO
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
 * storage := facade.Storage.Upload(facade.Storage.Path() + suffix, bytes)
 */
var Storage StorageInterface
var LocalStorage *LocalStorageStruct
var OSS *OSSStruct
var COS *COSStruct
var KODO *KODOStruct

// =================================== 附件配置 - 开始 ===================================

// AttachmentConfig - 附件配置
type AttachmentConfig struct {
	AllowExtensions []string // 允许的文件扩展名
	MaxFileSize     int64    // 单个文件最大大小（KB）
	ConcurrentLimit int      // 并发上传限制
	LimitPerMinute  int      // 每分钟上传限制（0为不限制）
	LimitPerHour    int      // 每小时上传限制（0为不限制）
	LimitPerDay     int      // 每天上传限制（0为不限制）
	LimitPerWeek    int      // 每周上传限制（0为不限制）
	LimitPerMonth   int      // 每月上传限制（0为不限制）
}

// AttachmentConfigInstance - 附件配置实例
var AttachmentConfigInstance *AttachmentConfig

// InitAttachmentConfig - 初始化附件配置
func InitAttachmentConfig() {
	AttachmentConfigInstance = &AttachmentConfig{
		AllowExtensions: parseExtensions(cast.ToString(StorageToml.Get("attachment.allow_extensions"))),
		MaxFileSize:     cast.ToInt64(StorageToml.Get("attachment.max_file_size")),
		ConcurrentLimit: cast.ToInt(StorageToml.Get("attachment.concurrent_limit")),
		LimitPerMinute:  cast.ToInt(StorageToml.Get("attachment.limit_per_minute")),
		LimitPerHour:    cast.ToInt(StorageToml.Get("attachment.limit_per_hour")),
		LimitPerDay:     cast.ToInt(StorageToml.Get("attachment.limit_per_day")),
		LimitPerWeek:    cast.ToInt(StorageToml.Get("attachment.limit_per_week")),
		LimitPerMonth:   cast.ToInt(StorageToml.Get("attachment.limit_per_month")),
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
	Path() string
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

	// 去除前面的 public
	result.Path = strings.Replace(path, "public", "", 1)
	result.Domain = cast.ToString(StorageToml.Get("local.domain"))

	return
}

// Path - 本地存储位置 - 生成文件路径
func (this *LocalStorageStruct) Path() string {
	// 生成年月日目录 - 如：2006-01/02/
	dir := time.Now().Format("2006-01/02/")
	// 生成文件名 - 年月日+毫秒时间戳
	name := cast.ToString(time.Now().UnixNano() / 1e6)
	path := cast.ToString(StorageToml.Get("local.path"))
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return "public/" + path + dir + name
}

// Delete - 删除文件
func (this *LocalStorageStruct) Delete(key string) error {
	path := strings.TrimPrefix(key, "/")
	if !strings.HasPrefix(path, "public/") {
		path = "public/" + path
	}
	return os.Remove(path)
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

// ================================== 阿里云对象存储 - 开始 ==================================

// OSSStruct 阿里云对象存储
type OSSStruct struct {
	Client *oss.Client
}

// init 初始化 阿里云对象存储
func (this *OSSStruct) init() {

	accessKeyId := cast.ToString(StorageToml.Get("oss.access_key_id"))
	accessKeySecret := cast.ToString(StorageToml.Get("oss.access_key_secret"))
	endpoint := cast.ToString(StorageToml.Get("oss.endpoint"))

	client, err := oss.New(endpoint, accessKeyId, accessKeySecret)

	if err != nil {
		Log.Error(map[string]any{
			"error":     err.Error(),
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "OSS 初始化错误")
		return
	}

	this.Client = client
}

// Bucket - 获取Bucket（存储桶）
func (this *OSSStruct) Bucket() *oss.Bucket {

	exist, err := this.Client.IsBucketExist(cast.ToString(StorageToml.Get("oss.bucket")))

	if err != nil {
		Log.Error(map[string]any{
			"error":     err,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "OSS Bucket 初始化错误")
	}

	wg := sync.WaitGroup{}

	if !exist {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			// 创建存储空间。
			err = this.Client.CreateBucket(cast.ToString(StorageToml.Get("oss.bucket")))
			if err != nil {
				Log.Error(map[string]any{
					"error":     err,
					"func_name": utils.Caller().FuncName,
					"file_name": utils.Caller().FileName,
					"file_line": utils.Caller().Line,
				}, "OSS Bucket 创建错误")
			}
		}(&wg)
	}

	wg.Wait()

	bucket, err := this.Client.Bucket(cast.ToString(StorageToml.Get("oss.bucket")))
	if err != nil {
		Log.Error(map[string]any{
			"error":     err,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "OSS Bucket 获取错误")
		return nil
	}

	return bucket
}

// Upload - 上传文件
func (this *OSSStruct) Upload(key string, reader io.Reader) (result *StorageResponse) {

	result = &StorageResponse{}

	err := OSS.Bucket().PutObject(key, reader)
	if err != nil {
		result.Error = err
		return
	}

	domain := cast.ToString(StorageToml.Get("oss.domain"))
	if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
		result.Domain = domain
	} else {
		result.Domain = "{{oss}}"
	}

	result.Path = "/" + key

	return
}

// Path - OSS存储位置 - 生成文件路径
func (this *OSSStruct) Path() string {
	// 生成年月日目录 - 如：2023-04/10
	dir := time.Now().Format("2006-01/02/")
	// 生成文件名 - 年月日+毫秒时间戳
	name := cast.ToString(time.Now().UnixNano() / 1e6)
	path := cast.ToString(StorageToml.Get("oss.path"))
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path + dir + name
}

// Delete - 删除文件
func (this *OSSStruct) Delete(key string) error {
	key = strings.TrimPrefix(key, "/")
	return this.Bucket().DeleteObject(key)
}

// DeleteMulti - 批量删除文件
func (this *OSSStruct) DeleteMulti(keys []string) error {
	for i := range keys {
		keys[i] = strings.TrimPrefix(keys[i], "/")
	}
	_, err := this.Bucket().DeleteObjects(keys)
	return err
}

// ================================== 腾讯云对象存储 - 开始 ==================================

// COSStruct 腾讯云对象存储
type COSStruct struct {
	Client *cos.Client
}

// init 初始化 腾讯云对象存储
func (this *COSStruct) init() {

	appId := cast.ToString(StorageToml.Get("cos.app_id"))
	secretId := cast.ToString(StorageToml.Get("cos.secret_id"))
	secretKey := cast.ToString(StorageToml.Get("cos.secret_key"))
	bucket := cast.ToString(StorageToml.Get("cos.bucket"))
	region := cast.ToString(StorageToml.Get("cos.region"))

	cosUrl, err := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", bucket, appId, region))
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

// Object - 获取Object（对象存储）
func (this *COSStruct) Object() *cos.ObjectService {

	// 查询存储桶
	exist, err := this.Client.Bucket.IsExist(context.Background())

	if err != nil {
		Log.Error(map[string]any{
			"error":     err,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "COS Bucket 查询失败")
	}

	wg := sync.WaitGroup{}

	if !exist {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			defer wg.Done()
			// 创建存储桶 - 默认公共读私有写
			_, err = this.Client.Bucket.Put(context.Background(), &cos.BucketPutOptions{
				XCosACL: "public-read",
			})
			if err != nil {
				Log.Error(map[string]any{
					"error":     err,
					"func_name": utils.Caller().FuncName,
					"file_name": utils.Caller().FileName,
					"file_line": utils.Caller().Line,
				}, "COS Bucket 创建失败")
			}
		}(&wg)
	}

	wg.Wait()

	return this.Client.Object
}

// Upload - 上传文件
func (this *COSStruct) Upload(key string, reader io.Reader) (result *StorageResponse) {

	result = &StorageResponse{}

	_, err := this.Object().Put(context.Background(), key, reader, nil)
	if err != nil {
		result.Error = err
		return
	}

	domain := cast.ToString(StorageToml.Get("cos.domain"))
	if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
		result.Domain = domain
	} else {
		result.Domain = "{{cos}}"
	}

	result.Path = "/" + key

	return
}

// Path - COS存储位置 - 生成文件路径
func (this *COSStruct) Path() string {
	// 生成年月日目录 - 如：2023-04/10
	dir := time.Now().Format("2006-01/02/")
	// 生成文件名 - 年月日+毫秒时间戳
	name := cast.ToString(time.Now().UnixNano() / 1e6)
	path := cast.ToString(StorageToml.Get("cos.path"))
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path + dir + name
}

// Delete - 删除文件
func (this *COSStruct) Delete(key string) error {
	key = strings.TrimPrefix(key, "/")
	_, err := this.Object().Delete(context.Background(), key)
	return err
}

// DeleteMulti - 批量删除文件
func (this *COSStruct) DeleteMulti(keys []string) error {
	objects := make([]cos.Object, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimPrefix(key, "/")
		objects = append(objects, cos.Object{Key: key})
	}
	_, _, err := this.Object().DeleteMulti(context.Background(), &cos.ObjectDeleteMultiOptions{
		Quiet:   false,
		Objects: objects,
	})
	return err
}

// ================================== 七牛云对象存储 - 开始 ==================================

// KODOStruct 七牛云对象存储
type KODOStruct struct {
	Client *qbox.Mac
}

// init 初始化 七牛云对象存储
func (this *KODOStruct) init() {

	this.Client = qbox.NewMac(
		cast.ToString(StorageToml.Get("kodo.access_key")),
		cast.ToString(StorageToml.Get("kodo.secret_key")),
	)
}

// IsExist - 存储空间是否存在
func (this *KODOStruct) IsExist() bool {

	bucket := storage.NewBucketManager(this.Client, nil)
	_, err := bucket.GetBucketInfo(cast.ToString(StorageToml.Get("kodo.bucket")))

	if err != nil {
		// 不存在则创建
		if strings.Contains(err.Error(), "no such entry") {
			return false
		}
	}

	return true
}

func (this *KODOStruct) CreateBucket() error {

	bucketName := cast.ToString(StorageToml.Get("kodo.bucket"))
	regionName := cast.ToString(StorageToml.Get("kodo.region"))

	// 创建存储空间
	config := storage.Config{
		// 空间对应的机房
		Zone: &storage.ZoneHuanan,
		// 是否使用https域名
		UseHTTPS: true,
		// 上传是否使用CDN上传加速
		UseCdnDomains: false,
	}

	// 创建存储空间
	bucket := storage.NewBucketManager(this.Client, &config)
	if region, ok := storage.GetRegionByID(storage.RegionID(regionName)); ok {

		config.Region = &region
		err := bucket.CreateBucket(bucketName, storage.RegionID(regionName))

		return utils.Ternary(err == nil, nil, err)
	}

	return errors.New("存储空间创建失败")
}

func (this *KODOStruct) Bucket() *qbox.Mac {

	// 如果存储空间不存在 - 则创建
	if !this.IsExist() {
		err := this.CreateBucket()
		if err != nil {
			Log.Error(map[string]any{
				"error":     err,
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line,
			}, "KODO 存储空间创建失败")
			return nil
		}
	}

	return this.Client
}

func (this *KODOStruct) Upload(key string, reader io.Reader) (result *StorageResponse) {

	result = &StorageResponse{}

	bucketName := cast.ToString(StorageToml.Get("kodo.bucket"))
	regionName := cast.ToString(StorageToml.Get("kodo.region"))

	policy := storage.PutPolicy{
		Scope: bucketName,
	}
	token := policy.UploadToken(this.Bucket())

	config := storage.Config{
		// 空间对应的机房
		Region: &storage.ZoneHuanan,
		// 上传是否使用CDN上传加速
		UseCdnDomains: false,
		// 是否使用https域名
		UseHTTPS: true,
	}

	// 构建表单上传的对象
	bucket := storage.NewFormUploader(&config)

	if region, ok := storage.GetRegionByID(storage.RegionID(regionName)); ok {
		config.Region = &region
	}

	body := storage.PutRet{}
	err := bucket.Put(context.Background(), &body, token, key, reader, -1, &storage.PutExtra{})
	if err != nil {
		result.Error = err
		return
	}

	result.Path = "/" + key

	domain := cast.ToString(StorageToml.Get("kodo.domain"))
	if !utils.Is.Empty(domain) && !strings.Contains(domain, "{{") {
		result.Domain = domain
	} else {
		result.Domain = "{{kodo}}"
	}

	return
}

// Path - OSS存储位置 - 生成文件路径
func (this *KODOStruct) Path() string {
	// 生成年月日目录 - 如：2023-04/10
	dir := time.Now().Format("2006-01/02/")
	// 生成文件名 - 年月日+毫秒时间戳
	name := cast.ToString(time.Now().UnixNano() / 1e6)
	return "storage/" + dir + name
}

// Delete - 删除文件
func (this *KODOStruct) Delete(key string) error {
	key = strings.TrimPrefix(key, "/")
	bucketName := cast.ToString(StorageToml.Get("kodo.bucket"))
	regionName := cast.ToString(StorageToml.Get("kodo.region"))

	config := storage.Config{
		UseHTTPS: true,
	}
	if region, ok := storage.GetRegionByID(storage.RegionID(regionName)); ok {
		config.Region = &region
	}

	bucket := storage.NewBucketManager(this.Client, &config)
	return bucket.Delete(bucketName, key)
}

// DeleteMulti - 批量删除文件
func (this *KODOStruct) DeleteMulti(keys []string) error {
	bucketName := cast.ToString(StorageToml.Get("kodo.bucket"))
	regionName := cast.ToString(StorageToml.Get("kodo.region"))

	config := storage.Config{
		UseHTTPS: true,
	}
	if region, ok := storage.GetRegionByID(storage.RegionID(regionName)); ok {
		config.Region = &region
	}

	bucket := storage.NewBucketManager(this.Client, &config)
	for _, key := range keys {
		key = strings.TrimPrefix(key, "/")
		if err := bucket.Delete(bucketName, key); err != nil {
			Log.Error(map[string]any{
				"error": err,
				"key":   key,
			}, "KODO批量删除文件失败")
		}
	}
	return nil
}
