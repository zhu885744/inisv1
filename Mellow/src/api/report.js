import { call } from './request'
import { uploadAttachments } from './attachment'

/**
 * 举报（动态 / 文章 / 评论 / 用户）
 *
 * 后端：app/api/controller/report.go
 *   - 提交：POST report/create（必须登录；每人每天上限 20 条；同一对象只允许一条待处理）
 *   - 我的：GET report/mine
 *   - 撤回：DELETE report/cancel（仅「待处理」）
 *   - 后台：GET report/all、PUT report/handle（需要权限点）
 *
 * 举报原因与对象类型的枚举与后端 model/ReportReasons、ReportTargets 一一对应，
 * 增删时两边都要改（后端 reportOutput 会回带 *_text 文案，前端直接用即可）。
 */

/** 举报对象类型（value 与后端一致，label 仅用于展示） */
export const REPORT_TARGETS = [
  { value: 'moment', label: '动态' },
  { value: 'article', label: '文章' },
  { value: 'comment', label: '评论' },
  { value: 'user', label: '用户' }
]

/** 举报原因（与 model.ReportReasons 同源） */
export const REPORT_REASONS = [
  { value: 'spam', label: '垃圾广告 / 刷屏' },
  { value: 'abuse', label: '辱骂 / 人身攻击' },
  { value: 'porn', label: '色情低俗' },
  { value: 'illegal', label: '违法违规' },
  { value: 'politics', label: '敏感政治' },
  { value: 'infringe', label: '侵权 / 抄袭' },
  { value: 'harass', label: '骚扰 / 恶意私信' },
  { value: 'other', label: '其他' }
]

/**
 * 提交举报
 * @param {{target_type: string, target_id: number, reason: string, detail?: string, images?: string[]}} data
 *        images 为证据图片 URL 数组（最多 3 张，后端会再清洗一次：只留 http(s)、限量截断）
 */
export const createReport = (data) => call('report', 'create', { method: 'POST', data })

/**
 * 上传举报证据图片（统一走 uploadAttachments，含 WebP 压缩处理）
 * 返回 res.data.results：数组项含 status 与 full_url
 */
export const uploadReportImages = (formData) => uploadAttachments(formData)

/** 我提交的举报（分页） */
export const myReports = (params = {}) =>
  call('report', 'mine', { method: 'GET', params: { page: 1, limit: 20, ...params } })

/** 撤回举报（仅待处理） */
export const cancelReport = (id) => call('report', 'cancel', { method: 'DELETE', params: { id } })

/** 后台：举报列表（status 不传为全部） */
export const listReports = (params = {}) =>
  call('report', 'all', { method: 'GET', params: { page: 1, limit: 20, ...params } })

/**
 * 后台：处理举报
 * @param {{id: number, status: 1|2, remark?: string, remove_content?: 0|1}} data
 *        status 1=属实（可勾选同时移除内容）2=驳回
 */
export const handleReport = (data) => call('report', 'handle', { method: 'PUT', data })
