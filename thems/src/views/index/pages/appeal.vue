<template>
    <div class="mt-2">
        <div class="row justify-content-center">
            <div class="col-md-8 col-lg-7 col-xl-6">
                <!-- ===== 页面头部 ===== -->
                <div class="text-center mb-4">
                    <div class="d-inline-flex align-items-center justify-content-center bg-warning bg-opacity-10 rounded-circle p-3 mb-2" style="width: 72px; height: 72px;">
                        <i class="bi bi-shield-exclamation" style="font-size: 2.4rem; color: #e6a23c;"></i>
                    </div>
                    <h4 class="fw-bold mb-1">封禁申诉</h4>
                    <p class="text-muted small mb-0">
                        如果您认为封禁有误，可以在此提交申诉。管理员审核后会将结果通过站内信通知您。
                    </p>
                </div>

                <!-- ===== 步骤指示器 ===== -->
                <div class="d-flex justify-content-between align-items-center mb-4 px-2">
                    <div class="d-flex flex-column align-items-center flex-fill">
                        <span class="badge bg-warning rounded-circle d-flex align-items-center justify-content-center" style="width: 32px; height: 32px; font-size: 0.85rem;">1</span>
                        <span class="small text-muted mt-1">查找记录</span>
                    </div>
                    <div class="flex-fill" style="height: 2px; background: #e9ecef; position: relative;">
                        <div class="bg-warning" style="height: 100%; width: 100%;"></div>
                    </div>
                    <div class="d-flex flex-column align-items-center flex-fill">
                        <span class="badge bg-warning rounded-circle d-flex align-items-center justify-content-center" style="width: 32px; height: 32px; font-size: 0.85rem;">2</span>
                        <span class="small text-muted mt-1">填写表单</span>
                    </div>
                    <div class="flex-fill" style="height: 2px; background: #e9ecef; position: relative;">
                        <div class="bg-warning" style="height: 100%; width: 0%;" :style="{ width: success ? '100%' : '0%' }"></div>
                    </div>
                    <div class="d-flex flex-column align-items-center flex-fill">
                        <span class="badge rounded-circle d-flex align-items-center justify-content-center" style="width: 32px; height: 32px; font-size: 0.85rem;"
                        :class="success ? 'bg-success' : 'bg-secondary'">
                        {{ success ? '✓' : '3' }}
                    </span>
                    <span class="small text-muted mt-1">提交完成</span>
                </div>
            </div>

            <!-- ===== 主卡片 ===== -->
            <div class="card shadow-sm border-0">
                <div class="card-header bg-transparent border-bottom d-flex align-items-center py-3">
                    <i class="bi bi-pencil-square text-warning me-2"></i>
                    <span class="fw-semibold">提交申诉</span>
                    <span class="ms-auto">
                        <span class="badge bg-light text-muted fw-normal">
                            <i class="bi bi-clock me-1"></i>预计 1-3 个工作日
                        </span>
                    </span>
                </div>

                <div class="card-body p-4">
                    <!-- 提示 -->
                    <div class="alert alert-info d-flex align-items-start mb-4" role="alert">
                        <i class="bi bi-info-circle me-2 mt-1 flex-shrink-0"></i>
                        <div class="small">
                            请先在 <router-link to="/blackroom" class="alert-link">小黑屋公示</router-link> 中找到您的封禁记录ID，
                            然后填写下方表单。如有疑问，可联系 <a href="mailto:admin@example.com" class="alert-link">管理员邮箱</a>。
                        </div>
                    </div>

                    <!-- ===== 表单 ===== -->
                    <form ref="formEl" novalidate @submit.prevent="submitAppeal">
                        <!-- 封禁记录ID -->
                        <div class="mb-3">
                            <label for="recordId" class="form-label fw-semibold">
                                封禁记录ID <span class="text-danger">*</span>
                            </label>
                            <div class="input-group">
                                <span class="input-group-text bg-light">
                                    <i class="bi bi-hash"></i>
                                </span>
                                <input
                                type="number"
                                id="recordId"
                                v-model="form.recordId"
                                class="form-control"
                                :class="{ 'is-invalid': submitted && errors.recordId }"
                                placeholder="请输入封禁记录ID"
                                min="1"
                                aria-describedby="recordIdFeedback"
                                @input="submitted && validateField('recordId')"
                                />
                            </div>
                            <div id="recordIdFeedback" class="invalid-feedback" v-if="submitted && errors.recordId">
                                {{ errors.recordId }}
                            </div>
                            <div class="form-text small">
                                <i class="bi bi-lightbulb me-1"></i>
                                封禁记录ID 可在 <router-link to="/blackroom" class="text-decoration-none">小黑屋公示</router-link> 中找到
                            </div>
                        </div>

                        <!-- 账号 -->
                        <div class="mb-3">
                            <label for="account" class="form-label fw-semibold">
                                您的账号 <span class="text-danger">*</span>
                            </label>
                            <div class="input-group">
                                <span class="input-group-text bg-light">
                                    <i class="bi bi-person"></i>
                                </span>
                                <input
                                type="text"
                                id="account"
                                v-model="form.account"
                                class="form-control"
                                :class="{ 'is-invalid': submitted && errors.account }"
                                placeholder="请输入您的登录账号"
                                aria-describedby="accountFeedback"
                                @input="submitted && validateField('account')"
                                />
                            </div>
                            <div id="accountFeedback" class="invalid-feedback" v-if="submitted && errors.account">
                                {{ errors.account }}
                            </div>
                            <div class="form-text small">
                                用于验证您与被封禁用户的关联关系
                            </div>
                        </div>

                        <!-- 申诉内容 -->
                        <div class="mb-3">
                            <label for="content" class="form-label fw-semibold">
                                申诉内容 <span class="text-danger">*</span>
                            </label>
                            <div class="input-group">
                                <span class="input-group-text bg-light align-items-start pt-3">
                                    <i class="bi bi-chat-dots"></i>
                                </span>
                                <textarea
                                id="content"
                                v-model="form.content"
                                class="form-control"
                                :class="{ 'is-invalid': submitted && errors.content }"
                                rows="5"
                                placeholder="请详细说明您的申诉理由，包括相关证据或说明..."
                                aria-describedby="contentFeedback"
                                @input="submitted && validateField('content')"
                                ></textarea>
                            </div>
                            <div id="contentFeedback" class="invalid-feedback" v-if="submitted && errors.content">
                                {{ errors.content }}
                            </div>
                            <div class="form-text small d-flex justify-content-between">
                                <span><i class="bi bi-info-circle me-1"></i>请提供尽可能详细的信息以便审核</span>
                                <span :class="form.content.length > 500 ? 'text-danger' : 'text-muted'">
                                    {{ form.content.length }} / 500
                                </span>
                            </div>
                        </div>

                        <!-- 提交按钮 -->
                        <button
                        type="submit"
                        class="btn btn-warning w-100 py-2 fw-semibold"
                        :disabled="submitting || success"
                        >
                        <template v-if="submitting">
                            <span class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                            提交中...
                        </template>
                        <template v-else-if="success">
                            <i class="bi bi-check-circle me-2"></i>已提交
                        </template>
                        <template v-else>
                            <i class="bi bi-send me-2"></i>提交申诉
                        </template>
                    </button>

                    <!-- 提交后的额外提示 -->
                    <div v-if="success" class="mt-3 text-center">
                        <div class="alert alert-success d-flex align-items-center" role="alert">
                            <i class="bi bi-check-circle-fill me-2"></i>
                            <div class="small">
                                申诉已成功提交！请耐心等待管理员审核，结果将通过站内信通知您。
                            </div>
                        </div>
                    </div>
                </form>
            </div>
        </div>

        <!-- ===== 底部信息 ===== -->
        <div class="d-flex justify-content-between align-items-center mt-3 small">
            <router-link to="/blackroom" class="text-muted text-decoration-none">
                <i class="bi bi-arrow-left me-1"></i>返回小黑屋公示
            </router-link>
            <div class="text-muted">
                <i class="bi bi-question-circle me-1"></i>
                <a href="#" class="text-muted text-decoration-none" @click.prevent="showHelp = !showHelp">
                    需要帮助？
                </a>
            </div>
        </div>

        <!-- 帮助折叠 -->
        <div v-if="showHelp" class="card mt-2 border-info bg-info bg-opacity-10">
            <div class="card-body small p-3">
                <h6 class="mb-2"><i class="bi bi-life-preserver me-1"></i> 常见问题</h6>
                <ul class="mb-0 ps-3">
                    <li>封禁记录ID 可在 <router-link to="/blackroom" class="text-decoration-none">小黑屋公示</router-link> 中查看</li>
                    <li>请使用被封禁的账号登录后提交申诉</li>
                    <li>申诉提交后不可修改，请确认信息无误</li>
                    <li>审核结果会通过站内信通知，请留意查收</li>
                </ul>
            </div>
        </div>
    </div>
</div>
</div>
</template>

<script setup>
import { ref, reactive, nextTick } from 'vue'
import { request } from '@/utils/network'
import { usePageTitle, toast } from '@/utils/app'

usePageTitle('封禁申诉')

// ===== 表单数据 =====
const form = reactive({
    account: '',
    code: '',
    content: '',
})

// ===== 错误状态 =====
const errors = reactive({
    account: '',
    code: '',
    content: '',
})

// ===== UI 状态 =====
const submitting = ref(false)
const success = ref(false)
const submitted = ref(false)
const showHelp = ref(false)
const formEl = ref(null)

// 验证码相关
const sendingCode = ref(false)
const countdown = ref(0)
const contactMasked = ref('')
const contactType = ref('')
let countdownTimer = null

// ===== 验证函数 =====
function validateField(field) {
    switch (field) {
        case 'account':
            if (!form.account.trim()) {
                errors.account = '请输入您的账号'
                return false
            }
            if (form.account.trim().length < 2) {
                errors.account = '账号至少2个字符'
                return false
            }
            errors.account = ''
            return true
        case 'code':
            if (!form.code.trim()) {
                errors.code = '请输入验证码'
                return false
            }
            if (form.code.trim().length !== 6) {
                errors.code = '验证码为6位数字'
                return false
            }
            errors.code = ''
            return true
        case 'content':
            if (!form.content.trim()) {
                errors.content = '请输入申诉内容'
                return false
            }
            if (form.content.trim().length < 10) {
                errors.content = '申诉内容至少10个字符，请详细说明'
                return false
            }
            if (form.content.trim().length > 500) {
                errors.content = '申诉内容不能超过500个字符'
                return false
            }
            errors.content = ''
            return true
        default:
            return true
    }
}

function validateAll() {
    const fields = ['account', 'code', 'content']
    let valid = true
    for (const field of fields) {
        if (!validateField(field)) {
            valid = false
        }
    }
    return valid
}

// ===== 提交申诉 =====
async function submitAppeal() {
    submitted.value = true

    // 滚动到第一个错误
    if (!validateAll()) {
        const firstInvalid = formEl.value?.querySelector('.is-invalid')
        if (firstInvalid) {
            firstInvalid.focus({ preventScroll: true })
            firstInvalid.scrollIntoView({ behavior: 'smooth', block: 'center' })
        }
        return
    }

    if (submitting.value || success.value) return

    submitting.value = true
    try {
        const res = await request.post('/api/users/appeal-public', {
            account: form.account.trim(),
            code: form.code.trim(),
            content: form.content.trim(),
        })

        if (res.code === 200) {
            success.value = true
            toast.success('申诉已提交，请耐心等待管理员审核！')
            // 滚动到顶部查看成功状态
            window.scrollTo({ top: 0, behavior: 'smooth' })
        } else {
            toast.error(res.msg || '提交失败，请稍后重试')
            // 如果服务器返回了具体字段错误，显示在对应字段上
            if (res.data?.errors) {
                const fieldErrors = res.data.errors
                if (fieldErrors.account) errors.account = fieldErrors.account
                if (fieldErrors.code) errors.code = fieldErrors.code
                if (fieldErrors.content) errors.content = fieldErrors.content
            }
        }
    } catch (err) {
        console.error('申诉提交失败:', err)
        toast.error('网络异常，请稍后重试')
    } finally {
        submitting.value = false
    }
}
</script>

<style scoped>
/* ===== 自定义样式 ===== */
.card {
    border-radius: 12px;
    transition: box-shadow 0.2s ease;
}

.card:hover {
    box-shadow: 0 0.5rem 1rem rgba(0, 0, 0, 0.08) !important;
}

.card-header {
    border-radius: 12px 12px 0 0 !important;
}

/* 表单输入框焦点样式 */
.form-control:focus,
.input-group-text:focus-within {
    border-color: #e6a23c;
    box-shadow: 0 0 0 0.25rem rgba(230, 162, 60, 0.25);
}

/* 自定义按钮悬浮效果 */
.btn-warning {
    background-color: #e6a23c;
    border-color: #d4922d;
    color: #fff;
    transition: all 0.2s ease;
}

.btn-warning:hover:not(:disabled) {
    background-color: #d4922d;
    border-color: #c4822a;
    color: #fff;
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(230, 162, 60, 0.35);
}

.btn-warning:disabled {
    opacity: 0.7;
    cursor: not-allowed;
}

/* 步骤指示器 */
.badge.rounded-circle {
    font-weight: 600;
    transition: all 0.3s ease;
}

/* 表单验证过渡 */
.is-invalid {
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.invalid-feedback {
    display: block;
    animation: fadeSlideDown 0.25s ease;
}

@keyframes fadeSlideDown {
    from {
        opacity: 0;
        transform: translateY(-6px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

/* 成功动画 */
.alert-success {
    animation: fadeSlideDown 0.4s ease;
}

/* 帮助折叠过渡 */
.card.border-info {
    animation: fadeSlideDown 0.3s ease;
}

/* 响应式微调 */
@media (max-width: 576px) {
    .card-body {
        padding: 1.25rem !important;
    }

    .step-label {
        font-size: 0.7rem;
    }

    .badge.rounded-circle {
        width: 28px !important;
        height: 28px !important;
        font-size: 0.75rem !important;
    }
}

/* 输入框数字箭头隐藏 (火狐) */
input[type=number] {
    -moz-appearance: textfield;
}
input[type=number]::-webkit-outer-spin-button,
input[type=number]::-webkit-inner-spin-button {
    -webkit-appearance: none;
    margin: 0;
}

/* 文本域自动调整高度（可选） */
textarea.form-control {
    resize: vertical;
    min-height: 120px;
}
</style>