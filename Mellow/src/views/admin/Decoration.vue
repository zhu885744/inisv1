<template>
  <div class="decoration-admin">
    <!-- 概览 -->
    <div class="stat-row">
      <div class="stat-card card card-pad">
        <span class="stat-label"><i class="bi bi-palette" /> 装扮总数</span>
        <span class="stat-value">{{ stats.total }}</span>
        <span class="stat-sub">其中上架 {{ stats.on_sale }} 个</span>
      </div>
      <div class="stat-card card card-pad">
        <span class="stat-label"><i class="bi bi-person-badge" /> 头像框</span>
        <span class="stat-value">{{ stats.by_type.avatar_frame || 0 }}</span>
        <span class="stat-sub">展示在头像外圈</span>
      </div>
      <div class="stat-card card card-pad">
        <span class="stat-label"><i class="bi bi-award" /> 头衔</span>
        <span class="stat-value">{{ stats.by_type.title || 0 }}</span>
        <span class="stat-sub">展示在昵称旁</span>
      </div>
      <div class="stat-card card card-pad">
        <span class="stat-label"><i class="bi bi-box-seam" /> 用户持有</span>
        <span class="stat-value">{{ stats.owned }}</span>
        <span class="stat-sub">累计发放 / 兑换次数</span>
      </div>
    </div>

    <div class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">装扮管理</h2>
          <p class="block-desc">
            头像框 / 头衔的创建、上下架与批量发放
            <template v-if="total > 0"> · 共 {{ total }} 个</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="busy" @click="reload()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
          <button class="btn btn-primary btn-sm" :disabled="trash" @click="openAdd()">
            <i class="bi bi-plus-lg" /> 新建装扮
          </button>
        </div>
      </header>

      <!-- 视图切换：装扮列表 / 发放回收 -->
      <div class="view-tabs">
        <button type="button" class="view-tab" :class="{ 'is-active': view === 'list' }" @click="view = 'list'">
          <i class="bi bi-palette" /> 装扮列表
        </button>
        <button type="button" class="view-tab" :class="{ 'is-active': view === 'grant' }" @click="switchGrantView()">
          <i class="bi bi-gift" /> 发放 / 回收
        </button>
      </div>

      <!-- ===== 装扮列表 ===== -->
      <template v-if="view === 'list'">
        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in STATUS_TABS"
              :key="tab.key"
              type="button"
              class="status-tab"
              :class="{ 'is-active': statusFilter === tab.key, 'trash-tab': tab.key === 'trash' }"
              @click="switchStatus(tab.key)"
            >
              <i v-if="tab.key === 'trash'" class="bi bi-trash3" /> {{ tab.label }}
            </button>
          </div>

          <div class="filter-right">
            <div class="type-chips">
              <button
                v-for="t in typeChips"
                :key="t.type"
                type="button"
                class="chip"
                :class="{ 'is-active': typeFilter === t.type }"
                @click="switchType(t.type)"
              >
                <i :class="t.icon" aria-hidden="true" /> {{ t.name }}
              </button>
            </div>
            <!-- 价格筛选：全部 / 免费 / 付费 -->
            <div class="price-seg" role="group" aria-label="按价格筛选">
              <button
                v-for="p in PRICE_TABS"
                :key="p.key || 'all'"
                type="button"
                class="price-seg-item"
                :class="{ 'is-active': priceFilter === p.key }"
                @click="switchPrice(p.key)"
              >
                {{ p.label }}
              </button>
            </div>
            <div class="search-box">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="keyword"
                class="search-input"
                type="search"
                placeholder="搜索装扮名称…"
                aria-label="搜索装扮名称"
                @input="doSearch" />
            </div>
          </div>
        </div>

        <div v-if="trash" class="trash-bar">
          <span><i class="bi bi-trash3" /> 回收站内的装扮不在前台展示，关联商品已自动下架</span>
          <button class="btn btn-sm btn-danger" :disabled="busy || !list.length" @click="askClearRecycle()">
            清空回收站
          </button>
        </div>

        <div v-if="selectedIds.length" class="batch-bar">
          <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 个</span>
          <div class="batch-actions">
            <template v-if="trash">
              <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
                <i class="bi bi-arrow-counterclockwise" /> 批量恢复
              </button>
              <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
                <i class="bi bi-x-octagon" /> 批量彻底删除
              </button>
            </template>
            <template v-else>
              <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
                <i class="bi bi-trash" /> 批量删除
              </button>
            </template>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection()">取消选择</button>
          </div>
        </div>

        <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!list.length" class="empty-row">
          <EmptyState
            :icon="trash ? 'bi bi-trash3' : 'bi bi-palette'"
            :text="trash ? '回收站是空的' : '还没有装扮，点击右上角「新建装扮」开始'"
          />
        </div>

        <template v-else>
          <div class="list-head-row">
            <label class="pick" title="全选本页">
              <FormCheckbox
                :model-value="pageAllSelected"
                :disabled="busy"
                aria-label="全选本页"
                @change="toggleSelectAll($event)" />
            </label>
            <span class="list-head-text">本页 {{ list.length }} 个 · 共 {{ total }} 个</span>
          </div>

          <div class="decoration-grid">
            <div
              v-for="item in list"
              :key="item.id"
              class="decoration-card"
              :class="{ 'is-selected': isSelected(item.id), 'is-off': Number(item.status) !== 1 }"
            >
              <label class="pick card-pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
                <FormCheckbox :model-value="isSelected(item.id)" @change="toggleSelect(item.id)" />
              </label>

              <div class="decoration-preview" :class="{ 'is-title': item.type === 'title' }">
                <img
                  v-if="item.type === 'avatar_frame'"
                  :src="item.preview || item.payload?.image"
                  class="decoration-frame"
                  :alt="item.name"
                  loading="lazy"
                />
                <!-- 图片头衔只显示图片；「纯样式模板」（无文字无图片）才用装扮名占位预览 -->
                <TitleBadge
                  v-else
                  :payload="item.payload"
                  :text="item.payload?.text || (item.payload?.image ? '' : item.name)"
                  :size="'lg'"
                />
              </div>

              <div class="decoration-info">
                <div class="decoration-name">
                  {{ item.name }}
                  <span class="decoration-tag">{{ typeName(item.type) }}</span>
                  <span v-if="Number(item.is_default) === 1" class="decoration-tag is-default">系统默认</span>
                </div>
                <div class="decoration-desc">{{ item.description || '暂无描述' }}</div>
                <div class="decoration-meta">
                  <span class="decoration-tag" :class="Number(item.status) === 1 ? 'is-on' : 'is-off-tag'">
                    {{ Number(item.status) === 1 ? '已上架' : '已下架' }}
                  </span>
                  <span class="decoration-tag">{{ priceText(item) }}</span>
                  <span class="decoration-tag">{{ durationText(item) }}</span>
                  <span v-if="renewChip(item)" class="decoration-tag is-renew" :title="renewTip(item)">
                    {{ renewChip(item) }}
                  </span>
                  <span class="decoration-tag">售出 {{ Number(item.sold) || 0 }}</span>
                </div>
              </div>

              <div class="decoration-actions">
                <button class="btn btn-sm" :disabled="busy" @click="openEdit(item)">
                  <i class="bi bi-pencil" /> 编辑
                </button>
                <button v-if="trash" class="btn btn-sm" :disabled="busy" @click="restoreOne(item)">
                  <i class="bi bi-arrow-counterclockwise" /> 恢复
                </button>
                <template v-else>
                  <button class="btn btn-sm" :disabled="busy" @click="openGrant(item)">
                    <i class="bi bi-gift" /> 发放
                  </button>
                  <button class="btn btn-sm btn-danger" :disabled="busy" @click="askRemove(item)">
                    <i class="bi bi-trash" /> 删除
                  </button>
                </template>
              </div>
            </div>
          </div>

          <Pagination
          v-model:current="page"
          :total="total"
          :page-size="pageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { pageSize = size; page = 1; load() }"
          @update:current="load()"
        />
        </template>
      </template>

      <!-- ===== 发放 / 回收 ===== -->
      <template v-else>
        <p class="form-hint grant-tip">
          <i class="bi bi-info-circle" />
          发放为幂等操作（重复发放只续期，不产生重复记录）；回收会同时卸下正在佩戴的该装扮。
        </p>

        <div class="grant-layout">
          <section class="grant-col">
            <div class="grant-col-head">
              <span>① 选择用户</span>
              <span class="grant-count">已选 {{ grant.selectedUids.length }}</span>
            </div>
            <div class="search-box full">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="grant.keyword"
                class="search-input"
                type="search"
                placeholder="搜索昵称后回车"
                aria-label="搜索用户昵称"
                @keyup.enter="loadGrantUsers()" />
            </div>
            <div v-if="grant.loading" class="is-loading small"><span class="spinner" /> 加载中...</div>
            <ul v-else class="pick-list">
              <li v-for="user in grant.users" :key="user.id" class="pick-item">
                <label class="pick">
                  <FormCheckbox
                    :model-value="grant.selectedUids.includes(Number(user.id))"
                    @change="toggleGrantUser(user.id)" />
                </label>
                <img v-if="user.avatar" class="pick-avatar" :src="user.avatar" :alt="user.nickname" loading="lazy" />
                <span v-else class="pick-avatar is-empty"><i class="bi bi-person" /></span>
                <span class="pick-name">{{ user.nickname || '未命名' }}</span>
                <span class="pick-sub">#{{ user.id }}</span>
              </li>
              <li v-if="!grant.users.length" class="pick-empty">没有匹配的用户</li>
            </ul>
          </section>

          <section class="grant-col">
            <div class="grant-col-head">
              <span>② 选择装扮</span>
              <span class="grant-count">已选 {{ grant.selectedDecos.length }}</span>
            </div>
            <div class="type-chips grant-chips">
              <button
                v-for="t in typeChips"
                :key="t.type"
                type="button"
                class="chip"
                :class="{ 'is-active': grant.type === t.type }"
                @click="grant.type = t.type"
              >
                <i :class="t.icon" aria-hidden="true" /> {{ t.name }}
              </button>
            </div>
            <div v-if="grant.itemsLoading" class="is-loading small"><span class="spinner" /> 加载中...</div>
            <ul v-else class="pick-list">
              <li v-for="item in grantDecorations" :key="item.id" class="pick-item">
                <label class="pick">
                  <FormCheckbox
                    :model-value="grant.selectedDecos.includes(Number(item.id))"
                    @change="toggleGrantDeco(item.id)" />
                </label>
                <img
                  v-if="item.type === 'avatar_frame'"
                  class="pick-avatar"
                  :src="item.preview || item.payload?.image"
                  :alt="item.name"
                  loading="lazy"
                />
                <span v-else class="pick-avatar is-empty"><i class="bi bi-award" /></span>
                <span class="pick-name">{{ item.name }}</span>
                <span class="pick-sub">
                  {{ typeName(item.type) }}
                  <template v-if="Number(item.status) !== 1"> · 已下架</template>
                </span>
              </li>
              <li v-if="!grantDecorations.length" class="pick-empty">该分类下暂无装扮</li>
            </ul>
          </section>
        </div>

        <div class="grant-foot">
          <div class="form-item inline">
            <label class="form-label">有效期</label>
            <FormSelect v-model="grant.duration" :options="DURATION_OPTIONS" />
          </div>
          <div class="grant-actions">
            <button
              class="btn btn-primary"
              :disabled="grant.busy || !grant.selectedUids.length || !grant.selectedDecos.length"
              @click="askGrant()"
            >
              <span v-if="grant.busy" class="spinner" />
              <i v-else class="bi bi-gift" /> 批量发放
            </button>
            <button
              class="btn btn-danger"
              :disabled="grant.busy || !grant.selectedUids.length || !grant.selectedDecos.length"
              @click="askRevoke()"
            >
              <i class="bi bi-arrow-return-left" /> 批量回收
            </button>
            <button class="btn btn-ghost" :disabled="grant.busy" @click="clearGrantSelection()">清空选择</button>
          </div>
        </div>
      </template>
    </div>

    <!-- ===== 弹窗：新增 / 编辑装扮 ===== -->
    <Teleport to="body">
      <Transition name="modal-fade">
        <div v-if="form.visible" class="modal-overlay" @click.self="closeForm()">
          <div class="modal-card modal-lg">
            <div class="modal-header">
              <span class="modal-title">{{ form.id ? '编辑装扮' : '新建装扮' }}</span>
              <button class="btn btn-icon btn-sm btn-round" @click="closeForm()"><i class="bi bi-x-lg" /></button>
            </div>

            <div class="modal-body">
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">装扮类型 <span class="required">*</span></label>
                  <FormSelect v-model="form.type" :options="typeSelectOptions" :disabled="!!form.id" />
                  <p class="form-hint">{{ currentTypeMeta.description }}</p>
                </div>
                <div class="form-item">
                  <label class="form-label">装扮名称 <span class="required">*</span></label>
                  <FormInput v-model="form.name" maxlength="40" placeholder="如：掌门、星辰头像框" />
                </div>
              </div>

              <div class="form-item">
                <label class="form-label">描述</label>
                <FormInput v-model="form.description" maxlength="120" placeholder="展示给用户的说明文字" />
              </div>

              <div class="section-title">
                <i class="bi bi-brush" /> 样式配置
                <span class="section-preview">
                  预览：
                  <img
                    v-if="form.type === 'avatar_frame'"
                    class="preview-frame"
                    :src="form.p_image || form.preview"
                    alt="预览"
                    loading="lazy"
                  />
                  <!-- 图片头衔由 TitleBadge 自己识别：有图片就只显示图片，这里传的文字会被忽略 -->
                  <TitleBadge
                    v-else
                    :payload="formPayloadPreview"
                    :text="form.p_text || '头衔预览'"
                    :size="'lg'"
                  />
                </span>
              </div>

              <!-- 素材地址 / 缩放比例：头像框必填；头衔可选（填了就是「图片头衔」） -->
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">
                    {{ form.type === 'avatar_frame' ? '素材地址' : '头衔图片（可选）' }}
                    <span v-if="form.type === 'avatar_frame'" class="required">*</span>
                  </label>
                  <FormInput
                    v-model="form.p_image"
                    :placeholder="
                      form.type === 'avatar_frame'
                        ? 'https://.../frame.png（支持 gif）'
                        : 'https://.../badge.png，留空则为纯文字头衔'
                    " />
                  <div class="image-actions">
                    <button type="button" class="btn btn-sm" :disabled="uploading" @click="pickDecorationImage">
                      <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
                    </button>
                    <button type="button" class="btn btn-sm" title="从附件库选择" @click="showImageLibrary = true">
                      <i class="bi bi-folder2-open" /> 附件库
                    </button>
                    <button v-if="form.p_image" type="button" class="btn btn-sm btn-ghost" @click="form.p_image = ''">
                      移除
                    </button>
                  </div>
                  <input ref="imageFileRef" type="file" accept="image/*" hidden @change="onDecorationImageChange" />

                  <AttachmentLibrary
                    v-model:visible="showImageLibrary"
                    title="选择装扮素材"
                    accept="image"
                    @select="onImageLibrarySelect"
                  />

                  <p v-if="form.type === 'title'" class="form-hint">
                    填了图片即为「纯图片头衔」：只显示图片、不再显示文字（与文字头衔二选一）
                  </p>
                </div>
                <div v-if="form.type !== 'avatar_frame'" class="form-item">
                  <label class="form-label">缩放比例</label>
                  <FormInput v-model="form.p_scale" type="number" step="0.05" min="0.5" max="3" />
                  <p class="form-hint">控制图片头衔的高度，1 为默认尺寸</p>
                </div>
              </div>

              <!--
                头像框对齐器（仅 avatar_frame）

                为什么不是「一个缩放数字」：等比放大会把框中间的洞一起放大，
                装饰越大洞就越撑，头像怎么都填不满。这里改成以「洞」为基准：
                洞大小固定贴合头像，装饰自然向外延伸，再用两个偏移修画布留白不对称。
              -->
              <div v-if="form.type === 'avatar_frame'" class="frame-aligner">
                <div class="frame-aligner-head">
                  <span class="frame-aligner-title">
                    <i class="bi bi-crosshair" /> 对齐预览
                  </span>
                  <button type="button" class="btn btn-sm btn-ghost" @click="resetFrameAlign">
                    <i class="bi bi-arrow-counterclockwise" /> 重置
                  </button>
                </div>

                <div class="frame-aligner-body">
                  <div class="frame-aligner-stage">
                    <img class="frame-aligner-avatar" :src="alignerAvatar" alt="头像示例" loading="lazy" />
                    <img
                      v-if="form.p_image"
                      class="frame-aligner-overlay"
                      :style="alignerStyle"
                      :src="form.p_image"
                      alt=""
                      draggable="false"
                      @pointerdown="onAlignPointerDown"
                      @pointermove="onAlignPointerMove"
                      @pointerup="onAlignPointerUp"
                      @pointercancel="onAlignPointerUp"
                      loading="lazy"
                    />
                    <p v-else class="frame-aligner-empty">先填上面的素材地址</p>
                  </div>

                  <div class="frame-aligner-controls">
                    <label class="align-row">
                      <span class="align-row-label">洞大小</span>
                      <input v-model.number="form.p_hole" class="align-range" type="range" min="0.2" max="1.5" step="0.01" />
                      <span class="align-row-value">{{ Number(form.p_hole).toFixed(2) }}</span>
                    </label>
                    <label class="align-row">
                      <span class="align-row-label">水平偏移</span>
                      <input v-model.number="form.p_ox" class="align-range" type="range" min="-0.5" max="0.5" step="0.005" />
                      <span class="align-row-value">{{ Number(form.p_ox).toFixed(3) }}</span>
                    </label>
                    <label class="align-row">
                      <span class="align-row-label">垂直偏移</span>
                      <input v-model.number="form.p_oy" class="align-range" type="range" min="-0.5" max="0.5" step="0.005" />
                      <span class="align-row-value">{{ Number(form.p_oy).toFixed(3) }}</span>
                    </label>

                    <p class="form-hint">
                      「洞大小」= 框中间透出头像的圆 ÷ 素材宽度。洞偏了就调下面的偏移，
                      也可以直接在预览里拖动素材。
                    </p>
                  </div>
                </div>
              </div>

              <div v-if="form.type === 'title'" class="form-grid">
                <div class="form-item">
                  <label class="form-label">头衔文字</label>
                  <FormInput v-model="form.p_text" maxlength="20" placeholder="留空则由用户佩戴时自填" />
                  <p class="form-hint">留空则是纯样式模板（佩戴时需用户输入文字）；与上面的图片二选一</p>
                </div>
                <div class="form-item">
                  <label class="form-label">图标（Bootstrap Icons）</label>
                  <FormInput v-model="form.p_icon" placeholder="如 bi bi-award" />
                </div>
                <div class="form-item">
                  <label class="form-label">文字颜色</label>
                  <div class="color-field">
                    <FormInput v-model="form.p_color" placeholder="#ffffff" />
                    <input v-model="form.p_color" class="color-pick" type="color" />
                  </div>
                </div>
                <div class="form-item">
                  <label class="form-label">背景</label>
                  <div class="color-field">
                    <FormInput v-model="form.p_bg" placeholder="#8a8a82 或 linear-gradient(...)" />
                    <input v-model="form.p_bg" class="color-pick" type="color" />
                  </div>
                </div>
                <div class="form-item">
                  <label class="form-label">边框</label>
                  <FormInput v-model="form.p_border" placeholder="留空无边框，也可填 #fcd34d" />
                </div>
                <div class="form-item">
                  <label class="form-label">发光（box-shadow）</label>
                  <FormInput v-model="form.p_glow" placeholder="如 0 0 10px rgba(0,0,0,.3)" />
                </div>
              </div>

              <label class="switch-row">
                <FormCheckbox v-model="form.p_animated" />
                <span>开启动效（头衔呼吸 / 头像框闪烁）</span>
              </label>

              <details class="advanced">
                <summary>高级：其它样式字段（JSON）</summary>
                <FormTextarea
                  v-model="form.extraJson"
                  rows="3"
                  spellcheck="false"
                  placeholder='{"key": "value"}，会与上方字段合并写入 payload' />
              </details>

              <div class="section-title"><i class="bi bi-coin" /> 获取方式</div>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">价格类型</label>
                  <FormSelect v-model="form.price_type" :options="PRICE_TYPE_OPTIONS" />
                </div>
                <div class="form-item">
                  <label class="form-label">所需积分</label>
                  <FormInput
                    v-model="form.price"
                    type="number"
                    min="0"
                    :disabled="form.price_type === 'free' || Number(form.is_default) === 1" />
                  <p class="form-hint">0 或免费装扮不出售（可作为默认装扮 / 活动发放）</p>
                </div>
                <div class="form-item">
                  <label class="form-label">有效期（天）</label>
                  <FormInput v-model="form.durationDays" type="number" min="0" />
                  <p class="form-hint">0 表示永久有效</p>
                </div>
                <div class="form-item">
                  <label class="form-label">库存</label>
                  <FormInput v-model="form.stock" type="number" min="-1" />
                  <p class="form-hint">-1 表示不限量，0 表示已售罄</p>
                </div>
                <div class="form-item">
                  <label class="form-label">分类名</label>
                  <FormInput v-model="form.category" placeholder="留空自动按类型命名" />
                </div>
                <div class="form-item">
                  <label class="form-label">稀有度</label>
                  <FormSelect v-model="form.rarity" :options="RARITY_OPTIONS" />
                </div>
                <div class="form-item">
                  <label class="form-label">排序值</label>
                  <FormInput v-model="form.sort" type="number" />
                  <p class="form-hint">越大越靠前</p>
                </div>
              </div>

              <!-- 续期：档位与价格是这件装扮自己的，用户过期后按这里的档位续费 -->
              <RenewOptionsEditor v-model="form.renew" />
              <p v-if="Number(form.durationDays) <= 0" class="form-hint">
                当前装扮有效期天数为 0（永久有效），续期档位不会在前台展示
              </p>

              <div class="switch-group">
                <label class="switch-row">
                  <FormCheckbox v-model="form.status" :true-value="1" :false-value="0" />
                  <span>上架（前台可见、可佩戴）</span>
                </label>
                <label class="switch-row">
                  <FormCheckbox v-model="form.is_default" :true-value="1" :false-value="0" />
                  <span>系统默认（所有用户免费拥有，商城不售卖）</span>
                </label>
              </div>
            </div>

            <div class="modal-footer">
              <button class="btn btn-ghost" :disabled="form.loading" @click="closeForm()">取消</button>
              <button class="btn btn-primary" :disabled="form.loading" @click="save()">
                <span v-if="form.loading" class="spinner" /> 保存
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :danger="confirm.danger"
      :loading="confirm.loading"
      :confirm-text="confirm.confirmText"
      @confirm="runConfirm()"
    />
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 装扮管理（/admin/decoration）
 *
 * 两个视图：
 *  1. 装扮列表 —— 头像框 / 头衔的增删改查、上下架、回收站；
 *  2. 发放 / 回收 —— 批量用户 × 批量装扮，发放幂等（重复发放只续期）。
 *
 * 后端约束（app/api/controller/decoration.go、app/model/decoration.go）：
 * - 装扮商城是独立分区，与积分商城（goods）完全解耦：价格 / 库存 / 经验门槛
 *   都存在装扮自身，前台兑换走 decoration/buy（只扣积分，不生成商品与订单）；
 * - 库存取值：-1 不限量（默认）、0 已售罄、>0 剩余数量（稀缺性只用库存控制，
 *   每人限购已移除：装扮是一件拥有、重复兑换即续期，限购会连续期一起挡掉）；
 * - payload 必须提交 JSON 字符串，后端按 utils.Json.Decode 解析后原样下发；
 * - duration 单位为秒，0 表示永久；表单里按「天」录入后换算；
 * - 彻底删除会连带清理用户已拥有的记录（不可恢复）。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import { listUsers } from '@/api/users'
import {
  listDecorationsAdmin,
  countDecorations,
  getDecorationTypes,
  saveDecoration,
  removeDecorations,
  forceDeleteDecorations,
  restoreDecorations,
  clearDecorationRecycle,
  // 与本地 computed「grantDecorations」重名，这里起别名
  grantDecorations as grantDecorationsApi,
  revokeDecorations as revokeDecorationsApi
} from '@/api/decoration'
import { toast } from '@/utils/toast'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { frameGeometryFromPayload } from '@/utils/helper'
import { useUserStore } from '@/stores/user'
import TitleBadge from '@/components/TitleBadge.vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import RenewOptionsEditor from '@/components/RenewOptionsEditor.vue'
import { emptyRenewConfig, parseRenewConfig } from '@/utils/renew'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 头像框对齐器的示例头像用当前管理员的，更接近真实观感
const userStore = useUserStore()

// ===== 装扮素材：上传 / 附件库 =====
const imageFileRef = ref(null)
const uploading = ref(false)
const showImageLibrary = ref(false)

function pickDecorationImage() {
  imageFileRef.value?.click()
}

async function onDecorationImageChange(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return

  if (!file.type.startsWith('image/')) {
    toast.warning('请选择图片文件')
    return
  }
  if (file.size > 5 * 1024 * 1024) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploading.value = true
  try {
    const fd = new FormData()
    fd.append('files', file)
    const res = await uploadAttachments(fd)
    const url = res.data?.results?.[0]?.full_url || res.data?.results?.[0]?.url || ''
    if (!url) throw new Error('empty')

    form.p_image = url
    toast.success('图片已上传')
  } catch {
    toast.error('图片上传失败')
  } finally {
    uploading.value = false
  }
}

// 从附件库选素材：直接复用已上传图片的地址
function onImageLibrarySelect(urls = []) {
  const url = urls[0]
  showImageLibrary.value = false
  if (url) form.p_image = url
}

// 列表状态：all 全部 / on 已上架 / off 已下架 / trash 回收站
const STATUS_TABS = [
  { key: 'all', label: '全部' },
  { key: 'on', label: '已上架' },
  { key: 'off', label: '已下架' },
  { key: 'trash', label: '回收站' }
]

// 价格筛选：'' 全部 / free 免费 / paid 付费（= price_type 非 free）
// 与后端 decoration/all 的 price_type 参数一一对应（见 model.FilterDecorationPrice）
const PRICE_TABS = [
  { key: '', label: '全部' },
  { key: 'free', label: '免费' },
  { key: 'paid', label: '付费' }
]

const DURATION_OPTIONS = [
  { value: 0, label: '按装扮自身配置' },
  { value: 86400 * 7, label: '7 天' },
  { value: 86400 * 30, label: '30 天' },
  { value: 86400 * 90, label: '90 天' },
  { value: 86400 * 365, label: '365 天' },
  { value: -1, label: '永久（忽略装扮有效期）' }
]

const PRICE_TYPE_OPTIONS = [
  { value: 'integral', label: '积分兑换' },
  { value: 'free', label: '免费' }
]

const RARITY_OPTIONS = [
  { value: 'normal', label: '普通' },
  { value: 'rare', label: '稀有' },
  { value: 'epic', label: '史诗' },
  { value: 'legend', label: '传说' }
]

// ===== 基础数据 =====
const view = ref('list')
const types = ref([])

const stats = reactive({ total: 0, on_sale: 0, owned: 0, by_type: {} })

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)

const statusFilter = ref('all')
const typeFilter = ref('')
// 价格筛选：'' 全部 / free 免费 / paid 付费
const priceFilter = ref('')
const keyword = ref('')
// 搜索防抖：输入时只改这个值，watchEffect 之外的请求由它驱动
const searchKey = ref('')

const selectedIds = ref([])

const trash = computed(() => statusFilter.value === 'trash')
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((item) => selectedIds.value.includes(Number(item.id)))
)

// 类型 chips：全部 + 后端声明的类型
const typeChips = computed(() => [
  { type: '', name: '全部', icon: 'bi-grid' },
  ...types.value.map((t) => ({ type: t.type, name: t.name, icon: t.icon }))
])
const typeSelectOptions = computed(() =>
  types.value.map((t) => ({ value: t.type, label: `${t.name}（${t.type}）` }))
)
const currentTypeMeta = computed(
  () => types.value.find((t) => t.type === form.type) || { description: '' }
)

function typeName(type) {
  const found = types.value.find((t) => t.type === type)
  return found?.name || type || '装扮'
}

// ===== 列表 =====
function parseJson(value) {
  if (!value) return {}
  if (typeof value === 'object') return value
  try {
    const data = JSON.parse(value)
    return data && typeof data === 'object' ? data : {}
  } catch {
    return {}
  }
}

async function load() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (statusFilter.value === 'on') params.status = 1
    if (statusFilter.value === 'off') params.status = 0
    if (trash.value) params.onlyTrashed = 'true'
    if (typeFilter.value) params.type = typeFilter.value
    if (priceFilter.value) params.price_type = priceFilter.value
    if (searchKey.value) params.keyword = searchKey.value

    const res = await listDecorationsAdmin(params)
    list.value = (res?.data?.data || []).map((item) => ({ ...item, payload: parseJson(item.payload) }))
    total.value = Number(res?.data?.count || 0)
    clearSelection()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function loadStats() {
  try {
    const res = await countDecorations()
    const data = res?.data || {}
    stats.total = Number(data.total || 0)
    stats.on_sale = Number(data.on_sale || 0)
    stats.owned = Number(data.owned || 0)
    stats.by_type = data.by_type || {}
  } catch {
    // 统计失败不影响主流程
  }
}

async function loadTypes() {
  try {
    const res = await getDecorationTypes()
    types.value = res?.data?.types || []
  } catch {
    types.value = []
  }
}

function reload() {
  load()
  loadStats()
}

// ===== 筛选 =====
function switchStatus(key) {
  if (statusFilter.value === key) return
  statusFilter.value = key
  page.value = 1
  load()
}

function switchType(type) {
  if (typeFilter.value === type) return
  typeFilter.value = type
  page.value = 1
  load()
}

function switchPrice(key) {
  if (priceFilter.value === key) return
  priceFilter.value = key
  page.value = 1
  load()
}

let searchTimer = null
function doSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchKey.value = keyword.value.trim()
    page.value = 1
    load()
  }, 350)
}

// ===== 选择 =====
function isSelected(id) {
  return selectedIds.value.includes(Number(id))
}

function toggleSelect(id) {
  const value = Number(id)
  const index = selectedIds.value.indexOf(value)
  if (index === -1) selectedIds.value.push(value)
  else selectedIds.value.splice(index, 1)
}

function toggleSelectAll(event) {
  selectedIds.value = event.target.checked ? list.value.map((item) => Number(item.id)) : []
}

function clearSelection() {
  selectedIds.value = []
}

// ===== 展示辅助 =====
function priceText(item) {
  if (Number(item.is_default) === 1) return '系统默认'
  if (item.price_type === 'free' || Number(item.price) <= 0) return '免费'
  return `${Number(item.price)} 积分`
}

function durationText(item) {
  const seconds = Number(item.duration || 0)
  if (seconds <= 0) return '永久'
  return `${Math.round(seconds / 86400)} 天`
}

// 续期档位：价格是每件装扮自己的，列表只给概览，明细放 title
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

// ===== 新增 / 编辑 =====
const form = reactive({
  visible: false,
  loading: false,
  id: 0,
  type: 'avatar_frame',
  name: '',
  description: '',
  category: '',
  price_type: 'integral',
  price: 0,
  rarity: 'normal',
  status: 1,
  sort: 100,
  is_default: 0,
  // -1 = 不限量（后端约定：0 表示已售罄）
  stock: -1,
  durationDays: 0,
  // 续期档位（这件装扮自己的，见 components/RenewOptionsEditor.vue）
  renew: emptyRenewConfig(),
  p_image: '',
  p_text: '',
  p_color: '',
  p_bg: '',
  p_border: '',
  p_glow: '',
  p_icon: '',
  p_scale: 1.15,
  // 头像框的「洞」参数（见 frameGeometryFromPayload / 后端 frameGeometry）
  p_hole: 1,
  p_ox: 0,
  p_oy: 0,
  p_animated: false,
  extraJson: ''
})

// 已知的结构化字段：其余键进入「高级 JSON」编辑
const KNOWN_PAYLOAD_KEYS = [
  'image', 'text', 'color', 'bg', 'border', 'glow', 'icon',
  'scale', 'hole', 'offset_x', 'offset_y', 'animated'
]

// 弹窗内的实时预览（结构与提交时的 payload 保持一致）
const formPayloadPreview = computed(() => buildPayload())

function buildPayload() {
  const payload = {}
  // 素材地址与缩放：头像框必填、头衔选填（头衔填了即「图片头衔」），两者共用一套字段
  if (form.p_image) {
    payload.image = form.p_image

    if (form.type === 'avatar_frame') {
      // 头像框用「洞模型」：hole = 洞直径 ÷ 素材宽度，offset 修洞心偏移。
      // 同时回写一个等价的 scale（= 1/hole），让还在读旧字段的消费方保持一致。
      const hole = clampRange(Number(form.p_hole), HOLE_MIN, HOLE_MAX, 1)
      payload.hole = hole
      payload.offset_x = clampRange(Number(form.p_ox), -OFFSET_LIMIT, OFFSET_LIMIT, 0)
      payload.offset_y = clampRange(Number(form.p_oy), -OFFSET_LIMIT, OFFSET_LIMIT, 0)
      payload.scale = Number((1 / hole).toFixed(3))
    } else {
      const scale = Number(form.p_scale)
      if (scale > 0) payload.scale = scale
    }
  }

  if (form.type !== 'avatar_frame') {
    if (form.p_text) payload.text = form.p_text
    if (form.p_color) payload.color = form.p_color
    if (form.p_bg) payload.bg = form.p_bg
    if (form.p_border) payload.border = form.p_border
    if (form.p_glow) payload.glow = form.p_glow
    if (form.p_icon) payload.icon = form.p_icon
  }
  payload.animated = !!form.p_animated

  // 合并高级 JSON 中的自定义字段
  const text = form.extraJson.trim()
  if (text) {
    try {
      const extra = JSON.parse(text)
      if (extra && typeof extra === 'object' && !Array.isArray(extra)) {
        Object.assign(payload, extra)
      }
    } catch {
      // 非法 JSON 在保存时统一拦截，这里预览忽略
    }
  }
  return payload
}

// ===== 头像框对齐器 =====
//
// 舞台里放一个「真实尺寸的头像」（120px），素材按「头像直径 / hole」渲染，
// 洞正好贴住头像。拖素材或拖滑块改的是 hole / offset，实时看到的就是前台效果。
const HOLE_MIN = 0.2
const HOLE_MAX = 1.5
const OFFSET_LIMIT = 0.5
const ALIGNER_AVATAR = 120

const FRAME_FALLBACK_AVATAR =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

function clampRange(value, min, max, fallback = 0) {
  if (!Number.isFinite(value)) return fallback
  return Math.min(max, Math.max(min, value))
}

const alignerAvatar = computed(() => userStore.user?.avatar || FRAME_FALLBACK_AVATAR)

const alignerHole = computed(() => clampRange(Number(form.p_hole), HOLE_MIN, HOLE_MAX, 1))

// 素材渲染宽度 = 头像直径 / hole（hole 越小，素材越大、装饰越舒展）
const alignerOverlayWidth = computed(() => ALIGNER_AVATAR / alignerHole.value)

const alignerStyle = computed(() => {
  const ox = clampRange(Number(form.p_ox), -OFFSET_LIMIT, OFFSET_LIMIT, 0)
  const oy = clampRange(Number(form.p_oy), -OFFSET_LIMIT, OFFSET_LIMIT, 0)

  return {
    width: `${alignerOverlayWidth.value.toFixed(2)}px`,
    height: 'auto',
    // translate 的百分比以素材自身宽度为基准，与 AvatarFrame 的实现保持一致
    transform: `translate(calc(-50% - ${(ox * 100).toFixed(2)}%), calc(-50% - ${(oy * 100).toFixed(2)}%))`
  }
})

// 拖动素材：像素位移 ÷ 素材宽度 = 偏移量。方向取反 ——
// offset 变大时素材是往左/上移的（洞心右移等于素材左移）。
const alignDrag = ref(null)

function onAlignPointerDown(event) {
  if (!form.p_image) return

  alignDrag.value = {
    x: event.clientX,
    y: event.clientY,
    ox: Number(form.p_ox) || 0,
    oy: Number(form.p_oy) || 0,
    width: alignerOverlayWidth.value || 1
  }

  event.currentTarget?.setPointerCapture?.(event.pointerId)
}

function onAlignPointerMove(event) {
  const drag = alignDrag.value
  if (!drag) return

  const width = drag.width || 1
  const nextX = drag.ox - (event.clientX - drag.x) / width
  const nextY = drag.oy - (event.clientY - drag.y) / width

  form.p_ox = Number(clampRange(nextX, -OFFSET_LIMIT, OFFSET_LIMIT, 0).toFixed(3))
  form.p_oy = Number(clampRange(nextY, -OFFSET_LIMIT, OFFSET_LIMIT, 0).toFixed(3))
}

function onAlignPointerUp(event) {
  if (!alignDrag.value) return

  alignDrag.value = null
  event.currentTarget?.releasePointerCapture?.(event.pointerId)
}

function resetFrameAlign() {
  form.p_hole = 1
  form.p_ox = 0
  form.p_oy = 0
}

function resetForm(type) {
  form.id = 0
  form.type = type || types.value[0]?.type || 'avatar_frame'
  form.name = ''
  form.description = ''
  form.category = ''
  form.price_type = 'integral'
  form.price = 0
  form.rarity = 'normal'
  form.status = 1
  form.sort = 100
  form.is_default = 0
  form.stock = -1
  form.durationDays = 0
  form.renew = emptyRenewConfig()
  form.p_image = ''
  form.p_text = ''
  form.p_color = ''
  form.p_bg = ''
  form.p_border = ''
  form.p_glow = ''
  form.p_icon = ''
  // 头衔图片的 1 就是默认高度；头像框不再用 scale，改由下面的洞参数控制
  form.p_scale = 1
  // 洞默认 1：素材按头像大小 1:1 渲染，洞 = 画布内切圆
  form.p_hole = 1
  form.p_ox = 0
  form.p_oy = 0
  form.p_animated = false
  form.extraJson = ''
}

function openAdd() {
  resetForm()
  form.visible = true
}

function openEdit(item) {
  resetForm(item.type)

  form.id = Number(item.id)
  form.type = item.type
  form.name = item.name || ''
  form.description = item.description || ''
  form.category = item.category || ''
  form.price_type = item.price_type === 'free' ? 'free' : 'integral'
  form.price = Number(item.price || 0)
  form.rarity = item.rarity || 'normal'
  form.status = Number(item.status) === 1 ? 1 : 0
  form.sort = Number(item.sort || 0)
  form.is_default = Number(item.is_default) === 1 ? 1 : 0
  form.durationDays = Math.round(Number(item.duration || 0) / 86400)
  // 列表返回的是库里的 JSON 字符串，统一解析成 { enabled, options }
  form.renew = parseRenewConfig(item.renew)

  const payload = parseJson(item.payload)
  form.p_image = payload.image || item.preview || ''
  form.p_text = payload.text || ''
  form.p_color = payload.color || ''
  form.p_bg = payload.bg || ''
  form.p_border = payload.border || ''
  form.p_glow = payload.glow || ''
  form.p_icon = payload.icon || ''
  form.p_scale = payload.scale ?? 1
  form.p_animated = payload.animated === true || payload.animated === 1

  // 头像框：优先读 hole；老数据只有 scale 时按 hole = 1/scale 换算（与后端同口径）
  if (item.type === 'avatar_frame') {
    const geometry = frameGeometryFromPayload(payload)
    // 历史数据可能超出滑块范围（或 hole=0），这里夹一下，避免滑块位置和实际值不一致
    form.p_hole = clampRange(geometry.hole, HOLE_MIN, HOLE_MAX, 1)
    form.p_ox = clampRange(geometry.offsetX, -OFFSET_LIMIT, OFFSET_LIMIT, 0)
    form.p_oy = clampRange(geometry.offsetY, -OFFSET_LIMIT, OFFSET_LIMIT, 0)
  } else {
    form.p_hole = 1
    form.p_ox = 0
    form.p_oy = 0
  }

  const extra = {}
  Object.keys(payload).forEach((key) => {
    if (!KNOWN_PAYLOAD_KEYS.includes(key)) extra[key] = payload[key]
  })
  form.extraJson = Object.keys(extra).length ? JSON.stringify(extra, null, 2) : ''

  form.visible = true
}

function closeForm() {
  if (form.loading) return
  form.visible = false
}

async function save() {
  if (!form.type) {
    toast.warning('请选择装扮类型')
    return
  }
  if (!form.name.trim()) {
    toast.warning('请输入装扮名称')
    return
  }
  if (form.type === 'avatar_frame' && !form.p_image.trim()) {
    toast.warning('请填写头像框素材地址')
    return
  }
  // 头衔只有两种形态且互斥：纯文字 / 纯图片。
  // 都为空 -> 没有任何内容可展示；同时填 -> 图片会盖掉文字（用户以为配了文字却没生效）
  if (form.type === 'title') {
    const hasText = form.p_text.trim()
    const hasImage = form.p_image.trim()
    if (!hasText && !hasImage) {
      toast.warning('头衔需要填写文字或图片')
      return
    }
    if (hasText && hasImage) {
      toast.warning('头衔只能二选一：纯文字 或 纯图片')
      return
    }
  }

  const text = form.extraJson.trim()
  if (text) {
    try {
      const extra = JSON.parse(text)
      if (!extra || typeof extra !== 'object' || Array.isArray(extra)) throw new Error('invalid')
    } catch {
      toast.warning('高级 JSON 格式有误，请检查后重试')
      return
    }
  }

  const isFree = form.price_type === 'free' || Number(form.is_default) === 1
  // 库存：-1 不限量 / 0 已售罄 / >0 剩余（空值按「不限量」处理）
  const rawStock = form.stock === '' || form.stock === null ? -1 : Number(form.stock)
  const stock = Number.isFinite(rawStock) ? Math.max(-1, Math.trunc(rawStock)) : -1
  const payload = {
    type: form.type,
    name: form.name.trim(),
    description: form.description.trim(),
    category: form.category.trim(),
    // 头像框以素材地址作为列表缩略图
    preview: form.type === 'avatar_frame' ? form.p_image.trim() : '',
    payload: JSON.stringify(buildPayload()),
    price_type: isFree ? 'free' : 'integral',
    price: isFree ? 0 : Math.max(0, Number(form.price) || 0),
    rarity: form.rarity,
    status: Number(form.status) === 1 ? 1 : 0,
    sort: Number(form.sort) || 0,
    is_default: Number(form.is_default) === 1 ? 1 : 0,
    // 秒；0 = 永久
    duration: Math.max(0, Number(form.durationDays) || 0) * 86400,
    renew: form.renew,
    stock
  }
  if (form.id) payload.id = form.id

  form.loading = true
  try {
    await saveDecoration(payload)
    toast.success(form.id ? '装扮已更新' : '装扮已创建')
    form.visible = false
    reload()
  } catch {
    // 错误已由拦截器提示
  } finally {
    form.loading = false
  }
}

// ===== 删除 / 回收站 =====
const confirm = reactive({
  visible: false,
  loading: false,
  title: '操作确认',
  message: '',
  danger: false,
  confirmText: '确定',
  action: null
})

function ask({ title, message, confirmText = '确定', danger = false, action }) {
  confirm.title = title
  confirm.message = message
  confirm.confirmText = confirmText
  confirm.danger = danger
  confirm.action = action
  confirm.loading = false
  confirm.visible = true
}

async function runConfirm() {
  if (!confirm.action) return
  confirm.loading = true
  try {
    await confirm.action()
    confirm.visible = false
  } catch {
    // 错误已由拦截器提示，保留弹窗让管理员重试
  } finally {
    confirm.loading = false
  }
}

function removeByIds(ids) {
  return removeDecorations(ids)
}

function askRemove(item) {
  ask({
    title: '删除装扮',
    message: `确定把「${item.name}」移入回收站吗？前台将立即不可见，关联商品会自动下架。`,
    confirmText: '移入回收站',
    danger: true,
    action: async () => {
      await removeByIds([item.id])
      toast.success('已移入回收站')
      reload()
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  ask({
    title: '批量删除',
    message: `确定把选中的 ${ids.length} 个装扮移入回收站吗？`,
    confirmText: '移入回收站',
    danger: true,
    action: async () => {
      await removeByIds(ids)
      toast.success('已移入回收站')
      reload()
    }
  })
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  ask({
    title: '彻底删除',
    message: `彻底删除后不可恢复，且会一并清理用户已拥有的这 ${ids.length} 个装扮记录。确定继续吗？`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteDecorations(ids)
      toast.success('已彻底删除')
      reload()
    }
  })
}

function askClearRecycle() {
  ask({
    title: '清空回收站',
    message: '回收站内的装扮将被彻底删除且不可恢复，同时清理用户持有记录。确定继续吗？',
    confirmText: '清空回收站',
    danger: true,
    action: async () => {
      await clearDecorationRecycle()
      toast.success('回收站已清空')
      page.value = 1
      reload()
    }
  })
}

async function restoreOne(item) {
  busy.value = true
  try {
    await restoreDecorations([item.id])
    toast.success('已恢复')
    reload()
  } catch {
    // 错误已由拦截器提示
  } finally {
    busy.value = false
  }
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreDecorations(ids)
    toast.success('已恢复')
    reload()
  } catch {
    // 错误已由拦截器提示
  } finally {
    busy.value = false
  }
}

// ===== 发放 / 回收 =====
const grant = reactive({
  keyword: '',
  users: [],
  loading: false,
  items: [],
  itemsLoading: false,
  selectedUids: [],
  selectedDecos: [],
  duration: 0,
  type: '',
  busy: false
})

// 发放视图可选的装扮：独立拉取（不受列表分页影响），按类型过滤
const grantDecorations = computed(() => {
  if (!grant.type) return grant.items
  return grant.items.filter((item) => item.type === grant.type)
})

async function loadGrantDecorations() {
  grant.itemsLoading = true
  try {
    // 发放不受上架状态限制，因此这里不带 status 过滤，让管理员能发放待上架的装扮
    const res = await listDecorationsAdmin({ page: 1, limit: 200 })
    grant.items = (res?.data?.data || []).map((item) => ({ ...item, payload: parseJson(item.payload) }))
  } catch {
    grant.items = []
  } finally {
    grant.itemsLoading = false
  }
}

async function loadGrantUsers() {
  grant.loading = true
  try {
    const params = { page: 1, limit: 30, field: 'id,nickname,avatar', order: 'id desc' }
    if (grant.keyword.trim()) params.like = `nickname|${grant.keyword.trim()}`
    const res = await listUsers(params)
    grant.users = res?.data?.data || []
  } catch {
    grant.users = []
  } finally {
    grant.loading = false
  }
}

function toggleGrantUser(id) {
  const value = Number(id)
  const index = grant.selectedUids.indexOf(value)
  if (index === -1) grant.selectedUids.push(value)
  else grant.selectedUids.splice(index, 1)
}

function toggleGrantDeco(id) {
  const value = Number(id)
  const index = grant.selectedDecos.indexOf(value)
  if (index === -1) grant.selectedDecos.push(value)
  else grant.selectedDecos.splice(index, 1)
}

function clearGrantSelection() {
  grant.selectedUids = []
  grant.selectedDecos = []
}

// 从列表页点「发放」进入发放视图，并预选该装扮
function openGrant(item) {
  view.value = 'grant'
  grant.selectedDecos = [Number(item.id)]
  prepareGrantView()
}

function switchGrantView() {
  view.value = 'grant'
  prepareGrantView()
}

// 发放视图需要的数据只在首次进入时加载
function prepareGrantView() {
  if (!grant.items.length) loadGrantDecorations()
  if (!grant.users.length) loadGrantUsers()
}

function askGrant() {
  const uids = [...grant.selectedUids]
  const decos = [...grant.selectedDecos]
  ask({
    title: '批量发放装扮',
    message: `将向 ${uids.length} 位用户发放 ${decos.length} 个装扮，确定继续吗？`,
    confirmText: '发放',
    action: async () => {
      grant.busy = true
      try {
        const res = await grantDecorationsApi({ uids, decoration_ids: decos, duration: grant.duration })
        const data = res?.data || {}
        toast.success(`发放完成：成功 ${data.success || 0} 条，失败 ${data.failed || 0} 条`)
        clearGrantSelection()
        loadStats()
      } finally {
        grant.busy = false
      }
    }
  })
}

function askRevoke() {
  const uids = [...grant.selectedUids]
  const decos = [...grant.selectedDecos]
  ask({
    title: '批量回收装扮',
    message: `将从 ${uids.length} 位用户回收 ${decos.length} 个装扮（正在佩戴的会一并卸下），确定继续吗？`,
    confirmText: '回收',
    danger: true,
    action: async () => {
      grant.busy = true
      try {
        const res = await revokeDecorationsApi({ uids, decoration_ids: decos })
        const data = res?.data || {}
        toast.success(`回收完成：成功 ${data.success || 0} 条，失败 ${data.failed || 0} 条`)
        clearGrantSelection()
        loadStats()
      } finally {
        grant.busy = false
      }
    }
  })
}

onMounted(() => {
  loadTypes()
  loadStats()
  load()
})
</script>

<style scoped>
.decoration-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ===== 概览 ===== */
.stat-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
}
.stat-value {
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-sub {
  font-size: 12px;
  color: var(--text-muted);
}

/* ===== 头部 ===== */
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.block-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  gap: 8px;
}

/* ===== 视图切换 ===== */
.view-tabs {
  display: inline-flex;
  gap: 4px;
  margin-top: 14px;
  padding: 4px;
  border-radius: 10px;
  background: var(--bg-muted);
  align-self: flex-start;
}
.view-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-soft);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.18s;
}
.view-tab:hover {
  color: var(--primary);
}
.view-tab.is-active {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

/* ===== 筛选栏 ===== */
.list-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.status-tabs {
  display: flex;
  gap: 4px;
}
.status-tab {
  padding: 5px 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--text-soft);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.18s;
}
.status-tab:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.status-tab.is-active {
  background: var(--bg-muted);
  color: var(--primary-deep);
  font-weight: 600;
}
.status-tab.trash-tab.is-active {
  color: #d9534f;
}
.filter-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.type-chips {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
/* 价格筛选：全部 / 免费 / 付费（分段控件，与类型 chip 区分开） */
.price-seg {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  border-radius: 999px;
  background: var(--bg-muted);
}
.price-seg-item {
  padding: 3px 10px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12px;
  line-height: 1.5;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.18s;
}
.price-seg-item:hover {
  color: var(--primary);
}
.price-seg-item.is-active {
  background: var(--bg-card);
  color: var(--primary-deep);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}
.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 13px;
  line-height: 1.5;
  cursor: pointer;
  transition: all 0.18s;
}
.chip:hover {
  border-color: var(--primary-soft);
  color: var(--primary);
}
.chip.is-active {
  background: var(--accent-soft);
  border-color: var(--primary);
  color: var(--primary-deep);
  font-weight: 600;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  color: var(--text-muted);
}
.search-box.full {
  width: 100%;
  margin: 10px 0;
}
.search-input {
  flex: 1;
  min-width: 120px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
}

/* ===== 回收站 / 批量操作 ===== */
.trash-bar,
.batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 12px;
  padding: 10px 14px;
  border-radius: 10px;
  font-size: 13px;
}
.trash-bar {
  background: rgba(217, 83, 79, 0.08);
  color: #c9302c;
}
.batch-bar {
  background: var(--bg-muted);
}
.batch-count strong {
  color: var(--primary-deep);
}
.batch-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* ===== 列表 ===== */
.is-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 0;
  color: var(--text-muted);
  font-size: 14px;
}
.is-loading.small {
  padding: 24px 0;
  font-size: 13px;
}
.empty-row {
  padding: 32px 0;
}
.list-head-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 14px 0 10px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}
.pick {
  display: inline-flex;
  align-items: center;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  cursor: pointer;
  accent-color: var(--primary);
}
.card-pick {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 2;
  padding: 2px 4px;
  border-radius: 6px;
  background: var(--bg-card);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12);
}

.decoration-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
}
.decoration-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg, 12px);
  background: var(--bg-card);
  transition: all 0.18s;
}
.decoration-card:hover {
  border-color: var(--primary-soft);
}
.decoration-card.is-selected {
  border-color: var(--primary);
  box-shadow: 0 0 0 1px var(--primary) inset;
}
.decoration-card.is-off {
  opacity: 0.72;
}

.decoration-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 96px;
  padding: 6px;
  border-radius: 10px;
  background: var(--bg-muted);
  overflow: hidden;
}
.decoration-preview.is-title {
  padding: 14px;
}
.decoration-frame {
  width: 72px;
  height: 72px;
  object-fit: contain;
}

.decoration-info {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: 1;
}
.decoration-name {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 14px;
  font-weight: 600;
}
.decoration-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.decoration-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: auto;
}
.decoration-tag {
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 500;
  border-radius: 4px;
  background: var(--bg-muted);
  color: var(--text-muted);
}
.decoration-tag.is-on {
  background: rgba(63, 160, 106, 0.14);
  color: #3fa06a;
}
.decoration-tag.is-off-tag {
  background: rgba(217, 83, 79, 0.12);
  color: #c9302c;
}
.decoration-tag.is-default {
  background: var(--accent-soft, rgba(64, 128, 255, 0.14));
  color: var(--primary-deep);
}
/* 续期档位：绿松色，与「已上架」的绿色、价格/时长的中性底色区分开 */
.decoration-tag.is-renew {
  background: rgba(47, 143, 91, 0.12);
  color: #2f8f5b;
}
.decoration-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

/* ===== 发放 / 回收 ===== */
.grant-tip {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  margin: 14px 0 4px;
  font-size: 13px;
  color: var(--text-muted);
}
.grant-layout {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
  margin-top: 10px;
}
.grant-col {
  display: flex;
  flex-direction: column;
  padding: 12px;
  border: 1px solid var(--border-soft);
  border-radius: 12px;
  background: var(--bg);
}
.grant-col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
}
.grant-count {
  font-size: 12px;
  font-weight: 500;
  color: var(--primary-deep);
}
.grant-chips {
  margin-bottom: 4px;
}
.pick-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 300px;
  overflow-y: auto;
}
.pick-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 4px;
  border-bottom: 1px solid var(--border-soft);
  font-size: 13px;
}
.pick-item:last-child {
  border-bottom: none;
}
.pick-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.pick-avatar.is-empty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-muted);
  color: var(--text-muted);
}
.pick-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pick-sub {
  font-size: 12px;
  color: var(--text-muted);
}
.pick-empty {
  padding: 20px 0;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}
.grant-foot {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
}
.grant-actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

/* ===== 弹窗 ===== */
.modal-overlay {
  position: fixed;
  inset: 0;
  /* 与 AdminFormDialog 保持同一层（3000）：附件库（3500）要能盖在本弹窗之上 */
  z-index: 3000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.5);
}
.modal-card {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 420px;
  max-height: 90vh;
  border-radius: var(--radius-xl, 16px);
  background: var(--bg-card);
  box-shadow: var(--shadow-md);
  overflow: hidden;
}
.modal-card.modal-lg {
  max-width: 640px;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-soft);
}
.modal-title {
  font-size: 16px;
  font-weight: 600;
}
.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
}
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--border-soft);
}
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity 0.2s;
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}

/* ===== 表单 ===== */
.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0 14px;
}
.form-hint {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.45;
}
.required {
  color: #d9534f;
}
.form-item.inline {
  display: flex;
  align-items: center;
  gap: 10px;
}
.form-item.inline .form-label {
  margin: 0;
  white-space: nowrap;
}
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 18px 0 6px;
  padding-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  border-bottom: 1px solid var(--border-soft);
}
.section-preview {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  font-size: 12px;
  font-weight: 400;
  color: var(--text-muted);
}
.preview-frame {
  width: 40px;
  height: 40px;
  object-fit: contain;
}

/* 素材地址下方的操作按钮（上传 / 附件库 / 移除） */
.image-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  /* .form-item 不是 flex 布局，兄弟节点之间没有间距，这里必须自己补 */
  margin-top: 8px;
}

/* ---------- 头像框对齐器 ---------- */
.frame-aligner {
  margin-bottom: 14px;
  padding: 12px 14px;
  background: var(--bg-alt);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.frame-aligner-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
}
.frame-aligner-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text);
}
.frame-aligner-body {
  display: flex;
  align-items: center;
  gap: 16px;
}
.frame-aligner-stage {
  position: relative;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 200px;
  height: 200px;
  overflow: hidden;
  /* 棋盘格底：方便看清素材的透明区域和实际边界 */
  background: repeating-conic-gradient(var(--bg-card) 0% 25%, var(--bg-muted) 0% 50%) 50% / 16px 16px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
}
.frame-aligner-avatar {
  width: 120px;
  height: 120px;
  object-fit: cover;
  border-radius: 50%;
}
.frame-aligner-overlay {
  position: absolute;
  top: 50%;
  left: 50%;
  cursor: grab;
  touch-action: none;
  user-select: none;
  transform-origin: center center;
  /* 同上：素材宽度会超过舞台，不能被全局 img{max-width:100%} 截断 */
  max-width: none;
}
.frame-aligner-overlay:active {
  cursor: grabbing;
}
.frame-aligner-empty {
  margin: 0;
  font-size: 12px;
  color: var(--text-light);
}
.frame-aligner-controls {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.align-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.align-row-label {
  flex-shrink: 0;
  width: 58px;
  font-size: 12px;
  color: var(--text-muted);
}
.align-range {
  flex: 1;
  min-width: 0;
  accent-color: var(--primary);
}
.align-row-value {
  flex-shrink: 0;
  width: 46px;
  font-size: 12px;
  color: var(--text-soft);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 900px) {
  .frame-aligner-body {
    flex-direction: column;
    align-items: stretch;
  }
  .frame-aligner-stage {
    width: 100%;
    max-width: 200px;
    margin: 0 auto;
  }
}
.color-field {
  display: flex;
  align-items: center;
  gap: 8px;
}
.color-pick {
  width: 34px;
  height: 32px;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  cursor: pointer;
  flex-shrink: 0;
}
.switch-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 16px;
  padding: 12px 14px;
  border-radius: 10px;
  background: var(--bg-muted);
}
.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  cursor: pointer;
}
.switch-row input[type='checkbox'] {
  width: 15px;
  height: 15px;
  cursor: pointer;
  accent-color: var(--primary);
}
.advanced {
  margin-top: 14px;
  font-size: 13px;
}
.advanced summary {
  cursor: pointer;
  color: var(--text-soft);
  user-select: none;
}
.advanced summary:hover {
  color: var(--primary);
}
.advanced .textarea {
  margin-top: 8px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
}
</style>
