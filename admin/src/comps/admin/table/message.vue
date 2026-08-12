<template>
  <div>
    <!-- 批量操作栏 -->
    <div v-if="state.selection.length" class="d-flex align-items-center gap-2 mb-2">
      <template v-if="props.type !== 'remove'">
        <el-button size="small" type="primary" @click="method.batchRead">批量已读</el-button>
        <el-button size="small" type="danger" @click="method.batchDelete">批量删除</el-button>
      </template>
      <template v-else>
        <el-button size="small" type="primary" @click="method.batchRestore">批量恢复</el-button>
        <el-button size="small" type="danger" @click="method.batchDelete('force')">批量彻底删除</el-button>
      </template>
      <span class="text-muted small">已选 {{ state.selection.length }} 项</span>
    </div>

    <i-table ref="i-table" :opts="state.opts" @selection:change="method.onSelection">
      <template #start>
        <el-table-column type="selection" width="40" />
      </template>

      <!-- 标题 -->
      <template #i-title="{ scope = {} }">
        <div class="d-flex align-items-center">
          <el-tag v-if="method.isBroadcast(scope)" type="warning" size="small" effect="dark" class="me-1">公告</el-tag>
          <el-tooltip :content="scope.title" placement="top" :show-after="200">
            <span class="text-truncate d-inline-block" style="max-width: 200px;">{{ scope.title }}</span>
          </el-tooltip>
        </div>
      </template>

      <!-- 内容 -->
      <template #i-content="{ scope = {} }">
        <el-tooltip :content="scope.content" placement="top" :show-after="200">
          <span class="text-truncate d-inline-block" style="max-width: 320px;">{{ scope.content }}</span>
        </el-tooltip>
      </template>

      <!-- 类型 -->
      <template #i-type="{ scope = {} }">
        <el-tag v-if="method.isBroadcast(scope)" type="warning" size="small">系统公告</el-tag>
        <el-tag v-else-if="scope.type === 'comment'" type="info" size="small">回复</el-tag>
        <el-tag v-else-if="scope.type === 'like'" type="danger" size="small">点赞</el-tag>
        <el-tag v-else-if="scope.type === 'follow'" type="success" size="small">关注</el-tag>
        <el-tag v-else-if="scope.type === 'system'" type="primary" size="small">系统</el-tag>
        <el-tag v-else size="small">{{ scope.type }}</el-tag>
      </template>

      <!-- 接收用户 -->
      <template #i-uid="{ scope = {} }">
        <el-tag v-if="method.isBroadcast(scope)" type="info" size="small" effect="plain">全体用户</el-tag>
        <span v-else>{{ scope.uid }}</span>
      </template>

      <!-- 触发用户 -->
      <template #i-from_uid="{ scope = {} }">
        <span v-if="method.isBroadcast(scope)" class="text-muted">系统</span>
        <span v-else>{{ scope.from_uid }}</span>
      </template>

      <!-- 状态 -->
      <template #i-is_read="{ scope = {} }">
        <el-tag v-if="method.isBroadcast(scope)" type="warning" size="small" effect="plain">公告</el-tag>
        <el-tag v-else :type="scope.is_read === 1 ? 'success' : 'warning'" size="small">
          {{ scope.is_read === 1 ? '已读' : '未读' }}
        </el-tag>
      </template>

      <!-- 时间 -->
      <template #i-create_time="{ scope = {} }">
        <el-tooltip :content="method.formatTime(scope.create_time)" placement="top">
          <span>{{ method.formatTime(scope.create_time) }}</span>
        </el-tooltip>
      </template>

      <!-- 操作列 -->
      <template #end>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="scope">
            <template v-if="props.type !== 'remove'">
              <el-button
                v-if="!method.isBroadcast(scope.row) && scope.row.is_read === 0"
                size="small" type="primary" link
                @click="method.read(scope.row.id)"
              >标记已读</el-button>
              <el-button size="small" link @click="method.show(scope.row)">详情</el-button>
              <el-button size="small" type="danger" link @click="method.delete(scope.row.id, '', scope.row)">删 除</el-button>
            </template>
            <template v-else>
              <el-button size="small" type="primary" link @click="method.restore(scope.row.id)">恢 复</el-button>
              <el-button size="small" type="danger" link @click="method.delete(scope.row.id, 'force', scope.row)">彻底删除</el-button>
            </template>
          </template>
        </el-table-column>
      </template>
    </i-table>

    <!-- 详情弹窗 -->
    <el-dialog v-model="state.dialog.show" title="通知详情" width="560px">
      <table class="table table-bordered mb-0" v-if="state.dialog.data">
        <tr><td class="table-active" width="110">通知ID</td><td>{{ state.dialog.data.id }}</td></tr>
        <tr><td class="table-active">类型</td><td>{{ method.typeLabel(state.dialog.data) }}</td></tr>
        <tr><td class="table-active">标题</td><td>{{ state.dialog.data.title }}</td></tr>
        <tr><td class="table-active">内容</td><td>{{ state.dialog.data.content }}</td></tr>
        <tr>
          <td class="table-active">接收范围</td>
          <td>
            <el-tag v-if="method.isBroadcast(state.dialog.data)" type="warning" size="small">全体用户（广播公告，单条记录）</el-tag>
            <span v-else>用户ID：{{ state.dialog.data.uid }}</span>
          </td>
        </tr>
        <tr>
          <td class="table-active">触发用户</td>
          <td>{{ method.isBroadcast(state.dialog.data) ? '系统' : state.dialog.data.from_uid }}</td>
        </tr>
        <tr>
          <td class="table-active">已读状态</td>
          <td>
            <template v-if="method.isBroadcast(state.dialog.data)">
              <el-tag type="info" size="small">公告</el-tag>
              <span class="text-muted small ms-1">已读状态按用户独立记录于 notification_reads</span>
            </template>
            <template v-else>{{ state.dialog.data.is_read === 1 ? '已读' : '未读' }}</template>
          </td>
        </tr>
        <tr><td class="table-active">创建时间</td><td>{{ method.formatTime(state.dialog.data.create_time) }}</td></tr>
      </table>
    </el-dialog>
  </div>
</template>

<script setup>
import { reactive, watch, onMounted, getCurrentInstance } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'

const props = defineProps({
  type: { type: String, default: 'all' },
  params: { type: Object, default: () => ({}) },
  init: { type: Boolean, default: false },
})

const emits = defineEmits(['update:init', 'refresh'])

const { proxy } = getCurrentInstance()

const state = reactive({
  selection: [],
  opts: {
    url: '',
    params: {},
    columns: [
      { label: '#', prop: 'id', width: '70' },
      { label: '标题', prop: 'title', slot: 'i-title' },
      { label: '内容', prop: 'content', slot: 'i-content' },
      { label: '类型', prop: 'type', width: '100', slot: 'i-type' },
      { label: '接收用户', prop: 'uid', width: '100', slot: 'i-uid' },
      { label: '触发用户', prop: 'from_uid', width: '80', slot: 'i-from_uid' },
      { label: '状态', prop: 'is_read', width: '80', slot: 'i-is_read' },
      { label: '时间', prop: 'create_time', width: '160', slot: 'i-create_time', sortable: true },
    ],
  },
  dialog: {
    show: false,
    data: null,
  },
})

const method = {
  // 重新加载表格数据
  init: async () => {
    if (state.opts.url) {
      await proxy.$refs['i-table']?.init?.()
    }
  },

  // 广播通知：uid === 0（推送给全体用户，仅一条共享记录）
  isBroadcast(row) {
    return row && Number(row.uid) === 0
  },

  onSelection(selection) {
    state.selection = selection || []
  },

  // 统一转换为逗号串，后端 utils.Unity.Ids 可正确解析
  idsOf(rows = state.selection) {
    return rows.map(r => r.id).join(',')
  },

  read(id) {
    axios.put('/api/notification/read', { id }).then(() => {
      ElMessage.success('标记已读成功')
      method.init()
    })
  },

  batchRead() {
    const ids = method.idsOf()
    if (!ids) return
    axios.put('/api/notification/read-batch', { ids }).then(() => {
      ElMessage.success(`已标记 ${state.selection.length} 条为已读`)
      state.selection = []
      method.init()
    })
  },

  batchRestore() {
    const ids = method.idsOf()
    if (!ids) return
    axios.put('/api/notification/restore', { ids }).then(() => {
      ElMessage.success(`已恢复 ${state.selection.length} 条`)
      state.selection = []
      emits('refresh', 'remove')
      method.init()
    })
  },

  delete(id, type = '', row = null) {
    const isBroadcast = method.isBroadcast(row)
    const apiPath = type === 'force' ? '/api/notification/delete' : '/api/notification/remove'
    let msg
    if (type === 'force') {
      msg = isBroadcast
        ? '确定要彻底删除该公告吗？删除后全体用户将永远无法看到，此操作不可恢复！'
        : '确定要彻底删除该通知吗？此操作不可恢复！'
    } else {
      msg = isBroadcast
        ? '确定要删除该公告吗？删除后全体用户将看不到此公告（可到回收站恢复）。'
        : '确定要删除该通知吗？'
    }

    ElMessageBox.confirm(msg, '提示', { type: 'warning' }).then(() => {
      axios.del(apiPath, { ids: String(id) }).then(() => {
        ElMessage.success('删除成功')
        emits('refresh', type === 'force' ? 'remove' : 'all')
        method.init()
      })
    }).catch(() => {})
  },

  batchDelete(type = '') {
    const ids = method.idsOf()
    if (!ids) return

    const hasBroadcast = state.selection.some(r => method.isBroadcast(r))
    const apiPath = type === 'force' ? '/api/notification/delete' : '/api/notification/remove'
    const msg = type === 'force'
      ? (hasBroadcast ? '选中项包含系统公告，彻底删除后全体用户将永远无法看到，此操作不可恢复！' : '确定要彻底删除选中的通知吗？此操作不可恢复！')
      : (hasBroadcast ? '选中项包含系统公告，删除后全体用户将看不到该公告（可到回收站恢复）。' : '确定要删除选中的通知吗？')

    ElMessageBox.confirm(msg, '提示', { type: 'warning' }).then(() => {
      axios.del(apiPath, { ids }).then(() => {
        ElMessage.success(`已删除 ${state.selection.length} 条`)
        state.selection = []
        emits('refresh', type === 'force' ? 'remove' : 'all')
        method.init()
      })
    }).catch(() => {})
  },

  restore(id) {
    axios.put('/api/notification/restore', { ids: String(id) }).then(() => {
      ElMessage.success('恢复成功')
      emits('refresh', 'all')
      method.init()
    })
  },

  typeLabel(row) {
    if (method.isBroadcast(row)) return '系统公告'
    const map = { comment: '回复', like: '点赞', follow: '关注', system: '系统消息' }
    return map[row.type] || row.type
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

onMounted(async () => {
  state.opts.url = '/api/notification/all'
  state.opts.params = props.params
  if (props.init) await method.init()
})

// 父组件通过 v-model:init 控制首次/切换加载
watch(() => props.init, (val) => {
  if (val) method.init()
})

defineExpose({ init: method.init, show: method.show })
</script>
