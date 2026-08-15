<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        通知保留天数：已读通知与广播通知超过该天数后，<br>
                        将在每日凌晨由定时任务自动清理，控制通知表的数据量增长。
                    </template>
                    <span style="font-weight: 600">通知清理</span>
                </el-tooltip>
                <el-tag size="small" type="warning">定时</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="bell" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">通知保留天数</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">当前保留 {{ state.struct.retention_days }} 天</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
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
            <strong class="flex-center">配置通知保留天数</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="保留天数">
                    <el-input-number v-model="state.struct.retention_days" :min="1" :max="3650"></el-input-number>
                </el-form-item>
            </el-form>
            <el-alert type="info" :closable="false" show-icon style="margin-top: 4px">
                已读通知与广播通知超过该天数后，将在每日凌晨自动清理。
            </el-alert>
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
const state = reactive({
    struct: {
        retention_days: 30
    },
    status: {
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

        const { code, data } = await axios.get('/api/toml/notification')

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        state.status.finish = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('通知配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    save: async () => {

        state.status.wait = true

        const { code, msg } = await axios.put('/api/toml/notification', state.struct)

        state.status.wait = false

        if (code !== 200) return ElMessage.error(`保存失败：${msg}`)

        ElMessage.success('保存成功')
        state.status.dialog = false
    }
}

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
