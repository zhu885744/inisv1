<template>
  <article class="feed-article">
    <!-- 头部：作者信息（头像 / 昵称 / 等级 / 头衔 / 置顶），与动态卡片保持一致 -->
    <div class="feed-head">
      <!-- 作者头像 + 装扮头像框（与侧栏 / 作者主页同一套渲染参数） -->
      <AvatarFrame
        class="feed-avatar"
        :src="author?.avatar"
        :frame="author?.json?.frame"
        :frame-hole="frameRender.hole"
        :frame-offset-x="frameRender.offsetX"
        :frame-offset-y="frameRender.offsetY"
        :animated="frameRender.animated"
        :fallback="defaultAvatar"
        :alt="author?.nickname"
        :verified="author?.cert"
        size="40px"
        @click="goAuthor"
      />
      <div class="feed-info">
        <div class="feed-name">
          <span class="feed-name-link" @click="goAuthor">{{ author?.nickname || '匿名' }}</span>
          <span v-if="authorLevel" class="level-tag">Lv.{{ authorLevel }}</span>
          <TitleBadge
            v-if="authorTitle || authorTitlePayload"
            :payload="authorTitlePayload || {}"
            :text="authorTitle"
            size="sm"
          />
          <span v-else-if="authorTitle" :class="['title-tag', authorTitleClass]">{{ authorTitle }}</span>
          <!-- 佩戴中的勋章（作者信息带 users.json） -->
          <WearingBadge v-if="author?.json?.wearing_badge" :badge="author.json.wearing_badge" />
          <span v-if="article.top" class="feed-top">置顶</span>
        </div>
        <div class="feed-time">
          <span>{{ fromNow(article.publish_time || article.create_time) }}</span>
          <!-- 当前可见范围：仅未登录访客显示 -->
          <ScopeBadge
            v-if="showScopeNotice"
            :scope="article.scope"
            :integral="article.scope_integral"
          />
        </div>
      </div>
      <!-- 右上角三点菜单：分享 / 打赏 / 举报，卡片底部只留互动按钮 -->
      <CardActionMenu :items="actionMenuItems" @select="onActionMenu" />
    </div>

    <!-- 标题：文章比动态多的信息，点击进详情 -->
    <h3 class="feed-title" @click="goDetail">{{ article.title }}</h3>

    <!-- 正文片段：只展示前面一部分，超出给「查看更多」入口 -->
    <p v-if="previewText" class="feed-body">
      <EmojiText :text="previewText" :size="15" />
    </p>

    <button v-if="hasMore" class="feed-more" type="button" @click="goDetail">
      查看更多
      <i class="bi bi-arrow-right" aria-hidden="true" />
    </button>

    <!-- 首页不展示封面图：列表里只保留标题 + 正文片段，图片全部交给动态卡片 -->

    <!-- 底部互动：左侧「类型 + 分类」，右侧点赞 / 评论 / 阅读
         （分享、打赏、举报在右上角三点菜单）。分组是为了窄屏换行时仍然左标签右按钮，不会错位 -->
    <div class="feed-actions">
      <div class="action-tags">
        <span class="kind-tag"><i class="bi bi-file-text" /> 长文</span>
        <span v-if="groupName" class="kind-group" :title="groupName">{{ groupName }}</span>
      </div>
      <div class="action-btns">
        <button
          ref="likeBtn"
          class="feed-action like-btn"
          :class="{ 'is-liked': liked }"
          type="button"
          title="点赞"
          @click="toggleLike"
        >
          <i ref="likeIcon" class="bi" :class="liked ? 'bi-heart-fill' : 'bi-heart'" />
          <span>{{ likeCount }}</span>
        </button>
        <button
          v-if="showComment"
          class="feed-action"
          type="button"
          title="查看评论"
          @click="goDetail"
        >
          <i class="bi bi-chat-dots" />
          <span>{{ commentCount }}</span>
        </button>
        <span class="feed-action is-static" title="阅读量">
          <i class="bi bi-eye" />
          <span>{{ article.views || 0 }}</span>
        </span>
      </div>
    </div>

    <ReportDialog
      :visible="reportVisible"
      target-type="article"
      :target-id="article.id"
      :target-label="reportLabel"
      :loading="reporting"
      @update:visible="(v) => (reportVisible = v)"
      @confirm="submitReport"
    />

    <!-- 打赏：积分打赏（自己的文章不显示入口） -->
    <RewardDialog
      v-model:visible="rewardVisible"
      bind-type="article"
      :bind-id="article.id"
      :to-uid="authorUid"
      :to-name="author?.nickname || ''"
    />
  </article>
</template>

<script setup>
/**
 * 首页「发现」流里的文章卡片（动态风格）
 *
 * 与 ArticleCard（列表页用的通栏文章行）不同：这里头部是作者信息，正文是文章内容的
 * 纯文本片段，超出部分用「查看更多」引导到详情页，整体观感与 MomentItem 对齐。
 * 只在首页使用，列表页 / 搜索页仍用 ArticleCard。
 */
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { fromNow } from '@/utils/time'
import {
  truncate,
  copyText,
  pickUserLevel,
  pickUserTitle,
  pickUserTitlePayload,
  getTitleColorClass,
  stripArticleMarkup
} from '@/utils/helper'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import TitleBadge from './TitleBadge.vue'
import WearingBadge from './WearingBadge.vue'
import EmojiText from './EmojiText.vue'
import ScopeBadge from './ScopeBadge.vue'
import AvatarFrame from './AvatarFrame.vue'
import CardActionMenu from './CardActionMenu.vue'
import ReportDialog from './ReportDialog.vue'
import RewardDialog from './RewardDialog.vue'
import { like, unlike, isLiked, likesCount } from '@/api/tags'
import { createReport } from '@/api/report'
import { popIcon, popOut, burstHeart } from '@/utils/likeFx'
import { resolveFrameRender } from '@/utils/frameRender'

const props = defineProps({
  article: { type: Object, required: true },
  // 正文片段的最大字符数（超出显示「查看更多」）
  textLimit: { type: Number, default: 240 },
  /**
   * 列表页批量带下来的点赞状态：{ is_liked, count }
   *
   * 有值时不再自己请求 is-liked / counts —— 首页一页十几张卡片各查一次，
   * 滚动几屏就是上百个请求。不传（其它页面单独使用本组件）时保持原有单条查询。
   */
  likeState: { type: Object, default: null }
})

const router = useRouter()
const userStore = useUserStore()

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'

// 头像加载失败的回退由 AvatarFrame 内部处理（fallback 属性），这里不再需要手动兜底

const author = computed(() => props.article.result?.author || props.article.user || null)
const authorLink = computed(() => {
  const id = author.value?.id || props.article.uid
  return id ? `/author/${id}` : ''
})
const authorLevel = computed(() => pickUserLevel(author.value))
const authorTitle = computed(() => pickUserTitle(author.value))
const authorTitleClass = computed(() => getTitleColorClass(authorTitle.value))
const authorTitlePayload = computed(() => pickUserTitlePayload(author.value))
// 头像框渲染参数（洞大小 / 偏移 / 动效）：以装扮 payload 为准，users.json 副本兜底
const frameRender = computed(() => resolveFrameRender(author.value?.json))

const groupName = computed(() => {
  const g = props.article.result?.group
  if (!g) return ''
  return (Array.isArray(g) ? g[0] : g)?.name || ''
})

const commentCount = computed(() => props.article.result?.comment?.count || 0)
// json.comment.show 为 2（不显示评论）时，列表里也不展示评论数
const showComment = computed(() => Number(props.article.result?.comment?.show ?? 1) !== 2)

// 可见范围：受限时后端会清空 content（abstract 保留），这里按访客身份提示范围
const scopeLocked = computed(() => {
  const state = props.article.scope_state
  return state === 'login' || state === 'pay'
})
const showScopeNotice = computed(
  () => !userStore.isLogged && Number(props.article.scope) > 1
)

// 正文转纯文本（列表摘要用）：媒体短代码 / 图片 / 链接 / 残留标签都在
// utils/helper.js 的 stripArticleMarkup 里统一清掉，这里不再重复实现
const plainText = computed(() => {
  const raw = stripArticleMarkup(props.article.abstract || props.article.text || props.article.content || '')
  const title = String(props.article.title || '').trim()
  // 摘要通常从正文第一行（也就是标题）开始，和卡片上方的大标题重复，去掉
  if (!title || !raw.startsWith(title)) return raw
  return raw.slice(title.length).replace(/^[\s，。、,.：:·\-—]+/, '')
})

// truncate 会自动去掉残留标签并补 "..."
const previewText = computed(() => truncate(plainText.value, props.textLimit))
const hasMore = computed(
  () =>
    scopeLocked.value ||
    plainText.value.length > props.textLimit ||
    String(props.article.content || '').length > props.textLimit
)

function goDetail() {
  router.push(`/archives/${props.article.id}`)
}

function goAuthor() {
  if (authorLink.value) router.push(authorLink.value)
}

async function copyLink() {
  const url = `${window.location.origin}/archives/${props.article.id}`
  const ok = await copyText(`【${props.article.title}】${url}`)
  if (ok) toast.success('已复制链接和标题，去分享吧')
  else toast.error('复制失败，请手动复制')
}

/* ===== 底部互动：点赞 / 评论 / 阅读 =====
 * 列表接口的 result.like 是点赞用户 id 数组，用长度当点赞数；
 * 是否已点赞仍需按卡片查一次 is-liked（与动态卡片一致），换卡片状态即时。
 */
const likeBtn = ref(null)
const likeIcon = ref(null)
const liked = ref(false)
// 先用列表里自带的点赞数组兜个数，避免首帧显示 0 再跳变
const likeCount = ref(Number(props.article.result?.like?.length || props.article.likes || 0))

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
      const res = await isLiked('article', props.article.id)
      liked.value = !!(res?.data?.is_liked ?? res?.data)
      likeCount.value = res?.data?.count ?? 0
    } else {
      const res = await likesCount('article', [props.article.id])
      const counts = res?.data?.counts || res?.data?.data?.counts || {}
      likeCount.value = counts[props.article.id] || 0
    }
  } catch {
    /* 保持初值 */
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

  const prevLiked = liked.value
  const prevCount = likeCount.value
  liked.value = !prevLiked
  likeCount.value = Math.max(0, prevCount + (liked.value ? 1 : -1))

  if (liked.value) {
    popIcon(likeIcon.value)
    burstHeart(likeBtn.value)
    toast.success('点赞成功')
  } else {
    popOut(likeIcon.value)
    toast.info('已取消点赞')
  }

  try {
    if (liked.value) await like('article', props.article.id)
    else await unlike('article', props.article.id)
  } catch {
    liked.value = prevLiked
    likeCount.value = prevCount
    toast.error('操作失败，请重试')
  }
}

/* ===== 打赏 / 举报 ===== */
const authorUid = computed(() => Number(props.article.uid || author.value?.id || 0))
const rewardVisible = ref(false)
// 自己的文章不打赏（后端同样会拒绝）
const canReward = computed(() => authorUid.value > 0 && Number(userStore.user?.id) !== authorUid.value)

function openReward() {
  rewardVisible.value = true
}

const reportVisible = ref(false)
const reporting = ref(false)
// 未登录不显示入口；自己的文章也不显示（后端同样拒绝「举报自己」）
const canReport = computed(() => {
  const me = userStore.user?.id
  if (!me) return false
  return String(props.article.uid || author.value?.id || '') !== String(me)
})

// 弹窗里显示「文章「标题」」，避免在信息流里点错条目
const reportLabel = computed(() => `文章「${String(props.article.title || '').slice(0, 20)}」`)

async function submitReport({ reason, detail, images }) {
  reporting.value = true
  try {
    await createReport({ target_type: 'article', target_id: props.article.id, reason, detail, images })
    reportVisible.value = false
    toast.success('举报已提交，我们会尽快处理')
  } catch {
    /* 拦截器已提示（重复举报 / 超过每日上限等） */
  } finally {
    reporting.value = false
  }
}

/* ===== 右上角三点菜单 ===== */
// 低频操作收进菜单，底部只留点赞、评论、阅读
const actionMenuItems = computed(() => {
  const items = [{ key: 'share', label: '分享', icon: 'bi bi-share' }]

  if (canReward.value) items.push({ key: 'reward', label: '打赏', icon: 'bi bi-coin' })
  if (canReport.value) items.push({ key: 'report', label: '举报', icon: 'bi bi-flag' })

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
  }
}

onMounted(() => {
  // 列表页已经批量下发状态时不要再单独查一次（否则优化就白做了）
  if (!applyLikeState(props.likeState)) loadLikeState()
})

// 批量结果可能比卡片晚到（例如首屏先渲染、接口稍后返回）
watch(
  () => props.likeState,
  (state) => {
    applyLikeState(state)
  }
)
</script>

<style scoped>
/* 与 MomentItem / 全站 .card 同款：浅描边 + 轻阴影
   描边用 --border-soft（主题里的「浅描边」），后台改配色时卡片边会跟着变 */
.feed-article {
  padding: 20px;
  margin-bottom: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  transition: box-shadow 0.2s;
}
.feed-article:hover {
  box-shadow: var(--shadow);
}

.feed-head {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
/* 尺寸由 AvatarFrame 的 size 属性控制（inline width/height），这里只管布局与交互 */
.feed-avatar {
  flex-shrink: 0;
  cursor: pointer;
}
.feed-info {
  flex: 1;
  min-width: 0;
}
.feed-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.feed-name-link {
  cursor: pointer;
}
.feed-name-link:hover {
  color: var(--primary);
}
/* 等级 / 头衔：与 MomentItem 完全一致（scoped 样式不会跨组件生效，这里必须复制一份） */
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

.feed-top {
  padding: 1px 6px;
  font-size: 11px;
  font-weight: 400;
  border-radius: 3px;
  background: var(--accent-soft);
  color: var(--accent);
}
.feed-time {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 2px;
  font-size: 12px;
  color: var(--text-muted);
}
/* 文章分类（分组名）：与「长文」同为胶囊，放在底部同行，过长省略 */
.kind-group {
  padding: 2px 8px;
  max-width: 8em;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 999px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.feed-title {
  margin: 12px 0 8px;
  font-size: 17px;
  font-weight: 600;
  line-height: 1.45;
  color: var(--text);
  cursor: pointer;
}
.feed-title:hover {
  color: var(--primary);
}

.feed-body {
  margin: 0;
  font-size: 13px;
  line-height: 1.8;
  color: var(--text-soft);
  word-break: break-word;
}

/* 查看更多：轻量文字按钮，与正文左对齐 */
.feed-more {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  padding: 0;
  font-size: 12px;
  color: var(--primary);
  background: none;
  border: none;
  cursor: pointer;
}
.feed-more:hover {
  color: var(--primary-deep);
}
.feed-more i {
  font-size: 11px;
}

/* 底部互动栏：点赞 / 评论 / 阅读，图标 + 数字，与动态卡片一致 */
.feed-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  column-gap: 12px;
  row-gap: 8px;
  margin-top: 12px;
  padding-top: 10px;
  /* 虚线属于「更弱的分隔」，取 --border-soft（--line-soft），与动态卡片一致 */
  border-top: 1px dashed var(--border-soft);
  font-size: 13px;
  color: var(--text-muted);
}
/* 左侧：类型 + 分类；右侧：互动。两组各自不换行，整体放不下时按钮组整体落到下一行 */
.action-tags,
.action-btns {
  display: flex;
  align-items: center;
  min-width: 0;
}
.action-tags {
  gap: 6px;
}
.action-btns {
  gap: 18px;
  margin-left: auto;
}
@media (max-width: 480px) {
  .action-btns {
    gap: 14px;
  }
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
.feed-action {
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
.feed-action i {
  font-size: 15px;
}
.feed-action:hover {
  color: var(--primary);
}
/* 只读的阅读数：不要 hover 变色，避免看着像可点 */
.feed-action.is-static {
  cursor: default;
}
.feed-action.is-static:hover {
  color: inherit;
}
/* 点赞飘心的定位容器 */
.like-btn {
  position: relative;
}
.feed-action.is-liked {
  color: var(--accent);
}
</style>
