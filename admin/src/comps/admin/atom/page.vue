<template>
    <el-card v-loading="state.status.loading" style="margin-bottom: 12px">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● Markdown编辑器：Vditor支持所见即所得、即时渲染（类似 Typora）和分屏预览模式。
                    </template>
                    <span style="font-weight: 600">页面</span>
                </el-tooltip>
                <el-tag size="small" type="success">更多</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="book" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">页面管理</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">管理单页面的编辑器与评论审核配置</div>
                    </div>
                </div>
                <el-button text type="primary" v-on:click="method.show()">
                    配置
                    <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
                </el-button>
            </div>
        </template>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">页面配置</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="编辑器">
                    <el-select v-model="state.cache.json.editor" disabled>
                        <el-option value="vditor" label="Markdown">
                            <span>Markdown</span>
                            <small style="float: right">vditor</small>
                        </el-option>
                    </el-select>
                </el-form-item>
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
        name: 'page',
        json: {
            editor: 'vditor'
        }
    },
    struct: {
        key: 'PAGE',
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
        wait: false  // 补充wait状态定义，避免保存时未定义错误
    },
    select: {
        editor: [
            { value: 'vditor', label: 'Markdown' }
        ],
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
            key: 'PAGE'
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

        if (code !== 200) return ElMessage.error(`保存失败：${msg}`)
        
        ElMessage.success('保存成功')
        state.status.dialog = false

        cache.set(state.cache.name, state.cache.json)
    },
    // 获取缓存
    cache: (json = state.cache.json) => {
        // 缓存存在 - 直接返回
        if (cache.has(state.cache.name)) {
            state.cache.json = cache.get(state.cache.name)
            return
        }

        // 缓存不存在 - 保存缓存
        cache.set(state.cache.name, json)
    },
}

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>