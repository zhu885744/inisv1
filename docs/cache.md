# Cache 缓存系统 - 开发者文档

## 概述

缓存系统提供统一的缓存管理接口，支持 Redis、文件缓存、内存缓存三种驱动模式。包含缓存命中统计、标签批量清理等高级特性。

### 缓存模式

| 模式 | 值 | 说明 |
| :--- | :--- | :--- |
| **Redis** | `redis` | 使用 Redis 作为缓存存储 |
| **文件缓存** | `file` | 使用文件系统存储缓存数据 |
| **内存缓存** | `ram` | 使用内存（BigCache）存储缓存数据 |

### 配置文件

路径：`config/cache.toml`

```toml
[cache]
open = true
default = "file"

[cache.local]
expire = 300

[cache.redis]
host = "localhost"
port = "6379"
password = ""
expire = "2 * 60 * 60"
prefix = "inis:"
database = 0

[cache.file]
expire = "2 * 60 * 60"
path = "runtime/cache"
prefix = "inis_"

[cache.ram]
expire = "2 * 60 * 60"
```

---

## 接口规范

### CacheInterface

```go
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
}
```

### CacheStats 统计结构体

```go
type CacheStats struct {
    Hits      int64   // 缓存命中次数
    Misses    int64   // 缓存未命中次数
    HitRate   float64 // 缓存命中率（百分比）
    TotalGets int64   // 总查询次数
}
```

---

## 使用示例

### 基本操作

```go
package main

import "inis/app/facade"

func main() {
    // 设置缓存
    facade.Cache.Set("user:1", map[string]any{
        "id":       1,
        "nickname": "张三",
        "email":    "zhangsan@example.com",
    }, 3600) // 3600秒过期

    // 获取缓存
    user := facade.Cache.Get("user:1")
    if user != nil {
        fmt.Println(user)
    }

    // 检查缓存是否存在
    exists := facade.Cache.Has("user:1")

    // 删除缓存
    facade.Cache.Del("user:1")
}
```

### 前缀删除

```go
// 删除所有以 article: 开头的缓存
facade.Cache.DelPrefix("article:")
```

### 标签删除

```go
// 删除带有指定标签的缓存
facade.Cache.DelTags("article", "home")
```

### 获取统计信息

```go
stats := facade.Cache.Stats()
fmt.Printf("命中率: %.2f%%\n", stats.HitRate)
fmt.Printf("命中次数: %d\n", stats.Hits)
fmt.Printf("未命中次数: %d\n", stats.Misses)
fmt.Printf("总查询次数: %d\n", stats.TotalGets)
```

### 切换缓存驱动

```go
// 切换为 Redis 驱动
facade.NewCache("redis")

// 切换为文件缓存驱动
facade.NewCache("file")

// 切换为内存缓存驱动
facade.NewCache("ram")
```

---

## 特殊说明

### 1. 缓存命中统计

- 每次调用 `Get()` 方法时自动更新统计
- 命中时调用 `IncrementHits()`
- 未命中时调用 `IncrementMisses()`
- 统计数据使用互斥锁保证并发安全
- 统计数据仅存储在内存中，重启后重置

### 2. 缓存标签机制

缓存标签用于批量管理相关缓存：

```go
// 设置带有标签的缓存
facade.Cache.Set("article:1", data, 3600)
facade.Cache.Set("article:2", data, 3600)

// 使用标签清除相关缓存
facade.Cache.DelTags("article")
```

**标签匹配规则**:
- Redis 模式：使用 `*标签*` 通配符匹配键名
- 文件缓存模式：使用文件系统匹配
- 内存缓存模式：使用模糊匹配算法

### 3. 配置热更新

缓存配置支持热更新：
- 修改 `config/cache.toml` 文件后会自动生效
- 系统会重新初始化所有缓存驱动实例
- 当前使用的缓存驱动会根据 `default` 配置重新选择

### 4. 缓存键名前缀

- Redis 模式：默认前缀为 `inis:`
- 文件缓存模式：默认前缀为 `inis_`
- 内存缓存模式：默认前缀为 `cache_`
- 前缀会自动添加到所有缓存键名前

### 5. 过期时间表达式

配置文件中的 `expire` 字段支持表达式计算：

```toml
expire = "2 * 60 * 60"  # 2小时 = 7200秒
expire = "60 * 60"      # 1小时 = 3600秒
expire = "300"          # 5分钟 = 300秒
```

---

## 驱动实现差异

| 方法 | Redis | FileCache | BigCache |
| :--- | :--- | :--- | :--- |
| `Has` | 使用 EXISTS 命令 | 文件系统检查 | map 检查 |
| `Get` | 使用 GET 命令 | 文件读取 | BigCache.Get |
| `Set` | 使用 SET 命令 | 文件写入 | BigCache.Set |
| `Del` | 使用 DEL 命令 | 文件删除 | BigCache.Delete |
| `DelPrefix` | 使用 KEYS + DEL | 文件遍历删除 | map 遍历删除 |
| `DelTags` | 使用 KEYS + DEL | 文件遍历删除 | 模糊匹配删除 |
| `Clear` | 使用 FLUSHDB | 删除目录 | 重置所有缓存 |
| `Stats` | 内存统计 | 内存统计 | 内存统计 |
