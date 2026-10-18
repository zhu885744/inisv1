## Socket 实时通道（`/socket`）

### 概述

| 项 | 说明 |
| :--- | :--- |
| 端点 | `GET /socket`（WebSocket）——见 `app/socket/route/app.go` |
| 中间件 | `api/middleware.Auth()` + `socket/middleware.App` |
| 鉴权 | 握手时按登录校验中间件取 token（请求头 `Authorization` 或 Cookie `INIS_LOGIN_TOKEN`，即服务端会话 id）；浏览器无法给握手设置请求头、跨域也带不上 Cookie，所以连接后应发送 `{"type":"auth","token":"…"}` 升级身份（`app/socket/controller/index.go#handleAuth`，内部走 `facade.AuthenticateToken`，与 HTTP 同一套会话校验），升级成功后会重新广播在线状态 |
| 心跳 | 服务端定期发 WS `ping`（浏览器自动回 `pong`）；客户端可发 `{"type":"ping"}` 收到 `{"type":"pong"}` |
| 前端封装 | `Mellow/src/utils/socket.js`（`createSocketClient`，断线自动重连，退避 1s→20s） |
| 后台页面 | 「管理 → 数据统计」`/admin/stats`（`Mellow/src/views/admin/Stats.vue`） |

### 连接建立与准入

握手路径 `GET /socket`（`/socket/` 等价），中间件链 `api/middleware.Auth()` → `socket/middleware.App`：

1. `Auth()` **可选登录**：带凭证则识别用户，不强制；
2. `App` 生成连接标识 `client_id`：
   - 登录用户：`user_<uid>`；
   - 访客：`guest_<md5(User-Agent + IP + Host + Accept + Accept-Language)>`（同一浏览器稳定，换 IP 会变）；
3. 服务端在握手响应头里回传 `X-Client-Id`，**客户端应保存它**：重连时离线消息补发、私聊 `to`、后台踢下线都靠这个 id 定位。

握手阶段的准入检查（`app/socket/controller/index.go#Connect`），任一不通过直接以 HTTP 响应拒绝、不升级：

| 顺序 | 检查 | 失败响应 |
| :--- | :--- | :--- |
| 1 | 是否带 `Upgrade: websocket` | `400 {"error":"非WebSocket连接请求"}` |
| 2 | `Hub.IsBlacklisted(ip)`（进程内黑名单） | `403 {"error":"您的IP已被封禁"}` |
| 3 | `Hub.IsOriginAllowed(origin)`（默认关闭） | `403 {"error":"不允许的来源"}` |
| 4 | 单 IP 并发连接数 | `429 {"error":"连接数超限，请稍后再试"}` |
| 5 | 升级成功 | 推 `connect`，并向全体广播在线状态 |

几点容易踩的：

- **Origin 校验默认关闭**（`securityConfig.enableOriginCheck = false`），且允许列表目前没有配置入口；
  开启后若列表为空，等于拒绝所有来源；
- **`/socket` 不挂 API 的 `IpBlack` / QPS 中间件**（只有 `Auth` + socket 自己的 `App`），
  站点 IP 黑名单不会自动作用于 WebSocket 握手，需要在反向代理层另外封禁；
- 单 IP 连接数默认 10（`max_connections_per_ip`）；`enable_ip_ban = true` 时累计 3 次超限会在内存里封该 IP 300 秒；
- 身份升级（`auth`）**不改变 `client_id`**，只更新 `username` / `isAdmin`（`IsRootAdmin(uid)`），
  这样重连仍能匹配到原来的离线消息。

### 消息类型

| type | 方向 | 负载 | 可见性 |
| :--- | :--- | :--- | :--- |
| `connect` | 服务端 → 客户端 | `{ type, content: "连接成功", id }` | 连接者 |
| `pong` | 服务端 → 客户端 | `{ type }` | 请求者 |
| `status`（在线状态） | 服务端 → 全部 | `{ content: { online_users: [...], online_count } }` | 所有人 |
| `status`（系统状态） | 服务端 → 管理员 | `{ content: { info, database, cache, resource }, admin_only: true, timestamp }` | **仅管理员** |
| `kicked` | 服务端 → 客户端 | `{ type, content }` | 被踢者 / 发起者 |
| `error` | 服务端 → 客户端 | `{ type, content }` | 发起者（如非管理员发 kick） |
| `broadcast` / `single` / `private` | 双向 | 广播 / 单播 / 私聊（含 `msg_id`、ACK 重传），字段见下方「下行消息结构」 | 按需 |

#### 下行消息结构（两种，字段名不同）

**① 中继类**（`broadcast` / `single` / `private` / `notification`）走 `serialize.go` 的 `info` 结构：

```json
{
  "id": "user_3",
  "to": "user_9",
  "type": "single",
  "data": { "type": "single", "text": "hi" },
  "msg_id": "c1f2…"
}
```

| 字段 | 说明 |
| :--- | :--- |
| `id` | **发送方**的 clientId |
| `to` | 接收方 clientId（广播时为空串，字段仍存在，没有 `omitempty`） |
| `type` | 服务端重写后的类型：`broadcast` / `single` / `private` |
| `data` | 客户端发来的原始消息对象（**注意字段名是 `data`，不是 `content`**；其中 `to` 已被服务端删除） |
| `msg_id` | 私聊消息 id；客户端没给时由服务端生成（`omitempty`） |

**② 控制 / 状态类**（`connect` / `pong` / `kicked` / `error` / `status`）是手写 JSON，字段名是 `content`：

```json
{ "type": "connect", "content": "连接成功", "id": "guest_6f1c…" }
{ "type": "status",  "content": { "online_users": [ … ], "online_count": 12 } }
{ "type": "status",  "content": { "info": { … }, "database": { … }, "cache": { … }, "resource": { … }, "status": "healthy", "timestamp": 1760000000 }, "admin_only": true, "timestamp": 1760000000 }
```

所以前端处理时应按 `type` 分支，不要假设所有消息都读 `content`：中继类读 `data`。

#### 上行消息结构（客户端 → 服务端）

客户端发送任意 JSON，但必须满足（`ValidateMessage`）：

- 合法 JSON，且**必须含 `type`**；
- `type` 必须在白名单内：`broadcast` / `single` / `private` / `notification` / `ack` / `read` / `ping` / `pong` / `auth`（`kick` 由管理员使用，走单独分支）；
- `type` 为 `single` / `private` 时**必须含 `to`**；
- 单条消息不超过 `max_message_size`（默认 1 MB；连接层 `SetReadLimit` 固定 1 MB，超出直接断开）。

服务端转发前会重写类型与目标（`index.go#read`）：

| 客户端发送 | 实际分发 |
| :--- | :--- |
| 无 `to` | `broadcast`，发给所有在线连接 |
| 有 `to`，`type=private` | `private`，只发给 `to` |
| 有 `to`，其它 type | 统一改写为 `single`，只发给 `to` |
| `type=single` 但没给 `to` | `ValidateMessage` 直接拒绝（不转发） |
| `ack` / `read` / `ping` / `auth` | 不转发，只做本地处理 |

> `single` 的目标不在线时，服务端会把它缓存为离线消息（见下文「离线消息」）；
> 私聊 `private` 还会补 `msg_id` 并等待 ACK。

客户端可发送的控制消息（见 `app/socket/controller/index.go` 的 `read()`）：

| type | 负载 | 说明 |
| :--- | :--- | :--- |
| `ping` | `{ type }` | 应用层心跳，服务端回 `pong`（WS 层另有 ping） |
| `auth` | `{ type, token }` | 连接后升级身份（跨域带不上 Cookie 时用） |
| `kick` | `{ type, to }` | **仅管理员**：断开 `to` 指向的连接（`user_1` / `guest_xxx`） |

> 两种 `status` 靠 `content` 字段区分：含 `online_count` → 在线状态；含 `resource` / `database` → 系统状态。
> 系统状态由 `app/socket/controller/status.go` 推送，间隔见 `socket.status_interval`（默认 3 秒）；
> **非管理员连接不会收到**，且无管理员在线时服务端会跳过采集（见下文「采集成本与节流」）。

#### ⚠️ 一帧可能包含多条消息

服务端的写循环会把发送队列里的多条消息**拼进同一帧**（`index.go#write` 逐条 `Write`，
`serialize.go` 里的分隔符 `line = []byte{'\n'}`）。因此客户端**必须按 `\n` 拆分后逐条 `JSON.parse`**：

```js
for (const chunk of event.data.split('\n')) {
  if (!chunk.trim()) continue
  try { handle(JSON.parse(chunk)) } catch { /* 跳过坏帧 */ }
}
```

直接 `JSON.parse(event.data)` 会在服务端出现推送抖动（多条消息排队）时抛错，
表现为「实时数据偶尔整段丢失」且难以定位。
Go 的 `json.Marshal` 会把字符串内的换行转义成 `\n` 两个字符，所以按真实换行字节拆分是安全的。

### 系统状态字段

| 路径 | 字段 |
| :--- | :--- |
| `content.info` | `app_name` / `go_version` / `os` / `arch` / `cpu_count` / `goroutines` / `current_time` |
| `content.database` | `connected` / `latency` / `error` / `counts`（`users` `articles` `moments` `comments` `pages` `links` `banners` `placards` `tags` `attachments`） |
| `content.cache` | `enabled` / `type` / `working` / `error`（每次推送都做一次「写入 → 读取 → 删除」校验） |
| `content.resource.memory` | `alloc` / `total_alloc` / `sys` / `gc_count` / `system_total` / `system_used` / `system_free` / `system_usage` |
| `content.resource.cpu` | `count` / `model` / `usage` / `load_1m` / `load_5m` / `load_15m` |
| `content.resource.disk` | `total` / `used` / `free` / `usage` / `fs_type` / `read` / `write` / `read_per_sec` / `write_per_sec` / `io_latency` |
| `content.resource.network` | `bytes_sent` / `bytes_recv` / `packets_sent` / `packets_recv` / `sent_per_sec` / `recv_per_sec` / `up` / `down` / `total_sent` / `total_received` |
| `content.resource.system` | `os` / `os_version` / `kernel` / `boot_time` |
| `content.status` | 固定 `healthy` |
| `content.timestamp` | 采集时间（秒级时间戳，与外层 `timestamp` 相同） |

> 带单位的字段（`usage` / `system_usage` / `up` / `down` 等）是**字符串**，如 `"12.34%"`、`"1.23 MB/s"`；
> 前端做图表时需要自行解析成数值（见 `Stats.vue` 的 `toPercent` / `toKB`）。
> `disk.io_latency` 目前是按时间戳模拟的值，不是真实 IO 延迟测量（`status.go` 里有注释说明）。

### 在线用户与踢下线

在线列表来自 `status`（在线状态）里的 `online_users`，每项为 `{ id, name, ip, last_active }`
（`ip` 由服务端脱敏，`name` 为昵称，未登录显示「访客」）。

踢下线的最小实现（后台「数据统计 → 在线用户」按钮，见 `Mellow/src/views/admin/Stats.vue`）：

```json
// 客户端 → 服务端
{ "type": "kick", "to": "guest_6f1c…" }

// 服务端 → 被踢者（随后连接被关闭）
{ "type": "kicked", "content": "您已被管理员断开连接" }
```

几个实现要点：

- **权限校验发生在 hub goroutine**：`client.isAdmin` 由 hub 在身份升级时写入，
  若在 read goroutine 里直接读会有数据竞争，因此 `kick` 消息经 `hub.kick` 通道转交处理
  （`handleKick`），非管理员会被拒回 `error`；
- 断开前先给被踢者发一条 `kicked`（延迟 200ms 再关连接，留出写出的时间），
  客户端收到后应**停止自动重连**（否则会立刻连回来，踢了等于没踢）——
  前端 `createSocketClient({ onKicked })` 已按此处理，用户仍可手动「重新连接」；
- 被踢只是断开连接，**不是封禁**；连接关闭后其 `read()` 会走统一的下线清理，
  在线列表随之刷新（全体收到新的 `status` 在线状态）；
- 踢游客与踢登录用户都可用（`id` 取自在线列表，格式 `user_<uid>` / `guest_<hash>`）。

### ACK、已读回执与离线消息

**私聊 ACK 重传**（`sendWithAck` / `waitForAck`）：

```json
// 服务端下发（自带 msg_id）
{ "id": "user_3", "to": "user_9", "type": "private", "data": { … }, "msg_id": "c1f2…" }

// 客户端确认收到
{ "type": "ack", "msg_id": "c1f2…" }
```

- 发出后等待 `ack_timeout`（默认 10 秒）没收到 `ack` 就重传，最多重传 `max_retries`（默认 3）次，之后放弃（内部计数 `ack_failed`）；
- 收到 `ack` 后从待确认表里删除，不再重传；
- 这条链路目前只用于**私聊**；广播 / 单播不要求 ACK。

**已读回执**：客户端发 `{"type":"read","msg_id":"c1f2…"}`，服务端只更新内存里的
`messageStatuses[msgId].readBy[clientId]`，**不会转发给对端**；需要「对方已读」提示的场景要自己再发一条业务消息。

**离线消息**（只在内存，进程重启即丢）：

| 项 | 值 | 配置项 |
| :--- | :--- | :--- |
| 缓存对象 | `single` / `private` 的目标不在线时 | — |
| 有效期 | 300 秒（补发时过滤过期） | `offline_msg_ttl` |
| 条数上限 | 每客户端 100 条，超出丢最旧的 | `max_offline_msgs` |
| 会话保留 | 断线后 30 秒内重连可按 `client_id` 恢复并补发 | `reconnect_timeout` |
| 跨实例 | **不缓存**其它实例转发来的消息（避免 N 个实例各存一份、重连收 N 条） | — |

> 补发没有确认机制、也没有持久化：`client_id` 变了（访客换了 IP / UA）或超过 30 秒，
> 状态会被清理，之前的离线消息就取不回来了。业务上需要可靠投递时，应以 HTTP 拉取（如通知列表）为准。

### 频率限制与消息校验

| 项 | 默认 | 说明 |
| :--- | :--- | :--- |
| 上行消息频率 | 100 条 / 分钟 / 连接 | `max_messages_per_minute`；超限**静默丢弃**，不回 `error`（避免被刷时反而放大流量） |
| 单条消息大小 | 1 MB | `max_message_size`；连接层读取上限同样是 1 MB |
| 单 IP 并发连接 | 10 | `max_connections_per_ip` |
| IP 临时封禁 | 关闭（阈值 3 次 / 300 秒） | `enable_ip_ban` / `ip_ban_threshold` / `ip_ban_duration` |
| WS 层心跳 | 服务端每 108 秒 ping，120 秒没 pong 判定掉线 | 固定值（`pingPeriod` / `pongWait`），不可配置 |

- `ping` / `auth` / `ack` / `read` / `kick` 在限流检查**之前**处理，不占配额；
- 频率超限只影响「转发类」消息，不影响心跳与回执，避免限流后连接被误判为僵尸；
- 校验失败的消息直接丢弃并写 debug 日志（`socket.debug = true` 才可见），客户端不会收到错误回执。

### 采集成本与节流

`getSystemStatus()` 每次包含 `SELECT 1` + 约 17 条 `COUNT(*)`（用户 4、文章 3、动态 3，
评论 / 页面 / 友链 / 轮播 / 公告 / 标签 / 附件各 1）+ 一次缓存的「写入 → 读取 → 删除」校验。
原先固定每秒执行且不判断有没有人在看，属于典型的空转负载，现已做三层节流：

| 节流 | 行为 | 配置项（`config/app.toml` 的 `[socket]`） |
| :--- | :--- | :--- |
| 无人在线跳过 | 在线管理员数为 0 时，本轮**完全不采集**（不产生任何数据库 / 缓存操作） | — |
| 间隔可配 | 默认 **3 秒**（原 1 秒）；配合前端 60 个采样点 = 最近 3 分钟 | `status_interval = 3` |
| 重活 TTL 缓存 | 各表计数与缓存探测按 TTL 复用，最快每 5 秒才真正打一次库 | `status_count_cache = 5` |

- 在线管理员数由 `hub.adminOnline`（`atomic.Int64`）维护，在 `hub.run` 的 connect / close / upgrade
  三个分支里增减 —— 不直接读 `hub.clients`：那个 map 只在 hub goroutine 内读写，跨 goroutine 读会有数据竞争；
- 缓存命中时，前端看到的 CPU / 内存 / 磁盘 / 网络仍是**实时**的，只有「各表数据量」与「缓存探测结果」
  最多滞后 `status_count_cache` 秒；
- 统计出错时的 `recover` 已挪进缓存构建闭包内：只退化为默认值，不再像以前那样中断整轮采集
  （以前一旦统计 panic，CPU / 内存 / 网络等会一起丢失）；
- 前端 `Stats.vue` 会按**实测推送间隔**换算趋势图的时间窗口文案，所以调整间隔不必改前端。

### 多实例部署（Redis Pub/Sub）

hub 的连接表与分发全在进程内存里，多实例时 A 上的广播 / 单播 / 私聊 / 踢下线默认只对 A 的连接生效，
因此 `app/socket/controller/cluster.go` 用 Redis Pub/Sub 做了跨实例转发：

| 项 | 说明 |
| :--- | :--- |
| 启用条件 | `cache.toml` 的 `default = "redis"` 且 Redis 客户端已创建；`file` / `ram` 驱动仍是单实例行为 |
| 频道 | `inis:socket:cluster`，信封 `{ from, kind, data }` |
| `kind=msg` | 普通消息：本实例分发后转发，其它实例收到后**只做本地分发，不再二次发布**（防 A→B→A 回环） |
| `kind=kick` | 踢下线：目标不在本实例时转发，持有连接的实例执行断开 |
| 不转发的内容 | `type=status` 的系统状态（那是本机资源快照，转发会让管理员看到别的机器的 CPU / 内存） |
| 订阅断线 | 每 5 秒重建订阅，Redis 恢复后自动接上 |
| 队列满 | 直接丢弃并记日志（`cluster` 1024 / `clusterKick` 256），不阻塞 hub |

> 启用条件同时要求「缓存驱动是 redis」：只配了 redis 但 `default=file` 时会话本身都不能跨实例，转发也没有意义。

### 服务端配置项（`config/app.toml` 的 `[socket]`）

| 键 | 默认 | 说明 |
| :--- | :--- | :--- |
| `debug` | `false` | 打印 socket 调试日志（连接 / 消息 / 限流 / 重传） |
| `status_interval` | `3` | 系统状态推送间隔（秒，最小 1） |
| `status_count_cache` | `5` | 表计数与缓存探测的 TTL（秒），0 表示不缓存 |
| `max_message_size` | `1048576` | 单条上行消息上限（字节） |
| `max_connections_per_ip` | `10` | 单 IP 并发连接上限 |
| `max_messages_per_minute` | `100` | 单连接每分钟上行消息上限 |
| `enable_ip_ban` | `false` | 是否对频繁超限的 IP 做内存临时封禁 |
| `ip_ban_threshold` | `3` | 触发临时封禁的超限次数 |
| `ip_ban_duration` | `300` | 临时封禁时长（秒） |
| `ack_timeout` | `10` | 私聊 ACK 等待时间（秒） |
| `max_retries` | `3` | 私聊重传次数上限 |
| `offline_msg_ttl` | `300` | 离线消息有效期（秒） |
| `max_offline_msgs` | `100` | 单客户端离线消息条数上限 |
| `reconnect_timeout` | `30` | 断线后保留会话状态（可用于补发）的时长（秒） |

> `[socket]` 段可以整段省略，全部走默认值。WS 层心跳（108 秒 ping / 120 秒 pong）是代码常量，不在此表内。

### 部署注意（反向代理必须放通 WebSocket 升级）

```nginx
location /socket {
    proxy_pass http://127.0.0.1:9852;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400s;
    proxy_send_timeout 86400s;
}
```

- 若 `location /` 里无条件带了 `Connection "upgrade"`，会让 Nginx 无法复用到上游的长连接（每请求可能新建 TCP），
  因此建议按上面把 WebSocket 单独拆一个 location；
- 连不上时「数据统计」页会给出提示：① 当前账号是否为管理员；② 代理是否放通 `/socket`。

### 本地开发（Vite dev server）

开发态页面在 `5173`、后端在别的端口/域名，需要 dev 代理转发升级握手（`Mellow/vite.config.js`）：

```js
'/socket': { target: 'wss://zhuxu.asia', ws: true, changeOrigin: true }
```

**`target` 只能给「源」（scheme + host[:port]），不能带路径**：若写成 `wss://zhuxu.asia/socket`，
http-proxy 会把 target 的路径拼在请求前（`/socket` → `/socket/socket`），后端匹配不到该路由、
回落到 SPA 首页，浏览器侧只会看到
`WebSocket connection failed: Error during WebSocket handshake: Unexpected response code: 200`
（配置里的 `VITE_SOCKET` 可以照常带路径，`vite.config.js` 会自动截取到源）。漏配 `ws: true` 则不会转发握手。

### 前端配置 socket 地址

解析顺序（`Mellow/src/utils/socket.js#socketUrl`，配置文件 `Mellow/src/utils/runtimeConfig.js`）：

| 优先级 | 位置 | 是否需要重新打包 |
| :--- | :--- | :--- |
| 1 | `Mellow/public/runtime-config.js` → `window.__INIS_CONFIG__.socketUri` | 否（改服务器上的 `public/runtime-config.js` 刷新即可） |
| 2 | `Mellow/.env` → `VITE_SOCKET` | 是（`npm run build`） |
| 3 | `socketUri` 为空时回落 `apiUri` / `VITE_API_URI` | — |
| 4 | 以上都为空 → 与当前站点同源 | — |

写法兼容（路径一律固定为 `/socket`，写在地址里也只是被覆盖，不影响结果）：

| 配置值 | 页面为 https 时解析结果 |
| :--- | :--- |
| `''`（推荐：同源） | `wss://当前域名/socket` |
| `wss://api.example.com/socket` | `wss://api.example.com/socket` |
| `https://api.example.com` | `wss://api.example.com/socket` |
| `//api.example.com` | `wss://api.example.com/socket` |
| `api.example.com`（只写域名） | `wss://api.example.com/socket` |
| `/socket`（只写路径） | `wss://当前域名/socket` |
| `ws://api.example.com:8080` | `wss://api.example.com:8080/socket`（页面是 https 时**强制升级为 wss**，否则浏览器按混合内容拦截） |

结论：

- **前后端同域部署（推荐）**：`socketUri` 与 `apiUri` 都留空，换域名 / 换端口都不用改配置；
- **前后端分离**：只填 `socketUri: 'wss://api.example.com'` 即可（后端 `/socket` 与 `/api` 在同一进程，填接口域名即可）；
- **本地开发**：页面在 `http://127.0.0.1:5173`、后端在 `8642` 时，填 `ws://127.0.0.1:8642`（或让 `apiUri` 指向它，socket 自动跟随）；
- 若 `socketUri` 跨域，浏览器不会带上登录 Cookie —— 前端会自动在使用 `{ type: 'auth', token }` 补鉴权，无需额外配置。

### 前端接入（`createSocketClient`）

```js
import { createSocketClient } from '@/utils/socket'

const client = createSocketClient({
  onMessage(payload) {
    switch (payload.type) {
      case 'connect':
        // payload.id 就是 clientId，建议存下来（重连 / 私聊 to / 后台踢下线都用它）
        break
      case 'status':
        if (payload.content.online_count !== undefined) {
          // 在线列表：content.online_users / content.online_count
        } else {
          // 系统状态：content.info / database / cache / resource（仅管理员收得到）
        }
        break
      case 'single':
      case 'private':
      case 'broadcast':
        // 业务消息在 payload.data 里；私聊需回 ack
        break
      case 'pong':
        break
    }
  },
  onState(state) {
    // connecting / open / closed / failed / kicked
  },
  onKicked(payload) {
    // 被管理员断开：客户端已自动停止重连，这里只负责提示用户
  },
  autoReconnect: true
})

client.connect()                 // 建立连接（onopen 后自动发 auth + 启动 30 秒应用层 ping）
client.send({ type: 'ping' })    // 发送消息（会自动 JSON.stringify）
client.send({ type: 'ack', msg_id: 'c1f2…' })
client.reconnect()               // 手动重连：重置退避与失败计数
client.close()                   // 主动关闭（不再自动重连）
console.log(client.state)        // 当前状态
```

行为约定：

| 项 | 说明 |
| :--- | :--- |
| 重连退避 | 1 → 2 → 3 → 5 → 8 → 13 → 20 秒循环；连续失败 8 次后停止，状态转 `failed`，需手动 `reconnect()` |
| 鉴权 | `onopen` 时自动读取 Cookie `INIS_LOGIN_TOKEN`（回退 `localStorage.mellow_token`）并发 `{ type: 'auth', token }` |
| 应用层心跳 | 每 30 秒发 `{ type: 'ping' }`，用于刷新服务端最后活跃时间 |
| 拆帧 | 客户端已按 `\n` 拆分并逐条 `JSON.parse`，业务侧拿到的都是单条消息 |
| 被踢 | 收到 `kicked` 立即停止自动重连，状态停留 `kicked`（不会被 `onclose` 覆盖成 `closed`） |
| 状态 | `connecting` / `open` / `closed` / `failed` / `kicked` 五态，可直接驱动界面提示 |

> 该封装是**通用**的（不依赖后台）：需要实时能力的页面自行 `createSocketClient`，
> 在页面卸载时调用 `client.close()` 即可，避免重复连接累积。
