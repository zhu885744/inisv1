<template>
    <div class="page-container profile-page">
        <div class="profile-layout">
            <!-- 左侧：用户卡片 -->
            <div class="profile-card">
                <el-card shadow="never" class="user-card">
                    <div class="user-card-body">
                        <el-avatar :src="user.avatar" :size="88" class="user-avatar">
                            {{ (user.nickname || user.account || 'U')[0] }}
                        </el-avatar>
                        <div class="user-name">{{ user.nickname || '未设置昵称' }}</div>
                        <div class="user-account">@{{ user.account || '暂无账号' }}</div>
                        <el-tag v-if="user.title" size="small" class="user-title-tag">{{ user.title }}</el-tag>
                        <el-divider />
                        <div class="user-meta">
                            <div class="meta-item">
                                <span class="meta-label">用户ID</span>
                                <span class="meta-value">{{ user.id || '-' }}</span>
                            </div>
                            <div class="meta-item">
                                <span class="meta-label">注册时间</span>
                                <span class="meta-value">{{ formatTime(user.create_time) }}</span>
                            </div>
                            <div class="meta-item">
                                <span class="meta-label">最近登录</span>
                                <span class="meta-value">{{ formatTime(user.login_time) }}</span>
                            </div>
                        </div>
                    </div>
                </el-card>
            </div>

            <!-- 右侧：设置区 -->
            <div class="profile-settings">
                <!-- 基本资料 -->
                <el-card shadow="never" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span class="card-title">基本资料</span>
                        </div>
                    </template>
                    <el-form :model="basicForm" label-width="90px" label-position="right">
                        <el-form-item label="头像">
                            <div class="avatar-editor">
                                <el-avatar :src="basicForm.avatar" :size="64" class="avatar-editor-preview">
                                    {{ (basicForm.nickname || 'U')[0] }}
                                </el-avatar>
                                <div class="avatar-editor-actions">
                                    <el-button size="small" :loading="uploading" @click="uploadAvatar">上传头像</el-button>
                                    <el-button size="small" text type="danger" v-if="basicForm.avatar" @click="basicForm.avatar = ''">移除</el-button>
                                </div>
                            </div>
                        </el-form-item>
                        <el-form-item label="昵称">
                            <el-input v-model="basicForm.nickname" maxlength="20" placeholder="请输入昵称" show-word-limit />
                        </el-form-item>
                        <el-form-item label="性别">
                            <el-radio-group v-model="basicForm.gender">
                                <el-radio :value="'boy'">男</el-radio>
                                <el-radio :value="'girl'">女</el-radio>
                                <el-radio :value="''">保密</el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <el-form-item label="头衔">
                            <el-input v-model="basicForm.title" maxlength="20" placeholder="请输入头衔" />
                        </el-form-item>
                        <el-form-item label="个人简介">
                            <el-input
                                v-model="basicForm.description"
                                type="textarea"
                                :rows="4"
                                maxlength="200"
                                placeholder="请输入个人简介"
                                show-word-limit
                            />
                        </el-form-item>
                        <el-form-item>
                            <el-button type="primary" :loading="basicSaving" @click="saveBasic">保存修改</el-button>
                            <el-button @click="resetBasic">重置</el-button>
                        </el-form-item>
                    </el-form>
                </el-card>

                <!-- 账号安全 -->
                <el-card shadow="never" class="settings-card">
                    <template #header>
                        <div class="card-header">
                            <span class="card-title">账号安全</span>
                        </div>
                    </template>

                    <!-- 修改账号 -->
                    <div class="security-block">
                        <div class="security-row">
                            <div class="security-info">
                                <div class="security-title">登录账号</div>
                                <div class="security-desc">当前账号：{{ user.account || '-' }}</div>
                            </div>
                            <el-button size="small" @click="accountVisible = !accountVisible">
                                {{ accountVisible ? '取消' : '修改' }}
                            </el-button>
                        </div>
                        <transition name="el-fade-in">
                            <div v-if="accountVisible" class="security-form">
                                <el-alert type="warning" :closable="false" show-icon title="修改账号后将影响登录，请谨慎操作" class="mb-12" />
                                <el-input v-model="accountForm.account" maxlength="20" placeholder="请输入新账号（字母、数字、下划线，4-20位）" />
                                <div class="form-actions">
                                    <el-button type="danger" :loading="accountSaving" @click="saveAccount">确认修改账号</el-button>
                                </div>
                            </div>
                        </transition>
                    </div>

                    <el-divider />

                    <!-- 修改邮箱 -->
                    <div class="security-block">
                        <div class="security-row">
                            <div class="security-info">
                                <div class="security-title">邮箱</div>
                                <div class="security-desc">当前邮箱：{{ user.email || '未设置' }}</div>
                            </div>
                            <el-button size="small" @click="emailVisible = !emailVisible">
                                {{ emailVisible ? '取消' : '修改' }}
                            </el-button>
                        </div>
                        <transition name="el-fade-in">
                            <div v-if="emailVisible" class="security-form">
                                <div class="inline-input">
                                    <el-input v-model="emailForm.email" placeholder="请输入新邮箱" />
                                    <el-button :loading="emailSending" :disabled="emailCountdown > 0" @click="sendEmailCode">
                                        {{ emailCountdown > 0 ? `${emailCountdown}秒后重试` : '发送验证码' }}
                                    </el-button>
                                </div>
                                <el-input v-model="emailForm.code" placeholder="请输入6位数字验证码" maxlength="6" class="mt-12" />
                                <div class="form-actions">
                                    <el-button type="primary" :loading="emailSaving" @click="saveEmail">修改邮箱</el-button>
                                </div>
                            </div>
                        </transition>
                    </div>

                    <el-divider />

                    <!-- 修改手机号 -->
                    <div class="security-block">
                        <div class="security-row">
                            <div class="security-info">
                                <div class="security-title">手机号</div>
                                <div class="security-desc">当前手机号：{{ user.phone || '未设置' }}</div>
                            </div>
                            <el-button size="small" @click="phoneVisible = !phoneVisible">
                                {{ phoneVisible ? '取消' : '修改' }}
                            </el-button>
                        </div>
                        <transition name="el-fade-in">
                            <div v-if="phoneVisible" class="security-form">
                                <div class="inline-input">
                                    <el-input v-model="phoneForm.phone" placeholder="请输入新手机号" maxlength="11" />
                                    <el-button :loading="phoneSending" :disabled="phoneCountdown > 0" @click="sendPhoneCode">
                                        {{ phoneCountdown > 0 ? `${phoneCountdown}秒后重试` : '发送验证码' }}
                                    </el-button>
                                </div>
                                <el-input v-model="phoneForm.code" placeholder="请输入6位数字验证码" maxlength="6" class="mt-12" />
                                <div class="form-actions">
                                    <el-button type="primary" :loading="phoneSaving" @click="savePhone">修改手机号</el-button>
                                </div>
                            </div>
                        </transition>
                    </div>

                    <el-divider />

                    <!-- 重置密码 -->
                    <div class="security-block">
                        <div class="security-row">
                            <div class="security-info">
                                <div class="security-title">登录密码</div>
                                <div class="security-desc">建议定期更换密码以保障账号安全</div>
                            </div>
                            <el-button size="small" @click="passwordVisible = !passwordVisible">
                                {{ passwordVisible ? '取消' : '重置' }}
                            </el-button>
                        </div>
                        <transition name="el-fade-in">
                            <div v-if="passwordVisible" class="security-form">
                                <el-input v-model="passwordForm.contact" placeholder="请输入邮箱或手机号" />
                                <div class="inline-input mt-12">
                                    <el-input v-model="passwordForm.code" placeholder="请输入验证码" />
                                    <el-button :loading="passwordSending" :disabled="passwordCountdown > 0" @click="sendPasswordCode">
                                        {{ passwordCountdown > 0 ? `${passwordCountdown}秒后重试` : '发送验证码' }}
                                    </el-button>
                                </div>
                                <el-input v-model="passwordForm.password" type="password" placeholder="请输入新密码（6-20位）" maxlength="20" class="mt-12" show-password />
                                <el-input v-model="passwordForm.verifyPwd" type="password" placeholder="请再次输入新密码" maxlength="20" class="mt-12" show-password />
                                <div class="form-actions">
                                    <el-button type="primary" :loading="passwordSaving" @click="savePassword">重置密码</el-button>
                                </div>
                            </div>
                        </transition>
                    </div>
                </el-card>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import axios from '{src}/utils/request'
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import { useCommStore } from '{src}/store/comm'

const store = useCommStore()

// ====================== 当前用户 ======================
const user = computed(() => store.getLogin.user || {})

// ====================== 基本资料 ======================
const basicForm = reactive({
    avatar: '',
    nickname: '',
    gender: '',
    title: '',
    description: '',
})
const basicOriginal = reactive({})
const basicSaving = ref(false)
const uploading = ref(false)

// ====================== 账号 ======================
const accountVisible = ref(false)
const accountForm = reactive({ account: '' })
const accountSaving = ref(false)

// ====================== 邮箱 ======================
const emailVisible = ref(false)
const emailForm = reactive({ email: '', code: '' })
const emailSaving = ref(false)
const emailSending = ref(false)
const emailCountdown = ref(0)
let emailTimer = null

// ====================== 手机号 ======================
const phoneVisible = ref(false)
const phoneForm = reactive({ phone: '', code: '' })
const phoneSaving = ref(false)
const phoneSending = ref(false)
const phoneCountdown = ref(0)
let phoneTimer = null

// ====================== 重置密码 ======================
const passwordVisible = ref(false)
const passwordForm = reactive({ contact: '', code: '', password: '', verifyPwd: '' })
const passwordSaving = ref(false)
const passwordSending = ref(false)
const passwordCountdown = ref(0)
let passwordTimer = null

// ====================== 工具 ======================
const formatTime = (timestamp) => {
    if (!timestamp) return '-'
    const t = Number(timestamp)
    if (!t) return '-'
    const date = new Date(t > 100000000000 ? t : t * 1000)
    const pad = n => String(n).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// 刷新本地缓存的用户信息（admin 无 checkLoginState，手动更新 cache + store）
const refreshUser = async () => {
    const { code, data } = await axios.post('/api/comm/check-token')
    if (code === 200 && data?.user) {
        store.login.user = data.user
        const validSeconds = Number(data.valid_time) > 0 ? Number(data.valid_time) : 15 * 24 * 60 * 60
        cache.set('user-info', data.user, Math.ceil(validSeconds / 60))
    }
}

const startCountdown = (type) => {
    const map = { email: 'emailCountdown', phone: 'phoneCountdown', password: 'passwordCountdown' }
    const key = map[type]
    const timerKey = { email: 'emailTimer', phone: 'phoneTimer', password: 'passwordTimer' }[type]
    // 重置倒计时
    if (type === 'email') { emailCountdown.value = 60; if (emailTimer) clearInterval(emailTimer); emailTimer = setInterval(() => { if (emailCountdown.value > 0) emailCountdown.value--; else clearInterval(emailTimer) }, 1000) }
    if (type === 'phone') { phoneCountdown.value = 60; if (phoneTimer) clearInterval(phoneTimer); phoneTimer = setInterval(() => { if (phoneCountdown.value > 0) phoneCountdown.value--; else clearInterval(phoneTimer) }, 1000) }
    if (type === 'password') { passwordCountdown.value = 60; if (passwordTimer) clearInterval(passwordTimer); passwordTimer = setInterval(() => { if (passwordCountdown.value > 0) passwordCountdown.value--; else clearInterval(passwordTimer) }, 1000) }
}

// ====================== 头像上传 ======================
const uploadAvatar = async () => {
    if (uploading.value) return
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/*'
    input.onchange = async () => {
        const file = input.files?.[0]
        if (!file) return
        if (file.size > 10 * 1024 * 1024) return ElMessage.warning('图片大小不能超过 10MB')
        if (!['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)) {
            return ElMessage.warning('请选择 JPG、PNG、GIF 或 WebP 格式的图片')
        }
        uploading.value = true
        try {
            const { code: checkCode, msg: checkMsg } = await axios.post('/api/attachment/checkType', { file_names: [file.name] })
            if (checkCode !== 200) return ElMessage.error(checkMsg || '文件类型检查失败')

            const params = new FormData()
            params.append('file', file)
            const { code, msg, data } = await axios.post('/api/attachment/batch', params)
            if (code !== 200) throw new Error(msg || '上传失败')
            basicForm.avatar = data.results?.[0]?.full_url || ''
            ElMessage.success('头像上传成功，请点击「保存修改」完成更新')
        } catch (e) {
            ElMessage.error(e.message || '上传失败')
        } finally {
            uploading.value = false
        }
    }
    input.click()
}

// ====================== 基本资料保存 ======================
const saveBasic = async () => {
    if (basicSaving.value) return
    if (utils.is.empty(basicForm.nickname)) return ElMessage.warning('昵称不能为空')
    basicSaving.value = true
    try {
        const payload = {
            id: user.value.id,
            nickname: basicForm.nickname,
            description: basicForm.description,
            avatar: basicForm.avatar,
            title: basicForm.title,
        }
        if (basicForm.gender) payload.gender = basicForm.gender
        const { code, msg } = await axios.put('/api/users/update', payload)
        if (code !== 200) return ElMessage.error(msg || '保存失败')
        ElMessage.success('基本资料已保存')
        await refreshUser()
        Object.assign(basicOriginal, { ...basicForm })
    } catch (e) {
        ElMessage.error(e.message || '保存失败，请重试')
    } finally {
        basicSaving.value = false
    }
}

const resetBasic = () => {
    Object.assign(basicForm, { ...basicOriginal })
}

// ====================== 账号修改 ======================
const saveAccount = async () => {
    if (accountSaving.value) return
    const val = (accountForm.account || '').trim()
    if (!val) return ElMessage.warning('请输入新账号')
    if (!/^[a-zA-Z0-9_]{4,20}$/.test(val)) return ElMessage.warning('账号只能包含字母、数字和下划线，长度 4-20 位')
    if (val === user.value.account) return ElMessage.warning('新账号不能与当前账号相同')
    accountSaving.value = true
    try {
        const { code, msg } = await axios.put('/api/users/update', { id: user.value.id, account: val })
        if (code !== 200) return ElMessage.error(msg || '修改失败')
        ElMessage.success('账号修改成功')
        await refreshUser()
        accountVisible.value = false
        accountForm.account = ''
    } catch (e) {
        ElMessage.error(e.message || '修改失败，请重试')
    } finally {
        accountSaving.value = false
    }
}

// ====================== 邮箱修改 ======================
const sendEmailCode = async () => {
    const email = (emailForm.email || '').trim()
    if (!email) return ElMessage.warning('请输入邮箱')
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) return ElMessage.warning('请输入有效的邮箱地址')
    emailSending.value = true
    try {
        const { code, msg } = await axios.put('/api/users/email', { email })
        if (code !== 200 && code !== 201) return ElMessage.error(msg || '发送验证码失败')
        ElMessage.success(msg || '验证码已发送，请查收')
        startCountdown('email')
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        emailSending.value = false
    }
}

const saveEmail = async () => {
    const email = (emailForm.email || '').trim()
    if (!email) return ElMessage.warning('请输入邮箱')
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) return ElMessage.warning('请输入有效的邮箱地址')
    if (!/^\d{6}$/.test(emailForm.code || '')) return ElMessage.warning('请输入6位数字验证码')
    emailSaving.value = true
    try {
        const { code, msg } = await axios.put('/api/users/email', { email, code: emailForm.code })
        if (code !== 200) return ElMessage.error(msg || '邮箱修改失败')
        ElMessage.success('邮箱修改成功')
        await refreshUser()
        emailVisible.value = false
        emailForm.email = ''
        emailForm.code = ''
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        emailSaving.value = false
    }
}

// ====================== 手机号修改 ======================
const sendPhoneCode = async () => {
    const phone = (phoneForm.phone || '').trim()
    if (!phone) return ElMessage.warning('请输入手机号')
    if (!/^1[3-9]\d{9}$/.test(phone)) return ElMessage.warning('请输入有效的手机号')
    phoneSending.value = true
    try {
        const { code, msg } = await axios.put('/api/users/phone', { phone })
        if (code !== 200 && code !== 201) return ElMessage.error(msg || '发送验证码失败')
        ElMessage.success(msg || '验证码已发送，请查收')
        startCountdown('phone')
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        phoneSending.value = false
    }
}

const savePhone = async () => {
    const phone = (phoneForm.phone || '').trim()
    if (!phone) return ElMessage.warning('请输入手机号')
    if (!/^1[3-9]\d{9}$/.test(phone)) return ElMessage.warning('请输入有效的手机号')
    if (!/^\d{6}$/.test(phoneForm.code || '')) return ElMessage.warning('请输入6位数字验证码')
    phoneSaving.value = true
    try {
        const { code, msg } = await axios.put('/api/users/phone', { phone, code: phoneForm.code })
        if (code !== 200) return ElMessage.error(msg || '手机号修改失败')
        ElMessage.success('手机号修改成功')
        await refreshUser()
        phoneVisible.value = false
        phoneForm.phone = ''
        phoneForm.code = ''
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        phoneSaving.value = false
    }
}

// ====================== 重置密码 ======================
const sendPasswordCode = async () => {
    const contact = (passwordForm.contact || '').trim()
    if (!contact) return ElMessage.warning('请输入邮箱或手机号')
    const isPhone = /^1[3-9]\d{9}$/.test(contact)
    const isEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contact)
    if (!isPhone && !isEmail) return ElMessage.warning('请输入正确的邮箱或手机号')
    passwordSending.value = true
    try {
        const { code, msg } = await axios.post('/api/comm/reset-password', { contact })
        if (code !== 200 && code !== 201) return ElMessage.error(msg || '发送验证码失败')
        ElMessage.success(msg || '验证码已发送，请查收')
        startCountdown('password')
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        passwordSending.value = false
    }
}

const savePassword = async () => {
    const { contact, code, password, verifyPwd } = passwordForm
    if (!contact) return ElMessage.warning('请输入邮箱或手机号')
    if (!code) return ElMessage.warning('请输入验证码')
    if (!password) return ElMessage.warning('请输入新密码')
    if (password.length < 6) return ElMessage.warning('密码长度不能少于6位')
    if (password !== verifyPwd) return ElMessage.warning('两次输入的密码不一致')
    passwordSaving.value = true
    try {
        const { code: resCode, msg } = await axios.post('/api/comm/reset-password', {
            social: contact,
            code,
            password,
        })
        if (resCode !== 200) return ElMessage.error(msg || '重置密码失败')
        ElMessage.success('密码修改成功')
        passwordVisible.value = false
        Object.assign(passwordForm, { contact: '', code: '', password: '', verifyPwd: '' })
    } catch (e) {
        ElMessage.error(e.message || '网络错误，请稍后重试')
    } finally {
        passwordSaving.value = false
    }
}

// ====================== 初始化 ======================
const init = () => {
    const u = user.value || {}
    basicForm.avatar = u.avatar || ''
    basicForm.nickname = u.nickname || ''
    basicForm.gender = u.gender || ''
    basicForm.title = u.title || ''
    basicForm.description = u.description || ''
    Object.assign(basicOriginal, { ...basicForm })
}

// 监听用户信息变化（check-token 异步返回后回填表单）
watch(user, (newUser) => {
    if (newUser && Object.keys(newUser).length > 0) init()
}, { immediate: true })

onMounted(() => {
    init()
})

onUnmounted(() => {
    [emailTimer, phoneTimer, passwordTimer].forEach(t => t && clearInterval(t))
})
</script>

<style lang="scss" scoped>
.profile-page {
    max-width: 1100px;
    margin: 0 auto;
}

.profile-layout {
    display: flex;
    gap: 16px;
    align-items: flex-start;
}

.profile-card {
    width: 280px;
    flex-shrink: 0;
    position: sticky;
    top: 16px;
}

.user-card {
    border-radius: 4px;
}

.user-card-body {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 8px 0;
}

.user-avatar {
    background: #1890ff;
    color: #fff;
    font-size: 32px;
    margin-bottom: 12px;
}

.user-name {
    font-size: 18px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.85);
}

.user-account {
    font-size: 13px;
    color: rgba(0, 0, 0, 0.45);
    margin: 4px 0 8px;
}

.user-title-tag {
    margin-bottom: 4px;
}

.user-meta {
    width: 100%;
    text-align: left;
}

.meta-item {
    display: flex;
    justify-content: space-between;
    padding: 6px 0;
    font-size: 13px;
}

.meta-label {
    color: rgba(0, 0, 0, 0.45);
}

.meta-value {
    color: rgba(0, 0, 0, 0.85);
}

.profile-settings {
    flex: 1;
    min-width: 0;
}

.settings-card {
    border-radius: 4px;
    margin-bottom: 16px;
}

.card-header {
    font-weight: 600;
}

.card-title {
    font-size: 15px;
    color: rgba(0, 0, 0, 0.85);
}

.avatar-editor {
    display: flex;
    align-items: center;
    gap: 16px;
}

.avatar-editor-preview {
    background: #1890ff;
    color: #fff;
    font-size: 24px;
}

.avatar-editor-actions {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.security-block {
    padding: 4px 0;
}

.security-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.security-info {
    flex: 1;
}

.security-title {
    font-size: 14px;
    font-weight: 500;
    color: rgba(0, 0, 0, 0.85);
}

.security-desc {
    font-size: 13px;
    color: rgba(0, 0, 0, 0.45);
    margin-top: 4px;
}

.security-form {
    margin-top: 12px;
    padding: 16px;
    background: #fafafa;
    border-radius: 4px;
}

.inline-input {
    display: flex;
    gap: 8px;
}

.inline-input .el-input {
    flex: 1;
}

.form-actions {
    margin-top: 12px;
    text-align: right;
}

.mt-12 {
    margin-top: 12px;
}

.mb-12 {
    margin-bottom: 12px;
}

@media (max-width: 768px) {
    .profile-layout {
        flex-direction: column;
    }

    .profile-card {
        width: 100%;
        position: static;
    }
}
</style>
