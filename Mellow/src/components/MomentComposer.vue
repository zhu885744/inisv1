<template>
  <div class="composer">
    <!-- 作者 -->
    <div class="composer-head">
      <img v-if="avatar" :src="avatar" class="composer-avatar" alt="" loading="lazy" />
      <i v-else class="bi bi-person-circle composer-avatar-icon" aria-hidden="true" />
      <span class="composer-name">{{ nickname }}</span>
    </div>

    <!-- 工具栏：与文章编辑器（MarkdownEditor）同一套视觉 —— 纯图标按钮 + 分隔线。
         按钮都带 @mousedown.prevent，保证点击时不会丢正文选区（表情按光标位置插入）。 -->
    <div class="composer-toolbar">
      <button type="button" class="composer-tool" title="插入表情" @mousedown.prevent @click="openEmoji">
        <i class="bi bi-emoji-smile" />
      </button>
      <button type="button" class="composer-tool" title="@ 提及用户" @mousedown.prevent @click="openMention">
        <i class="bi bi-at" />
      </button>
      <button
        type="button"
        class="composer-tool"
        title="上传图片"
        :disabled="uploading"
        @mousedown.prevent
        @click="triggerPick"
      >
        <i class="bi" :class="uploading ? 'bi-hourglass-split' : 'bi-image'" />
      </button>
      <button
        type="button"
        class="composer-tool"
        title="从附件库选择已有图片"
        :disabled="uploading"
        @mousedown.prevent
        @click="showLibrary = true"
      >
        <i class="bi bi-folder2-open" />
      </button>

      <button
        type="button"
        class="composer-tool"
        title="插入话题（#话题#，发布后自动创建并聚合）"
        @mousedown.prevent
        @click="insertTopicMark"
      >
        <i class="bi bi-hash" />
      </button>

      <span class="composer-toolbar-split" />
      <span class="composer-count">{{ images.length }}/{{ MAX_IMAGES }}</span>
      <span v-if="uploading" class="composer-count">上传中…</span>
    </div>

    <!-- max-height="none"：输入区随内容长高，滚动交给外层（弹窗 / 页面），
         否则编辑器会在 300px 处自己滚动，弹窗看起来就"不再自适应"了。
         inline-picker：表情面板以文档流展开（撑高容器），弹窗里必须开 —— 否则面板是
         absolute 浮层，会被弹窗的 overflow: hidden 裁掉，且不会让弹窗长高。
         hide-toolbar：内置工具栏由上面那排图标按钮替代 -->
    <EmojiEditor
      ref="editorRef"
      v-model="content"
      placeholder="此刻的想法..."
      max-height="none"
      :inline-picker="inlinePicker"
      hide-toolbar
    />

    <input ref="fileInput" type="file" accept="image/*" multiple class="composer-file" @change="onPick" />

    <!-- 附件库（选择已上传图片加入待发布动态） -->
    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择动态图片"
      accept="image"
      multiple
      :max="MAX_IMAGES"
      @select="onLibrarySelect"
    />

    <!-- 已选图片 -->
    <div v-if="images.length" class="composer-images">
      <div v-for="(img, i) in images" :key="img.image" class="composer-image">
        <img :src="img.image" class="composer-image-img" alt="预览" loading="lazy" />
        <span v-if="img.video" class="composer-image-live">LIVE</span>
        <!-- 设为实况图：本地上传短视频，或从附件库挑一条已上传的视频 -->
        <LivePhotoVideoPicker v-model:video="img.video" :disabled="uploading" />
        <button type="button" class="composer-image-del" title="移除" @click="removeImage(i)">
          <i class="bi bi-x-lg" />
        </button>
      </div>
    </div>

    <!-- 所在位置（可选） -->
    <div class="composer-loc">
      <i class="bi bi-geo-alt" />
      <FormInput
        v-model="location"
        class="composer-loc-input"
        type="text"
        maxlength="64"
        placeholder="所在位置（可选）"
        @keyup.enter="submit" />
    </div>

    <!-- 可见范围：公开 / 登录可见 / 付费可见（积分兑换），用单选框平铺 -->
    <div class="composer-scope">
      <span class="composer-scope-label">
        <i class="bi bi-eye" aria-hidden="true" />
        可见范围
      </span>
      <ScopeRadio v-model="scope" :options="scopeOptions" />
      <FormInput
        v-if="Number(scope) === 3"
        v-model.number="scopeIntegral"
        class="composer-scope-input"
        type="number"
        min="1"
        max="999999"
        placeholder="查看需消耗的积分" />
    </div>

    <div class="composer-actions">
      <button class="btn btn-primary btn-sm btn-block" :disabled="publishing || uploading" @click="submit">
        {{ publishing ? '发布中...' : '发布' }}
      </button>
      <span class="composer-tip">
        「用 <code>#话题#</code> 参与话题（发布后自动创建话题，点击可看该话题下的内容）；若站点开启了内容审核，发布后需管理员审核通过才对外显示」
      </span>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * 动态发布器（**唯一实现**）
 *
 * 唯一调用方是 `components/MomentComposeDialog.vue`（发布动态弹窗，首页 / 手机端底栏 / 「我的」
 * 三处入口都用它）。动态列表页（原 `views/moments/Index.vue`）已下线，不要再为某个页面另写一份表单。
 *
 * 接口与参数口径：api/moments.js 的 createMoment —— content 必填、images 传逗号分隔字符串、
 * status=1 表示发布（0 为草稿）；不传 audit，是否进待审核由后端按站点「动态审核」开关决定
 * （强制传 audit=1 会让审核开关形同虚设）。
 *
 * 正文用 EmojiEditor，所以 @ 提及、表情能力与其它输入处完全一致。
 */
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import EmojiEditor from './EmojiEditor.vue'
import AttachmentLibrary from './AttachmentLibrary.vue'
import ScopeRadio from './ScopeRadio.vue'
import LivePhotoVideoPicker from './LivePhotoVideoPicker.vue'
import { useUserStore } from '@/stores/user'
import { createMoment, uploadMomentImages } from '@/api/moments'
import { serializeLiveList } from '@/utils/livePhoto'
import { toast } from '@/utils/toast'

const MAX_IMAGES = 9
const MAX_SIZE = 50 * 1024 * 1024

defineProps({
  // 表情面板是否内联展开：弹窗里必须 true（面板撑高弹窗）；
  // 保留默认 false 是为了以后若在页面里内联复用（浮层，不把页面顶长）
  inlinePicker: { type: Boolean, default: false }
})

const emit = defineEmits(['published'])

const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const content = ref('')
/**
 * 待发布配图：每项 { image, video }
 * video 为空是普通图片；有视频就是实况图（存库时写成 图片::视频，见 utils/livePhoto.js）
 */
const images = ref([])
const location = ref('')
// 可见范围（后端字段 scope）：1 公开 2 登录可见 3 付费可见；付费时 scope_integral 为兑换积分
const scope = ref(1)
const scopeIntegral = ref(0)
const scopeOptions = [
  { value: 1, label: '公开可见' },
  { value: 2, label: '登录可见' },
  { value: 3, label: '付费可见' }
]
const publishing = ref(false)
const uploading = ref(false)
const showLibrary = ref(false)
const fileInput = ref(null)
const editorRef = ref(null)

// 自画工具栏用：开关编辑器的表情面板
function openEmoji() {
  editorRef.value?.openEmoji()
}

// 自画工具栏用：打开 @ 提及候选（直接选人插入「@昵称」，不用手打 @）
// 带上事件：编辑器里还没落光标时，面板以这个按钮为锚点，避免定位到视口左上角
function openMention(event) {
  editorRef.value?.openMention(event)
}

// 自画工具栏用：插入话题标记「##」（光标落在两个 # 之间，接着输入话题名即可）
function insertTopicMark() {
  editorRef.value?.insertTopic()
}

const avatar = computed(() => user.value?.avatar || '')
const nickname = computed(() => user.value?.nickname || '')

function triggerPick() {
  fileInput.value?.click()
}

// 选择图片后立即上传（attachment/batch），成功后记录 URL 用于预览
async function onPick(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length || uploading.value) return

  if (images.value.length + files.length > MAX_IMAGES) {
    toast.warning(`最多上传 ${MAX_IMAGES} 张图片`)
    return
  }

  const valid = files.filter((file) => {
    if (file.size > MAX_SIZE) {
      toast.warning(`图片「${file.name}」超过 50MB 限制`)
      return false
    }
    if (!file.type.startsWith('image/')) {
      toast.warning(`文件「${file.name}」不是图片`)
      return false
    }
    return true
  })
  if (!valid.length) return

  uploading.value = true
  try {
    const formData = new FormData()
    valid.forEach((file) => formData.append('files', file))

    const res = await uploadMomentImages(formData)
    const results = res.data?.results || []
    const rows = results.filter((item) => item.status !== 'fail' && item.full_url)

    // 安卓实况照片是一张内嵌视频的 jpg：后端拆完会带回 live_video，
    // 这里直接配成实况图，用户不用再点「实况」按钮单独传一遍视频
    images.value.push(...rows.map((row) => ({ image: row.full_url, video: row.live_video || '' })))

    const liveCount = rows.filter((row) => row.live_video).length
    if (liveCount) toast.success(`已识别 ${liveCount} 张实况图`)

    if (results.some((item) => item.status === 'fail')) {
      toast.warning('部分图片上传失败')
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

function removeImage(index) {
  images.value.splice(index, 1)
}

// 附件库：把选中的已上传图片加入待发布列表（去重 + 数量上限）
// items 里带 live_video（后端按「<封面名>-live.mp4」反查配对视频），
// 从库里选历史实况图时也能直接成为实况图，不需要再手动指定视频
function onLibrarySelect(urls = [], items = []) {
  const rest = MAX_IMAGES - images.value.length
  if (rest <= 0) {
    toast.warning(`最多上传 ${MAX_IMAGES} 张图片`)
    return
  }

  const exists = new Set(images.value.map((item) => item.image))
  const liveByUrl = new Map(items.map((item) => [item.full_url, item.live_video || '']))
  const add = urls.filter((url) => url && !exists.has(url)).slice(0, rest)
  images.value.push(...add.map((url) => ({ image: url, video: liveByUrl.get(url) || '' })))

  if (!add.length) {
    toast.info('所选图片已在列表中')
  } else if (add.length < urls.length) {
    toast.warning('受数量限制，部分附件未添加')
  } else {
    toast.success(`已添加 ${add.length} 张图片`)
  }
}

/** 发布：成功后清空表单并通知父级（父级决定是否刷新列表 / 关闭弹窗） */
async function submit() {
  if (publishing.value || uploading.value) return
  if (!content.value.trim()) {
    toast.warning('内容不能为空')
    return
  }

  publishing.value = true
  try {
    const scopeValue = [1, 2, 3].includes(Number(scope.value)) ? Number(scope.value) : 1
    const res = await createMoment({
      content: content.value,
      // 实况图写成 图片::视频（普通图不受影响）
      images: serializeLiveList(images.value),
      location: location.value.trim(),
      scope: scopeValue,
      scope_integral: scopeValue === 3 ? Math.max(1, Number(scopeIntegral.value) || 0) : 0,
      status: 1
    })

    content.value = ''
    images.value = []
    location.value = ''
    scope.value = 1
    scopeIntegral.value = 0
    // 按后端回传的 audit 提示：站点关闭「内容审核」时 audit=1（已通过），
    // 不能再无条件提示「等待审核」——否则用户会以为内容进了待审核
    toast.success(Number(res?.data?.audit) === 0 ? '发布成功，等待审核通过后对外可见' : '发布成功')
    emit('published')
  } catch {
    /* 拦截器已提示 */
  } finally {
    publishing.value = false
  }
}

/** 是否已填写内容（弹窗外壳据此决定关闭时要不要二次确认） */
const hasDraft = computed(() => !!content.value.trim() || images.value.length > 0)

/** 清空表单（取消/关闭弹窗时用；发布成功后 submit 内部已清空） */
function reset() {
  content.value = ''
  images.value = []
  location.value = ''
  scope.value = 1
  scopeIntegral.value = 0
}

defineExpose({ submit, reset, hasDraft })
</script>

<style scoped>
.composer-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.composer-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
}
.composer-avatar-icon {
  font-size: 30px;
  color: var(--text-light, var(--text-muted));
}
.composer-name {
  font-size: 13px;
  font-weight: 500;
}
.composer-count {
  font-size: 12px;
  color: var(--text-muted);
}
.composer-file {
  display: none;
}

/* ---------- 工具栏：与 MarkdownEditor 的 .markdown-editor-toolbar / .markdown-editor-tool 保持一致 ---------- */
.composer-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-soft);
}
.composer-tool {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 30px;
  height: 30px;
  padding: 0 6px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  font-size: 15px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.composer-tool:hover:not(:disabled) {
  background: var(--bg-muted);
  color: var(--primary);
}
.composer-tool:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.composer-toolbar-split {
  width: 1px;
  height: 16px;
  margin: 0 6px;
  background: var(--border);
}

.composer-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.composer-image {
  position: relative;
  width: 72px;
  height: 72px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border-soft);
}
.composer-image-img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
/* 实况图：左上角 LIVE 角标（右下角的「实况」按钮由 LivePhotoVideoPicker 提供） */
.composer-image-live {
  position: absolute;
  left: 4px;
  top: 4px;
  padding: 0 4px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 14px;
}

.composer-image-del {
  position: absolute;
  top: 4px;
  right: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.2s;
}
.composer-image-del:hover {
  background: var(--danger);
}

.composer-loc {
  position: relative;
  margin-top: 10px;
}
.composer-loc .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
  pointer-events: none;
}
.composer-loc-input {
  width: 100%;
  /* 与发布器工具栏按钮（.btn-sm）等高 */
  height: var(--control-h-sm);
  padding: 0 12px 0 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.composer-loc-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

/* 可见范围：标签 + 单选框组 +（付费时的）兑换积分输入 */
.composer-scope {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 10px;
}
.composer-scope-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: none;
  font-size: 12px;
  color: var(--text-muted);
}
.composer-scope-input {
  width: 160px;
  height: var(--control-h-sm);
  padding: 0 12px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.composer-scope-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

.composer-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 8px;
}
.composer-tip {
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-muted);
}

/* 手机端：位置输入与按钮更紧凑一点 */
@media (max-width: 640px) {
  .composer-image {
    width: 64px;
    height: 64px;
  }
}
</style>
