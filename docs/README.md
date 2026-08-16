# inis 文档中心（inis-docs）

> 欢迎使用 inis 文档中心。本文档是 inis 内容管理系统（CMS）的官方文档总入口，涵盖快速上手、系统架构、API 接口、二次开发、前端主题开发、数据库与缓存优化等全部内容。

inis 是一款基于 Go 语言开发的高性能内容管理系统（CMS），基于 Gin 框架二次开发，采用 GORM 作为数据库 ORM 工具。系统以「轻量核心、高效响应、灵活扩展」为核心定位，为开发者提供易上手、具备良好扩展基础的 CMS 解决方案。

---

## 文档导航

### 快速开始

| 文档 | 说明 |
| :--- | :--- |
| [README](../README.md) | 项目总览：核心特性、快速开始、打包教程、系统架构、技术栈、目录结构、常见问题 |
| 本页 | 文档中心总索引，快速定位你需要的文档 |

### API 接口文档

| 文档 | 说明 |
| :--- | :--- |
| [api汇总.md](./api汇总.md) | **全部接口汇总**：35 个控制器、中间件、通用 CRUD 方法、通用查询参数、快速索引（强烈推荐先看这篇） |
| [API docs/](./API%20docs/) | 各模块接口详细文档目录（36 篇） |

#### API 详细文档（API docs 目录）

**内容模块**

| 文档 | 说明 |
| :--- | :--- |
| [article.md](./API%20docs/article.md) | 文章接口（CRUD、浏览量、经验值、草稿、审核） |
| [article-group.md](./API%20docs/article-group.md) | 文章分组接口（多级分类、树形结构） |
| [comment.md](./API%20docs/comment.md) | 评论接口（CRUD、扁平列表） |
| [moments.md](./API%20docs/moments.md) | 动态接口（CRUD、评论、置顶） |
| [pages.md](./API%20docs/pages.md) | 独立页面接口 |
| [tags.md](./API%20docs/tags.md) | 标签接口 |
| [placard.md](./API%20docs/placard.md) | 公告接口 |
| [banner.md](./API%20docs/banner.md) | 轮播图接口 |
| [attachment.md](./API%20docs/attachment.md) | 附件/文件上传接口 |
| [file.md](./API%20docs/file.md) | 文件接口 |
| [search.md](./API%20docs/search.md) | 搜索接口（全站/分类搜索） |
| [rss.md](./API%20docs/rss.md) | RSS 订阅接口 |

**用户模块**

| 文档 | 说明 |
| :--- | :--- |
| [users.md](./API%20docs/users.md) | 用户接口（CRUD、封禁、申诉、注销） |
| [comm.md](./API%20docs/comm.md) | 公共/认证接口（登录、注册、Token） |
| [oauth.md](./API%20docs/oauth.md) | 第三方登录接口（QQ / GitHub） |
| [level.md](./API%20docs/level.md) | 用户等级接口 |
| [exp.md](./API%20docs/exp.md) | 经验值接口（签到、活跃榜） |
| [user-likes.md](./API%20docs/user-likes.md) | 点赞接口 |
| [user-collects.md](./API%20docs/user-collects.md) | 收藏接口 |
| [user-follows.md](./API%20docs/user-follows.md) | 关注接口 |
| [user-interaction-guide.md](./API%20docs/user-interaction-guide.md) | 用户互动模块前端调用指南 |

**系统管理模块**

| 文档 | 说明 |
| :--- | :--- |
| [config.md](./API%20docs/config.md) | 系统配置接口 |
| [toml.md](./API%20docs/toml.md) | TOML 配置接口（短信/缓存/存储/加密） |
| [api-keys.md](./API%20docs/api-keys.md) | API 密钥接口 |
| [links.md](./API%20docs/links.md) | 友情链接接口 |
| [links-group.md](./API%20docs/links-group.md) | 友链分组接口 |
| [notification.md](./API%20docs/notification.md) | 消息通知接口 |

**权限与安全模块**

| 文档 | 说明 |
| :--- | :--- |
| [auth-group.md](./API%20docs/auth-group.md) | 权限组接口 |
| [auth-pages.md](./API%20docs/auth-pages.md) | 权限页面接口 |
| [auth-rules.md](./API%20docs/auth-rules.md) | 权限规则接口 |
| [ip-black.md](./API%20docs/ip-black.md) | IP 黑名单接口 |
| [ip-white.md](./API%20docs/ip-white.md) | IP 白名单接口 |
| [qps-warn.md](./API%20docs/qps-warn.md) | QPS 预警接口 |

**其他**

| 文档 | 说明 |
| :--- | :--- |
| [base.md](./API%20docs/base.md) | 基础控制器公共能力说明 |
| [proxy.md](./API%20docs/proxy.md) | 反向代理接口 |
| [test.md](./API%20docs/test.md) | 测试接口 |

### 开发指南

| 文档 | 说明 |
| :--- | :--- |
| [二次开发规范.md](./二次开发规范.md) | 后端二次开发规范：RESTful 规范、参数传递、权限规则注册、缓存/通知/搜索系统、异常处理、安全规范 |
| [前端主题开发及API调用规范.md](./前端主题开发及API调用规范.md) | 前端主题开发与 API 调用规范 |
| [socket前端使用指南.md](./socket前端使用指南.md) | WebSocket 前端使用指南（实时通信） |
| [消息通知api使用文档.md](./消息通知api使用文档.md) | 消息通知 API 使用文档 |
| [友链api使用文档.md](./友链api使用文档.md) | 友链 API 使用文档 |

### 数据库与缓存

| 文档 | 说明 |
| :--- | :--- |
| [cache.md](./cache.md) | 缓存系统开发者文档（Redis/文件/内存三种驱动、标签清理、命中统计） |
| [database-index.md](./database-index.md) | 数据库索引优化建议（12 张核心表索引方案、慢查询分析） |

### 项目规划

| 文档 | 说明 |
| :--- | :--- |
| [项目规划.md](./项目规划.md) | 逻辑梳理（时间字段）、待开发功能、当前系统存在的问题 |
| [项目后期发展规划.md](./项目后期发展规划.md) | 项目后期发展规划 |

---

## 快速上手

### 运行环境要求

- Go 1.24.0+（详见 `go.mod`）
- MySQL（当前默认数据库）

### 三步启动

```bash
# 1. 克隆项目
git clone https://github.com/zhu885744/inisv1.git
cd inisv1

# 2. 安装依赖
go mod tidy

# 3. 运行
go run main.go
```

启动后访问 `http://localhost:8642`，系统会进入图形化安装引导，按提示完成数据库配置即可。

> - 默认管理员账号：`admin`
> - 默认管理员密码：`admin123456`

### 打包

```bash
# Windows
go build -o inis.exe main.go

# Linux
go build -o inis main.go
chmod +x inis
```

也可使用项目根目录的 `build.bat` 脚本按提示选择平台编译。

---

## 系统架构速览

### 技术栈

| 分类 | 技术 |
| :--- | :--- |
| 语言 | Go 1.25.0 |
| Web 框架 | Gin 1.12.0 |
| ORM | GORM 1.31.2 |
| 数据库 | MySQL / PostgreSQL（默认 MySQL） |
| 缓存 | Redis / BigCache / 文件缓存 |
| 认证 | JWT（golang-jwt/jwt v5） |
| 安全 | bluemonday（XSS）、unrolled/secure（安全头） |
| 实时通信 | gorilla/websocket |
| 定时任务 | gocron |
| 配置管理 | Viper（TOML） |
| 日志 | Zap + lumberjack |

### 架构分层

```
客户端层 → 表现层（Route/Controller/Middleware）→ 业务层（Facade/Service/Validator）
        → 数据层（Model/GORM → MySQL/Redis）→ 基础设施层（云存储/短信/定时任务/日志）
```

### 核心功能

- **配置管理系统**：动态配置存储与缓存
- **多语言国际化**：中、英、日、韩、俄等语言包
- **安全防护机制**：安装锁（install.lock）、API 签名、QPS 限流、SQL 注入防护、XSS 防护、CSRF 防护
- **媒体资源处理**：图片压缩、格式转换、多存储模式
- **高效缓存策略**：内存缓存 + 标签批量清理
- **用户权限系统**：基于 RBAC 的权限管理
- **文章管理系统**：完整 CRUD、分类、标签、审核、置顶
- **评论系统**：评论、审核、回复
- **互动系统**：点赞、收藏、关注、分享
- **消息通知系统**：评论/点赞/关注/收藏通知，WebSocket 实时推送
- **社交登录**：QQ、GitHub 第三方登录

---

## 接口规范速查

### 统一响应结构

```json
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {}
}
```

### 路由规则

所有业务接口统一通过 `/api/{控制器}/{方法}` 访问：

```
GET    /api/article/all     # 获取文章列表
POST   /api/comm/login      # 用户登录
PUT    /api/users/update    # 更新用户
DELETE /api/article/remove  # 软删除文章
```

### 通用 CRUD 方法（15 个）

| HTTP | method | 说明 |
| :--- | :--- | :--- |
| GET | `one` / `all` / `rand` / `count` / `sum` / `min` / `max` / `column` | 查询类 |
| POST | `save` / `create` | 新增类 |
| PUT | `update` / `restore` | 更新/恢复类 |
| DELETE | `remove` / `delete` / `clear` | 删除类 |

> 完整规范请参阅 [api汇总.md](./api汇总.md)。

---

## 目录结构

```
inisv1/
├── main.go                 # 程序入口
├── admin/                  # 管理后台
├── themes/                 # 主题目录
├── public/                 # 静态资源
├── config/                 # 配置文件目录
├── docs/                   # 文档目录（本目录）
├── app/
│   ├── api/                # API 接口（controller / middleware / route）
│   ├── dev/                # 开发功能（系统信息、安装引导）
│   ├── facade/             # 门面层（缓存/数据库/日志/短信/存储等）
│   ├── index/              # 首页路由/控制器
│   ├── middleware/         # 全局中间件
│   ├── model/              # 数据模型
│   ├── socket/             # WebSocket
│   ├── timer/              # 定时任务
│   └── validator/          # 数据验证器
```

---

## 常见问题

| 问题 | 解决方案 |
| :--- | :--- |
| 端口被占用 | 修改 `config/app.toml` 中的端口配置 |
| 数据库连接失败 | 检查数据库连接信息和权限 |
| 404 错误 | 确保主题文件已部署到 `public` 目录 |
| 502 错误 | 检查应用运行状态和 Nginx 配置 |
| 如何修改默认端口 | 修改 `config/app.toml` |
| 如何启用缓存 | 配置 `config/cache.toml`（file/ram/redis） |

---

## 贡献与许可

- 欢迎提交 Issue 和 Pull Request
- 许可证：[Apache-2.0](../LICENSE)
- 交流群：119300889
- 邮箱：xz@zhuxu.asia

> 致谢原作者「陈兔子」及原开源仓库 [inis-io/inis](https://github.com/inis-io/inis)。
