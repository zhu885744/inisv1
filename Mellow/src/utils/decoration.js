/**
 * 装扮元数据（前台展示用）
 *
 * 稀有度取值与后台「装扮管理 → 稀有度」保持一致（后端 model/decoration.go 的
 * Decoration.Rarity，默认 normal）。这里只做「取值 → 文案 + 配色」的映射，
 * 商城卡片与「我的装扮」共用，避免两处各写一份。
 */
export const DECORATION_RARITY = {
  normal: { label: '普通', tone: 'normal' },
  rare: { label: '稀有', tone: 'rare' },
  epic: { label: '史诗', tone: 'epic' },
  legend: { label: '传说', tone: 'legend' }
}

// 未知 / 空值一律按「普通」处理：后台新增稀有度档位时前台不会显示成空白
export function rarityMeta(rarity) {
  return DECORATION_RARITY[rarity] || DECORATION_RARITY.normal
}
