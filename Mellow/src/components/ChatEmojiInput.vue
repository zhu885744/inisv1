<template>
  <div
    ref="editorRef"
    class="chat-emoji-input"
    :class="{ 'is-empty': isEmpty, 'is-disabled': disabled }"
    :contenteditable="disabled ? 'false' : 'true'"
    :data-placeholder="placeholder"
    :style="editorStyle"
    role="textbox"
    aria-multiline="true"
    @input="onInput"
    @keydown="onKeydown"
    @beforeinput="onBeforeInput"
    @paste="onPaste"
    @compositionstart="composing = true"
    @compositionend="onCompositionEnd"
    @focus="emit('focus')"
    @blur="emit('blur')"
  />
</template>

<script setup>
/**
 * ChatEmojiInput - 聊天输入框（contenteditable）
 *
 * 为什么不用 <textarea>：textarea 只能显示纯文本，插入表情后用户看到的是一长串
 * `[emoji:/assets/emoji/tiktok/bomb.png]` 短代码。这里用 contenteditable 承载编辑，
 * 存储格式仍然是纯文本 `[emoji:url]`（与评论 / 动态同一套），但编辑时渲染成 <img>，
 * 发送时再把 <img> 还原成短代码 —— 后端与渲染层完全不用改。
 *
 * 与 textarea 对齐的行为：
 *   - v-model 仍是一条字符串；输入法组字期间不误发；
 *   - Enter 发送、Shift+Enter 换行（Enter 的选择权交给父组件，见 emit('send')）；
 *   - 粘贴图片 → 抛给父组件上传，粘贴文本 → 只保留纯文本；
 *   - 随内容自动长高（maxHeight 由父组件给，配合 overflow-y:auto）。
 *
 * 被暴露给父组件的方法（用于 @ 提及这类需要「光标上下文」的场景）：
 *   getText / textBeforeCaret / replaceBeforeCaret / insertText / insertEmoji / focus / clear
 */

import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { escapeHtml, safeEmojiUrl } from '@/utils/emoji'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  // 0 = 不限制
  maxlength: { type: Number, default: 0 },
  maxHeight: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue', 'input', 'keydown', 'send', 'files', 'focus', 'blur'])

const editorRef = ref(null)
const isEmpty = ref(!props.modelValue)
// 输入法组字中：这时的回车是「选字」，不能当发送
const composing = ref(false)

const editorStyle = computed(() => (props.maxHeight ? { maxHeight: props.maxHeight } : null))

/* ---------------- 纯文本 ⇄ 编辑器内容 ---------------- */

const EMOJI_TOKEN = /\[emoji:\s*([^\]]+)\]/g

/** 编辑器 DOM → 纯文本（表情节点还原成 [emoji:url]，块级元素与 <br> 还原成换行） */
function serializeNode(node) {
  let out = ''

  node.childNodes.forEach((child) => {
    if (child.nodeType === Node.TEXT_NODE) {
      out += child.textContent.replace(/\u00a0/g, ' ')
      return
    }
    if (child.nodeType !== Node.ELEMENT_NODE) return

    // 表情是行内 <span data-emoji>（兼容早期直接插 <img> 的内容）
    if (child.hasAttribute?.('data-emoji')) {
      out += `[emoji:${child.getAttribute('data-emoji') || ''}]`
      return
    }

    const tag = child.nodeName
    if (tag === 'BR') {
      out += '\n'
      return
    }

    // 浏览器换行会包一层 div（部分场景是 p / li），进入块级前补一个换行
    const isBlock = tag === 'DIV' || tag === 'P' || tag === 'LI'
    if (isBlock && out && !out.endsWith('\n')) out += '\n'
    out += serializeNode(child)
  })

  return out
}

/**
 * 纯文本 → 编辑器 HTML
 *
 * ⚠ 必须先整体转义再替换表情：正文是用户输入的，直接拼 innerHTML 等于给了 XSS 入口。
 */
function contentToHtml(content) {
  let html = escapeHtml(String(content || ''))
  html = html.replace(EMOJI_TOKEN, (match, url) => {
    const safe = safeEmojiUrl(url)
    // 非法地址原样保留，当作普通文本显示
    return safe ? buildEmojiNode(safe).outerHTML : match
  })
  return html.replace(/\n/g, '<br>')
}

/**
 * 表情在编辑器里的 DOM 结构
 *
 * 用 contenteditable="false" 的行内 <span> 包住 <img>：
 *   - 光标不会钻进图片里面，退格一次删掉整个表情；
 *   - 是明确的行内盒子，不会被浏览器的编辑管线包成块级元素（那会导致一个表情占一行）。
 * 恢复历史内容（contentToHtml）与实时插入（insertEmoji）共用这一份结构，避免两边不一致。
 */
function buildEmojiNode(url) {
  const span = document.createElement('span')
  span.className = 'chat-emoji'
  span.setAttribute('data-emoji', url)
  span.setAttribute('contenteditable', 'false')

  const img = document.createElement('img')
  img.src = url
  img.alt = ''
  span.appendChild(img)

  return span
}

function getText() {
  return editorRef.value ? serializeNode(editorRef.value) : ''
}

function syncContent() {
  const el = editorRef.value
  if (!el) return
  el.innerHTML = contentToHtml(props.modelValue || '')
  isEmpty.value = !(props.modelValue || '')
}

/* ---------------- 光标 ---------------- */

function currentRange() {
  const el = editorRef.value
  const selection = window.getSelection()
  if (!el || !selection || selection.rangeCount === 0) return null
  const range = selection.getRangeAt(0)
  if (!el.contains(range.startContainer)) return null
  return range
}

/**
 * 记住最后一次落在编辑器里的选区
 *
 * 点表情面板 / @ 候选按钮时焦点会离开编辑器，window.getSelection() 就不一定还在正文里；
 * 没有这份记录的话「在句子中间插入表情」会退化成插到末尾。
 */
let savedRange = null

function onSelectionChange() {
  const el = editorRef.value
  const selection = window.getSelection()
  if (!el || !selection || selection.rangeCount === 0) return
  const range = selection.getRangeAt(0)
  if (el.contains(range.startContainer)) {
    savedRange = range.cloneRange()
  }
}

/** 当前可用的光标位置：优先实时选区，其次最近一次记录 */
function activeRange() {
  const live = currentRange()
  if (live) return live
  const el = editorRef.value
  if (savedRange && el?.contains(savedRange.startContainer)) return savedRange
  return null
}

/** 把光标放回编辑器并返回可用 Range：有记录就还原，没有就落到末尾 */
function focusAtCaret() {
  const el = editorRef.value
  if (!el) return null

  // 点进空的 contenteditable 时浏览器会塞一个 <div><br></div>，
  // 不清理的话它会变成插入内容前面的一条空行
  if (!el.textContent && !el.querySelector('[data-emoji]')) el.innerHTML = ''

  // 先取光标再 focus：focus() 可能把选区重置到开头，取晚了就会插到错误位置
  const range = activeRange()
  el.focus()

  const selection = window.getSelection()
  if (!selection) return null

  selection.removeAllRanges()
  if (range) selection.addRange(range)
  else setCaretAtEnd()

  return currentRange() || activeRange()
}

/** 把光标放到内容末尾（输入框还没聚焦过时用） */
function setCaretAtEnd() {
  const el = editorRef.value
  if (!el) return
  const range = document.createRange()
  range.selectNodeContents(el)
  range.collapse(false)
  const selection = window.getSelection()
  if (!selection) return
  selection.removeAllRanges()
  selection.addRange(range)
}

/** 光标之前的纯文本（@ 提及就是靠它判断「光标前是不是 @关键词」） */
function textBeforeCaret() {
  const el = editorRef.value
  if (!el) return ''
  const range = activeRange()
  if (!range) return getText()

  const pre = document.createRange()
  pre.selectNodeContents(el)
  pre.setEnd(range.endContainer, range.endOffset)
  return serializeNode(pre.cloneContents())
}

function onInput() {
  const text = getText()
  isEmpty.value = !text
  emit('update:modelValue', text)
  emit('input', text)
}

/* ---------------- 插入 ---------------- */

function focus() {
  editorRef.value?.focus()
}

/** 在光标处插入纯文本（自动兜底到末尾） */
function insertText(text) {
  if (!editorRef.value) return
  if (!focusAtCaret()) return
  // execCommand 会正常触发 input 事件与撤销栈，比手动拼 DOM 可靠
  document.execCommand('insertText', false, String(text ?? ''))
  onInput()
}

/**
 * 在光标处插入表情（存储格式仍是 [emoji:url]）
 *
 * 这里手动插 DOM 而不用 execCommand('insertHTML')：后者会把插入内容交给浏览器的
 * 编辑管线，在空编辑器 / 块级光标位置会给每个表情套一层 <div>，
 * 结果就是「明明没换行，表情却各自占一行」。
 */
function insertEmoji(url) {
  const safe = safeEmojiUrl(url)
  if (!safe || !editorRef.value) return

  const range = focusAtCaret()
  if (!range) return

  range.deleteContents()

  // 表情后面跟一个空格：给光标落脚，连续插入也不会挤在一起
  const space = document.createTextNode('\u00a0')

  const frag = document.createDocumentFragment()
  frag.appendChild(buildEmojiNode(safe))
  frag.appendChild(space)
  range.insertNode(frag)

  // 光标挪到空格后面，接着输入就是普通文字
  const next = document.createRange()
  next.setStartAfter(space)
  next.collapse(true)
  const selection = window.getSelection()
  if (selection) {
    selection.removeAllRanges()
    selection.addRange(next)
    savedRange = next.cloneRange()
  }

  onInput()
}

/** 删掉光标前 length 个字符并插入 text（@ 提及替换「@关键词」用） */
function replaceBeforeCaret(length, text) {
  if (!editorRef.value) return
  if (!focusAtCaret()) return

  const count = Math.max(0, Number(length) || 0)
  for (let i = 0; i < count; i += 1) document.execCommand('delete')
  document.execCommand('insertText', false, String(text ?? ''))
  onInput()
}

function clear() {
  const el = editorRef.value
  if (el) el.innerHTML = ''
  isEmpty.value = true
  emit('update:modelValue', '')
}

/* ---------------- 键盘 / 粘贴 ---------------- */

function onKeydown(event) {
  // 交给父组件先处理（@ 候选面板的上下键 / 回车选中）
  emit('keydown', event)
  if (event.defaultPrevented) return

  if (event.key === 'Enter' && !event.shiftKey && !event.isComposing && !composing.value) {
    event.preventDefault()
    emit('send')
  }
}

function onCompositionEnd() {
  composing.value = false
  // 组字结束后 textarea 会自动触发 input，contenteditable 也补一次，保证 v-model 同步
  onInput()
}

function onBeforeInput(event) {
  if (!props.maxlength) return
  if (event.inputType !== 'insertText' && event.inputType !== 'insertFromPaste') return
  if (getText().length >= props.maxlength) event.preventDefault()
}

function onPaste(event) {
  const files = Array.from(event.clipboardData?.items || [])
    .filter((item) => item.kind === 'file' && String(item.type).startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter(Boolean)

  if (files.length) {
    event.preventDefault()
    emit('files', files)
    return
  }

  // 只收纯文本：否则从网页复制会带一堆样式标签进来
  event.preventDefault()
  const text = event.clipboardData?.getData('text/plain') || ''
  if (text) insertText(text.replace(/\r\n?/g, '\n'))
}

/* ---------------- 同步 ---------------- */

watch(
  () => props.modelValue,
  (val) => {
    if (!editorRef.value) return
    // 内容一致就不动 DOM，避免光标跳动
    if ((val || '') === getText()) return
    syncContent()
  }
)

onMounted(() => {
  syncContent()
  document.addEventListener('selectionchange', onSelectionChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('selectionchange', onSelectionChange)
  savedRange = null
})

defineExpose({
  focus,
  clear,
  getText,
  textBeforeCaret,
  replaceBeforeCaret,
  insertText,
  insertEmoji,
  setCaretAtEnd
})
</script>

<style scoped>
.chat-emoji-input {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
  outline: none;
  cursor: text;
}
.chat-emoji-input.is-empty::before {
  content: attr(data-placeholder);
  color: var(--text-light);
  pointer-events: none;
}
.chat-emoji-input.is-disabled {
  cursor: not-allowed;
  opacity: 0.7;
}
/* 编辑态里的表情：行内盒子 + 与文字基线对齐，尺寸跟一行文字相当。
   必须是 inline-block（而不是让 <img> 直接裸排），否则块级化后一个表情占一行 */
.chat-emoji-input :deep(.chat-emoji) {
  display: inline-block;
  width: 18px;
  height: 18px;
  margin: 0 1px;
  vertical-align: -3px;
  line-height: 1;
  user-select: none;
}
.chat-emoji-input :deep(.chat-emoji img) {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
  pointer-events: none;
}
</style>
