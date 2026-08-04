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
        <template v-if="props.type === 'remove'" #end>
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
                <el-tooltip v-if="parseInt(scope.common) === 1" content="公共权限，不需要登录即可使用的接口" placement="top">
                    <i-svg color="rgb(var(--icon-color))" name="!" size="14px"></i-svg>
                </el-tooltip>
                <el-tooltip :content="scope.name" :disabled="utils.is.empty(scope.name)" placement="top">
                    <span>{{ method.omit(scope?.name, 16, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-route="{ scope = {} }">
            <el-tooltip placement="top">
                <template #content>
                    <span v-if="scope.type === 'login'">登录类型</span>
                    <span v-else-if="scope.type === 'common'">公共类型</span>
                    <span v-else>默认类型</span>
                </template>
                <span style="margin-right: 4px">
                            <i-svg color="rgb(var(--icon-color))" v-if="scope.type === 'login'" name="user" size="18px"></i-svg>
                            <i-svg color="rgb(var(--icon-color))" v-else-if="scope.type === 'common'" name="common" size="18px"></i-svg>
                            <i-svg color="rgb(var(--icon-color))" v-else name="!" size="16px"></i-svg>
                        </span>
            </el-tooltip>
            <el-tooltip :content="'双击复制：' + scope.route" :disabled="utils.is.empty(scope.route)" placement="top">
                <span v-on:dblclick="method.copy(scope.route, '复制成功！')">
                    <span :style="'color: ' + (method.color(scope.method))">[{{ scope?.method }}]</span>
                    <span style="margin-left: 4px">{{ method.omit(scope?.route, 30, ' ...', 'end') }}</span>
                </span>
            </el-tooltip>
        </template>

        <template #i-remark="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.remark)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.remark)"></span>
                </template>
                <span>{{ method.omit(scope?.remark) }}</span>
            </el-tooltip>
        </template>

    </i-table>

    <el-dialog v-model="state.item.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 权 限 规 则</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="名称">
                    <el-input v-model="state.struct.name" placeholder="请输入接口名称，如：【分组名】API名" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="费用">
                    <el-input-number v-model="state.struct.cost" :min="0" style="width: 100%"></el-input-number>
                </el-form-item>
                <el-form-item label="请求类型">
                    <el-select v-model="state.struct.method" placeholder="请选择请求类型" style="width: 100%" class="custom">
                        <el-option v-for="item in state.select.method" :key="item.value" :label="item.value" :value="item.label">
                            <span style="font-size: 13px" :style="'color: ' + method.color(item.value)">{{ item.value }}</span>
                            <small style="float: right">{{ item.label }} 请求</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="API">
                    <el-input v-model="state.struct.route" placeholder="请输入接口请求地址" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="接口类型">
                    <el-select v-model="state.struct.type" placeholder="请选择接口类型" style="width: 100%" class="custom">
                        <el-option v-for="item in state.select.type" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                            <small style="float: right">{{ item.value }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 3, maxRows: 10 }" placeholder="备注一下，避免忘记！" type="textarea" style="width: 100%"></el-input>
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

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    params: {
        type: Object,
        default: () => ({
            order: 'hash asc',
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

// table - fixed
const left = computed(() => {
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

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        table: 'auth-rules',
        dialog: false,
        wait: false,
    },
    struct: {},
    opts: {
        url: '/api/auth-rules/all',
        params: props.params,
        columns: [
            { prop: 'name', label: '名称', slot: true, fixed: left },
            { prop: 'route', label: 'API', slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    // 下拉框
    select: {
        method: [
            { label: 'GET', value: 'GET' },
            { label: 'PUT', value: 'PUT' },
            { label: 'POST', value: 'POST' },
            { label: 'DELETE', value: 'DELETE' },
        ],
        type: [{ value: 'default', label: '默认' },{ value: 'common', label: '公共' }, { value: 'login', label: '登录' }],
    },
})

const method = {
    // 刷新数据
    init: async () => {
        // 重新加载数据
        await proxy.$refs['i-table']['init']()
    },
    // 保存数据
    save: async (params = state.struct || {}) => {

        if (utils.is.empty(params)) return ElMessage.warning('你在想什么？什么都不填！')
        if (utils.is.empty(params.route)) return ElMessage.warning('API是必填项！')

        state.item.wait     = true

        const { code, msg } = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait     = false

        if (code !== 200) return ElMessage.error(msg)

        // 关闭对话框
        state.item.dialog = false
        // 重新加载数据
        await method.init()
        ElMessage.success('保存成功')
    },
    // 编辑数据
    edit: struct => {
        state.struct = struct
        state.item.dialog = true
    },
    // 显示盒子
    show: () => (state.item.dialog = true),
     // 真删 和 软删
    async delete(ids = [], isSoft = true) {

        if (utils.is.empty(ids)) return

        // 拼接服务地址
        const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`

        const { code, msg } = await axios.del(uri, { ids })

        if (code !== 200) return ElMessage.error(msg)

        // 刷新回收站数据
        emit('refresh', 'remove')

        // 重新加载数据
        await method.init()
        ElMessage.success('删除成功')
    },
    // 恢复数据
    async restore(ids = []) {

        if (utils.is.empty(ids)) return

        const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })

        if (code !== 200) return ElMessage.error(msg)

        // 刷新全部数据
        emit('refresh', 'all')

        // 重新加载数据
        await method.init()
        ElMessage.success('恢复成功')
    },
    // 自动换行
    autoWrap(text = '', length = 40, symbol = '<br>') {
        // 判断 text 是否为空
        if (utils.is.empty(text)) return text
        // 每隔 length 个字符添加一个换行符
        return text.replace(new RegExp(`(.{${length}})`, 'g'), `$1${symbol}`)
    },
    // 复制文本
    copy  : (text = null, msg = '复制成功！') => {

        if (utils.is.empty(text)) return

        utils.set.copy.text(text)

        if (!utils.is.empty(msg)) return ElMessage.info(msg)
    },
    // 省略文本
    omit  : (text = null, length = 10, omission = ' ... ', location = 'center') => {
        if (utils.is.empty(text)) return '空'
        return utils.string.omit(text, length, omission, location)
    },
    // 分配颜色
    color : (value = 'GET') => {
        // 强转大写
        value = value.toUpperCase()
        let opts = {'GET':'success', 'POST':'warning', 'PUT':'info', 'DELETE':'danger'}
        return opts[value] || 'dark'
    },
}

onMounted(async () => {
    if (props.init) await method.init()
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