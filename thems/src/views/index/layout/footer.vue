<template>
  <footer id="footer" class="fs-6 py-3 mt-2 bg-body-tertiary text-body-secondary">
    <div class="container">
      <div class="text-center">
        <!-- 版权年份 -->
        <div class="text">
          Copyright © {{ startYear || '2020' }} ~ {{ currentYear }} {{ siteTitle }} 版权所有
        </div>
        
        <!-- ICP备案号 -->
        <div class="test" v-if="hasIcp">
          <a 
            :href="icpLink" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            aria-label="ICP备案号"
          >
            {{ icpCode }}
          </a>
        </div>
        
        <!-- 公安备案号 -->
        <div class="test" v-if="hasPolice">
          <a 
            :href="policeLink" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            aria-label="公安备案号"
          >
            {{ policeCode }}
          </a>
        </div>
        
        <!-- 技术支持 -->
        <div class="text" aria-label="技术支持">
          <span>Powered by </span>
          <a 
            href="https://github.com/zhu885744/inisv1" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            title="inisv1 开源地址"
          >
            inis
          </a>
          <span class="mx-1">|</span>
          <span>Theme by </span>
          <a 
            href="https://github.com/zhu885744/Cardify" 
            target="_blank" 
            rel="noopener noreferrer"
            class="text-decoration-none text-reset hover-text-primary transition-opacity"
            title="Cardify"
          >
            Cardify v{{ themeVersion }}
          </a>
        </div>
      </div>
    </div>
  </footer>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useCommStore } from '@/store/comm'

// 环境变量
const THEME_VERSION = import.meta.env.VITE_VERSION || '1.0.0'

// 响应式数据
const currentYear = new Date().getFullYear()
const commStore = useCommStore()

// 直接使用store中的siteInfo
const siteInfo = ref(commStore.siteInfo)

const startYear = computed(() => {
  const timestamp = siteInfo.value?.date
  if (!timestamp) return currentYear
  
  try {
    const milliseconds = parseInt(timestamp) * 1000
    const date = new Date(milliseconds)
    const year = date.getFullYear()
    return isNaN(year) ? currentYear : year
  } catch (error) {
    return currentYear
  }
})

const siteTitle = computed(() => {
  return siteInfo.value?.title || '未设置网站名'
})

// ICP备案
const hasIcp = computed(() => !!siteInfo.value?.copy?.code)
const icpCode = computed(() => siteInfo.value?.copy?.code || '请在后台填写备案号')
const icpLink = computed(() => siteInfo.value?.copy?.link || 'https://beian.miit.gov.cn/#/Integrated/index')

// 公安备案
const hasPolice = computed(() => !!siteInfo.value?.police?.code)
const policeCode = computed(() => siteInfo.value?.police?.code || '请在后台填写公安备案号')
const policeLink = computed(() => siteInfo.value?.police?.link || 'https://beian.mps.gov.cn/#/query/webSearch')

const themeVersion = computed(() => THEME_VERSION)

// 组件挂载
onMounted(async () => {
  if (!siteInfo.value || Object.keys(siteInfo.value).length === 0) {
    await commStore.fetchSiteInfo()
    siteInfo.value = commStore.siteInfo
  }
})
</script>

<style scoped>
.hover-text-primary:hover {
  color: var(--bs-primary) !important;
}

.transition-opacity {
  transition: opacity 0.2s ease;
}

.transition-opacity:hover {
  opacity: 0.8;
}
</style>