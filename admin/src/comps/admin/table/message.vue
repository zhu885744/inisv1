<template>
  <div>
    <i-table ref="i-table" :opts="state.opts">
      <template #start>
        <el-table-column type="selection" width="40" />
      </template>

      <template #end>
        <template v-if="props.type === 'all'">
          <el-button size="small" type="primary" link @click="method.read(scope.row.id)" v-if="scope.row.is_read === 0">标记已读</el-button>
          <el-button size="small" type="danger" link @click="method.delete(scope.row.id)">删 除</el-button>
        </template>
        <template v-if="props.type === 'remove'">
          <el-button size="small" type="primary" link @click="method.restore(scope.row.id)">恢 复</el-button>
          <el-button size="small" link @click="method.delete(scope.row.id, 'force')">彻底删除</el-button>
        </template>
      </template>

      <template #i-type="{ row }">
        <el-tag v-if="row.type === 'comment'" type="info" size="small">回复</el-tag>
        <el-tag v-else-if="row.type === 'like'" type="danger" size="small">点赞</el-tag>
        <el-tag v-else-if="row.type === 'follow'" type="success" size="small">关注</el-tag>
        <el-tag v-else-if="row.type === 'system'" type="warning" size="small">系统</el-tag>
        <el-tag v-else size="small">{{ row.type }}</el-tag>
      </template>

      <template #i-title="{ row }">
        <el-tooltip :content="row.title" placement="top" :show-after="200">
          <span class="text-truncate d-inline-block" style="max-width: 260px;">{{ row.title }}</span>
        </el-tooltip>
      </template>

      <template #i-content="{ row }">
        <el-tooltip :content="row.content" placement="top" :show-after="200">
          <span class="text-truncate d-inline-block" style="max-width: 320px;">{{ row.content }}</span>
        </el-tooltip>
      </template>

      <template #i-is_read="{ row }">
        <el-tag :type="row.is_read === 1 ? 'success' : 'warning'" size="small">
          {{ row.is_read === 1 ? '已读' : '未读' }}
        </el-tag>
      </template>

      <template #i-create_time="{ row }">
        <el-tooltip :content="method.formatTime(row.create_time)" placement="top">
          <span>{{ method.formatTime(row.create_time) }}</span>
        </el-tooltip>
      </template>
    </i-table>

    <!-- 详情弹窗 -->
    <el-dialog v-model="state.dialog.show" title="通知详情" width="560px">
      <table class="table table-bordered mb-0" v-if="state.dialog.data">
        <tr><td class="table-active" width="100">通知ID</td><td>{{ state.dialog.data.id }}</td></tr>
        <tr><td class="table-active">类型</td><td>{{ method.typeLabel(state.dialog.data.type) }}</td></tr>
        <tr><td class="table-active">标题</td><td>{{ state.dialog.data.title }}</td></tr>
        <tr><td class="table-active">内容</td><td>{{ state.dialog.data.content }}</td></tr>
        <tr><td class="table-active">接收用户ID</td><td>{{ state.dialog.data.uid }}</td></tr>
        <tr><td class="table-active">触发用户ID</td><td>{{ state.dialog.data.from_uid }}</td></tr>
        <tr><td class="table-active">已读状态</td><td>{{ state.dialog.data.is_read === 1 ? '已读' : '未读' }}</td></tr>
        <tr><td class="table-active">创建时间</td><td>{{ method.formatTime(state.dialog.data.create_time) }}</td></tr>
      </table>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from '{src}/utils/request.js'

const props = defineProps({
  type: { type: String, default: 'all' },
  params: { type: Object, default: () => ({}) },
  init: { type: Boolean, default: false },
})

const emits = defineEmits(['update:init', 'refresh'])

const state = reactive({
  opts: {
    url: '',
    params: {},
    columns: [
      { label: '#', prop: 'id', width: '70' },
      { label: '标题', prop: 'title', slot: 'i-title' },
      { label: '内容', prop: 'content', slot: 'i-content' },
      { label: '类型', prop: 'type', width: '80', slot: 'i-type' },
      { label: '接收用户ID', prop: 'uid', width: '110' },
      { label: '触发用户ID', prop: 'from_uid', width: '110' },
      { label: '状态', prop: 'is_read', width: '80', slot: 'i-is_read' },
      { label: '时间', prop: 'create_time', width: '160', slot: 'i-create_time', sortable: true },
    ],
  },
  dialog: {
    show: false,
    data: null,
  },
})

onMounted(() => {
  state.opts.url = '/api/notification/all'
  state.opts.params = props.params
})

const method = {
  init() {
    if (state.opts.url) {
      emits('update:init', false)
      this.$nextTick(() => {
        this.$refs['i-table'] && this.$refs['i-table'].init()
      })
    }
  },

  read(id) {
    axios.put('/api/notification/read', { id }).then(() => {
      ElMessage.success('标记已读成功')
      method.init()
    })
  },

  delete(id, type = '') {
    const apiPath = type === 'force' ? '/api/notification/delete' : '/api/notification/remove'
    const msg = type === 'force' ? '确定要彻底删除该通知吗？此操作不可恢复！' : '确定要删除该通知吗？'

    ElMessageBox.confirm(msg, '提示', { type: 'warning' }).then(() => {
      axios.del(apiPath, { ids: [id] }).then(() => {
        ElMessage.success('删除成功')
        emits('refresh', type === 'force' ? 'remove' : 'all')
        method.init()
      })
    }).catch(() => {})
  },

  restore(id) {
    axios.put('/api/notification/restore', { ids: [id] }).then(() => {
      ElMessage.success('恢复成功')
      emits('refresh', 'all')
      method.init()
    })
  },

  typeLabel(type) {
    const map = { comment: '回复', like: '点赞', follow: '关注', system: '系统消息' }
    return map[type] || type
  },

  formatTime(ts) {
    if (!ts) return '-'
    const d = new Date(ts * 1000)
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    const h = String(d.getHours()).padStart(2, '0')
    const min = String(d.getMinutes()).padStart(2, '0')
    const s = String(d.getSeconds()).padStart(2, '0')
    return `${y}-${m}-${day} ${h}:${min}:${s}`
  },

  show(row) {
    state.dialog.data = row
    state.dialog.show = true
  },
}

defineExpose({ init: method.init, show: method.show })
</script>
