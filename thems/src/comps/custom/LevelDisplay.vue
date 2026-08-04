<template>
  <div class="level-display">
    <!-- 我的等级卡片 -->
    <div class="level-card level-my">
      <div class="level-card-header">
        <i class="bi bi-person-badge"></i>
        <span>我的等级</span>
      </div>
      <div class="level-card-body">
        <div class="level-badge">
          <span class="level-badge-label">LV</span>
          <span class="level-badge-num">{{ currentLevel?.value || 1 }}</span>
        </div>
        <div class="level-info">
          <div class="level-name">{{ currentLevel?.name || '凡人' }}</div>
          <div class="level-exp-line">
            <span class="exp-current">{{ currentExp }}/{{ nextLevelExp }}</span>
            <span v-if="nextLevelExp > currentExp" class="exp-tip">升级还需 {{ nextLevelExp - currentExp }} 经验</span>
            <span v-else class="exp-tip exp-max">已达最高境界</span>
          </div>
        </div>
      </div>
      <div class="level-progress">
        <div class="level-progress-bar" :style="{ width: expPercent + '%' }"></div>
      </div>
    </div>

    <!-- 成长体系卡片 -->
    <div class="level-card level-system">
      <div class="level-card-header">
        <i class="bi bi-stars"></i>
        <span>成长体系</span>
        <span class="level-exp-tag">我的经验值：{{ currentExp }}</span>
      </div>
      <div class="level-card-body">
        <!-- 时间轴 -->
        <div class="level-timeline">
          <div
            v-for="(level, idx) in levels"
            :key="level.id"
            class="level-timeline-item"
            :class="{
              current: isCurrentLevel(level),
              passed: isLevelPassed(level),
              future: !isCurrentLevel(level) && !isLevelPassed(level)
            }"
          >
            <div class="level-timeline-milestone">
              <div class="milestone-exp">{{ level.exp }}</div>
              <div class="milestone-dot"></div>
            </div>
            <div class="level-timeline-content">
              <div class="level-chip" :class="{ active: isCurrentLevel(level) }">
                <span class="chip-label">LV{{ level.value }}</span>
                <span class="chip-name">{{ level.name }}</span>
              </div>
            </div>
            <div v-if="idx < levels.length - 1" class="level-connector"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 关闭按钮 -->
    <button class="level-close-btn" @click="$emit('close')">
      <i class="bi bi-x-lg"></i>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { request } from '@/utils/network'

const props = defineProps({
  currentLevel: {
    type: Object,
    default: () => ({})
  },
  currentExp: {
    type: Number,
    default: 0
  },
  nextLevelExp: {
    type: Number,
    default: 0
  }
})

defineEmits(['close'])

const levels = ref([])
const loading = ref(false)

const expPercent = computed(() => {
  const current = props.currentExp
  const base = props.currentLevel?.exp || 0
  const target = props.nextLevelExp || base
  if (target <= base) return 100
  return Math.min(100, Math.max(0, Math.round(((current - base) / (target - base)) * 100)))
})

const sortedLevels = computed(() => {
  return [...levels.value].sort((a, b) => (a.exp || 0) - (b.exp || 0))
})

const isLevelPassed = (level) => {
  return props.currentExp >= (level.exp || 0)
}

const isCurrentLevel = (level) => {
  const lv = props.currentLevel
  if (!lv || !lv.value) return false
  return Number(level.value) === Number(lv.value)
}

const loadLevels = async () => {
  if (levels.value.length > 0) return
  loading.value = true
  try {
    const res = await request.get('/api/level/all', {
      page: 1,
      limit: 9999,
      order: 'exp asc'
    })
    if (res.code === 200 && res.data?.data) {
      levels.value = res.data.data
    }
  } catch (err) {
    console.error('获取等级列表失败:', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadLevels()
})
</script>

<style scoped>
.level-display {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1.5fr;
  gap: 16px;
  padding: 8px;
}

/* 关闭按钮 */
.level-close-btn {
  position: absolute;
  top: 0;
  right: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: var(--bs-secondary-bg);
  color: var(--bs-secondary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--wx-transition);
  z-index: 10;
}

.level-close-btn:hover {
  background: var(--bs-secondary-bg-subtle);
  color: var(--bs-body-color);
  transform: rotate(90deg);
}

.level-close-btn i {
  font-size: 14px;
}

/* 卡片基础样式 */
.level-card {
  border-radius: var(--wx-radius-lg);
  padding: 20px;
  position: relative;
  overflow: hidden;
}

.level-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
  margin-bottom: 16px;
}

.level-card-header i {
  font-size: 18px;
}

/* 我的等级卡片 */
.level-my {
  background: linear-gradient(135deg, #f5e6c8 0%, #e8d4a8 50%, #d4b88a 100%);
  color: #5a4a30;
  box-shadow: var(--wx-shadow-md);
}

.level-my::before {
  content: '';
  position: absolute;
  top: -30px;
  right: -30px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.25);
}

.level-my::after {
  content: '';
  position: absolute;
  bottom: -40px;
  left: -20px;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.15);
}

.level-my .level-card-header {
  color: #5a4a30;
}

.level-my .level-card-body {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  position: relative;
  z-index: 1;
}

.level-badge {
  display: flex;
  align-items: baseline;
  gap: 2px;
  flex-shrink: 0;
}

.level-badge-label {
  font-size: 12px;
  font-weight: 600;
  color: #7a6a50;
}

.level-badge-num {
  font-size: 36px;
  font-weight: 700;
  color: #6b5530;
  line-height: 1;
}

.level-info {
  flex: 1;
}

.level-name {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 6px;
  color: #4a3a20;
}

.level-exp-line {
  font-size: 13px;
  color: #7a6a50;
  line-height: 1.5;
}

.level-exp-line .exp-tip {
  display: block;
}

.level-exp-line .exp-max {
  color: #b8860b;
  font-weight: 600;
}

.level-progress {
  height: 8px;
  background: rgba(255, 255, 255, 0.4);
  border-radius: 4px;
  overflow: hidden;
  position: relative;
  z-index: 1;
}

.level-progress-bar {
  height: 100%;
  background: linear-gradient(90deg, #b8860b, #daa520);
  border-radius: 4px;
  transition: width 0.5s ease;
}

/* 成长体系卡片 */
.level-system {
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  box-shadow: var(--wx-shadow-sm);
}

.level-system .level-card-header {
  color: var(--bs-body-color);
}

.level-exp-tag {
  margin-left: auto;
  font-size: 13px;
  font-weight: 400;
  color: var(--bs-secondary-color);
  padding: 4px 10px;
  background: var(--bs-secondary-bg);
  border-radius: var(--wx-radius-md);
}

/* 时间轴 */
.level-timeline {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0;
  padding: 0 8px;
  overflow-x: auto;
}

.level-timeline-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  min-width: 80px;
  position: relative;
}

.level-timeline-milestone {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
  padding-bottom: 8px;
}

.milestone-exp {
  font-size: 11px;
  color: var(--bs-secondary-color);
  margin-bottom: 6px;
  white-space: nowrap;
}

.milestone-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--bs-tertiary-color);
  border: 3px solid var(--bs-tertiary-bg);
  transition: var(--wx-transition);
  position: relative;
  z-index: 2;
}

.level-timeline-item.passed .milestone-dot {
  background: #b8860b;
  border-color: #daa520;
}

.level-timeline-item.current .milestone-dot {
  background: #daa520;
  border-color: #ffd700;
  box-shadow: 0 0 0 4px rgba(218, 165, 32, 0.25);
  width: 16px;
  height: 16px;
}

.level-timeline-item.future .milestone-dot {
  background: var(--bs-secondary-color);
  border-color: var(--bs-secondary-bg);
  opacity: 0.5;
}

.level-timeline-content {
  margin-top: 4px;
}

.level-chip {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 10px;
  border-radius: var(--wx-radius-sm);
  background: var(--bs-secondary-bg);
  transition: var(--wx-transition);
  min-width: 60px;
}

.level-chip.active {
  background: linear-gradient(135deg, #daa520, #b8860b);
  color: #fff;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(218, 165, 32, 0.35);
}

.level-timeline-item.passed .level-chip {
  background: rgba(184, 134, 11, 0.15);
  color: #b8860b;
}

.level-timeline-item.future .level-chip {
  opacity: 0.6;
}

.chip-label {
  font-size: 11px;
  font-weight: 600;
}

.chip-name {
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

/* 连接线 */
.level-connector {
  position: absolute;
  top: 7px;
  left: calc(50% + 10px);
  right: calc(-50% + 10px);
  height: 2px;
  background: var(--bs-border-color);
  z-index: 1;
}

.level-timeline-item.passed + .level-timeline-item .level-connector,
.level-connector {
  /* connector is after the item */
}

/* 响应式 */
@media (max-width: 768px) {
  .level-display {
    grid-template-columns: 1fr;
  }

  .level-timeline {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .level-my .level-card-body {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
