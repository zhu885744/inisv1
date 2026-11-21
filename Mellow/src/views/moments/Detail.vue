<template>
  <div class="moment-detail-page">
    <div v-if="loading" class="state-tip">加载中…</div>
    <div v-else-if="!moment" class="state-tip empty">动态不存在或已被删除</div>
    <MomentItem
      v-else
      :moment="moment"
      :highlight-id="route.query.comment"
      :force-open="true"
      @delete="onDeleted"
      @refresh="load"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import MomentItem from '@/components/MomentItem.vue'
import { getMoment } from '@/api/moments'

const props = defineProps({
  id: { type: [String, Number], required: true }
})
const route = useRoute()
const router = useRouter()

const moment = ref(null)
const loading = ref(true)

async function load() {
  loading.value = true
  try {
    const res = await getMoment(props.id)
    moment.value = res?.data?.data || res?.data || null
  } catch {
    moment.value = null
  } finally {
    loading.value = false
  }
}

function onDeleted() {
  // 动态已被删除，详情页无内容可留：回首页（列表页已下线，不再有 /moments 可回）
  router.replace({ name: 'home' })
}

onMounted(load)
</script>

<style scoped>
.back-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
}
.back-link:hover {
  color: var(--primary);
}
.state-tip {
  padding: 40px 0;
  text-align: center;
  color: var(--text-muted);
  font-size: 14px;
}
.state-tip.empty {
  color: var(--text-soft);
}
</style>
