<template>
  <div class="decorations-page">
    <SectionTitle title="我的装扮">
      <template #extra>
        <RouterLink to="/goods?tab=decoration" class="link">
          前往装扮商城 <i class="bi bi-arrow-right" />
        </RouterLink>
      </template>
    </SectionTitle>

    <!-- 当前佩戴：头像框 + 头衔，一眼看清现在用的是什么 -->
    <div class="card card-pad wearing-card">
      <AvatarFrame
        :src="user?.avatar"
        :name="user?.nickname"
        :frame="wearingFrame"
        :animated="wearingFrameAnimated"
        :fallback="defaultAvatar"
        :frame-hole="wearingFrameGeometry.hole"
        :frame-offset-x="wearingFrameGeometry.offsetX"
        :frame-offset-y="wearingFrameGeometry.offsetY"
        :verified="user?.cert"
        size="64px"
      />
      <div class="wearing-info">
        <div class="wearing-label">当前佩戴</div>
        <div class="wearing-row">
          <TitleBadge
            v-if="wearingTitle"
            :payload="wearingTitle.payload"
            :text="wearingTitle.custom_text"
          />
          <span v-else class="wearing-empty">未佩戴头衔</span>
          <span class="wearing-sep">·</span>
          <span :class="wearingFrame ? 'wearing-name' : 'wearing-empty'">
            {{ wearingFrame ? (wearingFrame.name || '已使用头像框') : '未使用头像框' }}
          </span>
        </div>
      </div>
      <div class="wearing-count">
        已拥有 <strong>{{ ownedList.length }}</strong> 件
      </div>
    </div>

    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

    <template v-else>
      <!-- 按类型分区：类型元数据由后端 types 驱动，后台新增类型后自动出现 -->
      <section v-for="group in groups" :key="group.type" class="decoration-section">
        <div class="section-head">
          <h3 class="section-name"><i :class="group.icon" aria-hidden="true" /> {{ group.name }}</h3>
          <span class="section-count">{{ group.list.length }} 件</span>
        </div>

        <div v-if="group.list.length" class="decoration-grid">
          <div
            v-for="item in group.list"
            :key="item.id"
            class="decoration-card card"
            :class="{ 'is-expired': item.expired || item.off_shelf, 'is-wearing': item.wearing }"
          >
            <div class="decoration-preview" :class="{ 'is-title': item.type === 'title' }">
              <img
                v-if="item.type === 'avatar_frame'"
                :src="item.preview || item.payload?.image"
                class="decoration-frame"
                :alt="item.name"
                loading="lazy"
              />
              <TitleBadge v-else :payload="item.payload" :text="item.custom_text" size="lg" />
              <!-- 稀有度角标：与装扮商城保持一致，取值来自后台「装扮管理」 -->
              <span v-if="item.rarity" class="rarity-badge" :class="`is-${rarityMeta(item.rarity).tone}`">
                {{ rarityMeta(item.rarity).label }}
              </span>
            </div>

            <div class="decoration-body">
              <div class="decoration-name">
                <span class="decoration-name-text">{{ item.name }}</span>
                <span v-if="item.wearing" class="decoration-tag is-wearing">佩戴中</span>
                <span v-else-if="item.expired" class="decoration-tag is-expired">已过期</span>
                <span v-else-if="item.off_shelf" class="decoration-tag is-expired">已下架</span>
                <span v-else-if="item.is_default" class="decoration-tag">默认</span>
              </div>

              <div class="decoration-meta">
                <span class="decoration-duration" :class="{ 'is-urgent': isUrgent(item) }">
                  <i class="bi bi-clock-history" aria-hidden="true" /> {{ durationText(item) }}
                </span>
                <span class="decoration-source">{{ sourceText(item) }}</span>
              </div>

              <div class="decoration-actions">
                <button
                  v-if="item.wearing"
                  type="button"
                  class="btn btn-sm btn-ghost btn-block"
                  :disabled="actingId === item.id"
                  @click="unwear(item)"
                >
                  <i class="bi bi-dash-circle" /> {{ actingId === item.id ? '处理中...' : '卸下' }}
                </button>
                <button
                  v-else
                  type="button"
                  class="btn btn-sm btn-primary btn-block"
                  :disabled="!!blockReason(item) || actingId === item.id"
                  :title="blockTip(item)"
                  @click="wear(item)"
                >
                  <span v-if="actingId === item.id" class="spinner" />
                  <template v-else>{{ blockReason(item) || '立即佩戴' }}</template>
                </button>

                <!-- 有限期 + 后台给这件装扮配了续期档位，才显示续期入口（永久装扮后端也会拒绝） -->
                <button
                  v-if="Number(item.expire_time) > 0 && item.renew_options?.length"
                  type="button"
                  class="btn btn-sm btn-ghost btn-block"
                  @click="openRenew(item)"
                >
                  <i class="bi bi-arrow-repeat" /> 续期
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="card card-pad">
          <EmptyState icon="bi bi-palette" :text="`还没有${group.name}，去装扮商城兑换一个吧`" />
        </div>
      </section>
    </template>

    <!-- 纯样式类头衔：佩戴前需要自填文字 -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="pendingTitle" class="modal-overlay" @click.self="pendingTitle = null">
          <div class="modal-card">
            <div class="modal-header">
              <span class="modal-title">设置头衔文字</span>
              <button class="btn btn-icon btn-sm btn-round" @click="pendingTitle = null"><i class="bi bi-x-lg" /></button>
            </div>
            <div class="modal-body">
              <div class="pending-preview">
                <TitleBadge :payload="pendingTitle.payload" :text="customTitleText" size="lg" />
              </div>
              <FormInput
                v-model="customTitleText"
                maxlength="10"
                placeholder="请输入头衔文字（最多 10 字）"
                @keyup.enter="confirmWear" />
            </div>
            <div class="modal-footer">
              <button class="btn btn-ghost" @click="pendingTitle = null">取消</button>
              <button class="btn btn-primary" @click="confirmWear">确定佩戴</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- 续期：价格与档位是这件装扮自己的（后台在装扮编辑里配置） -->
    <RenewPlanDialog
      :visible="renew.visible"
      scope="decoration"
      :target-id="renew.id"
      :target-name="renew.name"
      :current-expire="renew.expire"
      :options="renew.options"
      @update:visible="(v) => (renew.visible = v)"
      @done="loadMine"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, computed, onMounted } from 'vue'
import { useUserStore } from '@/stores/user'
import { getMyDecorations, wearDecoration, unwearDecoration } from '@/api/decoration'
import { toast } from '@/utils/toast'
import { formatDuration } from '@/utils/time'
import { rarityMeta } from '@/utils/decoration'
import { frameGeometryFromPayload } from '@/utils/helper'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import RenewPlanDialog from '@/components/RenewPlanDialog.vue'
import AvatarFrame from '@/components/AvatarFrame.vue'
import TitleBadge from '@/components/TitleBadge.vue'

const userStore = useUserStore()
const user = computed(() => userStore.user)

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const loading = ref(false)
const actingId = ref(null)
// 续期弹层：只对「有限期 + 配了续期档位」的装扮开放
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
// 后端 /decoration/mine 返回：{ types, wearing, list }
const store = ref({ types: [], wearing: {}, list: {} })

// 装扮来源 → 文案（让用户知道这件是怎么来的）
const SOURCE_TEXT = {
  shop: '商城兑换',
  admin: '管理员发放',
  activity: '活动获得',
  level: '等级解锁',
  gacha: '抽卡获得',
  default: '系统默认'
}

function sourceText(item) {
  return SOURCE_TEXT[item.source] || ''
}

// 剩余天数（expire_time=0 表示永久）
function daysLeft(item) {
  const expire = Number(item.expire_time) || 0
  if (!expire) return null
  return Math.ceil((expire * 1000 - Date.now()) / 86400000)
}

function durationText(item) {
  if (item.expired) return '已过期'
  const days = daysLeft(item)
  if (days === null) return '永久有效'
  return days <= 0 ? '即将过期' : `剩余 ${days} 天`
}

// 3 天内到期标红提醒
function isUrgent(item) {
  const days = daysLeft(item)
  return !item.expired && days !== null && days <= 3
}

// 不可佩戴的原因（返回空字符串表示可佩戴）：
// 已过期 → 需要重新获取；已下架 → 后台停用了该装扮（服务端 WearDecoration 会拒绝）
function blockReason(item) {
  if (item.expired) return '已过期'
  if (item.off_shelf) return '已下架'
  return ''
}

function blockTip(item) {
  if (item.expired) return '该装扮已过期，可在装扮商城重新获取'
  if (item.off_shelf) return '该装扮已被后台下架，暂时无法佩戴'
  return ''
}

const groups = computed(() =>
  (store.value.types || []).map((type) => ({
    type: type.type,
    name: type.name,
    icon: type.icon,
    list: (store.value.list?.[type.type] || []).map((item) => ({
      ...item,
      expired: !!item.expired
    }))
  }))
)

const ownedList = computed(() => groups.value.flatMap((g) => g.list))

// 当前佩戴（wearing: { [type]: 装扮对象 }）
const wearingFrameItem = computed(() => store.value.wearing?.avatar_frame || null)
const wearingFrame = computed(
  () => wearingFrameItem.value?.preview || wearingFrameItem.value?.payload?.image || ''
)
// 头像框定位参数：这里拿得到装扮的 payload，比 users.json 上的回写值更直接
const wearingFrameGeometry = computed(() => frameGeometryFromPayload(wearingFrameItem.value?.payload || {}))
// 动效开关：之前误写成 wearingFrame.payload（wearingFrame 是图片地址字符串，永远取不到）
const wearingFrameAnimated = computed(() => {
  const animated = wearingFrameItem.value?.payload?.animated
  return animated === true || Number(animated) === 1
})
const wearingTitle = computed(() => store.value.wearing?.title || null)

async function loadMine() {
  loading.value = true
  try {
    const res = await getMyDecorations()
    store.value = res.data || { types: [], wearing: {}, list: {} }
  } catch {
    store.value = { types: [], wearing: {}, list: {} }
  } finally {
    loading.value = false
  }
}

// ===== 佩戴 / 卸下 =====
const pendingTitle = ref(null)
const customTitleText = ref('')

function wear(item) {
  const blocked = blockReason(item)
  if (blocked) {
    toast.info(item.off_shelf ? '该装扮已被后台下架，暂时无法佩戴' : '该装扮已过期')
    return
  }
  // 纯样式类头衔（payload 既没有文字也没有图片）：先让用户填写文字；
  // 图片头衔（payload.image）直接佩戴，不需要用户提供文字
  if (item.type === 'title' && !item.payload?.text && !item.payload?.image) {
    pendingTitle.value = item
    customTitleText.value = item.custom_text || ''
    return
  }
  doWear(item, '')
}

function confirmWear() {
  const item = pendingTitle.value
  if (!item) return
  if (!customTitleText.value.trim()) {
    toast.warning('请输入头衔文字')
    return
  }
  doWear(item, customTitleText.value.trim())
}

async function doWear(item, text) {
  actingId.value = item.id
  try {
    await wearDecoration(item.id, text)
    toast.success(item.type === 'title' ? '头衔已佩戴' : '头像框已佩戴')
    pendingTitle.value = null
    await Promise.all([loadMine(), userStore.checkLoginState()])
  } catch {
    // 错误已由拦截器提示
  } finally {
    actingId.value = null
  }
}

async function unwear(item) {
  actingId.value = item.id
  try {
    await unwearDecoration(item.type)
    toast.success(item.type === 'title' ? '头衔已卸下' : '头像框已移除')
    await Promise.all([loadMine(), userStore.checkLoginState()])
  } catch {
    // 错误已由拦截器提示
  } finally {
    actingId.value = null
  }
}

onMounted(loadMine)
</script>

<style scoped>
.decorations-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.link {
  font-size: 13px;
  color: var(--text-muted);
  text-decoration: none;
}
.link:hover {
  color: var(--primary);
}

/* 当前佩戴 */
.wearing-card {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px 16px;
  background: linear-gradient(135deg, var(--gold-soft), var(--accent-wash));
  border: 1px solid var(--gold-line);
}
.wearing-info {
  flex: 1;
  min-width: 0;
}
.wearing-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 4px;
}
.wearing-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  min-width: 0;
}
.wearing-empty {
  font-size: 13px;
  color: var(--text-muted);
}
.wearing-name {
  font-size: 13px;
  color: var(--primary-deep);
  font-weight: 600;
}
.wearing-sep {
  color: var(--border);
}
.wearing-count {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
}
.wearing-count strong {
  font-size: 16px;
  color: #d4a148;
  font-variant-numeric: tabular-nums;
}

/* 分区 */
.decoration-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
}
.section-name {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.section-count {
  font-size: 12px;
  color: var(--text-muted);
}

.decoration-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
}
.decoration-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 0;
}
.decoration-card.is-wearing {
  border-color: var(--primary);
  box-shadow: 0 0 0 1px var(--primary-soft);
}
.decoration-card.is-expired {
  opacity: 0.72;
}
.decoration-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 92px;
  background: var(--bg-muted);
  overflow: hidden;
}
.decoration-preview.is-title {
  padding: 0 10px;
}

/* 稀有度角标：普通 / 稀有 / 史诗 / 传说（与装扮商城同一套配色） */
.rarity-badge {
  position: absolute;
  top: 6px;
  right: 6px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-muted);
}
.rarity-badge.is-rare {
  background: rgba(74, 144, 226, 0.14);
  color: #4a90e2;
}
.rarity-badge.is-epic {
  background: rgba(143, 106, 214, 0.16);
  color: #8f6ad6;
}
.rarity-badge.is-legend {
  background: var(--gold-soft);
  color: #c7902f;
}
.decoration-frame {
  width: 72px;
  height: 72px;
  object-fit: contain;
}
.decoration-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  flex: 1;
}
.decoration-name {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.decoration-name-text {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.decoration-tag {
  flex-shrink: 0;
  padding: 1px 7px;
  font-size: 11px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
}
.decoration-tag.is-wearing {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.decoration-tag.is-expired {
  background: rgba(217, 84, 77, 0.12);
  color: var(--danger);
}
.decoration-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 11px;
  color: var(--text-muted);
}
.decoration-duration {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.decoration-duration.is-urgent {
  color: var(--danger);
}
.decoration-source {
  white-space: nowrap;
}
.decoration-actions {
  margin-top: auto;
  padding-top: 2px;
  /* 佩戴 / 续期两个整行按钮：竖排留一点缝，避免贴成一块 */
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.is-loading {
  padding: 48px;
  text-align: center;
  color: var(--text-muted);
}

/* 弹窗（与商城保持一致） */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.5);
}
.modal-card {
  width: 100%;
  max-width: 380px;
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
.pending-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 56px;
  margin-bottom: 14px;
  background: var(--bg-muted);
  border-radius: var(--radius);
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
</style>
