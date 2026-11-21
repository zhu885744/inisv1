import { call } from './request'

/**
 * 主题配色（设计 token）
 *
 * 后端实现见 app/api/controller/theme.go 与 app/model/theme.go：
 * - config：token 表 / 分组 / 内置预设 / 当前配色 / 当前 CSS（公开）
 * - save  ：保存配色，/theme-custom.css 立即生效
 * - reset ：恢复默认预设
 */

// 主题配色配置
export const getThemeConfig = () => call('theme', 'config', { method: 'GET' })

/**
 * 保存主题配色
 * @param {{preset: string, light: Record<string, string>, dark: Record<string, string>}} data
 */
export const saveThemeConfig = (data) => call('theme', 'save', { method: 'POST', data })

// 恢复默认配色
export const resetThemeConfig = () => call('theme', 'reset', { method: 'POST' })
