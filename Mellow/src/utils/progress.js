/**
 * 轻量路由顶部进度条（无第三方依赖）
 * 使用 transform 动画，不引起重排
 */
let bar = null
let inner = null
let timer = null
let pct = 0
let visible = false

function ensureBar() {
  if (bar) return
  bar = document.createElement('div')
  bar.style.cssText = `
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 2px;
    z-index: 10001;
    pointer-events: none;
    opacity: 0;
    transition: opacity 0.25s ease;
  `
  inner = document.createElement('div')
  inner.style.cssText = `
    height: 100%;
    width: 100%;
    transform-origin: 0 50%;
    transform: scaleX(0);
    background: var(--primary);
    box-shadow: 0 0 6px var(--accent-glow);
    transition: transform 0.2s ease-out;
  `
  bar.appendChild(inner)
  document.body.appendChild(bar)
}

function setScale(v) {
  if (inner) inner.style.transform = `scaleX(${v})`
}

/** 开始：显示进度条并渐进爬升到约 90% */
export function progressStart() {
  ensureBar()
  visible = true
  pct = 0.15
  bar.style.opacity = '1'
  setScale(pct)
  clearInterval(timer)
  timer = setInterval(() => {
    pct += (0.9 - pct) * 0.15
    setScale(pct)
  }, 300)
}

/** 完成：推进到 100% 后淡出 */
export function progressDone() {
  if (!bar || !visible) return
  clearInterval(timer)
  setScale(1)
  setTimeout(() => {
    bar.style.opacity = '0'
    setTimeout(() => {
      setScale(0)
      visible = false
    }, 300)
  }, 180)
}
