<template>
  <div class="renew-editor">
    <header class="renew-head">
      <h4 class="renew-title"><i class="bi bi-arrow-repeat" /> 续期设置</h4>
      <FormCheckbox
        class="renew-switch"
        variant="switch"
        v-model="enabled"
        :label="enabled ? '支持续期' : '不支持续期'" />
    </header>

    <p class="renew-desc">
      <template v-if="enabled">
        过期后用户可按下面的档位续期，<strong>价格只对这件商品生效</strong>；下单只认天数，实际扣费按这里配置的价格。
      </template>
      <template v-else>开启后可为这件商品配置「多少天 / 多少积分」的续期档位。</template>
    </p>

    <template v-if="enabled">
      <div v-if="!list.length" class="renew-empty">还没有档位，先加一个吧</div>

      <div v-for="(option, index) in list" :key="index" class="renew-row">
        <div class="renew-field">
          <label class="renew-label">天数</label>
          <FormInput
            v-model.number="option.days"
            type="number"
            min="1"
            max="3650"
            step="1"
            placeholder="30"
            @change="sortOptions" />
        </div>

        <div class="renew-field">
          <label class="renew-label">价格（积分）</label>
          <FormInput v-model.number="option.price" type="number" min="0" step="1" placeholder="0 = 免费" />
        </div>

        <div class="renew-field">
          <label class="renew-label">划线价</label>
          <FormInput
            v-model.number="option.origin_price"
            type="number"
            min="0"
            step="1"
            placeholder="不填则不显示" />
        </div>

        <div class="renew-field">
          <label class="renew-label">标签</label>
          <FormInput v-model="option.tag" type="text" maxlength="8" placeholder="如：推荐" />
        </div>

        <button type="button" class="btn btn-ghost btn-sm renew-remove" title="删除该档位" @click="removeOption(index)">
          <i class="bi bi-trash3" />
        </button>
      </div>

      <p v-if="discountHint" class="renew-warn">
        <i class="bi bi-exclamation-triangle" /> {{ discountHint }}
      </p>

      <div class="renew-actions">
        <button type="button" class="btn btn-sm btn-ghost" @click="addOption(30)">
          <i class="bi bi-plus-lg" /> 添加档位
        </button>
        <span class="renew-presets">
          常用：
          <button v-for="preset in PRESETS" :key="preset" type="button" class="renew-preset" @click="addOption(preset)">
            {{ preset }} 天
          </button>
        </span>
      </div>
    </template>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 续期档位编辑器（后台「勋章 / 装扮」编辑表单共用）
 *
 * v-model 的值就是落库的 renew 结构：
 *   { enabled: true, options: [{ days: 30, price: 100, origin_price: 150, tag: '推荐' }] }
 *
 * 价格挂在这件商品自己身上（不用全站统一套餐），所以编辑器直接内嵌在
 * 勋章 / 装扮的表单里，改一件只影响一件。
 * 价格 / 划线价的最终校验在服务端（model.NormalizeRenewConfig），这里只做提示。
 */
import { computed, ref, watch } from 'vue'
import { parseRenewConfig } from '@/utils/renew'

const props = defineProps({
  // 允许传对象、JSON 字符串或 null（后台列表返回的是库里的原始字符串）
  modelValue: { type: [Object, String], default: () => ({}) }
})

const emit = defineEmits(['update:modelValue'])

const PRESETS = [30, 90, 180, 365]

// 本地可编辑副本：v-model 直接改它，改动再整体 emit 出去。
// （不能直接绑 props 派生出的对象 —— 那是每次求值新建的临时对象，输入会丢。）
const state = ref(parseRenewConfig(props.modelValue))

watch(
  () => props.modelValue,
  (val) => {
    const next = parseRenewConfig(val)
    // 与本地一致 = 自己刚 emit 出去的值回声，忽略；否则是切换了商品 / 被重置，同步过来
    if (JSON.stringify(next) === JSON.stringify(state.value)) return
    state.value = next
  }
)

watch(
  state,
  () => {
    emit('update:modelValue', {
      enabled: !!state.value.enabled,
      options: state.value.options.map((item) => ({ ...item }))
    })
  },
  { deep: true }
)

const enabled = computed({
  get: () => !!state.value.enabled,
  set: (val) => {
    state.value.enabled = !!val
    // 打开开关时没有档位就补一个默认 30 天档，省得用户以为坏了
    if (val && !state.value.options.length) {
      state.value.options = [{ days: 30, price: 0, origin_price: 0, tag: '' }]
    }
  }
})

const list = computed(() => state.value.options)

// 划线价填反了（低于售价）时提示：服务端会拒绝保存，别等提交才报错
const discountHint = computed(() => {
  const bad = list.value.filter(
    (item) => Number(item.origin_price) > 0 && Number(item.origin_price) < Number(item.price)
  )
  if (!bad.length) return ''
  return `${bad.map((item) => `${item.days} 天`).join('、')}档的划线价低于售价，保存会被拒绝`
})

function addOption(days) {
  const value = Number(days) || 30
  // 天数重复的档位后端会拒绝，这里直接跳过
  if (list.value.some((item) => Number(item.days) === value)) return
  state.value.options = [...list.value, { days: value, price: 0, origin_price: 0, tag: '' }].sort(
    (a, b) => Number(a.days) - Number(b.days)
  )
}

function removeOption(index) {
  state.value.options = list.value.filter((_, i) => i !== index)
}

// 天数改完再重排（输入过程中重排会让光标乱跳），保持「30 天 → 90 天 → 一年」的顺序
function sortOptions() {
  state.value.options = [...list.value].sort((a, b) => Number(a.days) - Number(b.days))
}
</script>

<style scoped>
.renew-editor {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px 14px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-muted);
}

/* 标题与开关同一行：说明文案另起一行，避免长段落把开关挤成竖排文字 */
.renew-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.renew-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  min-width: 0;
}

/* 开关是固定宽度的控件：不参与收缩，文案也不许折行 */
.renew-switch {
  flex: 0 0 auto;
  white-space: nowrap;
}

.renew-desc {
  margin: 0;
  font-size: 12px;
  line-height: 1.65;
  color: var(--text-muted);
}

.renew-empty {
  padding: 6px 0;
  font-size: 12px;
  color: var(--text-light);
}

.renew-row {
  display: grid;
  grid-template-columns: 88px 120px 120px minmax(96px, 1fr) 32px;
  align-items: end;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
}

.renew-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.renew-label {
  font-size: 11px;
  color: var(--text-muted);
}

.renew-remove {
  justify-content: center;
  color: var(--text-light);
}

.renew-remove:hover {
  color: var(--danger);
}

.renew-warn {
  margin: 0;
  font-size: 12px;
  color: var(--warning, #e08b00);
}

.renew-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.renew-presets {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-light);
}

.renew-preset {
  padding: 1px 8px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.renew-preset:hover {
  color: var(--primary-deep);
  border-color: var(--primary);
}

@media (max-width: 640px) {
  .renew-row {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .renew-remove {
    grid-column: 2 / -1;
    justify-self: end;
  }
}
</style>
