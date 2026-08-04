<!-- 用户设置页面 -->
<template>
  <div class="card wx-card mt-2">
    <div class="card-header">
      <div class="d-flex justify-content-between align-items-center">
        <h5 class="card-title mb-0 d-flex align-items-center gap-2 fw-semibold"><i class="bi bi-gear-fill"></i>用户设置</h5>
      </div>
    </div>
    <div class="card-body">
        <!-- 导航标签 -->
        <div class="user-tabs-container mb-4">
          <ul class="nav nav-tabs user-nav-tabs" id="userSettingsTabs" role="tablist">
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link active" 
                id="basic-info-tab" 
                data-bs-toggle="tab" 
                data-bs-target="#basic-info" 
                type="button" 
                role="tab" 
                aria-controls="basic-info" 
                aria-selected="true"
              >
                基础信息
              </button>
            </li>
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link" 
                id="account-security-tab" 
                data-bs-toggle="tab" 
                data-bs-target="#account-security" 
                type="button" 
                role="tab" 
                aria-controls="account-security" 
                aria-selected="false"
              >
                账号安全
              </button>
            </li>
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link" 
                id="contact-info-tab" 
                data-bs-toggle="tab" 
                data-bs-target="#contact-info" 
                type="button" 
                role="tab" 
                aria-controls="contact-info" 
                aria-selected="false"
              >
                联系方式
              </button>
            </li>
          </ul>
        </div>

        <!-- 标签内容 -->
        <div class="tab-content" id="userSettingsTabsContent">
          <!-- 基础信息 -->
          <div 
            class="tab-pane fade show active" 
            id="basic-info" 
            role="tabpanel" 
            aria-labelledby="basic-info-tab"
          >
            <BasicInfoSettings />
          </div>
          
          <!-- 账号安全 -->
          <div 
            class="tab-pane fade" 
            id="account-security" 
            role="tabpanel" 
            aria-labelledby="account-security-tab"
          >
            <AccountSecuritySettings />
          </div>
          
          <!-- 联系方式 -->
          <div 
            class="tab-pane fade" 
            id="contact-info" 
            role="tabpanel" 
            aria-labelledby="contact-info-tab"
          >
            <ContactInfoSettings />
          </div>
        </div>
      </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import BasicInfoSettings from '@/comps/user/settings/basic-info.vue'
import AccountSecuritySettings from '@/comps/user/settings/account-security.vue'
import ContactInfoSettings from '@/comps/user/settings/contact-info.vue'
import { usePageTitle } from '@/utils/app'

// 使用页面标题管理
const { setDynamicTitle } = usePageTitle({
  staticTitle: '用户设置',
  defaultTitle: '用户设置'
})

// 组件挂载时的逻辑
onMounted(async () => {
  // 路由守卫已经做了登录校验，这里不需要重复检查
})
</script>

<style scoped>
/* 页面卡片头部 */
.card-header {
  background: var(--bs-body-bg);
  border-bottom: 1px solid var(--bs-border-color);
  padding: 1rem 1.25rem;
}

.card-title {
  font-size: 1.125rem;
  color: var(--bs-body-color);
}

.card-title i {
  color: var(--bs-primary);
}

/* 导航标签：现代胶囊风格 */
.user-nav-tabs {
  border: none;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.user-nav-tabs .nav-item {
  margin-bottom: 0;
}

.user-nav-tabs .nav-link {
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  padding: 0.5rem 1.25rem;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--bs-secondary-color);
  background: var(--bs-body-bg);
  transition: var(--wx-transition);
}

.user-nav-tabs .nav-link:hover {
  border-color: var(--bs-primary);
  color: var(--bs-primary);
  background: rgba(var(--bs-primary-rgb), 0.05);
  transform: translateY(-1px);
}

.user-nav-tabs .nav-link.active {
  background: var(--wx-gradient-primary);
  color: #fff;
  border-color: transparent;
  box-shadow: 0 4px 12px rgba(var(--bs-primary-rgb), 0.25);
}

.user-nav-tabs .nav-link:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(var(--bs-primary-rgb), 0.15);
}

@media (max-width: 576px) {
  .user-nav-tabs .nav-link {
    padding: 0.5rem 0.875rem;
    font-size: 0.85rem;
  }
}
</style>
