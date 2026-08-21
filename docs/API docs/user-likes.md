# User Likes 用户点赞接口文档

## 接口概述

`user-likes` 控制器用于管理用户点赞数据，采用"存在即点赞"的设计模式：记录存在表示已点赞，删除记录表示取消点赞。支持点赞、取消点赞、检查点赞状态、获取点赞列表及批量查询等功能。所有接口均支持缓存优化和权限控制。

### 设计模式

- **点赞**：在 `user_likes` 表中新增一条记录
- **取消点赞**：从 `user_likes` 表中删除对应记录
- **是否已点赞**：通过查询记录是否存在来判断
- **唯一约束**：`(uid, target_type, target_id)` 联合唯一索引，防止重复点赞

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | one、all、rand、count、sum、min、max、column、remove、delete、clear、save、create、update |
| **业务接口** | like（点赞）、unlike（取消点赞）、is-liked（检查是否已点赞）、likes（获取我的点赞列表）、counts（批量查询点赞数量） |

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

### UserLikes 结构

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 主键，自增 |
| `uid` | int | 用户ID |
| `target_type` | string | 目标类型：article/page/moments/comment |
| `target_id` | int | 目标ID |
| `json` | any | JSON扩展数据 |
| `text` | any | 文本扩展数据 |
| `result` | any | 复合返回结果（含author信息） |
| `create_time` | int64 | 创建时间戳 |

> **注意**：模型已移除 `status` 和 `delete_time` 字段，采用"记录存在即表示已点赞"的设计。

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取单个点赞记录 [基础接口-获取指定]

- **路径**: `/api/user-likes/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个点赞记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 点赞记录ID |
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

**权限说明**: 非管理员只能查看自己的点赞记录

#### 1.2 获取所有点赞记录 [基础接口-获取全部]

- **路径**: `/api/user-likes/all`
- **方法**: `GET`
- **描述**: 分页获取点赞记录列表

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

**权限说明**: 非管理员只能查看自己的点赞记录

#### 1.3 随机获取点赞记录 [基础接口-随机获取]

- **路径**: `/api/user-likes/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的点赞记录

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

**权限说明**: 非管理员只能获取自己的点赞记录

#### 1.4 查询数量 [基础接口-查询数量]

- **路径**: `/api/user-likes/count`
- **方法**: `GET`
- **描述**: 查询点赞记录数量

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

- **路径**: `/api/user-likes/sum`
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

- **路径**: `/api/user-likes/min`
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

- **路径**: `/api/user-likes/max`
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

- **路径**: `/api/user-likes/column`
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

#### 1.9 检查是否已点赞 [业务接口]

- **路径**: `/api/user-likes/is-liked`
- **方法**: `GET`
- **描述**: 检查当前用户是否已点赞指定目标，同时返回点赞总数

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moments/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "is_liked": true,
        "count": 42
    }
}
```

**权限说明**: 公共接口，无需登录。未登录时 `is_liked` 始终为 `false`

#### 1.10 获取点赞列表 [业务接口]

- **路径**: `/api/user-likes/likes`
- **方法**: `GET`
- **描述**: 获取指定用户（或当前登录用户）的点赞列表。**公共接口**，任何人都能通过 `uid` 查询对应用户的点赞数据

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 否 | 目标用户ID。未登录时**必填**；已登录时若不传则查询自己 |
| `target_type` | string | 否 | 目标类型过滤：article/page/moments/comment |
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

**权限说明**: 公共接口，无需登录。
- 未登录时必须传 `uid` 参数，否则返回 `code:400, msg:"请指定要查询的用户（uid）！"`
- 查询他人点赞时，若对方隐私设置中「公开我的点赞」为关闭（likes=0），返回 `private:true` 及提示「对方设置了私密，无法查看！」
- 查询自己的点赞不受隐私限制

#### 1.11 批量查询点赞数量 [业务接口]

- **路径**: `/api/user-likes/counts`
- **方法**: `GET`
- **描述**: 批量查询指定目标类型和目标ID列表的点赞数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moments/comment |
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
GET /api/user-likes/counts?target_type=article&target_ids=1,2,3
```

**权限说明**: 公共接口，无需登录

---

### 2. POST 请求接口

#### 2.1 保存点赞 [基础接口-保存数据]

- **路径**: `/api/user-likes/save`
- **方法**: `POST`
- **描述**: 保存点赞记录（新增或更新，id为空时新增）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 点赞记录ID，为空时新增 |
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

#### 2.2 创建点赞 [基础接口-添加数据]

- **路径**: `/api/user-likes/create`
- **方法**: `POST`
- **描述**: 新增点赞记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 否 | 用户ID（管理员） |
| `target_type` | string | **是** | 目标类型 |
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

#### 2.3 点赞 [业务接口]

- **路径**: `/api/user-likes/like`
- **方法**: `POST`
- **描述**: 点赞指定目标。已点赞时返回错误提示

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moments/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "点赞成功！",
    "data": {
        "target_type": "moments",
        "target_id": 10
    }
}
```

**错误响应** (400 - 已点赞):
```json
{
    "code": 400,
    "msg": "已经点赞过了",
    "data": null
}
```

**权限说明**: 需要登录

#### 2.4 取消点赞 [业务接口]

- **路径**: `/api/user-likes/unlike`
- **方法**: `POST` / `PUT`
- **描述**: 取消点赞指定目标。删除对应记录，即使记录不存在也返回成功

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moments/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "取消点赞成功！",
    "data": {
        "target_type": "moments",
        "target_id": 10
    }
}
```

**权限说明**: 需要登录

---

### 3. PUT 请求接口

#### 3.1 更新点赞 [基础接口-修改数据]

- **路径**: `/api/user-likes/update`
- **方法**: `PUT`
- **描述**: 更新点赞记录信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 点赞记录ID |
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

#### 3.2 取消点赞 [业务接口]

> 与 2.4 相同，支持 `POST` 和 `PUT` 两种请求方式

- **路径**: `/api/user-likes/unlike`
- **方法**: `PUT`
- **描述**: 取消点赞指定目标

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | **是** | 目标类型：article/page/moments/comment |
| `target_id` | int | **是** | 目标ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "取消点赞成功！",
    "data": {
        "target_type": "moments",
        "target_id": 10
    }
}
```

**权限说明**: 需要登录

---

### 4. DELETE 请求接口

#### 4.1 删除点赞记录 [基础接口-删除]

- **路径**: `/api/user-likes/remove`
- **方法**: `DELETE`
- **描述**: 删除指定的点赞记录（物理删除）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 点赞记录ID列表，逗号分隔 |

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

#### 4.2 彻底删除点赞 [基础接口-彻底删除]

- **路径**: `/api/user-likes/delete`
- **方法**: `DELETE`
- **描述**: 永久删除点赞记录，不可恢复

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 点赞记录ID列表，逗号分隔 |

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

#### 4.3 清空所有点赞 [基础接口-清空]

- **路径**: `/api/user-likes/clear`
- **方法**: `DELETE`
- **描述**: 清空所有点赞记录

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

- **路径**: `/api/user-likes/index`
- **方法**: `GET`
- **描述**: 用户点赞首页（无实际功能）

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

### 场景一：展示文章点赞数并判断是否已点赞

```javascript
// 获取点赞状态和数量（一个请求搞定）
const res = await request.get('/api/user-likes/is-liked', {
    target_type: 'article',
    target_id: articleId
})
// res.data = { is_liked: true, count: 42 }
```

### 场景二：执行点赞/取消点赞

```javascript
// 点赞
await request.post('/api/user-likes/like', {
    target_type: 'article',
    target_id: articleId
})

// 取消点赞
await request.post('/api/user-likes/unlike', {
    target_type: 'article',
    target_id: articleId
})
```

### 场景三：列表页批量获取点赞数

```javascript
// 批量获取多个文章的点赞数
const res = await request.get('/api/user-likes/counts', {
    target_type: 'article',
    target_ids: '1,2,3,4,5'
})
// res.data.counts = { "1": 10, "2": 5, "3": 3 }
```

### 场景四：我的点赞列表

```javascript
// 获取当前用户的点赞历史
const res = await request.get('/api/user-likes/likes', {
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
- 普通用户只能操作自己的点赞数据
- 管理员可以操作所有用户的点赞数据

### 3. 目标类型说明

| 类型 | 说明 |
| :--- | :--- |
| `article` | 文章 |
| `page` | 独立页面 |
| `moments` | 动态 |
| `comment` | 评论 |

### 4. 点赞奖励机制
- 用户点赞文章/页面/动态/评论后，会给作者增加经验值奖励
- 奖励类型：article-like（内容获赞）、comment-like（评论获赞）
- 奖励规则由经验值配置控制

### 5. 设计说明
- **无 status 字段**：通过记录是否存在来判断点赞状态，简化数据模型
- **无软删除**：取消点赞即物理删除记录，不存在回收站概念
- **唯一约束**：`(uid, target_type, target_id)` 保证每个用户对同一目标只能有一条点赞记录

### 6. 点赞/取消点赞特性
- 点赞时如已存在记录，会返回"已经点赞过了"错误
- 取消点赞不检查记录是否存在，直接执行删除操作
- unlike 同时支持 POST 和 PUT 两种请求方式