<template>
  <div class="mt-2">
    <!-- 轮播图 -->
    <div v-if="banners.length > 0 || bannersLoading" class="mb-2">
      <div v-if="bannersLoading" class="carousel-loading card shadow-sm">
        <div class="skeleton skeleton-carousel"></div>
      </div>
      <div v-else id="carouselExampleControls" class="carousel slide position-relative">
        <div class="carousel-inner">
          <div 
            v-for="(banner, index) in banners" 
            :key="banner.id"
            class="carousel-item" 
            :class="{ active: index === 0 }"
          >
            <a :href="banner.url" :target="banner.target" class="d-block w-100">
              <img 
                :src="banner.image" 
                :alt="banner.title" 
                class="d-block w-100 carousel-img"
              >
            </a>
          </div>
        </div>
        <button class="carousel-control-prev" type="button" data-bs-target="#carouselExampleControls" data-bs-slide="prev">
          <span class="carousel-control-prev-icon" aria-hidden="true"></span>
          <span class="visually-hidden">Previous</span>
        </button>
        <button class="carousel-control-next" type="button" data-bs-target="#carouselExampleControls" data-bs-slide="next">
          <span class="carousel-control-next-icon" aria-hidden="true"></span>
          <span class="visually-hidden">Next</span>
        </button>
      </div>
    </div>

    <!-- 单条动态详情视图 -->
    <div v-if="singleMoment" class="moment-detail">
      <div class="mb-3">
        <router-link to="/" class="wx-btn-gradient">
          <i class="bi bi-arrow-left me-1"></i>返回动态列表
        </router-link>
      </div>

      <!-- 骨架屏 -->
      <div v-if="detailLoading" class="wx-card mb-3">
        <div class="card-body p-3">
          <div class="d-flex align-items-center mb-3">
            <div class="wx-skeleton me-2" style="width:40px;height:40px;"></div>
            <div>
              <div class="wx-skeleton mb-1" style="width:100px;height:16px;"></div>
              <div class="wx-skeleton" style="width:80px;height:12px;"></div>
            </div>
          </div>
          <div class="wx-skeleton mb-2" style="width:100%;height:14px;"></div>
          <div class="wx-skeleton mb-2" style="width:80%;height:14px;"></div>
          <div class="wx-skeleton" style="width:60%;height:14px;"></div>
        </div>
      </div>

      <!-- 动态卡片（评论区已内嵌） -->
      <div v-else-if="singleMoment" class="wx-card moment-item mb-3">
        <div class="card-body p-3">
          <MomentCard
            :moment="singleMoment"
            :is-login="isLogin"
            :is-dark-mode="isDarkMode"
            @edit="handleEdit"
            @delete="handleMomentDeleted"
            @comment-added="refreshCommentCount"
            @set-top="handleMomentSetTop"
            ref="momentCardRef"
          />
        </div>
      </div>
    </div>

    <!-- 动态列表视图 -->
    <div v-else>
      <!-- 发布动态编辑器（仅登录用户） -->
      <MomentEditor
        v-if="isLogin && showEditor"
        :edit-moment="editingMoment"
        :is-dark-mode="isDarkMode"
        @published="handleMomentPublished"
        @draft-saved="handleDraftSaved"
        @cancel-edit="handleCancelEdit"
        ref="editorRef"
      />

      <!-- 筛选栏 -->
      <div class="d-flex align-items-center justify-content-between mb-3">
        <div class="d-flex align-items-center gap-2">
          <button
            class="wx-btn-gradient"
          >
            全部
          </button>
        </div>

        <div class="d-flex align-items-center gap-2">
          <button
            v-if="isLogin"
            class="wx-btn-outline"
            @click="showEditor = !showEditor"
          >
            <i class="bi" :class="showEditor ? 'bi-eye-slash' : 'bi-pencil-square'"></i>
            {{ showEditor ? '隐藏编辑器' : '发布动态' }}
          </button>
        </div>
      </div>

      <!-- 骨架屏 -->
      <div v-if="loading" class="space-y-3">
        <div v-for="i in 3" :key="i" class="wx-card mb-3">
          <div class="card-body p-3">
            <div class="d-flex align-items-center mb-3">
              <div class="wx-skeleton me-2" style="width:40px;height:40px;"></div>
              <div>
                <div class="wx-skeleton mb-1" style="width:100px;height:16px;"></div>
                <div class="wx-skeleton" style="width:80px;height:12px;"></div>
              </div>
            </div>
            <div class="wx-skeleton mb-2" style="width:100%;height:14px;"></div>
            <div class="wx-skeleton mb-2" style="width:80%;height:14px;"></div>
            <div class="wx-skeleton" style="width:60%;height:14px;"></div>
          </div>
        </div>
      </div>

      <!-- 动态列表 -->
      <div v-else-if="moments.length > 0">
        <div v-for="moment in moments" :key="moment.id" class="wx-card wx-card-hover moment-item mb-3">
          <div class="card-body p-3">
            <MomentCard
              :moment="moment"
              :is-login="isLogin"
              :is-dark-mode="isDarkMode"
              @edit="handleEdit"
              @delete="handleMomentDeleted"
              @comment-added="refreshCommentCount"
              @set-top="handleMomentSetTop"
            />
          </div>
        </div>

        <!-- 分页 -->
        <nav v-if="totalPages > 1" aria-label="动态分页" class="mt-4">
          <ul class="pagination justify-content-center">
            <li class="page-item" :class="{ disabled: currentPage === 1 }">
              <button class="page-link" @click="handlePageChange(currentPage - 1)" :disabled="currentPage === 1">
                <span aria-hidden="true">&laquo;</span>
              </button>
            </li>
            <li v-for="page in displayedPages" :key="page" class="page-item" :class="{ active: currentPage === page, 'd-none': page === '...' }">
              <button class="page-link" @click="handlePageChange(page)" v-if="page !== '...'">
                {{ page }}
              </button>
              <span class="page-link text-muted" v-else>...</span>
            </li>
            <li class="page-item" :class="{ disabled: currentPage === totalPages }">
              <button class="page-link" @click="handlePageChange(currentPage + 1)" :disabled="currentPage === totalPages">
                <span aria-hidden="true">&raquo;</span>
              </button>
            </li>
          </ul>
        </nav>
      </div>

      <!-- 空状态 -->
      <div v-else class="wx-empty-state">
        <i class="bi bi-inbox"></i>
        <p>暂无动态</p>
        <button v-if="isLogin" class="wx-btn-gradient mt-3" @click="showEditor = true">
          <i class="bi bi-pencil-square me-1"></i>发布第一条动态
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCommStore } from '@/store/comm'
import { request } from '@/utils/network'
import { toast } from '@/utils/app'
import MomentCard from '@/comps/moments/MomentCard.vue'
import MomentEditor from '@/comps/moments/MomentEditor.vue'
import { useBannerStore } from '@/store/banner'

const route = useRoute()
const router = useRouter()
const store = useCommStore()
const bannerStore = useBannerStore()

// 登录状态
const isLogin = computed(() => !!store.login?.finish)
const isDarkMode = computed(() => store.darkMode)

// 动态列表数据
const moments = ref([])
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const totalMoments = ref(0)
const filter = ref('all')

// 单条动态详情
const singleMoment = ref(null)
const detailLoading = ref(false)

// 编辑器
const showEditor = ref(true)
const editingMoment = ref(null)
const editorRef = ref(null)
const momentCardRef = ref(null)

// 轮播图
const banners = ref([])
const bannersLoading = ref(false)

// 评论展开由 MomentCard 内部管理

// 分页计算
const totalPages = computed(() => Math.ceil(totalMoments.value / pageSize.value))

const displayedPages = computed(() => {
  const pages = []
  const total = totalPages.value
  const current = currentPage.value

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    if (current <= 3) {
      pages.push(1, 2, 3, 4, '...', total)
    } else if (current >= total - 2) {
      pages.push(1, '...', total - 3, total - 2, total - 1, total)
    } else {
      pages.push(1, '...', current - 1, current, current + 1, '...', total)
    }
  }
  return pages
})

// 加载动态列表
const loadMoments = async () => {
  loading.value = true

  try {
    const params = {
      page: currentPage.value,
      limit: pageSize.value,
      order: 'top desc, create_time desc'
    }

    // 根据筛选条件设置参数
    if (filter.value === 'mine' && isLogin.value) {
      params.uid = store.login.user?.id
      params.audit = 1
    } else if (filter.value === 'draft' && isLogin.value) {
      params.uid = store.login.user?.id
      params.status = 0
    } else {
      params.audit = 1
    }

    const res = await request.get('/api/moments/all', params)

    if (res.code === 200) {
      moments.value = res.data?.data || res.data || []
      // 本地兜底排序：置顶优先(desc) + 时间倒序(desc)
      moments.value.sort((a, b) => {
        const topA = Number(a.top) || 0
        const topB = Number(b.top) || 0
        if (topA !== topB) return topB - topA
        return Number(b.create_time || 0) - Number(a.create_time || 0)
      })
      totalMoments.value = res.data?.count || moments.value.length || 0
    }
  } catch (err) {
    console.error('加载动态失败:', err)
    toast.error('加载动态失败')
  } finally {
    loading.value = false
  }
}

// 加载单条动态详情
const loadMomentDetail = async (id) => {
  detailLoading.value = true

  try {
    const res = await request.get('/api/moments/one', { id })

    if (res.code === 200) {
      singleMoment.value = res.data
    } else {
      toast.error(res.msg || '动态不存在')
      router.push('/')
    }
  } catch (err) {
    console.error('加载动态详情失败:', err)
    toast.error('加载失败')
    router.push('/')
  } finally {
    detailLoading.value = false
  }
}

// 筛选变化
const handleFilterChange = (newFilter) => {
  filter.value = newFilter
  currentPage.value = 1
  loadMoments()
}

// 页码变化
const handlePageChange = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  loadMoments()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 刷新评论数
const refreshCommentCount = () => {
  if (momentCardRef.value) {
    momentCardRef.value.refreshCounts()
  }
}
const handleEdit = (moment) => {
  editingMoment.value = moment
  showEditor.value = true
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 取消编辑
const handleCancelEdit = () => {
  editingMoment.value = null
}

// 发布/更新成功
const handleMomentPublished = () => {
  editingMoment.value = null

  if (singleMoment.value) {
    // 详情页模式，刷新详情
    loadMomentDetail(singleMoment.value.id)
  } else {
    // 列表模式，刷新列表
    currentPage.value = 1
    loadMoments()
  }
}

// 草稿保存成功
const handleDraftSaved = () => {
  if (filter.value === 'draft') {
    loadMoments()
  }
}

// 动态删除成功
const handleMomentDeleted = (momentId) => {
  if (singleMoment.value) {
    router.push('/')
  } else {
    moments.value = moments.value.filter(m => m.id !== momentId)
  }
}

// 动态置顶状态变更
const handleMomentSetTop = ({ id, top }) => {
  if (singleMoment.value && Number(singleMoment.value.id) === Number(id)) {
    singleMoment.value = { ...singleMoment.value, top }
  } else {
    const idx = moments.value.findIndex(m => Number(m.id) === Number(id))
    if (idx !== -1) {
      moments.value[idx] = { ...moments.value[idx], top }
      // 按置顶优先 + 时间倒序重排
      moments.value.sort((a, b) => {
        const topA = Number(a.top) || 0
        const topB = Number(b.top) || 0
        if (topA !== topB) return topB - topA
        return Number(b.create_time || 0) - Number(a.create_time || 0)
      })
    }
  }
}

// 获取轮播图数据
const getBanners = async () => {
  bannersLoading.value = true
  try {
    const { code, data } = await bannerStore.setCurrent()
    if (code === 200) {
      banners.value = data || []
    } else {
      banners.value = []
    }
  } catch (error) {
    console.error('获取轮播图数据失败', error)
    banners.value = []
  } finally {
    bannersLoading.value = false
  }
}

// 监听路由变化
watch(() => route.params.id, (newId) => {
  if (newId) {
    loadMomentDetail(newId)
  } else {
    singleMoment.value = null
    loadMoments()
  }
}, { immediate: true })

// 监听登录状态变化
watch(() => store.login?.finish, () => {
  if (!route.params.id) {
    loadMoments()
  }
})

onMounted(() => {
  getBanners()
  if (!route.params.id) {
    loadMoments()
  }
})
</script>

<style scoped>
/* 动态卡片容器：覆盖 wx-card 的 overflow:hidden，
   避免 MomentCard 操作面板下拉被裁切 */
.moment-item {
  overflow: visible;
}

/* 分页美化 */
.page-link {
  border-radius: var(--wx-radius-sm);
  margin: 0 0.2rem;
  color: var(--bs-body-color);
  border-color: var(--bs-border-color);
  transition: var(--wx-transition);
}

.page-link:hover {
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.05);
}

.page-item.active .page-link {
  background: var(--wx-gradient-primary);
  border-color: transparent;
  color: #fff;
  box-shadow: 0 4px 12px rgba(var(--bs-primary-rgb), 0.25);
}

.page-item.disabled .page-link {
  color: var(--bs-tertiary-color);
  opacity: 0.6;
}

/* ================= 轮播图样式 ================= */
.carousel-img {
  height: 300px;
  object-fit: cover;
  transition: transform 0.6s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  width: 100%;
}

.carousel-item {
  transition: transform 0.6s cubic-bezier(0.25, 0.46, 0.45, 0.94);
  border-radius: 0.25rem;
  overflow: hidden;
}

.carousel-item:hover .carousel-img {
  transform: scale(1.03);
}

/* 轮播图控制按钮 */
#carouselExampleControls .carousel-control-prev,
#carouselExampleControls .carousel-control-next {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 60px;
  height: 60px;
  opacity: 0.8 !important;
  transition: all 0.3s ease;
  display: flex !important;
  align-items: center;
  justify-content: center;
  z-index: 10;
  border-radius: 50%;
}

#carouselExampleControls .carousel-control-prev {
  left: 10px;
}

#carouselExampleControls .carousel-control-next {
  right: 10px;
}

#carouselExampleControls .carousel-control-prev:hover,
#carouselExampleControls .carousel-control-next:hover {
  opacity: 1 !important;
  transform: translateY(-50%) scale(1.1);
}

#carouselExampleControls .carousel-control-prev-icon,
#carouselExampleControls .carousel-control-next-icon {
  width: 1.5rem;
  height: 1.5rem;
  background-size: 100% 100%;
  background-color: rgba(255, 255, 255, 0.2);
  padding: 1.25rem;
  transition: all 0.3s ease;
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 0.25rem;
}

#carouselExampleControls .carousel-control-prev:hover .carousel-control-prev-icon,
#carouselExampleControls .carousel-control-next:hover .carousel-control-next-icon {
  background-color: rgba(255, 255, 255, 0.3);
  transform: scale(1.1);
  border-color: rgba(255, 255, 255, 0.5);
}

/* 轮播图骨架屏 */
.carousel-loading {
  width: 100%;
  padding-top: 40%;
  position: relative;
  overflow: hidden;
}

.skeleton-carousel {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border-radius: 0;
}

/* ================= 响应式轮播图 ================= */
@media (max-width: 992px) {
  .carousel-img {
    height: 300px;
  }
}

@media (max-width: 768px) {
  .carousel-img {
    height: 250px;
  }
}

@media (max-width: 576px) {
  .carousel-img {
    height: 200px;
  }
}

@media (max-width: 400px) {
  .carousel-img {
    height: 180px;
  }
}

/* 移动端适配 */
@media (max-width: 768px) {
  .page-link {
    margin: 0 0.1rem;
  }
}
</style>
