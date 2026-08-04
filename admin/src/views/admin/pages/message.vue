<template>
    <div class="container-box">
        <el-result icon="info" title="消息通知" sub-title="当前页面开发中">
            <template #extra>
                <el-button type="primary" @click="$router.back()">返回</el-button>
            </template>
        </el-result>
    </div>
</template>

<script setup>
</script>
