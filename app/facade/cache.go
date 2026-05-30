package facade

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

const (
	ConfigNameCache    = "cache"
	DefaultCacheDriver = CacheModeFile
)

// 缓存模式常量
const (
	// CacheModeRedis - Redis缓存
	CacheModeRedis = "redis"
	// CacheModeFile  - 文件缓存
	CacheModeFile = "file"
	// CacheModeRAM   - 内存缓存
	CacheModeRAM = "ram"
)

// CacheToml - 缓存配置文件
var CacheToml *utils.ViperResponse

// NewCache - 创建Cache实例
func NewCache(mode any) CacheInterface {
	switch strings.ToLower(cast.ToString(mode)) {
	case CacheModeRedis:
		Cache = Redis
	case CacheModeFile:
		Cache = FileCache
	case CacheModeRAM:
		Cache = BigCache
	default:
		Cache = FileCache
	}
	return Cache
}

// Cache - Cache实例
var Cache CacheInterface
var Redis *RedisCacheStruct
var FileCache *FileCacheStruct
var BigCache *BigCacheStruct

type CacheInterface interface {
	Has(key any) bool
	Get(key any) any
	Set(key any, value any, expire ...any) bool
	Del(key any) bool
	DelPrefix(prefix ...any) bool
	DelTags(tag ...any) bool
	Clear() bool
}

// init - 初始化
func init() {
	initCacheToml()
	initCache()

	WatchConfigChange(CacheToml, initCache)
}

// initCacheToml - 初始化缓存配置文件
func initCacheToml() {
	item := utils.Viper(utils.ViperModel{
		Path: ConfigPath,
		Mode: ModeToml,
		Name: ConfigNameCache,
		Content: utils.Replace(TempCache, map[string]any{
			"${open}":           "false",
			"${default}":        DefaultCacheDriver,
			"${local.expire}":   300,
			"${redis.host}":     "localhost",
			"${redis.port}":     "6379",
			"${redis.password}": "",
			"${redis.expire}":   "2 * 60 * 60",
			"${redis.prefix}":   "inis:",
			"${redis.database}": 0,
			"${file.expire}":    "2 * 60 * 60",
			"${file.path}":      "runtime/cache",
			"${file.prefix}":    "inis_",
			"${ram.expire}":     "2 * 60 * 60",
		}),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line}, "Cache配置初始化错误")
	}

	CacheToml = &item
}

// initCache - 初始化缓存
func initCache() {
	// Redis 缓存
	Redis = &RedisCacheStruct{}
	Redis.init()

	// File 缓存
	FileCache = &FileCacheStruct{}
	FileCache.init()

	// BigCache 缓存
	BigCache = &BigCacheStruct{}
	BigCache.init()

	switch cast.ToString(CacheToml.Get("default")) {
	case CacheModeRedis:
		Cache = Redis
	case CacheModeFile:
		Cache = FileCache
	case CacheModeRAM:
		Cache = BigCache
	default:
		Cache = FileCache
	}
}

// ==================== Redis 缓存 ====================

type RedisCacheStruct struct {
	Client *redis.Client
	Prefix string
	Expire time.Duration
}

func (this *RedisCacheStruct) init() {
	host := cast.ToString(CacheToml.Get("redis.host"))
	port := cast.ToString(CacheToml.Get("redis.port"))

	this.Prefix = cast.ToString(CacheToml.Get("redis.prefix"))
	this.Client = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		DB:       cast.ToInt(CacheToml.Get("redis.database")),
		Password: cast.ToString(CacheToml.Get("redis.password")),
	})
	this.Expire = time.Duration(cast.ToInt(utils.Calc(CacheToml.Get("redis.expire", 7200)))) * time.Second
}

func (this *RedisCacheStruct) Has(key any) bool {
	ctx := context.Background()
	result, err := this.Client.Exists(ctx, this.Prefix+cast.ToString(key)).Result()
	return utils.Ternary[bool](err != nil, false, result == 1)
}

func (this *RedisCacheStruct) Get(key any) any {
	ctx := context.Background()
	result, err := this.Client.Get(ctx, this.Prefix+cast.ToString(key)).Result()
	return utils.Ternary[any](err != nil, nil, utils.Json.Decode(result))
}

func (this *RedisCacheStruct) Set(key any, value any, expire ...any) bool {
	ctx := context.Background()
	expiration := this.Expire

	if len(expire) > 0 {
		if !utils.Is.Empty(expire[0]) {
			if reflect.ValueOf(expire[0]).Kind() == reflect.Int64 && expire[0] != 0 {
				expiration = time.Duration(cast.ToInt(expire[0])) * time.Second
			} else if reflect.TypeOf(expire[0]).String() == "time.Duration" {
				expiration = expire[0].(time.Duration)
			}
		}
	}

	err := this.Client.Set(ctx, this.Prefix+cast.ToString(key), utils.Json.Encode(value), expiration).Err()
	return utils.Ternary[bool](err != nil, false, true)
}

func (this *RedisCacheStruct) Del(key any) bool {
	ctx := context.Background()
	err := this.Client.Del(ctx, this.Prefix+cast.ToString(key)).Err()
	return utils.Ternary[bool](err != nil, false, true)
}

func (this *RedisCacheStruct) DelPrefix(prefix ...any) bool {
	ctx := context.Background()
	var keys []string
	var prefixes []string

	if len(prefix) == 0 {
		return false
	}

	for _, value := range prefix {
		if reflect.ValueOf(value).Kind() == reflect.Slice {
			for _, val := range cast.ToSlice(value) {
				prefixes = append(prefixes, this.Prefix+cast.ToString(val)+"*")
			}
		} else {
			prefixes = append(prefixes, this.Prefix+cast.ToString(value)+"*")
		}
	}

	for _, val := range prefixes {
		item, err := this.Client.Keys(ctx, val).Result()
		if err != nil {
			return false
		}
		keys = append(keys, item...)
	}

	keys = cast.ToStringSlice(utils.ArrayEmpty(utils.ArrayUnique(keys)))
	if len(keys) > 0 {
		err := this.Client.Del(ctx, keys...).Err()
		if err != nil {
			return false
		}
	}
	return true
}

func (this *RedisCacheStruct) DelTags(tag ...any) bool {
	ctx := context.Background()
	var keys []string
	var tags []string

	if len(tag) == 0 {
		return false
	}

	for _, value := range tag {
		var item string
		if reflect.ValueOf(value).Kind() == reflect.Slice {
			var tmp []string
			for _, val := range cast.ToSlice(value) {
				tmp = append(tmp, cast.ToString(val))
			}
			item = strings.Join(tmp, "*")
		} else {
			item = cast.ToString(value)
		}
		tags = append(tags, fmt.Sprintf("%s*%s*", this.Prefix, item))
	}

	for _, val := range tags {
		item, err := this.Client.Keys(ctx, val).Result()
		if err != nil {
			return false
		}
		keys = append(keys, item...)
	}

	keys = cast.ToStringSlice(utils.ArrayEmpty(utils.ArrayUnique(keys)))
	if len(keys) > 0 {
		err := this.Client.Del(ctx, keys...).Err()
		if err != nil {
			return false
		}
	}
	return true
}

func (this *RedisCacheStruct) Clear() bool {
	ctx := context.Background()
	err := this.Client.FlushDB(ctx).Err()
	return utils.Ternary[bool](err != nil, false, true)
}

// ==================== 文件缓存 ====================

type FileCacheStruct struct {
	Client *utils.FileCacheClient
}

func (this *FileCacheStruct) init() {
	var err error
	this.Client, err = utils.NewFileCache(
		CacheToml.Get("file.path"),
		utils.Calc(CacheToml.Get("file.expire", 7200)),
		CacheToml.Get("file.prefix"),
	)
	if err != nil {
		fmt.Println("文件缓存初始化失败: " + err.Error())
	}
}

func (this *FileCacheStruct) Has(key any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Has(key)
}

func (this *FileCacheStruct) Get(key any) any {
	if this.Client == nil {
		return nil
	}
	return utils.Json.Decode(this.Client.Get(key))
}

func (this *FileCacheStruct) Set(key any, value any, expire ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Set(key, []byte(utils.Json.Encode(value)), expire...)
}

func (this *FileCacheStruct) Del(key any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Del(key)
}

func (this *FileCacheStruct) DelPrefix(prefix ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.DelPrefix(prefix...)
}

func (this *FileCacheStruct) DelTags(tag ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.DelTags(tag...)
}

func (this *FileCacheStruct) Clear() bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Clear()
}

// ==================== 内存缓存 ====================

type BigCacheStruct struct {
	Client *BigCacheClient
}

func (this *BigCacheStruct) init() {
	this.Client = NewBigCache(utils.Calc(CacheToml.Get("file.expire", 7200)))
}

func (this *BigCacheStruct) Has(key any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Has(key)
}

func (this *BigCacheStruct) Get(key any) any {
	if this.Client == nil {
		return nil
	}
	return utils.Json.Decode(this.Client.Get(key))
}

func (this *BigCacheStruct) Set(key any, value any, expire ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Set(key, []byte(utils.Json.Encode(value)), expire...)
}

func (this *BigCacheStruct) Del(key any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Del(key)
}

func (this *BigCacheStruct) DelPrefix(prefix ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.DelPrefix(prefix...)
}

func (this *BigCacheStruct) DelTags(tag ...any) bool {
	if this.Client == nil {
		return false
	}
	return this.Client.DelTags(tag...)
}

func (this *BigCacheStruct) Clear() bool {
	if this.Client == nil {
		return false
	}
	return this.Client.Clear()
}

// BigCacheClient 缓存
type BigCacheClient struct {
	mutex  sync.Mutex
	prefix string
	expire int64
	items  map[string]*bigcache.BigCache
}

func NewBigCache(expire any, prefix ...string) *BigCacheClient {
	var cache BigCacheClient
	cache.expire = cast.ToInt64(expire)
	cache.items = make(map[string]*bigcache.BigCache)
	cache.prefix = "cache_"
	if len(prefix) > 0 {
		cache.prefix = prefix[0]
	}
	return &cache
}

func (this *BigCacheClient) Get(key any) []byte {
	res, err := this.GetE(key)
	return utils.Ternary(err != nil, nil, res)
}

func (this *BigCacheClient) Has(key any) bool {
	_, ok := this.items[this.name(key)]
	return ok
}

func (this *BigCacheClient) Set(key any, value []byte, expire ...any) bool {
	expiration := this.expire
	if len(expire) > 0 && !utils.Is.Empty(expire[0]) {
		if reflect.TypeOf(expire[0]).String() == "time.Duration" {
			expiration = cast.ToInt64(cast.ToDuration(expire[0]).Seconds())
		} else {
			expiration = cast.ToInt64(expire[0])
		}
	}

	err := this.SetE(key, value, expiration)
	return utils.Ternary(err != nil, false, true)
}

func (this *BigCacheClient) Del(key any) bool {
	err := this.DelE(key)
	return utils.Ternary(err != nil, false, true)
}

func (this *BigCacheClient) Clear() bool {
	err := this.ClearE()
	return utils.Ternary(err != nil, false, true)
}

func (this *BigCacheClient) DelPrefix(prefix ...any) bool {
	err := this.DelPrefixE(prefix...)
	return utils.Ternary(err != nil, false, true)
}

func (this *BigCacheClient) DelTags(tags ...any) bool {
	err := this.DelTagsE(tags...)
	return utils.Ternary(err != nil, false, true)
}

func (this *BigCacheClient) GetE(key any) ([]byte, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	item, ok := this.items[this.name(key)]
	if !ok {
		delete(this.items, this.name(key))
		return nil, fmt.Errorf("cache %s not exists", this.name(key))
	}

	value, err := item.Get(this.name(key))
	if err != nil {
		return nil, err
	}

	return value, nil
}

func (this *BigCacheClient) SetE(key any, value []byte, expire int64) error {
	duration := time.Duration(100 * 365 * 24 * 60 * 60 * 1e9) // 100 years
	if expire != 0 {
		duration = time.Duration(expire) * time.Second
	}

	item, _ := bigcache.New(context.Background(), bigcache.DefaultConfig(duration))
	err := item.Set(this.name(key), value)
	if err != nil {
		return err
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.items[this.name(key)] = item
	return nil
}

func (this *BigCacheClient) DelE(key any) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	item, ok := this.items[this.name(key)]
	if !ok {
		return fmt.Errorf("cache %s not exists", this.name(key))
	}

	err := item.Delete(this.name(key))
	if err != nil {
		return err
	}

	delete(this.items, this.name(key))
	return nil
}

func (this *BigCacheClient) ClearE() error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, item := range this.items {
		err := item.Reset()
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *BigCacheClient) DelPrefixE(prefix ...any) error {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for key, item := range this.items {
		if strings.HasPrefix(key, cast.ToString(prefix)) {
			err := item.Reset()
			if err != nil {
				return err
			}
			delete(this.items, key)
		}
	}
	return nil
}

func (this *BigCacheClient) DelTagsE(tag ...any) error {
	var keys []string
	var tags []string

	if len(tag) == 0 {
		return nil
	}

	for _, value := range tag {
		var item string
		if reflect.ValueOf(value).Kind() == reflect.Slice {
			var tmp []string
			for _, val := range cast.ToSlice(value) {
				tmp = append(tmp, cast.ToString(val))
			}
			item = strings.Join(tmp, "*")
		} else {
			item = cast.ToString(value)
		}
		tags = append(tags, fmt.Sprintf("*%s*", item))
	}

	for key := range this.items {
		keys = append(keys, key)
	}

	keys = this.fuzzyMatch(keys, tags)
	for _, key := range keys {
		item, ok := this.items[key]
		if !ok {
			continue
		}
		err := item.Reset()
		if err != nil {
			return err
		}
		delete(this.items, key)
	}
	return nil
}

func (this *BigCacheClient) name(key any) string {
	return fmt.Sprintf("%s%s", this.prefix, cast.ToString(key))
}

func (this *BigCacheClient) fuzzyMatch(keys []string, tags []string) []string {
	var result []string
	for _, item := range keys {
		for _, tag := range tags {
			if matched, _ := filepath.Match(tag, item); matched {
				result = append(result, item)
				break
			}
		}
	}
	return result
}
