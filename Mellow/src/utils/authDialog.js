/**
 * 登录状态异常确认弹窗（原生 DOM 实现，与 toast.js 同一风格，避免引入 UI 库）
 *
 * 用于 401（token 失效/签名异常）场景：自动清理本地登录态后，
 * 弹窗提示用户，点击按钮即可彻底清除 cookie 与本地存储并跳转登录页。
 */

// ==================== 本地数据彻底清除 ====================

/**
 * 清除当前域下的全部 cookie。
 * 按 名称 × 路径 × 域 组合逐一过期删除，覆盖以不同 path/domain 写入的 cookie，
 * 避免单一路径删除导致"清了但没清掉"。
 * 注意：前端 JS 只能操作当前域的 cookie；后端域的 cookie 由后端在 401 响应中自行过期。
 */
export function clearAllCookies() {
  const items = document.cookie ? document.cookie.split(';') : []
  const hostname = window.location.hostname
  const paths = ['/', window.location.pathname || '/']
  // 无 domain 属性（当前主机）+ 点号前缀域（覆盖子域共享写入的场景）
  const domains = ['', `; domain=${hostname}`, `; domain=.${hostname}`]

  for (const item of items) {
    const name = item.split('=')[0].trim()
    if (!name) continue
    for (const path of paths) {
      for (const domain of domains) {
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=${path}${domain}`
      }
    }
  }
}

/**
 * 彻底清除本地登录数据：全部 cookie + localStorage + sessionStorage
 */
export function clearLocalAuthData() {
  try {
    clearAllCookies()
  } catch (e) { /* 忽略 */ }
  try {
    window.localStorage?.clear()
  } catch (e) { /* 忽略 */ }
  try {
    window.sessionStorage?.clear()
  } catch (e) { /* 忽略 */ }
}

// 单例：并发 401 时只弹一次
let activeMask = null

// 注入一次性动画
if (typeof document !== 'undefined' && !document.getElementById('auth-dialog-style')) {
  const style = document.createElement('style')
  style.id = 'auth-dialog-style'
  style.textContent = `
    @keyframes authDialogIn { from { opacity: 0; } to { opacity: 1; } }
    @keyframes authDialogBoxIn { from { opacity: 0; transform: translateY(-12px) scale(0.97); } to { opacity: 1; transform: translateY(0) scale(1); } }
  `
  document.head.appendChild(style)
}

function closeDialog() {
  if (!activeMask) return
  const mask = activeMask
  activeMask = null
  mask.style.transition = 'opacity 0.2s'
  mask.style.opacity = '0'
  setTimeout(() => mask.remove(), 200)
}

/**
 * 显示登录异常确认弹窗
 * @param {string} message 后端返回的提示信息
 * @param {Function} onConfirm 点击「清除并重新登录」后的回调（清理 + 跳转）
 */
export function showAuthDialog(message, onConfirm) {
  if (typeof document === 'undefined' || activeMask) return

  const mask = document.createElement('div')
  mask.style.cssText = `
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    z-index: 100000;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: authDialogIn 0.2s ease-out;
  `

  const box = document.createElement('div')
  box.style.cssText = `
    background: var(--bg-card, #fff);
    border: 1px solid var(--border, #e5e7eb);
    border-radius: var(--radius-xl, 18px);
    box-shadow: var(--shadow, 0 8px 30px rgba(0, 0, 0, 0.15));
    width: calc(100% - 48px);
    max-width: 380px;
    padding: 24px;
    text-align: center;
    animation: authDialogBoxIn 0.25s ease-out;
  `

  const title = document.createElement('div')
  title.style.cssText = `
    font-size: 16px;
    font-weight: 600;
    color: var(--text, #1f2329);
    margin-bottom: 10px;
  `
  title.textContent = '登录状态异常'

  const desc = document.createElement('div')
  desc.style.cssText = `
    font-size: 14px;
    color: var(--text-secondary, #646a73);
    line-height: 1.7;
    margin-bottom: 20px;
    word-break: break-all;
  `
  // textContent 赋值，防止后端 msg 注入 XSS
  desc.textContent = `${message || '登录状态异常，请清除cookie后重新登录！'}（需要您自行手动清除）`

  const btnRow = document.createElement('div')
  btnRow.style.cssText = 'display: flex; gap: 10px; justify-content: center;'

  const btnBase = `
    padding: 9px 20px;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    border: 1px solid transparent;
    transition: opacity 0.15s;
  `

  const confirmBtn = document.createElement('button')
  confirmBtn.style.cssText = btnBase + `
    background: var(--primary, #3b82f6);
    color: #fff;
  `
  confirmBtn.textContent = '重新登录'
  confirmBtn.onmouseenter = () => (confirmBtn.style.opacity = '0.85')
  confirmBtn.onmouseleave = () => (confirmBtn.style.opacity = '1')
  confirmBtn.onclick = () => {
    closeDialog()
    if (typeof onConfirm === 'function') onConfirm()
  }

  const cancelBtn = document.createElement('button')
  cancelBtn.style.cssText = btnBase + `
    background: transparent;
    color: var(--text-secondary, #646a73);
    border-color: var(--border, #e5e7eb);
  `
  cancelBtn.textContent = '稍后再说'
  cancelBtn.onclick = closeDialog

  btnRow.appendChild(confirmBtn)
  btnRow.appendChild(cancelBtn)

  box.appendChild(title)
  box.appendChild(desc)
  box.appendChild(btnRow)
  mask.appendChild(box)

  // 点击遮罩关闭（不执行清理跳转）
  mask.onclick = (e) => {
    if (e.target === mask) closeDialog()
  }

  document.body.appendChild(mask)
  activeMask = mask
}

export default { showAuthDialog }
