import { onBeforeUnmount, ref, watch } from 'vue'

/** localStorage 键前缀：按页面路径（或调用方指定的 key）区分，互不覆盖 */
const STORAGE_PREFIX = 'mellow:editor-draft:'
/** 超过这个长度不本地暂存：localStorage 单键上限约 5MB，超长正文写入会抛错 */
const MAX_LENGTH = 200000
/** 输入停顿多久落盘：既不会每个按键都写，也不会丢失太多内容 */
const DEFAULT_DELAY = 1200

/**
 * 编辑器本地草稿（防丢稿）
 *
 * 只在浏览器本地暂存，不上传服务器：
 *   - 输入停顿 1.2s 后写入 localStorage；
 *   - 打开编辑器时若发现「与加载内容不同」的草稿，返回给调用方提示用户是否恢复；
 *   - 表单真正保存成功后，调用方调用 clear() 清掉，避免下次误提示。
 *
 * 使用方：components/MarkdownEditor.vue
 *
 * @param {object} options
 * @param {() => string} options.key  草稿键（一般取路由路径）
 * @param {() => string} options.getValue  读取当前正文内容的函数
 * @param {number} [options.delay]    防抖时长
 */
export function useEditorDraft({ key, getValue, delay = DEFAULT_DELAY }) {
  // 打开时发现的既有草稿：{ content, updatedAt }，null 表示没有
  const restored = ref(null)
  // 最近一次自动保存的时间戳（毫秒），用于状态栏展示
  const lastSavedAt = ref(0)

  let timer = null
  // 初始化阶段（加载文章 / 恢复草稿）产生的变更不落盘，避免把旧草稿冲掉
  let muted = true

  function storageKey() {
    const name = String(key?.() || '')
    return name ? STORAGE_PREFIX + name : ''
  }

  function read() {
    const storeKey = storageKey()
    if (!storeKey) return null
    try {
      const raw = localStorage.getItem(storeKey)
      if (!raw) return null
      const data = JSON.parse(raw)
      const content = String(data?.content ?? '')
      if (!content.trim()) return null
      return { content, updatedAt: Number(data?.updatedAt) || 0 }
    } catch {
      // 数据损坏 / 隐私模式禁用 storage：当作没有草稿
      return null
    }
  }

  function write(value) {
    const storeKey = storageKey()
    if (!storeKey) return
    try {
      if (!value.trim()) {
        localStorage.removeItem(storeKey)
        lastSavedAt.value = 0
        return
      }
      if (value.length > MAX_LENGTH) return
      localStorage.setItem(storeKey, JSON.stringify({ content: value, updatedAt: Date.now() }))
      lastSavedAt.value = Date.now()
    } catch {
      // 超额 / 禁用：静默失败，绝不能打断写作
    }
  }

  /** 清掉本地草稿（保存成功或用户主动忽略） */
  function clear() {
    const storeKey = storageKey()
    if (storeKey) {
      try {
        localStorage.removeItem(storeKey)
      } catch {
        /* 忽略 */
      }
    }
    restored.value = null
    lastSavedAt.value = 0
  }

  /**
   * 启动：读取既有草稿
   * 与当前内容一致的草稿直接忽略（多半是上次保存成功但没清掉）
   */
  function init(currentValue = '') {
    const found = read()
    restored.value = found && found.content !== String(currentValue) ? found : null
    muted = true
  }

  /** 用户点「恢复草稿」：返回草稿正文，由调用方写回 modelValue */
  function consume() {
    const value = restored.value?.content || ''
    restored.value = null
    // 恢复后允许继续自动保存（恢复动作本身不写）
    muted = true
    return value
  }

  /** 用户点「忽略」：丢弃草稿并删掉本地副本 */
  function discard() {
    clear()
  }

  watch(
    getValue,
    (value) => {
      if (muted) {
        muted = false
        return
      }
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => write(String(value || '')), delay)
    }
  )

  onBeforeUnmount(() => {
    if (timer) clearTimeout(timer)
  })

  return { restored, lastSavedAt, init, clear, consume, discard }
}
