import { call } from './request'

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

// ===== 权限组（auth-group）=====

/**
 * 权限组列表
 * uids 以 "|1|2|" 形式存储组内成员，编辑用户权限时需要读出来判断归属
 */
export const listAuthGroups = (params = {}) =>
  call('auth-group', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 100,
      order: 'id asc',
      field: 'id,name,key,root,uids,remark',
      ...params
    }
  })

/**
 * 设置用户所属权限组（后端会以本次 ids 为准覆盖全部组）
 * 即：不在 ids 中的组会移除该用户，ids 中的组会加入该用户
 * @param {number} uid 用户 ID
 * @param {number[]} ids 目标权限组 ID 数组
 */
export const setUserAuthGroups = (uid, ids = []) =>
  call('auth-group', 'uids', {
    method: 'PUT',
    data: { uid: Number(uid), ids: (Array.isArray(ids) ? ids : [ids]).map((i) => Number(i)).filter(Boolean) }
  })

// ===== 后台（管理员）=====

// 后台列表字段：需要 result（内含成员解析结果）与 rules / pages 才能展示授权概况
export const AUTH_GROUP_ADMIN_FIELD =
  'id,name,key,root,uids,rules,pages,default,remark,create_time,update_time,delete_time,result'

/**
 * 权限组列表（管理员；传 onlyTrashed=true 看回收站）
 */
export const listAuthGroupsAdmin = (params = {}) =>
  call('auth-group', 'all', { method: 'GET', params })

// 权限组数量（count 支持 onlyTrashed）
export const countAuthGroups = (params = {}) =>
  call('auth-group', 'count', { method: 'GET', params })

/**
 * 保存权限组：无 id 为新增，有 id 为更新（后端 save 自动分流）
 *
 * 字段格式（见 model.AuthGroup）：
 * - uids  "|1|2|" 竖线包裹的成员 ID
 * - rules 规则 hash 逗号分隔，或字面量 "all" 表示全部规则
 * - pages 页面 hash 逗号分隔，或字面量 "all" 表示全部页面
 * - root  是否具备越权操作数据的能力（0/1）
 * 注意：default（系统内置标记）不在后端允许字段里，不能修改。
 */
export const saveAuthGroup = (data) =>
  call('auth-group', 'save', { method: 'POST', data })

// 软删除（系统管理员分组 id=1 与系统内置分组 default=1 会被后端拒绝）
export const removeAuthGroups = (ids) =>
  call('auth-group', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复，同样受系统分组保护）
export const forceDeleteAuthGroups = (ids) =>
  call('auth-group', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreAuthGroups = (ids) =>
  call('auth-group', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearAuthGroupRecycle = () =>
  call('auth-group', 'clear', { method: 'DELETE' })
