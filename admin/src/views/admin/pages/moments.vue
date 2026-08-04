<template>
    <div class="container-box">
        <el-row :gutter="20" style="display: flex">
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
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="内容" />
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
                        <table-moments :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-moments>
                    </el-tab-pane>

                    <el-tab-pane name="check">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">审核通过</span>
                        </template>
                        <table-moments :params="state.params.check" v-model:init="state.tabs.check" v-on:refresh="method.refresh" ref="check"></table-moments>
                    </el-tab-pane>

                    <el-tab-pane name="audit">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">等待审核</span>
                        </template>
                        <table-moments :params="state.params.audit" v-model:init="state.tabs.audit" v-on:refresh="method.refresh" ref="audit"></table-moments>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <table-moments :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-moments>
                    </el-tab-pane>

                    <el-tab-pane name="setting">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">设置</span>
                        </template>
                        <el-row :gutter="20">
                            <el-col :span="8">
                                <atom-moments ref="moments"></atom-moments>
                            </el-col>
                        </el-row>
                    </el-tab-pane>

                </el-tabs>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import AtomMoments from '{src}/comps/admin/atom/moments.vue'
import TableMoments from '{src}/comps/admin/table/moments.vue'
import cache from "{src}/utils/cache.js";

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    user: cache.get('user-info'),
    item: {
        timer : null,
        title : '动态列表',
        search: null,
        sort  : '排序',
        tabs  : 'all',
    },
    params: {
        all: {
            order: 'id desc',
        },
        check: {
            order: 'id desc',
            where: [['audit','=',1],['status','=',1]]
        },
        audit: {
            order: 'id desc',
            where: [['audit','=',0]]
        },
        remove: {
            order: 'id desc',
            onlyTrashed: true
        },
    },
    tabs: {
        all: false,
        check: false,
        audit: false,
        remove: false,
    }
})

const method = {
    order(order = 'create_time asc', sort = '排序') {
        state.item.sort = sort
        for (let item in state.params) state.params[item].order = order
        method.refresh('all', 'check', 'audit', 'remove')
    },
    add: () => proxy.$refs['all'].open(),
    refresh(...args) {
        let allow = ['all', 'check', 'audit' , 'remove', 'moments']
        if (args.length === 0) args = allow
        else args = args.filter(item => allow.includes(item))
        for (let item of args) {
            if (proxy.$refs[item]) proxy.$refs[item]['init']()
        }
    },
    change: (name) => state.tabs[name] = true
}

onMounted(async () => {
    const allow = ['all', 'check', 'audit', 'remove']

    let root = state.user?.result?.auth?.all ?? false
    let userId = state.user?.id
    if (!root && userId) {
        for (let item of allow) {
            if (!state.params[item].where) state.params[item].where = []
            state.params[item].where.push(['uid', '=', userId])
        }
    }

    await nextTick()
    state.tabs.all = true
})

watch(() => state.item.search, (val) => {

    const allow = ['all', 'check', 'audit', 'remove']

    for (let item of allow) {
        if (!utils.is.empty(val)) state.params[item].like = [
            ['content', `%${val}%`],
        ]
        else delete state.params[item].like
    }

    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => method.refresh(...allow), globalThis.inis?.lazy_time ?? 500)
})
</script>
