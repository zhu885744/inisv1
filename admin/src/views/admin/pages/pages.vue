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
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="标题 | 内容 | 备注" />
                </div>
                <el-button v-on:click="method.refresh()">刷新</el-button>
                <el-button v-on:click="method.add()" v-if="!utils.in.array(state.item.tabs, ['remove','setting'])">添加</el-button>
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
                        <table-pages :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-pages>
                    </el-tab-pane>

                    <el-tab-pane name="check">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">审核通过</span>
                        </template>
                        <table-pages :params="state.params.check" v-model:init="state.tabs.check" v-on:refresh="method.refresh" ref="check"></table-pages>
                    </el-tab-pane>

                    <el-tab-pane name="audit">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">等待审核</span>
                        </template>
                        <table-pages :params="state.params.audit" v-model:init="state.tabs.audit" v-on:refresh="method.refresh" ref="audit"></table-pages>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <table-pages :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-pages>
                    </el-tab-pane>

                    <el-tab-pane name="setting">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">设置</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-page ref="page"></atom-page>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                </el-tabs>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import { push } from '{src}/utils/route'
import TablePages from '{src}/comps/admin/table/pages.vue'
import AtomPage from '{src}/comps/admin/atom/page.vue'

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    user: cache.get('user-info'),
    item: {
        timer : null,
        title : '页面管理',
        search: null,
        sort  : '排序',
        tabs  : 'all',
    },
    params: {
        all: {
            order: 'id asc',
            where: []
        },
        check: {
            order: 'top desc, id desc',
            where: [['audit','=',1]]
        },
        audit: {
            order: 'top desc, id desc',
            where: [['audit','=',0]]
        },
        remove: {
            order: 'id asc',
            where: [],
            onlyTrashed: true
        },
    },
    tabs: {
        all: false,
        remove: false,
    }
})

// 方法
const method = {
    // 设置排序方式
    order(order = 'create_time asc', sort = '排序') {
        state.item.sort = sort
        for (let item in state.params) state.params[item].order = order
        // 指定刷新
        method.refresh('all','check','audit','remove')
    },
    // 添加
    add: () => push({ name: 'admin-pages-write' }),
    // 刷新
    refresh(...args) {
        // 允许刷新的参数
        let allow = ['all','check','audit','remove','page']
        // 如果没有传参则刷新所有
        if (args.length === 0) args = allow
        // 如果传参则过滤不允许的参数
        else args = args.filter(item => allow.includes(item))
        // 批量刷新
        for (let item of args) {
            if (proxy.$refs[item]) proxy.$refs[item]['init']()
        }
    },
    // 切换 tab
    change: (name) => state.tabs[name] = true
}

onMounted(async () => {

    const allow = ['all', 'check', 'audit', 'remove']

    let root = state.user?.result?.auth?.all ?? false
    if (!root) {
        for (let item of allow) state.params[item].where.push(['uid', '=', state.user?.id])
    }

    // 使用 nextTick 确保 DOM 更新后再初始化
    await nextTick()
    state.tabs.all = true
})

watch(() => state.item.search, (val) => {

    const allow = ['all', 'check', 'audit', 'remove']

    for (let item of allow) {
        if (!utils.is.empty(val)) state.params[item].like = [
            ['title'  , `%${val}%`],
            ['remark' , `%${val}%`],
            ['content', `%${val}%`],
        ]
        else delete state.params[item].like
    }

    // 防抖 - 没变化的 500ms 后再刷新
    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => method.refresh(...allow), globalThis.inis?.lazy_time ?? 500)
})
</script>