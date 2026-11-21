# INIS-cms

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/zhu885744/INIS-cms)](https://github.com/zhu885744/INIS-cms/releases)
[![Go](https://img.shields.io/badge/Go-1.27.4-00ADD8)](go.mod)
[![Vue](https://img.shields.io/badge/Vue-3.x-42b883)](Mellow/package.json)

INIS-cms 是一套 **Go 后端 + Vue 主题/后台 + uni-app x 移动端** 的全栈内容社区系统。

后端基于 Gin + GORM 二次开发，架构参考 ThinkPHP 6 的简洁分层理念（controller / model / validator / middleware / facade），
以「轻量核心、单文件部署、易于二次开发」为定位；默认主题 **Mellow** 提供文章、动态、话题、文档、群聊等完整社区界面，以及完整的管理后台。

构建时可选把前端产物 `go:embed` 打进二进制，**部署只需要「一个可执行文件」**。

---

## 核心特性

### 内容

- **文章**：富文本 / Markdown 双编辑器，封面、分组树、置顶、回收站、审核流转
- **动态**：图文 + 实况图（图片 + 视频），与文章在首页按时间混排
- **话题（Hashtag）**：正文 `#话题#` 自动规范化成链接，聚合页 / 热度榜 / 关注 / 主持人治理（置顶、移出、删除）
- **文档站**：多项目、多版本、分组树，独立布局
- **独立页面 / 公告 / 轮播 / 友链 / RSS**
- **可见范围**：公开 / 登录可见 / 积分兑换可见，列表与详情统一脱敏

### 社区

- **评论**：楼中楼、审核、批量审核、按内容类型统一口径
- **互动**：点赞、收藏、关注、拉黑、@提及（含候选浮层与通知）
- **私信**：会话置顶、免打扰、正在输入、撤回、已读
- **群聊**：公共大厅 + 自建群，群角色、入群审批、邀请码 / 二维码、在线成员、禁言
- **举报**：前端提交 / 撤回，后台处理与留证
- **实时通道**：自研 WebSocket Hub（`/socket`），支持多实例 Redis 广播

### 激励

- 经验值、积分、每日签到、等级、勋章（含佩戴与续期）、装扮（头衔 / 头像框）、积分商城、积分打赏、卡密

### 运营与治理

- 内容审核（文章 / 动态 / 页面 / 文档 / 评论 / 群聊）、封禁与申诉、IP 黑白名单、QPS 限流与自动封禁、接口密钥、隐私分级、数据统计与资源告警

### 工程

- **单文件部署**：`-tags embed` 内嵌主题与安装向导，首次启动自动生成 `config/`、`runtime/`、`storage/`
- **多平台发布**：GoReleaser 一次产出 Linux / Windows / macOS × amd64 / arm64 六个平台包
- **国际化**：内置 zh-cn / en-us / ko-kr / ru-ru 语言包，单二进制部署也会自动释放
- **发布前漏洞阻断**：CI 内 `govulncheck` 命中漏洞即终止发布

---

## 技术栈

> 版本以 `go.mod` / `Mellow/package.json` 为准，升级依赖后请同步本表。

### 后端（Go）

| 分类 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 语言 | Go | 1.27.4 | 后端服务主语言 |
| Web 框架 | Gin | v1.12.0 | HTTP 路由与中间件 |
| ORM | GORM | v1.31.2 | 参数化查询 + `soft_delete` 回收站 |
| 数据库 | MySQL | 5.7+ / 8.x | 驱动 `go-sql-driver/mysql` v1.10.0 |
| 缓存 | Redis / BigCache / 本地 / 文件 | go-redis v9.21.0、BigCache v3.1.0 | 四驱动，`cache.toml` 切换 |
| 鉴权 | 服务端会话 | - | Cookie / `Authorization`，权限规则走 `auth_rules` |
| WebSocket | gorilla/websocket | v1.5.3 | `/socket` 实时通道 |
| HTTP/3 | quic-go（间接） | v0.61.0 | Gin 的 QUIC 支持 |
| 安全 | bluemonday / unrolled/secure | v1.0.27 / v1.17.0 | XSS 过滤、安全响应头 |
| 定时任务 | gocron | v0.0.1 | 封禁解封、装扮到期、日志清理等 |
| 配置 | Viper + TOML + fsnotify | v1.21.0 / v1.10.1 | 配置热更新 |
| 日志 | Zap + lumberjack | v1.28.0 / v2.0.0 | 结构化日志与轮转 |
| 存储 | 本地 / 腾讯云 COS | cos-go-sdk-v5 v0.7.75 | 上传与附件 |
| 短信 | 阿里云 / 腾讯云 | darabonba v2.2.4、tencentcloud v1.3.146 | 验证码、号码认证 |
| 邮件 | gomail | v2.0.0 | SMTP + 发送队列 |
| 其他 | validator / imaging / gopsutil / go-utils | v10.30.3 / v1.6.2 / v3.21.11 / v1.3.6 | 参数校验、图片处理、系统信息、通用工具 |

### 前台主题 + 管理后台（Mellow）

| 分类 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 框架 | Vue | ^3.4 | `<script setup>` 组合式 API |
| 构建 | Vite | ^8.3 | 构建产物直接内嵌进 Go 二进制 |
| 状态 | Pinia | ^2.1 | user / site / notification 等 store |
| 路由 | Vue Router | ^4.3 | SPA，主题路由由后端 `NoRoute` 回退 |
| 渲染 | markdown-it + highlight.js | ^14 / ^11 | 文档与文章正文前端渲染 |
| 网络 | axios | ^1.21 | 统一 `call()` 封装、拦截器提示 |
| UI | 自定义设计令牌 + bootstrap-icons | ^1.13 | 朱砂 / 纸感主题，颜色由后台配置驱动 |
| 其他 | qrcode / dayjs / crypto-js | - | 邀请二维码、时间格式化、前端加密 |

### 移动端（inis-app）

| 分类 | 技术 | 说明 |
|------|------|------|
| 框架 | uni-app x（`.uvue` + TypeScript） | Vue 3 语法，编译到 Android / iOS / 小程序 |
| 配置 | `inis-app/config/env.ts` | `API_DOMAIN`、`API_PREFIX`、`APP_SOURCE` |

### 构建与发布

| 分类 | 技术 | 说明 |
|------|------|------|
| 打包 | GoReleaser v2 | 6 平台交叉编译 + 内嵌前端 + UPX 压缩 + checksum + changelog |
| CI | GitHub Actions | 推 `v*` tag 自动发布；发布前跑 `govulncheck` 阻断有漏洞的版本 |
| 本地脚本 | `build.bat` / `build.sh` | 交互式单平台 / 批量编译，可选 UPX 压缩 |

---

## 快速开始

### 环境要求

- [Go](https://golang.org/dl/) ≥ 1.27.4
- [Node.js](https://nodejs.org/) ≥ 20.19（或 22.12+）与 npm（仅构建 Mellow 时需要）
- MySQL 5.7+ / 8.x
- 移动端需要 HBuilderX（uni-app x 项目）

### 1. 克隆并安装依赖

```bash
git clone git@github.com:zhu885744/INIS-cms.git
cd INIS-cms
go mod tidy
```

### 2. 运行后端

```bash
go run main.go
```

访问 <http://localhost:8642>：未安装时首页会自动 302 到图形化安装向导（`/install`）。

> 安装流程：检查环境 → 填写数据库 → 初始化数据 → 完成安装（自动解除安装锁）
> 默认管理员账号：`admin` / `admin123456`（首次登录后请立即修改）

### 3. 开发前台主题

```bash
cd Mellow
npm install
npm run dev      # Vite 开发服务器，接口代理见 vite.config.js
```

生产构建产物默认在 `Mellow/dist`，需要内嵌时由构建脚本同步到 `theme/dist`。

### 4. 移动端

用 HBuilderX 打开 `inis-app` 目录，先把 `inis-app/config/env.ts` 的 `API_DOMAIN` 改成你的后端地址
（App 端必须写完整域名并把来源加入 CORS 白名单），然后运行到 Android / iOS / 小程序模拟器或真机。

---

## 构建与发布

### 本地打包脚本

**Windows（`build.bat`，推荐）**

1. 双击 `build.bat`
2. 选择平台（单平台或「所有平台」）
3. 询问「是否把前端构建产物打包进二进制」时选 `y`：
   脚本会自动 `npm install`（首次）→ `npm run build` → 同步到 `theme/dist` → 带 `-tags embed` 编译
4. 产物输出到 `dist/`；若装了 UPX 会询问压缩等级

**Linux / macOS（`build.sh`）**

```bash
bash build.sh                    # 当前系统/架构，内嵌前端
bash build.sh linux amd64        # 交叉编译 Linux x86_64
bash build.sh linux arm64        # 交叉编译 Linux ARM64（鲲鹏 / 树莓派）
bash build.sh linux amd64 skip   # 跳过前端构建（不内嵌主题）
```

### GoReleaser（正式发版）

配置见 [`.goreleaser.yaml`](.goreleaser.yaml)，产物命名 `inis_<version>_<os>_<arch>.tar.gz`（Windows 为 `.zip`），
另附 `checksums.txt`；压缩包内含 `LICENSE` / `README.md` / `CHANGELOG.md`。

Linux / Windows 的二进制会用 **UPX（`-9` + LZMA）** 再压一道：内嵌完整前端时约 42 MB → 12 MB
macOS 因签名问题跳过（与 `build.bat` 的处理一致）。构建机需要预装 `upx`，
CI 里由 `apt-get install upx-ucl` 提供。

```bash
# 本地试跑（不发布，版本为「下一个补丁号-next」）
goreleaser release --snapshot --clean

# 正式发布：打 tag 并推送，GitHub Actions 自动执行
git tag v1.0.8
git push origin v1.0.8
```

CI 流程（`.github/workflows/release.yml`）：

```
checkout → setup-go(1.27.4) → setup-node(22) → govulncheck → goreleaser release --clean
```

任一环节失败即不产出 Release；`govulncheck` 命中漏洞会直接终止（避免带漏洞的版本被发布）。

### 单文件部署

内嵌构建出的可执行文件**自带主题与安装向导**，部署只需要：

```
inis_linux_amd64     # 二进制（内含 Mellow 前端产物 + 安装向导 + 语言包）
```

首次启动会自动创建 `config/`、`config/i18n/`、`runtime/`、`storage/`，并释放内置语言包；
随后访问首页跳转 `/install` 完成安装即可。

### 手动编译

```bash
# Windows（无控制台窗口的后台版本）
go build -o inis.exe main.go
go build -ldflags -H=windowsgui -o inis.exe main.go

# Linux
go build -o inis main.go
chmod +x inis

# 内嵌主题（单文件部署）
cd Mellow && npm run build && cd ..
rm -rf theme/dist && mkdir -p theme/dist && cp -R Mellow/dist/. theme/dist/
go build -tags embed -o inis main.go
```

---

## 配置说明

配置文件位于 `config/`，首次启动按模板自动生成（`database.toml` 由安装向导写入）：

| 文件 | 说明 |
|------|------|
| `app.toml` | 端口（默认 8642）、调试开关、token、socket、通知等 |
| `cache.toml` | 缓存驱动（local / file / redis）与过期时间 |
| `crypt.toml` | 登录会话配置 |
| `log.toml` | 日志级别、保留天数、轮转策略 |
| `sms.toml` | 短信 / 邮件驱动（阿里云、腾讯云、SMTP）与发件队列 |
| `storage.toml` | 存储驱动（本地 / 腾讯云 COS）、上传限制与命名规则 |
| `database.toml` | 数据库连接（安装向导写入） |

> `config/*.toml` 含数据库密码、短信密钥等敏感信息，已在 `.gitignore` 排除，请勿提交。

### 版本号

后端版本定义在 `app/facade/const.go` 的 `Version`（当前 `1.0.8`）。
该变量是 `var`，正式发布时由 GoReleaser 通过 `-ldflags "-X inis/app/facade.Version=<tag>"` 注入，无需手改。

设备签名盐值 `DefaultToken` 可用环境变量 `INIS_DEVICE_TOKEN` 覆盖。

### 文档

- 在线接口文档：<https://zhuxu.asia/docs/inisdocs>
- 仓库内文档更详细：`docs/API docs/`（按模块划分）、`docs/功能清单.md`、`docs/socket.md`、`docs/topics.md`、
  `docs/cache.md`、`docs/database-index.md`、`docs/二次开发规范.md`、`docs/前端主题开发及API调用规范.md`

---

## 功能模块一览

完整清单（接口 / 参数 / 数据表）见 [`docs/功能清单.md`](docs/功能清单.md)。

| 分类 | 模块 |
|------|------|
| 内容 | `article` 文章、`article-group` 分组、`pages` 独立页面、`moments` 动态、`comment` 评论、`tags` 标签、`topics` 话题、`feed` 内容流、`search` 搜索、`attachment` 附件、`docs` 文档站、`rss`、`unlock` 解锁、`tip` 打赏 |
| 用户 | `comm` 登录注册、`users`、`OAuth` 第三方登录、`user-certification` 认证、`user-follows` / `user-likes` / `user-collects` / `user-blocks`、`notification` 通知、`privacy` 隐私分级 |
| 社区 | `chat` 私信、`room` 群聊、`chat-moderation`、`report` 举报、`mention` @提及、`mail-notify` |
| 激励 | `exp` 经验、`integral` 积分、`checkin` 签到、`level` 等级、`badge` 勋章、`decoration` 装扮、`goods` 商城 |
| 运营 | `links` 友链、`banner` 轮播、`placard` 公告、`stats` 统计、`log` 日志、`config` 系统配置、`theme` 主题 |
| 安全与系统 | `auth-rules` / `auth-group` / `auth-pages` 权限体系、`ip-black` / `ip-white`、`qps-warn`、`api-keys`、`toml` 配置读写、`proxy` 代理、`audit` / `scope` 审核与可见范围 |

**统一接口约定**：`{ code, msg, data }`；`all` 支持 `page` / `limit` / `order` / `where` / `like` / `withTrashed`；
标准 CRUD 含 `save`（upsert）、`remove`（软删除）、`restore`、`delete`（物理删除）、`clear`（清空回收站）。

---

## 目录结构

```
INIS-cms/
├── main.go                     # 程序入口（启动 / 配置热重载 / 优雅退出）
├── go.mod / go.sum             # Go 依赖
├── build.bat / build.sh        # 本地编译脚本（可内嵌前端）
├── install.sh / install.lock   # Linux 安装脚本 / 安装锁
├── .goreleaser.yaml            # 多平台发布配置
├── .github/workflows/release.yml  # 推 tag 自动发布（含 govulncheck 阻断）
├── scripts/prepare-frontend.mjs   # 构建 Mellow 并同步到 theme/dist（Goreleaser before hook）
│
├── app/                        # 后端核心代码
│   ├── api/                    # 接口层
│   │   ├── controller/         # 控制器（50+，与模块同名）
│   │   ├── middleware/         # 接口中间件（api-key / captcha / rule / restriction / file_limit…）
│   │   └── route/app.go        # /api 路由注册
│   ├── dev/                    # 安装向导、主机信息（/dev）
│   ├── index/                  # 首页渲染与 SPA 路由回退
│   ├── socket/                 # WebSocket（/socket）
│   ├── timer/                  # 定时任务（封禁解封、装扮到期、日志清理…）
│   ├── middleware/             # 全局中间件（CORS / 安装检测 / QPS / 日志脱敏 / TLS…）
│   ├── facade/                 # 门面层（DB、缓存、存储、短信、邮件队列、配置模板…）
│   ├── model/                  # 数据模型与业务逻辑（50+）
│   └── validator/              # 请求参数校验（20+）
│
├── config/                     # 配置目录（*.toml 首次启动生成，含密钥不提交）
│   ├── app.go                  # 配置加载与服务启动
│   ├── i18n-assets.go          # 语言包内嵌（go:embed）
│   └── i18n/                   # zh-cn / en-us / ko-kr / ru-ru
│
├── Mellow/                     # 前台主题 + 管理后台（Vue 3 + Vite）
│   ├── src/views/              # 页面：home / article / topic / moments / docs / chat / user / admin…
│   ├── src/components/         # 组件：卡片、编辑器、表单控件（components/form）、弹窗…
│   ├── src/api/                # 接口封装
│   └── vite.config.js          # 开发代理与构建配置
│
├── inis-app/                   # 移动端（uni-app x）
│   ├── pages/                  # 页面（.uvue）
│   ├── components/             # 组件（.uvue）
│   ├── config/env.ts           # API_DOMAIN 等运行环境配置
│   └── manifest.json           # 应用配置
│
├── theme/                      # 内嵌主题
│   ├── theme.go                # 主题查找与响应元信息（默认空实现）
│   ├── theme_embed.go          # //go:embed all:dist（仅 -tags embed）
│   └── dist/                   # Mellow 构建产物（构建时同步）
│
├── public/                     # 静态资源（表情包；非内嵌构建时主题产物部署位置）
└── docs/                       # 接口文档、功能清单、二次开发与主题开发规范
```

> `app/api/controller`、`app/model`、`app/validator` 三层的文件名与模块一一对应
> （例如 `article.go` → 文章），新增模块时三层同步创建即可。

---

## 开发指南

### 代码规范

- 遵循 Go 官方代码规范，提交前执行 `gofmt`
- 函数保持单一职责，注释说明用途、参数与隐含逻辑
- 接口文档与代码保持同步，文档缺失即视为未完成

### 新增一个功能模块

1. `app/model/` 建数据模型，并在 `app/model/base.go` 注册建表
2. `app/api/controller/` 建控制器（`allow` 白名单映射方法）
3. `app/api/route/app.go` 注册路由
4. `app/validator/` 补参数校验（如需要）
5. `app/model/auth-rules.go` 补权限规则、`auth-pages.go` 补后台页面权限（如需要）
6. 前端在 `Mellow/src/api/` 与 `Mellow/src/views/` 对应实现

### 数据库

使用 GORM `AutoMigrate` 自动建表 / 加字段，启动时同步权限规则种子。**升级后必须重启一次后端**，
只替换前端产物不会创建新表、也不会同步新权限点。

---

## 常见问题

### Q：端口被占用 / 想换端口？

改 `config/app.toml` 的端口配置（默认 8642），保存后配置会自动热重载。

### Q：首页 404 或空白？

主题文件没有部署到位。要么把 `Mellow/dist` 拷到 `public/`，要么用内嵌构建（`go build -tags embed`，主题随二进制自带）。

### Q：升级代码后新接口返回 403，或提示表不存在？

权限规则与新表都在**后端启动时**同步（`EnsureAuthRules` + `AutoMigrate`）。升级后必须重启后端，
只替换前端产物不够。

### Q：`#话题#` 没变成可点击链接？

话题解析与规范化在后端完成，需重新编译并重启后端。

### Q：发布流水线失败，日志里有 `exit code 3`？

`govulncheck` 命中漏洞时会以退出码 3 终止（CI 故意设成阻断）。按日志升级对应依赖
（例如 `go get golang.org/x/net@v0.60.0`）后重新打 tag 即可。

### Q：单文件部署后上传的图片存在哪？

上传落盘在 `storage/`（或按 `storage.toml` 走 COS），不会写进二进制；主题静态资源才在内嵌产物里。

---

## 贡献

欢迎提交 Issue 与 Pull Request。提交前请确认：

- `go build ./...` 通过
- 涉及的依赖升级后同步更新 `README.md` 的技术栈表格
- 新增接口补充 `docs/API docs/` 文档

## 联系方式

- GitHub Issues：<https://github.com/zhu885744/INIS-cms/issues>
- 交流群：119300889
- 邮箱：xz@zhuxu.asia

## 致谢

- 原作者「陈兔子」：<https://github.com/racns>
- 原开源仓库（已停更）：<https://github.com/inis-io/inis>
- 感谢所有为开源社区做出贡献的开发者

## 开源许可

[Apache License 2.0](LICENSE)
