<template>
  <div class="cert-page">
    <SectionTitle title="我的认证" />

    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

    <template v-else>
      <!-- 状态卡：待审核 / 已驳回 / 已通过 都有对应结论，避免用户重复提交 -->
      <div v-if="record" class="card card-pad cert-status" :class="`is-${record.status}`">
        <div class="cert-status-head">
          <VerifiedBadge v-if="record.status === 1" :cert="record" size="md" />
          <i v-else class="bi" :class="record.status === 0 ? 'bi-hourglass-split' : 'bi-exclamation-circle'" />
          <strong>{{ statusTitle }}</strong>
          <span class="cert-status-tag">{{ record.cert_type_text }}</span>
        </div>
        <p class="cert-status-desc">{{ statusDesc }}</p>
        <p v-if="record.status === 2 && record.reason" class="cert-reason">
          驳回原因：{{ record.reason }}
        </p>
      </div>

      <!-- 已通过：展示当前认证信息（与用户主页一致的展示口径） -->
      <div v-if="record?.status === 1" class="card card-pad cert-card">
        <div class="cert-card-title">
          <VerifiedBadge :cert="record" show-text />
        </div>
        <dl class="cert-dl">
          <div><dt>认证类型</dt><dd>{{ record.cert_type_text }}</dd></div>
          <div><dt>认证领域</dt><dd>{{ record.category || '—' }}</dd></div>
          <div><dt>有效期</dt><dd>{{ expireText }}</dd></div>
          <div><dt>通过时间</dt><dd>{{ formatDate(record.handle_time) }}</dd></div>
        </dl>

        <!-- 后台配置为「公开展示」的字段（公司 / 职位等），敏感项不下发 -->
        <dl v-if="publicFields.length" class="cert-dl">
          <div v-for="field in publicFields" :key="field.key">
            <dt>{{ field.label }}</dt>
            <dd>{{ field.text || '—' }}</dd>
          </div>
        </dl>

        <!-- 认证说明：由后台审核时填写，用户端只读 -->
        <p v-if="record.description" class="cert-desc">
          <span class="cert-desc-label">认证说明</span>{{ record.description }}
        </p>

        <p class="cert-tip">认证信息会展示在你的个人主页昵称旁；资料有变可重新提交，重新提交后需再次审核。</p>
      </div>

      <!-- 申请表单：字段全部来自后台配置（类型 / 领域 / 必填 / 选项都可改） -->
      <div v-if="showForm && !noTypes" class="card card-pad cert-form-card">
        <header class="cert-form-head">
          <h3>{{ record ? '修改认证资料' : '提交认证申请' }}</h3>
          <p>认证通过后，你的主页昵称旁会展示认证标识，访客可查看认证类型与认证说明。</p>
        </header>

        <div class="cert-form">
          <div class="form-row">
            <label>认证类型<span class="required">*</span></label>
            <FormSelect v-model="form.cert_type" :options="typeOptions" placeholder="请选择认证类型" />
            <p v-if="activeType?.intro" class="form-hint">{{ activeType.intro }}</p>
          </div>

          <!-- 认证说明：后台填写，这里只展示 -->
          <div v-if="activeType?.notice" class="cert-notice">
            <i class="bi bi-info-circle" />
            <div>
              <strong>认证说明</strong>
              <p>{{ activeType.notice }}</p>
            </div>
          </div>

          <!-- 申请条件：门槛由后台配置，未达标直接拦住，不让用户白填一整张表 -->
          <div v-if="conditionItems.length" class="cert-conditions">
            <div class="cc-head">
              <i class="bi bi-shield-check" />
              <strong>申请条件</strong>
              <span v-if="!conditionKnown" class="cc-tag">登录后可查看进度</span>
              <span v-else class="cc-tag" :class="conditionsMet ? 'is-ok' : 'is-no'">
                {{ conditionsMet ? '已满足，可直接申请' : '暂未满足' }}
              </span>
            </div>
            <ul v-if="conditionKnown" class="cc-list">
              <li v-for="item in conditionItems" :key="item.key" :class="{ 'is-ok': item.ok }">
                <span class="cc-label">{{ item.label }}</span>
                <span class="cc-value">{{ item.current }} / {{ item.need }} {{ item.unit }}</span>
                <span v-if="!item.ok" class="cc-short">还差 {{ item.short }} {{ item.unit }}</span>
                <i v-else class="bi bi-check2" />
              </li>
            </ul>
          </div>

          <!-- 字段渲染与后台改资料共用同一组件，避免两端口径漂移 -->
          <CertFieldsForm
            v-model="form.values"
            :fields="activeFields"
            :categories="activeType?.categories || []"
            @update:uploading="(value) => (uploading = value)"
          />

          <div class="cert-actions">
            <button type="button" class="btn btn-primary" :disabled="submitting || uploading || !canSubmit" @click="submit">
              <i class="bi bi-send" /> {{ record ? '重新提交审核' : '提交申请' }}
            </button>
            <button v-if="editing" type="button" class="btn btn-ghost" :disabled="submitting" @click="cancelEdit">
              取消
            </button>
          </div>
          <p v-if="!canSubmit" class="cert-block-hint">
            当前未达到该类型的申请条件，达成后条件会自动满足（无需重复提交）。
          </p>
        </div>
      </div>

      <!-- 没有可用类型：不展示表单，避免用户对着空表单发呆 -->
      <div v-else-if="noTypes" class="card card-pad cert-empty">
        <i class="bi bi-patch-question" />
        <p>站点暂未开放认证申请，请稍后再来。</p>
      </div>

      <!-- 待审核：不直接展示表单，避免重复提交 -->
      <div v-else-if="record?.status === 0" class="cert-actions-center">
        <button type="button" class="btn btn-ghost" :disabled="submitting" @click="editing = true">
          <i class="bi bi-pencil" /> 修改资料并重新提交
        </button>
      </div>
    </template>
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
/**
 * 我的认证（/user/verification）
 *
 * 表单完全由后台「认证类型」配置驱动（api/cert-type）：
 *   - 认证类型下拉 = 启用中的类型；切换类型会重建表单（不同类型字段不同）；
 *   - 每个字段按 type 渲染（text / textarea / number / select / date / link / image），
 *     必填、占位、说明、选项、图片张数上限全部读配置；
 *   - 「认证说明」是后台填写的文案，这里只做展示，用户没有任何输入入口。
 *
 * 状态机（后端 user-certification/mine 返回的 status）：
 *   null → 未申请：直接展示表单
 *   0    → 待审核：展示状态卡 + 「修改资料并重新提交」（默认不展开表单，避免重复提交）
 *   1    → 已通过：展示认证信息卡 + 重新提交入口（重新提交会回到待审核）
 *   2    → 已驳回：展示原因 + 表单（预填上次填写的内容）
 */
import { computed, onMounted, reactive, ref, watch } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import VerifiedBadge from '@/components/VerifiedBadge.vue'
import CertFieldsForm from '@/components/CertFieldsForm.vue'
import { applyCertification, getMyCertification } from '@/api/user-certification'
import { CERT_CONDITIONS, getEnabledCertTypes } from '@/api/cert-type'
import { formatDate } from '@/utils/time'
import { toast } from '@/utils/toast'

const loading = ref(false)
const submitting = ref(false)
const uploading = ref(false)
const editing = ref(false)
const record = ref(null)
const types = ref([])
const form = reactive({
  cert_type: null,
  values: {}
})

const typeOptions = computed(() => types.value.map((item) => ({ value: Number(item.id), label: item.name })))

const activeType = computed(() => types.value.find((item) => Number(item.id) === Number(form.cert_type)) || null)

const activeFields = computed(() => activeType.value?.fields || [])

const noTypes = computed(() => !types.value.length)

// 申请条件：门槛来自类型配置，进度来自 cert-type/enable 返回的 stats
const conditionItems = computed(() => {
  const need = activeType.value?.conditions || {}
  const stats = activeType.value?.stats || {}

  return CERT_CONDITIONS.filter((item) => Number(need[item.key]) > 0).map((item) => {
    const needValue = Number(need[item.key])
    const current = Number(stats[item.key]) || 0
    return { ...item, need: needValue, current, short: Math.max(0, needValue - current), ok: current >= needValue }
  })
})

// 未登录时 enable 不返回 stats：此时不做判定，避免把「未登录」显示成「条件不达标」
const conditionKnown = computed(() => Object.keys(activeType.value?.stats || {}).length > 0)
const conditionsMet = computed(() => conditionItems.value.every((item) => item.ok))
const canSubmit = computed(() => !conditionKnown.value || conditionsMet.value)

// 公开展示的字段（已通过卡片）：跳过图片（单独渲染）与仅后台可见项
const publicFields = computed(() => {
  const rows = record.value?.fields || []
  return rows.filter((item) => item.public && !item.admin_only && item.type !== 'image' && item.text)
})

const showForm = computed(() => !record.value || record.value.status !== 0 || editing.value)

const statusTitle = computed(() => {
  if (!record.value) return ''
  if (record.value.status === 0) return '认证审核中'
  if (record.value.status === 1) return '已通过认证'
  return '认证未通过'
})

const statusDesc = computed(() => {
  if (!record.value) return ''
  if (record.value.status === 0) return '已提交，管理员通常会在 1-3 个工作日内完成审核，结果会通过站内信通知你。'
  if (record.value.status === 1) return '认证信息已展示在你的个人主页，访客可以在你的昵称旁看到认证标识。'
  return '本次申请未通过，可按驳回原因修改资料后重新提交。'
})

const expireText = computed(() => {
  const expire = Number(record.value?.expire_time) || 0
  if (!expire) return '长期有效'
  return formatDate(expire)
})

function defaultValue(field) {
  if (field.type === 'image') return []
  if (field.type === 'select' && field.multiple) return []
  return ''
}

function toImageList(value) {
  if (Array.isArray(value)) return value.filter(Boolean)
  const text = String(value || '').trim()
  if (!text) return []
  return text.split(',').map((item) => item.trim()).filter(Boolean)
}

function toMultiList(value) {
  if (Array.isArray(value)) return value.filter(Boolean)
  const text = String(value || '').trim()
  if (!text) return []
  return text.split(',').map((item) => item.trim()).filter(Boolean)
}

/** 按当前类型的字段重建 values（切换类型 / 打开编辑时都要重置，避免残留别的类型的键） */
function resetValues(item) {
  const next = {}
  for (const field of activeFields.value) {
    next[field.key] = defaultValue(field)
  }

  if (item) {
    // 后端 fields 已经做过 data + 老列回落，优先用它
    const rows = item.fields || []
    for (const field of activeFields.value) {
      const row = rows.find((entry) => entry.key === field.key)
      let value = row?.value
      if (value === undefined || value === null || value === '') {
        value = item[field.key]
      }
      if (field.type === 'image') {
        next[field.key] = toImageList(value)
      } else if (field.type === 'select' && field.multiple) {
        next[field.key] = toMultiList(value)
      } else if (field.type === 'number') {
        next[field.key] = value === undefined || value === null || value === '' ? '' : String(value)
      } else if (field.type === 'date') {
        next[field.key] = String(value || '').slice(0, 10)
      } else {
        next[field.key] = String(value || '')
      }
    }
  }

  form.values = next
}

async function load() {
  loading.value = true
  try {
    const [typeRes, mineRes] = await Promise.all([
      getEnabledCertTypes().catch(() => ({ data: [] })),
      getMyCertification().catch(() => ({ data: null }))
    ])

    types.value = typeRes.data || []
    record.value = mineRes.data || null

    const recordType = Number(record.value?.cert_type) || 0
    const available = types.value.some((item) => Number(item.id) === recordType)
    // 认证类型可能已被后台停用 / 删除：优先用记录自身的类型，保证「我的认证」还能正常展示与修改
    form.cert_type = available ? recordType : Number(types.value[0]?.id) || null

    resetValues(record.value)
  } finally {
    loading.value = false
  }
}

// 切换类型：清空表单（不同类型的字段完全不同，保留旧值没有意义）
watch(
  () => form.cert_type,
  (value, old) => {
    if (old === undefined || old === null || Number(value) === Number(old)) return
    resetValues(null)
  }
)

function cancelEdit() {
  editing.value = false
  resetValues(record.value)
}

function validate() {
  if (!activeType.value) {
    toast.warning('请选择认证类型')
    return false
  }

  if (!canSubmit.value) {
    toast.warning('当前未达到该类型的申请条件')
    return false
  }

  for (const field of activeFields.value) {
    const value = form.values[field.key]

    if (field.type === 'image') {
      const list = Array.isArray(value) ? value : []
      if (field.required && !list.length) {
        toast.warning(`请上传「${field.label}」`)
        return false
      }
      continue
    }

    if (field.type === 'select' && field.multiple) {
      const list = Array.isArray(value) ? value : []
      if (field.required && !list.length) {
        toast.warning(`请选择「${field.label}」`)
        return false
      }
      continue
    }

    const text = String(value ?? '').trim()
    if (field.required && !text) {
      toast.warning(`请填写「${field.label}」`)
      return false
    }
    if (text && field.type === 'link' && !/^https?:\/\//i.test(text)) {
      toast.warning(`「${field.label}」需要以 http:// 或 https:// 开头`)
      return false
    }
  }

  return true
}

function buildValues() {
  const values = {}
  for (const field of activeFields.value) {
    const value = form.values[field.key]

    if (field.type === 'image' || (field.type === 'select' && field.multiple)) {
      const list = Array.isArray(value) ? value.filter(Boolean) : []
      if (list.length) values[field.key] = list
      continue
    }

    if (field.type === 'number') {
      const text = String(value ?? '').trim()
      if (text !== '') values[field.key] = Number(text)
      continue
    }

    const text = String(value ?? '').trim()
    if (text) values[field.key] = text
  }
  return values
}

async function submit() {
  if (!validate()) return

  submitting.value = true
  try {
    await applyCertification({
      cert_type: Number(form.cert_type),
      values: buildValues()
    })
    toast.success('已提交，等待管理员审核')
    editing.value = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.cert-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.cert-status-head {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  color: var(--text);
}
.cert-status-head .bi {
  font-size: 16px;
}
.cert-status.is-0 .cert-status-head .bi {
  color: var(--warning);
}
.cert-status.is-1 .cert-status-head .bi {
  color: var(--success);
}
.cert-status.is-2 .cert-status-head .bi {
  color: var(--danger);
}
.cert-status-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 1px 7px;
  border-radius: 999px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.cert-status-desc {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}
.cert-reason {
  margin: 8px 0 0;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--danger);
  background: rgba(217, 84, 77, 0.08);
  border-radius: var(--radius-sm);
}
.cert-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.cert-dl {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px 18px;
  margin: 0;
}
.cert-dl + .cert-dl {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--border-soft);
}
.cert-dl div {
  display: flex;
  gap: 8px;
  font-size: 13px;
}
.cert-dl dt {
  color: var(--text-light);
  flex-shrink: 0;
}
.cert-dl dd {
  margin: 0;
  color: var(--text);
  overflow-wrap: break-word;
}
.cert-desc {
  margin: 12px 0 0;
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-alt);
  border-radius: var(--radius-sm);
}
.cert-desc-label {
  display: block;
  margin-bottom: 4px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-light);
}
.cert-tip {
  margin: 14px 0 0;
  font-size: 12px;
  color: var(--text-light);
}
.cert-form-head h3 {
  margin: 0 0 6px;
  font-size: 15px;
  color: var(--text);
}
.cert-form-head p {
  margin: 0 0 16px;
  font-size: 12.5px;
  color: var(--text-light);
}
.cert-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.cert-notice {
  display: flex;
  gap: 8px;
  padding: 10px 12px;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--accent-soft);
  border-radius: var(--radius-sm);
}
.cert-notice .bi {
  margin-top: 2px;
  color: var(--primary);
}
.cert-notice strong {
  display: block;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--primary-deep);
}
.cert-notice p {
  margin: 0;
  white-space: pre-wrap;
}
.form-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.form-row > label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}
.form-row > label em {
  font-style: normal;
  font-weight: 400;
  color: var(--text-light);
  margin-left: 6px;
}
.required {
  color: var(--danger);
  margin-left: 2px;
}
.form-hint {
  margin: 0;
  font-size: 11.5px;
  color: var(--text-light);
}
.cert-conditions {
  padding: 10px 12px;
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.cc-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-muted);
}
.cc-head .bi {
  color: var(--primary);
}
.cc-tag {
  margin-left: auto;
  padding: 1px 8px;
  font-size: 11.5px;
  color: var(--text-light);
  background: var(--bg-muted);
  border-radius: 999px;
}
.cc-tag.is-ok {
  color: var(--success);
  background: rgba(58, 160, 106, 0.12);
}
.cc-tag.is-no {
  color: var(--warning);
  background: var(--gold-wash);
}
.cc-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
}
.cc-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.cc-list li .bi {
  color: var(--success);
}
.cc-label {
  min-width: 64px;
}
.cc-value {
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.cc-short {
  margin-left: auto;
  color: var(--danger);
}
.cert-block-hint {
  margin: 0;
  font-size: 12px;
  color: var(--text-light);
}
.cert-actions {
  display: flex;
  gap: 10px;
  margin-top: 4px;
}
.cert-actions-center {
  display: flex;
  justify-content: center;
  padding: 4px 0 8px;
}
.cert-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 16px;
  color: var(--text-light);
}
.cert-empty .bi {
  font-size: 28px;
}
.cert-empty p {
  margin: 0;
  font-size: 13px;
}
</style>
