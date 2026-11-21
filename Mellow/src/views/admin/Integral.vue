<template>
  <div class="integral-admin">
    <!-- 概览：随当前模块切换统计口径 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">积分管理</h2>
          <p class="block-desc">{{ activeView.desc }}</p>
        </div>
        <div class="head-actions">
          <template v-if="view === 'cards'">
            <button class="btn btn-sm" :disabled="busy" @click="exportCards">
              <i class="bi bi-download" /> 导出未使用
            </button>
            <button class="btn btn-sm" @click="openImport">
              <i class="bi bi-upload" /> 自定义导入
            </button>
            <button class="btn btn-primary btn-sm" @click="openGenerate">
              <i class="bi bi-plus-lg" /> 生成卡密
            </button>
          </template>
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
      <!-- 模块切换：用户积分 / 卡密 / 规则 -->
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

      <!-- ============ 1. 用户积分管理 ============ -->
      <template v-if="view === 'users'">
        <header class="list-head">
          <div>
            <h2 class="block-title">用户积分</h2>
            <p class="block-desc">
              调整后余额不能为负（后端会拦截「积分不足」）；批量调整逐个处理，失败不影响其它用户
              <template v-if="userTotal > 0"> · 共 {{ userTotal }} 位</template>
            </p>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in userTabs"
              :key="tab.key"
              type="button"
              class="status-tab"
              :class="{ 'is-active': userFilter === tab.key }"
              @click="switchUserFilter(tab.key)"
            >{{ tab.label }}</button>
          </div>

          <div class="filter-right">
            <FormSelect variant="pill"
              v-model="userSort"
              :options="userSortOptions"
              icon="bi bi-sort-down"
              placeholder="排序方式"
              @change="reloadUsers" />

            <FormSelect variant="pill"
              v-model="userSearchField"
              :options="USER_SEARCH_FIELDS"
              icon="bi bi-funnel"
              placeholder="搜索字段" />

            <div class="search-box">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="userKeyword"
                class="search-input"
                type="search"
                placeholder="搜索用户…"
                aria-label="搜索用户"
                @input="doUserSearch" />
            </div>

            <button class="btn btn-sm" :disabled="userLoading" @click="loadUsers()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <!-- 批量调整 -->
        <div v-if="selectedUids.length" class="batch-bar">
          <span class="batch-count">已选 <strong>{{ selectedUids.length }}</strong> 位用户</span>
          <div class="batch-actions">
            <button class="btn btn-sm btn-primary" :disabled="busy" @click="openBatchAdjust">
              <i class="bi bi-coin" /> 批量调整积分
            </button>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
          </div>
        </div>

        <div v-if="userLoading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!users.length" class="empty-row">
          <EmptyState icon="bi bi-people" :text="userEmptyText" />
        </div>

        <template v-else>
          <div class="list-head-row">
            <label class="pick" title="全选本页">
              <FormCheckbox
                ref="userAllRef"
                :model-value="userPageAllSelected"
                :disabled="busy"
                aria-label="全选本页"
                @change="toggleUserAll" />
            </label>
            <span class="list-head-text">本页 {{ users.length }} 位</span>
          </div>

          <ul class="user-list">
            <li v-for="item in users" :key="item.id" class="user-row" :class="{ 'is-selected': isUserSelected(item.id) }">
              <label class="pick" :title="isUserSelected(item.id) ? '取消选择' : '选择'">
                <FormCheckbox
                  :model-value="isUserSelected(item.id)"
                  :disabled="busy"
                  :aria-label="`选择用户 ${item.nickname || item.id}`"
                  @change="toggleUser(item.id)" />
              </label>

              <div class="user-avatar">
                <img v-if="item.avatar" :src="item.avatar" :alt="item.nickname" loading="lazy" />
                <i v-else class="bi bi-person-circle" />
              </div>

              <div class="user-main">
                <div class="user-name-row">
                  <span class="user-name">{{ item.nickname || '未设置昵称' }}</span>
                  <span class="id-chip">#{{ item.id }}</span>
                </div>
                <div class="user-meta">
                  <span class="meta-text"><i class="bi bi-star" /> 当前积分 {{ item.integral || 0 }}</span>
                  <span class="meta-text"><i class="bi bi-clock" /> 注册于 {{ fromNow(item.create_time) }}</span>
                </div>
              </div>

              <div class="user-integral">
                <i class="bi bi-coin" />
                <span class="integral-value">{{ item.integral || 0 }}</span>
              </div>

              <div class="user-actions">
                <button class="btn btn-primary btn-sm" :disabled="busy" @click="openAdjust(item)">
                  <i class="bi bi-coin" /> 调整积分
                </button>
              </div>
            </li>
          </ul>
        </template>

        <Pagination
          v-if="!userLoading && userTotal > 0"
          :current="userPage"
          :total="userTotal"
          :page-size="userPageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { userPageSize = size; userPage = 1; loadUsers() }"
          @update:current="changeUserPage"
        />
      </template>

      <!-- ============ 2. 积分卡密管理 ============ -->
      <template v-else-if="view === 'cards'">
        <header class="list-head">
          <div>
            <h2 class="block-title">积分卡密</h2>
            <p class="block-desc">
              生成后请及时导出保存；卡密明文仅在生成接口返回一次
              <template v-if="cardTotal > 0"> · 共 {{ cardTotal }} 张</template>
            </p>
          </div>
        </header>

        <div class="list-filter">
          <div class="status-tabs">
            <button
              v-for="tab in cardTabs"
              :key="tab.key"
              type="button"
              class="status-tab"
              :class="{ 'is-active': cardStatus === tab.key }"
              @click="switchCardStatus(tab.key)"
            >{{ tab.label }}</button>
          </div>

          <div class="filter-right">
            <FormSelect variant="pill"
              v-model="cardSort"
              :options="cardSortOptions"
              icon="bi bi-sort-down"
              placeholder="排序方式"
              @change="reloadCards" />

            <div class="search-box">
              <i class="bi bi-search" aria-hidden="true" />
              <FormInput
                v-model="cardKeyword"
                class="search-input"
                type="search"
                placeholder="搜索卡密…"
                aria-label="搜索卡密"
                @input="doCardSearch" />
            </div>

            <button class="btn btn-sm" :disabled="cardLoading" @click="loadCards()">
              <i class="bi bi-arrow-clockwise" /> 刷新
            </button>
          </div>
        </div>

        <!-- 批量操作 -->
        <div v-if="selectedIds.length" class="batch-bar">
          <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 张</span>
          <div class="batch-actions">
            <!-- 批量把选中的未兑换卡密绑定给同一个用户 -->
            <button class="btn btn-sm" :disabled="busy" @click="openBatchBind()">
              <i class="bi bi-person-plus" /> 设置兑换人
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 删除
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
              <i class="bi bi-x-octagon" /> 彻底删除
            </button>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
          </div>
        </div>

        <div v-if="cardLoading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!cards.length" class="empty-row">
          <EmptyState icon="bi bi-credit-card-2-front" :text="cardEmptyText" />
        </div>

        <template v-else>
          <div class="list-head-row">
            <label class="pick" title="全选本页">
              <FormCheckbox
                ref="cardAllRef"
                :model-value="cardPageAllSelected"
                :disabled="busy"
                aria-label="全选本页"
                @change="toggleCardAll" />
            </label>
            <span class="list-head-text">本页 {{ cards.length }} 张</span>
          </div>

          <ul class="card-list">
            <li v-for="item in cards" :key="item.id" class="card-row" :class="{ 'is-selected': isCardSelected(item.id) }">
              <label class="pick" :title="isCardSelected(item.id) ? '取消选择' : '选择'">
                <FormCheckbox
                  :model-value="isCardSelected(item.id)"
                  :disabled="busy || Number(item.status) === 1 || Number(item.status) === 2"
                  :aria-label="`选择卡密 ${item.card}`"
                  @change="toggleCard(item.id)" />
              </label>

              <div class="card-main">
                <div class="card-code-row">
                  <code class="card-code">{{ item.card }}</code>
                  <button class="btn btn-ghost btn-sm" title="复制卡密" aria-label="复制卡密" @click="copyText(item.card)">
                    <i class="bi bi-clipboard" />
                  </button>
                  <span class="state-chip" :class="cardStateClass(item)">{{ cardStateLabel(item) }}</span>
                  <span class="value-chip"><i class="bi bi-coin" /> {{ item.value }}</span>
                </div>
                <div class="card-meta">
                  <span class="meta-text">批次 {{ item.batch }}</span>
                  <span class="meta-text"><i class="bi bi-clock" /> {{ fromNow(item.create_time) }}</span>
                  <span class="meta-text"><i class="bi bi-hourglass-split" /> {{ expireText(item) }}</span>
                  <span v-if="Number(item.status) === 1" class="meta-text">
                    <i class="bi bi-person-check" /> 已被 {{ item.nickname || `用户 ${item.uid}` }} 使用
                  </span>
                  <span v-else-if="Number(item.status) === 2" class="meta-text">
                    <i class="bi bi-person-plus" /> 已发放给 {{ item.nickname || `用户 ${item.uid}` }}（待兑换）
                  </span>
                  <span v-if="item.remark" class="meta-text"><i class="bi bi-sticky" /> {{ item.remark }}</span>
                </div>
              </div>

              <div class="card-actions">
                <!-- 设置 / 改绑兑换人：已兑换的卡密不能改（积分流水已落库） -->
                <button
                  class="btn btn-ghost btn-sm"
                  :title="cardBindTitle(item)"
                  :aria-label="cardBindTitle(item)"
                  :disabled="busy || Number(item.status) === 1"
                  @click="openBind(item)"
                >
                  <i :class="Number(item.uid) > 0 ? 'bi bi-person-gear' : 'bi bi-person-plus'" />
                </button>
                <button
                  class="btn btn-ghost btn-sm danger"
                  :title="Number(item.status) === 2 ? '已发放给用户的卡密不能删除' : '删除'"
                  aria-label="删除"
                  :disabled="busy || Number(item.status) === 2"
                  @click="askRemove(item)"
                >
                  <i class="bi bi-trash" />
                </button>
                <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                  <i class="bi bi-x-octagon" />
                </button>
              </div>
            </li>
          </ul>
        </template>

        <Pagination
          v-if="!cardLoading && cardTotal > 0"
          :current="cardPage"
          :total="cardTotal"
          :page-size="cardPageSize"
          :sizes="PAGE_SIZE_OPTIONS"
          @update:page-size="(size) => { cardPageSize = size; cardPage = 1; loadCards() }"
          @update:current="changeCardPage"
        />
      </template>

      <!-- ============ 3. 积分获取规则管理 ============ -->
      <template v-else>
        <header class="list-head">
          <div>
            <h2 class="block-title">积分获取规则</h2>
            <p class="block-desc">各行为可获得的积分与每日上限，修改后立即影响后续发放</p>
            <p class="block-desc">
              <i class="bi bi-info-circle" />
              每日签到的积分不在这里配置（签到已独立）——它的积分来自
              <router-link to="/admin/goods/checkin" class="stock-link">签到管理</router-link>
              的基础奖励，用户端的「今日任务 / 获取途径」会自动跟随。
            </p>
          </div>
        </header>

        <div v-if="ruleLoading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <div v-else-if="!rules.length" class="empty-row">
          <EmptyState icon="bi bi-coin" text="暂无积分规则" />
        </div>

        <table v-else class="rule-table">
          <thead>
            <tr>
              <th>类型</th>
              <th>名称</th>
              <th class="cell-num">单次积分</th>
              <th class="cell-num">每日上限</th>
              <th class="cell-act">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in rules" :key="r.type">
              <td><code class="rule-type">{{ r.type }}</code></td>
              <td>
                <i v-if="r.icon" :class="`bi ${r.icon}`" class="rule-icon" />
                {{ r.name || '—' }}
              </td>
              <td class="cell-num">{{ r.value }}</td>
              <td class="cell-num">{{ r.daily_limit > 0 ? `${r.daily_limit} 次` : '不限' }}</td>
              <td class="cell-act">
                <button class="btn btn-ghost btn-sm" title="编辑规则" aria-label="编辑规则" @click="openRuleEdit(r)">
                  <i class="bi bi-pencil" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </template>
    </div>

    <!-- 调整积分（单个 / 批量） -->
    <AdminFormDialog
      v-model:visible="adjust.visible"
      :title="adjust.title"
      icon="bi bi-coin"
      :loading="adjust.loading"
      confirm-text="确认调整"
      @confirm="submitAdjust"
    >
      <!-- 单个用户：展示当前余额，便于核对 -->
      <div v-if="adjust.mode === 'single'" class="adjust-user">
        <div class="user-avatar sm">
          <img v-if="adjust.user?.avatar" :src="adjust.user.avatar" :alt="adjust.user.nickname" loading="lazy" />
          <i v-else class="bi bi-person-circle" />
        </div>
        <div class="user-main">
          <div class="user-name-row">
            <span class="user-name">{{ adjust.user?.nickname || '未设置昵称' }}</span>
            <span class="id-chip">#{{ adjust.user?.id }}</span>
          </div>
          <div class="user-meta">
            <span class="meta-text">当前积分 <strong>{{ adjust.user?.integral || 0 }}</strong></span>
          </div>
        </div>
      </div>

      <div v-else class="adjust-user">
        <div class="user-avatar sm multi">
          <i class="bi bi-people" />
        </div>
        <div class="user-main">
          <div class="user-name-row">
            <span class="user-name">已选 {{ adjust.uids.length }} 位用户</span>
          </div>
          <div class="user-meta">
            <span class="meta-text">将为每位用户分别调整相同积分值</span>
          </div>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">积分调整值</label>
        <FormInput v-model="adjust.value" type="number" step="1" placeholder="正数发放，负数扣除" />
        <div class="quick-row">
          <button v-for="n in [10, 50, 100]" :key="`add-${n}`" class="btn btn-sm" type="button" @click="adjust.value = n">
            +{{ n }}
          </button>
          <button v-for="n in [10, 50, 100]" :key="`sub-${n}`" class="btn btn-sm" type="button" @click="adjust.value = -n">
            -{{ n }}
          </button>
        </div>
        <p class="form-hint">
          正数发放、负数扣除；扣除超过用户当前余额时后端会拒绝并提示「积分不足」
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">说明</label>
        <FormInput v-model="adjust.description" type="text" placeholder="可选，会记入用户积分流水" />
      </div>
    </AdminFormDialog>

    <!-- 生成卡密 -->
    <AdminFormDialog
      v-model:visible="gen.visible"
      title="生成卡密"
      icon="bi bi-plus-circle"
      :loading="gen.loading"
      confirm-text="生成"
      @confirm="submitGenerate"
    >
      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">积分面额</label>
          <FormInput v-model="gen.value" type="number" min="1" step="1" placeholder="每张卡密兑换的积分" />
        </div>
        <div class="form-item">
          <label class="form-label">生成数量</label>
          <FormInput v-model="gen.count" type="number" min="1" max="1000" step="1" placeholder="1 ~ 1000" />
        </div>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">卡密长度</label>
          <FormInput v-model="gen.length" type="number" min="8" max="64" step="1" placeholder="默认 16" />
        </div>
        <div class="form-item">
          <label class="form-label">有效期</label>
          <FormInput v-model="gen.expire" type="date" />
          <p class="form-hint">留空表示永久有效</p>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="gen.remark" type="text" maxlength="255" placeholder="可选，便于区分批次用途" />
      </div>
    </AdminFormDialog>

    <!-- 自定义导入卡密（卡密内容自己填，不随机生成） -->
    <AdminFormDialog
      v-model:visible="imp.visible"
      title="自定义导入卡密"
      icon="bi bi-upload"
      :loading="imp.loading"
      confirm-text="导入"
      width="560px"
      @confirm="submitImport"
    >
      <div class="form-item">
        <label class="form-label">卡密内容</label>
        <FormTextarea
          v-model="imp.codes"
          rows="8"
          placeholder="一行一个，例如：&#10;SN2026000001&#10;SN2026000002" />
        <p class="form-hint">
          每行一个，长度 8 ~ 64 位；会自动去掉空格与连字符并转成大写（与用户兑换时的输入规则一致），
          与系统里已有卡密重复的会被跳过，单次最多 1000 张。
        </p>
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">积分面额</label>
          <FormInput v-model="imp.value" type="number" min="1" step="1" placeholder="每张卡密兑换的积分" />
        </div>
        <div class="form-item">
          <label class="form-label">有效期</label>
          <FormInput v-model="imp.expire" type="date" />
          <p class="form-hint">留空表示永久有效</p>
        </div>
      </div>

      <div class="form-item">
        <label class="form-label">备注</label>
        <FormInput v-model="imp.remark" type="text" maxlength="255" placeholder="可选，便于区分批次用途" />
      </div>
    </AdminFormDialog>

    <!-- 设置 / 改绑卡密兑换人 -->
    <AdminFormDialog
      v-model:visible="bind.visible"
      title="设置兑换人"
      icon="bi bi-person-plus"
      width="460px"
      :loading="bind.loading"
      confirm-text="保存"
      @confirm="submitBind"
    >
      <div class="form-item">
        <label class="form-label">操作</label>
        <FormSelect v-model="bind.mode" :options="bindModeOptions" />
      </div>

      <template v-if="bind.mode === 'bind'">
        <div class="form-item">
          <label class="form-label">兑换人用户 ID</label>
          <FormInput
            v-model="bind.uid"
            type="text"
            inputmode="numeric"
            placeholder="请输入用户 ID（可在「用户管理」中查看）"
            @input="onBindUidInput" />
          <p class="form-hint">绑定后只有该用户能兑换这些卡密，其他用户兑换会提示「卡密已被使用」</p>
        </div>

        <!-- 目标用户回显：确认选对人 -->
        <div class="bind-user">
          <template v-if="bind.user">
            <img v-if="bind.user.avatar" :src="bind.user.avatar" class="bind-avatar" alt="头像" loading="lazy" />
            <i v-else class="bi bi-person-circle bind-avatar is-icon" />
            <span class="bind-user-name">{{ bind.user.nickname || bind.user.account }}</span>
            <span v-if="bind.user.account" class="bind-user-account">{{ bind.user.account }}</span>
          </template>
          <span v-else-if="bind.checking" class="bind-user-hint">查询中...</span>
          <span v-else-if="bind.checked" class="bind-user-hint is-error">
            <i class="bi bi-exclamation-circle" /> 没有找到该用户，请检查用户 ID
          </span>
          <span v-else class="bind-user-hint">输入用户 ID 后会自动校验</span>
        </div>
      </template>

      <p v-if="bindSummary" class="dialog-tip">{{ bindSummary }}</p>
    </AdminFormDialog>

    <!-- 生成结果 / 导出结果 -->
    <AdminFormDialog
      v-model:visible="result.visible"
      :title="result.title"
      icon="bi bi-clipboard-check"
      :loading="false"
      confirm-text="关闭"
      width="560px"
      @confirm="result.visible = false"
    >
      <p class="dialog-tip">{{ result.tip }}</p>
      <!-- 导入时被跳过的卡密明细（重复 / 长度不合法等） -->
      <ul v-if="result.details.length" class="result-details">
        <li v-for="(item, index) in result.details" :key="index">{{ item }}</li>
      </ul>
      <textarea ref="resultRef" v-model="result.text" class="textarea" rows="8" readonly />
      <div class="result-actions">
        <button class="btn btn-sm" @click="copyText(result.text)">
          <i class="bi bi-clipboard" /> 复制全部
        </button>
      </div>
    </AdminFormDialog>

    <!-- 编辑积分规则 -->
    <AdminFormDialog
      v-model:visible="rule.visible"
      :title="`编辑规则：${rule.name || rule.type}`"
      icon="bi bi-pencil-square"
      :loading="rule.loading"
      confirm-text="保存"
      @confirm="saveRule"
    >
      <div class="form-item">
        <label class="form-label">规则名称</label>
        <FormInput v-model="rule.name" type="text" maxlength="32" placeholder="如「每日签到」" />
      </div>

      <div class="form-grid">
        <div class="form-item">
          <label class="form-label">单次积分</label>
          <FormInput v-model="rule.value" type="number" min="0" step="1" placeholder="每次动作获得的积分" />
        </div>
        <div class="form-item">
          <label class="form-label">每日上限</label>
          <FormInput v-model="rule.limit" type="number" min="0" step="1" placeholder="0 表示不限" />
          <p class="form-hint">0 表示不限制次数</p>
        </div>
      </div>

      <p class="dialog-tip">
        规则类型 <strong>{{ rule.type }}</strong> 不可修改；保存只覆盖该类型，其余类型保持原样。
      </p>
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
 * 积分管理（/admin/integral）
 *
 * 三个模块：
 *  1. 用户积分管理 —— 用户列表 + 单个 / 批量调整积分（integral/give）
 *  2. 积分卡密管理 —— 卡密生成、导出、删除（integral/card-*）
 *  3. 积分获取规则 —— 各行为积分与每日上限（integral/rules + 配置表 SYSTEM_INTEGRAL_RULES）
 *
 * 后端约束（app/api/controller/integral.go、app/model/integral*.go）：
 * - integral/give 需权限点，支持 uid（单个）或 uids（数组批量），value 不能为 0；
 *   扣除时后端校验余额（不足返回「积分不足！」），批量时逐个处理、失败不影响其它用户；
 * - integral/all 只返回「当前登录用户」的流水，后端无全站 / 指定用户的流水接口，
 *   因此后台不提供积分流水视图；
 * - 用户列表走 users/all：order 未做白名单校验由前端限定，where 支持 $gt / $lte 等运算符；
 * - card-generate：面额 1~1000000、数量 1~1000、长度 8~64（默认 16）、有效期必须晚于当前时间，
 *   卡密明文仅在该接口返回一次；
 * - card-all 仅返回未软删除的卡密（不支持 onlyTrashed），已使用的卡密不允许删除。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import { listUsers, countUsers, listUsersByIds } from '@/api/users'
import {
  listIntegralCards,
  getIntegralCardStats,
  generateIntegralCards,
  importIntegralCards,
  bindIntegralCards,
  exportIntegralCards,
  removeIntegralCards,
  forceDeleteIntegralCards,
  getIntegralRules,
  getIntegralConfig,
  saveIntegralConfig,
  getIntegralRank,
  giveIntegral
} from '@/api/integral'
import { USER_SEARCH_FIELDS, sanitizeUserKeyword } from '@/utils/user'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'

// 用户积分与积分卡密是两个列表，各自记住自己的每页条数
const userPageSize = ref(PAGE_SIZE_DEFAULT)
const cardPageSize = ref(PAGE_SIZE_DEFAULT)

const views = [
  {
    key: 'users',
    label: '用户积分',
    icon: 'bi bi-people',
    desc: '查询用户积分余额并直接发放 / 扣除，支持批量调整'
  },
  {
    key: 'cards',
    label: '积分卡密',
    icon: 'bi bi-credit-card-2-front',
    desc: '批量生成卡密供用户兑换，支持导出与回收'
  },
  {
    key: 'rules',
    label: '获取规则',
    icon: 'bi bi-coin',
    desc: '配置发文、评论、登录等行为可获得的积分与每日上限（签到积分见「签到管理」）'
  }
]

// 用户积分筛选（where 由后端 IWhere 解析，$gt / 等值均支持）
const userTabs = [
  { key: 'all', label: '全部', where: null },
  { key: 'has', label: '有积分', where: { integral: { $gt: 0 } } },
  { key: 'zero', label: '零积分', where: { integral: 0 } }
]

// 排序白名单：后端 users/all 的 order 未校验，必须由前端限定
const userSortOptions = [
  { value: 'integral desc', label: '积分最多' },
  { value: 'integral asc', label: '积分最少' },
  { value: 'id desc', label: '最新注册' },
  { value: 'id asc', label: '最早注册' },
  { value: 'login_time desc', label: '最近登录' }
]

// 卡密状态筛选（后端用 status + expired 两个参数表达）
// granted = 已发放：活动（如签到）发给用户的卡密，已绑定用户、等他自己兑换
const cardTabs = [
  { key: 'all', label: '全部' },
  { key: 'unused', label: '未使用' },
  { key: 'granted', label: '已发放' },
  { key: 'used', label: '已使用' },
  { key: 'expired', label: '已过期' }
]

const cardSortOptions = [
  { value: 'id desc', label: '最新生成' },
  { value: 'id asc', label: '最早生成' },
  { value: 'value desc', label: '面额从高到低' },
  { value: 'value asc', label: '面额从低到高' }
]

// ===== 当前模块 =====
const view = ref('users')
const activeView = computed(() => views.find((v) => v.key === view.value) || views[0])

// ===== 用户积分 =====
// 只取列表展示需要的字段，避免拉取 result / json 等大字段
const USER_FIELD = 'id,nickname,avatar,integral,exp,login_time,create_time'
const users = ref([])
const userTotal = ref(0)
const userPage = ref(1)
const userLoading = ref(false)
const busy = ref(false)
const userFilter = ref('all')
const userSort = ref(userSortOptions[0].value)
const userSearchField = ref('nickname')
const userKeyword = ref('')
const userSearchKey = ref('')
const selectedUids = ref([])
const userStats = reactive({ total: 0, hasIntegral: 0, topBalance: 0, topEarned: 0 })

// ===== 卡密 =====
const cards = ref([])
const cardTotal = ref(0)
const cardPage = ref(1)
const cardLoading = ref(false)
const cardStatus = ref('all')
const cardSort = ref(cardSortOptions[0].value)
const cardKeyword = ref('')
const cardSearchKey = ref('')
const selectedIds = ref([])
const cardStats = ref(null)

// ===== 规则 =====
const rules = ref([])
const ruleLoading = ref(false)

// ===== 统计 =====
const loadingStats = ref(false)

const userStatCards = computed(() => [
  { label: '全部用户', value: userStats.total, icon: 'bi bi-people', color: 'var(--primary)' },
  { label: '有积分用户', value: userStats.hasIntegral, icon: 'bi bi-coin', color: 'var(--success)' },
  { label: '零积分用户', value: Math.max(0, userStats.total - userStats.hasIntegral), icon: 'bi bi-dash-circle', color: 'var(--text-muted)' }
])

const cardStatCards = computed(() => {
  const s = cardStats.value || {}
  return [
    { label: '卡密总数', value: s.total ?? 0, icon: 'bi bi-credit-card-2-front', color: 'var(--primary)' },
    { label: '未使用「卡密」', value: s.unused ?? 0, icon: 'bi bi-patch-check', color: 'var(--success)' },
    { label: '已发放「卡密」', value: s.granted ?? 0, icon: 'bi bi-ticket-perforated', color: '#8b5cf6' },
    { label: '已使用「卡密」', value: s.used ?? 0, icon: 'bi bi-check2-circle', color: '#0ea5e9' },
    { label: '已过期「卡密」', value: s.expired ?? 0, icon: 'bi bi-hourglass-split', color: 'var(--warning)' },
    { label: '累计发放「积分」', value: s.value_total ?? 0, icon: 'bi bi-coin', color: 'var(--primary)' },
    { label: '已兑换', value: s.value_used ?? 0, icon: 'bi bi-cart-check', color: 'var(--success)' }
  ]
})

const ruleStatCards = computed(() => {
  const list = rules.value
  // 每日上限合计：只统计有次数上限的规则（不限次数的无法估算）
  const daily = list.reduce(
    (sum, r) => sum + (Number(r.daily_limit) > 0 ? Number(r.daily_limit) * Number(r.value || 0) : 0),
    0
  )
  const maxOne = list.reduce((max, r) => Math.max(max, Number(r.value || 0)), 0)
  const unlimited = list.filter((r) => Number(r.daily_limit) <= 0).length
  return [
    { label: '规则数量', value: list.length, icon: 'bi bi-list-check', color: 'var(--primary)' },
    { label: '每日上限合计', value: daily, icon: 'bi bi-calendar-day', color: 'var(--success)' },
    { label: '单次最高积分', value: maxOne, icon: 'bi bi-coin', color: 'var(--warning)' },
    { label: '不限次数规则', value: unlimited, icon: 'bi bi-infinity', color: '#0ea5e9' }
  ]
})

const statCards = computed(() => {
  if (view.value === 'cards') return cardStatCards.value
  if (view.value === 'rules') return ruleStatCards.value
  return userStatCards.value
})

// ===== 弹窗 =====
const adjust = reactive({
  visible: false,
  loading: false,
  mode: 'single', // single | batch
  user: null,
  uids: [],
  value: '',
  description: '',
  title: '调整积分'
})

// 自定义导入卡密（卡密内容由管理员自己填写，不随机生成）
const imp = reactive({
  visible: false,
  loading: false,
  codes: '',
  value: 100,
  expire: '',
  remark: ''
})

const gen = reactive({
  visible: false,
  loading: false,
  value: 100,
  count: 10,
  length: 16,
  expire: '',
  remark: ''
})

// 设置 / 改绑卡密兑换人（卡片可绑定给指定用户，也可解除绑定）
const bind = reactive({
  visible: false,
  loading: false,
  // bind：绑定 / 改绑给指定用户；unbind：解除绑定（退回未使用）
  mode: 'bind',
  uid: '',
  user: null,
  checking: false,
  checked: false,
  ids: []
})

const bindModeOptions = [
  { value: 'bind', label: '绑定 / 改绑给指定用户' },
  { value: 'unbind', label: '解除绑定（退回未使用）' }
]

// 可设置兑换人的选中卡密：已兑换（status=1）的不能改，流水已落库
const bindableSelectedIds = computed(() =>
  cards.value
    .filter((item) => isCardSelected(item.id) && Number(item.status) !== 1)
    .map((item) => Number(item.id))
)

// 弹窗底部说明：这次操作会影响哪些卡密（张数 / 总面额 / 是否涉及改绑）
const bindSummary = computed(() => {
  const target = cards.value.filter((item) => bind.ids.includes(Number(item.id)))
  if (!target.length) return ''

  const total = target.reduce((sum, item) => sum + Number(item.value || 0), 0)
  const granted = target.filter((item) => Number(item.status) === 2)

  if (bind.mode === 'unbind') {
    return granted.length
      ? `将解除 ${granted.length} 张已发放卡密的绑定，退回「未使用」（之后任何用户都能兑换）`
      : '选中的卡密本来就没有绑定用户，无需解绑'
  }

  let text =
    target.length > 1 ? `共 ${target.length} 张卡密，合计 ${total} 积分` : `面额 ${total} 积分`
  if (granted.length) {
    const names = granted
      .map((item) => item.nickname || `用户 ${item.uid}`)
      .slice(0, 3)
      .join('、')
    text += `；其中 ${granted.length} 张已发放给 ${names}，保存后会改绑`
  }
  return text
})

const result = reactive({
  visible: false,
  title: '生成结果',
  tip: '',
  // details：导入卡密时被跳过的卡密明细（生成 / 导出结果为空数组）
  details: [],
  text: ''
})
const resultRef = ref(null)

const rule = reactive({
  visible: false,
  loading: false,
  type: '',
  name: '',
  value: 0,
  limit: 0
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

const userEmptyText = computed(() => {
  if (userSearchKey.value) return '没有匹配的用户'
  if (userFilter.value === 'has') return '暂无有积分的用户'
  if (userFilter.value === 'zero') return '暂无零积分用户'
  return '暂无用户'
})

const cardEmptyText = computed(() => {
  if (cardSearchKey.value) return '没有匹配的卡密'
  if (cardStatus.value !== 'all') return '该状态下暂无卡密'
  return '还没有卡密，先生成一批吧'
})

// ---------- 展示辅助 ----------
function isExpired(item) {
  const expire = Number(item?.expire_time || 0)
  return expire > 0 && expire < Math.floor(Date.now() / 1000)
}

function cardStateClass(item) {
  if (Number(item?.status) === 1) return 'is-used'
  if (Number(item?.status) === 2) return 'is-granted'
  return isExpired(item) ? 'is-expired' : 'is-unused'
}

function cardStateLabel(item) {
  if (Number(item?.status) === 1) return '已使用'
  // 已发放（活动奖励）：已绑定用户，等他到「我的积分 → 卡密兑换」兑换
  if (Number(item?.status) === 2) return '已发放（待兑换）'
  return isExpired(item) ? '已过期' : '未使用'
}

function expireText(item) {
  const expire = Number(item?.expire_time || 0)
  return expire > 0 ? `${fromNow(expire)}过期` : '永久有效'
}

async function copyText(text) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(String(text))
    toast.success('已复制到剪贴板')
  } catch {
    // 降级：选中文本让用户手动复制
    resultRef.value?.select?.()
    toast.info('已选中内容，请手动复制')
  }
}

// ---------- 统计加载 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    if (view.value === 'users') await loadUserStats()
    else if (view.value === 'cards') {
      const res = await getIntegralCardStats()
      cardStats.value = res?.data || null
    } else await loadRules()
  } finally {
    loadingStats.value = false
  }
}

async function loadUserStats() {
  const [totalRes, hasRes, balanceRes, earnedRes] = await Promise.all([
    countUsers(),
    countUsers({ where: JSON.stringify({ integral: { $gt: 0 } }) }),
    getIntegralRank({ by: 'balance', limit: 1 }),
    getIntegralRank({ by: 'earned', limit: 1 })
  ])
  userStats.total = Number(totalRes?.data || 0)
  userStats.hasIntegral = Number(hasRes?.data || 0)
  userStats.topBalance = Number(balanceRes?.data?.list?.[0]?.value || 0)
  userStats.topEarned = Number(earnedRes?.data?.list?.[0]?.value || 0)
}

// ---------- 用户加载 ----------
async function loadUsers() {
  userLoading.value = true
  try {
    const params = {
      page: userPage.value,
      limit: userPageSize.value,
      field: USER_FIELD,
      order: userSort.value
    }
    const tab = userTabs.find((t) => t.key === userFilter.value)
    if (tab?.where) params.where = JSON.stringify(tab.where)

    // 关键词走后端 like 的「字段名|值」格式
    const kw = sanitizeUserKeyword(userSearchKey.value)
    if (kw) {
      const field = USER_SEARCH_FIELDS.some((f) => f.value === userSearchField.value)
        ? userSearchField.value
        : 'nickname'
      params.like = `${field}|${kw}`
    }

    const res = await listUsers(params)
    users.value = res.data?.data || []
    userTotal.value = res.data?.count || 0
    clearSelection()
  } catch {
    users.value = []
    userTotal.value = 0
  } finally {
    userLoading.value = false
  }
}

function reloadUsers() {
  userPage.value = 1
  loadUsers()
}

const doUserSearch = debounce(() => {
  userSearchKey.value = userKeyword.value
  reloadUsers()
}, 350)

async function switchUserFilter(key) {
  if (userFilter.value === key) return
  userFilter.value = key
  userKeyword.value = ''
  userSearchKey.value = ''
  reloadUsers()
}

function changeUserPage(p) {
  if (p < 1 || p === userPage.value) return
  userPage.value = p
  loadUsers()
}

// ---------- 用户选择 ----------
const userSelectedSet = computed(() => new Set(selectedUids.value))
const userPageAllSelected = computed(
  () => users.value.length > 0 && users.value.every((i) => userSelectedSet.value.has(i.id))
)
const userSomeSelected = computed(() => selectedUids.value.length > 0 && !userPageAllSelected.value)

const userAllRef = ref(null)

watchEffect(() => {
  if (userAllRef.value) userAllRef.value.setIndeterminate(userSomeSelected.value)
})

function isUserSelected(id) {
  return userSelectedSet.value.has(id)
}

function toggleUser(id) {
  const next = new Set(selectedUids.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedUids.value = [...next]
}

function toggleUserAll() {
  selectedUids.value = userPageAllSelected.value ? [] : users.value.map((i) => i.id)
}

function clearSelection() {
  selectedUids.value = []
  selectedIds.value = []
}

// ---------- 调整积分 ----------
function openAdjust(item) {
  adjust.mode = 'single'
  adjust.user = {
    id: Number(item.id),
    nickname: item.nickname,
    avatar: item.avatar,
    integral: Number(item.integral || 0)
  }
  adjust.uids = []
  adjust.value = ''
  adjust.description = ''
  adjust.title = `调整积分：${item.nickname || '未设置昵称'}`
  adjust.visible = true
}

function openBatchAdjust() {
  adjust.mode = 'batch'
  adjust.user = null
  adjust.uids = [...selectedUids.value]
  adjust.value = ''
  adjust.description = ''
  adjust.title = `批量调整积分（${adjust.uids.length} 位用户）`
  adjust.visible = true
}

async function submitAdjust() {
  const value = Number(String(adjust.value).trim())
  if (!Number.isInteger(value) || value === 0) {
    toast.warning('积分调整值必须为不为 0 的整数（正数发放，负数扣除）')
    return
  }

  const ids = adjust.mode === 'single' ? [adjust.user?.id] : [...adjust.uids]
  const uidList = ids.map((i) => Number(i)).filter((n) => Number.isInteger(n) && n > 0)
  if (!uidList.length) {
    toast.warning('请选择有效的用户')
    return
  }

  adjust.loading = true
  try {
    const payload = { value, description: adjust.description.trim() }
    if (uidList.length === 1) payload.uid = uidList[0]
    else payload.uids = uidList

    const res = await giveIntegral(payload)
    adjust.visible = false

    if (uidList.length === 1) {
      // 单个调整返回最新余额，便于确认结果
      const balance = Number(res?.data?.integral || 0)
      toast.success(value > 0 ? `已发放 ${value} 积分，当前余额 ${balance}` : `已扣除 ${Math.abs(value)} 积分，当前余额 ${balance}`)
    } else {
      const failed = Number(res?.data?.failed || 0)
      toast.success(`批量调整完成：成功 ${res?.data?.success || 0} 个${failed ? `，失败 ${failed} 个` : ''}`)
    }

    clearSelection()
    await Promise.all([loadUsers(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出（如用户不存在、积分不足、无权限）
  } finally {
    adjust.loading = false
  }
}

// ---------- 卡密加载 ----------
async function loadCards() {
  cardLoading.value = true
  try {
    const params = { page: cardPage.value, limit: cardPageSize.value, order: cardSort.value }
    switch (cardStatus.value) {
      case 'unused':
        params.status = 0
        params.expired = '0'
        break
      case 'granted':
        // 已发放（活动奖励），可能已过期，这里不过滤有效期
        params.status = 2
        break
      case 'used':
        params.status = 1
        break
      case 'expired':
        params.status = 0
        params.expired = '1'
        break
    }
    const kw = cardSearchKey.value.trim()
    if (kw) params.keyword = kw
    const res = await listIntegralCards(params)
    cards.value = res.data?.data || []
    cardTotal.value = res.data?.count || 0
    selectedIds.value = []
  } catch {
    cards.value = []
    cardTotal.value = 0
  } finally {
    cardLoading.value = false
  }
}

function reloadCards() {
  cardPage.value = 1
  loadCards()
}

const doCardSearch = debounce(() => {
  cardSearchKey.value = cardKeyword.value
  reloadCards()
}, 350)

async function switchCardStatus(key) {
  if (cardStatus.value === key) return
  cardStatus.value = key
  cardKeyword.value = ''
  cardSearchKey.value = ''
  reloadCards()
}

function changeCardPage(p) {
  if (p < 1 || p === cardPage.value) return
  cardPage.value = p
  loadCards()
}

// ---------- 卡密选择 ----------
const cardSelectedSet = computed(() => new Set(selectedIds.value))
const cardPageAllSelected = computed(
  () => cards.value.length > 0 && cards.value.every((i) => cardSelectedSet.value.has(i.id))
)
const cardSomeSelected = computed(() => selectedIds.value.length > 0 && !cardPageAllSelected.value)

const cardAllRef = ref(null)

watchEffect(() => {
  if (cardAllRef.value) cardAllRef.value.setIndeterminate(cardSomeSelected.value)
})

function isCardSelected(id) {
  return cardSelectedSet.value.has(id)
}

function toggleCard(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleCardAll() {
  // 已使用的卡密属于兑换凭证，不允许批量删除
  const pickable = cards.value.filter((i) => Number(i.status) !== 1).map((i) => i.id)
  selectedIds.value =
    pickable.length && pickable.every((id) => cardSelectedSet.value.has(id)) ? [] : pickable
}

// ---------- 生成卡密 ----------
function openGenerate() {
  gen.value = 100
  gen.count = 10
  gen.length = 16
  gen.expire = ''
  gen.remark = ''
  gen.visible = true
}

// 正整数校验
function positiveInt(raw) {
  const num = Number(String(raw ?? '').trim())
  return Number.isInteger(num) && num > 0 ? num : null
}

async function submitGenerate() {
  const value = positiveInt(gen.value)
  if (value === null) {
    toast.warning('积分面额必须为大于 0 的整数')
    return
  }
  const count = positiveInt(gen.count)
  if (count === null) {
    toast.warning('生成数量必须为大于 0 的整数')
    return
  }
  if (count > 1000) {
    toast.warning('单次最多生成 1000 张')
    return
  }
  const length = positiveInt(gen.length)
  if (length === null) {
    toast.warning('卡密长度必须为大于 0 的整数')
    return
  }

  gen.loading = true
  try {
    const res = await generateIntegralCards({
      value,
      count,
      length,
      // 日期字符串由后端解析为当天 23:59:59；留空表示永久有效
      expire: gen.expire || '',
      remark: gen.remark.trim()
    })
    const list = res?.data?.cards || []
    gen.visible = false
    result.title = `生成成功（批次 ${res?.data?.batch || '-'}）`
    result.tip = '卡密明文仅返回这一次，请立即复制保存；关闭后无法再次查看。'
    result.details = []
    result.text = list.join('\n')
    result.visible = true
    toast.success(`已生成 ${list.length} 张卡密`)
    await Promise.all([loadCards(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出（如面额超限、有效期早于当前时间）
  } finally {
    gen.loading = false
  }
}

// ---------- 自定义导入卡密 ----------
function openImport() {
  imp.codes = ''
  imp.value = 100
  imp.expire = ''
  imp.remark = ''
  imp.visible = true
}

/** 按行拆出卡密（去掉空行，供前端校验数量；后端也会再解析一次） */
function importCodes() {
  return String(imp.codes || '')
    .split(/[\n\r,，;；]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

async function submitImport() {
  const value = positiveInt(imp.value)
  if (value === null) {
    toast.warning('积分面额必须为大于 0 的整数')
    return
  }

  const codes = importCodes()
  if (!codes.length) {
    toast.warning('请先填写卡密内容，每行一个')
    return
  }
  if (codes.length > 1000) {
    toast.warning('单次最多导入 1000 张，请分批导入')
    return
  }

  imp.loading = true
  try {
    const res = await importIntegralCards({
      codes,
      value,
      // 日期字符串由后端解析为当天 23:59:59；留空表示永久有效
      expire: imp.expire || '',
      remark: imp.remark.trim()
    })

    const data = res?.data || {}
    const report = data.report || {}
    const skipped = Array.isArray(report.skipped) ? report.skipped : []

    imp.visible = false
    result.title = `导入成功 ${data.count || 0} 张（批次 ${data.batch || '-'}）`
    result.tip = skipped.length
      ? `成功导入 ${data.count || 0} 张；跳过 ${report.skipped_count || skipped.length} 张（明细见下方，重复或格式不合法）：`
      : `成功导入 ${data.count || 0} 张，没有重复或不合法的卡密。`
    result.details = skipped.map((item) => `${item.code}：${item.reason}`)
    result.text = (data.cards || []).join('\n')
    result.visible = true

    toast.success(`已导入 ${data.count || 0} 张卡密`)
    await Promise.all([loadCards(), loadStats()])
  } catch {
    // 失败提示由请求拦截器统一给出（如所有卡密都重复、超过单次上限）
  } finally {
    imp.loading = false
  }
}

// ---------- 导出卡密 ----------
async function exportCards() {
  busy.value = true
  try {
    const res = await exportIntegralCards()
    const list = res?.data?.cards || []
    if (!list.length) {
      toast.info('暂无可导出的未使用卡密')
      return
    }
    result.title = `导出未使用卡密（${list.length} 张）`
    result.tip = res?.data?.truncated
      ? '数量超过单次导出上限，已截断；可通过筛选分批导出。'
      : '仅包含未使用且未过期的卡密。'
    result.details = []
    result.text = list.join('\n')
    result.visible = true
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

// ---------- 积分规则 ----------
async function loadRules() {
  ruleLoading.value = true
  try {
    const res = await getIntegralRules()
    const list = Array.isArray(res?.data) ? res.data : []
    // 签到已独立成模块（积分由「签到管理」的基础奖励决定），这条虚拟任务不在这里编辑，
    // 否则保存规则时会把它写回 SYSTEM_INTEGRAL_RULES
    rules.value = list.filter((item) => item?.source !== 'checkin')
  } catch {
    rules.value = []
  } finally {
    ruleLoading.value = false
  }
}

function openRuleEdit(item) {
  rule.type = item.type || ''
  rule.name = item.name || ''
  rule.value = Number(item.value || 0)
  rule.limit = Number(item.daily_limit || 0)
  rule.visible = true
}

async function saveRule() {
  const name = rule.name.trim()
  if (!name) {
    toast.warning('请输入规则名称')
    return
  }
  const value = Number(String(rule.value).trim())
  if (!Number.isInteger(value) || value < 0) {
    toast.warning('单次积分必须为不小于 0 的整数')
    return
  }
  const limit = Number(String(rule.limit).trim())
  if (!Number.isInteger(limit) || limit < 0) {
    toast.warning('每日上限必须为不小于 0 的整数')
    return
  }

  rule.loading = true
  try {
    // 以 config 中的原始配置为基底，只覆盖当前编辑的类型
    const res = await getIntegralConfig()
    const raw = res?.data?.json
    const config = typeof raw === 'string' ? JSON.parse(raw) : { ...(raw || {}) }

    rules.value.forEach((r) => {
      if (!config[r.type]) {
        config[r.type] = { name: r.name, value: Number(r.value || 0), daily_limit: Number(r.daily_limit || 0) }
      }
    })

    config[rule.type] = { ...config[rule.type], name, value, daily_limit: limit }

    await saveIntegralConfig(config)
    toast.success('规则已保存')
    rule.visible = false
    await loadRules()
  } catch {
    // 失败提示由请求拦截器统一给出
  } finally {
    rule.loading = false
  }
}

// ---------- 卡密删除 ----------
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

// ---------- 设置 / 改绑卡密兑换人 ----------

// 行内按钮文案：已兑换的卡密不能改绑
function cardBindTitle(item) {
  if (Number(item.status) === 1) return '已兑换的卡密不能改绑'
  return Number(item.uid) > 0 ? '改绑兑换人' : '设置兑换人'
}

// 打开弹窗时的公共准备：复用上次填的用户 ID，但每次都重新校验一次，
// 避免用到过期 / 已被改名的用户信息
function prepareBind() {
  bind.user = null
  bind.checked = false
  bind.checking = false
  if (Number(bind.uid) > 0) onBindUidInput()
}

function openBind(item) {
  bind.ids = [Number(item.id)]
  bind.mode = 'bind'
  prepareBind()
  bind.visible = true
}

// 批量设置兑换人：只针对选中的、未兑换的卡密
function openBatchBind() {
  const ids = bindableSelectedIds.value
  if (!ids.length) {
    toast.warning('选中的卡密都已兑换，不能设置兑换人')
    return
  }
  bind.ids = ids
  bind.mode = 'bind'
  prepareBind()
  bind.visible = true
}

// 输入用户 ID 后自动校验（防抖），确认选对人再提交
let bindUidTimer = null
function onBindUidInput() {
  clearTimeout(bindUidTimer)
  bind.user = null
  bind.checked = false

  const uid = Number(bind.uid)
  if (!uid) {
    bind.checking = false
    return
  }

  bind.checking = true
  bindUidTimer = setTimeout(async () => {
    try {
      const res = await listUsersByIds([uid])
      bind.user = (res.data?.data || [])[0] || null
    } catch {
      bind.user = null
    } finally {
      bind.checking = false
      bind.checked = true
    }
  }, 350)
}

async function submitBind() {
  if (!bind.ids.length) return

  const uid = bind.mode === 'unbind' ? 0 : Number(bind.uid)
  if (bind.mode === 'bind') {
    if (!uid) {
      toast.warning('请输入兑换人的用户 ID')
      return
    }
    if (!bind.user) {
      toast.warning('没有查询到该用户，请检查用户 ID')
      return
    }
  }

  bind.loading = true
  try {
    const res = await bindIntegralCards(bind.ids, uid)
    toast.success(res.msg || '已保存')
    bind.visible = false
    clearSelection()
    await Promise.all([loadCards(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    bind.loading = false
  }
}

function askRemove(item) {
  openConfirm({
    title: '删除卡密',
    message: `确定删除卡密 ${item.card} 吗？删除后将从列表移除（已使用的卡密不可删除，避免丢失兑换凭证）。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeIntegralCards([item.id])
      toast.success('已删除')
      await Promise.all([loadCards(), loadStats()])
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除卡密 ${item.card} 吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteIntegralCards([item.id])
      toast.success('已彻底删除')
      await Promise.all([loadCards(), loadStats()])
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 张卡密吗？`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeIntegralCards(ids)
      toast.success(`已删除 ${ids.length} 张`)
      selectedIds.value = []
      await Promise.all([loadCards(), loadStats()])
    }
  })
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 张卡密吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteIntegralCards(ids)
      toast.success(`已彻底删除 ${ids.length} 张`)
      selectedIds.value = []
      await Promise.all([loadCards(), loadStats()])
    }
  })
}

// ---------- 模块切换 ----------
async function switchView(key) {
  if (view.value === key) return
  view.value = key
  if (key === 'users' && !users.value.length) await loadUsers()
  if (key === 'cards' && !cards.value.length) await loadCards()
  if (key === 'rules' && !rules.value.length) await loadRules()
  await loadStats()
}

onMounted(async () => {
  await Promise.all([loadUsers(), loadStats()])
})
</script>

<style scoped>
.integral-admin {
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
/* 跳转到其它配置页的链接（如「签到管理」） */
.stock-link {
  margin: 0 2px;
  color: var(--primary);
}
.stock-link:hover {
  text-decoration: underline;
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.stat-grid {
  display: grid;
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

/* ---------- 模块切换 ---------- */
.view-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.view-tabs .status-tab,
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
.view-tabs .status-tab:hover,
.status-tab:hover {
  color: var(--primary);
}
.view-tabs .status-tab.is-active,
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
}

/* ---------- 列表 ---------- */
.list-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin: 16px 0;
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

.user-list,
.card-list {
  display: flex;
  flex-direction: column;
}
.user-row,
.card-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.user-row:last-child,
.card-row:last-child {
  border-bottom: none;
}
.user-row.is-selected,
.card-row.is-selected {
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
.pick input:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.user-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--bg-muted);
  color: var(--text-light);
  font-size: 22px;
}
.user-avatar img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.user-avatar.sm {
  width: 36px;
  height: 36px;
  font-size: 20px;
}
.user-avatar.multi {
  color: var(--primary);
}

.user-main {
  flex: 1;
  min-width: 0;
}
.user-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.user-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.id-chip {
  padding: 1px 6px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 3px;
}
.user-meta {
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

.user-integral {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius-sm);
  font-variant-numeric: tabular-nums;
}

.user-actions,
.card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.card-main {
  flex: 1;
  min-width: 0;
}
.card-code-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.card-code {
  padding: 2px 8px;
  font-size: 13px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--text);
  background: var(--bg-muted);
  border-radius: 3px;
  word-break: break-all;
}
.state-chip {
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.state-chip.is-granted {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.12);
}
.state-chip.is-unused {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.state-chip.is-used {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.state-chip.is-expired {
  color: var(--warning);
  background: var(--gold-wash);
}
.value-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 3px;
  font-variant-numeric: tabular-nums;
}
.card-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}

.card-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.card-actions .btn:disabled,
.user-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 规则表 ---------- */
.rule-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.rule-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.rule-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
}
.rule-table tr:last-child td {
  border-bottom: none;
}
.rule-table tbody tr:hover td {
  background: var(--bg-muted);
}
.rule-type {
  padding: 1px 6px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.rule-icon {
  margin-right: 4px;
  color: var(--text-muted);
}
.cell-num {
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.cell-act {
  width: 64px;
  text-align: right;
}

/* ---------- 弹窗 ---------- */
.form-item {
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
/* 设置兑换人：目标用户回显 */
.bind-user {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-soft);
}
.bind-user .bind-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
}
.bind-user .bind-avatar.is-icon {
  font-size: 22px;
  color: var(--text-light);
}
.bind-user-name {
  font-weight: 600;
  color: var(--text);
}
.bind-user-account {
  font-size: 12px;
  color: var(--text-muted);
}
.bind-user-hint {
  font-size: 12px;
  color: var(--text-light);
}
.bind-user-hint.is-error {
  color: var(--danger);
}

.dialog-tip {
  margin: 0;
  padding: 10px 12px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.result-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
}
/* 导入结果里被跳过的卡密明细 */
.result-details {
  margin: 0;
  padding: 10px 12px 10px 28px;
  max-height: 132px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.8;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.result-details li {
  word-break: break-all;
}

.adjust-user {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.quick-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
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
  .user-row,
  .card-row {
    flex-wrap: wrap;
  }
  .user-main,
  .card-main {
    flex: 1 1 60%;
  }
  .user-actions,
  .card-actions {
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
