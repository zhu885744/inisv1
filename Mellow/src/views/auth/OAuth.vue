<template>
  <!--
    第三方登录回调页（/auth/oauth）
    第三方带 code/state 跳回来，这里校验 state 后把 code 交给后端换登录态：
      - 已绑定 → 直接登录（收尾同密码登录：持久化 token + 用户信息）
      - 未绑定 + 未登录 → 展示选择页：创建新账号 / 输入账号密码绑定已有账号
      - 未绑定 + 已登录 → 直接绑定到当前账号
    选择页需要「创建新账号」时由后端 can_register 决定是否可选（注册总开关 + 后台允许第三方建号）。
  -->
  <div class="oauth-callback">
    <!-- 处理中 -->
    <template v-if="status === 'loading'">
      <div class="oauth-loading">
        <span class="spinner" />
        <p class="oauth-text">正在登录，请稍候…</p>
      </div>
    </template>

    <!-- 注册成功但需要人工审核（后端不发 token） -->
    <template v-else-if="status === 'audit'">
      <div class="oauth-icon is-info"><i class="bi bi-hourglass-split" /></div>
      <h3 class="oauth-title">注册成功</h3>
      <p class="oauth-text">账号正在审核中，管理员审核通过后即可使用{{ platformName }}登录。</p>
      <div class="oauth-actions">
        <router-link class="btn btn-primary btn-block" to="/auth/login">去登录</router-link>
        <router-link class="btn btn-block" to="/">返回首页</router-link>
      </div>
    </template>

    <!-- 未绑定：选择「创建新账号」或「绑定已有账号」 -->
    <template v-else-if="status === 'choice'">
      <div class="oauth-peer">
        <img v-if="peer.avatar" class="oauth-peer-avatar" :src="peer.avatar" alt="" loading="lazy" />
        <div class="oauth-peer-info">
          <div class="oauth-peer-name">{{ peer.nickname || '第三方账号' }}</div>
          <div class="oauth-peer-tip">该 {{ platformName }} 账号还未绑定本站账号</div>
        </div>
      </div>

      <div class="oauth-choice">
        <button
          type="button"
          class="oauth-choice-item"
          :class="{ 'is-active': mode === 'register' }"
          :disabled="!canRegister"
          @click="mode = 'register'"
        >
          <i class="bi bi-person-plus" />
          <span>创建新账号</span>
        </button>
        <button
          type="button"
          class="oauth-choice-item"
          :class="{ 'is-active': mode === 'bind' }"
          @click="mode = 'bind'"
        >
          <i class="bi bi-link-45deg" />
          <span>绑定已有账号</span>
        </button>
      </div>

      <p v-if="!canRegister" class="oauth-note">
        本站未开放「第三方账号创建新账号」，请用已有账号绑定；还没有账号请先
        <router-link to="/auth/register">注册</router-link>。
      </p>

      <!--
        创建新账号：昵称头像沿用第三方，但邮箱与密码必须自己填。
        邮箱用来登录 / 找回密码，必须通过验证码验证；密码按注册规则校验强度。
      -->
      <form v-if="mode === 'register'" class="oauth-form" @submit.prevent="doRegister">
        <p class="oauth-note">
          将使用 {{ platformName }} 的昵称与头像建号，邮箱与密码需自行设置（用于登录与找回密码）。
        </p>
        <FormInput
          v-model="registerForm.email"
          type="email"
          autocomplete="email"
          placeholder="邮箱" />
        <div class="oauth-code-row">
          <FormInput
            v-model="registerForm.code"
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            placeholder="邮箱验证码" />
          <button
            type="button"
            class="btn"
            :disabled="sendingCode || codeCountdown > 0 || submitting"
            @click="sendEmailCode"
          >{{ codeCountdown > 0 ? `${codeCountdown}s` : '获取验证码' }}</button>
        </div>
        <FormInput
          v-model="registerForm.password"
          type="password"
          autocomplete="new-password"
          placeholder="设置密码（至少 6 位，不能是纯数字）" />
        <FormInput
          v-model="registerForm.password2"
          type="password"
          autocomplete="new-password"
          placeholder="确认密码" />
        <button class="btn btn-primary btn-block" type="submit" :disabled="submitting">
          {{ submitting ? '处理中…' : '创建并登录' }}
        </button>
      </form>

      <!-- 绑定已有账号：账号 + 密码校验通过后绑定并登录 -->
      <form v-else class="oauth-form" @submit.prevent="doBindAccount">
        <FormInput
          v-model="form.account"
          type="text"
          autocomplete="username"
          placeholder="账号 / 邮箱 / 手机号" />
        <FormInput
          v-model="form.password"
          type="password"
          autocomplete="current-password"
          placeholder="密码" />
        <button class="btn btn-primary btn-block" type="submit" :disabled="submitting">
          {{ submitting ? '绑定中…' : '绑定并登录' }}
        </button>
      </form>

      <div class="oauth-actions">
        <router-link class="btn btn-block" to="/auth/login">去登录页</router-link>
      </div>
    </template>

    <!-- 失败 -->
    <template v-else>
      <div class="oauth-icon is-error"><i class="bi bi-exclamation-triangle" /></div>
      <h3 class="oauth-title">登录失败</h3>
      <p class="oauth-text">{{ message }}</p>
      <div class="oauth-actions">
        <router-link class="btn btn-primary btn-block" to="/auth/login">返回登录</router-link>
        <router-link class="btn btn-block" to="/">返回首页</router-link>
      </div>
    </template>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { oauthBindAccount, oauthRegisterByTicket, oauthSignIn } from '@/api/oauth'
// 邮箱验证码复用常规注册的发码接口（后端缓存键同为 [register][email=xxx]）
import { registerSendCode } from '@/api/comm'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import {
  clearBindTicket,
  finishPendingBind,
  platformFromState,
  saveBindTicket,
  takeCallbackContext
} from '@/utils/oauth'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

// loading / choice / audit / error
const status = ref('loading')
const message = ref('')
const platformCode = ref('')

// 未绑定时「选择页」的状态
const ticket = ref('')                            // 后端下发的一次性票据（10 分钟）
const callbackContext = ref(null)                 // 发起时记录的 redirect / platform
const mode = ref('register')                      // register：创建新账号；bind：绑定已有账号
const canRegister = ref(false)                    // 后端 can_register：注册开关 + 允许第三方建号
const submitting = ref(false)
const peer = ref({ nickname: '', avatar: '' })    // 第三方昵称 / 头像（选择页展示）
const form = ref({ account: '', password: '' })

const nameMap = { qq: 'QQ', github: 'GitHub', gitee: 'Gitee' }
const platformName = computed(() => nameMap[platformCode.value] || '第三方账号')

function fail(text) {
  status.value = 'error'
  message.value = text
}

// 登录 / 注册成功后的统一收尾（与密码登录一致）
function afterSignIn(data, context) {
  const validTime = Number(data.valid_time) > 0 ? Number(data.valid_time) : 15 * 24 * 60 * 60
  if (data.token) {
    userStore.persistToken(data.token, validTime)
  }
  userStore.setUser(data.user, validTime)
  // 用 check-token 的权威结构刷新用户信息（确保 result.auth 等字段完整）
  userStore.checkLoginState()
  toast.success(data.register ? '注册成功，已自动登录' : '登录成功')
  router.replace(context.redirect || '/')
}

// 选择页：未登录 + 未绑定，由用户决定「创建新账号」还是「绑定已有账号」
function beginChoice(data, context) {
  ticket.value = String(data.ticket || '')
  peer.value = { nickname: data.nickname || '', avatar: data.avatar || '' }
  // 后端下发的是布尔值（true/false），Number(true) === 1 兼容两种写法
  canRegister.value = Number(data.can_register) === 1
  mode.value = canRegister.value ? 'register' : 'bind'
  callbackContext.value = context
  // 票据同时留一份在 sessionStorage：万一用户刷新了选择页，
  // 仍可走「去登录页 → 登录成功后自动绑定」这条兜底路径（后端票据未消费）
  saveBindTicket(ticket.value)
  status.value = 'choice'
}

/** 选择页两条路（创建新账号 / 绑定已有账号）的统一收尾 */
function applyAuthResult(data) {
  if (data.token && data.user) {
    clearBindTicket()
    afterSignIn(data, callbackContext.value || {})
    return
  }
  if (data.need_audit) {
    status.value = 'audit'
    return
  }
  fail('第三方登录未完成，请重新发起登录')
}

// 建号表单：邮箱 + 验证码 + 密码（后端会再校验一遍，前端只做即时提示）
const registerForm = ref({ email: '', code: '', password: '', password2: '' })
const sendingCode = ref(false)
const codeCountdown = ref(0)
let codeTimer = null

const EMAIL_RE = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/

async function sendEmailCode() {
  if (sendingCode.value || codeCountdown.value > 0) return
  const email = registerForm.value.email.trim()
  if (!EMAIL_RE.test(email)) {
    toast.warning('请先填写正确的邮箱')
    return
  }

  sendingCode.value = true
  try {
    await registerSendCode(email)
    toast.success('验证码已发送，请注意查收')
    codeCountdown.value = 60
    if (codeTimer) clearInterval(codeTimer)
    codeTimer = setInterval(() => {
      codeCountdown.value -= 1
      if (codeCountdown.value <= 0) {
        clearInterval(codeTimer)
        codeTimer = null
      }
    }, 1000)
  } catch {
    // 邮箱已注册 / 发送频繁 / 管理员未开启邮件服务等由拦截器提示
  } finally {
    sendingCode.value = false
  }
}

function validateRegisterForm() {
  const { email, code, password, password2 } = registerForm.value
  if (!EMAIL_RE.test(email.trim())) return '请填写正确的邮箱'
  if (!code.trim()) return '请填写邮箱验证码'
  if (!password) return '请设置密码'
  if (password.length < 6 || password.length > 20) return '密码长度需为 6-20 位'
  if (/\s/.test(password)) return '密码不能包含空格'
  if (/^\d+$/.test(password)) return '密码不能是纯数字，请混合字母或符号'
  if (password !== password2) return '两次输入的密码不一致'
  return ''
}

async function doRegister() {
  if (submitting.value) return

  const message = validateRegisterForm()
  if (message) {
    toast.warning(message)
    return
  }

  submitting.value = true
  try {
    const res = await oauthRegisterByTicket({
      key: ticket.value,
      email: registerForm.value.email.trim(),
      code: registerForm.value.code.trim(),
      password: registerForm.value.password
    })
    applyAuthResult(res?.data || {})
  } catch {
    // 失败原因（验证码错误 / 邮箱已注册 / 密码太弱 / 票据过期等）由请求拦截器统一 toast，这里保持选择页
  } finally {
    submitting.value = false
  }
}

async function doBindAccount() {
  if (submitting.value) return
  if (!form.value.account.trim() || !form.value.password) {
    toast.warning('请填写账号和密码')
    return
  }
  submitting.value = true
  try {
    const res = await oauthBindAccount({
      key: ticket.value,
      account: form.value.account.trim(),
      password: form.value.password
    })
    applyAuthResult(res?.data || {})
  } catch {
    // 密码错误 / 账户不存在 / 票据已过期等由请求拦截器统一 toast，这里停在表单上便于重试
  } finally {
    submitting.value = false
  }
}

async function handle(code, state, context) {
  try {
    const res = await oauthSignIn(context.platform, code, state)
    const data = res?.data || {}

    // 1) 登录 / 注册成功：后端下发了 token
    if (data.token && data.user) {
      afterSignIn(data, context)
      return
    }

    // 2) 建号成功但需要人工审核：不发 token
    if (data.need_audit) {
      status.value = 'audit'
      return
    }

    // 3) 未登录且未绑定：展示选择页（创建新账号 / 绑定已有账号）
    if (data.need_choice || data.need_bind) {
      beginChoice(data, context)
      return
    }

    // 4) 已登录状态下发起的绑定
    if (data.bind) {
      await finishPendingBind()
      toast.success('绑定成功')
      router.replace('/user/security')
      return
    }

    fail('第三方登录失败，请重新发起登录')
  } catch (e) {
    // 错误提示已在拦截器里给过，这里在页面上再保留一份原因（整页流程，toast 会消失）
    fail(e?.msg || '第三方登录失败，请稍后重试')
  }
}

onMounted(() => {
  // 带走并清空发起时记录的信息（刷新页面后由于 state 已焚，会给出「请重新发起」的提示）
  const context = takeCallbackContext()

  const code = String(route.query.code || '')
  const state = String(route.query.state || '')
  const error = String(route.query.error || '')

  // 平台来源：URL 参数 → 发起时记录的上下文 → state 前缀。
  // 手机端（尤其从 QQ 应用内发起）回调页常在新标签 / 内置浏览器里打开，
  // sessionStorage 是空的，这时只能从 state 前缀还原，否则会误报「未找到发起平台」。
  const platform = String(route.query.platform || context.platform || platformFromState(state) || '')
  platformCode.value = platform

  if (error) {
    fail(error === 'access_denied' ? '你取消了授权，或第三方平台拒绝了本次请求' : `第三方平台返回错误：${error}`)
    return
  }
  if (!code) {
    fail('未获取到授权码，请重新发起登录')
    return
  }
  if (!platform) {
    fail('登录会话已过期（未找到发起平台），请重新发起登录')
    return
  }
  if (context.state && state && state !== context.state) {
    fail('state 校验未通过，为保障账号安全已中止本次登录，请重新发起')
    return
  }

  handle(code, state, { ...context, platform })
})

onUnmounted(() => {
  if (codeTimer) clearInterval(codeTimer)
})
</script>

<style scoped>
.oauth-callback {
  text-align: center;
  padding: 6px 0 4px;
}
.oauth-loading {
  padding: 12px 0 8px;
}
.oauth-loading .spinner {
  margin: 0 auto 14px;
}
.oauth-icon {
  font-size: 34px;
  line-height: 1;
  margin-bottom: 12px;
  color: var(--text-light);
}
.oauth-icon.is-error {
  color: var(--danger);
}
.oauth-icon.is-info {
  color: var(--primary);
}
.oauth-title {
  font-family: var(--font-serif);
  font-size: 19px;
  font-weight: 600;
  margin-bottom: 8px;
}
.oauth-text {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
  margin: 0 auto 20px;
  max-width: 300px;
}
.oauth-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* ---------- 未绑定「选择页」 ---------- */
.oauth-peer {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 320px;
  margin: 0 auto 16px;
  text-align: left;
}
.oauth-peer-avatar {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  border: 1px solid var(--border);
}
.oauth-peer-info {
  min-width: 0;
}
.oauth-peer-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.oauth-peer-tip {
  font-size: 12px;
  color: var(--text-muted);
}
.oauth-choice {
  display: flex;
  gap: 10px;
  margin-bottom: 14px;
}
.oauth-choice-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm, 6px);
  background: var(--bg-card);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.oauth-choice-item i {
  font-size: 18px;
}
.oauth-choice-item:hover:not(:disabled) {
  border-color: var(--primary);
  color: var(--text);
}
.oauth-choice-item.is-active {
  border-color: var(--primary);
  color: var(--primary-deep, var(--primary));
  background: var(--primary-wash, var(--bg-muted));
}
.oauth-choice-item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.oauth-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}
.oauth-note {
  margin: 0 0 4px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
/* 验证码行：输入框占满剩余宽度，按钮保持单行 */
.oauth-code-row {
  display: flex;
  gap: 8px;
}
.oauth-code-row .input {
  flex: 1;
  min-width: 0;
}
.oauth-code-row .btn {
  flex-shrink: 0;
  white-space: nowrap;
}
</style>
