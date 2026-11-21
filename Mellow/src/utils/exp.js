/**
 * 经验明细的展示辅助（经验页 + 明细弹窗共用）
 *
 * 后端 exp/all 返回的字段：id / value / type / description / json / create_time。
 * type 与 model/exp.go 的规则 key 对应；json 里可能有签到相关的附加信息
 * （新签到模块写 { source, streak, cycle }，历史记录里可能还有 bonus / milestone）。
 */
import { formatTime, fromNow } from '@/utils/time'

const TYPE_ICON = {
  like: 'bi bi-hand-thumbs-up',
  collect: 'bi bi-bookmark',
  visit: 'bi bi-eye',
  share: 'bi bi-share',
  login: 'bi bi-box-arrow-in-right',
  comment: 'bi bi-chat-dots',
  'check-in': 'bi bi-calendar-check',
  moments: 'bi bi-chat-square-quote',
  'article-create': 'bi bi-file-earmark-text',
  'article-like': 'bi bi-hand-thumbs-up',
  'article-collect': 'bi bi-bookmark-star',
  'comment-create': 'bi bi-chat-square-text',
  'comment-like': 'bi bi-chat-heart'
}

const TYPE_NAME = {
  like: '点赞',
  collect: '收藏',
  visit: '访问',
  share: '分享',
  login: '登录',
  comment: '评论',
  'check-in': '签到',
  moments: '发布动态',
  'article-create': '发布文章',
  'article-like': '内容获赞',
  'article-collect': '内容被收藏',
  'comment-create': '发表评论',
  'comment-like': '评论获赞'
}

export function expIconOf(type) {
  return TYPE_ICON[type] || 'bi bi-stars'
}

export function expTypeName(type) {
  return TYPE_NAME[type] || type || '经验变动'
}

function safeParse(text) {
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

/** 附加说明（补签 / 连续签到加成 / 里程碑 / 周期第几天） */
export function expExtraText(json) {
  if (!json) return ''
  // 列表接口可能给对象（已解码）也可能给原始字符串，两种都要兼容
  const data = typeof json === 'string' ? safeParse(json) : json
  if (!data || typeof data !== 'object') return ''

  const parts = []
  if (Number(data.source) === 2) parts.push('补签')
  if (Number(data.bonus) > 0) parts.push(`连续签到加成 +${data.bonus}`)
  if (Number(data.milestone) > 0) parts.push(`里程碑奖励 +${data.milestone}`)
  if (Number(data.streak) > 0) parts.push(`已连续签到 ${data.streak} 天`)
  if (Number(data.cycle) > 0) parts.push(`周期第 ${data.cycle} 天`)
  return parts.join(' · ')
}

/**
 * 一条经验流水 -> 渲染所需字段
 *
 * 在 computed 里一次性算好，模板里不再逐行调用函数（列表翻页后行数翻倍也不会重复计算）。
 */
export function buildExpRow(item) {
  const value = Number(item?.value) || 0
  const time = Number(item?.create_time) || 0

  return {
    id: item?.id,
    icon: expIconOf(item?.type),
    title: item?.description || expTypeName(item?.type),
    extra: expExtraText(item?.json),
    minus: value < 0,
    valueText: `${value > 0 ? '+' : ''}${value}`,
    timeAgo: fromNow(time),
    timeFull: formatTime(time),
    datetime: time ? new Date(time * 1000).toISOString() : ''
  }
}
