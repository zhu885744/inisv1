<template>
    <div ref="containerRef" class="md-editor" :class="{ 'fullscreen': isFullscreen }">
        <div class="md-toolbar">
            <div class="toolbar-group">
                <el-tooltip content="加粗 (Ctrl+B)" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('bold')"><el-icon><EditPen /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="斜体 (Ctrl+I)" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('italic')"><el-icon><Edit /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="删除线" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('strikethrough')"><el-icon><Delete /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="下划线 (Ctrl+U)" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('underline')"><el-icon><EditPen /></el-icon></el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-divider"></div>

            <div class="toolbar-group">
                <el-tooltip content="标题1" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('h1')" class="toolbar-text-btn">H1</el-button>
                </el-tooltip>
                <el-tooltip content="标题2" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('h2')" class="toolbar-text-btn">H2</el-button>
                </el-tooltip>
                <el-tooltip content="标题3" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('h3')" class="toolbar-text-btn">H3</el-button>
                </el-tooltip>
                <el-tooltip content="标题4" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('h4')" class="toolbar-text-btn">H4</el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-divider"></div>

            <div class="toolbar-group">
                <el-tooltip content="引用" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('quote')"><el-icon><Message /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="行内代码" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('code')"><el-icon><Document /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="代码块" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('codeblock')"><el-icon><DocumentCopy /></el-icon></el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-divider"></div>

            <div class="toolbar-group">
                <el-tooltip content="无序列表" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('ul')"><el-icon><List /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="有序列表" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('ol')"><el-icon><List /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="任务列表" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('task')"><el-icon><Check /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="折叠列表" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('toggle')"><el-icon><CaretRight /></el-icon></el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-divider"></div>

            <div class="toolbar-group">
                <el-tooltip content="链接" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('link')"><el-icon><Link /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="图片" placement="top">
                    <el-button type="text" size="small" @click="showImageModal = true"><el-icon><Picture /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="表格" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('table')"><el-icon><Grid /></el-icon></el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-divider"></div>

            <div class="toolbar-group">
                <el-tooltip content="分割线" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('hr')"><el-icon><Minus /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="Emoji" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('emoji')"><el-icon><ChatDotSquare /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="撤销 (Ctrl+Z)" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('undo')"><el-icon><RefreshLeft /></el-icon></el-button>
                </el-tooltip>
                <el-tooltip content="重做 (Ctrl+Y)" placement="top">
                    <el-button type="text" size="small" @click="insertFormat('redo')"><el-icon><RefreshRight /></el-icon></el-button>
                </el-tooltip>
            </div>

            <div class="toolbar-spacer"></div>

            <div class="toolbar-group">
                <el-tooltip content="预览" placement="top">
                    <el-button type="text" size="small" :class="{ active: showPreview }" @click="showPreview = !showPreview">
                        <el-icon><View /></el-icon>
                    </el-button>
                </el-tooltip>
                <el-tooltip :content="isFullscreen ? '退出全屏' : '全屏'" placement="top">
                    <el-button type="text" size="small" :class="{ active: isFullscreen }" @click="toggleFullscreen">
                        <el-icon><component :is="isFullscreen ? 'ScaleToOriginal' : 'FullScreen'" /></el-icon>
                    </el-button>
                </el-tooltip>
            </div>
        </div>

        <el-progress 
            v-if="state.progress.show" 
            :percentage="state.progress.value" 
            :stroke-width="6" 
            :show-text="true"
            :text-inside="false"
            style="margin: 8px 12px"
        />

        <div class="md-body" :class="{ 'fullscreen': isFullscreen }" :style="{ height: computedHeight + 'px' }">
            <div class="md-edit-area" :style="{ width: showPreview ? '50%' : '100%' }">
                <textarea
                    ref="textareaRef"
                    v-model="state.content"
                    class="md-textarea"
                    :placeholder="placeholder"
                    @keydown="handleKeydown"
                    @input="handleInput"
                    @paste="handlePaste"
                    @dragover.prevent
                    @drop="handleDrop"
                ></textarea>
                <div class="md-line-numbers" v-if="showLineNumbers">
                    <span v-for="n in lineCount" :key="n">{{ n }}</span>
                </div>
            </div>

            <div v-if="showPreview" class="md-preview-area">
                <div class="article-content" v-html="renderedContent"></div>
            </div>
        </div>

        <el-alert 
            v-if="state.error" 
            :title="state.error" 
            type="error" 
            closable 
            @close="state.error = ''"
            style="margin: 0"
        />

        <el-dialog v-model="showImageModal" title="插入图片" width="480px" destroy-on-close>
            <el-form label-width="auto" :model="imageForm">
                <el-form-item label="方式一：上传图片">
                    <el-upload
                        class="upload-demo"
                        :show-file-list="false"
                        :before-upload="beforeUpload"
                        :http-request="customUpload"
                    >
                        <el-button type="primary" :disabled="uploadingImage">
                            <el-icon><Upload /></el-icon> {{ uploadingImage ? '上传中...' : '选择图片上传' }}
                        </el-button>
                    </el-upload>
                </el-form-item>
                <el-form-item label="方式二：输入图片链接">
                    <el-input 
                        v-model="imageUrl" 
                        placeholder="请输入图片链接" 
                        @keyup.enter="insertImageFromUrl"
                    >
                        <template #append>
                            <el-button @click="insertImageFromUrl" :disabled="!imageUrl.trim()">插入</el-button>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item v-if="previewImage" label="图片预览">
                    <img :src="previewImage" class="img-preview" />
                </el-form-item>
            </el-form>
        </el-dialog>
    </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { EditPen, Edit, Delete, Message, Document, DocumentCopy, List, Check, CaretRight, Link, Picture, Grid, Minus, ChatDotSquare, RefreshLeft, RefreshRight, View, ScaleToOriginal, FullScreen, Upload } from '@element-plus/icons-vue'
import request from '{src}/utils/request'
import utils from '{src}/utils/utils'
import MarkdownIt from 'markdown-it'

const props = defineProps({
    modelValue: {
        type: String,
        default: ''
    },
    placeholder: {
        type: String,
        default: '写点什么吧！'
    },
    height: {
        type: Number,
        default: 400
    },
    preview: {
        type: Boolean,
        default: false
    },
    lineNumbers: {
        type: Boolean,
        default: false
    }
})

const emit = defineEmits(['update:modelValue'])

const textareaRef = ref(null)
const showImageModal = ref(false)
const imageUrl = ref('')
const previewImage = ref('')
const uploadingImage = ref(false)
const isFullscreen = ref(false)
const showLineNumbers = ref(props.lineNumbers)
const editorHeight = ref(500)
const containerRef = ref(null)
let resizeObserver = null

const md = new MarkdownIt({
    html: true,
    linkify: true,
    typographer: true
})

const state = reactive({
    content: '',
    progress: {
        value: 0,
        show: false
    },
    error: '',
    history: [],
    historyIndex: -1
})

const imageForm = reactive({
    url: ''
})

const showPreview = ref(props.preview)

const lineCount = computed(() => {
    return state.content.split('\n').length
})

const computedHeight = computed(() => {
    if (isFullscreen.value) {
        return window.innerHeight - 52
    }
    return editorHeight.value
})

const handleResize = () => {
    if (!containerRef.value || isFullscreen.value) return
    
    const parent = containerRef.value.parentElement
    if (parent) {
        const parentHeight = parent.clientHeight
        const toolbarHeight = 52
        const padding = 20
        
        const viewportHeight = window.innerHeight - 150
        const maxHeight = Math.min(parentHeight, viewportHeight)
        const newHeight = Math.max(400, Math.min(maxHeight - toolbarHeight - padding, 600))
        editorHeight.value = newHeight
    }
}

watch(() => props.modelValue, (newVal) => {
    if (newVal !== state.content) {
        state.content = newVal
    }
}, { immediate: true })

const renderedContent = computed(() => {
    if (!state.content) return ''
    return md.render(state.content)
})

const handleInput = () => {
    emit('update:modelValue', state.content)
    debouncedSaveHistory()
}

let saveHistoryTimer = null
const saveHistory = () => {
    if (state.historyIndex < state.history.length - 1) {
        state.history = state.history.slice(0, state.historyIndex + 1)
    }
    state.history.push(state.content)
    state.historyIndex = state.history.length - 1
    if (state.history.length > 100) {
        state.history.shift()
        state.historyIndex--
    }
}

const debouncedSaveHistory = () => {
    if (saveHistoryTimer) {
        clearTimeout(saveHistoryTimer)
    }
    saveHistoryTimer = setTimeout(saveHistory, 500)
}

const handleKeydown = (e) => {
    if (e.key === 'Tab') {
        e.preventDefault()
        insertText('  ')
        return
    }

    if (e.ctrlKey && e.key === 'b') {
        e.preventDefault()
        insertFormat('bold')
        return
    }

    if (e.ctrlKey && e.key === 'i') {
        e.preventDefault()
        insertFormat('italic')
        return
    }

    if (e.ctrlKey && e.key === 'u') {
        e.preventDefault()
        insertFormat('underline')
        return
    }

    if (e.ctrlKey && e.key === 'z') {
        e.preventDefault()
        undo()
        return
    }

    if (e.ctrlKey && e.key === 'y') {
        e.preventDefault()
        redo()
        return
    }
}

const insertFormat = (type) => {
    const textarea = textareaRef.value
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const text = state.content
    const selectedText = text.substring(start, end)

    let insertText = ''
    let cursorOffset = 0

    switch (type) {
        case 'bold':
            insertText = `**${selectedText || '粗体文本'}**`
            cursorOffset = selectedText ? insertText.length : 2
            break
        case 'italic':
            insertText = `*${selectedText || '斜体文本'}*`
            cursorOffset = selectedText ? insertText.length : 1
            break
        case 'underline':
            insertText = `<u>${selectedText || '下划线文本'}</u>`
            cursorOffset = selectedText ? insertText.length : 3
            break
        case 'strikethrough':
            insertText = `~~${selectedText || '删除线文本'}~~`
            cursorOffset = selectedText ? insertText.length : 2
            break
        case 'h1':
            insertText = `# ${selectedText || '标题1'}`
            cursorOffset = insertText.length
            break
        case 'h2':
            insertText = `## ${selectedText || '标题2'}`
            cursorOffset = insertText.length
            break
        case 'h3':
            insertText = `### ${selectedText || '标题3'}`
            cursorOffset = insertText.length
            break
        case 'h4':
            insertText = `#### ${selectedText || '标题4'}`
            cursorOffset = insertText.length
            break
        case 'quote':
            insertText = `> ${selectedText || '引用文本'}`
            cursorOffset = insertText.length
            break
        case 'code':
            insertText = `\`${selectedText || '代码'}\``
            cursorOffset = selectedText ? insertText.length : 1
            break
        case 'codeblock':
            insertText = `\`\`\`\n${selectedText || '代码块'}\n\`\`\``
            cursorOffset = selectedText ? insertText.length : 4
            break
        case 'ul':
            insertText = `- ${selectedText || '列表项'}`
            cursorOffset = insertText.length
            break
        case 'ol':
            insertText = `1. ${selectedText || '列表项'}`
            cursorOffset = insertText.length
            break
        case 'task':
            insertText = `- [ ] ${selectedText || '任务项'}`
            cursorOffset = insertText.length
            break
        case 'toggle':
            insertText = `<details>\n<summary>${selectedText || '标题'}</summary>\n${selectedText || '内容'}\n</details>`
            cursorOffset = insertText.length
            break
        case 'link':
            insertText = `[${selectedText || '链接文本'}](url)`
            cursorOffset = selectedText ? insertText.length - 1 : 1
            break
        case 'table':
            insertText = `| 列1 | 列2 | 列3 |\n| --- | --- | --- |\n| 内容 | 内容 | 内容 |`
            cursorOffset = insertText.length
            break
        case 'emoji':
            insertText = `:${selectedText || 'smile'}:`
            cursorOffset = selectedText ? insertText.length : 1
            break
        case 'hr':
            insertText = '\n---\n'
            cursorOffset = insertText.length
            break
        case 'undo':
            undo()
            return
        case 'redo':
            redo()
            return
        default:
            return
    }

    insertTextAtCursor(insertText, cursorOffset)
}

const insertText = (text) => {
    const textarea = textareaRef.value
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const content = state.content

    state.content = content.substring(0, start) + text + content.substring(end)
    emit('update:modelValue', state.content)

    setTimeout(() => {
        textarea.focus()
        textarea.selectionStart = textarea.selectionEnd = start + text.length
    }, 0)
}

const insertTextAtCursor = (text, cursorOffset) => {
    const textarea = textareaRef.value
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const content = state.content

    state.content = content.substring(0, start) + text + content.substring(end)
    emit('update:modelValue', state.content)

    setTimeout(() => {
        textarea.focus()
        textarea.selectionStart = textarea.selectionEnd = start + cursorOffset
    }, 0)
}

const undo = () => {
    if (state.historyIndex > 0) {
        state.historyIndex--
        state.content = state.history[state.historyIndex]
        emit('update:modelValue', state.content)
    }
}

const redo = () => {
    if (state.historyIndex < state.history.length - 1) {
        state.historyIndex++
        state.content = state.history[state.historyIndex]
        emit('update:modelValue', state.content)
    }
}

const beforeUpload = async (file) => {
    if (!file.type.startsWith('image/')) {
        ElMessage.error('请选择图片文件')
        return false
    }

    if (file.size > 5 * 1024 * 1024) {
        ElMessage.error('图片大小不能超过 5MB')
        return false
    }

    const { code, data } = await request.post('/api/attachment/checkType', { file_names: [file.name] })
    if (code !== 200) {
        ElMessage.error('文件类型检查失败')
        return false
    }
    const result = data.results?.[0]
    if (!result?.is_allowed) {
        ElMessage.error(result?.message || '不允许上传该类型的文件')
        return false
    }

    uploadingImage.value = true
    state.progress.show = true
    return true
}

const customUpload = async (options) => {
    const { file, onProgress, onSuccess, onError } = options
    
    try {
        const params = new FormData()
        params.append('files', file)
        params.append('target_type', 'article')

        const { code, msg, data } = await request.post('/api/attachment/batch', params, {
            onUploadProgress: (event) => {
                const percent = Math.round((event.loaded / event.total) * 100)
                state.progress.value = percent
                onProgress({ percent })
            }
        })

        if (code !== 200) {
            throw new Error(msg)
        }

        const result = data.results?.[0]
        if (!result?.full_url) {
            throw new Error('上传失败，未获取到图片链接')
        }

        previewImage.value = result.full_url
        const imageMarkdown = `![${file.name}](${result.full_url})`
        insertText(imageMarkdown)
        ElMessage.success('图片上传成功')
        showImageModal.value = false
        resetImageModal()
        onSuccess(result)
    } catch (error) {
        ElMessage.error(error.message || '上传失败')
        onError(error)
    } finally {
        uploadingImage.value = false
        state.progress.show = false
    }
}

const insertImageFromUrl = () => {
    const url = imageUrl.value.trim()
    if (utils.is.empty(url)) {
        ElMessage.warning('请输入图片链接')
        return
    }

    if (!utils.is.url(url)) {
        ElMessage.warning('请输入有效的图片链接')
        return
    }

    previewImage.value = url
    const imageMarkdown = `![图片](${url})`
    insertText(imageMarkdown)
    ElMessage.success('图片插入成功')
    showImageModal.value = false
    resetImageModal()
}

const resetImageModal = () => {
    imageUrl.value = ''
    previewImage.value = ''
    uploadingImage.value = false
    state.progress.show = false
}

const handlePaste = async (e) => {
    const items = e.clipboardData?.items
    if (!items) return

    for (const item of items) {
        if (item.type.startsWith('image/')) {
            e.preventDefault()
            const file = item.getAsFile()
            if (!file) continue

            const isValid = await beforeUpload(file)
            if (!isValid) continue

            uploadingImage.value = true
            state.progress.show = true

            try {
                const params = new FormData()
                params.append('files', file)
                params.append('target_type', 'article')

                const { code, msg, data } = await request.post('/api/attachment/batch', params, {
                    onUploadProgress: (event) => {
                        state.progress.value = Math.round((event.loaded / event.total) * 100)
                    }
                })

                if (code !== 200) {
                    throw new Error(msg)
                }

                const result = data.results?.[0]
                if (!result?.full_url) {
                    throw new Error('上传失败')
                }

                const imageMarkdown = `![${file.name}](${result.full_url})`
                insertText(imageMarkdown)
                ElMessage.success('图片粘贴上传成功')
            } catch (error) {
                ElMessage.error(error.message || '粘贴上传失败')
            } finally {
                uploadingImage.value = false
                state.progress.show = false
            }
        }
    }
}

const handleDrop = async (e) => {
    const files = e.dataTransfer?.files
    if (!files || files.length === 0) return

    e.preventDefault()

    for (const file of files) {
        if (!file.type.startsWith('image/')) {
            ElMessage.warning(`${file.name} 不是图片文件，已跳过`)
            continue
        }

        const isValid = await beforeUpload(file)
        if (!isValid) continue

        uploadingImage.value = true
        state.progress.show = true

        try {
            const params = new FormData()
            params.append('files', file)
            params.append('target_type', 'article')

            const { code, msg, data } = await request.post('/api/attachment/batch', params, {
                onUploadProgress: (event) => {
                    state.progress.value = Math.round((event.loaded / event.total) * 100)
                }
            })

            if (code !== 200) {
                throw new Error(msg)
            }

            const result = data.results?.[0]
            if (!result?.full_url) {
                throw new Error('上传失败')
            }

            const imageMarkdown = `![${file.name}](${result.full_url})`
            insertText(imageMarkdown)
            ElMessage.success(`${file.name} 上传成功`)
        } catch (error) {
            ElMessage.error(`${file.name} 上传失败: ${error.message}`)
        } finally {
            uploadingImage.value = false
            state.progress.show = false
        }
    }
}

const toggleFullscreen = () => {
    isFullscreen.value = !isFullscreen.value
    
    if (isFullscreen.value) {
        document.documentElement.classList.add('md-editor-fullscreen-active')
        document.body.style.overflow = 'hidden'
    } else {
        document.documentElement.classList.remove('md-editor-fullscreen-active')
        document.body.style.overflow = ''
    }
}

const handleKeydownGlobal = (e) => {
    if (e.key === 'Escape' && isFullscreen.value) {
        toggleFullscreen()
    }
}

defineExpose({
    getValue: () => state.content,
    setValue: (value) => {
        state.content = value
        emit('update:modelValue', value)
    },
    insertText,
    insertFormat,
    clear: () => {
        state.content = ''
        emit('update:modelValue', '')
    },
    togglePreview: () => {
        showPreview.value = !showPreview.value
    }
})

onMounted(() => {
    if (state.history.length === 0) {
        state.history.push(state.content)
        state.historyIndex = 0
    }
    document.addEventListener('keydown', handleKeydownGlobal)
    
    handleResize()
    window.addEventListener('resize', handleResize)
    
    if (containerRef.value) {
        resizeObserver = new ResizeObserver(handleResize)
        resizeObserver.observe(containerRef.value.parentElement)
    }
})

onUnmounted(() => {
    document.removeEventListener('keydown', handleKeydownGlobal)
    window.removeEventListener('resize', handleResize)
    
    if (resizeObserver) {
        resizeObserver.disconnect()
    }
    
    if (saveHistoryTimer) {
        clearTimeout(saveHistoryTimer)
    }
    
    if (isFullscreen.value) {
        document.documentElement.classList.remove('md-editor-fullscreen-active')
        document.body.style.overflow = ''
    }
})
</script>

<style scoped>
.md-editor {
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    overflow: hidden;
    background: var(--el-bg-color);
    position: relative;
    transition: all 0.3s ease;
}

.md-editor.fullscreen {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9999;
    border-radius: 0;
    border: none;
}

.md-toolbar {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color);
    flex-wrap: wrap;
    gap: 4px;
}

.toolbar-group {
    display: flex;
    gap: 2px;
}

.toolbar-text-btn {
    font-weight: 600;
    font-size: 12px;
}

.toolbar-divider {
    width: 1px;
    height: 24px;
    background: var(--el-border-color);
    margin: 0 4px;
}

.toolbar-spacer {
    flex: 1;
}

.md-body {
    display: flex;
    min-height: 400px;
}

.md-body.fullscreen {
    height: calc(100vh - 52px);
}

.md-edit-area {
    flex: 1;
    display: flex;
    height: 100%;
    position: relative;
}

.md-textarea {
    width: 100%;
    height: 100%;
    padding: 16px;
    padding-left: 50px;
    border: none;
    resize: none;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 14px;
    line-height: 1.8;
    outline: none;
    background: transparent;
    color: var(--el-text-color-primary);
    tab-size: 2;
}

.md-textarea::placeholder {
    color: var(--el-text-color-regular);
}

.md-line-numbers {
    position: absolute;
    left: 0;
    top: 0;
    bottom: 0;
    width: 40px;
    padding: 16px 0;
    text-align: center;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 14px;
    line-height: 1.8;
    color: var(--el-text-color-regular);
    background: var(--el-fill-color-light);
    border-right: 1px solid var(--el-border-color);
    user-select: none;
    overflow: hidden;
    pointer-events: none;
}

.md-line-numbers span {
    display: block;
}

.md-preview-area {
    width: 50%;
    border-left: 1px solid var(--el-border-color);
    overflow-y: auto;
    background: var(--el-fill-color-light);
    height: 100%;
}

.md-preview-area .article-content {
    padding: 16px;
}

.md-preview-area :deep(.article-content) {
    font-size: 15px;
    line-height: 1.8;
}

.md-preview-area :deep(.article-content h1) {
    font-size: 2em;
    margin: 1.5em 0 0.5em;
    border-bottom: 2px solid var(--el-border-color);
    padding-bottom: 0.3em;
}

.md-preview-area :deep(.article-content h2) {
    font-size: 1.5em;
    margin: 1.2em 0 0.4em;
    border-bottom: 1px solid var(--el-border-color);
    padding-bottom: 0.3em;
}

.md-preview-area :deep(.article-content h3) {
    font-size: 1.25em;
    margin: 1em 0 0.3em;
}

.md-preview-area :deep(.article-content h4) {
    font-size: 1.1em;
    margin: 0.8em 0 0.2em;
}

.md-preview-area :deep(.article-content p) {
    margin: 0 0 0.8em;
}

.md-preview-area :deep(.article-content ul),
.md-preview-area :deep(.article-content ol) {
    padding-left: 2em;
    margin: 0 0 0.8em;
}

.md-preview-area :deep(.article-content li) {
    margin: 0.2em 0;
}

.md-preview-area :deep(.article-content blockquote) {
    margin: 0 0 0.8em;
    padding: 0.5em 1em;
    border-left: 4px solid var(--el-color-primary);
    background: var(--el-fill-color);
    border-radius: 0 4px 4px 0;
}

.md-preview-area :deep(.article-content pre) {
    margin: 0 0 0.8em;
    border-radius: 4px;
    padding: 1em;
    overflow-x: auto;
    background: #282c34;
}

.md-preview-area :deep(.article-content pre code) {
    color: #e0e0e0;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 13px;
    line-height: 1.6;
}

.md-preview-area :deep(.article-content code) {
    background-color: var(--el-fill-color);
    padding: 0.15rem 0.3rem;
    border-radius: 0.25rem;
    font-size: 0.9em;
    color: var(--el-text-color-primary);
    font-family: 'Consolas', 'Monaco', monospace;
}

.md-preview-area :deep(.article-content table) {
    width: 100%;
    border-collapse: collapse;
    margin: 0 0 0.8em;
}

.md-preview-area :deep(.article-content table td) {
    border: 1px solid var(--el-border-color);
    padding: 0.5em 0.8em;
}

.md-preview-area :deep(.article-content table tr:nth-child(even)) {
    background: var(--el-fill-color);
}

.md-preview-area :deep(.article-content img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
    margin: 0.5em 0;
}

.md-preview-area :deep(.article-content hr) {
    margin: 1.5em 0;
    border: none;
    border-top: 1px solid var(--el-border-color);
}

.md-preview-area :deep(.article-content details) {
    margin: 0.5em 0;
    padding: 0.5em 1em;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
}

.md-preview-area :deep(.article-content details summary) {
    cursor: pointer;
    font-weight: 600;
}

.img-preview {
    max-width: 100%;
    max-height: 200px;
    border-radius: 4px;
}

:deep(.el-dialog__header) {
    border-bottom: 1px solid var(--el-border-color);
}

:deep(.el-dialog__footer) {
    border-top: 1px solid var(--el-border-color);
}

:deep(.el-button--text) {
    width: 32px;
    height: 32px;
    padding: 0;
    border-radius: 4px;
    transition: all 0.2s;
}

:deep(.el-button--text:hover) {
    background: var(--el-fill-color);
}

:deep(.el-button--text.active) {
    background: var(--el-color-primary);
    color: #fff;
}

@media (max-width: 768px) {
    .md-body {
        flex-direction: column;
    }

    .md-edit-area,
    .md-preview-area {
        width: 100% !important;
    }

    .md-preview-area {
        border-left: none;
        border-top: 1px solid var(--el-border-color);
        max-height: 300px;
    }

    :deep(.el-button--text) {
        width: 28px;
        height: 28px;
        font-size: 12px;
    }
}
</style>