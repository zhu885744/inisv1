/**
 * 图片上传前压缩为 WebP（浏览器端 canvas 实现，不依赖后端图像库）
 *
 * 取舍：
 * - 只处理常见位图（jpeg / png / webp / bmp / tiff）；
 * - 跳过 gif（动图会丢帧）、svg（矢量）等，避免破坏原语义；
 * - 浏览器不支持 WebP 编码时（toBlob 未返回 image/webp）原样返回，保证上传不失败。
 */

// 同时按 MIME 与扩展名识别：部分系统/软件导出的图片 file.type 可能为空或不标准（如 image/jpg）
const COMPRESSIBLE_TYPES = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp', 'image/bmp', 'image/tiff']
const COMPRESSIBLE_EXT = /\.(jpe?g|png|webp|bmp|tiff?)$/i

/** 是否为可压缩为 WebP 的位图 */
export function canCompressImage(file) {
  if (!file) return false
  const type = String(file.type || '').toLowerCase()
  if (COMPRESSIBLE_TYPES.includes(type)) return true
  return COMPRESSIBLE_EXT.test(String(file.name || ''))
}

/**
 * 压缩并转换为 WebP
 *
 * @param {File} file 原始图片
 * @param {{ quality?: number, maxEdge?: number }} options
 *        quality 0~1（默认 0.82）；maxEdge > 0 时按最长边等比缩小
 * @returns {Promise<File>} 转换后的 File；失败时返回原 File
 */
export async function compressImageToWebp(file, options = {}) {
  const { quality = 0.82, maxEdge = 0 } = options
  try {
    if (typeof createImageBitmap !== 'function' || typeof document === 'undefined') return file

    const bitmap = await createImageBitmap(file)
    let width = bitmap.width
    let height = bitmap.height
    if (maxEdge > 0 && Math.max(width, height) > maxEdge) {
      const scale = maxEdge / Math.max(width, height)
      width = Math.max(1, Math.round(width * scale))
      height = Math.max(1, Math.round(height * scale))
    }

    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) {
      bitmap.close?.()
      return file
    }
    // 不填底色：保留 PNG 的透明区域，WebP 支持 alpha
    ctx.drawImage(bitmap, 0, 0, width, height)
    bitmap.close?.()

    const blob = await new Promise((resolve) => canvas.toBlob(resolve, 'image/webp', quality))
    // 浏览器不支持 WebP 编码时会拿到 png 等：直接回退原文件
    if (!blob || blob.type !== 'image/webp') {
      console.warn('[webp] 浏览器未返回 WebP，回退原图：', file.name, blob && blob.type)
      return file
    }

    const baseName = String(file.name || 'image').replace(/\.[^.]+$/, '') || 'image'
    const output = new File([blob], `${baseName}.webp`, { type: 'image/webp', lastModified: Date.now() })
    console.info('[webp] 已转换：', file.name, `${file.size}B ->`, `${output.size}B`)
    return output
  } catch (error) {
    console.warn('[webp] 压缩失败，回退原图：', file.name, error)
    return file
  }
}


