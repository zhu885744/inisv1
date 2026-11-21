<template>
  <div class="topics-page">
    <SectionTitle title="话题广场">
      <template #extra>
        <span class="text-muted">{{ keyword ? `搜索到 ${list.length} 个` : `热度前 ${list.length} 个` }}</span>
      </template>
    </SectionTitle>

    <div class="topic-search">
      <i class="bi bi-search" />
      <FormInput
        v-model="keyword"
        type="search"
        placeholder="搜索话题，如：旅行 / 美食 / 摄影"
        @keyup.enter="search" />
      <button v-if="keyword" type="button" title="清空" @click="clearSearch">
        <i class="bi bi-x-lg" />
      </button>
      <button class="topic-search-btn" type="button" @click="search">搜索</button>
    </div>

    <!-- 我关注的话题 -->
    <section v-if="mine.length" class="topic-mine">
      <div class="topic-block-label"><i class="bi bi-star" /> 我关注的</div>
      <div class="topic-chips">
        <RouterLink v-for="item in mine" :key="item.id" :to="`/topic/${item.id}`" class="topic-chip">
          <span class="topic-chip-hash">#</span>{{ item.name }}
          <span class="topic-chip-count">{{ item.moment_count }}</span>
        </RouterLink>
      </div>
    </section>

    <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>
    <TopicBoard
      v-else
      :items="list"
      :empty-text="keyword ? '没有找到相关话题' : '还没有话题，去动态或文章里写 #话题# 试试'"
      @changed="loadMine"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * 话题广场 /topics
 *
 * - 热门榜：topics/board 按热度（文章×3 + 动态×2 + 关注×2）排序
 * - 我关注的：topics/mine（未登录不展示）
 * - 搜索：keyword 传给 board，按话题名模糊匹配
 * 榜单卡片（含关注按钮）由 TopicBoard 渲染，/feed 的「话题」tab 复用同一组件。
 */
import { ref, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import SectionTitle from '@/components/SectionTitle.vue'
import TopicBoard from '@/components/TopicBoard.vue'
import { listTopics, listMyTopics } from '@/api/topics'
import { useUserStore } from '@/stores/user'

const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const loading = ref(true)
const keyword = ref('')
const list = ref([])
const mine = ref([])

async function loadBoard() {
  loading.value = true
  try {
    const res = await listTopics({ keyword: keyword.value.trim(), limit: 50 })
    list.value = res?.data || []
  } catch {
    list.value = []
  } finally {
    loading.value = false
  }
}

async function loadMine() {
  if (!user.value?.id) {
    mine.value = []
    return
  }
  try {
    const res = await listMyTopics()
    mine.value = res?.data || []
  } catch {
    mine.value = []
  }
}

function search() {
  loadBoard()
}

function clearSearch() {
  keyword.value = ''
  loadBoard()
}

onMounted(() => {
  loadBoard()
  loadMine()
})
</script>

<style scoped>
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}

.topic-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
  padding: 0 10px 0 14px;
  height: 40px;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-light);
}
.topic-search:focus-within {
  border-color: var(--primary-soft);
}
.topic-search input {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  outline: none;
  color: var(--text);
  font-size: 14px;
  font-family: inherit;
}
.topic-search input::-webkit-search-cancel-button {
  display: none;
}
.topic-search button {
  border: none;
  background: transparent;
  color: var(--text-light);
  font-size: 12px;
  cursor: pointer;
}
.topic-search-btn {
  padding: 6px 14px;
  border-radius: 999px !important;
  background: var(--accent-soft) !important;
  color: var(--primary) !important;
  font-size: 13px;
  font-weight: 600;
}

.topic-mine {
  margin-bottom: 22px;
}
.topic-block-label {
  margin-bottom: 10px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
}
.topic-block-label .bi {
  color: var(--primary);
  margin-right: 4px;
}
.topic-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.topic-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-soft);
  font-size: 13px;
  text-decoration: none;
  transition: all 0.18s;
}
.topic-chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.topic-chip-hash {
  color: var(--primary);
}
.topic-chip-count {
  font-size: 11px;
  color: var(--text-light);
}
</style>
