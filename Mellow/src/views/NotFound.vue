<template>
  <div class="not-found">
    <div class="big-code">404</div>
    <div class="text">您访问的页面不存在</div>
    <router-link to="/" class="btn btn-primary">返回首页</router-link>
  </div>
</template>

<style scoped>
.not-found {
  text-align: center;
  padding: 96px 16px;
}
.big-code {
  font-family: var(--font-serif);
  font-size: 96px;
  font-weight: 700;
  color: var(--primary);
  line-height: 1;
}
.text {
  font-size: 16px;
  color: var(--text-muted);
  margin: 16px 0 24px;
}
</style>
