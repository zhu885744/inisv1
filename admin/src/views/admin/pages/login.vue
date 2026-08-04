<template>
    <div class="login-page">
        <div class="login-container">
            <div class="login-card">
                <div class="login-header">
                    <h2 class="login-title">管理后台登录</h2>
                </div>

                <el-form class="login-form" @submit.prevent="method.login()">
                    <el-form-item>
                        <el-input
                            v-model="state.struct.account"
                            placeholder="帐号 | 邮箱 | 手机号"
                            size="large"
                        >
                            <template #prefix>
                                <el-icon><User /></el-icon>
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item>
                        <el-input
                            v-model="state.struct.password"
                            type="password"
                            placeholder="请输入密码"
                            size="large"
                            show-password
                            @keyup.enter="method.login()"
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
                            @click="method.login()"
                        >
                            {{ state.item.wait ? '登录中...' : '登 录' }}
                        </el-button>
                    </el-form-item>
                </el-form>

                <div class="login-footer">
                    <el-button text @click="router.push('/register')">
                        注册账号
                    </el-button>
                    <el-button text @click="router.push('/reset-password')">
                        忘记密码？
                    </el-button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import cache from '{src}/utils/cache.js'
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import crypto from '{src}/utils/crypto.js'
import { useCommStore } from '{src}/store/comm'
import { useConfigStore } from '{src}/store/config'

const router = useRouter()
const store = {
    comm: useCommStore(),
    config: useConfigStore()
}

const state = reactive({
    item: {
        wait: false,
    },
    struct: {
        account: null,
        password: null,
    },
})

const method = {
    async login() {
        if (!state.struct.account) return ElMessage.warning('请输入账号')
        if (!state.struct.password) return ElMessage.warning('请输入密码')

        state.item.wait = true

        const unix = await method.unix()
        const iv = crypto.token(`iv-${unix}`, 16, 'login')
        const key = crypto.token(`key-${unix}`, 16, 'login')
        const AES = crypto.AES(key, iv)

        const params = {
            account: AES.encrypt(state.struct.account),
            password: AES.encrypt(state.struct.password)
        }

        try {
            const { data, code, msg } = await axios.post('/api/comm/login', params, {
                headers: {
                    'X-Khronos': unix,
                    'X-Gorgon': `${key} ${iv}`,
                    'X-Argus': AES.encrypt(JSON.stringify({
                        unix, account: state.struct.account, password: state.struct.password
                    }))
                }
            })

            state.item.wait = false

            if (code === 200) {
                const userData = data?.user || {}
                const userStatus = Number(userData.status) || 0
                if (userStatus === 1) {
                    method.clearCache()
                    ElMessage.error('当前账号已被冻结，请联系管理员！')
                    return
                }

                cache.set('user-info', data.user, 7 * 24 * 60)
                utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', data.token, 7 * 24 * 60 * 60)
                store.comm.login.finish = true
                store.comm.login.user = data.user
                ElMessage.success('登录成功')
                router.push('/admin')
                return
            }

            ElMessage.error(msg)
            method.clearCache()

        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '登录失败')
            method.clearCache()
        }
    },

    clearCache() {
        utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', '', -1)
    },

    async unix() {
        const { code, data } = await axios.get('/dev/info/time')
        if (code !== 200) return Math.round(new Date() / 1000)
        return data.unix
    },
}
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
    display: flex;
    justify-content: space-between;
    padding-top: 15px;
}
</style>