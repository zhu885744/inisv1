<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">推荐开启，有利于减少数据库和服务器的负担！</strong><br>
                        开启后会对API数据进行缓存，减少重复执行数据库操作以及对数据的运算，<br>
                        从而提高API的响应速度，减少服务器的负担。<br>
                        PS：缓存数据会存储在专门的缓存数据库Redis中。
                    </template>
                    <span style="font-weight: 600">Redis 缓存</span>
                </el-tooltip>
                <el-tag size="small" type="danger">Redis</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="redis" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">Redis缓存</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">使用高性能Redis缓存服务</div>
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
            <strong class="flex-center">配置 Redis 缓存服务</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="主机">
                    <el-input v-model="state.struct.host" placeholder="localhost"></el-input>
                </el-form-item>
                <el-form-item label="端口">
                    <el-input-number v-model="state.struct.port" :min="1" :max="65535"></el-input-number>
                </el-form-item>
                <el-form-item label="数据库">
                    <el-select v-model="state.struct.database">
                        <el-option v-for="(_, index) in 16" :key="index" :label="index" :value="index">
                            <span>{{ index }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="密码">
                    <el-input v-model="state.struct.password" show-password placeholder="无密码为空"></el-input>
                </el-form-item>
                <el-form-item label="过期时间">
                    <el-input v-model="state.struct.expire" placeholder="2 * 60 * 60"></el-input>
                </el-form-item>
                <el-form-item label="前缀">
                    <el-input v-model="state.struct.prefix" placeholder="inis:"></el-input>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.status.dialog = false">取 消</el-button>
            <el-button v-on:click="method.test()" :loading="state.status.test">测试连接</el-button>
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
        host:     'localhost',
        port:     6379,
        database: 0,
        password: '',
        prefix:   'inis:',
        expire:   '2 * 60 * 60',
    },
    status: {
        finish: false,
        active: false,
        dialog: false,
        loading: true,
        test: false,
    },
    backup: {}
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/toml/cache', {
            name: 'redis'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        // 拷贝一份备份
        state.backup = JSON.parse(JSON.stringify(data))

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        const { code, msg } = await axios.put('/api/toml/cache-default', {
            value: 'redis', open: value
        })

        if (code === 200) return emit('refresh', 'cache-file', 'cache-ram')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        let field = ['host', 'port', 'database', 'password']

        // 密码已脱敏隐藏时，跳过「是否有变化」检查
        const secretChanged = !utils.is.masked(state.struct.password)

        // 检查关键配置是否有变化
        if (secretChanged && !utils.object.equal(state.struct, state.backup, field)) return ElMessage.warning('请先完成测试连接')

        if (utils.is.empty(state.struct.host))      return ElMessage.warning('请填写 主机地址！')
        if (utils.is.empty(state.struct.port))      return ElMessage.warning('请填写 端口号！')
        if (utils.is.empty(state.struct.database))  return ElMessage.warning('请选择 数据库！')

        state.status.wait   = true

        // 剔除脱敏占位字段，由后端保留原值
        const data = utils.object.withoutMasked(state.struct, ['password'])
        const { code, msg } = await axios.put('/api/toml/cache-redis', data)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)
        
        ElMessage.success('保存成功')
        state.status.dialog = false
    },
    test: async () => {

        if (utils.is.empty(state.struct.host))      return ElMessage.warning('请填写 主机地址！')
        if (utils.is.empty(state.struct.port))      return ElMessage.warning('请填写 端口号！')
        if (utils.is.empty(state.struct.database))  return ElMessage.warning('请选择 数据库！')

        state.status.test = true

        // 密码脱敏时，测试请求剔除该字段（后端保留原密码进行连接测试）
        const data = utils.object.withoutMasked(state.struct, ['password'])
        const { code, msg, data: resp } = await axios.post('/api/toml/test-redis', data)

        state.status.test = false

        if (code === 200) {
            // 拷贝一份备份
            state.backup = JSON.parse(JSON.stringify(state.struct))
            return ElMessage.success(msg)
        }

        ElMessage.error(`${msg}<br>${resp}`)
    },
}

watch(() => state.struct, () => {
    state.status.active = state.struct.default === 'redis' && state.struct.open
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
