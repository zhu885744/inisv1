<template>
  <div class="goods-admin">
    <!-- 概览：商城统计 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">商品管理</h2>
          <p class="block-desc">积分商城：商品上下架、库存与卡密池，以及兑换订单流转</p>
        </div>
        <div class="head-actions">
          <button
            class="btn btn-ghost btn-sm"
            :disabled="loadingStats"
            title="刷新统计"
            aria-label="刷新统计"
            @click="loadStats"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </div>
      </header>

      <div class="stat-grid" :aria-busy="loadingStats">
        <div v-for="s in statCards" :key="s.label" class="stat-card">
          <span class="stat-icon" :style="{ color: s.color }"><i :class="s.icon" /></span>
          <span class="stat-body">
            <span class="stat-value">{{ loadingStats ? '···' : s.value }}</span>
            <span class="stat-label">{{ s.label }}</span>
          </span>
        </div>
      </div>
    </section>

    <div class="card card-pad">
      <!-- 视图切换 -->
      <div class="view-tabs">
        <button
          v-for="v in views"
          :key="v.key"
          type="button"
          class="status-tab"
          :class="{ 'is-active': view === v.key }"
          @click="switchView(v.key)"
        >
          <i :class="v.icon" /> {{ v.label }}
        </button>
      </div>

      <!-- ============ 商品列表 ============ -->
      <template v-if="view === 'goods'">
        <header class="list-head">
          <div>
            <h2 class="block-title">商品列表</h2>
            <p class="block-desc">
              库存不足 5 件会进入统计的「库存预警」
              <template v-if="total > 0"> · 共 {{ total }} 件</template>
            </p>
          </div>
          <div class="head-actions">
            <button class="btn btn-primary btn-sm" :disabled="trash" @click="openCreate">
              <i class="bi bi-plus-lg" /> 新建商品
            </button>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in goodsTabs"
              :key="tab.key"
              type="button"
              class="status-tab"
              :class="{ 'is-active': !trash && status === tab.key }"
              @click="switchStatus(tab.key)"
            >{{ tab.label }}</button>
            <button
              type="button"
              class="status-tab trash-tab"
              :class="{ 'is-active': trash }"
              @click="toggleTrash"
            >
              <i class="bi bi-trash3" /> 回收站
            </button>
          </div>

          <div class="filter-right">
            <FormSelect variant="pill"
              v-model="category"
              :options="categoryOptions"
              icon="bi bi-collection"
              placeholder="全部分类"
              @change="reloadGoods" />

            <div class="search-box">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="keyword"
                class="search-input"
                type="search"
                placeholder="搜索商品名…"
                aria-label="搜索商品名"
                @input="doSearch" />
            </div>

            <button class="btn btn-sm" :disabled="loading" @click="loadGoods()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <!-- 批量操作 -->
        <div v-if="selectedIds.length" class="batch-bar">
          <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 件</span>
          <div class="batch-actions">
            <template v-if="trash">
              <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
                <i class="bi bi-arrow-counterclockwise" /> 批量恢复
              </button>
              <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
                <i class="bi bi-x-octagon" /> 彻底删除
              </button>
            </template>
            <template v-else>
              <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
                <i class="bi bi-trash" /> 删除
              </button>
            </template>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
          </div>
        </div>

        <div v-if="trash" class="trash-bar">
          <span><i class="bi bi-trash3" /> 回收站内的商品不会在前台展示，可恢复或彻底删除</span>
          <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
            清空回收站
          </button>
        </div>

        <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!list.length" class="empty-row">
          <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-bag'" :text="emptyText" />
        </div>

        <template v-else>
          <div class="list-head-row">
            <label class="pick" title="全选本页">
              <FormCheckbox
                ref="selectAllRef"
                :model-value="pageAllSelected"
                :disabled="busy"
                aria-label="全选本页"
                @change="toggleSelectAll" />
            </label>
            <span class="list-head-text">本页 {{ list.length }} 件</span>
          </div>

          <ul class="goods-list">
            <li v-for="item in list" :key="item.id" class="goods-row" :class="{ 'is-selected': isSelected(item.id) }">
              <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
                <FormCheckbox
                  :model-value="isSelected(item.id)"
                  :disabled="busy"
                  :aria-label="`选择商品 ${item.title}`"
                  @change="toggleSelect(item.id)" />
              </label>

              <div class="goods-cover">
                <img v-if="item.cover" :src="item.cover" :alt="item.title" loading="lazy" />
                <i v-else class="bi bi-image" />
              </div>

              <div class="goods-main">
                <div class="goods-name-row">
                  <span class="goods-name">{{ item.title }}</span>
                  <span class="state-chip" :class="Number(item.status) === 1 ? 'is-on' : 'is-off'">
                    {{ Number(item.status) === 1 ? '上架' : '下架' }}
                  </span>
                  <span v-if="item.category" class="type-chip">{{ item.category }}</span>
                  <span class="type-chip">{{ typeLabel(item) }}</span>
                  <!-- 兑换时间窗口：只有设置了「结束时间」才显示（0 = 不限）
                       - 未到期：限时 10-20 12:00
                       - 已到期：已结束（该商品此时用户已无法兑换，见后端 BuyGoods 的窗口校验） -->
                  <span
                    v-if="Number(item.end_time) > 0"
                    class="state-chip"
                    :class="isEnded(item) ? 'is-expired' : 'is-limited'"
                    :title="endTimeTip(item)"
                  >
                    <i class="bi bi-clock-history" />
                    {{ isEnded(item) ? '已结束' : '限时' }} {{ formatTime(item.end_time).slice(5) }}
                  </span>
                </div>
                <div class="goods-meta">
                  <span class="meta-text">
                    <i :class="isFreePrice(item.price) ? 'bi bi-gift' : 'bi bi-coin'" />
                    {{ isFreePrice(item.price) ? '免费' : `${item.price} 积分` }}
                  </span>
                  <span class="meta-text" :class="{ 'is-warn': Number(item.stock) <= 5 }">
                    <i class="bi bi-box-seam" /> 库存 {{ item.stock }}
                  </span>
                  <span class="meta-text"><i class="bi bi-cart-check" /> 已兑换 {{ item.sold || 0 }}</span>
                  <span class="meta-text">#{{ item.id }}</span>
                  <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                </div>
              </div>

              <div class="goods-actions">
                <template v-if="trash">
                  <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                    <i class="bi bi-arrow-counterclockwise" />
                  </button>
                  <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                    <i class="bi bi-x-octagon" />
                  </button>
                </template>
                <template v-else>
                  <button class="btn btn-ghost btn-sm" title="编辑" aria-label="编辑" @click="openEdit(item)">
                    <i class="bi bi-pencil" />
                  </button>
                  <button
                    class="btn btn-ghost btn-sm"
                    :title="Number(item.status) === 1 ? '下架' : '上架'"
                    :aria-label="Number(item.status) === 1 ? '下架' : '上架'"
                    @click="toggleSale(item)"
                  >
                    <i :class="Number(item.status) === 1 ? 'bi bi-eye-slash' : 'bi bi-eye'" />
                  </button>
                  <button class="btn btn-ghost btn-sm danger" title="删除" aria-label="删除" :disabled="busy" @click="askRemove(item)">
                    <i class="bi bi-trash" />
                  </button>
                </template>
              </div>
            </li>
          </ul>
        </template>

        <Pagination
          v-if="!loading && total > 0"
          :current="page"
          :total="total"
          :page-size="pageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { pageSize = size; page = 1; loadGoods() }"
          @update:current="changePage"
        />
      </template>

      <!-- ============ 兑换订单 ============ -->
      <template v-else>
        <header class="list-head">
          <div>
            <h2 class="block-title">兑换订单</h2>
            <p class="block-desc">
              按状态处理订单；取消订单会自动退还积分并回滚库存
              <template v-if="orderTotal > 0"> · 共 {{ orderTotal }} 单</template>
            </p>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in orderTabs"
              :key="tab.value"
              type="button"
              class="status-tab"
              :class="{ 'is-active': orderStatus === tab.value }"
              @click="switchOrderStatus(tab.value)"
            >{{ tab.label }}</button>
          </div>

          <div class="filter-right">
            <button class="btn btn-sm" :disabled="orderLoading" @click="loadOrders()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <div v-if="orderLoading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!orders.length" class="empty-row">
          <EmptyState icon="bi bi-receipt" text="暂无兑换订单" />
        </div>

        <ul v-else class="order-list">
          <li v-for="order in orders" :key="order.id" class="order-row">
            <div class="order-main">
              <div class="order-name-row">
                <span class="order-no">{{ order.order_no }}</span>
                <span class="state-chip" :class="`is-${orderStatusKey(order.status)}`">
                  {{ orderStatusLabel(order.status) }}
                </span>
              </div>
              <div class="order-goods">
                <span class="goods-title">{{ orderGoodsTitle(order) }}</span>
                <span class="meta-text">
                  <i :class="isFreePrice(order.price) ? 'bi bi-gift' : 'bi bi-coin'" />
                  {{ isFreePrice(order.price) ? '免费' : `${order.price} 积分` }}
                </span>
              </div>
              <div class="order-meta">
                <span class="meta-text"><i class="bi bi-person" /> {{ orderUserText(order) }}</span>
                <span class="meta-text"><i class="bi bi-clock" /> {{ fromNow(order.create_time) }}</span>
                <span v-if="order.logistics" class="meta-text" :title="order.logistics">
                  <i class="bi bi-truck" /> {{ order.logistics }}
                </span>
              </div>
              <!-- 收货信息（实物商品下单时填写，后端存 JSON 并在 result.address 解析） -->
              <div v-if="orderAddress(order)" class="order-address">
                <i class="bi bi-geo-alt" />
                <span class="address-name">{{ orderAddress(order).name }}</span>
                <span class="address-phone">{{ orderAddress(order).phone }}</span>
                <span class="address-detail" :title="orderAddress(order).address">
                  {{ orderAddress(order).address }}
                </span>
                <button
                  class="btn btn-ghost btn-sm"
                  title="复制收货信息"
                  aria-label="复制收货信息"
                  @click="copyAddress(order)"
                >
                  <i class="bi bi-clipboard" />
                </button>
              </div>
            </div>

            <div class="order-actions">
              <button
                v-if="Number(order.status) === 0"
                class="btn btn-sm btn-primary"
                :disabled="busy"
                @click="openShip(order)"
              >
                <i class="bi bi-truck" /> 发货
              </button>
              <button
                v-if="Number(order.status) === 1"
                class="btn btn-sm"
                :disabled="busy"
                @click="askOrderStatus(order, 2, '确认已完成')"
              >
                <i class="bi bi-check2" /> 完成
              </button>
              <button
                v-if="Number(order.status) === 0 || Number(order.status) === 1"
                class="btn btn-sm btn-danger"
                :disabled="busy"
                @click="askOrderStatus(order, 3, '取消订单')"
              >
                <i class="bi bi-x" /> 取消
              </button>
              <button class="btn btn-ghost btn-sm" title="查看发货内容" aria-label="查看发货内容" @click="viewDeliver(order)">
                <i class="bi bi-eye" />
              </button>
            </div>
          </li>
        </ul>

        <Pagination
          v-if="!orderLoading && orderTotal > 0"
          :current="orderPage"
          :total="orderTotal"
          :page-size="orderPageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { orderPageSize = size; orderPage = 1; loadOrders() }"
          @update:current="changeOrderPage"
        />
      </template>
    </div>

    <!-- 新增 / 编辑商品 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.id ? `编辑商品 #${edit.id}` : '新建商品'"
      :icon="edit.id ? 'bi bi-pencil-square' : 'bi bi-plus-circle'"
      :loading="edit.loading"
      confirm-text="保存"
      width="640px"
      @confirm="saveGoodsItem"
    >
      <section class="form-section">
        <h4 class="section-title">基本资料</h4>

        <div class="form-item">
          <label class="form-label">商品名称</label>
          <FormInput v-model="edit.title" type="text" maxlength="128" placeholder="最多 128 字" />
        </div>

        <div class="form-item">
          <label class="form-label">商品描述</label>
          <FormTextarea v-model="edit.description" rows="2" placeholder="可选，展示在商品详情" />
        </div>

        <div class="form-item">
          <label class="form-label">商品封面</label>
          <div class="cover-row">
            <div class="cover-box">
              <img v-if="edit.cover" :src="edit.cover" alt="封面预览" loading="lazy" />
              <div v-else class="cover-empty" @click="pickCover">
                <i class="bi bi-image" />
                <span>上传</span>
              </div>
              <button v-if="edit.cover" type="button" class="cover-del" title="移除封面" @click="edit.cover = ''">
                <i class="bi bi-x" />
              </button>
            </div>
            <div class="cover-fields">
              <FormInput v-model="edit.cover" type="text" placeholder="粘贴图片链接，或点击下方按钮上传" />
              <div class="cover-actions">
                <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickCover">
                  <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
                </button>
                <button class="btn btn-sm" type="button" title="从附件库选择" @click="showLibrary = true">
                  <i class="bi bi-folder2-open" /> 附件库
                </button>
                <button v-if="edit.cover" class="btn btn-sm btn-ghost" type="button" @click="edit.cover = ''">移除</button>
              </div>
            </div>
            <input ref="coverRef" type="file" accept="image/*" hidden @change="onCoverChange" />

            <!-- 附件库：选一张已上传的图片作为商品封面 -->
            <AttachmentLibrary
              v-model:visible="showLibrary"
              title="选择商品封面"
              accept="image"
              @select="onLibrarySelect"
            />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">分类</label>
            <FormInput v-model="edit.category" type="text" maxlength="32" placeholder="可选，如「虚拟道具」" />
          </div>
          <div class="form-item">
            <label class="form-label">排序权重</label>
            <FormInput v-model="edit.sort" type="number" step="1" placeholder="越大越靠前" />
          </div>
        </div>
      </section>

      <section class="form-section">
        <h4 class="section-title">兑换规则</h4>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">积分价格</label>
            <FormInput v-model="edit.price" type="number" min="0" step="1" placeholder="兑换所需积分" />
          </div>
          <div class="form-item">
            <label class="form-label">库存</label>
            <FormInput v-model="edit.stock" type="number" min="0" step="1" placeholder="0 表示售罄" />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">每人限购</label>
            <FormInput v-model="edit.limit_per_user" type="number" min="0" step="1" placeholder="0 表示不限购" />
          </div>
          <div class="form-item">
            <label class="form-label">最低经验</label>
            <FormInput v-model="edit.min_exp" type="number" min="0" step="1" placeholder="0 表示不限" />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">上架状态</label>
            <FormSelect v-model="edit.status" :options="SALE_OPTIONS" />
          </div>
          <div class="form-item">
            <label class="form-label">商品类型</label>
            <FormSelect v-model="edit.type" :options="TYPE_OPTIONS" />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">开始时间</label>
            <FormInput v-model="edit.startAt" type="datetime-local" />
            <p class="form-hint">留空表示立即开始</p>
          </div>
          <div class="form-item">
            <label class="form-label">结束时间</label>
            <FormInput v-model="edit.endAt" type="datetime-local" />
            <p class="form-hint">留空表示长期有效</p>
          </div>
        </div>
      </section>

      <section class="form-section">
        <h4 class="section-title">发货设置</h4>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">发货方式</label>
            <FormSelect v-model="edit.deliver_type" :options="deliverOptions" />
            <p class="form-hint">仅虚拟商品需要；实物商品下单时填写收货地址</p>
          </div>
          <div v-if="edit.deliver_type === 'card'" class="form-item">
            <label class="form-label">卡密剩余</label>
            <p class="form-hint">{{ cardCount }} 个</p>
          </div>
        </div>

        <div v-if="edit.deliver_type === 'text'" class="form-item">
          <label class="form-label">发货内容</label>
          <FormTextarea v-model="edit.deliver_content" rows="3" placeholder="所有订单发放同一段文本" />
        </div>

        <div v-else-if="edit.deliver_type === 'card'" class="form-item">
          <label class="form-label">卡密池</label>
          <FormTextarea v-model="edit.cardsText" rows="4" placeholder="一行一个卡密，发货时随机抽取并移除" />
          <p class="form-hint">仅管理员可见；已发出的卡密会由后端自动移出卡密池</p>
        </div>

      </section>
    </AdminFormDialog>

    <!-- 订单发货 -->
    <AdminFormDialog
      v-model:visible="ship.visible"
      :title="`订单发货：${ship.orderNo}`"
      icon="bi bi-truck"
      :loading="ship.loading"
      confirm-text="确认发货"
      @confirm="submitShip"
    >
      <!-- 实物商品：发货前核对收货信息 -->
      <div v-if="ship.address" class="ship-address">
        <div class="ship-address-head">
          <span class="ship-address-title"><i class="bi bi-geo-alt" /> 收货信息</span>
          <button class="btn btn-ghost btn-sm" @click="copyAddress(ship.address)">复制</button>
        </div>
        <div class="ship-address-line">
          <span>{{ ship.address.name }}</span>
          <span>{{ ship.address.phone }}</span>
        </div>
        <div class="ship-address-detail">{{ ship.address.address }}</div>
      </div>
      <div class="form-item">
        <label class="form-label">物流信息</label>
        <FormTextarea v-model="ship.logistics" rows="3" placeholder="快递公司 + 运单号，虚拟商品可留空" />
      </div>
    </AdminFormDialog>

    <!-- 发货内容 -->
    <AdminFormDialog
      v-model:visible="deliver.visible"
      :title="`发货内容：${deliver.orderNo}`"
      icon="bi bi-eye"
      :loading="false"
      confirm-text="关闭"
      @confirm="deliver.visible = false"
    >
      <p class="dialog-text">{{ deliver.content || '该订单暂无发货内容' }}</p>
    </AdminFormDialog>

    <!-- 操作确认 -->
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
/**
 * 商品管理（/admin/goods）
 *
 * 两个视图：商品列表（CRUD + 回收站）与兑换订单（状态流转）。
 *
 * 后端约束（app/api/controller/goods.go）：
 * - goods/all：管理员可见全部（含下架），支持 status / category / keyword / onlyTrashed；
 *   普通用户只看到上架商品，且 cards / deliver_content 会被剔除，故后台列表 field 要显式带上它们；
 * - goods/save 自动分流 create / update；cards 必须存 JSON 数组字符串（后端 utils.Json.Decode 解析），
 *   传数组会被 processFieldValue 用逗号拼接，导致卡密池无法解析；
 * - orders-all / order-status 仅管理员：status 0待发货 1已发货 2已完成 3已取消，
 *   置为 3 会自动退还积分并回滚库存；
 * - 商品删除为软删除，可在回收站恢复或彻底删除。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listGoodsAdmin,
  saveGoods,
  removeGoods,
  forceDeleteGoods,
  restoreGoods,
  clearGoodsRecycle,
  getGoodsCategories,
  getGoodsStats,
  listOrdersAdmin,
  setOrderStatus
} from '@/api/goods'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { fromNow, formatTime, toLocalInput, fromLocalInput } from '@/utils/time'
import { debounce, copyText } from '@/utils/helper'
import { toast } from '@/utils/toast'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'

// 商品与兑换订单是两个列表，各自记住自己的每页条数
const pageSize = ref(PAGE_SIZE_DEFAULT)
const orderPageSize = ref(PAGE_SIZE_DEFAULT)

// 商品状态：0 下架 / 1 上架
const SALE_OPTIONS = [
  { value: 1, label: '上架' },
  { value: 0, label: '下架' }
]

// 商品类型
const TYPE_OPTIONS = [
  { value: 'virtual', label: '虚拟商品' },
  { value: 'physical', label: '实物商品' }
]

// 发货方式（仅虚拟商品）
// 说明：装扮商城已独立（装扮自带价格 / 库存 / 经验门槛，兑换走 decoration/buy），
// 商品不再提供「装扮发货」，避免同一件装扮出现两个售卖入口。
const DELIVER_OPTIONS = [
  { value: '', label: '无需发货' },
  { value: 'text', label: '文本发货' },
  { value: 'card', label: '卡密发货' }
]

const goodsTabs = [
  { key: 'all', label: '全部' },
  { key: 'on', label: '上架' },
  { key: 'off', label: '下架' }
]

// 订单状态：与后端 OrderStatus* 常量一致
const orderTabs = [
  { value: '', label: '全部' },
  { value: '0', label: '待发货' },
  { value: '1', label: '已发货' },
  { value: '2', label: '已完成' },
  { value: '3', label: '已取消' }
]

const views = [
  { key: 'goods', label: '商品列表', icon: 'bi bi-bag' },
  { key: 'orders', label: '兑换订单', icon: 'bi bi-receipt' }
]

// ===== 统计 =====
const stats = ref(null)
const loadingStats = ref(false)

const statCards = computed(() => {
  const goods = stats.value?.goods || {}
  const order = stats.value?.order || {}
  const integral = stats.value?.integral || {}
  return [
    { label: '商品总数', value: goods.total ?? 0, icon: 'bi bi-bag', color: 'var(--primary)' },
    { label: '上架中', value: goods.on ?? 0, icon: 'bi bi-eye', color: 'var(--success)' },
    { label: '已下架', value: goods.off ?? 0, icon: 'bi bi-eye-slash', color: 'var(--text-muted)' },
    { label: '库存预警', value: goods.stock_warn ?? 0, icon: 'bi bi-exclamation-triangle', color: 'var(--warning)' },
    { label: '订单总数', value: order.total ?? 0, icon: 'bi bi-receipt', color: 'var(--primary)' },
    { label: '待发货', value: order.pending ?? 0, icon: 'bi bi-truck', color: 'var(--warning)' },
    { label: '已完成', value: order.completed ?? 0, icon: 'bi bi-check2-circle', color: 'var(--success)' },
    { label: '消耗积分', value: integral.net ?? 0, icon: 'bi bi-coin', color: 'var(--primary)' }
  ]
})

// ===== 商品列表 =====
const view = ref('goods')
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const trash = ref(false)
const status = ref('all')
const category = ref('')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])
const categories = ref([])

// ===== 订单 =====
const orders = ref([])
const orderTotal = ref(0)
const orderPage = ref(1)
const orderLoading = ref(false)
const orderStatus = ref('')

// ===== 弹窗 =====
const coverRef = ref(null)
const uploading = ref(false)
const showLibrary = ref(false)

// 从附件库选封面：直接复用已上传图片的地址
function onLibrarySelect(urls = []) {
  const url = urls[0]
  showLibrary.value = false
  if (url) edit.cover = url
}

const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  title: '',
  description: '',
  cover: '',
  category: '',
  sort: 0,
  price: 0,
  stock: 0,
  limit_per_user: 0,
  min_exp: 0,
  status: 1,
  type: 'virtual',
  startAt: '',
  endAt: '',
  deliver_type: '',
  deliver_content: '',
  cardsText: ''
})

const ship = reactive({
  visible: false,
  loading: false,
  id: 0,
  orderNo: '',
  logistics: '',
  // 实物商品的收货信息，发货前用于核对
  address: null
})

const deliver = reactive({
  visible: false,
  orderNo: '',
  content: ''
})

const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

// 虚拟商品才需要发货方式
const deliverOptions = computed(() =>
  edit.type === 'virtual' ? DELIVER_OPTIONS : [{ value: '', label: '实物商品无需发货' }]
)

const categoryOptions = computed(() => [
  { value: '', label: '全部分类' },
  ...categories.value.map((c) => ({ value: c.category, label: `${c.category}（${c.count}）` }))
])

// 卡密池行数（用于提示剩余数量）
const cardCount = computed(() =>
  edit.cardsText
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean).length
)

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的商品'
  if (trash.value) return '回收站是空的'
  return '还没有商品，先新建一个吧'
})

// ---------- 展示辅助 ----------
function typeLabel(item) {
  return item?.type === 'physical' ? '实物' : '虚拟'
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

// 是否已过兑换结束时间（口径与后端一致：end_time = 0 表示不限时间）
function isEnded(item) {
  const end = Number(item?.end_time) || 0
  return end > 0 && Math.floor(Date.now() / 1000) > end
}

// 结束时间标签的悬停提示：完整时间 + 剩余时长（已结束则提示用户已无法兑换）
function endTimeTip(item) {
  const end = Number(item?.end_time) || 0
  if (end <= 0) return ''
  const full = formatTime(end)
  if (isEnded(item)) return `兑换结束时间：${full}（已结束，用户无法再兑换）`
  const left = end - Math.floor(Date.now() / 1000)
  const remain = left >= 86400 ? `${Math.ceil(left / 86400)} 天` : `${Math.max(1, Math.ceil(left / 3600))} 小时`
  return `兑换结束时间：${full}（还剩约 ${remain}）`
}

function orderStatusLabel(value) {
  return { 0: '待发货', 1: '已发货', 2: '已完成', 3: '已取消' }[Number(value)] || '未知'
}

function orderStatusKey(value) {
  return { 0: 'pending', 1: 'shipped', 2: 'done', 3: 'canceled' }[Number(value)] || 'pending'
}

// ---------- 数据加载 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const res = await getGoodsStats()
    stats.value = res?.data || null
  } catch {
    stats.value = null
  } finally {
    loadingStats.value = false
  }
}

async function loadCategories() {
  try {
    const res = await getGoodsCategories()
    const data = Array.isArray(res?.data) ? res.data : []
    // 首个元素是「全部」（category 为空），过滤掉
    categories.value = data.filter((c) => c?.category)
  } catch {
    categories.value = []
  }
}

async function loadGoods() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (trash.value) {
      params.onlyTrashed = true
    } else if (status.value !== 'all') {
      params.status = status.value === 'on' ? 1 : 0
    }
    if (category.value) params.category = category.value
    const kw = keyword.value.trim()
    if (kw) params.keyword = kw
    const res = await listGoodsAdmin(params)
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function reloadGoods() {
  page.value = 1
  loadGoods()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reloadGoods()
}, 350)

async function switchStatus(key) {
  if (!trash.value && status.value === key) return
  trash.value = false
  status.value = key
  reloadGoods()
}

async function toggleTrash() {
  trash.value = !trash.value
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  await loadGoods()
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  loadGoods()
}

async function loadOrders() {
  orderLoading.value = true
  try {
    const params = { page: orderPage.value, limit: orderPageSize.value }
    if (orderStatus.value !== '') params.status = Number(orderStatus.value)
    const res = await listOrdersAdmin(params)
    orders.value = res.data?.data || []
    orderTotal.value = res.data?.count || 0
  } catch {
    orders.value = []
    orderTotal.value = 0
  } finally {
    orderLoading.value = false
  }
}

async function switchOrderStatus(value) {
  if (orderStatus.value === value) return
  orderStatus.value = value
  orderPage.value = 1
  await loadOrders()
}

function changeOrderPage(p) {
  if (p < 1 || p === orderPage.value) return
  orderPage.value = p
  loadOrders()
}

async function switchView(key) {
  if (view.value === key) return
  view.value = key
  if (key === 'orders' && !orders.value.length) await loadOrders()
}

// ---------- 选择 ----------
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id))
)
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)

const selectAllRef = ref(null)

watchEffect(() => {
  if (selectAllRef.value) selectAllRef.value.setIndeterminate(someSelected.value)
})

function isSelected(id) {
  return selectedSet.value.has(id)
}

function toggleSelect(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleSelectAll() {
  selectedIds.value = pageAllSelected.value ? [] : list.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 封面上传 ----------
function pickCover() {
  coverRef.value?.click()
}

async function onCoverChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')
    edit.cover = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// ---------- 新增 / 编辑商品 ----------
function resetEdit() {
  edit.id = 0
  edit.title = ''
  edit.description = ''
  edit.cover = ''
  edit.category = ''
  edit.sort = 0
  edit.price = 0
  edit.stock = 0
  edit.limit_per_user = 0
  edit.min_exp = 0
  edit.status = 1
  edit.type = 'virtual'
  edit.startAt = ''
  edit.endAt = ''
  edit.deliver_type = ''
  edit.deliver_content = ''
  edit.cardsText = ''
}

function openCreate() {
  resetEdit()
  edit.visible = true
}

// 卡密池：后端存 JSON 数组字符串
function cardsToText(raw) {
  if (!raw) return ''
  let list = []
  if (Array.isArray(raw)) {
    list = raw
  } else if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      list = Array.isArray(parsed) ? parsed : []
    } catch {
      list = raw.split(/[,\n]/)
    }
  }
  return list.map((s) => String(s).trim()).filter(Boolean).join('\n')
}

function openEdit(item) {
  edit.id = Number(item.id)
  edit.title = item.title || ''
  edit.description = item.description || ''
  edit.cover = item.cover || ''
  edit.category = item.category || ''
  edit.sort = Number(item.sort || 0)
  edit.price = Number(item.price || 0)
  edit.stock = Number(item.stock || 0)
  edit.limit_per_user = Number(item.limit_per_user || 0)
  edit.min_exp = Number(item.min_exp || 0)
  edit.status = Number(item.status) === 0 ? 0 : 1
  edit.type = item.type === 'physical' ? 'physical' : 'virtual'
  edit.startAt = toLocalInput(item.start_time)
  edit.endAt = toLocalInput(item.end_time)
  edit.deliver_type = edit.type === 'virtual' ? item.deliver_type || '' : ''
  edit.deliver_content = item.deliver_content || ''
  edit.cardsText = cardsToText(item.cards)
  edit.visible = true
}

// 0 积分商品按「免费」展示，列表与订单里不再出现「0 积分」
const isFreePrice = (price) => Number(price) <= 0

// 非负整数校验（允许 0）
function intOrNull(raw, allowZero = true) {
  const text = String(raw ?? '').trim()
  if (text === '') return allowZero ? 0 : null
  const num = Number(text)
  if (!Number.isInteger(num) || num < 0) return null
  return num
}

async function saveGoodsItem() {
  const title = edit.title.trim()
  if (!title) {
    toast.warning('请输入商品名称')
    return
  }
  const price = intOrNull(edit.price, false)
  if (price === null) {
    toast.warning('积分价格必须为不小于 0 的整数')
    return
  }
  const stock = intOrNull(edit.stock)
  const limit = intOrNull(edit.limit_per_user)
  const minExp = intOrNull(edit.min_exp)
  const sort = intOrNull(edit.sort)
  if ([stock, limit, minExp, sort].some((v) => v === null)) {
    toast.warning('库存 / 限购 / 经验 / 排序必须为不小于 0 的整数')
    return
  }

  const startTime = fromLocalInput(edit.startAt)
  const endTime = fromLocalInput(edit.endAt)
  if (startTime > 0 && endTime > 0 && endTime <= startTime) {
    toast.warning('结束时间必须晚于开始时间')
    return
  }

  const cards = edit.cardsText
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)

  const payload = {
    title,
    description: edit.description.trim(),
    cover: edit.cover.trim(),
    category: edit.category.trim(),
    sort,
    price,
    stock,
    limit_per_user: limit,
    min_exp: minExp,
    status: Number(edit.status),
    type: edit.type,
    // 实物商品不需要发货方式
    deliver_type: edit.type === 'virtual' ? edit.deliver_type : '',
    deliver_content: edit.deliver_type === 'text' ? edit.deliver_content.trim() : '',
    // 后端按 JSON 数组解析，必须传字符串
    cards: edit.deliver_type === 'card' ? JSON.stringify(cards) : '',
    start_time: startTime,
    end_time: endTime
  }
  if (edit.id) payload.id = edit.id

  edit.loading = true
  try {
    await saveGoods(payload)
    toast.success(edit.id ? '修改已保存' : '商品已创建')
    edit.visible = false
    await Promise.all([loadGoods(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    edit.loading = false
  }
}

// ---------- 上下架 / 删除 ----------
async function toggleSale(item) {
  const next = Number(item.status) === 1 ? 0 : 1
  busy.value = true
  try {
    await saveGoods({
      id: Number(item.id),
      status: next,
      title: item.title || ''
    })
    toast.success(next === 1 ? '已上架' : '已下架')
    await Promise.all([loadGoods(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function openConfirm({ title, message, confirmText = '确定', danger = false, action }) {
  confirm.title = title
  confirm.message = message
  confirm.confirmText = confirmText
  confirm.danger = danger
  confirm.action = action
  confirm.visible = true
}

async function runConfirm() {
  if (typeof confirm.action !== 'function') {
    confirm.visible = false
    return
  }
  confirm.loading = true
  try {
    const ok = await confirm.action()
    if (ok !== false) confirm.visible = false
  } catch {
    // 请求失败时保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    confirm.loading = false
  }
}

function askRemove(item) {
  openConfirm({
    title: '删除商品',
    message: `确定删除商品「${item.title}」吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeGoods([item.id])
      toast.success('已移入回收站')
      await Promise.all([loadGoods(), loadStats()])
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreGoods([item.id])
    toast.success('已恢复')
    await Promise.all([loadGoods(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除商品「${item.title}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteGoods([item.id])
      toast.success('已彻底删除')
      await Promise.all([loadGoods(), loadStats()])
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 件商品吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeGoods(ids)
      toast.success(`已删除 ${ids.length} 件`)
      clearSelection()
      await Promise.all([loadGoods(), loadStats()])
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreGoods(ids)
    toast.success(`已恢复 ${ids.length} 件`)
    clearSelection()
    await Promise.all([loadGoods(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 件商品吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteGoods(ids)
      toast.success(`已彻底删除 ${ids.length} 件`)
      clearSelection()
      await Promise.all([loadGoods(), loadStats()])
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有商品将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearGoodsRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await Promise.all([loadGoods(), loadStats()])
    }
  })
}

// ---------- 订单流转 ----------

// 收货信息归一化为 { name, phone, address }，字段全空视为没有收货信息（虚拟商品）
function normalizeAddress(raw) {
  if (!raw || typeof raw !== 'object') return null
  const name = raw.name || ''
  const phone = raw.phone || ''
  const address = raw.address || ''
  if (!name && !phone && !address) return null
  return { name, phone, address }
}

// 订单的收货信息：实物商品下单时填写，后端存为 JSON 字符串并在 result.address 里解析；
// result 缺失时回退到原始 address 字符串自行解析
function orderAddress(order) {
  let raw = order?.result?.address
  if (raw == null && typeof order?.address === 'string' && order.address.trim()) {
    try {
      raw = JSON.parse(order.address)
    } catch {
      raw = null
    }
  }
  return normalizeAddress(raw)
}

function orderUserText(order) {
  const nickname = order?.result?.user?.nickname
  return nickname ? `${nickname} UID:${order.uid}` : `用户 UID:${order.uid}`
}

// 商品标题：与用户端「我的订单」口径一致 ——
// goods_title 是下单快照（早期订单没有该字段，为空），result.goods.title 是实时商品信息，
// 两者都没有（商品已被彻底删除且无快照）才退回商品 ID
function orderGoodsTitle(order) {
  return order?.goods_title
    || order?.result?.goods?.title
    || `商品 #${order?.goods_id}`
}

// target 既可以是订单对象，也可以是已解析好的地址对象（发货弹窗里复用）
async function copyAddress(target) {
  const a = normalizeAddress(target?.result?.address ?? target)
  if (!a) return
  const text = [a.name, a.phone, a.address].filter(Boolean).join(' ')
  const ok = await copyText(text)
  if (ok) toast.success('收货信息已复制')
  else toast.error('复制失败，请手动复制')
}

function openShip(order) {
  ship.id = Number(order.id)
  ship.orderNo = order.order_no || `#${order.id}`
  ship.logistics = ''
  ship.address = orderAddress(order)
  ship.visible = true
}

async function submitShip() {
  ship.loading = true
  try {
    await setOrderStatus({
      id: ship.id,
      status: 1,
      logistics: ship.logistics.trim()
    })
    toast.success('已发货')
    ship.visible = false
    await Promise.all([loadOrders(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    ship.loading = false
  }
}

function askOrderStatus(order, value, label) {
  const extra = Number(value) === 3 ? '取消后积分将原路退还并回滚库存。' : ''
  openConfirm({
    title: label,
    message: `确定${label}订单「${order.order_no || '#' + order.id}」吗？${extra}`,
    confirmText: label,
    danger: Number(value) === 3,
    action: async () => {
      await setOrderStatus({ id: Number(order.id), status: Number(value) })
      toast.success('操作成功')
      await Promise.all([loadOrders(), loadStats()])
    }
  })
}

function viewDeliver(order) {
  deliver.orderNo = order.order_no || `#${order.id}`
  deliver.content = order.deliver_content || ''
  deliver.visible = true
}

onMounted(() => {
  loadStats()
  loadCategories()
  loadGoods()
})
</script>

<style scoped>
.goods-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览 ---------- */
.panel-head {
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
}

.stat-grid {
  display: grid;
  /* 8 张卡片用自适应列数，窄屏自动换行 */
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.stat-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--bg-card);
  font-size: 18px;
}
.stat-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.stat-value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 视图切换 ---------- */
.view-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.view-tabs .status-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  /* 与其它筛选控件等高 */
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
.view-tabs .status-tab:hover {
  color: var(--primary);
}
.view-tabs .status-tab.is-active {
  background: var(--primary);
  color: #fff;
}

/* ---------- 列表头部 ---------- */
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin: 16px 0;
}

/* ---------- 工具栏 ---------- */
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
  /* 与下拉 / 搜索框 / 按钮等高 */
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
  width: 170px;
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

/* ---------- 批量 / 回收站 ---------- */
.batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--accent-wash);
  border: 1px solid var(--accent-soft);
  border-radius: var(--radius);
}
.batch-count {
  font-size: 13px;
  color: var(--text-soft);
}
.batch-count strong {
  color: var(--primary-deep);
}
.batch-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
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

/* ---------- 列表 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}

.list-head-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 4px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}

.goods-list,
.order-list {
  display: flex;
  flex-direction: column;
}
.goods-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.goods-row:last-child {
  border-bottom: none;
}
.goods-row.is-selected {
  background: var(--accent-wash);
}

.pick {
  display: flex;
  align-items: center;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.goods-cover {
  width: 56px;
  height: 56px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 20px;
}
.goods-cover img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.goods-main {
  flex: 1;
  min-width: 0;
}
.goods-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.goods-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.state-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.state-chip.is-on {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.state-chip.is-off {
  color: var(--text-muted);
  background: var(--bg-muted);
}
/* 兑换时间窗口标签：限时（未到期）/ 已结束 */
.state-chip.is-limited {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.state-chip.is-expired {
  color: var(--danger);
  background: rgba(220, 38, 38, 0.12);
}
.type-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.goods-meta,
.order-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.meta-text.is-warn {
  color: var(--warning);
}

.goods-actions,
.order-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.goods-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.goods-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 订单 ---------- */
.order-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
}
.order-row:last-child {
  border-bottom: none;
}
.order-main {
  flex: 1;
  min-width: 0;
}
.order-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.order-no {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.state-chip.is-pending {
  color: var(--warning);
  background: var(--gold-wash);
}
.state-chip.is-shipped {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.state-chip.is-done {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.state-chip.is-canceled {
  color: var(--text-muted);
  background: var(--bg-muted);
}
.order-goods {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 4px;
  font-size: 13px;
  color: var(--text-soft);
}
.goods-title {
  font-weight: 500;
}
/* 收货信息（实物商品） */
.order-address {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.order-address > .bi {
  color: var(--primary);
}
.order-address .address-name {
  font-weight: 500;
  color: var(--text);
}
.order-address .address-phone {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
.order-address .address-detail {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-muted);
}

.ship-address {
  margin-bottom: 16px;
  padding: 12px 14px;
  background: var(--bg-muted);
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
}
.ship-address-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}
.ship-address-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.ship-address-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 13px;
  color: var(--text-soft);
}
.ship-address-detail {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
  word-break: break-all;
}

/* ---------- 弹窗 ---------- */
.form-section + .form-section {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px dashed var(--border-soft);
}
.section-title {
  margin: 0 0 12px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.06em;
  color: var(--text-muted);
}
.form-section .form-item {
  margin-bottom: 14px;
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
.dialog-text {
  margin: 0;
  padding: 12px;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--bg-muted);
  border-radius: var(--radius);
}

.cover-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.cover-box {
  position: relative;
  width: 96px;
  height: 72px;
  flex-shrink: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
}
.cover-box img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.cover-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  font-size: 11px;
  color: var(--text-light);
  cursor: pointer;
  transition: color 0.2s;
}
.cover-empty:hover {
  color: var(--primary);
}
.cover-del {
  position: absolute;
  right: 0;
  top: 0;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  line-height: 1;
  cursor: pointer;
}
.cover-fields {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cover-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

@media (max-width: 768px) {
  .list-filter {
    gap: 8px;
  }
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .goods-row,
  .order-row {
    flex-wrap: wrap;
  }
  .goods-main,
  .order-main {
    flex: 1 1 60%;
  }
  .goods-actions,
  .order-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
