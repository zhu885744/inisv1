import { ref } from 'vue'

export function useEmoji() {
  const showEmojiPicker = ref(false)
  
  const toggleEmojiPicker = () => {
    showEmojiPicker.value = !showEmojiPicker.value
  }
  
  const insertEmoji = (emoji, textareaRef) => {
    if (textareaRef && textareaRef.value) {
      const textarea = textareaRef.value
      const start = textarea.selectionStart
      const end = textarea.selectionEnd
      const value = textarea.value
      textarea.value = value.substring(0, start) + emoji + value.substring(end)
      textarea.selectionStart = textarea.selectionEnd = start + emoji.length
      textarea.focus()
      return textarea.value
    }
    return null
  }
  
  const appendEmoji = (emoji, inputRef) => {
    if (inputRef && inputRef.value) {
      inputRef.value += emoji
      setTimeout(() => {
        const textarea = document.querySelector(`textarea[placeholder="请输入你的评论..."]`) || 
                         document.querySelector(`textarea[placeholder="请输入你的回复..."]`) ||
                         document.querySelector(`textarea[placeholder="请输入留言内容..."]`)
        if (textarea) {
          textarea.focus()
          textarea.setSelectionRange(textarea.value.length, textarea.value.length)
        }
      }, 50)
    }
  }
  
  return {
    showEmojiPicker,
    toggleEmojiPicker,
    insertEmoji,
    appendEmoji
  }
}

export function useMultipleEmoji() {
  const primaryEmoji = useEmoji()
  const secondaryEmoji = useEmoji()
  
  const togglePrimary = () => {
    primaryEmoji.showEmojiPicker.value = !primaryEmoji.showEmojiPicker.value
    secondaryEmoji.showEmojiPicker.value = false
  }
  
  const toggleSecondary = () => {
    secondaryEmoji.showEmojiPicker.value = !secondaryEmoji.showEmojiPicker.value
    primaryEmoji.showEmojiPicker.value = false
  }
  
  const closeAll = () => {
    primaryEmoji.showEmojiPicker.value = false
    secondaryEmoji.showEmojiPicker.value = false
  }
  
  return {
    primaryEmoji,
    secondaryEmoji,
    togglePrimary,
    toggleSecondary,
    closeAll
  }
}
