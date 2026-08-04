<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        评论管理配置，包括全局开关、频率限制、敏感词过滤等设置。
                    </template>
                    <span style="font-weight: 600">评论</span>
                </el-tooltip>
                <el-tag size="small" type="info">配置</el-tag>
            </div>
        </template>
        <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 12px">
                <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                    <i-svg name="comment" size="20px"></i-svg>
                </div>
                <div>
                    <div style="font-weight: 600; font-size: 14px; line-height: 1.4">评论配置</div>
                    <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">全局开关 · 频率限制 · 敏感词过滤</div>
                </div>
            </div>
            <el-button text type="primary" v-on:click="method.show()">
                配置
                <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
            </el-button>
        </div>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false" width="600px">
        <template #header>
            <strong class="flex-center">评论配置</strong>
        </template>
        <template #default>
            <el-form label-width="140px" label-position="left">
                <el-form-item label="全局评论开关">
                    <el-select v-model="state.struct.json.allow" placeholder="请选择">
                        <el-option v-for="item in state.select.switch" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-divider content-position="left">频率限制</el-divider>
                <el-form-item label="开启频率限制">
                    <el-select v-model="state.struct.json.rate_limit.enabled" placeholder="请选择">
                        <el-option v-for="item in state.select.switch" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="时间窗口最大评论数">
                    <el-input-number v-model="state.struct.json.rate_limit.max_count" :min="1" :max="100" style="width: 200px" />
                </el-form-item>
                <el-form-item label="时间窗口（秒）">
                    <el-input-number v-model="state.struct.json.rate_limit.time_window" :min="10" :max="3600" style="width: 200px" />
                </el-form-item>

                <el-divider content-position="left">内容限制</el-divider>
                <el-form-item label="评论最大长度">
                    <el-input-number v-model="state.struct.json.max_length" :min="10" :max="2000" style="width: 200px" />
                    <span style="margin-left: 8px; color: var(--el-text-color-secondary)">字符</span>
                </el-form-item>
                <el-form-item label="要求包含中文">
                    <el-select v-model="state.struct.json.require_chinese" placeholder="请选择">
                        <el-option v-for="item in state.select.switch" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-divider content-position="left">敏感词过滤</el-divider>
                <el-form-item label="开启敏感词过滤">
                    <el-select v-model="state.struct.json.sensitive_filter" placeholder="请选择">
                        <el-option v-for="item in state.select.switch" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="敏感词列表">
                    <el-input v-model="state.sensitiveWordsText" type="textarea" :rows="3" placeholder="每行一个敏感词" />
                </el-form-item>

                <el-divider content-position="left">邮件通知</el-divider>
                <el-form-item label="开启邮件通知">
                    <el-select v-model="state.struct.json.email_notify.enabled" placeholder="请选择">
                        <el-option v-for="item in state.select.switch" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="发送失败重试次数">
                    <el-input-number v-model="state.struct.json.email_notify.retry_count" :min="1" :max="10" style="width: 200px" />
                </el-form-item>
                <el-form-item label="重试间隔">
                    <el-input-number v-model="state.struct.json.email_notify.retry_interval" :min="1" :max="60" style="width: 200px" />
                    <span style="margin-left: 8px; color: var(--el-text-color-secondary)">分钟</span>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
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
const state = reactive({
    cache: {
        name: 'comment',
        json: {}
    },
    struct: {
        key: 'COMMENT',
        json: {
            allow: 1,
            rate_limit: {
                enabled: 1,
                max_count: 5,
                time_window: 60
            },
            max_length: 500,
            require_chinese: 1,
            sensitive_filter: 1,
            sensitive_words: ['色情', '广告', '开户'],
            email_notify: {
                enabled: 1,
                retry_count: 3,
                retry_interval: 5
            }
        }
    },
    sensitiveWordsText: '',
    status: {
        finish: false,
        loading: true,
        dialog: false,
        wait: false
    },
    select: {
        switch: [
            { value: 1, label: '开启' },
            { value: 0, label: '关闭' },
        ]
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
            key: 'COMMENT'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data
        
        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！') 
        state.sensitiveWordsText = state.struct.json.sensitive_words?.join('\n') || ''
        state.status.dialog = true
    },
    save: async () => {
        state.status.wait   = true
        
        state.struct.json.sensitive_words = state.sensitiveWordsText.split('\n').filter(word => word.trim())
        
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