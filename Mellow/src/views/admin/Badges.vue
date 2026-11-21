<template>
  <div class="badge-admin">
    <!-- 左侧分类栏：只影响展示分组，不影响发放条件 -->
    <aside class="category-side">
      <div class="category-side-head">
        <span>勋章分类</span>
        <button class="btn btn-ghost btn-sm" title="新建分类" @click="openCategory(null)">
          <i class="bi bi-plus-lg" />
        </button>
      </div>

      <button
        type="button"
        class="category-item"
        :class="{ 'is-active': catFilter === null }"
        @click="pickCat(null)"
      >
        <i class="bi bi-grid-3x3-gap" />
        <span class="category-name">全部勋章</span>
        <span class="category-count">{{ totalAll }}</span>
      </button>

      <button
        v-for="cat in categories"
        :key="cat.id"
        type="button"
        class="category-item"
        :class="{ 'is-active': catFilter === cat.id, 'is-off': !cat.status }"
        :title="cat.description || cat.name"
        @click="pickCat(cat.id)"
      >
        <i :class="cat.icon || 'bi bi-collection'" :style="{ color: cat.color || undefined }" />
        <span class="category-name">{{ cat.name }}</span>
        <span class="category-count">{{ counts[cat.id] || 0 }}</span>
        <span class="category-edit" title="编辑分类" @click.stop="openCategory(cat)">
          <i class="bi bi-pencil" />
        </span>
      </button>

      <button
        type="button"
        class="category-item"
        :class="{ 'is-active': catFilter === 0 }"
        @click="pickCat(0)"
      >
        <i class="bi bi-collection" />
        <span class="category-name">未分类</span>
        <span class="category-count">{{ counts[0] || 0 }}</span>
      </button>

      <p class="category-hint">分类只决定前台怎么分组展示，不影响获得条件</p>
    </aside>

    <div class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">勋章管理</h2>
          <p class="block-desc">
            勋章可以按任务条件自动发放（发布文章、注册满 N 天…），也可以手动发放 / 收回
            <template v-if="total > 0"> · 共 {{ total }} 枚</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="trash || busy" @click="openGrant('grant')">
            <i class="bi bi-send" /> 发放勋章
          </button>
          <button class="btn btn-sm" :disabled="trash || busy" @click="openGrant('revoke')">
            <i class="bi bi-arrow-counterclockwise" /> 收回勋章
          </button>
          <button
            class="btn btn-sm"
            :disabled="trash || busy"
            title="按当前数据给全站满足条件的用户补发勋章（新建/修改条件后可用）"
            @click="askScan"
          >
            <i class="bi bi-magic" /> 补发判定
          </button>
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd">
            <i class="bi bi-plus-lg" /> 新建勋章
          </button>
        </div>
      </header>

      <div class="list-filter">
        <div class="status-tabs">
          <button type="button" class="status-tab" :class="{ 'is-active': !trash }" @click="switchTab(false)">
            全部勋章
          </button>
          <button type="button" class="status-tab trash-tab" :class="{ 'is-active': trash }" @click="switchTab(true)">
            <i class="bi bi-trash3" /> 回收站
          </button>
        </div>

        <div class="filter-right">
          <label v-if="!trash && list.length" class="check-all" title="选择本页全部勋章">
            <FormCheckbox :model-value="allSelected" @change="toggleSelectAll" />
            全选本页
          </label>
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索勋章名称…"
              aria-label="搜索勋章名称"
              @input="doSearch" />
          </div>
          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站内的勋章不再参与自动发放，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClear">
          清空回收站
        </button>
      </div>

      <!-- 批量操作：勾选勋章后直接挪到某个分类（分类改了不影响发放条件） -->
      <div v-if="!trash && selectedIds.length" class="batch-bar">
        <span class="batch-count"><i class="bi bi-check2-square" /> 已选 {{ selectedIds.length }} 枚</span>
        <FormSelect variant="pill" v-model="batchCategory" :options="batchCategoryOptions" placeholder="目标分类" />
        <button class="btn btn-primary btn-sm" :disabled="busy" @click="applyBatchMove">
          <i class="bi bi-arrow-right-square" /> 移动到该分类
        </button>
        <button class="btn btn-sm" :disabled="busy" @click="clearSelect">取消选择</button>
      </div>

      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-award'" :text="emptyText" />
      </div>

      <ul v-else class="badge-list">
        <li v-for="item in list" :key="item.id" class="badge-row">
          <label v-if="!trash" class="row-check" :title="`选择「${item.name}」`">
            <FormCheckbox
              :model-value="selectedIds.includes(item.id)"
              @change="toggleSelect(item.id)" />
          </label>
          <span class="badge-icon" :style="iconStyle(item)">
            <img v-if="item.image" :src="item.image" :alt="item.name" loading="lazy" />
            <i v-else :class="item.icon || 'bi bi-award'" />
          </span>

          <div class="badge-main">
            <div class="badge-name-row">
              <span class="badge-name">{{ item.name }}</span>
              <span class="tag" :class="item.status ? 'tag-primary' : ''">
                {{ item.status ? '启用' : '停用' }}
              </span>
              <span class="cond-chip">
                <i class="bi bi-flag" /> {{ conditionText(item) }}
              </span>
              <span v-if="item.condition === 'manual'" class="cond-chip is-manual">
                <i class="bi bi-hand-index-thumb" /> 手动
              </span>
              <span class="cond-chip is-cat">
                <i class="bi bi-collection" /> {{ categoryName(item.category_id) }}
              </span>
              <span class="cond-chip is-rarity" :class="`rarity-${item.rarity || 'common'}`">
                {{ BADGE_RARITY_TEXT[item.rarity] || '普通' }}
              </span>
              <span v-if="isLimited(item)" class="cond-chip is-limit" :title="limitText(item)">
                <i class="bi bi-hourglass-split" /> {{ limitText(item) }}
              </span>
              <span v-if="renewChip(item)" class="cond-chip is-renew" :title="renewTip(item)">
                <i class="bi bi-arrow-repeat" /> {{ renewChip(item) }}
              </span>
            </div>
            <p class="badge-desc">{{ item.description || '暂无说明' }}</p>
            <div class="badge-meta">
              <span class="meta-text">#{{ item.id }}</span>
              <span class="meta-text">排序 {{ item.sort ?? 0 }}</span>
              <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
              <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
            </div>
          </div>

          <div class="badge-actions">
            <template v-if="trash">
              <button class="btn btn-ghost btn-sm" title="恢复" :disabled="busy" @click="restore(item)">
                <i class="bi bi-arrow-counterclockwise" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                <i class="bi bi-x-octagon" />
              </button>
            </template>
            <template v-else>
              <button class="btn btn-ghost btn-sm" title="持有记录" @click="openRecords(item)">
                <i class="bi bi-people" />
              </button>
              <button class="btn btn-ghost btn-sm" title="发放 / 收回" @click="openGrant('grant', item)">
                <i class="bi bi-send" />
              </button>
              <button class="btn btn-ghost btn-sm" title="编辑" @click="openEdit(item)">
                <i class="bi bi-pencil" />
              </button>
              <button class="btn btn-ghost btn-sm danger" title="删除" :disabled="busy" @click="askRemove(item)">
                <i class="bi bi-trash" />
              </button>
            </template>
          </div>
        </li>
      </ul>

      <Pagination
        v-if="!loading && total > 0"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :sizes="PAGE_SIZE_OPTIONS"
        @update:page-size="(size) => { pageSize = size; page = 1; load() }"
        @update:current="changePage"
      />
    </div>

    <!-- 新建 / 编辑 -->
    <AdminFormDialog
      v-model:visible="dialog.visible"
      :title="dialog.id ? '编辑勋章' : '新建勋章'"
      :icon="dialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="dialog.loading"
      confirm-text="保存"
      width="640px"
      @confirm="save"
    >
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">勋章名称</label>
          <FormInput v-model="dialog.name" type="text" maxlength="64" placeholder="如「初出茅庐」" />
        </div>
        <div class="form-item">
          <label class="form-label">排序</label>
          <FormInput v-model="dialog.sort" type="number" step="1" placeholder="数字越大越靠前" />
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">所属分类</label>
        <div class="chip-list">
          <button
            type="button"
            class="pick-chip"
            :class="{ 'is-active': dialog.category_id === 0 }"
            @click="dialog.category_id = 0"
          >
            <i class="bi bi-collection" /> 未分类
          </button>
          <button
            v-for="cat in categories"
            :key="cat.id"
            type="button"
            class="pick-chip"
            :class="{ 'is-active': dialog.category_id === cat.id }"
            @click="dialog.category_id = cat.id"
          >
            <i :class="cat.icon || 'bi bi-collection'" /> {{ cat.name }}
          </button>
        </div>
        <p class="form-hint">分类决定前台勋章墙怎么分组，随时可改</p>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">图标（Bootstrap Icons）</label>
          <div class="icon-input">
            <span class="icon-preview" :style="iconStyle(dialog)">
              <i :class="dialog.icon || 'bi bi-award'" />
            </span>
            <FormInput v-model="dialog.icon" type="text" placeholder="bi bi-award" />
          </div>
          <p class="form-hint">形如 <code>bi bi-award</code>，可在 icons.getbootstrap.com 查</p>
        </div>
        <div class="form-item">
          <label class="form-label">主题色</label>
          <div class="color-input">
            <FormInput v-model="dialog.color" type="text" placeholder="#d4a017" />
            <input v-model="dialog.color" class="color-picker" type="color" />
          </div>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">勋章图片（可选，优先于图标）</label>
        <div class="image-field">
          <span class="image-preview">
            <img v-if="dialog.image" :src="dialog.image" alt="" />
            <i v-else class="bi bi-image" />
          </span>
          <FormInput v-model="dialog.image" type="text" placeholder="填写图片 URL，留空则用图标" />
        </div>
        <div class="image-actions">
          <button type="button" class="btn btn-sm" :disabled="uploadingImage" @click="pickBadgeImage">
            <i class="bi bi-upload" /> {{ uploadingImage ? '上传中...' : '上传图片' }}
          </button>
          <button type="button" class="btn btn-sm" title="从附件库选择" @click="showImageLibrary = true">
            <i class="bi bi-folder2-open" /> 附件库
          </button>
          <button v-if="dialog.image" type="button" class="btn btn-sm btn-ghost" @click="dialog.image = ''">
            移除
          </button>
        </div>
        <input ref="imageFileRef" type="file" accept="image/*" hidden @change="onBadgeImageChange" />
        <p class="form-hint">支持 jpg / png / webp / gif，建议正方形，留空则用上面的图标</p>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">获得条件</label>
          <FormSelect
            v-model="dialog.condition"
            :options="conditionOptions"
            @change="onConditionChange" />
        </div>
        <div class="form-item">
          <label class="form-label">条件阈值</label>
          <FormInput
            v-model="dialog.condition_value"
            type="number"
            min="0"
            step="1"
            :disabled="dialog.condition === 'manual'"
            :placeholder="conditionPlaceholder" />
          <p class="form-hint">手动发放的勋章不需要阈值</p>
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">稀有度</label>
          <FormSelect v-model="dialog.rarity" :options="RARITY_OPTIONS" />
          <p class="form-hint">只影响展示：描边、光效、标签</p>
        </div>
        <div class="form-item">
          <label class="form-label">获得后有效天数</label>
          <FormInput
            v-model="dialog.expire_days"
            type="number"
            min="0"
            step="1"
            placeholder="0 = 永久有效" />
          <p class="form-hint">填 >0 即为限定勋章：按发放时间算过期，过期后不能佩戴</p>
        </div>
      </div>

      <!-- 续期：档位与价格是这枚勋章自己的，用户过期后按这里的档位续费 -->
      <RenewOptionsEditor v-model="dialog.renew" />
      <p v-if="Number(dialog.expire_days) <= 0" class="form-hint">
        当前勋章有效天数为 0（永久有效），续期档位不会在前台展示
      </p>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">可获取开始时间</label>
          <FormInput v-model="dialog.start_time" type="datetime-local" />
        </div>
        <div class="form-item">
          <label class="form-label">可获取结束时间</label>
          <FormInput v-model="dialog.end_time" type="datetime-local" />
        </div>
      </div>
      <p class="form-hint">留空 = 不限；窗口关上后不再自动发放，已经获得的不受影响</p>

      <div class="form-item">
        <label class="form-label">勋章说明</label>
        <FormTextarea v-model="dialog.description" rows="2" placeholder="展示给用户的获得说明，留空则按条件自动生成" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">状态</label>
          <FormSelect v-model="dialog.status" :options="STATUS_OPTIONS" />
        </div>
        <div class="form-item">
          <label class="form-label">备注</label>
          <FormInput v-model="dialog.remark" type="text" placeholder="仅后台可见" />
        </div>
      </div>
    </AdminFormDialog>

    <!-- 发放 / 收回 -->
    <AdminFormDialog
      v-model:visible="grant.visible"
      :title="grant.mode === 'grant' ? '发放勋章' : '收回勋章'"
      :icon="grant.mode === 'grant' ? 'bi bi-send' : 'bi bi-arrow-counterclockwise'"
      :loading="grant.loading"
      :confirm-text="grant.mode === 'grant' ? '发放' : '收回'"
      width="620px"
      @confirm="submitGrant"
    >
      <div class="form-item">
        <label class="form-label">选择勋章</label>
        <div class="chip-list">
          <button
            v-for="item in allBadges"
            :key="item.id"
            type="button"
            class="pick-chip"
            :class="{ 'is-active': grant.badgeIds.includes(item.id) }"
            @click="toggleBadgePick(item.id)"
          >
            <i :class="item.icon || 'bi bi-award'" /> {{ item.name }}
          </button>
        </div>
        <p class="form-hint">可多选</p>
      </div>

      <div class="form-item">
        <label class="form-label">用户 ID</label>
        <FormInput v-model="grant.uids" type="text" placeholder="多个用户用逗号分隔，如 2,3,4" />
        <p class="form-hint">可以从「用户管理」里复制用户 ID</p>
      </div>

      <div class="form-item">
        <label class="form-label">备注（可选）</label>
        <FormInput v-model="grant.remark" type="text" placeholder="记录发放原因，如「活动奖励」" />
      </div>
    </AdminFormDialog>

    <!-- 勋章图片：直接复用附件库里已上传的图片 -->
    <AttachmentLibrary
      v-model:visible="showImageLibrary"
      title="选择勋章图片"
      accept="image"
      @select="onImageLibrarySelect"
    />

    <!-- 持有记录 -->
    <AdminFormDialog
      v-model:visible="records.visible"
      title="持有记录"
      icon="bi bi-people"
      width="720px"
      @confirm="records.visible = false"
    >
      <template #footer>
        <button class="btn" type="button" @click="records.visible = false">关闭</button>
      </template>

      <div v-if="records.loading" class="is-loading"><span class="spinner" /> 加载中...</div>
      <div v-else-if="!records.list.length" class="empty-row">
        <EmptyState icon="bi bi-people" text="还没有用户获得这枚勋章" />
      </div>
      <ul v-else class="record-list">
        <li v-for="item in records.list" :key="item.id" class="record-row">
          <img v-if="item.user?.avatar" class="record-avatar" :src="item.user.avatar" alt="" loading="lazy" />
          <span v-else class="record-avatar is-empty"><i class="bi bi-person" /></span>
          <div class="record-main">
            <span class="record-name">{{ item.user?.nickname || item.user?.account || `用户 #${item.uid}` }}</span>
            <span class="record-meta">
              UID {{ item.uid }} · {{ item.source === 'auto' ? '自动发放' : '手动发放' }} · {{ fromNow(item.create_time) }}
            </span>
          </div>
          <span v-if="item.remark" class="record-remark">{{ item.remark }}</span>
          <button class="btn btn-ghost btn-sm danger" title="收回" :disabled="busy" @click="revokeOne(item)">
            <i class="bi bi-arrow-counterclockwise" />
          </button>
        </li>
      </ul>
      <p class="form-hint">共 {{ records.total }} 条</p>
    </AdminFormDialog>

    <!-- 新建 / 编辑分类 -->
    <AdminFormDialog
      v-model:visible="catDialog.visible"
      :title="catDialog.id ? '编辑分类' : '新建分类'"
      :icon="catDialog.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="catDialog.loading"
      confirm-text="保存"
      width="560px"
      @confirm="saveCategory"
    >
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">分类名称</label>
          <FormInput v-model="catDialog.name" type="text" maxlength="32" placeholder="如「限定勋章」" />
        </div>
        <div class="form-item">
          <label class="form-label">排序</label>
          <FormInput v-model="catDialog.sort" type="number" step="1" placeholder="数字越大越靠前" />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">图标（Bootstrap Icons）</label>
          <div class="icon-input">
            <span class="icon-preview" :style="iconStyle(catDialog)">
              <i :class="catDialog.icon || 'bi bi-collection'" />
            </span>
            <FormInput v-model="catDialog.icon" type="text" placeholder="bi bi-collection" />
          </div>
        </div>
        <div class="form-item">
          <label class="form-label">主题色</label>
          <div class="color-input">
            <FormInput v-model="catDialog.color" type="text" placeholder="#4f7cff" />
            <input v-model="catDialog.color" class="color-picker" type="color" />
          </div>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">分类说明（前台分组头部展示）</label>
        <FormTextarea
          v-model="catDialog.description"
          rows="2"
          placeholder="如「靠日常活跃攒出来的勋章」" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">状态</label>
          <FormSelect v-model="catDialog.status" :options="CATEGORY_STATUS_OPTIONS" />
        </div>
        <div class="form-item">
          <label class="form-label">封面图（可选）</label>
          <FormInput v-model="catDialog.cover" type="text" placeholder="图片 URL，留空则不显示" />
        </div>
      </div>

      <template #footer>
        <button
          v-if="catDialog.id"
          class="btn btn-danger"
          type="button"
          :disabled="catDialog.loading"
          @click="askRemoveCategory"
        >
          <i class="bi bi-trash" /> 删除分类
        </button>
        <span class="footer-spacer" />
        <button class="btn" type="button" :disabled="catDialog.loading" @click="catDialog.visible = false">
          取消
        </button>
        <button class="btn btn-primary" type="button" :disabled="catDialog.loading" @click="saveCategory">
          {{ catDialog.loading ? '保存中...' : '保存' }}
        </button>
      </template>
    </AdminFormDialog>

    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :danger="confirm.danger"
      :loading="confirm.loading"
      @confirm="runConfirm"
    />
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 勋章管理（/admin/badges）
 *
 * 两类能力：
 *  1. 勋章定义 CRUD（名称/图标/颜色/条件/阈值/启用状态）；
 *  2. 手动发放 / 收回（按 uid + 勋章 ID 批量），以及查看「谁获得了」。
 *
 * 自动发放由后端负责（app/model/badge.go 的 BadgeAutoGrant + app/timer/badge.go），
 * 本页只负责配置与人工干预；改完条件对「已经发放」的记录不追溯，需要收回请手动操作。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import RenewOptionsEditor from '@/components/RenewOptionsEditor.vue'
import { emptyRenewConfig, parseRenewConfig } from '@/utils/renew'
import {
  listBadges,
  saveBadge,
  removeBadges,
  forceDeleteBadges,
  restoreBadges,
  clearBadgeRecycle,
  getBadgeConditions,
  grantBadges,
  revokeBadges,
  listBadgeRecords,
  scanBadges,
  listBadgeCategories,
  saveBadgeCategory,
  removeBadgeCategory,
  moveBadgesToCategory,
  BADGE_CONDITION_TEXT,
  BADGE_RARITY_TEXT
} from '@/api/badge'
import { uploadAttachments } from '@/api/attachment'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const keyword = ref('')
const searchKey = ref('')

// 条件清单：优先后端返回，接口异常时用前端兜底文案
const conditions = ref(
  Object.keys(BADGE_CONDITION_TEXT).map((key) => ({ key, name: BADGE_CONDITION_TEXT[key], unit: '' }))
)

// 下拉一律用 SelectMenu（主题自绘），不用浏览器原生 select
const conditionOptions = computed(() =>
  conditions.value.map((item) => ({
    value: item.key,
    label: item.unit ? `${item.name}（${item.unit}）` : item.name
  }))
)

const RARITY_OPTIONS = Object.keys(BADGE_RARITY_TEXT).map((key) => ({
  value: key,
  label: BADGE_RARITY_TEXT[key]
}))

const STATUS_OPTIONS = [
  { value: 1, label: '启用' },
  { value: 0, label: '停用' }
]

const CATEGORY_STATUS_OPTIONS = [
  { value: 1, label: '启用' },
  { value: 0, label: '停用（前台不展示该分组）' }
]

// 发放弹窗里的勋章选择（取全部启用勋章，不受分页影响）
const allBadges = ref([])

// 分类栏：categories 是分类定义，counts 是「分类 → 勋章数量」（key=0 为未分类）
const categories = ref([])
const counts = ref({})
// null = 全部；0 = 未分类；其余为分类 id
const catFilter = ref(null)
// side 栏「全部勋章」的数量：counts 之和（不受当前筛选影响）
const totalAll = computed(() =>
  Object.values(counts.value).reduce((sum, n) => sum + (Number(n) || 0), 0)
)

// 批量移动：勾选的勋章 id + 目标分类
const selectedIds = ref([])
const batchCategory = ref(0)
const batchCategoryOptions = computed(() => [
  { value: 0, label: '未分类' },
  ...categories.value.map((cat) => ({ value: cat.id, label: cat.name }))
])
const allSelected = computed(
  () => list.value.length > 0 && list.value.every((item) => selectedIds.value.includes(item.id))
)

// 分类筛选 → 面包屑 / 空态文案
const currentCategory = computed(() => {
  if (catFilter.value === null) return null
  if (catFilter.value === 0) return { id: 0, name: '未分类' }
  return categories.value.find((item) => item.id === catFilter.value) || null
})

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (currentCategory.value) return `「${currentCategory.value.name}」下还没有勋章`
  return '还没有勋章，点右上角「新建勋章」'
})

const dialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  description: '',
  icon: 'bi bi-award',
  image: '',
  color: '#d4a017',
  condition: 'manual',
  condition_value: 0,
  category_id: 0,
  rarity: 'common',
  // 表单里用 datetime-local 字符串，提交时转成 unix 秒
  start_time: '',
  end_time: '',
  expire_days: 0,
  // 续期档位（这枚勋章自己的，见 components/RenewOptionsEditor.vue）
  renew: emptyRenewConfig(),
  sort: 0,
  status: 1,
  remark: ''
})

// 勋章图片：本地上传 / 附件库二选一，最终都是把图片地址写回 dialog.image
const uploadingImage = ref(false)
const showImageLibrary = ref(false)
const imageFileRef = ref(null)

function pickBadgeImage() {
  imageFileRef.value?.click()
}

async function onBadgeImageChange(e) {
  const file = e.target.files?.[0]
  // 清掉 value，否则再选同一个文件不会触发 change
  e.target.value = ''
  if (!file) return

  if (!String(file.type).startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploadingImage.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')

    dialog.image = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploadingImage.value = false
  }
}

function onImageLibrarySelect(urls = []) {
  showImageLibrary.value = false
  const url = urls[0]
  if (url) dialog.image = url
}

// 分类弹窗：与勋章弹窗共用同一套表单控件
const catDialog = reactive({
  visible: false,
  loading: false,
  id: 0,
  name: '',
  description: '',
  icon: 'bi bi-collection',
  color: '#4f7cff',
  cover: '',
  sort: 0,
  status: 1
})

const grant = reactive({
  visible: false,
  loading: false,
  mode: 'grant',
  badgeIds: [],
  uids: '',
  remark: ''
})

const records = reactive({
  visible: false,
  loading: false,
  badgeId: 0,
  list: [],
  total: 0
})

const confirm = reactive({
  visible: false,
  loading: false,
  title: '',
  message: '',
  confirmText: '确定',
  danger: false,
  action: null
})

const doSearch = debounce(() => {
  searchKey.value = keyword.value.trim()
  page.value = 1
  load()
}, 300)

function switchTab(value) {
  trash.value = value
  page.value = 1
  load()
}

function changePage(next) {
  page.value = next
  load()
}

async function load() {
  loading.value = true
  // 翻页 / 换筛选后旧的勾选已经不在视图里，直接清掉，避免「批量移动」误伤
  selectedIds.value = []
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (searchKey.value) params.like = `name|${searchKey.value}`
    if (trash.value) params.onlyTrashed = true
    // 分类筛选走 where：后端 buildQuery 会解析 JSON（0 = 未分类，也能精确筛到）
    if (catFilter.value !== null) params.where = JSON.stringify({ category_id: catFilter.value })
    const res = await listBadges(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    loading.value = false
  }
}

// 分类列表 + 每个分类的勋章数量（counts 由 badge-category/all 一并返回）
async function loadCategories() {
  try {
    const res = await listBadgeCategories()
    categories.value = res.data?.data || []
    counts.value = res.data?.counts || {}
  } catch {
    categories.value = []
    counts.value = {}
  }
}

// 切换分类筛选
function pickCat(id) {
  catFilter.value = id
  page.value = 1
  load()
}

// 分类 id → 名称（列表行的分类标签）
function categoryName(id) {
  const categoryId = Number(id) || 0
  if (!categoryId) return '未分类'
  return categories.value.find((item) => item.id === categoryId)?.name || '未分类'
}

/* ---------------- 批量移动分类 ---------------- */

function toggleSelect(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleSelectAll(event) {
  selectedIds.value = event.target.checked ? list.value.map((item) => item.id) : []
}

function clearSelect() {
  selectedIds.value = []
}

async function applyBatchMove() {
  if (!selectedIds.value.length) return
  busy.value = true
  try {
    const count = selectedIds.value.length
    await moveBadgesToCategory(selectedIds.value, batchCategory.value)
    toast.success(`已把 ${count} 枚勋章移动到「${categoryName(batchCategory.value)}」`)
    load()
    loadCategories()
  } catch {
    // 拦截器已提示
  } finally {
    busy.value = false
  }
}

async function loadAllBadges() {
  try {
    const res = await listBadges({ page: 1, limit: 100, field: 'id,name,icon,image,color,condition' })
    allBadges.value = res.data?.data || []
  } catch {
    allBadges.value = []
  }
}

async function loadConditions() {
  try {
    const res = await getBadgeConditions()
    if (Array.isArray(res.data) && res.data.length) conditions.value = res.data
  } catch {
    // 用兜底文案
  }
}

/* ---------------- 表单 ---------------- */

function resetDialog(categoryId = 0) {
  Object.assign(dialog, {
    visible: true,
    loading: false,
    id: 0,
    name: '',
    description: '',
    icon: 'bi bi-award',
    image: '',
    color: '#d4a017',
    condition: 'manual',
    condition_value: 0,
    category_id: categoryId,
    rarity: 'common',
    start_time: '',
    end_time: '',
    expire_days: 0,
    renew: emptyRenewConfig(),
    sort: 0,
    status: 1,
    remark: ''
  })
}

// 正在按某个分类筛选时新建，默认就落在该分类里（省一次手动选择）
function openAdd() {
  resetDialog(catFilter.value && catFilter.value > 0 ? catFilter.value : 0)
}

function openEdit(item) {
  resetDialog()
  Object.assign(dialog, {
    id: item.id,
    name: item.name || '',
    description: item.description || '',
    icon: item.icon || 'bi bi-award',
    image: item.image || '',
    color: item.color || '#d4a017',
    condition: item.condition || 'manual',
    condition_value: Number(item.condition_value) || 0,
    category_id: Number(item.category_id) || 0,
    rarity: item.rarity || 'common',
    start_time: toLocalInput(item.start_time),
    end_time: toLocalInput(item.end_time),
    expire_days: Number(item.expire_days) || 0,
    // 列表返回的是库里的 JSON 字符串，统一解析成 { enabled, options }
    renew: parseRenewConfig(item.renew),
    sort: Number(item.sort) || 0,
    status: Number(item.status ?? 1),
    remark: item.remark || ''
  })
}

const conditionPlaceholder = computed(() => {
  const hit = conditions.value.find((item) => item.key === dialog.condition)
  return hit?.unit ? `达到多少${hit.unit}` : '手动勋章无需填写'
})

function onConditionChange() {
  if (dialog.condition === 'manual') dialog.condition_value = 0
}

async function save() {
  if (!dialog.name.trim()) {
    toast.warning('请填写勋章名称')
    return
  }
  // 时间窗反了就发不出去，提前拦一下（后端只按 0 判断「不限」，不会自动纠正）
  if (dialog.start_time && dialog.end_time && toUnix(dialog.end_time) <= toUnix(dialog.start_time)) {
    toast.warning('结束时间必须晚于开始时间')
    return
  }
  dialog.loading = true
  try {
    const payload = {
      name: dialog.name.trim(),
      description: dialog.description,
      icon: dialog.icon,
      image: dialog.image,
      color: dialog.color,
      condition: dialog.condition,
      condition_value: Number(dialog.condition_value) || 0,
      category_id: Number(dialog.category_id) || 0,
      rarity: dialog.rarity || 'common',
      start_time: toUnix(dialog.start_time),
      end_time: toUnix(dialog.end_time),
      expire_days: Math.max(0, Number(dialog.expire_days) || 0),
      renew: dialog.renew,
      sort: Number(dialog.sort) || 0,
      status: Number(dialog.status)
    }
    if (dialog.remark) payload.remark = dialog.remark
    if (dialog.id) payload.id = dialog.id
    await saveBadge(payload)
    toast.success(dialog.id ? '已保存' : '已创建')
    dialog.visible = false
    load()
    loadAllBadges()
    // 分类数量（未分类 → 某分类）会变，侧栏跟着刷新
    loadCategories()
  } catch {
    // 拦截器已提示
  } finally {
    dialog.loading = false
  }
}

/* ---------------- 分类管理 ---------------- */

function openCategory(cat) {
  Object.assign(catDialog, {
    visible: true,
    loading: false,
    id: cat?.id || 0,
    name: cat?.name || '',
    description: cat?.description || '',
    icon: cat?.icon || 'bi bi-collection',
    color: cat?.color || '#4f7cff',
    cover: cat?.cover || '',
    sort: Number(cat?.sort) || 0,
    status: Number(cat?.status ?? 1)
  })
}

async function saveCategory() {
  if (!catDialog.name.trim()) {
    toast.warning('请填写分类名称')
    return
  }
  catDialog.loading = true
  try {
    const payload = {
      name: catDialog.name.trim(),
      description: catDialog.description,
      icon: catDialog.icon,
      color: catDialog.color,
      cover: catDialog.cover,
      sort: Number(catDialog.sort) || 0,
      status: Number(catDialog.status)
    }
    if (catDialog.id) payload.id = catDialog.id
    await saveBadgeCategory(payload)
    toast.success(catDialog.id ? '已保存' : '已创建')
    catDialog.visible = false
    loadCategories()
    load()
  } catch {
    // 拦截器已提示
  } finally {
    catDialog.loading = false
  }
}

// 删除分类：force=1 —— 分类下的勋章退化为「未分类」，勋章本身与用户持有记录都不动
function askRemoveCategory() {
  confirm.title = '删除分类'
  confirm.message = `确定删除「${catDialog.name}」？该分类下的勋章会变成「未分类」，勋章本身和用户已获得的记录都不会丢失。`
  confirm.confirmText = '删除'
  confirm.danger = true
  confirm.action = async () => {
    await removeBadgeCategory([catDialog.id], true)
    toast.success('已删除分类')
    catDialog.visible = false
    // 当前正按这个分类筛选的话，退回「全部」
    if (catFilter.value === catDialog.id) catFilter.value = null
    loadCategories()
  }
  confirm.visible = true
}

/* ---------------- 发放 / 收回 ---------------- */

function openGrant(mode, item) {
  grant.mode = mode
  grant.badgeIds = item ? [item.id] : []
  grant.uids = ''
  grant.remark = ''
  grant.visible = true
  if (!allBadges.value.length) loadAllBadges()
}

function toggleBadgePick(id) {
  const next = new Set(grant.badgeIds)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  grant.badgeIds = [...next]
}

async function submitGrant() {
  if (!grant.badgeIds.length) {
    toast.warning('请选择勋章')
    return
  }
  const uids = grant.uids
    .split(/[,，\s]+/)
    .map((item) => Number(item))
    .filter((item) => item > 0)

  if (!uids.length) {
    toast.warning('请填写用户 ID')
    return
  }

  grant.loading = true
  try {
    const action = grant.mode === 'grant' ? grantBadges : revokeBadges
    const res = await action({
      uids,
      badge_ids: grant.badgeIds,
      remark: grant.remark
    })
    const success = res.data?.success ?? 0
    const failed = res.data?.failed ?? 0
    if (success && !failed) toast.success(`成功 ${success} 条`)
    else if (success) toast.info(`成功 ${success} 条，失败 ${failed} 条（可能已持有 / 未持有）`)
    else toast.warning('没有变更（该用户可能已持有或未持有这枚勋章）')
    grant.visible = false
    if (records.visible) loadRecords(records.badgeId)
  } catch {
    // 拦截器已提示
  } finally {
    grant.loading = false
  }
}

/* ---------------- 持有记录 ---------------- */

function openRecords(item) {
  records.badgeId = item.id
  records.list = []
  records.total = 0
  records.visible = true
  loadRecords(item.id)
}

async function loadRecords(badgeId) {
  records.loading = true
  try {
    const res = await listBadgeRecords({ badge_id: badgeId, page: 1, limit: 50 })
    records.list = res.data?.data || []
    records.total = res.data?.count || 0
  } catch {
    // 拦截器已提示
  } finally {
    records.loading = false
  }
}

async function revokeOne(item) {
  busy.value = true
  try {
    await revokeBadges({ uids: [item.uid], badge_ids: [item.badge_id] })
    toast.success('已收回')
    loadRecords(records.badgeId)
    load()
  } catch {
    // 拦截器已提示
  } finally {
    busy.value = false
  }
}

/* ---------------- 删除 / 恢复 ---------------- */

function askRemove(item) {
  confirm.title = '删除勋章'
  confirm.message = `确定把「${item.name}」移入回收站？已发放的记录会保留。`
  confirm.confirmText = '删除'
  confirm.danger = true
  confirm.action = async () => {
    await removeBadges([item.id])
    toast.success('已移入回收站')
  }
  confirm.visible = true
}

function askForceDelete(item) {
  confirm.title = '彻底删除'
  confirm.message = `「${item.name}」将被永久删除且不可恢复，确定继续？`
  confirm.confirmText = '彻底删除'
  confirm.danger = true
  confirm.action = async () => {
    await forceDeleteBadges([item.id])
    toast.success('已彻底删除')
  }
  confirm.visible = true
}

// 全站补发：老站上线勋章功能、或改了条件之后，让存量用户不再「条件满足却没勋章」
function askScan() {
  confirm.title = '补发判定'
  confirm.message = '将按当前条件扫描全站用户并补发勋章（已持有的不会重复发放），确定继续？'
  confirm.confirmText = '开始扫描'
  confirm.danger = false
  confirm.action = async () => {
    await scanBadges()
    toast.success('已触发扫描，稍后刷新列表查看')
  }
  confirm.visible = true
}

function askClear() {
  confirm.title = '清空回收站'
  confirm.message = '回收站内的勋章将被永久删除，确定继续？'
  confirm.confirmText = '清空'
  confirm.danger = true
  confirm.action = async () => {
    await clearBadgeRecycle()
    toast.success('已清空')
  }
  confirm.visible = true
}

async function runConfirm() {
  if (typeof confirm.action !== 'function') {
    confirm.visible = false
    return
  }
  confirm.loading = true
  try {
    await confirm.action()
    confirm.visible = false
    load()
    loadAllBadges()
    loadCategories()
  } catch {
    // 拦截器已提示
  } finally {
    confirm.loading = false
  }
}

async function restore(item) {
  busy.value = true
  try {
    await restoreBadges([item.id])
    toast.success('已恢复')
    load()
    loadAllBadges()
  } catch {
    // 拦截器已提示
  } finally {
    busy.value = false
  }
}

/* ---------------- 展示辅助 ---------------- */

// 主题色 → 半透明边框 / 背景（8 位十六进制，避免依赖 color-mix）
function tint(value, alpha) {
  return /^#[0-9a-f]{6}$/i.test(value) ? `${value}${alpha}` : 'var(--border)'
}

function iconStyle(item) {
  const color = item.color || 'var(--warning)'
  return {
    color,
    borderColor: tint(color, '59'),
    background: tint(color, '1f')
  }
}

function conditionText(item) {
  if (item.condition === 'manual') return '手动发放'
  const name = item.result?.condition_name || BADGE_CONDITION_TEXT[item.condition] || item.condition
  const unit = item.result?.condition_unit || ''
  return `${name} ≥ ${item.condition_value ?? 0}${unit}`
}

function timeText(item) {
  return item.create_time ? fromNow(item.create_time) : '—'
}

/* ---------------- 限时 / 稀有度 ---------------- */

// unix 秒 → datetime-local 字符串（按浏览器本地时区）
function toLocalInput(unix) {
  const value = Number(unix) || 0
  if (!value) return ''
  const date = new Date(value * 1000)
  const pad = (v) => String(v).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// datetime-local 字符串 → unix 秒（空值 = 0 = 不限）
function toUnix(value) {
  if (!value) return 0
  const time = new Date(value).getTime()
  return Number.isFinite(time) ? Math.floor(time / 1000) : 0
}

function isLimited(item) {
  return Boolean(item.start_time || item.end_time || Number(item.expire_days) > 0)
}

function dateText(unix) {
  const date = new Date(Number(unix) * 1000)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function limitText(item) {
  const parts = []
  if (item.start_time && item.end_time) parts.push(`${dateText(item.start_time)} ~ ${dateText(item.end_time)}`)
  else if (item.end_time) parts.push(`${dateText(item.end_time)} 截止`)
  else if (item.start_time) parts.push(`${dateText(item.start_time)} 起`)
  if (Number(item.expire_days) > 0) parts.push(`有效 ${item.expire_days} 天`)
  return parts.join(' · ')
}

// 续期档位：价格是每枚勋章自己的，列表只给概览，明细放 title
function renewOptionsOf(item) {
  const config = parseRenewConfig(item.renew)
  return config.enabled ? config.options : []
}

function renewChip(item) {
  const count = renewOptionsOf(item).length
  return count ? `续期 ${count} 档` : ''
}

function renewTip(item) {
  return renewOptionsOf(item)
    .map((option) => `${option.days} 天 / ${option.price} 积分`)
    .join(' · ')
}

onMounted(() => {
  loadCategories()
  load()
  loadAllBadges()
  loadConditions()
})
</script>

<style scoped>
/* 左侧分类栏 + 右侧勋章列表：分类多了也是左侧滚动，右侧内容不会被挤窄 */
.badge-admin {
  display: grid;
  grid-template-columns: 216px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.category-side {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  position: sticky;
  top: 12px;
}
.category-side-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 2px 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.02em;
}
.category-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.category-item:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.category-item.is-active {
  background: var(--primary-soft, rgba(79, 124, 255, 0.12));
  color: var(--primary);
  font-weight: 600;
}
/* 停用的分类仍然显示（老勋章要有归属），但视觉上弱化 */
.category-item.is-off {
  opacity: 0.55;
}
.category-item .bi {
  flex-shrink: 0;
  font-size: 13px;
}
.category-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.category-count {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}
/* 编辑按钮：鼠标悬停才出现，避免侧栏太吵 */
.category-edit {
  display: none;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text-light);
}
.category-item:hover .category-edit {
  display: inline-flex;
}
.category-edit:hover {
  background: var(--border-soft);
  color: var(--text);
}
.category-hint {
  margin: 8px 2px 0;
  padding-top: 8px;
  border-top: 1px dashed var(--border-soft);
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-light);
}
.footer-spacer {
  flex: 1;
}

@media (max-width: 900px) {
  .badge-admin {
    grid-template-columns: minmax(0, 1fr);
  }
  /* 窄屏：分类改成横向滚动的胶囊条 */
  .category-side {
    position: static;
    flex-direction: row;
    flex-wrap: nowrap;
    overflow-x: auto;
    align-items: center;
  }
  .category-side-head,
  .category-hint {
    display: none;
  }
  .category-item {
    width: auto;
    flex-shrink: 0;
    white-space: nowrap;
  }
  .category-edit {
    display: none;
  }
}

/* ---------- 后台列表页公共骨架 ----------
 * 项目里的 .list-head / .status-tab / .search-box / .form-grid 等**不是全局样式**，
 * 每个后台页面各自 scoped 一份（见 Level.vue 的 style 段），所以新页面必须自带。
 */
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.block-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.list-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.status-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.status-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: var(--control-h-sm);
  padding: 0 14px;
  font-size: 13px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  background: var(--bg-muted);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.status-tab:hover {
  color: var(--primary);
}
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
}
.filter-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-left: auto;
}
.search-box {
  position: relative;
  width: 190px;
}
.search-box .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
}
.search-input {
  width: 100%;
  height: var(--control-h-sm);
  padding: 0 12px 0 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
/* 批量选择 / 批量移动 */
.check-all {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  color: var(--text-muted);
  cursor: pointer;
  user-select: none;
}
.row-check {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  cursor: pointer;
}
.batch-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  padding: 8px 12px;
  font-size: 12px;
  border: 1px solid var(--primary);
  border-radius: var(--radius-sm);
  background: var(--accent-wash);
}
.batch-count {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--primary-deep);
}
.trash-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  font-size: 12px;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: var(--radius);
}
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.badge-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.badge-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
}
.badge-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--border);
  border-radius: 50%;
  font-size: 18px;
  overflow: hidden;
}
.badge-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.badge-main {
  flex: 1;
  min-width: 0;
}
.badge-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.badge-name {
  font-weight: 600;
  font-size: 14px;
}
.badge-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.badge-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 6px;
}
.meta-text {
  font-size: 11px;
  color: var(--text-light);
}
.cond-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 8px;
  font-size: 11px;
  color: var(--primary-deep);
  background: var(--accent-wash);
  border-radius: 999px;
}
.cond-chip.is-manual {
  color: var(--text-muted);
  background: var(--bg-muted);
}
/* 所属分类：弱于获得条件，只做归属提示 */
.cond-chip.is-cat {
  color: var(--text-muted);
  background: var(--bg-muted);
}
/* 稀有度：只改文字/底色，不给整行加光效（列表里太花） */
.cond-chip.is-rarity {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.cond-chip.rarity-rare {
  color: #2f7fe0;
  background: rgba(79, 157, 255, 0.12);
}
.cond-chip.rarity-epic {
  color: #7c5cff;
  background: rgba(160, 107, 255, 0.14);
}
.cond-chip.rarity-legendary {
  color: #b8860b;
  background: rgba(240, 180, 41, 0.18);
}
.cond-chip.is-limit {
  color: #e08b00;
  background: rgba(240, 160, 32, 0.12);
}
/* 续期档位：与「限时」区分开（续期是运营配的付费选项，不是勋章本身的限制） */
.cond-chip.is-renew {
  color: #2f8f5b;
  background: rgba(47, 143, 91, 0.12);
}
.badge-actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.badge-actions .danger {
  color: var(--danger);
}
/* 图标预览 */
.icon-input,
.color-input {
  display: flex;
  align-items: center;
  gap: 8px;
}
.icon-preview {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  border: 1px solid var(--border);
  border-radius: 50%;
  font-size: 16px;
}
.color-picker {
  width: 34px;
  height: 34px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-xs);
  background: transparent;
  cursor: pointer;
}
/* 勋章图片：左边小预览，右边地址输入，下面上传 / 附件库按钮 */
.image-field {
  display: flex;
  align-items: center;
  gap: 8px;
}
.image-preview {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-xs);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 16px;
}
.image-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.image-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
/* 勋章多选 */
.chip-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.pick-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  color: var(--text);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 999px;
  cursor: pointer;
  transition: all var(--dur) var(--ease);
}
.pick-chip.is-active {
  color: var(--primary-deep);
  border-color: var(--primary);
  background: var(--accent-wash);
}
/* 持有记录 */
.record-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 420px;
  overflow-y: auto;
}
.record-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.record-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.record-avatar.is-empty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-muted);
  color: var(--text-light);
}
.record-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.record-name {
  font-size: 13px;
  font-weight: 600;
}
.record-meta {
  font-size: 11px;
  color: var(--text-light);
}
.record-remark {
  font-size: 11px;
  color: var(--text-muted);
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 768px) {
  .list-filter {
    gap: 8px;
  }
  .filter-right {
    width: 100%;
    margin-left: 0;
  }
  .search-box {
    width: 100%;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 640px) {
  .badge-row {
    flex-wrap: wrap;
  }
  .badge-actions {
    width: 100%;
    justify-content: flex-end;
  }
}
</style>
