<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        经验值规则配置，包括点赞、收藏、访问、分享、登录、评论、签到、发布动态、发布文章、文章获赞、文章被收藏、评论获赞、用户点赞等操作的经验值规则。
                    </template>
                    <span style="font-weight: 600">经验值规则</span>
                </el-tooltip>
                <el-tag size="small" type="info">配置</el-tag>
            </div>
        </template>
        <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 12px">
                <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                    <i-svg name="exp" size="20px"></i-svg>
                </div>
                <div>
                    <div style="font-weight: 600; font-size: 14px; line-height: 1.4">经验值规则</div>
                    <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">点赞 · 收藏 · 签到 · 文章 · 评论</div>
                </div>
            </div>
            <el-button text type="primary" v-on:click="method.show()">
                配置
                <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
            </el-button>
        </div>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="exp-rules-dialog" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">经验值规则配置</strong>
        </template>
        <template #default>
            <div class="table-container">
                <el-table :data="state.expRulesList" border style="width: 100%;">
                    <el-table-column prop="key" label="类型" min-width="100">
                        <template #default="scope">
                            <span style="font-weight: 600">{{ state.expTypes[scope.row.key]?.label || scope.row.key }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column prop="name" label="操作名称" min-width="100">
                        <template #default="scope">
                            <el-input v-model="scope.row.name" size="small" />
                        </template>
                    </el-table-column>
                    <el-table-column prop="value" label="经验值" min-width="100">
                        <template #default="scope">
                            <el-input-number v-model="scope.row.value" :min="0" :max="9999" size="small" />
                        </template>
                    </el-table-column>
                    <el-table-column prop="daily_limit" label="每日限制次数" min-width="140">
                        <template #default="scope">
                            <div style="display: flex; align-items: center; gap: 4px;">
                                <el-input-number v-model="scope.row.daily_limit" :min="0" :max="999" size="small" :disabled="['login', 'check-in'].includes(scope.row.key)" />
                                <span v-if="['login', 'check-in'].includes(scope.row.key)" style="font-size: 11px; color: var(--el-text-color-secondary)">（固定1次）</span>
                                <span v-else style="font-size: 11px; color: var(--el-text-color-secondary)">（0=不限制）</span>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column prop="desc" label="说明" min-width="150">
                        <template #default="scope">
                            <span style="font-size: 12px; color: var(--el-text-color-secondary)">
                                {{ state.expTypes[scope.row.key]?.desc || '-' }}
                            </span>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </template>
        <template #footer>
            <el-button v-on:click="method.reset()">重置默认</el-button>
            <el-button v-on:click="state.status.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.status.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import cache from '{src}/utils/cache'
import axios from '{src}/utils/request'

const emit = defineEmits(['refresh'])
const { ctx, proxy } = getCurrentInstance()

const DEFAULT_EXP_RULES = {
    like: { name: '点赞', value: 1, daily_limit: 10 },
    collect: { name: '收藏', value: 1, daily_limit: 10 },
    visit: { name: '访问', value: 1, daily_limit: 10 },
    share: { name: '分享', value: 1, daily_limit: 10 },
    login: { name: '登录', value: 5, daily_limit: 1 },
    comment: { name: '评论', value: 1, daily_limit: 10 },
    'check-in': { name: '签到', value: 10, daily_limit: 1 },
    moments: { name: '发布动态', value: 50, daily_limit: 1 },
    'article-create': { name: '发布文章', value: 5, daily_limit: 10 },
    'article-like': { name: '内容获赞', value: 5, daily_limit: 10 },
    'article-collect': { name: '内容被收藏', value: 5, daily_limit: 10 },
    'comment-create': { name: '发表评论', value: 5, daily_limit: 10 },
    'comment-like': { name: '评论获赞', value: 5, daily_limit: 10 }
}

const EXP_TYPES = {
    like: { label: '点赞', desc: '点赞文章/页面/评论/动态' },
    collect: { label: '收藏', desc: '收藏文章/页面' },
    visit: { label: '访问', desc: '访问文章/页面' },
    share: { label: '分享', desc: '分享文章/页面/动态' },
    login: { label: '登录', desc: '每日首次登录' },
    comment: { label: '评论', desc: '发表评论' },
    'check-in': { label: '签到', desc: '每日签到' },
    moments: { label: '发布动态', desc: '发布动态（自动触发）' },
    'article-create': { label: '发布文章', desc: '发布文章获得经验值（自动触发）' },
    'article-like': { label: '内容获赞', desc: '内容被点赞获得经验值（自动触发）' },
    'article-collect': { label: '内容被收藏', desc: '内容被收藏获得经验值（自动触发）' },
    'comment-create': { label: '发表评论', desc: '发表评论获得经验值（自动触发）' },
    'comment-like': { label: '评论获赞', desc: '评论被点赞获得经验值（自动触发）' }
}

const state = reactive({
    cache: {
        name: 'exp-rules',
        json: {}
    },
    struct: {
        key: 'SYSTEM_EXP_RULES',
        json: { ...DEFAULT_EXP_RULES }
    },
    expRulesList: [],
    expTypes: EXP_TYPES,
    status: {
        finish: false,
        loading: true,
        dialog: false,
        wait: false
    }
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {
        method.cache()

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/config/one', {
            key: 'SYSTEM_EXP_RULES'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data
        
        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！') 
        
        state.expRulesList = Object.keys(state.expTypes).map(key => ({
            key,
            ...state.struct.json[key] || DEFAULT_EXP_RULES[key]
        }))
        
        // 强制登录和签到的每日限制为1
        state.expRulesList.forEach(item => {
            if (['login', 'check-in'].includes(item.key)) {
                item.daily_limit = 1
            }
        })
        
        state.status.dialog = true
    },
    reset() {
        ElMessageBox.confirm('确定要重置为默认值吗？', '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        }).then(() => {
            state.expRulesList = Object.keys(state.expTypes).map(key => ({
                key,
                ...DEFAULT_EXP_RULES[key]
            }))
            ElMessage.success('已重置为默认值')
        }).catch(() => {})
    },
    save: async () => {
        state.status.wait   = true
        
        const rules = {}
        state.expRulesList.forEach(item => {
            rules[item.key] = {
                name: item.name,
                value: item.value,
                daily_limit: ['login', 'check-in'].includes(item.key) ? 1 : item.daily_limit
            }
        })
        state.struct.json = rules
        
        const { code, msg } = await axios.post('/api/config/save', {
            ...state.struct,
            json: JSON.stringify(state.struct.json)
        })

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
        ElMessage.success('保存成功') 
        cache.set(state.cache.name, state.cache.json)
    },
    cache: (json = state.cache.json) => {
        if (cache.has(state.cache.name)) {
            const cached = cache.get(state.cache.name)
            state.cache.json = { ...cached }
            return
        }
        cache.set(state.cache.name, json)
    },
}

defineExpose({
    init: method.init,
})
</script>

<style scoped>
:deep(.exp-rules-dialog) {
    width: auto !important;
    min-width: 600px;
    max-width: 90vw;
}

:deep(.exp-rules-dialog .el-dialog__body) {
    max-height: 70vh;
    overflow-y: auto;
    padding: 16px;
}

.table-container {
    width: 100%;
    overflow-x: auto;
}

:deep(.exp-rules-dialog .el-input-number) {
    width: 100%;
}

:deep(.exp-rules-dialog .el-input) {
    width: 100%;
}
</style>