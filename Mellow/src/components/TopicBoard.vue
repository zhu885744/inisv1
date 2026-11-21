<template>
  <div>
    <div v-if="!items.length" class="topic-board-empty">
      <EmptyState :text="emptyText" />
    </div>
    <div v-else class="topic-grid">
      <div v-for="(item, index) in items" :key="item.id" class="topic-card">
        <RouterLink v-if="!plain" :to="`/topic/${item.id}`" class="topic-card-link">
          <span v-if="showRank" class="topic-rank" :class="{ hot: index < 3 }">{{ index + 1 }}</span>
          <img v-if="item.avatar" :src="item.avatar" class="topic-card-cover" alt="" @error="onCoverError" loading="lazy" />
          <span v-else class="topic-card-cover topic-card-cover-empty">#</span>
          <span class="topic-card-main">
            <span class="topic-card-name">
              #{{ item.name }}#
              <span v-if="item.recommend" class="topic-badge">精选</span>
            </span>
            <span class="topic-card-stats">
              <span><i class="bi bi-chat-square-quote" /> {{ item.moment_count }}</span>
              <span><i class="bi bi-file-earmark-text" /> {{ item.article_count }}</span>
              <span><i class="bi bi-people" /> {{ item.follow_count }}</span>
            </span>
          </span>
        </RouterLink>

        <!-- plain：外层自己包跳转（feed 页的 tab 里整块卡片可点） -->
        <div v-else class="topic-card-link">
          <span v-if="showRank" class="topic-rank" :class="{ hot: index < 3 }">{{ index + 1 }}</span>
          <img v-if="item.avatar" :src="item.avatar" class="topic-card-cover" alt="" @error="onCoverError" loading="lazy" />
          <span v-else class="topic-card-cover topic-card-cover-empty">#</span>
          <span class="topic-card-main">
            <span class="topic-card-name">
              #{{ item.name }}#
              <span v-if="item.recommend" class="topic-badge">精选</span>
            </span>
            <span class="topic-card-stats">
              <span><i class="bi bi-chat-square-quote" /> {{ item.moment_count }}</span>
              <span><i class="bi bi-file-earmark-text" /> {{ item.article_count }}</span>
              <span><i class="bi bi-people" /> {{ item.follow_count }}</span>
            </span>
          </span>
        </div>

        <button
          class="topic-follow"
          type="button"
          :class="{ 'is-active': item.is_following }"
          :disabled="busyId === item.id"
          @click="toggleFollow(item)"
        >
          {{ item.is_following ? '已关注' : '关注' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
/**
 * 话题榜卡片列表（话题广场 /feed 的「话题」tab 共用）
 *
 * 自己处理关注 / 取关：关注状态变化后 emit('changed')，让父级刷新「我关注的」之类联动区块。
 */
import { ref } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import { followTopic, unfollowTopic } from '@/api/topics'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { storeToRefs } from 'pinia'

const props = defineProps({
  items: { type: Array, default: () => [] },
  // 是否显示名次（广场与 feed 都显示，方便看出热度）
  showRank: { type: Boolean, default: true },
  // plain：不自己包 <RouterLink>（外层已有跳转容器时用）
  plain: { type: Boolean, default: false },
  emptyText: { type: String, default: '还没有话题，去动态或文章里写 #话题# 试试' }
})

const emit = defineEmits(['changed'])

const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const busyId = ref(0)

function onCoverError(e) {
  e.target.style.display = 'none'
}

async function toggleFollow(item) {
  if (!user.value?.id) {
    toast.warning('请先登录后再关注话题')
    return
  }
  if (busyId.value) return

  busyId.value = item.id
  try {
    const wasFollowing = !!item.is_following
    const res = wasFollowing ? await unfollowTopic(item.id) : await followTopic(item.id)
    item.is_following = !wasFollowing
    if (res?.data?.follow_count !== undefined) {
      item.follow_count = res.data.follow_count
    }
    // 计数已经变了，但 props 是父级数据，这里显式通知父级刷新
    emit('changed', { id: item.id, is_following: item.is_following, follow_count: item.follow_count })
  } catch (e) {
    toast.error(e?.msg || '操作失败，请稍后重试')
  } finally {
    busyId.value = 0
  }
}
</script>

<style scoped>
.topic-board-empty {
  padding: 20px 0;
}
.topic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.topic-card {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 14px 14px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  transition: border-color 0.18s, box-shadow 0.18s;
}
.topic-card:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow-sm);
}
.topic-card-link {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
  color: inherit;
  text-decoration: none;
}
.topic-rank {
  width: 22px;
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--text-light);
  text-align: center;
}
.topic-rank.hot {
  color: var(--warning);
}
.topic-card-cover {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.topic-card-cover-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  font-size: 20px;
  font-weight: 700;
}
.topic-card-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.topic-card-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.topic-badge {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--gold-wash);
  color: var(--warning);
  font-size: 11px;
  font-weight: 500;
}
.topic-card-stats {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: var(--text-muted);
}
.topic-card-stats .bi {
  margin-right: 3px;
}
.topic-follow {
  flex-shrink: 0;
  padding: 5px 12px;
  border: 1px solid var(--primary-soft);
  border-radius: 999px;
  background: transparent;
  color: var(--primary);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.18s;
}
.topic-follow:hover {
  background: var(--accent-soft);
}
.topic-follow.is-active {
  border-color: var(--border);
  color: var(--text-muted);
}
.topic-follow:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
