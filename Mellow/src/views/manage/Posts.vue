<template>
  <div class="card card-pad">
    <!-- 头部 -->
    <header class="posts-head">
      <div>
        <h2 class="block-title">我的文章</h2>
        <p class="block-desc">管理自己发布的文章与草稿</p>
      </div>
      <router-link to="/manage/posts/write" class="btn btn-primary btn-sm">
        <i class="bi bi-pencil-square" /> 写文章
      </router-link>
    </header>

    <!-- 筛选栏 -->
    <div class="posts-filter">
      <div class="status-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          type="button"
          class="status-tab"
          :class="{ 'is-active': status === tab.key }"
          @click="switchTab(tab.key)"
        >
          {{ tab.label }}
          <em v-if="tab.key === 'audit' && pendingCount > 0" class="tab-badge">{{ pendingCount }}</em>
        </button>
      </div>
      <div class="search-box">
        <i class="bi bi-search" />
        <FormInput
          v-model="keyword"
          class="search-input"
          type="search"
          placeholder="搜索标题…"
          @input="doSearch" />
      </div>
    </div>

    <p v-if="pendingCount > 0" class="audit-tip">
      <i class="bi bi-info-circle" /> 有 {{ pendingCount }} 篇文章正在等待审核，审核通过后访客才能看到。
    </p>

    <!-- 列表 -->
    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>

    <div v-else-if="!list.length" class="empty-row">
      <EmptyState :text="keyword ? '没有匹配的文章' : '还没有文章，去写一篇吧'" />
      <div class="empty-action">
        <router-link to="/manage/posts/write" class="btn btn-primary btn-sm">写文章</router-link>
      </div>
    </div>

    <ul v-else class="post-list">
      <li v-for="item in list" :key="item.id" class="post-row">
        <div class="post-cover" @click="viewArticle(item)">
          <img v-if="coverOf(item)" :src="coverOf(item)" :alt="item.title" loading="lazy" />
          <i v-else class="bi bi-file-richtext" />
        </div>

        <div class="post-main">
          <h3 class="post-title">
            <a href="javascript:;" @click="viewArticle(item)">{{ item.title || '无标题' }}</a>
            <span class="post-status" :class="statusClass(item)">
              {{ statusLabel(item) }}
            </span>
          </h3>

          <p class="post-abstract">
            <EmojiText v-if="item.abstract" :text="item.abstract" :size="16" />
            <template v-else>暂无摘要</template>
          </p>

          <div class="post-meta">
            <!-- 当前内容的可见范围：公开 / 登录可见 / 付费 · N 积分 -->
            <ScopeBadge :scope="item.scope" :integral="item.scope_integral" />
            <span v-if="groupNameOf(item)" class="meta-chip">
              <i class="bi bi-folder2" /> {{ groupNameOf(item) }}
            </span>
            <span v-for="t in tagNamesOf(item)" :key="t" class="meta-chip">
              <i class="bi bi-tag" /> {{ t }}
            </span>
            <span class="meta-text"><i class="bi bi-eye" /> {{ item.views || 0 }}</span>
            <span class="meta-text"><i class="bi bi-clock" /> {{ fromNow(item.create_time) }}</span>
          </div>

          <!-- 驳回原因：管理员审核未通过时填写，作者据此修改（审核通过后后端会清空） -->
          <p v-if="item.reason" class="audit-reason">
            <i class="bi bi-info-circle" /> 驳回原因：{{ item.reason }}
          </p>
        </div>

        <div class="post-actions">
          <button class="btn btn-ghost btn-sm" title="查看" @click="viewArticle(item)">
            <i class="bi bi-eye" />
          </button>
          <router-link :to="`/manage/posts/edit/${item.id}`" class="btn btn-ghost btn-sm" title="编辑">
            <i class="bi bi-pencil" />
          </router-link>
          <button class="btn btn-ghost btn-sm danger" title="删除" @click="confirmRemove(item)">
            <i class="bi bi-trash" />
          </button>
        </div>
      </li>
    </ul>

    <Pagination
      v-if="!loading && total > pageSize"
      :current="page"
      :total="total"
      :page-size="pageSize"
      @update:current="changePage"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import EmojiText from '@/components/EmojiText.vue'
import ScopeBadge from '@/components/ScopeBadge.vue'
import { getMyArticles, countMyArticles, removeArticle, getArticleGroups } from '@/api/article'
import { listAllTags } from '@/api/tags'
import { useUserStore } from '@/stores/user'
import { storeToRefs } from 'pinia'
import { fromNow } from '@/utils/time'
import { parseIdField, debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const router = useRouter()
const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const uid = computed(() => user.value?.id || 0)

const pageSize = 10
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const pendingCount = ref(0)

const status = ref('all')
const keyword = ref('')
const searchKey = ref('')

const groups = ref([])
const tags = ref([])

// 筛选标签：where 为附加查询条件（uid 由 load 统一带上）
// 待审核 / 未通过只有作者本人能看到（后端对「查询自己」的请求不过滤 audit）
const tabs = [
  { key: 'all', label: '全部', where: {} },
  { key: 'pub', label: '已发布', where: { status: 1, audit: 1 } },
  { key: 'audit', label: '待审核', where: { status: 1, audit: 0 } },
  { key: 'draft', label: '草稿', where: { status: 0 } }
]

const currentTab = computed(() => tabs.find((tab) => tab.key === status.value) || tabs[0])

// 状态文案 / 配色：草稿、待审核、未通过、已发布
function statusLabel(item) {
  if (Number(item.status) === 0) return '草稿'
  const audit = Number(item.audit)
  if (audit === 0) return '待审核'
  if (audit === 2) return '未通过'
  return '已发布'
}

function statusClass(item) {
  if (Number(item.status) === 0) return 'is-draft'
  const audit = Number(item.audit)
  if (audit === 0) return 'is-audit'
  if (audit === 2) return 'is-reject'
  return 'is-pub'
}

const groupMap = computed(() => {
  const map = {}
  groups.value.forEach((g) => { map[g.id] = g.name })
  return map
})

const tagMap = computed(() => {
  const map = {}
  tags.value.forEach((t) => { map[t.id] = t.name })
  return map
})

function coverOf(item) {
  const c = item.covers
  if (!c) return ''
  if (Array.isArray(c)) return c[0] || ''
  return String(c).split(',')[0] || ''
}

function groupNameOf(item) {
  const ids = parseIdField(item.group)
  return ids.length ? groupMap.value[ids[0]] || '' : ''
}

function tagNamesOf(item) {
  return parseIdField(item.tags)
    .map((id) => tagMap.value[id])
    .filter(Boolean)
    .slice(0, 3)
}

async function load() {
  if (!uid.value) return
  loading.value = true
  try {
    const where = { uid: uid.value, ...currentTab.value.where }
    const params = {
      page: page.value,
      limit: pageSize,
      where: JSON.stringify(where)
    }
    // like 使用后端支持的「字段名|搜索值」格式，并剔除危险字符
    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `title|${kw}`

    const res = await getMyArticles(uid.value, params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function loadPending() {
  if (!uid.value) return
  try {
    const res = await countMyArticles(uid.value, { audit: 0 })
    pendingCount.value = Number(res.data) || 0
  } catch {
    pendingCount.value = 0
  }
}

async function loadMeta() {
  try {
    const res = await getArticleGroups()
    groups.value = res.data?.data || []
  } catch {
    groups.value = []
  }
  try {
    const res = await listAllTags()
    tags.value = res.data?.data || []
  } catch {
    tags.value = []
  }
}

function switchTab(val) {
  if (status.value === val) return
  status.value = val
  page.value = 1
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  page.value = 1
  load()
}, 350)

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

function viewArticle(item) {
  // 草稿没有前台页面，直接进编辑器；
  // 其余状态（含待审核 / 未通过）后端允许作者本人查看，可以正常打开详情页预览
  router.push(Number(item.status) === 0 ? `/manage/posts/edit/${item.id}` : `/archives/${item.id}`)
}

function confirmRemove(item) {
  if (!window.confirm(`确定删除《${item.title || '无标题'}》吗？删除后可在回收站找回。`)) return
  removeArticle(item.id)
    .then(() => {
      toast.success('已删除')
      // 最后一页只剩一条时回退一页
      if (list.value.length === 1 && page.value > 1) page.value -= 1
      load()
    })
    .catch(() => {})
}

// 清空关键词时立即恢复完整列表
watch(keyword, (val) => {
  if (!val) {
    searchKey.value = ''
    page.value = 1
    load()
  }
})

onMounted(() => {
  loadMeta()
  load()
  loadPending()
})
</script>

<style scoped>
.posts-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.block-title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}

.posts-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.status-tabs {
  display: flex;
  gap: 6px;
}
.status-tab {
  /* 与筛选栏内的下拉 / 搜索框 / 按钮等高 */
  height: var(--control-h-sm);
  padding: 0 14px;
  font-size: 13px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  background: var(--bg-muted);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.status-tab:hover {
  color: var(--primary);
}
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
}
/* 待审核数量角标 */
.tab-badge {
  display: inline-block;
  margin-left: 4px;
  padding: 0 5px;
  font-size: 11px;
  font-style: normal;
  line-height: 16px;
  border-radius: 8px;
  background: var(--gold-wash);
  color: var(--warning);
}
.status-tab.is-active .tab-badge {
  background: rgba(255, 255, 255, 0.24);
  color: #fff;
}
.search-box {
  position: relative;
  margin-left: auto;
  width: 200px;
}
.search-box .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
}
.search-input {
  width: 100%;
  height: var(--control-h-sm);
  padding: 0 12px 0 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

.audit-tip {
  margin: 12px 0 0;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: var(--radius-sm);
}

.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 24px 0;
  text-align: center;
}
.empty-action {
  margin-top: 12px;
}

.post-list {
  display: flex;
  flex-direction: column;
}
.post-row {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.post-row:last-child {
  border-bottom: none;
}
.post-cover {
  width: 108px;
  height: 72px;
  flex-shrink: 0;
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--bg-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-light);
  font-size: 22px;
  cursor: pointer;
}
.post-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}
.post-cover:hover img {
  transform: scale(1.05);
}
.post-main {
  flex: 1;
  min-width: 0;
}
.post-title {
  margin: 0 0 6px;
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.post-title a {
  color: var(--text);
}
.post-title a:hover {
  color: var(--primary);
}
.post-status {
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 400;
  border-radius: 3px;
}
.post-status.is-pub {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.post-status.is-draft {
  background: var(--bg-muted);
  color: var(--text-muted);
}
/* 驳回原因：文字可能较长，允许换行 */
.audit-reason {
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--danger);
  word-break: break-word;
}
.post-status.is-audit {
  background: var(--gold-wash);
  color: var(--warning);
}
.post-status.is-reject {
  background: var(--accent-soft);
  color: var(--danger);
}
.post-abstract {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--text-soft);
  line-height: 1.7;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.post-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  background: var(--bg-muted);
  border-radius: 3px;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.post-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.post-actions .danger:hover {
  border-color: var(--danger);
  color: var(--danger);
}

@media (max-width: 640px) {
  .posts-filter {
    gap: 8px;
  }
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .post-row {
    flex-wrap: wrap;
  }
  .post-cover {
    width: 84px;
    height: 60px;
  }
  .post-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
