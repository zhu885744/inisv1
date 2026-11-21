<template>
  <div class="auth-layout">
    <div class="auth-bg"></div>
    <div class="auth-card">
      <!-- 顶部品牌区：配了网站 LOGO 就只显示 LOGO（标题已在图片里，不再重复），否则显示站点标题 -->
      <router-link to="/" class="auth-logo" :class="{ 'auth-logo-image': site.avatar }">
        <img v-if="site.avatar" class="auth-logo-img" :src="site.avatar" :alt="site.title" loading="lazy" />
        <span v-else>{{ site.title }}</span>
      </router-link>
      <router-view />
      <div class="auth-footer">
        <router-link to="/"><i class="bi bi-arrow-left" /> 返回首页</router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { useSiteStore } from '@/stores/site'

const siteStore = useSiteStore()

const site = computed(() => {
  const c = siteStore.config || {}
  return {
    title: c.title || 'Mellow',
    // 网站 LOGO（后台「网站设置 → 网站 LOGO」）
    avatar: c.avatar || ''
  }
})

onMounted(() => {
  siteStore.load()
})
</script>

<style scoped>
.auth-layout {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-soft);
  position: relative;
  padding: 32px 16px;
}
.auth-bg {
  position: absolute;
  inset: 0;
  background-image:
    radial-gradient(at 20% 30%, var(--accent-wash) 0px, transparent 50%),
    radial-gradient(at 80% 70%, var(--accent-wash) 0px, transparent 50%);
  pointer-events: none;
}
.auth-card {
  width: 100%;
  max-width: 400px;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  border: 1px solid var(--border-soft);
  padding: 40px 32px;
  position: relative;
}
.auth-logo {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  text-align: center;
  font-family: var(--font-serif);
  font-size: 22px;
  letter-spacing: 2px;
  color: var(--text);
  margin-bottom: 32px;
}
/* 网站 LOGO：横向 LOGO 自动缩到卡片宽度内，不裁切 */
.auth-logo-img {
  max-width: 100%;
  max-height: 72px;
  object-fit: contain;
}
/* 只显示 LOGO 时（标题已隐藏）可以再大一点 */
.auth-logo-image .auth-logo-img {
  max-height: 88px;
}
.auth-footer {
  text-align: center;
  margin-top: 24px;
  font-size: 13px;
}
</style>