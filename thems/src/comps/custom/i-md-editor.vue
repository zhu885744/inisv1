<template>
  <div 
    class="md-editor" 
    :class="{ 'md-editor--fullscreen': isFullscreen }"
    :style="{ height: isFullscreen ? '100vh' : editorHeight + 'px' }"
  >
    <div class="md-editor__toolbar">
      <div class="md-editor__toolbar-group">
        <button class="md-editor__btn" @click="triggerImageUpload" title="插入图片">
          <i class="bi bi-image"></i>
        </button>
        <button class="md-editor__btn" @click="insertFormat('bold')" title="加粗">
          <i class="bi bi-type-bold"></i>
        </button>
        <button class="md-editor__btn" @click="insertFormat('italic')" title="斜体">
          <i class="bi bi-type-italic"></i>
        </button>
        <button class="md-editor__btn" @click="insertFormat('link')" title="链接">
          <i class="bi bi-link-45deg"></i>
        </button>
      </div>
      <div class="md-editor__toolbar-divider"></div>
      <div class="md-editor__toolbar-group">
        <button class="md-editor__btn" @click="insertFormat('quote')" title="引用">
          <i class="bi bi-quote"></i>
        </button>
        <button class="md-editor__btn" @click="insertFormat('code')" title="代码">
          <i class="bi bi-code"></i>
        </button>
      </div>
      <div class="md-editor__toolbar-divider"></div>
      <div class="md-editor__toolbar-group">
        <button class="md-editor__btn emoji-button" @click="toggleEmojiPicker" :class="{ active: showEmojiPicker }" title="表情">
          <i class="bi bi-emoji-smile"></i>
        </button>
      </div>
      <div class="md-editor__toolbar-spacer"></div>
      <div class="md-editor__toolbar-group">
        <button class="md-editor__btn" @click="toggleFullscreen" :title="isFullscreen ? '退出全屏' : '全屏'">
          <i :class="isFullscreen ? 'bi bi-arrows-fullscreen' : 'bi bi-arrows-fullscreen'"></i>
        </button>
      </div>
    </div>

    <!-- 表情选择面板 -->
    <i-emoji-picker
      v-model="showEmojiPicker"
      @select="insertEmoji"
    />

    <textarea
      ref="textareaRef"
      v-model="state.content"
      class="md-editor__textarea"
      :placeholder="placeholder"
      @input="handleInput"
      @keydown="handleKeydown"
    ></textarea>

    <div 
      class="md-editor__resize-handle" 
      @mousedown="startResize"
      title="拖动调整高度"
    >
      <i class="bi bi-grip-vertical"></i>
    </div>

    <input
      type="file"
      ref="fileInputRef"
      class="d-none"
      accept="image/*"
      @change="handleFileChange"
    />
  </div>
</template>

<script setup>
import { ref, reactive, watch, onMounted, onUnmounted } from 'vue'
import { request, checkFileType } from '@/utils/network'
import { toast, getSync } from '@/utils/app'
import iEmojiPicker from '@/comps/custom/i-emoji-picker.vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  placeholder: {
    type: String,
    default: '写点什么吧！'
  },
  height: {
    type: Number,
    default: 400
  }
})

const emit = defineEmits(['update:modelValue'])

const textareaRef = ref(null)
const fileInputRef = ref(null)

const state = reactive({
  content: ''
})

const editorHeight = ref(props.height)
const isFullscreen = ref(false)
const isResizing = ref(false)
const startY = ref(0)
const startHeight = ref(0)
const showEmojiPicker = ref(false)

watch(() => props.modelValue, (newVal) => {
  if (newVal !== state.content) {
    state.content = newVal
  }
}, { immediate: true })

watch(() => props.height, (newVal) => {
  editorHeight.value = newVal
})

const handleInput = () => {
  emit('update:modelValue', state.content)
}

const handleKeydown = (e) => {
  if (e.key === 'Tab') {
    e.preventDefault()
    insertText('  ')
  }
}

const insertText = (text) => {
  const textarea = textareaRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const content = state.content

  state.content = content.substring(0, start) + text + content.substring(end)
  emit('update:modelValue', state.content)

  setTimeout(() => {
    textarea.focus()
    textarea.selectionStart = textarea.selectionEnd = start + text.length
  }, 0)
}

// 表情功能
const toggleEmojiPicker = () => {
  showEmojiPicker.value = !showEmojiPicker.value
}

const insertEmoji = (emoji) => {
  if (emoji && emoji.startsWith('[emoji:')) {
    // 图片表情：转为 markdown 图片语法插入
    const url = emoji.slice(7, -1)
    insertText(`![表情](${url})`)
  } else {
    // 文本表情（颜文字）：直接插入
    insertText(emoji + ' ')
  }
}

const insertFormat = (type) => {
  const textarea = textareaRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const selectedText = state.content.substring(start, end)

  let insertText = ''
  let cursorOffset = 0

  switch (type) {
    case 'bold':
      insertText = `**${selectedText || '粗体文本'}**`
      cursorOffset = selectedText ? insertText.length : 2
      break
    case 'italic':
      insertText = `*${selectedText || '斜体文本'}*`
      cursorOffset = selectedText ? insertText.length : 1
      break
    case 'link':
      insertText = `[${selectedText || '链接文本'}](url)`
      cursorOffset = selectedText ? insertText.length - 1 : 1
      break
    case 'quote':
      insertText = `> ${selectedText || '引用文本'}`
      cursorOffset = insertText.length
      break
    case 'code':
      insertText = `\`${selectedText || '代码'}\``
      cursorOffset = selectedText ? insertText.length : 1
      break
    default:
      return
  }

  const content = state.content
  state.content = content.substring(0, start) + insertText + content.substring(end)
  emit('update:modelValue', state.content)

  setTimeout(() => {
    textarea.focus()
    textarea.selectionStart = textarea.selectionEnd = start + cursorOffset
  }, 0)
}

const triggerImageUpload = () => {
  fileInputRef.value?.click()
}

const handleFileChange = async (e) => {
  const file = e.target.files?.[0]
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.error('请选择图片文件')
    return
  }

  if (file.size > 5 * 1024 * 1024) {
    toast.error('图片大小不能超过 5MB')
    return
  }

  try {
    await checkFileType([file.name])

    const formData = new FormData()
    formData.append('file', file)

    const { code, msg, data } = await request.post('/api/attachment/batch', formData)

    if (code !== 200) {
      throw new Error(msg)
    }

    const fullUrl = data.results?.[0]?.full_url
    if (!fullUrl) {
      throw new Error('上传失败，未返回文件链接')
    }
    const imageMarkdown = `![${file.name}](${fullUrl})`
    insertText(imageMarkdown)
    toast.success('图片上传成功')
  } catch (err) {
    toast.error(err.message || '上传失败')
  } finally {
    e.target.value = ''
  }
}

const startResize = (e) => {
  if (isFullscreen.value) return
  isResizing.value = true
  startY.value = e.clientY
  startHeight.value = editorHeight.value
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

const handleResize = (e) => {
  if (!isResizing.value) return
  const deltaY = e.clientY - startY.value
  const newHeight = Math.max(200, Math.min(1200, startHeight.value + deltaY))
  editorHeight.value = newHeight
}

const stopResize = () => {
  isResizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

const toggleFullscreen = () => {
  if (isFullscreen.value) {
    exitFullscreen()
  } else {
    enterFullscreen()
  }
}

const enterFullscreen = () => {
  isFullscreen.value = true
  document.body.style.overflow = 'hidden'
}

const exitFullscreen = () => {
  isFullscreen.value = false
  document.body.style.overflow = ''
}

const handleKeydownFullscreen = (e) => {
  if (e.key === 'Escape' && isFullscreen.value) {
    exitFullscreen()
  }
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydownFullscreen)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydownFullscreen)
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.overflow = ''
})

defineExpose({
  getValue: () => state.content,
  setValue: (value) => {
    state.content = value
    emit('update:modelValue', value)
  },
  insertText,
  toggleFullscreen,
  exitFullscreen
})
</script>

<style scoped>
.md-editor {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-lg);
  overflow: hidden;
  background: var(--bs-body-bg);
  position: relative;
  min-height: 200px;
  box-shadow: var(--wx-shadow-sm);
  transition: box-shadow var(--wx-transition), height 0.1s ease;
}

.md-editor:focus-within {
  border-color: var(--bs-primary);
  box-shadow: var(--wx-shadow-md), 0 0 0 3px rgba(var(--bs-primary-rgb), 0.12);
}

.md-editor--fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  border: none;
  border-radius: 0;
  box-shadow: none;
}

.md-editor__fullscreen-header {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
}

.md-editor__close-btn {
  width: 36px;
  height: 36px;
  border: none;
  background: rgba(0, 0, 0, 0.5);
  border-radius: 50%;
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--wx-transition);
}

.md-editor__close-btn:hover {
  background: rgba(0, 0, 0, 0.7);
}

.md-editor__toolbar {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  background: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
  gap: 2px;
}

.md-editor__toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.md-editor__toolbar-divider {
  width: 1px;
  height: 20px;
  background: var(--bs-border-color);
  margin: 0 4px;
}

.md-editor__toolbar-spacer {
  flex: 1;
}

.md-editor__btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--wx-radius-sm);
  color: var(--bs-body-color);
  font-size: 14px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--wx-transition);
}

.md-editor__btn:hover {
  background: var(--bs-secondary-bg);
  color: var(--bs-primary);
  transform: translateY(-1px);
}

.md-editor__btn:active {
  background: var(--bs-primary);
  color: #fff;
  transform: translateY(0);
}

.md-editor__btn.active {
  background: rgba(var(--bs-primary-rgb), 0.12);
  color: var(--bs-primary);
}

.md-editor__textarea {
  flex: 1;
  width: 100%;
  padding: 16px;
  border: none;
  resize: none;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.7;
  outline: none;
  background: transparent;
  color: var(--bs-body-color);
}

.md-editor__textarea::placeholder {
  color: var(--bs-secondary-color);
}

.md-editor__resize-handle {
  height: 6px;
  cursor: ns-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--bs-border-color);
  font-size: 10px;
  transition: var(--wx-transition);
}

.md-editor__resize-handle:hover {
  background: var(--bs-secondary-bg);
  color: var(--bs-secondary-color);
}

.md-editor--fullscreen .md-editor__resize-handle {
  display: none;
}
</style>
