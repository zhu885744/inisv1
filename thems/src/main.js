import { createApp } from 'vue'
import App from './App.vue'
import { createPinia } from 'pinia'
import router from './router'
import { useCommStore } from './store/comm'
import iSvg from './comps/custom/i-svg.vue'
import './assets/css/bootstrap.css'
import './assets/css/style.css'
import 'bootstrap-icons/font/bootstrap-icons.css'
import '@fancyapps/ui/dist/fancybox/fancybox.css'
import 'virtual:svg-icons-register'
import Region from 'v-region'
import { log, logError, init as setupGlobalTools, setupSiteInfo } from './utils/app'

const createAndConfigureApp = async (isRetry = false) => {
  log(isRetry ? '重试初始化应用...' : '开始初始化应用...')
  
  const app = createApp(App)
  const pinia = createPinia()
  
  app.use(pinia)
  app.use(router)
  app.use(Region) // v-region 

  // 全局组件注册
  app.component('iSvg', iSvg)
  
  const commStore = useCommStore()
  
  await router.isReady()
  
  // 挂载页面
  app.mount('#app')
  
  await Promise.all([
    setupGlobalTools(app),
    setupSiteInfo(commStore)
  ])
  
  log('应用初始化完成')
}

async function initApp() {
  log('配置文件已加载')
  
  try {
    await createAndConfigureApp()
  } catch (error) {
    logError('应用初始化失败:', error)
    
    try {
      await createAndConfigureApp(true)
    } catch (innerError) {
      logError('重试启动应用失败:', innerError)
    }
  }
}

initApp()