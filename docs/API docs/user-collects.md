# User Collects 用户收藏接口文档

## 接口概述

`user-collects` 控制器用于管理用户收藏数据，采用"存在即收藏"的设计模式：记录存在表示已收藏，删除记录表示取消收藏。支持收藏、取消收藏、检查收藏状态、获取收藏列表及批量查询收藏数量等功能。所有接口均支持缓存优化和权限控制。

### 设计模式

- **收藏**：在 `user_collects` 表中新增一条记录
- **取消收藏**：从 `user_collects` 表中删除对应记录
- **是否已收藏**：通过查询记录是否存在来判断
- **唯一约束**：`(uid, target_type, target_id)` 联合唯一索引，防止重复收藏
- **不支持收藏用户**：`target_type` 不支持 `user` 类型

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | one、all、rand、count、sum、min、max、column、remove、delete、clear、save、create、update |
| **业务接口** | collect（收藏）、uncollect（取消收藏）、is-collected（检查是否已收藏）、collects（获取我的收藏列表）、counts（批量查询收藏数量） |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **202** | 接受请求 | 请求已接受（无实际功能） |
| **204** | 无内容 | 查询无数据、无可操作数据 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 无操作权限 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

---

## 数据模型

### UserCollects 结构

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 主键，自增 |
| `uid` | int | 用户ID |
| `target_type` | string | 目标类型：article/page/moment/comment |
| `target_id` | int | 目标ID |
| `json` | any | JSON扩展数据 |
| `text` | any | 文本扩展数据 |
| `result` | any | 复合返回结果（含author信息） |
| `create_time` | int64 | 创建时间戳 |

> **注意**：模型已移除 `status` 和 `delete_time` 字段，采用"记录存在即表示已收藏"的设计。

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取单个收藏记录 [基础接口-获取指定]

- **路径**: `/api/user-collects/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个收藏记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 收藏记录ID |
| `uid` | int | 否 | 用户ID |
| `target_type` | string | 否 | 目标类型 |
| `target_id` | int | 否 | 目标ID |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "id": 1,
        "uid": 1,
        "target_type": "article",
        "target_id": 10,
        "create_time": 1699900000,
        "result": {
            "author": {
                "id": 2,
                "nickname": "作者昵称",
                "avatar": "",
                "description": ""
            }
        }
    }
}
```

**权限说明**: 非管理员只能查看自己的收藏记录

#### 1.2 获取所有收藏记录 [基础接口-获取全部]

- **路径**: `/api/user-collects/all`
- **方法**: `GET`
- **描述**: 分页获取收藏记录列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `size` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "data": [...],
        "count": 100,
        "page": 10
    }
}
```

**权限说明**: 非管理员只能查看自己的收藏记录

#### 1.3 随机获取收藏记录 [基础接口-随机获取]

- **路径**: `/api/user-collects/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的收藏记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `size` | int | 否 | 返回数量 |
| `except` | string | 否 | 排除的ID，逗号分隔 |
| `field` | string | 否 | 返回字段 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "好的！",
    "data": [...]
}
```

**权限说明**: 非管理员只能获取自己的收藏记录

#### 1.4 查询数量 [基础接口-查询数量]

- **路径**: `/api/user-collects/count`
- **方法**: `GET`
- **描述**: 查询收藏记录数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `where` | json | 否 | 条件查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": 100
}
```

#### 1.5 求和 [基础接口-求和]

- **路径**: `/api/user-collects/sum`
- **方法**: `GET`
- **描述**: 对指定字段求和

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 求和字段 |
| `where` | json | 否 | 条件查询 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "target_id": 10000
    }
}
```

#### 1.6 最小值 [基础接口-最小值]

- **路径**: `/api/user-collects/min`
- **方法**: `GET`
- **描述**: 获取指定字段最小值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1699900000
    }
}
```

#### 1.7 最大值 [基础接口-最大值]

- **路径**: `/api/user-collects/max`
- **方法**: `GET`
- **描述**: 获取指定字段最大值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1700000000
    }
}
```

#### 1.8 查询列 [基础接口-查询列]

- **路径**: `/api/user-collects/column`
- **方法**: `GET`
- **描述**: 获取指定字段列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |
| `order` | string | 否 | 排序字段 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": [1, 2, 3]
}
```

#### 1.9 检查是否已收藏 [业务接口]

- **路径**: `/api/user-collects/is-collected`
- **方法**: `GET`
- **描述**: 检查当前用户是否已收藏指定目标，同时返回收藏总数

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moment/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "is_collected": true,
        "count": 15
    }
}
```

**权限说明**: 公共接口，无需登录。未登录时 `is_collected` 始终为 `false`

#### 1.10 获取我的收藏列表 [业务接口]

- **路径**: `/api/user-collects/collects`
- **方法**: `GET`
- **描述**: 获取当前用户的收藏列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | 否 | 目标类型过滤：article/page/moment/comment |
| `field` | string | 否 | 返回字段，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "list": [
            {
                "id": 1,
                "target_type": "article",
                "target_id": 10,
                "result": {
                    "author": {
                        "id": 2,
                        "nickname": "作者昵称",
                        "avatar": "",
                        "description": ""
                    }
                }
            }
        ],
        "count": 10
    }
}
```

**权限说明**: 需要登录

#### 1.11 批量查询收藏数量 [业务接口]

- **路径**: `/api/user-collects/counts`
- **方法**: `GET`
- **描述**: 批量查询指定目标类型和目标ID列表的收藏数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moment/comment |
| `target_ids` | string | **是** | 目标ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "counts": {
            "1": 10,
            "2": 5,
            "3": 3
        }
    }
}
```

**使用示例**:
```
GET /api/user-collects/counts?target_type=article&target_ids=1,2,3
```

**权限说明**: 公共接口，无需登录

---

### 2. POST 请求接口

#### 2.1 保存收藏 [基础接口-保存数据]

- **路径**: `/api/user-collects/save`
- **方法**: `POST`
- **描述**: 保存收藏记录（新增或更新，id为空时新增）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 收藏记录ID，为空时新增 |
| `uid` | int | 否 | 用户ID（管理员） |
| `target_type` | string | 否 | 目标类型 |
| `target_id` | int | 否 | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1
    }
}
```

**权限说明**: 非管理员只能操作自己的数据

#### 2.2 创建收藏 [基础接口-添加数据]

- **路径**: `/api/user-collects/create`
- **方法**: `POST`
- **描述**: 新增收藏记录。注意：target_type 为 "user" 时会返回错误

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 否 | 用户ID（管理员） |
| `target_type` | string | **是** | 目标类型（不支持 "user"） |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1
    }
}
```

**错误响应** (400 - 不支持收藏用户):
```json
{
    "code": 400,
    "msg": "不支持收藏用户！",
    "data": null
}
```

#### 2.3 收藏 [业务接口]

- **路径**: `/api/user-collects/collect`
- **方法**: `POST`
- **描述**: 收藏指定目标。已收藏时返回错误提示

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moment/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "收藏成功！",
    "data": {
        "target_type": "moment",
        "target_id": 10
    }
}
```

**错误响应** (400 - 已收藏):
```json
{
    "code": 400,
    "msg": "已经收藏过了",
    "data": null
}
```

**权限说明**: 需要登录。不支持收藏用户（target_type=user）

#### 2.4 取消收藏 [业务接口]

- **路径**: `/api/user-collects/uncollect`
- **方法**: `POST` / `PUT`
- **描述**: 取消收藏指定目标。删除对应记录，即使记录不存在也返回成功

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moment/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "取消收藏成功！",
    "data": {
        "target_type": "moment",
        "target_id": 10
    }
}
```

**权限说明**: 需要登录

---

### 3. PUT 请求接口

#### 3.1 更新收藏 [基础接口-修改数据]

- **路径**: `/api/user-collects/update`
- **方法**: `PUT`
- **描述**: 更新收藏记录信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 收藏记录ID |
| `target_type` | string | 否 | 目标类型 |
| `target_id` | int | 否 | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "更新成功！",
    "data": {
        "id": 1
    }
}
```

**权限说明**: 非管理员只能更新自己的数据

#### 3.2 取消收藏 [业务接口]

> 与 2.4 相同，支持 `POST` 和 `PUT` 两种请求方式

- **路径**: `/api/user-collects/uncollect`
- **方法**: `PUT`
- **描述**: 取消收藏指定目标

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moment/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "取消收藏成功！",
    "data": {
        "target_type": "moment",
        "target_id": 10
    }
}
```

**权限说明**: 需要登录

---

### 4. DELETE 请求接口

#### 4.1 删除收藏记录 [基础接口-删除]

- **路径**: `/api/user-collects/remove`
- **方法**: `DELETE`
- **描述**: 删除指定的收藏记录（物理删除）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 收藏记录ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 非管理员只能删除自己的数据

#### 4.2 彻底删除收藏 [基础接口-彻底删除]

- **路径**: `/api/user-collects/delete`
- **方法**: `DELETE`
- **描述**: 永久删除收藏记录，不可恢复

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 收藏记录ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 非管理员只能删除自己的数据

#### 4.3 清空所有收藏 [基础接口-清空]

- **路径**: `/api/user-collects/clear`
- **方法**: `DELETE`
- **描述**: 清空所有收藏记录

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "清空成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

---

### 5. INDEX 接口

- **路径**: `/api/user-collects/index`
- **方法**: `GET`
- **描述**: 用户收藏首页（无实际功能）

**成功响应** (202):
```json
{
    "code": 202,
    "msg": "没什么用！",
    "data": null
}
```

---

## 前端使用指南

### 场景一：展示文章收藏数并判断是否已收藏

```javascript
// 获取收藏状态和数量（一个请求搞定）
const res = await request.get('/api/user-collects/is-collected', {
    target_type: 'article',
    target_id: articleId
})
// res.data = { is_collected: true, count: 15 }
```

### 场景二：执行收藏/取消收藏

```javascript
// 收藏
await request.post('/api/user-collects/collect', {
    target_type: 'article',
    target_id: articleId
})

// 取消收藏
await request.post('/api/user-collects/uncollect', {
    target_type: 'article',
    target_id: articleId
})
```

### 场景三：列表页批量获取收藏数

```javascript
// 批量获取多个文章的收藏数
const res = await request.get('/api/user-collects/counts', {
    target_type: 'article',
    target_ids: '1,2,3,4,5'
})
// res.data.counts = { "1": 10, "2": 5, "3": 3 }
```

### 场景四：我的收藏列表

```javascript
// 获取当前用户的收藏历史
const res = await request.get('/api/user-collects/collects', {
    target_type: 'article',  // 可选，筛选类型
    field: 'id,target_type,target_id,result'  // 可选，指定返回字段
})
```

---

## 特殊说明

### 1. 缓存策略
- 所有 GET 查询接口均支持缓存
- POST/PUT/DELETE 数据修改后会自动清除相关缓存

### 2. 权限控制
- 普通用户只能操作自己的收藏数据
- 管理员可以操作所有用户的收藏数据

### 3. 目标类型说明

| 类型 | 说明 |
| :--- | :--- |
| `article` | 文章 |
| `page` | 独立页面 |
| `moment` | 动态 |
| `comment` | 评论 |

> **注意**：收藏不支持 `user` 类型，收藏用户会返回错误。

### 4. 收藏奖励机制
- 用户收藏文章/页面/动态后，会给作者增加经验值奖励
- 奖励类型：article-collect（内容被收藏）
- 奖励规则由经验值配置控制

### 5. 设计说明
- **无 status 字段**：通过记录是否存在来判断收藏状态，简化数据模型
- **无软删除**：取消收藏即物理删除记录，不存在回收站概念
- **唯一约束**：`(uid, target_type, target_id)` 保证每个用户对同一目标只能有一条收藏记录
- **不支持收藏用户**：与点赞不同，收藏不支持对用户的收藏操作

### 6. 收藏/取消收藏特性
- 收藏时如已存在记录，会返回"已经收藏过了"错误
- 取消收藏不检查记录是否存在，直接执行删除操作
- uncollect 同时支持 POST 和 PUT 两种请求方式