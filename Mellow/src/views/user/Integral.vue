<template>
  <div class="integral-page">
    <!-- 积分概览 -->
    <section class="card card-pad">
      <div class="balance-top">
        <div class="balance-main">
          <span class="balance-label">当前积分</span>
          <strong class="balance-num">{{ summary.integral }}</strong>
          <div class="today-row">
            <span class="today-pill is-plus">今日 +{{ summary.today_income }}</span>
            <span class="today-pill is-minus">今日 -{{ summary.today_expense }}</span>
          </div>
        </div>
        <RouterLink to="/goods" class="balance-link">
          积分商城 <i class="bi bi-arrow-right" aria-hidden="true" />
        </RouterLink>
      </div>

      <dl class="balance-stats">
        <div class="balance-stat">
          <dt>累计获得</dt>
          <dd class="is-plus">{{ summary.total_income }}</dd>
        </div>
        <div class="balance-stat">
          <dt>累计消耗</dt>
          <dd class="is-minus">{{ summary.total_expense }}</dd>
        </div>
      </dl>
    </section>

    <!-- 卡密兑换 -->
    <section v-if="isLogged" class="card card-pad">
      <header class="section-head">
        <h3 class="section-title">卡密兑换</h3>
        <span v-if="myCards.length" class="section-extra">{{ myCards.length }} 张待兑换</span>
      </header>

      <div class="redeem-row">
        <FormInput
          v-model="cardCode"
          class="redeem-input"
          type="text"
          maxlength="64"
          autocomplete="off"
          spellcheck="false"
          placeholder="请输入卡密"
          :disabled="redeeming"
          @keyup.enter="redeem" />
        <button
          type="button"
          class="btn btn-primary"
          :disabled="redeeming || !cardCode.trim()"
          @click="redeem">
          {{ redeeming ? '兑换中...' : '立即兑换' }}
        </button>
      </div>
      <p class="hint">不区分大小写，空格与连字符会被忽略；请勿向他人泄露卡密。</p>

      <!-- 我的待兑换卡密：签到等活动直接发到账号里的卡密在这里找回 -->
      <div v-if="myCards.length" class="mine-cards">
        <div class="mine-cards-head">
          <span>我的待兑换卡密</span>
          <button class="text-btn" type="button" :disabled="redeeming" @click="redeemAll">全部兑换</button>
        </div>

        <div v-for="item in myCards" :key="item.card_id || item.card" class="mine-card">
          <span class="mine-card-icon"><i class="bi bi-ticket-perforated" aria-hidden="true" /></span>
          <div class="mine-card-main">
            <code class="mine-card-code">{{ item.card }}</code>
            <span class="mine-card-sub">
              {{ cardSourceText(item.source) }}
              <template v-if="item.expired"> · <em class="is-expired">已过期</em></template>
            </span>
          </div>
          <span class="mine-card-face">{{ item.value }} 积分</span>
          <button
            class="btn btn-sm btn-ghost mine-card-btn"
            type="button"
            :disabled="redeeming || item.expired"
            @click="redeemCard(item)">
            兑换
          </button>
        </div>
      </div>
    </section>

    <!-- 今日任务 -->
    <section v-if="isLogged" class="card card-pad">
      <header class="section-head">
        <h3 class="section-title">今日任务</h3>
        <span class="section-extra">
          已完成 {{ tasks.done_count }}/{{ tasks.total_count }}
          <template v-if="tasks.streak > 0"> · 连签 {{ tasks.streak }} 天</template>
        </span>
      </header>

      <div v-if="tasks.total_count" class="task-summary">
        <span class="task-summary-value">+{{ tasks.today_income }}</span>
        <span class="task-summary-label">今日已获得积分</span>
        <span class="task-summary-progress">{{ donePercent }}%</span>
      </div>

      <div v-if="loadingTasks" class="is-loading"><span class="spinner" /> 加载中...</div>
      <ul v-else-if="taskList.length" class="task-list">
        <li v-for="t in taskList" :key="t.type" class="task-item" :class="{ 'is-done': t.done }">
          <span class="task-icon"><i :class="taskIcon(t)" aria-hidden="true" /></span>
          <div class="task-main">
            <div class="task-row">
              <span class="task-name">{{ t.name }}</span>
              <span class="task-state">
                <template v-if="t.done">
                  <i class="bi bi-check-circle-fill" aria-hidden="true" /> 已完成
                </template>
                <template v-else-if="Number(t.remain) > 0">还可 +{{ t.remain }}</template>
                <template v-else>可继续</template>
              </span>
            </div>
            <div class="task-progress">
              <span class="task-bar"><span class="task-fill" :style="{ width: t.progress + '%' }" /></span>
              <span class="task-count">{{ t.today_count }}/{{ Number(t.daily_limit) > 0 ? t.daily_limit : '不限' }}</span>
            </div>
          </div>
        </li>
      </ul>
      <p v-else class="empty-text">暂无任务</p>
    </section>

    <!-- 积分明细 -->
    <section v-if="isLogged" class="card card-pad">
      <header class="section-head">
        <h3 class="section-title">积分明细</h3>
        <span v-if="logSummary" class="section-extra">
          收入 <span class="is-plus">{{ logSummary.income }}</span>
          · 支出 <span class="is-minus">{{ logSummary.expense }}</span>
        </span>
      </header>

      <div class="log-filter">
        <div class="chip-group">
          <button
            v-for="opt in directionOptions"
            :key="opt.value"
            type="button"
            class="chip"
            :class="{ 'is-active': direction === opt.value }"
            @click="setDirection(opt.value)">
            {{ opt.label }}
          </button>
        </div>
        <FormSelect
          v-model="typeFilter"
          variant="pill"
          :options="typeOptions"
          icon="bi bi-funnel"
          placeholder="全部类型"
          @change="setType" />
      </div>

      <ul v-if="loadingLogs" class="log-list" aria-hidden="true">
        <li v-for="i in 4" :key="i" class="log-item is-skeleton">
          <span class="log-icon skeleton-block" />
          <div class="log-main">
            <span class="skeleton-line" style="width: 44%" />
            <span class="skeleton-line" style="width: 22%" />
          </div>
          <span class="skeleton-line" style="width: 40px" />
        </li>
      </ul>

      <p v-else-if="!logRows.length" class="empty-text">暂无积分记录</p>

      <template v-else>
        <ul class="log-list">
          <li v-for="row in logRows" :key="row.id" class="log-item">
            <span class="log-icon"><i :class="row.icon" aria-hidden="true" /></span>
            <div class="log-main">
              <span class="log-title">{{ row.title }}</span>
              <time class="log-time" :datetime="row.datetime" :title="row.timeFull">{{ row.timeAgo }}</time>
            </div>
            <span class="log-value" :class="{ 'is-minus': row.minus }">{{ row.valueText }}</span>
          </li>
        </ul>

        <div class="log-more">
          <button v-if="hasMore" class="btn btn-sm btn-ghost" :disabled="loadingMore" @click="loadMore">
            {{ loadingMore ? '加载中...' : `加载更多（还有 ${restCount} 条）` }}
          </button>
          <span v-else class="log-end">已经到底啦</span>
        </div>
      </template>
    </section>
  </div>
</template>

<script setup>
/**
 * 我的积分（/user/integral）
 *
 * 性能要点：
 *   1. 首屏只发 3 个请求（概览 / 今日任务 / 明细第一页），且并行；
 *      「积分规则」只用于类型下拉的选项、「我的待兑换卡密」只影响卡密卡片，
 *      都挪到空闲时段加载（规则另有 30 分钟本地缓存），不占首屏关键路径。
 *   2. 明细支持分页加载（原来只显示前 20 条、看不到更多），并用请求序号丢弃过期响应，
 *      快速切换筛选不会出现「点了支出却闪回收入」的错乱。
 *   3. 明细由表格改成列表：窄屏没有横向滚动，行内容在 computed 里一次算好。
 *   4. 概览把「今日收支」提到主数字下面，累计收支降为次级信息，一眼看到重点。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { getIntegral, getIntegralLogs, getIntegralRules, getIntegralTasks, redeemIntegralCard } from '@/api/goods'
import { getMyIntegralCards } from '@/api/integral'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { cache } from '@/utils/cache'
import { formatTime, fromNow } from '@/utils/time'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'

const userStore = useUserStore()
const isLogged = computed(() => userStore.isLogged)

// 积分规则缓存（只影响类型下拉选项，30 分钟内复用）
const CACHE_KEY_RULES = 'user-integral-rules'
const LOG_LIMIT = 20

const summary = ref({ integral: 0, total_income: 0, total_expense: 0, today_income: 0, today_expense: 0 })
const tasks = ref({ list: [], today_income: 0, done_count: 0, total_count: 0, streak: 0 })

const loadingTasks = ref(false)

/* ---------------- 积分明细 ---------------- */

const logs = ref([])
const logSummary = ref(null)
const loadingLogs = ref(false)
const loadingMore = ref(false)
const logPage = ref(1)
const logCount = ref(0)
// 请求序号：筛选切得快时，只认最后一次请求的结果
let logReqId = 0

const direction = ref('')
const typeFilter = ref('')
const directionOptions = [
  { value: '', label: '全部' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' }
]

const hasMore = computed(() => logs.value.length < logCount.value)
const restCount = computed(() => Math.max(0, logCount.value - logs.value.length))

const donePercent = computed(() => {
  const total = Number(tasks.value.total_count) || 0
  if (!total) return 0
  return Math.round((Number(tasks.value.done_count) / total) * 100)
})

// 类型 -> 图标（规则接口里没给图标时兜底）
const TYPE_ICON = {
  'check-in': 'bi bi-calendar-check',
  login: 'bi bi-box-arrow-in-right',
  'article-create': 'bi bi-file-earmark-text',
  comment: 'bi bi-chat-dots',
  moments: 'bi bi-chat-square-quote',
  share: 'bi bi-share',
  buy: 'bi bi-bag-check',
  refund: 'bi bi-arrow-counterclockwise',
  give: 'bi bi-gift',
  card: 'bi bi-ticket-perforated',
  unlock: 'bi bi-unlock',
  tip: 'bi bi-send',
  'tip-income': 'bi bi-inbox'
}

// 明细类型 -> 文案
const TYPE_NAME = {
  'check-in': '每日签到',
  login: '每日登录',
  'article-create': '发布文章',
  comment: '发表评论',
  moments: '发布动态',
  share: '分享内容',
  buy: '积分兑换',
  refund: '订单退款',
  give: '管理员积分调整',
  card: '卡密兑换',
  unlock: '兑换内容可见权限',
  tip: '打赏支出',
  'tip-income': '收到打赏'
}

const iconOf = (type) => TYPE_ICON[type] || 'bi bi-coin'

function taskIcon(task) {
  // 后端规则里的 icon 形如 bi-coin（不带 bi 前缀），两种写法都要兼容
  const raw = String(task?.icon || '').trim()
  if (raw) return raw.startsWith('bi ') ? raw : `bi ${raw}`
  return iconOf(task?.type)
}

// 行数据一次算好：模板里不再逐行调用函数
const logRows = computed(() =>
  logs.value.map((item) => {
    const value = Number(item.value) || 0
    const time = Number(item.create_time) || 0
    return {
      id: item.id,
      icon: iconOf(item.type),
      title: item.description || TYPE_NAME[item.type] || '积分变动',
      minus: value < 0,
      valueText: `${value > 0 ? '+' : ''}${value}`,
      timeAgo: fromNow(time),
      timeFull: formatTime(time),
      datetime: time ? new Date(time * 1000).toISOString() : ''
    }
  })
)

// 类型下拉选项：首项「全部类型」+ 规则类型 + 内置类型（按 value 去重）
const rules = ref([])
const typeOptions = computed(() => {
  const list = [{ value: '', label: '全部类型' }]
  const seen = new Set([''])
  const candidates = [
    ...(rules.value || []).map((r) => ({ value: r.type, label: r.name })),
    { value: 'buy', label: '积分兑换' },
    { value: 'card', label: '卡密兑换' },
    { value: 'refund', label: '退款返还' },
    { value: 'give', label: '管理员积分调整' }
  ]
  for (const opt of candidates) {
    if (!opt.value || seen.has(opt.value)) continue
    seen.add(opt.value)
    list.push(opt)
  }
  return list
})

const taskList = computed(() => tasks.value.list || [])

async function loadSummary() {
  if (!isLogged.value) return
  try {
    const res = await getIntegral()
    const data = res.data || {}
    summary.value = {
      integral: Number(data.integral) || 0,
      total_income: Number(data.total_income) || 0,
      total_expense: Number(data.total_expense) || 0,
      today_income: Number(data.today_income) || 0,
      today_expense: Number(data.today_expense) || 0
    }
  } catch {
    // 拦截器已提示
  }
}

async function loadRules() {
  if (rules.value.length) return
  try {
    const res = await getIntegralRules()
    const data = Array.isArray(res.data) ? res.data : res.data?.data || []
    data.sort((a, b) => Number(b.value) - Number(a.value))
    rules.value = data
    cache.set(CACHE_KEY_RULES, data, 30)
  } catch {
    rules.value = []
  }
}

async function loadTasks() {
  if (!isLogged.value) return
  loadingTasks.value = true
  try {
    const res = await getIntegralTasks()
    tasks.value = res.data || tasks.value
  } catch {
    // 忽略：任务卡展示空列表即可
  } finally {
    loadingTasks.value = false
  }
}

/**
 * 明细加载
 * @param {boolean} more 是否翻页（false = 重新从第一页拉，用于筛选变化 / 兑换后刷新）
 */
async function loadLogs(more = false) {
  if (!isLogged.value) return

  const reqId = ++logReqId
  if (more) {
    loadingMore.value = true
  } else {
    // 重新拉第一页（筛选变化 / 兑换后刷新）
    logPage.value = 1
    loadingLogs.value = true
  }

  try {
    const params = { page: more ? logPage.value : 1, limit: LOG_LIMIT }
    if (direction.value) params.direction = direction.value
    if (typeFilter.value) params.type = typeFilter.value

    const res = await getIntegralLogs(params)
    // 期间又切了筛选：这次结果作废（否则会把新筛选的数据覆盖成旧的）
    if (reqId !== logReqId) return

    const items = res.data?.data || []
    if (more) {
      const seen = new Set(logs.value.map((item) => item.id))
      logs.value = [...logs.value, ...items.filter((item) => !seen.has(item.id))]
    } else {
      logs.value = items
    }

    logCount.value = Number(res.data?.count) || logs.value.length
    logSummary.value = res.data?.summary || null
  } catch {
    if (reqId !== logReqId) return
    if (!more) {
      logs.value = []
      logSummary.value = null
    }
  } finally {
    if (reqId === logReqId) {
      loadingLogs.value = false
      loadingMore.value = false
    }
  }
}

function loadMore() {
  if (loadingLogs.value || loadingMore.value || !hasMore.value) return
  logPage.value += 1
  loadLogs(true)
}

function setDirection(value) {
  if (direction.value === value) return
  direction.value = value
  loadLogs()
}

function setType() {
  loadLogs()
}

/* ---------------- 卡密兑换 ---------------- */

const cardCode = ref('')
const redeeming = ref(false)
const myCards = ref([])

const cardSourceMap = {
  'check-in': '签到获得'
}

function cardSourceText(source) {
  return cardSourceMap[source] || '活动获得'
}

async function loadMyCards() {
  if (!isLogged.value) {
    myCards.value = []
    return
  }
  try {
    const res = await getMyIntegralCards({ limit: 20 })
    myCards.value = Array.isArray(res?.data) ? res.data : []
  } catch {
    // 拿不到不影响页面其它部分
    myCards.value = []
  }
}

// 兑换成功后刷新：概览 / 明细 / 待兑换卡密
async function refreshAfterRedeem() {
  await Promise.all([loadSummary(), loadLogs(), loadMyCards()])
}

async function redeem() {
  const code = cardCode.value.trim()
  if (!code) {
    toast.warning('请输入卡密')
    return
  }
  redeeming.value = true
  try {
    const res = await redeemIntegralCard(code)
    const value = Number(res?.data?.value) || 0
    toast.success(value > 0 ? `兑换成功，获得 ${value} 积分` : '兑换成功')
    cardCode.value = ''
    await refreshAfterRedeem()
  } catch {
    // 失败原因（卡密不存在/已使用/已过期/尝试过于频繁等）已由请求拦截器统一提示
  } finally {
    redeeming.value = false
  }
}

// 兑换列表里的某一张（等价于填入输入框再点兑换）
async function redeemCard(item) {
  if (redeeming.value || !item?.card) return
  cardCode.value = item.card
  await redeem()
}

// 一键兑换全部：逐张兑换，单张失败不影响其它（失败的会留在列表里）
async function redeemAll() {
  if (redeeming.value || !myCards.value.length) return

  redeeming.value = true
  let success = 0
  let income = 0

  for (const item of [...myCards.value]) {
    if (item.expired) continue
    try {
      const res = await redeemIntegralCard(item.card)
      if (Number(res?.code) === 200) {
        success++
        income += Number(res?.data?.value) || 0
      }
    } catch {
      // 单张失败（已过期 / 已被使用等）跳过即可
    }
  }

  redeeming.value = false
  toast.success(success ? `已兑换 ${success} 张，共获得 ${income} 积分` : '没有可兑换的卡密')

  await refreshAfterRedeem()
}

/* ---------------- 首屏调度 ---------------- */

/**
 * 规则 / 待兑换卡密：不影响首屏主内容，放到浏览器空闲时再拉
 *
 * 规则先读缓存（30 分钟），没有才请求；卡密每次都要新的（兑换后会变）。
 */
function loadSecondary() {
  const cached = cache.get(CACHE_KEY_RULES)
  if (Array.isArray(cached) && cached.length) rules.value = cached
  else loadRules()

  loadMyCards()
}

function scheduleSecondary() {
  if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
    window.requestIdleCallback(loadSecondary, { timeout: 1200 })
    return
  }
  setTimeout(loadSecondary, 120)
}

// 只在登录态就绪后启动一次（刷新时用户信息可能比页面晚一拍）
let started = false

function start() {
  if (started || !isLogged.value) return
  started = true

  // 关键路径：概览 + 任务 + 明细第一页，三个请求并行
  loadSummary()
  loadTasks()
  loadLogs()
  scheduleSecondary()
}

onMounted(start)
watch(isLogged, (val) => {
  if (val) start()
})
</script>

<style scoped>
.integral-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* ---------- 通用小节 ---------- */
.section-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.section-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.section-extra {
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.is-plus {
  color: var(--success);
}
.is-minus {
  color: var(--danger);
}
.is-expired {
  font-style: normal;
  color: var(--danger);
}
.hint {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
}
.text-btn {
  padding: 0;
  font-size: 12px;
  font-weight: 500;
  color: var(--primary);
  background: none;
  border: none;
  cursor: pointer;
}
.text-btn:hover {
  text-decoration: underline;
}
.text-btn:disabled {
  color: var(--text-light);
  cursor: not-allowed;
  text-decoration: none;
}

/* ---------- 概览 ---------- */
.balance-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.balance-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.balance-label {
  font-size: 12px;
  color: var(--text-muted);
}
.balance-num {
  font-size: 32px;
  font-weight: 600;
  line-height: 1.1;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.today-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}
.today-pill {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.today-pill.is-plus {
  background: rgba(47, 143, 91, 0.12);
}
.today-pill.is-minus {
  background: rgba(217, 84, 77, 0.1);
}
.balance-link {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-muted);
  text-decoration: none;
  white-space: nowrap;
}
.balance-link:hover {
  color: var(--primary);
}
.balance-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 14px 0 0;
  padding-top: 12px;
  border-top: 1px dashed var(--border-soft);
}
.balance-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.balance-stat dt {
  font-size: 12px;
  color: var(--text-muted);
}
.balance-stat dd {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}

/* ---------- 卡密兑换 ---------- */
.redeem-row {
  display: flex;
  gap: 8px;
}
.redeem-input {
  flex: 1;
  min-width: 0;
  letter-spacing: 1px;
}
.redeem-input::placeholder {
  letter-spacing: normal;
}
.mine-cards {
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid var(--border-soft);
}
.mine-cards-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.mine-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.mine-card:last-child {
  border-bottom: none;
}
.mine-card-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 13px;
}
.mine-card-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.mine-card-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
  letter-spacing: 1px;
  color: var(--text);
  user-select: all;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mine-card-sub {
  font-size: 11px;
  color: var(--text-light);
}
.mine-card-face {
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}
.mine-card-btn {
  flex-shrink: 0;
}

/* ---------- 今日任务 ---------- */
.task-summary {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 10px 12px;
  margin-bottom: 10px;
  border-radius: var(--radius-sm);
  background: var(--accent-soft);
}
.task-summary-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}
.task-summary-label {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text-soft);
}
.task-summary-progress {
  flex-shrink: 0;
  font-size: 12px;
  font-weight: 600;
  color: var(--primary-deep);
  font-variant-numeric: tabular-nums;
}
.task-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}
.task-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 0;
  border-bottom: 1px solid var(--border-soft);
}
.task-item:last-child {
  border-bottom: none;
}
.task-icon {
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
.task-item.is-done .task-icon {
  background: rgba(47, 143, 91, 0.12);
  color: var(--success);
}
.task-main {
  flex: 1;
  min-width: 0;
}
.task-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}
.task-name {
  font-size: 13px;
  color: var(--text);
}
.task-item.is-done .task-name {
  color: var(--text-muted);
}
.task-state {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.task-item.is-done .task-state {
  color: var(--success);
}
.task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
}
.task-bar {
  flex: 1;
  height: 4px;
  border-radius: 999px;
  background: var(--bg-muted);
  overflow: hidden;
}
.task-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--primary);
  transition: width 0.3s;
}
.task-item.is-done .task-fill {
  background: var(--success);
}
.task-count {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}

/* ---------- 明细 ---------- */
.log-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}
.chip-group {
  display: flex;
  gap: 4px;
}
.chip {
  display: inline-flex;
  align-items: center;
  height: var(--control-h-sm);
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: none;
  color: var(--text-muted);
  font-size: 12px;
  font-family: inherit;
  line-height: 1.5;
  cursor: pointer;
  transition: color 0.18s, background 0.18s;
}
.chip:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.chip.is-active {
  background: var(--bg-muted);
  color: var(--text);
  font-weight: 600;
}
.log-list {
  display: flex;
  flex-direction: column;
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
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 14px;
}
.log-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.log-title {
  font-size: 13px;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log-time {
  font-size: 11px;
  color: var(--text-light);
}
.log-value {
  flex-shrink: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--success);
  font-variant-numeric: tabular-nums;
}
.log-value.is-minus {
  color: var(--danger);
}
.log-more {
  display: flex;
  justify-content: center;
  padding-top: 12px;
}
.log-end {
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
  animation: integral-skeleton 1.2s ease-in-out infinite;
}
.skeleton-line + .skeleton-line {
  margin-top: 6px;
}
@keyframes integral-skeleton {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.45;
  }
}

.is-loading {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}
.empty-text {
  padding: 20px;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .balance-num {
    font-size: 28px;
  }
  .redeem-row {
    flex-direction: column;
  }
  .mine-card-face {
    font-size: 12px;
  }
  .log-title {
    font-size: 12.5px;
  }
}
</style>
