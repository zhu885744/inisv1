<template>
    <div class="container-box">
        <el-row :gutter="20" style="display: flex;">
            <el-col :span="12" style="display: flex;">
                <el-dropdown v-if="!state.item.tabs.includes('setting')" style="margin-right: 8px" trigger="click">
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
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="搜索用户" />
                </div>
                <el-button v-on:click="method.refresh()">刷新</el-button>
                <el-button v-on:click="method.addExp()" type="primary">调整经验值</el-button>
            </el-col>
            <el-col :span="12" style="display: flex; justify-content: flex-end; z-index: -1">
                <el-button disabled>
                    {{ state.item.title }}
                </el-button>
            </el-col>
        </el-row>

        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-card>
                    <template #header>
                        <div style="display: flex; align-items: center; gap: 8px">
                            <i-svg name="exp" size="18px" style="color: var(--el-color-primary)"></i-svg>
                            <span style="font-weight: 600">用户经验管理</span>
                        </div>
                    </template>
                    <el-table :data="state.userExpList" border style="width: 100%;" :loading="state.tableLoading">
                        <el-table-column prop="id" label="用户ID" width="80" align="center"></el-table-column>
                        <el-table-column prop="nickname" label="用户昵称" min-width="120"></el-table-column>
                        <el-table-column prop="exp" label="经验值" width="120" align="center">
                            <template #default="scope">
                                <span style="font-weight: 600; color: var(--el-color-primary);">{{ scope.row.exp }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column prop="create_time" label="注册时间" width="180" align="center">
                            <template #default="scope">
                                <span>{{ utils.time.to.date(scope.row.create_time, 'Y-m-d H:i:s') }}</span>
                            </template>
                        </el-table-column>
                        <el-table-column label="操作" width="150" align="center">
                            <template #default="scope">
                                <el-button size="small" v-on:click="method.addExp(scope.row)">调整经验</el-button>
                            </template>
                        </el-table-column>
                    </el-table>
                    <el-pagination
                        v-if="state.userExpTotal > 0"
                        class="pagination"
                        background
                        layout="total, prev, pager, next, jumper"
                        :total="state.userExpTotal"
                        :page-size="state.userExpPageSize"
                        :current-page="state.userExpPage"
                        @current-change="method.loadUserExpList"
                    />
                </el-card>
            </el-col>
        </el-row>

        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-tabs v-model="state.item.tabs" v-on:tab-change="method.change" id="tabs-area">
                    <el-tab-pane name="all">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">经验记录</span>
                        </template>
                        <table-exp :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-exp>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <table-exp :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-exp>
                    </el-tab-pane>
                </el-tabs>
            </el-col>
        </el-row>

        <el-dialog v-model="state.expDialog" class="custom" draggable :close-on-click-modal="false">
            <template #header>
                <strong class="flex-center">调整经验值</strong>
            </template>
            <template #default>
                <el-form label-width="100px" label-position="left">
                    <el-form-item label="选择用户">
                        <el-select v-model="state.expForm.uid" placeholder="请选择用户" filterable @change="method.onUserChange">
                            <el-option v-for="user in state.userList" :key="user.id" :label="user.nickname" :value="user.id"></el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item label="用户昵称">
                        <el-input v-model="state.expForm.nickname" disabled></el-input>
                    </el-form-item>
                    <el-form-item label="经验值">
                        <div style="display: flex; align-items: center; gap: 8px;">
                            <el-button size="small" @click="state.expForm.value = -Math.abs(state.expForm.value)">扣除</el-button>
                            <el-input-number v-model="state.expForm.value" :min="-99999" :max="99999" style="width: 200px"></el-input-number>
                            <el-button size="small" @click="state.expForm.value = Math.abs(state.expForm.value)">增加</el-button>
                        </div>
                        <span style="font-size: 12px; color: var(--el-text-color-secondary);">正数为增加，负数为扣除</span>
                    </el-form-item>
                    <el-form-item label="描述">
                        <el-input v-model="state.expForm.description" placeholder="请输入操作描述"></el-input>
                    </el-form-item>
                </el-form>
            </template>
            <template #footer>
                <el-button v-on:click="state.expDialog = false">取 消</el-button>
                <el-button v-on:click="method.saveExp()" :loading="state.expLoading">保 存</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'
import TableExp from '{src}/comps/admin/table/exp.vue'

const { ctx, proxy } = getCurrentInstance()
const state = reactive({
    item: {
        timer: null,
        title: '经验管理',
        search: null,
        sort: '排序',
        tabs: 'all',
    },
    userExpList: [],
    userExpTotal: 0,
    userExpPage: 1,
    userExpPageSize: 20,
    tableLoading: false,
    userList: [],
    expDialog: false,
    expLoading: false,
    expForm: {
        uid: null,
        nickname: '',
        value: 0,
        description: ''
    },
    params: {
        all: {
            order: 'create_time desc'
        },
        remove: {
            order: 'create_time desc',
            onlyTrashed: true
        },
    },
    tabs: {
        all: false,
        remove: false,
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
        for (let item of args) proxy.$refs[item]?.init?.()
        method.loadUserExpList()
    },
    change: (name) => state.tabs[name] = true,
    loadUserExpList: async (page = state.userExpPage) => {
        state.userExpPage = page
        state.tableLoading = true
        
        const params = {
            page: state.userExpPage,
            limit: state.userExpPageSize,
            order: 'create_time desc'
        }
        
        if (!utils.is.empty(state.item.search)) {
            params.like = [['nickname', `%${state.item.search}%`]]
        }
        
        const userResult = await axios.get('/api/users/all', params)
        
        if (userResult.code === 200) {
            const users = userResult.data.data || []
            
            // 使用 Promise.all 并行请求经验值总和，解决 N+1 查询问题
            const expPromises = users.map(user => 
                axios.get('/api/exp/sum', { where: { uid: user.id }, field: 'value' })
            )
            
            const expResults = await Promise.all(expPromises)
            
            users.forEach((user, index) => {
                const expResult = expResults[index]
                user.exp = expResult.code === 200 ? (expResult.data?.value || 0) : 0
            })
            
            state.userExpList = users
            state.userExpTotal = userResult.data.count || 0
        }
        
        state.tableLoading = false
    },
    loadUsers: async () => {
        const { code, data } = await axios.get('/api/users/all', { limit: 100 })
        if (code === 200) {
            state.userList = data.data || []
        }
    },
    addExp: (user = null) => {
        state.expDialog = true
        if (user) {
            state.expForm.uid = user.id
            state.expForm.nickname = user.nickname
            state.expForm.value = 0
            state.expForm.description = ''
        } else {
            state.expForm = {
                uid: null,
                nickname: '',
                value: 0,
                description: ''
            }
        }
        method.loadUsers()
    },
    onUserChange: (uid) => {
        const user = state.userList.find(item => item.id === uid)
        state.expForm.nickname = user?.nickname || ''
    },
    saveExp: async () => {
        if (!state.expForm.uid) return ElMessage.warning('请选择用户')
        if (state.expForm.value === 0) return ElMessage.warning('请输入经验值')
        
        state.expLoading = true
        const { code, msg } = await axios.post('/api/exp/give', {
            uid: state.expForm.uid,
            value: state.expForm.value,
            description: state.expForm.description || (state.expForm.value > 0 ? `管理员发放经验值 ${state.expForm.value}` : `管理员扣除经验值 ${Math.abs(state.expForm.value)}`)
        })
        state.expLoading = false
        
        if (code === 200) {
            ElMessage.success(msg)
            state.expDialog = false
            await method.refresh()
        } else {
            ElMessage.error(msg)
        }
    },
}

onMounted(async () => {
    state.tabs.all = true
    await method.loadUserExpList()
})

watch(() => state.item.search, (val) => {
    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => {
        method.loadUserExpList(1)
    }, globalThis.inis?.lazy_time ?? 500)
})
</script>