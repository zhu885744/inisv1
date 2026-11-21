<template>
  <div class="user-list">
    <div class="card card-pad">
      <!-- 头部 -->
      <header class="list-head">
        <div>
          <h2 class="block-title">用户列表</h2>
          <p class="block-desc">
            查看用户资料、分配权限组、冻结与封禁账号
            <template v-if="total > 0"> · 共 {{ total }} 位</template>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" @click="openCreate">
            <i class="bi bi-person-plus" /> 新建用户
          </button>
        </div>
      </header>

      <!-- 筛选栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash && status === tab.key }"
            @click="switchTab(tab.key)"
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
            v-model="searchField"
            :options="searchFields"
            icon="bi bi-funnel"
            placeholder="搜索范围"
            @change="reload" />

          <FormSelect variant="pill"
            v-model="sortKey"
            :options="sortOptions"
            :disabled="trash"
            icon="bi bi-sort-down"
            placeholder="排序方式"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              :placeholder="`搜索${searchFieldLabel}…`"
              :aria-label="`搜索${searchFieldLabel}`"
              @input="doSearch" />
          </div>

          <button
            class="btn btn-ghost btn-sm"
            :disabled="loading"
            title="刷新"
            aria-label="刷新列表"
            @click="load"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </div>
      </div>

      <!-- 批量操作栏（选中后出现） -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 位</span>
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
            <button class="btn btn-sm" :disabled="busy" @click="batchStatus(USER_STATUS_NORMAL)">
              <i class="bi bi-person-check" /> 批量通过/解冻
            </button>
            <button class="btn btn-sm" :disabled="busy" @click="batchStatus(USER_STATUS_FROZEN)">
              <i class="bi bi-snow" /> 批量冻结
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchRemove()">
              <i class="bi bi-trash" /> 删除
            </button>
          </template>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 回收站提示条 -->
      <div v-if="trash" class="trash-bar">
        <span><i class="bi bi-trash3" /> 回收站中的用户无法登录，可恢复或彻底删除</span>
        <button class="btn btn-sm btn-danger" :disabled="busy || !total" @click="askClearRecycle()">
          清空回收站
        </button>
      </div>

      <!-- 列表 -->
      <div v-if="loading" class="is-loading">
        <span class="spinner" /> 加载中...
      </div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="trash ? 'bi bi-trash3' : 'bi bi-people'" :text="emptyText" />
      </div>

      <template v-else>
        <!-- 本页全选 -->
        <div class="list-head-row">
          <label class="pick" title="全选本页（系统管理员与当前账号不可选）">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy || !selectableUsers.length"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 位 · 共 {{ total }} 位</span>
        </div>

        <ul class="user-items">
          <li
            v-for="item in list"
            :key="item.id"
            class="user-row"
            :class="{ 'is-selected': isSelected(item.id) }"
          >
            <label class="pick" :title="isProtected(item) ? protectedText(item) : (isSelected(item.id) ? '取消选择' : '选择')">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy || isProtected(item)"
                :aria-label="`选择用户 ${item.nickname || item.id}`"
                @change="toggleSelect(item.id)" />
            </label>

            <img class="user-avatar" :src="item.avatar || defaultAvatar" :alt="item.nickname || 'avatar'" loading="lazy" />

            <div class="user-main">
              <div class="user-name-row">
                <span class="user-name">{{ item.nickname || '未设置昵称' }}</span>

                <span v-if="Number(item.id) === 1" class="flag is-admin">
                  <i class="bi bi-patch-check-fill" /> 系统管理员
                </span>
                <span v-else-if="isSelf(item)" class="flag is-self">当前账号</span>

                <span v-for="g in groupsOf(item)" :key="g.id" class="flag is-group">
                  <i class="bi bi-shield-check" /> {{ g.name || g.key }}
                </span>

                <span class="status-chip" :class="userStatusClass(item)">
                  <i :class="userStatusIcon(item)" />
                  {{ userStatusLabel(item) }}
                </span>

                <!-- 封禁状态：申诉中 / 申诉驳回也仍然属于封禁中（直到真正解封） -->
                <span v-if="isBanned(item)" class="status-chip" :class="banChipClass(item)">
                  <i :class="banChipIcon(item)" /> {{ banChipText(item) }}
                </span>
              </div>

              <div class="user-contact">
                <span v-if="item.account" class="contact-item"><i class="bi bi-person-badge" /> {{ item.account }}</span>
                <span v-if="item.email" class="contact-item"><i class="bi bi-envelope" /> {{ item.email }}</span>
                <span v-if="item.phone" class="contact-item"><i class="bi bi-telephone" /> {{ item.phone }}</span>
                <!-- 第三方账号绑定（QQ / GitHub / Gitee）：后端只对管理员返回该字段，
                     悬停可看第三方昵称 -->
                <span
                  v-for="oa in item.oauth || []"
                  :key="oa.platform"
                  class="contact-item is-oauth"
                  :title="`${oa.name}${oa.nickname ? '：' + oa.nickname : ''}`"
                >
                  <i :class="platformIcon(oa.platform)" /> {{ oa.name }}
                </span>
                <span
                  v-if="!item.account && !item.email && !item.phone && !(item.oauth || []).length"
                  class="contact-item is-empty"
                >未绑定账号 / 邮箱 / 手机号</span>
              </div>

              <div class="user-meta">
                <span class="meta-text">#{{ item.id }}</span>
                <span v-if="item.title" class="meta-chip"><i class="bi bi-award" /> {{ item.title }}</span>
                <span class="meta-text"><i class="bi bi-lightning-charge" /> {{ item.exp || 0 }} 经验</span>
                <span class="meta-text"><i class="bi bi-coin" /> {{ item.integral || 0 }} 积分</span>
                <span v-if="item.source" class="meta-text"><i class="bi bi-box-arrow-in-right" /> {{ item.source }}</span>
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                <span v-if="banCountOf(item) > 0" class="meta-text is-warn">
                  <i class="bi bi-exclamation-octagon" /> 累计封禁 {{ banCountOf(item) }} 次
                </span>
              </div>

              <p v-if="isBanned(item)" class="ban-line">
                <i class="bi bi-shield-x" /> {{ banLine(item) }}
              </p>
            </div>

            <div class="user-actions">
              <template v-if="trash">
                <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                  <i class="bi bi-arrow-counterclockwise" />
                </button>
                <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                  <i class="bi bi-x-octagon" />
                </button>
              </template>

              <template v-else>
                <!-- 账号状态（0 正常 / 1 冻结）：单选切换，当前状态高亮 -->
                <div class="status-switch" role="radiogroup" aria-label="账号状态">
                  <label
                    v-for="opt in STATUS_OPTIONS"
                    :key="opt.key"
                    class="status-option"
                    :class="[`is-${opt.key}`, { 'is-active': Number(item.status) === opt.value, 'is-disabled': busy || isProtected(item) }]"
                    :title="isProtected(item) ? protectedText(item) : `设为${opt.label}`"
                  >
                    <FormRadio
                      class="status-radio"
                      :name="`user-status-${item.id}`"
                      :value="opt.value"
                      :model-value="Number(item.status) === opt.value"
                      :disabled="busy || isProtected(item)"
                      :aria-label="opt.label"
                      @change="changeStatus(item, opt.value)" />
                    <i :class="opt.icon" aria-hidden="true" />
                  </label>
                </div>

                <router-link
                  :to="`/author/${item.id}`"
                  class="btn btn-ghost btn-sm"
                  title="查看主页"
                  aria-label="查看主页"
                >
                  <i class="bi bi-box-arrow-up-right" />
                </router-link>

                <!-- 查看该用户拥有 / 佩戴中的装扮（只读） -->
                <button class="btn btn-ghost btn-sm" title="查看装扮" aria-label="查看装扮" @click="openDecorations(item)">
                  <i class="bi bi-gem" />
                </button>

                <button class="btn btn-ghost btn-sm" title="编辑资料" aria-label="编辑资料" @click="openEdit(item)">
                  <i class="bi bi-pencil" />
                </button>

                <button class="btn btn-ghost btn-sm" title="分配权限组" aria-label="分配权限组" @click="openGroups(item)">
                  <i class="bi bi-shield-check" />
                </button>

                <button
                  v-if="isBanned(item)"
                  class="btn btn-ghost btn-sm ok"
                  title="解除封禁"
                  aria-label="解除封禁"
                  :disabled="busy"
                  @click="askUnban(item)"
                >
                  <i class="bi bi-shield-check" />
                </button>
                <button
                  v-else
                  class="btn btn-ghost btn-sm danger"
                  :title="isProtected(item) ? protectedText(item) : '封禁账号'"
                  aria-label="封禁账号"
                  :disabled="busy || isProtected(item)"
                  @click="openBan(item)"
                >
                  <i class="bi bi-shield-x" />
                </button>

                <button
                  v-if="hasBanTrace(item)"
                  class="btn btn-ghost btn-sm"
                  :title="isProtected(item) ? protectedText(item) : '清空封禁信息'"
                  aria-label="清空封禁信息"
                  :disabled="busy || isProtected(item)"
                  @click="askClearBan(item)"
                >
                  <i class="bi bi-eraser" />
                </button>

                <!-- 处理申诉：仅在申诉审核中（status=3）出现 -->
                <button
                  v-if="isAppealing(item)"
                  class="btn btn-ghost btn-sm is-appeal"
                  title="处理申诉"
                  aria-label="处理申诉"
                  :disabled="busy"
                  @click="openAppeal(item)"
                >
                  <i class="bi bi-chat-left-text" />
                </button>

                <button
                  class="btn btn-ghost btn-sm danger"
                  :title="isProtected(item) ? protectedText(item) : '删除'"
                  aria-label="删除"
                  :disabled="busy || isProtected(item)"
                  @click="askRemove(item)"
                >
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
        @update:page-size="(size) => { pageSize = size; page = 1; load() }"
        @update:current="changePage"
      />
    </div>

    <!-- 新建 / 编辑用户 -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="edit.dialogTitle"
      :icon="edit.mode === 'create' ? 'bi bi-person-plus' : 'bi bi-pencil-square'"
      :loading="edit.loading"
      confirm-text="保存"
      width="600px"
      @confirm="saveUser"
    >
      <!-- 基本资料 -->
      <section class="form-section">
        <h4 class="section-title">基本资料</h4>

        <div class="form-item">
          <label class="form-label">头像</label>
          <div class="avatar-row">
            <div class="avatar-box">
              <img v-if="edit.avatar" :src="edit.avatar" alt="头像预览" loading="lazy" />
              <div v-else class="avatar-empty" @click="pickAvatar">
                <i class="bi bi-image" />
                <span>上传</span>
              </div>
              <button v-if="edit.avatar" type="button" class="avatar-del" title="移除头像" @click="edit.avatar = ''">
                <i class="bi bi-x" />
              </button>
            </div>
            <div class="avatar-fields">
              <FormInput v-model="edit.avatar" type="text" placeholder="粘贴图片链接，或点击下方按钮上传" />
              <div class="avatar-actions">
                <button class="btn btn-sm" type="button" :disabled="uploading" @click="pickAvatar">
                  <i class="bi bi-upload" /> {{ uploading ? '上传中...' : '上传图片' }}
                </button>
                <button class="btn btn-sm" type="button" title="从附件库选择" @click="showAvatarLibrary = true">
                  <i class="bi bi-folder2-open" /> 附件库
                </button>
                <button v-if="edit.avatar" class="btn btn-sm btn-ghost" type="button" :disabled="uploading" @click="edit.avatar = ''">
                  移除
                </button>
              </div>
            </div>
            <input ref="avatarFileRef" type="file" accept="image/*" hidden @change="onAvatarChange" />

            <!-- 附件库：选一张已上传的图片作为用户头像 -->
            <AttachmentLibrary
              v-model:visible="showAvatarLibrary"
              title="选择用户头像"
              accept="image"
              @select="onAvatarLibrarySelect"
            />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">昵称</label>
            <FormInput v-model="edit.nickname" type="text" maxlength="32" placeholder="汉字、字母、数字、下划线或中划线" />
          </div>
          <div class="form-item">
            <label class="form-label">性别</label>
            <FormSelect v-model="edit.gender" :options="GENDER_OPTIONS" />
          </div>
        </div>

        <div class="form-item">
          <label class="form-label">个人简介</label>
          <FormTextarea v-model="edit.description" rows="2" maxlength="256" placeholder="可选，最多 256 字" />
        </div>
      </section>

      <!-- 账号与安全 -->
      <section class="form-section">
        <h4 class="section-title">账号与安全</h4>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">账号</label>
            <FormInput v-model="edit.account" type="text" maxlength="32" placeholder="字母或数字，4-32 位" />
          </div>
          <div class="form-item">
            <label class="form-label">{{ edit.mode === 'create' ? '登录密码' : '重置密码' }}</label>
            <FormInput v-model="edit.password" type="text" maxlength="32" placeholder="6-32 位" />
            <p class="form-hint">
              {{ edit.mode === 'create' ? '留空则该账号只能通过邮箱验证码登录' : '留空表示不修改当前密码' }}
            </p>
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">邮箱{{ edit.mode === 'create' ? '（必填）' : '' }}</label>
            <FormInput v-model="edit.email" type="email" placeholder="user@example.com" />
          </div>
          <div class="form-item">
            <label class="form-label">手机号</label>
            <FormInput v-model="edit.phone" type="text" maxlength="32" placeholder="可选" />
          </div>
        </div>

        <div class="form-grid">
          <div class="form-item">
            <label class="form-label">注册来源</label>
            <FormInput v-model="edit.source" type="text" maxlength="32" placeholder="如 mellow / default" />
            <p class="form-hint">用于区分注册渠道，留空则不修改</p>
          </div>
          <div class="form-item">
            <label class="form-label">账号状态</label>
            <FormSelect
              v-model="edit.status"
              :options="STATUS_OPTIONS"
              :disabled="Number(edit.id) === 1" />
            <p v-if="Number(edit.id) === 1" class="form-hint">系统管理员账号状态不可修改</p>
          </div>
        </div>

        <div class="form-item">
          <label class="form-label">管理员备注</label>
          <FormInput v-model="edit.remark" type="text" placeholder="仅后台可见" />
        </div>
      </section>

      <!-- 第三方账号（只读）：该用户绑定的 QQ / GitHub / Gitee -->
      <section v-if="edit.mode === 'edit'" class="form-section">
        <h4 class="section-title">第三方账号</h4>
        <p v-if="edit.oauthLoading" class="form-hint">加载中…</p>
        <p v-else-if="!edit.oauth.length" class="form-hint">该用户未绑定第三方账号</p>
        <ul v-else class="user-overview-list">
          <li v-for="oa in edit.oauth" :key="oa.platform" class="user-overview-item">
            <i class="user-overview-icon" :class="platformIcon(oa.platform)" />
            <span class="user-overview-name">{{ oa.name }}</span>
            <span class="user-overview-nick" :title="oa.nickname">{{ oa.nickname || '已绑定' }}</span>
            <span v-if="oa.create_time" class="user-overview-time">{{ fromNow(oa.create_time) }}绑定</span>
          </li>
        </ul>
        <p class="form-hint">绑定 / 解绑由用户在「账号安全」里自行操作，后台仅查看</p>
      </section>
    </AdminFormDialog>

    <!-- 封禁用户 -->
    <AdminFormDialog
      v-model:visible="ban.visible"
      title="封禁用户"
      icon="bi bi-shield-x"
      :loading="ban.loading"
      confirm-text="确认封禁"
      width="560px"
      @confirm="submitBan"
    >
      <p class="dialog-tip">
        将对 <strong>{{ ban.nickname || `用户 ${ban.uid}` }}</strong> 执行封禁；封禁会强制其下线，可随时解封。
      </p>

      <div class="form-item">
        <label class="form-label">封禁范围</label>
        <label class="check-line">
          <FormCheckbox v-model="ban.all" />
          <span>全面封禁（限制全部操作）</span>
        </label>
        <div v-if="!ban.all" class="check-grid">
          <label v-for="t in BAN_TYPES" :key="t.bit" class="check-line">
            <FormCheckbox v-model="ban.bits" :value="t.bit" />
            <span>{{ t.label }}</span>
          </label>
        </div>
        <p class="form-hint">当前封禁内容：{{ banTypeText(banTypeMask) }}</p>
      </div>

      <div class="form-item">
        <label class="form-label">封禁时长</label>
        <FormSelect v-model="ban.durationMode" :options="BAN_DURATION_OPTIONS" />
        <p class="form-hint">
          自动梯度按该用户累计封禁次数计算：首次 1 天、二次 7 天、三次 15 天、四次 30 天、五次及以上永久。
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">封禁原因</label>
        <FormInput v-model="ban.reason" type="text" maxlength="512" placeholder="默认为「违反社区规定」" />
      </div>

      <div class="form-item">
        <label class="form-label">违规证据</label>
        <FormTextarea v-model="ban.evidence" rows="3" maxlength="1024" placeholder="可选，如违规内容链接、截图地址等" />
      </div>

      <div class="form-item">
        <label class="form-label">附加处理</label>
        <label class="check-line">
          <FormCheckbox v-model="ban.freezeUser" />
          <span>同时冻结账号（无法登录，解封时一并恢复）</span>
        </label>
        <label class="check-line">
          <FormCheckbox v-model="ban.banAppeal" />
          <span>禁止申诉</span>
        </label>
        <label class="check-line is-danger">
          <FormCheckbox v-model="ban.deleteContent" />
          <span>同时删除该用户的全部内容（文章 / 动态 / 评论，移入回收站）</span>
        </label>
      </div>
    </AdminFormDialog>

    <!-- 处理申诉（管理员）：通过=立即解封 / 驳回=封禁继续生效 -->
    <AdminFormDialog
      v-model:visible="appeal.visible"
      title="处理封禁申诉"
      icon="bi bi-chat-left-text"
      :loading="appeal.loading"
      confirm-text="提交处理"
      width="560px"
      @confirm="submitAppeal"
    >
      <p class="dialog-tip">
        用户 <strong>{{ appeal.nickname }}</strong> 对封禁记录 #{{ appeal.recordId }} 提交了申诉。
      </p>

      <dl class="appeal-info">
        <div>
          <dt>申诉时间</dt>
          <dd>{{ appeal.time ? formatTime(appeal.time) : '—' }}</dd>
        </div>
        <div>
          <dt>封禁原因</dt>
          <dd>{{ appeal.reason || '未说明' }}</dd>
        </div>
        <div>
          <dt>封禁范围</dt>
          <dd>{{ appeal.banType || '全面封禁' }}</dd>
        </div>
      </dl>

      <div class="form-item">
        <label class="form-label">申诉内容</label>
        <p class="appeal-content">{{ appeal.content || '（用户未填写申诉说明）' }}</p>
      </div>

      <div class="form-item">
        <label class="form-label">处理结果</label>
        <FormSelect v-model="appeal.action" :options="APPEAL_ACTIONS" />
        <p class="form-hint">
          通过：立即解封并恢复账号；驳回：封禁继续生效（用户均可收到站内消息）。
        </p>
      </div>

      <div class="form-item">
        <label class="form-label">
          回复内容{{ appeal.action === 'reject' ? '（驳回必填）' : '（可选）' }}
        </label>
        <FormTextarea
          v-model="appeal.reply"
          rows="3"
          maxlength="1024"
          placeholder="驳回时请说明理由，用户会在站内消息里看到" />
      </div>
    </AdminFormDialog>

    <!-- 分配权限组 -->
    <AdminFormDialog
      v-model:visible="groups.visible"
      title="分配权限组"
      icon="bi bi-shield-check"
      :loading="groups.loading"
      confirm-text="保存"
      @confirm="saveGroups"
    >
      <p class="dialog-tip">
        为 <strong>{{ groups.nickname || `用户 ${groups.uid}` }}</strong> 指定所属权限组，保存后立即生效（该用户需重新登录刷新权限）。
      </p>

      <div v-if="groups.fetching" class="is-loading"><span class="spinner" /> 加载权限组...</div>

      <div v-else-if="!groups.all.length" class="dialog-empty">
        暂无可用权限组：请先到「权限分组」页面创建，或确认当前账号有权访问权限组列表。
      </div>

      <div v-else class="check-grid is-block">
        <label v-for="g in groups.all" :key="g.id" class="check-line">
          <FormCheckbox v-model="groups.selected" :value="Number(g.id)" />
          <span>
            {{ g.name || g.key }}
            <code v-if="g.key" class="group-key">{{ g.key }}</code>
            <span v-if="Number(g.root) === 1" class="root-flag">超管</span>
          </span>
        </label>
      </div>
    </AdminFormDialog>

    <!-- 查看用户装扮（只读）：佩戴中的用标签标出，含有效期与获得来源 -->
    <AdminFormDialog
      v-model:visible="decos.visible"
      title="用户装扮"
      icon="bi bi-gem"
      width="620px"
      :loading="decos.loading"
    >
      <p class="dialog-tip">
        <strong>{{ decos.nickname || `用户 ${decos.uid}` }}</strong> 的装扮清单（含系统默认装扮，标注「默认」）。
        发放 / 回收请到「装扮管理」操作。
      </p>

      <div v-if="decos.loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <template v-else>
        <div v-if="!decos.groups.length" class="dialog-empty">该用户暂无任何装扮</div>

        <div v-for="group in decos.groups" :key="group.type" class="decoration-group">
          <div class="decoration-group-head">
            <span><i :class="group.icon" /> {{ group.name }}</span>
            <span class="decoration-group-count">{{ group.items.length }} 件</span>
          </div>
          <ul class="decoration-item-list">
            <li v-for="item in group.items" :key="item.id" class="decoration-item">
              <img
                v-if="item.type === 'avatar_frame'"
                :src="item.preview || item.payload?.image"
                class="decoration-thumb"
                :alt="item.name"
                loading="lazy"
              />
              <i v-else class="bi bi-award decoration-thumb-icon" />

              <span class="decoration-item-name">{{ item.name }}</span>
              <span v-if="item.wearing" class="decoration-tag is-wearing">佩戴中</span>
              <span v-else-if="item.expired" class="decoration-tag is-expired">已过期</span>
              <span v-else-if="item.off_shelf" class="decoration-tag is-expired">已下架</span>
              <span v-else-if="item.is_default" class="decoration-tag">默认</span>

              <span class="decoration-item-meta">
                {{ decoDuration(item) }}<template v-if="item.source"> · {{ decoSource(item) }}</template>
              </span>
            </li>
          </ul>
        </div>
      </template>

      <template #footer>
        <button class="btn" type="button" @click="decos.visible = false">关闭</button>
      </template>
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

    <!-- 清空封禁信息（危险操作：物理删除封禁记录 + 归零用户封禁字段） -->
    <ConfirmDialog
      v-model:visible="clearBan.visible"
      title="清空封禁信息"
      :message="clearBan.message"
      confirm-text="清空"
      danger
      :loading="clearBan.loading"
      @confirm="runClearBan"
    >
      <label v-if="clearBan.frozen" class="ban-clear-option">
        <FormCheckbox v-model="clearBan.unfreeze" />
        <span>同时解除「冻结」状态（该账号当前处于冻结中，只清记录仍无法登录）</span>
      </label>
    </ConfirmDialog>
  </div>
</template>

<script setup>
import FormRadio from '@/components/form/FormRadio.vue'
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 用户列表（/admin/users 子路由）
 *
 * 与父级 Users.vue 通过 inject('usersAdmin') 联动：统计卡片点击 → 列表筛选，
 * 列表增删改 → 刷新统计卡片。列表状态（标签 / 回收站 / 排序 / 搜索 / 页码）都在这里维护。
 *
 * 关键约束（后端 users 控制器）：
 * - users/update 的允许字段随权限变化，管理员（root）才能改 email/phone/source/remark；
 * - users/status 只允许 0/1，且禁止修改系统管理员；
 * - users/ban 的 duration 传字符串 '0' 表示永久，不传并置 auto_gradient=true 走自动梯度；
 * - users/remove|delete 禁止包含 id=1 与当前登录账号，故前端直接禁用这些行的选择与操作。
 */
import { ref, reactive, computed, inject, watchEffect, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listUsers,
  createUser,
  updateUser,
  setUserStatus,
  banUser,
  unbanUser,
  clearUserBan,
  handleUserAppeal,
  removeUsers,
  forceDeleteUsers,
  restoreUsers,
  clearUserRecycle
} from '@/api/users'
import { listAuthGroups, setUserAuthGroups } from '@/api/auth-group'
import { getUserDecorations } from '@/api/decoration'
// 第三方登录平台图标（bi-tencent-qq / bi-github / bi-git）
import { platformIcon } from '@/utils/oauth'
import { getMyOauth } from '@/api/oauth'
import { uploadAttachments } from '@/api/attachment'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { fromNow, formatTime } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'
import {
  USER_TABS,
  USER_SORT_OPTIONS,
  USER_SEARCH_FIELDS,
  USER_TRASH_KEY,
  USER_TRASH_ORDER,
  USER_DEFAULT_ORDER,
  USER_STATUS_NORMAL,
  USER_STATUS_AUDIT,
  USER_STATUS_FROZEN,
  BAN_TYPES,
  BAN_TYPE_ALL,
  BAN_DURATION_OPTIONS,
  userListParams,
  userStatusLabel,
  banTypeText,
  banDurationText,
  banRecordOf,
  isUserBanned,
  isUserAppealing,
  userBanStatus,
  userBanStatusLabel,
  hasUserBanInfo,
  userGroupsOf,
  isProtectedUser,
  protectedReason
} from '@/utils/user'

const store = useUserStore()
const { user: currentUser } = storeToRefs(store)
const currentUid = computed(() => Number(currentUser.value?.id || 0))

// 父级 Users.vue 注入的联动能力（统计卡片 ←→ 列表筛选）
const admin = inject('usersAdmin', null)

const pageSize = ref(PAGE_SIZE_DEFAULT)
const tabs = USER_TABS
const sortOptions = USER_SORT_OPTIONS
const searchFields = USER_SEARCH_FIELDS

// 账号状态切换项（对应后端 users/status 的 0/1/2）
// 同时用作行内状态单选与编辑弹窗的 SelectMenu 选项（后者只取 value / label）
// 待审核（2）由「注册验证方式=人工审核」写入，管理员点「正常」即视为审核通过
const STATUS_OPTIONS = [
  { value: USER_STATUS_NORMAL, key: 'normal', label: '正常', icon: 'bi bi-person-check' },
  { value: USER_STATUS_AUDIT, key: 'audit', label: '待审核', icon: 'bi bi-hourglass-split' },
  { value: USER_STATUS_FROZEN, key: 'frozen', label: '冻结', icon: 'bi bi-snow' }
]

// 性别下拉选项（后端存 boy / girl / 空）
const GENDER_OPTIONS = [
  { value: '', label: '保密' },
  { value: 'boy', label: '男' },
  { value: 'girl', label: '女' }
]

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="34" fill="%238a8a82" font-family="serif">用</text></svg>'

const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
// 回收站模式
const trash = ref(false)
const status = ref('all')
const sortKey = ref(USER_DEFAULT_ORDER)
const searchField = ref('nickname')
const keyword = ref('')
const searchKey = ref('')
const selectedIds = ref([])

const currentTab = computed(() => tabs.find((tab) => tab.key === status.value) || tabs[0])

const searchFieldLabel = computed(
  () => searchFields.find((f) => f.value === searchField.value)?.label || '昵称'
)

const emptyText = computed(() => {
  if (trash.value) return '回收站是空的'
  if (searchKey.value) return `没有匹配的用户`
  if (status.value !== 'all') return `暂无「${currentTab.value.label}」的用户`
  return '暂无用户'
})

// ---------- 展示辅助 ----------
function isSelf(item) {
  return currentUid.value > 0 && Number(item?.id) === currentUid.value
}

function isProtected(item) {
  return isProtectedUser(item, currentUid.value)
}

function protectedText(item) {
  return protectedReason(item, currentUid.value)
}

const groupsOf = userGroupsOf

function isBanned(item) {
  return isUserBanned(item)
}

// 是否需要展示「清空封禁信息」入口：封禁中或有历史封禁痕迹（含累计次数 / 最后封禁时间）
function hasBanTrace(item) {
  return hasUserBanInfo(item)
}

// 申诉审核中（status=3）：此时才出现「处理申诉」按钮
function isAppealing(item) {
  return isUserAppealing(item)
}

// 封禁状态标签：封禁中 / 申诉中 / 申诉驳回（后端 is_banned 在申诉期间仍为 true）
function banChipText(item) {
  return userBanStatusLabel(item)
}

function banChipClass(item) {
  const status = userBanStatus(item)
  if (status === 3) return 'is-appeal'
  if (status === 5) return 'is-appeal-rejected'
  return 'is-ban'
}

function banChipIcon(item) {
  const status = userBanStatus(item)
  if (status === 3) return 'bi bi-hourglass-split'
  if (status === 5) return 'bi bi-x-circle'
  return 'bi bi-shield-x'
}

function banCountOf(item) {
  return Number(item?.result?.ban?.ban_count ?? item?.ban_count ?? 0) || 0
}

function userStatusClass(item) {
  const status = Number(item?.status)
  if (status === USER_STATUS_FROZEN) return 'is-frozen'
  if (status === USER_STATUS_AUDIT) return 'is-audit'
  return 'is-normal'
}

function userStatusIcon(item) {
  const status = Number(item?.status)
  if (status === USER_STATUS_FROZEN) return 'bi bi-snow'
  if (status === USER_STATUS_AUDIT) return 'bi bi-hourglass-split'
  return 'bi bi-person-check'
}

function banLine(item) {
  const record = banRecordOf(item)
  if (!record) return '该账号处于封禁状态'
  const parts = [banTypeText(record.ban_type), banDurationText(record.duration)]
  if (record.reason) parts.push(`原因：${record.reason}`)
  if (Number(record.expires_at) > 0) parts.push(`到期：${formatTime(record.expires_at)}`)
  // 申诉中 / 申诉驳回也要标出来：管理员据此决定是否需要处理申诉
  const status = userBanStatus(item)
  if (status === 3) {
    parts.push(record.appeal_time ? `申诉中（${formatTime(record.appeal_time)}）` : '申诉中')
  }
  if (status === 5) parts.push('申诉已驳回')
  return parts.join(' · ')
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return `注册于 ${fromNow(item.create_time)}`
}

// ---------- 数据加载 ----------
// 请求序号：快速切换筛选 / 搜索时丢弃过期响应，避免旧数据覆盖新结果
let reqSeq = 0

async function load() {
  const run = ++reqSeq
  loading.value = true
  try {
    const res = await listUsers(
      userListParams({
        key: trash.value ? USER_TRASH_KEY : status.value,
        page: page.value,
        limit: pageSize.value,
        keyword: searchKey.value,
        searchField: searchField.value,
        order: trash.value ? USER_TRASH_ORDER : sortKey.value
      })
    )
    if (run !== reqSeq) return
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
  } catch {
    if (run !== reqSeq) return
    list.value = []
    total.value = 0
  } finally {
    if (run === reqSeq) loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

// ---------- 筛选联动 ----------
function switchTab(key) {
  if (!trash.value && status.value === key) return
  trash.value = false
  status.value = key
  admin?.setActive(key)
  reload()
}

function toggleTrash() {
  trash.value = !trash.value
  admin?.setActive(trash.value ? USER_TRASH_KEY : status.value)
  reload()
}

// 父级统计卡片点击后的回调（key 与 tabs / trash 对应）
function applyFilter(key) {
  if (key === USER_TRASH_KEY) {
    if (trash.value && page.value === 1) return
    trash.value = true
    reload()
    return
  }
  const next = tabs.some((t) => t.key === key) ? key : 'all'
  if (!trash.value && status.value === next) return
  trash.value = false
  status.value = next
  reload()
}

admin?.registerFilter(applyFilter)
onUnmounted(() => admin?.unregisterFilter?.(applyFilter))

// ---------- 选中 ----------
// 系统管理员与当前登录账号不可批量操作（后端会拒绝），直接排除在可选范围外
const selectableUsers = computed(() => list.value.filter((u) => !isProtected(u)))
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => selectableUsers.value.length > 0 && selectableUsers.value.every((u) => selectedSet.value.has(u.id))
)
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)

const selectAllRef = ref(null)

// 部分选中时展示半选态（indeterminate 只能通过 DOM 属性设置）
watchEffect(() => {
  if (selectAllRef.value) selectAllRef.value.setIndeterminate(someSelected.value)
})

function isSelected(id) {
  return selectedSet.value.has(id)
}

function toggleSelect(id) {
  if (isProtected({ id })) return
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleSelectAll() {
  selectedIds.value = pageAllSelected.value ? [] : selectableUsers.value.map((u) => u.id)
}

function clearSelection() {
  selectedIds.value = []
}

// ---------- 确认弹窗 ----------
const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

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

// ---------- 清空封禁信息（危险操作） ----------
// 与「解封」不同：会物理删除该用户的全部封禁记录，并把累计封禁次数 / 当前封禁记录 /
// 封禁限制 / 最后封禁时间全部归零（后端 users/clear-ban），不可恢复。
// 账号处于「冻结」中时额外给一个「同时解除冻结」的选项：只清记录不解冻会变成
// 「没有封禁记录却登录不了」的状态。
const clearBan = reactive({
  visible: false,
  loading: false,
  message: '',
  uid: 0,
  frozen: false,
  unfreeze: true
})

function askClearBan(item) {
  clearBan.uid = Number(item.id)
  clearBan.frozen = Number(item.status) === USER_STATUS_FROZEN
  // 冻结中的账号默认勾选，避免清完记录后账号仍然登录不了
  clearBan.unfreeze = clearBan.frozen
  clearBan.message =
    `确定清空用户「${item.nickname || item.id}」的全部封禁信息吗？` +
    '该用户的所有封禁记录会被永久删除，累计封禁次数、当前封禁记录、封禁限制与最后封禁时间一并归零，此操作不可恢复！'
  clearBan.visible = true
}

async function runClearBan() {
  clearBan.loading = true
  try {
    const res = await clearUserBan({
      uid: clearBan.uid,
      unfreeze: clearBan.unfreeze ? 1 : 0
    })
    const records = Number(res?.data?.records || 0)
    toast.success(records > 0 ? `已清空 ${records} 条封禁记录` : '已清空该用户的封禁信息')
    clearBan.visible = false
    await afterMutation(0)
  } catch {
    // 请求失败时保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    clearBan.loading = false
  }
}

// ---------- 处理申诉（管理员） ----------
// 通过：立即解封（后端清空 restrictions / current_ban_id，记录状态=4）
// 驳回：封禁继续生效（记录状态=5），回复内容必填并会写进用户的站内消息
const APPEAL_ACTIONS = [
  { value: 'approve', label: '通过（立即解封）' },
  { value: 'reject', label: '驳回（封禁继续生效）' }
]

const appeal = reactive({
  visible: false,
  loading: false,
  uid: 0,
  nickname: '',
  recordId: 0,
  content: '',
  time: 0,
  reason: '',
  banType: '',
  action: 'approve',
  reply: ''
})

function openAppeal(item) {
  const record = banRecordOf(item) || {}
  appeal.uid = Number(item.id)
  appeal.nickname = item.nickname || `用户 ${item.id}`
  appeal.recordId = Number(record.id) || 0
  appeal.content = record.appeal_content || ''
  appeal.time = Number(record.appeal_time) || 0
  appeal.reason = record.reason || ''
  appeal.banType = banTypeText(record.ban_type)
  appeal.action = 'approve'
  appeal.reply = ''
  appeal.visible = true
}

async function submitAppeal() {
  if (!appeal.recordId) {
    toast.warning('未找到对应的封禁记录')
    return
  }
  if (appeal.action === 'reject' && !appeal.reply.trim()) {
    toast.warning('驳回时请填写回复内容')
    return
  }

  appeal.loading = true
  try {
    await handleUserAppeal({
      record_id: appeal.recordId,
      action: appeal.action,
      reply: appeal.reply.trim()
    })
    toast.success(appeal.action === 'approve' ? '申诉已通过，账号已解封' : '申诉已驳回，封禁继续生效')
    appeal.visible = false
    await afterMutation(0)
  } catch {
    // 请求失败时保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    appeal.loading = false
  }
}

// ---------- 增删改统一收尾 ----------
// removedCount：本次从当前页移除的条数；当前页被清空时自动回退一页
async function afterMutation(removedCount = 0) {
  if (removedCount > 0 && list.value.length <= removedCount && page.value > 1) {
    page.value -= 1
  }
  await load()
  refreshStats()
}

function refreshStats() {
  admin?.refreshStats?.()
}

// ---------- 单条操作 ----------
async function changeStatus(item, value) {
  if (Number(item.status) === Number(value)) return
  busy.value = true
  try {
    const wasAudit = Number(item.status) === USER_STATUS_AUDIT
    await setUserStatus(item.id, value)
    item.status = value
    if (value === USER_STATUS_FROZEN) toast.success('账号已冻结')
    else if (value === USER_STATUS_AUDIT) toast.success('已标记为待审核')
    else toast.success(wasAudit ? '审核通过，账号已可登录' : '账号已解冻')
    // 状态筛选下该条目可能已不属于当前标签
    if (status.value !== 'all') await afterMutation(0)
    else refreshStats()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askUnban(item) {
  openConfirm({
    title: '解除封禁',
    message: `确定解除用户「${item.nickname || item.id}」的封禁吗？解封后其账号恢复正常使用。`,
    confirmText: '解除封禁',
    action: async () => {
      await unbanUser({ uid: Number(item.id) })
      toast.success('已解除封禁')
      await afterMutation(0)
    }
  })
}

function askRemove(item) {
  openConfirm({
    title: '删除用户',
    message: `确定删除用户「${item.nickname || item.id}」吗？删除后该账号无法登录，可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeUsers([item.id])
      toast.success('已移入回收站')
      await afterMutation(1)
    }
  })
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除用户「${item.nickname || item.id}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteUsers([item.id])
      toast.success('已彻底删除')
      await afterMutation(1)
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreUsers([item.id])
    toast.success('已恢复')
    await afterMutation(1)
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

// ---------- 批量操作 ----------
// 逐条调用（后端 status 仅支持单条），统计成功/失败
async function batchStatus(value) {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  let ok = 0
  try {
    for (const id of ids) {
      try {
        await setUserStatus(id, value)
        const target = list.value.find((u) => Number(u.id) === Number(id))
        if (target) target.status = value
        ok += 1
      } catch {
        /* 单条失败不影响其它 */
      }
    }
  } finally {
    busy.value = false
  }
  clearSelection()
  toast.success(`操作完成：成功 ${ok} 位，失败 ${ids.length - ok} 位`)
  // 有失败条目或处于状态筛选下，重新拉取保证列表与统计准确
  if (ok !== ids.length || status.value !== 'all') await afterMutation(0)
  else refreshStats()
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量删除',
    message: `确定删除选中的 ${ids.length} 位用户吗？删除后可在回收站找回。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      await removeUsers(ids)
      toast.success(`已删除 ${ids.length} 位`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  if (!ids.length) return
  busy.value = true
  try {
    await restoreUsers(ids)
    toast.success(`已恢复 ${ids.length} 位`)
    clearSelection()
    await afterMutation(ids.length)
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
    message: `确定彻底删除选中的 ${ids.length} 位用户吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteUsers(ids)
      toast.success(`已彻底删除 ${ids.length} 位`)
      clearSelection()
      await afterMutation(ids.length)
    }
  })
}

function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？回收站内所有用户将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearUserRecycle()
      toast.success('回收站已清空')
      page.value = 1
      await afterMutation(0)
    }
  })
}

// ---------- 新建 / 编辑用户 ----------
const avatarFileRef = ref(null)
const uploading = ref(false)
const showAvatarLibrary = ref(false)

// 从附件库选头像：直接复用已上传图片的地址
function onAvatarLibrarySelect(urls = []) {
  const url = urls[0]
  showAvatarLibrary.value = false
  if (url) edit.avatar = url
}

const edit = reactive({
  visible: false,
  loading: false,
  mode: 'create',
  // 弹窗标题（与用户头衔无关，命名上避免和 title 字段混淆）
  dialogTitle: '',
  id: 0,
  account: '',
  nickname: '',
  email: '',
  phone: '',
  avatar: '',
  // 注：不再提供 title（头衔）—— 头衔由「我的装扮」统一管理：
  // 后台在「装扮管理」发放头衔，用户在自己的「我的装扮」里佩戴 / 更换。
  gender: '',
  description: '',
  remark: '',
  source: 'default',
  password: '',
  status: USER_STATUS_NORMAL,
  // 第三方账号绑定（只读展示，见「第三方账号」分组）
  oauth: [],
  oauthLoading: false
})

function resetEdit() {
  edit.id = 0
  edit.dialogTitle = ''
  edit.account = ''
  edit.nickname = ''
  edit.email = ''
  edit.phone = ''
  edit.avatar = ''
  edit.gender = ''
  edit.description = ''
  edit.remark = ''
  edit.source = 'default'
  edit.password = ''
  edit.status = USER_STATUS_NORMAL
  edit.oauth = []
  edit.oauthLoading = false
}

function openCreate() {
  resetEdit()
  edit.mode = 'create'
  edit.dialogTitle = '新建用户'
  edit.visible = true
}

function openEdit(item) {
  resetEdit()
  edit.mode = 'edit'
  edit.id = Number(item.id)
  edit.account = item.account || ''
  edit.nickname = item.nickname || ''
  edit.email = item.email || ''
  edit.phone = item.phone || ''
  edit.avatar = item.avatar || ''
  edit.gender = item.gender === 'boy' || item.gender === 'girl' ? item.gender : ''
  edit.description = item.description || ''
  edit.remark = item.remark || ''
  edit.source = item.source || 'default'
  edit.status = [USER_STATUS_NORMAL, USER_STATUS_FROZEN, USER_STATUS_AUDIT].includes(Number(item.status))
    ? Number(item.status)
    : USER_STATUS_NORMAL
  edit.dialogTitle = `编辑用户 #${item.id}`
  // 列表行里已经带回了绑定信息，先占位显示，再拉一次最新的（管理员可查任意用户）
  edit.oauth = Array.isArray(item.oauth) ? item.oauth : []
  edit.visible = true
  loadEditOauth(edit.id)
}

// loadEditOauth - 拉取指定用户的第三方绑定（编辑弹窗里的「第三方账号」分组）
async function loadEditOauth(uid) {
  const target = Number(uid)
  if (!target) return

  edit.oauthLoading = true
  try {
    const res = await getMyOauth({ uid: target })
    // 期间可能已经切到别的用户，避免串数据
    if (Number(edit.id) !== target) return
    if (Array.isArray(res?.data?.list)) edit.oauth = res.data.list
  } catch {
    // 失败就保留列表带过来的数据，不打断编辑流程
  } finally {
    if (Number(edit.id) === target) edit.oauthLoading = false
  }
}

function pickAvatar() {
  avatarFileRef.value?.click()
}

async function onAvatarChange(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
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
    edit.avatar = url
    toast.success('头像已上传')
  } catch {
    toast.error('头像上传失败')
  } finally {
    uploading.value = false
  }
}

function validateEdit() {
  const nickname = edit.nickname.trim()
  if (!nickname) {
    toast.warning('请输入昵称')
    return false
  }
  if (!/^[\u4e00-\u9fa5A-Za-z0-9_-]{1,32}$/.test(nickname)) {
    toast.warning('昵称只能是汉字、字母、数字、下划线或中划线，且不超过 32 位')
    return false
  }
  const account = edit.account.trim()
  if (account && !/^[A-Za-z0-9]{4,32}$/.test(account)) {
    toast.warning('账号只能由字母和数字组成，长度 4-32 位')
    return false
  }
  const email = edit.email.trim()
  if (edit.mode === 'create' && !email) {
    toast.warning('请输入邮箱')
    return false
  }
  if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    toast.warning('邮箱格式不正确')
    return false
  }
  const password = edit.password.trim()
  if (password && (password.length < 6 || password.length > 32)) {
    toast.warning('密码长度需为 6-32 位')
    return false
  }
  if (edit.description.length > 256) {
    toast.warning('个人简介最多 256 字')
    return false
  }
  return true
}

async function saveUser() {
  if (!validateEdit()) return

  // 注：不再提交 title —— 头衔由「我的装扮」统一管理（后台在「装扮管理」发放，
  // 用户在「我的装扮」佩戴 / 更换）；后端 users/update、users/save 也已把 title 收紧为管理员专用字段
  const payload = {
    nickname: edit.nickname.trim(),
    avatar: edit.avatar.trim(),
    gender: edit.gender,
    description: edit.description.trim()
  }
  // 管理员专属字段（后端仅 root 允许更新，非管理员会被静默忽略）
  // 留空则不提交，避免把「未填写」当成「要清空」触发后端的唯一性/格式校验
  const account = edit.account.trim()
  if (account) payload.account = account
  const email = edit.email.trim()
  if (email) payload.email = email
  const phone = edit.phone.trim()
  if (phone) payload.phone = phone
  const remark = edit.remark.trim()
  if (remark) payload.remark = remark
  const source = edit.source.trim()
  if (source) payload.source = source

  const password = edit.password.trim()
  if (password) payload.password = password

  edit.loading = true
  try {
    if (edit.mode === 'create') {
      payload.status = Number(edit.status)
      await createUser(payload)
      toast.success('用户已创建')
    } else {
      payload.id = edit.id
      // 系统管理员状态不允许修改，避免连带触发后端保护逻辑
      if (Number(edit.id) !== 1) payload.status = Number(edit.status)
      await updateUser(payload)
      toast.success('修改已保存')
    }
    edit.visible = false
    await afterMutation(0)
  } catch {
    // 失败提示由请求拦截器统一给出（如账号/邮箱已存在）
  } finally {
    edit.loading = false
  }
}

// ---------- 封禁 ----------
const ban = reactive({
  visible: false,
  loading: false,
  uid: 0,
  nickname: '',
  all: true,
  bits: [],
  durationMode: 'auto',
  reason: '违反社区规定',
  evidence: '',
  deleteContent: false,
  banAppeal: false,
  freezeUser: true
})

// 勾选「全面封禁」时取全部权限位；否则按勾选项按位或
const banTypeMask = computed(() => {
  if (ban.all) return BAN_TYPE_ALL
  return ban.bits.reduce((sum, bit) => sum | Number(bit), 0)
})

function openBan(item) {
  ban.uid = Number(item.id)
  ban.nickname = item.nickname || ''
  ban.all = true
  ban.bits = []
  ban.durationMode = 'auto'
  ban.reason = '违反社区规定'
  ban.evidence = ''
  ban.deleteContent = false
  ban.banAppeal = false
  ban.freezeUser = true
  ban.visible = true
}

async function submitBan() {
  if (!banTypeMask.value) {
    toast.warning('请至少选择一项限制，或勾选「全面封禁」')
    return
  }
  if (!ban.reason.trim()) {
    toast.warning('请输入封禁原因')
    return
  }

  const payload = {
    uid: ban.uid,
    ban_type: banTypeMask.value,
    reason: ban.reason.trim(),
    evidence: ban.evidence.trim(),
    delete_content: ban.deleteContent ? 1 : 0,
    ban_appeal: ban.banAppeal ? 1 : 0,
    freeze_user: ban.freezeUser ? 1 : 0
  }
  if (ban.durationMode === 'auto') {
    payload.auto_gradient = true
  } else {
    // 必须传字符串：后端对数字 0 会按「空值」处理并回落为自动梯度
    payload.duration = String(ban.durationMode)
  }

  ban.loading = true
  try {
    await banUser(payload)
    toast.success('已封禁该用户')
    ban.visible = false
    await afterMutation(0)
  } catch {
    /* 拦截器已提示 */
  } finally {
    ban.loading = false
  }
}

// ---------- 用户装扮（只读查看） ----------
// 数据来自 decoration/user-decorations（管理员），结构同「我的装扮」：
// { user, types, wearing, list }，list 按类型分组，含系统默认装扮。
const decos = reactive({
  visible: false,
  loading: false,
  uid: 0,
  nickname: '',
  groups: []
})

const DECORATION_SOURCE_TEXT = {
  shop: '商城兑换',
  admin: '管理员发放',
  activity: '活动获得',
  level: '等级解锁',
  gacha: '抽卡获得',
  default: '系统默认'
}

function decoSource(item) {
  return DECORATION_SOURCE_TEXT[item.source] || item.source
}

// 有效期文案：已过期 / 永久 / 剩余 N 天
function decoDuration(item) {
  if (item.expired) return '已过期'
  const expire = Number(item.expire_time) || 0
  if (!expire) return '永久有效'
  const days = Math.ceil((expire * 1000 - Date.now()) / 86400000)
  return days <= 0 ? '即将过期' : `剩余 ${days} 天`
}

async function openDecorations(item) {
  decos.uid = Number(item.id)
  decos.nickname = item.nickname || ''
  decos.groups = []
  decos.visible = true
  decos.loading = true
  try {
    const res = await getUserDecorations(decos.uid)
    const data = res?.data || {}
    decos.groups = (data.types || [])
      .map((type) => ({
        type: type.type,
        name: type.name,
        icon: type.icon,
        items: data.list?.[type.type] || []
      }))
      // 只展示该用户确实有内容的分类，避免一堆空分区
      .filter((group) => group.items.length > 0)
  } catch {
    decos.groups = []
  } finally {
    decos.loading = false
  }
}

// ---------- 权限组 ----------
const groups = reactive({
  visible: false,
  loading: false,
  fetching: false,
  uid: 0,
  nickname: '',
  all: [],
  selected: []
})

async function openGroups(item) {
  groups.uid = Number(item.id)
  groups.nickname = item.nickname || ''
  groups.all = []
  groups.selected = userGroupsOf(item).map((g) => Number(g.id))
  groups.visible = true
  groups.fetching = true
  try {
    const res = await listAuthGroups()
    groups.all = res.data?.data || []
  } catch {
    groups.all = []
  } finally {
    groups.fetching = false
  }
}

async function saveGroups() {
  groups.loading = true
  try {
    await setUserAuthGroups(groups.uid, groups.selected)
    toast.success('权限组已更新')
    groups.visible = false
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    groups.loading = false
  }
}

onMounted(load)

onUnmounted(() => {
  reqSeq += 1
})
</script>

<style scoped>
.user-list {
  display: flex;
  flex-direction: column;
}

/* ---------- 头部 ---------- */
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
}

/* ---------- 筛选栏 ---------- */
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
  /* 与筛选栏内的下拉 / 搜索框 / 按钮等高 */
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
  width: 200px;
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

/* ---------- 批量操作栏 ---------- */
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

/* ---------- 回收站提示 ---------- */
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

.user-items {
  display: flex;
  flex-direction: column;
}
.user-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 16px 8px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.user-row:last-child {
  border-bottom: none;
}
.user-row.is-selected {
  background: var(--accent-wash);
}

.pick {
  display: flex;
  align-items: center;
  align-self: center;
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
  opacity: 0.4;
}

.user-avatar {
  width: 46px;
  height: 46px;
  flex-shrink: 0;
  align-self: center;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  border: 1px solid var(--border);
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
  margin-bottom: 6px;
}
.user-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.flag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.flag.is-admin {
  color: var(--warning);
  background: var(--gold-wash);
}
.flag.is-self {
  color: var(--primary);
  background: var(--accent-soft);
}
.flag.is-group {
  color: var(--text-muted);
  background: var(--bg-muted);
}

.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 3px;
}
.status-chip.is-normal {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.status-chip.is-audit {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.status-chip.is-frozen {
  color: var(--warning);
  background: var(--gold-wash);
}
.status-chip.is-ban {
  color: var(--danger);
  background: var(--accent-soft);
}
/* 申诉中：琥珀色，提醒管理员有待处理 */
.status-chip.is-appeal {
  color: var(--warning);
  background: var(--gold-wash);
}
/* 申诉驳回：封禁继续生效，用偏暗的红区分「封禁中」 */
.status-chip.is-appeal-rejected {
  color: var(--danger);
  background: rgba(199, 72, 42, 0.08);
}

.user-contact {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--text-soft);
}
.contact-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.contact-item.is-empty {
  color: var(--text-light);
}
/* 第三方账号绑定：图标带一点主题色，与账号/邮箱这类本站信息区分开 */
.contact-item.is-oauth i {
  color: var(--primary);
}

.user-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 2px 8px;
  background: var(--bg-muted);
  border-radius: 3px;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}
.meta-text.is-warn {
  color: var(--warning);
}

.ban-line {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 8px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--danger);
  word-break: break-word;
}

.user-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  flex-shrink: 0;
  align-self: center;
  justify-content: flex-end;
}
.user-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.user-actions .ok:hover:not(:disabled) {
  border-color: var(--success);
  color: var(--success);
}
.user-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 账号状态单选（正常 / 待审核 / 冻结） ---------- */
.status-switch {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--bg-muted);
}
.status-option {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 22px;
  border-radius: 3px;
  color: var(--text-light);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
/* 原生单选框只保留语义与键盘操作，视觉交给 .status-option */
.status-radio {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.status-option:hover:not(.is-active):not(.is-disabled) {
  background: var(--bg-card);
  color: var(--primary);
}
.status-option:focus-within {
  outline: 2px solid var(--primary);
  outline-offset: 1px;
}
.status-option.is-disabled {
  cursor: not-allowed;
  opacity: 0.6;
}
.status-option.is-active.is-normal {
  background: rgba(108, 154, 77, 0.14);
  color: var(--success);
}
.status-option.is-active.is-audit {
  background: rgba(14, 165, 233, 0.14);
  color: #0ea5e9;
}
.status-option.is-active.is-frozen {
  background: var(--gold-wash);
  color: var(--warning);
}

/* ---------- 弹窗表单 ---------- */
.dialog-tip {
  margin: 0 0 16px;
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.dialog-tip strong {
  color: var(--text);
}
.dialog-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 13px;
  color: var(--text-muted);
}

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
/* 弹窗内字段间距略收紧，避免分组叠加后表单过长 */
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
/* 编辑弹窗里的「第三方账号」只读列表 */
.user-overview-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.user-overview-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--border-soft);
  border-radius: 4px;
  background: var(--bg-muted);
  font-size: 13px;
}
.user-overview-icon {
  font-size: 16px;
  color: var(--primary);
}
.user-overview-name {
  color: var(--text);
  font-weight: 600;
}
.user-overview-nick {
  color: var(--text-soft);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.user-overview-time {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-light);
}

.check-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px 12px;
  margin-top: 8px;
}
.check-grid.is-block {
  grid-template-columns: 1fr;
  margin-top: 0;
}
.check-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 0;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
.check-line input {
  width: 15px;
  height: 15px;
  flex-shrink: 0;
  accent-color: var(--primary);
  cursor: pointer;
}
.check-line.is-danger span {
  color: var(--danger);
}
.group-key {
  padding: 0 6px;
  font-size: 11px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: 3px;
}
.root-flag {
  margin-left: 6px;
  padding: 0 6px;
  font-size: 11px;
  color: var(--warning);
  background: var(--gold-wash);
  border-radius: 3px;
}

.avatar-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.avatar-box {
  position: relative;
  width: 64px;
  height: 64px;
  flex-shrink: 0;
  border-radius: 50%;
  /* 不能加 overflow: hidden：圆形裁剪会把右上角的「移除」按钮一起切掉，
     表现为按钮「被挡住」。圆角改由图片自己承担（见 .avatar-box img） */
  border: 1px solid var(--border);
  background: var(--bg-muted);
}
.avatar-box img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
}
.avatar-empty {
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
  border-radius: 50%;
}
.avatar-empty:hover {
  color: var(--primary);
}
/* 移除按钮：浮在圆形头像的右上角外侧，需要能超出容器，且盖在图片之上 */
.avatar-del {
  position: absolute;
  right: -4px;
  top: -4px;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  line-height: 1;
  cursor: pointer;
}
.avatar-del:hover {
  background: rgba(0, 0, 0, 0.75);
}
.avatar-fields {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.avatar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 处理申诉：待处理的操作按钮用警示色，便于在行内一眼找到 */
.btn.is-appeal {
  color: var(--warning);
}
.btn.is-appeal:hover:not(:disabled) {
  background: var(--gold-wash);
}

/* 处理申诉弹窗：封禁信息速览 + 申诉正文 */
.appeal-info {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin: 0 0 14px;
  padding: 12px 14px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.appeal-info dt {
  font-size: 11px;
  color: var(--text-muted);
}
.appeal-info dd {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--text);
  word-break: break-word;
}
.appeal-content {
  margin: 0;
  padding: 10px 12px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-left: 3px solid var(--warning);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  word-break: break-word;
  white-space: pre-wrap;
}

/* 清空封禁信息弹窗里的附加选项（ConfirmDialog 的默认插槽内容） */
.ban-clear-option {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
}
.ban-clear-option input {
  margin-top: 2px;
  flex-shrink: 0;
}

/* ---------- 查看用户装扮 ---------- */
.decoration-group + .decoration-group {
  margin-top: 14px;
}
.decoration-group-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-soft);
  font-size: 13px;
  font-weight: 600;
}
.decoration-group-count {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-muted);
}
.decoration-item-list {
  margin: 8px 0 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.decoration-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.decoration-thumb {
  width: 26px;
  height: 26px;
  object-fit: contain;
  flex-shrink: 0;
}
.decoration-thumb-icon {
  width: 26px;
  text-align: center;
  color: var(--primary);
}
.decoration-item-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.decoration-tag {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 11px;
  line-height: 18px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
}
.decoration-tag.is-wearing {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.decoration-tag.is-expired {
  background: rgba(217, 84, 77, 0.12);
  color: var(--danger);
}
.decoration-item-meta {
  margin-left: auto;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-muted);
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
  .user-row {
    flex-wrap: wrap;
  }
  .user-main {
    flex: 1 1 60%;
  }
  .user-actions {
    width: 100%;
  }
  .form-grid,
  .check-grid {
    grid-template-columns: 1fr;
  }
}
</style>
