/**
 * 系统级桌面通知（Notification API）
 *
 * 与页内右下角的浮层不是一回事：
 *   - 浮层只能在自己这个标签页可见时看到；
 *   - 系统通知由操作系统绘制，页面切到后台 / 窗口最小化时也能弹出来。
 *
 * 使用前必须拿到用户授权（permission），而且 requestPermission() 只能在**用户手势**
 * （点击、触摸）里调用，否则 Chrome 会直接按「被阻止」处理且不再询问。
 */
/** 当前环境能不能用：非 HTTPS（含 localhost 之外的 http）浏览器不会暴露 Notification */
export function desktopNoticeSupported() {
  if (typeof window === 'undefined') return false
  if (!('Notification' in window)) return false
  return window.isSecureContext !== false
}

/** 当前授权状态：granted / denied / default / unsupported */
export function desktopNoticePermission() {
  if (!desktopNoticeSupported()) return 'unsupported'
  return Notification.permission
}

/**
 * 申请通知权限（必须在用户手势里调用）
 * @returns {Promise<'granted'|'denied'|'default'|'unsupported'>}
 */
export async function requestDesktopNoticePermission() {
  if (!desktopNoticeSupported()) return 'unsupported'

  // 已经有结果就不用再问：重复调用浏览器不会弹窗，只会立刻返回旧状态
  if (Notification.permission === 'granted') return 'granted'
  if (Notification.permission === 'denied') return 'denied'

  try {
    return await Notification.requestPermission()
  } catch {
    return 'denied'
  }
}

// 保活：Chrome 里没有被引用的 Notification 可能被 GC 掉，onclick 就不会触发了
const alive = new Map()

/**
 * 弹一条系统通知
 *
 * @param {object} options
 * @param {string} options.title  标题（私信里用「私信 · 昵称」）
 * @param {string} options.body   正文（消息摘要）
 * @param {string} [options.icon] 图标（绝对地址，头像）
 * @param {string} [options.tag]  同 tag 的通知会替换旧的，不会堆一屏
 * @param {boolean} [options.silent] 是否静音（跟随用户的提示音开关）
 * @param {Function} [options.onClick] 点击后的回调（聚焦窗口 + 跳转由调用方决定）
 */
export function showDesktopNotice({ title, body, icon, tag, silent = false, onClick } = {}) {
  // 权限被用户在浏览器设置里撤销后，这里直接跳过，不要抛错打断消息处理
  if (desktopNoticePermission() !== 'granted') return null
  if (!title && !body) return null

  try {
    // 同一个会话：先把上一条关掉，避免系统通知中心堆一列同名通知
    if (tag && alive.has(tag)) {
      alive.get(tag).close()
      alive.delete(tag)
    }

    const notice = new Notification(title || '新消息', {
      body: body || '',
      icon,
      badge: icon,
      tag,
      silent,
      // 同 tag 替换时仍然提示一次（Chrome 支持，其它浏览器忽略）
      renotify: Boolean(tag)
    })

    notice.onclick = () => {
      try {
        window.focus()
        notice.close()
      } catch {
        /* 关闭失败无所谓 */
      }
      if (tag) alive.delete(tag)
      if (typeof onClick === 'function') onClick()
    }
    notice.onclose = () => {
      if (tag) alive.delete(tag)
    }
    notice.onerror = () => {
      if (tag) alive.delete(tag)
    }

    if (tag) alive.set(tag, notice)
    return notice
  } catch {
    // 移动端 Chrome 不允许直接 new Notification（要求 ServiceWorker），
    // 这里静默失败：手机上前台已有页内浮层兜底
    return null
  }
}
