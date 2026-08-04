<template>
  <div class="wx-comments">
    <!-- 顶部操作栏 -->
    <div class="wx-comments-bar">
      <span v-if="totalCount > 0" class="wx-comments-count">共 {{ totalCount }} 条评论</span>
      <span v-else class="wx-comments-count wx-no-comment">暂无评论，来抢沙发吧</span>
      <button
        v-if="isLogin"
        class="wx-btn-outline wx-comment-toggle-btn"
        @click="toggleInput"
      >
        <i class="bi bi-chat-dots"></i>
        {{ showInput ? '收起评论' : '发表评论' }}
      </button>
      <button
        v-else
        class="wx-btn-outline wx-comment-toggle-btn"
        @click="handleLogin"
      >
        <i class="bi bi-chat-dots"></i>
        登录评论
      </button>
    </div>

    <!-- 登录状态：评论输入（可折叠） -->
    <div v-if="isLogin && showInput" class="wx-comment-input-wrap">
      <div
        v-if="replyTo"
        class="wx-reply-hint"
        @click="cancelReply"
      >
        回复 <span class="wx-reply-name">{{ replyTo.name }}</span>
        <i class="bi bi-x-circle-fill"></i>
      </div>
      <div
        ref="commentEditorRef"
        class="form-control border border-secondary-subtle bg-body wx-contenteditable wx-comment-input"
        :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !commentInput }"
        :data-placeholder="replyTo ? `回复 ${replyTo.name}...` : '评论...'"
        contenteditable="true"
        @input="onCommentInput"
        style="min-height: 40px; max-height: 200px; overflow-y: auto;"
      ></div>
      <div class="wx-input-actions">
        <button class="wx-input-btn emoji-button" @click="toggleEmoji" :class="{ active: showEmoji }">
          <i class="bi bi-emoji-smile"></i>
        </button>
        <button
          class="wx-btn-gradient wx-send-btn"
          :disabled="!commentInput.trim() || isSubmitting"
          @click="handleSubmit"
        >
          发送
        </button>
      </div>

      <!-- 表情选择面板 -->
      <i-emoji-picker
        v-model="showEmoji"
        :is-dark-mode="isDarkMode"
        @select="insertEmoji"
      />
    </div>

    <!-- 评论列表 -->
    <div v-if="commentList.length > 0" class="wx-comment-list">
      <div
        v-for="item in commentList"
        :key="item.id"
        class="wx-comment-item"
      >
        <!-- 一级评论内容 -->
        <div class="wx-comment-content">
          <span class="wx-comment-author">{{ item.nickname }}</span>
          <span class="wx-comment-text" v-html="renderContent(item.content)"></span>
        </div>

        <!-- 回复列表（二级评论） -->
        <div v-if="item.replies && item.replies.length > 0" class="wx-replies">
          <div
            v-for="reply in item.replies"
            :key="reply.id"
            class="wx-reply-item"
          >
            <div class="wx-reply-content">
              <span class="wx-comment-author">{{ reply.nickname }}</span>
              <span class="wx-comment-text" v-html="renderContent(reply.content)"></span>
            </div>
            <div class="wx-reply-footer">
              <button
                class="wx-like-action"
                :class="{ active: getLikeStatus(reply.id) }"
                @click="handleLike(reply.id)"
              >
                <i :class="getLikeStatus(reply.id) ? 'bi bi-heart-fill' : 'bi bi-heart'"></i>
                <span v-if="getLikeCount(reply.id) > 0">{{ getLikeCount(reply.id) }}</span>
              </button>
              <span class="wx-dot">·</span>
              <button class="wx-reply-action" @click="startReply(reply, item)">
                回复
              </button>
            </div>
          </div>
        </div>

        <!-- 一级评论底部操作栏 -->
        <div class="wx-comment-footer">
          <button
            class="wx-like-action"
            :class="{ active: getLikeStatus(item.id) }"
            @click="handleLike(item.id)"
          >
            <i :class="getLikeStatus(item.id) ? 'bi bi-heart-fill' : 'bi bi-heart'"></i>
            <span v-if="getLikeCount(item.id) > 0">{{ getLikeCount(item.id) }}</span>
          </button>
          <span class="wx-dot">·</span>
          <button class="wx-reply-action" @click="startReply(item)">
            回复
          </button>
        </div>
      </div>
    </div>

    <!-- 加载更多 -->
    <div v-if="hasMore" class="wx-load-more" @click="loadMore">
      <span>更多评论</span>
      <span class="wx-comment-more-count">共 {{ totalCount }} 条</span>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, nextTick } from 'vue'
import { useCommStore } from '@/store/comm'
import { request } from '@/utils/network'
import { toast } from '@/utils/app'
import iEmojiPicker from '@/comps/custom/i-emoji-picker.vue'

const props = defineProps({
  momentId: {
    type: [String, Number],
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

const emit = defineEmits(['commentAdded'])

const store = useCommStore()

const commentInput = ref('')
const showEmoji = ref(false)
const showInput = ref(false)
const isSubmitting = ref(false)
const commentEditorRef = ref(null)

const commentList = ref([])
const currentPage = ref(1)
const pageSize = ref(5)
const totalCount = ref(0)

const commentLikes = ref(new Map())
const commentLikeCounts = ref(new Map())

const replyTo = ref(null)

const maxLength = ref(500)

const flatCount = ref(0)

const hasMore = computed(() => {
  return flatCount.value > currentPage.value * pageSize.value
})

const getLikeStatus = (id) => commentLikes.value.get(id) || false
const getLikeCount = (id) => commentLikeCounts.value.get(id) || 0

// ===== contenteditable 编辑器辅助函数 =====
const getEditorContent = (el) => {
  if (!el) return ''
  const clone = el.cloneNode(true)
  const imgs = clone.querySelectorAll('img[data-emoji]')
  imgs.forEach(img => {
    const url = img.getAttribute('data-emoji') || img.src || ''
    const textNode = document.createTextNode(`[emoji:${url}]`)
    img.parentNode.replaceChild(textNode, img)
  })
  return clone.innerText.trim()
}

const clearEditor = (el) => {
  if (!el) return
  el.innerHTML = ''
}

const insertHTMLAtCursor = (html, editorEl) => {
  if (!editorEl) return
  editorEl.focus()
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) {
    editorEl.innerHTML += html
    return
  }
  const range = selection.getRangeAt(0)
  range.deleteContents()
  const temp = document.createElement('div')
  temp.innerHTML = html
  const frag = document.createDocumentFragment()
  let lastNode = null
  while (temp.firstChild) {
    lastNode = temp.firstChild
    frag.appendChild(temp.firstChild)
  }
  range.insertNode(frag)
  if (lastNode) {
    range.setStartAfter(lastNode)
    range.collapse(true)
  }
  selection.removeAllRanges()
  selection.addRange(range)
}

const onCommentInput = () => {
  commentInput.value = getEditorContent(commentEditorRef.value)
}

const toggleInput = () => {
  showInput.value = !showInput.value
  if (!showInput.value) {
    cancelReply()
  }
}

const toggleEmoji = () => {
  showEmoji.value = !showEmoji.value
}

const insertEmoji = (emoji) => {
  const editor = commentEditorRef.value
  if (!editor) {
    commentInput.value += emoji
    return
  }
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  onCommentInput()
}

// 渲染评论内容：将 [emoji:url] 转为图片
const renderContent = (content) => {
  if (!content) return ''
  let processed = content
  processed = processed.replace(/\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g, '<img src="$1" class="inline-emoji" style="width: 24px; height: 24px; vertical-align: middle; display: inline-block; object-fit: contain;" loading="lazy">')
  processed = processed.replace(/\n/g, '<br>')
  return processed
}

const handleLogin = () => {
  store.switchAuth('login', true)
}

const loadComments = async (reset = false) => {
  if (reset) {
    currentPage.value = 1
    commentList.value = []
  }

  try {
    const [flatRes, countRes] = await Promise.all([
      request.get('/api/comment/flat', {
        bind_id: props.momentId,
        bind_type: 'moments',
        page: currentPage.value,
        limit: pageSize.value
      }),
      request.get('/api/moments/comment_count', {
        bind_id: props.momentId
      }).catch(() => ({ code: 500, data: 0 }))
    ])

    if (flatRes.code === 200) {
      const rawData = flatRes.data?.data || []
      const newList = rawData.map(processComment)
      if (reset) {
        commentList.value = newList
      } else {
        commentList.value = [...commentList.value, ...newList]
      }
      totalCount.value = countRes.code === 200 ? countRes.data : 0
      flatCount.value = flatRes.data?.count || 0
      await initLikeData()
    }
  } catch (err) {
    console.error('加载评论失败:', err)
  }
}

const processComment = (item) => {
  return {
    id: item.id,
    authorId: item.result?.author?.id || item.uid,
    nickname: item.result?.author?.nickname || '匿名用户',
    avatar: item.result?.author?.avatar || '',
    content: item.content || '',
    replies: Array.isArray(item.replies) ? item.replies.map(processReply) : []
  }
}

const processReply = (reply) => {
  return {
    id: reply.id,
    authorId: reply.result?.author?.id || reply.uid,
    nickname: reply.result?.author?.nickname || '匿名用户',
    avatar: reply.result?.author?.avatar || '',
    content: reply.content || ''
  }
}

const handleSubmit = async () => {
  const content = commentInput.value.trim()
  if (!content) return

  isSubmitting.value = true
  try {
    const data = {
      content,
      bind_id: props.momentId,
      bind_type: 'moments'
    }
    if (replyTo.value) {
      data.pid = replyTo.value.pid || replyTo.value.id
    }

    const res = await request.post('/api/comment/create', data)

    if (res.code === 200) {
      toast.success(replyTo.value ? '回复成功' : '评论成功')
      commentInput.value = ''
      clearEditor(commentEditorRef.value)
      replyTo.value = null
      showEmoji.value = false
      await loadComments(true)
      emit('commentAdded')
    } else {
      toast.error(res.msg || '操作失败')
    }
  } catch (err) {
    console.error('提交评论失败:', err)
    toast.error('操作失败')
  } finally {
    isSubmitting.value = false
  }
}

const startReply = (target, parentItem = null) => {
  if (!props.isLogin) {
    handleLogin()
    return
  }

  showInput.value = true
  showEmoji.value = false

  if (parentItem) {
    replyTo.value = { id: target.id, name: target.nickname, pid: parentItem.id }
  } else {
    replyTo.value = { id: target.id, name: target.nickname, pid: target.id }
  }

  commentInput.value = ''
  clearEditor(commentEditorRef.value)
  nextTick(() => {
    if (commentEditorRef.value) commentEditorRef.value.focus()
  })
}

const cancelReply = () => {
  replyTo.value = null
  commentInput.value = ''
  clearEditor(commentEditorRef.value)
}

const handleLike = async (commentId) => {
  if (!props.isLogin) {
    handleLogin()
    return
  }

  const oldState = getLikeStatus(commentId)
  const oldCount = getLikeCount(commentId)

  commentLikes.value.set(commentId, !oldState)
  commentLikeCounts.value.set(commentId, oldState ? Math.max(0, oldCount - 1) : oldCount + 1)

  try {
    const apiPath = oldState ? '/api/user-likes/unlike' : '/api/user-likes/like'
    const method = oldState ? request.put : request.post
    const res = await method(apiPath, {
      target_type: 'comment',
      target_id: commentId
    })

    if (res.code === 200) {
      if (props.isLogin) {
        const statusRes = await request.get('/api/user-likes/is-liked', {
          target_type: 'comment',
          target_id: commentId
        })
        if (statusRes.code === 200) {
          commentLikes.value.set(commentId, !!statusRes.data?.is_liked)
          commentLikeCounts.value.set(commentId, statusRes.data?.count || 0)
        }
      }
    } else {
      commentLikes.value.set(commentId, oldState)
      commentLikeCounts.value.set(commentId, oldCount)
      toast.error(res.msg || '操作失败')
    }
  } catch (err) {
    commentLikes.value.set(commentId, oldState)
    commentLikeCounts.value.set(commentId, oldCount)
    console.error('评论点赞失败:', err)
  }
}

const initLikeData = async () => {
  const allIds = []
  commentList.value.forEach(c => {
    if (c.id) allIds.push(c.id)
    if (c.replies) {
      c.replies.forEach(r => {
        if (r.id) allIds.push(r.id)
      })
    }
  })

  if (!allIds.length) return

  if (props.isLogin) {
    await Promise.all(allIds.map(async (id) => {
      try {
        const res = await request.get('/api/user-likes/is-liked', {
          target_type: 'comment',
          target_id: id
        })
        if (res.code === 200) {
          commentLikes.value.set(id, !!res.data?.is_liked)
          commentLikeCounts.value.set(id, res.data?.count || 0)
        }
      } catch (err) {
        // ignore
      }
    }))
  } else {
    try {
      const res = await request.get('/api/user-likes/counts', {
        target_type: 'comment',
        target_ids: allIds
      })
      if (res.code === 200 && res.data?.counts) {
        allIds.forEach(id => {
          commentLikeCounts.value.set(id, res.data.counts[String(id)] || 0)
        })
      }
    } catch (err) {
      // ignore
    }
  }
}

const loadMore = async () => {
  currentPage.value++
  await loadComments()
}

watch(() => props.momentId, () => {
  currentPage.value = 1
  commentList.value = []
  loadComments(true)
})

onMounted(() => {
  loadComments(true)
})

defineExpose({ loadComments })
</script>

<style scoped>
.wx-comments {
  width: 100%;
}

/* 顶部操作栏 */
.wx-comments-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
  padding-bottom: 0.625rem;
  border-bottom: 1px solid var(--bs-border-color);
}

.wx-comments-count {
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
}

.wx-comments-count.wx-no-comment {
  color: var(--bs-tertiary-color);
  font-style: italic;
}

/* 复用全局 .wx-btn-outline，仅做尺寸微调 */
.wx-comment-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
}

.wx-comment-toggle-btn i {
  font-size: 0.95rem;
}

/* 输入区 */
.wx-comment-input-wrap {
  margin-bottom: 0.75rem;
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  padding: 0.75rem;
  transition: var(--wx-transition);
}

.wx-comment-input-wrap:focus-within {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.12);
}

.wx-reply-hint {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.8rem;
  color: var(--bs-primary);
  margin-bottom: 0.5rem;
  cursor: pointer;
  padding: 0.25rem 0.625rem;
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
}

.wx-reply-hint:hover {
  border-color: var(--bs-primary);
}

.wx-reply-hint i {
  font-size: 0.9rem;
  color: var(--bs-secondary-color);
}

.wx-reply-name {
  font-weight: 600;
}

.wx-comment-input {
  width: 100%;
  font-size: 0.9rem;
}

.wx-input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 0.5rem;
}

.wx-input-btn {
  background: transparent;
  border: none;
  font-size: 1.2rem;
  color: var(--bs-secondary-color);
  padding: 0.25rem 0.5rem;
  border-radius: var(--wx-radius-sm);
  cursor: pointer;
  transition: var(--wx-transition);
}

.wx-input-btn:hover,
.wx-input-btn.active {
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.08);
}

/* 复用全局 .wx-btn-gradient，仅做尺寸微调 */
.wx-send-btn {
  padding: 0.375rem 1.25rem;
  font-size: 0.85rem;
}

.wx-send-btn.small {
  padding: 0.25rem 0.75rem;
  font-size: 0.8rem;
}

/* 登录提示 */
.wx-login-prompt {
  text-align: center;
  font-size: 0.875rem;
  color: var(--bs-primary);
  padding: 0.75rem 0;
  cursor: pointer;
}

.wx-login-prompt:hover {
  text-decoration: underline;
}

/* 评论列表 */
.wx-comment-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.wx-comments-header {
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--bs-border-color);
  margin-bottom: 0.25rem;
}

.wx-comments-title {
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
}

/* 评论项卡片化 */
.wx-comment-item {
  padding: 0.625rem 0.75rem;
  font-size: 0.9rem;
  line-height: 1.6;
  border-radius: var(--wx-radius-md);
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  transition: var(--wx-transition);
}

.wx-comment-item:hover {
  background: var(--bs-tertiary-bg);
  border-color: rgba(var(--bs-primary-rgb), 0.25);
}

.wx-comment-content {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  color: var(--bs-body-color);
  word-break: break-word;
}

.wx-comment-author {
  color: var(--bs-primary);
  font-weight: 600;
}

.wx-comment-text {
  color: var(--bs-body-color);
}

/* 回复列表 */
.wx-replies {
  margin-top: 0.5rem;
  padding: 0.5rem 0.625rem;
  background: var(--bs-tertiary-bg);
  border-radius: var(--wx-radius-sm);
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.wx-reply-item {
  padding: 0.25rem 0;
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
}

.wx-reply-item:hover {
  transform: translateX(2px);
}

.wx-reply-content {
  display: flex;
  flex-wrap: wrap;
  gap: 0.375rem;
  color: var(--bs-body-color);
  word-break: break-word;
}

.wx-reply-footer {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.25rem;
}

/* 底部操作栏 */
.wx-comment-footer {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  margin-top: 0.375rem;
}

.wx-like-action,
.wx-reply-action {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  background: transparent;
  border: none;
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
}

.wx-like-action:hover,
.wx-reply-action:hover {
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.06);
}

.wx-like-action.active {
  color: var(--bs-danger);
}

.wx-like-action i {
  font-size: 0.85rem;
}

.wx-dot {
  color: var(--bs-tertiary-color);
  font-size: 0.8rem;
}

/* 加载更多 */
.wx-load-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem;
  margin-top: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--bs-primary);
  cursor: pointer;
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  transition: var(--wx-transition);
}

.wx-load-more:hover {
  border-color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.05);
  transform: translateY(-1px);
  box-shadow: var(--wx-shadow-sm);
}

.wx-comment-more-count {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  padding: 0.125rem 0.5rem;
  background: var(--bs-tertiary-bg);
  border-radius: var(--wx-radius-sm);
}

/* 移动端 */
@media (max-width: 768px) {
  .wx-comment-input {
    font-size: 0.85rem;
  }

  .wx-comment-item {
    font-size: 0.85rem;
  }

  .wx-comment-footer {
    gap: 0.25rem;
  }

  .wx-emoji-grid {
    grid-template-columns: repeat(6, 1fr);
  }
}
</style>