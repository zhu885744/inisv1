<template>
  <div class="emoji-editor">
    <!-- 富文本输入框（contenteditable） -->
    <!-- 输入区 + 工具栏放同一个相对定位容器里：
         工具栏的 absolute 锚点必须是「输入区」，不能是整个 .emoji-editor ——
         内联展开表情面板后 .emoji-editor 会包含面板，工具栏就会浮到面板上把表情挡住 -->
    <div class="emoji-editor-field">
      <div
        ref="editorRef"
        class="editor-content"
        :class="{ 'is-empty': !modelValue, 'no-toolbar': hideToolbar }"
        contenteditable="true"
        :data-placeholder="placeholder"
        :style="contentStyle"
        @input="onInput"
        @blur="onInput"
        @keydown="onKeydown"
      ></div>

      <!-- 工具栏（浮于输入区底部，输入区已用 padding-bottom 为它留位）
           hideToolbar：调用方要自己画工具栏时（如发布动态的图标工具栏）隐藏它,
           并同时去掉输入区下方那 40px 的留白 -->
      <div v-if="!hideToolbar" ref="toolbarRef" class="editor-toolbar">
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :class="{ 'is-active': showPicker }"
          @click="togglePicker"
          title="插入表情"
        >😊 表情</button>
        <!-- @ 提及：点开候选（我关注的人），选中后插入「@昵称」，不需要手动敲 @ -->
        <button
          v-if="mentionEnabled"
          type="button"
          class="btn btn-secondary btn-sm"
          :class="{ 'is-active': mentionOpen }"
          title="@ 提及用户"
          @mousedown.prevent
          @click="openMention"
        ><i class="bi bi-at" />提及用户</button>
        <slot name="extra" />
      </div>
    </div>

    <!-- 表情选择面板 -->
    <EmojiPicker
      v-if="showPicker"
      v-model="showPicker"
      :inline="inlinePicker"
      @select="insertEmoji"
    />

    <!-- @ 用户选择面板（候选只含「我关注的人」，见 api/mention.js） -->
    <MentionPicker
      v-if="mentionOpen"
      ref="mentionRef"
      :keyword="mentionKeyword"
      :position="mentionPosition"
      @select="applyMention"
    />

    <!-- # 话题选择面板（候选来自 topics/candidates，需登录） -->
    <MentionPicker
      v-if="topicOpen"
      ref="topicRef"
      :items="topicItems"
      name-key="name"
      title="选择话题"
      empty-text="还没有话题，先写一个 #话题# 试试"
      keyword-empty-text="没有匹配的话题"
      :keyword="topicKeyword"
      :position="topicPosition"
      :loading="topicLoading"
      @select="applyTopic"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import EmojiPicker from './EmojiPicker.vue'
import MentionPicker from './MentionPicker.vue'
import { getTopicCandidates } from '@/api/topics'
import { escapeHtml, safeEmojiUrl } from '@/utils/emoji'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '说点什么吧...' },
  // 内联表情面板（弹窗等需要自适应高度的场景设为 true）
  inlinePicker: { type: Boolean, default: false },
  // 输入区最大高度：默认与评论框一致（超过就在编辑器内滚动）。
  // 发布弹窗这类「容器随内容长高」的场景传 'none'，把滚动交给外层容器。
  maxHeight: { type: String, default: '300px' },
  // 隐藏内置工具栏（调用方自己画）：同时会把输入区底部为工具栏预留的 40px 去掉
  hideToolbar: { type: Boolean, default: false },
  // 是否提供 @ 提及入口（内置工具栏按钮 + openMention 方法）
  // 默认开启；未登录场景（如未登录时的评论引导）可以关掉
  mentionEnabled: { type: Boolean, default: true }
})

const emit = defineEmits(['update:modelValue'])

const editorRef = ref(null)
const showPicker = ref(false)

/* ---------------- 工具栏高度 → 输入区留白 ----------------
 * 工具栏是 absolute 浮在输入区底部的，原来输入区只固定留 40px。
 * 窄屏（手机 / 400px 左右的容器）工具栏放不下会换行成两三层，高度超过 40px 后
 * 就直接盖住输入区，导致「看不到光标、点不进去」。这里实测工具栏高度，
 * 把它换算成输入区的 padding-bottom 与 min-height，换几行都不会压到正文。
 */
const toolbarRef = ref(null)
const toolbarHeight = ref(0)
let toolbarObserver = null

const contentStyle = computed(() => {
  const style = { maxHeight: props.maxHeight }
  // 调用方自画工具栏（如发布动态）：保持 CSS 里的 10px 留白即可
  if (props.hideToolbar) return style
  const h = toolbarHeight.value
  if (!h) return style

  // 8px 底边距 + 8px 与正文的间隔
  const paddingBottom = h + 16
  style.paddingBottom = `${paddingBottom}px`
  // 高度兜底 = 工具栏留白 + 上内边距(10) + 正文至少 48px（约两行），
  // 否则窄屏下工具栏换行后仍会把正文区挤成一条缝
  style.minHeight = `${paddingBottom + 58}px`
  return style
})

// 工具栏会随宽度换行（v-if 也可能切换），所以跟着 ref 重新 observe
watch(toolbarRef, (el) => {
  toolbarObserver?.disconnect()
  if (!el) {
    toolbarHeight.value = 0
    return
  }
  toolbarObserver?.observe(el)
})

// 将存储格式内容转为 HTML（[emoji:url] -> img）
//
// ⚠ 安全：结果会被赋给编辑器的 innerHTML，而正文是用户输入（动态正文后端没有 XSS 过滤）。
// 必须**先整体转义**再替换表情 —— 否则打开（或在后台编辑）一条含
// `<img src=x onerror=...>` 的动态就会执行脚本，管理员编辑别人的动态等于被提权。
function contentToHtml(content) {
  if (!content) return ''
  let html = escapeHtml(String(content))
  html = html.replace(/\[emoji:\s*([^\]]+)\]/g, (match, url) => {
    // 与 renderEmoji 共用同一个地址白名单（http(s) / 站内相对路径 + 已转义）
    const safe = safeEmojiUrl(url)
    if (!safe) return match
    return `<img src="${safe}" data-emoji="${safe}" class="editor-emoji" style="width:24px;height:24px;vertical-align:middle;display:inline-block;object-fit:contain;margin:0 2px;">`
  })
  html = html.replace(/\n/g, '<br>')
  return html
}

// 从 contenteditable 提取纯文本，emoji 图片转回 [emoji:url]
function getEditorContent(el) {
  if (!el) return ''
  const clone = el.cloneNode(true)
  const imgs = clone.querySelectorAll('img[data-emoji]')
  imgs.forEach((img) => {
    const url = img.getAttribute('data-emoji') || img.src || ''
    const textNode = document.createTextNode(`[emoji:${url}]`)
    img.parentNode.replaceChild(textNode, img)
  })
  return clone.innerText.replace(/\u00a0/g, ' ').trim()
}

function onInput() {
  const text = getEditorContent(editorRef.value)
  emit('update:modelValue', text)
  syncPickerState()
}

/* ---------------- @ 提及 / # 话题 候选面板 ---------------- */

const mentionRef = ref(null)
const mentionOpen = ref(false)
const mentionKeyword = ref('')
const mentionPosition = ref({ left: 0, top: 0, flipTo: null })
// 工具栏按钮打开面板（true）：光标前没有「@关键词」可替换，选中后直接插到保存的光标处
const mentionFromButton = ref(false)
const savedRange = ref(null)

const topicRef = ref(null)
const topicOpen = ref(false)
const topicKeyword = ref('')
const topicPosition = ref({ left: 0, top: 0, flipTo: null })
const topicItems = ref([])
const topicLoading = ref(false)

// @ 提及：@ 前必须是行首或空白，避免把邮箱 a@b.com 当提及
const MENTION_RE = /(?:^|\s)@([^\s@]{0,32})$/
// # 话题：不加左边界（中文里「文字#话题」很常见），关键词里不允许再出现 # 与空白
const TOPIC_RE = /#([^#\s@]{0,32})$/

/** 光标矩形；空编辑器 / range 不在文档里时返回 null（Chrome 会给出全 0 的矩形） */
function caretRectOf(range) {
  if (!range) return null

  const rects = range.getClientRects()
  if (rects.length) return rects[rects.length - 1]

  const box = range.getBoundingClientRect()
  // 空 contenteditable 的 range 是 0x0 且坐标全 0，拿来定位会让面板跑到视口左上角
  if (box && (box.width || box.height || box.top || box.left)) return box
  return null
}

/** 矩形 → 面板坐标（视口坐标）；下方空间不足（面板最高 236 + 间距）时改为向上展开 */
function panelStyleByRect(rect) {
  const gap = 6
  const below = rect.bottom + gap

  return {
    left: rect.left,
    top: below,
    flipTo: below + 246 > window.innerHeight ? rect.top - gap : null
  }
}

/**
 * 光标位置 → 面板坐标（视口坐标）
 *
 * MentionPicker 是 position: fixed，直接取光标矩形即可，不做「相对容器」的换算 ——
 * 那种算法要扣除编辑器内边距与祖先偏移，容器层级一变就会整体错位。
 * 取不到光标矩形（编辑器为空、刚点开弹窗还没落光标）时退回编辑器自身矩形。
 */
function caretPanelPosition(range) {
  const caret = caretRectOf(range)
  if (caret) return panelStyleByRect(caret)

  const box = editorRef.value?.getBoundingClientRect?.()
  if (box) {
    // 退化成「内容区左上角第一行」，与落光标后的观感一致
    return panelStyleByRect({
      left: box.left + 12,
      top: box.top + 10,
      bottom: box.top + 34
    })
  }
  return { left: 8, top: 8, flipTo: null }
}

/** 取回话题候选（懒加载：第一次输入 # 时才请求；接口需要登录，失败就退化为空列表） */
async function ensureTopicItems() {
  if (topicItems.value.length || topicLoading.value) return
  topicLoading.value = true
  try {
    const res = await getTopicCandidates({ limit: 200 })
    topicItems.value = res?.data || []
  } catch {
    topicItems.value = []
  } finally {
    topicLoading.value = false
  }
}

/**
 * 按光标前的文本决定开哪个候选面板
 *
 * - 「@关键词」→ @提及（只匹配已关注的人，见 app/model/mention.go）
 * - 「#关键词」→ 话题（保存时规范化成 #[话题名](/topic/12)，不存在的话题自动创建）
 *
 * 两者同时命中时（如 "@a#b"）取离光标更近的触发符，避免两个面板打架。
 */
function syncPickerState() {
  const editor = editorRef.value
  const selection = window.getSelection()

  const close = () => {
    mentionOpen.value = false
    topicOpen.value = false
    mentionFromButton.value = false
  }

  if (!editor || !selection || selection.rangeCount === 0) {
    close()
    return
  }

  const range = selection.getRangeAt(0)
  const node = range.startContainer

  if (!range.collapsed || !editor.contains(node) || node.nodeType !== Node.TEXT_NODE) {
    close()
    return
  }

  const before = node.textContent.slice(0, range.startOffset)
  const mention = before.match(MENTION_RE)
  const topic = before.match(TOPIC_RE)

  if (topic && (!mention || before.lastIndexOf('#') > before.lastIndexOf('@'))) {
    mentionOpen.value = false
    topicKeyword.value = topic[1]
    topicOpen.value = true
    topicPosition.value = caretPanelPosition(range)
    ensureTopicItems()
    return
  }

  topicOpen.value = false

  if (!mention) {
    mentionOpen.value = false
    return
  }

  // 手打 @ 触发的联想：选中后要连「@关键词」一起替换
  mentionFromButton.value = false
  mentionKeyword.value = mention[1]
  mentionOpen.value = true
  mentionPosition.value = caretPanelPosition(range)
}

// 上/下键移动、回车选中、Esc 关闭（焦点在 contenteditable 里，事件不会落到面板上，所以在这里转发）
// 同一时刻只会有一个候选面板（见 syncPickerState）
function onKeydown(event) {
  if (!mentionOpen.value && !topicOpen.value) return

  const picker = topicOpen.value ? topicRef.value : mentionRef.value
  if (!picker) return

  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    picker.move(event.key === 'ArrowDown' ? 1 : -1)
    return
  }

  if (event.key === 'Enter') {
    if (picker.selectActive()) event.preventDefault()
    return
  }

  if (event.key === 'Escape') {
    event.preventDefault()
    mentionOpen.value = false
    topicOpen.value = false
  }
}

/**
 * 工具栏按钮打开 @ 提及面板（不需要用户先手打 @）
 *
 * 与 insertTopic 的区别：这里不往正文插 @ 字符，而是记住当前光标，
 * 选中候选后把「@昵称 」直接插到该位置 —— 否则在「文字」后面点按钮时，
 * 插入的 @ 前面不是空白，MENTION_RE 匹配不到，面板根本不会弹。
 */
function openMention(event) {
  const editor = editorRef.value
  if (!editor) return

  editor.focus()

  let range = null
  const selection = window.getSelection()
  if (selection && selection.rangeCount > 0) {
    const current = selection.getRangeAt(0)
    if (editor.contains(current.startContainer)) range = current.cloneRange()
  }
  if (!range) {
    // 编辑器为空 / 还没拿到选区：插到内容末尾
    range = document.createRange()
    range.selectNodeContents(editor)
    range.collapse(false)
  }

  savedRange.value = range
  mentionFromButton.value = true
  mentionKeyword.value = ''
  topicOpen.value = false

  // 定位顺序：光标 → @ 按钮 → 编辑器。编辑器为空时光标没有矩形，
  // 直接贴按钮展开，避免面板跑到视口左上角。
  const caretRect = caretRectOf(range)
  const buttonRect = event?.currentTarget?.getBoundingClientRect?.()
  if (caretRect) {
    mentionPosition.value = panelStyleByRect(caretRect)
  } else if (buttonRect && (buttonRect.width || buttonRect.height)) {
    mentionPosition.value = panelStyleByRect(buttonRect)
  } else {
    mentionPosition.value = caretPanelPosition(range)
  }

  mentionOpen.value = true
}

// 选中候选：把已输入的 @关键词 替换成 @昵称 + 空格（按钮触发时直接插到光标处）
function applyMention(user) {
  const keyword = mentionKeyword.value
  const fromButton = mentionFromButton.value
  mentionOpen.value = false
  mentionFromButton.value = false

  const editor = editorRef.value
  if (!editor || !user?.nickname) return

  editor.focus()
  const selection = window.getSelection()
  if (!selection) return

  if (fromButton) {
    if (savedRange.value) {
      selection.removeAllRanges()
      selection.addRange(savedRange.value)
    }
    savedRange.value = null
    document.execCommand('insertText', false, `@${user.nickname} `)
    onInput()
    return
  }

  if (selection.rangeCount === 0) return

  const range = selection.getRangeAt(0)
  const node = range.startContainer

  // 先把「@关键词」整段选中，下面的 execCommand 会用 @昵称 覆盖它
  if (node.nodeType === Node.TEXT_NODE) {
    const start = Math.max(0, range.startOffset - keyword.length - 1)
    const replaceRange = document.createRange()
    replaceRange.setStart(node, start)
    replaceRange.setEnd(node, range.startOffset)
    selection.removeAllRanges()
    selection.addRange(replaceRange)
  }

  // execCommand 插入纯文本：会正常触发 input 事件与撤销栈，比手动拼 DOM 可靠
  document.execCommand('insertText', false, `@${user.nickname} `)
  syncPickerState()
  onInput()
}

// 选中话题候选：把已输入的「#关键词」替换成 #话题名#（后端保存时会规范化成链接）
function applyTopic(topic) {
  const keyword = topicKeyword.value
  topicOpen.value = false

  const editor = editorRef.value
  if (!editor || !topic?.name) return

  editor.focus()
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) return

  const range = selection.getRangeAt(0)
  const node = range.startContainer

  // 先把「#关键词」整段选中，下面的 execCommand 会用 #话题名# 覆盖它
  if (node.nodeType === Node.TEXT_NODE) {
    const start = Math.max(0, range.startOffset - keyword.length - 1)
    const replaceRange = document.createRange()
    replaceRange.setStart(node, start)
    replaceRange.setEnd(node, range.startOffset)
    selection.removeAllRanges()
    selection.addRange(replaceRange)
  }

  document.execCommand('insertText', false, `#${topic.name}# `)
  onInput()
}

// 在光标位置插入 HTML
function insertHTMLAtCursor(html) {
  const editor = editorRef.value
  if (!editor) return
  editor.focus()
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) {
    editor.innerHTML += html
    return
  }
  const range = selection.getRangeAt(0)
  range.deleteContents()
  const temp = document.createElement('div')
  temp.innerHTML = html
  const frag = document.createDocumentFragment()
  let lastNode = null
  while (temp.firstChild) {
    lastNode = temp.firstChild
    frag.appendChild(temp.firstChild)
  }
  range.insertNode(frag)
  if (lastNode) {
    range.setStartAfter(lastNode)
    range.collapse(true)
  }
  selection.removeAllRanges()
  selection.addRange(range)
}

// 插入表情
function insertEmoji(emoji) {
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width:24px;height:24px;vertical-align:middle;display:inline-block;object-fit:contain;margin:0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml)
  } else {
    insertHTMLAtCursor(emoji + ' ')
  }
  onInput()
}

function togglePicker() {
  showPicker.value = !showPicker.value
}

// 监听外部 modelValue 变化（如重置），同步编辑器内容
watch(
  () => props.modelValue,
  (val) => {
    if (editorRef.value) {
      const currentText = getEditorContent(editorRef.value)
      // 仅当内容不一致时更新，避免光标跳动
      if (val !== currentText) {
        editorRef.value.innerHTML = contentToHtml(val || '')
      }
    }
  }
)

onMounted(() => {
  nextTick(() => {
    if (editorRef.value) {
      editorRef.value.innerHTML = contentToHtml(props.modelValue || '')
    }
  })

  // 输入区留白依赖工具栏实际高度（窄屏会换行），所以挂了就量一次并持续跟随
  if (typeof ResizeObserver !== 'undefined') {
    toolbarObserver = new ResizeObserver((entries) => {
      // 用 offsetHeight：contentRect 不含 padding/border，会比实际矮几像素，
      // 窄屏换行后这点误差正好会让工具栏压住最后一行文字
      const el = entries[0]?.target
      toolbarHeight.value = el ? Math.ceil(el.offsetHeight) : 0
    })
    if (toolbarRef.value) toolbarObserver.observe(toolbarRef.value)
  }
})

onBeforeUnmount(() => {
  toolbarObserver?.disconnect()
  toolbarObserver = null
})

// 暴露清空方法
function clear() {
  if (editorRef.value) editorRef.value.innerHTML = ''
  emit('update:modelValue', '')
}

// openEmoji：供外部工具栏（如发布动态的图标工具栏）调用 —— 开关表情面板
/**
 * 在光标处插入一个「#」并弹出话题候选面板
 *
 * 选中候选后会补全成 #话题名#，后端保存时规范化成 #[话题名](/topic/12)（见 app/model/topics.go）；
 * 也可以不用面板，直接手写 #话题#。
 */
function insertTopic() {
  const editor = editorRef.value
  if (!editor) return

  editor.focus()
  // execCommand 插入纯文本：会正常触发 input 事件与撤销栈，和 @提及 的做法一致
  document.execCommand('insertText', false, '#')
  // onInput 里会走 syncPickerState，光标前是「#」时自动弹出候选面板
  onInput()
}

defineExpose({
  clear,
  focus: () => editorRef.value?.focus(),
  openEmoji: togglePicker,
  insertTopic,
  openMention
})
</script>

<style scoped>
.emoji-editor {
  width: 100%;
  position: relative;
}
/* 工具栏的定位基准：只包住输入区（不含下方的表情/@ 面板） */
.emoji-editor-field {
  position: relative;
}
.editor-content {
  min-height: 100px;
  /* 最大高度由 maxHeight 属性控制（内联样式）；默认 300px 与评论框一致，
     发布弹窗传 'none' 让输入区随内容长高 */
  overflow-y: auto;
  padding: 10px 12px 40px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  word-break: break-word;
  cursor: text;
  transition: border-color 0.2s, box-shadow 0.2s;
}
/* 外部自画工具栏时：去掉为浮动工具栏预留的底部留白 */
.editor-content.no-toolbar {
  padding-bottom: 10px;
}
.editor-content:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
  outline: none;
}
.editor-content.is-empty::before {
  content: attr(data-placeholder);
  color: var(--text-light);
  pointer-events: none;
}
.editor-toolbar {
  position: absolute;
  left: 12px;
  /* 不写 right 时宽度由内容撑开，窄屏会横向溢出；写死右边距让按钮在容器内换行 */
  right: 12px;
  bottom: 8px;
  z-index: 1;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 4px 6px;
  background: var(--bg-card);
  border-radius: var(--radius-sm);
}
/* 窄屏：按钮间距收紧一点，少换一行、少一点遮挡 */
@media (max-width: 480px) {
  .editor-toolbar {
    left: 8px;
    right: 8px;
    gap: 6px;
    padding: 4px;
  }
}

</style>
