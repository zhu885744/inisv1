<template>
  <div class="admin-dashboard">
    <!-- 顶部状态栏 -->
    <div class="status-bar">
      <el-row :gutter="16">
        <el-col :xs="24" :sm="12" :md="6">
          <div class="status-item" :class="connected ? 'status-success' : 'status-danger'">
            <el-icon class="status-icon" :size="24">
              <component :is="connected ? 'Connection' : 'CircleClose'" />
            </el-icon>
            <div class="status-content">
              <div class="status-label">连接状态</div>
              <div class="status-value">{{ connected ? '已连接' : '未连接' }}</div>
            </div>
            <el-button v-if="!connected" type="primary" size="small" @click="reconnect" :disabled="isReconnecting">
              {{ isReconnecting ? '重连中...' : '重连' }}
            </el-button>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <div class="status-item status-info">
            <el-icon class="status-icon" :size="24"><User /></el-icon>
            <div class="status-content">
              <div class="status-label">在线人数</div>
              <div class="status-value">{{ onlineCount }} 人</div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <div class="status-item status-primary">
            <el-icon class="status-icon" :size="24"><Timer /></el-icon>
            <div class="status-content">
              <div class="status-label">刷新频率</div>
              <div class="status-value">{{ refreshRate }} 秒</div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="6">
          <div class="status-item" :class="systemHealth">
            <el-icon class="status-icon" :size="24">
              <component :is="systemHealth === 'status-success' ? 'CircleCheck' : 'Warning'" />
            </el-icon>
            <div class="status-content">
              <div class="status-label">系统状态</div>
              <div class="status-value">{{ systemHealthText }}</div>
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 系统概览 -->
    <div class="section-title">
      <el-icon><InfoFilled /></el-icon>
      <span>系统概览</span>
    </div>

    <el-row :gutter="16" class="mb-4">
      <!-- 系统基本信息 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><Cpu /></el-icon>
              <span>基本信息</span>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="应用名称">
              <el-tag size="small">{{ systemInfoParsed?.app_name || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Go 版本">
              {{ systemInfoParsed?.go_version || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="操作系统">
              {{ systemInfoParsed?.os || '-' }} {{ systemInfoParsed?.arch || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="CPU 核心">
              {{ systemInfoParsed?.cpu_count || '-' }}
            </el-descriptions-item>
            <el-descriptions-item label="协程数">
              <el-tag size="small" type="info">{{ systemInfoParsed?.goroutines || '-' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="当前时间">
              {{ systemInfoParsed?.current_time || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 数据库状态 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><Connection /></el-icon>
              <span>数据库状态</span>
              <el-tag size="small" :type="databaseStatusParsed?.connected ? 'success' : 'danger'" class="ml-auto">
                {{ databaseStatusParsed?.connected ? '正常' : '异常' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="连接状态">
              <el-tag size="small" :type="databaseStatusParsed?.connected ? 'success' : 'danger'">
                {{ databaseStatusParsed?.connected ? '已连接' : '未连接' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="响应时间">
              <span :class="databaseStatusParsed?.latency ? 'text-success' : ''">
                {{ databaseStatusParsed?.latency || '-' }}
              </span>
            </el-descriptions-item>
          </el-descriptions>
          <el-divider content-position="left">数据统计</el-divider>
          <el-row :gutter="12">
            <el-col :span="6" v-for="(item, key) in databaseCounts" :key="key">
              <div class="stat-item">
                <div class="stat-value">{{ item.value }}</div>
                <div class="stat-label">{{ item.label }}</div>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </el-col>
    </el-row>

    <!-- 数据统计 -->
    <div class="section-title">
      <el-icon><TrendCharts /></el-icon>
      <span>数据统计</span>
    </div>

    <el-row :gutter="16" class="mb-4">
      <!-- 用户统计 -->
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><User /></el-icon>
              <span>用户统计</span>
            </div>
          </template>
          <div class="stats-grid">
            <div class="stat-box">
              <div class="stat-num">{{ userStats.total }}</div>
              <div class="stat-name">用户总数</div>
            </div>
            <div class="stat-box stat-success">
              <div class="stat-num">{{ userStats.active }}</div>
              <div class="stat-name">活跃用户</div>
            </div>
            <div class="stat-box stat-primary">
              <div class="stat-num">{{ userStats.normal }}</div>
              <div class="stat-name">正常状态</div>
            </div>
            <div class="stat-box stat-danger">
              <div class="stat-num">{{ userStats.frozen }}</div>
              <div class="stat-name">冻结用户</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 文章统计 -->
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><Document /></el-icon>
              <span>文章统计</span>
            </div>
          </template>
          <div class="stats-grid">
            <div class="stat-box">
              <div class="stat-num">{{ articleStats.total }}</div>
              <div class="stat-name">文章总数</div>
            </div>
            <div class="stat-box stat-warning">
              <div class="stat-num">{{ articleStats.draft }}</div>
              <div class="stat-name">草稿</div>
            </div>
            <div class="stat-box stat-success">
              <div class="stat-num">{{ articleStats.published }}</div>
              <div class="stat-name">已发布</div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 动态统计 -->
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><ChatDotSquare /></el-icon>
              <span>动态统计</span>
            </div>
          </template>
          <div class="stats-grid">
            <div class="stat-box">
              <div class="stat-num">{{ momentStats.total }}</div>
              <div class="stat-name">动态总数</div>
            </div>
            <div class="stat-box stat-warning">
              <div class="stat-num">{{ momentStats.draft }}</div>
              <div class="stat-name">草稿</div>
            </div>
            <div class="stat-box stat-success">
              <div class="stat-num">{{ momentStats.published }}</div>
              <div class="stat-name">已发布</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mb-4">
      <!-- 附件统计 -->
      <el-col :xs="24" :sm="8">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><FolderOpened /></el-icon>
              <span>附件统计</span>
            </div>
          </template>
          <div class="stats-grid">
            <div class="stat-box">
              <div class="stat-num">{{ attachmentCount }}</div>
              <div class="stat-name">附件总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 缓存状态 -->
    <el-card shadow="hover" class="mb-4">
      <template #header>
        <div class="card-header">
          <el-icon><Tickets /></el-icon>
          <span>缓存服务</span>
        </div>
      </template>
      <el-row :gutter="24">
        <el-col :xs="24" :sm="8">
          <div class="cache-status-item">
            <el-icon :size="32" :class="cacheStatusParsed?.enabled ? 'text-success' : 'text-muted'">
              <component :is="cacheStatusParsed?.enabled ? 'CircleCheckFilled' : 'CircleCloseFilled'" />
            </el-icon>
            <div class="cache-status-content">
              <div class="cache-status-label">启用状态</div>
              <div class="cache-status-value">{{ cacheStatusParsed?.enabled ? '已启用' : '未启用' }}</div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="8">
          <div class="cache-status-item">
            <el-icon :size="32" class="text-primary">
              <Box />
            </el-icon>
            <div class="cache-status-content">
              <div class="cache-status-label">缓存类型</div>
              <div class="cache-status-value">
                <el-tag size="small">{{ cacheStatusParsed?.type?.toUpperCase() || '-' }}</el-tag>
              </div>
            </div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="8">
          <div class="cache-status-item">
            <el-icon :size="32" :class="cacheStatusParsed?.working ? 'text-success' : 'text-danger'">
              <component :is="cacheStatusParsed?.working ? 'CircleCheckFilled' : 'CircleCloseFilled'" />
            </el-icon>
            <div class="cache-status-content">
              <div class="cache-status-label">工作状态</div>
              <div class="cache-status-value">{{ cacheStatusParsed?.working ? '正常运行' : '异常' }}</div>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 系统资源 -->
    <div class="section-title">
      <el-icon><Monitor /></el-icon>
      <span>系统资源</span>
    </div>

    <el-row :gutter="16" class="mb-4">
      <!-- CPU 状态 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card resource-card">
          <template #header>
            <div class="card-header">
              <el-icon><Cpu /></el-icon>
              <span>CPU</span>
              <span class="ml-auto text-muted">{{ systemResourcesParsed?.cpu?.usage || '0%' }}</span>
            </div>
          </template>
          <el-progress
            :percentage="parseFloat(systemResourcesParsed?.cpu?.usage || 0)"
            :color="getProgressColor(systemResourcesParsed?.cpu?.usage)"
            :stroke-width="20"
          />
          <el-divider />
          <el-descriptions :column="3" border size="small">
            <el-descriptions-item label="CPU 核心">{{ systemResourcesParsed?.cpu?.count || '-' }}</el-descriptions-item>
            <el-descriptions-item label="1分钟负载">{{ systemResourcesParsed?.cpu?.load_1m || '-' }}</el-descriptions-item>
            <el-descriptions-item label="5分钟负载">{{ systemResourcesParsed?.cpu?.load_5m || '-' }}</el-descriptions-item>
            <el-descriptions-item label="15分钟负载" :span="3">{{ systemResourcesParsed?.cpu?.load_15m || '-' }}</el-descriptions-item>
            <el-descriptions-item label="CPU 型号" :span="3">
              <span class="text-truncate">{{ systemResourcesParsed?.cpu?.model || '-' }}</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 内存状态 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card resource-card">
          <template #header>
            <div class="card-header">
              <el-icon><Monitor /></el-icon>
              <span>内存</span>
              <span class="ml-auto text-muted">{{ systemResourcesParsed?.memory?.system_usage || '0%' }}</span>
            </div>
          </template>
          <el-progress
            :percentage="parseFloat(systemResourcesParsed?.memory?.system_usage || 0)"
            :color="getProgressColor(systemResourcesParsed?.memory?.system_usage)"
            :stroke-width="20"
          />
          <el-divider />
          <el-descriptions :column="3" border size="small">
            <el-descriptions-item label="总量">{{ systemResourcesParsed?.memory?.system_total || '-' }}</el-descriptions-item>
            <el-descriptions-item label="已用">{{ systemResourcesParsed?.memory?.system_used || '-' }}</el-descriptions-item>
            <el-descriptions-item label="可用">{{ systemResourcesParsed?.memory?.system_free || '-' }}</el-descriptions-item>
            <el-descriptions-item label="已分配">{{ systemResourcesParsed?.memory?.alloc || '-' }}</el-descriptions-item>
            <el-descriptions-item label="总分配">{{ systemResourcesParsed?.memory?.total_alloc || '-' }}</el-descriptions-item>
            <el-descriptions-item label="GC次数">{{ systemResourcesParsed?.memory?.gc_count || '-' }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mb-4">
      <!-- 网络状态 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <el-icon><Link /></el-icon>
              <span>网络</span>
            </div>
          </template>
          <el-row :gutter="12">
            <el-col :span="12" v-for="(item, key) in networkStats" :key="key">
              <div class="mini-stat-item">
                <div class="mini-stat-label">{{ item.label }}</div>
                <div class="mini-stat-value">{{ item.value || '-' }}</div>
              </div>
            </el-col>
          </el-row>
        </el-card>
      </el-col>

      <!-- 磁盘状态 -->
      <el-col :xs="24" :lg="12">
        <el-card shadow="hover" class="dashboard-card resource-card">
          <template #header>
            <div class="card-header">
              <el-icon><FolderOpened /></el-icon>
              <span>磁盘</span>
              <span class="ml-auto text-muted">{{ systemResourcesParsed?.disk?.usage || '0%' }}</span>
            </div>
          </template>
          <el-progress
            :percentage="parseFloat(systemResourcesParsed?.disk?.usage || 0)"
            :color="getProgressColor(systemResourcesParsed?.disk?.usage)"
            :stroke-width="20"
          />
          <el-divider />
          <el-descriptions :column="3" border size="small">
            <el-descriptions-item label="总量">{{ systemResourcesParsed?.disk?.total || '-' }}</el-descriptions-item>
            <el-descriptions-item label="已用">{{ systemResourcesParsed?.disk?.used || '-' }}</el-descriptions-item>
            <el-descriptions-item label="可用">{{ systemResourcesParsed?.disk?.free || '-' }}</el-descriptions-item>
            <el-descriptions-item label="文件系统">{{ systemResourcesParsed?.disk?.fs_type || '-' }}</el-descriptions-item>
            <el-descriptions-item label="IO延迟">{{ systemResourcesParsed?.disk?.io_latency || '-' }}</el-descriptions-item>
            <el-descriptions-item label="读写总量">
              {{ systemResourcesParsed?.disk?.read || '-' }} / {{ systemResourcesParsed?.disk?.write || '-' }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统信息 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <el-icon><Platform /></el-icon>
          <span>系统信息</span>
        </div>
      </template>
      <el-descriptions :column="4" border size="small">
        <el-descriptions-item label="操作系统">{{ systemResourcesParsed?.system?.os || '-' }}</el-descriptions-item>
        <el-descriptions-item label="系统版本">{{ systemResourcesParsed?.system?.os_version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="内核版本">{{ systemResourcesParsed?.system?.kernel || '-' }}</el-descriptions-item>
        <el-descriptions-item label="启动时间">{{ systemResourcesParsed?.system?.boot_time || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import socket from '{src}/utils/socket'

// 连接状态
const connected = ref(false)
const onlineCount = ref(0)
const refreshRate = ref('0')
const systemStatus = ref('healthy')
// 重连防抖锁
const isReconnecting = ref(false)
// 轮询定时器兜底
let pollTimer = null

// 解析后的数据
const systemInfoParsed = ref({})
const databaseStatusParsed = ref({})
const cacheStatusParsed = ref({})
const systemResourcesParsed = ref({})

let lastUpdateTime = 0
// 存储所有socket事件回调，用于组件销毁解绑
const socketHandlers = {}

// 计算属性
const systemHealth = computed(() => {
  const usage = parseFloat(systemResourcesParsed.value?.cpu?.usage || 0)
  const memory = parseFloat(systemResourcesParsed.value?.memory?.system_usage || 0)
  const disk = parseFloat(systemResourcesParsed.value?.disk?.usage || 0)

  if (usage >= 90 || memory >= 90 || disk >= 95) return 'status-danger'
  if (usage >= 70 || memory >= 70 || disk >= 80) return 'status-warning'
  return 'status-success'
})

const systemHealthText = computed(() => {
  const map = {
    'status-success': '正常运行',
    'status-warning': '负载较高',
    'status-danger': '资源紧张'
  }
  return map[systemHealth.value] || '未知'
})

const databaseCounts = computed(() => {
  const counts = databaseStatusParsed.value?.counts || {}
  return [
    { label: '用户', value: counts.users?.total || counts.users || 0 },
    { label: '文章', value: counts.articles?.total || counts.articles || 0 },
    { label: '评论', value: counts.comments || 0 },
    { label: '页面', value: counts.pages || 0 },
    { label: '标签', value: counts.tags || 0 },
    { label: '友链', value: counts.links || 0 }
  ]
})

const userStats = computed(() => {
  const users = databaseStatusParsed.value?.counts?.users || {}
  return {
    total: users.total || 0,
    active: users.active || 0,
    normal: users.normal || users.status?.['0'] || 0,
    frozen: users.frozen || users.status?.['1'] || 0
  }
})

const articleStats = computed(() => {
  const articles = databaseStatusParsed.value?.counts?.articles || {}
  return {
    total: articles.total || 0,
    draft: articles.draft || articles.status?.['0'] || 0,
    published: articles.published || articles.status?.['1'] || 0
  }
})

const momentStats = computed(() => {
  const moments = databaseStatusParsed.value?.counts?.moments || {}
  return {
    total: moments.total || 0,
    draft: moments.draft || moments.status?.['0'] || 0,
    published: moments.published || moments.status?.['1'] || 0
  }
})

const attachmentCount = computed(() => {
  return databaseStatusParsed.value?.counts?.attachments || 0
})

const networkStats = computed(() => {
  const net = systemResourcesParsed.value?.network || {}
  return [
    { label: '上行速率', value: net.up || '-' },
    { label: '下行速率', value: net.down || '-' },
    { label: '总发送量', value: net.total_sent || '-' },
    { label: '总接收量', value: net.total_received || '-' },
    { label: '发送包数', value: net.packets_sent || '-' },
    { label: '接收包数', value: net.packets_recv || '-' }
  ]
})

// 获取进度条颜色
const getProgressColor = (value) => {
  const num = parseFloat(value || 0)
  if (num < 50) return '#67c23a'
  if (num < 80) return '#e6a23c'
  return '#f56c6c'
}

// 处理状态消息
const handleStatus = (content) => {
  console.log('处理状态消息:', content)

  // 在线人数更新
  if (content.online_count !== undefined) {
    onlineCount.value = content.online_count
    return
  }
  if (content.online_users !== undefined) {
    onlineCount.value = Array.isArray(content.online_users) ? content.online_users.length : 0
    return
  }

  // 系统监控数据
  if (content.info || content.database || content.cache || content.resource) {
    updateSystemStatus(content)
    return
  }
  if (content.content && (content.content.info || content.content.database)) {
    if (content.content.online_count !== undefined) {
      onlineCount.value = content.content.online_count
    }
    updateSystemStatus(content.content)
  }
}

// 更新系统状态
const updateSystemStatus = (content) => {
  try {
    systemInfoParsed.value = content.info || {}
    databaseStatusParsed.value = content.database || {}
    cacheStatusParsed.value = content.cache || {}
    systemResourcesParsed.value = content.resource || {}

    const now = Date.now() / 1000
    if (lastUpdateTime > 0) {
      refreshRate.value = (now - lastUpdateTime).toFixed(1)
    }
    lastUpdateTime = now
  } catch (error) {
    console.error('更新状态失败:', error)
  }
}

// 绑定socket事件并保存回调，用于销毁解绑
const bindSocketEvents = () => {
  socketHandlers.open = () => {
    connected.value = true
    isReconnecting.value = false
    console.log('WebSocket 连接已建立')
    // WS恢复后关闭http兜底轮询
    stopPoll()
  }
  socketHandlers.connect = (data) => {
    console.log('连接成功，客户端ID:', data?.id)
  }
  socketHandlers.status = handleStatus
  socketHandlers.broadcast = (data) => {
    ElMessage.info(data?.content?.message || '收到广播消息')
  }
  socketHandlers.single = (data) => {
    ElMessage.info(data?.content?.message || '收到单播消息')
  }
  socketHandlers.private = (data) => {
    ElMessage.success(`收到私聊: ${data?.content?.message || ''}`)
  }
  socketHandlers.close = () => {
    connected.value = false
    console.log('WebSocket 连接已关闭')
    // WS断开，开启http兜底轮询保证面板数据可用
    startPoll()
  }
  socketHandlers.error = (error) => {
    console.error('WebSocket 错误:', error)
    isReconnecting.value = false
  }

  socket.on('open', socketHandlers.open)
  socket.on('connect', socketHandlers.connect)
  socket.on('status', socketHandlers.status)
  socket.on('broadcast', socketHandlers.broadcast)
  socket.on('single', socketHandlers.single)
  socket.on('private', socketHandlers.private)
  socket.on('close', socketHandlers.close)
  socket.on('error', socketHandlers.error)
}

// 解绑所有socket事件，防止内存泄漏、后台自动重连
const unbindSocketEvents = () => {
  Object.entries(socketHandlers).forEach(([event, fn]) => {
    socket.off(event, fn)
  })
}

// http兜底轮询（WS断连备用，避免监控面板空白）
const startPoll = () => {
  if (pollTimer) return
  pollTimer = setInterval(async () => {
    try {
      const res = await fetch('/api/system/status')
      const data = await res.json()
      updateSystemStatus(data)
      onlineCount.value = data.online_count ?? 0
    } catch (e) {
      console.debug('轮询获取监控数据失败', e)
    }
  }, 5000)
}
const stopPoll = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// 初始化 Socket 连接
const initSocket = () => {
  if (isReconnecting.value) return
  isReconnecting.value = true
  bindSocketEvents()
  socket.connect()
  if (socket.isConnected()) {
    connected.value = true
    stopPoll()
  } else {
    startPoll()
  }
}

// 手动重连（带防抖锁，防止重复点击）
const reconnect = () => {
  if (isReconnecting.value) return ElMessage.warning('正在重连中，请稍候')
  // 先清理旧连接与事件
  socket.close()
  unbindSocketEvents()
  setTimeout(() => {
    initSocket()
  }, 300)
}

onMounted(() => {
  initSocket()
})

// 组件销毁：彻底清理连接、事件、轮询、重连定时器
onBeforeUnmount(() => {
  isReconnecting.value = true
  stopPoll()
  unbindSocketEvents()
  socket.close()
})
</script>

<style lang="scss" scoped>
// 全局统一基础变量，柔和后台标准
$radius-sm: 6px;
$radius-md: 8px;
$radius-lg: 10px;

$gap-xs: 4px;
$gap-sm: 8px;
$gap-md: 12px;
$gap-lg: 16px;
$gap-xl: 24px;

// 柔和阴影分层（轻量化，不厚重）
$shadow-light: 0 1px 4px rgba(0, 0, 0, 0.04);
$shadow-hover: 0 3px 10px rgba(0, 0, 0, 0.06);

.mb-4 {
  margin-bottom: $gap-lg;
}

.section-title {
  display: flex;
  align-items: center;
  gap: $gap-sm;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: $gap-md;
  padding-left: $gap-xs;
}

// ========== 顶部状态栏优化（柔和低饱和渐变） ==========
.status-bar {
  margin-bottom: $gap-xl;
}

.status-item {
  display: flex;
  align-items: center;
  gap: $gap-md;
  padding: $gap-lg;
  border-radius: $radius-md;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  margin-bottom: $gap-md;
  transition: all 0.24s ease;

  &:hover {
    box-shadow: $shadow-hover;
    border-color: var(--el-border-color-light);
  }

  // 弱化渐变透明度，护眼不刺眼
  &.status-success {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.08) 0%, rgba(103, 194, 58, 0.03) 100%);
    border-color: rgba(103, 194, 58, 0.25);
    .status-icon { color: #67c23a; }
  }
  &.status-danger {
    background: linear-gradient(135deg, rgba(245, 108, 108, 0.08) 0%, rgba(245, 108, 108, 0.03) 100%);
    border-color: rgba(245, 108, 108, 0.25);
    .status-icon { color: #f56c6c; }
  }
  &.status-warning {
    background: linear-gradient(135deg, rgba(230, 162, 60, 0.08) 0%, rgba(230, 162, 60, 0.03) 100%);
    border-color: rgba(230, 162, 60, 0.25);
    .status-icon { color: #e6a23c; }
  }
  &.status-primary {
    background: linear-gradient(135deg, rgba(64, 158, 255, 0.08) 0%, rgba(64, 158, 255, 0.03) 100%);
    border-color: rgba(64, 158, 255, 0.25);
    .status-icon { color: #409eff; }
  }
  &.status-info {
    background: linear-gradient(135deg, rgba(144, 147, 153, 0.06) 0%, rgba(144, 147, 153, 0.02) 100%);
    border-color: rgba(144, 147, 153, 0.2);
    .status-icon { color: #909399; }
  }
}

.status-icon {
  flex-shrink: 0;
}
.status-content {
  flex: 1;
  min-width: 0;
}
.status-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: $gap-xs;
  opacity: 0.85;
}
.status-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.ml-auto {
  margin-left: auto;
}

// ========== 全局卡片统一美化 ==========
.dashboard-card {
  height: 100%;
  border-radius: $radius-md;
  transition: box-shadow 0.24s ease;

  :deep(.el-card__header) {
    padding: $gap-md $gap-lg;
    background: rgba(var(--el-fill-color-rgb), 0.4); // 淡化头部背景，不突兀
    margin-bottom: $gap-md;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  :deep(.el-card__body) {
    padding: $gap-lg;
  }
  :deep(.el-divider--horizontal) {
    margin: $gap-md 0;
    opacity: 0.7;
  }
}
.dashboard-card:hover {
  box-shadow: $shadow-hover;
}

.card-header {
  display: flex;
  align-items: center;
  gap: $gap-sm;
  font-weight: 600;
  color: var(--el-text-color-primary);
  font-size: 14px;
}

// ========== 数据库统计小块 ==========
.stat-item {
  text-align: center;
  padding: $gap-md $gap-sm;
  background: var(--el-fill-color-light);
  border-radius: $radius-sm;
  margin-bottom: $gap-sm;
  transition: background 0.2s ease;

  &:hover {
    background: var(--el-fill-color);
  }
}
.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: var(--el-color-primary);
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: $gap-xs;
  opacity: 0.85;
}

// ========== 数据统计网格 ==========
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  gap: $gap-md;
}

.stat-box {
  text-align: center;
  padding: $gap-lg $gap-sm;
  background: var(--el-fill-color-light);
  border-radius: $radius-md;
  transition: all 0.2s ease;

  &:hover {
    background: var(--el-fill-color);
    transform: translateY(-2px);
  }

  &.stat-success {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.1) 0%, rgba(103, 194, 58, 0.05) 100%);
    .stat-num { color: #67c23a; }
  }

  &.stat-danger {
    background: linear-gradient(135deg, rgba(245, 108, 108, 0.1) 0%, rgba(245, 108, 108, 0.05) 100%);
    .stat-num { color: #f56c6c; }
  }

  &.stat-warning {
    background: linear-gradient(135deg, rgba(230, 162, 60, 0.1) 0%, rgba(230, 162, 60, 0.05) 100%);
    .stat-num { color: #e6a23c; }
  }

  &.stat-primary {
    background: linear-gradient(135deg, rgba(64, 158, 255, 0.1) 0%, rgba(64, 158, 255, 0.05) 100%);
    .stat-num { color: #409eff; }
  }
}

.stat-num {
  font-size: 24px;
  font-weight: 700;
  color: var(--el-color-primary);
  line-height: 1.2;
}

.stat-name {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: $gap-xs;
  opacity: 0.85;
}

// ========== 缓存状态模块 ==========
.cache-status-item {
  display: flex;
  align-items: center;
  gap: $gap-lg;
  padding: $gap-lg;
  background: var(--el-fill-color-light);
  border-radius: $radius-md;
  margin-bottom: $gap-md;
  transition: background 0.2s ease;

  &:hover {
    background: var(--el-fill-color);
  }
}
.cache-status-content {
  flex: 1;
}
.cache-status-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: $gap-xs;
  opacity: 0.85;
}
.cache-status-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

// ========== 网络迷你统计块 ==========
.mini-stat-item {
  padding: $gap-md;
  background: var(--el-fill-color-light);
  border-radius: $radius-sm;
  margin-bottom: $gap-sm;
  transition: background 0.2s ease;

  &:hover {
    background: var(--el-fill-color);
  }
}
.mini-stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
  opacity: 0.85;
}
.mini-stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

// ========== 工具文字颜色类 ==========
.text-success { color: #67c23a; }
.text-danger { color: #f56c6c; }
.text-warning { color: #e6a23c; }
.text-primary { color: #409eff; }
.text-muted { color: var(--el-text-color-secondary); opacity: 0.8; }

.text-truncate {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

// ========== 进度条美化，圆角柔和 ==========
:deep(.el-progress-bar__outer) {
  border-radius: $radius-lg;
  height: 20px !important;
  background: var(--el-fill-color-light);
}
:deep(.el-progress-bar__inner) {
  border-radius: $radius-lg;
  transition: width 0.3s ease;
}
</style>
