<template>
  <div class="comment-item" :id="`comment-${comment.id}`" :class="{ 'is-child': isChild, 'comment-highlight': highlightId && String(highlightId) === String(comment.id) }">
    <div class="comment-avatar">
      <!-- 与动态 / 文章 / 作者主页共用同一套头像渲染：头像框 + 右下角认证角标 -->
      <AvatarFrame
        :src="author.avatar"
        :name="author.nickname"
        :fallback="defaultAvatar"
        :verified="author.cert"
        alt="评论者头像"
        size="40px"
      />
    </div>
    <div class="comment-body">
      <div class="comment-head">
        <span v-if="authorLink" class="comment-name comment-name-link" @click="goAuthor">{{ author.nickname }}</span>
        <span v-else class="comment-name">{{ author.nickname }}</span>
        <span v-if="author.level" class="level-tag">Lv.{{ author.level }}</span>
        <!-- 有装扮样式用 TitleBadge 还原后台配色；没有则回退旧色板 -->
        <TitleBadge
          v-if="author.title || titlePayload"
          :payload="titlePayload || {}"
          :text="author.title"
          size="sm"
        />
        <span v-else-if="author.title" :class="['title-tag', titleClass]">{{ author.title }}</span>
        <!-- 佩戴中的勋章（作者信息里带 users.json，不需要额外请求） -->
        <WearingBadge v-if="author.json?.wearing_badge" :badge="author.json.wearing_badge" />
        <span v-if="author.id === authorId" class="comment-badge">作者</span>
        <!-- 仅作者可见：评论者 / 内容作者自己看得到正文，用徽标标明这条是私密评论 -->
        <span
          v-if="Number(comment.scope) === 4 && !isAuthorOnly"
          class="comment-scope"
          title="仅作者可见：只有内容作者与评论者本人能看到"
        >
          <i class="bi bi-eye-slash" aria-hidden="true" /> 仅作者
        </span>
        <!-- 时间 + 设备：设备取自后端返回的 comment.agent（普通用户为脱敏值），
             解析成「系统 · 浏览器」展示，原始 UA 放在 title 里悬停可见 -->
        <span class="comment-meta">
          <span class="comment-time">{{ formatDate(comment.create_time) }}</span>
          <span v-if="device.label" class="comment-device" :title="agentRaw">
            <i :class="device.icon" aria-hidden="true" />
            <span>{{ device.label }}</span>
          </span>
        </span>
      </div>

      <!-- 可见范围：仅作者可见（评论）——非内容作者/评论者只看到占位提示 -->
      <div v-if="isAuthorOnly" class="comment-author-only">
        <i class="bi bi-eye-slash" aria-hidden="true" /> 该评论仅作者可见
      </div>
      <!-- 可见范围：登录可见的评论（评论不支持付费可见），后端已清空正文 -->
      <ContentLock
        v-else-if="commentLocked"
        class="comment-lock"
        :state="comment.scope_state"
        :integral="0"
        bind-type="comment"
        :bind-id="comment.id"
        compact
        @unlocked="$emit('unlocked', comment)"
      />
      <div
        v-else
        class="comment-content"
        :class="{ 'comment-content-link': authorLink }"
        v-html="renderedContent"
        @click="goAuthor"
      ></div>

      <!-- 评论图片：实况图用 LivePhoto（封面 + 悬停/按住播放），与动态保持一致 -->
      <div v-if="images.length && !commentLocked" class="comment-images">
        <template v-for="(img, i) in images" :key="`${img.image}-${i}`">
          <LivePhoto
            v-if="img.video"
            class="comment-img comment-img-live"
            :image="img.image"
            :video="img.video"
            alt="评论实况图"
            fit="cover"
            @click="preview(i)"
          />
          <img
            v-else
            class="comment-img"
            :src="img.image"
            alt="评论图片"
            loading="lazy"
            @click="preview(i)"
            @error="onImgError"
          />
        </template>
      </div>

      <div class="comment-actions">
        <button v-if="!isAuthorOnly" ref="likeBtn" class="comment-action like-btn" :class="{ 'is-active': comment.liked }" @click="$emit('like', comment)">
          <span class="comment-icon"><i ref="likeIcon" class="bi" :class="comment.liked ? 'bi-heart-fill' : 'bi-heart'" /></span>
          <span>{{ comment.likeCount || 0 }}</span>
        </button>
        <button v-if="allowComment && !isAuthorOnly" class="comment-action" @click="$emit('reply', comment)">
          <span class="comment-icon"><i class="bi bi-chat-dots" /></span> 回复
        </button>
        <!-- 打赏：自己的评论不显示；弹窗由父级 CommentTree 统一承载（同举报） -->
        <button
          v-if="canTip && !isAuthorOnly"
          class="comment-action"
          title="打赏这条评论"
          @click="$emit('tip', comment)"
        >
          <span class="comment-icon"><i class="bi bi-coin" /></span> 打赏
        </button>
        <button v-if="canDelete" class="comment-action comment-del" @click="$emit('remove', comment)">删除</button>
        <!-- 举报：自己的评论不显示（举报弹窗由父级 CommentTree 统一承载，避免每条评论各挂一个弹窗）；
             看不到正文的「仅作者可见」评论同样不提供举报入口 -->
        <button
          v-if="canReport && !isAuthorOnly"
          class="comment-action"
          title="举报这条评论"
          @click="$emit('report', comment)"
        >
          <span class="comment-icon"><i class="bi bi-flag" /></span> 举报
        </button>
      </div>

      <!-- 回复框（禁止评论时不展示） -->
      <div v-if="allowComment && replyTo && replyTo.id === comment.id" class="comment-reply-box">
        <EmojiEditor
          v-model="replyText"
          :placeholder="`回复 @${replyTo.name}：`"
        >
          <template #extra>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :class="{ 'is-loading': uploading }"
              title="上传图片"
              @click="fileInput?.click()"
            >
              <i class="bi bi-image" />
              {{ uploading ? '上传中…' : '上传图片' }}
            </button>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              title="从附件库选择已有图片"
              @click="showLibrary = true"
            >
              <i class="bi bi-folder2-open" /> 附件库
            </button>
          </template>
        </EmojiEditor>
        <div v-if="replyImages.length" class="comment-reply-images">
          <div v-for="(img, i) in replyImages" :key="img.image || i" class="reply-thumb">
            <img :src="img.image" alt="" loading="lazy" />
            <!-- 实况图：本地缩略图只展示封面，角标提示已识别到视频 -->
            <span v-if="img.video" class="thumb-live">LIVE</span>
            <span class="img-del" title="移除" @click="removeImage(i)">×</span>
          </div>
        </div>
        <input
          ref="fileInput"
          type="file"
          accept="image/*"
          multiple
          class="hidden-file"
          @change="onPickImages"
        />

        <!-- 附件库（选择已上传图片加入回复图片） -->
        <AttachmentLibrary
          v-model:visible="showLibrary"
          title="选择回复图片"
          accept="image"
          multiple
          :max="9"
          @select="onLibrarySelect"
        />
        <div class="comment-reply-actions">
          <!-- 回复可见范围：公开 / 登录可见 / 仅作者可见（评论不支持付费可见），用单选框平铺 -->
          <ScopeRadio v-model="replyScope" :options="scopeOptions" class="reply-scope" />
          <span v-if="replyImages.length" class="img-tip">已选 {{ replyImages.length }} 张</span>
          <button class="btn btn-sm" @click="cancelReply">取消</button>
          <button
            class="btn btn-primary btn-sm"
            :disabled="(!replyText.trim() && !replyImages.length) || uploading"
            @click="submitReply"
          >
            发送
          </button>
        </div>
      </div>

      <!-- 子评论 -->
      <div v-if="comment.replies && comment.replies.length" class="comment-children">
        <CommentItem
          v-for="child in comment.replies"
          :key="child.id"
          :comment="child"
          :bind-type="bindType"
          :is-child="true"
          :author-id="authorId"
          :allow-comment="allowComment"
          :reply-to="replyTo"
          :highlight-id="highlightId"
          @like="$emit('like', $event)"
          @reply="$emit('reply', $event)"
          @submit="$emit('submit', $event)"
          @remove="$emit('remove', $event)"
          @report="$emit('report', $event)"
          @unlocked="$emit('unlocked', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { formatDate } from '@/utils/time'
import { useUserStore } from '@/stores/user'
import { isAdmin, pickCommentAuthor, getTitleColorClass, pickUserTitlePayload } from '@/utils/helper'
import AvatarFrame from './AvatarFrame.vue'
import TitleBadge from './TitleBadge.vue'
import WearingBadge from './WearingBadge.vue'
import { renderEmojiWithBreaks } from '@/utils/emoji'
import { parseUserAgent } from '@/utils/ua'
import { popIcon, popOut, burstHeart } from '@/utils/likeFx'
import EmojiEditor from './EmojiEditor.vue'
import AttachmentLibrary from './AttachmentLibrary.vue'
import ContentLock from './ContentLock.vue'
import ScopeRadio from './ScopeRadio.vue'
import LivePhoto from './LivePhoto.vue'
import { uploadCommentImages } from '@/api/comment'
import { openLightbox } from '@/utils/lightbox'
import { parseCommentMedia } from '@/utils/comment'
import { serializeLiveItem } from '@/utils/livePhoto'
import { toast } from '@/utils/toast'

const props = defineProps({
  comment: { type: Object, required: true },
  bindType: { type: String, default: 'article' },
  isChild: { type: Boolean, default: false },
  authorId: { type: [String, Number], default: null },
  replyTo: { type: Object, default: null },
  highlightId: { type: [String, Number], default: null },
  // 是否允许评论：false 时隐藏「回复」按钮与回复框
  allowComment: { type: Boolean, default: true }
})

const emit = defineEmits(['like', 'reply', 'submit', 'remove', 'report', 'unlocked', 'tip'])

const userStore = useUserStore()
const router = useRouter()
const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'
const replyText = ref('')

// 回复的可见范围（后端 scope）：1 公开 / 2 登录可见 / 4 仅作者可见
// 评论不提供「付费可见」（后端会把 3 降级为 2）
const replyScope = ref(1)
const scopeOptions = [
  { value: 1, label: '公开可见' },
  { value: 2, label: '登录可见' },
  { value: 4, label: '仅作者可见' }
]

// 后端把评论作者放在 result.author（原始对象里带 users.json，头衔样式要从这里取）
const rawAuthor = computed(
  () => props.comment?.result?.author || props.comment?.author || props.comment?.user || null
)
const author = computed(() => pickCommentAuthor(props.comment))
const renderedContent = computed(() => renderEmojiWithBreaks(props.comment.content))

// 仅作者可见：后端把正文清空，只回传 scope_state = 'author'
const isAuthorOnly = computed(() => props.comment?.scope_state === 'author')

// 可见范围：scope_state 为 login / pay / author 时说明后端已按可见范围清空正文
const commentLocked = computed(() => {
  const state = props.comment?.scope_state
  return state === 'login' || state === 'pay' || state === 'author'
})

// 评论者 UA（后端 comment.agent；普通用户看到的是脱敏后的截断值，管理员是完整值）
const agentRaw = computed(() => String(props.comment.agent || '').trim())
// 解析成简短设备描述（如「Windows 10 · Chrome 120」），无法识别时 label 为空，标签自动隐藏
const device = computed(() => parseUserAgent(agentRaw.value))

// 评论图片：后端以逗号分隔字符串存储；实况图是「图片::视频」，解析成 { image, video }
const images = computed(() => parseCommentMedia(props.comment.images))

// 点击图片打开系统全局灯箱（支持左右切换、缩放、键盘操作；实况图带 video 才有「播放实况」按钮）
function preview(index) {
  openLightbox(images.value, index)
}

// 作者主页链接（作者 id 存在时才可点击）
const authorLink = computed(() => {
  const id = author.value?.id
  return id ? `/author/${id}` : ''
})
function goAuthor() {
  if (authorLink.value) router.push(authorLink.value)
}

// 头衔配色（复用作者主页映射）：仅在没有装扮样式时兜底
const titleClass = computed(() => getTitleColorClass(author.value.title))
// 头衔样式（装扮系统写入的 payload：文字颜色 / 背景 / 边框 / 发光 / 图标 / 动效）
const titlePayload = computed(() => pickUserTitlePayload(rawAuthor.value))

const parentName = computed(() => props.comment.parentName || '')

const canDelete = computed(() => {
  const uid = userStore.user?.id
  const cid = author.value.id
  return !!(uid && (String(uid) === String(cid) || isAdmin(userStore.user) || props.authorId === uid))
})

// 举报入口：未登录不显示；自己的评论不显示（能删的评论没必要举报）
const canReport = computed(() => {
  const me = userStore.user?.id
  if (!me || canDelete.value) return false
  const cid = author.value?.id
  return String(cid || '') !== String(me)
})

// 打赏入口：自己的评论不显示；未登录也显示（点开弹窗会引导登录）
const canTip = computed(() => {
  const cid = author.value?.id
  if (!cid) return false
  const me = userStore.user?.id
  return !me || String(cid) !== String(me)
})

function onImgError(e) {
  e.target.src = defaultAvatar
}

const likeIcon = ref(null)
const likeBtn = ref(null)

// 点赞状态由父组件（CommentTree）乐观更新，此处监听触发动画
watch(
  () => props.comment.liked,
  (val, old) => {
    if (val === old || val === undefined) return
    if (val) {
      popIcon(likeIcon.value)
      burstHeart(likeBtn.value)
    } else {
      popOut(likeIcon.value)
    }
  }
)

const replyImages = ref([])
const uploading = ref(false)
const fileInput = ref(null)

async function onPickImages(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  if (!files.length) return

  if (replyImages.value.length + files.length > 9) {
    toast.warning('最多只能上传 9 张图片')
    return
  }
  const valid = files.filter((file) => {
    if (!file.type.startsWith('image/')) {
      toast.warning(`文件「${file.name}」不是图片`)
      return false
    }
    if (file.size > 10 * 1024 * 1024) {
      toast.warning(`图片「${file.name}」超过 10MB 限制`)
      return false
    }
    return true
  })
  if (!valid.length) return

  uploading.value = true
  try {
    const fd = new FormData()
    valid.forEach((file) => fd.append('files', file))
    const res = await uploadCommentImages(fd)
    const results = res.data?.results || []
    // 实况图：后端会把 JPEG 尾部内嵌的 MP4 拆出来单独存一条附件，地址在 live_video
    const rows = results.filter((r) => r.status !== 'fail' && r.full_url)
    replyImages.value.push(...rows.map((r) => ({ image: r.full_url, video: r.live_video || '' })))
    const liveCount = rows.filter((r) => r.live_video).length
    if (liveCount) toast.success(`已识别 ${liveCount} 张实况图`)
    if (results.some((r) => r.status === 'fail')) toast.warning('部分图片上传失败')
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

function removeImage(i) {
  replyImages.value.splice(i, 1)
}

// 附件库：把选中的已上传图片加入回复图片（去重 + 数量上限）
// items 里带 live_video（后端按「<封面名>-live.mp4」反查配对视频），
// 从库里选历史实况图时也能直接成为实况图
const showLibrary = ref(false)
function onLibrarySelect(urls = [], items = []) {
  const rest = 9 - replyImages.value.length
  if (rest <= 0) {
    toast.warning('最多只能上传 9 张图片')
    return
  }
  const exists = new Set(replyImages.value.map((item) => item.image))
  const liveByUrl = new Map(items.map((item) => [item.full_url, item.live_video || '']))
  const add = urls.filter((url) => url && !exists.has(url)).slice(0, rest)
  replyImages.value.push(...add.map((url) => ({ image: url, video: liveByUrl.get(url) || '' })))
  if (!add.length) {
    toast.info('所选图片已在列表中')
  } else if (add.length < urls.length) {
    toast.warning('受数量限制，部分附件未添加')
  } else {
    toast.success(`已添加 ${add.length} 张图片`)
  }
}

function cancelReply() {
  replyText.value = ''
  replyImages.value = []
  replyScope.value = 1
  emit('reply', { id: null })
}

function submitReply() {
  if (!replyText.value.trim() && !replyImages.value.length) return
  emit('submit', {
    content: replyText.value.trim(),
    // 实况图序列化成「图片::视频」，后端按逗号拼接后原样存库
    images: replyImages.value.map(serializeLiveItem).filter(Boolean),
    pid: props.comment.id,
    scope: Number(replyScope.value)
  })
  replyText.value = ''
  replyImages.value = []
  replyScope.value = 1
}
</script>

<style scoped>
.comment-item {
  display: flex;
  gap: 12px;
  padding: 16px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.comment-item:last-child {
  border-bottom: none;
}
/* 头像由 AvatarFrame 渲染（自带圆形裁剪 / 占位 / 角标），这里只管占位与不换行 */
.comment-avatar {
  display: flex;
  flex-shrink: 0;
}
.comment-body {
  flex: 1;
  min-width: 0;
}
.comment-head {
  display: flex;
  align-items: center;
  /* 横向 8px 保持昵称/等级/头衔之间呼吸感，纵向 2px 收紧昵称与日期两行间距 */
  gap: 2px 8px;
  font-size: 13px;
  flex-wrap: wrap;
}
.comment-name {
  font-weight: 500;
  color: var(--text);
}
.comment-name-link {
  cursor: pointer;
  transition: color 0.2s;
}
.comment-name-link:hover {
  color: var(--primary);
}
.level-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-xs);
  background: var(--accent-soft);
  color: var(--primary-deep);
  flex-shrink: 0;
}
.title-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-xs);
  font-weight: 600;
  color: #fff;
  flex-shrink: 0;
}
/* 头衔配色（与 Profile 页 10 个头衔一致） */
.title-zhangmen { background: linear-gradient(135deg, #c8a04a, #b07d2e); }
.title-zhanglao { background: linear-gradient(135deg, #8e6f3e, #6c4f24); }
.title-hufa { background: linear-gradient(135deg, #c0392b, #a93226); }
.title-neimen { background: linear-gradient(135deg, #2980b9, #1f618d); }
.title-waimen { background: linear-gradient(135deg, #16a085, #117a65); }
.title-lianqi { background: linear-gradient(135deg, #27ae60, #1e8449); }
.title-zhuji { background: linear-gradient(135deg, #7cb342, #558b2f); }
.title-jiedan { background: linear-gradient(135deg, #e67e22, #ca6f1e); }
.title-yuanying { background: linear-gradient(135deg, #6c5ce7, #5a3fd4); }
.title-huashen { background: linear-gradient(135deg, #f6d365, #fda085); text-shadow: 0 1px 2px rgba(0, 0, 0, 0.25); }
.title-xiake { background: linear-gradient(135deg, #4a9e6f, #2f7d52); }
.title-xuetu { background: linear-gradient(135deg, #9aa0a6, #6c757d); }
.title-default { background: #6c757d; }
.comment-badge {
  font-size: 10px;
  padding: 0 5px;
  border-radius: 3px;
  background: var(--accent-soft);
  color: var(--accent);
}
/* 可见范围徽标（仅作者可见的私密评论） */
.comment-scope {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 10px;
  padding: 0 5px;
  border-radius: 3px;
  background: var(--bg-muted);
  color: var(--text-muted);
  border: 1px dashed var(--border-soft);
}
/* 时间 + 设备：独占一行，显示在昵称下面 */
.comment-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px 10px;
  flex-basis: 100%;
}
.comment-time {
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.3;
  font-variant-numeric: tabular-nums;
}
/* 设备标签：弱化处理，不抢正文注意力；hover 显示原始 UA（title） */
.comment-device {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-muted);
  cursor: default;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.comment-device > i {
  font-size: 12px;
  flex-shrink: 0;
  color: var(--text-light);
}
.comment-del {
  color: var(--text-muted);
  font-size: 13px;
}
.comment-del:hover {
  color: var(--accent);
}
.comment-content {
  font-size: 15px;
  line-height: 1.7;
  color: var(--text-soft);
  margin: 8px 0;
  word-break: break-word;
}
/* 仅作者可见：非内容作者的占位提示（正文已由后端清空） */
.comment-author-only {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin: 8px 0;
  padding: 4px 10px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-muted);
  background: var(--bg-muted);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
}
.comment-content-link {
  cursor: pointer;
  transition: color 0.2s;
}
.comment-content-link:hover {
  color: var(--primary);
}
.comment-reply-to {
  color: var(--primary);
}
.comment-actions {
  display: flex;
  gap: 16px;
}
.comment-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: none;
  background: none;
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  padding: 0;
  transition: color var(--dur) var(--ease);
}
.comment-action:hover {
  color: var(--primary);
}
.like-btn {
  position: relative;
}
.comment-action.is-active {
  color: var(--accent);
}
.comment-icon {
  font-size: 14px;
}
.comment-reply-box {
  margin-top: 10px;
}
.comment-reply-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
}
/* 回复可见范围：选择器靠左，取消 / 发送靠右 */
.reply-scope {
  margin-right: auto;
}

/* ===== 评论图片 ===== */
.comment-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 8px 0;
}
.comment-img {
  width: 88px;
  height: 88px;
  object-fit: cover;
  border-radius: var(--radius-xs);
  border: 1px solid var(--border);
  cursor: zoom-in;
  transition: transform var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}
.comment-img:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow);
}
/* 实况图：与普通评论图同尺寸，悬停/按住时的播放由 LivePhoto 内部处理 */
.comment-img-live {
  width: 88px;
  height: 88px;
  flex: none;
  border: 1px solid var(--border);
  cursor: zoom-in;
  transition: transform var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
}
.comment-img-live:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow);
}
/* 已选缩略图上的实况角标（缩略图只显示封面） */
.thumb-live {
  position: absolute;
  left: 2px;
  bottom: 2px;
  padding: 0 4px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 1.6;
  pointer-events: none;
}

/* ===== 回复框图片上传 ===== */
.hidden-file {
  display: none;
}
.img-tip {
  font-size: 12px;
  color: var(--text-muted);
  margin-right: auto;
}
.comment-reply-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.reply-thumb {
  position: relative;
  width: 56px;
  height: 56px;
  border-radius: var(--radius-xs);
  overflow: hidden;
  border: 1px solid var(--border);
}
.reply-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.img-del {
  position: absolute;
  top: 0;
  right: 0;
  width: 16px;
  height: 16px;
  line-height: 15px;
  text-align: center;
  font-size: 13px;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
  border-radius: 0 0 0 var(--radius-xs);
  cursor: pointer;
}
.img-del:hover {
  background: var(--accent);
}
.comment-children {
  margin-top: 12px;
  padding-left: 14px;
  border-left: 2px solid var(--border);
}

/* ========== 移动端适配 ========== */
@media (max-width: 768px) {
  .comment-item {
    gap: 10px;
    padding: 14px 0;
  }
  /* AvatarFrame 的尺寸是内联 style（40px），移动端要缩小只能 !important */
  .comment-avatar :deep(.avatar-frame) {
    width: 36px !important;
    height: 36px !important;
  }
  .comment-time {
    font-size: 12px;
  }
  .comment-meta {
    gap: 2px 8px;
  }
  .comment-device {
    font-size: 11px;
  }
  /* 子评论缩进收紧，更适配窄屏 */
  .comment-children {
    padding-left: 10px;
  }
}
</style>
