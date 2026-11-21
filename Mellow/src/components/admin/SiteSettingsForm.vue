<template>
  <div class="site-settings">
    <!-- 附件库（站点 LOGO / favicon 选择已有图片） -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      :title="libraryTitle"
      accept="image"
      @select="onLibrarySelect"
    />

    <!-- 网站基本信息 -->
    <section class="card card-pad">
      <header class="config-head">
        <div>
          <h3 class="config-title">网站基本信息</h3>
          <p class="config-desc">标题、描述、LOGO、图标与建站日期，前台头部与浏览器标签均取用这里的值</p>
        </div>
      </header>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">网站标题</label>
          <FormInput v-model="globalConfig.title" placeholder="输入您的网站标题" />
          <p class="form-hint">显示在前台左栏顶部、浏览器标签与页脚版权处</p>
        </div>
        <div class="form-item">
          <label class="form-label">网站关键词</label>
          <FormInput v-model="globalConfig.keyword" placeholder="输入关键词，用逗号分隔" />
          <p class="form-hint">写入页面 meta keywords，多个关键词用逗号分隔</p>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">网站描述</label>
        <FormInput v-model="globalConfig.description" placeholder="输入您的网站描述" />
        <p class="form-hint">写入页面 meta description，同时显示在前台左栏标题下方</p>
      </div>

      <div class="form-item">
        <label class="form-label">网站 LOGO</label>
        <div class="input-group">
          <FormInput v-model="globalConfig.avatar" placeholder="输入 LOGO URL" />
          <button type="button" class="btn btn-sm" title="从附件库选择" @click="openLibrary('avatar')">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
        </div>
        <p class="form-hint">
          显示在前台左栏顶部（标题上方）、登录页与手机端顶栏；建议用透明底的方形或横向图片，留空则不显示
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">网站图标</label>
        <div class="input-group">
          <FormInput v-model="globalConfig.favicon" placeholder="输入 favicon URL" />
          <button type="button" class="btn btn-sm" title="从附件库选择" @click="openLibrary('favicon')">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
        </div>
        <p class="form-hint">
          浏览器标签页图标（favicon，支持 .ico / .png / .svg），留空则用主题默认的 /favicon.ico；
          改完刷新一次页面即可看到
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">建站日期</label>
        <FormInput type="date" :value="formatDate(globalConfig.date)" @change="handleDateChange" />
        <p class="form-hint">网站的建立日期</p>
      </div>
    </section>

    <!-- 备案信息 -->
    <section class="card card-pad">
      <header class="config-head">
        <div>
          <h3 class="config-title">备案信息</h3>
          <p class="config-desc">页脚展示的 ICP 与公安联网备案信息</p>
        </div>
      </header>
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">ICP 备案编号</label>
          <FormInput v-model="globalConfig.copy.code" placeholder="输入 ICP 备案编号" />
        </div>
        <div class="form-item">
          <label class="form-label">ICP 备案链接</label>
          <FormInput v-model="globalConfig.copy.link" placeholder="输入备案链接" />
        </div>
        <div class="form-item">
          <label class="form-label">公安联网备案编号</label>
          <FormInput v-model="globalConfig.police.code" placeholder="输入公安联网备案编号" />
        </div>
        <div class="form-item">
          <label class="form-label">公安备案链接</label>
          <FormInput v-model="globalConfig.police.link" placeholder="输入公安备案链接" />
        </div>
      </div>
    </section>

    <!-- 登录协议 -->
    <section class="card card-pad">
      <header class="config-head">
        <div>
          <h3 class="config-title">登录协议提示配置</h3>
          <p class="config-desc">登录 / 注册 / 找回密码弹窗中的协议勾选提示</p>
        </div>
      </header>
      <div class="switch-row">
        <div>
          <div class="form-label">启用登录协议提示</div>
          <p class="form-hint">在登录/注册/找回密码弹窗中显示协议同意提示</p>
        </div>
        <FormCheckbox variant="switch" v-model="globalConfig.auth_dialog_agreement.enabled" />
      </div>
      <template v-if="globalConfig.auth_dialog_agreement.enabled">
        <div class="form-item">
          <label class="form-label">用户协议内容</label>
          <FormTextarea v-model="globalConfig.auth_dialog_agreement.user_agreement_content" rows="4" />
        </div>
        <div class="form-item">
          <label class="form-label">隐私协议内容</label>
          <FormTextarea v-model="globalConfig.auth_dialog_agreement.privacy_agreement_content" rows="4" />
        </div>
      </template>
    </section>

    <!-- 右侧悬浮按钮 -->
    <section class="card card-pad">
      <header class="config-head">
        <div>
          <h3 class="config-title">右侧悬浮按钮设置</h3>
          <p class="config-desc">前台右侧的快捷入口，可自行增删与排序</p>
        </div>
      </header>

      <div class="switch-row">
        <div>
          <div class="form-label">启用悬浮按钮</div>
          <p class="form-hint">在网站右侧显示悬浮按钮</p>
        </div>
        <FormCheckbox variant="switch" v-model="globalConfig.float_buttons.enabled" />
      </div>

      <template v-if="globalConfig.float_buttons.enabled">
        <div class="switch-row compact">
          <label class="form-label">显示返回顶部按钮</label>
          <FormCheckbox variant="switch" v-model="globalConfig.float_buttons.show_back_to_top" />
        </div>
        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">按钮样式</label>
            <FormSelect
              v-model="globalConfig.float_buttons.style"
              :options="floatStyleOptions" />
          </div>
          <div class="form-item">
            <label class="form-label">显示位置</label>
            <FormSelect
              v-model="globalConfig.float_buttons.position"
              :options="floatPositionOptions" />
          </div>
        </div>

        <!-- 按钮管理 -->
        <div class="float-btn-manager">
          <div class="manager-head">
            <span class="form-label">悬浮按钮管理</span>
            <button type="button" class="btn btn-sm" @click="addFloatButton">
              <i class="bi bi-plus-lg" /> 添加
            </button>
          </div>
          <p v-if="!globalConfig.float_buttons.buttons.length" class="manager-empty">
            还没有自定义按钮，点击「添加」创建第一个。
          </p>
          <div
            v-for="(button, index) in globalConfig.float_buttons.buttons"
            :key="button.id"
            class="float-btn-item"
          >
            <div class="float-btn-head">
              <strong>{{ button.name }}</strong>
              <div class="float-btn-actions">
                <button type="button" class="btn btn-sm" :disabled="index === 0" title="上移" @click="moveFloatButton(index, 'up')">
                  <i class="bi bi-arrow-up" />
                </button>
                <button
                  type="button"
                  class="btn btn-sm"
                  :disabled="index === globalConfig.float_buttons.buttons.length - 1"
                  title="下移"
                  @click="moveFloatButton(index, 'down')"
                >
                  <i class="bi bi-arrow-down" />
                </button>
                <button type="button" class="btn btn-sm btn-danger" @click="removeFloatButton(index)">删除</button>
              </div>
            </div>
            <div class="switch-row compact">
              <label class="form-label">启用</label>
              <FormCheckbox variant="switch" v-model="button.enabled" />
            </div>
            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">按钮名称</label>
                <FormInput v-model="button.name" />
              </div>
              <div class="form-item">
                <label class="form-label">按钮图标</label>
                <FormInput v-model="button.icon" placeholder="例如：bi bi-qq" />
              </div>
            </div>
            <div class="form-item">
              <label class="form-label">按钮链接</label>
              <FormInput v-model="button.url" placeholder="输入按钮链接" />
            </div>
            <div class="form-item">
              <label class="form-label">按钮简介</label>
              <FormInput v-model="button.tooltip" placeholder="鼠标悬停时显示" />
            </div>
            <div class="form-item">
              <label class="form-label">悬浮图片</label>
              <FormInput v-model="button.image_url" placeholder="鼠标移到此处的图片" />
              <div class="image-actions">
                <button
                  type="button"
                  class="btn btn-sm"
                  :disabled="uploading"
                  @click="pickButtonImage(button.id)"
                >
                  <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
                </button>
                <button type="button" class="btn btn-sm" title="从附件库选择" @click="openButtonLibrary(button.id)">
                  <i class="bi bi-folder2-open" /> 附件库
                </button>
                <button
                  v-if="button.image_url"
                  type="button"
                  class="btn btn-sm btn-ghost"
                  @click="button.image_url = ''"
                >
                  移除
                </button>
              </div>
              <p class="form-hint">设置此处后，按钮链接将失效</p>
            </div>
          </div>
        </div>
      </template>
    </section>

    <!-- 悬浮按钮图片上传（多个按钮共用一个隐藏 input，靠 uploadingButtonId 定位） -->
    <input ref="buttonFileRef" type="file" accept="image/*" hidden @change="onButtonImageChange" />

    <!-- 保存 -->
    <section class="card card-pad">
      <div class="config-foot">
        <span class="save-hint">
          修改后需点击保存才会写入配置，前台会立即读取最新值；自定义菜单与个性侧边栏在「网站表单设置」里可视化调整
        </span>
        <button class="btn" :disabled="saving" @click="resetGlobalConfig">重置</button>
        <button class="btn btn-primary" :disabled="saving" @click="saveGlobalConfig">
          {{ saving ? '保存中...' : '保存网站设置' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { ref, onMounted, computed } from 'vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { saveConfig } from '@/api/config'
import { uploadAttachments } from '@/api/attachment'
import { useSiteStore } from '@/stores/site'
import { toast } from '@/utils/toast'

const siteStore = useSiteStore()

// 站点设置存储在 config 表的 Mellow_functions 记录（value 为空，配置本体在 json）
const CONFIG_KEY = 'Mellow_functions'

// 悬浮按钮下拉选项（主题内置 SelectMenu）
const floatStyleOptions = [
  { value: 'rounded', label: '圆角按钮' },
  { value: 'square', label: '方形按钮' }
]
const floatPositionOptions = [
  { value: 'center', label: '右侧居中' },
  { value: 'bottom', label: '右侧底部' }
]

// 附件库：选择已上传图片作为站点 LOGO / favicon / 悬浮按钮图片
const showLibrary = ref(false)
const libraryKey = ref('avatar')
// 悬浮按钮图片：记录要写入哪个按钮（按按钮 id 定位，避免增删后下标错位）
const libraryButtonId = ref(0)
const libraryTitle = computed(() => {
  if (libraryButtonId.value) return '选择悬浮按钮图片'
  return libraryKey.value === 'favicon' ? '选择网站图标' : '选择网站 LOGO'
})

function openLibrary(key) {
  libraryKey.value = key
  libraryButtonId.value = 0
  showLibrary.value = true
}

function openButtonLibrary(buttonId) {
  libraryButtonId.value = Number(buttonId) || 0
  showLibrary.value = true
}

function onLibrarySelect(urls = []) {
  const url = urls[0]
  if (!url) return

  if (libraryButtonId.value) {
    const button = globalConfig.value.float_buttons.buttons.find((item) => item.id === libraryButtonId.value)
    if (button) button.image_url = url
    libraryButtonId.value = 0
  } else {
    globalConfig.value[libraryKey.value] = url
  }

  toast.success('图片已应用，记得点「保存」')
}

// 悬浮按钮图片：上传
const uploading = ref(false)
const buttonFileRef = ref(null)
const uploadingButtonId = ref(0)

function pickButtonImage(buttonId) {
  uploadingButtonId.value = Number(buttonId) || 0
  buttonFileRef.value?.click()
}

async function onButtonImageChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')

    const button = globalConfig.value.float_buttons.buttons.find((item) => item.id === uploadingButtonId.value)
    if (button) button.image_url = url
    toast.success('图片已上传，记得点「保存」')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
    uploadingButtonId.value = 0
  }
}

// 全局配置
const globalConfig = ref(defaultConfig())

function defaultConfig() {
  return {
    title: '',
    description: '',
    keyword: '',
    avatar: '',
    favicon: '',
    date: Math.floor(Date.now() / 1000).toString(),
    copy: { code: '', link: 'https://beian.miit.gov.cn/#/Integrated/index' },
    police: { code: '', link: 'https://beian.mps.gov.cn/#/query/webSearch' },
    auth_dialog_agreement: {
      enabled: true,
      user_agreement_content: '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议。\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性。\n\n3. 用户行为规范\n请勿发布违法或侵犯他人权益的内容。',
      privacy_agreement_content: '隐私协议\n\n我们重视您的隐私。\n\n1. 收集的信息\n我们可能收集您的账户信息和使用数据。\n\n2. 信息使用\n用于提供和改进服务。\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。'
    },
    float_buttons: {
      enabled: true,
      style: 'rounded',
      position: 'center',
      show_back_to_top: true,
      buttons: []
    },
  }
}

const saving = ref(false)

// 格式化日期
function formatDate(timestamp) {
  if (!timestamp) return ''
  const date = new Date(parseInt(timestamp) * 1000)
  return date.toISOString().split('T')[0]
}

function handleDateChange(event) {
  const dateStr = event.target.value
  globalConfig.value.date = dateStr
    ? Math.floor(new Date(dateStr).getTime() / 1000).toString()
    : Math.floor(Date.now() / 1000).toString()
}

// 获取全局配置（force 保证进入后台时拿到的是最新值，避免覆盖后台刚改的内容）
async function getGlobalConfig() {
  try {
    const config = (await siteStore.load(true)) || {}
    const base = defaultConfig()
    globalConfig.value = {
      title: config.title || '',
      description: config.description || '',
      keyword: config.keyword || '',
      avatar: config.avatar || '',
      favicon: config.favicon || '',
      date: config.date || base.date,
      copy: { code: config.copy?.code || '', link: config.copy?.link || base.copy.link },
      police: { code: config.police?.code || '', link: config.police?.link || base.police.link },
      auth_dialog_agreement: {
        enabled: config.auth_dialog_agreement?.enabled !== false,
        user_agreement_content: config.auth_dialog_agreement?.user_agreement_content || base.auth_dialog_agreement.user_agreement_content,
        privacy_agreement_content: config.auth_dialog_agreement?.privacy_agreement_content || base.auth_dialog_agreement.privacy_agreement_content
      },
      float_buttons: {
        enabled: config.float_buttons?.enabled !== false,
        style: config.float_buttons?.style || 'rounded',
        position: config.float_buttons?.position || 'center',
        show_back_to_top: config.float_buttons?.show_back_to_top !== false,
        buttons: config.float_buttons?.buttons || []
      }
      // 自定义菜单 / 个性侧边栏已抽到「网站表单设置」（components/admin/SiteLayoutForm.vue）：
      // 本表单不再读写 custom_menus / sidebar，保存时靠合并最新配置把它们原样保留
    }
  } catch {
    toast.error('获取网站设置失败')
  }
}

// 保存全局配置
async function saveGlobalConfig() {
  saving.value = true
  try {
    // 先取一次最新完整配置再合并：本表单只认识自己的字段，
    // 直接整体覆盖会把其它后台页面（默认头像、上传模式等）写入的键抹掉
    const latest = (await siteStore.load(true)) || {}
    await saveConfig(CONFIG_KEY, { ...latest, ...globalConfig.value })
    toast.success('网站设置保存成功')
    // 同步失效 site store 缓存，保证前台立即读取到最新配置
    await siteStore.invalidate()
  } catch {
    toast.error('网站设置保存失败')
  } finally {
    saving.value = false
  }
}

// 悬浮按钮管理
function addFloatButton() {
  const buttons = globalConfig.value.float_buttons.buttons
  const newId = buttons.length > 0 ? Math.max(...buttons.map((b) => b.id)) + 1 : 1
  buttons.push({ id: newId, name: '新按钮', icon: 'bi bi-link', url: '', tooltip: '', enabled: true, content: '' })
}

function removeFloatButton(index) {
  globalConfig.value.float_buttons.buttons.splice(index, 1)
}

function moveFloatButton(index, direction) {
  const buttons = globalConfig.value.float_buttons.buttons
  const target = direction === 'up' ? index - 1 : index + 1
  if (target < 0 || target >= buttons.length) return
  const temp = buttons[index]
  buttons[index] = buttons[target]
  buttons[target] = temp
}

// 重置为默认值（仅本地，需保存才生效）
function resetGlobalConfig() {
  globalConfig.value = defaultConfig()
}

onMounted(getGlobalConfig)
</script>

<style scoped>
.site-settings {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* 卡片头（与 /admin/system 其它模块保持一致） */
.config-head {
  margin-bottom: 16px;
}
.config-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.config-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.config-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 0;
}
.save-hint {
  margin-right: auto;
  font-size: 12px;
  color: var(--text-muted);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.form-hint code {
  background: var(--bg-muted);
  padding: 1px 5px;
  border-radius: 3px;
  color: var(--primary-deep);
  font-size: 11px;
}

/* 输入框 + 附件库按钮同行 */
.input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.input-group .input {
  flex: 1;
  min-width: 0;
}
.input-group .btn {
  flex-shrink: 0;
}

/* 图片字段下方的操作按钮（上传 / 附件库 / 移除） */
.image-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.code-textarea {
  font-family: var(--font-mono);
  font-size: 12px;
}

/* 开关 */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  margin-bottom: 14px;
}
.switch-row .form-label {
  margin-bottom: 0;
}
.switch-row .form-hint {
  margin-top: 2px;
}
.switch-row.compact {
  padding: 8px 14px;
  min-height: auto;
}
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  flex-shrink: 0;
  cursor: pointer;
}
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.switch-slider {
  position: absolute;
  inset: 0;
  background: var(--border);
  border-radius: 22px;
  transition: 0.2s;
}
.switch-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 3px;
  top: 3px;
  background: var(--switch-knob, #fff);
  border-radius: 50%;
  transition: 0.2s;
}
.switch input:checked + .switch-slider {
  background: var(--primary);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}
.switch input:focus-visible + .switch-slider {
  outline: 2px solid var(--primary-soft);
  outline-offset: 2px;
}

/* 悬浮按钮管理 */
.float-btn-manager {
  margin-top: 8px;
}
.manager-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.manager-head .form-label {
  margin-bottom: 0;
}
.manager-empty {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--text-muted);
}
.float-btn-item {
  background: var(--bg-soft);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  padding: 14px;
  margin-bottom: 12px;
}
.float-btn-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 10px;
}
.float-btn-head strong {
  font-size: 13px;
}
.float-btn-actions {
  display: flex;
  gap: 4px;
}

@media (max-width: 720px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
  .config-foot {
    flex-wrap: wrap;
  }
  .save-hint {
    width: 100%;
    margin-right: 0;
  }
}
</style>
