<template>
  <Teleport to="body">
    <Transition name="confirm-fade">
      <div v-if="visible" class="confirm-mask" @click.self="onCancel">
        <div class="confirm-box card card-pad" role="dialog" aria-modal="true" :aria-label="title">
          <h3 class="confirm-title">
            <i v-if="danger" class="bi bi-exclamation-triangle-fill confirm-title-icon" aria-hidden="true" />
            {{ title }}
          </h3>
          <p v-if="message" class="confirm-message">{{ message }}</p>
          <!-- 附加内容（如危险操作的额外选项）：不传插槽时不渲染 -->
          <div v-if="$slots.default" class="confirm-extra">
            <slot />
          </div>
          <div class="confirm-actions">
            <button type="button" class="btn" :disabled="loading" @click="onCancel">
              {{ cancelText }}
            </button>
            <button
              ref="confirmBtn"
              type="button"
              class="btn"
              :class="danger ? 'btn-danger' : 'btn-primary'"
              :disabled="loading"
              @click="onConfirm"
            >
              {{ loading ? loadingText : confirmText }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '操作确认' },
  message: { type: String, default: '' },
  confirmText: { type: String, default: '确定' },
  cancelText: { type: String, default: '取消' },
  loadingText: { type: String, default: '处理中...' },
  // 危险操作：确认按钮使用 danger 配色，标题带警示图标
  danger: { type: Boolean, default: false },
  // 处理中：两个按钮禁用，遮罩不可关闭
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

const confirmBtn = ref(null)

function close() {
  emit('update:visible', false)
}

function onConfirm() {
  if (props.loading) return
  emit('confirm')
}

function onCancel() {
  // 处理中不允许关闭，避免中断进行中的请求
  if (props.loading) return
  close()
  emit('cancel')
}

// 打开时聚焦确认按钮，便于键盘操作
watch(
  () => props.visible,
  async (val) => {
    if (!val) return
    await nextTick()
    confirmBtn.value?.focus()
  }
)
</script>

<style scoped>
.confirm-mask {
  position: fixed;
  inset: 0;
  /* 页面弹窗 3000 < 附件库 3500 < 确认弹窗 3600：确认要能盖在附件库之上 */
  z-index: 3600;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}
.confirm-box {
  width: 100%;
  max-width: 380px;
}
.confirm-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.confirm-title-icon {
  font-size: 15px;
  color: var(--danger);
}
.confirm-message {
  margin: 10px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}
/* 附加内容插槽（如危险操作的额外选项） */
.confirm-extra {
  margin-top: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-muted);
}
.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

/* 过渡：遮罩淡入，卡片轻微上浮 */
.confirm-fade-enter-active,
.confirm-fade-leave-active {
  transition: opacity 0.2s ease;
}
.confirm-fade-enter-active .confirm-box,
.confirm-fade-leave-active .confirm-box {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.confirm-fade-enter-from,
.confirm-fade-leave-to {
  opacity: 0;
}
.confirm-fade-enter-from .confirm-box,
.confirm-fade-leave-to .confirm-box {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

@media (max-width: 640px) {
  .confirm-box {
    max-width: none;
  }
}
</style>
