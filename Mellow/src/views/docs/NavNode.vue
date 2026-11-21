<template>
  <div class="nav-node">
    <!--
      目录只作为分组标题，不做成链接：
      前台路由是 /docs/:项目/:目录/:页面，少一段的 /docs/:项目/:目录 没有对应页面，
      做成链接点了就是死链。下面的子文档已经缩进列出了。

      有子内容时标题可点击，用于展开 / 收起整组。
    -->
    <button
      v-if="hasChildren"
      type="button"
      class="nav-group is-toggle"
      :class="{ 'is-active': isGroupActive, 'is-collapsed': !expanded }"
      :aria-expanded="expanded ? 'true' : 'false'"
      @click="toggle"
    >
      <i class="bi" :class="node.icon || 'bi-folder2'" aria-hidden="true" />
      <span class="nav-text">{{ node.name }}</span>
      <i class="bi nav-caret" :class="expanded ? 'bi-chevron-down' : 'bi-chevron-right'" aria-hidden="true" />
    </button>

    <div v-else class="nav-group">
      <i class="bi" :class="node.icon || 'bi-folder2'" aria-hidden="true" />
      <span class="nav-text">{{ node.name }}</span>
    </div>

    <!-- v-show 而不是 v-if：收起再展开时子节点的折叠状态不会丢 -->
    <div v-if="hasChildren" v-show="expanded" class="nav-children">
      <NavNode
        v-for="child in node.children || []"
        :key="`g-${child.id}`"
        :node="child"
        :version="version"
      />

      <RouterLink
        v-for="doc in node.docs || []"
        :key="`d-${doc.id}`"
        :to="`/docs/${version}/${node.slug || '-'}/${doc.slug}`"
        class="nav-item nav-item-doc"
        :class="{ 'is-active': isDocActive(doc) }"
      >
        <i class="bi bi-file-earmark-text" aria-hidden="true" />
        <span class="nav-text">{{ doc.title }}</span>
      </RouterLink>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'

const props = defineProps({
  node: { type: Object, required: true },
  version: { type: String, default: 'latest' },
  /**
   * 是否默认展开
   *
   * 侧边栏整体策略是「默认全部收起，只有第一个目录默认展开」，
   * 由 Layout 的首项判断传进来；嵌套目录不传，一律收起。
   */
  defaultExpanded: { type: Boolean, default: false }
})

const route = useRoute()

/*
 * 折叠状态存在 localStorage：key = 「版本/目录」，值 = 是否展开。
 *
 * 存成对象而不是「已收起」的数组，是为了区分两种「没记录」的情况：
 *   - 用户从来没点过 → 用默认值（首个目录展开、其余收起）
 *   - 用户手动收起过 → 尊重用户的选择，不被默认值覆盖
 */
const STORAGE_KEY = 'mellow_docs_expanded_groups'

function readState() {
  try {
    const obj = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    return obj && typeof obj === 'object' && !Array.isArray(obj) ? obj : {}
  } catch {
    return {}
  }
}

function writeState(key, value) {
  try {
    const obj = readState()
    obj[key] = value
    localStorage.setItem(STORAGE_KEY, JSON.stringify(obj))
  } catch {
    /* 隐私模式下写不进去，忽略即可 */
  }
}

const collapseKey = computed(() => `${props.version}/${props.node.slug || 'g' + props.node.id}`)

const hasChildren = computed(() => {
  const children = props.node.children || []
  const docs = props.node.docs || []
  return children.length > 0 || docs.length > 0
})

// 当前文档所在目录（含它的祖先目录）
function hasActiveDescendant(node, groupSlug) {
  if (!groupSlug) return false
  if (node.slug === groupSlug) return true
  return (node.children || []).some((child) => hasActiveDescendant(child, groupSlug))
}

function initialExpanded() {
  const state = readState()
  if (collapseKey.value in state) return !!state[collapseKey.value]

  // 没记录过：当前文档所在的目录链自动展开，否则看默认值（只有首个目录是 true）
  if (hasActiveDescendant(props.node, String(route.params.group || ''))) return true
  return props.defaultExpanded
}

// 同步计算初值，避免先收起再展开的闪烁
const expanded = ref(initialExpanded())

function toggle() {
  expanded.value = !expanded.value
  writeState(collapseKey.value, expanded.value)
}

const isGroupActive = computed(() => route.params.group === props.node.slug)

// 跳到本目录下的文档时（搜索、外链、上一篇/下一篇）自动展开，否则看不到当前页在哪
watch(
  () => route.params.group,
  (groupSlug) => {
    if (hasActiveDescendant(props.node, String(groupSlug || ''))) {
      expanded.value = true
    }
  }
)

onMounted(() => {
  if (hasActiveDescendant(props.node, String(route.params.group || ''))) {
    expanded.value = true
  }
})

function isDocActive(doc) {
  return route.params.slug === doc.slug
}
</script>

<style scoped>
.nav-node {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.nav-group {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px 4px;
  font-size: 12px;
  color: var(--text-light);
  letter-spacing: 1px;
  text-align: left;
}
/* 有子内容的目录：整行可点，鼠标移上去给点反馈 */
.nav-group.is-toggle {
  padding: 7px 10px;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color 0.2s, background-color 0.2s;
}
.nav-group.is-toggle:hover {
  color: var(--text-soft);
  background: var(--bg-muted);
}
.nav-group.is-active {
  color: var(--primary);
}
.nav-group i {
  flex-shrink: 0;
  font-size: 13px;
}
.nav-group .nav-text {
  flex: 1;
  min-width: 0;
}
.nav-group .nav-caret {
  flex-shrink: 0;
  font-size: 10px;
  opacity: 0.75;
  transition: transform 0.2s ease;
}
.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 10px;
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--text-soft);
  text-decoration: none;
  border-radius: var(--radius-sm);
  transition: color 0.2s, background-color 0.2s;
}
.nav-item i {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--text-light);
  transition: color 0.2s;
}
.nav-item:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.nav-item:hover i {
  color: var(--primary);
}
.nav-item.is-active {
  color: var(--primary);
  font-weight: 600;
  background: var(--accent-wash);
}
.nav-item.is-active i {
  color: var(--primary);
}
.nav-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nav-children {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-left: 10px;
  padding-left: 8px;
  border-left: 1px solid var(--border-soft);
}
</style>
