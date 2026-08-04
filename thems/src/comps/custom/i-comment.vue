<!-- src/comps/CommentList.vue 通用评论组件 -->
<template>
  <div v-if="articleId && isCommentVisible" class="wx-card">
    <!-- 评论区标题：接收props的评论数，动态展示 -->
    <div class="card-header bg-transparent border-0">
      <h3 class="h5 fw-bold mt-2">
        <i class="bi bi-chat-dots me-2"></i>
        评论 ({{ commentCount || 0 }})
      </h3>
    </div>
    <div class="card-body">
      <!-- 评论功能关闭提示 -->
      <div v-if="!isCommentEnabled" class="wx-empty-state">
        <i class="bi bi-chat-x"></i>
        <p>感谢您的关注，评论功能正在维护中</p>
      </div>
      
      <!-- 评论输入框：仅登录状态显示 -->
      <div class="mb-5" v-if="isCommentEnabled && isLogin">
        <div
          ref="commentEditorRef"
          class="form-control border border-secondary-subtle bg-body wx-contenteditable"
          :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !commentInput }"
          contenteditable="true"
          data-placeholder="随便说点什么吧..."
          @input="onCommentInput"
          @keydown="onEditorKeydown"
          style="min-height: 60px; max-height: 300px; overflow-y: auto;"
        ></div>
        
        <!-- 表情选择面板 -->
        <i-emoji-picker 
          v-model="showEmojiPicker"
          :is-dark-mode="isDarkMode"
          @select="insertEmoji"
        />
        
        <!-- 按钮区域：表情按钮和发布按钮 -->
        <div class="d-flex gap-2 mt-2 align-items-center">
          <button 
            @click.stop="toggleEmojiPicker"
            class="wx-btn-outline emoji-button"
            title="表情"
          >
            <i class="bi bi-emoji-smile fs-5"></i>
          </button>
          <button 
            @click="openImageModal"
            class="wx-btn-outline"
            title="图片"
          >
            <i class="bi bi-image fs-5"></i>
          </button>
          <div class="flex-grow-1"></div>
          <button 
              @click="handlePublish"
              class="wx-btn-gradient px-5"
              :disabled="!commentInput.trim() || isCommenting"
            >
              {{ isCommenting ? '发布中...' : '评论' }}
          </button>
        </div>
      </div>

      <!-- 未登录引导区 -->
      <div class="mb-5 p-4 bg-body text-center border rounded-3" v-else-if="isCommentEnabled && !isLogin">
        <i class="bi bi-person-circle fs-3 mb-2"></i>
        <p class="mb-3 text-muted">登录后即可发表评论～</p>
        <div class="d-flex gap-2 justify-content-center">
          <button 
            @click="handleToLogin()"
            class="wx-btn-gradient btn-sm px-4"
          >
            登录
          </button>
          <button 
            @click="handleToRegister()"
            class="wx-btn-outline btn-sm px-4"
          >
            注册
          </button>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="isLoading" class="text-center py-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">加载中...</span>
        </div>
        <p class="text-muted mt-2">加载评论中...</p>
      </div>

      <!-- 评论列表 -->
      <div class="comments-list" v-else-if="isCommentEnabled && processedCommentList.length > 0">
        <div 
          class="comment-item pb-3 mb-3"
          v-for="(item, index) in processedCommentList" 
          :key="item.id || index"
        >
          <!-- 分割线：不是第一个评论时显示 -->
          <hr v-if="index > 0" class="my-3 border-top border-secondary-subtle" />
          <div class="d-flex">
            <!-- 头像 -->
            <i-avatar-frame 
              :src="item.avatar || fallbackAvatar" 
              :frame="item.frame"
              :alt="item.nickname || '用户头像'"
              size="40px"
              :frame-scale="1.5"
              class="me-3"
            />
            
            <div class="flex-grow-1">
              <!-- 昵称和徽章 -->
              <div class="d-flex align-items-center gap-2 mb-1">
                <h6 class="fw-semibold mb-0 me-2">
                  <router-link v-if="item.authorId" :to="`/author/${item.authorId}`" class="text-decoration-none text-reset">
                    {{ item.nickname || '匿名用户' }}
                  </router-link>
                  <span v-else>{{ item.nickname || '匿名用户' }}</span>
                </h6>
                <span v-if="item.level" class="badge bg-primary text-white rounded-pill" style="font-size: 12px; padding: 4px 10px;">Lv.{{ item.level }} {{ item.levelName }}</span>
                <span v-if="item.isAuthor" class="badge bg-warning text-dark rounded-pill" style="font-size: 12px; padding: 4px 10px;">作者</span>
                <span v-if="item.title" class="badge rounded-pill title-badge" :class="getTitleColorClass(item.title)" style="font-size: 12px; padding: 4px 10px;">
                  <i class="bi bi-person-badge"></i> {{ item.title }}
                </span>
              </div>
              
              <!-- 内容 -->
              <p class="mb-2 text-reset" v-html="item.content"></p>
              
              <!-- 时间、位置和操作按钮 -->
              <div class="d-flex align-items-center text-muted small mb-2">
                <span class="me-3">{{ item.time || '未知时间' }}</span>
                <span v-if="item.location" class="me-3">
                  <i class="bi bi-geo-alt me-1"></i>{{ item.location }}
                </span>
                <span v-if="item.device" class="me-auto">{{ item.device }}</span>
                
                <button 
                  class="btn btn-link btn-sm text-decoration-none text-muted p-0 me-3"
                  @click="toggleReplyForm(index)"
                >
                  <i class="bi bi-emoji-smile me-1"></i>回复
                </button>
                
                <button 
                  class="btn btn-link btn-sm text-decoration-none text-muted p-0 me-3"
                  @click="handleCommentLike(item.id)"
                >
                  <i :class="getLikeStatus(item.id) ? 'bi bi-hand-thumbs-up-fill text-primary' : 'bi bi-hand-thumbs-up'"></i>
                  <span class="ms-1">{{ getLikeCount(item.id) || '' }}</span>
                </button>
              </div>
              
              <!-- 回复输入框 -->
              <div v-if="showReplyIndex === index" class="mb-3 reply-form">
                <div
                  class="form-control border border-secondary-subtle bg-body wx-contenteditable mb-2"
                  :data-placeholder="replyTarget ? `回复 ${replyTarget.nickname}...` : '回复评论...'"
                  :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !replyInput }"
                  contenteditable="true"
                  @input="onReplyInput"
                  @keydown="onEditorKeydown"
                  style="min-height: 40px; max-height: 200px; overflow-y: auto;"
                ></div>
                
                <!-- 表情选择面板 -->
                <i-emoji-picker 
                  v-model="showReplyEmojiPicker"
                  :is-dark-mode="isDarkMode"
                  @select="insertReplyEmoji"
                />
                
                <div class="d-flex gap-2">
                  <button 
                    @click.stop="toggleReplyEmojiPicker"
                    class="wx-btn-outline emoji-button"
                  >
                    <i class="bi bi-emoji-smile me-1"></i>表情
                  </button>
                  <button 
                    @click="handleSubmitReply()"
                    class="wx-btn-gradient publish-btn flex-grow-1"
                    :disabled="!replyInput.trim() || isCommenting"
                  >
                    <i class="bi" :class="isCommenting ? 'bi-arrow-clockwise spin' : 'bi-paper-plane-fill'"></i>
                    {{ isCommenting ? ' 发送中...' : ' 发送' }}
                  </button>
                </div>
              </div>
              
              <!-- 折叠回复区域 -->
              <div v-if="item.replies && item.replies.length > 0">
                <button 
                  v-if="!expandedReplies.has(item.id)"
                  class="btn btn-link btn-sm text-decoration-none p-0 text-primary"
                  @click="toggleExpandReply(item.id)"
                >
                  <i class="bi bi-chevron-right me-1"></i>
                  展开 {{ item.replies.length }} 条回复
                </button>
                
                <div v-else class="mt-2">
                  <div 
                    class="reply-item mb-2"
                    v-for="(reply, rIndex) in item.replies" 
                    :key="reply.id || rIndex"
                  >
                    <div class="d-flex">
                      <i-avatar-frame 
                        :src="reply.avatar || fallbackAvatar" 
                        :frame="reply.frame"
                        :alt="reply.nickname || '用户头像'"
                        size="32px"
                        :frame-scale="1.5"
                        class="me-2 rounded"
                      />
                      
                      <div class="flex-grow-1">
                        <!-- 回复昵称和徽章 -->
                        <div class="d-flex align-items-center gap-2 mb-1">
                          <h6 class="fw-semibold mb-0 me-2 small">
                            <router-link v-if="reply.authorId" :to="`/author/${reply.authorId}`" class="text-decoration-none text-reset">
                              {{ reply.nickname || '匿名用户' }}
                            </router-link>
                            <span v-else>{{ reply.nickname || '匿名用户' }}</span>
                          </h6>
                          <span v-if="reply.level" class="badge bg-primary text-white rounded-pill" style="font-size: 11px; padding: 3px 8px;">Lv.{{ reply.level }} {{ reply.levelName }}</span>
                          <span v-if="reply.isAuthor" class="badge bg-warning text-dark rounded-pill" style="font-size: 11px; padding: 3px 8px;">作者</span>
                          <span v-if="reply.title" class="badge rounded-pill title-badge" :class="getTitleColorClass(reply.title)" style="font-size: 11px; padding: 3px 8px;">
                            <i class="bi bi-person-badge"></i> {{ reply.title }}
                          </span>
                        </div>
                        
                        <!-- 回复内容 -->
                        <p class="mb-1 small text-reset" v-html="reply.content"></p>
                        
                        <!-- 时间和操作 -->
                        <div class="d-flex align-items-center text-muted text-xs">
                          <span class="me-3">{{ reply.time || '未知时间' }}</span>
                          <span v-if="reply.location" class="me-3">
                            <i class="bi bi-geo-alt me-1"></i>{{ reply.location }}
                          </span>
                          <span v-if="reply.device" class="me-auto">{{ reply.device }}</span>
                          
                          <button 
                            class="btn btn-link btn-sm text-decoration-none text-muted p-0 me-2"
                            @click="toggleReplyForm(index, rIndex)"
                          >
                            <i class="bi bi-emoji-smile me-1"></i>回复
                          </button>
                          
                          <button 
                            class="btn btn-link btn-sm text-decoration-none text-muted p-0 me-2"
                            @click="handleCommentLike(reply.id)"
                          >
                            <i :class="getLikeStatus(reply.id) ? 'bi bi-hand-thumbs-up-fill text-primary' : 'bi bi-hand-thumbs-up'"></i>
                            <span class="ms-1">{{ getLikeCount(reply.id) || '' }}</span>
                          </button>
                        </div>
                        
                        <!-- 二级回复的回复输入框 -->
                        <div v-if="showReplyIndex === `${index}-${rIndex}`" class="mb-3 reply-form">
                          <div
                            class="form-control border border-secondary-subtle bg-body wx-contenteditable"
                            :data-placeholder="replyTarget ? `回复 ${replyTarget.nickname}...` : '回复评论...'"
                            :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !replyInput }"
                            contenteditable="true"
                            @input="onReplyInput"
                            @keydown="onEditorKeydown"
                            style="min-height: 40px; max-height: 200px; overflow-y: auto;"
                          ></div>
                          
                          <!-- 表情选择面板 -->
                          <i-emoji-picker 
                            v-model="showReplyEmojiPicker"
                            :is-dark-mode="isDarkMode"
                            @select="insertReplyEmoji"
                          />
                          
                          <div class="d-flex gap-2">
                            <button 
                              @click.stop="toggleReplyEmojiPicker"
                              class="wx-btn-outline emoji-button"
                            >
                              <i class="bi bi-emoji-smile me-1"></i>表情
                            </button>
                            <button 
                              @click="handleSubmitReply()"
                              class="wx-btn-gradient publish-btn flex-grow-1"
                              :disabled="!replyInput.trim() || isCommenting"
                            >
                              <i class="bi" :class="isCommenting ? 'bi-arrow-clockwise spin' : 'bi-paper-plane-fill'"></i>
                              {{ isCommenting ? ' 发送中...' : ' 发送' }}
                            </button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  
                  <button 
                    class="btn btn-link btn-sm text-decoration-none p-0 text-primary"
                    @click="toggleExpandReply(item.id)"
                  >
                    <i class="bi bi-chevron-down me-1"></i>
                    收起回复
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 无评论提示 -->
      <div v-else-if="isCommentEnabled && !isLoading" class="wx-empty-state">
        <i class="bi bi-chat-square-text"></i>
        <p>暂无评论，快来抢沙发吧～</p>
      </div>

      <!-- 分页控件 -->
      <div v-if="isCommentEnabled && totalComments > pageSize" class="mt-4">
        <nav aria-label="评论分页">
          <ul class="pagination justify-content-center">
            <li class="page-item" :class="{ disabled: currentPage === 1 }">
              <button class="page-link" @click="handlePageChange(currentPage - 1)" :disabled="currentPage === 1">
                <span aria-hidden="true">&laquo;</span>
              </button>
            </li>
            <li v-for="page in displayedPages" :key="page" class="page-item" :class="{ active: currentPage === page, 'd-none': page === '...' }">
              <button class="page-link" @click="handlePageChange(page)" v-if="page !== '...'">
                {{ page }}
              </button>
              <span class="page-link text-muted" v-else>...</span>
            </li>
            <li class="page-item" :class="{ disabled: currentPage === totalPages }">
              <button class="page-link" @click="handlePageChange(currentPage + 1)" :disabled="currentPage === totalPages">
                <span aria-hidden="true">&raquo;</span>
              </button>
            </li>
          </ul>
        </nav>
      </div>
    </div>
  </div>

  <!-- 图片链接输入模态框 -->
  <div class="modal fade" :class="{ show: showImageModal }" tabindex="-1" v-if="showImageModal" style="display: block; z-index: 1055;">
    <div class="modal-backdrop show" @click="closeImageModal" style="z-index: 1054;"></div>
    <div class="modal-dialog" style="z-index: 1056;">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">插入图片</h5>
          <button type="button" class="btn-close" @click="closeImageModal"></button>
        </div>
        <div class="modal-body">
          <div class="mb-3">
            <label class="form-label d-block mb-2">上传图片</label>
            <button
              type="button"
              class="wx-btn-gradient btn-sm"
              @click="handleUploadImage"
              :disabled="uploadingImage"
            >
              <i class="bi bi-upload me-2"></i>
              {{ uploadingImage ? '上传中...' : '上传图片' }}
            </button>
          </div>

          <div class="mb-3">
            <button
              type="button"
              class="wx-btn-outline btn-sm"
              @click="showCustomUrlInput = !showCustomUrlInput"
            >
              <i class="bi bi-link-45deg me-2"></i>自定义链接
            </button>
          </div>
          
          <div v-if="showCustomUrlInput" class="mb-3">
            <div class="input-group">
              <span class="input-group-text"><i class="bi bi-globe"></i></span>
              <input 
                type="text" 
                class="form-control" 
                id="imageUrl" 
                v-model="imageUrl" 
                placeholder="请输入图片链接"
                @keyup.enter="insertImage"
              >
              <button
                type="button"
                class="wx-btn-outline"
                @click="applyCustomImageUrl"
                :disabled="!imageUrl.trim()"
              >
                应用
              </button>
            </div>
          </div>
          
          <div v-if="previewImage" class="mb-3">
            <label class="form-label d-block mb-2">图片预览</label>
            <img :src="previewImage" class="img-fluid rounded" style="max-height: 200px;" />
          </div>
          
          <div class="mb-3">
            <label for="imageName" class="form-label">图片名称（可选）</label>
            <input type="text" class="form-control" id="imageName" v-model="imageName" placeholder="请输入图片名称">
          </div>
        </div>
        <div class="modal-footer">
          <button type="button" class="wx-btn-outline" @click="closeImageModal">取消</button>
          <button type="button" class="wx-btn-gradient" @click="insertImage" :disabled="!previewImage">确定</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useCommStore } from '@/store/comm'
import { request, uploadImage } from '@/utils/network'
import { toast } from '@/utils/app'
import iEmojiPicker from './i-emoji-picker.vue'
import iAvatarFrame from './i-avatar-frame.vue'
import { validateComment, checkRateLimit as checkRateLimitUtil, setRateLimit, getTitleColorClass } from '@/utils/app'
import utils from '@/utils/utils'
import { STORAGE_KEYS } from '@/constants'

/**
 * @typedef {Object} CommentItem
 * @property {string|number} id - 评论ID
 * @property {string|number} authorId - 作者ID
 * @property {string} avatar - 头像地址
 * @property {string} nickname - 昵称
 * @property {number|null} level - 等级
 * @property {string} levelName - 等级名称
 * @property {string} time - 时间
 * @property {string} content - 内容（HTML）
 * @property {boolean} isAuthor - 是否为作者
 * @property {CommentItem[]} replies - 回复列表
 * @property {string} location - 位置
 * @property {string} device - 设备
 */

/**
 * @typedef {Object} ArticleAuthor
 * @property {string|number} id - 作者ID
 */

/**
 * @typedef {Object} Props
 * @property {string|number} articleId - 文章ID（必填）
 * @property {string|number} commentCount - 评论总数
 * @property {Array} commentList - 评论列表数据
 * @property {boolean} isLogin - 是否已登录（必填）
 * @property {ArticleAuthor} articleAuthor - 文章作者信息
 * @property {boolean} isDarkMode - 是否深色模式
 * @property {number} currentPage - 当前页码
 * @property {number} pageSize - 每页条数
 * @property {number} totalComments - 总评论数
 */

/** @type {Props} */
const props = defineProps({
  articleId: {
    type: [String, Number],
    required: false,
    default: null
  },
  commentCount: {
    type: [String, Number],
    default: 0
  },
  commentList: {
    type: Array,
    default: () => []
  },
  isLogin: {
    type: Boolean,
    required: true,
    default: false
  },
  articleAuthor: {
    type: Object,
    default: () => ({})
  },
  isDarkMode: {
    type: Boolean,
    default: false
  },
  currentPage: {
    type: Number,
    default: 1
  },
  articleCommentConfig: {
    type: Object,
    default: () => ({ allow: null, show: null })
  },
  pageSize: {
    type: Number,
    default: 10
  },
  totalComments: {
    type: Number,
    default: 0
  }
})

/** @typedef {'publishComment' | 'replyComment' | 'toLogin' | 'toRegister' | 'pageChange'} Emits */
const emit = defineEmits(['publishComment', 'replyComment', 'toLogin', 'toRegister', 'pageChange'])

const store = useCommStore()

// 响应式状态
const commentInput = ref('')
const replyInput = ref('')
const commentEditorRef = ref(null)
const showReplyIndex = ref(null)
const replyTarget = ref(null)
const isSystemDark = ref(false)
const isLoading = ref(false)
const showEmojiPicker = ref(false)
const showReplyEmojiPicker = ref(false)
// 展开回复的状态集合
const expandedReplies = ref(new Set())
// 图片弹窗状态
const showImageModal = ref(false)
const imageUrl = ref('')
const imageName = ref('')
const previewImage = ref('')
const uploadingImage = ref(false)
const showCustomUrlInput = ref(false)

// 点赞状态
const commentLikes = ref(new Map())
const commentLikeCounts = ref(new Map())

// 评论配置
const commentConfig = ref({})
const globalCommentEnabled = ref(true)
const lastCommentTime = ref(0)
const isCommenting = ref(false)
const maxCommentLength = ref(500)

// 是否允许评论（综合全局配置和文章配置）
const isCommentEnabled = computed(() => {
  // 文章级别的配置优先级更高
  if (props.articleCommentConfig.allow !== null && props.articleCommentConfig.allow !== undefined) {
    return props.articleCommentConfig.allow === 1
  }
  // 使用全局配置
  return globalCommentEnabled.value
})

// 是否显示评论（综合全局配置和文章配置）
const isCommentVisible = computed(() => {
  if (!globalCommentEnabled.value) {
    return false
  }
  if (props.articleCommentConfig.show !== null && props.articleCommentConfig.show !== undefined) {
    return props.articleCommentConfig.show === 1
  }
  return true
})

// 常量
const fallbackAvatar = 'https://picsum.photos/60/60'

// 分页相关计算属性
const totalPages = computed(() => {
  return Math.ceil(props.totalComments / props.pageSize)
})

const displayedPages = computed(() => {
  const pages = []
  const total = totalPages.value
  const current = props.currentPage
  
  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
  } else {
    if (current <= 3) {
      pages.push(1, 2, 3, 4, '...', total)
    } else if (current >= total - 2) {
      pages.push(1, '...', total - 3, total - 2, total - 1, total)
    } else {
      pages.push(1, '...', current - 1, current, current + 1, '...', total)
    }
  }
  return pages
})

// 获取点赞数
const getLikeCount = (commentId) => {
  return commentLikeCounts.value.get(commentId) || 0
}

// 获取点赞状态
const getLikeStatus = (commentId) => {
  return commentLikes.value.get(commentId) || false
}

// 切换展开/收起回复
const toggleExpandReply = (commentId) => {
  if (expandedReplies.value.has(commentId)) {
    expandedReplies.value.delete(commentId)
  } else {
    expandedReplies.value.add(commentId)
  }
}

// 获取评论配置
const getCommentConfig = async () => {
  try {
    const response = await request.get('/api/config/one', { key: 'COMMENT' })
    if (response.code === 200 && response.data) {
      return response.data.json || {}
    }
    return {}
  } catch (error) {
    console.error('获取评论配置失败:', error)
    return {}
  }
}

// 验证评论内容
const validateCommentContent = (content) => {
  const result = validateComment(content, {
    minLength: 1,
    maxLength: maxCommentLength.value
  })
  
  if (!result.valid) {
    toast.error(result.message)
    return false
  }
  
  const config = commentConfig.value
  
  if (config.require_chinese === 1 && !/[\u4e00-\u9fa5]/.test(content)) {
    toast.error('评论内容必须包含中文')
    return false
  }
  
  if (config.sensitive_filter === 1 && config.sensitive_words) {
    for (const word of config.sensitive_words) {
      if (content.includes(word)) {
        toast.error('评论内容包含敏感词，请修改后重试')
        return false
      }
    }
  }
  
  return true
}

// 检查速率限制
const checkRateLimit = () => {
  const rateLimit = commentConfig.value.rate_limit || {}
  if (rateLimit.enabled !== 1) return true
  
  const result = checkRateLimitUtil('comment_rate_limit', rateLimit.time_window || 60)
  
  if (!result.allowed) {
    toast.error(`评论过于频繁，请等待 ${result.remaining} 秒后再试`)
    return false
  }
  return true
}

// 保存评论时间
const saveCommentTime = () => {
  const currentTime = Date.now() / 1000
  lastCommentTime.value = currentTime
  setRateLimit('comment_rate_limit')
  try {
    localStorage.setItem(STORAGE_KEYS.LAST_COMMENT_TIME, currentTime.toString())
  } catch (error) {
    console.error('存储评论时间失败:', error)
  }
}

// 处理@提及、换行、表情图片和markdown图片（带fancybox）
const processContent = (content, enableMention = false) => {
  if (!content) return ''
  let processed = content
  // 解析 markdown 图片 ![名称](链接) 并添加 fancybox 支持，前后加换行
  processed = processed.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<br><a href="$2" data-fancybox="comments" data-caption="$1"><img src="$2" alt="$1" class="img-fluid rounded shadow-sm" style="max-width: 100%; max-height: 400px; object-fit: contain; margin: 8px 0; cursor: pointer;"></a><br>')
  // 解析表情图片 [emoji:url]
  processed = processed.replace(/\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g, '<img src="$1" class="inline-emoji" style="width: 24px; height: 24px; vertical-align: middle; display: inline-block; object-fit: contain;" loading="lazy">')
  // 处理换行
  processed = processed.replace(/\n/g, '<br>')
  // 处理@提及
  if (enableMention) {
    processed = processed.replace(/@([\u4e00-\u9fa5\w]+)/g, '<span class="at-mention">@$1</span>')
  }
  return processed
}

// 获取等级名称
const getLevelName = (item) => {
  const paths = [
    item.result?.author?.result?.level?.current?.name,
    item.result?.author?.level?.current?.name,
    item.author?.result?.level?.current?.name,
    item.level?.current?.name,
    item.result?.author?.result?.levelName,
    item.result?.author?.levelName,
    item.author?.levelName,
    item.levelName
  ]
  return paths.find(Boolean) || ''
}

// 获取等级值
const getLevelValue = (item) => {
  const paths = [
    item.result?.author?.result?.level?.current?.value,
    item.result?.author?.level?.current?.value,
    item.author?.result?.level?.current?.value,
    item.level?.current?.value,
    item.level
  ]
  const value = paths.find(v => v !== undefined && v !== null)
  return value !== undefined ? value : null
}

// 获取作者ID
const getAuthorId = (item) => {
  return item.result?.author?.id || item.author?.id || null
}

// 获取用户头衔
const getTitle = (item) => {
  const paths = [
    item.result?.author?.title,
    item.result?.author?.json?.title,
    item.author?.title,
    item.author?.json?.title,
    item.title
  ]
  return paths.find(v => v && String(v).trim()) || ''
}

// 判断是否为文章作者
const isArticleAuthor = (commentAuthorId) => {
  const articleAuthorId = props.articleAuthor.id
  return commentAuthorId && articleAuthorId && String(commentAuthorId) === String(articleAuthorId)
}

// 处理评论数据
const processedCommentList = computed(() => {
  const formatTime = (timestamp) => {
    if (!timestamp || timestamp === 0) return '未知时间'
    return utils.natureTime(timestamp, 5) // 使用 utils.js 的 natureTime 显示相对时间
  }

  const processReply = (reply) => {
    const authorId = getAuthorId(reply)
    return {
      id: reply.id,
      authorId,
      avatar: reply.result?.author?.avatar?.trim() || reply.author?.avatar?.trim() || reply.avatar || fallbackAvatar,
      nickname: reply.result?.author?.nickname || reply.author?.nickname || reply.nickname || '匿名用户',
      level: getLevelValue(reply),
      levelName: getLevelName(reply),
      title: getTitle(reply),
      frame: reply.result?.author?.json?.frame || reply.author?.json?.frame || '',
      time: formatTime(reply.create_time || reply.time || reply.update_time),
      content: processContent(reply.content || '', true),
      isAuthor: isArticleAuthor(authorId) || reply.result?.author?.result?.isAuthor || reply.result?.author?.isAuthor || reply.author?.result?.isAuthor || reply.isAuthor || false,
      location: reply.location || '',
      device: reply.device || ''
    }
  }

  return props.commentList.map(item => {
    const authorId = getAuthorId(item)
    return {
      id: item.id,
      authorId,
      avatar: item.result?.author?.avatar?.trim() || item.author?.avatar?.trim() || item.avatar || fallbackAvatar,
      nickname: item.result?.author?.nickname || item.author?.nickname || item.nickname || '匿名用户',
      level: getLevelValue(item),
      levelName: getLevelName(item),
      title: getTitle(item),
      frame: item.result?.author?.json?.frame || item.author?.json?.frame || '',
      time: formatTime(item.create_time || item.time || item.update_time),
      content: processContent(item.content || ''),
      isAuthor: isArticleAuthor(authorId) || item.result?.author?.result?.isAuthor || item.result?.author?.isAuthor || item.author?.result?.isAuthor || item.isAuthor || false,
      replies: Array.isArray(item.replies) ? item.replies.map(processReply) : [],
      location: item.location || '',
      device: item.device || ''
    }
  })
})

// 发布评论
const handlePublish = async () => {
  const content = commentInput.value.trim()
  if (!content) return
  if (!checkRateLimit()) return
  if (!validateCommentContent(content)) return
  
  isCommenting.value = true
  
  try {
    saveCommentTime()
    
    emit('publishComment', {
      articleId: props.articleId,
      content
    })
    
    commentInput.value = ''
    clearEditor(commentEditorRef.value)
  } catch (error) {
    console.error('发布评论失败:', error)
    toast.error('发布评论失败，请稍后重试')
  } finally {
    isCommenting.value = false
  }
}

// 切换回复输入框
const toggleReplyForm = (index, replyIndex = null) => {
  if (!props.isLogin) {
    toast.info('请先登录后再回复')
    store.switchAuth('login', true)
    return
  }

  const uniqueKey = replyIndex !== null ? `${index}-${replyIndex}` : index
  
  if (showReplyIndex.value === uniqueKey) {
    cancelReply()
  } else {
    showReplyIndex.value = uniqueKey
    const parentComment = processedCommentList.value[index]
    const targetComment = replyIndex !== null ? parentComment.replies[replyIndex] : parentComment
    
    replyTarget.value = targetComment
    
    nextTick(() => {
      const editor = document.querySelector('.reply-form .wx-contenteditable')
      if (editor) editor.focus()
    })
  }
}

// 提交回复
const handleSubmitReply = async () => {
  const content = replyInput.value.trim()
  if (!content) return
  if (!checkRateLimit()) return
  if (!validateCommentContent(content)) return
  
  const commentId = replyTarget.value?.id
  if (!commentId) return
  
  isCommenting.value = true
  
  try {
    saveCommentTime()
    
    emit('replyComment', {
      articleId: props.articleId,
      commentId,
      content
    })
    
    cancelReply()
  } catch (error) {
    console.error('提交回复失败:', error)
    toast.error('提交回复失败，请稍后重试')
  } finally {
    isCommenting.value = false
  }
}

// 取消回复
const cancelReply = () => {
  showReplyIndex.value = null
  replyInput.value = ''
  replyTarget.value = null
  showReplyEmojiPicker.value = false
  const editor = document.querySelector('.reply-form .wx-contenteditable')
  if (editor) editor.innerHTML = ''
}

// 表情功能
// ===== contenteditable 编辑器辅助函数 =====

// 从contenteditable div中提取纯文本内容，emoji图片转为[emoji:url]格式
const getEditorContent = (el) => {
  if (!el) return ''
  const clone = el.cloneNode(true)
  // 将表情图片转为[emoji:url]标记
  const imgs = clone.querySelectorAll('img[data-emoji]')
  imgs.forEach(img => {
    const url = img.getAttribute('data-emoji') || img.src || ''
    const textNode = document.createTextNode(`[emoji:${url}]`)
    img.parentNode.replaceChild(textNode, img)
  })
  // 将markdown图片转为![name](url)
  const mdImgs = clone.querySelectorAll('img[data-md-img]')
  mdImgs.forEach(img => {
    const url = img.getAttribute('data-md-img') || img.src || ''
    const name = img.getAttribute('data-md-name') || '图片'
    const textNode = document.createTextNode(`![${name}](${url})`)
    img.parentNode.replaceChild(textNode, img)
  })
  return clone.innerText.trim()
}

// 清空编辑器内容
const clearEditor = (el) => {
  if (!el) return
  el.innerHTML = ''
}

// 在光标位置插入HTML
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

// 输入事件处理
const onCommentInput = () => {
  const content = getEditorContent(commentEditorRef.value)
  if (content.length > maxCommentLength.value) {
    toast.warning(`评论内容不能超过 ${maxCommentLength.value} 字`)
    return
  }
  commentInput.value = content
}

const onReplyInput = (e) => {
  const editor = e?.target || document.querySelector('.reply-form .wx-contenteditable')
  if (!editor) return
  const content = getEditorContent(editor)
  if (content.length > maxCommentLength.value) {
    toast.warning(`评论内容不能超过 ${maxCommentLength.value} 字`)
    return
  }
  replyInput.value = content
}

// 键盘事件处理（支持Enter换行，Shift+Enter提交）
const onEditorKeydown = (e) => {
  // 允许Enter换行，不拦截（默认contenteditable行为）
  // Shift+Enter 保持默认行为
}

const toggleEmojiPicker = () => {
  showEmojiPicker.value = !showEmojiPicker.value
  showReplyEmojiPicker.value = false
}

const toggleReplyEmojiPicker = () => {
  showReplyEmojiPicker.value = !showReplyEmojiPicker.value
  showEmojiPicker.value = false
}

const insertEmoji = (emoji) => {
  const editor = commentEditorRef.value
  if (!editor) {
    commentInput.value += emoji
    return
  }
  // 表情图片格式: [emoji:url]
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    // 文本表情（颜文字）
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  onCommentInput()
}

const insertReplyEmoji = (emoji) => {
  const editor = document.querySelector('.reply-form .wx-contenteditable')
  if (!editor) {
    replyInput.value += emoji
    return
  }
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  replyInput.value = getEditorContent(editor)
}

// 图片功能
const openImageModal = () => {
  showImageModal.value = true
  imageUrl.value = ''
  imageName.value = ''
  previewImage.value = ''
  uploadingImage.value = false
  showCustomUrlInput.value = false
}

const closeImageModal = () => {
  showImageModal.value = false
  imageUrl.value = ''
  imageName.value = ''
  previewImage.value = ''
  uploadingImage.value = false
  showCustomUrlInput.value = false
}

const handleUploadImage = async () => {
  if (uploadingImage.value) return
  uploadingImage.value = true

  try {
    const path = await uploadImage()
    imageUrl.value = path
    previewImage.value = path
    toast.success('图片上传成功')
  } catch (error) {
    toast.error(error.message || '上传失败')
  } finally {
    uploadingImage.value = false
  }
}

const applyCustomImageUrl = () => {
  const url = imageUrl.value.trim()
  if (!url) {
    toast.warning('请输入图片链接')
    return
  }

  if (!/^https?:\/\//.test(url)) {
    toast.warning('请输入有效的图片链接（以 http:// 或 https:// 开头）')
    return
  }

  previewImage.value = url
  toast.success('图片链接已应用')
}

const insertImage = () => {
  if (!previewImage.value.trim()) {
    toast.error('请先上传图片或输入图片链接')
    return
  }
  const name = imageName.value.trim() || '图片'
  const url = previewImage.value.trim()
  const editor = commentEditorRef.value
  if (editor) {
    const imgHtml = `<img src="${url}" data-md-img="${url}" data-md-name="${name}" class="editor-image" style="max-width: 100%; max-height: 200px; vertical-align: middle; display: inline-block; border-radius: 4px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
    onCommentInput()
  } else {
    commentInput.value += `![${name}](${url})`
  }
  closeImageModal()
}

// 登录注册处理
const handleToLogin = () => {
  store.switchAuth('login', true)
}

const handleToRegister = () => {
  store.switchAuth('register', true)
}

// 点赞功能
const handleCommentLike = async (commentId) => {
  if (!props.isLogin) {
    store.switchAuth('login', true)
    return
  }
  
  if (!commentId) return
  
  const currentState = !!commentLikes.value.get(commentId)
  const newState = !currentState
  const currentCount = commentLikeCounts.value.get(commentId) || 0
  const newCount = newState ? currentCount + 1 : Math.max(0, currentCount - 1)
  
  // 乐观更新
  commentLikes.value.set(commentId, newState)
  commentLikeCounts.value.set(commentId, newCount)
  
  try {
    const apiPath = currentState ? '/api/user-likes/unlike' : '/api/user-likes/like'
    const res = await (currentState ? request.put : request.post)(apiPath, {
      target_type: 'comment',
      target_id: commentId
    })
    
    if (res.code === 200) {
      toast.success(newState ? '点赞成功！' : '已取消点赞')
    } else {
      commentLikes.value.set(commentId, currentState)
      commentLikeCounts.value.set(commentId, currentCount)
      toast.error(res.msg || '操作失败，请重试')
    }
  } catch (error) {
    commentLikes.value.set(commentId, currentState)
    commentLikeCounts.value.set(commentId, currentCount)
    console.error('评论点赞操作失败:', error)
    toast.error('网络异常，操作失败')
  }
}

// 批量获取点赞数（使用新 API）
const getCommentLikeCounts = async (commentIds) => {
  if (!commentIds.length) return
  
  try {
    const res = await request.get('/api/user-likes/counts', {
      target_type: 'comment',
      target_ids: commentIds
    })
    if (res.code === 200 && res.data?.counts) {
      commentIds.forEach(id => {
        commentLikeCounts.value.set(id, res.data.counts[String(id)] || 0)
      })
    }
  } catch (error) {
    commentIds.forEach(id => commentLikeCounts.value.set(id, 0))
  }
}

// 检查点赞状态（使用新 API）
const checkCommentLikeStatus = async (commentId) => {
  if (!props.isLogin || !commentId) return
  
  try {
    const res = await request.get('/api/user-likes/is-liked', {
      target_type: 'comment',
      target_id: commentId
    })
    if (res.code === 200) {
      commentLikes.value.set(commentId, !!res.data?.is_liked)
      if (res.data?.count !== undefined) {
        commentLikeCounts.value.set(commentId, res.data.count)
      }
    } else {
      commentLikes.value.set(commentId, false)
    }
  } catch (error) {
    commentLikes.value.set(commentId, false)
  }
}

// 初始化点赞数据（防抖处理）
let initLikeDataTimer = null
const initCommentLikeData = async () => {
  if (initLikeDataTimer) clearTimeout(initLikeDataTimer)
  
  initLikeDataTimer = setTimeout(async () => {
    if (processedCommentList.value.length === 0) return
    
    const allCommentIds = []
    processedCommentList.value.forEach(comment => {
      if (comment.id) allCommentIds.push(comment.id)
      if (comment.replies) {
        comment.replies.forEach(reply => {
          if (reply.id) allCommentIds.push(reply.id)
        })
      }
    })
    
    if (props.isLogin) {
      // 已登录：is-liked 同时返回状态和数量，只需逐个检查
      await Promise.all(allCommentIds.map(id => checkCommentLikeStatus(id)))
    } else {
      // 未登录：批量获取点赞数量
      await getCommentLikeCounts(allCommentIds)
    }
  }, 100)
}

// 处理页码变化
const handlePageChange = (page) => {
  if (page < 1 || page > totalPages.value) return
  emit('pageChange', page)
}

// 应用评论配置
const applyCommentConfig = (config) => {
  maxCommentLength.value = config.max_length || 500
}

// 初始化 Fancybox
const initFancybox = () => {
  setTimeout(() => {
    if (window.Fancybox) {
      Fancybox.unbind("[data-fancybox]")
      Fancybox.bind("[data-fancybox]", {
        Hash: false,
        Thumbs: { autoStart: false }
      })
    }
  }, 100)
}

// 生命周期
onMounted(async () => {
  if (!props.isDarkMode) {
    isSystemDark.value = window.matchMedia('(prefers-color-scheme: dark)').matches
  }
  
  const config = await getCommentConfig()
  commentConfig.value = config
  globalCommentEnabled.value = config.enabled !== 0
  applyCommentConfig(config)
  
  try {
    const storedTime = localStorage.getItem(STORAGE_KEYS.LAST_COMMENT_TIME)
    if (storedTime) lastCommentTime.value = parseFloat(storedTime) || 0
  } catch (error) {
    console.error('读取评论时间失败:', error)
  }
  
  initCommentLikeData()
  initFancybox()
})

onUnmounted(() => {
  if (initLikeDataTimer) clearTimeout(initLikeDataTimer)
  if (window.Fancybox) {
    Fancybox.unbind("[data-fancybox]")
  }
})

// 监听深色模式变化
watch([() => props.isDarkMode, isSystemDark], () => {
  // 深模式处理
})

// 监听评论列表变化
watch(
  () => props.commentList,
  () => {
    initCommentLikeData()
    initFancybox()
  },
  { deep: true }
)
</script>

<style scoped>
/* contenteditable 编辑器样式 */
.wx-contenteditable {
  outline: none;
  padding: 0.5rem 0.75rem;
  font-size: 0.875rem;
  line-height: 1.5;
  white-space: pre-wrap;
  word-wrap: break-word;
  word-break: break-word;
  position: relative;
  min-height: 60px;
}

.wx-contenteditable.is-empty::before {
  content: attr(data-placeholder);
  color: var(--bs-secondary-color);
  pointer-events: none;
  position: absolute;
  left: 0.75rem;
  top: 0.5rem;
  font-size: 0.875rem;
}

.wx-contenteditable .editor-emoji,
.wx-contenteditable .editor-image {
  vertical-align: middle;
  display: inline-block;
}

.wx-contenteditable img {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
}

/* 卡片头部统一间距 */
.wx-card > .card-header {
  padding: 1rem 1.25rem 0.5rem;
}

.wx-card > .card-body {
  padding: 1.25rem;
}

/* 头像 */
.avatar {
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--bs-border-color);
  transition: var(--wx-transition);
}

.avatar:hover {
  transform: scale(1.05);
  border-color: rgba(var(--bs-primary-rgb), 0.4);
}

/* 评论项 */
.comment-item {
  padding: 0.875rem;
  border-radius: var(--wx-radius-md);
  transition: var(--wx-transition);
}

.comment-item:hover {
  background-color: var(--bs-tertiary-bg);
}

/* 回复项 */
.reply-item {
  padding: 0.75rem;
  border-radius: var(--wx-radius-md);
  transition: var(--wx-transition);
}

.reply-item:hover {
  background-color: var(--bs-tertiary-bg);
}

/* 回复输入框 */
.reply-form {
  transition: var(--wx-transition);
  padding: 0.75rem;
  background: var(--bs-tertiary-bg);
  border-radius: var(--wx-radius-md);
  border: 1px solid var(--bs-border-color);
}

/* 移动端适配 */
@media (max-width: 768px) {
  .comment-item .avatar {
    width: 36px !important;
    height: 36px !important;
  }

  .reply-item .avatar {
    width: 28px !important;
    height: 28px !important;
  }
}

/* 输入框焦点 */
:deep(textarea.form-control) {
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
}

:deep(textarea:focus) {
  border-color: var(--bs-primary) !important;
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.15) !important;
  outline: none !important;
}

/* 按钮统一过渡 */
:deep(.btn) {
  transition: var(--wx-transition);
}

/* 加载动画 */
.spin {
  animation: wx-spin 1s linear infinite;
}

/* 评论内容 */
.comment-item p,
.reply-item p {
  line-height: 1.6;
  word-break: break-word;
}

/* @提及 */
:deep(.at-mention) {
  color: var(--bs-primary);
  font-weight: 600;
  text-decoration: none;
}

/* 发布按钮 */
.publish-btn {
  font-weight: 600;
  letter-spacing: 0.5px;
}

/* 表情按钮 */
.emoji-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

/* 引用样式 */
.reply-item:deep(.text-muted span.fw-medium) {
  color: var(--bs-secondary) !important;
  background-color: rgba(var(--bs-secondary-rgb), 0.1);
  padding: 2px 8px;
  border-radius: var(--wx-radius-sm);
  display: inline-block;
}

/* 空状态图标颜色 */
.wx-empty-state i {
  color: var(--bs-secondary-color);
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
</style>