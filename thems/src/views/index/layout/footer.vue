<template>
  <footer id="footer" class="fs-6 py-3 mt-2 bg-body-tertiary text-body-secondary">
    <div class="container">
      <div class="text-center">
        <!-- 小黑屋 -->
        <div class="text">
          <router-link class="nav-link" :to="`/blackroom`" active-class="active" exact-active-class="active">
              小黑屋
          </router-link>
        </div>
        <!-- 在线用户 -->
        <div class="text online-users">
          <a 
            href="javascript:void(0)" 
            class="text-decoration-none text-reset hover-text-primary transition-opacity d-inline-flex align-items-center gap-1"
            @click="openOnlineModal"
            aria-label="查看在线用户"
          >
            <i class="bi bi-people-fill"></i>
            <span>{{ onlineCount }}</span> 人在线
          </a>
        </div>
        <!-- 版权年份 -->
        <div class="text">
          Copyright © {{ startYear || '2020' }} ~ {{ currentYear }} {{ siteTitle }} 版权所有
        </div>
        
        <!-- ICP备案号 -->
        <div class="test" v-if="hasIcp">
          <a 
            :href="icpLink" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            aria-label="ICP备案号"
          >
            {{ icpCode }}
          </a>
        </div>
        
        <!-- 公安备案号 -->
        <div class="test" v-if="hasPolice">
          <a 
            :href="policeLink" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            aria-label="公安备案号"
          >
            {{ policeCode }}
          </a>
        </div>
        
        <!-- 技术支持 -->
        <div class="text" aria-label="技术支持">
          <span>Powered by </span>
          <a 
            href="https://github.com/zhu885744/inisv1" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            title="inisv1 开源地址"
          >
            inis
          </a>
          <span class="mx-1">|</span>
          <span>Theme by </span>
          <a 
            href="https://github.com/zhu885744/Cardify" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            title="Cardify"
          >
            Cardify v{{ themeVersion }}
          </a>
        </div>
      </div>
    </div>

    <!-- 在线用户弹窗 -->
    <div class="modal fade" ref="onlineModalRef" tabindex="-1" aria-labelledby="onlineModalLabel" aria-hidden="true">
      <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title" id="onlineModalLabel">
              <i class="bi bi-people-fill me-1"></i>在线访客（{{ onlineCount }}）
            </h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="关闭"></button>
          </div>
          <div class="modal-body">
            <ul v-if="onlineUsers.length" class="list-group list-group-flush">
              <li 
                v-for="(u, idx) in onlineUsers" 
                :key="u.id || idx" 
                class="list-group-item d-flex justify-content-between align-items-center px-0"
              >
                <span class="fw-semibold text-body d-flex align-items-center gap-2">
                  <i class="bi bi-person-circle text-secondary"></i>{{ u.name }}
                </span>
                <span class="text-muted small">
                  {{ u.ip }} · {{ activeText(u.last_active) }}
                </span>
              </li>
            </ul>
            <div v-else class="text-center text-muted py-4">
              <i class="bi bi-emoji-neutral d-block mb-2 fs-3"></i>暂无在线用户
            </div>
          </div>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { Modal } from 'bootstrap'
import { useCommStore } from '@/store/comm'
import { useSocketStore } from '@/store/socket'

// 环境变量
const THEME_VERSION = import.meta.env.VITE_VERSION || '1.0.0'

// 响应式数据
const currentYear = new Date().getFullYear()
const commStore = useCommStore()

// 直接使用store中的siteInfo
const siteInfo = ref(commStore.siteInfo)

const startYear = computed(() => {
  const timestamp = siteInfo.value?.date
  if (!timestamp) return currentYear
  
  try {
    const milliseconds = parseInt(timestamp) * 1000
    const date = new Date(milliseconds)
    const year = date.getFullYear()
    return isNaN(year) ? currentYear : year
  } catch (error) {
    return currentYear
  }
})

const siteTitle = computed(() => {
  return siteInfo.value?.title || '未设置网站名'
})

// ICP备案
const hasIcp = computed(() => !!siteInfo.value?.copy?.code)
const icpCode = computed(() => siteInfo.value?.copy?.code || '请在后台填写备案号')
const icpLink = computed(() => siteInfo.value?.copy?.link || 'https://beian.miit.gov.cn/#/Integrated/index')

// 公安备案
const hasPolice = computed(() => !!siteInfo.value?.police?.code)
const policeCode = computed(() => siteInfo.value?.police?.code || '请在后台填写公安备案号')
const policeLink = computed(() => siteInfo.value?.police?.link || 'https://beian.mps.gov.cn/#/query/webSearch')

const themeVersion = computed(() => THEME_VERSION)

// ===== 在线用户 =====
const socketStore = useSocketStore()
const onlineCount = computed(() => socketStore.onlineCount || 0)
const onlineUsers = computed(() => socketStore.onlineUsers || [])

const onlineModalRef = ref(null)
let onlineModal = null

const openOnlineModal = () => {
  if (!onlineModalRef.value) return
  if (!onlineModal) {
    onlineModal = new Modal(onlineModalRef.value, { backdrop: true, keyboard: true })
  }
  onlineModal.show()
}

// 活跃时间文案
const activeText = (lastActive) => {
  if (!lastActive) return '刚刚活跃'
  const diff = Math.floor(Date.now() / 1000) - Number(lastActive)
  if (diff < 60) return '刚刚活跃'
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前活跃`
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前活跃`
  return `${Math.floor(diff / 86400)} 天前活跃`
}

// 组件挂载
onMounted(async () => {
  if (!siteInfo.value || Object.keys(siteInfo.value).length === 0) {
    await commStore.fetchSiteInfo()
    siteInfo.value = commStore.siteInfo
  }
})
</script>

<style scoped>
.hover-text-primary:hover {
  color: var(--bs-primary) !important;
}

.transition-opacity {
  transition: opacity 0.2s ease;
}

.transition-opacity:hover {
  opacity: 0.8;
}
</style>