<template>
    <div class="login-page">
        <div class="login-container">
            <div class="login-card">
                <div class="login-header">
                    <h2 class="login-title">找回密码</h2>
                </div>

                <el-form class="login-form" @submit.prevent="method.reset()">
                    <el-form-item>
                        <el-input
                            v-model="state.struct.social"
                            placeholder="邮箱或手机号"
                            size="large"
                            clearable
                        >
                            <template #prefix>
                                <el-icon><Message /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.struct.code"
                            placeholder="请输入验证码"
                            size="large"
                            clearable
                        >
                            <template #prefix>
                                <el-icon><Key /></el-icon>
                            </template>
                            <template #append>
                                <el-button @click="method.code()" :loading="state.item.loading" :disabled="state.item.loading">
                                    {{ state.item.loading ? `${state.item.second}s` : '获取验证码' }}
                                </el-button>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.password.value"
                            type="password"
                            placeholder="请输入新密码（至少6位）"
                            size="large"
                            show-password
                            clearable
                        >
                            <template #prefix>
                                <el-icon><Lock /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.password.verify"
                            type="password"
                            placeholder="请再次输入新密码"
                            size="large"
                            show-password
                            clearable
                        >
                            <template #prefix>
                                <el-icon><Lock /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-button
                            type="primary"
                            size="large"
                            :loading="state.item.wait"
                            class="login-btn"
                            @click="method.reset()"
                        >
                            {{ state.item.wait ? '重置中...' : '重置密码' }}
                        </el-button>
                    </el-form-item>
                </el-form>

                <div class="login-footer">
                    <el-button text @click="router.push('/')">
                        记起来了？点我登录
                    </el-button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'

const router = useRouter()

const state = reactive({
    item: {
        loading: false,
        wait: false,
        second: 0,
    },
    struct: {
        social: null,
        code: null,
    },
    password: {
        value: null,
        verify: null,
    },
    timer: null,
})

const method = {
    async reset() {
        const { social, code } = state.struct
        const { value: pwd, verify: pwdVerify } = state.password

        if (!social) return ElMessage.warning('请填写邮箱或手机号')
        if (!pwd) return ElMessage.warning('请输入新密码')
        if (pwd.length < 6) return ElMessage.warning('密码长度至少6位')
        if (!pwdVerify) return ElMessage.warning('请再次输入密码')
        if (!code) return ElMessage.warning('请输入验证码')
        if (pwd !== pwdVerify) return ElMessage.warning('两次密码不一致')

        const isPhone = /^1[3-9]\d{9}$/.test(social)
        const isEmail = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(social)
        if (!isPhone && !isEmail) return ElMessage.warning('请填写正确的手机号或邮箱')

        try {
            state.item.wait = true
            const { code: resCode, msg } = await axios.post('/api/comm/reset-password', {
                social,
                code,
                password: pwd
            })

            state.item.wait = false

            if (resCode !== 200) {
                ElMessage.error(msg || '重置密码失败')
                return
            }

            ElMessage.success('密码重置成功')
            router.push('/')

        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '网络异常，请稍后再试')
        }
    },

    async code() {
        const { social } = state.struct
        if (!social) return ElMessage.warning('请填写邮箱或手机号')

        const isPhone = /^1[3-9]\d{9}$/.test(social)
        const isEmail = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(social)
        if (!isPhone && !isEmail) return ElMessage.warning('请填写正确的手机号或邮箱')

        try {
            const { code: resCode, msg } = await axios.post('/api/comm/reset-password', { social })

            if (!utils.in.array(resCode, [200, 201])) {
                ElMessage.error(msg || '发送验证码失败')
                return
            }

            ElMessage.success(msg || '验证码发送成功')

            if (state.timer) clearInterval(state.timer)
            state.item.second = 60
            state.timer = setInterval(() => {
                state.item.second--
                if (state.item.second <= 0) {
                    clearInterval(state.timer)
                    state.timer = null
                    state.item.second = 0
                }
            }, 1000)

        } catch (error) {
            ElMessage.error(error.message || '网络异常，验证码发送失败')
        }
    },
}

watch(() => state.item.second, (val) => {
    state.item.loading = val > 0
})

onUnmounted(() => {
    if (state.timer) clearInterval(state.timer)
})
</script>

<style scoped>
.login-page {
    min-height: 100vh;
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    position: fixed;
    top: 0;
    left: 0;
    margin: 0;
    padding: 0;
}

.login-container {
    width: 100%;
    max-width: 400px;
    padding: 20px;
}

.login-card {
    background: #fff;
    border-radius: 12px;
    padding: 40px 30px;
    box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.login-header {
    text-align: center;
    margin-bottom: 30px;
}

.login-logo {
    margin-bottom: 15px;
}

.login-title {
    font-size: 20px;
    color: #333;
    margin: 0;
}

.login-form {
    margin-bottom: 20px;
}

.login-btn {
    width: 100%;
}

.login-footer {
    text-align: center;
    padding-top: 15px;
}
</style>