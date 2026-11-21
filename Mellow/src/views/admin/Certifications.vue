<template>
  <div class="cert-admin">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">用户认证</h2>
          <p class="block-desc">
            审核用户提交的认证申请（类型 / 领域 / 申请字段由「认证类型」页配置）。待审核 {{ pending }} 条 · 合计 {{ total }} 条
            <template v-if="pending > 0">（通过后会在用户主页头像右下角展示认证标识）</template>
          </p>
        </div>
        <div class="head-actions">
          <RouterLink class="btn btn-sm btn-soft" to="/admin/users/cert-types">
            <i class="bi bi-sliders" /> 认证类型
          </RouterLink>
          <!-- 运营主动核验后直接发放，不依赖用户先提交申请 -->
          <button class="btn btn-sm btn-primary" :disabled="busy" @click="openGrant">
            <i class="bi bi-patch-plus" /> 直接授予认证
          </button>
          <button class="btn btn-sm btn-soft" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </header>

      <!-- 筛选：审核状态 + 认证类型 -->
      <div class="cert-filter">
        <div class="chip-group" aria-label="审核状态">
          <button
            v-for="opt in STATUS_OPTIONS"
            :key="opt.value"
            type="button"
            class="chip"
            :class="{ 'is-active': status === opt.value }"
            @click="setStatus(opt.value)"
          >{{ opt.label }}</button>
        </div>
        <div class="chip-group" aria-label="认证类型">
          <button
            v-for="opt in TYPE_OPTIONS"
            :key="opt.value"
            type="button"
            class="chip"
            :class="{ 'is-active': certType === opt.value }"
            @click="setType(opt.value)"
          >{{ opt.label }}</button>
        </div>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <EmptyState v-else-if="!list.length" icon="bi bi-patch-check" text="没有符合条件的认证申请" />

      <ul v-else class="cert-list">
        <li v-for="item in list" :key="item.id" class="cert-item">
          <div class="cert-head">
            <span class="cert-type">{{ item.cert_type_text }}</span>
            <strong class="cert-title">{{ item.title }}</strong>
            <span class="cert-status" :class="statusClass(item.status)">{{ item.status_text }}</span>
            <time class="cert-time">{{ formatDate(item.create_time) }} 提交</time>
          </div>

          <div class="cert-owner">
            <OwnerInfo :uid="item.uid" label="申请人" fallback="未知用户" />
            <span class="meta-text">认证领域：{{ item.category || '—' }}</span>
          </div>

          <!-- 申请字段：顺序与后台「认证类型」配置一致，图片单独渲染；仅后台可见项也在这里 -->
          <dl v-if="detailFields(item).length" class="cert-detail">
            <div v-for="field in detailFields(item)" :key="field.key">
              <dt>{{ field.label }}</dt>
              <dd>{{ field.text || '—' }}</dd>
            </div>
            <div v-if="Number(item.expire_time) > 0"><dt>有效期至</dt><dd>{{ formatDate(item.expire_time) }}</dd></div>
          </dl>

          <!-- 认证说明：后台审核时填写，展示在用户主页 -->
          <p class="cert-desc">
            <span class="cert-desc-label">认证说明</span>
            {{ item.description || '（未填写，通过后主页不展示说明）' }}
          </p>

          <!-- 证明材料：点击放大核对 -->
          <div v-if="(item.images || []).length" class="cert-images">
            <img
              v-for="(img, i) in item.images"
              :key="img"
              :src="img"
              class="cert-image"
              alt="证明材料"
              loading="lazy"
              @click="openLightbox(item.images, i)"
            />
          </div>

          <p v-if="item.reason" class="cert-reason">驳回原因：{{ item.reason }}</p>

          <div class="cert-actions">
            <!-- 改资料：不改变审核状态，用户填错 / 资料过时由运营直接修 -->
            <button class="btn btn-sm btn-soft" :disabled="busy" @click="openEdit(item)">
              <i class="bi bi-pencil-square" /> 编辑资料
            </button>
            <template v-if="Number(item.status) !== 1">
              <button class="btn btn-sm btn-primary" :disabled="busy" @click="openPass(item)">
                <i class="bi bi-check2-circle" /> 通过
              </button>
              <button class="btn btn-sm btn-ghost" :disabled="busy" @click="openReject(item)">
                <i class="bi bi-slash-circle" /> 驳回
              </button>
            </template>
            <template v-else>
              <span class="meta-text">
                已通过 · {{ Number(item.expire_time) > 0 ? `${formatDate(item.expire_time)} 到期` : '长期有效' }}
              </span>
              <!-- 收回：认证记录删除，主页标识立即消失（与驳回是两回事） -->
              <button class="btn btn-sm btn-ghost" :disabled="busy" @click="openRevoke(item)">
                <i class="bi bi-x-circle" /> 收回
              </button>
            </template>
          </div>
        </li>
      </ul>

      <Pagination
        v-if="!loading && total > 0"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :sizes="PAGE_SIZE_OPTIONS"
        @update:page-size="(size) => { pageSize = size; page = 1; load() }"
        @update:current="(p) => { page = p; load() }"
      />
    </section>

    <!-- 通过：可选择认证有效期（0 = 永久） -->
    <ConfirmDialog
      :visible="passDialog.visible"
      title="通过认证"
      confirm-text="确认通过"
      :loading="passDialog.loading"
      @update:visible="(v) => (passDialog.visible = v)"
      @confirm="runPass"
    >
      <p class="dialog-line">
        认证对象：<strong>{{ passDialog.item?.applicant?.nickname || `UID ${passDialog.item?.uid || ''}` }}</strong>
        · {{ passDialog.item?.cert_type_text }} · {{ passDialog.item?.title }}
      </p>
      <label class="dialog-field">
        <span class="dialog-field-label">认证有效期</span>
        <FormSelect v-model="passDialog.days" :options="EXPIRE_OPTIONS" />
      </label>
      <label class="dialog-field">
        <span class="dialog-field-label">认证说明<em>由后台填写，展示在用户主页；用户端没有填写入口</em></span>
        <FormTextarea
          v-model="passDialog.description"
          rows="3"
          maxlength="255"
          placeholder="一句话介绍该认证，如「某某科技前端工程师，专注工程化与性能优化」" />
      </label>
      <p class="dialog-hint">
        通过后认证标识立即展示在用户主页昵称旁，用户会收到站内信通知。
      </p>
    </ConfirmDialog>

    <!-- 编辑资料：只改用户提交的字段，不改变审核状态 -->
    <AdminFormDialog
      :visible="editDialog.visible"
      title="修改认证资料"
      icon="bi bi-pencil-square"
      width="680px"
      confirm-text="保存资料"
      :loading="editDialog.loading || editDialog.uploading"
      @update:visible="(v) => (editDialog.visible = v)"
      @confirm="runEdit"
    >
      <p class="dialog-line">
        申请人：<strong>{{ editDialog.item?.applicant?.nickname || `UID ${editDialog.item?.uid || ''}` }}</strong>
        · {{ editDialog.item?.cert_type_text }} · 当前状态 {{ editDialog.item?.status_text }}
      </p>

      <CertFieldsForm
        v-model="editDialog.values"
        :fields="editFields"
        :categories="editCategories"
        @update:uploading="(value) => (editDialog.uploading = value)"
      />

      <label class="dialog-field edit-desc">
        <span class="dialog-field-label">认证说明<em>展示在用户主页；用户端没有填写入口</em></span>
        <FormTextarea
          v-model="editDialog.description"
          rows="2"
          maxlength="255"
          placeholder="可空；通过时也可以再填" />
      </label>

      <p class="dialog-hint">
        这里只改资料（含证明材料），不会改变审核状态；通过 / 驳回与有效期仍走对应按钮。
        保存后若有实际变化，会给用户发一条站内提醒（只列改动了哪几项）。
      </p>
    </AdminFormDialog>

    <!-- 直接授予认证：运营线下核验后发放，用户端没有提交过申请 -->
    <AdminFormDialog
      :visible="grantDialog.visible"
      title="直接授予认证"
      icon="bi bi-patch-plus"
      width="680px"
      confirm-text="确认授予"
      :loading="grantDialog.loading || grantDialog.uploading"
      @update:visible="(v) => (grantDialog.visible = v)"
      @confirm="runGrant"
    >
      <label class="dialog-field">
        <span class="dialog-field-label">用户 UID<em>可从「用户管理」复制；已认证的用户会被本次内容整条覆盖</em></span>
        <FormInput
          v-model="grantDialog.uid"
          type="number"
          min="1"
          step="1"
          placeholder="如 12" />
      </label>
      <OwnerInfo :uid="Number(grantDialog.uid) || 0" label="认证对象" fallback="请输入用户 UID" />

      <label class="dialog-field">
        <span class="dialog-field-label">认证类型</span>
        <FormSelect
          v-model="grantDialog.certType"
          :options="grantTypeOptions"
          placeholder="请选择认证类型"
          @change="onGrantTypeChange" />
      </label>

      <CertFieldsForm
        v-if="grantFields.length"
        v-model="grantDialog.values"
        :fields="grantFields"
        :categories="grantCategories"
        @update:uploading="(value) => (grantDialog.uploading = value)"
      />
      <p v-else-if="grantDialog.certType" class="dialog-hint">
        该认证类型没有配置申请字段，授予后只展示认证名称与说明。
      </p>

      <label class="dialog-field">
        <span class="dialog-field-label">认证有效期</span>
        <FormSelect v-model="grantDialog.days" :options="EXPIRE_OPTIONS" />
      </label>

      <label class="dialog-field">
        <span class="dialog-field-label">认证说明<em>展示在用户主页；可留空</em></span>
        <FormTextarea
          v-model="grantDialog.description"
          rows="2"
          maxlength="255"
          placeholder="一句话介绍该认证，如「某某科技前端工程师，专注工程化与性能优化」" />
      </label>

      <p class="dialog-hint">
        授予后认证标识立即展示在用户主页头像右下角，用户会收到一条站内信。
        必填项不强制：没填认证名称时按类型名展示。
      </p>
    </AdminFormDialog>

    <!-- 收回：已发出的认证被撤销（冒名 / 资质过期 / 机构注销），记录删除 -->
    <ConfirmDialog
      :visible="revokeDialog.visible"
      title="收回认证"
      confirm-text="确认收回"
      :loading="revokeDialog.loading"
      @update:visible="(v) => (revokeDialog.visible = v)"
      @confirm="runRevoke"
    >
      <p class="dialog-line">
        认证对象：<strong>{{ revokeDialog.item?.applicant?.nickname || `UID ${revokeDialog.item?.uid || ''}` }}</strong>
        · {{ revokeDialog.item?.cert_type_text }} · {{ revokeDialog.item?.title }}
      </p>
      <label class="dialog-field">
        <span class="dialog-field-label">收回原因<em>可空；会随站内信一起发给用户</em></span>
        <FormTextarea
          v-model="revokeDialog.reason"
          rows="2"
          maxlength="200"
          placeholder="如「资质已过期」「冒用他人身份」" />
      </label>
      <p class="dialog-hint">
        收回后认证记录会被删除，主页的认证标识立即消失，用户可重新提交申请。
        如果只是资料填错，用「编辑资料」而不是收回。
      </p>
    </ConfirmDialog>

    <!-- 驳回：必须填写原因（复用后台统一的审核弹窗） -->
    <AuditDialog
      :visible="rejectDialog.visible"
      :audit="2"
      :loading="rejectDialog.loading"
      :owner-uid="rejectDialog.item?.uid || 0"
      owner-label="申请人"
      :title="`驳回认证：${rejectDialog.item?.title || ''}`"
      message="驳回后用户会收到通知，其中包含你填写的原因；用户可修改资料后重新提交。"
      confirm-text="确认驳回"
      @update:visible="(v) => (rejectDialog.visible = v)"
      @confirm="runReject"
    />
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 后台「用户认证」审核
 *
 * 与举报 / 内容审核的区别：这里审的是**账号属性**，通过后不影响内容，只决定用户主页
 * 是否展示认证标识（前端 components/VerifiedBadge.vue）。审核结论会发站内信给申请人。
 *
 * 交互口径与其它审核页保持一致：
 *   - 通过：可选有效期（长期 / 1 年 / 2 年），调用 PUT user-certification/handle
 *   - 驳回：走 AuditDialog，原因必填
 *   - 直接授予：运营线下核验过的账号（知名创作者 / 机构）不必等用户提交申请，
 *     填 UID + 类型 + 资料即可发放，调用 POST user-certification/grant
 *   - 收回：已发出的认证被撤销（冒名 / 资质过期），记录删除，调用 DELETE user-certification/revoke
 */
import { computed, onMounted, reactive, ref } from 'vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import CertFieldsForm from '@/components/CertFieldsForm.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AuditDialog from '@/components/admin/AuditDialog.vue'
import OwnerInfo from '@/components/admin/OwnerInfo.vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import {
  editCertification,
  grantCertification,
  handleCertification,
  listCertifications,
  revokeCertification
} from '@/api/user-certification'
import { listCertTypes } from '@/api/cert-type'
import { formatDate } from '@/utils/time'
import { openLightbox } from '@/utils/lightbox'
import { toast } from '@/utils/toast'

const STATUS_OPTIONS = [
  { value: '', label: '全部' },
  { value: '0', label: '待审核' },
  { value: '1', label: '已通过' },
  { value: '2', label: '已驳回' }
]

// 类型筛选项来自后端（cert-type 表），后台增删类型后这里自动跟着变
const typeOptions = ref([])
const TYPE_OPTIONS = computed(() => [
  { value: '', label: '全部类型' },
  ...typeOptions.value.map((item) => ({ value: String(item.value), label: item.label }))
])

const EXPIRE_OPTIONS = [
  { value: 0, label: '长期有效' },
  { value: 365, label: '1 年' },
  { value: 730, label: '2 年' }
]

const list = ref([])
const total = ref(0)
const pending = ref(0)
const loading = ref(false)
const busy = ref(false)
const page = ref(1)
const pageSize = ref(PAGE_SIZE_DEFAULT)
const status = ref('0')
const certType = ref('')

const passDialog = reactive({ visible: false, loading: false, item: null, days: 0, description: '' })
const rejectDialog = reactive({ visible: false, loading: false, item: null })
const revokeDialog = reactive({ visible: false, loading: false, item: null, reason: '' })

// 完整的类型配置（含字段定义），只在编辑资料 / 直接授予时用；筛选下拉仍用 all 返回的精简 types
const certTypes = ref([])
const editDialog = reactive({ visible: false, loading: false, uploading: false, item: null, values: {}, description: '' })

// 直接授予：不经过用户提交，uid + 类型 + 资料 + 有效期直接落库
const grantDialog = reactive({
  visible: false,
  loading: false,
  uploading: false,
  uid: '',
  certType: '',
  values: {},
  days: 0,
  description: ''
})

const grantTypeOptions = computed(() =>
  certTypes.value.map((item) => ({ value: Number(item.id), label: item.name }))
)
const grantConfig = computed(
  () => certTypes.value.find((item) => Number(item.id) === Number(grantDialog.certType)) || null
)
const grantFields = computed(() => grantConfig.value?.fields || [])
const grantCategories = computed(() => grantConfig.value?.categories || [])

// 当前编辑记录的类型配置
const editConfig = computed(() => {
  const id = Number(editDialog.item?.cert_type) || 0
  return certTypes.value.find((item) => Number(item.id) === id) || null
})
const editFields = computed(() => editConfig.value?.fields || [])
const editCategories = computed(() => editConfig.value?.categories || [])

/**
 * 审核时要看的字段：全部字段（含「仅后台可见」），但图片单独用缩略图渲染，
 * 空值也保留一行，方便确认用户确实没填（比直接隐藏更有信息量）。
 */
function detailFields(item) {
  return (item?.fields || []).filter((field) => field.type !== 'image')
}

function statusClass(value) {
  const current = Number(value)
  if (current === 0) return 'is-pending'
  if (current === 1) return 'is-passed'
  return 'is-rejected'
}

async function load() {
  loading.value = true
  try {
    const res = await listCertifications({
      page: page.value,
      limit: pageSize.value,
      status: status.value,
      cert_type: certType.value
    })
    list.value = res.data?.data || []
    total.value = Number(res.data?.count) || 0
    pending.value = Number(res.data?.pending) || 0
    typeOptions.value = res.data?.types || []
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function setStatus(value) {
  if (status.value === value) return
  status.value = value
  page.value = 1
  load()
}

function setType(value) {
  if (certType.value === value) return
  certType.value = value
  page.value = 1
  load()
}

function openPass(item) {
  passDialog.item = item
  passDialog.days = 0
  // 重新审核时把上一次填过的说明带出来，运营只需改变动的部分
  passDialog.description = item.description || ''
  passDialog.visible = true
}

function openReject(item) {
  rejectDialog.item = item
  rejectDialog.visible = true
}

function openRevoke(item) {
  revokeDialog.item = item
  revokeDialog.reason = ''
  revokeDialog.visible = true
}

// ---------- 编辑资料 ----------

function toList(value) {
  if (Array.isArray(value)) return value.filter(Boolean)
  const text = String(value || '').trim()
  if (!text) return []
  return text.split(',').map((item) => item.trim()).filter(Boolean)
}

/**
 * 按类型配置生成编辑表单的初值。
 * 优先用记录里的 data（认证字段原始值），老记录没有 data 时用 fields 行里的 value
 * （后端已做过老列回落），都没有才按字段类型给空值。
 */
function buildEditValues(item, config) {
  const values = {}
  const rows = item?.fields || []

  for (const field of config?.fields || []) {
    const row = rows.find((entry) => entry.key === field.key)
    let value = row?.value
    if (value === undefined || value === null || value === '') {
      value = item?.[field.key]
    }

    if (field.type === 'image' || (field.type === 'select' && field.multiple)) {
      values[field.key] = toList(value)
    } else if (field.type === 'number') {
      values[field.key] = value === undefined || value === null || value === '' ? '' : String(value)
    } else if (field.type === 'date') {
      values[field.key] = String(value || '').slice(0, 10)
    } else {
      values[field.key] = String(value || '')
    }
  }

  return values
}

function openEdit(item) {
  const config = certTypes.value.find((type) => Number(type.id) === Number(item.cert_type))
  if (!config) {
    toast.warning('该认证类型的配置已删除，无法编辑资料')
    return
  }

  editDialog.item = item
  editDialog.values = buildEditValues(item, config)
  editDialog.description = item.description || ''
  editDialog.visible = true
}

async function runEdit() {
  const item = editDialog.item
  if (!item) {
    editDialog.visible = false
    return
  }

  editDialog.loading = true
  busy.value = true
  try {
    // values 全量提交：管理员清空的项就是「清空」，后端按类型配置整体重写
    await editCertification({
      id: item.id,
      values: editDialog.values,
      description: editDialog.description.trim()
    })
    toast.success('资料已保存')
    editDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    editDialog.loading = false
    busy.value = false
  }
}

// ---------- 直接授予认证 ----------

function openGrant() {
  grantDialog.uid = ''
  grantDialog.days = 0
  grantDialog.description = ''

  // 默认选第一个类型，字段初值按该类型配置生成（全空）
  const first = certTypes.value[0] || null
  grantDialog.certType = first ? Number(first.id) : ''
  grantDialog.values = buildEditValues(null, first || {})
  grantDialog.visible = true
}

function onGrantTypeChange(value) {
  const config = certTypes.value.find((item) => Number(item.id) === Number(value)) || null
  grantDialog.certType = config ? Number(config.id) : ''
  // 换了类型，旧字段的 key 不再适用，整份重建
  grantDialog.values = buildEditValues(null, config || {})
}

async function runGrant() {
  const uid = Number(grantDialog.uid) || 0
  if (uid <= 0) {
    toast.warning('请填写用户 UID')
    return
  }
  if (!Number(grantDialog.certType)) {
    toast.warning('请选择认证类型')
    return
  }

  grantDialog.loading = true
  busy.value = true
  try {
    await grantCertification({
      uid,
      cert_type: Number(grantDialog.certType),
      values: grantDialog.values,
      description: grantDialog.description.trim(),
      expire_days: Number(grantDialog.days) || 0
    })
    toast.success('已授予认证')
    grantDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示（用户不存在 / 字段格式不合法等） */
  } finally {
    grantDialog.loading = false
    busy.value = false
  }
}

async function loadTypes() {
  try {
    const res = await listCertTypes()
    certTypes.value = res.data?.data || []
  } catch {
    certTypes.value = []
  }
}

async function runPass() {
  const item = passDialog.item
  if (!item) {
    passDialog.visible = false
    return
  }

  passDialog.loading = true
  busy.value = true
  try {
    await handleCertification({
      id: item.id,
      status: 1,
      expire_days: Number(passDialog.days) || 0,
      description: passDialog.description.trim()
    })
    toast.success('已通过认证')
    passDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    passDialog.loading = false
    busy.value = false
  }
}

async function runReject(reason) {
  const item = rejectDialog.item
  if (!item) {
    rejectDialog.visible = false
    return
  }

  rejectDialog.loading = true
  busy.value = true
  try {
    await handleCertification({ id: item.id, status: 2, reason })
    toast.success('已驳回')
    rejectDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    rejectDialog.loading = false
    busy.value = false
  }
}

async function runRevoke() {
  const item = revokeDialog.item
  if (!item) {
    revokeDialog.visible = false
    return
  }

  revokeDialog.loading = true
  busy.value = true
  try {
    await revokeCertification({ id: item.id, reason: revokeDialog.reason.trim() })
    toast.success('已收回认证')
    revokeDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    revokeDialog.loading = false
    busy.value = false
  }
}

onMounted(() => {
  load()
  // 类型配置单独取一份完整版（含字段定义），编辑资料时渲染表单用
  loadTypes()
})
</script>

<style scoped>
/*
 * 与 Reports.vue / UserList.vue 保持同一套口径：.panel-head / .chip / .block-title
 * 都不是全局样式，每个后台页自己声明一份。
 */
.cert-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.block-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.chip-group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: var(--control-h-sm);
  padding: 0 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-family: inherit;
  font-size: 12px;
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
}
.cert-filter {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.cert-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.cert-item {
  padding: 14px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.cert-item:last-child {
  border-bottom: none;
}
.cert-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.cert-type {
  padding: 1px 8px;
  font-size: 12px;
  border-radius: 999px;
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.cert-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.cert-status {
  padding: 1px 8px;
  font-size: 12px;
  border-radius: 999px;
}
.cert-status.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.cert-status.is-passed {
  color: var(--success);
  background: rgba(58, 160, 106, 0.12);
}
.cert-status.is-rejected {
  color: var(--danger);
  background: rgba(217, 84, 77, 0.12);
}
.cert-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-light);
}
.cert-owner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-top: 8px;
}
.meta-text {
  font-size: 12px;
  color: var(--text-muted);
}
.cert-detail {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 6px 18px;
  margin: 10px 0 0;
}
.cert-detail div {
  display: flex;
  gap: 6px;
  font-size: 12.5px;
}
.cert-detail dt {
  flex-shrink: 0;
  color: var(--text-light);
}
.cert-detail dd {
  margin: 0;
  color: var(--text-soft);
  overflow-wrap: break-word;
}
.cert-desc {
  margin: 10px 0 0;
  padding: 8px 10px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-alt);
  border-radius: var(--radius-sm);
}
.cert-desc-label {
  display: block;
  margin-bottom: 2px;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-light);
}
.cert-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.cert-image {
  width: 84px;
  height: 84px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
  cursor: zoom-in;
}
.cert-reason {
  margin: 10px 0 0;
  padding: 8px 10px;
  font-size: 12.5px;
  color: var(--danger);
  background: rgba(217, 84, 77, 0.08);
  border-radius: var(--radius-sm);
}
.cert-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}
.dialog-line {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--text-soft);
}
.dialog-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 10px;
}
.edit-desc {
  margin-top: 14px;
}
.dialog-field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}
.dialog-field-label em {
  font-style: normal;
  font-weight: 400;
  color: var(--text-light);
  margin-left: 6px;
}
.dialog-hint {
  margin: 0;
  font-size: 12px;
  color: var(--text-light);
}
</style>
