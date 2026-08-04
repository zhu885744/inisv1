<template>
    <div class="container-box">
        <el-row :gutter="20" style="display: flex">
            <el-col :span="12" style="display: flex;">
                <el-dropdown style="margin-right: 8px" trigger="click">
                    <el-button>
                        {{ state.item.sort }}
                        <i-svg name="down"></i-svg>
                    </el-button>
                    <template #dropdown>
                        <el-dropdown-item v-on:click="method.order('create_time desc', '最新')">最新</el-dropdown-item>
                        <el-dropdown-item v-on:click="method.order('create_time asc', '最早')">最早</el-dropdown-item>
                    </template>
                </el-dropdown>
                <div style="margin-right: 4px">
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="文件名 | 链接" />
                </div>
                <el-button v-on:click="method.refresh()">刷新</el-button>
                <el-button type="primary" v-on:click="state.upload.show = true">
                    <i-svg name="upload" color="rgb(var(--icon-color))" size="16px"></i-svg>
                    <span style="margin-left: 4px">上传附件</span>
                </el-button>
            </el-col>
            <el-col :span="12" style="display: flex; justify-content: flex-end; z-index: -1">
                <el-button disabled>
                    {{ state.item.title }}
                </el-button>
            </el-col>
        </el-row>
        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-tabs v-model="state.item.tabs" v-on:tab-change="method.change" id="tabs-area">

                    <el-tab-pane name="all">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">全部</span>
                        </template>
                        <table-attachment :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-attachment>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <div style="margin-bottom: 12px; display: flex; justify-content: flex-end">
                            <el-button 
                                v-if="state.user?.result?.auth?.all" 
                                type="danger" 
                                size="small" 
                                @click="method.clearRecycleBin"
                                :loading="state.item.loading"
                                :disabled="state.item.loading"
                            >
                                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                                <span style="margin-left: 4px">清空回收站</span>
                            </el-button>
                        </div>
                        <table-attachment :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-attachment>
                    </el-tab-pane>

                    <el-tab-pane name="config" v-if="state.user?.result?.auth?.all">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">配置</span>
                        </template>
                        <el-card v-loading="state.config.loading">
                            <el-form :model="state.config.form" label-width="160px" style="max-width: 600px">
                                <el-form-item label="允许上传的文件类型">
                                    <el-input 
                                        v-model="state.config.form.allow_extensions" 
                                        placeholder="多个用逗号分隔，小写"
                                    />
                                    <div style="font-size: 12px; color: #999; margin-top: 4px">
                                        默认: jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md
                                    </div>
                                </el-form-item>
                                <el-form-item label="单个文件最大大小(KB)">
                                    <el-input-number 
                                        v-model="state.config.form.max_file_size" 
                                        :min="1" 
                                        :max="102400"
                                        placeholder="默认51200(50MB)"
                                    />
                                    <div style="font-size: 12px; color: #999; margin-top: 4px">
                                        51200 KB = 50MB
                                    </div>
                                </el-form-item>
                                <el-form-item label="并发上传限制">
                                    <el-input-number 
                                        v-model="state.config.form.concurrent_limit" 
                                        :min="1" 
                                        :max="20"
                                        placeholder="默认5"
                                    />
                                </el-form-item>

                                <el-form-item>
                                    <el-button 
                                        type="primary" 
                                        @click="method.saveConfig"
                                        :loading="state.config.saving"
                                    >
                                        <i-svg name="save" color="rgb(var(--icon-color))" size="16px"></i-svg>
                                        <span style="margin-left: 4px">保存配置</span>
                                    </el-button>
                                    <el-button @click="method.loadConfig" style="margin-left: 8px">
                                        <i-svg name="refresh" color="rgb(var(--icon-color))" size="16px"></i-svg>
                                        <span style="margin-left: 4px">重置</span>
                                    </el-button>
                                </el-form-item>
                            </el-form>
                        </el-card>
                    </el-tab-pane>

                </el-tabs>
            </el-col>
        </el-row>

        <el-dialog v-model="state.upload.show" title="上传附件" width="560px" destroy-on-close>
            <el-upload
                class="upload-demo"
                action="/api/attachment/batch"
                :show-file-list="true"
                :auto-upload="true"
                :before-upload="method.beforeUpload"
                :on-success="method.uploadSuccess"
                :on-error="method.uploadError"
                :on-change="method.uploadChange"
                :data="{ target_type: 'attachment' }"
                drag
            >
                <el-icon size="48" color="#1890ff" style="margin-bottom: 12px">
                    <Upload />
                </el-icon>
                <div style="font-size: 16px; font-weight: 600; margin-bottom: 4px">拖拽文件到此处上传</div>
                <div style="font-size: 13px; color: #999; margin-bottom: 16px">或点击选择文件</div>
                <el-button type="primary" size="default">
                    <i-svg name="upload" color="rgb(var(--icon-color))" size="16px"></i-svg>
                    <span style="margin-left: 4px">选择文件</span>
                </el-button>
            </el-upload>
        </el-dialog>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import TableAttachment from '{src}/comps/admin/table/attachment.vue'
import cache from "{src}/utils/cache.js";
import axios from '{src}/utils/request.js';
import { Upload } from '@element-plus/icons-vue';
import { reactive, onMounted, nextTick, watch } from 'vue';

const { proxy } = getCurrentInstance()
const state  = reactive({
    user: cache.get('user-info'),
    item: {
        timer : null,
        title : '附件管理',
        search: null,
        sort  : '排序',
        tabs  : 'all',
        loading: false,
    },
    params: {
        all: {
            order: 'create_time desc',
        },
        remove: {
            order: 'create_time desc',
            onlyTrashed: true
        },
    },
    tabs: {
        all: false,
        remove: false,
        config: false,
    },
    upload: {
        show: false,
        total: 0,
        success: 0,
        timer: null,
    },
    config: {
        loading: false,
        saving: false,
        form: {
            allow_extensions: '',
            max_file_size: 51200,
            concurrent_limit: 5,
        },
        original: {}
    }
})

const method = {
    order(order = 'create_time asc', sort = '排序') {
        state.item.sort = sort
        for (let item in state.params) state.params[item].order = order
        method.refresh('all', 'remove')
    },
    refresh(...args) {
        let allow = ['all', 'remove']
        if (args.length === 0) args = allow
        else args = args.filter(item => allow.includes(item))
        for (let item of args) {
            if (proxy.$refs[item]) proxy.$refs[item]['init']()
        }
    },
    change: async (name) => {
        state.tabs[name] = true
        if (name === 'config') {
            await method.loadConfig()
        }
    },
    beforeUpload: async (file) => {
        const { code, data } = await axios.post('/api/attachment/checkType', { file_names: [file.name] })
        if (code !== 200) {
            ElMessage.error('文件类型检查失败')
            return false
        }
        const result = data.results?.[0]
        if (!result?.is_allowed) {
            ElMessage.error(result?.message || '不允许上传该类型的文件')
            return false
        }
        return true
    },
    uploadChange: (file, fileList) => {
        if (file.status === 'ready') {
            state.upload.total++
        }
    },
    uploadSuccess: (response, file, fileList) => {
        const { code, msg, data } = response
        if (code !== 200) {
            return ElMessage.error(msg)
        }
        state.upload.success += data.results?.length || 1
        if (state.upload.timer) clearTimeout(state.upload.timer)
        state.upload.timer = setTimeout(() => {
            ElMessage.success(`上传完成！成功 ${state.upload.success}/${state.upload.total}`)
            method.refresh('all')
            state.upload.total = 0
            state.upload.success = 0
        }, 500)
    },
    uploadError: (err, file, fileList) => {
        ElMessage.error('上传失败')
        if (state.upload.timer) clearTimeout(state.upload.timer)
        state.upload.timer = setTimeout(() => {
            method.refresh('all')
            state.upload.total = 0
            state.upload.success = 0
        }, 500)
    },
    clearRecycleBin: async () => {
        if (state.item.loading) return
        ElMessageBox.confirm(
            '确定要清空回收站吗？此操作将永久删除所有已删除的附件文件，不可恢复！',
            '警告',
            {
                confirmButtonText: '确定清空',
                cancelButtonText: '取消',
                type: 'warning'
            }
        ).then(async () => {
            state.item.loading = true
            const { code, msg } = await axios.del('/api/attachment/clear')
            state.item.loading = false
            if (code !== 200) return ElMessage.error(msg)
            ElMessage.success('清空成功！')
            method.refresh('all', 'remove')
        }).catch(() => {
            state.item.loading = false
            ElMessage.info('已取消清空')
        })
    },
    loadConfig: async () => {
        state.config.loading = true
        try {
            const { code, msg, data } = await axios.get('/api/toml/storage?name=attachment')
            state.config.loading = false
            if (code !== 200) {
                ElMessage.warning('获取配置失败，使用默认值')
                method.resetToDefault()
                return
            }
            if (!data || Object.keys(data).length === 0) {
                ElMessage.info('配置为空，使用默认值')
                method.resetToDefault()
                return
            }
            Object.assign(state.config.form, data)
            state.config.original = JSON.parse(JSON.stringify(data))
        } catch (error) {
            state.config.loading = false
            ElMessage.warning('网络异常，使用默认值')
            method.resetToDefault()
        }
    },
    resetToDefault: () => {
        const defaults = {
            allow_extensions: 'jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md',
            max_file_size: 51200,
            concurrent_limit: 5,
        }
        Object.assign(state.config.form, defaults)
        state.config.original = JSON.parse(JSON.stringify(defaults))
    },
    saveConfig: async () => {
        state.config.saving = true
        try {
            const payload = {}
            for (const key in state.config.form) {
                if (state.config.form[key] !== state.config.original[key]) {
                    payload[key] = state.config.form[key]
                }
            }
            if (Object.keys(payload).length === 0) {
                state.config.saving = false
                return ElMessage.info('没有需要修改的配置')
            }
            const { code, msg } = await axios.put('/api/toml/storage-attachment', payload)
            state.config.saving = false
            if (code === 403) return ElMessage.warning('您没有权限修改配置，请联系管理员')
            if (code === 400) return ElMessage.warning(msg)
            if (code !== 200) return ElMessage.error('保存失败：' + msg)
            ElMessage.success('配置保存成功！')
            await method.loadConfig()
        } catch (error) {
            state.config.saving = false
            ElMessage.error('网络请求失败，请稍后重试')
        }
    }
}

onMounted(async () => {
    const allow = ['all', 'remove']

    let root = state.user?.result?.auth?.all ?? false
    let userId = state.user?.id
    if (!root && userId) {
        for (let item of allow) {
            if (!state.params[item].where) state.params[item].where = []
            state.params[item].where.push(['uid', '=', userId])
        }
    }

    await nextTick()
    state.tabs.all = true
})

watch(() => state.item.search, (val) => {
    const allow = ['all', 'remove']

    for (let item of allow) {
        if (!utils.is.empty(val)) state.params[item].like = [
            ['original_name', `%${val}%`],
            ['full_url'     , `%${val}%`],
        ]
        else delete state.params[item].like
    }

    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => method.refresh(...allow), globalThis.inis?.lazy_time ?? 500)
})
</script>

<style scoped>
.upload-tip {
    text-align: center;
}

:deep(.el-upload-dragger) {
    width: 100%;
    padding: 40px 20px !important;
    border-radius: 8px;
    background-color: #fafafa;
    transition: all 0.3s ease;
}

:deep(.el-upload-dragger:hover) {
    border-color: #1890ff !important;
    background-color: #f0f5ff;
}

:deep(.el-upload-list) {
    margin-top: 16px;
}
</style>