# Tip 积分打赏接口文档

积分打赏：用户把自己的积分转给「内容作者」或「站点用户」，支持留言。
一次打赏会在同一个事务里产生三件事（见 `app/model/tip.go`）：

1. 一条打赏记录（`inis_tips`）：谁打赏谁、多少积分、打赏的什么内容、留言；
2. 两条积分流水：打赏者 `-N`（`type=tip`）、接收者 `+N`（`type=tip-income`）；
3. 一条站内通知给接收者（`type=tip`，见 [notification.md](./notification.md)）。

- 控制器：`app/api/controller/tip.go`
- 模型：`app/model/tip.go`
- 参数校验：`app/validator/tip.go`
- 前端调用：`Mellow/src/api/tip.js`、`Mellow/src/components/RewardDialog.vue`

> 与「内容兑换」（[unlock.md](./unlock.md)）的区别：兑换是买查看权限、有幂等要求；
> 打赏是一次性赠予，**允许对同一内容重复打赏**，但同样用「带余额条件的原子更新」扣款，
> 并发下不会把余额扣成负数。

---

## 接口概览

| 接口 | 方法 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| `/api/tip/summary` | GET | `type=common` | 某内容的打赏概览（次数 / 积分合计）+ 我的余额与上下限 |
| `/api/tip/stats` | GET | `type=common` | 某用户收到的打赏统计（用户主页） |
| `/api/tip/list` | GET | `type=login` | 打赏记录：某内容的打赏墙 / 我收到 / 我发出 |
| `/api/tip/send` | POST | `type=login` | 打赏（扣当前用户积分） |

`PUT` / `DELETE` 不开放：打赏记录不可修改、不可删除（返回 405）。

---

## 绑定类型（bind_type）

| bind_type | bind_id | 接收者 |
| :--- | :--- | :--- |
| `article` | 文章 ID | 文章作者 |
| `moments` | 动态 ID | 动态作者 |
| `comment` | 评论 ID | 评论作者 |
| `page` | 独立页面 ID | 页面作者 |
| `user` | 用户 ID | 该用户本人（用户主页打赏入口） |

- 内容类打赏的接收者**由后端按内容作者解析**，前端传的 `to_uid` 只做一致性校验，
  因此「传别人的 uid 把钱打给别人」是做不到的；
- 内容已删除 / 不存在时返回「打赏对象不能为空！」或 `ScopeContentInfo` 的错误文案；
- `bind_type` 为空时按 `user` 处理（兼容只传 `bind_id` 的旧调用）。

## 打赏约束

| 常量 | 值 | 说明 |
| :--- | :--- | :--- |
| `TipMinValue` | 1 | 单次最少积分 |
| `TipMaxValue` | 100000 | 单次最多积分（防误操作 / 防刷） |
| `TipMessageMaxLength` | 100 | 留言最大字数（按 rune 计） |

---

## summary - 内容的打赏概览

- **请求方式**: `GET`
- **请求路径**: `/api/tip/summary`
- **权限**: `type=common`（访客可看次数与积分，余额为 0）

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `bind_type` | string | 是 | 见上方绑定类型表 |
| `bind_id` | int | 是 | 内容 ID（`bind_type=user` 时为被 打赏用户 uid） |

**返回字段**：

| 字段 | 说明 |
| :--- | :--- |
| `bind_type` / `bind_id` | 回显请求的绑定对象 |
| `count` | 该内容累计被打赏次数 |
| `integral` | 该内容累计被打赏积分 |
| `balance` | **当前登录用户**的积分余额（未登录为 0） |
| `min` / `max` | 单次打赏积分下限 / 上限 |
| `message_max` | 留言最大字数 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "bind_type": "article",
    "bind_id": 12,
    "count": 3,
    "integral": 46,
    "balance": 120,
    "min": 1,
    "max": 100000,
    "message_max": 100
  }
}
```

**错误**：

| code | msg | 场景 |
| :--- | :--- | :--- |
| 400 | `bind_type 不能为空！` | 未传 `bind_type` |

---

## stats - 用户收到的打赏统计

- **请求方式**: `GET`
- **请求路径**: `/api/tip/stats`
- **权限**: `type=common`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | 否 | 目标用户；不传时取当前登录用户 |

**返回字段**：

| 字段 | 说明 |
| :--- | :--- |
| `uid` | 目标用户 ID |
| `count` | 收到打赏的次数 |
| `integral` | 收到打赏的积分合计 |
| `tippers` | 打赏过的人数（按 `uid` 去重） |
| `recent` | 最近 8 条打赏（头像墙用），见下表 |

`recent[]` 字段：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 打赏记录 ID |
| `uid` / `nickname` / `avatar` | 打赏者 |
| `value` | 打赏积分 |
| `message` | 留言（未填为空串） |
| `time` | 打赏时间（秒级时间戳） |

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "uid": 3,
    "count": 12,
    "integral": 260,
    "tippers": 7,
    "recent": [
      {
        "id": 88,
        "uid": 9,
        "nickname": "小明",
        "avatar": "/attachment/avatar/9.webp",
        "value": 20,
        "message": "写得好",
        "time": 1760000000
      }
    ]
  }
}
```

**错误**：

| code | msg | 场景 |
| :--- | :--- | :--- |
| 400 | `请指定用户！` | 未登录且未传 `uid` |

---

## list - 打赏记录

- **请求方式**: `GET`
- **请求路径**: `/api/tip/list`
- **权限**: `type=login`

三种用法（优先级从上到下）：

| 用法 | 参数 | 说明 |
| :--- | :--- | :--- |
| 某内容的打赏墙 | `bind_type` + `bind_id` | 公开，留言原样展示 |
| 我发出的打赏 | `box=sent`（或 `give`） | 只返回 `uid=当前用户` 的记录 |
| 我收到的打赏 | 不传 `box` | 默认 `to_uid=当前用户`；传 `uid` 可看他人，此时**留言会被隐藏**（管理员除外） |

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `bind_type` | string | 否 | 内容类型，配合 `bind_id` 查某内容 |
| `bind_id` | int | 否 | 内容 ID |
| `box` | string | 否 | `sent` / `give` 表示「我发出的」 |
| `uid` | int | 否 | 查看某用户收到的打赏（非本人时隐藏留言） |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页数量，默认 10，受站点「分页限制」配置约束 |
| `order` | string | 否 | 排序，默认 `id desc` |

**返回字段**：

| 字段 | 说明 |
| :--- | :--- |
| `data` | 打赏记录数组 |
| `count` | 总条数 |
| `page` | **总页数**（`ceil(count / limit)`，沿用基础接口的字段名，注意不是当前页码） |

`data[]` 字段：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 记录 ID |
| `uid` / `to_uid` | 打赏者 / 接收者 |
| `value` | 打赏积分 |
| `bind_type` / `bind_id` | 打赏对象（用户主页打赏为 `user` + 接收者 uid） |
| `message` | 留言（无权查看时为空串） |
| `time` | 打赏时间（秒级时间戳） |
| `from` | `{ id, nickname, avatar }` 打赏者信息 |
| `to` | `{ id, nickname, avatar }` 接收者信息 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "data": [
      {
        "id": 88,
        "uid": 9,
        "to_uid": 3,
        "value": 20,
        "bind_type": "article",
        "bind_id": 12,
        "message": "写得好",
        "time": 1760000000,
        "from": { "id": 9, "nickname": "小明", "avatar": "/attachment/avatar/9.webp" },
        "to": { "id": 3, "nickname": "站长", "avatar": "/attachment/avatar/3.webp" }
      }
    ],
    "count": 1,
    "page": 1
  }
}
```

**错误**：

| code | msg | 场景 |
| :--- | :--- | :--- |
| 401 | `请先登录！` | 既没有 `bind_type+bind_id`，也没有登录态 |

> 无数据时仍返回 200，`data.data` 为空数组、`count=0`（前端按空态展示即可）。

---

## send - 打赏

- **请求方式**: `POST`
- **请求路径**: `/api/tip/send`
- **权限**: `type=login`

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `bind_type` | string | 是 | `article` / `moments` / `comment` / `page` / `user` |
| `bind_id` | int | 是 | 内容 ID（`user` 时为接收者 uid） |
| `value` | int | 是 | 打赏积分，`1 ~ 100000` |
| `message` | string | 否 | 留言，最多 100 字 |
| `to_uid` | int | 否 | 接收者 uid，仅用于校验；不传则由后端按内容作者补全 |

**处理流程**（单个事务）：

1. 校验积分范围与留言长度；
2. `TipTarget` 按 `bind_type + bind_id` 解析接收者，与 `to_uid` 不一致直接拒绝；
3. 校验「不能打赏自己」；
4. 条件更新 `integral >= value` 扣打赏者积分，`RowsAffected=0` 即余额不足并回滚；
5. 给接收者加积分，写两条积分流水（`tip` / `tip-income`）；
6. 事务提交后**异步**发站内通知（`type=tip`），通知失败不影响打赏结果。

**返回字段**：

| 字段 | 说明 |
| :--- | :--- |
| `id` | 打赏记录 ID |
| `uid` / `to_uid` | 打赏者 / 接收者 |
| `value` | 本次打赏积分 |
| `bind_type` / `bind_id` | 打赏对象 |
| `balance` | 打赏后**我**的余额 |
| `to_balance` | 打赏后**接收者**的余额 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "打赏成功，感谢支持！",
  "data": {
    "id": 89,
    "uid": 9,
    "to_uid": 3,
    "value": 20,
    "bind_type": "article",
    "bind_id": 12,
    "balance": 100,
    "to_balance": 280
  }
}
```

**错误**（均为 400，`msg` 为具体原因）：

| msg | 场景 |
| :--- | :--- |
| `请先登录！` | 未登录（401） |
| `bind_type 不能为空！` / `bind_id 不能为空！` | 参数缺失 |
| `打赏积分不能少于 1 分！` / `单次打赏不能超过 100000 分！` | 超出上下限 |
| `打赏留言不能超过 100 个字！` | 留言过长 |
| `积分不足！` | 余额不够（并发安全） |
| `不能打赏自己！` | `to_uid == 当前用户` |
| `接收者与打赏对象不一致！` | `to_uid` 与内容作者不符 |
| `不支持的打赏类型！` | `bind_type` 不在上表内 |
| `用户不存在！` | `bind_type=user` 且目标用户不存在 |
| `打赏对象不能为空！` | `bind_id <= 0` |

---

## 与通知 / 积分流水的联动

- **站内通知**：接收者收到 `type=tip`、`bind_type/bind_id` 指向被打赏内容的通知，
  正文为「{昵称} 打赏了你 N 积分（{对象}），积分已存入你的账户。」，有留言时追加「留言：…」；
  点击跳转规则见 [notification.md](./notification.md) 与前端 `Mellow/src/utils/notification.js`。
- **积分流水**：打赏者 `type=tip`（负值）、接收者 `type=tip-income`（正值），
  在 `/user/integral` 的「积分明细」中可直接筛选到，描述分别为「打赏{对象}」「收到打赏（{对象}）」。
- **打赏统计**：内容页用 `summary`，用户主页用 `stats`；打赏墙用 `list`。
