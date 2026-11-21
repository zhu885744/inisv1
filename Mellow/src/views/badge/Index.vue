<template>
  <div class="badge-hall">
    <!-- 头部：全站统计 + 入口 -->
    <section class="card card-pad hall-head">
      <div>
        <h1 class="hall-title">勋章馆</h1>
        <p class="hall-sub">
          共 {{ stats.badges || 0 }} 枚勋章 · 全站已点亮 {{ stats.held || 0 }} 次 · 正在佩戴
          {{ stats.wearers || 0 }} 人次
        </p>
      </div>
      <router-link v-if="isLogged" to="/user/badges" class="btn btn-primary btn-sm">
        <i class="bi bi-bag-check" /> 我的勋章
      </router-link>
    </section>

    <!-- 分类 -->
    <div v-if="categories.length" class="category-tabs">
      <button type="button" class="category-tab" :class="{ 'is-active': activeCat === null }" @click="activeCat = null">
        <i class="bi bi-grid-3x3-gap" /> 全部
        <span class="category-tab-count">{{ stats.badges || 0 }}</span>
      </button>
      <button
        v-for="cat in categories"
        :key="cat.id"
        type="button"
        class="category-tab"
        :class="{ 'is-active': activeCat === cat.id }"
        @click="activeCat = activeCat === cat.id ? null : cat.id"
      >
        <i :class="cat.icon || 'bi bi-collection'" :style="{ color: cat.color || undefined }" />
        {{ cat.name }}
        <span class="category-tab-count">{{ cat.total }}</span>
      </button>
    </div>

    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>

    <!-- 按稀有度分组的荣誉墙 -->
    <template v-else>
      <section v-for="group in rarityGroups" :key="group.key" class="rarity-group">
        <header class="rarity-head">
          <span class="rarity-dot" :class="`is-${group.key}`" />
          <h2 class="rarity-title">{{ group.name }}</h2>
          <span class="rarity-count">{{ group.list.length }} 枚</span>
        </header>

        <div class="badge-grid">
          <article
            v-for="item in group.list"
            :key="item.id"
            class="badge-card"
            :class="[`rarity-${item.rarity || 'common'}`, { 'is-held': item.held }]"
          >
            <span class="badge-orb" :style="orbStyle(item)">
              <img v-if="item.image" :src="item.image" :alt="item.name" loading="lazy" />
              <i v-else :class="item.icon || 'bi bi-award'" />
            </span>

            <div class="badge-body">
              <div class="badge-title">
                <strong class="badge-name">{{ item.name }}</strong>
                <span v-if="item.held" class="badge-owned" title="你已获得"><i class="bi bi-check-circle-fill" /></span>
              </div>

              <p class="badge-desc">{{ item.description || item.condition_name }}</p>

              <div class="badge-meta">
                <span><i class="bi bi-people" /> {{ item.held_count }} 人获得</span>
                <span v-if="item.wear_count"><i class="bi bi-pin-angle" /> {{ item.wear_count }} 人佩戴</span>
                <span v-if="!item.available" class="badge-off">已停止产出</span>
              </div>

              <!-- 佩戴榜：此刻正戴着这枚勋章的用户 -->
              <div v-if="item.wearers && item.wearers.length" class="wearers">
                <router-link
                  v-for="wearer in item.wearers"
                  :key="wearer.uid"
                  :to="`/author/${wearer.uid}`"
                  class="wearer"
                  :title="wearer.nickname"
                >
                  <img :src="wearer.avatar || defaultAvatar" :alt="wearer.nickname" loading="lazy" />
                </router-link>
                <span class="wearers-hint">正在佩戴</span>
              </div>
            </div>
          </article>
        </div>
      </section>

      <div v-if="!rarityGroups.length" class="hall-empty">
        <i class="bi bi-award" />
        <p>这个分类下还没有勋章</p>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getBadgeWall } from '@/api/badge'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const isLogged = computed(() => userStore.isLogged)

const loading = ref(false)
const list = ref([])
const categories = ref([])
const stats = ref({ badges: 0, held: 0, wearers: 0 })
// null = 全部分类
const activeCat = ref(null)

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/></svg>'

const RARITY_ORDER = ['legendary', 'epic', 'rare', 'common']
const RARITY_NAME = { legendary: '传说', epic: '史诗', rare: '稀有', common: '普通' }

// 分类筛选在前，稀有度分组在后：荣誉墙的阅读顺序是「先看是哪一类，再由高到低」
const filtered = computed(() =>
  activeCat.value === null
    ? list.value
    : list.value.filter((item) => Number(item.category_id) === Number(activeCat.value))
)

const rarityGroups = computed(() =>
  RARITY_ORDER.map((key) => ({
    key,
    name: RARITY_NAME[key],
    list: filtered.value.filter((item) => (item.rarity || 'common') === key)
  })).filter((group) => group.list.length)
)

function orbStyle(item) {
  const color = item.color || 'var(--warning)'
  const hex = /^#[0-9a-f]{6}$/i.test(color)
  return {
    color,
    borderColor: hex ? `${color}59` : 'var(--border)',
    background: hex ? `${color}1f` : 'var(--bg-muted)'
  }
}

async function load() {
  loading.value = true
  try {
    const res = await getBadgeWall()
    const data = res.data || {}
    list.value = data.data || []
    categories.value = data.categories || []
    stats.value = data.stats || { badges: 0, held: 0, wearers: 0 }
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.badge-hall {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hall-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.hall-title {
  margin: 0;
  font-size: 20px;
  font-weight: 700;
}
.hall-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}

/* 分类胶囊（与「我的勋章」页保持一致的视觉） */
.category-tabs {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 2px;
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

/* 稀有度分组 */
.rarity-group + .rarity-group {
  margin-top: 18px;
}
.rarity-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.rarity-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-light);
}
.rarity-dot.is-rare {
  background: #4f9dff;
  box-shadow: 0 0 6px rgba(79, 157, 255, 0.6);
}
.rarity-dot.is-epic {
  background: #a06bff;
  box-shadow: 0 0 6px rgba(160, 107, 255, 0.6);
}
.rarity-dot.is-legendary {
  background: #f0b429;
  box-shadow: 0 0 8px rgba(240, 180, 41, 0.75);
}
.rarity-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.rarity-count {
  font-size: 11px;
  color: var(--text-light);
}

/* 卡片：两列（窄屏一列） */
.badge-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
@media (max-width: 720px) {
  .badge-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

.badge-card {
  display: flex;
  gap: 12px;
  padding: 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  transition: border-color 0.15s, transform 0.15s;
}
.badge-card:hover {
  border-color: var(--border);
  transform: translateY(-1px);
}
.badge-card.is-held {
  border-color: var(--primary);
}
.badge-card.rarity-rare {
  box-shadow: inset 0 0 0 1px rgba(79, 157, 255, 0.25);
}
.badge-card.rarity-epic {
  box-shadow: inset 0 0 0 1px rgba(160, 107, 255, 0.3);
}
.badge-card.rarity-legendary {
  box-shadow: inset 0 0 0 1px rgba(240, 180, 41, 0.4), 0 0 14px rgba(240, 180, 41, 0.12);
}

.badge-orb {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  border: 1px solid var(--border);
  border-radius: 12px;
  font-size: 22px;
  overflow: hidden;
}
.badge-orb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.badge-body {
  flex: 1;
  min-width: 0;
}
.badge-title {
  display: flex;
  align-items: center;
  gap: 6px;
}
.badge-name {
  font-size: 14px;
  color: var(--text);
}
.badge-owned {
  font-size: 12px;
  color: var(--primary);
}
.badge-desc {
  margin: 3px 0 6px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.badge-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 11px;
  color: var(--text-light);
}
.badge-off {
  color: var(--warning, #e08b00);
}

.wearers {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
}
.wearer {
  display: inline-flex;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  overflow: hidden;
  border: 1px solid var(--bg-card);
  box-shadow: 0 0 0 1px var(--border-soft);
}
.wearer + .wearer {
  margin-left: -8px;
}
.wearer img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.wearers-hint {
  margin-left: 6px;
  font-size: 11px;
  color: var(--text-light);
}

.hall-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 0;
  color: var(--text-light);
}
.hall-empty .bi {
  font-size: 28px;
}
.hall-empty p {
  margin: 0;
  font-size: 13px;
}
</style>
