<template>
  <div class="profile-page">
    <div class="profile-layout">
      <!-- 左列：头像 + 主页封面（都是「图」相关，放一起方便对照最终效果） -->
      <div class="profile-col">
        <section class="card card-pad">
          <header class="block-head">
            <h2 class="block-title">头像</h2>
          </header>
          <p class="block-desc">上传后可拖动、缩放裁切，圆形区域即最终展示效果</p>

          <div class="avatar-top">
            <div class="avatar-preview">
              <AvatarFrame
                :src="form.avatar"
                :name="form.nickname"
                :frame="form.frame"
                :animated="frameRender.animated"
                :fallback="defaultAvatar"
                :size="'96px'"
                :frame-hole="frameRender.hole"
                :frame-offset-x="frameRender.offsetX"
                :frame-offset-y="frameRender.offsetY"
                :verified="userStore.user?.cert"
                alt="头像预览"
              />
            </div>
            <div class="avatar-controls">
              <div class="avatar-actions">
                <label class="btn btn-sm btn-primary">
                  <i class="bi bi-cloud-upload" aria-hidden="true" />
                  {{ uploading ? '上传中…' : (form.avatar ? '更换头像' : '上传头像') }}
                  <input
                    v-if="!uploading"
                    type="file"
                    accept="image/*"
                    hidden
                    @change="onAvatarFileChange"
                  />
                </label>
                <button
                  type="button"
                  class="btn btn-sm"
                  title="从附件库选择头像"
                  @click="showLibrary = true"
                >
                  <i class="bi bi-folder2-open" aria-hidden="true" /> 附件库
                </button>
                <button
                  v-if="form.avatar"
                  type="button"
                  class="btn btn-sm btn-ghost"
                  :disabled="uploading"
                  @click="removeAvatar"
                >
                  <i class="bi bi-trash3" aria-hidden="true" /> 移除
                </button>
              </div>
              <p class="hint">支持 JPG、PNG、WebP；GIF 保留动图、不裁切。保存修改后生效。</p>
            </div>
          </div>

          <!-- 附件库（选择已上传图片作为头像） -->
          <AttachmentLibrary
            v-model:visible="showLibrary"
            title="选择头像"
            accept="image"
            @select="onLibrarySelect"
          />

          <!-- 默认头像 -->
          <div v-if="DEFAULT_AVATARS.length" class="preset-section">
            <div class="preset-label">
              <span><i class="bi bi-people" aria-hidden="true" /> 默认头像</span>
              <span class="preset-count">{{ DEFAULT_AVATARS.length }} 个</span>
            </div>
            <div class="preset-grid">
              <button
                v-for="(url, idx) in DEFAULT_AVATARS"
                :key="idx"
                type="button"
                class="preset-item"
                :class="{ 'is-selected': form.avatar === url }"
                :title="`默认头像 ${idx + 1}`"
                @click="selectDefaultAvatar(url)"
              >
                <img :src="thumbUrl(url, 112)" :alt="`默认头像 ${idx + 1}`" loading="lazy" decoding="async" />
              </button>
            </div>
          </div>
        </section>

        <!-- 主页封面（存 users.json.cover）：自定义图片 / 自定义颜色 二选一，用户主页顶部横幅使用 -->
        <section class="card card-pad">
          <header class="block-head">
            <h2 class="block-title">主页封面</h2>
            <button v-if="hasCover" type="button" class="btn btn-xs btn-ghost" @click="clearCover">
              <i class="bi bi-arrow-counterclockwise" aria-hidden="true" /> 恢复默认
            </button>
          </header>
          <p class="block-desc">显示在你的个人主页顶部，图片会按 3:1 裁切</p>

          <!-- 封面类型：图片 / 颜色 -->
          <div class="cover-mode">
            <button
              v-for="m in COVER_MODES"
              :key="m.key"
              type="button"
              class="cover-mode-item"
              :class="{ 'is-active': coverMode === m.key }"
              @click="switchCoverMode(m.key)"
            >
              <i :class="m.icon" aria-hidden="true" /> {{ m.label }}
            </button>
          </div>

          <!-- 自定义图片 -->
          <template v-if="coverMode === 'image'">
            <div class="cover-preview" :class="{ 'is-empty': !form.cover }">
              <img
                v-if="form.cover"
                :src="thumbUrl(form.cover, 960)"
                alt="主页封面预览"
                loading="lazy"
                decoding="async"
              />
              <div v-else class="cover-empty">
                <i class="bi bi-image" aria-hidden="true" />
                <span>未设置封面，将使用主题色渐变</span>
              </div>
              <span v-if="form.cover" class="cover-ratio">3:1</span>
            </div>
            <div class="cover-actions">
              <label class="btn btn-sm btn-primary">
                <i class="bi bi-crop" aria-hidden="true" />
                {{ coverUploading ? '上传中…' : '上传并裁切' }}
                <input
                  v-if="!coverUploading"
                  type="file"
                  accept="image/*"
                  hidden
                  @change="onCoverFileChange"
                />
              </label>
              <button
                type="button"
                class="btn btn-sm"
                title="从附件库选择封面"
                @click="showCoverLibrary = true"
              >
                <i class="bi bi-folder2-open" aria-hidden="true" /> 附件库
              </button>
            </div>
            <p class="hint">建议 3:1（如 1620×540），主体放中间；保存修改后生效。</p>
          </template>

          <!-- 自定义颜色 -->
          <template v-else>
            <div class="cover-preview is-color" :style="{ background: coverColorValue }">
              <span class="cover-color-hex">{{ coverColorValue }}</span>
            </div>
            <div class="cover-color-row">
              <ColorPicker v-model="form.coverColor" :show-alpha="false" :presets="COVER_COLOR_PRESETS" />
            </div>
            <p class="hint">用取色器或直接填色值都可以；保存修改后生效。</p>
          </template>

          <!-- 附件库（选择已上传图片作为封面） -->
          <AttachmentLibrary
            v-model:visible="showCoverLibrary"
            title="选择封面"
            accept="image"
            @select="onCoverLibrarySelect"
          />
        </section>
      </div>

      <!-- 右列：基本信息表单 -->
      <div class="profile-col">
        <section class="card card-pad">
          <header class="block-head">
            <h2 class="block-title">个人信息</h2>
          </header>
          <p class="block-desc">这些内容会展示在你的个人主页</p>

          <form id="profile-form" novalidate @submit.prevent="save">
            <div class="form-item">
              <label class="form-label" for="profile-nickname">昵称</label>
              <FormInput
                id="profile-nickname"
                v-model="form.nickname"
                placeholder="请输入昵称"
                maxlength="20"
                autocomplete="nickname" />
            </div>

            <div class="form-item">
              <label class="form-label">性别</label>
              <div class="radio-row">
                <label class="radio-item"><FormRadio v-model="form.gender" :value="1" /> 男</label>
                <label class="radio-item"><FormRadio v-model="form.gender" :value="2" /> 女</label>
                <label class="radio-item"><FormRadio v-model="form.gender" :value="0" /> 保密</label>
              </div>
            </div>

            <div class="form-item">
              <label class="form-label">头衔</label>
              <!-- 只读展示当前头衔：佩戴 / 卸下 / 换字统一在「我的装扮」里操作 -->
              <div class="current-title-row">
                <span class="current-title-label">当前头衔</span>
                <!-- 佩戴了装扮头衔时用 TitleBadge 还原后台配色（文字色 / 背景 / 边框 / 发光 / 图标 / 动效） -->
                <TitleBadge
                  v-if="form.title || currentTitlePayload"
                  :payload="currentTitlePayload || {}"
                  :text="form.title"
                />
                <span v-else class="current-title empty">未设置</span>
                <RouterLink to="/user/decorations" class="decoration-link">去更换</RouterLink>
              </div>
              <p class="hint">头衔与头像框统一在「我的装扮」中佩戴、卸下。</p>
            </div>

            <div class="form-item">
              <label class="form-label" for="profile-desc">个人简介</label>
              <FormTextarea
                id="profile-desc"
                v-model="form.description"
                rows="4"
                maxlength="200"
                placeholder="介绍一下自己吧" />
              <div class="char-count" :class="{ 'is-full': form.description.length >= 200 }">
                {{ form.description.length }}/200
              </div>
            </div>

            <div class="form-item">
              <label class="form-label">个人网站</label>
              <FormInput v-model="form.websiteName" placeholder="网站名称" maxlength="50" />
              <FormInput
                v-model="form.websiteUrl"
                class="website-url"
                placeholder="https://blog.example.com"
                maxlength="100"
                inputmode="url" />
              <p class="hint">展示在你的个人主页上，两项都留空表示不展示</p>
            </div>

            <div class="form-item">
              <label class="form-label">联系方式</label>
              <div class="contact-readonly">
                <div class="input is-disabled"><i class="bi bi-phone" aria-hidden="true" /> {{ userStore.user?.phone || '未设置' }}</div>
                <div class="input is-disabled"><i class="bi bi-envelope" aria-hidden="true" /> {{ userStore.user?.email || '未设置' }}</div>
              </div>
              <p class="hint">
                邮箱和手机号是登录凭证，修改入口在
                <RouterLink to="/user/security" class="hint-link">账号安全</RouterLink>
              </p>
            </div>
          </form>
        </section>
      </div>
    </div>

    <!-- 底部保存条：有改动才可点，避免误提交 / 无意义请求 -->
    <div class="save-bar" :class="{ 'is-dirty': isDirty }">
      <span class="save-state">
        <i :class="isDirty ? 'bi bi-exclamation-circle' : 'bi bi-check-circle'" aria-hidden="true" />
        {{ isDirty ? '有未保存的修改' : '资料已是最新' }}
      </span>
      <div class="save-actions">
        <button type="button" class="btn btn-sm" :disabled="loading || !isDirty" @click="resetForm">重置</button>
        <button
          type="submit"
          form="profile-form"
          class="btn btn-sm btn-primary"
          :disabled="loading || !isDirty"
        >
          {{ loading ? '保存中…' : '保存修改' }}
        </button>
      </div>
    </div>

    <!-- 裁切弹窗：异步加载，只有真正上传图片时才拉这段代码 -->
    <ImageCropper
      v-if="crop.visible"
      v-model:visible="crop.visible"
      :file="crop.file"
      :src="crop.src"
      :title="cropPreset.title"
      :aspect="cropPreset.aspect"
      :shape="cropPreset.shape"
      :output-width="cropPreset.outputWidth"
      :busy="crop.busy"
      @confirm="onCropConfirm"
      @unsupported="onCropUnsupported"
    />
  </div>
</template>

<script setup>
import FormRadio from '@/components/form/FormRadio.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { ref, reactive, computed, onMounted, watch, nextTick, defineAsyncComponent } from 'vue'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { updateUser, uploadAvatar } from '@/api/users'
import { toast } from '@/utils/toast'
import { thumbUrl } from '@/utils/image'
import AvatarFrame from '@/components/AvatarFrame.vue'
import TitleBadge from '@/components/TitleBadge.vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import ColorPicker from '@/components/ColorPicker.vue'
import { pickUserTitlePayload } from '@/utils/helper'
import { resolveFrameRender } from '@/utils/frameRender'

// 裁切弹窗按需加载：首屏不必为「可能不上传图片」的场景付解析成本
const ImageCropper = defineAsyncComponent(() => import('@/components/ImageCropper.vue'))

const userStore = useUserStore()
const siteStore = useSiteStore()

// 头衔样式 / 头像框动效（都由装扮系统写入 users.json，佩戴时服务端回写）
const currentTitlePayload = computed(() => pickUserTitlePayload(userStore.user))
// 头像框渲染参数：以装扮 payload 为准（users.json 上的副本只作兜底）
const frameRender = computed(() => resolveFrameRender(userStore.user?.json))

const BUILTIN_AVATARS = [
  'https://img.zhuxu.asia/tx/1.png',
  'https://img.zhuxu.asia/tx/2.png',
  'https://img.zhuxu.asia/tx/3.png',
  'https://img.zhuxu.asia/tx/4.png',
  'https://img.zhuxu.asia/tx/5.png'
]
// 优先用后台「默认值设置 → 默认头像」配置；未配置时使用主题内置头像兜底
const DEFAULT_AVATARS = ref(BUILTIN_AVATARS)

// 注意：头像框 / 头衔的佩戴、卸下、换字全部收口在「我的装扮」(/user/decorations)，
// 本页只做「头像上传 + 基本信息」——这里不再拉取装扮数据、也不提供装扮设置控件。

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/></svg>'

const loading = ref(false)
const uploading = ref(false)
const showLibrary = ref(false)
// 主页封面：上传状态 + 附件库开关
const coverUploading = ref(false)
const showCoverLibrary = ref(false)

// ===== 主页封面：自定义图片 / 自定义颜色 二选一 =====
// 当前选择的封面类型：image 自定义图片 / color 自定义颜色
const coverMode = ref('image')
const COVER_MODES = [
  { key: 'image', label: '自定义图片', icon: 'bi-image' },
  { key: 'color', label: '自定义颜色', icon: 'bi-palette' }
]
// 常用色（第一枚即站点朱砂主色）
const COVER_COLORS = ['#c7482a', '#c8a04a', '#2e7a5c', '#2f6f8f', '#6c5ce7', '#4a4f57']
// 默认封面色：优先取当前主题的主色（--primary），取不到时用朱砂兜底
const DEFAULT_COVER_COLOR = (() => {
  const value = typeof document !== 'undefined'
    ? getComputedStyle(document.documentElement).getPropertyValue('--primary').trim()
    : ''
  return /^#[0-9a-f]{6}$/i.test(value) ? value.toLowerCase() : '#c7482a'
})()

// 色板：主题色优先，避免「默认色」不在预设里选中态落空
const COVER_COLOR_PRESETS = [
  DEFAULT_COVER_COLOR,
  ...COVER_COLORS.filter((c) => c.toLowerCase() !== DEFAULT_COVER_COLOR)
]

const hasCover = computed(() => !!(form.cover || form.coverColor))
// 颜色预览统一走合法色值，避免空串 / 非法值时预览变成透明
const coverColorValue = computed(() => normalizeHex(form.coverColor) || DEFAULT_COVER_COLOR)

// #rgb / #rrggbb → #rrggbb（后端只认 6 位色值，#abc 要展开后再提交）
function normalizeHex(value) {
  const raw = String(value || '').trim()
  if (/^#[0-9a-f]{3}$/i.test(raw)) {
    return `#${raw[1]}${raw[1]}${raw[2]}${raw[2]}${raw[3]}${raw[3]}`.toLowerCase()
  }
  return /^#[0-9a-f]{6}$/i.test(raw) ? raw.toLowerCase() : ''
}

function switchCoverMode(mode) {
  coverMode.value = mode
  // 切到颜色模式且还没选过颜色时，先给一个主题主色，避免预览空白
  if (mode === 'color' && !form.coverColor) form.coverColor = DEFAULT_COVER_COLOR
}

function clearCover() {
  form.cover = ''
  form.coverColor = ''
  coverMode.value = 'image'
}

const form = reactive({
  id: '',
  nickname: '',
  gender: 0,
  description: '',
  avatar: '',
  title: '',
  frame: '',
  // 主页封面（存 users.json.cover）：用户主页顶部横幅，图片 / 颜色二选一
  cover: '',
  // 颜色封面的色值（json.cover = { type: 'color', value }）
  coverColor: '',
  // 个人网站（存 users.json.website）：展示在个人主页，原「联系方式」页的功能
  websiteName: '',
  websiteUrl: ''
})
const originalData = reactive({})
const originalMode = ref('image')

// ---------- 未保存改动 ----------
// 用快照对比而不是逐字段 watch：字段多、来源杂（封面还分模式），一行 JSON 最省事
const baseline = ref('')
const ready = ref(false)

function snapshot() {
  return JSON.stringify({
    nickname: form.nickname.trim(),
    gender: Number(form.gender),
    description: form.description,
    avatar: form.avatar,
    coverMode: coverMode.value,
    cover: form.cover,
    coverColor: normalizeHex(form.coverColor),
    websiteName: form.websiteName.trim(),
    websiteUrl: form.websiteUrl.trim()
  })
}

const isDirty = computed(() => ready.value && snapshot() !== baseline.value)

function markClean() {
  baseline.value = snapshot()
  ready.value = true
}

// ===== 图片裁切 =====
// 头像 1:1 圆形（导出 512）、封面 3:1 矩形（导出 1620）——都远大于实际展示尺寸，
// 但又不至于把原图（可能 4000px）整张丢给 canvas。
const CROP_PRESETS = {
  avatar: { aspect: 1, shape: 'circle', outputWidth: 512, title: '裁切头像' },
  cover: { aspect: 3, shape: 'square', outputWidth: 1620, title: '裁切封面' }
}
const crop = reactive({
  visible: false,
  target: 'avatar',
  file: null,
  src: '',
  busy: false
})
const cropPreset = computed(() => CROP_PRESETS[crop.target] || CROP_PRESETS.avatar)

// 关闭后立刻放掉 File / 远程地址引用，避免大图长期驻留内存
watch(() => crop.visible, (visible) => {
  if (visible) return
  crop.file = null
  crop.src = ''
})

const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']

function validateImage(file) {
  if (!file) return false
  if (!IMAGE_TYPES.includes(file.type)) {
    toast.warning('请选择 JPG、PNG、GIF 或 WebP 格式的图片')
    return false
  }
  if (file.size > 10 * 1024 * 1024) {
    toast.warning('图片大小不能超过 10MB')
    return false
  }
  return true
}

function openCropper(target, { file = null, src = '' } = {}) {
  crop.target = target
  crop.file = file
  crop.src = src
  crop.busy = false
  crop.visible = true
}

/** 上传到附件库并返回图片地址（走统一的 WebP 压缩策略） */
async function uploadImage(payload, target) {
  const name = payload.name || `${target}-${Date.now()}.webp`
  const params = new FormData()
  params.append('file', payload, name)
  const res = await uploadAvatar(params)
  const item = res.data?.results?.[0]
  return item?.full_url || item?.url || ''
}

function applyImage(url, target) {
  if (!url) {
    toast.error('上传失败，请重试')
    return false
  }
  if (target === 'cover') {
    form.cover = url
    form.coverColor = ''
    coverMode.value = 'image'
  } else {
    form.avatar = url
  }
  return true
}

function onAvatarFileChange(event) {
  const file = event.target.files?.[0]
  // 立刻清空，保证「同一张图重选」也能触发 change
  event.target.value = ''
  if (!validateImage(file)) return
  // GIF 裁切会丢动画帧：直接原图上传
  if (file.type === 'image/gif') {
    uploadDirect(file, 'avatar')
    return
  }
  openCropper('avatar', { file })
}

function onCoverFileChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!validateImage(file)) return
  if (file.type === 'image/gif') {
    uploadDirect(file, 'cover')
    return
  }
  openCropper('cover', { file })
}

/** 不裁切、直接上传（GIF 动图） */
async function uploadDirect(file, target) {
  const flag = target === 'cover' ? coverUploading : uploading
  flag.value = true
  try {
    const url = await uploadImage(file, target)
    if (applyImage(url, target)) toast.success('已上传，请点击「保存修改」完成更新')
  } catch {
    toast.error('上传失败，请稍后重试')
  } finally {
    flag.value = false
  }
}

async function onCropConfirm({ blob }) {
  const target = crop.target
  const flag = target === 'cover' ? coverUploading : uploading
  crop.busy = true
  flag.value = true
  try {
    const url = await uploadImage(blob, target)
    if (applyImage(url, target)) {
      crop.visible = false
      toast.success('已应用，请点击「保存修改」完成更新')
    }
  } catch {
    toast.error('上传失败，请稍后重试')
  } finally {
    crop.busy = false
    flag.value = false
  }
}

/**
 * 远程图无法导出（图床未开 CORS，canvas 会被污染）：
 * 不阻断流程，直接沿用原图，把「裁切」降级为「选取」。
 * 但如果这张图本身就打不开（404 / 失效链接），不能把死链写进资料里。
 */
function canLoadImage(url) {
  return new Promise((resolve) => {
    const img = new Image()
    img.onload = () => resolve(true)
    img.onerror = () => resolve(false)
    img.src = url
  })
}

async function onCropUnsupported() {
  const target = crop.target
  const url = crop.src
  crop.visible = false
  if (!url) return
  if (await canLoadImage(url)) {
    if (applyImage(url, target)) toast.info('该图片不支持在线裁切，已直接使用原图')
  } else {
    toast.error('图片读取失败，请换一张试试')
  }
}

// 附件库：选择已上传图片作为头像 / 封面（同样尝试裁切，失败自动回落原图）
// 延后到 nextTick 再开裁切：附件库关闭时会恢复 body 滚动，同步打开会被它「解锁」
function onLibrarySelect(urls = []) {
  const url = urls[0]
  if (url) nextTick(() => openCropper('avatar', { src: url }))
}

function onCoverLibrarySelect(urls = []) {
  const url = urls[0]
  if (url) nextTick(() => openCropper('cover', { src: url }))
}

function selectDefaultAvatar(url) {
  form.avatar = url
  toast.success('已选择默认头像，请点击「保存修改」完成更新')
}

function removeAvatar() {
  form.avatar = ''
  toast.info('头像已移除，请点击「保存修改」完成更新')
}

function fetchUserInfo() {
  const u = userStore.user
  if (!u || !Object.keys(u).length) return
  form.id = u.id
  form.nickname = u.nickname || ''
  // 性别映射
  if (u.gender === 'boy') form.gender = 1
  else if (u.gender === 'girl') form.gender = 2
  else if (typeof u.gender === 'number') form.gender = u.gender
  else form.gender = 0
  form.description = u.description || ''
  form.avatar = u.avatar || ''
  // title / frame 由装扮系统维护（佩戴时服务端自动回写），这里仅用于展示
  form.title = u.title || ''
  form.frame = u.json?.frame || ''
  // 主页封面：兼容早期「字符串 = 图片地址」与 { type, value } 两种写法
  const rawCover = u.json?.cover
  if (typeof rawCover === 'string' && rawCover) {
    form.cover = rawCover
    form.coverColor = ''
    coverMode.value = 'image'
  } else if (rawCover && typeof rawCover === 'object' && rawCover.value) {
    if (rawCover.type === 'color') {
      form.coverColor = normalizeHex(rawCover.value) || DEFAULT_COVER_COLOR
      form.cover = ''
      coverMode.value = 'color'
    } else {
      form.cover = rawCover.value
      form.coverColor = ''
      coverMode.value = 'image'
    }
  } else {
    form.cover = ''
    form.coverColor = ''
    coverMode.value = 'image'
  }
  // 个人网站（json.website）：缺省按未设置处理
  const website = u.json?.website && typeof u.json.website === 'object' ? u.json.website : {}
  form.websiteName = website.name || ''
  form.websiteUrl = website.url || ''
  Object.assign(originalData, { ...form })
  originalMode.value = coverMode.value
  markClean()
}

function resetForm() {
  Object.assign(form, { ...originalData })
  coverMode.value = originalMode.value
}

async function save() {
  if (!isDirty.value || loading.value) return
  if (!form.nickname.trim()) {
    toast.error('昵称不能为空')
    return
  }
  loading.value = true
  try {
    // 性别映射：1->boy, 2->girl, 0->空
    const genderMap = { 1: 'boy', 2: 'girl', 0: '' }
    // 头衔 / 头像框由装扮系统维护（佩戴走 /decoration/wear 服务端回写），
    // 这里不再提交，避免把服务端已写入的值覆盖回旧值；json 原样回传以防后端整体替换
    const currentJson = (userStore.user?.json && typeof userStore.user.json === 'object') ? { ...userStore.user.json } : {}
    // 主页封面写入 json.cover（图片 / 颜色二选一）；未设置时删掉该键，回到主题色渐变
    if (coverMode.value === 'color') {
      // 颜色模式下色值非法时回落到默认色，避免「填错就静默清空封面」
      currentJson.cover = { type: 'color', value: normalizeHex(form.coverColor) || DEFAULT_COVER_COLOR }
    } else if (coverMode.value === 'image' && form.cover) {
      currentJson.cover = { type: 'image', value: form.cover }
    } else {
      delete currentJson.cover
    }

    // 个人网站写入 json.website（名称 + 网址）；两项都为空时删掉该键，个人主页不再展示
    const websiteName = form.websiteName.trim()
    const websiteUrl = form.websiteUrl.trim()
    if (websiteUrl && !/^https?:\/\//.test(websiteUrl)) {
      toast.error('请输入有效的网站网址（以 http:// 或 https:// 开头）')
      return
    }
    if (websiteName || websiteUrl) {
      currentJson.website = { name: websiteName, url: websiteUrl }
    } else {
      delete currentJson.website
    }

    const payload = {
      id: form.id,
      nickname: form.nickname.trim(),
      description: form.description,
      avatar: form.avatar,
      json: currentJson
    }
    // 始终提交性别：选「保密」时传空字符串。若在 0 时跳过该字段，
    // 用户一旦选过男/女就再也改不回保密（后端会保留旧值）。
    payload.gender = genderMap[form.gender] ?? ''

    await updateUser(payload)
    toast.success('用户信息更新成功')
    Object.assign(originalData, { ...form })
    originalMode.value = coverMode.value
    markClean()
    // 同步用户信息（会触发下面的 watch，重新按服务端数据回填）
    await userStore.checkLoginState()
  } catch {
    // 错误已由拦截器提示
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  // 用户信息由下面 watch 的 immediate 负责回填，这里只补「默认头像」配置，
  // 避免同一份表单被回填两次。
  try {
    const site = await siteStore.load()
    // 开关关闭：前台不展示任何预设头像，用户只能自行上传
    if (site?.default_avatars_enabled === false) {
      DEFAULT_AVATARS.value = []
      return
    }
    const list = site?.default_avatars
    if (Array.isArray(list)) {
      // 已配置（含空数组）：尊重之——管理员删光后前台即无预设头像
      DEFAULT_AVATARS.value = list.filter((u) => typeof u === 'string' && u.trim())
    }
    // 字段不存在（从未配置）：保持内置兜底（DEFAULT_AVATARS 初始值）
  } catch {
    /* 读取失败保持内置头像兜底 */
  }
})

watch(() => userStore.user, (nu) => {
  if (nu && Object.keys(nu).length > 0) fetchUserInfo()
}, { immediate: true })
</script>

<style scoped>
/* ---------- 布局：窄屏单列，宽屏「外观 / 资料」并排 ---------- */
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.profile-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
.profile-col {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
@media (min-width: 1080px) {
  .profile-layout {
    grid-template-columns: minmax(0, 400px) minmax(0, 1fr);
  }
}

/* ---------- 区块标题 ---------- */
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 4px;
}
.block-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.block-desc {
  margin: 6px 0 16px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.block-head + .block-desc {
  margin-top: 4px;
}

/* ---------- 头像 ---------- */
.avatar-top {
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 16px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  margin-bottom: 16px;
}
.avatar-preview {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--bg-card);
}
.avatar-controls {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}
.avatar-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

/* ---------- 默认头像 ---------- */
.preset-section {
  margin-bottom: 4px;
}
.preset-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-soft);
  margin-bottom: 10px;
}
.preset-label i {
  margin-right: 6px;
  color: var(--primary);
}
.preset-count {
  font-size: 11px;
  font-weight: 400;
  color: var(--text-muted);
}
.preset-grid {
  /* 固定格子尺寸 + auto-fill：宽屏不会被均分成大方块，窄屏自动换行 */
  --preset-size: 56px;
  display: grid;
  grid-template-columns: repeat(auto-fill, var(--preset-size));
  gap: 8px;
}
.preset-item {
  position: relative;
  width: var(--preset-size);
  height: var(--preset-size);
  padding: 3px;
  background: var(--bg-card);
  border: 2px solid var(--border);
  border-radius: 50%;
  cursor: pointer;
  overflow: hidden;
  transition: border-color var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}
.preset-item img {
  display: block;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  /* 预设图在 CDN 上是 2048px，缩到 56px 显示，交给解码器降采样即可 */
}
.preset-item:hover {
  border-color: var(--primary);
  transform: translateY(-2px);
}
.preset-item.is-selected {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

/* ---------- 主页封面 ---------- */
.cover-mode {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  margin-bottom: 12px;
  border-radius: 999px;
  background: var(--bg-muted);
}
.cover-mode-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12px;
  font-family: inherit;
  line-height: 1.5;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.cover-mode-item:hover {
  color: var(--primary);
}
.cover-mode-item.is-active {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}
/* 3:1 预览：与个人主页横幅比例一致，所见即所得 */
.cover-preview {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  aspect-ratio: 3 / 1;
  max-height: 180px;
  margin-bottom: 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-muted);
  overflow: hidden;
}
.cover-preview img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
/* 未设置封面：虚线框 + 说明，明确会回落到主题色渐变（虚线用更弱的浅描边） */
.cover-preview.is-empty {
  border-style: dashed;
  border-color: var(--border-soft);
}
.cover-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 0 12px;
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-muted);
  text-align: center;
}
.cover-empty .bi {
  font-size: 18px;
  color: var(--text-light);
}
/* 比例角标：让用户知道会被裁成 3:1 */
.cover-ratio {
  position: absolute;
  right: 8px;
  bottom: 8px;
  padding: 1px 6px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.5);
  color: #fff;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
}
.cover-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 8px;
}
/* 颜色预览：铺满所选颜色，中间标出色值 */
.cover-preview.is-color {
  border-color: transparent;
  transition: background-color var(--dur-fast) var(--ease);
}
.cover-color-hex {
  padding: 2px 10px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.32);
  color: #fff;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}
.cover-color-row {
  margin-bottom: 8px;
}

/* ---------- 当前头衔：只读展示 ---------- */
.current-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 12px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.current-title-label {
  font-size: 12px;
  color: var(--text-muted);
}
.current-title {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xs);
}
.current-title.empty {
  color: var(--text-muted);
  font-weight: 400;
}
.decoration-link {
  margin-left: auto;
  font-size: 12px;
  color: var(--primary);
  text-decoration: none;
  white-space: nowrap;
}
.decoration-link:hover {
  text-decoration: underline;
}

/* ---------- 表单 ---------- */
.radio-row {
  display: flex;
  gap: 16px;
  padding: 4px 0;
}
.radio-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
.radio-item input {
  accent-color: var(--primary);
  cursor: pointer;
}
.char-count {
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-muted);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.char-count.is-full {
  color: var(--warning);
}
/* 个人网站：名称 + 网址上下两行，第二个输入框留一点间距 */
.website-url {
  margin-top: 8px;
}
.hint {
  margin: 6px 0 0;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-muted);
}
.hint-link {
  color: var(--primary);
  text-decoration: none;
}
.hint-link:hover {
  text-decoration: underline;
}
.contact-readonly {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
/* 只读展示：这两格是 <div>，不是 <input>。
   .input 的垂直居中原本依赖 <input> 自身的行内居中，div 只会把这一行按 line-height
   排在盒顶（看起来「偏上」），所以这里手动用 flex 居中。 */
.input.is-disabled {
  display: flex;
  align-items: center;
  gap: 6px;
  /* 只读值可能较长（尤其手机端两列更窄）：保持单行，右侧裁掉，不撑破固定高度 */
  white-space: nowrap;
  overflow: hidden;
  background: var(--bg-muted);
  color: var(--text-muted);
  cursor: not-allowed;
}
.input.is-disabled i {
  flex-shrink: 0;
  color: var(--text-light);
}

/* ---------- 底部保存条 ---------- */
.save-bar {
  position: sticky;
  bottom: 12px;
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  /* 老浏览器不认 color-mix 时退回实色，避免保存条变透明 */
  background: var(--bg-card);
  background: color-mix(in srgb, var(--bg-card) 92%, transparent);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  backdrop-filter: blur(6px);
  transition: border-color var(--dur-fast) var(--ease);
}
.save-bar.is-dirty {
  border-color: var(--accent-ring);
}
.save-state {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
}
.save-bar.is-dirty .save-state {
  color: var(--primary-deep);
}
.save-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

/* ---------- 断点 ---------- */
@media (max-width: 768px) {
  .avatar-top {
    flex-direction: column;
    align-items: flex-start;
    gap: 14px;
  }
  .avatar-controls {
    align-items: flex-start;
  }
  .preset-grid {
    --preset-size: 48px;
  }
}
@media (max-width: 640px) {
  /* 给悬浮 Dock + 凸出的发布圆钮让位（Dock 顶边 68 + 间隙），保存条别被压住 */
  .save-bar {
    bottom: calc(82px + env(safe-area-inset-bottom));
  }
  .save-state {
    font-size: 11px;
  }
  .contact-readonly {
    grid-template-columns: minmax(0, 1fr);
  }
}
@media (prefers-reduced-motion: reduce) {
  .preset-item,
  .cover-preview.is-color,
  .save-bar {
    transition: none;
  }
}
</style>
