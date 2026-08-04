<template>
    <el-card style="margin-bottom: 1rem" v-loading="state.status.init">
        <template #header>
            <div class="card-header-content">
                <el-tooltip placement="top">
                    <template #content>
                        <strong style="color: var(--el-color-success)">inis-admin也就是当前您看到的后台界面</strong><br>
                        ● inis-admin一般情况下是自动升级的，在超级管理员权限下检查到新版本会自动升级<br>
                        ● 自动升级检查10分钟会检查一次，也可以通过当前卡片的功能手动检查并进行升级
                    </template>
                    <span>inis-admin「开发中...」</span>
                </el-tooltip>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap">
                    <el-button v-on:click="method.init(false)" :loading="state.status.loading" type="primary" size="small" round plain>
                        <span>检 查 更 新</span>
                    </el-button>
                    <el-button v-if="state.show.upgrade" v-on:click="method.upgrade" :loading="state.status.upgrade" type="primary" size="small" round plain style="margin-left: 0.5rem">
                        <span>升 级</span>
                    </el-button>
                    <el-button v-on:click="method.show" type="primary" size="small" round style="margin-left: 0.5rem">
                        <span>日 志</span>
                    </el-button>
                </div>
            </div>
            <div style="display: flex; justify-content: space-between; margin-top: 0.5rem">
                <span>当前版本：{{ state.version }}</span>
                <span>最新版本：{{ state.struct.version || '∞' }}</span>
            </div>
        </template>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong>更新日志</strong>
        </template>
        <template #default>
            <div v-if="!utils.is.empty(state.struct)" style="width: 100%">
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 0 0.5rem; margin-bottom: 0.5rem">
                    <el-tooltip :content="`${state.struct.result?.author?.nickname}：${state.struct.result?.author?.description}`" placement="top">
                        <div style="display: flex; align-items: center; cursor: pointer">
                            <el-avatar :src="state.struct.result?.author?.avatar" :size="25" style="box-shadow: var(--el-box-shadow-light); transform: scaleX(-1)"></el-avatar>
                            <span style="font-size: 13px; margin-left: 0.25rem">{{ state.struct.result?.author?.nickname }}</span>
                        </div>
                    </el-tooltip>
                    <span style="font-size: 13px">时间：{{ utils.time.to.date(state.struct?.create_time) }}</span>
                </div>
                <el-alert type="success" :closable="false" style="display: flex; justify-content: space-between; margin-bottom: 0.5rem">
                    <span>{{ state.struct.title }}</span>
                    <span style="display: flex; align-items: center">
    
                        {{ method.color(state.struct.progress).text }}：{{ state.struct.version }}
                    </span>
                </el-alert>
                <div v-if="!utils.is.empty(state.struct.content)" style="margin-top: 0.5rem" class="markdown">
                    <el-scrollbar max-height="400px">
                        <div v-html="method.markdown(state.struct.content)" class="white-space-line"></div>
                    </el-scrollbar>
                </div>
            </div>
            <div v-else style="width: 100%">
                <el-alert type="success" :closable="false" style="text-align: center">
                    无更新日志
                </el-alert>
            </div>
        </template>
        <template #footer>
            <el-button v-on:click="state.status.dialog = false">
                取 消
            </el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import MarkdownIt from 'markdown-it'
import utils from '{src}/utils/utils'

const emit = defineEmits(['refresh'])
const { ctx, proxy } = getCurrentInstance()
const state = reactive({
    // 清空所有模拟数据，仅保留空结构
    struct: {},
    status: {
        init: false, // 关闭初始加载，仅保留样式
        finish: false,
        dialog: false,
        loading: false,
        upgrade: false,
    },
    show: {
        upgrade: false,
    },
    version: '1.0.0', // 清空版本号，仅保留展示位置
})

const method = {
    // 简化init方法，仅更新状态，无逻辑、无提示、无延迟
    init: async (init = true) => {
        state.status.init = init
        state.status.loading = true
        state.status.init = false
        state.status.loading = false
    },
    // 简化升级方法，仅更新状态，无逻辑、无提示、无延迟
    upgrade: async () => {
        state.status.upgrade = true
        state.status.upgrade = false
    },
    // 仅保留弹窗展示逻辑
    show() {
        state.status.dialog = true
    },
    // 保留Markdown解析方法（仅保留方法结构，无实际解析需求）
    markdown: content => {
        const md = new MarkdownIt()
        return md.render(content || '')
    },
    // 保留颜色映射方法（仅保留方法结构）
    color: (value) => {
        switch (value) {
            case 'design':
                return { color: 'var(--bs-secondary)', text: '设计中' }
            case 'dev':
                return { color: 'var(--bs-primary)', text: '开发版' }
            case 'test':
                return { color: 'var(--bs-warning)', text: '测试版' }
            case 'pro':
                return { color: 'var(--bs-success)', text: '正式版' }
            case 'abandon':
                return { color: 'var(--bs-danger)', text: '停止维护' }
            default:
                return { color: 'var(--bs-light)', text: '未知' }
        }
    },
}

// 组件挂载时仅执行空方法，无数据请求
onMounted(async () => {
    await method.init()
})

// 保留监听结构，无实际逻辑
watch(() => state.struct, (item = {}) => {
    state.show.upgrade = false
})

// 暴露方法（仅保留结构）
defineExpose({
    init: method.init,
})
</script>