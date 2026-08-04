<!-- 主题更新成功提示组件 -->
<template>

</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { toast } from '@/utils/app'

// 使用普通对象，确保不是响应式的
const state = {
    lastScripts: [],
    pollingInterval: null,
}

// 默认轮询时间（毫秒）
const DEFAULT_POLLING_INTERVAL = 30000

// 初始化__updateChecker对象
if (typeof window !== 'undefined') {
    // 确保__updateChecker对象存在
    window.__updateChecker = window.__updateChecker || {}
    
    // 直接暴露lastScripts，使用引用
    window.__updateChecker.lastScripts = state.lastScripts
    
    // 暴露方法
    window.__updateChecker.checkUpdate = checkUpdate
    window.__updateChecker.startPolling = startPolling
    window.__updateChecker.stopPolling = stopPolling
    window.__updateChecker.getLastScripts = () => {
        // console.log('getLastScripts called, state.lastScripts:', state.lastScripts)
        // 确保返回的是普通数组，不是Proxy
        return Array.isArray(state.lastScripts) ? [...state.lastScripts] : []
    }
    
    // console.log('检查更新功能已初始化，可通过 window.__updateChecker 访问')
    // console.log('查看当前存储的脚本URL: window.__updateChecker.lastScripts 或 window.__updateChecker.getLastScripts()')
    // console.log('检查更新: window.__updateChecker.checkUpdate()')
    // console.log('初始化时state.lastScripts:', state.lastScripts)
    // console.log('初始化时window.__updateChecker.lastScripts:', window.__updateChecker.lastScripts)
}

onMounted(async () => {
    // 仅在生产环境下执行
    if (import.meta.env.PROD) {
        startPolling()
    }
})

onUnmounted(() => {
    stopPolling()
})

// 开始轮询
function startPolling() {
    // console.log('开始轮询检查更新...')
    // 立即执行一次检查
    checkUpdate()
    
    // 设置定时轮询
    state.pollingInterval = setInterval(() => {
        // 使用setTimeout包装，避免阻塞
        setTimeout(() => {
            checkUpdate()
        }, 0)
    }, DEFAULT_POLLING_INTERVAL)
}

// 停止轮询
function stopPolling() {
    if (state.pollingInterval) {
        clearInterval(state.pollingInterval)
        state.pollingInterval = null
        // console.log('停止轮询检查更新')
    }
}

// 检查更新
async function checkUpdate() {
    try {
        // console.log('开始检查更新...')
        const scripts = await getScriptUrls()
        // console.log('获取到的脚本URL:', scripts)
        const hasUpdate = compareScripts(scripts)
        // console.log('是否有更新:', hasUpdate)
        
        if (hasUpdate) {
            // console.log('检测到更新，显示通知')
            showUpdateNotification()
        } else {
            // console.log('未检测到更新')
        }
    } catch (error) {
        // console.error('检查更新失败:', error)
    }
}

// 获取页面中的所有script标签URL
async function getScriptUrls() {
    let result = []
    let match  = null
    const reg  = /<script.*src=["'](?<src>[^"]+)/gm
    
    try {
        // 使用相对路径，避免跨域问题
        const currentPath = window.location.pathname
        const url = currentPath + (currentPath.includes('?') ? '&' : '?') + 't=' + Date.now()
        
        const html = await fetch(url, {
            cache: 'no-cache',
            headers: {
                'Cache-Control': 'no-cache',
                'Pragma': 'no-cache',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
                'Accept-Language': 'en-US,en;q=0.5',
                'Connection': 'keep-alive',
                'Upgrade-Insecure-Requests': '1'
            },
            credentials: 'same-origin'
        }).then(res => {
            if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`)
            return res.text()
        })
        
        reg.lastIndex = 0
        while ((match = reg.exec(html)) !== null) {
            result.push(match.groups.src)
        }
        // console.log('提取script标签完成:', result.length, '个脚本')
    } catch (error) {
        // console.error('获取页面HTML失败:', error)
        // 失败时返回空数组，避免后续逻辑出错
        return []
    }
    
    return result
}

// 比较脚本URL是否有变化
function compareScripts(scripts) {
    if (!state.lastScripts.length) {
        // 第一次检查，存储脚本列表
        // console.log('首次检查，存储脚本列表')
        state.lastScripts = scripts
        // 更新window上的引用
        if (typeof window !== 'undefined' && window.__updateChecker) {
            window.__updateChecker.lastScripts = state.lastScripts
        }
        return false
    }
    
    // 检查脚本数量是否变化
    if (state.lastScripts.length !== scripts.length) {
        // console.log('脚本数量变化:', state.lastScripts.length, '->', scripts.length)
        state.lastScripts = scripts
        // 更新window上的引用
        if (typeof window !== 'undefined' && window.__updateChecker) {
            window.__updateChecker.lastScripts = state.lastScripts
            // console.log('更新window.__updateChecker.lastScripts:', window.__updateChecker.lastScripts)
        }
        return true
    }
    
    // 检查脚本URL是否变化
    for (let i = 0; i < state.lastScripts.length; i++) {
        if (state.lastScripts[i] !== scripts[i]) {
            // console.log('脚本URL变化:', state.lastScripts[i], '->', scripts[i])
            state.lastScripts = scripts
            // 更新window上的引用
            if (typeof window !== 'undefined' && window.__updateChecker) {
                window.__updateChecker.lastScripts = state.lastScripts
                // console.log('更新window.__updateChecker.lastScripts:', window.__updateChecker.lastScripts)
            }
            return true
        }
    }
    
    // console.log('脚本无变化')
    return false
}

// 显示更新通知
function showUpdateNotification() {
    // console.log('显示更新通知...')
    toast.success(`
        <div class="d-flex flex-column align-items-start">
            <h6 class="fw-bold mb-1 d-flex align-items-center gap-2"><i class="bi bi-stars"></i>主题更新完成！</h6>
            <p class="mb-2">刷新页面体验新功能？</p>
            <button onclick="location.reload()" type="button" class="wx-btn-gradient w-100">刷 新</button>
        </div>
    `, {
        autohide: false,
        delay: 0,
        showClose: false
    })
}

</script>