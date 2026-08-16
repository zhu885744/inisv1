<template>
    <div class="mobile-nav">
        <el-drawer
            v-model="state.drawer.show"
            direction="ltr"
            size="80%"
            :show-close="false"
            class="mobile-drawer"
        >
            <template #header>
                <div class="drawer-header">
                    <div class="logo-section">
                        <div class="logo">
                            <div class="logo-mark">
                                <el-icon :size="18">
                                    <component :is="componentMap['Menu']" />
                                </el-icon>
                            </div>
                            <span class="logo-title">管理后台</span>
                        </div>
                        <button @click="state.drawer.show = false" class="close-btn">
                            <el-icon :size="18">
                                <component :is="componentMap['Close']" />
                            </el-icon>
                        </button>
                    </div>
                    <div v-if="store.comm.getLogin.finish" class="user-section">
                        <el-avatar :src="store.comm.getLogin.user?.avatar" :size="44" class="user-avatar">
                            {{ store.comm.getLogin.user?.nickname?.[0] }}
                        </el-avatar>
                        <div class="user-info">
                            <span class="user-nickname">{{ store.comm.getLogin.user?.nickname }}</span>
                            <span class="user-title">{{ store.comm.getLogin.user?.title || '管理员' }}</span>
                        </div>
                    </div>
                </div>
            </template>
            <template #default>
                <el-menu
                    class="drawer-menu"
                    :default-active="activeMenu"
                    unique-opened
                >
                    <el-menu-item index="/admin" @click="go('/admin')">
                        <template #title>
                            <span>首页</span>
                        </template>
                    </el-menu-item>
                    <el-menu-item index="/admin/profile" @click="go('/admin/profile')">
                        <template #title>
                            <span>个人中心</span>
                        </template>
                    </el-menu-item>
                    <template v-for="(item, index) in state.menu" :key="index">
                        <el-sub-menu v-if="item.children?.length" :index="item.name">
                            <template #title>
                                <span>{{ item.label }}</span>
                            </template>
                            <el-menu-item
                                v-for="(child, key) in item.children"
                                :key="key"
                                :index="child.path"
                                @click="go(child.path)"
                            >
                                <template #title>
                                    <span>{{ child.label }}</span>
                                </template>
                            </el-menu-item>
                        </el-sub-menu>
                        <el-menu-item v-else :index="item.path" @click="go(item.path)">
                            <template #title>
                                <span>{{ item.label }}</span>
                            </template>
                        </el-menu-item>
                    </template>
                </el-menu>
            </template>
            <template #footer>
                <div class="drawer-footer">
                    <el-button text class="logout-btn" @click="store.comm.logout('/')">
                        <el-icon :size="16" class="mr-1">
                            <component :is="componentMap['SwitchButton']" />
                        </el-icon>
                        <span>退出登录</span>
                    </el-button>
                </div>
            </template>
        </el-drawer>

        <el-header class="mobile-header">
            <button @click="state.drawer.show = true" class="menu-btn">
                <el-icon :size="22">
                    <component :is="componentMap['Menu']" />
                </el-icon>
            </button>
            <span class="header-title">{{ store.comm.nav.title }}</span>
            <div class="header-right">
                <el-avatar
                    v-if="store.comm.getLogin.finish"
                    :src="store.comm.getLogin.user?.avatar"
                    :size="28"
                    class="user-avatar-sm"
                >
                    {{ store.comm.getLogin.user?.nickname?.[0] }}
                </el-avatar>
            </div>
        </el-header>
    </div>
</template>

<script setup>
import { reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { list as MenuList } from '{src}/utils/menu'
import { push } from '{src}/utils/route'
import { useCommStore } from '{src}/store/comm'
const route = useRoute()
const store = {
    comm: useCommStore(),
}

const state = reactive({
    drawer: {
        show: false,
    },
    menu: [],
})

const componentMap = new Proxy({}, { get: (_, name) => name })

const activeMenu = computed(() => route.path)

// 导航跳转：跳转并关闭抽屉
const go = (path) => {
    state.drawer.show = false
    push(path)
}

onMounted(async () => {
    state.menu = await MenuList()
})
</script>

<style lang="scss" scoped>
.mobile-nav {
    display: none;
}

@media (max-width: 768px) {
    .mobile-nav {
        display: block;
    }
}

.mobile-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: 56px;
    background: #ffffff;
    border-bottom: 1px solid #f0f0f0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 12px;
    z-index: 100;
    box-shadow: 0 1px 4px rgba(0, 21, 41, 0.04);
}

.menu-btn {
    background: transparent;
    border: none;
    padding: 8px;
    color: #262626;
    cursor: pointer;
    border-radius: 4px;
}

.menu-btn:hover {
    background: #f5f5f5;
}

.header-title {
    font-size: 16px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.85);
}

.mobile-header .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
}

.user-avatar-sm {
    cursor: pointer;
    border: none;
    background: #1890ff;
    color: #fff;
    font-size: 13px;
}

.mobile-drawer :deep(.el-drawer__header) {
    padding: 0;
    margin-bottom: 0;
}

.mobile-drawer :deep(.el-drawer__body) {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #ffffff;
    padding: 0;
}

.drawer-header {
    padding: 16px 20px;
    border-bottom: 1px solid #f0f0f0;
}

.logo-section {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
}

.logo {
    display: flex;
    align-items: center;
    gap: 10px;
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
}

.logo-title {
    font-size: 18px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.85);
}

.close-btn {
    background: transparent;
    border: none;
    padding: 8px;
    color: #595959;
    cursor: pointer;
    border-radius: 4px;
}

.close-btn:hover {
    background: #f5f5f5;
    color: #262626;
}

.user-section {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f5f5f5;
    border-radius: 6px;
}

.user-avatar {
    border: none;
    background: #1890ff;
    color: #fff;
    font-size: 16px;
}

.user-section .user-info {
    display: flex;
    flex-direction: column;
}

.user-nickname {
    font-size: 15px;
    font-weight: 600;
    color: rgba(0, 0, 0, 0.85);
}

.user-title {
    font-size: 12px;
    color: rgba(0, 0, 0, 0.45);
}

.drawer-menu {
    flex: 1;
    border-right: none;
    padding: 8px 0;
    background: transparent;
    --el-menu-bg-color: transparent;
    --el-menu-text-color: #595959;
    --el-menu-hover-bg-color: #f5f5f5;
    --el-menu-active-color: #1890ff;
}

.drawer-menu :deep(.el-menu-item),
.drawer-menu :deep(.el-sub-menu__title) {
    color: #595959;
    height: 44px;
    line-height: 44px;
    padding: 0 20px;
    font-size: 14px;
}

.drawer-menu :deep(.el-menu-item:hover),
.drawer-menu :deep(.el-sub-menu__title:hover) {
    background: #f5f5f5;
    color: #262626;
}

.drawer-menu :deep(.el-menu-item.is-active) {
    background: #e6f7ff;
    color: #1890ff;
}

.drawer-menu :deep(.el-sub-menu .el-menu-item) {
    padding-left: 30px;
}

.menu-icon {
    margin-right: 12px;
}

.nav-svg-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    flex-shrink: 0;
    margin-right: 12px;
}

.nav-svg-icon :deep(svg) {
    width: 18px;
    height: 18px;
}

/* 覆盖 SVG 内联的 fill 颜色，使图标跟随当前文字颜色（hover/active 变色） */
.nav-svg-icon :deep(svg path),
.nav-svg-icon :deep(svg rect),
.nav-svg-icon :deep(svg circle) {
    fill: currentColor;
}

.submenu-icon {
    margin-right: 10px;
    color: rgba(0, 0, 0, 0.35);
}

.drawer-footer {
    padding: 12px 20px;
    border-top: 1px solid #f0f0f0;
}

.logout-btn {
    width: 100%;
    justify-content: center;
    color: #595959;
}

.logout-btn:hover {
    color: #1890ff;
}
</style>
