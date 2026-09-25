# Notification 消息通知接口文档

## 接口概述

`notification` 控制器用于管理系统通知，支持通知列表查询、未读计数、标记已读、批量操作等功能。当用户收到新回复、被点赞、被收藏、被关注时，系统自动创建通知（仅落库），前端通过 API 拉取列表与未读数。所有接口均需登录，数据按用户隔离。

### 可见范围与权限模型（重要）

所有查询类接口（`one` / `all` / `count` / `rand` / `column` / `sum` / `min` / `max`）共用同一套可见范围（`controller.applyScope`）：

| 身份 | 可见范围 | 说明 |
| :--- | :--- | :--- |
| 普通用户 | `uid = 当前用户` | 只能看到自己的个人通知 |
| 管理员（root） | `uid IN (0, 当前用户)` | 自己的通知 + 全体可见的系统消息（uid=0），便于查看/统计 |
| 管理员（root）+ `scope=admin` | 不限 `uid` | 管理端视角：全站通知（含其它用户收到的记录），供后台「消息管理」使用 |

- 显式传 `uid` 参数会在此基础上做精确过滤（`uid=0` 只看广播，`uid=自己的 id` 只看个人通知），后台「个人消息」统计与列表都传自己的 uid，保证口径一致；
- 广播通知的「已读 / 隐藏」状态属于每个用户各自的记录（见下方 `NotificationRead`），
  因此 `list` / `unread-count` 会按状态表过滤，而 `all` / `count` 等基础接口只做「可见性」判断；
- `scope=admin` 只对管理员（root）生效：非 root 即使传了该参数也会被忽略，回落到默认可见范围；
- 不带 `scope=admin` 时，后台**无法查看其它用户收到的个人消息**（默认范围只有「自己 + 广播」）。

### 系统消息（全体可见，uid=0）

「系统消息」是功能名，发送给全体用户时在 `notification` 表中只存**一条** `uid=0` 的记录
（后台「消息」页的默认模块），不会给每个用户各写一条：

- 每个用户对该广播的「已读 / 隐藏」状态记录在 `notification_read` 表（唯一键 `notification_id + uid`）；
- 用户删除广播 = 写 `is_deleted=1`（只对自己隐藏）；管理员删除 = 软删除广播本体（对全体撤回）；
- `read-all` 只处理「该用户未读且未隐藏」的广播，避免为每条广播做一次无谓的 upsert。

### 通知触发机制

| 触发事件 | 通知类型 | 通知对象 | 通知内容 |
| :--- | :--- | :--- | :--- |
| 文章/动态/页面收到新评论 | `comment` | 内容作者 | "xxx 回复了你的{内容标题}" |
| 评论被点赞 | `like` | 评论作者 | "xxx 赞了你的评论「{评论摘要}」" |
| 文章/页面/动态被点赞 | `like` | 内容作者 | "xxx 赞了你的{内容类型}「{内容标题}」" |
| 文章/页面/动态被收藏 | `collect` | 内容作者 | "xxx 收藏了你的{内容类型}「{内容标题}」" |
| 被用户关注 | `follow` | 被关注者 | "xxx 关注了你" |
| 管理员调整积分 | `system` | 目标用户 | "管理员为你增加了 N 积分，当前积分余额为 M。"（归入系统通知，不单独占用类型） |
| 系统消息 | `system` | 指定用户 / 全体 | 管理员通过 `send-system` 发送；`target_type=all` 时生成一条广播（uid=0） |
| 注册欢迎消息 | `system` | 新注册用户 | "欢迎加入 {站点标题}"（后台「网站设置 → 注册」开关控制） |
| 账号登录通知 | `system` | 登录用户 | 记录登录账号 / 时间 / IP / 设备，提示确认是否本人操作 |

### 接口类型说明

| 接口类型 | 说明 |
| :--- | :--- |
| **基础接口** | one、all、rand、count、sum、min、max、column、remove、delete、clear、restore、save、create、update |
| **业务接口** | list（获取通知列表）、unread-count（获取未读数量）、read（标记已读）、read-all（全部已读）、read-batch（批量已读）、remove-all（清空通知） |

> 批量上限：`read-batch` 单次最多 200 个 ID（超出返回 400，与 `controller.notificationBatchLimit` 一致）；
> `remove-all` 单次最多处理 200 条，返回 `{ cleared, remaining }`，`remaining > 0` 时需再次调用（前端可用 `clearAllNotifications` 循环）。

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 获取数据成功、操作成功 |
| **202** | 接受请求 | 请求已接受（无实际功能） |
| **204** | 无内容 | 查询无数据、无可操作数据 |
| **400** | 请求错误 | 参数校验失败、操作失败 |
| **401** | 未授权 | 用户未登录 |
| **403** | 无权限 | 无操作权限 |
| **405** | 方法不允许 | 请求方法错误或方法名错误 |
| **500** | 服务器错误 | 系统内部错误 |

---

## 数据模型

### Notification 结构

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 主键，自增 |
| `uid` | int | 接收通知的用户ID |
| `from_uid` | int | 触发通知的用户ID |
| `type` | string | 通知类型：comment/like/collect/follow/system（管理员调整积分触发的积分变动通知归入 `system`） |
| `title` | string | 通知标题 |
| `content` | string | 通知内容 |
| `bind_id` | int | 关联实体ID（文章/评论/用户等） |
| `bind_type` | string | 关联实体类型：article/page/moments/comment/user |
| `is_read` | int | 是否已读：0=未读，1=已读 |
| `json` | any | JSON扩展数据 |
| `text` | any | 文本扩展数据 |
| `result` | any | 复合返回结果（含 from_user 信息） |
| `create_time` | int64 | 创建时间戳 |
| `update_time` | int64 | 更新时间戳 |
| `delete_time` | soft_delete | 软删除时间戳 |

> `uid = 0` 表示「系统消息（全体可见）」：只存一条记录，用户侧的已读/隐藏状态见下表。

### NotificationRead 结构（广播通知的用户状态）

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 主键，自增 |
| `notification_id` | int | 广播通知 ID（与 `uid` 组成唯一键） |
| `uid` | int | 用户 ID |
| `is_read` | int | 该用户是否已读：0=未读，1=已读 |
| `is_deleted` | int | 该用户是否隐藏：0=否，1=是（用户删除广播后写入） |
| `create_time` | int64 | 创建时间戳 |
| `update_time` | int64 | 更新时间戳 |

---

## 接口列表

### 1. GET 请求接口

#### 1.1 获取通知列表 [业务接口]

- **路径**: `/api/notification/list`
- **方法**: `GET`
- **描述**: 分页获取当前用户的通知列表，支持按类型和已读状态筛选

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `size` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `type` | string | 否 | 通知类型过滤：comment/like/follow/system |
| `is_read` | int | 否 | 已读状态过滤：0=未读，1=已读；**不传 / 传空串 / 传 null 均表示「全部」** |
| `field` | string | 否 | 返回字段，逗号分隔 |

> 列表范围：当前用户的通知 + 广播通知（`uid=0`，且该用户未隐藏）。
> 广播的已读状态取自 `notification_read` 表（`is_read` / `is_deleted`），
> 因此同一条广播在不同用户视角下的 `is_read` 可能不同。

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "data": [
            {
                "id": 5,
                "uid": 100,
                "from_uid": 200,
                "type": "comment",
                "title": "收到新回复",
                "content": "张三 回复了你的文章《Go语言入门》",
                "bind_id": 10,
                "bind_type": "article",
                "is_read": 0,
                "create_time": 1734567890,
                "result": {
                    "from_user": {
                        "id": 200,
                        "nickname": "张三",
                        "avatar": "https://...",
                        "description": "个人简介",
                        "title": "LV.3"
                    }
                }
            }
        ],
        "count": 15,
        "page": 3
    }
}
```

**使用示例**:
```
GET /api/notification/list?type=comment&is_read=0&page=1&size=20
```

**权限说明**: 需要登录，仅返回「当前用户的通知 + 广播通知」；无法读取其它用户的个人通知

#### 1.2 获取未读通知数量 [业务接口]

- **路径**: `/api/notification/unread-count`
- **方法**: `GET`
- **描述**: 获取当前用户的未读通知总数

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": {
        "count": 5
    }
}
```

**使用示例**:
```javascript
// 前端轮询（或进入消息页时）更新未读角标
const res = await request.get('/api/notification/unread-count')
// res.data = { count: 5 }
```

**权限说明**: 需要登录

#### 1.3 获取单个通知 [基础接口-获取指定]

- **路径**: `/api/notification/one`
- **方法**: `GET`
- **描述**: 根据条件获取单个通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 通知ID |
| `type` | string | 否 | 通知类型 |
| `is_read` | int | 否 | 已读状态 |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |
| `withTrashed` | bool | 否 | 是否包含已删除数据 |
| `onlyTrashed` | bool | 否 | 是否只查询已删除数据 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "id": 1,
        "uid": 100,
        "from_uid": 200,
        "type": "like",
        "title": "获得新点赞",
        "content": "李四 赞了你的评论「说得很有道理...」",
        "bind_id": 5,
        "bind_type": "comment",
        "is_read": 0,
        "create_time": 1734567890,
        "result": {
            "from_user": {
                "id": 200,
                "nickname": "李四",
                "avatar": "https://..."
            }
        }
    }
}
```

**权限说明**: 需要登录，用户只能查询自己的通知；管理员（root）可同时查询广播通知（uid=0）

#### 1.4 获取所有通知 [基础接口-获取全部]

- **路径**: `/api/notification/all`
- **方法**: `GET`
- **描述**: 分页获取通知记录列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认1 |
| `size` | int | 否 | 每页数量 |
| `order` | string | 否 | 排序字段，默认 `create_time desc` |
| `field` | string | 否 | 返回字段，逗号分隔 |
| `uid` | int | 否 | 精确过滤接收人：传 `0` 只看系统消息（全体可见），传自己的 id 只看个人消息 |
| `where` | json | 否 | 条件查询 |
| `like` | json | 否 | 模糊查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "data": [...],
        "count": 50,
        "page": 5
    }
}
```

**权限说明**: 需要登录。可见范围：普通用户仅自己的通知；管理员（root）为自己 + 广播通知（uid=0）；
管理员传 `scope=admin`（管理端视角）时不限 `uid`，可列出全站通知（含发给指定用户的消息与评论/点赞等自动通知）。

后台「个人消息」列表与统计都传自己的 `uid`，避免把全体可见的系统消息混入个人消息口径；
后台「消息管理」则用 `scope=admin`（可按 `where={"type":"system"}` / `where={"is_read":0}` / `onlyTrashed` 收窄）。

#### 1.5 随机获取通知 [基础接口-随机获取]

- **路径**: `/api/notification/rand`
- **方法**: `GET`
- **描述**: 随机获取指定数量的通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `size` | int | 否 | 返回数量 |
| `except` | string | 否 | 排除的ID，逗号分隔 |
| `field` | string | 否 | 返回字段 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "好的！",
    "data": [...]
}
```

**权限说明**: 需要登录，可见范围同 `all`（普通用户仅自己，管理员含广播）

#### 1.6 查询数量 [基础接口-查询数量]

- **路径**: `/api/notification/count`
- **方法**: `GET`
- **描述**: 查询通知记录数量

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `where` | json | 否 | 条件查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "查询成功！",
    "data": 50
}
```

**权限说明**: 需要登录，可见范围同上（管理员可用 `scope=admin` 统计全站通知）；
可用 `where` / `uid` 进一步收窄（后台「个人消息」统计即传自己的 `uid`）

#### 1.7 求和 [基础接口-求和]

- **路径**: `/api/notification/sum`
- **方法**: `GET`
- **描述**: 对指定字段求和

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 求和字段 |
| `where` | json | 否 | 条件查询 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "is_read": 42
    }
}
```

**权限说明**: 需要登录，可见范围同上

#### 1.8 最小值 [基础接口-最小值]

- **路径**: `/api/notification/min`
- **方法**: `GET`
- **描述**: 获取指定字段最小值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1734500000
    }
}
```

**权限说明**: 需要登录，可见范围同上

#### 1.9 最大值 [基础接口-最大值]

- **路径**: `/api/notification/max`
- **方法**: `GET`
- **描述**: 获取指定字段最大值

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": {
        "create_time": 1734600000
    }
}
```

**权限说明**: 需要登录，可见范围同上

#### 1.10 查询列 [基础接口-查询列]

- **路径**: `/api/notification/column`
- **方法**: `GET`
- **描述**: 获取指定字段列表

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `field` | string | **是** | 字段名 |
| `where` | json | 否 | 条件查询 |
| `order` | string | 否 | 排序字段 |
| `ids` | string | 否 | 指定ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "数据请求成功！",
    "data": [1, 2, 3]
}
```

**权限说明**: 需要登录，可见范围同上

---

### 2. POST 请求接口

#### 2.1 保存通知 [基础接口-保存数据]

- **路径**: `/api/notification/save`
- **方法**: `POST`
- **描述**: 保存通知记录（id为空时新增，不为空时更新）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | 否 | 通知ID，为空时新增 |
| `uid` | int | 否 | 接收用户ID |
| `from_uid` | int | 否 | 触发用户ID |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `bind_id` | int | 否 | 关联实体ID |
| `bind_type` | string | 否 | 关联实体类型 |
| `is_read` | int | 否 | 已读状态 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1
    }
}
```

**权限说明**: 需要登录。id 为空时等同 `create`；不为空时等同 `update`，
两者都带归属校验（普通用户只能操作自己的通知）

#### 2.2 创建通知 [基础接口-添加数据]

- **路径**: `/api/notification/create`
- **方法**: `POST`
- **描述**: 新增通知记录（仅落库，前台通过 API 拉取，无 WebSocket 推送）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 否 | 接收用户ID；**普通用户传了也会被忽略**（一律落到自己名下） |
| `from_uid` | int | 否 | 触发用户ID；普通用户同样不可指定 |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `bind_id` | int | 否 | 关联实体ID |
| `bind_type` | string | 否 | 关联实体类型 |
| `is_read` | int | 否 | 已读状态，默认0 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "创建成功！",
    "data": {
        "id": 1
    }
}
```

**权限说明**: 需要登录。

- 普通用户：`uid` / `from_uid` 会被强制改写为本人，**无法伪造广播（uid=0）或给他人投递消息**；
- 管理员（root）：可指定 `uid` / `from_uid`；
- 需要「给全体用户或指定用户发送系统消息」请使用业务接口 `send-system`（有独立的管理员校验）。

#### 2.3 系统消息推送 [业务接口]

- **路径**: `/api/notification/send-system`
- **方法**: `POST`
- **描述**: 管理员向全体 / 指定用户推送系统消息（同时可选发送邮件）

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_type` | string | 否 | `all`（广播，默认）/ `partial` / `single` |
| `user_ids` | array/string | 条件必填 | 目标用户 ID 列表，`target_type != all` 时必填 |
| `title` | string | **是** | 通知标题 |
| `content` | string | **是** | 通知内容 |
| `send_email` | bool | 否 | 是否同时发邮件（仅指定用户时生效，广播不支持邮件） |
| `as_system` | bool | 否 | 是否以「系统」身份发送：`true` 标题加 `【短消息】` 前缀；`false` 正文前缀为「{管理员昵称} 发送了一条短消息：…」，昵称为空时回退「管理员」 |

**成功响应** (200)：

```json
{
    "code": 200,
    "msg": "广播成功！全体用户可见",
    "data": {
        "broadcast": true,
        "id": 12,
        "total": 1,
        "success": 1
    }
}
```

指定用户时返回 `{ "total": N, "success": M }`（逐个创建通知，失败会记日志并继续）。

**邮件通知**（`send_email=true`，仅指定用户生效）：每个目标用户发一封，使用 `facade.SendMessageNotify`
（`app/facade/sms.go`）的「用户消息通知」模板 —— 品牌栏 + 通知标题 + 通知内容 + 发送时间，
标题与正文按纯文本处理（转义后换行转 `<br>`），不含评论通知模板里的「评论者 / 评论 IP」等字段。
该链路直接走邮箱驱动（GoMail），与 `config/sms.toml` 的驱动模式无关（配了短信驱动也照发邮件）。

邮件**只入队不阻塞**：由邮箱发件队列分批投递（默认每 10 分钟最多 10 封，见 `config/sms.toml` 的 `[email]` 段
与 `app/facade/mail_queue.go`），失败会自动延迟重试，超过最大次数标记失败；发送结果记入
`runtime/sms/email.log`（`type: 消息通知`），失败只记日志，不影响站内通知投递。
群发量较大时收件人会按批次先后陆续收到，属于预期行为。

**权限说明**: 仅管理员（root）可调用；`target_type=all` 只创建一条 `uid=0` 的广播记录，
并由发送者自己标记已读（避免后台角标出现未读）。

---

### 3. PUT 请求接口

#### 3.1 标记已读 [业务接口]

- **路径**: `/api/notification/read`
- **方法**: `PUT`
- **描述**: 将指定通知标记为已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 通知ID |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "标记已读成功！",
    "data": {
        "id": 5
    }
}
```

**错误响应** (400):
```json
{
    "code": 400,
    "msg": "标记已读失败！",
    "data": null
}
```

**权限说明**: 需要登录。个人通知只能标记自己的（非本人的通知为无操作的空结果）；
广播通知（uid=0）会写入该用户在 `notification_read` 中的已读状态。

#### 3.2 全部标记已读 [业务接口]

- **路径**: `/api/notification/read-all`
- **方法**: `PUT`
- **描述**: 将当前用户所有未读通知标记为已读

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "全部标记已读成功！",
    "data": null
}
```

**权限说明**: 需要登录。

- 个人通知：只更新「未读」的记录；
- 广播通知：只处理「该用户未读且未隐藏」的广播再写入已读状态，
  已读/已隐藏的广播不会重复写状态记录（避免无谓的全量 upsert）。

#### 3.3 批量标记已读 [业务接口]

- **路径**: `/api/notification/read-batch`
- **方法**: `PUT`
- **描述**: 批量将多条通知标记为已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表，支持数组 `[1,2,3]` 或字符串 `"1,2,3"`，**单次最多 200 个** |

**请求示例**:
```json
PUT /api/notification/read-batch
Content-Type: application/json

{
    "ids": [1, 2, 3, 4, 5]
}
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "批量标记已读成功！",
    "data": {
        "ids": [1, 2, 3, 4, 5]
    }
}
```

**错误响应** (400):
```json
{
    "code": 400,
    "msg": "一次最多处理 200 条，请分批操作！",
    "data": null
}
```

**权限说明**: 需要登录。个人通知只更新属于当前用户的（其它 ID 被忽略）；
广播通知写入该用户的已读状态。

#### 3.4 更新通知 [基础接口-修改数据]

- **路径**: `/api/notification/update`
- **方法**: `PUT`
- **描述**: 更新通知记录信息

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` | int | **是** | 通知ID |
| `type` | string | 否 | 通知类型 |
| `title` | string | 否 | 通知标题 |
| `content` | string | 否 | 通知内容 |
| `is_read` | int | 否 | 已读状态 |
| `uid` / `from_uid` | int | 否 | **仅管理员可改**，普通用户提交会被忽略 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "更新成功！",
    "data": {
        "id": 1
    }
}
```

**无权限/无数据** (204):
```json
{
    "code": 204,
    "msg": "无可操作数据！",
    "data": null
}
```

**权限说明**: 需要登录。

- 普通用户：只能更新自己的通知（`uid` 必须是自己），且不能修改 `uid` / `from_uid`
  （防止把别人的通知「搬」到自己名下或把消息投递到任意用户）；
- 管理员（root）：可更新自己的与广播通知（`uid IN (0, 自己)`），允许修改投递字段；
- 管理员 + `scope=admin`（管理端视角）：可更新任意用户的通知（后台「消息管理」修改发给指定用户的消息）。

#### 3.5 恢复通知 [基础接口-恢复数据]

- **路径**: `/api/notification/restore`
- **方法**: `PUT`
- **描述**: 恢复已软删除的通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "恢复成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 需要登录。归属校验：普通用户只能恢复自己的通知（`uid` 必须是自己）；
管理员（root）可恢复自己的与广播通知（`uid IN (0, 自己)`）；
管理员 + `scope=admin`（管理端视角）可恢复任意用户的通知。

---

### 4. DELETE 请求接口

#### 4.1 删除通知 [基础接口-删除]

- **路径**: `/api/notification/remove`
- **方法**: `DELETE`
- **描述**: 软删除指定的通知记录

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 需要登录。普通用户只能删除（撤回）自己的通知；管理员（root）删除广播 = 全体撤回；
管理员 + `scope=admin`（管理端视角）可撤回任意用户的通知（后台「消息管理」）。

#### 4.2 彻底删除通知 [基础接口-彻底删除]

- **路径**: `/api/notification/delete`
- **方法**: `DELETE`
- **描述**: 永久删除通知记录，不可恢复

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `ids` | array/string | **是** | 通知ID列表 |

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "删除成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 需要登录。普通用户只能彻底删除自己的通知；管理员（root）可彻底删除自己的与广播通知；
管理员 + `scope=admin`（管理端视角）可彻底删除任意用户的通知。

#### 4.3 清空通知 [业务接口]

- **路径**: `/api/notification/remove-all`
- **方法**: `DELETE`
- **描述**: 清空当前用户的通知，支持按类型批量清空、支持只清空已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `type` | string | 否 | 通知类型：comment/like/follow/system，不传则清空全部 |
| `is_read` | int | 否 | 传 `1` 时只清空「已读」通知（未读不受影响），不传则全部清空 |

**请求示例**:
```json
DELETE /api/notification/remove-all
Content-Type: application/json

{
    "type": "comment"
}
```

只清空已读（前端「清空已读」按钮对应场景，参数通过 query 传递）：

```http
DELETE /api/notification/remove-all?is_read=1
```

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "清空成功！",
    "data": {
        "ids": [1, 2, 3, 4, 5],
        "cleared": 5,
        "remaining": 0
    }
}
```

| 字段 | 说明 |
| :--- | :--- |
| `ids` | 本次软删除的个人通知 ID |
| `cleared` | 本次处理条数（个人通知 + 被隐藏的广播） |
| `remaining` | 清空后仍「可见」的条数；> 0 表示单次上限已满，需再次调用 |

> 单次最多处理 200 条（`controller.notificationBatchLimit`）：消息量大的账号一次请求
> 会构造超长 `IN` 子句，因此改为分批。前端可直接用 `clearAllNotifications(params)`
> （`@/api/notification`）循环调用直到 `remaining` 为 0。

**权限说明**: 需要登录，只能清空自己的通知。

- 个人通知：按条件软删除（可在回收站恢复，`onlyTrashed` 回收站数据不受影响）；
- 广播通知：只写该用户的「隐藏」状态（`notification_read.is_deleted=1`），
  不会删除全体共享的广播记录；`is_read=1` 时只隐藏「该用户已读」的广播。

#### 4.4 清空回收站 [基础接口-清空回收站]

- **路径**: `/api/notification/clear`
- **方法**: `DELETE`
- **描述**: 永久删除所有已软删除的通知记录（清空回收站）

**请求参数**: 无

**成功响应** (200):
```json
{
    "code": 200,
    "msg": "清空成功！",
    "data": {
        "ids": [1, 2, 3]
    }
}
```

**权限说明**: 需要登录

---

### 5. INDEX 接口

- **路径**: `/api/notification/index`
- **方法**: `GET`
- **描述**: 通知首页（无实际功能）

**成功响应** (202):
```json
{
    "code": 202,
    "msg": "没什么用！",
    "data": null
}
```

---

## 通知获取方式（API 拉取，无 WebSocket 推送）

### 机制

通知创建时**只落库**（`model.Notification.CreateNotification` / `CreateBroadcastNotification`），
不做任何 WebSocket 推送；前台统一通过 API 获取：

| 场景 | 接口 |
| :--- | :--- |
| 消息列表（含广播，按已读/隐藏过滤） | `GET /api/notification/list` |
| 基础列表 / 后台「消息管理」 | `GET /api/notification/all` |
| 未读角标 | `GET /api/notification/unread-count` |
| 未读数（可带筛选条件） | `GET /api/notification/count` |

### 示例

```javascript
// 进入消息页 / 定时轮询时拉取未读角标
const { data } = await request.get('/api/notification/unread-count')

// 消息列表
const list = await request.get('/api/notification/list', {
    params: { page: 1, size: 20, order: 'create_time desc' }
})
```

> 说明：`/socket`（`app/socket`）是独立的实时通道，目前仅用于后台系统状态推送与通用消息转发，
> 与通知模块无关；通知的投递、已读、隐藏完全由上面的查询接口决定。

---

## 前端使用指南

### 场景一：页面初始化获取通知列表

```javascript
const res = await request.get('/api/notification/list', {
    params: {
        page: 1,
        size: 20,
        order: 'create_time desc'
    }
})
const { data: notifications, count, page } = res.data.data
```

### 场景二：获取未读数量显示角标

```javascript
const res = await request.get('/api/notification/unread-count')
const unreadCount = res.data.data.count
// 在导航栏显示未读角标
```

### 场景三：点击通知标记已读

```javascript
await request.put('/api/notification/read', { id: 5 })
```

### 场景四：一键全部已读

```javascript
await request.put('/api/notification/read-all')
```

### 场景五：按类型筛选通知

```javascript
// 只看评论回复通知
const res = await request.get('/api/notification/list', {
    params: { type: 'comment', page: 1, size: 20 }
})
```

### 场景六：清空某类通知

```javascript
// 清除所有点赞通知
await request.delete('/api/notification/remove-all', {
    data: { type: 'like' }
})
```

### 场景七：跳转到关联内容

```javascript
// 根据 bind_type 和 bind_id 构建跳转链接
function getActionLink(notif) {
    switch (notif.bind_type) {
        case 'article': return `/blog/${notif.bind_id}`
        case 'moments': return `/moment/${notif.bind_id}`
        case 'page':     return `/page/${notif.bind_id}`
        case 'comment':  return null // 需要额外查询评论所属内容
        case 'user':     return `/user/${notif.bind_id}`
        default:         return null
    }
}
```

---

## 特殊说明

### 1. 权限控制
- 所有接口均需登录（`type=login`），未登录返回 401
- 查询/统计类接口统一按 `applyScope` 限定可见范围：普通用户仅自己的通知，
  管理员（root）为自己 + 广播通知（uid=0）
- 管理端视角：管理员（root）传 `scope=admin` 时不限制 `uid`，可查看 / 统计 / 维护全站通知，
  后台「消息管理」的「全部通知」视图即用该视角（`update` / `remove` / `delete` / `restore` 同样支持）
- `create` / `update` 有归属校验：普通用户不能指定 `uid` / `from_uid`，
  也无法修改他人的通知（详见对应接口说明）
- 通知创建由系统自动触发，管理端推送走 `send-system`（仅 root）

### 2. 数据隔离
- 查询接口自动过滤 `uid`（普通用户为自己，管理员包含广播；`scope=admin` 除外）
- 删除 / 恢复 / 标记已读操作强制校验 `uid` 匹配（`scope=admin` 除外）
- `count` / `sum` / `min` / `max` / `column` 等聚合操作与查询同一套可见范围
- 查询缓存键包含当前用户 ID 与全部请求参数（`...&uid=N`），避免不同用户 / 不同筛选命中同一份缓存

### 3. 通知类型说明

| 类型 | 值 | 触发时机 | bind_type | bind_id |
| :--- | :--- | :--- | :--- | :--- |
| 回复通知 | `comment` | 收到新评论 | article/page/moments | 内容ID |
| 点赞通知 | `like` | 评论被点赞 | comment | 评论ID |
| 关注通知 | `follow` | 被用户关注 | user | 关注者ID |
| 系统通知 | `system` | 平台/管理员发送（含注册欢迎消息、账号登录通知） | 自定义 | 自定义 |
| 系统消息（全体可见） | `system` | 管理员 `send-system` 且 `target_type=all`（uid=0 一条记录） | — | — |

### 4. 缓存策略
- 所有 GET 查询接口均支持缓存
- 缓存键 = `[METHOD].路径&参数hash&uid=<当前用户>`，**按用户隔离**
  （`base.cache.name()` 只由方法/路径/参数决定，不带 uid 会导致不同用户相互读到对方的消息）
- POST/PUT/DELETE 数据修改后会自动清除相关缓存（标签 `[GET]notification`），清理失败会记录 Warn 日志

### 5. 获取方式（无 WebSocket 推送）
- 通知创建后只落库，不做长连接推送；前端通过 API 拉取（列表 / 未读数）
- 用户离线时通知同样持久化在数据库，上线后查询即可看到
- 未读角标由前端定时轮询 `unread-count`（或进入消息页时）刷新

### 6. 软删除机制
- `remove` 方法执行软删除（设置 `delete_time`），可通过 `restore` 恢复
- `delete` 方法执行物理删除，不可恢复
- `remove-all` 方法执行软删除，支持按类型批量操作，传 `is_read=1` 时只软删除已读通知（清空已读）；
  **单次最多 200 条**，按 `cleared` / `remaining` 判断是否需要继续调用
- `clear` 方法彻底删除所有已软删除的数据
- 广播通知的「删除」对普通用户表现为隐藏（写 `notification_read.is_deleted=1`），
  管理员删除广播则是软删除共享记录（全体撤回）

### 7. 索引优化
数据库自动创建以下索引以保证查询性能（`InitNotification` / `InitNotificationRead`）：
- `idx_notifications_uid`：按用户ID快速查询
- `idx_notifications_type`：按通知类型过滤
- `idx_notifications_is_read`：按已读状态筛选
- `idx_notifications_create_time`：按时间排序 / 过期清理
- `idx_notification_reads_uid`：按用户查询广播的已读/隐藏状态
- 唯一索引 `uk_notification_reads_nid_uid`（`notification_id + uid`）：保证同一用户对同一广播只有一条状态记录
