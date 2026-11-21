<template>
  <Teleport to="body">
    <Transition name="goods-detail-fade">
      <div v-if="visible" class="goods-detail-mask" @click.self="close">
        <div class="goods-detail" role="dialog" aria-modal="true" :aria-label="view.title || '商品详情'">
          <button type="button" class="goods-detail-close" title="关闭" aria-label="关闭" @click="close">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>

          <div class="goods-detail-scroll">
            <!-- 封面 -->
            <div class="goods-detail-cover">
              <img :src="view.cover || defaultCover" :alt="view.title" loading="lazy" @error="onCoverError" />
              <span class="goods-detail-type" :class="isPhysical ? 'is-physical' : 'is-virtual'">
                <i :class="isPhysical ? 'bi bi-box-seam' : 'bi bi-lightning-charge'" aria-hidden="true" />
                {{ isPhysical ? '实物商品' : '虚拟商品' }}
              </span>
              <span v-if="stateText" class="goods-detail-state" :class="stateType">
                <i :class="stateIcon" aria-hidden="true" /> {{ stateText }}
              </span>
            </div>

            <div class="goods-detail-main">
              <h2 class="goods-detail-title">{{ view.title || '商品详情' }}</h2>

              <div class="goods-detail-price">
                <span class="goods-detail-coin" :class="{ 'is-free': isFree }">
                  <i :class="isFree ? 'bi bi-gift' : 'bi bi-coin'" aria-hidden="true" />
                  <strong v-if="isFree">免费</strong>
                  <template v-else>
                    <strong>{{ view.price ?? '—' }}</strong>
                    <em>积分</em>
                  </template>
                </span>
                <span class="goods-detail-stock" :class="{ 'is-empty': Number(view.stock) <= 0 }">
                  {{ Number(view.stock) > 0 ? `库存 ${view.stock}` : '已售罄' }}
                </span>
              </div>

              <div class="goods-detail-tags">
                <span v-if="Number(view.sold) > 0" class="goods-detail-tag">已兑 {{ view.sold }} 件</span>
                <span v-if="Number(view.limit_per_user) > 0" class="goods-detail-tag">
                  每人限购 {{ view.limit_per_user }} 件<template v-if="Number(view.limit_remain) >= 0"> · 还可兑 {{ view.limit_remain }}</template>
                </span>
                <span v-if="Number(view.min_exp) > 0" class="goods-detail-tag">需经验达到 {{ view.min_exp }}</span>
                <span v-if="hasDuration" class="goods-detail-tag is-gold">
                  <i class="bi bi-clock-history" aria-hidden="true" /> 有效期 {{ formatDuration(view.decoration_duration) }}
                </span>
                <span v-if="Number(view.my_bought) > 0" class="goods-detail-tag is-bought">你已兑换 {{ view.my_bought }} 次</span>
              </div>

              <!-- 倒计时 / 时间窗 -->
              <p v-if="countdownText" class="goods-detail-countdown" :class="{ 'is-urgent': countdownUrgent }">
                <i class="bi bi-hourglass-split" aria-hidden="true" /> {{ countdownText }}
              </p>
              <p v-else-if="timeRangeText" class="goods-detail-countdown">
                <i class="bi bi-calendar-event" aria-hidden="true" /> {{ timeRangeText }}
              </p>

              <!-- 详情 -->
              <section class="goods-detail-section">
                <h3 class="goods-detail-subtitle">商品详情</h3>
                <p v-if="detailLoading" class="goods-detail-loading"><span class="spinner" /> 加载详情...</p>
                <p v-else class="goods-detail-desc" :class="{ 'is-empty': !view.description }">
                  {{ view.description || '作者还没有填写商品描述' }}
                </p>
              </section>

              <!-- 兑换说明 -->
              <section class="goods-detail-section">
                <h3 class="goods-detail-subtitle">兑换说明</h3>
                <ul class="goods-detail-notice">
                  <li v-if="isFree">该商品为免费兑换，不消耗积分，提交后即可领取。</li>
                  <li v-else>兑换后立即扣除积分，虚拟商品自动发放，不支持无理由退还。</li>
                  <li v-if="isPhysical">实物商品需填写收货人、电话与详细地址，发货进度可在「我的兑换记录」查看。</li>
                  <li v-else>兑换成功后请及时复制并妥善保管发放内容。</li>
                  <li v-if="Number(view.limit_per_user) > 0">该商品每人限兑 {{ view.limit_per_user }} 件，取消订单会返还积分并回滚限购次数。</li>
                  <li v-if="Number(view.min_exp) > 0">经验不足 {{ view.min_exp }} 时无法兑换，可通过发布内容、签到提升经验。</li>
                </ul>
              </section>
            </div>
          </div>

          <!-- 底部兑换栏 -->
          <footer class="goods-detail-foot">
            <div class="goods-detail-balance">
              <template v-if="isLogged">
                <span class="goods-detail-balance-label">我的积分</span>
                <span class="goods-detail-balance-num">{{ balance }}</span>
                <span v-if="shortage > 0" class="goods-detail-shortage">还差 {{ shortage }}</span>
              </template>
              <span v-else class="goods-detail-balance-label">登录后即可用积分兑换</span>
            </div>
            <button
              type="button"
              class="btn btn-primary goods-detail-buy"
              :disabled="buyDisabled"
              @click="onBuy"
            >
              <span v-if="buying" class="spinner" />
              {{ buyText }}
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * 积分商品详情弹窗
 *
 * 列表页点「查看详情」打开：展示封面、价格、库存、限购 / 经验门槛 / 有效期、
 * 兑换时间窗与详情描述，底部固定「兑换」按钮，兑换流程仍由列表页统一处理
 * （虚拟商品确认弹窗、实物商品收货地址、兑换成功后的发放内容）。
 *
 * 打开时会用 goods/one 拉一次最新数据，保证 can_buy / limit_remain / stock
 * 与后端一致（列表可能是几分钟前加载的）。
 */
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getGoodsOne } from '@/api/goods'
import { formatCountdown, remainSeconds } from '@/utils/countdown'
import { formatDuration, formatTime } from '@/utils/time'

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 列表里已经拿到的商品（含倒计时装饰字段 _state 等），作为详情加载前的兜底
  goods: { type: Object, default: null },
  balance: { type: Number, default: 0 },
  isLogged: { type: Boolean, default: false },
  // 是否正在兑换当前商品（父级 buyingId === goods.id）
  buying: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'buy'])

const router = useRouter()

const defaultCover =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect width="120" height="120" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="28" fill="%238a8a82" font-family="serif">兑</text></svg>'

const detail = ref(null)
const detailLoading = ref(false)

/** 列表数据 + 详情数据：详情优先，倒计时等装饰字段由列表提供 */
const view = computed(() => ({ ...(props.goods || {}), ...(detail.value || {}) }))

const isPhysical = computed(() => view.value.type === 'physical')
// 0 积分商品按「免费」展示，不再出现「0 积分」「使用 0 积分兑换」
const isFree = computed(() => Number(view.value.price) <= 0)
const hasDuration = computed(
  () => view.value.decoration_duration !== undefined && view.value.decoration_duration !== null
)

function onCoverError(e) {
  e.target.src = defaultCover
}

/* ===== 倒计时 ===== */
const nowSec = ref(Math.floor(Date.now() / 1000))
let timer = null
// 1 小时内高亮（与列表页保持一致）
const ENDING_SOON = 3600

function startTimer() {
  if (timer) return
  timer = setInterval(() => {
    nowSec.value = Math.floor(Date.now() / 1000)
  }, 1000)
}

function stopTimer() {
  if (!timer) return
  clearInterval(timer)
  timer = null
}

const startLeft = computed(() => remainSeconds(view.value.start_time, nowSec.value * 1000))
const endLeft = computed(() => remainSeconds(view.value.end_time, nowSec.value * 1000))
const hasEndTime = computed(() => Number(view.value.end_time) > 0)
const ended = computed(() => hasEndTime.value && endLeft.value <= 0)

const countdownText = computed(() => {
  if (startLeft.value > 0) return `距开抢还有 ${formatCountdown(startLeft.value)}`
  if (hasEndTime.value && endLeft.value > 0) return `距结束还有 ${formatCountdown(endLeft.value)}`
  return ''
})
const countdownUrgent = computed(() => startLeft.value === 0 && endLeft.value > 0 && endLeft.value <= ENDING_SOON)

const timeRangeText = computed(() => {
  const start = Number(view.value.start_time) > 0 ? formatTime(view.value.start_time) : ''
  const end = hasEndTime.value ? formatTime(view.value.end_time) : ''
  if (start && end) return `兑换时间：${start} ~ ${end}`
  if (start) return `开抢时间：${start}`
  if (end) return `结束时间：${end}`
  return ''
})

const stateText = computed(() => {
  if (startLeft.value > 0) return '未开始'
  if (ended.value) return '已结束'
  if (endLeft.value > 0 && endLeft.value <= ENDING_SOON) return '即将结束'
  return ''
})
const stateType = computed(() => {
  if (startLeft.value > 0) return 'is-soon'
  if (ended.value) return 'is-end'
  return 'is-ending'
})
const stateIcon = computed(() => {
  if (startLeft.value > 0) return 'bi bi-clock'
  if (ended.value) return 'bi bi-slash-circle'
  return 'bi bi-hourglass-split'
})

/* ===== 兑换态 ===== */
const limitUsedUp = computed(
  () => Number(view.value.limit_per_user) > 0 && Number(view.value.limit_remain) === 0
)

const canBuy = computed(() => {
  const v = view.value
  if (!v || !v.id) return false
  if (v.can_buy === false) return false
  if (startLeft.value > 0 || ended.value) return false
  if (Number(v.stock) <= 0) return false
  if (limitUsedUp.value) return false
  return true
})

const buyReason = computed(() => {
  const v = view.value
  if (v.buy_reason) return v.buy_reason
  if (Number(v.stock) <= 0) return '已售罄'
  if (startLeft.value > 0) return '活动未开始'
  if (ended.value) return '活动已结束'
  if (limitUsedUp.value) return '已达限购上限'
  return '暂不可兑换'
})

const buyText = computed(() => {
  if (!props.isLogged) return '登录后兑换'
  if (props.buying) return '兑换中...'
  if (!canBuy.value) return buyReason.value
  return isFree.value ? '免费兑换' : `使用 ${view.value.price ?? 0} 积分兑换`
})

const buyDisabled = computed(() => props.isLogged && (!canBuy.value || props.buying))

// 余额不足以兑换时在底部提示差额，避免点下去才被后端拒绝
const shortage = computed(() => {
  const price = Number(view.value.price) || 0
  if (!props.isLogged || price <= 0) return 0
  return Math.max(0, price - (Number(props.balance) || 0))
})

function onBuy() {
  if (!props.isLogged) {
    router.push('/auth/login')
    return
  }
  if (!canBuy.value || props.buying) return
  emit('buy', view.value)
}

/* ===== 开关 ===== */
let bodyOverflow = ''

function lockScroll() {
  bodyOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
}

function restoreScroll() {
  document.body.style.overflow = bodyOverflow || ''
}

function close() {
  emit('update:visible', false)
}

async function loadDetail() {
  const id = Number(props.goods?.id) || 0
  if (!id) return
  detailLoading.value = true
  try {
    const res = await getGoodsOne(id)
    detail.value = res?.data && typeof res.data === 'object' ? res.data : null
  } catch {
    // 详情接口失败时退回列表数据，不打断兑换
    detail.value = null
  } finally {
    detailLoading.value = false
  }
}

watch(
  () => props.visible,
  (open) => {
    if (!open) {
      stopTimer()
      restoreScroll()
      detail.value = null
      return
    }
    nowSec.value = Math.floor(Date.now() / 1000)
    startTimer()
    lockScroll()
    loadDetail()
  }
)

function onKeydown(e) {
  if (e.key === 'Escape' && props.visible) close()
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  stopTimer()
  restoreScroll()
})
</script>

<style scoped>
.goods-detail-mask {
  position: fixed;
  inset: 0;
  /* 低于确认弹窗（3600）与发货内容弹窗，保证兑换流程浮在其上 */
  z-index: 3200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(32, 28, 22, 0.55);
  backdrop-filter: blur(2px);
}

.goods-detail {
  position: relative;
  width: 100%;
  max-width: 680px;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.goods-detail-close {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 3;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 50%;
  background: rgba(32, 28, 22, 0.48);
  color: #fff;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}
.goods-detail-close:hover {
  background: rgba(32, 28, 22, 0.72);
}

.goods-detail-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* ---------- 封面 ---------- */
.goods-detail-cover {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  background: var(--bg-muted);
}
.goods-detail-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.goods-detail-cover::after {
  content: '';
  position: absolute;
  inset: auto 0 0 0;
  height: 56%;
  background: linear-gradient(180deg, transparent, rgba(32, 28, 22, 0.55));
  pointer-events: none;
}

.goods-detail-type,
.goods-detail-state {
  position: absolute;
  top: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  line-height: 1.7;
  color: #fff;
  z-index: 2;
}
.goods-detail-type {
  left: 12px;
  background: rgba(32, 28, 22, 0.55);
}
.goods-detail-type.is-virtual {
  background: rgba(46, 122, 92, 0.9);
}
.goods-detail-type.is-physical {
  background: rgba(199, 72, 42, 0.9);
}
.goods-detail-state {
  left: 12px;
  bottom: 12px;
  top: auto;
}
.goods-detail-state.is-soon {
  background: var(--warning);
}
.goods-detail-state.is-ending {
  background: var(--danger);
}
.goods-detail-state.is-end {
  background: rgba(32, 28, 22, 0.72);
}

/* ---------- 主体 ---------- */
.goods-detail-main {
  padding: 18px 22px 22px;
}
.goods-detail-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 20px;
  font-weight: 900;
  line-height: 1.5;
  color: var(--text);
}

.goods-detail-price {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
}
.goods-detail-coin {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  color: #d4a148;
  font-variant-numeric: tabular-nums;
}
.goods-detail-coin .bi {
  font-size: 15px;
}
.goods-detail-coin strong {
  font-size: 26px;
  font-weight: 800;
  line-height: 1.2;
}
.goods-detail-coin em {
  font-style: normal;
  font-size: 12px;
  color: var(--text-muted);
}
/* 0 积分商品：用「免费」代替 0 积分 */
.goods-detail-coin.is-free {
  color: var(--success);
}
.goods-detail-coin.is-free strong {
  font-size: 22px;
  color: var(--success);
  letter-spacing: 0.02em;
}
.goods-detail-stock {
  font-size: 12px;
  color: var(--text-muted);
}
.goods-detail-stock.is-empty {
  color: var(--danger);
}

.goods-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
.goods-detail-tag {
  padding: 2px 9px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.7;
}
.goods-detail-tag.is-gold {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--gold-wash);
  color: #c7902f;
}
.goods-detail-tag.is-bought {
  background: rgba(46, 122, 92, 0.12);
  color: var(--success);
}

.goods-detail-countdown {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 12px 0 0;
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  background: var(--gold-wash);
  color: #b5822a;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.goods-detail-countdown.is-urgent {
  background: var(--accent-soft);
  color: var(--danger);
}

/* ---------- 分区 ---------- */
.goods-detail-section {
  margin-top: 18px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft);
}
.goods-detail-subtitle {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
  letter-spacing: 0.02em;
}
.goods-detail-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.8;
  color: var(--text-soft);
  white-space: pre-wrap;
  word-break: break-word;
}
.goods-detail-desc.is-empty {
  color: var(--text-light);
}
.goods-detail-loading {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}
.goods-detail-notice {
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}

/* ---------- 底部操作 ---------- */
.goods-detail-foot {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 20px;
  background: var(--bg-card);
  border-top: 1px solid var(--border-soft);
}
.goods-detail-balance {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  font-size: 12px;
  color: var(--text-muted);
}
.goods-detail-balance-num {
  font-size: 16px;
  font-weight: 700;
  color: #d4a148;
  font-variant-numeric: tabular-nums;
}
.goods-detail-shortage {
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--danger);
  font-size: 11px;
}
.goods-detail-buy {
  flex: 0 0 auto;
  min-width: 168px;
  height: var(--control-h);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

/* ---------- 过渡 ---------- */
.goods-detail-fade-enter-active,
.goods-detail-fade-leave-active {
  transition: opacity 0.2s ease;
}
.goods-detail-fade-enter-active .goods-detail,
.goods-detail-fade-leave-active .goods-detail {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.goods-detail-fade-enter-from,
.goods-detail-fade-leave-to {
  opacity: 0;
}
.goods-detail-fade-enter-from .goods-detail,
.goods-detail-fade-leave-to .goods-detail {
  opacity: 0;
  transform: translateY(12px) scale(0.98);
}

/* ---------- 手机端：贴底抽屉 ---------- */
@media (max-width: 640px) {
  .goods-detail-mask {
    align-items: flex-end;
    padding: 0;
  }
  .goods-detail {
    max-width: none;
    max-height: 92vh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }
  .goods-detail-main {
    padding: 16px 16px 18px;
  }
  .goods-detail-title {
    font-size: 18px;
  }
  .goods-detail-coin strong {
    font-size: 22px;
  }
  .goods-detail-foot {
    padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
  }
  .goods-detail-buy {
    min-width: 0;
    flex: 1;
  }
}
</style>
