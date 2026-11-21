<template>
  <!-- 后台没开启任何第三方登录时整块不渲染，登录页视觉保持不变 -->
  <div v-if="platforms.length" class="social-login">
    <div class="social-divider">
      <span>{{ mode === 'register' ? '其他方式注册' : '其他方式登录' }}</span>
    </div>

    <div class="social-list">
      <button
        v-for="item in platforms"
        :key="item.platform"
        type="button"
        class="social-btn"
        :title="`使用 ${item.name}${mode === 'register' ? ' 注册' : ' 登录'}`"
        @click="go(item)"
      >
        <!-- 各平台图标：bi-tencent-qq / bi-github / bi-git -->
        <i class="social-icon" :class="platformIcon(item.platform)" />
        <span class="social-name">{{ item.name }}</span>
      </button>
    </div>

    <p v-if="hint" class="social-hint">{{ hint }}</p>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { loadOauthConfig, platformIcon, startOauth } from '@/utils/oauth'

const props = defineProps({
  // login：未绑定过的第三方账号会引导「先登录再绑定」；register：未绑定过则直接建号
  mode: { type: String, default: 'login' }
})

const route = useRoute()
const platforms = ref([])
const autoRegister = ref(true)

const hint = computed(() => {
  if (!platforms.value.length) return ''
  if (props.mode === 'register') {
    return autoRegister.value
      ? '首次使用将自动创建本站账号，昵称与头像取自第三方平台'
      : '后台未开启自动注册，第三方账号需先在本站注册后再绑定'
  }
  return autoRegister.value
    ? '未注册过的第三方账号将自动创建新账号'
    : '第三方账号需先登录本站账号后在「账号安全」中绑定'
})

function go(item) {
  startOauth(item, {
    mode: props.mode === 'register' ? 'login' : props.mode,
    redirect: route.query.redirect || '/'
  })
}

onMounted(async () => {
  const config = await loadOauthConfig()
  platforms.value = config.platforms
  autoRegister.value = config.autoRegister
})
</script>

<style scoped>
.social-login {
  margin-top: 4px;
}
.social-divider {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 18px 0 14px;
  font-size: 12px;
  color: var(--text-light);
}
.social-divider::before,
.social-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border-soft);
}
.social-divider span {
  white-space: nowrap;
}
.social-list {
  display: flex;
  gap: 10px;
}
.social-btn {
  /* 1 1 0：无论几个平台、屏幕多窄，都等分一行；min-width 允许文字省略而不是换行 */
  flex: 1 1 0;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 42px;
  padding: 0 12px;
  border: none;
  border-radius: 6px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 14px;
  font-family: inherit;
  cursor: pointer;
  transition: color 0.18s, background 0.18s;
}
.social-btn:hover {
  color: var(--primary);
  /* 无边框按钮靠底色区分：hover 时掺入一点主色，比直接用 primary-soft 更柔 */
  background: color-mix(in srgb, var(--primary) 9%, var(--bg-muted));
}
/* 平台图标（bi-tencent-qq / bi-github / bi-git）：跟随按钮文字色 */
.social-icon {
  flex-shrink: 0;
  font-size: 18px;
  line-height: 1;
}
/* 平台名过长时省略，保证三个按钮始终在同一行 */
.social-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.social-hint {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-light);
  text-align: center;
}

/* 窄屏：仍保持一行，只收紧间距与字号，不改成竖排 */
@media (max-width: 420px) {
  .social-list {
    gap: 8px;
  }
  .social-btn {
    gap: 6px;
    padding: 0 8px;
    font-size: 13px;
  }
  .social-icon {
    font-size: 16px;
  }
}
</style>
