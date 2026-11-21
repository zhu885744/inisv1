<template>
  <!-- 卡片右上角的「三个点」菜单：把分享 / 打赏 / 举报等低频操作收进来，底部只留互动按钮 -->
  <div ref="rootRef" class="card-menu">
    <button
      type="button"
      class="card-menu-trigger"
      :class="{ 'is-open': open }"
      :title="title"
      aria-haspopup="menu"
      :aria-expanded="open"
      @click.stop="toggle"
    >
      <i class="bi bi-three-dots" />
    </button>

    <Transition name="menu-pop">
      <div v-if="open" class="card-menu-panel" role="menu">
        <button
          v-for="item in items"
          :key="item.key"
          type="button"
          class="card-menu-item"
          :class="{ 'is-danger': item.danger }"
          role="menuitem"
          @click.stop="choose(item)"
        >
          <i v-if="item.icon" :class="item.icon" />
          <span>{{ item.label }}</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup>
/**
 * 卡片右上角的「更多操作」菜单
 *
 * 首页发现流的文章 / 动态卡片共用：右上角一个三点按钮，点开是纵向菜单，
 * 用来收纳「分享 / 打赏 / 举报 / 置顶 / 删除」这类低频操作，
 * 卡片底部只保留点赞、评论、阅读这类高频互动。
 *
 * 用法：
 *   <CardActionMenu :items="[{ key: 'share', label: '分享', icon: 'bi bi-share' }]" @select="onMenu" />
 * items 项：{ key, label, icon?, danger? }，点击后 emit('select', key)
 */
import { ref, onMounted, onBeforeUnmount } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  title: { type: String, default: '更多操作' }
})

const emit = defineEmits(['select'])

const open = ref(false)
const rootRef = ref(null)

function toggle() {
  open.value = !open.value
}

function close() {
  open.value = false
}

function choose(item) {
  open.value = false
  emit('select', item.key)
}

// 点菜单外任意位置 / 按 Esc 关闭：手机上尤其需要，避免菜单一直挂在卡片上
function onDocPointerDown(e) {
  if (!open.value) return
  if (rootRef.value && !rootRef.value.contains(e.target)) close()
}

function onKeydown(e) {
  if (e.key === 'Escape') close()
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocPointerDown)
  document.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointerDown)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.card-menu {
  position: relative;
  flex-shrink: 0;
}

.card-menu-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  font-size: 15px;
  line-height: 1;
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: 50%;
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease);
}
.card-menu-trigger:hover,
.card-menu-trigger.is-open {
  color: var(--text);
  background: var(--bg-muted);
}

.card-menu-panel {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  z-index: 40;
  min-width: 128px;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}
/* 右边缘卡片里，菜单不要顶出屏幕 */
@media (max-width: 640px) {
  .card-menu-panel {
    right: -4px;
  }
}

.card-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--text-soft);
  text-align: left;
  white-space: nowrap;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease);
}
.card-menu-item:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.card-menu-item i {
  font-size: 14px;
  opacity: 0.9;
}
.card-menu-item.is-danger {
  color: var(--danger);
}
.card-menu-item.is-danger:hover {
  color: var(--danger);
  background: var(--accent-soft);
}

/* 展开动画：轻微下滑 + 淡入，移动端不拖沓 */
.menu-pop-enter-active,
.menu-pop-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}
.menu-pop-enter-from,
.menu-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
