/**
 * 全局图片灯箱状态（配合 src/components/Lightbox.vue 使用）
 * 任意组件调用 openLightbox(images, index) 即可打开灯箱预览
 *
 * 入参支持两种写法，可以混着传：
 *   - 字符串：普通图片地址
 *   - 对象：{ image, video }，video 有值时灯箱里会多一个「播放实况」按钮
 *     （动态配图的 图片::视频 结构，见 utils/moment.js 的 parseMomentMedia）
 */
import { reactive } from 'vue'

export const lightboxState = reactive({
  visible: false,
  images: [],
  index: 0
})

/** 统一成 { image, video }，兼容 { src } / { url } 这类历史写法 */
function toMediaItem(item) {
  if (!item) return null
  if (typeof item === 'string') return { image: item, video: '' }
  if (typeof item !== 'object') return null
  return {
    image: item.image || item.src || item.url || '',
    video: item.video || ''
  }
}

export function openLightbox(images, index = 0) {
  const list = (Array.isArray(images) ? images : [images]).map(toMediaItem).filter((item) => item?.image)
  if (!list.length) return
  lightboxState.images = list
  lightboxState.index = Math.min(Math.max(index || 0, 0), list.length - 1)
  lightboxState.visible = true
}

export function closeLightbox() {
  lightboxState.visible = false
}
