<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 腾讯云对象存储COS可以替代传统的本地存储，有能力的情况推荐开启COS存储<br>
                        ● 开启后，后续上传的文件将会自动上传到COS，不会占用服务器的空间和带宽
                    </template>
                    <span style="font-weight: 600">腾讯云对象存储</span>
                </el-tooltip>
                <el-tag size="small" type="danger">COS</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="upload" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">腾讯云COS</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">将资源文件存储到腾讯云对象存储</div>
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
            <strong style="display: flex; align-items: center; justify-content: center">配置 腾讯云COS 存储</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="SecretId">
                    <el-input v-model="state.struct.secret_id" show-password placeholder="请输入 SecretId"></el-input>
                </el-form-item>
                <el-form-item label="SecretKey">
                    <el-input v-model="state.struct.secret_key" show-password placeholder="请输入 SecretKey"></el-input>
                </el-form-item>
                <el-form-item label="AppId">
                    <el-input v-model="state.struct.app_id" placeholder="请输入 AppId"></el-input>
                </el-form-item>
                <el-form-item label="COS Bucket">
                    <el-input v-model="state.struct.bucket" placeholder="请输入 COS Bucket"></el-input>
                </el-form-item>
                <el-form-item label="COS Region">
                    <el-select v-model="state.struct.region" placeholder="请选择所在地区">
                        <el-option v-for="item in state.select.region" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.value }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="存储目录">
                    <el-input v-model="state.struct.path" placeholder="inis"></el-input>
                </el-form-item>
                <el-form-item label="COS 外网域名">
                    <el-input v-model="state.struct.domain" placeholder="选填"></el-input>
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
        path:       'inis',
        default:    null,
        domain:     null,
        app_id:     null,
        bucket:     null,
        region:     null,
        secret_id:  null,
        secret_key: null,
    },
    status: {
        active: true,
        finish: false,
        dialog: false,
        loading: true,
        wait: false,
        test: false,
    },
    backup: {},
    select: {
        region: [
            { value: 'ap-beijing-1', label: '北京一区', area: '中国大陆' },
            { value: 'ap-beijing', label: '北京', area: '中国大陆' },
            { value: 'ap-nanjing', label: '南京', area: '中国大陆' },
            { value: 'ap-shanghai', label: '上海', area: '中国大陆' },
            { value: 'ap-guangzhou', label: '广州', area: '中国大陆' },
            { value: 'ap-chengdu', label: '成都', area: '中国大陆' },
            { value: 'ap-chongqing', label: '重庆', area: '中国大陆' },
            { value: 'ap-shenzhen-fsi', label: '深圳金融', area: '中国大陆' },
            { value: 'ap-shanghai-fsi', label: '上海金融', area: '中国大陆' },
            { value: 'ap-beijing-fsi', label: '北京金融', area: '中国大陆' },
            { value: 'ap-hongkong', label: '中国香港', area: '亚太' },
            { value: 'ap-singapore', label: '新加坡', area: '亚太' },
            { value: 'ap-mumbai', label: '孟买', area: '亚太' },
            { value: 'ap-jakarta', label: '雅加达', area: '亚太' },
            { value: 'ap-seoul', label: '首尔', area: '亚太' },
            { value: 'ap-bangkok', label: '曼谷', area: '亚太' },
            { value: 'ap-tokyo', label: '东京', area: '亚太' },
            { value: 'na-siliconvalley', label: '硅谷（美西）', area: '北美' },
            { value: 'na-ashburn', label: '弗吉尼亚（美东）', area: '北美' },
            { value: 'na-toronto', label: '多伦多', area: '北美' },
            { value: 'sa-saopaulo', label: '圣保罗', area: '南美' },
            { value: 'eu-frankfurt', label: '法兰克福', area: '欧洲' },
        ]
    },
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/toml/storage', {
            name: 'cos'
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

        if (!value) return state.status.active = true

        const { code, msg } = await axios.put('/api/toml/storage-default', {
            value: value ? 'cos' : null
        })

        if (code === 200) return emit('refresh')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        let field = ['secret_id', 'secret_key', 'app_id', 'region', 'bucket']

        // 检查关键配置是否有变化
        if (!utils.object.equal(state.struct, state.backup, field)) return ElMessage.warning('请先OSS连接测试')

        if (utils.is.empty(state.struct.secret_id))  return ElMessage.warning('请填写 SecretId！')
        if (utils.is.empty(state.struct.secret_key)) return ElMessage.warning('请填写 SecretKey！')
        if (utils.is.empty(state.struct.app_id))     return ElMessage.warning('请填写 AppId！')
        if (utils.is.empty(state.struct.region))     return ElMessage.warning('请填写 Bucket！')
        if (utils.is.empty(state.struct.bucket))     return ElMessage.warning('请填写 Region！')

        state.status.wait   = true

        const { code, msg } = await axios.put('/api/toml/storage-cos', state.struct)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
        ElMessage.success('保存成功')
    },
    test: async () => {

        if (utils.is.empty(state.struct.secret_id))  return ElMessage.warning('请填写 SecretId！')
        if (utils.is.empty(state.struct.secret_key)) return ElMessage.warning('请填写 SecretKey！')
        if (utils.is.empty(state.struct.app_id))     return ElMessage.warning('请填写 AppId！')
        if (utils.is.empty(state.struct.region))     return ElMessage.warning('请填写 Bucket！')
        if (utils.is.empty(state.struct.bucket))     return ElMessage.warning('请填写 Region！')

        state.status.test         = true

        const { code, msg, data } = await axios.post('/api/toml/test-cos', state.struct)

        state.status.test         = false

        if (code === 200) {
            // 拷贝一份备份
            state.backup = JSON.parse(JSON.stringify(state.struct))
            return ElMessage.success(msg)
        }

        ElMessage.error(`${msg}<br>${data}`)
    },
}

watch(() => state.struct, () => {
    state.status.active = state.struct.default === 'cos'
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
