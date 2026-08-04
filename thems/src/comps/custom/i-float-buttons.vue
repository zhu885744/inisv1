<!-- 悬浮按钮组件 -->
<template>
  <div 
    v-if="config.enabled" 
    class="float-buttons-container"
    :class="{ 'position-center': config.position === 'center', 'position-bottom': config.position === 'bottom' }"
  >
    <!-- 公告按钮 -->
    <div 
      v-if="config.show_notice"
      class="float-button"
      :class="{ 'rounded-button': config.style === 'rounded', 'square-button': config.style === 'square' }"
      title="系统公告"
      @click="openNoticeList"
    >
      <i class="bi bi-megaphone-fill"></i>
      <span 
        v-if="noticeList.length > 0" 
        class="notice-badge position-absolute top-0 start-0 translate-middle badge"
        style="padding: 0; min-width: 1.4rem; height: 1.4rem; display: flex; align-items: center; justify-content: center;"
      >
        {{ noticeList.length }}
      </span>
    </div>

    <!-- 自定义按钮 -->
    <div 
      v-for="button in enabledButtons" 
      :key="button.id"
      class="float-button"
      :class="{ 'rounded-button': config.style === 'rounded', 'square-button': config.style === 'square' }"
      :title="button.tooltip"
      @click="handleButtonClick(button)"
    >
      <i :class="button.icon"></i>
      <div 
        v-if="button.image_url" 
        class="float-button-content"
      >
        <img 
          :src="button.image_url" 
          :alt="button.name || '图片'"
          :title="button.name || '图片'"
          style="width: 100%; height: auto; border-radius: 4px;"
        >
      </div>
    </div>

    <!-- 返回顶部按钮 -->
    <div 
      v-if="config.show_back_to_top && showBackToTop"
      class="float-button"
      :class="{ 'rounded-button': config.style === 'rounded', 'square-button': config.style === 'square' }"
      title="返回顶部"
      @click="scrollToTop"
    >
      <i class="bi bi-arrow-up"></i>
    </div>
  </div>

  <!-- 公告列表弹窗 -->
  <div 
    class="modal fade notice-modal" 
    id="noticeListModal" 
    tabindex="-1" 
    aria-hidden="true" 
    ref="noticeListModal"
  >
    <div class="modal-dialog modal-dialog-centered modal-md modal-dialog-scrollable">
      <div class="modal-content border-0 shadow-lg animate-fade">
        <div class="modal-header">
          <h5 class="modal-title d-flex align-items-center">
            <i class="bi bi-megaphone-fill me-2"></i>公告
          </h5>
          <button 
            type="button" 
            class="btn-close btn-close-white" 
            data-bs-dismiss="modal" 
            aria-label="关闭"
          ></button>
        </div>
        <div class="modal-body p-0">
          <!-- 公告列表 -->
          <div v-if="noticeList.length > 0" class="list-group list-group-flush">
            <button
              v-for="notice in noticeList"
              :key="notice.id"
              class="list-group-item list-group-item-action py-3 px-4 border-start-0 border-end-0"
              @click="openNoticeDetail(notice)"
            >
              <div class="d-flex justify-content-between align-items-center">
                <div class="flex-grow-1 me-3 text-start">
                  <h6 class="mb-1 fw-semibold">{{ notice.title }}</h6>
                  <small class="text-muted">{{ formatDate(notice.create_time) }}</small>
                </div>
                <i class="bi bi-chevron-right text-muted opacity-75"></i>
              </div>
            </button>
          </div>
          
          <!-- 空状态 -->
          <div v-else class="wx-empty-state">
            <i class="bi bi-inbox"></i>
            <p>暂无系统公告</p>
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="wx-btn-outline" data-bs-dismiss="modal">关闭</button>
        </div>
      </div>
    </div>
  </div>

  <!-- 公告详情弹窗 -->
  <div 
    class="modal fade notice-modal" 
    id="noticeDetailModal" 
    tabindex="-1" 
    aria-hidden="true" 
    ref="noticeDetailModal"
  >
    <div class="modal-dialog modal-dialog-centered modal-md modal-dialog-scrollable">
      <div class="modal-content border-0 shadow-lg animate-fade">
        <div class="modal-header">
          <h5 class="modal-title">{{ currentNotice?.title || '公告详情' }}</h5>
          <button 
            type="button" 
            class="btn-close btn-close-white" 
            data-bs-dismiss="modal" 
            aria-label="关闭"
          ></button>
        </div>
        <div class="modal-body">
          <!-- 公告信息 -->
          <div class="d-flex flex-wrap gap-2 mb-3">
            <span class="badge text-bg-secondary">
              <i class="bi bi-tag me-1"></i>{{ currentNotice?.type || '系统公告' }}
            </span>
            <span class="badge text-bg-secondary">
              <i class="bi bi-calendar3 me-1"></i>{{ formatDate(currentNotice?.create_time) }}
            </span>
          </div>
          
          <!-- 公告内容 -->
          <div class="border rounded-3 p-4" style="background: var(--bs-tertiary-bg);">
            <div v-html="formatContent(currentNotice?.content)" class="text-break"></div>
          </div>

          <!-- 查看详情链接 -->
          <div v-if="currentNotice?.url" class="mt-3 text-center">
            <a
              :href="currentNotice.url"
              :target="currentNotice.target || '_blank'"
              class="wx-btn-gradient"
            >
              <i class="bi bi-link-45deg me-1"></i>查看详情
            </a>
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="wx-btn-outline" @click="backToList">
            <i class="bi bi-arrow-left me-1"></i>返回列表
          </button>
          <button type="button" class="wx-btn-outline" data-bs-dismiss="modal">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useCommStore } from '@/store/comm'
import { Modal } from 'bootstrap'
import { request } from '@/utils/network'
import { toast, formatters } from '@/utils/app'
import utils from '@/utils/utils'

const store = useCommStore()
const config = ref({
  enabled: true,
  style: 'rounded',
  position: 'center',
  show_back_to_top: true,
  show_notice: true,
  buttons: []
})

// 响应式变量
const showBackToTop = ref(false)
let scrollTicking = false

// 公告相关
const noticeList = ref([])
const currentNotice = ref(null)
const noticeListModal = ref(null)
const noticeDetailModal = ref(null)
let listModalInstance = null
let detailModalInstance = null

// 计算启用的按钮
const enabledButtons = computed(() => {
  return config.value.buttons.filter(button => button.enabled)
})

// 处理按钮点击
const handleButtonClick = (button) => {
  if (button.url && !button.image_url) {
    window.open(button.url, '_blank')
  }
}

// 返回顶部
const scrollToTop = () => {
  window.scrollTo({
    top: 0,
    behavior: 'smooth'
  })
}

// 获取公告列表
const fetchNotices = async () => {
  try {
    const res = await request.get('/api/placard/all', {
      page: 1,
      limit: 10,
      order: 'create_time desc'
    })
    
    if (res.code === 200 && res.data?.data) {
      noticeList.value = res.data.data
    }
  } catch (error) {
    console.error('获取公告失败:', error)
  }
}

// 打开公告列表
const openNoticeList = () => {
  if (!listModalInstance) {
    if (noticeListModal.value) {
      listModalInstance = new Modal(noticeListModal.value, {
        backdrop: true,
        keyboard: true
      })
    }
  }
  if (listModalInstance) {
    listModalInstance.show()
  }
}

// 打开公告详情
const openNoticeDetail = (notice) => {
  currentNotice.value = notice

  noticeListModal.value.addEventListener('hidden.bs.modal', function onHide() {
    detailModalInstance.show()
    noticeListModal.value.removeEventListener('hidden.bs.modal', onHide)
  })
  
  listModalInstance.hide()
}

// 返回公告列表
const backToList = () => {
  noticeDetailModal.value.addEventListener('hidden.bs.modal', function onHide() {
    listModalInstance.show()
    noticeDetailModal.value.removeEventListener('hidden.bs.modal', onHide)
  })
  
  detailModalInstance.hide()
}

// 格式化日期
const formatDate = (timestamp) => {
  if (!timestamp) return '未知时间'
  
  try {
    return utils.timeToDate(timestamp, 'Y-m-d')
  } catch {
    return '未知时间'
  }
}

// 格式化内容
const formatContent = (content) => {
  if (!content) return '<p class="text-muted mb-0">暂无公告内容</p>'
  return content.replace(/\n/g, '<br>')
}

// 监听配置变化
watch(
  () => store.siteInfo?.float_buttons,
  (newConfig) => {
    if (newConfig) {
      config.value = newConfig
    }
  }
)

// 初始化弹窗
const initModals = () => {
  if (noticeListModal.value) {
    listModalInstance = new Modal(noticeListModal.value, {
      backdrop: true,
      keyboard: true
    })
  }
  if (noticeDetailModal.value) {
    detailModalInstance = new Modal(noticeDetailModal.value, {
      backdrop: true,
      keyboard: true
    })
  }
}

// 监听滚动事件（节流）
const handleScroll = () => {
  if (!scrollTicking) {
    requestAnimationFrame(() => {
      showBackToTop.value = window.scrollY > 300
      scrollTicking = false
    })
    scrollTicking = true
  }
}

onMounted(() => {
  // 初始化弹窗
  initModals()
  // 从store获取配置
  if (store.siteInfo?.float_buttons) {
    config.value = store.siteInfo.float_buttons
  }
  // 获取公告列表
  fetchNotices()
  // 添加滚动事件监听
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  // 移除滚动事件监听
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style scoped>
/* 悬浮按钮容器 */
.float-buttons-container {
  position: fixed;
  right: 10px;
  z-index: 999;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* 位置样式 */
.position-center {
  top: 50%;
  transform: translateY(-50%);
}

.position-bottom {
  bottom: 80px;
}

/* 按钮样式 */
.float-button {
  width: 50px;
  height: 50px;
  background: linear-gradient(135deg, var(--bs-secondary), var(--bs-dark));
  border: none;
  color: #fff;
  font-size: 1.2rem;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--wx-transition);
  position: relative;
  box-shadow: var(--wx-shadow-md);
}

.rounded-button {
  border-radius: 50%;
}

.square-button {
  border-radius: var(--wx-radius-sm);
}

/* 按钮悬停效果 */
.float-button:hover {
  transform: translateY(-3px);
  filter: brightness(1.1);
  box-shadow: var(--wx-shadow-lg);
}

/* 悬浮内容 */
.float-button-content {
  position: absolute;
  right: 100%;
  top: 50%;
  transform: translateY(-50%);
  margin-right: 10px;
  padding: 8px;
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  box-shadow: var(--wx-shadow-lg);
  white-space: normal;
  word-break: break-word;
  z-index: 1000;
  opacity: 0;
  visibility: hidden;
  transition: var(--wx-transition);
  pointer-events: none;
  width: 200px;
  max-height: calc(100vh - 100px);
  overflow-y: auto;
  overflow-x: hidden;
}

/* 图片样式 */
.float-button-content img {
  width: 100%;
  height: auto;
  border-radius: var(--wx-radius-sm);
  display: block;
}

.float-button:hover .float-button-content {
  opacity: 1;
  visibility: visible;
  transform: translateY(-50%);
}

/* 公告徽章样式 */
.notice-badge {
  position: absolute;
  top: 0;
  left: 0;
  transform: translate(-50%, -50%);
  font-size: 0.75rem;
  padding: 0.5rem 0.5rem;
  min-width: 1.4rem;
  height: 1.4rem;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bs-warning);
  color: var(--bs-warning-text, #212529);
  border: 2px solid var(--bs-body-bg);
  border-radius: 50%;
  z-index: 1000;
}

/* 核心丝滑弹窗动画 */
.modal.fade .modal-dialog {
  transition: transform 0.32s cubic-bezier(0.16, 1, 0.3, 1),
              opacity 0.28s ease !important;
  transform: translate3d(0, 20px, 0) scale(0.96);
  opacity: 0;
}

.modal.fade.show .modal-dialog {
  transform: translate3d(0, 0, 0) scale(1);
  opacity: 1;
}

.animate-fade {
  animation: contentFade 0.3s ease forwards;
}

@keyframes contentFade {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* 列表项交互 */
.list-group-item {
  transition: var(--wx-transition);
  border-color: var(--bs-border-color);
}

/* 空状态图标 */
.wx-empty-state i {
  color: var(--bs-secondary-color);
}

/* 响应式调整 */
@media (max-width: 768px) {
  .float-buttons-container {
    right: 10px;
  }

  .float-button {
    width: 40px;
    height: 40px;
    font-size: 1rem;
  }

  .float-button-content {
    max-width: 240px !important;
  }

  .position-bottom {
    bottom: 20px;
  }
}
</style>