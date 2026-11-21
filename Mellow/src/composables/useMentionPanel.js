import { ref } from 'vue'

// 面板与光标 / 按钮之间的间距
const PANEL_GAP = 6
// 面板最大高度 236 + 间距（与 MentionPicker 的样式保持一致）
const PANEL_HEIGHT = 246

/**
 * 编辑器 @ 提及面板的状态、定位与键盘接管
 *
 * 候选只含「我关注的人」（见 api/mention.js），插入纯文本「@昵称」；
 * 保存时后端会规范化为 @[昵称](/author/uid) 并给被 @ 的人发站内通知。
 *
 * 两条触发路径：
 *   ① 正文里直接输入 @ —— 弹面板并随输入过滤；
 *   ② 工具栏的 @ 按钮 —— 列出全部候选，插到光标处。
 *
 * 抽出来的原因：定位算法（视口坐标 + 下方空间不足向上翻）是编辑器里最容易写错
 * 又和正文逻辑无关的一段，独立后 MarkdownEditor 只需要转发一次输入事件。
 *
 * @param {object} options
 * @param {() => HTMLElement|null} options.getRoot   正文 contenteditable 根节点
 * @param {import('vue').Ref} options.panelRef       MentionPicker 的模板 ref
 */
export function useMentionPanel({ getRoot, panelRef }) {
  const open = ref(false)
  const keyword = ref('')
  // 本次面板是「输入 @ 触发」还是「工具栏按钮触发」：
  // 前者正文里已有一个 @关键词 需要被替换掉，后者不能回删字符
  const replaceMode = ref(false)
  // 视口坐标，配合 MentionPicker 的 position: fixed
  const position = ref({ left: 0, top: 0, flipTo: null })

  /** 由光标 / 按钮的矩形算出面板坐标（视口坐标），下方空间不足时向上翻 */
  function positionByRect(rect) {
    if (!rect) return
    const below = rect.bottom + PANEL_GAP
    position.value = {
      left: rect.left,
      top: below,
      flipTo: below + PANEL_HEIGHT > window.innerHeight ? rect.top - PANEL_GAP : null
    }
  }

  /** 光标前正处于「@关键词」时打开面板并按关键词过滤 */
  function syncFromSelection() {
    const root = getRoot()
    const selection = window.getSelection()

    if (!root || !selection || !selection.rangeCount) {
      open.value = false
      return
    }

    const range = selection.getRangeAt(0)
    const node = range.startContainer

    if (!range.collapsed || !root.contains(node) || node.nodeType !== Node.TEXT_NODE) {
      open.value = false
      return
    }

    // @ 前必须是行首或空白，避免把邮箱 a@b.com 当成 @ 提及
    const before = node.textContent.slice(0, range.startOffset)
    const matched = before.match(/(?:^|\s)@([^\s@]{0,32})$/)
    if (!matched) {
      open.value = false
      return
    }

    const rects = range.getClientRects()
    const caret = rects.length ? rects[rects.length - 1] : range.getBoundingClientRect()

    keyword.value = matched[1]
    replaceMode.value = true
    positionByRect(caret)
    open.value = true
  }

  /** 工具栏 @ 按钮：不做关键词过滤，面板贴在按钮下方 */
  function toggle(rect) {
    keyword.value = ''
    replaceMode.value = false
    if (rect) positionByRect(rect)
    open.value = !open.value
  }

  /** 面板打开时接管方向键 / 回车 / Esc；返回 true 表示该按键已被面板处理 */
  function onKeydown(event) {
    if (!open.value) return false

    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      panelRef.value?.move(event.key === 'ArrowDown' ? 1 : -1)
      return true
    }

    if (event.key === 'Enter') {
      // 候选为空时返回 false，让回车继续走编辑器的换行
      return panelRef.value?.selectActive() === true
    }

    if (event.key === 'Escape') {
      open.value = false
      return true
    }

    return false
  }

  /** 选中候选人：把「@关键词」替换成「@昵称 」 */
  function apply(user) {
    const typed = keyword.value
    // 输入触发时正文里已经有一个「@关键词」，长度是 keyword.length + 1（那个 @）；
    // 工具栏触发时正文里没有 @，这里必须是 0 —— 否则会把它前面一个字符误删
    const consume = replaceMode.value ? typed.length + 1 : 0

    replaceMode.value = false
    open.value = false
    if (!user?.nickname) return

    const root = getRoot()
    if (!root) return

    // 工具栏按钮都带 @mousedown.prevent，点按钮不会丢选区，光标仍在正文里
    root.focus()
    const selection = window.getSelection()
    if (!selection || !selection.rangeCount) return

    const range = selection.getRangeAt(0)
    const node = range.startContainer

    // 先把已输入的「@关键词」整段选中，下面的 execCommand 会用 @昵称 覆盖它
    if (consume > 0 && node.nodeType === Node.TEXT_NODE) {
      const start = Math.max(0, range.startOffset - consume)
      const replaceRange = document.createRange()
      replaceRange.setStart(node, start)
      replaceRange.setEnd(node, range.startOffset)
      selection.removeAllRanges()
      selection.addRange(replaceRange)
    }

    // 用 execCommand 插入纯文本：会触发 input 事件，交给正文的 onInput 同步源码
    document.execCommand('insertText', false, `@${user.nickname} `)
  }

  function close() {
    open.value = false
  }

  return { open, keyword, position, syncFromSelection, toggle, onKeydown, apply, close }
}
