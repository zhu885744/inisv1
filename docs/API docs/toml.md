# 配置管理 接口文档

## 接口概述

`toml` 控制器用于管理系统配置文件，包括短信、缓存、加密、存储等配置的读取、更新和测试功能。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **GET 接口** | log、sms、cache、crypt、storage、notification、sensitive、sensitive-stats - 获取配置信息 |
| **POST 接口** | 测试各类服务连接；敏感词库测试与回滚 |
| **PUT 接口** | 更新各类配置，新增统一存储配置接口 `storage`、敏感词库 `sensitive` |
| **DELETE 接口** | 暂不支持 |

### 新增接口说明

| 接口 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/api/toml/storage` | PUT | 统一更新存储配置，支持同时修改 default、local、cos、attachment 配置 |
| `/api/toml/storage-attachment` | PUT | 更新附件管理配置 |
| `/api/toml/sms-email-queue` | PUT | 更新邮件发件队列（分批 + 重试）参数，写入 `config/sms.toml` 的 `[email]` 段 |
| `/api/toml/sensitive` | GET / PUT | 读取 / 保存敏感词库（独立 `config/sensitive.toml`）：PUT 支持 TOML 原文与可视化表单两种载荷 |
| `/api/toml/sensitive-test` | POST | 用当前词库试跑一条文本，不写命中审计 |
| `/api/toml/sensitive-rollback` | POST | 回滚到上一次保存前的词库（单步） |
| `/api/toml/sensitive-stats` | GET | 最近 7 天命中统计（按场景 + 动作聚合） |

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

### 1. 获取短信配置 [sms]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/sms`
- **说明**: 获取短信服务配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：email、aliyun、aliyun_number_verify、tencent |

> `email` 分组（`name=email` 或整份返回里的 `data.email`）同时包含**发件队列参数**：
> `batch_size` / `batch_interval` / `retry_delay` / `max_attempts` / `send_timeout` / `verify_wait` / `queue_size`，
> 修改请用 `PUT /api/toml/sms-email-queue`（见 28）。

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

### 2. 获取缓存配置 [cache]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/cache`
- **说明**: 获取缓存服务配置

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

### 3. 获取加密配置（登录会话） [crypt]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/crypt`
- **说明**: 获取登录会话配置（JWT 已移除，crypt.toml 只剩 `[session] expire`）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 指定配置项：session（旧前端传 jwt 也返回同一份数据） |

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "session": {
            "expire": "15 * 24 * 60 * 60"
        }
    }
}
```

> 老站点的 `crypt.toml` 里还是 `[jwt]` 段时，后端会把 `expire` 映射到 `session` 返回，管理员保存一次配置即完成迁移。

### 4. 获取存储配置 [storage]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/storage`
- **说明**: 获取存储服务配置

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

### 5. 获取日志配置 [log]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/log`
- **说明**: 获取日志服务配置

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {}
}
```

### 6. 获取通知配置 [notification]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/notification`
- **说明**: 获取站内通知配置（`config/app.toml` 的 `[notification] retention_days`）

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "retention_days": 30
    }
}
```

> `retention_days` 为站内通知保留天数，读取时缺省 30（配置文件里没有该键时按默认值返回）。

---

### 7. 测试邮箱服务 [test-sms-email]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-sms-email`
- **说明**: 测试邮箱发送功能

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

### 8. 测试阿里云短信 [test-sms-aliyun]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-sms-aliyun`
- **说明**: 测试阿里云短信发送功能

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

### 9. 测试阿里云号码验证 [test-sms-aliyun-number-verify]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-sms-aliyun-number-verify`
- **说明**: 测试阿里云号码验证服务

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

### 10. 测试腾讯云短信 [test-sms-tencent]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-sms-tencent`
- **说明**: 测试腾讯云短信发送功能

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

### 11. 测试 Redis 连接 [test-redis]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-redis`
- **说明**: 测试 Redis 连接

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

### 12. 测试 COS 连接 [test-cos]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/test-cos`
- **说明**: 测试腾讯云 COS 连接

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

### 13. 更新短信驱动配置 [sms-drive]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-drive`
- **说明**: 更新短信驱动配置

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

### 14. 更新邮箱配置 [sms-email]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-email`
- **说明**: 更新邮箱配置

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
> `PUT /api/toml/sms-email-queue` 单独更新（见 28），本接口不会改动它们。

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

### 15. 更新阿里云短信配置 [sms-aliyun]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-aliyun`
- **说明**: 更新阿里云短信配置

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

### 16. 更新阿里云号码验证配置 [sms-aliyun-number-verify]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-aliyun-number-verify`
- **说明**: 更新阿里云号码验证配置

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

### 17. 更新腾讯云短信配置 [sms-tencent]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-tencent`
- **说明**: 更新腾讯云短信配置

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

### 18. 更新登录会话配置 [crypt-session]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/crypt-session`
- **说明**: 更新登录会话有效期

> 说明：登录态是「服务端会话 + Redis」（见 `comm.md` 的「登录会话」），JWT 已彻底移除。
> 保存时会用新模板重写 `crypt.toml`，老配置里的 `[jwt]` 段会被自动清理。

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `expire` | string | **是** | 会话有效期，支持算式（如 `15 * 24 * 60 * 60`），计算结果必须 > 0 |

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

### 19. 更新缓存默认配置 [cache-default]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/cache-default`
- **说明**: 更新缓存默认服务类型

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

### 20. 更新 Redis 缓存配置 [cache-redis]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/cache-redis`
- **说明**: 更新 Redis 缓存配置

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

### 21. 更新文件缓存配置 [cache-file]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/cache-file`
- **说明**: 更新文件缓存配置

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

### 22. 更新内存缓存配置 [cache-ram]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/cache-ram`
- **说明**: 更新内存缓存配置

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

### 23. 统一更新存储配置 [storage]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/storage`
- **说明**: 统一更新存储配置，支持同时修改多个存储配置段

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

### 24. 更新存储默认配置 [storage-default]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/storage-default`
- **说明**: 更新存储默认服务类型

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

### 25. 更新本地存储配置 [storage-local]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/storage-local`
- **说明**: 更新本地存储配置

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

### 26. 更新 COS 存储配置 [storage-cos]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/storage-cos`
- **说明**: 更新腾讯云 COS 存储配置

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

### 27. 更新附件配置 [storage-attachment]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/storage-attachment`
- **说明**: 更新附件管理配置

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

### 28. 更新发件队列配置 [sms-email-queue]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sms-email-queue`
- **说明**: 更新邮件发件队列（分批限流 + 失败重试）参数，写入 `config/sms.toml` 的 `[email]` 段

- **同一份配置的另一种写法**: `PUT /api/toml/sms` + `{ "name": "email_queue", ...同下表字段 }`

**请求参数**（全部可选，只提交要改的字段，未提交的保持原值）：

| 参数名 | 类型 | 必填 | 取值范围 | 默认值 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `batch_size` | int | 否 | 1 ~ 1000 | 10 | 每个批次窗口最多发送多少封通知类邮件 |
| `batch_interval` | int | 否 | 1 ~ 86400 | 600 | 批次间隔（秒），一批发满后等待多久再发下一批（600 = 10 分钟） |
| `retry_delay` | int | 否 | 1 ~ 86400 | 60 | 发送失败后的重试延迟（秒） |
| `max_attempts` | int | 否 | 1 ~ 10 | 3 | 单封邮件最大尝试次数（含首次），超出后标记失败并丢弃 |
| `send_timeout` | int | 否 | 5 ~ 600 | 30 | 单封发送超时（秒），超时按失败处理，避免 SMTP 卡住整个队列 |
| `verify_wait` | int | 否 | 0 ~ 60 | 10 | 验证码等待首轮发送结果的超时（秒），0 = 不等待（纯异步） |
| `queue_size` | int | 否 | 10 ~ 1000000 | 1000 | 队列最大长度（仅限制通知类邮件，验证码始终受理） |

**行为说明**:

- 值超出取值范围（或非数字）返回 `400`，字段名会在 msg 里给出；
- 分批限制只作用于**通知类邮件**（评论 / 回复、消息通知、欢迎邮件、运营通知等），
  验证码属于高优先级任务，入队即发、不占用批次额度；
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

### 29. 更新通知配置 [notification]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/notification`
- **说明**: 更新站内通知保留天数，写入 `config/app.toml` 的 `[notification]` 段

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `retention_days` | int | 是 | 通知保留天数，必须大于 0 |

**行为说明**:

- 配置文件已有 `retention_days` 时按行替换；有 `[notification]` 段但无该键时插入到段名下；
  两者都没有时追加 `[notification]` 段到文件末尾；
- 保存后由配置文件监听热更新，无需重启。

**请求示例**:

```json
{
    "retention_days": 60
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

**失败响应** (400):

```json
{
    "code": 400,
    "msg": "retention_days 必须大于 0！",
    "data": null
}
```

---

### 30. 获取敏感词库 [sensitive]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/sensitive`
- **说明**: 读取敏感词库（`config/sensitive.toml`）：解析结果 + 文件原文，后台用原文直接编辑

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "enabled": 1,
        "default_action": "review",
        "version": "3",
        "rules": [
            { "word": "广告", "action": "review", "scenes": ["comment"], "level": "medium", "match": "contains" }
        ],
        "whitelist": [],
        "scenes": ["comment", "article", "moment", "dm", "chat"],
        "actions": ["block", "limit", "review", "warn", "log"],
        "content": "# ======== 全站敏感词库 ========\nenabled = true\n..."
    }
}
```

> `enabled` 为 0/1；`content` 是文件原文（含注释），编辑后通过 31 原样提交。

### 31. 保存敏感词库 [sensitive]

- **请求方式**: `PUT`
- **请求地址**: `/api/toml/sensitive`
- **说明**: 覆盖保存词库原文，校验通过后写盘并立即热更新（不依赖文件监听时序）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `content` | string | 源码模式必填 | 完整 TOML 原文（含注释，末尾换行会被规整）；可视化模式改传下文的表单字段 |

**保存流程**（任一步失败均不落盘）:

1. 内容非空；
2. TOML 语法可解析；
3. 规则字段合法（见「特殊说明 → 6」）；
4. 顶层 `version` 为纯数字时自增（写入命中审计，用于回溯「某次误伤是哪一版规则判的」）；
5. 旧文件备份为 `config/sensitive.toml.bak`（只保留一份，供单步回滚）；
6. 写盘 + `ReloadSensitiveToml()`，返回最新解析结果（同 30）。

**成功响应** (200): 结构同 30，`msg` 为「保存成功！」。

**失败响应** (400):

```json
{
    "code": 400,
    "msg": "TOML 语法错误：toml: line 12: expected '.' or '=', but got '：' instead",
    "data": null
}
```

```json
{
    "code": 400,
    "msg": "rules[2] 的 action 不支持：blockx",
    "data": null
}
```

**可视化表单提交**（同一接口 `PUT /api/toml/sensitive` 的另一种载荷）:

后台「可视化配置」模式提交结构化数据（**不带 `content`**），服务端按模板生成规范 TOML 后，
走与上面完全相同的校验 / 备份 / 版本自增 / 热更新流程，响应结构也一致。
分派规则：带非空 `content` 走源码模式；否则只要出现 `rules` / `whitelist` / `enabled` /
`default_action` 任一字段就走表单模式；两者都没有返回 `400 配置内容不能为空！`（避免空载荷把词库清空）。

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | 否 | 总开关 |
| `default_action` | string | 否 | 规则缺省动作，缺省 `review` |
| `rules` | array | 否 | 规则数组，字段同 `[[rules]]` |
| `whitelist` | array | 否 | 白名单数组，字段同 `[[whitelist]]`（`action` 固定为放行，不需要传） |

**请求示例**:

```json
{
    "enabled": true,
    "default_action": "review",
    "rules": [
        { "word": "广告", "action": "review", "scenes": ["comment", "moment"], "level": "medium", "match": "contains" },
        { "word": "开户", "action": "review", "scenes": ["article"], "level": "low", "match": "exact" }
    ],
    "whitelist": [
        { "word": "官方公告", "scenes": ["article"], "match": "contains" }
    ]
}
```

**行为说明**:

- 表单保存会**按模板重写整个文件**：头部注释固定生成，源码里的手写注释不保留；
  需要保留注释请改用源码方式（`PUT /api/toml/sensitive` 提交 `content`）；
- `level = medium` 与 `match = contains` 是默认值，生成时省略，解析结果等价；
- `version` 不由前端提交：服务端沿用当前文件的版本号，保存时统一 +1；
- 校验与源码保存共用同一套规则（缺 `word`、`action` / `match` / `level` / `scenes` 拼错、
  词条超过 64 字符都会返回 400），失败的提交不会落盘、也不会覆盖备份。

### 32. 测试敏感词 [sensitive-test]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/sensitive-test`
- **说明**: 用当前内存词库试跑一条文本；`uid=0`，**不写命中审计**（避免测试流量污染统计）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `scene` | string | **是** | comment、article、moment、dm、chat |
| `text` | string | 否 | 待测文本，留空返回 `allow` |

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "测试完成！",
    "data": {
        "scene": "comment",
        "text_length": 12,
        "decision": {
            "action": "block",
            "word": "广告发票",
            "level": "high",
            "match": "contains",
            "scene": "comment",
            "version": "3"
        }
    }
}
```

> 未命中 / 命中白名单时 `decision.action` 为 `allow`，其余字段为空串。测试文本同样经过归一化与 HTML 剥标签，`广<b>告</b>` 会被判定命中。

### 33. 回滚敏感词库 [sensitive-rollback]

- **请求方式**: `POST`
- **请求地址**: `/api/toml/sensitive-rollback`
- **说明**: 回滚到上一次保存前的词库（单步），覆盖「保存后发现大面积误伤」的场景

**请求参数**: 无。

**行为说明**:

- 备份 `config/sensitive.toml.bak` 不存在或为空 → `400 没有可回滚的备份！`；
- 备份内容先过 TOML 语法与规则字段校验，损坏的备份直接拒绝（避免回滚后整份词库静默失效）；
- 版本号不倒退：取 `max(备份版本, 回滚前版本) + 1`，否则同一个 `version` 会对应两份内容，审计回溯会指错版本；
- 写盘后立即重载，返回最新解析结果（同 30）。

**成功响应** (200): 结构同 30，`msg` 为「已回滚到上一次保存前的版本！」。

### 34. 命中统计 [sensitive-stats]

- **请求方式**: `GET`
- **请求地址**: `/api/toml/sensitive-stats`
- **说明**: 最近 7 天命中统计，按「场景 + 处置动作」聚合（数据来自 `inis_sensitive_hits`）

**成功响应** (200):

```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "days": 7,
        "items": [
            { "scene": "comment", "action": "block", "total": 12 },
            { "scene": "comment", "action": "review", "total": 5 },
            { "scene": "dm", "action": "log", "total": 3 }
        ]
    }
}
```

> 没有命中记录时 `items` 为空数组；统计只聚合，不返回命中的词条与内容摘要在内。

---

### 35. INDEX 接口

- **接口类型**: INDEX 接口
- **请求方式**: `GET`
- **请求地址**: `/api/toml/index`
- **说明**: 配置管理首页（无实际功能）

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
- 涉及脱敏的字段：`password`、`access_key_secret`、`secret_key`、`access_key`、`access_key_id`、`secret_id`、`key`
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

### 6. 敏感词库（config/sensitive.toml）

词库**不在 config 表里**，独立成文件（与 `app.toml` 解耦），便于备份 / 回滚 / diff，也方便运维直接看规则；
保存后只重载这一份配置，不影响数据库、通知等其它配置。实现见 `app/model/sensitive.go`、`app/facade/sensitive.go`。

**文件结构**:

| 字段 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | bool | 否 | 总开关，缺省 `true`；关闭后所有场景直接放行 |
| `default_action` | string | 否 | 规则未写 `action` 时的兜底动作，缺省 `review` |
| `version` | string | 否 | 词库版本，写入命中审计；纯数字时每次保存自增（`v1` / 日期这类保持原样） |
| `[[rules]]` | array | 否 | 词条规则，字段见下 |
| `[[whitelist]]` | array | 否 | 与 `rules` 同构，命中即放行（官方公告、认证机构、业务模板） |

**规则字段**:

| 字段 | 类型 | 必填 | 取值 / 说明 |
| :--- | :--- | :--- | :--- |
| `word` | string | **是** | 词条，最长 64 个字符（超长在保存时直接 400） |
| `action` | string | 否 | `block` 拒绝 / `limit` 限流拒绝 / `review` 进审核 / `warn` 提示后放行 / `log` 仅记录；白名单里写 `allow` 也合法 |
| `scenes` | array | 否 | `comment` / `article` / `moment` / `dm` / `chat`；留空 = 全部场景 |
| `level` | string | 否 | `high` / `medium` / `low`，缺省 `medium`，用于同动作下择优 |
| `match` | string | 否 | `contains`（默认，走 AC 自动机）/ `exact`（整段相等） |

**匹配行为**:

- 判定前统一归一化：全角转半角、大小写折叠、去零宽字符、常见繁简映射、剥 HTML 标签，再移除空白 / 标点 / 符号，
  因此 `广 告`、`廣-告`、`广<b>告</b>` 都能命中「广告」；
- **白名单优先**：命中白名单直接放行，不再看规则；
- 命中多条时择优：动作权重（block > limit > review > warn > log）→ 风险等级 → 词条更长者；
- `contains` 规则编译成 AC 自动机，规则量大时不会退化成逐条 `Contains`；`exact` 走哈希表整段比对；
- 配置变更推进「配置代次」，model 层按代次重建匹配器，无需重启。

**各业务场景的处置**:

| 场景 | 入口 | 处置 |
| :--- | :--- | :--- |
| 评论 / 回复 | `comment.create` / `comment.update` | block、limit 拒绝；review 进待审核（编辑命中 review 重新进审核） |
| 文章 | `article.create` / `article.update` | 判定「标题 + 摘要 + 正文」；block、limit 拒绝；review 进待审核 |
| 动态 | `moments.create` / `moments.update` | 同上，判定正文 |
| 私信 | `model.ChatMessages.Send` | 文本与拍一拍后缀判定；block、limit 拒绝；review / warn / log 放行并记审计 |
| 聊天室 | 聊天室发送接口 | 同上（私信 / 聊天室隐私敏感，不塞人工审核队列） |

> 命中 `warn` 时，评论 / 文章 / 动态的成功响应 `msg` 会追加提示文案，例如
> `创建成功！（内容包含敏感词「广告」，已放行，请注意用词！）`，前端展示 `msg` 即可，无需额外解析字段。
> 私信 / 聊天室只记审计，处置依靠撤回与禁言能力。

**命中审计（inis_sensitive_hits）**:

| 字段 | 说明 |
| :--- | :--- |
| `uid` / `scene` / `action` / `word` / `level` / `match` | 命中上下文 |
| `version` | 判定时的词库版本，用于回溯「这次误伤是哪一版规则判的」 |
| `content_hash` | 归一化后内容的 SHA-256，用于同内容聚合，不存全文 |
| `excerpt` | 内容摘要，最多 120 字；**私信 / 聊天室不落摘要**，仅 block / review 且非私密场景才存 |

审计异步落库（独立 goroutine），失败只记日志，不影响业务主流程。

**备份与版本**:

- 每次保存前把旧文件备份为 `config/sensitive.toml.bak`（只保留一份，单步回滚）；
- 备份文件不会被版本库收录（`config/.gitignore` 的 `*.toml` / `*.toml.bak`）。
