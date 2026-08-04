<template>
    <i-nav ref="navRef"></i-nav>
    <div class="container width">
      <router-view key="$route.fullPath" v-slot="{ Component }">
        <transition name="slide-fade" mode="out-in">
          <div class="router-view-wrapper">
            <component :is="Component" />
          </div>
        </transition>
      </router-view>
    </div>
  <i-footer></i-footer>
  <i-float-buttons></i-float-buttons>
  <upgrade-page></upgrade-page>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import upgradePage from '@/comps/upgrade/page.vue'
import iNav from '@/views/index/layout/nav.vue'
import iFooter from '@/views/index/layout/footer.vue'
import iFloatButtons from '@/comps/custom/i-float-buttons.vue'
import { useCommStore } from '@/store/comm'
import { request } from '@/utils/network'

const store = useCommStore()
const customCodeInjected = ref(false)

const injectCustomCode = async () => {
  if (customCodeInjected.value) return
  
  try {
    const cacheKey = 'custom_code_config'
    const cachedConfig = localStorage.getItem(cacheKey)
    let config = {}
    
    if (cachedConfig) {
      const parsed = JSON.parse(cachedConfig)
      if (Date.now() - parsed.timestamp < 24 * 60 * 60 * 1000) {
        config = parsed.data
      }
    }
    
    if (!Object.keys(config).length) {
      const response = await request.get('/api/config/one', { key: 'cardify_functions' })
      if (response.code === 200 && response.data) {
        config = response.data.json || {}
        localStorage.setItem(cacheKey, JSON.stringify({
          data: config,
          timestamp: Date.now()
        }))
      }
    }
    
    const customCodeData = config.custom_code || {}
    
    if (customCodeData.css) {
      const style = document.createElement('style')
      style.textContent = customCodeData.css
      style.setAttribute('data-custom-css', 'true')
      document.head.appendChild(style)
    }
    
    if (customCodeData.header) {
      const tempDiv = document.createElement('div')
      tempDiv.innerHTML = customCodeData.header
      while (tempDiv.firstChild) {
        document.head.appendChild(tempDiv.firstChild)
      }
    }
    
    if (customCodeData.js || customCodeData.footer || customCodeData.analytics) {
      const scripts = []
      if (customCodeData.js) scripts.push({ type: 'js', content: customCodeData.js })
      if (customCodeData.footer) scripts.push({ type: 'footer', content: customCodeData.footer })
      if (customCodeData.analytics) scripts.push({ type: 'analytics', content: customCodeData.analytics })
      
      setTimeout(() => {
        scripts.forEach(({ content }) => {
          const tempDiv = document.createElement('div')
          tempDiv.innerHTML = content
          while (tempDiv.firstChild) {
            document.body.appendChild(tempDiv.firstChild)
          }
        })
      }, 100)
    }
    
    customCodeInjected.value = true
  } catch (error) {
    console.error('注入自定义代码失败:', error)
  }
}

const initAfterMount = async () => {
  await injectCustomCode()
}

onMounted(async () => {
  await initAfterMount()
})
</script>

<style>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-fade-enter-active {
  transition: all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.slide-fade-leave-active {
  transition: all 0.25s ease-in;
}

.slide-fade-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-fade-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.router-view-wrapper {
  width: 100%;
  min-height: calc(100vh - 200px);
}

@media (prefers-reduced-motion: reduce) {
  .fade-enter-active,
  .fade-leave-active,
  .slide-fade-enter-active,
  .slide-fade-leave-active {
    transition: none;
  }
}
</style>