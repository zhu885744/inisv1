<template>
  <svg :class="svgClass" :style="svgStyle" :width="size" :height="size" :viewBox="viewBox">
    <use :xlink:href="iconPath"></use>
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  icon: {
    type: String,
    required: true
  },
  size: {
    type: [Number, String],
    default: 20
  },
  color: {
    type: String,
    default: 'currentColor'
  },
  className: {
    type: String,
    default: ''
  }
})

const iconPath = computed(() => `#icon-${props.icon}`)
const svgClass = computed(() => ['svg-icon', props.className])
const svgStyle = computed(() => ({ color: props.color }))
const viewBox = computed(() => '0 0 1024 1024')
</script>

<style scoped>
.svg-icon {
  display: inline-block;
  vertical-align: middle;
  fill: none;
}
</style>