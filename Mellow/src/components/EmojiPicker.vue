<template>
  <div v-if="show" ref="panelRef" class="emoji-picker-panel" :class="{ 'is-inline': inline, 'drop-up': dropUp }">
    <!-- 分类导航 -->
    <div class="emoji-cats">
      <button
        v-for="cat in allCategories"
        :key="cat.key"
        type="button"
        :class="['emoji-cat', { 'is-active': activeCategory === cat.key }]"
        @click="activeCategory = cat.key"
      >{{ cat.label }}</button>
    </div>

    <!-- 表情网格 -->
    <div class="emoji-grid">
      <!-- 本地颜文字 -->
      <template v-if="currentCategoryType === 'local'">
        <button
          v-for="(emoji, idx) in currentEmojis"
          :key="idx"
          type="button"
          class="emoji-item text-emoji"
          :title="emoji.text || emoji"
          @click="selectEmoji(emoji.icon || emoji)"
        >{{ emoji.icon || emoji }}</button>
      </template>
      <!-- API 表情图片 -->
      <template v-else>
        <button
          v-for="(emoji, idx) in currentEmojis"
          :key="idx"
          type="button"
          class="emoji-item img-emoji"
          :title="emoji.name"
          @click="selectApiEmoji(emoji)"
        >
          <img :src="getFullUrl(emoji.url)" :alt="emoji.name" loading="lazy" />
        </button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { fetchEmojiCategories, getFullUrl } from '@/utils/emoji'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // 内联模式：表情面板以文档流方式展开（撑大父容器），用于弹窗等需要自适应高度的场景
  inline: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'select'])

const show = ref(props.modelValue)
const apiCategories = ref([])
const panelRef = ref(null)
// 下方空间不足时向上展开（移动端编辑器常位于屏幕下半部，向下展开会超出视口）
const dropUp = ref(false)

// 展开后检测面板是否超出视口底部，超出则改为向上展开
async function checkDropDirection() {
  if (props.inline) return
  await nextTick()
  const el = panelRef.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  dropUp.value = rect.bottom > window.innerHeight - 8 && rect.top > rect.height
}

// 本地颜文字（精简内置，作为 API 表情的补充）
const LOCAL_EMOJIS = {
  '颜文字': {
    type: 'local',
    items: ['(＾▽＾)', '(≧∇≦)', '(◕‿◕)', '(￣▽￣)', '(°▽°)', 'ヾ(≧▽≦*)o', '(｡♥‿♥｡)', '(╯°□°）╯', '(╥_╥)', '(＾＾)b', '(￣ε￣)', '(¬‿¬)', '(・∀・)', '(´・ω・`)', '(✿◡‿◡)', '(⊙_⊙)', '(；一_一)', '(┬┬﹏┬┬)', '(￢_￢)', '(・_・;)']
  }
}

const allCategories = computed(() => {
  const local = Object.keys(LOCAL_EMOJIS).map((key) => ({
    key: `local_${key}`,
    label: key,
    type: 'local'
  }))
  const api = apiCategories.value.map((cat) => ({
    key: `api_${cat.name}`,
    label: cat.name,
    type: 'api'
  }))
  return [...local, ...api]
})

const activeCategory = ref('')

const currentCategoryType = computed(() => {
  const cat = allCategories.value.find((c) => c.key === activeCategory.value)
  return cat?.type || 'local'
})

const currentEmojis = computed(() => {
  if (!activeCategory.value) return []
  if (currentCategoryType.value === 'local') {
    const localKey = activeCategory.value.replace('local_', '')
    return LOCAL_EMOJIS[localKey]?.items || []
  } else {
    const apiName = activeCategory.value.replace('api_', '')
    const cat = apiCategories.value.find((c) => c.name === apiName)
    return cat?.items || []
  }
})

function selectEmoji(emoji) {
  emit('select', emoji)
  close()
}

function selectApiEmoji(emoji) {
  const fullUrl = getFullUrl(emoji.url)
  // 存储格式: [emoji:URL]
  emit('select', `[emoji:${fullUrl}]`)
  close()
}

function close() {
  show.value = false
  emit('update:modelValue', false)
}

async function loadApiEmojis() {
  try {
    apiCategories.value = await fetchEmojiCategories()
  } catch {
    apiCategories.value = []
  }
}

function handleClickOutside(event) {
  const picker = event.target.closest('.emoji-picker-panel')
  const emojiButton = event.target.closest('[title="插入表情"]')
  if (!picker && !emojiButton) {
    close()
  }
}

watch(
  () => props.modelValue,
  (val) => {
    show.value = val
    if (val) checkDropDirection()
  }
)

onMounted(() => {
  // 默认选中第一个本地分类
  const firstKey = Object.keys(LOCAL_EMOJIS)[0]
  if (firstKey) activeCategory.value = `local_${firstKey}`
  loadApiEmojis()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.emoji-picker-panel {
  position: absolute;
  top: 100%;
  left: 0;
  z-index: 100;
  min-width: 320px;
  max-width: calc(100vw - 24px);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  max-height: 280px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: emojiFadeIn 0.2s ease-out;
  margin-top: 4px;
}
@keyframes emojiFadeIn {
  from { opacity: 0; transform: translateY(-5px); }
  to { opacity: 1; transform: translateY(0); }
}
/* 下方空间不足时向上展开 */
.emoji-picker-panel.drop-up {
  top: auto;
  bottom: 100%;
  margin-top: 0;
  margin-bottom: 4px;
}
/* 内联模式：占据文档流，撑大父容器（用于弹窗自适应高度） */
.emoji-picker-panel.is-inline {
  position: static;
  width: 100%;
  min-width: 0;
  max-width: none;
  margin-top: 8px;
  box-shadow: none;
  border-color: var(--border-soft);
  /* 解除高度限制：面板与网格都不自己滚动，高度完全交给外层容器（弹窗 / 页面）撑开，
     这样表情能一次看全，而不是在面板内部出现滚动条 */
  max-height: none;
  overflow: visible;
}
.emoji-picker-panel.is-inline .emoji-grid {
  max-height: none;
  overflow-y: visible;
}
.emoji-cats {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  padding: 8px;
  border-bottom: 1px solid var(--border-soft);
  flex-shrink: 0;
}
.emoji-cat {
  padding: 5px 12px;
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  border: 1px solid transparent;
  color: var(--text-soft);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}
.emoji-cat.is-active {
  background: var(--primary);
  color: #fff;
}
.emoji-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 10px;
  overflow-y: auto;
  max-height: 210px;
}
.emoji-item {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  min-height: 32px;
  padding: 4px 8px;
  border: none;
  background: var(--bg-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
  flex-shrink: 0;
  line-height: 1.2;
}
.emoji-item:hover {
  background: var(--accent-soft);
  transform: scale(1.08);
}
.emoji-item:active {
  transform: scale(0.95);
}
.text-emoji {
  font-size: 14px;
  color: var(--text);
}
.img-emoji {
  padding: 2px;
}
.img-emoji img {
  width: 28px;
  height: 28px;
  object-fit: contain;
  pointer-events: none;
}

@media (max-width: 768px) {
  /* 面板宽度严格跟随编辑器容器，绝不超出容器/屏幕 */
  .emoji-picker-panel {
    left: 0;
    right: 0;
    width: 100%;
    min-width: 0;
    max-width: 100%;
    max-height: min(320px, 60vh);
  }
  /* 分类导航：单行横向滚动（不换行），避免折行吃掉表情网格空间 */
  .emoji-cats {
    flex-wrap: nowrap;
    overflow-x: auto;
    overflow-y: hidden;
    max-height: none;
    padding: 6px 8px;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }
  .emoji-cats::-webkit-scrollbar { display: none; }
  .emoji-cat {
    flex-shrink: 0;
    padding: 4px 10px;
  }
  .emoji-grid {
    flex: 1;
    max-height: none; /* 高度由面板整体约束 */
    gap: 4px;
    padding: 8px;
    justify-content: space-between;
  }
  .emoji-item {
    min-width: 34px;
    min-height: 34px;
    padding: 4px 6px;
    font-size: 13px;
  }
  .img-emoji img { width: 26px; height: 26px; }
}

@media (max-width: 480px) {
  /* 不再使用 calc(100vw - 24px)：面板相对编辑器容器定位，
     按视口宽度计算会在容器偏右时溢出屏幕 */
  .emoji-grid { gap: 3px; padding: 6px; }
  .emoji-item { min-width: 32px; min-height: 32px; font-size: 12px; }
}
</style>
