<template>
  <p class="owner-info" :title="title">
    <i class="bi bi-person-badge" aria-hidden="true" />
    <span class="owner-info-label">{{ label }}</span>
    <span class="owner-info-name">{{ name }}</span>
  </p>
</template>

<script setup>
/**
 * 内容归属人展示（审核弹窗 / 编辑弹窗共用）
 *
 * 后台四类内容都要求「审核或编辑时能直接看到这条是谁的」：
 *   文章 / 独立页面 / 动态 → 作者：昵称
 *   友链                  → 申请人：昵称
 *
 * 数据来自 utils/contentOwner 的模块级缓存：同一个 uid 只请求一次，
 * 列表与弹窗共用；取不到时回退「用户 {uid}」，uid=0（管理员自己创建）显示 fallback。
 * hover 提示里给出账号 / 昵称 / UID，方便核对具体是哪个账号。
 *
 * 用法：
 *   <OwnerInfo :uid="item.uid" label="作者" />
 *   <OwnerInfo :uid="edit.uid" label="申请人" fallback="管理员添加" />
 */
import { computed, onMounted, watch } from 'vue'
import { loadOwners, ownerName, ownerTitle } from '@/utils/contentOwner'

const props = defineProps({
  // 内容归属人 uid（文章/页面/动态的 author、友链的申请人）
  uid: { type: [Number, String], default: 0 },
  // 展示文案：文章/页面/动态用「作者」，友链用「申请人」
  label: { type: String, default: '作者' },
  // uid=0（管理员直接创建 / 老数据）时的兜底文案
  fallback: { type: String, default: '管理员添加' }
})

const name = computed(() => ownerName(props.uid, props.fallback))
const title = computed(() => ownerTitle(props.uid, props.fallback))

function ensure() {
  if (Number(props.uid) > 0) loadOwners([props.uid])
}

onMounted(ensure)
// 弹窗复用同一个组件实例时（切换编辑对象）要重新补数据
watch(() => props.uid, ensure)
</script>

<style scoped>
.owner-info {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-soft);
}
.owner-info > i {
  color: var(--text-muted);
}
.owner-info-label {
  color: var(--text-muted);
}
.owner-info-name {
  font-weight: 600;
  color: var(--text);
  word-break: break-all;
}
</style>
