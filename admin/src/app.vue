<template>
    <el-config-provider :locale="state.locale">
        <router-view></router-view>
    </el-config-provider>
</template>

<script setup>
import lang from 'element-plus/es/locale/lang/zh-cn'
// import lang from 'element-plus/lib/locale/lang/zh-cn'
const state = reactive({
    // 将默认语言设置为中文
    locale: lang
})

// 追加配置
globalThis.inis = {
    api: { uri: '', key: '' },
    cache: 10,
    lazy_time : 500,
    token_name: import.meta.env.VITE_TOKEN_NAME,
    ...globalThis.inis,
    version   : import.meta.env.VITE_VERSION,
}

</script>

<style>
.emoji-img {
    display: inline-block;
    vertical-align: middle;
    width: 20px;
    height: 20px;
    max-width: 20px;
    max-height: 20px;
    margin: 0 2px;
    object-fit: contain;
    pointer-events: none;
}

.limit-1-line .emoji-img,
.limit-2-lines .emoji-img {
    width: 16px;
    height: 16px;
}
</style>