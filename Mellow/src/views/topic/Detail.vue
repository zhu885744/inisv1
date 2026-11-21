<template>
  <div class="topic-page">
    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>

    <EmptyState v-else-if="!topic" text="话题不存在或已被删除" />

    <template v-else>
      <!-- 话题头部：封面 / 简介 / 数据 / 关注 -->
      <header class="topic-hero card card-pad">
        <img
          v-if="topic.avatar"
          :src="topic.avatar"
          class="topic-cover"
          alt=""
          @error="onCoverError"
          loading="lazy"
        />
        <div class="topic-main">
          <h1 class="topic-title">#{{ topic.name }}#</h1>
          <p class="topic-desc">{{ topic.description || '这个话题还没有简介' }}</p>
          <div class="topic-meta">
            <span><i class="bi bi-chat-square-quote" /> {{ format(topic.moment_count) }} 条动态</span>
            <span><i class="bi bi-file-earmark-text" /> {{ format(topic.article_count) }} 篇文章</span>
            <span><i class="bi bi-people" /> {{ format(topic.follow_count) }} 人关注</span>
            <span v-if="topic.moderator" class="topic-moderator">
              <i class="bi bi-award" /> 主持人：
              <RouterLink :to="`/author/${topic.moderator.id}`">{{ topic.moderator.nickname }}</RouterLink>
            </span>
          </div>
        </div>

        <!-- 按钮统一 btn-sm（32px，与作者主页头部的操作按钮同一尺寸）；
             .topic-actions 是纵向 stretch，所以三者宽度、高度都一致 -->
        <div class="topic-actions">
          <button class="btn btn-sm btn-primary" type="button" :disabled="followBusy" @click="toggleFollow">
            <i :class="topic.is_following ? 'bi bi-check2-circle' : 'bi bi-plus-lg'" />
            {{ topic.is_following ? '已关注' : '关注话题' }}
          </button>
          <button class="btn btn-sm btn-ghost" type="button" @click="shareTopic">
            <i class="bi bi-share" /> 分享
          </button>
          <!-- 主持人本人（或管理员）可以改简介 / 封面 -->
          <button v-if="canEdit" class="btn btn-sm btn-ghost" type="button" @click="openEdit">
            <i class="bi bi-pencil" /> 编辑话题
          </button>
        </div>
      </header>

      <!-- 内容切换：动态 / 文章 + 排序 -->
      <div class="topic-bar">
        <div class="topic-tabs">
          <button
            v-for="item in tabs"
            :key="item.key"
            type="button"
            class="topic-tab"
            :class="{ 'is-active': tab === item.key }"
            @click="switchTab(item.key)"
          >
            {{ item.label }}
          </button>
        </div>
        <div class="topic-sort">
          <button
            v-for="item in sorts"
            :key="item.key"
            type="button"
            class="topic-sort-item"
            :class="{ 'is-active': sort === item.key }"
            @click="switchSort(item.key)"
          >
            {{ item.label }}
          </button>
        </div>
      </div>

      <div class="topic-list">
        <div v-if="listLoading" class="is-loading"><span class="spinner" /> 加载中...</div>
        <template v-else>
          <div v-for="row in rows" :key="`${row.type}-${row.item.id}`" class="topic-item">
            <!-- 主持人操作条：置顶 / 移出话题 / 删除内容（置顶徽标所有人可见） -->
            <div v-if="canEdit || row.item.is_pinned" class="topic-item-bar">
              <span v-if="row.item.is_pinned" class="topic-pin-badge">
                <i class="bi bi-pin-angle-fill" /> 话题内置顶
              </span>
              <template v-if="canEdit">
                <button
                  type="button"
                  class="topic-item-btn"
                  :disabled="itemBusy === row.item.id"
                  @click="togglePin(row)"
                >
                  <i class="bi" :class="row.item.is_pinned ? 'bi-pin-angle' : 'bi-pin-angle-fill'" />
                  {{ row.item.is_pinned ? '取消置顶' : '置顶' }}
                </button>
                <button
                  type="button"
                  class="topic-item-btn"
                  :disabled="itemBusy === row.item.id"
                  @click="removeFromTopic(row)"
                >
                  <i class="bi bi-box-arrow-up" /> 移出话题
                </button>
                <button
                  type="button"
                  class="topic-item-btn is-danger"
                  :disabled="itemBusy === row.item.id"
                  @click="deleteItem(row)"
                >
                  <i class="bi bi-trash" /> 删除内容
                </button>
              </template>
            </div>

            <MomentItem v-if="row.type === 'moments'" :moment="row.item" @delete="loadContent" />
            <!-- 文章卡片与首页「发现」流同款（作者头部 + 标题 + 正文片段 + 互动栏） -->
            <ArticleFeedItem v-else :article="row.item" :like-state="articleLikeStates[row.item.id]" />
          </div>

          <EmptyState
            v-if="!currentList.length"
            :text="tab === 'moments' ? '这个话题下还没有动态' : '这个话题下还没有文章'"
          />
        </template>
      </div>

      <Pagination
        :current="page"
        :total="total"
        :page-size="limit"
        @update:current="(p) => { page = p; loadContent() }"
      />
    </template>

    <!-- 主持人编辑：简介 / 封面 -->
    <div v-if="editVisible" class="topic-edit-mask" @click.self="editVisible = false">
      <div class="topic-edit card card-pad">
        <h3 class="topic-edit-title">编辑话题资料</h3>
        <label class="topic-edit-label">
          话题简介
          <FormTextarea v-model="editForm.description" rows="4" maxlength="200" placeholder="介绍一下这个话题" />
        </label>
        <label class="topic-edit-label">
          封面图片
          <FormInput v-model="editForm.avatar" type="text" placeholder="https://..." />
        </label>
        <div class="topic-edit-images">
          <button type="button" class="btn btn-sm" :disabled="uploading" @click="coverFileRef?.click()">
            <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
          </button>
          <button type="button" class="btn btn-sm" title="从附件库选择" @click="showCoverLibrary = true">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
          <button v-if="editForm.avatar" type="button" class="btn btn-sm btn-ghost" @click="editForm.avatar = ''">
            移除
          </button>
          <input ref="coverFileRef" type="file" accept="image/*" hidden @change="onCoverChange" />
        </div>

        <!-- 附件库：选一张已上传的图片作为话题封面 -->
        <AttachmentLibrary
          v-model:visible="showCoverLibrary"
          title="选择话题封面"
          accept="image"
          @select="onCoverLibrarySelect"
        />
        <p class="topic-edit-tip">话题名不支持修改：正文里的 #话题# 链接是按名称写入的。</p>
        <div class="topic-edit-actions">
          <button class="btn btn-sm" type="button" @click="editVisible = false">取消</button>
          <button class="btn btn-sm btn-primary" type="button" :disabled="editBusy" @click="saveEdit">
            {{ editBusy ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 话题聚合页 /topic/:key（key 支持话题 id 或话题名）
 *
 * 数据来源：
 * - topics/detail  → 话题信息 + 内容量 + 关注数 + 主持人
 * - topics/content → 话题下的动态 / 文章分页（动态只算已发布且审核通过的）
 * - topics/follow | unfollow → 关注 / 取关
 *
 * 正文里的 #话题# 由后端规范化为 #[话题名](/topic/id)（见 app/model/topics.go），
 * 动态 / 文章里的链接都会回到这里。
 */
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { storeToRefs } from 'pinia'
import ArticleFeedItem from '@/components/ArticleFeedItem.vue'
import MomentItem from '@/components/MomentItem.vue'
import Pagination from '@/components/Pagination.vue'
import EmptyState from '@/components/EmptyState.vue'
import {
  getTopic,
  getTopicContent,
  followTopic,
  unfollowTopic,
  updateTopicProfile,
  pinTopicContent,
  unpinTopicContent,
  removeTopicContent,
  deleteTopicContent
} from '@/api/topics'
import { useUserStore } from '@/stores/user'
import { uploadAttachments } from '@/api/attachment'
import { likeStates } from '@/api/tags'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { isAdmin, copyText } from '@/utils/helper'
import { toast } from '@/utils/toast'

const route = useRoute()
const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const tabs = [
  { key: 'moments', label: '动态' },
  { key: 'article', label: '文章' }
]
const sorts = [
  { key: 'new', label: '最新' },
  { key: 'hot', label: '最热' },
  { key: 'top', label: '置顶' }
]

const loading = ref(true)
const listLoading = ref(false)
const topic = ref(null)
const moments = ref([])
const articles = ref([])
const tab = ref('moments')
const sort = ref('new')
const page = ref(1)
const total = ref(0)
const limit = 10
const followBusy = ref(false)

const currentList = computed(() => (tab.value === 'moments' ? moments.value : articles.value))

/* ---------- 文章卡片的批量点赞状态 ----------
 * 与首页同一套做法：拿到一页文章后批量查一次，再通过 like-state 下发给卡片，
 * 避免每张卡片挂载后各自请求 is-liked / counts（一页 10 条就是 10 个请求）。
 */
const articleLikeStates = reactive({})

async function loadArticleLikeStates(list) {
  const ids = list.map((item) => Number(item.id)).filter(Boolean)
  if (!ids.length) return

  try {
    const res = await likeStates('article', ids)
    const states = res?.data?.states || {}
    Object.entries(states).forEach(([id, state]) => {
      articleLikeStates[id] = state
    })
  } catch {
    // 批量失败不阻塞列表：卡片会退化成自己查一次（见 ArticleFeedItem 的 onMounted）
  }
}

// 统一成一种行结构，便于同一个模板里渲染「主持人操作条 + 不同类型的内容卡片」
const rows = computed(() =>
  tab.value === 'moments'
    ? moments.value.map((item) => ({ type: 'moments', item }))
    : articles.value.map((item) => ({ type: 'article', item }))
)

// 主持人本人或管理员可编辑话题资料
const canEdit = computed(() => {
  const uid = Number(user.value?.id) || 0
  const moderator = Number(topic.value?.moderator_uid) || 0
  return isAdmin(user.value) || (uid > 0 && moderator > 0 && uid === moderator)
})

function format(n) {
  return typeof n === 'number' ? n.toLocaleString('zh-CN') : n ?? 0
}

function onCoverError(e) {
  e.target.style.display = 'none'
}

// ---------- 话题信息 ----------
async function loadTopic() {
  const key = String(route.params.key || '')
  const params = /^\d+$/.test(key) ? { id: key } : { key: decodeURIComponent(key) }

  try {
    const res = await getTopic(params)
    topic.value = res?.data || null
  } catch {
    topic.value = null
  }
}

// ---------- 话题内容 ----------
async function loadContent() {
  if (!topic.value?.id) return
  listLoading.value = true
  try {
    const res = await getTopicContent({
      id: topic.value.id,
      type: tab.value,
      sort: sort.value,
      page: page.value,
      limit
    })
    const list = res?.data?.data || []
    total.value = res?.data?.count || 0
    if (tab.value === 'moments') {
      moments.value = list
      articles.value = []
    } else {
      // 先备好点赞状态再渲染，否则卡片挂载后会先各自查一次
      await loadArticleLikeStates(list)
      articles.value = list
      moments.value = []
    }
  } catch {
    total.value = 0
    moments.value = []
    articles.value = []
  } finally {
    listLoading.value = false
  }
}

function switchTab(key) {
  if (tab.value === key) return
  tab.value = key
  page.value = 1
  loadContent()
}

// 「置顶」只有动态有意义（文章没有置顶字段）
function switchSort(key) {
  if (sort.value === key) return
  sort.value = key
  page.value = 1
  loadContent()
}

// ---------- 关注 / 取关 ----------
async function toggleFollow() {
  if (followBusy.value) return
  if (!user.value?.id) {
    toast.warning('请先登录后再关注话题')
    return
  }
  followBusy.value = true
  try {
    const isFollowing = !!topic.value.is_following
    const res = isFollowing ? await unfollowTopic(topic.value.id) : await followTopic(topic.value.id)
    topic.value.is_following = !isFollowing
    if (res?.data?.follow_count !== undefined) {
      topic.value.follow_count = res.data.follow_count
    }
    toast.success(isFollowing ? '已取消关注' : '关注成功')
  } catch (e) {
    toast.error(e?.msg || '操作失败，请稍后重试')
  } finally {
    followBusy.value = false
  }
}

// ---------- 分享 ----------
/**
 * 分享话题：复制「话题名 + 话题链接」
 *
 * 与独立页面（views/page/Index.vue 的 sharePage）同一口径：统一复制文本而不是调
 * navigator.share —— 桌面端没有原生分享面板，行为一致更可预期。
 * 链接用 id 形式的规范地址（而不是当前地址栏），这样即使用户是从
 * /topic/旅行 进来的、或话题后来改了名，分享出去的链接依然有效。
 */
async function shareTopic() {
  if (!topic.value?.name) return

  const url = `${window.location.origin}/topic/${topic.value.id}`
  const ok = await copyText(`【#${topic.value.name}#】${url}`)

  if (ok) {
    toast.success('已复制话题链接和标题，去分享吧')
  } else {
    toast.error('复制失败，请手动复制')
  }
}

// ---------- 主持人编辑 ----------
const editVisible = ref(false)
const editBusy = ref(false)
const editForm = ref({ description: '', avatar: '' })

// 封面图片：上传 / 附件库
const coverFileRef = ref(null)
const uploading = ref(false)
const showCoverLibrary = ref(false)

async function onCoverChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')

    editForm.value.avatar = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

function onCoverLibrarySelect(urls = []) {
  const url = urls[0]
  showCoverLibrary.value = false
  if (url) editForm.value.avatar = url
}

function openEdit() {
  editForm.value = {
    description: topic.value?.description || '',
    avatar: topic.value?.avatar || ''
  }
  editVisible.value = true
}

async function saveEdit() {
  if (editBusy.value) return
  editBusy.value = true
  try {
    await updateTopicProfile({ id: topic.value.id, ...editForm.value })
    topic.value.description = editForm.value.description
    topic.value.avatar = editForm.value.avatar
    editVisible.value = false
    toast.success('话题资料已更新')
  } catch (e) {
    toast.error(e?.msg || '保存失败，请稍后重试')
  } finally {
    editBusy.value = false
  }
}

// ---------- 主持人操作：置顶 / 移出话题 / 删除内容 ----------
const itemBusy = ref(0)

function targetPayload(row) {
  return { id: topic.value.id, type: row.type, target_id: row.item.id }
}

// 置顶：影响列表排序，操作完重新拉一遍
async function togglePin(row) {
  if (itemBusy.value) return
  itemBusy.value = row.item.id
  try {
    const on = !row.item.is_pinned
    const res = on ? await pinTopicContent(targetPayload(row)) : await unpinTopicContent(targetPayload(row))
    row.item.is_pinned = res?.data?.is_pinned ?? on
    toast.success(on ? '已置顶' : '已取消置顶')
    await loadContent()
  } catch (e) {
    toast.error(e?.msg || '操作失败，请稍后重试')
  } finally {
    itemBusy.value = 0
  }
}

// 移出话题：只解除内容与话题的关联，不删除内容本身
async function removeFromTopic(row) {
  if (itemBusy.value) return
  if (!confirm('把这条内容移出该话题？内容本身不会被删除。')) return

  itemBusy.value = row.item.id
  try {
    await removeTopicContent(targetPayload(row))
    toast.success('已移出话题')
    await loadTopic()
    await loadContent()
  } catch (e) {
    toast.error(e?.msg || '操作失败，请稍后重试')
  } finally {
    itemBusy.value = 0
  }
}

// 删除内容：软删除，后台回收站可恢复
async function deleteItem(row) {
  if (itemBusy.value) return
  if (!confirm('确定删除这条内容？删除后可在后台回收站恢复。')) return

  itemBusy.value = row.item.id
  try {
    await deleteTopicContent(targetPayload(row))
    toast.success('已删除')
    await loadTopic()
    await loadContent()
  } catch (e) {
    toast.error(e?.msg || '操作失败，请稍后重试')
  } finally {
    itemBusy.value = 0
  }
}

// ---------- 初始化 ----------
async function init() {
  loading.value = true
  page.value = 1
  await loadTopic()
  if (topic.value) await loadContent()
  loading.value = false
}

// /topic/1 → /topic/2 组件复用，重新加载
watch(() => route.params.key, init)

onMounted(init)
</script>

<style scoped>
.is-loading {
  padding: 48px;
  text-align: center;
  color: var(--text-muted);
}

/* ---------- 头部 ---------- */
.topic-hero {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 20px;
  background: linear-gradient(135deg, var(--accent-wash), var(--bg-card));
}
.topic-cover {
  width: 88px;
  height: 88px;
  border-radius: var(--radius);
  object-fit: cover;
  flex-shrink: 0;
  background: var(--bg-muted);
}
.topic-main {
  flex: 1;
  min-width: 0;
}
.topic-title {
  font-family: var(--font-serif);
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 6px;
  word-break: break-all;
}
.topic-desc {
  margin: 0 0 10px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-muted);
}
.topic-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  font-size: 12px;
  color: var(--text-muted);
}
.topic-meta .bi {
  color: var(--primary);
  margin-right: 3px;
}
.topic-moderator a {
  color: var(--primary);
  text-decoration: none;
}
.topic-moderator a:hover {
  text-decoration: underline;
}
.topic-actions {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  flex-shrink: 0;
}

/* ---------- 切换条 ---------- */
.topic-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}
.topic-tabs {
  display: flex;
  gap: 6px;
}
.topic-tab {
  padding: 7px 16px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.18s;
}
.topic-tab:hover {
  border-color: var(--primary-soft);
  color: var(--text);
}
.topic-tab.is-active {
  background: var(--accent-soft);
  border-color: var(--primary-soft);
  color: var(--primary);
  font-weight: 600;
}
.topic-sort {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 13px;
}
.topic-sort-item {
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 13px;
  font-family: inherit;
  cursor: pointer;
  padding: 2px 0;
}
.topic-sort-item:hover {
  color: var(--text);
}
.topic-sort-item.is-active {
  color: var(--primary);
  font-weight: 600;
}

.topic-list {
  padding: 8px 0;
}
.topic-item {
  position: relative;
}
/* 主持人操作条：贴在内容卡片上方，轻量、不抢正文 */
.topic-item-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 6px;
  padding: 0 4px;
}
.topic-pin-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  border-radius: 999px;
  background: var(--gold-wash);
  color: var(--warning);
  font-size: 12px;
}
.topic-item-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 12px;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.18s;
}
.topic-item-btn:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.topic-item-btn.is-danger:hover {
  border-color: var(--danger);
  color: var(--danger);
}
.topic-item-btn:disabled {
  opacity: 0.6;
  cursor: default;
}

/* ---------- 编辑弹窗 ---------- */
.topic-edit-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.45);
}
.topic-edit {
  width: 100%;
  max-width: 420px;
}
.topic-edit-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 14px;
}
.topic-edit-label {
  display: block;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--text-soft);
}
.topic-edit-label textarea,
.topic-edit-label input {
  display: block;
  width: 100%;
  margin-top: 6px;
  padding: 9px 10px;
  background: var(--bg-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
  outline: none;
}
.topic-edit-label textarea:focus,
.topic-edit-label input:focus {
  border-color: var(--primary-soft);
}
.topic-edit-tip {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--text-light);
}
.topic-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.topic-edit-images {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
  margin-bottom: 12px;
}

@media (max-width: 640px) {
  .topic-hero {
    flex-direction: column;
  }
  .topic-actions {
    flex-direction: row;
  }
}
</style>
