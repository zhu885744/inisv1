<template>
  <!--
    管理后台文章编辑器
    复用 /manage/posts/write 的完整功能（Markdown 编辑器、封面、摘要、话题、分类、评论开关、审核状态等），
    差异：返回列表为 /admin/article；管理员可编辑任意作者的文章。
  -->
  <ContentWriteForm
    mode="article"
    base-path="/admin/article"
    edit-base="/admin/article/edit"
    can-edit-others
  />
</template>

<script setup>
import ContentWriteForm from '@/components/admin/ContentWriteForm.vue'
</script>
