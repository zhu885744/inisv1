<template>
  <div class="install-page">
    <div class="install-box">
      <header class="install-head">
        <span class="install-logo">INIS</span>
        <div class="install-head-text">
          <h1 class="install-title">安装向导</h1>
          <p class="install-sub">Mellow 主题 · 按提示完成初始化即可开始使用</p>
        </div>
      </header>

      <!-- 已安装 / 刚装完：只给出入口 -->
      <div v-if="finished" class="card card-pad install-done">
        <i class="bi bi-check-circle-fill install-done-icon" />
        <p class="install-done-title">{{ installed ? '系统已完成安装' : '安装完成，可以开始使用了' }}</p>
        <p class="install-done-hint">
          {{
            installed
              ? '如需重新安装，请删除程序根目录下的 install.lock 后重启服务。'
              : '默认管理员账号 admin / admin123456，请登录后立即修改密码。'
          }}
        </p>
        <div class="install-done-actions">
          <a class="btn btn-primary" href="/">进入首页</a>
          <a class="btn" href="/auth/login">登录后台</a>
        </div>
      </div>

      <template v-else>
        <!-- 步骤条 -->
        <ol class="install-steps">
          <li
            v-for="item in steps"
            :key="item.step"
            class="install-steps-item"
            :class="{ 'is-done': current > item.step, 'is-active': current === item.step }"
          >
            <span class="install-steps-dot">
              <i v-if="current > item.step" class="bi bi-check-lg" />
              <template v-else>{{ item.step }}</template>
            </span>
            <span class="install-steps-label">{{ item.label }}</span>
          </li>
        </ol>

        <div class="card card-pad install-panel">
          <!-- 1. 环境检查 -->
          <section v-if="current === 1" class="install-step">
            <p class="install-desc">
              先确认运行环境与安装状态：程序根目录下的 <code>install.lock</code> 存在表示尚未安装。
            </p>

            <ul class="install-env">
              <li v-for="item in envList" :key="item.label">
                <span class="install-env-label">{{ item.label }}</span>
                <b class="install-env-value">{{ item.value }}</b>
              </li>
            </ul>

            <div class="install-actions">
              <button
                class="btn"
                :class="{ 'is-load': loading }"
                type="button"
                :disabled="loading"
                @click="check(true)"
              >
                <i class="bi bi-arrow-clockwise" /> 重新检测
              </button>
              <button class="btn btn-primary" type="button" :disabled="loading || installed" @click="current = 2">
                下一步：配置数据库 <i class="bi bi-arrow-right" />
              </button>
            </div>
          </section>

          <!-- 2. 数据库配置 -->
          <section v-else-if="current === 2" class="install-step">
            <p class="install-desc">
              填写 MySQL 连接信息（数据库需提前创建好空库），验证通过后会写入
              <code>config/database.toml</code>。
            </p>

            <form class="install-form" @submit.prevent="connectDB">
              <label class="form-item">
                <span class="form-label">主机地址</span>
                <FormInput v-model="form.hostname" type="text" required placeholder="localhost" />
              </label>
              <label class="form-item">
                <span class="form-label">端口</span>
                <FormInput v-model="form.hostport" type="number" required placeholder="3306" />
              </label>
              <label class="form-item">
                <span class="form-label">数据库名</span>
                <FormInput v-model="form.database" type="text" required placeholder="需提前创建" />
              </label>
              <label class="form-item">
                <span class="form-label">用户名</span>
                <FormInput v-model="form.username" type="text" required autocomplete="off" />
              </label>
              <label class="form-item">
                <span class="form-label">密码</span>
                <FormInput
                  v-model="form.password"
                  type="password"
                  required
                  autocomplete="new-password" />
              </label>
              <!-- 用 div 而不是 label：SelectMenu 的触发器是 button，
                   label 会把点击再转发一次，导致下拉刚打开就被关掉 -->
              <div class="form-item">
                <span class="form-label">字符集</span>
                <FormSelect v-model="form.charset" :options="CHARSET_OPTIONS" />
              </div>

              <button
                class="btn btn-primary btn-block"
                :class="{ 'is-load': loading }"
                type="submit"
                :disabled="loading"
              >
                连接数据库并保存配置
              </button>
            </form>
          </section>

          <!-- 3. 初始化数据 -->
          <section v-else-if="current === 3" class="install-step">
            <p class="install-desc">接下来会创建全部数据表，并生成一个默认管理员账号。</p>
            <ul class="install-tips">
              <li><i class="bi bi-person-badge" /> 默认账号 <b>admin</b> / 密码 <b>admin123456</b>（登录后请立即修改）</li>
              <li><i class="bi bi-database-add" /> 已存在的表不会重复创建，可放心执行</li>
              <li><i class="bi bi-clock-history" /> 首次初始化可能需要十几秒，请勿关闭页面</li>
            </ul>
            <button
              class="btn btn-primary btn-block"
              :class="{ 'is-load': loading }"
              type="button"
              :disabled="loading"
              @click="initDB"
            >
              初始化数据库
            </button>
          </section>

          <!-- 4. 完成安装 -->
          <section v-else class="install-step">
            <p class="install-desc">
              最后一步：解除安装锁（删除 <code>install.lock</code>），完成后即可正常访问站点。
            </p>
            <ul class="install-tips">
              <li><i class="bi bi-shield-lock" /> 解除后安装接口会自动关闭（返回 412），无需再手动处理</li>
            </ul>
            <button
              class="btn btn-success btn-block"
              :class="{ 'is-load': loading }"
              type="button"
              :disabled="loading"
              @click="lock"
            >
              完成安装
            </button>
          </section>

          <p v-if="message.text" class="install-message" :class="`is-${message.type}`">
            <i :class="message.icon" /> {{ message.text }}
          </p>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 安装向导（/install）
 *
 * 未安装时后端 Install 中间件会把首页 302 到这里（见 app/middleware/install.go）。
 * 本页只调用 /dev/install/*，**不使用任何 /api 接口**：
 * 未安装时 /api 会被中间件整体拦住（412），加载站点数据只会产生无意义的报错
 * （所以 App.vue 也在本路由跳过 siteStore.load）。
 *
 * 与老版 public/install.html 的步骤完全一致：检查 → 连接数据库 → 初始化 → 完成。
 */
import { computed, onMounted, reactive, ref } from 'vue'
import { installCheck, installConnectDB, installInfo, installInitDB, installLock } from '@/api/install'

const steps = [
  { step: 1, label: '环境检查' },
  { step: 2, label: '数据库配置' },
  { step: 3, label: '初始化数据' },
  { step: 4, label: '完成安装' }
]

// 数据库字符集（统一走 SelectMenu，不用浏览器原生 select）
const CHARSET_OPTIONS = [
  { value: 'utf8mb4', label: 'utf8mb4（推荐）' },
  { value: 'utf8', label: 'utf8' },
  { value: 'gbk', label: 'gbk' }
]

const current = ref(1)
const loading = ref(false)
// installed：后端告诉我们「已经装过了」；locked：本次刚刚完成安装
const installed = ref(false)
const locked = ref(false)

const form = reactive({
  hostname: 'localhost',
  hostport: 3306,
  database: '',
  username: '',
  password: '',
  charset: 'utf8mb4'
})

const message = reactive({ type: '', text: '', icon: '' })

// 运行环境信息（第一步展示用，不参与安装流程；取不到就显示 —）
const env = reactive({ goos: '', goarch: '', go: '', inis: '', cpu: 0 })
const checked = ref(false)

/** finished - 结束态（已安装或刚装完）：不再显示步骤 */
const finished = computed(() => installed.value || locked.value)

/** 第一步展示的环境清单 */
const envList = computed(() => [
  { label: '操作系统', value: env.goos || '—' },
  { label: '系统架构', value: env.goarch || '—' },
  { label: 'Go 版本', value: env.go || '—' },
  { label: '程序版本', value: env.inis || '—' },
  { label: 'CPU 核数', value: env.cpu ? String(env.cpu) : '—' },
  {
    label: '安装状态',
    value: !checked.value
      ? '检测中…'
      : installed.value
        ? '已完成安装'
        : '待安装（install.lock 存在）'
  }
])

/** 读取运行环境信息（后端 /dev/install 的 INDEX，返回 system 字段） */
async function loadEnv() {
  try {
    const res = await installInfo()
    const system = res?.data?.system || {}
    env.goos = system.GOOS || ''
    env.goarch = system.GOARCH || ''
    env.go = system.go || ''
    env.inis = system.inis || ''
    env.cpu = Number(system.NumCPU || 0)
  } catch {
    // 只是展示信息，失败不影响安装流程
  }
}

function showMessage(text, type = 'info') {
  const icons = {
    success: 'bi bi-check-circle',
    danger: 'bi bi-exclamation-triangle',
    info: 'bi bi-info-circle'
  }
  message.type = type
  message.text = text
  message.icon = icons[type] || icons.info
}

/** 检查安装锁状态：data 为 true 表示已安装（安装锁已解除） */
async function check(manual = false) {
  if (loading.value) return

  loading.value = true
  try {
    const res = await installCheck()
    checked.value = true

    if (res?.data) {
      installed.value = true
      showMessage('系统已完成安装', 'success')
      return
    }

    // 停在第一步：让用户看清环境与安装状态，点「下一步」再进入数据库配置
    current.value = 1
    showMessage(manual ? '安装锁状态正常，可以继续安装' : '环境检测完成，可以继续安装', 'success')
  } catch (err) {
    showMessage(err?.msg || '检查失败：请确认服务已正常启动后重试', 'danger')
  } finally {
    loading.value = false
  }
}

/** 连接数据库并写入 config/database.toml */
async function connectDB() {
  if (loading.value) return

  loading.value = true
  showMessage('')
  try {
    const res = await installConnectDB({ ...form })
    if (res?.code === 200) {
      current.value = 3
      showMessage('数据库连接成功，配置已保存', 'success')
      return
    }
    showMessage(res?.msg || '数据库连接失败', 'danger')
  } catch (err) {
    showMessage(err?.msg || '数据库连接失败，请检查填写的信息', 'danger')
  } finally {
    loading.value = false
  }
}

/** 初始化数据表 + 默认管理员 */
async function initDB() {
  if (loading.value) return

  loading.value = true
  showMessage('正在初始化，请稍候…', 'info')
  try {
    const res = await installInitDB()
    if (res?.code === 200) {
      current.value = 4
      showMessage('数据库初始化成功，默认管理员账号已创建', 'success')
      return
    }
    showMessage(res?.msg || '数据库初始化失败', 'danger')
  } catch (err) {
    showMessage(err?.msg || '数据库初始化失败，请查看服务端日志', 'danger')
  } finally {
    loading.value = false
  }
}

/** 解除安装锁，完成安装 */
async function lock() {
  if (loading.value) return

  loading.value = true
  showMessage('')
  try {
    const res = await installLock()
    if (res?.code === 200) {
      locked.value = true
      showMessage('安装完成', 'success')
      return
    }
    showMessage(res?.msg || '解除安装锁失败', 'danger')
  } catch (err) {
    showMessage(err?.msg || '解除安装锁失败，请查看服务端日志', 'danger')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadEnv()
  check()
})
</script>

<style scoped>
/* 安装页独立于所有布局：自己撑满视口并居中 */
.install-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px 16px;
  background: var(--bg);
}
.install-box {
  width: 100%;
  max-width: 560px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* ---------- 头部品牌 ---------- */
.install-head {
  display: flex;
  align-items: center;
  gap: 14px;
}
.install-logo {
  flex-shrink: 0;
  width: 52px;
  height: 52px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
  background: linear-gradient(135deg, var(--primary), var(--primary-deep));
  color: #fff;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.5px;
}
.install-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 20px;
  font-weight: 700;
  color: var(--text);
}
.install-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-muted);
}

/* ---------- 步骤条 ---------- */
.install-steps {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  align-items: center;
  gap: 6px;
}
.install-steps-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--text-muted);
  font-size: 12px;
}
.install-steps-dot {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--bg-card);
  font-size: 12px;
  font-weight: 600;
  transition: color 0.2s, background-color 0.2s, border-color 0.2s;
}
.install-steps-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.install-steps-item.is-active {
  color: var(--primary-deep);
}
.install-steps-item.is-active .install-steps-dot {
  border-color: var(--primary);
  background: var(--primary);
  color: #fff;
}
.install-steps-item.is-done .install-steps-dot {
  border-color: var(--primary-soft);
  background: var(--accent-soft);
  color: var(--primary-deep);
}

/* ---------- 步骤内容 ---------- */
.install-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.install-step {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.install-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.8;
  color: var(--text-soft);
}
.install-desc code,
.install-tips code {
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--bg-muted);
  font-size: 12px;
}

.install-form {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.form-label {
  font-size: 12px;
  color: var(--text-muted);
}
.install-form .btn-block {
  grid-column: 1 / -1;
}

/* 第一步：环境信息清单 */
.install-env {
  margin: 0;
  padding: 0;
  list-style: none;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}
.install-env li {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 12px;
  border-radius: var(--radius);
  background: var(--bg-muted);
  min-width: 0;
}
.install-env-label {
  font-size: 12px;
  color: var(--text-muted);
}
.install-env-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 步骤内的按钮行：主次按钮并排，窄屏堆叠 */
.install-actions {
  display: flex;
  gap: 10px;
}
.install-actions .btn {
  flex: 1;
}
@media (max-width: 560px) {
  .install-actions {
    flex-direction: column-reverse;
  }
}

.install-tips {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.install-tips li {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-soft);
}
.install-tips .bi {
  color: var(--primary);
}

/* ---------- 提示信息 ---------- */
.install-message {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--radius);
  font-size: 13px;
  background: var(--bg-muted);
  color: var(--text-soft);
}
.install-message.is-success {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.install-message.is-danger {
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

/* ---------- 完成态 ---------- */
.install-done {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 10px;
}
.install-done-icon {
  font-size: 40px;
  color: var(--success);
}
.install-done-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
}
.install-done-hint {
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}
.install-done-actions {
  display: flex;
  gap: 10px;
  margin-top: 6px;
}

/* 窄屏：表单改单列 */
@media (max-width: 560px) {
  .install-form {
    grid-template-columns: 1fr;
  }
  .install-steps-label {
    display: none;
  }
  .install-steps-item {
    justify-content: center;
  }
}
</style>
