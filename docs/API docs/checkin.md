# Checkin API 文档

## 接口概述

Checkin 控制器负责**每日签到**（`/api/checkin/*`）。签到原先寄生在经验模块里
（`/api/exp/check-in*` + `SYSTEM_EXP_RULES` 的 `check-in` 项 + `inis_exp` 流水），
奖励只有「经验 / 积分」两种，连签加成与里程碑也只能靠两个固定字段拼出来。

现在它独立成一个模块，并由**奖励引擎**发放奖励：

- **配置独立**：`SYSTEM_CHECKIN_RULES`（后台「签到管理」页可视化编辑），与经验 / 积分规则完全解耦；
- **记录独立**：新表 `inis_checkin`，一天一行、`(uid, date)` 唯一索引，天然防重复签到；
  连签天数一次查询即可算出（不再逐天回溯经验流水）；
- **奖励多样**：基础奖励 / 周期奖励（第 N 天）/ 连签加成 / 里程碑 / 月全勤 / 随机奖励（带概率与区间），
  每条奖励项都能选**资产**（内置经验、积分，可扩展注册新资产）；
- **机制增强**：每日重置时间可配（如凌晨 4 点算新一天）、补签（消耗积分）、签到文案池、站内信通知。

**接口类型**：业务接口（无通用 CRUD）

### 接口列表

| 方法 | 路径 | 权限 | 说明 |
| :--- | :--- | :--- | :--- |
| GET | `/api/checkin/status` | 需登录 | 签到状态（连签 / 周期 / 今日可得 / 里程碑 / 月进度 / 补签） |
| GET | `/api/checkin/calendar` | 需登录 | 签到日历（按月，含补签标记） |
| GET | `/api/checkin/rank` | 公开 | 签到排行榜（按时间范围统计次数 / 经验 / 积分） |
| GET | `/api/checkin/rules` | 公开 | 签到规则（开关 / 周期奖励表 / 里程碑 / 资产清单） |
| POST | `/api/checkin/sign` | 需登录 | 签到 |
| POST | `/api/checkin/makeup` | 需登录 | 补签（消耗积分） |

---

## 状态码规范

| 状态码 | 使用场景 |
| :--- | :--- |
| 200 | 成功 |
| 202 | 业务未通过（今天已签到、签到已关闭、补签超出范围、积分不足 …），`msg` 为具体原因 |
| 401 | 未登录 |
| 405 | 方法不存在 |

---

## 签到配置（`SYSTEM_CHECKIN_RULES`）

配置存在 `config` 表的 `SYSTEM_CHECKIN_RULES` 键（JSON），读写走通用的
`GET /api/config/one?key=SYSTEM_CHECKIN_RULES` 与 `POST /api/config/save`（仅管理员，root）。
保存后缓存立即失效，无需重启。

### 完整字段

| 字段 | 类型 | 默认 | 说明 |
| :--- | :--- | :--- | :--- |
| `enabled` | int | `1` | 总开关（`0` 关闭，前台提示未开放） |
| `name` | string | `每日签到` | 名称，用于流水描述与站内信标题 |
| `reset_hour` | int | `0` | 每日重置时间点（0-23）。设为 `4` 表示凌晨 4 点前算前一天 |
| `notice` | int | `0` | 签到成功后是否发站内信 |
| `tips` | array | 5 条 | 签到文案池，随机取一条 |
| `base` | array | 经验 10 + 积分 5 | **基础奖励**：每次签到都发放 |
| `streak` | object | 见下 | **连续签到加成** |
| `cycle` | object | 7 天 | **周期奖励**（连签第 N 天额外发放） |
| `milestones` | array | 7 / 15 / 30 天 | **里程碑**：连续天数达到档位时一次性发放（跨越判定，见下） |
| `monthly` | array | 20 / 28 天 | **月累计（全勤）**：当月签到天数达到档位时一次性发放 |
| `random` | array | 10% 概率 20 积分 | **随机奖励**：按概率 / 区间额外发放 |
| `makeup` | object | 见下 | **补签** |

### 奖励项写法

`base`、`cycle.days[].rewards`、`milestones[].rewards`、`monthly[].rewards`、`random`
里每一项都是「奖励项」：

```json
{ "asset": "exp", "value": 10 }
```

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `asset` | string | 资产标识：`exp`（经验）、`integral`（积分），或二次开发注册的自定义资产 |
| `value` | int | 固定数值（与 `min` / `max` 二选一） |
| `min` / `max` | int | 随机区间（都填时在区间内随机取值） |
| `chance` | int | 触发概率（0-100），`0` 或 `100` 表示必得 |
| `label` | string | 备注 / 标签（仅展示用） |

三种写法混用都可以：

```json
"base": [
  { "asset": "exp", "value": 10 },
  { "asset": "integral", "value": 5 }
]
```
```json
"base": { "exp": 10, "integral": 5 }
```
```json
"random": [
  { "asset": "integral", "value": 20, "chance": 10, "label": "幸运奖励" },
  { "asset": "integral", "min": 5, "max": 15 }
]
```

### 默认配置

```json
{
  "enabled": 1,
  "name": "每日签到",
  "reset_hour": 0,
  "notice": 0,
  "tips": [
    "签到成功，今天也要元气满满～",
    "滴，签到卡！新的一天开始啦",
    "坚持的人运气不会太差，明天见",
    "签到成功，离目标又近了一步",
    "连续签到有惊喜，别忘了明天再来"
  ],
  "base": [
    { "asset": "exp", "value": 10 },
    { "asset": "integral", "value": 5 }
  ],
  "streak": { "enabled": 1, "asset": "exp", "per_day": 2, "max": 50 },
  "cycle": {
    "enabled": 1,
    "loop": 1,
    "days": [
      { "label": "第 1 天" },
      { "label": "第 2 天" },
      { "label": "第 3 天", "rewards": [{ "asset": "integral", "value": 5 }] },
      { "label": "第 4 天" },
      { "label": "第 5 天" },
      { "label": "第 6 天" },
      { "label": "第 7 天", "rewards": [{ "asset": "integral", "value": 30, "label": "周期礼包" }] }
    ]
  },
  "milestones": [
    { "day": 7, "label": "连签一周", "rewards": [{ "asset": "exp", "value": 50 }, { "asset": "integral", "value": 20 }] },
    { "day": 15, "label": "半月坚持", "rewards": [{ "asset": "exp", "value": 100 }, { "asset": "integral", "value": 50 }] },
    { "day": 30, "label": "月度全勤", "rewards": [{ "asset": "exp", "value": 200 }, { "asset": "integral", "value": 100 }] }
  ],
  "monthly": [
    { "day": 20, "label": "月签满 20 天", "rewards": [{ "asset": "integral", "value": 50 }] },
    {
      "day": 28,
      "label": "当月全勤",
      "rewards": [
        { "asset": "integral", "value": 200 },
        { "asset": "card", "value": 50, "fallback": "integral", "label": "全勤卡密" }
      ]
    }
  ],
  "random": [{ "asset": "integral", "value": 20, "chance": 10, "label": "幸运奖励" }],
  "makeup": { "enabled": 1, "days": 7, "limit": 3, "asset": "integral", "cost": 20 }
}
```

### 子配置说明

**`streak` 连续签到加成**

| 字段 | 说明 |
| :--- | :--- |
| `enabled` | 是否启用 |
| `asset` | 奖励资产（默认 `exp`） |
| `per_day` | 每连续一天额外奖励 N |
| `max` | 加成上限（`0` 表示不封顶） |

> 连续第 N 天的加成为 `min(N × per_day, max)`。

**连签加成 与 里程碑 的区别（不冲突，可同时用）**

| | `streak` 连签加成 | `milestones` 里程碑 |
| :--- | :--- | :--- |
| 发放频率 | **每天都发** | 只在跨越档位的那一天发**一次** |
| 数量 | 随连签天数递增（`min(N × per_day, max)`） | 固定（配置多少发多少） |
| 定位 | 日常正向激励 | 阶段性大奖（如连签 7/30 天） |
| 判定 | 已启用且未达上限即发 | `before < 档位 <= after`，跨越才发 |

两者会在同一天叠加（例如连签第 7 天 = 基础 + 周期奖励 + 连签加成 7×2 + 里程碑奖励）。
觉得收益给多了，可以只留其中一个（把另一个 `enabled` 设为 `0`）。

**`cycle` 周期奖励**

| 字段 | 说明 |
| :--- | :--- |
| `enabled` | 是否启用 |
| `loop` | `1` 循环（第 8 天重新从第 1 天算）、`0` 不循环（超过天数按最后一天算） |
| `days` | 天数数组，每项 `{ label, rewards }`；`rewards` 为空表示该天只有基础奖励 |

**`makeup` 补签**

| 字段 | 默认 | 说明 |
| :--- | :--- | :--- |
| `enabled` | `1` | 是否允许补签 |
| `days` | `7` | 可补签最近 N 天内的日期（不含今天） |
| `limit` | `3` | 每月补签次数上限（`0` 不限制） |
| `asset` / `cost` | `integral` / `20` | 每次补签消耗的资产与数量 |
| `rewards` | 缺省 | 补签发放的奖励项（缺省 = 基础奖励） |

> 补签发放**补签奖励（默认基础奖励）**，不补发周期奖励与连签加成；
> 但如果补签把断掉的连签重新接起来、从而**跨越**了某个里程碑或月档位，这些一次性奖励会照常补发
> （按跨越口径，只发一次、不会重复）。补签成功后连签天数会重新计算，响应里的 `crossed` 会列出本次跨越的档位。

### 奖励资产（可扩展）

内置三种资产：

| 资产 | 名称 | 单位 | 图标 | 落地 |
| :--- | :--- | :--- | :--- | :--- |
| `exp` | 经验 | 经验值 | `bi-star` | `inis_exp` 流水 + `users.exp` |
| `integral` | 积分 | 积分 | `bi-coin` | `inis_integral` 流水 + `users.integral` |
| `card` | 卡密 | 张 | `bi-ticket-perforated` | 从卡密池取一张绑定给用户（`inis_integral_card` 状态置为 `2 已发放`），用户到「我的积分 → 卡密兑换」兑换成积分（见下节） |

- 经验 / 积分：`app/model/reward.go`
- 卡密：`app/model/reward-card.go`

### 卡密奖励（`asset = card`）

想「连签满多少天送一张卡密」，把奖励项的资产选成 `card` 即可：

```json
{ "asset": "card", "value": 50, "fallback": "integral", "label": "全勤卡密" }
```

| 字段 | 说明 |
| :--- | :--- |
| `value` | 卡密**面额**（会挑池子里面额相同的卡密）；填 `0` 表示不限面额，取任意一张 |
| `fallback` | 卡密池没有可用卡密时的降级方式：`integral`（默认，改发等额积分）/ `exp`（改发等额经验）/ `none`（不发，只在明细里标注） |
| `fallback_value` | 可选，降级发放的数量（缺省用卡密的 `value` 面额） |

**发放流程**

1. 从卡密池取一张「未使用 + 未过期」的卡密（`value > 0` 时按面额匹配，否则任意一张）；
2. 原子占用（状态条件更新 + 影响行数判断，并发下不会重复发同一张）并置为 `2 已发放`、绑定 `uid`；
3. 卡密明文随奖励明细一起返回（`items[].extra.card`），签到记录里也会保存；
4. 用户到「我的积分 → 卡密兑换」输入卡密，或直接用「我的待兑换卡密」一键兑换（`GET /api/integral/card-mine`）。

**卡密池为空时**：既不报错也不会让签到失败，按 `fallback` 降级（默认改发等额积分），
响应里会带上 `{ "card_missing": true, "fallback": "integral", "fallback_value": 50 }`，
前端据此提示「卡密池暂无库存，本次已改发 50 积分」。

**其它说明**

- 已发放的卡密**只有绑定用户能兑换**（`RedeemIntegralCard` 放行 `status = 2 且 uid = 本人`），
  其它人拿到卡密会得到「卡密已被使用！」；已发放的卡密不允许在后台删除（避免用户找不回奖励）；
- 卡密需要在后台「积分 → 卡密」里先生成；后台「签到管理」页顶部会显示当前可用卡密数量；
- 卡密的状态：`0 未使用`（谁都能兑换）/ `1 已使用`（已兑换成积分）/ `2 已发放`（奖励已发到某用户账号，待其兑换）。

扩展新资产（例如金币、道具、会员天数）只需注册一次：

```go
model.RegisterRewardAsset(model.RewardAsset{
    Key: "coin", Name: "金币", Unit: "金币", Icon: "bi-coin",
    Grant: func(tx *gorm.DB, uid, value int, meta facade.H) error {
        // 自行实现发放（建议用传入的 tx，保证与签到记录同一事务）
        return nil
    },
    Balance: func(uid int) int { return 0 },
})
```

注册后：

1. 签到配置里写 `{ "asset": "coin", "value": 100 }` 即可生效；
2. `GET /api/checkin/status` / `rules` 返回的 `assets` 清单会自动带上它，前端无需改动
   （前台按 `assets` 里的名称 / 单位 / 图标渲染奖励）。

---

## 奖励构成

一次签到可能同时命中多种来源，响应里的 `rewards` 按来源分组返回：

| 分组（`rewards` 的键） | 来源 | 条件 |
| :--- | :--- | :--- |
| `base` | 基础奖励 | 每次都有 |
| `cycle` | 周期奖励 | 连签第 N 天（`cycle.days[N-1]`） |
| `streak` | 连签加成 | `streak.enabled = 1` 且未达上限（**每天**都发） |
| `milestone` | 里程碑 | 连续天数**跨越**某个 `milestones[].day`（一次性） |
| `monthly` | 月累计 | 当月签到天数**跨越**某个 `monthly[].day`（一次性） |
| `random` | 随机奖励 | 按 `chance` 命中 |
| `makeup` | 补签奖励 | 仅补签时 |

`total` 字段是按资产汇总的总数，例如 `{ "exp": 24, "integral": 5 }`。

---

## 数据表 `inis_checkin`

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| id | int | 主键 |
| uid | int | 用户 ID（与 `date` 组成唯一索引） |
| date | int | 签到日键 `yyyymmdd`（按 `reset_hour` 折算） |
| days | int | 签到后的连续签到天数 |
| cycle | int | 周期内第几天（补签为 0） |
| month | int | 所属月份 `yyyymm` |
| source | int | `1` 正常签到 / `2` 补签 |
| exp / integral | int | 本次获得的经验 / 积分合计 |
| tips | string | 本次文案 |
| ip | string | 签到 IP |
| rewards | json | 奖励明细（items / total / detail） |
| create_time / update_time / delete_time | int64 | 公共字段（秒级时间戳，软删除） |

> 排行榜按 `create_time` 统计签到次数（与旧实现口径一致）。

---

## 接口详情

### 1. 签到状态 [业务接口]

- **路径**：`/api/checkin/status`
- **方法**：`GET`
- **权限**：需登录
- **说明**：签到页的主数据源。未签到时返回「签到后」的**预期奖励**；已签到时返回今天的实际明细。

**响应字段**：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| enabled | bool | 签到是否开启 |
| name | string | 签到名称 |
| checked | bool | 今天是否已签到 |
| check_in_time | int | 今天签到时间戳（未签到为 0） |
| streak / days | int | 连续签到天数（未签到时为截至昨天的天数） |
| preview_days | int | 今天签到后的连续天数（展示「连续第 N 天」用） |
| cycle_day | int | 今天（或今天签到后）处于周期第几天 |
| cycle_label | string | 周期标签 |
| cycle_days | array | 周期奖励全表：`[{ day, label, rewards }]` |
| month_days | int | 本月已签到天数 |
| rewards | object | 奖励明细，按来源分组（见「奖励构成」），**含概率 / 区间项**（带 `chance` / `min` / `max`） |
| items | array | **必得**的奖励项列表（含 name / unit / icon），已排除概率与区间项 |
| total | object | 必得奖励的按资产汇总：`{ exp: 24, integral: 5 }`（卡密的值为面额合计） |
| chance_items | array | 概率 / 区间奖励项（非必得），结构与 `items` 相同，另有 `chance` / `min` / `max` |
| chance_total | object | 概率 / 区间奖励的按资产汇总（**不要**与 `total` 相加展示，否则用户会误以为必得） |

> 「签到可得」请只用 `items` / `total`（必得部分）；`chance_items` / `chance_total` 建议用
> 「另有 10% 概率获得 20 积分」「另有 5~15 积分」这类文案单独提示。
> 已签到时（`checked = true`）这两个字段为空数组 / 空对象，`items` / `total` 是实际发放结果。
| cards | array | 今日获得的卡密：`[{ card, value, card_missing, fallback, fallback_value, granted_at }]`（未签到或没配卡密时为 `[]`） |
| milestones | array | 里程碑列表：`[{ day, label, rewards, reached, days_left }]` |
| next_milestone | object | 下一个未达成的里程碑（无则 `null`） |
| monthly / next_monthly | array / object | 月累计档位与下一个档位 |
| makeup | object | 补签信息：`{ enabled, days, limit, used, remain, asset, cost, dates: [{ date, text, time, cost }] }` |
| assets | array | 奖励资产清单：`[{ key, name, unit, icon }]` |
| reset_hour | int | 每日重置时间点 |
| today / today_text | int / string | 今天（按重置时间折算）的日键与文本 |
| tips | string | 未签到时给出随机文案；已签到时为本次文案 |
| record | object | 已签到时返回今天的记录摘要（date / days / cycle / source / exp / integral / tips / rewards） |

**响应示例（未签到）**：

```json
{
  "code": 200,
  "msg": "查询成功！",
  "data": {
    "enabled": true,
    "name": "每日签到",
    "checked": false,
    "streak": 2,
    "preview_days": 3,
    "cycle_day": 3,
    "cycle_days": [
      { "day": 1, "label": "第 1 天", "rewards": [] },
      { "day": 2, "label": "第 2 天", "rewards": [] },
      { "day": 3, "label": "第 3 天", "rewards": [{ "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin" }] }
    ],
    "month_days": 5,
    "rewards": {
      "base": [
        { "asset": "exp", "value": 10, "name": "经验", "unit": "经验值", "icon": "bi-star" },
        { "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin" }
      ],
      "cycle": [{ "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin" }],
      "streak": [{ "asset": "exp", "value": 6, "name": "经验", "unit": "经验值", "icon": "bi-star", "label": "连续签到加成" }],
      "random": [{ "asset": "integral", "value": 20, "chance": 10, "label": "幸运奖励", "name": "积分", "unit": "积分", "icon": "bi-coin" }]
    },
    "items": [
      { "asset": "exp", "value": 10, "name": "经验", "unit": "经验值", "icon": "bi-star", "group": "base" },
      { "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "base" },
      { "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "cycle" },
      { "asset": "exp", "value": 6, "name": "经验", "unit": "经验值", "icon": "bi-star", "group": "streak", "label": "连续签到加成" }
    ],
    "total": { "exp": 16, "integral": 10 },
    "chance_items": [
      { "asset": "integral", "value": 20, "chance": 10, "label": "幸运奖励", "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "random" }
    ],
    "chance_total": { "integral": 20 },
    "milestones": [
      { "day": 7, "label": "连签一周", "rewards": [], "reached": false, "days_left": 4 }
    ],
    "makeup": {
      "enabled": true, "days": 7, "limit": 3, "used": 0, "remain": 3,
      "asset": "integral", "cost": 20,
      "dates": [{ "date": 20260926, "text": "2026-09-26", "time": 1790352000, "cost": 20 }]
    },
    "assets": [
      { "key": "exp", "name": "经验", "unit": "经验值", "icon": "bi-star" },
      { "key": "integral", "name": "积分", "unit": "积分", "icon": "bi-coin" }
    ]
  }
}
```

### 2. 签到 [业务接口]

- **路径**：`/api/checkin/sign`
- **方法**：`POST`
- **权限**：需登录

**说明**：签到成功后按配置计算并发放奖励，「写签到记录 + 发奖励」在同一事务里，
任一步失败整体回滚。重复签到由 `(uid, date)` 唯一索引兜底，并发请求只会成功一个。

**响应字段**（成功）：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| checked | bool | 恒为 `true` |
| days / streak | int | 签到后的连续天数 |
| cycle_day | int | 周期第几天 |
| month_days | int | 本月累计签到天数 |
| items | array | 本次实际发放的奖励项（含 name / unit / icon） |
| rewards | object | 按来源分组的明细 |
| total | object | 按资产汇总 |
| cards | array | 本次发放的卡密（`card` 为明文；池子为空降级时带 `card_missing` / `fallback_value`） |
| crossed | object | 本次跨越到的档位：`{ streak: [7], monthly: [20] }`，无则空数组 |
| tips | string | 本次随机文案 |
| milestone | array | 本次命中的里程碑奖励（无则省略） |
| next_milestone | object | 下一个里程碑 |

**响应示例**：

```json
{
  "code": 200,
  "msg": "签到成功！",
  "data": {
    "checked": true,
    "days": 7,
    "streak": 7,
    "cycle_day": 7,
    "month_days": 7,
    "items": [
      { "asset": "exp", "value": 10, "name": "经验", "unit": "经验值", "icon": "bi-star", "group": "base" },
      { "asset": "integral", "value": 5, "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "base" },
      { "asset": "integral", "value": 30, "label": "周期礼包", "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "cycle" },
      { "asset": "exp", "value": 14, "label": "连续签到加成", "name": "经验", "unit": "经验值", "icon": "bi-star", "group": "streak" },
      { "asset": "exp", "value": 50, "name": "经验", "unit": "经验值", "icon": "bi-star", "group": "milestone" },
      { "asset": "integral", "value": 20, "name": "积分", "unit": "积分", "icon": "bi-coin", "group": "milestone" }
    ],
    "total": { "exp": 74, "integral": 55 },
    "tips": "坚持的人运气不会太差，明天见"
  }
}
```

**失败响应**（202）：

```json
{ "code": 202, "msg": "今天已经签到过了！", "data": null }
```

| msg | 原因 |
| :--- | :--- |
| `今天已经签到过了！` | 重复签到（含并发） |
| `签到功能已关闭！` | `enabled = 0` |
| `签到失败，请稍后重试！` | 数据库异常（已回滚） |

### 3. 补签 [业务接口]

- **路径**：`/api/checkin/makeup`
- **方法**：`POST`
- **权限**：需登录

**请求参数**：

| 参数名 | 类型 | 必填 | 说明 |
| :--- | :--- | :--- | :--- |
| date | int | 否 | 补签日键 `yyyymmdd`，不传则补昨天 |

**响应字段**：`date`、`date_text`、`days`（补签后的连签天数）、`items`、`total`、`cards`、
`crossed`（补签跨越的里程碑 / 月档位，如 `{ streak: [7], monthly: [] }`）、
`cost`（`{ integral: 20 }`）、`month_days`、`makeup`（剩余次数与可补签日期）。

**失败响应**（202）：

| msg | 原因 |
| :--- | :--- |
| `补签功能已关闭！` | `makeup.enabled = 0` |
| `只能补签今天之前的日期！` | 日期是今天或未来 |
| `超出可补签范围！` | 超出 `makeup.days` 天 |
| `该日期已经签到过了！` | 该日已有记录 |
| `本月补签次数已用完！` | 超过 `makeup.limit` |
| `积分不足！` | 消耗检查失败（资产为积分时） |

### 4. 签到日历 [业务接口]

- **路径**：`/api/checkin/calendar`
- **方法**：`GET`
- **权限**：需登录

**请求参数**：`year`、`month`（不传为当前月）。

**响应字段**：

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| year / month | int | 年月 |
| days | array | `[{ day, checked, source, exp, integral }]`，`source`：`0` 未签到 / `1` 正常 / `2` 补签 |
| streak | int | 连续签到天数（当月看今天 / 昨天，历史月份看月末） |
| total / month_days | int | 当月已签到天数 |
| today | int | 今天（非当月为 0） |
| monthly / next_monthly | array / object | 月度档位与下一个档位 |

### 5. 签到排行榜 [业务接口]

- **路径**：`/api/checkin/rank`
- **方法**：`GET`
- **权限**：公开

**请求参数**：

| 参数名 | 类型 | 必填 | 默认 | 说明 |
| :--- | :--- | :--- | :--- | :--- |
| start | int | 否 | 本月 1 日 0 点 | 起始时间戳（秒） |
| end | int | 否 | 本月末 23:59:59 | 结束时间戳（秒） |
| limit | int | 否 | 10 | 条数（上限受 `SYSTEM_PAGE_LIMIT` 限制） |

**响应字段**：`list`（`[{ id, nickname, avatar, description, title, rank, check_in_count, total_exp, total_integral, is_me }]`）、
`my_rank`、`my_count`。

### 6. 签到规则 [业务接口]

- **路径**：`/api/checkin/rules`
- **方法**：`GET`
- **权限**：公开

**说明**：给前台渲染「签到规则说明 / 周期奖励表」用，未登录也能看。

**响应字段**：`enabled`、`name`、`reset_hour`、`tips`、`assets`、`base`、`cycle`
（`{ enabled, loop, days }`）、`streak`、`milestones`、`monthly`、`random`、`makeup`。

---

## 特殊说明

### 1. 与经验 / 积分流水的关系

- 签到发放的经验 / 积分仍会写入 `inis_exp` / `inis_integral` 流水（`type = check-in`），
  所以「我的经验」「我的积分」页照常显示；
  经验流水的 `json` 为 `{ source, streak, cycle }`（补签时 `source = 2`）；
- 积分任务页（`GET /api/integral/tasks`）里的「每日签到」：
  - **是否完成**以签到记录为准（而不是积分流水条数）；
  - **可得积分**取签到配置里基础奖励的积分合计，因此把签到积分改到 20，任务页也会显示 20；
  - 连签天数同样来自签到模块；
- **积分 / 经验规则里不再有 `check-in`**：`SYSTEM_INTEGRAL_RULES`、`SYSTEM_EXP_RULES` 里的
  签到项已移除（历史残留会在启动迁移时自动清理）；任务列表与「积分获取途径」里的签到项由
  签到配置动态注入，带 `source: "checkin"` 标记，与规则配置互不干扰。

### 2. 旧端点迁移对照

| 旧端点（保留兼容） | 新端点 |
| :--- | :--- |
| `POST /api/exp/check-in` | `POST /api/checkin/sign` |
| `GET /api/exp/check-in-status` | `GET /api/checkin/status` |
| `GET /api/exp/check-in-calendar` | `GET /api/checkin/calendar` |
| `GET /api/exp/check-in-rank` | `GET /api/checkin/rank` |
| 经验规则 `SYSTEM_EXP_RULES.check-in` | 签到配置 `SYSTEM_CHECKIN_RULES` |

旧端点内部直接调用新模块，行为完全一致（响应额外保留 `value = 获得的经验合计`），
配置与权限规则也都保留，已有前端 / 第三方调用不会中断。

### 3. 时区与「签到日」

- 所有时间戳都是 Unix 秒，日期边界按服务器本地时区；
- `reset_hour = 4` 时，凌晨 0:00-3:59 的签到算作「前一天」，因此会出现「同一自然日能签两次、
  或某个自然日不能签」的情况，属于预期行为；
- 修改 `reset_hour` 不会补齐 / 删除历史记录，切换当天可能出现一次「可以再签」的情况。

### 4. 并发与幂等

- 签到依赖 `inis_checkin(uid, date)` 唯一索引：并发请求只有一个能插入成功，其余返回
  「今天已经签到过了！」；
- 「写记录 + 发奖励」在同一数据库事务中，不会出现「经验加了但记录没写」的情况；
- 积分扣减（补签消耗）使用带余额条件的原子更新，不会扣成负数。

### 5. 唯一性提示

若奖励配置里包含随机项，签到每次结果可能不同，这是预期行为；但**签到记录一天只可能有一条**，
所以不会重复发放基础 / 周期 / 里程碑奖励（`random` 也只在该次签到时结算一次）。

### 6. 卡密奖励的注意事项

- 卡密需要先在后台「积分 → 卡密」生成，否则会走 `fallback`（默认改发等额积分）；
- 发放出去（已发放）的卡密不能删除，用户可以在「我的积分 → 卡密兑换」找回
  （`GET /api/integral/card-mine` 返回自己的待兑换卡密），也可以一键兑换；
- 一张卡密只会发给一个用户：发放时用「状态条件更新 + 影响行数」原子占用，并发签到不会重复发同一张；
- 卡密有有效期时同样受有效期约束：过期的卡密不会被发放，也不会被兑换。

### 7. 签到成功通知的文案

通知内容由 `model.checkinGrantText()` 生成（开关是配置里的 `notice`，默认关闭），规则：

1. **同一种奖励合并成合计**：基础 2 经验 + 连签加成 2 经验 → 显示「4 经验值」，
   不会再出现「2 经验值、2 经验值」这种看着像发重的写法；
2. **来源多于一种时写清构成**，用括号说明每一项是怎么来的；
3. **卡密按张数展示**（它的数值是面额，不能写成「50 张」），并带上明文与兑换入口。

示例（连签第 1 天：基础 2 经验 + 2 积分 + 连签加成 2 经验）：

> 今日签到成功，获得：4 经验值、2 积分（基础 2 经验值 + 2 积分 · 连签加成 +2 经验值）。已连续签到 1 天。

跨越里程碑时，括号里会多出「里程碑 +50 经验值 + 20 积分」这样的来源；发放卡密时会追加
「卡密：K7M9X2P7Q1Z8B4N6（面额 50 积分），到「我的积分 → 卡密兑换」兑换」。
