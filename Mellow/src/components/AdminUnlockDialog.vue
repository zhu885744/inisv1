<template>
  <Teleport to="body">
    <Transition name="unlock-fade">
      <div v-if="visible" class="unlock-mask" @click.self="onCancel">
        <form
          class="unlock-box card card-pad"
          role="dialog"
          aria-modal="true"
          aria-label="进入后台需要验证"
          @submit.prevent="onConfirm"
        >
          <h3 class="unlock-title">
            <i class="bi bi-shield-lock-fill unlock-title-icon" aria-hidden="true" />
            进入后台需要验证
          </h3>
          <p class="unlock-message">{{ hint }}</p>

          <div class="form-item">
            <label class="form-label" for="admin-unlock-password">登录密码</label>
            <FormInput
              id="admin-unlock-password"
              ref="inputRef"
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="请输入当前账号的登录密码"
              :disabled="loading" />
          </div>

          <p v-if="error" class="unlock-error">
            <i class="bi bi-exclamation-circle" aria-hidden="true" />
            {{ error }}
          </p>

          <p class="unlock-tip">
            后台使用独立的管理会话，退出登录或会话过期后需要重新验证。
          </p>

          <div class="unlock-actions">
            <button type="button" class="btn" :disabled="loading" @click="onCancel">取消</button>
            <button type="submit" class="btn btn-primary" :disabled="loading || !password.trim()">
              {{ loading ? '验证中...' : '验证' }}
            </button>
          </div>
        </form>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * AdminUnlockDialog - 全局「重新验证密码」弹窗（后台管理会话专用）
 *
 * 与后端的协作方式：
 *   1. 后台会话缺失 / 过期时，后端对 /admin 下的请求统一返回 401 + auth:'admin'；
 *   2. api/request.js 把这次失败交给 utils/adminSession → 本组件弹出；
 *   3. 用户输入密码，本组件调用 comm/admin-login 换一条后台会话（scope=admin）；
 *   4. 成功后拦截器自动重放原请求，调用方（页面组件）完全无感知。
 *
 * 为什么不用原生 DOM 弹窗：视觉 / 表单控件 / 按钮态都要与主题一致，
 * 这里直接复用 .card / .card-pad / .input / .form-label / .btn 等全局样式。
 */
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { registerAdminUnlockHandler, unregisterAdminUnlockHandler } from '@/utils/adminSession'
import { adminLogin } from '@/api/comm'

const visible = ref(false)
const hint = ref('')
const error = ref('')
const password = ref('')
const loading = ref(false)

const inputRef = ref(null)

// 当前弹窗对应的 Promise resolve（一次只处理一个请求）
let resolver = null

function open(message) {
  hint.value = message || '后台登录态已失效，请重新输入登录密码。'
  error.value = ''
  password.value = ''
  loading.value = false
  visible.value = true
  nextTick(() => inputRef.value?.focus())
}

function close(ok) {
  visible.value = false
  error.value = ''
  password.value = ''
  const done = resolver
  resolver = null
  if (done) done(ok)
}

function onCancel() {
  // 处理中不允许关闭，避免中断进行中的验证请求
  if (loading.value) return
  close(false)
}

async function onConfirm() {
  if (loading.value) return

  const value = password.value.trim()
  if (!value) {
    error.value = '请输入登录密码'
    return
  }

  loading.value = true
  error.value = ''
  try {
    await adminLogin(value)
    close(true)
  } catch (err) {
    // 密码错误 / 没有后台权限：留在弹窗里继续试，文案用后端返回的原因
    error.value = err?.msg || '验证失败，请重试'
    password.value = ''
    nextTick(() => inputRef.value?.focus())
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  registerAdminUnlockHandler((message) => {
    // 上一次没结束就再来一次：把上一次当作「取消」，避免旧 Promise 永远挂起
    if (resolver) {
      const stale = resolver
      resolver = null
      stale(false)
    }
    return new Promise((resolve) => {
      resolver = resolve
      open(message)
    })
  })
})

onUnmounted(() => {
  unregisterAdminUnlockHandler()
  if (resolver) {
    const done = resolver
    resolver = null
    done(false)
  }
})
</script>

<style scoped>
.unlock-mask {
  position: fixed;
  inset: 0;
  /* 与确认弹窗同级：后台会话失效时要能盖在任何页面弹窗之上 */
  z-index: 3600;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}
.unlock-box {
  width: 100%;
  max-width: 380px;
}
.unlock-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.unlock-title-icon {
  font-size: 15px;
  color: var(--primary);
}
.unlock-message {
  margin: 10px 0 16px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}
.unlock-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: -6px 0 12px;
  font-size: 13px;
  color: var(--danger);
}
.unlock-tip {
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.unlock-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
}

/* 过渡：遮罩淡入，卡片轻微上浮（与确认弹窗保持一致） */
.unlock-fade-enter-active,
.unlock-fade-leave-active {
  transition: opacity 0.2s ease;
}
.unlock-fade-enter-active .unlock-box,
.unlock-fade-leave-active .unlock-box {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.unlock-fade-enter-from,
.unlock-fade-leave-to {
  opacity: 0;
}
.unlock-fade-enter-from .unlock-box,
.unlock-fade-leave-to .unlock-box {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

@media (max-width: 640px) {
  .unlock-box {
    max-width: none;
  }
}
</style>
