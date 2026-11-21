<template>
  <RouterView />
  <!-- 全局图片灯箱：必须挂在 App 而不是 MainLayout。
       后台 / 用户中心 / 创作中心也有预览图片的入口（轮播图、举报证据、评论图片…），
       那些布局里没有灯箱时 openLightbox 只会把状态置为 visible，用户看不到也关不掉，
       等切回前台布局再挂载灯箱时就会「凭空」弹出来。 -->
  <Lightbox />
  <!-- 全局滑块验证：发送验证码被判定为频繁时，由 request.js 的拦截器唤起（见 utils/captchaGate） -->
  <SliderCaptcha />
  <!-- 全局「重新验证密码」弹窗：后台会话（scope=admin）失效时唤起（见 utils/adminSession），
       同样挂全局，后台任何页面（含守卫里的预校验）都能弹出来 -->
  <AdminUnlockDialog />
  <!-- 新消息悬浮通知（右下角）：同样挂全局，任何页面收到私信都能提示；
       是否弹出由 chat store 判断（后台标签页不弹、正在看的会话不弹），
       开关在消息页头部与用户中心「消息提醒」里。 -->
  <ChatNotice />
  <!-- 右侧悬浮按钮（主题切换 / 自定义按钮 / 返回顶部）
       同样挂全局：前台、登录页、后台、用户中心、创作中心、文档站都显示。
       两处例外：
         - 安装向导：未安装时 /api 会被整体拦住（412），拿不到站点配置，也不该出现在安装流程里；
         - 私信（/messages）：整屏聊天页，自己消息靠右排列，悬浮按钮正好压在自己的气泡上。 -->
  <FloatButtons v-if="route.name !== 'install' && route.name !== 'messages'" />
</template>

<script setup>
import { watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useSiteStore } from '@/stores/site'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import { applySiteAssets } from '@/utils/siteAssets'
import SliderCaptcha from '@/components/SliderCaptcha.vue'
import AdminUnlockDialog from '@/components/AdminUnlockDialog.vue'
import Lightbox from '@/components/Lightbox.vue'
import FloatButtons from '@/components/FloatButtons.vue'
import ChatNotice from '@/components/ChatNotice.vue'

const siteStore = useSiteStore()
const route = useRoute()
const userStore = useUserStore()
const chatStore = useChatStore()

// 私信实时通道：登录后建立 WebSocket（复用 /socket），退出登录时断开并清空未读。
// 挂在 App 而不是聊天页：不在聊天页时也要能收到新消息、更新用户中心的未读角标。
watch(
  () => userStore.isLogged,
  (logged) => {
    if (logged) chatStore.start()
    else chatStore.stop()
  },
  { immediate: true }
)

// 站点信息（网站标题 / 描述 / 关键词 / 图标）应用的唯一入口：
// 前台、登录页、后台、用户中心、创作中心都挂在 App 下，这里应用一次即可全覆盖。
// immediate 保证命中本地缓存时首屏也立即生效。
watch(
  () => siteStore.config,
  (config) => applySiteAssets(config || {}),
  { immediate: true, deep: true }
)

// 各布局也会调用（store 内部做并发去重与缓存），这里提前拉取让浏览器标签尽快用上配置。
// 安装向导页例外：未安装时 /api 会被后端整体拦住（412），拉配置只会弹无意义的错误提示，
// 安装页本身也不需要站点数据（装完是整页跳转，会重新加载）。
if (route.name !== 'install') {
  siteStore.load()
}
</script>
