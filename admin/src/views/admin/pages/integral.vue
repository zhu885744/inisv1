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

        <!-- 卡密管理 -->
        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-card>
                    <template #header>
                        <div style="display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px">
                            <div style="display: flex; align-items: center; gap: 8px">
                                <i-svg name="integral" size="18px" style="color: var(--el-color-warning)"></i-svg>
                                <span style="font-weight: 600">积分卡密管理</span>
                                <el-tooltip placement="top">
                                    <template #content>
                                        生成卡密后分发给用户，用户可在「个人中心 - 我的积分」中输入卡密自助兑换积分。
                                    </template>
                                    <el-tag size="small" type="warning">卡密兑换</el-tag>
                                </el-tooltip>
                            </div>
                            <div style="display: flex; align-items: center; gap: 8px">
                                <el-input
                                    v-model="state.card.search"
                                    style="width: 200px"
                                    clearable
                                    autocomplete="off"
                                    placeholder="搜索卡密"
                                    @keyup.enter="method.loadCards(1)"
                                    @clear="method.loadCards(1)"
                                />
                                <el-select v-model="state.card.status" style="width: 120px" clearable placeholder="全部状态" @change="method.loadCards(1)">
                                    <el-option label="未使用" :value="0"></el-option>
                                    <el-option label="已使用" :value="1"></el-option>
                                </el-select>
                                <el-select v-model="state.card.expired" style="width: 120px" clearable placeholder="有效期" @change="method.loadCards(1)">
                                    <el-option label="未过期" value="0"></el-option>
                                    <el-option label="已过期" value="1"></el-option>
                                </el-select>
                                <el-button @click="method.loadCards(1)">查询</el-button>
                                <el-button :loading="state.exporting" @click="method.copyUnusedCards()">复制未使用卡密</el-button>
                                <el-button type="primary" @click="method.openGenerate()">生成卡密</el-button>
                            </div>
                        </div>
                    </template>

                    <!-- 卡密统计 -->
                    <div class="card-stats">
                        <div class="stat-item">
                            <span class="stat-label">卡密总数</span>
                            <strong class="stat-value">{{ state.cardStats.total }}</strong>
                        </div>
                        <div class="stat-item">
                            <span class="stat-label">未使用</span>
                            <strong class="stat-value" style="color: var(--el-color-success)">{{ state.cardStats.unused }}</strong>
                        </div>
                        <div class="stat-item">
                            <span class="stat-label">已使用</span>
                            <strong class="stat-value" style="color: var(--el-color-primary)">{{ state.cardStats.used }}</strong>
                        </div>
                        <div class="stat-item">
                            <span class="stat-label">已过期</span>
                            <strong class="stat-value" style="color: var(--el-color-danger)">{{ state.cardStats.expired }}</strong>
                        </div>
                        <div class="stat-item">
                            <span class="stat-label">累计发放积分</span>
                            <strong class="stat-value">{{ state.cardStats.value_total }}</strong>
                        </div>
                        <div class="stat-item">
                            <span class="stat-label">累计兑换积分</span>
                            <strong class="stat-value">{{ state.cardStats.value_used }}</strong>
                        </div>
                    </div>

                    <el-table
                        :data="state.cardList"
                        border
                        style="width: 100%; margin-top: 12px;"
                        :loading="state.cardLoading"
                    >
                        <el-table-column prop="id" label="ID" width="70" align="center"></el-table-column>
                        <el-table-column prop="card" label="卡密" min-width="180">
                            <template #default="scope">
                                <span class="card-code">{{ scope.row.card }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column prop="value" label="面额" width="90" align="center">
                            <template #default="scope">
                                <span style="font-weight: 600; color: #d4a148">{{ scope.row.value }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="状态" width="100" align="center">
                            <template #default="scope">
                                <el-tag v-if="Number(scope.row.status) === 1" type="success" size="small">已使用</el-tag>
                                <el-tag
                                    v-else-if="Number(scope.row.expire_time) > 0 && Math.floor(Date.now() / 1000) > Number(scope.row.expire_time)"
                                    type="danger"
                                    size="small"
                                >已过期</el-tag>
                                <el-tag v-else type="info" size="small">未使用</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column prop="batch" label="批次" min-width="170"></el-table-column>
                        <el-table-column label="使用者" width="140" align="center" show-overflow-tooltip>
                            <template #default="scope">
                                <span v-if="Number(scope.row.uid) > 0">{{ scope.row.nickname || '未知用户' }}</span>
                                <span v-else>-</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="有效期" width="170" align="center">
                            <template #default="scope">
                                <span v-if="Number(scope.row.expire_time) > 0">{{ utils.time.to.date(scope.row.expire_time, 'Y-m-d H:i:s') }}</span>
                                <el-tag v-else size="small" type="warning">永久有效</el-tag>
                            </template>
                        </el-table-column>
                        <el-table-column label="使用时间" width="170" align="center">
                            <template #default="scope">
                                <span v-if="Number(scope.row.use_time) > 0">{{ utils.time.to.date(scope.row.use_time, 'Y-m-d H:i:s') }}</span>
                                <span v-else>-</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="创建时间" width="170" align="center">
                            <template #default="scope">{{ utils.time.to.date(scope.row.create_time, 'Y-m-d H:i:s') }}</template>
                        </el-table-column>
                        <el-table-column label="操作" width="90" align="center" fixed="right">
                            <template #default="scope">
                                <el-button size="small" type="danger" @click="method.removeCard(scope.row)">删除</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination
                        v-if="state.cardTotal > 0"
                        class="pagination"
                        background
                        layout="total, prev, pager, next, jumper"
                        :total="state.cardTotal"
                        :page-size="state.cardPageSize"
                        :current-page="state.cardPage"
                        @current-change="method.loadCards"
                    />
                </el-card>
            </el-col>
        </el-row>

        <!-- 生成卡密弹窗 -->
        <el-dialog v-model="state.genDialog" class="custom" draggable :close-on-click-modal="false" title="生成积分卡密">
            <template #default>
                <el-form label-width="100px" label-position="left">
                    <el-form-item label="积分面额" required>
                        <el-input-number v-model="state.genForm.value" :min="1" :max="1000000" style="width: 200px"></el-input-number>
                        <span class="form-tip">每张卡密可兑换的积分数</span>
                    </el-form-item>
                    <el-form-item label="生成数量" required>
                        <el-input-number v-model="state.genForm.count" :min="1" :max="1000" style="width: 200px"></el-input-number>
                        <span class="form-tip">单次最多 1000 张</span>
                    </el-form-item>
                    <el-form-item label="卡密长度">
                        <el-input-number v-model="state.genForm.length" :min="8" :max="64" style="width: 200px"></el-input-number>
                        <span class="form-tip">不小于 8 位，默认 16 位</span>
                    </el-form-item>
                    <el-form-item label="有效期">
                        <el-radio-group v-model="state.genForm.expireMode">
                            <el-radio value="forever">永久有效</el-radio>
                            <el-radio value="date">指定日期</el-radio>
                        </el-radio-group>
                    </el-form-item>
                    <el-form-item v-if="state.genForm.expireMode === 'date'" label="失效日期" required>
                        <el-date-picker
                            v-model="state.genForm.expireDate"
                            type="date"
                            placeholder="选择失效日期"
                            value-format="YYYY-MM-DD"
                            style="width: 200px"
                            :disabled-date="method.disabledDate"
                        />
                        <span class="form-tip">所选日期当天 23:59:59 后失效</span>
                    </el-form-item>
                    <el-form-item label="备注">
                        <el-input v-model="state.genForm.remark" placeholder="可选，便于按批次管理" maxlength="100" />
                    </el-form-item>
                </el-form>
            </template>
            <template #footer>
                <el-button @click="state.genDialog = false">取 消</el-button>
                <el-button type="primary" :loading="state.generating" @click="method.generate()">生成</el-button>
            </template>
        </el-dialog>

        <!-- 卡密生成结果弹窗 -->
        <el-dialog v-model="state.resultDialog" class="custom" draggable title="卡密生成结果">
            <el-alert
                type="warning"
                :closable="false"
                show-icon
                title="卡密明文仅本次展示，关闭后无法再次查看，请及时复制或下载保存"
                style="margin-bottom: 12px"
            />
            <div class="result-head">
                <span>
                    批次：{{ state.result.batch }} · 共 <strong>{{ state.result.cards.length }}</strong> 张 · 面额
                    <strong>{{ state.result.value }}</strong>
                </span>
                <div style="display: flex; gap: 8px">
                    <el-button size="small" @click="method.copyCards()">复制全部</el-button>
                    <el-button size="small" @click="method.downloadCards()">下载 TXT</el-button>
                </div>
            </div>
            <el-input v-model="state.result.text" type="textarea" :rows="12" readonly resize="none"></el-input>
        </el-dialog>

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
    },

    // ==================== 卡密管理 ====================
    card: {
        search: '',
        status: '',
        expired: ''
    },
    cardList: [],
    cardTotal: 0,
    cardPage: 1,
    cardPageSize: 20,
    cardLoading: false,
    // 导出未使用卡密（批量复制）进行中
    exporting: false,
    cardStats: {
        total: 0,
        unused: 0,
        used: 0,
        expired: 0,
        value_total: 0,
        value_used: 0
    },
    // 生成卡密弹窗
    genDialog: false,
    generating: false,
    genForm: {
        value: 100,
        count: 1,
        length: 16,
        expireMode: 'forever',
        expireDate: '',
        remark: ''
    },
    // 生成结果弹窗
    resultDialog: false,
    result: {
        batch: '',
        value: 0,
        cards: [],
        text: ''
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
    },

    // ==================== 卡密管理 ====================
    // 卡密列表
    async loadCards(page = state.cardPage) {
        state.cardPage = page
        state.cardLoading = true
        try {
            const params = {
                page: state.cardPage,
                limit: state.cardPageSize,
                order: 'id desc'
            }
            // status 可能为 0（未使用），需显式判断非空
            if (state.card.status !== '' && state.card.status !== null && state.card.status !== undefined) {
                params.status = state.card.status
            }
            if (!utils.is.empty(state.card.expired)) {
                params.expired = state.card.expired
            }
            if (!utils.is.empty(state.card.search)) {
                params.keyword = state.card.search
            }
            const { code, data } = await axios.get('/api/integral/card-all', params)
            if (code === 200) {
                state.cardList = data.data || []
                state.cardTotal = data.count || 0
            } else {
                state.cardList = []
                state.cardTotal = 0
            }
        } finally {
            state.cardLoading = false
        }
    },

    // 卡密统计
    async loadCardStats() {
        try {
            const { code, data } = await axios.get('/api/integral/card-stats')
            if (code === 200 && data) {
                state.cardStats = {
                    total: Number(data.total) || 0,
                    unused: Number(data.unused) || 0,
                    used: Number(data.used) || 0,
                    expired: Number(data.expired) || 0,
                    value_total: Number(data.value_total) || 0,
                    value_used: Number(data.value_used) || 0
                }
            }
        } catch (e) { /* 统计失败不阻断列表展示 */ }
    },

    // 打开生成弹窗（重置表单）
    openGenerate() {
        state.genForm = {
            value: 100,
            count: 1,
            length: 16,
            expireMode: 'forever',
            expireDate: '',
            remark: ''
        }
        state.genDialog = true
    },

    // 失效日期不可早于今天
    disabledDate(date) {
        const today = new Date()
        today.setHours(0, 0, 0, 0)
        return date.getTime() < today.getTime()
    },

    // 生成卡密
    async generate() {
        if (!state.genForm.value || state.genForm.value <= 0) return ElMessage.warning('请输入有效的积分面额')
        if (!state.genForm.count || state.genForm.count <= 0) return ElMessage.warning('请输入生成数量')
        if (state.genForm.length < 8) return ElMessage.warning('卡密长度不能小于 8 位')
        if (state.genForm.expireMode === 'date' && utils.is.empty(state.genForm.expireDate)) {
            return ElMessage.warning('请选择失效日期')
        }

        const payload = {
            value: state.genForm.value,
            count: state.genForm.count,
            length: state.genForm.length,
            remark: state.genForm.remark
        }
        // 永久有效传 expire_time=0；指定日期传 expire（后端按当天 23:59:59 失效）
        if (state.genForm.expireMode === 'date') {
            payload.expire = state.genForm.expireDate
        } else {
            payload.expire_time = 0
        }

        state.generating = true
        try {
            const { code, msg, data } = await axios.post('/api/integral/card-generate', payload)
            if (code !== 200) return ElMessage.error(msg)

            state.result = {
                batch: data?.batch || '',
                value: data?.value || 0,
                cards: data?.cards || [],
                text: (data?.cards || []).join('\n')
            }
            state.genDialog = false
            state.resultDialog = true
            ElMessage.success(`已生成 ${state.result.cards.length} 张卡密`)
            await Promise.all([method.loadCards(1), method.loadCardStats()])
        } finally {
            state.generating = false
        }
    },

    // 删除卡密（软删除）
    async removeCard(row) {
        try {
            await ElMessageBox.confirm(`确定删除卡密 ${row.card} 吗？`, '提示', { type: 'warning' })
        } catch {
            return
        }
        const { code, msg } = await axios.del('/api/integral/card-remove', { ids: [row.id] })
        if (code !== 200) return ElMessage.error(msg)
        ElMessage.success('删除成功')
        await Promise.all([method.loadCards(), method.loadCardStats()])
    },

    // 复制文本到剪贴板（优先 Clipboard API，非安全上下文回退 execCommand）
    async copyText(text) {
        try {
            await navigator.clipboard.writeText(text)
        } catch {
            // http 等非安全上下文下 clipboard 不可用，回退到 textarea 方式
            const el = document.createElement('textarea')
            el.value = text
            el.style.position = 'fixed'
            el.style.opacity = '0'
            document.body.appendChild(el)
            el.select()
            document.execCommand('copy')
            document.body.removeChild(el)
        }
    },

    // 复制本次生成的卡密
    async copyCards() {
        const text = state.result.text || state.result.cards.join('\n')
        if (utils.is.empty(text)) return ElMessage.warning('暂无可复制内容')
        await method.copyText(text)
        ElMessage.success('已复制到剪贴板')
    },

    // 批量复制未使用卡密（仅「未使用且未过期」，不受当前列表筛选影响）
    async copyUnusedCards() {
        state.exporting = true
        try {
            const { code, msg, data } = await axios.get('/api/integral/card-export')
            if (code !== 200) return ElMessage.error(msg)

            const cards = data?.cards || []
            if (cards.length === 0) return ElMessage.warning('暂无未使用卡密')

            await method.copyText(cards.join('\n'))

            if (data?.truncated) {
                ElMessage.warning(`已复制 ${cards.length} 张未使用卡密（超出单次上限，结果已截断）`)
            } else {
                ElMessage.success(`已复制 ${cards.length} 张未使用卡密`)
            }
        } finally {
            state.exporting = false
        }
    },

    // 下载卡密 TXT
    downloadCards() {
        const cards = state.result.cards || []
        if (cards.length === 0) return ElMessage.warning('暂无可下载内容')
        const blob = new Blob([cards.join('\n')], { type: 'text/plain;charset=utf-8' })
        const url = URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `积分卡密_${state.result.batch || Date.now()}.txt`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        URL.revokeObjectURL(url)
    }
}

onMounted(async () => {
    await Promise.all([
        method.loadUsers(),
        method.loadCards(1),
        method.loadCardStats()
    ])
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

/* ---------- 卡密统计 ---------- */
.card-stats {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 12px;
    padding: 12px 14px;
    background: var(--el-fill-color-light);
    border-radius: 8px;
}
.stat-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
}
.stat-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
.stat-value {
    font-size: 18px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
}
.card-code {
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    letter-spacing: 1px;
    word-break: break-all;
}

/* ---------- 表单 / 结果弹窗 ---------- */
.form-tip {
    margin-left: 10px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
.result-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 10px;
    font-size: 13px;
    color: var(--el-text-color-regular);
}
.result-head strong {
    color: var(--el-color-warning);
}

@media (max-width: 992px) {
    .card-stats {
        grid-template-columns: repeat(3, 1fr);
    }
}
</style>
