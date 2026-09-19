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
            <!-- 商城运营统计 -->
            <div class="stats-bar">
                <div class="stat-cell">
                    <span class="stat-label">商品总数</span>
                    <span class="stat-value">{{ state.stats.goods.total }}</span>
                </div>
                <div class="stat-cell">
                    <span class="stat-label">上架中</span>
                    <span class="stat-value">{{ state.stats.goods.on }}</span>
                </div>
                <div class="stat-cell">
                    <span class="stat-label">库存预警</span>
                    <span class="stat-value" :class="{ warn: state.stats.goods.stock_warn > 0 }">{{ state.stats.goods.stock_warn }}</span>
                </div>
                <div class="stat-cell">
                    <span class="stat-label">待发货</span>
                    <span class="stat-value" :class="{ warn: state.stats.order.pending > 0 }">{{ state.stats.order.pending }}</span>
                </div>
                <div class="stat-cell">
                    <span class="stat-label">累计消耗积分</span>
                    <span class="stat-value gold">{{ state.stats.integral.spent }}</span>
                </div>
                <div class="stat-cell">
                    <span class="stat-label">累计退还积分</span>
                    <span class="stat-value">{{ state.stats.integral.refund }}</span>
                </div>
            </div>

            <!-- 商品筛选 -->
            <div class="filter-bar">
                <el-input
                    v-model="state.filter.keyword"
                    placeholder="搜索商品名称"
                    clearable
                    style="width: 200px"
                    @keyup.enter="method.loadGoods(1)"
                    @clear="method.loadGoods(1)"
                />
                <el-input
                    v-model="state.filter.category"
                    placeholder="商品分类"
                    clearable
                    style="width: 150px"
                    @keyup.enter="method.loadGoods(1)"
                    @clear="method.loadGoods(1)"
                />
                <el-select v-model="state.filter.status" placeholder="全部状态" clearable style="width: 130px" @change="method.loadGoods(1)">
                    <el-option label="上架" :value="1" />
                    <el-option label="下架" :value="0" />
                </el-select>
                <el-button type="primary" @click="method.loadGoods(1)">查询</el-button>
                <el-button @click="method.resetFilter()">重置</el-button>
            </div>

            <el-table :data="state.goodsList" border style="width: 100%;" v-loading="state.loading">
                <el-table-column prop="id" label="ID" width="70" align="center"></el-table-column>
                <el-table-column prop="cover" label="封面" width="80" align="center">
                    <template #default="scope">
                        <img v-if="scope.row.cover" :src="scope.row.cover" style="width:40px;height:40px;border-radius:6px;object-fit:cover" />
                        <span v-else>-</span>
                    </template>
                </el-table-column>
                <el-table-column prop="title" label="商品名称" min-width="140"></el-table-column>
                <el-table-column label="分类" width="110" align="center">
                    <template #default="scope">
                        <el-tag v-if="scope.row.category" size="small" type="info">{{ scope.row.category }}</el-tag>
                        <span v-else style="color: var(--el-text-color-secondary)">-</span>
                    </template>
                </el-table-column>
                <el-table-column prop="price" label="积分价格" width="100" align="center">
                    <template #default="scope">
                        <span style="font-weight:600;color:#d4a148">{{ scope.row.price }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="stock" label="库存" width="80" align="center">
                    <template #default="scope">
                        <span :style="Number(scope.row.stock) <= 5 ? 'color: var(--el-color-danger); font-weight: 600' : ''">
                            {{ scope.row.stock }}
                        </span>
                    </template>
                </el-table-column>
                <el-table-column prop="sold" label="销量" width="80" align="center">
                    <template #default="scope">
                        <span>{{ scope.row.sold || 0 }}</span>
                    </template>
                </el-table-column>
                <el-table-column label="兑换限制" min-width="150">
                    <template #default="scope">
                        <div style="display: flex; flex-wrap: wrap; gap: 4px">
                            <el-tag v-if="Number(scope.row.limit_per_user) > 0" size="small" type="warning">
                                限购 {{ scope.row.limit_per_user }}
                            </el-tag>
                            <el-tag v-if="Number(scope.row.min_exp) > 0" size="small" type="info">
                                需经验 {{ scope.row.min_exp }}
                            </el-tag>
                            <el-tag v-if="method.timeWindowText(scope.row)" size="small" type="success">
                                {{ method.timeWindowText(scope.row) }}
                            </el-tag>
                            <span
                                v-if="!Number(scope.row.limit_per_user) && !Number(scope.row.min_exp) && !method.timeWindowText(scope.row)"
                                style="color: var(--el-text-color-secondary)"
                            >不限</span>
                        </div>
                    </template>
                </el-table-column>
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
                <div style="display: flex; align-items: center; justify-content: space-between; gap: 8px">
                    <div style="display: flex; align-items: center; gap: 8px">
                        <i-svg name="level" size="18px" style="color: var(--el-color-primary)"></i-svg>
                        <span style="font-weight: 600">订单管理</span>
                        <el-tag size="small" type="warning">用户兑换处理</el-tag>
                    </div>
                    <div style="display: flex; align-items: center; gap: 8px">
                        <el-select v-model="state.orderFilter.status" placeholder="全部状态" clearable style="width: 130px" @change="method.loadOrders(1)">
                            <el-option v-for="opt in method.orderStatusOptions()" :key="opt.value" :label="opt.label" :value="opt.value" />
                        </el-select>
                        <el-button size="small" @click="method.loadOrders(1)">刷新订单</el-button>
                    </div>
                </div>
            </template>
            <el-table :data="state.orderList" border style="width: 100%;" v-loading="state.orderLoading">
                <el-table-column prop="id" label="订单ID" width="80" align="center"></el-table-column>
                <el-table-column prop="order_no" label="订单号" min-width="180" align="center">
                    <template #default="scope">
                        <span style="font-size: 12px">{{ scope.row.order_no || '-' }}</span>
                    </template>
                </el-table-column>
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
                <el-table-column label="操作" width="190" align="center" fixed="right">
                    <template #default="scope">
                        <el-button v-if="Number(scope.row.status) === 0" size="small" type="primary" @click="method.openShip(scope.row)">发货</el-button>
                        <el-button v-if="Number(scope.row.status) === 1" size="small" type="success" @click="method.setOrderStatus(scope.row.id, 2)">完成</el-button>
                        <el-button
                            v-if="Number(scope.row.status) === 0"
                            size="small"
                            type="danger"
                            @click="method.cancelOrder(scope.row)"
                        >取消退款</el-button>
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

        <!-- 积分规则配置（与系统配置共用同一组件，避免两处维护） -->
        <div style="margin-top: 12px">
            <atom-integral-rules ref="integral-rules" />
        </div>

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
                    <el-form-item label="分类">
                        <el-input v-model="state.form.category" placeholder="如 vip / coupon（用于前台分类筛选，可留空）"></el-input>
                    </el-form-item>
                    <el-form-item label="每人限购">
                        <el-input-number v-model="state.form.limit_per_user" :min="0" :max="9999" style="width: 200px"></el-input-number>
                        <span style="font-size: 12px; color: var(--el-text-color-secondary); margin-left: 8px">0 = 不限购</span>
                    </el-form-item>
                    <el-form-item label="经验门槛">
                        <el-input-number v-model="state.form.min_exp" :min="0" :max="9999999" style="width: 200px"></el-input-number>
                        <span style="font-size: 12px; color: var(--el-text-color-secondary); margin-left: 8px">0 = 不限</span>
                    </el-form-item>
                    <el-form-item label="兑换时间">
                        <el-date-picker
                            v-model="state.form.start_time"
                            type="datetime"
                            value-format="X"
                            placeholder="开始时间（可留空）"
                            style="width: 200px"
                        />
                        <span style="margin: 0 6px">至</span>
                        <el-date-picker
                            v-model="state.form.end_time"
                            type="datetime"
                            value-format="X"
                            placeholder="结束时间（可留空）"
                            style="width: 200px"
                        />
                    </el-form-item>
                    <el-form-item label="排序权重">
                        <el-input-number v-model="state.form.sort" :min="0" :max="99999" style="width: 200px"></el-input-number>
                        <span style="font-size: 12px; color: var(--el-text-color-secondary); margin-left: 8px">越大越靠前</span>
                    </el-form-item>
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
import AtomIntegralRules from '{src}/comps/admin/atom/integral-rules.vue'

const { ctx, proxy } = getCurrentInstance()

const state = reactive({
    goodsList: [],
    total: 0,
    page: 1,
    pageSize: 20,
    loading: false,
    dialog: false,
    saving: false,
    form: {},
    // 商城统计
    stats: {
        goods: { total: 0, on: 0, off: 0, stock_warn: 0 },
        order: { total: 0, pending: 0, shipped: 0, completed: 0, canceled: 0 },
        integral: { spent: 0, refund: 0, net: 0 }
    },
    // 商品筛选
    filter: {
        keyword: '',
        category: '',
        status: null
    },
    orderList: [],
    orderTotal: 0,
    orderPage: 1,
    orderLoading: false,
    orderFilter: {
        status: null
    },
    shipDialog: false,
    shipSaving: false,
    shipForm: {}
})

const method = {
    async loadGoods(page = state.page) {
        state.page = page
        state.loading = true
        try {
            const params = {
                page: state.page,
                limit: state.pageSize,
                order: 'sort desc, id desc'
            }
            if (!utils.is.empty(state.filter.keyword)) params.keyword = state.filter.keyword
            if (!utils.is.empty(state.filter.category)) params.category = state.filter.category
            if (state.filter.status !== null && state.filter.status !== '') params.status = state.filter.status

            const { code, msg, data } = await axios.get('/api/goods/all', params)
            if (code !== 200) {
                ElMessage.error('商品列表加载失败：' + (msg || '未知错误'))
                state.goodsList = []
                state.total = 0
                return
            }
            state.goodsList = data.data || []
            state.total = data.count || 0
        } catch (e) {
            ElMessage.error('商品列表加载异常：' + (e?.message || e))
        } finally {
            state.loading = false
        }
    },
    resetFilter() {
        state.filter = { keyword: '', category: '', status: null }
        method.loadGoods(1)
    },
    async loadStats() {
        try {
            const { code, msg, data } = await axios.get('/api/goods/stats')
            // 不再静默失败：接口异常时明确提示，避免统计区一直显示 0 让人误以为没有数据
            if (code !== 200) {
                ElMessage.error('商城统计获取失败：' + (msg || '未知错误'))
                return
            }
            state.stats = {
                goods: data.goods || state.stats.goods,
                order: data.order || state.stats.order,
                integral: data.integral || state.stats.integral
            }
        } catch (e) {
            ElMessage.error('商城统计请求异常：' + (e?.message || e))
        }
    },
    // 兑换时间窗口展示文案
    timeWindowText(row) {
        const start = Number(row.start_time) || 0
        const end = Number(row.end_time) || 0
        if (!start && !end) return ''
        const fmt = (ts) => utils.time.to.date(ts, 'Y-m-d H:i')
        if (start && end) return `${fmt(start)} ~ ${fmt(end)}`
        if (start) return `${fmt(start)} 起`
        return `${fmt(end)} 止`
    },
    add() {
        state.form = {
            title: '', description: '', cover: '', price: 0, stock: 0, status: 1,
            type: 'virtual', deliver_type: 'text', deliver_content: '', cards_text: '',
            category: '', limit_per_user: 0, min_exp: 0, sort: 0, start_time: null, end_time: null
        }
        state.dialog = true
    },
    edit(row) {
        state.form = {
            ...row,
            // 时间戳转字符串，供 el-date-picker（value-format="X"）使用
            start_time: row.start_time ? String(row.start_time) : null,
            end_time: row.end_time ? String(row.end_time) : null
        }
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
            const payload = {
                ...state.form,
                start_time: Number(state.form.start_time) || 0,
                end_time: Number(state.form.end_time) || 0,
                limit_per_user: Number(state.form.limit_per_user) || 0,
                min_exp: Number(state.form.min_exp) || 0,
                sort: Number(state.form.sort) || 0
            }
            const { code, msg } = await axios.post('/api/goods/save', payload)
            if (code !== 200) return ElMessage.error(msg)
            ElMessage.success('保存成功')
            state.dialog = false
            await Promise.all([method.loadGoods(), method.loadStats()])
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
    async loadOrders(page = state.orderPage) {
        state.orderPage = page
        state.orderLoading = true
        try {
            const params = {
                page: state.orderPage,
                limit: state.pageSize,
                order: 'create_time desc'
            }
            if (state.orderFilter.status !== null && state.orderFilter.status !== '') {
                params.status = state.orderFilter.status
            }
            const { code, msg, data } = await axios.get('/api/goods/orders-all', params)
            if (code !== 200) {
                ElMessage.error('订单加载失败：' + (msg || '未知错误'))
                state.orderList = []
                state.orderTotal = 0
                return
            }
            state.orderList = data.data || []
            state.orderTotal = data.count || 0
        } catch (e) {
            ElMessage.error('订单加载异常：' + (e?.message || e))
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
    // 管理员取消订单：退还用户积分并回滚库存（后端事务处理）
    async cancelOrder(row) {
        try {
            await ElMessageBox.confirm(
                `确定取消订单 ${row.order_no || row.id} 吗？将退还用户 ${row.price} 积分并回滚商品库存。`,
                '取消订单',
                { type: 'warning', confirmButtonText: '确定取消', cancelButtonText: '再想想' }
            )
        } catch {
            return
        }
        const { code, msg } = await axios.put('/api/goods/order-status', { id: row.id, status: 3 })
        if (code !== 200) return ElMessage.error(msg)
        ElMessage.success('订单已取消，积分已退还')
        await Promise.all([method.loadOrders(), method.loadGoods(), method.loadStats()])
    },
    orderStatusOptions: () => ([
        { value: 0, label: '待发货' },
        { value: 1, label: '已发货' },
        { value: 2, label: '已完成' },
        { value: 3, label: '已取消' }
    ]),
    orderStatusText: (s) => {
        const map = { 0: '待发货', 1: '已发货', 2: '已完成', 3: '已取消' }
        return map[s] ?? '未知'
    },
    orderStatusTag: (s) => {
        const map = { 0: 'warning', 1: 'primary', 2: 'success', 3: 'info' }
        return map[s] ?? 'info'
    },
    async refreshAll() {
        await Promise.all([method.loadGoods(), method.loadOrders(), method.loadStats()])
        ElMessage.success('已刷新')
    }
}

onMounted(async () => {
    await method.loadGoods()
    await method.loadOrders()
    await method.loadStats()
})
</script>

<style scoped>
.pagination {
    margin-top: 12px;
    display: flex;
    justify-content: flex-end;
}

/* 商城运营统计 */
.stats-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-bottom: 14px;
}
.stat-cell {
    flex: 1;
    min-width: 120px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px 12px;
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
}
.stat-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
.stat-value {
    font-size: 18px;
    font-weight: 700;
    line-height: 1.3;
}
.stat-value.warn {
    color: var(--el-color-danger);
}
.stat-value.gold {
    color: #d4a148;
}

/* 商品筛选 */
.filter-bar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 12px;
}
</style>
