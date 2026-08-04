<template>
  <div class="users-page-wrapper mt-2">
    <!-- 页面标题 -->
    <div class="card mb-2">
      <div class="card-body">
        <div class="wx-page-header mb-0">
          <h5 class="wx-page-title">
            <i class="bi bi-people"></i>
            用户列表
          </h5>
          <p class="text-muted mb-0">共有 {{ totalCount }} 位用户</p>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="row g-3">
      <div v-for="i in 8" :key="i" class="col-6 col-md-4 col-lg-3">
        <div class="card h-100">
          <div class="card-body text-center">
            <div class="wx-skeleton skeleton-avatar mx-auto mb-3"></div>
            <div class="wx-skeleton skeleton-name mx-auto mb-2"></div>
            <div class="wx-skeleton skeleton-badge mx-auto"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="card">
      <div class="wx-empty-state">
        <i class="bi bi-exclamation-circle text-danger"></i>
        <p class="mt-2">{{ error }}</p>
        <button @click="fetchUsers" class="wx-btn-outline mt-2">
          重试
        </button>
      </div>
    </div>

    <!-- 用户列表 -->
    <div v-else-if="users.length > 0" class="row g-3">
      <div
        v-for="user in users"
        :key="user.id"
        class="col-6 col-md-4 col-lg-3"
      >
        <div
          class="card wx-card-hover user-card h-100"
          @click="goToAuthor(user.id)"
          style="cursor: pointer;"
        >
          <div class="card-body text-center">
            <!-- 用户头像 -->
            <div class="avatar-wrapper mb-3">
              <i-avatar-frame
                :src="user.avatar || defaultAvatar"
                :frame="user.json?.frame || ''"
                :alt="user.nickname"
                size="64px"
                :frame-scale="1.6"
              />
              
              <!-- 在线状态 -->
              <div
                class="avatar-status"
                :class="{ 'online': user.login_time && (Date.now() / 1000 - user.login_time < 86400) }"
              ></div>
            </div>

            <!-- 用户昵称 -->
            <h5 class="user-nickname mb-1">{{ user.nickname || user.account }}</h5>

            <!-- 用户等级 -->
            <span class="wx-tag user-level-badge mb-2">
              Lv.{{ getUserLevel(user) }} {{ getLevelName(user) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 无数据状态 -->
    <div v-else class="wx-card">
      <div class="wx-empty-state">
        <i class="bi bi-people"></i>
        <p>暂无用户</p>
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, shallowRef } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@/utils/network'
import defaultAvatar from '@/assets/img/avatar.png'
import iAvatarFrame from '@/comps/custom/i-avatar-frame.vue'
import { cache } from '@/utils/network'
import { useCommStore } from '@/store/comm'
import { usePageTitle } from '@/utils/app'

const { setDynamicTitle } = usePageTitle({
  staticTitle: '用户列表',
  defaultTitle: '用户列表'
})

const router = useRouter()
const store = useCommStore()

// 响应式数据
const loading = ref(false)
const error = ref('')
const users = ref([])
const currentPage = ref(1)
const pageSize = ref(12)
const totalCount = ref(0)
const levelInfo = ref(null)

// 缓存排序后的等级数据，避免在模板 v-for 中每次调用都重新排序
const sortedLevels = shallowRef([])

// 计算属性
const totalPages = computed(() => {
  return Math.ceil(totalCount.value / pageSize.value) || 1
})

// 获取可见的页码
const visiblePages = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  const pages = []
  
  if (total <= 7) {
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
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

// 获取用户等级
const getUserLevel = (user) => {
  if (!sortedLevels.value.length) {
    return Math.floor((user.exp || 0) / 100)
  }
  
  const exp = user.exp || 0
  
  for (const level of sortedLevels.value) {
    if (exp >= level.exp) {
      return level.value
    }
  }
  
  return Math.floor(exp / 100)
}

// 获取等级名称
const getLevelName = (user) => {
  if (!sortedLevels.value.length) {
    return ''
  }
  
  const exp = user.exp || 0
  
  for (const level of sortedLevels.value) {
    if (exp >= level.exp) {
      return level.name
    }
  }
  
  return ''
}

// 处理头像加载错误
const handleAvatarError = (event) => {
  event.target.src = defaultAvatar
}

// 跳转到用户主页
const goToAuthor = (userId) => {
  router.push(`/author/${userId}`)
}

// 切换页面
const changePage = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  fetchUsers()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 获取用户列表
const fetchUsers = async () => {
  loading.value = true
  error.value = ''
  
  try {
    const cacheKey = `users_list_page_${currentPage.value}`
    const cacheExpire = 5
    
    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      users.value = cachedData.users
      totalCount.value = cachedData.totalCount
      if (levelInfo.value) {
        loading.value = false
        return
      }
    }
    
    const res = await request.get('/api/users/all', {
      page: currentPage.value,
      limit: pageSize.value,
      order: 'create_time desc',
      where: 'status=0'
    })
    
    if (res.code === 200) {
      if (res.data && res.data.data) {
        users.value = res.data.data
        totalCount.value = res.data.count || 0
      } else if (res.data && Array.isArray(res.data)) {
        users.value = res.data
        totalCount.value = res.count || 0
      }
      cache.set(cacheKey, { users: users.value, totalCount: totalCount.value }, cacheExpire)
    } else {
      error.value = res.msg || '获取用户列表失败'
      users.value = []
    }
  } catch (err) {
    error.value = '网络错误，请稍后重试'
    users.value = []
  } finally {
    loading.value = false
  }
}

// 获取等级数据
const fetchLevelInfo = async () => {
  try {
    const cacheKey = 'users_level_info'
    const cacheExpire = 60 * 60
    
    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      levelInfo.value = cachedData
      updateSortedLevels()
      return
    }
    
    const res = await request.get('/api/level/all', {
      page: 1,
      limit: 50
    })
    
    if (res.code === 200) {
      levelInfo.value = res
      cache.set(cacheKey, res, cacheExpire)
      updateSortedLevels()
    }
  } catch (err) {
    levelInfo.value = null
  }
}

// 更新排序后的等级数据
const updateSortedLevels = () => {
  const data = levelInfo.value?.data?.data
  sortedLevels.value = data ? [...data].sort((a, b) => b.exp - a.exp) : []
}

// 组件挂载
onMounted(() => {
  fetchUsers()
  fetchLevelInfo()
})
</script>

<style scoped>
/* 用户卡片：基于 wx-card，补充专属细节 */
.user-card {
  cursor: pointer;
}

.avatar-wrapper {
  position: relative;
  display: inline-block;
}

.user-card:hover .avatar-wrapper {
  transform: scale(1.05);
}

.avatar-status {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background-color: var(--bs-tertiary-color);
  border: 2px solid var(--bs-body-bg);
  transition: var(--wx-transition);
}

.avatar-status.online {
  background-color: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
}

.user-nickname {
  font-size: 1rem;
  font-weight: 600;
  color: var(--bs-heading-color);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
  transition: var(--wx-transition);
}

.user-card:hover .user-nickname {
  color: var(--bs-primary);
}

/* 等级徽章：基于 wx-tag，强调成功色 */
.user-level-badge {
  background: rgba(var(--bs-success-rgb), 0.12);
  color: var(--bs-success);
  border-color: rgba(var(--bs-success-rgb), 0.25);
}

/* 骨架尺寸（基于 wx-skeleton 动画） */
.skeleton-avatar {
  width: 80px;
  height: 80px;
  border-radius: 50%;
}

.skeleton-name {
  height: 1.2rem;
  width: 100px;
}

.skeleton-badge {
  height: 1.2rem;
  width: 80px;
  border-radius: 1rem;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .user-nickname {
    font-size: 0.9rem;
  }
}
</style>