<template>
    <div class="container-box">
        <el-row :gutter="20">
            <el-col :span="12" style="display: flex;">
                <div style="margin-right: 8px">
                    <el-input v-model="state.item.search" style="width: 220px" autocomplete="new-password" type="text" placeholder="搜索用户昵称" />
                </div>
                <el-button @click="method.loadUsers()">刷新</el-button>
            </el-col>
            <el-col :span="12" style="display: flex; justify-content: flex-end; gap: 8px">
                <el-button @click="method.addIntegral()">调整积分</el-button>
                <el-button type="primary" :disabled="state.selected.length === 0" @click="method.batchIntegral()">
                    批量发放<template v-if="state.selected.length > 0">（已选 {{ state.selected.length }} 人）</template>
                </el-button>
            </el-col>
        </el-row>

        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-card>
                    <template #header>
                        <div style="display: flex; align-items: center; justify-content: space-between">
                            <div style="display: flex; align-items: center; gap: 8px">
                                <i-svg name="integral" size="18px" style="color: var(--el-color-warning)"></i-svg>
                                <span style="font-weight: 600">用户积分管理</span>
                                <el-tooltip placement="top">
                                    <template #content>
                                        用户通过签到、登录、发布内容等任务赚取积分；积分规则可在「系统配置 → 配置 → 积分规则」中调整。
                                    </template>
                                    <el-tag size="small" type="warning">积分规则</el-tag>
                                </el-tooltip>
                            </div>
                            <span style="font-size: 12px; color: var(--el-text-color-secondary)">
                                勾选用户后可批量发放/扣除积分
                            </span>
                        </div>
                    </template>
                    <el-table
                        :data="state.userList"
                        border
                        style="width: 100%;"
                        :loading="state.tableLoading"
                        @selection-change="method.onSelectionChange"
                    >
                        <el-table-column type="selection" width="46" align="center"></el-table-column>
                        <el-table-column prop="id" label="用户ID" width="80" align="center"></el-table-column>
                        <el-table-column prop="nickname" label="用户昵称" min-width="140"></el-table-column>
                        <el-table-column prop="integral" label="积分余额" width="120" align="center">
                            <template #default="scope">
                                <span style="font-weight: 600; color: #d4a148">{{ scope.row.integral || 0 }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column prop="exp" label="经验值" width="100" align="center">
                            <template #default="scope">
                                <span>{{ scope.row.exp || 0 }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="等级" width="120" align="center">
                            <template #default="scope">
                                <el-tag v-if="scope.row.result?.level?.current?.name" size="small" type="info">
                                    {{ scope.row.result.level.current.name }}
                                </el-tag>
                                <span v-else style="color: var(--el-text-color-secondary)">-</span>
                            </template>
                        </el-table-column>
                        <el-table-column prop="create_time" label="注册时间" width="180" align="center">
                            <template #default="scope">
                                <span>{{ utils.time.to.date(scope.row.create_time, 'Y-m-d H:i:s') }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="操作" width="120" align="center">
                            <template #default="scope">
                                <el-button size="small" @click="method.addIntegral(scope.row)">调整积分</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination
                        v-if="state.total > 0"
                        class="pagination"
                        background
                        layout="total, prev, pager, next, jumper"
                        :total="state.total"
                        :page-size="state.pageSize"
                        :current-page="state.page"
                        @current-change="method.loadUsers"
                    />
                </el-card>
            </el-col>
        </el-row>

        <!-- 调整积分弹窗（单个 / 批量共用） -->
        <el-dialog v-model="state.dialog" class="custom" draggable :close-on-click-modal="false">
            <template #header>
                <strong class="flex-center">{{ state.form.batch ? '批量调整积分' : '调整用户积分' }}</strong>
            </template>
            <template #default>
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="用户">
                        <!-- 批量模式：展示已选用户，不允许再改 -->
                        <div v-if="state.form.batch">
                            <el-tag
                                v-for="user in state.form.users"
                                :key="user.id"
                                size="small"
                                style="margin: 0 6px 6px 0"
                            >
                                {{ user.nickname }}（ID:{{ user.id }}）
                            </el-tag>
                        </div>
                        <el-select v-else v-model="state.form.uid" placeholder="请选择用户" filterable :disabled="!!state.form.nickname">
                            <el-option v-for="user in state.userList" :key="user.id" :label="`${user.nickname}（ID:${user.id}）`" :value="user.id"></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="积分变动">
                        <div style="display: flex; align-items: center; gap: 8px;">
                            <el-button size="small" @click="state.form.value = -Math.abs(state.form.value)">扣除</el-button>
                            <el-input-number v-model="state.form.value" :min="-99999" :max="99999" style="width: 200px"></el-input-number>
                            <el-button size="small" @click="state.form.value = Math.abs(state.form.value)">增加</el-button>
                        </div>
                        <span style="font-size: 12px; color: var(--el-text-color-secondary);">正数为增加，负数为扣除</span>
                    </el-form-item>
                    <el-form-item label="描述">
                        <el-input v-model="state.form.description" placeholder="请输入操作描述（可选）"></el-input>
                    </el-form-item>
                    <el-alert
                        v-if="state.form.batch"
                        type="warning"
                        :closable="false"
                        show-icon
                        :title="`将对 ${state.form.users.length} 位用户分别执行该积分的发放/扣除，失败的用户不会影响其它用户`"
                    />
                </el-form>
            </template>
            <template #footer>
                <el-button @click="state.dialog = false">取 消</el-button>
                <el-button type="primary" @click="method.save()" :loading="state.saving">保 存</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'

const { ctx, proxy } = getCurrentInstance()

const state = reactive({
    item: {
        search: null,
        timer: null
    },
    userList: [],
    selected: [],
    total: 0,
    page: 1,
    pageSize: 20,
    tableLoading: false,
    dialog: false,
    saving: false,
    form: {
        batch: false,
        uid: null,
        nickname: '',
        users: [],
        value: 0,
        description: ''
    }
})

const method = {
    async loadUsers(page = state.page) {
        state.page = page
        state.tableLoading = true
        try {
            const params = {
                page: state.page,
                limit: state.pageSize,
                order: 'create_time desc'
            }
            if (!utils.is.empty(state.item.search)) {
                params.like = [['nickname', `%${state.item.search}%`]]
            }
            const { code, data } = await axios.get('/api/users/all', params)
            if (code === 200) {
                state.userList = data.data || []
                state.total = data.count || 0
            }
        } finally {
            state.tableLoading = false
        }
    },
    onSelectionChange(rows) {
        state.selected = rows || []
    },
    addIntegral(user = null) {
        if (user) {
            state.form = { batch: false, uid: user.id, nickname: user.nickname || '', users: [], value: 0, description: '' }
        } else {
            state.form = { batch: false, uid: null, nickname: '', users: [], value: 0, description: '' }
        }
        state.dialog = true
    },
    batchIntegral() {
        if (state.selected.length === 0) return ElMessage.warning('请先勾选用户')
        state.form = {
            batch: true,
            uid: null,
            nickname: '',
            users: state.selected.map(item => ({ id: item.id, nickname: item.nickname })),
            value: 0,
            description: ''
        }
        state.dialog = true
    },
    async save() {
        if (!state.form.batch && !state.form.uid) return ElMessage.warning('请选择用户')
        if (state.form.value === 0) return ElMessage.warning('请输入积分变动值')

        state.saving = true
        try {
            const payload = {
                value: state.form.value,
                description: state.form.description
            }
            // 批量：uids 数组；单个：uid
            if (state.form.batch) {
                payload.uids = state.form.users.map(item => item.id)
            } else {
                payload.uid = state.form.uid
            }

            const { code, msg, data } = await axios.post('/api/integral/give', payload)
            if (code !== 200) return ElMessage.error(msg)

            if (state.form.batch) {
                ElMessage.success(`批量调整完成：成功 ${data?.success ?? 0} 个，失败 ${data?.failed ?? 0} 个`)
            } else {
                ElMessage.success('调整成功')
            }

            state.dialog = false
            await method.loadUsers()
        } finally {
            state.saving = false
        }
    }
}

onMounted(async () => {
    await method.loadUsers()
})

watch(() => state.item.search, (val) => {
    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => {
        method.loadUsers(1)
    }, globalThis.inis?.lazy_time ?? 500)
})
</script>

<style scoped>
.pagination {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
}
</style>
