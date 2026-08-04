<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'all'">
            <el-button v-on:click="method.batchDelete(true)" type="danger" size="small">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量删除</span>
            </el-button>
        </div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'remove'">
            <el-button v-on:click="method.batchRestore()" type="primary" size="small">
                <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                <span style="margin-left: 4px">批量恢复</span>
            </el-button>
            <el-button v-on:click="method.batchDelete(false)" type="danger" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量永久删除</span>
            </el-button>
        </div>
    <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">

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

        <template #i-user="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-tooltip :disabled="utils.is.empty(scope?.result?.author?.description)" placement="top">
                    <template #content>
                        <span v-html="method.autoWrap(scope?.result?.author?.nickname + '：' + scope?.result?.author?.description)"></span>
                    </template>
                    <el-avatar shape="square" :src="method.imageSize(scope?.result?.author?.avatar)" size="small" style="margin-right: 8px"></el-avatar>
                </el-tooltip>
                <el-tooltip v-if="!utils.is.empty(scope.url)" :content="`链接：${scope.url}`" placement="top">
                    <i-svg color="rgb(var(--icon-color))" v-on:click="method.window(scope.url)" name="link" size="12px" style="margin-right: 4px"></i-svg>
                </el-tooltip>
                <el-tooltip :content="scope?.result?.author?.nickname" :disabled="utils.is.empty(scope?.result?.author?.nickname)" placement="top">
                    <span>{{ method.omit(scope?.result?.author?.nickname, 4, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-content="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.content)" placement="top">
                <template #content>
                    <span class="user-select-text comment markdown" style="max-width: 400px; display: block; line-height: 1.7">
                        <i-markdown v-if="scope?.editor === 'markdown'" :model-value="method.renderEmoji(scope.content)"></i-markdown>
                        <span v-else-if="scope?.editor === 'html'" v-html="method.renderEmoji(scope.content)"></span>
                        <span v-else v-html="method.renderEmoji(scope?.content || '-')"></span>
                    </span>
                </template>
                <span 
                    class="user-select-text comment markdown content-cell-inline"
                    style="display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 320px; vertical-align: middle"
                >
                    <i-markdown v-if="scope?.editor === 'markdown'" :model-value="method.renderEmojiInline(scope.content)"></i-markdown>
                    <template v-else>
                        <span v-html="method.renderEmojiInline(scope?.content || '-')"></span>
                    </template>
                </span>
            </el-tooltip>
        </template>

        <template #i-source="{ scope = {} }">
            <el-tooltip :content="method.getSourceTitle(scope)" :disabled="utils.is.empty(method.getSourceTitle(scope))" placement="top">
                <span v-if="!utils.is.empty(method.getSourceTitle(scope))" class="limit-1-line">
                    {{ method.getSourceTitle(scope) || '-' }}
                </span>
                <span v-else>-</span>
            </el-tooltip>
        </template>

        <template #i-type="{ scope = {} }">
            <el-tag :type="method.getTypeTag(scope.bind_type)" size="small">
                {{ method.getTypeName(scope.bind_type) }}
            </el-tag>
        </template>

    </i-table>
    </div>

    <el-dialog v-model="state.item.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong class="flex-center">{{ utils.is.empty(state.struct.id) ? '添 加' : '编 辑' }} 评 论</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="内容">
                    <el-input v-model="state.struct.content" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea" placeholder="请输入评论内容"></el-input>
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
import IMarkdown from '{src}/comps/custom/i-markdown.vue'

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
        table: 'comment',
        dialog: false,
        wait: false,
        selection: [],
    },
    struct: {},
    opts: {
        url: '/api/comment/all',
        params: props.params,
        columns: [
            { prop: 'user', label: '用户', slot: true, fixed: left },
            { prop: 'content', label: '内容', slot: true },
            { prop: 'type', label: '类型', width: 100, slot: true, align: 'center' },
            { prop: 'source' , label: '源内容', slot: true },
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

        if (utils.is.empty(params)) return ElMessage.warning('你在想什么？什么都不填！')  // 使用Element Plus的Message
        if (utils.is.empty(params?.content)) return ElMessage.warning('评论内容怎么能是空的呢？！')  // 使用Element Plus的Message

        state.item.wait     = true

        const { code, msg } = await axios.post(`/api/${state.item.table}/save`, params)

        state.item.wait     = false

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        // 关闭对话框
        state.item.dialog = false
        // 重新加载数据
        await method.init()
        ElMessage.success('保存成功')  // 使用Element Plus的Message
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

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        // 刷新回收站数据
        emit('refresh', 'remove')

        // 重新加载数据
        await method.init()
        ElMessage.success('删除成功')  // 使用Element Plus的Message
    },
    // 恢复数据
    async restore(ids = []) {

        if (utils.is.empty(ids)) return

        const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })

        if (code !== 200) return ElMessage.error(msg)  // 使用Element Plus的Message

        // 刷新全部数据
        emit('refresh', 'all')

        // 重新加载数据
        await method.init()
        ElMessage.success('恢复成功')  // 使用Element Plus的Message
    },
    // 选择变化
    selectionChange(selection) {
        state.item.selection = selection
    },
    // 批量删除
    async batchDelete(isSoft = true) {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要操作的评论')

        try {
            await ElMessageBox.confirm(
                `确定要${isSoft ? '软删除' : '永久删除'}选中的 ${ids.length} 条评论吗？`,
                '提示',
                { type: 'warning' }
            )
        } catch {
            return
        }

        state.item.wait = true
        try {
            const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`
            const { code, msg } = await axios.del(uri, { ids })
            state.item.wait = false
            if (code !== 200) throw new Error(msg)

            ElMessage.success(isSoft ? '批量软删除成功！' : '批量永久删除成功！')
            emit('refresh', 'remove')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '删除失败')
        }
    },
    // 批量恢复
    async batchRestore() {
        const ids = state.item.selection.map(item => item.id)
        if (utils.is.empty(ids)) return ElMessage.warning('请选择要恢复的评论')

        state.item.wait = true
        try {
            const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })
            state.item.wait = false
            if (code !== 200) throw new Error(msg)

            ElMessage.success('批量恢复成功！')
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '恢复失败')
        }
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

        if (!utils.is.empty(msg)) return ElMessage.info(msg)  // 使用Element Plus的Message
    },
    // 省略文本
    omit  : (text = null, length = 10, omission = ' ... ', location = 'center') => {
        if (utils.is.empty(text)) return '空'
        return utils.string.omit(text, length, omission, location)
    },
    // 获取类型名称
    getTypeName: (bindType = '') => {
        const types = {
            article: '文章',
            page: '页面',
            moments: '动态',
        }
        return types[bindType] || '未知'
    },
    // 获取类型标签样式
    getTypeTag: (bindType = '') => {
        const tags = {
            article: 'success',
            page: 'warning',
            moments: 'primary',
        }
        return tags[bindType] || 'info'
    },
    // 获取源内容标题
    getSourceTitle: (scope = {}) => {
        const bindType = scope.bind_type || ''
        const result = scope.result || {}
        switch (bindType) {
            case 'article':
                return result.article?.title || ''
            case 'page':
                return result.page?.title || ''
            case 'moments':
                return result.moments?.content ? method.omit(result.moments.content, 30) : ''
            default:
                return ''
        }
    },
    renderEmoji: (text = '') => {
        return utils.string.emoji(text)
    },
    renderEmojiInline: (text = '') => {
        if (!text) return ''
        const emojiHtml = utils.string.emoji(String(text).replace(/\n/g, ' '))
        return emojiHtml.replace(/class="emoji-img"/g, `class="emoji-img" style="width:16px;height:16px;max-width:16px;max-height:16px"`)
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