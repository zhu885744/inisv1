import { onMounted, onUnmounted } from 'vue'

export function useClickOutside(elementRef, callback) {
  const handleClick = (event) => {
    if (elementRef.value && !elementRef.value.contains(event.target)) {
      callback(event)
    }
  }
  
  onMounted(() => {
    document.addEventListener('click', handleClick)
  })
  
  onUnmounted(() => {
    document.removeEventListener('click', handleClick)
  })
  
  return {
    handleClick
  }
}

export function useClickOutsideMultiple(elements, callback) {
  const handleClick = (event) => {
    let shouldCallback = true
    for (const element of elements) {
      if (element.value && element.value.contains(event.target)) {
        shouldCallback = false
        break
      }
    }
    if (shouldCallback) {
      callback(event)
    }
  }
  
  onMounted(() => {
    document.addEventListener('click', handleClick)
  })
  
  onUnmounted(() => {
    document.removeEventListener('click', handleClick)
  })
  
  return {
    handleClick
  }
}
