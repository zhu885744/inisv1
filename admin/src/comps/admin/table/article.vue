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
                            <i-svg name="edit" color="rgb(var(--icon-color))" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, true)" size="small" style="margin-left: 0">
                            <i-svg name="delete" color="rgb(var(--icon-color))" size="21px"></i-svg>
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
                            <i-svg name="restore" color="rgb(var(--icon-color))" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.edit(scope.row)" size="small" style="margin-left: 0">
                            <i-svg name="edit" color="rgb(var(--icon-color))" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, false)" size="small" style="margin-left: 0">
                            <i-svg name="delete" color="rgb(var(--icon-color))" size="21px"></i-svg>
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
                <el-tooltip v-if="scope.top === 1" content="置顶" placement="top">
                    <i-svg name="top" color="rgb(var(--icon-color))" size="16px" style="margin-right: 4px"></i-svg>
                </el-tooltip>
                <el-tooltip v-if="scope.status === 0" content="草稿" placement="top">
                    <el-icon><Edit /></el-icon>
                </el-tooltip>
                <el-tooltip :disabled="utils.is.empty(scope.title)" placement="top">
                    <template #content>
                        <span v-html="method.renderEmoji(scope.title)"></span>
                    </template>
                    <span class="limit-1-line" v-html="method.renderEmoji(scope?.title)"></span>
                </el-tooltip>
            </span>
        </template>

        <template #i-abstract="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.abstract)" placement="top">
                <template #content>
                    <span v-html="method.renderEmoji(method.autoWrap(scope.abstract))"></span>
                </template>
                <span v-html="method.renderEmoji(method.omit(scope?.abstract))"></span>
            </el-tooltip>
        </template>

        <template #i-remark="{ scope }">
            <el-tooltip :disabled="utils.is.empty(scope?.remark)" placement="top">
                <template #content>
                    <span v-html="method.renderEmoji(method.autoWrap(scope?.remark))"></span>
                </template>
                <span v-html="method.renderEmoji(method.omit(scope?.remark))"></span>
            </el-tooltip>
        </template>

    </i-table>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import { useRouter } from 'vue-router'
import ITable from '{src}/comps/custom/i-table.vue'
import { computed, reactive, getCurrentInstance, onMounted, watch, toRaw } from 'vue'

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    // 新增表名props，组件复用无需改内部代码
    tableName: {
        type: String,
        default: 'article'
    },
    params: {
        type: Object,
        default: () => ({
            order: 'top desc,id desc', // 统一和业务请求排序一致，消除冲突
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

// 移动端左右固定自适应
const left = computed(() => utils.is.mobile() ? false : 'left')
const right = computed(() => utils.is.mobile() ? false : 'right')

const { proxy } = getCurrentInstance()
const router = useRouter()
const state  = reactive({
    struct: {},
    opts: {
        url: `/api/${props.tableName}/all`, // 动态表名，不再写死article
        params: toRaw(props.params),
        columns: [
            { prop: 'title'  , label: '标题', slot: true, fixed: left },
            { prop: 'abstract', label: '摘要', slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'create_time', label: '创建时间', width: 180, slot: true, sortable: true },
            { prop: 'update_time', label: '更新时间', width: 180, slot: true, sortable: true },
        ],
    },
})

const method = {
    // 重载表格（清空缓存筛选，使用默认params）
    init: async () => {
        await proxy.$refs['i-table'].init()
    },
    // 编辑数据
    edit: struct => {
        router.push({path: `/admin/${props.tableName}/write/${parseInt(struct.id)}`})
    },
    // 删除：兼容单数字ID / 数组批量ID
    async delete(ids, isSoft = true) {
        if (utils.is.empty(ids)) return
        const idList = Array.isArray(ids) ? ids : [ids]
        const uri = `/api/${props.tableName}/${isSoft ? 'remove' : 'delete'}`
        const { code, msg } = await axios.del(uri, { ids: idList })
        if (code !== 200) return ElMessage.error(msg)
        emit('refresh', 'remove', 'check', 'audit')
        await method.init()
    },
    async restore(ids) {
        if (utils.is.empty(ids)) return
        const idList = Array.isArray(ids) ? ids : [ids]
        const { code, msg } = await axios.put(`/api/${props.tableName}/restore`, { ids: idList })
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
        if (!utils.is.empty(msg)) ElMessage.info(msg)
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

onMounted(async () => {
    if (props.init) await method.init()
})

// 父组件传递init刷新标识时重载
watch(() => props.init, (val) => {
    if (val) method.init()
})

// 对外暴露重载、重置筛选方法给父页面
defineExpose({
    init: method.init,
    resetTable: method.resetTable
})
</script>