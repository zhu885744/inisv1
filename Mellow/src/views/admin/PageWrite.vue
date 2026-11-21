<template>
  <!--
    管理后台独立页面编辑器
    与文章编辑器共用 components/admin/ContentWriteForm.vue（mode=page）：
    Markdown 编辑器、话题、评论开关、审核状态一致；
    字段差异为「唯一标识 key + 备注 remark」，且没有草稿/发布之分（仅保存）。
  -->
  <ContentWriteForm
    mode="page"
    base-path="/admin/pages"
    edit-base="/admin/pages/edit"
    can-edit-others
  />
</template>

<script setup>
import ContentWriteForm from '@/components/admin/ContentWriteForm.vue'
</script>
