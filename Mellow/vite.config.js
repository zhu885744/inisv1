import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig(({ mode }) => {
  // 读取 .env 中的后端地址（VITE_API_URI 等）
  const env = loadEnv(mode, process.cwd(), '')
  const API_URI = env.VITE_API_URI || 'https://zhuxu.asia'
  const SOCKET_URI = env.VITE_SOCKET || 'wss://zhuxu.asia/socket'

  // 代理 target 只能给「源」（scheme + host[:port]），不能带路径：
  // VITE_SOCKET 通常写成 wss://host/socket，若把带路径的值直接当 target，
  // http-proxy 会把 target 的路径拼在请求路径前（/socket → /socket/socket），
  // 后端匹配不到该路由、回落到 SPA 首页，浏览器侧表现为
  // 「WebSocket handshake: Unexpected response code: 200」。
  const socketTarget = (() => {
    try {
      const url = new URL(SOCKET_URI, 'https://placeholder.local')
      const secure = url.protocol === 'wss:' || url.protocol === 'https:'
      return `${secure ? 'wss:' : 'ws:'}//${url.host}`
    } catch {
      return 'wss://zhuxu.asia'
    }
  })()

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        // 使用 import.meta.dirname（Node 20.11+），替代 Vite 8 原生配置加载器不支持的 __dirname
        '@': path.resolve(import.meta.dirname, './src')
      }
    },
    server: {
      port: 5173,
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: API_URI,
          changeOrigin: true,
          // 后端登录写入的 Set-Cookie 带 Domain=cs.zhuxu.asia，
          // 浏览器在 localhost 下会拒绝保存，需改写为空（即当前域），开发态才能保持登录态
          cookieDomainRewrite: '',
          cookiePathRewrite: '/'
        },
        // 主题配色样式表：由后端按后台配置动态生成（见 app/model/theme.go），
        // dev 下不代理会拿到 Vite 的 index.html，配色改动看不到效果
        '/theme-custom.css': {
          target: API_URI,
          changeOrigin: true,
          cookieDomainRewrite: '',
          cookiePathRewrite: '/'
        },
        // 静态资源（含表情图片、上传附件等）也转发到后端
        '/assets': {
          target: API_URI,
          changeOrigin: true,
          cookieDomainRewrite: '',
          cookiePathRewrite: '/'
        },
        // WebSocket 实时通道（见 src/utils/socket.js）：ws: true 才会转发 Upgrade 握手
        '/socket': {
          target: socketTarget,
          ws: true,
          changeOrigin: true
        },
        // dev 类接口（如 /dev/info/time 服务器时间，用于登录加密盐派生），不走 /api 前缀
        '/dev': {
          target: API_URI,
          changeOrigin: true,
          cookieDomainRewrite: '',
          cookiePathRewrite: '/'
        }
      }
    },
    build: {
      outDir: 'dist',
      // 产物静态目录统一为 static（默认 assets），避免与后端自身的静态资源目录
      // public/assets（emoji、随机头像、上传附件等）撞路径：
      // 两者混在一个目录下时，前端产物缺失会命中后端的 JSON 错误响应，
      // 浏览器对 <script type="module"> 做严格 MIME 校验后直接报错。
      // 注意：后端 config/app.toml 的 app.theme_ignore_prefix 已包含 /static（见 config/app.go）
      assetsDir: 'static',
      sourcemap: false,
      // 阈值回到默认量级：1500 会把所有真实的分包告警一起吞掉
      chunkSizeWarningLimit: 600,
      rollupOptions: {
        output: {
          // 大依赖单独成 chunk：语法高亮只在正文页用到，不必拖累首屏
          // （用函数形式匹配，避免写死模块 id 后在依赖升级时失效）
          manualChunks(id) {
            if (id.includes('highlight.js')) return 'highlight'
            if (id.includes('markdown-it')) return 'markdown'
            return undefined
          }
        }
      }
    }
  }
})
