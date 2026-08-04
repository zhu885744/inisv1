<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-danger)">风险操作！此功能不懂请勿开启！</strong><br>
                        ● 开启之后，所有的API均需要在请求头中提交 <strong style="color: var(--el-color-danger)">i-api-key=密钥</strong> 方能使用！<br>
                        ● 于此同时API安全性将提升90%，剩下10%取决于你的密钥复杂度和对手的能力强弱！
                    </template>
                    <span style="font-weight: 600">API KEY</span>
                </el-tooltip>
                <el-tag size="small" type="danger">密钥</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="key" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">API密钥</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">对外接口访问的身份凭证</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
                    <el-switch v-model="state.status.active" v-on:change="method.change" :disabled="!state.status.finish"
                               active-text="开启" inactive-text="关闭">
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
            <strong class="flex-center">配置</strong>
        </template>
        <template #default>
            <p style="margin-top: 0.25rem; margin-bottom: 0.25rem">● API KEY 不是什么很NB的技术，却能大大提高的接口安全</p>
            <p style="margin-top: 0.25rem; margin-bottom: 0.25rem">● 正常来说不开启也没关系，因为除此之外还有QPS限流器在帮你拦截异常流量</p>
            <p style="margin-top: 0.25rem; margin-bottom: 0.25rem">● 但是如果您开启了API KEY，在使用其它主题的时候，需要按照要求配置密钥到您的主题中，否则主题会拿不到任何数据</p>
        </template>
        <template #footer>
            <el-button v-on:click="state.status.dialog = false">
                取 消
            </el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import axios from '{src}/utils/request.js'

const { ctx, proxy } = getCurrentInstance()
const state = reactive({
    struct: {},
    status: {
        finish: false,
        active: false,
        dialog: false,
        loading: true,
    }
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/config/one', {
            key: 'SYSTEM_API_KEY'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('API KEY配置获取失败，无法进行配置！') 
        state.status.dialog = true
    },
    change: async value => {

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_API_KEY',
            value: value ? 1 : 0
        })

        if (code === 200) return

        state.status.active = !value
        ElMessage.error(msg)
    },
}

watch(() => state.struct, () => {
    state.status.active = parseInt(state.struct.value) === 1
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
