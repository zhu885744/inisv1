<template>
  <div v-if="article" class="article-detail">
    <h1 class="title">
      <EmojiText :text="article.title" :size="24" />
    </h1>
    <div class="meta">
      <!-- 作者入口：小头像 + 昵称，点击进入作者主页（无 id 时退化为纯文本） -->
      <component
        :is="authorLink ? 'router-link' : 'span'"
        :to="authorLink || undefined"
        class="meta-author"
        :class="{ 'is-link': !!authorLink }"
      >
        <img :src="author?.avatar || defaultAvatar" class="meta-author-avatar" alt="" loading="lazy" />
        <span class="meta-author-name">{{ author?.nickname || '匿名' }}</span>
      </component>
      <span class="dot">·</span>
      <span>{{ formatDate(article.create_time) }}</span>
      <span class="dot">·</span>
      <span>{{ article.views || 0 }} 次阅读</span>
      <template v-if="commentShow">
        <span class="dot">·</span>
        <span>{{ commentsCount }} 评论</span>
      </template>
    </div>

    <!-- 审核状态提示：只有作者本人 / 管理员能看到未审核文章，这里说明可见范围 -->
    <div v-if="auditTip" class="audit-banner" :class="auditTip.type">
      <i :class="auditTip.icon" aria-hidden="true" />
      <span>
        {{ auditTip.text }}
        <!-- 驳回原因：管理员审核未通过时填写 -->
        <template v-if="auditTip.reason">
          <br />驳回原因：{{ auditTip.reason }}
        </template>
      </span>
    </div>

    <div v-if="abstractText" class="abstract">
      <EmojiText :text="abstractText" :size="18" />
    </div>

    <!-- 当前可见范围：作者 / 管理员可直接查看时也把设置值显示出来，避免误以为设置没生效 -->
    <div v-if="scopeNotice" class="scope-notice">
      <i class="bi bi-eye" aria-hidden="true" />
      <span>{{ scopeNotice }}</span>
    </div>

    <!-- 可见范围：登录可见 / 付费可见时后端已清空正文，这里展示引导并支持积分兑换 -->
    <ContentLock
      v-if="articleLocked"
      :state="article.scope_state"
      :integral="Number(article.scope_integral) || 0"
      bind-type="article"
      :bind-id="article.id"
      @unlocked="load"
    />

    <article
      v-else
      ref="contentRef"
      :class="['content', isRichText ? 'rich-text' : 'markdown-body']"
      v-html="contentHtml"
      @click="onContentClick"
    />

    <div v-if="tagList.length" class="article-tags">
      <router-link
        v-for="t in tagList"
        :key="t.id"
        :to="`/topic/${t.id}`"
        class="tag tag-primary"
      >#{{ t.name }}</router-link>
    </div>

    <!-- 操作区：宽屏是横排按钮，窄屏收成等宽的「图标在上、文字在下」操作栏 -->
    <div class="article-actions">
      <button ref="likeBtn" class="btn btn-sm action-btn" :class="{ 'is-on': liked }" @click="toggleLike">
        <i ref="likeIcon" class="bi" :class="liked ? 'bi-star-fill' : 'bi-star'" />
        <span class="action-text">点赞<em v-if="likeCount" class="action-count">{{ likeCount }}</em></span>
      </button>
      <button class="btn btn-sm action-btn" :class="{ 'is-on': collected }" @click="toggleCollect">
        <i ref="collectIcon" class="bi" :class="collected ? 'bi-bookmark-fill' : 'bi-bookmark'" />
        <span class="action-text">{{ collected ? '已收藏' : '收藏' }}<em v-if="collectCount" class="action-count">{{ collectCount }}</em></span>
      </button>
      <button v-if="canReward" class="btn btn-sm action-btn" @click="openReward">
        <i class="bi bi-coin" />
        <span class="action-text">打赏</span>
      </button>
      <button class="btn btn-sm action-btn" @click="shareArticle">
        <i class="bi bi-share" />
        <span class="action-text">分享</span>
      </button>
      <!-- 举报：自己的文章不显示；提交后只通知管理员，作者不会收到提示 -->
      <button v-if="canReport" class="btn btn-sm action-btn" title="举报这篇文章" @click="reportVisible = true">
        <i class="bi bi-flag" />
        <span class="action-text">举报</span>
      </button>
    </div>

    <ReportDialog
      :visible="reportVisible"
      target-type="article"
      :target-id="route.params.id"
      :target-label="`文章「${(article?.title || '').slice(0, 20)}」`"
      :loading="reporting"
      @update:visible="(v) => (reportVisible = v)"
      @confirm="submitReport"
    />

    <!-- 文末：上一篇 / 下一篇（按发布顺序切换，只统计已发布且审核通过的文章） -->
    <nav v-if="prevArticle || nextArticle" class="article-nav" aria-label="文章导航">
      <component
        :is="prevArticle ? 'router-link' : 'span'"
        :to="prevArticle ? `/archives/${prevArticle.id}` : undefined"
        class="article-nav-item article-nav-item-prev"
        :class="{ 'is-disabled': !prevArticle }"
      >
        <span class="article-nav-label">
          <i class="bi bi-arrow-left" aria-hidden="true" />
          上一篇
        </span>
        <span class="article-nav-title">{{ prevArticle?.title || '已经是第一篇了' }}</span>
      </component>

      <component
        :is="nextArticle ? 'router-link' : 'span'"
        :to="nextArticle ? `/archives/${nextArticle.id}` : undefined"
        class="article-nav-item article-nav-item-next"
        :class="{ 'is-disabled': !nextArticle }"
      >
        <span class="article-nav-label">
          下一篇
          <i class="bi bi-arrow-right" aria-hidden="true" />
        </span>
        <span class="article-nav-title">{{ nextArticle?.title || '已经是最后一篇了' }}</span>
      </component>
    </nav>

    <!-- 评论（json.comment.show：1 显示 / 2 不显示；allow：1 允许 / 2 禁止，禁止时仅隐藏输入框） -->
    <template v-if="commentShow">
      <SectionTitle title="评论">
        <template #extra>
          <span v-if="commentAllow" class="text-muted">{{ commentsCount }} 条评论</span>
          <span v-else class="text-muted">评论已关闭</span>
        </template>
      </SectionTitle>
      <CommentTree
        :bind-id="article.id"
        bind-type="article"
        :author-id="article.result?.author?.id"
        :allow-comment="commentAllow"
        :highlight-id="route.query.comment"
        @loaded="(n) => commentsCount = n"
      />
    </template>
  </div>

  <EmptyState v-else-if="!loading" text="文章不存在或已被删除" />
  <div v-else-if="loading" class="is-loading">
    <span class="spinner" /> 加载中...
  </div>

  <!-- 打赏弹窗：积分打赏（作者自己看自己的文章不显示入口） -->
  <RewardDialog
    v-model:visible="rewardVisible"
    bind-type="article"
    :bind-id="route.params.id"
    :to-uid="authorUid"
    :to-name="author?.nickname || ''"
  />
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import SectionTitle from '@/components/SectionTitle.vue'
import CommentTree from '@/components/CommentTree.vue'
import ReportDialog from '@/components/ReportDialog.vue'
import RewardDialog from '@/components/RewardDialog.vue'
import { createReport } from '@/api/report'
import EmptyState from '@/components/EmptyState.vue'
import EmojiText from '@/components/EmojiText.vue'
import ContentLock from '@/components/ContentLock.vue'
import WearingBadge from '@/components/WearingBadge.vue'
import { getArticle } from '@/api/article'
import {
  like, unlike, isLiked,
  collect, uncollect, isCollected
} from '@/api/tags'
import { useUserStore } from '@/stores/user'
import { useRouter } from 'vue-router'
import { formatDate } from '@/utils/time'
import { renderMarkdown } from '@/utils/markdown'
import { sanitizeHtml, isRichTextEditor } from '@/utils/html'
import { toast } from '@/utils/toast'
import { copyText, stripArticleMarkup } from '@/utils/helper'
import { openLightbox } from '@/utils/lightbox'
import { popIcon, popOut, burstHeart } from '@/utils/likeFx'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/></svg>'

const article = ref(null)
const loading = ref(false)
const liked = ref(false)
const collected = ref(false)
const likeCount = ref(0)
const collectCount = ref(0)
const commentsCount = ref(0)

// 评论开关：取自文章 json.comment（后端已按全局 ARTICLE 配置做继承）
// show: 1 显示 / 2 不显示（整块隐藏）    allow: 1 允许 / 2 禁止（仅隐藏输入发布模块，列表照常展示）
const commentAllow = computed(() => Number(article.value?.result?.comment?.allow ?? 1) !== 2)
const commentShow = computed(() => Number(article.value?.result?.comment?.show ?? 1) !== 2)

// 审核状态提示（草稿 / 待审核 / 未通过；普通访客拿不到这类文章，只有作者与管理员可见）
const auditTip = computed(() => {
  const item = article.value
  if (!item) return null
  if (Number(item.status) === 0) {
    return { type: 'is-draft', icon: 'bi bi-pencil-square', text: '草稿，仅自己可见' }
  }
  const audit = Number(item.audit)
  if (audit === 0) {
    return { type: 'is-audit', icon: 'bi bi-clock-history', text: '待审核，通过后其他访客才能看到' }
  }
  if (audit === 2) {
    return {
      type: 'is-reject',
      icon: 'bi bi-exclamation-triangle',
      text: '未通过审核，仅自己（与管理员）可见',
      reason: String(item.reason || '').trim()
    }
  }
  return null
})

// 打赏对象：文章作者（自己的文章不显示入口，后端同样会拒绝自打赏）
const rewardVisible = ref(false)
const authorUid = computed(() => Number(article.value?.uid || article.value?.result?.author?.id || 0))
const canReward = computed(() => authorUid.value > 0 && Number(userStore.user?.id) !== authorUid.value)

function openReward() {
  rewardVisible.value = true
}

const isRichText = computed(() => isRichTextEditor(article.value?.editor))

// 摘要展示前清掉 markdown / 媒体短代码：历史数据的 abstract 可能存着
// @[live](视频 "封面") 这类源码（旧版自动摘要没处理 live）
const abstractText = computed(() => stripArticleMarkup(article.value?.abstract || ''))

// 可见范围：scope_state 为 login / pay 时说明后端已按可见范围清空正文
const articleLocked = computed(() => {
  const state = article.value?.scope_state
  return state === 'login' || state === 'pay'
})

// 当前可见范围：**只给未登录访客看**。
// 登录用户要么直接能看（公开 / 作者 / 管理员 / 已兑换），要么 ContentLock 已经给出兑换引导，
// 再挂一条范围提示只会重复。
const scopeNotice = computed(() => {
  if (userStore.isLogged) return ''
  const scope = Number(article.value?.scope)
  if (scope === 2) return '当前可见范围：登录可见'
  if (scope === 3) {
    const integral = Number(article.value?.scope_integral) || 0
    return `当前可见范围：付费可见 · ${integral} 积分`
  }
  return ''
})

// 富文本：content 本身是 HTML，清洗后原样渲染；markdown（vditor 等）：走 markdown 渲染
const contentHtml = computed(() => {
  const content = article.value?.content || ''
  if (!content) return ''
  return isRichText.value ? sanitizeHtml(content) : renderMarkdown(content)
})
const contentRef = ref(null)
const likeBtn = ref(null)
const likeIcon = ref(null)

// 点击文章内容中的图片时打开灯箱（事件委托，兼容 v-html 渲染的图片）
function onContentClick(e) {
  const img = e.target.closest('img')
  if (!img || !img.src) return
  const srcs = Array.from(contentRef.value?.querySelectorAll('img') || [])
    .map((el) => el.src)
    .filter(Boolean)
  openLightbox(srcs, srcs.indexOf(img.src))
}

const author = computed(() => article.value?.result?.author || null)
// 作者主页链接（无 id 时标题区退化为不可点击的纯文本）
const authorLink = computed(() => {
  const id = author.value?.id
  return id ? `/author/${id}` : ''
})
// 上一篇 / 下一篇：后端在详情里按发布顺序给出相邻文章（只含已发布且审核通过的）
const prevArticle = computed(() => article.value?.prev || null)
const nextArticle = computed(() => article.value?.next || null)
const tagList = computed(() => {
  const t = article.value?.result?.tags
  if (Array.isArray(t)) return t
  return []
})

async function load() {
  loading.value = true
  try {
    const res = await getArticle(route.params.id)
    article.value = res.data

    // 评论数直接取详情返回里的 result.comment.count
    // （comment/count 不支持 bind_id/bind_type 过滤，会返回全站评论总数）
    commentsCount.value = res.data?.result?.comment?.count || 0
  } catch {
    article.value = null
  } finally {
    loading.value = false
  }

  // 点赞 / 收藏状态
  try {
    if (userStore.isLogged) {
      const [l, c] = await Promise.all([
        isLiked('article', route.params.id).catch(() => null),
        isCollected('article', route.params.id).catch(() => null)
      ])
      // is-liked 返回 { data: { is_liked, count } }
      liked.value = !!(l?.data?.is_liked ?? l?.data)
      collected.value = !!(c?.data?.is_collected ?? c?.data)
      // 已登录：优先用接口返回的精确计数，缺失时回退到文章详情自带计数
      likeCount.value = l?.data?.count ?? article.value?.result?.like?.length ?? article.value?.likes ?? 0
      collectCount.value = c?.data?.count ?? article.value?.result?.collect?.length ?? article.value?.favorites ?? 0
    } else {
      // 未登录：直接使用文章详情接口返回的计数（参考 archives.vue 做法，
      // 不调用 user-likes/counts、user-collects/counts 批量接口，避免 target_ids 报错）
      likeCount.value = article.value?.result?.like?.length ?? article.value?.likes ?? 0
      collectCount.value = article.value?.result?.collect?.length ?? article.value?.favorites ?? 0
    }
  } catch {}
}

function requireLogin() {
  if (!userStore.isLogged) {
    toast.warning('请先登录')
    router.push({ name: 'login', query: { redirect: route.fullPath } })
    return false
  }
  return true
}

async function toggleLike() {
  if (!requireLogin()) return
  if (liked.value) {
    // 取消点赞：图标收缩动画 + 乐观提示
    popOut(likeIcon.value)
    try {
      await unlike('article', route.params.id)
      liked.value = false
      likeCount.value = Math.max(0, likeCount.value - 1)
      toast.info('已取消点赞')
    } catch {}
  } else {
    // 点赞：立即弹跳 + 飘心
    popIcon(likeIcon.value)
    burstHeart(likeBtn.value)
    try {
      await like('article', route.params.id)
      liked.value = true
      likeCount.value += 1
      toast.success('点赞成功')
    } catch {}
  }
}

async function toggleCollect() {
  if (!requireLogin()) return
  try {
    if (collected.value) {
      await uncollect('article', route.params.id)
      collected.value = false
      collectCount.value = Math.max(0, collectCount.value - 1)
      toast.info('已取消收藏')
    } else {
      await collect('article', route.params.id)
      collected.value = true
      collectCount.value += 1
      toast.success('收藏成功')
    }
  } catch {}
}

// 分享：复制「文章标题 + 当前链接」
async function shareArticle() {
  if (!article.value) return
  const text = `【${article.value.title}】${window.location.href}`
  const ok = await copyText(text)
  if (ok) {
    toast.success('已复制链接和标题，去分享吧')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

// ---------- 举报 ----------
// 未登录不显示入口；自己的文章也不显示（后端同样会拒「举报自己」）
const reportVisible = ref(false)
const reporting = ref(false)
const canReport = computed(() => {
  const me = userStore.user?.id
  if (!me) return false
  return String(article.value?.uid || '') !== String(me)
})

async function submitReport({ reason, detail, images }) {
  reporting.value = true
  try {
    await createReport({ target_type: 'article', target_id: route.params.id, reason, detail, images })
    reportVisible.value = false
    toast.success('举报已提交，我们会尽快处理')
  } catch {
    /* 拦截器已提示（重复举报 / 超过每日上限等） */
  } finally {
    reporting.value = false
  }
}

watch(() => route.params.id, load)
onMounted(() => {
  load()
})
onUnmounted(() => {
  document.body.style.overflow = ''
})
</script>

<style scoped>
/* 与动态详情页对齐：components/MomentItem.vue 的 .moment-card 是
   「背景 + 描边 + 大圆角 + 20px 内边距」的一张卡片，文章详情整块也照这个来，
   两边的阅读体验才一致（评论区在卡片内，和动态展开评论后的形态相同）。
   不加 hover 阴影：详情页不是列表项，鼠标一直在卡片上会让整块一直泛着阴影。 */
.article-detail {
  padding: 20px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  margin-bottom: 16px;
}
.title {
  font-family: var(--font-serif);
  font-size: 26px;
  font-weight: 600;
  margin-bottom: 12px;
  line-height: 1.4;
}
.meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 16px;
}
.meta .dot {
  color: var(--border);
}
/* 作者入口：小头像 + 昵称 */
.meta-author {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: inherit;
  text-decoration: none;
  transition: color 0.2s;
}
.meta-author.is-link {
  cursor: pointer;
}
.meta-author.is-link:hover {
  color: var(--primary);
}
.meta-author-avatar {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.meta-author-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 12em;
}
.abstract {
  padding: 16px 20px;
  background: var(--bg-muted);
  border-left: 3px solid var(--primary);
  font-size: 13px;
  color: var(--text-soft);
  border-radius: 0 var(--radius) var(--radius) 0;
  margin-bottom: 24px;
}
/* 当前可见范围提示（仅作者 / 管理员可直接查看时出现） */
.scope-notice {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
  padding: 6px 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius-sm);
}
/* 审核状态提示条（草稿 / 待审核 / 未通过） */
.audit-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 16px;
  padding: 10px 14px;
  font-size: 13px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
}
.audit-banner.is-draft {
  background: var(--bg-muted);
  color: var(--text-muted);
}
.audit-banner.is-audit {
  background: var(--gold-wash);
  border-color: var(--gold-line);
  color: var(--warning);
}
.audit-banner.is-reject {
  background: var(--accent-soft);
  border-color: var(--primary-soft);
  color: var(--danger);
}
/* .content 基础排版由全局 .markdown-body 提供 */

.article-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 24px 0 16px;
}
.article-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  justify-content: center;
  padding: 16px 0;
  border-top: 1px dashed var(--border-soft);
  border-bottom: 1px dashed var(--border-soft);
}
/* 计数跟在文字后面，比原来的「点赞 (3)」省地方，窄屏也不容易换行 */
.action-text {
  display: inline-flex;
  align-items: baseline;
  gap: 3px;
  white-space: nowrap;
}
.action-count {
  font-style: normal;
  font-size: 11px;
  color: var(--text-muted);
}
/* 点赞飘心的定位容器（burstHeart 会把飘心贴在按钮上） */
.action-btn {
  position: relative;
}
/* 已点赞 / 已收藏：整颗按钮变色，不再是「图标黄了、文字还是灰的」 */
.action-btn.is-on {
  color: var(--primary);
  border-color: var(--primary-soft);
}
.action-btn.is-on .action-count {
  color: inherit;
  opacity: 0.75;
}
/* 文末：上一篇 / 下一篇 */
.article-nav {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin: 28px 0 8px;
}
.article-nav-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 14px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: inherit;
  text-decoration: none;
  -webkit-tap-highlight-color: transparent;
  transition: border-color 0.2s, background-color 0.2s, transform 0.2s;
}
.article-nav-item-next {
  text-align: right;
}
a.article-nav-item:hover {
  border-color: var(--primary-soft);
  background: var(--bg-muted);
  /* 覆盖全局 a:hover 的链接色，标题颜色单独控制 */
  color: var(--text);
}
a.article-nav-item:hover .article-nav-title {
  color: var(--primary);
}
a.article-nav-item:active {
  transform: translateY(1px);
}
/* 到头了：保留位置与占位文案，避免整块突然只剩一边 */
.article-nav-item.is-disabled {
  opacity: 0.55;
  cursor: default;
}
.article-nav-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-light);
}
.article-nav-item-next .article-nav-label {
  justify-content: flex-end;
}
.article-nav-title {
  font-size: 14px;
  line-height: 1.5;
  color: var(--text);
  /* 最多两行：长标题不会把左右两块撑得高低不齐 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  transition: color 0.2s;
}
.is-loading {
  padding: 64px;
  text-align: center;
  color: var(--text-muted);
}

/* 窄屏：上一篇 / 下一篇改为上下堆叠，右对齐的那块也回归左对齐 */
@media (max-width: 640px) {
  .article-nav {
    grid-template-columns: 1fr;
    gap: 10px;
    margin: 22px 0 8px;
  }
  .article-nav-item-next {
    text-align: left;
  }
  .article-nav-item-next .article-nav-label {
    justify-content: flex-start;
  }

  /* 操作区：窄屏放不下横排按钮（5 个按钮 + 计数会挤成两行），
     改成等宽操作栏 —— 图标在上、文字在下，去按钮框，靠留白分隔 */
  .article-actions {
    gap: 0;
    padding: 4px 0;
    border-bottom: 0;
  }
  .article-actions .action-btn {
    flex: 1 1 0;
    min-width: 0;
    height: auto;
    flex-direction: column;
    gap: 3px;
    padding: 8px 2px;
    font-size: 11px;
    line-height: 1.2;
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-muted);
  }
  .article-actions .action-btn i {
    font-size: 19px;
    line-height: 1;
  }
  .article-actions .action-text {
    justify-content: center;
    font-size: 11px;
  }
  .article-actions .action-count {
    font-size: 10px;
  }
  /* 触摸反馈代替 hover：手机上 hover 会「粘住」，点一下整排按钮都像被选中 */
  .article-actions .action-btn:hover {
    color: var(--text-muted);
    background: transparent;
  }
  .article-actions .action-btn:active {
    background: var(--bg-muted);
  }
  /* 覆盖上面的中性色，已点赞 / 已收藏仍然高亮 */
  .article-actions .action-btn.is-on {
    color: var(--primary);
  }
}

/* Markdown 排版样式统一使用全局 styles.css 中的 .markdown-body */
</style>