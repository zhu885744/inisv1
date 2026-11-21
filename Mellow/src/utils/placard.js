/**
 * 公告类型（placard.type）：notice 公告 / warning 警告 / info 提示
 *
 * 后台「公告类型」下拉与列表圆点/标签（views/admin/Placard.vue）、首页轮播圆点与
 * 公告内容弹窗的类型标签（views/home/Index.vue）全部读这里，改文案或配色只动一处。
 *
 * 配色说明：三者的色相刻意拉开，避免「公告」和「警告」看起来是同一个颜色
 * （历史上公告用主题朱砂 --primary、警告用 #d9544d，两个红色几乎无法区分）：
 *   公告 → 主题朱砂（品牌色，正式发布）
 *   警告 → 主题琥珀（--warning，警示语义，浅色/暗色主题各自适配）
 *   提示 → 天蓝（信息语义）
 *
 * color 用于文字/圆点，bg 是同色淡底（CSS 变量无法直接算 rgba，这里写死与主题同源的色值）。
 */
export const PLACARD_TYPES = [
  { value: 'notice', label: '公告', color: 'var(--primary)', bg: 'rgba(199, 72, 42, 0.12)' },
  { value: 'warning', label: '警告', color: 'var(--warning)', bg: 'rgba(192, 138, 46, 0.16)' },
  { value: 'info', label: '提示', color: '#4a90e2', bg: 'rgba(74, 144, 226, 0.14)' }
]

/** 归一化类型值：未知或空值回退 notice（与后端默认类型一致） */
export function placardTypeKey(type) {
  return PLACARD_TYPES.some((item) => item.value === type) ? type : 'notice'
}

/** 取类型定义（含 label / color / bg） */
export function placardTypeOf(type) {
  const key = placardTypeKey(type)
  return PLACARD_TYPES.find((item) => item.value === key) || PLACARD_TYPES[0]
}

/** 类型文案，如「公告」「警告」「提示」 */
export function placardTypeLabel(type) {
  return placardTypeOf(type).label
}

/** 类型主色（圆点 / 文字用） */
export function placardTypeColor(type) {
  return placardTypeOf(type).color
}

/** 类型标签的行内样式：文字用类型色、底色用同色淡底 */
export function placardTypeStyle(type) {
  const item = placardTypeOf(type)
  return { color: item.color, background: item.bg }
}
