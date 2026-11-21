<template>
  <div v-if="ready" class="auth-agreement">
    <!-- 勾选模式（注册页） -->
    <label v-if="required" class="agree-row">
      <FormCheckbox v-model="checked" class="agree-check" />
      <span class="agree-text">
        我已阅读并同意
        <a class="agree-link" @click.stop.prevent="open('user')">《用户协议》</a>
        和
        <a class="agree-link" @click.stop.prevent="open('privacy')">《隐私协议》</a>
      </span>
    </label>
    <!-- 提示模式（登录/找回密码页） -->
    <p v-else class="agree-tip">
      登录即代表同意
      <a class="agree-link" @click.stop.prevent="open('user')">《用户协议》</a>
      和
      <a class="agree-link" @click.stop.prevent="open('privacy')">《隐私协议》</a>
    </p>

    <!-- 协议弹窗 -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="dialog" class="agree-overlay" @click.self="close">
          <div class="agree-dialog">
            <div class="agree-header">
              <span class="agree-title"><i class="bi bi-file-text" /> {{ dialog.title }}</span>
              <button class="btn btn-icon btn-sm btn-round" @click="close"><i class="bi bi-x-lg" /></button>
            </div>
            <div class="agree-body">{{ dialog.content }}</div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useSiteStore } from '@/stores/site'

const siteStore = useSiteStore()

const props = defineProps({
  required: { type: Boolean, default: false },
  modelValue: { type: Boolean, default: false }
})
const emit = defineEmits(['update:modelValue'])

const checked = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

const ready = ref(false)
const dialog = ref(null)

// 默认协议内容（与后台「网站设置」SiteSettingsForm.vue 默认值保持一致，配置缺失时兜底）
const DEFAULT_USER = '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议。\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性。\n\n3. 用户行为规范\n请勿发布违法或侵犯他人权益的内容。'
const DEFAULT_PRIVACY = '隐私协议\n\n我们重视您的隐私。\n\n1. 收集的信息\n我们可能收集您的账户信息和使用数据。\n\n2. 信息使用\n用于提供和改进服务。\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。'

const conf = ref({ user: DEFAULT_USER, privacy: DEFAULT_PRIVACY })

async function load() {
  try {
    const cfg = await siteStore.load()
    const a = cfg?.auth_dialog_agreement || {}
    // 后台关闭了协议提示则不显示
    if (a.enabled === false) return
    conf.value = {
      user: a.user_agreement_content || DEFAULT_USER,
      privacy: a.privacy_agreement_content || DEFAULT_PRIVACY
    }
  } catch {
    // 请求失败时展示默认协议，保证页面可用
  }
  ready.value = true
}

function open(type) {
  dialog.value = {
    title: type === 'user' ? '用户协议' : '隐私协议',
    content: type === 'user' ? conf.value.user : conf.value.privacy
  }
  document.body.style.overflow = 'hidden'
}
function close() {
  dialog.value = null
  document.body.style.overflow = ''
}

// 供父组件校验：未启用协议提示或非勾选模式时视为已同意
function validate() {
  if (!ready.value || !props.required) return true
  return checked.value
}

onMounted(load)
onUnmounted(() => {
  document.body.style.overflow = ''
})

defineExpose({ validate })
</script>

<style scoped>
.auth-agreement {
  margin-top: 10px;
  text-align: center;
  font-size: 12px;
  color: var(--text-muted);
}
.agree-row {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
}
.agree-check {
  accent-color: var(--primary);
  cursor: pointer;
}
.agree-text,
.agree-tip {
  line-height: 1.6;
}
.agree-link {
  color: var(--primary-deep);
  cursor: pointer;
  text-decoration: none;
}
.agree-link:hover {
  color: var(--primary);
  text-decoration: underline;
}

/* 弹窗 */
.agree-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.agree-dialog {
  width: 100%;
  max-width: 560px;
  max-height: 75vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.agree-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: linear-gradient(135deg, var(--accent-soft), var(--accent-wash));
  border-bottom: 1px solid var(--border-soft);
  flex-shrink: 0;
}
.agree-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.agree-body {
  padding: 20px;
  overflow-y: auto;
  font-size: 13px;
  line-height: 1.8;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.25s;
}
.modal-fade-enter-active .agree-dialog,
.modal-fade-leave-active .agree-dialog {
  transition: transform 0.25s, opacity 0.25s;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-from .agree-dialog,
.modal-fade-leave-to .agree-dialog {
  transform: scale(0.95) translateY(-10px);
  opacity: 0;
}
</style>
