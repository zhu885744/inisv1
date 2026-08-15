<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        JWT（JSON Web Token）是一种在网络应用中传递声明信息的轻量级、安全的方式。<br>
                        JWT具有通用性和可扩展性，可以应用在很多场景，比如用户认证、单点登录、API访问授权等。
                    </template>
                    <span style="font-weight: 600">JWT</span>
                </el-tooltip>
                <el-tag size="small" type="primary">+5%</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="key" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">JWT认证</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">JSON Web Token 用户认证服务</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
                    <el-switch v-model="state.status.active" v-on:change="method.change" :disabled="!state.status.finish"
                               active-text="开始" inactive-text="关闭">
                    </el-switch>
                    <el-button text type="primary" v-on:click="method.show()">
                        配置
                        <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
                    </el-button>
                </div>
            </div>
        </template>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">配置 JSON Web Token</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="签发者">
                    <el-input v-model="state.struct.issuer" placeholder="请输入签发者"></el-input>
                </el-form-item>
                <el-form-item label="主题">
                    <el-input v-model="state.struct.subject" placeholder="请输入主题"></el-input>
                </el-form-item>
                <el-form-item label="密钥">
                    <el-input v-model="state.struct.key" placeholder="请输入密钥">
                        <template #append>
                            <el-button v-on:click="method.rand()">随机</el-button>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item label="过期时间">
                    <el-input v-model="state.struct.expire" placeholder="15 * 24 * 60 * 60"></el-input>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.status.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.status.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'

const { ctx, proxy } = getCurrentInstance()
const state = reactive({
    struct: {
        key    : null,
        issuer : null,
        subject: null,
        expire : null,
    },
    status: {
        active: true,
        finish: false,
        dialog: false,
        loading: true,
        wait: false,
    }
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/toml/crypt', {
            name: 'jwt'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('分页限制配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {
        if (!value) {
            state.status.active = true
            ElMessage.warning({
                message: 'JWT是基础服务，这可不能关',
                dangerouslyUseHTMLString: true
            })
        }
    },
    save: async () => {

        state.status.wait   = true

        // 密钥已脱敏隐藏时，不提交该字段，由后端保留原值
        const data = utils.object.withoutMasked(state.struct, ['key'])

        const { code, msg } = await axios.put('/api/toml/crypt-jwt', data)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error(`保存失败：${msg}`)
        
        ElMessage.success('保存成功')
        state.status.dialog = false
    },
    rand(field = 'key') {
        let result  = 'INIS-'
        const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
        const len   = chars.length
        for (let i  = 0; i < 32; i++) {
            result += chars.charAt(Math.floor(Math.random() * len))
        }
        state.struct[field] = result
    },
}

watch(() => state.struct, () => {
    // key 只允许输入字母、数字和全部的特殊字符
    state.struct.key    = state.struct.key.replace(/[^\w!@#$%^&*()_+\-=\[\]{};:'"\\|\/?,.<>~`\s]/g, '')
    // 只能是 数字、空格和运算符
    state.struct.expire = state.struct.expire.replace(/[^\d\s*+\-\/]/g, '')

}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>