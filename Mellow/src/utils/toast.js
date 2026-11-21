/**
 * 简易 toast 工具（避免引入额外 UI 库）
 */
let container = null

function getContainer() {
  if (container) return container
  container = document.createElement('div')
  container.id = 'toast-container'
  container.style.cssText = `
    position: fixed;
    top: 24px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 99999;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
  `
  document.body.appendChild(container)
  return container
}

// 避免批量请求失败时刷屏
const MAX_TOASTS = 5
// 相同内容短时间内的重复提示（如并发请求同时报错）只展示一次
let lastMessage = ''
let lastTime = 0

function showToast(message, type = 'info', duration = 2400) {
  if (!duration || duration <= 0) duration = 2400
  const c = getContainer()
  const now = Date.now()
  if (message === lastMessage && now - lastTime < 800) return
  lastMessage = message
  lastTime = now

  const el = document.createElement('div')
  // 无障碍：错误/警告用 alert 实时播报，其它用 status
  if (type === 'error' || type === 'warning') {
    el.setAttribute('role', 'alert')
    el.setAttribute('aria-live', 'assertive')
  } else {
    el.setAttribute('role', 'status')
    el.setAttribute('aria-live', 'polite')
  }
  // 使用 CSS 变量，自动适配明暗主题
  const colors = {
    info: { bg: 'var(--bg-card)', border: 'var(--border)', color: 'var(--text)' },
    success: { bg: 'var(--bg-card)', border: 'var(--success)', color: 'var(--success)' },
    error: { bg: 'var(--bg-card)', border: 'var(--danger)', color: 'var(--danger)' },
    warning: { bg: 'var(--bg-card)', border: 'var(--warning)', color: 'var(--warning)' }
  }
  const c1 = colors[type] || colors.info
  el.style.cssText = `
    padding: 10px 18px;
    background: ${c1.bg};
    border: 1px solid ${c1.border};
    border-radius: 8px;
    color: ${c1.color};
    box-shadow: var(--shadow);
    font-size: 14px;
    max-width: 360px;
    pointer-events: auto;
    animation: toastIn 0.25s ease-out;
  `
  el.textContent = message
  c.appendChild(el)
  // 超出上限时移除最早的提示
  while (c.childElementCount > MAX_TOASTS) {
    c.firstElementChild?.remove()
  }
  setTimeout(() => {
    el.style.transition = 'opacity 0.3s, transform 0.3s'
    el.style.opacity = '0'
    el.style.transform = 'translateY(-6px)'
    setTimeout(() => el.remove(), 300)
  }, duration)
}

// 注入一次性动画
if (typeof document !== 'undefined' && !document.getElementById('toast-style')) {
  const style = document.createElement('style')
  style.id = 'toast-style'
  style.textContent = `@keyframes toastIn { from { opacity: 0; transform: translateY(-8px); } to { opacity: 1; transform: translateY(0); } }`
  document.head.appendChild(style)
}

// duration 可选：卡密这类需要用户抄下来的长内容可以传更长的展示时间
export const toast = {
  info: (m, duration) => showToast(m, 'info', duration),
  success: (m, duration) => showToast(m, 'success', duration),
  error: (m, duration) => showToast(m, 'error', duration),
  warning: (m, duration) => showToast(m, 'warning', duration)
}