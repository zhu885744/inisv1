# 消息通知 API 使用文档

## 概述

消息通知系统提供完整的通知管理功能，支持系统通知的获取、标记已读、删除等操作。当用户收到新回复、评论被点赞、被关注时，系统会自动创建通知并通过 WebSocket 实时推送给用户。

### 通知类型

| 类型 | 值 | 触发条件 | 通知标题 |
|------|-----|---------|---------|
| 回复通知 | `comment` | 用户发布的内容收到新评论 | 收到新回复 |
| 点赞通知 | `like` | 用户的评论被点赞 | 获得新点赞 |
| 关注通知 | `follow` | 用户被其他人关注 | 有新关注 |
| 系统通知 | `system` | 平台或管理员发送的系统消息 | 由发送方自定义 |

### 通知数据结构

```json
{
  "id": 1,
  "uid": 100,
  "from_uid": 200,
  "type": "comment",
  "title": "收到新回复",
  "content": "张三 回复了你的文章标题...",
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
```

### 通用说明

- **所有接口均需登录**，未登录返回 401
- 响应格式统一为 `{ code, msg, data }`
- `code=200` 表示成功，`204` 表示无数据
- 支持软删除（`remove` 方法），可通过 `restore` 恢复
- `withTrashed=true` 可查询已删除的通知
- `onlyTrashed=true` 仅查询已删除的通知

---

## 一、获取通知列表

### GET /api/notification/list

分页获取当前用户的通知列表。

**请求参数**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | int | 1 | 页码 |
| size | int | 10 | 每页数量 |
| order | string | create_time desc | 排序规则 |
| type | string | - | 通知类型过滤：comment/like/follow/system |
| is_read | int | - | 已读状态过滤：0=未读，1=已读 |
| field | string/array | - | 返回字段过滤，如 "id,title,content" |

**请求示例**

```
GET /api/notification/list?page=1&size=20&type=comment&is_read=0
```

**响应示例**

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
        "content": "张三 回复了你的文章",
        "bind_id": 10,
        "bind_type": "article",
        "is_read": 0,
        "create_time": 1734567890,
        "result": {
          "from_user": {
            "id": 200,
            "nickname": "张三",
            "avatar": "https://..."
          }
        }
      }
    ],
    "count": 1,
    "page": 1
  }
}
```

---

## 二、获取未读通知数

### GET /api/notification/unread-count

获取当前用户的未读通知数量。

**请求参数**

无额外参数。

**请求示例**

```
GET /api/notification/unread-count
```

**响应示例**

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "count": 5
  }
}
```

---

## 三、标记已读

### PUT /api/notification/read

标记指定通知为已读。

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | int | 是 | 通知ID |

**请求示例**

```
PUT /api/notification/read
Content-Type: application/json

{ "id": 5 }
```

**响应示例**

```json
{
  "code": 200,
  "msg": "标记已读成功！",
  "data": { "id": 5 }
}
```

---

## 四、全部标记已读

### PUT /api/notification/read-all

将当前用户所有未读通知标记为已读。

**请求参数**

无额外参数。

**请求示例**

```
PUT /api/notification/read-all
```

**响应示例**

```json
{
  "code": 200,
  "msg": "全部标记已读成功！",
  "data": null
}
```

---

## 五、批量标记已读

### PUT /api/notification/read-batch

批量标记多条通知为已读。

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | array/string | 是 | 通知ID列表，支持数组 `[1,2,3]` 或字符串 `"1,2,3"` |

**请求示例**

```
PUT /api/notification/read-batch
Content-Type: application/json

{ "ids": [1, 2, 3] }
```

**响应示例**

```json
{
  "code": 200,
  "msg": "批量标记已读成功！",
  "data": { "ids": [1, 2, 3] }
}
```

---

## 六、删除通知

### DELETE /api/notification/remove

软删除指定通知（可通过 `restore` 恢复）。

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| ids | array/string | 是 | 通知ID列表 |

**请求示例**

```
DELETE /api/notification/remove
Content-Type: application/json

{ "ids": [5] }
```

**响应示例**

```json
{
  "code": 200,
  "msg": "删除成功！",
  "data": { "ids": [5] }
}
```

---

## 七、清空通知

### DELETE /api/notification/remove-all

清空当前用户的通知。支持按类型清空。

**请求参数**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 否 | 通知类型，不传则清空所有类型 |

**请求示例**

```
DELETE /api/notification/remove-all
Content-Type: application/json

{ "type": "comment" }
```

**响应示例**

```json
{
  "code": 200,
  "msg": "清空成功！",
  "data": { "ids": [1, 2, 3] }
}
```

---

## 八、WebSocket 实时推送

当新通知创建时，系统会通过 WebSocket 自动推送给目标用户。

### 消息格式

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

### 前端监听示例

```js
import { useSocketStore } from '@/store/socket'

const socketStore = useSocketStore()
socketStore.init() // 初始化连接

// 监听实时通知（store 内部已注册 notification 事件）
watch(() => socketStore.unreadNotificationCount, (count) => {
  console.log(`当前未读通知数：${count}`)
})
```

---

## 九、标准 CRUD 方法（通用）

### GET /api/notification/one - 获取单条通知

```
GET /api/notification/one?id=5
```

### GET /api/notification/all - 获取全部通知（分页）

```
GET /api/notification/all?page=1&size=20&order=create_time desc
```

### GET /api/notification/count - 查询通知数量

```
GET /api/notification/count?is_read=0
```

响应：`{ "code": 200, "msg": "查询成功！", "data": 5 }`

### POST /api/notification/save - 添加/修改通知

```json
POST /api/notification/save

{
  "uid": 100,
  "type": "system",
  "title": "系统公告",
  "content": "欢迎使用消息通知系统！"
}
```

### PUT /api/notification/update - 更新通知

```json
PUT /api/notification/update

{
  "id": 1,
  "is_read": 1
}
```

### DELETE /api/notification/delete - 彻底删除

```json
DELETE /api/notification/delete

{ "ids": [1, 2] }
```

### PUT /api/notification/restore - 恢复已删除通知

```json
PUT /api/notification/restore

{ "ids": [1, 2] }
```

---

## 十、前端消息中心集成

主题前端已内置消息中心页面，访问路径：`/messages`

### 功能特性

| 功能 | 说明 |
|------|------|
| 分类筛选 | 支持全部/未读/回复/点赞/关注/系统 六个分类 |
| 标记已读 | 支持单条标记和全部标记 |
| 删除管理 | 支持单条删除和按分类清空 |
| 分页浏览 | 支持分页加载历史通知 |
| 实时推送 | 新通知到达时自动更新列表和未读计数 |
| 详情跳转 | 点击通知可跳转到关联内容（文章/动态/用户主页等） |

### 状态码对照

| code | 说明 |
|------|------|
| 200 | 操作成功 |
| 204 | 查询成功但无数据 |
| 400 | 参数错误（如 ids 为空） |
| 401 | 未登录或 Token 失效 |
| 403 | 权限不足 |
