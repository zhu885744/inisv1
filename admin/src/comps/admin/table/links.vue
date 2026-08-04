<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0">
            <el-button v-on:click="method.batchAudit()" type="primary" size="small">
                <i-svg color="rgb(var(--icon-color))" name="audit" size="16px"></i-svg>
                <span style="margin-left: 4px">批量审核</span>
            </el-button>
            <el-button v-on:click="method.batchUnAudit()" type="warning" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="audit" size="16px"></i-svg>
                <span style="margin-left: 4px">批量待审核</span>
            </el-button>
        </div>
        <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">

        <template #start>
            <el-table-column type="selection" width="55"></el-table-column>
        </template>

        <template #end>
            <el-table-column :fixed="right" label="操作" :width="props.type === 'remove' ? '160' : '150'" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-if="props.type === 'remove'" v-on:click="method.restore(scope.row.id)" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.edit(scope.row)" size="small" :style="{ marginLeft: props.type === 'remove' ? '0' : '' }">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-if="props.type !== 'remove' && scope.row.audit === 0" v-on:click="method.audit([scope.row.id])" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="audit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, props.type !== 'remove')" size="small" :style="{ marginLeft: props.type === 'remove' ? '0' : '0' }">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>

        <template #i-nickname="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-avatar shape="square" :src="method.imageSize(scope?.avatar)" size="small" style="margin-right: 8px"></el-avatar>
                <el-tooltip :content="scope.nickname" :disabled="utils.is.empty(scope.nickname)" placement="top">
                    <span>{{ method.omit(scope?.nickname, 4, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-url="{ scope = {} }">
            <el-tooltip v-if="!utils.is.empty(scope.url) && scope.target === '_blank'" content="新窗口打开" placement="top">
                <i-svg color="rgb(var(--icon-color))" v-on:click="method.window(scope.url)" name="n" size="16px" style="margin-right: 4px"></i-svg>
            </el-tooltip>
            <el-tooltip :content="scope.url" :disabled="utils.is.empty(scope.url)" placement="top">
                <span>{{ method.omit(scope?.url, 16, '...', 'end') }}</span>
            </el-tooltip>
        </template>

        <template #i-description="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.description)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.description)"></span>
                </template>
                <span>{{ method.omit(scope?.description) }}</span>
            </el-tooltip>
        </template>

        <template #i-audit="{ scope = {} }">
            <el-tag :type="scope.audit === 1 ? 'success' : 'warning'" size="small">
                {{ scope.audit === 1 ? '已审核' : '待审核' }}
            </el-tag>
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
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 友 链</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="昵称">
                    <el-input v-model="state.struct.nickname" placeholder="请输入好友昵称"></el-input>
                </el-form-item>
                <el-form-item label="分组">
                    <el-select v-model="state.struct.group" filterable placeholder="请选择分组" class="custom">
                        <el-option v-for="item in state.select.group" :key="item.value" :label="item.label" :value="item.value">
                            <div style="display: flex; align-items: center">
                                <el-avatar shape="square" :src="method.imageSize(item?.avatar)" size="small" style="margin-right: 8px"></el-avatar>
                                <span style="font-size: 13px">{{ item.label }}</span>
                            </div>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="头像">
                    <el-input v-model="state.struct.avatar" class="custom" placeholder="填写图片地址或点击上传图片">
                        <template #append>
                            <el-button v-on:click="method.upload('avatar')" :loading="state.item.upload">
                                <i-svg v-if="!state.item.upload" name="upload" color="rgb(var(--icon-color))" size="14px"></i-svg>
                                <span style="margin-left: 4px">上传</span>
                            </el-button>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item label="审核状态">
                    <el-select v-model="state.struct.audit" placeholder="请选择审核状态" class="custom">
                        <el-option label="待审核" :value="0">待审核</el-option>
                        <el-option label="已审核" :value="1">已审核</el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="跳转链接">
                    <el-input v-model="state.struct.url" placeholder="请输入跳转链接"></el-input>
                </el-form-item>
                <el-form-item label="跳转方式">
                    <el-select v-model="state.struct.target" placeholder="请选择方式" class="custom">
                        <el-option v-for="item in state.select.target" :key="item.value" :label="item.value" :value="item.label">
                            <span style="font-size: 13px">{{ item.value }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.label }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="描述">
                    <el-input v-model="state.struct.description" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea" placeholder="请输入描述内容"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 3, maxRows: 10 }" placeholder="备注一下，避免忘记！" type="textarea"></el-input>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.item.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.item.wait">保 存</el-button>
        </template>
    </el-dialog>
    </div>
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
        table: 'links',
        dialog: false,
        upload: false,
        wait: false,
        selection: [],
    },
    struct: {},
    opts: {
        url: '/api/links/all',
        params: props.params,
        columns: [
            { prop: 'uid', label: '用户ID', width: 80, align: 'center' },
            { prop: 'nickname', label: '昵称', slot: true, fixed: 'left' },
            { prop: 'url', label: '链接', slot: true },
            { prop: 'description' , label: '描述', slot: true },
            { prop: 'audit', label: '审核状态', width: 100, slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    // 下拉框 - 移除默认分组选项
    select: {
        target: [{ value: '新窗口', label: '_blank' }, { value: '当前窗口', label: '_self' }],
        group: [], // 初始化为空数组，不包含默认分组
    },
})

const method = {
    // 获取分组
    async group() {
        const { code, data } = await axios.get('/api/links-group/column', {
            field: 'id,name,description,avatar',
        })
        if (code !== 200) return

        // 直接使用接口返回的分组数据，不添加默认分组
        state.select.group = data.map(item => ({ value: item.id, label: item.name, ...item }))
    },
    // 刷新数据
    init: async () => {
        // 重新加载数据
        await proxy.$refs['i-table']['init']()
    },
    // 保存数据
    save: async (params = state.struct || {}) => {

        if (utils.is.empty(params)) return ElMessage.warning('你在想什么？什么都不填！')
        if (utils.is.empty(params?.nickname)) return ElMessage.warning('您朋友叫什么？')

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
    // 审核友链
    async audit(ids = [], audit = 1) {

        if (utils.is.empty(ids)) return

        state.item.wait = true

        // 根据 API 规范，update 接口需要单个 id
        for (const id of ids) {
            const { code, msg } = await axios.put(`/api/${state.item.table}/update`, {
                id,
                audit
            })
            if (code !== 200) {
                state.item.wait = false
                return ElMessage.error(msg)
            }
        }

        state.item.wait = false

        // 刷新全部数据和审核数据
        emit('refresh', 'all')
        emit('refresh', 'audit')

        // 重新加载数据
        await method.init()
        ElMessage.success(audit === 1 ? '审核成功' : '已重置为待审核')
    },
    // 批量审核
    async batchAudit() {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要审核的友链')
        await method.audit(ids, 1)
    },
    // 批量改为待审核
    async batchUnAudit() {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要重置的友链')
        await method.audit(ids, 0)
    },
    // 选择变化
    selectionChange(selection) {
        state.item.selection = selection
    },
    // 上传
    async upload(field = 'image') {

        // 创建一个 input
        const input  = document.createElement('input')
        input.type   = 'file'
        input.accept = 'image/*'

        // 监听 input 的 change 事件
        input.addEventListener('change', async () => {
            const file = input.files[0]
            const { code: checkCode, data: checkData } = await axios.post('/api/attachment/checkType', { file_names: [file.name] })
            if (checkCode !== 200) {
                ElMessage.error('文件类型检查失败')
                return
            }
            const result = checkData.results?.[0]
            if (!result?.is_allowed) {
                ElMessage.error(result?.message || '不允许上传该类型的文件')
                return
            }

            // 创建一个 formData
            const params = new FormData
            params.append('files', file)
            params.append('target_type', 'links')

            state.item.upload         = true

            // 上传图片
            const { code, msg, data } = await axios.post('/api/attachment/batch', params)

            state.item.upload         = false

            if (code !== 200) return ElMessage.error(msg)
            // 设置图片
            state.struct[field] = data.results[0]?.full_url || ''
            // 清空 input
            input.value = ''
            ElMessage.success('上传成功！')
        })

        // 触发 input 的 click 事件
        input.click()
    },
    window(url = null, target = '_blank'){
        if (utils.is.empty(url)) return
        // 新窗口打开
        globalThis.open(url, target)
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
    await method.group()
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