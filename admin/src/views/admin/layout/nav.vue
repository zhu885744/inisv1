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
                            <el-icon :size="24" class="logo-icon">
                                <component :is="componentMap['LayoutDashboard']" />
                            </el-icon>
                            <span class="logo-title">管理后台</span>
                        </div>
                        <button @click="state.drawer.show = false" class="close-btn">
                            <el-icon :size="18">
                                <component :is="componentMap['X']" />
                            </el-icon>
                        </button>
                    </div>
                    <div v-if="store.comm.getLogin.finish" class="user-section">
                        <el-avatar :src="store.comm.getLogin.user?.avatar" :size="48" class="user-avatar" />
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
                    <el-menu-item index="/admin" @click="handleNavClick(push('/admin'))">
                        <el-icon :size="18" class="menu-icon">
                            <component :is="componentMap['House']" />
                        </el-icon>
                        <span>首页</span>
                    </el-menu-item>
                    <template v-for="(item, index) in state.menu" :key="index">
                        <el-sub-menu v-if="item.children?.length" :index="item.name">
                            <template #title>
                                <span v-if="item.icon" class="nav-svg-icon" v-html="item.icon" />
                                <el-icon :size="18" class="menu-icon" v-else>
                                    <component :is="getIcon(item)" />
                                </el-icon>
                                <span>{{ item.label }}</span>
                            </template>
                            <el-menu-item
                                v-for="(child, key) in item.children"
                                :key="key"
                                :index="child.path"
                                @click="handleNavClick(child.fn())"
                            >
                                <span v-if="child.icon" class="nav-svg-icon" v-html="child.icon" />
                                <el-icon :size="14" class="submenu-icon" v-else>
                                    <component :is="componentMap['Circle']" />
                                </el-icon>
                                <span>{{ child.label }}</span>
                            </el-menu-item>
                        </el-sub-menu>
                        <el-menu-item v-else :index="item.path" @click="handleNavClick(item.fn?.())">
                            <span v-if="item.icon" class="nav-svg-icon" v-html="item.icon" />
                            <el-icon :size="18" class="menu-icon" v-else>
                                <component :is="getIcon(item)" />
                            </el-icon>
                            <span>{{ item.label }}</span>
                        </el-menu-item>
                    </template>
                </el-menu>
            </template>
            <template #footer>
                <div class="drawer-footer">
                    <el-button text class="logout-btn" @click="store.comm.logout('/')">
                        <el-icon :size="16" class="mr-1">
                            <component :is="componentMap['LogOut']" />
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
                />
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

const getIcon = (item) => {
    const iconMap = {
        'create': 'Edit',
        'manage': 'Grid3x3',
        'security': 'Shield',
    }
    return componentMap[iconMap[item.name] || 'Menu']
}

const handleNavClick = (fn) => {
    state.drawer.show = false
    fn
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
    background: #fff;
    border-bottom: 1px solid #e8e8e8;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    z-index: 100;
}

.menu-btn {
    background: transparent;
    border: none;
    padding: 8px;
    color: #595959;
    cursor: pointer;
    border-radius: 4px;
}

.menu-btn:hover {
    background: #f5f5f5;
}

.header-title {
    font-size: 16px;
    font-weight: 600;
    color: #262626;
}

.mobile-header .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
}

.user-avatar-sm {
    cursor: pointer;
    border: none;
}

.mobile-drawer :deep(.el-drawer__body) {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: #fff;
    padding: 0;
}

.drawer-header {
    padding: 16px 20px;
    border-bottom: 1px solid #e8e8e8;
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

.logo-icon {
    color: #1890ff;
}

.logo-title {
    font-size: 18px;
    font-weight: 600;
    color: #262626;
}

.close-btn {
    background: transparent;
    border: none;
    padding: 8px;
    color: #8c8c8c;
    cursor: pointer;
    border-radius: 4px;
}

.close-btn:hover {
    background: #f5f5f5;
}

.user-section {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f5f5f5;
    border-radius: 4px;
}

.user-avatar {
    border: none;
}

.user-section .user-info {
    display: flex;
    flex-direction: column;
}

.user-nickname {
    font-size: 15px;
    font-weight: 600;
    color: #262626;
}

.user-title {
    font-size: 12px;
    color: #8c8c8c;
}

.drawer-menu {
    flex: 1;
    border-right: none;
    padding: 8px 0;
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
    border-left: 3px solid #1890ff;
    padding-left: 17px;
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

.submenu-icon {
    margin-right: 10px;
    color: #bfbfbf;
}

.drawer-footer {
    padding: 12px 20px;
    border-top: 1px solid #e8e8e8;
}

.logout-btn {
    width: 100%;
    justify-content: center;
    color: #ff4d4f;
}
</style>
