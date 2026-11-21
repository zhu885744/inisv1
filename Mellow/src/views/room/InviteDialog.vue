<template>
  <div class="inv-modal" @click.self="$emit('close')">
    <div class="inv-card">
      <header class="inv-head">
        <span>邀请加入「{{ roomName }}」</span>
        <button type="button" class="inv-icon-btn" title="关闭" @click="$emit('close')">
          <i class="bi bi-x-lg" aria-hidden="true" />
        </button>
      </header>

      <div class="inv-body">
        <p v-if="error" class="inv-error">
          <span>{{ error }}</span>
          <button type="button" class="inv-retry" @click="load">重试</button>
        </p>

        <div v-if="loading" class="inv-loading">
          <span class="inv-spinner" />加载中…
        </div>

        <template v-else-if="url">
          <div class="inv-qr">
            <canvas ref="canvasRef" class="inv-canvas" />
            <p class="inv-qr-tip">对方扫码后打开站点，登录即可加入</p>
          </div>

          <label class="inv-field">
            <span>邀请链接（永久有效）</span>
            <div class="inv-link">
              <FormInput :value="url" readonly @focus="$event.target.select()" />
              <button type="button" class="inv-copy" @click="copy">
                <i :class="copied ? 'bi bi-check2' : 'bi bi-clipboard'" aria-hidden="true" />
                {{ copied ? '已复制' : '复制' }}
              </button>
            </div>
          </label>

          <div class="inv-actions">
            <button type="button" class="inv-btn" @click="share">
              <i class="bi bi-send" aria-hidden="true" />发到输入框
            </button>
          </div>

          <p class="inv-tip">
            <i class="bi bi-shield-exclamation" aria-hidden="true" />
            这个链接不会过期，拿到的人随时都能进群，请只发给信任的人。
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { nextTick, onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import { createRoomInvite, listRoomInvites } from '@/api/room'
import { copyText } from '@/utils/helper'
import { toast } from '@/utils/toast'

/**
 * 邀请弹窗：链接 / 二维码
 *
 * 两者是同一份数据：服务端只发一个 code，这里拼成
 *   <站点>/join/<code>
 * 二维码画的就是这个 URL，所以「复制链接」和「扫码」等价。
 *
 * 链接是**永久**的（days=0 / max_uses=0），不提供有效期与次数设置：
 * 群邀请常被贴到主页、群公告、二维码海报上，带时限就等于过几天变死链；
 * 而且每次打开都新建一条，同一个人会在聊天记录里看到好几个不同链接。
 * 所以这里先复用该群已有的永久邀请，没有才新建一条。
 */

const props = defineProps({
  roomId: { type: [Number, String], required: true },
  roomName: { type: String, default: '' }
})

const emit = defineEmits(['close', 'share'])

const code = ref('')
const url = ref('')
const error = ref('')
const loading = ref(false)
const copied = ref(false)
const canvasRef = ref(null)

/** 从邀请列表里挑一条可用的永久邀请（0 = 不过期 / 不限次 / 未停用） */
function pickPermanent(list) {
  const rows = Array.isArray(list) ? list : []
  const row = rows.find(
    (item) =>
      Number(item.expire_time) === 0 && Number(item.max_uses) === 0 && Number(item.status) === 1
  )
  return row?.code ? String(row.code) : ''
}

async function load() {
  if (loading.value) return

  loading.value = true
  error.value = ''
  code.value = ''
  url.value = ''

  let target = ''

  // 先找已有的永久邀请：同一个群的链接应当稳定，不该每打开一次就换一个
  try {
    const res = await listRoomInvites(props.roomId)
    if (res?.code === 200) target = pickPermanent(res.data?.data)
  } catch {
    /* 查列表失败不影响新建，直接走下面的兜底 */
  }

  if (!target) {
    try {
      const res = await createRoomInvite(props.roomId, { days: 0, maxUses: 0 })
      if (res?.code === 200 && res.data?.code) {
        target = String(res.data.code)
      } else {
        error.value = res?.msg || '生成失败，请稍后重试'
      }
    } catch {
      error.value = '生成失败，请稍后重试'
    }
  }

  if (target) {
    code.value = target
    url.value = `${window.location.origin}/join/${target}`
    copied.value = false
  }

  // 先结束 loading：canvas 是 v-else-if 分支里的节点，
  // loading 还没落下去时 ref 是 null，二维码会画不出来
  loading.value = false
  if (!target) return

  await nextTick()
  await drawQr()
}

async function drawQr() {
  if (!canvasRef.value || !url.value) return

  try {
    await QRCode.toCanvas(canvasRef.value, url.value, {
      width: 168,
      margin: 1,
      errorCorrectionLevel: 'M',
      color: { dark: '#1f2328', light: '#ffffff' }
    })
  } catch {
    // 二维码画失败不影响复制链接，链接才是主路径
    error.value = ''
  }
}

async function copy() {
  const ok = await copyText(url.value)
  if (!ok) {
    toast.warning('复制失败，请手动选中复制')
    return
  }

  copied.value = true
  toast.success('邀请链接已复制')
  setTimeout(() => {
    copied.value = false
  }, 1600)
}

function share() {
  emit('share', url.value)
  emit('close')
}

onMounted(load)
</script>

<style scoped>
.inv-modal {
  position: fixed;
  inset: 0;
  z-index: 3600;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.3);
}
.inv-card {
  display: flex;
  flex-direction: column;
  width: min(360px, 100%);
  max-height: min(86vh, 620px);
  overflow: hidden;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md, 0 18px 48px rgba(0, 0, 0, 0.18));
}
.inv-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 12px 12px 16px;
  font-size: 13.5px;
  font-weight: 600;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  border-bottom: 1px solid var(--border-soft);
}
.inv-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  font-size: 13px;
  color: var(--text-light);
  border-radius: var(--radius);
}
.inv-icon-btn:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.inv-body {
  padding: 14px 16px 16px;
  overflow-y: auto;
}
.inv-field {
  display: block;
  margin-bottom: 12px;
}
.inv-field > span {
  display: block;
  margin-bottom: 5px;
  font-size: 12px;
  color: var(--text-light);
}
.inv-error {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin: 0 0 10px;
  font-size: 12px;
  color: var(--danger);
}
.inv-retry {
  flex-shrink: 0;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--danger);
  background: var(--bg-muted);
  border-radius: var(--radius-sm);
}
.inv-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 26px 0;
  font-size: 12px;
  color: var(--text-light);
}
.inv-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: inv-spin 0.7s linear infinite;
}
@keyframes inv-spin {
  to {
    transform: rotate(360deg);
  }
}
.inv-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 4px 0 12px;
}
.inv-canvas {
  width: 168px;
  height: 168px;
  padding: 8px;
  background: #fff;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.inv-qr-tip {
  margin: 0;
  font-size: 11px;
  color: var(--text-light);
  text-align: center;
}
.inv-link {
  display: flex;
  align-items: center;
  gap: 6px;
}
.inv-link input {
  flex: 1;
  min-width: 0;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  outline: none;
}
.inv-link input:focus {
  border-color: var(--primary);
}
.inv-copy {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: 8px 10px;
  font-size: 12px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: var(--radius);
}
.inv-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}
.inv-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 1;
  padding: 8px 0;
  font-size: 12.5px;
  color: var(--text);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.inv-btn:disabled {
  opacity: 0.6;
}
.inv-tip {
  display: flex;
  gap: 6px;
  margin: 0;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-light);
}
.inv-tip i {
  margin-top: 1px;
}
</style>
