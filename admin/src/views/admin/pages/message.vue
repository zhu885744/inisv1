<template>
  <div class="container-box">
    <!-- 顶部栏 -->
    <el-row justify="space-between" align="middle" class="mb-3">
      <el-col :span="16">
        <div class="d-flex align-items-center gap-2">
          <el-input
            v-model="state.item.search"
            placeholder="搜索通知标题或内容"
            clearable
            style="width: 280px"
          />
          <el-button type="primary" @click="method.refresh()" :icon="'Refresh'">刷新</el-button>
        </div>
      </el-col>
      <el-col :span="8" class="text-end">
        <el-button type="primary" :icon="'Promotion'" @click="method.gotoPush">推送消息</el-button>
      </el-col>
    </el-row>

    <!-- 选项卡 -->
    <el-tabs v-model="state.item.tabs" @tab-change="method.change">
      <el-tab-pane label="全部通知" name="all">
        <table-message ref="allTable" v-model:init="state.tabs.all" :params="state.params.all" type="all" @refresh="method.refresh" />
      </el-tab-pane>

      <el-tab-pane label="回收站" name="remove">
        <table-message ref="removeTable" v-model:init="state.tabs.remove" :params="state.params.remove" type="remove" @refresh="method.refresh" />
      </el-tab-pane>

      <el-tab-pane label="系统公告" name="broadcast">
        <table-message ref="broadcastTable" v-model:init="state.tabs.broadcast" :params="state.params.broadcast" type="broadcast" @refresh="method.refresh" />
      </el-tab-pane>

      <el-tab-pane label="推送系统消息" name="push">
        <el-card>
          <template #header>
            <span class="fw-bold">发送系统通知</span>
          </template>

          <el-form :model="state.pushForm" label-width="110px" style="max-width: 700px;">
            <!-- 目标用户 -->
            <el-form-item label="目标用户" required>
              <el-radio-group v-model="state.pushForm.target_type" @change="method.onTargetChange">
                <el-radio value="all">全部用户</el-radio>
                <el-radio value="partial">部分用户</el-radio>
                <el-radio value="single">单个用户</el-radio>
              </el-radio-group>
            </el-form-item>

            <!-- 用户选择 -->
            <el-form-item v-if="state.pushForm.target_type !== 'all'" label="选择用户" required>
              <el-select
                v-model="state.pushForm.user_ids"
                :multiple="state.pushForm.target_type === 'partial'"
                filterable
                remote
                reserve-keyword
                placeholder="输入昵称或账号搜索用户"
                :remote-method="method.searchUsers"
                :loading="state.pushForm.userLoading"
                style="width: 100%"
              >
                <el-option
                  v-for="user in state.pushForm.userOptions"
                  :key="user.id"
                  :label="`${user.nickname} (ID:${user.id})`"
                  :value="user.id"
                />
              </el-select>
            </el-form-item>

            <!-- 通知标题 -->
            <el-form-item label="通知标题" required>
              <el-input v-model="state.pushForm.title" placeholder="请输入通知标题" maxlength="100" show-word-limit />
            </el-form-item>

            <!-- 通知内容 -->
            <el-form-item label="通知内容" required>
              <el-input
                v-model="state.pushForm.content"
                type="textarea"
                :rows="4"
                placeholder="请输入通知内容"
                maxlength="500"
                show-word-limit
              />
            </el-form-item>

            <!-- 发送方式 -->
            <el-form-item label="发送方式">
              <el-checkbox-group v-model="state.pushForm.send_methods">
                <el-checkbox value="system">系统消息（应用内通知）</el-checkbox>
                <el-checkbox value="email" :disabled="state.pushForm.target_type === 'all'">邮件通知</el-checkbox>
              </el-checkbox-group>
              <div class="text-muted small mt-1">
                系统消息将出现在用户的消息中心；邮件通知将发送至用户绑定的邮箱。
                <template v-if="state.pushForm.target_type === 'all'">
                  <el-tag type="warning" size="small" effect="plain" class="ms-1">广播模式</el-tag>
                  <span class="text-warning">仅创建 1 条公告记录（uid=0），全体用户可见；已读/删除按用户独立记录，不产生百万条数据。</span>
                </template>
                <template v-else>
                  <el-tag type="info" size="small" effect="plain" class="ms-1">定向模式</el-tag>
                  <span>为每个目标用户创建独立通知记录。</span>
                </template>
              </div>
            </el-form-item>

            <!-- 系统身份 -->
            <el-form-item label="发送身份">
              <el-radio-group v-model="state.pushForm.as_system">
                <el-radio :value="true">
                  <el-tag type="warning" size="small">系统身份</el-tag>
                  <span class="ms-1 text-muted small">标题前显示【系统消息】</span>
                </el-radio>
                <el-radio :value="false">
                  <el-tag type="primary" size="small">个人身份</el-tag>
                  <span class="ms-1 text-muted small">显示"管理员xxx发送了一条系统通知"</span>
                </el-radio>
              </el-radio-group>
            </el-form-item>

            <!-- 预览 -->
            <el-form-item label="预览效果" v-if="state.pushForm.title">
              <el-alert type="info" :closable="false" show-icon>
                <template #title>
                  <strong>{{ state.pushForm.as_system ? '【系统消息】' + state.pushForm.title : state.pushForm.title }}</strong>
                </template>
                <template #default>
                  <p class="mb-0 mt-1">{{ method.previewContent() }}</p>
                </template>
              </el-alert>
            </el-form-item>

            <!-- 发送按钮 -->
            <el-form-item>
              <el-button type="success" size="large" @click="method.sendPush" :loading="state.pushForm.sending">
                {{ state.pushForm.target_type === 'all' ? '广播给全体用户' : '发送通知' }}
              </el-button>
              <el-button @click="method.resetPushForm">重置表单</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { reactive, watch, getCurrentInstance, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import axios from '{src}/utils/request.js'
import TableMessage from '@/comps/admin/table/message.vue'

const { proxy } = getCurrentInstance()

const state = reactive({
  item: {
    title: '消息通知',
    search: '',
    tabs: 'all',
  },
  tabs: {
    all: false,
    remove: false,
    broadcast: false,
  },
  params: {
    all: { order: 'create_time desc' },
    remove: { order: 'create_time desc', onlyTrashed: true },
    // 系统公告：仅展示广播通知（uid=0，推送给全体用户的单条记录）
    broadcast: { order: 'create_time desc', uid: 0 },
  },
  pushForm: {
    target_type: 'all',
    user_ids: [],
    title: '',
    content: '',
    send_methods: ['system'],
    as_system: true,
    sending: false,
    userOptions: [],
    userLoading: false,
  },
})

const method = {
  refresh(tab = '') {
    // tab 名 -> 组件 ref 名
    const tables = { all: 'allTable', remove: 'removeTable', broadcast: 'broadcastTable' }
    if (tab && tables[tab]) {
      proxy.$refs[tables[tab]]?.init?.()
      return
    }
    // 无参：刷新所有已挂载的表格
    for (const key of Object.keys(tables)) {
      proxy.$refs[tables[key]]?.init?.()
    }
  },

  change(name) {
    state.tabs[name] = true
  },

  gotoPush() {
    state.item.tabs = 'push'
    state.tabs.push = true
  },

  order(key, val) {
    state.params.all.order = `${key} ${val}`
    state.params.remove.order = `${key} ${val}`
    state.params.broadcast.order = `${key} ${val}`
    method.refresh()
  },

  searchUsers(query) {
    if (!query) {
      state.pushForm.userOptions = []
      return
    }
    state.pushForm.userLoading = true
    axios.get('/api/search/users', { keyword: query, limit: 20 }).then(res => {
      // 后端返回 {code,msg,data:{data:[...],count,...}}，用户列表嵌套在 data.data（兼容 data.list）
      const data = res?.data?.data || res?.data?.list || []

      state.pushForm.userOptions = Array.isArray(data) ? data.map(u => ({
        id: u.id,
        nickname: u.nickname || u.account || '未知',
      })) : []
    }).finally(() => {
      state.pushForm.userLoading = false
    })
  },

  onTargetChange() {
    state.pushForm.user_ids = []
    state.pushForm.userOptions = []
    // 切到"全部用户"时移除邮件方式（全量推送为广播模式，不支持逐用户邮件）
    if (state.pushForm.target_type === 'all') {
      state.pushForm.send_methods = state.pushForm.send_methods.filter(m => m !== 'email')
    }
  },

  previewContent() {
    const adminName = localStorage.getItem('admin_nickname') || '管理员'
    if (state.pushForm.as_system) {
      return state.pushForm.content
    }
    return `${adminName} 发送了一条系统通知：${state.pushForm.content}`
  },

  async sendPush() {
    if (!state.pushForm.title) {
      ElMessage.warning('请输入通知标题')
      return
    }
    if (!state.pushForm.content) {
      ElMessage.warning('请输入通知内容')
      return
    }
    if (state.pushForm.send_methods.length === 0) {
      ElMessage.warning('请选择至少一种发送方式')
      return
    }
    if (state.pushForm.target_type !== 'all' && !state.pushForm.user_ids.length) {
      ElMessage.warning('请选择目标用户')
      return
    }

    state.pushForm.sending = true
    try {
      const payload = {
        target_type: state.pushForm.target_type,
        user_ids: state.pushForm.target_type !== 'all'
          ? (Array.isArray(state.pushForm.user_ids) ? state.pushForm.user_ids : [state.pushForm.user_ids])
          : [],
        title: state.pushForm.title,
        content: state.pushForm.content,
        send_email: state.pushForm.send_methods.includes('email'),
        as_system: state.pushForm.as_system,
      }

      const res = await axios.post('/api/notification/send-system', payload)
      const data = res?.data || res

      // 全量推送：后端返回 broadcast:true，只写 1 条广播记录，全体用户可见
      if (data?.broadcast) {
        ElMessage.success('已广播给全体用户！仅创建 1 条公告记录，用户进入消息中心即可看到')
        method.resetPushForm()
        method.refresh('all')
        method.refresh('broadcast')
      } else {
        ElMessage.success(`推送完成！共 ${data?.total || 0} 个目标，成功 ${data?.success || 0} 条`)
        method.resetPushForm()
        method.refresh('all')
      }
    } catch (err) {
      ElMessage.error('推送失败：' + (err?.msg || '未知错误'))
    } finally {
      state.pushForm.sending = false
    }
  },

  resetPushForm() {
    state.pushForm = {
      target_type: 'all',
      user_ids: [],
      title: '',
      content: '',
      send_methods: ['system'],
      as_system: true,
      sending: false,
      userOptions: [],
      userLoading: false,
    }
  },
}

// 搜索防抖
watch(() => state.item.search, val => {
  clearTimeout(state.item.timer)
  state.item.timer = setTimeout(() => {
    const like = val
      ? { title: `%${val}%`, content: `%${val}%` }
      : undefined

    state.params.all.like = like
    state.params.remove.like = like
    state.params.broadcast.like = like
    method.refresh()
  }, 500)
})

// 初始化加载
onMounted(() => {
  state.tabs.all = true
})
</script>
