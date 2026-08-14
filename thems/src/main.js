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
  
  // 通知加载页可以隐藏了，无需等待 window.onload
  window.dispatchEvent(new Event('app-mounted'))
  
  // 站点信息、全局工具与登录态校验相互独立，并行执行缩短首屏可用时间。
  // 登录态在此统一校验一次，页面与组件后续直接读取 store，不再各自发起校验。
  await Promise.all([
    setupGlobalTools(app),
    setupSiteInfo(commStore),
    commStore.ensureLogin().catch(err => logError('登录态校验失败:', err))
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