import { call } from './request'
import { getConfig } from './config'
import { canCompressImage, compressImageToWebp } from '@/utils/imageCompress'

// ids 归一化：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
const toIdsString = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)
    .join(',')

// PUT/POST 走 JSON body，直接传数组即可（utils.Unity.Ids 会解析）
const toIdsArray = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)

// 我的附件库（当前用户上传的附件，分页）
// 文档：docs/API docs/attachment.md · GET /api/attachment/list
export const listMyAttachments = (params = {}) =>
  call('attachment', 'list', {
    method: 'GET',
    params: {
      page: 1,
      limit: 24,
      field: 'id,uuid,original_name,full_url,file_size,mime_type,file_ext,create_time',
      ...params
    }
  })

/**
 * 读取「图片上传压缩」模式（每次上传读一次，不缓存）
 *
 * 不走 siteStore（30 分钟本地缓存会让人误以为改了配置没生效），也不做模块级缓存：
 * 上传频率远低于列表查询，多这一次 config/one 请求换「改完立即生效」是值得的。
 * 配置来自 Mellow_functions.upload_webp_mode（off / force），读取失败按 off 处理。
 */
// config/one 的 json 可能是已解码对象，也可能是 JSON 字符串（后端 ParseConfigJson 两种都兼容），
// 这里同样两种都处理，避免因形态不同导致配置读成 undefined
function parseConfigJson(raw) {
  if (!raw) return {}
  if (typeof raw === 'object') return raw
  if (typeof raw === 'string') {
    try {
      const decoded = JSON.parse(raw)
      return decoded && typeof decoded === 'object' ? decoded : {}
    } catch {
      return {}
    }
  }
  return {}
}

async function getUploadWebpMode() {
  try {
    const res = await getConfig('Mellow_functions', { silent: true })
    const config = parseConfigJson(res?.data?.json)
    const mode = config.upload_webp_mode === 'force' ? 'force' : 'off'
    console.info('[webp] 上传压缩模式：', mode)
    return mode
  } catch (error) {
    console.warn('[webp] 读取压缩配置失败，按关闭处理：', error)
    return 'off'
  }
}

// FormData 里的值在个别环境可能不是 File 实例（跨 iframe / 粘贴产生的 Blob），用鸭子类型判断更稳
function isFileLike(value) {
  return Boolean(value)
    && typeof value === 'object'
    && typeof value.type === 'string'
    && typeof value.size === 'number'
}

/**
 * 是否安卓实况照片（Motion Photo：JPEG 尾巴里追加了一段 MP4）
 *
 * 只读头部 256KB 找 XMP 标记，覆盖 Pixel / 小米 / 三星 / 华为等主流机型。
 * 这类文件**不能走 canvas 重编码**：WebP 转换只保留画面，内嵌的视频会被丢掉，
 * 而后端 attachment/batch 正是靠原始字节把视频拆出来的
 * （见 app/api/controller/motion_photo.go）。
 *
 * 识别不出时按普通图片处理，最坏结果只是压缩后没有实况效果。
 */
async function isMotionPhoto(file) {
  if (!isFileLike(file) || !/^image\//i.test(file.type || '')) return false
  if (file.size < 4096) return false
  try {
    const head = await file.slice(0, 256 * 1024).text()
    // 两类写法都要认：
    //  - MicroVideoOffset / MotionPhoto_Data / GCamera:MicroVideo：Pixel、小米、三星等，
    //    XMP 里带视频偏移；
    //  - MotionPhoto="1"：华为（鸿蒙）动态照片只有这个标记、没有偏移字段，
    //    视频固定在 JPEG 之后，由后端按 MP4 的 ftyp 盒子扫描定位。
    return /MicroVideoOffset|MotionPhoto_Data|GCamera:MicroVideo|MotionPhoto\s*[:=]/i.test(head)
  } catch {
    return false
  }
}

/**
 * 按后台「图片上传压缩」配置，把 FormData 里的图片先转成 WebP
 *
 * - off：原样上传
 * - force：所有可压缩图片一律转 WebP
 * 读取失败按 off 处理，绝不阻断上传。
 */
async function prepareUploadFormData(formData) {
  // 兼容两种字段名：批量上传用 'files'（多文件），头像 / 封面等单文件场景用 'file'
  // （后端 batch 优先取 files，取不到再取 file，两种都接受）
  const single = formData.getAll('file')
  const multiple = formData.getAll('files')
  const all = [...single, ...multiple]
  if (!all.length) return formData

  if ((await getUploadWebpMode()) !== 'force') return formData

  // 没有可压缩的图片时避免多余重建
  const hasImage = all.some((file) => isFileLike(file) && canCompressImage(file))
  if (!hasImage) {
    console.info('[webp] 未识别到可压缩图片，按原样上传：', all.map((f) => `${f?.name || '?'}(${f?.type || '?'})`))
    return formData
  }

  const compressOne = async (file) => {
    if (!isFileLike(file) || !canCompressImage(file)) return file
    // 实况图原样上传：canvas 重编码会丢掉 JPEG 里内嵌的 MP4
    if (await isMotionPhoto(file)) {
      console.info('[webp] 跳过实况图压缩：', file.name)
      return file
    }
    return compressImageToWebp(file)
  }

  const next = new FormData()
  // 保留文件之外的字段（target_type / target_id 等）
  for (const [key, value] of formData.entries()) {
    if (key !== 'file' && key !== 'files') next.append(key, value)
  }
  // 按原字段名重建，避免后端按 files / file 分支解析时取不到
  for (const file of single) {
    next.append('file', await compressOne(file))
  }
  for (const file of multiple) {
    next.append('files', await compressOne(file))
  }
  return next
}

// 批量上传（单文件/多文件通用），multipart/form-data
// 注意：不要手动设置 Content-Type，request 拦截器会移除它，由浏览器自动补 boundary
export const uploadAttachments = async (formData) => {
  const data = await prepareUploadFormData(formData)
  return call('attachment', 'batch', { method: 'POST', data })
}

// ===== 后台（管理员）=====

/**
 * 附件列表（管理员）
 * 非管理员只能查到 uploader_id 为自己的附件，后台依赖管理员身份才能管理全站附件。
 * 支持 where / like（「字段名|值」）/ order / onlyTrashed。
 */
export const listAttachmentsAdmin = (params = {}) =>
  call('attachment', 'all', {
    method: 'GET',
    params: { page: 1, limit: 24, order: 'create_time desc', ...params }
  })

/**
 * 附件数量（count 支持 onlyTrashed，回收站数量可直接用该接口）
 */
export const countAttachments = (params = {}) =>
  call('attachment', 'count', { method: 'GET', params })

// 附件体积合计（返回 { file_size: 合计字节数 }）
export const sumAttachmentSize = (params = {}) =>
  call('attachment', 'sum', { method: 'GET', params: { field: 'file_size', ...params } })

/**
 * 当前启用的存储方式（返回 { driver, name }）
 *
 * 注意与附件记录里的 storage_driver 区分：后者是**每个附件上传时**用的驱动（历史数据可能
 * 残留已下线的 oss / kodo），本接口返回的是配置里当前生效的驱动，即「新上传的文件存到哪儿」。
 * driver 只有 local（本地存储）/ cos（腾讯云 COS），后端对未知值已回退 local。
 */
export const getAttachmentStorage = () =>
  call('attachment', 'storage', { method: 'GET' })

// 更新附件：允许字段仅 original_name / target_type / target_id
export const updateAttachment = (data) =>
  call('attachment', 'update', { method: 'PUT', data })

// 软删除（移入回收站，不删除存储文件）
export const removeAttachments = (ids) =>
  call('attachment', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（仅管理员：连同存储文件一起删除，不可恢复）
export const forceDeleteAttachments = (ids) =>
  call('attachment', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreAttachments = (ids) =>
  call('attachment', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站（仅管理员：连同存储文件一起删除）
export const clearAttachmentRecycle = () =>
  call('attachment', 'clear', { method: 'DELETE' })
