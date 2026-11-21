/**
 * 内容归属人（作者 / 申请人）信息缓存
 *
 * 后台的审核弹窗、编辑弹窗、列表都要展示「这条内容是谁的」，如果各自去查用户会重复请求。
 * 这里统一维护一份模块级缓存：同一个 uid 只请求一次，弹窗与列表共用同一份数据。
 *
 * 用法：
 *   import { loadOwners, ownerName, ownerTitle } from '@/utils/contentOwner'
 *   loadOwners([uid])           // 需要时补数据（内部去重、失败可重试）
 *   ownerName(uid)              // 昵称 → 账号 → 「用户 {uid}」
 *   ownerTitle(uid)             // hover 提示：账号 / 昵称 / UID
 *
 * 组件里直接用 OwnerInfo.vue（内部已经接好这一套）。
 */
import { ref } from 'vue'
import { listUsersByIds } from '@/api/users'

// uid → 用户信息（id/nickname/avatar/account）
const cache = ref({})
// 正在请求中的 uid，避免并发重复请求
const pending = new Set()

/** 取缓存的用户信息（未加载到返回 null） */
export function ownerOf(uid) {
  const id = Number(uid)
  if (!id) return null
  return cache.value[id] || null
}

/** 归属人显示名：昵称优先，其次账号，都取不到回退「用户 {uid}」；uid=0 用 fallback */
export function ownerName(uid, fallback = '管理员添加') {
  const id = Number(uid)
  if (!id) return fallback
  const user = cache.value[id]
  return user?.nickname || user?.account || `用户 ${id}`
}

/** 归属人 hover 提示：账号 / 昵称 / UID 都摆出来，方便核对 */
export function ownerTitle(uid, fallback = '这条内容由管理员创建') {
  const id = Number(uid)
  if (!id) return fallback
  const user = cache.value[id]
  if (!user) return `用户 UID：${id}`
  const parts = []
  if (user.account) parts.push(`账号：${user.account}`)
  if (user.nickname) parts.push(`昵称：${user.nickname}`)
  parts.push(`UID：${id}`)
  return parts.join(' · ')
}

/**
 * 批量补数据：只请求「还没缓存且不在请求中」的 uid
 * - 失败时把这些 uid 从 pending 里放掉，下次展示还能重试
 * - 只做基础信息（id/nickname/avatar/account），不会带出敏感字段
 */
export async function loadOwners(uids) {
  const ids = [
    ...new Set(
      (uids || [])
        .map((i) => Number(i))
        .filter((id) => id > 0 && !cache.value[id] && !pending.has(id))
    )
  ]
  if (!ids.length) return

  ids.forEach((id) => pending.add(id))
  try {
    const res = await listUsersByIds(ids)
    const map = { ...cache.value }
    ;(res.data?.data || []).forEach((u) => {
      if (u?.id) map[u.id] = u
    })
    cache.value = map
  } catch {
    // 拿不到用户信息时按「用户 {uid}」展示，不阻断弹窗
  } finally {
    ids.forEach((id) => pending.delete(id))
  }
}
