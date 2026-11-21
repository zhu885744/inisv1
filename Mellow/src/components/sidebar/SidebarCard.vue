<template>
  <section class="sb-card" :class="{ 'is-collapsed': collapsed, 'is-flat': flat }">
    <header v-if="hasHeader" class="sb-head">
      <h3 class="sb-title">
        <i v-if="icon" :class="icon" aria-hidden="true" />
        <span class="sb-title-text">{{ title }}</span>
        <slot name="badge" />
      </h3>

      <div class="sb-extra">
        <RouterLink v-if="moreHref && !isExternal" :to="moreHref" class="sb-more">
          {{ moreText || '更多' }} <i class="bi bi-arrow-right" aria-hidden="true" />
        </RouterLink>
        <a
          v-else-if="moreHref"
          :href="moreHref"
          class="sb-more"
          target="_blank"
          rel="noopener noreferrer"
        >
          {{ moreText || '更多' }} <i class="bi bi-box-arrow-up-right" aria-hidden="true" />
        </a>

        <button
          v-if="collapsible"
          type="button"
          class="sb-toggle"
          :title="collapsed ? '展开' : '收起'"
          :aria-expanded="!collapsed"
          @click="toggle"
        >
          <i class="bi" :class="collapsed ? 'bi-chevron-down' : 'bi-chevron-up'" aria-hidden="true" />
        </button>
      </div>
    </header>

    <div v-show="!collapsed" class="sb-body">
      <slot />
    </div>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { safeUrl } from '@/utils/url'

/**
 * 侧边栏卡片外壳
 *
 * 只负责「标题 / 图标 / 更多入口 / 折叠」，内容由各模块自己通过默认插槽提供。
 * 折叠状态按 storeKey（一般是模块 id）记在 localStorage，刷新后保持用户的选择。
 */
const props = defineProps({
  title: { type: String, default: '' },
  icon: { type: String, default: '' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  /** 折叠状态的存储键：传了才持久化 */
  storeKey: { type: [String, Number], default: '' },
  /** 无标题模块（如纯 HTML 卡片）去掉头部 */
  flat: { type: Boolean, default: false }
})

const STORAGE_PREFIX = 'mellow_sidebar_collapsed:'

const hasHeader = computed(() => !props.flat && (Boolean(props.title) || Boolean(props.icon) || props.collapsible))
// 「更多」入口同样出自后台配置：先过协议白名单（非法协议返回空串，整个入口不渲染），
// 带协议或协议相对的走新窗口，其余仍交给 router —— 与原口径一致
const moreHref = computed(() => safeUrl(props.moreUrl))
const isExternal = computed(() => /^([a-z][a-z0-9+.-]*:|\/\/)/i.test(moreHref.value))

const collapsed = ref(readCollapsed(props.defaultCollapsed))

function readCollapsed(fallback) {
  if (!props.storeKey) return fallback
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + props.storeKey)
    if (raw === '1') return true
    if (raw === '0') return false
  } catch {
    // 隐私模式下读不到就按默认值
  }
  return fallback
}

function toggle() {
  collapsed.value = !collapsed.value
  if (!props.storeKey) return
  try {
    localStorage.setItem(STORAGE_PREFIX + props.storeKey, collapsed.value ? '1' : '0')
  } catch {
    // 忽略
  }
}
</script>

<style scoped>
.sb-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-soft);
  box-shadow: var(--shadow-sm);
  padding: 14px;
  transition: box-shadow 0.2s;
}

.sb-card:hover {
  box-shadow: var(--shadow);
}

.sb-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}

.sb-title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.sb-title i {
  color: var(--primary);
  font-size: 13px;
}

.sb-title-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sb-extra {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.sb-more {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: normal;
  white-space: nowrap;
}

.sb-more:hover {
  color: var(--primary);
}

.sb-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-muted);
  font-size: 11px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.sb-toggle:hover {
  background: var(--bg-muted);
  color: var(--primary);
}

.sb-body {
  min-width: 0;
}
</style>
