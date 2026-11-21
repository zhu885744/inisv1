// 点赞/取消点赞的即时动画反馈工具
// 依赖全局样式中的 .like-pop / .like-pop-out / .like-burst 动画

// 图标弹跳（点赞）
export function popIcon(el) {
  if (!el) return
  el.classList.remove('like-pop')
  // 强制 reflow，保证连续点击能重新触发动画
  void el.offsetWidth
  el.classList.add('like-pop')
  window.setTimeout(() => el.classList.remove('like-pop'), 500)
}

// 图标收缩回位（取消点赞）
export function popOut(el) {
  if (!el) return
  el.classList.remove('like-pop-out')
  void el.offsetWidth
  el.classList.add('like-pop-out')
  window.setTimeout(() => el.classList.remove('like-pop-out'), 350)
}

// 从按钮位置飘出一颗心（仅点赞时）
export function burstHeart(container) {
  if (!container) return
  const heart = document.createElement('i')
  heart.className = 'like-burst bi bi-heart-fill'
  container.appendChild(heart)
  window.setTimeout(() => heart.remove(), 650)
}
