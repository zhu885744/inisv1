<template>
  <Teleport to="body">
    <Transition name="exp-dialog">
      <div v-if="visible" class="exp-mask" @click.self="close">
        <div class="exp-dialog card" role="dialog" aria-modal="true" aria-label="经验明细">
          <header class="exp-head">
            <h3 class="exp-title">
              <i class="bi bi-clock-history" aria-hidden="true" />
              经验明细
              <span v-if="count" class="exp-count">共 {{ count }} 条</span>
            </h3>
            <button type="button" class="exp-close" aria-label="关闭" @click="close">
              <i class="bi bi-x-lg" aria-hidden="true" />
            </button>
          </header>

          <!-- 弹窗自己滚动：列表翻页不影响页面滚动位置 -->
          <div ref="scroller" class="exp-body" @scroll.passive="onScroll">
            <ul v-if="loading" class="log-list" aria-hidden="true">
              <li v-for="i in 6" :key="i" class="log-item is-skeleton">
                <span class="log-icon skeleton-block" />
                <div class="log-main">
                  <span class="skeleton-line" style="width: 46%" />
                  <span class="skeleton-line" style="width: 26%" />
                </div>
                <div class="log-right">
                  <span class="skeleton-line" style="width: 34px" />
                  <span class="skeleton-line" style="width: 52px" />
                </div>
              </li>
            </ul>

            <p v-else-if="!rows.length" class="empty-text">
              <template v-if="!uid">登录后查看经验明细</template>
              <template v-else-if="error">
                加载失败
                <button type="button" class="link-btn" @click="reload">重试</button>
              </template>
              <template v-else>暂无经验记录</template>
            </p>

            <template v-else>
              <ul class="log-list">
                <li v-for="row in rows" :key="row.id" class="log-item">
                  <span class="log-icon"><i :class="row.icon" aria-hidden="true" /></span>
                  <div class="log-main">
                    <span class="log-title">{{ row.title }}</span>
                    <span v-if="row.extra" class="log-extra">{{ row.extra }}</span>
                  </div>
                  <div class="log-right">
                    <span class="log-value" :class="{ 'is-minus': row.minus }">{{ row.valueText }}</span>
                    <time class="log-time" :datetime="row.datetime" :title="row.timeFull">{{ row.timeAgo }}</time>
                  </div>
                </li>
              </ul>

              <p class="log-footer">
                <template v-if="loadingMore">加载中...</template>
                <template v-else-if="error">加载失败，<button type="button" class="link-btn" @click="loadMore">重试</button></template>
                <template v-else-if="hasMore">上滑加载更多</template>
                <template v-else>已经到底啦</template>
              </p>
            </template>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * 经验明细弹窗
 *
 * 为什么放弹窗：明细是「按需深看」的数据，页面首屏没必要为它发请求；
 * 打开时才拉第一页，滚动到底自动翻页，关掉即释放。
 *
 * 用法：
 *   <ExpLogDialog :visible="show" :uid="user.id" @update:visible="show = $event" />
 */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { getExpLogs } from '@/api/users'
import { buildExpRow } from '@/utils/exp'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 明细按用户过滤（exp/all 是通用接口，用 where 限定 uid）
  uid: { type: [Number, String], default: 0 }
})

const emit = defineEmits(['update:visible'])

const PAGE_SIZE = 20

const logs = ref([])
const count = ref(0)
const page = ref(1)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref(false)
const scroller = ref(null)

const rows = computed(() => logs.value.map(buildExpRow))
const hasMore = computed(() => logs.value.length < count.value)

async function load(reset = false) {
  if (!props.uid) {
    logs.value = []
    count.value = 0
    return
  }

  if (reset) {
    page.value = 1
    loading.value = true
  } else {
    loadingMore.value = true
  }
  error.value = false

  try {
    const res = await getExpLogs({
      page: page.value,
      limit: PAGE_SIZE,
      where: JSON.stringify({ uid: Number(props.uid) }),
      field: 'id,value,type,description,json,create_time'
    })

    const items = Array.isArray(res.data) ? res.data : res.data?.data || []
    if (reset) {
      logs.value = items
    } else {
      // 翻页期间可能有新记录插到前面，按 id 去重
      const seen = new Set(logs.value.map((item) => item.id))
      logs.value = [...logs.value, ...items.filter((item) => !seen.has(item.id))]
    }

    count.value = Number(res.data?.count) || logs.value.length
  } catch {
    error.value = true
    if (reset) logs.value = []
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function reload() {
  load(true)
}

function loadMore() {
  if (loading.value || loadingMore.value || !hasMore.value || !props.uid) return
  page.value += 1
  load(false)
}

// 滚动容器内触底（离底部 120px）就翻页
function onScroll(event) {
  const el = event.target
  if (!el || !hasMore.value) return
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 120) loadMore()
}

function close() {
  emit('update:visible', false)
}

function onKeydown(event) {
  if (event.key === 'Escape') close()
}

// 记住打开前的 body overflow，关闭时还原（避免把其他弹窗的状态覆盖掉）
let lastOverflow = null

watch(
  () => props.visible,
  async (val) => {
    if (val) {
      lastOverflow = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      window.addEventListener('keydown', onKeydown)
      logs.value = []
      count.value = 0
      page.value = 1
      load(true)
      await nextTick()
      if (scroller.value) scroller.value.scrollTop = 0
      return
    }

    document.body.style.overflow = lastOverflow ?? ''
    window.removeEventListener('keydown', onKeydown)
  }
)

onBeforeUnmount(() => {
  document.body.style.overflow = lastOverflow ?? ''
  window.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.exp-mask {
  position: fixed;
  inset: 0;
  /* 页面弹窗 3000 < 附件库 3500 < 确认弹窗 3600：明细弹窗与普通页面弹窗同层 */
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}

.exp-dialog {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 520px;
  max-height: min(78vh, 640px);
  padding: 0;
  overflow: hidden;
}

.exp-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border-soft);
}

.exp-title {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.exp-count {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}

.exp-close {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 50%;
  background: var(--bg-muted);
  color: var(--text-soft);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.exp-close:hover {
  background: var(--accent-soft);
  color: var(--primary-deep);
}

.exp-body {
  flex: 1;
  min-height: 0;
  padding: 0 16px;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}

/* ---------- 列表 ---------- */
.log-list {
  margin: 0;
  padding: 0;
  list-style: none;
}

.log-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-soft);
}

.log-item:last-child {
  border-bottom: none;
}

.log-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 15px;
}

.log-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.log-title {
  font-size: 13px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-extra {
  font-size: 12px;
  color: var(--text-light);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-right {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.log-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}

.log-value.is-minus {
  color: var(--danger);
}

.log-time {
  font-size: 11px;
  color: var(--text-light);
}

.log-footer {
  margin: 0;
  padding: 12px 0 16px;
  text-align: center;
  font-size: 12px;
  color: var(--text-light);
}

/* 骨架屏 */
.is-skeleton {
  pointer-events: none;
}

.skeleton-block {
  background: var(--bg-muted);
}

.skeleton-line {
  display: block;
  height: 10px;
  border-radius: 999px;
  background: var(--bg-muted);
  animation: exp-skeleton 1.2s ease-in-out infinite;
}

.skeleton-line + .skeleton-line {
  margin-top: 6px;
}

@keyframes exp-skeleton {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.45;
  }
}

.empty-text {
  padding: 32px 0;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}

.link-btn {
  padding: 0;
  border: none;
  background: none;
  color: var(--primary-deep);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
}

.link-btn:hover {
  text-decoration: underline;
}

/* 过渡 */
.exp-dialog-enter-active,
.exp-dialog-leave-active {
  transition: opacity 0.2s ease;
}

.exp-dialog-enter-active .exp-dialog,
.exp-dialog-leave-active .exp-dialog {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.exp-dialog-enter-from,
.exp-dialog-leave-to {
  opacity: 0;
}

.exp-dialog-enter-from .exp-dialog,
.exp-dialog-leave-to .exp-dialog {
  opacity: 0;
  transform: translateY(12px) scale(0.98);
}

@media (max-width: 560px) {
  .exp-mask {
    align-items: flex-end;
    padding: 0;
  }

  /* 移动端做成底部抽屉：拇指更好点，列表也更高 */
  .exp-dialog {
    max-width: none;
    max-height: 86vh;
    border-radius: var(--radius) var(--radius) 0 0;
    padding-bottom: env(safe-area-inset-bottom, 0);
  }
}
</style>
