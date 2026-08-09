<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0">
            <el-tag type="info" style="margin-right: 8px">{{ state.item.selection.length }} 条选中</el-tag>
        </div>
        <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">
            <template #start>
                <el-table-column type="selection" width="55"></el-table-column>
            </template>

            <template #end>
                <el-table-column :fixed="right" label="操作" width="160" class-name="text-end">
                    <template #default="scope">
                        <span style="display: flex; justify-content: flex-end; gap: 4px">
                            <!-- 生效中可解封 -->
                            <el-button v-if="scope.row.status === 0" v-on:click="method.unbanById(scope.row)" size="small" style="color: #67c23a">
                                解封
                            </el-button>
                            <!-- 申诉中可审核 -->
                            <template v-if="scope.row.status === 3">
                                <el-button v-on:click="method.appealApprove(scope.row)" size="small" type="success">
                                    通过
                                </el-button>
                                <el-button v-on:click="method.appealReject(scope.row)" size="small" type="danger">
                                    驳回
                                </el-button>
                            </template>
                            <el-tag v-else-if="scope.row.status === 1" type="success" size="small">已解封</el-tag>
                            <el-tag v-else-if="scope.row.status === 2" type="info" size="small">已撤销</el-tag>
                            <el-tag v-else-if="scope.row.status === 4" type="success" size="small">申诉通过</el-tag>
                            <el-tag v-else-if="scope.row.status === 5" type="danger" size="small">申诉驳回</el-tag>
                            <el-tag v-else size="small">{{ scope.row.status }}</el-tag>
                        </span>
                    </template>
                </el-table-column>
            </template>

            <template #i-uid="{ scope = {} }">
                <el-tooltip :content="'双击复制ID：' + scope?.uid" placement="top">
                    <span v-on:dblclick="method.copy(String(scope?.uid))" style="cursor: pointer">
                        {{ scope?.uid }}
                    </span>
                </el-tooltip>
            </template>

            <template #i-ban_type="{ scope = {} }">
                <span v-if="scope?.ban_type === 31" style="color: #f56c6c">全部封禁</span>
                <span v-else>
                    <el-tag v-if="scope?.ban_type & 1" size="small" type="danger" style="margin: 1px">登录</el-tag>
                    <el-tag v-if="scope?.ban_type & 2" size="small" type="warning" style="margin: 1px">发文</el-tag>
                    <el-tag v-if="scope?.ban_type & 4" size="small" style="margin: 1px">评论</el-tag>
                    <el-tag v-if="scope?.ban_type & 8" size="small" type="info" style="margin: 1px">上传</el-tag>
                    <el-tag v-if="scope?.ban_type & 16" size="small" style="margin: 1px">互动</el-tag>
                </span>
            </template>

            <template #i-duration="{ scope = {} }">
                <span v-if="scope?.duration === 0" style="color: #f56c6c; font-weight: bold">永久</span>
                <span v-else>{{ scope?.duration }} 天</span>
            </template>

            <template #i-status="{ scope = {} }">
                <el-tag v-if="scope?.status === 0" type="danger" size="small">生效中</el-tag>
                <el-tag v-else-if="scope?.status === 1" type="success" size="small">已解封</el-tag>
                <el-tag v-else-if="scope?.status === 2" type="info" size="small">已撤销</el-tag>
                <el-tag v-else-if="scope?.status === 3" type="warning" size="small">申诉中</el-tag>
                <el-tag v-else-if="scope?.status === 4" type="success" size="small">申诉通过</el-tag>
                <el-tag v-else-if="scope?.status === 5" type="danger" size="small">申诉驳回</el-tag>
                <el-tag v-else size="small">{{ scope?.status }}</el-tag>
            </template>

            <template #i-result="{ scope = {} }">
                <div v-if="scope?.result?.user" style="display: flex; align-items: center">
                    <el-avatar shape="square" :src="scope?.result?.user?.avatar" size="small" style="margin-right: 4px" />
                    <span>{{ scope?.result?.user?.nickname || scope?.uid }}</span>
                </div>
                <span v-else>用户ID: {{ scope?.uid }}</span>
            </template>

            <template #i-create_time="{ scope = {} }">
                <span>{{ utils.time.nature(scope?.create_time) }}</span>
            </template>

            <template #i-ban_time="{ scope = {} }">
                <span v-if="scope?.ban_time > 0">{{ utils.time.date(scope?.ban_time) }}</span>
                <span v-else>-</span>
            </template>

            <template #i-expires_at="{ scope = {} }">
                <span v-if="scope?.duration === 0">-</span>
                <span v-else-if="scope?.expires_at > 0">
                    <span v-if="scope?.expires_at * 1000 < Date.now()" style="color: #67c23a">已到期</span>
                    <span v-else>{{ utils.time.date(scope?.expires_at) }}</span>
                </span>
                <span v-else>-</span>
            </template>
        </i-table>
    </div>
</template>

<script setup>
import { reactive, computed, onMounted, watch, getCurrentInstance } from 'vue'
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'

const emit  = defineEmits(['refresh', 'update:init'])
const props = defineProps({
    init: {
        type: Boolean,
        default: false,
    }
})

const { proxy } = getCurrentInstance()
const right = computed(() => utils.is.mobile() ? false : 'right')

const state  = reactive({
    item: {
        selection: [],
    },
    opts: {
        url: '/api/users/blackroom',
        params: {
            order: 'create_time desc',
        },
        columns: [
            { prop: 'id', label: 'ID', width: 70, align: 'center' },
            { prop: 'result', label: '用户', width: 160, slot: true },
            { prop: 'uid', label: 'UID', width: 80, slot: true },
            { prop: 'ban_type', label: '封禁类型', width: 140, slot: true },
            { prop: 'reason', label: '原因', minWidth: 140 },
            { prop: 'duration', label: '时长', width: 80, slot: true },
            { prop: 'status', label: '状态', width: 100, slot: true },
            { prop: 'violation_num', label: '违规次数', width: 90, align: 'center' },
            { prop: 'ban_time', label: '封禁时间', width: 140, slot: true },
            { prop: 'expires_at', label: '到期时间', width: 140, slot: true },
            { prop: 'create_time', label: '创建时间', width: 140, slot: true },
        ],
    },
})

const method = {
    init: async () => {
        await proxy.$refs['i-table']['init']()
    },
    selectionChange(selection) {
        state.item.selection = selection
    },
    // 根据封禁记录ID解封
    async unbanById(row) {
        if (!row) return
        try {
            await ElMessageBox.confirm(
                `确定要解封记录 #${row.id}（用户UID: ${row.uid}）吗？`,
                '解封确认',
                { type: 'warning' }
            )
        } catch {
            return
        }

        try {
            const { code, msg } = await axios.put('/api/users/unban', { record_id: row.id })
            if (code !== 200) throw new Error(msg)
            ElMessage.success('解封成功！')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '解封失败')
        }
    },
    // 申诉通过
    async appealApprove(row) {
        if (!row) return
        try {
            await ElMessageBox.confirm(
                `确定通过申诉并解封用户 #${row.uid} 吗？`,
                '申诉审核',
                { type: 'warning' }
            )
        } catch {
            return
        }

        try {
            const { code, msg } = await axios.put('/api/users/appeal-handle', {
                record_id: row.id,
                action: 'approve',
                reply: '申诉审核通过'
            })
            if (code !== 200) throw new Error(msg)
            ElMessage.success('申诉已通过，用户已解封！')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '操作失败')
        }
    },
    // 申诉驳回
    async appealReject(row) {
        if (!row) return
        try {
            const { value: reply } = await ElMessageBox.prompt(
                '请输入驳回理由',
                '申诉驳回',
                {
                    confirmButtonText: '确认驳回',
                    cancelButtonText: '取消',
                    inputValidator: (val) => !val ? '驳回理由不能为空' : true,
                    inputPlaceholder: '请输入驳回理由...',
                }
            )

            if (!reply) return

            const { code, msg } = await axios.put('/api/users/appeal-handle', {
                record_id: row.id,
                action: 'reject',
                reply: reply
            })
            if (code !== 200) throw new Error(msg)
            ElMessage.success('申诉已驳回！')
            await method.init()
        } catch (error) {
            if (error !== 'cancel' && error !== 'close') {
                ElMessage.error(error.message || '操作失败')
            }
        }
    },
    copy: (text = null, msg = '复制成功！') => {
        if (utils.is.empty(text)) return
        utils.set.copy.text(text)
        ElMessage.info(msg)
    },
}

onMounted(async () => {
    if (props.init) await method.init()
})

watch(() => props.init, (val) => {
    if (val) method.init()
})

defineExpose({ init: method.init })
</script>
