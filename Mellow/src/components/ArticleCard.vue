<template>
  <article class="article-item">
    <!-- 封面图（左图右文布局） -->
    <div class="article-layout" :class="{ 'has-cover': cover }">
      <div v-if="cover" class="article-cover" @click="goDetail">
        <img :src="cover" :alt="article.title" loading="lazy" />
      </div>
      <div class="article-main">
        <h3 class="article-title">
          <router-link :to="`/archives/${article.id}`">{{ article.title }}</router-link>
          <span v-if="article.top" class="top-tag">置顶</span>
          <span v-else-if="article.status === 0" class="status-tag">草稿</span>
        </h3>
        <div class="article-meta">
          <span class="meta-date">{{ formatDate(article.publish_time || article.create_time) }}</span>
          <span v-if="group" class="meta-group">
            <router-link :to="`/category/${group.key || group.id}`">{{ group.name }}</router-link>
          </span>
          <span
            v-for="t in tagList"
            :key="t.id"
            class="meta-tag"
          >
            <router-link :to="`/topic/${t.id}`">#{{ t.name }}</router-link>
          </span>
        </div>
        <p class="article-abstract">
          <EmojiText :text="abstractText" :size="16" />
        </p>
        <div class="article-foot">
          <span v-if="author" class="meta-stat">
            <router-link :to="`/author/${author.id}`">{{ author.nickname }}</router-link>
          </span>
          <span class="meta-stat">{{ article.views || 0 }} 次阅读</span>
          <span v-if="showComment" class="meta-stat">{{ commentCount }} 评论</span>
        </div>
      </div>
    </div>
  </article>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { formatDate } from '@/utils/time'
import { truncate, stripArticleMarkup } from '@/utils/helper'
import EmojiText from '@/components/EmojiText.vue'
import WearingBadge from '@/components/WearingBadge.vue'
import { thumbUrl } from '@/utils/image'

const props = defineProps({
  article: { type: Object, required: true },
  abstractLimit: { type: Number, default: 140 }
})

const router = useRouter()

// 摘要：先洗掉 markdown / 媒体短代码（@[live] 等）再按长度截断，
// 否则列表里会露出 @live、封面图地址这类源码字样
const abstractText = computed(() =>
  truncate(stripArticleMarkup(props.article.abstract || props.article.text || props.article.content), props.abstractLimit)
)

// 封面
//
// 卡片里的封面框只有 160×108（见下方 .article-cover），直接拉原图等于把
// 上千像素的图下载 + 解码后缩到 1/10 显示 —— 列表一页十几张卡就是几十 MB 的浪费，
// 也是首屏卡顿的主要来源。这里按 2 倍图（320px）取缩略图，
// 图床不支持时 thumbUrl 会原样返回，不影响外链图与本地资源。
const COVER_WIDTH = 320

const cover = computed(() => {
  const c = props.article.covers
  if (!c) return ''
  const first = Array.isArray(c) ? c[0] || '' : String(c).split(',')[0] || ''
  return first ? thumbUrl(first.trim(), COVER_WIDTH) : ''
})

// 分组（后端返回对象）
const group = computed(() => {
  const g = props.article.result?.group
  if (!g) return null
  return Array.isArray(g) ? g[0] : g
})

// 话题（后端返回数组对象）
const tagList = computed(() => {
  const t = props.article.result?.tags
  if (Array.isArray(t)) return t.slice(0, 3)
  return []
})

// 作者
const author = computed(() => {
  return props.article.result?.author || null
})

// 评论数直接取文章返回里的 result.comment.count
// （comment/count 不支持 bind_id/bind_type 过滤，会返回全站评论总数）
const commentCount = computed(() => props.article.result?.comment?.count || 0)

// 文章 json.comment.show 为 2（不显示评论）时，列表里也不展示评论数
const showComment = computed(() => Number(props.article.result?.comment?.show ?? 1) !== 2)

function goDetail() {
  router.push(`/archives/${props.article.id}`)
}
</script>

<style scoped>
.article-item {
  padding: 20px 0;
  border-bottom: 1px dashed var(--border-soft);
}
.article-item:last-child {
  border-bottom: none;
}

.article-layout {
  display: flex;
  gap: 16px;
}
.article-layout.has-cover {
  align-items: flex-start;
}
.article-cover {
  width: 160px;
  height: 108px;
  flex-shrink: 0;
  border-radius: var(--radius);
  overflow: hidden;
  cursor: pointer;
  background: var(--bg-muted);
}
.article-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s;
}
.article-cover:hover img {
  transform: scale(1.05);
}
.article-main {
  flex: 1;
  min-width: 0;
}

.article-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
  line-height: 1.4;
}
.article-title a {
  color: var(--text);
}
.article-title a:hover {
  color: var(--primary);
}
.top-tag {
  display: inline-block;
  margin-left: 8px;
  padding: 1px 6px;
  font-size: 11px;
  border-radius: 3px;
  background: var(--accent-soft);
  color: var(--accent);
  vertical-align: middle;
}
.status-tag {
  display: inline-block;
  margin-left: 8px;
  padding: 1px 6px;
  font-size: 11px;
  border-radius: 3px;
  background: var(--bg-muted);
  color: var(--text-muted);
  vertical-align: middle;
}

.article-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 8px;
}
.meta-date { color: var(--primary-deep); }
.meta-group a,
.meta-tag a {
  color: var(--text-soft);
}
.meta-group a:hover,
.meta-tag a:hover {
  color: var(--primary);
}

.article-abstract {
  font-size: 13px;
  color: var(--text-soft);
  line-height: 1.7;
  margin: 0 0 8px;
}

.article-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: var(--text-muted);
}
.article-foot a {
  color: var(--text-muted);
}
.article-foot a:hover {
  color: var(--primary);
}
.meta-stat { color: var(--text-light); }

@media (max-width: 480px) {
  .article-layout.has-cover {
    flex-direction: column;
  }
  .article-cover {
    width: 100%;
    height: 160px;
  }
}
</style>
