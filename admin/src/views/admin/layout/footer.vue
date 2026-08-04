<template>
    <el-footer class="footer-wrapper">
        <div class="footer-content">
            <span>Copyright © {{ currentYear }} 管理后台 版权所有</span>
            <span class="footer-divider">|</span>
            <span>Powered by inis v{{ state.version }}</span>
        </div>
    </el-footer>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import cache from '{src}/utils/cache'
import axios from '{src}/utils/request'

const currentYear = new Date().getFullYear()

const state = reactive({
    version: '1.0.0',
})

onMounted(async () => {
    await fetchVersion()
})

const fetchVersion = async () => {
    const cacheName = 'system-version-local'
    if (cache.has(cacheName)) {
        state.version = cache.get(cacheName)
        return
    }
    const { code, data } = await axios.get('/dev/info/version')
    if (code === 200) {
        state.version = data?.inis
        cache.set(cacheName, data?.inis, inis.cache)
    }
}
</script>

<style lang="scss" scoped>
.footer-wrapper {
    background: #ffffff;
    border-top: 1px solid #e8e8e8;
    display: flex;
    align-items: center;
    justify-content: center;
    height: 48px;
    padding: 0 20px;
}

.footer-content {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #8c8c8c;
}

.footer-divider {
    color: #d9d9d9;
}

@media (max-width: 768px) {
    .footer-content {
        font-size: 11px;
        gap: 4px;
    }

    .footer-wrapper {
        padding: 0 12px;
    }
}
</style>
