<template>
  <div class="checkin-page">
    <!-- 未登录：引导登录后回到本页（/auth/login?redirect=/checkin） -->
    <section v-if="!isLogged" class="state-card">
      <span class="state-icon"><i class="bi bi-calendar2-check" /></span>
      <h2 class="state-title">登录后即可签到</h2>
      <p class="state-desc">签到可得经验、积分等奖励，连续签到与周期签到还有额外加成。</p>
      <router-link to="/auth/login?redirect=/checkin" class="btn btn-primary">去登录</router-link>
    </section>

    <template v-else>
      <!-- 功能未开启 -->
      <section v-if="!status.enabled" class="state-card">
        <span class="state-icon"><i class="bi bi-calendar2-x" /></span>
        <h2 class="state-title">签到暂未开放</h2>
        <p class="state-desc">站点当前关闭了每日签到，稍后再来看看吧。</p>
      </section>

      <template v-else>
        <!-- 签到主卡：环形进度 + 状态 + 主操作，关键数据并入卡内 -->
        <section class="hero" :class="{ 'is-checked': status.checked }">
          <div class="hero-top">
            <span class="hero-tag">
              <i class="bi bi-calendar2-check" />
              {{ status.name }}
            </span>
            <span class="hero-date">{{ calendar.month }} 月 {{ calendar.today }} 日</span>
          </div>

          <div class="hero-main">
            <!-- 环形进度：优先周期进度，没有周期配置时用「连签 / 下个里程碑」 -->
            <div class="hero-ring" :style="{ '--ring': `${ringProgress}%` }">
              <span class="hero-ring-core">
                <i class="bi" :class="status.checked ? 'bi-check-lg' : 'bi-lightning-charge-fill'" />
              </span>
            </div>

            <div class="hero-text">
              <h1 class="hero-title">{{ status.checked ? '今日已签到' : '今日还未签到' }}</h1>
              <p class="hero-sub">
                <template v-if="status.checked">
                  {{ status.tips || '签到成功' }} · 已连续 {{ status.streak }} 天
                </template>
                <template v-else>
                  已连续 {{ status.streak }} 天<template v-if="nextMilestone">，距「{{ nextMilestone.label || `连续 ${nextMilestone.day} 天` }}」还差 {{ nextMilestone.days_left }} 天</template>
                </template>
              </p>

              <div class="hero-rewards">
                <template v-if="!status.checked">
                  <span class="hero-rewards-label">签到可得</span>
                  <span v-for="item in totalChips" :key="item.asset" class="chip">
                    <i class="bi" :class="item.icon" />{{ item.text }}
                  </span>
                  <span v-if="!totalChips.length" class="hero-rewards-label">奖励</span>
                </template>
                <!-- 概率 / 区间奖励不是必得，单独一行弱化提示，不混进「签到可得」 -->
                <span v-if="variableRewards.length" class="hero-variable">
                  <i class="bi bi-dice-5" /> 另有 {{ variableRewards.join('、') }}
                </span>
              </div>
            </div>

            <div class="hero-actions">
              <button
                class="btn btn-primary hero-btn"
                :disabled="loading || status.checked"
                @click="doSign"
              >
                <span v-if="loading" class="spinner spinner-light" />
                {{ loading ? '签到中...' : status.checked ? '今日已签到' : '立即签到' }}
              </button>
              <button
                v-if="makeup.enabled"
                class="btn btn-ghost hero-ghost"
                type="button"
                @click="scrollToMakeup"
              >
                <i class="bi bi-calendar-plus" /> 补签
              </button>
            </div>
          </div>

          <div class="hero-stats">
            <div class="hero-stat">
              <span class="hero-stat-label">连续签到</span>
              <strong class="hero-stat-value">{{ status.streak }}<em>天</em></strong>
            </div>
            <div class="hero-stat">
              <span class="hero-stat-label">本月签到</span>
              <strong class="hero-stat-value">{{ status.month_days }}<em>天</em></strong>
            </div>
            <div class="hero-stat">
              <span class="hero-stat-label">周期进度</span>
              <strong class="hero-stat-value">
                {{ cycleDays.length ? `${status.cycle_day}/${cycleDays.length}` : '—' }}
              </strong>
            </div>
            <div class="hero-stat">
              <span class="hero-stat-label">下个里程碑</span>
              <strong class="hero-stat-value">
                {{ nextMilestone ? nextMilestone.day : '—' }}<em v-if="nextMilestone">天</em>
              </strong>
            </div>
          </div>
        </section>

        <!-- 奖励明细（按来源分组，含概率 / 区间说明） -->
        <section v-if="rewardGroups.length" class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-gift" /> 今日奖励明细</h3>
            <span class="panel-extra">{{ status.checked ? '本次已发放' : '签到可得' }}</span>
          </header>

          <div v-for="group in rewardGroups" :key="group.key" class="reward-group">
            <div class="reward-group-head">
              <span class="reward-group-title">{{ group.label }}</span>
              <span v-if="group.hint" class="reward-group-hint">{{ group.hint }}</span>
            </div>
            <div class="reward-items">
              <span v-for="(item, index) in group.items" :key="index" class="chip">
                <i class="bi" :class="item.icon" />{{ chipText(item) }}
                <em v-if="item.chance" class="chip-note">{{ item.chance }}% 概率</em>
                <em v-else-if="item.label" class="chip-note">{{ item.label }}</em>
              </span>
            </div>
          </div>
        </section>

        <!-- 今日获得的卡密（发放后才拿得到明文；纯卡密，站内无需兑换） -->
        <section v-if="todayCards.length" class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-ticket-perforated" /> 今日获得的卡密</h3>
          </header>

          <div v-for="(item, index) in todayCards" :key="index" class="code-row">
            <template v-if="item.card">
              <code class="code-text">{{ item.card }}</code>
              <button class="btn btn-sm btn-ghost" type="button" @click="copyText(item.card)">
                <i class="bi bi-clipboard" /> 复制
              </button>
            </template>
            <span v-else class="code-missing">活动卡密已发完（库存不足），本次未发放</span>
          </div>

          <p class="panel-note">
            <i class="bi bi-info-circle" />
            这是本次获得的卡密，请自行保存 / 使用（纯卡密，与积分无关，站内不做兑换）。
          </p>
        </section>

        <!-- 周期奖励 -->
        <section v-if="cycleDays.length" class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-calendar3-week" /> 周期签到奖励</h3>
            <span class="panel-extra">每 {{ cycleDays.length }} 天一轮</span>
          </header>

          <div class="cycle-track">
            <div
              v-for="item in cycleDays"
              :key="item.day"
              class="cycle-node"
              :class="{
                'is-done': item.day < status.cycle_day,
                'is-current': item.day === status.cycle_day
              }"
            >
              <span class="cycle-dot">
                <i v-if="item.day < status.cycle_day" class="bi bi-check-lg" />
                <template v-else>{{ item.day }}</template>
              </span>
              <span class="cycle-reward">
                <template v-if="item.rewards.length">
                  <template v-for="(reward, index) in item.rewards" :key="index">
                    +{{ reward.value }} {{ reward.unit }}
                  </template>
                </template>
                <template v-else>基础</template>
              </span>
              <span class="cycle-label">{{ item.label || ' ' }}</span>
            </div>
          </div>
        </section>

        <!-- 里程碑 + 月度奖励 -->
        <section v-if="milestones.length || monthly.length" class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-trophy" /> 连签与月签奖励</h3>
            <span v-if="nextMilestone" class="panel-extra">
              还差 {{ nextMilestone.days_left }} 天达成下一档
            </span>
          </header>

          <div v-if="nextMilestone" class="ms-progress">
            <div class="ms-progress-bar">
              <div class="ms-progress-fill" :style="{ width: `${milestoneProgress}%` }" />
            </div>
            <span class="ms-progress-text">
              {{ nextMilestone.label || `连续 ${nextMilestone.day} 天` }} · {{ milestoneProgress }}%
            </span>
          </div>

          <div class="ms-list">
            <div
              v-for="item in ladderList"
              :key="`${item.kind}-${item.day}`"
              class="ms-row"
              :class="{ 'is-reached': item.reached }"
            >
              <span class="ms-node">
                <i class="bi" :class="item.reached ? 'bi-check-lg' : item.icon" />
              </span>
              <div class="ms-info">
                <span class="ms-title">
                  {{ item.scope }} {{ item.day }} 天
                  <em v-if="item.label">{{ item.label }}</em>
                </span>
                <span class="ms-rewards">
                  <template v-for="(reward, index) in item.rewards" :key="index">
                    <i class="bi" :class="reward.icon" /> {{ reward.value }} {{ reward.unit }}
                  </template>
                  <span v-if="!item.rewards.length">—</span>
                </span>
              </div>
              <span class="ms-state">
                {{ item.reached ? '已达成' : `还差 ${item.days_left} 天` }}
              </span>
            </div>
          </div>
        </section>

        <!-- 本月日历 -->
        <section class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-calendar3" /> {{ calendar.year }} 年 {{ calendar.month }} 月</h3>
            <span class="panel-extra">已签到 {{ calendar.total }} 天</span>
          </header>

          <div class="cal-grid">
            <span v-for="week in WEEK_LABELS" :key="week" class="cal-cell cal-week">{{ week }}</span>
            <span v-for="i in calendarBlank" :key="`blank-${i}`" class="cal-cell cal-blank" />
            <span
              v-for="day in calendar.days"
              :key="day.day"
              class="cal-cell cal-day"
              :class="{
                'is-checked': day.checked,
                'is-makeup': day.checked && day.source === 2,
                'is-today': day.day === calendar.today
              }"
              :title="dayTitle(day)"
            >
              {{ day.day }}
            </span>
          </div>

          <div class="cal-legend">
            <span class="cal-legend-item"><i class="cal-dot is-checked" /> 已签到</span>
            <span class="cal-legend-item"><i class="cal-dot is-makeup" /> 补签</span>
            <span class="cal-legend-item"><i class="cal-dot is-today" /> 今天</span>
          </div>
        </section>

        <!-- 补签 -->
        <section v-if="makeup.enabled" ref="makeupRef" class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-calendar-plus" /> 补签</h3>
            <span class="panel-extra">
              {{ makeup.cost }} {{ makeup.unit }} / 次<template v-if="makeup.limit > 0"> · 剩余 {{ makeup.remain }} 次</template>
            </span>
          </header>

          <EmptyState
            v-if="!makeup.dates || !makeup.dates.length"
            icon="bi bi-calendar2-check"
            text="最近没有可补签的日期"
          />

          <template v-else>
            <p class="panel-note">可补签最近 {{ makeup.days }} 天内的日期，补签只发放基础奖励。</p>
            <div class="makeup-dates">
              <button
                v-for="item in makeup.dates"
                :key="item.date"
                type="button"
                class="makeup-date"
                :class="{ 'is-active': selectedMakeup === item.date }"
                @click="selectedMakeup = selectedMakeup === item.date ? 0 : item.date"
              >
                {{ item.text }}
              </button>
            </div>
            <div class="makeup-foot">
              <button
                class="btn btn-primary"
                type="button"
                :disabled="!selectedMakeup || makeupLoading"
                @click="doMakeup"
              >
                <span v-if="makeupLoading" class="spinner spinner-light" />
                {{ makeupLoading ? '补签中...' : `确认补签（-${makeup.cost} ${makeup.unit}）` }}
              </button>
            </div>
          </template>
        </section>

        <!-- 签到排行榜 -->
        <section class="panel">
          <header class="panel-head">
            <h3 class="panel-title"><i class="bi bi-bar-chart-line" /> 签到排行榜</h3>
            <div class="seg">
              <button
                v-for="item in TIME_RANGES"
                :key="item.key"
                type="button"
                class="seg-item"
                :class="{ 'is-active': timeRange === item.key }"
                @click="changeRange(item.key)"
              >
                {{ item.label }}
              </button>
            </div>
          </header>

          <div v-if="rankLoading" class="panel-loading"><span class="spinner" /> 加载中...</div>

          <EmptyState v-else-if="!rankList.length" icon="bi bi-bar-chart-line" text="暂无排行数据" />

          <ul v-else class="rank-list">
            <li
              v-for="(item, index) in rankList"
              :key="item.id || index"
              class="rank-row"
              :class="{ 'is-me': item.is_me }"
            >
              <span class="rank-badge" :class="`is-top-${item.rank}`">{{ item.rank }}</span>
              <img class="rank-avatar" :src="item.avatar || defaultAvatar" :alt="item.nickname || '用户'" loading="lazy" />
              <div class="rank-body">
                <span class="rank-name">{{ item.nickname || '匿名用户' }}</span>
                <span class="rank-meta">
                  <span><i class="bi bi-calendar-check" /> {{ item.check_in_count || 0 }} 次</span>
                  <span><i class="bi bi-star" /> {{ item.total_exp || 0 }}</span>
                  <span><i class="bi bi-coin" /> {{ item.total_integral || 0 }}</span>
                </span>
              </div>
            </li>
          </ul>
        </section>
      </template>
    </template>
  </div>
</template>

<script setup>
/**
 * 每日签到（/checkin）
 *
 * 签到已从经验模块独立：接口为 /api/checkin/*（见 src/api/checkin.js），奖励由后台
 * 「签到管理」页的配置驱动（SYSTEM_CHECKIN_RULES），不再只有「经验 + 积分」两种：
 * - 基础奖励 / 周期奖励（第 N 天）/ 连签加成 / 里程碑 / 月全勤 / 随机奖励（带概率）都能配；
 * - 奖励项按「资产」渲染（经验、积分，或二次开发注册的自定义资产），名称 / 单位 / 图标
 *   都由后端 assets 清单下发，前端不写死；
 * - 未签到时展示「预期奖励」（含概率与随机区间），签到后展示实际发放明细；
 * - 支持补签（消耗积分，次数与范围由后台配置）。
 */
import { ref, reactive, computed, onMounted, watch } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import { useUserStore } from '@/stores/user'
import {
  getCheckinStatus,
  getCheckinCalendar,
  getCheckinRank,
  signIn,
  makeupSign
} from '@/api/checkin'
import { toast } from '@/utils/toast'

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'

const WEEK_LABELS = ['日', '一', '二', '三', '四', '五', '六']

const TIME_RANGES = [
  { key: 'today', label: '今日' },
  { key: 'week', label: '本周' },
  { key: 'month', label: '本月' }
]

// 奖励来源分组 → 展示名（与后端 model/checkin.go 的 CheckinGroup* 一致）
const GROUP_LABELS = {
  base: '基础奖励',
  cycle: '周期奖励',
  streak: '连续签到加成',
  milestone: '里程碑奖励',
  monthly: '月度奖励',
  random: '随机奖励',
  makeup: '补签奖励'
}
const GROUP_ORDER = ['base', 'cycle', 'streak', 'milestone', 'monthly', 'random', 'makeup']

// 卡密资产标识（与后端 model/reward-card.go 的 RewardCardAssetKey 对应）：
// 它的 value 是「面额」而不是数量，展示时需要特殊处理
const RewardCardKey = 'card'

const userStore = useUserStore()
const isLogged = computed(() => userStore.isLogged)

const loading = ref(false)
const makeupLoading = ref(false)
const selectedMakeup = ref(0)
const makeupRef = ref(null)

const status = reactive({
  enabled: true,
  name: '每日签到',
  checked: false,
  check_in_time: 0,
  streak: 0,
  preview_days: 1,
  cycle_day: 0,
  cycle_days: [],
  month_days: 0,
  rewards: {},
  items: [],
  total: {},
  // 概率 / 区间奖励（非必得）：由后端拆分返回，单独提示
  chance_items: [],
  chance_total: {},
  milestones: [],
  monthly: [],
  next_milestone: null,
  next_monthly: null,
  assets: [],
  makeup: {},
  cards: [],
  tips: ''
})

const calendar = reactive({
  year: new Date().getFullYear(),
  month: new Date().getMonth() + 1,
  days: [],
  total: 0,
  today: new Date().getDate()
})

const timeRange = ref('month')
const rankList = ref([])
const rankLoading = ref(false)

// 当月 1 号是星期几（日历空白占位）
const calendarBlank = computed(
  () => new Date(calendar.year, calendar.month - 1, 1).getDay()
)

const cycleDays = computed(() => status.cycle_days || [])
const milestones = computed(() => status.milestones || [])
const monthly = computed(() => status.monthly || [])
const makeup = computed(() => {
  const data = status.makeup || {}
  return {
    enabled: !!data.enabled,
    days: data.days || 0,
    limit: data.limit || 0,
    remain: data.remain || 0,
    cost: data.cost || 0,
    unit: unitOf(data.asset || 'integral'),
    dates: data.dates || []
  }
})

const nextMilestone = computed(() => status.next_milestone || null)

// 里程碑进度：当前连签 / 下个里程碑天数
const milestoneProgress = computed(() => {
  const milestone = nextMilestone.value
  if (!milestone) return 100
  const day = Number(milestone.day) || 1
  return Math.min(100, Math.max(0, Math.round(((status.streak || 0) / day) * 100)))
})

// 主卡环形进度：优先用「周期第几天 / 周期总天数」，没有周期配置时退回里程碑进度
const ringProgress = computed(() => {
  const total = cycleDays.value.length
  if (!total) return milestoneProgress.value
  const current = Number(status.cycle_day || 0)
  return Math.min(100, Math.max(0, Math.round((current / total) * 100)))
})

// 资产清单：asset 标识 → { name, unit, icon }
const assetMap = computed(() => {
  const map = {}
  ;(status.assets || []).forEach((item) => {
    map[item.key] = item
  })
  return map
})

function unitOf(asset) {
  return assetMap.value[asset]?.unit || asset
}

// 主卡上的奖励预览（把各项奖励按资产合并成「12 经验、5 积分、卡密 ×1」）
const totalChips = computed(() => {
  const total = status.total || {}
  return Object.entries(total)
    .filter(([, value]) => Number(value) > 0)
    .map(([asset, value]) => {
      // 卡密的 value 是面额而不是数量，这里按「张数」展示
      if (asset === RewardCardKey) {
        const count = (status.items || []).filter((item) => item.asset === RewardCardKey).length
        return {
          asset,
          text: `卡密 ×${count || 1}`,
          icon: assetMap.value[asset]?.icon || 'bi-ticket-perforated'
        }
      }
      return {
        asset,
        text: `${Number(value)} ${unitOf(asset)}`,
        icon: assetMap.value[asset]?.icon || 'bi-gift'
      }
    })
})

// 今日获得的卡密（后端在发放后写入明细的 extra，未签到 / 没配卡密时为空）
// 纯卡密：只有卡密明文，没有面额；库存发完时 card 为空、card_missing 为 true
const todayCards = computed(() =>
  (status.cards || []).map((item) => ({
    card: item.card || '',
    missing: Boolean(item.card_missing),
    reason: item.reason || ''
  }))
)

// 概率 / 区间奖励的说明（例如「10% 概率 20 积分」「5~15 积分」）
// 这些奖励不是必得，所以单独提示，不能算进「签到可得」的合计里
const variableRewards = computed(() => (status.chance_items || []).map((item) => variableText(item)))

function variableText(item) {
  const unit = item?.unit || unitOf(item?.asset)
  const value = Number(item?.value || 0)
  const min = Number(item?.min || 0)
  const max = Number(item?.max || 0)
  const chance = Number(item?.chance || 0)

  // 只配了 min/max 时按区间展示，配了固定值就展示固定值
  const amount = value > 0 ? `${value} ${unit}` : `${min}~${max} ${unit}`
  return chance > 0 && chance < 100 ? `${chance}% 概率 ${amount}` : amount
}

// 奖励项文案：卡密显示面额（或降级说明），其它资产显示「数值 单位」（区间奖励显示上下限）
function chipText(item) {
  if (item?.asset !== RewardCardKey) {
    const unit = item?.unit || unitOf(item?.asset)
    const value = Number(item?.value || 0)
    const min = Number(item?.min || 0)
    const max = Number(item?.max || 0)
    if (!value && (min > 0 || max > 0)) {
      return `${min}~${max} ${unit}`
    }
    return `${value} ${unit}`
  }

  const extra = item.extra || {}
  if (extra.card_missing) return '卡密（库存已发完）'
  return '卡密 1 张'
}

// 复制文本（卡密）：优先用剪贴板 API，非 HTTPS / 老浏览器走 execCommand 兜底
async function copyText(text) {
  if (!text) return

  try {
    await navigator.clipboard.writeText(text)
    toast.success('卡密已复制')
    return
  } catch {
    // 继续走兜底方案
  }

  const input = document.createElement('textarea')
  input.value = text
  input.style.position = 'fixed'
  input.style.opacity = '0'
  document.body.appendChild(input)
  input.select()
  document.execCommand('copy')
  document.body.removeChild(input)
  toast.success('卡密已复制')
}

// 奖励明细分组（后端 rewards 是按来源分好组的对象）
const rewardGroups = computed(() => {
  const groups = status.rewards || {}
  return GROUP_ORDER.filter((key) => Array.isArray(groups[key]) && groups[key].length)
    .map((key) => {
      // 概率奖励给个提示文案，避免用户以为「一定能拿到」
      const chance = groups[key].filter((item) => item.chance)
      let hint = ''
      if (chance.length) {
        hint = chance.map((item) => `${item.chance}% 概率`).join('、')
      }
      return { key, label: GROUP_LABELS[key] || key, items: groups[key], hint }
    })
})

// 里程碑 + 月度奖励合成一个列表展示（scope 区分是「连续」还是「当月」）
const ladderList = computed(() => [
  ...milestones.value.map((item) => ({ ...item, kind: 'streak', scope: '连续', icon: 'bi-gift' })),
  ...monthly.value.map((item) => ({ ...item, kind: 'month', scope: '当月签到', icon: 'bi-calendar-check' }))
])

// ---------- 数据加载 ----------
async function loadStatus() {
  try {
    const res = await getCheckinStatus()
    if (res.code === 200 && res.data) {
      Object.assign(status, {
        enabled: res.data.enabled !== false,
        name: res.data.name || '每日签到',
        checked: !!res.data.checked,
        check_in_time: res.data.check_in_time || 0,
        streak: res.data.streak || 0,
        preview_days: res.data.preview_days || 1,
        cycle_day: res.data.cycle_day || 0,
        cycle_days: res.data.cycle_days || [],
        month_days: res.data.month_days || 0,
        rewards: res.data.rewards || {},
        items: res.data.items || [],
        total: res.data.total || {},
        chance_items: res.data.chance_items || [],
        chance_total: res.data.chance_total || {},
        milestones: res.data.milestones || [],
        monthly: res.data.monthly || [],
        next_milestone: res.data.next_milestone || null,
        next_monthly: res.data.next_monthly || null,
        assets: res.data.assets || [],
        makeup: res.data.makeup || {},
        cards: res.data.cards || [],
        tips: res.data.tips || ''
      })
    } else if (res.code === 202) {
      // 功能关闭等提示
      status.enabled = false
    }
  } catch {
    // 静默：错误提示由请求拦截器统一给出
  }
}

async function loadCalendar() {
  try {
    const res = await getCheckinCalendar()
    if (res.code === 200 && res.data) {
      Object.assign(calendar, {
        year: res.data.year || calendar.year,
        month: res.data.month || calendar.month,
        days: res.data.days || [],
        total: res.data.total || 0,
        today: res.data.today || calendar.today
      })
    }
  } catch {
    // 静默
  }
}

async function loadRank() {
  rankLoading.value = true
  try {
    const res = await getCheckinRank(rangeParams())
    if (res.code === 200 && res.data) {
      rankList.value = Array.isArray(res.data.list) ? res.data.list : []
    }
  } catch {
    rankList.value = []
  } finally {
    rankLoading.value = false
  }
}

// 排行榜时间范围 → start / end 时间戳
function rangeParams() {
  const now = new Date()
  let start
  let end

  switch (timeRange.value) {
    case 'today':
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate())
      end = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59)
      break
    case 'week': {
      const day = now.getDay() || 7
      start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - day + 1)
      end = new Date(now.getFullYear(), now.getMonth(), now.getDate() - day + 7, 23, 59, 59)
      break
    }
    case 'month':
    default:
      start = new Date(now.getFullYear(), now.getMonth(), 1)
      end = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59)
      break
  }

  return {
    start: start ? Math.floor(start.getTime() / 1000) : undefined,
    end: end ? Math.floor(end.getTime() / 1000) : undefined
  }
}

function changeRange(key) {
  if (timeRange.value === key) return
  timeRange.value = key
  rankList.value = []
  loadRank()
}

function dayTitle(day) {
  if (!day.checked) return `${calendar.month} 月 ${day.day} 日 · 未签到`
  const parts = [`${calendar.month} 月 ${day.day} 日 · ${day.source === 2 ? '补签' : '已签到'}`]
  if (Number(day.exp) > 0) parts.push(`经验 +${day.exp}`)
  if (Number(day.integral) > 0) parts.push(`积分 +${day.integral}`)
  return parts.join('\n')
}

// 奖励明细 → 文案（「12 经验、5 积分、卡密 ×1」）
function itemsText(items) {
  const total = {}
  ;(items || []).forEach((item) => {
    const asset = item.asset || 'reward'
    // 卡密按张数统计（它的 value 是面额）
    total[asset] = (total[asset] || 0) + (asset === RewardCardKey ? 1 : Number(item.value || 0))
  })
  const parts = Object.entries(total)
    .filter(([, value]) => value !== 0)
    .map(([asset, value]) => (asset === RewardCardKey ? `卡密 ×${value}` : `${value} ${unitOf(asset)}`))
  return parts.join('、') || '奖励'
}

// ---------- 签到 / 补签 ----------
async function doSign() {
  if (loading.value || status.checked) return
  loading.value = true
  try {
    const res = await signIn()
    if (res.code === 200 && res.data) {
      const {
        streak, total, items, tips, cards,
        cycle_day: cycleDay, month_days: monthDays
      } = res.data

      const parts = [tips || '签到成功']
      if (items?.length || total) parts.push(`获得 ${itemsText(items)}`)

      // 拿到卡密时直接报出来，用户不用再去翻消息中心
      const codes = (cards || []).filter((item) => item.card).map((item) => item.card)
      if (codes.length) parts.push(`卡密 ${codes.join('、')}`)

      if (streak > 1) parts.push(`已连续 ${streak} 天`)
      if (cycleDay) parts.push(`周期第 ${cycleDay} 天`)

      // 里程碑 / 月度档位达成的反馈（明细里也有，这里提示得更显眼）
      const crossed = res.data.crossed || {}
      const achieved = []
      if ((crossed.streak || []).length) achieved.push(`连续 ${crossed.streak.join('、')} 天里程碑`)
      if ((crossed.monthly || []).length) achieved.push(`当月满 ${crossed.monthly.join('、')} 天奖励`)
      if (achieved.length) parts.push(`达成 ${achieved.join('、')}`)

      toast.success(parts.join('，'), codes.length ? 6000 : undefined)

      if (monthDays) status.month_days = monthDays
      if (cards) status.cards = cards
      loadStatus()
      loadCalendar()
      loadRank()
    } else if (res.code === 202) {
      toast.info(res.msg || '今日已签到')
      loadStatus()
    }
  } catch {
    // 错误已由拦截器提示
  } finally {
    loading.value = false
  }
}

function scrollToMakeup() {
  makeupRef.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

async function doMakeup() {
  if (!selectedMakeup.value || makeupLoading.value) return
  makeupLoading.value = true
  try {
    const res = await makeupSign(selectedMakeup.value)
    if (res.code === 200 && res.data) {
      const cost = res.data.cost?.integral || res.data.cost?.exp || makeup.value.cost || 0
      const parts = [
        `${res.data.date_text || '补签'}成功`,
        `获得 ${itemsText(res.data.items)}`,
        `消耗 ${cost} ${makeup.value.unit}`
      ]

      // 补签把断掉的连签补起来时，可能顺带达成里程碑 / 月度档位（后端按跨越口径补发）
      const crossed = res.data.crossed || {}
      const achieved = []
      if ((crossed.streak || []).length) achieved.push(`连续 ${crossed.streak.join('、')} 天里程碑`)
      if ((crossed.monthly || []).length) achieved.push(`当月满 ${crossed.monthly.join('、')} 天奖励`)
      if (achieved.length) parts.push(`并达成 ${achieved.join('、')}`)

      toast.success(parts.join('，'), achieved.length ? 5000 : undefined)
      selectedMakeup.value = 0
      loadStatus()
      loadCalendar()
      loadRank()
    } else if (res.code === 202) {
      toast.info(res.msg || '补签失败')
      loadStatus()
    }
  } catch {
    // 错误已由拦截器提示
  } finally {
    makeupLoading.value = false
  }
}

function loadAll() {
  if (!isLogged.value) return
  loadStatus()
  loadCalendar()
  loadRank()
}

onMounted(loadAll)

// 在本页登录成功（或其它标签页登录）后自动补加载，不必手动刷新
watch(isLogged, (logged) => {
  if (logged) loadAll()
})
</script>

<style scoped>
.checkin-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* ---------- 未登录 / 未开放 ---------- */
.state-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 56px 24px;
  text-align: center;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-xl);
}
.state-icon {
  width: 56px;
  height: 56px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  font-size: 24px;
  color: var(--primary);
  background: var(--accent-soft);
}
.state-title {
  margin: 4px 0 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.state-desc {
  max-width: 420px;
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-muted);
}

/* ---------- 主卡 ---------- */
.hero {
  position: relative;
  overflow: hidden;
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-sm);
}
/* 右上角柔光：只做氛围，不压住文字 */
.hero::after {
  content: '';
  position: absolute;
  top: -70px;
  right: -50px;
  width: 220px;
  height: 220px;
  border-radius: 50%;
  background: radial-gradient(circle, var(--accent-soft), transparent 70%);
  pointer-events: none;
}
.hero > * {
  position: relative;
  z-index: 1;
}
.hero.is-checked::after {
  background: radial-gradient(circle, rgba(90, 138, 58, 0.16), transparent 70%);
}

.hero-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.hero-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  font-size: 12px;
  font-weight: 500;
  color: var(--primary);
  background: var(--accent-soft);
  border-radius: 999px;
}
.hero-date {
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

.hero-main {
  display: flex;
  align-items: center;
  gap: 16px;
}

/* 环形进度：--ring 由模板注入（0~100） */
.hero-ring {
  width: 72px;
  height: 72px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: conic-gradient(var(--primary) var(--ring, 0%), var(--accent-soft) 0);
  transition: background 0.3s ease;
}
.hero.is-checked .hero-ring {
  background: conic-gradient(var(--success) 100%, var(--accent-soft) 0);
}
.hero-ring-core {
  width: 58px;
  height: 58px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  font-size: 20px;
  color: var(--primary);
  background: var(--bg-card);
}
.hero.is-checked .hero-ring-core {
  color: var(--success);
}

.hero-text {
  flex: 1;
  min-width: 0;
}
.hero-title {
  margin: 0;
  font-size: 19px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--text);
  letter-spacing: -0.2px;
}
.hero-sub {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-muted);
}
.hero-rewards {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
}
.hero-rewards-label {
  font-size: 12px;
  color: var(--text-muted);
}
.hero-variable {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  width: 100%;
  font-size: 12px;
  color: var(--text-muted);
}
.hero-variable i {
  color: var(--warning);
}

.hero-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.hero-btn {
  min-width: 108px;
  padding: 10px 20px;
  border-radius: 999px;
}
.hero-ghost {
  font-size: 13px;
  border-radius: 999px;
}

.hero-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
}
.hero-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 12px;
  border-right: 1px solid var(--border-soft);
}
.hero-stat:first-child {
  padding-left: 0;
}
.hero-stat:last-child {
  border-right: none;
  padding-right: 0;
}
.hero-stat-label {
  font-size: 12px;
  color: var(--text-muted);
}
.hero-stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  font-variant-numeric: tabular-nums;
  line-height: 1.3;
}
.hero-stat-value em {
  margin-left: 2px;
  font-style: normal;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}

/* ---------- 通用面板 ---------- */
.panel {
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}
.panel-title {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.panel-title i {
  font-size: 14px;
  color: var(--text-muted);
}
.panel-extra {
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.panel-note {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 0 0 10px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.panel-note i {
  margin-top: 2px;
}
.panel-loading {
  padding: 32px;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}

/* 通用小胶囊 */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
}
.chip i {
  font-size: 12px;
  color: var(--primary);
}
.chip-note {
  font-style: normal;
  font-size: 11px;
  color: var(--text-muted);
}

/* ---------- 奖励明细 ---------- */
.reward-group + .reward-group {
  margin-top: 14px;
}
.reward-group-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.reward-group-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.3px;
}
.reward-group-hint {
  font-size: 11px;
  color: var(--warning);
}
.reward-items {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* ---------- 卡密 ---------- */
.code-row {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 10px 12px;
  margin-bottom: 8px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.code-text {
  padding: 2px 0;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.6px;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  user-select: all;
}
.code-missing {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 周期 ---------- */
.cycle-track {
  display: flex;
  gap: 4px;
  overflow-x: auto;
  padding-bottom: 2px;
  /* 窄屏可横向滑动，但不显示滚动条（避免卡片内出现一条灰杠） */
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.cycle-track::-webkit-scrollbar {
  display: none;
}
.cycle-node {
  position: relative;
  flex: 1 0 84px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 4px;
  text-align: center;
}
/* 相邻节点之间的连线 */
.cycle-node::before {
  content: '';
  position: absolute;
  top: 19px;
  left: -2px;
  right: calc(50% + 14px);
  height: 1px;
  background: var(--border);
}
.cycle-node:first-child::before {
  display: none;
}
.cycle-node.is-done::before,
.cycle-node.is-current::before {
  background: var(--primary);
  opacity: 0.45;
}
.cycle-dot {
  position: relative;
  z-index: 1;
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 50%;
  font-variant-numeric: tabular-nums;
}
.cycle-node.is-done .cycle-dot {
  color: var(--primary);
  border-color: var(--accent-ring);
  background: var(--accent-soft);
}
.cycle-node.is-current .cycle-dot {
  color: #fff;
  border-color: transparent;
  background: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.cycle-reward {
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.cycle-node.is-done .cycle-reward {
  color: var(--text-muted);
  font-weight: 500;
}
.cycle-label {
  min-height: 14px;
  font-size: 11px;
  color: var(--text-muted);
}

/* ---------- 里程碑 ---------- */
.ms-progress {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.ms-progress-bar {
  flex: 1;
  height: 6px;
  border-radius: 999px;
  background: var(--bg-muted);
  overflow: hidden;
}
.ms-progress-fill {
  height: 100%;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--primary), var(--primary-deep));
  transition: width 0.3s ease;
}
.ms-progress-text {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

.ms-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ms-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 8px;
  border-radius: var(--radius);
  transition: background 0.15s;
}
.ms-row:hover {
  background: var(--bg-muted);
}
.ms-node {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  font-size: 13px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 50%;
}
.ms-row.is-reached .ms-node {
  color: var(--success);
  background: rgba(90, 138, 58, 0.12);
}
.ms-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.ms-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.ms-title em {
  margin-left: 6px;
  font-style: normal;
  font-size: 12px;
  font-weight: 400;
  color: var(--text-muted);
}
.ms-rewards {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.ms-rewards i {
  font-size: 11px;
  color: var(--primary);
}
.ms-state {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.ms-row.is-reached .ms-state {
  color: var(--success);
}

/* ---------- 日历 ---------- */
.cal-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 4px;
}
.cal-cell {
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  border-radius: 10px;
  font-variant-numeric: tabular-nums;
}
.cal-week {
  height: 28px;
  font-size: 12px;
  color: var(--text-muted);
}
.cal-blank {
  visibility: hidden;
}
.cal-day {
  color: var(--text-soft);
  border: 1px solid transparent;
  transition: background 0.15s;
}
.cal-day:hover {
  background: var(--bg-muted);
}
.cal-day.is-checked {
  color: var(--primary);
  font-weight: 600;
  background: var(--accent-soft);
}
/* 补签：虚框，与正常签到区分 */
.cal-day.is-makeup {
  color: var(--primary);
  background: transparent;
  border: 1px dashed var(--accent-ring);
}
.cal-day.is-today {
  color: var(--primary);
  font-weight: 600;
  border-color: var(--primary);
}
.cal-day.is-today.is-checked {
  color: #fff;
  background: var(--primary);
}

.cal-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-soft);
}
.cal-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-muted);
}
.cal-dot {
  width: 10px;
  height: 10px;
  border-radius: 3px;
  background: var(--bg-muted);
  border: 1px solid var(--border);
}
.cal-dot.is-checked {
  background: var(--accent-soft);
  border-color: var(--accent-ring);
}
.cal-dot.is-makeup {
  background: transparent;
  border: 1px dashed var(--primary);
}
.cal-dot.is-today {
  background: transparent;
  border-color: var(--primary);
}

/* ---------- 补签 ---------- */
.makeup-dates {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.makeup-date {
  padding: 6px 12px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s, background 0.15s;
}
.makeup-date:hover {
  color: var(--primary);
  border-color: var(--accent-ring);
}
.makeup-date.is-active {
  color: #fff;
  background: var(--primary);
  border-color: transparent;
}
.makeup-foot {
  margin-top: 14px;
}

/* ---------- 排行榜 ---------- */
.seg {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  background: var(--bg-muted);
  border-radius: 999px;
}
.seg-item {
  padding: 4px 12px;
  font-size: 12px;
  color: var(--text-muted);
  border-radius: 999px;
  transition: color 0.15s, background 0.15s;
}
.seg-item:hover {
  color: var(--text);
}
.seg-item.is-active {
  color: var(--text);
  font-weight: 600;
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
}

.rank-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}
.rank-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: var(--radius);
  transition: background 0.15s;
}
.rank-row:hover {
  background: var(--bg-muted);
}
.rank-row.is-me {
  background: var(--accent-wash);
}
.rank-badge {
  width: 24px;
  flex-shrink: 0;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.rank-badge.is-top-1 {
  color: #d9a441;
}
.rank-badge.is-top-2 {
  color: #9aa3ad;
}
.rank-badge.is-top-3 {
  color: #b8825a;
}
.rank-avatar {
  width: 38px;
  height: 38px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.rank-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rank-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rank-meta {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.rank-meta i {
  margin-right: 3px;
  font-size: 11px;
}

/* 主色按钮上的转圈：白边，避免深色底看不清 */
.spinner-light {
  width: 15px;
  height: 15px;
  border-color: rgba(255, 255, 255, 0.4);
  border-top-color: #fff;
}

/* ---------- 响应式 ---------- */
@media (max-width: 768px) {
  .hero {
    padding: 14px;
  }
  .hero-main {
    flex-wrap: wrap;
    gap: 12px;
  }
  .hero-ring {
    width: 60px;
    height: 60px;
  }
  .hero-ring-core {
    width: 48px;
    height: 48px;
    font-size: 17px;
  }
  .hero-title {
    font-size: 17px;
  }
  .hero-actions {
    width: 100%;
  }
  .hero-btn {
    flex: 1;
  }
  .hero-stats {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px 0;
  }
  .hero-stat {
    padding: 0 10px;
  }
  .hero-stat:nth-child(2n) {
    border-right: none;
    padding-right: 0;
  }
  .hero-stat:nth-child(2n + 1) {
    padding-left: 0;
  }
  .panel {
    padding: 14px;
  }
  .panel-head {
    flex-wrap: wrap;
  }
  .cal-cell {
    height: 34px;
    font-size: 12px;
  }
  .cycle-node {
    flex: 0 0 76px;
  }
}
</style>
