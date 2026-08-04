<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'all'">
            <el-button v-on:click="method.batchDelete" type="danger" size="small">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量删除</span>
            </el-button>
        </div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'remove'">
            <el-button v-on:click="method.batchRestore" type="success" size="small">
                <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                <span style="margin-left: 4px">批量恢复</span>
            </el-button>
            <el-button v-on:click="method.batchForceDelete" type="danger" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量彻底删除</span>
            </el-button>
        </div>
    <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">
        <template #start>
            <el-table-column type="selection" width="55"></el-table-column>
        </template>

        <template v-if="props.type === 'all'" #end>
            <el-table-column :fixed="right" label="操作" width="200" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.info(scope.row)" size="small" title="查看信息">
                            <span style="font-size: 11px">查看信息</span>
                        </el-button>
                        <el-button v-on:click="method.remove(scope.row.id)" size="small" style="margin-left: 0" title="移入回收站">
                            <i-svg name="delete" color="rgb(var(--icon-color))" size="16px"></i-svg>
                            <span style="font-size: 11px">删除</span>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>
        <template v-if="props.type === 'remove'">
            <el-table-column :fixed="right" label="操作" width="260" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.info(scope.row)" size="small" title="查看信息">
                            <span style="font-size: 11px">查看信息</span>
                        </el-button>
                        <el-button v-on:click="method.restore(scope.row.id)" size="small" style="margin-left: 0" title="恢复">
                            <i-svg name="restore" color="rgb(var(--icon-color))" size="16px"></i-svg>
                            <span style="font-size: 11px">恢复</span>
                        </el-button>
                        <el-button v-on:click="method.forceDelete(scope.row.id)" size="small" style="margin-left: 0" title="彻底删除（不可恢复）">
                            <i-svg name="delete" color="rgb(var(--icon-color))" size="16px"></i-svg>
                            <span style="font-size: 11px">彻底删除</span>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>

        <template #i-original_name="{ scope = {} }">
            <span v-on:dblclick="method.preview(scope)" style="display: flex; align-items: center">
                <el-image 
                    v-if="method.isImage(scope.file_ext)" 
                    :src="scope.full_url" 
                    @click.stop="method.preview(scope)"
                    style="width: 40px; height: 40px; margin-right: 8px; border-radius: 4px; cursor: pointer"
                    fit="cover"
                />
                <el-icon v-else size="32" style="margin-right: 8px">
                    <Document />
                </el-icon>
                <el-tooltip :content="scope.original_name" :disabled="utils.is.empty(scope.original_name)" placement="top">
                    <span class="limit-1-line">{{ method.omit(scope?.original_name) }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-file_size="{ scope = {} }">
            <span>{{ method.formatSize(scope.file_size) }}</span>
        </template>

        <template #i-mime_type="{ scope = {} }">
            <el-tag size="small">{{ scope.mime_type }}</el-tag>
        </template>

        <template #i-file_ext="{ scope = {} }">
            <span class="label label-info">{{ scope.file_ext }}</span>
        </template>

        <template #i-create_time="{ scope = {} }">
            <el-tooltip 
                :content="method.formatTime(scope.create_time)" 
                :disabled="!scope.create_time" 
                placement="top"
            >
                <span>{{ method.formatTime(scope.create_time) }}</span>
            </el-tooltip>
        </template>

        <template #i-full_url="{ scope = {} }">
            <el-tooltip :content="'双击复制：' + scope.full_url" :disabled="utils.is.empty(scope.full_url)" placement="top">
                <span v-on:dblclick="method.copy(scope.full_url)" class="limit-1-line">{{ scope.full_url }}</span>
            </el-tooltip>
        </template>

    </i-table>

    <el-dialog v-model="state.info.show" title="附件信息" width="500px">
        <el-descriptions :column="1" border>
            <el-descriptions-item label="文件名">
                <el-tooltip :content="state.info.item?.original_name" placement="top">
                    <span class="limit-1-line">{{ state.info.item?.original_name }}</span>
                </el-tooltip>
            </el-descriptions-item>
            <el-descriptions-item label="大小">
                {{ method.formatSize(state.info.item?.file_size) }}
            </el-descriptions-item>
            <el-descriptions-item label="扩展名">
                <span class="label label-info">{{ state.info.item?.file_ext }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="类型">
                <el-tag size="small">{{ state.info.item?.mime_type }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="上传用户">
                <span v-if="state.info.user">{{ state.info.user.nickname }} (ID: {{ state.info.item?.uploader_id }})</span>
                <span v-else-if="state.info.loading" style="color: #999">加载中...</span>
                <span v-else>{{ state.info.item?.uploader_id || '未知' }}</span>
            </el-descriptions-item>
            <el-descriptions-item label="上传时间">
                {{ method.formatTime(state.info.item?.create_time) }}
            </el-descriptions-item>
            <el-descriptions-item label="URL">
                <el-tooltip :content="'点击复制链接'" placement="top">
                    <span 
                        v-on:click="method.copy(state.info.item?.full_url)" 
                        class="limit-1-line cursor-pointer text-primary"
                    >{{ state.info.item?.full_url }}</span>
                </el-tooltip>
            </el-descriptions-item>
        </el-descriptions>
    </el-dialog>

    <el-image-viewer 
        v-if="state.viewer.show" 
        :url-list="state.viewer.list" 
        :initial-index="state.viewer.index"
        @close="state.viewer.show = false"
        append-to-body
    />
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'
import { computed, reactive, getCurrentInstance, onMounted, watch, toRaw } from 'vue'
import { Document } from '@element-plus/icons-vue'

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    tableName: {
        type: String,
        default: 'attachment'
    },
    params: {
        type: Object,
        default: () => ({
            order: 'create_time desc',
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

const left = computed(() => utils.is.mobile() ? false : 'left')
const right = computed(() => utils.is.mobile() ? false : 'right')

const { proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        selection: [],
    },
    info: {
        show: false,
        item: null,
        user: null,
        loading: false,
    },
    viewer: {
        show: false,
        list: [],
        index: 0
    },
    opts: {
        url: `/api/${props.tableName}/all`,
        params: toRaw(props.params),
        columns: [
            { prop: 'original_name', label: '文件名', slot: true, fixed: left },
            { prop: 'file_size', label: '大小', width: 100, slot: true },
            { prop: 'file_ext', label: '扩展名', width: 80, slot: true },
            { prop: 'mime_type', label: '类型', width: 150, slot: true },
            { prop: 'full_url', label: '链接', slot: true },
            { prop: 'create_time', label: '创建时间', width: 180, slot: true, sortable: true },
        ],
    },
})

const method = {
    init: async () => {
        await proxy.$refs['i-table'].init()
    },
    selectionChange: (val) => {
        state.item.selection = val
    },
    batchDelete: async () => {
        const ids = state.item.selection.map(item => item.id)
        await method.remove(ids)
        state.item.selection = []
    },
    batchRestore: async () => {
        const ids = state.item.selection.map(item => item.id)
        await method.restore(ids)
        state.item.selection = []
    },
    batchForceDelete: async () => {
        const ids = state.item.selection.map(item => item.id)
        await method.forceDelete(ids)
        state.item.selection = []
    },
    info: async (item) => {
        state.info.item = item
        state.info.user = null
        state.info.loading = true
        state.info.show = true
        if (item?.uploader_id) {
            try {
                const { code, data } = await axios.get(`/api/users/one?id=${item.uploader_id}`)
                if (code === 200) {
                    state.info.user = data
                }
            } catch (error) {
                console.error('获取用户信息失败:', error)
            }
        }
        state.info.loading = false
    },
    preview: (item) => {
        if (method.isImage(item?.file_ext)) {
            state.viewer.list = [item.full_url]
            state.viewer.index = 0
            state.viewer.show = true
        } else {
            method.download(item)
        }
    },
    download: (item) => {
        if (utils.is.empty(item?.full_url)) return
        const link = document.createElement('a')
        link.href = item.full_url
        link.download = item.original_name
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
    },
    async remove(ids) {
        if (utils.is.empty(ids)) return
        const idList = Array.isArray(ids) ? ids : [ids]
        ElMessageBox.confirm(
            `确定要将 ${idList.length} 个附件移入回收站吗？文件可在回收站中恢复。`,
            '确认删除',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning'
            }
        ).then(async () => {
            const { code, msg, data } = await axios.del(`/api/${props.tableName}/remove`, { ids: idList.join(',') })
            if (code === 400) return ElMessage.error(msg)
            if (code === 207) {
                ElMessage.warning(`${data.success_ids.length}个成功移入回收站，${data.failed_ids.length}个失败`)
            } else {
                ElMessage.success('已移入回收站')
            }
            emit('refresh', 'all', 'remove')
            await method.init()
        }).catch(() => {
            ElMessage.info('已取消')
        })
    },
    async forceDelete(ids) {
        if (utils.is.empty(ids)) return
        const idList = Array.isArray(ids) ? ids : [ids]
        ElMessageBox.confirm(
            `确定要彻底删除 ${idList.length} 个附件吗？此操作将永久删除数据库记录和存储桶文件，不可恢复！`,
            '警告',
            {
                confirmButtonText: '确定删除',
                cancelButtonText: '取消',
                type: 'error'
            }
        ).then(async () => {
            const { code, msg, data } = await axios.del(`/api/${props.tableName}/delete`, { ids: idList.join(',') })
            if (code === 400) return ElMessage.error(msg)
            if (code === 207) {
                ElMessage.warning(`${data.success_ids.length}个成功删除，${data.failed_ids.length}个失败`)
            } else {
                ElMessage.success('删除成功')
            }
            emit('refresh', 'all', 'remove')
            await method.init()
        }).catch(() => {
            ElMessage.info('已取消')
        })
    },
    async restore(ids) {
        if (utils.is.empty(ids)) return
        const idList = Array.isArray(ids) ? ids : [ids]
        ElMessageBox.confirm(
            `确定要恢复 ${idList.length} 个附件吗？`,
            '确认恢复',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'info'
            }
        ).then(async () => {
            const { code, msg, data } = await axios.put(`/api/${props.tableName}/restore`, { ids: idList.join(',') })
            if (code === 400) return ElMessage.error(msg)
            if (code === 207) {
                ElMessage.warning(`${data.success_ids.length}个成功恢复，${data.failed_ids.length}个失败`)
            } else {
                ElMessage.success('恢复成功')
            }
            emit('refresh', 'all', 'remove')
            await method.init()
        }).catch(() => {
            ElMessage.info('已取消')
        })
    },
    isImage: (ext) => {
        const images = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg']
        return images.includes((ext || '').toLowerCase())
    },
    formatSize: (bytes) => {
        if (!bytes || bytes === 0) return '0 B'
        const k = 1024
        const sizes = ['B', 'KB', 'MB', 'GB']
        const i = Math.floor(Math.log(bytes) / Math.log(k))
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
    },
    formatTime: (timestamp) => {
        if (!timestamp || timestamp === 0) return '无数据'
        const time = timestamp.toString().length === 10 ? timestamp * 1000 : timestamp
        const date = new Date(time)
        const pad = (num) => num.toString().padStart(2, '0')
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
    },
    copy: (text = null, msg = '复制成功！') => {
        if (utils.is.empty(text)) return
        utils.set.copy.text(text)
        if (!utils.is.empty(msg)) ElMessage.info(msg)
    },
    omit: (text = null, length = 10, omission = ' ... ', location = 'center') => {
        if (utils.is.empty(text)) return '空'
        return utils.string.omit(text, length, omission, location)
    }
}

onMounted(async () => {
    if (props.init) await method.init()
})

watch(() => props.init, (val) => {
    if (val) method.init()
})

defineExpose({
    init: method.init
})
</script>