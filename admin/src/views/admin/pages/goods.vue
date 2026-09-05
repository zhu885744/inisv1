<template>
    <div class="container-box">
        <!-- 商品管理 -->
        <el-card>
            <template #header>
                <div style="display: flex; align-items: center; justify-content: space-between">
                    <div style="display: flex; align-items: center; gap: 8px">
                        <i-svg name="level" size="18px" style="color: var(--el-color-primary)"></i-svg>
                        <span style="font-weight: 600">商品管理</span>
                        <el-tag size="small" type="info">积分兑换</el-tag>
                    </div>
                    <div style="display: flex; align-items: center; gap: 8px">
                        <el-button type="primary" size="small" @click="method.add()">新增商品</el-button>
                        <el-button size="small" @click="method.refreshAll()">刷新数据</el-button>
                    </div>
                </div>
            </template>
            <el-table :data="state.goodsList" border style="width: 100%;" v-loading="state.loading">
                <el-table-column prop="id" label="ID" width="70" align="center"></el-table-column>
                <el-table-column prop="cover" label="封面" width="80" align="center">
                    <template #default="scope">
                        <img v-if="scope.row.cover" :src="scope.row.cover" style="width:40px;height:40px;border-radius:6px;object-fit:cover" />
                        <span v-else>-</span>
                    </template>
                </el-table-column>
                <el-table-column prop="title" label="商品名称" min-width="140"></el-table-column>
                <el-table-column prop="price" label="积分价格" width="100" align="center">
                    <template #default="scope">
                        <span style="font-weight:600;color:#d4a148">{{ scope.row.price }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="stock" label="库存" width="80" align="center"></el-table-column>
                <el-table-column prop="type" label="类型" width="90" align="center">
                    <template #default="scope">
                        <el-tag :type="scope.row.type === 'physical' ? 'warning' : 'success'" size="small">
                            {{ scope.row.type === 'physical' ? '实物' : '虚拟' }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" width="90" align="center">
                    <template #default="scope">
                        <el-tag :type="Number(scope.row.status) === 1 ? 'success' : 'info'" size="small">
                            {{ Number(scope.row.status) === 1 ? '上架' : '下架' }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column prop="create_time" label="创建时间" width="170" align="center">
                    <template #default="scope">
                        <span>{{ utils.time.to.date(scope.row.create_time, 'Y-m-d H:i:s') }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="150" align="center" fixed="right">
                    <template #default="scope">
                        <el-button size="small" @click="method.edit(scope.row)">编辑</el-button>
                        <el-button size="small" type="danger" @click="method.remove(scope.row.id)">删除</el-button>
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
                @current-change="method.loadGoods"
            />
        </el-card>

        <!-- 订单管理 -->
        <el-card style="margin-top: 12px">
            <template #header>
                <div style="display: flex; align-items: center; gap: 8px">
                    <i-svg name="level" size="18px" style="color: var(--el-color-primary)"></i-svg>
                    <span style="font-weight: 600">订单管理</span>
                    <el-tag size="small" type="warning">用户兑换处理</el-tag>
                </div>
            </template>
            <el-table :data="state.orderList" border style="width: 100%;" v-loading="state.orderLoading">
                <el-table-column prop="id" label="订单ID" width="80" align="center"></el-table-column>
                <el-table-column label="用户" width="150">
                    <template #default="scope">
                        <span>{{ scope.row.result?.user?.nickname || `用户#${scope.row.uid}` }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="商品" min-width="150">
                    <template #default="scope">
                        <span>{{ scope.row.result?.goods?.title || `商品#${scope.row.goods_id}` }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="price" label="消耗积分" width="100" align="center">
                    <template #default="scope">
                        <span style="font-weight:600;color:#d4a148">{{ scope.row.price }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="status" label="状态" width="90" align="center">
                    <template #default="scope">
                        <el-tag :type="method.orderStatusTag(scope.row.status)" size="small">
                            {{ method.orderStatusText(scope.row.status) }}
                        </el-tag>
                    </template>
                </el-table-column>
                <el-table-column label="收货信息" min-width="160">
                    <template #default="scope">
                        <template v-if="scope.row.result?.address">
                            <span>{{ scope.row.result.address.name }} {{ scope.row.result.address.phone }}</span>
                            <div style="font-size: 12px; color: var(--el-text-color-secondary)">{{ scope.row.result.address.address }}</div>
                        </template>
                        <span v-else style="color: var(--el-text-color-secondary)">-</span>
                    </template>
                </el-table-column>
                <el-table-column label="发货内容/物流" min-width="150">
                    <template #default="scope">
                        <el-tooltip v-if="scope.row.deliver_content" placement="top" :content="scope.row.deliver_content">
                            <span>{{ scope.row.deliver_content }}</span>
                        </el-tooltip>
                        <el-tooltip v-else-if="scope.row.logistics" placement="top" :content="scope.row.logistics">
                            <span>{{ scope.row.logistics }}</span>
                        </el-tooltip>
                        <span v-else style="color: var(--el-text-color-secondary)">-</span>
                    </template>
                </el-table-column>
                <el-table-column prop="create_time" label="下单时间" width="170" align="center">
                    <template #default="scope">
                        <span>{{ utils.time.to.date(scope.row.create_time, 'Y-m-d H:i:s') }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="操作" width="160" align="center" fixed="right">
                    <template #default="scope">
                        <el-button v-if="Number(scope.row.status) === 0" size="small" type="primary" @click="method.openShip(scope.row)">发货</el-button>
                        <el-button v-if="Number(scope.row.status) === 1" size="small" type="success" @click="method.setOrderStatus(scope.row.id, 2)">完成</el-button>
                    </template>
                </el-table-column>
            </el-table>
            <el-pagination
                v-if="state.orderTotal > 0"
                class="pagination"
                background
                layout="total, prev, pager, next, jumper"
                :total="state.orderTotal"
                :page-size="state.pageSize"
                :current-page="state.orderPage"
                @current-change="method.loadOrders"
            />
        </el-card>

        <!-- 积分规则配置 -->
        <el-card style="margin-top: 12px">
            <template #header>
                <div style="display: flex; align-items: center; gap: 8px">
                    <span style="font-weight: 600">积分任务规则</span>
                    <el-tag size="small" type="warning">用户赚取积分</el-tag>
                </div>
            </template>
            <el-table :data="state.rulesList" border style="width: 100%;">
                <el-table-column prop="name" label="任务名称" min-width="140">
                    <template #default="scope">
                        <el-input v-model="scope.row.name" size="small" />
                    </template>
                </el-table-column>
                <el-table-column prop="value" label="积分" width="140">
                    <template #default="scope">
                        <el-input-number v-model="scope.row.value" :min="0" :max="9999" size="small" />
                    </template>
                </el-table-column>
                <el-table-column prop="daily_limit" label="每日限制次数" width="160">
                    <template #default="scope">
                        <div style="display: flex; align-items: center; gap: 4px;">
                            <el-input-number v-model="scope.row.daily_limit" :min="0" :max="999" size="small" />
                            <span style="font-size: 11px; color: var(--el-text-color-secondary)">（0=不限制）</span>
                        </div>
                    </template>
                </el-table-column>
                <el-table-column prop="type" label="类型" min-width="120">
                    <template #default="scope">
                        <span style="font-size: 12px; color: var(--el-text-color-secondary)">{{ scope.row.type }}</span>
                    </template>
                </el-table-column>
            </el-table>
            <div style="margin-top: 12px; display: flex; justify-content: flex-end">
                <el-button @click="method.resetRules()">重置默认</el-button>
                <el-button type="primary" @click="method.saveRules()" :loading="state.rulesSaving">保存规则</el-button>
            </div>
        </el-card>

        <!-- 商品编辑弹窗 -->
        <el-dialog v-model="state.dialog" class="custom" draggable :close-on-click-modal="false">
            <template #header>
                <strong class="flex-center">{{ utils.is.empty(state.form.id) ? '新增' : '编辑' }}商品</strong>
            </template>
            <template #default>
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="商品名称">
                        <el-input v-model="state.form.title" placeholder="请输入商品名称"></el-input>
                    </el-form-item>
                    <el-form-item label="商品描述">
                        <el-input v-model="state.form.description" type="textarea" :rows="3" placeholder="请输入商品描述"></el-input>
                    </el-form-item>
                    <el-form-item label="封面图">
                        <el-input v-model="state.form.cover" placeholder="请输入封面图片 URL"></el-input>
                    </el-form-item>
                    <el-form-item label="积分价格">
                        <el-input-number v-model="state.form.price" :min="0" :max="99999" style="width: 200px"></el-input-number>
                    </el-form-item>
                    <el-form-item label="库存">
                        <el-input-number v-model="state.form.stock" :min="0" :max="99999" style="width: 200px"></el-input-number>
                    </el-form-item>
                    <el-form-item label="商品类型">
                        <el-radio-group v-model="state.form.type">
                            <el-radio value="virtual">虚拟商品</el-radio>
                            <el-radio value="physical">实物商品</el-radio>
                        </el-radio-group>
                    </el-form-item>
                    <template v-if="state.form.type !== 'physical'">
                        <el-form-item label="发货方式">
                            <el-radio-group v-model="state.form.deliver_type">
                                <el-radio value="text">文本内容</el-radio>
                                <el-radio value="card">卡密</el-radio>
                            </el-radio-group>
                        </el-form-item>
                        <el-form-item v-if="state.form.deliver_type === 'text'" label="文本内容">
                            <el-input v-model="state.form.deliver_content" type="textarea" :rows="3" placeholder="购买后直接展示给用户的文本内容"></el-input>
                        </el-form-item>
                        <el-form-item v-if="state.form.deliver_type === 'card'" label="卡密池">
                            <el-input v-model="state.form.cards_text" type="textarea" :rows="6" placeholder="每行一个卡密"></el-input>
                            <span style="font-size: 12px; color: var(--el-text-color-secondary);">每行一个卡密，购买后随机发放一个</span>
                        </el-form-item>
                    </template>
                    <el-form-item label="上架状态">
                        <el-switch v-model="state.form.status" :active-value="1" :inactive-value="0" active-text="上架" inactive-text="下架" />
                    </el-form-item>
                </el-form>
            </template>
            <template #footer>
                <el-button @click="state.dialog = false">取 消</el-button>
                <el-button type="primary" @click="method.save()" :loading="state.saving">保 存</el-button>
            </template>
        </el-dialog>

        <!-- 发货弹窗 -->
        <el-dialog v-model="state.shipDialog" class="custom" draggable :close-on-click-modal="false">
            <template #header>
                <strong class="flex-center">订单发货</strong>
            </template>
            <template #default>
                <el-form label-width="90px" label-position="left">
                    <el-form-item label="收货信息">
                        <div v-if="state.shipForm.address" style="font-size: 13px; color: var(--el-text-color-secondary)">
                            <div>{{ state.shipForm.address.name }} · {{ state.shipForm.address.phone }}</div>
                            <div>{{ state.shipForm.address.address }}</div>
                        </div>
                        <span v-else>虚拟商品，无需收货地址</span>
                    </el-form-item>
                    <el-form-item label="物流信息">
                        <el-input v-model="state.shipForm.logistics" type="textarea" :rows="3" placeholder="请输入物流公司及单号（可选）"></el-input>
                    </el-form-item>
                </el-form>
            </template>
            <template #footer>
                <el-button @click="state.shipDialog = false">取 消</el-button>
                <el-button type="primary" @click="method.confirmShip()" :loading="state.shipSaving">确认发货</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'

const { ctx, proxy } = getCurrentInstance()

const DEFAULT_RULES = {
    'check-in': { name: '每日签到', value: 5, daily_limit: 1 },
    'login': { name: '每日登录', value: 2, daily_limit: 1 },
    'article-create': { name: '发布文章', value: 10, daily_limit: 5 },
    'comment': { name: '发表评论', value: 2, daily_limit: 10 },
    'moments': { name: '发布动态', value: 20, daily_limit: 1 }
}

const state = reactive({
    goodsList: [],
    total: 0,
    page: 1,
    pageSize: 20,
    loading: false,
    dialog: false,
    saving: false,
    form: {},
    rulesList: [],
    rulesSaving: false,
    orderList: [],
    orderTotal: 0,
    orderPage: 1,
    orderLoading: false,
    shipDialog: false,
    shipSaving: false,
    shipForm: {}
})

const method = {
    async loadGoods(page = state.page) {
        state.page = page
        state.loading = true
        try {
            const { code, data } = await axios.get('/api/goods/all', {
                page: state.page,
                limit: state.pageSize,
                order: 'create_time desc'
            })
            if (code === 200) {
                state.goodsList = data.data || []
                state.total = data.count || 0
            }
        } finally {
            state.loading = false
        }
    },
    add() {
        state.form = { title: '', description: '', cover: '', price: 0, stock: 0, status: 1, type: 'virtual', deliver_type: 'text', deliver_content: '', cards_text: '' }
        state.dialog = true
    },
    edit(row) {
        state.form = { ...row }
        // 卡密池 JSON 数组转成每行一个的文本
        let cards = []
        try { cards = JSON.parse(row.cards || '[]') } catch { cards = [] }
        state.form.cards_text = Array.isArray(cards) ? cards.join('\n') : ''
        state.dialog = true
    },
    async save() {
        if (utils.is.empty(state.form.title)) return ElMessage.warning('请输入商品名称')
        // 卡密商品：卡密池文本转 JSON 数组
        if (state.form.type === 'virtual' && state.form.deliver_type === 'card') {
            const cards = (state.form.cards_text || '').split('\n').map(s => s.trim()).filter(Boolean)
            if (cards.length === 0) return ElMessage.warning('请填写卡密池（每行一个）')
            state.form.cards = JSON.stringify(cards)
        }
        state.saving = true
        try {
            const { code, msg } = await axios.post('/api/goods/save', state.form)
            if (code !== 200) return ElMessage.error(msg)
            ElMessage.success('保存成功')
            state.dialog = false
            await method.loadGoods()
        } finally {
            state.saving = false
        }
    },
    async remove(id) {
        try {
            await ElMessageBox.confirm('确定要删除该商品吗？', '提示', { type: 'warning' })
        } catch {
            return
        }
        const { code, msg } = await axios.del('/api/goods/remove', { ids: [id] })
        if (code !== 200) return ElMessage.error(msg)
        ElMessage.success('删除成功')
        await method.loadGoods()
    },
    async loadRules() {
        const { code, data } = await axios.get('/api/config/one', { key: 'SYSTEM_INTEGRAL_RULES' })
        if (code !== 200) return
        const json = data?.json || {}
        state.rulesList = Object.keys(DEFAULT_RULES).map(key => ({
            type: key,
            name: json[key]?.name || DEFAULT_RULES[key].name,
            value: Number(json[key]?.value ?? DEFAULT_RULES[key].value),
            daily_limit: Number(json[key]?.daily_limit ?? DEFAULT_RULES[key].daily_limit)
        }))
    },
    resetRules() {
        state.rulesList = Object.keys(DEFAULT_RULES).map(key => ({
            type: key,
            name: DEFAULT_RULES[key].name,
            value: DEFAULT_RULES[key].value,
            daily_limit: DEFAULT_RULES[key].daily_limit
        }))
        ElMessage.success('已重置为默认值')
    },
    async saveRules() {
        state.rulesSaving = true
        try {
            const rules = {}
            state.rulesList.forEach(item => {
                rules[item.type] = { name: item.name, value: item.value, daily_limit: item.daily_limit }
            })
            const { code, msg } = await axios.post('/api/config/save', {
                key: 'SYSTEM_INTEGRAL_RULES',
                json: JSON.stringify(rules)
            })
            if (code !== 200) return ElMessage.error('保存失败：' + msg)
            ElMessage.success('保存成功')
        } finally {
            state.rulesSaving = false
        }
    },
    async loadOrders(page = state.orderPage) {
        state.orderPage = page
        state.orderLoading = true
        try {
            const { code, data } = await axios.get('/api/goods/orders-all', {
                page: state.orderPage,
                limit: state.pageSize,
                order: 'create_time desc'
            })
            if (code === 200) {
                state.orderList = data.data || []
                state.orderTotal = data.count || 0
            }
        } finally {
            state.orderLoading = false
        }
    },
    async setOrderStatus(id, status) {
        const { code, msg } = await axios.put('/api/goods/order-status', { id, status })
        if (code !== 200) return ElMessage.error(msg)
        ElMessage.success('更新成功')
        await method.loadOrders()
    },
    openShip(row) {
        state.shipForm = { id: row.id, logistics: '', address: row.result?.address || null }
        state.shipDialog = true
    },
    async confirmShip() {
        state.shipSaving = true
        try {
            const { code, msg } = await axios.put('/api/goods/order-status', {
                id: state.shipForm.id,
                status: 1,
                logistics: state.shipForm.logistics
            })
            if (code !== 200) return ElMessage.error(msg)
            ElMessage.success('发货成功')
            state.shipDialog = false
            await method.loadOrders()
        } finally {
            state.shipSaving = false
        }
    },
    orderStatusText: (s) => {
        const map = { 0: '待发货', 1: '已发货', 2: '已完成' }
        return map[s] ?? '未知'
    },
    orderStatusTag: (s) => {
        const map = { 0: 'warning', 1: 'primary', 2: 'success' }
        return map[s] ?? 'info'
    },
    async refreshAll() {
        await Promise.all([method.loadGoods(), method.loadRules(), method.loadOrders()])
        ElMessage.success('已刷新')
    }
}

onMounted(async () => {
    await method.loadGoods()
    await method.loadRules()
    await method.loadOrders()
})
</script>

<style scoped>
.pagination {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
}
</style>
