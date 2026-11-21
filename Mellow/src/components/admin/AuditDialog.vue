<template>
  <ConfirmDialog
    :visible="visible"
    :title="resolvedTitle"
    :message="resolvedMessage"
    :confirm-text="resolvedConfirmText"
    :danger="needReason"
    :loading="loading"
    @update:visible="emit('update:visible', $event)"
    @confirm="onConfirm"
    @cancel="onCancel"
  >
    <!-- 内容归属：审核前先确认这条是谁的（文章/独立页面/动态 = 作者，友链 = 申请人） -->
    <OwnerInfo v-if="Number(ownerUid) > 0" :uid="ownerUid" :label="ownerLabel" class="audit-owner" />

    <label class="audit-reason">
      <span class="audit-reason-label">
        驳回原因
        <em v-if="needReason" class="audit-reason-tag is-required">必填</em>
        <em v-else class="audit-reason-tag">选填</em>
      </span>
      <FormTextarea
        v-model="reason"
        class="audit-reason-input"
        rows="3"
        maxlength="512"
        :placeholder="needReason ? '说明不通过的原因，作者会在通知与「我的内容」里看到' : '可选：补充说明，作者可见'" />
      <span class="audit-reason-hint">{{ reason.length }}/512 · 原因会随通知一起发给作者</span>
    </label>
  </ConfirmDialog>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
/**
 * 审核驳回弹窗（动态 / 文章 / 友链后台列表共用）
 *
 * 为什么单独封装：三处后台的审核交互各不相同（动态是单选框、文章是按钮、友链是按钮 +
 * 编辑弹窗），但「驳回必须写原因」的表单与文案完全一致，抽出来避免三份重复实现。
 *
 * 用法：
 *   <AuditDialog
 *     :visible="auditDialog.visible"
 *     :audit="auditDialog.audit"
 *     :loading="auditDialog.loading"
 *     @update:visible="closeAuditDialog"
 *     @confirm="runAudit"
 *   />
 *   - confirm 事件会带上原因文本：@confirm="(reason) => ..."
 *   - audit = 2（未通过）时原因必填，audit = 0（打回待审核）时选填
 *   - 具体是谁（单条还是批量）由业务侧自己记在 auditDialog.target 里
 */
import { ref, computed, watch } from 'vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import OwnerInfo from '@/components/admin/OwnerInfo.vue'
import { toast } from '@/utils/toast'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 目标审核状态：2 = 未通过（原因必填）；0 = 打回待审核（原因选填）
  audit: { type: Number, default: 2 },
  // 目标条目已有的驳回原因：打开时回填，刷新页面后再驳回也能看到上次写的原因
  defaultReason: { type: String, default: '' },
  // 内容归属人：审核时能看到这条是谁的（单条审核传 uid；批量审核不传，界面不显示这一行）
  ownerUid: { type: [Number, String], default: 0 },
  // 归属人文案：文章/页面/动态传「作者」，友链传「申请人」
  ownerLabel: { type: String, default: '作者' },
  loading: { type: Boolean, default: false },
  // 文案默认值可按业务覆盖（如「驳回动态」「驳回文章」）
  title: { type: String, default: '' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: '' }
})

const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

const reason = ref('')

// 驳回（audit=2）必须填原因：作者需要知道为什么没通过
const needReason = computed(() => Number(props.audit) === 2)

const resolvedTitle = computed(
  () => props.title || (needReason.value ? '驳回并填写原因' : '打回待审核')
)

const resolvedMessage = computed(
  () =>
    props.message ||
    (needReason.value
      ? '驳回后作者会收到通知，其中包含你填写的原因。'
      : '打回后该内容会重新进入待审核队列。')
)

const resolvedConfirmText = computed(() => props.confirmText || (needReason.value ? '驳回' : '确定'))

// 每次打开时用「目标条目已有的原因」回填（没有则清空），避免复用到别的条目上
watch(
  () => props.visible,
  (val) => {
    if (val) reason.value = String(props.defaultReason || '')
  }
)

function onConfirm() {
  const text = reason.value.trim()
  if (needReason.value && !text) {
    toast.error('请填写驳回原因')
    return
  }
  emit('confirm', text)
}

function onCancel() {
  reason.value = ''
  emit('cancel')
}
</script>

<style scoped>
/* 归属人一行与下面的原因输入框留点距离 */
.audit-owner {
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--border-soft);
}
.audit-reason {
  display: block;
}
.audit-reason-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.audit-reason-tag {
  padding: 0 6px;
  font-size: 11px;
  font-style: normal;
  font-weight: 500;
  border-radius: 999px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.audit-reason-tag.is-required {
  color: var(--danger);
  background: rgba(217, 84, 77, 0.12);
}
.audit-reason-input {
  margin-top: 8px;
  width: 100%;
  font-size: 13px;
  resize: vertical;
}
.audit-reason-hint {
  display: block;
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-light);
}
</style>
