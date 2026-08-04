import path from 'path'
import { loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createHtmlPlugin } from 'vite-plugin-html'
import { createSvgIconsPlugin } from 'vite-plugin-svg-icons'

// element-plus 自动按需引入
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import ElementPlus from 'unplugin-element-plus/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

import { visualizer } from 'rollup-plugin-visualizer'

// 修复路径：确保 vite.plugin.js 存在于项目根目录
import { externals } from './vite.plugin.js' // 补充 .js 后缀，避免路径解析问题

// https://vitejs.dev/config/
export default ({ mode }) => {
    // 加载环境变量（修复解构赋值，避免 VITE_LIBS 初始值为空字符串）
    const env = { 
        ...loadEnv(mode, process.cwd()), // 改用 process.cwd() 更稳定
        VITE_LIBS: '' 
    }

    // CDN 配置优化
    const cdn = {
        name: (env.VITE_CDN || '').toLowerCase(), // 统一转小写，简化后续判断
        path: '//unpkg.com',
    }

    // 处理CDN
    if (cdn.name) {
        switch (cdn.name) {
            case 'jsdelivr':
                cdn.path = '//cdn.jsdelivr.net/npm'
                break
            case 'unpkg':
            default:
                cdn.path = '//unpkg.com'
                break
        }
    }

    const libs = {
        css: [''],
        js: [
            '/vue@3/dist/vue.global.js',
            '/vuex@4.0.0/dist/vuex.global.js',
            '/vue-router@4.1.6/dist/vue-router.global.js',
            '/axios/dist/axios.min.js',
            '/@vueuse/shared'
        ],
    }

    // 生成 CDN 引入代码（优化逻辑，避免空值处理）
    if (cdn.name) {
        libs.css = libs.css.map(item => `${cdn.path}${item}`)
        libs.js = libs.js.map(item => `${cdn.path}${item}`)

        let html = ''
        libs.css.forEach(item => {
            html += `<link rel="stylesheet" href="${item}">\n`
        })
        libs.js.forEach(item => {
            html += `<script src="${item}" defer></script>\n`
        })

        env.VITE_LIBS = html
    }

    // 时间戳
    const timestamp = Math.round(new Date() / 1000)
    // 修复环境变量判断（mode 更准确，process.env.NODE_ENV 可能未生效）
    const base_url = mode === 'production' ? (env.VITE_BASE || '/') : '/'

    // 插件配置（修复 inject 语法错误）
    const plugins = [
        vue(),
        createSvgIconsPlugin({
            // 指定需要缓存的图标文件夹（优化路径写法）
            iconDirs: [path.resolve(process.cwd(), 'src/assets/svg')],
            // 指定symbolId格式
            symbolId: 'icon-[dir]-[name]',
            /**
             * 自定义插入位置（修复 TypeScript 类型写法，改为字符串）
             */
            inject: 'body-last', // 可选值：'body-last' | 'body-first'
            /**
             * custom dom id
             * @default: __svg__icons__dom__
             */
            customDomId: '__svg__icons__dom__',
        }),
        AutoImport({
            // 自动导入 - 全局
            imports: ['vue', 'vue-router'],
            // 生成全局变量名
            dts: 'src/auto-imports.d.ts',
            // element-plus - 自动按需引入 - 注册中文
            resolvers: [ElementPlusResolver()], // 启用 Element Plus 自动导入
        }),
        Components({
            // element-plus - 自动按需引入 - 注册组件
            resolvers: [ElementPlusResolver()], // 启用 Element Plus 组件自动导入
        }),
        ElementPlus({
            // 支持 Element Plus 主题定制
            useSource: true,
        }),
        createHtmlPlugin({
            inject: {
                data: { ...env },
            },
        }),
        // 打包优化（按需启用）
        // visualizer(),
    ]

    // CDN 模式下才添加 externals 插件
    if (cdn.name) {
        plugins.push(externals())
    }

    return {
        base: base_url,
        plugins: plugins,
        // 修复路径别名配置（核心！）
        resolve: {
            alias: {
                // 正确写法：@ 指向 src 目录，末尾不要多余 /
                '@': path.resolve(process.cwd(), 'src'),
                // 可选：保留你的自定义别名（如果需要）
                '{src}': path.resolve(process.cwd(), 'src'),
            },
            // 补充：添加扩展名自动解析，避免导入时漏写 .vue/.js
            extensions: ['.vue', '.js', '.jsx', '.json'],
        },
        build: {
            minify: 'terser',
            // 打包后的文件名
            outDir: 'dist',
            // 打包后的静态资源文件夹名称
            assetsDir: 'static',
            // 分包大小警告阈值
            chunkSizeWarningLimit: 2000,
            terserOptions: {
                compress: {
                    // 移除console
                    drop_console: true,
                    // 移除debugger
                    drop_debugger: true,
                }
            },
            // 打包后更改文件名
            rollupOptions: {
                output: {
                    assetFileNames: asset => {
                        const info = asset.name.split('.')
                        const ext = info[info.length - 1]
                        let folder = 'other'
                        if (/\.(mp4|webm|ogg|mp3|wav|flac|aac)(\?.*)?$/i.test(asset.name)) {
                            folder = 'media'
                        } else if (/\.(png|jpe?g|gif|svg)(\?.*)?$/.test(asset.name)) {
                            folder = 'images'
                        } else if (/\.(woff2?|eot|ttf|otf)(\?.*)?$/i.test(asset.name)) {
                            folder = 'font'
                        } else if (ext === 'css') {
                            folder = 'css'
                        } else if (ext === 'js') {
                            folder = 'js'
                        }
                        return `static/${folder}/${timestamp}.[name][extname]`
                    },
                    chunkFileNames: `static/js/${timestamp}.[name].js`,
                    entryFileNames: `static/js/${timestamp}.[name].js`,
                    manualChunks(id) {
                        if (id.includes('node_modules')) {
                            return id.toString().split('node_modules/')[1].split('/')[0].toString()
                        }
                    }
                },
            },
        },
        server: {
            // 热更新优化
            hmr: {
                overlay: true, // 报错时显示覆盖层（可关闭：false）
            },
            // 代理
            proxy: {
                '/api': {
                    target: env.VITE_API,
                    changeOrigin: true,
                    // 可选：添加路径重写（如果需要）
                    // rewrite: (path) => path.replace(/^\/api/, ''),
                },
                '/dev': {
                    target: env.VITE_API,
                    changeOrigin: true,
                },
                '/inis': {
                    target: env.VITE_API,
                    changeOrigin: true,
                },
                '/socket.io': {
                    target: env.VITE_SOCKET,
                    ws: true,
                    changeOrigin: true,
                }
            },
        }
    }
}