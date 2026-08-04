<template>
  <!-- 发布模式：内联卡片（保持原样） -->
  <div v-if="!isEditing" class="moment-editor wx-card">
    <div class="card-body p-3">
      <!-- 标题 -->
      <div class="d-flex align-items-center mb-3">
        <h5 class="mb-0">
          <i class="bi bi-pencil-square me-2"></i>
          发布动态【发布后请耐心等待审核】
        </h5>
      </div>

      <!-- 内容输入 -->
      <div class="mb-3">
        <div
          ref="publishEditorRef"
          class="form-control border border-secondary-subtle bg-body wx-contenteditable"
          :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !formData.content }"
          contenteditable="true"
          data-placeholder="分享你的想法..."
          @input="onContentInput"
          style="min-height: 100px; max-height: 300px; overflow-y: auto;"
        ></div>
        <div class="d-flex justify-content-between mt-1">
          <button
            type="button"
            class="wx-btn-outline btn-sm emoji-button"
            @click="toggleEmojiPicker('publish')"
            :class="{ active: showEmojiPicker && activeEditorMode === 'publish' }"
          >
            <i class="bi bi-emoji-smile"></i> 表情
          </button>
          <small class="text-muted">{{ formData.content.length }}/2000</small>
        </div>
        <!-- 表情选择面板 -->
        <i-emoji-picker
          v-if="activeEditorMode === 'publish'"
          v-model="showEmojiPicker"
          :is-dark-mode="isDarkMode"
          @select="insertEmoji"
        />
      </div>

      <!-- 图片上传 -->
      <div class="mb-3">
        <div class="d-flex align-items-center gap-2 mb-2">
          <button
            type="button"
            class="wx-btn-outline"
            @click="handleUploadImage"
            :disabled="uploadingImage || imageList.length >= MAX_IMAGES"
          >
            <i class="bi" :class="uploadingImage ? 'bi-arrow-clockwise spin' : 'bi-image'"></i>
            {{ uploadingImage ? '上传中...' : '添加图片' }}
          </button>
          <small class="text-muted">
            最多 {{ MAX_IMAGES }} 张
            <span v-if="imageList.length > 0" class="ms-1">{{ imageList.length }}/{{ MAX_IMAGES }}</span>
          </small>
        </div>

        <!-- 图片预览 -->
        <div v-if="imageList.length > 0" class="image-preview-grid">
          <div
            v-for="(img, idx) in imageList"
            :key="idx"
            class="image-preview-item"
          >
            <img :src="img" :alt="`图片 ${idx + 1}`" />
            <button
              type="button"
              class="btn btn-danger btn-sm image-remove-btn"
              @click="removeImage(idx)"
            >
              <i class="bi bi-x"></i>
            </button>
          </div>
        </div>
      </div>

      <!-- 位置信息 -->
      <div class="mb-3">
        <div class="input-group">
          <span class="input-group-text"><i class="bi bi-geo-alt"></i></span>
          <v-region-selects
            v-model="regionValues"
            :city="false"
            :area="false"
            :town="false"
            class="flex-grow-1"
            @change="handleRegionChange"
          />
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="d-flex gap-2 justify-content-end">
        <button
          type="button"
          class="wx-btn-outline"
          @click="handleSaveDraft"
          :disabled="isSubmitting || !formData.content.trim()"
        >
          <i class="bi bi-save me-1"></i>
          {{ isSubmitting ? '保存中...' : '保存草稿' }}
        </button>
        <button
          type="button"
          class="wx-btn-gradient"
          @click="handlePublish"
          :disabled="isSubmitting || !formData.content.trim()"
        >
          <i class="bi bi-send me-1"></i>
          {{ isSubmitting ? '发布中...' : '发布' }}
        </button>
      </div>
    </div>
  </div>

  <!-- 编辑模式：弹窗（始终挂载，由 Bootstrap Modal 控制显示/隐藏） -->
  <Teleport to="body">
    <div ref="modalEl" class="modal fade" tabindex="-1" aria-hidden="true">
      <div class="modal-dialog modal-dialog-centered modal-lg modal-dialog-scrollable">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">
              <i class="bi bi-pencil-square me-2"></i>编辑动态
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="关闭"></button>
          </div>
          <div class="modal-body">
            <!-- 内容输入 -->
            <div class="mb-3">
              <div
                ref="editEditorRef"
                class="form-control border border-secondary-subtle bg-body wx-contenteditable"
                :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !formData.content }"
                contenteditable="true"
                data-placeholder="分享你的想法..."
                @input="onContentInput"
                style="min-height: 100px; max-height: 300px; overflow-y: auto;"
              ></div>
              <div class="d-flex justify-content-between mt-1">
                <button
                  type="button"
                  class="wx-btn-outline btn-sm emoji-button"
                  @click="toggleEmojiPicker('edit')"
                  :class="{ active: showEmojiPicker && activeEditorMode === 'edit' }"
                >
                  <i class="bi bi-emoji-smile"></i> 表情
                </button>
                <small class="text-muted">{{ formData.content.length }}/2000</small>
              </div>
              <!-- 表情选择面板 -->
              <i-emoji-picker
                v-if="activeEditorMode === 'edit'"
                v-model="showEmojiPicker"
                :is-dark-mode="isDarkMode"
                @select="insertEmoji"
              />
            </div>

            <!-- 图片上传 -->
            <div class="mb-3">
              <div class="d-flex align-items-center gap-2 mb-2">
                <button
                  type="button"
                  class="wx-btn-outline"
                  @click="handleUploadImage"
                  :disabled="uploadingImage || imageList.length >= MAX_IMAGES"
                >
                  <i class="bi" :class="uploadingImage ? 'bi-arrow-clockwise spin' : 'bi-image'"></i>
                  {{ uploadingImage ? '上传中...' : '添加图片' }}
                </button>
                <small class="text-muted">
                  最多 {{ MAX_IMAGES }} 张
                  <span v-if="imageList.length > 0" class="ms-1">{{ imageList.length }}/{{ MAX_IMAGES }}</span>
                </small>
              </div>

              <div v-if="imageList.length > 0" class="image-preview-grid">
                <div
                  v-for="(img, idx) in imageList"
                  :key="idx"
                  class="image-preview-item"
                >
                  <img :src="img" :alt="`图片 ${idx + 1}`" />
                  <button
                    type="button"
                    class="btn btn-danger btn-sm image-remove-btn"
                    @click="removeImage(idx)"
                  >
                    <i class="bi bi-x"></i>
                  </button>
                </div>
              </div>
            </div>

            <!-- 位置信息 -->
            <div class="mb-3">
              <div class="input-group">
                <span class="input-group-text"><i class="bi bi-geo-alt"></i></span>
                <v-region-selects
                  v-model="regionValues"
                  :city="false"
                  :area="false"
                  :town="false"
                  class="flex-grow-1"
                  @change="handleRegionChange"
                />
              </div>
            </div>
          </div>

          <div class="modal-footer">
            <button
              type="button"
              class="wx-btn-outline"
              @click="handleSaveDraft"
              :disabled="isSubmitting || !formData.content.trim()"
            >
              <i class="bi bi-save me-1"></i>
              {{ isSubmitting ? '保存中...' : '保存草稿' }}
            </button>
            <button
              type="button"
              class="wx-btn-gradient"
              @click="handlePublish"
              :disabled="isSubmitting || !formData.content.trim()"
            >
              <i class="bi bi-send me-1"></i>
              {{ isSubmitting ? '保存中...' : '保存修改' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onBeforeUnmount, nextTick } from 'vue'
import { request, uploadImage as uploadImageUtil } from '@/utils/network'
import { toast } from '@/utils/app'
import * as bootstrap from 'bootstrap/dist/js/bootstrap.bundle.min.js'
import iEmojiPicker from '@/comps/custom/i-emoji-picker.vue'

const props = defineProps({
  editMoment: {
    type: Object,
    default: null
  },
  isDarkMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['published', 'draftSaved', 'cancelEdit'])

const MAX_IMAGES = 9

// 表单数据
const formData = ref({
  content: '',
  images: '',
  location: ''
})

const isSubmitting = ref(false)
const uploadingImage = ref(false)
const showImageUrlInput = ref(false)
const newImageUrl = ref('')
const regionValues = ref({})

// 富文本编辑器相关
const publishEditorRef = ref(null)
const editEditorRef = ref(null)
const showEmojiPicker = ref(false)
const activeEditorMode = ref('publish')

// 是否编辑模式
const isEditing = computed(() => !!props.editMoment)

// 图片列表
const imageList = computed(() => {
  const images = formData.value.images
  if (!images) return []
  if (Array.isArray(images)) return images.filter(Boolean)
  if (typeof images === 'string') return images.split(',').map(s => s.trim()).filter(Boolean)
  return []
})

// 弹窗实例管理
const modalEl = ref(null)
let modalInstance = null

const showModal = async () => {
  await nextTick()
  if (!modalEl.value) return
  if (!modalInstance) {
    modalInstance = new bootstrap.Modal(modalEl.value, { backdrop: true, keyboard: true })
    modalEl.value.addEventListener('hidden.bs.modal', () => {
      // 任意方式关闭（按钮/背景/ESC）都重置表单并通知父组件
      resetForm()
      emit('cancelEdit')
    })
  }
  modalInstance.show()
}

const hideModal = () => {
  if (modalInstance) modalInstance.hide()
}

// 监听编辑数据变化：进入编辑显示弹窗，退出编辑隐藏弹窗
watch(() => props.editMoment, (moment) => {
  if (moment) {
    formData.value = {
      content: moment.content || '',
      images: moment.images || '',
      location: moment.location || ''
    }
    activeEditorMode.value = 'edit'
    showModal()
    // 弹窗渲染后设置编辑器内容
    nextTick(() => {
      setTimeout(() => {
        if (editEditorRef.value) {
          editEditorRef.value.innerHTML = contentToHtml(formData.value.content)
        }
      }, 100)
    })
  } else {
    activeEditorMode.value = 'publish'
    hideModal()
  }
}, { immediate: true })

// ===== contenteditable 编辑器辅助函数 =====
// 将存储格式的内容转为HTML（[emoji:url] -> img）
const contentToHtml = (content) => {
  if (!content) return ''
  let html = content
  html = html.replace(/\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g, '<img src="$1" data-emoji="$1" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">')
  html = html.replace(/\n/g, '<br>')
  return html
}

// 从contenteditable div中提取纯文本内容，emoji图片转为[emoji:url]格式
const getEditorContent = (el) => {
  if (!el) return ''
  const clone = el.cloneNode(true)
  const imgs = clone.querySelectorAll('img[data-emoji]')
  imgs.forEach(img => {
    const url = img.getAttribute('data-emoji') || img.src || ''
    const textNode = document.createTextNode(`[emoji:${url}]`)
    img.parentNode.replaceChild(textNode, img)
  })
  return clone.innerText.trim()
}

const clearEditor = (el) => {
  if (!el) return
  el.innerHTML = ''
}

// 在光标位置插入HTML
const insertHTMLAtCursor = (html, editorEl) => {
  if (!editorEl) return
  editorEl.focus()
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) {
    editorEl.innerHTML += html
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

// 获取当前活跃编辑器
const getActiveEditor = () => {
  return activeEditorMode.value === 'edit' ? editEditorRef.value : publishEditorRef.value
}

// 内容输入事件
const onContentInput = () => {
  formData.value.content = getEditorContent(getActiveEditor())
}

// 表情功能
const toggleEmojiPicker = (mode) => {
  activeEditorMode.value = mode
  showEmojiPicker.value = !showEmojiPicker.value
}

const insertEmoji = (emoji) => {
  const editor = getActiveEditor()
  if (!editor) {
    formData.value.content += emoji
    return
  }
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  onContentInput()
}

onBeforeUnmount(() => {
  if (modalInstance) {
    modalInstance.dispose()
    modalInstance = null
  }
})

// 上传图片
const handleUploadImage = async () => {
  if (uploadingImage.value) return
  if (imageList.value.length >= MAX_IMAGES) {
    toast.warning(`最多只能上传 ${MAX_IMAGES} 张图片`)
    return
  }

  uploadingImage.value = true

  try {
    const imageUrl = await uploadImageUtil()
    if (imageUrl) {
      const currentImages = imageList.value
      currentImages.push(imageUrl)
      formData.value.images = currentImages.join(',')
      toast.success('图片上传成功')
    }
  } catch (err) {
    if (err.message === '已取消选择' || err.message === '未选择文件') {
      return
    }
    console.error('图片上传失败:', err)
    toast.error(err.message || '图片上传失败')
  } finally {
    uploadingImage.value = false
  }
}

// 添加图片URL
const addImageUrl = () => {
  const url = newImageUrl.value.trim()
  if (!url) return

  if (imageList.value.length >= MAX_IMAGES) {
    toast.warning(`最多只能上传 ${MAX_IMAGES} 张图片`)
    return
  }

  if (!/^https?:\/\//.test(url)) {
    toast.error('请输入有效的图片链接')
    return
  }

  const currentImages = imageList.value
  currentImages.push(url)
  formData.value.images = currentImages.join(',')
  newImageUrl.value = ''
  showImageUrlInput.value = false
  toast.success('图片添加成功')
}

// 移除图片
const removeImage = (index) => {
  const currentImages = imageList.value
  currentImages.splice(index, 1)
  formData.value.images = currentImages.join(',')
}

// 地区选择变化
const handleRegionChange = (data) => {
  if (data && data.province) {
    formData.value.location = data.province.value || ''
  }
}

// 保存草稿
const handleSaveDraft = async () => {
  if (!formData.value.content.trim()) {
    toast.error('请输入动态内容')
    return
  }

  isSubmitting.value = true

  try {
    const data = {
      content: formData.value.content,
      images: formData.value.images,
      location: formData.value.location,
      status: 0 // 草稿
    }

    let res
    if (isEditing.value) {
      data.id = props.editMoment.id
      res = await request.put('/api/moments/update', data)
    } else {
      res = await request.post('/api/moments/create', data)
    }

    if (res.code === 200) {
      toast.success('草稿保存成功')
      resetForm()
      emit('draftSaved', res.data)
    } else {
      toast.error(res.msg || '保存失败')
    }
  } catch (err) {
    console.error('保存草稿失败:', err)
    toast.error('保存失败，请稍后重试')
  } finally {
    isSubmitting.value = false
  }
}

// 发布动态
const handlePublish = async () => {
  if (!formData.value.content.trim()) {
    toast.error('请输入动态内容')
    return
  }

  isSubmitting.value = true

  try {
    const data = {
      content: formData.value.content,
      images: formData.value.images,
      location: formData.value.location,
      status: 1 // 发布
    }

    let res
    if (isEditing.value) {
      data.id = props.editMoment.id
      res = await request.put('/api/moments/update', data)
    } else {
      res = await request.post('/api/moments/create', data)
    }

    if (res.code === 200) {
      toast.success(isEditing.value ? '更新成功' : '发布成功')
      resetForm()
      emit('published', res.data)
    } else {
      toast.error(res.msg || '发布失败')
    }
  } catch (err) {
    console.error('发布动态失败:', err)
    toast.error('发布失败，请稍后重试')
  } finally {
    isSubmitting.value = false
  }
}

// 重置表单
const resetForm = () => {
  formData.value = {
    content: '',
    images: '',
    location: ''
  }
  regionValues.value = {}
  clearEditor(publishEditorRef.value)
  clearEditor(editEditorRef.value)
  showEmojiPicker.value = false
}

// 暴露方法供父组件调用
defineExpose({ resetForm })
</script>

<style scoped>
.moment-editor {
  margin-bottom: 1rem;
}

/* 表情按钮激活态 */
.emoji-button.active {
  color: var(--bs-primary);
  border-color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.08);
}

/* 图片预览网格 */
.image-preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.image-preview-item {
  position: relative;
  aspect-ratio: 1;
  border-radius: var(--wx-radius-sm);
  overflow: hidden;
  background-color: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  transition: var(--wx-transition);
}

.image-preview-item:hover {
  border-color: var(--bs-primary);
  box-shadow: var(--wx-shadow-sm);
  transform: translateY(-2px);
}

.image-preview-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.image-remove-btn {
  position: absolute;
  top: 0.375rem;
  right: 0.375rem;
  width: 1.5rem;
  height: 1.5rem;
  padding: 0;
  border-radius: var(--wx-radius-xl);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  box-shadow: var(--wx-shadow-sm);
}

/* 加载动画 */
.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* 编辑器内表单控件聚焦（统一为 wx 主色聚焦环） */
:deep(textarea:focus),
:deep(input:focus) {
  border-color: var(--bs-primary) !important;
  box-shadow: 0 0 0 2px rgba(var(--bs-primary-rgb), 0.12) !important;
  outline: none !important;
}

/* 输入组前缀图标统一配色 */
:deep(.input-group-text) {
  border-color: var(--bs-border-color);
  background: var(--bs-tertiary-bg);
  color: var(--bs-secondary-color);
}

/* 模态框圆角与阴影统一 */
:deep(.modal-content) {
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-lg);
  box-shadow: var(--wx-shadow-lg);
}

:deep(.modal-header) {
  border-bottom-color: var(--bs-border-color);
}

:deep(.modal-footer) {
  border-top-color: var(--bs-border-color);
}

/* 按钮过渡 */
:deep(.btn) {
  transition: var(--wx-transition);
}

/* 移动端适配 */
@media (max-width: 768px) {
  .image-preview-grid {
    grid-template-columns: repeat(auto-fill, minmax(80px, 1fr));
  }
}

/* ==========================================
 * 城市选择器深色模式适配
 * 触发按钮在组件内，用 :deep() 覆盖 v-region 默认硬编码色值
 * ========================================== */
:deep(.rg-selects) {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
}

:deep(.rg-selects .dd-trigger-container) {
  display: block;
  flex: 1 1 auto;
}

:deep(.dd-default-trigger) {
  width: 100%;
  justify-content: space-between;
  background-color: var(--bs-body-bg);
  border-color: var(--bs-border-color);
  color: var(--bs-body-color);
}

:deep(.dd-default-trigger:hover) {
  border-color: var(--bs-primary);
  color: var(--bs-body-color);
}

:deep(.dd-default-trigger.dd-opened) {
  border-color: var(--bs-primary);
  color: var(--bs-body-color);
  box-shadow: 0 0 0 2px rgba(var(--bs-primary-rgb), 0.12);
}
</style>

<style>
/* ==========================================
 * 城市选择器下拉浮层深色模式适配
 * .dd-content 通过 Teleport 挂载到 body，无法用 scoped 样式覆盖，需全局样式
 * 同时提升 z-index，确保在编辑弹窗之上正常显示
 * ========================================== */
.dd-content {
  z-index: 1080;
}

[data-bs-theme=dark] .dd-content {
  background-color: var(--bs-body-bg) !important;
  border-color: var(--bs-border-color) !important;
  box-shadow: 0 9px 24px rgba(0, 0, 0, 0.5), 0 3px 6px rgba(0, 0, 0, 0.3) !important;
}

[data-bs-theme=dark] .rg-select__list li {
  color: var(--bs-body-color);
}

[data-bs-theme=dark] .rg-select__list li:hover {
  background-color: var(--bs-tertiary-bg);
}

[data-bs-theme=dark] .rg-select__list li.selected {
  background-color: var(--bs-primary);
  color: var(--bs-white);
}
</style>
