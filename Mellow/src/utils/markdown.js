import MarkdownIt from 'markdown-it'
// 按需引入 highlight.js：默认入口会把 190+ 种语言全部注册进 bundle（min 后 500KB+），
// 而正文页（文章 / 独立页）每次打开都要下载它。这里只注册博客实际会出现的语言，
// 未注册的语言会退化成不高亮的纯文本（不影响阅读），需要时在下面补一行即可。
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import c from 'highlight.js/lib/languages/c'
import cpp from 'highlight.js/lib/languages/cpp'
import csharp from 'highlight.js/lib/languages/csharp'
import css from 'highlight.js/lib/languages/css'
import diff from 'highlight.js/lib/languages/diff'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import go from 'highlight.js/lib/languages/go'
import ini from 'highlight.js/lib/languages/ini'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import less from 'highlight.js/lib/languages/less'
import markdown from 'highlight.js/lib/languages/markdown'
import nginx from 'highlight.js/lib/languages/nginx'
import php from 'highlight.js/lib/languages/php'
import plaintext from 'highlight.js/lib/languages/plaintext'
import python from 'highlight.js/lib/languages/python'
import rust from 'highlight.js/lib/languages/rust'
import scss from 'highlight.js/lib/languages/scss'
import shell from 'highlight.js/lib/languages/shell'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'
// 表情图片地址统一走运行时解析（与 @/utils/emoji 的 renderEmoji 保持一致）
import { getFullUrl } from '@/utils/runtimeConfig'
import { defineAudioPlayer, defineLivePhoto, defineVideoPlayer } from '@/utils/mediaPlayer'

// 注册正文音视频播放器（<mellow-audio> / <mellow-video> 自定义元素，幂等）
// v-html 插入同名标签时浏览器会自动升级，文章 / 独立页 / 文档 / 编辑器预览都不用单独接
defineAudioPlayer()
defineVideoPlayer()
defineLivePhoto()

// 语言名 + 常见别名（同一份语法注册多次，兼容 ```js / ```javascript 等写法）
const HIGHLIGHT_LANGUAGES = {
  javascript, js: javascript, jsx: javascript, mjs: javascript, cjs: javascript, node: javascript,
  typescript, ts: typescript, tsx: typescript,
  json, jsonc: json,
  bash, sh: bash, zsh: bash,
  shell, console: shell,
  go, golang: go,
  python, py: python,
  java, kotlin: java,
  php,
  rust, rs: rust,
  c, cpp, 'c++': cpp, cc: cpp, h: c, hpp: cpp, csharp, cs: csharp,
  sql, mysql: sql,
  xml, html: xml, vue: xml, svg: xml,
  css, scss, less,
  yaml, yml: yaml,
  ini, toml: ini, conf: ini,
  markdown, md: markdown,
  diff, patch: diff,
  dockerfile, docker: dockerfile,
  nginx, conf_nginx: nginx,
  plaintext, text: plaintext, txt: plaintext
}

Object.entries(HIGHLIGHT_LANGUAGES).forEach(([name, language]) => {
  hljs.registerLanguage(name, language)
})

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight(str, lang) {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs"><code>${
          hljs.highlight(str, { language: lang, ignoreIllegals: true }).value
        }</code></pre>`
      } catch (e) {}
    }
    return `<pre class="hljs"><code>${md.utils.escapeHtml(str)}</code></pre>`
  }
})

// 自定义链接默认在新窗口打开
const defaultLinkOpen = md.renderer.rules.link_open || function (tokens, idx, options, env, self) {
  return self.renderToken(tokens, idx, options)
}
md.renderer.rules.link_open = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  const hrefIndex = token.attrIndex('href')
  if (hrefIndex >= 0) {
    const href = token.attrs[hrefIndex][1]
    if (href.startsWith('http')) {
      token.attrSet('target', '_blank')
      token.attrSet('rel', 'noopener noreferrer')
    }
    // @ 提及：后端保存时把 @昵称 规范化为 @[昵称](/author/12)（见 app/model/mention.go），
    // 这里补一个 class，供主题样式与 main.js 的 SPA 点击拦截识别
    if (href.startsWith('/author/')) {
      token.attrSet('class', 'mention')
    }
    // 话题：后端保存时把 #话题# 规范化为 #[话题名](/topic/12)（见 app/model/topics.go），
    // 同样补 class，供主题样式与 SPA 点击拦截识别
    if (href.startsWith('/topic/')) {
      token.attrSet('class', 'hashtag')
    }
  }
  return defaultLinkOpen(tokens, idx, options, env, self)
}

// 正文图片（![alt](src)）：统一补懒加载 + 异步解码。
// 文章 / 独立页面 / 文档正文都是长文，一次性把所有配图都拉下来既慢又费流量。
const defaultImage = md.renderer.rules.image || function (tokens, idx, options, env, self) {
  return self.renderToken(tokens, idx, options)
}
md.renderer.rules.image = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  const src = token.attrGet('src') || ''
  // 把视频 / 音频地址当图片贴进来（![](a.mp4)）时自动升级成播放器：
  // 这是不熟悉短代码的作者最自然的写法，升级一下比显示成裂图友好得多
  if (MEDIA_VIDEO_EXT.test(src)) return renderMedia('video', src, token.content)
  if (MEDIA_AUDIO_EXT.test(src)) return renderMedia('audio', src, token.content)
  token.attrSet('loading', 'lazy')
  token.attrSet('decoding', 'async')
  return defaultImage(tokens, idx, options, env, self)
}

/**
 * 行内表情：[emoji:表情链接] -> <img class="inline-emoji">
 *
 * 与主题其它页面（动态 / 评论 / 通知）的 renderEmoji 使用同一存储规范，
 * 这样文章正文、独立页面、编辑器预览里的表情也能正常显示，
 * 而不是把 [emoji:https://...] 原样当文本输出。
 */
const EMOJI_RE = /^\[emoji:\s*([^\]\s]+)\s*\]/

md.inline.ruler.before('link', 'emoji', function (state, silent) {
  const match = EMOJI_RE.exec(state.src.slice(state.pos))
  if (!match) return false

  if (!silent) {
    const token = state.push('emoji', 'img', 0)
    token.attrSet('src', getFullUrl(match[1]))
  }
  state.pos += match[0].length
  return true
})

md.renderer.rules.emoji = function (tokens, idx) {
  const src = md.utils.escapeHtml(tokens[idx].attrGet('src') || '')
  if (!src) return ''
  // class="inline-emoji" 由全局样式统一尺寸（styles.css）
  return `<img src="${src}" alt="emoji" class="inline-emoji" loading="lazy" decoding="async">`
}

/* ============================================================
 * 正文媒体 / 附件短代码
 *
 *   @[video](url "标题")
 *   @[audio](url "标题")
 *   @[file](url "文件名" "2.3 MB")
 *
 * 为什么用短代码而不是直接写 <video>：
 *  1. 上面的 markdown-it 是 html:false —— 正文里的 HTML 会被当纯文本转义，写了也不生效；
 *  2. 把 html 打开等于让每个作者都能往正文里塞标签（正文是 v-html 渲染的），XSS 风险不可控；
 *  3. 短代码是纯文本：可搜索、可迁移、可回退，后端存储结构完全不用动，老文章零影响。
 * ============================================================ */

const MEDIA_SHORTCODE_RE = /^@\[(video|audio|file|live)\]\(\s*([^)\s]+)\s*(?:"([^"]*)")?\s*(?:"([^"]*)")?\s*\)/
const MEDIA_VIDEO_EXT = /\.(mp4|webm|ogv|mov|m4v)(\?|#|$)/i
const MEDIA_AUDIO_EXT = /\.(mp3|wav|ogg|m4a|aac|flac)(\?|#|$)/i

/**
 * 短代码里的 URL 来自用户输入，只放行 http(s) 与站内相对路径
 * （javascript: / data: 一律不渲染，避免变成 XSS 入口）
 */
function safeMediaUrl(url) {
  const value = String(url || '').trim()
  if (!value) return ''
  if (/^(https?:)?\/\//i.test(value) || value.startsWith('/')) return value
  return ''
}

/** 从 URL 里取文件名（短代码没写文件名时的兜底） */
function mediaFileName(url) {
  const clean = String(url || '').split('?')[0].split('#')[0]
  try {
    return decodeURIComponent(clean.slice(clean.lastIndexOf('/') + 1))
  } catch {
    return clean.slice(clean.lastIndexOf('/') + 1)
  }
}

/**
 * 媒体 / 附件卡片 HTML
 *
 * 用 span / a 而不是 div：短代码常写在一行文字里，markdown-it 会把它包进 <p>，
 * 块级元素嵌在 <p> 里会被浏览器强行拆开；span 靠 display:block 拿到同样的版式。
 * 所有用户输入（标题、文件名、大小）都转义后再拼接。
 */
function renderMedia(kind, url, caption, meta) {

  const raw = safeMediaUrl(url)
  if (!raw) {
    // URL 非法就当普通文本显示，至少不让内容凭空消失
    return md.utils.escapeHtml(`@[${kind}](${url || ''})`)
  }

  const src = md.utils.escapeHtml(getFullUrl(raw))
  const title = caption
    ? `<span class="markdown-media-caption">${md.utils.escapeHtml(caption)}</span>`
    : ''

  if (kind === 'video') {
    // 同音频：浏览器里走自绘播放器，不支持自定义元素时退回原生 controls
    if (typeof window !== 'undefined' && window.customElements) {
      const captionAttr = caption ? ` caption="${md.utils.escapeHtml(caption)}"` : ''
      return `<mellow-video class="markdown-video" src="${src}"${captionAttr}></mellow-video>`
    }
    return (
      `<span class="markdown-media markdown-video">` +
      `<video src="${src}" controls preload="metadata" playsinline></video>` +
      title +
      `</span>`
    )
  }

  if (kind === 'audio') {
    // 浏览器里走自绘播放器（原生 controls 只能改尺寸，做不出卡片 + 渐变进度条的观感）；
    // 极端环境（无自定义元素支持）退回原生 <audio controls>
    if (typeof window !== 'undefined' && window.customElements) {
      const captionAttr = caption ? ` caption="${md.utils.escapeHtml(caption)}"` : ''
      return `<mellow-audio class="markdown-audio" src="${src}"${captionAttr}></mellow-audio>`
    }
    return (
      `<span class="markdown-media markdown-audio">` +
      `<audio src="${src}" controls preload="metadata"></audio>` +
      title +
      `</span>`
    )
  }

  const urlName = mediaFileName(raw)
  const fileName = caption || urlName || '附件'
  // 扩展名只从 URL 里取：标题可能是任意中文，拿它 split('.') 会得出奇怪的类型
  const ext = urlName.includes('.') ? (urlName.split('.').pop() || '').toLowerCase() : ''
  const metaText = [meta, ext].filter(Boolean).join(' · ')

  return (
    `<a class="markdown-file" href="${src}" target="_blank" rel="noopener noreferrer" download>` +
    `<span class="markdown-file-icon"><i class="bi bi-file-earmark-arrow-down"></i></span>` +
    `<span class="markdown-file-main">` +
    `<span class="markdown-file-name">${md.utils.escapeHtml(fileName)}</span>` +
    `<span class="markdown-file-meta">${md.utils.escapeHtml(metaText)}</span>` +
    `</span>` +
    `<span class="markdown-file-action"><i class="bi bi-download"></i> 下载</span>` +
    `</a>`
  )
}

/**
 * 实况图 HTML：@[live](视频URL "封面图URL" "说明")
 *
 * 文章正文的图片是 markdown 语法，一个 `![]()` 只能带一个 URL，放不下「封面 + 视频」这一对，
 * 所以复用媒体短代码：位置参数是视频（实况部分），第一个引号参数是静态封面。
 * 渲染成 <mellow-live>（见 utils/mediaPlayer.js）：悬停 / 按住播放，播不了就只显示封面。
 */
function renderLive(videoUrl, imageUrl, caption) {

  const video = safeMediaUrl(videoUrl)
  const image = safeMediaUrl(imageUrl)
  if (!video || !image) {
    // 参数不全就原样显示源码，至少不让内容凭空消失
    return md.utils.escapeHtml(`@[live](${videoUrl || ''}${imageUrl ? ` "${imageUrl}"` : ''})`)
  }

  const src = md.utils.escapeHtml(getFullUrl(video))
  const cover = md.utils.escapeHtml(getFullUrl(image))
  const text = String(caption || '')
  const captionAttr = text ? ` caption="${md.utils.escapeHtml(text)}"` : ''

  if (typeof window !== 'undefined' && window.customElements) {
    return `<mellow-live class="markdown-live" src="${src}" image="${cover}"${captionAttr}></mellow-live>`
  }

  // 兜底：封面图 + 原生视频（没有自定义元素支持的极端环境）
  return (
    `<span class="markdown-media markdown-live">` +
    `<img src="${cover}" alt="${md.utils.escapeHtml(text || '实况图')}" loading="lazy" decoding="async">` +
    `<video src="${src}" muted loop playsinline preload="metadata"></video>` +
    `</span>`
  )
}

md.inline.ruler.before('link', 'media_shortcode', function (state, silent) {
  // 绝大多数字符不是 @，先做一次廉价判断，避免每个字符都跑正则
  if (state.src.charCodeAt(state.pos) !== 0x40) return false

  const match = MEDIA_SHORTCODE_RE.exec(state.src.slice(state.pos))
  if (!match) return false

  if (!silent) {
    const token = state.push('media_shortcode', 'span', 0)
    token.meta = { kind: match[1], url: match[2], caption: match[3] || '', size: match[4] || '' }
  }

  state.pos += match[0].length
  return true
})

md.renderer.rules.media_shortcode = function (tokens, idx) {
  const { kind, url, caption, size } = tokens[idx].meta || {}
  // 实况图：URL 是视频，第一个引号参数是封面图，第二个是说明
  if (kind === 'live') return renderLive(url, caption, size)
  return renderMedia(kind, url, caption, size)
}

export const renderMarkdown = (content) => {
  if (!content) return ''
  return md.render(content)
}
