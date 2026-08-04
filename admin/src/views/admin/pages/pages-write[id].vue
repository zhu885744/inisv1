<template>
    <div class="container-box" style="padding-left: 4px; padding-right: 4px;">
        <el-row :gutter="20">
            <el-col :span="18">
                <div v-loading="utils.is.empty(state.struct.editor)" style="min-height: 485px">
                    <i-md-editor ref="mdEditor" v-model="state.struct.content" :height="600"></i-md-editor>
                </div>
                <el-card style="margin-bottom: 10px">
                    <el-button @click="method.save()" :loading="state.item.wait" style="float: right">发布页面</el-button>
                    <el-button @click="method.saveDraft()" :loading="state.item.wait">保存草稿</el-button>
                </el-card>
            </el-col>
            <el-col :span="6" v-loading="state.item.loading" id="page-header-title">
                <!-- 模块1：展示信息 独立卡片 -->
                <el-card header="展示信息" style="margin-bottom: 10px">
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="（必须）页面的标题" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px" class="required">标题：</span>
                            </span>
                        </el-tooltip>
                        <el-input v-model="state.struct.title" placeholder="页面标题"></el-input>
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
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="（必须）可以用做页面的唯一识别码或页面入口" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px" class="required">唯一键：</span>
                            </span>
                        </el-tooltip>
                        <el-input v-model="state.struct.key" autocomplete="new-password" placeholder="唯一识别码"></el-input>
                    </el-form-item>
                    <el-form-item>
                        <el-tooltip content="备注一下" placement="top">
                            <span>
                                <i-svg name="hint" size="14px"></i-svg>
                                <span style="margin-left: 4px">备注：</span>
                            </span>
                        </el-tooltip>
                        <el-input v-model="state.struct.remark" :autosize="{ minRows: 3, maxRows: 10 }" type="textarea"></el-input>
                    </el-form-item>
                </el-card>

                <!-- 模块2：高级选项 独立卡片 -->
                <el-card header="高级选项">
                    <el-form-item style="margin-bottom: 12px">
                        <el-tooltip content="可同时选择多个分类" placement="top">
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
                        <el-tooltip content="可同时选择多个分类" placement="top">
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
        tags: [],
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
    }
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
        await method.getTags()
        if (!utils.is.empty(state.item.id)) await method.getPage(state.item.id)
        else state.struct.editor = 'md'
    },
    // 获取文章标签
    getTags: async () => {
        const { code, data } = await axios.get('/api/tags/column', {
            field: 'id,name'
        })
        if (code !== 200) return
        state.select.tags = data.map(item => ({ value: item.id, label: item.name }))
    },
    // 获取页面信息
    getPage: async (id = null) => {
        const { code, msg, data } = await axios.get('/api/pages/one', { id })
        if (code !== 200) {
            await router.push({path: '/admin/pages/write'})
            ElMessage.error(msg)
            return ElMessage.warning('已为您跳转到页面撰写页！')
        }

        // 合并 json 项默认数据
        state.struct = {...data, json: Object.assign({}, data.json, state.struct.json), editor: 'vditor'}

        // 处理发布时间 - 转换时间戳为日期格式
        if (!utils.is.empty(data.publish_time)) {
            const date = new Date(data.publish_time * 1000)
            state.struct.publishTime = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`
        }

        // 封面图 - 字符串转数组 - name 正则出文件名部分
        if (!utils.is.empty(data.covers)) {
            state.item.fileList = data.covers.split(',').map(item => ({
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
    // 保存
    save: async () => {
        // 获取编辑器内容（v-model 已自动绑定）

        // 正则匹配 html 纯文本内容 - 去除换行符
        const reg = /<[^>]+>/g
        // 页面字数
        let length = state.struct?.content?.replace(reg, '')?.replace(/\n/g, '')?.length || 0
        switch (length) {
        case 0:

            return ElMessage.warning('你这页面一个字都没写，糊弄谁呢？')
        case 1:
            return ElMessage.warning('真就只写一个字呗？')
        default:
            if (length < 10) return ElMessage.warning('你这太水了，10个字都不到。')
        }
        if (utils.is.empty(state.struct?.title)) return ElMessage.warning('你可能忘记写标题了')

        state.struct.tags = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''

        state.item.wait = true

        const { publishTime, ...restStruct } = state.struct
        const publish_time = !utils.is.empty(publishTime) ? Math.floor(new Date(publishTime).getTime() / 1000) : null

        const { code, msg, data } = await axios.post('/api/pages/save', {
            ...restStruct,
            json: JSON.stringify(state.struct.json),
            publish_time
        })

        state.item.wait = false

        if (code !== 200) return ElMessage.error(msg)

        ElMessage.success(msg)

        state.item.id   = data.id
        state.struct.id = data.id

        await router.push({path: '/admin/pages/write/' + parseInt(data.id)})
    },
    // 保存草稿
    saveDraft: async () => {
        // 获取编辑器内容（v-model 已自动绑定）

        if (utils.is.empty(state.struct?.title)) return ElMessage.warning('你可能忘记写标题了')

        state.struct.tags = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''

        state.item.wait = true

        const { publishTime, ...restStruct } = state.struct
        const publish_time = !utils.is.empty(publishTime) ? Math.floor(new Date(publishTime).getTime() / 1000) : null

        const { code, msg, data } = await axios.post('/api/pages/save', {
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

watch(() => route.params?.id, (value) => {
    if (utils.is.empty(value)) return
    method.init()
})
</script>