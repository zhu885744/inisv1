<template>
  <div class="goods-page">
    <SectionTitle title="积分商城">
      <template #extra>
        <span class="section-count">{{ total || goodsList.length }} 件可兑换</span>
      </template>
    </SectionTitle>

    <!-- 积分余额卡片：左余额 / 右兑换统计 -->
    <div v-if="userStore.isLogged" class="card balance-card">
      <div class="balance-main">
        <span class="balance-coin" aria-hidden="true">
          <i class="bi bi-coin" />
        </span>
        <div class="balance-info">
          <div class="balance-label">我的积分余额</div>
          <div class="balance-value">
            {{ balance }}<em>积分</em>
          </div>
        </div>
        <router-link to="/user/integral" class="balance-more">
          积分明细 <i class="bi bi-chevron-right" aria-hidden="true" />
        </router-link>
      </div>

      <div class="balance-stats">
        <div class="stat-item">
          <span class="stat-num">{{ myStats.pending }}</span>
          <span class="stat-label">待发货</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ myStats.shipped }}</span>
          <span class="stat-label">已发货</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ myStats.completed }}</span>
          <span class="stat-label">已完成</span>
        </div>
        <div class="stat-item">
          <span class="stat-num">{{ myStats.integral_spent }}</span>
          <span class="stat-label">累计消耗</span>
        </div>
      </div>
    </div>

    <div v-else class="card balance-card">
      <div class="balance-main">
        <span class="balance-coin" aria-hidden="true">
          <i class="bi bi-coin" />
        </span>
        <div class="balance-info">
          <div class="balance-label">登录后查看积分余额</div>
          <div class="balance-hint">签到、发布内容、参与互动都能赚积分</div>
        </div>
        <router-link to="/auth/login" class="btn btn-sm btn-primary">立即登录</router-link>
      </div>
    </div>

    <!-- 商城分区：积分商品 / 装扮（装扮来自后台「装扮管理」，可自由扩展类型） -->
    <div class="shop-tabs">
      <button
        type="button"
        class="shop-tab"
        :class="{ 'is-active': activeTab === 'goods' }"
        @click="activeTab = 'goods'"
      >
        <i class="bi bi-bag" aria-hidden="true" /> 积分商品
      </button>
      <!-- 装扮分区受后台「装扮配置」的总开关与「商城显示」开关控制 -->
      <button
        v-if="decorationVisible"
        type="button"
        class="shop-tab"
        :class="{ 'is-active': activeTab === 'decoration' }"
        @click="switchTab('decoration')"
      >
        <i class="bi bi-palette" aria-hidden="true" /> 装扮
      </button>
    </div>

    <!-- 装扮分区被后台关闭时，始终回落到积分商品，避免出现空白区 -->
    <template v-if="activeTab === 'goods' || !decorationVisible">

    <!-- 商品分类 -->
    <div v-if="categories.length > 1" class="category-bar">
      <button
        v-for="c in categories"
        :key="c.category || 'all'"
        type="button"
        class="chip"
        :class="{ 'is-active': activeCategory === c.category }"
        @click="setCategory(c.category)"
      >
        {{ c.name }}
        <em class="chip-count">{{ c.count }}</em>
      </button>
    </div>

    <!-- 商品列表 -->
    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>
    <div v-else-if="!goodsList.length" class="card card-pad">
      <EmptyState text="暂无商品" />
    </div>
    <div v-else class="goods-grid">
      <article
        v-for="g in decoratedGoods"
        :key="g.id"
        class="goods-card"
        :class="{ 'is-disabled': g.can_buy === false }"
        role="button"
        tabindex="0"
        :aria-label="`查看 ${g.title} 详情`"
        @click="openDetail(g)"
        @keydown.enter.prevent="openDetail(g)"
        @keydown.space.prevent="openDetail(g)"
      >
        <div class="goods-cover-wrap">
          <img :src="g.cover || defaultCover" class="goods-cover" :alt="g.title" @error="onCoverError" loading="lazy" />
          <!-- 类型角标 -->
          <span class="goods-type" :class="g.type === 'physical' ? 'is-physical' : 'is-virtual'">
            <i :class="g.type === 'physical' ? 'bi bi-box-seam' : 'bi bi-lightning-charge'" aria-hidden="true" />
            {{ g.type === 'physical' ? '实物' : '虚拟' }}
          </span>
          <!-- 兑换时间状态角标：未开始 / 即将结束（1 小时内）/ 已结束 -->
          <span v-if="g._state" class="goods-state" :class="g._state.type">
            <i :class="g._state.icon" aria-hidden="true" /> {{ g._state.text }}
          </span>
        </div>

        <div class="goods-body">
          <h4 class="goods-title" :title="g.title">{{ g.title }}</h4>
          <p class="goods-desc">{{ g.description || '暂无描述' }}</p>

          <div class="goods-tags">
            <span v-if="Number(g.limit_per_user) > 0" class="goods-tag">
              限购 {{ g.limit_per_user }} 件<template v-if="Number(g.limit_remain) >= 0"> · 剩 {{ g.limit_remain }}</template>
            </span>
            <span v-if="Number(g.min_exp) > 0" class="goods-tag">需经验 {{ g.min_exp }}</span>
            <!-- 装扮类商品：展示关联装扮的有效期（永久 / N 天），兑换前就告知使用期限 -->
            <span
              v-if="g.decoration_duration !== undefined && g.decoration_duration !== null"
              class="goods-tag is-duration"
            >
              <i class="bi bi-clock-history" aria-hidden="true" /> {{ formatDuration(g.decoration_duration) }}
            </span>
            <!-- 距结束倒计时（1 小时内转为警示色） -->
            <span
              v-if="g._endCountdown"
              class="goods-tag is-deadline"
              :class="{ 'is-urgent': g._endUrgent }"
            >
              <i class="bi bi-hourglass-split" aria-hidden="true" /> {{ g._endCountdown }}
            </span>
          </div>

          <div class="goods-foot">
            <span class="goods-price" :class="{ 'is-free': isFreePrice(g.price) }">
              <i :class="isFreePrice(g.price) ? 'bi bi-gift' : 'bi bi-coin'" aria-hidden="true" />
              <span v-if="isFreePrice(g.price)" class="goods-free">免费</span>
              <template v-else>{{ g.price }}<em>积分</em></template>
            </span>
            <span class="goods-stock" :class="{ 'is-empty': Number(g.stock) <= 0 }">
              {{ Number(g.stock) > 0 ? `剩 ${g.stock}` : '已售罄' }}
            </span>
          </div>

          <!-- 整卡可点，这里只是视觉上的入口，避免出现嵌套按钮 -->
          <span class="goods-detail-entry" :title="g._title">
            查看详情 <i class="bi bi-arrow-right" aria-hidden="true" />
          </span>
        </div>
      </article>
    </div>

    <!-- 加载更多：商品超过一页时才出现 -->
    <div v-if="hasMore || loadingMore" class="goods-more">
      <button type="button" class="btn btn-ghost" :disabled="loadingMore" @click="loadMore">
        <span v-if="loadingMore" class="spinner" />
        {{ loadingMore ? '加载中...' : `加载更多（${goodsList.length}/${total}）` }}
      </button>
    </div>

    <!-- 我的订单 -->
    <template v-if="userStore.isLogged">
      <h3 class="block-title">我的兑换记录</h3>
      <div v-if="ordersLoading" class="is-loading"><span class="spinner" /> 加载中...</div>
      <div v-else-if="!orders.length" class="card card-pad">
        <EmptyState text="还没有兑换记录" />
      </div>
      <div v-else class="order-list">
        <div v-for="o in orders" :key="o.id" class="order-card card">
          <img :src="o.result?.goods?.cover || defaultCover" class="order-card-cover" :alt="o.result?.goods?.title" @error="onCoverError" loading="lazy" />
          <div class="order-card-body">
            <div class="order-card-head">
              <span class="order-card-title">{{ o.result?.goods?.title || o.goods_title || `商品 #${o.goods_id}` }}</span>
              <span class="status-tag" :class="`status-${o.status}`">{{ statusText(o.status) }}</span>
            </div>
            <div class="order-card-meta">
              <span class="order-card-price">
                <i class="bi bi-coin" /> {{ isFreePrice(o.price) ? '免费' : o.price }}
              </span>
              <span v-if="o.order_no" class="order-card-no">单号 {{ o.order_no }}</span>
              <span class="order-card-time">{{ formatTime(o.create_time) }}</span>
            </div>
            <!-- 发货内容（虚拟商品） -->
            <div v-if="o.deliver_content" class="order-card-deliver">
              <span class="deliver-tag">发货</span>
              <span class="deliver-text">{{ o.deliver_content }}</span>
              <button class="copy-btn" title="复制" @click="copyDeliverText(o.deliver_content)"><i class="bi bi-copy" /></button>
            </div>
            <!-- 物流（实物商品） -->
            <div v-else-if="o.logistics" class="order-card-deliver">
              <span class="deliver-tag">物流</span>
              <span class="deliver-text">{{ o.logistics }}</span>
            </div>
            <!-- 收货地址（实物商品） -->
            <div v-if="o.result?.address" class="order-card-address">
              <i class="bi bi-geo-alt" />
              <span>{{ o.result.address.name }} · {{ o.result.address.phone }}</span>
              <span class="address-detail">{{ o.result.address.address }}</span>
            </div>
            <!-- 已取消：展示退款金额 -->
            <div v-if="o.status === 3" class="order-card-refund">
              <i class="bi bi-arrow-counterclockwise" />
              <span v-if="isFreePrice(o.refund || o.price)">订单已取消，未消耗积分</span>
              <span v-else>订单已取消，已退还 {{ o.refund || o.price }} 积分</span>
            </div>
            <!-- 订单操作 -->
            <div v-if="o.status === 0 || o.status === 1" class="order-card-actions">
              <button
                v-if="o.status === 0"
                type="button"
                class="btn btn-sm btn-ghost"
                :disabled="cancelingId === o.id"
                @click="askCancel(o)"
              >
                <i class="bi bi-x-circle" /> {{ cancelingId === o.id ? '取消中...' : '取消订单' }}
              </button>
              <button
                v-if="o.status === 1"
                type="button"
                class="btn btn-sm btn-primary"
                :disabled="receivingId === o.id"
                @click="doReceive(o)"
              >
                <i class="bi bi-check2-circle" /> {{ receivingId === o.id ? '处理中...' : '确认收货' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 兑换确认弹窗（虚拟商品） -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="showConfirm" class="modal-overlay" @click.self="showConfirm = false">
          <div class="modal-card">
            <div class="modal-header">
              <span class="modal-title">确认兑换</span>
              <button class="btn btn-icon btn-sm btn-round" @click="showConfirm = false"><i class="bi bi-x-lg" /></button>
            </div>
            <div class="modal-body">
              <div class="confirm-goods">
                <img :src="confirmGoods?.cover || defaultCover" class="confirm-cover" :alt="confirmGoods?.title" @error="onCoverError" loading="lazy" />
                <div class="confirm-info">
                  <div class="confirm-title">{{ confirmGoods?.title }}</div>
                  <div class="confirm-price">
                    <i class="bi bi-coin" />
                    {{ isFreePrice(confirmGoods?.price) ? '免费' : `${confirmGoods?.price} 积分` }}
                  </div>
                </div>
              </div>
              <div class="confirm-tip">
                <template v-if="isFreePrice(confirmGoods?.price)">兑换后立即发货，确定免费兑换吗？</template>
                <template v-else>兑换后立即发货，确定使用 <strong>{{ confirmGoods?.price }}</strong> 积分兑换吗？</template>
              </div>
              <!-- 装扮类商品：兑换前说明有效期与后续佩戴入口 -->
              <div
                v-if="confirmGoods?.decoration_duration !== undefined && confirmGoods?.decoration_duration !== null"
                class="confirm-tip"
              >
                有效期：<strong>{{ formatDuration(confirmGoods.decoration_duration) }}</strong>，兑换后可前往「我的装扮」佩戴。
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" @click="showConfirm = false">取消</button>
              <button class="btn btn-primary" @click="confirmBuy">确认兑换</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 收货地址弹窗（实物商品） -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="showAddress" class="modal-overlay" @click.self="showAddress = false">
          <div class="modal-card">
            <div class="modal-header">
              <span class="modal-title">填写收货地址</span>
              <button class="btn btn-icon btn-sm btn-round" @click="showAddress = false"><i class="bi bi-x-lg" /></button>
            </div>
            <div class="modal-body">
              <div class="field">
                <label>收货人</label>
                <FormInput v-model="addressForm.name" placeholder="请输入收货人姓名" />
              </div>
              <div class="field">
                <label>联系电话</label>
                <FormInput v-model="addressForm.phone" placeholder="请输入联系电话" />
              </div>
              <div class="field">
                <label>收货地址</label>
                <FormTextarea v-model="addressForm.address" rows="3" placeholder="请输入详细收货地址" />
              </div>
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" @click="showAddress = false">取消</button>
              <button class="btn btn-primary" :disabled="addressSubmitting" @click="submitAddress">确认兑换</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 发货内容弹窗（虚拟商品） -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="showDeliver" class="modal-overlay" @click.self="showDeliver = false">
          <div class="modal-card">
            <div class="modal-header">
              <span class="modal-title">兑换成功</span>
              <button class="btn btn-icon btn-sm btn-round" @click="showDeliver = false"><i class="bi bi-x-lg" /></button>
            </div>
            <div class="modal-body">
              <div class="deliver-label">{{ deliverGoods?.title }}</div>
              <div class="deliver-content">{{ deliverContent }}</div>
              <button class="btn btn-sm btn-ghost btn-block" @click="copyDeliver">复制内容</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 商品详情弹窗：列表卡片与「查看详情」都打开它，兑换入口也在里面 -->
    <GoodsDetailDialog
      v-model:visible="showDetail"
      :goods="detailGoods"
      :balance="balance"
      :is-logged="userStore.isLogged"
      :buying="buyingId === detailGoods?.id"
      @buy="onDetailBuy"
    />

    <!-- 取消订单确认弹窗 -->
    <ConfirmDialog
      v-model:visible="showCancelConfirm"
      title="取消订单"
      message="确定取消该订单吗？取消后积分将原路退还，商品库存会恢复。"
      confirm-text="取消订单"
      cancel-text="再想想"
      loading-text="取消中..."
      danger
      :loading="cancelingId !== null"
      @confirm="doCancel"
    />
    </template>

    <!-- 装扮商城：头像框 / 头衔（类型分区由后端元数据驱动） -->
    <DecorationShop v-else-if="decorationVisible" @purchased="onDecorationPurchased" />
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useUserStore } from '@/stores/user'
import {
  getGoods,
  buyGoods,
  getOrders,
  getIntegral,
  getGoodsCategories,
  getMyGoodsStats,
  cancelOrder,
  receiveOrder
} from '@/api/goods'
import { toast } from '@/utils/toast'
import { formatCountdown, remainSeconds } from '@/utils/countdown'
import { formatDuration } from '@/utils/time'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import DecorationShop from '@/components/DecorationShop.vue'
import GoodsDetailDialog from '@/components/goods/GoodsDetailDialog.vue'
import { getDecorationConfig } from '@/api/decoration'

const route = useRoute()
const userStore = useUserStore()

// 商城分区：goods 积分商品 / decoration 装扮
const activeTab = ref(route.query.tab === 'decoration' ? 'decoration' : 'goods')
// 装扮分区是否展示（受后台「装扮配置」的 enabled + show_in_goods 控制）
const decorationVisible = ref(false)

async function loadDecorationConfig() {
  try {
    const res = await getDecorationConfig()
    const data = res?.data || {}
    decorationVisible.value = Number(data.enabled) === 1 && Number(data.show_in_goods) === 1
    if (!decorationVisible.value && activeTab.value === 'decoration') activeTab.value = 'goods'
  } catch {
    decorationVisible.value = false
  }
}

const goodsList = ref([])
const orders = ref([])
const balance = ref(0)
const loading = ref(false)
const ordersLoading = ref(false)
const buyingId = ref(null)

// 分页：「加载更多」逐页追加；静默刷新时按已加载数量整体重取，避免把追加的页冲掉
const LIST_LIMIT = 24
const page = ref(1)
const total = ref(0)
const loadingMore = ref(false)
const hasMore = computed(() => goodsList.value.length < total.value)

// 分类
const categories = ref([])
const activeCategory = ref('')

// 我的兑换统计
const myStats = ref({ pending: 0, shipped: 0, completed: 0, canceled: 0, integral_spent: 0 })

// 订单操作
const showCancelConfirm = ref(false)
const cancelTarget = ref(null)
const cancelingId = ref(null)
const receivingId = ref(null)
const showAddress = ref(false)
const addressForm = ref({ name: '', phone: '', address: '', goods: null })
const addressSubmitting = ref(false)
const showConfirm = ref(false)
const confirmGoods = ref(null)
const showDeliver = ref(false)
const deliverContent = ref('')
const deliverGoods = ref(null)

const defaultCover = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect width="120" height="120" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="28" fill="%238a8a82" font-family="serif">兑</text></svg>'

/** 0 积分商品按「免费」展示，UI 里不再出现「0 积分」 */
const isFreePrice = (price) => Number(price) <= 0

function onCoverError(e) {
  e.target.src = defaultCover
}

function statusText(s) {
  const map = { 0: '待发货', 1: '已发货', 2: '已完成', 3: '已取消' }
  return map[s] ?? '未知'
}

function formatTime(t) {
  if (!t) return '-'
  const d = new Date(Number(t) * 1000)
  if (isNaN(d)) return '-'
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// ===== 兑换时间倒计时 =====
// 「未开始」显示距开抢倒计时，「进行中」显示距结束倒计时（每秒刷新）；
// 任一倒计时走完后静默刷新列表，拿回后端最新的 can_buy / buy_reason。
const nowSec = ref(Math.floor(Date.now() / 1000))
let timer = null
// 即将结束的阈值：1 小时内高亮提醒
const ENDING_SOON = 3600

const decoratedGoods = computed(() =>
  goodsList.value.map((g) => {
    const nowMs = nowSec.value * 1000
    const startLeft = remainSeconds(g.start_time, nowMs)
    const endLeft = remainSeconds(g.end_time, nowMs)
    const startCountdown = startLeft > 0 ? formatCountdown(startLeft) : ''
    const endCountdown = endLeft > 0 ? formatCountdown(endLeft) : ''

    const startText = Number(g.start_time) > 0 ? formatTime(g.start_time) : ''
    const endText = Number(g.end_time) > 0 ? formatTime(g.end_time) : ''
    const title = [startText && `开抢 ${startText}`, endText && `结束 ${endText}`].filter(Boolean).join(' · ')

    let state = null
    if (startCountdown) {
      state = { type: 'is-soon', icon: 'bi bi-clock', text: '未开始' }
    } else if (endCountdown && endLeft <= ENDING_SOON) {
      state = { type: 'is-ending', icon: 'bi bi-hourglass-split', text: '限时兑换即将结束' }
    } else if (endText && !endCountdown) {
      state = { type: 'is-end', icon: 'bi bi-slash-circle', text: '已结束' }
    }

    return {
      ...g,
      _startCountdown: startCountdown,
      _endCountdown: endCountdown,
      _endUrgent: endCountdown !== '' && endLeft <= ENDING_SOON,
      _title: title,
      // 是否仍有倒计时在跑（用于驱动定时器）
      _live: !!(startCountdown || endCountdown),
      _state: state
    }
  })
)

const countingCount = computed(() => decoratedGoods.value.filter((g) => g._live).length)

function startTimer() {
  if (timer) return
  timer = setInterval(() => {
    const before = countingCount.value
    nowSec.value = Math.floor(Date.now() / 1000)
    // 有商品的倒计时刚走完（开抢 / 结束）→ 静默刷新兑换状态
    if (countingCount.value < before) loadGoods(true)
  }, 1000)
}

function stopTimer() {
  if (!timer) return
  clearInterval(timer)
  timer = null
}

watch(
  countingCount,
  (count) => {
    if (count > 0) startTimer()
    else stopTimer()
  },
  { immediate: true }
)

onUnmounted(stopTimer)

// silent = true 时不显示整页 loading（用于倒计时结束后的静默刷新）
// append = true 时追加下一页（「加载更多」）
async function loadGoods(silent = false, append = false) {
  if (append) loadingMore.value = true
  else if (!silent) loading.value = true
  try {
    const params = { page: append ? page.value + 1 : 1, limit: LIST_LIMIT }
    if (activeCategory.value) params.category = activeCategory.value

    const res = await getGoods(params)
    const rows = res.data?.data || []
    total.value = Number(res.data?.count) || rows.length

    if (append) {
      goodsList.value = [...goodsList.value, ...rows]
    } else if (silent) {
      // 静默刷新只更新第一页的兑换状态，保留「加载更多」出来的后续页
      const latest = new Map(rows.map((row) => [row.id, row]))
      goodsList.value = goodsList.value.map((item) => latest.get(item.id) || item)
    } else {
      goodsList.value = rows
    }

    if (!silent) page.value = params.page
  } catch {
    if (!append && !silent) goodsList.value = []
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadGoods(false, true)
}

async function loadCategories() {
  try {
    const res = await getGoodsCategories()
    const data = Array.isArray(res.data) ? res.data : (res.data?.data || [])
    categories.value = data
  } catch {
    categories.value = []
  }
}

function setCategory(category) {
  if (activeCategory.value === category) return
  activeCategory.value = category || ''
  // 换分类回到第一页，避免带着「已展开页数」去请求新分类
  page.value = 1
  loadGoods()
}

async function loadMyStats() {
  if (!userStore.isLogged) return
  try {
    const res = await getMyGoodsStats()
    const data = res.data || {}
    myStats.value = {
      pending: Number(data.pending) || 0,
      shipped: Number(data.shipped) || 0,
      completed: Number(data.completed) || 0,
      canceled: Number(data.canceled) || 0,
      integral_spent: Number(data.integral_spent) || 0
    }
  } catch {
    // 忽略
  }
}

// 取消订单：弹窗二次确认后提交，积分原路退还
function askCancel(order) {
  cancelTarget.value = order
  showCancelConfirm.value = true
}

async function doCancel() {
  const order = cancelTarget.value
  if (!order || cancelingId.value) return
  cancelingId.value = order.id
  try {
    const res = await cancelOrder(order.id)
    toast.success(`订单已取消，退还 ${res.data?.refund ?? order.price} 积分`)
    if (res.data?.integral !== undefined) balance.value = Number(res.data.integral) || 0
    showCancelConfirm.value = false
    cancelTarget.value = null
    await Promise.all([loadOrders(), loadGoods(), loadMyStats()])
  } catch {
    // 拦截器已提示
  } finally {
    cancelingId.value = null
  }
}

async function doReceive(order) {
  if (receivingId.value) return
  receivingId.value = order.id
  try {
    await receiveOrder(order.id)
    toast.success('确认收货成功')
    await Promise.all([loadOrders(), loadMyStats()])
  } catch {
    // 拦截器已提示
  } finally {
    receivingId.value = null
  }
}

async function loadBalance() {
  if (!userStore.isLogged) return
  try {
    const res = await getIntegral()
    balance.value = Number(res.data?.integral) || 0
  } catch {
    balance.value = 0
  }
}

async function loadOrders() {
  if (!userStore.isLogged) return
  ordersLoading.value = true
  try {
    const res = await getOrders()
    orders.value = res.data?.data || []
  } catch {
    orders.value = []
  } finally {
    ordersLoading.value = false
  }
}

// ===== 商品详情弹窗 =====
const showDetail = ref(false)
const detailGoods = ref(null)

function openDetail(goods) {
  detailGoods.value = goods
  showDetail.value = true
}

/** 详情里点兑换：先收起详情，再走原有兑换流程（虚拟商品确认 / 实物填地址） */
function onDetailBuy(goods) {
  showDetail.value = false
  buy(goods)
}

async function buy(g) {
  if (!userStore.isLogged) {
    toast.info('请先登录')
    return
  }
  if (buyingId.value) return

  // 实物商品：先填写收货地址
  if (g.type === 'physical') {
    addressForm.value = { name: '', phone: '', address: '', goods: g }
    showAddress.value = true
    return
  }

  // 虚拟商品：弹窗确认后立即购买
  confirmGoods.value = g
  showConfirm.value = true
}

function confirmBuy() {
  const g = confirmGoods.value
  if (!g) return
  showConfirm.value = false
  doBuy(g, '')
}

async function submitAddress() {
  const g = addressForm.value.goods
  if (!g) return
  if (!addressForm.value.name.trim()) return toast.info('请填写收货人')
  if (!addressForm.value.phone.trim()) return toast.info('请填写联系电话')
  if (!addressForm.value.address.trim()) return toast.info('请填写收货地址')

  const address = JSON.stringify({
    name: addressForm.value.name.trim(),
    phone: addressForm.value.phone.trim(),
    address: addressForm.value.address.trim()
  })

  addressSubmitting.value = true
  showAddress.value = false
  try {
    await doBuy(g, address)
  } finally {
    addressSubmitting.value = false
  }
}

async function doBuy(g, address) {
  buyingId.value = g.id
  try {
    const res = await buyGoods(g.id, address)
    if (res.code === 200) {
      toast.success(`兑换成功！剩余 ${res.data?.integral ?? 0} 积分`)
      balance.value = Number(res.data?.integral) || 0
      // 虚拟商品：弹出发货内容
      if (g.type !== 'physical' && res.data?.deliver_content) {
        deliverContent.value = res.data.deliver_content
        deliverGoods.value = g
        showDeliver.value = true
      }
      await Promise.all([loadGoods(), loadOrders(), loadMyStats()])
    } else {
      toast.error(res.msg || '兑换失败')
    }
  } catch {
    // 错误已由拦截器提示
  } finally {
    buyingId.value = null
  }
}

async function copyDeliver() {
  try {
    await navigator.clipboard.writeText(deliverContent.value || '')
    toast.success('已复制到剪贴板')
  } catch {
    toast.info('复制失败，请手动复制')
  }
}

async function copyDeliverText(text) {
  try {
    await navigator.clipboard.writeText(text || '')
    toast.success('已复制到剪贴板')
  } catch {
    toast.info('复制失败，请手动复制')
  }
}

// 切换分区：装扮分区的数据由 DecorationShop 自己加载
function switchTab(tab) {
  activeTab.value = tab
}

// 装扮兑换 / 佩戴成功后刷新积分与统计
function onDecorationPurchased() {
  loadBalance()
  loadMyStats()
  loadOrders()
}

onMounted(() => {
  loadDecorationConfig()
  loadGoods()
  loadBalance()
  loadOrders()
  loadCategories()
  loadMyStats()
})
</script>

<style scoped>
.goods-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.link {
  font-size: 13px;
  text-decoration: none;
}
.link:hover {
  color: var(--primary);
}
.block-title {
  font-size: 15px;
  font-weight: 600;
  margin: 8px 0 0;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-soft);
}

.balance-card {
  display: flex;
  align-items: stretch;
  gap: 0;
  padding: 0;
  overflow: hidden;
  background: linear-gradient(135deg, var(--gold-soft), var(--accent-wash));
  border: 1px solid var(--gold-line);
}
.balance-main {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
}
.balance-coin {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--bg-card);
  border: 1px solid var(--gold-line);
  color: #d4a148;
  font-size: 24px;
  box-shadow: var(--shadow-sm);
}
.balance-info {
  min-width: 0;
  flex: 1;
}
.balance-label {
  font-size: 12px;
  color: var(--text-muted);
}
.balance-value {
  margin-top: 2px;
  font-size: 26px;
  font-weight: 800;
  line-height: 1.2;
  color: var(--primary-deep);
  /* 数字等宽，余额变化时不抖动 */
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
}
.balance-value em {
  margin-left: 5px;
  font-style: normal;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
}
.balance-hint {
  margin-top: 2px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
}
.balance-more {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: var(--control-h-sm);
  padding: 0 12px;
  border: 1px solid var(--gold-line);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 12px;
  text-decoration: none;
  transition: all var(--dur-fast) var(--ease);
}
.balance-more:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
/* 我的兑换统计：与余额区用竖线隔开 */
.balance-stats {
  flex: 0 0 auto;
  display: grid;
  grid-template-columns: repeat(4, minmax(64px, 1fr));
  align-items: center;
  gap: 4px;
  padding: 14px 18px;
  border-left: 1px dashed var(--gold-line);
}
.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}
.stat-num {
  font-size: 17px;
  font-weight: 700;
  color: #d4a148;
  font-variant-numeric: tabular-nums;
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
}
.section-count {
  font-size: 12px;
  color: var(--text-muted);
}

/* 商城分区切换：积分商品 / 装扮 */
.shop-tabs {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 10px;
  background: var(--bg-muted);
  align-self: flex-start;
}
.shop-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 18px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-soft);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.18s;
}
.shop-tab:hover {
  color: var(--primary);
}
.shop-tab.is-active {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

/* 商品分类 */
.category-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  line-height: 1.5;
  cursor: pointer;
  transition: all 0.18s;
}
.chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}
.chip-count {
  font-size: 11px;
  font-style: normal;
  color: var(--text-muted);
}
.chip.is-active .chip-count {
  color: var(--primary-deep);
}

.is-loading {
  padding: 48px;
  text-align: center;
  color: var(--text-muted);
}

.goods-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(224px, 1fr));
  gap: 16px;
}
.goods-more {
  display: flex;
  justify-content: center;
  padding: 2px 0 4px;
}
.goods-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: transform var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease);
}
.goods-card:hover,
.goods-card:focus-visible {
  transform: translateY(-2px);
  border-color: var(--primary-soft);
  box-shadow: var(--shadow);
  outline: none;
}
.goods-card.is-disabled {
  opacity: 0.88;
}
.goods-cover-wrap {
  position: relative;
  width: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  background: var(--bg-muted);
}
.goods-cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform var(--dur-slow) var(--ease);
}
.goods-card:hover .goods-cover {
  transform: scale(1.03);
}
/* 类型角标（右上）：虚拟 / 实物 */
.goods-type {
  position: absolute;
  top: 10px;
  right: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1.7;
  color: #fff;
}
.goods-type.is-virtual {
  background: rgba(46, 122, 92, 0.92);
}
.goods-type.is-physical {
  background: rgba(199, 72, 42, 0.92);
}
/* 兑换时间状态角标（左上） */
.goods-state {
  position: absolute;
  top: 10px;
  left: 10px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  font-size: 11px;
  line-height: 1.6;
  color: #fff;
  border-radius: 999px;
}
.goods-state.is-soon {
  background: var(--warning);
}
.goods-state.is-ending {
  background: var(--danger);
}
.goods-state.is-end {
  background: rgba(50, 46, 38, 0.72);
}
.goods-body {
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
}
.goods-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.45;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.goods-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  min-height: 38px;
}
.goods-foot {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
  padding-top: 2px;
}
.goods-price {
  /* 图标与金额是同级的 flex 子项：用 gap 控间距，
     不要依赖模板里的换行（Vue 会丢掉元素之间的纯空白文本节点） */
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 18px;
  font-weight: 800;
  color: #d4a148;
  font-variant-numeric: tabular-nums;
}
.goods-price .bi {
  font-size: 13px;
}
.goods-price em {
  margin-left: 0;
  font-style: normal;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
}
/* 0 积分商品：显示「免费」而不是「0 积分」 */
.goods-price.is-free {
  color: var(--success);
}
.goods-free {
  font-size: 15px;
  font-weight: 800;
  color: var(--success);
  letter-spacing: 0.02em;
}
.goods-stock {
  font-size: 12px;
  color: var(--text-muted);
}
.goods-stock.is-empty {
  color: var(--danger);
}
/* 商品标签：限购 / 经验门槛 / 有效期 / 倒计时 */
.goods-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.goods-tag {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  white-space: nowrap;
}
/* 距结束倒计时标签（1 小时内转警示色） */
.goods-tag.is-deadline {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--gold-wash);
  color: var(--warning);
}
.goods-tag.is-deadline.is-urgent {
  background: var(--accent-soft);
  color: var(--danger);
}
/* 有效期标签（装扮类商品） */
.goods-tag.is-duration {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--gold-wash);
  color: #c7902f;
}
/* 卡片底部的详情入口（整卡可点，这里只做视觉提示） */
.goods-detail-entry {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: var(--control-h-sm);
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 13px;
  font-weight: 500;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.goods-card:hover .goods-detail-entry {
  background: var(--primary);
  color: #fff;
}

.status-tag {
  display: inline-block;
  padding: 2px 10px;
  font-size: 12px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-soft);
}
.status-tag.status-1 {
  background: rgba(74, 144, 226, 0.12);
  color: #4a90e2;
}
.status-tag.status-2 {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.status-tag.status-3 {
  background: var(--bg-muted);
  color: var(--text-light);
}

/* 兑换记录卡片 */
.order-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.order-card {
  display: flex;
  gap: 14px;
  padding: 14px;
}
.order-card-cover {
  width: 56px;
  height: 56px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
}
.order-card-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.order-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
}
.order-card-title {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.order-card-meta {
  display: flex;
  align-items: center;
  gap: 14px;
  font-size: 12px;
  color: var(--text-muted);
}
.order-card-price {
  /* 图标与文字用 gap 控间距（模板换行会被 Vue 压缩掉） */
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: #d4a148;
  font-weight: 600;
}
.order-card-price .bi {
  font-size: 12px;
}
.order-card-no {
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}
.order-card-refund {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--bg-muted);
  font-size: 12px;
  color: var(--text-muted);
}
.order-card-refund .bi {
  color: var(--success);
}
.order-card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 2px;
}
.order-card-deliver {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 10px;
  background: var(--bg-muted);
  border-radius: 6px;
  font-size: 12px;
}
.deliver-tag {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 11px;
  border-radius: 4px;
  background: var(--gold-soft);
  color: #c7902f;
  font-weight: 600;
  line-height: 18px;
}
.deliver-text {
  flex: 1;
  min-width: 0;
  color: var(--text-soft);
  word-break: break-all;
  line-height: 1.5;
}
.copy-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s;
}
.copy-btn:hover {
  background: var(--bg-card);
  color: var(--primary);
}
.order-card-address {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
  flex-wrap: wrap;
}
.order-card-address .bi {
  color: var(--primary);
}
.address-detail {
  color: var(--text-soft);
}
.text-muted {
  color: var(--text-muted);
}

/* 弹窗 */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.modal-card {
  width: 100%;
  max-width: 420px;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-soft);
}
.modal-title {
  font-size: 16px;
  font-weight: 600;
}
.modal-body {
  padding: 20px;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 16px 20px;
  border-top: 1px solid var(--border-soft);
}
.field {
  margin-bottom: 14px;
}
.field label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: var(--text-soft);
}
.input {
  width: 100%;
  padding: 9px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--text);
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
  font-family: inherit;
}
.input:focus {
  border-color: var(--primary);
}
.deliver-label {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 10px;
}
.deliver-content {
  padding: 14px;
  background: var(--bg-muted);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
  font-size: 14px;
  line-height: 1.6;
  word-break: break-all;
  white-space: pre-wrap;
  margin-bottom: 14px;
}

.confirm-goods {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
}
.confirm-cover {
  width: 52px;
  height: 52px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
}
.confirm-info {
  min-width: 0;
}
.confirm-title {
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.confirm-price {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 15px;
  font-weight: 700;
  color: #d4a148;
}
.confirm-tip {
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.6;
}
.confirm-tip strong {
  color: #d4a148;
  font-weight: 700;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s;
}
.modal-fade-enter-active .modal-card,
.modal-fade-leave-active .modal-card {
  transition: transform 0.2s, opacity 0.2s;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-from .modal-card,
.modal-fade-leave-to .modal-card {
  transform: scale(0.95) translateY(-8px);
  opacity: 0;
}

/* 手机端：余额与统计上下排列，统计改为四等分一排 */
@media (max-width: 640px) {
  .balance-card {
    flex-direction: column;
  }
  .balance-main {
    gap: 12px;
    padding: 14px;
  }
  .balance-coin {
    width: 40px;
    height: 40px;
    font-size: 20px;
  }
  .balance-value {
    font-size: 22px;
  }
  .balance-stats {
    grid-template-columns: repeat(4, 1fr);
    padding: 12px 10px;
    border-left: none;
    border-top: 1px dashed var(--gold-line);
  }
  .stat-label {
    white-space: normal;
  }
  .goods-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 12px;
  }
  .goods-body {
    padding: 12px;
    gap: 6px;
  }
  .goods-desc {
    min-height: 0;
  }
  .goods-detail-entry {
    height: var(--control-h);
  }
}
</style>
