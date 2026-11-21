<template>
  <article v-if="loading" class="docs-state">加载中…</article>

  <article v-else-if="!page" class="docs-state">
    <h1 class="docs-state-title">文档不存在</h1>
    <p class="docs-state-desc">这篇文档可能已下线，或需要登录后查看。</p>
  </article>

  <article v-else class="docs-page">
    <!-- 面包屑：目录层级 -->
    <nav v-if="page.breadcrumb?.length" class="docs-breadcrumb" aria-label="面包屑">
      <RouterLink :to="versionPath">文档</RouterLink>
      <template v-for="item in page.breadcrumb" :key="item.id">
        <i class="bi bi-chevron-right" aria-hidden="true" />
        <span>{{ item.name }}</span>
      </template>
    </nav>

    <div class="docs-page-layout">
      <div class="docs-page-body">
        <header class="docs-page-head">
          <h1 class="docs-page-title">{{ page.title }}</h1>
          <div class="docs-page-meta">
            <span>更新于 {{ formatDate(page.update_time || page.publish_time || page.create_time) }}</span>
            <span v-if="page.views">· {{ page.views }} 次阅读</span>
          </div>
        </header>

        <!-- 始终用 Markdown 原文 + 前端渲染器（服务端不认单换行，会出现「内容不换行」） -->
        <div ref="contentRef" class="markdown-body" v-html="contentHtml" @click="onContentClick" />

        <!-- 反馈 -->
        <div class="docs-feedback">
          <template v-if="!feedbackDone">
            <span class="docs-feedback-text">这篇文档有帮助吗？</span>
            <button type="button" class="btn btn-sm" @click="sendFeedback(1)">
              <i class="bi bi-hand-thumbs-up" aria-hidden="true" /> 有帮助
            </button>
            <button type="button" class="btn btn-sm" @click="sendFeedback(0)">
              <i class="bi bi-hand-thumbs-down" aria-hidden="true" /> 没帮助
            </button>
          </template>
          <span v-else class="docs-feedback-done">
            <i class="bi bi-check2-circle" aria-hidden="true" /> 感谢反馈
          </span>
        </div>

        <!-- 上一篇 / 下一篇 -->
        <nav v-if="page.prev || page.next" class="docs-pager" aria-label="翻页">
          <RouterLink v-if="page.prev?.path" :to="page.prev.path" class="docs-pager-item">
            <span class="docs-pager-label"><i class="bi bi-arrow-left" /> 上一篇</span>
            <span class="docs-pager-title">{{ page.prev.title }}</span>
          </RouterLink>
          <span v-else class="docs-pager-item is-empty" />

          <RouterLink v-if="page.next?.path" :to="page.next.path" class="docs-pager-item is-next">
            <span class="docs-pager-label">下一篇 <i class="bi bi-arrow-right" /></span>
            <span class="docs-pager-title">{{ page.next.title }}</span>
          </RouterLink>
        </nav>
      </div>

      <!-- 右侧页内目录 -->
      <aside v-if="toc.length" class="docs-toc">
        <div class="docs-toc-title">本页目录</div>
        <a
          v-for="item in toc"
          :key="item.id"
          :href="`#${item.id}`"
          class="docs-toc-link"
          :class="`is-lv${item.level}`"
        >{{ item.text }}</a>
      </aside>
    </div>

    <!-- 评论：由项目设置里的「启用评论」控制，后端返回 comment.{show,allow,count} -->
    <section v-if="commentShow" class="docs-comments">
      <div class="docs-comments-head">
        <h2 class="docs-comments-title">
          评论
          <span v-if="commentAllow" class="docs-comments-count">{{ commentsCount }} 条</span>
          <span v-else class="docs-comments-count">评论已关闭</span>
        </h2>
      </div>

      <CommentTree
        :bind-id="page.id"
        bind-type="docs"
        :author-id="page.uid || null"
        :allow-comment="commentAllow"
        :highlight-id="route.query.comment"
        @loaded="(n) => (commentsCount = n)"
      />
    </section>
  </article>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { call } from '@/api/request'
import CommentTree from '@/components/CommentTree.vue'
import { renderMarkdown } from '@/utils/markdown'
import { openLightbox } from '@/utils/lightbox'
import { formatDate } from '@/utils/time'

const props = defineProps({
  version: { type: String, default: 'latest' },
  group: { type: String, default: '' },
  slug: { type: String, default: '' }
})

const route = useRoute()

const page = ref(null)
const loading = ref(true)
const feedbackDone = ref(false)

const contentRef = ref(null)
const toc = ref([])

const commentsCount = ref(0)

// 评论开关：show 决定整块是否展示，allow 只隐藏发表框（与文章详情页口径一致）
const commentAllow = computed(() => Number(page.value?.comment?.allow ?? 1) !== 2)
const commentShow = computed(() => Number(page.value?.comment?.show ?? 1) !== 2)

const versionPath = computed(() => `/docs/${props.version}`)

// 始终用 Markdown 原文在前端渲染（项目内置 markdown-it，breaks 打开，单换行也会生效）
const contentHtml = computed(() => renderMarkdown(page.value?.content || ''))

// 渲染完成后：给 h2/h3 补锚点 id、生成 TOC、外链新窗口打开
async function afterRender() {
  await nextTick()

  const root = contentRef.value
  if (!root) {
    toc.value = []
    return
  }

  const headings = Array.from(root.querySelectorAll('h2, h3'))
  toc.value = headings.map((el, index) => {
    if (!el.id) el.id = `docs-heading-${index + 1}`
    return {
      level: Number(el.tagName.slice(1)),
      id: el.id,
      text: (el.textContent || '').trim()
    }
  })

  root.querySelectorAll('a[href^="http"]').forEach((link) => {
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
  })
}

/**
 * 点击正文里的图片打开灯箱
 *
 * 用事件委托：正文是 v-html 渲染的，没法给每个 img 单独绑事件。
 * 图片列表取正文内全部 img，灯箱里可以左右翻（与文章详情页 / 独立页面同一套交互）。
 */
function onContentClick(e) {
  const img = e.target.closest('img')
  if (!img || !img.src) return

  const srcs = Array.from(contentRef.value?.querySelectorAll('img') || [])
    .map((el) => el.src)
    .filter(Boolean)
  const index = srcs.indexOf(img.src)
  openLightbox(srcs, index < 0 ? 0 : index)
}

async function load() {
  if (!props.slug) {
    page.value = null
    loading.value = false
    return
  }

  loading.value = true
  feedbackDone.value = false
  commentsCount.value = 0

  try {
    const res = await call('docs', 'one', {
      method: 'GET',
      params: {
        version: props.version,
        group: props.group,
        slug: props.slug
      }
    })
    page.value = res?.data || null
    // 评论数先取详情里的口径，CommentTree 加载完会再覆盖一次
    commentsCount.value = Number(page.value?.comment?.count) || 0
  } catch {
    page.value = null
  } finally {
    loading.value = false
  }
}

async function sendFeedback(useful) {
  if (!page.value) return

  try {
    await call('docs', 'feedback', {
      method: 'POST',
      data: { docs_id: page.value.id, useful }
    })
  } finally {
    feedbackDone.value = true
  }
}

watch(
  () => [props.version, props.group, props.slug],
  async () => {
    await load()
    await afterRender()
  },
  { immediate: true }
)
</script>

<style scoped>
.docs-breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 16px;
  font-size: 12.5px;
  color: var(--text-light);
}
.docs-breadcrumb a {
  color: var(--text-muted);
  text-decoration: none;
}
.docs-breadcrumb a:hover {
  color: var(--primary);
}
.docs-breadcrumb i {
  font-size: 10px;
}

.docs-page-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 168px;
  gap: 28px;
  align-items: start;
}

/* 正文里的第一个 h1 与页面上方的文档标题重复，隐藏掉 */
.docs-page-body :deep(.markdown-body > h1:first-child) {
  display: none;
}
.docs-page-head {
  padding-bottom: 18px;
  margin-bottom: 22px;
  border-bottom: 1px solid var(--border-soft);
}
.docs-page-title {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 26px;
  font-weight: 700;
  line-height: 1.4;
  color: var(--text);
  letter-spacing: 0.5px;
}
.docs-page-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 10px;
  font-size: 12px;
  color: var(--text-light);
}

/* 右侧页内目录 */
.docs-toc {
  position: sticky;
  top: 88px;
  padding-left: 14px;
  border-left: 1px solid var(--border-soft);
}
.docs-toc-title {
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 1px;
}
.docs-toc-link {
  display: block;
  padding: 3px 0;
  font-size: 12.5px;
  line-height: 1.5;
  color: var(--text-light);
  text-decoration: none;
  transition: color 0.2s;
}
.docs-toc-link:hover {
  color: var(--primary);
}
.docs-toc-link.is-lv3 {
  padding-left: 12px;
}

/* 反馈 */
.docs-feedback {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-top: 40px;
  padding: 16px 18px;
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  font-size: 13px;
  color: var(--text-muted);
}
.docs-feedback-text {
  margin-right: 4px;
}
.docs-feedback-done {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--primary);
}

/* 上一篇 / 下一篇 */
.docs-pager {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-top: 28px;
}
.docs-pager-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  text-decoration: none;
  transition: border-color 0.2s;
}
.docs-pager-item:hover {
  border-color: var(--primary-soft);
}
.docs-pager-item.is-next {
  text-align: right;
}
.docs-pager-label {
  font-size: 12px;
  color: var(--text-light);
}
.docs-pager-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}

.docs-state {
  padding: 40px 0;
  text-align: center;
  color: var(--text-muted);
}
.docs-state-title {
  margin: 0 0 8px;
  font-size: 18px;
  color: var(--text);
}
.docs-state-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-light);
}

/* 评论 */
.docs-comments {
  padding-top: 22px;
  margin-top: 26px;
  border-top: 1px solid var(--border-soft);
}
.docs-comments-title {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin: 0 0 14px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.docs-comments-count {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-light);
}

@media (max-width: 1100px) {
  .docs-page-layout {
    grid-template-columns: 1fr;
  }
  .docs-toc {
    display: none;
  }
}
@media (max-width: 640px) {
  .docs-page-title {
    font-size: 22px;
  }
  .docs-page-head {
    padding-bottom: 14px;
    margin-bottom: 18px;
  }
  .docs-breadcrumb {
    gap: 4px;
    font-size: 12px;
  }
  .docs-feedback {
    padding: 12px 14px;
    margin-top: 28px;
  }
  .docs-pager {
    grid-template-columns: 1fr;
    margin-top: 20px;
  }
}
</style>
