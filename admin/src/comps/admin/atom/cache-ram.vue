<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">推荐开启，有利于减少数据库和服务器的负担！</strong><br>
                        开启后会对API数据进行缓存，减少重复执行数据库操作以及对数据的运算，<br>
                        从而提高API的响应速度，减少服务器的负担。<br>
                        PS：缓存数据会通过内存的方式存储在服务器的内存中。
                    </template>
                    <span style="font-weight: 600">内存缓存</span>
                </el-tooltip>
                <el-tag size="small" type="warning">内存</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="ram" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">内存缓存</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">将数据缓存到服务器运行内存</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
                    <el-tooltip content="请不要作死，内存很宝贵，不要用在这种地方，除非你是本地环境或者不怕死" placement="top">
                        <el-switch v-model="state.status.active" v-on:change="method.change" disabled
                                   active-text="开启" inactive-text="关闭">
                        </el-switch>
                    </el-tooltip>
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
            <strong class="flex-center">配置 内存 缓存服务</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="过期时间">
                    <el-input v-model="state.struct.expire" placeholder="2 * 60 * 60"></el-input>
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
const emit  = defineEmits(['refresh'])
const state = reactive({
    struct: {
        open:     false,
        default:  null,
        expire:   '2 * 60 * 60',
    },
    status: {
        finish: false,
        active: false,
        dialog: false,
        loading: true,
        wait: false,
    },
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/toml/cache', {
            name: 'ram'
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

        const { code, msg } = await axios.put('/api/toml/cache-default', {
            value: 'ram', open: value
        })

        if (code === 200) return emit('refresh', 'cache-redis', 'cache-file')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        if (utils.is.empty(state.struct.expire)) return ElMessage.warning('请填写 过期时间！')

        state.status.wait   = true

        const { code, msg } = await axios.put('/api/toml/cache-ram', state.struct)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)
        
        ElMessage.success('保存成功')
        state.status.dialog = false
    },
}

watch(() => state.struct, () => {
    state.status.active = state.struct.default === 'ram' && state.struct.open
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
