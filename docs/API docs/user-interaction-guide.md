# 用户互动数据查询指南

> 本文档说明如何查询**指定用户**的粉丝、关注、点赞、收藏数据，适用于用户主页（author 页面）等场景。

---

## 一、查询数据总览

| 需求 | 使用的 API | 说明 |
|------|-----------|------|
| 用户粉丝列表 | `/api/user-follows/followers` | 返回粉丝用户信息（含头像、昵称等） |
| 用户关注列表 | `/api/user-follows/following` | 返回关注的用户信息 |
| 用户粉丝数量 | `/api/user-follows/counts` | 返回粉丝数量 |
| 用户关注数量 | `/api/user-follows/counts` | 返回关注数量 |
| 用户点赞列表 | `/api/user-likes/likes` | 返回点赞记录（关联目标信息） |
| 用户点赞数量 | `/api/user-likes/counts` | 返回用户总点赞数 |
| 用户收藏列表 | `/api/user-collects/collects` | 返回收藏记录（关联目标信息） |
| 用户收藏数量 | `/api/user-collects/counts` | 返回用户总收藏数 |
| 是否关注某用户 | `/api/user-follows/is-following` | 当前用户是否关注目标用户 |
| 是否点赞/收藏 | `/api/user-likes/is-liked` / `/api/user-collects/is-collected` | 当前用户是否点赞/收藏目标 |

---

## 二、粉丝与关注（user-follows）

### 2.1 获取指定用户的粉丝列表

- **路径**: `GET /api/user-follows/followers`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `uid` | int | 否 | 目标用户 ID。不传则取当前登录用户 |
| `page` | int | 否 | 页码，默认 1 |
| `field` | string | 否 | 返回字段 |

- **请求示例**:
```
GET /api/user-follows/followers?uid=100&page=1
```

- **响应示例**:
```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "uid": 2,
        "follow_uid": 100,
        "status": 1,
        "create_time": 1700000000,
        "result": {
          "follow_user": {
            "id": 2,
            "nickname": "粉丝昵称",
            "avatar": "/uploads/avatar.jpg",
            "description": "个人简介"
          }
        }
      }
    ],
    "count": 42,
    "page": 1
  }
}
```

### 2.2 获取指定用户的关注列表

- **路径**: `GET /api/user-follows/following`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `uid` | int | 否 | 目标用户 ID。不传则取当前登录用户 |
| `page` | int | 否 | 页码，默认 1 |

- **请求示例**:
```
GET /api/user-follows/following?uid=100&page=1
```

- **响应结构**: 与 followers 相同，`result.follow_user` 返回的是被关注用户的信息。

### 2.3 批量查询粉丝数/关注数

- **路径**: `GET /api/user-follows/counts`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | `followers`（粉丝数）或 `following`（关注数） |
| `target_ids` | string | 是 | 用户 ID 列表，逗号分隔，如 `100` 或 `100,101,102` |

- **请求示例**:
```
# 查询单个用户的粉丝数
GET /api/user-follows/counts?target_type=followers&target_ids=100

# 查询单个用户的关注数
GET /api/user-follows/counts?target_type=following&target_ids=100

# 批量查询多个用户的粉丝数
GET /api/user-follows/counts?target_type=followers&target_ids=100,101,102
```

- **响应示例**:
```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "counts": {
      "100": 42,
      "101": 15,
      "102": 8
    }
  }
}
```

### 2.4 判断是否关注某用户

- **路径**: `GET /api/user-follows/is-following`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `follow_uid` | int | 是 | 被关注的用户 ID |

- **请求示例**:
```
GET /api/user-follows/is-following?follow_uid=100
```

- **响应示例**:
```json
{
  "code": 200,
  "data": {
    "is_following": true
  }
}
```

---

## 三、点赞数据（user-likes）

### 3.1 获取指定用户的点赞列表

- **路径**: `GET /api/user-likes/likes`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `uid` | int | 否 | 目标用户 ID。不传则取当前登录用户 |
| `target_type` | string | 否 | 筛选类型：`article` / `page` / `moments` / `comment` |
| `field` | string | 否 | 返回字段 |

- **请求示例**:
```
# 获取用户 100 点赞的所有内容
GET /api/user-likes/likes?uid=100

# 获取用户 100 点赞的文章
GET /api/user-likes/likes?uid=100&target_type=article
```

- **响应示例**:
```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "list": [
      {
        "id": 1,
        "uid": 100,
        "target_type": "article",
        "target_id": 10,
        "create_time": 1700000000,
        "result": {
          "author": {
            "id": 2,
            "nickname": "作者昵称",
            "avatar": "/uploads/avatar.jpg"
          }
        }
      }
    ],
    "count": 25
  }
}
```

### 3.2 查询用户总点赞数

- **路径**: `GET /api/user-likes/counts`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 查询用户总点赞数时传 `user_likes` |
| `target_ids` | string | 是 | 用户 ID 列表，逗号分隔 |

- **请求示例**:
```
# 查询用户 100 的总点赞数（作为被点赞者收到的总赞数）
GET /api/user-likes/counts?target_type=user_likes&target_ids=100

# 查询多个用户的点赞数
GET /api/user-likes/counts?target_type=user_likes&target_ids=100,101,102
```

- **响应示例**:
```json
{
  "code": 200,
  "data": {
    "counts": {
      "100": 128,
      "101": 56
    }
  }
}
```

> **注意**: `target_type=user_likes` 特殊处理，统计的是该用户**收到的**点赞总数（即作为被点赞目标的次数累加）。

### 3.3 查询指定内容的点赞数

- **路径**: `GET /api/user-likes/counts`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 内容类型：`article` / `page` / `moment` / `comment` / `user` |
| `target_ids` | string | 是 | 内容 ID 列表，逗号分隔 |

- **请求示例**:
```
# 查询文章 10,20,30 的点赞数
GET /api/user-likes/counts?target_type=article&target_ids=10,20,30
```

### 3.4 判断当前用户是否已点赞

- **路径**: `GET /api/user-likes/is-liked`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 目标类型 |
| `target_id` | int | 是 | 目标 ID |

- **请求示例**:
```
GET /api/user-likes/is-liked?target_type=article&target_id=10
```

- **响应示例**:
```json
{
  "code": 200,
  "data": {
    "is_liked": true,
    "count": 42
  }
}
```

---

## 四、收藏数据（user-collects）

### 4.1 获取用户的收藏列表

- **路径**: `GET /api/user-collects/collects`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 否 | 筛选类型：`article` / `page` / `moment` / `comment` |
| `field` | string | 否 | 返回字段 |

- **请求示例**:
```
# 获取当前用户的所有收藏
GET /api/user-collects/collects

# 获取当前用户收藏的文章
GET /api/user-collects/collects?target_type=article
```

- **响应示例**:
```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "list": [
      {
        "id": 1,
        "uid": 100,
        "target_type": "article",
        "target_id": 10,
        "create_time": 1700000000,
        "result": {
          "author": {
            "id": 2,
            "nickname": "作者昵称",
            "avatar": "/uploads/avatar.jpg"
          }
        }
      }
    ],
    "count": 15
  }
}
```

> **注意**: `collects` 接口**不支持**传 `uid` 参数查询其他用户的收藏，仅返回当前登录用户的收藏。

### 4.2 查询用户总收藏数

- **路径**: `GET /api/user-collects/counts`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 查询用户总收藏数时传 `user_collects` |
| `target_ids` | string | 是 | 用户 ID 列表，逗号分隔 |

- **请求示例**:
```
# 查询用户 100 的总收藏数
GET /api/user-collects/counts?target_type=user_collects&target_ids=100

# 查询多个用户的收藏数
GET /api/user-collects/counts?target_type=user_collects&target_ids=100,101
```

- **响应示例**:
```json
{
  "code": 200,
  "data": {
    "counts": {
      "100": 36,
      "101": 12
    }
  }
}
```

> **注意**: `target_type=user_collects` 特殊处理，统计的是该用户**收到的**收藏总数。

### 4.3 查询指定内容的收藏数

- **路径**: `GET /api/user-collects/counts`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 内容类型：`article` / `page` / `moment` / `comment` |
| `target_ids` | string | 是 | 内容 ID 列表，逗号分隔 |

- **请求示例**:
```
# 查询文章 10,20,30 的收藏数
GET /api/user-collects/counts?target_type=article&target_ids=10,20,30
```

### 4.4 判断当前用户是否已收藏

- **路径**: `GET /api/user-collects/is-collected`
- **参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `target_type` | string | 是 | 目标类型 |
| `target_id` | int | 是 | 目标 ID |

- **请求示例**:
```
GET /api/user-collects/is-collected?target_type=article&target_id=10
```

- **响应示例**:
```json
{
  "code": 200,
  "data": {
    "is_collected": true,
    "count": 15
  }
}
```

---

## 五、前端实战示例

### 场景：用户主页（author 页面）数据加载

```javascript
// 用户 ID - 来自路由参数
const authorId = route.params.uid

// 1. 并行获取所有统计数据
const [followersRes, followingRes, likesRes, collectsRes] = await Promise.all([
  // 粉丝数
  request.get('/api/user-follows/counts', {
    target_type: 'followers',
    target_ids: String(authorId)
  }),
  // 关注数
  request.get('/api/user-follows/counts', {
    target_type: 'following',
    target_ids: String(authorId)
  }),
  // 点赞数（用户收到的总赞数）
  request.get('/api/user-likes/counts', {
    target_type: 'user_likes',
    target_ids: String(authorId)
  }),
  // 收藏数（用户收到的总收藏数）
  request.get('/api/user-collects/counts', {
    target_type: 'user_collects',
    target_ids: String(authorId)
  })
])

// 提取数据
const followerCount = followersRes.data.counts[authorId] || 0
const followingCount = followingRes.data.counts[authorId] || 0
const likeCount = likesRes.data.counts[authorId] || 0
const collectCount = collectsRes.data.counts[authorId] || 0
```

### 场景：获取粉丝列表 + 判断关注状态

```javascript
// 并行获取粉丝列表和关注状态
const [fansRes, isFollowingRes] = await Promise.all([
  request.get('/api/user-follows/followers', {
    uid: authorId,
    page: 1
  }),
  // 需要登录态
  request.get('/api/user-follows/is-following', {
    follow_uid: authorId
  })
])

const fansList = fansRes.data.data      // 粉丝列表
const fansTotal = fansRes.data.count    // 粉丝总数
const isFollowing = isFollowingRes.data.is_following  // 当前用户是否已关注
```

### 场景：获取用户点赞/收藏的内容列表

```javascript
// 获取用户点赞的文章列表
const likedRes = await request.get('/api/user-likes/likes', {
  uid: authorId,
  target_type: 'article'
})
// likedRes.data.list - 点赞记录数组
// likedRes.data.count - 总数

// 获取当前用户的收藏列表（仅支持当前登录用户）
const collectedRes = await request.get('/api/user-collects/collects', {
  target_type: 'article'
})
// collectedRes.data.list - 收藏记录数组
```

### 场景：列表页批量获取互动数据

```javascript
// 文章列表渲染时，批量获取互动数据
const articleIds = articles.map(a => a.id).join(',')

const [likesRes, collectsRes] = await Promise.all([
  request.get('/api/user-likes/counts', {
    target_type: 'article',
    target_ids: articleIds
  }),
  request.get('/api/user-collects/counts', {
    target_type: 'article',
    target_ids: articleIds
  })
])

// likesRes.data.counts = { "1": 10, "2": 5, "3": 3 }
// collectsRes.data.counts = { "1": 8, "2": 3, "3": 1 }
```

---

## 六、接口对比表

### 按用户查询（需传 uid）

| API | 路径 | 支持 uid 参数 | 用途 |
|-----|------|--------------|------|
| followers | `/api/user-follows/followers?uid=xxx` | ✅ | 获取指定用户的粉丝列表 |
| following | `/api/user-follows/following?uid=xxx` | ✅ | 获取指定用户的关注列表 |
| likes | `/api/user-likes/likes?uid=xxx` | ✅ | 获取指定用户的点赞列表 |
| collects | `/api/user-collects/collects` | ❌ | 仅支持当前用户，不支持查其他用户 |

### 按用户 ID 查询数量（counts 接口）

| API | target_type | 说明 |
|-----|------------|------|
| user-follows/counts | `followers` | 查询指定用户的粉丝数 |
| user-follows/counts | `following` | 查询指定用户的关注数 |
| user-likes/counts | `user_likes` | 查询指定用户收到的总点赞数 |
| user-collects/counts | `user_collects` | 查询指定用户收到的总收藏数 |

### 按内容 ID 查询数量（counts 接口）

| API | target_type | 说明 |
|-----|------------|------|
| user-likes/counts | `article` / `page` / `moment` / `comment` / `user` | 查询内容的点赞数 |
| user-collects/counts | `article` / `page` / `moment` / `comment` | 查询内容的收藏数 |

---

## 七、权限说明

| 场景 | 权限要求 |
|------|---------|
| 查询自己的粉丝/关注/点赞/收藏列表 | 需登录 |
| 查询他人的粉丝/关注列表 | 需登录（支持 uid 参数） |
| 查询他人的点赞列表 | 需登录（支持 uid 参数） |
| 查询他人的收藏列表 | **不支持**，只能查自己的 |
| 查询粉丝数/关注数/点赞数/收藏数 | 无需登录（counts 接口为公开接口） |
| 判断是否关注/点赞/收藏 | 需登录（未登录时返回 false） |

---

## 八、数据模型关联

### 点赞/收藏记录的 result 字段

点赞和收藏记录的 `result` 字段会自动关联以下信息：

| 字段 | 说明 |
|------|------|
| `result.author` | 目标内容的作者信息（id、nickname、avatar、description） |

### 关注记录的 result 字段

关注记录的 `result` 字段会自动关联以下信息：

| 字段 | 说明 |
|------|------|
| `result.follow_user` | 关注关系中的用户信息 |

---

## 九、常见问题

### Q: 为什么 counts 接口返回空的 counts？
A: 检查 `target_ids` 参数是否正确。注意 `target_ids` 应该是**字符串**格式，如 `"100"` 或 `"100,101"`，而不是数字类型。

### Q: 为什么 collects 接口不能查询其他用户的收藏？
A: 出于隐私保护设计，收藏数据仅对本人可见。如果需要查询某个用户的收藏总数，可以使用 `/api/user-collects/counts?target_type=user_collects&target_ids=xxx`。

### Q: likes 接口的 target_type=user_likes 和其他值有什么区别？
A:
- `target_type=user_likes`：统计的是用户**作为被点赞对象**收到的总赞数（即别人点了你的赞）
- `target_type=article`/`page`/`moment` 等：统计的是**具体内容**的点赞数

### Q: 查询他人数据需要管理员权限吗？
A: 不需要。`followers`、`following`、`likes` 接口在传 `uid` 参数时，允许查询任意用户的公开互动数据。只有修改操作（关注、点赞、收藏等）需要登录且只能操作自己的数据。