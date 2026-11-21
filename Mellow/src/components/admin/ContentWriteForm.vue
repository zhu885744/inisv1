<template>
  <div class="write-page">
    <!-- 顶部返回 -->
    <div class="write-top">
      <router-link :to="basePath" class="back-link">
        <i class="bi bi-arrow-left" /> 返回列表
      </router-link>
      <span class="write-mode">{{ modeLabel }}</span>
    </div>

    <!-- 编辑卡片 -->
    <div class="card editor-card">
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <template v-else>
        <!-- 标题 -->
        <div class="title-row">
          <FormInput
            v-model="form.title"
            class="title-input"
            type="text"
            maxlength="50"
            placeholder="输入长文标题" />
          <span class="title-count" :class="{ 'is-over': form.title.length >= titleLimit }">
            {{ form.title.length }}/{{ titleLimit }}
          </span>
        </div>
        <p v-if="error.title" class="form-error">{{ error.title }}</p>

        <!-- 正文编辑器（工具栏 + 正文 + 字数） -->
        <MarkdownEditor
          ref="editorRef"
          v-model="form.content"
          placeholder="来说点什么吧..."
          min-height="420px"
          :max-length="contentLimit"
          @save="submit(1)"
        />
        <p v-if="error.content" class="form-error">{{ error.content }}</p>

        <!-- 更多设置 -->
        <div v-show="showExtra" class="extra-panel">
          <!-- 封面（置顶，单独一行；仅文章） -->
          <div v-if="!isPage" class="form-item">
            <label class="form-label">封面</label>
            <div class="cover-box">
              <div v-if="form.cover" class="cover-preview">
                <img :src="form.cover" alt="封面预览" loading="lazy" />
                <button type="button" class="cover-del" title="移除封面" @click="form.cover = ''">
                  <i class="bi bi-x" />
                </button>
              </div>
              <div v-else class="cover-empty" @click="pickCover">
                <i class="bi bi-image" />
                <span>点击上传</span>
              </div>
              <div class="cover-url">
                <FormInput
                  v-model="form.cover"
                  type="text"
                  placeholder="或直接粘贴图片链接" />
                <button type="button" class="btn btn-ghost btn-sm" :disabled="uploading" @click="pickCover">
                  {{ uploading ? '上传中...' : '上传' }}
                </button>
                <button type="button" class="btn btn-ghost btn-sm" title="从附件库选择封面" @click="showCoverLibrary = true">
                  附件库
                </button>
              </div>
            </div>
            <input ref="coverFileRef" type="file" accept="image/*" hidden @change="onCoverChange" />

            <!-- 附件库（选择已上传图片作为封面） -->
            <AttachmentLibrary
              v-model:visible="showCoverLibrary"
              title="选择封面图片"
              accept="image"
              @select="onCoverLibrarySelect"
            />
          </div>

          <!-- 摘要（仅文章） -->
          <div v-if="!isPage" class="form-item">
            <label class="form-label">摘要</label>
            <FormTextarea
              v-model="form.abstract"
              rows="3"
              maxlength="200"
              placeholder="请勿留空哦" />
          </div>

          <!-- 唯一标识（仅独立页面）：决定前台访问路径 /{key} -->
          <div v-if="isPage" class="form-item">
            <label class="form-label">唯一标识</label>
            <FormInput
              v-model="form.key"
              type="text"
              maxlength="64"
              placeholder="如 about，前台访问路径即为 /about" />
            <p class="form-hint">必填，仅支持字母、数字、下划线、中划线；需全站唯一，前台访问路径为 /{key}</p>
          </div>

          <!-- 备注（仅独立页面，不在前台展示） -->
          <div v-if="isPage" class="form-item">
            <label class="form-label">备注</label>
            <FormInput
              v-model="form.remark"
              type="text"
              maxlength="200"
              placeholder="仅后台可见，用于标记页面用途" />
          </div>

          <div class="form-item">
            <label class="form-label">发布时间</label>
            <FormInput
              v-model="publishAt"
              type="datetime-local"
              @input="publishAtTouched = true" />
          </div>

          <!-- 作者：管理员编辑他人文章 / 独立页面时能看到这条是谁的（与审核状态同一区块） -->
          <div v-if="isAdminUser && isEdit" class="form-item">
            <OwnerInfo :uid="ownerUid" label="作者" />
          </div>

          <!-- 审核状态（仅管理员可见/可改） -->
          <div v-if="isAdminUser" class="form-item">
            <label class="form-label">审核状态</label>
            <FormSelect v-model="articleAudit" :options="auditOptions" />
            <p class="form-hint">
              仅管理员可设置：「通过」后{{ isPage ? '该页面对访客可见' : '文章对访客可见' }}；普通作者保存不会覆盖该状态
            </p>
          </div>

          <!-- 驳回原因（仅管理员、且选「不通过」时出现，作者会收到通知） -->
          <div v-if="isAdminUser && Number(articleAudit) === 2" class="form-item">
            <label class="form-label">驳回原因</label>
            <FormTextarea
              v-model="articleReason"
              rows="2"
              maxlength="512"
              :placeholder="`说明不通过的原因，作者会在通知与「我的${isPage ? '页面' : '文章'}」里看到`" />
          </div>

          <!-- 评论设置（json.comment：allow 允许/禁止，show 显示/隐藏） -->
          <div class="form-item">
            <label class="form-label">允许评论</label>
            <FormSelect
              v-model="commentAllow"
              :options="commentAllowOptions" />
            <p class="form-hint">选「禁止」后仍展示已有评论，只是关闭发表与回复入口</p>
          </div>

          <div class="form-item">
            <label class="form-label">显示评论</label>
            <FormSelect
              v-model="commentShow"
              :options="commentShowOptions" />
            <p class="form-hint">选「隐藏」后{{ isPage ? '该页面' : '文章详情页' }}整个评论模块都不显示</p>
          </div>

          <!-- 可见范围：1 公开 / 2 登录可见 / 3 付费可见（积分兑换），用单选框平铺 -->
          <div class="form-item">
            <label class="form-label">可见范围</label>
            <ScopeRadio v-model="scope" :options="scopeOptions" />
            <p class="form-hint">
              公开：所有用户（含访客）可见；登录可见：访客只看到登录提示；付费可见：登录用户消耗积分兑换后可看全文
            </p>
          </div>

          <!-- 付费可见的兑换积分（1 积分起，兑换所得归作者） -->
          <div v-if="Number(scope) === 3" class="form-item">
            <label class="form-label">兑换积分</label>
            <FormInput
              v-model.number="scopeIntegral"
              type="number"
              min="1"
              max="999999"
              placeholder="查看全文需消耗的积分，如 10" />
            <p class="form-hint">同一用户只需兑换一次，兑换所得积分归内容作者</p>
          </div>

          <div class="form-item">
            <label class="form-label">话题</label>
            <div class="tag-picker">
              <span
                v-for="t in selectedTags"
                :key="t.id"
                class="tag tag-primary tag-removable"
                @click="toggleTag(t)"
              >
                {{ t.name }} <i class="bi bi-x" />
              </span>
              <span
                v-for="t in unselectedTags"
                :key="'u' + t.id"
                class="tag"
                @click="toggleTag(t)"
              >{{ t.name }}</span>
              <span class="tag-add">
                <FormInput
                  v-model="newTag"
                  class="tag-input"
                  type="text"
                  placeholder="+ 新话题"
                  maxlength="16"
                  @keyup.enter="addTag" />
              </span>
            </div>
          </div>
        </div>

        <!-- 底部操作栏 -->
        <div class="action-bar">
          <div class="action-left">
            <!-- 选择分类（仅文章） -->
            <FormSelect variant="pill"
              v-if="!isPage"
              v-model="form.group"
              :options="groupSelectOptions"
              placeholder="选择分类"
              icon="bi bi-collection" />

            <!-- 更多设置 -->
            <button
              type="button"
              class="pill"
              :class="{ 'is-active': showExtra }"
              @click="showExtra = !showExtra"
            >
              <i class="bi bi-sliders" aria-hidden="true" />
              更多设置
              <i :class="showExtra ? 'bi bi-chevron-up caret' : 'bi bi-chevron-down caret'" aria-hidden="true" />
            </button>
          </div>

          <div class="action-right">
            <span class="save-tip">{{ savedTip }}</span>
            <!-- 独立页面没有草稿/发布之分，仅保留一个保存按钮 -->
            <!-- 与右侧主按钮 / 左侧「更多设置」统一为全圆角（.btn-pill） -->
            <button v-if="!isPage" class="btn btn-ghost btn-pill" :disabled="saving" @click="submit(0)">
              {{ saving ? '处理中...' : '保存草稿' }}
            </button>
            <button class="btn btn-primary btn-pill" :disabled="saving" @click="submit(1)">
              {{ primaryText }}
            </button>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 内容编辑器（可复用）—— 同时服务于「文章」与「独立页面」
 *
 * 使用方：
 *   - 创作中心 /manage/posts/write、/manage/posts/edit/:id（mode=article，仅能编辑自己的文章）
 *   - 管理后台 /admin/article/write、/admin/article/edit/:id（mode=article，管理员可编辑任意文章）
 *   - 管理后台 /admin/pages/write、/admin/pages/edit/:id（mode=page）
 *
 * 两者共用同一套编辑器 / 封面 / 话题 / 评论开关 / 审核状态逻辑，字段差异由 mode 决定：
 *   - article：摘要、封面、分类，有草稿与发布（status）
 *   - page  ：唯一标识（key）、备注（remark），无草稿概念
 * 通过 props 注入差异，避免维护两份几乎相同的页面代码。
 */
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter, onBeforeRouteLeave } from 'vue-router'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import ScopeRadio from '@/components/ScopeRadio.vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import OwnerInfo from '@/components/admin/OwnerInfo.vue'
import {
  createArticle,
  updateArticle,
  getArticleForEdit,
  getArticleGroups,
  uploadArticleImage
} from '@/api/article'
import {
  createPage,
  updatePage,
  getPageForEdit
} from '@/api/pages'
import { listAllTags, createTag } from '@/api/tags'
import { useUserStore } from '@/stores/user'
import { storeToRefs } from 'pinia'
import { parseIdField, toIdField, truncate, isAdmin, stripArticleMarkup } from '@/utils/helper'

/**
 * 自动摘要：先把正文里的标记（媒体短代码 / 图片 / 表情）剥掉再截断
 *
 * 否则摘要会直接显示 @[live](https://.../a.mp4 "封面") / @[video](...) 这种源码
 * —— 列表页不渲染 markdown。媒体短代码（含实况图 live）统一由
 * stripArticleMarkup 清理，这里只额外去掉表情标记：摘要字段有长度限制，
 * [emoji:url] 一个就占掉几十个字符。
 */
function autoAbstract(content) {
  const plain = stripArticleMarkup(content).replace(/\[emoji:[^\]]+\]/g, ' ')
  return truncate(plain.replace(/\s+/g, ' ').trim(), 120)
}
import { toLocalInput } from '@/utils/time'
import { toast } from '@/utils/toast'

const props = defineProps({
  // 内容类型：article 文章 / page 独立页面
  mode: { type: String, default: 'article' },
  // 列表页路径：返回链接、加载失败兜底跳转的基准
  basePath: { type: String, default: '/manage/posts' },
  // 编辑页路径前缀：默认取 `${basePath}/edit`
  editBase: { type: String, default: '' },
  // 是否允许编辑他人内容（管理后台为 true）
  canEditOthers: { type: Boolean, default: false }
})

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { user } = storeToRefs(userStore)

// 是否为「独立页面」模式
const isPage = computed(() => props.mode === 'page')
// 顶部模式文案
const modeLabel = computed(() => `${isEdit.value ? '编辑' : '撰写'}${isPage.value ? '页面' : '文章'}`)

const editPrefix = computed(() => props.editBase || `${props.basePath}/edit`)

const isEdit = computed(() => !!route.params.id)
// 当前内容 ID（文章 / 页面通用）
const contentId = computed(() => Number(route.params.id) || 0)

// 标题 / 正文字数展示上限（与编辑器右下角计数一致）
const titleLimit = 50
const contentLimit = 9999

const loading = ref(false)
const saving = ref(false)
// 主按钮文案（独立页面无草稿概念，仅「保存」）
const primaryText = computed(() => {
  if (saving.value) return '处理中...'
  if (isPage.value) return '保存'
  return isEdit.value ? '保存修改' : '发布'
})
const uploading = ref(false)
const savedTip = ref('')
const newTag = ref('')
// 发布时间：默认预填当前系统时间
const publishAt = ref(toLocalInput(Math.floor(Date.now() / 1000)))
// 用户是否手动改过发布时间（决定保存时用所选时间还是实时时间）
const publishAtTouched = ref(false)

// 更多设置面板 / 评论开关（json.comment，选项与 admin 端一致）
// allow：0 继承总开关 1 允许 2 禁止（禁止仍展示已有评论，只关掉发表与回复入口）
// show：0 继承总开关 1 显示 2 隐藏（隐藏则整个评论模块不显示）
// 总开关见「系统配置 → 文章配置 / 动态配置 / 独立页面配置」，设为禁止或隐藏时覆盖此处
const showExtra = ref(false)
const commentAllow = ref(0)
const commentShow = ref(0)

const commentAllowOptions = [
  { value: 0, label: '继承总开关' },
  { value: 1, label: '允许' },
  { value: 2, label: '禁止' }
]
const commentShowOptions = [
  { value: 0, label: '继承总开关' },
  { value: 1, label: '显示' },
  { value: 2, label: '隐藏' }
]

// 可见范围（后端字段 scope）：1 公开 2 登录可见 3 付费可见；
// 付费可见时 scope_integral 为兑换所需积分
const scope = ref(1)
const scopeIntegral = ref(0)
const scopeOptions = [
  { value: 1, label: '公开（所有人可见）' },
  { value: 2, label: '登录可见' },
  { value: 3, label: '付费可见（积分兑换）' }
]

// 审核状态（仅管理员可见/可改）：0 待审核 1 通过 2 不通过
const isAdminUser = computed(() => isAdmin(user.value))
const articleAudit = ref(1)
// 驳回原因（仅管理员可写）：选「不通过」时填，作者会在通知与「我的文章 / 页面」里看到
const articleReason = ref('')
// 作者 uid：仅用于管理员编辑他人内容时展示「作者：昵称」，不参与提交
const ownerUid = ref(0)
const auditOptions = [
  { value: 0, label: '待审核' },
  { value: 1, label: '通过' },
  { value: 2, label: '不通过' }
]
// 原文 json，保存时合并，避免覆盖其它自定义配置
const articleJson = ref({})

const coverFileRef = ref(null)
const editorRef = ref(null)
const groups = ref([])
const tags = ref([])
// 原文的发布时间（0 表示未设置，用于草稿首次发布时补上当前时间）
const originalPublishTime = ref(0)

const form = reactive({
  title: '',
  abstract: '',
  content: '',
  cover: '',
  group: '',
  tagIds: [],
  // 以下仅独立页面使用
  key: '',
  remark: ''
})

const error = reactive({ title: '', content: '' })

const groupOptions = computed(() => {
  // 依据 pid 生成带缩进的展示名
  const list = groups.value || []
  const build = (pid, level) => {
    return list
      .filter((g) => Number(g.pid || 0) === Number(pid))
      .flatMap((g) => [
        { id: g.id, label: `${'　'.repeat(level)}${level ? '└ ' : ''}${g.name}` },
        ...build(g.id, level + 1)
      ])
  }
  return build(0, 0)
})

// 下拉选项：首项为「不选择分类」，其余为带缩进的板块树
const groupSelectOptions = computed(() => [
  { value: '', label: '不选择分类' },
  ...groupOptions.value.map((g) => ({ value: g.id, label: g.label }))
])

const selectedTags = computed(() => tags.value.filter((t) => form.tagIds.includes(t.id)))
const unselectedTags = computed(() => tags.value.filter((t) => !form.tagIds.includes(t.id)).slice(0, 30))

// 是否有未保存内容（用于离开提醒）
const dirty = ref(false)
watch(
  () => [
    form.title,
    form.abstract,
    form.content,
    form.cover,
    form.group,
    form.key,
    form.remark,
    form.tagIds.join(','),
    publishAt.value,
    commentAllow.value,
    commentShow.value,
    scope.value,
    scopeIntegral.value,
    articleAudit.value
  ],
  () => {
    if (loading.value) return
    dirty.value = true
    savedTip.value = '有未保存的修改'
  }
)

function toggleTag(t) {
  const idx = form.tagIds.indexOf(t.id)
  if (idx > -1) form.tagIds.splice(idx, 1)
  else form.tagIds.push(t.id)
}

async function addTag() {
  const name = newTag.value.trim()
  if (!name) return
  const exist = tags.value.find((t) => t.name === name)
  if (exist) {
    if (!form.tagIds.includes(exist.id)) form.tagIds.push(exist.id)
    newTag.value = ''
    return
  }
  try {
    const res = await createTag(name)
    const id = res.data?.id
    if (!id) throw new Error('no id')
    tags.value.push({ id, name })
    form.tagIds.push(id)
    newTag.value = ''
    toast.success('话题已创建')
  } catch {
    toast.error('话题创建失败')
  }
}

function pickCover() {
  coverFileRef.value?.click()
}

// 附件库：选择已上传图片作为封面
const showCoverLibrary = ref(false)
function onCoverLibrarySelect(urls = []) {
  const url = urls[0]
  if (!url) return
  form.cover = url
  toast.success('封面已应用')
}

async function onCoverChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    toast.warning('图片不能超过 10MB')
    return
  }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadArticleImage(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')
    form.cover = url
    toast.success('封面已上传')
  } catch {
    toast.error('封面上传失败')
  } finally {
    uploading.value = false
  }
}

function validate() {
  error.title = form.title.trim() ? '' : '标题不能为空'
  error.content = form.content.trim() ? '' : '正文不能为空'
  return !error.title && !error.content
}

function buildPayload(status) {
  // 评论开关：allow 1 允许 2 禁止 / show 1 显示 2 不显示
  // （合并原有 json.comment，避免覆盖其它配置）
  const json = {
    ...(articleJson.value || {}),
    comment: {
      ...((articleJson.value || {}).comment || {}),
      allow: commentAllow.value,
      show: commentShow.value
    }
  }

  // 可见范围：scope 1 公开 / 2 登录可见 / 3 付费可见，非付费内容不需要积分
  const scopeValue = [1, 2, 3].includes(Number(scope.value)) ? Number(scope.value) : 1
  const scopeIntegralValue = scopeValue === 3 ? Math.max(1, Number(scopeIntegral.value) || 0) : 0

  // 独立页面：无摘要 / 封面 / 分类 / 草稿，字段为 key + title + content + remark + tags
  if (isPage.value) {
    const pagePayload = {
      key: form.key.trim(),
      title: form.title.trim(),
      content: form.content,
      remark: form.remark.trim(),
      tags: toIdField(form.tagIds),
      editor: 'markdown',
      scope: scopeValue,
      scope_integral: scopeIntegralValue,
      json
    }
    const publishTime = resolvePublishTime(1)
    if (publishTime) pagePayload.publish_time = publishTime
    // 审核状态仅管理员可写（后端 root 才允许提交 audit 字段）
    if (isAdminUser.value) {
      pagePayload.audit = articleAudit.value
      // 只有「不通过」才保留原因，其余状态清空，避免作者看到过期原因
      pagePayload.reason = Number(articleAudit.value) === 2 ? articleReason.value.trim() : ''
    }
    return pagePayload
  }

  const payload = {
    title: form.title.trim(),
    abstract: form.abstract.trim() || autoAbstract(form.content),
    content: form.content,
    covers: form.cover,
    group: toIdField(form.group === '' ? [] : [form.group]),
    tags: toIdField(form.tagIds),
    editor: 'markdown',
    status,
    scope: scopeValue,
    scope_integral: scopeIntegralValue,
    json
  }
  const publishTime = resolvePublishTime(status)
  if (publishTime) payload.publish_time = publishTime
  // 审核状态仅管理员可写（后端 root 才允许提交 audit 字段）
  if (isAdminUser.value) {
    payload.audit = articleAudit.value
    // 只有「不通过」才保留原因，其余状态清空，避免作者看到过期原因
    payload.reason = Number(articleAudit.value) === 2 ? articleReason.value.trim() : ''
  }
  return payload
}

/**
 * 计算实际写入的发布时间
 * - 用户手动改过 → 用所选时间
 * - 编辑已有文章（原本已有发布时间）→ 沿用原值
 * - 其它（新文章 / 还没发布过的草稿）→ 发布时取当前系统时间，存草稿不写入
 */
function resolvePublishTime(status) {
  if (publishAtTouched.value || originalPublishTime.value) {
    const ts = publishAt.value ? Math.floor(new Date(publishAt.value).getTime() / 1000) : 0
    return Number.isNaN(ts) ? 0 : ts
  }
  return status === 1 ? Math.floor(Date.now() / 1000) : 0
}

async function submit(status) {
  if (saving.value) return
  if (!validate()) {
    toast.warning(error.title || error.content)
    return
  }
  // 独立页面的唯一标识为后端必填项（validator: required,alphaDash），这里同步校验
  if (isPage.value) {
    const key = form.key.trim()
    if (!key) {
      toast.warning('请填写唯一标识')
      return
    }
    if (!/^[A-Za-z0-9_-]+$/.test(key)) {
      toast.warning('唯一标识只能由字母、数字、- 和 _ 组成')
      return
    }
  }
  // 独立页面没有草稿概念，统一按发布处理（status 仅用于推导发布时间）
  const effectiveStatus = isPage.value ? 1 : status
  saving.value = true
  try {
    const payload = buildPayload(effectiveStatus)
    let id = contentId.value
    // 新建时后端回传实际审核状态：站点关闭「内容审核」时 audit=1（已通过），据此给正确提示
    let createdAudit = null

    if (isPage.value) {
      if (isEdit.value) {
        await updatePage({ id, ...payload })
      } else {
        const res = await createPage(payload)
        id = res.data?.id || 0
        createdAudit = Number(res.data?.audit)
      }
    } else if (isEdit.value) {
      await updateArticle({ id, ...payload })
    } else {
      const res = await createArticle(payload)
      id = res.data?.id || 0
      createdAudit = Number(res.data?.audit)
    }

    // 回填实际生效的发布时间，避免界面显示与存储不一致
    if (payload.publish_time) {
      originalPublishTime.value = payload.publish_time
      publishAt.value = toLocalInput(payload.publish_time)
      publishAtTouched.value = false
    }
    dirty.value = false
    // 已落库：清掉编辑器里的本地草稿，避免下次打开误提示「发现本地草稿」
    editorRef.value?.clearDraft?.()

    if (isPage.value) {
      savedTip.value = '已保存'
      toast.success(
        isEdit.value ? '修改已保存' : createdAudit === 0 ? '页面已创建，等待审核通过后对外可见' : '页面已创建'
      )
    } else {
      savedTip.value = status === 1 ? '已保存' : '草稿已保存'
      toast.success(
        status === 1
          ? isEdit.value
            ? '修改已保存'
            : createdAudit === 0
              ? '发布成功，等待审核通过后对外可见'
              : '发布成功'
          : '草稿已保存'
      )
    }

    if (!isEdit.value && id) {
      router.replace(`${editPrefix.value}/${id}`)
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    saving.value = false
  }
}

// 分类字典仅文章需要；话题两者共用
async function loadMeta() {
  if (!isPage.value) {
    try {
      const res = await getArticleGroups()
      groups.value = res.data?.data || []
    } catch {
      groups.value = []
    }
  }
  try {
    const res = await listAllTags()
    tags.value = (res.data?.data || []).map((t) => ({ id: t.id, name: t.name }))
  } catch {
    tags.value = []
  }
}

async function loadContent() {
  if (!isEdit.value) return
  loading.value = true
  try {
    const res = isPage.value
      ? await getPageForEdit(contentId.value)
      : await getArticleForEdit(contentId.value)
    const item = res.data
    if (!item) {
      toast.error(isPage.value ? '页面不存在或已被删除' : '文章不存在或已被删除')
      router.replace(props.basePath)
      return
    }
    // 非管理员只能编辑自己的内容（管理后台可通过 canEditOthers 放开）
    const mine = Number(item.uid) === Number(user.value?.id)
    if (!mine && !props.canEditOthers) {
      toast.error(isPage.value ? '只能编辑自己的页面' : '只能编辑自己的文章')
      router.replace(props.basePath)
      return
    }

    form.title = item.title || ''
    form.content = item.content || ''
    form.tagIds = parseIdField(item.tags)
    originalPublishTime.value = Number(item.publish_time) || 0

    if (isPage.value) {
      form.key = item.key || ''
      form.remark = item.remark || ''
    } else {
      form.abstract = item.abstract || ''
      const covers = Array.isArray(item.covers) ? item.covers[0] : String(item.covers || '').split(',')[0]
      form.cover = covers || ''
      const groupIds = parseIdField(item.group)
      form.group = groupIds.length ? groupIds[0] : ''
    }

    // 评论开关：0 继承 / 1 允许（显示）/ 2 禁止（隐藏），缺失视为继承
    articleJson.value = item.json && typeof item.json === 'object' ? item.json : {}
    const allow = Number(articleJson.value?.comment?.allow)
    commentAllow.value = [0, 1, 2].includes(allow) ? allow : 0
    const show = Number(articleJson.value?.comment?.show)
    commentShow.value = [0, 1, 2].includes(show) ? show : 0

    // 可见范围：1 公开 / 2 登录可见 / 3 付费可见（历史数据无该字段时按公开处理）
    const itemScope = Number(item.scope)
    scope.value = [1, 2, 3].includes(itemScope) ? itemScope : 1
    scopeIntegral.value = Number(item.scope_integral) || 0

    // 审核状态（仅管理员会用到，0 待审核 1 通过 2 不通过）
    const audit = Number(item.audit)
    articleAudit.value = [0, 1, 2].includes(audit) ? audit : 1
    articleReason.value = String(item.reason || '')
    // 作者（仅管理员编辑他人内容时展示）
    ownerUid.value = Number(item.uid) || 0

    await loadMeta()
    // 发布时间：已有内容用原值，未设置则默认当前系统时间
    publishAt.value = toLocalInput(originalPublishTime.value || Math.floor(Date.now() / 1000))
    publishAtTouched.value = false
    loading.value = false
    // 数据回填完成后再开启脏值检测
    setTimeout(() => {
      dirty.value = false
      savedTip.value = ''
    }, 0)
    return
  } catch {
    toast.error(isPage.value ? '页面加载失败' : '文章加载失败')
    router.replace(props.basePath)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  if (!isEdit.value) await loadMeta()
  await loadContent()
  window.addEventListener('beforeunload', onBeforeUnload)
})

onUnmounted(() => {
  window.removeEventListener('beforeunload', onBeforeUnload)
})

function onBeforeUnload(e) {
  if (!dirty.value) return
  e.preventDefault()
  e.returnValue = ''
}

onBeforeRouteLeave(() => {
  if (!dirty.value) return true
  return window.confirm('有未保存的修改，确定离开吗？')
})
</script>

<style scoped>
.write-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ---------- 顶部 ---------- */
.write-top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.back-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: var(--text-muted);
  text-decoration: none;
}
.back-link:hover {
  color: var(--primary);
}
.write-mode {
  font-size: 12px;
  color: var(--text-light);
}

/* ---------- 编辑卡片 ---------- */
.editor-card {
  padding: 0;
  overflow: visible;
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.form-error {
  margin: 0;
  padding: 0 18px 10px;
  font-size: 12px;
  color: var(--danger);
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

/* 标题 */
.title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 18px 16px;
}
.title-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 22px;
  font-weight: 500;
  line-height: 1.4;
  color: var(--text);
  font-family: var(--font-sans);
}
.title-input::placeholder {
  color: var(--text-light);
  font-weight: 400;
}
.title-count {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
  font-variant-numeric: tabular-nums;
}
.title-count.is-over {
  color: var(--danger);
}

/* 更多设置面板 */
.extra-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px 18px 20px;
  border-top: 1px dashed var(--border-soft);
}

/* ---------- 底部操作栏 ---------- */
.action-bar {
  position: sticky;
  bottom: 12px;
  z-index: 6;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding: 12px 14px;
  background: var(--bg-card);
  border-top: 1px solid var(--border-soft);
  border-radius: 0 0 var(--radius-lg) var(--radius-lg);
}
.action-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.action-right {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}

/* 胶囊按钮 */
.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  transition: all 0.18s;
}
.pill:hover:not(:disabled) {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.pill:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.pill.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}

.save-tip {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 表单（更多设置内） ---------- */

/* 话题 */
.tag-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
.tag {
  cursor: pointer;
}
.tag-removable {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.tag-add {
  display: inline-flex;
}
.tag-input {
  width: 90px;
  padding: 2px 8px;
  font-size: 12px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  outline: none;
}
.tag-input:focus {
  border-color: var(--primary);
  width: 120px;
}

/* 封面 */
.cover-box {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  flex-wrap: wrap;
}
.cover-preview {
  position: relative;
  width: 160px;
  height: 100px;
  border-radius: var(--radius);
  overflow: hidden;
  border: 1px solid var(--border);
  flex-shrink: 0;
}
.cover-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-del {
  position: absolute;
  right: 4px;
  top: 4px;
  width: 22px;
  height: 22px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
}
.cover-empty {
  width: 160px;
  height: 100px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-light);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
  cursor: pointer;
  transition: all 0.2s;
}
.cover-empty:hover {
  border-color: var(--primary);
  color: var(--primary);
}
.cover-empty .bi {
  font-size: 20px;
}
.cover-url {
  flex: 1;
  min-width: 200px;
  display: flex;
  gap: 8px;
}

@media (max-width: 640px) {
  .title-row {
    padding: 16px 14px 12px;
  }
  .title-input {
    font-size: 18px;
  }
  .extra-panel {
    padding: 14px;
  }
  .form-error {
    padding: 0 14px 8px;
  }
  .action-bar {
    position: static;
    padding: 12px;
  }
  .action-left,
  .action-right {
    width: 100%;
  }
  .action-right {
    justify-content: flex-end;
  }
  .cover-url {
    width: 100%;
  }
}
</style>
