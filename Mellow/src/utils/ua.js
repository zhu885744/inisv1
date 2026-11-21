/**
 * User-Agent 解析（用于评论区里显示「来自什么设备」）
 *
 * 数据来源：评论接口返回的 `agent` 字段。
 * - comment/flat 未传 field 时返回全字段，因此 `agent` 本来就在响应里；
 * - 普通用户拿到的是后端脱敏值（超 50 字符截断加 `...`，见 facade.Comm.MaskUA），
 *   管理员能看到完整 UA（见 controller/comment.go 的 maskItem）。
 *
 * 为什么不直接铺原始 UA：一串
 * 「Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36」
 * 又长又难读，所以按后端账号登录通知的口径（model/notification.go 的 parseDevice）
 * 解析成「系统 · 浏览器 版本」，原始 UA 由调用方放进 `title` 提示，悬停仍可看到。
 */

// 系统识别：有序匹配，先命中者生效（Windows NT 版本号要写在通用 Windows 之前）
const OS_RULES = [
  { re: /Windows NT 10\.0/i, name: 'Windows 10' },
  { re: /Windows NT 6\.3/i, name: 'Windows 8.1' },
  { re: /Windows NT 6\.2/i, name: 'Windows 8' },
  { re: /Windows NT 6\.1/i, name: 'Windows 7' },
  { re: /Windows/i, name: 'Windows' },
  { re: /HarmonyOS/i, name: 'HarmonyOS' },
  { re: /Android/i, name: 'Android' },
  { re: /(iPhone|iPad|iPod)/i, name: 'iOS' },
  { re: /(Mac OS X|Macintosh)/i, name: 'macOS' },
  { re: /Linux/i, name: 'Linux' }
]

// 浏览器识别：应用内浏览器（微信 / QQ / UC）要排在 Chrome、Safari 之前 ——
// 它们的 UA 里同样带 Chrome/ 或 Safari/，顺序错了就会认成 Chrome。
const BROWSER_RULES = [
  { re: /MicroMessenger|WeChat/i, name: '微信' },
  { re: /QQBrowser/i, name: 'QQ 浏览器' },
  { re: /UCBrowser/i, name: 'UC 浏览器' },
  { re: /Edg[A-Z]?\//i, name: 'Edge' },
  { re: /(OPR\/|Opera)/i, name: 'Opera' },
  { re: /Firefox\//i, name: 'Firefox' },
  { re: /Chrome\//i, name: 'Chrome' },
  { re: /(Safari\/|Version\/)/i, name: 'Safari' }
]

// 系统 → 图标（bootstrap-icons）；没命中时用显示器图标兜底
const OS_ICONS = [
  { re: /Windows/i, icon: 'bi bi-windows' },
  { re: /Android/i, icon: 'bi bi-android2' },
  { re: /(iOS|macOS)/i, icon: 'bi bi-apple' },
  { re: /Linux/i, icon: 'bi bi-ubuntu' },
  { re: /HarmonyOS/i, icon: 'bi bi-phone' }
]

// 爬虫 / 命令行工具：这类 UA 里常常没有系统和浏览器标识，单独标注更直观
const BOT_RE = /(bot|crawler|spider|curl|wget|python-requests|postmanruntime|okhttp)/i

// 取主版本号用的标记（按浏览器名对应）
const VERSION_TOKENS = {
  Edge: 'Edg/',
  Opera: 'OPR/',
  Firefox: 'Firefox/',
  Chrome: 'Chrome/',
  微信: 'MicroMessenger/',
  'QQ 浏览器': 'QQBrowser/',
  'UC 浏览器': 'UCBrowser/',
  Safari: 'Version/'
}

// 「Edg/120.0.0.0」→「120」：只保留主版本号，和登录通知里的写法一致
function pickVersion(agent, browser) {
  const token = VERSION_TOKENS[browser]
  // 自己的标记取不到时退回 Chrome 版本（应用内浏览器大多是 Chromium 内核）
  const tokens = token ? [token, 'Chrome/'] : ['Chrome/']

  for (const item of tokens) {
    const at = agent.indexOf(item)
    if (at < 0) continue

    const match = agent.slice(at + item.length).match(/^\d+/)
    if (match) return match[0]
  }

  return ''
}

/**
 * 解析 User-Agent 为简短设备描述
 *
 * @param {string} agent 原始（或后端脱敏后的）UA
 * @returns {{ os: string, browser: string, version: string, label: string, icon: string, bot: boolean }}
 *          label 为空表示 UA 无法识别，调用方据此隐藏设备标签
 */
export function parseUserAgent(agent) {
  const raw = String(agent || '').trim()

  if (!raw) {
    return { os: '', browser: '', version: '', label: '', icon: '', bot: false }
  }

  // 爬虫：不做系统 / 浏览器解读，直接标注
  if (BOT_RE.test(raw)) {
    return { os: '', browser: '', version: '', label: '机器人 / 爬虫', icon: 'bi bi-robot', bot: true }
  }

  const os = OS_RULES.find((rule) => rule.re.test(raw))?.name || ''
  const browser = BROWSER_RULES.find((rule) => rule.re.test(raw))?.name || ''
  const version = browser ? pickVersion(raw, browser) : ''

  const label = [os, [browser, version].filter(Boolean).join(' ')].filter(Boolean).join(' · ')
  const icon = OS_ICONS.find((rule) => rule.re.test(os))?.icon || 'bi bi-display'

  return { os, browser, version, label, icon, bot: false }
}
