<template>
  <!-- 独立页面（路由 /blackroom，与 404 同级）：不带任何导航 / 布局组件，自己撑满视口 -->
  <div class="blackroom-page">
    <main class="blackroom-container">
      <SectionTitle title="小黑屋">
        <template #extra>
          <div class="blackroom-head-extra">
            <span>共 {{ total }} 条封禁记录</span>
            <router-link to="/" class="btn btn-ghost btn-sm">
              <i class="bi bi-arrow-left" /> 返回首页
            </router-link>
          </div>
        </template>
      </SectionTitle>

      <p class="blackroom-desc">
        以下用户因违反社区规范被封禁，昵称已做脱敏处理。若你认为封禁有误，可在登录后提交申诉。
      </p>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="card card-pad">
        <EmptyState text="目前没有用户被关进小黑屋" />
      </div>

      <!-- 瀑布流卡片：CSS 多列实现，卡片不跨列断开（break-inside: avoid） -->
      <template v-else>
        <div class="ban-waterfall">
          <article v-for="item in list" :key="item.id" class="ban-card">
            <header class="ban-card-head">
              <img class="ban-card-avatar" :src="avatarOf(item)" :alt="nameOf(item)" loading="lazy" />
              <div class="ban-card-who">
                <span class="ban-card-name">{{ nameOf(item) }}</span>
                <span class="ban-card-time">
                  <i class="bi bi-clock" /> {{ formatDate(item.ban_time) }}
                </span>
              </div>
              <span class="ban-state" :class="stateOf(item).tone">{{ stateOf(item).label }}</span>
            </header>

            <div class="ban-card-tags">
              <span v-for="tag in banTags(item)" :key="tag" class="type-tag">{{ tag }}</span>
            </div>

            <div class="ban-card-reason">
              <span class="ban-card-label">封禁原因</span>
              <p class="ban-card-text">{{ item.reason || '未说明' }}</p>
            </div>

            <footer class="ban-card-foot">
              <span class="ban-duration" :class="{ 'is-perm': durationOf(item) === 0 }">
                {{ durationText(item) }}
              </span>
              <span v-if="violationOf(item) > 1" class="ban-card-foot-item">
                违规 {{ violationOf(item) }} 次
              </span>
              <span v-if="expiresOf(item) > 0" class="ban-card-foot-item">
                到期 {{ formatDate(expiresOf(item)) }}
              </span>
            </footer>
          </article>
        </div>

        <Pagination
          v-if="total > pageSize"
          :current="page"
          :total="total"
          :page-size="pageSize"
          @update:current="changePage"
        />
      </template>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import { getBlackroom } from '@/api/users'
import { formatDate } from '@/utils/time'

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'

// 全量封禁位掩码（model.BanTypeAll = 31）：命中时只显示一个「全面封禁」标签，避免一排标签刷屏
const BAN_TYPE_ALL = 31

// 封禁状态文案与配色（0 生效中 1 已解封 2 已撤销 3 申诉中 4 申诉通过 5 申诉驳回）
const BAN_STATES = {
  0: { label: '封禁中', tone: 'is-danger' },
  1: { label: '已解封', tone: 'is-ok' },
  2: { label: '已撤销', tone: 'is-muted' },
  3: { label: '申诉中', tone: 'is-warn' },
  4: { label: '申诉通过', tone: 'is-ok' },
  5: { label: '申诉驳回', tone: 'is-danger' }
}

const list = ref([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

// ---------- 展示辅助 ----------
function avatarOf(item) {
  return item.result?.user?.avatar || defaultAvatar
}

function nameOf(item) {
  return item.result?.user?.nickname || '匿名用户'
}

function durationOf(item) {
  return Number(item.duration) || 0
}

// 封禁时长：0 表示永久
function durationText(item) {
  const d = durationOf(item)
  return d === 0 ? '永久封禁' : `${d} 天`
}

function violationOf(item) {
  return Number(item.violation_num) || 0
}

function expiresOf(item) {
  return Number(item.expires_at) || 0
}

function stateOf(item) {
  return BAN_STATES[Number(item.status)] || { label: '已解除', tone: 'is-muted' }
}

// 限制范围标签：全面封禁（31）时收敛成一个标签，否则按位掩码逐个展示
function banTags(item) {
  if (Number(item.ban_type) === BAN_TYPE_ALL) return ['全面封禁']
  const types = item.result?.ban_types || []
  if (!types.length) return ['全面封禁']
  return types.map((t) => t.name)
}

async function load() {
  loading.value = true
  try {
    const res = await getBlackroom({
      page: page.value,
      limit: pageSize
    })
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

onMounted(load)
</script>

<style scoped>
/* 独立页面容器：没有布局组件兜底，这里自己撑满视口并居中内容 */
.blackroom-page {
  min-height: 100vh;
  padding: 32px 16px 64px;
  background: var(--bg);
}
/* 单栏页面，比三栏布局的内容区窄一些更易读 */
.blackroom-container {
  max-width: 960px;
  margin: 0 auto;
}
.blackroom-head-extra {
  display: flex;
  align-items: center;
  gap: 12px;
}

.blackroom-desc {
  margin: 0 0 16px;
  padding: 12px 14px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-left: 3px solid var(--danger);
  border-radius: 0 var(--radius) var(--radius) 0;
}

.is-loading {
  padding: 48px;
  text-align: center;
  color: var(--text-muted);
}

/* ---------- 瀑布流（CSS 多列，卡片不跨列断开） ---------- */
.ban-waterfall {
  column-count: 3;
  column-gap: 16px;
}
@media (max-width: 1024px) {
  .ban-waterfall {
    column-count: 2;
  }
}
@media (max-width: 640px) {
  .ban-waterfall {
    column-count: 1;
  }
}

.ban-card {
  break-inside: avoid;
  page-break-inside: avoid;
  margin: 0 0 16px;
  padding: 14px 16px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.ban-card:hover {
  border-color: var(--border);
  box-shadow: var(--shadow-md);
}

/* 卡片头：头像 + 昵称/时间 + 状态 */
.ban-card-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ban-card-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.ban-card-who {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  line-height: 1.4;
}
.ban-card-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ban-card-time {
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.ban-card-time i {
  margin-right: 2px;
}

/* 状态徽章 */
.ban-state {
  flex-shrink: 0;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 999px;
  white-space: nowrap;
  border: 1px solid transparent;
}
.ban-state.is-danger {
  color: var(--danger);
  background: rgba(199, 72, 42, 0.12);
}
.ban-state.is-warn {
  color: var(--warning);
  background: rgba(192, 138, 46, 0.14);
}
.ban-state.is-ok {
  color: var(--success);
  background: rgba(46, 122, 92, 0.14);
}
.ban-state.is-muted {
  color: var(--text-muted);
  background: var(--bg-muted);
}

/* 限制范围标签 */
.ban-card-tags {
  margin-top: 10px;
  display: flex;
  flex-wrap: wrap;
  gap: 4px 0;
}
.type-tag {
  display: inline-block;
  margin: 2px 4px 2px 0;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 999px;
  background: rgba(199, 72, 42, 0.1);
  color: var(--danger);
  white-space: nowrap;
}

/* 封禁原因 */
.ban-card-reason {
  margin-top: 10px;
}
.ban-card-label {
  display: block;
  margin-bottom: 2px;
  font-size: 11px;
  color: var(--text-muted);
}
.ban-card-text {
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
  word-break: break-word;
}

/* 卡片底：时长 + 违规次数 + 到期时间 */
.ban-card-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding-top: 10px;
  border-top: 1px dashed var(--border-soft);
  font-size: 12px;
  color: var(--text-muted);
}
.ban-duration {
  padding: 1px 8px;
  border-radius: 999px;
  font-weight: 600;
  color: var(--text-soft);
  background: var(--bg-muted);
}
.ban-duration.is-perm {
  color: var(--danger);
  background: rgba(199, 72, 42, 0.12);
}
.ban-card-foot-item {
  font-variant-numeric: tabular-nums;
}
</style>
