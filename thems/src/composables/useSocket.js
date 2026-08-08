import { ref, onBeforeUnmount, getCurrentInstance } from 'vue'
import utils from '@/utils/utils'
import { getSync } from '@/utils/app'
import { request } from '@/utils/network'

const DEV = import.meta.env.DEV || false

// 单例实例和状态
let socketInstance = null
let listeners = new Map()
let heartbeatTimer = null
let reconnectTimer = null
let reconnectCount = 0
const MAX_RECONNECT_COUNT = 10
const RECONNECT_INTERVAL = 5000
const HEARTBEAT_INTERVAL = 10000

// 连接状态
const connectionState = ref('disconnected') // disconnected | connecting | connected | error
const clientId = ref('')
const lastError = ref(null)

/**
 * 获取WebSocket连接URL
 * - 优先直接读取 VITE_SOCKET 环境变量（完整地址，无论生产/开发）
 * - 兼容跨域场景，token通过query参数传递
 */
const getSocketUrl = () => {
  const TOKEN_NAME = getSync('token_name') || 'INIS_LOGIN_TOKEN'
  const envSocket = import.meta.env.VITE_SOCKET

  let baseSocketUrl = ''

  // 优先级1: 直接使用 .env 中 VITE_SOCKET（用户要求无论生产/开发都直接读这个）
  if (envSocket) {
    baseSocketUrl = envSocket.replace(/\/$/, '')
  }
  // 优先级2: 回退逻辑（没有配置 VITE_SOCKET 时才尝试从 API_URI 推导）
  else {
    const baseUrl = request.getBaseURL() || import.meta.env.VITE_API_URI || ''
    if (baseUrl) {
      let wsBase = baseUrl.replace(/\/$/, '')
      // 将 http(s) 转换为 ws(s)
      wsBase = wsBase.replace(/^https?:\/\//, (match) => {
        return match === 'https://' ? 'wss://' : 'ws://'
      })
      baseSocketUrl = `${wsBase}/socket`
    } else if (typeof window !== 'undefined') {
      // 最终回退：当前 host
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      baseSocketUrl = `${protocol}//${window.location.host}/socket`
    }
  }

  // 获取 token
  let token = utils.get.cookie(TOKEN_NAME) || ''
  if (!token) {
    const authHeader = request.axios?.defaults?.headers?.common?.Authorization
    if (authHeader) {
      token = authHeader
    }
  }

  // 拼接 query token（跨域场景要求）
  if (token) {
    const separator = baseSocketUrl.includes('?') ? '&' : '?'
    return `${baseSocketUrl}${separator}${encodeURIComponent(TOKEN_NAME)}=${encodeURIComponent(token)}&token=${encodeURIComponent(token)}`
  }

  return baseSocketUrl
}

/**
 * 脱敏 URL 中的敏感 token 参数
 */
const maskSocketUrl = (url) => {
  return url
    .replace(/token=[^&]+/g, 'token=***')
    .replace(/INIS_LOGIN_TOKEN=[^&]+/g, 'INIS_LOGIN_TOKEN=***')
    .replace(/([?&])([^=]+_TOKEN)=[^&]+/g, '$1$2=***')
}

/**
 * 启动心跳
 */
const startHeartbeat = () => {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    if (socketInstance && socketInstance.readyState === WebSocket.OPEN) {
      try {
        socketInstance.send(JSON.stringify({ type: 'ping' }))
      } catch (e) {
        console.warn('[Socket] 发送心跳失败:', e)
      }
    }
  }, HEARTBEAT_INTERVAL)
}

/**
 * 停止心跳
 */
const stopHeartbeat = () => {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

/**
 * 执行重连
 */
const scheduleReconnect = () => {
  if (reconnectTimer) return
  if (reconnectCount >= MAX_RECONNECT_COUNT) {
    console.warn('[Socket] 达到最大重连次数，停止重连')
    connectionState.value = 'error'
    return
  }

  reconnectCount++
  const delay = Math.min(RECONNECT_INTERVAL * reconnectCount, 30000)
  
  if (DEV) {
    console.log(`[Socket] ${delay / 1000}秒后尝试第${reconnectCount}次重连...`)
  }

  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    connect()
  }, delay)
}

/**
 * 取消重连计划
 */
const cancelReconnect = () => {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

/**
 * 解析消息内容 - 先检查data层，再回退到content层
 */
const parseContent = (data) => {
  if (!data) return null
  // 自定义 toMapStringAny 安全转换逻辑
  const toMapStringAny = (val) => {
    if (val === null || val === undefined) return null
    if (typeof val === 'object') return val
    if (typeof val === 'string') {
      try {
        return JSON.parse(val)
      } catch {
        return { message: val }
      }
    }
    return { value: val }
  }

  // 优先检查 data 层
  if (data.data !== undefined) {
    return toMapStringAny(data.data)
  }
  // 回退到 content 层
  if (data.content !== undefined) {
    return toMapStringAny(data.content)
  }
  return data
}

/**
 * 将原始字符串拆分为多条 JSON（后端可能把多条 JSON 拼在同一个 WebSocket 帧里）
 * 返回解析好的对象数组
 * 
 * 算法：用"括号深度计数"精确定位每个 JSON 的边界，避免正则/字符串分割出错
 */
const splitAndParseMultipleJson = (raw) => {
  if (!raw || typeof raw !== 'string') return []

  const result = []
  const str = raw
  let i = 0
  const len = str.length

  while (i < len) {
    // 跳过空白字符（空格、\r、\n、\t 等）
    while (i < len && /\s/.test(str.charAt(i))) {
      i++
    }
    if (i >= len) break

    const firstChar = str.charAt(i)
    // JSON 只可能以 { 或 [ 开头
    if (firstChar !== '{' && firstChar !== '[') {
      // 不合法的字符，跳过并告警
      console.warn('[Socket] 原始数据存在非 JSON 起始字符，位置:', i, 'char:', firstChar)
      break
    }

    let depth = 0
    let inString = false
    let escape = false
    let start = i

    for (; i < len; i++) {
      const ch = str.charAt(i)

      if (inString) {
        if (escape) {
          escape = false
          continue
        }
        if (ch === '\\') {
          escape = true
          continue
        }
        if (ch === '"') {
          inString = false
        }
        continue
      }

      // 不在字符串中
      if (ch === '"') {
        inString = true
        continue
      }
      if (ch === '{' || ch === '[') {
        depth++
        continue
      }
      if (ch === '}' || ch === ']') {
        depth--
        if (depth === 0) {
          // 完整 JSON 结束
          i++
          const jsonStr = str.substring(start, i)
          try {
            result.push(JSON.parse(jsonStr))
          } catch (err) {
            console.warn('[Socket] 子 JSON 解析失败:', jsonStr.substring(0, 100), err)
          }
          // 跳出内层 for，回到外层 while 继续处理下一条
          break
        }
      }
    }

    // 如果 for 循环正常结束（非 break），说明没找到配对的结束括号
    if (depth !== 0) {
      console.warn('[Socket] JSON 括号未闭合，剩余部分丢弃:', str.substring(start).substring(0, 100))
      break
    }
  }

  return result
}

/**
 * 分发并处理单条消息
 */
const dispatchMessage = (message) => {
  if (!message || typeof message !== 'object') return

  const { type } = message
  const content = parseContent(message)

  if (DEV) {
    console.log('[Socket] 收到消息:', type, content || message)
  }

  switch (type) {
    case 'connect':
      clientId.value = message.id || ''
      reconnectCount = 0
      break
    case 'pong':
      // 心跳响应，不需要特殊处理
      break
    case 'ack':
      // 消息确认
      break
    case 'status':
    case 'broadcast':
    case 'single':
    case 'private':
      // 私聊消息自动发送ACK
      if (type === 'private' && message.msg_id) {
        try {
          sendMessage({ type: 'ack', msg_id: message.msg_id })
        } catch (e) {}
      }
      break
  }

  // 触发对应类型的监听器
  if (listeners.has(type)) {
    listeners.get(type).forEach(cb => {
      try {
        cb(content !== null ? content : message, message)
      } catch (e) {
        console.error(`[Socket] 监听器执行错误 [${type}]:`, e)
      }
    })
  }

  // 触发通配符监听器
  if (listeners.has('*')) {
    listeners.get('*').forEach(cb => {
      try {
        cb(type, content !== null ? content : message, message)
      } catch (e) {
        console.error('[Socket] 通配符监听器执行错误:', e)
      }
    })
  }
}

/**
 * 处理收到的消息（支持一帧内多条拼接的 JSON）
 */
const handleMessage = (event) => {
  const raw = event.data
  let messages

  // 如果是二进制 Blob，转文本再解析
  if (typeof raw !== 'string') {
    if (raw instanceof Blob) {
      raw.text().then(text => {
        const parsed = splitAndParseMultipleJson(text)
        parsed.forEach(dispatchMessage)
      }).catch(e => {
        console.warn('[Socket] 二进制消息转文本失败:', e)
      })
      return
    }
    console.warn('[Socket] 收到不支持的消息类型:', typeof raw)
    return
  }

  messages = splitAndParseMultipleJson(raw)

  // 如果一条都解析不出来，打告警（保留原始内容前200字符便于排查）
  if (messages.length === 0) {
    console.warn('[Socket] 消息解析失败: 未找到有效 JSON，原始数据:', raw.substring(0, 200))
    return
  }

  messages.forEach(dispatchMessage)
}

/**
 * 建立连接
 */
const connect = () => {
  // 已经连接或正在连接则跳过
  if (socketInstance && (socketInstance.readyState === WebSocket.OPEN || socketInstance.readyState === WebSocket.CONNECTING)) {
    return
  }

  connectionState.value = 'connecting'
  lastError.value = null

  try {
    const url = getSocketUrl()
    if (DEV) {
      console.log('[Socket] 开始连接:', maskSocketUrl(url))
    }

    socketInstance = new WebSocket(url)

    socketInstance.onopen = () => {
      if (DEV) {
        console.log('[Socket] 连接成功')
      }
      connectionState.value = 'connected'
      reconnectCount = 0
      startHeartbeat()
      emit('open', {})
    }

    socketInstance.onmessage = handleMessage

    socketInstance.onclose = (event) => {
      if (DEV) {
        console.log('[Socket] 连接关闭 code:', event.code, 'reason:', event.reason)
      }
      connectionState.value = 'disconnected'
      stopHeartbeat()
      emit('close', { code: event.code, reason: event.reason })
      
      // 非正常关闭触发重连（1000是正常关闭码）
      if (event.code !== 1000) {
        scheduleReconnect()
      }
    }

    socketInstance.onerror = (error) => {
      console.error('[Socket] 连接错误:', error)
      connectionState.value = 'error'
      lastError.value = error
      emit('error', error)
    }
  } catch (error) {
    console.error('[Socket] 创建连接失败:', error)
    connectionState.value = 'error'
    lastError.value = error
    scheduleReconnect()
  }
}

/**
 * 关闭连接
 */
const disconnect = () => {
  cancelReconnect()
  stopHeartbeat()
  
  if (socketInstance) {
    try {
      socketInstance.close(1000, 'Client disconnect')
    } catch (e) {}
    socketInstance = null
  }
  
  connectionState.value = 'disconnected'
  clientId.value = ''
}

/**
 * 发送消息
 */
const sendMessage = (data) => {
  if (!socketInstance || socketInstance.readyState !== WebSocket.OPEN) {
    throw new Error('Socket未连接')
  }
  socketInstance.send(JSON.stringify(data))
}

/**
 * 注册事件监听
 */
const on = (event, callback) => {
  if (!listeners.has(event)) {
    listeners.set(event, new Set())
  }
  listeners.get(event).add(callback)
  
  // 返回取消监听函数
  return () => off(event, callback)
}

/**
 * 移除事件监听
 */
const off = (event, callback) => {
  if (listeners.has(event)) {
    listeners.get(event).delete(callback)
    if (listeners.get(event).size === 0) {
      listeners.delete(event)
    }
  }
}

/**
 * 触发事件
 */
const emit = (event, data) => {
  if (listeners.has(event)) {
    listeners.get(event).forEach(cb => {
      try {
        cb(data)
      } catch (e) {
        console.error(`[Socket] 事件派发错误 [${event}]:`, e)
      }
    })
  }
}

/**
 * 组件内使用的 composable
 */
export const useSocket = () => {
  const instance = getCurrentInstance()
  
  const componentCleanups = []

  // 组件卸载时自动清理监听器
  if (instance) {
    onBeforeUnmount(() => {
      componentCleanups.forEach(cleanup => {
        try { cleanup() } catch (e) {}
      })
      componentCleanups.length = 0
    })
  }

  const onEvent = (event, callback) => {
    const cleanup = on(event, callback)
    componentCleanups.push(cleanup)
    return cleanup
  }

  const send = (type, content, to = null) => {
    const msg = { type, content }
    if (to) msg.to = to
    sendMessage(msg)
  }

  const broadcast = (content) => send('broadcast', content)
  const single = (to, content) => send('single', content, to)
  const privateMsg = (to, content) => send('private', content, to)
  const sendAck = (msgId) => sendMessage({ type: 'ack', msg_id: msgId })
  const markRead = (msgId) => sendMessage({ type: 'read', msg_id: msgId })

  return {
    // 状态
    connectionState,
    clientId,
    lastError,
    
    // 连接管理
    connect,
    disconnect,
    
    // 发送消息
    send,
    sendMessage,
    broadcast,
    single,
    private: privateMsg,
    sendAck,
    markRead,
    
    // 事件监听
    on: onEvent,
    off
  }
}

// 全局单例导出（供App.vue等非组件场景使用）
export const socketManager = {
  connectionState,
  clientId,
  lastError,
  connect,
  disconnect,
  send: (type, content, to = null) => {
    const msg = { type, content }
    if (to) msg.to = to
    sendMessage(msg)
  },
  sendMessage,
  broadcast: (content) => sendMessage({ type: 'broadcast', content }),
  single: (to, content) => sendMessage({ type: 'single', to, content }),
  private: (to, content) => sendMessage({ type: 'private', to, content }),
  sendAck: (msgId) => sendMessage({ type: 'ack', msg_id: msgId }),
  markRead: (msgId) => sendMessage({ type: 'read', msg_id: msgId }),
  on,
  off
}

// 页面卸载时关闭连接，避免内存泄漏
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => {
    disconnect()
  })
  window.addEventListener('pagehide', () => {
    disconnect()
  })
}

export default useSocket
