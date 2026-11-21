<template>
  <div class="mention-picker" :style="panelStyle" role="listbox" :aria-label="title">
    <p v-if="isLoading" class="mention-picker-tip">加载中…</p>
    <p v-else-if="!list.length" class="mention-picker-tip">
      {{ keyword ? keywordEmptyText : emptyText }}
    </p>
    <template v-else>
      <button
        v-for="(item, index) in list"
        :key="item.id"
        type="button"
        class="mention-item"
        :class="{ 'is-active': index === active }"
        role="option"
        :aria-selected="index === active"
        @mousedown.prevent="select(item)"
        @mouseenter="active = index"
      >
        <img v-if="item.avatar" class="mention-item-avatar" :src="fullUrl(item.avatar)" :alt="nameOf(item)" loading="lazy" />
        <span v-else class="mention-item-avatar mention-item-avatar-text">{{ nameOf(item).slice(0, 1) }}</span>
        <span class="mention-item-name">{{ nameOf(item) }}</span>
      </button>
    </template>
  </div>
</template>

<script setup>
/**
 * 候选选择浮层（@ 用户 / # 话题共用）
 *
 * 默认数据源是**当前用户已关注的人**（与后端 app/model/mention.go 的口径一致：
 * 未关注的人后端也不会解析成提及）；`#` 话题联想时由父级传 items 进来复用本组件
 * （见 EmojiEditor.vue），两者的交互与外观完全一致。
 *
 * 用法：父级（编辑器）自己监听光标前的「@关键词 / #关键词」，把关键词与浮层坐标传进来；
 * 键盘上下键 / 回车由父级转发给自己的 move() / selectActive()（见 EmojiEditor.vue），
 * 因为焦点始终在 contenteditable 里，事件不会落到本组件上。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { getMentionCandidates } from '@/api/mention'
import { getFullUrl } from '@/utils/runtimeConfig'

const props = defineProps({
  // 光标前已输入的关键词（@ / # 之后的部分）
  keyword: { type: String, default: '' },
  // 候选项：传了就直接用（话题联想），不传则拉取「我关注的人」
  items: { type: Array, default: null },
  // 名称字段：用户是 nickname，话题是 name
  nameKey: { type: String, default: 'nickname' },
  // 无障碍标题与两种空态文案
  title: { type: String, default: '选择要 @ 的用户' },
  emptyText: { type: String, default: '你还没有关注任何人' },
  keywordEmptyText: { type: String, default: '没有匹配的已关注用户' },
  // 外部数据源是否仍在加载（话题候选由父级懒加载，见 EmojiEditor.vue）
  loading: { type: Boolean, default: false },
  // 浮层坐标（**视口坐标**，配合 position: fixed）：
  //   left   面板左边缘
  //   top    向下展开时的上边缘
  //   flipTo 向上展开时的下边缘（下方空间不足时由调用方给出，null 表示向下展开）
  //
  // 用视口坐标而不是"相对某个容器"的坐标：后者要扣除内边距与祖先偏移，
  // 一旦容器层级变化（比如编辑器外面多包一层）就会整体错位。
  position: { type: Object, default: () => ({ left: 0, top: 0, flipTo: null }) }
})

// 面板宽度 200 + 右侧留白；最大高度 236 + 间距（与样式保持一致）
const PANEL_WIDTH = 208
const PANEL_HEIGHT = 246

// 视口坐标 → fixed 定位样式：左右贴边收拢、下方空间不足时向上翻开
const panelStyle = computed(() => {
  const left = Math.max(8, Math.min(props.position.left || 0, Math.max(8, window.innerWidth - PANEL_WIDTH)))

  if (props.position.flipTo != null) {
    const bottom = Math.max(8, window.innerHeight - props.position.flipTo)
    return { left: `${left}px`, top: 'auto', bottom: `${bottom}px` }
  }

  const top = Math.max(8, Math.min(props.position.top || 0, Math.max(8, window.innerHeight - PANEL_HEIGHT)))
  return { left: `${left}px`, top: `${top}px`, bottom: 'auto' }
})

const emit = defineEmits(['select'])

const MAX_SHOW = 8

const all = ref([])
const loading = ref(true)
const active = ref(0)

const fullUrl = getFullUrl

function nameOf(item) {
  return String(item?.[props.nameKey] ?? '')
}

const isLoading = computed(() => props.loading || loading.value)

const list = computed(() => {
  const keyword = props.keyword.trim().toLowerCase()
  const matched = keyword
    ? all.value.filter((item) => nameOf(item).toLowerCase().includes(keyword))
    : all.value
  return matched.slice(0, MAX_SHOW)
})

// 关键词变化后回到第一项，避免上下键从上次的位置继续
watch(
  () => props.keyword,
  () => {
    active.value = 0
  }
)

watch(list, () => {
  if (active.value >= list.value.length) active.value = 0
})

// 话题候选是懒加载：面板可能先挂载、数据后到，这里跟着更新（否则第一次打开是空面板）
watch(
  () => props.items,
  (value) => {
    if (Array.isArray(value)) all.value = value
  }
)

onMounted(async () => {
  // 话题联想：父级已经取回候选（比组件自己去请求更可控），这里直接用
  if (Array.isArray(props.items)) {
    all.value = props.items
    loading.value = false
    return
  }

  all.value = await getMentionCandidates()
  loading.value = false
})

/** 上下键移动高亮项（父级转发） */
function move(delta) {
  if (!list.value.length) return
  const total = list.value.length
  active.value = (active.value + delta + total) % total
}

/** 回车选中当前高亮项；返回是否消费了这次回车 */
function selectActive() {
  const item = list.value[active.value]
  if (!item) return false
  select(item)
  return true
}

function select(item) {
  emit('select', item)
}

defineExpose({ move, selectActive })
</script>

<style scoped>
.mention-picker {
  /* fixed + 视口坐标：直接贴在光标处，不受编辑器容器内边距 / 祖先偏移影响，
     也不会被弹窗的 overflow: hidden 裁掉（高于弹窗层级 2900，低于确认弹窗 3000） */
  position: fixed;
  z-index: 2950;
  width: 200px;
  max-height: 236px;
  overflow-y: auto;
  padding: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.mention-picker-tip {
  margin: 0;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.mention-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
  text-align: left;
  cursor: pointer;
}
.mention-item.is-active {
  background: var(--bg-muted);
}
.mention-item-avatar {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.mention-item-avatar-text {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--text-muted);
}
.mention-item-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
