# Toml 配置管理接口文档

## 接口概述

`toml` 控制器用于管理系统配置文件，包括短信、缓存、加密、存储等配置的读取、更新和测试功能。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **GET 接口** | log、sms、cache、crypt、storage - 获取配置信息 |
| **POST 接口** | 测试各类服务连接 |
| **PUT 接口** | 更新各类配置，新增统一存储配置接口 `storage` |
| **DELETE 接口** | 暂不支持 |

### 新增接口说明

| 接口 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/api/toml/storage` | PUT | 统一更新存储配置，支持同时修改 default、local、cos、attachment 配置 |
| `/api/toml/storage-attachment` | PUT | 更新附件管理配置 |
| `/api/toml/sms-email-queue` | PUT | 更新邮件发件队列（分批 + 重试）参数，写入 `config/sms.toml` 的 `[email]` 段 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取短信配置

- **路径**: `/api/toml/sms`
- **方法**: `GET`
- **描述**: 获取短信服务配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：email、aliyun、aliyun_number_verify、tencent |

> `email` 分组（`name=email` 或整份返回里的 `data.email`）同时包含**发件队列参数**：
> `batch_size` / `batch_interval` / `retry_delay` / `max_attempts` / `send_timeout` / `verify_wait` / `queue_size`，
> 修改请用 `PUT /api/toml/sms-email-queue`（见 3.16）。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "email": {},
        "aliyun": {},
        "tencent": {},
        "drive": "email"
    }
}
```

#### 1.2 获取缓存配置

- **路径**: `/api/toml/cache`
- **方法**: `GET`
- **描述**: 获取缓存服务配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：redis、file、ram |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "redis": {},
        "file": {},
        "ram": {},
        "open": true,
        "default": "file"
    }
}
```

#### 1.3 获取加密配置

- **路径**: `/api/toml/crypt`
- **方法**: `GET`
- **描述**: 获取加密服务配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：jwt |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "jwt": {
            "key": "",
            "expire": 3600,
            "issuer": "",
            "subject": ""
        }
    }
}
```

#### 1.4 获取存储配置

- **路径**: `/api/toml/storage`
- **方法**: `GET`
- **描述**: 获取存储服务配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：local、cos、attachment |

> `local` / `cos` 分组里始终包含上传命名规则的生效值 `dir_rule`（目录规则）与
> `file_rule`（文件规则）：老配置文件里没有这两项时，接口会按默认值返回
> （见「特殊说明 → 5. 上传命名规则」）。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "local": {
            "domain": "",
            "path": "storage",
            "dir_rule": "{Y}-{m}/{d}",
            "file_rule": "{timestamp}{str-random-10}"
        },
        "cos": {},
        "attachment": {},
        "default": "local"
    }
}
```

#### 1.5 获取日志配置

- **路径**: `/api/toml/log`
- **方法**: `GET`
- **描述**: 获取日志服务配置

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {}
}
```

---

### 2. POST 请求接口（测试服务）

#### 2.1 测试邮箱服务

- **路径**: `/api/toml/test-sms-email`
- **方法**: `POST`
- **描述**: 测试邮箱发送功能

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `host` | string | **是** | SMTP 服务器地址 |
| `port` | int | **是** | SMTP 端口 |
| `account` | string | **是** | 邮箱账号 |
| `password` | string | **是** | 邮箱密码 |
| `sign_name` | string | **是** | 签名名称 |
| `nickname` | string | 否 | 发件人昵称 |
| `email` | string | **是** | 接收邮箱 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试邮件发送成功！",
    "data": null
}
```

#### 2.2 测试阿里云短信

- **路径**: `/api/toml/test-sms-aliyun`
- **方法**: `POST`
- **描述**: 测试阿里云短信发送功能

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `sign_name` | string | **是** | 签名名称 |
| `verify_code` | string | **是** | 验证码模板ID |
| `phone` | string | **是** | 测试手机号 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试短信发送成功！",
    "data": null
}
```

#### 2.3 测试阿里云号码验证

- **路径**: `/api/toml/test-sms-aliyun-number-verify`
- **方法**: `POST`
- **描述**: 测试阿里云号码验证服务

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `sign_name` | string | **是** | 签名名称 |
| `template_code` | string | **是** | 模板代码 |
| `phone` | string | **是** | 测试手机号 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试号码验证成功！",
    "data": {
        "verify_code": "666666",
        "message": "测试验证码发送成功"
    }
}
```

#### 2.4 测试腾讯云短信

- **路径**: `/api/toml/test-sms-tencent`
- **方法**: `POST`
- **描述**: 测试腾讯云短信发送功能

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `secret_id` | string | **是** | Secret ID |
| `secret_key` | string | **是** | Secret Key |
| `endpoint` | string | **是** | 端点地址 |
| `sms_sdk_app_id` | string | **是** | SDK App ID |
| `sign_name` | string | **是** | 签名名称 |
| `verify_code` | string | **是** | 验证码模板ID |
| `region` | string | **是** | 地域 |
| `phone` | string | **是** | 测试手机号 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试短信发送成功！",
    "data": null
}
```

#### 2.5 测试 Redis 连接

- **路径**: `/api/toml/test-redis`
- **方法**: `POST`
- **描述**: 测试 Redis 连接

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `host` | string | 否 | Redis 地址，默认 localhost |
| `port` | int | 否 | Redis 端口，默认 6379 |
| `database` | int | 否 | 数据库编号，默认 0 |
| `password` | string | 否 | Redis 密码 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试Redis连接成功！",
    "data": "PONG"
}
```

#### 2.6 测试 COS 连接

- **路径**: `/api/toml/test-cos`
- **方法**: `POST`
- **描述**: 测试腾讯云 COS 连接

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `secret_id` | string | **是** | Secret ID |
| `secret_key` | string | **是** | Secret Key |
| `app_id` | string | **是** | App ID |
| `bucket` | string | **是** | Bucket 名称 |
| `region` | string | **是** | 地域 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试COS连接成功！",
    "data": null
}
```

---

### 3. PUT 请求接口（更新配置）

#### 3.1 更新短信驱动配置

- **路径**: `/api/toml/sms-drive`
- **方法**: `PUT`
- **描述**: 更新短信驱动配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `email` | bool | 否 | 是否启用邮箱 |
| `sms` | bool | 否 | 是否启用短信 |
| `default` | string | 否 | 默认驱动，默认 email |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.2 更新邮箱配置

- **路径**: `/api/toml/sms-email`
- **方法**: `PUT`
- **描述**: 更新邮箱配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `host` | string | **是** | SMTP 服务器地址 |
| `port` | int | **是** | SMTP 端口 |
| `account` | string | **是** | 邮箱账号 |
| `password` | string | **是** | 邮箱密码 |
| `sign_name` | string | **是** | 签名名称 |
| `nickname` | string | 否 | 发件人昵称 |

> 发件队列参数（`batch_size` 等）与邮箱配置同属 `[email]` 段，但用
> `PUT /api/toml/sms-email-queue` 单独更新（见 3.18），本接口不会改动它们。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.3 更新阿里云短信配置

- **路径**: `/api/toml/sms-aliyun`
- **方法**: `PUT`
- **描述**: 更新阿里云短信配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `sign_name` | string | **是** | 签名名称 |
| `verify_code` | string | **是** | 验证码模板ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.4 更新阿里云号码验证配置

- **路径**: `/api/toml/sms-aliyun-number-verify`
- **方法**: `PUT`
- **描述**: 更新阿里云号码验证配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `sign_name` | string | **是** | 签名名称 |
| `template_code` | string | **是** | 模板代码 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.5 更新腾讯云短信配置

- **路径**: `/api/toml/sms-tencent`
- **方法**: `PUT`
- **描述**: 更新腾讯云短信配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `secret_id` | string | **是** | Secret ID |
| `secret_key` | string | **是** | Secret Key |
| `endpoint` | string | **是** | 端点地址 |
| `sms_sdk_app_id` | string | **是** | SDK App ID |
| `sign_name` | string | **是** | 签名名称 |
| `verify_code` | string | **是** | 验证码模板ID |
| `region` | string | **是** | 地域 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.6 更新 JWT 加密配置

- **路径**: `/api/toml/crypt-jwt`
- **方法**: `PUT`
- **描述**: 更新 JWT 加密配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `key` | string | **是** | 密钥 |
| `expire` | int | **是** | 过期时间（秒） |
| `issuer` | string | **是** | 发行者 |
| `subject` | string | **是** | 主题 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.7 更新缓存默认配置

- **路径**: `/api/toml/cache-default`
- **方法**: `PUT`
- **描述**: 更新缓存默认服务类型

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `value` | string | 否 | 默认缓存类型：redis、file、ram，默认 file |
| `open` | bool | 否 | 是否开启缓存，默认 false |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.8 更新 Redis 缓存配置

- **路径**: `/api/toml/cache-redis`
- **方法**: `PUT`
- **描述**: 更新 Redis 缓存配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `host` | string | 否 | Redis 地址，默认 localhost |
| `port` | int | 否 | Redis 端口，默认 6379 |
| `database` | int | 否 | 数据库编号，默认 0 |
| `password` | string | 否 | Redis 密码 |
| `prefix` | string | 否 | 键前缀，默认 inis: |
| `expire` | string | 否 | 过期时间，默认 2 * 60 * 60 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.9 更新文件缓存配置

- **路径**: `/api/toml/cache-file`
- **方法**: `PUT`
- **描述**: 更新文件缓存配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `path` | string | 否 | 缓存目录，默认 runtime/cache |
| `prefix` | string | 否 | 文件前缀，默认 inis_ |
| `expire` | string | 否 | 过期时间，默认 2 * 60 * 60 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.10 更新内存缓存配置

- **路径**: `/api/toml/cache-ram`
- **方法**: `PUT`
- **描述**: 更新内存缓存配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `expire` | string | 否 | 过期时间，默认 2 * 60 * 60 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.11 统一更新存储配置

- **路径**: `/api/toml/storage`
- **方法**: `PUT`
- **描述**: 统一更新存储配置，支持同时修改多个存储配置段

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `default` | string | 否 | 默认存储类型：local、cos |
| `local` | object | 否 | 本地存储配置 |
| `cos` | object | 否 | 腾讯云 COS 存储配置 |
| `attachment` | object | 否 | 附件配置 |

**local 配置项**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `domain` | string | 否 | 访问域名，留空 = 相对路径（/storage/xxx），也可填 CDN 域名 |
| `path` | string | 否 | public 下的存储子目录，留空则直接放 public 下 |
| `dir_rule` | string | 否 | 上传目录命名规则，默认 `{Y}-{m}/{d}`；填 `/` 表示不要子目录（占位符见特殊说明 5） |
| `file_rule` | string | 否 | 上传文件命名规则（不含扩展名），默认 `{timestamp}{str-random-10}` |

**cos 配置项**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `secret_id` | string | 否 | Secret ID |
| `secret_key` | string | 否 | Secret Key |
| `app_id` | string | 否 | App ID（桶名数字后缀） |
| `bucket` | string | 否 | Bucket 名称（裸桶名自动补 -app_id，也支持控制台全名） |
| `region` | string | 否 | 地域（留空按 ap-guangzhou） |
| `domain` | string | 否 | 自定义访问域名（CDN），留空用默认域名 |
| `path` | string | 否 | 对象键前缀，留空则键直接从目录命名规则开始 |
| `dir_rule` | string | 否 | 上传目录命名规则，默认 `{Y}-{m}/{d}`；填 `/` 表示不要子目录 |
| `file_rule` | string | 否 | 上传文件命名规则（不含扩展名），默认 `{timestamp}{str-random-10}` |

**attachment 配置项**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `allow_extensions` | string | 否 | 允许的文件扩展名，多个用逗号分隔 |
| `max_file_size` | int | 否 | 单个文件最大大小（KB） |
| `concurrent_limit` | int | 否 | 并发上传限制 |

**请求示例**:
```json
{
    "default": "cos",
    "local": {
        "domain": "",
        "path": "storage",
        "dir_rule": "{Y}-{m}/{d}",
        "file_rule": "{timestamp}{str-random-10}"
    },
    "cos": {
        "secret_id": "your-secret-id",
        "secret_key": "your-secret-key",
        "app_id": "1250000000",
        "bucket": "inis-cos",
        "region": "ap-guangzhou",
        "domain": "",
        "path": "inis",
        "dir_rule": "{Y}/{m}/{d}",
        "file_rule": "{Y}{m}{d}-{str-random-10}"
    },
    "attachment": {
        "max_file_size": 10240,
        "concurrent_limit": 5
    }
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.12 更新存储默认配置

- **路径**: `/api/toml/storage-default`
- **方法**: `PUT`
- **描述**: 更新存储默认服务类型

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `value` | string | **是** | 默认存储类型：local、cos |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.13 更新本地存储配置

- **路径**: `/api/toml/storage-local`
- **方法**: `PUT`
- **描述**: 更新本地存储配置

**请求参数**（all 可选，只提交要改的字段）:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `domain` | string | 否 | 访问域名（留空 = 相对路径 /storage/xxx） |
| `path` | string | 否 | public 下的存储子目录（留空则直接放 public 下） |
| `dir_rule` | string | 否 | 上传目录命名规则，默认 `{Y}-{m}/{d}`；填 `/` 表示不要子目录 |
| `file_rule` | string | 否 | 上传文件命名规则（不含扩展名，扩展名自动追加），默认 `{timestamp}{str-random-10}` |

> 上传落地路径 = `public` + `path` + `dir_rule` + `file_rule` + 扩展名，
> 例如 `path=storage`、`dir_rule={Y}-{m}/{d}`、`file_rule={timestamp}{str-random-10}`
> → `public/storage/2026-09/26/17588888881a2b3c4d5e.jpg`；
> 占位符清单见「特殊说明 → 5. 上传命名规则」，未知占位符返回 400。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.14 更新 COS 存储配置

- **路径**: `/api/toml/storage-cos`
- **方法**: `PUT`
- **描述**: 更新腾讯云 COS 存储配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `secret_id` | string | **是** | Secret ID |
| `secret_key` | string | **是** | Secret Key |
| `app_id` | string | **是** | App ID（桶名的数字后缀，如 1250000000） |
| `bucket` | string | **是** | Bucket 名称：可填 `inis-cos`，SDK 自动补成 `inis-cos-<app_id>`；也可直接填控制台复制的全名 `inis-cos-1250000000`（不会再重复拼 app_id） |
| `region` | string | **是** | 地域（留空按 `ap-guangzhou` 处理） |
| `domain` | string | 否 | 自定义访问域名（CDN）；留空时用默认域名 `https://<bucket>-<app_id>.cos.<region>.myqcloud.com` |
| `path` | string | 否 | 对象键前缀（留空则键直接从目录命名规则开始，不会带前导 `/`） |
| `dir_rule` | string | 否 | 上传目录命名规则，默认 `{Y}-{m}/{d}`；填 `/` 表示不要子目录 |
| `file_rule` | string | 否 | 上传文件命名规则（不含扩展名，扩展名自动追加），默认 `{timestamp}{str-random-10}` |

> 对象键 = `path` + `dir_rule` + `file_rule` + 扩展名，
> 例如 `path=inis`、`dir_rule={Y}/{m}/{d}`、`file_rule={Y}{m}{d}-{str-random-10}`
> → `inis/2026/09/26/20260926-1a2b3c4d5e.jpg`；
> 占位符清单见「特殊说明 → 5. 上传命名规则」，未知占位符返回 400。
>
> 上传时会对单个对象设置 `x-cos-acl: public-read`（桶不存在时同时以「公共读私有写」创建），
> 保证附件地址可直接访问；删除走 `DELETE Object` / `DELETE Multiple Objects`（单请求分批 500 个，
> 对象不存在（404）视为删除成功）。详见「附件接口文档」的 4.2 / 4.3。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.15 更新附件配置

- **路径**: `/api/toml/storage-attachment`
- **方法**: `PUT`
- **描述**: 更新附件管理配置

**请求参数**（只提交要改的字段，未提交的保持原值）:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `allow_extensions` | string | 否 | 允许的文件扩展名，多个用逗号分隔 |
| `max_file_size` | int | 否 | 单个文件最大大小（KB） |
| `concurrent_limit` | int | 否 | 并发上传限制（同时上传的文件数上限） |

> 附件上传目前只有这三项限制真正生效：扩展名白名单、单文件大小、并发数。
> 历史上还有 5 个「每时段上传上限」（`limit_per_minute` / `limit_per_hour` / `limit_per_day` /
> `limit_per_week` / `limit_per_month`），后端从未实现校验逻辑，已移除；
> 老配置里若残留这些键会被忽略（保存一次附件配置即被清理）。

**请求示例**:
```json
{
    "allow_extensions": "jpg,png,gif,webp,pdf",
    "max_file_size": 10240,
    "concurrent_limit": 5
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.16 更新发件队列配置

- **路径**: `/api/toml/sms-email-queue`
- **方法**: `PUT`
- **描述**: 更新邮件发件队列（分批限流 + 失败重试）参数，写入 `config/sms.toml` 的 `[email]` 段
- **同一份配置的另一种写法**: `PUT /api/toml/sms` + `{ "name": "email_queue", ...同下表字段 }`

**请求参数**（全部可选，只提交要改的字段，未提交的保持原值）：

| 参数名 | 类型 | 必填 | 取值范围 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `batch_size` | int | 否 | 1 ~ 1000 | 10 | 每个批次窗口最多发送多少封通知类邮件 |
| `batch_interval` | int | 否 | 1 ~ 86400 | 600 | 批次间隔（秒），一批发满后等待多久再发下一批（600 = 10 分钟） |
| `retry_delay` | int | 否 | 1 ~ 86400 | 60 | 发送失败后的重试延迟（秒） |
| `max_attempts` | int | 否 | 1 ~ 10 | 3 | 单封邮件最大尝试次数（含首次），超出后标记失败并丢弃 |
| `send_timeout` | int | 否 | 5 ~ 600 | 30 | 单封发送超时（秒），超时按失败处理，避免 SMTP 卡住整个队列 |
| `verify_wait` | int | 否 | 0 ~ 60 | 10 | 验证码 / 注册验证邮件等待首轮发送结果的超时（秒），0 = 不等待（纯异步） |
| `queue_size` | int | 否 | 10 ~ 1000000 | 1000 | 队列最大长度（仅限制通知类邮件，验证码始终受理） |

**行为说明**:

- 值超出取值范围（或非数字）返回 `400`，字段名会在 msg 里给出；
- 分批限制只作用于**通知类邮件**（评论 / 回复、消息通知、欢迎邮件、运营通知等），
  验证码与注册验证邮件属于高优先级任务，入队即发、不占用批次额度；
- 保存后由配置文件监听自动热更新（`initSMS` → `mailQueue.reload`），无需重启；
  队列在内存中，重启后未发送的任务会丢失（邮件通知按「尽力而为」处理）。

**请求示例**:
```json
{
    "batch_size": 30,
    "batch_interval": 300,
    "retry_delay": 120,
    "max_attempts": 5
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "发件队列配置已保存！",
    "data": null
}
```

**失败响应** (400):
```json
{
    "code": 400,
    "msg": "batch_size 取值范围 1 ~ 1000！",
    "data": null
}
```

---

### 4. INDEX 接口

- **路径**: `/api/toml/index`
- **方法**: `GET`
- **描述**: 配置管理首页（无实际功能）

**成功响应** (202):
```json
{
    "code": 202,
    "msg": "没什么用！",
    "data": null
}
```

---

## 特殊说明

### 1. 配置文件管理
- 配置修改后立即生效
- 配置文件存储在 `config/` 目录下

### 2. 模板变量
- 配置文件支持模板变量替换
- 使用 `${variable}` 格式引用其他配置

### 3. 测试服务
- 测试接口不修改实际配置
- 测试成功后需要手动保存配置

### 4. 安全性
- 敏感信息（如密钥）在 GET 响应中**脱敏返回**，形如 `前4位****后4位`，短密钥则全部隐藏为 `******`
- 涉及脱敏的字段：`password`、`access_key_secret`、`secret_key`、`access_key`、`access_key_id`、`secret_id`、JWT 的 `key`
- 更新配置（PUT）时，若密钥字段回传的是脱敏占位值（含 `****` 或全 `*`），后端将**保留原值不更新**，避免把脱敏值写坏配置
- 测试接口（POST）需重新输入真实密钥后方可测试
- 配置修改需要管理员权限

### 5. 上传命名规则（dir_rule / file_rule）

本地存储（`[local]`）与腾讯云 COS（`[cos]`）都支持自定义上传的**目录结构**与**文件名**，
字段为 `dir_rule`（目录命名规则）与 `file_rule`（文件命名规则），实现见 `app/facade/storage-rule.go`。

**最终位置 = `path` 前缀 / 目录规则 / 文件规则 + 扩展名**（扩展名按原文件自动追加，规则里不用写）。

| 占位符 | 说明 |
| :--- | :--- |
| `{Y}` | 年份（2026） |
| `{y}` | 两位数年份（26） |
| `{m}` | 月份（09） |
| `{d}` | 当月的第几号（26） |
| `{timestamp}` | 时间戳（秒） |
| `{uniqid}` | 唯一字符串（微秒时间戳 + 随机） |
| `{md5}` | 32 位随机 md5 |
| `{md5-16}` | 16 位随机 md5（32 位的中段） |
| `{str-random-16}` | 16 位随机字符串 |
| `{str-random-10}` | 10 位随机字符串 |
| `{filename}` | 文件原始名称（不含扩展名） |
| `{uid}` | 上传者用户 ID，游客为 0 |

**行为说明**：

- 默认值：`dir_rule = {Y}-{m}/{d}`、`file_rule = {timestamp}{str-random-10}`
  （与历史行为一致的年月日目录，文件名带秒级时间戳 + 随机串保证唯一）；
- 规则留空 = 用默认值；想让文件直接放在 `path` 前缀下（不要子目录），把 `dir_rule` 设为 `/`（或 `.`）；
- 同一次上传里目录与文件名共用一组取值：`{timestamp}` / `{md5}` / `{str-random-*}` 在两处一致，
  其中 `{md5}` 与 `{md5-16}` 同源、`{str-random-16}` 与 `{str-random-10}` 同源；
- 规则结果会清洗后再落盘：统一斜杠、剔除 `.` / `..` 片段（防目录穿越）、
  去掉 `<>:"|?*` 与控制字符、单段超过 100 字符截断，规则被清空时用毫秒时间戳兜底；
- 保存时校验占位符，出现未知占位符（如 `{mm}`）返回 `400`，避免生成带 `{xx}` 的目录；
- 只要规则里保留了 `{timestamp}` / `{uniqid}` / `{md5}` / `{str-random-*}` 之一即可保证文件名唯一；
  若只用 `{filename}` 这类不唯一的规则，同名文件会互相覆盖（同名同内容的秒传不受影响）。

**配置示例**（`PUT /api/toml/storage-local`）：

```json
{
    "path": "storage",
    "dir_rule": "{Y}/{m}/{d}",
    "file_rule": "{Y}{m}{d}-{str-random-10}"
}
```
→ `public/storage/2026/09/26/20260926-1a2b3c4d5e.jpg`

**失败响应** (400)：
```json
{
    "code": 400,
    "msg": "本地存储 的 file_rule 里 {mm} 不是可用占位符！",
    "data": null
}
```