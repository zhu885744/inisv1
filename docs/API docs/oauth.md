## OAuth 第三方登录 API

### 接口概述

第三方登录控制器，支持 **QQ / GitHub / Gitee** 三种方式，一个入口同时覆盖三种场景：

| 场景 | 条件 | 结果 |
| :--- | :--- | :--- |
| 登录 | 第三方账号已绑定过本站账号 | 直接发 token |
| 注册 | 未绑定 + 未登录 + 后台允许自动注册 | 建号 + 绑定 + 发 token |
| 绑定 | 未绑定 + 已登录 | 绑到当前账号 |

设计取舍：token 一律由后端在响应体里下发（不做「整跳转把 token 放 URL」），
AppSecret 只留在服务端；前端只负责拼授权页地址（从 `config` 接口拿 AppID / 回调地址 / scope）。

### 配置（config 表 key = `SYSTEM_OAUTH`）

配置保存在 `config` 表的一条记录里（json 字段），可在后台「系统配置」保存，路径 `/api/config/save`。
未保存过时使用代码里的默认值（三个平台默认 `enable=0`，即关闭）。

```json
{
  "qq":     { "enable": 1, "app_id": "应用ID", "app_key": "应用密钥", "redirect": "https://你的站点/auth/oauth" },
  "github": { "enable": 1, "app_id": "Client ID", "app_key": "Client Secret", "redirect": "https://你的站点/auth/oauth" },
  "gitee":  { "enable": 1, "app_id": "Client ID", "app_key": "Client Secret", "redirect": "https://你的站点/auth/oauth" },
  "auto_register": 1,
  "timeout": 15,
  "proxy": ""
}
```

| 字段 | 说明 |
| :--- | :--- |
| `enable` | 平台开关。同时要求 app_id / app_key 已填写，平台才算「可用」（见 `config` 接口的 enable 字段） |
| `app_id` / `app_key` | 第三方应用凭证。`app_key` 只在服务端使用，接口不会返回 |
| `redirect` | 回调地址，**必须与第三方后台登记的完全一致**。QQ / Gitee 换 token 时必须回传该值，GitHub 可省略但仍建议保持一致 |
| `auto_register` | 第三方账号未绑定时是否自动注册新账号；`0` 时接口返回 `need_bind`，引导「先登录再绑定」 |
| `timeout` | 第三方接口请求超时（秒），3 ~ 60，默认 15。出网慢的服务器可调大 |
| `proxy` | 第三方接口请求走的代理，支持 `http://`、`https://`、`socks5://`（如 `http://127.0.0.1:7890`），留空表示直连。**GitHub 直连不通（context deadline exceeded）时就是靠它解决** |

保存后立即生效（`config/save` 会清掉 `config[SYSTEM_OAUTH]` 缓存）。

> 前端回调页固定为 `/auth/oauth`，三个平台可以共用同一个回调地址。
> 发起登录时前端把「平台 / state / 落地页」记在 sessionStorage，回调时再取出来用；
> 不要把 platform 拼进 redirect_uri —— QQ / Gitee 对回调地址是全字符串精确匹配，多一个 query 就会报 redirect_uri 不一致。
>
> 注意：换 token 是**服务器出网**完成的（与浏览器无关）。若服务器连不上第三方站点，
> 接口会返回「服务器出网超时」类提示，此时请在后台配置 `proxy`，或放通服务器出网。

### 状态码规范

| 状态码 | 含义 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 登录 / 注册 / 绑定 / 解绑成功，或返回 `need_bind` 引导 |
| 204 | 数据不存在 | 绑定关系指向的用户已被删除 |
| 400 | 请求参数错误 | 缺 code、平台未开启、平台参数未配置、第三方返回错误、解绑会失去唯一登录方式 |
| 401 | 未登录 | 调用需登录的接口（mine / bind / unbind） |
| 405 | 方法不允许 | 调用了不存在的方法 |

### 接口列表

#### 1. 获取第三方登录配置 [config]

**请求方式**: `GET` ｜ **公开**

**请求地址**: `/api/oauth/config`

**响应示例**:

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "qq": {
      "platform": "qq",
      "name": "QQ",
      "enable": 1,
      "app_id": "102045704",
      "redirect": "https://你的站点/auth/oauth",
      "authorize": "https://graph.qq.com/oauth2.0/authorize",
      "scope": "get_user_info"
    },
    "github": { "platform": "github", "name": "GitHub", "enable": 1, "app_id": "Iv1.xxxx", "redirect": "...", "authorize": "https://github.com/login/oauth/authorize", "scope": "read:user" },
    "gitee":  { "platform": "gitee",  "name": "Gitee",  "enable": 0, "app_id": "",         "redirect": "...", "authorize": "https://gitee.com/oauth/authorize",       "scope": "user_info" },
    "auto_register": 1,
    "platforms": [ "…同上，便于前端直接遍历…" ]
  }
}
```

前端拼授权地址的方式（`state` 由前端生成并自行校验，服务端不校验）：

```
{authorize}?response_type=code&client_id={app_id}&redirect_uri={redirect}&scope={scope}&state={state}
```

#### 2. QQ / GitHub / Gitee 登录 [qq / github / gitee]

**请求方式**: `GET` / `POST` ｜ **公开**

**请求地址**: `/api/oauth/qq`、`/api/oauth/github`、`/api/oauth/gitee`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| code | string | 是 | 第三方授权回调带回的授权码 |
| state | string | 否 | 由前端自行校验，服务端透传不校验 |

**响应示例 A：登录 / 注册成功（与密码登录同一结构）**

```json
{
  "code": 200,
  "msg": "登录成功！",
  "data": {
    "user": { "id": 1, "account": "xxxx", "nickname": "用户昵称", "avatar": "头像URL" },
    "token": "JWT Token",
    "valid_time": 1296000,
    "bind": true,
    "register": true,
    "platform": "github"
  }
}
```

> `register: true` 表示本次顺带建了新号；`bind: true` 表示该第三方账号已与本站账号建立绑定。
> 后台开启「注册人工审核」时，新账号会置为待审核并返回 `{ user, need_audit: true, bind: true, register: true }`（**不发 token**），审核通过后可直接再次点第三方登录。

**响应示例 B：未登录且未绑定（需先登录再绑定）**

```json
{
  "code": 200,
  "msg": "该第三方账号还未绑定本站账号，请先登录后完成绑定！",
  "data": {
    "need_bind": true,
    "ticket": "用于登录后调 /api/oauth/bind 的短期票据（10 分钟）",
    "platform": "qq",
    "name": "QQ",
    "nickname": "第三方昵称",
    "avatar": "第三方头像"
  }
}
```

触发条件：未登录 + 第三方账号未绑定 + （后台关闭了自动注册 或 关闭了注册功能）。

#### 3. 绑定第三方账号 [bind]

**请求方式**: `POST` / `PUT` ｜ **需登录**

**请求地址**: `/api/oauth/bind`

**请求参数**（二选一）:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| key | string | 方式一 | 登录接口返回的 `ticket`（先登录、再绑定） |
| platform | string | 方式二 | `qq` / `github` / `gitee` |
| code | string | 方式二 | 已登录状态下重新发起授权拿到的 code |

**响应示例**:

```json
{ "code": 200, "msg": "绑定成功！", "data": { "bind": true, "uid": 1, "platform": "qq" } }
```

#### 4. 解绑第三方账号 [unbind]

**请求方式**: `POST` / `DELETE` ｜ **需登录**

**请求地址**: `/api/oauth/unbind`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| platform | string | 是 | `qq` / `github` / `gitee` |

安全约束：解绑后如果密码、邮箱、手机号、其它第三方绑定都不存在（即解绑后无法再登录），
接口会拒绝并返回 400「解绑后你将无法登录本站，请先设置密码或绑定其它登录方式！」。

#### 5. 我绑定的第三方账号 [mine]

**请求方式**: `GET` ｜ **需登录**

**请求地址**: `/api/oauth/mine`

**响应示例**:

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "list": [
      { "id": 1, "platform": "github", "name": "GitHub", "nickname": "octocat", "avatar": "…", "create_time": 1750000000 }
    ],
    "available": [
      { "platform": "qq", "name": "QQ", "app_id": "102045704", "redirect": "…", "authorize": "…", "scope": "get_user_info" }
    ]
  }
}
```

`list` 为已绑定平台（可直接渲染「已绑定」），`available` 为「已开启且当前账号还没绑」的平台
（可直接渲染「去绑定」按钮，字段与 `config` 接口一致）。

### 数据表

绑定关系落在独立表 `inis_user_oauth`（服务启动时自动迁移），关键约束：

| 字段 | 说明 |
| :--- | :--- |
| uid | 本站用户 ID |
| platform | 平台标识（qq / github / gitee） |
| openid | 平台用户唯一标识（GitHub 用数字 id；换用户名不影响） |
| unionid | QQ 的 UnionID（同主体多应用打通用，其它平台为空） |
| nickname / avatar | 第三方昵称头像，仅展示用 |

约束与语义：

1. `(platform, openid)` 唯一 —— 一个第三方账号只能绑一个本站用户（绑定的若是别人，接口明确拒绝，不做「偷偷切账号」）；
2. 同一用户同一平台只能绑一个第三方账号（要换号请先解绑）；
3. 解绑是软删除，重新绑定时复用同一行（因此重新绑定不会丢创建时间、也不会撞唯一索引）；
4. **不保存 access_token / refresh_token**：登录只依赖 openid，长期留一份第三方令牌等于多一个泄露面。

### 特殊说明

1. **公开 / 需登录**：`config` 与三个平台登录接口为公开接口（已在 `auth_rules` 标记 `type=common`，
   并在 `app/api/middleware/rule.go` 的 `publicRoutes` 兜底，保证规则尚未落库时也能登录）；
   `mine` / `bind` / `unbind` 为 `type=login`，必须已登录。
2. **账号策略**：第三方登录建号时只取昵称与头像，**不写第三方邮箱/手机号** ——
   这两个字段在本站是唯一索引且用于登录与找回密码，直接回写可能把已有账号的邮箱占掉。
   需要邮箱请走「联系方式」页的验证码流程。
3. **注册开关联动**：管理员关闭注册（`ALLOW_REGISTER`）时不会通过第三方登录建号，
   而是返回 `need_bind` 引导用户先注册再绑定；开启「注册人工审核」时新账号进入待审核。
4. **登录风控**：发token 前会走与密码登录完全相同的状态校验（冻结 / 待审核 / 封禁），
   避免出现「密码登录被拦、第三方登录能进」的绕过。
5. **错误信息**：平台未开启、AppID/AppKey/回调地址未配置、第三方返回的具体错误都会原样返回 msg，便于定位。
