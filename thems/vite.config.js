import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'

export default defineConfig(({ mode }) => {
  const isProduction = mode === 'production'

  return {
    plugins: [
      vue({
        template: {
          compilerOptions: {
            preserveWhitespace: false
          }
        }
      }),
      createSvgIconsPlugin({
        iconDirs: [path.resolve(__dirname, 'src/assets/svg')],
        symbolId: 'icon-[name]',
        inject: 'body-last',
        customDomId: '__svg__icons__dom__'
      })
    ],

    resolve: {
      extensions: ['.vue', '.js', '.jsx', '.ts', '.tsx'],
      alias: {
        '@': path.resolve(__dirname, 'src'),
      }
    },

    base: '/',

    server: {
      port: 3000,
      open: true,
      cors: true,
      timeout: 60000,
      http2: false,
      hmr: { overlay: false, timeout: 3000 },
      proxy: {
        '/api': {
          target: 'https://cs.zhuxu.asia',
          changeOrigin: true,
          timeout: 60000,
          ws: true
        }
      },
      headers: {
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'SAMEORIGIN',
        'X-XSS-Protection': '1; mode=block',
        'Referrer-Policy': 'strict-origin-when-cross-origin',
        'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https://cs.zhuxu.asia; frame-src 'none';",
        'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
        'Permissions-Policy': 'geolocation=(), camera=(), microphone=(), payment=(), usb=()'
      }
    },

    preview: {
      headers: {
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'SAMEORIGIN',
        'X-XSS-Protection': '1; mode=block',
        'Referrer-Policy': 'strict-origin-when-cross-origin',
        'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' https://cs.zhuxu.asia; frame-src 'none';",
        'Strict-Transport-Security': 'max-age=31536000; includeSubDomains',
        'Permissions-Policy': 'geolocation=(), camera=(), microphone=(), payment=(), usb=()'
      }
    },

    optimizeDeps: {
      include: [
        'vue',
        'vue-router',
        'pinia',
        'axios',
        'qs',
        'bootstrap',
        '@popperjs/core'
      ],
      exclude: ['@codemirror/lang-*', 'codemirror/mode-*']
    },

    build: {
      target: 'es2020',
      outDir: 'dist',
      assetsDir: 'static',
      cacheDir: false,
      minify: isProduction ? 'terser' : 'esbuild',
      sourcemap: false,
      chunkSizeWarningLimit: 1500,
      cssCodeSplit: 'divider',

      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('node_modules/vue') || id.includes('vue-router') || id.includes('pinia')) {
              return 'vue-vendor'
            }
            if (id.includes('bootstrap') || id.includes('@popperjs/core')) {
              return 'bootstrap-vendor'
            }
            if (id.includes('highlight.js')) {
              return 'highlight'
            }
            if (id.includes('@fancyapps/ui')) {
              return 'fancybox'
            }
            if (id.includes('marked')) {
              return 'marked'
            }
            if (id.includes('axios') || id.includes('crypto-js') || id.includes('qs')) {
              return 'utils-vendor'
            }
            if (id.includes('@vueuse/core')) {
              return 'vueuse'
            }
          },

          assetFileNames: 'static/[name].[hash:8].[ext]',
          chunkFileNames: 'static/js/[name].[hash:8].js',
          entryFileNames: 'static/js/[name].[hash:8].js'
        }
      },

      terserOptions: {
        compress: {
          drop_console: isProduction,
          drop_debugger: isProduction,
          dead_code: true,
          pure_funcs: ['console.log', 'console.warn', 'console.error']
        },
        format: {
          comments: false
        }
      },
    },
  }
})