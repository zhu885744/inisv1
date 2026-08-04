<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">限制分页查询单次最大数据量，开启让系统毫无瑕疵！</strong><br>
                        分页限制可以限制坏人发起恶意请求，如恶意请求 999999 乃至无限大的数据，<br>
                        导致服务器崩溃，分页限制可以限制每页最大请求数据量，超过则强制性=最大值<br>
                        注意：如果开启了缓存，此项可不开，因为缓存数据不会导致重复查询
                    </template>
                    <span style="font-weight: 600">分页限制</span>
                </el-tooltip>
                <el-tag size="small" type="warning">限制</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="lock" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">分页限制</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">限制列表分页的最大查询数量</div>
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
                <el-form-item label="最大值">
                    <el-input-number v-model="state.struct.text" :min="5"></el-input-number>
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
        text: 10
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
            key: 'SYSTEM_PAGE_LIMIT'
        })
        if (code !== 200) return
        state.struct = data

        state.status.finish  = true
        state.status.loading = false
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('分页限制配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_PAGE_LIMIT',
            value: value ? 1 : 0
        })

        if (code === 200) return

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        state.status.wait   = true

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'SYSTEM_PAGE_LIMIT',
            text: state.struct.text
        })

        state.status.wait   = false

        if (code !== 200) return ElMessage.error(`保存失败：${msg}`)
        
        ElMessage.success('保存成功')
        state.status.dialog = false
    }
}

watch(() => state.struct, () => {
    state.status.active = parseInt(state.struct.value) === 1
    // 限制 state.struct.text 最小值为 5
    if (state.struct.text < 5)  state.struct.text = 5
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>