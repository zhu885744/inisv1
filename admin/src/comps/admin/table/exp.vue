<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'all'">
            <el-button v-on:click="method.batchDelete(true)" type="danger" size="small">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量删除</span>
            </el-button>
        </div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'remove'">
            <el-button v-on:click="method.batchRestore()" type="primary" size="small">
                <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                <span style="margin-left: 4px">批量恢复</span>
            </el-button>
            <el-button v-on:click="method.batchDelete(false)" type="danger" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量永久删除</span>
            </el-button>
        </div>
    <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">

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

        <template #i-uid="{ scope = {} }">
            <span>{{ scope.uid || '-' }}</span>
        </template>

        <template #i-value="{ scope = {} }">
            <el-tag :type="scope.value > 0 ? 'success' : 'danger'" size="small">
                {{ scope.value > 0 ? '+' : '' }}{{ scope.value }}
            </el-tag>
        </template>

        <template #i-type="{ scope = {} }">
            <el-tag :type="method.getTypeTag(scope.type)" size="small">
                {{ method.getTypeName(scope.type) }}
            </el-tag>
        </template>

        <template #i-description="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.description)" placement="top">
                <template #content>
                    <span>{{ scope.description }}</span>
                </template>
                <span>{{ method.omit(scope.description) }}</span>
            </el-tooltip>
        </template>

    </i-table>
    </div>

    <el-dialog v-model="state.item.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 经验记录</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="用户ID">
                    <el-input v-model="state.struct.uid" type="number" placeholder="请输入用户ID"></el-input>
                </el-form-item>
                <el-form-item label="经验值">
                    <el-input-number v-model="state.struct.value" :min="-99999" :max="99999" style="width: 200px"></el-input-number>
                </el-form-item>
                <el-form-item label="类型">
                    <el-select v-model="state.struct.type" placeholder="请选择类型">
                        <el-option label="点赞" value="like"></el-option>
                        <el-option label="收藏" value="collect"></el-option>
                        <el-option label="访问" value="visit"></el-option>
                        <el-option label="分享" value="share"></el-option>
                        <el-option label="登录" value="login"></el-option>
                        <el-option label="评论" value="comment"></el-option>
                        <el-option label="签到" value="check-in"></el-option>
                        <el-option label="发布动态" value="moments"></el-option>
                        <el-option label="管理员发放" value="give"></el-option>
                        <el-option label="发布文章" value="article-create"></el-option>
                        <el-option label="文章获赞" value="article-like"></el-option>
                        <el-option label="文章被收藏" value="article-collect"></el-option>
                        <el-option label="发表评论" value="comment-create"></el-option>
                        <el-option label="评论获赞" value="comment-like"></el-option>
                        <el-option label="用户点赞" value="user-like"></el-option>
                        <el-option label="管理员操作" value="admin"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="描述">
                    <el-input v-model="state.struct.description" placeholder="请输入描述"></el-input>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.item.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.item.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import { reactive, computed, watch, onMounted, getCurrentInstance } from 'vue'
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
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

const left = computed(() => {
    let result = 'left'
    if (utils.is.mobile()) result = false
    return result
})

const right = computed(() => {
    let result = 'right'
    if (utils.is.mobile()) result = false
    return result
})

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        table: 'exp',
        dialog: false,
        wait: false,
        selection: [],
    },
    struct: {},
    opts: {
        url: '/api/exp/all',
        params: props.params,
        columns: [
            { prop: 'id', label: 'ID', width: 80, align: 'center' },
            { prop: 'uid', label: '用户ID', width: 100, slot: true, align: 'center' },
            { prop: 'value', label: '经验值', width: 120, slot: true, align: 'center' },
            { prop: 'type', label: '类型', width: 120, slot: true, align: 'center' },
            { prop: 'description', label: '描述', slot: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
})

const method = {
    init: async () => {
        await proxy.$refs['i-table']['init']()
    },
    save: async (params = state.struct || {}) => {
        if (utils.is.empty(params)) return ElMessage.warning('表单数据不能为空！')
        if (utils.is.empty(params?.uid)) return ElMessage.warning('用户ID为必填项！')
        if (params.value === undefined || params.value === null) return ElMessage.warning('经验值为必填项！')

        state.item.wait = true

        const { code, msg } = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait = false

        if (code !== 200) return ElMessage.error(msg)

        state.item.dialog = false
        await method.init()
        ElMessage.success('保存成功')
    },
    edit: struct => {
        state.struct = struct
        state.item.dialog = true
    },
    show: () => (state.item.dialog = true),
    async delete(ids = [], isSoft = true) {
        if (utils.is.empty(ids)) return

        const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`

        const { code, msg } = await axios.del(uri, { ids })

        if (code !== 200) return ElMessage.error(msg)

        emit('refresh', 'remove')
        await method.init()
        ElMessage.success('删除成功')
    },
    async restore(ids = []) {
        if (utils.is.empty(ids)) return

        const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })

        if (code !== 200) return ElMessage.error(msg)

        emit('refresh', 'all')
        await method.init()
        ElMessage.success('恢复成功')
    },
    selectionChange(selection) {
        state.item.selection = selection
    },
    async batchDelete(isSoft = true) {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要操作的记录')

        try {
            await ElMessageBox.confirm(
                `确定要${isSoft ? '软删除' : '永久删除'}选中的 ${ids.length} 条记录吗？`,
                '提示',
                { type: 'warning' }
            )
        } catch {
            return
        }

        state.item.wait = true
        try {
            const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`
            const { code, msg } = await axios.del(uri, { ids })
            state.item.wait = false
            if (code !== 200) throw new Error(msg)

            ElMessage.success(isSoft ? '批量软删除成功！' : '批量永久删除成功！')
            emit('refresh', 'remove')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '删除失败')
        }
    },
    async batchRestore() {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要恢复的记录')

        state.item.wait = true
        try {
            const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })
            state.item.wait = false
            if (code !== 200) throw new Error(msg)

            ElMessage.success('批量恢复成功！')
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '恢复失败')
        }
    },
    omit: (text = null, length = 15, omission = ' ...', location = 'center') => {
        if (utils.is.empty(text)) return '-'
        return utils.string.omit(text, length, omission, location)
    },
    getTypeName: (type = '') => {
        const types = {
            'like': '点赞',
            'collect': '收藏',
            'visit': '访问',
            'share': '分享',
            'login': '登录',
            'comment': '评论',
            'check-in': '签到',
            'moments': '发布动态',
            'give': '管理员发放',
            'article-create': '发布文章',
            'article-like': '文章获赞',
            'article-collect': '文章被收藏',
            'comment-create': '发表评论',
            'comment-like': '评论获赞',
            'user-like': '用户点赞',
            'admin': '管理员操作',
        }
        return types[type] || type || '未知'
    },
    getTypeTag: (type = '') => {
        const tags = {
            'like': 'primary',
            'collect': 'success',
            'visit': 'info',
            'share': 'warning',
            'login': 'success',
            'comment': 'primary',
            'check-in': 'danger',
            'moments': 'warning',
            'give': 'danger',
            'article-create': 'success',
            'article-like': 'primary',
            'article-collect': 'success',
            'comment-create': 'primary',
            'comment-like': 'primary',
            'user-like': 'primary',
            'admin': 'info',
        }
        return tags[type] || 'info'
    },
}

onMounted(async () => {
    if (props.init) await method.init()
})

watch(() => props.init, (val) => {
    if (val) method.init()
})

watch(() => state.item.dialog, (value) => {
    if (!value) state.struct = {}
})

defineExpose({
    init: method.init,
    show: method.show,
})
</script>