import { call } from './request'
import { uploadAttachments } from './attachment'

/**
 * 用户认证（类型 / 领域 / 申请字段由后台配置，见 @/api/cert-type）
 *
 * 后端：app/api/controller/user-certification.go
 *   - 公开：GET  user-certification/one?uid=   用户主页展示的认证摘要（未认证返回 null）
 *   - 本人：GET  user-certification/mine       我的认证（含状态 / 驳回原因 / 已提交资料 / fields）
 *   - 本人：POST user-certification/apply      提交或修改申请（values 里按字段 key 传值）
 *   - 后台：GET  user-certification/all        认证列表（status / cert_type / uid / category 筛选）
 *   - 后台：PUT  user-certification/edit       修改用户提交的资料（不改变审核状态）
 *   - 后台：PUT  user-certification/handle     审核通过 / 驳回（通过可写认证说明，驳回必须写原因）
 *
 * 注意：认证类型与领域**不再有前端枚举副本**，一律来自 cert-type/enable（后台可随时增删改）。
 * 「认证说明」由后台审核时填写，申请页没有任何让用户填写说明的入口。
 */

/**
 * 提交认证申请（重新提交会覆盖旧资料并回到待审核）
 * @param {{cert_type:number, values:Object<string, any>}} data
 *        values 的键就是类型配置里字段的 key（title / category / images / 自定义…）
 */
export const applyCertification = (data) =>
  call('user-certification', 'apply', { method: 'POST', data })

/** 我的认证（未提交过时 data 为 null） */
export const getMyCertification = () => call('user-certification', 'mine', { method: 'GET' })

/** 某个用户的认证摘要（公开，未认证时 data 为 null） */
export const getUserCertification = (uid) =>
  call('user-certification', 'one', { method: 'GET', params: { uid } })

/**
 * 后台：直接授予认证（运营主动核验，不经过用户提交 / 审核队列）
 * @param {{uid:number, cert_type:number, values?:Object<string, any>, description?:string, expire_days?:number}} data
 *        values 的键就是类型配置里字段的 key；必填项不强制，缺认证名称时后端回落类型名
 */
export const grantCertification = (data) =>
  call('user-certification', 'grant', { method: 'POST', data })

/**
 * 后台：收回已发出的认证（记录会被删除，用户主页不再展示标识，可重新申请）
 * @param {{id:number, reason?:string}} data
 *        reason 可空，会随站内信一起发给用户
 */
export const revokeCertification = ({ id, reason = '' }) =>
  call('user-certification', 'revoke', { method: 'DELETE', params: { id, reason } })

/** 上传证明材料（走附件模块，含 WebP 压缩） */
export const uploadCertificationImages = (formData) => uploadAttachments(formData)

/** 后台：认证列表（返回值里的 types 是类型筛选项） */
export const listCertifications = (params = {}) =>
  call('user-certification', 'all', { method: 'GET', params: { page: 1, limit: 20, ...params } })

/**
 * 后台：修改用户提交的认证资料（不改变审核状态）
 * @param {{id:number, values?:Object<string, any>, description?:string}} data
 *        values 传了才按类型配置重写字段值（缺失项视为清空）；不传则只改认证说明
 */
export const editCertification = (data) =>
  call('user-certification', 'edit', { method: 'PUT', data })

/**
 * 后台：审核认证
 * @param {{id: number, status: 1|2, reason?: string, description?: string, expire_days?: number}} data
 *        status 1=通过 2=驳回（reason 必填）；description 是后台填写的认证说明；
 *        expire_days 0 表示永久
 */
export const handleCertification = (data) =>
  call('user-certification', 'handle', { method: 'PUT', data })
