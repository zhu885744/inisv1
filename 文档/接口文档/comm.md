## Comm 接口文档

### 接口概述
公共控制器提供用户登录、注册、退出等核心认证功能。

### 状态码规范
| 状态码 | 含义 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 正常返回数据 |
| 201 | 创建成功 | 资源创建成功 |
| 400 | 请求错误 | 参数验证失败 |
| 401 | 未授权 | 未登录或权限不足 |
| 403 | 禁止访问 | 无操作权限 |
| 405 | 方法不允许 | 请求方法错误 |
| 412 | 前置条件失败 | Token为空 |
| 500 | 服务器错误 | 服务端异常 |

### 接口列表

#### 1. 用户登录
- **请求方式**: `POST`
- **请求路径**: `/api/comm/login`
- **参数**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | account | string | 是 | 账号/邮箱/手机号 |
  | password | string | 是 | 密码 |
  | source | string | 否 | 登录来源，默认`default` |

- **响应示例**:
```json
{
  "code": 200,
  "msg": "登录成功！",
  "data": {
    "user": {...},
    "token": "xxx"
  }
}
```

#### 2. 用户注册
- **请求方式**: `POST`
- **请求路径**: `/api/comm/register`
- **参数**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | social | string | 是 | 邮箱或手机号（用于接收验证码） |
  | code | string | 是 | 验证码 |
  | password | string | 是 | 密码 |
  | account | string | 否 | 用户名 |
  | nickname | string | 否 | 昵称 |

- **响应示例**:
```json
{
  "code": 200,
  "msg": "注册成功！",
  "data": {
    "user": {...},
    "token": "xxx"
  }
}
```

#### 3. 重置密码
- **请求方式**: `POST`
- **请求路径**: `/api/comm/reset-password`
- **参数**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | account | string | 否 | 账号（与social二选一） |
  | social | string | 否 | 邮箱或手机号（与account二选一） |
  | code | string | 是 | 验证码 |
  | password | string | 是 | 新密码 |

#### 4. 验证Token
- **请求方式**: `POST`
- **请求路径**: `/api/comm/check-token`
- **参数**:
  | 参数名 | 类型 | 必填 | 说明 |
  | :--- | :--- | :--- | :--- |
  | renew | bool | 否 | 是否刷新token |

- **响应示例**:
```json
{
  "code": 200,
  "msg": "合法的token！",
  "data": {
    "user": {...},
    "token": "xxx",
    "valid_time": 7200
  }
}
```

#### 5. 退出登录
- **请求方式**: `DELETE`
- **请求路径**: `/api/comm/logout`

### 特殊说明
- 登录支持密码密文解密（通过X-Gorgon、X-Khronos、X-Argus请求头）
- 注册需先发送验证码（不传code参数），再提交验证码完成注册
- 登录成功后会自动写入cookie存储token
- 登录成功后会增加用户经验值