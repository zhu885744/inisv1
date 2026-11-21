<template>
  <div>
    <h2 class="title">找回密码</h2>
    <p class="subtitle">通过邮箱或手机号重置密码</p>

    <!-- 步骤条 -->
    <div class="steps">
      <div :class="['step', { 'is-active': step >= 1 }]">
        <span class="num">1</span>
        <span class="label">验证身份</span>
      </div>
      <div class="line" :class="{ 'is-active': step >= 2 }"></div>
      <div :class="['step', { 'is-active': step >= 2 }]">
        <span class="num">2</span>
        <span class="label">设置新密码</span>
      </div>
    </div>

    <!-- 第一步：发送验证码 -->
    <form v-if="step === 1" class="auth-form" @submit.prevent="onSendCode">
      <div class="form-item">
        <label class="form-label">邮箱 / 手机号 *</label>
        <FormInput v-model="form.social" required placeholder="已注册的邮箱或手机号" />
      </div>

      <div class="form-item">
        <label class="form-label">验证码 *</label>
        <div class="code-row">
          <FormInput v-model="form.code" required maxlength="6" />
          <button
            type="button"
            class="btn"
            :disabled="sending || countdown > 0"
            @click="onSendCode(false)"
          >
            {{ countdown > 0 ? `${countdown}s 后重试` : '发送验证码' }}
          </button>
        </div>
      </div>

      <button class="btn btn-primary btn-block btn-lg" type="submit">
        下一步
      </button>
    </form>

    <!-- 第二步：重置密码 -->
    <form v-else class="auth-form" @submit.prevent="onSubmit">
      <div class="form-item">
        <label class="form-label">新密码 *</label>
        <FormInput
          v-model="form.password"
          type="password"
          minlength="6"
          required />
      </div>
      <div class="form-item">
        <label class="form-label">确认新密码 *</label>
        <FormInput
          v-model="form.password2"
          type="password"
          minlength="6"
          required />
        <p v-if="form.password2 && form.password !== form.password2" class="form-error">
          两次密码不一致
        </p>
      </div>
      <button class="btn btn-primary btn-block btn-lg" :disabled="loading" type="submit">
        {{ loading ? '提交中...' : '重置密码' }}
      </button>
    </form>

    <AuthAgreement />

    <div class="auth-bottom">
      想起来了？<router-link to="/auth/login">立即登录</router-link>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { resetPasswordSendCode, resetPassword } from '@/api/comm'
import { toast } from '@/utils/toast'
import AuthAgreement from '@/components/AuthAgreement.vue'

const router = useRouter()

const step = ref(1)
const sending = ref(false)
const loading = ref(false)
const countdown = ref(0)
let timer = null

const form = ref({
  social: '',
  code: '',
  password: '',
  password2: ''
})

async function onSendCode(e) {
  e?.preventDefault?.()
  if (!form.value.social) {
    toast.warning('请输入邮箱或手机号')
    return
  }

  // 如果是直接提交 step1
  if (typeof e === 'object' && e?.type === 'submit') {
    if (!form.value.code) {
      toast.warning('请输入验证码')
      return
    }
    step.value = 2
    return
  }

  sending.value = true
  try {
    await resetPasswordSendCode(form.value.social)
    toast.success('验证码已发送，请注意查收')
    countdown.value = 60
    timer = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) clearInterval(timer)
    }, 1000)
  } catch {} finally {
    sending.value = false
  }
}

async function onSubmit() {
  if (!form.value.password || form.value.password.length < 6) {
    toast.warning('密码至少 6 位')
    return
  }
  if (form.value.password !== form.value.password2) {
    toast.warning('两次密码不一致')
    return
  }
  loading.value = true
  try {
    await resetPassword(form.value.social, form.value.code, form.value.password)
    toast.success('密码重置成功，请重新登录')
    router.replace('/auth/login')
  } catch {} finally {
    loading.value = false
  }
}

onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.title {
  text-align: center;
  font-family: var(--font-serif);
  font-size: 22px;
  margin-bottom: 8px;
}
.subtitle {
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
  margin-bottom: 24px;
}
.steps {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 24px;
}
.step {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-muted);
  font-size: 13px;
}
.step .num {
  display: inline-flex;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid var(--border);
  align-items: center;
  justify-content: center;
  background: var(--bg-card);
  color: var(--text-muted);
  font-size: 12px;
  transition: all 0.3s;
}
.step.is-active {
  color: var(--primary-deep);
}
.step.is-active .num {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}
.line {
  width: 40px;
  height: 1px;
  background: var(--border);
  margin: 0 16px;
}
.line.is-active {
  background: var(--primary);
}

.code-row {
  display: flex;
  gap: 8px;
}
.code-row .input {
  flex: 1;
}
.auth-bottom {
  text-align: center;
  margin-top: 24px;
  font-size: 13px;
  color: var(--text-muted);
}
</style>