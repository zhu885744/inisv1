import { defineStore } from 'pinia'
import { socketManager } from '@/composables/useSocket'

// 防重复注册监听的标志
let statusListenerRegistered = false
let openListenerRegistered = false
let closeListenerRegistered = false

export const useSocketStore = defineStore('socket', {
  state: () => ({
    // 连接状态
    connectionState: 'disconnected',
    clientId: '',
    lastError: null,

    // 在线状态
    onlineUsers: [],
    onlineCount: 0,

    // 系统状态（每秒推送）
    systemStatus: null,

    // 消息列表
    messages: [],
    maxMessages: 100,

    // 私聊消息（按用户/会话分组）
    privateMessages: new Map(),

    // 广播消息
    broadcastMessages: [],

    // 连接时间
    connectedAt: null,
  }),

  getters: {
    isConnected: (state) => state.connectionState === 'connected',
    isConnecting: (state) => state.connectionState === 'connecting',
    isError: (state) => state.connectionState === 'error',

    // 应用信息
    appInfo: (state) => state.systemStatus?.info || {},
    databaseStatus: (state) => state.systemStatus?.database || {},
    cacheStatus: (state) => state.systemStatus?.cache || {},
    resourceUsage: (state) => state.systemStatus?.resource || {},
    systemHealth: (state) => state.systemStatus?.status || 'unknown',

    // 数据库统计
    userStats: (state) => state.systemStatus?.database?.counts?.users || {},
    articleStats: (state) => state.systemStatus?.database?.counts?.articles || {},
    momentStats: (state) => state.systemStatus?.database?.counts?.moments || {},

    // 资源
    memoryInfo: (state) => state.systemStatus?.resource?.memory || {},
    cpuInfo: (state) => state.systemStatus?.resource?.cpu || {},
    diskInfo: (state) => state.systemStatus?.resource?.disk || {},
    networkInfo: (state) => state.systemStatus?.resource?.network || {},
    systemInfo: (state) => state.systemStatus?.resource?.system || {},

    // 连接时长（秒）
    connectedDuration: (state) => {
      if (!state.connectedAt) return 0
      return Math.floor((Date.now() - state.connectedAt) / 1000)
    },
  },

  actions: {
    /**
     * 初始化 Socket 监听并建立连接
     */
    init() {
      // 只注册一次监听器，避免重复
      if (!statusListenerRegistered) {
        statusListenerRegistered = true
        socketManager.on('status', (content) => {
          this.handleStatus(content)
        })
      }

      if (!openListenerRegistered) {
        openListenerRegistered = true
        socketManager.on('open', () => {
          this.connectionState = 'connected'
          this.connectedAt = Date.now()
          this.lastError = null
        })
      }

      if (!closeListenerRegistered) {
        closeListenerRegistered = true
        socketManager.on('close', ({ code, reason }) => {
          this.connectionState = 'disconnected'
          this.connectedAt = null
          // 正常关闭不清理数据，仅状态变更
        })
        socketManager.on('error', (error) => {
          this.connectionState = 'error'
          this.lastError = error
        })
        socketManager.on('connect', (data) => {
          if (data?.id) {
            this.clientId = data.id
          }
        })
      }

      // 同步现有状态
      this.connectionState = socketManager.connectionState.value
      this.clientId = socketManager.clientId.value

      // 建立连接
      socketManager.connect()
    },

    /**
     * 处理 status 类型消息
     * 区分在线状态和系统状态两种格式
     */
    handleStatus(content) {
      if (!content) return

      // 在线状态消息格式
      if (content.online_count !== undefined) {
        this.onlineCount = content.online_count || 0
        this.onlineUsers = content.online_users || []
        return
      }

      // 系统状态消息格式（含 info/database/cache/resource 等字段）
      if (content.info || content.resource || content.database) {
        this.systemStatus = { ...this.systemStatus, ...content }
        return
      }

      // 兼容：直接是系统状态（顶层字段）
      if (content.status || content.timestamp) {
        this.systemStatus = { ...this.systemStatus, ...content }
      }
    },

    /**
     * 添加消息到列表
     */
    addMessage(message) {
      this.messages.unshift(message)
      if (this.messages.length > this.maxMessages) {
        this.messages.pop()
      }
    },

    /**
     * 添加广播消息
     */
    addBroadcastMessage(message) {
      this.broadcastMessages.unshift(message)
      if (this.broadcastMessages.length > this.maxMessages) {
        this.broadcastMessages.pop()
      }
      this.addMessage(message)
    },

    /**
     * 添加私聊消息
     */
    addPrivateMessage(from, message) {
      const key = from
      if (!this.privateMessages.has(key)) {
        this.privateMessages.set(key, [])
      }
      const list = this.privateMessages.get(key)
      list.unshift(message)
      if (list.length > this.maxMessages) {
        list.pop()
      }
      this.addMessage(message)
    },

    /**
     * 手动重连
     */
    reconnect() {
      socketManager.disconnect()
      setTimeout(() => {
        socketManager.connect()
      }, 500)
    },

    /**
     * 断开连接
     */
    disconnect() {
      socketManager.disconnect()
    },

    /**
     * 清除所有消息
     */
    clearMessages() {
      this.messages = []
      this.broadcastMessages = []
      this.privateMessages.clear()
    },
  },
})

export default useSocketStore
