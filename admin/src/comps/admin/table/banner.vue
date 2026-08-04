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

        <template #i-title="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-tooltip placement="top" effect="light">
                    <template #content>
                        <el-image v-on:click="method.window(scope?.image)" :src="method.imageSize(scope?.image, '300x160')" fit="cover" :lazy="true" style="width: 300px; height: 160px">
                            <template #error class="image-slot">
                                <el-image src="/assets/images/gif/404.gif" fit="cover" :lazy="true" style="width: 300px; height: 160px"></el-image>
                            </template>
                        </el-image>
                    </template>
                    <el-avatar shape="square" :src="method.imageSize(scope?.image)" size="small" style="margin-right: 8px"></el-avatar>
                </el-tooltip>
                <el-tooltip v-if="!utils.is.empty(scope.url)" :content="`链接：${scope.url}`" placement="top">
                    <i-svg color="rgb(var(--icon-color))" v-on:click="method.window(scope.url)" name="link" size="12px" style="margin-right: 4px"></i-svg>
                </el-tooltip>
                <el-tooltip :content="scope.title" :disabled="utils.is.empty(scope.title)" placement="top">
                    <span>{{ method.omit(scope?.title, 4, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-content="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.content)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.content)"></span>
                </template>
                <span>{{ method.omit(scope?.content) }}</span>
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
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 轮 播</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="标题">
                    <el-input v-model="state.struct.title" placeholder="为空不显示"></el-input>
                </el-form-item>
                <el-form-item label="时间">
                    <el-date-picker v-model="state.struct.time" type="datetimerange" start-placeholder="开始时间" end-placeholder="结束时间">
                    </el-date-picker>
                </el-form-item>
                <el-form-item label="跳转链接">
                    <el-input v-model="state.struct.url" placeholder="如：https://inis.cn"></el-input>
                </el-form-item>
                <el-form-item label="跳转方式">
                    <el-select v-model="state.struct.target" placeholder="请选择方式" class="custom">
                        <el-option v-for="item in state.select.target" :key="item.value" :label="item.value" :value="item.label">
                            <span style="font-size: 13px">{{ item.value }}</span>
                            <small style="float: right">{{ item.label }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="图片地址">
                    <el-input v-model="state.struct.image" class="custom" placeholder="填写图片地址或点击上传图片">
                        <template #append>
                            <el-button v-on:click="method.upload()" :loading="state.item.upload">
                                <i-svg v-if="!state.item.upload" name="upload" color="rgb(var(--icon-color))" size="15px"></i-svg>
                                <span style="margin-left: 4px">上传</span>
                            </el-button>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item label="内容">
                    <el-input v-model="state.struct.content" :autosize="{ minRows: 1, maxRows: 10 }" type="textarea" placeholder="为空不显示"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 1, maxRows: 10 }" placeholder="备注一下，避免忘记！" type="textarea"></el-input>
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
        table: 'banner',
        dialog: false,
        upload: false,
        wait: false,
    },
    struct: {
        time: null,
        target: '_blank',
    },
    opts: {
        url: '/api/banner/all',
        params: props.params,
        columns: [
            { prop: 'title'  , label: '标题', slot: true, fixed: left },
            { prop: 'content', label: '内容', slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'update_time', label: '更新时间', width: 140, sortable: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    // 下拉框
    select: {
        target: [{ value: '新窗口', label: '_blank' }, { value: '当前窗口', label: '_self' }],
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
        if (utils.is.empty(params?.image)) return ElMessage.warning('兄dei，图片地址没有设置！')

        // 日期格式转时间戳
        let time1 = Date.parse(new Date(params.time?.[0]))
        let time2 = Date.parse(new Date(params.time?.[1]))
        params.start_time = utils.is.empty(time1) ? Math.round(new Date() / 1000) : Math.round(time1 / 1000)
        params.end_time   = utils.is.empty(time2) ? params.start_time + 86400 : Math.round(time2 / 1000)
        delete params.time

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

        // 时间戳转日期格式
        struct.time = []
        for (const key in struct) {
            if (key === 'start_time') struct.time[0] = new Date(struct[key] * 1000)
            if (key === 'end_time')   struct.time[1] = new Date(struct[key] * 1000)
        }

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
            params.append('target_type', 'banner')

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
    // 打开新窗口
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

        if (!utils.is.empty(msg)) return ElMessage.info(msg)  // 使用ElMessage
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