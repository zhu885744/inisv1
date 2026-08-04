<template>
  <el-card style="margin-bottom: 12px" v-loading="state.status.loading">
    <template #header>
      <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
        <el-tooltip placement="top">
          <template #content>
            ● 阿里云号码认证<br>
            ● 注册、登录、找回密码、通知等功能都需要依赖此服务<br>
            ● 号码认证：无需认证，无需资质，个人、企业都可以使用，接入简单，但只能使用阿里云自动分配的短信签名
          </template>
          <span style="font-weight: 600">阿里云号码认证</span>
        </el-tooltip>
        <el-tag size="small" type="info">验证</el-tag>
      </div>
    </template>
    <template #default>
      <div style="display: flex; align-items: center; justify-content: space-between">
        <div style="display: flex; align-items: center; gap: 12px">
          <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
            <i-svg name="aliyun" size="20px"></i-svg>
          </div>
          <div>
            <div style="font-weight: 600; font-size: 14px; line-height: 1.4">阿里云短信验证</div>
            <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">发送验证码进行身份验证</div>
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
      <strong>配置阿里云号码验证服务</strong>
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
          <el-input v-model="state.struct.endpoint" placeholder="请输入 endpoint"></el-input>
        </el-form-item>
        <el-form-item label="短信签名">
          <el-input v-model="state.struct.sign_name" placeholder="请输入短信签名"></el-input>
        </el-form-item>
        <el-form-item label="模板ID">
          <el-input v-model="state.struct.template_code" placeholder="请输入模板ID"></el-input>
        </el-form-item>
        <el-form-item label="测试手机号">
          <el-input v-model="state.struct.phone" v-on:keydown.enter="method.test()" placeholder="请输入手机号">
            <template #append>
              <el-button v-on:click="method.test()" :loading="state.status.test">测试验证服务</el-button>
            </template>
          </el-input>
        </el-form-item>
      </el-form>
    </template>
    <template #footer>
      <el-button v-on:click="state.status.dialog = false">取 消</el-button>
      <el-button v-on:click="method.save()" :loading="state.status.wait" type="primary">保 存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'

const { ctx, proxy } = getCurrentInstance()
const emit = defineEmits(['refresh'])
const state = reactive({
  struct: {
    access_key_id: null,
    access_key_secret: null,
    endpoint: null,
    sign_name: null,
    template_code: null,
    phone: null,
    drive: {
      sms: null,
      default: null,
    },
  },
  status: {
    finish: false,
    active: false,
    dialog: false,
    loading: true,
    wait: false,
    test: false,
  },
  backup: {}
})

onMounted(async () => {
  await method.init()
})

const method = {
  // 初始化配置
  init: async () => {
    state.status.finish = false
    state.status.loading = true

    const { code, data } = await axios.get('/api/toml/sms', {
      name: 'aliyun_number_verify'
    })

    state.status.loading = false

    if (code !== 200) return
    state.struct = data
    // 拷贝备份
    state.backup = JSON.parse(JSON.stringify(data))
    // 驱动状态同步
    state.status.active = data.drive.sms === 'aliyun_number_verify'

    state.status.finish = true
  },
  // 显示配置弹窗
  show() {
    if (!state.status.finish) return ElMessage.warning('号码验证配置获取失败，无法进行配置！')
    state.status.dialog = true
  },
  // 切换启用状态
  change: async value => {
    const { code, msg } = await axios.put('/api/toml/sms-drive', {
      sms: value ? 'aliyun_number_verify' : ''
    })

    if (code === 200) return emit('refresh', 'sms-aliyun', 'sms-tencent')

    state.status.active = !value
    ElMessage.error(msg)
  },
  // 保存配置
  save: async () => {
    let field = ['access_key_id', 'access_key_secret', 'endpoint', 'sign_name', 'template_code']
    // 关键配置校验
    for (let key of field) {
      if (utils.is.empty(state.struct[key])) {
        return ElMessage.warning(`${key} 不能为空！`)
      }
    }

    state.status.wait = true

    const { code, msg } = await axios.put('/api/toml/sms-aliyun-number-verify', state.struct)

    state.status.wait = false

    if (code !== 200) return ElMessage.error(`保存失败：${msg}`)

    ElMessage.success('保存成功')
    state.status.dialog = false
    // 刷新配置
    await method.init()
  },
  // 测试号码验证服务
  test: async () => {
    if (utils.is.empty(state.struct.phone)) {
      return ElMessage.warning('测试手机号不能为空！')
    }
    if (!utils.is.phone(state.struct.phone)) {
      return ElMessage.warning('手机号格式不正确！')
    }

    // 必传参数校验
    let checkField = ['access_key_id', 'access_key_secret', 'endpoint', 'sign_name', 'template_code']
    for (let key of checkField) {
      if (utils.is.empty(state.struct[key])) {
        return ElMessage.warning(`${key} 不能为空！`)
      }
    }

    state.status.test = true

    const { code, data, msg } = await axios.post('/api/toml/test-sms-aliyun-number-verify', state.struct)

    state.status.test = false

    if (code !== 200) return ElMessage.error(`测试失败：${msg || data}`)

    ElMessage.success(`测试成功！验证码：${data.verify_code || '已发送'}`)
  },
}

defineExpose({ init: method.init })

watch(() => state.struct, () => {
  // 仅允许字母、数字、特殊字符（沿用JWT配置的校验逻辑）
}, { deep: true })
</script>
