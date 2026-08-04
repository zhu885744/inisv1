<template>
    <i-table :opts="state.opts" ref="i-table">

        <template #start>
            <el-table-column type="selection" width="55"></el-table-column>
        </template>

        <template v-if="props.type === 'all'" #end>
            <el-table-column :fixed="right" label="操作" width="100" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.edit(scope.row)" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, true)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>
        <template v-if="props.type === 'remove'">
            <el-table-column :fixed="right" label="操作" width="160" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.restore(scope.row.id)" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.edit(scope.row)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, false)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>

        <!-- 创建时间自定义插槽 -->
        <template #i-create_time="{ scope = {} }">
            <el-tooltip 
                :content="method.formatTime(scope.create_time)" 
                :disabled="!scope.create_time" 
                placement="top"
            >
                <span>{{ method.formatTime(scope.create_time) }}</span>
            </el-tooltip>
        </template>

        <!-- 更新时间自定义插槽 -->
        <template #i-update_time="{ scope = {} }">
            <el-tooltip 
                :content="method.formatTime(scope.update_time)" 
                :disabled="!scope.update_time" 
                placement="top"
            >
                <span>{{ method.formatTime(scope.update_time) }}</span>
            </el-tooltip>
        </template>

        <template #i-title="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-tooltip v-if="scope.audit === 1" content="已审核" placement="top">
                    <i-svg name="audit" size="20px" style="margin-right: 4px"></i-svg>
                </el-tooltip>
                <el-tooltip :disabled="utils.is.empty(scope.title)" placement="top">
                    <template #content>
                        <span v-html="method.renderEmoji(scope.title)"></span>
                    </template>
                    <span class="limit-1-line" v-html="method.renderEmoji(scope?.title)"></span>
                </el-tooltip>
            </span>
        </template>

        <template #i-remark="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.remark)" placement="top">
                <template #content>
                    <span v-html="method.renderEmoji(method.autoWrap(scope.remark))"></span>
                </template>
                <span v-html="method.renderEmoji(method.omit(scope?.remark))"></span>
            </el-tooltip>
        </template>

    </i-table>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'
import { computed, reactive, getCurrentInstance, onMounted, watch, toRaw } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'

const router = useRouter()
const emit   = defineEmits(['refresh','update:init'])
const props  = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    tableName: {
        type: String,
        default: 'pages'
    },
    params: {
        type: Object,
        default: () => ({
            order: 'id asc',
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

// 移动端自适应左右固定
const left = computed(() => utils.is.mobile() ? false : 'left')
const right = computed(() => utils.is.mobile() ? false : 'right')

const { proxy } = getCurrentInstance()
const state  = reactive({
    struct: {},
    opts: {
        url: `/api/${props.tableName}/all`,
        // 初始使用纯净默认参数，不继承历史where
        params: { order: 'id asc' },
        columns: [
            { prop: 'title',  label: '标题', slot: true, fixed: left },
            { prop: 'key',    label: '唯一键' },
            { prop: 'remark', label: '备注', slot: true },
            { prop: 'create_time', label: '创建时间', width: 180, slot: true, sortable: true },
            { prop: 'update_time', label: '更新时间', width: 180, slot: true, sortable: true },
        ],
    },
})

const method = {
    // 重载表格
    init: async () => {
        state.opts.params = { ...toRaw(props.params) }
        await proxy.$refs['i-table'].init()
    },
    // 重置表格：清空筛选，恢复默认参数，回到第一页
    resetTable: async () => {
        // 强制覆盖为纯净无筛选参数
        state.opts.params = { order: 'id asc' }
        await proxy.$refs['i-table'].reset()
    },
    edit: struct => {
        router.push({path: `/admin/${props.tableName}/write/${parseInt(struct.id)}`})
    },
    async delete(ids = [], isSoft = true) {
        if (utils.is.empty(ids)) return
        const idArr = Array.isArray(ids) ? ids : [ids]
        const uri = `/api/${props.tableName}/${isSoft ? 'remove' : 'delete'}`
        const { code, msg } = await axios.del(uri, { ids: idArr })
        if (code !== 200) return ElMessage.error(msg)
        emit('refresh', 'remove', 'check', 'audit')
        await method.init()
    },
    async restore(ids = []) {
        if (utils.is.empty(ids)) return
        const idArr = Array.isArray(ids) ? ids : [ids]
        const { code, msg } = await axios.put(`/api/${props.tableName}/restore`, { ids: idArr })
        if (code !== 200) return ElMessage.error(msg)
        emit('refresh', 'all', 'check', 'audit')
        await method.init()
    },
    autoWrap(text = '', length = 40, symbol = '<br>') {
        if (utils.is.empty(text)) return text
        return text.replace(new RegExp(`(.{${length}})`, 'g'), `$1${symbol}`)
    },
    copy: (text = null, msg = '复制成功！') => {
        if (utils.is.empty(text)) return
        utils.set.copy.text(text)
        if (!utils.is.empty(msg)) ElMessage.success(msg)
    },
    omit: (text = null, length = 10, omission = ' ... ', location = 'center') => {
        if (utils.is.empty(text)) return '空'
        return utils.string.omit(text, length, omission, location)
    },
    renderEmoji: (text = '') => {
        return utils.string.emoji(text)
    },
    formatTime: (timestamp) => {
        if (!timestamp || timestamp === 0) return '无数据'
        const time = timestamp.toString().length === 10 ? timestamp * 1000 : timestamp
        const date = new Date(time)
        const pad = (num) => num.toString().padStart(2, '0')
        return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
    }
}

// 页面每次刷新挂载，强制重载表格，清空历史筛选
onMounted(async () => {
    await method.init()
})

// 父组件传入筛选参数变化时自动刷新
watch(() => props.params, () => method.init(), { deep: true })
watch(() => props.init, (val) => {
    if (val) method.init()
})

// 对外暴露重载、重置方法给父页面
defineExpose({
    init: method.init,
    resetTable: method.resetTable
})
</script>