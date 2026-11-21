/**
 * 附件（attachment）领域公共逻辑
 *
 * 后台附件管理的类型分组、图标、体积格式化统一在这里维护，
 * 列表与统计共用同一份扩展名分组，避免口径分叉。
 */

// 列表字段：不含 file_hash（无需展示），保留定位与展示需要的全部字段
// uploader_name / uploader_account 由后端 attachment/all 批量补齐（昵称，为空回退账号），
// 列表展示上传者昵称；字段里保留 uploader 关键字，后端才会把它们补进裁剪结果
export const ATTACHMENT_FIELD =
  'id,uuid,original_name,save_name,save_path,full_url,file_size,mime_type,file_ext,storage_driver,uploader_id,uploader_name,uploader_account,target_type,target_id,create_time,update_time,delete_time'

/** 扩展名分组：用于类型筛选（where.file_ext 的 $in / $nin）与类型图标 */
export const ATTACHMENT_EXT_GROUPS = {
  image: ['png', 'jpg', 'jpeg', 'jfif', 'apng', 'gif', 'webp', 'bmp', 'svg', 'ico', 'avif', 'heic', 'heif', 'tif', 'tiff'],
  video: ['mp4', 'webm', 'avi', 'mov', 'mkv', 'flv', 'wmv', 'm4v', '3gp', 'mpg', 'mpeg', 'ogv', 'm3u8', 'ts'],
  audio: ['mp3', 'wav', 'ogg', 'oga', 'flac', 'm4a', 'aac', 'wma', 'opus'],
  doc: ['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'md', 'csv', 'rtf'],
  archive: ['zip', 'rar', '7z', 'tar', 'gz', 'tgz', 'bz2', 'tbz2', 'xz', 'txz', 'cab']
}

/**
 * 浏览器能直接用 <img> 显示的图片格式
 *
 * heic / heif / tif / tiff 虽然属于图片类别（筛选、图标仍算图片），但 Chrome 等
 * 浏览器解不了，当缩略图会显示破图，所以预览判断单独用这份清单。
 */
export const PREVIEW_IMAGE_EXTS = ['jpg', 'jpeg', 'jfif', 'png', 'apng', 'gif', 'webp', 'bmp', 'svg', 'ico', 'avif']

/**
 * 浏览器能直接用 <video> 播放的视频格式
 *
 * avi / mkv / flv / wmv / rmvb 等属于视频类别（筛选、图标仍算视频），但 Chrome
 * 解不了容器或编码，放进播放器只会黑屏，所以预览判断单独用这份清单。
 */
export const PREVIEW_VIDEO_EXTS = ['mp4', 'm4v', 'webm', 'ogv']

/** 可直接播放的视频 MIME（扩展名缺失 / 不规范时兜底） */
const PREVIEW_VIDEO_MIMES = ['video/mp4', 'video/webm', 'video/ogg']

/** 类型下拉（value 与 ATTACHMENT_EXT_GROUPS 的 key 对应，other 需排除已知类型） */
export const ATTACHMENT_TYPE_OPTIONS = [
  { value: '', label: '全部类型' },
  { value: 'image', label: '图片' },
  { value: 'video', label: '视频' },
  { value: 'audio', label: '音频' },
  { value: 'doc', label: '文档' },
  { value: 'archive', label: '压缩包' },
  { value: 'other', label: '其他' }
]

/** 类型文案 */
const ATTACHMENT_TYPE_LABELS = {
  image: '图片',
  video: '视频',
  audio: '音频',
  doc: '文档',
  archive: '压缩包',
  other: '其他'
}

/** 类型 -> Bootstrap Icons */
const ATTACHMENT_TYPE_ICONS = {
  image: 'bi bi-file-earmark-image',
  video: 'bi bi-file-earmark-play',
  audio: 'bi bi-file-earmark-music',
  doc: 'bi bi-file-earmark-text',
  archive: 'bi bi-file-earmark-zip',
  other: 'bi bi-file-earmark'
}

/** 扩展名归一化（去点、转小写） */
export function normalizeExt(ext) {
  return String(ext ?? '')
    .trim()
    .replace(/^\./, '')
    .toLowerCase()
}

/** 判断附件类型：图片 / 视频 / 音频 / 文档 / 压缩包 / 其他 */
export function attachmentTypeOf(item) {
  const ext = normalizeExt(item?.file_ext)
  for (const [type, list] of Object.entries(ATTACHMENT_EXT_GROUPS)) {
    if (list.includes(ext)) return type
  }
  return 'other'
}

/** 类型文案 */
export function attachmentTypeLabel(item) {
  return ATTACHMENT_TYPE_LABELS[attachmentTypeOf(item)] || '其他'
}

/** 类型图标 */
export function attachmentIcon(item) {
  return ATTACHMENT_TYPE_ICONS[attachmentTypeOf(item)] || ATTACHMENT_TYPE_ICONS.other
}

/** 是否可直接用 <img> 预览（heic / tiff 浏览器解不了，按文件卡片展示） */
export function isPreviewableImage(item) {
  const mime = String(item?.mime_type || '').toLowerCase()
  if (mime.startsWith('image/')) return true
  return PREVIEW_IMAGE_EXTS.includes(normalizeExt(item?.file_ext))
}

/** 是否可直接用 <video> 预览（avi / mkv 等浏览器不支持，按文件卡片展示） */
export function isPreviewableVideo(item) {
  const mime = String(item?.mime_type || '')
    .toLowerCase()
    .split(';')[0]
    .trim()
  if (PREVIEW_VIDEO_MIMES.includes(mime)) return true
  return PREVIEW_VIDEO_EXTS.includes(normalizeExt(item?.file_ext))
}

/**
 * 构造类型筛选的 where 条件（JSON 字符串）
 *
 * 已知类型用 file_ext 的 $in；「其他」用 $nin 排除全部已知扩展名。
 * 无条件时返回 null（调用方不要传 where）。
 */
export function attachmentWhereJSON(type) {
  if (!type) return null
  if (type === 'other') {
    const known = Object.values(ATTACHMENT_EXT_GROUPS).flat()
    return JSON.stringify({ file_ext: { $nin: known } })
  }
  const list = ATTACHMENT_EXT_GROUPS[type]
  if (!list?.length) return null
  return JSON.stringify({ file_ext: { $in: list } })
}

/** 体积格式化：1024 进制，保留一位小数（字节不保留） */
export function formatFileSize(bytes) {
  const size = Number(bytes || 0)
  if (!Number.isFinite(size) || size <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(size) / Math.log(1024)), units.length - 1)
  const value = size / Math.pow(1024, index)
  return `${index === 0 ? Math.round(value) : value.toFixed(1)} ${units[index]}`
}

/**
 * 扩展名 → 正文短代码类型
 *
 * video / audio 用播放器渲染，其余（pdf、zip、docx…）一律走「附件下载卡」，
 * 所以这里只需要区分三种，不需要更细的类型。
 */
export function mediaKindOf(nameOrUrl) {
  const clean = String(nameOrUrl ?? '')
    .split('?')[0]
    .split('#')[0]
  const ext = normalizeExt(clean.slice(clean.lastIndexOf('.') + 1))
  if (ATTACHMENT_EXT_GROUPS.video.includes(ext)) return 'video'
  if (ATTACHMENT_EXT_GROUPS.audio.includes(ext)) return 'audio'
  return 'file'
}

/**
 * 生成正文短代码（编辑器插入用）
 *
 *   @[video](url "标题")
 *   @[audio](url "标题")
 *   @[file](url "文件名" "2.3 MB")
 *
 * kind 可显式指定 video / audio / file（编辑器的「视频 / 音频 / 附件」按钮按类型取用），
 * 不传则按扩展名自动判断。解析侧见 @/utils/markdown 的 media_shortcode 规则。
 */
export function attachmentMarker({ url, name, size } = {}, kind) {
  const src = String(url ?? '').trim()
  if (!src) return ''

  // 引号会截断短代码参数，统一换成单引号；换行直接抹掉
  const title = String(name ?? '')
    .replace(/["\r\n]/g, ' ')
    .trim()

  const type = ['video', 'audio', 'file'].includes(kind) ? kind : mediaKindOf(title || src)
  if (type === 'file') {
    const bytes = Number(size) || 0
    const meta = bytes > 0 ? formatFileSize(bytes) : ''
    return `@[file](${src} "${title}" "${meta}")`
  }

  return `@[${type}](${src}${title ? ` "${title}"` : ''})`
}

/** 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符 */
export function sanitizeAttachmentKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}
