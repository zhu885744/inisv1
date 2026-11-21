<template>
  <div class="badge-wall">
    <section class="card card-pad">
      <header class="block-head">
        <h2 class="block-title"><i class="bi bi-award" /> 我的勋章</h2>
        <div class="head-actions">
          <span v-if="!loading" class="block-count">已获得 {{ heldCount }} / {{ list.length }}</span>
          <!-- 勋章馆入口：移动端底部 Tab 没有位置，从这里进公开的荣誉墙 -->
          <router-link to="/badges" class="hall-link">
            <i class="bi bi-grid-3x3-gap" /> 勋章馆
          </router-link>
        </div>
      </header>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState icon="bi bi-award" text="站点还没有配置勋章" />
      </div>

      <template v-else>
        <!-- 分类切换：全部 / 成长 / 荣誉 / 成就 / 限定（带「已获得/总数」） -->
        <div v-if="groups.length > 1" class="category-tabs">
          <button
            type="button"
            class="category-tab"
            :class="{ 'is-active': activeCat === null }"
            @click="activeCat = null"
          >
            <i class="bi bi-grid-3x3-gap" /> 全部
            <span class="category-tab-count">{{ heldCount }}/{{ list.length }}</span>
          </button>
          <button
            v-for="group in groups"
            :key="group.id"
            type="button"
            class="category-tab"
            :class="{ 'is-active': activeCat === group.id }"
            @click="activeCat = activeCat === group.id ? null : group.id"
          >
            <i :class="group.icon || 'bi bi-collection'" :style="{ color: group.color || undefined }" />
            {{ group.name }}
            <span class="category-tab-count">{{ group.held }}/{{ group.total }}</span>
          </button>
        </div>

        <!-- 按分类分组：每组先列已获得，再列进行中（含进度） -->
        <section v-for="group in visibleGroups" :key="group.id" class="category-group">
          <header class="category-head">
            <span class="category-icon" :style="groupTint(group)">
              <i :class="group.icon || 'bi bi-collection'" />
            </span>
            <div class="category-head-main">
              <h3 class="category-title">{{ group.name }}</h3>
              <p v-if="group.description" class="category-desc">{{ group.description }}</p>
            </div>
            <span class="category-progress">{{ group.held }}/{{ group.total }}</span>
          </header>

          <div v-if="!group.heldList.length" class="wall-hint">这一类还没有点亮的勋章</div>
          <div v-else class="held-grid">
            <div
              v-for="item in group.heldList"
              :key="item.id"
              class="held-item"
              :class="[`rarity-${item.rarity || 'common'}`, { 'is-expired': item.expired }]"
              :style="tint(item)"
            >
              <span class="held-icon">
                <img v-if="item.image" :src="item.image" :alt="item.name" loading="lazy" />
                <i v-else :class="item.icon || 'bi bi-award'" />
              </span>
              <div class="held-main">
                <strong class="held-name">
                  <span class="held-name-text" :title="item.name">{{ item.name }}</span>
                  <span v-if="item.expired" class="held-expired">已过期</span>
                </strong>
                <span class="held-desc" :title="item.description || item.condition_name">
                  {{ item.description || item.condition_name }}
                </span>
                <span class="held-time">
                  {{ fromNow(item.held_at) }}获得
                  <template v-if="expireText(item)">
                    · <span class="held-expire" :class="{ 'is-past': item.expired }">{{ expireText(item) }}</span>
                  </template>
                </span>
              </div>
              <!-- 操作按钮独占一行：两个按钮时不会再把文字列挤成竖排 -->
              <div v-if="Number(item.expire_time) > 0 || !item.expired" class="held-actions">
                <!-- 有限期 + 后台给这枚勋章配了续期档位，才显示续期入口（永久勋章后端也会拒绝） -->
                <button
                  v-if="Number(item.expire_time) > 0 && item.renew_options?.length"
                  type="button"
                  class="wear-btn renew-btn"
                  title="续期"
                  @click="openRenew(item)"
                >
                  <i class="bi bi-arrow-repeat" />
                  续期
                </button>

                <button
                  v-if="!item.expired"
                  type="button"
                  class="wear-btn"
                  :class="{ 'is-active': wearingId === item.id }"
                  :disabled="wearingBusy === item.id"
                  :title="wearingId === item.id ? '点击卸下' : '佩戴到昵称旁'"
                  @click="toggleWear(item)"
                >
                  <i :class="wearingId === item.id ? 'bi bi-pin-angle-fill' : 'bi bi-pin-angle'" />
                  {{ wearingId === item.id ? '佩戴中' : '佩戴' }}
                </button>
              </div>
            </div>
          </div>

          <ul v-if="group.todoList.length" class="todo-list">
            <li v-for="item in group.todoList" :key="item.id" class="todo-item">
              <span class="todo-icon">
                <i :class="item.icon || 'bi bi-award'" />
              </span>
              <div class="todo-main">
                <div class="todo-head">
                  <strong class="todo-name">{{ item.name }}</strong>
                  <span class="todo-progress-text">
                    {{ item.available ? progressText(item) : windowText(item) }}
                  </span>
                </div>
                <p class="todo-desc">{{ item.description || conditionText(item) }}</p>
                <div v-if="item.available" class="progress">
                  <span class="progress-fill" :style="{ width: percent(item) }" />
                </div>
                <p v-else class="todo-locked">
                  <i class="bi bi-hourglass-split" /> {{ windowText(item) }}
                </p>
                <p v-if="item.available && item.end_time" class="todo-limit">
                  <i class="bi bi-alarm" /> {{ dateText(item.end_time) }} 截止
                </p>
              </div>
            </li>
          </ul>
        </section>

        <div v-if="!visibleGroups.length" class="wall-hint">这个分类下暂时没有勋章</div>
      </template>
    </section>

    <!-- 获得动效：只有本次访问真的新判定了勋章才出现（后端 mine 返回 granted） -->
    <Transition name="reward">
      <div v-if="showReward && newBadges.length" class="reward-mask" @click.self="closeReward">
        <div class="reward-card">
          <span class="reward-kicker"><i class="bi bi-stars" /> 获得新勋章</span>
          <div class="reward-list">
            <div v-for="item in newBadges" :key="item.id" class="reward-item">
              <span class="reward-icon" :class="`rarity-${item.rarity || 'common'}`" :style="rewardStyle(item)">
                <img v-if="item.image" :src="item.image" :alt="item.name" loading="lazy" />
                <i v-else :class="item.icon || 'bi bi-award'" />
              </span>
              <strong class="reward-name">{{ item.name }}</strong>
              <span class="reward-rarity">{{ item.rarity_name || '普通' }}</span>
            </div>
          </div>
          <button class="btn btn-primary btn-sm" type="button" @click="closeReward">收下</button>
        </div>
      </div>
    </Transition>

    <!-- 续期：价格与档位是这枚勋章自己的（后台在勋章编辑里配置） -->
    <RenewPlanDialog
      :visible="renew.visible"
      scope="badge"
      :target-id="renew.id"
      :target-name="renew.name"
      :current-expire="renew.expire"
      :options="renew.options"
      @update:visible="(v) => (renew.visible = v)"
      @done="load"
    />
  </div>
</template>

<script setup>
/**
 * 我的勋章（/user/badges）
 *
 * 数据来自 badge/mine：返回全部启用勋章 + 是否获得 + 当前进度。
 * 已获得按获得时间倒序，未获得的按后台排序（sort desc），进度条用 current / target。
 * 手动发放的勋章没有进度概念，未获得时只显示「手动发放」。
 */
import { ref, computed, onMounted, onUnmounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import RenewPlanDialog from '@/components/RenewPlanDialog.vue'
import { getMyBadges, wearBadge, unwearBadge } from '@/api/badge'
import { formatDate, fromNow } from '@/utils/time'
import { toast } from '@/utils/toast'

const loading = ref(false)
const list = ref([])
// 分类分组：后端 badge/mine 一并返回（已按 sort 排好、空分类已过滤）
const categories = ref([])
// 当前选中的分类，null = 全部
const activeCat = ref(null)
// 当前佩戴的勋章 id（后端 users.json.wearing_badge，mine 里一并返回）
const wearingId = ref(0)
// 正在提交佩戴的勋章 id（避免连点）
const wearingBusy = ref(0)
// 续期弹层：只对「有限期 + 配了续期档位」的勋章开放（永久勋章后端也会拒绝）
const renew = ref({ visible: false, id: 0, name: '', expire: 0, options: [] })

function openRenew(item) {
  renew.value = {
    visible: true,
    id: Number(item.id) || 0,
    name: item.name || '',
    expire: Number(item.expire_time) || 0,
    options: item.renew_options || []
  }
}
// 获得动效：本次访问新判定发放的勋章（后端 mine 的 granted，通常是空数组）
const newBadges = ref([])
const showReward = ref(false)
let rewardTimer = null

const heldCount = computed(() => list.value.filter((item) => item.held).length)

// 分类 → 该分类下的勋章（含「已获得 / 进行中」两个子集）
const groups = computed(() => {
  const meta = new Map(categories.value.map((cat) => [Number(cat.id), cat]))
  const known = new Set(categories.value.map((cat) => Number(cat.id)))
  // 分类被停用 / 删除后，里面的勋章退到「未分类」：
  // 分类可以不在前台露出，但用户已经获得的勋章不能凭空消失
  const normalize = (id) => {
    const cid = Number(id) || 0
    return known.has(cid) ? cid : 0
  }

  const buckets = new Map()
  for (const item of list.value) {
    const cid = normalize(item.category_id)
    if (!buckets.has(cid)) buckets.set(cid, [])
    buckets.get(cid).push(item)
  }

  // 顺序以后端 categories 为准；「未分类」永远排最后
  const order = categories.value.map((cat) => Number(cat.id))
  if (buckets.has(0)) order.push(0)

  const result = []
  for (const cid of order) {
    const items = buckets.get(cid)
    if (!items || !items.length) continue
    const cat = meta.get(cid) || {
      id: 0,
      name: '未分类',
      description: '',
      icon: 'bi bi-collection',
      color: ''
    }
    result.push({
      ...cat,
      total: items.length,
      held: items.filter((item) => item.held).length,
      heldList: items.filter((item) => item.held),
      todoList: items.filter((item) => !item.held)
    })
  }

  return result
})

const visibleGroups = computed(() =>
  activeCat.value === null
    ? groups.value
    : groups.value.filter((group) => Number(group.id) === Number(activeCat.value))
)

async function load() {
  loading.value = true
  try {
    const res = await getMyBadges()
    list.value = res.data?.data || []
    categories.value = res.data?.categories || []
    wearingId.value = Number(res.data?.wearing?.id) || 0

    // 只有「进来之前还没拿到」的勋章才会出现在 granted 里 —— 天然不会重复播放
    const granted = res.data?.granted || []
    if (granted.length) {
      newBadges.value = granted
      showReward.value = true
      if (rewardTimer) clearTimeout(rewardTimer)
      rewardTimer = setTimeout(closeReward, 6000)
    }
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    loading.value = false
  }
}

// 佩戴 / 卸下：同时只能戴一枚，成功后本地即时更新，不用整页刷新
async function toggleWear(item) {
  if (!item?.held || wearingBusy.value) return
  wearingBusy.value = item.id
  try {
    if (wearingId.value === item.id) {
      await unwearBadge()
      wearingId.value = 0
      toast.success('已卸下')
    } else {
      const res = await wearBadge(item.id)
      wearingId.value = Number(res.data?.wearing?.id) || item.id
      toast.success(`已佩戴「${item.name}」`)
    }
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    wearingBusy.value = 0
  }
}

/* ---------------- 限时 / 获得动效 ---------------- */

function dateText(unix) {
  const date = new Date(Number(unix) * 1000)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

/**
 * 勋章有效期文案（expire_time 为发放时算好的 unix 秒，0 = 永久）
 *
 *   永久勋章           → ''（不显示，绝大多数勋章走这条）
 *   距到期还有 30 天   → 「2026-12-01 到期（剩 12 天）」
 *   距到期更远         → 「2026-12-01 到期」
 *   已过期             → 「2026-11-01 到期」（名称旁已有「已过期」标签，不再重复写）
 *
 * 剩余天数只在 30 天内提示：每枚勋章都挂一串数字反而看不出重点。
 */
function expireText(item) {
  const expire = Number(item.expire_time) || 0
  if (expire <= 0) return ''

  const date = formatDate(expire)
  if (item.expired) return `${date} 到期`

  const days = Math.ceil((expire - Date.now() / 1000) / 86400)
  if (days > 0 && days <= 30) return `${date} 到期（剩 ${days} 天）`
  return `${date} 到期`
}

// 现在拿不到时的原因：还没开始 / 已经结束 / 已过期
function windowText(item) {
  const now = Math.floor(Date.now() / 1000)
  if (item.start_time && now < Number(item.start_time)) return `${dateText(item.start_time)} 开启`
  if (item.end_time && now > Number(item.end_time)) return '限时已结束'
  if (item.expired) return '已过期'
  return '暂不可获得'
}

function rewardStyle(item) {
  const color = item.color || 'var(--warning)'
  const hex = /^#[0-9a-f]{6}$/i.test(color)
  return {
    color,
    borderColor: hex ? `${color}59` : 'var(--border)',
    background: hex ? `${color}1f` : 'var(--bg-muted)'
  }
}

function closeReward() {
  showReward.value = false
  if (rewardTimer) {
    clearTimeout(rewardTimer)
    rewardTimer = null
  }
}

onUnmounted(() => {
  if (rewardTimer) clearTimeout(rewardTimer)
})

// 主题色 → 半透明边框 / 背景（8 位十六进制，避免依赖 color-mix）
function tint(item) {
  const color = item.color || 'var(--warning)'
  const alpha = (value, hex) => (/^#[0-9a-f]{6}$/i.test(value) ? `${value}${hex}` : 'var(--border)')
  return {
    borderColor: alpha(color, '59'),
    background: alpha(color, '14')
  }
}

// 分类主题色 → 图标底色（分类没配颜色时用中性底色）
function groupTint(group) {
  const color = group.color || 'var(--primary)'
  const background = /^#[0-9a-f]{6}$/i.test(group.color || '') ? `${group.color}1f` : 'var(--bg-muted)'
  return { color, background }
}

function percent(item) {
  const target = Number(item.target) || 0
  if (!target) return '0%'
  const current = Math.min(Number(item.current) || 0, target)
  return `${Math.round((current / target) * 100)}%`
}

function progressText(item) {
  if (item.condition === 'manual') return '待发放'
  return `${Math.min(Number(item.current) || 0, Number(item.target) || 0)} / ${item.target}${item.condition_unit || ''}`
}

function conditionText(item) {
  if (item.condition === 'manual') return '由管理员手动发放'
  return `${item.condition_name}达到 ${item.target}${item.condition_unit || ''} 后自动获得`
}

onMounted(load)
</script>

<style scoped>
.badge-wall {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
/* .is-loading / .empty-row 同样是各页面自带（不是全局样式），这里必须补一份 */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}
.block-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 15px;
}
.block-count {
  font-size: 12px;
  color: var(--text-muted);
}
.wall-section + .wall-section {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px dashed var(--border-soft);
}
.wall-title {
  margin: 0 0 10px;
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 600;
}
.wall-hint {
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 分类 ---------- */
.head-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.hall-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 999px;
  transition: color 0.15s, background 0.15s;
}
.hall-link:hover {
  color: var(--primary-deep);
  background: var(--accent-wash);
}

.category-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
  overflow-x: auto;
  scrollbar-width: none;
}
.category-tabs::-webkit-scrollbar {
  display: none;
}
.category-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
  padding: 5px 12px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.category-tab:hover {
  color: var(--text);
}
.category-tab.is-active {
  color: var(--primary-deep);
  background: var(--accent-wash);
  border-color: var(--primary);
}
.category-tab-count {
  font-size: 11px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}
.category-tab.is-active .category-tab-count {
  color: var(--primary-deep);
}
.category-group + .category-group {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px dashed var(--border-soft);
}
.category-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}
.category-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 9px;
  font-size: 15px;
}
.category-head-main {
  flex: 1;
  min-width: 0;
}
.category-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.category-desc {
  margin: 2px 0 0;
  font-size: 11px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.category-progress {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
/* 组内的「进行中」与上面的已获得留出间距 */
.category-group .todo-list {
  margin-top: 10px;
}

/* ---------- 稀有度 / 过期 ---------- */
.held-item.rarity-rare {
  box-shadow: inset 0 0 0 1px rgba(79, 157, 255, 0.45);
}
.held-item.rarity-epic {
  box-shadow: inset 0 0 0 1px rgba(160, 107, 255, 0.5);
}
.held-item.rarity-legendary {
  box-shadow: inset 0 0 0 1px rgba(240, 180, 41, 0.6), 0 0 10px rgba(240, 180, 41, 0.2);
}
.held-item.is-expired {
  opacity: 0.72;
}
.held-expired {
  flex-shrink: 0;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 500;
  color: var(--text-light);
  border: 1px solid var(--border);
  border-radius: 4px;
}
.todo-locked {
  margin: 4px 0 0;
  font-size: 11px;
  color: var(--warning, #e08b00);
}
.todo-limit {
  margin: 4px 0 0;
  font-size: 11px;
  color: var(--text-light);
}

/* ---------- 获得动效 ---------- */
.reward-mask {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.45);
}
.reward-card {
  width: 100%;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 22px 18px 18px;
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-md);
  text-align: center;
}
.reward-kicker {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  letter-spacing: 0.04em;
  color: var(--text-muted);
}
.reward-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.reward-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
}
.reward-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 66px;
  height: 66px;
  border: 1px solid var(--border);
  border-radius: 50%;
  font-size: 30px;
  overflow: hidden;
  animation: reward-pop 0.55s cubic-bezier(0.2, 1.3, 0.4, 1) both;
}
.reward-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.reward-icon.rarity-rare {
  box-shadow: 0 0 0 2px rgba(79, 157, 255, 0.6);
}
.reward-icon.rarity-epic {
  box-shadow: 0 0 0 2px rgba(160, 107, 255, 0.7), 0 0 16px rgba(160, 107, 255, 0.4);
}
.reward-icon.rarity-legendary {
  box-shadow: 0 0 0 2px #f0b429, 0 0 22px rgba(240, 180, 41, 0.65);
  animation: reward-pop 0.55s cubic-bezier(0.2, 1.3, 0.4, 1) both,
    reward-halo 2s ease-in-out infinite 0.55s;
}
.reward-name {
  font-size: 14px;
}
.reward-rarity {
  font-size: 11px;
  color: var(--text-muted);
}

@keyframes reward-pop {
  from {
    transform: scale(0.4);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}
@keyframes reward-halo {
  0%,
  100% {
    box-shadow: 0 0 0 2px #f0b429, 0 0 12px rgba(240, 180, 41, 0.45);
  }
  50% {
    box-shadow: 0 0 0 2px #ffd166, 0 0 26px rgba(240, 180, 41, 0.8);
  }
}
.reward-enter-active,
.reward-leave-active {
  transition: opacity 0.25s ease;
}
.reward-enter-from,
.reward-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .reward-icon,
  .reward-icon.rarity-legendary {
    animation: none;
  }
}
/* 列宽下限要容得下「图标 38 + 间距 + 文本」，「佩戴/续期」按钮已挪到卡片底部独占一行 */
.held-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 10px;
}
.held-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  column-gap: 10px;
  row-gap: 8px;
  align-content: space-between;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.15s;
}
.held-item:hover {
  border-color: var(--primary);
}
.held-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  margin-top: 1px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: var(--bg-card);
  font-size: 18px;
  overflow: hidden;
}
.held-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.held-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
/* 名称行：名字过长时省略，「已过期」标签不许被压缩 */
.held-name {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 13px;
  line-height: 1.4;
}
.held-name-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.held-desc {
  font-size: 11px;
  line-height: 1.4;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.held-time {
  font-size: 11px;
  line-height: 1.45;
  color: var(--text-light);
}
/* 有效期：比「获得时间」略强一点，让到期日更好扫；已过期的弱化处理
   （结论由名称旁的「已过期」标签负责，这里只留日期） */
.held-expire {
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.held-expire.is-past {
  color: var(--text-light);
}
/* 操作区：横跨卡片底部一行，两个按钮平分宽度 */
.held-actions {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 6px;
  padding-top: 8px;
  border-top: 1px dashed var(--border-soft);
}
/* 佩戴按钮：佩戴中高亮 */
.wear-btn {
  flex: 1 1 0;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  height: 26px;
  padding: 0 8px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.wear-btn:hover:not(:disabled) {
  color: var(--primary-deep);
  border-color: var(--primary);
}
.wear-btn.is-active {
  color: var(--primary-deep);
  background: var(--accent-wash);
  border-color: var(--primary);
}
.wear-btn:disabled {
  opacity: 0.6;
  cursor: default;
}
/* 续期按钮：用主色底区分「佩戴」，过期后这是唯一能救回这枚勋章的入口 */
.renew-btn {
  color: var(--primary-deep);
  background: var(--accent-wash);
}
.todo-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.todo-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-muted);
}
.todo-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 1px dashed var(--border-soft);
  color: var(--text-light);
  font-size: 15px;
}
.todo-main {
  flex: 1;
  min-width: 0;
}
.todo-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.todo-name {
  font-size: 13px;
}
.todo-progress-text {
  font-size: 11px;
  color: var(--text-muted);
}
.todo-desc {
  margin: 2px 0 6px;
  font-size: 11px;
  color: var(--text-muted);
}
.progress {
  height: 5px;
  border-radius: 999px;
  background: var(--border);
  overflow: hidden;
}
.progress-fill {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--primary);
  transition: width var(--dur) var(--ease);
}
</style>
