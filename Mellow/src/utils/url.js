/**
 * 链接安全工具
 *
 * ⚠ 为什么需要这一层（改渲染代码前必读）
 *
 * Vue 的 `:href` **不会**拦截 `javascript:` 协议（只有 React 早年做过提示，Vue 从来没有）。
 * 而本主题里有多处 href 来自「用户可写」的字段：
 *
 *   - 用户资料里的个人网站（users.json.website.url），展示在 /author/{id}；
 *   - 友链（用户提交、管理员审核）；
 *   - 后台公告、自定义菜单、侧栏链接。
 *
 * 保存时前端虽然校验了 `^https?://`，但那只是**输入框上的**约束：直接调
 * `/api/users/update` 提交 `json.website.url = "javascript:..."` 就能绕过去。
 * 之后任何访客（包括管理员）点开那个作者主页上的「个人网站」胶囊，脚本就在本站
 * 域下执行 —— token 存在可读 cookie / localStorage，等于账号接管。
 *
 * 因此**渲染端必须再判一次**：只放行明确安全的协议，其余一律不生成链接。
 */

// 控制字符与零宽字符：URL 解析器会忽略它们（`java\tscript:` 仍算 javascript:），
// 但 `startsWith('javascript:')` 这种朴素判断会被绕过 —— 先剥干净再判断。
const CONTROL_CHARS = /[\u0000-\u0020\u007f-\u009f\u200b-\u200d\u2060\ufeff]/g

// 允许的协议：只保留跳转类，data: / javascript: / vbscript: / file: 全部拒绝
const SAFE_PROTOCOLS = new Set(['http', 'https', 'mailto', 'tel'])

// 形如 `scheme:` 的开头（scheme 允许 + . -，与 RFC 3986 一致）
const SCHEME_RE = /^([a-z][a-z0-9+.-]*):/i

/** 取 URL 的协议名；无协议（相对路径 / 锚点 / 裸域名）返回空串 */
export function protocolOf(value) {
  const cleaned = String(value ?? '').replace(CONTROL_CHARS, '')
  const matched = SCHEME_RE.exec(cleaned)
  return matched ? matched[1].toLowerCase() : ''
}

/**
 * 判断链接是否可以安全地放进 href / src
 *
 * 允许：http / https / mailto / tel，以及不含协议的相对路径、锚点、协议相对地址（//host）。
 * 拒绝：javascript:、data:、vbscript:，以及 `java\tscript:` 这类加了控制字符的变体。
 */
export function isSafeUrl(value) {
  const raw = String(value ?? '').replace(CONTROL_CHARS, '').trim()
  if (!raw) return false
  const protocol = protocolOf(raw)
  if (!protocol) return true
  return SAFE_PROTOCOLS.has(protocol)
}

/**
 * 规范化外链：安全则原样返回（已去除控制字符与首尾空白），不安全返回空串
 *
 * 用法：`:href="safeUrl(item.url) || undefined"`
 * 返回空串时绑定成 undefined，Vue 会移除 href 属性，<a> 变成不可点击的占位，
 * 避免退化成 `href=""`（那会导航到当前页）。
 */
export function safeUrl(value) {
  const raw = String(value ?? '').replace(CONTROL_CHARS, '').trim()
  if (!raw || !isSafeUrl(raw)) return ''
  return raw
}

/**
 * 站内路径判断（供「站内走 router、站外新窗口」的分支使用）
 *
 * 只有以单个 `/` 开头（且不是 `//` 协议相对）才算站内；
 * 不安全的值一律按「不可跳转」处理，不交给 router。
 */
export function isInternalPath(value) {
  const raw = safeUrl(value)
  return raw.startsWith('/') && !raw.startsWith('//')
}
