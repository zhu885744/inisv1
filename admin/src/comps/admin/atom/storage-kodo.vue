<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 七牛云对象存储KODO可以替代传统的本地存储<br>
                        ● 开启后，后续上传的文件将会自动上传到KODO，不会占用服务器的空间和带宽
                    </template>
                    <span style="font-weight: 600">七牛云对象存储</span>
                </el-tooltip>
                <el-tag size="small" type="primary">KODO</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="upload" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">七牛云KODO</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">将资源文件存储到七牛云对象存储</div>
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
            <strong style="display: flex; align-items: center; justify-content: center">配置 七牛云KODO 存储</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="AccessKey">
                    <el-input v-model="state.struct.access_key" show-password placeholder="请输入 AccessKey"></el-input>
                </el-form-item>
                <el-form-item label="SecretKey">
                    <el-input v-model="state.struct.secret_key" show-password placeholder="请输入 SecretKey"></el-input>
                </el-form-item>
                <el-form-item label="Bucket">
                    <el-input v-model="state.struct.bucket" placeholder="请输入 Bucket"></el-input>
                </el-form-item>
                <el-form-item label="Region">
                    <el-select v-model="state.struct.region" placeholder="请选择所在地区">
                        <el-option v-for="item in state.select.region" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.value }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="外网域名">
                    <el-input v-model="state.struct.domain" placeholder="请输入外网域名"></el-input>
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
        default:    null,
        domain:     null,
        access_key: null,
        secret_key: null,
        bucket:     null,
        region:     null,
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
            { value: 'z0',  label: '华东' },
            { value: 'z1',  label: '华北河北' },
            { value: 'z2',  label: '华南广东' },
            { value: 'na0', label: '北美' },
            { value: 'as0', label: '新加坡' },
            { value: 'cn-east-2',      label: '华东浙江' },
            { value: 'ap-northeast-1', label: '亚太-首尔机房' },
        ],
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
            name: 'kodo'
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
            value: value ? 'kodo' : null
        })

        if (code === 200) return emit('refresh')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        let field = ['access_key', 'secret_key', 'bucket', 'region']

        // 密钥已脱敏隐藏时，跳过「是否有变化」检查
        const secretChanged = !utils.is.masked(state.struct.secret_key) && !utils.is.masked(state.struct.access_key)

        // 检查关键配置是否有变化
        if (secretChanged && !utils.object.equal(state.struct, state.backup, field)) return ElMessage.warning('请先KODO连接测试')

        if (utils.is.masked(state.struct.access_key)) return ElMessage.warning('AccessKey 已隐藏，如需修改请重新输入！')
        if (utils.is.empty(state.struct.access_key)) return ElMessage.warning('请填写 AccessKey！')
        if (utils.is.masked(state.struct.secret_key)) return ElMessage.warning('SecretKey 已隐藏，如需修改请重新输入！')
        if (utils.is.empty(state.struct.secret_key)) return ElMessage.warning('请填写 SecretKey！')
        if (utils.is.empty(state.struct.bucket))     return ElMessage.warning('请填写 Bucket！')
        if (utils.is.empty(state.struct.region))     return ElMessage.warning('请填写 Region！')
        if (utils.is.empty(state.struct.domain))     return ElMessage.warning('请填写 外网域名！')

        state.status.wait   = true

        // 剔除脱敏占位字段，由后端保留原值
        const data = utils.object.withoutMasked(state.struct, ['access_key', 'secret_key'])
        const { code, msg } = await axios.put('/api/toml/storage-kodo', data)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
    },
    test: async () => {

        if (utils.is.masked(state.struct.access_key)) return ElMessage.warning('AccessKey 已隐藏，请重新输入后再测试！')
        if (utils.is.empty(state.struct.access_key)) return ElMessage.warning('请填写 AccessKey！')
        if (utils.is.masked(state.struct.secret_key)) return ElMessage.warning('SecretKey 已隐藏，请重新输入后再测试！')
        if (utils.is.empty(state.struct.secret_key)) return ElMessage.warning('请填写 SecretKey！')
        if (utils.is.empty(state.struct.bucket))     return ElMessage.warning('请填写 Bucket！')
        if (utils.is.empty(state.struct.region))     return ElMessage.warning('请填写 Region！')

        state.status.test         = true

        const { code, msg, data } = await axios.post('/api/toml/test-kodo', state.struct)

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
    state.status.active = state.struct.default === 'kodo'
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>