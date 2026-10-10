## Socket 实时通道（`/socket`）

### 概述

| 项 | 说明 |
| :--- | :--- |
| 端点 | `GET /socket`（WebSocket）——见 `app/socket/route/app.go` |
| 中间件 | `api/middleware.Jwt()` + `socket/middleware.App` |
| 鉴权 | 握手时按 Jwt 中间件取 token（请求头 `Authorization` 或 Cookie `INIS_LOGIN_TOKEN`）；浏览器无法给握手设置请求头、跨域也带不上 Cookie，所以连接后应发送 `{"type":"auth","token":"…"}` 升级身份（`app/socket/controller/index.go#handleAuth`），升级成功后会重新广播在线状态 |
| 心跳 | 服务端定期发 WS `ping`（浏览器自动回 `pong`）；客户端可发 `{"type":"ping"}` 收到 `{"type":"pong"}` |
| 前端封装 | `Mellow/src/utils/socket.js`（`createSocketClient`，断线自动重连，退避 1s→20s） |
| 后台页面 | 「管理 → 数据统计」`/admin/stats`（`Mellow/src/views/admin/Stats.vue`） |

### 消息类型

| type | 方向 | 负载 | 可见性 |
| :--- | :--- | :--- | :--- |
| `connect` | 服务端 → 客户端 | `{ type, content: "连接成功", id }` | 连接者 |
| `pong` | 服务端 → 客户端 | `{ type }` | 请求者 |
| `status`（在线状态） | 服务端 → 全部 | `{ content: { online_users: [...], online_count } }` | 所有人 |
| `status`（系统状态） | 服务端 → 管理员 | `{ content: { info, database, cache, resource }, admin_only: true, timestamp }` | **仅管理员** |
| `kicked` | 服务端 → 客户端 | `{ type, content }` | 被踢者 / 发起者 |
| `error` | 服务端 → 客户端 | `{ type, content }` | 发起者（如非管理员发 kick） |
| `broadcast` / `single` / `private` | 双向 | 广播 / 单播 / 私聊（含 `msg_id`、ACK 重传） | 按需 |

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
