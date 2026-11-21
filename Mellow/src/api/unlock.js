import { call } from './request'

// ===== 内容可见范围（登录可见 / 付费可见）=====

// 查询某条内容的可见状态：
//   scope 1 公开 / 2 登录可见 / 3 付费可见
//   scope_state '' 可查看 / login 需登录 / pay 需兑换
//   can_view 当前是否可查看完整内容
export const checkUnlock = (bindType, bindId) =>
  call('unlock', 'check', {
    method: 'GET',
    params: { bind_type: bindType, bind_id: bindId }
  })

// 使用积分兑换内容可见权限（同一用户只需兑换一次）
export const payUnlock = (bindType, bindId) =>
  call('unlock', 'pay', {
    method: 'POST',
    data: { bind_type: bindType, bind_id: bindId }
  })

// 我的兑换记录
export const listUnlocks = (params = {}) =>
  call('unlock', 'list', { method: 'GET', params })
