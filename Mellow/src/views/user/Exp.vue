<template>
  <div class="exp-page">
    <!-- 成长概览：等级 / 经验进度 / 关键数字 -->
    <section class="card card-pad hero">
      <div class="hero-top">
        <div class="hero-avatar-wrap">
          <img
            :src="user?.avatar || defaultAvatar"
            class="hero-avatar"
            alt="avatar"
            width="64"
            height="64"
            loading="lazy"
            decoding="async" />
          <span class="hero-lv">Lv.{{ levelValue ?? 0 }}</span>
        </div>

        <div class="hero-main">
          <div class="hero-name-row">
            <h1 class="hero-name">{{ user?.nickname || '未登录' }}</h1>
            <span v-if="genderText" class="tag">{{ genderText }}</span>
            <span v-if="levelName" class="tag is-primary">{{ levelName }}</span>
          </div>

          <p class="hero-desc">{{ user?.description || '这个人很懒，什么都没留下' }}</p>

          <div class="bar-head">
            <span class="bar-exp">{{ expText }}</span>
            <span class="bar-percent">{{ expPercent }}%</span>
          </div>
          <div class="exp-bar" role="progressbar" :aria-valuenow="expPercent" aria-valuemin="0" aria-valuemax="100">
            <div class="exp-bar-fill" :style="{ width: expPercent + '%' }" />
          </div>
          <p class="hero-next">{{ nextTip }}</p>
        </div>
      </div>

      <!-- 三个关键数字：一眼能看到「我在哪、有多少、还差多少」 -->
      <div class="hero-stats">
        <div class="stat">
          <span class="stat-label">当前等级</span>
          <strong class="stat-value">{{ levelValue ?? 0 }}</strong>
        </div>
        <div class="stat">
          <span class="stat-label">累计经验</span>
          <strong class="stat-value">{{ exp }}</strong>
        </div>
        <div class="stat">
          <span class="stat-label">距下一级</span>
          <strong class="stat-value">{{ expGapText }}</strong>
        </div>
      </div>
    </section>

    <!-- 经验获取途径：卡片网格，窄屏不再横向滚动 -->
    <section class="card card-pad">
      <header class="block-head">
        <h3 class="block-title">经验获取途径</h3>
        <span class="block-meta">每日有次数上限的按上限计算</span>
      </header>

      <div v-if="loadingRules" class="is-loading"><span class="spinner" /> 加载中...</div>
      <div v-else-if="rules.length" class="rule-grid">
        <div v-for="r in rules" :key="r.type" class="rule">
          <span class="rule-icon"><i :class="iconOf(r.type)" /></span>
          <div class="rule-main">
            <strong class="rule-name">{{ r.name }}</strong>
            <span class="rule-limit">{{ Number(r.daily_limit) > 0 ? `每天最多 ${r.daily_limit} 次` : '不限次数' }}</span>
          </div>
          <span class="rule-value">+{{ r.value }}</span>
        </div>
      </div>
      <p v-else class="empty-text">暂无经验规则</p>
    </section>

    <!-- 等级体系：阶梯列表，默认只展开当前等级附近，避免长列表 -->
    <section class="card card-pad">
      <header class="block-head">
        <h3 class="block-title">等级体系</h3>
        <button v-if="levels.length > visibleLimit" class="link-btn" @click="levelsExpanded = !levelsExpanded">
          {{ levelsExpanded ? '收起' : `展开全部 ${levels.length} 级` }}
        </button>
      </header>

      <div v-if="loadingLevels" class="is-loading"><span class="spinner" /> 加载中...</div>
      <ol v-else-if="levels.length" class="ladder">
        <li
          v-for="lv in visibleLevels"
          :key="lv.id"
          class="ladder-item"
          :class="{ 'is-current': isCurrentLevel(lv) }">
          <span class="ladder-dot" aria-hidden="true" />
          <div class="ladder-main">
            <div class="ladder-name-row">
              <strong class="ladder-name">{{ lv.name }}</strong>
              <span class="tag">Lv.{{ lv.value }}</span>
              <span v-if="isCurrentLevel(lv)" class="tag is-primary">当前</span>
            </div>
            <p class="ladder-desc">{{ lv.description || '—' }}</p>
          </div>
          <span class="ladder-exp">{{ lv.exp }}<span class="ladder-exp-unit"> EXP</span></span>
        </li>
      </ol>
      <p v-else class="empty-text">暂无等级数据</p>
    </section>

    <!-- 经验明细：明细数据按需加载，点开弹窗才请求，页面首屏不为它等待 -->
    <section class="card card-pad log-entry">
      <div class="log-entry-main">
        <span class="log-entry-icon"><i class="bi bi-clock-history" /></span>
        <div class="log-entry-text">
          <h3 class="log-entry-title">经验明细</h3>
          <p class="log-entry-desc">按时间倒序记录每一次经验增减，含来源、变动值与获得时间。</p>
        </div>
      </div>
      <button type="button" class="btn btn-sm btn-primary log-entry-btn" @click="logVisible = true">
        <i class="bi bi-list-ul" /> 查看明细
      </button>
    </section>

    <ExpLogDialog :visible="logVisible" :uid="user?.id || 0" @update:visible="logVisible = $event" />
  </div>
</template>

<script setup>
/**
 * 我的经验（/user/exp）
 *
 * 性能要点：
 *   1. 首屏 0 个明细请求：经验明细（数量最多、增长最快的数据）改成弹窗按需拉取，
 *      进页面不再为它等一次网络往返。
 *   2. 规则 / 等级是配置型数据：空闲时段加载 + localStorage 缓存 30 分钟，
 *      二次进入不请求、也不闪 loading。
 *   3. 规则与等级由表格改成网格 / 阶梯：窄屏没有横向滚动，等级默认只渲染当前 ±2 级。
 */
import { computed, onMounted, ref } from 'vue'
import { getLevels, getExpRules } from '@/api/users'
import { useUserStore } from '@/stores/user'
import { storeToRefs } from 'pinia'
import { cache } from '@/utils/cache'
import { expIconOf } from '@/utils/exp'
import ExpLogDialog from '@/components/ExpLogDialog.vue'

const userStore = useUserStore()
const { user } = storeToRefs(userStore)

// 配置型数据的缓存键（规则 / 等级很少变，30 分钟够用）
const CACHE_KEY_RULES = 'user-exp-rules'
const CACHE_KEY_LEVELS = 'user-exp-levels'
const CACHE_TTL_MINUTES = 30

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

// 明细弹窗（数据由弹窗自己按需加载）
const logVisible = ref(false)

const iconOf = expIconOf

/* ---------------- 经验规则 / 等级 ---------------- */

const rules = ref([])
const levels = ref([])
const loadingRules = ref(false)
const loadingLevels = ref(false)
const levelsExpanded = ref(false)

// 折叠状态下显示「当前等级 ±2」，没匹配到时退回前 5 级
const visibleLimit = 5
const currentIndex = computed(() => levels.value.findIndex((lv) => isCurrentLevel(lv)))
const visibleLevels = computed(() => {
  if (levelsExpanded.value || levels.value.length <= visibleLimit) return levels.value

  const index = currentIndex.value
  if (index < 0) return levels.value.slice(0, visibleLimit)

  const start = Math.max(0, index - 2)
  return levels.value.slice(start, Math.max(visibleLimit, index + 3))
})

function isCurrentLevel(lv) {
  const current = levelValue.value
  // 未拿到等级信息时不要把 Lv.0 误标成「当前」
  if (current === null || current === undefined) return false
  return Number(lv.value) === Number(current)
}

async function loadRules() {
  if (rules.value.length) return
  loadingRules.value = true
  try {
    const res = await getExpRules()
    const data = Array.isArray(res.data) ? res.data : res.data?.data || []
    // 经验值高的排前面：用户更关心「做什么最划算」
    data.sort((a, b) => Number(b.value) - Number(a.value))
    rules.value = data
    cache.set(CACHE_KEY_RULES, data, CACHE_TTL_MINUTES)
  } catch {
    rules.value = []
  } finally {
    loadingRules.value = false
  }
}

async function loadLevels() {
  if (levels.value.length) return
  loadingLevels.value = true
  try {
    const res = await getLevels()
    levels.value = res.data?.data || []
    cache.set(CACHE_KEY_LEVELS, levels.value, CACHE_TTL_MINUTES)
  } catch {
    levels.value = []
  } finally {
    loadingLevels.value = false
  }
}

/**
 * 规则与等级是「配置数据」：先读缓存，没有就等首屏画完再请求
 *
 * 页面打开时不再有阻塞请求，内容该出来的立刻出来，
 * 两个配置接口在浏览器空闲时补上，用户看不到转圈排队。
 */
function loadSecondary() {
  const cachedRules = cache.get(CACHE_KEY_RULES)
  if (Array.isArray(cachedRules) && cachedRules.length) rules.value = cachedRules
  else loadRules()

  const cachedLevels = cache.get(CACHE_KEY_LEVELS)
  if (Array.isArray(cachedLevels) && cachedLevels.length) levels.value = cachedLevels
  else loadLevels()
}

function scheduleSecondary() {
  if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
    window.requestIdleCallback(loadSecondary, { timeout: 1200 })
    return
  }
  setTimeout(loadSecondary, 120)
}

/* ---------------- 登录态派生 ---------------- */

const genderText = computed(() => {
  const g = user.value?.gender
  if (g === 'boy' || g === 1) return '男'
  if (g === 'girl' || g === 2) return '女'
  return ''
})

const levelInfo = computed(() => user.value?.result?.level || null)
const levelName = computed(() => levelInfo.value?.current?.name || '')
const levelValue = computed(() => (levelInfo.value?.current?.value === undefined ? null : Number(levelInfo.value.current.value)))
const exp = computed(() => Number(user.value?.exp) || 0)

const expPercent = computed(() => {
  const cur = Number(levelInfo.value?.current?.exp) || 0
  const next = Number(levelInfo.value?.next?.exp) || 0
  if (!next || next <= cur) return 100
  const percent = ((exp.value - cur) / (next - cur)) * 100
  return Math.max(0, Math.min(100, Math.round(percent)))
})

const expText = computed(() => {
  const next = Number(levelInfo.value?.next?.exp) || 0
  return next ? `${exp.value} / ${next} EXP` : `${exp.value} EXP`
})

// 还差多少经验：直接给差额，比让用户自己减 x/y 更好读
const expGap = computed(() => {
  const cur = Number(levelInfo.value?.current?.exp) || 0
  const next = Number(levelInfo.value?.next?.exp) || 0
  if (!next || next <= cur) return 0
  return Math.max(0, next - exp.value)
})

const expGapText = computed(() => {
  if (!Number(levelInfo.value?.next?.exp)) return '已满级'
  if (expGap.value <= 0) return '可升级'
  return `${expGap.value} EXP`
})

const nextTip = computed(() => {
  if (!Number(levelInfo.value?.next?.exp)) return '已经是最高等级'

  const name = String(levelInfo.value?.next?.name || '').trim()
  const value = Number(levelInfo.value?.next?.value) || 0
  const label = [name, value ? `Lv.${value}` : ''].filter(Boolean).join(' ')

  if (expGap.value <= 0) return `经验已达标，等待升级到 ${label || '下一级'}`
  return `还差 ${expGap.value} EXP 升到 ${label || '下一级'}`
})

onMounted(() => {
  scheduleSecondary()
})
</script>

<style scoped>
.exp-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 卡片标题 ---------- */
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-soft);
}
.block-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.block-meta {
  font-size: 12px;
  color: var(--text-light);
  white-space: nowrap;
}

/* ---------- 成长概览 ---------- */
.hero-top {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}
.hero-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}
.hero-avatar {
  display: block;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--border);
  background: var(--bg-muted);
}
/* 等级角标压在头像右下角：原来写在一行文字里，长昵称会把它挤走 */
.hero-lv {
  position: absolute;
  right: -6px;
  bottom: -4px;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--primary);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  border: 2px solid var(--bg-card);
}
.hero-main {
  flex: 1;
  min-width: 0;
}
.hero-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.hero-name {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
}
.tag {
  flex-shrink: 0;
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 11px;
  line-height: 1.7;
  white-space: nowrap;
}
.tag.is-primary {
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-weight: 600;
}
.hero-desc {
  margin: 6px 0 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
}
.bar-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.bar-percent {
  color: var(--primary-deep);
  font-weight: 600;
}
.exp-bar {
  height: 10px;
  border-radius: 999px;
  background: var(--border);
  overflow: hidden;
}
.exp-bar-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--primary), var(--primary-deep));
  transition: width 0.35s ease;
}
.hero-next {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--primary-deep);
}
.hero-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft);
}
.stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}
.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ---------- 获取途径 ---------- */
.rule-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 10px;
}
.rule {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
}
.rule-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 14px;
}
.rule-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.rule-name {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rule-limit {
  font-size: 11px;
  color: var(--text-light);
}
.rule-value {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}

/* ---------- 等级阶梯 ---------- */
.ladder {
  position: relative;
  margin: 0;
  padding: 0 0 0 18px;
  list-style: none;
}
.ladder::before {
  content: '';
  position: absolute;
  left: 4px;
  top: 6px;
  bottom: 6px;
  width: 2px;
  background: var(--border);
  border-radius: 2px;
}
.ladder-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.ladder-item + .ladder-item {
  margin-top: 2px;
}
.ladder-item:hover {
  background: var(--bg-muted);
}
.ladder-dot {
  position: absolute;
  left: -18px;
  top: 50%;
  width: 10px;
  height: 10px;
  margin-top: -5px;
  border-radius: 50%;
  background: var(--bg-card);
  border: 2px solid var(--border);
}
.ladder-item.is-current {
  background: var(--accent-soft);
}
.ladder-item.is-current .ladder-dot {
  background: var(--primary);
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.ladder-main {
  flex: 1;
  min-width: 0;
}
.ladder-name-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ladder-name {
  font-size: 13px;
  font-weight: 600;
}
.ladder-item.is-current .ladder-name {
  color: var(--primary-deep);
}
.ladder-desc {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
}
.ladder-exp {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-soft);
  font-variant-numeric: tabular-nums;
}
.ladder-item.is-current .ladder-exp {
  color: var(--primary-deep);
}
.ladder-exp-unit {
  font-size: 11px;
  font-weight: 500;
  color: var(--text-light);
}

/* ---------- 明细入口 ---------- */
.log-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
}
.log-entry-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.log-entry-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 17px;
}
.log-entry-text {
  min-width: 0;
}
.log-entry-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}
.log-entry-desc {
  margin: 3px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.log-entry-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

/* ---------- 其它 ---------- */
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
.is-loading {
  padding: 20px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}
.empty-text {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}

@media (max-width: 560px) {
  .hero-top {
    gap: 12px;
  }
  .hero-avatar {
    width: 52px;
    height: 52px;
  }
  .hero-name {
    font-size: 16px;
  }
  .hero-stats {
    gap: 6px;
  }
  .stat-value {
    font-size: 16px;
  }
  .rule-grid {
    grid-template-columns: 1fr;
  }
  .ladder {
    padding-left: 16px;
  }
  .ladder-dot {
    left: -16px;
  }
  .log-entry {
    align-items: flex-start;
  }
  .log-entry-btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
