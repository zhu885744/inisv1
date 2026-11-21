<template>
  <!--
    创作中心文章编辑器：默认只能编辑自己的文章，返回列表为 /manage/posts
    编辑器本体与后台共用，统一放在 components/admin 下维护（后台为其超集）
  -->
  <ContentWriteForm mode="article" base-path="/manage/posts" />
</template>

<script setup>
import ContentWriteForm from '@/components/admin/ContentWriteForm.vue'
</script>
