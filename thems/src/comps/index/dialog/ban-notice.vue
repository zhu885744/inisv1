<!-- 封禁提醒弹窗组件 -->
<template>
  <Teleport to="body">
    <transition name="modal-fade" mode="out-in">
    <div
      v-if="state.visible"
      class="modal fade show"
      style="display: block;"
      tabindex="-1"
      aria-modal="true"
      role="dialog"
    >
      <div class="modal-dialog modal-dialog-centered" style="max-width: 480px;">
        <div class="modal-content border-0 shadow-lg" style="border-radius: 12px;">
          <!-- 头部 -->
          <div class="modal-header border-0 pb-0">
            <h5 class="modal-title d-flex align-items-center text-warning">
              <i class="bi bi-exclamation-triangle-fill me-2"></i>
              账号封禁通知
            </h5>
          </div>

          <!-- 主体 -->
          <div class="modal-body pt-3">
            <div class="text-center mb-3">
              <div class="d-inline-flex align-items-center justify-content-center bg-warning bg-opacity-10 rounded-circle mb-2"
                   style="width: 64px; height: 64px;">
                <i class="bi bi-shield-exclamation" style="font-size: 2rem; color: #e6a23c;"></i>
              </div>
              <p class="text-muted small mb-0">您的账号因违反社区规定已被封禁</p>
            </div>

            <div class="ban-info-box bg-light rounded p-3 mb-3">
              <div class="row g-2 small">
                <div class="col-12" v-if="state.banInfo.reason">
                  <span class="text-muted">封禁原因：</span>
                  <span class="text-dark">{{ state.banInfo.reason }}</span>
                </div>
                <div class="col-6">
                  <span class="text-muted">限制类型：</span>
                  <span v-for="bt in state.banTypes" :key="bt.bit" class="badge bg-light text-dark me-1">{{ bt.name }}</span>
                  <span v-if="!state.banTypes.length" class="badge bg-danger">全部限制</span>
                </div>
                <div class="col-6">
                  <span class="text-muted">封禁时长：</span>
                  <span v-if="state.banInfo.duration === 0" class="text-danger fw-bold">永久封禁</span>
                  <span v-else>{{ state.banInfo.duration }} 天</span>
                </div>
                <div class="col-6">
                  <span class="text-muted">违规次数：</span>
                  <span>第 {{ state.banInfo.violation_num || 1 }} 次</span>
                </div>
                <div class="col-6">
                  <span class="text-muted">剩余时间：</span>
                  <span v-if="state.banInfo.duration === 0 && state.banInfo.violation_num >= 5" class="text-danger">永久</span>
                  <span v-else-if="state.banInfo.duration === 0">-</span>
                  <span v-else-if="state.banInfo.expires_at">
                    {{ remainingText }}
                  </span>
                  <span v-else>-</span>
                </div>
              </div>
            </div>

            <div class="d-flex gap-2">
              <button v-if="state.banInfo.violation_num < 5" class="btn btn-warning flex-fill" @click="openAppeal">
                <i class="bi bi-journal-text me-1"></i> 提交申诉
              </button>
              <button class="btn btn-outline-secondary flex-fill" @click="hide">
                我知道了
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </transition>

  <!-- 遮罩层 -->
  <transition name="modal-fade" mode="out-in">
    <div v-if="state.visible" class="modal-backdrop fade show" @click="hide()"></div>
  </transition>
  </Teleport>
</template>

<script setup>
import { reactive, computed, onBeforeUnmount } from 'vue'

const emit = defineEmits(['appeal'])

const state = reactive({
  visible: false,
  banInfo: {
    reason: '',
    duration: 0,
    violation_num: 0,
    expires_at: 0,
    ban_types: '',
  },
  banTypes: [],
})

const remainingText = computed(() => {
  const remaining = state.banInfo.expires_at - Math.floor(Date.now() / 1000)
  if (remaining <= 0) return '已到期'
  const days = Math.floor(remaining / 86400)
  const hours = Math.floor((remaining % 86400) / 3600)
  if (days > 0) return `${days} 天 ${hours} 小时`
  return `${hours} 小时内`
})

const banTypeMap = [
  { bit: 1,  name: '限制登录' },
  { bit: 2,  name: '限制发文' },
  { bit: 4,  name: '限制评论' },
  { bit: 8,  name: '限制上传' },
  { bit: 16, name: '限制互动' },
]

function parseBanTypes(banTypeVal) {
  const raw = typeof banTypeVal === 'number' ? banTypeVal : (parseInt(banTypeVal) || 0)
  if (raw === 0 || raw === 31) return []
  return banTypeMap.filter(t => raw & t.bit)
}

function show(banData) {
  if (banData) {
    state.banInfo.reason = banData.reason || '违反社区规定'
    state.banInfo.duration = banData.duration || 0
    state.banInfo.violation_num = banData.violation_num || 1
    state.banInfo.expires_at = banData.expires_at || 0
    state.banTypes = parseBanTypes(banData.ban_type)
  }
  state.visible = true
  document.body.style.overflow = 'hidden'
}

function hide() {
  state.visible = false
  document.body.style.overflow = ''
}

function openAppeal() {
  hide()
  emit('appeal')
}

onBeforeUnmount(() => {
  document.body.style.overflow = ''
})

defineExpose({ show, hide })
</script>

<style scoped>
.ban-info-box {
  border-left: 3px solid #e6a23c;
}

.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s ease;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
</style>
