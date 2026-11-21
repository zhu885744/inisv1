<template>
  <div class="cert-fields">
    <div v-if="!list.length" class="cert-fields-empty">
      还没有字段，申请页将没有可填项。至少配置一个（例如「认证名称」）。
    </div>

    <div v-for="(field, index) in list" :key="index" class="cf-card">
      <header class="cf-head">
        <span class="cf-index">{{ index + 1 }}</span>
        <FormInput
          v-model="field.label"
          class="cf-label"
          type="text"
          maxlength="20"
          placeholder="字段名称，如「公司 / 组织」" />
        <div class="cf-type">
          <FormSelect v-model="field.type" :options="CERT_FIELD_TYPES" />
        </div>
        <label class="cf-check">
          <FormCheckbox v-model="field.required" />
          必填
        </label>
        <button class="btn btn-ghost btn-sm" type="button" title="删除该字段" @click="remove(index)">
          <i class="bi bi-x-lg" />
        </button>
      </header>

      <div class="cf-grid">
        <div class="form-item">
          <label class="form-label">存储键</label>
          <FormInput v-model="field.key" type="text" maxlength="32" placeholder="英文 / 数字 / 下划线，如 company" />
          <p v-if="keyError(index)" class="cf-warn"><i class="bi bi-exclamation-triangle" /> {{ keyError(index) }}</p>
          <p v-else-if="CERT_SYSTEM_KEYS[field.key]" class="form-hint">{{ CERT_SYSTEM_KEYS[field.key] }}</p>
        </div>
        <div class="form-item">
          <label class="form-label">输入提示</label>
          <FormInput v-model="field.placeholder" type="text" maxlength="64" placeholder="可选，显示在输入框内" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">字段说明</label>
        <FormInput
          v-model="field.hint"
          type="text"
          maxlength="100"
          placeholder="可选，显示在输入框下方（说明用途，不要写营销话术）" />
      </div>

      <!-- 图片：张数上限 -->
      <div v-if="field.type === 'image'" class="form-item cf-inline">
        <label class="form-label">最多上传</label>
        <FormInput v-model.number="field.max_count" class="cf-num" type="number" min="1" max="12" />
        <span class="form-hint">张（1 ~ 12，留空按 6）</span>
      </div>

      <!-- 下拉：候选值 -->
      <div v-if="field.type === 'select'" class="form-item">
        <label class="form-label">候选值</label>
        <div v-if="(field.options || []).length" class="cf-options">
          <span v-for="(option, i) in field.options" :key="`${option}-${i}`" class="cf-option">
            {{ option }}
            <button type="button" :title="`移除「${option}」`" @click="field.options.splice(i, 1)">
              <i class="bi bi-x" />
            </button>
          </span>
        </div>
        <div class="cf-option-add">
          <FormInput
            v-model="drafts[index]"
            type="text"
            maxlength="32"
            placeholder="输入候选值后回车添加"
            @keydown.enter.prevent="addOption(index)" />
          <button class="btn btn-sm" type="button" @click="addOption(index)">添加</button>
        </div>
        <p v-if="field.key === 'category'" class="form-hint">
          留空时自动使用本类型的「认证领域」列表，改领域不用回来同步这里
        </p>
      </div>

      <!-- 文本类：长度上限 -->
      <div v-if="field.type === 'text' || field.type === 'textarea'" class="form-item cf-inline">
        <label class="form-label">最大字符数</label>
        <FormInput v-model.number="field.max" class="cf-num" type="number" min="0" max="2000" placeholder="0 = 默认" />
        <span class="form-hint">0 表示按类型默认（单行 64 / 多行 500）</span>
      </div>

      <div class="cf-flags">
        <label class="cf-check">
          <FormCheckbox v-model="field.public" />
          公开展示
        </label>
        <label class="cf-check">
          <FormCheckbox v-model="field.admin_only" />
          仅后台可见
        </label>
        <label v-if="field.type === 'select'" class="cf-check">
          <FormCheckbox v-model="field.multiple" />
          允许多选
        </label>
      </div>
    </div>

    <div class="cf-foot">
      <button class="btn btn-sm" type="button" @click="add"><i class="bi bi-plus-lg" /> 添加字段</button>
      <span class="form-hint">字段顺序 = 申请页的填写顺序；「公开展示」只影响用户主页，不影响后台审核</span>
    </div>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 认证申请字段编辑器（后台「认证类型」页复用）
 *
 * 一行 = 一个申请字段，结构对齐后端 model.CertField：
 *   { key, label, type, required, placeholder, hint, max, options, multiple, max_count, public, admin_only }
 *
 * 设计口径：
 *   - key 是用户提交值的存储键，同类型内不能重复（重复时行内直接提示，保存前也会拦）；
 *   - title / category / images 是内置键，有特殊语义（写回记录独立列），行内给提示但不禁止改名；
 *   - 其它键的值进记录的 data JSON，前台只展示「公开展示」且非「仅后台可见」的项。
 */
import { computed, reactive } from 'vue'
import { CERT_FIELD_TYPES, CERT_SYSTEM_KEYS } from '@/api/cert-type'

const props = defineProps({
  // 字段数组：由父组件持有，编辑时原位更新
  modelValue: { type: Array, default: () => [] }
})

const emit = defineEmits(['update:modelValue'])

const list = computed(() => props.modelValue || [])

// 候选值输入草稿（按行索引存，避免每行都挂一个临时字段到数据里）
const drafts = reactive({})

// 存储键格式 / 重复校验：前后端同一口径（中文键名写进 JSON 也能存，但后续排查很痛苦）
function keyError(index) {
  const field = list.value[index]
  if (!field) return ''
  const key = String(field.key || '').trim()
  if (!key) return '存储键不能为空'
  if (!/^[a-zA-Z_][a-zA-Z0-9_]*$/.test(key)) return '存储键只能用英文 / 数字 / 下划线，且不能以数字开头'
  const duplicated = list.value.some((item, i) => i !== index && String(item.key || '').trim() === key)
  if (duplicated) return '存储键重复'
  return ''
}

function addOption(index) {
  const field = list.value[index]
  if (!field) return
  const value = String(drafts[index] || '').trim()
  if (!value) return
  if (!Array.isArray(field.options)) field.options = []
  if (field.options.includes(value)) {
    drafts[index] = ''
    return
  }
  field.options.push(value)
  drafts[index] = ''
  emit('update:modelValue', list.value)
}

function add() {
  list.value.push({
    key: '',
    label: '',
    type: 'text',
    required: false,
    placeholder: '',
    hint: '',
    max: 0,
    options: [],
    multiple: false,
    max_count: 0,
    public: false,
    admin_only: false
  })
  emit('update:modelValue', list.value)
}

function remove(index) {
  list.value.splice(index, 1)
  delete drafts[index]
  emit('update:modelValue', list.value)
}

/** 父组件保存前调用：返回第一条错误（空串表示通过） */
defineExpose({
  validate() {
    if (!list.value.length) return '至少配置一个需要用户填写的字段'
    for (let index = 0; index < list.value.length; index += 1) {
      const field = list.value[index]
      const msg = keyError(index)
      if (msg) return `第 ${index + 1} 个字段：${msg}`
      if (!String(field.label || '').trim()) return `第 ${index + 1} 个字段：名称不能为空`
    }
    return ''
  }
})
</script>

<style scoped>
.cert-fields {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cert-fields-empty {
  padding: 14px;
  font-size: 12.5px;
  color: var(--text-light);
  text-align: center;
  background: var(--bg-alt);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
}
.cf-card {
  padding: 12px;
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.cf-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.cf-index {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 50%;
}
.cf-label {
  flex: 1;
  min-width: 0;
}
.cf-type {
  width: 120px;
  flex-shrink: 0;
}
.cf-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.cf-inline {
  display: flex;
  align-items: center;
  gap: 8px;
}
.cf-inline .form-label {
  margin-bottom: 0;
  flex-shrink: 0;
}
.cf-num {
  width: 84px;
}
.cf-check {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-muted);
  cursor: pointer;
  white-space: nowrap;
}
.cf-check input {
  margin: 0;
}
.cf-flags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  padding-top: 8px;
  margin-top: 2px;
  border-top: 1px dashed var(--border-soft);
}
.cf-warn {
  margin: 4px 0 0;
  font-size: 11.5px;
  color: var(--danger);
}
.cf-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.cf-option {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 4px 2px 8px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
}
.cf-option button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--text-light);
  font-size: 10px;
  cursor: pointer;
}
.cf-option button:hover {
  color: var(--danger);
}
.cf-option-add {
  display: flex;
  gap: 8px;
}
.cf-foot {
  display: flex;
  align-items: center;
  gap: 10px;
}
@media (max-width: 640px) {
  .cf-grid {
    grid-template-columns: 1fr;
  }
  .cf-head {
    flex-wrap: wrap;
  }
  .cf-type {
    width: 100%;
  }
}
</style>
