# Attachment 接口文档

## 接口概述

`attachment` 控制器用于管理附件文件，支持文件的上传、下载、查询、删除等操作。支持多种存储驱动（本地、阿里云OSS、腾讯云COS、七牛云KODO），包含秒传去重、文件安全校验、图片尺寸检测等功能。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | 支持15个基础接口：one、all、rand、count、sum、min、max、column、remove、delete、clear、restore、save、create、update |
| **特殊接口** | 文件上传、批量上传、绑定业务类型、获取我的附件列表 |

> **接口规范说明**：`save` 接口为内部兼容接口，无ID时新增，有ID时更新。**推荐外部调用使用 `create`（新增）和 `update`（更新）**，语义更清晰。

### 存储驱动说明

| 驱动标识 | 说明 |
| :--- | :--- |
| **local** | 本地存储，默认写入 `public/uploads/` 目录 |
| **oss** | 阿里云OSS存储 |
| **cos** | 腾讯云COS存储 |
| **kodo** | 七牛云KODO存储 |

### 业务类型说明

| 类型标识 | 说明 |
| :--- | :--- |
| **article** | 文章配图 |
| **comment** | 评论图片 |
| **user_avatar** | 用户头像 |
| **其他** | 可自定义扩展其他业务类型 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **201** | 创建成功 | 新增数据成功 |
| **204** | 无内容 | 查询无数据、无可操作数据 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 无操作权限、私有文件无权限 |
| **207** | 部分成功 | 批量操作部分成功、部分失败 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

### 批量操作响应格式

涉及批量操作的接口（`bind`、`remove`、`delete`）采用部分成功模式，返回统一格式：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `success_ids` | array | 操作成功的ID列表 |
| `failed_ids` | array | 操作失败的ID列表 |
| `errors` | object | 失败原因映射，key为ID，value为错误信息 |

**状态码说明**：
- `200`：全部成功
- `207`：部分成功，部分失败
- `400`：全部失败

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取单个附件 [基础接口-获取指定]

- **路径**: `/api/attachment/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个附件

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 附件ID |
| `uuid` | string | 否 | 附件UUID（优先使用） |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx",
        "original_name": "test.jpg",
        "full_url": "https://cdn.example.com/storage/2024-01/01/xxx.jpg",
        "file_size": 102400,
        "mime_type": "image/jpeg",
        "file_ext": "jpg",
        "width": 1920,
        "height": 1080,
        "create_time": 1699900000
    }
}
```

**权限说明**:
- 公开附件：所有人可访问
- 私有附件：仅上传者和管理员可访问
- 禁用附件：仅管理员可访问

#### 1.2 获取所有附件 [基础接口-获取全部]

- **路径**: `/api/attachment/all`
- **方法**: `GET`
- **描述**: 分页获取附件列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `limit` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |
| `target_type` | string | 否 | 业务类型筛选 |
| `target_id` | int | 否 | 业务ID筛选 |
| `storage_driver` | string | 否 | 存储驱动筛选 |
| `file_ext` | string | 否 | 文件扩展名筛选 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

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

**权限说明**:
- 管理员：可查看所有附件
- 普通用户：仅查看自己的附件

#### 1.3 随机获取附件 [基础接口-随机获取]

- **路径**: `/api/attachment/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的附件

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

- **路径**: `/api/attachment/count`
- **方法**: `GET`
- **描述**: 查询附件数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `where` | json | 否 | 条件查询 |
| `target_type` | string | 否 | 业务类型筛选 |
| `target_id` | int | 否 | 业务ID筛选 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": 100
}
```

#### 1.5 求和 [基础接口-求和]

- **路径**: `/api/attachment/sum`
- **方法**: `GET`
- **描述**: 对指定字段求和（如文件大小总和）

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
        "file_size": 104857600
    }
}
```

#### 1.6 最小值 [基础接口-最小值]

- **路径**: `/api/attachment/min`
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
        "file_size": 1024
    }
}
```

#### 1.7 最大值 [基础接口-最大值]

- **路径**: `/api/attachment/max`
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
        "file_size": 52428800
    }
}
```

#### 1.8 查询列 [基础接口-查询列]

- **路径**: `/api/attachment/column`
- **方法**: `GET`
- **描述**: 获取指定字段列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |
| `or` | json | 否 | 或条件查询 |
| `like` | json | 否 | 模糊查询 |
| `not` | json | 否 | 排除条件 |
| `null` | string | 否 | 空值查询 |
| `notNull` | string | 否 | 非空查询 |
| `order` | string | 否 | 排序字段 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": ["test1.jpg", "test2.jpg"]
}
```

**安全说明**:
- 普通用户调用时必须携带筛选条件（where/or/like/not/null/notNull/ids），否则返回400错误
- 普通用户查询结果最多返回100条记录，防止全表查询导致内存溢出

#### 1.9 获取我的附件 [特殊接口]

- **路径**: `/api/attachment/list`
- **方法**: `GET`
- **描述**: 获取当前用户上传的附件列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `limit` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `field` | string | 否 | 返回字段，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "data": [...],
        "count": 10,
        "page": 1
    }
}
```

**权限说明**: 需要用户登录

---

### 2. POST 请求接口

#### 2.1 保存附件 [基础接口-保存数据]

- **路径**: `/api/attachment/save`
- **方法**: `POST`
- **描述**: 保存附件（新增或更新）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 附件ID，为空时新增 |
| `uuid` | string | 否 | 附件UUID |
| `original_name` | string | 否 | 原始文件名 |
| `target_type` | string | 否 | 业务类型 |
| `target_id` | int | 否 | 业务ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx"
    }
}
```

#### 2.2 创建附件 [基础接口-添加数据]

- **路径**: `/api/attachment/create`
- **方法**: `POST`
- **描述**: 新增附件记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `original_name` | string | 否 | 原始文件名 |
| `target_type` | string | 否 | 业务类型 |
| `target_id` | int | 否 | 业务ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx"
    }
}
```

**权限说明**: 需要用户登录

#### 2.3 上传附件 [特殊接口]

- **路径**: `/api/attachment/upload`
- **方法**: `POST`
- **描述**: 单文件上传，支持秒传去重

**Content-Type**: `multipart/form-data`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `file` | file | **是** | 上传的文件 |
| `target_type` | string | 否 | 业务类型 |
| `target_id` | int | 否 | 业务ID |


**成功响应** (200):
```json
{
    "code": 200,
    "msg": "上传成功！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx",
        "original_name": "test.jpg",
        "full_url": "https://cdn.example.com/storage/2024-01/01/xxx.jpg",
        "file_size": 102400,
        "mime_type": "image/jpeg",
        "file_ext": "jpg",
        "width": 1920,
        "height": 1080
    }
}
```

**秒传响应** (200):
```json
{
    "code": 200,
    "msg": "文件已存在（秒传）！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx",
        "original_name": "test.jpg",
        "full_url": "https://cdn.example.com/storage/2024-01/01/xxx.jpg",
        "file_size": 102400,
        "mime_type": "image/jpeg",
        "file_ext": "jpg"
    }
}
```

**权限说明**: 需要用户登录

#### 2.4 批量上传附件 [特殊接口]

- **路径**: `/api/attachment/batch`
- **方法**: `POST`
- **描述**: 批量上传文件，最多10个

**Content-Type**: `multipart/form-data`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `files` | file[] | **是** | 上传的文件数组，最多10个 |
| `target_type` | string | 否 | 业务类型 |
| `target_id` | int | 否 | 业务ID |


**成功响应** (200):
```json
{
    "code": 200,
    "msg": "批量上传完成！",
    "data": {
        "results": [
            {
                "id": 1,
                "uuid": "xxx",
                "original_name": "a.jpg",
                "full_url": "...",
                "file_size": 102400,
                "width": 1920,
                "height": 1080,
                "status": "success"
            },
            {
                "original_name": "b.jpg",
                "full_url": "...",
                "status": "exist"
            }
        ],
        "success": 2,
        "fail": 0
    }
}
```

**结果状态说明**:
- `success`：上传成功
- `exist`：文件已存在（秒传）
- 无状态：上传失败（未在results中）

**权限说明**: 需要用户登录

---

### 3. PUT 请求接口

#### 3.1 更新附件 [基础接口-修改数据]

- **路径**: `/api/attachment/update`
- **方法**: `PUT`
- **描述**: 更新附件信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 附件ID |
| `uuid` | string | 否 | 附件UUID（优先使用） |
| `original_name` | string | 否 | 原始文件名 |
| `target_type` | string | 否 | 业务类型 |
| `target_id` | int | 否 | 业务ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "更新成功！",
    "data": {
        "id": 1,
        "uuid": "xxx-xxx-xxx"
    }
}
```

**权限说明**:
- 管理员可以更新所有附件
- 普通用户只能更新自己的附件

#### 3.2 恢复附件 [基础接口-恢复数据]

- **路径**: `/api/attachment/restore`
- **方法**: `PUT`
- **描述**: 从回收站恢复附件，恢复后附件可正常访问

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 附件ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "恢复成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [],
        "errors": {}
    }
}
```

**部分成功响应** (207):
```json
{
    "code": 207,
    "msg": "部分恢复成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [3],
        "errors": {
            "3": "无权恢复该附件或附件不存在"
        }
    }
}
```

**权限说明**:
- 管理员可以恢复所有附件
- 普通用户只能恢复自己的附件

#### 3.3 绑定业务类型 [特殊接口]

- **路径**: `/api/attachment/bind`
- **方法**: `PUT`
- **描述**: 将附件绑定到指定业务类型

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | 否 | 附件ID列表，逗号分隔 |
| `uuids` | string | 否 | 附件UUID列表，逗号分隔（与ids二选一） |
| `target_type` | string | **是** | 业务类型 |
| `target_id` | int | **是** | 业务ID |
| `is_cover` | bool | 否 | 是否覆盖已有绑定，默认false。若附件已绑定其他业务，且is_cover=false，则绑定失败 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "绑定成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [],
        "errors": {}
    }
}
```

**部分成功响应** (207):
```json
{
    "code": 207,
    "msg": "部分绑定成功！",
    "data": {
        "success_ids": [1],
        "failed_ids": [2],
        "errors": {
            "2": "附件已绑定到其他业务(article:100)，请设置 is_cover=true 允许覆盖"
        }
    }
}
```

**权限说明**:
- 管理员可以绑定所有附件
- 普通用户只能绑定自己的附件
- 若附件已绑定其他业务，默认报错，需设置 is_cover=true 允许覆盖

---

### 4. DELETE 请求接口

#### 4.1 软删除附件 [基础接口-软删除]

- **路径**: `/api/attachment/remove`
- **方法**: `DELETE`
- **描述**: 将附件移入回收站，仅标记数据库记录，不会删除存储桶文件

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 附件ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [],
        "errors": {}
    }
}
```

**部分成功响应** (207):
```json
{
    "code": 207,
    "msg": "部分删除成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [3],
        "errors": {
            "3": "无权删除该附件"
        }
    }
}
```

**权限说明**:
- 管理员可以删除所有附件
- 普通用户只能删除自己的附件

**特殊说明**: 软删除仅标记数据库记录，**不会删除存储桶文件**，支持通过`/api/attachment/restore`恢复。恢复后附件可正常访问。批量删除时采用部分成功模式，有权限的删除成功，无权限的返回失败信息。

#### 4.2 彻底删除附件 [基础接口-彻底删除]

- **路径**: `/api/attachment/delete`
- **方法**: `DELETE`
- **描述**: 永久删除附件，不可恢复，同时从存储删除物理文件

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | string | **是** | 附件ID列表，逗号分隔 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [],
        "errors": {}
    }
}
```

**部分成功响应** (207):
```json
{
    "code": 207,
    "msg": "部分删除成功！",
    "data": {
        "success_ids": [1, 2],
        "failed_ids": [3],
        "errors": {
            "3": "附件不存在"
        }
    }
}
```

**权限说明**: 仅管理员可操作

**特殊说明**: 删除时会根据附件记录的`storage_driver`字段自动选择对应的存储驱动（local/oss/cos/kodo），**按存储驱动分组后批量删除**物理文件，提高删除效率。COS和OSS支持单次请求批量删除多个对象（最多1000个），本地存储和七牛云KODO采用遍历删除。删除操作异步执行，不会阻塞请求响应。

#### 4.3 清空回收站 [基础接口-清空回收站]

- **路径**: `/api/attachment/clear`
- **方法**: `DELETE`
- **描述**: 清空回收站中的附件，同时从存储删除物理文件

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

**权限说明**: 仅管理员可操作

**特殊说明**: 清空时会根据附件记录的`storage_driver`字段自动选择对应的存储驱动（local/oss/cos/kodo），**按存储驱动分组后批量删除**物理文件，提高删除效率。COS和OSS支持单次请求批量删除多个对象（最多1000个），本地存储和七牛云KODO采用遍历删除。删除操作异步执行，不会阻塞请求响应。

---

## 特殊说明

### 1. 秒传去重机制
- 上传前计算文件MD5值
- 数据库中已存在相同MD5的文件时，直接返回已有记录
- 避免重复存储相同文件，节省存储空间

### 2. 文件安全校验
- **扩展名白名单**：仅允许配置的文件类型
- **文件内容校验**：根据扩展名验证文件头（Magic Bytes），防止改后缀上传恶意文件
  - 图片格式（jpg/png/gif/webp/bmp）：验证文件头标识
  - PDF：验证 `%PDF` 头部
  - Office 旧格式（doc/xls/ppt）：验证 OLE2 复合文档头部
  - Office 新格式（docx/xlsx/pptx）：验证 ZIP 头部标识
  - 压缩格式（zip/rar/7z）：验证文件头标识
  - SVG：验证 `<svg` 或 `<?xml` 头部，防止存储型 XSS 攻击
- **文件名安全**：使用 `filepath.Clean` + 正则过滤非法字符，防止路径遍历攻击
  - 过滤字符：`<>`:`/\|?*` 和控制字符（0x00-0x1F）
  - 移除 `../` 类路径穿越
- **SVG XSS 防护**：上传 SVG 时自动清洗 `<script>`、`<foreignObject>` 标签、`on*` 事件属性、`CDATA` 块、`javascript:`、`vbscript:` 协议
- **文件哈希算法**：使用 SHA-256 计算文件哈希，防止碰撞攻击

### 3. 并发上传限制
- 使用 Redis 计数器实现分布式并发控制
- Redis 不可用时拒绝上传（防止分布式环境下限流失效）
- Redis key 使用命名空间前缀 `inis:attachment:`
- Redis key 仅首次创建时设置过期，防止持续上传场景下永不过期
- 服务异常崩溃时自动归零（过期时间 5 分钟）

### 4. 查询安全
- **column 接口**：普通用户必须携带筛选条件（where/or/like/not/null/notNull/ids），最多返回100条记录
- **rand 接口**：使用 `ORDER BY RAND()` 避免全表 ID 查询，限制最多返回100条
- **聚合查询**（sum/min/max）：仅支持管理员操作，防止全表聚合拖垮数据库

### 5. 缓存策略
- 缓存设置 10 分钟过期时间，避免缓存永久堆积
- 异步删除缓存时添加 panic 捕获，防止协程崩溃

### 6. 秒传安全
- 秒传时校验文件上传者权限，非上传者且非管理员仅返回"文件已存在"提示，不暴露文件详情

### 7. 图片自动处理
- 自动检测图片宽高并记录到数据库
- 支持后续扩展：图片压缩、裁剪、生成缩略图

### 8. 存储驱动切换
- 通过配置文件 `config/storage.toml` 配置默认存储驱动（default字段）
- 支持本地、阿里云OSS、腾讯云COS、七牛云KODO四种驱动
- **上传**：使用default配置的驱动存储新文件
- **删除**：根据附件记录的`storage_driver`字段自动选择对应的驱动删除物理文件
- **驱动独立性**：即使切换了默认驱动，历史附件仍能被正确管理（每个附件记录了自己的存储驱动）

### 9. 附件配置选项

所有配置在 `config/storage.toml` 的 `[attachment]` 段中设置：

| 配置项 | 类型 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- |
| `allow_extensions` | string | `jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md` | 允许上传的文件类型，多个用逗号分隔（小写） |
| `max_file_size` | int | `51200` | 单个文件最大大小（KB），默认50MB |
| `concurrent_limit` | int | `5` | 并发上传限制（同时上传的最大文件数），使用互斥锁保证并发安全 |

**配置示例**：
```toml
[attachment]
allow_extensions = "jpg,png,gif,webp,pdf,doc,docx,xlsx"
max_file_size = 51200
concurrent_limit = 5
```

**配置说明**：
- `concurrent_limit = 5`：同时最多5个文件上传

**配置接口**：

附件配置可通过以下接口进行动态修改，无需重启服务：

| 接口 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/api/toml/storage?name=attachment` | GET | 获取附件配置 |
| `/api/toml/storage-attachment` | PUT | 修改附件配置 |

**修改配置请求示例**：
```json
{
    "allow_extensions": "jpg,png,gif,webp",
    "max_file_size": 10240,
    "concurrent_limit": 5
}
```

**注意**：修改配置后会自动重新加载，立即生效。

### 6. 权限控制
- 普通用户：只能查看、上传、删除自己的附件
- 管理员：可管理所有附件，包括物理删除、清空回收站
- 私有文件：仅上传者和管理员可访问

### 7. 业务绑定
- 通过`target_type`和`target_id`实现与其他模块的关联
- 支持动态扩展业务类型

### 8. 缓存策略
- 所有查询接口均支持缓存
- 数据修改后会自动清除相关缓存

### 9. 错误处理
- 删除存储文件失败不会影响数据库操作
- 失败信息会记录到日志中，便于排查

### 10. 上传限制说明
- **并发上传限制**：同时进行的上传请求数不能超过 `concurrent_limit` 配置值，使用互斥锁保证并发安全
- **批量上传**：批量上传时，会先检查总数是否超过限制和并发限制，再逐个处理
- **秒传文件**：秒传文件（已存在相同MD5）不计入上传数量限制，但会占用并发上传计数
- **错误提示**：超过限制时会返回对应的错误信息，如"并发上传数量已达上限（5个）！"