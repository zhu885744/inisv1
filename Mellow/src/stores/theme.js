import { defineStore } from 'pinia'

const THEME_KEY = 'mellow_theme'

function systemPrefersDark() {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-color-scheme: dark)').matches
  )
}

/**
 * 主题状态：只有 light / dark 两种
 *
 * 一次点击就切到另一种，不再有「跟随系统」这个中间态：
 *   - 首次访问（或旧版本存了 auto）时按系统偏好定一次，随后立即落盘成明确值；
 *   - 之后系统换主题不再影响页面，用户选了什么就是什么。
 *
 * 实际生效主题写入 <html data-theme>，配合 index.html 内联脚本避免刷新闪烁。
 */
export const useThemeStore = defineStore('theme', {
  state: () => ({
    mode: 'light'
  }),
  getters: {
    resolved: (state) => (state.mode === 'dark' ? 'dark' : 'light'),
    isDark() {
      return this.resolved === 'dark'
    }
  },
  actions: {
    init() {
      let saved = ''
      try {
        saved = localStorage.getItem(THEME_KEY) || ''
      } catch {
        /* localStorage 不可用时按系统偏好 */
      }

      if (saved === 'light' || saved === 'dark') {
        this.mode = saved
      } else {
        // 首次访问 / 旧版 auto：取一次系统偏好，并持久化成两态里的一个
        this.mode = systemPrefersDark() ? 'dark' : 'light'
        this.persist()
      }

      this.apply()
    },
    persist() {
      try {
        localStorage.setItem(THEME_KEY, this.mode)
      } catch {
        /* 忽略持久化失败 */
      }
    },
    set(mode) {
      // 只接受两态：传其它值（如历史遗留的 auto）按浅色处理
      this.mode = mode === 'dark' ? 'dark' : 'light'
      this.persist()
      this.apply()
    },
    /** 浅色 ⇄ 深色，一次点击即生效 */
    toggle() {
      this.set(this.isDark ? 'light' : 'dark')
    },
    apply() {
      document.documentElement.dataset.theme = this.resolved
    }
  }
})
