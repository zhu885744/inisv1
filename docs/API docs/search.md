# Search 搜索服务 API

## 接口概述

搜索服务控制器，提供文章、页面、标签、用户、友链、动态的全局搜索功能。支持模糊搜索、关键词高亮、多字段联合搜索等高级特性。

### 状态码规范

| 状态码 | 含义 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 搜索成功 |
| 202 | 接受请求 | 无实际功能接口 |
| 400 | 请求参数错误 | 参数校验失败 |
| 405 | 方法不允许 | 调用了不存在的方法 |
| 500 | 服务器错误 | 服务器内部错误 |

### 接口列表

#### 1. 文章搜索 [article]

- **路径**: `/api/search/article`
- **方法**: `GET`
- **描述**: 搜索文章，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：title、content、abstract、tags，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "title": "<mark>搜索</mark>结果标题",
        "covers": "https://example.com/cover.jpg",
        "abstract": "文章<mark>摘要</mark>...",
        "tags": "标签1,标签2",
        "views": 123,
        "create_time": 1699920000,
        "audit": 1
      }
    ],
    "count": 15,
    "page": 1,
    "keyword": "关键词",
    "type": "article"
  }
}
```

**说明**:
- 搜索结果中的关键词会被 `<mark>` 标签包裹，实现高亮显示
- 默认搜索字段：title、content、abstract、tags
- 可通过 `fields=title,abstract` 指定只搜索标题和摘要

---

#### 2. 页面搜索 [pages]

- **路径**: `/api/search/pages`
- **方法**: `GET`
- **描述**: 搜索独立页面，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：title、content、key，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "key": "about",
        "title": "<mark>关于</mark>我们",
        "create_time": 1699920000,
        "views": 45,
        "audit": 1
      }
    ],
    "count": 5,
    "page": 1,
    "keyword": "关键词",
    "type": "pages"
  }
}
```

**说明**:
- 默认搜索字段：title、content、key
- 仅搜索审核通过的页面（audit = 1）

---

#### 3. 标签搜索 [tags]

- **路径**: `/api/search/tags`
- **方法**: `GET`
- **描述**: 搜索标签，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：name、description，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "name": "<mark>技术</mark>分享",
        "avatar": "https://example.com/tag.jpg",
        "description": "关于<mark>技术</mark>的分享"
      }
    ],
    "count": 3,
    "page": 1,
    "keyword": "关键词",
    "type": "tags"
  }
}
```

**说明**:
- 默认搜索字段：name、description

---

#### 4. 用户搜索 [users]

- **路径**: `/api/search/users`
- **方法**: `GET`
- **描述**: 搜索用户，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：nickname、email、description、title，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "nickname": "<mark>张三</mark>",
        "avatar": "https://example.com/avatar.jpg",
        "description": "<mark>技术</mark>爱好者",
        "title": "<mark>高级</mark>工程师",
        "email": "zh***g@example.com"
      }
    ],
    "count": 5,
    "page": 1,
    "keyword": "关键词",
    "type": "users"
  }
}
```

**说明**:
- 默认搜索字段：nickname、email、description、title
- 仅搜索正常状态的用户（status = 0）
- 邮箱地址会自动脱敏处理，保护用户隐私

---

#### 5. 友链搜索 [links]

- **路径**: `/api/search/links`
- **方法**: `GET`
- **描述**: 搜索友链，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：nickname、description、url，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "nickname": "<mark>技术</mark>博客",
        "avatar": "https://example.com/avatar.jpg",
        "description": "<mark>技术</mark>分享网站",
        "url": "https://example.com",
        "audit": 1
      }
    ],
    "count": 3,
    "page": 1,
    "keyword": "关键词",
    "type": "links"
  }
}
```

**说明**:
- 默认搜索字段：nickname、description、url
- 仅搜索审核通过的友链（audit = 1）

---

#### 6. 动态搜索 [moments]

- **路径**: `/api/search/moments`
- **方法**: `GET`
- **描述**: 搜索动态，支持模糊搜索、关键词高亮、多字段联合搜索

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |
| fields | string | 否 | 指定搜索字段，逗号分隔。可选值：content、location，默认搜索所有字段 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "data": [
      {
        "id": 1,
        "content": "<mark>今天</mark>天气不错",
        "images": "https://example.com/img1.jpg,https://example.com/img2.jpg",
        "location": "<mark>北京</mark>市",
        "create_time": 1699920000,
        "audit": 1,
        "status": 1
      }
    ],
    "count": 8,
    "page": 1,
    "keyword": "关键词",
    "type": "moments"
  }
}
```

**说明**:
- 默认搜索字段：content、location
- 仅搜索审核通过的动态（audit = 1）

---

#### 7. 全局搜索 [all]

- **路径**: `/api/search/all`
- **方法**: `GET`
- **描述**: 搜索所有类型（文章、页面、标签、用户、友链、动态），返回综合搜索结果

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| keyword | string | 是 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| limit | int | 否 | 每页数量，默认10 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "搜索成功！",
  "data": {
    "article": {
      "data": [
        {
          "id": 1,
          "title": "<mark>搜索</mark>结果",
          "abstract": "文章<mark>摘要</mark>",
          "tags": "标签",
          "views": 123,
          "create_time": 1699920000,
          "audit": 1
        }
      ],
      "count": 10,
      "page": 1,
      "type": "article"
    },
    "pages": {
      "data": [...],
      "count": 5,
      "page": 1,
      "type": "pages"
    },
    "tags": {
      "data": [...],
      "count": 3,
      "page": 1,
      "type": "tags"
    },
    "users": {
      "data": [...],
      "count": 2,
      "page": 1,
      "type": "users"
    },
    "links": {
      "data": [...],
      "count": 1,
      "page": 1,
      "type": "links"
    },
    "moments": {
      "data": [...],
      "count": 4,
      "page": 1,
      "type": "moments"
    },
    "total": 25,
    "keyword": "关键词",
    "type": "all"
  }
}
```

**说明**:
- 全局搜索会将 limit 平均分配给6种类型，每种类型最多返回 limit/6 条结果（最少1条）
- 所有类型的搜索结果都会进行关键词高亮处理
- total 字段为所有类型搜索结果的总数

---

## 特殊说明

### 1. 搜索范围与默认字段

| 搜索类型 | 默认搜索字段 |
| :--- | :--- |
| 文章搜索 | title、content、abstract、tags |
| 页面搜索 | title、content、key |
| 标签搜索 | name、description |
| 用户搜索 | nickname、email、description、title |
| 友链搜索 | nickname、description、url |
| 动态搜索 | content、location |

### 2. 多字段联合搜索

通过 `fields` 参数可以指定搜索字段，实现精准搜索：

```bash
# 只搜索文章标题
GET /api/search/article?keyword=技术&fields=title

# 搜索标题和摘要
GET /api/search/article?keyword=技术&fields=title,abstract

# 只搜索用户昵称
GET /api/search/users?keyword=张三&fields=nickname
```

### 3. 关键词高亮

搜索结果中的匹配关键词会自动被 `<mark>` 标签包裹，前端可通过 CSS 样式实现高亮显示：

```css
mark {
  background-color: #ffeb3b;
  padding: 0 2px;
}
```

高亮字段包括：title、content、abstract、description、name、nickname

### 4. 邮箱脱敏

用户搜索结果中的邮箱地址会自动脱敏处理：
- 用户名长度 <= 3：保留第一位，如 `z***@example.com`
- 用户名长度 > 3：保留前两位和最后一位，如 `zh***g@example.com`

### 5. 分页限制

全局搜索时每个类型最多返回 `limit/6` 条结果（最少1条）

### 6. 审核过滤

- 文章和页面只搜索审核通过的数据（audit = 1）
- 友链只搜索审核通过的数据（audit = 1）
- 动态只搜索审核通过的数据（audit = 1）
- 用户排除冻结用户（status = 0 表示正常）

### 7. 搜索算法

采用数据库级别的 `LIKE` 查询，支持模糊匹配：
- 搜索关键词会被自动添加 `%` 通配符（前后匹配）
- 多个字段之间使用 `OR` 连接
- 审核条件使用 `AND` 连接