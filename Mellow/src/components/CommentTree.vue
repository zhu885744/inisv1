<template>
  <div class="comment-tree">
    <!-- 发表评论（根评论）；allowComment = false 时只隐藏输入发布模块，评论列表照常展示 -->
    <div v-if="allowComment" class="root-comment-box">
      <img v-if="userStore.isLogged" class="root-avatar" :src="userStore.user?.avatar || defaultAvatar" alt="" @click="goMe" loading="lazy" />
      <div class="root-input">
        <div v-if="userStore.isLogged" class="root-meta">
          <span class="root-name" @click="goMe">{{ userStore.user.nickname || '我' }}</span>
          <span v-if="myLevel" class="level-tag">Lv.{{ myLevel }}</span>
          <!-- 有装扮样式用 TitleBadge 还原后台配色；没有则回退旧色板 -->
          <TitleBadge
            v-if="myTitle || myTitlePayload"
            :payload="myTitlePayload || {}"
            :text="myTitle"
            size="sm"
          />
          <span v-else-if="myTitle" :class="['title-tag', myTitleClass]">{{ myTitle }}</span>
        </div>
        <template v-if="userStore.isLogged">
          <EmojiEditor
            v-model="newComment"
            placeholder="写下你的评论…"
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
          <!-- 已选图片预览（实况图只显示封面 + LIVE 角标） -->
          <div v-if="newImages.length" class="img-preview">
            <div v-for="(img, i) in newImages" :key="img.image || i" class="img-thumb">
              <img :src="img.image" alt="" loading="lazy" />
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
          <!-- 评论可见范围：公开 / 登录可见 / 仅作者可见（评论不提供付费可见），用单选框平铺 -->
          <div class="root-scope">
            <ScopeRadio v-model="scope" :options="scopeOptions" />
          </div>
          <div class="root-actions">
            <span v-if="newImages.length" class="img-tip">已选 {{ newImages.length }} 张图片</span>
            <button
              class="btn btn-primary btn-sm btn-block"
              :disabled="(!newComment.trim() && !newImages.length) || uploading"
              @click="submitRoot"
            >
              发表评论
            </button>
          </div>
        </template>
        <div v-else class="login-tip">
          <span class="login-tip-text">登录后可参与回复讨论。</span>
          <div class="login-tip-actions">
            <RouterLink :to="{ name: 'login', query: { redirect: $route.fullPath } }" class="btn btn-sm btn-primary">登录</RouterLink>
            <RouterLink :to="{ name: 'register' }" class="btn btn-sm">注册</RouterLink>
          </div>
        </div>
      </div>
    </div>

    <!-- 评论已关闭提示（仍展示已有评论） -->
    <div v-else class="comment-closed">
      <i class="bi bi-chat-square" aria-hidden="true" />
      该内容的评论功能已关闭
    </div>

    <!-- 附件库（选择已上传图片加入待发表列表） -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择评论图片"
      accept="image"
      multiple
      :max="9"
      @select="onLibrarySelect"
    />

    <!-- 待审核占位：审核开启时刚提交的评论不会立刻出现在列表里（后端返回 audit=0） -->
    <div v-if="pendingComments.length" class="pending-list">
      <div v-for="item in pendingComments" :key="item.key" class="pending-item">
        <span class="pending-badge"><i class="bi bi-hourglass-split" /> 待审核</span>
        <span class="pending-content">
          {{ item.content || '（图片评论）' }}<em v-if="item.hasImages"> · 含图片</em>
        </span>
        <span class="pending-tip">审核通过后其他访客可见</span>
        <button class="pending-del" type="button" title="不再提示这条" @click="removePending(item.key)">
          <i class="bi bi-x-lg" />
        </button>
      </div>
    </div>

    <div v-if="loading" class="comment-loading">
      <span class="spinner" /> 加载中...
    </div>
    <EmptyState v-else-if="!tree.length" icon="bi bi-chat-dots" text="还没有评论，快来抢沙发~" />
    <template v-else>
      <CommentItem
        v-for="c in tree"
        :key="c.id"
        :comment="c"
        :bind-type="bindType"
        :author-id="authorId"
        :allow-comment="allowComment"
        :reply-to="replyTo"
        :highlight-id="highlightId"
        @like="onLike"
        @reply="onReply"
        @submit="onSubmit"
        @remove="onRemove"
        @report="onReport"
        @tip="onTip"
        @unlocked="load"
      />
      <Pagination
        v-if="total > pageSize"
        :current="page"
        :total="total"
        :page-size="pageSize"
        @update:current="(p) => { page = p; load() }"
      />
    </template>

    <!-- 举报评论：提交后只通知管理员，被举报人不会收到提示 -->
    <ReportDialog
      :visible="reportVisible"
      target-type="comment"
      :target-id="reportTarget?.id || 0"
      :target-label="reportLabel"
      :loading="reporting"
      @update:visible="(v) => (reportVisible = v)"
      @confirm="submitReport"
    />

    <!-- 打赏评论：整棵评论树共用一个弹窗（同举报） -->
    <RewardDialog
      v-model:visible="tipVisible"
      bind-type="comment"
      :bind-id="tipTarget?.id || 0"
      :to-uid="tipAuthorId"
      :to-name="tipAuthorName"
    />
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import CommentItem from './CommentItem.vue'
import Pagination from './Pagination.vue'
import EmptyState from './EmptyState.vue'
import EmojiEditor from './EmojiEditor.vue'
import AttachmentLibrary from './AttachmentLibrary.vue'
import {
  getCommentTree,
  createComment,
  removeComment,
  uploadCommentImages
} from '@/api/comment'
import { likesCount, likeStates, isLiked, like, unlike } from '@/api/tags'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import {
  pickCommentAuthor,
  pickUserLevel,
  pickUserTitle,
  pickUserTitlePayload,
  getTitleColorClass
} from '@/utils/helper'
import TitleBadge from './TitleBadge.vue'
import ReportDialog from './ReportDialog.vue'
import RewardDialog from './RewardDialog.vue'
import ScopeRadio from './ScopeRadio.vue'
import { createReport } from '@/api/report'
import { serializeLiveItem } from '@/utils/livePhoto'

const props = defineProps({
  bindId: { type: [String, Number], required: true },
  bindType: { type: String, default: 'article' },
  authorId: { type: [String, Number], default: null },
  highlightId: { type: [String, Number], default: null },
  // 是否允许评论：false 时隐藏「发表评论 / 回复」输入模块，评论内容仍展示
  allowComment: { type: Boolean, default: true }
})

// 通知父组件真实评论总数（发表/删除后同步）
const emit = defineEmits(['loaded'])

const userStore = useUserStore()
const router = useRouter()
const route = useRoute()

// 评论的可见范围（后端 scope）：1 公开 / 2 登录可见 / 4 仅作者可见
// 评论不提供「付费可见」——后端会把 3 降级为 2，且不接受评论的积分兑换
// 「仅作者可见」= 只有被评论内容的作者（以及评论者本人、管理员）能看到
const scope = ref(1)
const scopeOptions = [
  { value: 1, label: '公开可见' },
  { value: 2, label: '登录可见' },
  { value: 4, label: '仅作者可见' }
]

// 当前登录用户信息（根评论框展示 + 跳转作者主页）
const myLevel = computed(() => pickUserLevel(userStore.user))
const myTitle = computed(() => pickUserTitle(userStore.user))
const myTitleClass = computed(() => getTitleColorClass(myTitle.value))
// 我自己的头衔样式（装扮系统写入 users.json.title_decoration.payload）
const myTitlePayload = computed(() => pickUserTitlePayload(userStore.user))
function goMe() {
  const id = userStore.user?.id
  if (userStore.isLogged && id) router.push(`/author/${id}`)
}

// ---------- 举报 ----------
// 整棵评论树共用一个举报弹窗：每条评论各挂一个会白白渲染几十个弹窗
const reportTarget = ref(null)
const reportVisible = ref(false)
const reporting = ref(false)

function onReport(comment) {
  reportTarget.value = comment
  reportVisible.value = true
}

// 弹窗里显示「评论「…」」，方便确认没点错条目（表情指令等标记先去掉）
const reportLabel = computed(() => {
  const text = String(reportTarget.value?.content || '')
    .replace(/\[emoji:[^\]]*\]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  return text ? `评论「${text.slice(0, 20)}」` : '评论'
})

async function submitReport({ reason, detail, images }) {
  const target = reportTarget.value
  if (!target?.id) return
  reporting.value = true
  try {
    await createReport({ target_type: 'comment', target_id: target.id, reason, detail, images })
    reportVisible.value = false
    toast.success('举报已提交，我们会尽快处理')
  } catch {
    /* 拦截器已提示（重复举报 / 超过每日上限等） */
  } finally {
    reporting.value = false
  }
}

// ---------- 打赏 ----------
// 与举报一致：整棵评论树共用一个打赏弹窗
const tipTarget = ref(null)
const tipVisible = ref(false)

// 评论作者 id：优先评论自身的 uid，兜底取后端带回来的作者对象
const tipAuthorId = computed(() => {
  const item = tipTarget.value
  return Number(
    item?.uid || item?.result?.author?.id || item?.author?.id || item?.user?.id || 0
  )
})
const tipAuthorName = computed(() => {
  const item = tipTarget.value
  return String(item?.result?.author?.nickname || item?.author?.nickname || item?.user?.nickname || '')
})

/** 打开评论打赏弹窗（接收者由后端按评论作者解析） */
function onTip(comment) {
  tipTarget.value = comment
  tipVisible.value = true
}

const rawList = ref([])
const tree = ref([])
const newComment = ref('')

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23e8e6dd"/></svg>'
const loading = ref(false)
const page = ref(1)
const pageSize = 15
const total = ref(0)
const replyTo = ref(null)

// 拉取点赞数与已点赞状态（递归处理嵌套树里的所有评论，含子评论）
// 统一走 user-likes/states 批量接口：一次请求返回整棵树的 { is_liked, count }
// （此前是每条评论各调一次 is-liked，15 条根评论带回复能到几十个请求）
function flattenTree(list) {
  const result = []
  const walk = (arr) => {
    arr.forEach((c) => {
      result.push(c)
      if (c.replies?.length) walk(c.replies)
    })
  }
  walk(list)
  return result
}

// 递归统计全部评论数（含所有层级的子回复）
// 后端 flat 返回的 count 只统计根评论，需前端补齐子评论
function countAllComments(list) {
  let n = 0
  const walk = (arr) => {
    arr.forEach((c) => {
      n += 1
      if (c.replies?.length) walk(c.replies)
    })
  }
  walk(list)
  return n
}

async function loadLikes(list) {
  const flat = flattenTree(list)
  if (!flat.length) return
  const ids = flat.map((c) => c.id)

  // 批量查状态 + 数量：未登录时后端 is_liked 一律 false、count 照常返回，
  // 所以登录与否都走这一个接口，不需要再分两条路径。
  // 后端 states 单次最多接受 200 个 id（超出会被截断），评论树常大于这个量，按 200 分组并发。
  const CHUNK = 200
  const chunks = []
  for (let i = 0; i < ids.length; i += CHUNK) chunks.push(ids.slice(i, i + CHUNK))

  const states = {}
  let complete = true
  await Promise.all(
    chunks.map(async (part) => {
      const res = await likeStates('comment', part).catch(() => null)
      if (!res?.data?.states) {
        complete = false
        return
      }
      Object.assign(states, res.data.states)
    })
  )

  if (complete) {
    flat.forEach((c) => {
      const s = states[c.id] ?? states[String(c.id)]
      if (!s) return
      c.liked = !!s.is_liked
      c.likeCount = Number(s.count) || 0
    })
    return
  }

  // 兜底：states 不可用时退回批量计数，只补数量（状态保持默认未赞）
  const countsRes = await likesCount('comment', ids).catch(() => null)
  const countMap = countsRes?.data?.counts || {}
  flat.forEach((c) => {
    c.likeCount = countMap[c.id] ?? countMap[String(c.id)] ?? 0
  })
}

async function load() {
  loading.value = true
  try {
    const res = await getCommentTree(props.bindId, props.bindType, {
      page: page.value,
      limit: pageSize
    })
    const list = res.data?.data || []
    total.value = res.data?.count || list.length
    await loadLikes(list)
    rawList.value = list
    // 后端 comment/flat 已返回嵌套树（根评论带 replies 字段），无需前端再构树
    tree.value = list
    // 回传真实评论总数：后端 count 只含根评论，递归统计补齐子评论（取较大值兜底分页）
    emit('loaded', Math.max(total.value, countAllComments(list)))
  } catch {
    rawList.value = []
    tree.value = []
    total.value = 0
    emit('loaded', 0)
  } finally {
    loading.value = false
    // 从侧边栏等入口带 ?comment=xxx 跳转时，定位并高亮该评论
    if (props.highlightId) {
      nextTick(() => {
        const el = document.getElementById(`comment-${props.highlightId}`)
        if (el) {
          el.scrollIntoView({ behavior: 'smooth', block: 'center' })
          el.classList.add('comment-flash')
          setTimeout(() => el.classList.remove('comment-flash'), 2000)
        }
      })
    }
  }
}

async function onLike(comment) {
  if (!userStore.isLogged) {
    toast.warning('请先登录')
    return
  }
  const next = !comment.liked
  const prevCount = comment.likeCount || 0
  // 乐观更新
  comment.liked = next
  comment.likeCount = prevCount + (next ? 1 : -1)
  try {
    if (next) await like('comment', comment.id)
    else await unlike('comment', comment.id)
    // 用 is-liked 返回的实时 count 校准（后端为唯一数据源）
    const r = await isLiked('comment', comment.id).catch(() => null)
    if (r?.data?.count !== undefined) comment.likeCount = r.data.count
    comment.liked = next
    if (next) toast.success('点赞成功')
    else toast.info('已取消点赞')
  } catch {
    comment.liked = !next
    comment.likeCount = prevCount
    toast.error('操作失败')
  }
}

function onReply(comment) {
  replyTo.value = { id: comment.id, name: pickCommentAuthor(comment).nickname || '匿名' }
}

const newImages = ref([])
const uploading = ref(false)
const fileInput = ref(null)

// 选择图片后立即上传，仅保存返回的 URL
async function onPickImages(e) {
  const files = Array.from(e.target.files || [])
  // 清空 value，避免连续选择同一文件时不再触发 change
  e.target.value = ''
  if (!files.length) return

  if (newImages.value.length + files.length > 9) {
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
    newImages.value.push(...rows.map((r) => ({ image: r.full_url, video: r.live_video || '' })))
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
  newImages.value.splice(i, 1)
}

// 附件库：把选中的已上传图片加入待发表列表（去重 + 数量上限）
// items 里带 live_video（后端按「<封面名>-live.mp4」反查配对视频），
// 从库里选历史实况图时也能直接成为实况图
const showLibrary = ref(false)
function onLibrarySelect(urls = [], items = []) {
  const rest = 9 - newImages.value.length
  if (rest <= 0) {
    toast.warning('最多只能上传 9 张图片')
    return
  }
  const exists = new Set(newImages.value.map((item) => item.image))
  const liveByUrl = new Map(items.map((item) => [item.full_url, item.live_video || '']))
  const add = urls.filter((url) => url && !exists.has(url)).slice(0, rest)
  newImages.value.push(...add.map((url) => ({ image: url, video: liveByUrl.get(url) || '' })))
  if (!add.length) {
    toast.info('所选图片已在列表中')
  } else if (add.length < urls.length) {
    toast.warning('受数量限制，部分附件未添加')
  } else {
    toast.success(`已添加 ${add.length} 张图片`)
  }
}

// 发表根评论
async function submitRoot() {
  const content = newComment.value.trim()
  if (!content && !newImages.value.length) return
  await onSubmit({
    content,
    // 实况图序列化成「图片::视频」，后端按逗号拼接后原样存库
    images: newImages.value.map(serializeLiveItem).filter(Boolean),
    pid: 0,
    scope: Number(scope.value)
  })
  newComment.value = ''
  newImages.value = []
  scope.value = 1
}

// 待审核占位（只存本地，不写库）：审核开启时，刚提交的评论不会立刻出现在列表里
// （非管理员的评论列表只返回 audit=1），用一条占位提示用户「已提交、等待审核」
// 审核状态取值与后端 app/api/controller/audit.go 一致：0 待审核 / 1 已通过 / 2 未通过
const AUDIT_PENDING = 0
const pendingComments = ref([])
const MAX_PENDING = 3

function pushPending({ content, images }) {
  pendingComments.value.unshift({
    key: `pending-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`,
    content: String(content || ''),
    hasImages: (images || []).length > 0
  })
  if (pendingComments.value.length > MAX_PENDING) {
    pendingComments.value = pendingComments.value.slice(0, MAX_PENDING)
  }
}

function removePending(key) {
  pendingComments.value = pendingComments.value.filter((item) => item.key !== key)
}

async function onSubmit({ content, images, pid, scope: rawScope = 1 }) {
  if (!userStore.isLogged) {
    toast.warning('请先登录')
    return
  }
  try {
    // 不传 audit：是否进入待审核由后端按站点「评论审核」开关决定
    // （后台「系统设置 → 评论配置 → 内容审核」）。
    // 过去这里强制传 audit=1 / status=1，虽然会被后端 allow 白名单丢掉，
    // 但一旦哪天放开就会让审核开关形同虚设，所以直接不传
    // 可见范围：1 公开 / 2 登录可见 / 4 仅作者可见（评论不支持付费可见）
    const commentScope = [1, 2, 4].includes(Number(rawScope)) ? Number(rawScope) : 1
    const res = await createComment({
      bind_id: props.bindId,
      bind_type: props.bindType,
      pid: pid || 0,
      content,
      // 后端 processFieldValue 会把数组转为逗号分隔字符串
      images: images || [],
      scope: commentScope
    })
    replyTo.value = null

    // 后端返回本次写入的审核状态（0 待审核 / 1 已通过）
    const audit = Number(res?.data?.audit ?? 1)
    if (audit === AUDIT_PENDING) {
      pushPending({ content, images })
      toast.success('评论已提交，审核通过后显示')
      return
    }

    toast.success('评论成功')
    page.value = 1
    load()
  } catch {
    toast.error('评论失败')
  }
}

async function onRemove(comment) {
  if (!confirm('确定删除该评论吗？')) return
  try {
    await removeComment(String(comment.id))
    toast.success('已删除')
    load()
  } catch {
    toast.error('删除失败')
  }
}

watch(() => props.bindId, () => {
  page.value = 1
  load()
})
onMounted(load)

defineExpose({ load })
</script>

<style scoped>
.comment-tree {
  margin-top: 8px;
}
/* 待审核占位条：刚提交、等待管理员审核的评论 */
.pending-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}
.pending-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  font-size: 13px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm, 4px);
  background: var(--bg-muted);
}
.pending-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--warning, #b8860b);
  background: var(--gold-wash, rgba(184, 134, 11, 0.12));
}
.pending-content {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text);
}
.pending-content em {
  font-style: normal;
  color: var(--text-muted);
}
.pending-tip {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
}
.pending-del {
  flex-shrink: 0;
  padding: 2px 4px;
  font-size: 11px;
  border: none;
  background: transparent;
  color: var(--text-light, var(--text-muted));
  cursor: pointer;
}
.pending-del:hover {
  color: var(--danger);
}
/* 来自侧边栏等入口的评论定位高亮 */
:deep(.comment-item.comment-highlight) {
  scroll-margin-top: 80px;
}
:deep(.comment-item.comment-flash) {
  animation: comment-flash 2s ease;
  border-radius: var(--radius-lg);
}
@keyframes comment-flash {
  0%, 100% { background: transparent; box-shadow: none; }
  20% { background: var(--accent-soft); box-shadow: 0 0 0 2px var(--accent-glow); }
}

.comment-loading {
  padding: 32px;
  text-align: center;
  color: var(--text-muted);
  font-size: 13px;
}

/* 评论已关闭（禁止评论时替代输入模块） */
.comment-closed {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 16px;
  padding: 12px 14px;
  font-size: 13px;
  color: var(--text-muted);
  background: var(--bg-alt);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-lg);
}
.comment-closed .bi {
  font-size: 14px;
}

.root-comment-box {
  display: flex;
  gap: 10px;
  padding: 12px;
  background: var(--bg-alt);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  margin-bottom: 16px;
}
.root-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  flex-shrink: 0;
  object-fit: cover;
  cursor: pointer;
}
.root-input {
  flex: 1;
  min-width: 0;
}
.root-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
  font-size: 13px;
}
.login-tip {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.login-tip-text {
  color: var(--text-muted);
  font-size: 13px;
}
.login-tip-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.root-name {
  font-weight: 600;
  color: var(--text);
  cursor: pointer;
  transition: color 0.2s;
}
.root-name:hover {
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
.root-input textarea {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 8px 10px;
  font-size: 15px;
  line-height: 1.7;
  resize: vertical;
  background: var(--bg-muted);
  color: var(--text);
  font-family: inherit;
  box-sizing: border-box;
}
.root-input textarea:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.root-scope {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.root-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 8px;
}
.hidden-file {
  display: none;
}
.img-tip {
  font-size: 12px;
  color: var(--text-muted);
}
/* 已选图片预览 */
.img-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}
.img-thumb {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: var(--radius-xs);
  overflow: hidden;
  border: 1px solid var(--border);
}
.img-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
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
</style>
