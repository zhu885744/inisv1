/**
 * 通知跳转：把通知的 bind_type / bind_id 翻译成前台路由地址
 *
 * 各类型的 bind 语义并不统一（见 app/model/notification.go、app/model/tip.go）：
 *   article / moments / page  → 内容 id，直接进详情页
 *   comment                   → 评论 id，需要先查评论拿到所属内容再跳（本站评论通知也走这里）
 *   user                      → 用户 uid，进用户主页 /author/:id
 *   user-certification        → 认证记录 id，进「我的认证」查看资料与审核状态
 *   links / integral-card / decoration → 固定页面
 *
 * 消息中心（views/notifications/Index.vue）与侧栏最新消息（components/SidebarRight.vue）
 * 共用本模块，避免两处维护两份跳转规则。
 */
import { getComment } from '@/api/comment'

/**
 * 内容类型 → 详情地址
 * @param {string} bindType article / moments / page
 * @param {number|string} bindId 内容 id
 * @param {number|string} commentId 可选：评论 id，带上 ?comment= 让详情页高亮该评论
 * @returns {string} 未知类型或 id 非法时返回空串
 */
export function contentLink(bindType, bindId, commentId = 0) {
  const id = Number(bindId) || 0
  if (!id) return ''

  const highlight = Number(commentId) > 0 ? `?comment=${Number(commentId)}` : ''

  switch (String(bindType || '').toLowerCase()) {
    case 'article':
      return `/archives/${id}${highlight}`
    case 'moments':
    case 'moment':
      return `/moments/${id}${highlight}`
    case 'page':
      return `/page/${id}${highlight}`
    default:
      return ''
  }
}

/**
 * 评论 id → 所属内容地址（评论区 / 打赏评论的通知 bind_id 是评论 id）
 * 查不到（评论已删除 / 待审核对当前用户不可见）时返回空串。
 */
export async function commentLink(commentId) {
  const id = Number(commentId) || 0
  if (!id) return ''

  try {
    const res = await getComment(id)
    const data = res?.data || {}
    return contentLink(data.bind_type, data.bind_id, id)
  } catch {
    return ''
  }
}

/**
 * 通知 → 跳转地址
 * @param {{ bind_type?: string, bind_id?: number }} n 通知对象
 * @returns {Promise<string>} 没有可跳转目标时返回空串（调用方自行决定兜底页）
 */
export async function notificationLink(n) {
  const bindType = String(n?.bind_type || '').toLowerCase()
  const bindId = Number(n?.bind_id) || 0

  switch (bindType) {
    case 'links':
      // 友链审核结果：去「我的友链」看状态与驳回原因
      return '/manage/links'
    case 'integral-card':
      // 管理员绑定的积分卡密：去「我的积分 → 卡密兑换」
      return '/user/integral'
    case 'decoration':
      // 装扮（获得 / 续期 / 回收 / 过期）
      return '/user/decorations'
    case 'badge':
      // 勋章（限定勋章即将到期）：去「我的勋章」看进度并佩戴
      return '/user/badges'
    case 'user-certification':
      // 认证（审核结果 / 后台改资料）：去「我的认证」看状态与最新资料
      return '/user/verification'
    case 'user':
      // 用户主页打赏 / 关注 / 提及
      return bindId ? `/author/${bindId}` : ''
    case 'comment':
      return commentLink(bindId)
    default:
      return contentLink(bindType, bindId)
  }
}
