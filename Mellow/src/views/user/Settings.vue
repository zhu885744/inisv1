<template>
  <div>
    <div class="profile-layout">
      <!-- 消息提醒：本机偏好，点开即生效，不走「保存设置」也不写后端 -->
      <div class="card card-pad">
        <h2 class="block-title">消息提醒</h2>

        <div class="form-item">
          <div class="switch-row">
            <div class="switch-text">
              <span class="switch-title">新消息提示音</span>
              <span class="hint">页面在前台时收到新私信播放一声「叮咚」，点开关即可试听。</span>
            </div>
            <button
              :class="['switch', { on: chatStore.alert.sound }]"
              role="switch"
              :aria-checked="chatStore.alert.sound"
              @click="chatStore.setAlert('sound', !chatStore.alert.sound)"
            >
              <span class="knob" />
            </button>
          </div>
        </div>

        <div class="form-item">
          <div class="switch-row">
            <div class="switch-text">
              <span class="switch-title">右下角悬浮通知</span>
              <span class="hint">在任意页面收到新私信时，右下角弹出提示卡片；正在看的会话不会打扰。</span>
            </div>
            <button
              :class="['switch', { on: chatStore.alert.toast }]"
              role="switch"
              :aria-checked="chatStore.alert.toast"
              @click="chatStore.setAlert('toast', !chatStore.alert.toast)"
            >
              <span class="knob" />
            </button>
          </div>
        </div>

        <div class="form-item">
          <div class="switch-row">
            <div class="switch-text">
              <span class="switch-title">桌面通知（窗口在后台时）</span>
              <span class="hint">{{ desktopHint }}</span>
            </div>
            <button
              :class="['switch', { on: desktopEnabled }]"
              role="switch"
              :aria-checked="desktopEnabled"
              :disabled="!desktopSupported"
              @click="toggleDesktop"
            >
              <span class="knob" />
            </button>
          </div>
        </div>

        <p class="hint">以上设置只保存在当前浏览器（不随账号同步），关闭后仍有未读角标与未读列表。</p>
      </div>

      <!-- 隐私设置 -->
      <div class="card card-pad">
        <h2 class="block-title">隐私设置</h2>

        <!-- 关注与粉丝列表可见范围 -->
        <div class="form-item">
          <label class="form-label">关注与粉丝列表</label>
          <div class="radio-col">
            <label class="radio-item" v-for="opt in followsOptions" :key="opt.value">
              <FormRadio :value="opt.value" v-model="privacy.follows" />
              <span>{{ opt.label }}</span>
            </label>
          </div>
          <p class="hint">控制其他用户能否查看你的关注列表与粉丝列表。</p>
        </div>

        <!-- 我的收藏 -->
        <div class="form-item">
          <div class="switch-row">
            <div class="switch-text">
              <span class="switch-title">公开我的收藏</span>
              <span class="hint">开启后，其他用户可查看你收藏的内容。</span>
            </div>
            <button :class="['switch', { on: privacy.collects === 1 }]" @click="privacy.collects = privacy.collects === 1 ? 0 : 1">
              <span class="knob" />
            </button>
          </div>
        </div>

        <!-- 我的点赞 -->
        <div class="form-item">
          <div class="switch-row">
            <div class="switch-text">
              <span class="switch-title">公开我的点赞</span>
              <span class="hint">开启后，其他用户可查看你点赞过的内容。</span>
            </div>
            <button :class="['switch', { on: privacy.likes === 1 }]" @click="privacy.likes = privacy.likes === 1 ? 0 : 1">
              <span class="knob" />
            </button>
          </div>
        </div>
      </div>

      <div class="btn-row">
        <button class="btn btn-primary" :disabled="loading" @click="save">
          {{ loading ? '保存中...' : '保存设置' }}
        </button>
        <button class="btn" :disabled="loading" @click="reset">重置</button>
      </div>

    </div>
  </div>
</template>

<script setup>
import FormRadio from '@/components/form/FormRadio.vue'
import { computed, reactive, ref, onMounted, watch } from 'vue'
import { useUserStore } from '@/stores/user'
import { useChatStore } from '@/stores/chat'
import { useDesktopNotice } from '@/composables/useDesktopNotice'
import { updateUser } from '@/api/users'
import { toast } from '@/utils/toast'

const userStore = useUserStore()
// 消息提醒开关直接读写 chat store（localStorage 持久化），不需要保存按钮
const chatStore = useChatStore()

// 桌面通知要走浏览器授权，申请必须发生在点击里，统一交给 composable
const {
  supported: desktopSupported,
  enabled: desktopEnabled,
  blocked: desktopBlocked,
  toggle: toggleDesktop
} = useDesktopNotice()

const desktopHint = computed(() => {
  if (!desktopSupported) return '当前环境不支持（浏览器要求 HTTPS）。'
  if (desktopBlocked.value) return '浏览器已拒绝通知权限，请在地址栏左侧的站点设置里允许通知。'
  if (desktopEnabled.value) return '窗口最小化或切到其他标签页时，也会弹出系统通知。'
  return '开启后，窗口在后台也能收到系统级通知（需浏览器授权）。'
})

const followsOptions = [
  { value: 'all', label: '全部公开' },
  { value: 'following', label: '仅公开关注列表' },
  { value: 'followers', label: '仅公开粉丝列表' },
  { value: 'none', label: '全部私密' }
]

const privacy = reactive({ follows: 'all', collects: 0, likes: 0 })
const loading = ref(false)

function loadFromStore() {
  const setting = userStore.user?.result?.setting
  if (setting && typeof setting === 'object') {
    if (setting.privacy) {
      privacy.follows = setting.privacy.follows || 'all'
      privacy.collects = setting.privacy.collects === 1 ? 1 : 0
      privacy.likes = setting.privacy.likes === 1 ? 1 : 0
    }
  }
}

const defaultSetting = () => ({
  privacy: { follows: 'all', collects: 0, likes: 0 }
})

function reset() {
  const d = defaultSetting()
  Object.assign(privacy, d.privacy)
  toast.info('已重置为默认设置')
}

async function save() {
  loading.value = true
  try {
    // 合并既有 json（保留 frame 等），再写入隐私设置
    const currentJson = (userStore.user?.json && typeof userStore.user.json === 'object')
      ? { ...userStore.user.json }
      : {}
    const mergedJson = {
      ...currentJson,
      privacy: {
        follows: privacy.follows,
        collects: privacy.collects,
        likes: privacy.likes
      }
    }

    await updateUser({
      id: userStore.user?.id,
      json: mergedJson
    })
    toast.success('设置已保存')
    // 同步最新用户信息（含新的 result.setting）
    await userStore.checkLoginState()
  } catch {
    // 错误已由拦截器提示
  } finally {
    loading.value = false
  }
}

onMounted(loadFromStore)
watch(
  () => userStore.user,
  (nu) => {
    if (nu && nu.id) loadFromStore()
  },
  { immediate: true }
)
</script>

<style scoped>
.block-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.profile-layout {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.form-item {
  margin-bottom: 18px;
}
.form-item:last-child {
  margin-bottom: 0;
}
.form-label {
  display: block;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
  margin-bottom: 10px;
}
.radio-col {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.radio-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
}
.hint {
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.6;
  margin-top: 8px;
}
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.switch-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.switch-title {
  font-size: 13px;
  color: var(--text-soft);
}
.switch {
  position: relative;
  width: 44px;
  height: 24px;
  border-radius: 999px;
  background: var(--border);
  border: none;
  cursor: pointer;
  transition: background 0.2s;
  flex-shrink: 0;
}
.switch.on {
  background: var(--primary);
}
.switch:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  /* 深色模式下用暖白滑块（见 styles.css 的 --switch-knob），浅色回落纯白 */
  background: var(--switch-knob, #fff);
  transition: transform 0.2s;
}
.switch.on .knob {
  transform: translateX(20px);
}
.btn-row {
  display: flex;
  gap: 8px;
}
</style>
