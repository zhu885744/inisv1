<template>
    <div>
        <div style="margin-bottom: 12px" v-if="state.item.selection.length > 0 && props.type === 'all'">
            <el-button v-on:click="method.batchStatus(0)" type="success" size="small">
                <i-svg color="rgb(var(--icon-color))" name="finish" size="16px"></i-svg>
                <span style="margin-left: 4px">批量正常</span>
            </el-button>
            <el-button v-on:click="method.batchStatus(1)" type="warning" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="lock" size="16px"></i-svg>
                <span style="margin-left: 4px">批量冻结</span>
            </el-button>
            <el-button v-on:click="method.batchDelete(true)" type="danger" size="small" style="margin-left: 8px">
                <i-svg color="rgb(var(--icon-color))" name="delete" size="16px"></i-svg>
                <span style="margin-left: 4px">批量删除</span>
            </el-button>
        </div>
    <i-table :opts="state.opts" ref="i-table" @selection:change="method.selectionChange">
        <template #start>
            <el-table-column type="selection" width="55"></el-table-column>
        </template>

        <template v-if="props.type === 'all'" #end>
            <el-table-column :fixed="right" label="操作" width="180" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.edit(scope.row)" class="custom" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, true)" size="small" style="margin-left: 0" :disabled="scope.row.id === 1">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                        <el-button v-if="!scope.row.current_ban_id" v-on:click="method.ban(scope.row)" size="small" style="margin-left: 0; color: #e6a23c" :disabled="scope.row.id === 1">
                            <el-icon style="font-size: 16px"><Warning /></el-icon>
                        </el-button>
                        <el-button v-if="scope.row.current_ban_id" v-on:click="method.unban(scope.row)" size="small" style="margin-left: 0; color: #67c23a" :disabled="scope.row.id === 1">
                            <el-icon style="font-size: 16px"><CircleCheck /></el-icon>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>
        <template v-if="props.type === 'remove'">
            <el-table-column :fixed="right" label="操作" width="160" class-name="text-end">
                <template #default="scope">
                    <span style="display: flex; justify-content: flex-end">
                        <el-button v-on:click="method.restore(scope.row.id)" class="custom" size="small">
                            <i-svg color="rgb(var(--icon-color))" name="restore" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.edit(scope.row)" size="small" style="margin-left: 0">
                            <i-svg color="rgb(var(--icon-color))" name="edit" size="16px"></i-svg>
                        </el-button>
                        <el-button v-on:click="method.delete(scope.row.id, false)" size="small" style="margin-left: 0":disabled="scope.row.id === 1">
                            <i-svg color="rgb(var(--icon-color))" name="delete" size="21px"></i-svg>
                        </el-button>
                    </span>
                </template>
            </el-table-column>
        </template>

        <template #i-nickname="{ scope = {} }">
            <span v-on:dblclick="method.edit(scope)" style="display: flex; align-items: center">
                <el-tooltip :disabled="utils.is.empty(scope.description)" placement="top">
                    <template #content>
                        <span v-html="method.autoWrap(scope.nickname + '：' + scope.description)"></span>
                    </template>
                    <el-avatar shape="square" :src="method.imageSize(scope?.avatar)" size="small" style="margin-right: 4px"></el-avatar>
                </el-tooltip>
                <el-tooltip v-if="!utils.is.empty(scope.url)" :content="`链接：${scope.url}`" placement="top">
                    <i-svg color="rgb(var(--icon-color))" v-on:click="method.window(scope.url)" name="link" size="12px" style="margin-right: 4px"></i-svg>
                </el-tooltip>
                <el-tooltip :content="scope.nickname" :disabled="utils.is.empty(scope.nickname)" placement="top">
                    <span>{{ method.omit(scope?.nickname, 4, ' ...', 'end') }}</span>
                </el-tooltip>
            </span>
        </template>

        <template #i-account="{ scope = {} }">
            <el-tooltip :content="'双击复制：' + scope?.account" :disabled="utils.is.empty(scope?.account)" placement="top">
                <span v-on:dblclick="method.copy(scope?.account, '复制成功！')">
                    {{ method.omit(scope?.account) }}
                </span>
            </el-tooltip>
        </template>

        <template #i-email="{ scope = {} }">
            <el-tooltip :content="'双击复制：' + scope?.email" :disabled="utils.is.empty(scope?.email)" placement="top">
                <span v-on:dblclick="method.copy(scope?.email, '复制成功！')">
                    {{ method.omit(scope?.email, 9) }}
                </span>
            </el-tooltip>
        </template>

        <template #i-phone="{ scope = {} }">
            <el-tooltip :content="'双击复制：' + scope?.phone" :disabled="utils.is.empty(scope?.phone)" placement="top">
                <span v-on:dblclick="method.copy(scope?.phone, '复制成功！')">
                    {{ method.omit(scope?.phone, 7, '***') }}
                </span>
            </el-tooltip>
        </template>

        <template #i-login_time="{ scope = { login_time: 0 } }">
            <span v-if="scope?.login_time === 0">从未登录</span>
            <span v-else>{{ utils.time.nature(scope?.login_time) }}</span>
        </template>

        <template #i-remark="{ scope = {} }">
            <el-tooltip :disabled="utils.is.empty(scope.remark)" placement="top">
                <template #content>
                    <span v-html="method.autoWrap(scope.remark)"></span>
                </template>
                <span v-on:dblclick="method.copy(scope.remark, '复制成功！')">
                    {{ method.omit(scope?.remark) }}
                </span>
            </el-tooltip>
        </template>

        <template #i-status="{ scope = {} }">
            <el-tag :type="scope.status === 0 ? 'success' : 'danger'" class="cursor-default">
                {{ scope.status === 0 ? '正常' : '冻结' }}
            </el-tag>
        </template>
    </i-table>
    </div>

    <el-dialog v-model="state.item.dialog" title="用户信息编辑" width="800px" class="custom" draggable :close-on-click-modal="false">
        <div style="padding: 10px 0;">
            <el-form label-width="120px" label-position="left">
                <el-form-item label="昵称">
                    <el-input v-model="state.struct.nickname" placeholder="请输入用户昵称"></el-input>
                </el-form-item>
                <el-form-item label="账号">
                    <el-input v-model="state.struct.account" placeholder="请输入登录账号"></el-input>
                </el-form-item>
                <el-form-item label="账号状态">
                    <el-select v-model="state.struct.status" :disabled="state.struct.id === 1">
                        <el-option label="正常" :value="0"></el-option>
                        <el-option label="冻结" :value="1"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="头像">
                    <el-input v-model="state.struct.avatar" class="custom" placeholder="图片地址/点击上传">
                        <template #append>
                            <el-button @click="method.upload('avatar')" :loading="state.item.upload" class="upload-btn">
                                <i-svg v-if="!state.item.upload" name="upload" color="rgb(var(--icon-color))" size="14px"></i-svg>
                                <span style="margin-left: 4px">上传</span>
                            </el-button>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item label="邮箱">
                    <el-input v-model="state.struct.email" type="email" placeholder="请输入邮箱地址"></el-input>
                </el-form-item>
                <el-form-item label="手机号">
                    <el-input v-model="state.struct.phone" placeholder="请输入手机号码"></el-input>
                </el-form-item>
                <el-form-item label="密码">
                    <el-input v-model="state.struct.password" type="password" placeholder="为空不修改密码"></el-input>
                </el-form-item>
                <el-form-item label="权限">
                    <el-select v-model="state.struct.result.auth.group.ids" multiple collapse-tags placeholder="请选择权限组">
                        <el-option v-for="item in state.select.auth_group" :key="item.value" :label="item.label" :value="item.value">
                            <span style="font-size: 13px">{{ item.label }}</span>
                        </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="性别">
                    <el-select v-model="state.struct.gender" placeholder="请选择性别">
                        <el-option label="保密" :value="null"></el-option>
                        <el-option label="男" value="boy"></el-option>
                        <el-option label="女" value="girl"></el-option>
                    </el-select>
                </el-form-item>
                <el-form-item label="专属头衔">
                    <el-input v-model="state.struct.title" placeholder="请输入专属头衔"></el-input>
                </el-form-item>
                <el-form-item label="个人简介">
                    <el-input v-model="state.struct.description" :autosize="{ minRows: 3, maxRows: 5 }" type="textarea" placeholder="请输入个人简介"></el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="state.struct.remark" :autosize="{ minRows: 2, maxRows: 4 }" type="textarea" placeholder="请输入备注信息（仅内部可见）"></el-input>
                </el-form-item>
            </el-form>
        </div>

        <template #footer>
            <div style="text-align: right; padding: 10px 0;">
                <el-button @click="state.item.dialog = false" size="default" style="margin-right: 10px;">取 消</el-button>
                <el-button @click="method.save()" :loading="state.item.wait" type="primary" size="default">保 存</el-button>
            </div>
        </template>
    </el-dialog>

    <!-- 封禁对话框 -->
    <el-dialog v-model="state.banDialog.visible" title="拉入小黑屋" width="620px" class="custom" draggable :close-on-click-modal="false">
        <div style="padding: 10px 0;">
            <el-alert :title="`拉黑用户：${state.banDialog.target?.nickname || ''} (ID: ${state.banDialog.target?.id || ''})`" type="warning" :closable="false" show-icon style="margin-bottom: 16px" />
            <el-text type="info" size="small" style="display: block; margin-bottom: 16px;">
                <i class="bi bi-info-circle me-1"></i> 将用户拉入小黑屋后，用户将失去对应的发布及操作权限。
            </el-text>

            <el-form label-width="120px" label-position="left">
                <!-- 封禁原因（预设+自定义） -->
                <el-form-item label="封禁原因" required>
                    <div style="display: flex; flex-wrap: wrap; gap: 8px; margin-bottom: 10px;">
                        <el-tag
                            v-for="r in state.banDialog.presetReasons"
                            :key="r"
                            :type="state.banDialog.reason === r ? 'warning' : 'info'"
                            :effect="state.banDialog.reason === r ? 'dark' : 'plain'"
                            style="cursor: pointer;"
                            @click="state.banDialog.reason = r"
                        >{{ r }}</el-tag>
                        <el-tag
                            :type="state.banDialog.isCustomReason ? 'warning' : 'info'"
                            :effect="state.banDialog.isCustomReason ? 'dark' : 'plain'"
                            style="cursor: pointer;"
                            @click="method.toggleCustomReason()"
                        >其他原因</el-tag>
                    </div>
                    <el-input
                        v-if="state.banDialog.isCustomReason"
                        v-model="state.banDialog.customReason"
                        type="textarea"
                        :rows="2"
                        placeholder="请填写具体原因..."
                        @input="state.banDialog.reason = state.banDialog.customReason"
                    />
                </el-form-item>

                <el-divider content-position="left" style="margin: 12px 0;">
                    <el-text type="info" size="small">权限限制</el-text>
                </el-divider>

                <!-- 封禁类型 -->
                <el-form-item label="封禁类型">
                    <el-checkbox-group v-model="state.banDialog.banTypes">
                        <el-checkbox :value="1" label="限制登录" style="margin-right: 12px" />
                        <el-checkbox :value="2" label="限制发文" style="margin-right: 12px" />
                        <el-checkbox :value="4" label="限制评论" style="margin-right: 12px" />
                        <el-checkbox :value="8" label="限制上传" style="margin-right: 12px" />
                        <el-checkbox :value="16" label="限制互动" />
                    </el-checkbox-group>
                    <div style="margin-top: 4px; font-size: 12px; color: #909399">
                        可多选，不选则默认全部封禁
                    </div>
                </el-form-item>

                <el-form-item label="封禁模式">
                    <el-radio-group v-model="state.banDialog.mode">
                        <el-radio value="auto">自动梯度</el-radio>
                        <el-radio value="manual">手动指定</el-radio>
                    </el-radio-group>
                </el-form-item>

                <el-form-item v-if="state.banDialog.mode === 'manual'" label="封禁时长">
                    <el-select v-model="state.banDialog.duration" placeholder="请选择封禁时长">
                        <el-option label="1 天" :value="1" />
                        <el-option label="3 天" :value="3" />
                        <el-option label="7 天" :value="7" />
                        <el-option label="15 天" :value="15" />
                        <el-option label="30 天" :value="30" />
                        <el-option label="永久封禁" :value="0" />
                    </el-select>
                </el-form-item>

                <el-form-item v-if="state.banDialog.mode === 'auto'" label="梯度规则">
                    <div style="font-size: 12px; color: #909399; line-height: 1.8">
                        当前累计违规 {{ state.banDialog.target?.ban_count || 0 }} 次<br/>
                        <span v-if="(state.banDialog.target?.ban_count || 0) === 0">自动封禁 <b>1 天</b>（首次）</span>
                        <span v-else-if="(state.banDialog.target?.ban_count || 0) === 1">自动封禁 <b>7 天</b>（二次）</span>
                        <span v-else-if="(state.banDialog.target?.ban_count || 0) === 2">自动封禁 <b>15 天</b>（三次）</span>
                        <span v-else-if="(state.banDialog.target?.ban_count || 0) === 3">自动封禁 <b>30 天</b>（四次）</span>
                        <span v-else style="color: #f56c6c">自动封禁 <b>永久</b>（五次及以上，禁止申诉）</span>
                    </div>
                </el-form-item>

                <el-divider content-position="left" style="margin: 12px 0;">
                    <el-text type="info" size="small">高级选项</el-text>
                </el-divider>

                <el-form-item label="删除全部内容">
                    <el-switch v-model="state.banDialog.deleteContent" active-text="是" inactive-text="否" />
                    <el-text type="danger" size="small" style="margin-left: 8px;">
                        <i class="bi bi-exclamation-triangle me-1"></i>移至回收站（文章/动态/评论/点赞/收藏）
                    </el-text>
                </el-form-item>

                <el-form-item label="禁止申诉">
                    <el-switch v-model="state.banDialog.banAppeal" active-text="是" inactive-text="否" />
                    <el-text type="info" size="small" style="margin-left: 8px;">开启后将不允许用户提交申诉</el-text>
                </el-form-item>

                <el-divider content-position="left" style="margin: 12px 0;">
                    <el-text type="info" size="small">附加信息</el-text>
                </el-divider>

                <el-form-item label="封禁证据">
                    <el-input v-model="state.banDialog.evidence" placeholder="文本说明或链接（可选）" />
                </el-form-item>
            </el-form>
        </div>

        <template #footer>
            <div style="text-align: right; padding: 10px 0;">
                <el-button @click="state.banDialog.visible = false" size="default" style="margin-right: 10px;">取 消</el-button>
                <el-button @click="method.doBan()" :loading="state.banDialog.loading" type="danger" size="default">
                    <i class="bi bi-slash-circle me-1"></i>确认拉黑
                </el-button>
            </div>
        </template>
    </el-dialog>
</template>

<script setup>
import { reactive, computed, watch, onMounted, getCurrentInstance } from 'vue'
import utils from '{src}/utils/utils.js'
import axios from '{src}/utils/request.js'
import ITable from '{src}/comps/custom/i-table.vue'

const emit  = defineEmits(['refresh','update:init'])
const props = defineProps({
    type: {
        type: String,
        default: 'all',
    },
    params: {
        type: Object,
        default: () => ({
            order: 'id asc',
        }),
    },
    init: {
        type: Boolean,
        default: false,
    }
})

// table - fixed
const left = computed(() => {
    let result = 'left'
    if (utils.is.mobile()) result = false
    return result
})
const right = computed(() => {
    let result = 'right'
    if (utils.is.mobile()) result = false
    return result
})

const { ctx, proxy } = getCurrentInstance()
const state  = reactive({
    item: {
        table: 'users',
        dialog: false,
        upload: false,
        wait: false,
        selection: [],
    },
    banDialog: {
        visible: false,
        loading: false,
        target: null,
        banTypes: [1, 2, 4, 8, 16], // 默认全选
        mode: 'auto', // auto | manual
        duration: 7, // 手动模式默认7天
        reason: '',
        evidence: '',
        // 新增
        deleteContent: false,   // 删除全部内容
        banAppeal: false,       // 禁止申诉
        // 预设原因
        presetReasons: ['发布色情、违法内容', '存在欺诈骗钱行为', '骚扰他人', '涉嫌侵权', '发布垃圾广告信息'],
        isCustomReason: false,
        customReason: '',
    },
    struct: {
        id: null, // 存储当前编辑用户的ID
        remark: null,
        status: 0, // 默认值为数字0，对应"正常"
        result: {
            auth: {
                all: false,
                root: false,
                group: {
                    ids: [],
                    list: [],
                },
            }
        }
    },
    opts: {
        url: '/api/users/all',
        params: props.params,
        columns: [
            { prop: 'id', label: 'ID', width: 80, align: 'center' },
            { prop: 'nickname',label: '昵称', width: 130, slot: true, fixed: left },
            { prop: 'account', label: '账号', width: 130, slot: true },
            { prop: 'email',   label: '邮箱', slot: true },
            { prop: 'phone',   label: '手机号', slot: true },
            { prop: 'status',  label: '状态', width: 100, slot: true },
            { prop: 'remark' , label: '备注', slot: true },
            { prop: 'login_time', label: '最近登录', width: 140, sortable: true, slot: true },
            { prop: 'create_time', label: '创建时间', width: 140, sortable: true },
        ],
    },
    select: {
        auth_group: [],
        gender: [
            { value: null, label: '保密'},
            { value: 'boy', label: '男' },
            { value: 'girl', label: '女' },
        ]
    },
})

const method = {
    // 刷新数据
    init: async () => {
        state.struct = {
            id: null, // 重置用户ID
            remark: null,
            status: 0, // 重置为数字0，显示"正常"
            result: {
                auth: {
                    all: false,
                    root: false,
                    group: {
                        ids: [],
                        list: [],
                    },
                }
            }
        }
        await proxy.$refs['i-table']['init']()
    },
    // 获取权限分组
    async authGroup() {
        const { code, data } = await axios.get('/api/auth-group/column',{
            field: 'id,name'
        })
        if (code !== 200) return
        state.select.auth_group = data.map(item => ({ value: item.id, label: item.name }))
    },
    // 更新用户权限
    async updateAuthGroup(uid = null, ids = []) {
        if (utils.is.empty(uid)) return
        const { code, msg } = await axios.put('/api/auth-group/uids', { uid, ids })
        if (code !== 200) return ElMessage.error(msg)
    },
    // 保存数据 - 仅限制状态修改
    save: async (params = state.struct || {}) => {
        try {
            if (utils.is.empty(params)) return ElMessage.warning('表单数据不能为空！')
            if (utils.is.empty(params?.account)) return ElMessage.warning('账号为必填项！')
            
            // 邮箱格式验证
            if (!utils.is.empty(params?.email) && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(params.email)) {
                return ElMessage.warning('邮箱格式不正确！')
            }
            
            // 手机号格式验证
            if (!utils.is.empty(params?.phone) && !/^1[3-9]\d{9}$/.test(params.phone)) {
                return ElMessage.warning('手机号格式不正确！')
            }

            state.item.wait = true

            const { code, msg, data } = await axios.post(`/api/${state.item.table}/save`, params)

            state.item.wait = false

            if (code !== 200) throw new Error(msg || '保存失败')
            
            ElMessage.success('保存成功！')
            state.item.dialog = false
            await method.init()
            await method.updateAuthGroup(data.id, params.result.auth.group.ids)
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '保存失败，请重试')
        }
    },
    // 编辑数据
    edit: struct => {
        // 确保status是数字类型，匹配下拉框的value类型
        const editStruct = {
            ...struct,
            status: Number(struct.status) // 强制转换为数字
        }
        state.struct = editStruct
        state.item.dialog = true
    },
    // 显示盒子（新增用户）
    show: () => {
        method.init()
        state.item.dialog = true
    },
    // 真删 和 软删 - 禁用超级管理员删除
    async delete(ids = [], isSoft = true) {
        // 禁止删除ID为1的用户
        if (ids === 1 || (Array.isArray(ids) && ids.includes(1))) {
            ElMessage.error('禁止删除系统管理员！')
            return
        }
        
        if (utils.is.empty(ids)) return
        
        const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`
        try {
            const { code, msg } = await axios.del(uri, { ids })
            if (code !== 200) throw new Error(msg)
            
            ElMessage.success(isSoft ? '软删除成功！' : '永久删除成功！')
            emit('refresh', 'remove')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '删除失败')
        }
    },
    // 恢复数据
    async restore(ids = []) {
        if (utils.is.empty(ids)) return

        try {
            const { code, msg } = await axios.put(`/api/${state.item.table}/restore`, { ids })
            if (code !== 200) throw new Error(msg)

            ElMessage.success('恢复成功！')
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '恢复失败')
        }
    },
    // 选择变化
    selectionChange(selection) {
        state.item.selection = selection
    },
    // 批量修改状态（冻结/正常）
    async batchStatus(status = 0) {
        // 过滤掉系统管理员（ID=1）
        const items = state.item.selection.filter(item => item.id !== 1)

        if (utils.is.empty(items)) {
            return ElMessage.warning('请选择要操作的用户（系统管理员不可操作）')
        }

        state.item.wait = true
        try {
            // 根据 API 规范，status 接口需要单个 id + status
            const failed = []
            for (const item of items) {
                // 同时传 data 和 params，确保兼容性
                const payload = {
                    id    : Number(item.id),
                    status: Number(status),
                }
                const { code, msg } = await axios.put(`/api/${state.item.table}/status`, payload, {
                    params: payload
                })
                if (code !== 200) failed.push({ id: item.id, msg })
            }

            state.item.wait = false

            if (failed.length > 0) {
                return ElMessage.error(`操作完成，失败 ${failed.length} 个：${failed[0].msg}`)
            }

            ElMessage.success(status === 0 ? '已批量设为正常' : '已批量冻结')
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '操作失败')
        }
    },
    // 批量删除
    async batchDelete(isSoft = true) {
        // 过滤掉系统管理员（ID=1）
        const ids = state.item.selection
            .filter(item => item.id !== 1)
            .map(item => item.id)

        if (utils.is.empty(ids)) {
            return ElMessage.warning('请选择要删除的用户（系统管理员不可删除）')
        }

        try {
            await ElMessageBox.confirm(
                `确定要${isSoft ? '软删除' : '永久删除'}选中的 ${ids.length} 个用户吗？`,
                '提示',
                { type: 'warning' }
            )
        } catch {
            return
        }

        state.item.wait = true
        try {
            const uri = `/api/${state.item.table}/${isSoft ? 'remove' : 'delete'}`
            const { code, msg } = await axios.del(uri, { ids })
            state.item.wait = false
            if (code !== 200) throw new Error(msg)

            ElMessage.success(isSoft ? '批量软删除成功！' : '批量永久删除成功！')
            emit('refresh', 'remove')
            await method.init()
        } catch (error) {
            state.item.wait = false
            ElMessage.error(error.message || '删除失败')
        }
    },
    // 上传
    async upload(field = 'image') {
        try {
            const input = document.createElement('input')
            input.type = 'file'
            input.accept = 'image/*'
            
            input.addEventListener('change', async () => {
                if (!input.files.length) return

                const file = input.files[0]
                const { code: checkCode, data: checkData } = await axios.post('/api/attachment/checkType', { file_names: [file.name] })
                if (checkCode !== 200) {
                    ElMessage.error('文件类型检查失败')
                    return
                }
                const result = checkData.results?.[0]
                if (!result?.is_allowed) {
                    ElMessage.error(result?.message || '不允许上传该类型的文件')
                    return
                }
                
                const params = new FormData()
                params.append('files', file)
                params.append('target_type', 'user_avatar')
                
                state.item.upload = true
                const { code, msg, data } = await axios.post('/api/attachment/batch', params)
                state.item.upload = false
                
                if (code !== 200) throw new Error(msg)
                
                state.struct[field] = data.results[0]?.full_url || ''
                ElMessage.info('上传成功！')
            })
            
            input.click()
        } catch (error) {
            state.item.upload = false
            ElMessage.error(error.message || '上传失败')
        }
    },
    window(url = null, target = '_blank') {
        if (utils.is.empty(url)) return
        globalThis.open(url, target)
    },
    // 打开封禁对话框
    ban(target) {
        if (!target || target.id === 1) {
            ElMessage.error('禁止封禁系统管理员！')
            return
        }
        state.banDialog.target = target
        state.banDialog.banTypes = [1, 2, 4, 8, 16]
        state.banDialog.mode = 'auto'
        state.banDialog.duration = 7
        state.banDialog.reason = ''
        state.banDialog.evidence = ''
        state.banDialog.deleteContent = false
        state.banDialog.banAppeal = false
        state.banDialog.isCustomReason = false
        state.banDialog.customReason = ''
        state.banDialog.visible = true
    },
    // 切换自定义原因
    toggleCustomReason() {
        state.banDialog.isCustomReason = !state.banDialog.isCustomReason
        if (!state.banDialog.isCustomReason) {
            state.banDialog.customReason = ''
            state.banDialog.reason = ''
        }
    },
    // 执行封禁
    async doBan() {
        const target = state.banDialog.target
        if (!target) return

        // 校验原因
        const reason = state.banDialog.isCustomReason ? state.banDialog.customReason : state.banDialog.reason
        if (!reason.trim()) {
            ElMessage.error('请选择或填写封禁原因！')
            return
        }

        let banType = 0
        for (const bit of state.banDialog.banTypes) {
            banType |= bit
        }
        // 未选任何类型则全部封禁
        if (banType === 0) banType = 31

        const params = {
            uid: target.id,
            ban_type: banType,
            reason: reason.trim(),
            evidence: state.banDialog.evidence,
            delete_content: state.banDialog.deleteContent ? 1 : 0,
            ban_appeal: state.banDialog.banAppeal ? 1 : 0,
        }

        if (state.banDialog.mode === 'auto') {
            params.auto_gradient = true
        } else {
            params.duration = state.banDialog.duration
        }

        // 删除全部内容二次确认
        if (state.banDialog.deleteContent) {
            try {
                await ElMessageBox.confirm(
                    `确定要删除该用户全部内容吗？文章、动态、评论等将移入回收站。`,
                    '确认删除内容',
                    { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'warning' }
                )
            } catch {
                return // 用户取消
            }
        }

        state.banDialog.loading = true
        try {
            const { code, msg } = await axios.put('/api/users/ban', params)
            if (code !== 200) throw new Error(msg)
            ElMessage.success('拉黑成功！')
            state.banDialog.visible = false
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '拉黑失败')
        }
        state.banDialog.loading = false
    },
    // 解封用户
    async unban(target) {
        if (!target || !target.id) return
        try {
            await ElMessageBox.confirm(
                `确定要解封用户「${target.nickname}」吗？`,
                '解封确认',
                { type: 'warning' }
            )
        } catch {
            return
        }

        try {
            const { code, msg } = await axios.put('/api/users/unban', { uid: target.id })
            if (code !== 200) throw new Error(msg)
            ElMessage.success('解封成功！')
            emit('refresh', 'all')
            await method.init()
        } catch (error) {
            ElMessage.error(error.message || '解封失败')
        }
    },
    imageSize(url = '', size = '50x50') {
        if (utils.is.empty(url)) return url
        return url.includes('?') ? `${url}&size=${size}` : `${url}?size=${size}`
    },
    autoWrap(text = '', length = 40, symbol = '<br>') {
        if (utils.is.empty(text)) return text
        return text.replace(new RegExp(`(.{${length}})`, 'g'), `$1${symbol}`)
    },
    copy: (text = null, msg = '复制成功！') => {
        if (utils.is.empty(text)) return
        
        utils.set.copy.text(text)
        ElMessage.info(msg)
    },
    omit: (text = null, length = 10, omission = ' ...', location = 'center') => {
        if (utils.is.empty(text)) return '-'
        return utils.string.omit(text, length, omission, location)
    },
}

onMounted(async () => {
    await method.authGroup()
    if (props.init) await method.init()
})

watch(() => props.init, (val) => {
    if (val) method.init()
})

// 监听对话框关闭，重置表单
watch(() => state.item.dialog, (value) => {
    if (!value) {
        state.struct = {
            id: null, // 重置用户ID
            remark: null,
            status: 0, // 重置为数字0，显示"正常"
            result: {
                auth: {
                    all: false,
                    root: false,
                    group: {
                        ids: [],
                        list: [],
                    },
                }
            }
        }
    }
})

defineExpose({
    init: method.init,
    show: method.show,
})
</script>