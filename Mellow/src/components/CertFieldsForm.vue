<template>
  <div class="cff">
    <div v-for="field in fields" :key="field.key" class="cff-row">
      <label class="cff-label">
        {{ field.label }}<span v-if="field.required" class="cff-req">*</span>
        <em v-if="field.hint">{{ field.hint }}</em>
      </label>

      <!-- 单行文本 / 链接 -->
      <FormInput
        v-if="field.type === 'text' || field.type === 'link'"
        v-model="values[field.key]"
        type="text"
        :maxlength="field.max || (field.type === 'link' ? 512 : 64)"
        :placeholder="field.placeholder || ''" />

      <!-- 多行文本 -->
      <FormTextarea
        v-else-if="field.type === 'textarea'"
        v-model="values[field.key]"
        rows="3"
        :maxlength="field.max || 500"
        :placeholder="field.placeholder || ''" />

      <!-- 数字 / 日期 -->
      <FormInput
        v-else-if="field.type === 'number'"
        v-model="values[field.key]"
        class="cff-num"
        type="number"
        :placeholder="field.placeholder || ''" />
      <FormInput
        v-else-if="field.type === 'date'"
        v-model="values[field.key]"
        class="cff-num"
        type="date" />

      <!-- 单选下拉 -->
      <FormSelect
        v-else-if="field.type === 'select' && !field.multiple"
        v-model="values[field.key]"
        :options="optionsOf(field)"
        :placeholder="field.placeholder || '请选择'" />

      <!-- 多选：SelectMenu 不支持多选，这里用可点选的标签 -->
      <div v-else-if="field.type === 'select' && field.multiple" class="cff-chips">
        <button
          v-for="option in optionsOf(field)"
          :key="option.value"
          type="button"
          class="chip"
          :class="{ 'is-active': (values[field.key] || []).includes(option.value) }"
          @click="toggleOption(field.key, option.value)"
        >
          {{ option.label }}
        </button>
      </div>

      <!-- 图片 -->
      <div v-else-if="field.type === 'image'" class="cff-images">
        <div v-for="(img, index) in values[field.key] || []" :key="img" class="cff-image">
          <img :src="img" alt="已上传的图片" loading="lazy" @click="openLightbox(values[field.key], index)" />
          <button type="button" class="cff-image-del" title="移除" @click="removeImage(field.key, index)">
            <i class="bi bi-x" />
          </button>
        </div>
        <button
          v-if="(values[field.key] || []).length < maxCountOf(field)"
          type="button"
          class="cff-upload"
          :disabled="uploading"
          @click="pickFiles(field.key)"
        >
          <i class="bi" :class="uploading ? 'bi-hourglass-split' : 'bi-plus-lg'" />
          <span>{{ uploading ? '上传中' : '添加图片' }}</span>
        </button>
      </div>
    </div>

    <!-- 图片上传共用一个 file input：点哪个图片字段就记住 key -->
    <input ref="fileInput" type="file" accept="image/*" multiple hidden @change="onFilesChange" />
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 认证字段表单（用户申请页 / 后台改资料 共用）
 *
 * 只做一件事：按后台配置的字段定义渲染输入控件，值写回 modelValue（{字段key: 值}）。
 * 不负责校验与提交——必填规则由调用方决定（用户提交强制必填，后台改资料允许清空），
 * 后端始终按同一份字段定义做格式校验，所以两端不会出现「前端能填、后端不接受」的错位。
 *
 * 字段类型：text / textarea / number / select（含多选）/ date / link / image。
 */
import { computed, ref } from 'vue'
import { uploadCertificationImages } from '@/api/user-certification'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const props = defineProps({
  // 字段定义数组（来自 cert-type 配置）
  fields: { type: Array, default: () => [] },
  // 字段值：{字段key: 值}，图片为 URL 数组，多选为字符串数组
  modelValue: { type: Object, default: () => ({}) },
  // 认证领域候选，供 key=category 的 select 兜底
  categories: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue', 'update:uploading'])

// 用 computed 而不是捕获一次：父组件切换认证类型时会整体替换 values 对象，
// 捕获引用会让表单继续写进已被丢弃的旧对象（表现为「切了类型但值没变 / 改了不生效」）
const values = computed(() => props.modelValue || {})
const fileInput = ref(null)
const uploadKey = ref('')
const uploading = ref(false)

function optionsOf(field) {
  if (field.key === 'category' && !(field.options || []).length) {
    return (props.categories || []).map((item) => ({ value: item, label: item }))
  }
  return (field.options || []).map((item) => ({ value: item, label: item }))
}

function maxCountOf(field) {
  return Number(field.max_count) > 0 ? Number(field.max_count) : 6
}

function toggleOption(key, value) {
  const list = Array.isArray(values.value[key]) ? [...values.value[key]] : []
  const index = list.indexOf(value)
  if (index >= 0) {
    list.splice(index, 1)
  } else {
    list.push(value)
  }
  values.value[key] = list
  emit('update:modelValue', values.value)
}

function removeImage(key, index) {
  values.value[key]?.splice(index, 1)
  emit('update:modelValue', values.value)
}

function pickFiles(key) {
  uploadKey.value = key
  fileInput.value?.click()
}

async function onFilesChange(e) {
  const files = [...(e.target.files || [])]
  e.target.value = ''
  if (!files.length) return

  const field = props.fields.find((item) => item.key === uploadKey.value)
  if (!field) return

  const current = Array.isArray(values.value[field.key]) ? values.value[field.key] : []
  const room = maxCountOf(field) - current.length
  if (room <= 0) {
    toast.warning(`最多上传 ${maxCountOf(field)} 张`)
    return
  }

  const picked = []
  for (const file of files.slice(0, room)) {
    if (!file.type.startsWith('image/')) {
      toast.warning('只能上传图片')
      continue
    }
    if (file.size > 10 * 1024 * 1024) {
      toast.warning(`「${file.name}」超过 10MB`)
      continue
    }
    picked.push(file)
  }
  if (!picked.length) return

  uploading.value = true
  emit('update:uploading', true)
  try {
    const formData = new FormData()
    for (const file of picked) formData.append('files', file, file.name)
    const res = await uploadCertificationImages(formData)
    const urls = (res.data?.results || [])
      .filter((item) => item.status !== 'fail' && item.full_url)
      .map((item) => item.full_url)
    if (!urls.length) {
      toast.error('上传失败，请重试')
      return
    }
    values.value[field.key] = [...current, ...urls.slice(0, room)]
    emit('update:modelValue', values.value)
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
    emit('update:uploading', false)
  }
}
</script>

<style scoped>
.cff {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.cff-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.cff-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}
.cff-label em {
  font-style: normal;
  font-weight: 400;
  color: var(--text-light);
  margin-left: 6px;
}
.cff-req {
  color: var(--danger);
  margin-left: 2px;
}
.cff-num {
  max-width: 220px;
}
.cff-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  display: inline-flex;
  align-items: center;
  height: var(--control-h-sm);
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-family: inherit;
  font-size: 12px;
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
}
.cff-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.cff-image {
  position: relative;
  width: 84px;
  height: 84px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border-soft);
}
.cff-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  cursor: zoom-in;
}
.cff-image-del {
  position: absolute;
  top: 2px;
  right: 2px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 12px;
  cursor: pointer;
}
.cff-upload {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 84px;
  height: 84px;
  font-size: 11.5px;
  color: var(--text-light);
  background: var(--bg-alt);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.cff-upload:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary-soft);
}
</style>
