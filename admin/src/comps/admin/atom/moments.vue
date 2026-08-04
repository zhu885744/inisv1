<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        动态管理配置，包括审核开关等设置。
                    </template>
                    <span style="font-weight: 600">动态</span>
                </el-tooltip>
                <el-tag size="small" type="info">配置</el-tag>
            </div>
        </template>
        <div style="display: flex; align-items: center; justify-content: space-between">
            <div style="display: flex; align-items: center; gap: 12px">
                <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                    <i-svg name="moments" size="20px"></i-svg>
                </div>
                <div>
                    <div style="font-weight: 600; font-size: 14px; line-height: 1.4">动态配置</div>
                    <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">审核开关 · 评论设置</div>
                </div>
            </div>
            <el-button text type="primary" v-on:click="method.show()">
                配置
                <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
            </el-button>
        </div>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">动态配置</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="审核">
                    <el-select v-model="state.struct.json.audit" placeholder="请选择">
                        <el-option v-for="item in state.select.audit" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.subtitle }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="允许评论">
                    <el-select v-model="state.struct.json.comment.allow" placeholder="请选择">
                        <el-option v-for="item in state.select.comment.allow" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="显示评论">
                    <el-select v-model="state.struct.json.comment.show" placeholder="请选择">
                        <el-option v-for="item in state.select.comment.show" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
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
        name: 'moments',
        json: {}
    },
    struct: {
        key: 'MOMENTS',
        json: {
            'comment': {
                'allow': 1, 'show': 1
            },
            'audit': 1,
        }
    },
    status: {
        finish: false,
        loading: true,
        dialog: false,
        wait: false
    },
    select: {
        comment: {
            allow: [
                { value: 1, label: '允许' },
                { value: 0, label: '禁止' },
            ],
            show: [
                { value: 1, label: '显示' },
                { value: 0, label: '隐藏' },
            ]
        },
        audit: [
            { value: 1, label: '开启', subtitle: '严格一点，防止乱搞' },
            { value: 0, label: '关闭', subtitle: '宽松一点，方便用户' },
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
            key: 'MOMENTS'
        })

        state.status.loading = false

        if (code !== 200) return
        state.struct = data
        
        state.status.finish  = true
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！') 
        state.status.dialog = true
    },
    save: async () => {
        state.status.wait   = true
        
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
