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
	Stats() CacheStats
	IncrementHits()
	IncrementMisses()

	// ---- 原子操作 ----
	//
	// 这三个方法是为「限流 / 分布式锁 / 跨实例计数」准备的：
	// 之前的 Get -> 计算 -> Set 写法在 Redis 下是两次网络往返，多实例并发必然算错。
	//
	// Redis 驱动走原生命令（INCRBY / SET NX / EXPIRE），真正的跨实例原子；
	// file / ram 驱动没有跨进程能力，只能用进程内锁模拟，**仅单实例部署下语义正确**。

	// Incr 自增（delta 省略时为 1），返回自增后的值；失败时 ok=false
	Incr(key any, delta ...int64) (value int64, ok bool)
	// SetNX 仅当 key 不存在时写入（用于抢占锁）；expire 语义同 Set
	SetNX(key any, value any, expire ...any) bool
	// Expire 给已存在的 key 重新设置过期时间（key 不存在返回 false）
	Expire(key any, expire any) bool
}

// CacheStats - 缓存统计信息
type CacheStats struct {
	Hits      int64
	Misses    int64
	HitRate   float64
	TotalGets int64
}

// init - 初始化
func init() {
	// 同 app.go：本文件初始化早于 log.go，先在配置异常路径前确保日志就绪
	ensureLogReady()

	initCacheToml()
	initCache()

	WatchConfigChange(CacheToml, initCache)
}

// initCacheToml - 初始化缓存配置文件
func initCacheToml() {
	item := utils.Viper(utils.ViperModel{
		Path:    ConfigPath,
		Mode:    ModeToml,
		Name:    ConfigNameCache,
		Content: cacheTomlContent(),
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
	// 重新读取配置文件，确保使用最新配置
	if CacheToml != nil && CacheToml.Viper != nil {
		err := CacheToml.Viper.ReadInConfig()
		if err == nil {
			CacheToml.Result = CacheToml.Viper.AllSettings()
		}
	}

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
	// DefaultExpire 默认过期时间（cache.toml 的 redis.expire）
	// 不能叫 Expire：与 RedisCacheStruct.Expire(key, expire) 方法同名会编译失败
	DefaultExpire time.Duration
	hits          int64
	misses        int64
	mutex         sync.Mutex
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
	this.DefaultExpire = time.Duration(cast.ToInt(utils.Calc(CacheToml.Get("redis.expire", 7200)))) * time.Second
}

func (this *RedisCacheStruct) Has(key any) bool {
	ctx := context.Background()
	result, err := this.Client.Exists(ctx, this.Prefix+cast.ToString(key)).Result()
	return utils.Ternary[bool](err != nil, false, result == 1)
}

func (this *RedisCacheStruct) Get(key any) any {
	ctx := context.Background()
	result, err := this.Client.Get(ctx, this.Prefix+cast.ToString(key)).Result()
	if err != nil {
		this.IncrementMisses()
		return nil
	}
	this.IncrementHits()
	return utils.Json.Decode(result)
}

func (this *RedisCacheStruct) Stats() CacheStats {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	total := this.hits + this.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(this.hits) / float64(total) * 100
	}
	return CacheStats{
		Hits:      this.hits,
		Misses:    this.misses,
		HitRate:   hitRate,
		TotalGets: total,
	}
}

func (this *RedisCacheStruct) IncrementHits() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.hits++
}

func (this *RedisCacheStruct) IncrementMisses() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.misses++
}

// expireOf 把调用方传入的 expire 参数归一化成 time.Duration
//
// 兼容三类写法：
//   - time.Duration（推荐）：直接使用；
//   - 数字（int / int64 / float64 …）：按「秒」处理；
//   - 字符串："7200" 或 "2 * 60 * 60" 这类表达式（与 cache.toml 里的写法一致）。
//
// 未传 / 传 0 / 解析失败 → 回落到驱动默认过期时间（cache.toml 的 redis.expire）。
// 注意：这与「0 表示永不过期」的直觉不同；要长期缓存请传足够大的时长。
func (this *RedisCacheStruct) expireOf(expire ...any) time.Duration {
	if len(expire) == 0 || utils.Is.Empty(expire[0]) {
		return this.DefaultExpire
	}

	if value, ok := expire[0].(time.Duration); ok {
		if value <= 0 {
			return this.DefaultExpire
		}
		return value
	}

	if seconds := cast.ToInt64(utils.Calc(expire[0])); seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	return this.DefaultExpire
}

func (this *RedisCacheStruct) Set(key any, value any, expire ...any) bool {
	ctx := context.Background()
	err := this.Client.Set(ctx, this.Prefix+cast.ToString(key), utils.Json.Encode(value), this.expireOf(expire...)).Err()
	return utils.Ternary[bool](err != nil, false, true)
}

// Incr 自增（Redis INCRBY，跨实例原子）
func (this *RedisCacheStruct) Incr(key any, delta ...int64) (int64, bool) {
	if this.Client == nil {
		return 0, false
	}

	step := int64(1)
	if len(delta) > 0 {
		step = delta[0]
	}

	result, err := this.Client.IncrBy(context.Background(), this.Prefix+cast.ToString(key), step).Result()
	if err != nil {
		return 0, false
	}

	return result, true
}

// SetNX 仅当 key 不存在时写入（Redis SET NX，分布式锁的基础）
func (this *RedisCacheStruct) SetNX(key any, value any, expire ...any) bool {
	if this.Client == nil {
		return false
	}

	ok, err := this.Client.SetNX(context.Background(), this.Prefix+cast.ToString(key), utils.Json.Encode(value), this.expireOf(expire...)).Result()
	return utils.Ternary[bool](err != nil, false, ok)
}

// Expire 给已存在的 key 重新设置过期时间
func (this *RedisCacheStruct) Expire(key any, expire any) bool {
	if this.Client == nil {
		return false
	}

	expiration := this.expireOf(expire)
	if expiration <= 0 {
		return false
	}

	ok, err := this.Client.Expire(context.Background(), this.Prefix+cast.ToString(key), expiration).Result()
	return utils.Ternary[bool](err != nil, false, ok)
}

func (this *RedisCacheStruct) Del(key any) bool {
	ctx := context.Background()
	err := this.Client.Del(ctx, this.Prefix+cast.ToString(key)).Err()
	return utils.Ternary[bool](err != nil, false, true)
}

// deleteByPattern 按模式批量删除（SCAN + UNLINK）
//
// 不用 KEYS：KEYS 是 O(N) 且阻塞 Redis 单线程，key 多的时候一次「后台保存配置」
// 就能把整站所有请求卡住；SCAN 是游标式遍历，不会长时间占住服务端。
// 也不用 DEL：UNLINK 在后台线程回收内存，批量删除大 key 时不阻塞。
func (this *RedisCacheStruct) deleteByPattern(patterns []string) bool {
	ctx := context.Background()

	for _, pattern := range patterns {
		iter := this.Client.Scan(ctx, 0, pattern, 200).Iterator()
		batch := make([]string, 0, 200)

		for iter.Next(ctx) {
			batch = append(batch, iter.Val())
			if len(batch) >= 200 {
				this.unlink(ctx, batch)
				batch = batch[:0]
			}
		}

		if err := iter.Err(); err != nil {
			return false
		}

		this.unlink(ctx, batch)
	}

	return true
}

// unlink 批量异步删除（低版本 Redis 不支持 UNLINK，回退到 DEL）
func (this *RedisCacheStruct) unlink(ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	if err := this.Client.Unlink(ctx, keys...).Err(); err != nil {
		_ = this.Client.Del(ctx, keys...).Err()
	}
}

func (this *RedisCacheStruct) DelPrefix(prefix ...any) bool {
	if len(prefix) == 0 {
		return false
	}

	var patterns []string

	for _, value := range prefix {
		if reflect.ValueOf(value).Kind() == reflect.Slice {
			for _, val := range cast.ToSlice(value) {
				patterns = append(patterns, this.Prefix+cast.ToString(val)+"*")
			}
		} else {
			patterns = append(patterns, this.Prefix+cast.ToString(value)+"*")
		}
	}

	return this.deleteByPattern(patterns)
}

func (this *RedisCacheStruct) DelTags(tag ...any) bool {
	if len(tag) == 0 {
		return false
	}

	var tags []string

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

	return this.deleteByPattern(tags)
}

// Clear 清空本应用的缓存
//
// 原来直接 FlushDB：会把**同一个 Redis DB 里所有数据**清掉（别的应用、甚至本应用
// 其它命名空间的数据都会一起没），cluster 模式下 FLUSHDB 还不可用。
// 现在改为只删除带 prefix 的键；没配 prefix 时拒绝执行，避免误伤。
func (this *RedisCacheStruct) Clear() bool {
	if this.Prefix == "" {
		Log.Error(map[string]any{
			"cache_driver": CacheModeRedis,
		}, "拒绝清空缓存：未配置 redis.prefix，按前缀删除会误删整个 DB，请在 cache.toml 里配置前缀")
		return false
	}

	return this.deleteByPattern([]string{this.Prefix + "*"})
}

// ==================== 文件缓存 ====================

type FileCacheStruct struct {
	Client *utils.FileCacheClient
	hits   int64
	misses int64
	mutex  sync.Mutex
	// atomicMu 串行化 Incr / SetNX / Expire 的「读改写」，
	// 与 stats 的 mutex 分开，避免嵌套调用 Get 时自锁
	atomicMu sync.Mutex
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
		this.IncrementMisses()
		return nil
	}
	result := this.Client.Get(key)
	if result == nil {
		this.IncrementMisses()
		return nil
	}
	this.IncrementHits()
	return utils.Json.Decode(result)
}

func (this *FileCacheStruct) Stats() CacheStats {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	total := this.hits + this.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(this.hits) / float64(total) * 100
	}
	return CacheStats{
		Hits:      this.hits,
		Misses:    this.misses,
		HitRate:   hitRate,
		TotalGets: total,
	}
}

func (this *FileCacheStruct) IncrementHits() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.hits++
}

func (this *FileCacheStruct) IncrementMisses() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.misses++
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

// Incr 自增：文件缓存没有原生原子操作，这里用进程内锁包住「读 -> 加 -> 写」。
// 单实例部署语义正确；多实例部署请使用 redis 驱动（跨进程无法互斥）。
func (this *FileCacheStruct) Incr(key any, delta ...int64) (int64, bool) {
	if this.Client == nil {
		return 0, false
	}

	step := int64(1)
	if len(delta) > 0 {
		step = delta[0]
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	next := cast.ToInt64(this.Get(key)) + step
	if !this.Set(key, next) {
		return 0, false
	}

	return next, true
}

// SetNX 仅当 key 不存在时写入（进程内互斥，多实例请用 redis 驱动）
func (this *FileCacheStruct) SetNX(key any, value any, expire ...any) bool {
	if this.Client == nil {
		return false
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	if this.Has(key) {
		return false
	}

	return this.Set(key, value, expire...)
}

// Expire 重新设置过期时间：文件缓存只能通过「按新 TTL 重写」实现
func (this *FileCacheStruct) Expire(key any, expire any) bool {
	if this.Client == nil {
		return false
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	value := this.Get(key)
	if value == nil {
		return false
	}

	return this.Set(key, value, expire)
}

// ==================== 内存缓存 ====================

type BigCacheStruct struct {
	Client *BigCacheClient
	hits   int64
	misses int64
	mutex  sync.Mutex
	// atomicMu 串行化 Incr / SetNX / Expire 的「读改写」（理由同 FileCacheStruct）
	atomicMu sync.Mutex
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
		this.IncrementMisses()
		return nil
	}
	result := this.Client.Get(key)
	if result == nil {
		this.IncrementMisses()
		return nil
	}
	this.IncrementHits()
	return utils.Json.Decode(result)
}

func (this *BigCacheStruct) Stats() CacheStats {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	total := this.hits + this.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(this.hits) / float64(total) * 100
	}
	return CacheStats{
		Hits:      this.hits,
		Misses:    this.misses,
		HitRate:   hitRate,
		TotalGets: total,
	}
}

func (this *BigCacheStruct) IncrementHits() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.hits++
}

func (this *BigCacheStruct) IncrementMisses() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.misses++
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

// Incr 自增（内存驱动用进程内锁模拟，仅单实例语义）
func (this *BigCacheStruct) Incr(key any, delta ...int64) (int64, bool) {
	if this.Client == nil {
		return 0, false
	}

	step := int64(1)
	if len(delta) > 0 {
		step = delta[0]
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	next := cast.ToInt64(this.Get(key)) + step
	if !this.Set(key, next) {
		return 0, false
	}

	return next, true
}

// SetNX 仅当 key 不存在时写入（内存驱动用进程内锁模拟）
func (this *BigCacheStruct) SetNX(key any, value any, expire ...any) bool {
	if this.Client == nil {
		return false
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	if this.Has(key) {
		return false
	}

	return this.Set(key, value, expire...)
}

// Expire 重新设置过期时间（内存驱动只能按新 TTL 重写）
func (this *BigCacheStruct) Expire(key any, expire any) bool {
	if this.Client == nil {
		return false
	}

	this.atomicMu.Lock()
	defer this.atomicMu.Unlock()

	value := this.Get(key)
	if value == nil {
		return false
	}

	return this.Set(key, value, expire)
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
