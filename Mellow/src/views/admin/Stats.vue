<template>
  <div class="stats-page">
    <!-- 实时通道状态 -->
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">数据统计</h2>
          <p class="block-desc">
            数据来自后端 WebSocket（<code>/socket</code>）实时推送，服务端按固定间隔下发
            （默认 {{ intervalText }}，由后端 <code>socket.status_interval</code> 配置；
            无管理员在线时服务端自动跳过采集）；系统资源与数据库统计仅推送给管理员，
            在线人数对所有连接广播
          </p>
        </div>
        <div class="head-actions">
          <span class="connection-badge" :class="`is-${connection}`">
            <i class="bi" :class="connIcon" />
            {{ connText }}
          </span>
          <button class="btn btn-sm" :disabled="connection === 'connecting'" @click="reconnect">
            <i class="bi bi-arrow-repeat" /> 重新连接
          </button>
          <button class="btn btn-sm" @click="paused = !paused">
            <i class="bi" :class="paused ? 'bi-play-fill' : 'bi-pause-fill'" />
            {{ paused ? '继续刷新' : '暂停刷新' }}
          </button>
        </div>
      </header>

      <div v-if="!ready" class="offline-tip">
        <i class="bi" :class="connection === 'failed' ? 'bi-exclamation-triangle' : 'bi-plug'" />
        <div>
          <p class="offline-title">
            {{ connection === 'failed' ? '自动重连已停止' : '尚未收到实时数据' }}
          </p>
          <p class="offline-desc">
            请确认：① 当前账号是管理员（系统状态仅推送给管理员）；② 反向代理已放通
            <code>/socket</code> 的 WebSocket 升级（Nginx 需要 <code>Upgrade</code> /
            <code>Connection: upgrade</code> 头）；③ 本地开发时 <code>vite.config.js</code> 已代理
            <code>/socket</code>。
          </p>
          <p v-if="connection === 'failed'" class="offline-desc">
            已连续失败，自动重连已停止以避免刷屏；修复配置后点上方「重新连接」即可。
          </p>
        </div>
      </div>
    </section>

    <!-- 资源概览：一行 2 / 3 / 4 个（断点见样式），卡片内改为纵向排列，避免全部挤在一起 -->
    <div class="resource-grid">
      <div
        v-for="item in resourceCards"
        :key="item.label"
        class="resource-card"
        :class="{ 'is-alert': item.alert }"
      >
        <div class="resource-head">
          <span class="resource-icon" :style="{ color: item.color }"><i :class="item.icon" /></span>
          <span class="resource-label">{{ item.label }}</span>
          <span v-if="item.alert" class="resource-flag">
            <i class="bi bi-exclamation-triangle" /> 超阈值
          </span>
        </div>
        <div class="resource-value">{{ item.value }}</div>
        <div v-if="item.hint" class="resource-hint" :title="item.hint">{{ item.hint }}</div>
      </div>
    </div>

    <!-- 趋势（最近 60 秒） -->
    <section class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">实时趋势</h2>
          <p class="block-desc">
            滑动窗口：最近 {{ windowText }}（每 {{ intervalText }} 一个采样点，最多 {{ MAX_POINTS }} 个）
          </p>
        </div>
      </header>

      <div class="trend-list">
        <div v-for="item in trends" :key="item.key" class="trend-item">
          <div class="trend-head">
            <span class="trend-name">
              <i :style="{ color: item.color }" :class="item.icon" /> {{ item.name }}
            </span>
            <span class="trend-value">{{ item.value }}</span>
          </div>
          <LineChart
            :series="item.series"
            :max="item.max"
            :capacity="MAX_POINTS"
            :height="64"
            :fill="item.fill"
          />
          <div v-if="item.legend" class="trend-legend">
            <span v-for="one in item.legend" :key="one.name">
              <i class="dot" :style="{ background: one.color }" />{{ one.name }} {{ one.value }}
            </span>
          </div>
        </div>
      </div>
    </section>

    <!-- 告警阈值 -->
    <section class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">告警阈值</h2>
          <p class="block-desc">
            超过阈值时本页对应卡片会高亮，并给所有管理员发站内信 + 邮件（同类告警在冷却期内不重复通知）。
            判定发生在服务端每分钟的采样里，所以即使没人打开后台也能收到告警
          </p>
        </div>
      </header>

      <div class="alert-form">
        <!-- 用 div 而不是 label：SelectMenu 的触发器是 button，
             label 会把点击再转发给 button，导致「点一下开关又立刻关上」 -->
        <div class="alert-field">
          <span class="alert-label">开启告警</span>
          <FormSelect v-model="alertForm.enabled" :options="ALERT_ON_OFF_OPTIONS" />
        </div>
        <label class="alert-field">
          <span class="alert-label">CPU 阈值 (%)</span>
          <FormInput v-model.number="alertForm.cpu" class="alert-input" type="number" min="1" max="100" />
        </label>
        <label class="alert-field">
          <span class="alert-label">内存阈值 (%)</span>
          <FormInput v-model.number="alertForm.mem" class="alert-input" type="number" min="1" max="100" />
        </label>
        <label class="alert-field">
          <span class="alert-label">磁盘阈值 (%)</span>
          <FormInput v-model.number="alertForm.disk" class="alert-input" type="number" min="1" max="100" />
        </label>
        <label class="alert-field">
          <span class="alert-label">冷却 (秒)</span>
          <FormInput
            v-model.number="alertForm.cooldown"
            class="alert-input"
            type="number"
            min="60"
            max="86400" />
        </label>
        <button class="btn btn-primary btn-sm" :disabled="alertSaving" @click="saveAlert">
          <i class="bi bi-check2" /> 保存阈值
        </button>
      </div>
    </section>

    <!-- 内容统计 -->
    <section class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">内容统计</h2>
          <p class="block-desc">
            数据库连接
            <span :class="database.connected ? 'is-ok' : 'is-bad'">
              {{ database.connected ? '正常' : '异常' }}
            </span>
            · 查询延迟 {{ database.latency || '—' }}
            <template v-if="database.error"> · {{ database.error }}</template>
          </p>
        </div>
      </header>

      <div class="count-grid">
        <div v-for="item in countItems" :key="item.key" class="count-card">
          <div class="count-head">
            <i :class="item.icon" />
            <span class="count-label">{{ item.label }}</span>
            <span class="count-total">{{ item.total ?? 0 }}</span>
          </div>
          <ul v-if="item.parts && item.parts.length" class="count-parts">
            <li v-for="part in item.parts" :key="part.label">
              <span>{{ part.label }}</span>
              <strong>{{ part.value ?? 0 }}</strong>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <div class="two-col">
      <!-- 缓存 -->
      <section class="card card-pad">
        <header class="list-head">
          <div>
            <h2 class="block-title">缓存</h2>
            <p class="block-desc">写入 / 读取一次校验连通性</p>
          </div>
        </header>
        <ul class="kv-list">
          <li>
            <span>开关</span>
            <strong :class="cache.enabled ? 'is-ok' : 'is-muted'">
              {{ cache.enabled ? '已开启' : '未开启' }}
            </strong>
          </li>
          <li>
            <span>驱动</span>
            <strong>{{ cache.type || '—' }}</strong>
          </li>
          <li>
            <span>读写</span>
            <strong :class="cache.working ? 'is-ok' : 'is-bad'">
              {{ cache.working ? '正常' : '异常' }}
            </strong>
          </li>
          <li v-if="cache.error">
            <span>错误</span>
            <strong class="is-bad">{{ cache.error }}</strong>
          </li>
        </ul>
      </section>

      <!-- 系统信息 -->
      <section class="card card-pad">
        <header class="list-head">
          <div>
            <h2 class="block-title">系统信息</h2>
            <p class="block-desc">后端运行环境</p>
          </div>
        </header>
        <ul class="kv-list">
          <li><span>应用</span><strong>{{ info.app_name || '—' }}</strong></li>
          <li><span>Go 版本</span><strong>{{ info.go_version || '—' }}</strong></li>
          <li><span>运行平台</span><strong>{{ info.os || '—' }} / {{ info.arch || '—' }}</strong></li>
          <li><span>系统</span><strong>{{ osText }}</strong></li>
          <li><span>内核</span><strong>{{ system.kernel || '—' }}</strong></li>
          <li><span>启动时间</span><strong>{{ system.boot_time || '—' }}</strong></li>
          <li><span>服务器时间</span><strong>{{ info.current_time || '—' }}</strong></li>
        </ul>
      </section>
    </div>

    <!-- 在线用户（socket 连接） -->
    <section class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">在线用户</h2>
          <p class="block-desc">
            WebSocket 实时连接数：<strong>{{ online.count }}</strong>
            （含游客；IP 已由服务端脱敏）
          </p>
        </div>
      </header>

      <EmptyState v-if="!online.users.length" icon="bi bi-people" text="当前没有 WebSocket 连接" />

      <ul v-else class="online-list">
        <li v-for="item in online.users" :key="item.id">
          <span class="online-name">
            <i class="bi bi-person-circle" />{{ item.name || '访客' }}
          </span>
          <span class="online-ip">{{ item.ip || '—' }}</span>
          <span class="online-time">{{ activeText(item.last_active) }}</span>
          <span v-if="item.id === selfId" class="online-self">当前连接</span>
          <button v-else type="button" class="online-kick" title="断开该连接" @click="kickUser(item)">
            <i class="bi bi-x-octagon" /> 踢下线
          </button>
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import LineChart from '@/components/admin/LineChart.vue'
import EmptyState from '@/components/EmptyState.vue'
import { createSocketClient } from '@/utils/socket'
import { fromNow } from '@/utils/time'
import { toast } from '@/utils/toast'
import { getStatsAlert, saveStatsAlert } from '@/api/stats'

/**
 * 数据统计（后台）
 *
 * 数据源是后端 socket 推送（app/socket/controller/status.go）：
 *   - 每秒一条系统状态（CPU / 内存 / 磁盘 / 网络 / 数据库统计 / 缓存 / 系统信息），仅管理员；
 *   - 连接与断开时广播在线人数与在线列表（所有连接）。
 * 两者 type 都是 status，靠 content 里的字段区分（见 onMessage）。
 */
const MAX_POINTS = 60

const connection = ref('closed') // connecting / open / closed
const status = ref(null)
const lastUpdate = ref(0)
const paused = ref(false)
const online = ref({ count: 0, users: [] })
const history = ref([])

// 实测推送间隔（秒）：后端间隔可配（socket.status_interval，默认 3 秒），
// 趋势图的时间窗口按实测值换算，避免改了后端配置后前端文案「撒谎」；
// 用指数滑动平均抹平网络抖动。
const sampleInterval = ref(3)
let lastPushAt = 0

let client = null

// 自己这条连接的 client_id（来自服务端的 connect 消息）：在线列表里标记「当前连接」，
// 不给它显示踢下线按钮
const selfId = ref('')

// ---------- 告警阈值 ----------
// 下拉选项（统一走 SelectMenu，不用浏览器原生 select）
const ALERT_ON_OFF_OPTIONS = [
  { value: 0, label: '关闭' },
  { value: 1, label: '开启' }
]

const alertForm = ref({ enabled: 0, cpu: 90, mem: 90, disk: 90, cooldown: 1800 })
const alertSaving = ref(false)

// ---------- 状态字段 ----------
const info = computed(() => status.value?.info || {})
const database = computed(() => status.value?.database || {})
const cache = computed(() => status.value?.cache || {})
const resource = computed(() => status.value?.resource || {})
const memory = computed(() => resource.value.memory || {})
const cpu = computed(() => resource.value.cpu || {})
const disk = computed(() => resource.value.disk || {})
const network = computed(() => resource.value.network || {})
const system = computed(() => resource.value.system || {})

const ready = computed(() => connection.value === 'open' || !!status.value)
const osText = computed(() =>
  [system.value.os, system.value.os_version].filter(Boolean).join(' ') || '—'
)

const connText = computed(
  () =>
    ({
      open: '实时连接中',
      connecting: '连接中…',
      closed: '已断开',
      failed: '自动重连已停止',
      kicked: '已被管理员断开'
    })[connection.value] || '已断开'
)
const connIcon = computed(
  () =>
    ({
      open: 'bi-broadcast',
      connecting: 'bi-arrow-repeat',
      closed: 'bi-plug',
      failed: 'bi-exclamation-triangle',
      kicked: 'bi-x-octagon'
    })[connection.value] || 'bi-plug'
)

const lastUpdateText = computed(() =>
  lastUpdate.value ? `${new Date(lastUpdate.value * 1000).toLocaleTimeString()}（${fromNow(lastUpdate.value)}）` : '—'
)

const intervalText = computed(() => {
  const seconds = Math.round(sampleInterval.value * 10) / 10
  if (seconds < 1) return '不到 1 秒'
  return Number.isInteger(seconds) ? `${seconds} 秒` : `${seconds.toFixed(1)} 秒`
})

// 趋势图横轴总时长 = 采样点数 × 实测间隔
const windowText = computed(() => {
  const seconds = Math.max(1, Math.round(MAX_POINTS * sampleInterval.value))
  return seconds < 90 ? `${seconds} 秒` : `${Math.round((seconds / 60) * 10) / 10} 分钟`
})

// ---------- 数值解析（服务端下发的是带单位的字符串） ----------
/** "12.34%" → 12.34 */
function toPercent(value) {
  const num = parseFloat(String(value ?? '').replace(/[^\d.]/g, ''))
  return Number.isFinite(num) ? num : 0
}

/** "1.23 MB/s" / "2.00 GB" → 统一换算成 KB（图表纵轴用） */
function toKB(value) {
  const match = String(value ?? '')
    .trim()
    .match(/^([\d.]+)\s*([KMGTP]?B)/i)
  if (!match) return 0
  const units = { B: 1 / 1024, KB: 1, MB: 1024, GB: 1024 * 1024, TB: 1024 * 1024 * 1024 }
  return (parseFloat(match[1]) || 0) * (units[match[2].toUpperCase()] || 1)
}

// ---------- 资源卡片 ----------
// 阈值命中判断（仅用于本页卡片高亮）：阈值取自本页表单，
// 与服务端定时采样的判定口径一致（见 model/stats.go 的 checkStatsAlert）
const alertState = computed(() => {
  const on = Number(alertForm.value.enabled) === 1
  return {
    on,
    cpu: on && toPercent(cpu.value.usage) >= Number(alertForm.value.cpu || 0),
    mem: on && toPercent(memory.value.system_usage) >= Number(alertForm.value.mem || 0),
    disk: on && toPercent(disk.value.usage) >= Number(alertForm.value.disk || 0)
  }
})

const resourceCards = computed(() => [
  {
    label: 'CPU 使用率',
    icon: 'bi bi-cpu',
    color: 'var(--primary)',
    alert: alertState.value.cpu,
    value: cpu.value.usage || '0.00%',
    hint: `${cpu.value.count || '—'} 核 · 负载 ${num(cpu.value.load_1m)} / ${num(cpu.value.load_5m)} / ${num(cpu.value.load_15m)}`
  },
  {
    label: '系统内存',
    icon: 'bi bi-memory',
    color: '#0ea5e9',
    alert: alertState.value.mem,
    value: memory.value.system_usage || '0.00%',
    hint: `已用 ${memory.value.system_used || '—'} / 共 ${memory.value.system_total || '—'}`
  },
  {
    label: '磁盘使用率',
    icon: 'bi bi-device-hdd',
    color: 'var(--warning, #b8860b)',
    alert: alertState.value.disk,
    value: disk.value.usage || '0.00%',
    hint: `${disk.value.used || '—'} / ${disk.value.total || '—'}${disk.value.fs_type ? ` · ${disk.value.fs_type}` : ''}`
  },
  {
    label: '磁盘读写',
    icon: 'bi bi-speedometer2',
    color: '#8b5cf6',
    value: `${disk.value.read_per_sec || '0 B/s'} / ${disk.value.write_per_sec || '0 B/s'}`,
    hint: `读 / 写 · 累计 ${disk.value.read || '—'} → ${disk.value.write || '—'}`
  },
  {
    label: '网络下行',
    icon: 'bi bi-arrow-down-circle',
    color: '#16a34a',
    value: network.value.down || '0.00 B/s',
    hint: `累计接收 ${network.value.total_received || '—'}`
  },
  {
    label: '网络上行',
    icon: 'bi bi-arrow-up-circle',
    color: '#f97316',
    value: network.value.up || '0.00 B/s',
    hint: `累计发送 ${network.value.total_sent || '—'}`
  },
  {
    label: 'Go 协程',
    icon: 'bi bi-diagram-3',
    color: '#64748b',
    value: resource.value.goroutines ?? '—',
    hint: `进程内存 ${memory.value.alloc || '—'} · GC ${memory.value.gc_count ?? 0} 次`
  },
  {
    label: '数据库延迟',
    icon: 'bi bi-database-check',
    color: database.value.connected ? 'var(--success, #16a34a)' : 'var(--danger)',
    value: database.value.latency || '—',
    hint: database.value.connected ? '连接正常' : database.value.error || '连接异常'
  }
])

function num(value) {
  const n = Number(value)
  return Number.isFinite(n) ? n.toFixed(2) : '0.00'
}

// ---------- 趋势 ----------
const historySeries = (key) => history.value.map((item) => item[key])

const trends = computed(() => [
  {
    key: 'cpu',
    name: 'CPU 使用率',
    icon: 'bi bi-cpu',
    color: 'var(--primary)',
    value: cpu.value.usage || '0.00%',
    max: 100,
    fill: true,
    series: [{ name: 'CPU', color: 'var(--primary)', values: historySeries('cpu') }]
  },
  {
    key: 'mem',
    name: '系统内存占用',
    icon: 'bi bi-memory',
    color: '#0ea5e9',
    value: memory.value.system_usage || '0.00%',
    max: 100,
    fill: true,
    series: [{ name: '内存', color: '#0ea5e9', values: historySeries('mem') }]
  },
  {
    key: 'disk',
    name: '磁盘使用率',
    icon: 'bi bi-device-hdd',
    color: 'var(--warning, #b8860b)',
    value: disk.value.usage || '0.00%',
    max: 100,
    fill: true,
    series: [{ name: '磁盘', color: 'var(--warning, #b8860b)', values: historySeries('disk') }]
  },
  {
    key: 'net',
    name: '网络吞吐（KB/s）',
    icon: 'bi bi-activity',
    color: '#16a34a',
    value: `${network.value.down || '0 B/s'} ↓ / ${network.value.up || '0 B/s'} ↑`,
    max: 0,
    fill: false,
    series: [
      { name: '下行', color: '#16a34a', values: historySeries('down') },
      { name: '上行', color: '#f97316', values: historySeries('up') }
    ],
    legend: [
      { name: '下行', color: '#16a34a', value: network.value.down || '0 B/s' },
      { name: '上行', color: '#f97316', value: network.value.up || '0 B/s' }
    ]
  }
])

// ---------- 内容统计 ----------
const countItems = computed(() => {
  const counts = database.value.counts || {}
  const users = counts.users || {}
  const articles = counts.articles || {}
  const moments = counts.moments || {}

  return [
    {
      key: 'users',
      label: '用户',
      icon: 'bi bi-people',
      total: users.total,
      parts: [
        { label: '近 30 天活跃', value: users.active },
        { label: '正常', value: users.normal },
        { label: '冻结', value: users.frozen }
      ]
    },
    {
      key: 'articles',
      label: '文章',
      icon: 'bi bi-file-earmark-text',
      total: articles.total,
      parts: [
        { label: '已发布', value: articles.published },
        { label: '草稿', value: articles.draft }
      ]
    },
    {
      key: 'moments',
      label: '动态',
      icon: 'bi bi-chat-square-text',
      total: moments.total,
      parts: [
        { label: '已发布', value: moments.published },
        { label: '草稿', value: moments.draft }
      ]
    },
    { key: 'comments', label: '评论', icon: 'bi bi-chat-dots', total: counts.comments },
    { key: 'pages', label: '独立页面', icon: 'bi bi-window', total: counts.pages },
    { key: 'links', label: '友链', icon: 'bi bi-link-45deg', total: counts.links },
    { key: 'banners', label: '轮播', icon: 'bi bi-images', total: counts.banners },
    { key: 'placards', label: '公告', icon: 'bi bi-megaphone', total: counts.placards },
    { key: 'tags', label: '话题', icon: 'bi bi-tags', total: counts.tags },
    { key: 'attachments', label: '附件', icon: 'bi bi-paperclip', total: counts.attachments }
  ]
})

function activeText(timestamp) {
  const value = Number(timestamp) || 0
  return value ? fromNow(value) : '—'
}

// ---------- socket 消息 ----------
function onMessage(payload) {
  // 连接成功消息里带自己的 client_id（服务端生成，形如 user_1 / guest_xxx）
  if (payload.type === 'connect') {
    if (payload.id) selfId.value = String(payload.id)
    return
  }

  if (payload.type !== 'status') return
  const content = payload.content
  if (!content || typeof content !== 'object') return

  // 在线状态（广播给所有连接）：只有 online_users / online_count
  if ('online_count' in content || Array.isArray(content.online_users)) {
    online.value = {
      count: Number(content.online_count) || 0,
      users: Array.isArray(content.online_users) ? content.online_users : []
    }
    return
  }

  // 系统状态（仅管理员）：含 info / database / cache / resource
  if (!('resource' in content) && !('database' in content)) return

  lastUpdate.value = Number(content.timestamp) || Math.floor(Date.now() / 1000)
  status.value = content

  // 估计真实推送间隔（只采信 1~300 秒的合理值，避免首次连接或长时间暂停后把间隔算飞）；
  // 暂停期间也继续估计，恢复后文案立即正确
  if (lastPushAt && lastUpdate.value > lastPushAt) {
    const gap = lastUpdate.value - lastPushAt
    if (gap >= 1 && gap <= 300) sampleInterval.value = sampleInterval.value * 0.7 + gap * 0.3
  }
  lastPushAt = lastUpdate.value

  if (paused.value) return
  history.value = [
    ...history.value,
    {
      cpu: toPercent(content.resource?.cpu?.usage),
      mem: toPercent(content.resource?.memory?.system_usage),
      disk: toPercent(content.resource?.disk?.usage),
      up: toKB(content.resource?.network?.up),
      down: toKB(content.resource?.network?.down)
    }
  ].slice(-MAX_POINTS)
}

// ---------- 告警阈值（HTTP 接口，与 socket 互不影响） ----------
async function loadAlert() {
  try {
    const res = await getStatsAlert()
    const data = res?.data || {}
    alertForm.value = {
      enabled: Number(data.enabled) === 1 ? 1 : 0,
      cpu: Number(data.cpu) || 90,
      mem: Number(data.mem) || 90,
      disk: Number(data.disk) || 90,
      cooldown: Number(data.cooldown) || 1800
    }
  } catch {
    // 读取失败保留默认值，避免表单空掉
  }
}

async function saveAlert() {
  alertSaving.value = true
  try {
    const res = await saveStatsAlert({ ...alertForm.value })
    const data = res?.data
    // 用服务端返回的「收敛后的值」回填（越界会被 clamp，避免界面与实际配置不一致）
    if (data && typeof data === 'object') {
      alertForm.value = {
        enabled: Number(data.enabled) === 1 ? 1 : 0,
        cpu: Number(data.cpu) || alertForm.value.cpu,
        mem: Number(data.mem) || alertForm.value.mem,
        disk: Number(data.disk) || alertForm.value.disk,
        cooldown: Number(data.cooldown) || alertForm.value.cooldown
      }
    }
    toast.success('告警阈值已保存')
  } catch {
    // request 拦截器已弹出错误提示
  } finally {
    alertSaving.value = false
  }
}

// 踢下线：只断开连接（不是封禁），对方可以自行重连
function kickUser(item) {
  if (!item?.id) return
  if (!window.confirm(`确定断开「${item.name || '访客'}」的当前连接吗？`)) return

  const ok = client?.send({ type: 'kick', to: item.id })
  if (ok) {
    toast.success('已发送断开指令')
  } else {
    toast.error('发送失败：实时通道未连接')
  }
}

function reconnect() {
  client?.reconnect()
}

onMounted(() => {
  // 告警阈值走 HTTP 接口，实时数据走 socket，两者互不影响
  loadAlert()

  client = createSocketClient({
    onMessage,
    onState: (state) => {
      connection.value = state
    },
    // 被管理员踢下线：socket 客户端已停止自动重连，这里只提示（仍可点「重新连接」恢复）
    onKicked: (payload) => {
      toast.warning(payload?.content || '您已被管理员断开连接')
    }
  })
  client.connect()
})

onUnmounted(() => {
  client?.close()
  client = null
})
</script>

<style scoped>
/* 页面根容器：与其它后台页一致（如 QpsWarn 的 .qps-warn-admin），
   用 flex + gap 统一各大卡片之间的间距 —— 之前漏了这条，卡片的上下间距只能靠
   个别块自己的 margin，于是出现「有的挨在一起、有的有间距」 */
.stats-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-head,
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.block-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.block-desc code {
  padding: 1px 5px;
  font-size: 11px;
  border-radius: 3px;
  background: var(--bg-muted);
}
.head-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.connection-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px;
  font-size: 12px;
  border-radius: 999px;
  border: 1px solid var(--border);
  color: var(--text-soft);
  background: var(--bg-card);
}
.connection-badge.is-open {
  color: var(--success, #16a34a);
  border-color: var(--success, #16a34a);
}
.connection-badge.is-connecting {
  color: var(--warning, #b8860b);
  border-color: var(--warning, #b8860b);
}
.connection-badge.is-closed,
.connection-badge.is-failed,
.connection-badge.is-kicked {
  color: var(--danger);
  border-color: var(--danger);
}
.offline-tip {
  display: flex;
  gap: 10px;
  padding: 12px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 12px;
}
.offline-tip .bi {
  font-size: 18px;
  color: var(--text-light, var(--text-muted));
}
.offline-title {
  margin: 0 0 4px;
  font-weight: 600;
  color: var(--text);
}
.offline-desc {
  margin: 0;
  line-height: 1.8;
}

/* 资源概览卡片：固定一行 2 / 3 / 4 个（断点见下）
   之前用 repeat(auto-fit, minmax(180px, 1fr))，宽屏会被塞进 5~6 列，卡片又窄又挤 */
.resource-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
@media (min-width: 900px) {
  .resource-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (min-width: 1280px) {
  .resource-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
.resource-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}
.resource-head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.resource-icon {
  font-size: 15px;
  color: var(--text-light);
}
.resource-label {
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.resource-value {
  font-size: 19px;
  font-weight: 600;
  line-height: 1.25;
  word-break: break-all;
}
.resource-hint {
  font-size: 11px;
  color: var(--text-light, var(--text-muted));
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 命中告警阈值：整卡描红 + 右上角角标 */
.resource-card.is-alert {
  border-color: var(--danger);
  background: var(--danger-wash, rgba(220, 53, 69, 0.08));
}
.resource-flag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin-left: auto;
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: 10px;
  border-radius: 3px;
  color: var(--danger);
  background: var(--danger-wash, rgba(220, 53, 69, 0.16));
}

/* 趋势 */
.trend-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 16px;
}
.trend-item {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}
.trend-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}
.trend-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-soft);
}
.trend-value {
  font-size: 12px;
  font-weight: 600;
}
.trend-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 8px;
  font-size: 11px;
  color: var(--text-muted);
}
.trend-legend .dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  margin-right: 4px;
  border-radius: 50%;
}

/* 内容统计 */
.count-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 10px;
}
.count-card {
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}
.count-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-soft);
}
.count-total {
  margin-left: auto;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.count-parts {
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 11px;
  color: var(--text-muted);
}
.count-parts li {
  display: flex;
  justify-content: space-between;
}
.count-parts strong {
  color: var(--text);
  font-weight: 600;
}

/* 两列：缓存 / 系统信息 */
.two-col {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}
.kv-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: 12px;
}
.kv-list li {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}
.kv-list span {
  color: var(--text-muted);
}
.kv-list strong {
  font-weight: 600;
  text-align: right;
  word-break: break-all;
}
.is-ok {
  color: var(--success, #16a34a);
}
.is-bad {
  color: var(--danger);
}
.is-muted {
  color: var(--text-muted);
}

/* 在线用户 */
.online-list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.online-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  font-size: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}
.online-name {
  flex: 1;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.online-ip {
  flex-shrink: 0;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.online-time {
  flex-shrink: 0;
  width: 90px;
  text-align: right;
  color: var(--text-light, var(--text-muted));
}
.online-self {
  flex-shrink: 0;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.online-kick {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 11px;
  border: 1px solid var(--border);
  border-radius: 3px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
}
.online-kick:hover {
  color: var(--danger);
  border-color: var(--danger);
}

/* 告警阈值表单 */
.alert-form {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px;
}
.alert-field {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.alert-label {
  font-size: 12px;
  color: var(--text-muted);
}
.alert-input {
  width: 130px;
  padding: 6px 10px;
  font-size: 13px;
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-input, var(--bg-card));
  outline: none;
}
.alert-input:focus {
  border-color: var(--primary);
}
</style>
