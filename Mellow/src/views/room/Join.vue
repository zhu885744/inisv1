<template>
  <div class="join-page">
    <div class="join-card">
      <template v-if="loading">
        <div class="join-loading">
          <span class="join-spinner" />
          <p>正在读取邀请…</p>
        </div>
      </template>

      <template v-else-if="error">
        <i class="bi bi-exclamation-circle join-icon" aria-hidden="true" />
        <h2>邀请不可用</h2>
        <p class="join-desc">{{ error }}</p>
        <RouterLink to="/chat" class="join-btn join-btn-ghost">返回消息列表</RouterLink>
      </template>

      <template v-else>
        <span class="join-avatar">
          <img v-if="room.avatar" :src="fullUrl(room.avatar)" alt="" loading="lazy" />
          <i v-else class="bi bi-people-fill" aria-hidden="true" />
        </span>

        <h2>{{ room.name }}</h2>
        <p class="join-meta">
          {{ room.member_count || 0 }} 位成员
          <template v-if="inviter?.nickname">
            · 由 {{ inviter.nickname }} 邀请
          </template>
        </p>
        <p v-if="room.description" class="join-desc">{{ room.description }}</p>

        <p v-if="Number(room.join_type) === 2" class="join-hint">
          <i class="bi bi-shield-lock" aria-hidden="true" />该群平时仅限邀请加入，凭这条链接可以直接进
        </p>

        <button
          type="button"
          class="join-btn join-btn-primary"
          :disabled="joining"
          @click="submit"
        >
          {{ joining ? '处理中…' : joined ? '进入群聊' : '加入群聊' }}
        </button>

        <RouterLink to="/chat" class="join-back">先去逛逛</RouterLink>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRoomInvite, joinRoomByCode } from '@/api/room'
import { toast } from '@/utils/toast'
import { getFullUrl } from '@/utils/emoji'

/**
 * 邀请落地页（/join/:code）
 *
 * 链接与二维码指向同一个地址，扫码 / 点链接都落到这里：
 *   - 未登录：路由守卫先跳登录，登录后带 redirect 回来；
 *   - 已在群里：直接「进入群聊」，不重复入群；
 *   - 过期 / 满员 / 群已解散：只展示原因，不给出会失败的按钮。
 *
 * 页面本身不展示群消息：没加入之前不拉历史（后端也会 403）。
 */

const route = useRoute()
const router = useRouter()

const code = computed(() => String(route.params.code || ''))

const loading = ref(true)
const joining = ref(false)
const error = ref('')
const room = ref({})
const inviter = ref(null)
const joined = ref(false)

function fullUrl(url) {
  return getFullUrl(url || '')
}

async function load() {
  loading.value = true
  error.value = ''

  try {
    const res = await getRoomInvite(code.value)
    if (res?.code !== 200) {
      error.value = res?.msg || '邀请链接不存在或已失效'
      return
    }

    room.value = res.data?.room || {}
    inviter.value = res.data?.inviter || null
    joined.value = !!res.data?.joined

    if (res.data?.expired) error.value = '邀请链接已过期，请让群管理重新生成'
    else if (res.data?.full) error.value = '该邀请的使用次数已用完'
  } catch {
    error.value = '邀请链接不存在或已失效'
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (joining.value) return

  const roomId = Number(room.value.id || 0)
  if (joined.value && roomId) {
    router.replace(`/chat/room/${roomId}`)
    return
  }

  joining.value = true
  try {
    const res = await joinRoomByCode(code.value)
    if (res?.code !== 200) return

    toast.success('已加入群聊')
    router.replace(`/chat/room/${res.data?.room_id || roomId}`)
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    joining.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.join-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  min-height: 100dvh;
  padding: 24px;
  background: var(--bg);
}
.join-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: min(360px, 100%);
  padding: 28px 22px 22px;
  text-align: center;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md, 0 18px 48px rgba(0, 0, 0, 0.08));
}
.join-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 20px 0;
  font-size: 12px;
  color: var(--text-light);
}
.join-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: join-spin 0.7s linear infinite;
}
@keyframes join-spin {
  to {
    transform: rotate(360deg);
  }
}
.join-icon {
  font-size: 30px;
  color: var(--text-light);
}
.join-avatar {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  margin-bottom: 12px;
  overflow: hidden;
  font-size: 26px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: 18px;
}
.join-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.join-card h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
}
.join-meta {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--text-light);
}
.join-desc {
  margin: 10px 0 0;
  font-size: 12.5px;
  line-height: 1.7;
  color: var(--text-light);
}
.join-hint {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 12px 0 0;
  padding: 8px 10px;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--text-light);
  text-align: left;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.join-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  margin-top: 18px;
  padding: 10px 0;
  font-size: 13.5px;
  border-radius: var(--radius);
}
.join-btn-primary {
  color: #fff;
  background: var(--primary);
}
.join-btn-primary:disabled {
  opacity: 0.6;
}
.join-btn-ghost {
  color: var(--text);
  background: var(--bg-muted);
}
.join-back {
  margin-top: 12px;
  font-size: 12px;
  color: var(--text-light);
}
</style>
