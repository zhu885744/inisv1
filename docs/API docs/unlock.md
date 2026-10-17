# Unlock 接口文档

内容可见范围（登录可见 / 付费可见）的查询与积分兑换接口。

- 控制器：`app/api/controller/unlock.go`
- 判定与脱敏：`app/api/controller/scope.go`
- 字段口径、其余模块的返回约定：见 [content-scope.md](./content-scope.md)

---

## 接口概览

| 接口 | 方法 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| `/api/unlock/check` | GET | `type=common` | 查询内容可见状态（访客可调用） |
| `/api/unlock/pay` | POST | `type=login` | 使用积分兑换可见权限 |
| `/api/unlock/list` | GET | `type=login` | 我的兑换记录 |

---

## check - 查询可见状态

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `bind_type` | string | 是 | `article` / `moments` / `page`（`comment` 不支持兑换） |
| `bind_id` | int | 是 | 内容 ID |

**返回字段**：

| 字段 | 说明 |
| :--- | :--- |
| `scope` | 归一化后的可见范围：1 公开 / 2 登录可见 / 3 付费可见 |
| `scope_integral` | 兑换所需积分 |
| `scope_state` | `""` 可查看 / `login` 需登录 / `pay` 需兑换 |
| `can_view` | 当前是否可查看完整内容 |

**返回示例**：

```json
{
  "code": 200,
  "data": {
    "bind_type": "article",
    "bind_id": 12,
    "scope": 3,
    "scope_integral": 10,
    "scope_state": "pay",
    "can_view": false
  },
  "msg": "查询成功！"
}
```

---

## pay - 积分兑换可见权限

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `bind_type` | string | 是 | `article` / `moments` / `page` / `comment` |
| `bind_id` | int | 是 | 内容 ID |

**规则**：

- 仅 `scope=3` 的内容可兑换，其它范围返回「该内容无需付费查看！」；
- **评论不支持兑换**：`bind_type=comment` 直接返回「该内容不支持积分兑换！」（评论只有公开 / 登录可见两档）；
- 作者本人 / 管理员无需兑换（接口层已放行）；
- 已兑换过则**幂等返回**，不会重复扣积分（唯一索引 `uk_content_unlock(uid, bind_type, bind_id)`）；
- 扣款使用带余额条件的原子更新，积分不足整体回滚（返回「积分不足！」）；
- 扣除的积分转给内容作者，双方各记一条积分流水（`type=unlock`）；作者不存在时视为平台回收。

**返回字段**：`bind_type`、`bind_id`、`integral`（本次消耗）、`balance`（兑换后余额）、`unlocked`、`time`。

**返回示例**：

```json
{
  "code": 200,
  "data": {
    "bind_type": "article",
    "bind_id": 12,
    "integral": 10,
    "balance": 90,
    "unlocked": true,
    "time": 1776384000
  },
  "msg": "兑换成功！"
}
```

**常见失败**：未登录（401）、内容不存在 / 类型非法（400）、内容无需付费（202）、未设置兑换积分（202）、积分不足（202）、这是您自己的内容（202）。

---

## list - 我的兑换记录

**请求参数**：

| 参数 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页数量（受系统 `LIMIT` 限制） |
| `bind_type` | string | 否 | 按内容类型过滤 |
| `order` | string | 否 | 排序，默认 `create_time desc` |

**返回**：`data`（记录数组：`id`、`uid`、`bind_type`、`bind_id`、`integral`、`create_time`）、`count`、`page`。

---

## 权限登记

`app/model/auth-rules.go` 的 `unlock` 节点：

| 规则 | 类型 |
| :--- | :--- |
| `GET path=check` | `common`（详情页判断是否需要登录 / 兑换，访客必须可调） |
| `GET path=list` | `login` |
| `POST path=pay` | `login` |

> 升级部署后需要**重启一次后端**：权限规则种子只在启动时同步进数据库，否则新接口会按未登记处理返回 403。
