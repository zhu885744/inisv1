<template>
  <div class="sec-page">
    <!-- ============ 概览条：一眼看到两条链路是否在跑 ============ -->
    <header class="card card-pad hero">
      <div class="hero-main">
        <span class="hero-badge"><i class="bi bi-shield-check" /></span>
        <div class="hero-text">
          <h2 class="hero-title">内容审核</h2>
          <p class="hero-desc">
            本地敏感词库同步快筛，云端机审异步兜底（变体词 / 长文尾部 / 图片）。
            违规内容直接下架，疑似内容转人工复核
          </p>
        </div>
      </div>

      <div class="hero-side">
        <span class="pill" :class="cfg.sensitive.enabled ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> 本地词库{{ cfg.sensitive.enabled ? '已启用' : '未启用' }}
        </span>
        <span class="pill is-quiet">版本 v{{ cfg.sensitive.version }}</span>
        <span class="pill" :class="auditActive ? 'is-on' : 'is-off'">
          <i class="bi bi-dot" /> 云端机审{{ auditText }}
        </span>
        <button class="btn btn-sm" :disabled="loading" @click="loadAll">
          <i class="bi bi-arrow-clockwise" /> 刷新
        </button>
      </div>
    </header>

    <!-- ============ 分段：敏感词库 / 云端机审 ============ -->
    <nav class="seg" role="tablist">
      <button
        v-for="item in PANES"
        :key="item.key"
        type="button"
        class="seg-item"
        :class="{ 'is-active': pane === item.key }"
        role="tab"
        :aria-selected="pane === item.key"
        @click="pane = item.key"
      >
        <i :class="item.icon" />
        <span>{{ item.label }}</span>
        <span v-if="item.key === 'sensitive'" class="seg-count">{{ cfg.sensitive.rules.length }}</span>
      </button>
    </nav>

    <div v-if="loading" class="tip-line"><span class="spinner" /> 正在读取词库与机审配置…</div>

    <template v-else>
      <!-- ==================== 敏感词库 ==================== -->
      <template v-if="pane === 'sensitive'">
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">词库开关与缺省策略</h3>
              <p class="card-desc">独立 config/sensitive.toml，保存后立即热更新，无需重启</p>
            </div>
            <div class="seg seg-sm">
              <button
                type="button"
                class="seg-item"
                :class="{ 'is-active': cfg.sensitiveMode === 'form' }"
                @click="cfg.sensitiveMode = 'form'"
              >
                <i class="bi bi-ui-checks" /> 可视化
              </button>
              <button
                type="button"
                class="seg-item"
                :class="{ 'is-active': cfg.sensitiveMode === 'source' }"
                @click="cfg.sensitiveMode = 'source'"
              >
                <i class="bi bi-code-slash" /> 源码
              </button>
            </div>
          </div>

          <template v-if="cfg.sensitiveMode === 'form'">
            <div class="switch-row">
              <div>
                <div class="row-title">启用词库</div>
                <p class="row-hint">关闭后所有场景直接放行，规则仍然保留</p>
              </div>
              <FormCheckbox variant="switch" v-model="cfg.sensitive.enabled" />
            </div>

            <div class="grid-2">
              <div class="form-item">
                <label class="form-label">缺省动作</label>
                <FormSelect
                  v-model="cfg.sensitive.default_action"
                  :options="SENSITIVE_ACTION_OPTIONS" />
                <p class="row-hint">规则未单独指定动作时使用，建议保持「进审核」</p>
              </div>
              <div class="form-item">
                <label class="form-label">当前版本</label>
                <FormInput :value="cfg.sensitive.version" readonly />
                <p class="row-hint">每次保存自动 +1，命中审计按此版本回溯误伤</p>
              </div>
            </div>
          </template>

          <div v-else class="form-item">
            <label class="form-label">规则文件（TOML）</label>
            <FormTextarea
              v-model="cfg.sensitiveContent"
              class="code"
              rows="20"
              spellcheck="false" />
            <p class="row-hint">
              每条 [[rules]] 支持 word / action / scenes / level / match；action 可选
              block / limit / review / warn / log；[[whitelist]] 命中即放行。保存前会做 TOML
              语法校验，可视化模式保存会按模板重写文件、手写注释不保留
            </p>
          </div>
        </section>

        <!-- 白名单 -->
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">
                白名单 <span class="count">{{ cfg.sensitive.whitelist.length }}</span>
              </h3>
              <p class="card-desc">命中即放行，优先于规则（官方公告、认证机构、业务模板）</p>
            </div>
            <button class="btn btn-sm" @click="addSensitiveRule('whitelist')">
              <i class="bi bi-plus-lg" /> 添加
            </button>
          </div>

          <p v-if="!cfg.sensitive.whitelist.length" class="empty">还没有白名单词条</p>

          <div v-else class="rule-list">
            <article
              v-for="(rule, index) in cfg.sensitive.whitelist"
              :key="`whitelist-${index}`"
              class="rule"
            >
              <div class="rule-top">
                <span class="rule-no">{{ index + 1 }}</span>
                <div class="rule-fields">
                  <div class="form-item is-flat">
                    <label class="form-label">词条</label>
                    <FormInput v-model="rule.word" maxlength="64" placeholder="如：官方公告" />
                  </div>
                  <div class="form-item is-flat">
                    <label class="form-label">匹配方式</label>
                    <FormSelect v-model="rule.match" :options="SENSITIVE_MATCH_OPTIONS" />
                  </div>
                </div>
                <button class="icon-btn" title="删除" @click="removeSensitiveRule('whitelist', index)">
                  <i class="bi bi-trash3" />
                </button>
              </div>

              <div class="rule-scenes">
                <span class="scenes-label">生效场景</span>
                <div class="scene-chips">
                  <button
                    v-for="scene in SENSITIVE_SCENES"
                    :key="scene.value"
                    type="button"
                    class="scene-chip"
                    :class="{ 'is-active': rule.scenes.includes(scene.value) }"
                    @click="toggleSensitiveScene(rule, scene.value)"
                  >
                    <i v-if="rule.scenes.includes(scene.value)" class="bi bi-check2" />
                    {{ scene.label }}
                  </button>
                  <span class="scenes-hint">不选 = 全部场景</span>
                </div>
              </div>
            </article>
          </div>
        </section>

        <!-- 词条规则 -->
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">
                词条规则 <span class="count">{{ cfg.sensitive.rules.length }}</span>
              </h3>
              <p class="card-desc">
                按场景决定处置；同时命中多条时按 动作权重 → 风险等级 → 词条长度 择优
              </p>
            </div>
            <button class="btn btn-sm" @click="addSensitiveRule('rules')">
              <i class="bi bi-plus-lg" /> 添加
            </button>
          </div>

          <p v-if="!cfg.sensitive.rules.length" class="empty">还没有规则，敏感词命中后将直接放行</p>

          <div v-else class="rule-list">
            <article
              v-for="(rule, index) in cfg.sensitive.rules"
              :key="`rules-${index}`"
              class="rule"
            >
              <div class="rule-top">
                <span class="rule-no">{{ index + 1 }}</span>
                <div class="rule-fields is-four">
                  <div class="form-item is-flat">
                    <label class="form-label">词条</label>
                    <FormInput v-model="rule.word" maxlength="64" placeholder="如：广告" />
                  </div>
                  <div class="form-item is-flat">
                    <label class="form-label">处置动作</label>
                    <FormSelect v-model="rule.action" :options="SENSITIVE_ACTION_OPTIONS" />
                  </div>
                  <div class="form-item is-flat">
                    <label class="form-label">风险等级</label>
                    <FormSelect v-model="rule.level" :options="SENSITIVE_LEVEL_OPTIONS" />
                  </div>
                  <div class="form-item is-flat">
                    <label class="form-label">匹配方式</label>
                    <FormSelect v-model="rule.match" :options="SENSITIVE_MATCH_OPTIONS" />
                  </div>
                </div>
                <button class="icon-btn" title="删除" @click="removeSensitiveRule('rules', index)">
                  <i class="bi bi-trash3" />
                </button>
              </div>

              <div class="rule-scenes">
                <span class="scenes-label">生效场景</span>
                <div class="scene-chips">
                  <button
                    v-for="scene in SENSITIVE_SCENES"
                    :key="scene.value"
                    type="button"
                    class="scene-chip"
                    :class="{ 'is-active': rule.scenes.includes(scene.value) }"
                    @click="toggleSensitiveScene(rule, scene.value)"
                  >
                    <i v-if="rule.scenes.includes(scene.value)" class="bi bi-check2" />
                    {{ scene.label }}
                  </button>
                  <span class="scenes-hint">不选 = 全部场景</span>
                </div>
              </div>
            </article>
          </div>

          <div class="card-foot">
            <button class="btn btn-primary btn-sm" :disabled="isSaving('sensitive')" @click="saveSensitive">
              <i class="bi bi-check2" /> 保存并生效
            </button>
            <button class="btn btn-sm" :disabled="isSaving('sensitive')" @click="rollbackSensitive">
              <i class="bi bi-arrow-counterclockwise" /> 回滚上一次
            </button>
            <span class="foot-hint">保存后版本自动 +1，旧版本留一份备份用于回滚</span>
          </div>
        </section>

        <!-- 规则测试 -->
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">规则测试</h3>
              <p class="card-desc">用当前词库试跑一段文本，不写命中审计</p>
            </div>
          </div>

          <div class="grid-2">
            <div class="form-item">
              <label class="form-label">场景</label>
              <FormSelect v-model="cfg.sensitiveTest.scene" :options="SENSITIVE_SCENES" />
            </div>
            <div class="form-item">
              <label class="form-label">测试文本</label>
              <FormInput
                v-model="cfg.sensitiveTest.text"
                placeholder="输入一段文本，检查会命中哪条规则"
                @keyup.enter="runSensitiveTest" />
            </div>
          </div>

          <div class="card-foot">
            <button class="btn btn-sm" :disabled="isSaving('sensitive')" @click="runSensitiveTest">
              <i class="bi bi-play-circle" /> 测试
            </button>
            <div v-if="cfg.sensitiveTest.result" class="test-result">
              <span class="pill" :class="resultTone(cfg.sensitiveTest.result.action)">
                {{
                  SENSITIVE_ACTION_LABEL[cfg.sensitiveTest.result.action] ||
                  cfg.sensitiveTest.result.action
                }}
              </span>
              <span v-if="cfg.sensitiveTest.result.word" class="result-meta">
                命中：<b>{{ cfg.sensitiveTest.result.word }}</b>
              </span>
              <span v-if="cfg.sensitiveTest.result.level" class="result-meta">
                等级：{{ cfg.sensitiveTest.result.level }}
              </span>
              <span v-if="cfg.sensitiveTest.result.version" class="result-meta">
                版本：v{{ cfg.sensitiveTest.result.version }}
              </span>
            </div>
          </div>
        </section>

        <!-- 命中统计 -->
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">命中统计</h3>
              <p class="card-desc">按机审维度与场景聚合；私信 / 群聊不保存原文</p>
            </div>
            <div class="head-filters">
              <FormSelect v-model="cfg.sensitiveDays" :options="SENSITIVE_STATS_DAYS" />
              <FormSelect v-model="cfg.sensitiveType" :options="cfg.sensitiveTypeOptions" />
            </div>
          </div>

          <!-- 维度分布：点一下按该维度筛选，再点一次取消 -->
          <div v-if="cfg.sensitiveDimensions.length" class="dim-bar">
            <button
              v-for="item in cfg.sensitiveDimensions"
              :key="`dim-${item.type}`"
              type="button"
              class="dim-chip"
              :class="{ 'is-active': Number(cfg.sensitiveType) === Number(item.type) }"
              @click="cfg.sensitiveType = Number(cfg.sensitiveType) === Number(item.type) ? 0 : Number(item.type)"
            >
              {{ item.label }}
              <b>{{ item.total }}</b>
            </button>
          </div>

          <p v-if="!cfg.sensitiveStats.length" class="empty">这段时间还没有命中记录</p>

          <ul v-else class="stat-list">
            <li v-for="item in cfg.sensitiveStats" :key="`${item.scene}-${item.action}`" class="stat-row">
              <span class="stat-scene">{{ sensitiveSceneLabel(item.scene) }}</span>
              <span class="stat-action">{{ SENSITIVE_ACTION_LABEL[item.action] || item.action }}</span>
              <span class="stat-total">{{ item.total }} 次</span>
            </li>
          </ul>
        </section>
      </template>

      <!-- ==================== 云端机审 ==================== -->
      <template v-else>
        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">云端审核开关</h3>
              <p class="card-desc">
                与敏感词库互补：本地同步快筛，云端异步兜底。凭据不全会自动失效，不会静默扣量
              </p>
            </div>
            <FormCheckbox variant="switch" v-model="toml.audit.enabled" />
          </div>

          <div class="grid-2">
            <div class="form-item">
              <label class="form-label">服务商</label>
              <FormSelect v-model="toml.audit.provider" :options="AUDIT_PROVIDER_OPTIONS" />
              <p class="row-hint">目前只接入了百度内容安全</p>
            </div>
            <div class="form-item">
              <label class="form-label">请求超时（秒）</label>
              <FormInput v-model.number="toml.audit.baidu.timeout" type="number" min="1" max="60" />
              <p class="row-hint">图片审核按此值 ×3（下限 10 秒）；超时按失败处理，不误伤内容状态</p>
            </div>
          </div>

          <div class="state-line" :class="auditActive ? 'is-on' : 'is-off'">
            <i :class="auditActive ? 'bi bi-check-circle-fill' : 'bi bi-exclamation-circle'" />
            <span>{{ auditStateText }}</span>
          </div>
        </section>

        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">凭据</h3>
              <p class="card-desc">AK/SK 只写在服务器 config/audit.toml，不落数据库、不下发前台</p>
            </div>
          </div>

          <div class="form-item">
            <label class="form-label">API Key</label>
            <FormInput
              v-model.trim="toml.audit.baidu.api_key"
              autocomplete="off"
              placeholder="百度智能云 → 内容安全 → 应用列表" />
          </div>

          <div class="form-item">
            <label class="form-label">Secret Key</label>
            <FormInput
              v-model.trim="toml.audit.baidu.secret_key"
              type="password"
              autocomplete="new-password"
              placeholder="只在服务器 config/audit.toml 里保存" />
          </div>
        </section>

        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">策略与并发</h3>
              <p class="card-desc">策略 ID 留空用百度默认策略；并发与队列要与账号配额匹配</p>
            </div>
          </div>

          <div class="grid-2">
            <div class="form-item">
              <label class="form-label">文本审核策略 ID</label>
              <FormInput v-model.trim="toml.audit.baidu.text_strategy_id" placeholder="留空使用默认策略" />
            </div>
            <div class="form-item">
              <label class="form-label">图像审核策略 ID</label>
              <FormInput v-model.trim="toml.audit.baidu.image_strategy_id" placeholder="留空使用默认策略" />
            </div>
            <div class="form-item">
              <label class="form-label">并发协程</label>
              <FormInput v-model.number="toml.audit.baidu.workers" type="number" min="1" max="32" />
              <p class="row-hint">与账号 QPS 配额匹配，建议 2~8</p>
            </div>
            <div class="form-item">
              <label class="form-label">待审队列长度</label>
              <FormInput v-model.number="toml.audit.baidu.queue_size" type="number" min="16" max="8192" />
              <p class="row-hint">队列满时丢弃本次并记日志，不会阻塞发布</p>
            </div>
          </div>
        </section>

        <section class="card card-pad">
          <div class="card-head">
            <div>
              <h3 class="card-title">试跑</h3>
              <p class="card-desc">真实调用一次文本审核，会消耗配额；用于确认凭据与策略是否可用</p>
            </div>
          </div>

          <div class="form-item">
            <label class="form-label">测试文本</label>
            <FormTextarea
              v-model="toml.auditTest.text"
              rows="3"
              placeholder="粘贴一段文本，例如正常的句子或已知的违规词" />
          </div>

          <div class="card-foot">
            <button class="btn btn-sm" :disabled="isSaving('audit')" @click="runAuditTest">
              <i class="bi bi-play-circle" /> 测试
            </button>
            <div v-if="toml.auditTest.result" class="test-result">
              <span class="pill" :class="auditTone(toml.auditTest.result.conclusion)">
                {{ AUDIT_CONCLUSION_LABEL[toml.auditTest.result.conclusion] || toml.auditTest.result.conclusion }}
              </span>
              <span
                v-if="toml.auditTest.result.messages && toml.auditTest.result.messages.length"
                class="result-meta"
              >
                {{ toml.auditTest.result.messages.join('；') }}
              </span>
              <span
                v-else-if="toml.auditTest.result.words && toml.auditTest.result.words.length"
                class="result-meta"
              >
                命中：{{ toml.auditTest.result.words.join('、') }}
              </span>
            </div>
          </div>

          <div class="card-foot is-plain">
            <button class="btn btn-primary btn-sm" :disabled="isSaving('audit')" @click="saveAudit">
              <i class="bi bi-check2" /> 保存并生效
            </button>
          </div>
        </section>
      </template>
    </template>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 安全中心 · 内容审核（/admin/security/content）
 *
 * 敏感词库与云端机审原来分散在「系统配置 → 互动配置」的两个标签页里，
 * 但它们本来就是同一条链路上的两道闸（本地同步快筛 + 云端异步兜底）：
 * - 敏感词库 = config/sensitive.toml，按场景分级处置，保存即热更新；
 * - 云端机审 = config/audit.toml，AK/SK 只落服务器，active 才是「真正生效」。
 * 合并到安全中心后，同一个页面就能看到两条链路的状态与命中情况。
 *
 * 后端接口（与系统配置页共用，权限点仍是 /admin/security 一级模块）：
 *   GET  /api/toml/sensitive           读词库（解析结果 + 原文）
 *   PUT  /api/toml/sensitive           保存（结构化 或 { content } 源码）
 *   POST /api/toml/sensitive-rollback  回滚上一次保存
 *   POST /api/toml/sensitive-test      规则试跑（不写命中审计）
 *   GET  /api/toml/sensitive-stats     命中统计（days / type 筛选）
 *   GET  /api/toml/audit               读机审配置（含 active 生效状态）
 *   PUT  /api/toml/audit               保存并热更新
 *   POST /api/toml/audit-test          真实调用一次文本审核（会计费）
 */
import { computed, reactive, ref, watch, onMounted } from 'vue'
import {
  getTomlSensitive,
  getTomlSensitiveStats,
  saveTomlSensitive,
  rollbackTomlSensitive,
  testTomlSensitive,
  getTomlAudit,
  saveTomlAudit,
  testTomlAudit
} from '@/api/toml'
import { toast } from '@/utils/toast'

// 两个分段的定义
const PANES = [
  { key: 'sensitive', label: '敏感词库', icon: 'bi bi-funnel' },
  { key: 'audit', label: '云端机审', icon: 'bi bi-cloud-check' }
]
const pane = ref('sensitive')

// 云端审核服务商（后端目前只支持百度）
const AUDIT_PROVIDER_OPTIONS = [{ value: 'baidu', label: '百度智能云 · 内容安全' }]

// 云端审核结论文案（与后端 facade.BaiduAudit* 一致）
const AUDIT_CONCLUSION_LABEL = {
  pass: '通过',
  review: '疑似违规（进人工复核）',
  block: '违规（已下架）',
  failed: '审核失败（未判定）'
}

// 命中统计的时间范围
const SENSITIVE_STATS_DAYS = [
  { value: 7, label: '近 7 天' },
  { value: 30, label: '近 30 天' },
  { value: 90, label: '近 90 天' }
]

// 敏感词场景（与后端 model.Scene* 一致）
const SENSITIVE_SCENES = [
  { value: 'comment', label: '评论' },
  { value: 'article', label: '文章' },
  { value: 'moment', label: '动态' },
  { value: 'dm', label: '私信' },
  { value: 'chat', label: '公共群聊' }
]

// 命中审计里的动作文案
const SENSITIVE_ACTION_LABEL = {
  block: '拒绝',
  limit: '限流拒绝',
  review: '进审核',
  warn: '提示放行',
  log: '仅记录',
  allow: '放行'
}

// 可视化表单的下拉选项（动作 / 等级 / 匹配方式）
const SENSITIVE_ACTION_OPTIONS = [
  { value: 'block', label: '拒绝（block）' },
  { value: 'limit', label: '限流拒绝（limit）' },
  { value: 'review', label: '进审核（review）' },
  { value: 'warn', label: '提示后放行（warn）' },
  { value: 'log', label: '仅记录（log）' }
]
const SENSITIVE_LEVEL_OPTIONS = [
  { value: 'high', label: '高（high）' },
  { value: 'medium', label: '中（medium）' },
  { value: 'low', label: '低（low）' }
]
const SENSITIVE_MATCH_OPTIONS = [
  { value: 'contains', label: '包含（contains）' },
  { value: 'exact', label: '整段相等（exact）' }
]

const loading = ref(false)
// 每个动作独立的 loading 标记，避免保存与测试互相阻塞
const saving = reactive({})

const cfg = reactive({
  // 词库：可视化表单 / TOML 源码 两种编辑模式
  sensitiveMode: 'form',
  sensitive: {
    enabled: true,
    default_action: 'review',
    version: '1',
    rules: [],
    whitelist: []
  },
  sensitiveContent: '',
  sensitiveTest: { scene: 'comment', text: '', result: null },
  sensitiveStats: [],
  // 命中统计的筛选条件（days 天数 / type 机审维度）与维度分布
  sensitiveDays: 7,
  sensitiveType: 0,
  sensitiveDimensions: [],
  sensitiveTypeOptions: []
})

// 云端机审（config/audit.toml）：enabled 是配置开关，active 才是「凭据齐全、真正生效」
const toml = reactive({
  audit: {
    enabled: false,
    active: false,
    provider: 'baidu',
    baidu: {
      api_key: '',
      secret_key: '',
      text_strategy_id: '',
      image_strategy_id: '',
      timeout: 3,
      workers: 3,
      queue_size: 256
    }
  },
  auditTest: { text: '', result: null }
})

const auditActive = computed(() => !!toml.audit.active)
const auditText = computed(() => {
  if (toml.audit.active) return '生效中'
  return toml.audit.enabled ? '凭据不全' : '已关闭'
})
const auditStateText = computed(() => {
  if (toml.audit.active) return '当前已生效：新内容会异步送审，命中违规自动下架'
  if (toml.audit.enabled) return '已开启但凭据不全（API Key / Secret Key），暂未生效'
  return '关闭中：仅本地敏感词库生效，云端不做兜底'
})

function isSaving(key) {
  return !!saving[key]
}

/** 统一的异步动作包装：失败提示由请求拦截器给出，这里只负责 loading 与 403 提示 */
async function withSaving(key, fn) {
  saving[key] = true
  try {
    await fn()
    return true
  } catch (err) {
    if (err?.code === 403) {
      toast.warning('当前账号没有这个权限点，请在「权限分组」中勾选后重试（新接口需重启后端补录规则）')
    }
    return false
  } finally {
    saving[key] = false
  }
}

function toText(value, fallback = '') {
  if (value === null || value === undefined) return fallback
  return String(value)
}

// ---------- 敏感词库 ----------

/** 用后端返回的解析结果同时填充源码原文与可视化表单 */
function applySensitivePayload(data) {
  if (!data) return

  cfg.sensitiveContent = String(data.content || '')
  cfg.sensitive.enabled = Number(data.enabled) === 1
  cfg.sensitive.default_action = String(data.default_action || 'review')
  cfg.sensitive.version = String(data.version || '1')
  cfg.sensitive.rules = normalizeSensitiveRules(data.rules)
  cfg.sensitive.whitelist = normalizeSensitiveRules(data.whitelist)
}

/** 规则列表归一化：补全缺省值，避免表单控件拿到 undefined */
function normalizeSensitiveRules(rows) {
  if (!Array.isArray(rows)) return []

  return rows.map((row) => ({
    word: String(row?.word || ''),
    action: String(row?.action || 'review'),
    scenes: Array.isArray(row?.scenes) ? row.scenes.map(String) : [],
    level: String(row?.level || 'medium'),
    match: String(row?.match || 'contains')
  }))
}

/** 提交前裁剪：只留后端认识的字段 */
function pickSensitiveRule(rule) {
  return {
    word: String(rule.word || '').trim(),
    action: rule.action,
    scenes: rule.scenes,
    level: rule.level,
    match: rule.match
  }
}

/** 新增一条规则（kind：rules / whitelist） */
function addSensitiveRule(kind) {
  cfg.sensitive[kind].push({
    word: '',
    action: 'review',
    scenes: [],
    level: 'medium',
    match: 'contains'
  })
}

/** 删除一条规则 */
function removeSensitiveRule(kind, index) {
  cfg.sensitive[kind].splice(index, 1)
}

/** 场景多选：点一下切换选中态（不选 = 全部场景） */
function toggleSensitiveScene(rule, scene) {
  const index = rule.scenes.indexOf(scene)
  if (index >= 0) {
    rule.scenes.splice(index, 1)
    return
  }
  rule.scenes.push(scene)
}

/** 保存词库：源码模式提交原文，可视化模式提交结构化数据 */
async function saveSensitive() {
  if (cfg.sensitiveMode === 'source') {
    if (!String(cfg.sensitiveContent || '').trim()) {
      toast.warning('规则内容不能为空')
      return
    }

    await withSaving('sensitive', async () => {
      const res = await saveTomlSensitive({ content: cfg.sensitiveContent })
      applySensitivePayload(res?.data)
      toast.success('已保存并生效')
      await refreshSensitiveStats()
    })
    return
  }

  // 表单模式：词条必填（后端也会拦，这里先给即时反馈）
  for (const group of [cfg.sensitive.whitelist, cfg.sensitive.rules]) {
    if (group.some((rule) => !String(rule.word || '').trim())) {
      toast.warning('有规则还没填写词条')
      return
    }
  }

  await withSaving('sensitive', async () => {
    const res = await saveTomlSensitive({
      enabled: cfg.sensitive.enabled,
      default_action: cfg.sensitive.default_action,
      rules: cfg.sensitive.rules.map(pickSensitiveRule),
      whitelist: cfg.sensitive.whitelist.map(pickSensitiveRule)
    })
    applySensitivePayload(res?.data)
    toast.success('已保存并生效')
    await refreshSensitiveStats()
  })
}

/** 回滚到上一次保存前的词库 */
async function rollbackSensitive() {
  // 危险操作：会直接覆盖当前规则，先确认
  if (!window.confirm('确定回滚到上一次保存前的词库吗？当前编辑的内容会被覆盖。')) return

  await withSaving('sensitive', async () => {
    const res = await rollbackTomlSensitive()
    applySensitivePayload(res?.data)
    toast.success('已回滚')
  })
}

/** 命中统计：应用后端返回（场景聚合 + 维度分布 + 筛选项） */
function applySensitiveStats(data) {
  cfg.sensitiveStats = Array.isArray(data?.items) ? data.items : []
  cfg.sensitiveDimensions = Array.isArray(data?.dimensions) ? data.dimensions : []

  const options = Array.isArray(data?.options) ? data.options : []
  cfg.sensitiveTypeOptions = [
    { value: 0, label: '全部维度' },
    ...options
      .filter((item) => Number(item.type) > 0)
      .map((item) => ({ value: Number(item.type), label: item.label }))
  ]
}

/** 按当前筛选（天数 / 维度）重新拉统计 */
async function refreshSensitiveStats() {
  const stats = await getTomlSensitiveStats({
    days: cfg.sensitiveDays,
    type: cfg.sensitiveType
  }).catch(() => null)
  applySensitiveStats(stats?.data)
}

// 筛选条件一变就重拉（首次加载由 loadAll 负责，值未变不会触发）
watch(
  () => [cfg.sensitiveDays, cfg.sensitiveType],
  () => {
    refreshSensitiveStats()
  }
)

/** 用当前词库试跑一条文本 */
async function runSensitiveTest() {
  if (!String(cfg.sensitiveTest.text || '').trim()) {
    toast.warning('请输入测试文本')
    return
  }

  await withSaving('sensitive', async () => {
    const res = await testTomlSensitive({
      scene: cfg.sensitiveTest.scene,
      text: cfg.sensitiveTest.text
    })
    cfg.sensitiveTest.result = res?.data?.decision || null
  })
}

/** 场景文案 */
function sensitiveSceneLabel(scene) {
  const found = SENSITIVE_SCENES.find((item) => item.value === scene)
  return found?.label || scene
}

/** 动作 → 结果标签配色 */
function resultTone(action) {
  if (action === 'block' || action === 'limit') return 'is-danger'
  if (action === 'review') return 'is-warn'
  if (action === 'warn') return 'is-warn'
  return 'is-on'
}

// ---------- 云端机审 ----------

/** 用后端返回的配置填充表单 */
function applyAuditPayload(data) {
  if (!data) return

  toml.audit.enabled = !!data.enabled
  toml.audit.active = !!data.active
  toml.audit.provider = toText(data.provider, 'baidu')

  const baidu = data.baidu || {}
  toml.audit.baidu = {
    api_key: toText(baidu.api_key, ''),
    secret_key: toText(baidu.secret_key, ''),
    text_strategy_id: toText(baidu.text_strategy_id, ''),
    image_strategy_id: toText(baidu.image_strategy_id, ''),
    timeout: Number(baidu.timeout || 3),
    workers: Number(baidu.workers || 3),
    queue_size: Number(baidu.queue_size || 256)
  }
}

/** 保存机审配置：成功后重新拉一次，让「是否真正生效」的状态跟着更新 */
async function saveAudit() {
  const baidu = toml.audit.baidu
  if (toml.audit.enabled && (!String(baidu.api_key || '').trim() || !String(baidu.secret_key || '').trim())) {
    toast.warning('开启云端审核前请先填写 API Key 与 Secret Key')
    return
  }

  await withSaving('audit', async () => {
    await saveTomlAudit({
      enabled: toml.audit.enabled,
      provider: toml.audit.provider,
      api_key: baidu.api_key,
      secret_key: baidu.secret_key,
      text_strategy_id: baidu.text_strategy_id,
      image_strategy_id: baidu.image_strategy_id,
      timeout: baidu.timeout,
      workers: baidu.workers,
      queue_size: baidu.queue_size
    })
    const fresh = await getTomlAudit().catch(() => null)
    applyAuditPayload(fresh?.data)
    toast.success('已保存并生效')
  })
}

/** 试跑一条文本（真实调用，会计费） */
async function runAuditTest() {
  if (!String(toml.auditTest.text || '').trim()) {
    toast.warning('请输入测试文本')
    return
  }

  await withSaving('audit', async () => {
    const res = await testTomlAudit({ text: toml.auditTest.text })
    toml.auditTest.result = res?.data || null
    toast.success('测试完成')
  })
}

/** 结论 → 结果标签配色 */
function auditTone(conclusion) {
  if (conclusion === 'block') return 'is-danger'
  if (conclusion === 'review') return 'is-warn'
  if (conclusion === 'failed') return 'is-quiet'
  return 'is-on'
}

// ---------- 加载 ----------

async function loadAll() {
  loading.value = true
  try {
    const [sensitive, stats, audit] = await Promise.all([
      getTomlSensitive().catch(() => null),
      getTomlSensitiveStats().catch(() => null),
      getTomlAudit().catch(() => null)
    ])
    applySensitivePayload(sensitive?.data)
    applySensitiveStats(stats?.data)
    applyAuditPayload(audit?.data)
  } finally {
    loading.value = false
  }
}

onMounted(loadAll)
</script>

<style scoped>
.sec-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览条 ---------- */
.hero {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  background:
    radial-gradient(120% 140% at 0% 0%, var(--accent-wash), transparent 60%),
    var(--bg-card);
}
.hero-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.hero-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius);
}
.hero-badge .bi {
  font-size: 20px;
}
.hero-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.02em;
}
.hero-desc {
  margin: 4px 0 0;
  max-width: 62ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.hero-side {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

/* ---------- 状态标签 ---------- */
.pill {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 3px 10px;
  font-size: 12px;
  line-height: 1.6;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--text-soft);
  background: var(--bg-muted);
}
.pill .bi {
  font-size: 16px;
  line-height: 1;
}
.pill.is-on {
  color: var(--success);
  border-color: color-mix(in srgb, var(--success) 30%, transparent);
  background: color-mix(in srgb, var(--success) 8%, transparent);
}
.pill.is-off {
  color: var(--text-muted);
}
.pill.is-warn {
  color: var(--warning);
  border-color: color-mix(in srgb, var(--warning) 32%, transparent);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
}
.pill.is-danger {
  color: var(--danger);
  border-color: color-mix(in srgb, var(--danger) 30%, transparent);
  background: color-mix(in srgb, var(--danger) 8%, transparent);
}
.pill.is-quiet {
  color: var(--text-muted);
}

/* ---------- 分段切换 ---------- */
.seg {
  display: inline-flex;
  align-self: flex-start;
  gap: 4px;
  padding: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.seg-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  font-size: 13px;
  color: var(--text-soft);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background-color var(--dur-fast, 0.15s), color var(--dur-fast, 0.15s);
}
.seg-item:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.seg-item.is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
  font-weight: 600;
}
.seg-count {
  padding: 0 6px;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 999px;
}
.seg-sm .seg-item {
  padding: 5px 12px;
  font-size: 12px;
}

/* ---------- 卡片内部的标题行 ---------- */
.card-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 14px;
}
.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.count {
  padding: 0 7px;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 999px;
}
.card-desc {
  margin: 4px 0 0;
  max-width: 70ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.head-filters {
  display: flex;
  gap: 8px;
}

/* ---------- 开关行 ---------- */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  margin-bottom: 14px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.row-title {
  font-size: 13px;
  font-weight: 600;
}
.row-hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

/* 开关（与站点其它页面同一套视觉） */
.switch {
  position: relative;
  display: inline-block;
  width: 42px;
  height: 24px;
  flex-shrink: 0;
}
.switch input {
  width: 0;
  height: 0;
  opacity: 0;
}
.switch-slider {
  position: absolute;
  inset: 0;
  cursor: pointer;
  background: var(--border);
  border-radius: 999px;
  transition: background-color var(--dur-fast, 0.15s);
}
.switch-slider::before {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  background: #fff;
  border-radius: 50%;
  box-shadow: var(--shadow-sm);
  transition: transform var(--dur-fast, 0.15s);
}
.switch input:checked + .switch-slider {
  background: var(--primary);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}

/* ---------- 栅格 ---------- */
.grid-2 {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}

/* ---------- 规则卡片 ---------- */
.rule-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.rule {
  padding: 12px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft, var(--border));
  border-radius: var(--radius);
  transition: border-color var(--dur-fast, 0.15s), box-shadow var(--dur-fast, 0.15s);
}
.rule:hover {
  border-color: var(--accent-ring, var(--border));
  box-shadow: var(--shadow-sm);
}
.rule-top {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}
.rule-no {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  flex-shrink: 0;
  margin-bottom: 8px;
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 50%;
}
.rule-fields {
  display: grid;
  flex: 1;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 12px;
  min-width: 0;
}
.rule-fields.is-four {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
.form-item.is-flat {
  margin-bottom: 0;
}
.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  margin-bottom: 4px;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--dur-fast, 0.15s), border-color var(--dur-fast, 0.15s);
}
.icon-btn:hover {
  color: var(--danger);
  border-color: var(--danger);
}

.rule-scenes {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px dashed var(--border-soft, var(--border));
}
.scenes-label {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
}
.scene-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.scene-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px 11px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  transition: all var(--dur-fast, 0.15s);
}
.scene-chip:hover {
  color: var(--primary);
  border-color: var(--accent-ring, var(--border));
}
.scene-chip.is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-color: var(--primary-soft, var(--primary));
  font-weight: 500;
}
.scenes-hint {
  font-size: 11px;
  color: var(--text-light, var(--text-muted));
}

/* ---------- 页脚操作条 ---------- */
.card-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft, var(--border));
}
.card-foot.is-plain {
  margin-top: 12px;
  padding-top: 0;
  border-top: none;
  justify-content: flex-end;
}
.foot-hint {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-light, var(--text-muted));
}

/* ---------- 测试结果 ---------- */
.test-result {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.result-meta {
  font-size: 12px;
  color: var(--text-soft);
}

/* ---------- 命中统计 ---------- */
.dim-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.dim-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  font-size: 12px;
  color: var(--text-soft);
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 999px;
  cursor: pointer;
  transition: all var(--dur-fast, 0.15s);
}
.dim-chip b {
  color: var(--primary-deep);
  font-weight: 600;
}
.dim-chip:hover {
  border-color: var(--primary);
  color: var(--primary);
}
.dim-chip.is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-color: var(--primary-soft, var(--primary));
}
.stat-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}
.stat-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 2px;
  font-size: 13px;
  border-bottom: 1px dashed var(--border-soft, var(--border));
}
.stat-row:last-child {
  border-bottom: none;
}
.stat-scene {
  min-width: 88px;
  color: var(--text);
}
.stat-action {
  flex: 1;
  color: var(--text-muted);
}
.stat-total {
  font-variant-numeric: tabular-nums;
  color: var(--primary-deep);
  font-weight: 600;
}

/* ---------- 机审生效状态 ---------- */
.state-line {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding: 10px 14px;
  font-size: 12px;
  border-radius: var(--radius);
}
.state-line.is-on {
  color: var(--success);
  background: color-mix(in srgb, var(--success) 8%, transparent);
}
.state-line.is-off {
  color: var(--warning);
  background: color-mix(in srgb, var(--warning) 10%, transparent);
}

/* ---------- 杂项 ---------- */
.empty {
  margin: 0;
  padding: 20px 0;
  font-size: 12px;
  text-align: center;
  color: var(--text-light, var(--text-muted));
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.code {
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  line-height: 1.7;
}
.tip-line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.spinner {
  width: 13px;
  height: 13px;
  border: 2px solid var(--border);
  border-top-color: var(--primary);
  border-radius: 50%;
  animation: sec-spin 0.7s linear infinite;
}
@keyframes sec-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .grid-2,
  .rule-fields,
  .rule-fields.is-four {
    grid-template-columns: minmax(0, 1fr);
  }
  .rule-top {
    align-items: flex-start;
  }
  .rule-scenes {
    align-items: flex-start;
    flex-direction: column;
  }
  .foot-hint {
    margin-left: 0;
  }
}
</style>
