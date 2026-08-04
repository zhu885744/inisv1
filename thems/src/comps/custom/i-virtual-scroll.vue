<template>
  <div
    ref="containerRef"
    class="virtual-scroll-container wx-scrollable"
    @scroll="handleScroll"
    :style="{ height: containerHeight + 'px' }"
  >
    <div
      class="virtual-scroll-content"
      :style="{ height: totalHeight + 'px' }"
    >
      <div
        v-for="item in visibleItems"
        :key="getKey(item)"
        class="virtual-scroll-item"
        :style="{ transform: `translateY(${getItemOffset(item)}px)` }"
      >
        <slot :item="item" />
      </div>
    </div>
  </div>
</template>

<script setup>import { ref, computed } from 'vue';
const props = defineProps({
 items: {
 type: Array,
 default: () => []
 },
 itemHeight: {
 type: Number,
 default: 100
 },
 containerHeight: {
 type: Number,
 default: 600
 },
 keyField: {
 type: String,
 default: 'id'
 }
});
const containerRef = ref(null);
const scrollTop = ref(0);
const totalHeight = computed(() => props.items.length * props.itemHeight);
const startIndex = computed(() => Math.max(0, Math.floor(scrollTop.value / props.itemHeight) - 2));
const endIndex = computed(() => {
 const containerItems = Math.ceil(props.containerHeight / props.itemHeight);
 return Math.min(props.items.length, startIndex.value + containerItems + 4);
});
const visibleItems = computed(() => props.items.slice(startIndex.value, endIndex.value));
const getItemOffset = (item) => {
 const index = props.items.findIndex(i => i[props.keyField] === item[props.keyField]);
 return index * props.itemHeight;
};
const getKey = (item) => item[props.keyField] || props.items.indexOf(item);
const handleScroll = () => {
 if (containerRef.value) {
 scrollTop.value = containerRef.value.scrollTop;
 }
};
</script>

<style scoped>
.virtual-scroll-container {
  overflow-y: auto;
  position: relative;
  width: 100%;
}

.virtual-scroll-content {
  position: relative;
  width: 100%;
}

.virtual-scroll-item {
  position: absolute;
  width: 100%;
}
</style>