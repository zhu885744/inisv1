# Integral API 文档

## 接口概述

Integral 控制器负责用户积分管理。积分是独立于经验值（EXP）的货币体系，用户通过任务（签到、登录、发布内容等）赚取积分，并用积分在积分商城兑换商品。

**接口类型**：业务接口（无通用 CRUD）

### 积分规则

积分规则通过 `SYSTEM_INTEGRAL_RULES` 配置项进行管理，定义各任务可获得的积分及每日上限。以下为默认规则：

| 任务类型 | 名称 | 单次积分 | 每日限制次数 | 图标 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `check-in` | 每日签到 | 5 | 1 | `bi-calendar-check` | 每日签到 |
| `login` | 每日登录 | 2 | 1 | `bi-box-arrow-in-right` | 每日首次登录 |
| `article-create` | 发布文章 | 10 | 5 | `bi-file-earmark-text` | 发布文章（自动触发） |
| `comment` | 发表评论 | 2 | 10 | `bi-chat-dots` | 发表评论（自动触发） |
| `moments` | 发布动态 | 20 | 1 | `bi-lightning` | 发布动态（自动触发） |
| `share` | 分享内容 | 2 | 3 | `bi-share` | 分享文章/页面/动态（自动触发） |

**配置方式**：管理员可通过配置 API 修改 `SYSTEM_INTEGRAL_RULES` 的 JSON 值来调整规则，修改后立即生效。

> 规则字段说明：`name` 名称、`value` 单次积分、`daily_limit` 每日上限（**`0` 表示不限制**，不再拦截）、`icon` 前端图标类名（可选，缺省返回 `bi-coin`）。

### 流水类型（type）

| type | 方向 | 说明 |
| :--- | :--- | :--- |
| `check-in` / `login` / `article-create` / `comment` / `moments` / `share` | 收入 | 任务奖励 |
| `give` | 收入/支出 | 管理员调整 |
| `buy` | 支出 | 积分商城兑换 |
| `refund` | 收入 | 订单取消退款返还 |
| `card` | 收入 | 卡密兑换积分 |

---

## 状态码规范

| 状态码 | 使用场景 |
|--------|----------|
| 200 | 接口调用成功且数据返回正常 |
| 202 | 业务逻辑处理完成但有特殊状态（如积分不足、已达上限） |
| 204 | 无数据 |
| 400 | 请求参数错误 |
| 401 | 未登录 |
| 403 | 无权限（非管理员） |
| 405 | 方法不存在 |

---

## 接口列表

### 1. 积分概览 [业务接口]

**请求方式**：GET
**请求路径**：`/api/integral/status`

**说明**：查询当前登录用户的积分概览（余额 + 累计收支 + 今日收支），需登录

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "integral": 125,
    "total_income": 420,
    "total_expense": 295,
    "today_income": 25,
    "today_expense": 100
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| integral | int | 当前积分余额 |
| total_income | int | 累计获得积分 |
| total_expense | int | 累计消耗积分 |
| today_income | int | 今日获得积分 |
| today_expense | int | 今日消耗积分 |

> 兼容说明：早期版本仅返回 `integral`，本次升级为向后兼容的字段扩展。

### 2. 积分明细 [业务接口]

**请求方式**：GET
**请求路径**：`/api/integral/all`

**说明**：查询当前登录用户的积分流水（需登录，仅返回自己的），支持多条件筛选并返回区间收支合计

**请求参数**：

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 系统配置 | 每页数量 |
| order | string | 否 | create_time desc | 排序（白名单：create_time desc/asc、value desc/asc、id desc/asc） |
| type | string | 否 | - | 流水类型：check-in / login / article-create / comment / moments / share / buy / refund / give |
| direction | string | 否 | - | 收支方向：`income` 只看收入 / `expense` 只看支出 |
| start | int64 | 否 | - | 起始时间戳（秒） |
| end | int64 | 否 | - | 结束时间戳（秒） |
| keyword | string | 否 | - | 描述关键词（模糊匹配） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {
    "data": [
      {
        "id": 12,
        "uid": 1,
        "value": -100,
        "type": "buy",
        "description": "兑换商品：会员月卡",
        "json": { "goods_id": 3, "balance_after": 125 },
        "create_time": 1750982400
      }
    ],
    "count": 1,
    "page": 1,
    "summary": {
      "income": 0,
      "expense": 100,
      "net": -100
    }
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| data[].value | int | 积分变动值（正=获得，负=消耗） |
| data[].json.balance_after | int | 变动后余额快照（便于对账） |
| summary.income | int | 当前筛选条件下收入合计 |
| summary.expense | int | 当前筛选条件下支出合计 |
| summary.net | int | 净变动（收入 - 支出） |

### 3. 今日任务进度 [业务接口]

**请求方式**：GET
**请求路径**：`/api/integral/tasks`

**说明**：查询当前登录用户的今日任务完成情况（需登录），用于「做任务赚积分」面板

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "list": [
      {
        "type": "check-in",
        "name": "每日签到",
        "icon": "bi-calendar-check",
        "value": 5,
        "daily_limit": 1,
        "today_count": 1,
        "today_income": 5,
        "progress": 100,
        "done": true,
        "remain": 0
      },
      {
        "type": "comment",
        "name": "发表评论",
        "icon": "bi-chat-dots",
        "value": 2,
        "daily_limit": 10,
        "today_count": 3,
        "today_income": 6,
        "progress": 30,
        "done": false,
        "remain": 14
      }
    ],
    "today_income": 11,
    "done_count": 1,
    "total_count": 6,
    "streak": 4
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| list[].today_count | int | 今日已获得该奖励的次数 |
| list[].today_income | int | 今日该任务已获得积分 |
| list[].progress | int | 进度百分比（不限制的任务固定 100） |
| list[].done | bool | 今日是否已达上限 |
| list[].remain | int | 还能获得的积分（用于「还可 +N」提示） |
| today_income | int | 今日获得积分合计 |
| done_count / total_count | int | 已完成任务数 / 任务总数 |
| streak | int | 连续签到天数 |

### 4. 积分排行榜 [业务接口]

**请求方式**：GET
**请求路径**：`/api/integral/rank`

**说明**：公开接口，无需登录。返回积分排行榜与当前登录用户的排名（未登录时 `my_rank`、`my_value` 为 0）

**请求参数**：

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| by | string | 否 | earned | `earned` 累计获得榜 / `balance` 当前余额榜 |
| limit | int | 否 | 20 | 返回条数（最大 100） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "by": "earned",
    "list": [
      {
        "rank": 1,
        "id": 3,
        "nickname": "张三",
        "avatar": "https://...",
        "description": "这个人很懒",
        "title": "掌门",
        "value": 1280,
        "is_me": false
      }
    ],
    "my_rank": 12,
    "my_value": 420
  }
}
```

### 5. 积分任务规则 [业务接口]

**请求方式**：GET
**请求路径**：`/api/integral/rules`

**说明**：获取积分任务规则列表（公共接口，无需登录），返回各行为可获得的积分及每日上限

**响应字段**（数组项）：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| type | string | 任务类型（check-in/login/article-create 等） |
| name | string | 任务名称 |
| value | int | 单次获得的积分 |
| daily_limit | int | 每日限制次数（0 表示不限制） |
| icon | string | 前端图标类名（如 `bi-calendar-check`） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": [
    { "type": "check-in", "name": "每日签到", "value": 5, "daily_limit": 1, "icon": "bi-calendar-check" },
    { "type": "login", "name": "每日登录", "value": 2, "daily_limit": 1, "icon": "bi-box-arrow-in-right" }
  ]
}
```

### 6. 调整积分 [业务接口-管理员专用]

**请求方式**：POST
**请求路径**：`/api/integral/give`

**说明**：管理员手动给指定用户增加或扣除积分，不受每日规则限制，直接修改用户积分余额。**支持单个（`uid`）与批量（`uids`）两种方式**。

**权限要求**：仅 root 用户（管理员）可调用

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| uid | int | 二选一 | 目标用户ID（单个） |
| uids | array | 二选一 | 目标用户ID数组（批量，优先于 uid） |
| value | int | 是 | 积分数量（正数为增加，负数为扣除，不能为0） |
| description | string | 否 | 操作描述 |

**单个用户响应**：

```json
{
  "code": 200,
  "msg": "调整成功！",
  "data": {
    "uid": 1,
    "value": 100,
    "integral": 225
  }
}
```

**批量发放响应**（逐个处理，失败不影响其它用户）：

```json
{
  "code": 200,
  "msg": "批量调整完成！",
  "data": {
    "total": 3,
    "success": 2,
    "failed": 1,
    "list": [
      { "uid": 1, "success": true, "msg": "成功", "balance": 225 },
      { "uid": 2, "success": true, "msg": "成功", "balance": 100 },
      { "uid": 999, "success": false, "msg": "用户不存在", "balance": 0 }
    ]
  }
}
```

**失败响应**：

| 状态码 | 说明 |
|--------|------|
| 400 | 参数错误（uid/uids 为空、value 为0） |
| 403 | 无权限（非管理员调用） |
| 202 | 业务失败（如扣减时积分不足） |

> 说明：管理员调整积分成功后，系统会向目标用户发送一条 `system`（系统通知）类型的消息通知（增加/扣除各一条），详见 [Notification API 文档](notification.md)。

### 7. 生成卡密 [业务接口-管理员专用]

**请求方式**：POST
**请求路径**：`/api/integral/card-generate`

**说明**：管理员批量生成积分卡密，用户可凭卡密自助兑换积分。卡密使用密码学安全随机数生成，字符集剔除易混淆字符（`0/1/I/O`），默认长度 16 位。

**权限要求**：仅管理员（`root` 权限点）可调用

**请求参数**：

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| value | int | 是 | - | 积分面额（1 ~ 1000000） |
| count | int | 否 | 1 | 生成数量（1 ~ 1000） |
| length | int | 否 | 16 | 卡密长度（**不小于 8**，最大 64） |
| expire_time | int64 | 否 | 0 | 卡密有效期（秒级时间戳，**0 表示永久有效**） |
| expire | string | 否 | - | 卡密有效期（日期字符串，支持 `2006-01-02` 或 `2006-01-02 15:04:05`；仅日期时到当天 23:59:59 失效）。与 `expire_time` 二选一，`expire_time` 优先 |
| remark | string | 否 | - | 备注（便于按批次管理） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "生成成功！",
  "data": {
    "batch": "20260920153012ABCDEF",
    "count": 2,
    "value": 100,
    "expire_time": 1798905599,
    "cards": ["2K7M9PQ4XZAH3TUV", "B4WX8YRJ2M6NPQST"],
    "list": [
      { "id": 1, "card": "2K7M9PQ4XZAH3TUV", "value": 100, "status": 0, "batch": "20260920153012ABCDEF", "expire_time": 1798905599, "uid": 0, "use_time": 0 }
    ]
  }
}
```

> 安全提示：`cards` 为卡密明文，仅在本接口返回一次，请管理员妥善保管并自行导出下发。

**失败响应**：

| 状态码 | 说明 |
|--------|------|
| 400 | 参数错误（面额为0/超限、数量超限、长度小于 8、有效期早于当前时间或格式错误） |
| 403 | 无权限（非管理员调用） |

### 8. 卡密兑换积分 [业务接口-需登录]

**请求方式**：POST
**请求路径**：`/api/integral/card-redeem`

**说明**：登录用户凭卡密兑换积分，兑换成功后积分立即到账并写入 `card` 类型流水。同一张卡密在并发场景下仅能被兑换一次（数据库条件更新 + 影响行数校验）。

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| card | string | 是 | 卡密（自动忽略空格/连字符并转为大写，容错输入） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "兑换成功！",
  "data": {
    "card_id": 1,
    "value": 100,
    "batch": "20260920153012ABCDEF",
    "integral": 325
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| card_id | int | 卡密记录ID |
| value | int | 本次兑换的积分面额 |
| batch | string | 所属批次号 |
| integral | int | 兑换后的积分余额 |

**失败响应**：

| 状态码 | 说明 |
|--------|------|
| 400 | `card` 为空 |
| 401 | 未登录 |
| 429 | 失败次数过多，临时锁定（默认连续失败 10 次锁定 10 分钟，防暴力枚举） |
| 202 | 业务失败（卡密不存在 / 已被使用 / 已过期 / 格式不正确） |

### 9. 卡密列表 [业务接口-管理员专用]

**请求方式**：GET
**请求路径**：`/api/integral/card-all`

**权限要求**：仅管理员（`root` 权限点）可调用

**请求参数**：

| 参数名 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 系统配置 | 每页数量 |
| order | string | 否 | id desc | 排序（白名单：id、value、create_time、use_time 的 asc/desc） |
| status | int | 否 | - | 状态：0 未使用 / 1 已使用 |
| batch | string | 否 | - | 批次号 |
| uid | int | 否 | - | 使用该卡密的用户ID |
| value | int | 否 | - | 积分面额 |
| keyword | string | 否 | - | 卡密模糊搜索 |
| expired | int | 否 | - | 1 仅已过期 / 0 仅未过期（含永久） |
| field | string/array | 否 | - | 字段过滤 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "数据请求成功！",
  "data": {
    "data": [
      {
        "id": 1, "card": "2K7M9PQ4XZAH3TUV", "value": 100, "status": 1,
        "batch": "20260920153012ABCDEF", "expire_time": 0,
        "uid": 3, "nickname": "张三", "use_time": 1798905600, "create_time": 1798800000
      }
    ],
    "count": 1,
    "page": 1
  }
}
```

> 说明：列表项中的 `nickname` 为使用者昵称（后端一次性批量补充，未使用的卡密为空字符串），管理端应优先展示昵称而非 `uid`。

### 10. 卡密统计 [业务接口-管理员专用]

**请求方式**：GET
**请求路径**：`/api/integral/card-stats`

**权限要求**：仅管理员（`root` 权限点）可调用

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "total": 100,
    "unused": 80,
    "used": 20,
    "expired": 5,
    "value_total": 10000,
    "value_used": 2000
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| total | int | 卡密总数 |
| unused | int | 未使用且未过期数量 |
| used | int | 已使用数量 |
| expired | int | 已过期未使用数量 |
| value_total | int | 累计发放积分面额 |
| value_used | int | 累计已兑换积分 |

### 11. 删除卡密 [业务接口-管理员专用]

**请求方式**：DELETE
**请求路径**：`/api/integral/card-remove`（软删除）、`/api/integral/card-delete`（彻底删除）

**权限要求**：仅管理员（`root` 权限点）可调用

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| ids | array/string | 是 | 卡密ID列表，支持数组或在字符串中用任意分隔符（后端正则提取数字） |

**响应示例**：

```json
{ "code": 200, "msg": "删除成功！", "data": { "ids": [1, 2, 3] } }
```

### 12. 导出未使用卡密 [业务接口-管理员专用]

**请求方式**：GET
**请求路径**：`/api/integral/card-export`

**说明**：导出全部「**未使用且未过期**」的卡密明文，用于管理端批量复制 / 下发。单次最多导出 10000 张，超出部分不会返回，并通过 `truncated` 标记。

**权限要求**：仅管理员（`root` 权限点）可调用

> 安全提示：该接口返回卡密明文，属敏感操作，后端会记录操作日志（操作人、导出数量、是否截断）以便审计。

**请求参数**（均为可选，用于按条件筛选导出范围）：

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| batch | string | 否 | 仅导出指定批次 |
| value | int | 否 | 仅导出指定积分面额 |
| keyword | string | 否 | 卡密模糊匹配 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "导出成功！",
  "data": {
    "count": 2,
    "truncated": false,
    "cards": ["2K7M9PQ4XZAH3TUV", "B4WX8YRJ2M6NPQST"]
  }
}
```

| 字段名 | 类型 | 说明 |
|--------|------|------|
| count | int | 本次导出的卡密数量 |
| truncated | bool | 是否因超出单次上限（10000）而被截断 |
| cards | array | 卡密明文列表（未使用且未过期） |

**失败响应**：

| 状态码 | 说明 |
|--------|------|
| 400 | 导出失败（数据库异常等） |
| 403 | 无权限（非管理员调用） |

---

## 卡密数据模型（inis_integral_card）

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | int | 主键，自增 |
| card | varchar(64) | 卡密（**唯一索引**，字符集剔除 `0/1/I/O`） |
| value | int | 积分面额 |
| status | tinyint | 状态：0 未使用 / 1 已使用 |
| batch | varchar(32) | 批次号 |
| expire_time | int64 | 过期时间戳（**0 表示永久有效**） |
| uid | int | 使用该卡密的用户ID |
| use_time | int64 | 使用时间戳 |
| remark | varchar(255) | 备注 |
| create_time / update_time / delete_time | int64 | 公共字段（软删除） |

---

## 特殊说明

### 积分获取方式

积分通过以下行为自动获得（均由后端在对应业务触发点自动发放）：

| 行为 | 触发位置 | 积分类型 |
|------|----------|----------|
| 签到 | `exp.go` `checkIn` | `check-in` |
| 登录 | `comm.go` `loginExp` | `login` |
| 发布文章 | `article.go` `create` | `article-create` |
| 发表评论 | `comment.go` `create` | `comment` |
| 发布动态 | `moments.go` `create` | `moments` |
| 分享内容 | `exp.go` `share` | `share`（可扩展） |
| 卡密兑换 | `integral.go` `cardRedeem` | `card` |

### 积分消耗方式

积分通过在积分商城兑换商品消耗（详见 [Goods API 文档](goods.md)）：
- 购买时由 `Goods.Buy` 在**事务**中扣减积分并写入 `buy` 类型流水
- 取消订单时由 `GoodsOrder.CancelOrder` 在事务中退还积分并写入 `refund` 类型流水

### 积分变动通知

管理员通过 `give` 接口增加/扣除用户积分成功后，系统会调用 `model.IntegralGiveNotify` 向目标用户发送一条消息通知（增加/扣除各对应一个标题）。该通知**归入系统通知（`type=system`）**，不单独占用通知类型。**积分变动通知仅此一个触发场景**，任务奖励、商城消耗、卡密兑换等其它积分变动均不发送通知。

### 卡密安全机制

- 卡密使用 `crypto/rand`（密码学安全随机数）生成，默认 16 位、字符集 32 个（剔除易混淆字符），单卡熵约 80 bit，无法被枚举猜测；
- 卡密字段带**唯一索引**，从数据库层面防止重复；
- 兑换接口需登录，并通过「状态条件更新 + 影响行数判断」保证同一张卡密并发下只被兑换一次；
- 兑换失败累计达 10 次将锁定 10 分钟（基于缓存），防止暴力枚举；
- 卡密明文仅在生成接口返回一次，积分流水中记录的是脱敏后的卡密。

### 每日上限规则

- `daily_limit > 0`：当日该类型奖励次数达到上限后，`Add` 返回「今日奖励已达上限！」
- `daily_limit = 0`：不限制次数（升级前 `0` 会被误判为「已达上限」，现已修复）

### 权限控制

- 普通用户只能查询自己的积分余额、明细、任务进度
- 排行榜与任务规则为公开接口
- 只有管理员可以调整用户积分（`give`）、生成/查询/删除卡密（`card-generate`/`card-all`/`card-stats`/`card-remove`/`card-delete`）
- 登录用户可凭卡密自助兑换积分（`card-redeem`）

### 权限规则（auth-rules）

`integral` 控制器各接口在 `app/model/auth-rules.go` 中登记，规则类型（type）说明：

| type | 含义 | 接口 |
| :--- | :--- | :--- |
| `common` | 公开接口，无需登录 | `rules`、`rank` |
| `login` | 需登录 | `status`、`all`、`tasks`、`card-redeem` |
| `root` | 需管理员权限点 | `give`、`card-generate`、`card-all`、`card-stats`、`card-export`、`card-remove`、`card-delete` |

> 新增接口必须在 `createAuthRules()` 的 `integral`/`goods` 分组中登记，否则中间件会走「默认」分支要求权限点，普通用户将直接收到 403。
