# Auth Pages 接口文档

## 接口概述

`auth-pages` 控制器用于管理后台权限页面，支持页面的创建、查询、更新和删除操作。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | 支持15个基础接口：one、all、rand、count、sum、min、max、column、remove、delete、clear、restore、save、create、update |
| **特殊接口** | 无 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **201** | 创建成功 | 新增数据成功 |
| **204** | 无内容 | 查询无数据、无可操作数据 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 无操作权限 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取单个页面 [基础接口-获取指定]

- **路径**: `/api/auth-pages/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个权限页面

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 页面ID |
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
        "name": "仪表盘",
        "path": "/dashboard",
        "icon": "icon-home",
        "create_time": 1699900000
    }
}
```

#### 1.2 获取所有页面 [基础接口-获取全部]

- **路径**: `/api/auth-pages/all`
- **方法**: `GET`
- **描述**: 分页获取权限页面列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `limit` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
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
        "data": [...],
        "count": 10,
        "page": 1
    }
}
```

#### 1.3 随机获取页面 [基础接口-随机获取]

- **路径**: `/api/auth-pages/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的页面

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `limit` | int | 否 | 返回数量 |
| `except` | string | 否 | 排除的ID，逗号分隔 |
| `field` | string | 否 | 返回字段 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "好的！",
    "data": [...]
}
```

#### 1.4 查询数量 [基础接口-查询数量]

- **路径**: `/api/auth-pages/count`
- **方法**: `GET`
- **描述**: 查询页面数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `where` | json | 否 | 条件查询 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": 10
}
```

#### 1.5 求和 [基础接口-求和]

- **路径**: `/api/auth-pages/sum`
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
        "size": 100
    }
}
```

#### 1.6 最小值 [基础接口-最小值]

- **路径**: `/api/auth-pages/min`
- **方法**: `GET`
- **描述**: 获取指定字段最小值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |

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

- **路径**: `/api/auth-pages/max`
- **方法**: `GET`
- **描述**: 获取指定字段最大值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |

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

- **路径**: `/api/auth-pages/column`
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
    "data": ["仪表盘", "用户管理"]
}
```

---

### 2. POST 请求接口

#### 2.1 保存页面 [基础接口-保存数据]

- **路径**: `/api/auth-pages/save`
- **方法**: `POST`
- **描述**: 保存页面（新增或更新）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 页面ID，为空时新增 |
| `name` | string | **是** | 页面名称 |
| `path` | string | **是** | 页面路径 |
| `icon` | string | 否 | 图标名称 |
| `svg` | string | 否 | SVG图标内容 |
| `size` | int | 否 | 页面大小 |
| `remark` | string | 否 | 备注 |
| `json` | json | 否 | JSON数据 |
| `text` | string | 否 | 文本内容 |

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

#### 2.2 创建页面 [基础接口-添加数据]

- **路径**: `/api/auth-pages/create`
- **方法**: `POST`
- **描述**: 新增权限页面

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | **是** | 页面名称 |
| `path` | string | **是** | 页面路径 |
| `icon` | string | 否 | 图标名称 |
| `svg` | string | 否 | SVG图标内容 |
| `size` | int | 否 | 页面大小 |
| `remark` | string | 否 | 备注 |
| `json` | json | 否 | JSON数据 |
| `text` | string | 否 | 文本内容 |

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

---

### 3. PUT 请求接口

#### 3.1 更新页面 [基础接口-修改数据]

- **路径**: `/api/auth-pages/update`
- **方法**: `PUT`
- **描述**: 更新页面信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 页面ID |
| `name` | string | 否 | 页面名称 |
| `path` | string | 否 | 页面路径 |
| `icon` | string | 否 | 图标名称 |
| `svg` | string | 否 | SVG图标内容 |
| `size` | int | 否 | 页面大小 |
| `remark` | string | 否 | 备注 |
| `json` | json | 否 | JSON数据 |
| `text` | string | 否 | 文本内容 |

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

#### 3.2 恢复页面 [基础接口-恢复数据]

- **路径**: `/api/auth-pages/restore`
- **方法**: `PUT`
- **描述**: 从回收站恢复页面

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 页面ID列表，逗号分隔 |

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

---

### 4. DELETE 请求接口

#### 4.1 软删除页面 [基础接口-软删除]

- **路径**: `/api/auth-pages/remove`
- **方法**: `DELETE`
- **描述**: 将页面移入回收站

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 页面ID列表，逗号分隔 |

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

#### 4.2 彻底删除页面 [基础接口-彻底删除]

- **路径**: `/api/auth-pages/delete`
- **方法**: `DELETE`
- **描述**: 永久删除页面，不可恢复

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 页面ID列表，逗号分隔 |

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

#### 4.3 清空回收站 [基础接口-清空回收站]

- **路径**: `/api/auth-pages/clear`
- **方法**: `DELETE`
- **描述**: 清空所有已删除的页面

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

## 特殊说明

### 1. 字段处理
- `json` 和数组类型字段会自动序列化

### 2. 缓存策略
- 所有查询接口均支持缓存
- 数据修改后会自动清除相关缓存

### 3. 系统自带页面

系统启动时会自动初始化以下后台管理页面，无需手动创建：

| 序号 | 页面名称 | 图标 | 路径 |
| :--- | :--- | :--- | :--- |
| 1 | 撰写文章 | `article` | `/admin/article/write` |
| 2 | 文章管理 | `article` | `/admin/article` |
| 3 | 文章分类 | `group` | `/admin/article/group` |
| 4 | 用户管理 | `user` | `/admin/users` |
| 5 | 评论管理 | `comment` | `/admin/comment` |
| 6 | 公告管理 | `bell` | `/admin/placard` |
| 7 | 轮播管理 | `banner` | `/admin/banner` |
| 8 | 标签管理 | `tag` | `/admin/tags` |
| 9 | 等级管理 | `level` | `/admin/level` |
| 10 | 经验管理 | `level` | `/admin/exp` |
| 11 | 消息通知 | `bell` | `/admin/message` |
| 12 | 友链管理 | `link` | `/admin/links` |
| 13 | 系统配置 | `system` | `/admin/system` |
| 14 | 独立页面 | `open` | `/admin/pages` |
| 15 | 撰写独立页面 | `article` | `/admin/pages/write` |
| 16 | 友链分组 | `group` | `/admin/links/group` |
| 17 | 权限规则 | `rule` | `/admin/auth/rules` |
| 18 | 权限分组 | `group` | `/admin/auth/group` |
| 19 | 接口密钥 | `key` | `/admin/api/keys` |
| 20 | IP黑名单 | `qps` | `/admin/ip/black` |
| 21 | IP白名单 | `white` | `/admin/ip/white` |
| 22 | QPS预警 | `black` | `/admin/qps/warn` |
| 23 | 后台页面管理 | `open` | `/admin/auth/pages` |
| 24 | 动态管理 | `article` | `/admin/moments` |
| 25 | 附件管理 | `file` | `/admin/attachment` |

### 4. 页面初始化机制

- 页面通过 `path` 字段的哈希值进行去重
- 已存在的页面不会重复创建
- 初始化逻辑位于 [auth-pages.go](app/model/auth-pages.go) 的 `InitAuthPages()` 函数中