## Cardify

### 📖 项目简介

Cardify 是一款基于 Vue 3 + JavaScript + Vite 开发的现代化博客主题，专为个人博客和内容创作者设计。

**设计理念**：简约而不简单，专注于内容的呈现，为读者提供沉浸式的阅读体验。

**技术栈**：
- **前端框架**：Vue 3 (Composition API)
- **构建工具**：Vite
- **路由管理**：Vue Router
- **状态管理**：Pinia
- **样式方案**：Bootstrap 5 + 自定义 CSS
- **Markdown渲染**：支持文章内容的 Markdown 解析

**核心优势**：
- **高性能**：Vite 构建，代码分割，懒加载等优化技术
- **响应式**：完美适配桌面、平板、手机等各种设备
- **现代化**：使用 Vue 3 最新特性，代码结构清晰
- **易部署**：支持多种部署平台，提供详细的部署指南
- **用户友好**：直观的界面设计，流畅的交互体验

## 功能特点

### 🎨 界面设计
- **完全响应式布局**：适配桌面、平板、手机等各种屏幕尺寸
- **现代美学风格**：简约单栏设计，注重内容可读性
- **深色模式支持**：自动检测系统主题，支持手动切换
- **平滑过渡动画**：按钮、标签等元素的悬停和点击效果

### 🚀 性能优化
- **基于 Vite 构建**：快速的开发服务器和优化的生产构建
- **懒加载技术**：图片和组件的按需加载，提升首屏加载速度
- **代码分割**：减小初始包大小，加快页面渲染
- **缓存策略**：合理的资源缓存，减少重复请求

### 📝 内容管理
- **文章展示**：支持Markdown格式，自动渲染文章内容
- **动态展示与发布**：支持完整的动态发布与展示
- **分类与标签**：文章分类管理和标签云展示
- **评论系统**：支持文章评论和回复功能
- **文章操作**：点赞、收藏、分享功能，带状态反馈

### 🔧 技术特性
- **Vue 3 Composition API**：使用最新的组合式API，代码结构清晰
- **Vue Router**：支持Hash和History两种路由模式
- **状态管理**：使用Pinia进行轻量级状态管理
- **API请求封装**：统一的网络请求处理，支持错误捕获

### 📱 用户体验
- **优雅的加载状态**：骨架屏和加载动画
- **即时反馈**：操作后的Toast提示
- **智能导航**：响应式导航栏，移动端侧边栏
- **无障碍访问**：符合WCAG标准的语义化HTML

### 🌐 部署便捷
- **多平台支持**：可自由部署到Vercel、Netlify、GitHub Pages等
- **服务器配置**：提供Nginx、Apache等服务器的部署配置示例
- **环境变量**：支持通过环境变量配置站点信息

### 🛠️ 开发体验
- **热更新**：开发过程中实时预览修改效果
- **ESLint集成**：代码质量检查，保持代码风格一致
- **模块化设计**：组件化开发，便于维护和扩展

## 开发
```
# 拉取代码
git clone https://github.com/zhu885744/Cardify.git

# 进入项目目录
cd Cardify

# 安装项目依赖
npm install

# 开启本地开发服务器（默认地址：http://localhost:3000，热更新实时预览）
npm run dev
```

## 打包
```
# 构建生产环境包（打包后生成 dist 目录）
npm run build

# 本地预览生产打包结果（验证打包后功能是否正常，无需部署服务器）
npm run preview
```

## 部署

将打包生成的 `dist` 目录内的所有文件上传到服务器主题目录即可。

### 路由模式部署指南

本项目支持两种路由模式：Hash模式和History模式。不同的路由模式需要不同的部署配置。

#### 1. Hash模式部署

Hash模式是Vue Router的默认模式，URL中会包含一个`#`符号，例如：`http://example.com/#/archives/1`。

**部署步骤：**
1. 确保`vite.config.js`或路由配置中使用Hash模式
2. 执行`npm run build`打包项目
3. 将生成的`dist`目录上传到服务器
4. 无需特殊服务器配置，直接访问即可

**优点：** 部署简单，无需服务器配置
**缺点：** URL中包含`#`符号，不够美观

#### 2. History模式部署

History模式使用HTML5 History API，URL更加干净，例如：`http://example.com/archives/1`。

**部署步骤：**
1. 在`vite.config.js`中配置基础路径：
   ```javascript
   export default defineConfig({
     base: '/', // 填写你的网站根路径
     // 其他配置...
   })
   ```

2. 在路由配置中使用History模式：
   ```javascript
   const router = createRouter({
     history: createWebHistory(),
     routes: [
       // 路由配置...
     ]
   })
   ```

3. 执行`npm run build`打包项目

4. 将生成的`dist`目录上传到服务器

5. **配置服务器重写规则**（关键步骤）：

   **Nginx配置示例：**
   ```nginx
   server {
     listen 80;
     server_name example.com;
     root /path/to/your/dist;
     index index.html;

     location / {
       try_files $uri $uri/ /index.html;
     }
   }
   ```

   **Apache配置示例：**
   ```apache
   <IfModule mod_rewrite.c>
     RewriteEngine On
     RewriteBase /
     RewriteRule ^index\.html$ - [L]
     RewriteCond %{REQUEST_FILENAME} !-f
     RewriteCond %{REQUEST_FILENAME} !-d
     RewriteRule . /index.html [L]
   </IfModule>
   ```

   **GitHub Pages部署：**
   - 使用Hash模式或
   - 在根目录创建`404.html`文件，内容与`index.html`相同

**优点：** URL美观，符合RESTful风格
**缺点：** 需要服务器配置，否则刷新页面会出现404错误

### 部署平台推荐

- **Vercel**：零配置部署，支持History模式
- **Netlify**：简单易用，支持自动部署
- **GitHub Pages**：适合静态网站，推荐使用Hash模式
- **阿里云/腾讯云**：适合企业级应用，需要手动配置服务器

### 开源许可
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)