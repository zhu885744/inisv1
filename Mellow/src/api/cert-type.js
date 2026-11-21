import { call } from './request'

/**
 * 认证类型配置（后台可自定义）
 *
 * 后端：app/model/cert-type.go + app/api/controller/cert-type.go
 *   - 公开：GET  cert-type/enable   启用中的类型（领域候选 / 字段定义 / 认证说明）
 *   - 后台：GET  cert-type/all      全部类型（含停用，带 used 记录数）
 *   - 后台：POST cert-type/create / PUT cert-type/update / DELETE cert-type/remove
 *
 * 一个类型 = 一套认证表单，字段定义见下面 CERT_FIELD_TYPES；
 * 前端只负责按定义渲染，不再维护「类型 / 领域」的硬编码副本。
 */

/** 字段类型（value 与后端 model.CertFieldText 等常量一致，新增类型必须同时补渲染器） */
export const CERT_FIELD_TYPES = [
  { value: 'text', label: '单行文本' },
  { value: 'textarea', label: '多行文本' },
  { value: 'number', label: '数字' },
  { value: 'select', label: '下拉选择' },
  { value: 'date', label: '日期' },
  { value: 'link', label: '链接' },
  { value: 'image', label: '图片' }
]

/**
 * 申请条件（与后端 model.CertConditions 一一对应，0 表示不限）
 *
 * 只支持站点里真实存在的四类统计口径：粉丝 / 已发布文章 / 已发布动态 / 注册天数。
 * 「已发布」= audit=1 且 status=1，草稿与待审内容不计入，避免刷数量。
 */
export const CERT_CONDITIONS = [
  { key: 'fans', label: '粉丝数', unit: '人', hint: '关注你的用户数' },
  { key: 'articles', label: '发布文章', unit: '篇', hint: '已通过审核的文章（草稿 / 待审不算）' },
  { key: 'moments', label: '发布动态', unit: '条', hint: '已通过审核的动态' },
  { key: 'days', label: '注册天数', unit: '天', hint: '账号注册至今的天数' }
]

/** 空白条件（全部不限） */
export function blankCertConditions() {
  return CERT_CONDITIONS.reduce((result, item) => {
    result[item.key] = 0
    return result
  }, {})
}

/** 条件摘要文案（后台列表 / 申请页共用口径） */
export function certConditionText(conditions) {
  const list = CERT_CONDITIONS.filter((item) => Number(conditions?.[item.key]) > 0)
  if (!list.length) return '无门槛'
  return list.map((item) => `${item.label} ≥ ${Number(conditions[item.key])}${item.unit}`).join(' · ')
}

/** 内置字段键：有特殊语义（写回记录的独立列），后台编辑器里给提示用 */
export const CERT_SYSTEM_KEYS = {
  title: '认证名称 → 记录 title 列，展示在主页昵称旁',
  category: '认证领域 → 记录 category 列，后台可按它筛选',
  images: '证明材料 → 记录 images 列，后台审核时查看'
}

/** 新建类型的默认字段（与旧版表单一致，运营可随意增删） */
export function blankCertFields() {
  return [
    {
      key: 'title', label: '认证名称', type: 'text', required: true,
      placeholder: '不超过 32 字', hint: '展示在昵称旁的身份', max: 32,
      options: [], multiple: false, max_count: 0, public: true, admin_only: false
    },
    {
      key: 'category', label: '认证领域', type: 'select', required: true,
      placeholder: '', hint: '选项来自本类型的认证领域', max: 0,
      options: [], multiple: false, max_count: 0, public: true, admin_only: false
    },
    {
      key: 'images', label: '证明材料', type: 'image', required: true,
      placeholder: '', hint: '最多 6 张', max: 0,
      options: [], multiple: false, max_count: 6, public: false, admin_only: true
    }
  ]
}

/** 新建字段的空白模板 */
export function blankCertField() {
  return {
    key: '', label: '', type: 'text', required: false,
    placeholder: '', hint: '', max: 0,
    options: [], multiple: false, max_count: 0, public: false, admin_only: false
  }
}

/** 启用中的认证类型（公开，申请页据此渲染动态表单） */
export const getEnabledCertTypes = () => call('cert-type', 'enable', { method: 'GET' })

/** 后台：全部认证类型 */
export const listCertTypes = (params = {}) =>
  call('cert-type', 'all', { method: 'GET', params: { page: 1, limit: 100, ...params } })

/** 后台：新建类型 */
export const createCertType = (data) => call('cert-type', 'create', { method: 'POST', data })

/** 后台：更新类型（局部更新：没带的键沿用原值） */
export const updateCertType = (data) => call('cert-type', 'update', { method: 'PUT', data })

/**
 * 后台：删除类型（软删除）
 * @param {number|number[]} ids
 * @param {boolean} force 类型下还有认证记录时必须为 true（否则后端拒绝）
 */
export const removeCertType = (ids, force = false) =>
  call('cert-type', 'remove', {
    method: 'DELETE',
    params: {
      ids: (Array.isArray(ids) ? ids : [ids]).map((item) => Number(item)).filter((item) => item > 0).join(','),
      ...(force ? { force: 1 } : {})
    }
  })

/** 按 id 找类型（前端多处要用同一个口径，统一在这里兜底） */
export function findCertType(types, id) {
  const list = Array.isArray(types) ? types : []
  return list.find((item) => Number(item.id) === Number(id)) || null
}

/** 类型下某个字段的候选值（领域字段跟随类型的领域列表） */
export function certFieldOptions(type, field) {
  if (!field) return []
  if (field.key === 'category' && !(field.options || []).length) {
    return (type?.categories || []).map((item) => ({ value: item, label: item }))
  }
  return (field.options || []).map((item) => ({ value: item, label: item }))
}
