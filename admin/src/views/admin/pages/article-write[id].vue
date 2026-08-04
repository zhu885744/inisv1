<template>
    <div class="container-box" style="padding-left: 4px; padding-right: 4px;">
        <el-row :gutter="20">
            <!-- 左侧编辑器区域 -->
            <el-col :span="18">
                <div v-loading="utils.is.empty(state.struct.editor)" style="min-height: 485px">
                    <i-md-editor ref="mdEditor" v-model="state.struct.content" :height="600"></i-md-editor>
                </div>
                <el-card style="margin-bottom: 10px">
                    <el-button @click="method.save()" :loading="state.item.wait" style="float: right">发布文章</el-button>
                    <el-button @click="method.saveDraft()" :loading="state.item.wait">保存草稿</el-button>
                </el-card>
            </el-col>

            <!-- 右侧侧边栏 四个独立Card，全部默认展开无折叠 -->
            <el-col :span="6" v-loading="state.item.loading" id="page-header-title">
                <!-- 模块1：展示信息 -->
                <el-card header="展示信息" style="margin-bottom: 10px">
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="（必须）文章的标题" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px" class="required">标题：</span>
                            </span>
                        </el-tooltip>
                        <el-input v-model="state.struct.title" placeholder="文章标题"></el-input>
                    </el-form-item>
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="文章的发布时间，留空则为当前时间" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">发布时间：</span>
                            </span>
                        </el-tooltip>
                        <el-date-picker
                            v-model="state.struct.publishTime"
                            type="datetime"
                            placeholder="选择发布时间"
                            format="YYYY-MM-DD HH:mm:ss"
                            value-format="YYYY-MM-DD HH:mm:ss"
                            style="width: 100%"
                        />
                    </el-form-item>
                    <el-form-item v-if="store.comm.login.user.result.auth.all === true" style="margin-bottom: 12px">
                        <el-tooltip content="审核状态" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">审核状态：</span>
                            </span>
                        </el-tooltip>
                        <el-select v-model="state.struct.audit" style="display: block; font-size: 13px" placeholder="请选择">
                            <el-option v-for="item in state.select.audit" :key="item.value" :label="item.label" :value="item.value">
                                <span style="font-size: 13px">{{ item.label }}</span>
                                <small style="color: #999; float: right">{{ item.value }}</small>
                            </el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="文章的摘要" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">摘要：</span>
                            </span>
                        </el-tooltip>
                        <el-input v-model="state.struct.abstract" :autosize="{ minRows: 3, maxRows: 10 }" placeholder="简单的描述一下您的文章" type="textarea">
                        </el-input>
                    </el-form-item>
                </el-card>

                <!-- 模块2：封面图 -->
                <el-card header="封面图" style="margin-bottom: 10px">
                    <el-tabs v-model="state.item.tabs" :stretch="true">
                        <el-tab-pane label="预览" name="preview">
                            <el-upload class="custom upload" action="/api/attachment/batch" :headers="method.headers()" :multiple="true" list-type="picture"
                                :before-upload="method.beforeUpload" :on-remove="method.cover.remove" :on-success="method.cover.success"
                                :on-error="method.cover.error" :file-list="state.item.cover.preview"
                                :data="{ target_type: 'article' }">
                                <el-button type="primary" style="width: 100%">上 传</el-button>
                            </el-upload>
                        </el-tab-pane>
                        <el-tab-pane label="外链" name="links">
                            <el-input v-model="state.item.cover.links" @change="method.cover.change" wrap="off"
                                :autosize="{ minRows: 3, maxRows: 10 }" placeholder="外链图片地址，一行一个" type="textarea">
                            </el-input>
                        </el-tab-pane>
                    </el-tabs>
                </el-card>

                <!-- 模块3：置顶、分类、标签 -->
                <el-card header="置顶、分类、标签" style="margin-bottom: 10px">
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="可同时选择多个分类" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">置顶：</span>
                            </span>
                        </el-tooltip>
                        <el-select v-model="state.struct.top" style="display: block" placeholder="请选择" filterable>
                            <el-option v-for="item in state.select.top" :key="item.value" :label="item.label" :value="item.value">
                            </el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="可同时选择多个分类" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">分类：</span>
                            </span>
                        </el-tooltip>
                        <el-cascader :options="state.select.group" :props="{ multiple: true, checkStrictly: true }"
                             v-model="state.item.group" collapse-tags clearable filterable style="width: 100%">
                        </el-cascader>
                    </el-form-item>
                    <el-form-item>
                        <el-tooltip content="可同时选择多个标签" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">标签：</span>
                            </span>
                        </el-tooltip>
                        <el-select v-model="state.item.tags" @change="method.change.tags"
                            multiple collapse-tags filterable allow-create default-first-option style="display: block" placeholder="请选择">
                            <el-option v-for="item in state.select.tags" :key="item.value" :label="item.label" :value="item.value">
                            </el-option>
                        </el-select>
                    </el-form-item>
                </el-card>

                <!-- 模块4：高级选项 -->
                <el-card header="高级选项">
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="对当前文章的评论选项单独控制" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">允许评论：</span>
                            </span>
                        </el-tooltip>
                        <el-select v-model="state.struct.json.comment.allow" style="display: block; font-size: 13px" placeholder="请选择">
                            <el-option v-for="item in state.select.comment.allow" :key="item.value" :label="item.label" :value="item.value">
                                <span style="font-size: 13px">{{ item.label }}</span>
                                <small style="color: #999; float: right">{{ item.value }}</small>
                            </el-option>
                        </el-select>
                    </el-form-item>
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="对当前文章的评论选项单独控制" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">显示评论：</span>
                            </span>
                        </el-tooltip>
                        <el-select v-model="state.struct.json.comment.show" style="display: block; font-size: 13px" placeholder="请选择">
                            <el-option v-for="item in state.select.comment.show" :key="item.value" :label="item.label" :value="item.value">
                                <span style="font-size: 13px">{{ item.label }}</span>
                                <small style="color: #999; float: right">{{ item.value }}</small>
                            </el-option>
                        </el-select>
                    </el-form-item>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script setup>
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'
import IMdEditor from '{src}/comps/custom/i-md-editor.vue'
import { useCommStore } from '{src}/store/comm'

const { ctx, proxy } = getCurrentInstance()

const route  = useRoute()
const router = useRouter()
const store  = {
    comm: useCommStore(),
}
const state  = reactive({
    item: {
        id: null,
        tags: [],
        group: [],
        tabs: 'preview',
        // 封面数据
        cover: {
            // 预览图
            preview: [],
            // 外链图
            links: ''
        },
        loading: false,
        wait: false
    },
    struct: {
        content: '',
        editor: 'vditor',
        publishTime: '',
        json: { comment: { allow: 1, show: 1 } }
    },
    select: {
        top: [{ value: 1, label: '置顶' }, { value: 0, label: '不置顶' }],
        tags: [],
        group: [],
        comment: {
            allow: [
                { value: 1, label: '允许' },
                { value: 2, label: '禁止' },
            ],
            show: [
                { value: 1, label: '显示' },
                { value: 2, label: '隐藏' },
            ]
        },
        audit: [
            { value: 0, label: '待审核' },
            { value: 1, label: '通过' },
            { value: 2, label: '不通过' },
        ],
    },
    backup: {
        group: [],
    },
})

onMounted(async () => {
    await method.init()
})

const method = {
    init: async () => {
        let id = route.params?.id
        if (!utils.is.empty(id)) {
            state.item.id = parseInt(id)
            state.item.loading = true
        }
        await method.getGroup()
        await method.getTags()
        if (!utils.is.empty(state.item.id))  await method.getArticle(state.item.id)
        else {
            state.struct.editor = 'md'
        }
    },
    // 获取文章信息
    getArticle: async (id = null) => {

        const { code, msg, data } = await axios.get('/api/article/one', { id })
        if (code !== 200) {
            await router.push({path: '/admin/article/write'})
            return ElMessage.error(`已为您跳转到文章撰写页！${msg}`)
        }

        // 合并 json 项默认数据
        state.struct = {...data, json: Object.assign({}, data.json, state.struct.json)}
        // 处理发布时间 - 转换时间戳为日期格式
        if (!utils.is.empty(data.publish_time)) {
            const date = new Date(data.publish_time * 1000)
            state.struct.publishTime = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
        }
        // 强制使用 Vditor
        state.struct.editor = 'md'

        // 封面图 - 字符串转数组 - name 正则出文件名部分
        if (!utils.is.empty(data.covers)) {
            state.item.cover.preview = data.covers.split(',').map(item => ({
                name: item.replace(/.*\//, ''), url: item,
            }))
        }
        // 分类 - 字符串转数组 - 去空 去重 转int
        if (!utils.is.empty(data.group)) {
            let group = data.group.split('|').filter(item => !utils.is.empty(item)).map(item => parseInt(item))
            state.item.group = method.tree.parse(state.backup.group, group)
        }
        // 标签 - 字符串转数组 - 去空 去重 转int
        if (!utils.is.empty(data.tags)) {
            state.item.tags = data.tags.split('|').filter(item => !utils.is.empty(item)).map(item => parseInt(item))
        }

        // 关闭加载状态
        if (!utils.is.empty(id)) state.item.loading = false
    },
    // 获取文章分组
    getGroup: async () => {
        const { code, data } = await axios.get('/api/article-group/column', {
            field: 'id,pid,name,avatar'
        })
        if (code !== 200) return
        state.backup.group = data
        state.select.group = method.tree.stringify(data, 0)
    },
    // 获取文章标签
    getTags: async () => {
        const { code, data } = await axios.get('/api/tags/column', {
            field: 'id,name'
        })
        if (code !== 200) return
        state.select.tags = data.map(item => ({ value: item.id, label: item.name }))
    },
    // 保存
    save: async () => {

        // 获取编辑器内容（v-model 已自动绑定）

        // 正则匹配纯文本内容 - 去除换行符
        const reg = /<[^>]+>/g
        // 文章字数
        let length = state.struct?.content?.replace(reg, '')?.replace(/\n/g, '')?.length || 0
        switch (length) {
        case 0:
            return ElMessage.warning('你这文章一个字都没写，糊弄谁呢？')
        case 1:
            return ElMessage.warning('真就只写一个字呗？')
        default:
            if (length < 10) return ElMessage.warning('你这太水了，10个字都不到。')
        }
        if (utils.is.empty(state.struct?.title)) return ElMessage.warning('你可能忘记写标题了')

        // 封面图 - 去空
        let covers = state.item.cover.links.split('\n').filter(item => !utils.is.empty(item))
        // 把 state.item.group 的二维数组转换成一维数组
        let group = state.item.group.reduce((prev, next) => prev.concat(next), [])
        // 去空 去重 排序
        group = Array.from(new Set(group.filter(item => item))).sort((a, b) => a - b)

        state.struct.covers = !utils.is.empty(covers) ? covers.join(',') : ''
        state.struct.tags   = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''
        state.struct.group  = !utils.is.empty(group) ? `|${group.join('|')}|` : ''

        state.item.wait = true

        const { publishTime, ...restStruct } = state.struct
        const publish_time = !utils.is.empty(publishTime) ? Math.floor(new Date(publishTime).getTime() / 1000) : null

        const { code, msg, data } = await axios.post('/api/article/save', {
            ...restStruct, 
            json: JSON.stringify(state.struct.json),
            publish_time,
            status: 1
        })

        state.item.wait = false

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success(msg)

        state.item.id   = data.id
        state.struct.id = data.id

        await router.push({path: '/admin/article/write/' + parseInt(data.id)})
    },
    // 保存草稿
    saveDraft: async () => {
        // 获取编辑器内容（v-model 已自动绑定）

        if (utils.is.empty(state.struct?.title)) return ElMessage.warning('你可能忘记写标题了')

        // 封面图 - 去空
        let covers = state.item.cover.links.split('\n').filter(item => !utils.is.empty(item))
        // 把 state.item.group 的二维数组转换成一维数组
        let group = state.item.group.reduce((prev, next) => prev.concat(next), [])
        // 去空 去重 排序
        group = Array.from(new Set(group.filter(item => item))).sort((a, b) => a - b)

        state.struct.covers = !utils.is.empty(covers) ? covers.join(',') : ''
        state.struct.tags   = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''
        state.struct.group  = !utils.is.empty(group) ? `|${group.join('|')}|` : ''

        state.item.wait = true

        const { publishTime, ...restStruct } = state.struct
        const publish_time = !utils.is.empty(publishTime) ? Math.floor(new Date(publishTime).getTime() / 1000) : null

        const { code, msg, data } = await axios.post('/api/article/save', {
            ...restStruct, 
            json: JSON.stringify(state.struct.json),
            publish_time,
            status: 0
        })

        state.item.wait = false

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success(msg)

        state.item.id   = data.id
        state.struct.id = data.id
    },
    // 树
    tree: {
        stringify: (data = [], pid = 0) => {
            const result = []
            for (const item of data) {
                if (item.pid === pid) {
                    const node = { value: item.id, label: item.name, children: [] }
                    node.children = method.tree.stringify(data, item.id)
                    result.push(node)
                }
            }
            return result
        },
        parse: (data, selected) => {
            let result = []
            for (let item of data) {
                if (selected.includes(item.id)) {
                    if (item.pid === 0) result.push([item.id])
                    else {
                        for (let i = 0; i < result.length; i++) {
                            if (result[i][result[i].length - 1] === item.pid) {
                                result.push([...result[i], item.id])
                                break
                            }
                        }
                    }
                }
            }
            return result
        }
    },
    beforeUpload: async (file) => {
        const { code, data } = await axios.post('/api/attachment/checkType', { file_names: [file.name] })
        if (code !== 200) {
            ElMessage.error('文件类型检查失败')
            return false
        }
        const result = data.results?.[0]
        if (!result?.is_allowed) {
            ElMessage.error(result?.message || '不允许上传该类型的文件')
            return false
        }
        return true
    },
    cover: {
        // 移除封面图
        remove: (file) => {
            delete state.item.cover.preview[file.uid]
        },
        // 上传成功
        success: async (response, file, list) => {

            const { code, msg } = response
            if (code !== 200) return ElMessage.error(msg)

            for (let key = 0; key < list.length; key++) {
                // 判断是否存在 response
                if (list[key].response) {
                    const { data } = list[key].response
                    if (!data?.results?.[0]?.full_url) continue
                    const result = data.results[0]
                    list[key] = { name: result.original_name, url: result.full_url }
                }
            }
            state.item.cover.preview = list
        },
        // 上传失败
        error: (err, file, fileList) => {
            console.log(err, file, fileList)
        },
        // 外链输入框事件
        change: (data) => {
            state.item.cover.preview = data.split('\n').filter(item => !utils.is.empty(item)).map(item => ({ name: item.replace(/.*\//, ''), url: item }))
        }
    },
    // 文件上传自定义头
    headers: () => {
        let result = {}
        if (!utils.is.empty(globalThis?.inis?.api?.key)) {
            result['i-api-key'] = globalThis?.inis?.api?.key
        }
        let TOKEN_NAME = globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN'
        if (utils.has.cookie(TOKEN_NAME)) {
            let token = utils.get.cookie(TOKEN_NAME)
            if (!utils.is.empty(token)) {
                result.Authorization = token
            }
        }
        return result
    },
    empty: value => utils.is.empty(value),
    // 数据变化
    change: {
        // 标签变化
        tags: (data) => {

            if (utils.is.empty(data)) return

            data.forEach(async (item, index) => {
                if (typeof item === 'string') {
                    const {code, msg, data} = await axios.post('/api/tags/save', {name: item})
                    // 创建失败，删除对应的 tag
                    if (code !== 200) {
                        ElMessage.error('添加标签失败：' + msg)
                        return state.item.tags.splice(index, 1)
                    }
                    // 把原来的 tag 替换成新的 tag.id
                    state.item.tags[index] = data.id
                    // 把新的 tag 添加到 select.tags 列表中
                    state.select.tags.push({value: data.id, label: item})
                }
            })
        }
    },
}

// 监听封面图预览图变化
watch(() => state.item.cover.preview, (value = {}) => {
    state.item.cover.links = value.map(item => item.url).join('\n') + '\n'
}, { deep: true })

watch(() => state.item.cover.links, (value) => {
    // 去除空格和换行
    value = value?.replace(/[\s\n]/g, '')
    // 判断是否为空
    if (!utils.is.empty(value)) return
    state.item.cover.links = ''
})

watch(() => route.params?.id, (value) => {
    if (utils.is.empty(value)) return
    method.init()
})
</script>