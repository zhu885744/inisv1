/**
 * 后台列表的分页条数
 *
 * 约定：后台列表接口默认每页 10 条（后端 limit 缺省值也是 10），
 * 用户可在分页条旁的下拉里改成 20 / 30 / 40 / 50 / 100。
 * 选项集中在这里，避免 28 个列表页各写一份数字，改一处就全站生效。
 *
 * 用法：
 *   const pageSize = ref(PAGE_SIZE_DEFAULT)
 *   <Pagination v-model:page-size="pageSize" :sizes="PAGE_SIZE_OPTIONS" ... />
 */
export const PAGE_SIZE_DEFAULT = 10

// 下拉选项的取值（数字，直接作为接口的 limit）
export const PAGE_SIZES = [10, 20, 30, 40, 50, 100]

// SelectMenu 需要的 [{ value, label }] 形式
export const PAGE_SIZE_OPTIONS = PAGE_SIZES.map((size) => ({
  value: size,
  label: `${size} 条/页`
}))
