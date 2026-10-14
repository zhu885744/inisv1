# Topics 话题（Hashtag）接口文档

## 接口概述

`topics` 控制器提供面向用户的话题能力：话题榜、话题详情、话题内内容聚合、话题关注、
编辑器联想候选，以及话题主持人的内容治理（置顶 / 移出话题 / 删除内容）。

话题与**标签是同一张表**（`tags`）——管理侧的增删改仍在 `tags` 控制器下
（见 `docs/API docs/tags.md`），本控制器只放面向用户的读写。
完整设计说明见 `docs/topics.md`。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **读接口** | `board`、`detail`、`content` —— 公开可访问（`type=common`） |
| **个人读接口** | `candidates`、`mine` —— 需登录（`type=login`），只返回本人数据 |
| **写接口** | `follow`、`unfollow` —— 需登录 |
| **主持人接口** | `moderator-update`、`pin`、`unpin`、`remove-content`、`delete-content` —— 需登录，且要求是话题主持人本人或系统管理员 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **204** | 无内容 | 话题不存在 |
| **400** | 请求错误 | 参数校验失败、内容不在该话题里、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 不是话题主持人（`moderator-update` / `pin` / `unpin` / `remove-content` / `delete-content`） |
| **405** | 方法不允许 | 方法名错误或请求方法不支持 |

---

## 接口列表

### 1. GET 请求接口

#### 1.1 话题榜 [读接口]

- **路径**: `/api/topics/board`
- **方法**: `GET`
- **描述**: 按热度排序的话题列表。热度 = 文章数 × 3 + 动态数 × 2 + 关注数 × 2；结果缓存 60 秒

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `keyword` | string | 否 | 按话题名模糊过滤 |
| `recommend` | bool | 否 | 传 `1` 只返回精选话题 |
| `limit` | int | 否 | 返回条数，默认 20，最大 100 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": [
        {
            "id": 12,
            "name": "旅行",
            "description": "在路上的故事",
            "avatar": "https://example.com/topic.png",
            "moderator_uid": 2,
            "recommend": 1,
            "create_time": 1699900000,
            "article_count": 6,
            "moment_count": 18,
            "follow_count": 42,
            "heat": 6,
            "is_following": false
        }
    ]
}
```

> `heat` 由后端按权重计算后排序；`is_following` 表示当前登录用户是否已关注（未登录恒为 `false`）。

#### 1.2 话题详情 [读接口]

- **路径**: `/api/topics/detail`
- **方法**: `GET`
- **描述**: 获取话题信息、内容量、关注数与主持人。`id` 与 `key` 二选一（与前台 `/topic/:key` 一致）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 话题ID（与 `key` 二选一） |
| `key` | string | 否 | 话题名或话题ID（支持中文名访问；重名时取 id 最小者） |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "id": 12,
        "name": "旅行",
        "description": "在路上的故事",
        "avatar": "",
        "moderator_uid": 2,
        "recommend": 0,
        "create_time": 1699900000,
        "article_count": 6,
        "moment_count": 18,
        "follow_count": 42,
        "heat": 154,
        "moderator": {
            "id": 2,
            "nickname": "张三",
            "avatar": "",
            "description": "",
            "title": ""
        },
        "is_following": false
    }
}
```

> `moderator` 为 `null` 表示未设置主持人；`is_following` 需登录才有意义（未登录恒为 `false`）。

**无内容** (204):
```json
{ "code": 204, "msg": "话题不存在！", "data": null }
```

#### 1.3 话题内内容聚合 [读接口]

- **路径**: `/api/topics/content`
- **方法**: `GET`
- **描述**: 分页获取某话题下的动态或文章。动态只返回已发布且审核通过的（`audit=1`、`status=1`），文章只返回审核通过的（`audit=1`）；话题内置顶的内容排在最前

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 话题ID（与 `key` 二选一） |
| `key` | string | 否 | 话题名或话题ID |
| `type` | string | 否 | `moments`（默认）/ `article` |
| `sort` | string | 否 | `new`（默认，按创建时间）/ `hot`（按浏览量）/ `top`（仅动态：按内容自身置顶） |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页条数，默认取站点分页配置 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "data": [
            {
                "id": 66,
                "uid": 1,
                "content": "这次去了 #[旅行](/topic/12) 的路线",
                "tags": "|12|",
                "create_time": 1699900000,
                "is_pinned": true,
                "result": { "author": {}, "comment": {} }
            }
        ],
        "count": 24,
        "page": 1
    }
}
```

> `is_pinned` 表示该内容在本话题内是否被置顶；置顶项排在当前页最前（排序在 SQL 层完成，翻页不会重复或漏项）。

**无内容** (204):
```json
{ "code": 204, "msg": "无数据！", "data": { "data": [], "count": 0, "page": 0 } }
```

#### 1.4 话题联想候选 [个人读接口]

- **路径**: `/api/topics/candidates`
- **方法**: `GET`
- **权限**: `type=login`
- **描述**: 供编辑器在输入 `#` 时展示候选（前端一次取回后在本地过滤）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `keyword` | string | 否 | 按话题名模糊过滤 |
| `limit` | int | 否 | 返回条数，默认 50，最大 500 |

**成功响应** (200): 结构同 `board`（数组）。

#### 1.5 我关注的话题 [个人读接口]

- **路径**: `/api/topics/mine`
- **方法**: `GET`
- **权限**: `type=login`
- **描述**: 当前用户关注的话题，最近关注的在前

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": [
        {
            "id": 12,
            "name": "旅行",
            "description": "",
            "avatar": "",
            "moderator_uid": 0,
            "recommend": 0,
            "article_count": 6,
            "moment_count": 18,
            "follow_count": 42,
            "heat": 154,
            "is_following": true
        }
    ]
}
```

---

### 2. POST 请求接口

#### 2.1 关注话题 [写接口]

- **路径**: `/api/topics/follow`
- **方法**: `POST`（也支持 `PUT`：`/api/topics/...` 见 3.1）
- **权限**: `type=login`
- **描述**: 关注话题（幂等，重复关注不会报错也不会产生第二条记录）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 话题ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "关注成功！",
    "data": { "id": 12, "is_following": true, "follow_count": 43 }
}
```

#### 2.2 取消关注话题 [写接口]

- **路径**: `/api/topics/unfollow`
- **方法**: `POST`
- **权限**: `type=login`
- **描述**: 取消关注（幂等：本就没关注也返回成功）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 话题ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "已取消关注！",
    "data": { "id": 12, "is_following": false, "follow_count": 42 }
}
```

#### 2.3 主持人编辑话题资料 [主持人接口]

- **路径**: `/api/topics/moderator-update`
- **方法**: `POST`
- **权限**: `type=login` + 话题主持人本人或系统管理员
- **描述**: 编辑话题简介与封面。**话题名不可改**（正文里的话题链接按名称写入）；
  精选、主持人的变更仍在后台「标签管理」里操作

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 话题ID |
| `description` | string | 否 | 话题简介（做 XSS 检测与转义） |
| `avatar` | string | 否 | 封面图片地址（做 XSS 检测，链接不做转义） |

**成功响应** (200):
```json
{ "code": 200, "msg": "更新成功！", "data": { "id": 12 } }
```

#### 2.4 话题内置顶内容 [主持人接口]

- **路径**: `/api/topics/pin`
- **方法**: `POST`
- **权限**: `type=login` + 话题主持人本人或系统管理员
- **描述**: 把一条内容在本话题内置顶（幂等）。与内容自身的全局置顶是两回事

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 话题ID |
| `type` | string | 否 | 内容类型：`moments`（默认）/ `article` |
| `target_id` | int | **是** | 内容ID |

**成功响应** (200):
```json
{ "code": 200, "msg": "操作成功！", "data": { "target_id": 66, "is_pinned": true } }
```

**内容不在该话题里** (400):
```json
{ "code": 400, "msg": "该内容不在这个话题里！", "data": null }
```

#### 2.5 取消话题内置顶 [主持人接口]

- **路径**: `/api/topics/unpin`
- **方法**: `POST`（也支持 `PUT`）
- **权限**: `type=login` + 话题主持人本人或系统管理员

**请求参数**: 同 `pin`。

**成功响应** (200):
```json
{ "code": 200, "msg": "操作成功！", "data": { "target_id": 66, "is_pinned": false } }
```

#### 2.6 把内容移出话题 [主持人接口]

- **路径**: `/api/topics/remove-content`
- **方法**: `POST`
- **权限**: `type=login` + 话题主持人本人或系统管理员
- **描述**: 只解除内容与话题的关联（摘掉 `tags` 字段里的该话题 id），**不删除内容**；
  若该内容在本话题内被置顶，会一并清掉置顶

**请求参数**: 同 `pin`。

**成功响应** (200):
```json
{ "code": 200, "msg": "已移出话题！", "data": { "target_id": 66 } }
```

#### 2.7 删除话题内内容 [主持人接口]

- **路径**: `/api/topics/delete-content`
- **方法**: `POST`
- **权限**: `type=login` + 话题主持人本人或系统管理员
- **描述**: 删除话题内的内容（**软删除**，可到后台回收站恢复），并记录一条告警日志便于追溯。
  只有「确实关联着该话题」的内容才会被删除

**请求参数**: 同 `pin`。

**成功响应** (200):
```json
{ "code": 200, "msg": "已删除！", "data": { "target_id": 66 } }
```

---

### 3. PUT 请求接口

#### 3.1 取消关注话题

- **路径**: `/api/topics/unfollow`
- **方法**: `PUT`
- **权限**: `type=login`
- **描述**: 与 `POST /api/topics/unfollow` 等价（与 `user-follows` 的约定保持一致）

**请求参数**: `id`（话题ID，必填）。

#### 3.2 取消话题内置顶

- **路径**: `/api/topics/unpin`
- **方法**: `PUT`
- **权限**: `type=login` + 话题主持人本人或系统管理员

**请求参数**: 同 `POST /api/topics/unpin`。

---

### 4. DELETE 请求接口

本控制器**不开放删除**：话题的删除 / 回收站仍在后台「标签管理」（`/api/tags/remove` 等）。

`DELETE /api/topics/*` 固定返回：

```json
{ "code": 403, "msg": "话题删除请到后台「标签管理」操作！", "data": null }
```

---

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| `docs/topics.md` | 话题功能设计：存储格式、匹配规则、关注 / 置顶 / 主持人、热度口径、前端接入 |
| `docs/API docs/tags.md` | 标签（话题）的后台管理接口，含 `moderator_uid` / `recommend` 扩展字段 |
| `docs/API docs/moments.md` | 动态接口（正文里的 `#话题#` 会被后端自动规范化，并写入 `tags` 字段） |
| `docs/mention.md` | @提及（与话题共用候选浮层与渲染方式，互为参照） |
