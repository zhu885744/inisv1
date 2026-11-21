<template>
  <div class="moment-card">
    <div class="moment-head">
      <!-- 作者头像 + 装扮头像框（与侧栏 / 作者主页同一套渲染参数） -->
      <AvatarFrame
        class="avatar"
        :src="author?.avatar"
        :frame="author?.json?.frame"
        :frame-hole="frameRender.hole"
        :frame-offset-x="frameRender.offsetX"
        :frame-offset-y="frameRender.offsetY"
        :animated="frameRender.animated"
        :fallback="defaultAvatar"
        :alt="author?.nickname"
        :verified="author?.cert"
        size="44px"
        @click="goAuthor"
      />
      <div class="info">
        <div class="name">
          <span v-if="authorLink" class="name-link" @click="goAuthor">{{ author?.nickname || '匿名' }}</span>
          <span v-else>{{ author?.nickname || '匿名' }}</span>
          <span v-if="authorLevel" class="level-tag">Lv.{{ authorLevel }}</span>
          <!-- 有装扮样式用 TitleBadge 还原后台配色；没有则回退旧色板 -->
          <TitleBadge
            v-if="authorTitle || authorTitlePayload"
            :payload="authorTitlePayload || {}"
            :text="authorTitle"
            size="sm"
          />
          <span v-else-if="authorTitle" :class="['title-tag', authorTitleClass]">{{ authorTitle }}</span>
          <!-- 佩戴中的勋章（作者信息带 users.json） -->
          <WearingBadge v-if="author?.json?.wearing_badge" :badge="author.json.wearing_badge" />
          <span v-if="moment.top" class="top-tag">置顶</span>
        </div>
        <div class="time">
          <span>{{ fromNow(moment.create_time) }}</span>
          <!-- 当前可见范围：仅未登录访客显示，跟时间同排，避免正文区多出一条突兀的标签 -->
          <ScopeBadge
            v-if="showScopeNotice"
            :scope="moment.scope"
            :integral="moment.scope_integral"
          />
        </div>
      </div>
      <!-- 右上角三点菜单：分享 / 打赏 / 举报 / 置顶 / 删除，卡片底部只留互动按钮 -->
      <CardActionMenu :items="actionMenuItems" @select="onActionMenu" />
    </div>
    <!-- 可见范围：登录可见 / 付费可见时后端已清空正文，这里展示引导并支持积分兑换 -->
    <ContentLock
      v-if="momentLocked"
      :state="moment.scope_state"
      :integral="Number(moment.scope_integral) || 0"
      bind-type="moments"
      :bind-id="moment.id"
      compact
      @unlocked="onUnlocked"
    />
    <div v-else class="moment-content" v-html="renderedContent"></div>
    <div
      v-if="shownImages.length"
      class="moment-images"
      :class="'img-grid-' + getGridType(shownImages.length)"
    >
      <div v-for="(img, i) in shownImages" :key="img.image || i" class="moment-img-wrap">
        <!-- 实况图：静态封面 + 悬停/按住播放的短视频（见 components/LivePhoto.vue） -->
        <LivePhoto
          v-if="img.video"
          :image="thumbUrl(img.image, previewThumbWidth)"
          :video="img.video"
          :square="shownImages.length > 1"
          :fit="shownImages.length > 1 ? 'cover' : 'contain'"
          class="moment-img-live"
          @click="previewImage(i)"
        />
        <img
          v-else
          :src="thumbUrl(img.image, previewThumbWidth)"
          class="moment-img"
          loading="lazy"
          decoding="async"
          @click="previewImage(i)"
        />
        <!-- 超出预览数量的图片：最后一张压一层「+N」，点击进详情看全部 -->
        <span
          v-if="hiddenImageCount && i === shownImages.length - 1"
          class="moment-img-more"
          @click="goDetail"
        >+{{ hiddenImageCount }}</span>
      </div>
    </div>
    <!-- 列表卡片只展示一部分：正文被截断 / 还有图没展示时给入口 -->
    <button v-if="contentTruncated || hiddenImageCount" type="button" class="moment-more" @click="goDetail">
      查看更多
      <i class="bi bi-arrow-right" aria-hidden="true" />
    </button>
    <div v-if="moment.location" class="moment-loc">
      <i class="bi bi-geo-alt" /> {{ moment.location }}
    </div>
    <!-- 底部只保留高频互动：点赞 / 评论 / 阅读（分享、打赏、举报在右上角三点菜单） -->
    <div class="moment-actions">
      <!-- 内容类型标注：首页混排里一眼分清动态和长文 -->
      <span class="kind-tag"><i class="bi bi-chat-square-text" /> 动态</span>
      <button
        ref="likeBtn"
        class="action-item like-btn push-right"
        :class="{ 'is-liked': liked }"
        title="点赞"
        @click="toggleLike"
      >
        <span class="like-heart"><i ref="likeHeart" class="bi" :class="liked ? 'bi-heart-fill' : 'bi-heart'" /></span>
        <span class="like-count">{{ likeCount }}</span>
      </button>
      <button
        v-if="commentShow"
        class="action-item"
        :class="{ 'is-open': showComments }"
        :title="showComments ? '收起评论' : '查看评论'"
        @click="toggleComments"
      >
        <i class="bi" :class="showComments ? 'bi-chat-dots-fill' : 'bi-chat-dots'" />
        <span>{{ commentCount }}</span>
      </button>
      <span class="action-item is-static" title="阅读量">
        <i class="bi bi-eye" />
        <span>{{ moment.views || 0 }}</span>
      </span>
    </div>

    <ReportDialog
      :visible="reportVisible"
      target-type="moment"
      :target-id="moment.id"
      :target-label="reportLabel"
      :loading="reporting"
      @update:visible="(v) => (reportVisible = v)"
      @confirm="submitReport"
    />

    <!-- 打赏：积分打赏（自己的动态不显示入口） -->
    <RewardDialog
      v-model:visible="rewardVisible"
      bind-type="moments"
      :bind-id="moment.id"
      :to-uid="authorId"
      :to-name="author?.nickname || ''"
    />

    <!-- 动态评论：allow 为 2 时只关掉发表入口，show 为 2 时整个评论入口都不显示 -->
    <div v-if="showComments && commentShow" class="moment-comments">
      <CommentTree
        :bind-id="moment.id"
        bind-type="moments"
        :author-id="moment.uid"
        :allow-comment="commentAllow"
        :highlight-id="highlightId"
        @loaded="(n) => commentCount = n"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { fromNow } from '@/utils/time'
import {
  isAdmin as helperIsAdmin,
  pickUserLevel,
  pickUserTitle,
  pickUserTitlePayload,
  getTitleColorClass
} from '@/utils/helper'
import TitleBadge from './TitleBadge.vue'
import CardActionMenu from './CardActionMenu.vue'
import WearingBadge from './WearingBadge.vue'
import AvatarFrame from './AvatarFrame.vue'
import { resolveFrameRender } from '@/utils/frameRender'
import { renderEmojiWithBreaks } from '@/utils/emoji'
import ContentLock from './ContentLock.vue'
import ScopeBadge from './ScopeBadge.vue'
import { getMoment } from '@/api/moments'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import CommentTree from './CommentTree.vue'
import ReportDialog from './ReportDialog.vue'
import RewardDialog from './RewardDialog.vue'
import { createReport } from '@/api/report'
import { call } from '@/api/request'
import { like, unlike, isLiked, likesCount } from '@/api/tags'
import { openLightbox } from '@/utils/lightbox'
import LivePhoto from './LivePhoto.vue'
import { parseMomentMedia } from '@/utils/moment'
import { thumbUrl } from '@/utils/image'
import { popIcon, popOut, burstHeart } from '@/utils/likeFx'
import { useRouter } from 'vue-router'

const props = defineProps({
  moment: { type: Object, required: true },
  highlightId: { type: [String, Number], default: null },
  forceOpen: { type: Boolean, default: false },
  /**
   * 列表页批量下发的互动状态（见 views/home/Index.vue）
   *
   * 有值时不再自己请求 is-liked / counts / comment/count ——
   * 否则每条动态 2~3 个请求，滚动几屏就是上百个。
   * 不传（详情页等单独使用场景）时保持原有单条查询。
   */
  likeState: { type: Object, default: null },
  commentTotal: { type: Number, default: undefined },
  /**
   * 列表卡片模式：正文按可见字符截断、图片只展示前几张，其余走「查看更多」进详情页
   *
   * 详情页（views/moments/Detail.vue）不传，保持全文 + 全部图片。
   */
  compact: { type: Boolean, default: false }
})
const emit = defineEmits(['delete', 'refresh'])

const userStore = useUserStore()
const router = useRouter()

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'

// 头像加载失败的回退由 AvatarFrame 内部处理（fallback 属性），这里不再需要手动兜底

const showComments = ref(false)

// 可见范围：scope_state 为 login / pay 时说明后端已按可见范围清空正文
const momentLocked = computed(() => {
  const state = props.moment?.scope_state
  return state === 'login' || state === 'pay'
})

// 兑换成功后重新拉取该动态（后端已放行，正文会正常返回）。
// 列表里拿到的是同一个对象引用，直接合并即可让卡片与父级列表同步刷新
async function onUnlocked() {
  try {
    const res = await getMoment(props.moment.id)
    if (res?.data) {
      // 放行后的返回体不再带 scope_state，必须先删掉旧值，否则遮罩不会消失
      delete props.moment.scope_state
      Object.assign(props.moment, res.data)
    }
  } catch {
    emit('refresh', props.moment)
  }
}

// 评论开关：取自动态 result.comment（后端已按 MOMENTS 总开关做覆盖与继承）
// allow：1 允许 / 2 禁止（禁止仍展示已有评论，只关掉发表与回复入口）
// show：1 显示 / 2 隐藏（隐藏则整个评论入口都不显示）
const commentAllow = computed(() => Number(props.moment?.result?.comment?.allow ?? 1) !== 2)
const commentShow = computed(() => Number(props.moment?.result?.comment?.show ?? 1) !== 2)

// 详情页（forceOpen）或带 ?comment 跳入时，自动展开该动态评论区
onMounted(() => {
  if (commentShow.value && (props.forceOpen || props.highlightId)) {
    showComments.value = true
  }
})
const likeBtn = ref(null)
const likeHeart = ref(null)

// 点赞状态（参考 Cardify MomentCard：is-liked + counts 初始化，乐观更新）
const liked = ref(false)
const likeCount = ref(0)

/**
 * 应用列表批量下发的点赞状态
 * @returns {boolean} 是否真的应用了（false 表示没有带来状态，调用方应自行查询）
 */
function applyLikeState(state) {
  if (!state || typeof state !== 'object') return false
  liked.value = !!state.is_liked
  likeCount.value = Number(state.count) || 0
  return true
}

async function loadLikeState() {
  try {
    if (userStore.isLogged) {
      const res = await isLiked('moment', props.moment.id)
      // is-liked 返回 { data: { is_liked, count } }
      liked.value = !!(res?.data?.is_liked ?? res?.data)
      likeCount.value = res?.data?.count ?? res?.data?.total ?? 0
    } else {
      const res = await likesCount('moment', [props.moment.id])
      // counts 返回 { data: { counts: { [id]: n } } }
      const counts = res?.data?.counts || res?.data?.data?.counts || {}
      likeCount.value = counts[props.moment.id] || 0
    }
  } catch {
    /* 保持默认值 */
  }
}

function requireLogin() {
  if (!userStore.isLogged) {
    toast.warning('请先登录')
    router.push({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
    return false
  }
  return true
}

async function toggleLike() {
  if (!requireLogin()) return
  // 乐观更新，失败回滚
  const prevLiked = liked.value
  const prevCount = likeCount.value
  liked.value = !prevLiked
  likeCount.value = Math.max(0, prevCount + (liked.value ? 1 : -1))
  // 即时动画 + 消息提示
  if (liked.value) {
    popIcon(likeHeart.value)
    burstHeart(likeBtn.value)
    toast.success('点赞成功')
  } else {
    popOut(likeHeart.value)
    toast.info('已取消点赞')
  }
  try {
    if (liked.value) {
      await like('moment', props.moment.id)
    } else {
      await unlike('moment', props.moment.id)
    }
  } catch {
    liked.value = prevLiked
    likeCount.value = prevCount
    toast.error('操作失败，请重试')
  }
}

// 后端 moments 列表不返回 comment_count，需单独统计。
// 不能用 moments/comment_count：它只统计一级评论（pid=0）。
// 用 comment/count 统计全部评论（含二级及更深的回复），但注意：
// 该接口的 bind_id/bind_type 快捷参数实测无效（会返回全站评论总数），
// 必须走 where 条件过滤，与 countArticlesByAuthor 等先例一致
const commentCount = ref(
  typeof props.commentTotal === 'number' ? props.commentTotal : Number(props.moment.comment_count || 0)
)
async function loadCommentCount() {
  try {
    const res = await call('comment', 'count', {
      method: 'GET',
      params: {
        where: { bind_id: props.moment.id, bind_type: 'moments' }
      }
    })
    commentCount.value = res.data || 0
  } catch {
    /* 保持现有值 */
  }
}

// 列表页已经批量下发时不再单条查询（首页一页十几条动态，逐个查就是十几个请求）
if (!applyLikeState(props.likeState)) loadLikeState()
if (typeof props.commentTotal !== 'number') loadCommentCount()

// 批量结果晚于卡片挂载时补上（不会触发重复请求，只是覆盖显示）
watch(
  () => props.likeState,
  (state) => applyLikeState(state)
)
watch(
  () => props.commentTotal,
  (total) => {
    if (typeof total === 'number') commentCount.value = total
  }
)

const author = computed(() => props.moment.result?.author || props.moment.user)
// 作者信息：等级/头衔 + 跳转作者主页（与评论一致）
const authorLink = computed(() => {
  const id = author.value?.id || props.moment.uid
  return id ? `/author/${id}` : ''
})
const authorLevel = computed(() => pickUserLevel(author.value))
const authorTitle = computed(() => pickUserTitle(author.value))
const authorTitleClass = computed(() => getTitleColorClass(authorTitle.value))
// 头衔样式（装扮系统写入 users.json.title_decoration.payload）
const authorTitlePayload = computed(() => pickUserTitlePayload(author.value))
// 头像框渲染参数（洞大小 / 偏移 / 动效）：以装扮 payload 为准，users.json 副本兜底
const frameRender = computed(() => resolveFrameRender(author.value?.json))
function goAuthor() {
  if (authorLink.value) router.push(authorLink.value)
}

// ---------- 举报 ----------
// 提交后不改动卡片本身的状态（举报是「给管理员看」，不改变内容的可见性）
const reportVisible = ref(false)
const reporting = ref(false)
const canReport = computed(
  () => userStore.isLogged && Number(props.moment.uid) !== Number(userStore.user?.id)
)

// ---------- 打赏 ----------
const rewardVisible = ref(false)

const authorId = computed(() => Number(props.moment.uid || author.value?.id || 0))
// 自己的动态不打赏（后端同样会拒绝）；未登录也显示入口，点开会引导登录
const canReward = computed(() => authorId.value > 0 && Number(userStore.user?.id) !== authorId.value)

function openReward() {
  rewardVisible.value = true
}
// 弹窗里显示「动态「今天的风景」」，方便用户确认没点错条目
const reportLabel = computed(() => {
  const text = String(props.moment.content || '')
    .replace(/<[^>]*>/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  return text ? `动态「${text.slice(0, 20)}」` : '动态'
})
async function submitReport({ reason, detail, images }) {
  reporting.value = true
  try {
    await createReport({
      // 注意：举报接口的对象类型是单数 moment（与动态的 bind_type=moments 不同）
      target_type: 'moment',
      target_id: props.moment.id,
      reason,
      detail,
      images
    })
    reportVisible.value = false
    toast.success('举报已提交，我们会尽快处理')
  } catch {
    /* 拦截器已提示（重复举报 / 超过每日上限等） */
  } finally {
    reporting.value = false
  }
}
// 配图字段可能是数组、逗号分隔字符串或空；实况图项为 { image, video }
const imageList = computed(() => parseMomentMedia(props.moment.images))

/* ---------------- 列表卡片：正文 / 图片只展示一部分 ---------------- */
// 正文预览的可见字符数（一个表情 / @提及 / #话题# 都算 1 个）
const PREVIEW_TEXT_LIMIT = 180
// 最多展示几张图，其余压成「+N」
const PREVIEW_IMAGE_LIMIT = 3

// 正文里的非文本标记：截断时要整体跳过，不能切成半截源码
const CONTENT_MARKER_RE = /^(?:\[emoji:[^\]]*\]|@\[[^\]]*\]\(\/author\/\d+\)|#\[[^\]]*\]\(\/topic\/\d+\))/

/**
 * 按「可见字符」截断正文
 *
 * 直接 slice 会把 [emoji:xx] / @[昵称](/author/1) 这类标记切成两半，
 * v-html 之后会漏出 `[emoji:/assets/e` 这种半截源码，所以遇到标记整体跳过、只计 1 个字符。
 */
function clipContent(text, limit) {
  let out = ''
  let visible = 0
  let i = 0
  while (i < text.length && visible < limit) {
    const ch = text[i]
    if (ch === '[' || ch === '@' || ch === '#') {
      const marker = CONTENT_MARKER_RE.exec(text.slice(i))
      if (marker) {
        out += marker[0]
        i += marker[0].length
        visible += 1
        continue
      }
    }
    out += ch
    i += 1
    visible += 1
  }
  return { text: out, truncated: i < text.length }
}

const contentPreview = computed(() => clipContent(String(props.moment.content || ''), PREVIEW_TEXT_LIMIT))
const contentTruncated = computed(() => props.compact && contentPreview.value.truncated)
const renderedContent = computed(() =>
  renderEmojiWithBreaks(props.compact ? contentPreview.value.text : props.moment.content)
)

// 列表里只放前几张图，剩下的用「+N」引导进详情
const shownImages = computed(() =>
  props.compact ? imageList.value.slice(0, PREVIEW_IMAGE_LIMIT) : imageList.value
)
const hiddenImageCount = computed(() => imageList.value.length - shownImages.value.length)

// 列表里图片最大显示 320px（单图）/ 136px（九宫格），按 2 倍图取缩略图；
// 点开灯箱时仍用原图（见 previewImage）
const previewThumbWidth = computed(() => (getGridType(shownImages.value.length) === '1' ? 640 : 320))

// 「查看更多」/「+N」：进动态详情页（全文 + 全部图片 + 完整评论）
function goDetail() {
  router.push(`/moments/${props.moment.id}`)
}

// 多图网格类型：1 单图大图 / 2-3 三列 / 4 两列 / 5-9 九宫格
function getGridType(count) {
  if (count === 1) return '1'
  if (count <= 3) return '3'
  if (count <= 4) return '4'
  return '9'
}

const canDelete = computed(() => {
  return userStore.user?.id === props.moment.uid || helperIsAdmin(userStore.user)
})

// 仅管理员可置顶/删除（置顶按钮在删除按钮之前）
const canManage = computed(() => helperIsAdmin(userStore.user))

// 当前可见范围：**只给未登录访客看**（登录用户要么能看，要么已有兑换引导）
const showScopeNotice = computed(
  () => !userStore.isLogged && Number(props.moment.scope) > 1
)

async function toggleTop() {
  const next = props.moment.top ? 0 : 1
  try {
    await call('moments', 'set_top', {
      method: 'PUT',
      data: { ids: String(props.moment.id), top: next }
    })
    props.moment.top = next
    toast.success(next ? '已设为置顶' : '已取消置顶')
    emit('refresh')
  } catch {
    /* 拦截器已提示 */
  }
}

function toggleComments() {
  showComments.value = !showComments.value
}

// 分享：复制动态详情页链接
async function copyLink() {
  const url = `${window.location.origin}/moments/${props.moment.id}`
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(url)
    } else {
      const ta = document.createElement('textarea')
      ta.value = url
      ta.style.position = 'fixed'
      ta.style.opacity = '0'
      document.body.appendChild(ta)
      ta.select()
      document.execCommand('copy')
      document.body.removeChild(ta)
    }
    toast.success('链接已复制')
  } catch {
    toast.error('复制失败，请手动复制')
  }
}

function previewImage(i) {
  // 带上 video：灯箱里是手动点「播放实况」才播，不会一打开就自动放
  openLightbox(imageList.value, i)
}

// ---------- 右上角三点菜单 ----------
// 低频操作（分享 / 打赏 / 举报 / 置顶 / 删除）收进菜单，底部只留点赞、评论、阅读。
// 权限口径不变：打赏、举报对「自己的动态」隐藏；置顶仅管理员；删除按 canDelete。
const actionMenuItems = computed(() => {
  const items = [{ key: 'share', label: '分享', icon: 'bi bi-share' }]

  if (canReward.value) {
    items.push({ key: 'reward', label: '打赏', icon: 'bi bi-coin' })
  }
  if (canReport.value) {
    items.push({ key: 'report', label: '举报', icon: 'bi bi-flag' })
  }
  // 管理项口径与改版前一致（按钮原来就只在 canManage 下出现）：
  // 具体的删除动作由父级处理（首页会调接口，详情页只做跳转），这里不擅自改权限
  if (canManage.value) {
    items.push({
      key: 'top',
      label: props.moment.top ? '取消置顶' : '设为置顶',
      icon: props.moment.top ? 'bi bi-pin-fill' : 'bi bi-pin'
    })
    items.push({ key: 'delete', label: '删除', icon: 'bi bi-trash', danger: true })
  }

  return items
})

function onActionMenu(key) {
  switch (key) {
    case 'share':
      copyLink()
      break
    case 'reward':
      openReward()
      break
    case 'report':
      reportVisible.value = true
      break
    case 'top':
      toggleTop()
      break
    case 'delete':
      emit('delete', props.moment)
      break
  }
}
</script>

<style scoped>
.moment-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-soft);
  padding: 20px;
  margin-bottom: 16px;
  transition: box-shadow 0.2s;
}
.moment-card:hover {
  box-shadow: var(--shadow);
}
.moment-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
/* 尺寸由 AvatarFrame 的 size 属性控制（inline width/height），这里只管交互 */
.avatar {
  flex-shrink: 0;
  cursor: pointer;
}
.info {
  flex: 1;
  min-width: 0;
}
.name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  flex-wrap: wrap;
}
.name-link {
  cursor: pointer;
  transition: color 0.2s;
}
.name-link:hover {
  color: var(--primary);
}
.level-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  flex-shrink: 0;
}
.title-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
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
.top-tag {
  display: inline-block;
  margin-left: 6px;
  font-size: 11px;
  padding: 1px 6px;
  background: var(--accent-soft);
  color: var(--accent);
  border-radius: 3px;
}
.time {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}


.moment-content {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  margin-bottom: 12px;
  white-space: pre-wrap;
  word-break: break-word;
}
.moment-images {
  display: grid;
  gap: 6px;
  margin-bottom: 12px;
}
.moment-images.img-grid-1 {
  grid-template-columns: 1fr;
  max-width: 320px;
}
.moment-images.img-grid-3 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 320px;
}
.moment-images.img-grid-4 {
  grid-template-columns: repeat(2, 1fr);
  max-width: 320px;
}
.moment-images.img-grid-9 {
  grid-template-columns: repeat(3, 1fr);
  max-width: 420px;
}
/* 图片格子的包裹层：承载「+N」遮罩，同时作为 grid item 参与布局 */
.moment-img-wrap {
  position: relative;
  min-width: 0;
}
.moment-img {
  width: 100%;
  aspect-ratio: 1;
  object-fit: cover;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: transform 0.2s;
  background: var(--bg-muted);
  display: block;
}
.img-grid-1 .moment-img-wrap {
  justify-self: center;
}
.img-grid-1 .moment-img {
  width: auto;
  max-height: 260px;
  aspect-ratio: auto;
  object-fit: contain;
  background: none;
}
.moment-img:hover {
  transform: scale(1.02);
}
/* 实况图：与普通图同一套尺寸/圆角/交互观感（尺寸由组件内部按 square/fit 决定） */
.moment-img-live {
  cursor: pointer;
  transition: transform 0.2s;
}
.moment-img-live:hover {
  transform: scale(1.02);
}
/* 超出预览数量的图片：最后一张压一层「+N」 */
.moment-img-more {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.45);
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  user-select: none;
}
/* 「查看更多」：正文被截断 / 还有图没展示时出现 */
.moment-more {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin: -4px 0 12px;
  padding: 0;
  border: none;
  background: none;
  color: var(--primary);
  font-size: 13px;
  cursor: pointer;
}
.moment-more:hover {
  color: var(--primary-deep);
}
.moment-more i {
  font-size: 11px;
}
.moment-loc {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 12px;
}

/* 底部互动栏：只有点赞 / 评论 / 阅读，图标 + 数字，等间距排开 */
.moment-actions {
  display: flex;
  align-items: center;
  gap: 18px;
  padding-top: 10px;
  border-top: 1px dashed var(--border-soft);
  font-size: 13px;
  color: var(--text-muted);
}
/* 内容类型标注（动态 / 长文）：低对比度胶囊，靠左；互动按钮靠右 */
.kind-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 999px;
  white-space: nowrap;
}
.kind-tag i {
  font-size: 12px;
}
.push-right {
  margin-left: auto;
}
.action-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 0;
  font: inherit;
  color: inherit;
  background: none;
  border: none;
  cursor: pointer;
  transition: color var(--dur) var(--ease);
}
.action-item i {
  font-size: 15px;
}
.action-item:hover {
  color: var(--primary);
}
/* 只读的阅读数：不要 hover 变色，避免看着像可点 */
.action-item.is-static {
  cursor: default;
}
.action-item.is-static:hover {
  color: inherit;
}
/* 评论区展开时，评论按钮保持主色提示「再点一下收起」 */
.action-item.is-open {
  color: var(--primary);
}
.like-btn {
  position: relative;
}
.action-item .like-heart {
  display: inline-block;
  font-size: 15px;
  transition: transform 0.2s;
}
.action-item.is-liked {
  color: var(--accent);
}
.action-item.is-liked .like-heart {
  transform: scale(1.2);
}
.moment-comments {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
}
.moment-comment-box {
  margin-top: 12px;
}
.moment-comment-input {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 8px 10px;
  font-size: 13px;
  font-family: inherit;
  resize: vertical;
  background: var(--bg);
  color: var(--text);
}
.moment-comment-input:focus {
  outline: none;
  border-color: var(--primary);
}
.moment-comment-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 6px;
}
</style>
