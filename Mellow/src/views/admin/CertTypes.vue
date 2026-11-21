<template>
  <div class="cert-types">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">认证类型</h2>
          <p class="block-desc">
            自定义认证类型、该类型下的认证领域与申请字段。「认证说明」由后台填写，申请页只展示给用户看。
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm btn-soft" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
          <button class="btn btn-sm btn-primary" @click="openCreate">
            <i class="bi bi-plus-lg" /> 新建类型
          </button>
        </div>
      </header>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <EmptyState v-else-if="!list.length" icon="bi bi-sliders" text="还没有认证类型，点右上角新建一个" />

      <ul v-else class="ct-list">
        <li v-for="item in list" :key="item.id" class="ct-item" :class="{ 'is-off': !Number(item.enable) }">
          <div class="ct-head">
            <i class="ct-icon" :class="item.icon || 'bi bi-patch-check'" :style="{ color: certColor(item) }" />
            <strong class="ct-name">{{ item.name }}</strong>
            <span v-if="item.intro" class="ct-intro">{{ item.intro }}</span>
            <span class="ct-tag" :class="Number(item.enable) ? 'is-on' : 'is-off'">
              {{ Number(item.enable) ? '启用中' : '已停用' }}
            </span>
            <span class="ct-meta">{{ (item.categories || []).length }} 个领域 · {{ (item.fields || []).length }} 个申请字段</span>
            <span class="ct-cond"><i class="bi bi-shield-check" /> {{ certConditionText(item.conditions) }}</span>
            <span class="ct-meta">已用 {{ Number(item.used) || 0 }} 条</span>
          </div>

          <p v-if="item.notice" class="ct-notice">{{ item.notice }}</p>

          <div class="ct-fields">
            <span v-for="field in item.fields" :key="field.key" class="ct-field" :class="{ 'is-req': field.required }">
              {{ field.label }}
              <em>{{ fieldTypeText(field.type) }}</em>
            </span>
          </div>

          <div class="ct-actions">
            <button class="btn btn-sm" @click="openEdit(item)">
              <i class="bi bi-pencil" /> 编辑
            </button>
            <button class="btn btn-sm btn-ghost" @click="toggleEnable(item)">
              <i class="bi" :class="Number(item.enable) ? 'bi-pause-circle' : 'bi-play-circle'" />
              {{ Number(item.enable) ? '停用' : '启用' }}
            </button>
            <button class="btn btn-sm btn-ghost" @click="openRemove(item)">
              <i class="bi bi-trash3" /> 删除
            </button>
          </div>
        </li>
      </ul>
    </section>

    <!-- 新建 / 编辑 -->
    <AdminFormDialog
      :visible="dialog.visible"
      :title="dialog.id ? '编辑认证类型' : '新建认证类型'"
      icon="bi bi-sliders"
      width="820px"
      :loading="dialog.loading"
      confirm-text="保存"
      @update:visible="(v) => (dialog.visible = v)"
      @confirm="save"
    >
      <div class="ct-form">
        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">类型名称</label>
            <FormInput v-model="form.name" type="text" maxlength="32" placeholder="如：职业认证 / 创作者认证" />
          </div>
          <div class="form-item">
            <label class="form-label">图标</label>
            <FormInput v-model="form.icon" type="text" maxlength="64" placeholder="Bootstrap Icons 类名，如 bi-briefcase" />
            <p class="form-hint">
              预览：<i :class="form.icon || 'bi bi-patch-check'" />
            </p>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">
              标识颜色
              <em>昵称旁的盾牌与简介下的认证行共用这一色，留空走内置兜底色</em>
            </label>
            <div class="ct-color-row">
              <FormInput
                v-model="form.color"
                class="ct-color-text"
                type="text"
                maxlength="9"
                placeholder="#e8a020" />
              <input
                class="ct-color-picker"
                type="color"
                :value="form.color || certColor({ ...form, cert_type: dialog.id })"
                aria-label="选择认证标识颜色"
                @input="form.color = $event.target.value"
              />
              <button v-if="form.color" type="button" class="btn btn-sm btn-ghost" @click="form.color = ''">
                恢复默认
              </button>
            </div>
            <p class="form-hint">
              预览：<i class="bi bi-patch-check-fill" :style="{ color: certColor({ ...form, cert_type: dialog.id }) }" />
              {{ form.name || '认证类型' }}
            </p>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">一句话简介</label>
            <FormInput v-model="form.intro" type="text" maxlength="60" placeholder="选类型时展示的副文案，可空" />
          </div>
          <div class="form-item">
            <label class="form-label">排序</label>
            <FormInput v-model.number="form.sort" type="number" min="0" max="9999" />
            <p class="form-hint">数字小的排前面</p>
          </div>
        </div>

        <div class="form-item">
          <label class="form-label">
            认证说明
            <em>后台填写，申请页原样展示给用户；用户不能自己写</em>
          </label>
          <FormTextarea
            v-model="form.notice"
            rows="4"
            maxlength="2000"
            placeholder="说明该认证适合谁、需要准备什么材料、审核与展开口径" />
        </div>

        <div class="form-item">
          <label class="form-label">
            申请条件
            <em>0 表示不限；未达标的用户在申请页不能提交，并会看到还差多少</em>
          </label>
          <div class="ct-conds">
            <label v-for="cond in CERT_CONDITIONS" :key="cond.key" class="ct-cond-field">
              <span>{{ cond.label }}</span>
              <FormInput
                v-model.number="form.conditions[cond.key]"
                class="cf-num"
                type="number"
                min="0"
                max="100000000"
                :placeholder="cond.hint" />
              <em>{{ cond.unit }}</em>
            </label>
          </div>
          <p class="form-hint">
            只统计已发布内容（audit=1 且 status=1），草稿与待审不算；注册天数按账号创建时间计算。
          </p>
        </div>

        <div class="form-item">
          <label class="form-label">认证领域<em>申请页「认证领域」下拉的候选值，顺序即展示顺序</em></label>
          <div v-if="form.categories.length" class="ct-cats">
            <span v-for="(category, index) in form.categories" :key="`${category}-${index}`" class="ct-cat">
              {{ category }}
              <button type="button" :title="`移除「${category}」`" @click="form.categories.splice(index, 1)">
                <i class="bi bi-x" />
              </button>
            </span>
          </div>
          <div class="ct-cat-add">
            <FormInput
              v-model="categoryDraft"
              type="text"
              maxlength="32"
              placeholder="输入领域后回车添加，如「摄影」"
              @keydown.enter.prevent="addCategory" />
            <button class="btn btn-sm" type="button" @click="addCategory">添加</button>
          </div>
        </div>

        <div class="form-item">
          <label class="form-label">
            申请字段
            <em>用户要填什么、怎么校验，全在这里定义</em>
          </label>
          <CertFieldsEditor ref="fieldsRef" v-model="form.fields" />
        </div>

        <label class="ct-enable">
          <FormCheckbox v-model="form.enable" :true-value="1" :false-value="0" />
          启用该类型（停用后前台不再可选，已有认证记录照常展示）
        </label>
      </div>
    </AdminFormDialog>

    <!-- 删除 -->
    <ConfirmDialog
      :visible="removeDialog.visible"
      title="删除认证类型"
      danger
      confirm-text="确认删除"
      :loading="removeDialog.loading"
      @update:visible="(v) => (removeDialog.visible = v)"
      @confirm="runRemove"
    >
      <p class="dialog-line">将删除「{{ removeDialog.item?.name }}」，前台不再出现该类型。</p>
      <p v-if="Number(removeDialog.item?.used) > 0" class="dialog-warn">
        该类型下还有 {{ removeDialog.item.used }} 条认证记录，删除后这些记录的类型会显示为「未知类型」（记录本身不删）。
      </p>
    </ConfirmDialog>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 后台「认证类型」配置
 *
 * 这里是认证模块的唯一配置入口（对应后端 cert-type 控制器）：
 *   - 类型本身：名称 / 图标 / 简介 / 排序 / 启停；
 *   - 认证说明：后台填写的对外文案，申请页只读展示（用户不能填，避免主页变广告位）；
 *   - 认证领域：该类型下「认证领域」下拉的候选值；
 *   - 申请字段：用户要填哪些项、类型 / 必填 / 选项 / 公开展示范围（CertFieldsEditor）。
 *
 * 保存后申请页（views/user/Verification.vue）立刻按新配置渲染，前端没有枚举副本需要同步。
 */
import { onMounted, reactive, ref } from 'vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import CertFieldsEditor from '@/components/admin/CertFieldsEditor.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import EmptyState from '@/components/EmptyState.vue'
import {
  CERT_CONDITIONS,
  CERT_FIELD_TYPES,
  blankCertConditions,
  blankCertFields,
  certConditionText,
  listCertTypes,
  removeCertType,
  updateCertType,
  createCertType
} from '@/api/cert-type'
import { toast } from '@/utils/toast'
import { certColor } from '@/utils/cert'

const list = ref([])
const loading = ref(false)
const categoryDraft = ref('')
const fieldsRef = ref(null)

const dialog = reactive({ visible: false, loading: false, id: 0 })
const removeDialog = reactive({ visible: false, loading: false, item: null })

const form = reactive({
  name: '',
  icon: '',
  // 标识配色：留空 = 用 utils/cert 的内置兜底色
  color: '',
  intro: '',
  notice: '',
  sort: 0,
  enable: 1,
  categories: [],
  fields: [],
  // 申请门槛：粉丝 / 文章 / 动态 / 注册天数，0 = 不限
  conditions: blankCertConditions()
})

function fieldTypeText(type) {
  return CERT_FIELD_TYPES.find((item) => item.value === type)?.label || type
}

async function load() {
  loading.value = true
  try {
    const res = await listCertTypes()
    list.value = res.data?.data || []
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

function resetForm(item) {
  form.name = item?.name || ''
  form.icon = item?.icon || ''
  form.color = item?.color || ''
  form.intro = item?.intro || ''
  form.notice = item?.notice || ''
  form.sort = Number(item?.sort) || 0
  form.enable = item ? (Number(item.enable) ? 1 : 0) : 1
  form.categories = [...(item?.categories || [])]
  // 条件补全默认值：老配置没有 conditions 时按「不限」渲染，避免输入框绑到 undefined
  form.conditions = { ...blankCertConditions(), ...(item?.conditions || {}) }
  // 深拷贝 + 兜底：options 可能是 null（后端 normalize 后为 null），编辑器里按数组操作
  form.fields = JSON.parse(JSON.stringify(item?.fields || blankCertFields())).map((field) => ({
    ...field,
    options: Array.isArray(field.options) ? field.options : [],
    max: Number(field.max) || 0,
    max_count: Number(field.max_count) || 0
  }))
  categoryDraft.value = ''
}

function openCreate() {
  dialog.id = 0
  resetForm(null)
  dialog.visible = true
}

function openEdit(item) {
  dialog.id = Number(item.id)
  resetForm(item)
  dialog.visible = true
}

function addCategory() {
  const value = categoryDraft.value.trim()
  if (!value) return
  if (form.categories.includes(value)) {
    categoryDraft.value = ''
    return
  }
  form.categories.push(value)
  categoryDraft.value = ''
}

async function save() {
  if (!form.name.trim()) {
    toast.warning('请填写类型名称')
    return
  }
  const fieldError = fieldsRef.value?.validate?.() || ''
  if (fieldError) {
    toast.warning(fieldError)
    return
  }

  const payload = {
    name: form.name.trim(),
    icon: form.icon.trim(),
    color: form.color.trim(),
    intro: form.intro.trim(),
    notice: form.notice.trim(),
    sort: Number(form.sort) || 0,
    enable: Number(form.enable) ? 1 : 0,
    categories: form.categories,
    fields: form.fields,
    conditions: form.conditions
  }

  dialog.loading = true
  try {
    if (dialog.id) {
      await updateCertType({ id: dialog.id, ...payload })
      toast.success('已保存')
    } else {
      await createCertType(payload)
      toast.success('已创建')
    }
    dialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    dialog.loading = false
  }
}

async function toggleEnable(item) {
  try {
    await updateCertType({ id: item.id, enable: Number(item.enable) ? 0 : 1 })
    toast.success(Number(item.enable) ? '已停用' : '已启用')
    await load()
  } catch {
    /* 拦截器已提示 */
  }
}

function openRemove(item) {
  removeDialog.item = item
  removeDialog.visible = true
}

async function runRemove() {
  const item = removeDialog.item
  if (!item) {
    removeDialog.visible = false
    return
  }

  removeDialog.loading = true
  try {
    // 弹窗里已经把「历史记录会变成未知类型」说清楚了，这里直接带 force
    await removeCertType(item.id, true)
    toast.success('已删除')
    removeDialog.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    removeDialog.loading = false
  }
}

onMounted(load)
</script>

<style scoped>
.cert-types {
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
.ct-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.ct-item {
  padding: 14px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.ct-item:last-child {
  border-bottom: none;
}
.ct-item.is-off .ct-name,
.ct-item.is-off .ct-icon {
  opacity: 0.55;
}
.ct-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.ct-icon {
  font-size: 15px;
  color: var(--primary);
}
.ct-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.ct-intro {
  font-size: 12px;
  color: var(--text-muted);
}
.ct-tag {
  padding: 1px 8px;
  font-size: 12px;
  border-radius: 999px;
}
.ct-tag.is-on {
  color: var(--success);
  background: rgba(58, 160, 106, 0.12);
}
.ct-tag.is-off {
  color: var(--text-light);
  background: var(--bg-muted);
}
.ct-meta {
  font-size: 12px;
  color: var(--text-light);
}
.ct-meta:first-of-type {
  margin-left: auto;
}
.ct-cond {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.ct-cond .bi {
  color: var(--primary);
}
.ct-color-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ct-color-text {
  width: 130px;
}
.ct-color-picker {
  width: 38px;
  height: 32px;
  padding: 2px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.ct-conds {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
  gap: 8px 12px;
}
.ct-cond-field {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.ct-cond-field span {
  flex-shrink: 0;
}
.ct-cond-field em {
  font-style: normal;
  color: var(--text-light);
}
.ct-cond-field .input {
  width: 96px;
}
.ct-notice {
  margin: 8px 0 0;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--text-soft);
}
.ct-fields {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
.ct-field {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 8px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
}
.ct-field em {
  font-style: normal;
  font-size: 11px;
  color: var(--text-light);
}
.ct-field.is-req {
  color: var(--primary-deep);
  border-color: var(--primary-soft);
}
.ct-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
}
.ct-form {
  display: flex;
  flex-direction: column;
}
.ct-cats {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}
.ct-cat {
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
.ct-cat button {
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
.ct-cat button:hover {
  color: var(--danger);
}
.ct-cat-add {
  display: flex;
  gap: 8px;
}
.ct-enable {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  font-size: 12.5px;
  color: var(--text-muted);
  cursor: pointer;
}
.form-label em {
  font-style: normal;
  font-weight: 400;
  color: var(--text-light);
  margin-left: 6px;
}
.dialog-line {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--text-soft);
}
.dialog-warn {
  margin: 0;
  padding: 8px 10px;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: var(--radius-sm);
}
</style>
