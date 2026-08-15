<template>
    <el-card style="margin-bottom: 12px" v-loading="state.status.loading">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 用于发送验证码相关的服务<br>
                        ● 注册、登录、找回密码、通知等功能都需要依赖此服务
                    </template>
                    <span style="font-weight: 600">阿里云短信</span>
                </el-tooltip>
                <el-tag size="small" type="warning">企业</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="aliyun" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">阿里云短信</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">用于发送验证码、通知等短信服务</div>
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
            <strong>配置阿里云短信服务</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="AccessKey ID">
                    <el-input v-model="state.struct.access_key_id" show-password placeholder="请输入 AccessKey ID"></el-input>
                </el-form-item>
                <el-form-item label="AccessKey Secret">
                    <el-input v-model="state.struct.access_key_secret" show-password placeholder="请输入 AccessKey Secret"></el-input>
                </el-form-item>
                <el-form-item label="endpoint">
                    <el-input v-model="state.struct.endpoint" placeholder="dysmsapi.aliyuncs.com"></el-input>
                </el-form-item>
                <el-form-item label="短信签名">
                    <el-input v-model="state.struct.sign_name" placeholder="请输入短信签名"></el-input>
                </el-form-item>
                <el-form-item label="验证码模板">
                    <el-input v-model="state.struct.verify_code" placeholder="SMS_XXX02"></el-input>
                </el-form-item>
                <el-form-item label="接收者手机号">
                    <el-input v-model="state.struct.phone" v-on:keydown.enter="method.test()" placeholder="请输入手机号">
                        <template #append>
                            <el-button v-on:click="method.test()" :loading="state.status.test">阿里云短信测试</el-button>
                        </template>
                    </el-input>
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
        access_key_id:      null,
        access_key_secret:  null,
        endpoint:       null,
        sign_name:      null,
        verify_code:    null,
        phone:          null,
        drive:     {
            sms: null,
            default: null,
        },
    },
    status: {
        finish: false,
        active: false,
        dialog: false,
        loading: true,
        test: false,
        wait: false,
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

        const { code, data } = await axios.get('/api/toml/sms', {
            name: 'aliyun'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data

        // 拷贝一份备份
        state.backup = JSON.parse(JSON.stringify(data))

        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('SMS服务配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    change: async value => {

        const { code, msg } = await axios.put('/api/toml/sms-drive', {
            sms: value ? 'aliyun' : ''
        })

        if (code === 200) return emit('refresh', 'sms-tencent', 'sms-aliyun-verify')

        state.status.active = !value
        ElMessage.error(msg)
    },
    save: async () => {

        let field = ['access_key_id', 'access_key_secret', 'endpoint', 'sign_name', 'verify_code']

        // 密钥已脱敏隐藏时，跳过「是否有变化」检查（用户可能仅修改其他字段）
        const secretChanged = !utils.is.masked(state.struct.access_key_secret)

        // 检查关键配置是否有变化
        if (secretChanged && !utils.object.equal(state.struct, state.backup, field)) return ElMessage.warning('请先完成阿里云短信测试')

        if (utils.is.empty(state.struct.access_key_id))      return ElMessage.warning('请填写阿里云AccessKey ID！')
        if (utils.is.masked(state.struct.access_key_secret)) return ElMessage.warning('AccessKey Secret 已隐藏，如需修改请重新输入！')
        if (utils.is.empty(state.struct.endpoint))           return ElMessage.warning('请填写阿里云短信服务endpoint！')
        if (utils.is.empty(state.struct.sign_name))          return ElMessage.warning('请填写短信签名！')
        if (utils.is.empty(state.struct.verify_code))        return ElMessage.warning('请填写验证码模板！')

        state.status.wait   = true

        // 剔除脱敏占位字段，由后端保留原值
        const data = utils.object.withoutMasked(state.struct, ['access_key_secret'])
        const { code, msg } = await axios.put('/api/toml/sms-aliyun', data)

         state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
        ElMessage.success('保存成功')
    },
    test: async () => {

        if (utils.is.empty(state.struct.phone))              return ElMessage.warning('请填写接收者手机号！')
        if (utils.is.empty(state.struct.access_key_id))      return ElMessage.warning('请填写阿里云AccessKey ID！')
        if (utils.is.masked(state.struct.access_key_secret)) return ElMessage.warning('AccessKey Secret 已隐藏，请重新输入后再测试！')
        if (utils.is.empty(state.struct.access_key_secret))  return ElMessage.warning('请填写阿里云AccessKey Secret！')
        if (utils.is.empty(state.struct.endpoint))           return ElMessage.warning('请填写阿里云短信服务endpoint！')
        if (utils.is.empty(state.struct.sign_name))          return ElMessage.warning('请填写短信签名！')
        if (utils.is.empty(state.struct.verify_code))        return ElMessage.warning('请填写验证码模板！')
        if (!utils.is.phone(state.struct.phone))             return ElMessage.warning('接收者手机号格式不正确！')

        state.status.test         = true

        const { code, msg, data } = await axios.post('/api/toml/test-sms-aliyun', state.struct)

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
    state.status.active = state.struct.drive.sms === 'aliyun'
}, { deep: true })

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
