<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 本地存储是指将图片等资源文件存储到服务器本地，然后通过访问服务器的方式获取资源文件<br>
                        ● 本地存储的优点是不花钱，缺点是占用服务器空间和带宽
                    </template>
                    <span style="font-weight: 600">本地存储</span>
                </el-tooltip>
                <el-tag size="small" type="info">传统</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="save" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">本地存储</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">将资源文件存储到服务器本地磁盘</div>
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
            <strong style="display: flex; align-items: center; justify-content: center">配置本地存储</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="域名">
                    <el-input v-model="state.struct.domain" placeholder="请输入域名"></el-input>
                </el-form-item>
                <el-form-item label="存储目录">
                    <el-input v-model="state.struct.path" placeholder="storage"></el-input>
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
import axios from '{src}/utils/request.js'

const { ctx, proxy } = getCurrentInstance()
const emit  = defineEmits(['refresh'])
const state = reactive({
    struct: {
        default: null,
        domain: null,
        path: 'storage',
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

        const { code, data } = await axios.get('/api/toml/storage', {
            name: 'local'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        if (!value) return state.status.active = true

        const { code, msg } = await axios.put('/api/toml/storage-default', {
            value: value ? 'local' : null
        })

        if (code === 200) return emit('refresh')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        state.status.wait   = true

        const { code, msg } = await axios.put('/api/toml/storage-local', state.struct)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        ElMessage.success('保存成功')
        state.status.dialog = false
    },
}

watch(() => state.struct, () => {
    state.status.active = state.struct.default === 'local'
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
