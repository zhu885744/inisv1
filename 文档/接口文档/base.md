# Base 基础控制器

## 接口概述

`base` 是所有 API 控制器的基类，提供通用的工具方法和中间件功能。该控制器不对外暴露接口，仅作为其他控制器的父类使用。

### 核心功能

| 功能模块 | 说明 |
| :--- | :--- |
| **响应封装** | 统一的 JSON 响应格式封装 |
| **参数处理** | 获取请求参数、请求头信息 |
| **缓存管理** | 缓存启用判断、缓存名称生成 |
| **上下文解析** | 用户信息、路由信息、权限检查 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **201** | 创建成功 | 新增数据成功 |
| **202** | 接受请求 | 请求已接受，无实际功能 |
| **204** | 无内容 | 查询无数据、无可操作数据 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 无操作权限 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

---

## 核心方法

### 1. 响应封装

#### json - 统一响应
```go
func (this base) json(ctx *gin.Context, data, msg, code any)
```

**说明**: 将数据封装为统一的 JSON 响应格式

**响应结构**:
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {}
}
```

---

### 2. 参数处理

#### param - 获取单个参数
```go
func (this base) param(ctx *gin.Context, key string, def ...any) any
```

**说明**: 获取单个请求参数，支持默认值

#### params - 获取全部参数
```go
func (this base) params(ctx *gin.Context, def ...map[string]any) map[string]any
```

**说明**: 获取所有请求参数，支持默认参数合并

#### header - 获取单个请求头
```go
func (this base) header(ctx *gin.Context, key string, def ...any) string
```

**说明**: 获取指定的请求头信息

#### headers - 获取全部请求头
```go
func (this base) headers(ctx *gin.Context) map[string]any
```

**说明**: 获取所有请求头信息

---

### 3. 缓存管理

#### cache.enable - 检查缓存是否启用
```go
func (this cache) enable(ctx *gin.Context) bool
```

**说明**: 根据请求参数和系统配置判断是否启用缓存

#### cache.name - 生成缓存名称
```go
func (this cache) name(ctx *gin.Context) string
```

**说明**: 根据请求方法、路径和参数生成唯一的缓存键名

---

### 4. 上下文解析

#### meta.user - 获取用户信息
```go
func (this meta) user(ctx *gin.Context) model.Users
```

**说明**: 从上下文中解析当前登录用户信息

#### meta.route - 获取路由信息
```go
func (this meta) route(ctx *gin.Context) model.AuthRules
```

**说明**: 从上下文中解析当前路由信息

#### meta.rules - 获取权限列表
```go
func (this meta) rules(ctx *gin.Context) []any
```

**说明**: 获取当前用户的权限列表

#### meta.root - 检查超级管理员权限
```go
func (this meta) root(ctx *gin.Context) bool
```

**说明**: 判断当前用户是否拥有超级管理员权限

#### meta.limit - 分页限制
```go
func (this meta) limit(ctx *gin.Context) int
```

**说明**: 获取分页限制数量（受系统配置限制）

---

### 5. 方法调用

#### call - 方法调用封装
```go
func (this base) call(allow map[string]any, name string, params ...any) error
```

**说明**: 安全的方法调用，验证方法名是否在允许列表中

---

## 接口协议

`base` 控制器不直接对外暴露接口，作为基类被其他控制器继承使用。

### 继承示例

```go
type Users struct {
    base  // 继承 base 控制器
}
```

### 通用接口结构

所有继承 `base` 的控制器遵循以下接口协议：

```go
type ApiInterface interface {
    IGET(ctx *gin.Context)
    IPUT(ctx *gin.Context)
    IDEL(ctx *gin.Context)
    IPOST(ctx *gin.Context)
    INDEX(ctx *gin.Context)
}
```

---

## 特殊说明

### 1. 缓存策略
- 所有查询接口默认支持缓存
- 数据修改后会自动清除相关缓存
- 缓存名称包含请求参数的哈希值，确保唯一性

### 2. 参数安全
- 参数获取支持类型自动转换
- 默认参数机制防止空指针异常
- 支持请求体、URL 参数、表单数据的统一获取

### 3. 权限控制
- 内置超级管理员权限检查
- 支持上下文挂载的用户信息解析
- 权限验证基于角色组进行