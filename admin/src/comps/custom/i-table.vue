<template>
    <el-table ref="selected" v-loading="state.item.loading.data" element-loading-text="数据加载中 ..." element-loading-size="20"
          @selection-change="handle.selectionChange" @row-contextmenu="handle.mouseMenu"
          :data="state.item.data" :row-style="state.config.table.rowStyle" :cell-style="state.config.table.cellStyle"
          :header-row-style="state.config.table.headerRowStyle" :header-cell-style="state.config.table.headerCellStyle"
          :default-sort="state.config.table.defaultSort" :style="state.config.table.style">

        <!-- 自定义列 - 开始位置 -->
        <slot name="start"></slot>

        <!-- 自定义列 - 默认位置 -->
        <slot></slot>

        <!-- 自定义插槽 - 开始 -->
        <el-table-column v-for="(item, index) in state.config.opts.columns" :key="index"
             :prop="item.prop" :label="item.label" :width="item.width" :class-name="item.class"
             :fixed="item.fixed" :sortable="item.sortable" :align="method.inArray(item.prop, ['create_time', 'update_time']) ? 'center' : item.align">
            <!-- 自定义列 - 如果有就定义，否则默认 -->
            <template v-if="item.slot" #default="scope">
                <slot :name="'i-' + item.prop" :scope="scope.row"></slot>
            </template>
            <template v-else #default="scope">
                <span v-if="method.inArray(item.prop, ['create_time', 'update_time'])">
                    <span v-if="!method.empty(scope.row[item.prop])">
                    {{ method.nature(scope.row[item.prop]) }}
                    </span>
                    <strong v-else>-</strong>
                </span>
            </template>
        </el-table-column>
        <!-- 自定义插槽 - 结束 -->

        <slot name="end"></slot>
        <!-- 自定义列 - 结束位置 -->
    </el-table>
    <div class="table-footer">
        <div v-if="state.item.count > 0" class="total-count">
            总计 {{ state.item.count }} 条数据
        </div>
        <div class="pagination-wrapper">
            <el-pagination :background="state.config.pagination.background"
               v-on:size-change="handle.sizeChange"
               v-on:current-change="handle.currentChange"
               :page-sizes="state.config.pagination.sizes"
               :layout="state.config.pagination.layout"
               :popper-class="state.config.pagination.class"
               :pager-count="state.config.pagination.count"
               :hide-on-single-page="state.config.pagination.single"
               :page-size="state.item.limit"
               :page-count="state.item.page.total">
            </el-pagination>
        </div>
    </div>
</template>

<script setup>
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'

const { ctx, proxy } = getCurrentInstance()
const emit = defineEmits(['selection:change'])

const props = defineProps({
    opts: {
        type: Object,
        default: {
            url   : '',
            method: 'get',
            params: {},
            headers: {},
            columns: [],
            menu: {},
        },
        required: true
    },
    table: {
            type: Object,
            default: {
                defaultSort: {
                    prop: 'id',
                    order: 'descending'
                },
                rowStyle: {
                    backgroundColor: 'transparent',
                },
                cellStyle: {
                    backgroundColor: 'transparent',
                    borderBottom: '1px solid var(--el-border-color-light)',
                },
                headerRowStyle: {
                    backgroundColor: 'var(--el-bg-color-page)',
                },
                headerCellStyle: {
                    backgroundColor: 'var(--el-bg-color)',
                    borderBottom: '2px solid var(--el-color-primary)',
                    fontWeight: '600',
                    color: 'var(--el-text-color-primary)',
                },
                style: {
                    background: 'var(--el-bg-color)',
                    borderRadius: '8px',
                    boxShadow: '0 2px 12px rgba(0, 0, 0, 0.08)',
                },
            }
        },
    pagination: {
        type: Object,
        default: {
            count: 5,
            single: true,
            class: 'custom',
            background: true,
            sizes: [10, 50, 100, 500],
            layout: 'sizes, prev, pager, next',
        },
    },
})

const state = reactive({
    item: {
        data: [],
        count: 0,
        page: {
            code: 1,
            total: 1,
        },
        limit: props.pagination.sizes[0],
        order: props.opts?.order || 'create_time asc',
        loading: {
            data: false,
            page: false,
        },
        selection: [],
    },
    // 配置
    config: {
        table: {
            defaultSort: {
                prop: 'id',
                order: 'descending'
            },
            rowStyle: {
                backgroundColor: 'transparent',
            },
            cellStyle: {
                backgroundColor: 'transparent',
                borderBottom: '1px solid var(--el-border-color-light)',
            },
            headerRowStyle: {
                backgroundColor: 'var(--el-bg-color-page)',
            },
            headerCellStyle: {
                backgroundColor: 'var(--el-bg-color)',
                borderBottom: '2px solid var(--el-color-primary)',
                fontWeight: '600',
                color: 'var(--el-text-color-primary)',
            },
            style: {
                background: 'var(--el-bg-color)',
                borderRadius: '8px',
                boxShadow: '0 2px 12px rgba(0, 0, 0, 0.08)',
            },
            ...props.table
        },
        pagination: {
            count: 5,
            single: true,
            class: 'custom',
            background: true,
            sizes: [10, 50, 100, 500],
            layout: 'sizes, prev, pager, next',
            ...props.pagination
        },
        opts: {
            url   : '',
            method: 'get',
            params: {},
            headers: {},
            columns: [],
            menu: {},
            ...props.opts
        },
    },
})

const method = {
    init   : async (page = state.item.page.code, limit = state.item.limit) => {

        // 数据加载中
        state.item.loading.data = true

        // 过滤掉空数组参数（始终读取最新的 props.opts，保证父组件动态修改的参数能生效）
        const params = { ...(props.opts?.params || state.config.opts.params || {}) }
        if (params.where && params.where.length === 0) delete params.where
        if (params.like && params.like.length === 0) delete params.like

        const { data, code, msg } = await axios[props.opts?.method || state.config.opts.method || 'get'](props.opts?.url || state.config.opts.url, {
            page, limit, order: state.item.order, ...params
        })

        // 数据加载失败
        if (!utils.in.array(code, [200, 204])) return ElMessage.error(msg)  // 替换为Element Plus消息提示

        state.item.data       = data.data
        state.item.count      = data.count
        state.item.page.total = data.page

        // 更新页码
        state.item.page.code   = page

        // 数据加载动画
        state.item.loading.data= false
        // 页码加载动画
        state.item.loading.page= false
    },
    // 是否为空
    empty  : value => utils.is.empty(value),
    // 是否在数组中
    inArray: (value, array) => utils.in.array(value, array),
    // 格式化数字
    format : value => utils.format.number(value),
    // 时间戳转人性化时间
    nature : value => utils.time.nature(value),
}

const handle = {
    // 分页大小改变
    sizeChange: val => {
        state.item.limit = val
        method.init()
    },
    // 页码改变
    currentChange: val => method.init(val),
    // 选中
    selectionChange(selection) {
        state.item.selection = selection
        emit('selection:change', selection)
    },
}

// 主动将子组件方法暴露给父组件
defineExpose({
    init: method.init,
})
</script>

<style lang="css" scoped>
.table-footer {
    margin-top: 0.5rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.total-count {
    float: left;
    font-size: 12px;
}
.pagination-wrapper {
    float: right;
}

@media (max-width: 768px) {
    :deep(.el-table) {
        font-size: 12px;
    }

    :deep(.el-table__header),
    :deep(.el-table__body) {
        min-width: 640px;
    }

    :deep(.el-table__header-wrapper),
    :deep(.el-table__body-wrapper) {
        overflow-x: auto;
    }

    :deep(.el-table .el-table__cell) {
        padding: 8px 6px;
    }

    :deep(.el-table .el-table__header .el-table__cell) {
        padding: 10px 6px;
    }

    :deep(.el-pagination) {
        font-size: 11px;
    }

    :deep(.el-pagination .el-pagination__sizes) {
        display: none;
    }

    :deep(.el-pagination .el-pager li) {
        padding: 0 6px;
        min-width: 24px;
        height: 24px;
        line-height: 24px;
    }

    .table-footer {
        flex-direction: column;
        gap: 8px;
        align-items: flex-start;
    }

    .pagination-wrapper {
        width: 100%;
        display: flex;
        justify-content: center;
    }
}
</style>