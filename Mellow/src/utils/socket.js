import { API_URI, SOCKET_URI } from './runtimeConfig'
import { getCookie } from './cookie'
import { TOKEN_NAME } from '@/api/request'

/**
 * WebSocket 客户端（后端端点 /socket，见 app/socket/route/app.go）
 *
 * 用途：后台「数据统计」页消费服务端主动推送的实时数据（每秒一条），
 * 目前服务端推送的消息类型：
 *
 *   { type: 'connect', content: '连接成功', id }
 *   { type: 'pong' }
 *   { type: 'status', content: { online_users, online_count } }        ← 广播给所有人
 *   { type: 'status', content: { info, database, cache, resource }, admin_only: true }
 *                                                                     ← 仅推送给管理员
 *   { type: 'kicked', content: '…' }                                   ← 被管理员踢下线（服务端随后断连）
 *   { type: 'single' | 'private' | 'broadcast', ... }                  ← 聊天 / 通知类
 *   { type: 'chat_typing', data: { from_uid } }                        ← 对方正在输入（私聊）
 *   { type: 'room_typing', data: { uid } }                             ← 有人在群聊输入（全员广播）
 *   { type: 'chat_mention', data: { room_id, message_id, from_uid, preview } }
 *                                                                     ← 群聊里有人 @ 我
 *
 * 可发送的消息：
 *   { type: 'ping' }                                   ← 心跳
 *   { type: 'auth', token }                            ← 连接后认领登录身份
 *   { type: 'ack', data: { target_uid, seq } }         ← 私信送达确认（批量，seq 即消息 id）
 *   { type: 'read', data: { target_uid, seq } }        ← 私信已读水位（可选，正常走 HTTP）
 *   { type: 'typing', to: 'user_<uid>' }               ← 告诉对方「我在打字」（私聊，服务端 1.5s 节流）
 *   { type: 'typing', room: 1 }                        ← 告诉群聊「我在打字」
 *   { type: 'kick', to: '<client_id>' }                 ← 断开某个连接（仅管理员，服务端会校验身份）
 *
 * 两个关键点：
 *   1. 浏览器无法给 WebSocket 握手设置请求头，跨域时也带不上 Cookie ——
 *      所以鉴权按服务端约定走「连接后发 { type: 'auth', token }」升级身份
 *      （见 app/socket/controller/index.go 的 handleAuth，成功后会重新广播在线状态）；
 *   2. 服务端有 WS 层 ping，客户端浏览器会自动回 pong；这里再补一层应用层 ping，
 *      用于刷新服务端的「最后活跃时间」，避免被判定为僵尸连接。
 */

// 重连退避（毫秒）：先快后慢，服务端重启时不至于被空转打满
const RECONNECT_DELAYS = [1000, 2000, 3000, 5000, 8000, 13000, 20000]

// 应用层心跳间隔（毫秒）
const PING_INTERVAL = 30000

// 连续失败上限：达到后转入低速重试（状态 failed），不再按退避表高频重试
const MAX_ATTEMPTS = 8

// 低速重试间隔（毫秒）
//
// 失败不等于永久放弃：反向代理重启、切 Wi-Fi / 蜂窝、休眠唤醒之后，
// 连接必须能自己回来。否则用户看到的现象就是「消息不再实时，
// 必须刷新页面才恢复」——那正是客户端最常见的投诉。
const SLOW_RETRY_INTERVAL = 30000

/** 读取登录 token（与 api/request.js 同一约定：cookie 优先，回退 localStorage） */
function readToken() {
  try {
    return getCookie(TOKEN_NAME) || localStorage.getItem('mellow_token') || ''
  } catch {
    return ''
  }
}

/**
 * 解析 WebSocket 地址
 *
 * 取值优先级：运行时 socketUri > 接口根地址 apiUri > 当前站点（同源）。
 * 支持四种写法：wss://host/socket、wss://host、//host、/socket（以及直接写域名 host）。
 * 路径固定为 /socket —— 以 / 开头是绝对路径，会覆盖 base 的 path，
 * 所以填 `wss://host` 与 `wss://host/socket` 等价（后端就注册在 /socket）。
 */
export function socketUrl() {
  const loc = typeof window !== 'undefined' ? window.location : null
  const origin = loc?.origin || 'http://localhost'

  let base = String(SOCKET_URI || API_URI || '').trim()
  if (!base) {
    base = origin
  } else if (/^[a-z][a-z0-9+.-]*:\/\//i.test(base)) {
    // 已带协议（http / https / ws / wss），直接使用
  } else if (base.startsWith('//')) {
    // 协议相对（//host）：跟随当前页面协议
    base = `${loc?.protocol || 'https:'}${base}`
  } else if (base.startsWith('/')) {
    // 同源路径（/socket）
    base = `${origin}${base}`
  } else {
    // 只写了域名或「域名:端口」
    base = `${loc?.protocol || 'https:'}//${base}`
  }

  const url = new URL('/socket', base)

  // 协议归一：https / wss → wss，其余 → ws；
  // 页面为 https 时强制 wss，否则浏览器会按「混合内容」直接拦掉 ws 连接
  const secure = url.protocol === 'https:' || url.protocol === 'wss:' || loc?.protocol === 'https:'
  url.protocol = secure ? 'wss:' : 'ws:'

  return url.toString()
}

/**
 * 创建 socket 客户端
 * @param {object} options
 * @param {(payload: object) => void} options.onMessage 收到 JSON 消息
 * @param {(state: string) => void}   [options.onState] 连接状态：connecting / open / closed / failed / kicked
 * @param {(payload: object) => void} [options.onKicked] 被管理员断开时回调（此后不再自动重连）
 * @param {boolean}                   [options.autoReconnect] 断线自动重连（默认 true）
 * @returns {{ connect: Function, close: Function, send: Function, state: string }}
 */
export function createSocketClient(options = {}) {
  const { onMessage, onState, onKicked, autoReconnect = true } = options

  let ws = null
  let timer = null
  let pingTimer = null
  let attempt = 0
  let manual = false
  let state = 'closed'
  let warned = false

  /** 连续失败达到上限后转入低速重试（状态 failed）
   *  目的：配置/代理有问题时不要无限重试刷屏（浏览器每个失败连接都会打一条错误），
   *  但也不能就此放弃 —— 每 30 秒再试一轮，成功后自动回到正常退避。 */
  function failOut() {
    setState('failed')
    if (!warned) {
      warned = true
      console.warn(
        `[socket] 连续 ${MAX_ATTEMPTS} 次连接失败，转入 ${SLOW_RETRY_INTERVAL / 1000}s 低速重试：${socketUrl()}\n` +
          '常见原因：① 反向代理未放通 /socket 的 WebSocket 升级（Upgrade / Connection: upgrade）；' +
          '② 本地开发未在 vite.config.js 里代理 /socket；' +
          '③ socketUri / VITE_SOCKET 指向了错误地址。\n' +
          '排查：浏览器控制台若报 "Unexpected response code: 200/404"，基本就是 ①②；' +
          '修好后点页面上的「重新连接」即可，不必刷新。'
      )
    }

    clearTimeout(timer)
    timer = setTimeout(() => {
      if (manual) return
      // 新一轮从最短退避重新开始：恢复网络后应当立刻可用
      attempt = 0
      connect()
    }, SLOW_RETRY_INTERVAL)
  }

  function setState(next) {
    if (state === next) return
    state = next
    onState?.(next)
  }

  function stopPing() {
    if (pingTimer) {
      clearInterval(pingTimer)
      pingTimer = null
    }
  }

  function startPing() {
    stopPing()
    pingTimer = setInterval(() => send({ type: 'ping' }), PING_INTERVAL)
  }

  function send(payload) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return false
    try {
      ws.send(typeof payload === 'string' ? payload : JSON.stringify(payload))
      return true
    } catch {
      return false
    }
  }

  function scheduleReconnect() {
    if (attempt >= MAX_ATTEMPTS) {
      failOut()
      return
    }
    const delay = RECONNECT_DELAYS[Math.min(attempt, RECONNECT_DELAYS.length - 1)]
    attempt += 1
    clearTimeout(timer)
    timer = setTimeout(connect, delay)
  }

  function connect() {
    manual = false
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    if (typeof WebSocket === 'undefined') return

    clearTimeout(timer)
    setState('connecting')

    try {
      ws = new WebSocket(socketUrl())
    } catch {
      setState('closed')
      if (autoReconnect) scheduleReconnect()
      return
    }

    ws.onopen = () => {
      attempt = 0
      warned = false
      setState('open')
      // 升级为管理员连接（跨域 / 代理场景握手带不上 Cookie 时的唯一凭证）
      const token = readToken()
      if (token) send({ type: 'auth', token })
      startPing()
    }

    ws.onmessage = (event) => {
      if (typeof event.data !== 'string') return

      // 服务端的写循环会把发送队列里的多条消息拼进同一帧，用 '\n' 分隔
      // （app/socket/controller/index.go 的 write() + serialize.go 的 line）。
      // 所以必须按行拆开逐条解析：整帧 JSON.parse 会直接抛错，导致数据被静默丢弃。
      // Go 的 json.Marshal 会把字符串内的换行转义成 \n 两个字符，因此按真实换行字节拆分是安全的。
      for (const chunk of event.data.split('\n')) {
        const text = chunk.trim()
        if (!text) continue
        let payload = null
        try {
          payload = JSON.parse(text)
        } catch {
          continue
        }
        if (!payload || typeof payload !== 'object') continue

        // 被管理员踢下线：立刻停止自动重连（否则会马上又连回来，踢了等于没踢），
        // 由调用方决定怎么提示用户；服务端随后会关闭连接，onclose 里 manual=true 不会再排程
        if (payload.type === 'kicked') {
          manual = true
          clearTimeout(timer)
          stopPing()
          setState('kicked')
          onKicked?.(payload)
          continue
        }

        onMessage?.(payload)
      }
    }

    // 出错后浏览器一定会触发 onclose，重连逻辑统一放在 onclose，避免重复排程
    ws.onerror = () => {}

    ws.onclose = () => {
      stopPing()
      ws = null
      // 被踢下线时服务端紧接着就会关闭连接：保留 kicked 状态，
      // 否则界面上「已被管理员断开」的提示会被覆盖成普通的「已断开」
      if (state !== 'kicked') setState('closed')
      if (!manual && autoReconnect) scheduleReconnect()
    }
  }

  function close() {
    manual = true
    clearTimeout(timer)
    stopPing()
    detachWake()
    if (ws) {
      try {
        ws.close()
      } catch {
        /* 忽略：关闭失败不影响后续状态 */
      }
      ws = null
    }
    setState('closed')
  }

  /**
   * 立刻重试一次（不等退避计时）
   *
   * 手机锁屏、切走标签页、Wi-Fi 与蜂窝互切，长连接经常是「悄无声息」地断掉：
   * 对端已经消失，但 onclose 要等系统回收才触发。这些场景下主动补一次，
   * 用户回到页面就能立刻恢复实时消息。
   */
  function wake() {
    if (manual) return
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return
    clearTimeout(timer)
    attempt = 0
    warned = false
    connect()
  }

  function onVisibilityChange() {
    if (document.visibilityState === 'visible') wake()
  }

  // 监听网络恢复与「回到前台」：这两个时机是先于用户操作的，
  // 早一步连上，用户点进来的第一眼就能看到新消息
  const hasDom = typeof window !== 'undefined' && typeof document !== 'undefined'
  if (hasDom) {
    window.addEventListener('online', wake)
    document.addEventListener('visibilitychange', onVisibilityChange)
  }

  function detachWake() {
    if (!hasDom) return
    window.removeEventListener('online', wake)
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }

  return {
    connect,
    close,
    send,
    // 主动重连（用户点「重新连接」时用：清掉退避计数与失败上限立即重试）
    reconnect() {
      clearTimeout(timer)
      attempt = 0
      warned = false
      if (ws) {
        manual = true
        try {
          ws.close()
        } catch {
          /* 忽略 */
        }
        ws = null
      }
      connect()
    },
    get state() {
      return state
    }
  }
}
