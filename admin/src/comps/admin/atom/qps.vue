<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">开启后让DDOS和CC等异常流量无懈可击！</strong><br>
                        ● QPS俗称限流器，是一种作用于网络层的中间件，主要用于拦截异常流量，如DDOS和CC等<br>
                        ● QPS会将每个IP的请求次数限制在每秒的指定次数，超过则拒绝访问，从而保护服务器安全
                    </template>
                    <span style="font-weight: 600">QPS</span>
                </el-tooltip>
                <el-tag size="small" type="warning">+30%</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="qps" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">QPS限制</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">限制接口每秒查询率</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
                    <el-switch v-model="state.status.active" v-on:change="method.change" :disabled="!state.status.finish"
                               active-text="我怂" inactive-text="无所畏惧">
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
            <strong style="display: flex; align-items: center; justify-content: center">配置</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="全局限制">
                    <el-input-number v-model="state.struct.json.global" :min="10"></el-input-number>
                </el-form-item>
                <el-form-item label="单接口限制">
                    <el-input-number v-model="state.struct.json.point" :min="5"></el-input-number>
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
const state = reactive({
    struct: {
        value: 0,
        json: { global: 30, point: 15 },
    },
    status: {
        finish: false,
        active: false,
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

        const { code, data } = await axios.get('/api/config/one', {
            key: 'SYSTEM_QPS'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('QPS配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_QPS',
            value: value ? 1 : 0
        })

        if (code === 200) return

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        state.status.wait   = true

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_QPS',
            json: JSON.stringify(state.struct.json)
        })

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
        ElMessage.success('保存成功')
    }
}

watch(() => state.struct, () => {
    state.status.active = parseInt(state.struct.value) === 1
    // 限制 state.struct.json.global 最小值为 10
    if (state.struct.json.global < 10) state.struct.json.global = 10
    // 限制 state.struct.json.point 最小值为 5
    if (state.struct.json.point < 5)  state.struct.json.point = 5
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>