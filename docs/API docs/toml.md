# Toml 配置管理接口文档

## 接口概述

`toml` 控制器用于管理系统配置文件，包括短信、缓存、加密、存储等配置的读取、更新和测试功能。

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **GET 接口** | log、sms、cache、crypt、storage - 获取配置信息 |
| **POST 接口** | 测试各类服务连接 |
| **PUT 接口** | 更新各类配置 |
| **DELETE 接口** | 暂不支持 |

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
| `name` | string | 否 | 指定配置项：local、oss、cos、kodo |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "local": {},
        "oss": {},
        "cos": {},
        "kodo": {},
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

#### 2.6 测试 OSS 连接

- **路径**: `/api/toml/test-oss`
- **方法**: `POST`
- **描述**: 测试阿里云 OSS 连接

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `bucket` | string | **是** | Bucket 名称 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试OSS连接成功！",
    "data": null
}
```

#### 2.7 测试 COS 连接

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

#### 2.8 测试 KODO 连接

- **路径**: `/api/toml/test-kodo`
- **方法**: `POST`
- **描述**: 测试七牛云 KODO 连接

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key` | string | **是** | Access Key |
| `secret_key` | string | **是** | Secret Key |
| `bucket` | string | **是** | Bucket 名称 |
| `region` | string | **是** | 地域 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "测试KODO连接成功！",
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

#### 3.11 更新存储默认配置

- **路径**: `/api/toml/storage-default`
- **方法**: `PUT`
- **描述**: 更新存储默认服务类型

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `value` | string | **是** | 默认存储类型：local、oss、cos、kodo |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.12 更新本地存储配置

- **路径**: `/api/toml/storage-local`
- **方法**: `PUT`
- **描述**: 更新本地存储配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `domain` | string | 否 | 域名 |
| `path` | string | 否 | 存储路径 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.13 更新 OSS 存储配置

- **路径**: `/api/toml/storage-oss`
- **方法**: `PUT`
- **描述**: 更新阿里云 OSS 存储配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key_id` | string | **是** | AccessKey ID |
| `access_key_secret` | string | **是** | AccessKey Secret |
| `endpoint` | string | **是** | 端点地址 |
| `bucket` | string | **是** | Bucket 名称 |
| `domain` | string | 否 | 域名 |
| `path` | string | 否 | 存储路径 |

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
| `app_id` | string | **是** | App ID |
| `bucket` | string | **是** | Bucket 名称 |
| `region` | string | **是** | 地域 |
| `domain` | string | 否 | 域名 |
| `path` | string | 否 | 存储路径 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
    "data": null
}
```

#### 3.15 更新 KODO 存储配置

- **路径**: `/api/toml/storage-kodo`
- **方法**: `PUT`
- **描述**: 更新七牛云 KODO 存储配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `access_key` | string | **是** | Access Key |
| `secret_key` | string | **是** | Secret Key |
| `bucket` | string | **是** | Bucket 名称 |
| `region` | string | **是** | 地域 |
| `domain` | string | **是** | 域名 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "修改成功！",
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
- 敏感信息（如密钥）不在响应中返回
- 配置修改需要管理员权限