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
                <span v-if="!utils.is.empty(scope.svg)" v-html="scope.svg"></span>
                <i-svg color="rgb(var(--icon-color))" v-else-if="!utils.is.empty(scope.icon)" :name="scope.icon" :size="scope.size"></i-svg>
                <el-tooltip :content="scope.name" :disabled="utils.is.empty(scope.name)" placement="top">
                    <span style="margin-left: 4px">{{ method.omit(scope?.name, 16, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
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
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 管 理 页 面</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="名称">
                    <el-input v-model="state.struct.name" placeholder="请输入页面名称" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="路径">
                    <el-input v-model="state.struct.path" :disabled="!utils.is.empty(state.struct.id)" placeholder="请输入页面路径" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="内置图标">
                    <el-select v-model="state.struct.icon" placeholder="请选择内置图标" style="width: 100%" class="custom">
                        <el-option v-for="item in state.select.icons" :key="item.value" :label="item.value" :value="item.label">
                            <span style="font-size: 13px">
                                <i-svg color="rgb(var(--icon-color))" :name="item.value" size="16px"></i-svg>
                            </span>
                            <small style="float: right">{{ item.label }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="图标大小">
                    <el-input v-model="state.struct.size" placeholder="如：16px" style="width: 200px"></el-input>
                </el-form-item>
                <el-form-item label="自定义图标 - SVG代码">
                    <el-input v-model="state.struct.svg" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea" placeholder="请输入自定义SVG代码" style="width: 100%"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea" placeholder="备注一下" style="width: 100%"></el-input>
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
            order: 'id asc',
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
        table: 'auth-pages',
        dialog: false,
        wait: false,
    },
    struct: {},
    opts: {
        url: '/api/auth-pages/all',
        params: props.params,
        columns: [
            { prop: 'name', label: '名称', slot: true, fixed: left },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    // 下拉框
    select: {
        icons: [{value: '', label: '无'}],
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

        if (utils.is.empty(params))      return ElMessage.warning('你在想什么？什么都不填！')
        if (utils.is.empty(params?.name)) return ElMessage.warning('名称是必填项！')
        if (utils.is.empty(params?.path)) return ElMessage.warning('路径是必填项！')

        state.item.wait     = true

        const { code, msg } = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait     = false

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success('保存成功')  // 添加成功提示
        // 关闭对话框
        state.item.dialog = false
        // 重新加载数据
        await method.init()
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

        ElMessage.success('删除成功')  // 添加成功提示
        // 刷新回收站数据
        emit('refresh', 'remove')

        // 重新加载数据
        await method.init()
    },
    // 恢复数据
    async restore(ids = []) {

        if (utils.is.empty(ids)) return

        const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success('恢复成功')  // 添加成功提示
        // 刷新全部数据
        emit('refresh', 'all')

        // 重新加载数据
        await method.init()
    },
    // 内置 icon 图标库
    async icons() {
        const modulesFiles = import.meta.glob('../../../assets/svg/*.svg')
        let files = []
        for (let path in modulesFiles) files.push(path)
        let icons = files.map(item=>item.split('/').pop().split('.').shift())
        state.select.icons = [...state.select.icons, ...icons.map(item => ({value: item, label: item}))]
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
}

onMounted(async () => {
    await method.icons()
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