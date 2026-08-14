<!-- src\comps\custom\i-markdown.vue 文章markdown内容渲染组件 -->
<template>
  <div ref="containerRef" class="markdown-content" v-html="renderedMd" @click="handleCopyClick"></div>
</template>

<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { marked } from 'marked'

// ==============================================
// highlight.js 按需引入
// 完整包会打入约 190 种语言（压缩后约 300KB），
// 这里只注册常用语言，其余语言按 plaintext 渲染
// ==============================================
import hljs from 'highlight.js/lib/core'
import 'highlight.js/styles/agate.css'

import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import scss from 'highlight.js/lib/languages/scss'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import markdown from 'highlight.js/lib/languages/markdown'
import bash from 'highlight.js/lib/languages/bash'
import go from 'highlight.js/lib/languages/go'
import python from 'highlight.js/lib/languages/python'
import java from 'highlight.js/lib/languages/java'
import php from 'highlight.js/lib/languages/php'
import sql from 'highlight.js/lib/languages/sql'
import ini from 'highlight.js/lib/languages/ini'
import diff from 'highlight.js/lib/languages/diff'
import plaintext from 'highlight.js/lib/languages/plaintext'

// 语言注册表：键为语言名，值为对应的语法定义
const LANGUAGES = {
  javascript, typescript, xml, css, scss, json, yaml,
  markdown, bash, go, python, java, php, sql, ini, diff, plaintext
}

Object.entries(LANGUAGES).forEach(([name, lang]) => hljs.registerLanguage(name, lang))

// 常用别名，便于识别 ```js / ```vue / ```sh 之类的写法
const LANGUAGE_ALIASES = {
  js: 'javascript', jsx: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', tsx: 'typescript',
  html: 'xml', vue: 'xml', svg: 'xml',
  sh: 'bash', shell: 'bash', zsh: 'bash', console: 'bash',
  py: 'python', golang: 'go', yml: 'yaml',
  md: 'markdown', conf: 'ini', toml: 'ini',
  text: 'plaintext', txt: 'plaintext'
}

// 转义 HTML，用于高亮失败时安全地原样输出代码
const escapeHtml = (str = '') => String(str)
  .replace(/&/g, '&amp;')
  .replace(/</g, '&lt;')
  .replace(/>/g, '&gt;')

// 归一化语言名：未注册的语言统一降级为 plaintext，避免高亮抛错
const resolveLanguage = (lang) => {
  if (!lang || lang === 'language') return 'plaintext'
  const normalized = String(lang).toLowerCase()
  const mapped = LANGUAGE_ALIASES[normalized] || normalized
  return hljs.getLanguage(mapped) ? mapped : 'plaintext'
}

// props
const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
})

const renderedMd = ref('')
const containerRef = ref(null)

// 渲染 markdown
const renderMarkdown = (content) => {
  if (!content) {
    renderedMd.value = ''
    return
  }
  
  let processedContent = content
  
  processedContent = processedContent.replace(/```(\w+)?\n([\s\S]*?)```/g, (match, lang, code) => {
    try {
      const safeLang = resolveLanguage(lang)
      const highlighted = hljs.highlight(code.trim(), { language: safeLang }).value
      return `<div class="code-block-container">
        <div class="code-block-header">
          <span class="code-language">${safeLang}</span>
          <button class="copy-button" data-code="${code.replace(/"/g, '&quot;')}">
            <i class="bi bi-clipboard"></i>
            <span class="copy-text">复制</span>
          </button>
        </div>
        <pre class="hljs"><code class="language-${safeLang}">${highlighted}</code></pre>
      </div>`
    } catch (error) {
      console.error('代码高亮处理失败:', error)
      return `<div class="code-block-container">
        <div class="code-block-header">
          <span class="code-language">plaintext</span>
          <button class="copy-button" data-code="${code.replace(/"/g, '&quot;')}">
            <i class="bi bi-clipboard"></i>
            <span class="copy-text">复制</span>
          </button>
        </div>
        <pre class="hljs"><code>${escapeHtml(code)}</code></pre>
      </div>`
    }
  })
  
  let html = marked.parse(processedContent, {
    gfm: true,
    breaks: true,
    html: true
  })
  
  html = html.replace(/<img\s+src="([^"]+)"\s+alt="([^"]*)"\s*(.*?)\s*>/g, '<a href="$1" data-fancybox="gallery" data-caption="$2"><img src="$1" alt="$2" $3></a>')
  
  html = html.replace(/<a\s+([^>]*)>/g, (match, attributes) => {
    let safeAttributes = attributes.replace(/\bon\w+\s*=\s*["'][^"']*["']/gi, '')
    safeAttributes = safeAttributes.replace(/href\s*=\s*["']javascript:[^"']*["']/gi, '')
    if (!safeAttributes.match(/target\s*=/i)) {
      safeAttributes += ' target="_blank"'
    }
    if (!safeAttributes.match(/rel\s*=/i)) {
      safeAttributes += ' rel="noopener noreferrer"'
    }
    return `<a ${safeAttributes}>`
  })
  
  renderedMd.value = html
}

// 复制功能（事件委托）
const handleCopyClick = (e) => {
  const button = e.target.closest('.copy-button')
  if (!button) return
  const code = button.getAttribute('data-code')
  if (!code) return

  navigator.clipboard.writeText(code).then(() => {
    const originalText = button.innerHTML
    button.innerHTML = '<i class="bi bi-check"></i><span class="copy-text">已复制</span>'
    button.classList.add('copied')
    
    setTimeout(() => {
      button.innerHTML = originalText
      button.classList.remove('copied')
    }, 2000)
  }).catch(err => {
    console.error('复制失败:', err)
  })
}

// Fancybox 绑定
const bindFancybox = () => {
  if (window.Fancybox) {
    Fancybox.unbind("[data-fancybox]")
    Fancybox.bind("[data-fancybox]", {
      Hash: false,
      Thumbs: { autoStart: false }
    })
  }
}

// 监听 + 生命周期
watch(() => props.modelValue, (newVal) => {
  renderMarkdown(newVal)
  nextTick(() => bindFancybox())
}, { immediate: true })

onMounted(() => {
  nextTick(() => bindFancybox())
})
</script>

<style>
/* 代码块容器样式 */
.code-block-container {
  margin-bottom: 1.2rem;
  border-radius: var(--wx-radius-md);
  border: 1px solid var(--bs-border-color);
  overflow: hidden;
  box-shadow: var(--wx-shadow-sm);
}

/* 代码块头部样式 */
.code-block-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.5rem 1rem;
  background-color: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
}

/* 代码语言标签 */
.code-language {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--bs-secondary-color);
}

/* 复制按钮样式 */
.copy-button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.25rem 0.75rem;
  background-color: transparent;
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  cursor: pointer;
  transition: var(--wx-transition);
}

.copy-button:hover {
  background-color: var(--bs-tertiary-bg);
  border-color: var(--bs-primary);
  color: var(--bs-primary);
}

.copy-button:active {
  transform: scale(0.95);
}

.copy-button.copied {
  background-color: var(--bs-success);
  border-color: var(--bs-success);
  color: white;
}

/* 代码块样式 */
pre {
  margin: 0;
  border-radius: 0;
  padding: 1rem;
  overflow-x: auto;
  background-color: #282c34;
  border: none;
}

/* 行内代码样式 */
code:not(pre code) {
  background-color: var(--bs-tertiary-bg);
  padding: 0.15rem 0.3rem;
  border-radius: var(--wx-radius-sm);
  font-size: 0.95em;
  color: var(--bs-body-color);
}

/* 适配深色模式下行内代码 */
[data-bs-theme="dark"] code:not(pre code) {
  background-color: rgba(var(--bs-primary-rgb), 0.12);
  color: var(--bs-body-color);
}
</style>