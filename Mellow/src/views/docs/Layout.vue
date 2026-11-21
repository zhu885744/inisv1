<template>
  <div class="docs-shell">
    <!-- 移动端遮罩 -->
    <div v-if="drawerOpen" class="docs-mask" @click="drawerOpen = false" />

    <!-- 独立侧边栏：文档目录 -->
    <aside class="docs-sidebar" :class="{ 'is-open': drawerOpen }">
      <div class="docs-brand">
        <i class="bi bi-journal-bookmark" aria-hidden="true" />
        <span class="docs-brand-text">文档</span>
      </div>

      <div v-if="projects.length > 1" class="docs-version-row">
        <FormSelect
          :model-value="version"
          :options="projectOptions"
          aria-label="文档项目"
          @update:model-value="switchProject" />
      </div>

      <nav class="docs-nav" aria-label="文档目录">
        <RouterLink
          to="/docs"
          class="docs-nav-item"
          :class="{ 'is-active': !version }"
          @click="drawerOpen = false"
        >
          <i class="bi bi-collection" aria-hidden="true" />
          <span class="docs-nav-text">全部项目</span>
        </RouterLink>

        <template v-if="version">
          <RouterLink
            :to="`/docs/${version}`"
            class="docs-nav-item"
            :class="{ 'is-active': !route.params.slug }"
            @click="drawerOpen = false"
          >
            <i class="bi bi-house" aria-hidden="true" />
            <span class="docs-nav-text">概览</span>
          </RouterLink>

          <!-- 侧边栏默认全部收起，只有第一个目录默认展开 -->
          <NavNode
            v-for="(node, index) in tree"
            :key="node.id"
            :node="node"
            :version="version"
            :default-expanded="index === 0"
            @select="drawerOpen = false"
          />

          <p v-if="!loading && !tree.length" class="docs-empty">暂无文档内容</p>
        </template>
      </nav>

      <div class="docs-sidebar-footer">
        <RouterLink to="/" class="docs-nav-item">
          <i class="bi bi-box-arrow-left" aria-hidden="true" />
          <span class="docs-nav-text">返回前台</span>
        </RouterLink>
      </div>
    </aside>

    <!-- 独立内容区 -->
    <div class="docs-main">
      <header class="docs-topbar">
        <button type="button" class="docs-toggle" title="目录" @click="drawerOpen = !drawerOpen">
          <i class="bi bi-list" aria-hidden="true" />
        </button>
        <div class="docs-topbar-title">{{ pageTitle }}</div>
      </header>

      <main class="docs-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { call } from '@/api/request'
import { listDocsProjects } from '@/api/docs'
import NavNode from './NavNode.vue'

const route = useRoute()
const router = useRouter()

const tree = ref([])
const projects = ref([])
const loading = ref(true)
const drawerOpen = ref(false)

// 未带项目名（/docs）时展示项目列表，不加载目录树
const version = computed(() => String(route.params.version || ''))

// 项目切换下拉（SelectMenu 选项格式：{ value, label }）
const projectOptions = computed(() =>
  projects.value.map((item) => ({ value: item.slug, label: item.name }))
)

// 顶栏标题：详情页显示项目名，项目列表页显示「文档中心」
const pageTitle = computed(() => {
  if (!version.value) return '文档中心'
  const hit = projects.value.find((item) => item.slug === version.value)
  return hit?.name || '文档'
})

async function loadTree() {
  if (!version.value) {
    tree.value = []
    return
  }

  loading.value = true
  try {
    const res = await call('docs-group', 'tree', {
      method: 'GET',
      params: { version: version.value, with_docs: 1 }
    })
    tree.value = res?.data || []
  } catch {
    tree.value = []
  } finally {
    loading.value = false
  }
}

async function loadProjects() {
  try {
    const res = await listDocsProjects()
    projects.value = res?.data || []
  } catch {
    projects.value = []
  }
}

function switchProject(next) {
  if (!next || next === version.value) return
  router.push(`/docs/${next}`)
}

// 项目列表页没有 loading 态的目录树，进来时把 loading 收掉
watch(
  version,
  (val) => {
    if (val) loadTree()
    else {
      tree.value = []
      loading.value = false
    }
  },
  { immediate: true }
)

// 切路由自动收起移动端抽屉
watch(() => route.path, () => {
  drawerOpen.value = false
})

onMounted(loadProjects)
</script>

<style scoped>
.docs-shell {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
}

/* ---------- 侧边栏 ---------- */
.docs-sidebar {
  position: sticky;
  top: 0;
  z-index: 60;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 260px;
  height: 100vh;
  padding: 18px 12px 12px;
  background: var(--bg-card);
  border-right: 1px solid var(--border-soft);
}
.docs-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 8px 14px;
  margin-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
  font-family: var(--font-serif);
  font-size: 17px;
  font-weight: 700;
  color: var(--text);
  letter-spacing: 1px;
}
.docs-brand i {
  font-size: 18px;
  color: var(--primary);
}
.docs-brand-text {
  flex: 1;
}
.docs-version-row {
  padding: 0 8px 10px;
}

.docs-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.docs-nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--text-soft);
  text-decoration: none;
  border-radius: var(--radius-sm);
  transition: color 0.15s, background-color 0.15s;
}
.docs-nav-item i {
  flex-shrink: 0;
  font-size: 14px;
  color: var(--text-light);
  transition: color 0.15s;
}
.docs-nav-item:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.docs-nav-item:hover i {
  color: var(--primary);
}
.docs-nav-item.is-active {
  color: var(--primary);
  font-weight: 600;
  background: var(--accent-wash);
}
.docs-nav-item.is-active i {
  color: var(--primary);
}
.docs-nav-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.docs-empty {
  margin: 14px 10px;
  font-size: 12px;
  color: var(--text-light);
}

.docs-sidebar-footer {
  padding-top: 10px;
  margin-top: 8px;
  border-top: 1px solid var(--border-soft);
}

/* ---------- 内容区 ---------- */
.docs-main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}
.docs-topbar {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 56px;
  padding: 0 24px;
  background: var(--bg);
  border-bottom: 1px solid var(--border-soft);
}
.docs-topbar-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.docs-toggle {
  display: none;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  font-size: 16px;
  color: var(--text-soft);
  background: transparent;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.docs-content {
  flex: 1;
  width: 100%;
  max-width: 980px;
  margin: 0 auto;
  padding: 28px 24px 64px;
}

/* ---------- 响应式 ---------- */
@media (max-width: 900px) {
  .docs-sidebar {
    position: fixed;
    left: 0;
    top: 0;
    z-index: 200;
    transform: translateX(-100%);
    transition: transform 0.2s ease;
    box-shadow: 0 0 24px rgba(0, 0, 0, 0.18);
  }
  .docs-sidebar.is-open {
    transform: translateX(0);
  }
  .docs-mask {
    position: fixed;
    inset: 0;
    z-index: 190;
    background: rgba(0, 0, 0, 0.42);
  }
  .docs-toggle {
    display: inline-flex;
  }
  .docs-topbar {
    padding: 0 16px;
  }
  .docs-content {
    padding: 20px 16px calc(40px + env(safe-area-inset-bottom));
  }
}
</style>
