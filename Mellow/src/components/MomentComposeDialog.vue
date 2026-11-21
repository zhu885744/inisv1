<template>
  <Teleport to="body">
    <Transition name="moment-compose-fade">
      <div v-if="visible" class="moment-compose-mask" @click.self="onClose">
        <div class="moment-compose-box" role="dialog" aria-modal="true" aria-label="发布动态">
          <header class="moment-compose-head">
            <span class="moment-compose-title">
              <i class="bi bi-chat-square-dots" aria-hidden="true" />
              发布动态
            </span>
            <button class="moment-compose-close" type="button" aria-label="关闭" @click="onClose">
              <i class="bi bi-x-lg" />
            </button>
          </header>

          <!-- 表单本体在 components/MomentComposer.vue：
               正文 / 图片 / 附件库 / 位置 / 发布都在那边，这里只负责弹窗外壳 -->
          <div class="moment-compose-body">
            <MomentComposer ref="composerRef" inline-picker @published="onPublished" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
/**
 * 发布动态弹窗
 *
 * 所有「发动态」入口都走这里（首页、手机端底栏、我的页），动态列表页已下线。
 * 只是一层「弹窗外壳」：表单本体是 components/MomentComposer.vue，
 * 所以各入口的字段、校验、图片上传、@ 提及行为完全一致 —— 不会再出现多份实现各自演化的问题。
 */
import { ref } from 'vue'
import MomentComposer from './MomentComposer.vue'

defineProps({
  visible: { type: Boolean, default: false }
})

const emit = defineEmits(['update:visible', 'published'])

const composerRef = ref(null)

function onClose() {
  // 已填内容时二次确认，避免误点遮罩丢掉写了一半的动态
  if (composerRef.value?.hasDraft) {
    if (!window.confirm('关闭后已填写的内容会丢失，确认关闭？')) return
  }

  composerRef.value?.reset()
  emit('update:visible', false)
}

function onPublished() {
  emit('update:visible', false)
  emit('published')
}
</script>

<style scoped>
.moment-compose-mask {
  position: fixed;
  inset: 0;
  /* 与附件库同一层级：低于确认弹窗（3000），高于全屏编辑器（2000） */
  z-index: 2900;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.45);
}
.moment-compose-box {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 560px;
  /* 展开表情面板后内容会明显变高：用满视口（只留遮罩的 16px 内边距），
     让弹窗尽量跟着内容长高（真的要超出视口时才由 .moment-compose-body 滚动） */
  max-height: calc(100vh - 32px);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}

.moment-compose-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border-soft);
}
.moment-compose-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.moment-compose-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: background-color 0.15s, color 0.15s;
}
.moment-compose-close:hover {
  background: var(--bg-muted);
  color: var(--text);
}

.moment-compose-body {
  /* 按内容自然高度：flex: 0 1 auto（不用 flex: 1，那会把它拉到容器的最大高度）。
     内容少 → 弹窗就矮；内容多 → 长到 .moment-compose-box 的 max-height 之后由这里滚动。 */
  flex: 0 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 18px;
}

/* 过渡：与附件库 / 管理端弹窗一致 */
.moment-compose-fade-enter-active,
.moment-compose-fade-leave-active {
  transition: opacity 0.2s ease;
}
.moment-compose-fade-enter-active .moment-compose-box,
.moment-compose-fade-leave-active .moment-compose-box {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.moment-compose-fade-enter-from,
.moment-compose-fade-leave-to {
  opacity: 0;
}
.moment-compose-fade-enter-from .moment-compose-box,
.moment-compose-fade-leave-to .moment-compose-box {
  opacity: 0;
  transform: translateY(-10px) scale(0.98);
}

/* 手机端：贴底弹出，接近系统输入面板的体感 */
@media (max-width: 640px) {
  .moment-compose-mask {
    padding: 0;
    align-items: flex-end;
  }
  .moment-compose-box {
    max-width: none;
    max-height: 92vh;
    border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  }
  .moment-compose-head {
    padding: 12px 14px;
  }
  .moment-compose-body {
    padding: 12px;
  }
}
</style>
