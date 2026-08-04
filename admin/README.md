# inis-admin

## 一、介绍

inis-admin 是一个基于 Vue 3 + Vite 构建的后台管理项目，采用 `<script setup>` 单文件组件语法进行开发，包含前台展示和后台管理功能。项目结构清晰，功能模块完善，支持安装引导、首页文章列表展示、文章管理、用户管理、系统配置等核心功能，适用于内容管理类应用场景。

### 核心功能模块
- **前台模块**：首页展示、文章详情、三方登录、图标展示等
- **后台模块**：用户管理、内容管理（文章/页面/评论）、系统配置（权限/IP黑名单/QPS预警等）
- **辅助功能**：安装引导、权限校验、路由守卫等

## 二、部署方法

### 1. 环境准备
- 确保已安装 Node.js（推荐 v16+）及包管理工具（npm/yarn/pnpm）
- 生产环境需准备部署载体（静态托管平台/Nginx服务器/后端项目）

### 2. 部署步骤

#### 步骤1：获取项目代码
克隆或下载项目源码到本地/服务器

#### 步骤2：安装依赖
```bash
cd inis-admin
npm install
```

#### 步骤3：打包生产环境代码
```bash
npm run build  # 生成 dist 文件夹
```

#### 步骤4：选择部署方式

| 部署方式       | 操作说明                                                                 |
|----------------|--------------------------------------------------------------------------|
| 静态托管平台   | 直接将 `dist` 文件夹上传至 GitHub Pages、Netlify、Vercel 等平台          |
| Nginx 服务器   | 1. 将 `dist` 文件夹复制到 Nginx 根目录（如 `/usr/share/nginx/html`）<br>2. 配置 Nginx 指向该目录并重启服务 |
| 后端集成部署   | 将 `dist` 文件夹复制到后端项目的静态资源目录，由后端框架转发请求          |

### 3. 部署注意事项
- 若部署到非根目录（如 `https://xxx.com/admin/`），需修改 `vite.config.js`：
  ```javascript
  export default defineConfig({
    base: '/admin/'  // 配置为实际部署的子路径
  })
  ```
- 生产环境会加载 `/config.js` 配置文件，需确保该文件存在且配置正确


## 三、打包方法

### 1. 开发环境打包（用于预览）
```bash
npm run build  # 生成 dist 文件夹
npm run preview  # 本地预览打包结果
```

### 2. 生产环境打包优化
项目通过 `vite.plugin.js` 配置了外部化处理，减小打包体积：
```javascript
// 外部化常用库，通过 CDN 加载
export const externals = () => {
    return viteExternalsPlugin({
        'vue': 'Vue',
        'vuex': 'Vuex',
        'vue-router': 'VueRouter',
        'axios': 'axios',
        'bootstrap': 'bootstrap',
        'vue-demi': 'VueDemi',
    })
}
```

### 3. 打包输出
打包完成后，所有静态资源（HTML/CSS/JS/图片等）会输出到 `dist` 文件夹，可直接用于部署。

## 四、技术栈与框架

### 1. 核心框架
- **Vue 3**：前端核心框架，使用 `<script setup>` 语法简化组件开发
- **Vite**：构建工具，提供极速开发体验和优化的构建流程
- **Vue Router 4**：路由管理，处理页面跳转和权限控制

### 2. 状态管理
- **Pinia**：Vue 官方推荐的状态管理库，替代 Vuex
- **Vuex**：兼容使用的状态管理方案（通过外部化配置）

### 3. UI 组件库
- **Element Plus**：基于 Vue 3 的企业级 UI 组件库

### 4. 网络与数据交互
- **Axios**：HTTP 客户端，处理 API 请求
- **qs**：URL 参数序列化/反序列化工具

### 5. 开发辅助工具
- **unplugin-auto-import**：自动导入 Vue API 和组件
- **unplugin-vue-components**：自动注册组件，减少手动声明
- **vite-plugin-externals**：打包时外部化处理依赖
- **Sass**：CSS 预处理器，增强样式编写能力

### 6. 功能扩展库
- **echarts**：数据可视化图表库
- **md-editor-v3**：Markdown 编辑器 https://imzbf.github.io/md-editor-v3/zh-CN/about
- **plyr**：视频播放器
- **lottie-web**（`vue3-lottie`）：动画渲染库
- **highlight.js**：代码高亮库
- **crypto-js**：加密算法库
- **@fancyapps/ui**：图片查看器等交互组件

### 7. 其他工具
- **js-base64**：Base64 编码解码工具
- **lscache**：本地存储管理工具
- **animejs**：动画库
- **colorthief**：提取图片主色调工具