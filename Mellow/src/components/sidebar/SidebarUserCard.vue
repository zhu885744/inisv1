<template>
  <div class="user-card card card-pad">
    <template v-if="userStore.isLogged">
      <AvatarFrame
        :src="user?.avatar"
        :name="user?.nickname"
        :frame="user?.json?.frame"
        :animated="frameRender.animated"
        :fallback="defaultAvatar"
        :size="'75px'"
        :frame-hole="frameRender.hole"
        :frame-offset-x="frameRender.offsetX"
        :frame-offset-y="frameRender.offsetY"
        :verified="user?.cert"
        alt="用户头像"
      />
      <div class="user-name">{{ user?.nickname }}</div>
      <div class="user-desc">{{ user?.description || '这个人很懒，什么都没留下' }}</div>
      <div class="user-actions">
        <RouterLink to="/user" class="btn btn-sm btn-ghost"><i class="bi bi-person" /> 用户中心</RouterLink>
        <RouterLink :to="`/author/${user?.id}`" class="btn btn-sm btn-ghost"><i class="bi bi-person-square" /> 用户主页</RouterLink>
        <RouterLink to="/manage" class="btn btn-sm btn-ghost"><i class="bi bi-grid-1x2" /> 创作中心</RouterLink>
        <RouterLink to="/manage/posts/write" class="btn btn-sm btn-ghost"><i class="bi bi-pencil-square" /> 写文章</RouterLink>
      </div>
      <div class="signin-entry">
        <RouterLink to="/checkin" class="btn btn-sm btn-primary btn-block">
          <i class="bi bi-calendar-check" /> 每日签到
        </RouterLink>
        <button class="btn btn-sm btn-ghost btn-block" @click="userStore.logout">
          <i class="bi bi-box-arrow-right" />退出登录
        </button>
      </div>
    </template>

    <template v-else>
      <AvatarFrame :src="touxiang" :size="'80px'" alt="未登录头像" />
      <div class="user-name">未登录</div>
      <div class="user-desc">登录后享受更多功能</div>
      <div class="user-actions">
        <RouterLink to="/auth/login" class="btn btn-sm btn-primary">登录</RouterLink>
        <RouterLink to="/auth/register" class="btn btn-sm">注册</RouterLink>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import { resolveFrameRender } from '@/utils/frameRender'
import AvatarFrame from '@/components/AvatarFrame.vue'
import touxiang from '@/assets/img/touxiang.webp'

/** 侧边栏「用户卡片」模块：登录态与未登录引导 */
const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

/** 头像框渲染参数：以装扮 payload 为准（users.json 上的副本只作兜底） */
const frameRender = computed(() => resolveFrameRender(user.value?.json))
</script>

<style scoped>
.user-card {
  text-align: center;
}
.user-name {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 4px;
}
.user-desc {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 16px;
  min-height: 18px;
}
.user-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}
.signin-entry {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
</style>
