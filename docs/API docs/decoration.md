# Decoration 装扮接口文档

## 接口概述

`decoration` 控制器提供**装扮**（头像框 / 头衔等）的商城浏览、佩戴卸载、兑换，
以及后台的增删改、批量发放与回收。

### 权限说明

权限由 `app/model/auth-rules.go` 的 `decoration` 节点登记，分三类：

| `type` | 含义 | 未满足时的表现 |
| :--- | :--- | :--- |
| `common` | 免登录 | —— |
| `login` | 登录即可 | 中间件返回 **401** |
| `default` | 需权限点 | 控制器内 `meta.permit` 判定，**403** `无权限：当前账号未被授予该权限点！` |

> `default` 是「**登录 + 权限点**」双重要求：未登录时会被中间件先拦成 401，
> 不会走到控制器里的 403。

### 可写字段白名单

```
type, name, description, preview, payload, price_type, price, rarity,
duration, unlock, category, status, sort, is_default, stock, min_exp, json, text
```

> ⚠️ `limit_per_user`（每人限购）**已废弃**，不在白名单内，传了也不会生效。

### 排序白名单（`order`）

只有以下值生效，**非法值会被静默回退**成 `sort desc, id desc`（不报错）：
`sort desc, id desc`、`sort asc, id asc`、`price desc, id desc`、`price asc, id asc`、
`create_time desc`、`create_time asc`、`id desc`、`id asc`。

### 方法总表

| 方法 | 动词 | 描述 | 权限 |
| :--- | :--- | :--- | :--- |
| `types` | GET | 装扮类型元数据 + 配置 | 公开 |
| `config` | GET | 装扮配置 | 公开 |
| `shop` | GET | 商城列表（仅上架） | 公开 |
| `one` | GET | 装扮详情 | 公开 |
| `all` | GET | 装扮列表（管理员可看全量） | 公开（含管理分支） |
| `count` | GET | 装扮统计 | `default` |
| `mine` | GET | 我的装扮（拥有 + 默认，按类型分组） | 登录 |
| `wearing` | GET | 我当前佩戴的装扮 | 登录 |
| `user-decorations` | GET | 指定用户的装扮（后台排查用） | `default` |
| `wear` | POST | 佩戴装扮 | 登录 |
| `unwear` | POST | 卸下某类型装扮 | 登录 |
| `buy` | POST | 兑换装扮（扣积分发放） | 登录 |
| `save` | POST | 保存（无 `id` → 新增 / 有 `id` → 更新） | `default` |
| `create` | POST | 新增装扮 | `default` |
| `update` | PUT | 更新装扮 | `default` |
| `grant` | PUT | **批量**发放装扮 | `default` |
| `revoke` | PUT | **批量**回收装扮 | `default` |
| `restore` | PUT | 从回收站恢复 | `default` |
| `remove` | DELETE | 软删除（进回收站） | `default` |
| `delete` | DELETE | 彻底删除（不可逆） | `default` |
| `clear` | DELETE | 清空装扮回收站（**无参数**） | `default` |

> 所有写操作（POST / PUT / DELETE）在派发后会异步清一次缓存（`go this.delCache()`）——
> **写完立刻读 GET 可能命中旧缓存**。

---

## 状态码规范

| 状态码 | 说明 | 使用场景 |
| :--- | :--- | :--- |
| **200** | 请求成功 | 正常返回（业务码在 body 的 `code`） |
| **202** | 业务拒绝 | `wear` / `unwear` / `buy` 的**业务失败**（如积分不足、已售罄），前端直接弹 msg |
| **204** | 无内容 | 查询无数据；批量操作时「无可操作数据」 |
| **400** | 请求错误 | 参数缺失、类型不支持、入库失败 |
| **401** | 未授权 | 未登录（`login` 接口） |
| **403** | 无权限 | 权限点未授予（`default` 接口） |
| **404** | 资源不存在 | `user-decorations` 的目标用户不存在 |
| **405** | 方法不允许 | method 不在 allow |

> ⚠️ **`202` 在这个项目里表示「业务拒绝」**（不是「已接受」），前端要按 `code` 弹提示；
> 而「商城未开启」更特殊：返回 **200** + `data = null`。

---

## 接口列表

### 1. GET 请求接口

#### 1.1 装扮类型元数据 [业务接口]

- **路径**: `/api/decoration/types`　**方法**: `GET`　**权限**: 公开
- **描述**: 返回装扮类型定义（名称 / 图标 / 挂载位 / 可配字段）与装扮配置

```json
{
  "types": [
    { "type": "avatar_frame", "name": "头像框", "icon": "bi-person-badge", "slot": "avatar",
      "allow_custom_text": 0, "fields": ["image", "scale", "animated"], "description": "展示在头像外圈，可随时切换。" },
    { "type": "title", "name": "头衔", "slot": "title", "allow_custom_text": 1,
      "fields": ["text", "color", "bg", "border", "glow", "icon", "animated"], "description": "..." }
  ],
  "config": { "enabled": 1, "show_in_goods": 1, "currency": "integral",
              "avatar_frame": { "enabled": 1, "frame_scale": 1.2 },
              "title": { "enabled": 1, "allow_custom": 1, "max_length": 10, "presets": [] } }
}
```

#### 1.2 装扮配置 [业务接口]

- **路径**: `/api/decoration/config`　**方法**: `GET`　**权限**: 公开
- **描述**: 返回装扮配置对象（**直接就是配置本体**，不是 `{config:...}` 包裹）

#### 1.3 商城列表 [业务接口]

- **路径**: `/api/decoration/shop`　**方法**: `GET`　**权限**: 公开

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `type` | string | 否 | 只看某装扮类型；空 = 全部 |
| `price_type` | string | 否 | `free` 只看免费 / `paid` 只看付费 / 空或 `all` = 全部 / 其他值按 `price_type` 精确匹配 |

- 装扮商城未开启时不报错：**200** + `data = null` + `装扮商城未开启！`
- 列表项额外带计算字段：`payload`(map)、`is_default`、`price`、`stock`、`unlimited`、
  `sold_out`、`duration`、`permanent`、`on_sale`、`buy_reason`

```json
{ "code": 200, "msg": "数据请求成功！", "data": {
  "data": [ { "id": 3, "type": "title", "name": "掌门", "price": 300, "price_type": "integral",
              "stock": -1, "unlimited": true, "sold_out": false, "duration": 0,
              "permanent": true, "on_sale": true, "buy_reason": "", "is_default": false } ],
  "count": 1, "types": [ "..." ] } }
```

#### 1.4 装扮详情 [业务接口]

- **路径**: `/api/decoration/one`　**方法**: `GET`　**权限**: 公开

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `id` / `type` / `status` | int / string | 否 | 仅这三个 key 参与过滤 |
| `field` | string | 否 | 字段裁剪（逗号分隔） |

- 无数据 **204** `无数据！`；命中 **200** `数据请求成功！`

#### 1.5 装扮列表 [业务接口]

- **路径**: `/api/decoration/all`　**方法**: `GET`　**权限**: 公开（含管理分支）

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `page` | int | 否 | 页码，默认 1 |
| `limit` | int | 否 | 每页条数，默认取 `this.meta.limit(ctx)`（默认 10，可被配置 `SYSTEM_PAGE_LIMIT` 覆盖） |
| `order` | string | 否 | 排序，仅白名单生效，默认 `sort desc, id desc` |
| `status` | string | 否 | **仅在有权限点时生效**，否则强制只看上架的 |
| `type` | string | 否 | 装扮类型 |
| `price_type` | string | 否 | 同 `shop` 语义 |
| `keyword` | string | 否 | 按 `name` 模糊匹配 |
| `onlyTrashed` | bool | 否 | 只看回收站 |
| `withTrashed` | bool | 否 | 含回收站 |
| `field` | string | 否 | 字段裁剪 |

- 响应 `{data, count, page}`，`page` 为**总页数**；无数据 **204**，否则 **200**

#### 1.6 装扮统计（后台）[业务接口]

- **路径**: `/api/decoration/count`　**方法**: `GET`　**权限**: `default`

```json
{ "code": 200, "msg": "查询成功！", "data": {
  "total": 12, "on_sale": 10, "owned": 35,
  "by_type": { "avatar_frame": 5, "title": 7 } } }
```

#### 1.7 我的装扮 [业务接口]

- **路径**: `/api/decoration/mine`　**方法**: `GET`　**权限**: 登录（仅本人，无 `uid` 参数）
- **描述**: 拥有 + 默认的装扮，按类型分组；响应 `{types, wearing, list}`

#### 1.8 我佩戴的装扮 [业务接口]

- **路径**: `/api/decoration/wearing`　**方法**: `GET`　**权限**: 登录
- **描述**: 按类型返回当前佩戴的装扮（`{type: 装扮}` 的 map，未佩戴则为 `{}`）

#### 1.9 指定用户的装扮（后台）[业务接口]

- **路径**: `/api/decoration/user-decorations`　**方法**: `GET`　**权限**: `default`

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uid` | int | **是** | 目标用户ID；`<=0` 返回 400 |

- 用户不存在 → **404** `用户不存在！`
- 响应在 `MyDecorations(uid)` 基础上挂 `user`（**只含 `id / account / nickname / avatar`**，
  刻意裁剪以防泄露敏感字段）

---

### 2. POST 请求接口

#### 2.1 佩戴装扮 [业务接口]

- **路径**: `/api/decoration/wear`　**方法**: `POST`　**权限**: 登录

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `decoration_id` | int | **是** | 装扮ID；`<=0` 返回 400 |
| `text` | string | 否 | 纯样式头衔的自定义文字（装扮自带文字时忽略） |

- 成功：`{wearing: {...}}` + `佩戴成功！`
- **业务失败返回 `202`**，msg 来自模型层（如 `装扮不存在！`、`你还没有这个装扮！`）

#### 2.2 卸下装扮 [业务接口]

- **路径**: `/api/decoration/unwear`　**方法**: `POST`　**权限**: 登录

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `type` | string | **是** | 装扮类型；空返回 400 |

- 成功：`{wearing: {...}}` + `已卸下！`；业务失败同样是 **202**

#### 2.3 兑换装扮 [业务接口]

- **路径**: `/api/decoration/buy`　**方法**: `POST`　**权限**: 登录

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `decoration_id` | int | **是** | 商城装扮ID；`<=0` 返回 400 |

```json
{ "code": 200, "msg": "兑换成功！", "data": {
  "decoration_id": 3, "name": "掌门", "type": "title",
  "price": 300, "integral": 1200, "expire_time": 0, "duration": 0 } }
```

- **业务拒绝返回 `202`** + msg：`装扮商城未开启！`、`装扮不存在！`、`装扮已下架！`、
  `积分不足！`、`已售罄！`、`已达限购上限！` 等（文案定义在 `app/model/decoration.go`）

#### 2.4 保存装扮（后台）[业务接口]

- **路径**: `/api/decoration/save`　**方法**: `POST`　**权限**: `default`
- **描述**: 无 `id` 走新增、有 `id` 走更新（等价于 `create` / `update`）

#### 2.5 新增装扮（后台）[业务接口]

- **路径**: `/api/decoration/create`　**方法**: `POST`　**权限**: `default`
- **参数**: 上面的可写字段白名单（`type`、`name` 必填）

| code | msg | 触发条件 |
| :--- | :--- | :--- |
| 400 | `type 不能为空！` | `type` 空 |
| 400 | `不支持的装扮类型：xxx` | 类型不在元数据里 |
| 400 | `name 不能为空！` | `name` 空 |

- 成功：`{id}` + `创建成功！`

---

### 3. PUT 请求接口

#### 3.1 更新装扮（后台）[业务接口]

- **路径**: `/api/decoration/update`　**方法**: `PUT`　**权限**: `default`
- **参数**: `id`（必填）+ 可写字段白名单；成功 `{id}` + `更新成功！`

#### 3.2 批量发放装扮（后台）[业务接口]

- **路径**: `/api/decoration/grant`　**方法**: `PUT`　**权限**: `default`

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| `uids` | 数组 / 逗号串 / 单值 | **是** | 目标用户，多个 |
| `decoration_ids` | 数组 / 逗号串 / 单值 | **是** | 装扮，多个 |
| `duration` | int64 | 否 | 有效期（秒）：`>0` 指定秒数；**`0` = 沿用装扮自身有效期**；**`<0` = 永久** |
| `source` | string | 否 | 来源标记，默认 `admin` |

- 目标用户 × 装扮做**笛卡尔积**发放，逐条失败只计数不中断
- 成功 **200** `发放完成！` + `{success, failed}`

#### 3.3 批量回收装扮（后台）[业务接口]

- **路径**: `/api/decoration/revoke`　**方法**: `PUT`　**权限**: `default`
- **参数**: `uids`、`decoration_ids`（必填，同上）
- 正在佩戴的会先卸下；拥有记录**软删除**（便于续期幂等）；成功 `回收完成！` + `{success, failed}`

#### 3.4 恢复装扮 [业务接口]

- **路径**: `/api/decoration/restore`　**方法**: `PUT`　**权限**: `default`
- **参数**: `ids`（必填）；只能恢复软删除的记录

---

### 4. DELETE 请求接口

| 方法 | 路径 | 参数 | 行为 |
| :--- | :--- | :--- | :--- |
| `remove` | `/api/decoration/remove` | `ids`（必填） | **软删除**（进回收站），同时把关联商品置为下架 |
| `delete` | `/api/decoration/delete` | `ids`（必填） | **彻底删除**（不可逆）：硬删装扮 + 清空拥有记录（含正在佩戴的）+ 商品下架 |
| `clear` | `/api/decoration/clear` | **无** | 清空整个装扮回收站 |

统一表现：非权限点 → 403；`ids` 空 → 400 `ids 不能为空！`；可操作集合为空 → **204**
`无可操作数据！`；成功 `{ids: [...]}` + `删除成功！` / `清空成功！`。

---

## 注意事项

1. **HTTP 状态码恒为 200**，业务码在 body；`202` = 业务拒绝、`204` = 无数据，
   前端不要只判 200；
2. **权限是「登录 + 权限点」双轨**：`default` 接口未登录会被中间件拦成 401，
   而不是控制器里的 403；文案也不同（`无权限！` vs `无权限：当前账号未被授予该权限点！`）；
3. **分页 `limit` 不可写死**：默认 10 且可被配置覆盖；`page` 返回的是**总页数**；
   不要传 `limit=0`；
4. **`order` 非法值被静默吞掉**：不报错，直接回退默认排序；
5. **`grant` 的 `duration` 语义反直觉**：`0` ≠ 永久（`<0` 才是永久）；
6. **批量接口只回计数不回来明细**：`{success, failed}`，失败原因整体丢失；
7. **`remove` / `delete` / `restore` / `clear` 极易混淆**（软删 / 硬删 / 恢复 / 清空）；
8. **`limit_per_user` 已废弃**，不在写白名单；
9. **写操作清缓存是异步的**：写完立刻读可能命中旧缓存；
10. **`mine` / `wearing` 是「仅本人」接口**，按 uid 查别人要走 `user-decorations`（管理员）。

## 相关文档

| 文档 | 内容 |
| :--- | :--- |
| `docs/API docs/goods.md` | 商品 / 装扮商城（装扮可同步为商品） |
| `docs/API docs/integral.md` | 积分（兑换扣减积分） |
| `docs/API docs/users.md` | 用户管理（装扮发放记录可在用户详情查看） |
