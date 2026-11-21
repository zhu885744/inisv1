/**
 * 实况图（Live Photo）：一张静态封面图 + 一段很短的视频
 *
 * 存储口径（**不改表结构**）：
 *   - 沿用动态 `images` 的「逗号分隔」写法，单项写成 `图片URL::视频URL`；
 *   - 没有 `::` 的项就是普通图片，老数据 / 老客户端完全不受影响；
 *   - 读取方用 parseLiveList / parseMomentImages 拿到图片地址，视频单独取。
 *
 * 之所以不做成独立字段：images 是唯一的配图来源（前台卡片、后台列表、回收站、
 * 详情页、附件迁移都读它），加字段意味着每处都要多一次判断，而 `::` 只是
 * 一项内的附加信息，任何只想要图片的地方 split 后自然就拿到了封面。
 */

/** 图片与视频之间的分隔符（URL 里不会出现 ::） */
export const LIVE_SEPARATOR = '::'

/** 视频扩展名：用于把附件库 / 上传结果归到「视频」而不是「图片」 */
const VIDEO_EXT_RE = /\.(mp4|webm|mov|m4v|ogv|mkv)(\?|#|$)/i

export function isVideoUrl(url) {
  return VIDEO_EXT_RE.test(String(url || '').trim())
}

/**
 * 解析单项：`图片` 或 `图片::视频`
 * @returns {{ image: string, video: string }}
 */
export function parseLiveItem(raw) {
  const value = String(raw ?? '').trim()
  if (!value) return { image: '', video: '' }

  const index = value.indexOf(LIVE_SEPARATOR)
  if (index < 0) return { image: value, video: '' }

  return {
    image: value.slice(0, index).trim(),
    video: value.slice(index + LIVE_SEPARATOR.length).trim()
  }
}

/** 序列化单项：没有视频时只存图片，保持与旧数据完全一致 */
export function serializeLiveItem(item) {
  const image = String(item?.image || '').trim()
  const video = String(item?.video || '').trim()
  if (!image) return ''
  return video ? `${image}${LIVE_SEPARATOR}${video}` : image
}

/**
 * 解析整串：兼容数组 / 逗号分隔 / 竖线分隔（历史写法）
 * @returns {{ image: string, video: string }[]}
 */
export function parseLiveList(raw) {
  if (!raw) return []
  const list = Array.isArray(raw) ? raw : String(raw).split(/[,|]/)
  return list.map(parseLiveItem).filter((item) => item.image)
}

/** 是否实况图（带视频） */
export function isLivePhoto(item) {
  return Boolean(item && item.video)
}

/** 序列化整串（动态发布 / 编辑时使用） */
export function serializeLiveList(items) {
  return (items || []).map(serializeLiveItem).filter(Boolean).join(',')
}
