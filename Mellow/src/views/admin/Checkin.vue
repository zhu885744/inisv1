<template>
  <div class="checkin-admin">
    <!-- 概览 / 操作 -->
    <section class="card card-pad">
      <header class="head">
        <div>
          <h2 class="block-title">签到管理</h2>
          <p class="block-desc">
            签到奖励不再只有经验和积分：基础奖励、周期奖励、连签加成、里程碑、月度全勤、随机奖励
            都能单独配置；奖励类型由后端「奖励资产表」提供（内置经验 / 积分 / <strong>卡密</strong>，可扩展）。
            想「连签满 N 天送一张卡密」，在对应奖励里把类型选成「卡密」、数值填面额，
            并在行下方写上要发的那几张卡密（一行一个）。
          </p>
        </div>
        <div class="head-actions">
          <router-link to="/checkin" target="_blank" class="btn btn-sm">
            <i class="bi bi-box-arrow-up-right" /> 前台效果
          </router-link>
          <button class="btn btn-sm" type="button" :disabled="loading" title="重新读取" @click="load">
            <i class="bi bi-arrow-clockwise" />
          </button>
          <button class="btn btn-primary btn-sm" type="button" :disabled="saving || loading" @click="save">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </header>
      <p class="hint">
        配置键 <code>SYSTEM_CHECKIN_RULES</code>（独立于经验 / 积分规则）；
        保存后立即生效，已签到的用户当天不会重复发放。
      </p>
      <p class="hint">
        <i class="bi bi-ticket-perforated" /> 卡密奖励发的是<strong>你在奖励项里填写的卡密</strong>（纯卡密，
        与积分、与「积分 → 卡密」的池子都无关）；每发一次消耗一张，<strong>库存发完即失效</strong>，
        不会改发别的奖励。
      </p>
    </section>

    <!-- 基础设置 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-sliders" /> 基础设置</h3>
      <div class="grid-2">
        <div class="form-item">
          <label class="form-label">签到开关</label>
          <FormCheckbox variant="switch" v-model.number="form.enabled" :true-value="1" :false-value="0" :label="form.enabled ? '已开启' : '已关闭（前台不再展示签到）'" />
        </div>
        <div class="form-item">
          <label class="form-label">名称</label>
          <FormInput v-model="form.name" type="text" maxlength="16" placeholder="每日签到" />
        </div>
        <div class="form-item">
          <label class="form-label">每日重置时间</label>
          <FormSelect v-model="form.reset_hour" :options="RESET_HOURS" />
          <p class="form-hint">选择 4 点表示凌晨 4 点之前仍算前一天，方便熬夜用户。</p>
        </div>
        <div class="form-item">
          <label class="form-label">签到成功通知</label>
          <FormCheckbox variant="switch" v-model.number="form.notice" :true-value="1" :false-value="0" :label="form.notice ? '发送站内信' : '不发送'" />
        </div>
      </div>
    </section>

    <!-- 基础奖励 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-gift" /> 基础奖励</h3>
      <p class="block-desc">每次签到都会发放（补签也走这里，除非补签单独配置了奖励）。</p>
      <RewardItemsEditor v-model="form.base" :assets="assets" hint="例如：经验 10 + 积分 5" />
    </section>

    <!-- 连续签到加成 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-fire" /> 连续签到加成</h3>
      <p class="block-desc">
        <strong>每天都发</strong>：连续第 N 天的额外奖励 = min(N × 每连续一天, 加成上限)，
        连签越久拿得越多、到上限封顶。适合「日常正向激励」。
      </p>
      <div class="grid-2">
        <div class="form-item">
          <label class="form-label">启用</label>
          <FormCheckbox variant="switch" v-model.number="form.streak.enabled" :true-value="1" :false-value="0" :label="form.streak.enabled ? '已启用' : '未启用'" />
        </div>
        <div class="form-item">
          <label class="form-label">奖励类型</label>
          <FormSelect v-model="form.streak.asset" :options="assetOptions" />
        </div>
        <div class="form-item">
          <label class="form-label">每连续一天额外奖励</label>
          <FormInput v-model.number="form.streak.per_day" type="number" min="0" step="1" />
        </div>
        <div class="form-item">
          <label class="form-label">加成上限</label>
          <FormInput v-model.number="form.streak.max" type="number" min="0" step="1" placeholder="0 = 不封顶" />
        </div>
      </div>
      <p class="hint">连续第 N 天额外奖励 = min(N × 每连续一天, 上限)。</p>
    </section>

    <!-- 周期奖励 -->
    <section class="card card-pad">
      <header class="block-head">
        <h3 class="block-title"><i class="bi bi-calendar3-week" /> 周期奖励</h3>
        <FormCheckbox variant="switch" v-model.number="form.cycle.enabled" :true-value="1" :false-value="0" :label="form.cycle.enabled ? '已启用' : '未启用'" />
      </header>
      <p class="block-desc">连签第 N 天额外发放，例如常见的「7 天一轮，第 7 天有大奖」。</p>

      <template v-if="form.cycle.enabled">
        <div class="form-item">
          <FormCheckbox variant="switch" v-model.number="form.cycle.loop" :true-value="1" :false-value="0" :label="form.cycle.loop ? '循环（第 8 天重新从第 1 天算）' : '不循环（超过天数按最后一天算）'" />
        </div>

        <div v-for="(day, index) in form.cycle.days" :key="index" class="sub-card">
          <div class="sub-card-head">
            <span class="sub-card-title">第 {{ index + 1 }} 天</span>
            <FormInput v-model="day.label" class="sub-card-label" type="text" maxlength="16" placeholder="标签（可选）" />
            <button class="btn btn-ghost btn-sm" type="button" title="删除该天" @click="removeDay(index)">
              <i class="bi bi-trash" />
            </button>
          </div>
          <RewardItemsEditor v-model="day.rewards" :assets="assets" hint="留空表示这天只有基础奖励" />
        </div>

        <button class="btn btn-sm" type="button" @click="addDay">
          <i class="bi bi-plus-lg" /> 添加一天
        </button>
      </template>
    </section>

    <!-- 里程碑 -->
    <section class="card card-pad">
      <header class="block-head">
        <h3 class="block-title"><i class="bi bi-trophy" /> 连续签到里程碑</h3>
        <button class="btn btn-sm" type="button" @click="addLadder('milestones')">
          <i class="bi bi-plus-lg" /> 添加里程碑
        </button>
      </header>
      <p class="block-desc">
        <strong>只在达成的那一天发一次</strong>：连续天数达到第 7 / 15 / 30 天时一次性发放，
        适合「阶段性大奖」。与上面的连签加成<strong>不冲突</strong>（一个是天天递增，一个是一次性），
        两者可以同时开，也可以只用其中一个；奖励类型可选「卡密」，连签满多少天送一张卡密就配在这里。
      </p>

      <div v-for="(item, index) in form.milestones" :key="index" class="sub-card">
        <div class="sub-card-head">
          <span class="sub-card-title">连续</span>
          <FormInput v-model.number="item.day" class="sub-card-day" type="number" min="1" step="1" />
          <span class="sub-card-title">天</span>
          <FormInput v-model="item.label" class="sub-card-label" type="text" maxlength="16" placeholder="标签（可选）" />
          <button class="btn btn-ghost btn-sm" type="button" title="删除" @click="removeLadder('milestones', index)">
            <i class="bi bi-trash" />
          </button>
        </div>
        <RewardItemsEditor v-model="item.rewards" :assets="assets" />
      </div>

      <p v-if="!form.milestones.length" class="hint">暂无里程碑</p>
    </section>

    <!-- 月度奖励 -->
    <section class="card card-pad">
      <header class="block-head">
        <h3 class="block-title"><i class="bi bi-calendar-check" /> 月度奖励（全勤）</h3>
        <button class="btn btn-sm" type="button" @click="addLadder('monthly')">
          <i class="bi bi-plus-lg" /> 添加月度档位
        </button>
      </header>
      <p class="block-desc">
        当月签到天数达到档位时一次性发放，例如当月累计 20 天、28 天（全勤）；
        奖励类型选「卡密」即可发卡密（默认配置的第 28 天就带了一张全勤卡密示例）。
        判定按「跨越」口径：如果补签让当月天数直接跨过某个档位，也会照常发放，不会漏。
      </p>

      <div v-for="(item, index) in form.monthly" :key="index" class="sub-card">
        <div class="sub-card-head">
          <span class="sub-card-title">当月签到满</span>
          <FormInput v-model.number="item.day" class="sub-card-day" type="number" min="1" step="1" />
          <span class="sub-card-title">天</span>
          <FormInput v-model="item.label" class="sub-card-label" type="text" maxlength="16" placeholder="标签（可选）" />
          <button class="btn btn-ghost btn-sm" type="button" title="删除" @click="removeLadder('monthly', index)">
            <i class="bi bi-trash" />
          </button>
        </div>
        <RewardItemsEditor v-model="item.rewards" :assets="assets" />
      </div>

      <p v-if="!form.monthly.length" class="hint">暂无月度档位</p>
    </section>

    <!-- 随机奖励 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-dice-5" /> 随机奖励</h3>
      <p class="block-desc">
        按概率额外发放，例如「10% 概率获得 20 积分」。概率留空或填 100 表示必得。
      </p>
      <RewardItemsEditor v-model="form.random" :assets="assets" chance hint="配置还支持随机区间（min / max），可在配置里手工填写" />
    </section>

    <!-- 补签 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-calendar-plus" /> 补签</h3>
      <div class="grid-2">
        <div class="form-item">
          <label class="form-label">启用</label>
          <FormCheckbox variant="switch" v-model.number="form.makeup.enabled" :true-value="1" :false-value="0" :label="form.makeup.enabled ? '已启用' : '未启用'" />
        </div>
        <div class="form-item">
          <label class="form-label">可补签天数</label>
          <FormInput v-model.number="form.makeup.days" type="number" min="1" step="1" placeholder="7" />
          <p class="form-hint">只能补最近 N 天内、且没有签到过的日期</p>
        </div>
        <div class="form-item">
          <label class="form-label">每月次数上限</label>
          <FormInput v-model.number="form.makeup.limit" type="number" min="0" step="1" placeholder="0 = 不限制" />
        </div>
        <div class="form-item">
          <label class="form-label">消耗类型</label>
          <FormSelect v-model="form.makeup.asset" :options="assetOptions" />
        </div>
        <div class="form-item">
          <label class="form-label">每次消耗</label>
          <FormInput v-model.number="form.makeup.cost" type="number" min="0" step="1" placeholder="20" />
        </div>
      </div>
      <p class="hint">
        补签发放「补签奖励」（默认基础奖励），不补发周期奖励与连签加成；
        但如果补签把断掉的连签重新接起来、从而跨越了某个里程碑或月度档位，
        这些一次性奖励会照常补发（按跨越口径，只发一次不会重复）；补签后连签天数会自动重算。
      </p>
    </section>

    <!-- 文案 -->
    <section class="card card-pad">
      <h3 class="block-title"><i class="bi bi-chat-quote" /> 签到文案</h3>
      <p class="block-desc">签到成功后随机展示一条，每行一条。</p>
      <FormTextarea v-model="form.tips" rows="5" placeholder="签到成功，今天也要元气满满～" />
    </section>

    <div class="foot-actions">
      <button class="btn btn-primary" type="button" :disabled="saving || loading" @click="save">
        {{ saving ? '保存中...' : '保存配置' }}
      </button>
    </div>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 签到管理（/admin/checkin）
 *
 * 配置存 config 表的 SYSTEM_CHECKIN_RULES 键，结构与后端 model/checkin.go 的
 * defaultCheckinConfig() 一一对应：
 *
 *   enabled / name / reset_hour / notice / tips
 *   base      基础奖励   [{ asset, value, label }]
 *   streak    连签加成   { enabled, asset, per_day, max }
 *   cycle     周期奖励   { enabled, loop, days: [{ label, rewards: [...] }] }
 *   milestones 里程碑     [{ day, label, rewards: [...] }]
 *   monthly   月度奖励   [{ day, label, rewards: [...] }]
 *   random    随机奖励   [{ asset, value, chance, label }]
 *   makeup    补签       { enabled, days, limit, asset, cost }
 *
 * 奖励项里的 asset 由后端「奖励资产表」决定（内置 exp / integral，可扩展注册），
 * 前端不写死类型，资产清单从 /api/checkin/rules 拉取。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import RewardItemsEditor from '@/components/admin/RewardItemsEditor.vue'
import { getCheckinConfig, saveCheckinConfig, getCheckinRules } from '@/api/checkin'
// 卡密奖励不再依赖「积分 → 卡密」的池子（卡密内容在奖励项里自己填），因此这里不再拉卡密统计
import { toast } from '@/utils/toast'

const RESET_HOURS = [
  { value: 0, label: '00:00（自然日）' },
  { value: 1, label: '01:00' },
  { value: 2, label: '02:00' },
  { value: 3, label: '03:00' },
  { value: 4, label: '04:00（推荐，熬夜党友好）' },
  { value: 5, label: '05:00' },
  { value: 6, label: '06:00' }
]

const loading = ref(false)
const saving = ref(false)
const assets = ref([])

// 资产清单 → SelectMenu 选项（{key,name} → {value,label}）
const assetOptions = computed(() =>
  assets.value.map((asset) => ({ value: asset.key, label: asset.name }))
)

const form = reactive({
  enabled: 1,
  name: '每日签到',
  reset_hour: 0,
  notice: 0,
  tips: '',
  base: [],
  streak: { enabled: 1, asset: 'exp', per_day: 2, max: 50 },
  cycle: { enabled: 1, loop: 1, days: [] },
  milestones: [],
  monthly: [],
  random: [],
  makeup: { enabled: 1, days: 7, limit: 3, asset: 'integral', cost: 20 }
})

// 奖励项 → 表单结构（去掉后端的 reached / days_left 等展示字段）
function toItem(item = {}) {
  const asset = item.asset || 'exp'
  return {
    asset,
    // 卡密是纯卡密、没有面额，数值固定为 1（奖励引擎要求非 0，且它不参与卡密逻辑）
    value: asset === 'card' ? 1 : Number(item.value || 0),
    label: item.label || '',
    chance: Number(item.chance || 0),
    // 卡密内容在表单里按「一行一个」编辑（配置里存的是数组）
    codesText: Array.isArray(item.codes) ? item.codes.join('\n') : String(item.codes || '')
  }
}

// 表单结构 → 配置结构（去掉「概率 0」「空备注」这类噪音键）
function cleanItem(item) {
  const result = { asset: item.asset || 'exp', value: Number(item.value || 0) }
  if (item.label) result.label = item.label
  if (Number(item.chance) > 0) result.chance = Number(item.chance)

  if (item.asset === 'card') {
    // 卡密是纯卡密：数值固定 1（占位，不代表面额）
    result.value = 1
    delete result.chance

    // 卡密内容（一行一个）就是库存清单，留空则这份奖励不发卡密
    const codes = String(item.codesText || '')
      .split(/[\n\r,，;；]/)
      .map((code) => code.trim())
      .filter(Boolean)
    if (codes.length) result.codes = codes
  }

  return result
}

function normalize(data = {}) {
  const cycle = data.cycle || {}
  const streak = data.streak || {}
  const makeup = data.makeup || {}

  return {
    enabled: Number(data.enabled ?? 1) ? 1 : 0,
    name: data.name || '每日签到',
    reset_hour: Number(data.reset_hour || 0),
    notice: Number(data.notice || 0) ? 1 : 0,
    tips: Array.isArray(data.tips) ? data.tips.join('\n') : '',
    base: (data.base || []).map(toItem),
    streak: {
      enabled: Number(streak.enabled ?? 1) ? 1 : 0,
      asset: streak.asset || 'exp',
      per_day: Number(streak.per_day || 0),
      max: Number(streak.max || 0)
    },
    cycle: {
      enabled: Number(cycle.enabled ?? 1) ? 1 : 0,
      loop: Number(cycle.loop ?? 1) ? 1 : 0,
      days: (cycle.days || []).map((day) => ({
        label: day.label || '',
        rewards: (day.rewards || []).map(toItem)
      }))
    },
    milestones: (data.milestones || []).map((item) => ({
      day: Number(item.day || 0),
      label: item.label || '',
      rewards: (item.rewards || []).map(toItem)
    })),
    monthly: (data.monthly || []).map((item) => ({
      day: Number(item.day || 0),
      label: item.label || '',
      rewards: (item.rewards || []).map(toItem)
    })),
    random: (data.random || []).map(toItem),
    makeup: {
      enabled: Number(makeup.enabled ?? 1) ? 1 : 0,
      days: Number(makeup.days || 0),
      limit: Number(makeup.limit || 0),
      asset: makeup.asset || 'integral',
      cost: Number(makeup.cost || 0)
    }
  }
}

function serialize() {
  return {
    enabled: Number(form.enabled) ? 1 : 0,
    name: form.name.trim() || '每日签到',
    reset_hour: Number(form.reset_hour || 0),
    notice: Number(form.notice) ? 1 : 0,
    tips: form.tips.split('\n').map((item) => item.trim()).filter(Boolean),
    base: form.base.map(cleanItem).filter((item) => item.value > 0),
    streak: {
      enabled: Number(form.streak.enabled) ? 1 : 0,
      asset: form.streak.asset,
      per_day: Number(form.streak.per_day || 0),
      max: Number(form.streak.max || 0)
    },
    cycle: {
      enabled: Number(form.cycle.enabled) ? 1 : 0,
      loop: Number(form.cycle.loop) ? 1 : 0,
      days: form.cycle.days.map((day) => ({
        label: day.label || '',
        rewards: day.rewards.map(cleanItem).filter((item) => item.value > 0)
      }))
    },
    milestones: form.milestones
      .filter((item) => Number(item.day) > 0)
      .map((item) => ({
        day: Number(item.day),
        label: item.label || '',
        rewards: item.rewards.map(cleanItem).filter((reward) => reward.value > 0)
      })),
    monthly: form.monthly
      .filter((item) => Number(item.day) > 0)
      .map((item) => ({
        day: Number(item.day),
        label: item.label || '',
        rewards: item.rewards.map(cleanItem).filter((reward) => reward.value > 0)
      })),
    random: form.random.map(cleanItem).filter((item) => item.value > 0),
    makeup: {
      enabled: Number(form.makeup.enabled) ? 1 : 0,
      days: Number(form.makeup.days || 0),
      limit: Number(form.makeup.limit || 0),
      asset: form.makeup.asset,
      cost: Number(form.makeup.cost || 0)
    }
  }
}

async function load() {
  loading.value = true
  try {
    const [configRes, rulesRes] = await Promise.all([getCheckinConfig(), getCheckinRules()])

    const rules = rulesRes?.data || {}
    assets.value = rules.assets || []

    // 已保存过就以 config 为准；老站点没这条配置时用规则接口（后端默认值）兜底
    const saved = configRes?.data?.json || null
    Object.assign(form, normalize(saved || rules))
  } catch {
    // 错误提示由拦截器统一给出
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!form.name.trim()) {
    toast.info('请填写签到名称')
    return
  }
  if (form.cycle.enabled && !form.cycle.days.length) {
    toast.info('已启用周期奖励，请至少添加一天')
    return
  }

  saving.value = true
  try {
    const res = await saveCheckinConfig(serialize())
    if (res.code === 200) {
      toast.success('签到配置已保存')
      load()
    } else {
      toast.info(res.msg || '保存失败')
    }
  } catch {
    // 错误提示由拦截器统一给出
  } finally {
    saving.value = false
  }
}

function addDay() {
  form.cycle.days.push({ label: '', rewards: [] })
}

function removeDay(index) {
  form.cycle.days.splice(index, 1)
}

function addLadder(key) {
  form[key].push({ day: 0, label: '', rewards: [] })
}

function removeLadder(key, index) {
  form[key].splice(index, 1)
}

onMounted(load)
</script>

<style scoped>
.checkin-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}
.block-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 6px;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 700;
}
.block-desc {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.7;
}
.hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.8;
}
.hint code {
  padding: 0 4px;
  background: var(--bg-muted);
  border-radius: 3px;
}
.grid-2 {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 0 16px;
}
.form-item {
  margin-bottom: 14px;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
.sub-card {
  padding: 12px;
  margin-bottom: 12px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.sub-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.sub-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  white-space: nowrap;
}
.sub-card-day {
  width: 90px;
  flex-shrink: 0;
}
.sub-card-label {
  flex: 1;
  min-width: 0;
}
.foot-actions {
  display: flex;
  justify-content: flex-end;
  padding-bottom: 8px;
}
</style>
