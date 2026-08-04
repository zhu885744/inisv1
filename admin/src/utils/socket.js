import utils from '{src}/utils/utils'

// socket 实例
let socket = null
// 客户端ID
let clientId = null
// 心跳定时器
let heartbeatTimer = null
// 重连定时器
let reconnectTimer = null
// 事件监听器
const listeners = {}
// 阶梯重连延迟序列（逐步拉长，上限30s，和服务reconnect_timeout对齐）
const reconnectDelayList = [1000, 3000, 8000, 15000, 30000]
let delayIndex = 0

// 配置
const config = {
    reconnectInterval: globalThis?.inis?.socket?.reconnect_interval || 5000,
    heartbeatInterval: globalThis?.inis?.socket?.heartbeat_interval || 15000, // 拉长至15s，减少心跳压力
}

// 获取 socket URI
const getSocketUri = () => {
    // 优先使用 config.js 中配置的 socket.uri
    const configUri = globalThis?.inis?.socket?.uri
    if (configUri) return configUri

    // 判断是否为生产环境
    const isProduction = import.meta.env.MODE === 'production'

    if (isProduction) {
        // 生产环境：使用 globalThis.inis.api.uri
        const uri = globalThis?.inis?.api?.uri
        if (uri) return uri.replace('http', 'ws') + '/socket'
    } else {
        // 开发环境：使用环境变量
        const { VITE_SOCKET } = import.meta.env
        if (VITE_SOCKET) return VITE_SOCKET
    }

    return null
}

// socket 连接
const connect = (uri = null, params = {}) => {
    // 重置重连阶梯计数（连接成功后清零）
    delayIndex = 0
    // 处理服务端地址
    uri = !utils.is.empty(uri) ? uri : getSocketUri()

    // 额外的连接参数
    if (!utils.is.empty(params)) {
        if (typeof params === 'string') uri += `?${params}`
        else if (typeof params === 'object') {
            const array = []
            for (const key in params) array.push(`${key}=${params[key]}`)
            uri += `?${array.join('&')}`
        }
    }

    // 如果已存在连接，先正常关闭
    if (socket) {
        socket.close(1000, 'reconnect new link')
        socket = null
    }

    socket = new WebSocket(uri)

    // 连接打开
    socket.onopen = (event) => {
        console.log('Socket 连接成功')
        // 启动心跳
        startHeartbeat()
        // 触发事件
        emit('open', event)
    }

    // 接收消息
    socket.onmessage = (event) => {
        try {
            // 尝试解析 JSON，如果失败则作为原始字符串处理
            let data
            try {
                data = JSON.parse(event.data)
            } catch {
                // 非 JSON 格式，直接作为消息事件传递
                data = { type: 'message', raw: event.data }
            }
            handleMessage(data)
        } catch (error) {
            console.error('处理消息失败:', error)
        }
    }

    // 连接关闭
    socket.onclose = (event) => {
        console.log('Socket 连接关闭 code:', event.code, 'reason:', event.reason)
        // 停止心跳
        stopHeartbeat()
        // 触发事件
        emit('close', event)
        // 关键：正常关闭码1000 不自动重连，仅异常断开重连
        if (event.code !== 1000) {
            reconnect()
        }
    }

    // 连接错误
    socket.onerror = (error) => {
        console.error('Socket 错误:', error)
        emit('error', error)
    }

    // 页面切后台/前台保活监听（仅绑定一次）
    if (!document._wsVisibilityBind) {
        document._wsVisibilityBind = true
        document.addEventListener('visibilitychange', () => {
            // 页面隐藏、后台运行时主动发一次心跳保活
            if (document.hidden && socket && socket.readyState === WebSocket.OPEN) {
                ping()
            }
        })
    }

    return socket
}

// 处理消息
const handleMessage = (data) => {
    // 如果是原始消息类型，直接触发 message 事件
    if (data.type === 'message' && data.raw) {
        emit('message', data.raw)
        return
    }

    console.log('Socket 收到消息:', data)

    switch (data.type) {
        case 'connect':
            // 连接成功，获取客户端ID
            clientId = data.id
            emit('connect', data)
            break
        case 'status':
            // 状态消息（在线状态、系统状态）
            console.log('Status 消息内容:', data.content)
            emit('status', data.content)
            break
        case 'broadcast':
            // 广播消息
            emit('broadcast', data)
            break
        case 'single':
            // 单播消息
            emit('single', data)
            break
        case 'private':
            // 私聊消息
            emit('private', data)
            // 发送 ACK 确认
            if (data.msg_id) sendAck(data.msg_id)
            break
        case 'pong':
            // 心跳响应
            emit('pong')
            break
        case 'ack':
            // ACK 确认响应
            emit('ack', data)
            break
        default:
            // 其他消息类型
            emit('message', data)
    }
}

// 发送消息
const send = (data = {}) => {
    try {
        if (!socket || socket.readyState !== 1) {
            console.log('socket 未连接')
            return false
        }
        socket.send(JSON.stringify(data))
        return true
    } catch (error) {
        console.error('发送消息失败:', error)
        return false
    }
}

// 广播消息（所有人可见）
const broadcast = (content) => {
    return send({ type: 'broadcast', content })
}

// 单播消息（发送给指定用户）
const single = (to, content) => {
    return send({ type: 'single', to, content })
}

// 私聊消息
const sendPrivate = (to, content) => {
    return send({ type: 'private', to, content })
}

// 发送心跳
const ping = () => {
    return send({ type: 'ping' })
}

// 发送 ACK 确认
const sendAck = (msgId) => {
    return send({ type: 'ack', msg_id: msgId })
}

// 标记消息已读
const markRead = (msgId) => {
    return send({ type: 'read', msg_id: msgId })
}

// 启动心跳
const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
        ping()
    }, config.heartbeatInterval)
}

// 停止心跳
const stopHeartbeat = () => {
    if (heartbeatTimer) {
        clearInterval(heartbeatTimer)
        heartbeatTimer = null
    }
}

// 阶梯退避重连（核心优化）
const reconnect = () => {
    if (reconnectTimer) return
    // 获取当前阶梯延迟，不超过最大值
    const waitMs = reconnectDelayList[Math.min(delayIndex, reconnectDelayList.length - 1)]
    delayIndex++
    reconnectTimer = setTimeout(() => {
        console.log(`阶梯重连，等待${waitMs/1000}s后尝试连接`)
        reconnectTimer = null
        connect()
    }, waitMs)
}

// 事件监听
const on = (event, callback) => {
    if (!listeners[event]) {
        listeners[event] = []
    }
    listeners[event].push(callback)
}

// 触发事件
const emit = (event, data) => {
    if (listeners[event]) {
        listeners[event].forEach(callback => {
            try {
                callback(data)
            } catch (error) {
                console.error(`事件 ${event} 处理失败:`, error)
            }
        })
    }
}

// 移除事件监听
const off = (event, callback) => {
    if (!listeners[event]) return
    if (callback) {
        listeners[event] = listeners[event].filter(cb => cb !== callback)
    } else {
        delete listeners[event]
    }
}

// 主动关闭连接（正常关闭码1000，不触发自动重连）
const close = () => {
    stopHeartbeat()
    if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = null
    }
    if (socket) {
        socket.close(1000, 'user manual close')
        socket = null
    }
    clientId = null
    delayIndex = 0
}

// 获取客户端ID
const getClientId = () => clientId

// 获取连接状态
const isConnected = () => socket && socket.readyState === 1

export default {
    connect,
    send,
    broadcast,
    single,
    sendPrivate,
    ping,
    sendAck,
    markRead,
    startHeartbeat,
    stopHeartbeat,
    reconnect,
    on,
    off,
    close,
    getClientId,
    isConnected,
    // 兼容旧 API
    open: (cb) => on('open', cb),
    message: (cb) => on('message', cb),
    error: (cb) => on('error', cb),
}