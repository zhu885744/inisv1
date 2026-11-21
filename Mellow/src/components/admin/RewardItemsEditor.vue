<template>
  <div class="reward-editor">
    <div v-if="!list.length" class="reward-editor-empty">暂无奖励项，点下方「添加奖励」新增</div>

    <div v-for="(item, index) in list" :key="index" class="reward-row-wrap">
      <div class="reward-row">
        <div class="reward-row-asset" title="奖励类型（可扩展：见 model/reward.go 的资产注册表）">
          <FormSelect v-model="item.asset" :options="assetOptions" />
        </div>
        <!-- 卡密是「纯卡密」：没有面额，所以不显示数值输入；发什么由下方的卡密内容决定 -->
        <FormInput
          v-if="!isCard(item)"
          v-model.number="item.value"
          class="reward-row-value"
          type="number"
          min="0"
          step="1"
          :placeholder="valuePlaceholder(item)"
          :title="assetOf(item)?.desc || ''" />
        <FormInput
          v-if="chance"
          v-model.number="item.chance"
          class="reward-row-chance"
          type="number"
          min="0"
          max="100"
          step="1"
          placeholder="概率%"
          title="触发概率（0 或 100 表示必得）" />
        <FormInput
          v-model="item.label"
          class="reward-row-label"
          type="text"
          maxlength="16"
          placeholder="备注（可选）" />
        <button class="btn btn-ghost btn-sm" type="button" title="删除该奖励" @click="remove(index)">
          <i class="bi bi-x-lg" />
        </button>
      </div>

      <!-- 卡密：卡密内容（签到发出去的就是这些码） -->
      <div v-if="isCard(item)" class="reward-codes">
        <FormTextarea
          v-model="item.codesText"
          class="reward-codes-input"
          rows="3"
          placeholder="卡密内容：一行一个，例如&#10;SN2026000001&#10;SN2026000002"
          @input="scheduleStock(item)" />
        <p class="reward-codes-hint" :class="{ 'is-warn': !codeCount(item) }">
          <i class="bi bi-ticket-perforated" />
          <template v-if="!codeCount(item)">
            还没填卡密：这份奖励不会发卡密（库存为空，发完即失效，不会改发别的奖励）。
          </template>
          <template v-else>
            共 {{ codeCount(item) }} 张 · 剩余 {{ stockText(item) }}；一行一个，会自动去掉空格与连字符并转大写（长度需 8 ~ 64 位），
            每发一次消耗一张，库存发完即失效。
          </template>
        </p>
      </div>
    </div>

    <div class="reward-editor-foot">
      <button class="btn btn-sm" type="button" @click="add">
        <i class="bi bi-plus-lg" /> 添加奖励
      </button>
      <span v-if="hint" class="reward-editor-hint">{{ hint }}</span>
    </div>

    <!-- 选中卡密时给出该资产的说明，避免把「面额」当成张数填 -->
    <p v-if="cardHint" class="reward-editor-hint reward-editor-hint-full">
      <i class="bi bi-ticket-perforated" /> {{ cardHint }}
    </p>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 奖励项编辑器（后台签到页复用）
 *
 * 一行 = 一个奖励项：资产类型 + 数值（+ 卡密的卡密内容、随机奖励的概率），
 * 对应后端 model.RewardItem：
 *   { asset: 'exp' | 'integral' | 'card' | 自定义, value: 10, label: '幸运奖励' }
 *   { asset: 'card', value: 1, codes: ['SN2026000001'] } // 纯卡密：内容自己填，库存发完即失效
 *
 * 卡密没有面额（value 只是引擎要求的占位），所以不显示数值列；
 * 卡密内容用 textarea 编辑（item.codesText，一行一个），由父组件在保存时拆成数组 ——
 * 配置里存数组，编辑时按「一行一个」更顺手。
 *
 * 资产清单（名称 / 单位 / 图标 / 说明）来自 /api/checkin/rules 的 assets 字段，
 * 因此二次开发注册的新资产会自动出现在下拉框里。
 */
import { computed, onMounted, reactive } from 'vue'
import { getRewardCardStock } from '@/api/checkin'

const props = defineProps({
  // 奖励项数组：由父组件持有，编辑时原位更新
  modelValue: { type: Array, default: () => [] },
  // 可选资产清单：[{ key, name, unit, icon, desc }]，来自 /api/checkin/rules
  assets: { type: Array, default: () => [] },
  // 是否显示「概率」列（随机奖励用）
  chance: { type: Boolean, default: false },
  // 底部提示文案
  hint: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue'])

const list = computed(() => props.modelValue || [])

// 资产清单 → SelectMenu 选项（{key,name} → {value,label}），
// 新注册的资产（见 model/reward.go）会自动出现在这里
const assetOptions = computed(() =>
  (props.assets || []).map((asset) => ({ value: asset.key, label: asset.name }))
)

function assetOf(item) {
  return props.assets.find((asset) => asset.key === item?.asset) || null
}

function isCard(item) {
  return item?.asset === 'card'
}

// 数值的含义随资产而变；卡密是纯卡密、没有面额，模板里不给它显示这一列
function valuePlaceholder() {
  return '数值'
}

// 自定义卡密的张数（按换行/逗号/分号拆分，空行忽略），用于提示与空态提醒
function codeCount(item) {
  return String(item?.codesText || '')
    .split(/[\n\r,，;；]/)
    .map((code) => code.trim())
    .filter(Boolean).length
}

// ---------- 卡密库存 ----------
// 后台直接能看到每个卡密奖励还剩多少张：把当前填写的卡密交给后端统计
// （后端会顺带把新填的卡密补进库存，幂等）。按「内容签名」缓存，输入停顿 600ms 再请求。
const stocks = reactive({})
const stockTimers = new Map()

function splitCodes(text) {
  return String(text || '')
    .split(/[\n\r,，;；]/)
    .map((code) => code.trim())
    .filter(Boolean)
}

/** 内容签名：空格 / 换行不影响，用作缓存 key */
function signatureOf(item) {
  return splitCodes(item?.codesText).join('|')
}

function scheduleStock(item) {
  const signature = signatureOf(item)
  const codes = splitCodes(item?.codesText)

  if (!signature) {
    stocks[signature] = { total: 0, issued: 0, remain: 0, loading: false }
    return
  }

  stocks[signature] = { ...(stocks[signature] || { total: codes.length, issued: 0, remain: 0 }), loading: true }

  clearTimeout(stockTimers.get(signature))
  stockTimers.set(
    signature,
    setTimeout(async () => {
      try {
        const res = await getRewardCardStock(codes)
        const data = res?.data || {}
        stocks[signature] = {
          total: Number(data.total || 0),
          issued: Number(data.issued || 0),
          remain: Number(data.remain || 0),
          loading: false
        }
      } catch {
        stocks[signature] = { ...(stocks[signature] || {}), loading: false }
      }
    }, 600)
  )
}

/** 库存文案：统计中… / 还剩 N 张（已发 M） */
function stockText(item) {
  const stock = stocks[signatureOf(item)]
  if (!stock) return '未统计'
  if (stock.loading) return '统计中…'
  return `${Number(stock.remain || 0)} 张（已发 ${Number(stock.issued || 0)}）`
}

// 打开页面时先给已有的卡密奖励项统计一次
onMounted(() => {
  list.value.filter((item) => isCard(item)).forEach((item) => scheduleStock(item))
})

// 列表里只要有卡密项，就在底部提示一次卡密的规则
const cardHint = computed(() => {
  if (!list.value.some((item) => isCard(item))) return ''
  return '卡密是纯卡密：内容就是你填的那些码，与积分无关（没有面额、不在「积分 → 卡密」里兑换）；每发一次消耗一张，库存发完即失效，不会改发别的奖励。'
})

function add() {
  const asset = props.assets[0]?.key || 'exp'
  emit('update:modelValue', [...list.value, { asset, value: 1, label: '', codesText: '' }])
}

function remove(index) {
  const next = [...list.value]
  next.splice(index, 1)
  emit('update:modelValue', next)
}
</script>

<style scoped>
/* 每行一个奖励项；卡密的自定义卡密输入框挂在行下方，所以外面再包一层 */
.reward-row-wrap {
  margin-bottom: 8px;
}
.reward-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
/* 卡密的自定义卡密：一行一个的文本框 + 说明 */
.reward-codes {
  margin-top: 6px;
}
.reward-codes-input {
  width: 100%;
  resize: vertical;
  font-family: var(--font-mono, monospace);
  font-size: 12px;
  line-height: 1.7;
}
.reward-codes-hint {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.reward-codes-hint i {
  color: var(--primary);
}
/* 没填卡密时提醒一下：这份奖励不会发卡密 */
.reward-codes-hint.is-warn {
  color: var(--warning);
}
.reward-codes-hint.is-warn i {
  color: var(--warning);
}
.reward-row-asset {
  width: 110px;
  flex-shrink: 0;
}
.reward-row-value {
  width: 96px;
  flex-shrink: 0;
}
.reward-row-chance {
  width: 96px;
  flex-shrink: 0;
}
.reward-row-label {
  flex: 1;
  min-width: 0;
}
.reward-editor-empty {
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--text-muted);
}
.reward-editor-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.reward-editor-hint {
  font-size: 12px;
  color: var(--text-muted);
}
.reward-editor-hint-full {
  display: block;
  width: 100%;
  margin: 8px 0 0;
  line-height: 1.7;
}
.reward-editor-hint-full i {
  color: var(--primary);
}
</style>
