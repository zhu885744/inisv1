<template>
    <el-container class="app-wrapper" :class="{ 'sidebar-mini': state.sidebarCollapsed }">
        <el-aside
            :width="state.sidebarCollapsed ? '64px' : '220px'"
            class="main-sidebar"
        >
            <div class="sidebar-header">
                <div class="logo" @click="method.push('/admin')">
                    <div class="logo-mark">
                        <el-icon :size="18">
                            <component :is="componentMap['Menu']" />
                        </el-icon>
                    </div>
                    <span v-show="!state.sidebarCollapsed" class="logo-title">管理后台</span>
                </div>
            </div>

            <nav class="sidebar-menu">
                <el-menu
                    :default-active="activeMenu"
                    mode="vertical"
                    :collapse="state.sidebarCollapsed"
                    :collapse-transition="false"
                    unique-opened
                    @select="handleMenuSelect"
                    class="el-menu-vertical"
                >
                    <el-menu-item index="/admin">
                        <span class="menu-text-short">{{ '首页'.slice(0, 2) }}</span>
                        <template #title>
                            <span>{{ '首页' }}</span>
                        </template>
                    </el-menu-item>

                    <template v-for="(item, index) in state.menu" :key="index">
                        <el-sub-menu v-if="item.children?.length" :index="item.name">
                            <template #title>
                                <span>{{ item.label }}</span>
                                <span class="menu-text-short">{{ item.label.slice(0, 2) }}</span>
                            </template>
                            <el-menu-item
                                v-for="(child, key) in item.children"
                                :key="key"
                                :index="child.path"
                            >
                                <template #title>
                                    <span>{{ child.label }}</span>
                                    <span class="menu-text-short">{{ child.label.slice(0, 2) }}</span>
                                </template>
                            </el-menu-item>
                        </el-sub-menu>

                        <el-menu-item v-else :index="item.path">
                            <template #title>
                                <span>{{ item.label }}</span>
                                <span class="menu-text-short">{{ item.label.slice(0, 2) }}</span>
                            </template>
                        </el-menu-item>
                    </template>
                </el-menu>
            </nav>

            <div class="sidebar-footer">
                <span class="version-text">v{{ state.version }}</span>
            </div>
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
                            <component :is="componentMap[state.sidebarCollapsed ? 'Expand' : 'Fold']" />
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
                    <el-dropdown trigger="click" class="user-menu">
                        <div class="user-info">
                            <el-avatar :src="store.comm.getLogin.user?.avatar" :size="28" class="user-avatar">
                                {{ store.comm.getLogin.user?.nickname?.[0] }}
                            </el-avatar>
                            <span class="user-name">{{ store.comm.getLogin.user?.nickname }}</span>
                            <el-icon :size="12" class="user-arrow">
                                <component :is="componentMap['ArrowDown']" />
                            </el-icon>
                        </div>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item @click="method.push('/admin/profile')">
                                    <el-icon :size="14">
                                        <component :is="componentMap['User']" />
                                    </el-icon>
                                    <span>个人中心</span>
                                </el-dropdown-item>
                                <el-dropdown-item divided @click="store.comm.logout('/')">
                                    <el-icon :size="14">
                                        <component :is="componentMap['SwitchButton']" />
                                    </el-icon>
                                    <span>退出登录</span>
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
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
    /* 覆盖全局的浅灰菜单图标色，白色侧边栏需使用深色图标 */
    --menu-icon-color: 89, 89, 89;
    display: flex;
    flex-direction: column;
    border-right: 1px solid #f0f0f0;
    transition: width 0.28s cubic-bezier(0.4, 0, 0.2, 1);
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
    background: #ffffff;
    border-bottom: 1px solid #f0f0f0;
    flex-shrink: 0;
    overflow: hidden;
}

.logo {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    white-space: nowrap;
}

.logo:hover {
    opacity: 0.85;
}

.logo-mark {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: #1890ff;
    color: #fff;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}

.logo-title {
    font-size: 16px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.85);
    white-space: nowrap;
    letter-spacing: 0.5px;
}

.sidebar-menu {
    flex: 1;
    padding: 8px 0;
    overflow-y: auto;
    overflow-x: hidden;
}

.sidebar-menu::-webkit-scrollbar {
    width: 4px;
}

.sidebar-menu::-webkit-scrollbar-thumb {
    background: rgba(0, 0, 0, 0.1);
    border-radius: 2px;
}

.el-menu-vertical {
    border: none;
    background: transparent;
    --el-menu-bg-color: transparent;
    --el-menu-text-color: #595959;
    --el-menu-hover-bg-color: #f5f5f5;
    --el-menu-active-color: #1890ff;
}

.el-menu-vertical :deep(.el-menu-item),
.el-menu-vertical :deep(.el-sub-menu__title) {
    color: #595959;
    height: 44px;
    line-height: 44px;
    margin: 4px 8px;
    border-radius: 4px;
    padding: 0 12px;
    transition: all 0.2s ease;
    font-size: 14px;
}

.el-menu-vertical :deep(.el-menu-item:hover),
.el-menu-vertical :deep(.el-sub-menu__title:hover) {
    background: #f5f5f5;
    color: #262626;
}

.el-menu-vertical :deep(.el-menu-item.is-active) {
    background: #e6f7ff;
    color: #1890ff;
}

.el-menu-vertical :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
    color: #1890ff;
}

.el-menu-vertical :deep(.el-sub-menu .el-menu-item) {
    padding-left: 30px;
    margin: 2px 8px;
    border-radius: 4px;
    font-size: 13px;
    height: 40px;
    line-height: 40px;
}

.el-menu-vertical :deep(.el-sub-menu .el-menu-item.is-active) {
    background: #e6f7ff;
    color: #1890ff;
}

.menu-svg-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    margin-right: 8px;
}

.menu-svg-icon :deep(svg) {
    width: 16px;
    height: 16px;
}

/* 覆盖 SVG 内联的 fill 颜色，使图标跟随当前文字颜色（hover/active 变色） */
.menu-svg-icon :deep(svg path),
.menu-svg-icon :deep(svg rect),
.menu-svg-icon :deep(svg circle) {
    fill: currentColor;
}

/* 默认展开状态：隐藏短文字，显示完整文字 */
/* 默认展开状态：隐藏短文字，显示完整文字 */
.menu-text-short {
    display: none;
}

/* 收缩（collapse）状态：菜单项居中 */
.el-menu-vertical.el-menu--collapse :deep(.el-menu-item),
.el-menu-vertical.el-menu--collapse :deep(.el-sub-menu__title) {
    margin: 4px 8px;
    padding: 0;
    justify-content: center;
}

/* 顶层菜单项（父级为 ElMenu，collapse 时标题移入 tooltip）的短文字 */
.el-menu-vertical.el-menu--collapse :deep(.el-menu-item > .el-menu-tooltip__trigger) {
    justify-content: center;
    padding: 0;
}

.el-menu-vertical.el-menu--collapse :deep(.el-menu-item > .el-menu-tooltip__trigger .menu-text-short) {
    display: inline-block;
    font-size: 14px;
    font-weight: 500;
    letter-spacing: 1px;
}

/* 子菜单（el-sub-menu）标题的短文字：覆盖 element-plus 对直接子 span 的隐藏规则 */
.el-menu-vertical.el-menu--collapse :deep(.el-sub-menu__title > span.menu-text-short) {
    visibility: visible !important;
    width: auto !important;
    height: auto !important;
    overflow: visible !important;
    display: inline-block !important;
    font-size: 14px;
    font-weight: 500;
    letter-spacing: 1px;
}

/* 收缩状态：取消选中效果 */
.el-menu-vertical.el-menu--collapse :deep(.el-menu-item.is-active) {
    background: transparent;
    color: #595959;
}

.el-menu-vertical.el-menu--collapse :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
    color: #595959;
}

.el-menu-vertical.el-menu--collapse :deep(.el-sub-menu .el-menu-item.is-active) {
    background: transparent;
    color: #595959;
}

.sidebar-footer {
    padding: 12px 16px;
    border-top: 1px solid #f0f0f0;
    flex-shrink: 0;
}

.version-text {
    font-size: 12px;
    color: rgba(0, 0, 0, 0.35);
}

.main-content-area {
    min-height: 100vh;
    transition: margin-left 0.28s cubic-bezier(0.4, 0, 0.2, 1);
    padding-top: 56px;
    padding-bottom: 48px;
}

.top-header {
    background: #ffffff;
    border-bottom: 1px solid #f0f0f0;
    box-shadow: 0 1px 4px rgba(0, 21, 41, 0.04);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    height: 56px;
    position: fixed;
    top: 0;
    right: 0;
    left: 220px;
    z-index: 99;
    transition: left 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.header-left {
    display: flex;
    align-items: center;
    gap: 8px;
}

.sidebar-toggle {
    width: 36px;
    height: 36px;
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
    font-size: 14px;
}

.breadcrumb :deep(.el-breadcrumb__item) {
    color: rgba(0, 0, 0, 0.45);
}

.breadcrumb :deep(.el-breadcrumb__inner a),
.breadcrumb :deep(.el-breadcrumb__inner.is-link) {
    color: rgba(0, 0, 0, 0.45);
    font-weight: 400;
}

.breadcrumb :deep(.el-breadcrumb__inner a:hover),
.breadcrumb :deep(.el-breadcrumb__inner.is-link:hover) {
    color: #1890ff;
}

.breadcrumb :deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
    color: rgba(0, 0, 0, 0.85);
    font-weight: 500;
}

.breadcrumb :deep(.el-breadcrumb__separator) {
    color: rgba(0, 0, 0, 0.25);
    margin: 0 8px;
}

.header-right {
    display: flex;
    align-items: center;
    gap: 8px;
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
    background: rgba(0, 0, 0, 0.04);
}

.user-avatar {
    background: #1890ff;
    color: #fff;
    font-size: 13px;
    border: none;
}

.user-name {
    font-size: 14px;
    color: rgba(0, 0, 0, 0.85);
}

.user-arrow {
    color: rgba(0, 0, 0, 0.45);
}

.content-wrapper {
    padding: 16px;
}

.content-wrapper :deep(.page-container) {
    background: #ffffff;
    border-radius: 4px;
    padding: 20px;
    border: 1px solid #f0f0f0;
}

.main-footer {
    background: #ffffff;
    border-top: 1px solid #f0f0f0;
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
    transition: left 0.28s cubic-bezier(0.4, 0, 0.2, 1);
}

.footer-content {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: rgba(0, 0, 0, 0.45);
}

.footer-divider {
    color: rgba(0, 0, 0, 0.25);
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
