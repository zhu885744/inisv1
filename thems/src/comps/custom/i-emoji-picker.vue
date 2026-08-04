<template>
  <div v-if="showPicker" class="emoji-picker-panel" :class="{ 'bg-dark border-secondary': isDarkMode }">
    <!-- 表情分类导航 -->
    <ul class="nav nav-pills px-2 pt-2 pb-1" role="tablist">
      <li class="nav-item" v-for="cat in allCategories" :key="cat.key">
        <button
          class="nav-link rounded-3 px-3 py-1 mb-1"
          :class="activeCategory === cat.key ? 'active bg-primary text-white' : 'text-secondary'"
          @click="activeCategory = cat.key"
          type="button"
          role="tab"
        >
          {{ cat.label }}
        </button>
      </li>
    </ul>
    <!-- 表情网格 -->
    <div class="emoji-grid wx-scrollable px-2 pb-2">
      <!-- 本地颜文字 -->
      <template v-if="currentCategoryType === 'local'">
        <button
          v-for="(emoji, index) in currentEmojis"
          :key="index"
          @click="selectEmoji(emoji.icon)"
          class="emoji-btn"
          :title="emoji.text"
          type="button"
        >
          {{ emoji.icon }}
        </button>
      </template>
      <!-- API表情图片 -->
      <template v-else>
        <button
          v-for="(emoji, index) in currentEmojis"
          :key="index"
          @click="selectApiEmoji(emoji)"
          class="emoji-btn emoji-img-btn"
          :title="emoji.name"
          type="button"
        >
          <img :src="getFullUrl(emoji.url)" :alt="emoji.name" loading="lazy" class="emoji-img">
        </button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import owoJson from '@/assets/json/OwO.json'
import { request } from '@/utils/network'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  isDarkMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'select'])

const showPicker = ref(props.modelValue)
const apiCategories = ref([]) // 后端API表情分类

// 构建所有分类列表（本地 + API）
const allCategories = computed(() => {
  const local = Object.keys(owoJson).map(key => ({
    key: `local_${key}`,
    label: key,
    type: 'local'
  }))
  const api = apiCategories.value.map(cat => ({
    key: `api_${cat.name}`,
    label: cat.name,
    type: 'api'
  }))
  return [...local, ...api]
})

const activeCategory = ref('')

// 当前分类类型
const currentCategoryType = computed(() => {
  const cat = allCategories.value.find(c => c.key === activeCategory.value)
  return cat?.type || 'local'
})

// 当前分类的表情列表
const currentEmojis = computed(() => {
  if (!activeCategory.value) return []
  if (currentCategoryType.value === 'local') {
    const localKey = activeCategory.value.replace('local_', '')
    return owoJson[localKey]?.container || []
  } else {
    const apiName = activeCategory.value.replace('api_', '')
    const cat = apiCategories.value.find(c => c.name === apiName)
    return cat?.items || []
  }
})

// 获取完整URL
const getFullUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http')) return url
  const base = request.getBaseURL ? request.getBaseURL() : ''
  return base + url
}

// 选择本地表情
const selectEmoji = (emoji) => {
  emit('select', emoji)
  showPicker.value = false
  emit('update:modelValue', false)
}

// 选择API表情图片
const selectApiEmoji = (emoji) => {
  const fullUrl = getFullUrl(emoji.url)
  // 存储格式: [emoji:完整URL]
  emit('select', `[emoji:${fullUrl}]`)
  showPicker.value = false
  emit('update:modelValue', false)
}

// 加载后端API表情
const loadApiEmojis = async () => {
  try {
    const res = await request.get('/api/attachment/emoji')
    if (res.code === 200 && res.data?.categories) {
      apiCategories.value = res.data.categories
    }
  } catch (err) {
    console.error('加载表情API失败:', err)
  }
}

const handleClickOutside = (event) => {
  const emojiPickers = event.target.closest('.emoji-picker-panel')
  const emojiButtons = event.target.closest('[title="表情"], .emoji-button')
  if (!emojiPickers && !emojiButtons) {
    showPicker.value = false
    emit('update:modelValue', false)
  }
}

watch(() => props.modelValue, (newVal) => {
  showPicker.value = newVal
})

watch(showPicker, (newVal) => {
  if (newVal !== props.modelValue) {
    emit('update:modelValue', newVal)
  }
})

onMounted(() => {
  // 默认选中第一个本地分类
  const firstLocalKey = Object.keys(owoJson)[0]
  if (firstLocalKey) {
    activeCategory.value = `local_${firstLocalKey}`
  }
  // 加载API表情
  loadApiEmojis()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.emoji-picker-panel {
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  box-shadow: var(--wx-shadow-md);
  animation: emojiFadeIn var(--wx-transition);
  max-height: 280px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

@keyframes emojiFadeIn {
  from {
    opacity: 0;
    transform: translateY(-5px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.emoji-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  overflow-y: auto;
  padding: 6px;
  max-height: 220px;
}

.emoji-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  min-height: 32px;
  padding: 4px 8px;
  font-size: 14px;
  font-family: 'Segoe UI Emoji', 'Noto Color Emoji', sans-serif;
  border: none;
  background: var(--bs-tertiary-bg);
  cursor: pointer;
  transition: var(--wx-transition);
  border-radius: var(--wx-radius-sm);
  line-height: 1.2;
  color: inherit;
  flex-shrink: 0;
}

.emoji-btn:hover {
  background: rgba(var(--bs-primary-rgb), 0.15);
  transform: scale(1.08);
  box-shadow: var(--wx-shadow-sm);
}

.emoji-btn:active {
  transform: scale(0.95);
}

/* API表情图片按钮 */
.emoji-img-btn {
  padding: 2px;
}

.emoji-img {
  width: 28px;
  height: 28px;
  object-fit: contain;
  pointer-events: none;
}

:deep(.nav-pills .nav-link) {
  transition: var(--wx-transition);
  font-size: 0.85rem;
  font-weight: 500;
  border-radius: var(--wx-radius-sm);
}

:deep(.nav-pills .nav-link:hover:not(.active)) {
  background: rgba(var(--bs-primary-rgb), 0.1);
  color: var(--bs-primary);
}

:deep(.nav-pills .nav-link.active) {
  box-shadow: 0 2px 6px rgba(var(--bs-primary-rgb), 0.3);
}

/* 深色模式 */
:deep(.bg-dark .emoji-btn) {
  background: rgba(255, 255, 255, 0.08);
}

:deep(.bg-dark .emoji-btn:hover) {
  background: rgba(var(--bs-primary-rgb), 0.25);
}

:deep(.bg-dark .nav-pills .nav-link:hover:not(.active)) {
  background: rgba(255, 255, 255, 0.1);
}

/* 移动端适配 */
@media (max-width: 768px) {
  .emoji-grid {
    gap: 4px;
    padding: 4px;
  }

  .emoji-btn {
    min-width: 28px;
    min-height: 28px;
    padding: 3px 6px;
    font-size: 12px;
  }

  .emoji-img {
    width: 24px;
    height: 24px;
  }
}
</style>
