<!-- 基础信息设置组件 -->
<template>
  <div class="basic-info-settings">
    <!-- 图片裁切弹窗 -->
    <div 
      v-if="showCropperModal" 
      class="modal fade show"
      style="display: block;"
      @click.self="closeCropperModal"
    >
      <div class="modal-dialog modal-lg">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">裁切头像</h5>
            <button type="button" class="btn-close" @click="closeCropperModal"></button>
          </div>
          <div class="modal-body">
            <div class="cropper-wrapper">
              <img ref="cropperImage" :src="cropImageSrc" alt="裁切图片">
            </div>
            <div class="cropper-toolbar mt-3">
              <button type="button" class="btn btn-sm btn-outline-secondary" @click="rotateImage(-90)">
                <i class="bi bi-arrow-counterclockwise"></i>
              </button>
              <button type="button" class="btn btn-sm btn-outline-secondary" @click="rotateImage(90)">
                <i class="bi bi-arrow-clockwise"></i>
              </button>
              <button type="button" class="btn btn-sm btn-outline-secondary" @click="resetCropper">
                <i class="bi bi-arrow-counterclockwise me-1"></i>重置
              </button>
              <div class="cropper-zoom-controls ms-2">
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="zoomImage(-0.1)">
                  <i class="bi bi-zoom-out"></i>
                </button>
                <button type="button" class="btn btn-sm btn-outline-secondary" @click="zoomImage(0.1)">
                  <i class="bi bi-zoom-in"></i>
                </button>
              </div>
            </div>
          </div>
          <div class="modal-footer">
            <button type="button" class="btn wx-btn-outline" @click="closeCropperModal">取消</button>
            <button 
              type="button" 
              class="btn wx-btn-gradient" 
              @click="uploadCroppedImage"
              :disabled="uploading"
            >
              <i v-if="uploading" class="bi bi-arrow-clockwise animate-spin me-1"></i>
              {{ uploading ? '上传中...' : '上传头像' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="row">
      <div class="col-md-4 mb-4">
        <div class="card wx-card">
          <div class="card-body">
            <div class="skeleton-loader" style="height: 20px; width: 60%; margin-bottom: 1.5rem;"></div>
            <div class="text-center mb-4">
              <div class="skeleton-loader" style="width: 120px; height: 120px; border-radius: 50%; margin: 0 auto 1rem;"></div>
              <div class="skeleton-loader" style="height: 36px; width: 80%; margin: 0 auto 0.5rem;"></div>
              <div class="skeleton-loader" style="height: 14px; width: 90%; margin: 0 auto;"></div>
            </div>
          </div>
        </div>
      </div>
      <div class="col-md-8">
        <div class="card wx-card">
          <div class="card-body">
            <div class="skeleton-loader" style="height: 20px; width: 60%; margin-bottom: 1.5rem;"></div>
            <div class="space-y-4">
              <div class="skeleton-loader" style="height: 40px; width: 100%;"></div>
              <div class="space-y-2">
                <div class="skeleton-loader" style="height: 16px; width: 30%;"></div>
                <div class="d-flex gap-4">
                  <div class="skeleton-loader" style="height: 20px; width: 20%;"></div>
                  <div class="skeleton-loader" style="height: 20px; width: 20%;"></div>
                  <div class="skeleton-loader" style="height: 20px; width: 20%;"></div>
                </div>
              </div>
              <div class="space-y-2">
                <div class="skeleton-loader" style="height: 16px; width: 30%;"></div>
                <div class="skeleton-loader" style="height: 100px; width: 100%;"></div>
              </div>
              <div class="d-flex gap-2">
                <div class="skeleton-loader" style="height: 40px; width: 30%;"></div>
                <div class="skeleton-loader" style="height: 40px; width: 30%;"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="row">
      <!-- 头像设置 -->
      <div class="col-md-4 mb-4">
        <div class="card wx-card">
          <div class="card-body text-center">
            <h6 class="card-title mb-3">头像设置</h6>
            <div class="avatar-upload-container mb-3">
              <div class="avatar-preview">
                <i-avatar-frame
                  :src="formData.avatar || defaultAvatar"
                  :frame="formData.frame"
                  :alt="formData.nickname || '用户头像'"
                  size="100px"
                  :frame-scale="1.5"
                />
              </div>
            </div>
            <div class="d-flex flex-column gap-2 mb-3">
              <button 
                type="button" 
                class="btn btn-sm wx-btn-gradient"
                @click="handleUploadAvatar"
                :disabled="uploading"
              >
                <i class="bi bi-upload me-2"></i>
                {{ uploading ? '上传中...' : '上传头像' }}
              </button>
              <button 
                type="button" 
                class="btn btn-sm wx-btn-outline"
                @click="showAvatarUrlInput = !showAvatarUrlInput"
              >
                <i class="bi bi-link-45deg me-2"></i>自定义链接
              </button>
            </div>
            <div v-if="showAvatarUrlInput" class="mb-3">
              <div class="input-group">
                <span class="input-group-text"><i class="bi bi-globe"></i></span>
                <input 
                  type="text" 
                  v-model="customAvatarUrl"
                  class="form-control"
                  placeholder="请输入头像图片链接"
                  @keyup.enter="applyCustomAvatar"
                >
                <button 
                  type="button" 
                  class="btn btn-sm wx-btn-outline"
                  @click="applyCustomAvatar"
                  :disabled="!customAvatarUrl.trim()"
                >
                  应用
                </button>
              </div>
            </div>

            <!-- 头像框选择 -->
            <div class="default-avatars mb-3">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span class="text-muted small"><i class="bi bi-puzzle me-1"></i>头像框</span>
                <button
                  v-if="formData.frame"
                  type="button"
                  class="btn btn-sm btn-outline-secondary px-2 py-0"
                  @click="formData.frame = ''"
                  style="font-size: 0.75rem;"
                >
                  <i class="bi bi-x-circle me-1"></i>移除
                </button>
              </div>
              <div class="default-avatars-grid frame-grid">
                <button
                  v-for="(url, idx) in PRESET_FRAMES"
                  :key="idx"
                  type="button"
                  class="default-avatar-item frame-item"
                  :class="{ 'is-selected': formData.frame === url }"
                  @click="formData.frame = formData.frame === url ? '' : url"
                  :title="`头像框 ${idx + 1}`"
                >
                  <img :src="url" :alt="`头像框 ${idx + 1}`" class="default-avatar-img">
                  <i v-if="formData.frame === url" class="bi bi-check-circle-fill default-avatar-check"></i>
                </button>
              </div>
            </div>

            <!-- 默认头像选择 -->
            <div class="default-avatars mb-3">
              <div class="d-flex align-items-center justify-content-between mb-2">
                <span class="text-muted small"><i class="bi bi-grid-3x3-gap me-1"></i>默认头像</span>
              </div>
              <div class="default-avatars-grid">
                <button
                  v-for="(url, idx) in DEFAULT_AVATARS"
                  :key="idx"
                  type="button"
                  class="default-avatar-item"
                  :class="{ 'is-selected': formData.avatar === url }"
                  @click="selectDefaultAvatar(url)"
                  :title="`默认头像 ${idx + 1}`"
                >
                  <img :src="url" :alt="`默认头像 ${idx + 1}`" class="default-avatar-img">
                  <i v-if="formData.avatar === url" class="bi bi-check-circle-fill default-avatar-check"></i>
                </button>
              </div>
            </div>

            <button 
              v-if="formData.avatar"
              type="button" 
              class="btn btn-sm btn-outline-danger w-100"
              @click="removeAvatar"
            >
              <i class="bi bi-trash me-2"></i>移除头像
            </button>
            <p class="text-muted small mt-3 mb-0">
              <i class="bi bi-info-circle me-1"></i>
              支持 JPG、PNG、GIF 格式，建议 1:1 比例
              同时上传头像后请点击「保存修改」按钮才能正常应用头像否则无法显示！！
              点击保存修改后页面将自动刷新，头像即可正常显示。
            </p>
          </div>
        </div>
      </div>

      <!-- 基本信息表单 -->
      <div class="col-md-8">
        <div class="card wx-card">
          <div class="card-body">
            <h6 class="card-title mb-3">个人信息</h6>
            <form @submit.prevent="updateBasicInfo">
              <div class="mb-3">
                <label for="nickname" class="form-label">昵称</label>
                <input 
                  type="text" 
                  id="nickname" 
                  v-model="formData.nickname" 
                  class="form-control"
                  placeholder="请输入昵称"
                  maxlength="20"
                >
              </div>

              <div class="mb-3">
                <label class="form-label">性别</label>
                <div class="d-flex gap-4">
                  <div class="form-check">
                    <input 
                      type="radio" 
                      id="gender-boy" 
                      v-model.number="formData.gender" 
                      :value="1"
                      class="form-check-input"
                    >
                    <label for="gender-boy" class="form-check-label">男</label>
                  </div>
                  <div class="form-check">
                    <input 
                      type="radio" 
                      id="gender-girl" 
                      v-model.number="formData.gender" 
                      :value="2"
                      class="form-check-input"
                    >
                    <label for="gender-girl" class="form-check-label">女</label>
                  </div>
                  <div class="form-check">
                    <input 
                      type="radio" 
                      id="gender-none" 
                      v-model.number="formData.gender" 
                      :value="0"
                      class="form-check-input"
                    >
                    <label for="gender-none" class="form-check-label">保密</label>
                  </div>
                </div>
              </div>

              <div class="mb-3">
                <label class="form-label">头衔</label>
                <input
                  type="text"
                  v-model="formData.title"
                  class="form-control"
                  placeholder="请选择预设头衔"
                  disabled
                >
                <div class="preset-titles mt-2">
                  <button
                    v-for="title in PRESET_TITLES"
                    :key="title"
                    type="button"
                    class="preset-title-item title-badge"
                    :class="[getTitleColorClass(title), { 'is-selected': formData.title === title }]"
                    @click="selectPresetTitle(title)"
                  >
                    {{ title }}
                  </button>
                </div>
                <div class="text-muted small mt-1">
                  <i class="bi bi-info-circle me-1"></i>
                  点击预设头衔快速应用，保存后生效
                </div>
              </div>

              <div class="mb-3">
                <label for="description" class="form-label">个人简介</label>
                <textarea 
                  id="description" 
                  v-model="formData.description" 
                  class="form-control"
                  placeholder="请输入个人简介"
                  rows="4"
                  maxlength="200"
                ></textarea>
                <div class="text-end text-muted small mt-1">
                  {{ formData.description.length }}/200
                </div>
              </div>

              <div class="mb-3">
                <label class="form-label">联系方式</label>
                <div class="row">
                  <div class="col-md-6 mb-2">
                    <div class="input-group input-group-contact">
                      <span class="input-group-text"><i class="bi bi-telephone"></i></span>
                      <input 
                        type="text" 
                        class="form-control" 
                        :value="userContact.phone || '未设置'" 
                        disabled
                      >
                    </div>
                  </div>
                  <div class="col-md-6">
                    <div class="input-group input-group-contact">
                      <span class="input-group-text"><i class="bi bi-envelope"></i></span>
                      <input 
                        type="email" 
                        class="form-control" 
                        :value="userContact.email || '未设置'" 
                        disabled
                      >
                    </div>
                  </div>
                </div>
                <div class="text-muted small mt-2">
                  <i class="bi bi-info-circle me-1"></i>
                  手机号和邮箱可在"联系方式"标签页中修改
                </div>
              </div>

              <div class="d-flex gap-2">
                <button 
                  type="submit" 
                  class="btn wx-btn-gradient"
                  :disabled="saving"
                >
                  <i class="bi bi-save me-2"></i>
                  {{ saving ? '保存中...' : '保存修改' }}
                </button>
                <button 
                  type="button" 
                  class="btn wx-btn-outline"
                  @click="resetForm"
                  :disabled="saving"
                >
                  <i class="bi bi-arrow-counterclockwise me-2"></i>重置
                </button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, watch, nextTick } from 'vue'
import { request, checkFileType } from '@/utils/network'
import { toast, getSync, getTitleColorClass } from '@/utils/app'
import { useCommStore } from '@/store/comm'
import defaultAvatar from '@/assets/img/avatar.png'
import iAvatarFrame from '@/comps/custom/i-avatar-frame.vue'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const DEFAULT_AVATARS = [
  'https://img.zhuxu.asia/tx/1.png',
  'https://img.zhuxu.asia/tx/2.png',
  'https://img.zhuxu.asia/tx/3.png',
  'https://img.zhuxu.asia/tx/4.png',
  'https://img.zhuxu.asia/tx/5.png'
]

const PRESET_FRAMES = [
  'https://img.zhuxu.asia/txk/1.gif',
  'https://img.zhuxu.asia/txk/2.png',
  'https://img.zhuxu.asia/txk/3.gif',
  'https://img.zhuxu.asia/txk/4.png',
  'https://img.zhuxu.asia/txk/5.gif',
  'https://img.zhuxu.asia/txk/6.png',
  'https://img.zhuxu.asia/txk/7.gif',
  'https://img.zhuxu.asia/txk/8.png',
  'https://img.zhuxu.asia/txk/9.gif',
  'https://img.zhuxu.asia/txk/10.png',
  'https://img.zhuxu.asia/txk/11.gif',
  'https://img.zhuxu.asia/txk/12.png',
  'https://img.zhuxu.asia/txk/13.gif',
  'https://img.zhuxu.asia/txk/14.png',
  'https://img.zhuxu.asia/txk/15.gif',
  'https://img.zhuxu.asia/txk/16.png',
  'https://img.zhuxu.asia/txk/17.gif',
  'https://img.zhuxu.asia/txk/18.png',
  'https://img.zhuxu.asia/txk/19.gif',
  'https://img.zhuxu.asia/txk/20.png'
]

const PRESET_TITLES = [
  '掌门',
  '长老',
  '护法',
  '内门弟子',
  '外门弟子',
  '炼气修士',
  '筑基修士',
  '结丹修士',
  '元婴老祖',
  '化神大能'
]

const store = useCommStore()
const loading = ref(false)
const saving = ref(false)
const uploading = ref(false)
const showAvatarUrlInput = ref(false)
const customAvatarUrl = ref('')

// 裁切相关
const showCropperModal = ref(false)
const cropperImage = ref(null)
const cropImageSrc = ref('')
const selectedFile = ref(null)
let cropperInstance = null

const formData = reactive({
  id: '',
  nickname: '',
  gender: 0,
  description: '',
  avatar: '',
  title: '',
  frame: ''
})

const userContact = reactive({
  phone: '',
  email: ''
})

const originalData = reactive({})

const handleUploadAvatar = () => {
  if (uploading.value) return

  const input = document.createElement('input')
  input.type = 'file'
  input.accept = 'image/*'

  input.addEventListener('change', async () => {
    if (!input.files || input.files.length === 0) {
      return
    }

    const file = input.files[0]
    
    // 验证文件大小（最大 10MB）
    if (file.size > 10 * 1024 * 1024) {
      toast.warning('图片大小不能超过 10MB')
      return
    }

    // 验证文件类型
    if (!['image/jpeg', 'image/png', 'image/gif', 'image/webp'].includes(file.type)) {
      toast.warning('请选择 JPG、PNG、GIF 或 WebP 格式的图片')
      return
    }

    selectedFile.value = file
    
    // 读取图片并显示裁切弹窗
    const reader = new FileReader()
    reader.onload = async (e) => {
      cropImageSrc.value = e.target.result
      showCropperModal.value = true
      
      // 固定 body 防止滚动
      const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth
      document.body.style.overflow = 'hidden'
      document.body.style.paddingRight = scrollbarWidth > 0 ? `${scrollbarWidth}px` : ''
      
      await nextTick()
      
      // 初始化裁切器
      if (cropperImage.value) {
        cropperInstance = new Cropper(cropperImage.value, {
          aspectRatio: 1,
          viewMode: 1,
          dragMode: 'move',
          autoCropArea: 0.8,
          restore: false,
          guides: true,
          center: true,
          highlight: true,
          cropBoxMovable: true,
          cropBoxResizable: true,
          toggleDragModeOnDblclick: true,
          minContainerWidth: 400,
          minContainerHeight: 400,
          minCropBoxWidth: 100,
          minCropBoxHeight: 100,
        })
      }
    }
    reader.readAsDataURL(file)
    
    input.value = ''
  })

  input.click()
}

const rotateImage = (degree) => {
  if (cropperInstance) {
    cropperInstance.rotate(degree)
  }
}

const zoomImage = (ratio) => {
  if (cropperInstance) {
    cropperInstance.zoom(ratio)
  }
}

const resetCropper = () => {
  if (cropperInstance) {
    cropperInstance.reset()
  }
}

const closeCropperModal = () => {
  showCropperModal.value = false
  if (cropperInstance) {
    cropperInstance.destroy()
    cropperInstance = null
  }
  cropImageSrc.value = ''
  selectedFile.value = null
  
  // 移除 body 固定
  document.body.style.overflow = ''
  document.body.style.paddingRight = ''
}

const uploadCroppedImage = async () => {
  if (!cropperInstance || uploading.value) return
  
  uploading.value = true
  
  try {
    // 获取裁切后的图片
    const canvas = cropperInstance.getCroppedCanvas({
      width: 200,
      height: 200,
      minWidth: 100,
      minHeight: 100,
      maxWidth: 4096,
      maxHeight: 4096,
      fillColor: '#fff',
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high',
    })
    
    // 将 canvas 转换为 Blob
    const blob = await new Promise((resolve) => {
      canvas.toBlob(resolve, 'image/jpeg', 0.9)
    })
    
    // 创建 FormData 上传
    const params = new FormData()
    params.append('file', blob, 'avatar.jpg')
    
    await checkFileType(['avatar.jpg'])
    
    const { code, msg, data } = await request.post('/api/attachment/batch', params, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
    
    if (code === 200) {
      formData.avatar = data.results?.[0]?.full_url || ''
      toast.success('头像上传成功，请点击"保存修改"完成更新')
      closeCropperModal()
    } else {
      toast.error(msg || '上传失败')
    }
  } catch (error) {
    console.error('上传失败:', error)
    toast.error('上传失败，请稍后重试')
  } finally {
    uploading.value = false
  }
}

const applyCustomAvatar = () => {
  const url = customAvatarUrl.value.trim()
  if (!url) {
    toast.warning('请输入头像链接')
    return
  }

  if (!/^https?:\/\//.test(url)) {
    toast.warning('请输入有效的图片链接（以 http:// 或 https:// 开头）')
    return
  }

  formData.avatar = url
  showAvatarUrlInput.value = false
  customAvatarUrl.value = ''
  toast.success('头像链接已应用，请点击"保存修改"完成更新')
}

const selectDefaultAvatar = (url) => {
  formData.avatar = url
  showAvatarUrlInput.value = false
  customAvatarUrl.value = ''
  toast.success('已选择默认头像，请点击"保存修改"完成更新')
}

const selectPresetTitle = (title) => {
  formData.title = title
  toast.success('头衔已应用，请点击"保存修改"完成更新')
}

const removeAvatar = () => {
  formData.avatar = ''
  toast.info('头像已移除，请点击"保存修改"完成更新')
}

const updateBasicInfo = async () => {
  if (saving.value) return

  saving.value = true
  try {
    // 性别映射：1->boy, 2->girl, 0->空
    const genderMap = {
      1: 'boy',
      2: 'girl',
      0: ''
    }

    // 合并既有 json，只更新 frame（避免覆盖 website 等其他字段）
    const loginUser = store.getLogin?.user || {}
    const currentJson = (loginUser.json && typeof loginUser.json === 'object') ? { ...loginUser.json } : {}
    const mergedJson = {
      ...currentJson,
      frame: formData.frame || ''
    }
    
    const payload = {
      id: formData.id,
      nickname: formData.nickname,
      description: formData.description,
      avatar: formData.avatar,
      title: formData.title,
      json: mergedJson
    }
    
    // 只有选择了性别才传递gender字段
    if (formData.gender !== 0) {
      payload.gender = genderMap[formData.gender]
    }

    const res = await request.put('/api/users/update', payload)

    if (res.code === 200) {
      toast.success('用户信息更新成功')
      await syncUserInfo()
      Object.assign(originalData, { ...formData })
      /* setTimeout(() => {
        window.location.reload()
      }, 1500) */
    } else {
      toast.error(res.msg || '用户信息更新失败')
    }
  } catch (error) {
    console.error('更新失败:', error)
    toast.error('网络错误，请稍后重试')
  } finally {
    saving.value = false
  }
}

const syncUserInfo = async () => {
  try {
    await store.checkLoginState()
  } catch (error) {
    console.error('同步用户信息失败:', error)
  }
}

const resetForm = () => {
  Object.assign(formData, { ...originalData })
  showAvatarUrlInput.value = false
  customAvatarUrl.value = ''
}

const fetchUserInfo = () => {
  try {
    const loginState = store.getLogin
    const userInfo = loginState.user
    if (userInfo && Object.keys(userInfo).length > 0) {
      formData.id = userInfo.id
      formData.nickname = userInfo.nickname || ''
      
      // 性别映射：boy->1, girl->2, 其他->0
      const genderReverseMap = {
        'boy': 1,
        'girl': 2
      }
      if (userInfo.gender === 'boy' || userInfo.gender === 'girl') {
        formData.gender = genderReverseMap[userInfo.gender]
      } else if (typeof userInfo.gender === 'number') {
        formData.gender = userInfo.gender
      } else {
        formData.gender = 0
      }
      
      formData.description = userInfo.description || ''
      formData.avatar = userInfo.avatar || ''
      formData.title = userInfo.title || ''
      formData.frame = userInfo.json?.frame || ''
      Object.assign(originalData, { ...formData })
      userContact.phone = userInfo.phone || ''
      userContact.email = userInfo.email || ''
    }
  } catch (error) {
    console.error('获取用户信息失败:', error)
  }
}

onMounted(() => {
  fetchUserInfo()
})

// 监听 store 中用户信息变化，确保 token 校验完成后能正确回填表单
watch(
  () => store.login?.user,
  (newUser) => {
    if (newUser && Object.keys(newUser).length > 0) {
      fetchUserInfo()
    }
  },
  { immediate: true }
)
</script>

<style scoped>
/* 默认头像选择网格 */
.default-avatars-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
}

.default-avatar-item {
  position: relative;
  aspect-ratio: 1;
  padding: 0;
  background: none;
  border: 2px solid var(--bs-border-color);
  border-radius: 50%;
  cursor: pointer;
  overflow: hidden;
  transition: var(--wx-transition);
  display: flex;
  align-items: center;
  justify-content: center;
}

.default-avatar-item:hover {
  border-color: var(--bs-primary);
  transform: scale(1.08);
  box-shadow: var(--wx-shadow-md);
}

.default-avatar-item.is-selected {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.25);
}

.default-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  background: var(--bs-secondary-bg);
}

.default-avatar-check {
  position: absolute;
  top: -2px;
  right: -2px;
  font-size: 20px;
  color: var(--bs-primary);
  background: var(--bs-body-bg);
  border-radius: 50%;
  line-height: 1;
}

.form-control {
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
  padding: 0.625rem 0.875rem;
  font-size: 0.9rem;
  background: var(--bs-body-bg);
  color: var(--bs-body-color);
  border: 1px solid var(--bs-border-color);
}

.form-control:focus {
  border-color: var(--bs-primary);
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.15);
}

.form-label {
  font-size: 0.875rem;
  font-weight: 500;
  margin-bottom: 0.375rem;
  color: var(--bs-body-color);
}

.default-avatars-grid.frame-grid {
  grid-template-columns: repeat(5, 1fr);
}
.preset-titles {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.preset-title-item {
  padding: 0.3rem 0.75rem;
  font-size: 0.8125rem;
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  color: var(--bs-body-color);
  cursor: pointer;
  transition: var(--wx-transition);
}

.preset-title-item:hover {
  border-color: var(--bs-primary);
  color: var(--bs-primary);
  transform: translateY(-1px);
}

.preset-title-item.is-selected {
  background: var(--bs-primary);
  border-color: var(--bs-primary);
  color: #fff;
}

.skeleton-loader {
  background: linear-gradient(90deg, var(--bs-tertiary-bg) 25%, var(--bs-border-color) 50%, var(--bs-tertiary-bg) 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
  border-radius: var(--wx-radius-sm);
}

@keyframes skeleton-loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@media (max-width: 768px) {
  .card-body {
    padding: 1rem;
  }

  .col-md-4,
  .col-md-8 {
    width: 100%;
  }

  .form-control {
    padding: 0.5rem 0.625rem;
    font-size: 0.8125rem;
  }

  .btn {
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
  }

  .d-flex.gap-2 {
    flex-direction: column;
    gap: 0.5rem !important;
  }

  .d-flex.gap-2 .btn {
    width: 100%;
  }

  .default-avatars-grid {
    grid-template-columns: repeat(5, 1fr);
    gap: 6px;
  }

  .default-avatar-check {
    font-size: 16px;
  }
}

/* 裁切器样式 */
.cropper-wrapper {
  width: 100%;
  height: 350px;
  background: var(--bs-tertiary-bg);
  border-radius: var(--wx-radius-md);
  overflow: hidden;
}

.cropper-wrapper img {
  display: block;
  max-width: 100%;
}

.cropper-toolbar {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.cropper-zoom-controls {
  display: flex;
  gap: 4px;
}

.animate-spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.cropper-wrapper .cropper-crop-box {
  border: 2px solid #39f;
}

.cropper-wrapper .cropper-view-box {
  border-radius: 50%;
}

.cropper-wrapper .cropper-face {
  background-color: transparent;
}

.cropper-wrapper .cropper-modal {
  background-color: rgba(0, 0, 0, 0.5);
}

@media (max-width: 576px) {
  .cropper-wrapper {
    height: 250px;
  }
}

/* 头衔颜色 - 修仙体系 */
.title-badge {
  color: #fff;
  font-weight: 500;
  border: none;
}
.title-zhangmen { background: linear-gradient(135deg, #f6d365, #fda085) !important; color: #5a3e00 !important; }
.title-zhanglao { background: #8e44ad !important; }
.title-hufa { background: #c0392b !important; }
.title-neimen { background: #2980b9 !important; }
.title-waimen { background: #16a085 !important; }
.title-lianqi { background: #27ae60 !important; }
.title-zhuji { background: #8bc34a !important; color: #1a3d00 !important; }
.title-jiedan { background: #e67e22 !important; }
.title-yuanying { background: #6c5ce7 !important; }
.title-huashen { background: #00b894 !important; }
.title-default { background: #6c757d !important; }

.preset-title-item.is-selected {
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.5);
  transform: translateY(-1px);
}
</style>
