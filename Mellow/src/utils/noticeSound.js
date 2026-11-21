/**
 * 新消息提示音
 *
 * 为什么用 Web Audio 合成而不是放一个 mp3：
 *   - 不占体积、不用额外静态资源，也不会因为 CDN / 部署路径不同导致 404；
 *   - 音色可调（这里是一声短促的「叮—咚」），和站点的克制风格更搭。
 *
 * 浏览器的自动播放策略：AudioContext 默认是 suspended，
 * 必须在**用户手势**里 resume 过一次才能出声（见 unlockNoticeSound）。
 * 调用方在 App 里挂一次 pointerdown 解锁；点开开关时也会主动解锁 + 试听。
 */

let context = null

function getContext() {
  if (typeof window === 'undefined') return null
  const AudioCtx = window.AudioContext || window.webkitAudioContext
  if (!AudioCtx) return null
  if (!context) {
    try {
      context = new AudioCtx()
    } catch {
      return null
    }
  }
  return context
}

/** 解锁音频（必须在用户手势回调里调用，例如 pointerdown / click） */
export function unlockNoticeSound() {
  const ctx = getContext()
  if (!ctx) return
  if (ctx.state === 'suspended') ctx.resume().catch(() => {})
}

/**
 * 播放提示音
 *
 * 两个音符（E6 → A6），总长约 0.26s：
 * 单音容易被忽略，两个上行音更容易被识别成「有新消息」，
 * 又不像长铃声那样打断正在看的页面。
 */
export function playNoticeSound() {
  const ctx = getContext()
  if (!ctx) return
  if (ctx.state === 'suspended') ctx.resume().catch(() => {})

  const now = ctx.currentTime
  const notes = [
    { freq: 1318.51, start: 0, duration: 0.12 },
    { freq: 1760.0, start: 0.1, duration: 0.16 }
  ]

  notes.forEach(({ freq, start, duration }) => {
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'sine'
    osc.frequency.setValueAtTime(freq, now + start)

    // 用指数衰减收尾，避免「咔」的爆音
    gain.gain.setValueAtTime(0.0001, now + start)
    gain.gain.exponentialRampToValueAtTime(0.18, now + start + 0.012)
    gain.gain.exponentialRampToValueAtTime(0.0001, now + start + duration)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start(now + start)
    osc.stop(now + start + duration + 0.02)
  })
}
