## 日志解析 API

### 接口概述

后台运维用的**只读**接口：把 `runtime/` 下的站点日志解析成结构化数据返回，
由后台「管理 → 日志」（`/admin/logs`）消费。

站点日志有两类，落盘位置与格式不同：

| 来源 | 文件 | 格式 |
| :--- | :--- | :--- |
| 系统日志 | `runtime/logs/<日期>/{info,warn,error,debug}.log` | zap JSON 编码，一行一条 |
| 通知日志 | `runtime/sms/{sms,email}.log` | 文本行：`[时间] [状态] 类型 \| k: v \| k: v` |

系统日志按天分目录；lumberjack 按「单文件 2MB / 保留 7 天 / 最多 20 个备份」轮转，
备份文件名形如 `info-2026-10-10T12-00-00.000.log`（同样可被列出与解析）。
轮转参数见 `config/log.toml`（`size` / `age` / `backups`），后台「系统设置 → 消息与日志」可改。

解析后每条日志统一为：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `time` | string | 日志时间（本地时间，`YYYY-MM-DD HH:mm:ss`） |
| `level` | string | `debug` / `info` / `warn` / `error`；解析不出来的行为空 |
| `msg` | string | 日志正文（通知日志为 `sms` / `email`） |
| `caller` | string | 调用位置，如 `comment.go:42`（系统日志才有） |
| `status` | string | 通知日志的「成功 / 失败」 |
| `fields` | object | 业务字段：系统日志的附加键；通知日志的 `k: v` 明细 |
| `raw` | string | 原始行（始终返回，便于比对与复制） |

**权限**：三个接口都是 `type=default`（需在「权限规则」里给账号授予对应权限点），
管理员（root）默认可见。

**安全**：只允许读取 `runtime/` 下白名单目录里的 `.log`；
日期（`YYYY-MM-DD`）与文件名都做正则白名单校验，不含路径分隔符，
因此 `../` 之类的目录穿越会被直接拒绝（返回 400）。

---

### 接口列表

#### 1. 可查看的日期 [dates]

**请求方式**: `GET` ｜ **需权限点**

**请求地址**: `/api/log/dates`

**响应示例**:

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "dates": [
      { "date": "2026-10-10", "files": 4, "size": 1839402 },
      { "date": "2026-10-09", "files": 3, "size": 512300 }
    ],
    "notify": [
      { "name": "email.log", "size": 20480, "update_time": 1760123456 },
      { "name": "sms.log", "size": 1024, "update_time": 1760123456 }
    ],
    "today": "2026-10-10"
  }
}
```

> `files` / `size` 是该日期目录下的文件数与总字节数（用于前端展示概览）；
> `dates` 按日期倒序（最近的在最前）。

#### 2. 文件列表 [files]

**请求方式**: `GET` ｜ **需权限点**

**请求地址**: `/api/log/files`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| scope | string | 否 | `system`（默认，系统日志） / `notify`（通知日志） |
| date | string | 否 | `YYYY-MM-DD`，默认今天；`scope=notify` 时忽略 |

**响应示例**:

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "scope": "system",
    "date": "2026-10-10",
    "list": [
      { "name": "info.log", "size": 51200, "update_time": 1760123456, "rotated": false },
      { "name": "warn.log", "size": 1024, "update_time": 1760123456, "rotated": false },
      { "name": "error.log", "size": 2048, "update_time": 1760123456, "rotated": false },
      { "name": "debug.log", "size": 0, "update_time": 1760123456, "rotated": false },
      { "name": "info-2026-10-10T06-00-00.000.log", "size": 2097152, "update_time": 1760104800, "rotated": true }
    ]
  }
}
```

> `rotated: true` 表示 lumberjack 轮转出来的历史文件（文件名里带时间戳）。
> 排序：`info → warn → error → debug`，同类中轮转备份排在当前文件之后。

#### 3. 解析并读取日志 [read]

**请求方式**: `GET` ｜ **需权限点**

**请求地址**: `/api/log/read`

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| scope | string | 否 | `system`（默认） / `notify` |
| date | string | 否 | `YYYY-MM-DD`，默认今天（`scope=system` 时有效） |
| file | string | 是 | 文件名，如 `error.log` / `email.log`（白名单校验） |
| level | string | 否 | 只看某一级别：`error` / `warn` / `info` / `debug` |
| keyword | string | 否 | 关键字（大小写不敏感，匹配**原始行**，含字段值） |
| page | number | 否 | 页码，默认 1 |
| limit | number | 否 | 每页条数，默认 50，最大 200 |
| order | string | 否 | `desc`（默认，最新在前） / `asc` |

**响应示例**:

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "scope": "system",
    "date": "2026-10-10",
    "file": "error.log",
    "list": [
      {
        "time": "2026-10-10 19:49:47",
        "level": "error",
        "msg": "创建评论失败",
        "caller": "comment.go:42",
        "fields": { "error": "Duplicate entry 'x' for key 'uid'", "uid": 7 },
        "raw": "{\"level\":\"error\",\"time\":\"2026-10-10 19:49:47\",\"caller\":\"comment.go:42\",\"msg\":\"创建评论失败\",\"error\":\"Duplicate entry 'x' for key 'uid'\",\"uid\":7}"
      }
    ],
    "total": 1,
    "lines": 128,
    "pages": 1,
    "page": 1,
    "limit": 50,
    "levels": { "debug": 0, "info": 120, "warn": 7, "error": 1, "other": 0 },
    "truncated": false
  }
}
```

**字段说明**：

| 字段 | 说明 |
| :--- | :--- |
| `total` | 过滤后的条数（分页用） |
| `lines` | 文件总行数（扫描到的行数，不受过滤影响） |
| `levels` | 整个文件的级别统计（同样不受过滤影响；`other` = 解析失败的行） |
| `truncated` | 是否因超过单文件扫描上限（20 万行）而截断 |
| `empty` | 文件不存在时为 `true`，此时 `list` 为空数组、`msg` 提示「该日志文件不存在或已被清理」，**状态码仍是 200** |

---

### 注意事项

1. **大文件保护**：单文件最多扫描 20 万行（超出时 `truncated: true`），单页最多 200 条；
   日志字段很长时单行上限 2MB。
2. **过滤口径**：`level` 按解析出的级别过滤；`keyword` 按原始行做大小写不敏感匹配
   （因此字段值里的内容也能搜到）。
3. **只读**：`POST` / `PUT` / `DELETE` 一律返回 405「日志为只读数据，不支持该操作！」。
   日志清理仍由 lumberjack 的保留策略（`config/log.toml`）负责。
4. **解析兜底**：JSON 解析失败或文本行不匹配时，条目只带 `raw`（`level` 为空，计入
   `levels.other`），保证任何一行都能在后台看到原文。
