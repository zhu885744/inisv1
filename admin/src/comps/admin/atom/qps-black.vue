<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">开启后满足条件的客户端IP会被自动拉黑！</strong><br>
                        ● 此功能需要搭配 QPS 使用，在 时间频率 内，达到 触发 QPS 次数上限，自动拉黑对应的客户端IP<br>
                        ● 如：在1小时内，某客户端IP触发了3次 QPS 警告，则自动拉黑此客户端IP，并且后续无法访问任何API<br>
                        ● 此功能可以有效防止恶意攻击，如：CC攻击、DDOS攻击等<br>
                        ● 需要注意的是，随机图片接口也会受到影响<br>
                    </template>
                    <span style="font-weight: 600">自动 IP 黑名单</span>
                </el-tooltip>
                <el-tag size="small" type="danger">黑名单</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="qps" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">QPS黑名单</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">限制特定IP的接口访问频率</div>
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
            <strong class="flex-center">配置</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="触发次数">
                    <el-input-number v-model="state.struct.json.count" :min="1"></el-input-number>
                </el-form-item>
                <el-form-item label="时间频率">
                    <el-input v-model="state.struct.json.second" placeholder="60 * 60"></el-input>
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
        json: { count: null, second: '' },
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
            key: 'SYSTEM_QPS_BLOCK'
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

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_QPS_BLOCK',
            value: value ? 1 : 0
        })

        if (code === 200) return

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        state.status.wait   = true

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_QPS_BLOCK',
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
    // 只能是 数字、空格和运算符
    state.struct.expire = state.struct.json.second.replace(/[^\d\s*+\-\/]/g, '')
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>