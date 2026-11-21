<template>
  <div class="sec-page">
    <!-- ============ 概览条 ============ -->
    <header class="card card-pad hero">
      <div class="hero-main">
        <span class="hero-badge"><i class="bi bi-shield-lock" /></span>
        <div class="hero-text">
          <h2 class="hero-title">安全与限流</h2>
          <p class="hero-desc">
            接口密钥、访问频率与自动封禁、人机校验——挡扫描器和脚本的几道闸。
            改完立即生效，无需重启
          </p>
        </div>
      </div>

      <div class="hero-side">
        <span class="pill" :class="cfg.apiKey.value === '1' ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> 密钥校验{{ cfg.apiKey.value === '1' ? '已开启' : '未开启' }}
        </span>
        <span class="pill" :class="cfg.qps.value === '1' ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> QPS 限流{{ cfg.qps.value === '1' ? '已开启' : '未开启' }}
        </span>
        <span class="pill" :class="cfg.qpsBlock.value === '1' ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> 自动封禁{{ cfg.qpsBlock.value === '1' ? '已开启' : '未开启' }}
        </span>
        <button class="btn btn-sm" :disabled="loading" @click="loadAll">
          <i class="bi bi-arrow-clockwise" /> 刷新
        </button>
      </div>
    </header>

    <div v-if="loading" class="tip-line"><span class="spinner" /> 正在读取安全策略…</div>

    <template v-else>
      <!-- ============ API KEY ============ -->
      <section class="card card-pad">
        <div class="card-head">
          <div>
            <h3 class="card-title"><i class="bi bi-key" /> 接口密钥</h3>
            <p class="card-desc">
              开启后调用接口需携带有效密钥；密钥本身的增删、有效期与状态在「接口密钥」里管理
            </p>
          </div>
          <RouterLink class="btn btn-sm" to="/admin/security/keys">
            <i class="bi bi-box-arrow-up-right" /> 管理密钥
          </RouterLink>
        </div>

        <div class="grid-2">
          <div class="form-item">
            <label class="form-label">启用校验</label>
            <FormSelect v-model="cfg.apiKey.value" :options="ON_OFF" />
            <p class="row-hint">
              开启时若没有任何可用密钥，系统会自动创建一个；「已停用 / 已过期」的密钥一律拒绝
            </p>
          </div>
          <div class="form-item">
            <label class="form-label">放行本站请求</label>
            <FormSelect v-model="cfg.apiKey.local" :options="ON_OFF" />
            <p class="row-hint">
              关闭后，打包进二进制的站内前端与后台都会 403（必须在反代里注入 i-api-key）
            </p>
          </div>
        </div>

        <p class="warn-line">
          <i class="bi bi-exclamation-triangle" />
          <span>
            同源判定（Origin / Referer / Sec-Fetch-Site）可被脚本伪造，它挡的是扫描器与无脑爬虫，
            不是安全边界；需要强边界时请让后端只监听内网端口，由反代注入 i-api-key
          </span>
        </p>

        <div class="card-foot is-plain">
          <button class="btn btn-primary btn-sm" :disabled="isSaving('apiKey')" @click="saveApiKey">
            <i class="bi bi-check2" /> 保存
          </button>
        </div>
      </section>

      <!-- ============ QPS 限流 ============ -->
      <section class="card card-pad">
        <div class="card-head">
          <div>
            <h3 class="card-title"><i class="bi bi-speedometer2" /> QPS 限流</h3>
            <p class="card-desc">单接口与全局分别限流，超出阈值的请求会被直接拒绝</p>
          </div>
        </div>

        <div class="grid-3">
          <div class="form-item">
            <label class="form-label">启用限流</label>
            <FormSelect v-model="cfg.qps.value" :options="ON_OFF" />
          </div>
          <div class="form-item">
            <label class="form-label">单接口速率（次/秒）</label>
            <FormInput v-model="cfg.qps.json.point" type="number" min="1" step="1" />
          </div>
          <div class="form-item">
            <label class="form-label">全局速率（次/秒）</label>
            <FormInput v-model="cfg.qps.json.global" type="number" min="1" step="1" />
          </div>
        </div>

        <div class="card-foot is-plain">
          <button class="btn btn-primary btn-sm" :disabled="isSaving('qps')" @click="saveQps">
            <i class="bi bi-check2" /> 保存
          </button>
        </div>
      </section>

      <!-- ============ 自动 IP 黑名单 ============ -->
      <section class="card card-pad">
        <div class="card-head">
          <div>
            <h3 class="card-title"><i class="bi bi-slash-circle" /> 自动 IP 黑名单</h3>
            <p class="card-desc">
              触发 QPS 阈值达到指定次数后自动封禁该 IP；同一套记录也用于限制特定 IP 的访问频率
            </p>
          </div>
          <RouterLink class="btn btn-sm" to="/admin/security/ip-black">
            <i class="bi bi-box-arrow-up-right" /> 查看黑名单
          </RouterLink>
        </div>

        <div class="grid-3">
          <div class="form-item">
            <label class="form-label">启用自动封禁</label>
            <FormSelect v-model="cfg.qpsBlock.value" :options="ON_OFF" />
          </div>
          <div class="form-item">
            <label class="form-label">触发次数</label>
            <FormInput v-model="cfg.qpsBlock.json.count" type="number" min="1" step="1" />
            <p class="row-hint">同一 IP 每 10 秒只记一次超限；低于 30 按 30 处理。App / NAT 出口建议 200 以上</p>
          </div>
          <div class="form-item">
            <label class="form-label">统计窗口（秒，支持表达式）</label>
            <FormInput v-model="cfg.qpsBlock.json.second" type="text" placeholder="如 60 * 60" />
            <p class="row-hint">窗口内累计达标才封禁，时长按次数升级（1 小时 → 24 小时 → 7 天 → 永久）</p>
          </div>
        </div>

        <div class="card-foot is-plain">
          <button class="btn btn-primary btn-sm" :disabled="isSaving('qpsBlock')" @click="saveQpsBlock">
            <i class="bi bi-check2" /> 保存
          </button>
        </div>
      </section>

      <!-- ============ 安全验证（滑块） ============ -->
      <section class="card card-pad">
        <div class="card-head">
          <div>
            <h3 class="card-title"><i class="bi bi-sliders2" /> 安全验证（滑块）</h3>
            <p class="card-desc">
              给验证码发送与敏感操作加一层人机校验：命中时前台弹出滑块，拖动对齐后自动继续原请求，
              用户不需要重新提交表单
            </p>
          </div>
        </div>

        <div class="switch-row">
          <div>
            <div class="row-title">发送验证码防刷</div>
            <p class="row-hint">同一手机号 / 邮箱 5 分钟内重复请求发送验证码时要求滑块（首次发送不受影响）</p>
          </div>
          <FormCheckbox variant="switch" v-model="captchaSendCode" />
        </div>

        <div class="scene-title">敏感操作（勾选后每次提交都要求滑块）</div>
        <div class="tiles">
          <button
            v-for="item in CAPTCHA_SCENE_OPTIONS"
            :key="item.key"
            type="button"
            class="tile"
            :class="{ 'is-active': Number(cfg.captcha[item.key]) === 1 }"
            :aria-pressed="Number(cfg.captcha[item.key]) === 1"
            @click="toggleCaptchaScene(item.key)"
          >
            <span class="tile-check">
              <i v-if="Number(cfg.captcha[item.key]) === 1" class="bi bi-check-lg" />
            </span>
            <span class="tile-body">
              <span class="tile-label">{{ item.label }}</span>
              <span class="tile-hint">{{ item.hint }}</span>
            </span>
          </button>
        </div>

        <p class="row-hint is-block">
          滑块通过后 5 分钟内重试同一操作不会重复弹窗（例如密码输错后重试）；全部关闭则完全不影响原有流程
        </p>

        <div class="card-foot is-plain">
          <button class="btn btn-primary btn-sm" :disabled="isSaving('captcha')" @click="saveCaptcha">
            <i class="bi bi-check2" /> 保存
          </button>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 安全中心 · 安全与限流（/admin/security/policy）
 *
 * 原来在「系统配置 → 账号与登录 → 安全与限流」里，但那几项管的是接口访问
 * （密钥校验 / QPS / 自动封禁 / 滑块），与权限、IP 名单同属安全域，
 * 放在安全中心才和「接口密钥」「IP 黑名单」相邻，处置时不用来回跳。
 *
 * 全部是 config 表记录（value 是 "0"/"1" 字符串）：
 *   SYSTEM_API_KEY / SYSTEM_API_KEY_LOCAL / SYSTEM_QPS / SYSTEM_QPS_BLOCK / SYSTEM_CAPTCHA
 * 一次 config/all 拉全量，避免多个 config/one 并发把单接口 QPS 打满。
 */
import { reactive, ref, computed, onMounted } from 'vue'
import { getAllConfigs, saveSystemConfig, getConfig } from '@/api/config'
import { toast } from '@/utils/toast'

// 开关选项（config.value 是字符串）
const ON_OFF = [
  { value: '1', label: '开启' },
  { value: '0', label: '关闭' }
]

/**
 * 滑块验证场景（SYSTEM_CAPTCHA 的 json 字段，与后端 model/captcha-config.go 的场景键一一对应）
 * send_code 单独用一个开关控制（它是防刷策略），其余是「敏感操作」开关
 */
const CAPTCHA_SCENE_OPTIONS = [
  { key: 'login', label: '密码登录', hint: '每次密码登录都要滑块' },
  { key: 'code_login', label: '验证码登录', hint: '提交验证码登录时校验' },
  { key: 'register', label: '注册', hint: '提交注册时校验，可挡批量注册脚本' },
  { key: 'reset_password', label: '重置 / 找回密码', hint: '提交新密码时校验' },
  { key: 'change_contact', label: '修改邮箱 / 手机号', hint: '提交验证码改绑时校验' },
  { key: 'change_account', label: '修改账号', hint: '提交修改账号时校验（该接口没有发码阶段，命中即校验）' },
  { key: 'destroy', label: '注销账户', hint: '提交注销验证时校验' }
]

const loading = ref(false)
// 每个保存动作独立的 loading 标记，避免互相阻塞
const saving = reactive({})

const cfg = reactive({
  apiKey: { value: '0', local: '1' },
  qps: { value: '1', json: { point: 120, global: 600 } },
  qpsBlock: { value: '0', json: { count: 200, second: '60 * 60' } },
  // 安全验证（SYSTEM_CAPTCHA）：send_code 是防刷总开关，其余场景按 0/1 存
  captcha: {
    send_code: '1',
    login: 0,
    code_login: 0,
    register: 0,
    reset_password: 0,
    change_contact: 0,
    change_account: 0,
    destroy: 0
  }
})

// 防刷总开关用「勾选框 → 字符串」映射，保存时统一归一成 0/1
const captchaSendCode = computed({
  get: () => Number(cfg.captcha.send_code) === 1,
  set: (value) => {
    cfg.captcha.send_code = value ? '1' : '0'
  }
})

function isSaving(key) {
  return !!saving[key]
}

/** 统一的异步动作包装：失败提示由请求拦截器给出，这里只负责 loading 与 403 提示 */
async function withSaving(key, fn) {
  saving[key] = true
  try {
    await fn()
    return true
  } catch (err) {
    if (err?.code === 403) {
      toast.warning('当前账号没有这个权限点，请在「权限分组」中勾选后重试（新接口需重启后端补录规则）')
    }
    return false
  } finally {
    saving[key] = false
  }
}

// config 的 json 字段可能是字符串，也可能是后端解码后的对象
function parseJson(raw, fallback = {}) {
  if (!raw) return { ...fallback }
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      return typeof parsed === 'object' && parsed !== null ? parsed : { ...fallback }
    } catch {
      return { ...fallback }
    }
  }
  if (typeof raw === 'object') return raw
  return { ...fallback }
}

// 把对象里的值统一成字符串，便于输入框绑定
function toText(value, fallback = '') {
  if (value === null || value === undefined) return fallback
  return String(value)
}

async function readConfig(key) {
  // 配置项可能尚未初始化，失败时不弹全局提示
  const res = await getConfig(key, { silent: true })
  return res?.data && typeof res.data === 'object' ? res.data : null
}

/** 安全验证配置：只认已知场景，缺失项按后端默认值兜底（send_code 默认开，其余默认关） */
function readCaptchaConfig(raw) {
  const json = parseJson(raw?.json, {})
  const fallback = (key) => (key === 'send_code' ? 1 : 0)

  cfg.captcha.send_code = toText(json.send_code, '1')
  CAPTCHA_SCENE_OPTIONS.forEach((item) => {
    const value = json[item.key]
    cfg.captcha[item.key] = Number(value === undefined || value === null ? fallback(item.key) : value) ? 1 : 0
  })
}

/** 场景开关：点一下切换 0/1 */
function toggleCaptchaScene(key) {
  cfg.captcha[key] = Number(cfg.captcha[key]) === 1 ? 0 : 1
}

async function loadAll() {
  loading.value = true
  try {
    const res = await getAllConfigs({ silent: true }).catch(() => null)

    const configMap = new Map()
    const list = Array.isArray(res?.data?.data) ? res.data.data : []
    list.forEach((item) => {
      if (item && item.key) configMap.set(item.key, item)
    })
    const read = (key) => configMap.get(key) || null

    const apiKey = read('SYSTEM_API_KEY')
    const apiKeyLocal = read('SYSTEM_API_KEY_LOCAL')
    const qps = read('SYSTEM_QPS')
    const block = read('SYSTEM_QPS_BLOCK')
    const captcha = read('SYSTEM_CAPTCHA')

    if (apiKey) cfg.apiKey.value = toText(apiKey.value, '0')
    // 取不到时按「放行」处理：老库升级后这项由 InitConfig 补种为 "1"，
    // 即使还没播种，也不该让站内前端被自己的密钥校验挡住
    cfg.apiKey.local = apiKeyLocal ? toText(apiKeyLocal.value, '1') : '1'
    if (qps) {
      cfg.qps.value = toText(qps.value, '1')
      cfg.qps.json = { point: 120, global: 600, ...parseJson(qps.json, {}) }
    }
    if (block) {
      cfg.qpsBlock.value = toText(block.value, '0')
      cfg.qpsBlock.json = { count: 200, second: '60 * 60', ...parseJson(block.json, {}) }
    }
    if (captcha) {
      readCaptchaConfig(captcha)
    }
  } finally {
    loading.value = false
  }
}

/** 保存 config 记录 */
async function saveConfigItem(key, payload, okText) {
  await withSaving(key, async () => {
    await saveSystemConfig({ key: payload.key, ...payload.data })
    toast.success(okText)
  })
}

// 密钥校验卡片的两个开关一起保存（共用同一个「保存」按钮）
function saveApiKey() {
  return withSaving('apiKey', async () => {
    await saveSystemConfig({ key: 'SYSTEM_API_KEY', value: cfg.apiKey.value })
    await saveSystemConfig({ key: 'SYSTEM_API_KEY_LOCAL', value: cfg.apiKey.local })
    toast.success('已保存')
  })
}

/**
 * 保存安全验证配置（SYSTEM_CAPTCHA）
 *
 * 只提交 json：滑块开关是纯策略配置，不需要 value / text 字段；
 * 保存后回读一次，保证「字符串 / 数字」口径与后端一致（后端统一按 0/1 处理）。
 */
async function saveCaptcha() {
  const setting = { send_code: Number(cfg.captcha.send_code) ? 1 : 0 }
  CAPTCHA_SCENE_OPTIONS.forEach((item) => {
    setting[item.key] = Number(cfg.captcha[item.key]) ? 1 : 0
  })

  await saveConfigItem('captcha', { key: 'SYSTEM_CAPTCHA', data: { json: setting } }, '已保存')

  const fresh = await readConfig('SYSTEM_CAPTCHA')
  if (fresh) readCaptchaConfig(fresh)
}

function saveQps() {
  return saveConfigItem(
    'qps',
    {
      key: 'SYSTEM_QPS',
      data: {
        value: cfg.qps.value,
        json: { point: Number(cfg.qps.json.point), global: Number(cfg.qps.json.global) }
      }
    },
    '已保存'
  )
}

function saveQpsBlock() {
  return saveConfigItem(
    'qpsBlock',
    {
      key: 'SYSTEM_QPS_BLOCK',
      data: {
        value: cfg.qpsBlock.value,
        json: { count: Number(cfg.qpsBlock.json.count), second: String(cfg.qpsBlock.json.second) }
      }
    },
    '已保存'
  )
}

onMounted(loadAll)
</script>

<style scoped>
.sec-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览条 ---------- */
.hero {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  background:
    radial-gradient(120% 140% at 0% 0%, var(--accent-wash), transparent 60%),
    var(--bg-card);
}
.hero-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.hero-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius);
}
.hero-badge .bi {
  font-size: 20px;
}
.hero-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.hero-desc {
  margin: 4px 0 0;
  max-width: 62ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.hero-side {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

/* ---------- 状态标签 ---------- */
.pill {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px 10px;
  font-size: 12px;
  line-height: 1.6;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-soft);
  background: var(--bg-muted);
}
.pill .bi {
  font-size: 16px;
  line-height: 1;
}
.pill.is-on {
  color: var(--success);
  border-color: color-mix(in srgb, var(--success) 30%, transparent);
  background: color-mix(in srgb, var(--success) 8%, transparent);
}
.pill.is-off {
  color: var(--text-muted);
}

/* ---------- 卡片标题行 ---------- */
.card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.card-title .bi {
  font-size: 14px;
  color: var(--primary);
}
.card-desc {
  margin: 4px 0 0;
  max-width: 74ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}

/* ---------- 栅格 ---------- */
.grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}
.grid-3 {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 14px;
}

/* ---------- 提示 ---------- */
.row-title {
  font-size: 13px;
  font-weight: 600;
}
.row-hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.row-hint.is-block {
  margin-top: 12px;
}
.warn-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 0;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 9%, transparent);
  border-radius: var(--radius);
}
.warn-line .bi {
  margin-top: 2px;
}

/* ---------- 开关 ---------- */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  margin-bottom: 16px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.switch {
  position: relative;
  display: inline-block;
  width: 42px;
  height: 24px;
  flex-shrink: 0;
}
.switch input {
  width: 0;
  height: 0;
  opacity: 0;
}
.switch-slider {
  position: absolute;
  inset: 0;
  cursor: pointer;
  background: var(--border);
  border-radius: 999px;
  transition: background-color var(--dur-fast, 0.15s);
}
.switch-slider::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  background: #fff;
  border-radius: 50%;
  box-shadow: var(--shadow-sm);
  transition: transform var(--dur-fast, 0.15s);
}
.switch input:checked + .switch-slider {
  background: var(--primary);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}

/* ---------- 场景开关瓦片 ---------- */
.scene-title {
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-soft);
}
.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 8px;
}
.tile {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  text-align: left;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  cursor: pointer;
  transition: border-color var(--dur-fast, 0.15s), background-color var(--dur-fast, 0.15s);
}
.tile:hover {
  border-color: var(--accent-ring, var(--primary));
}
.tile.is-active {
  background: var(--accent-wash);
  border-color: var(--primary-soft, var(--primary));
}
.tile-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  margin-top: 1px;
  color: #fff;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: 5px;
  transition: all var(--dur-fast, 0.15s);
}
.tile.is-active .tile-check {
  background: var(--primary);
  border-color: var(--primary);
}
.tile-check .bi {
  font-size: 11px;
}
.tile-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.tile-label {
  font-size: 13px;
  color: var(--text);
}
.tile.is-active .tile-label {
  font-weight: 600;
  color: var(--primary-deep);
}
.tile-hint {
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-muted);
}

/* ---------- 页脚操作条 ---------- */
.card-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft, var(--border));
}
.card-foot.is-plain {
  margin-top: 12px;
  padding-top: 0;
  border-top: none;
  justify-content: flex-end;
}

/* ---------- 杂项 ---------- */
.tip-line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.spinner {
  width: 13px;
  height: 13px;
  border: 2px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: sec-spin 0.7s linear infinite;
}
@keyframes sec-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .grid-2,
  .grid-3 {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
