<template>
  <article class="agreement-page">
    <header class="agreement-head">
      <h1 class="agreement-title">
        <i :class="icon" aria-hidden="true" /> {{ title }}
      </h1>
      <p class="agreement-desc">{{ desc }}</p>
    </header>

    <section class="card card-pad">
      <div class="agreement-body">{{ content }}</div>
    </section>

    <footer class="agreement-foot">
      <router-link class="agreement-foot-link" :to="otherPath">
        <i class="bi bi-file-earmark-text" /> 查看{{ otherTitle }}
      </router-link>
      <router-link class="agreement-foot-link" to="/">
        <i class="bi bi-house-door" /> 返回首页
      </router-link>
    </footer>
  </article>
</template>

<script setup>
/**
 * 用户协议 / 隐私政策（前台独立页面）
 *
 * 内容来自站点配置 Mellow_functions → auth_dialog_agreement（后台「网站设置 → 登录协议」）：
 *   - 用户协议：auth_dialog_agreement.user_agreement_content
 *   - 隐私协议：auth_dialog_agreement.privacy_agreement_content
 *
 * 与登录 / 注册弹窗里的协议弹窗（components/AuthAgreement.vue）读的是同一份配置，
 * 因此两处展示的内容始终一致；配置缺失时回退到内置默认文案，保证页面不会空白。
 *
 * 路由（见 router/index.js，必须注册在 /:key 兜底路由之前）：
 *   /agreement → type=user（用户协议）
 *   /privacy   → type=privacy（隐私协议）
 */
import { computed, onMounted } from 'vue'
import { useSiteStore } from '@/stores/site'

const props = defineProps({
  // user：用户协议；privacy：隐私协议
  type: { type: String, default: 'user' }
})

const siteStore = useSiteStore()

// 默认文案：与 AuthAgreement.vue、后台 SiteSettingsForm.vue 的默认值保持一致
const DEFAULT_USER =
  '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议。\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性。\n\n3. 用户行为规范\n请勿发布违法或侵犯他人权益的内容。'
const DEFAULT_PRIVACY =
  '隐私协议\n\n我们重视您的隐私。\n\n1. 收集的信息\n我们可能收集您的账户信息和使用数据。\n\n2. 信息使用\n用于提供和改进服务。\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。'

const isUser = computed(() => props.type !== 'privacy')
const title = computed(() => (isUser.value ? '用户协议' : '隐私协议'))
const icon = computed(() => (isUser.value ? 'bi bi-file-earmark-text' : 'bi bi-shield-check'))
const desc = computed(() =>
  isUser.value ? '使用本站服务前，请先阅读并同意以下条款' : '我们如何收集、使用与保护你的信息'
)
const otherPath = computed(() => (isUser.value ? '/privacy' : '/agreement'))
const otherTitle = computed(() => (isUser.value ? '隐私协议' : '用户协议'))

const content = computed(() => {
  const agreement = siteStore.config?.auth_dialog_agreement || {}
  const text = isUser.value
    ? agreement.user_agreement_content
    : agreement.privacy_agreement_content
  const value = String(text || '').trim()
  return value || (isUser.value ? DEFAULT_USER : DEFAULT_PRIVACY)
})

onMounted(() => {
  // 站点配置由 site store 统一去重缓存，重复进入本页不会重复请求
  siteStore.load()
})

// 浏览器标签标题不在这里写：页面标题由路由 meta.title 提供（见 router/index.js 的
// agreement / privacy），与站点名拼装由 utils/pageTitle.js 在每次路由切换时重算。
// 以前在这里写 document.title 且离开时不还原，导致跳到别的页面后标题仍是「用户协议 - 站点名」。
</script>

<style scoped>
.agreement-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.agreement-head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.agreement-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-family: var(--font-serif);
  font-size: 24px;
  font-weight: 700;
  color: var(--text);
}
.agreement-title > i {
  font-size: 20px;
  color: var(--primary);
}
.agreement-desc {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}
/* 协议正文：保留后台填写的换行与缩进，长文可读性优先 */
.agreement-body {
  font-size: 14px;
  line-height: 1.9;
  color: var(--text);
  white-space: pre-wrap;
  word-break: break-word;
}
.agreement-foot {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 13px;
}
.agreement-foot-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--text-muted);
}
.agreement-foot-link:hover {
  color: var(--primary);
}

@media (max-width: 640px) {
  .agreement-title {
    font-size: 20px;
  }
  .agreement-body {
    font-size: 13px;
  }
}
</style>
