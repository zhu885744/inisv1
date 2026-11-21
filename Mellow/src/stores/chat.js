import { reactive, ref } from 'vue'
import { defineStore } from 'pinia'
import { createSocketClient } from '@/utils/socket'
import { playNoticeSound } from '@/utils/noticeSound'
import { ackChatMessages, getChatUnread } from '@/api/chat'
import { useUserStore } from './user'

// ACK 批量上报的合并窗口（毫秒）：连收多条消息只上报一次最大的 seq
const ACK_DELAY = 800

// 新消息提醒偏好的本地存储键
// 只存本机：这是「这台设备要不要响 / 要不要弹」的偏好，不需要同步到账号
const ALERT_KEY = 'mellow_chat_alert'

// 「正在输入」上报间隔（毫秒）：服务端还有 1.5s 的兜底节流
const TYPING_INTERVAL = 3000

/**
 * 私信（用户私聊）实时通道
 *
 * 复用系统现有的 WebSocket（/socket）：
 *   - 连接后自动发 {type:'auth', token} 认领身份（见 utils/socket.js）；
 *   - 服务端把私聊消息以 { type:'private', data:{ id, seq, from_uid, to_uid, content, ... } } 推下来，
 *     见 app/api/controller/chat.go 的 send 与 app/socket/controller/chat.go。
 *
 * 可靠投递的两条补充机制（与后端 chat_states 两张水位对应）：
 *   - ACK：收到消息后把「这个会话我收到的最大 seq」合并上报（socket type:'ack'，
 *     连不上时退回 PUT chat/ack），服务端记为 last_ack_seq，只表示送达；
 *   - 补拉：断线重连时发 {type:'reconnect'} 事件，聊天页用本地最大 seq 当 after_seq
 *     调 GET chat/sync，把断线期间漏掉的推送补齐。
 *
 * 全局只保持一条连接（App.vue 里按登录态启停），
 * 这样「不在聊天页」也能拿到新消息并按需更新未读数。
 */
export const useChatStore = defineStore('chat', () => {
  // 私信未读总数（用户中心菜单角标）
  const unread = ref(0)
  // WebSocket 是否已连上（聊天页的在线提示用，不代表对方在线）
  const connected = ref(false)
  // 最近一条收到的消息
  const incoming = ref(null)

  // 在线成员快照（服务端 status 广播，已按 uid 去重）
  //
  // 群聊的「N 人在线」、@ 选人面板都读这里：
  // 服务端的在线列表是按连接算的（多端会重复），这里统一收敛成「一个人一条」。
  const presence = ref({ count: 0, users: [] })

  // 群聊里我被 @ 的次数（私信列表的房间入口显示红点，进房间清零）
  const roomMentions = ref(0)

  // 免打扰的会话（对方 uid 集合）
  //
  // 只影响提醒，不影响未读：免打扰的会话照常累加角标，
  // 否则用户会以为消息没收到。
  const mutedPeers = ref(new Set())

  // 订阅者：聊天页打开时挂一个，用来把新消息追加到当前对话
  const listeners = new Set()
  // 事件订阅：已读回执 / 撤回（不改变未读数，只改当前会话的气泡状态）
  const eventListeners = new Set()
  // 新消息提醒的订阅者（右下角悬浮通知，见 components/ChatNotice.vue）
  const noticeListeners = new Set()

  // 消息提醒偏好：提示音 / 页内浮层 / 系统桌面通知
  //
  // 桌面通知默认关：它需要浏览器授权，不能在用户没同意的情况下偷偷开。
  const alert = reactive({ sound: true, toast: true, desktop: false })
  try {
    const saved = JSON.parse(localStorage.getItem(ALERT_KEY) || '{}')
    if (typeof saved.sound === 'boolean') alert.sound = saved.sound
    if (typeof saved.toast === 'boolean') alert.toast = saved.toast
    if (typeof saved.desktop === 'boolean') alert.desktop = saved.desktop
  } catch {
    /* 本地值损坏：按默认处理，不影响使用 */
  }

  let client = null
  // 是否已经连上过一次：用来区分「首次连接」与「断线重连」（重连才需要补拉）
  let hasOpened = false
  // 未读数校准请求的序号：并发时只认最后一次的结果。
  // 否则「标已读后的新请求」可能被一个更早发出、更晚返回的旧结果覆盖，
  // 表现就是角标数字清掉了又自己冒回来。
  let refreshSeq = 0
  // 本地已经拿到更靠后的结论时（标已读返回的总数 / 本地先扣 / socket 累加），
  // 把序号往前推一格：还在飞的那次 refresh 回来就对不上号、结果作废，
  // 否则它会把刚清掉的数字按旧值又顶回去（观感就是「清完又冒出来」）。
  function bumpRefreshSeq() {
    refreshSeq += 1
  }
  // 当前正在查看的会话（对方 uid，0 = 没在聊任何会话）
  //
  // 用户正盯着这段对话时收到新消息，不该让角标先 +1 再等已读接口把它减回去：
  // 那一下跳动很像「数字清不掉」。页面进入/退出会话时通过 setViewing 同步这里。
  let viewingUid = 0
  // 是否正停留在群聊页面
  //
  // 在房间里被 @ 时页面自己会 toast，不必再弹右下角浮层（会重复提示）；
  // 但页面切到后台时仍要发系统通知，否则用户完全错过。
  let viewingRoom = false

  // 每个会话（键 = 对方 uid）已收到的最大 seq / 已上报 ACK 的最大 seq
  //
  // seq 就是消息 id（服务端自增主键），重连时把它当 after_seq 传给后端，
  // 就能只取「本地没有的那些」，不必整页重拉。
  const receivedSeq = new Map()
  const ackedSeq = new Map()
  const ackTimers = new Map()

  /** 订阅新私信，返回取消订阅函数 */
  function onMessage(cb) {
    listeners.add(cb)
    return () => listeners.delete(cb)
  }

  /**
   * 订阅会话事件，返回取消订阅函数
   *
   * 事件形态：
   *   { type: 'read', byUid, lastMsgId }  —— 对方读到了 lastMsgId（含）之前的消息
   *   { type: 'revoke', message }         —— 某条消息被撤回
   */
  function onEvent(cb) {
    eventListeners.add(cb)
    return () => eventListeners.delete(cb)
  }

  function emitEvent(event) {
    eventListeners.forEach((cb) => {
      try {
        cb(event)
      } catch {
        /* 单个订阅者出错不影响其它订阅者 */
      }
    })
  }

  // ---------- 新消息提醒（提示音 / 右下角浮层） ----------

  /** 订阅新消息提醒，返回取消订阅函数 */
  function onNotice(cb) {
    noticeListeners.add(cb)
    return () => noticeListeners.delete(cb)
  }

  function emitNotice(item) {
    noticeListeners.forEach((cb) => {
      try {
        cb(item)
      } catch {
        /* 单个订阅者出错不影响其它订阅者 */
      }
    })
  }

  /**
   * 收到新私信时的提醒
   *
   * 分两条路：
   *   - 页面在前台：提示音 + 右下角浮层（浮层看得见，才有意义）；
   *   - 页面在后台：页内浮层看不见，改走系统级桌面通知（Notification API），
   *     由订阅方（components/ChatNotice.vue）负责申请权限与绘制。
   * 注意后台不再播 Web Audio：系统通知自己带提示音，两处一起响会很吵。
   */
  function notifyIncoming(message, peerUid) {
    // 免打扰的会话只累计未读，不出声、不弹任何提示
    if (isPeerMuted(peerUid)) return

    const hidden = typeof document !== 'undefined' && document.visibilityState === 'hidden'

    if (hidden) {
      if (alert.desktop) emitNotice({ peerUid, message, background: true })
      return
    }

    if (alert.sound) playNoticeSound()
    if (alert.toast) {
      emitNotice({ peerUid, message, background: false })
      return
    }

    // 页内浮层关掉了、但桌面通知开着：前台也用系统通知顶上，
    // 否则用户除了响一声就再也看不到任何提示
    if (alert.desktop) emitNotice({ peerUid, message, background: true })
  }

  /**
   * 修改提醒偏好
   *
   * 写入本机（localStorage）并立即生效；打开提示音时试听一声，
   * 顺便解锁音频上下文（浏览器要求用户手势后才允许出声）。
   *
   * 注意 desktop 的授权不在这里做：requestPermission() 必须在用户点击的调用栈里，
   * 由 UI 层（composables/useDesktopNotice.js）先申请，拿到 granted 再调用本方法。
   */
  function setAlert(key, value) {
    if (key !== 'sound' && key !== 'toast' && key !== 'desktop') return

    const next = !!value
    if (alert[key] === next) return

    alert[key] = next

    try {
      localStorage.setItem(
        ALERT_KEY,
        JSON.stringify({ sound: alert.sound, toast: alert.toast, desktop: alert.desktop })
      )
    } catch {
      /* 隐私模式 / 存储写满：只在本次会话生效，不影响功能 */
    }

    if (key === 'sound' && next) playNoticeSound()
  }

  // ---------- 免打扰 / 在线成员 ----------

  /** 用会话列表刷新「哪些会话被免打扰」（列表加载 / 设置变更后调用） */
  function setMutedPeers(uids) {
    const next = new Set()
    ;(uids || []).forEach((uid) => {
      const value = Number(uid) || 0
      if (value > 0) next.add(value)
    })
    mutedPeers.value = next
  }

  function isPeerMuted(uid) {
    return mutedPeers.value.has(Number(uid) || 0)
  }

  /** 进房间时清掉 @ 角标 */
  function clearRoomMentions() {
    roomMentions.value = 0
  }

  // ---------- 正在输入 ----------

  // 每个会话上次上报的时间：同一个人连续打字时，3 秒内只发一次
  const typingSentAt = new Map()
  // 按房间分别节流：多群切换时不该被上一个群的 3 秒窗口挡住
  const roomTypingAt = new Map()

  /**
   * 上报「我正在给某人打字」
   *
   * 服务端只转发、不落库，丢了无所谓，所以连不上时直接忽略，
   * 不必退回 HTTP（多一次往返去告诉对方「我在打字」很不划算）。
   */
  function sendTyping(peerUid) {
    const key = Number(peerUid) || 0
    if (key <= 0 || !client) return

    const now = Date.now()
    if (now - (typingSentAt.get(key) || 0) < TYPING_INTERVAL) return
    typingSentAt.set(key, now)

    client.send({ type: 'typing', to: `user_${key}` })
  }

  /**
   * 上报「我正在某个群聊打字」
   *
   * 带 room_id：服务端据此只推给该群成员（公共大厅仍是全员广播），
   * 否则在 A 群打字会让 B 群的人看到「有人在输入」。
   */
  function sendRoomTyping(roomId = 1) {
    if (!client) return

    const id = Number(roomId) || 1
    const now = Date.now()
    if (now - (roomTypingAt.get(id) || 0) < TYPING_INTERVAL) return
    roomTypingAt.set(id, now)

    client.send({ type: 'typing', room: 1, room_id: id })
  }

  // ---------- 送达确认（ACK） ----------

  /** 记录某个会话收到的最大 seq（只前进不后退） */
  function noteSeq(peerUid, seq) {
    const key = Number(peerUid) || 0
    const value = Number(seq) || 0
    if (key <= 0 || value <= 0) return
    if (value > (receivedSeq.get(key) || 0)) receivedSeq.set(key, value)
  }

  /**
   * 上报某个会话的 ACK（送达确认，不等于已读）
   *
   * 优先走 socket（{type:'ack'}）：已经在长连接上了，省一次 HTTP 往返；
   * 连不上时退回 HTTP，保证服务端的 last_ack_seq 不会永久停在旧值。
   */
  function flushAck(peerUid) {
    const key = Number(peerUid) || 0
    const seq = receivedSeq.get(key) || 0
    if (key <= 0 || seq <= 0 || seq <= (ackedSeq.get(key) || 0)) return

    if (client?.send({ type: 'ack', data: { target_uid: key, seq } })) {
      ackedSeq.set(key, seq)
      return
    }

    ackChatMessages(key, seq)
      .then(() => ackedSeq.set(key, seq))
      .catch(() => {
        /* 离线 / 未登录：不回滚，下次有消息或重连时再补 */
      })
  }

  /** 延迟合并上报：连续收到多条消息只发一次最大的 seq */
  function scheduleAck(peerUid) {
    if (ackTimers.has(peerUid)) return
    ackTimers.set(
      peerUid,
      setTimeout(() => {
        ackTimers.delete(peerUid)
        flushAck(peerUid)
      }, ACK_DELAY)
    )
  }

  /**
   * 手动确认到某个 seq（聊天页加载完一页历史后调用）
   *
   * 历史里的消息同样是「已送达」，但也可能早就 ack 过，flushAck 会自己跳过。
   */
  function ack(peerUid, seq) {
    noteSeq(peerUid, seq)
    flushAck(peerUid)
  }

  /** 某个会话本地已知的最大 seq（补拉时的 after_seq） */
  function lastSeq(peerUid) {
    return receivedSeq.get(Number(peerUid) || 0) || 0
  }

  /** 重连后把所有还没确认的水位补报一遍 */
  function flushPendingAcks() {
    receivedSeq.forEach((_seq, key) => flushAck(key))
  }

  /** 拉取未读总数（登录态变化、进页面、读完消息后调用） */
  async function refresh() {
    const userStore = useUserStore()
    if (!userStore.isLogged) {
      unread.value = 0
      return
    }

    const seq = ++refreshSeq

    try {
      const res = await getChatUnread()
      // 已有更新的一次请求在飞：这次结果作废（否则会把刚扣掉的数字顶回来）
      if (seq !== refreshSeq) return
      if (res?.code === 200) unread.value = Number(res.data?.count || 0)
    } catch {
      /* 网络抖动：保留当前值，下次再校准 */
    }
  }

  /**
   * 本地先扣掉一部分未读
   *
   * 进入会话 / 标已读成功后立刻调用：用户已经看到这段对话了，
   * 数字不该再等「标已读 + 重新拉总数」两个往返才消失。
   * 请求失败时下一次 refresh() 会用服务端真实值把它加回来。
   */
  function consumeUnread(count) {
    const value = Number(count) || 0
    if (value <= 0) return
    bumpRefreshSeq()
    unread.value = Math.max(0, unread.value - value)
  }

  /**
   * 用服务端返回的未读总数直接覆盖角标
   *
   * 与 consumeUnread（本地先扣）互补：这里是「以服务端为准」的校准。
   * 标已读接口会顺带把最新的未读总数带回来，一次请求就能摆正，
   * 不用再发 GET /chat/unread —— 少一次往返，也少一次被并发/缓存写回旧值的机会。
   */
  function setUnread(count) {
    const value = Number(count)
    if (!Number.isFinite(value) || value < 0) return
    // 这是服务端在「刚标完已读」之后给的数，比任何更早发出的 refresh 都新
    bumpRefreshSeq()
    unread.value = value
  }

  /** 告知当前正在查看的会话（0 = 不在任何会话里），这段对话的新消息不计未读 */
  function setViewing(peerUid) {
    viewingUid = Number(peerUid) || 0
  }

  /** 告知是否停留在群聊（房间里被 @ 由页面自己提示，避免重复弹窗） */
  function setViewingRoom(value) {
    viewingRoom = !!value
  }

  function handleSocketMessage(payload) {
    if (!payload) return

    // 在线成员快照：服务端节流后广播给所有人（含访客）
    if (payload.type === 'status') {
      const content = payload.content || {}
      const users = Array.isArray(content.online_users) ? content.online_users : []
      const seen = new Map()

      users.forEach((item) => {
        const uid = Number(item?.uid || 0)
        // uid 为 0 是未登录访客；同一个人的多台设备只保留一条
        if (uid <= 0 || seen.has(uid)) return
        seen.set(uid, {
          uid,
          name: item.name || `用户 #${uid}`,
          lastActive: Number(item.last_active) || 0
        })
      })

      presence.value = {
        count: Number(content.online_count) || users.length,
        users: [...seen.values()]
      }
      return
    }

    // 对方正在输入：纯瞬时状态，由页面自己定时清除（这里不维护超时）
    if (payload.type === 'chat_typing') {
      const fromUid = Number(payload.data?.from_uid || 0)
      if (fromUid > 0) emitEvent({ type: 'typing', fromUid })
      return
    }

    // 群聊里有人 @ 我：累计角标 + 走与私信同一套提醒通道
    // （页面在后台时由 ChatNotice 转成系统通知，前台走右下角浮层）
    if (payload.type === 'chat_mention') {
      const item = payload.data || {}
      roomMentions.value += 1

      const uid = Number(item.from_uid || 0)
      if (uid > 0) {
        const hidden = typeof document !== 'undefined' && document.visibilityState === 'hidden'
        // roomId：多群之后 @ 提醒必须能跳回「被 @ 的那个群」，不能一律进大厅
        const notice = {
          kind: 'room',
          peerUid: uid,
          roomId: Number(item.room_id || 1) || 1,
          preview: String(item.preview || '')
        }

        if (hidden) {
          // 页面不可见（包括停在房间里但切走了标签页）：走系统通知
          if (alert.desktop) emitNotice({ ...notice, background: true })
        } else if (!viewingRoom) {
          if (alert.sound) playNoticeSound()
          if (alert.toast) emitNotice({ ...notice, background: false })
          else if (alert.desktop) emitNotice({ ...notice, background: true })
        }
      }

      emitEvent({ type: 'mention', item })
      return
    }

    // 公共群聊：服务端全员广播的消息与各类事件（撤回 / 禁言 / 解禁 / 改资料）。
    // 它不参与私信的未读与提醒逻辑，原样转给订阅者（房间页）自己处理。
    if (String(payload.type || '').startsWith('room')) {
      emitEvent({ type: 'room', event: payload })
      return
    }

    // 已读回执：对方读到了哪一条（只影响「我发出的消息」的已读标记）
    if (payload.type === 'chat_read') {
      const data = payload.data || {}
      emitEvent({
        type: 'read',
        byUid: Number(data.by_uid || 0),
        lastMsgId: Number(data.last_msg_id || 0)
      })
      return
    }

    // 撤回：把对应气泡替换成「撤回了一条消息」
    if (payload.type === 'chat_revoke') {
      const revoked = payload.data
      if (revoked && revoked.id) emitEvent({ type: 'revoke', message: revoked })
      return
    }

    // 删除：只删自己这一侧（服务端推给同一账号的其它在线设备），直接把气泡移除
    if (payload.type === 'chat_delete') {
      const id = Number(payload.data?.id || 0)
      if (id > 0) emitEvent({ type: 'delete', id })
      return
    }

    if (payload.type !== 'private') return

    // 服务端推的是 { type, data }；兼容早期把消息体平铺在 content 上的写法
    const raw = payload.data || payload.content
    const message = raw && typeof raw === 'object' ? raw : null
    if (!message || !message.id) return

    incoming.value = message

    const me = Number(useUserStore().user?.id || 0)
    // seq 与 id 同值（服务端自增主键）；旧服务端没下发 seq 时退回用 id
    const seq = Number(message.seq || message.id) || 0
    const peer = Number(message.from_uid) === me ? Number(message.to_uid) : Number(message.from_uid)
    noteSeq(peer, seq)

    if (me && Number(message.to_uid) === me) {
      // 正在看的会话不计未读、也不提醒：页面会立刻把它标成已读
      if (peer !== viewingUid) {
        // 这条消息是刚到的、更新的事实：早前发出的 refresh 未必算上它，
        // 回来时别把这次 +1 抹掉
        bumpRefreshSeq()
        unread.value += 1
        notifyIncoming(message, peer)
      }
      // 送达确认：合并上报（收到不等于读过，已读另走 readChatMessages）
      scheduleAck(peer)
    }

    listeners.forEach((cb) => {
      try {
        cb(message)
      } catch {
        /* 单个订阅者出错不影响其它订阅者 */
      }
    })
  }

  /** 建立连接（幂等：已连接直接返回） */
  function start() {
    const userStore = useUserStore()
    if (!userStore.isLogged) return
    if (client) return

    client = createSocketClient({
      onState: (state) => {
        const opened = state === 'open'
        connected.value = opened
        if (!opened) return
        // 重连：把离线期间没能上报的 ACK 先补一遍；
        // 再通知订阅方（聊天页）按本地最大 seq 做一次增量补拉，补回断线期间漏掉的消息
        flushPendingAcks()
        if (hasOpened) emitEvent({ type: 'reconnect' })
        hasOpened = true
      },
      onMessage: handleSocketMessage
    })
    client.connect()
    refresh()
  }

  /**
   * 立刻重连（页面发现长时间没有连接时调用）
   *
   * utils/socket.js 自己也会在失败后低速重试、在回到前台时唤醒，
   * 这里是给「用户明确感知到没连上」的场景一个即时入口，不必等下一轮。
   */
  function reconnect() {
    const userStore = useUserStore()
    if (!userStore.isLogged) return
    if (!client) {
      start()
      return
    }
    // 被管理员踢下线是明确的处置结果，不能被自动重连悄悄撤销
    if (client.state === 'kicked') return
    client.reconnect()
  }

  /** 断开连接并清空未读（退出登录时调用） */
  function stop() {
    if (client) {
      client.close()
      client = null
    }
    ackTimers.forEach((timer) => clearTimeout(timer))
    ackTimers.clear()
    // 水位属于「当前登录用户」，换人登录必须清掉，否则会把上一个人的 seq 当成本地进度
    receivedSeq.clear()
    ackedSeq.clear()
    typingSentAt.clear()
    roomTypingAt.clear()
    mutedPeers.value = new Set()
    presence.value = { count: 0, users: [] }
    roomMentions.value = 0
    hasOpened = false
    viewingUid = 0
    viewingRoom = false
    connected.value = false
    unread.value = 0
    incoming.value = null
  }

  return {
    unread,
    connected,
    incoming,
    onMessage,
    onEvent,
    refresh,
    start,
    stop,
    // 立即重连（长时间未连上时页面主动调用）
    reconnect,
    // 送达确认（聊天页加载完历史 / 收到消息后调用）
    ack,
    // 本地已知的最大 seq：重连补拉的 after_seq
    lastSeq,
    // 本地扣减未读（标已读后立刻见效，不必等服务端往返）
    consumeUnread,
    // 用服务端返回的总数覆盖（标已读接口顺带返回）
    setUnread,
    // 正在查看的会话：这段对话的新消息不计入角标
    setViewing,
    // 是否停留在群聊（房间里不重复弹 @ 通知）
    setViewingRoom,
    // 消息提醒偏好（提示音 / 悬浮通知）与订阅
    alert,
    setAlert,
    onNotice,
    // 在线成员快照（群聊的在线人数 / 成员列表 / @ 选人）
    presence,
    // 群聊 @ 角标
    roomMentions,
    clearRoomMentions,
    // 免打扰的会话
    setMutedPeers,
    isPeerMuted,
    // 正在输入（私聊 / 群聊）
    sendTyping,
    sendRoomTyping
  }
})
