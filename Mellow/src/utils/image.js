/**
 * 图片地址工具：按显示尺寸取缩略图
 *
 * 站点图片走七牛图床（img.zhuxu.asia / img1.zhuxu.asia / img-zx.zhuxu.asia），
 * 支持 `?imageView2/1/w/<宽度>` 按宽度取图。实测：
 *   tx/2.png（2048px 默认头像）1.54MB → 88px 后 12KB
 *   zhublog/xxx.jpg（1280px 配图）229KB  → 272px 后 20KB
 * 而列表里这些图的显示尺寸只有 20~140px，直接拉原图等于把下载和
 * 解码（首屏 35 张图 / 35.8MP）全都白烧掉 —— 页面卡顿的主要来源。
 *
 * 只对已知图床域名追加参数：外链图、本地 assets、data URI 原样返回，
 * 避免把别人的图拼坏；域名换了只需改 THUMB_HOST_RE。
 */
const THUMB_HOST_RE = /^https?:\/\/img[\w-]*\.zhuxu\.asia\//i

/**
 * @param {string} url 原始图片地址（可以是相对路径 / data URI）
 * @param {number} width 期望的取图宽度（px），建议传「显示宽度 × 2」兼顾高 DPI 屏
 * @returns {string} 处理后的地址；不满足条件时原样返回
 */
export function thumbUrl(url, width) {
  const src = String(url || '')
  const px = Math.round(Number(width) || 0)
  if (!src || px <= 0) return src
  if (!THUMB_HOST_RE.test(src)) return src
  // 已经带过 imageView2（或调用方自己拼了处理参数）就不再叠加
  if (/[?&]imageView2=/i.test(src)) return src
  return `${src}${src.includes('?') ? '&' : '?'}imageView2/1/w/${px}`
}
