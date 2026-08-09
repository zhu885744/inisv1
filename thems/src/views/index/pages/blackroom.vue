<template>
    <div class="mt-2">
        <div class="row">
            <div class="col-12">
                <div class="card mb-4">
                    <div class="card-header">
                        <span class="card-title">小黑屋公示</span>
                    </div>
                    <div class="card-body">
                        <p class="text-muted small">本站用户违规处理记录公示，包含封禁原因、时长及处理结果</p>
                    </div>
                </div>

                <!-- Loading -->
                <div v-if="loading" class="text-center py-5">
                    <div class="spinner-border text-muted" role="status">
                        <span class="visually-hidden">加载中...</span>
                    </div>
                </div>

                <!-- Error -->
                <div v-else-if="error" class="text-center py-5">
                    <p class="text-muted">{{ error }}</p>
                    <button class="btn btn-outline-secondary btn-sm" @click="loadData">重新加载</button>
                </div>

                <!-- Empty -->
                <div v-else-if="list.length === 0" class="text-center py-5">
                    <i class="bi bi-shield-check" style="font-size: 3rem; color: #67c23a"></i>
                    <p class="text-muted mt-3">暂无封禁记录，社区氛围良好~</p>
                </div>

                <!-- Ban Record List -->
                <div v-else>
                    <div v-for="item in list" :key="item.id" class="card mb-3 ban-record-card">
                        <div class="card-body py-3">
                            <div class="d-flex justify-content-between align-items-start flex-wrap">
                                <div class="d-flex align-items-center mb-2 mb-md-0">
                                    <!-- 用户头像（脱敏后可能为空） -->
                                    <img v-if="item.result?.user?.avatar"
                                        :src="item.result.user.avatar"
                                        class="rounded-circle me-2"
                                        width="36" height="36"
                                        alt="avatar" />
                                    <span v-else class="rounded-circle me-2 bg-secondary d-flex align-items-center justify-content-center text-white"
                                        style="width: 36px; height: 36px; font-size: 14px">
                                        ?
                                    </span>
                                    <div>
                                        <span class="fw-bold">
                                            {{ item.result?.user?.nickname || '用户' + item.uid }}
                                        </span>
                                        <div class="text-muted small">
                                            {{ formatTime(item.ban_time) }}
                                        </div>
                                    </div>
                                </div>
                                <div class="d-flex align-items-center gap-2">
                                    <span :class="['badge', statusClass(item.status)]">
                                        {{ statusText(item.status) }}
                                    </span>
                                </div>
                            </div>

                            <hr class="my-2" />

                            <div class="row g-2 small">
                                <div class="col-md-4">
                                    <span class="text-muted">违规原因：</span>
                                    <span>{{ item.reason || '违反社区规定' }}</span>
                                </div>
                                <div class="col-md-3">
                                    <span class="text-muted">封禁时长：</span>
                                    <span v-if="item.duration === 0" class="text-danger fw-bold">永久封禁</span>
                                    <span v-else>{{ item.duration }} 天</span>
                                </div>
                                <div class="col-md-3">
                                    <span class="text-muted">违规次数：</span>
                                    <span>第 {{ item.violation_num }} 次</span>
                                </div>
                                <div class="col-md-4 mt-2 mt-md-0">
                                    <span class="text-muted">封禁类型：</span>
                                    <span v-for="bt in item.result?.ban_types" :key="bt.bit" class="badge bg-light text-dark me-1">
                                        {{ bt.name }}
                                    </span>
                                </div>
                            </div>

                            <!-- 到期时间 -->
                            <div v-if="item.status === 0 && item.duration > 0" class="mt-2 small text-muted">
                                预计解封时间：{{ formatTime(item.expires_at) }}
                            </div>
                        </div>
                    </div>

                    <!-- Pagination -->
                    <div class="text-center mt-4">
                        <button class="btn btn-outline-secondary btn-sm"
                            :disabled="page <= 1"
                            @click="prevPage">上一页</button>
                        <span class="mx-2 small text-muted">第 {{ page }} / {{ totalPage }} 页</span>
                        <button class="btn btn-outline-secondary btn-sm"
                            :disabled="page >= totalPage"
                            @click="nextPage">下一页</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { request, cache } from '@/utils/network'
import { usePageTitle } from '@/utils/app'

usePageTitle('小黑屋')

const loading = ref(true)
const error = ref('')
const list = ref([])
const page = ref(1)
const totalCount = ref(0)
const pageSize = 12

const totalPage = computed(() => Math.max(1, Math.ceil(totalCount.value / pageSize)))

const statusMap = {
    0: { text: '生效中', class: 'bg-danger' },
    1: { text: '已解封', class: 'bg-success' },
    2: { text: '已撤销', class: 'bg-secondary' },
    3: { text: '申诉中', class: 'bg-warning' },
    4: { text: '申诉通过', class: 'bg-success' },
    5: { text: '申诉驳回', class: 'bg-danger' },
}

function statusText(status) {
    return statusMap[status]?.text || '未知'
}
function statusClass(status) {
    return statusMap[status]?.class || 'bg-secondary'
}

function formatTime(ts) {
    if (!ts || ts === 0) return '-'
    const d = new Date(ts * 1000)
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

async function loadData() {
    loading.value = true
    error.value = ''

    const cacheKey = `blackroom_${page.value}`

    try {
        // 尝试读缓存（10分钟有效）
        const cached = cache.get(cacheKey)
        if (cached) {
            list.value = cached.data
            totalCount.value = cached.count
            loading.value = false
            return
        }

        const res = await request.get('/api/users/blackroom', {
            page: page.value,
            limit: pageSize,
            order: 'create_time desc',
        })

        if (res.code === 200 && res.data) {
            list.value = res.data.data || []
            totalCount.value = res.data.count || 0

            // 缓存10分钟
            cache.set(cacheKey, { data: list.value, count: totalCount.value }, 10)
        } else if (res.code === 204) {
            list.value = []
            totalCount.value = 0
        } else {
            throw new Error(res.msg || '加载失败')
        }
    } catch (err) {
        console.error('小黑屋数据加载失败:', err)
        error.value = '加载失败，请稍后重试'
    }
    loading.value = false
}

function prevPage() {
    if (page.value > 1) {
        page.value--
        loadData()
    }
}

function nextPage() {
    if (page.value < totalPage.value) {
        page.value++
        loadData()
    }
}

onMounted(() => {
    loadData()
})
</script>

<style scoped>
.ban-record-card {
    border-left: 4px solid #e6a23c;
    transition: box-shadow 0.2s;
}
.ban-record-card:hover {
    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}
</style>
