// 富文本编辑器标识（值为这些时 content 按富文本 HTML 渲染）
const RICH_EDITORS = ['wangeditor', 'quill', 'tinymce', 'rich', 'richtext', 'rich-text', 'html']

// 判断某文章是否使用富文本编辑器（editor 字段来自 app/model/article.go）
export const isRichTextEditor = (editor) => {
  if (!editor) return false
  const e = String(editor).trim().toLowerCase()
  if (!e || e === 'vditor' || e === 'markdown' || e === 'md') return false
  return RICH_EDITORS.includes(e)
}

// 会被浏览器当成地址解析的属性：这些值必须过协议白名单
const URL_ATTRS = new Set([
  'href', 'src', 'xlink:href', 'action', 'formaction', 'poster',
  'background', 'cite', 'data', 'srcset', 'longdesc', 'ping'
])

// 允许的协议（与 utils/url.js 的口径保持一致）
const SAFE_SCHEMES = new Set(['http', 'https', 'mailto', 'tel'])

// 图片类属性额外放行内联图片（历史富文本里可能存了 base64 配图），
// 但**排除 svg** —— data:image/svg+xml 里可以带脚本
const SAFE_DATA_IMAGE = /^data:image\/(png|jpe?g|gif|webp|avif|bmp);/i
const IMAGE_ATTRS = new Set(['src', 'poster', 'background'])

// 去掉控制字符与空白后再判断协议。
// 浏览器解析 URL 时会忽略 tab / 换行 / 回车，`java\tscript:alert(1)` 依然执行，
// 而 `value.startsWith('javascript:')` 这种朴素判断会被这类写法绕过。
const stripForScheme = (value) => String(value || '').replace(/[\u0000-\u0020\u007f-\u009f\u200b-\u200d\ufeff]/g, '')

/**
 * 属性值里是否含有不该出现的协议
 *
 * 判据是「有没有协议」而不是「像不像 javascript:」：只要带了协议且不在白名单里就拒绝，
 * 这样 data:、vbscript:、file: 以及将来可能出现的奇怪协议都一并挡掉。
 */
export function hasUnsafeScheme(value, attrName) {
  const raw = String(value || '')
  // srcset 是「URL + 描述符」的候选列表，逐个取首个 token 判断
  const candidates = attrName === 'srcset'
    ? raw.split(',').map((item) => item.trim().split(/\s+/)[0])
    : [raw]

  return candidates.some((candidate) => {
    const cleaned = stripForScheme(candidate)
    if (!cleaned) return false
    const matched = /^([a-z][a-z0-9+.-]*):/i.exec(cleaned)
    // 没有协议 = 相对路径 / 锚点，安全
    if (!matched) return false
    const scheme = matched[1].toLowerCase()
    if (SAFE_SCHEMES.has(scheme)) return false
    if (IMAGE_ATTRS.has(attrName) && SAFE_DATA_IMAGE.test(cleaned)) return false
    return true
  })
}

// 轻量 HTML 清洗：剥离脚本/iframe 等危险标签与事件属性，防 XSS。
// 纯浏览器 API 实现，无第三方依赖；保留常用富文本标签与内联样式。
export const sanitizeHtml = (html) => {
  if (!html) return ''
  const doc = new DOMParser().parseFromString(String(html), 'text/html')

  // 1. 移除危险标签
  const dangerous = doc.querySelectorAll(
    'script, style, frame, object, embed, link, meta, base, form, input, button, textarea, select'
  )
  dangerous.forEach((el) => el.remove())

  // 2. iframe 仅允许 https/http 的 src（视频嵌入），其余移除
  doc.querySelectorAll('iframe').forEach((el) => {
    const src = (el.getAttribute('src') || '').trim().toLowerCase()
    if (!/^https?:/.test(src)) el.remove()
  })

  // 3. 剥离事件属性、srcdoc 与不安全协议的地址
  doc.querySelectorAll('*').forEach((el) => {
    for (const attr of Array.from(el.attributes)) {
      const name = attr.name.toLowerCase()
      if (name.startsWith('on') || name === 'srcdoc') {
        el.removeAttribute(attr.name)
        continue
      }
      if (URL_ATTRS.has(name) && hasUnsafeScheme(attr.value, name)) {
        el.removeAttribute(attr.name)
      }
    }
  })

  return doc.body.innerHTML
}
