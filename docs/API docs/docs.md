# 文档站管理 接口文档

## 接口概述

站点内置文档功能：目录 + 多页面，支持多级目录、多版本、排序、可见性、历史与反馈。

- 后端：`app/api/controller/docs.go`、`app/api/controller/docs-group.go`
- 模型：`app/model/docs*.go`
- 前台：`inis-cms/src/views/docs/*`

### 数据模型

| 表 | 说明 |
| :--- | :--- |
| `docs_group` | 目录，`pid` 自关联实现无限层级 |
| `docs` | 文档页面，`content` 存 Markdown 原文（由前端渲染） |
| `docs_history` | 每次保存的历史快照 |
| `docs_feedback` | “这篇文档有帮助吗”的反馈 |
| `docs_redirect` | slug 变更后的旧路径 → 新路径（用于 301） |

### 关键字段

| 字段 | 说明 |
| :--- | :--- |
| `version` | 版本号，默认 `latest` |
| `slug` | 目录 / 页面标识，同版本同父级内唯一 |
| `sort` | 升序排列，值小的在前 |
| `status` | 目录：1 启用 / 0 禁用；页面：0 草稿 / 1 发布（文档不走审核流程） |
| `visible` | 1 公开 / 2 登录可见 / 3 指定角色组 |
| `role_ids` | `visible=3` 时生效，逗号分隔的权限组 ID |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 查询、保存、排序、移动成功 |
| 201 | 创建成功 | 新建文档 / 目录成功 |
| 204 | 无内容 | 查询无数据、无可操作数据 |
| 400 | 请求错误 | 参数校验失败、移动目录时移入自身子树 |
| 401 | 未授权 | 未登录，或访问 `visible=2` 的内容 |
| 403 | 禁止访问 | 无权限点、不在 `role_ids` 内的角色组 |
| 404 | 未找到 | 文档 / 目录不存在或已删除 |
| 405 | 方法不允许 | 请求方法或方法名错误 |
| 500 | 服务器错误 | 服务端内部错误 |

---

## 接口列表

### 文档目录接口

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `/api/docs-group/tree` | 公开 | 目录树，参数 `version`、`with_docs=1`、`admin=1` |
| GET | `/api/docs-group/one` | 公开 | 单个目录（`id` 或 `slug` + `version`） |
| GET | `/api/docs-group/versions` | 公开 | 版本列表 |
| GET | `/api/docs-group/all` | 权限点 | 平铺列表（分页） |
| GET | `/api/docs-group/count` | 权限点 | 数量 |
| POST/PUT | `/api/docs-group/save` | 权限点 | 新建 / 更新（带 `id` 为更新） |
| POST/PUT | `/api/docs-group/sort` | 权限点 | 批量排序：`{"items":[{"id":1,"sort":1}]}` |
| POST/PUT | `/api/docs-group/move` | 权限点 | 移动目录：`{"id":1,"pid":2}`，会校验不能移入自身子树 |
| POST | `/api/docs-group/copy-version` | 权限点 | 复制版本：`{"from":"latest","to":"v2"}` |
| DELETE | `/api/docs-group/remove` | 权限点 | 软删除（有子目录或页面时拒绝） |
| DELETE | `/api/docs-group/delete` | 权限点 | 彻底删除 |
| PUT | `/api/docs-group/restore` | 权限点 | 恢复 |

---

### 文档页面接口

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `/api/docs/one` | 公开 | 详情：`version` + `group`(目录 slug) + `slug`，或 `id` |
| GET | `/api/docs/all` | 公开 | 列表，支持 `gid`、`status`、分页 |
| GET | `/api/docs/search` | 公开 | 搜索：`keyword`，可选 `version`、`gid`、分页 |
| POST | `/api/docs/feedback` | 公开 | 反馈：`{"docs_id":1,"useful":1}` |
| POST/PUT | `/api/docs/save` | 权限点 | 新建 / 更新，保存后写历史快照 |
| POST/PUT | `/api/docs/sort` | 权限点 | 批量排序 |
| POST/PUT | `/api/docs/move` | 权限点 | 页面换目录：`{"id":1,"gid":2}` |
| GET | `/api/docs/history` | 权限点 | 历史列表：`docs_id` |
| POST | `/api/docs/rollback` | 权限点 | 回滚：`history_id` |
| DELETE | `/api/docs/remove` | 权限点 | 软删除 |
| DELETE | `/api/docs/delete` | 权限点 | 彻底删除（连带历史 / 反馈） |
| PUT | `/api/docs/restore` | 权限点 | 恢复 |
| GET | `/api/docs/feedback-stats` | 权限点 | 单篇反馈统计 |
| POST | `/api/docs/import` | 权限点 | 导入 Markdown（单个 `.md` 或 `.zip`），`multipart/form-data` |

### 导入接口

| 字段 | 必填 | 说明 |
| :--- | :--- | :--- |
| `file` | 是 | `.md` / `.markdown` / `.zip` |
| `version` | 否 | 目标版本，默认 `latest` |
| `gid` | 否 | 导入到指定目录（压缩包内的子目录继续挂在这个目录下） |
| `overwrite` | 否 | `1` 覆盖同 slug 文档（默认），`0` 跳过 |

解析规则：

- 单个 Markdown：文件名转 slug，正文第一个 `# 标题` 作为标题；
- 压缩包：目录结构 → 文档目录，自动剥掉统一的顶层目录（如 `my-docs/`），
  跳过 `__MACOSX`、隐藏文件和非 Markdown 文件；
- front-matter（`---` 包裹）支持 `title` / `slug` / `sort` / `status`；
- 中文文件名转不出 slug 时用 `doc-<hash>` 兜底，建议在 front-matter 里显式指定 `slug`；
- 限制：上传 32MB、压缩包最多 800 篇、单篇 5MB。

返回：

```json
{
  "groups": 3,
  "created": 12,
  "updated": 2,
  "skipped": 0,
  "errors": []
}
```

### `/api/docs/one` 返回结构

```json
{
  "id": 1,
  "gid": 2,
  "version": "latest",
  "slug": "install",
  "title": "安装",
  "content": "## 安装\n\n...\n",
  "breadcrumb": [{ "id": 2, "name": "快速开始", "slug": "start" }],
  "prev": { "id": 1, "title": "介绍", "path": "/docs/latest/start/intro" },
  "next": null,
  "useful": 3,
  "useless": 0,
  "path": "/docs/latest/start/install"
}
```

---

## 特殊说明

### 渲染说明

- 服务端只存 Markdown 原文（`content`），**不做任何渲染**，也不缓存 HTML；
- 前台用项目内置的 `markdown-it` 渲染，开启 `breaks`（单换行即`<br>`）、GFM 表格与代码高亮；
- 渲染后遍历 `h2` / `h3` 补锚点 `id` 并生成右侧 TOC；
- 外链自动补 `target="_blank"` 与 `rel="noopener noreferrer"`；
- 前端渲染时需自行处理 XSS（markdown-it 默认不执行内联 HTML）。

### 可见性与状态

- 前台 `one` / `all` / `tree` 只返回 `status=1` 的页面与启用目录；
- `visible=2` 未登录返回 `401`；`visible=3` 校验用户所属权限组是否在 `role_ids` 中；
- 后台传 `admin=1` 可读取草稿内容（需要权限点）。

### 尚未实现

- 后台管理页面（目录树拖拽、编辑器、历史 diff）
- 前端代码高亮 / 复制按钮 / 阅读进度条
- 评论接入（`comment.bind_type=docs`）
- `docs_redirect` 的前台 301 消费
- sitemap / TechArticle 结构化数据
- Markdown 文件夹导入导出
