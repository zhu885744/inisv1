<template>
  <div class="wx-moment">
    <div class="wx-moment-inner">
      <!-- 左侧头像 -->
      <div class="wx-avatar-wrap">
        <router-link :to="`/author/${authorId}`" class="wx-avatar-link">
          <i-avatar-frame
            :src="authorAvatar"
            :frame="authorFrame"
            :alt="authorName"
            size="2.625rem"
            :frame-scale="1.6"
            :rounded="false"
          />
        </router-link>
      </div>

      <!-- 右侧内容 -->
      <div class="wx-content">
        <!-- 昵称 + 等级 + 头衔 -->
        <div class="wx-nickname-row">
          <router-link
            :to="`/author/${authorId}`"
            class="wx-nickname"
          >{{ authorName }}</router-link>
          <span v-if="Number(moment.top) === 1" class="badge text-bg-warning wx-top-badge">
            <i class="bi bi-pin-angle-fill"></i> 置顶
          </span>
          <span v-if="authorLevel" class="badge bg-primary text-white rounded-pill wx-level-badge">Lv.{{ authorLevel }} {{ authorLevelName }}</span>
          <span v-if="authorTitle" class="badge rounded-pill title-badge wx-title-badge" :class="getTitleColorClass(authorTitle)">
            <i class="bi bi-person-badge"></i> {{ authorTitle }}
          </span>
        </div>

        <!-- 动态内容 -->
        <div class="wx-text" @click="goDetail" v-html="renderContent(moment.content)"></div>

        <!-- 图片 -->
        <div v-if="imageList.length > 0" class="wx-images" :class="'img-grid-' + getGridType(imageList.length)">
          <div
            v-for="(img, idx) in imageList"
            :key="idx"
            class="wx-img-wrap"
            @click.stop="openImage(idx)"
          >
            <img :src="img" class="wx-img" />
          </div>
        </div>

        <!-- 位置 -->
        <div v-if="moment.location" class="wx-location">
          <i class="bi bi-geo-alt"></i>{{ moment.location }}
        </div>

        <!-- 时间 + 浏览量 + 点赞数 + 更多按钮 -->
        <div class="wx-meta">
          <div class="wx-meta-info">
            <span class="wx-time">{{ formatTime(moment.create_time) }}</span>
            <span class="wx-meta-count wx-meta-views">
              <i class="bi bi-eye"></i>{{ Number(moment.views) || 0 }}
            </span>
            <span v-if="likeCount > 0" class="wx-meta-count">
              <i class="bi bi-heart-fill"></i>{{ likeCount }}
            </span>
          </div>
          <div class="wx-more-wrap">
            <button
              class="wx-more-btn"
              @click.stop="toggleActionPanel"
            >
              <i class="bi bi-three-dots"></i>
            </button>

            <!-- 操作面板 -->
            <div
              v-if="showActionPanel"
              class="wx-action-panel"
              @click.stop
            >
              <button
                class="wx-action-item"
                :class="{ active: isLiked }"
                @click="handleLike"
              >
                <i :class="isLiked ? 'bi bi-heart-fill' : 'bi bi-heart'"></i>
                <span>赞</span>
                <span v-if="likeCount > 0" class="wx-action-count">{{ likeCount }}</span>
              </button>
              <div class="wx-action-divider"></div>
              <button
                class="wx-action-item"
                @click="handleToggleComments"
              >
                <i class="bi bi-chat"></i>
                <span>评论</span>
                <span v-if="commentCount > 0" class="wx-action-count">{{ commentCount }}</span>
              </button>
              <template v-if="isOwner">
                <div class="wx-action-divider"></div>
                <button class="wx-action-item" @click="handleEdit">
                  <i class="bi bi-pencil"></i>
                  <span>编辑</span>
                </button>
                <div class="wx-action-divider"></div>
                <button class="wx-action-item danger" @click="handleDelete">
                  <i class="bi bi-trash"></i>
                  <span>删除</span>
                </button>
              </template>
              <template v-if="isAdmin">
                <div class="wx-action-divider"></div>
                <button
                  v-if="Number(moment.top) === 1"
                  class="wx-action-item"
                  @click="handleSetTop(0)"
                >
                  <i class="bi bi-pin-angle"></i>
                  <span>取消置顶</span>
                </button>
                <button
                  v-else
                  class="wx-action-item"
                  @click="handleSetTop(1)"
                >
                  <i class="bi bi-pin-angle-fill"></i>
                  <span>置顶</span>
                </button>
              </template>
            </div>
          </div>
        </div>

        <!-- 草稿标识 -->
        <span v-if="moment.status === 0" class="wx-draft-tag">草稿</span>
      </div>
    </div>

    <!-- 评论区（内嵌） -->
    <div v-if="showComments" class="wx-comments-section">
      <CommentList
        :moment-id="moment.id"
        :is-login="isLogin"
        :is-dark-mode="isDarkMode"
        @comment-added="onCommentAdded"
        ref="commentListRef"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@/utils/network'
import { useCommStore } from '@/store/comm'
import { toast, getTitleColorClass } from '@/utils/app'
import utils from '@/utils/utils'
import CommentList from './CommentList.vue'
import iAvatarFrame from '@/comps/custom/i-avatar-frame.vue'

const props = defineProps({
  moment: {
    type: Object,
    required: true
  },
  isLogin: {
    type: Boolean,
    default: false
  },
  isDarkMode: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['edit', 'delete', 'toggleComments', 'commentAdded', 'setTop'])

const router = useRouter()
const store = useCommStore()
const commentListRef = ref(null)

const isLiked = ref(false)
const isCollected = ref(false)
const likeCount = ref(0)
const collectCount = ref(0)
const commentCount = ref(0)
const showActionPanel = ref(false)
const showComments = ref(true)

const authorId = computed(() => props.moment.result?.author?.id || props.moment.uid)
const authorName = computed(() => props.moment.result?.author?.nickname || '匿名用户')
const authorAvatar = computed(() => props.moment.result?.author?.avatar || 'https://picsum.photos/80/80')
const authorLevel = computed(() => {
  const val = props.moment.result?.author?.result?.level?.current?.value
  return val != null ? val : null
})
const authorLevelName = computed(() => props.moment.result?.author?.result?.level?.current?.name || '')
const authorTitle = computed(() => props.moment.result?.author?.title || '')
const authorFrame = computed(() => props.moment.result?.author?.json?.frame || '')

const isOwner = computed(() => {
  if (!props.isLogin) return false
  const user = store.login?.user
  return user && String(user.id) === String(authorId.value)
})

const isAdmin = computed(() => {
  if (!props.isLogin) return false
  const user = store.login?.user
  if (!user) return false
  if (user?.result?.auth?.all) return true
  const groups = user?.result?.auth?.group?.list || user?.group?.list || []
  return groups.some(group => group.key === 'admin')
})

const imageList = computed(() => {
  const images = props.moment.images
  if (!images) return []
  if (Array.isArray(images)) return images.filter(Boolean)
  if (typeof images === 'string') return images.split(',').map(s => s.trim()).filter(Boolean)
  return []
})

const getGridType = (count) => {
  if (count === 1) return '1'
  if (count <= 3) return '3'
  if (count <= 4) return '4'
  return '9'
}

const formatTime = (timestamp) => {
  if (!timestamp) return ''
  return utils.natureTime(timestamp, 5)
}

// 渲染动态内容：将 [emoji:url] 转为图片
const renderContent = (content) => {
  if (!content) return ''
  let processed = content
  processed = processed.replace(/\[emoji:\s*(https?:\/\/[^\]]+|\/[^\]]+)\]/g, '<img src="$1" style="width: 24px; height: 24px; vertical-align: middle; display: inline-block; object-fit: contain;">')
  processed = processed.replace(/\n/g, '<br>')
  return processed
}

const goDetail = () => {
  router.push(`/moments/${props.moment.id}`)
}

const openImage = (idx) => {
  if (window.Fancybox) {
    const gallery = imageList.value.map((src, i) => ({
      src,
      caption: ''
    }))
    Fancybox.show(gallery, { startIndex: idx })
  }
}

const toggleActionPanel = () => {
  showActionPanel.value = !showActionPanel.value
}

const handleToggleComments = () => {
  showActionPanel.value = false
  showComments.value = !showComments.value
  emit('toggleComments', props.moment.id)
}

const handleEdit = () => {
  showActionPanel.value = false
  emit('edit', props.moment)
}

const handleDelete = async () => {
  showActionPanel.value = false
  if (!confirm('确定要删除这条动态吗？')) return

  try {
    const res = await request.delete('/api/moments/remove', { ids: String(props.moment.id) })
    if (res.code === 200) {
      toast.success('删除成功')
      emit('delete', props.moment.id)
    } else {
      toast.error(res.msg || '删除失败')
    }
  } catch (err) {
    console.error('删除失败:', err)
    toast.error('删除失败')
  }
}

const handleSetTop = async (topValue) => {
  showActionPanel.value = false
  if (!isAdmin.value) {
    toast.error('仅管理员可操作置顶')
    return
  }
  const actionText = topValue === 1 ? '置顶' : '取消置顶'
  if (!confirm(`确定要${actionText}这条动态吗？`)) return

  try {
    const res = await request.put('/api/moments/set_top', {
      ids: String(props.moment.id),
      top: topValue
    })
    if (res.code === 200) {
      toast.success(`${actionText}成功`)
      emit('setTop', { id: props.moment.id, top: topValue })
    } else {
      toast.error(res.msg || `${actionText}失败`)
    }
  } catch (err) {
    console.error(`${actionText}失败:`, err)
    toast.error(`${actionText}失败`)
  }
}

const handleLike = async () => {
  if (!props.isLogin) {
    store.switchAuth('login', true)
    return
  }

  const oldState = isLiked.value
  const oldCount = likeCount.value

  isLiked.value = !oldState
  likeCount.value = oldState ? Math.max(0, oldCount - 1) : oldCount + 1

  try {
    const apiPath = oldState ? '/api/user-likes/unlike' : '/api/user-likes/like'
    const method = oldState ? request.put : request.post
    const res = await method(apiPath, {
      target_type: 'moment',
      target_id: props.moment.id
    })
    if (res.code === 200) {
      showActionPanel.value = false
      await refreshCounts()
    } else {
      isLiked.value = oldState
      likeCount.value = oldCount
      toast.error(res.msg || '操作失败')
    }
  } catch (err) {
    isLiked.value = oldState
    likeCount.value = oldCount
    console.error('点赞失败:', err)
  }
}

const refreshCounts = async () => {
  const momentId = props.moment.id
  try {
    const requests = []
    if (props.isLogin) {
      requests.push(
        request.get('/api/user-likes/is-liked', {
          target_type: 'moment', target_id: momentId
        }).then(res => {
          if (res.code === 200) {
            isLiked.value = !!res.data?.is_liked
            likeCount.value = res.data?.count || 0
          }
        })
      )
      requests.push(
        request.get('/api/user-collects/is-collected', {
          target_type: 'moment', target_id: momentId
        }).then(res => {
          if (res.code === 200) {
            isCollected.value = !!res.data?.is_collected
            collectCount.value = res.data?.count || 0
          }
        })
      )
    } else {
      requests.push(
        request.get('/api/user-likes/counts', {
          target_type: 'moment', target_ids: [momentId]
        }).then(res => {
          if (res.code === 200 && res.data?.counts) {
            likeCount.value = res.data.counts[String(momentId)] || 0
          }
        })
      )
      requests.push(
        request.get('/api/user-collects/counts', {
          target_type: 'moment', target_ids: [momentId]
        }).then(res => {
          if (res.code === 200 && res.data?.counts) {
            collectCount.value = res.data.counts[String(momentId)] || 0
          }
        })
      )
    }
    requests.push(
      request.get('/api/moments/comment_count', { bind_id: momentId }).then(res => {
        if (res.code === 200) {
          commentCount.value = res.data || 0
        }
      })
    )
    await Promise.all(requests)
  } catch (err) {
    console.error('刷新数量失败:', err)
  }
}

const batchLoadActions = async () => {
  await refreshCounts()
}

const onCommentAdded = () => {
  refreshCounts()
  emit('commentAdded')
}

const handleClickOutside = (event) => {
  if (showActionPanel.value && !event.target.closest('.wx-more-wrap')) {
    showActionPanel.value = false
  }
}

onMounted(() => {
  batchLoadActions()
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})

defineExpose({ refreshCounts, batchLoadActions })
</script>

<style scoped>
.wx-moment {
  padding: 0;
}

.wx-moment-inner {
  display: flex;
  gap: 0.75rem;
}

/* 左侧头像 */
.wx-avatar-wrap {
  flex-shrink: 0;
}

.wx-avatar-link {
  display: inline-block;
  text-decoration: none;
}

.wx-avatar-link:hover {
  transform: scale(1.05);
}

/* 右侧内容 */
.wx-content {
  flex: 1;
  min-width: 0;
  position: relative;
}

.wx-nickname {
  display: inline-block;
  font-size: 1rem;
  font-weight: 600;
  color: var(--bs-primary);
  text-decoration: none;
  transition: var(--wx-transition);
  margin-right: 0.5rem;
}

.wx-nickname-row {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin-bottom: 0.25rem;
  flex-wrap: wrap;
}

.wx-level-badge {
  font-size: 0.75rem !important;
  padding: 2px 8px !important;
}

.wx-title-badge {
  font-size: 0.75rem !important;
  padding: 2px 8px !important;
}

.wx-top-badge {
  font-size: 0.7rem !important;
  padding: 2px 8px !important;
  font-weight: 600;
  box-shadow: 0 1px 4px rgba(255, 193, 7, 0.4);
}

.wx-top-badge .bi {
  margin-right: 2px;
}

/* 头衔颜色 - 修仙体系 */
.title-badge {
  color: #fff;
  font-weight: 500;
}
.title-zhangmen { background: linear-gradient(135deg, #f6d365, #fda085) !important; color: #5a3e00 !important; }
.title-zhanglao { background: #8e44ad !important; }
.title-hufa { background: #c0392b !important; }
.title-neimen { background: #2980b9 !important; }
.title-waimen { background: #16a085 !important; }
.title-lianqi { background: #27ae60 !important; }
.title-zhuji { background: #8bc34a !important; color: #1a3d00 !important; }
.title-jiedan { background: #e67e22 !important; }
.title-yuanying { background: #6c5ce7 !important; }
.title-huashen { background: #00b894 !important; }
.title-default { background: #6c757d !important; }

.wx-nickname:hover {
  text-decoration: underline;
  opacity: 0.85;
}

.wx-text {
  font-size: 1rem;
  line-height: 1.5;
  color: var(--bs-body-color);
  word-break: break-word;
  white-space: pre-wrap;
  margin-bottom: 0.5rem;
  cursor: pointer;
}

/* 图片网格 */
.wx-images {
  display: grid;
  gap: 0.375rem;
  margin-bottom: 0.5rem;
}

.wx-images.img-grid-1 {
  grid-template-columns: 1fr;
  max-width: 260px;
}

.wx-images.img-grid-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 280px;
}

.wx-images.img-grid-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 280px;
}

.wx-images.img-grid-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 320px;
}

.wx-img-wrap {
  position: relative;
  aspect-ratio: 1;
  overflow: hidden;
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
}

.wx-img-wrap:hover {
  transform: scale(1.02);
  box-shadow: var(--wx-shadow-sm);
}

.img-grid-1 .wx-img-wrap {
  aspect-ratio: auto;
  max-height: 260px;
}

.wx-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: var(--wx-transition);
}

.wx-img-wrap:hover .wx-img {
  transform: scale(1.04);
}

.img-grid-1 .wx-img {
  max-height: 260px;
  width: auto;
  object-fit: contain;
}

/* 位置 */
.wx-location {
  font-size: 0.8rem;
  color: var(--bs-primary);
  margin-bottom: 0.375rem;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

/* 元信息 */
.wx-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.25rem;
}

.wx-meta-info {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.wx-time {
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
}

.wx-meta-count {
  display: inline-flex;
  align-items: center;
  gap: 0.125rem;
  font-size: 0.8rem;
  color: var(--bs-danger);
}

.wx-meta-count.wx-meta-views {
  color: var(--bs-secondary-color);
}

.wx-meta-count.wx-meta-views i {
  color: var(--bs-secondary-color);
}

.wx-meta-count i {
  font-size: 0.8rem;
}

.wx-more-btn {
  width: 1.5rem;
  height: 1.5rem;
  border: none;
  background: transparent;
  color: var(--bs-secondary-color);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
}

.wx-more-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.wx-more-btn:hover {
  background: var(--bs-tertiary-bg);
  color: var(--bs-body-color);
}

.wx-more-btn i {
  font-size: 0.9rem;
}

/* 操作面板 */
.wx-action-panel {
  position: absolute;
  right: 0;
  top: 100%;
  margin-top: 0.25rem;
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  display: flex;
  align-items: center;
  padding: 0.375rem;
  box-shadow: var(--wx-shadow-lg);
  z-index: 10;
  animation: wx-slide-down 0.18s ease;
  white-space: nowrap;
}

@keyframes wx-slide-down {
  from {
    opacity: 0;
    transform: translateY(-0.25rem);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.wx-action-item {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  background: transparent;
  border: none;
  color: var(--bs-body-color);
  font-size: 0.85rem;
  padding: 0.375rem 0.75rem;
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
  white-space: nowrap;
}

.wx-action-item:hover {
  background: var(--bs-tertiary-bg);
}

.wx-action-item.active {
  color: var(--bs-danger);
}

.wx-action-item.danger {
  color: var(--bs-danger);
}

.wx-action-item.danger:hover {
  background: rgba(var(--bs-danger-rgb), 0.1);
}

.wx-action-count {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
}

.wx-action-divider {
  width: 1px;
  height: 0.875rem;
  background: var(--bs-border-color);
  margin: 0 0.125rem;
}

/* 草稿标签 */
.wx-draft-tag {
  position: absolute;
  top: 0;
  right: 0;
  font-size: 0.75rem;
  background: var(--bs-warning);
  color: var(--bs-dark);
  padding: 0.0625rem 0.5rem;
  border-radius: var(--wx-radius-sm);
  font-weight: 600;
}

/* 评论区 */
.wx-comments-section {
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  margin-top: 0.5rem;
  padding: 0.75rem;
}

/* 移动端 */
@media (max-width: 768px) {
  .wx-text {
    font-size: 0.9rem;
  }

  .wx-images.img-grid-1 {
    max-width: 240px;
  }

  .wx-images.img-grid-3 {
    max-width: 260px;
  }

  .wx-images.img-grid-9 {
    max-width: 280px;
  }

  .img-grid-1 .wx-img-wrap,
  .img-grid-1 .wx-img {
    max-height: 240px;
  }

  .wx-nickname {
    font-size: 0.9rem;
  }
}
</style>
