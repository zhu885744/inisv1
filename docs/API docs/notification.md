# Notification 消息通知接口文档

## 接口概述

`notification` 控制器用于管理系统通知，支持通知列表查询、未读计数、标记已读、批量操作等功能。当用户收到新回复、被点赞、被收藏、被关注时，系统自动创建通知并通过 WebSocket 实时推送。所有接口均需登录，数据按用户隔离。

### 通知触发机制

| 触发事件 | 通知类型 | 通知对象 | 通知内容 |
| :--- | :--- | :--- | :--- |
| 文章/动态/页面收到新评论 | `comment` | 内容作者 | "xxx 回复了你的{内容标题}" |
| 评论被点赞 | `like` | 评论作者 | "xxx 赞了你的评论「{评论摘要}」" |
| 文章/页面/动态被点赞 | `like` | 内容作者 | "xxx 赞了你的{内容类型}「{内容标题}」" |
| 文章/页面/动态被收藏 | `collect` | 内容作者 | "xxx 收藏了你的{内容类型}「{内容标题}」" |
| 被用户关注 | `follow` | 被关注者 | "xxx 关注了你" |
| 系统消息 | `system` | 指定用户 | 自定义 |

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | one、all、rand、count、sum、min、max、column、remove、delete、clear、restore、save、create、update |
| **业务接口** | list（获取通知列表）、unread-count（获取未读数量）、read（标记已读）、read-all（全部已读）、read-batch（批量已读）、remove-all（清空通知） |

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

### Notification 结构

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 主键，自增 |
| `uid` | int | 接收通知的用户ID |
| `from_uid` | int | 触发通知的用户ID |
| `type` | string | 通知类型：comment/like/follow/system |
| `title` | string | 通知标题 |
| `content` | string | 通知内容 |
| `bind_id` | int | 关联实体ID（文章/评论/用户等） |
| `bind_type` | string | 关联实体类型：article/page/moments/comment/user |
| `is_read` | int | 是否已读：0=未读，1=已读 |
| `json` | any | JSON扩展数据 |
| `text` | any | 文本扩展数据 |
| `result` | any | 复合返回结果（含 from_user 信息） |
| `create_time` | int64 | 创建时间戳 |
| `update_time` | int64 | 更新时间戳 |
| `delete_time` | soft_delete | 软删除时间戳 |

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取通知列表 [业务接口]

- **路径**: `/api/notification/list`
- **方法**: `GET`
- **描述**: 分页获取当前用户的通知列表，支持按类型和已读状态筛选

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `size` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `type` | string | 否 | 通知类型过滤：comment/like/follow/system |
| `is_read` | int | 否 | 已读状态过滤：0=未读，1=已读 |
| `field` | string | 否 | 返回字段，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "data": [
            {
                "id": 5,
                "uid": 100,
                "from_uid": 200,
                "type": "comment",
                "title": "收到新回复",
                "content": "张三 回复了你的文章《Go语言入门》",
                "bind_id": 10,
                "bind_type": "article",
                "is_read": 0,
                "create_time": 1734567890,
                "result": {
                    "from_user": {
                        "id": 200,
                        "nickname": "张三",
                        "avatar": "https://...",
                        "description": "个人简介",
                        "title": "LV.3"
                    }
                }
            }
        ],
        "count": 15,
        "page": 3
    }
}
```

**使用示例**:
```
GET /api/notification/list?type=comment&is_read=0&page=1&size=20
```

**权限说明**: 需要登录，仅返回当前登录用户的通知数据

#### 1.2 获取未读通知数量 [业务接口]

- **路径**: `/api/notification/unread-count`
- **方法**: `GET`
- **描述**: 获取当前用户的未读通知总数

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "count": 5
    }
}
```

**使用示例**:
```javascript
// 前端轮询或Socket推送后更新未读角标
const res = await request.get('/api/notification/unread-count')
// res.data = { count: 5 }
```

**权限说明**: 需要登录

#### 1.3 获取单个通知 [基础接口-获取指定]

- **路径**: `/api/notification/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 通知ID |
| `type` | string | 否 | 通知类型 |
| `is_read` | int | 否 | 已读状态 |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "id": 1,
        "uid": 100,
        "from_uid": 200,
        "type": "like",
        "title": "获得新点赞",
        "content": "李四 赞了你的评论「说得很有道理...」",
        "bind_id": 5,
        "bind_type": "comment",
        "is_read": 0,
        "create_time": 1734567890,
        "result": {
            "from_user": {
                "id": 200,
                "nickname": "李四",
                "avatar": "https://..."
            }
        }
    }
}
```

**权限说明**: 需要登录，用户只能查询自己的通知

#### 1.4 获取所有通知 [基础接口-获取全部]

- **路径**: `/api/notification/all`
- **方法**: `GET`
- **描述**: 分页获取通知记录列表

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
        "count": 50,
        "page": 5
    }
}
```

**权限说明**: 需要登录

#### 1.5 随机获取通知 [基础接口-随机获取]

- **路径**: `/api/notification/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的通知记录

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

**权限说明**: 需要登录

#### 1.6 查询数量 [基础接口-查询数量]

- **路径**: `/api/notification/count`
- **方法**: `GET`
- **描述**: 查询通知记录数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `where` | json | 否 | 条件查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": 50
}
```

**权限说明**: 需要登录

#### 1.7 求和 [基础接口-求和]

- **路径**: `/api/notification/sum`
- **方法**: `GET`
- **描述**: 对指定字段求和

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 求和字段 |
| `where` | json | 否 | 条件查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "is_read": 42
    }
}
```

**权限说明**: 需要登录

#### 1.8 最小值 [基础接口-最小值]

- **路径**: `/api/notification/min`
- **方法**: `GET`
- **描述**: 获取指定字段最小值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1734500000
    }
}
```

**权限说明**: 需要登录

#### 1.9 最大值 [基础接口-最大值]

- **路径**: `/api/notification/max`
- **方法**: `GET`
- **描述**: 获取指定字段最大值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1734600000
    }
}
```

**权限说明**: 需要登录

#### 1.10 查询列 [基础接口-查询列]

- **路径**: `/api/notification/column`
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

**权限说明**: 需要登录

---

### 2. POST 请求接口

#### 2.1 保存通知 [基础接口-保存数据]

- **路径**: `/api/notification/save`
- **方法**: `POST`
- **描述**: 保存通知记录（id为空时新增，不为空时更新）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 通知ID，为空时新增 |
| `uid` | int | 否 | 接收用户ID |
| `from_uid` | int | 否 | 触发用户ID |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `bind_id` | int | 否 | 关联实体ID |
| `bind_type` | string | 否 | 关联实体类型 |
| `is_read` | int | 否 | 已读状态 |

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

**权限说明**: 需要登录

#### 2.2 创建通知 [基础接口-添加数据]

- **路径**: `/api/notification/create`
- **方法**: `POST`
- **描述**: 新增通知记录，同时通过 WebSocket 推送给目标用户

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | **是** | 接收用户ID |
| `from_uid` | int | 否 | 触发用户ID |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `bind_id` | int | 否 | 关联实体ID |
| `bind_type` | string | 否 | 关联实体类型 |
| `is_read` | int | 否 | 已读状态，默认0 |

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

**权限说明**: 需要登录

---

### 3. PUT 请求接口

#### 3.1 标记已读 [业务接口]

- **路径**: `/api/notification/read`
- **方法**: `PUT`
- **描述**: 将指定通知标记为已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 通知ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "标记已读成功！",
    "data": {
        "id": 5
    }
}
```

**错误响应** (400):
```json
{
    "code": 400,
    "msg": "标记已读失败！",
    "data": null
}
```

**权限说明**: 需要登录，只能标记自己的通知

#### 3.2 全部标记已读 [业务接口]

- **路径**: `/api/notification/read-all`
- **方法**: `PUT`
- **描述**: 将当前用户所有未读通知标记为已读

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "全部标记已读成功！",
    "data": null
}
```

**权限说明**: 需要登录

#### 3.3 批量标记已读 [业务接口]

- **路径**: `/api/notification/read-batch`
- **方法**: `PUT`
- **描述**: 批量将多条通知标记为已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表，支持数组 `[1,2,3]` 或字符串 `"1,2,3"` |

**请求示例**:
```json
PUT /api/notification/read-batch
Content-Type: application/json

{
    "ids": [1, 2, 3, 4, 5]
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "批量标记已读成功！",
    "data": {
        "ids": [1, 2, 3, 4, 5]
    }
}
```

**权限说明**: 需要登录，只能标记自己的通知

#### 3.4 更新通知 [基础接口-修改数据]

- **路径**: `/api/notification/update`
- **方法**: `PUT`
- **描述**: 更新通知记录信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 通知ID |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `is_read` | int | 否 | 已读状态 |

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

**权限说明**: 需要登录

#### 3.5 恢复通知 [基础接口-恢复数据]

- **路径**: `/api/notification/restore`
- **方法**: `PUT`
- **描述**: 恢复已软删除的通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "恢复成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 需要登录

---

### 4. DELETE 请求接口

#### 4.1 删除通知 [基础接口-删除]

- **路径**: `/api/notification/remove`
- **方法**: `DELETE`
- **描述**: 软删除指定的通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

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

**权限说明**: 需要登录，只能删除自己的通知

#### 4.2 彻底删除通知 [基础接口-彻底删除]

- **路径**: `/api/notification/delete`
- **方法**: `DELETE`
- **描述**: 永久删除通知记录，不可恢复

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

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

**权限说明**: 需要登录

#### 4.3 清空通知 [业务接口]

- **路径**: `/api/notification/remove-all`
- **方法**: `DELETE`
- **描述**: 清空当前用户的通知，支持按类型批量清空

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `type` | string | 否 | 通知类型：comment/like/follow/system，不传则清空全部 |

**请求示例**:
```json
DELETE /api/notification/remove-all
Content-Type: application/json

{
    "type": "comment"
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "清空成功！",
    "data": {
        "ids": [1, 2, 3, 4, 5]
    }
}
```

**权限说明**: 需要登录，只能清空自己的通知

#### 4.4 清空回收站 [基础接口-清空回收站]

- **路径**: `/api/notification/clear`
- **方法**: `DELETE`
- **描述**: 永久删除所有已软删除的通知记录（清空回收站）

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

**权限说明**: 需要登录

---

### 5. INDEX 接口

- **路径**: `/api/notification/index`
- **方法**: `GET`
- **描述**: 通知首页（无实际功能）

**成功响应** (202):
```json
{
    "code": 202,
    "msg": "没什么用！",
    "data": null
}
```

---

## WebSocket 实时推送

### 推送机制

当系统自动创建通知后，会通过 WebSocket 的 `notification` 消息类型实时推送给目标用户。

### 推送消息格式

```json
{
    "type": "notification",
    "to": "user_100",
    "content": {
        "id": 6,
        "type": "comment",
        "title": "收到新回复",
        "content": "李四 回复了你的动态",
        "bind_id": 20,
        "bind_type": "moments",
        "from_uid": 300,
        "create_time": 1734567900
    }
}
```

### 前端监听

```javascript
import { useSocketStore } from '@/store/socket'

const socketStore = useSocketStore()
socketStore.init()

// 未读通知数变化时自动更新
watch(() => socketStore.unreadNotificationCount, (count) => {
    document.title = count > 0 ? `(${count}) 我的网站` : '我的网站'
})

// 新通知实时追加
watch(() => socketStore.notifications, (notifs) => {
    console.log('收到新通知:', notifs[0])
})
```

---

## 前端使用指南

### 场景一：页面初始化获取通知列表

```javascript
const res = await request.get('/api/notification/list', {
    params: {
        page: 1,
        size: 20,
        order: 'create_time desc'
    }
})
const { data: notifications, count, page } = res.data.data
```

### 场景二：获取未读数量显示角标

```javascript
const res = await request.get('/api/notification/unread-count')
const unreadCount = res.data.data.count
// 在导航栏显示未读角标
```

### 场景三：点击通知标记已读

```javascript
await request.put('/api/notification/read', { id: 5 })
```

### 场景四：一键全部已读

```javascript
await request.put('/api/notification/read-all')
```

### 场景五：按类型筛选通知

```javascript
// 只看评论回复通知
const res = await request.get('/api/notification/list', {
    params: { type: 'comment', page: 1, size: 20 }
})
```

### 场景六：清空某类通知

```javascript
// 清除所有点赞通知
await request.delete('/api/notification/remove-all', {
    data: { type: 'like' }
})
```

### 场景七：跳转到关联内容

```javascript
// 根据 bind_type 和 bind_id 构建跳转链接
function getActionLink(notif) {
    switch (notif.bind_type) {
        case 'article': return `/blog/${notif.bind_id}`
        case 'moments': return `/moment/${notif.bind_id}`
        case 'page':     return `/page/${notif.bind_id}`
        case 'comment':  return null // 需要额外查询评论所属内容
        case 'user':     return `/user/${notif.bind_id}`
        default:         return null
    }
}
```

---

## 特殊说明

### 1. 权限控制
- 所有接口均需登录（`type=login`），未登录返回 401
- 用户只能查看和操作自己的通知数据
- 通知创建由系统自动触发，不对外暴露手动创建接口

### 2. 数据隔离
- 查询接口自动过滤 `uid` 为当前登录用户
- 删除/标记已读操作强制校验 `uid` 匹配
- `columnCount` 等聚合操作同样限定用户范围

### 3. 通知类型说明

| 类型 | 值 | 触发时机 | bind_type | bind_id |
| :--- | :--- | :--- | :--- | :--- |
| 回复通知 | `comment` | 收到新评论 | article/page/moments | 内容ID |
| 点赞通知 | `like` | 评论被点赞 | comment | 评论ID |
| 关注通知 | `follow` | 被用户关注 | user | 关注者ID |
| 系统通知 | `system` | 平台/管理员发送 | 自定义 | 自定义 |

### 4. 缓存策略
- 所有 GET 查询接口均支持缓存
- POST/PUT/DELETE 数据修改后会自动清除相关缓存
- 缓存标签：`[GET]notification`

### 5. 实时推送特性
- 通知创建后立即通过 WebSocket 推送给目标用户
- 如果用户离线，通知仍会持久化到数据库，用户上线后可查询
- 前端 `socketStore` 自动维护 `notifications` 列表和 `unreadNotificationCount` 计数

### 6. 软删除机制
- `remove` 方法执行软删除（设置 `delete_time`），可通过 `restore` 恢复
- `delete` 方法执行物理删除，不可恢复
- `remove-all` 方法执行软删除，支持按类型批量操作
- `clear` 方法彻底删除所有已软删除的数据

### 7. 索引优化
数据库自动创建以下索引以保证查询性能：
- `idx_notifications_uid`：按用户ID快速查询
- `idx_notifications_type`：按通知类型过滤
- `idx_notifications_is_read`：按已读状态筛选
