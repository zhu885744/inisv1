<template>
  <div v-if="page" class="page-detail">
    <h1 class="title">
      <EmojiText :text="page.title" :size="24" />
    </h1>
    <div class="meta">
      <span>{{ formatDate(page.publish_time || page.create_time) }}</span>
      <span class="dot">·</span>
      <span>{{ page.views || 0 }} 次阅读</span>
      <template v-if="commentShow">
        <span class="dot">·</span>
        <span>{{ commentsCount }} 评论</span>
      </template>
    </div>

    <!-- 审核状态提示：只有作者本人 / 管理员能看到未审核页面，这里说明可见范围 -->
    <div v-if="auditTip" class="audit-banner" :class="auditTip.type">
      <i :class="auditTip.icon" aria-hidden="true" />
      <span>{{ auditTip.text }}</span>
    </div>

    <!-- 当前可见范围：作者 / 管理员可直接查看时也把设置值显示出来，避免误以为设置没生效 -->
    <div v-if="scopeNotice" class="scope-notice">
      <i class="bi bi-eye" aria-hidden="true" />
      <span>{{ scopeNotice }}</span>
    </div>

    <!-- 可见范围：登录可见 / 付费可见时后端已清空正文，这里展示引导并支持积分兑换 -->
    <ContentLock
      v-if="pageLocked"
      :state="page.scope_state"
      :integral="Number(page.scope_integral) || 0"
      bind-type="page"
      :bind-id="page.id"
      @unlocked="load"
    />

    <article
      v-else
      ref="contentRef"
      class="content markdown-body"
      v-html="contentHtml"
      @click="onContentClick"
    />

    <div v-if="tagList.length" class="page-tags">
      <router-link
        v-for="t in tagList"
        :key="t.id"
        :to="`/topic/${t.id}`"
        class="tag tag-primary"
      >#{{ t.name }}</router-link>
    </div>

    <div class="page-actions">
      <button ref="likeBtn" class="btn btn-sm like-btn" @click="toggleLike">
        <i ref="likeIcon" class="bi" :class="liked ? 'bi-star-fill' : 'bi-star'" /> 点赞 <span v-if="likeCount">({{ likeCount }})</span>
      </button>
      <button class="btn btn-sm" @click="toggleCollect">
        <i class="bi" :class="collected ? 'bi-bookmark-fill' : 'bi-bookmark'" /> {{ collected ? '已收藏' : '收藏' }}
        <span v-if="collectCount">({{ collectCount }})</span>
      </button>
      <button class="btn btn-sm like-btn" @click="sharePage">
        <i class="bi bi-share" /> 分享
      </button>
    </div>

    <!-- 作者（点击进入作者主页） -->
    <component
      :is="authorLink ? 'router-link' : 'div'"
      v-if="author"
      :to="authorLink || undefined"
      class="author-card card card-pad"
      :class="{ 'is-link': !!authorLink }"
    >
      <img :src="author.avatar || defaultAvatar" class="author-avatar" loading="lazy" />
      <div class="author-info">
        <div class="author-name">{{ author.nickname }}</div>
        <div class="author-desc">{{ author.description || '' }}</div>
      </div>
      <i v-if="authorLink" class="bi bi-chevron-right author-arrow" />
    </component>

    <!-- 评论（json.comment：show 1 显示 / 2 隐藏；allow 1 允许 / 2 禁止，禁止时仅隐藏输入框） -->
    <template v-if="commentShow">
      <SectionTitle title="评论">
        <template #extra>
          <span v-if="commentAllow" class="text-muted">{{ commentsCount }} 条评论</span>
          <span v-else class="text-muted">评论已关闭</span>
        </template>
      </SectionTitle>
      <CommentTree
        :bind-id="page.id"
        bind-type="page"
        :author-id="page.uid"
        :allow-comment="commentAllow"
        :highlight-id="route.query.comment"
        @loaded="(n) => commentsCount = n"
      />
    </template>
  </div>

  <EmptyState v-else-if="!loading" text="页面不存在" />
  <div v-else-if="loading" class="is-loading">
    <span class="spinner" /> 加载中...
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SectionTitle from '@/components/SectionTitle.vue'
import CommentTree from '@/components/CommentTree.vue'
import EmptyState from '@/components/EmptyState.vue'
import EmojiText from '@/components/EmojiText.vue'
import ContentLock from '@/components/ContentLock.vue'
import { call } from '@/api/request'
import { getUser } from '@/api/users'
import {
  like, unlike, isLiked, likesCount,
  collect, uncollect, isCollected, collectsCount
} from '@/api/tags'
import { useUserStore } from '@/stores/user'
import { formatDate } from '@/utils/time'
import { renderMarkdown } from '@/utils/markdown'
import { toast } from '@/utils/toast'
import { copyText } from '@/utils/helper'
import { openLightbox } from '@/utils/lightbox'
import { popIcon, popOut, burstHeart } from '@/utils/likeFx'

const props = defineProps({
  pageKey: { type: String, default: '' },
  id: { type: [String, Number], default: '' }
})

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/></svg>'

const page = ref(null)
const loading = ref(false)
const author = ref(null)
const liked = ref(false)
const collected = ref(false)
const likeCount = ref(0)
const collectCount = ref(0)
const commentsCount = ref(0)

const contentRef = ref(null)
const likeBtn = ref(null)
const likeIcon = ref(null)

const contentHtml = computed(() => renderMarkdown(page.value?.content || ''))

// 可见范围：scope_state 为 login / pay 时说明后端已按可见范围清空正文
const pageLocked = computed(() => {
  const state = page.value?.scope_state
  return state === 'login' || state === 'pay'
})

// 当前可见范围：**只给未登录访客看**（登录用户已有内容或兑换引导，不再重复提示）
const scopeNotice = computed(() => {
  if (userStore.isLogged) return ''
  const scope = Number(page.value?.scope)
  if (scope === 2) return '当前可见范围：登录可见'
  if (scope === 3) {
    const integral = Number(page.value?.scope_integral) || 0
    return `当前可见范围：付费可见 · ${integral} 积分`
  }
  return ''
})

// 评论开关：取自页面 result.comment（后端已按全局 PAGE 配置做覆盖与继承）
// show：1 显示 / 2 隐藏（整块隐藏）    allow：1 允许 / 2 禁止（仅隐藏输入发布模块，列表照常展示）
const commentAllow = computed(() => Number(page.value?.result?.comment?.allow ?? 1) !== 2)
const commentShow = computed(() => Number(page.value?.result?.comment?.show ?? 1) !== 2)

// 审核状态提示（独立页面没有草稿态，只有待审核 / 未通过）
const auditTip = computed(() => {
  const item = page.value
  if (!item) return null
  const audit = Number(item.audit)
  if (audit === 0) {
    return { type: 'is-audit', icon: 'bi bi-clock-history', text: '待审核，通过后其他访客才能看到' }
  }
  if (audit === 2) {
    return { type: 'is-reject', icon: 'bi bi-exclamation-triangle', text: '未通过审核，仅自己（与管理员）可见' }
  }
  return null
})

const tagList = computed(() => {
  const t = page.value?.result?.tags
  return Array.isArray(t) ? t : []
})

const authorLink = computed(() => {
  const id = author.value?.id
  return id ? `/author/${id}` : ''
})

// 点击页面内容中的图片时打开灯箱（事件委托，兼容 v-html 渲染的图片）
function onContentClick(e) {
  const img = e.target.closest('img')
  if (!img || !img.src) return
  const srcs = Array.from(contentRef.value?.querySelectorAll('img') || [])
    .map((el) => el.src)
    .filter(Boolean)
  openLightbox(srcs, srcs.indexOf(img.src))
}

// 独立页面的 result 里没有 author，按 uid 单独拉取
async function loadAuthor() {
  const uid = Number(page.value?.uid)
  if (!uid) {
    author.value = null
    return
  }
  try {
    const res = await getUser(uid, 'id,nickname,avatar,description')
    author.value = res?.data || null
  } catch {
    author.value = null
  }
}

// 点赞 / 收藏状态与计数（page 类型同样走 user-likes / user-collects）
async function loadInteractions() {
  const id = page.value?.id
  if (!id) return
  try {
    if (userStore.isLogged) {
      const [l, c] = await Promise.all([
        isLiked('page', id).catch(() => null),
        isCollected('page', id).catch(() => null)
      ])
      liked.value = !!(l?.data?.is_liked ?? l?.data)
      collected.value = !!(c?.data?.is_collected ?? c?.data)
      likeCount.value = l?.data?.count ?? 0
      collectCount.value = c?.data?.count ?? 0
    } else {
      const [l, c] = await Promise.all([
        likesCount('page', [id]).catch(() => null),
        collectsCount('page', [id]).catch(() => null)
      ])
      const likeCounts = l?.data?.counts || l?.data?.data?.counts || {}
      const collectCounts = c?.data?.counts || c?.data?.data?.counts || {}
      likeCount.value = likeCounts[id] || 0
      collectCount.value = collectCounts[id] || 0
    }
  } catch {}
}

async function load() {
  loading.value = true
  page.value = null
  author.value = null
  liked.value = false
  collected.value = false
  likeCount.value = 0
  collectCount.value = 0
  commentsCount.value = 0

  // 两种入口：/:key（分享 / 导航用 slug）与 /page/:id（作者页 / 评论通知 / 打赏通知用 id）
  // pages/one 同时支持 id、key 查询，但参数名不同，这里按入口分别传
  const pageId = Number(props.id || route.params.id) || 0
  const key = props.pageKey || route.params.key || props.id || route.params.id
  try {
    const res = await call('pages', 'one', {
      method: 'GET',
      params: pageId > 0 ? { id: pageId } : { key }
    })
    page.value = res.data
    // 评论数直接取详情返回里的 result.comment.count（避免等评论列表加载完才显示）
    commentsCount.value = res.data?.result?.comment?.count || 0
  } catch {
    page.value = null
  } finally {
    loading.value = false
  }

  if (!page.value) return
  loadAuthor()
  loadInteractions()
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
  const id = page.value?.id
  if (liked.value) {
    popOut(likeIcon.value)
    try {
      await unlike('page', id)
      liked.value = false
      likeCount.value = Math.max(0, likeCount.value - 1)
      toast.info('已取消点赞')
    } catch {}
  } else {
    popIcon(likeIcon.value)
    burstHeart(likeBtn.value)
    try {
      await like('page', id)
      liked.value = true
      likeCount.value += 1
      toast.success('点赞成功')
    } catch {}
  }
}

async function toggleCollect() {
  if (!requireLogin()) return
  const id = page.value?.id
  try {
    if (collected.value) {
      await uncollect('page', id)
      collected.value = false
      collectCount.value = Math.max(0, collectCount.value - 1)
      toast.info('已取消收藏')
    } else {
      await collect('page', id)
      collected.value = true
      collectCount.value += 1
      toast.success('收藏成功')
    }
  } catch {}
}

// 分享：复制「页面标题 + 当前链接」
async function sharePage() {
  if (!page.value) return
  const text = `【${page.value.title}】${window.location.href}`
  const ok = await copyText(text)
  if (ok) {
    toast.success('已复制链接和标题，去分享吧')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

watch(() => [props.pageKey, route.params.key, route.params.id], load)
onMounted(load)
</script>

<style scoped>
.page-detail {
  padding: 8px 0;
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
/* 审核状态提示条（待审核 / 未通过） */
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

.page-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 24px 0 16px;
}
.page-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  padding: 16px 0;
  border-top: 1px dashed var(--border-soft);
  border-bottom: 1px dashed var(--border-soft);
}
.like-btn {
  position: relative;
}
.author-card {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 24px 0;
  color: inherit;
  text-decoration: none;
}
/* 可跳转时给出可点击反馈 */
.author-card.is-link {
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s, transform 0.2s;
  -webkit-tap-highlight-color: transparent;
}
.author-card.is-link:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow-sm);
  /* 覆盖全局 a:hover 的链接色，避免主题色渗透到描述文字 */
  color: var(--text);
}
.author-card.is-link:hover .author-name {
  color: var(--primary);
}
.author-card.is-link:active {
  transform: translateY(1px);
}
.author-card.is-link:focus-visible {
  outline: 2px solid var(--primary-soft);
  outline-offset: 2px;
}
.author-arrow {
  flex-shrink: 0;
  color: var(--text-light);
  transition: transform 0.2s, color 0.2s;
}
.author-card.is-link:hover .author-arrow {
  color: var(--primary);
  transform: translateX(2px);
}
.author-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.author-info {
  flex: 1;
  min-width: 0;
}
.author-name {
  font-size: 15px;
  font-weight: 500;
  margin-bottom: 4px;
  transition: color 0.2s;
}
.author-desc {
  font-size: 12px;
  color: var(--text-muted);
}
.is-loading {
  padding: 64px;
  text-align: center;
  color: var(--text-muted);
}

/* Markdown 排版样式统一使用全局 styles.css 中的 .markdown-body */
</style>
