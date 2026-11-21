import { call } from '@/api/request'
// 表情图片为后端静态资源，统一复用运行时地址解析（见 @/utils/runtimeConfig）
import { getFullUrl } from '@/utils/runtimeConfig'

/**
 * 表情渲染工具
 * 后端规范：[emoji:表情链接] -> <img>
 *
 * ⚠️ 安全约定（改这个文件前必读）
 *
 * 正文是**用户输入**：动态 / 文章 / 独立页的正文在后端**没有** XSS 过滤（只有评论有）。
 * 因此对外只提供「**先整体转义、再做标记替换**」的渲染函数
 * （`renderEmoji` / `renderEmojiWithBreaks`），绝不能把原始文本直接拼进 `v-html`：
 * 否则任何人发一条 `<img src=x onerror=...>` 的动态，所有访客（含管理员）打开即执行，
 * 配合 token 存于可读 cookie / localStorage，等于账号接管。
 *
 * `renderMention` / `renderHashtag` 是内部函数，**只接受已经过 `escapeHtml` 的文本**，
 * 所以不再导出（避免被从外部拿原始文本误用）。要渲染纯文本标题 / 摘要里的表情，
 * 请用 `components/EmojiText.vue`（元素渲染，不走 v-html）。
 */

export { getFullUrl }

/** HTML 转义（v-html 前的兜底） */
export const escapeHtml = (text) => {
  return String(text ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

/**
 * HTML 还原（只用于从已转义文本里取回 [emoji:url] 的原始地址）
 *
 * 注意 `&amp;` 必须**最后**替换，否则 `&amp;lt;` 会被提前还原成 `<`。
 */
export const unescapeHtml = (text) => {
  return String(text ?? '')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&amp;/g, '&')
}

// 已规范化的 @提及：[昵称](/author/12)（见 app/model/mention.go）
const MENTION_RE = /@\[([^\[\]]*)\]\(\/author\/(\d+)\)/g

/**
 * 渲染 @ 提及（内部函数：`content` 必须是已转义文本）
 *
 * 后端保存内容时会把 @昵称 规范化为 @[昵称](/author/12)（只规范化「已关注的人」）。
 * 文章 / 独立页由 markdown-it 渲染，评论 / 动态走这里，四处表现一致。
 */
function renderMention(content) {
  if (!content || !content.includes('@[')) return content
  return content.replace(MENTION_RE, (match, name, id) => {
    return `<a class="mention" data-mention-id="${id}" href="/author/${id}">@${name}</a>`
  })
}

// 已规范化的 #话题#：[话题名](/topic/12)（见 app/model/topics.go）
const HASHTAG_RE = /#\[([^\[\]]*)\]\(\/topic\/(\d+)\)/g

/** 渲染 #话题#（内部函数：`content` 必须是已转义文本） */
function renderHashtag(content) {
  if (!content || !content.includes('#[')) return content
  return content.replace(HASHTAG_RE, (match, name, id) => {
    return `<a class="hashtag" data-topic-id="${id}" href="/topic/${id}">#${name}#</a>`
  })
}

/**
 * 表情地址白名单（供 v-html 与编辑器共用）
 *
 * [emoji:] 后面是可以带 `"` 的，若直接塞进 `src="..."` 就能闭合属性注入 onerror，
 * 所以这里只放行「http(s) 或站内相对路径」且**不含空白 / 引号 / 尖括号**的地址，
 * 返回已转义的地址；不合法返回空串（调用方保留原文，当作普通文本显示）。
 */
export function safeEmojiUrl(url) {
  const value = unescapeHtml(String(url || '')).trim()
  if (!value) return ''
  if (!/^(https?:\/\/|\/)/i.test(value)) return ''
  if (/[\s"'<>]/.test(value)) return ''
  return escapeHtml(getFullUrl(value))
}

/**
 * 渲染含 [emoji:url] 的文本为 HTML（v-html 的安全入口）
 *
 * 顺序很重要：**先整体转义，再做标记替换**。
 * @提及 / #话题# / [emoji:] 这些标记里都不含 `< > & " '`，转义不会破坏它们；
 * 而转义能保证正文里的 `<img onerror=...>` 之类只是普通文本。
 *
 * @param {string} content 原始内容（用户输入，不要预先转义）
 * @param {object} options { size: 图片尺寸(px) }
 * @returns {string} HTML 字符串（需配合 v-html 使用）
 */
export const renderEmoji = (content, options = {}) => {
  if (!content) return ''
  const size = options.size || 24

  let html = escapeHtml(String(content))

  // @ 提及、#话题# 转链接：链接文字在上一步已转义，不会出现 [emoji:]，互不干扰
  html = renderHashtag(renderMention(html))

  // [emoji:url] → <img>（地址白名单 + 转义，失败则保留原文）
  html = html.replace(/\[emoji:\s*([^\]]+)\]/g, (match, url) => {
    const fullUrl = safeEmojiUrl(url)
    if (!fullUrl) return match
    return `<img src="${fullUrl}" alt="emoji" class="inline-emoji" style="width:${size}px;height:${size}px;vertical-align:middle;display:inline-block;object-fit:contain;margin:0 2px;" loading="lazy" decoding="async">`
  })

  return html
}

/**
 * 渲染含 [emoji:url] 的文本，同时保留换行（转 <br>）
 */
export const renderEmojiWithBreaks = (content, options = {}) => {
  if (!content) return ''
  let html = renderEmoji(content, options)
  html = html.replace(/\n/g, '<br>')
  return html
}

/**
 * 获取后端表情包列表
 * @returns {Promise<Array>} 分类数组 [{ name, items: [{name, url}] }]
 */
export const fetchEmojiCategories = async () => {
  try {
    const res = await call('attachment', 'emoji', { method: 'GET' })
    if (res.code === 200 && res.data?.categories) {
      return res.data.categories
    }
    return []
  } catch {
    return []
  }
}

/**
 * 从文本中提取所有 emoji url（用于编辑时还原）
 */
export const extractEmojiUrls = (content) => {
  if (!content) return []
  const urls = []
  const regex = /\[emoji:\s*([^\]]+)\]/g
  let m
  while ((m = regex.exec(String(content))) !== null) {
    urls.push(m[1].trim())
  }
  return urls
}
