<!-- 搜索弹窗组件 -->
<template>
  <transition name="search-modal">
    <div
      v-if="visible"
      class="search-modal-overlay"
      @click.self="hide"
    >
      <div class="search-modal-container">
        <div class="search-modal-content">
          <!-- 头部区域 -->
          <div class="search-modal-header">
            <h3 class="search-modal-title">搜索</h3>
            <button
              class="search-modal-close"
              @click="hide"
              aria-label="关闭"
            >
              <i class="bi bi-x-lg"></i>
            </button>
          </div>

          <!-- 搜索输入区域 -->
          <div class="search-input-wrapper">
            <div class="search-input-group">
              <i class="bi bi-search search-icon"></i>
              <input
                ref="searchInputRef"
                type="text"
                v-model="searchQuery"
                class="search-input"
                :placeholder="getSearchPlaceholder()"
                @input="handleInput"
                @keydown.up.prevent="handleKeyUp"
                @keydown.down.prevent="handleKeyDown"
                @keydown.enter.prevent="handleEnterKey"
                @keydown.escape.prevent="hide"
              />
              <button
                v-if="searchQuery"
                class="search-clear-btn"
                @click="clearQuery"
                aria-label="清除"
              >
                <i class="bi bi-x"></i>
              </button>
            </div>

            <!-- 搜索范围选择 -->
            <div class="search-scopes">
              <button
                v-for="scope in scopes"
                :key="scope.key"
                class="search-scope-btn"
                :class="{ active: searchScope === scope.key }"
                @click="changeSearchScope(scope.key)"
                :title="scope.label"
              >
                <i :class="scope.icon"></i>
                <span>{{ scope.label }}</span>
              </button>
            </div>
          </div>

          <!-- 搜索内容区域 -->
          <div class="search-content wx-scrollable">
            <!-- 热门搜索 -->
            <div v-if="!loading && !searchQuery && searchHistory.length === 0" class="search-section">
              <div class="section-header">
                <i class="bi bi-trending-up"></i>
                <span>热门搜索</span>
              </div>
              <div class="search-tags">
                <span
                  v-for="(item, index) in hotSearches"
                  :key="index"
                  class="search-tag"
                  @click="useSearchHistory(item)"
                >
                  {{ item }}
                </span>
              </div>
            </div>

            <!-- 搜索历史 -->
            <div v-if="!loading && !searchQuery && searchHistory.length > 0" class="search-section">
              <div class="section-header">
                <i class="bi bi-clock-history"></i>
                <span>搜索历史</span>
                <button @click="clearSearchHistory" class="clear-history-btn">
                  清除
                </button>
              </div>
              <div class="search-tags">
                <span
                  v-for="(item, index) in searchHistory"
                  :key="index"
                  class="search-tag"
                  @click="useSearchHistory(item)"
                >
                  {{ item }}
                  <button @click.stop="removeSearchHistory(index)" class="tag-remove">
                    <i class="bi bi-x"></i>
                  </button>
                </span>
              </div>
            </div>

            <!-- 搜索结果 -->
            <div v-else-if="!loading && searchResults.length > 0" class="search-section">
              <div class="results-list">
                <div
                  v-for="(result, index) in searchResults"
                  :key="result.id || result.key"
                  class="result-item"
                  :class="{ selected: selectedIndex === index }"
                  @click="navigateToResult(result)"
                >
                  <div class="result-icon-bar" :class="`result-icon-bar-${result.type}`">
                    <i :class="getResultIcon(result.type)"></i>
                  </div>
                  <div class="result-content">
                    <div class="result-header">
                      <span class="result-title" v-html="getResultTitle(result)"></span>
                      <span class="result-type" :class="`result-type-${result.type}`">
                        {{ getResultTypeName(result.type) }}
                      </span>
                    </div>
                    <p v-if="result.type === 'users' && result.description" class="result-desc" v-html="result.description"></p>
                    <p v-else-if="result.type === 'links' && result.description" class="result-desc" v-html="result.description"></p>
                    <p v-else-if="result.type === 'moments' && result.content" class="result-desc" v-html="truncateWithHighlight(result.content, 80)"></p>
                    <p v-else-if="result.abstract" class="result-desc" v-html="truncateWithHighlight(result.abstract, 100)"></p>
                  </div>
                  <div class="result-arrow">
                    <i class="bi bi-chevron-right"></i>
                  </div>
                </div>
              </div>
            </div>

            <!-- 无结果 -->
            <div v-else-if="!loading && searchQuery && searchResults.length === 0" class="search-empty">
              <div class="empty-icon">
                <i class="bi bi-search"></i>
              </div>
              <p class="empty-title">没有找到相关结果</p>
              <p class="empty-desc">试试这些热门搜索：</p>
              <div class="empty-suggestions">
                <button
                  v-for="(item, index) in hotSearches.slice(0, 3)"
                  :key="index"
                  class="suggest-btn"
                  @click="useSearchHistory(item)"
                >
                  {{ item }}
                </button>
              </div>
            </div>

            <!-- 加载中 -->
            <div v-else-if="loading" class="search-loading">
              <div class="loading-spinner"></div>
              <p>搜索中...</p>
            </div>
          </div>

          <!-- 底部快捷键提示 -->
          <div class="search-footer">
            <div class="shortcut-hints">
              <span class="hint"><kbd>Enter</kbd> 搜索/确认</span>
              <span class="hint"><kbd>↑↓</kbd> 选择</span>
              <span class="hint"><kbd>Esc</kbd> 关闭</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup>
import { ref, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@/utils/network'
import { toast } from '@/utils/app'
import { cache } from '@/utils/network'
import { STORAGE_KEYS } from '@/constants'

const router = useRouter()

const scopes = [
  { key: 'all', label: '全部', icon: 'bi bi-search' },
  { key: 'article', label: '文章', icon: 'bi bi-file-earmark-text' },
  { key: 'page', label: '页面', icon: 'bi bi-file-earmark' },
  { key: 'tag', label: '标签', icon: 'bi bi-tag' },
  { key: 'links', label: '友链', icon: 'bi bi-link' },
  { key: 'users', label: '用户', icon: 'bi bi-person' },
  { key: 'moments', label: '动态', icon: 'bi bi-chat-dots' },
]

const hotSearches = [
  '随记', '旅行', '博客', 'vue', 'Node.js',
  '前端开发', '后端开发', '算法', '数据库', '架构'
]

const visible = ref(false)
const searchQuery = ref('')
const loading = ref(false)
const searchResults = ref([])
const searchHistory = ref([])
const searchScope = ref('all')
const selectedIndex = ref(-1)
const searchInputRef = ref(null)
const debounceTimer = ref(null)
const globalKeyHandler = ref(null)

const show = () => {
  visible.value = true
  searchQuery.value = ''
  searchResults.value = []
  selectedIndex.value = -1
  searchScope.value = 'all'
  loadSearchHistory()
  document.body.style.overflow = 'hidden'
  
  globalKeyHandler.value = (e) => {
    if (!visible.value) return
    
    const target = e.target
    const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA'
    
    if (e.key === 'Escape') {
      e.preventDefault()
      hide()
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      handleKeyUp()
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      handleKeyDown()
    } else if (e.key === 'Enter' && !isInput) {
      e.preventDefault()
      handleEnterKey()
    }
  }
  
  window.addEventListener('keydown', globalKeyHandler.value)
  
  nextTick(() => {
    if (searchInputRef.value) {
      searchInputRef.value.focus()
    }
  })
}

const hide = () => {
  visible.value = false
  searchQuery.value = ''
  searchResults.value = []
  selectedIndex.value = -1
  clearTimeout(debounceTimer.value)
  document.body.style.overflow = ''
  
  if (globalKeyHandler.value) {
    window.removeEventListener('keydown', globalKeyHandler.value)
    globalKeyHandler.value = null
  }
}

const clearQuery = () => {
  searchQuery.value = ''
  searchResults.value = []
  selectedIndex.value = -1
  nextTick(() => {
    if (searchInputRef.value) {
      searchInputRef.value.focus()
    }
  })
}

const loadSearchHistory = () => {
  const history = localStorage.getItem(STORAGE_KEYS.SEARCH_HISTORY)
  if (history) {
    try {
      searchHistory.value = JSON.parse(history)
    } catch {
      searchHistory.value = []
    }
  }
}

const saveSearchHistory = (query) => {
  if (!query) return
  searchHistory.value = searchHistory.value.filter(item => item !== query)
  searchHistory.value.unshift(query)
  if (searchHistory.value.length > 10) {
    searchHistory.value = searchHistory.value.slice(0, 10)
  }
  localStorage.setItem(STORAGE_KEYS.SEARCH_HISTORY, JSON.stringify(searchHistory.value))
}

const clearSearchHistory = () => {
  searchHistory.value = []
  localStorage.removeItem(STORAGE_KEYS.SEARCH_HISTORY)
}

const removeSearchHistory = (index) => {
  searchHistory.value.splice(index, 1)
  localStorage.setItem(STORAGE_KEYS.SEARCH_HISTORY, JSON.stringify(searchHistory.value))
}

const useSearchHistory = (query) => {
  searchQuery.value = query
  selectedIndex.value = -1
  performSearch()
}

const changeSearchScope = (scope) => {
  searchScope.value = scope
  selectedIndex.value = -1
  if (searchQuery.value.trim()) {
    performSearch()
  }
}

const getSearchPlaceholder = () => {
  const map = {
    article: '搜索文章...',
    page: '搜索页面...',
    tag: '搜索标签...',
    links: '搜索友链...',
    users: '搜索用户...',
    moments: '搜索动态...',
  }
  return map[searchScope.value] || '搜索文章、页面、标签、友链、用户或动态...'
}

const handleInput = () => {
  clearTimeout(debounceTimer.value)
  debounceTimer.value = setTimeout(() => {
    if (searchQuery.value.trim()) {
      performSearch()
    } else {
      searchResults.value = []
      selectedIndex.value = -1
    }
  }, 300)
}

const handleKeyUp = () => {
  if (searchResults.value.length > 0) {
    if (selectedIndex.value > 0) {
      selectedIndex.value--
    } else if (selectedIndex.value === -1) {
      selectedIndex.value = searchResults.value.length - 1
    }
  }
}

const handleKeyDown = () => {
  if (searchResults.value.length > 0) {
    if (selectedIndex.value < searchResults.value.length - 1) {
      selectedIndex.value++
    } else {
      selectedIndex.value = -1
    }
  }
}

const handleEnterKey = () => {
  if (selectedIndex.value >= 0 && selectedIndex.value < searchResults.value.length) {
    navigateToResult(searchResults.value[selectedIndex.value])
  } else if (searchQuery.value.trim()) {
    performSearch()
  }
}

const performSearch = async () => {
  const query = searchQuery.value.trim()
  if (!query) return

  loading.value = true
  searchResults.value = []
  selectedIndex.value = -1

  try {
    const cacheKey = `search_results_${searchScope.value}_${query}`
    const cacheExpire = 5

    const cachedResults = cache.get(cacheKey)
    if (cachedResults) {
      searchResults.value = cachedResults
      saveSearchHistory(query)
      loading.value = false
      return
    }

    let results = []

    switch (searchScope.value) {
      case 'article':
        results = await searchArticles(query)
        break
      case 'page':
        results = await searchPages(query)
        break
      case 'tag':
        results = await searchTags(query)
        break
      case 'links':
        results = await searchLinks(query)
        break
      case 'users':
        results = await searchUsers(query)
        break
      case 'moments':
        results = await searchMoments(query)
        break
      default: {
        const [articleResults, pageResults, tagResults, linksResults, usersResults, momentsResults] = await Promise.all([
          searchArticles(query),
          searchPages(query),
          searchTags(query),
          searchLinks(query),
          searchUsers(query),
          searchMoments(query)
        ])

        const optimizedResults = []
        const seenIds = new Set()

        if (articleResults.length > 0) {
          articleResults.sort((a, b) => new Date(b.create_time || 0) - new Date(a.create_time || 0))
          articleResults.forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }
        if (pageResults.length > 0) {
          pageResults.sort((a, b) => new Date(b.create_time || 0) - new Date(a.create_time || 0))
          pageResults.forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }
        if (usersResults.length > 0) {
          usersResults.forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }
        if (linksResults.length > 0) {
          linksResults.forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }
        if (tagResults.length > 0) {
          tagResults.slice(0, 3).forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }
        if (momentsResults.length > 0) {
          momentsResults.forEach(item => {
            if (!seenIds.has(item.id)) { seenIds.add(item.id); optimizedResults.push(item) }
          })
        }

        results = optimizedResults
        break
      }
    }

    searchResults.value = results
    cache.set(cacheKey, results, cacheExpire)
    saveSearchHistory(query)
  } catch (error) {
    console.error('搜索失败:', error)
    toast.error('搜索失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

const searchArticles = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/article?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'article' }))
    }
    return []
  } catch { return [] }
}

const searchPages = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/pages?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'page' }))
    }
    return []
  } catch { return [] }
}

const searchTags = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/tags?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'tag' }))
    }
    return []
  } catch { return [] }
}

const searchLinks = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/links?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'links' }))
    }
    return []
  } catch { return [] }
}

const searchUsers = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/users?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'users' }))
    }
    return []
  } catch { return [] }
}

const searchMoments = async (keyword) => {
  try {
    const { code, data } = await request.get(`/api/search/moments?keyword=${encodeURIComponent(keyword)}&page=1&limit=50`)
    if (code === 200 && data) {
      const arr = data.data?.data || data.data
      if (Array.isArray(arr)) return arr.map(item => ({ ...item, type: 'moments' }))
    }
    return []
  } catch { return [] }
}

const getResultIcon = (type) => {
  const map = {
    article: 'bi bi-file-earmark-text',
    page: 'bi bi-file-earmark',
    tag: 'bi bi-tag',
    links: 'bi bi-link',
    users: 'bi bi-person',
    moments: 'bi bi-chat-dots',
  }
  return map[type] || 'bi bi-search'
}

const getResultTypeName = (type) => {
  const map = { article: '文章', page: '页面', tag: '标签', links: '友链', users: '用户', moments: '动态' }
  return map[type] || '未知'
}

const getResultTitle = (result) => {
  if (result.type === 'users' || result.type === 'links') {
    return result.nickname || result.name || result.title || '未知'
  }
  if (result.type === 'moments') {
    return truncateWithHighlight(result.content, 20) || '动态'
  }
  return result.title || result.name || '未知'
}

// 处理带<mark>标签的文本截断，保留高亮标签
const truncateWithHighlight = (text, maxLength = 100) => {
  if (!text) return ''
  
  // 移除标签计算纯文本长度
  const plainText = text.replace(/<[^>]*>/g, '')
  
  if (plainText.length <= maxLength) {
    return text
  }
  
  // 计算需要保留的字符数（包含标签）
  let charCount = 0
  let result = ''
  let tagOpen = 0
  let i = 0
  
  while (i < text.length && charCount < maxLength) {
    if (text[i] === '<') {
      // 找到标签结束位置
      const tagEnd = text.indexOf('>', i)
      if (tagEnd !== -1) {
        result += text.substring(i, tagEnd + 1)
        // 统计未闭合标签
        const tag = text.substring(i, tagEnd + 1)
        if (!tag.startsWith('</')) {
          tagOpen++
        } else {
          tagOpen--
        }
        i = tagEnd + 1
        continue
      }
    }
    
    result += text[i]
    charCount++
    i++
  }
  
  // 闭合未闭合的标签
  for (let j = 0; j < tagOpen; j++) {
    result += '</mark>'
  }
  
  result += '...'
  return result
}

const navigateToResult = (result) => {
  hide()
  switch (result.type) {
    case 'article':
      if (result.id) router.push(`/archives/${result.id}`)
      break
    case 'page':
      if (result.key) router.push(`/${result.key}`)
      else toast.error('页面路径无效')
      break
    case 'tag':
      if (result.id) router.push(`/tag/${result.id}`)
      break
    case 'links':
      if (result.url) window.open(result.url, '_blank')
      else toast.error('友链链接无效')
      break
    case 'users':
      if (result.id) router.push(`/author/${result.id}`)
      break
    case 'moments':
      if (result.id) router.push(`/moments/${result.id}`)
      break
  }
}

onUnmounted(() => {
  clearTimeout(debounceTimer.value)
  document.body.style.overflow = ''
  
  if (globalKeyHandler.value) {
    window.removeEventListener('keydown', globalKeyHandler.value)
    globalKeyHandler.value = null
  }
})

defineExpose({ show, hide })
</script>

<style scoped>
.search-modal-enter-active,
.search-modal-leave-active {
  transition: opacity 0.25s ease;
}

.search-modal-enter-from,
.search-modal-leave-to {
  opacity: 0;
}

.search-modal-enter-active .search-modal-container,
.search-modal-leave-active .search-modal-container {
  transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.25s ease;
}

.search-modal-enter-from .search-modal-container,
.search-modal-leave-to .search-modal-container {
  transform: translateY(-20px) scale(0.98);
  opacity: 0;
}

.search-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  z-index: 1060;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 10vh;
  padding-left: 1rem;
  padding-right: 1rem;
}

.search-modal-container {
  width: 100%;
  max-width: 560px;
  max-height: 75vh;
  overflow: hidden;
}

.search-modal-content {
  background: var(--bs-body-bg);
  border-radius: var(--wx-radius-lg);
  box-shadow: var(--wx-shadow-lg);
  overflow: hidden;
  border: 1px solid var(--bs-border-color);
  display: flex;
  flex-direction: column;
  max-height: 75vh;
}

/* 头部区域 */
.search-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  background: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
}

.search-modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--bs-body-color);
  margin: 0;
}

.search-modal-close {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--bs-secondary-color);
  font-size: 1rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--wx-transition);
}

.search-modal-close:hover {
  background: var(--bs-border-color);
  color: var(--bs-body-color);
  transform: rotate(90deg);
}

.search-input-wrapper {
  padding: 1rem 1.5rem;
  position: relative;
}

.search-input-group {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 1rem;
  color: var(--bs-secondary-color);
  font-size: 1rem;
  pointer-events: none;
  transition: color var(--wx-transition);
}

.search-input-group:focus-within .search-icon {
  color: var(--bs-primary);
}

.search-input {
  width: 100%;
  padding: 0.75rem 2.5rem 0.75rem 2.75rem;
  font-size: 0.95rem;
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  background: var(--bs-tertiary-bg);
  color: var(--bs-body-color);
  outline: none;
  transition: var(--wx-transition);
}

.search-input:focus {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.15);
  background: var(--bs-body-bg);
}

.search-input::placeholder {
  color: var(--bs-secondary-color);
}

.search-clear-btn {
  position: absolute;
  right: 0.75rem;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: var(--bs-border-color);
  color: var(--bs-body-color);
  font-size: 0.75rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--wx-transition);
}

.search-clear-btn:hover {
  background: var(--bs-secondary-color);
  color: #fff;
  transform: rotate(90deg);
}

.search-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  margin-top: 0.75rem;
}

.search-scope-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
  font-weight: 500;
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  background: transparent;
  color: var(--bs-secondary-color);
  cursor: pointer;
  transition: var(--wx-transition);
}

.search-scope-btn:hover {
  border-color: var(--bs-primary);
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.05);
  transform: translateY(-1px);
}

.search-scope-btn.active {
  border-color: transparent;
  color: #fff;
  background: var(--wx-gradient-primary);
  box-shadow: 0 2px 8px rgba(var(--bs-primary-rgb), 0.25);
}

.search-scope-btn i {
  font-size: 0.625rem;
}

.search-content {
  flex-grow: 1;
  overflow-y: auto;
  padding: 0 1.5rem;
  max-height: 400px;
}

.search-section {
  padding: 0.5rem 0 1rem;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin-bottom: 0.625rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--bs-secondary-color);
}

.section-header i {
  font-size: 0.75rem;
}

.clear-history-btn {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.125rem 0.5rem;
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
}

.clear-history-btn:hover {
  color: var(--bs-danger);
  background: rgba(var(--bs-danger-rgb), 0.1);
}

.search-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.search-tag {
  display: inline-flex;
  align-items: center;
  padding: 0.375rem 0.875rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--bs-body-color);
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
  position: relative;
}

.search-tag:hover {
  background: var(--wx-gradient-primary);
  color: #fff;
  border-color: transparent;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(var(--bs-primary-rgb), 0.25);
}

.tag-remove {
  margin-left: 0.375rem;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: none;
  background: rgba(0, 0, 0, 0.1);
  color: var(--bs-secondary-color);
  font-size: 0.5625rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--wx-transition);
  flex-shrink: 0;
}

.search-tag:hover .tag-remove {
  background: rgba(0, 0, 0, 0.25);
  color: #fff;
}

.tag-remove:hover {
  background: rgba(0, 0, 0, 0.4);
}

.results-list {
  padding: 0.25rem 0;
}

.result-item {
  display: flex;
  align-items: center;
  padding: 0.75rem 1rem;
  border-radius: var(--wx-radius-md);
  border: 1px solid transparent;
  cursor: pointer;
  transition: var(--wx-transition);
  margin-bottom: 0.25rem;
}

.result-item:hover {
  background: var(--bs-tertiary-bg);
  border-color: rgba(var(--bs-primary-rgb), 0.3);
  transform: translateX(4px);
}

.result-item.selected {
  background: rgba(var(--bs-primary-rgb), 0.1);
  border-color: rgba(var(--bs-primary-rgb), 0.3);
}

.result-icon-bar {
  width: 40px;
  height: 40px;
  border-radius: var(--wx-radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-right: 0.875rem;
  font-size: 1.125rem;
  color: #fff;
}

.result-icon-bar-article { background: linear-gradient(135deg, #3b82f6, #60a5fa); }
.result-icon-bar-page { background: linear-gradient(135deg, #8b5cf6, #a78bfa); }
.result-icon-bar-tag { background: linear-gradient(135deg, #10b981, #34d399); }
.result-icon-bar-links { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
.result-icon-bar-users { background: linear-gradient(135deg, #ec4899, #f472b6); }
.result-icon-bar-moments { background: linear-gradient(135deg, #06b6d4, #22d3ee); }

.result-content {
  flex-grow: 1;
  min-width: 0;
}

.result-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.25rem;
}

.result-title {
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--bs-body-color);
  line-height: 1.4;
}

.result-type {
  font-size: 0.6875rem;
  padding: 0.125rem 0.5rem;
  border-radius: var(--wx-radius-sm);
  font-weight: 500;
  color: #fff;
}

.result-type-article { background: #3b82f6; }
.result-type-page { background: #8b5cf6; }
.result-type-tag { background: #10b981; }
.result-type-links { background: #f59e0b; }
.result-type-users { background: #ec4899; }
.result-type-moments { background: #06b6d4; }

.result-desc {
  font-size: 0.8125rem;
  color: var(--bs-secondary-color);
  line-height: 1.5;
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.result-arrow {
  color: var(--bs-secondary-color);
  font-size: 0.875rem;
  margin-left: 0.5rem;
  opacity: 0;
  transition: var(--wx-transition);
}

.result-item:hover .result-arrow,
.result-item.selected .result-arrow {
  opacity: 1;
  color: var(--bs-primary);
  transform: translateX(4px);
}

.highlight,
.result-content mark {
  background-color: rgba(250, 204, 21, 0.4);
  color: var(--bs-body-color);
  padding: 1px 3px;
  border-radius: var(--wx-radius-sm);
}

.search-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2.5rem 1.25rem;
}

.empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--bs-tertiary-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--bs-secondary-color);
  font-size: 2rem;
  margin-bottom: 1rem;
}

.empty-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--bs-body-color);
  margin: 0 0 0.375rem;
}

.empty-desc {
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
  margin: 0 0 1rem;
}

.empty-suggestions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  justify-content: center;
}

.suggest-btn {
  padding: 0.375rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--bs-secondary-color);
  background: transparent;
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
}

.suggest-btn:hover {
  border-color: var(--bs-primary);
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.05);
  transform: translateY(-2px);
  box-shadow: var(--wx-shadow-sm);
}

.search-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2.5rem 1.25rem;
}

.loading-spinner {
  width: 36px;
  height: 36px;
  border: 3px solid var(--bs-border-color);
  border-top-color: var(--bs-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 0.75rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.search-loading p {
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
  margin: 0;
}

.search-footer {
  padding: 0.75rem 1.5rem 1rem;
  background: var(--bs-tertiary-bg);
  border-top: 1px solid var(--bs-border-color);
}

.shortcut-hints {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
}

.hint {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

kbd {
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  padding: 2px 6px;
  font-size: 0.6875rem;
  font-family: inherit;
  font-weight: 500;
  color: var(--bs-body-color);
  box-shadow: var(--wx-shadow-sm);
}

@media (max-width: 575.98px) {
  .search-modal-overlay {
    padding-top: 5vh;
  }

  .search-modal-content {
    border-radius: var(--wx-radius-md);
  }

  .search-modal-header {
    padding: 0.75rem 1rem;
  }

  .search-modal-title {
    font-size: 1rem;
  }

  .search-input-wrapper {
    padding: 0.75rem 1rem;
  }

  .search-content {
    padding: 0 1rem;
  }

  .result-item {
    padding: 0.625rem 0.75rem;
  }

  .result-icon-bar {
    width: 36px;
    height: 36px;
    margin-right: 0.75rem;
    font-size: 1rem;
  }

  .result-title {
    font-size: 0.875rem;
  }

  .search-scope-btn span {
    display: none;
  }

  .search-scope-btn {
    padding: 0.375rem 0.625rem;
  }

  .shortcut-hints {
    gap: 0.625rem;
  }
}

[data-bs-theme=dark] {
  .highlight,
  .result-content mark {
    background-color: rgba(250, 204, 21, 0.25);
  }
}
</style>
