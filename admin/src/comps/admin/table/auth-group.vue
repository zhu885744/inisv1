<template>
    <i-table :opts="state.opts" ref="i-table">

        <template #start>
            <el-table-column type="selection" width="55"></el-table-column>
        </template>

        <template v-if="props.type === 'all'" #end>
            <el-table-column :fixed="right" label="操作" width="100" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.edit(scope.row)" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, true)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>
        <template v-if="props.type === 'remove'">
            <el-table-column :fixed="right" label="操作" width="160" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.restore(scope.row.id)" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.edit(scope.row)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, false)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>

        <template #i-name="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-tooltip :content="scope.name" :disabled="utils.is.empty(scope.name)" placement="top">
                    <span style="display: flex; align-items: center">
                        <i-svg name="dot" :color="scope.root === 1 ? 'var(--bs-success)' : 'var(--bs-secondary)'" size="20px"></i-svg>
                        <span class="limit-1-line">{{ scope?.name }}</span>
                    </span>
                </el-tooltip>
            </span>
        </template>

        <template #i-users="{ scope = {} }">
            <div style="display: flex; align-items: center">
                <el-tooltip v-for="(item, index) in scope.result?.users?.slice(0, 5)" :key="item.id" :content="`@${item.nickname} ${item.account}`" placement="top">
                    <el-avatar :src="item.avatar" size="small" :class="'avatar-shadow z-index-' + (10 - index)" :style="'margin-left: -8px'"></el-avatar>
                </el-tooltip>
                <el-tooltip v-if="scope.result?.users?.length > 12" :content="`有 ${scope.result?.users?.length} 人拥有该权限`" placement="top">
                    <span :style="'z-index: 0; background: var(--el-color-primary-light-3); display: flex; align-items: center; justify-content: center; color: white; margin-left: -8px; width: 24px; height: 24px; border-radius: 200px'" class="avatar-shadow">+</span>
                </el-tooltip>
            </div>
        </template>

        <template #i-remark="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.remark)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.remark)"></span>
                </template>
                <span class="limit-1-line">{{ scope?.remark || '无' }}</span>
            </el-tooltip>
        </template>

        <template #i-key="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.key)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.key)"></span>
                </template>
                <span class="limit-1-line">{{ scope?.key || '-' }}</span>
            </el-tooltip>
        </template>

    </i-table>

    <el-dialog v-model="state.item.dialog" class="custom" :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 权 限 分 组</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="名称">
                    <el-input v-model="state.struct.name" placeholder="请输入分组名称" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="唯一识别码">
                    <el-input v-model="state.struct.key" placeholder="请输入唯一识别码" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="权限">
                    <el-select v-model="state.struct.root" placeholder="请选择权限" style="width: 100%" class="custom">
                        <el-option v-for="item in state.select.root" :key="item.value" :label="item.label" :value="item.value" style="display: flex; justify-content: space-between">
                            <span style="font-size: 13px; display: flex; align-items: center">
                                <i-svg name="dot" :color="item.color" size="20px"></i-svg>
                                {{ item.label }}
                            </span>
                            <small>{{ item.subtitle }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 3, maxRows: 10 }" placeholder="备注一下，避免忘记！" type="textarea" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="成员">
                    <el-select v-model="state.selected.users" multiple filterable default-first-option placeholder="请选择成员" style="width: 100%" class="custom multiple">
                        <el-option v-for="item in state.select.users" :key="item.id" :label="item.nickname" :value="item.id">
                            <span style="display: flex; justify-content: space-between">
                                <span style="display: flex; align-items: center">
                                    <el-avatar :src="item.avatar" size="small" class="avatar-shadow"></el-avatar>
                                    <span style="font-size: 14px; margin-left: 4px">{{ item.nickname }}</span>
                                </span>
                                <small>{{ item.account }}</small>
                            </span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="页面">
                    <el-select v-model="state.selected.pages" multiple filterable default-first-option placeholder="请选择权限" style="width: 100%" class="custom multiple">
                        <el-option v-for="item in state.select.pages" :key="item.hash" :label="item.name" :value="item.hash">
                            <span style="font-size: 13px">
                                <span v-if="!utils.is.empty(item.svg)" v-html="item.svg"></span>
                                <i-svg color="rgb(var(--icon-color))" v-else-if="!utils.is.empty(item.icon)" :name="item.icon" :size="item.size"></i-svg>
                                {{ item.name }}
                            </span>
                            <small style="float: right">{{ item.path }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="规则">
                    <el-cascader placeholder="试试搜索：文章" :options="state.select.rules" :props="{ multiple: true }" filterable
                        class="custom multiple" style="display: block; width: 100%" v-model="state.rules.select" v-on:change="method.change">
                        <template #default="{ node, data }">
                            <span>{{ data.label }} </span>
                            <span v-if="!node.isLeaf"> ({{ data.children.length }}) </span>
                        </template>
                    </el-cascader>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.item.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.item.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'
import { useUsersStore } from '{src}/store/users'
import { useAuthRulesStore } from '{src}/store/auth-rules'
import { useAuthPagesStore } from '{src}/store/auth-pages'

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    params: {
        type: Object,
        default: () => ({
            order: 'id asc',
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

// table - fixed
const left  = computed(() => {
    let result = 'left'
    if (utils.is.mobile()) result = false
    return result
})
// table - fixed
const right = computed(() => {
    let result = 'right'
    if (utils.is.mobile()) result = false
    return result
})

const { proxy } = getCurrentInstance()
const store  = {
    users: useUsersStore(),
    authRules: useAuthRulesStore(),
    authPages: useAuthPagesStore(),
}
const state  = reactive({
    item: {
        table: 'auth-group',
        dialog: false,
        wait: false,
    },
    struct: { root: 0 },
    opts: {
        url: '/api/auth-group/all',
        params: props.params,
        columns: [
            { prop: 'name', label: '名称', slot: true, fixed: left },
            { prop: 'key', label: '识别码', width: 80, slot: true, align: 'center' },
            { prop: 'users', label: '成员', slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    rules: {
        // 选中的值
        select: [],
        // 规则原始列表
        column: []
    },
    // 下拉框
    select: {
        root: [
            { value: 0, label: '默认', subtitle: '只允许操作自己的数据', color: 'var(--bs-secondary)' },
            { value: 1, label: '管理', subtitle: '（穿透）允许操作他人的数据', color: 'var(--bs-success)' }
        ],
        pages: store.authPages.getFlat,
        rules: store.authRules.getTree,
    },
    selected: {
        users: [],
        pages: [],
    },
})

const method = {
    // 自动换行
    autoWrap(text = '', length = 40, symbol = '<br>') {
        // 判断 text 是否为空
        if (utils.is.empty(text)) return text
        // 每隔 length 个字符添加一个换行符
        return text.replace(new RegExp(`(.{${length}})`, 'g'), `$1${symbol}`)
    },
    // 规则树变化
    change() {
        let array = []
        // 强转数组
        for (let item of [...state.rules.select]) array.push(item[1])
        // 去重去空
        array = [...new Set(array)].filter(item => item).join(',')
        state.struct.rules = array
    },
    // 计算用户数量
    count: (value = null) => {
        if (utils.is.empty(value)) return 0
        let array = value.split('|')
        // 去空去重
        array = [...new Set(array.filter(item => !utils.is.empty(item)))]
        return array.length
    },
    // 删除数据
    async delete(ids = [], isSoft = true) {

        if (utils.is.empty(ids)) return

        // 拼接服务地址
        const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`

        const {code, msg} = await axios.del(uri, {ids})

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        // 刷新回收站数据
        emit('refresh', 'remove')

        // 重新加载数据
        await method.init()
    },
    // 编辑数据
    edit: async struct => {

        await method.users()

        // 获取规则树
        const { data } = await store.authRules.setTree()
        state.select.rules = data

        state.struct = struct

        if (struct.pages?.indexOf('all') !== -1) {
            state.selected.pages = state.select.pages?.map(item => item?.hash)
        } else {
            // 去空
            state.selected.pages = struct.pages?.split(',').filter(item => item)
        }
        // 字符串转数组 - 去空 - 去重
        state.selected.users = struct.result?.users.map(item => parseInt(item.id)).filter(item => item)

        // 判断 struct.rules 是否包含 all - 全部权限
        if (struct.rules.includes('all')) {
            let rules = []
            for (let item of state.select.rules) {
                for (let child of item?.children) {
                    rules.push([item?.label, child.value])
                }
            }
            state.rules.select = rules
            return state.item.dialog = true
        }

        // 只有部分权限
        let ids = []
        // 字符串转数组
        if (struct.rules) ids = struct.rules.split(',').map(item => parseInt(item))
        // 从原始数据中获取数据
        if (store.authRules.getFlat) {
            let rules = []
            const regex1 = /^[【|\[](.+?)[】|\]](.+)/
            // 匹配特殊字符串隔开的
            const regex2 = /(.+)[^\w\u4e00-\u9fa5\s](.+)/
            for (const item of store.authRules.getFlat) {
                if (ids.includes(parseInt(item.hash))) {
                    let match1 = item.name.match(regex1)
                    let match2 = item.name.match(regex2)
                    let name = ''
                    if (match1) name = match1[1].trim()
                    else if (match2) name = match2[1].trim()
                    rules.push([name, parseInt(item.hash)])
                }
            }
            state.rules.select = rules
        }

        state.item.dialog = true
    },
    // 初始化数据
    init: async () => {
        state.selected.users = []
        state.selected.pages = []
        // 重新加载数据
        await proxy.$refs['i-table']['init']()
    },
    // 省略文字
    omit: (text = null, length = 10, omission = ' ... ', location = 'center') => {
        if (utils.is.empty(text)) return '空'
        return utils.string.omit(text, length, omission, location)
    },
    // 恢复数据
    async restore(ids = []) {

        if (utils.is.empty(ids)) return

        const {code, msg} = await axios.put(`/api/${state.item.table}/restore`, {ids})

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        // 刷新全部数据
        emit('refresh', 'all')

        // 重新加载数据
        await method.init()
    },
    // 保存数据
    save: async (params = state.struct || {}) => {

        if (utils.is.empty(params)) return ElMessage.warning('你在想什么？什么都不填！')  // 使用Element Plus的Message
        if (utils.is.empty(params?.name)) return ElMessage.warning('分组名称是必须的哟！')  // 使用Element Plus的Message

        // 判断是否拥有全部的权限
        let ids = store.authRules.getFlat.map(item=>parseInt(item.hash))
        // 去重去空
        ids = [...new Set(ids)].filter(item => item)
        // 重新排序
        ids.sort((a, b) => a - b)
        // 选中的权限
        let select = state.rules.select.map(item => item[1] || null)
        // 去重去空
        select = [...new Set(select)].filter(item => item)
        // 重新排序
        select.sort((a, b) => a - b)
        // 判断是否拥有全部的权限
        if (utils.array.equal(ids, select)) params.rules = 'all'
        else if (select) params.rules = select.join(',')
        else params.rules = null

        // 页面权限
        let arr1 = state.select.pages?.map(item => parseInt(item.hash)).filter(item => item)
        let arr2 = state.selected.pages.map(item => parseInt(item)).filter(item => item)

        if (utils.array.equal(arr1, arr2)) params.pages = 'all'
        else if (utils.is.empty(arr2)) params.pages = ''
        else params.pages = arr2.join(',')

        if (utils.is.empty(state.selected.users)) params.uids = ''
        else params.uids = `|${state.selected.users.join('|')}|`

        state.item.wait = true

        const {code, msg} = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait = false

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        ElMessage.success('操作成功')  // 使用Element Plus的Message
        // 关闭模态框
        state.item.dialog = false
        // 重新加载数据
        await method.init()
    },
    // 显示盒子
    async show() {
        await method.users()
        state.struct = {}
        state.rules.select   = []
        state.selected.users = []
        state.selected.pages = []
        state.item.dialog    = true
    },
    // 获取用户列表
    users: async () => {

        const fn = data => {
            return data.map(item => {
                const { id, nickname, account, avatar } = item
                return { id, nickname, account, avatar }
            })
        }

        if (!utils.is.empty(store.users.column)) {
            state.select.users = fn(store.users.column)
            return
        }

        const { code, data } = await axios.get('/api/users/all?limit=99999&field=id,nickname,account,avatar')

        if (code !== 200) return

        store.users.column = data.data
        state.select.users = fn(data.data)
    }
}

onMounted(() => {
    if (props.init) method.init()
    method.users()
})

watch(() => props.init, (val) => {
    if (val) method.init()
})

// 监听对话框状态
watch(() => state.item.dialog, (value) => {
    // 关闭对话框时清空数据
    if (!value) state.struct = {}
})

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
    show: method.show,
})
</script>