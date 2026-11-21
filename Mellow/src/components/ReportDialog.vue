<template>
  <ConfirmDialog
    :visible="visible"
    title="举报"
    :message="message"
    confirm-text="提交举报"
    danger
    :loading="loading"
    @update:visible="emit('update:visible', $event)"
    @confirm="onConfirm"
    @cancel="onCancel"
  >
    <p class="report-target">
      <i class="bi bi-flag" aria-hidden="true" />
      举报对象：<strong>{{ targetLabel || typeText(targetType) }}</strong>
    </p>

    <div class="report-reason">
      <span class="report-reason-label">
        举报原因
        <em class="report-reason-tag is-required">必填</em>
      </span>
      <div class="report-reason-options">
        <button
          v-for="item in REPORT_REASONS"
          :key="item.value"
          type="button"
          class="chip"
          :class="{ 'is-active': reason === item.value }"
          :aria-pressed="reason === item.value"
          @click="reason = item.value"
        >{{ item.label }}</button>
      </div>
    </div>

    <label class="report-detail">
      <span class="report-detail-label">
        补充说明
        <em class="report-detail-tag">选填</em>
      </span>
      <FormTextarea
        v-model="detail"
        class="report-detail-input"
        rows="3"
        maxlength="200"
        placeholder="可补充具体情况（如时间、涉及哪一段内容），有助于我们判断" />
      <span class="report-detail-hint">{{ detail.length }}/200 · 仅管理员可见</span>
    </label>

    <!-- 证据图片：截图等直接上传（最多 3 张）。仅管理员可见，不会公开 -->
    <div class="report-images">
      <span class="report-images-label">
        证据图片
        <em class="report-images-tag">选填 · 最多 {{ MAX_IMAGES }} 张</em>
      </span>
      <div class="report-images-list">
        <div v-for="(img, i) in images" :key="img" class="report-thumb">
          <img :src="img" alt="举报证据" @click="previewImage(i)" loading="lazy" />
          <span class="report-thumb-del" title="移除" @click="removeImage(i)">×</span>
        </div>
        <button
          v-if="images.length < MAX_IMAGES"
          type="button"
          class="report-add"
          :disabled="uploading"
          @click="fileInput?.click()"
        >
          <i class="bi" :class="uploading ? 'bi-hourglass-split' : 'bi-image'" aria-hidden="true" />
          <span>{{ uploading ? '上传中…' : '添加图片' }}</span>
        </button>
      </div>
      <input
        ref="fileInput"
        type="file"
        accept="image/*"
        multiple
        class="hidden-file"
        @change="onPickImages"
      />
      <span class="report-images-hint">
        仅管理员可见；请上传能说明问题的截图，勿上传与举报无关的图片
      </span>
    </div>
  </ConfirmDialog>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
/**
 * 举报弹窗（动态 / 文章 / 评论 / 用户共用）
 *
 * 用法：
 *   <ReportDialog
 *     :visible="report.visible"
 *     :target-type="report.type"
 *     :target-id="report.id"
 *     :target-label="'动态「今天的风景」'"
 *     :loading="report.loading"
 *     @update:visible="(v) => (report.visible = v)"
 *     @confirm="submitReport"
 *   />
 *   - confirm 事件带上 { reason, detail }；提交请求由调用方负责（见 @/api/report）
 *   - 后端还会做一次校验：每天上限、同一对象只允许一条待处理，因此这里不做重复判断
 *
 * 隐私说明：举报提交后**不会**通知被举报人（只通知管理员），
 * 所以界面文案里也刻意不写「对方会知道」这类提示。
 */
import { ref, watch } from 'vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { REPORT_REASONS, REPORT_TARGETS, uploadReportImages } from '@/api/report'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

// 证据图片数量上限：与后端 reportMaxImages 保持一致
const MAX_IMAGES = 3
// 单张图片大小上限（与评论 / 动态图片一致）
const MAX_SIZE = 10 * 1024 * 1024

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 举报对象类型：moment / article / comment / user
  targetType: { type: String, default: '' },
  targetId: { type: [Number, String], default: 0 },
  // 对象描述（可选）：如「动态「今天的风景」」，不传则用类型名
  targetLabel: { type: String, default: '' },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

const reason = ref('')
const detail = ref('')
// 证据图片（上传成功后拿到的 URL 列表）
const images = ref([])
const uploading = ref(false)
const fileInput = ref(null)

const message = '请选择举报原因。恶意举报会影响他人正常使用，也会被记录。'

function typeText(type) {
  const item = REPORT_TARGETS.find((t) => t.value === type)
  return item ? item.label : '内容'
}

function previewImage(i) {
  openLightbox(images.value, i)
}

function removeImage(i) {
  images.value.splice(i, 1)
}

/**
 * 选择证据图片：类型 / 大小 / 数量三重校验后走 attachment/batch 上传（与评论、动态图片同一通道）。
 * 上传失败只提示，不阻断举报本身 —— 没有图片也能提交。
 */
async function onPickImages(e) {

  const files = Array.from(e.target.files || [])
  e.target.value = ''
  if (!files.length) return

  const rest = MAX_IMAGES - images.value.length
  if (rest <= 0) {
    toast.warning(`最多只能上传 ${MAX_IMAGES} 张证据图片`)
    return
  }

  const valid = files.filter((file) => {
    if (!file.type.startsWith('image/')) {
      toast.warning(`文件「${file.name}」不是图片`)
      return false
    }
    if (file.size > MAX_SIZE) {
      toast.warning(`图片「${file.name}」超过 10MB 限制`)
      return false
    }
    return true
  })
  if (!valid.length) return
  if (valid.length > rest) toast.warning('受数量限制，部分图片未上传')

  uploading.value = true
  try {
    const fd = new FormData()
    valid.slice(0, rest).forEach((file) => fd.append('files', file))
    const res = await uploadReportImages(fd)
    const urls = (res.data?.results || [])
      .filter((r) => r.status !== 'fail' && r.full_url)
      .map((r) => r.full_url)
    const add = urls.filter((url) => !images.value.includes(url))
    if (!add.length) {
      toast.warning('图片上传失败，请重试或直接提交举报')
    } else {
      images.value.push(...add)
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

// 每次打开都重置：避免上一次选的原因被带到下一条举报上
watch(
  () => props.visible,
  (val) => {
    if (val) {
      reason.value = ''
      detail.value = ''
      images.value = []
    }
  }
)

function onConfirm() {
  if (!reason.value) {
    toast.error('请选择举报原因')
    return
  }
  emit('confirm', {
    reason: reason.value,
    detail: detail.value.trim(),
    images: [...images.value]
  })
}

function onCancel() {
  reason.value = ''
  detail.value = ''
  images.value = []
  emit('cancel')
}
</script>

<style scoped>
.report-target {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 12px;
  padding-bottom: 10px;
  font-size: 13px;
  color: var(--text-soft);
  border-bottom: 1px dashed var(--border-soft);
}
.report-reason-label,
.report-detail-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.report-reason-tag,
.report-detail-tag {
  padding: 0 6px;
  font-size: 11px;
  font-style: normal;
  font-weight: 500;
  border-radius: 999px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.report-reason-tag.is-required {
  color: var(--danger);
  background: rgba(217, 84, 77, 0.12);
}
.report-reason-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 8px 0 14px;
}
/*
 * chip 视觉：全局没有 .chip（项目里每个用到的地方都在自己 scoped 里声明一份，
 * 见 views/goods/Index.vue、views/notifications/Index.vue），这里必须自带，
 * 否则按钮点得动但没有任何选中反馈（表现就是「点了没反应」）。
 */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-family: inherit;
  font-size: 13px;
  line-height: 1.5;
  cursor: pointer;
  transition: all 0.18s;
}
.chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: 0 0 0 2px var(--accent-ring);
}
.report-detail {
  display: block;
}
.report-detail-input {
  margin-top: 8px;
  width: 100%;
  font-size: 13px;
  resize: vertical;
}
.report-detail-hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 证据图片 ---------- */
.report-images {
  margin-top: 14px;
}
.report-images-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.report-images-tag {
  padding: 0 6px;
  font-size: 11px;
  font-style: normal;
  font-weight: 500;
  border-radius: 999px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.report-images-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.report-thumb {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border);
}
.report-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  cursor: zoom-in;
}
.report-thumb-del {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  line-height: 1;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 50%;
  cursor: pointer;
}
.report-thumb-del:hover {
  background: var(--danger);
}
.report-add {
  width: 64px;
  height: 64px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-family: inherit;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.18s;
}
.report-add:hover:not(:disabled) {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.report-add:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}
.report-images-hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
}
.hidden-file {
  display: none;
}
</style>
