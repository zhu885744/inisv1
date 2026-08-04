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
            <el-tooltip :content="scope.name" :disabled="utils.is.empty(scope.name)" placement="top">
                <div style="display: flex; align-items: center">
                    <span class="limit-1-line">{{ scope?.name || '-' }}</span>
                </div>
            </el-tooltip>
        </template>

        <template #i-description="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.description)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.description)"></span>
                </template>
                <span class="limit-1-line">{{ scope?.description || '-' }}</span>
            </el-tooltip>
        </template>

        <template #i-remark="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.remark)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.remark)"></span>
                </template>
                <span class="limit-1-line">{{ scope?.remark || '-' }}</span>
            </el-tooltip>
        </template>

    </i-table>

    <el-dialog v-model="state.item.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 等 级</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="名称">
                    <el-input v-model="state.struct.name" placeholder="请输入等级名称"></el-input>
                </el-form-item>
                <el-form-item label="等级">
                    <el-input-number v-model="state.struct.value" :min="0"></el-input-number>
                </el-form-item>
                <el-form-item label="经验值">
                    <el-input-number v-model="state.struct.exp" :min="0"></el-input-number>
                </el-form-item>
                <el-form-item label="描述">
                    <el-input v-model="state.struct.description" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea" placeholder="请输入等级描述信息"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 4, maxRows: 10 }" placeholder="备注一下，避免忘记！" type="textarea"></el-input>
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
import ILevel from "{src}/comps/custom/i-level.vue";

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
        table: 'level',
        dialog: false,
        upload: false,
        wait: false,
    },
    struct: {},
    opts: {
        url: '/api/level/all',
        params: props.params,
        columns: [
            { prop: 'name'  , label: '名称', width: 100, slot: true, fixed: left },
            { prop: 'value', label: '等级', width: 80, align: 'center' },
            { prop: 'exp', label: '经验值', width: 80, align: 'center' },
            { prop: 'description', label: '描述', slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
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

        params = JSON.parse(JSON.stringify(params))

        if (utils.is.empty(params)) return ElMessage.warning('你在想什么？什么都不填！')
        if (utils.is.empty(params?.name)) return ElMessage.warning('等级名称不能为空！')
        if (utils.is.empty(params?.value)) return ElMessage.warning('等级不能为空！')

        state.item.wait     = true

        const { code, msg } = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait     = false

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success('保存成功')  // 新增成功提示
        // 关闭对话框
        state.item.dialog = false
        // 重新加载数据
        await method.init()
    },
    // 编辑数据
    edit: struct => {
        state.struct      = struct
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
        
        ElMessage.success(isSoft ? '删除成功' : '彻底删除成功')  // 根据删除类型显示不同提示
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

        ElMessage.success('恢复成功')  // 恢复成功提示
        // 刷新全部数据
        emit('refresh', 'all')

        // 重新加载数据
        await method.init()
    },
    // 图片大小
    imageSize(url = '', size = '50x50') {
        // 判断 url 是否为空
        if (utils.is.empty(url)) return url
        // 返回新的 url
        return url.includes('?') ? `${url}&size=${size}` : `${url}?size=${size}`
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