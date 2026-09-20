# 前端主题开发及 API 调用规范

## 一、前端主题开发规范

分类、标签、文章等路径规范
/archives/:id    文章详情
/:key            独立页面
/category/:key   分类详情
/tag/:key        标签详情


表情包渲染规范
/api/attachment/emoji   后端提供的表情包列表api
需要按照下面规范，后端才能渲染表情
[emoji:表情链接]