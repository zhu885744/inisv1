<template>
  <div>
    <h2 class="title">欢迎回来！</h2>
    <p class="subtitle">登录以访问更多功能</p>

    <!-- 登录方式切换：密码登录 / 验证码登录（免密） -->
    <div class="login-tabs">
      <button
        v-for="item in tabs"
        :key="item.value"
        type="button"
        class="login-tab"
        :class="{ 'is-active': mode === item.value }"
        @click="mode = item.value"
      >
        {{ item.label }}
      </button>
    </div>

    <!-- ===== 密码登录 ===== -->
    <form v-if="mode === 'password'" class="auth-form" @submit.prevent="handleLogin">
      <div class="form-item">
        <label class="form-label">账号 / 邮箱 / 手机号</label>
        <FormInput
          v-model="account"
          type="text"
          autocomplete="username"
          required
          placeholder="请输入账号" />
      </div>

      <div class="form-item">
        <label class="form-label">密码</label>
        <FormInput
          v-model="password"
          type="password"
          autocomplete="current-password"
          required
          placeholder="请输入密码" />
      </div>

      <div class="form-actions">
        <router-link to="/auth/forgot" class="link">忘记密码？</router-link>
      </div>

      <button class="btn btn-primary btn-block btn-lg" :disabled="loading" type="submit">
        {{ loading ? '登录中...' : '登 录' }}
      </button>

      <AuthAgreement />
    </form>

    <!-- ===== 验证码登录（免密） ===== -->
    <form v-else class="auth-form" @submit.prevent="handleCodeLogin">
      <div class="form-item">
        <label class="form-label">邮箱 / 手机号</label>
        <FormInput
          v-model="codeForm.social"
          type="text"
          autocomplete="username"
          required
          placeholder="请输入邮箱或手机号" />
      </div>

      <div class="form-item">
        <label class="form-label">验证码</label>
        <div class="code-row">
          <FormInput
            v-model="codeForm.code"
            required
            maxlength="6"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="请输入验证码" />
          <button
            type="button"
            class="btn"
            :disabled="sending || countdown > 0"
            @click="sendCode"
          >
            {{ countdown > 0 ? `${countdown}s 后重试` : '发送验证码' }}
          </button>
        </div>
        <p class="form-hint">验证码 5 分钟内有效，仅可使用一次</p>
      </div>

      <button class="btn btn-primary btn-block btn-lg" :disabled="loading" type="submit">
        {{ loading ? '登录中...' : '登 录' }}
      </button>

      <AuthAgreement />
    </form>

    <!-- 第三方登录（QQ / GitHub / Gitee）：后台开启后才会出现 -->
    <SocialLogin mode="login" />

    <div class="auth-bottom">
      还没有账号？<router-link to="/auth/register">立即注册</router-link>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { login, signCode, signCodeSend } from '@/api/comm'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { finishPendingBind } from '@/utils/oauth'
import AuthAgreement from '@/components/AuthAgreement.vue'
import SocialLogin from '@/components/SocialLogin.vue'

// 登录方式：password 账号密码 / code 验证码（免密）
const mode = ref('password')
const tabs = [
  { value: 'password', label: '密码登录' },
  { value: 'code', label: '验证码登录' }
]

// 密码登录
const account = ref('')
const password = ref('')
const loading = ref(false)

// 验证码登录
const codeForm = ref({ social: '', code: '' })
const sending = ref(false)
const countdown = ref(0)
let timer = null

const userStore = useUserStore()
const router = useRouter()
const route = useRoute()

// 登录成功后的统一收尾：持久化 token / 用户信息 → 刷新登录态 → 回跳
async function afterLogin(data, message) {
  const validTime = Number(data.valid_time) > 0 ? Number(data.valid_time) : 15 * 24 * 60 * 60
  // 关键：持久化 token（cookie + localStorage），后续请求拦截器注入 Authorization 才能通过鉴权
  if (data.token) {
    userStore.persistToken(data.token, validTime)
  }
  userStore.setUser(data.user, validTime)
  // 用 check-token 的权威结构刷新用户信息（确保 result.auth 等字段完整）
  userStore.checkLoginState()
  // 登录前若发起过第三方登录但未绑定，这里自动补上绑定（票据来自 /auth/oauth 回调页）
  await finishPendingBind()
  toast.success(message)
  const redirect = route.query.redirect || '/'
  router.replace(redirect)
}

async function handleLogin() {
  if (!account.value || !password.value) {
    toast.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    const res = await login(account.value, password.value)
    if (res.code === 200 || res.code === 201) {
      await afterLogin(res.data || {}, '登录成功')
    }
  } catch (e) {
    // toast 已显示
  } finally {
    loading.value = false
  }
}

// 发送验证码（此阶段不校验账号是否注册，后端统一返回发送成功）
async function sendCode() {
  if (!codeForm.value.social) {
    toast.warning('请输入邮箱或手机号')
    return
  }
  sending.value = true
  try {
    const res = await signCodeSend(codeForm.value.social)
    toast.success(res.msg || '验证码已发送，请注意查收')
    countdown.value = 60
    clearInterval(timer)
    timer = setInterval(() => {
      countdown.value -= 1
      if (countdown.value <= 0) clearInterval(timer)
    }, 1000)
  } catch (e) {
    // toast 已显示（如未开启邮箱/短信服务、发送过于频繁）
  } finally {
    sending.value = false
  }
}

async function handleCodeLogin() {
  if (!codeForm.value.social) {
    toast.warning('请输入邮箱或手机号')
    return
  }
  if (!codeForm.value.code) {
    toast.warning('请输入验证码')
    return
  }
  loading.value = true
  try {
    // 注意：验证码一经校验即失效，失败需重新发送，因此不做自动重试
    const res = await signCode(codeForm.value.social, codeForm.value.code)
    if (res.code === 200) {
      await afterLogin(res.data || {}, '登录成功')
    }
  } catch (e) {
    // toast 已显示
  } finally {
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
  margin-bottom: 20px;
}
/* 登录方式切换 */
.login-tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  margin-bottom: 20px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.04);
}
.login-tab {
  flex: 1;
  padding: 8px 0;
  font-size: 13px;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: color 0.2s, background 0.2s;
}
.login-tab.is-active {
  /* 用卡片色而不是 #fff：深色模式下白底 + 浅色文字会几乎不可读 */
  color: var(--text-strong, var(--text));
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
}
.auth-form {
  margin-bottom: 16px;
}
.code-row {
  display: flex;
  gap: 8px;
}
.code-row .input {
  flex: 1;
}
.form-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.form-actions {
  display: flex;
  justify-content: flex-end;
  margin: -8px 0 16px;
}
.link {
  font-size: 12px;
  color: var(--text-muted);
}
.link:hover {
  color: var(--primary);
}
.auth-bottom {
  text-align: center;
  margin-top: 24px;
  font-size: 13px;
  color: var(--text-muted);
}
</style>
