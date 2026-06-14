## Socket WebSocket 服务前端使用指南

### 概述

INIS Socket 服务基于 WebSocket 协议，提供实时双向通信功能，支持广播、私聊、系统状态推送等特性。

### 连接地址

```
ws://your-domain/socket
```

### 连接示例

#### JavaScript 原生 WebSocket

```javascript
// 创建 WebSocket 连接
const socket = new WebSocket('ws://your-domain/socket');

// 连接成功
socket.onopen = function(event) {
    console.log('WebSocket 连接成功');
    
    // 发送心跳
    setInterval(() => {
        socket.send(JSON.stringify({ type: 'ping' }));
    }, 10000);
};

// 接收消息
socket.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('收到消息:', data);
    
    // 根据消息类型处理
    switch(data.type) {
        case 'connect':
            // 连接成功，获取客户端ID
            console.log('客户端ID:', data.id);
            clientId = data.id;
            break;
        case 'status':
            // 状态消息（在线状态、系统状态）
            handleStatus(data.content);
            break;
        case 'broadcast':
            // 广播消息
            handleBroadcast(data.content);
            break;
        case 'single':
            // 单播消息
            handleSingle(data);
            break;
        case 'private':
            // 私聊消息
            handlePrivate(data);
            break;
        case 'pong':
            // 心跳响应
            console.log('心跳响应');
            break;
    }
};

// 连接关闭
socket.onclose = function(event) {
    console.log('WebSocket 连接关闭');
    // 重连逻辑
    reconnect();
};

// 连接错误
socket.onerror = function(error) {
    console.error('WebSocket 错误:', error);
};
```

---

### 消息类型

| 类型 | 说明 | 必需字段 |
| :--- | :--- | :--- |
| `connect` | 连接成功响应（服务端发送） | `id`, `content` |
| `broadcast` | 广播消息（所有人可见） | `type` |
| `single` | 单播消息（发送给指定用户） | `type`, `to` |
| `private` | 私聊消息 | `type`, `to` |
| `status` | 状态消息（在线状态、系统状态） | `type`, `content` |
| `ping` | 心跳请求 | `type` |
| `pong` | 心跳响应（服务端发送） | `type` |
| `ack` | 消息确认 | `type`, `msg_id` |
| `read` | 消息已读 | `type`, `msg_id` |

---

### 发送消息示例

#### 1. 广播消息

```javascript
// 发送广播消息（所有人可见）
socket.send(JSON.stringify({
    type: 'broadcast',
    content: {
        message: '大家好！',
        user: '用户名'
    }
}));
```

#### 2. 单播消息（发送给指定用户）

```javascript
// 发送单播消息
socket.send(JSON.stringify({
    type: 'single',
    to: 'target_client_id',  // 目标客户端ID
    content: {
        message: '你好！',
        user: '用户名'
    }
}));
```

#### 3. 私聊消息

```javascript
// 发送私聊消息
socket.send(JSON.stringify({
    type: 'private',
    to: 'target_client_id',  // 目标客户端ID
    content: {
        message: '这是私聊消息',
        user: '用户名'
    }
}));
```

#### 4. 心跳检测

```javascript
// 发送心跳（建议每10-30秒发送一次）
socket.send(JSON.stringify({
    type: 'ping'
}));
```

#### 5. 消息确认（ACK）

```javascript
// 确认收到消息
socket.send(JSON.stringify({
    type: 'ack',
    msg_id: 'message_id'  // 收到的消息ID
}));
```

#### 6. 消息已读

```javascript
// 标记消息已读
socket.send(JSON.stringify({
    type: 'read',
    msg_id: 'message_id'  // 消息ID
}));
```

---

### 接收消息格式

#### 1. 连接成功消息

```json
{
    "type": "connect",
    "content": "连接成功",
    "id": "client_uuid"
}
```

#### 2. 在线状态消息

```json
{
    "type": "status",
    "content": {
        "online_users": ["client_id_1", "client_id_2"],
        "online_count": 2
    }
}
```

#### 3. 系统状态消息（每秒推送）

服务端每秒推送一次系统状态消息，包含应用信息、数据库状态、缓存状态、系统资源等信息。

```json
{
    "type": "status",
    "content": {
        "info": {
            "app_name": "INIS",
            "go_version": "go1.21.0",
            "os": "windows",
            "arch": "amd64",
            "cpu_count": 8,
            "goroutines": 50,
            "current_time": "2024-01-01 12:00:00"
        },
        "database": {
            "connected": true,
            "latency": "1.234ms",
            "error": "",
            "counts": {
                "users": 100,
                "articles": 50,
                "comments": 200,
                "pages": 10,
                "links": 20,
                "banners": 5,
                "placards": 3,
                "tags": 15
            }
        },
        "cache": {
            "enabled": true,
            "type": "redis",
            "working": true,
            "error": ""
        },
        "resource": {
            "memory": {
                "alloc": "50.00 MB",
                "total_alloc": "100.00 MB",
                "sys": "80.00 MB",
                "gc_count": 10,
                "system_total": "16.00 GB",
                "system_used": "8.00 GB",
                "system_free": "8.00 GB",
                "system_usage": "50.00%"
            },
            "cpu": {
                "count": 8,
                "model": "Intel(R) Core(TM) i7-10700K CPU @ 3.80GHz",
                "usage": "30.00%",
                "load_1m": 1.5,
                "load_5m": 2.0,
                "load_15m": 1.8
            },
            "disk": {
                "total": "500.00 GB",
                "used": "200.00 GB",
                "free": "300.00 GB",
                "usage": "40.00%",
                "fs_type": "NTFS",
                "read": "1.00 TB",
                "write": "500.00 GB",
                "read_per_sec": "10.00 MB/s",
                "write_per_sec": "5.00 MB/s",
                "io_latency": "5ms"
            },
            "network": {
                "bytes_sent": "10.00 GB",
                "bytes_recv": "20.00 GB",
                "packets_sent": 1000000,
                "packets_recv": 2000000,
                "sent_per_sec": "1.00 MB/s",
                "recv_per_sec": "2.00 MB/s",
                "up": "1.00 MB/s",
                "down": "2.00 MB/s",
                "total_sent": "10.00 GB",
                "total_received": "20.00 GB"
            },
            "system": {
                "os": "Windows 10",
                "os_version": "10.0.19045",
                "kernel": "19045",
                "boot_time": "2024-01-01 08:00:00"
            },
            "goroutines": 50
        },
        "status": "healthy",
        "timestamp": 1704067200
    }
}
```

**字段说明：**

| 字段路径 | 类型 | 说明 |
| :--- | :--- | :--- |
| `info.app_name` | string | 应用名称 |
| `info.go_version` | string | Go版本 |
| `info.os` | string | 操作系统类型 |
| `info.arch` | string | 系统架构 |
| `info.cpu_count` | int | CPU核心数 |
| `info.goroutines` | int | 当前协程数 |
| `info.current_time` | string | 当前时间 |
| `database.connected` | bool | 数据库连接状态 |
| `database.latency` | string | 数据库延迟 |
| `database.error` | string | 数据库错误信息 |
| `database.counts.users` | int | 用户数量 |
| `database.counts.articles` | int | 文章数量 |
| `database.counts.comments` | int | 评论数量 |
| `database.counts.pages` | int | 页面数量 |
| `database.counts.links` | int | 友链数量 |
| `database.counts.banners` | int | 轮播数量 |
| `database.counts.placards` | int | 公告数量 |
| `database.counts.tags` | int | 标签数量 |
| `cache.enabled` | bool | 缓存是否启用 |
| `cache.type` | string | 缓存类型（redis/file/ram） |
| `cache.working` | bool | 缓存是否正常工作 |
| `cache.error` | string | 缓存错误信息 |
| `resource.memory.alloc` | string | Go程序分配的内存 |
| `resource.memory.total_alloc` | string | Go程序累计分配内存 |
| `resource.memory.sys` | string | Go程序从系统申请的内存 |
| `resource.memory.gc_count` | int | GC执行次数 |
| `resource.memory.system_total` | string | 系统总内存 |
| `resource.memory.system_used` | string | 系统已用内存 |
| `resource.memory.system_free` | string | 系统可用内存 |
| `resource.memory.system_usage` | string | 系统内存使用率 |
| `resource.cpu.count` | int | CPU核心数 |
| `resource.cpu.model` | string | CPU型号 |
| `resource.cpu.usage` | string | CPU使用率 |
| `resource.cpu.load_1m` | float | 1分钟平均负载 |
| `resource.cpu.load_5m` | float | 5分钟平均负载 |
| `resource.cpu.load_15m` | float | 15分钟平均负载 |
| `resource.disk.total` | string | 磁盘总容量 |
| `resource.disk.used` | string | 磁盘已用容量 |
| `resource.disk.free` | string | 磁盘可用容量 |
| `resource.disk.usage` | string | 磁盘使用率 |
| `resource.disk.fs_type` | string | 文件系统类型 |
| `resource.disk.read` | string | 磁盘累计读取量 |
| `resource.disk.write` | string | 磁盘累计写入量 |
| `resource.disk.read_per_sec` | string | 磁盘读取速率 |
| `resource.disk.write_per_sec` | string | 磁盘写入速率 |
| `resource.disk.io_latency` | string | 磁盘IO延迟 |
| `resource.network.bytes_sent` | string | 网络累计发送字节 |
| `resource.network.bytes_recv` | string | 网络累计接收字节 |
| `resource.network.packets_sent` | int | 网络累计发送包数 |
| `resource.network.packets_recv` | int | 网络累计接收包数 |
| `resource.network.sent_per_sec` | string | 网络发送速率 |
| `resource.network.recv_per_sec` | string | 网络接收速率 |
| `resource.network.up` | string | 上行速率（同sent_per_sec） |
| `resource.network.down` | string | 下行速率（同recv_per_sec） |
| `resource.network.total_sent` | string | 总发送量（同bytes_sent） |
| `resource.network.total_received` | string | 总接收量（同bytes_recv） |
| `resource.system.os` | string | 操作系统名称 |
| `resource.system.os_version` | string | 操作系统版本 |
| `resource.system.kernel` | string | 内核版本 |
| `resource.system.boot_time` | string | 系统启动时间 |
| `resource.goroutines` | int | 当前协程数 |
| `status` | string | 系统健康状态（healthy/unhealthy） |
| `timestamp` | int | Unix时间戳 |

#### 4. 广播消息

```json
{
    "type": "broadcast",
    "id": "sender_client_id",
    "content": {
        "message": "大家好！",
        "user": "用户名"
    }
}
```

#### 5. 私聊消息

```json
{
    "type": "private",
    "from": "sender_client_id",
    "to": "receiver_client_id",
    "msg_id": "message_uuid",
    "content": {
        "message": "这是私聊消息"
    },
    "sent_at": 1704067200
}
```

---

### 完整前端示例

```javascript
class InisSocket {
    constructor(url) {
        this.url = url;
        this.socket = null;
        this.clientId = null;
        this.reconnectTimer = null;
        this.reconnectInterval = 5000; // 重连间隔
        this.heartbeatTimer = null;
        this.heartbeatInterval = 10000; // 心跳间隔
        this.listeners = {};
    }
    
    // 连接
    connect() {
        this.socket = new WebSocket(this.url);
        
        this.socket.onopen = () => {
            console.log('Socket 连接成功');
            this.startHeartbeat();
            this.emit('open');
        };
        
        this.socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.handleMessage(data);
        };
        
        this.socket.onclose = () => {
            console.log('Socket 连接关闭');
            this.stopHeartbeat();
            this.emit('close');
            this.reconnect();
        };
        
        this.socket.onerror = (error) => {
            console.error('Socket 错误:', error);
            this.emit('error', error);
        };
    }
    
    // 处理消息
    handleMessage(data) {
        switch(data.type) {
            case 'connect':
                this.clientId = data.id;
                this.emit('connect', data);
                break;
            case 'status':
                this.emit('status', data.content);
                break;
            case 'broadcast':
                this.emit('broadcast', data);
                break;
            case 'single':
                this.emit('single', data);
                break;
            case 'private':
                this.emit('private', data);
                // 发送ACK确认
                this.sendAck(data.msg_id);
                break;
            case 'pong':
                this.emit('pong');
                break;
        }
    }
    
    // 发送消息
    send(type, content, to = null) {
        const message = { type, content };
        if (to) message.to = to;
        this.socket.send(JSON.stringify(message));
    }
    
    // 广播消息
    broadcast(content) {
        this.send('broadcast', content);
    }
    
    // 单播消息
    single(to, content) {
        this.send('single', content, to);
    }
    
    // 私聊消息
    private(to, content) {
        this.send('private', content, to);
    }
    
    // 发送ACK确认
    sendAck(msgId) {
        this.socket.send(JSON.stringify({
            type: 'ack',
            msg_id: msgId
        }));
    }
    
    // 标记消息已读
    markRead(msgId) {
        this.socket.send(JSON.stringify({
            type: 'read',
            msg_id: msgId
        }));
    }
    
    // 开始心跳
    startHeartbeat() {
        this.heartbeatTimer = setInterval(() => {
            this.socket.send(JSON.stringify({ type: 'ping' }));
        }, this.heartbeatInterval);
    }
    
    // 停止心跳
    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }
    
    // 重连
    reconnect() {
        this.reconnectTimer = setTimeout(() => {
            console.log('尝试重连...');
            this.connect();
        }, this.reconnectInterval);
    }
    
    // 事件监听
    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }
    
    // 触发事件
    emit(event, data) {
        if (this.listeners[event]) {
            this.listeners[event].forEach(callback => callback(data));
        }
    }
    
    // 关闭连接
    close() {
        this.stopHeartbeat();
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
        }
        if (this.socket) {
            this.socket.close();
        }
    }
}

// 使用示例
const socket = new InisSocket('ws://your-domain/socket');

// 监听事件
socket.on('connect', (data) => {
    console.log('连接成功，客户端ID:', data.id);
});

socket.on('status', (content) => {
    // 处理在线状态或系统状态
    if (content.online_count !== undefined) {
        console.log('在线人数:', content.online_count);
    } else {
        // 系统状态
        console.log('CPU使用率:', content.resource.cpu.usage);
        console.log('内存使用率:', content.resource.memory.system_usage);
    }
});

socket.on('broadcast', (data) => {
    console.log('广播消息:', data.content);
});

socket.on('private', (data) => {
    console.log('私聊消息:', data.content);
    // 标记已读
    socket.markRead(data.msg_id);
});

// 连接
socket.connect();

// 发送广播消息
socket.broadcast({ message: '大家好！' });

// 发送私聊消息
socket.private('target_client_id', { message: '你好！' });
```

---

### Vue 3 示例

```vue
<template>
    <div>
        <div>在线人数: {{ onlineCount }}</div>
        <div>CPU: {{ systemStatus?.resource?.cpu?.usage }}</div>
        <div>内存: {{ systemStatus?.resource?.memory?.system_usage }}</div>
        <div v-for="msg in messages" :key="msg.msg_id">
            {{ msg.content.message }}
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue';

const socket = ref(null);
const clientId = ref('');
const onlineCount = ref(0);
const systemStatus = ref(null);
const messages = ref([]);

const connectSocket = () => {
    socket.value = new WebSocket('ws://your-domain/socket');
    
    socket.value.onopen = () => {
        console.log('连接成功');
        // 开始心跳
        setInterval(() => {
            socket.value.send(JSON.stringify({ type: 'ping' }));
        }, 10000);
    };
    
    socket.value.onmessage = (event) => {
        const data = JSON.parse(event.data);
        
        switch(data.type) {
            case 'connect':
                clientId.value = data.id;
                break;
            case 'status':
                if (data.content.online_count !== undefined) {
                    onlineCount.value = data.content.online_count;
                } else {
                    systemStatus.value = data.content;
                }
                break;
            case 'broadcast':
            case 'private':
                messages.value.push(data);
                break;
        }
    };
};

const sendMessage = (type, content, to = null) => {
    const msg = { type, content };
    if (to) msg.to = to;
    socket.value.send(JSON.stringify(msg));
};

onMounted(() => {
    connectSocket();
});

onUnmounted(() => {
    if (socket.value) {
        socket.value.close();
    }
});
</script>
```

---

### 特性说明

> 以下所有限制参数均可通过 `config/app.toml` 配置文件中的 `[socket]` 段自由调整，修改后重启服务生效。

#### 1. 离线消息缓存

当目标用户离线时，消息会被缓存，用户重连后会自动接收离线消息（可配置：缓存时间、最大条数）。

配置项：
- `offline_msg_ttl` — 离线消息过期时间（默认300秒）
- `max_offline_msgs` — 离线消息最大数量（默认100条）

#### 2. 心跳检测

- 服务端每108秒发送一次 Ping
- 客户端应在120秒内响应 Pong
- 建议客户端主动每10-30秒发送心跳

配置项：
- `ping_timeout` — Ping超时时间（默认120秒）

#### 3. 消息确认机制（ACK）

重要消息发送后会等待 ACK 确认，超时未确认会自动重试。

配置项：
- `ack_timeout` — ACK超时时间（默认10秒）
- `max_retries` — 消息重试次数（默认3次）

#### 4. 消息频率限制

每个客户端每分钟发送消息数有限制，超限消息会被丢弃。

配置项：
- `max_messages_per_minute` — 每分钟最大消息数（默认100条）

#### 5. IP连接限制

每个IP同时建立的WebSocket连接数有限制，超限会被拒绝。

配置项：
- `max_connections_per_ip` — 每个IP最大连接数（默认10个）

#### 6. IP临时封禁（可选）

当某个IP频繁达到连接上限时，可自动临时封禁该IP，避免反复重连骚扰。

配置项：
- `enable_ip_ban` — 是否启用IP临时封禁（默认false）
- `ip_ban_threshold` — IP连续超限几次后封禁（默认3次）
- `ip_ban_duration` — 封禁时长（默认300秒）

#### 7. 消息大小限制

单条消息最大1MB。

配置项：
- `max_message_size` — 最大消息大小（默认1048576字节，即1MB）

---

### 错误处理

| 错误 | 说明 | 处理建议 |
| :--- | :--- | :--- |
| 非WebSocket连接请求 | 使用HTTP协议访问 | 使用 ws:// 或 wss:// 协议 |
| IP已被封禁 | IP在黑名单中 | 联系管理员解封 |
| IP临时封禁 | 频繁超限被临时封禁 | 等待封禁到期自动解封，或调整 `ip_ban_threshold`/`ip_ban_duration` 配置 |
| 不允许的来源 | Origin不在白名单 | 配置允许的Origin |
| 连接数超限 | IP连接数超过限制 | 减少连接数，或调整 `max_connections_per_ip` 配置 |
| 消息频率超限 | 发送消息过快 | 降低发送频率，或调整 `max_messages_per_minute` 配置 |
| 消息超过最大长度 | 消息超过大小限制 | 减小消息大小，或调整 `max_message_size` 配置 |
| 无效的JSON格式 | 消息不是有效JSON | 检查消息格式 |
| 缺少必需字段: type | 消息没有type字段 | 添加type字段 |
| 私聊消息缺少必需字段: to | 私聊没有to字段 | 添加to字段 |

---

### Socket配置参考

所有 Socket 参数均在 `config/app.toml` 的 `[socket]` 段配置：

```toml
[socket]
debug = false                    # 开启调试日志
ping_timeout = 120              # ping超时时间(秒)
max_message_size = 1048576      # 最大消息大小(字节)
max_connections_per_ip = 10     # 每个IP最大连接数
max_messages_per_minute = 100   # 每分钟最大消息数
ack_timeout = 10                # ACK超时时间(秒)
max_retries = 3                 # 消息重试次数
offline_msg_ttl = 300           # 离线消息过期时间(秒)
max_offline_msgs = 100          # 离线消息最大数量
reconnect_timeout = 30          # 重连超时时间(秒)
enable_ip_ban = false           # 是否启用IP临时封禁
ip_ban_threshold = 3            # IP连续超限几次后临时封禁
ip_ban_duration = 300           # IP临时封禁时长(秒)
```

### 最佳实践

1. **心跳机制**: 客户端主动发送心跳，避免连接被服务端断开
2. **重连机制**: 连接断开后自动重连，建议间隔5秒
3. **消息确认**: 重要消息发送ACK确认
4. **消息已读**: 私聊消息标记已读状态
5. **错误处理**: 监听 onerror 和 onclose 事件
6. **资源清理**: 页面卸载时关闭连接，避免内存泄漏