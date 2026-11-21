<template>
  <div class="markdown-editor" :class="{ 'is-fullscreen': fullscreen }">
    <EditorToolbar
      ref="toolbarRef"
      :tools="tools"
      :mode="mode"
      :split-view="splitView"
      :fullscreen="fullscreen"
      :uploading="uploading"
      :media-uploading="mediaUploading"
      :live-uploading="liveUploading"
      @select-tool="runTool($event.apply)"
      @insert-emoji="insertEmoji"
      @toggle-mention="onMentionToggle"
      @pick-image="pickImage"
      @open-library="showLibrary = true"
      @open-media-library="openMediaLibrary"
      @pick-media="pickMedia"
      @pick-live="pickLive"
      @toggle-mode="toggleMode"
      @toggle-split="toggleSplit"
      @toggle-fullscreen="toggleFullscreen()"
    />

    <!-- @ 提及候选（position: fixed，DOM 位置不影响定位） -->
    <MentionPicker
      v-if="mentionOpen"
      ref="mentionRef"
      :keyword="mentionKeyword"
      :position="mentionPosition"
      @select="applyMention"
    />

    <!-- 本地草稿恢复提示：只在「本地内容与当前正文不同」时出现 -->
    <div v-if="draftPrompt" class="markdown-editor-draft">
      <span class="markdown-editor-draft-text">
        <i class="bi bi-clock-history" aria-hidden="true" />
        发现 {{ draftPrompt.time }} 的本地草稿，内容与当前正文不一致
      </span>
      <span class="markdown-editor-draft-actions">
        <button type="button" class="btn btn-ghost btn-sm" @click="discardDraft">忽略</button>
        <button type="button" class="btn btn-primary btn-sm" @click="restoreDraft">恢复草稿</button>
      </span>
    </div>

    <!-- 正文：源码编辑 + 预览 -->
    <div class="markdown-editor-body" :class="{ 'is-split': splitView }">
      <!-- 源码编辑：只有 [emoji:url] 渲染成表情图，图片 markdown 与 @[video] 等短代码保持文本 -->
      <div
        v-show="showEditor"
        ref="editorRef"
        class="markdown-editor-content"
        :class="{ 'is-empty': !modelValue }"
        contenteditable="true"
        spellcheck="false"
        :data-placeholder="placeholder"
        :style="{ minHeight: fullscreen ? '0px' : minHeight }"
        @input="onInput"
        @keydown="onKeydown"
        @paste="onPaste"
        @dragover.prevent
        @drop.prevent="onDrop"
      />

      <div
        v-show="showPreview"
        class="markdown-editor-preview markdown-body"
        :style="{ minHeight: fullscreen ? '0px' : minHeight }"
        v-html="html"
      />
    </div>

    <EditorStatusBar
      :count="count"
      :max-length="maxLength"
      :saved-text="draftSavedText"
      :fullscreen="fullscreen"
      @save="emit('save')"
      @exit-fullscreen="toggleFullscreen(false)"
    />

    <input ref="fileRef" type="file" accept="image/*" hidden @change="onFileChange" />

    <!-- 实况图：图片与视频一起选（先选中的图片当封面，第一条视频当实况）；
         也可以只选一张安卓实况照片，视频由后端拆出 -->
    <input ref="liveRef" type="file" accept="image/*,video/*" multiple hidden @change="onLiveChange" />

    <!-- 视频 / 音频 / 附件：实际允许的类型由后端 storage 配置兜底校验 -->
    <input
      ref="mediaRef"
      type="file"
      accept="video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.csv,.zip,.rar,.7z,.tar,.gz,.tgz,.bz2,.tbz2,.xz,.txz,.mkv,.avi,.flv,.wmv,.3gp,.mpg,.mpeg,.ogv"
      multiple
      hidden
      @change="onMediaChange"
    />

    <!-- 附件库（选择已上传图片，按 markdown 图片语法插入光标处） -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择正文图片"
      accept="image"
      multiple
      :max="9"
      @select="onLibrarySelect"
    />

    <!-- 附件库（视频 / 音频 / 任意附件）：共用同一个弹窗，按当前类型过滤 -->
    <AttachmentLibrary
      v-model:visible="showMediaLibrary"
      :title="mediaTitle"
      :accept="mediaAccept"
      multiple
      :max="9"
      @select="onMediaLibrarySelect"
    />
  </div>
</template>

<script setup>
/**
 * 文章 / 页面 / 文档共用的 Markdown 编辑器
 *
 * 组成（重构后）：
 *   components/editor/EditorToolbar.vue   分组工具栏 + 插入面板（纯展示，不碰正文）
 *   components/editor/EditorStatusBar.vue 字数 / 阅读时长 / 草稿状态 / 全屏保存
 *   composables/useEditorMode.js          写作 · 预览 · 分屏 · 全屏
 *   composables/useMentionPanel.js        @ 候选面板的状态与定位
 *   composables/useEditorDraft.js         本地草稿自动保存（防丢稿）
 *
 * 本文件只保留「源码 ↔ DOM」这一块真正的编辑器内核：
 *   contenteditable 里是 Markdown 源码文本（表情与图片被渲染成原子节点），
 *   每次输入序列化回字符串交给 v-model，外部内容变化时整体重建 DOM。
 *
 * 对外契约（ContentWriteForm / DocsWrite 依赖，勿改）：
 *   props  modelValue / placeholder / minHeight / maxLength
 *   emits  update:modelValue / save
 *   expose focus() / clearDraft()
 */
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import EditorToolbar from '@/components/editor/EditorToolbar.vue'
import EditorStatusBar from '@/components/editor/EditorStatusBar.vue'
import MentionPicker from './MentionPicker.vue'
import AttachmentLibrary from './AttachmentLibrary.vue'
import { renderMarkdown } from '@/utils/markdown'
import { getFullUrl } from '@/utils/runtimeConfig'
import { uploadArticleImage } from '@/api/article'
import { uploadAttachments } from '@/api/attachment'
import { attachmentMarker } from '@/utils/attachment'
import { toast } from '@/utils/toast'
import { formatTime } from '@/utils/time'
import { useEditorMode } from '@/composables/useEditorMode'
import { useEditorDraft } from '@/composables/useEditorDraft'
import { useMentionPanel } from '@/composables/useMentionPanel'
import {
  sourceToHtml,
  serializeEditor,
  rangeFromOffsets,
  offsetOfPoint,
  wrapText,
  prefixLines,
  insertTextAt,
  codeBlockText,
  linkText,
  tableText,
  blockMarkerText,
  TAIL_MARKER_RE,
  HAS_MARKER_RE
} from '@/utils/markdownEdit'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '来说点什么吧...' },
  minHeight: { type: String, default: '420px' },
  // 字数展示上限（仅用于状态栏计数，不做硬性截断）
  maxLength: { type: Number, default: 5000 },
  // 本地草稿键：默认按当前路由路径区分；传空串可关闭本地草稿
  draftKey: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue', 'save'])

const editorRef = ref(null)
const toolbarRef = ref(null)
const fileRef = ref(null)
const uploading = ref(false)

const html = computed(() => renderMarkdown(props.modelValue))
const count = computed(() => (props.modelValue || '').replace(/\s/g, '').length)

// ===== 视图模式（写作 / 预览 / 分屏 / 全屏） =====
const { mode, splitView, fullscreen, showEditor, showPreview, toggleMode, toggleSplit, toggleFullscreen } = useEditorMode({
  focus: () => editorRef.value?.focus(),
  // 工具栏面板还开着时，第一次 Esc 只关面板
  shouldIgnoreEscape: () => toolbarRef.value?.isPanelOpen?.() === true
})

// ===== @ 提及 =====
const mentionRef = ref(null)
const {
  open: mentionOpen,
  keyword: mentionKeyword,
  position: mentionPosition,
  syncFromSelection: syncMention,
  toggle: toggleMentionPanel,
  onKeydown: onMentionKeydown,
  apply: applyMention
} = useMentionPanel({ getRoot: () => editorRef.value, panelRef: mentionRef })

/** 工具栏 @ 按钮：切回写作态并贴在按钮下方展开候选 */
function onMentionToggle(rect) {
  if (mode.value === 'preview') mode.value = 'write'
  nextTick(() => toggleMentionPanel(rect))
}

// ===== 本地草稿（防丢稿） =====
const {
  restored: draftRestored,
  lastSavedAt,
  init: initDraft,
  clear: clearDraft,
  consume: consumeDraft,
  discard: discardDraftCache
} = useEditorDraft({
  key: () => props.draftKey || window.location.pathname,
  getValue: () => props.modelValue
})

const draftPrompt = computed(() => {
  const found = draftRestored.value
  if (!found) return null
  return { time: found.updatedAt ? formatTime(Math.floor(found.updatedAt / 1000)) : '刚刚' }
})

const draftSavedText = computed(() => {
  if (!lastSavedAt.value) return ''
  return `本地草稿已存 ${formatTime(Math.floor(lastSavedAt.value / 1000)).slice(11)}`
})

function restoreDraft() {
  const value = consumeDraft()
  if (!value) return
  // 先重建 DOM 再同步给父级：emitModel 会更新 lastValue，父级回传时 watcher 就不再重建了
  renderFrom(value)
  emitModel(value)
  toast.success('已恢复本地草稿')
}

/** 点「忽略」= 删掉本地副本，避免下次打开又提示 */
function discardDraft() {
  discardDraftCache()
}

// ===== 工具栏 =====
const tools = [
  { key: 'bold', title: '加粗 (Ctrl+B)', icon: 'bi bi-type-bold', apply: (v, s, e) => wrapText(v, s, e, '**', '**', '加粗文字') },
  { key: 'italic', title: '斜体 (Ctrl+I)', icon: 'bi bi-type-italic', apply: (v, s, e) => wrapText(v, s, e, '*', '*', '斜体文字') },
  { key: 'strike', title: '删除线', icon: 'bi bi-type-strikethrough', apply: (v, s, e) => wrapText(v, s, e, '~~', '~~', '删除线') },
  { key: 'h2', title: '二级标题', text: 'H2', apply: (v, s, e) => prefixLines(v, s, e, '## ') },
  { key: 'h3', title: '三级标题', text: 'H3', apply: (v, s, e) => prefixLines(v, s, e, '### ') },
  { key: 'quote', title: '引用', icon: 'bi bi-quote', apply: (v, s, e) => prefixLines(v, s, e, '> ') },
  { key: 'ul', title: '无序列表', icon: 'bi bi-list-ul', apply: (v, s, e) => prefixLines(v, s, e, '- ') },
  { key: 'ol', title: '有序列表', icon: 'bi bi-list-ol', apply: (v, s, e) => prefixLines(v, s, e, '1. ') },
  { key: 'code', title: '行内代码', icon: 'bi bi-code', apply: (v, s, e) => wrapText(v, s, e, '`', '`', 'code') },
  { key: 'block', title: '代码块', icon: 'bi bi-code-square', apply: codeBlockText },
  { key: 'hr', title: '分割线', icon: 'bi bi-dash-lg', apply: (v, s, e) => insertTextAt(v, s, e, '\n\n---\n\n') },
  { key: 'link', title: '链接 (Ctrl+K)', icon: 'bi bi-link-45deg', apply: linkText },
  { key: 'table', title: '表格', icon: 'bi bi-table', apply: tableText }
]

// ===== 源码 <-> DOM =====

/** 用源码重建编辑器内容（不动光标） */
function renderFrom(value) {
  if (!editorRef.value) return
  editorRef.value.innerHTML = sourceToHtml(value || '', getFullUrl)
}

/** 读取当前 DOM 的源码，并同步给父组件 */
function syncFromDom() {
  if (!editorRef.value) return ''
  const { text } = serializeEditor(editorRef.value)
  emitModel(text)
  return text
}

function emitModel(value) {
  lastValue = value
  emit('update:modelValue', value)
}

/** 应用一次源码变更，并还原光标 / 选区 */
function applySource(value, selStart, selEnd = selStart) {
  const root = editorRef.value
  if (!root) return
  renderFrom(value)
  emitModel(value)
  nextTick(() => {
    root.focus()
    const { points } = serializeEditor(root)
    const range = rangeFromOffsets(root, points, selStart, selEnd)
    const selection = window.getSelection()
    if (!selection) return
    selection.removeAllRanges()
    selection.addRange(range)
  })
}

/** 当前选区的源码偏移 */
function currentOffsets() {
  const root = editorRef.value
  const selection = window.getSelection()
  if (!root || !selection || !selection.rangeCount) return null
  const range = selection.getRangeAt(0)
  const start = offsetOfPoint(root, range.startContainer, range.startOffset)
  const end = offsetOfPoint(root, range.endContainer, range.endOffset)
  if (start === null || end === null) return null
  return { start, end }
}

// ===== 输入 =====
let lastValue = ''

function onInput(e) {
  const root = editorRef.value
  if (!root) return
  const { text } = serializeEditor(root)
  emitModel(text)
  // 输入 @ / 继续输入关键词时，同步 @ 候选面板
  syncMention()

  // 手动敲完一个表情标记（[emoji:url]）时即时渲染成表情图；图片 / 短代码保持文本
  const data = e?.data || ''
  if (!e?.isComposing && (data === ']' || data === '］')) {
    nextTick(renderCompletedMarker)
  }
}

function renderCompletedMarker() {
  const root = editorRef.value
  if (!root) return
  const offsets = currentOffsets()
  if (!offsets) return
  const { text } = serializeEditor(root)
  const before = text.slice(0, offsets.start)
  const matched = before.match(TAIL_MARKER_RE)
  if (!matched) return
  // 文本不变，但重建一次 DOM 即可把刚输入的标记变成表情图
  applySource(text, offsets.end)
}

/** 外部（加载 / 重置 / 恢复草稿）变更内容时重建编辑器 */
watch(
  () => props.modelValue,
  (val) => {
    const next = val || ''
    if (next === lastValue) return
    if (editorRef.value && next === serializeEditor(editorRef.value).text) return
    renderFrom(next)
    lastValue = next
  }
)

onMounted(() => {
  nextTick(() => {
    renderFrom(props.modelValue)
    lastValue = props.modelValue || ''
    // 内容就位后再比对本地草稿，避免把「还没加载出来的正文」当成空稿
    initDraft(props.modelValue || '')
  })
})

function onPaste(e) {
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain') || ''
  if (!text) return
  document.execCommand('insertText', false, text)
  nextTick(() => {
    const value = syncFromDom()
    if (!HAS_MARKER_RE.test(value)) return
    // 粘贴内容里含标记时统一重建一次，直接显示图片 / 表情
    const offsets = currentOffsets()
    if (offsets) applySource(value, offsets.end)
  })
}

function onKeydown(e) {
  // @ 候选面板打开时优先接管方向键 / 回车 / Esc，避免回车被当成换行写进正文
  if (onMentionKeydown(e)) {
    e.preventDefault()
    return
  }

  const mod = e.ctrlKey || e.metaKey
  const key = e.key.toLowerCase()
  if (mod && key === 's') {
    e.preventDefault()
    emit('save')
    return
  }
  // 光标紧邻图片时，退格 / 删除整体移除该图片（原子节点）
  if ((e.key === 'Backspace' || e.key === 'Delete') && removeAdjacentImage(e.key)) {
    e.preventDefault()
    return
  }
  if (mod && key === 'b') {
    e.preventDefault()
    runTool((v, s, en) => wrapText(v, s, en, '**', '**', '加粗文字'))
    return
  }
  if (mod && key === 'i') {
    e.preventDefault()
    runTool((v, s, en) => wrapText(v, s, en, '*', '*', '斜体文字'))
    return
  }
  if (mod && key === 'k') {
    e.preventDefault()
    runTool(linkText)
    return
  }
  // Tab 缩进
  if (e.key === 'Tab') {
    e.preventDefault()
    document.execCommand('insertText', false, '  ')
    syncFromDom()
  }
}

// ===== 工具栏执行 =====
function runTool(apply) {
  if (mode.value === 'preview') mode.value = 'write'
  nextTick(() => {
    const root = editorRef.value
    if (!root) return
    root.focus()
    const { text } = serializeEditor(root)
    const offsets = currentOffsets() || { start: text.length, end: text.length }
    const result = apply(text, offsets.start, offsets.end)
    if (!result) return
    applySource(result.value, result.selStart, result.selEnd)
  })
}

// ===== 表情 =====
function insertEmoji(emoji) {
  const text = String(emoji || '')
  if (!text) return
  // 存储格式 [emoji:url]（或原生 emoji 字符）直接插入，插入后即时渲染
  if (mode.value === 'preview') mode.value = 'write'
  nextTick(() => insertMarker(text))
}

// ===== 插入 =====
/** 在光标处插入一段源码并即时渲染（图片会独占一行） */
function insertMarker(marker, block = false) {
  if (mode.value === 'preview') mode.value = 'write'
  nextTick(() => {
    const root = editorRef.value
    if (!root) return
    root.focus()
    const { text } = serializeEditor(root)
    const offsets = currentOffsets() || { start: text.length, end: text.length }
    const result = block
      ? blockMarkerText(text, offsets.start, offsets.end, marker)
      : insertTextAt(text, offsets.start, offsets.end, marker)
    applySource(result.value, result.selStart, result.selEnd)
  })
}

// ===== 图片上传 =====
function pickImage() {
  fileRef.value?.click()
}

function onFileChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (file) uploadAndInsert(file)
}

async function uploadAndInsert(file) {
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    toast.warning('图片不能超过 10MB')
    return
  }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadArticleImage(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')
    // 图片现在以 markdown 文本插入，文件名里的 [ ] 会破坏语法，先抹掉
    const alt = String(file.name || '').replace(/[[\]]/g, '').trim()
    insertMarker(`![${alt}](${url})`, true)
    toast.success('图片已插入')
  } catch {
    toast.error('图片上传失败，请稍后重试')
  } finally {
    uploading.value = false
  }
}

const isImageNode = (el) =>
  !!el && el.nodeType === 1 && (el.tagName === 'IMG' || (el.tagName === 'SPAN' && el.hasAttribute('data-md-src')))

/** 删除光标紧邻的表情原子（图片已是纯文本，不再需要整体删除），返回是否已处理 */
function removeAdjacentImage(key) {
  const root = editorRef.value
  const selection = window.getSelection()
  if (!root || !selection || !selection.rangeCount) return false
  const range = selection.getRangeAt(0)
  if (!range.collapsed) return false

  const node = range.startContainer
  const offset = range.startOffset
  let target = null
  if (node.nodeType === 1) {
    const children = Array.from(node.childNodes)
    target = key === 'Backspace' ? children[offset - 1] : children[offset]
  } else if (node.nodeType === 3) {
    if (key === 'Backspace' && offset === 0) target = node.previousSibling
    if (key === 'Delete' && offset === node.data.length) target = node.nextSibling
  }
  if (!isImageNode(target)) return false

  const parent = target.parentNode
  const at = offsetOfPoint(root, parent, Array.prototype.indexOf.call(parent.childNodes, target))
  target.remove()
  const { text } = serializeEditor(root)
  applySource(text, at === null ? text.length : at)
  return true
}

// 附件库：把选中的已上传图片按 markdown 图片语法插入光标处（各自独占一行）
const showLibrary = ref(false)
function onLibrarySelect(urls = []) {
  const list = urls.filter(Boolean)
  if (!list.length) return
  insertMarker(list.map((url) => `![](${url})`).join('\n\n'), true)
  toast.success(`已插入 ${list.length} 张图片`)
}

/* ===== 视频 / 音频 / 附件（正文短代码） =====
 * 工具栏「插入」面板三个入口共用同一个附件库弹窗，只是按当前类型过滤；
 * 插入的是 @[video](url "标题") / @[audio](url "标题") / @[file](url "文件名" "大小")，
 * 在编辑器里只是普通文本（不渲染播放器），解析在 @/utils/markdown（media_shortcode 规则）。
 */
const showMediaLibrary = ref(false)
const mediaRef = ref(null)
const mediaUploading = ref(false)
// 当前弹窗对应的短代码类型：video / audio / file
const mediaKind = ref('file')

const MEDIA_LIBRARY = {
  video: { title: '插入视频', accept: 'video' },
  audio: { title: '插入音频', accept: 'audio' },
  file: { title: '插入附件', accept: 'all' }
}

const mediaTitle = computed(() => MEDIA_LIBRARY[mediaKind.value]?.title || '插入附件')
const mediaAccept = computed(() => MEDIA_LIBRARY[mediaKind.value]?.accept || 'all')

function openMediaLibrary(kind) {
  mediaKind.value = MEDIA_LIBRARY[kind] ? kind : 'file'
  showMediaLibrary.value = true
}

function pickMedia() {
  mediaRef.value?.click()
}

/** 一批附件 → 若干短代码（每个独占一行）；kind 显式指定时不按扩展名猜 */
function attachmentMarkers(items = [], kind) {
  return items.map((item) => attachmentMarker(item, kind)).filter(Boolean).join('\n\n')
}

async function onMediaChange(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  if (!files.length) return

  mediaUploading.value = true
  try {
    const fd = new FormData()
    files.forEach((file) => fd.append('files', file))
    const res = await uploadAttachments(fd)

    // 后端返回的结果里带 original_name / file_size，取不到时回退本地文件信息
    const results = res.data?.results || []
    const items = results.map((row, index) => ({
      url: row.full_url || row.url || '',
      name: row.original_name || files[index]?.name || '',
      size: row.file_size || files[index]?.size || 0
    }))

    const markers = attachmentMarkers(items)
    if (!markers) throw new Error('empty')
    insertMarker(markers, true)
    toast.success(`已插入 ${items.length} 个文件`)
  } catch {
    toast.error('上传失败，请检查文件类型与大小限制')
  } finally {
    mediaUploading.value = false
  }
}

/* ===== 实况图（封面图 + 短视频） =====
 * 两种来源：
 *   1. 同时选「一张封面图 + 一段短视频」，分别上传后插入 @[live](视频URL "封面图URL")；
 *   2. 安卓手机拍的实况照片本身就是一张内嵌视频的 jpg，只选这张图即可，
 *      后端会拆出视频并通过 live_video 返回。
 */
const liveRef = ref(null)
const liveUploading = ref(false)

function pickLive() {
  liveRef.value?.click()
}

async function onLiveChange(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  if (!files.length) return

  const image = files.find((file) => file.type.startsWith('image/'))
  const video = files.find((file) => file.type.startsWith('video/'))
  if (!image) {
    toast.warning('请选择一张封面图（实况视频可选）')
    return
  }
  if (video && video.size > 30 * 1024 * 1024) {
    toast.warning('实况视频建议 1~3 秒、不超过 30MB')
    return
  }

  liveUploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', image)
    if (video) fd.append('files', video)
    const res = await uploadAttachments(fd)

    // 结果顺序与提交顺序一致：第 1 个是封面图，手动选了视频时第 2 个才是视频
    const rows = (res.data?.results || []).filter((row) => row.status !== 'fail' && (row.full_url || row.url))
    const imageUrl = rows[0]?.full_url || rows[0]?.url || ''
    const videoUrl = video ? rows[1]?.full_url || rows[1]?.url || '' : rows[0]?.live_video || ''
    if (!imageUrl || !videoUrl) {
      toast.warning(video ? '实况视频上传失败，请稍后重试' : '这张图片不是实况照片（未找到内嵌视频）')
      return
    }

    insertMarker(`@[live](${videoUrl} "${imageUrl}")`, true)
    toast.success('已插入实况图，发布后悬停（手机按住）可播放')
  } catch {
    toast.error('实况图上传失败，请检查文件类型与大小限制')
  } finally {
    liveUploading.value = false
  }
}

function onMediaLibrarySelect(urls = [], items = []) {
  const list = items?.length
    ? items.map((item) => ({
        url: item.full_url || item.url || '',
        name: item.original_name || '',
        size: item.file_size || 0
      }))
    : (urls || []).filter(Boolean).map((url) => ({ url }))

  // 显式带上当前按钮的类型，避免「附件」里选到视频时被猜成播放器短代码
  const markers = attachmentMarkers(list, mediaKind.value)
  if (!markers) return
  insertMarker(markers, true)
  toast.success(`已插入 ${list.length} 个短代码`)
}

/** 拖拽图片到编辑器直接上传 */
function onDrop(e) {
  const file = Array.from(e.dataTransfer?.files || []).find((f) => f.type.startsWith('image/'))
  if (file) uploadAndInsert(file)
}

defineExpose({
  focus: () => editorRef.value?.focus(),
  /** 表单保存成功后调用：清掉本地草稿，避免下次打开误提示 */
  clearDraft: () => clearDraft()
})
</script>

<style scoped>
/* ---------- 容器 ---------- */
.markdown-editor {
  position: relative;
  background: transparent;
}

/* ---------- 正文 ---------- */
.markdown-editor-body {
  position: relative;
}
.markdown-editor-content {
  display: block;
  width: 100%;
  padding: 20px 22px 24px;
  outline: none;
  min-height: 420px;
  color: var(--text);
  font-size: 15px;
  line-height: 1.95;
  font-family: var(--font-sans);
  /* 保留多个空格 / 空格缩进 */
  white-space: pre-wrap;
  word-break: break-word;
  overflow-wrap: anywhere;
  cursor: text;
  caret-color: var(--primary);
}
.markdown-editor-content.is-empty::before {
  content: attr(data-placeholder);
  color: var(--text-light);
  pointer-events: none;
}
.markdown-editor-preview {
  padding: 20px 22px 28px;
  overflow-y: auto;
  max-height: 620px;
  /* 预览区（尤其分屏时只有半屏宽）比正文窄，视频按 70vh 撑开会占满整列，
     这里用播放器暴露的 CSS 变量收一收高度（自定义属性会穿透 shadow DOM） */
  --md-video-max-height: 360px;
}

/* ---------- 行内表情（innerHTML 注入，需 deep） ---------- */
.markdown-editor-content :deep(.editor-emoji) {
  width: 24px;
  height: 24px;
  vertical-align: middle;
  display: inline-block;
  object-fit: contain;
  margin: 0 2px;
  border-radius: 4px;
}

/* ---------- 本地草稿提示条 ---------- */
.markdown-editor-draft {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 14px;
  background: var(--gold-wash);
  border-bottom: 1px solid var(--gold-line);
  font-size: 12px;
  color: var(--text-soft);
}
.markdown-editor-draft-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.markdown-editor-draft-text .bi {
  color: var(--warning);
}
.markdown-editor-draft-actions {
  display: inline-flex;
  gap: 6px;
  flex-shrink: 0;
}

/* ---------- 全屏编辑 ---------- */
.markdown-editor.is-fullscreen {
  position: fixed;
  inset: 0;
  /* 盖住页面布局（200）与悬浮按钮（999）、搜索（1060），低于各类弹窗（3000+） */
  z-index: 2000;
  display: flex;
  flex-direction: column;
  background: var(--bg);
  animation: mdFullscreenIn var(--dur) var(--ease);
}
.markdown-editor.is-fullscreen :deep(.editor-toolbar) {
  flex: 0 0 auto;
  padding: 8px 18px;
  border-top: none;
}
.markdown-editor.is-fullscreen .markdown-editor-body {
  flex: 1 1 auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.markdown-editor.is-fullscreen .markdown-editor-content,
.markdown-editor.is-fullscreen .markdown-editor-preview {
  flex: 1 1 auto;
  min-height: 0;
  max-height: none;
  width: 100%;
  max-width: 860px;
  margin: 0 auto;
  overflow-y: auto;
  padding: 28px 24px 64px;
}
/* 全屏分屏：左写右预览，两栏各自滚动 */
.markdown-editor.is-fullscreen .markdown-editor-body.is-split {
  flex-direction: row;
  align-items: stretch;
}
.markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-content,
.markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-preview {
  flex: 1 1 50%;
  min-width: 0;
  max-width: none;
  margin: 0;
  padding: 24px 22px 56px;
}
.markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-preview {
  border-left: 1px solid var(--border-soft);
  background: var(--bg-card);
}
.markdown-editor.is-fullscreen .markdown-editor-draft {
  padding-left: 22px;
  padding-right: 22px;
}

@keyframes mdFullscreenIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

@media (max-width: 768px) {
  .markdown-editor-content,
  .markdown-editor-preview {
    font-size: 14px;
    padding: 16px 14px 20px;
  }
  /* 窄屏放不下左右两栏：分屏改成上下两半（上写下看） */
  .markdown-editor.is-fullscreen .markdown-editor-body.is-split {
    flex-direction: column;
  }
  .markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-content,
  .markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-preview {
    flex: 1 1 50%;
    padding: 14px 14px 28px;
  }
  .markdown-editor.is-fullscreen .markdown-editor-body.is-split .markdown-editor-preview {
    border-left: none;
    border-top: 1px solid var(--border-soft);
  }
  .markdown-editor-draft {
    flex-wrap: wrap;
    padding: 8px 12px;
  }
}
</style>
