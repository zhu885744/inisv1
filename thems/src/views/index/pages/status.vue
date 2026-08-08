<template>
  <div class="status-page mt-3">
    <!-- ========== 页头卡片 ========== -->
    <div class="card mb-3 border">
      <div class="card-body">
        <div class="d-flex flex-wrap align-items-center justify-content-between gap-3">
          <div>
            <h1 class="h4 mb-1 d-flex align-items-center gap-2">
              <i class="bi bi-activity text-primary"></i>服务器状态
            </h1>
            <p class="text-muted small mb-0">通过 WebSocket 实时推送服务器运行数据</p>
          </div>
          <div class="d-flex align-items-center gap-3">
            <div class="d-flex align-items-center gap-2">
              <span class="status-indicator rounded-circle" :class="statusClass"></span>
              <span class="fw-medium" :class="statusTextClass">{{ statusText }}</span>
            </div>
            <button
              class="btn btn-outline-primary btn-sm"
              @click="handleReconnect"
              :disabled="socketStore.isConnecting"
            >
              <i class="bi bi-arrow-repeat me-1"></i>重连
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 概览卡片行 ========== -->
    <div class="row g-3 mb-3">
      <div class="col-6 col-md-3">
        <div class="card h-100 border">
          <div class="card-body d-flex flex-column">
            <div class="d-flex align-items-center gap-2 text-muted mb-2 border-bottom pb-2">
              <i class="bi bi-people-fill text-primary fs-5"></i>
              <small class="fw-medium">在线用户</small>
            </div>
            <div class="fs-2 fw-bold mt-1">{{ socketStore.onlineCount }}</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card h-100 border">
          <div class="card-body d-flex flex-column">
            <div class="d-flex align-items-center gap-2 text-muted mb-2 border-bottom pb-2">
              <i class="bi bi-diagram-3-fill text-info fs-5"></i>
              <small class="fw-medium">连接时长</small>
            </div>
            <div class="fs-2 fw-bold mt-1">{{ formatDuration(socketStore.connectedAt) }}</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card h-100 border">
          <div class="card-body d-flex flex-column">
            <div class="d-flex align-items-center gap-2 text-muted mb-2 border-bottom pb-2">
              <i class="bi bi-heart-pulse-fill fs-5" :class="healthClass"></i>
              <small class="fw-medium">健康状态</small>
            </div>
            <div class="fs-2 fw-bold mt-1" :class="healthClass">{{ healthText }}</div>
          </div>
        </div>
      </div>
      <div class="col-6 col-md-3">
        <div class="card h-100 border">
          <div class="card-body d-flex flex-column">
            <div class="d-flex align-items-center gap-2 text-muted mb-2 border-bottom pb-2">
              <i class="bi bi-clock-fill text-secondary fs-5"></i>
              <small class="fw-medium">系统时间</small>
            </div>
            <div class="fs-2 fw-bold mt-1 font-monospace">{{ currentTime }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 应用信息 ========== -->
    <div class="card mb-3 border" v-if="socketStore.appInfo.app_name">
      <div class="card-header bg-transparent border-bottom py-3">
        <h2 class="h6 mb-0 fw-bold d-flex align-items-center gap-2">
          <i class="bi bi-info-circle-fill text-primary"></i>应用信息
        </h2>
      </div>
      <div class="card-body p-0">
        <div class="list-group list-group-flush">
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">应用名称</span>
            <span class="fw-semibold">{{ socketStore.appInfo.app_name }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">Go 版本</span>
            <span class="fw-semibold">{{ socketStore.appInfo.go_version }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">操作系统 / 架构</span>
            <span class="fw-semibold">{{ socketStore.appInfo.os }} / {{ socketStore.appInfo.arch }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">CPU 核心数</span>
            <span class="fw-semibold">{{ socketStore.appInfo.cpu_count }} 核</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">协程数量</span>
            <span class="fw-semibold">{{ socketStore.systemStatus?.resource?.goroutines || socketStore.appInfo.goroutines }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 数据库 + 缓存 ========== -->
    <div class="row g-3 mb-3">
      <!-- 数据库 -->
      <div class="col-lg-6" v-if="socketStore.databaseStatus.connected !== undefined">
        <div class="card h-100 border">
          <div class="card-header bg-transparent border-bottom d-flex flex-wrap justify-content-between align-items-center py-3">
            <h2 class="h6 mb-0 fw-bold d-flex align-items-center gap-2">
              <i class="bi bi-database-fill text-primary"></i>数据库
            </h2>
            <div>
              <span class="badge rounded-pill" :class="socketStore.databaseStatus.connected ? 'bg-success' : 'bg-danger'">
                {{ socketStore.databaseStatus.connected ? '已连接' : '已断开' }}
              </span>
              <span class="small text-muted ms-2 d-none d-sm-inline">延迟 {{ socketStore.databaseStatus.latency || '-' }}</span>
            </div>
          </div>
          <div class="card-body">
            <!-- 错误提示 -->
            <div v-if="socketStore.databaseStatus.error" class="alert alert-danger py-2 small mb-3">
              <i class="bi bi-exclamation-triangle-fill me-1"></i>
              {{ socketStore.databaseStatus.error }}
            </div>

            <!-- 统计数据网格 -->
            <div v-if="socketStore.databaseStatus.counts" class="row g-2">
              <div class="col-sm-6">
                <div class="border rounded p-3 h-100">
                  <div class="d-flex justify-content-between align-items-center mb-1">
                    <small class="text-muted">用户总数</small>
                  </div>
                  <div class="h5 mb-1">{{ socketStore.userStats.total ?? 0 }}</div>
                  <small>
                    <span class="text-success">正常 {{ socketStore.userStats.normal ?? 0 }}</span>
                    <span class="mx-1 text-muted">·</span>
                    <span class="text-warning">冻结 {{ socketStore.userStats.frozen ?? 0 }}</span>
                  </small>
                </div>
              </div>
              <div class="col-sm-6">
                <div class="border rounded p-3 h-100">
                  <div class="d-flex justify-content-between align-items-center mb-1">
                    <small class="text-muted">活跃用户 (30天)</small>
                  </div>
                  <div class="h5 mb-1">{{ socketStore.userStats.active ?? 0 }}</div>
                  <small class="text-muted">最近 30 天内登录</small>
                </div>
              </div>
              <div class="col-sm-6">
                <div class="border rounded p-3 h-100">
                  <div class="d-flex justify-content-between align-items-center mb-1">
                    <small class="text-muted">文章</small>
                  </div>
                  <div class="h5 mb-1">{{ socketStore.articleStats.total ?? 0 }}</div>
                  <small>
                    <span class="text-success">已发布 {{ socketStore.articleStats.published ?? 0 }}</span>
                    <span class="mx-1 text-muted">·</span>
                    <span class="text-muted">草稿 {{ socketStore.articleStats.draft ?? 0 }}</span>
                  </small>
                </div>
              </div>
              <div class="col-sm-6">
                <div class="border rounded p-3 h-100">
                  <div class="d-flex justify-content-between align-items-center mb-1">
                    <small class="text-muted">动态</small>
                  </div>
                  <div class="h5 mb-1">{{ socketStore.momentStats.total ?? 0 }}</div>
                  <small>
                    <span class="text-success">已发布 {{ socketStore.momentStats.published ?? 0 }}</span>
                    <span class="mx-1 text-muted">·</span>
                    <span class="text-muted">草稿 {{ socketStore.momentStats.draft ?? 0 }}</span>
                  </small>
                </div>
              </div>
            </div>

            <!-- 其他计数（横向列表） -->
            <div v-if="socketStore.databaseStatus.counts" class="mt-3">
              <div class="d-flex flex-wrap gap-2">
                <div class="border rounded px-3 py-2 flex-fill text-center">
                  <div class="small text-muted">评论</div>
                  <div class="fw-bold">{{ formatNumber(socketStore.databaseStatus.counts.comments) }}</div>
                </div>
                <div class="border rounded px-3 py-2 flex-fill text-center">
                  <div class="small text-muted">附件</div>
                  <div class="fw-bold">{{ formatNumber(socketStore.databaseStatus.counts.attachments) }}</div>
                </div>
                <div class="border rounded px-3 py-2 flex-fill text-center">
                  <div class="small text-muted">标签</div>
                  <div class="fw-bold">{{ formatNumber(socketStore.databaseStatus.counts.tags) }}</div>
                </div>
                <div class="border rounded px-3 py-2 flex-fill text-center">
                  <div class="small text-muted">页面</div>
                  <div class="fw-bold">{{ formatNumber(socketStore.databaseStatus.counts.pages) }}</div>
                </div>
                <div class="border rounded px-3 py-2 flex-fill text-center">
                  <div class="small text-muted">友链</div>
                  <div class="fw-bold">{{ formatNumber(socketStore.databaseStatus.counts.links) }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 缓存 -->
      <div class="col-lg-6" v-if="socketStore.cacheStatus.enabled !== undefined">
        <div class="card h-100 border">
          <div class="card-header bg-transparent border-bottom py-3">
            <h2 class="h6 mb-0 fw-bold d-flex align-items-center gap-2">
              <i class="bi bi-hdd-rack-fill text-info"></i>缓存服务
            </h2>
          </div>
          <div class="card-body">
            <div class="d-flex flex-wrap align-items-center gap-3 mb-3">
              <div>
                <span class="badge rounded-pill me-2" :class="socketStore.cacheStatus.enabled ? 'bg-primary' : 'bg-secondary'">
                  {{ socketStore.cacheStatus.enabled ? '已启用' : '已禁用' }}
                </span>
                <span class="badge rounded-pill bg-info me-2" v-if="socketStore.cacheStatus.type">
                  {{ socketStore.cacheStatus.type.toUpperCase() }}
                </span>
                <span class="badge rounded-pill" :class="socketStore.cacheStatus.working ? 'bg-success' : 'bg-danger'">
                  {{ socketStore.cacheStatus.working ? '工作正常' : '工作异常' }}
                </span>
              </div>
            </div>

            <div v-if="socketStore.cacheStatus.error" class="alert alert-danger py-2 small mb-0">
              <i class="bi bi-exclamation-triangle-fill me-1"></i>
              {{ socketStore.cacheStatus.error }}
            </div>
            <div v-else class="text-center py-5">
              <i class="bi bi-check-circle-fill text-success display-5 d-block mb-3"></i>
              <p class="mb-0 text-muted">缓存服务运行正常</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 系统资源 ========== -->
    <div class="card mb-3 border" v-if="socketStore.resourceUsage.memory || socketStore.resourceUsage.cpu">
      <div class="card-header bg-transparent border-bottom py-3">
        <h2 class="h6 mb-0 fw-bold d-flex align-items-center gap-2">
          <i class="bi bi-cpu-fill text-warning"></i>系统资源
        </h2>
      </div>
      <div class="card-body p-0">
        <div class="list-group list-group-flush">
          <!-- CPU -->
          <div class="list-group-item py-3" v-if="socketStore.cpuInfo.usage">
            <div class="d-flex flex-wrap align-items-center justify-content-between mb-2">
              <div class="d-flex align-items-center gap-2">
                <i class="bi bi-cpu-fill text-warning"></i>
                <span class="fw-semibold">CPU</span>
                <span class="badge text-secondary border">{{ socketStore.cpuInfo.count }} 核</span>
              </div>
              <span class="fw-bold" :class="usageClass(cpuUsagePercent)">{{ socketStore.cpuInfo.usage || '0%' }}</span>
            </div>
            <div class="progress mb-2" style="height: 0.5rem;">
              <div class="progress-bar" :class="progressClass(cpuUsagePercent)" :style="{ width: socketStore.cpuInfo.usage || '0%' }" role="progressbar"></div>
            </div>
            <div class="small text-muted mb-1">型号: {{ socketStore.cpuInfo.model || '-' }}</div>
            <div class="d-flex flex-wrap gap-3 small">
              <span><i class="bi bi-graph-up me-1"></i>1分负载 <b>{{ socketStore.cpuInfo.load_1m ?? '-' }}</b></span>
              <span><i class="bi bi-graph-up me-1"></i>5分 <b>{{ socketStore.cpuInfo.load_5m ?? '-' }}</b></span>
              <span><i class="bi bi-graph-up me-1"></i>15分 <b>{{ socketStore.cpuInfo.load_15m ?? '-' }}</b></span>
            </div>
          </div>

          <!-- 内存 -->
          <div class="list-group-item py-3" v-if="socketStore.memoryInfo.system_usage">
            <div class="d-flex flex-wrap align-items-center justify-content-between mb-2">
              <div class="d-flex align-items-center gap-2">
                <i class="bi bi-boxes text-primary"></i>
                <span class="fw-semibold">系统内存</span>
              </div>
              <span class="fw-bold" :class="usageClass(memoryUsagePercent)">{{ socketStore.memoryInfo.system_usage || '0%' }}</span>
            </div>
            <div class="progress mb-2" style="height: 0.5rem;">
              <div class="progress-bar" :class="progressClass(memoryUsagePercent)" :style="{ width: socketStore.memoryInfo.system_usage || '0%' }" role="progressbar"></div>
            </div>
            <div class="d-flex justify-content-between small text-muted mb-2">
              <span>已用 <b class="text-body">{{ socketStore.memoryInfo.system_used || '-' }}</b></span>
              <span>总量 <b class="text-body">{{ socketStore.memoryInfo.system_total || '-' }}</b></span>
            </div>
            <div class="d-flex flex-wrap gap-3 small">
              <span class="text-info"><i class="bi bi-box-seam me-1"></i>Go内存 <b>{{ socketStore.memoryInfo.alloc || '-' }}</b></span>
              <span class="text-secondary"><i class="bi bi-recycle me-1"></i>GC <b>{{ socketStore.memoryInfo.gc_count ?? 0 }}</b> 次</span>
            </div>
          </div>

          <!-- 磁盘 -->
          <div class="list-group-item py-3" v-if="socketStore.diskInfo.total">
            <div class="d-flex flex-wrap align-items-center justify-content-between mb-2">
              <div class="d-flex align-items-center gap-2">
                <i class="bi bi-device-hdd-fill text-success"></i>
                <span class="fw-semibold">磁盘</span>
                <span class="badge text-secondary border">{{ socketStore.diskInfo.fs_type || '' }}</span>
              </div>
              <span class="fw-bold" :class="usageClass(diskUsagePercent)">{{ socketStore.diskInfo.usage || '0%' }}</span>
            </div>
            <div class="progress mb-2" style="height: 0.5rem;">
              <div class="progress-bar" :class="progressClass(diskUsagePercent)" :style="{ width: socketStore.diskInfo.usage || '0%' }" role="progressbar"></div>
            </div>
            <div class="d-flex justify-content-between small text-muted mb-2 flex-wrap">
              <span>已用 <b class="text-body">{{ socketStore.diskInfo.used || '-' }}</b></span>
              <span>可用 <b class="text-body">{{ socketStore.diskInfo.free || '-' }}</b></span>
              <span>总量 <b class="text-body">{{ socketStore.diskInfo.total || '-' }}</b></span>
            </div>
            <div class="d-flex flex-wrap gap-3 small">
              <span class="text-primary"><i class="bi bi-download me-1"></i>读 <b>{{ socketStore.diskInfo.read_per_sec || '-' }}/s</b></span>
              <span class="text-danger"><i class="bi bi-upload me-1"></i>写 <b>{{ socketStore.diskInfo.write_per_sec || '-' }}/s</b></span>
              <span>IO延迟 <b>{{ socketStore.diskInfo.io_latency || '-' }}</b></span>
            </div>
          </div>

          <!-- 网络 -->
          <div class="list-group-item py-3" v-if="socketStore.networkInfo.bytes_sent">
            <div class="d-flex flex-wrap align-items-center justify-content-between mb-2">
              <div class="d-flex align-items-center gap-2">
                <i class="bi bi-bar-chart-fill text-info"></i>
                <span class="fw-semibold">网络</span>
              </div>
              <div class="d-flex gap-3 small fw-medium">
                <span class="text-success"><i class="bi bi-arrow-up me-1"></i>{{ socketStore.networkInfo.up || '-' }}/s</span>
                <span class="text-primary"><i class="bi bi-arrow-down me-1"></i>{{ socketStore.networkInfo.down || '-' }}/s</span>
              </div>
            </div>
            <div class="row g-2 small">
              <div class="col-6">
                <div class="text-muted">总发送</div>
                <div class="fw-semibold">{{ socketStore.networkInfo.total_sent || '-' }}</div>
              </div>
              <div class="col-6">
                <div class="text-muted">总接收</div>
                <div class="fw-semibold">{{ socketStore.networkInfo.total_received || '-' }}</div>
              </div>
              <div class="col-6">
                <div class="text-muted">发送包数</div>
                <div class="fw-semibold">{{ formatNumber(socketStore.networkInfo.packets_sent) }}</div>
              </div>
              <div class="col-6">
                <div class="text-muted">接收包数</div>
                <div class="fw-semibold">{{ formatNumber(socketStore.networkInfo.packets_recv) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 系统信息 ========== -->
    <div class="card mb-3 border" v-if="socketStore.systemInfo.os || socketStore.systemStatus?.info?.current_time">
      <div class="card-header bg-transparent border-bottom py-3">
        <h2 class="h6 mb-0 fw-bold d-flex align-items-center gap-2">
          <i class="bi bi-window-stack text-secondary"></i>系统信息
        </h2>
      </div>
      <div class="card-body p-0">
        <div class="list-group list-group-flush">
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">操作系统</span>
            <span class="fw-semibold">{{ socketStore.systemInfo.os || '-' }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">系统版本</span>
            <span class="fw-semibold">{{ socketStore.systemInfo.os_version || '-' }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">内核版本</span>
            <span class="fw-semibold">{{ socketStore.systemInfo.kernel || '-' }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">启动时间</span>
            <span class="fw-semibold">{{ socketStore.systemInfo.boot_time || '-' }}</span>
          </div>
          <div class="list-group-item d-flex flex-wrap justify-content-between align-items-center py-3">
            <span class="text-muted small">服务端时间</span>
            <span class="fw-semibold">{{ socketStore.appInfo.current_time || '-' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ========== 加载中 ========== -->
    <div class="card mb-3 border" v-if="!socketStore.systemStatus && socketStore.isConnected">
      <div class="card-body text-center py-5">
        <div class="spinner-border text-primary mb-3" style="width: 2.5rem; height: 2.5rem;" role="status">
          <span class="visually-hidden">加载中...</span>
        </div>
        <p class="text-muted mb-0">正在接收服务器状态数据，稍后刷新...</p>
      </div>
    </div>

    <!-- ========== 未连接 ========== -->
    <div class="card mb-3 border border-danger" v-if="!socketStore.isConnected && !socketStore.isConnecting">
      <div class="card-body text-center py-5 px-4">
        <i class="bi bi-plug text-secondary display-4 d-block mb-3"></i>
        <h3 class="h5 mb-3 fw-bold">Socket 未连接</h3>
        <p class="text-muted mb-4">无法实时获取服务器状态，请检查网络或手动重连</p>
        <button class="btn btn-primary px-4" @click="handleReconnect">
          <i class="bi bi-arrow-repeat me-2"></i>重新连接
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useSocketStore } from '@/store/socket'

const socketStore = useSocketStore()

const currentTime = ref('')
let timeTimer = null

const updateCurrentTime = () => {
  const now = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  currentTime.value = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

// 连接状态
const statusText = computed(() => {
  switch (socketStore.connectionState) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中...'
    case 'error': return '连接错误'
    default: return '已断开'
  }
})
const statusClass = computed(() => {
  switch (socketStore.connectionState) {
    case 'connected': return 'status-connected'
    case 'connecting': return 'status-connecting'
    case 'error': return 'status-error'
    default: return 'status-disconnected'
  }
})
const statusTextClass = computed(() => {
  switch (socketStore.connectionState) {
    case 'connected': return 'text-success'
    case 'connecting': return 'text-warning'
    case 'error': return 'text-danger'
    default: return 'text-secondary'
  }
})

// 健康状态
const healthText = computed(() => {
  const s = socketStore.systemHealth
  if (s === 'healthy') return '健康'
  if (s === 'unhealthy') return '异常'
  return '未知'
})
const healthClass = computed(() => {
  const s = socketStore.systemHealth
  if (s === 'healthy') return 'text-success'
  if (s === 'unhealthy') return 'text-danger'
  return 'text-secondary'
})

// 使用率解析与分级
const parsePercent = (str) => {
  if (!str) return 0
  const num = parseFloat(String(str).replace('%', ''))
  return isNaN(num) ? 0 : num
}
const cpuUsagePercent = computed(() => parsePercent(socketStore.cpuInfo.usage))
const memoryUsagePercent = computed(() => parsePercent(socketStore.memoryInfo.system_usage))
const diskUsagePercent = computed(() => parsePercent(socketStore.diskInfo.usage))

const usageClass = (percent) => {
  if (percent >= 90) return 'text-danger'
  if (percent >= 70) return 'text-warning'
  if (percent >= 50) return 'text-info'
  return 'text-success'
}
const progressClass = (percent) => {
  if (percent >= 90) return 'bg-danger'
  if (percent >= 70) return 'bg-warning'
  if (percent >= 50) return 'bg-info'
  return 'bg-success'
}

// 连接时长格式化
const formatDuration = (startTime) => {
  if (!startTime) return '-'
  const seconds = Math.floor((Date.now() - startTime) / 1000)
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  if (seconds < 86400) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}时${m}分`
  }
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  return `${d}天${h}时`
}

// 数字格式化（万/亿）
const formatNumber = (num) => {
  if (num === null || num === undefined || num === '') return '0'
  const n = Number(num)
  if (isNaN(n)) return String(num)
  if (n >= 100000000) return (n / 100000000).toFixed(1) + '亿'
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  return n.toString()
}

const handleReconnect = () => {
  socketStore.reconnect()
}

onMounted(() => {
  updateCurrentTime()
  timeTimer = setInterval(updateCurrentTime, 1000)
})
onUnmounted(() => {
  if (timeTimer) {
    clearInterval(timeTimer)
    timeTimer = null
  }
})
</script>

<style scoped>
/* 状态指示灯 */
.status-indicator {
  width: 10px;
  height: 10px;
}
.status-connected {
  background-color: #198754;
  box-shadow: 0 0 0 3px rgba(25, 135, 84, 0.2);
}
.status-connecting {
  background-color: #ffc107;
  animation: blink 1s infinite;
}
.status-error {
  background-color: #dc3545;
  box-shadow: 0 0 0 3px rgba(220, 53, 69, 0.2);
}
.status-disconnected {
  background-color: #6c757d;
}
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
</style>