<template>
    <el-card style="margin-bottom: 12px" v-loading="state.status.loading">
        <template #header>
            <div class="card-header-content" style="display: flex; align-items: center; gap: 8px">
                <el-tooltip placement="top">
                    <template #content>
                        ● 是否允许用户自行注册账号（开放注册功能？）<br>
                        ● 如果不允许，那么只能通过管理员手动创建账号
                    </template>
                    <span style="font-weight: 600">注 册</span>
                </el-tooltip>
                <el-tag size="small" type="info">+0.1%</el-tag>
            </div>
        </template>
        <template #default>
            <div style="display: flex; align-items: center; justify-content: space-between">
                <div style="display: flex; align-items: center; gap: 12px">
                    <div style="display: flex; align-items: center; justify-content: center; width: 40px; height: 40px; border-radius: 8px; background: var(--el-color-primary-light-9); color: var(--el-color-primary)">
                        <i-svg name="register" size="20px"></i-svg>
                    </div>
                    <div>
                        <div style="font-weight: 600; font-size: 14px; line-height: 1.4">开放注册</div>
                        <div style="font-size: 12px; color: var(--el-text-color-secondary); margin-top: 2px; line-height: 1.4">允许/禁止用户自行注册账号</div>
                    </div>
                </div>
                <div style="display: flex; align-items: center; gap: 8px">
                    <el-switch v-model="state.status.active" v-on:change="method.change" :disabled="!state.status.finish"
                        active-text="允许" inactive-text="不允许">
                    </el-switch>
                    <el-button text type="primary" v-on:click="method.show()">
                        配置
                        <el-icon style="margin-left: 2px"><ArrowRight /></el-icon>
                    </el-button>
                </div>
            </div>
        </template>
    </el-card>

    <el-dialog v-model="state.status.dialog" class="custom" draggable :close-on-click-modal="false">
        <template #header>
            <strong>配置</strong>
        </template>
        <template #default>
            <el-form label-width="120px" label-position="left">
                <el-form-item label="注册">
                    <el-select v-model="state.struct.value" placeholder="请选择权限">
                        <el-option v-for="item in state.select.value" :key="item.value" :label="item.label" :value="item.value">
                            <span>{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="分配权限">
                    <el-select v-model="state.item.auth" multiple filterable collapse-tags placeholder="选择注册的默认权限">
                        <el-option v-for="item in state.select.auth" :key="item.id" :label="item.name" :value="item.id">
                            <span>{{ item.name }}</span>
                            <small style="float: right; color: var(--el-text-color-secondary)">{{ item.key }}</small>
                        </el-option>
                    </el-select>
                </el-form-item>
            </el-form>
        </template>
        <template #footer>
            <el-button v-on:click="state.status.dialog = false">取 消</el-button>
            <el-button v-on:click="method.save()" :loading="state.status.wait">保 存</el-button>
        </template>
    </el-dialog>
</template>

<script setup>
import axios from '{src}/utils/request.js'
import utils from "{src}/utils/utils.js";

const { ctx, proxy } = getCurrentInstance()
const state = reactive({
    item: {
        auth: [],
    },
    struct: {
        key: 'ALLOW_REGISTER',
    },
    status: {
        finish: false,
        active: false,
        dialog: false,
        loading: true,
        wait: false,
    },
    select: {
        auth: [],
        value: [
            { value: '0', label: '禁止注册', color: 'var(--bs-danger)' },
            { value: '1', label: '允许注册', color: 'var(--bs-success)' }
        ]
    }
})

const method = {
    init: async () => {

        state.status.finish  = false
        state.status.loading = true

        const { code, data } = await axios.get('/api/config/one', {
            key: 'ALLOW_REGISTER'
        })

        state.status.loading = false

        if (code !== 200) return

        state.struct = data

        state.status.finish  = true
    },
    change: async value => {

        const { code, msg } = await axios.post('/api/config/save', {
            key: 'ALLOW_REGISTER',
            value: value ? 1 : 0
        })

        if (code === 200) return

        state.status.active = !value
        ElMessage.error(msg)
    },
    show() {
        if (!state.status.finish) return ElMessage.warning('配置获取失败，无法进行配置！')
        state.status.dialog = true
    },
    // 保存配置
    save: async () => {

        state.status.wait   = true

        const { code, msg } = await axios.post('/api/config/save', {
            ...state.struct, text: state.item.auth
        })

        state.status.wait   = false

        if (code !== 200) return ElMessage.error('保存失败：' + msg)

        state.status.dialog = false
        ElMessage.success('保存成功')
    },
    // 获取权限分组
    auth: async () => {

        const { code, data } = await axios.get('/api/auth-group/column', {
            field: 'id,key,name,root'
        })
        if (code !== 200) return

        state.select.auth = data.map(item=>{
            item.color = item.root === 1 ? 'var(--bs-success)' : 'var(--bs-secondary)'
            return item
        })
    }
}

onMounted(() => {
    method.init()
    method.auth()
})

watch(() => state.struct, (row) => {
    state.item.auth     = (row?.text?.split(',') || []).filter(item => !utils.is.empty(item)).map(item=>parseInt(item))
    state.status.active = parseInt(state.struct?.value) === 1
}, { deep: true })

watch(() => state.status.active, (value) => {
    state.struct.value = value ? '1' : '0'
})

// 将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>
