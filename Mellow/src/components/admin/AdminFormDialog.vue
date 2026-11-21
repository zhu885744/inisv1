<template>
  <Teleport to="body">
    <Transition name="admin-form-dialog-fade">
      <div v-if="visible" class="admin-form-dialog-mask" @click.self="onCancel">
        <div class="admin-form-dialog-box" :style="{ maxWidth: width }" role="dialog" aria-modal="true" :aria-label="title">
          <header class="admin-form-dialog-head">
            <span class="admin-form-dialog-title">
              <i v-if="icon" :class="icon" />
              {{ title }}
            </span>
            <button class="admin-form-dialog-close" type="button" :disabled="loading" aria-label="关闭" @click="onCancel">
              <i class="bi bi-x-lg" />
            </button>
          </header>

          <div class="admin-form-dialog-body">
            <slot />
          </div>

          <footer class="admin-form-dialog-foot">
            <slot name="footer">
              <button class="btn" type="button" :disabled="loading" @click="onCancel">
                {{ cancelText }}
              </button>
              <button class="btn btn-primary" type="button" :disabled="loading" @click="$emit('confirm')">
                {{ loading ? loadingText : confirmText }}
              </button>
            </slot>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * 后台通用表单弹窗（标题 + 内容插槽 + 底部操作）
 * 与 ConfirmDialog 保持一致的视觉与交互：点击遮罩关闭、处理中禁止关闭。
 * 具体表单内容由使用方通过默认插槽提供，底部按钮可通过 footer 插槽整体替换。
 */
defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '' },
  // 标题左侧图标类名（如 bi bi-plus-circle）
  icon: { type: String, default: '' },
  width: { type: String, default: '520px' },
  confirmText: { type: String, default: '保存' },
  cancelText: { type: String, default: '取消' },
  loadingText: { type: String, default: '处理中...' },
  loading: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'confirm', 'cancel'])

function onCancel() {
  // 处理中不允许关闭，避免中断进行中的请求
  emit('update:visible', false)
  emit('cancel')
}
</script>

<style scoped>
.admin-form-dialog-mask {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}
.admin-form-dialog-box {
  width: 100%;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.admin-form-dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-soft);
}
.admin-form-dialog-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.admin-form-dialog-title .bi {
  font-size: 15px;
  color: var(--primary);
}
.admin-form-dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
}
.admin-form-dialog-close:hover:not(:disabled) {
  background: var(--bg-muted);
  color: var(--text);
}
.admin-form-dialog-close:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.admin-form-dialog-body {
  padding: 18px;
  overflow-y: auto;
}

.admin-form-dialog-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid var(--border-soft);
}

/* 过渡：遮罩淡入，卡片轻微上浮 */
.admin-form-dialog-fade-enter-active,
.admin-form-dialog-fade-leave-active {
  transition: opacity 0.2s ease;
}
.admin-form-dialog-fade-enter-active .admin-form-dialog-box,
.admin-form-dialog-fade-leave-active .admin-form-dialog-box {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.admin-form-dialog-fade-enter-from,
.admin-form-dialog-fade-leave-to {
  opacity: 0;
}
.admin-form-dialog-fade-enter-from .admin-form-dialog-box,
.admin-form-dialog-fade-leave-to .admin-form-dialog-box {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

@media (max-width: 640px) {
  .admin-form-dialog-box {
    max-width: none;
  }
}
</style>
