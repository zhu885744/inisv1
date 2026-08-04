<template>
    <div class="login-page">
        <div class="login-container">
            <div class="login-card">
                <div class="login-header">
                    <h2 class="login-title">注册账号</h2>
                </div>

                <el-form class="login-form" @submit.prevent="method.register()">
                    <el-form-item>
                        <el-input
                            v-model="state.struct.nickname"
                            placeholder="请输入昵称"
                            size="large"
                        >
                            <template #prefix>
                                <el-icon><UserFilled /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.struct.account"
                            placeholder="请输入账号"
                            size="large"
                        >
                            <template #prefix>
                                <el-icon><User /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.struct.social"
                            placeholder="邮箱或手机号"
                            size="large"
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
                            placeholder="请输入密码"
                            size="large"
                            show-password
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
                            placeholder="请再次输入密码"
                            size="large"
                            show-password
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
                            @click="method.register()"
                        >
                            {{ state.item.wait ? '注册中...' : '注 册' }}
                        </el-button>
                    </el-form-item>
                </el-form>

                <div class="login-footer">
                    <el-button text @click="router.push('/')">
                        已有账号？点我登录
                    </el-button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import cache from '{src}/utils/cache.js'
import { useCommStore } from '{src}/store/comm'

const router = useRouter()
const store = { comm: useCommStore() }

const state = reactive({
    item: {
        loading: false,
        wait: false,
        second: 0,
    },
    struct: {
        social: null,
        account: null,
        nickname: null,
        code: null,
    },
    password: {
        value: null,
        verify: null,
    },
    timer: null,
})

const method = {
    async register() {
        if (!state.struct.nickname) return ElMessage.warning('请填写昵称')
        if (!state.struct.account) return ElMessage.warning('请输入账号')
        if (!state.struct.social) return ElMessage.warning('请输入邮箱或手机号')
        if (!state.password.value) return ElMessage.warning('请输入密码')
        if (!state.password.verify) return ElMessage.warning('请再次输入密码')
        if (!state.struct.code) return ElMessage.warning('请输入验证码')
        if (state.password.value !== state.password.verify) return ElMessage.warning('两次密码不一致')

        state.item.wait = true

        try {
            const { code, msg, data } = await axios.post('/api/comm/register', {
                ...state.struct,
                password: state.password.value
            })

            state.item.wait = false

            if (code !== 200) return ElMessage.error(msg)

            ElMessage.success(`注册成功！欢迎您，${state.struct.nickname}！`)
            cache.set('user-info', data.user, 7 * 24 * 60)
            utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', data.token, 7 * 24 * 60 * 60)
            store.comm.login.finish = true
            store.comm.login.user = data.user
            router.push('/admin')

        } catch (error) {
            state.item.wait = false
            ElMessage.error('网络异常，请稍后再试')
        }
    },

    async code() {
        if (!state.struct.social) return ElMessage.warning('请输入邮箱或手机号')

        const social = state.struct.social
        const isPhone = /^1[3-9]\d{9}$/.test(social)
        const isEmail = /^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(social)
        if (!isPhone && !isEmail) return ElMessage.warning('请填写正确的手机号或邮箱')

        try {
            const { code, msg } = await axios.post('/api/comm/register', {
                social: state.struct.social,
            })

            if (!utils.in.array(code, [200, 201])) return ElMessage.error(msg)

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
            ElMessage.error('网络异常，验证码发送失败')
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