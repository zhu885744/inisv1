# MELLOW 主题

基于 Vue 3 + Vite + 纯 HTML/CSS/JS 开发的超强三栏INIS主题

## 功能模块

- 首页（导航卡片 + 最新文章）
- 文章列表 / 文章详情（含评论）
- 动态（Moments）瀑布流
- 友链（按分组）
- 登录 / 注册 / 找回密码 / 重置密码
- 用户中心（资料、收藏、点赞、动态、通知、签到、注销）

## 技术栈

- **Vue 3** + Vue Router + Pinia + vite
- **Vite** 构建
- **Axios** 请求
- **markdown-it** 渲染
- 纯 CSS（CSS 变量 + 响应式）

## 所用到的依赖
```bash
"axios": "^1.6.8",
"bootstrap-icons": "^1.13.1",
"crypto-js": "^4.2.0",
"dayjs": "^1.11.10",
"highlight.js": "^11.9.0",
"markdown-it": "^14.0.0",
"pinia": "^2.1.7",
"vue": "^3.4.21",
"vue-router": "^4.3.0"
```  

## 开发
拉取源码
```bash
git clone https://github.com/zhuxu-dev/mellow-theme.git
```  

## 启动

```bash
npm install
npm run dev
```

默认开发端口 `5173`，通过 `vite.config.js` 中的 `proxy` 转发 `/api` 到后端如`http://localhost:8000`。

## 构建

```bash
npm run build
```

生成静态文件至 `dist/`，可作为静态资源部署至任何 CDN / Nginx。