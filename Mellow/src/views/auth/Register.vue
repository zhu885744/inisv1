<template>
  <div>
    <h2 class="title">加入我们</h2>
    <p class="subtitle">注册账号，开启你的创作之旅</p>

    <!-- 注册后需要人工审核时展示的提示 -->
    <div v-if="notice" class="notice">
      <i class="bi bi-hourglass-split notice-icon" />
      <p class="notice-text">{{ notice.message }}</p>
      <router-link to="/auth/login" class="btn btn-primary btn-block btn-lg notice-action">去登录</router-link>
    </div>

    <form v-else class="auth-form" @submit.prevent="handleRegister">
      <div class="form-item">
        <label class="form-label">邮箱 / 手机号 *</label>
        <FormInput
          v-model="form.social"
          required
          placeholder="用于接收验证码" />
      </div>

      <div class="form-item">
        <label class="form-label">验证码 *</label>
        <div class="code-row">
          <FormInput v-model="form.code" required maxlength="6" />
          <button
            type="button"
            class="btn"
            :disabled="sending || countdown > 0"
            @click="sendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重试` : '发送验证码' }}
          </button>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">密码 *</label>
        <FormInput
          v-model="form.password"
          type="password"
          required
          minlength="6"
          autocomplete="new-password" />
      </div>

      <div class="form-item">
        <label class="form-label">确认密码 *</label>
        <FormInput
          v-model="form.password2"
          type="password"
          required />
        <p v-if="form.password2 && form.password !== form.password2" class="form-error">
          两次密码不一致
        </p>
      </div>

      <AuthAgreement v-model="agreed" ref="agreeRef" required />

      <button class="btn btn-primary btn-block btn-lg register-btn" :disabled="loading" type="submit">
        {{ loading ? '注册中...' : '注 册' }}
      </button>
    </form>

    <!-- 第三方注册（QQ / GitHub / Gitee）：后台开启后才会出现 -->
    <SocialLogin mode="register" />

    <div class="auth-bottom">
      已有账号？<router-link to="/auth/login">立即登录</router-link>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { register, registerSendCode } from '@/api/comm'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { finishPendingBind } from '@/utils/oauth'
import AuthAgreement from '@/components/AuthAgreement.vue'
import SocialLogin from '@/components/SocialLogin.vue'

const router = useRouter()
const userStore = useUserStore()

const form = ref({
  social: '',
  code: '',
  password: '',
  password2: ''
})

const loading = ref(false)
const sending = ref(false)
const countdown = ref(0)
let timer = null

const agreed = ref(false)
const agreeRef = ref(null)

// 注册结果需要后续确认时的提示（后台开启「人工审核」才会出现）
const notice = ref(null)

async function sendCode() {
  if (!form.value.social) {
    toast.warning('请输入邮箱或手机号')
    return
  }
  sending.value = true
  try {
    await registerSendCode(form.value.social)
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

async function handleRegister() {
  if (!form.value.password || form.value.password.length < 6) {
    toast.warning('密码至少 6 位')
    return
  }
  if (form.value.password !== form.value.password2) {
    toast.warning('两次密码不一致')
    return
  }
  if (agreeRef.value && !agreeRef.value.validate()) {
    toast.warning('请先阅读并同意用户协议和隐私政策')
    return
  }
  loading.value = true
  try {
    const res = await register(form.value.social, form.value.code, form.value.password)
    if (res.code === 200) {
      const data = res.data || {}

      // 后台开启「人工审核」：账号进入待审核，管理员通过后才能登录
      if (data.need_audit) {
        notice.value = {
          message: res.msg || '注册成功，请等待管理员审核通过后再登录！'
        }
        return
      }

      const validTime = Number(data.valid_time) > 0 ? Number(data.valid_time) : 15 * 24 * 60 * 60
      if (data.token) {
        userStore.persistToken(data.token, validTime)
      }
      userStore.setUser(data.user, validTime)
      // 注册前若发起过第三方登录但未绑定，这里自动补上绑定（票据来自 /auth/oauth 回调页）
      await finishPendingBind()
      toast.success('注册成功，已自动登录')
      router.replace('/')
    }
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
.code-row {
  display: flex;
  gap: 8px;
}
.code-row .input {
  flex: 1;
}
.auth-bottom {
  text-align: center;
  margin-top: 16px;
  font-size: 13px;
  color: var(--text-muted);
}

/* 注册后的确认提示（人工审核） */
.notice {
  text-align: center;
}
.notice-icon {
  display: block;
  margin: 0 auto 12px;
  font-size: 42px;
  color: var(--primary);
}
.notice-text {
  margin-bottom: 10px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-soft);
}
.notice-action {
  margin-top: 10px;
}

</style>