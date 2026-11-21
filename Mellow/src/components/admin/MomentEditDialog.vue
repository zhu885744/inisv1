<template>
  <AdminFormDialog
    :visible="visible"
    :title="title"
    icon="bi bi-pencil-square"
    width="560px"
    :loading="saving"
    @update:visible="onVisibleChange"
  >
    <!-- 作者：管理员编辑他人动态时能看到这条是谁的（新建时 form.id 为空，不显示） -->
    <OwnerInfo v-if="form.id" :uid="form.uid" label="作者" class="edit-owner" />

    <EmojiEditor v-model="form.content" placeholder="此刻的想法..." inline-picker>
      <template #extra>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :class="{ 'is-loading': uploading }"
          :disabled="uploading"
          title="上传图片"
          @click="fileInput?.click()"
        >
          <i class="bi bi-image" /> {{ uploading ? '上传中...' : '图片' }}
        </button>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          :disabled="uploading"
          title="从附件库选择已有图片"
          @click="showLibrary = true"
        >
          <i class="bi bi-folder2-open" /> 附件库
        </button>
      </template>
    </EmojiEditor>

    <input
      ref="fileInput"
      type="file"
      accept="image/*"
      multiple
      class="hidden-file"
      @change="onPick"
    />

    <AttachmentLibrary
      v-model:visible="showLibrary"
      title="选择动态图片"
      accept="image"
      multiple
      :max="MAX_IMAGES"
      @select="onLibrarySelect"
    />

    <div v-if="form.images.length" class="edit-images">
      <div v-for="(img, i) in form.images" :key="img + i" class="edit-img-item">
        <img :src="img" alt="配图" loading="lazy" />
        <span v-if="liveMap[img]" class="edit-img-live">LIVE</span>
        <!-- 实况视频：本地上传或从附件库挑选，保存时按图片地址回填 -->
        <LivePhotoVideoPicker
          :video="liveMap[img] || ''"
          :disabled="saving"
          @update:video="setLiveVideo(img, $event)"
        />
        <button type="button" class="img-remove" title="移除" @click="removeImage(i)">
          <i class="bi bi-x-lg" />
        </button>
      </div>
    </div>

    <div class="location-box">
      <i class="bi bi-geo-alt" aria-hidden="true" />
      <FormInput
        v-model="form.location"
        class="dialog-input"
        type="text"
        placeholder="位置（可选）"
        maxlength="64" />
    </div>

    <!-- 可见范围：1 公开 / 2 登录可见 / 3 付费可见（积分兑换），用单选框平铺 -->
    <div class="scope-box">
      <span class="scope-label">可见范围</span>
      <ScopeRadio v-model="form.scope" :options="scopeOptions" />
      <FormInput
        v-if="Number(form.scope) === 3"
        v-model.number="form.scopeIntegral"
        class="dialog-input scope-input"
        type="number"
        min="1"
        max="999999"
        placeholder="兑换所需积分" />
    </div>

    <label class="switch-row">
      <FormCheckbox v-model="form.status" :true-value="1" :false-value="0" />
      <span>{{ form.status === 1 ? '已发布' : '存为草稿' }}</span>
    </label>

    <template #footer>
      <button type="button" class="btn btn-sm" :disabled="saving" @click="requestClose">取消</button>
      <button
        type="button"
        class="btn btn-primary btn-sm"
        :class="{ 'is-loading': saving }"
        :disabled="saving || uploading || !canSave"
        @click="submit"
      >
        保存
      </button>
    </template>
  </AdminFormDialog>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import FormCheckbox from '@/components/form/FormCheckbox.vue'
/**
 * 后台动态编辑弹窗
 *
 * 与 manage/Moments.vue 的编辑弹窗保持同一套交互（表情编辑、上传、附件库、位置、发布状态），
 * 抽成组件后列表页只关心「打开哪一条」与「保存成功」两件事。
 */
import { ref, reactive, computed, watch } from 'vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import OwnerInfo from '@/components/admin/OwnerInfo.vue'
import EmojiEditor from '@/components/EmojiEditor.vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import LivePhotoVideoPicker from '@/components/LivePhotoVideoPicker.vue'
import ScopeRadio from '@/components/ScopeRadio.vue'
import { updateMoment, uploadMomentImages } from '@/api/moments'
import { parseMomentImages, parseMomentMedia, serializeMomentImages } from '@/utils/moment'
import { toast } from '@/utils/toast'

const MAX_IMAGES = 9
const MAX_SIZE = 10 * 1024 * 1024

const props = defineProps({
  visible: { type: Boolean, default: false },
  // 待编辑的动态原始数据（打开弹窗前由父级赋值）
  moment: { type: Object, default: null }
})

const emit = defineEmits(['update:visible', 'saved'])

const form = reactive({
  id: null,
  // 作者 uid：只用于展示「作者：昵称」，不参与提交
  uid: 0,
  // 审核状态：moments/update 会依据 status 重算 audit，保存时必须回传原值
  audit: 1,
  content: '',
  images: [],
  location: '',
  // 可见范围（后端字段 scope）：1 公开 2 登录可见 3 付费可见
  scope: 1,
  scopeIntegral: 0,
  status: 1
})

const scopeOptions = [
  { value: 1, label: '公开（所有人可见）' },
  { value: 2, label: '登录可见' },
  { value: 3, label: '付费可见（积分兑换）' }
]

const saving = ref(false)
const uploading = ref(false)
const showLibrary = ref(false)
const fileInput = ref(null)
/** 图片地址 → 实况视频地址：弹窗只编辑图片列表，保存时按地址回填视频 */
const liveMap = ref({})

const title = computed(() => (form.id ? `编辑动态 #${form.id}` : '编辑动态'))
const canSave = computed(() => !!form.content.trim() || form.images.length > 0)

// 每次打开用最新数据重建表单，避免上一条的残留内容串台
watch(
  () => props.visible,
  (visible) => {
    if (visible) resetForm()
  }
)

function resetForm() {
  const item = props.moment || {}
  form.id = item.id ?? null
  form.uid = Number(item.uid) || 0
  // 审核状态：0 待审核 / 1 通过 / 2 未通过。缺省按「通过」处理，
  // 避免旧数据没有该字段时 `Number(undefined) || 0` 把动态打回待审核
  const audit = Number(item.audit)
  form.audit = [0, 1, 2].includes(audit) ? audit : 1
  form.content = item.content || ''
  form.images = parseMomentImages(item.images)
  // 实况图的视频单独记住：本弹窗只编辑图片列表，保存时按图片地址回填视频，
  // 避免「后台改一次正文，实况图就退化成静态图」
  liveMap.value = {}
  parseMomentMedia(item.images).forEach((media) => {
    if (media.video) liveMap.value[media.image] = media.video
  })
  form.location = item.location || ''
  const scope = Number(item.scope)
  form.scope = [1, 2, 3].includes(scope) ? scope : 1
  form.scopeIntegral = Number(item.scope_integral) || 0
  form.status = Number(item.status) === 0 ? 0 : 1
  saving.value = false
  uploading.value = false
  showLibrary.value = false
}

function requestClose() {
  if (saving.value) return
  emit('update:visible', false)
}

// AdminFormDialog 不感知保存中状态，这里拦截一次，避免请求进行中弹窗被关闭
function onVisibleChange(visible) {
  if (visible) {
    emit('update:visible', true)
    return
  }
  requestClose()
}

function removeImage(index) {
  const [removed] = form.images.splice(index, 1)
  if (removed && liveMap.value[removed]) delete liveMap.value[removed]
}

/** 给某张图设置 / 取消实况视频（本地上传或附件库都走这里） */
function setLiveVideo(image, url) {
  if (!image) return
  if (url) liveMap.value[image] = url
  else delete liveMap.value[image]
}

async function onPick(event) {
  const files = Array.from(event.target.files || [])
  event.target.value = ''
  if (!files.length || uploading.value) return

  if (form.images.length + files.length > MAX_IMAGES) {
    toast.warning(`最多上传 ${MAX_IMAGES} 张图片`)
    return
  }

  const valid = files.filter((file) => {
    if (!file.type.startsWith('image/')) {
      toast.warning(`文件「${file.name}」不是图片`)
      return false
    }
    if (file.size > MAX_SIZE) {
      toast.warning(`图片「${file.name}」超过 10MB 限制`)
      return false
    }
    return true
  })
  if (!valid.length) return

  uploading.value = true
  try {
    const fd = new FormData()
    valid.forEach((file) => fd.append('files', file))
    const res = await uploadMomentImages(fd)
    const results = res.data?.results || []
    const rows = results.filter((r) => r.status !== 'fail' && r.full_url)
    // 安卓实况照片（内嵌视频的 jpg）：后端拆出的视频记到 liveMap，
    // 保存时再按图片地址回填成「图片::视频」
    rows.forEach((row) => {
      if (row.live_video) liveMap.value[row.full_url] = row.live_video
    })
    form.images.push(...rows.map((row) => row.full_url))
    if (rows.some((row) => row.live_video)) toast.success('已识别实况图')
    if (results.some((r) => r.status === 'fail')) toast.warning('部分图片上传失败')
  } catch {
    /* 请求拦截器已提示 */
  } finally {
    uploading.value = false
  }
}

// 附件库：把选中的已上传图片加入配图（去重 + 数量上限）
// items 里带 live_video：从库里选历史实况图时直接记进 liveMap，保存时拼成「图片::视频」
function onLibrarySelect(urls = [], items = []) {
  const rest = MAX_IMAGES - form.images.length
  if (rest <= 0) {
    toast.warning(`最多上传 ${MAX_IMAGES} 张图片`)
    return
  }
  items.forEach((item) => {
    if (item?.full_url && item.live_video) liveMap.value[item.full_url] = item.live_video
  })
  const add = urls.filter((url) => url && !form.images.includes(url)).slice(0, rest)
  form.images.push(...add)
  if (!add.length) {
    toast.info('所选图片已在列表中')
  } else if (add.length < urls.length) {
    toast.warning('受数量限制，部分附件未添加')
  } else {
    toast.success(`已添加 ${add.length} 张图片`)
  }
}

async function submit() {
  if (!form.id || saving.value || !canSave.value) return
  saving.value = true
  try {
    // 回传原 audit，避免保存时被后端的审核规则打回待审核
    const scopeValue = [1, 2, 3].includes(Number(form.scope)) ? Number(form.scope) : 1
    await updateMoment({
      id: form.id,
      content: form.content,
      // 回填实况视频：普通图仍是纯地址，实况图恢复成 图片::视频
      images: serializeMomentImages(
        form.images.map((image) => ({ image, video: liveMap.value[image] || '' }))
      ),
      location: form.location.trim(),
      scope: scopeValue,
      scope_integral: scopeValue === 3 ? Math.max(1, Number(form.scopeIntegral) || 0) : 0,
      status: form.status,
      audit: form.audit
    })
    toast.success('保存成功')
    emit('update:visible', false)
    emit('saved', form.id)
    form.id = null
  } catch {
    /* 请求拦截器已提示，保持弹窗打开便于重试 */
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.hidden-file {
  display: none;
}

/* 作者一行：与下面的正文编辑区留点距离 */
.edit-owner {
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px dashed var(--border-soft);
}

.edit-images {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.edit-img-item {
  position: relative;
  width: 72px;
  height: 72px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--border);
}
.edit-img-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
/* 实况图角标（左上角，与发布器的 LIVE 标识一致） */
.edit-img-live {
  position: absolute;
  left: 3px;
  top: 3px;
  padding: 0 3px;
  border-radius: 3px;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 9px;
  font-weight: 600;
  letter-spacing: 0.04em;
  line-height: 14px;
}
.img-remove {
  position: absolute;
  top: 0;
  right: 0;
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  cursor: pointer;
  border-radius: 0 0 0 6px;
}
.img-remove:hover {
  background: var(--danger);
}

.location-box {
  position: relative;
  margin-top: 10px;
}
.location-box .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
  pointer-events: none;
}
.dialog-input {
  width: 100%;
  padding: 7px 12px 7px 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.dialog-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

/* 可见范围：标签 + 单选框组 +（付费时的）积分输入 */
.scope-box {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  margin-top: 12px;
}
.scope-label {
  flex: none;
  font-size: 13px;
  color: var(--text-soft);
}
.scope-input {
  width: 140px;
  padding-left: 12px;
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
</style>
