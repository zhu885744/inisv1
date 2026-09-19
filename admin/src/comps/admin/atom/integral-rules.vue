<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        积分规则配置，用于控制用户通过签到、登录、发布内容等行为可获得的积分及每日上限。积分在积分商城中兑换商品时消耗。
                    </template>
                    <span style="font-weight: 600">积分规则</span>
                </el-tooltip>
                <el-tag size="small" type="warning">配置</el-tag>
            </div>
        </template>
        <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 12px">
                <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-warning-light-9); color: var(--el-color-warning)">
                    <i-svg name="integral" size="20px"></i-svg>
                </div>
                <div>
                    <div style="font-weight: 600; font-size: 14px; line-height: 1.4">积分规则</div>
                    <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">签到 · 登录 · 文章 · 评论 · 动态 · 分享</div>
                </div>
            </div>
            <el-button text type="primary" v-on:click="method.show()">
                配置
                <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
            </el-button>
        </div>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="integral-rules-dialog" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">积分规则配置</strong>
        </template>
        <template #default>
            <el-alert
                type="warning"
                :closable="false"
                show-icon
                title="每日限制次数填 0 表示不限制；修改后立即生效（缓存会自动刷新）"
                style="margin-bottom: 12px"
            />
            <div class="rule-list">
                <div v-for="item in state.rulesList" :key="item.key" class="rule-card">
                    <div class="rule-head">
                        <div class="rule-title">
                            <span class="rule-label">{{ state.integralTypes[item.key]?.label || item.key }}</span>
                            <el-tag size="small" type="info" effect="plain">{{ item.key }}</el-tag>
                        </div>
                        <span class="rule-desc">{{ state.integralTypes[item.key]?.desc || '-' }}</span>
                    </div>
                    <div class="rule-body">
                        <div class="rule-field">
                            <label>操作名称</label>
                            <el-input v-model="item.name" size="small" placeholder="展示给用户的任务名称" />
                        </div>
                        <div class="rule-field">
                            <label>单次积分</label>
                            <el-input-number v-model="item.value" :min="0" :max="9999" size="small" style="width: 100%" />
                        </div>
                        <div class="rule-field">
                            <label>每日限制次数</label>
                            <el-input-number v-model="item.daily_limit" :min="0" :max="999" size="small" style="width: 100%" />
                            <span class="rule-tip">0 = 不限制次数</span>
                        </div>
                        <div class="rule-field">
                            <label>前台图标</label>
                            <el-input v-model="item.icon" size="small" placeholder="如 bi-calendar-check" />
                            <span class="rule-tip">前端主题使用的图标类名</span>
                        </div>
                    </div>
                </div>
            </div>
        </template>
        <template #footer>
            <el-button v-on:click="method.reset()">重置默认</el-button>
            <el-button v-on:click="state.status.dialog = false">取 消</el-button>
            <el-button type="primary" v-on:click="method.save()" :loading="state.status.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import cache from '{src}/utils/cache'
import axios from '{src}/utils/request'

const emit = defineEmits(['refresh'])
const { ctx, proxy } = getCurrentInstance()

const DEFAULT_RULES = {
    'check-in': { name: '每日签到', value: 5, daily_limit: 1, icon: 'bi-calendar-check' },
    'login': { name: '每日登录', value: 2, daily_limit: 1, icon: 'bi-box-arrow-in-right' },
    'article-create': { name: '发布文章', value: 10, daily_limit: 5, icon: 'bi-file-earmark-text' },
    'comment': { name: '发表评论', value: 2, daily_limit: 10, icon: 'bi-chat-dots' },
    'moments': { name: '发布动态', value: 20, daily_limit: 1, icon: 'bi-lightning' },
    'share': { name: '分享内容', value: 2, daily_limit: 3, icon: 'bi-share' }
}

const INTEGRAL_TYPES = {
    'check-in': { label: '每日签到', desc: '每日签到（与经验值签到同时触发）' },
    'login': { label: '每日登录', desc: '每日首次登录' },
    'article-create': { label: '发布文章', desc: '发布文章后自动发放' },
    'comment': { label: '发表评论', desc: '发表评论后自动发放' },
    'moments': { label: '发布动态', desc: '发布动态后自动发放' },
    'share': { label: '分享内容', desc: '分享文章/页面/动态后发放' }
}

const state = reactive({
    cache: {
        name: 'integral-rules',
        json: {}
    },
    struct: {
        key: 'SYSTEM_INTEGRAL_RULES',
        json: { ...DEFAULT_RULES }
    },
    rulesList: [],
    integralTypes: INTEGRAL_TYPES,
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

        state.status.finish = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/config/one', {
            key: 'SYSTEM_INTEGRAL_RULES'
        })

        state.status.loading = false

        if (code === 200 && data) {
            state.struct = { ...data, key: 'SYSTEM_INTEGRAL_RULES' }
            if (typeof state.struct.json === 'string') {
                try { state.struct.json = JSON.parse(state.struct.json) } catch { state.struct.json = {} }
            }
            state.status.finish = true
        } else {
            // 配置项不存在时使用默认规则，保存时后端会自动创建该配置项
            state.struct = { key: 'SYSTEM_INTEGRAL_RULES', json: { ...DEFAULT_RULES } }
            state.status.finish = true
        }
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！')

        const json = state.struct.json || {}
        state.rulesList = Object.keys(INTEGRAL_TYPES).map(key => {
            const rule = json[key] || DEFAULT_RULES[key] || {}
            return {
                key,
                name: rule.name || DEFAULT_RULES[key]?.name || key,
                value: Number(rule.value ?? DEFAULT_RULES[key]?.value ?? 0),
                daily_limit: Number(rule.daily_limit ?? DEFAULT_RULES[key]?.daily_limit ?? 0),
                icon: rule.icon || DEFAULT_RULES[key]?.icon || ''
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
            state.rulesList = Object.keys(INTEGRAL_TYPES).map(key => ({
                key,
                ...DEFAULT_RULES[key]
            }))
            ElMessage.success('已重置为默认值')
        }).catch(() => {})
    },
    save: async () => {
        state.status.wait = true

        try {
            const rules = {}
            state.rulesList.forEach(item => {
                rules[item.key] = {
                    name: item.name,
                    value: Number(item.value) || 0,
                    daily_limit: Number(item.daily_limit) || 0,
                    icon: item.icon || ''
                }
            })

            // 保留配置中已有的其它自定义任务，避免被覆盖丢失
            const keep = {}
            Object.keys(state.struct.json || {}).forEach(key => {
                if (!rules[key]) keep[key] = state.struct.json[key]
            })

            const merged = { ...keep, ...rules }
            state.struct.json = merged

            const { code, msg } = await axios.post('/api/config/save', {
                key: 'SYSTEM_INTEGRAL_RULES',
                json: JSON.stringify(merged)
            })

            if (code !== 200) return ElMessage.error('保存失败：' + msg)

            state.status.dialog = false
            ElMessage.success('保存成功，规则已即时生效')

            cache.set(state.cache.name, merged)
            emit('refresh')
        } finally {
            state.status.wait = false
        }
    },
    cache: (json = state.cache.json) => {
        if (cache.has(state.cache.name)) {
            const cached = cache.get(state.cache.name)
            state.cache.json = { ...cached }
            return
        }
        cache.set(state.cache.name, json)
    }
}

defineExpose({
    init: method.init,
})
</script>

<style scoped>
:deep(.integral-rules-dialog) {
    /* 卡片式布局：宽度固定更紧凑，字段多时自动换行 */
    width: 720px !important;
    min-width: 480px;
    max-width: 92vw;
}

:deep(.integral-rules-dialog .el-dialog__body) {
    max-height: 70vh;
    overflow-y: auto;
    padding: 16px;
}

/* ---------- 规则卡片列表（不使用表格，避免字段多时横向挤压） ---------- */
.rule-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.rule-card {
    padding: 12px 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-blank);
    transition: border-color 0.2s;
}

.rule-card:hover {
    border-color: var(--el-color-primary-light-5);
}

.rule-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 6px 12px;
    margin-bottom: 10px;
}

.rule-title {
    display: flex;
    align-items: center;
    gap: 8px;
}

.rule-label {
    font-weight: 600;
    font-size: 14px;
}

.rule-desc {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.rule-body {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 12px;
}

.rule-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
}

.rule-field label {
    font-size: 12px;
    color: var(--el-text-color-regular);
}

.rule-tip {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
}

:deep(.integral-rules-dialog .el-input-number),
:deep(.integral-rules-dialog .el-input) {
    width: 100%;
}
</style>
