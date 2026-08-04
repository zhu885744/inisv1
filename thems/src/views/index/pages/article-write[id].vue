<template>
    <div class="px-1 px-lg-0 mt-2">
        <div v-if="state.item.loading" class="d-flex justify-content-center align-items-center py-5">
            <div class="spinner-border" role="status">
                <span class="visually-hidden">加载中...</span>
            </div>
            <span class="ms-2">加载数据中...</span>
        </div>
        <div v-else>
            <i-md-editor ref="vditorRef" v-model="state.struct.content" :height="600"></i-md-editor>

            <div class="mt-4">
                <div class="card shadow-sm mb-2">
                    <div class="card-header bg-body-secondary">
                        <h6 class="mb-0">基本信息</h6>
                    </div>
                    <div class="card-body">
                        <div class="row g-4">
                            <div class="col-md-6">
                                <label class="form-label">
                                    <span class="text-danger me-1">*</span>
                                    <span>标题：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="（必须）文章的标题">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <input type="text" v-model="state.struct.title" class="form-control" placeholder="文章标题">
                            </div>
                            <div class="col-md-6">
                                <label class="form-label">
                                    <span>发布时间：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="文章的发布时间，留空则为当前时间">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <input type="datetime-local" v-model="state.struct.publishTime" class="form-control" step="1" placeholder="选择发布时间">
                            </div>
                            <div class="col-12">
                                <label class="form-label">
                                    <span>置顶：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="设置文章是否置顶">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <select v-model="state.struct.top" class="form-select">
                                    <option v-for="item in state.select.top" :key="item.value" :value="item.value">
                                        {{ item.label }}
                                    </option>
                                </select>
                            </div>
                            <div class="col-12">
                                <label class="form-label">
                                    <span>分类：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="可同时选择多个分类">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <select v-model="state.item.group" class="form-select" multiple size="3">
                                    <option v-for="item in state.select.group" :key="item.value" :value="item.value">
                                        {{ item.label }}
                                    </option>
                                </select>
                            </div>
                            <div class="col-12">
                                <label class="form-label">
                                    <span>摘要：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="文章的摘要">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <textarea v-model="state.struct.abstract" class="form-control" rows="3" placeholder="简单的描述一下您的文章"></textarea>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="card shadow-sm mb-2">
                    <div class="card-header bg-body-secondary">
                        <h6 class="mb-0">标签</h6>
                    </div>
                    <div class="card-body">
                        <div class="input-group mb-3 position-relative">
                            <input
                                v-model="newTag"
                                type="text"
                                class="form-control"
                                placeholder="输入标签名选择或新建"
                                @keyup.enter="addTag"
                                @focus="showTagDropdown = true"
                                @blur="hideTagDropdown"
                            />
                            <button class="btn btn-outline-secondary" type="button" @click="addTag">
                                <i class="bi bi-plus"></i>
                            </button>
                            <div v-show="showTagDropdown && filteredExistingTags.length > 0" class="tag-suggest-dropdown">
                                <div
                                    v-for="tag in filteredExistingTags"
                                    :key="tag.value"
                                    class="tag-suggest-item"
                                    @mousedown.prevent="selectExistingTag(tag)"
                                >
                                    {{ tag.label }}
                                </div>
                            </div>
                        </div>
                        <div v-if="state.item.tags.length > 0" class="d-flex flex-wrap gap-2">
                            <span v-for="(tag, index) in state.item.tags" :key="index" class="badge bg-primary">
                                {{ getTagName(tag) }}
                                <button type="button" class="btn-close btn-close-white btn-sm ms-1" @click="removeTag(index)"></button>
                            </span>
                        </div>
                        <div v-else class="text-muted text-sm">暂无标签</div>
                    </div>
                </div>

                <div class="card shadow-sm mb-2">
                    <div class="card-header bg-body-secondary">
                        <h6 class="mb-0">封面图</h6>
                    </div>
                    <div class="card-body">
                        <ul class="nav nav-tabs mb-3" id="coverTabs" role="tablist">
                            <li class="nav-item" role="presentation">
                                <button class="nav-link" :class="{ active: state.item.tabs === 'preview' }" id="preview-tab" data-bs-toggle="tab" data-bs-target="#preview" type="button" role="tab" aria-controls="preview" aria-selected="true" @click="state.item.tabs = 'preview'">上传</button>
                            </li>
                            <li class="nav-item" role="presentation">
                                <button class="nav-link" :class="{ active: state.item.tabs === 'links' }" id="links-tab" data-bs-toggle="tab" data-bs-target="#links" type="button" role="tab" aria-controls="links" aria-selected="false" @click="state.item.tabs = 'links'">外链</button>
                            </li>
                        </ul>
                        <div class="tab-content" id="coverTabsContent">
                            <div class="tab-pane" :class="{ active: state.item.tabs === 'preview', show: state.item.tabs === 'preview' }" id="preview" role="tabpanel" aria-labelledby="preview-tab">
                                <input type="file" class="form-control mb-3" @change="method.cover.upload" multiple accept="image/*">
                                <div v-if="state.item.cover.preview.length > 0" class="d-flex flex-wrap gap-3">
                                    <div v-for="(file, index) in state.item.cover.preview" :key="index" class="position-relative" style="width: 120px; height: 120px;">
                                        <img :src="file.url" class="img-fluid rounded-3" style="width: 100%; height: 100%; object-fit: cover;">
                                        <button type="button" class="btn btn-danger btn-sm position-absolute top-0 end-0" @click="method.cover.remove(index)">
                                            <i class="bi bi-x"></i>
                                        </button>
                                    </div>
                                </div>
                                <div v-else class="text-muted text-sm">暂无封面图</div>
                            </div>
                            <div class="tab-pane" :class="{ active: state.item.tabs === 'links', show: state.item.tabs === 'links' }" id="links" role="tabpanel" aria-labelledby="links-tab">
                                <textarea v-model="state.item.cover.links" class="form-control" rows="4" placeholder="外链图片地址，一行一个" @change="method.cover.change"></textarea>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="card shadow-sm mb-2">
                    <div class="card-header bg-body-secondary">
                        <h6 class="mb-0">高级选项</h6>
                    </div>
                    <div class="card-body">
                        <div class="row g-4">
                            <div class="col-md-6">
                                <label class="form-label">
                                    <span>允许评论：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="对当前文章的评论选项单独控制">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <select v-model="state.struct.json.comment.allow" class="form-select">
                                    <option value="">请选择</option>
                                    <option v-for="item in state.select.comment.allow" :key="item.value" :value="item.value">
                                        {{ item.label }} ({{ item.value }})
                                    </option>
                                </select>
                            </div>
                            <div class="col-md-6">
                                <label class="form-label">
                                    <span>显示评论：</span>
                                    <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="对当前文章的评论选项单独控制">
                                        <i class="bi bi-info-circle ms-1"></i>
                                    </span>
                                </label>
                                <select v-model="state.struct.json.comment.show" class="form-select">
                                    <option value="">请选择</option>
                                    <option v-for="item in state.select.comment.show" :key="item.value" :value="item.value">
                                        {{ item.label }} ({{ item.value }})
                                    </option>
                                </select>
                            </div>
                        </div>
                        <div v-if="store.comm.login.user.result.auth.all === true" class="mt-4 pt-4 border-top">
                            <label class="form-label">
                                <span>审核状态：</span>
                                <span class="text-muted" data-bs-toggle="tooltip" data-bs-placement="top" title="审核状态">
                                    <i class="bi bi-info-circle ms-1"></i>
                                </span>
                            </label>
                            <select v-model="state.struct.audit" class="form-select">
                                <option value="">请选择</option>
                                <option v-for="item in state.select.audit" :key="item.value" :value="item.value">
                                    {{ item.label }} ({{ item.value }})
                                </option>
                            </select>
                        </div>
                    </div>
                </div>

                <div class="d-flex justify-content-end gap-3 mb-4">
                    <button class="btn btn-outline-secondary" type="button" @click="method.saveAsDraft()" :disabled="state.item.wait">
                        <span v-if="state.item.wait" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        保存草稿
                    </button>
                    <button class="btn btn-primary" type="button" @click="method.save()" :disabled="state.item.wait">
                        <span v-if="state.item.wait" class="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        发布文章
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import utils from '@/utils/utils'
import { request, cache, checkFileType } from '@/utils/network'
import IMdEditor from '@/comps/custom/i-md-editor.vue'
import { useCommStore } from '@/store/comm'
import { usePageTitle, getSync } from '@/utils/app'
import { toast } from '@/utils/app'

const route = useRoute()
const router = useRouter()
const store = {
    comm: useCommStore(),
}

const { setDynamicTitle, setLoadingTitle } = usePageTitle({
  staticTitle: '',
  defaultTitle: '文章'
})

const updatePageTitle = () => {
    const id = route.params?.id
    if (id) {
        if (state.struct.title) {
            setDynamicTitle(`编辑文章 - ${state.struct.title}`)
        } else {
            setLoadingTitle('编辑文章')
        }
    } else {
        setDynamicTitle('撰写文章')
    }
}

const vditorRef = ref(null)
const newTag = ref('')
const showTagDropdown = ref(false)

const filteredExistingTags = computed(() => {
    const search = newTag.value.trim().toLowerCase()
    const selectedIds = new Set(state.item.tags)
    return state.select.tags.filter(tag =>
        !selectedIds.has(tag.value) && tag.label.toLowerCase().includes(search)
    )
})

const getCurrentDateTime = () => {
    const now = new Date()
    const year = now.getFullYear()
    const month = String(now.getMonth() + 1).padStart(2, '0')
    const day = String(now.getDate()).padStart(2, '0')
    const hours = String(now.getHours()).padStart(2, '0')
    const minutes = String(now.getMinutes()).padStart(2, '0')
    const seconds = String(now.getSeconds()).padStart(2, '0')
    return `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`
}

const originalCommentConfig = ref({ allow: 1, show: 1 })

const initOriginalCommentConfig = (config) => {
    if (config && config.comment) {
        originalCommentConfig.value = {
            allow: config.comment.allow ?? 1,
            show: config.comment.show ?? 1
        }
    }
}

const hasCommentConfigChanged = () => {
    const current = state.struct.json?.comment || {}
    return (
        current.allow !== originalCommentConfig.value.allow ||
        current.show !== originalCommentConfig.value.show
    )
}

const clearArticleCache = () => {
    try {
        const cacheKeysToClear = [
            `article_${state.item.id}`,
            'article_list',
            'article_count',
            'article_archives'
        ]
        
        cacheKeysToClear.forEach(key => {
            cache.del(key)
        })
        
        const pageCachePatterns = [
            'index_articles_page_',
            'author_user_info_',
            'author_user_stats_',
            'article_detail_',
            'article_comments_'
        ]
        
        const allCacheKeys = cache.keys()
        allCacheKeys.forEach(key => {
            if (pageCachePatterns.some(pattern => key.startsWith(pattern))) {
                cache.del(key)
            }
        })
        
        console.log('文章相关缓存已清空')
    } catch (error) {
        console.error('清空缓存失败:', error)
    }
}

const state = reactive({
    item: {
        id: null,
        tags: [],
        group: [],
        tabs: 'preview',
        cover: {
            preview: [],
            links: ''
        },
        loading: false,
        wait: false
    },
    struct: {
        content: '',
        publishTime: getCurrentDateTime(),
        json: { comment: { allow: 1, show: 1 } }
    },
    select: {
        top: [{ value: 1, label: '置顶' }, { value: 0, label: '不置顶' }],
        tags: [],
        group: [],
        comment: {
            allow: [
                { value: 1, label: '允许' },
                { value: 0, label: '禁止' },
            ],
            show: [
                { value: 1, label: '显示' },
                { value: 0, label: '隐藏' },
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

const method = {
    init: async () => {
        let id = route.params?.id
        if (!utils.is.empty(id)) {
            state.item.id = parseInt(id)
            state.item.loading = true
        }
        await method.getGroup()
        await method.getTags()
        if (!utils.is.empty(state.item.id)) await method.getArticle(state.item.id)
    },
    getArticle: async (id = null) => {
        try {
            const { code, msg, data } = await request.get('/api/article/one', { id })
            if (code !== 200) {
                await router.push({ path: '/' })
                toast.info(`已为您跳转到首页！${msg}`)
                return
            }

            const articleData = { ...data, json: Object.assign({}, data.json, state.struct.json) }
            
            if (data.publish_time && data.publish_time > 0) {
                const date = new Date(data.publish_time * 1000)
                const year = date.getFullYear()
                const month = String(date.getMonth() + 1).padStart(2, '0')
                const day = String(date.getDate()).padStart(2, '0')
                const hours = String(date.getHours()).padStart(2, '0')
                const minutes = String(date.getMinutes()).padStart(2, '0')
                const seconds = String(date.getSeconds()).padStart(2, '0')
                articleData.publishTime = `${year}-${month}-${day}T${hours}:${minutes}:${seconds}`
            } else {
                articleData.publishTime = getCurrentDateTime()
            }
            
            state.struct = articleData
            initOriginalCommentConfig(data.json)

            if (!utils.is.empty(data.covers)) {
                state.item.cover.preview = data.covers.split(',').map(item => ({
                    name: item.replace(/.*\//, ''), url: item,
                }))
            }
            if (!utils.is.empty(data.group)) {
                state.item.group = data.group.split('|').filter(item => !utils.is.empty(item)).map(item => parseInt(item))
            }
            if (!utils.is.empty(data.tags)) {
                state.item.tags = data.tags.split('|').filter(item => !utils.is.empty(item)).map(item => parseInt(item))
            }

            if (!utils.is.empty(id)) state.item.loading = false
        } catch (error) {
            console.error('获取文章信息失败:', error)
            toast.error('获取文章信息失败，请重试')
            state.item.loading = false
        }
    },
    getGroup: async () => {
        try {
            const { code, data } = await request.get('/api/article-group/column', {
                field: 'id,pid,name,avatar'
            })
            if (code !== 200) return
            state.backup.group = data
            state.select.group = data.map(item => ({ value: item.id, label: item.name }))
        } catch (error) {
            console.error('获取文章分组失败:', error)
        }
    },
    getTags: async () => {
        try {
            const { code, data } = await request.get('/api/tags/column', {
                field: 'id,name'
            })
            if (code !== 200) return
            state.select.tags = data.map(item => ({ value: item.id, label: item.name }))
        } catch (error) {
            console.error('获取文章标签失败:', error)
        }
    },
    save: async () => {
        try {
            if (!state.struct.content) {
                toast.warning('编辑器尚未加载完成，请稍候重试')
                return
            }

            const reg = /<[^>]+>/g
            let length = state.struct?.content?.replace(reg, '')?.replace(/\n/g, '')?.length || 0
            switch (length) {
                case 0:
                    return toast.warning('你这文章一个字都没写，糊弄谁呢？')
                case 1:
                    return toast.warning('真就只写一个字呗？')
                default:
                    if (length < 10) return toast.warning('你这太水了，10个字都不到。')
            }
            if (utils.is.empty(state.struct?.title)) return toast.warning('你可能忘记写标题了')

            let covers = state.item.cover.links.split('\n').filter(item => !utils.is.empty(item))
            let group = Array.from(new Set(state.item.group.filter(item => item))).sort((a, b) => a - b)

            state.struct.covers = !utils.is.empty(covers) ? covers.join(',') : ''
            state.struct.tags = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''
            state.struct.group = !utils.is.empty(group) ? `|${group.join('|')}|` : ''

            const saveData = { ...state.struct, json: JSON.stringify(state.struct.json) }
            saveData.status = 1
            if (state.struct.publishTime) {
                saveData.publish_time = Math.floor(new Date(state.struct.publishTime).getTime() / 1000)
            }
            delete saveData.publishTime

            state.item.wait = true

            let response
            if (state.item.id) {
                response = await request.put('/api/article/update', saveData)
            } else {
                response = await request.post('/api/article/create', saveData)
            }

            const { code, msg, data } = response

            state.item.wait = false

            if (code !== 200) return toast.error('保存失败：' + msg)

            toast.success('保存成功：' + msg)

            if (hasCommentConfigChanged()) {
                clearArticleCache()
            }

            state.item.id = data.id
            state.struct.id = data.id

            await router.push({ path: '/article-write/' + parseInt(data.id) })
        } catch (error) {
            console.error('保存文章失败:', error)
            toast.error('保存文章失败，请重试')
            state.item.wait = false
        }
    },
    saveAsDraft: async () => {
        try {
            if (utils.is.empty(state.struct?.content)) return toast.warning('你可能忘记写内容了')
            if (utils.is.empty(state.struct?.title)) return toast.warning('你可能忘记写标题了')

            let covers = state.item.cover.links.split('\n').filter(item => !utils.is.empty(item))
            let group = Array.from(new Set(state.item.group.filter(item => item))).sort((a, b) => a - b)

            state.struct.covers = !utils.is.empty(covers) ? covers.join(',') : ''
            state.struct.tags = !utils.is.empty(state.item.tags) ? `|${state.item.tags.join('|')}|` : ''
            state.struct.group = !utils.is.empty(group) ? `|${group.join('|')}|` : ''

            const saveData = { ...state.struct, json: JSON.stringify(state.struct.json) }
            saveData.status = 0
            if (state.struct.publishTime) {
                saveData.publish_time = Math.floor(new Date(state.struct.publishTime).getTime() / 1000)
            }
            delete saveData.publishTime

            state.item.wait = true

            let response
            if (state.item.id) {
                response = await request.put('/api/article/update', saveData)
            } else {
                response = await request.post('/api/article/create', saveData)
            }

            const { code, msg, data } = response

            state.item.wait = false

            if (code !== 200) return toast.error('保存失败：' + msg)

            toast.success('草稿保存成功：' + msg)

            if (hasCommentConfigChanged()) {
                clearArticleCache()
            }

            state.item.id = data.id
            state.struct.id = data.id

            await router.push({ path: '/article-write/' + parseInt(data.id) })
        } catch (error) {
            console.error('保存草稿失败:', error)
            toast.error('保存草稿失败，请重试')
            state.item.wait = false
        }
    },
    cover: {
        remove: (index) => {
            state.item.cover.preview.splice(index, 1)
        },
        upload: async (event) => {
            const files = event.target.files
            if (!files || files.length === 0) return

            try {
                const fileNames = Array.from(files).map(f => f.name)
                await checkFileType(fileNames)

                const formData = new FormData()
                for (let i = 0; i < files.length; i++) {
                    formData.append('files', files[i])
                }

                const { code, msg, data } = await request.post('/api/attachment/batch', formData, {
                    headers: method.headers()
                })

                if (code !== 200) {
                    return toast.error('上传失败：' + msg)
                }

                if (data?.results) {
                    data.results.forEach((result) => {
                        if (result.full_url) {
                            state.item.cover.preview.push({
                                name: result.original_name || '图片',
                                url: result.full_url
                            })
                        }
                    })
                }
            } catch (error) {
                console.error('上传图片失败:', error)
                toast.error('上传图片失败，请重试')
            }
        },
        change: (event) => {
            const value = event.target.value
            state.item.cover.preview = value.split('\n').filter(item => !utils.is.empty(item)).map(item => ({
                name: item.replace(/.*\//, ''),
                url: item
            }))
        }
    },
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
}

const addTag = async () => {
    const name = newTag.value.trim()
    if (!name) return

    if (state.item.tags.length >= 5) {
        toast.warning('最多添加 5 个标签')
        return
    }

    if (state.item.tags.find(tagId => {
        const tag = state.select.tags.find(t => t.value === tagId)
        return tag && tag.label.toLowerCase() === name.toLowerCase()
    })) {
        newTag.value = ''
        return
    }

    const existing = state.select.tags.find(tag => tag.label.toLowerCase() === name.toLowerCase())
    if (existing) {
        state.item.tags.push(existing.value)
        newTag.value = ''
        showTagDropdown.value = false
        return
    }

    try {
        const { code, msg, data } = await request.post('/api/tags/save', { name })
        if (code !== 200) {
            toast.error('添加标签失败：' + msg)
            return
        }

        state.select.tags.push({ value: data.id, label: name })
        state.item.tags.push(data.id)
        newTag.value = ''
        showTagDropdown.value = false
    } catch (error) {
        console.error('添加标签失败:', error)
        toast.error('添加标签失败，请重试')
    }
}

const selectExistingTag = (tag) => {
    if (state.item.tags.length >= 5) {
        toast.warning('最多添加 5 个标签')
        return
    }
    if (!state.item.tags.includes(tag.value)) {
        state.item.tags.push(tag.value)
    }
    newTag.value = ''
    showTagDropdown.value = false
}

const hideTagDropdown = () => {
    setTimeout(() => { showTagDropdown.value = false }, 150)
}

const removeTag = (index) => {
    state.item.tags.splice(index, 1)
}

const getTagName = (tagId) => {
    const tag = state.select.tags.find(t => t.value === tagId)
    return tag ? tag.label : tagId
}

onMounted(async () => {
    await method.init()
    updatePageTitle()
    
    if (window.bootstrap) {
        const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]')
        const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new window.bootstrap.Tooltip(tooltipTriggerEl))
        
        const tabElementList = document.querySelectorAll('[data-bs-toggle="tab"]')
        const tabList = [...tabElementList].map(tabEl => new window.bootstrap.Tab(tabEl))
    }
})

watch(() => state.item.cover.preview, (value = []) => {
    state.item.cover.links = value.map(item => item.url).join('\n') + '\n'
}, { deep: true })

watch(() => state.item.cover.links, (value) => {
    value = value?.replace(/[\s\n]/g, '')
    if (!utils.is.empty(value)) return
    state.item.cover.links = ''
})

watch(() => route.params?.id, (value) => {
    if (utils.is.empty(value)) return
    method.init()
    updatePageTitle()
})

watch(() => state.struct.title, () => {
    updatePageTitle()
})
</script>

<style scoped>
.tag-suggest-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 38px;
    z-index: 1000;
    background: var(--bs-body-bg);
    border: 1px solid var(--bs-border-color);
    border-radius: 0.375rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
    max-height: 200px;
    overflow-y: auto;
    margin-top: 2px;
}

.tag-suggest-item {
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    transition: background-color 0.15s ease;
    color: var(--bs-body-color);
}

.tag-suggest-item:hover {
    background-color: var(--bs-secondary-bg);
}
</style>