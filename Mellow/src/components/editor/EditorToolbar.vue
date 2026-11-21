<template>
  <div class="editor-toolbar" role="toolbar" aria-label="编辑器工具栏">
    <!-- 插入组：表情 / @提及 / 图片 / 媒体 / 附件库 -->
    <div class="editor-toolbar-group">
      <button
        type="button"
        class="editor-toolbar-button"
        title="插入表情"
        aria-label="插入表情"
        @mousedown.prevent
        @click="toggleEmoji"
      >
        <i class="bi bi-emoji-smile" />
      </button>

      <button
        type="button"
        class="editor-toolbar-button"
        title="@ 提及"
        aria-label="@ 提及"
        @mousedown.prevent
        @click="emitMention"
      >
        <i class="bi bi-at" />
      </button>

      <button
        type="button"
        class="editor-toolbar-button"
        :title="uploading ? '图片上传中' : '上传图片'"
        aria-label="上传图片"
        :disabled="uploading"
        @mousedown.prevent
        @click="emit('pick-image')"
      >
        <i :class="uploading ? 'bi bi-arrow-repeat editor-toolbar-spin' : 'bi bi-image'" />
      </button>

      <!-- 媒体类动作收进下拉面板：工具栏不再被 5 个同类图标占满 -->
      <div ref="insertAnchor" class="editor-toolbar-anchor">
        <button
          type="button"
          class="editor-toolbar-button editor-toolbar-button-wide"
          :class="{ 'is-active': insertOpen }"
          title="插入视频 / 音频 / 附件 / 实况图"
          aria-haspopup="menu"
          :aria-expanded="insertOpen"
          @mousedown.prevent
          @click="insertOpen = !insertOpen"
        >
          <i class="bi bi-plus-lg" />
          <span class="editor-toolbar-label">插入</span>
        </button>

        <div v-if="insertOpen" class="editor-insert-menu" role="menu">
          <p class="editor-insert-title">插入内容</p>
          <button
            v-for="item in INSERT_ITEMS"
            :key="item.key"
            type="button"
            class="editor-insert-item"
            role="menuitem"
            :class="{ 'is-busy': isBusy(item.key) }"
            :disabled="isBusy(item.key)"
            @mousedown.prevent
            @click="chooseInsert(item)"
          >
            <i :class="isBusy(item.key) ? 'bi bi-arrow-repeat editor-toolbar-spin' : item.icon" />
            <span class="editor-insert-text">
              <em>{{ item.label }}</em>
              <small>{{ item.hint }}</small>
            </span>
          </button>
        </div>
      </div>

      <button
        type="button"
        class="editor-toolbar-button"
        title="从附件库插入图片"
        aria-label="从附件库插入图片"
        @mousedown.prevent
        @click="emit('open-library')"
      >
        <i class="bi bi-folder2-open" />
      </button>
    </div>

    <span class="editor-toolbar-line" aria-hidden="true" />

    <!-- 格式组：加粗 / 标题 / 列表 / 代码 / 表格… -->
    <div class="editor-toolbar-group">
      <button
        v-for="tool in tools"
        :key="tool.key"
        type="button"
        class="editor-toolbar-button"
        :title="tool.title"
        :aria-label="tool.title"
        @mousedown.prevent
        @click="emit('select-tool', tool)"
      >
        <i v-if="tool.icon" :class="tool.icon" />
        <span v-else class="editor-toolbar-text">{{ tool.text }}</span>
      </button>
    </div>

    <span class="editor-toolbar-flex" />

    <!-- 视图组：全屏 / 分屏 / 预览 -->
    <div class="editor-toolbar-group">
      <button
        type="button"
        class="editor-toolbar-button"
        :class="{ 'is-active': fullscreen }"
        :title="fullscreen ? '退出全屏 (Esc)' : '全屏编辑'"
        aria-label="全屏编辑"
        @mousedown.prevent
        @click="emit('toggle-fullscreen')"
      >
        <i :class="fullscreen ? 'bi bi-fullscreen-exit' : 'bi bi-arrows-fullscreen'" />
      </button>

      <button
        v-if="fullscreen"
        type="button"
        class="editor-toolbar-button"
        :class="{ 'is-active': splitView }"
        :title="splitView ? '退出分屏，单栏编辑' : '分屏预览（左写右预览）'"
        aria-label="分屏预览"
        @mousedown.prevent
        @click="emit('toggle-split')"
      >
        <i class="bi bi-layout-split" />
      </button>

      <button
        type="button"
        class="editor-toolbar-button"
        :class="{ 'is-active': mode === 'preview' && !splitView }"
        :title="mode === 'preview' && !splitView ? '继续编辑' : splitView ? '只看预览' : '预览'"
        :aria-label="previewing ? '继续编辑' : '预览'"
        @click="emit('toggle-mode')"
      >
        <i :class="previewing ? 'bi bi-pencil-square' : 'bi bi-eye'" />
      </button>
    </div>

    <!-- 表情面板：跟着工具栏定位，窄屏时宽度跟随整个工具栏而不是单个按钮 -->
    <EmojiPicker v-if="emojiOpen" v-model="emojiOpen" @select="emit('insert-emoji', $event)" />
  </div>
</template>

<script setup>
/**
 * 文章编辑器工具栏（纯展示 + 面板管理，不碰正文 DOM）
 *
 * 分三组，避免十多个图标平铺：
 *   插入组：表情 / @提及 / 图片 / 插入菜单（视频、音频、附件、实况图、上传媒体）/ 附件库
 *   格式组：由父级传入 tools（加粗、标题、列表、代码、表格…）
 *   视图组：全屏 / 分屏 / 预览
 *
 * 所有按钮都带 @mousedown.prevent：点击时不能抢走 contenteditable 的选区，
 * 否则工具会作用不到用户选中的文字。
 */
import { computed, onUnmounted, ref, watch } from 'vue'
import EmojiPicker from '@/components/EmojiPicker.vue'

const props = defineProps({
  // 格式工具：{ key, title, icon?, text?, apply }
  tools: { type: Array, default: () => [] },
  mode: { type: String, default: 'write' },
  splitView: { type: Boolean, default: false },
  fullscreen: { type: Boolean, default: false },
  // 三类上传各自的进行中状态
  uploading: { type: Boolean, default: false },
  mediaUploading: { type: Boolean, default: false },
  liveUploading: { type: Boolean, default: false }
})

const emit = defineEmits([
  'select-tool',
  'insert-emoji',
  'toggle-mention',
  'pick-image',
  'open-library',
  'open-media-library',
  'pick-media',
  'pick-live',
  'toggle-mode',
  'toggle-split',
  'toggle-fullscreen'
])

const INSERT_ITEMS = [
  { key: 'video', icon: 'bi bi-camera-video', label: '视频', hint: '从附件库选择' },
  { key: 'audio', icon: 'bi bi-music-note-beamed', label: '音频', hint: '从附件库选择' },
  { key: 'file', icon: 'bi bi-paperclip', label: '附件', hint: '任意文件，显示为下载卡' },
  { key: 'live', icon: 'bi bi-record-circle', label: '实况图', hint: '封面图 + 一段短视频' },
  { key: 'upload', icon: 'bi bi-cloud-arrow-up', label: '上传媒体', hint: '按格式自动识别类型' }
]

const previewing = computed(() => props.mode === 'preview' && !props.splitView)

const emojiOpen = ref(false)
const insertOpen = ref(false)
const insertAnchor = ref(null)

function toggleEmoji() {
  insertOpen.value = false
  emojiOpen.value = !emojiOpen.value
}

/** 工具栏 @ 按钮：把按钮矩形交给父级算面板坐标 */
function emitMention(event) {
  insertOpen.value = false
  emojiOpen.value = false
  const rect = event?.currentTarget?.getBoundingClientRect?.() || null
  emit('toggle-mention', rect)
}

function isBusy(key) {
  if (key === 'live') return props.liveUploading
  if (key === 'upload') return props.mediaUploading
  return false
}

function chooseInsert(item) {
  insertOpen.value = false
  if (item.key === 'live') {
    emit('pick-live')
    return
  }
  if (item.key === 'upload') {
    emit('pick-media')
    return
  }
  emit('open-media-library', item.key)
}

/* 插入面板：点外面 / 按 Esc 关闭，避免浮层一直挂在正文上方 */
function onPointerDownOutside(event) {
  if (!insertAnchor.value?.contains(event.target)) insertOpen.value = false
}
function onEscape(event) {
  if (event.key !== 'Escape') return
  insertOpen.value = false
  emojiOpen.value = false
  // 关面板这一下不要顺手把全屏也退掉（全屏的 Esc 监听挂在 window 上）
  event.stopPropagation()
}

watch(insertOpen, (open) => {
  if (open) {
    document.addEventListener('pointerdown', onPointerDownOutside)
    document.addEventListener('keydown', onEscape)
    emojiOpen.value = false
    return
  }
  document.removeEventListener('pointerdown', onPointerDownOutside)
  document.removeEventListener('keydown', onEscape)
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', onPointerDownOutside)
  document.removeEventListener('keydown', onEscape)
})

// 供编辑器判断「面板还开着吗」：此时第一次 Esc 只关面板，不退出全屏
defineExpose({
  isPanelOpen: () => emojiOpen.value || insertOpen.value
})
</script>

<style scoped>
.editor-toolbar {
  position: relative;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  padding: 8px 10px;
  background: var(--bg-card);
  border-top: 1px solid var(--border-soft);
  border-bottom: 1px solid var(--border-soft);
}

/* ---------- 分组：浅底胶囊，比平铺图标更易扫读 ---------- */
.editor-toolbar-group {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: var(--radius);
  background: var(--bg-muted);
}
.editor-toolbar-line {
  width: 1px;
  height: 18px;
  flex-shrink: 0;
  background: var(--border-soft);
}
.editor-toolbar-flex {
  flex: 1;
  min-width: 8px;
}

/* ---------- 按钮 ---------- */
.editor-toolbar-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 28px;
  height: 28px;
  padding: 0 6px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}
.editor-toolbar-button:hover:not(:disabled) {
  background: var(--bg-card);
  color: var(--primary);
  box-shadow: var(--shadow-sm);
}
.editor-toolbar-button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.editor-toolbar-button.is-active {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.editor-toolbar-button-wide {
  padding: 0 8px 0 6px;
}
.editor-toolbar-label {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.editor-toolbar-text {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: -0.2px;
}

/* ---------- 插入面板 ---------- */
.editor-toolbar-anchor {
  position: relative;
}
.editor-insert-menu {
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 120;
  width: 228px;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  animation: editorInsertIn var(--dur-fast) var(--ease);
}
.editor-insert-title {
  margin: 4px 6px 2px;
  font-size: 11px;
  letter-spacing: 0.08em;
  color: var(--text-light);
}
.editor-insert-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.editor-insert-item:hover:not(:disabled) {
  background: var(--bg-muted);
  color: var(--primary-deep);
}
.editor-insert-item:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.editor-insert-item .bi {
  font-size: 16px;
  flex-shrink: 0;
}
.editor-insert-text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.editor-insert-text em {
  font-style: normal;
  font-size: 13px;
  font-weight: 500;
}
.editor-insert-text small {
  font-size: 11px;
  color: var(--text-light);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@keyframes editorInsertIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: none;
  }
}

.editor-toolbar-spin {
  display: inline-block;
  animation: editorToolbarSpin 0.8s linear infinite;
}
@keyframes editorToolbarSpin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 窄屏：允许换行而不是横向滚动（弹出面板会被 overflow 裁掉） ---------- */
@media (max-width: 768px) {
  .editor-toolbar {
    gap: 4px;
    padding: 6px 8px;
  }
  .editor-toolbar-line,
  .editor-toolbar-flex {
    display: none;
  }
  .editor-toolbar-button {
    min-width: 30px;
    height: 30px;
  }
  .editor-toolbar-group:last-of-type {
    margin-left: auto;
  }
  .editor-insert-menu {
    width: min(228px, calc(100vw - 32px));
  }
}
</style>
