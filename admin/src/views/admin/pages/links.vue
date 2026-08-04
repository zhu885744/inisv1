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
                    <el-input v-model="state.item.search" style="width: 200px" autocomplete="new-password" type="text" placeholder="昵称 | 网址 | 描述 | 备注" />
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
                        <table-links :params="state.params.all" v-model:init="state.tabs.all" v-on:refresh="method.refresh" ref="all"></table-links>
                    </el-tab-pane>

                    <el-tab-pane name="audit">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">待审核</span>
                        </template>
                        <table-links :params="state.params.audit" v-model:init="state.tabs.audit" v-on:refresh="method.refresh" ref="audit"></table-links>
                    </el-tab-pane>

                    <el-tab-pane name="remove">
                        <template #label>
                            <span style="font-weight: bold; font-size: 12px">回收站</span>
                        </template>
                        <table-links :params="state.params.remove" v-model:init="state.tabs.remove" v-on:refresh="method.refresh" ref="remove" type="remove"></table-links>
                    </el-tab-pane>

                </el-tabs>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'
import TableLinks from '{src}/comps/admin/table/links.vue'

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        timer : null,
        title : '友链管理',
        search: null,
        sort  : '排序',
        tabs  : 'all', // 移除group相关配置
    },
    params: {
        all: {
            order: 'id asc'
            // 移除分组查询条件，默认查询全部
        },
        audit: {
            order: 'id asc',
            where: [['audit', 0]]
            // 查询待审核的友链
        },
        remove: {
            order: 'id asc',
            onlyTrashed: true
            // 移除分组查询条件，默认查询全部回收站数据
        },
    },
    // 移除分组相关的select配置
    tabs: {
        all: false,
        audit: false,
        remove: false,
    }
})

// 方法
const method = {
    // 初始化数据 - 移除分组加载逻辑
    async init() {
        // 无需加载分组数据，直接刷新友链列表
        method.refresh()
    },
    // 移除group方法（分组选择逻辑）
    // 设置排序方式
    order(order = 'create_time asc', sort = '排序') {
        state.item.sort = sort
        for (let item in state.params) state.params[item].order = order
        method.refresh('all','audit','remove')
    },
    // 添加
    add: () => proxy.$refs['all']['show'](),
    // 刷新
    refresh(...args) {
        let allow = ['all','audit','remove']
        if (args.length === 0) args = allow
        else args = args.filter(item => allow.includes(item))
        for (let item of args) proxy.$refs[item]['init']()
    },
    // 图片大小
    imageSize(url = '', size = '50x50') {
        if (utils.is.empty(url)) return url
        return url.includes('?') ? `${url}&size=${size}` : `${url}?size=${size}`
    },
    // 切换 tab
    change: (name) => state.tabs[name] = true
}

onMounted(async () => {
    state.tabs.all = true
    await method.init()
})

watch(() => state.item.search, (val) => {
    const allow = ['all', 'audit', 'remove']

    for (let item of allow) {
        if (!utils.is.empty(val)) state.params[item].like = [
            ['url', `%${val}%`],
            ['remark', `%${val}%`],
            ['nickname', `%${val}%`],
            ['description', `%${val}%`],
        ]
        else delete state.params[item].like
    }

    clearTimeout(state.item.timer)
    state.item.timer = setTimeout(() => method.refresh(...allow), globalThis.inis?.lazy_time ?? 500)
})
</script>