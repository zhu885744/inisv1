<!-- 认证组件（整合登录、注册、找回密码） -->
<template>
    <transition name="modal-fade" mode="out-in">
        <div 
            v-if="state.item.dialog"
            class="modal fade show"
            style="display: block;"
            tabindex="-1" 
            :aria-labelledby="`${state.item.type}ModalLabel`" 
            aria-hidden="false"
            aria-modal="true"
            :id="`${state.item.type}Modal`"
        >
            <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable auth-modal-dialog" style="max-width: 440px; margin: 1.75rem auto;">
                <div class="modal-content auth-modal-content">
                    <div class="modal-header auth-modal-header">
                        <h5 class="modal-title d-flex align-items-center" :id="`${state.item.type}ModalLabel`">
                            <i class="bi me-2" :class="getModalIcon"></i>
                            {{ getModalTitle }}
                        </h5>
                        <button type="button" class="btn-close" @click="method.hide()" aria-label="Close"></button>
                    </div>
                    
                    <div class="modal-body">
                        <!-- 表单切换过渡效果 -->
                        <transition name="form-switch" mode="out-in">
                            <!-- 账号密码登录表单 -->
                            <form key="password-login" v-if="state.item.type === 'login'" @submit.prevent="method.login()">
                                <div class="mb-3">
                                    <label for="loginAccountInput" class="form-label auth-form-label">帐号</label>
                                    <div class="input-group auth-input-group">
                                        <span class="input-group-text auth-input-icon"><i class="bi bi-person"></i></span>
                                        <input type="text"
                                               class="form-control"
                                               id="loginAccountInput"
                                               v-model="state.struct.account"
                                               placeholder="帐号 | 邮箱 | 手机号"
                                               required
                                               autocomplete="username"
                                           :class="{ 'is-invalid': state.errors.account, 'is-valid': state.valid.account }">
                                    </div>
                                    <div v-if="state.errors.account" class="invalid-feedback">
                                        {{ state.errors.account }}
                                    </div>
                                    <div v-if="state.valid.account" class="valid-feedback">
                                        格式正确
                                    </div>
                                </div>

                                <div class="mb-3">
                                    <label for="loginPasswordInput" class="form-label auth-form-label">密码</label>
                                    <div class="input-group auth-input-group">
                                        <span class="input-group-text auth-input-icon"><i class="bi bi-lock"></i></span>
                                        <input :type="showPassword ? 'text' : 'password'"
                                               class="form-control"
                                               id="loginPasswordInput"
                                               v-model="state.struct.password"
                                               placeholder="请输入密码"
                                               required
                                               autocomplete="current-password"
                                               :class="{ 'is-invalid': state.errors.password, 'is-valid': state.valid.password }">
                                        <button class="btn auth-password-toggle"
                                                type="button"
                                                @click="showPassword = !showPassword">
                                            <i class="bi" :class="showPassword ? 'bi-eye-slash' : 'bi-eye'"></i>
                                        </button>
                                    </div>
                                    <div v-if="state.errors.password" class="invalid-feedback">
                                        {{ state.errors.password }}
                                    </div>
                                    <div v-if="state.valid.password" class="valid-feedback">
                                        密码长度符合要求
                                    </div>
                                </div>

                                <div class="mb-3">
                                    <button type="submit"
                                            class="btn auth-submit-btn w-100"
                                            :disabled="state.item.wait || !isLoginFormValid">
                                        <span v-if="state.item.wait" class="spinner-border spinner-border-sm me-2" role="status"></span>
                                        <i v-else class="bi bi-box-arrow-in-right me-1"></i>
                                        {{ state.item.wait ? '登录中...' : '登录' }}
                                    </button>
                                </div>

                                <div v-if="authAgreementConfig.enabled" class="text-center mb-3">
                                    <p class="text-muted auth-agreement-text">
                                        登录即代表您同意
                                        <a href="javascript:void(0)" @click="method.showAgreement('user')" class="auth-agreement-link">《用户协议》</a>
                                        和
                                        <a href="javascript:void(0)" @click="method.showAgreement('privacy')" class="auth-agreement-link">《隐私协议》</a>
                                    </p>
                                </div>

                                <div class="row mt-3 auth-switch-row">
                                    <div class="col">
                                        <button type="button"
                                                class="btn auth-switch-btn w-100"
                                                @click="method.switchType('reset')">
                                            <i class="bi bi-key me-1"></i>忘记密码
                                        </button>
                                    </div>
                                    <div class="col">
                                        <button type="button"
                                                class="btn auth-switch-btn w-100"
                                                v-if="parseInt(store.config.getAllowRegister?.value) === 1"
                                                @click="method.switchType('register')">
                                            <i class="bi bi-person-plus me-1"></i>注册帐号
                                        </button>
                                    </div>
                                </div>
                            </form>

                            <!-- 注册表单 -->
                            <form key="register" v-else-if="state.item.type === 'register'" @submit.prevent="method.register()" novalidate>
                                <div class="mb-3">
                                    <label for="nicknameInput" class="form-label">
                                        昵称 <span class="text-danger">*</span>
                                    </label>
                                    <input type="text" 
                                           class="form-control" 
                                           id="nicknameInput"
                                           v-model="state.struct.nickname"
                                           placeholder="希望别人怎么称呼您？"
                                           required
                                           :class="{ 'is-invalid': state.errors.nickname, 'is-valid': state.valid.nickname }">
                                    <div v-if="state.errors.nickname" class="invalid-feedback">
                                        {{ state.errors.nickname }}
                                    </div>
                                    <div v-if="state.valid.nickname" class="valid-feedback">
                                        昵称格式正确
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="accountInput" class="form-label">
                                        账号 <span class="text-danger">*</span>
                                    </label>
                                    <input type="text" 
                                           class="form-control" 
                                           id="registerAccountInput"
                                           v-model="state.struct.account"
                                           placeholder="定制您的专属账号"
                                           required
                                           :class="{ 'is-invalid': state.errors.account, 'is-valid': state.valid.account }">
                                    <div v-if="state.errors.account" class="invalid-feedback">
                                        {{ state.errors.account }}
                                    </div>
                                    <div v-if="state.valid.account" class="valid-feedback">
                                        账号格式正确
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="socialInput" class="form-label">
                                        邮箱或手机号 <span class="text-danger">*</span>
                                    </label>
                                    <input :type="getSocialInputType" 
                                           class="form-control" 
                                           id="socialInput"
                                           v-model="state.struct.social"
                                           placeholder="电子邮箱或手机号"
                                           required
                                           :class="{ 'is-invalid': state.errors.social, 'is-valid': state.valid.social }">
                                    <div v-if="state.errors.social" class="invalid-feedback">
                                        {{ state.errors.social }}
                                    </div>
                                    <div v-if="state.valid.social" class="valid-feedback">
                                        格式正确
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="codeInput" class="form-label">
                                        验证码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input type="text" 
                                               class="form-control" 
                                               id="codeInput"
                                               v-model="state.struct.code"
                                               placeholder="请输入验证码"
                                               required
                                               autocomplete="username"
                                               :class="{ 'is-invalid': state.errors.code, 'is-valid': state.valid.code }">
                                        <button type="button" 
                                                class="btn btn-outline-secondary" 
                                                @click="method.code('register')"
                                                :disabled="state.item.loading || !state.valid.social">
                                            <span v-if="state.item.loading" class="spinner-border spinner-border-sm me-1"></span>
                                            {{ state.item.loading ? `${state.item.second}秒后重试` : '获取验证码' }}
                                        </button>
                                    </div>
                                    <div v-if="state.errors.code" class="invalid-feedback">
                                        {{ state.errors.code }}
                                    </div>
                                    <div v-if="state.valid.code" class="valid-feedback">
                                        验证码格式正确
                                    </div>
                                    <div class="form-text text-muted mt-1">
                                        验证码有效期为5分钟，请及时输入
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="passwordInput" class="form-label">
                                        密码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input 
                                            :type="showPassword ? 'text' : 'password'" 
                                            class="form-control" 
                                            id="registerPasswordInput"
                                            v-model="state.password.value"
                                            placeholder="请输入密码"
                                            required
                                            autocomplete="new-password"
                                            :class="{ 'is-invalid': state.errors.password, 'is-valid': state.valid.password }">
                                        <button class="btn btn-outline-secondary" type="button" @click="showPassword = !showPassword">
                                            <i class="bi" :class="showPassword ? 'bi-eye-slash' : 'bi-eye'"></i>
                                        </button>
                                    </div>
                                    <div v-if="state.errors.password" class="invalid-feedback">
                                        {{ state.errors.password }}
                                    </div>
                                    <div v-if="state.valid.password" class="valid-feedback">
                                        密码长度符合要求
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="confirmPasswordInput" class="form-label">
                                        确认密码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input :type="showConfirmPassword ? 'text' : 'password'" 
                                               class="form-control" 
                                               id="confirmPasswordInput"
                                               v-model="state.password.verify"
                                               placeholder="请再次输入密码"
                                               required
                                               autocomplete="new-password"
                                               :class="{ 'is-invalid': state.errors.verify, 'is-valid': state.valid.verify }">
                                        <button class="btn btn-outline-secondary" type="button" @click="showConfirmPassword = !showConfirmPassword">
                                            <i class="bi" :class="showConfirmPassword ? 'bi-eye-slash' : 'bi-eye'"></i>
                                        </button>
                                    </div>
                                    <div v-if="state.errors.verify" class="invalid-feedback">
                                        {{ state.errors.verify }}
                                    </div>
                                    <div v-if="state.valid.verify" class="valid-feedback">
                                        两次输入的密码一致
                                    </div>
                                </div>
                                
                                <div v-if="authAgreementConfig.enabled" class="text-center mb-3">
                                    <p class="text-muted auth-agreement-text">
                                        注册即代表您同意
                                        <a href="javascript:void(0)" @click="method.showAgreement('user')" class="auth-agreement-link">《用户协议》</a>
                                        和
                                        <a href="javascript:void(0)" @click="method.showAgreement('privacy')" class="auth-agreement-link">《隐私协议》</a>
                                    </p>
                                </div>

                                <div class="row mt-3 auth-switch-row">
                                    <div class="col">
                                        <button type="button"
                                                class="btn auth-switch-btn w-100"
                                                @click="method.switchType('login')">
                                            <i class="bi bi-box-arrow-in-right me-1"></i>已有账号？登录
                                        </button>
                                    </div>
                                    <div class="col">
                                        <button type="submit"
                                                class="btn auth-switch-btn w-100"
                                                :disabled="state.item.wait || !isRegisterFormValid">
                                            <span v-if="state.item.wait" class="spinner-border spinner-border-sm me-2"></span>
                                            <i v-else class="bi bi-person-plus me-1"></i>
                                            {{ state.item.wait ? '注册中...' : '注册' }}
                                        </button>
                                    </div>
                                </div>
                            </form>

                            <!-- 找回密码表单 -->
                            <form key="reset" v-else-if="state.item.type === 'reset'" @submit.prevent="method.reset()" novalidate>
                                <div class="mb-3">
                                    <label for="contactInput" class="form-label">邮箱或手机号</label>
                                    <input :type="getSocialInputType" 
                                           class="form-control" 
                                           id="contactInput"
                                           v-model="state.struct.social"
                                           placeholder="用于接收验证码"
                                           :class="{ 'is-invalid': state.errors.social, 'is-valid': state.valid.social }">
                                    <div v-if="state.errors.social" class="invalid-feedback">
                                        {{ state.errors.social }}
                                    </div>
                                    <div v-if="state.valid.social" class="valid-feedback">
                                        格式正确
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="verificationCode" class="form-label">
                                        验证码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input type="text" 
                                               class="form-control" 
                                               id="verificationCode"
                                               v-model="state.struct.code"
                                               placeholder="请输入验证码"
                                               required
                                               autocomplete="username"
                                               :class="{ 'is-invalid': state.errors.code, 'is-valid': state.valid.code }">
                                        <button type="button" 
                                                class="btn btn-outline-secondary" 
                                                @click="method.code('reset')"
                                                :disabled="state.item.loading || !state.valid.social">
                                            <span v-if="state.item.loading" class="spinner-border spinner-border-sm me-1"></span>
                                            {{ state.item.loading ? `${state.item.second}秒后重试` : '获取验证码' }}
                                        </button>
                                    </div>
                                    <div v-if="state.errors.code" class="invalid-feedback">
                                        {{ state.errors.code }}
                                    </div>
                                    <div v-if="state.valid.code" class="valid-feedback">
                                        验证码格式正确
                                    </div>
                                    <div class="form-text text-muted mt-1">
                                        验证码有效期为5分钟，请及时输入
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="newPassword" class="form-label">
                                        新的密码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input 
                                            :type="showPassword ? 'text' : 'password'" 
                                            class="form-control" 
                                            id="resetNewPassword"
                                            v-model="state.password.value"
                                            placeholder="请输入新密码（至少6位）"
                                            minlength="6"
                                            required
                                            autocomplete="new-password"
                                            :class="{ 'is-invalid': state.errors.password, 'is-valid': state.valid.password }">
                                        <button class="btn btn-outline-secondary" type="button" @click="showPassword = !showPassword">
                                            <i class="bi" :class="showPassword ? 'bi-eye-slash' : 'bi-eye'"></i>
                                        </button>
                                    </div>
                                    <div v-if="state.errors.password" class="invalid-feedback">
                                        {{ state.errors.password }}
                                    </div>
                                    <div v-if="state.valid.password" class="valid-feedback">
                                        密码长度符合要求
                                    </div>
                                </div>
                                
                                <div class="mb-3">
                                    <label for="confirmPassword" class="form-label">
                                        确认密码 <span class="text-danger">*</span>
                                    </label>
                                    <div class="input-group">
                                        <input :type="showConfirmPassword ? 'text' : 'password'" 
                                               class="form-control" 
                                               id="confirmPassword"
                                               v-model="state.password.verify"
                                               placeholder="请再次输入新密码"
                                               minlength="6"
                                               required
                                               autocomplete="new-password"
                                               :class="{ 'is-invalid': state.errors.verify, 'is-valid': state.valid.verify }">
                                        <button class="btn btn-outline-secondary" type="button" @click="showConfirmPassword = !showConfirmPassword">
                                            <i class="bi" :class="showConfirmPassword ? 'bi-eye-slash' : 'bi-eye'"></i>
                                        </button>
                                    </div>
                                    <div v-if="state.errors.verify" class="invalid-feedback">
                                        {{ state.errors.verify }}
                                    </div>
                                    <div v-if="state.valid.verify" class="valid-feedback">
                                        两次输入的密码一致
                                    </div>
                                </div>
                                
                                <div class="alert alert-info mb-4" v-if="state.password.value && state.password.value.length > 0">
                                    <div class="d-flex justify-content-between align-items-center">
                                        <span>密码强度：</span>
                                        <div class="progress" style="width: 70%; height: 8px;">
                                            <div class="progress-bar" :class="getPasswordStrengthClass" 
                                                 :style="{width: getPasswordStrength}"></div>
                                        </div>
                                        <small class="ms-2">{{ getPasswordStrengthText }}</small>
                                    </div>
                                </div>
                                
                                <div v-if="authAgreementConfig.enabled" class="text-center mb-3">
                                    <p class="text-muted auth-agreement-text">
                                        重置密码即代表您同意
                                        <a href="javascript:void(0)" @click="method.showAgreement('user')" class="auth-agreement-link">《用户协议》</a>
                                        和
                                        <a href="javascript:void(0)" @click="method.showAgreement('privacy')" class="auth-agreement-link">《隐私协议》</a>
                                    </p>
                                </div>

                                <div class="row auth-switch-row">
                                    <div class="col">
                                        <button type="button"
                                                class="btn auth-switch-btn w-100"
                                                @click="method.switchType('login')">
                                            <i class="bi bi-box-arrow-in-right me-1"></i>记起来了？登录
                                        </button>
                                    </div>
                                    <div class="col">
                                        <button type="submit"
                                                class="btn auth-switch-btn w-100"
                                                :disabled="state.item.wait || !isResetFormValid">
                                            <span v-if="state.item.wait" class="spinner-border spinner-border-sm me-2"></span>
                                            <i v-else class="bi bi-shield-lock me-1"></i>
                                            {{ state.item.wait ? '重置中...' : '重置密码' }}
                                        </button>
                                    </div>
                                </div>
                            </form>
                        </transition>
                    </div>
                </div>
            </div>
        </div>
    </transition>
    
    <transition name="backdrop-fade">
        <div 
            v-if="state.item.dialog"
            class="modal-backdrop fade show"
            @click="method.hide()"
        ></div>
    </transition>

    <!-- 协议弹窗 -->
    <transition name="modal-fade" mode="out-in">
        <div 
            v-if="state.agreement.show"
            class="modal fade show"
            style="display: block;"
            tabindex="-1" 
            aria-labelledby="agreementModalLabel" 
            aria-hidden="false"
            aria-modal="true"
            id="agreementModal"
        >
            <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable auth-modal-dialog" style="max-width: 600px; margin: 1.75rem auto;">
                <div class="modal-content auth-modal-content">
                    <div class="modal-header auth-modal-header">
                        <h5 class="modal-title d-flex align-items-center" id="agreementModalLabel">
                            <i class="bi me-2" :class="state.agreement.type === 'user' ? 'bi-file-earmark-text' : 'bi-shield-check'"></i>
                            {{ state.agreement.type === 'user' ? '《用户协议》' : '《隐私协议》' }}
                        </h5>
                        <button type="button" class="btn-close" @click="method.hideAgreement()" aria-label="Close"></button>
                    </div>

                    <div class="modal-body">
                        <div class="agreement-content whitespace-pre-wrap">{{ agreementContent }}</div>
                    </div>

                    <div class="modal-footer auth-modal-footer">
                        <button type="button" class="btn auth-submit-btn" @click="method.hideAgreement()">
                            <i class="bi bi-check-lg me-1"></i>我已知晓
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </transition>

    <transition name="backdrop-fade">
        <div 
            v-if="state.agreement.show"
            class="modal-backdrop fade show"
            @click="method.hideAgreement()"
        ></div>
    </transition>
</template>

<script setup>
import { reactive, ref, getCurrentInstance, onUnmounted, computed, watch } from 'vue'
import { cache } from '@/utils/network.js'
import utils from '@/utils/utils.js'
import { request } from '@/utils/network.js'
import cryptoUtil from '@/utils/crypto'
import { useCommStore } from '@/store/comm'
import { useConfigStore } from '@/store/config'
import { toast } from '@/utils/app'
import { escapeHtml, detectXSS } from '@/utils/security'

const { proxy } = getCurrentInstance()
const emit = defineEmits(['finish'])
const store = {
    comm: useCommStore(),
    config: useConfigStore()
}

const state = reactive({
    item: {
        type: 'login',
        wait: false,
        loading: false,
        finish: false,
        dialog: false,
        second: 0,
    },
    agreement: {
        show: false,
        type: 'user', // 'user' 用户协议, 'privacy' 隐私协议
    },
    struct: {
        account: '',
        password: '',
        nickname: '',
        social: '',
        code: '',
    },
    password: {
        value: '',
        verify: '',
    },
    errors: {
        account: '',
        password: '',
        nickname: '',
        social: '',
        code: '',
        verify: '',
    },
    valid: {
        account: false,
        password: false,
        nickname: false,
        social: false,
        code: false,
        verify: false,
    },
    timer: null,
})

const showPassword = ref(false)
const showConfirmPassword = ref(false)

// 输入类型判断计算属性
const getSocialInputType = computed(() => {
    const social = state.struct.social
    if (!social) return 'text'
    
    // 如果包含@符号，判断为邮箱
    if (social.includes('@')) {
        return 'email'
    }
    
    // 如果是数字且长度符合手机号，判断为手机号
    if (/^1[3-9]\d{0,9}$/.test(social)) {
        return 'tel'
    }
    
    return 'text'
})

// 协议配置（从 store.comm.siteInfo 读取）
const authAgreementConfig = computed(() => {
    const config = store.comm.siteInfo || {}
    const authDialogAgreement = config.auth_dialog_agreement || {}
    
    return {
        enabled: authDialogAgreement.enabled !== false,
        user_agreement_content: authDialogAgreement.user_agreement_content || getDefaultUserAgreement(),
        privacy_agreement_content: authDialogAgreement.privacy_agreement_content || getDefaultPrivacyAgreement()
    }
})

// 默认用户协议内容
const getDefaultUserAgreement = () => {
    return `用户协议

欢迎使用我们的服务！请仔细阅读以下用户协议：
1. 服务条款
您必须年满13周岁才能使用本服务。
2. 账户安全
您有责任维护账户密码的安全性，并对您账户下的所有活动负责。
3. 用户行为规范
您同意不会：
- 发布违法、有害或侵犯他人权益的内容
- 滥用或破坏服务
- 未经授权访问他人账户
4. 知识产权
您发布的内容的知识产权归您所有，但我们有权在服务中使用、复制和传播。
5. 服务变更
我们保留随时修改或终止服务的权利。
6. 免责声明
我们不对服务的准确性、完整性或可靠性提供保证。
7. 协议变更
我们可能会更新本协议，您继续使用服务即表示接受更新后的协议。`
}

// 默认隐私协议内容
const getDefaultPrivacyAgreement = () => {
    return `隐私协议
    
我们重视您的隐私。以下是我们的隐私政策：
1. 收集的信息
我们可能收集以下信息：
- 您的账户信息（用户名、邮箱、手机号）
- 使用数据（访问记录、浏览行为）
- 设备信息（IP地址、浏览器类型）
2. 信息使用
我们使用收集的信息来：
- 提供和改进服务
- 个性化您的体验
- 发送重要通知
- 保障服务安全
3. 信息共享
我们不会向第三方出售您的个人信息。仅在以下情况下共享：
- 法律要求
- 保护我们的权益
- 经您同意
4. 数据安全
我们采取合理措施保护您的数据，但无法保证绝对安全。
5. 您的权利
您有权访问、更正或删除您的个人信息。
6. Cookie使用
我们使用Cookie来改善您的体验，您可以在浏览器中禁用。
7. 政策变更
我们可能更新隐私政策，变更将在此页面发布。`
}

// 当前协议内容
const agreementContent = computed(() => {
    const content = state.agreement.type === 'user' 
        ? authAgreementConfig.value.user_agreement_content 
        : authAgreementConfig.value.privacy_agreement_content
    
    return escapeHtml(content || '')
})

// 获取模态框标题
const getModalTitle = computed(() => {
    switch (state.item.type) {
        case 'login':
            return '登录账号'
        case 'register':
            return '注册账号'
        case 'reset':
            return '忘记密码'
        default:
            return '登录账号'
    }
})

// 获取模态框图标
const getModalIcon = computed(() => {
    switch (state.item.type) {
        case 'login':
            return 'bi-box-arrow-in-right'
        case 'register':
            return 'bi-person-plus-fill'
        case 'reset':
            return 'bi-key-fill'
        default:
            return 'bi-box-arrow-in-right'
    }
})

// 登录表单验证计算属性
const isLoginFormValid = computed(() => {
    return state.valid.account && state.valid.password
})

// 注册表单验证计算属性
const isRegisterFormValid = computed(() => {
    return state.valid.nickname && 
           state.valid.account && 
           state.valid.social && 
           state.valid.code && 
           state.valid.password && 
           state.valid.verify
})

// 找回密码表单验证计算属性
const isResetFormValid = computed(() => {
    return state.valid.social && 
           state.valid.code && 
           state.valid.password && 
           state.valid.verify
})

// 账号验证
const validateAccount = (account) => {
    if (!account) {
        state.errors.account = '请输入账号'
        state.valid.account = false
        return
    }
    
    if (detectXSS(account)) {
        state.errors.account = '输入内容包含非法字符'
        state.valid.account = false
        return
    }
    
    const accountRegex = state.item.type === 'register' 
        ? /^[a-zA-Z0-9_]{4,20}$/ 
        : /^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$|^1[3-9]\d{9}$|^[a-zA-Z0-9_]{4,20}$/
    
    if (!accountRegex.test(account)) {
        state.errors.account = state.item.type === 'register' 
            ? '账号格式不正确（4-20位字母数字下划线）' 
            : '账号格式不正确（支持邮箱、手机号、4-20位字母数字下划线）'
        state.valid.account = false
        return
    }
    
    state.errors.account = ''
    state.valid.account = true
}

// 密码验证
const validatePassword = (password) => {
    if (!password) {
        state.errors.password = state.item.type === 'reset' ? '请输入新密码' : '请输入密码'
        state.valid.password = false
        return
    }
    
    if (password.length < 6) {
        state.errors.password = '密码长度不能少于6位'
        state.valid.password = false
        return
    }
    
    state.errors.password = ''
    state.valid.password = true
}

// 昵称验证
const validateNickname = (nickname) => {
    if (!nickname) {
        state.errors.nickname = '请输入昵称'
        state.valid.nickname = false
        return
    }
    
    if (detectXSS(nickname)) {
        state.errors.nickname = '输入内容包含非法字符'
        state.valid.nickname = false
        return
    }
    
    if (nickname.length < 2 || nickname.length > 20) {
        state.errors.nickname = '昵称长度应在2-20位之间'
        state.valid.nickname = false
        return
    }
    
    state.errors.nickname = ''
    state.valid.nickname = true
}

// 邮箱或手机号验证
const validateSocial = (social) => {
    if (!social) {
        state.errors.social = '请输入邮箱或手机号'
        state.valid.social = false
        return
    }
    
    const isPhone = /^1[3-9]\d{9}$/.test(social)
    const isEmail = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(social)
    
    if (!isPhone && !isEmail) {
        // 尝试判断输入类型并提供更具体的错误提示
        if (social.includes('@')) {
            state.errors.social = '邮箱格式不正确，请检查'
        } else {
            state.errors.social = '手机号格式不正确，请检查'
        }
        state.valid.social = false
        return
    }
    
    state.errors.social = ''
    state.valid.social = true
}

// 验证码验证
const validateCode = (code) => {
    if (!code) {
        state.errors.code = '请输入验证码'
        state.valid.code = false
        return
    }
    
    if (code.length !== 6) {
        state.errors.code = '验证码长度应为6位'
        state.valid.code = false
        return
    }
    
    if (!/^\d{6}$/.test(code)) {
        state.errors.code = '验证码应为纯数字'
        state.valid.code = false
        return
    }
    
    state.errors.code = ''
    state.valid.code = true
}

// 确认密码验证
const validateVerify = (verify) => {
    if (!verify) {
        state.errors.verify = state.item.type === 'reset' ? '请再次输入新密码' : '请再次输入密码'
        state.valid.verify = false
        return
    }
    
    if (verify !== state.password.value) {
        state.errors.verify = '两次输入的密码不一致'
        state.valid.verify = false
        return
    }
    
    state.errors.verify = ''
    state.valid.verify = true
}

// 监听表单输入变化
watch(() => state.struct.account, (newValue) => {
    validateAccount(newValue)
})

watch(() => state.struct.password, (newValue) => {
    validatePassword(newValue)
})

watch(() => state.struct.nickname, (newValue) => {
    validateNickname(newValue)
})

watch(() => state.struct.social, (newValue) => {
    validateSocial(newValue)
})

watch(() => state.struct.code, (newValue) => {
    validateCode(newValue)
})

watch(() => state.password.value, (newValue) => {
    validatePassword(newValue)
    validateVerify(state.password.verify)
})

watch(() => state.password.verify, (newValue) => {
    validateVerify(newValue)
})

// 密码强度相关计算属性
const getPasswordStrength = computed(() => {
    const password = state.password.value || ''
    if (password.length === 0) return '0%'
    
    let strength = 0
    if (password.length >= 6) strength += 25
    if (/[A-Z]/.test(password)) strength += 25
    if (/[a-z]/.test(password)) strength += 25
    if (/[0-9]/.test(password) || /[^A-Za-z0-9]/.test(password)) strength += 25
    
    return `${Math.min(strength, 100)}%`
})

const getPasswordStrengthClass = computed(() => {
    const strength = parseInt(getPasswordStrength.value)
    if (strength < 25) return 'bg-secondary'
    if (strength < 50) return 'bg-secondary'
    if (strength < 75) return 'bg-secondary'
    return 'bg-secondary'
})

const getPasswordStrengthText = computed(() => {
    const strength = parseInt(getPasswordStrength.value)
    if (strength < 25) return '很弱'
    if (strength < 50) return '较弱'
    if (strength < 75) return '中等'
    return '强'
})

// 验证码防刷缓存
const getCacheKey = (type, contact) => {
  return `verify-code-${type}-${contact}`
}

const getDailyLimitKey = (type, contact) => {
  return `verify-code-daily-${type}-${contact}`
}

// 检查是否可以发送验证码
const canSendCode = (type, contact) => {
  // 检查时间间隔
  const cacheKey = getCacheKey(type, contact)
  const lastSendTime = localStorage.getItem(cacheKey)
  if (lastSendTime) {
    const timeDiff = Date.now() - parseInt(lastSendTime)
    if (timeDiff < 60 * 1000) {
      return false
    }
  }
  
  // 检查每日发送上限
  const dailyLimitKey = getDailyLimitKey(type, contact)
  const dailyCount = localStorage.getItem(dailyLimitKey)
  if (dailyCount && parseInt(dailyCount) >= 10) {
    return false
  }
  
  return true
}

// 检查频率限制（用于验证码登录）
const checkRateLimit = (contact) => {
  return canSendCode('social', contact)
}

// 开始倒计时
const startCountdown = () => {
  state.struct.second = 60
  state.item.wait = false
  
  // 清除之前的定时器
  if (state.timer) {
    clearInterval(state.timer)
    state.timer = null
  }
  
  state.timer = setInterval(() => {
    if (state.struct.second > 0) {
      state.struct.second--
    } else {
      clearInterval(state.timer)
      state.timer = null
      state.item.wait = false
      state.item.loading = false
    }
  }, 1000)
}

// 记录发送验证码
const recordSendCode = (type, contact) => {
  // 记录发送时间
  const cacheKey = getCacheKey(type, contact)
  localStorage.setItem(cacheKey, Date.now().toString())
  
  // 记录每日发送次数
  const dailyLimitKey = getDailyLimitKey(type, contact)
  const currentCount = localStorage.getItem(dailyLimitKey) || '0'
  const newCount = parseInt(currentCount) + 1
  localStorage.setItem(dailyLimitKey, newCount.toString())
  
  // 设置每日计数过期时间（24小时后）
  setTimeout(() => {
    localStorage.removeItem(dailyLimitKey)
  }, 24 * 60 * 60 * 1000)
}

const cryptoUtils = {
    generateSecureKey(length = 16, salt = '') {
        const charset = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
        let secureKey = ''
        
        for (let i = 0; i < length; i++) {
            secureKey += charset.charAt(Math.floor(Math.random() * charset.length))
        }
        
        const hash = cryptoUtil.hash?.md5 ? cryptoUtil.hash.md5(`${secureKey}${Date.now()}${salt}`) : 
                    cryptoUtil.md5 ? cryptoUtil.md5(`${secureKey}${Date.now()}${salt}`) :
                    secureKey
        
        return hash.substring(0, length)
    }
}

const showNotification = (message, type = 'info') => {
    try {
        const toastType = type === 'success' ? 'success' : 
                       type === 'error' ? 'error' : 
                       type === 'warning' ? 'warning' : 'info'
        toast[toastType](message)
    } catch (error) {
        // 忽略显示消息失败的错误
    }
}

const method = {
    // 切换认证类型
    switchType(type) {
        state.item.type = type
        state.struct.code = ''
        state.errors.code = ''
        state.valid.code = false
        

        
        // 重置定时器
        if (state.timer) {
            clearInterval(state.timer)
            state.timer = null
            state.item.second = 0
            state.item.loading = false
        }
        
        // 重置错误和验证状态
        state.errors = {
            account: '',
            password: '',
            nickname: '',
            social: '',
            code: '',
            verify: '',
        }
        state.valid = {
            account: false,
            password: false,
            nickname: false,
            social: false,
            code: false,
            verify: false,
        }
        
        // 聚焦对应输入框并检查自动填充
        setTimeout(() => {
            if (type === 'login') {
                const accountInput = document.getElementById('loginAccountInput')
                const passwordInput = document.getElementById('loginPasswordInput')
                if (accountInput) accountInput.focus()
                // 检查自动填充并触发验证
                if (state.struct.account) {
                    validateAccount(state.struct.account)
                }
                if (state.struct.password) {
                    validatePassword(state.struct.password)
                }
            } else if (type === 'register') {
                const nicknameInput = document.getElementById('nicknameInput')
                if (nicknameInput) nicknameInput.focus()
                // 检查自动填充并触发验证
                if (state.struct.nickname) {
                    validateNickname(state.struct.nickname)
                }
                if (state.struct.account) {
                    validateAccount(state.struct.account)
                }
                if (state.struct.social) {
                    validateSocial(state.struct.social)
                }
                if (state.password.value) {
                    validatePassword(state.password.value)
                }
                if (state.password.verify) {
                    validateVerify(state.password.verify)
                }
            } else if (type === 'reset') {
                const contactInput = document.getElementById('contactInput')
                if (contactInput) contactInput.focus()
                // 检查自动填充并触发验证
                if (state.struct.social) {
                    validateSocial(state.struct.social)
                }
                if (state.password.value) {
                    validatePassword(state.password.value)
                }
                if (state.password.verify) {
                    validateVerify(state.password.verify)
                }
            }
        }, 300)
    },

    // 登录方法
    async login() {
        // 防并发重复提交
        if (state.item.wait) return
        // 手动触发验证
        validateAccount(state.struct.account)
        validatePassword(state.struct.password)
        
        if (!isLoginFormValid.value) {
            showNotification('请检查表单填写是否正确', 'warning')
            return
        }
        
        state.item.wait = true

        try {
            const unix = await method.unix()
            
            const iv = cryptoUtil.token ? cryptoUtil.token(`iv-${unix}`, 16, 'aes') : 
                      cryptoUtils.generateSecureKey(16, `iv-${unix}`)
            
            const key = cryptoUtil.token ? cryptoUtil.token(`key-${unix}`, 16, 'aes') : 
                       cryptoUtils.generateSecureKey(16, `key-${unix}`)
            
            const safeIv = iv.length >= 16 ? iv.substring(0, 16) : iv.padEnd(16, '0')
            const safeKey = key.length >= 16 ? key.substring(0, 16) : key.padEnd(16, '0')

            let AES
            if (cryptoUtil.createAES) {
                AES = cryptoUtil.createAES(safeKey, safeIv)
            } else if (cryptoUtil.AES) {
                AES = new cryptoUtil.AES(safeKey, safeIv)
            } else {
                throw new Error('加密工具不可用')
            }

            const params = {
                account: AES.encrypt(state.struct.account),
                password: AES.encrypt(state.struct.password)
            }

            const { data, code, msg } = await request.post('/api/comm/login', params, {
                headers: {
                    'X-Khronos': unix,
                    'X-Gorgon': `${safeKey} ${safeIv}`,
                    'X-Argus': AES.encrypt(JSON.stringify({
                        unix, 
                        account: state.struct.account, 
                        password: state.struct.password
                    }))
                },
                timeout: 10000
            })

            state.item.wait = false

            if (code === 200) {
                const userData = data?.user || {}
                const userStatus = Number(userData.status) || 0
                
                if (userStatus === 1) {
                    method.clearCache()
                    showNotification('当前账号已被冻结，请联系管理员！', 'error')
                    method.animation()
                    return
                }

                if (!data.token || !data.user || !data.user.id) {
                    throw new Error('登录成功但返回数据不完整')
                }

                if (cache.set) {
                    cache.set('user-info', data.user, 7 * 24 * 60)
                } else if (cache.put) {
                    cache.put('user-info', data.user, 7 * 24 * 60)
                } else {
                    localStorage.setItem('user-info', JSON.stringify({
                        data: data.user,
                        expire: Date.now() + 7 * 24 * 60 * 60 * 1000
                    }))
                }
                
                if (utils.set?.cookie) {
                    utils.set.cookie(
                        globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', 
                        data.token, 
                        7 * 24 * 60 * 60
                    )
                }
                
                state.item.finish = true
                method.hide()
                
                store.comm.login.finish = true
                store.comm.login.user = data.user
                store.comm.switchAuth('login', false)
                
                emit('finish', data.user)
                return
            }
            
            if (code === 201) {
                showNotification(msg, 'info')
                return
            }

            // 封禁检查（403）
            if (code === 403 && msg.includes('封禁')) {
                showNotification(msg, 'error')
                // 显示申诉入口
                setTimeout(() => {
                    toast.warning('如对封禁有异议，可前往 <a href="/appeal">申诉页面</a> 提交申诉')
                }, 500)
                return
            }

            method.animation()
            showNotification(msg || '登录失败，请检查账号密码', 'error')
            method.clearCache()

        } catch (error) {
            let errorMsg = '登录失败，请稍后重试'
            if (error.message.includes('超时')) {
                errorMsg = '请求超时，请检查网络连接'
            } else if (error.message.includes('加密')) {
                errorMsg = '加密处理失败'
            }
            
            showNotification(errorMsg, 'error')
            method.clearCache()
            method.animation()
            state.item.wait = false
        }
    },



    // 注册方法
    async register() {
        // 防并发重复提交
        if (state.item.wait) return
        // 手动触发验证
        validateNickname(state.struct.nickname)
        validateAccount(state.struct.account)
        validateSocial(state.struct.social)
        validateCode(state.struct.code)
        validatePassword(state.password.value)
        validateVerify(state.password.verify)
        
        if (!isRegisterFormValid.value) {
            showNotification('请检查表单填写是否正确', 'warning')
            return
        }

        state.item.wait = true

        try {
            const { code, msg, data } = await request.post('/api/comm/register', {
                ...state.struct, 
                password: state.password.value
            })

            state.item.wait = false

            if (code !== 200) {
                showNotification(msg, 'error')
                return
            }

            cache.set('user-info', data.user, 10)
            utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', data.token, 7 * 24 * 60 * 60)
            method.hide()
            
            store.comm.login.finish = true
            store.comm.login.user = data.user
            store.comm.switchAuth('register', false)
            
            emit('finish', data.user)
        } catch (error) {
            state.item.wait = false
            showNotification('网络异常，请稍后再试！', 'error')
        }
    },

    // 重置密码方法
    async reset() {
        // 防并发重复提交
        if (state.item.wait) return
        // 手动触发验证
        validateSocial(state.struct.social)
        validateCode(state.struct.code)
        validatePassword(state.password.value)
        validateVerify(state.password.verify)
        
        if (!isResetFormValid.value) {
            showNotification('请检查表单填写是否正确', 'warning')
            return
        }

        try {
            state.item.wait = true
            const { code: resCode, msg } = await request.post('/api/comm/reset-password', {
                ...state.struct, 
                password: state.password.value
            })

            state.item.wait = false

            if (resCode !== 200) {
                throw new Error(msg || '重置密码失败！')
            }

            method.hide()
            emit('finish')

        } catch (error) {
            state.item.wait = false
            showNotification(error.message || '网络异常，请稍后再试！', 'error')
        }
    },

    // 获取验证码方法
    async code(type) {
        const social = state.struct.social
        if (utils.is.empty(social)) {
            showNotification('请输入您的邮箱或手机号！', 'warning')
            return
        }

        const isPhone = /^1[3-9]\d{9}$/.test(social)
        const isEmail = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(social)
        if (!isPhone && !isEmail) {
            showNotification('请填写正确的手机号或邮箱！', 'warning')
            return
        }

        // 检查防刷
        if (!canSendCode(type, social)) {
            const cacheKey = getCacheKey(type, social)
            const lastSendTime = localStorage.getItem(cacheKey)
            const dailyLimitKey = getDailyLimitKey(type, social)
            const dailyCount = localStorage.getItem(dailyLimitKey)
            
            if (lastSendTime) {
                const timeDiff = Date.now() - parseInt(lastSendTime)
                if (timeDiff < 60 * 1000) {
                    showNotification('发送过于频繁，请60秒后再试', 'error')
                    return
                }
            }
            
            if (dailyCount && parseInt(dailyCount) >= 10) {
                showNotification('今日发送验证码次数已达上限，请明日再试', 'error')
                return
            }
        }

        try {
            state.item.loading = true
            const { code: resCode, msg } = await request.post(type === 'register' ? '/api/comm/register' : '/api/comm/reset-password', {
                social,
                account: state.struct.account
            })

            if (!utils.in.array(resCode, [200, 201])) {
                // 处理具体错误码
                let errorMsg = msg || '发送验证码失败！'
                if (msg && msg.includes('MOBILE_NUMBER_ILLEGAL')) {
                    errorMsg = '手机号码格式错误'
                } else if (msg && msg.includes('BUSINESS_LIMIT_CONTROL')) {
                    errorMsg = '触发号码天级流控，请明日再试'
                } else if (msg && msg.includes('FREQUENCY_FAIL')) {
                    errorMsg = '发送过于频繁，请60秒后再试'
                } else if (msg && msg.includes('INVALID_PARAMETERS')) {
                    errorMsg = '非法参数'
                } else if (msg && msg.includes('FUNCTION_NOT_OPENED')) {
                    errorMsg = '没有开通融合认证功能'
                }
                throw new Error(errorMsg)
            }

            showNotification(msg || '验证码发送成功！', 'success')
            // 记录发送验证码
            recordSendCode(type, social)
            // 使用统一的倒计时函数
            startCountdown()
            state.item.loading = true

        } catch (error) {
            state.item.loading = false
            showNotification(error.message || '网络异常，验证码发送失败！', 'error')
        }
    },

    // 显示方法
    show(type = 'login') {
        state.item.type = type
        state.item.dialog = true
        state.item.wait = false
        state.item.loading = false
        state.item.second = 0
        
        // 重置表单数据
        state.struct.account = ''
        state.struct.password = ''
        state.struct.nickname = ''
        state.struct.social = ''
        state.struct.code = ''
        state.password.value = ''
        state.password.verify = ''
        
        // 重置错误和验证状态
        state.errors = {
            account: '',
            password: '',
            nickname: '',
            social: '',
            code: '',
            verify: '',
        }
        state.valid = {
            account: false,
            password: false,
            nickname: false,
            social: false,
            code: false,
            verify: false,
        }
        
        // 重置密码显示状态
        showPassword.value = false
        showConfirmPassword.value = false
        
        // 禁止页面滚动
        document.body.style.overflow = 'hidden'
        document.body.style.position = 'fixed'
        document.body.style.width = '100%'
        document.body.style.height = '100%'
        document.body.style.top = '0'
        document.body.style.left = '0'
        document.body.style.zIndex = '1059'
        
        // 聚焦对应输入框
        setTimeout(() => {
            if (type === 'login') {
                const accountInput = document.getElementById('loginAccountInput')
                if (accountInput) accountInput.focus()
            } else if (type === 'register') {
                const nicknameInput = document.getElementById('nicknameInput')
                if (nicknameInput) nicknameInput.focus()
            } else if (type === 'reset') {
                const contactInput = document.getElementById('contactInput')
                if (contactInput) contactInput.focus()
            }
        }, 300)
    },

    // 隐藏方法
    hide() {
        state.item.dialog = false
        showPassword.value = false
        showConfirmPassword.value = false
        
        // 恢复页面滚动
        document.body.style.overflow = ''
        document.body.style.position = ''
        document.body.style.width = ''
        document.body.style.height = ''
        document.body.style.top = ''
        document.body.style.left = ''
        document.body.style.zIndex = ''
        
        // 重置定时器
        if (state.timer) {
            clearInterval(state.timer)
            state.timer = null
            state.item.second = 0
            state.item.loading = false
        }
        

        
        // 重置表单数据
        state.struct.account = ''
        state.struct.password = ''
        state.struct.nickname = ''
        state.struct.social = ''
        state.struct.code = ''
        state.password.value = ''
        state.password.verify = ''
        
        // 重置错误和验证状态
        state.errors = {
            account: '',
            password: '',
            nickname: '',
            social: '',
            code: '',
            verify: '',
        }
        state.valid = {
            account: false,
            password: false,
            nickname: false,
            social: false,
            code: false,
            verify: false,
        }
    },

    // 显示协议弹窗
    showAgreement(type) {
        state.agreement.type = type
        state.agreement.show = true
        document.body.style.overflow = 'hidden'
        document.body.style.position = 'fixed'
        document.body.style.width = '100%'
        document.body.style.height = '100%'
        document.body.style.top = '0'
        document.body.style.left = '0'
        document.body.style.zIndex = '1059'
    },

    // 隐藏协议弹窗
    hideAgreement() {
        state.agreement.show = false
        document.body.style.overflow = ''
        document.body.style.position = ''
        document.body.style.width = ''
        document.body.style.height = ''
        document.body.style.top = ''
        document.body.style.left = ''
        document.body.style.zIndex = ''
    },

    // 动画方法
    animation() {
        const el = proxy.$refs.password
        if (el) {
            el.classList.add('shake-horizontal')
            setTimeout(() => {
                el.classList.remove('shake-horizontal')
            }, 1000)
        }
    },

    // 清除缓存方法
    clearCache() {
        try {
            if (utils.set?.cookie) {
                utils.set.cookie(
                    globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', '', -1
                )
            }
            
            if (cache && typeof cache.remove === 'function') {
                cache.remove('user-info')
            } else if (cache && typeof cache.del === 'function') {
                cache.del('user-info')
            } else if (cache && typeof cache.delete === 'function') {
                cache.delete('user-info')
            } else if (cache && typeof cache.clear === 'function') {
                cache.clear()
            }
            
            if (typeof localStorage !== 'undefined') {
                localStorage.removeItem('user-info')
            }
            if (typeof sessionStorage !== 'undefined') {
                sessionStorage.removeItem('user-info')
            }
            
            if (store.comm.login) {
                store.comm.login.finish = false
                store.comm.login.user = null
            }
        } catch (error) {
            // 忽略缓存清理错误
        }
    },

    // 获取unix时间戳方法
    async unix() {
        try {
            const { code, data } = await request.get('/dev/info/time', {
                timeout: 5000
            })
            if (code === 200 && data.unix && typeof data.unix === 'number') {
                return data.unix
            }
        } catch (error) {
            // 忽略获取服务器时间失败的错误
        }
        
        return Math.round(Date.now() / 1000)
    }
}

watch(() => state.struct.code, (val) => {
    if (val) {
        state.struct.code = val.replace(/\s+/g, '')
    }
})

onUnmounted(() => {
    if (state.timer) clearInterval(state.timer)
    
    // 确保在组件卸载时恢复页面滚动
    document.body.style.overflow = ''
    document.body.style.position = ''
    document.body.style.width = ''
    document.body.style.height = ''
    document.body.style.top = ''
    document.body.style.left = ''
    document.body.style.zIndex = ''
})

defineExpose({
    show: method.show,
    hide: method.hide,
    switchType: method.switchType
})
</script>


<style scoped>
/* ==========================================
 * 过渡动画
 * ========================================== */
.modal-fade-enter-active,
.modal-fade-leave-active {
    transition: all 0.3s ease;
}

.modal-fade-enter-from,
.modal-fade-leave-to {
    opacity: 0;
    transform: scale(0.95) translateY(-10px);
}

.backdrop-fade-enter-active,
.backdrop-fade-leave-active {
    transition: opacity 0.3s ease;
}

.backdrop-fade-enter-from,
.backdrop-fade-leave-to {
    opacity: 0;
}

.form-switch-enter-active,
.form-switch-leave-active {
    transition: all 0.35s ease;
}

.form-switch-enter-from {
    opacity: 0;
    transform: translateX(20px);
}

.form-switch-leave-to {
    opacity: 0;
    transform: translateX(-20px);
}

/* ==========================================
 * 模态框容器
 * ========================================== */
:deep(.auth-modal-content) {
    border: 1px solid var(--bs-border-color);
    border-radius: var(--wx-radius-lg);
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15), 0 8px 20px rgba(0, 0, 0, 0.08);
    overflow: hidden;
}

:deep(.auth-modal-header) {
    background: var(--bs-tertiary-bg);
    border-bottom: 1px solid var(--bs-border-color);
    padding: 1.125rem 1.5rem;
}

:deep(.auth-modal-header .modal-title) {
    font-weight: 700;
    font-size: 1.1rem;
    color: var(--bs-body-color);
}

:deep(.auth-modal-header .modal-title i) {
    color: var(--bs-primary);
    font-size: 1.2rem;
}

:deep(.auth-modal-footer) {
    border-top: 1px solid var(--bs-border-color);
    padding: 0.875rem 1.5rem;
    background: var(--bs-tertiary-bg);
}

:deep(.modal-body) {
    padding: 1.5rem;
}

/* ==========================================
 * 主提交按钮
 * ========================================== */
.auth-submit-btn {
    background: linear-gradient(135deg, var(--bs-primary), color-mix(in srgb, var(--bs-primary) 75%, #6c5ce7));
    color: #fff;
    border: none;
    padding: 0.75rem 1rem;
    border-radius: var(--wx-radius-sm);
    font-size: 0.95rem;
    font-weight: 600;
    letter-spacing: 0.5px;
    transition: all 0.25s ease;
    box-shadow: 0 4px 12px rgba(var(--bs-primary-rgb), 0.25);
}

.auth-submit-btn:hover:not(:disabled) {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(var(--bs-primary-rgb), 0.35);
    filter: brightness(1.05);
}

.auth-submit-btn:active:not(:disabled) {
    transform: translateY(0);
}

.auth-submit-btn:disabled {
    opacity: 0.65;
    cursor: not-allowed;
    box-shadow: none;
}

/* ==========================================
 * 切换按钮
 * ========================================== */
.auth-switch-btn {
    background: transparent;
    border: 1px solid var(--bs-border-color);
    color: var(--bs-secondary-color);
    padding: 0.625rem 0.5rem;
    border-radius: var(--wx-radius-sm);
    font-size: 0.85rem;
    font-weight: 500;
    transition: all 0.2s ease;
}

.auth-switch-btn:hover {
    border-color: var(--bs-primary);
    color: var(--bs-primary);
    background-color: rgba(var(--bs-primary-rgb), 0.05);
}

/* ==========================================
 * 协议链接
 * ========================================== */
.auth-agreement-text {
    font-size: 0.8rem !important;
    line-height: 1.6;
    margin: 0;
}

.auth-agreement-link {
    color: var(--bs-primary);
    text-decoration: underline;
    transition: opacity 0.2s;
    margin: 0 0.25rem;
}

.auth-agreement-link:hover {
    opacity: 0.8;
}

/* ==========================================
 * 密码强度
 * ========================================== */
:deep(.alert-info) {
    background: var(--bs-tertiary-bg);
    border: 1px solid var(--bs-border-color);
    border-radius: var(--wx-radius-sm);
    color: var(--bs-body-color);
    padding: 0.75rem 1rem;
    font-size: 0.85rem;
}

:deep(.progress) {
    background: var(--bs-border-color);
    border-radius: 4px;
}

/* ==========================================
 * 协议内容
 * ========================================== */
.agreement-content {
    white-space: pre-wrap;
    word-break: break-all;
    line-height: 1.8;
    font-size: 0.9rem;
    color: var(--bs-body-color);
    padding: 0.5rem;
    background: var(--bs-tertiary-bg);
    border-radius: var(--wx-radius-sm);
    border: 1px solid var(--bs-border-color);
}

/* ==========================================
 * 表单文本
 * ========================================== */
:deep(.form-text) {
    font-size: 0.78rem;
}

:deep(.valid-feedback),
:deep(.invalid-feedback) {
    font-size: 0.8rem;
}

/* ==========================================
 * 验证码按钮
 * ========================================== */
:deep(.btn-outline-secondary) {
    font-size: 0.85rem;
    border-color: var(--bs-border-color);
    color: var(--bs-secondary-color);
}

:deep(.btn-outline-secondary:hover) {
    background-color: var(--bs-tertiary-bg);
    border-color: var(--bs-primary);
    color: var(--bs-primary);
}

/* ==========================================
 * 抖动动画
 * ========================================== */
:deep(.shake-horizontal) {
    animation: shake-horizontal 0.8s ease;
}

@keyframes shake-horizontal {
    0%, 100% { transform: translateX(0); }
    10%, 30%, 50%, 70%, 90% { transform: translateX(-6px); }
    20%, 40%, 60%, 80% { transform: translateX(6px); }
}

/* ==========================================
 * 响应式
 * ========================================== */
@media (max-width: 576px) {
    :deep(.auth-modal-content) {
        border-radius: var(--wx-radius-md);
    }

    :deep(.modal-body) {
        padding: 1.25rem 1rem;
    }

    :deep(.auth-modal-header) {
        padding: 1rem 1.125rem;
    }

    .auth-switch-btn {
        font-size: 0.8rem;
        padding: 0.5rem 0.25rem;
    }
}
</style>