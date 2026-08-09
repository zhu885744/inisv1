<!-- 封禁申诉弹窗组件 -->
<template>
  <Teleport to="body">
    <transition name="modal-fade" mode="out-in">
      <div
        v-if="state.visible"
        class="modal fade show"
        style="display: block;"
        tabindex="-1"
        aria-modal="true"
        role="dialog"
      >
        <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable" style="max-width: 480px;">
          <div class="modal-content border-0 shadow-lg" style="border-radius: 12px;">
            <div class="modal-header border-0 pb-0">
              <h5 class="modal-title d-flex align-items-center text-warning">
                <i class="bi bi-journal-text me-2"></i>
                封禁申诉
              </h5>
              <button type="button" class="btn-close" @click="hide()" aria-label="Close"></button>
            </div>

            <div class="modal-body">
              <div class="alert alert-info d-flex align-items-start small mb-3" role="alert">
                <i class="bi bi-info-circle me-2 mt-1 flex-shrink-0"></i>
                <span>请输入被封禁的账号，系统将向您绑定的手机或邮箱发送验证码。验证身份后即可提交申诉。</span>
              </div>

              <!-- 账号 -->
              <div class="mb-3">
                <label class="form-label fw-semibold small">账号 <span class="text-danger">*</span></label>
                <div class="input-group input-group-sm">
                  <span class="input-group-text"><i class="bi bi-person"></i></span>
                  <input type="text" v-model="form.account" class="form-control" :class="{ 'is-invalid': state.fieldErrors.account }"
                         placeholder="邮箱 / 手机号 / 用户名" @input="state.fieldErrors.account = ''" />
                </div>
                <div class="invalid-feedback d-block small" v-if="state.fieldErrors.account">{{ state.fieldErrors.account }}</div>
              </div>

              <!-- 验证码 -->
              <div class="mb-3">
                <label class="form-label fw-semibold small">验证码 <span class="text-danger">*</span></label>
                <div class="input-group input-group-sm">
                  <span class="input-group-text"><i class="bi bi-shield-lock"></i></span>
                  <input type="text" v-model="form.code" class="form-control" :class="{ 'is-invalid': state.fieldErrors.code }"
                         placeholder="6位数字" maxlength="6" @input="state.fieldErrors.code = ''" />
                  <button type="button" class="btn btn-outline-warning d-flex align-items-center" style="white-space: nowrap;"
                          :disabled="state.sendingCode || state.countdown > 0" @click="sendCode">
                    <template v-if="state.sendingCode">
                      <span class="spinner-border spinner-border-sm me-1"></span>发送中
                    </template>
                    <template v-else-if="state.countdown > 0">{{ state.countdown }}s</template>
                    <template v-else>发送验证码</template>
                  </button>
                </div>
                <div class="invalid-feedback d-block small" v-if="state.fieldErrors.code">{{ state.fieldErrors.code }}</div>
                <div class="form-text small text-success" v-if="state.contactMasked">
                  <i class="bi bi-check-circle me-1"></i>已发送至：{{ state.contactMasked }}
                </div>
              </div>

              <!-- 申诉内容 -->
              <div class="mb-3">
                <label class="form-label fw-semibold small">申诉内容 <span class="text-danger">*</span></label>
                <textarea v-model="form.content" class="form-control" :class="{ 'is-invalid': state.fieldErrors.content }"
                          rows="4" placeholder="请详细说明您的申诉理由..."
                          @input="state.fieldErrors.content = ''"></textarea>
                <div class="invalid-feedback d-block small" v-if="state.fieldErrors.content">{{ state.fieldErrors.content }}</div>
              </div>

              <button class="btn btn-warning w-100" :disabled="state.submitting || state.success" @click="submitAppeal">
                <template v-if="state.submitting">
                  <span class="spinner-border spinner-border-sm me-2"></span>提交中...
                </template>
                <template v-else-if="state.success">
                  <i class="bi bi-check-circle me-2"></i>已提交
                </template>
                <template v-else>
                  <i class="bi bi-send me-2"></i>提交申诉
                </template>
              </button>

              <div v-if="state.success" class="alert alert-success d-flex align-items-center small mt-3 mb-0" role="alert">
                <i class="bi bi-check-circle-fill me-2"></i>
                申诉已提交！请耐心等待管理员审核。
              </div>

              <div v-if="state.errorMsg" class="alert alert-danger d-flex align-items-center small mt-3 mb-0" role="alert">
                <i class="bi bi-exclamation-circle me-2"></i>
                {{ state.errorMsg }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>

    <!-- 遮罩层 -->
    <transition name="modal-fade" mode="out-in">
      <div v-if="state.visible" class="modal-backdrop fade show" @click="hide()"></div>
    </transition>
  </Teleport>
</template>

<script setup>
import { reactive, onBeforeUnmount } from 'vue'
import { request } from '@/utils/network'
import { toast } from '@/utils/app'

const state = reactive({
  visible: false,
  sendingCode: false,
  countdown: 0,
  submitting: false,
  success: false,
  contactMasked: '',
  errorMsg: '',
  fieldErrors: { account: '', code: '', content: '' },
})
const form = reactive({ account: '', code: '', content: '' })

let countdownTimer = null

function resetForm() {
  form.account = ''
  form.code = ''
  form.content = ''
  state.contactMasked = ''
  state.success = false
  state.errorMsg = ''
  state.fieldErrors = { account: '', code: '', content: '' }
}

function show() {
  resetForm()
  state.visible = true
  document.body.style.overflow = 'hidden'
}

function hide() {
  state.visible = false
  document.body.style.overflow = ''
  if (countdownTimer) { clearInterval(countdownTimer); countdownTimer = null }
  state.countdown = 0
}

async function sendCode() {
  state.fieldErrors = { account: '', code: '', content: '' }
  state.errorMsg = ''

  if (!form.account.trim()) {
    state.fieldErrors.account = '请输入账号'
    return
  }

  state.sendingCode = true
  try {
    const res = await request.post('/api/users/appeal-public', { account: form.account.trim() })
    if (res.code === 201) {
      state.contactMasked = res.data?.contact_masked || ''
      form.code = ''
      toast.success(res.msg || '验证码发送成功！')
      state.countdown = 60
      if (countdownTimer) clearInterval(countdownTimer)
      countdownTimer = setInterval(() => {
        state.countdown--
        if (state.countdown <= 0) { clearInterval(countdownTimer); countdownTimer = null }
      }, 1000)
    } else {
      state.errorMsg = res.msg || '发送失败'
    }
  } catch (err) {
    state.errorMsg = '网络异常，请稍后重试'
  }
  state.sendingCode = false
}

async function submitAppeal() {
  state.fieldErrors = { account: '', code: '', content: '' }
  state.errorMsg = ''

  let valid = true
  if (!form.account.trim()) { state.fieldErrors.account = '请输入账号'; valid = false }
  if (!form.code.trim() || form.code.trim().length !== 6) { state.fieldErrors.code = '请输入6位验证码'; valid = false }
  if (!form.content.trim()) { state.fieldErrors.content = '请输入申诉内容'; valid = false }
  else if (form.content.trim().length < 10) { state.fieldErrors.content = '申诉内容至少10个字符'; valid = false }
  if (!valid) return

  if (state.submitting) return

  state.submitting = true
  try {
    const res = await request.post('/api/users/appeal-public', {
      account: form.account.trim(),
      code: form.code.trim(),
      content: form.content.trim(),
    })
    if (res.code === 200) {
      state.success = true
      toast.success('申诉已提交，请耐心等待管理员审核！')
    } else {
      state.errorMsg = res.msg || '提交失败'
    }
  } catch (err) {
    state.errorMsg = '网络异常，请稍后重试'
  }
  state.submitting = false
}

onBeforeUnmount(() => {
  document.body.style.overflow = ''
  if (countdownTimer) clearInterval(countdownTimer)
})

defineExpose({ show, hide })
</script>

<style scoped>
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
</style>
