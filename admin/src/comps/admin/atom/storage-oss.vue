<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 阿里云对象存储OSS可以替代传统的本地存储，有能力的情况推荐开启OSS存储<br>
                        ● 开启后，后续上传的文件将会自动上传到OSS，不会占用服务器的空间和带宽
                    </template>
                    <span style="font-weight: 600">阿里云对象存储</span>
                </el-tooltip>
                <el-tag size="small" type="danger">OSS</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="upload" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">阿里云OSS</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">将资源文件存储到阿里云对象存储</div>
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
            <strong style="display: flex; align-items: center; justify-content: center">配置 阿里云OSS 存储</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="AccessKey ID">
                    <el-input v-model="state.struct.access_key_id" show-password placeholder="请输入 AccessKey ID"></el-input>
                </el-form-item>
                <el-form-item label="AccessKey Secret">
                    <el-input v-model="state.struct.access_key_secret" show-password placeholder="请输入 AccessKey Secret"></el-input>
                </el-form-item>
                <el-form-item label="Endpoint">
                    <el-select v-model="state.struct.endpoint" placeholder="请选择所在地区">
                        <el-option v-for="item in state.select.endpoint" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.value }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="OSS Bucket">
                    <el-input v-model="state.struct.bucket" placeholder="请输入 OSS Bucket"></el-input>
                </el-form-item>
                <el-form-item label="存储目录">
                    <el-input v-model="state.struct.path" placeholder="inis"></el-input>
                </el-form-item>
                <el-form-item label="OSS 外网域名">
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
        path:              'inis',
        default:           null,
        domain:            null,
        bucket:            null,
        endpoint:          null,
        access_key_id:     null,
        access_key_secret: null,
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
        endpoint: [
            { value: 'oss-cn-hangzhou.aliyuncs.com', label: '华东1（杭州）' },
            { value: 'oss-cn-shanghai.aliyuncs.com', label: '华东2（上海）' },
            { value: 'oss-cn-nanjing.aliyuncs.com', label: '华东5（南京-本地地域）' },
            { value: 'oss-cn-fuzhou.aliyuncs.com', label: '华东6（福州-本地地域）' },
            { value: 'oss-cn-qingdao.aliyuncs.com', label: '华北1（青岛）' },
            { value: 'oss-cn-beijing.aliyuncs.com', label: '华北2（北京）' },
            { value: 'oss-cn-zhangjiakou.aliyuncs.com', label: '华北3（张家口）' },
            { value: 'oss-cn-huhehaote.aliyuncs.com', label: '华北5（呼和浩特）' },
            { value: 'oss-cn-wulanchabu.aliyuncs.com', label: '华北6（乌兰察布）' },
            { value: 'oss-cn-shenzhen.aliyuncs.com', label: '华南1（深圳）' },
            { value: 'oss-cn-heyuan.aliyuncs.com', label: '华南2（河源）' },
            { value: 'oss-cn-guangzhou.aliyuncs.com', label: '华南3（广州）' },
            { value: 'oss-cn-chengdu.aliyuncs.com', label: '西南1（成都）' },
            { value: 'oss-cn-hongkong.aliyuncs.com', label: '中国香港' },
            { value: 'oss-us-west-1.aliyuncs.com', label: '美国（硅谷）①' },
            { value: 'oss-us-east-1.aliyuncs.com', label: '美国（弗吉尼亚）②' },
            { value: 'oss-ap-northeast-1.aliyuncs.com', label: '日本（东京）①' },
            { value: 'oss-ap-northeast-2.aliyuncs.com', label: '韩国（首尔）' },
            { value: 'oss-ap-southeast-1.aliyuncs.com', label: '新加坡①' },
            { value: 'oss-ap-southeast-2.aliyuncs.com', label: '澳大利亚（悉尼）①' },
            { value: 'oss-ap-southeast-3.aliyuncs.com', label: '马来西亚（吉隆坡）①' },
            { value: 'oss-ap-southeast-5.aliyuncs.com', label: '印度尼西亚（雅加达）①' },
            { value: 'oss-ap-southeast-6.aliyuncs.com', label: '菲律宾（马尼拉）' },
            { value: 'oss-ap-southeast-7.aliyuncs.com', label: '泰国（曼谷）' },
            { value: 'oss-ap-south-1.aliyuncs.com', label: '印度（孟买）①' },
            { value: 'oss-eu-central-1.aliyuncs.com', label: '德国（法兰克福）①' },
            { value: 'oss-eu-west-1.aliyuncs.com', label: '英国（伦敦）' },
            { value: 'oss-me-east-1.aliyuncs.com', label: '阿联酋（迪拜）①' },
            { value: 'oss-rg-china-mainland.aliyuncs.com', label: '无地域属性（中国内地）' },
            { value: 'oss-cn-hzfinance.aliyuncs.com', label: '杭州金融云公网' },
            { value: 'oss-cn-shanghai-finance-1-pub.aliyuncs.com', label: '上海金融云公网' },
            { value: 'oss-cn-szfinance.aliyuncs.com', label: '深圳金融云公网' },
            { value: 'oss-cn-beijing-finance-1-pub.aliyuncs.com', label: '北京金融云公网' },
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
            name: 'oss'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        // 拷贝一份备份
        state.backup = JSON.parse(JSON.stringify(data))

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('存储配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        if (!value) return state.status.active = true

        const { code, msg } = await axios.put('/api/toml/storage-default', {
            value: value ? 'oss' : null
        })

        if (code === 200) return emit('refresh')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        let field = ['access_key_id', 'access_key_secret', 'endpoint', 'bucket']

        // 密钥已脱敏隐藏时，跳过「是否有变化」检查
        const secretChanged = !utils.is.masked(state.struct.access_key_secret)

        // 检查关键配置是否有变化
        if (secretChanged && !utils.object.equal(state.struct, state.backup, field)) return ElMessage.warning('请先OSS连接测试')

        if (utils.is.empty(state.struct.access_key_id))     return ElMessage.warning('请填写 AccessKey ID！')
        if (utils.is.masked(state.struct.access_key_secret)) return ElMessage.warning('AccessKey Secret 已隐藏，如需修改请重新输入！')
        if (utils.is.empty(state.struct.endpoint))          return ElMessage.warning('请填写 Endpoint！')
        if (utils.is.empty(state.struct.bucket))            return ElMessage.warning('请填写 Bucket！')

        state.status.wait   = true

        // 剔除脱敏占位字段，由后端保留原值
        const data = utils.object.withoutMasked(state.struct, ['access_key_secret'])
        const { code, msg } = await axios.put('/api/toml/storage-oss', data)

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        ElMessage.success('保存成功')
        state.status.dialog = false
    },
    test: async () => {

        if (utils.is.empty(state.struct.access_key_id))     return ElMessage.warning('请填写 AccessKey ID！')
        if (utils.is.masked(state.struct.access_key_secret)) return ElMessage.warning('AccessKey Secret 已隐藏，请重新输入后再测试！')
        if (utils.is.empty(state.struct.access_key_secret)) return ElMessage.warning('请填写 AccessKey Secret！')
        if (utils.is.empty(state.struct.endpoint))          return ElMessage.warning('请填写 Endpoint！')
        if (utils.is.empty(state.struct.bucket))            return ElMessage.warning('请填写 Bucket！')

        state.status.test         = true

        const { code, msg, data } = await axios.post('/api/toml/test-oss', state.struct)

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
    state.status.active = state.struct.default === 'oss'
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
