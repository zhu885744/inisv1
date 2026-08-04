<template>
  <div class="attachments-page-wrapper mt-2">
    <!-- 未登录提示 -->
    <div v-if="!isLoggedIn" class="card shadow-sm">
      <div class="card-body text-center py-5">
        <i class="bi bi-lock text-muted" style="font-size: 3rem;"></i>
        <p class="mt-3 text-muted mb-3">请先登录后查看附件</p>
        <div class="d-flex gap-2 justify-content-center">
          <button class="btn btn-primary btn-sm px-4" @click="handleToLogin">登录</button>
          <button class="btn btn-outline-primary btn-sm px-4" @click="handleToRegister">注册</button>
        </div>
      </div>
    </div>

    <template v-else>
    <!-- 页面标题 -->
    <div class="card shadow-sm mb-2">
      <div class="card-body">
        <h5 class="card-title mb-1">
          <i class="bi bi-folder2-open me-2"></i>
          附件管理
          <span v-if="isAdmin" class="badge bg-danger ms-2">管理员</span>
        </h5>
        <p class="text-muted card-text mb-0">共有 {{ totalCount }} 个附件{{ isAdmin ? '（全部）' : '（我的）' }}</p>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="card shadow-sm mb-3">
      <div class="card-body py-3">
        <div class="row g-2 align-items-center">
          <!-- 搜索框 -->
          <div class="col-12 col-md-4">
            <div class="input-group input-group-sm">
              <span class="input-group-text bg-transparent">
                <i class="bi bi-search"></i>
              </span>
              <input
                v-model="searchKeyword"
                type="text"
                class="form-control"
                placeholder="搜索文件名"
                @keyup.enter="handleSearch"
              >
              <button
                v-if="searchKeyword"
                class="btn btn-outline-secondary"
                type="button"
                @click="clearSearch"
              >
                <i class="bi bi-x-lg"></i>
              </button>
            </div>
          </div>

          <!-- 排序 -->
          <div class="col-8">
            <select
              v-model="filters.order"
              class="form-select form-select-sm"
              @change="handleFilterChange"
            >
              <option value="create_time desc">最新上传</option>
              <option value="create_time asc">最早上传</option>
              <option value="file_size desc">文件最大</option>
              <option value="file_size asc">文件最小</option>
              <option value="original_name asc">名称A-Z</option>
            </select>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="row g-3">
      <div v-for="i in 12" :key="i" class="col-6 col-md-4 col-lg-3">
        <div class="card h-100">
          <div class="card-body">
            <div class="skeleton skeleton-thumb mb-2"></div>
            <div class="skeleton skeleton-line w-100 mb-1"></div>
            <div class="skeleton skeleton-line w-75"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="card">
      <div class="card-body text-center py-5">
        <i class="bi bi-exclamation-circle text-danger fs-1"></i>
        <p class="mt-3 text-muted">{{ error }}</p>
        <button @click="fetchAttachments" class="btn btn-sm btn-outline-secondary">
          重试
        </button>
      </div>
    </div>

    <!-- 附件列表 -->
    <div v-else-if="attachments.length > 0" class="row g-3">
      <div
        v-for="file in attachments"
        :key="file.id || file.uuid"
        class="col-6 col-md-4 col-lg-3"
      >
        <div class="card attachment-card h-100 shadow-sm hover-shadow transition-all">
          <!-- 预览区 -->
          <div class="attachment-preview" @click="previewAttachment(file)">
            <!-- 图片预览 -->
            <img
              v-if="isImage(file)"
              :src="file.full_url"
              :alt="file.original_name"
              class="preview-img"
              loading="lazy"
              decoding="async"
              @error="handleImageError"
            >
            <!-- 其他文件图标 -->
            <div v-else class="preview-icon">
              <i :class="getFileIcon(file)"></i>
            </div>

            <!-- 文件格式标签 -->
            <span class="badge text-bg-light attachment-ext-badge">
              {{ (file.file_ext || 'FILE').toUpperCase() }}
            </span>
          </div>

          <!-- 信息区 -->
          <div class="card-body p-2">
            <p class="attachment-name mb-1" :title="file.original_name">
              {{ file.original_name || '未命名文件' }}
            </p>
            <div class="d-flex align-items-center justify-content-between text-muted small">
              <span>{{ formatFileSize(file.file_size) }}</span>
              <span class="text-truncate ms-1" style="max-width: 80px;">
                {{ formatRelativeTime(file.create_time) }}
              </span>
            </div>
            <!-- 存储驱动标签 -->
            <div v-if="file.storage_driver" class="mt-1">
              <span class="badge text-bg-secondary storage-badge">
                {{ getStorageLabel(file.storage_driver) }}
              </span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="card-footer p-2 d-flex gap-1 bg-transparent border-top-0">
            <a
              :href="file.full_url"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-sm btn-outline-secondary flex-fill"
              title="在新窗口打开"
            >
              <i class="bi bi-box-arrow-up-right"></i>
            </a>
            <button
              v-if="canDelete(file)"
              class="btn btn-sm btn-outline-danger flex-fill"
              title="删除"
              @click="confirmDelete(file)"
            >
              <i class="bi bi-trash3"></i>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 无数据状态 -->
    <div v-else class="card">
      <div class="card-body text-center py-5">
        <i class="bi bi-inbox text-muted fs-1"></i>
        <p class="mt-3 text-muted mb-0">暂无附件</p>
        <p v-if="hasActiveFilters" class="text-muted small mt-1">
          没有符合筛选条件的附件，<a href="javascript:;" @click="resetFilters">重置筛选</a>
        </p>
      </div>
    </div>

    <!-- 分页 -->
    <div v-if="totalPages > 1" class="d-flex justify-content-center mt-4">
      <nav>
        <ul class="pagination mb-0">
          <li class="page-item" :class="{ disabled: currentPage <= 1 }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage - 1)">
              <i class="bi bi-chevron-left"></i>
            </a>
          </li>
          <li
            v-for="page in visiblePages"
            :key="page"
            class="page-item"
            :class="{ active: page === currentPage, disabled: page === '...' }"
          >
            <a class="page-link" href="#" @click.prevent="page !== '...' && changePage(page)">
              {{ page }}
            </a>
          </li>
          <li class="page-item" :class="{ disabled: currentPage >= totalPages }">
            <a class="page-link" href="#" @click.prevent="changePage(currentPage + 1)">
              <i class="bi bi-chevron-right"></i>
            </a>
          </li>
        </ul>
      </nav>
    </div>

    <!-- 图片预览模态框 -->
    <div
      ref="previewModalEl"
      class="modal fade"
      tabindex="-1"
      aria-hidden="true"
    >
      <div class="modal-dialog modal-lg modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title text-truncate">{{ previewFile?.original_name || '预览' }}</h5>
            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
          </div>
          <div class="modal-body text-center">
            <img
              v-if="previewFile && isImage(previewFile)"
              :src="previewFile.full_url"
              :alt="previewFile.original_name"
              class="img-fluid rounded"
              @error="handleImageError"
            >
            <div v-else-if="previewFile" class="py-5">
              <i :class="getFileIcon(previewFile)" class="text-muted" style="font-size: 4rem;"></i>
              <p class="mt-3 text-muted">此文件类型不支持在线预览</p>
              <a
                :href="previewFile.full_url"
                :download="previewFile.original_name"
                class="btn btn-primary"
              >
                <i class="bi bi-download me-1"></i>下载文件
              </a>
            </div>
          </div>
          <div v-if="previewFile" class="modal-footer justify-content-start">
            <div class="small text-muted">
              <span class="me-3">大小：{{ formatFileSize(previewFile.file_size) }}</span>
              <span class="me-3">类型：{{ previewFile.mime_type || '-' }}</span>
              <span v-if="previewFile.storage_driver">存储：{{ getStorageLabel(previewFile.storage_driver) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, reactive, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@/utils/network'
import { useCommStore } from '@/store/comm'
import { usePageTitle, toast } from '@/utils/app'
import * as bootstrap from 'bootstrap/dist/js/bootstrap.bundle.min.js'

const router = useRouter()

// 页面标题
const { setDynamicTitle } = usePageTitle({
  staticTitle: '附件管理',
  defaultTitle: '附件管理'
})

const store = useCommStore()

// 响应式数据
const loading = ref(false)
const error = ref('')
const attachments = ref([])
const currentPage = ref(1)
const pageSize = ref(12)
const totalCount = ref(0)

// 筛选条件
const searchKeyword = ref('')
const filters = reactive({
  target_type: '',
  file_ext: '',
  storage_driver: '',
  order: 'create_time desc'
})

// 筛选选项
const targetTypeOptions = ref(['article', 'comment', 'user_avatar'])
const fileExtOptions = ref(['jpg', 'png', 'gif', 'webp', 'bmp', 'svg', 'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'zip', 'rar', '7z', 'txt', 'md'])

// 预览相关
const previewFile = ref(null)
const previewModalEl = ref(null)
let previewModalInstance = null

// 当前登录用户
const currentUser = computed(() => store.login?.user || null)
const isLoggedIn = computed(() => !!currentUser.value)
const isAdmin = computed(() => {
  const user = currentUser.value
  if (!user) return false
  if (user?.result?.auth?.all) return true
  const groups = user?.result?.auth?.group?.list || []
  return groups.some(g => g.key === 'admin')
})

// 登录/注册跳转
const handleToLogin = () => router.push('/login')
const handleToRegister = () => router.push('/register')

// 计算属性
const totalPages = computed(() => {
  return Math.ceil(totalCount.value / pageSize.value) || 1
})

const hasActiveFilters = computed(() => {
  return !!(searchKeyword.value || filters.target_type || filters.file_ext || filters.storage_driver)
})

const visiblePages = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  const pages = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    if (current <= 4) {
      for (let i = 1; i <= 5; i++) pages.push(i)
      pages.push('...')
      pages.push(total)
    } else if (current >= total - 3) {
      pages.push(1)
      pages.push('...')
      for (let i = total - 4; i <= total; i++) pages.push(i)
    } else {
      pages.push(1)
      pages.push('...')
      for (let i = current - 1; i <= current + 1; i++) pages.push(i)
      pages.push('...')
      pages.push(total)
    }
  }
  return pages
})

// 工具函数
const isImage = (file) => {
  const ext = (file.file_ext || '').toLowerCase()
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'].includes(ext)
}

const getFileIcon = (file) => {
  const ext = (file.file_ext || '').toLowerCase()
  if (['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp', 'svg'].includes(ext)) return 'bi bi-file-image'
  if (['pdf'].includes(ext)) return 'bi bi-file-pdf'
  if (['doc', 'docx'].includes(ext)) return 'bi bi-file-word'
  if (['xls', 'xlsx'].includes(ext)) return 'bi bi-file-excel'
  if (['ppt', 'pptx'].includes(ext)) return 'bi bi-file-ppt'
  if (['zip', 'rar', '7z'].includes(ext)) return 'bi bi-file-zip'
  if (['txt', 'md'].includes(ext)) return 'bi bi-file-text'
  return 'bi bi-file-earmark'
}

const formatFileSize = (bytes) => {
  if (!bytes || isNaN(bytes)) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = Number(bytes)
  let i = 0
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

const formatRelativeTime = (timestamp) => {
  if (!timestamp) return ''
  const now = Date.now()
  let time = parseInt(timestamp)
  if (!isNaN(time) && time < 10000000000) time *= 1000
  if (isNaN(time)) return ''
  const diff = now - time
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  if (seconds < 60) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 30) return `${days}天前`
  const d = new Date(time)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

const getStorageLabel = (driver) => {
  const map = { local: '本地', oss: 'OSS', cos: 'COS', kodo: 'KODO' }
  return map[driver] || driver
}

const canDelete = (file) => {
  if (!currentUser.value) return false
  if (isAdmin.value) return true
  return file.uid === currentUser.value.id
}

// 事件处理
const handleImageError = (event) => {
  event.target.style.display = 'none'
}

const handleSearch = () => {
  currentPage.value = 1
  fetchAttachments()
}

const clearSearch = () => {
  searchKeyword.value = ''
  currentPage.value = 1
  fetchAttachments()
}

const handleFilterChange = () => {
  currentPage.value = 1
  fetchAttachments()
}

const resetFilters = () => {
  searchKeyword.value = ''
  filters.target_type = ''
  filters.file_ext = ''
  filters.storage_driver = ''
  filters.order = 'create_time desc'
  currentPage.value = 1
  fetchAttachments()
}

const changePage = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  fetchAttachments()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

const previewAttachment = async (file) => {
  previewFile.value = file
  await nextTick()
  if (!previewModalInstance && previewModalEl.value) {
    previewModalInstance = new bootstrap.Modal(previewModalEl.value)
  }
  if (previewModalInstance) {
    previewModalInstance.show()
  }
}

const confirmDelete = (file) => {
  if (!confirm(`确定要删除附件「${file.original_name}」吗？`)) return
  deleteAttachment(file)
}

// API 调用
const fetchAttachments = async () => {
  loading.value = true
  error.value = ''

  try {
    const params = {
      page: currentPage.value,
      limit: pageSize.value,
      order: filters.order
    }

    if (searchKeyword.value.trim()) {
      params.like = JSON.stringify({ original_name: searchKeyword.value.trim() })
    }
    if (filters.target_type) params.target_type = filters.target_type
    if (filters.file_ext) params.file_ext = filters.file_ext
    if (filters.storage_driver) params.storage_driver = filters.storage_driver

    const res = await request.get('/api/attachment/all', params)

    if (res.code === 200) {
      attachments.value = res.data?.data || []
      totalCount.value = res.data?.count || 0
    } else if (res.code === 204) {
      attachments.value = []
      totalCount.value = 0
    } else {
      error.value = res.msg || '获取附件列表失败'
      attachments.value = []
    }
  } catch (err) {
    error.value = err?.message || '网络错误，请稍后重试'
    attachments.value = []
  } finally {
    loading.value = false
  }
}

const deleteAttachment = async (file) => {
  try {
    const res = await request.delete('/api/attachment/remove', { ids: String(file.id) })

    if (res.code === 200) {
      toast.success('删除成功')
      // 从列表中移除
      attachments.value = attachments.value.filter(item => item.id !== file.id)
      totalCount.value = Math.max(0, totalCount.value - 1)
      // 如果当前页空了，回到上一页
      if (attachments.value.length === 0 && currentPage.value > 1) {
        currentPage.value -= 1
        fetchAttachments()
      }
    } else {
      toast.error(res.msg || '删除失败')
    }
  } catch (err) {
    toast.error(err?.message || '删除失败')
  }
}

// 组件挂载
onMounted(() => {
  if (isLoggedIn.value) {
    fetchAttachments()
  }
})
</script>

<style scoped>
.attachment-card {
  transition: all 0.3s ease;
  border-radius: 0.75rem;
  overflow: hidden;
}

.attachment-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.1);
}

.attachment-preview {
  position: relative;
  width: 100%;
  height: 140px;
  background-color: var(--bs-tertiary-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  overflow: hidden;
}

.preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}

.attachment-card:hover .preview-img {
  transform: scale(1.05);
}

.preview-icon {
  font-size: 3rem;
  color: var(--bs-secondary-color);
}

.attachment-ext-badge {
  position: absolute;
  top: 8px;
  right: 8px;
  font-size: 0.7rem;
  padding: 0.25rem 0.5rem;
  backdrop-filter: blur(4px);
  background-color: rgba(255, 255, 255, 0.85);
}

.attachment-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--bs-body-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.storage-badge {
  font-size: 0.7rem;
}

/* 骨架加载器 */
.skeleton {
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s infinite;
  border-radius: 4px;
}

@keyframes skeleton-loading {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.skeleton-thumb {
  width: 100%;
  height: 140px;
  border-radius: 0.5rem;
}

.skeleton-line {
  height: 12px;
  border-radius: 6px;
}

.hover-shadow {
  transition: box-shadow 0.3s ease, transform 0.3s ease;
}

.transition-all {
  transition: all 0.3s ease;
}

/* 响应式 */
@media (max-width: 768px) {
  .attachment-preview {
    height: 120px;
  }

  .preview-icon {
    font-size: 2.5rem;
  }
}

/* 暗黑模式 */
[data-bs-theme=dark] {
  .attachment-card {
    background-color: var(--bs-card-bg);
    border-color: var(--bs-border-color);
  }

  .attachment-preview {
    background-color: var(--bs-tertiary-bg);
  }

  .attachment-ext-badge {
    background-color: rgba(33, 37, 41, 0.85);
    color: var(--bs-body-color);
  }

  .skeleton {
    background: linear-gradient(90deg, #333 25%, #444 50%, #333 75%);
  }
}
</style>
