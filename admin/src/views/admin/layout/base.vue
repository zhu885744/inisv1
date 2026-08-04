<template>
    <el-container class="app-wrapper" :class="{ 'sidebar-mini': state.sidebarCollapsed }">
        <el-aside
            :width="state.sidebarCollapsed ? '64px' : '220px'"
            class="main-sidebar"
        >
            <div class="sidebar-header">
                <div class="logo" @click="method.push('/admin')">
                    <el-icon><Menu /></el-icon>
                    <span v-show="!state.sidebarCollapsed" class="logo-title">管理后台</span>
                </div>
            </div>

            <nav class="sidebar-menu">
                <el-menu
                    :default-active="activeMenu"
                    mode="vertical"
                    :collapse="state.sidebarCollapsed"
                    unique-opened
                    @select="handleMenuSelect"
                    class="el-menu-vertical"
                >
                    <el-menu-item index="/admin">
                            <el-icon :size="16">
                                <House />
                            </el-icon>
                            <span>首页</span>
                        </el-menu-item>
                        <template v-for="(item, index) in state.menu" :key="index">
                        <el-sub-menu v-if="item.children?.length" :index="item.name">
                            <template #title>
                                <el-icon :size="16" v-if="item.icon" v-html="item.icon" />
                                <el-icon :size="16" v-else>
                                    <component :is="getIcon(item)" />
                                </el-icon>
                                <span>{{ item.label }}</span>
                            </template>
                            <el-menu-item
                                v-for="(child, key) in item.children"
                                :key="key"
                                :index="child.path"
                            >
                                <el-icon :size="12" v-if="child.isSvg && child.icon" v-html="child.icon" />
                                <i-svg v-else-if="child.icon" :name="child.icon" size="14px" />
                                <el-icon :size="12" v-else>
                                    <component :is="componentMap['Dot']" />
                                </el-icon>
                                <span>{{ child.label }}</span>
                            </el-menu-item>
                        </el-sub-menu>
                        <el-menu-item v-else :index="item.path">
                            <el-icon :size="16" v-if="item.icon" v-html="item.icon" />
                            <el-icon :size="16" v-else>
                                <component :is="getIcon(item)" />
                            </el-icon>
                            <span>{{ item.label }}</span>
                        </el-menu-item>
                    </template>
                </el-menu>
            </nav>
        </el-aside>

        <el-container
            class="main-content-area"
            :style="{ marginLeft: state.sidebarCollapsed ? '64px' : '220px' }"
        >
            <el-header class="top-header">
                <div class="header-left">
                    <button
                        class="sidebar-toggle"
                        @click="state.sidebarCollapsed = !state.sidebarCollapsed"
                    >
                        <el-icon :size="18">
                            <DArrowLeft />
                        </el-icon>
                    </button>
                    <el-breadcrumb separator="/" class="breadcrumb">
                        <el-breadcrumb-item :to="'/admin'">
                            <span>首页</span>
                        </el-breadcrumb-item>
                        <template v-for="item in breadcrumbList" :key="item.path">
                            <el-breadcrumb-item :to="item.path">{{ item.label }}</el-breadcrumb-item>
                        </template>
                    </el-breadcrumb>
                </div>

                <div class="header-right">
                    <div class="header-actions">
                        <el-dropdown trigger="click" class="user-menu">
                            <div class="user-info">
                                <el-avatar :src="store.comm.getLogin.user?.avatar" size="small" class="user-avatar" />
                                <span v-show="!state.sidebarCollapsed" class="user-name">{{ store.comm.getLogin.user?.nickname }}</span>
                                <el-icon :size="12">
                                    <component :is="componentMap['ChevronDown']" />
                                </el-icon>
                            </div>
                            <template #dropdown>
                                <el-dropdown-menu>
                                    <el-dropdown-item @click="store.comm.logout('/')">
                                        <span>退出登录</span>
                                    </el-dropdown-item>
                                </el-dropdown-menu>
                            </template>
                        </el-dropdown>
                    </div>
                </div>
            </el-header>

            <el-main class="content-wrapper">
                <router-view v-slot="{ Component }">
                    <transition name="fade" mode="out-in">
                        <component :is="Component" />
                    </transition>
                </router-view>
            </el-main>

            <el-footer class="main-footer">
                <div class="footer-content">
                    <span>Copyright © {{ currentYear }} 管理后台</span>
                    <span class="footer-divider">|</span>
                    <span>Powered by inis</span>
                </div>
            </el-footer>
        </el-container>

        <upgrade-page />
        <nav-component />
    </el-container>
</template>

<script setup>
import { reactive, computed, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { list as MenuList } from '{src}/utils/menu'
import { push } from '{src}/utils/route'
import upgradePage from '{src}/comps/upgrade/page.vue'
import navComponent from '{src}/views/admin/layout/nav.vue'
import { useCommStore } from '{src}/store/comm'
const route = useRoute()
const store = {
    comm: useCommStore(),
}

const state = reactive({
    sidebarCollapsed: false,
    menu: [],
    searchText: '',
    notificationCount: 0,
    version: '1.0.0',
})

const componentMap = new Proxy({}, { get: (_, name) => name })

const currentYear = new Date().getFullYear()

const activeMenu = computed(() => route.path)

const breadcrumbList = computed(() => {
    const list = []
    for (const menu of state.menu) {
        if (menu.children?.length) {
            const child = menu.children.find(c => c.path === route.path)
            if (child) {
                list.push({ label: menu.label, path: menu.path })
                list.push({ label: child.label, path: child.path })
                break
            }
        }
    }
    return list
})

const getIcon = (item) => {
    const iconMap = {
        'create': 'PenTool',
        'manage': 'LayoutGrid',
        'security': 'ShieldCheck',
    }
    return componentMap[iconMap[item.name] || 'FileText']
}

const handleMenuSelect = (index) => {
    if (index.startsWith('/')) push(index)
}

const method = {
    push: params => push(params),
}

onMounted(async () => {
    state.menu = await MenuList()
})

nextTick(() => {
    const body = document.querySelector('body')
    body.setAttribute('data-layout', 'admin')
    body.setAttribute('inis-theme', 'white')
    body.classList.add('user-select-none')
})
</script>

<style lang="scss" scoped>
.app-wrapper {
    min-height: 100vh;
    background: #f0f2f5;
}

.main-sidebar {
    background: #ffffff;
    color: #595959;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #e8e8e8;
    transition: width 0.3s ease;
    overflow: hidden;
    position: fixed;
    left: 0;
    top: 0;
    height: 100vh;
    z-index: 100;
}

.sidebar-header {
    height: 56px;
    padding: 0 16px;
    display: flex;
    align-items: center;
    border-bottom: 1px solid #e8e8e8;
    background: #ffffff;
    flex-shrink: 0;
}

.logo {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
}

.logo:hover {
    opacity: 0.8;
}

.logo-title {
    font-size: 16px;
    font-weight: 600;
    color: #262626;
    white-space: nowrap;
}

.sidebar-menu {
    flex: 1;
    padding: 8px 0;
    overflow-y: auto;
}

.el-menu-vertical {
    border: none;
    background: transparent;
}

.el-menu-vertical :deep(.el-menu-item),
.el-menu-vertical :deep(.el-sub-menu__title) {
    color: #595959;
    height: 44px;
    line-height: 44px;
    margin: 2px 8px;
    border-radius: 4px;
    padding: 0 12px;
    transition: all 0.2s ease;
    font-size: 14px;
}

.el-menu-vertical :deep(.el-menu-item i),
.el-menu-vertical :deep(.el-sub-menu__title i),
.el-menu-vertical :deep(.el-menu-item svg),
.el-menu-vertical :deep(.el-sub-menu__title svg) {
    margin-right: 8px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
}

.el-menu-vertical :deep(.el-menu-item:hover),
.el-menu-vertical :deep(.el-sub-menu__title:hover) {
    background: #f5f5f5;
    color: #262626;
}

.el-menu-vertical :deep(.el-menu-item.is-active) {
    background: #e6f7ff;
    color: #1890ff;
    box-shadow: none;
    box-shadow: inset 3px 0 0 #1890ff;
}

.el-menu-vertical :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
    color: #1890ff;
}

.el-menu-vertical :deep(.el-sub-menu .el-menu-item) {
    padding-left: 36px;
    margin: 2px 8px;
    border-radius: 4px;
    font-size: 13px;
    height: 40px;
    line-height: 40px;
}

.el-menu-vertical :deep(.el-sub-menu .el-menu-item.is-active) {
    background: #e6f7ff;
    color: #1890ff;
    box-shadow: none;
    border-left: 3px solid #1890ff;
    padding-left: 33px;
}

.sidebar-footer {
    padding: 12px;
    border-top: 1px solid #e8e8e8;
    flex-shrink: 0;
}

.collapse-btn {
    width: 100%;
    padding: 8px;
    background: #f5f5f5;
    border: 1px solid #d9d9d9;
    border-radius: 4px;
    color: #595959;
    cursor: pointer;
    transition: all 0.2s ease;
    display: flex;
    align-items: center;
    justify-content: center;
}

.collapse-btn:hover {
    background: #e6f7ff;
    border-color: #1890ff;
    color: #1890ff;
}

.main-content-area {
    min-height: 100vh;
    transition: margin-left 0.3s ease;
    padding-top: 56px;
    padding-bottom: 48px;
}

.top-header {
    background: #ffffff;
    border-bottom: 1px solid #e8e8e8;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 20px;
    height: 56px;
    position: fixed;
    top: 0;
    right: 0;
    left: 220px;
    z-index: 99;
    transition: left 0.3s ease;
}

.header-left {
    display: flex;
    align-items: center;
    gap: 12px;
}

.sidebar-toggle {
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #595959;
    transition: all 0.2s ease;
}

.sidebar-toggle:hover {
    background: #f5f5f5;
    color: #1890ff;
}

.breadcrumb {
    font-size: 13px;
}

.breadcrumb :deep(.el-breadcrumb__item) {
    color: #8c8c8c;
}

.breadcrumb :deep(.el-breadcrumb__item:last-child) {
    color: #262626;
    font-weight: 500;
}

.breadcrumb :deep(.el-breadcrumb__separator) {
    color: #d9d9d9;
    margin: 0 6px;
}

.header-right {
    display: flex;
    align-items: center;
    gap: 12px;
}

.search-input {
    width: 200px;
}

.search-input :deep(.el-input__wrapper) {
    border-radius: 4px;
    background: #f5f5f5;
    box-shadow: 0 0 0 1px #d9d9d9 inset;
}

.search-input :deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 1px #1890ff inset;
    background: #ffffff;
}

.header-actions {
    display: flex;
    align-items: center;
    gap: 4px;
}

.action-item {
    cursor: pointer;
}

.action-btn {
    width: 32px;
    height: 32px;
    border: none;
    background: transparent;
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #8c8c8c;
    position: relative;
    transition: all 0.2s ease;
}

.action-btn:hover {
    background: #f5f5f5;
    color: #1890ff;
}

.badge {
    position: absolute;
    top: 2px;
    right: 2px;
    min-width: 16px;
    height: 16px;
    background: #ff4d4f;
    color: #ffffff;
    font-size: 10px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 4px;
}

.user-menu {
    cursor: pointer;
}

.user-info {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 8px;
    border-radius: 4px;
    transition: all 0.2s ease;
}

.user-info:hover {
    background: #f5f5f5;
}

.user-avatar {
    border: none;
}

.user-name {
    font-size: 13px;
    color: #262626;
    font-weight: 400;
}

.content-wrapper {
    padding: 16px;
}

.content-wrapper :deep(.page-container) {
    background: #ffffff;
    border-radius: 8px;
    padding: 20px;
    box-shadow: none;
    border: 1px solid #e8e8e8;
}

.main-footer {
    background: #ffffff;
    border-top: 1px solid #e8e8e8;
    padding: 0 20px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    position: fixed;
    bottom: 0;
    right: 0;
    left: 220px;
    z-index: 99;
    transition: left 0.3s ease;
}

.footer-content {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #8c8c8c;
}

.footer-divider {
    color: #d9d9d9;
}

.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.2s ease;
}

.fade-enter-from {
    opacity: 0;
}

.fade-leave-to {
    opacity: 0;
}

.sidebar-mini .top-header {
    left: 64px;
}

.sidebar-mini .main-footer {
    left: 64px;
}

/* 移动端适配 */
@media (max-width: 768px) {
    .main-sidebar {
        display: none;
    }

    .main-content-area {
        margin-left: 0 !important;
        padding-top: 56px;
        padding-bottom: 48px;
    }

    .top-header {
        display: none;
    }

    .main-footer {
        left: 0 !important;
    }

    .content-wrapper {
        padding: 12px;
    }

    .content-wrapper :deep(.page-container) {
        padding: 12px;
        border-radius: 4px;
    }
}
</style>
