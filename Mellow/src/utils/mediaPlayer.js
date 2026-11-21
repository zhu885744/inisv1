/**
 * 正文音频播放器：自定义元素 <mellow-audio src="..." caption="...">
 *
 * 为什么用 Web Component 而不是「渲染后再用 JS 增强」：
 *   正文是 v-html 注入的 HTML 字符串（文章 / 独立页 / 文档 / 编辑器预览共用 renderMarkdown），
 *   没有 Vue 组件可以挂。自定义元素只要注册一次，任何 v-html 插入的同名标签都会被浏览器
 *   自动升级成播放器 —— 四个渲染位置零改动，也不需要 MutationObserver 去追 DOM 变化。
 *
 * 视觉：卡片 + 渐变圆形播放键 + 细进度条 + 倍速，颜色全部走站点 CSS 变量
 * （shadow DOM 会继承 :root 上的自定义属性），亮色 / 暗色主题自动跟随。
 *
 * 原生 controls 是浏览器自绘的，CSS 只能改尺寸，做不出这套外观，所以这里完全自绘：
 * audio 元素不渲染 controls，只作为解码器放在 shadow 里。
 */

const SVG_PLAY =
  '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M8 5.14v13.72c0 .8.87 1.3 1.56.9l10.7-6.86a1.05 1.05 0 0 0 0-1.8L9.56 4.24A1.05 1.05 0 0 0 8 5.14z"/></svg>'

const SVG_PAUSE =
  '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><rect x="6" y="4" width="4" height="16" rx="1.2"/><rect x="14" y="4" width="4" height="16" rx="1.2"/></svg>'

const SPEEDS = [1, 1.25, 1.5, 2, 0.75]

/** 秒 → 0:00 / 1:02:03 */
function formatTime(value) {
  const total = Math.max(0, Math.floor(Number(value) || 0))
  const hour = Math.floor(total / 3600)
  const minute = Math.floor((total % 3600) / 60)
  const second = total % 60
  const mm = hour > 0 ? String(minute).padStart(2, '0') : String(minute)
  const ss = String(second).padStart(2, '0')
  return hour > 0 ? `${hour}:${mm}:${ss}` : `${mm}:${ss}`
}

const STYLE = `
  :host { display: block; margin: 14px 0; }
  *, *::before, *::after { box-sizing: border-box; }

  .player {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px 10px 10px;
    border: 1px solid var(--border-soft, #e6e4dc);
    border-radius: 14px;
    background: var(--bg-card, #fff);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }
  .player:hover {
    border-color: var(--primary, #4f7cff);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.07);
  }

  /* 播放键：渐变圆形，播放中带呼吸光环 */
  .play {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    padding: 0;
    border: none;
    border-radius: 50%;
    color: #fff;
    background: linear-gradient(135deg, var(--primary, #4f7cff), var(--primary-deep, #3560e0));
    cursor: pointer;
    transition: transform 0.15s ease, box-shadow 0.25s ease;
  }
  .play:hover { transform: scale(1.06); }
  .play:active { transform: scale(0.97); }
  .play:focus-visible { outline: 2px solid var(--primary, #4f7cff); outline-offset: 3px; }
  .play svg { width: 16px; height: 16px; }
  .play[data-state='playing'] {
    animation: breathe 2.2s ease-in-out infinite;
  }

  .main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 5px; }

  .meta { display: flex; align-items: center; gap: 8px; }
  .caption {
    flex: 1;
    min-width: 0;
    font-size: 12px;
    line-height: 1.4;
    color: var(--text-muted, #6b6a64);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .speed {
    flex-shrink: 0;
    padding: 1px 7px;
    font-size: 10px;
    font-variant-numeric: tabular-nums;
    color: var(--text-light, #98968e);
    background: var(--bg-muted, #f4f2ec);
    border: 1px solid transparent;
    border-radius: 999px;
    cursor: pointer;
    transition: color 0.15s, border-color 0.15s, background 0.15s;
  }
  .speed:hover {
    color: var(--primary-deep, #3560e0);
    border-color: var(--primary, #4f7cff);
  }

  .line { display: flex; align-items: center; gap: 10px; }

  /* 进度条：底层是 4px 的细条，上层铺一条透明的 range 负责拖动与键盘操作 */
  .track { position: relative; flex: 1; min-width: 0; height: 14px; display: flex; align-items: center; }
  .bar {
    position: absolute;
    left: 0; right: 0;
    top: 50%;
    transform: translateY(-50%);
    height: 4px;
    border-radius: 999px;
    background: var(--bg-muted, #eceae3);
    overflow: hidden;
  }
  .fill {
    display: block;
    width: 0%;
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, var(--primary, #4f7cff), var(--primary-deep, #3560e0));
    transition: width 0.1s linear;
  }
  input[type='range'] {
    position: relative;
    z-index: 2;
    width: 100%;
    height: 14px;
    margin: 0;
    background: transparent;
    -webkit-appearance: none;
    appearance: none;
    cursor: pointer;
  }
  input[type='range']:focus-visible { outline: none; }
  input[type='range']::-webkit-slider-runnable-track { height: 4px; background: transparent; }
  input[type='range']::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 10px; height: 10px;
    margin-top: -3px;
    border: 2px solid var(--primary, #4f7cff);
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .player:hover input[type='range']::-webkit-slider-thumb,
  input[type='range']:focus-visible::-webkit-slider-thumb { opacity: 1; }
  input[type='range']::-moz-range-track { height: 4px; background: transparent; }
  input[type='range']::-moz-range-thumb {
    width: 8px; height: 8px;
    border: 2px solid var(--primary, #4f7cff);
    border-radius: 50%;
    background: #fff;
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .player:hover input[type='range']::-moz-range-thumb { opacity: 1; }

  .time {
    flex-shrink: 0;
    font-size: 11px;
    color: var(--text-light, #98968e);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.01em;
  }

  .player.is-error { border-color: var(--danger, #d9544d); }
  .player.is-error .time { color: var(--danger, #d9544d); }

  @keyframes breathe {
    0%, 100% { box-shadow: 0 0 0 0 rgba(79, 124, 255, 0.45); }
    50% { box-shadow: 0 0 0 7px rgba(79, 124, 255, 0); }
  }

  @media (prefers-reduced-motion: reduce) {
    .play, .fill, .player { transition: none; }
    .play[data-state='playing'] { animation: none; }
  }
`

// 同一页面同时只播一个（音视频共用）：点第二个时自动暂停上一个
const activeMedia = new Set()

function pauseOthers(current) {
  activeMedia.forEach((item) => {
    if (item !== current) item.pause()
  })
}

class MellowAudio extends HTMLElement {

  connectedCallback() {
    if (this._mounted) return
    this._mounted = true

    const src = this.getAttribute('src') || ''
    const caption = (this.getAttribute('caption') || '').trim()

    const root = this.attachShadow({ mode: 'open' })
    // 有标题才渲染标题行：没有标题时卡片收成单行（播放键 + 进度 + 时间 + 倍速），不浪费高度
    const metaHtml = caption
      ? `<div class="meta"><span class="caption">${escapeText(caption)}</span></div>`
      : ''

    root.innerHTML =
      `<style>${STYLE}</style>` +
      '<div class="player">' +
      `<button class="play" type="button" data-state="paused" aria-label="播放">${SVG_PLAY}</button>` +
      '<div class="main">' +
      metaHtml +
      '<div class="line">' +
      '<div class="track"><span class="bar"><span class="fill"></span></span>' +
      '<input type="range" min="0" max="1000" value="0" step="1" aria-label="播放进度"></div>' +
      '<span class="time">0:00</span>' +
      `<button class="speed" type="button" aria-label="切换播放速度">1x</button>` +
      '</div>' +
      '</div>' +
      '</div>' +
      `<audio preload="metadata" src="${escapeAttr(src)}"></audio>`

    this._audio = root.querySelector('audio')
    this._player = root.querySelector('.player')
    this._playBtn = root.querySelector('.play')
    this._fill = root.querySelector('.fill')
    this._range = root.querySelector("input[type='range']")
    this._time = root.querySelector('.time')
    this._speedBtn = root.querySelector('.speed')
    this._speedIndex = 0

    this._playBtn.addEventListener('click', () => this.toggle())
    this._speedBtn.addEventListener('click', () => this.cycleSpeed())

    this._range.addEventListener('input', () => {
      const duration = this._audio.duration
      if (!Number.isFinite(duration) || duration <= 0) return
      this._audio.currentTime = (Number(this._range.value) / 1000) * duration
      this.renderProgress()
    })

    const audio = this._audio
    audio.addEventListener('loadedmetadata', this.renderProgress)
    audio.addEventListener('timeupdate', this.renderProgress)
    audio.addEventListener('play', () => {
      // 独占播放：暂停其它实例（包括正文里的视频）
      pauseOthers(this)
      activeMedia.add(this)
      this.setAttribute('playing', '')
      this._playBtn.dataset.state = 'playing'
      this._playBtn.innerHTML = SVG_PAUSE
      this._playBtn.setAttribute('aria-label', '暂停')
    })
    audio.addEventListener('pause', () => {
      this.removeAttribute('playing')
      this._playBtn.dataset.state = 'paused'
      this._playBtn.innerHTML = SVG_PLAY
      this._playBtn.setAttribute('aria-label', '播放')
    })
    audio.addEventListener('ended', () => {
      activeMedia.delete(this)
      this.renderProgress()
    })
    audio.addEventListener('error', () => {
      this._player.classList.add('is-error')
      this._time.textContent = '加载失败'
      this._range.disabled = true
    })
  }

  disconnectedCallback() {
    // v-html 重新渲染会把旧元素整段移除：必须停掉，否则声音会在后台继续播
    this.pause()
    activeMedia.delete(this)
  }

  play() {
    const result = this._audio?.play()
    if (result?.catch) result.catch(() => {})
  }

  pause() {
    if (this._audio && !this._audio.paused) this._audio.pause()
  }

  toggle() {
    if (!this._audio) return
    if (this._audio.paused) this.play()
    else this.pause()
  }

  cycleSpeed() {
    if (!this._audio) return
    this._speedIndex = (this._speedIndex + 1) % SPEEDS.length
    const rate = SPEEDS[this._speedIndex]
    this._audio.playbackRate = rate
    this._speedBtn.textContent = `${rate}x`
  }

  renderProgress = () => {
    const audio = this._audio
    if (!audio) return

    const duration = Number.isFinite(audio.duration) ? audio.duration : 0
    const current = Number.isFinite(audio.currentTime) ? audio.currentTime : 0
    const percent = duration > 0 ? Math.min(100, (current / duration) * 100) : 0

    this._fill.style.width = `${percent}%`
    this._range.value = String(Math.round(percent * 10))
    this._time.textContent = duration > 0
      ? `${formatTime(current)} / ${formatTime(duration)}`
      : formatTime(current)
  }
}

function escapeText(value) {
  return String(value || '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function escapeAttr(value) {
  return escapeText(value).replace(/"/g, '&quot;')
}

/** 幂等注册（HMR / 多次 import 都安全） */
export function defineAudioPlayer() {
  if (typeof window === 'undefined' || !window.customElements) return
  if (window.customElements.get('mellow-audio')) return
  window.customElements.define('mellow-audio', MellowAudio)
}

/* ============================================================
 * 视频播放器：<mellow-video src="..." caption="...">
 *
 * 和音频同一套思路，但视频的交互重心在画面上：
 *   - 画面即卡片，未播放时中央一颗半透明大播放键；
 *   - 控制条压在画面底部（播放中 2.6 秒无操作自动淡出，鼠标一动就回来）；
 *   - 进度条 / 当前时长 / 倍速 / 静音 / 画中画 / 全屏。
 * 全屏与画中画都做了能力检测，不支持的浏览器直接不渲染对应按钮。
 * ============================================================ */

const SVG_VOLUME =
  '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M4.5 9h3l4-3.2v12.4l-4-3.2h-3z"/><path d="M15.2 8.4a5.2 5.2 0 0 1 0 7.2" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>'

const SVG_MUTED =
  '<svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><path d="M4.5 9h3l4-3.2v12.4l-4-3.2h-3z"/><path d="M15.5 9.5l5 5m0-5l-5 5" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>'

const SVG_PIP =
  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linejoin="round" aria-hidden="true"><rect x="2.5" y="4.5" width="19" height="15" rx="2.5"/><rect x="12" y="11.5" width="7.5" height="6" rx="1.2" fill="currentColor" stroke="none"/></svg>'

const SVG_FULLSCREEN =
  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 4H5.5A1.5 1.5 0 0 0 4 5.5V9M15 4h3.5A1.5 1.5 0 0 1 20 5.5V9M9 20H5.5A1.5 1.5 0 0 1 4 18.5V15M15 20h3.5a1.5 1.5 0 0 0 1.5-1.5V15"/></svg>'

const SVG_FULLSCREEN_EXIT =
  '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9.5 4.5V8A1.5 1.5 0 0 1 8 9.5H4.5M14.5 4.5V8A1.5 1.5 0 0 0 16 9.5h3.5M9.5 19.5V16A1.5 1.5 0 0 0 8 14.5H4.5M14.5 19.5V16a1.5 1.5 0 0 1 1.5-1.5h3.5"/></svg>'

const VIDEO_STYLE = `
  :host { display: flex; justify-content: center; margin: 14px 0; }
  *, *::before, *::after { box-sizing: border-box; }

  /* 卡片尺寸完全由视频自身比例决定：卡片 is width:fit-content，视频只给上限，
     浏览器按 intrinsic ratio 撑开 —— 竖屏视频不会再出现「画面一条、黑块一大片」。
     高度上限可用 --md-video-max-height 覆盖（编辑器预览列较窄时会调小）。 */
  .vplayer {
    position: relative;
    width: fit-content;
    max-width: 100%;
    max-height: var(--md-video-max-height, 70vh);
    margin: 0 auto;
    overflow: hidden;
    border-radius: 14px;
    background: #000;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.18);
  }

  .stage {
    position: relative;
    display: flex;
    max-width: 100%;
    max-height: var(--md-video-max-height, 70vh);
    background: #000;
  }
  /* 不写死宽高：元数据没加载完时也不会先撑出一块 16:9 的黑底 */
  video {
    display: block;
    width: auto;
    height: auto;
    max-width: 100%;
    max-height: var(--md-video-max-height, 70vh);
    object-fit: contain;
    background: #000;
  }

  /* 中央大播放键：暂停时出现，播放后淡出 */
  .big {
    position: absolute;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    display: flex;
    align-items: center;
    justify-content: center;
    width: 58px;
    height: 58px;
    padding: 0;
    border: none;
    border-radius: 50%;
    color: #fff;
    background: rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(6px);
    cursor: pointer;
    transition: transform 0.16s ease, opacity 0.2s ease, background 0.2s ease;
  }
  .big:hover { background: rgba(0, 0, 0, 0.62); transform: translate(-50%, -50%) scale(1.06); }
  .big svg { width: 22px; height: 22px; margin-left: 2px; }
  .is-playing .big { opacity: 0; pointer-events: none; }

  .topbar {
    position: absolute;
    left: 0;
    right: 0;
    top: 0;
    padding: 10px 14px 22px;
    font-size: 12px;
    color: #fff;
    background: linear-gradient(to bottom, rgba(0, 0, 0, 0.65), transparent);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: opacity 0.2s ease;
  }

  .controls {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 24px 8px 8px;
    background: linear-gradient(to top, rgba(0, 0, 0, 0.78), transparent);
    transition: opacity 0.2s ease;
  }

  /* 播放中闲置 → 控制条与光标一起隐藏 */
  .is-hide-ui .controls,
  .is-hide-ui .topbar { opacity: 0; pointer-events: none; }
  .is-hide-ui { cursor: none; }

  .ctrl {
    flex-shrink: 0;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    padding: 0;
    border: none;
    border-radius: 8px;
    color: #fff;
    background: transparent;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .ctrl:hover { background: rgba(255, 255, 255, 0.16); }
  .ctrl:focus-visible { outline: 2px solid #fff; outline-offset: 1px; }
  .ctrl svg { width: 16px; height: 16px; }
  .speed { width: auto; padding: 0 6px; font-size: 11px; font-variant-numeric: tabular-nums; }

  .track { position: relative; flex: 1; min-width: 36px; height: 16px; display: flex; align-items: center; }
  .bar {
    position: absolute;
    left: 0; right: 0;
    top: 50%;
    transform: translateY(-50%);
    height: 4px;
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.28);
    overflow: hidden;
  }
  .fill { display: block; width: 0%; height: 100%; border-radius: 999px; background: #fff; }
  input[type='range'] {
    position: relative;
    z-index: 2;
    width: 100%;
    height: 16px;
    margin: 0;
    background: transparent;
    -webkit-appearance: none;
    appearance: none;
    cursor: pointer;
  }
  input[type='range']::-webkit-slider-runnable-track { height: 4px; background: transparent; }
  input[type='range']::-webkit-slider-thumb {
    -webkit-appearance: none;
    width: 10px; height: 10px;
    margin-top: -3px;
    border: none;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.45);
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .controls:hover input[type='range']::-webkit-slider-thumb,
  input[type='range']:focus-visible::-webkit-slider-thumb { opacity: 1; }
  input[type='range']::-moz-range-track { height: 4px; background: transparent; }
  input[type='range']::-moz-range-thumb {
    width: 8px; height: 8px;
    border: none;
    border-radius: 50%;
    background: #fff;
    opacity: 0;
    transition: opacity 0.15s ease;
  }
  .controls:hover input[type='range']::-moz-range-thumb { opacity: 1; }

  .time {
    flex-shrink: 0;
    font-size: 11px;
    color: rgba(255, 255, 255, 0.92);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.01em;
    padding: 0 2px;
  }

  .vplayer.is-error .stage::after {
    content: '视频加载失败';
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: #fff;
    background: rgba(0, 0, 0, 0.6);
  }

  /* 全屏时铺满屏幕（宿主自己进全屏，尺寸规则全部放开） */
  :host(:fullscreen) { display: block; margin: 0; }
  :host(:fullscreen) .vplayer {
    width: 100%;
    height: 100vh;
    max-width: none;
    max-height: none;
    margin: 0;
    border-radius: 0;
    box-shadow: none;
  }
  :host(:fullscreen) .stage {
    width: 100%;
    height: 100%;
    max-width: none;
    max-height: none;
  }
  :host(:fullscreen) video {
    width: 100%;
    height: 100%;
    max-width: none;
    max-height: none;
  }

  /* 竖屏视频会把卡片收窄，按钮太多会挤出边界，按容器宽度而不是视口宽度裁剪 */
  .vplayer.is-narrow .speed,
  .vplayer.is-narrow .pip { display: none; }
  .vplayer.is-narrow .controls { gap: 2px; padding: 22px 6px 6px; }
  .vplayer.is-narrow .ctrl { width: 28px; height: 28px; }

  @media (max-width: 520px) {
    .speed, .pip { display: none; }
    .controls { gap: 2px; padding: 22px 6px 6px; }
    .ctrl { width: 28px; height: 28px; }
  }

  @media (prefers-reduced-motion: reduce) {
    .big, .controls, .topbar { transition: none; }
  }
`

class MellowVideo extends HTMLElement {

  connectedCallback() {
    if (this._mounted) return
    this._mounted = true

    const src = this.getAttribute('src') || ''
    const caption = (this.getAttribute('caption') || '').trim()
    const canPip = typeof document !== 'undefined' && document.pictureInPictureEnabled

    const root = this.attachShadow({ mode: 'open' })
    root.innerHTML =
      `<style>${VIDEO_STYLE}</style>` +
      '<div class="vplayer">' +
      '<div class="stage">' +
      `<video playsinline preload="metadata" src="${escapeAttr(src)}"></video>` +
      `<button class="big" type="button" aria-label="播放">${SVG_PLAY}</button>` +
      (caption ? `<div class="topbar">${escapeText(caption)}</div>` : '') +
      '<div class="controls">' +
      `<button class="ctrl play" type="button" aria-label="播放">${SVG_PLAY}</button>` +
      '<div class="track"><span class="bar"><span class="fill"></span></span>' +
      '<input type="range" min="0" max="1000" value="0" step="1" aria-label="播放进度"></div>' +
      '<span class="time">0:00 / 0:00</span>' +
      '<button class="ctrl speed" type="button" aria-label="切换播放速度">1x</button>' +
      `<button class="ctrl mute" type="button" aria-label="静音">${SVG_VOLUME}</button>` +
      (canPip ? `<button class="ctrl pip" type="button" aria-label="画中画">${SVG_PIP}</button>` : '') +
      `<button class="ctrl full" type="button" aria-label="全屏">${SVG_FULLSCREEN}</button>` +
      '</div>' +
      '</div>' +
      '</div>'

    this._video = root.querySelector('video')
    this._player = root.querySelector('.vplayer')
    this._stage = root.querySelector('.stage')
    this._big = root.querySelector('.big')
    this._playBtn = root.querySelector('.ctrl.play')
    this._fill = root.querySelector('.fill')
    this._range = root.querySelector("input[type='range']")
    this._time = root.querySelector('.time')
    this._speedBtn = root.querySelector('.ctrl.speed')
    this._muteBtn = root.querySelector('.ctrl.mute')
    this._pipBtn = root.querySelector('.ctrl.pip')
    this._fullBtn = root.querySelector('.ctrl.full')
    this._speedIndex = 0
    this._hideTimer = null

    this._big.addEventListener('click', () => this.play())
    this._playBtn.addEventListener('click', () => this.toggle())
    // 单击播放/暂停、双击全屏：单击延迟 220ms 再执行，避免双击时先播又立刻暂停
    this._clickTimer = null
    this._video.addEventListener('click', () => {
      if (this._clickTimer) {
        clearTimeout(this._clickTimer)
        this._clickTimer = null
        return
      }
      this._clickTimer = setTimeout(() => {
        this._clickTimer = null
        this.toggle()
      }, 220)
    })
    this._video.addEventListener('dblclick', () => this.toggleFullscreen())
    this._speedBtn.addEventListener('click', () => this.cycleSpeed())
    this._muteBtn.addEventListener('click', () => this.toggleMute())
    this._fullBtn.addEventListener('click', () => this.toggleFullscreen())
    this._pipBtn?.addEventListener('click', () => this.togglePip())

    this._range.addEventListener('input', () => {
      const duration = this._video.duration
      if (!Number.isFinite(duration) || duration <= 0) return
      this._video.currentTime = (Number(this._range.value) / 1000) * duration
      this.renderProgress()
      this.showUi()
    })

    // 鼠标 / 触屏操作 → 控制条浮现，停手 2.6 秒后再淡出
    this._player.addEventListener('mousemove', () => this.showUi())
    this._player.addEventListener('touchstart', () => this.showUi(), { passive: true })
    this._player.addEventListener('mouseleave', () => {
      if (!this._video.paused) this.hideUi()
    })

    const video = this._video
    video.addEventListener('loadedmetadata', () => {
      this.applyLayout()
      this.renderProgress()
    })
    // 元数据可能已经加载完（元素被重新插入 DOM）时才挂监听，这里补一次
    if (video.readyState >= 1) this.applyLayout()

    // 卡片尺寸交给 CSS（fit-content + 视频比例），这里只按实际宽度决定要不要收起次要按钮；
    // 用 ResizeObserver 盯住画面本身：侧栏折叠、编辑器分屏、全屏切换都会触发
    this._ro = null
    if (typeof ResizeObserver !== 'undefined') {
      this._ro = new ResizeObserver(() => this.applyLayout())
      this._ro.observe(this._stage)
    }
    video.addEventListener('timeupdate', this.renderProgress)
    video.addEventListener('play', () => {
      pauseOthers(this)
      activeMedia.add(this)
      this._player.classList.add('is-playing')
      this._playBtn.innerHTML = SVG_PAUSE
      this._playBtn.setAttribute('aria-label', '暂停')
      this.scheduleHide()
    })
    video.addEventListener('pause', () => {
      this._player.classList.remove('is-playing')
      this._playBtn.innerHTML = SVG_PLAY
      this._playBtn.setAttribute('aria-label', '播放')
      this.showUi()
    })
    video.addEventListener('ended', () => {
      activeMedia.delete(this)
      this._player.classList.remove('is-playing')
      this.showUi()
      this.renderProgress()
    })
    video.addEventListener('volumechange', () => {
      const muted = video.muted || video.volume === 0
      this._muteBtn.innerHTML = muted ? SVG_MUTED : SVG_VOLUME
      this._muteBtn.setAttribute('aria-label', muted ? '取消静音' : '静音')
    })
    video.addEventListener('error', () => {
      this._player.classList.add('is-error')
      this._range.disabled = true
    })

    document.addEventListener('fullscreenchange', this.syncFullscreen)
  }

  disconnectedCallback() {
    this.pause()
    activeMedia.delete(this)
    if (this._hideTimer) clearTimeout(this._hideTimer)
    if (this._clickTimer) clearTimeout(this._clickTimer)
    this._ro?.disconnect()
    document.removeEventListener('fullscreenchange', this.syncFullscreen)
  }

  /**
   * 画面宽度实测：只用来决定次要按钮（倍速 / 画中画）要不要收起
   *
   * 卡片本身的尺寸由 CSS 决定（width: fit-content + 视频 max-width/max-height），
   * 浏览器按 intrinsic ratio 撑开，竖屏视频不会再留下「一边画面一边黑块」的空档。
   */
  applyLayout = () => {
    const stage = this._stage
    if (!stage) return
    this._player.classList.toggle('is-narrow', stage.clientWidth < 420)
  }

  play() {
    const result = this._video?.play()
    if (result?.catch) result.catch(() => {})
  }

  pause() {
    if (this._video && !this._video.paused) this._video.pause()
  }

  toggle() {
    if (!this._video) return
    if (this._video.paused) this.play()
    else this.pause()
  }

  cycleSpeed() {
    if (!this._video) return
    this._speedIndex = (this._speedIndex + 1) % SPEEDS.length
    const rate = SPEEDS[this._speedIndex]
    this._video.playbackRate = rate
    this._speedBtn.textContent = `${rate}x`
  }

  toggleMute() {
    if (!this._video) return
    this._video.muted = !this._video.muted
  }

  toggleFullscreen() {
    if (document.fullscreenElement) {
      // exitFullscreen 在部分浏览器返回 undefined，不能直接链 .catch
      try {
        document.exitFullscreen?.()
      } catch {}
      return
    }
    if (this.requestFullscreen) {
      const request = this.requestFullscreen()
      if (request?.catch) request.catch(() => this._video?.webkitEnterFullscreen?.())
      return
    }
    // iOS Safari 不支持元素全屏，只能走 video 自己的全屏
    this._video?.webkitEnterFullscreen?.()
  }

  async togglePip() {
    try {
      if (document.pictureInPictureElement) await document.exitPictureInPicture()
      else await this._video.requestPictureInPicture()
    } catch {
      // 浏览器不支持 / 用户拒绝：保持原样即可
    }
  }

  syncFullscreen = () => {
    const full = document.fullscreenElement === this
    this._fullBtn.innerHTML = full ? SVG_FULLSCREEN_EXIT : SVG_FULLSCREEN
    this._fullBtn.setAttribute('aria-label', full ? '退出全屏' : '全屏')
  }

  showUi() {
    this._player.classList.remove('is-hide-ui')
    this.scheduleHide()
  }

  hideUi() {
    this._player.classList.add('is-hide-ui')
  }

  scheduleHide() {
    if (this._hideTimer) clearTimeout(this._hideTimer)
    if (this._video?.paused) return
    this._hideTimer = setTimeout(() => {
      if (!this._video.paused) this.hideUi()
    }, 2600)
  }

  renderProgress = () => {
    const video = this._video
    if (!video) return

    const duration = Number.isFinite(video.duration) ? video.duration : 0
    const current = Number.isFinite(video.currentTime) ? video.currentTime : 0
    const percent = duration > 0 ? Math.min(100, (current / duration) * 100) : 0

    this._fill.style.width = `${percent}%`
    this._range.value = String(Math.round(percent * 10))
    this._time.textContent = `${formatTime(current)} / ${formatTime(duration)}`
  }
}

/** 幂等注册视频播放器 */
export function defineVideoPlayer() {
  if (typeof window === 'undefined' || !window.customElements) return
  if (window.customElements.get('mellow-video')) return
  window.customElements.define('mellow-video', MellowVideo)
}

/* ============================================================
 * 实况图：<mellow-live src="视频" image="封面图" caption="说明">
 *
 * 给文章正文用（动态那边是 Vue 组件 components/LivePhoto.vue，两者观感一致）：
 *   - 静态封面图铺满卡片，左下角一个 LIVE 角标；
 *   - 桌面悬停 / 触屏按住时叠一层静音循环视频，松手立刻暂停复位；
 *   - 视频编码不支持 / 文件 404 时把视频层摘掉，只留封面图，不弹错。
 * 短代码：@[live](视频URL "封面图URL" "说明")，见 utils/markdown.js。
 * ============================================================ */

const LIVE_STYLE = `
  :host { display: flex; justify-content: center; margin: 14px 0; }
  *, *::before, *::after { box-sizing: border-box; }

  .live {
    position: relative;
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    border-radius: 14px;
    background: #000;
    cursor: pointer;
    line-height: 0;
  }
  img.cover {
    display: block;
    max-width: 100%;
    max-height: var(--md-live-max-height, 70vh);
    object-fit: contain;
  }
  video.clip {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: 0;
    transition: opacity 0.18s ease;
    background: #000;
  }
  .live.is-playing video.clip { opacity: 1; }

  .badge {
    position: absolute;
    left: 8px;
    bottom: 8px;
    display: inline-flex;
    align-items: center;
    gap: 3px;
    padding: 2px 7px 2px 5px;
    border-radius: 999px;
    background: rgba(0, 0, 0, 0.5);
    color: #fff;
    font: 600 10px/1.6 system-ui, -apple-system, sans-serif;
    letter-spacing: 0.04em;
    backdrop-filter: blur(2px);
    pointer-events: none;
  }
  .badge i { font-style: normal; color: #ffd6a5; }

  .cap {
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    padding: 18px 12px 8px;
    font: 400 12px/1.5 system-ui, -apple-system, sans-serif;
    color: #fff;
    background: linear-gradient(to top, rgba(0, 0, 0, 0.65), transparent);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    pointer-events: none;
  }
`

class MellowLive extends HTMLElement {
  static get observedAttributes() {
    return ['src', 'image', 'caption']
  }

  constructor() {
    super()
    this._built = false
    this._broken = false
    this._root = this.attachShadow({ mode: 'open' })
    this.onEnter = () => {
      if (!this._hoverable) return
      this.play()
    }
    this.onLeave = () => {
      if (!this._hoverable) return
      this.stop()
    }
    this.onPressStart = () => {
      if (this._hoverable) return
      clearTimeout(this._pressTimer)
      this._pressTimer = setTimeout(() => this.play(), 120)
    }
    this.onPressEnd = () => {
      if (this._hoverable) return
      clearTimeout(this._pressTimer)
      this.stop()
    }
  }

  connectedCallback() {
    if (this._built) return this.sync()
    this._built = true
    this._hoverable =
      typeof window !== 'undefined' && window.matchMedia
        ? window.matchMedia('(hover: hover)').matches
        : true

    const style = document.createElement('style')
    style.textContent = LIVE_STYLE

    this._wrap = document.createElement('div')
    this._wrap.className = 'live'

    this._img = document.createElement('img')
    this._img.className = 'cover'
    this._img.loading = 'lazy'
    this._img.decoding = 'async'
    this._img.alt = ''

    this._video = document.createElement('video')
    this._video.className = 'clip'
    this._video.muted = true
    this._video.loop = true
    this._video.playsInline = true
    this._video.preload = 'metadata'
    this._video.addEventListener('error', () => this.fail())
    this._video.addEventListener('loadeddata', () => {
      if (!this._video.videoWidth) this.fail()
    })

    this._badge = document.createElement('span')
    this._badge.className = 'badge'

    this._cap = document.createElement('span')
    this._cap.className = 'cap'

    this._wrap.append(this._img, this._video, this._badge, this._cap)
    this._root.append(style, this._wrap)

    this._wrap.addEventListener('mouseenter', this.onEnter)
    this._wrap.addEventListener('mouseleave', this.onLeave)
    this._wrap.addEventListener('pointerdown', this.onPressStart)
    this._wrap.addEventListener('pointerup', this.onPressEnd)
    this._wrap.addEventListener('pointercancel', this.onPressEnd)
    this._wrap.addEventListener('pointerleave', this.onPressEnd)

    this.sync()
  }

  disconnectedCallback() {
    clearTimeout(this._pressTimer)
    this.stop()
  }

  attributeChangedCallback() {
    if (this._built) this.sync()
  }

  sync() {
    const video = this.getAttribute('src') || ''
    const image = this.getAttribute('image') || ''
    const caption = this.getAttribute('caption') || ''

    this._img.src = image
    this._cap.textContent = caption
    this._cap.style.display = caption ? '' : 'none'
    this._badge.innerHTML = '<i>●</i> LIVE'

    if (video && this._video.getAttribute('src') !== video) {
      this._broken = false
      this._video.src = video
    }
    this._video.style.display = video && !this._broken ? '' : 'none'
    this._badge.style.display = video && !this._broken ? '' : 'none'
  }

  async play() {
    if (this._broken || !this._video?.src) return
    try {
      this._video.currentTime = 0
      await this._video.play()
      this._wrap.classList.add('is-playing')
    } catch {
      this.fail()
    }
  }

  stop() {
    const video = this._video
    if (!video) return
    this._wrap.classList.remove('is-playing')
    video.pause()
    try {
      video.currentTime = 0
    } catch {}
  }

  fail() {
    this._broken = true
    this._video.style.display = 'none'
    this._badge.style.display = 'none'
    this._wrap.classList.remove('is-playing')
  }
}

/** 幂等注册实况图 */
export function defineLivePhoto() {
  if (typeof window === 'undefined' || !window.customElements) return
  if (window.customElements.get('mellow-live')) return
  window.customElements.define('mellow-live', MellowLive)
}
