# API 汇总

> INIS 项目（Go + Gin 框架）全部接口与控制器汇总文档。
> 本文档基于 `app/` 目录源码自动整理，覆盖 `/api`、`/dev`、`/socket` 及首页路由。

---

## 目录

1. [概述](#一概述)
2. [路由机制与请求规范](#二路由机制与请求规范)
3. [通用响应与状态码](#三通用响应与状态码)
4. [中间件](#四中间件)
5. [控制器总览](#五控制器总览)
6. [通用 CRUD 方法说明](#六通用-crud-方法说明)
7. [各控制器 API 明细](#七各控制器-api-明细)
8. [其他路由（dev / socket / index）](#八其他路由dev--socket--index)
9. [通用查询参数](#九通用查询参数)

---

## 一、概述

INIS 采用 **动态资源路由** 设计：所有业务接口统一通过 `/api/{控制器}/{方法}` 的形式访问，请求方式为 `GET` / `POST` / `PUT` / `DELETE`。控制器（Controller）是接口的分组单元，每个控制器内按 HTTP 方法挂载若干个业务方法（method）。

- 后端框架：`gin-gonic/gin`
- 控制器基类：`app/api/controller/base.go` 中的 `base`（提供参数解析、缓存、权限、响应封装等通用能力）
- 控制器统一接口契约：

```go
type ApiInterface interface {
    IGET(ctx *gin.Context)   // GET 请求入口
    IPOST(ctx *gin.Context)  // POST 请求入口
    IPUT(ctx *gin.Context)   // PUT 请求入口
    IDEL(ctx *gin.Context)   // DELETE 请求入口
    INDEX(ctx *gin.Context)  // 控制器根路径入口
}
```

---

## 二、路由机制与请求规范

### 1. 动态路由规则

| 路由 | 说明 |
| :--- | :--- |
| `/{controller}` | 控制器根路径，调用该控制器的 `INDEX` 方法（任意 HTTP 方法） |
| `/{controller}/{method}` | 按 HTTP 方法分发到 `IGET` / `IPOST` / `IPUT` / `IDEL`，再由其中的 `allow` 映射表路由到具体业务方法 |

### 2. 请求路径格式

```
/api/{controller}/{method}
```

例如：
- `GET  /api/article/all` —— 获取文章列表
- `POST /api/comm/login` —— 用户登录
- `PUT  /api/users/update` —— 更新用户
- `DELETE /api/article/remove` —— 软删除文章

### 3. 参数传递

- 请求参数支持 **Query 参数、表单（form-data）、JSON Body** 三种方式，由 `Params()` 中间件统一解析后挂载到上下文的 `params` 中。
- `method` 为路径参数，用于路由到具体业务方法（自动转为小写）。

---

## 三、通用响应与状态码

### 1. 统一响应结构

```json
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| code | int | 状态码 |
| msg | string | 提示信息 |
| data | any | 业务数据（无数据时为 `null` 或空数组） |

### 2. 状态码规范

| 状态码 | 含义 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 获取/操作成功 |
| 201 | 创建成功 | 新增数据、验证码发送成功 |
| 202 | 已接受 | 控制器根路径（INDEX）无实际功能 |
| 204 | 无内容 | 查询无数据、无可操作数据 |
| 400 | 请求错误 | 参数校验失败、操作失败 |
| 401 | 未授权 | 未登录或 token 无效 |
| 403 | 禁止访问 | 无权限操作 |
| 405 | 方法不允许 | 请求方式错误或 method 名不存在 |
| 412 | 前置条件失败 | 必要参数（如 token）为空 |
| 500 | 服务器错误 | 系统内部异常 |

---

## 四、中间件

### 1. 全局中间件（所有路由）

| 中间件 | 说明 |
| :--- | :--- |
| `Cors()` | 跨域请求处理 |
| `Install()` | 安装状态检测 |

### 2. API 路由中间件（`/api/*` 默认挂载，按顺序执行）

| 顺序 | 中间件 | 说明 |
| :--- | :--- | :--- |
| 1 | `IpBlack()` | IP 黑名单校验 |
| 2 | `QpsPoint()` | 单接口 QPS 限流 |
| 3 | `QpsGlobal()` | 全局 QPS 限流 |
| 4 | `Params()` | 统一参数解析 |
| 5 | `Jwt()` | JWT 鉴权（校验登录态；无 token 时以访客身份放行，是否强制登录由 `Rule()` 的规则类型决定） |
| 6 | `Rule()` | 接口权限规则校验 |
| 7 | `ApiKey()` | API Key 校验 |
| 8 | `Restriction()` | 用户限制（封禁/冻结等） |

### 3. 其他路由中间件

| 路由 | 中间件 | 说明 |
| :--- | :--- | :--- |
| `/dev/install/*` | `LocalOnly()` + `Params()` | 安装接口，仅允许本机 loopback 访问 |
| `/dev/info/*` | `Params()` | 信息接口；`time`/`version` 公开，`system`/`device`/`renew`/`kill` 仅本机 loopback 访问 |
| `/socket` | `Jwt()` + `App()` | WebSocket 鉴权与 App 校验 |

---

## 五、控制器总览

共 **37 个控制器**：`/api` 下 35 个，`/dev` 下 2 个。

| # | 控制器 key | 控制器类 | 中文名 | 主要能力 |
| :---: | :--- | :--- | :--- | :--- |
| 1 | `comm` | `Comm` | 公共/认证 | 登录、注册、Token、重置密码、退出 |
| 2 | `test` | `Test` | 测试 | 调试接口、上传测试（已下线，不再注册路由） |
| 3 | `exp` | `EXP` | 经验值 | 经验/签到/活跃榜 |
| 4 | `toml` | `Toml` | 配置管理 | 短信/缓存/存储/加密等配置读写与测试 |
| 5 | `tags` | `Tags` | 标签 | 标签 CRUD |
| 6 | `pages` | `Pages` | 单页 | 单页 CRUD |
| 7 | `users` | `Users` | 用户 | 用户 CRUD、封禁、申诉、注销 |
| 8 | `oauth` | `OAuth` | 第三方登录 | QQ / GitHub 登录 |
| 9 | `links` | `Links` | 友情链接 | 友链 CRUD、状态检测 |
| 10 | `proxy` | `Proxy` | 反向代理 | 请求转发 |
| 11 | `level` | `Level` | 等级 | 用户等级 CRUD |
| 12 | `banner` | `Banner` | 轮播图 | 轮播图 CRUD |
| 13 | `config` | `Config` | 系统配置 | 系统配置项 CRUD |
| 14 | `article` | `Article` | 文章 | 文章 CRUD |
| 15 | `comment` | `Comment` | 评论 | 评论 CRUD、扁平列表 |
| 16 | `placard` | `Placard` | 公告 | 公告 CRUD |
| 17 | `api-keys` | `ApiKeys` | API 密钥 | API 密钥 CRUD |
| 18 | `ip-black` | `IpBlack` | IP 黑名单 | 黑名单 CRUD |
| 19 | `ip-white` | `IpWhite` | IP 白名单 | 白名单 CRUD |
| 20 | `qps-warn` | `QpsWarn` | QPS 告警 | QPS 告警规则 CRUD |
| 21 | `auth-group` | `AuthGroup` | 权限组 | 权限组 CRUD、成员设置 |
| 22 | `auth-pages` | `AuthPages` | 权限页面 | 权限页面 CRUD |
| 23 | `auth-rules` | `AuthRules` | 权限规则 | 权限规则 CRUD |
| 24 | `links-group` | `LinksGroup` | 友链分组 | 友链分组 CRUD |
| 25 | `article-group` | `ArticleGroup` | 文章分组 | 文章分组 CRUD、树形结构 |
| 26 | `search` | `Search` | 搜索 | 全站/分类搜索 |
| 27 | `rss` | `Rss` | RSS | RSS 订阅输出 |
| 28 | `moments` | `Moments` | 动态 | 动态 CRUD、评论、置顶 |
| 29 | `attachment` | `Attachment` | 附件 | 附件上传/管理、表情 |
| 30 | `user-likes` | `UserLikes` | 点赞 | 点赞/取消点赞、点赞统计 |
| 31 | `user-collects` | `UserCollects` | 收藏 | 收藏/取消收藏、收藏统计 |
| 32 | `user-follows` | `UserFollows` | 关注 | 关注/取关、关注粉丝列表 |
| 33 | `notification` | `Notification` | 通知 | 通知收发、已读管理 |
| 34 | `integral` | `Integral` | 积分 | 积分余额、流水、任务规则、调整 |
| 35 | `goods` | `Goods` | 商品/积分商城 | 商品、订单、兑换购买 |
| 36 | `info` | `Info` | 系统信息（dev） | 系统/版本/设备/时间信息 |
| 37 | `install` | `Install` | 安装（dev） | 安装锁、数据库初始化、创建管理员 |

---

## 六、通用 CRUD 方法说明

大多数控制器继承同一套标准方法，语义完全一致（仅操作的数据表不同）。下表集中说明，后文各控制器明细中标注「通用」即指此表。

| HTTP | method | 说明 |
| :--- | :--- | :--- |
| GET | `one` | 获取单条数据（按条件查询第一条） |
| GET | `all` | 获取列表（分页，返回 `data`/`count`/`page`） |
| GET | `sum` | 对指定 `field` 求和 |
| GET | `min` | 求指定 `field` 的最小值 |
| GET | `max` | 求指定 `field` 的最大值 |
| GET | `rand` | 随机获取指定数量的数据 |
| GET | `count` | 统计满足条件的数据数量 |
| GET | `column` | 获取指定字段的单列数据 |
| POST | `save` | 保存数据（有 `id` 则更新，无则创建） |
| POST | `create` | 创建新数据 |
| PUT | `update` | 更新数据（需 `id`） |
| PUT | `restore` | 恢复软删除的数据（需 `ids`） |
| DELETE | `remove` | 软删除（移入回收站，需 `ids`） |
| DELETE | `delete` | 彻底删除（需 `ids`） |
| DELETE | `clear` | 清空回收站 |

---

## 七、各控制器 API 明细

> 说明：「通用」方法见[第六节](#六通用-crud-方法说明)；表格中 `method` 即路径中的 `{method}` 段。

### 1. comm 公共/认证控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| POST | `login` | `/api/comm/login` | 用户登录（账号/邮箱/手机号 + 密码） |
| POST | `register` | `/api/comm/register` | 用户注册（验证码 + 密码） |
| POST | `check-token` | `/api/comm/check-token` | 校验 Token，可选续期 |
| POST | `reset-password` | `/api/comm/reset-password` | 重置密码（验证码） |
| DELETE | `logout` | `/api/comm/logout` | 退出登录（清除 Cookie） |

### 2. test 测试控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `request` | `/api/test/request` | 测试网络请求 |
| POST | `return-url` | `/api/test/return-url` | 测试同步回调地址 |
| POST | `notify-url` | `/api/test/notify-url` | 测试异步回调地址 |
| POST | `request` | `/api/test/request` | 测试网络请求 |
| POST | `upload` | `/api/test/upload` | 测试文件上传 |
| PUT | `request` | `/api/test/request` | 测试网络请求 |
| DELETE | `request` | `/api/test/request` | 测试网络请求 |

### 3. exp 经验值控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/exp/{method}` | 通用 |
| GET | `active` | `/api/exp/active` | 经验活跃榜 |
| GET | `check-in-status` | `/api/exp/check-in-status` | 当日签到状态 |
| GET | `check-in-rank` | `/api/exp/check-in-rank` | 签到排行榜 |
| GET | `check-in-calendar` | `/api/exp/check-in-calendar` | 签到日历 |
| GET | `rules` | `/api/exp/rules` | 经验任务规则 |
| POST | `save` / `create` | `/api/exp/{method}` | 通用 |
| POST | `share` | `/api/exp/share` | 分享奖励 |
| POST | `check-in` | `/api/exp/check-in` | 每日签到 |
| POST | `give` | `/api/exp/give` | 给指定用户增减经验（管理员） |
| PUT | `update` / `restore` | `/api/exp/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/exp/{method}` | 通用 |

### 4. toml 配置管理控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `log` | `/api/toml/log` | 获取日志服务配置 |
| GET | `sms` | `/api/toml/sms` | 获取短信服务配置 |
| GET | `cache` | `/api/toml/cache` | 获取缓存服务配置 |
| GET | `crypt` | `/api/toml/crypt` | 获取加密服务配置 |
| GET | `storage` | `/api/toml/storage` | 获取存储服务配置 |
| POST | `test-sms-email` | `/api/toml/test-sms-email` | 测试邮件服务 |
| POST | `test-sms-aliyun` | `/api/toml/test-sms-aliyun` | 发送阿里云测试短信 |
| POST | `test-sms-aliyun-number-verify` | `/api/toml/test-sms-aliyun-number-verify` | 测试阿里云号码验证 |
| POST | `test-sms-tencent` | `/api/toml/test-sms-tencent` | 发送腾讯云测试短信 |
| POST | `test-redis` | `/api/toml/test-redis` | 测试 Redis 连接 |
| POST | `test-oss` | `/api/toml/test-oss` | 测试 OSS 连接 |
| POST | `test-cos` | `/api/toml/test-cos` | 测试 COS 连接 |
| POST | `test-kodo` | `/api/toml/test-kodo` | 测试 KODO 连接 |
| PUT | `sms` / `sms-email` / `sms-aliyun` / `sms-aliyun-number-verify` / `sms-tencent` / `sms-drive` | `/api/toml/{method}` | 修改短信相关配置 |
| PUT | `crypt-jwt` | `/api/toml/crypt-jwt` | 修改 JWT 配置 |
| PUT | `cache-default` / `cache-redis` / `cache-file` / `cache-ram` | `/api/toml/{method}` | 修改缓存相关配置 |
| PUT | `storage` / `storage-default` / `storage-local` / `storage-oss` / `storage-cos` / `storage-kodo` / `storage-attachment` | `/api/toml/{method}` | 修改存储相关配置 |

### 5. tags 标签控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/tags/{method}` | 通用 |
| POST | `save` / `create` | `/api/tags/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/tags/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/tags/{method}` | 通用 |

### 6. pages 单页控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/pages/{method}` | 通用 |
| POST | `save` / `create` | `/api/pages/{method}` | 通用 |
| POST | `update` | `/api/pages/update` | 更新页面（POST 方式） |
| PUT | `update` / `restore` | `/api/pages/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/pages/{method}` | 通用 |

### 7. users 用户控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/users/{method}` | 通用 |
| GET | `blackroom` | `/api/users/blackroom` | 小黑屋公示（封禁名单，公开） |
| POST | `save` / `create` | `/api/users/{method}` | 通用 |
| POST | `appeal` | `/api/users/appeal` | 用户提交封禁申诉 |
| POST | `appeal-public` | `/api/users/appeal-public` | 公开申诉（封禁用户免登录） |
| PUT | `update` / `restore` | `/api/users/{method}` | 通用 |
| PUT | `email` | `/api/users/email` | 修改绑定邮箱（验证码） |
| PUT | `phone` | `/api/users/phone` | 修改绑定手机号（验证码） |
| PUT | `status` | `/api/users/status` | 修改用户状态（冻结/解冻，管理员） |
| PUT | `ban` | `/api/users/ban` | 封禁用户（管理员） |
| PUT | `unban` | `/api/users/unban` | 解封用户（管理员） |
| PUT | `appeal-handle` | `/api/users/appeal-handle` | 处理申诉（通过/驳回，管理员） |
| DELETE | `remove` / `delete` / `clear` | `/api/users/{method}` | 通用 |
| DELETE | `destroy` | `/api/users/destroy` | 注销账户（验证码 + 密码） |

### 8. oauth 第三方登录控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `qq` | `/api/oauth/qq` | QQ 登录 |
| GET | `github` | `/api/oauth/github` | GitHub 登录 |
| POST | `qq` / `github` | `/api/oauth/{method}` | 登录回调处理 |
| PUT | `qq` | `/api/oauth/qq` | 绑定/更新 |
| DELETE | `qq` | `/api/oauth/qq` | 解绑 |

### 9. links 友情链接控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/links/{method}` | 通用 |
| POST | `save` / `create` | `/api/links/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/links/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/links/{method}` | 通用 |

### 10. proxy 反向代理控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| 任意 | —（INDEX） | `/api/proxy` | 反向代理转发，通过 `i-url` 指定目标地址、`i-type` 指定返回类型 |

> 该控制器无 `method` 子方法，通过 `INDEX` 直接转发请求。

### 11. level 等级控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/level/{method}` | 通用 |
| POST | `save` / `create` | `/api/level/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/level/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/level/{method}` | 通用 |

### 12. banner 轮播图控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/banner/{method}` | 通用 |
| POST | `save` / `create` | `/api/banner/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/banner/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/banner/{method}` | 通用 |

### 13. config 系统配置控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `count` / `column` | `/api/config/{method}` | 通用 |
| POST | `save` / `create` | `/api/config/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/config/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/config/{method}` | 通用 |

### 14. article 文章控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/article/{method}` | 通用 |
| POST | `save` / `create` | `/api/article/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/article/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/article/{method}` | 通用 |

### 15. comment 评论控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/comment/{method}` | 通用 |
| GET | `flat` | `/api/comment/flat` | 扁平评论列表 |
| POST | `save` / `create` | `/api/comment/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/comment/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/comment/{method}` | 通用 |

### 16. placard 公告控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/placard/{method}` | 通用 |
| POST | `save` / `create` | `/api/placard/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/placard/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/placard/{method}` | 通用 |

### 17. api-keys API 密钥控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/api-keys/{method}` | 通用 |
| POST | `save` / `create` | `/api/api-keys/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/api-keys/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/api-keys/{method}` | 通用 |

### 18. ip-black IP 黑名单控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/ip-black/{method}` | 通用 |
| POST | `save` / `create` | `/api/ip-black/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/ip-black/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/ip-black/{method}` | 通用 |

### 19. ip-white IP 白名单控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/ip-white/{method}` | 通用 |
| POST | `save` / `create` | `/api/ip-white/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/ip-white/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/ip-white/{method}` | 通用 |

### 20. qps-warn QPS 告警控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/qps-warn/{method}` | 通用 |
| POST | `save` / `create` | `/api/qps-warn/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/qps-warn/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/qps-warn/{method}` | 通用 |

### 21. auth-group 权限组控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/auth-group/{method}` | 通用 |
| POST | `save` / `create` | `/api/auth-group/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/auth-group/{method}` | 通用 |
| PUT | `uids` | `/api/auth-group/uids` | 设置权限组成员 |
| DELETE | `remove` / `delete` / `clear` | `/api/auth-group/{method}` | 通用 |

### 22. auth-pages 权限页面控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/auth-pages/{method}` | 通用 |
| POST | `save` / `create` | `/api/auth-pages/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/auth-pages/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/auth-pages/{method}` | 通用 |

### 23. auth-rules 权限规则控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/auth-rules/{method}` | 通用 |
| POST | `save` / `create` | `/api/auth-rules/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/auth-rules/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/auth-rules/{method}` | 通用 |

### 24. links-group 友链分组控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/links-group/{method}` | 通用 |
| POST | `save` / `create` | `/api/links-group/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/links-group/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/links-group/{method}` | 通用 |

### 25. article-group 文章分组控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/article-group/{method}` | 通用 |
| GET | `tree` | `/api/article-group/tree` | 分组树形结构 |
| POST | `save` / `create` | `/api/article-group/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/article-group/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/article-group/{method}` | 通用 |

### 26. search 搜索控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `article` | `/api/search/article` | 文章搜索 |
| GET | `pages` | `/api/search/pages` | 页面搜索 |
| GET | `tags` | `/api/search/tags` | 标签搜索 |
| GET | `users` | `/api/search/users` | 用户搜索 |
| GET | `links` | `/api/search/links` | 友链搜索 |
| GET | `moments` | `/api/search/moments` | 动态搜索 |
| GET | `all` | `/api/search/all` | 全局搜索 |

### 27. rss RSS 控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| 任意 | —（INDEX） | `/api/rss` | 输出 RSS 2.0 订阅（XML），无子方法 |

> 该控制器不支持 `method` 子方法，直接访问 `/api/rss` 输出文章订阅源。

### 28. moments 动态控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/moments/{method}` | 通用 |
| GET | `comment` | `/api/moments/comment` | 动态评论列表 |
| GET | `comment_count` | `/api/moments/comment_count` | 动态评论数 |
| POST | `save` / `create` | `/api/moments/{method}` | 通用 |
| PUT | `update` / `restore` | `/api/moments/{method}` | 通用 |
| PUT | `set_top` | `/api/moments/set_top` | 设置动态置顶 |
| DELETE | `remove` / `delete` / `clear` | `/api/moments/{method}` | 通用 |

### 29. attachment 附件控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/attachment/{method}` | 通用 |
| GET | `list` | `/api/attachment/list` | 附件列表 |
| GET | `emoji` | `/api/attachment/emoji` | 获取表情列表（扫描 emoji 目录） |
| POST | `save` / `create` | `/api/attachment/{method}` | 通用 |
| POST | `batch` | `/api/attachment/batch` | 批量上传 |
| POST | `checktype` | `/api/attachment/checktype` | 检查文件类型 |
| PUT | `update` / `restore` | `/api/attachment/{method}` | 通用 |
| DELETE | `remove` / `delete` / `clear` | `/api/attachment/{method}` | 通用 |

### 30. user-likes 点赞控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/user-likes/{method}` | 通用 |
| GET | `is-liked` | `/api/user-likes/is-liked` | 是否已点赞 |
| GET | `likes` | `/api/user-likes/likes` | 点赞列表 |
| GET | `counts` | `/api/user-likes/counts` | 点赞计数 |
| POST | `save` / `create` | `/api/user-likes/{method}` | 通用 |
| POST | `like` | `/api/user-likes/like` | 点赞 |
| POST | `unlike` | `/api/user-likes/unlike` | 取消点赞 |
| PUT | `update` | `/api/user-likes/update` | 通用 |
| PUT | `unlike` | `/api/user-likes/unlike` | 取消点赞 |
| DELETE | `remove` / `delete` / `clear` | `/api/user-likes/{method}` | 通用 |

### 31. user-collects 收藏控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/user-collects/{method}` | 通用 |
| GET | `is-collected` | `/api/user-collects/is-collected` | 是否已收藏 |
| GET | `collects` | `/api/user-collects/collects` | 收藏列表 |
| GET | `counts` | `/api/user-collects/counts` | 收藏计数 |
| POST | `save` / `create` | `/api/user-collects/{method}` | 通用 |
| POST | `collect` | `/api/user-collects/collect` | 收藏 |
| POST | `uncollect` | `/api/user-collects/uncollect` | 取消收藏 |
| PUT | `update` | `/api/user-collects/update` | 通用 |
| PUT | `uncollect` | `/api/user-collects/uncollect` | 取消收藏 |
| DELETE | `remove` / `delete` / `clear` | `/api/user-collects/{method}` | 通用 |

### 32. user-follows 关注控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/user-follows/{method}` | 通用 |
| GET | `following` | `/api/user-follows/following` | 我关注的人列表 |
| GET | `followers` | `/api/user-follows/followers` | 关注我的人（粉丝）列表 |
| GET | `is-following` | `/api/user-follows/is-following` | 是否已关注 |
| GET | `counts` | `/api/user-follows/counts` | 关注/粉丝计数 |
| POST | `save` / `create` | `/api/user-follows/{method}` | 通用 |
| POST | `follow` | `/api/user-follows/follow` | 关注 |
| POST | `unfollow` | `/api/user-follows/unfollow` | 取消关注 |
| PUT | `update` / `restore` | `/api/user-follows/{method}` | 通用 |
| PUT | `unfollow` | `/api/user-follows/unfollow` | 取消关注 |
| DELETE | `remove` / `delete` / `clear` | `/api/user-follows/{method}` | 通用 |

### 33. notification 通知控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `one` / `all` / `sum` / `min` / `max` / `rand` / `count` / `column` | `/api/notification/{method}` | 通用 |
| GET | `unread-count` | `/api/notification/unread-count` | 未读数量 |
| GET | `list` | `/api/notification/list` | 通知列表 |
| POST | `save` / `create` | `/api/notification/{method}` | 通用 |
| POST | `send-system` | `/api/notification/send-system` | 发送系统通知（管理员） |
| PUT | `update` / `restore` | `/api/notification/{method}` | 通用 |
| PUT | `read` | `/api/notification/read` | 标记单条已读 |
| PUT | `read-all` | `/api/notification/read-all` | 全部标记已读 |
| PUT | `read-batch` | `/api/notification/read-batch` | 批量标记已读 |
| DELETE | `remove` / `delete` / `clear` | `/api/notification/{method}` | 通用 |
| DELETE | `remove-all` | `/api/notification/remove-all` | 清空全部通知 |

#### 通知类型（type 字段）

通知按 `type` 字段区分消息类型，目前项目实际支持以下取值（`varchar(32)`，可扩展）：

| type 值 | 含义 | 触发场景 | 触发位置 |
| :--- | :--- | :--- | :--- |
| `system` | 系统通知 | 管理员后台发送系统通知；用户登录后的「账号登录通知」 | `notification.go` `sendSystem` / `model/notification.go` `CreateLoginNotification` |
| `comment` | 评论/回复通知 | 有人回复了你的评论 | `comment.go` `create` |
| `like` | 点赞通知 | 有人赞了你的评论 | `user-likes.go` `like` |
| `follow` | 关注通知 | 有人关注了你 | `user-follows.go` `follow` |
| `collect` | 收藏通知 | 有人收藏了你的文章/页面/动态 | `user-collects.go` `collect` |

> 补充：`model/notification.go` 中 `Type` 字段注释写的是 `comment/like/follow/system`，实际代码还多了 `collect`（收藏通知），注释未同步。`notification.go` 控制器自身仅硬编码 `system` 一种类型，其余类型由评论/点赞/关注/收藏等控制器在业务动作中调用 `CreateNotification()` 传入。

#### 广播 vs 个人通知（uid 字段）

| 类型 | 判定 | 说明 |
| :--- | :--- | :--- |
| 广播通知 | `uid = 0` | 只存一条记录，全员可见；每个用户的已读/隐藏状态记录在 `inis_notification_read` 表 |
| 个人通知 | `uid = 具体用户` | 每个用户一条独立记录 |

`send-system` 接口的 `target_type` 参数支持三种目标：

| target_type | 说明 |
| :--- | :--- |
| `all` | 全量广播（仅创建一条 `uid=0` 记录，不逐用户写入，不发邮件） |
| `partial` | 部分用户（需传 `user_ids`） |
| `single` | 单个用户（需传 `user_ids`） |

### 34. integral 积分控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `status` | `/api/integral/status` | 查询当前用户积分余额（登录） |
| GET | `all` | `/api/integral/all` | 积分流水列表（登录，仅自己的） |
| GET | `rules` | `/api/integral/rules` | 积分任务规则（公开） |
| POST | `give` | `/api/integral/give` | 调整用户积分（管理员） |

### 35. goods 商品控制器

| HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `all` | `/api/goods/all` | 商品列表（公开，仅上架；管理员可查全部） |
| GET | `one` | `/api/goods/one` | 商品详情（公开） |
| GET | `count` | `/api/goods/count` | 商品数量 |
| GET | `orders` | `/api/goods/orders` | 我的订单（登录，仅自己的） |
| GET | `orders-all` | `/api/goods/orders-all` | 全部订单（管理员） |
| POST | `buy` | `/api/goods/buy` | 购买商品（登录） |
| POST | `save` / `create` | `/api/goods/{method}` | 商品保存/创建（管理员） |
| PUT | `update` / `restore` | `/api/goods/{method}` | 商品更新/恢复（管理员） |
| PUT | `order-status` | `/api/goods/order-status` | 更新订单状态（管理员） |
| DELETE | `remove` / `delete` / `clear` | `/api/goods/{method}` | 商品删除（管理员） |

---

## 八、其他路由（dev / socket / index）

### 1. /dev 开发与安装接口（前缀 `/dev/`）

| 控制器 | HTTP | method | 完整路径 | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| info | GET | `time` | `/dev/info/time` | 时间信息 |
| info | GET | `system` | `/dev/info/system` | 系统信息 |
| info | GET | `device` | `/dev/info/device` | 设备信息 |
| info | GET | `version` | `/dev/info/version` | 版本信息 |
| info | GET | `renew` | `/dev/info/renew` | 更新 |
| info | GET | `kill` | `/dev/info/kill` | 杀死进程 |
| install | GET | `check` | `/dev/install/check` | 安装锁状态 |
| install | POST | `lock` | `/dev/install/lock` | 上锁（安装锁） |
| install | POST | `init-db` | `/dev/install/init-db` | 初始化数据库 |
| install | POST | `connect-db` | `/dev/install/connect-db` | 连接数据库 |
| install | POST | `create-admin` | `/dev/install/create-admin` | 创建管理员 |

### 2. /socket WebSocket 接口

| 路径 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/socket` | GET | 建立 WebSocket 连接（需 JWT 鉴权），支持广播/单播/私聊/状态推送/消息回执 |

### 3. 首页与模板接口

| 路径 | 方法 | 说明 |
| :--- | :--- | :--- |
| `/` | GET | 首页（渲染 `public/index.html`） |
| `/api/template-status` | GET | 模板状态实时检测 |

---

## 九、通用查询参数

所有通用查询类方法（`one` / `all` / `rand` / `count` / `sum` / `min` / `max` / `column` 等）支持以下公共参数：

| 参数名 | 类型 | 说明 |
| :--- | :--- | :--- |
| `where` | string | 查询条件（等值） |
| `or` | string | 或条件 |
| `like` | string | 模糊查询 |
| `not` | string | 排除条件 |
| `null` | string | 字段为空 |
| `notNull` | string | 字段非空 |
| `ids` | string | ID 列表（逗号/竖线分隔） |
| `field` | string | 返回字段过滤（逗号分隔） |
| `page` | int | 页码（`all` 默认 1） |
| `limit` | int | 每页数量（受系统 `SYSTEM_PAGE_LIMIT` 配置限制） |
| `order` | string | 排序方式（如 `create_time desc`） |
| `onlyTrashed` | bool | 仅查询回收站数据 |
| `withTrashed` | bool | 包含回收站数据 |
| `cache` | string | 是否启用缓存（默认 `true`，受系统缓存开关控制） |

写操作通用参数：

| 参数名 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 更新/保存时的记录 ID |
| `ids` | string | 删除/恢复时的 ID 列表 |

---

## 附：快速索引

- 认证相关：`/api/comm/*`、`/api/oauth/*`
- 内容相关：`/api/article/*`、`/api/comment/*`、`/api/moments/*`、`/api/placard/*`、`/api/pages/*`、`/api/tags/*`、`/api/search/*`、`/api/rss`
- 用户相关：`/api/users/*`、`/api/level/*`、`/api/exp/*`、`/api/user-likes/*`、`/api/user-collects/*`、`/api/user-follows/*`、`/api/notification/*`、`/api/integral/*`
- 积分商城：`/api/goods/*`
- 内容组织：`/api/article-group/*`、`/api/links/*`、`/api/links-group/*`
- 系统管理：`/api/config/*`、`/api/toml/*`、`/api/api-keys/*`、`/api/attachment/*`、`/api/banner/*`
- 权限与安全：`/api/auth-group/*`、`/api/auth-pages/*`、`/api/auth-rules/*`、`/api/ip-black/*`、`/api/ip-white/*`、`/api/qps-warn/*`
- 开发/安装：`/dev/info/*`、`/dev/install/*`
- 其他：`/api/proxy`、`/api/test/*`、`/socket`
