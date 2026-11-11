# 用户私聊 接口文档

## 接口概述

`chat` 控制器用于点对点私信：会话列表、历史记录、发送、撤回、删除、拍一拍、送达确认、已读回执与会话偏好。所有写操作最终收敛到 `model.ChatMessages.Send` 一个入口，WebSocket 私聊走同一条链路，因此两条通道发出的消息在历史记录与未读数上没有差异。

### 接口类型说明

本文档不含基础接口（`one` / `all` / `save` 等），全部为业务接口：

| 接口类型 | 说明 |
| :--- | :--- |
| **查询接口** | conversations、friends、messages、sync、unread、status |
| **写入接口** | send、revoke、delete、pat、ack、read、pref |

### 权限说明

全部接口登记为 `type=login`，登录即可调用，不设权限点。私信的发送门槛不在权限规则里，而是由 `model.ChatPermission` 统一判定，HTTP 与 WebSocket 共用：

| 判定顺序 | 条件 | 结果 |
| :--- | :--- | :--- |
| 1 | 自己 → 自己 | 拒绝 |
| 2 | 私信作用域被禁言（`chat_room_mutes.room_id = 0`） | 拒绝，带解禁时间 |
| 3 | 任一方向存在拉黑关系 | 拒绝 |
| 4 | 不是互相关注 | 拒绝 |

「能否发送」的结论可以先用 [私聊可用性 [status]](#6-私聊可用性-status) 查询，前端据此决定「发私信」按钮是否可点。

### 数据表

| 表名 | 说明 |
| :--- | :--- |
| `chat_conversations` | 两人一段会话，`key` 为「小 UID_大 UID」，唯一索引保证同一对用户只有一行 |
| `chat_messages` | 私信消息本体，`id` 即游标 `seq` |
| `chat_states` | 每人每会话的 `last_ack_seq` / `last_read_seq` 水位 |
| `chat_conversation_prefs` | 每人每会话的 `pinned` / `muted` 偏好 |
| `chat_message_deletes` | 「按人隐藏」记录，删除一条消息只写这里 |

### 字段说明

消息对象（`send`、`revoke`、`pat` 的 `data`，以及 `messages` / `sync` 的数组元素）：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 消息 ID |
| `seq` | int | 与 `id` 相同，作为去重与断线补拉的游标 |
| `conversation_id` | int | 所属会话 ID |
| `from_uid` | int | 发送者 UID |
| `to_uid` | int | 接收者 UID |
| `msg_type` | string | `text` / `image` / `pat` |
| `content` | string | 文本内容、图片地址；`pat` 时为拍一拍后缀 |
| `is_read` | int | `0` 未读 / `1` 已读 |
| `is_revoked` | int | `0` 正常 / `1` 已撤回 |
| `revoke_time` | int64 | 撤回时间戳，未撤回为 `0` |
| `client_msg_id` | string | 客户端幂等键 |
| `quote` | object | 引用快照，无引用为 `null`；结构见下表 |
| `create_time` | int64 | 发送时间戳 |

引用快照 `quote`：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 被引用消息 ID |
| `uid` | int | 被引用消息的发送者 |
| `msg_type` | string | 被引用消息的类型 |
| `content` | string | 被引用消息的摘要快照 |

引用存的是快照而非仅 `quote_id`：被引用消息撤回后内容会被清空，只存 ID 会让引用气泡变成空白。

会话对象（`conversations` 的 `data.data` 元素）：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | int | 会话 ID |
| `other_uid` | int | 对方 UID |
| `last_msg_id` | int | 最后一条消息 ID，`0` 表示本人侧已无可见消息 |
| `last_text` | string | 最后一条消息摘要：图片为「[图片]」、拍一拍为「[拍一拍]」、表情标签替换为「[表情]」，正文超过 60 字符截断加省略号 |
| `last_uid` | int | 最后一条消息的发送者 |
| `last_time` | int64 | 最后一条消息时间戳 |
| `unread` | int | 我在这段会话的未读数 |
| `pinned` | bool | 是否置顶 |
| `muted` | bool | 是否免打扰 |
| `user` | object | 对方公开资料：`id` / `nickname` / `avatar` / `description` / `title` |
| `online` | bool | 对方是否在线（WebSocket 连接态，非数据库字段） |

### 接口一览

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `/api/chat/conversations` | 登录 | 我的会话列表 |
| GET | `/api/chat/friends` | 登录 | 互关好友列表 |
| GET | `/api/chat/messages` | 登录 | 与某人的历史消息 |
| GET | `/api/chat/sync` | 登录 | 断线重连补拉 |
| GET | `/api/chat/unread` | 登录 | 私信未读总数 |
| GET | `/api/chat/status` | 登录 | 能否与某人私聊 |
| POST | `/api/chat/send` | 登录 | 发送私信 |
| POST | `/api/chat/revoke` | 登录 | 撤回私信 |
| POST | `/api/chat/delete` | 登录 | 删除私信（只对自己） |
| POST | `/api/chat/pat` | 登录 | 拍一拍 |
| PUT | `/api/chat/ack` | 登录 | 确认送达水位 |
| PUT | `/api/chat/read` | 登录 | 标记已读 |
| PUT | `/api/chat/pref` | 登录 | 设置会话置顶 / 免打扰 |

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| 200 | 请求成功 | 查询、发送、撤回等操作成功 |
| 400 | 请求错误 | 参数缺失、内容为空、超长、幂等冲突以外的业务校验失败 |
| 401 | 未授权 | 未登录或 token 无效 |
| 403 | 禁止访问 | 存在拉黑关系、不是互关好友、被禁言 |
| 404 | 数据不存在 | 目标用户不存在 |
| 405 | 方法不允许 | 方法名不在允许表内，或对 `/api/chat` 使用 `DELETE` |

`GET` 接口额外返回 `Cache-Control: no-store`，避免「已标已读但角标仍是旧数字」。

---

## 接口列表

### 1. 我的会话列表 [conversations]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/conversations`
- **权限**: 登录
- **说明**: 返回本人参与的全部会话，置顶优先、其余按最后消息时间倒序

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `limit` | int | 否 | 返回条数，默认 20，上限 50；超限或非法值回落 20 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 12,
        "other_uid": 7,
        "last_msg_id": 3381,
        "last_text": "明天见",
        "last_uid": 7,
        "last_time": 1762876800,
        "unread": 2,
        "pinned": true,
        "muted": false,
        "user": {
          "id": 7,
          "nickname": "林深",
          "avatar": "https://example.com/avatar/7.jpg",
          "description": "写字的人",
          "title": ""
        },
        "online": true
      }
    ]
  }
}
```

服务端为置顶会话多取一批候选再排序截断：置顶会话的 `last_time` 可能很旧，只按时间取前 `limit` 条会让「置顶」失效。最后一条消息被本人删除时，`last_text` 会回退到本人仍可见的上一条，对方看到的摘要不受影响。

---

### 2. 互关好友列表 [friends]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/friends`
- **权限**: 登录
- **说明**: 列出互相关注的用户，供会话列表合并出「还没聊过」的入口

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `limit` | int | 否 | 返回人数，默认 200，上限 200 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "uid": 9,
        "user": {
          "id": 9,
          "nickname": "青野",
          "avatar": "https://example.com/avatar/9.jpg",
          "description": "",
          "title": ""
        },
        "online": false
      }
    ]
  }
}
```

没有互关好友时 `data.data` 为空数组，`msg` 为「无数据！」，`code` 仍是 200。

---

### 3. 与某人的历史消息 [messages]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/messages`
- **权限**: 登录
- **说明**: 倒序取一页、正序返回；首屏（不传 `before_id`）会顺带把对方发来的消息标记已读

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `before_id` | int | 否 | 只取 ID 小于该值的消息，`0` 或省略表示取最新一页 |
| `limit` | int | 否 | 每页条数，默认 20，上限 50 |

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.data` | array | 消息对象数组，按 ID 升序 |
| `data.has_more` | bool | 是否还有更早的消息 |
| `data.target` | object | 对方公开资料 |
| `data.can_send` | bool | 当前是否允许发送 |
| `data.can_send_tip` | string | 不允许发送时的原因文案，可发送时为空串 |
| `data.online` | bool | 对方是否在线 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 3381,
        "seq": 3381,
        "conversation_id": 12,
        "from_uid": 7,
        "to_uid": 1,
        "msg_type": "text",
        "content": "明天见",
        "is_read": 1,
        "is_revoked": 0,
        "revoke_time": 0,
        "client_msg_id": "c-1762876790123",
        "quote": null,
        "create_time": 1762876800
      }
    ],
    "has_more": true,
    "target": {
      "id": 7,
      "nickname": "林深",
      "avatar": "https://example.com/avatar/7.jpg",
      "description": "写字的人",
      "title": ""
    },
    "can_send": true,
    "can_send_tip": "",
    "online": true
  }
}
```

即使 `can_send` 为 `false` 也照常返回历史：拉黑或取消互关后聊天记录不应凭空消失，只是不能再发。

---

### 4. 断线重连补拉 [sync]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/sync`
- **权限**: 登录
- **说明**: 客户端带着本地最大 `seq` 回来，服务端只返回更大的消息，并带回双方水位

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `after_seq` | int | 否 | 本地已有的最大 `seq`，默认 0 表示全量补拉 |
| `limit` | int | 否 | 每批条数，默认 50，上限 100 |

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.data` | array | 消息对象数组，按 ID 升序 |
| `data.last_seq` | int | 本批最后一条的 `seq`，无数据时为本会话最大值 |
| `data.has_more` | bool | 是否还有未补拉的消息 |
| `data.last_ack_seq` | int | 我在这段会话的送达水位 |
| `data.last_read_seq` | int | 我在这段会话的已读水位 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [],
    "last_seq": 3381,
    "has_more": false,
    "last_ack_seq": 3381,
    "last_read_seq": 3381
  }
}
```

`has_more` 为 `true` 时需用返回的 `last_seq` 作为下一次 `after_seq` 继续拉，直到为 `false`。

---

### 5. 私信未读总数 [unread]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/unread`
- **权限**: 登录
- **说明**: 全部会话的未读消息合计，用于导航角标

**请求参数**: 无

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "count": 3
  }
}
```

---

### 6. 私聊可用性 [status]

- **接口类型**: 查询接口
- **请求方式**: `GET`
- **请求地址**: `/api/chat/status`
- **权限**: 登录
- **说明**: 判断能否给对方发私信，供作者主页的「发私信」按钮决定是否可点

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "can_send": false,
    "reason": "仅互相关注的好友可以私信！",
    "online": true
  }
}
```

与发送接口的差异：这里对「不可发送」仍返回 `code: 200`，原因放在 `reason`；对方不存在时返回 `404`，与 `messages` 一致。

---

### 7. 发送私信 [send]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/chat/send`
- **权限**: 登录
- **说明**: 落库并通过 WebSocket 推给在线的对方，离线时由历史记录兜底

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `content` | string | 是 | 文本内容或图片地址，最长 2000 字符（按字符计，非字节） |
| `msg_type` | string | 否 | `text` / `image`，默认 `text`；`pat` 只能走 [拍一拍 [pat]](#10-拍一拍-pat) |
| `client_msg_id` | string | 否 | 客户端生成的幂等键，最长 64 字符 |
| `quote_id` | int | 否 | 被引用消息 ID，`0` 表示不引用 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "发送成功！",
  "data": {
    "id": 3382,
    "seq": 3382,
    "conversation_id": 12,
    "from_uid": 1,
    "to_uid": 7,
    "msg_type": "text",
    "content": "好的，明天见",
    "is_read": 0,
    "is_revoked": 0,
    "revoke_time": 0,
    "client_msg_id": "c-1762876900456",
    "quote": {
      "id": 3381,
      "uid": 7,
      "msg_type": "text",
      "content": "明天见"
    },
    "create_time": 1762876901
  }
}
```

`msg_type` 传其他值返回 `400`「不支持的消息类型！」。`quote_id` 会校验三件事：消息存在、未撤回、属于当前会话，任一不满足返回 `400`。`client_msg_id` 重复提交不产生新消息，直接返回首次入库的那条。

---

### 8. 撤回私信 [revoke]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/chat/revoke`
- **权限**: 登录
- **说明**: 撤回本人发送的消息，限发送后 2 分钟内

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
    "id": 3382,
    "seq": 3382,
    "conversation_id": 12,
    "from_uid": 1,
    "to_uid": 7,
    "msg_type": "text",
    "content": "",
    "is_read": 0,
    "is_revoked": 1,
    "revoke_time": 1762876920,
    "client_msg_id": "c-1762876900456",
    "quote": null,
    "create_time": 1762876901
  }
}
```

撤回只清内容、保留这一行，双方界面显示「撤回了一条消息」。超过时间窗、非本人消息、已撤回的消息返回 `400`。对方在线时同时收到 `chat_revoke` 事件。

---

### 9. 删除私信 [delete]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/chat/delete`
- **权限**: 登录
- **说明**: 删除一条消息，只对本人生效，对方完全无感

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
    "msg_id": 3382
  }
}
```

不删表、不广播、不推送对方，只写一条 `chat_message_deletes` 隐藏记录。同一账号的其它在线设备会收到推给自己的 `chat_delete`，据此就地移除气泡。私信不支持整段会话删除，`DELETE /api/chat` 返回 `405`。

---

### 10. 拍一拍 [pat]

- **接口类型**: 写入接口
- **请求方式**: `POST`
- **请求地址**: `/api/chat/pat`
- **权限**: 登录
- **说明**: 往会话里写一条 `pat` 类型的系统提示并推给对方

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `suffix` | string | 否 | 后缀文案，如「的后脑勺」，最长 20 字符；不传为「拍了拍你」 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已拍一拍！",
  "data": {
    "id": 3383,
    "seq": 3383,
    "conversation_id": 12,
    "from_uid": 1,
    "to_uid": 7,
    "msg_type": "pat",
    "content": "的后脑勺",
    "is_read": 0,
    "is_revoked": 0,
    "revoke_time": 0,
    "client_msg_id": "",
    "quote": null,
    "create_time": 1762877000
  }
}
```

同一对用户两次拍一拍的间隔不得小于 5 秒，过快返回 `400`「拍得太快啦，歇一下再拍！」。`msg_type=pat` 只能通过本接口产生，`send` 会拒绝该类型，否则可以绕过这里的节流。

---

### 11. 确认送达水位 [ack]

- **接口类型**: 写入接口
- **请求方式**: `PUT`
- **请求地址**: `/api/chat/ack`
- **权限**: 登录
- **说明**: 批量确认「已收到 `seq` 及之前的消息」，水位只增不减

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `seq` | int | 是 | 本批最大 `seq` |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已确认！",
  "data": {
    "last_ack_seq": 3383
  }
}
```

送达不等于已读：`ack` 只写 `chat_states.last_ack_seq`，已读另走 [标记已读 [read]](#12-标记已读-read)。两者分开才能区分「收到但没看」。

---

### 12. 标记已读 [read]

- **接口类型**: 写入接口
- **请求方式**: `PUT`
- **请求地址**: `/api/chat/read`
- **权限**: 登录
- **说明**: 把对方发来的未读消息标记为已读，并返回最新未读总数

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `seq` | int | 否 | 本地已读到的 `seq`，大于服务端记录时用于推进水位 |

**响应字段**:

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `data.updated` | int | 本次实际标记的行数 |
| `data.last_msg_id` | int | 已读到的最大消息 ID |
| `data.unread` | int | 标记完成后的未读总数 |

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "已标记为已读！",
  "data": {
    "updated": 2,
    "last_msg_id": 3383,
    "unread": 1
  }
}
```

响应里直接带回 `unread`，客户端可覆盖角标，不必再发一次 `GET /api/chat/unread`。`updated` 大于 0 时服务端会给对方发 `chat_read` 回执。

---

### 13. 设置会话置顶 / 免打扰 [pref]

- **接口类型**: 写入接口
- **请求方式**: `PUT`
- **请求地址**: `/api/chat/pref`
- **权限**: 登录
- **说明**: 设置本人对某个会话的置顶与免打扰，只影响自己

**请求参数**:

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `target_uid` | int | 是 | 对方 UID |
| `pinned` | bool | 否 | 是否置顶，`true` 置顶、`false` 取消 |
| `muted` | bool | 否 | 是否免打扰，`true` 开启、`false` 关闭 |

`pinned` 与 `muted` 至少传一个，两个都不传返回 `400`「没有需要修改的内容！」。未传的字段保持原值：只传 `pinned` 时，已经开启的 `muted` 不会被清掉。

**成功响应** (200):

```json
{
  "code": 200,
  "msg": "设置成功！",
  "data": {
    "conversation_id": 12,
    "pinned": 1,
    "muted": 0
  }
}
```

`pinned` / `muted` 返回 `0` 或 `1` 的整型。两者语义不同：

| 开关 | 影响范围 |
| :--- | :--- |
| `pinned` | 会话列表排序：置顶会话排在最前，组内仍按最后消息时间倒序 |
| `muted` | 仅抑制提示音、悬浮提醒与桌面通知；**未读数照常累计**，会话列表角标不变 |

双方还没有产生任何消息时无法设置，返回 `400`「你们还没有会话，先聊一句再设置！」。

---

## 特殊说明

### 1. 实时通道与离线兜底

HTTP 负责落库与历史查询，实时送达走 `/socket`。私信相关的下行事件：

| `type` | 触发时机 | 携带数据 |
| :--- | :--- | :--- |
| `private` | 新私信 / 拍一拍 | 消息对象 |
| `chat_read` | 对方标记已读 | `by_uid`、`last_msg_id` |
| `chat_revoke` | 对方或后台撤回 | 消息对象 |
| `chat_delete` | 本人在其它设备删除 | `id` |
| `chat_typing` | 对方正在输入 | `from_uid` |

离线期间的事件不入库、不缓存，客户端重连后用 [断线重连补拉 [sync]](#4-断线重连补拉-sync) 补齐。事件字段与上行格式见 [WebSocket 文档](../socket.md)。

### 2. 幂等与游标

`client_msg_id` 是幂等键，作用域为「同一发送者」：连点、网络重试、WebSocket 重发都只会落一条。`seq` 直接复用自增主键，全局单调，不需要额外的每会话计数器；`after_seq` 与 `before_id` 都基于它翻页。

### 3. 内容校验与审核

- 文本与拍一拍后缀都过全站敏感词库，命中 `block` / `limit` 直接拒绝，命中 `review` / `warn` / `log` 放行但留证，后台在聊天审核里人工判断。
- 图片消息的 `content` 只放行 `http(s)` 与站内相对路径，不接受空白、引号、尖括号，避免把脏值带进会话摘要、日志与后台列表。
- 撤回只清内容不清行；后台撤回不限时间与发送者，走 `chat-admin/revoke`。

### 4. 相关文档

- [WebSocket 文档](../socket.md)
- [聊天室 接口文档](./room.md)
- [接口文档撰写规范](../接口文档撰写规范.md)
