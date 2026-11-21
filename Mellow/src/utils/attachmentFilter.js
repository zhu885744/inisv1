/**
 * 附件类型判断工具（附件库弹窗使用）
 *
 * 不依赖构建别名，便于单独测试。
 */

// 常见图片 / 视频 / 音频扩展名（后端未返回 mime_type 时兜底判断）
//
// 图片只列浏览器能直接 <img> 显示的格式：heic / heif / tif / tiff 即使归到图片
// 类型也显示不出来，放进附件库缩略图会变破图。
export const IMAGE_EXTS = ['jpg', 'jpeg', 'jfif', 'png', 'apng', 'gif', 'webp', 'bmp', 'svg', 'ico', 'avif']
export const VIDEO_EXTS = ['mp4', 'webm', 'ogv', 'mov', 'm4v', 'avi', 'mkv', 'flv', 'wmv', '3gp', 'mpg', 'mpeg', 'm3u8', 'ts']
export const AUDIO_EXTS = ['mp3', 'wav', 'ogg', 'm4a', 'aac', 'flac', 'wma', 'opus']
const SIMPLE_EXTS = { image: IMAGE_EXTS, video: VIDEO_EXTS, audio: AUDIO_EXTS }

/** 取小写扩展名（去掉可能的点） */
export const extOf = (item) => String(item?.file_ext || '').replace(/^\./, '').toLowerCase()

/** 是否图片：优先看 mime_type，其次看扩展名 */
export const isImageFile = (item) => {
  const mime = String(item?.mime_type || '').toLowerCase()
  if (mime) return mime.startsWith('image/')
  return IMAGE_EXTS.includes(extOf(item))
}

/**
 * 判断附件是否符合 accept 规则
 *
 * 支持：
 *  - 空 / `*` / `all`      不限类型
 *  - `image`、`video`      类型简写（等价 image/*、video/*，并兼容常见图片扩展名）
 *  - `image/*`             前缀匹配 mime
 *  - `image/jpeg`          精确匹配 mime
 *  - `.png`                精确匹配扩展名
 *  - `png,pdf`             直接写扩展名
 *
 * 类型信息完全缺失（既无 mime_type 也无 file_ext）时不过滤，避免整页被隐藏。
 */
export function matchAccept(item, accept) {
  const rule = String(accept || '').toLowerCase().trim()
  if (!rule || rule === '*' || rule === 'all') return true

  const mime = String(item?.mime_type || '').toLowerCase()
  const ext = extOf(item)
  if (!mime && !ext) return true

  return rule.split(',').some((raw) => {
    const one = raw.trim()
    if (!one) return false
    if (one.endsWith('/*')) return mime.startsWith(one.slice(0, -1))
    if (one.startsWith('.')) return ext === one.slice(1)
    if (one.includes('/')) return mime === one
    // 简写：image / video / audio / pdf 等
    if (SIMPLE_EXTS[one]) return mime.startsWith(`${one}/`) || SIMPLE_EXTS[one].includes(ext)
    return mime.startsWith(`${one}/`) || ext === one
  })
}
