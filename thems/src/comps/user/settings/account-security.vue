<template>
  <div class="account-security-settings">
    <!-- 骨架加载 -->
    <div v-if="loading" class="space-y-4">
      <!-- 账号设置骨架 -->
      <div class="card wx-card">
        <div class="card-body">
          <div class="skeleton skeleton-title"></div>
          <div class="space-y-4 mt-4">
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-btn"></div>
          </div>
        </div>
      </div>
      <!-- 重置密码骨架 -->
      <div class="card wx-card">
        <div class="card-body">
          <div class="skeleton skeleton-title"></div>
          <div class="space-y-4 mt-4">
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-label"></div>
            <div class="skeleton skeleton-input"></div>
            <div class="skeleton skeleton-btn"></div>
          </div>
        </div>
      </div>
      <!-- 安全提示骨架 -->
      <div class="card wx-card">
        <div class="card-body">
          <div class="skeleton skeleton-title"></div>
          <div class="space-y-3 mt-4">
            <div class="skeleton skeleton-row"></div>
            <div class="skeleton skeleton-row"></div>
            <div class="skeleton skeleton-row"></div>
            <div class="skeleton skeleton-row"></div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="row">
      <!-- 修改账号 -->
      <div class="col-md-12">
        <div class="card wx-card mb-4">
          <div class="card-body">
            <h6 class="card-title mb-3">账号设置</h6>
            <form @submit.prevent="handleAccountSubmit">
              <div class="mb-3">
                <label class="form-label">当前账号</label>
                <div class="input-group">
                  <input
                    type="text"
                    class="form-control"
                    :value="currentAccount"
                    disabled
                  >
                  <button
                    type="button"
                    class="btn wx-btn-outline"
                    @click="toggleAccountEdit"
                    :disabled="accountLoading"
                  >
                    <i class="bi bi-pencil me-1"></i>{{ showAccountEdit ? '取消' : '修改' }}
                  </button>
                </div>
              </div>

              <div v-if="showAccountEdit" class="mt-3 p-3 border rounded-1 bg-body-secondary">
                <h6 class="mb-3 text-danger">
                  <i class="bi bi-exclamation-triangle me-2"></i>修改账号后将影响登录，请谨慎操作
                </h6>
                <div class="mb-3 mt-1">
                  <label for="newAccount" class="form-label">新账号</label>
                  <input
                    type="text"
                    id="newAccount"
                    v-model="accountForm.newAccount"
                    class="form-control"
                    placeholder="请输入新账号（字母、数字、下划线，4-20位）"
                    maxlength="20"
                    @input="validateNewAccount"
                  >
                  <div v-if="accountErrors.newAccount" class="text-danger small mt-1">{{ accountErrors.newAccount }}</div>
                </div>
                <div class="d-flex gap-2">
                  <button
                    type="submit"
                    class="btn btn-danger"
                    :disabled="accountLoading"
                  >
                    <span v-if="accountLoading" class="spinner-border spinner-border-sm me-2"></span>
                    <i v-else class="bi bi-check-circle me-2"></i>
                    {{ accountLoading ? '修改中...' : '确认修改账号' }}
                  </button>
                  <button
                    type="button"
                    class="btn wx-btn-outline"
                    @click="cancelAccountEdit"
                    :disabled="accountLoading"
                  >
                    取消
                  </button>
                </div>
              </div>
            </form>
          </div>
        </div>
      </div>

      <!-- 重置密码 -->
      <div class="col-md-12">
        <div class="card wx-card mb-4">
          <div class="card-body">
            <h6 class="card-title mb-3">重置密码</h6>
            <form @submit.prevent="handleResetPasswordSubmit">
              <div class="mb-3">
                <label for="contactInput" class="form-label">邮箱/手机号</label>
                <input
                  type="text"
                  id="contactInput"
                  v-model="resetForm.contact"
                  class="form-control"
                  placeholder="请输入您的邮箱或手机号"
                  @input="validateContact"
                >
                <div v-if="errors.contact" class="text-danger small mt-1">{{ errors.contact }}</div>
              </div>

              <div class="mb-3">
                <label for="codeInput" class="form-label">验证码</label>
                <div class="input-group">
                  <input
                    type="text"
                    id="codeInput"
                    v-model="resetForm.code"
                    class="form-control"
                    placeholder="请输入验证码"
                    @input="validateCode"
                  >
                  <button
                    type="button"
                    class="btn wx-btn-outline"
                    :disabled="countdown > 0 || sendCodeLoading"
                    @click="sendCode"
                  >
                    {{ countdown > 0 ? `${countdown}秒后重试` : '获取验证码' }}
                  </button>
                </div>
                <div v-if="errors.code" class="text-danger small mt-1">{{ errors.code }}</div>
              </div>

              <div class="mb-3">
                <label for="newPassword" class="form-label">新密码</label>
                <input
                  type="password"
                  id="newPassword"
                  v-model="resetForm.password"
                  class="form-control"
                  placeholder="请输入新密码"
                  minlength="6"
                  maxlength="20"
                  @input="validatePassword"
                >
                <div v-if="errors.password" class="text-danger small mt-1">{{ errors.password }}</div>
              </div>

              <div class="mb-3">
                <label for="confirmPassword" class="form-label">确认新密码</label>
                <input
                  type="password"
                  id="confirmPassword"
                  v-model="resetForm.verifyPwd"
                  class="form-control"
                  placeholder="请再次输入新密码"
                  @input="validateVerifyPwd"
                >
                <div v-if="errors.verifyPwd" class="text-danger small mt-1">{{ errors.verifyPwd }}</div>
              </div>

              <div v-if="resetForm.password" class="mb-3">
                <div class="password-strength">
                  <div class="d-flex justify-content-between mb-1">
                    <span class="text-muted small">密码强度</span>
                    <span :class="passwordStrengthClass" class="small fw-medium">
                      {{ passwordStrengthText }}
                    </span>
                  </div>
                  <div class="progress" style="height: 6px;">
                    <div
                      class="progress-bar"
                      :class="passwordStrengthClass"
                      :style="{ width: passwordStrengthWidth }"
                    ></div>
                  </div>
                </div>
              </div>

              <button
                type="submit"
                class="btn wx-btn-gradient"
                :disabled="resetLoading"
              >
                <span v-if="resetLoading" class="spinner-border spinner-border-sm me-2"></span>
                {{ resetLoading ? '重置中...' : '重置密码' }}
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>

    <!-- 安全提示 -->
    <div v-if="!loading" class="card wx-card">
      <div class="card-body">
        <h6 class="card-title mb-3">安全提示</h6>
        <ul class="list-unstyled d-flex flex-column gap-2">
          <li class="wx-list-item">
            <i class="bi bi-shield-check text-secondary me-2"></i>
            建议使用字母、数字和特殊字符组合的密码
          </li>
          <li class="wx-list-item">
            <i class="bi bi-clock-history text-secondary me-2"></i>
            定期更换密码，建议每3个月更换一次
          </li>
          <li class="wx-list-item">
            <i class="bi bi-exclamation-triangle text-secondary me-2"></i>
            不要在多个网站使用相同的密码
          </li>
          <li class="wx-list-item">
            <i class="bi bi-lock text-secondary me-2"></i>
            不要将密码告诉他人，包括网站客服
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { request } from '@/utils/network'
import { toast } from '@/utils/app'
import { useCommStore } from '@/store/comm'

const store = useCommStore()

// ====================== 常量统一管理 ======================
const CONST = {
  ACCOUNT_MIN: 4,
  ACCOUNT_MAX: 20,
  PWD_MIN: 6,
  PWD_MAX: 20,
  CODE_COOLDOWN: 60,
  DAILY_MAX_SEND: 10,
  DAY_MS: 24 * 60 * 60 * 1000,
  REG: {
    account: /^[a-zA-Z0-9_]+$/,
    phone: /^1[3-9]\d{9}$/,
    email: /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/
  },
  API: {
    updateUser: '/api/users/update',
    sendResetCode: '/api/comm/reset-password',
    resetPassword: '/api/comm/reset-password'
  },
  CACHE_KEY_PREFIX: 'verify-code-'
}

// ====================== 状态 ======================
const loading = ref(true)
const accountLoading = ref(false)
const resetLoading = ref(false)
const sendCodeLoading = ref(false)

const showAccountEdit = ref(false)
const currentAccount = ref('')
const countdown = ref(0)
let countdownTimer = null

// 修改账号表单
const accountForm = reactive({ newAccount: '' })
const accountErrors = reactive({ newAccount: '' })

// 重置密码表单（重命名字段，语义清晰）
const resetForm = reactive({
  contact: '',
  code: '',
  password: '',
  verifyPwd: ''
})
const errors = reactive({
  contact: '',
  code: '',
  password: '',
  verifyPwd: ''
})
const valid = reactive({
  contact: false,
  code: false,
  password: false,
  verifyPwd: false
})

// ====================== 本地存储工具（修复：使用时间戳判断过期，不再依赖setTimeout） ======================
function getCacheKey(type, contact) {
  return `${CONST.CACHE_KEY_PREFIX}${type}-${contact}`
}
function getDailyKey(type, contact) {
  return `${CONST.CACHE_KEY_PREFIX}daily-${type}-${contact}`
}

function getStorageItem(key) {
  const raw = localStorage.getItem(key)
  if (!raw) return null
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}
function setStorageItem(key, data) {
  localStorage.setItem(key, JSON.stringify(data))
}

/** 判断是否可以发送验证码 */
function canSendCode(type, contact) {
  const cacheKey = getCacheKey(type, contact)
  const last = getStorageItem(cacheKey)
  if (last && Date.now() - last < CONST.CODE_COOLDOWN * 1000) {
    return false
  }

  const dailyKey = getDailyKey(type, contact)
  const dailyData = getStorageItem(dailyKey)
  const now = Date.now()
  // 判断是否过期
  if (dailyData && now < dailyData.expire) {
    if (dailyData.count >= CONST.DAILY_MAX_SEND) return false
  }
  return true
}

/** 记录发送验证码 */
function recordSendCode(type, contact) {
  const cacheKey = getCacheKey(type, contact)
  setStorageItem(cacheKey, Date.now())

  const dailyKey = getDailyKey(type, contact)
  const dailyData = getStorageItem(dailyKey)
  const expire = Date.now() + CONST.DAY_MS
  let count = 1
  if (dailyData && Date.now() < dailyData.expire) {
    count = dailyData.count + 1
  }
  setStorageItem(dailyKey, { count, expire })
}

function startCountdown() {
  countdown.value = CONST.CODE_COOLDOWN
  if (countdownTimer) clearInterval(countdownTimer)
  countdownTimer = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--
    } else {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }, 1000)
}

// ====================== 校验函数 ======================
function validateNewAccount() {
  const val = accountForm.newAccount.trim()
  if (!val) {
    accountErrors.newAccount = '请输入新账号'
    return false
  }
  if (val.length < CONST.ACCOUNT_MIN) {
    accountErrors.newAccount = `账号长度不能少于${CONST.ACCOUNT_MIN}位`
    return false
  }
  if (val.length > CONST.ACCOUNT_MAX) {
    accountErrors.newAccount = `账号长度不能超过${CONST.ACCOUNT_MAX}位`
    return false
  }
  if (!CONST.REG.account.test(val)) {
    accountErrors.newAccount = '账号只能包含字母、数字和下划线'
    return false
  }
  if (val === currentAccount.value) {
    accountErrors.newAccount = '新账号不能与当前账号相同'
    return false
  }
  accountErrors.newAccount = ''
  return true
}

function validateContact() {
  const val = resetForm.contact.trim()
  if (!val) {
    errors.contact = '请输入邮箱或手机号'
    valid.contact = false
    return
  }
  const isPhone = CONST.REG.phone.test(val)
  const isEmail = CONST.REG.email.test(val)
  if (!isPhone && !isEmail) {
    errors.contact = '请输入正确的邮箱或手机号'
    valid.contact = false
    return
  }
  errors.contact = ''
  valid.contact = true
}

function validateCode() {
  const val = resetForm.code.trim()
  if (!val) {
    errors.code = '请输入验证码'
    valid.code = false
    return
  }
  if (val.length < 4) {
    errors.code = '验证码长度不足'
    valid.code = false
    return
  }
  errors.code = ''
  valid.code = true
}

function validatePassword() {
  const val = resetForm.password
  if (!val) {
    errors.password = '请输入新密码'
    valid.password = false
    return
  }
  if (val.length < CONST.PWD_MIN) {
    errors.password = `密码长度不能少于${CONST.PWD_MIN}位`
    valid.password = false
    return
  }
  errors.password = ''
  valid.password = true
  validateVerifyPwd()
}

function validateVerifyPwd() {
  const val = resetForm.verifyPwd
  if (!val) {
    errors.verifyPwd = '请确认新密码'
    valid.verifyPwd = false
    return
  }
  if (val !== resetForm.password) {
    errors.verifyPwd = '两次输入的密码不一致'
    valid.verifyPwd = false
    return
  }
  errors.verifyPwd = ''
  valid.verifyPwd = true
}

// 密码强度
function calculatePasswordStrength(pwd) {
  if (!pwd) return 0
  let strength = 0
  if (pwd.length >= 8) strength += 25
  else if (pwd.length >= 6) strength += 15
  if (/\d/.test(pwd)) strength += 25
  if (/[a-z]/.test(pwd)) strength += 25
  if (/[A-Z]/.test(pwd) || /[^a-zA-Z0-9]/.test(pwd)) strength += 25
  return Math.min(strength, 100)
}

const passwordStrengthValue = computed(() => calculatePasswordStrength(resetForm.password))
const passwordStrengthWidth = computed(() => `${passwordStrengthValue.value}%`)
const passwordStrengthText = computed(() => {
  const s = passwordStrengthValue.value
  if (s < 25) return '弱'
  if (s < 50) return '一般'
  if (s < 75) return '良好'
  return '强'
})
const passwordStrengthClass = computed(() => {
  const s = passwordStrengthValue.value
  if (s < 25) return 'bg-danger'
  if (s < 50) return 'bg-warning'
  if (s < 75) return 'bg-info'
  return 'bg-success'
})

// ====================== 账号修改逻辑 ======================
const toggleAccountEdit = () => {
  showAccountEdit.value = !showAccountEdit.value
  if (!showAccountEdit.value) resetAccountForm()
}
const cancelAccountEdit = () => {
  showAccountEdit.value = false
  resetAccountForm()
}
function resetAccountForm() {
  accountForm.newAccount = ''
  accountErrors.newAccount = ''
}

async function handleAccountSubmit() {
  if (!validateNewAccount()) return
  accountLoading.value = true
  try {
    const userInfo = store.getLogin.user
    if (!userInfo?.id) {
      toast.error('用户信息获取失败，请刷新页面重试')
      return
    }
    const { code: resCode, msg } = await request.put(CONST.API.updateUser, {
      id: userInfo.id,
      account: accountForm.newAccount.trim()
    })
    if (resCode === 200) {
      toast.success('账号修改成功')
      await syncUserInfo()
      showAccountEdit.value = false
      resetAccountForm()
      const newUser = store.getLogin.user
      currentAccount.value = newUser?.account || ''
    } else {
      toast.error(msg || '账号修改失败')
    }
  } catch (err) {
    console.error('修改账号失败:', err)
    toast.error('网络异常，请稍后再试！')
  } finally {
    accountLoading.value = false
  }
}

async function syncUserInfo() {
  try {
    await store.checkLoginState()
  } catch (err) {
    console.error('同步用户信息失败', err)
  }
}

// ====================== 重置密码逻辑 ======================
async function sendCode() {
  validateContact()
  if (!valid.contact.value) return
  const contact = resetForm.contact.trim()
  if (!canSendCode('reset', contact)) {
    toast.error('发送过于频繁或已达到每日上限，请稍后再试')
    return
  }
  sendCodeLoading.value = true
  try {
    const { code: resCode, msg } = await request.post(CONST.API.sendResetCode, { contact })
    if (resCode === 200 || resCode === 201) {
      toast.success(msg || '验证码发送成功！')
      recordSendCode('reset', contact)
      startCountdown()
    } else {
      toast.error(msg || '发送验证码失败！')
    }
  } catch (err) {
    toast.error('网络异常，验证码发送失败！')
  } finally {
    sendCodeLoading.value = false
  }
}

function handleResetPasswordSubmit() {
  validateContact()
  validateCode()
  validatePassword()
  validateVerifyPwd()
  if (!valid.contact.value || !valid.code.value || !valid.password.value || !valid.verifyPwd.value) {
    toast.warning('请检查表单填写是否正确')
    return
  }
  doResetPassword()
}

async function doResetPassword() {
  resetLoading.value = true
  try {
    const { code: resCode, msg } = await request.post(CONST.API.resetPassword, {
      social: resetForm.contact,
      code: resetForm.code,
      password: resetForm.password
    })
    if (resCode !== 200) throw new Error(msg || '重置密码失败！')
    toast.success('密码修改成功，请重新登录')
    // 清空表单
    resetForm.contact = ''
    resetForm.code = ''
    resetForm.password = ''
    resetForm.verifyPwd = ''
  } catch (err) {
    toast.error(err.message || '网络异常，请稍后再试！')
  } finally {
    resetLoading.value = false
  }
}

// 初始化加载用户信息
function fetchUserInfo() {
  const loginState = store.getLogin
  if (loginState.user) {
    currentAccount.value = loginState.user.account || ''
  }
  loading.value = false
}

onMounted(() => {
  fetchUserInfo()
})

onUnmounted(() => {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
})
</script>

<style scoped>
.account-security-settings {
  width: 100%;
}

/* 表单控件：对齐 wx-form-control 规范 */
.form-control {
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
  padding: 0.625rem 0.875rem;
  font-size: 0.9rem;
  background: var(--bs-body-bg);
  color: var(--bs-body-color);
  border: 1px solid var(--bs-border-color);
}

.form-control:focus {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.15);
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 0.375rem;
  color: var(--bs-body-color);
}

/* 骨架统一class */
.skeleton {
  background: linear-gradient(90deg, var(--bs-tertiary-bg) 25%, var(--bs-border-color) 50%, var(--bs-tertiary-bg) 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
  border-radius: var(--wx-radius-sm);
}
.skeleton-title { height: 20px; width: 60%; }
.skeleton-label { height: 16px; width: 30%; }
.skeleton-input { height: 40px; width: 100%; }
.skeleton-btn { height: 40px; width: 30%; }
.skeleton-row { height: 40px; width: 100%; }

@keyframes skeleton-loading {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* 移动端适配 */
@media (max-width: 768px) {
  .card-body {
    padding: 1rem;
  }
  .mb-4 {
    margin-bottom: 1rem !important;
  }
  .mb-3 {
    margin-bottom: 0.75rem !important;
  }
  .card-title {
    font-size: 0.9375rem;
    margin-bottom: 0.75rem !important;
  }
  .form-control {
    padding: 0.5rem 0.625rem;
    font-size: 0.8125rem;
  }
}
</style>