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
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="擅用搜索，事半功倍！" />
                </div>
                <el-button v-on:click="method.refresh()">刷新</el-button>
                <el-button v-on:click="method.add()" v-if="state.item.tabs.includes('all')">添加</el-button>
            </el-col>
            <el-col :span="12" style="display: flex; justify-content: flex-end; z-index: -1">
                <el-button disabled>
                    {{ state.item.title }}
                </el-button>
            </el-col>
        </el-row>
        <el-row :gutter="20" style="margin-top: 12px">
            <el-col :span="24">
                <el-tabs v-model="state.item.tabs" v-on:tab-change="method.change" id="tabs-area">

                    <el-tab-pane name="all">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">全部</span>
                        </template>
                        <table-users :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-users>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <table-users :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-users>
                    </el-tab-pane>

                    <el-tab-pane name="blackroom">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">封禁记录</span>
                        </template>
                        <table-blackroom v-model:init="state.tabs.blackroom" v-on:refresh="method.refresh" ref="blackroom"></table-blackroom>
                    </el-tab-pane>

                </el-tabs>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import TableUsers from '{src}/comps/admin/table/users.vue'
import TableBlackroom from '{src}/comps/admin/table/blackroom.vue'

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        timer : null,
        title : '用户管理',
        search: null,
        sort  : '排序',
        tabs  : 'all',
    },
    params: {
        all: {
            order: 'login_time desc, id desc'
        },
        remove: {
            order: 'login_time desc, id desc',
            onlyTrashed: true
        },
    },
    tabs: {
        all: false,
        remove: false,
        blackroom: false,
    }
})

// 方法
const method = {
    // 设置排序方式
    order(order = 'create_time asc', sort = '排序') {
        state.item.sort = sort
        for (let item in state.params) state.params[item].order = order
        // 指定刷新
        method.refresh('all','remove')
    },
    // 添加
    add: () => proxy.$refs['all']['show'](),
    // 刷新
    refresh(...args) {
        // 允许刷新的参数
        let allow = ['all','remove','blackroom']
        // 如果没有传参则刷新所有
        if (args.length === 0) args = allow
        // 如果传参则过滤不允许的参数
        else args = args.filter(item => allow.includes(item))
        // 批量刷新
        for (let item of args) proxy.$refs[item]['init']()
    },
    // 切换 tab
    change: (name) => state.tabs[name] = true
}

onMounted(async () => {
    state.tabs.all = true
})

watch(() => state.item.search, (val) => {

    const allow = ['all', 'remove', 'blackroom']

    for (let item of allow) {
        if (!utils.is.empty(val)) state.params[item].like = [
            ['email', `%${val}%`],
            ['phone', `%${val}%`],
            ['remark', `%${val}%`],
            ['account', `%${val}%`],
            ['nickname', `%${val}%`],
            ['description' , `%${val}%`],
        ]
        else delete state.params[item].like
    }

    // 防抖 - 没变化的 500ms 后再刷新
    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => method.refresh(...allow), globalThis.inis?.lazy_time ?? 500)
})
</script>

<style scoped>
@media (max-width: 768px) {
    .container-box :deep(.el-row) {
        flex-direction: column;
    }

    .container-box :deep(.el-col) {
        width: 100% !important;
        justify-content: flex-start !important;
        margin-bottom: 8px;
    }

    .container-box :deep(.el-input) {
        width: 100% !important;
    }

    .container-box :deep(.el-button) {
        font-size: 12px;
        padding: 6px 12px;
        margin-bottom: 4px;
    }

    .container-box :deep(.el-dropdown) {
        margin-right: 0 !important;
    }
}
</style>