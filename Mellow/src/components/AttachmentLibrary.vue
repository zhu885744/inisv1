<template>
  <Teleport to="body">
    <Transition name="attachment-fade">
      <div v-if="visible" class="attachment-mask" @click.self="close">
        <div class="attachment-panel" role="dialog" aria-modal="true" :aria-label="title">
          <!-- 头部 -->
          <header class="attachment-head">
            <h3 class="attachment-title">
              <i class="bi bi-folder2-open" aria-hidden="true" />
              {{ title }}
            </h3>
            <button type="button" class="attachment-close" title="关闭" @click="close">
              <i class="bi bi-x-lg" />
            </button>
          </header>

          <!-- 说明 + 刷新 -->
          <div class="attachment-toolbar">
            <span class="attachment-tip">
              {{ acceptTip }}
              <template v-if="multiple"> · 最多可选 {{ max }} 个</template>
            </span>
            <button type="button" class="btn btn-sm" :disabled="loading" @click="load()">
              <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loading }" /> 刷新
            </button>
          </div>

          <!-- 列表 -->
          <div class="attachment-body">
            <div v-if="loading" class="attachment-state">
              <span class="spinner" /> 加载中...
            </div>

            <div v-else-if="!filtered.length" class="attachment-state">
              <i class="bi bi-inbox attachment-state-icon" />
              <p>{{ items.length ? '当前页没有符合类型要求的附件' : '附件库还是空的，先上传一些吧' }}</p>
              <button v-if="items.length" type="button" class="btn btn-sm" @click="showAllTypes = true">
                显示全部类型
              </button>
            </div>

            <div v-else class="attachment-grid">
              <button
                v-for="item in filtered"
                :key="item.id"
                type="button"
                class="attachment-item"
                :class="{ 'is-picked': isPicked(item) }"
                :title="`${item.original_name}｜${formatSize(item.file_size)}｜${fromNow(item.create_time)}`"
                @click="toggle(item)"
              >
                <img v-if="isImage(item)" :src="item.full_url" :alt="item.original_name" loading="lazy" />
                <span v-else class="attachment-file">
                  <i class="bi bi-file-earmark-text" />
                  <em>{{ extOf(item) || 'file' }}</em>
                </span>
                <!-- 实况图：后端按命名反查到了配对视频，选中时会一起带出 -->
                <span v-if="item.live_video" class="attachment-live">LIVE</span>
                <span class="attachment-name">{{ item.original_name }}</span>
                <span v-if="isPicked(item)" class="attachment-check"><i class="bi bi-check-lg" /></span>
              </button>
            </div>
          </div>

          <!-- 底部分页 + 操作 -->
          <footer class="attachment-foot">
            <div class="attachment-pager">
              <Pagination
                v-if="total > pageSize"
                :current="page"
                :total="total"
                :page-size="pageSize"
                @update:current="changePage"
              />
              <span v-else class="attachment-total">共 {{ total }} 个附件</span>
            </div>
            <div class="attachment-actions">
              <span class="attachment-picked">{{ picked.length ? `已选 ${picked.length} 个` : '未选择' }}</span>
              <button type="button" class="btn btn-sm" @click="close">取消</button>
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="!picked.length"
                @click="confirm"
              >
                {{ multiple ? '应用所选' : '应用' }}
              </button>
            </div>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import Pagination from './Pagination.vue'
import { listMyAttachments } from '@/api/attachment'
import { extOf, isImageFile, matchAccept } from '@/utils/attachmentFilter'
import { fromNow } from '@/utils/time'
import { toast } from '@/utils/toast'

/**
 * 附件库弹窗
 *
 * 从「当前用户已上传的附件」中挑选文件并应用（可选多选），
 * 供动态 / 评论 / 文章编辑器 / 头像 / 收款码 / 站点 LOGO 等所有上传位置复用。
 *
 * 接口：GET /api/attachment/list（见 docs/API docs/attachment.md）
 */
const props = defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, default: '附件库' },
  // 类型限制：image（默认，仅图片）、all / *（全部）、或逗号分隔的 image/*、.png、image/jpeg
  accept: { type: String, default: 'image' },
  multiple: { type: Boolean, default: false },
  // 多选上限
  max: { type: Number, default: 9 },
  pageSize: { type: Number, default: 24 }
})

const emit = defineEmits(['update:visible', 'select'])

const loading = ref(false)
const items = ref([])
const total = ref(0)
const page = ref(1)
const picked = ref([])
// 兜底：类型筛选把本页全部隐藏时，允许临时显示全部类型
const showAllTypes = ref(false)

let bodyOverflow = ''

const isImage = isImageFile

// 按 accept 过滤当前页数据（无法识别类型时不过滤掉，避免「有数据却全被隐藏」）
const filtered = computed(() =>
  items.value.filter((item) => showAllTypes.value || matchAccept(item, props.accept))
)

const acceptTip = computed(() => {
  const accept = String(props.accept || '').toLowerCase()
  if (!accept || accept === '*' || accept === 'all' || showAllTypes.value) return '选择要应用的附件'
  // 类型简写 → 中文，避免所有场景都写死「图片」
  const LABELS = { image: '图片', video: '视频', audio: '音频', pdf: 'PDF', doc: '文档' }
  return `仅显示${LABELS[accept] || accept}类型的附件`
})

function isPicked(item) {
  return picked.value.some((i) => i.id === item.id)
}

function toggle(item) {
  if (props.multiple) {
    const index = picked.value.findIndex((i) => i.id === item.id)
    if (index > -1) {
      picked.value.splice(index, 1)
      return
    }
    if (picked.value.length >= props.max) {
      toast.warning(`最多只能选择 ${props.max} 个附件`)
      return
    }
    picked.value.push(item)
    return
  }
  // 单选：点击即应用
  picked.value = [item]
  confirm()
}

function confirm() {
  if (!picked.value.length) return
  // 统一返回 URL 数组（单选时取 [0]），同时给出完整附件对象便于扩展
  emit('select', picked.value.map((i) => i.full_url), picked.value.slice())
  close()
}

function close() {
  emit('update:visible', false)
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  // 关键：翻页时不能走 load() 的默认 reset —— 那会把 page 打回 1，看起来像分页没反应
  load(false)
}

/** 加载列表；reset 为 true 时回到第 1 页（打开弹窗 / 点「刷新」的语义） */
async function load(reset = true) {
  if (reset) page.value = 1
  loading.value = true
  try {
    const res = await listMyAttachments({ page: page.value, limit: props.pageSize })
    items.value = res.data?.data || []
    total.value = Number(res.data?.count) || 0
  } catch {
    items.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function lockBody(lock) {
  if (lock) {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = bodyOverflow || ''
  }
}

function onKeydown(e) {
  if (e.key === 'Escape') close()
}

watch(
  () => props.visible,
  (val) => {
    if (val) {
      picked.value = []
      showAllTypes.value = false
      lockBody(true)
      document.addEventListener('keydown', onKeydown)
      load()
    } else {
      lockBody(false)
      document.removeEventListener('keydown', onKeydown)
    }
  }
)

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  lockBody(false)

  // 宿主弹窗用 v-if 收起时会把本组件一起卸载，但宿主自己的 showLibrary 仍是 true；
  // 下次再打开宿主弹窗（重新挂载本组件）就会「跟着一起弹出来」。
  // 卸载时把 visible 同步回 false，从根上消掉这份残留状态。
  if (props.visible) emit('update:visible', false)
})

function formatSize(bytes) {
  const size = Number(bytes) || 0
  if (!size) return '未知大小'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}
</script>

<style scoped>
.attachment-mask {
  position: fixed;
  inset: 0;
  /*
   * 层级约定（详见 styles.css 末尾的说明）：
   *   全屏编辑器 2000 < 页面弹窗 3000 < 附件库 3500 < 确认弹窗 3600 < 灯箱 10000
   *
   * 附件库经常从「某个表单弹窗」里打开，所以必须比页面弹窗高一层；
   * 之前是 2900，被 AdminFormDialog（3000）和装扮编辑弹窗盖住，看起来在最下层。
   */
  z-index: 3500;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}
.attachment-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 760px;
  max-height: min(86vh, 720px);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

/* ---------- 头部 ---------- */
.attachment-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-soft);
}
.attachment-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.attachment-title .bi {
  color: var(--primary);
}
.attachment-close {
  width: 28px;
  height: 28px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.attachment-close:hover {
  background: var(--bg-muted);
  color: var(--primary);
}

/* ---------- 工具条 ---------- */
.attachment-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 18px;
  border-bottom: 1px solid var(--border-soft);
}
.attachment-tip {
  font-size: 12px;
  color: var(--text-muted);
}
.attachment-toolbar .bi.is-spin {
  display: inline-block;
  animation: attachment-spin 0.8s linear infinite;
}
@keyframes attachment-spin {
  to { transform: rotate(360deg); }
}

/* ---------- 内容 ---------- */
.attachment-body {
  flex: 1;
  min-height: 200px;
  overflow-y: auto;
  padding: 14px 18px;
}
.attachment-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 12px;
  font-size: 13px;
  color: var(--text-muted);
}
.attachment-state p {
  margin: 0;
}
.attachment-state-icon {
  font-size: 26px;
  color: var(--text-light);
}

.attachment-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
  gap: 10px;
}
.attachment-item {
  position: relative;
  padding: 0;
  aspect-ratio: 1 / 1;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-muted);
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
}
.attachment-item:hover {
  border-color: var(--primary-soft);
  transform: translateY(-1px);
}
.attachment-item.is-picked {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.attachment-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.attachment-file {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 100%;
  color: var(--text-muted);
}
.attachment-file .bi {
  font-size: 26px;
}
.attachment-file em {
  font-style: normal;
  font-size: 11px;
  text-transform: uppercase;
  color: var(--text-light);
}
.attachment-name {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 14px 6px 5px;
  font-size: 11px;
  line-height: 1.4;
  color: #fff;
  text-align: left;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.62));
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* 实况图角标：与右上角的选中圈分开摆，避免叠在一起 */
.attachment-live {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 0 4px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 1.6;
  pointer-events: none;
}
.attachment-check {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: #fff;
  background: var(--primary);
  border-radius: 50%;
}

/* ---------- 底部 ---------- */
.attachment-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 18px;
  border-top: 1px solid var(--border-soft);
}
.attachment-total {
  font-size: 12px;
  color: var(--text-muted);
}
.attachment-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.attachment-picked {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 过渡 ---------- */
.attachment-fade-enter-active,
.attachment-fade-leave-active {
  transition: opacity 0.2s ease;
}
.attachment-fade-enter-active .attachment-panel,
.attachment-fade-leave-active .attachment-panel {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.attachment-fade-enter-from,
.attachment-fade-leave-to {
  opacity: 0;
}
.attachment-fade-enter-from .attachment-panel,
.attachment-fade-leave-to .attachment-panel {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

@media (max-width: 640px) {
  .attachment-mask {
    padding: 0;
    align-items: flex-end;
  }
  .attachment-panel {
    max-width: none;
    max-height: 92vh;
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  }
  .attachment-foot {
    flex-direction: column;
    align-items: stretch;
  }
  .attachment-actions {
    justify-content: flex-end;
  }
  .attachment-grid {
    grid-template-columns: repeat(auto-fill, minmax(88px, 1fr));
  }
}
</style>
