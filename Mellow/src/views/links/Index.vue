<template>
  <div>
    <SectionTitle title="友情链接">
      <template #extra>
        <div class="links-extra">
          <button class="btn btn-sm" @click="openExplain">
            <i class="bi bi-info-circle" aria-hidden="true" /> 友链说明
          </button>
          <button class="btn btn-sm btn-primary" @click="openApply">
            <i class="bi bi-plus-lg" aria-hidden="true" /> 申请友链
          </button>
        </div>
      </template>
    </SectionTitle>

    <div v-if="loading" class="is-loading">
      <span class="spinner" /> 加载中...
    </div>

    <div v-else-if="!groups.length" class="card card-pad">
      <EmptyState text="暂无友链分组" />
    </div>

    <div v-else>
      <div class="links-toolbar">
        <!-- 分组筛选：仅当存在多个分组时出现，"全部" 为默认视图 -->
        <div v-if="groups.length > 1" class="group-filter">
          <button
            class="group-chip"
            :class="{ 'is-active': activeGroupId === null }"
            @click="activeGroupId = null"
          >
            全部<span class="group-chip-count">{{ totalCount }}</span>
          </button>
          <button
            v-for="g in groups"
            :key="g.id"
            class="group-chip"
            :class="{ 'is-active': activeGroupId === g.id }"
            @click="activeGroupId = g.id"
          >
            {{ g.name }}<span class="group-chip-count">{{ g.list.length }}</span>
          </button>
        </div>

        <div class="links-summary">
          <span>共 <strong>{{ totalCount }}</strong> 个站点</span>
          <template v-if="checkedCount">
            <span class="links-summary-sep" aria-hidden="true" />
            <span class="links-summary-online">
              <i class="bi bi-broadcast" aria-hidden="true" /> {{ onlineCount }} 个在线
            </span>
          </template>
        </div>
      </div>

      <div v-for="g in filteredGroups" :key="g.id" class="link-group">
        <!-- 分组标题：单分组时省略，避免出现无意义的「默认分组」 -->
        <div v-if="groups.length > 1" class="link-group-head">
          <h3 class="link-group-name"><i class="bi bi-collection" />{{ g.name }}</h3>
          <span class="link-group-count">{{ g.list.length }} 个站点</span>
        </div>
        <p v-if="groups.length > 1 && g.description" class="link-group-desc">{{ g.description }}</p>

        <div class="link-grid">
          <a
            v-for="link in (g.list || [])"
            :key="link.id"
            :href="safeUrl(link.url) || undefined"
            target="_blank"
            rel="noopener"
            class="link-card"
          >
            <img :src="link.avatar || defaultAvatar" alt="" class="link-avatar" loading="lazy" />

            <div class="link-main">
              <div class="link-head">
                <span class="link-name" :title="link.nickname">{{ link.nickname }}</span>
                <span
                  v-if="link.checked"
                  class="link-status"
                  :class="link.online ? 'is-online' : 'is-offline'"
                  :title="link.online ? `响应耗时 ${link.responseTime}ms` : '链接检测未通过'"
                >
                  <i class="link-status-dot" />{{ link.online ? '在线' : '离线' }}
                </span>
              </div>

              <div class="link-url" :title="link.url">
                <i class="bi bi-link-45deg" aria-hidden="true" />{{ host(link.url) }}
              </div>

              <p class="link-desc">{{ link.description || '期待与您的互换链接' }}</p>
            </div>

            <i class="bi bi-arrow-up-right link-open" aria-hidden="true" />
          </a>
        </div>
      </div>
    </div>

    <!-- 友链说明弹窗 -->
    <div v-if="showExplain" class="dialog-mask" @click.self="closeExplain">
      <div class="dialog dialog-lg" role="dialog" aria-modal="true" aria-label="友链说明">
        <header class="dialog-head">
          <h3 class="dialog-title">友链说明</h3>
          <button type="button" class="dialog-close" title="关闭" aria-label="关闭" @click="closeExplain">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <div class="dialog-body explain-body">
          <div v-if="explainLoading" class="explain-loading">
            <span class="spinner" /> 加载中...
          </div>
          <div v-else-if="explain" class="markdown-body explain-content" v-html="explainHtml" />
          <EmptyState v-else text="暂无说明" />
        </div>

        <footer class="dialog-actions">
          <button class="btn btn-primary" @click="closeExplain">我知道了</button>
        </footer>
      </div>
    </div>

    <!-- 申请对话框 -->
    <div v-if="showApply" class="dialog-mask" @click.self="closeApply">
      <div class="dialog" role="dialog" aria-modal="true" aria-label="申请友情链接">
        <header class="dialog-head">
          <h3 class="dialog-title">申请友情链接</h3>
          <button type="button" class="dialog-close" title="关闭" aria-label="关闭" @click="closeApply">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <div class="dialog-body">
          <!-- 图标预览：上传后即时可见，避免用户填完才知道图挂了 -->
          <div class="apply-avatar">
            <img :src="form.avatar || defaultAvatar" class="apply-avatar-img" alt="站点图标预览" />
            <div class="apply-avatar-side">
              <div class="apply-avatar-actions">
                <button type="button" class="btn btn-sm" :disabled="uploading" @click="avatarFileRef?.click()">
                  <i class="bi bi-upload" aria-hidden="true" /> {{ uploading ? '上传中...' : '上传图片' }}
                </button>
                <button type="button" class="btn btn-sm" title="从附件库选择" @click="showLibrary = true">
                  <i class="bi bi-folder2-open" aria-hidden="true" /> 附件库
                </button>
                <button v-if="form.avatar" type="button" class="btn btn-sm btn-ghost" @click="form.avatar = ''">
                  移除
                </button>
              </div>
              <p class="apply-avatar-tip">建议 200×200 正方形图片，5MB 以内</p>
            </div>
            <input ref="avatarFileRef" type="file" accept="image/*" hidden @change="onAvatarChange" />
          </div>

          <div class="form-item">
            <label class="form-label">站点名称 *</label>
            <FormInput v-model="form.nickname" maxlength="40" placeholder="例如：某某的博客" />
          </div>
          <div class="form-item">
            <label class="form-label">站点链接 *</label>
            <FormInput v-model="form.url" placeholder="https://" />
          </div>
          <div class="form-item">
            <label class="form-label">站点描述</label>
            <FormInput v-model="form.description" maxlength="80" placeholder="一句话介绍你的站点" />
          </div>
          <div class="form-item">
            <label class="form-label">图标链接</label>
            <FormInput v-model="form.avatar" placeholder="也可以直接粘贴图片地址" />
          </div>

          <p class="apply-tip">
            <i class="bi bi-info-circle" aria-hidden="true" />
            提交后进入人工审核，通过后会展示在友链列表中。
          </p>
        </div>

        <footer class="dialog-actions">
          <button class="btn" @click="closeApply">取消</button>
          <button class="btn btn-primary" :disabled="submitting" @click="submit">
            {{ submitting ? '提交中...' : '提交申请' }}
          </button>
        </footer>
      </div>
    </div>

    <!-- 附件库：选一张已上传的图片作为站点头像 -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择站点头像"
      accept="image"
      @select="onLibrarySelect"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { ref, computed, onMounted, onUnmounted } from 'vue'
import SectionTitle from '@/components/SectionTitle.vue'
import EmptyState from '@/components/EmptyState.vue'
import { listLinks, listLinkGroups, createLink } from '@/api/links'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { call } from '@/api/request'
import { cache } from '@/utils/cache'
import { renderMarkdown } from '@/utils/markdown'
import { sanitizeHtml } from '@/utils/html'
import { safeUrl } from '@/utils/url'
import { toast } from '@/utils/toast'

// 优化后的默认头像（渐变背景+白色文字）
const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><defs><linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" style="stop-color:%23667eea;stop-opacity:1" /><stop offset="100%" style="stop-color:%23764ba2;stop-opacity:1" /></linearGradient></defs><circle cx="40" cy="40" r="40" fill="url(%23grad)"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" font-family="sans-serif" fill="white">友</text></svg>'

const loading = ref(false)
const groups = ref([])
const showApply = ref(false)
const submitting = ref(false)

// 打开申请弹窗：顺手把附件库关掉，避免上一次残留的打开状态「跟着弹出来」
function openApply() {
  showLibrary.value = false
  showApply.value = true
}
const showExplain = ref(false)
const explainLoading = ref(false)
const explain = ref(null)

const explainHtml = computed(() => {
  const c = explain.value?.content || ''
  if (!c) return ''
  // 含标签时按富文本渲染，但**必须走与文章详情同一套清洗** —— 这里是 v-html 出口，
  // 直接输出原始 HTML 会引入 XSS（说明内容来自「页面管理」里的 links 页面）
  return /<\/?[a-z][\s\S]*>/i.test(c) ? sanitizeHtml(c) : renderMarkdown(c)
})

async function openExplain() {
  showExplain.value = true
  if (explain.value || explainLoading.value) return
  explainLoading.value = true
  try {
    const res = await call('pages', 'one', {
      method: 'GET',
      params: { key: 'links', field: 'id,key,title,content' }
    })
    explain.value = res.data || null
  } catch {
    explain.value = null
  } finally {
    explainLoading.value = false
  }
}

const form = ref({
  nickname: '',
  url: '',
  description: '',
  avatar: '',
  group: 1,
  target: '_blank'
})

// ===== 站点头像：上传 / 附件库 =====
const avatarFileRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

async function onAvatarChange(event) {
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

    form.value.avatar = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// 从附件库选头像：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) form.value.avatar = url
}

// 链接状态缓存键与有效期（12 小时 = 720 分钟）
const STATUS_CACHE_KEY = 'mellow_links_status'
const STATUS_CACHE_MINUTES = 12 * 60

// 列表查询参数（检测状态时复用，保证前后两次返回的集合一致）
const linkQuery = {
  page: 1,
  limit: 200,
  where: { audit: 1 },
  field: 'id,nickname,url,description,avatar,group'
}

let statusChecking = false

// 读取状态缓存：{ [url]: { online, responseTime } }
function readStatusCache() {
  const data = cache.get(STATUS_CACHE_KEY)
  return data && typeof data === 'object' ? data : null
}

// 把检测结果合并到友链对象上（checked 为 true 才在卡片上展示状态）
function applyStatus(list, statusMap) {
  if (!statusMap) return
  list.forEach((link) => {
    const item = statusMap[link.url]
    if (!item) return
    link.online = item.online === true
    link.responseTime = Number(item.responseTime) || 0
    link.checked = true
  })
}

// 当前已渲染的全部友链（检测结果返回后就地刷新卡片状态）
function currentLinks() {
  return groups.value.flatMap((g) => g.list || [])
}

// 检测链接状态：结果写入 12 小时缓存，缓存内不再重复检测
async function checkLinksStatus() {
  if (statusChecking) return
  statusChecking = true
  try {
    const res = await listLinks({ ...linkQuery, status: true })
    const list = res.data?.data || []
    const statusMap = {}
    list.forEach((link) => {
      if (!link.url) return
      statusMap[link.url] = {
        online: link.online === true,
        responseTime: Number(link.responseTime) || 0
      }
    })
    if (!Object.keys(statusMap).length) return
    cache.set(STATUS_CACHE_KEY, statusMap, STATUS_CACHE_MINUTES)
    applyStatus(currentLinks(), statusMap)
  } catch {
    // 检测失败不打断列表展示
  } finally {
    statusChecking = false
  }
}

// 当前选中的分组，null 表示展示全部分组
const activeGroupId = ref(null)

const totalCount = computed(() => groups.value.reduce((n, g) => n + (g.list?.length || 0), 0))

// 已拿到检测结果的站点数 / 其中在线数：两者都只统计 checked 的卡片
const checkedCount = computed(() =>
  groups.value.reduce((n, g) => n + (g.list || []).filter((link) => link.checked).length, 0)
)
const onlineCount = computed(() =>
  groups.value.reduce((n, g) => n + (g.list || []).filter((link) => link.checked && link.online).length, 0)
)

function closeExplain() {
  showExplain.value = false
}

function closeApply() {
  showApply.value = false
}

// Esc 关闭弹窗；附件库打开时交给它自己处理，避免一次关掉两层
function onKeydown(e) {
  if (e.key !== 'Escape' || showLibrary.value) return
  if (showApply.value) {
    showApply.value = false
    return
  }
  if (showExplain.value) showExplain.value = false
}

const filteredGroups = computed(() => {
  if (activeGroupId.value === null) return groups.value
  return groups.value.filter((g) => g.id === activeGroupId.value)
})

// 分组排序权重：兼容 sort / order / weigh 三种常见字段，缺失时保持后端返回顺序
function groupWeight(g) {
  return Number(g?.sort ?? g?.order ?? g?.weigh ?? 0)
}

/**
 * 把扁平友链列表按分组归类。
 * 优先使用后端返回的分组表（顺序、名称最权威），再补齐分组表里没有的 id，
 * 兼容接口未返回分组信息时退化为「按友链自带的分组信息分组」。
 */
function buildGroups(list, groupList = []) {
  const map = new Map()
  // 先按后端分组表建桶（顺序最权威），保证展示顺序与后台一致
  ;[...groupList]
    .sort((a, b) => groupWeight(a) - groupWeight(b))
    .forEach((g) => {
      const id = g.id ?? 0
      map.set(String(id), {
        id,
        name: g.name || '未命名分组',
        description: g.description || '',
        list: []
      })
    })

  list.forEach((link) => {
    const g = link.result?.group || {}
    const id = g.id ?? 0
    const key = String(id)
    if (!map.has(key)) {
      map.set(key, { id, name: g.name || '默认分组', description: g.description || '', list: [] })
    }
    map.get(key).list.push(link)
  })

  // 空分组不展示；未归类的默认分组（id 为 0）统一排到最后
  return [...map.values()]
    .filter((g) => g.list.length)
    .sort((a, b) => (Number(a.id) === 0 ? 1 : 0) - (Number(b.id) === 0 ? 1 : 0))
}

async function load() {
  loading.value = true
  try {
    // 分组接口单独容错：拿不到分组表时仍按友链自带的分组信息展示
    const [linksRes, groupRes] = await Promise.all([
      listLinks(linkQuery),
      listLinkGroups().catch(() => null)
    ])
    const linksData = linksRes.data?.data || []
    const groupList = Array.isArray(groupRes?.data) ? groupRes.data : (groupRes?.data?.data || [])
    const statusCache = readStatusCache()
    // 优先使用 12 小时内的缓存，避免每次进入页面都触发后端检测
    applyStatus(linksData, statusCache)
    groups.value = buildGroups(linksData, groupList)
    // 无缓存或缓存未覆盖全部友链时，异步检测一次（不阻塞首屏）
    if (linksData.some((link) => link.url && !statusCache?.[link.url])) {
      checkLinksStatus()
    }
  } catch {
    groups.value = []
  } finally {
    loading.value = false
  }
}

function host(url) {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

async function submit() {
  if (!form.value.nickname || !form.value.url) {
    toast.warning('请填写完整信息')
    return
  }
  submitting.value = true
  try {
    await createLink(form.value)
    toast.success('申请已提交，等待审核')
    showApply.value = false
    form.value = { nickname: '', url: '', description: '', avatar: '', group: 1, target: '_blank' }
  } catch {} finally {
    submitting.value = false
  }
}

onMounted(() => {
  load()
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
/* --- 顶部工具行：分组筛选 + 站点统计 --- */
.links-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px 16px;
  margin-bottom: 20px;
}

.links-summary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}

.links-summary strong {
  font-weight: 600;
  color: var(--text-soft);
}

.links-summary-sep {
  width: 1px;
  height: 11px;
  background: var(--border);
}

.links-summary-online {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--success);
}

.links-summary-online .bi {
  font-size: 12px;
}

/* --- 分组筛选 --- */
.group-filter {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 0;
}

.group-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: color 0.2s, border-color 0.2s, background-color 0.2s;
}

.group-chip:hover {
  color: var(--primary);
  border-color: var(--primary);
}

.group-chip.is-active {
  background: var(--primary);
  border-color: var(--primary);
  color: #fff;
}

.group-chip-count {
  padding: 0 6px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.6;
  font-variant-numeric: tabular-nums;
}

.group-chip.is-active .group-chip-count {
  background: rgba(255, 255, 255, 0.24);
  color: #fff;
}

/* --- 分组标题 --- */
.link-group {
  margin-bottom: 24px;
}

.link-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-soft);
}

.link-group-name {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.link-group-name i {
  color: var(--primary);
  font-size: 14px;
}

.link-group-count {
  flex-shrink: 0;
  color: var(--text-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.link-group-desc {
  margin: -6px 0 12px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.6;
}

.link-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(272px, 1fr));
  gap: 14px;
}

/* --- 友链卡片 --- */
.link-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-height: 106px;
  padding: 15px 16px;
  text-decoration: none;
  color: inherit;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  transition: transform var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}

.link-card:hover {
  transform: translateY(-2px);
  border-color: var(--primary-soft);
  box-shadow: var(--shadow);
}

.link-avatar {
  width: 42px;
  height: 42px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
}

.link-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.link-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.link-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.link-url {
  font-size: 12px;
  color: var(--text-light);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.link-url .bi {
  margin-right: 3px;
  font-size: 11px;
}

/* --- 链接检测状态 --- */
.link-status {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.6;
  white-space: nowrap;
}

.link-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.link-status.is-online {
  color: var(--success);
}

.link-status.is-offline {
  color: var(--danger);
}

.link-desc {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.6;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 悬浮时右下角淡入的外链箭头 */
.link-open {
  position: absolute;
  right: 14px;
  bottom: 12px;
  font-size: 12px;
  color: var(--primary);
  opacity: 0;
  transform: translate(-2px, 2px);
  transition: opacity var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease);
}

.link-card:hover .link-open {
  opacity: 1;
  transform: none;
}

/* --- 加载与空状态 --- */
.is-loading {
  padding: 48px;
  text-align: center;
  color: var(--text-muted);
}

/* --- 弹窗遮罩与容器 --- */
.dialog-mask {
  position: fixed;
  inset: 0;
  /* 与全站弹窗层级一致，避免被吸顶导航压住 */
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(32, 28, 22, 0.5);
  backdrop-filter: blur(2px);
}

.links-extra {
  display: flex;
  align-items: center;
  gap: 10px;
}

.dialog {
  width: 100%;
  max-width: 460px;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.dialog-lg {
  max-width: 680px;
}
.dialog-head {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-soft);
}
.dialog-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.dialog-close:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.dialog-body {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 18px;
}
.dialog-actions {
  flex: 0 0 auto;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 18px;
  border-top: 1px solid var(--border-soft);
  background: var(--bg-card);
}

.explain-loading {
  padding: 32px;
  text-align: center;
  color: var(--text-muted);
}

/* 说明内容：滚动交给 .dialog-body，这里只负责排版 */
.explain-body {
  padding: 18px 20px;
}
.explain-content {
  font-size: 14px;
}

/* --- 申请弹窗：图标预览 --- */
.apply-avatar {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px;
  margin-bottom: 14px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg);
}
.apply-avatar-img {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
}
.apply-avatar-side {
  min-width: 0;
}
.apply-avatar-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.apply-avatar-tip {
  margin: 6px 0 0;
  font-size: 11px;
  color: var(--text-light);
}
.apply-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 14px 0 0;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--gold-wash);
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.6;
}
.apply-tip .bi {
  margin-top: 2px;
  color: var(--warning);
}

/* --- 响应式适配 --- */
@media (max-width: 640px) {
  .link-grid {
    grid-template-columns: 1fr;
  }
  .links-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    margin-bottom: 16px;
  }
  /* 分组较多时横向滑动，避免 chips 折行占掉半屏 */
  .group-filter {
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    scrollbar-width: none;
    -webkit-overflow-scrolling: touch;
  }
  .group-filter::-webkit-scrollbar {
    display: none;
  }
  .group-chip {
    flex: 0 0 auto;
    min-height: 34px;
  }
  .links-summary {
    justify-content: flex-end;
  }
  .link-group {
    margin-bottom: 20px;
  }
  .link-card {
    min-height: 0;
    padding: 14px;
  }
  .link-open {
    display: none;
  }

  /* 弹窗改成底部抽屉，拇指更容易够到按钮 */
  .dialog-mask {
    align-items: flex-end;
    padding: 0;
  }
  .dialog {
    max-width: none;
    max-height: 92vh;
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
  }
  .dialog-actions {
    padding-bottom: calc(12px + env(safe-area-inset-bottom));
  }
}
</style>