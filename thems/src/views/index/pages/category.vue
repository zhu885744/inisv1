<template>
  <div class="category-page-wrapper">
    <!-- 加载状态 -->
    <div v-if="loading">
      <!-- 分类信息卡片骨架 -->
      <div class="category-info wx-card p-4 mt-2">
        <div class="category-info-inner">
          <!-- 分类头像骨架 -->
          <div class="category-info-avatar">
            <div class="wx-skeleton skeleton-avatar"></div>
          </div>

          <!-- 分类信息骨架 -->
          <div class="category-info-content">
            <div class="wx-skeleton skeleton-category-title mb-3"></div>
            <div class="wx-skeleton skeleton-category-description mb-4"></div>
          </div>
        </div>
      </div>

      <!-- 文章列表骨架 -->
      <div
        :class="[
          'article-list-container mt-2',
          displayMode === 'grid' ? 'grid-article-list' : (displayMode === 'horizontal' ? 'horizontal-article-list' : 'list-article-list'),
        ]"
      >
        <div
          v-for="i in 6"
          :key="`skeleton-${i}`"
          :class="[
            'wx-card',
            displayMode === 'grid' ? 'article-item-card' : (displayMode === 'horizontal' ? 'article-item-horizontal mt-2' : 'article-item-list mt-2'),
          ]"
        >
          <!-- 网格卡片模式骨架 -->
          <div
            v-if="displayMode === 'grid'"
            class="card-body p-0 d-flex flex-column h-100"
          >
            <!-- 封面骨架 -->
            <div class="article-cover flex-shrink-0">
              <div class="wx-skeleton skeleton-cover"></div>
            </div>

            <!-- 内容骨架 -->
            <div
              class="article-content p-2 flex-grow-1 d-flex flex-column"
            >
              <!-- 标题骨架 -->
              <div class="wx-skeleton skeleton-title mb-1"></div>

              <!-- 摘要骨架 -->
              <div class="wx-skeleton skeleton-desc mt-auto mb-1"></div>

              <!-- 元信息骨架 -->
              <div class="d-flex justify-content-between mt-2">
                <div class="wx-skeleton skeleton-meta-left"></div>
                <div class="wx-skeleton skeleton-meta-right"></div>
              </div>
            </div>
          </div>

          <!-- 列表模式骨架 -->
          <div v-else-if="displayMode === 'list'" class="card-body p-4">
            <div class="d-flex align-items-start gap-4">
              <div class="flex-shrink-0 w-10 h-10 bg-secondary-subtle rounded-lg"></div>
              <div class="flex-grow-1 min-width-0">
                <div class="wx-skeleton skeleton-title-list mb-2"></div>
                <div class="wx-skeleton skeleton-desc-list mb-2"></div>
                <div class="d-flex justify-content-between align-items-center mt-2">
                  <div class="wx-skeleton skeleton-meta-left"></div>
                  <div class="wx-skeleton skeleton-meta-right"></div>
                </div>
              </div>
            </div>
          </div>

          <!-- 横向图文模式骨架 -->
          <div v-else class="card-body p-0 d-flex h-100">
            <div class="article-horizontal-cover flex-shrink-0">
              <div class="wx-skeleton skeleton-horizontal-cover"></div>
            </div>
            <div class="article-horizontal-content p-3 flex-grow-1 d-flex flex-column">
              <div class="wx-skeleton skeleton-title-horizontal mb-2"></div>
              <div class="wx-skeleton skeleton-desc-horizontal mt-auto mb-2"></div>
              <div class="d-flex gap-1 mb-2">
                <div class="wx-skeleton skeleton-tag"></div>
                <div class="wx-skeleton skeleton-tag"></div>
                <div class="wx-skeleton skeleton-tag"></div>
              </div>
              <div class="d-flex justify-content-between mt-auto">
                <div class="wx-skeleton skeleton-meta-left-horizontal"></div>
                <div class="wx-skeleton skeleton-meta-right-horizontal"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 分页骨架 -->
      <div class="pagination-container mt-4">
        <div class="wx-skeleton skeleton-pagination"></div>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="wx-card mt-2">
      <div class="wx-empty-state">
        <i class="bi bi-exclamation-circle text-danger"></i>
        <p>{{ errorMsg }}</p>
      </div>
    </div>

    <!-- 分类页面主体 -->
    <div v-else class="category-main">
      <!-- 分类信息卡片 -->
      <div class="category-info wx-card wx-card-hover mt-2">
        <div class="category-info-inner">
          <!-- 分类头像 -->
          <div class="category-info-avatar">
            <img
              :src="categoryInfo.avatar || defaultCover"
              :alt="categoryInfo.name"
              class="category-info-avatar-img rounded-3"
              @error="handleImageError"
            />
          </div>

          <!-- 分类信息 -->
          <div class="category-info-content">
            <h1 class="category-title fw-bold mb-3">
              {{ categoryInfo.name }}
              <span class="text-sm text-muted">({{ articleCount }})</span>
            </h1>
            <p
              v-if="categoryInfo.description"
              class="category-description text-muted mb-4"
            >
              {{ categoryInfo.description }}
            </p>
          </div>
        </div>
      </div>

      <!-- 文章列表 -->
      <div
        :class="[
          'article-list-container mt-2',
          displayMode === 'grid' ? 'grid-article-list' : (displayMode === 'horizontal' ? 'horizontal-article-list' : 'list-article-list'),
        ]"
      >
        <!-- 空状态 -->
        <div
          v-if="articles.length === 0"
          class="wx-card col-12"
        >
          <div class="wx-empty-state">
            <i class="bi bi-file-earmark-text"></i>
            <p>该分类下暂无文章</p>
          </div>
        </div>

        <div
          v-for="article in articles"
          :key="article.id"
          :class="[
            'wx-card wx-card-hover',
            displayMode === 'grid'
              ? 'article-item-card'
              : (displayMode === 'horizontal' ? 'article-item-horizontal mt-2' : 'article-item-list mt-2'),
          ]"
          @click="goToArticle(article.id)"
          style="cursor: pointer"
        >
          <!-- 网格卡片模式布局 -->
          <div
            v-if="displayMode === 'grid'"
            class="card-body p-0 d-flex flex-column h-100"
          >
            <!-- 文章封面 -->
            <div class="article-cover flex-shrink-0">
              <img
                :src="getCoverImg(article)"
                :alt="article.title"
                class="article-cover-img w-100 h-100 object-cover"
                loading="lazy"
                decoding="async"
                @error="handleImageError"
              />
            </div>

            <!-- 文章内容 -->
            <div class="article-content p-2 flex-grow-1 d-flex flex-column">
              <!-- 文章标题 -->
              <h3 class="article-title fw-bold mb-1 m-0">
                {{ article.title }}
              </h3>

              <!-- 文章摘要 -->
              <p class="article-desc text-truncate-1 mt-auto mb-1">
                {{ article.abstract || "暂无摘要" }}
              </p>

              <!-- 元信息 -->
              <div class="article-meta d-flex align-items-center w-100 m-0">
                <div class="meta-left d-flex align-items-center gap-0.5">
                  <span class="meta-item">
                    <i class="bi bi-folder-fill"></i>
                    {{ article?.result?.group?.[0]?.name || "未分类" }}
                  </span>
                </div>
                <div class="meta-right d-flex align-items-center gap-0.5 ms-auto">
                  <span class="meta-item">
                    <i class="bi bi-calendar-fill"></i>
                    {{ formatTime(article.publish_time) }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- 列表模式布局 -->
          <div v-else-if="displayMode === 'list'" class="card-body p-4">
            <div class="d-flex align-items-start gap-4">
              <div class="flex-grow-1 min-width-0">
                <div class="d-flex align-items-center gap-2 mb-2">
                  <span class="wx-tag text-xs">{{ article?.result?.group?.[0]?.name || "未分类" }}</span>
                  <span class="text-xs text-muted">{{ formatTime(article.publish_time) }}</span>
                </div>
                <h3 class="article-title-list h5 fw-bold mb-2">
                  {{ article.title }}
                </h3>
                <p class="article-desc-list text-muted mb-3">
                  {{ article.abstract || "暂无摘要" }}
                </p>
                <div class="d-flex align-items-center gap-4 article-meta-info">
                  <span><i class="bi bi-person-fill me-1"></i>{{ article?.result?.author?.nickname || '匿名' }}</span>
                  <span><i class="bi bi-eye-fill me-1"></i>{{ article.views || 0 }}</span>
                  <span><i class="bi bi-heart-fill me-1"></i>{{ article?.result?.like?.length || 0 }}</span>
                  <span><i class="bi bi-chat-fill me-1"></i>{{ article?.result?.comment?.count || 0 }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 横向图文模式布局 -->
          <div v-else class="card-body p-4 d-flex gap-4 h-100">
            <!-- 文章封面（左侧） -->
            <div class="article-horizontal-cover flex-shrink-0">
              <img
                :src="getCoverImg(article)"
                :alt="article.title"
                class="article-horizontal-cover-img w-full h-full object-cover"
                loading="lazy"
                decoding="async"
                @error="handleImageError"
              />
            </div>
            <!-- 内容（右侧） -->
            <div class="article-horizontal-content flex-grow-1 d-flex flex-column min-width-0">
              <!-- 标签行：分类标签 -->
              <div class="flex-shrink-0 mb-2">
                <span class="wx-tag article-horizontal-tag text-xs" style="background: var(--bs-primary); color: #fff; border-color: var(--bs-primary);">{{ article?.result?.group?.[0]?.name || "未分类" }}</span>
              </div>
              <!-- 文章标题 -->
              <h3 class="article-title-horizontal fw-bold m-0">
                {{ article.title }}
              </h3>
              <!-- 文章摘要 -->
              <p class="article-desc-horizontal text-muted flex-grow-1 mb-3">
                {{ truncateAbstract(article.abstract) }}
              </p>
              <!-- 底部信息：分类/日期/浏览/评论 + 标签 -->
              <div class="article-horizontal-footer flex-shrink-0 d-flex align-items-center justify-content-between w-full">
                <div class="article-horizontal-meta d-flex align-items-center gap-3 text-body-secondary text-sm">
                  <span class="meta-item-horizontal"><i class="bi bi-folder-fill me-1"></i>{{ article?.result?.group?.[0]?.name || '未分类' }}</span>
                  <span class="meta-item-horizontal"><i class="bi bi-calendar-fill me-1"></i>{{ formatTime(article.publish_time) }}</span>
                  <span class="meta-item-horizontal"><i class="bi bi-eye-fill me-1"></i>{{ article.views || 0 }}</span>
                  <span class="meta-item-horizontal"><i class="bi bi-chat-fill me-1"></i>{{ article?.result?.comment?.count || 0 }}</span>
                </div>
                <div class="article-horizontal-tags d-flex flex-wrap gap-1">
                  <span
                    v-for="tag in getArticleTags(article)"
                    :key="tag"
                    class="text-xs text-body-secondary"
                  >
                    #{{ tag }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="total > 0" class="pagination-container mt-4">
        <nav aria-label="Page navigation">
          <ul class="pagination justify-content-center">
            <li
              class="page-item"
              :class="{ disabled: currentPage === 1 }"
            >
              <button class="page-link" @click="changePage(currentPage - 1)">
                <span aria-hidden="true">&laquo;</span>
              </button>
            </li>
            <li class="page-item active">
              <span class="page-link">
                {{ currentPage }} / {{ pageCount }}
              </span>
            </li>
            <li
              class="page-item"
              :class="{ disabled: currentPage === pageCount }"
            >
              <button class="page-link" @click="changePage(currentPage + 1)">
                <span aria-hidden="true">&raquo;</span>
              </button>
            </li>
          </ul>
        </nav>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { request } from '@/utils/network'

// 导入本地静态资源
import defaultCover from '@/assets/img/fm.avif'

import { useCommStore } from '@/store/comm'
import { usePageTitle } from '@/utils/app'

// 全局状态管理
const store = {
  comm: useCommStore()
}

const router = useRouter()
const route = useRoute()

// 页面标题管理
const { setDynamicTitle, setLoadingTitle, setErrorTitle } = usePageTitle({
  staticTitle: '分类',
  defaultTitle: '分类页面'
})

// 响应式状态
const loading = ref(true)                 // 是否处于加载状态
const error = ref(false)                  // 是否发生错误
const errorMsg = ref('')                 // 错误信息
const categoryInfo = ref({})             // 当前分类信息
const articles = ref([])                // 文章列表
const currentPage = ref(1)              // 当前页码
const total = ref(0)                    // 文章总数
const limit = ref(10)                   // 每页条数
const articleCount = ref(0)             // 分类下文章总数
const displayMode = ref('grid')          // 显示模式：grid=网格卡片模式，list=列表模式，horizontal=横向图文模式

// 获取文章标签
const getArticleTags = (article) => {
  const tags = article?.result?.tags || []
  return tags.slice(0, 3).map(t => t.name)
}

// 截断摘要，限制150字
const truncateAbstract = (text, maxLength = 150) => {
  if (!text) return '暂无摘要'
  const str = String(text).trim()
  if (str.length <= maxLength) return str
  return str.substring(0, maxLength) + '...'
}

// 从后端API获取显示模式设置
const loadDisplayMode = async () => {
  try {
    // 优先从 store 读取（siteInfo 已经在应用初始化时缓存过了）
    if (store.comm.siteInfo?.display_mode !== undefined) {
      let mode = store.comm.siteInfo.display_mode
      if (typeof mode === 'boolean') {
        mode = mode ? 'grid' : 'list'
      }
      displayMode.value = mode || 'grid'
      return
    }
    // 如果 store 没有，再请求 API（兜底）
    const response = await request.get('/api/config/one', { key: 'cardify_functions' })
    if (response.code === 200 && response.data) {
      const config = response.data.json || {}
      let mode = config.display_mode
      if (typeof mode === 'boolean') {
        mode = mode ? 'grid' : 'list'
      }
      displayMode.value = mode || 'grid'
    }
  } catch (error) {
    console.error('读取显示模式设置失败:', error)
    displayMode.value = 'grid'
  }
}

// 计算总页数
const pageCount = computed(() => {
  return Math.ceil(total.value / limit.value)
})

/**
 * 从路由中获取当前分类 ID
 */
const getCurrentCategoryId = () => {
  return route.params.id
}

/**
 * 格式化时间戳为中文日期
 */
const formatTime = (timestamp) => {
  if (!timestamp || timestamp === 0) return '未知时间'
  const date = new Date(timestamp * 1000)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

/**
 * 跳转到文章详情页
 */
const goToArticle = (articleId) => {
  const validArticleId = parseInt(articleId)
  if (!isNaN(validArticleId) && validArticleId > 0) {
    router.push(`/archives/${validArticleId}`)
  }
}

/**
 * 校验分类参数合法性
 */
const checkCategoryParam = (param) => {
  const paramVal = String(param).trim()
  if (!paramVal) {
    errorMsg.value = '分类参数不能为空，请检查访问地址'
    return false
  }
  return true
}

/**
 * 获取文章封面图（响应式处理）
 */
const getCoverImg = (article) => {
  // 优先使用文章自身封面
  if (article.covers && article.covers.trim() !== '') {
    return article.covers
  }
  // 兜底使用默认封面
  return defaultCover
}

/**
 * 图片加载失败回调
 */
const handleImageError = (event) => {
  const img = event.target
  if (img.src !== defaultCover) {
    img.src = defaultCover
  }
  img.onerror = null
}

/**
 * 获取分类下文章总数
 */
const getCategoryArticleCount = async (categoryId) => {
  try {
    const like = `Group|%7C${categoryId}%7C`
    const res = await request.get(`/api/article/count?like=${like}`)
    articleCount.value = res.code === 200 ? res.data || 0 : 0
  } catch {
    articleCount.value = 0
  }
}

/**
 * 获取分类详情
 */
const getCategoryDetail = async (categoryParam) => {
  setLoadingTitle()
  loading.value = true
  error.value = false

  try {
    const res = await request.get('/api/article-group/all', { cache: false })

    if (res.code === 200 && res.data?.data?.length) {
      const categories = res.data.data
      const matched =
        categories.find(c => c.key === categoryParam) ||
        categories.find(c => c.id == categoryParam)

      if (matched) {
        categoryInfo.value = matched
        await getCategoryArticleCount(matched.id)
        setDynamicTitle(matched.name)
      } else {
        error.value = true
        errorMsg.value = '未找到该分类'
        setErrorTitle('分类不存在')
      }
    } else {
      error.value = true
      errorMsg.value = '获取分类列表失败'
      setErrorTitle('获取失败')
    }
  } catch {
    error.value = true
    errorMsg.value = '网络异常，请稍后重试'
    setErrorTitle('网络异常')
  } finally {
    loading.value = false
  }
}

/**
 * 获取分类下的文章列表
 */
const getCategoryArticles = async (page = 1) => {
  try {
    const like = `Group|%7C${categoryInfo.value.id}%7C`
    const url = `/api/article/all?like=${like}&where[audit]=1&page=${page}&limit=${limit.value}&order=create_time+desc&cache=false`
    const res = await request.get(url)

    if (res.code === 200) {
      articles.value = res.data?.data || []
      total.value = res.data?.count || 0
    }
  } catch {
    articles.value = []
    total.value = 0
  }
}

/**
 * 分页切换
 */
const changePage = (page) => {
  if (page < 1 || page > pageCount.value) return
  currentPage.value = page
  getCategoryArticles(page)
}

/**
 * 初始化页面
 */
const initPage = async (categoryParam) => {
  loadDisplayMode()
  if (checkCategoryParam(categoryParam)) {
    await getCategoryDetail(categoryParam)
    if (categoryInfo.value.id) {
      await getCategoryArticles(currentPage.value)
    }
  } else {
    error.value = true
    loading.value = false
  }
}

// 监听路由参数变化
watch(
  () => route.params.id,
  async (newId) => {
    if (newId) {
      loading.value = true
      categoryInfo.value = {}
      articles.value = []
      total.value = 0
      currentPage.value = 1
      await initPage(newId)
    }
  }
)

// 页面挂载
onMounted(async () => {
  await initPage(getCurrentCategoryId())
})
</script>

<style scoped>
/* 骨架加载器尺寸（使用全局 wx-skeleton 基础样式） */
.skeleton-avatar {
  width: 100px;
  height: 100px;
  border-radius: var(--wx-radius-sm);
}

.skeleton-category-title {
  height: 2.5rem;
  width: 70%;
}

.skeleton-category-description {
  height: 1.1rem;
  width: 100%;
}

.skeleton-cover {
  width: 100%;
  height: 100%;
  position: absolute;
  top: 0;
  left: 0;
  border-radius: 0;
}

.skeleton-title {
  height: 1.2rem;
  width: 80%;
}

.skeleton-desc {
  height: 0.6rem;
  width: 100%;
}

.skeleton-meta-left {
  height: 0.7rem;
  width: 40%;
}

.skeleton-meta-right {
  height: 0.7rem;
  width: 30%;
}

.skeleton-title-list {
  height: 1.5rem;
  width: 90%;
}

.skeleton-desc-list {
  height: 0.9rem;
  width: 100%;
  margin-bottom: 0.5rem;
}

/* 横向图文模式骨架屏 */
.skeleton-horizontal-cover {
  width: 100%;
  height: 100%;
  position: absolute;
  top: 0;
  left: 0;
  border-radius: 0;
}

.skeleton-title-horizontal {
  height: 1.5rem;
  width: 80%;
}

.skeleton-desc-horizontal {
  height: 0.9rem;
  width: 100%;
  margin-bottom: 0.5rem;
}

.skeleton-tag {
  height: 0.7rem;
  width: 3rem;
  border-radius: var(--wx-radius-sm);
}

.skeleton-meta-left-horizontal {
  height: 0.7rem;
  width: 50%;
}

.skeleton-meta-right-horizontal {
  height: 0.7rem;
  width: 30%;
}

.skeleton-pagination {
  height: 2.5rem;
  width: 200px;
  margin: 0 auto;
  border-radius: var(--wx-radius-sm);
}

/* 分类信息卡片 */
.category-info {
  background: linear-gradient(135deg, var(--bs-tertiary-bg), var(--bs-body-bg));
}

.category-info-inner {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  flex-wrap: wrap;
  padding: 1.5rem;
}

/* 分类头像 */
.category-info-avatar {
  flex-shrink: 0;
  position: relative;
}

.category-info-avatar-img {
  width: 100px;
  height: 100px;
  object-fit: cover;
  box-shadow: var(--wx-shadow-md);
  transition: var(--wx-transition);
  border: 3px solid var(--bs-body-bg);
}

.category-info-avatar-img:hover {
  transform: scale(1.05);
  box-shadow: var(--wx-shadow-lg);
}

/* 分类信息内容 */
.category-info-content {
  flex-grow: 1;
  min-width: 0;
}

/* 分类标题 */
.category-title {
  font-size: clamp(1.8rem, 5vw, 2.5rem);
  line-height: 1.3;
  font-weight: 700;
  margin-bottom: 0.75rem !important;
  color: var(--bs-heading-color);
  transition: var(--wx-transition);
}

/* 分类描述 */
.category-description {
  font-size: 1.1rem;
  line-height: 1.6;
  margin-bottom: 0 !important;
  color: var(--bs-secondary-color);
  transition: var(--wx-transition);
}

/* 文章数量标签 */
.category-title .text-muted {
  font-size: 0.8em;
  font-weight: 400;
  opacity: 0.8;
  transition: var(--wx-transition);
}

.category-title:hover .text-muted {
  opacity: 1;
  color: var(--bs-secondary);
}

/* 响应式调整 */
@media (max-width: 768px) {
  .category-info-inner {
    flex-direction: column;
    text-align: center;
    gap: 1rem;
  }
  
  .category-info-avatar-img {
    width: 80px;
    height: 80px;
  }
  
  .category-title {
    font-size: clamp(1.5rem, 5vw, 2rem);
  }
  
  .category-description {
    font-size: 1rem;
  }
}

/* 文章列表Grid布局 - 有图模式 */
.grid-article-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1.5rem;
  margin: 0 auto;
}

/* 文章列表横向布局 */
.horizontal-article-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

/* 文章卡片基础样式 */
.article-item-card,
.article-item-list,
.article-item-horizontal {
  transition: var(--wx-transition);
  position: relative;
  overflow: hidden;
}

/* 封面容器 */
.article-cover {
  width: 100%;
  padding-top: 66.67%;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, var(--bs-tertiary-bg), var(--bs-secondary-bg));
  border-radius: var(--wx-radius-lg) var(--wx-radius-lg) 0 0;
}

/* 懒加载图片样式 */
.article-cover-img {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: var(--wx-transition);
  border-radius: var(--wx-radius-lg) var(--wx-radius-lg) 0 0;
}

/* 内容区域 */
.article-content {
  height: 100%;
  padding: 1.25rem !important;
}

/* 图片样式 */
img {
  transition: var(--wx-transition);
  max-width: 100%;
  height: auto;
}

.article-cover:hover .article-cover-img {
  transform: scale(1.08);
  filter: brightness(1.05);
}

/* 横向图文模式封面容器 */
.article-horizontal-cover {
  width: 200px;
  height: 160px;
  flex-shrink: 0;
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, var(--bs-tertiary-bg), var(--bs-secondary-bg));
  border-radius: var(--wx-radius-md);
}

/* 横向图文模式封面图片 */
.article-horizontal-cover-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: var(--wx-transition);
  border-radius: var(--wx-radius-md);
}

/* 横向图文模式内容区域 */
.article-horizontal-content {
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

/* 横向图文模式分类标签 */
.article-horizontal-tag {
  font-weight: 500;
  padding: 2px 8px;
}

/* 横向图文模式标题 */
.article-title-horizontal {
  font-size: clamp(1.05rem, 2vw, 1.25rem);
  line-height: 1.5;
  font-weight: 700;
  color: var(--bs-body-color);
  transition: var(--wx-transition);
  margin-bottom: 0.5rem !important;
  white-space: normal;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.article-item-horizontal:hover .article-title-horizontal {
  color: var(--bs-primary);
}

/* 横向图文模式摘要 */
.article-desc-horizontal {
  font-size: 0.85rem;
  color: var(--bs-secondary-color);
  line-height: 1.6;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 横向图文模式底部信息 */
.article-horizontal-footer {
  margin-top: auto;
}

/* 横向图文模式元信息 */
.article-horizontal-meta {
  font-size: 0.8rem;
  line-height: 1.3;
}

.meta-item-horizontal {
  position: relative;
  display: flex;
  align-items: center;
  white-space: nowrap;
  padding-left: 0 !important;
  transition: var(--wx-transition);
}

.meta-item-horizontal:hover {
  color: var(--bs-primary);
}

.meta-item-horizontal .bi {
  font-size: 0.85em;
  margin-right: 0.25rem;
  line-height: 1;
  vertical-align: middle;
  color: var(--bs-tertiary-color);
  transition: var(--wx-transition);
}

.meta-item-horizontal:hover .bi {
  color: var(--bs-primary);
}

/* 横向图文模式标签 */
.article-horizontal-tags {
  justify-content: flex-end;
}

.article-horizontal-cover:hover .article-horizontal-cover-img {
  transform: scale(1.08);
  filter: brightness(1.05);
}

/* 标题 */
.article-title {
  font-size: clamp(1.05rem, 1.5vw, 1.25rem);
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 700;
  color: var(--bs-heading-color);
  transition: var(--wx-transition);
  margin-bottom: 0.75rem !important;
}

.article-item-card:hover .article-title {
  color: var(--bs-primary);
}

/* 摘要 */
.article-desc {
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
  line-height: 1.4;
  margin: 0 0 1rem 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

.text-truncate-1 {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 列表模式标题 */
.article-title-list {
  font-size: clamp(1.15rem, 2vw, 1.4rem);
  line-height: 1.5;
  font-weight: 700;
  color: var(--bs-heading-color);
  transition: var(--wx-transition);
  margin-bottom: 0.5rem !important;
}

.article-item-list:hover .article-title-list {
  color: var(--bs-primary);
}

/* 列表模式摘要 */
.article-desc-list {
  font-size: 0.9rem;
  color: var(--bs-secondary-color);
  line-height: 1.6;
  margin: 0 0 0.75rem 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 元信息 */
.article-meta {
  font-size: 0.75rem;
  color: var(--bs-tertiary-color);
  line-height: 1.3;
  margin-top: auto;
}

.meta-left, .meta-right {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.meta-item {
  position: relative;
  display: flex;
  align-items: center;
  white-space: nowrap;
  padding-left: 0 !important;
  transition: var(--wx-transition);
}

.meta-item:hover {
  color: var(--bs-primary);
}

.meta-item .bi {
  font-size: 0.9em;
  margin-right: 0.3rem;
  line-height: 1;
  vertical-align: middle;
  color: var(--bs-tertiary-color);
  transition: var(--wx-transition);
}

.meta-item:hover .bi {
  color: var(--bs-primary);
}

/* 列表模式元信息 */
.article-meta-info {
  font-size: 0.85rem;
  color: var(--bs-tertiary-color);
  transition: var(--wx-transition);
}

.article-meta-info .bi {
  font-size: 0.9em;
  color: var(--bs-tertiary-color);
  transition: var(--wx-transition);
}

.article-item-list:hover .article-meta-info {
  color: var(--bs-primary);
}

.article-item-list:hover .article-meta-info .bi {
  color: var(--bs-primary);
}

/* 分页样式 */
.pagination-container {
  margin-top: 3rem;
  margin-bottom: 3rem;
  display: flex;
  justify-content: center;
  align-items: center;
}

.pagination {
  background: var(--bs-body-bg);
  border-radius: var(--wx-radius-lg);
  box-shadow: var(--wx-shadow-sm);
  padding: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.page-item {
  margin: 0;
}

.page-link {
  padding: 0.6rem 1rem;
  border: none;
  border-radius: var(--wx-radius-md);
  background: transparent;
  color: var(--bs-secondary-color);
  font-size: 0.9rem;
  font-weight: 500;
  transition: var(--wx-transition);
  cursor: pointer;
  min-width: 2.5rem;
  text-align: center;
}

.page-link:hover:not(.disabled) {
  background: var(--bs-tertiary-bg);
  color: var(--bs-primary);
  transform: translateY(-1px);
}

.page-item.active .page-link {
  background: var(--wx-gradient-primary);
  color: #fff;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(var(--bs-primary-rgb), 0.35);
}

.page-item.disabled .page-link {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}



/* 动画 */
@keyframes fadeIn {
  from {
    opacity: 0.7;
    filter: blur(5px);
  }
  to {
    opacity: 1;
    filter: blur(0);
  }
}

/* 响应式调整 */
@media (max-width: 768px) {
  .grid-article-list {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.8rem;
  }
  .article-item-card {
    min-width: 160px;
  }
  .article-content {
    padding: 1.5px;
  }
  
  .category-title {
    font-size: clamp(1.5rem, 5vw, 2rem);
  }
  
  .category-description {
    font-size: 1rem;
  }
  
  /* 横向图文模式中等屏幕响应式 */
  .article-horizontal-cover {
    width: 150px;
    height: 120px;
  }
  
  .article-horizontal-content {
    padding: 0 !important;
  }
  
  .article-title-horizontal {
    font-size: 1rem;
    line-height: 1.4;
    margin-bottom: 0.5rem !important;
    -webkit-line-clamp: 1;
    line-clamp: 1;
  }
  
  .article-desc-horizontal {
    font-size: 0.8rem;
    line-height: 1.5;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    margin-bottom: 0.4rem !important;
  }
  
  .article-horizontal-meta {
    font-size: 0.7rem;
    flex-wrap: wrap;
    gap: 0.5rem !important;
  }
  
  .article-horizontal-footer {
    flex-direction: column;
    align-items: flex-start !important;
    gap: 0.3rem;
  }
}

@media (max-width: 576px) {
  .grid-article-list {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.8rem;
  }
  
  .article-item-card:hover {
    transform: translateY(-2px);
  }
  
  .article-title {
    font-size: 0.95rem;
    line-height: 1.4;
    margin-bottom: 0.5rem !important;
  }
  
  .article-desc {
    font-size: 0.75rem;
    line-height: 1.3;
    margin-bottom: 0.5rem;
  }
  
  .article-meta {
    font-size: 0.65rem;
    line-height: 1.2;
  }
  
  .article-content {
    padding: 1rem !important;
  }
  
  .meta-item {
    font-size: 0.65rem;
  }
  
  .meta-item .bi {
    font-size: 0.8em;
  }
  
  /* 横向图文模式移动端响应式 */
  .article-horizontal-cover {
    width: 110px;
    height: 90px;
  }
  
  .article-horizontal-content {
    padding: 0 !important;
  }
  
  .article-title-horizontal {
    font-size: 0.9rem;
    line-height: 1.4;
    margin-bottom: 0.4rem !important;
    -webkit-line-clamp: 1;
    line-clamp: 1;
  }
  
  .article-desc-horizontal {
    font-size: 0.75rem;
    line-height: 1.5;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    margin-bottom: 0.4rem !important;
  }
  
  .article-horizontal-meta {
    font-size: 0.65rem;
    flex-wrap: wrap;
    gap: 0.5rem !important;
  }
  
  .article-horizontal-tags {
    display: none !important;
  }
  
  .article-horizontal-footer {
    flex-direction: column;
    align-items: flex-start !important;
    gap: 0.3rem;
  }
}

/* 暗黑模式适配（大部分由 Bootstrap CSS 变量 + wx-* 类自动处理） */
[data-bs-theme=dark] {
  /* 分类信息卡片渐变背景 */
  .category-info {
    background: linear-gradient(135deg, var(--bs-tertiary-bg), var(--bs-body-bg));
  }

  /* 分类头像边框 */
  .category-info-avatar-img {
    border-color: var(--bs-secondary-bg);
  }

  /* 文章数量标签 */
  .category-title .text-muted {
    opacity: 0.7;
  }

  .category-title:hover .text-muted {
    opacity: 1;
    color: var(--bs-secondary);
  }

  /* 横向图文模式封面背景 */
  .article-horizontal-cover {
    background: linear-gradient(135deg, var(--bs-body-bg), var(--bs-secondary-bg));
  }

  .article-horizontal-cover-img.lazy-error {
    background: linear-gradient(135deg, var(--bs-body-bg), var(--bs-secondary-bg));
  }
}

/* 响应式调整 */
@media (max-width: 992px) {
  .grid-article-list {
    grid-template-columns: repeat(2, 1fr);
    gap: 1.25rem;
  }
}

@media (max-width: 768px) {
  .grid-article-list {
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  
  .article-item-card {
    min-width: 160px;
  }
  
  .article-content {
    padding: 1rem !important;
  }
  
  .article-title {
    font-size: 1rem;
    margin-bottom: 0.5rem !important;
  }
  
  .article-desc {
    font-size: 0.75rem;
    margin-bottom: 0.75rem;
  }
  
  .article-meta {
    font-size: 0.7rem;
  }
  
  /* 分页响应式 */
  .pagination-container {
    margin-top: 2rem;
    margin-bottom: 2rem;
  }
  
  .pagination {
    padding: 0.25rem;
  }
  
  .page-link {
    padding: 0.5rem 0.75rem;
    font-size: 0.85rem;
    min-width: 2rem;
  }
  
  .category-title {
    font-size: clamp(1.5rem, 5vw, 2rem);
  }
  
  .category-description {
    font-size: 1rem;
  }
}

@media (max-width: 576px) {
  .grid-article-list {
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
  }
  
  .article-item-card:hover {
    transform: translateY(-3px);
  }
  
  .article-title {
    font-size: 0.95rem;
    line-height: 1.4;
    margin-bottom: 0.5rem !important;
  }
  
  .article-desc {
    font-size: 0.75rem;
    line-height: 1.3;
    margin-bottom: 0.5rem;
  }
  
  .article-meta {
    font-size: 0.65rem;
    line-height: 1.2;
  }
  
  .article-content {
    padding: 1rem !important;
  }
  
  .meta-item {
    font-size: 0.65rem;
  }
  
  .meta-item .bi {
    font-size: 0.8em;
  }
}
</style>