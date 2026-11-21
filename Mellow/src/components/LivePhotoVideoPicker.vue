<template>
  <!-- 配图右下角的「实况」按钮：点开可选本地上传或从附件库挑一条已上传的视频 -->
  <span ref="rootRef" class="live-picker">
    <button
      type="button"
      class="live-picker-btn"
      :class="{ 'is-on': hasVideo }"
      :disabled="busy"
      :title="hasVideo ? '更换实况视频（本地上传 / 附件库）' : '设为实况图（本地上传 / 附件库）'"
      @click.stop="toggleMenu"
    >
      <i
        class="bi"
        :class="busy ? 'bi-arrow-repeat live-picker-spin' : hasVideo ? 'bi-record-circle-fill' : 'bi-record-circle'"
      />
    </button>

    <!-- 菜单必须 teleport 出去：配图容器是 72×72 + overflow:hidden，直接放里面会被裁掉 -->
    <Teleport to="body">
      <div v-if="open" ref="panelRef" class="live-picker-panel" :style="panelStyle" role="menu">
        <button type="button" class="live-picker-item" role="menuitem" @click="pickUpload">
          <i class="bi bi-upload" /> 上传短视频
        </button>
        <button type="button" class="live-picker-item" role="menuitem" @click="pickLibrary">
          <i class="bi bi-folder2-open" /> 从附件库选择
        </button>
        <button v-if="hasVideo" type="button" class="live-picker-item is-danger" role="menuitem" @click="clearVideo">
          <i class="bi bi-x-circle" /> 取消实况
        </button>
      </div>
    </Teleport>

    <input
      ref="fileInput"
      type="file"
      accept="video/*,.mp4,.mov,.m4v,.webm"
      class="live-picker-file"
      @change="onPickFile"
    />

    <!-- 附件库：只列视频，单选后直接挂到这张图上 -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择实况视频"
      accept="video"
      @select="onLibrarySelect"
    />
  </span>
</template>

<script setup>
/**
 * 实况图视频选择器（动态发布 / 动态编辑共用）
 *
 * 每张配图右下角一个按钮，点开两个来源：
 *   - 本地上传 1~3 秒短视频（走 attachment/batch，与图片同一套上传逻辑）；
 *   - 从附件库挑选已上传的视频（accept="video"）。
 * 选中的地址通过 v-model:video 交回父级，父级负责拼「图片::视频」。
 *
 * 注意：菜单用 Teleport + fixed 定位。配图容器是 72×72 且 overflow:hidden，
 * 菜单若放在图片内会被裁掉；z-index 取 3400（高于页面弹窗 3000，低于附件库 3500）。
 */
import { computed, onBeforeUnmount, ref } from 'vue'
import AttachmentLibrary from './AttachmentLibrary.vue'
import { uploadAttachments } from '@/api/attachment'
import { toast } from '@/utils/toast'

const props = defineProps({
  // 当前实况视频地址，空串表示普通图片
  video: { type: String, default: '' },
  // 父级正在提交 / 禁用时置灰
  disabled: { type: Boolean, default: false }
})

const emit = defineEmits(['update:video'])

/** 实况短视频只做氛围动效，超过 30MB 基本是误传了完整视频 */
const MAX_LIVE_SIZE = 30 * 1024 * 1024
const PANEL_WIDTH = 168

const rootRef = ref(null)
const panelRef = ref(null)
const fileInput = ref(null)
const showLibrary = ref(false)
const uploading = ref(false)
const open = ref(false)
const panelStyle = ref({})

const hasVideo = computed(() => Boolean(props.video))
const busy = computed(() => props.disabled || uploading.value)

function toggleMenu() {
  if (busy.value) return
  if (open.value) {
    closeMenu()
    return
  }
  positionPanel()
  open.value = true
  document.addEventListener('pointerdown', onDocPointerDown, true)
  document.addEventListener('keydown', onKeydown)
}

function closeMenu() {
  open.value = false
  document.removeEventListener('pointerdown', onDocPointerDown, true)
  document.removeEventListener('keydown', onKeydown)
}

/** 按按钮位置定位菜单，靠近屏幕右/下边缘时向内收，避免溢出 */
function positionPanel() {
  const rect = rootRef.value?.getBoundingClientRect()
  if (!rect) {
    panelStyle.value = {}
    return
  }
  const left = Math.min(rect.left, window.innerWidth - PANEL_WIDTH - 12)
  const top = Math.min(rect.bottom + 6, window.innerHeight - 140)
  panelStyle.value = {
    top: `${Math.max(8, top)}px`,
    left: `${Math.max(8, left)}px`
  }
}

function onDocPointerDown(event) {
  if (panelRef.value?.contains(event.target) || rootRef.value?.contains(event.target)) return
  closeMenu()
}

function onKeydown(event) {
  if (event.key === 'Escape') closeMenu()
}

onBeforeUnmount(closeMenu)

function pickUpload() {
  closeMenu()
  fileInput.value?.click()
}

function pickLibrary() {
  closeMenu()
  showLibrary.value = true
}

function clearVideo() {
  closeMenu()
  emit('update:video', '')
}

async function onPickFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  if (!file.type.startsWith('video/')) {
    toast.warning('请选择视频文件（mp4 / mov / webm）')
    return
  }
  if (file.size > MAX_LIVE_SIZE) {
    toast.warning('实况视频建议 1~3 秒、不超过 30MB')
    return
  }

  uploading.value = true
  try {
    const formData = new FormData()
    formData.append('files', file)
    const res = await uploadAttachments(formData)
    const row = (res.data?.results || []).find((item) => item.status !== 'fail' && item.full_url)
    if (!row) {
      toast.warning('视频上传失败，请稍后重试')
      return
    }
    emit('update:video', row.full_url)
    toast.success('已设为实况图，发布后悬停（手机按住）可播放')
  } catch {
    /* 拦截器已提示（多半是站点未允许上传该扩展名） */
  } finally {
    uploading.value = false
  }
}

function onLibrarySelect(urls = []) {
  const url = urls[0]
  if (!url) return
  emit('update:video', url)
  toast.success('已选用附件库里的视频')
}
</script>

<style scoped>
.live-picker {
  position: absolute;
  right: 4px;
  bottom: 4px;
  display: block;
  line-height: 0;
}

.live-picker-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.live-picker-btn:hover {
  background: rgba(0, 0, 0, 0.65);
}
.live-picker-btn.is-on {
  color: #ffd6a5;
}
.live-picker-btn:disabled {
  opacity: 0.5;
  cursor: default;
}

.live-picker-spin {
  animation: live-picker-spin 0.9s linear infinite;
}
@keyframes live-picker-spin {
  to {
    transform: rotate(360deg);
  }
}

.live-picker-file {
  display: none;
}

/* 菜单：fixed，脱离图片的 overflow 裁剪 */
.live-picker-panel {
  position: fixed;
  z-index: 3400;
  width: 168px;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm, 8px);
  background: var(--bg-card, #fff);
  box-shadow: var(--shadow-md, 0 8px 24px rgba(0, 0, 0, 0.16));
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.live-picker-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 9px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text, #333);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
}
.live-picker-item:hover {
  background: var(--bg-muted, #f5f2ec);
}
.live-picker-item.is-danger {
  color: var(--danger, #c0392b);
}
</style>
