import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
/**
 * 编辑器视图模式：写作 / 预览 / 分屏 / 全屏
 *
 * 抽出原因：这三个开关互相牵连（全屏默认分屏、分屏必须可写、预览与分屏互斥），
 * 散在组件里很容易出现「切了预览但分屏还开着」这类状态矛盾。
 *
 * 使用方：components/MarkdownEditor.vue
 *
 * @param {object} options
 * @param {() => void} [options.focus]             切回写作态时把焦点还给正文
 * @param {() => boolean} [options.shouldIgnoreEscape] 是否暂不响应 Esc（如表情面板还开着）
 */
export function useEditorMode({ focus, shouldIgnoreEscape } = {}) {
  const mode = ref('write')
  const splitView = ref(false)
  const fullscreen = ref(false)

  // 预览与分屏都可以让正文区出现两栏，所以两者都要参与显隐判断
  const showEditor = computed(() => splitView.value || mode.value === 'write')
  const showPreview = computed(() => splitView.value || mode.value === 'preview')

  function focusEditor() {
    nextTick(() => focus?.())
  }

  /** 写作 <-> 预览；分屏下点预览 = 退出分屏，只看预览 */
  function toggleMode() {
    if (splitView.value) {
      splitView.value = false
      mode.value = 'preview'
      return
    }
    mode.value = mode.value === 'write' ? 'preview' : 'write'
    if (mode.value === 'write') focusEditor()
  }

  /** 全屏分屏：左写右预览，只在全屏下启用 */
  function toggleSplit() {
    splitView.value = !splitView.value
    // 分屏必须有一栏可写，否则和「只看预览」没区别
    if (splitView.value) mode.value = 'write'
    focusEditor()
  }

  let bodyOverflow = ''

  function lockBodyScroll() {
    bodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }

  function restoreBodyOverflow() {
    document.body.style.overflow = bodyOverflow || ''
  }

  /** 全屏 / 退出全屏；也可传明确的布尔值 */
  function toggleFullscreen(next) {
    const target = typeof next === 'boolean' ? next : !fullscreen.value
    if (target === fullscreen.value) return
    fullscreen.value = target
    if (target) {
      // 全屏默认就是「边写边预览」的分屏形态
      splitView.value = true
      mode.value = 'write'
      lockBodyScroll()
      focusEditor()
      return
    }
    splitView.value = false
    restoreBodyOverflow()
  }

  function onWindowKeydown(event) {
    if (event.key !== 'Escape' || !fullscreen.value) return
    // 面板打开时先关面板，再按一次才退出全屏
    if (shouldIgnoreEscape?.()) return
    event.preventDefault()
    toggleFullscreen(false)
  }

  onMounted(() => window.addEventListener('keydown', onWindowKeydown))
  onUnmounted(() => {
    window.removeEventListener('keydown', onWindowKeydown)
    restoreBodyOverflow()
  })

  return {
    mode,
    splitView,
    fullscreen,
    showEditor,
    showPreview,
    toggleMode,
    toggleSplit,
    toggleFullscreen,
    focusEditor
  }
}
