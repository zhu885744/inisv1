<template>
    <!-- 后台静默检测版本更新，无需渲染 UI -->
</template>

<script setup>
const { proxy } = getCurrentInstance()

// 上一次的脚本列表（用于对比是否更新）
let lastScripts = null

// 检查更新并通知用户
async function checkUpdate() {
    try {
        const scripts = await fetchScripts()
        
        // 首次加载，记录初始状态
        if (!lastScripts) {
            lastScripts = scripts
            schedule()
            return
        }
        
        // 对比脚本数量或内容变化
        const hasUpdate = lastScripts.length !== scripts.length 
            || lastScripts.some((s, i) => s !== scripts[i])
        
        lastScripts = scripts
        
        if (hasUpdate) {
            ElNotification({
                type: 'success',
                title: '系统更新完成！',
                duration: 0,
                dangerouslyUseHTMLString: true,
                message: `<div class="notify-content">
                    <span style="margin-right: 8px">检测到新版本，刷新页面体验新功能？</span>
                    <el-button onclick="location.reload()" type="primary" size="small" class="refresh-btn">立即刷新</el-button>
                </div>`,
                position: 'bottom-right',
            })
        }
    } catch (e) {
        // 请求失败静默处理，下次继续尝试
    }
    schedule()
}

// 获取页面中所有 script 标签的 src 地址
async function fetchScripts() {
    const reg = /<script[^>]+src=["']([^"']+)["']/gi
    const html = await fetch('/?_t=' + Date.now()).then(res => res.text())
    const scripts = []
    let match
    while ((match = reg.exec(html)) !== null) {
        scripts.push(match[1])
    }
    return scripts
}

// 定时调度检查
function schedule() {
    const delay = (globalThis.inis?.lazy_time ?? 500) * 10 * 30
    setTimeout(checkUpdate, delay)
}

onMounted(() => {
    if (import.meta.env.PROD) checkUpdate()
})
</script>
