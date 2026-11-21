<template>
  <div class="admin-hub">
    <!-- 标签页：一个菜单入口下面挂多个子功能，避免侧边栏被拆成几十项 -->
    <nav class="admin-tabs" role="tablist">
      <RouterLink
        v-for="tab in tabs"
        :key="tab.to"
        :to="tab.to"
        class="admin-tab"
        :class="{ 'is-active': tab === activeTab }"
        role="tab"
        :aria-selected="tab === activeTab"
      >
        <i v-if="tab.icon" :class="tab.icon" aria-hidden="true" />
        <span>{{ tab.label }}</span>
      </RouterLink>
    </nav>

    <!-- 子路由：文章列表 / 用户认证 / 主题配色 … 都渲染在这里 -->
    <RouterView />
  </div>
</template>

<script setup>
/**
 * AdminHub - 后台「大功能 → 子功能」的容器
 *
 * 侧边栏只保留 16 个大功能入口，每个入口对应的页面用标签页承载子功能：
 *
 *   文章管理  文章列表 / 文章分类 / 撰写文章
 *   用户管理  用户列表 / 用户认证 / 认证类型 / 等级 / 经验 / 勋章
 *   内容管理  动态 / 公告 / 轮播 / 话题
 *   商城管理  商品 / 积分 / 装扮 / 签到
 *   独立页面  页面列表 / 撰写页面
 *   友链管理  友链列表 / 友链分组
 *   系统配置  网站设置 / 主题配色
 *   聊天管理  私信会话 / 聊天审核 / 群聊消息 / 禁言名单 / 群聊设置
 *   安全中心  内容审核 / 安全与限流 / 权限规则 / 权限分组 / 接口密钥 / IP 黑白名单 / QPS 预警 / 后台页面管理
 *
 * 为什么单独抽一个文件：8 个分区的标签结构完全一致，只有数据不同，
 * 用一个组件 + 配置表，比复制 8 份 UI 好维护；以后加子功能只改这里和路由。
 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const props = defineProps({
  // 分区 key，见下方 HUB_TABS
  section: { type: String, required: true }
})

// 标签配置：to 为跳转地址，match 用于「子页面高亮」（如编辑页高亮「撰写文章」）
const HUB_TABS = {
  article: [
    { label: '文章列表', to: '/admin/article', icon: 'bi bi-list-ul' },
    { label: '文章分类', to: '/admin/article/group', icon: 'bi bi-collection' },
    {
      label: '撰写文章',
      to: '/admin/article/write',
      icon: 'bi bi-pencil-square',
      match: ['/admin/article/write', '/admin/article/edit']
    }
  ],
  users: [
    { label: '用户列表', to: '/admin/users', icon: 'bi bi-people' },
    { label: '用户认证', to: '/admin/users/certifications', icon: 'bi bi-patch-check' },
    { label: '认证类型', to: '/admin/users/cert-types', icon: 'bi bi-sliders' },
    { label: '等级', to: '/admin/users/level', icon: 'bi bi-graph-up-arrow' },
    { label: '经验', to: '/admin/users/exp', icon: 'bi bi-star-half' },
    { label: '勋章', to: '/admin/users/badges', icon: 'bi bi-award' }
  ],
  content: [
    { label: '动态', to: '/admin/content/moments', icon: 'bi bi-chat-square-quote' },
    { label: '公告', to: '/admin/content/placard', icon: 'bi bi-megaphone' },
    { label: '轮播', to: '/admin/content/banner', icon: 'bi bi-images' },
    { label: '话题', to: '/admin/content/tags', icon: 'bi bi-tags' }
  ],
  goods: [
    { label: '商品', to: '/admin/goods', icon: 'bi bi-bag' },
    { label: '积分', to: '/admin/goods/integral', icon: 'bi bi-coin' },
    { label: '装扮', to: '/admin/goods/decoration', icon: 'bi bi-palette' },
    { label: '签到', to: '/admin/goods/checkin', icon: 'bi bi-calendar-check' }
  ],
  pages: [
    { label: '页面列表', to: '/admin/pages', icon: 'bi bi-window' },
    {
      label: '撰写页面',
      to: '/admin/pages/write',
      icon: 'bi bi-file-earmark-plus',
      match: ['/admin/pages/write', '/admin/pages/edit']
    }
  ],
  links: [
    { label: '友链列表', to: '/admin/links', icon: 'bi bi-link-45deg' },
    { label: '友链分组', to: '/admin/links/group', icon: 'bi bi-diagram-3' }
  ],
  system: [
    { label: '网站设置', to: '/admin/system', icon: 'bi bi-gear' },
    { label: '主题配色', to: '/admin/system/theme', icon: 'bi bi-palette2' }
  ],
  chat: [
    { label: '私信会话', to: '/admin/chat/conversations', icon: 'bi bi-envelope-paper' },
    { label: '聊天审核', to: '/admin/chat/moderation', icon: 'bi bi-shield-exclamation' },
    { label: '群聊消息', to: '/admin/chat/room', icon: 'bi bi-chat-square-text' },
    { label: '群聊管理', to: '/admin/chat/groups', icon: 'bi bi-people' },
    { label: '禁言名单', to: '/admin/chat/mutes', icon: 'bi bi-mic-mute' },
    { label: '群聊设置', to: '/admin/chat/settings', icon: 'bi bi-sliders' }
  ],
  security: [
    // 内容审核：敏感词库 + 云端机审（同一条链路的本地快筛与云端兜底）
    { label: '内容审核', to: '/admin/security/content', icon: 'bi bi-shield-check' },
    // 安全与限流：密钥校验 / QPS / 自动封禁 / 滑块
    { label: '安全与限流', to: '/admin/security/policy', icon: 'bi bi-speedometer' },
    { label: '权限规则', to: '/admin/security/rules', icon: 'bi bi-diagram-3' },
    { label: '权限分组', to: '/admin/security/group', icon: 'bi bi-shield-lock' },
    { label: '接口密钥', to: '/admin/security/keys', icon: 'bi bi-key' },
    { label: 'IP 黑名单', to: '/admin/security/ip-black', icon: 'bi bi-slash-circle' },
    { label: 'IP 白名单', to: '/admin/security/ip-white', icon: 'bi bi-check-circle' },
    { label: 'QPS 预警', to: '/admin/security/qps', icon: 'bi bi-speedometer2' },
    { label: '后台页面管理', to: '/admin/security/pages', icon: 'bi bi-layout-text-window' }
  ]
}

const route = useRoute()

const tabs = computed(() => HUB_TABS[props.section] || [])

/**
 * 当前激活的标签：按「最长前缀匹配」判定
 *
 * 例：/admin/article/edit/3 → 命中「撰写文章」的 match（/admin/article/edit），
 * 而不是更短的「文章列表」（/admin/article）。
 */
const activeTab = computed(() => {
  let best = null
  let bestLen = -1
  for (const tab of tabs.value) {
    for (const path of tab.match || [tab.to]) {
      if ((route.path === path || route.path.startsWith(`${path}/`)) && path.length > bestLen) {
        best = tab
        bestLen = path.length
      }
    }
  }
  return best
})
</script>

<style scoped>
.admin-hub {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
/* 标签条：与内容区卡片同色，横向可滚（手机端标签多时不换行） */
.admin-tabs {
  display: flex;
  gap: 4px;
  padding: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow-x: auto;
  scrollbar-width: none;
}
.admin-tabs::-webkit-scrollbar {
  display: none;
}
.admin-tab {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-sm);
  color: var(--text-soft);
  font-size: 14px;
  line-height: 1.5;
  text-decoration: none;
  white-space: nowrap;
  transition: color 0.15s, background-color 0.15s;
}
.admin-tab .bi {
  font-size: 14px;
}
.admin-tab:hover {
  background: var(--bg-muted);
  color: var(--primary);
}
.admin-tab.is-active {
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-weight: 600;
}
.admin-tab.is-active .bi {
  color: var(--primary);
}

@media (max-width: 768px) {
  .admin-tab {
    padding: 8px 11px;
    font-size: 13px;
  }
}
</style>
