# 公共聊天室 接口文档

## 接口概述

`room` 控制器用于全站唯一的公共聊天室：房间资料、历史消息、发言、撤回、删除、禁言与自动清空配置。房间 ID 固定为 `1`，所有登录用户都能发言，管理动作只对超级管理员开放。

### 接口类型说明

本文档不含基础接口，全部为业务接口：

| 接口类型 | 说明 |
| :--- | :--- |
| **查询接口** | info、messages、mutes、auto-clear |
| **用户写入** | send、revoke、delete |
| **管理写入** | mute、unmute、update、auto-clear |

### 权限说明

规则表里全部登记为 `type=login`，登录即可通过中间件；管理动作的判定在控制器内，走 `model.IsRootAdmin`（`root` 权限组且持有 `all` 规则）：

| 接口 | 门槛 |
| :--- | :--- |
| `info`、`messages`、`send`、`revoke`、`delete` | 登录 |
| `mute`、`unmute`、`mutes`、`update`、`auto-clear`（GET 与 PUT） | **仅超级管理员** |

非超级管理员调用管理接口返回 `403`，错误信息为「无权限：仅超级管理员可执行该操作！」。

### 数据表

| 表名 | 说明 |
| :--- | :--- |
| `chat_rooms` | 房间资料，只有 `id = 1` 一行 |
| `chat_room_messages` | 聊天室消息，`id` 即游标 `seq` |
| `chat_room_mutes` | 禁言记录，`room_id = 1` 为聊天室作用域，`room_id = 0` 为私信作用域 |
| `chat_room_message_deletes` | 「按人隐藏」记录，删除一条消息只写这里 |

### 字段说明

房间对象（`info` 的 `data.room`、`update` 的 `data`）：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 房间 ID，固定 `1` |
| `name` | string | 名称，最长 60 字符 |
| `description` | string | 简介，最长 255 字符 |
| `avatar` | string | 头像地址，最长 255 字符 |
| `status` | int | `1` 正常 / `0` 关闭；关闭后只能看历史、不能发言 |
| `create_time` | int64 | 创建时间戳 |
| `update_time` | int64 | 更新时间戳 |

消息对象（`send`、`revoke` 的 `data`，以及 `messages` 的数组元素）：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 消息 ID |
| `seq` | int | 与 `id` 相同，作为去重与断线补拉的游标 |
| `room_id` | int | 房间 ID |
| `uid` | int | 发送者 UID |
| `msg_type` | string | `text` / `image` |
| `content` | string | 文本内容或图片地址 |
| `is_revoked` | int | `0` 正常 / `1` 已撤回 |
| `revoke_uid` | int | 撤回操作人：本人或管理员 |
| `revoke_time` | int64 | 撤回时间戳，未撤回为 `0` |
| `client_msg_id` | string | 客户端幂等键 |
| `quote` | object | 引用快照，无引用为 `null`；结构见下表 |
| `create_time` | int64 | 发送时间戳 |
| `user` | object | 发送者公开资料：`id` / `nickname` / `avatar` / `description` / `title` |

引用快照 `quote`：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 被引用消息 ID |
| `uid` | int | 被引用消息的发送者 |
| `msg_type` | string | 被引用消息的类型 |
| `content` | string | 被引用消息的摘要快照 |

`user` 由服务端按批补齐（一次 `IN` 查询），前端不需要再按 `uid` 逐个查用户。

### 接口一览

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `/api/room/info` | 登录 | 房间资料与我的状态 |
| GET | `/api/room/messages` | 登录 | 历史消息 / 断线补拉 |
| POST | `/api/room/send` | 登录 | 发言 |
| POST | `/api/room/revoke` | 登录 | 撤回消息 |
| POST | `/api/room/delete` | 登录 | 删除消息（只对自己） |
| POST | `/api/room/mute` | 超级管理员 | 禁言 |
| POST | `/api/room/unmute` | 超级管理员 | 解除禁言 |
| GET | `/api/room/mutes` | 超级管理员 | 禁言名单 |
| PUT | `/api/room/update` | 超级管理员 | 修改房间资料 |
| GET | `/api/room/auto-clear` | 超级管理员 | 读取自动清空配置 |
| PUT | `/api/room/auto-clear` | 超级管理员 | 保存自动清空间隔 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 查询、发言、撤回、管理操作成功 |
| 400 | 请求错误 | 参数缺失、内容为空或超长、被禁言、房间已关闭、配置值不在选项内 |
| 401 | 未授权 | 未登录或 token 无效 |
| 403 | 禁止访问 | 非超级管理员调用管理接口，或撤回他人未过期消息 |
| 404 | 数据不存在 | 房间不存在 |
| 405 | 方法不允许 | 方法名不在允许表内，或对 `/api/room` 使用 `DELETE` |

`GET` 接口额外返回 `Cache-Control: no-store`，否则管理员改名后其他人仍看到旧名字。

---

## 接口列表

### 1. 房间资料与我的状态 [info]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/room/info`
- **权限**: 登录
- **说明**: 房间资料、是否管理员、是否被禁言

**请求参数**: 无

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.room` | object | 房间对象 |
| `data.is_admin` | bool | 当前登录用户是否超级管理员 |
| `data.muted` | bool | 当前登录用户是否处于禁言中 |
| `data.muted_until` | int64 | 解禁时间戳，`0` 表示永久禁言或未禁言 |
| `data.uid` | int | 当前登录用户 UID |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "room": {
      "id": 1,
      "name": "公共聊天室",
      "description": "全站用户都在这里",
      "avatar": "https://example.com/room/1.jpg",
      "status": 1,
      "create_time": 1761000000,
      "update_time": 1762876800
    },
    "is_admin": false,
    "muted": false,
    "muted_until": 0,
    "uid": 7
  }
}
```

`muted` 为 `true` 且 `muted_until` 为 `0` 时表示永久禁言；大于 `0` 时需按时间戳格式化为「MM-DD HH:mm」展示。

---

### 2. 历史消息 [messages]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/room/messages`
- **权限**: 登录
- **说明**: 两种用法——`before_id` 往前翻页，`after_seq` 断线补拉

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `before_id` | int | 否 | 只取 ID 小于该值的消息；`0` 或省略表示取最新一页 |
| `after_seq` | int | 否 | 只取 `seq` 大于该值的消息，用于断线补拉；大于 0 时优先于 `before_id` |
| `limit` | int | 否 | 每页条数，默认 30，上限 100 |

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.data` | array | 消息对象数组 |
| `data.has_more` | bool | 是否还有更多 |
| `data.last_seq` | int | 仅补拉模式返回，本批最大 `seq` |

翻页模式的响应：

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 9821,
        "seq": 9821,
        "room_id": 1,
        "uid": 7,
        "msg_type": "text",
        "content": "@林深 你上来一下",
        "is_revoked": 0,
        "revoke_uid": 0,
        "revoke_time": 0,
        "client_msg_id": "r-1762876800123",
        "quote": null,
        "create_time": 1762876801,
        "user": {
          "id": 7,
          "nickname": "青野",
          "avatar": "https://example.com/avatar/7.jpg",
          "description": "",
          "title": ""
        }
      }
    ],
    "has_more": true
  }
}
```

补拉模式的响应多一个 `last_seq`：

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [],
    "last_seq": 9821,
    "has_more": false
  }
}
```

---

### 3. 发言 [send]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/room/send`
- **权限**: 登录
- **说明**: 写入一条消息、全员广播，并按需推送 @ 提及

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `content` | string | 是 | 文本内容或图片地址，最长 2000 字符（按字符计） |
| `msg_type` | string | 否 | `text` / `image`，默认 `text` |
| `client_msg_id` | string | 否 | 客户端幂等键，最长 64 字符 |
| `quote_id` | int | 否 | 被引用消息 ID，`0` 表示不引用 |
| `mention_uids` | int[] | 否 | @ 提及的用户 UID 数组，最多 5 个 |

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data` | object | 消息对象，含 `user` |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "发送成功！",
  "data": {
    "id": 9822,
    "seq": 9822,
    "room_id": 1,
    "uid": 1,
    "msg_type": "text",
    "content": "@青野 收到，我看看",
    "is_revoked": 0,
    "revoke_uid": 0,
    "revoke_time": 0,
    "client_msg_id": "r-1762876900456",
    "quote": {
      "id": 9821,
      "uid": 7,
      "msg_type": "text",
      "content": "@林深 你上来一下"
    },
    "create_time": 1762876901,
    "user": {
      "id": 1,
      "nickname": "林深",
      "avatar": "https://example.com/avatar/1.jpg",
      "description": "写字的人",
      "title": ""
    }
  }
}
```

发言前会依次校验：房间存在且 `status = 1`、本人未被禁言、内容非空且不超长、`msg_type` 合法、图片地址合法、敏感词放行。任一不通过返回 `400`。

`mention_uids` 的规则：

| 规则 | 说明 |
| :--- | :--- |
| 去重与限量 | 服务端去重、剔除 `0` 与负数，只取前 5 个，多传的部分静默丢弃 |
| 剔除自己 | 提及自己不会产生任何推送 |
| 昵称核验 | 批量查出被提及用户的真实昵称，**正文必须包含「@昵称」**才推送；否则丢弃 |
| 推送方式 | 被提及者收到 `chat_mention` 事件，不进/离线不补 |

昵称核验是防刷的关键：客户端可以伪造 `mention_uids`，但无法伪造「正文里出现对方昵称」，两者不一致时不推送。

---

### 4. 撤回消息 [revoke]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/room/revoke`
- **权限**: 登录
- **说明**: 本人撤回限 2 分钟内；超级管理员不限时间与发送者

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `msg_id` | int | 是 | 消息 ID |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已撤回！",
  "data": {
    "id": 9822,
    "seq": 9822,
    "room_id": 1,
    "uid": 1,
    "msg_type": "text",
    "content": "",
    "is_revoked": 1,
    "revoke_uid": 1,
    "revoke_time": 1762876920,
    "client_msg_id": "r-1762876900456",
    "quote": null,
    "create_time": 1762876901,
    "user": {
      "id": 1,
      "nickname": "林深",
      "avatar": "https://example.com/avatar/1.jpg",
      "description": "写字的人",
      "title": ""
    }
  }
}
```

撤回只清内容、保留这一行，时间线不会凭空少一条。`revoke_uid` 记录实际操作人：本人撤回时等于发送者，管理员撤回时是管理员 UID。同时向全房广播 `room_revoke`。

---

### 5. 删除消息 [delete]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/room/delete`
- **权限**: 登录
- **说明**: 删除一条消息，只对本人生效

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `msg_id` | int | 是 | 消息 ID |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已删除！",
  "data": {
    "msg_id": 9822
  }
}
```

与撤回的区别：不删表、不改内容、不广播，只写一条隐藏记录。同账号的其它在线设备会收到推给自己的 `room_delete`，据此就地移除气泡。聊天室不支持整段删除，`DELETE /api/room` 返回 `405`，历史由自动清空任务处理。

---

### 6. 禁言 [mute]

- **接口类型**: 管理写入
- **请求方式**: `POST`
- **请求地址**: `/api/room/mute`
- **权限**: 超级管理员
- **说明**: 禁言指定用户，可设时长与原因

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 是 | 被禁言用户 UID |
| `minutes` | int | 否 | 禁言分钟数，`0` 表示永久，上限 525600（一年） |
| `reason` | string | 否 | 禁言原因，最长 120 字符 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已禁言！",
  "data": {
    "uid": 7,
    "until": 1762963200,
    "reason": "刷屏"
  }
}
```

限制条件：不能禁言自己、不能禁言其他超级管理员，两者都返回 `400`。`minutes` 为负数按 `0`（永久）处理，超过 525600 按上限截断。禁言成功后向全房广播 `room_mute`，被禁言者若正开着聊天室，输入框立刻变为「已禁言」。

---

### 7. 解除禁言 [unmute]

- **接口类型**: 管理写入
- **请求方式**: `POST`
- **请求地址**: `/api/room/unmute`
- **权限**: 超级管理员
- **说明**: 解除聊天室禁言

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 是 | 被禁言用户 UID |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已解除禁言！",
  "data": {
    "uid": 7
  }
}
```

本接口只解除聊天室作用域（`room_id = 1`）。需要同时解除私信禁言时用后台的 `chat-admin/unmute`，那里不传 `scope` 会两处一起解除。

---

### 8. 禁言名单 [mutes]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/room/mutes`
- **权限**: 超级管理员
- **说明**: 生效中的聊天室禁言记录，最多 100 条

**请求参数**: 无

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.data` | array | 禁言记录数组 |
| `data.data[].id` | int | 记录 ID |
| `data.data[].uid` | int | 被禁言用户 UID |
| `data.data[].until` | int64 | 解禁时间戳，`0` 表示永久 |
| `data.data[].reason` | string | 禁言原因 |
| `data.data[].operator_uid` | int | 操作人 UID |
| `data.data[].create_time` | int64 | 禁言时间戳 |
| `data.data[].user` | object | 被禁言用户公开资料 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 4,
        "uid": 7,
        "until": 0,
        "reason": "刷屏",
        "operator_uid": 1,
        "create_time": 1762876800,
        "user": {
          "id": 7,
          "nickname": "青野",
          "avatar": "https://example.com/avatar/7.jpg",
          "description": "",
          "title": ""
        }
      }
    ]
  }
}
```

已到期的记录不在列表里展示，但数据行保留，不做后台清理。

---

### 9. 修改房间资料 [update]

- **接口类型**: 管理写入
- **请求方式**: `PUT`
- **请求地址**: `/api/room/update`
- **权限**: 超级管理员
- **说明**: 修改名称、简介、头像、开关状态

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `name` | string | 否 | 名称，非空且最长 60 字符 |
| `description` | string | 否 | 简介，最长 255 字符 |
| `avatar` | string | 否 | 头像地址，最长 255 字符 |
| `status` | int | 否 | `1` 正常 / `0` 关闭 |

未传的字段保持原值。`name` 传空串返回 `400`「聊天室名称不能为空！」。

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "修改成功！",
  "data": {
    "id": 1,
    "name": "公共聊天室",
    "description": "随便聊聊",
    "avatar": "https://example.com/room/1.jpg",
    "status": 1,
    "create_time": 1761000000,
    "update_time": 1762876800
  }
}
```

修改成功后向全房广播 `room_update`，开着的页面立刻换掉标题与头像。

---

### 10. 读取自动清空配置 [auto-clear]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/room/auto-clear`
- **权限**: 超级管理员
- **说明**: 读取清空间隔、上次清空时间与下次清空时间

**请求参数**: 无

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.days` | int | 清空间隔（天） |
| `data.last_time` | int64 | 上次清空时间戳，`0` 表示尚未清空过 |
| `data.last_count` | int64 | 上次清空的条数 |
| `data.next_time` | int64 | 下次清空时间戳，`last_time` 为 `0` 时返回 `0` |
| `data.interval` | int64 | 间隔秒数，等于 `days * 86400` |
| `data.options` | int[] | 后台下拉可选值：`1` / `3` / `7` / `30` / `182` / `365` |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "days": 3,
    "last_time": 1762700000,
    "last_count": 512,
    "next_time": 1762959200,
    "options": [1, 3, 7, 30, 182, 365],
    "interval": 259200
  }
}
```

本接口只读配置，不触发删除。真正的清空由定时任务执行，按 `last_time + days` 判断是否到期，重启或多实例都不会重置节奏。

---

### 11. 保存自动清空间隔 [auto-clear]

- **接口类型**: 管理写入
- **请求方式**: `PUT`
- **请求地址**: `/api/room/auto-clear`
- **权限**: 超级管理员
- **说明**: 保存清空间隔（天），返回最新配置

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `days` | int | 是 | 清空间隔，只接受 `1` / `3` / `7` / `30` / `182` / `365` |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "保存成功！",
  "data": {
    "days": 7,
    "last_time": 1762700000,
    "last_count": 512,
    "next_time": 1763304800,
    "options": [1, 3, 7, 30, 182, 365],
    "interval": 604800
  }
}
```

`days` 不在选项内（含 `0`）返回 `400`，避免把「每隔 0 天清一次」写进库。保存不清空数据、不重置 `last_time`：换间隔只影响「从上次清空起算」的下一轮。

---

## 特殊说明

### 1. 实时广播与离线兜底

聊天室事件走 `/socket` 全员广播，与私信的定向推送分开：

| `type` | 触发时机 | 携带数据 |
| :--- | :--- | :--- |
| `room` | 有人发言 | 消息对象 |
| `room_revoke` | 撤回（本人或管理员） | 消息对象 |
| `room_delete` | 本人在其它设备删除 | `id`（私推给自己） |
| `room_update` | 房间资料变更 | 房间对象 |
| `room_mute` / `room_unmute` | 禁言 / 解禁 | `uid`、`until`、`reason` |
| `room_typing` | 有人正在输入 | `uid` |
| `chat_mention` | 有人 @ 了我 | `room_id`、`message_id`、`from_uid`、`preview` |

`room_typing` 与 `chat_mention` 是瞬时事件，不入库、不缓存，离线期间不补。事件字段与上行格式见 [WebSocket 文档](../socket.md)。

### 2. 在线成员

`/socket` 的 `status` 事件携带 `online_users`，每项含 `id`、`uid`、`name`、`ip`、`last_active`。跨域连接在鉴权升级后连接 ID 仍是 `guest_xxx`，**按 `uid` 而不是 `id` 聚合在线成员**，并跳过 `uid = 0` 的匿名连接。

### 3. 禁言与私信共用一张表

`chat_room_mutes` 通过 `room_id` 区分作用域：`1` 为聊天室，`0`（`ChatMuteDM`）为私信。前台 `room/mute` 只写聊天室作用域，后台 `chat-admin/mute` 一次写两条，两处一起失效。私信发送时会先查 `room_id = 0` 的禁言记录，被禁言者哪怕与对方互关也发不出去。

### 4. 自动清空

定时任务按间隔清空聊天室消息，私信不在清理范围内。`last_time` 为 `0` 时首次启动会填入当前时间，避免一上线就把历史清掉。

### 5. 相关文档

- [WebSocket 文档](../socket.md)
- [用户私聊 接口文档](./chat.md)
- [接口文档撰写规范](../接口文档撰写规范.md)
