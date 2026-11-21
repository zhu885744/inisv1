<template>
  <div class="line-chart" :style="{ height: `${height}px` }">
    <svg class="chart-svg" :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" role="img" :aria-label="ariaLabel">
      <!-- 横向网格 -->
      <line
        v-for="y in grid"
        :key="`grid-${y}`"
        :x1="0"
        :x2="W"
        :y1="y"
        :y2="y"
        class="chart-grid"
        vector-effect="non-scaling-stroke"
      />
      <!-- 面积（仅第一条曲线） -->
      <path v-if="area" :d="area" :fill="firstColor" fill-opacity="0.12" />
      <!-- 曲线 -->
      <path
        v-for="item in paths"
        :key="item.name"
        :d="item.d"
        :stroke="item.color"
        class="chart-line"
        fill="none"
        vector-effect="non-scaling-stroke"
      />
    </svg>

    <div v-if="!hasData" class="chart-empty">等待数据…</div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

/**
 * 极简折线图（纯 SVG，无第三方依赖）
 *
 * 为什么不用图表库：项目本身没有引入任何图表依赖，这里只需要「最近 N 个采样点的走势」，
 * 手写 SVG 即可，避免为后台一个页面增加打包体积。
 *
 * 实现要点：
 *   - viewBox 固定 100×100 + preserveAspectRatio="none"，横纵随便拉伸；
 *   - 拉伸会让线宽变形，所以每条线都加 vector-effect="non-scaling-stroke" 保持 1px；
 *   - 按 capacity 固定横轴刻度：数据不足时从左往右长，满了之后整体左移（滑动窗口）。
 */
const props = defineProps({
  // [{ name, color, values: number[] }]
  series: { type: Array, default: () => [] },
  height: { type: Number, default: 72 },
  // 纵向固定上限（如 CPU 固定 100）；不传则按数据自适应该
  max: { type: Number, default: 0 },
  // 横轴固定槽位数（= 观察窗口长度，配合服务端 1 秒一条 = 最近 N 秒）
  capacity: { type: Number, default: 60 },
  // 是否绘制面积填充（只作用于第一条曲线）
  fill: { type: Boolean, default: true }
})

const W = 100
const H = 100

const hasData = computed(() => props.series.some((item) => (item.values || []).length >= 2))
const firstColor = computed(() => props.series[0]?.color || 'var(--primary)')

const ariaLabel = computed(() =>
  props.series.map((item) => `${item.name} 走势`).join('，') || '实时走势'
)

const grid = computed(() => [H, H / 2, 0])

/** 纵轴上限：显式传入优先，否则取数据最大值的 1.2 倍（至少 1，避免除以 0） */
const upper = computed(() => {
  if (props.max > 0) return props.max
  let top = 0
  props.series.forEach((item) => {
    ;(item.values || []).forEach((value) => {
      if (Number.isFinite(value) && value > top) top = value
    })
  })
  return top > 0 ? top * 1.2 : 1
})

/** 把数值序列映射成 SVG 路径（横轴按 capacity 固定，纵向 0 在下、上限在上） */
function toPath(values = []) {
  const list = values.slice(-props.capacity)
  if (list.length < 1) return ''
  const step = props.capacity > 1 ? W / (props.capacity - 1) : W

  return list
    .map((value, index) => {
      const x = index * step
      const safe = Number.isFinite(value) ? Math.max(0, value) : 0
      const y = H - Math.min(1, safe / upper.value) * H
      return `${index === 0 ? 'M' : 'L'}${x.toFixed(2)},${y.toFixed(2)}`
    })
    .join(' ')
}

const paths = computed(() =>
  props.series.map((item) => ({
    name: item.name || 'series',
    color: item.color || 'var(--primary)',
    d: toPath(item.values)
  }))
)

/** 面积：第一条曲线 + 右下两个角闭合 */
const area = computed(() => {
  if (!props.fill) return ''
  const line = toPath(props.series[0]?.values || [])
  if (!line) return ''
  const list = (props.series[0]?.values || []).slice(-props.capacity)
  const step = props.capacity > 1 ? W / (props.capacity - 1) : W
  const endX = ((list.length - 1) * step).toFixed(2)
  return `${line} L${endX},${H} L0,${H} Z`
})
</script>

<style scoped>
.line-chart {
  position: relative;
  width: 100%;
}
.chart-svg {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}
.chart-grid {
  stroke: var(--border);
  stroke-width: 1;
  stroke-dasharray: 2 4;
  opacity: 0.6;
}
.chart-line {
  stroke-width: 1.4;
  stroke-linejoin: round;
  stroke-linecap: round;
}
.chart-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--text-light, var(--text-muted));
}
</style>
