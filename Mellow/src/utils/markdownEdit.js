/**
 * 文章编辑器文本层工具（markdown 源码 + 表情即时渲染）
 *
 * 设计：正文始终以 markdown 源码字符串为唯一数据源，编辑器 DOM 只是它的可视化形式：
 *  - [emoji:url] -> <img class="editor-emoji">（行内）
 *  - 图片 ![alt](url "t")、媒体短代码 @[video](url) 等一律保持源码文本，所见即所存；
 *    真正的渲染只发生在预览与正文（@/utils/markdown），编辑器里不做视觉替换。
 *
 * 本文件不依赖构建别名（URL 解析通过参数注入），便于独立测试。
 */

/** 行内表情标记（非全局，便于 test/exec） */
export const EMOJI_RE = /\[emoji:\s*([^\]\s]+)\s*\]/

/** 光标前恰好是一个完整表情标记（用于「边输入边渲染」判断，作用于未转义源码） */
export const TAIL_MARKER_RE = /(\[emoji:[^\]\s]+\])$/

/** 源码中是否含有可渲染的标记（目前仅表情） */
export const HAS_MARKER_RE = /\[emoji:\s*[^\]\s]+\]/

/** 会被视为「独立一行」的块级标签 */
const BLOCK_TAGS = new Set([
  'DIV', 'P', 'LI', 'UL', 'OL', 'BLOCKQUOTE',
  'H1', 'H2', 'H3', 'H4', 'H5', 'H6', 'PRE', 'FIGURE'
])

export const escapeHtml = (str) =>
  String(str ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

export const emojiMarker = (url) => `[emoji:${url}]`

export const imageMarker = (alt, src, title) =>
  `![${alt || ''}](${src || ''}${title ? ` "${title}"` : ''})`

/** 表情原子（行内表情图片） */
function emojiAtomHtml(url, resolveUrl) {
  const src = resolveUrl(url)
  return `<img src="${src}" data-emoji="${url}" class="inline-emoji editor-emoji" contenteditable="false" alt="emoji" loading="lazy" decoding="async">`
}

/**
 * 源码 -> 编辑器 HTML
 *
 * 只把 [emoji:url] 换成行内表情图；图片 markdown 与媒体短代码保持纯文本，
 * 用户在编辑器里直接看到并编辑 `![alt](url)` / `@[video](url)`。
 *
 * @param {string} text markdown 源码
 * @param {(url: string) => string} resolveUrl 表情地址解析（默认原样返回）
 */
export function sourceToHtml(text, resolveUrl = (url) => url) {
  let html = escapeHtml(text)
  html = html.replace(/\[emoji:\s*([^\]\s]+)\s*\]/g, (match, url) => emojiAtomHtml(url, resolveUrl))
  html = html.replace(/\n/g, '<br>')
  return html
}

/**
 * 取元素的原子标记（表情；兼容旧内容里残留的图片卡片），普通元素返回 null
 */
export function atomMarker(el) {
  if (!el || el.nodeType !== 1) return null
  const tag = el.tagName
  if (tag === 'IMG') {
    const emoji = el.getAttribute('data-emoji')
    if (emoji !== null) return emojiMarker(emoji)
    const src = el.getAttribute('src') || ''
    if (!src) return null
    return imageMarker(el.getAttribute('alt') || '', src, '')
  }
  if (tag === 'SPAN' && el.hasAttribute('data-md-src')) {
    return imageMarker(
      el.getAttribute('data-md-alt') || '',
      el.getAttribute('data-md-src') || '',
      el.getAttribute('data-md-title') || ''
    )
  }
  return null
}

/**
 * 编辑器 DOM -> 源码文本
 *
 * 同时收集「源码偏移 -> DOM 位置」映射（points），供光标还原使用，
 * 两条逻辑同源，避免偏移量对不上。
 *
 * @returns {{ text: string, points: Array<{offset:number,node:Node,pos:any}> }}
 */
export function serializeEditor(root) {
  let text = ''
  const points = []

  const walk = (parent) => {
    const children = Array.from(parent.childNodes || [])
    children.forEach((child, i) => {
      const isLast = i === children.length - 1
      if (child.nodeType === 3) {
        points.push({ offset: text.length, node: child, pos: 0 })
        text += child.data
        points.push({ offset: text.length, node: child, pos: child.data.length })
        return
      }
      if (child.nodeType !== 1) return

      const tag = child.tagName
      if (tag === 'BR') {
        // 末尾的 <br> 是浏览器占位符，不属于用户内容
        if (isLast) return
        points.push({ offset: text.length, node: child, pos: 'before' })
        text += '\n'
        return
      }

      const marker = atomMarker(child)
      if (marker !== null) {
        points.push({ offset: text.length, node: child, pos: 'before' })
        text += marker
        points.push({ offset: text.length, node: child, pos: 'after' })
        return
      }

      if (BLOCK_TAGS.has(tag)) {
        if (text && !text.endsWith('\n')) {
          points.push({ offset: text.length, node: child, pos: 'before' })
          text += '\n'
        }
        walk(child)
        return
      }

      walk(child)
    })
  }

  walk(root)
  return { text, points }
}

/**
 * 源码中的某个位置 -> DOM Range（用于还原光标 / 选区）
 *
 * 位置落在原子标记内部时，就近吸附到原子的前 / 后。
 */
export function rangeFromOffsets(root, points, start, end = start) {
  const range = document.createRange()

  const locate = (offset) => {
    let best = null
    for (const point of points) {
      if (point.offset > offset) break
      // 同一偏移取更靠前的点：例如「行尾」优先于「下一行行首」，避免光标跳到下一行
      if (!best || point.offset > best.offset) best = point
    }
    if (!best) return null
    const { node, pos } = best
    // 文本节点可以精确落在内部
    if (typeof pos === 'number' && node.nodeType === 3) {
      return { node, pos: Math.min(pos + (offset - best.offset), node.data.length) }
    }
    return { node, pos }
  }

  const apply = (point, isStart) => {
    if (!point) {
      if (isStart) {
        range.selectNodeContents(root)
        range.collapse(true)
      }
      return
    }
    const { node, pos } = point
    if (pos === 'before') {
      isStart ? range.setStartBefore(node) : range.setEndBefore(node)
    } else if (pos === 'after') {
      isStart ? range.setStartAfter(node) : range.setEndAfter(node)
    } else {
      isStart ? range.setStart(node, pos) : range.setEnd(node, pos)
    }
  }

  apply(locate(start), true)
  apply(locate(end), false)
  return range
}

/** DOM 中的某个点 -> 源码偏移（与 serializeEditor 同源，保证一致） */
export function offsetOfPoint(root, node, offset) {
  if (!root || !node || !root.contains(node)) return null
  const range = document.createRange()
  range.selectNodeContents(root)
  try {
    range.setEnd(node, offset)
  } catch {
    return null
  }
  return serializeEditor(range.cloneContents()).text.length
}

// ===== 纯字符串编辑操作（返回 { value, selStart, selEnd }） =====

/** 用 before/after 包裹选区 */
export function wrapText(value, start, end, before, after, placeholder) {
  const selected = value.slice(start, end) || placeholder
  const text = before + selected + after
  const selStart = start + before.length
  return {
    value: value.slice(0, start) + text + value.slice(end),
    selStart,
    selEnd: selStart + selected.length
  }
}

/** 给选中范围内的每一行加前缀 */
export function prefixLines(value, start, end, mark) {
  const lineStart = value.lastIndexOf('\n', start - 1) + 1
  const nl = value.indexOf('\n', end)
  const lineEnd = nl === -1 ? value.length : nl
  const block = value.slice(lineStart, lineEnd)
  const lines = block.split('\n').map((line) => (line.startsWith(mark) ? line.slice(mark.length) : mark + line))
  const text = lines.join('\n')
  return {
    value: value.slice(0, lineStart) + text + value.slice(lineEnd),
    selStart: lineStart,
    selEnd: lineStart + text.length
  }
}

/** 在选区处插入文本，并把光标放到 caretOffset（相对插入内容） */
export function insertTextAt(value, start, end, text, caretOffset) {
  const offset = caretOffset === undefined ? text.length : caretOffset
  const caret = start + offset
  return {
    value: value.slice(0, start) + text + value.slice(end),
    selStart: caret,
    selEnd: caret
  }
}

export const codeBlockText = (value, start, end) => {
  const selected = value.slice(start, end) || '在这里写代码'
  return insertTextAt(value, start, end, `\n\n\`\`\`js\n${selected}\n\`\`\`\n\n`, 7)
}

export const linkText = (value, start, end) => {
  const selected = value.slice(start, end) || '链接文字'
  const insert = `[${selected}](https://)`
  // 光标落在 url 处
  return insertTextAt(value, start, end, insert, insert.length - 1)
}

export const tableText = (value, start, end) =>
  insertTextAt(value, start, end, '\n\n| 列1 | 列2 |\n| --- | --- |\n| 内容 | 内容 |\n\n', 2)

/** 图片独占一行插入 */
export const blockMarkerText = (value, start, end, marker) =>
  insertTextAt(value, start, end, `\n\n${marker}\n\n`, 2 + marker.length)
