<template>
  <div class="editor-status-bar" :class="{ 'is-fullscreen': fullscreen }">
    <div class="editor-status-left">
      <!-- 有本地草稿时优先显示保存状态，否则展示快捷键提示 -->
      <span v-if="savedText" class="editor-status-saved">
        <i class="bi bi-cloud-check" aria-hidden="true" />
        {{ savedText }}
      </span>
      <span v-else class="editor-status-hint">
        <i class="bi bi-keyboard" aria-hidden="true" />
        Ctrl + S 保存 · Ctrl + B 加粗 · Ctrl + K 链接
      </span>
    </div>

    <div class="editor-status-right">
      <span class="editor-status-item" :class="{ 'is-over': over }" :title="`已输入 ${count} 字，上限 ${maxLength}`">
        {{ count }}<em>/{{ maxLength }}</em>
      </span>
      <span class="editor-status-sep" aria-hidden="true" />
      <span class="editor-status-item editor-status-read">约 {{ readMinutes }} 分钟读完</span>

      <!-- 全屏时父级操作栏被遮住，这里补一个保存与退出口 -->
      <template v-if="fullscreen">
        <span class="editor-status-sep" aria-hidden="true" />
        <button type="button" class="btn btn-sm" @click="emit('exit-fullscreen')">退出全屏</button>
        <button type="button" class="btn btn-primary btn-sm" @click="emit('save')">保存</button>
      </template>
    </div>
  </div>
</template>

<script setup>
/**
 * 编辑器底部状态栏：字数 / 预估阅读时长 / 本地草稿状态 / 全屏内的保存出口
 *
 * 原来的字数统计是「右下角绝对定位」压在正文上的，正文一长就会挡住最后一两行；
 * 改成独立状态栏后既不遮字，也多了一行能放提示的位置。
 */
import { computed } from 'vue'

const props = defineProps({
  // 去掉空白后的字数
  count: { type: Number, default: 0 },
  maxLength: { type: Number, default: 5000 },
  // 本地草稿状态文案（如「本地草稿已存 14:32」）；空串表示还没写入过
  savedText: { type: String, default: '' },
  fullscreen: { type: Boolean, default: false }
})

const emit = defineEmits(['save', 'exit-fullscreen'])

const over = computed(() => props.maxLength > 0 && props.count > props.maxLength)
// 中文阅读速度按 400 字/分钟估算，不足 1 分钟显示 1 分钟
const readMinutes = computed(() => Math.max(1, Math.ceil(props.count / 400)))
</script>

<style scoped>
.editor-status-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 14px;
  border-top: 1px solid var(--border-soft);
  background: var(--bg-card);
  font-size: 12px;
  color: var(--text-muted);
  user-select: none;
}

.editor-status-left {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
}
.editor-status-hint,
.editor-status-saved {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.editor-status-saved {
  color: var(--success);
}
.editor-status-hint .bi,
.editor-status-saved .bi {
  font-size: 13px;
  flex-shrink: 0;
}

.editor-status-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.editor-status-item {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.editor-status-item em {
  font-style: normal;
  color: var(--text-light);
}
.editor-status-item.is-over,
.editor-status-item.is-over em {
  color: var(--danger);
}
.editor-status-sep {
  width: 1px;
  height: 12px;
  background: var(--border-soft);
}

/* 全屏：状态栏贴底，右侧多出保存按钮 */
.editor-status-bar.is-fullscreen {
  padding: 9px 18px;
  border-top: 1px solid var(--border);
  background: var(--bg-card);
}
.editor-status-bar.is-fullscreen .btn {
  height: var(--control-h-sm);
}

@media (max-width: 768px) {
  .editor-status-bar {
    padding: 6px 10px;
  }
  /* 窄屏放不下快捷键提示，只留草稿状态与计数 */
  .editor-status-hint {
    display: none;
  }
  .editor-status-right {
    gap: 6px;
  }
  .editor-status-read {
    display: none;
  }
}
</style>
