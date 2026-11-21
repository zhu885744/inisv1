<template>
  <div class="room-chat">
    <!-- ==================== 顶部：房间资料 ==================== -->
    <header class="room-head">
      <!-- 窄屏（单栏）才需要返回；桌面端用左栏列表切换 -->
      <RouterLink to="/chat" class="room-icon-btn room-back" title="返回消息列表">
        <i class="bi bi-chevron-left" aria-hidden="true" />
      </RouterLink>

      <span class="room-avatar">
        <img v-if="room.avatar" :src="fullUrl(room.avatar)" alt="" loading="lazy" />
        <i v-else class="bi bi-people-fill" aria-hidden="true" />
      </span>

      <div class="room-title">
        <h1>
          {{ room.name || '公共群聊' }}
          <span v-if="isPublicRoom" class="room-badge">大厅</span>
          <span v-else-if="myRole === 2" class="room-badge">群主</span>
          <span v-else-if="myRole === 1" class="room-badge">管理员</span>
        </h1>
        <p>
          <span class="room-desc">{{ room.description || '全站用户都在这里，随便聊点什么' }}</span>
          <button type="button" class="room-online" @click="memberOpen = true">
            <i class="bi bi-broadcast" aria-hidden="true" />
            {{ isPublicRoom ? `${onlineCount} 人在线` : `${memberCount} 位成员` }}
          </button>
        </p>
      </div>

      <div class="room-actions">
        <span v-if="muted" class="room-mute-tag" :title="muteTip">
          <i class="bi bi-mic-mute" aria-hidden="true" />已禁言
        </span>
        <button v-if="canModerate" type="button" class="room-btn" @click="openAdmin">
          <i class="bi bi-sliders2" aria-hidden="true" />管理
        </button>
        <button
          v-if="!isPublicRoom && joined && myRole !== 2"
          type="button"
          class="room-btn"
          @click="leave"
        >
          <i class="bi bi-box-arrow-right" aria-hidden="true" />退出
        </button>
      </div>
    </header>

    <!-- 群公告：有公告时才出现，点击弹出全文（长公告不再撑高聊天区） -->
    <section
      v-if="room.notice"
      class="room-notice"
      role="button"
      tabindex="0"
      title="查看群公告"
      @click="noticeOpen = true"
      @keydown.enter.prevent="noticeOpen = true"
    >
      <i class="bi bi-megaphone" aria-hidden="true" />
      <span>{{ room.notice }}</span>
      <i class="bi bi-arrows-angle-expand room-notice-more" aria-hidden="true" />
    </section>

    <!-- ==================== 消息 ==================== -->
    <div ref="bodyRef" class="room-body">
      <div v-if="loading && !messages.length" class="room-tip">加载中…</div>

      <button
        v-else-if="hasMore"
        type="button"
        class="room-more"
        :disabled="loading"
        @click="loadHistory(messages[0]?.id)"
      >
        <i class="bi bi-clock-history" aria-hidden="true" />加载更早的消息
      </button>

      <template v-for="row in rows" :key="row.key">
        <div v-if="row.type === 'day'" class="room-day"><span>{{ row.label }}</span></div>

        <!-- 撤回：与参考图一致，整行居中提示，不占气泡 -->
        <div v-else-if="row.msg.is_revoked === 1" class="room-revoked">
          {{ revokeText(row.msg) }}
        </div>

        <div
          v-else
          class="room-msg"
          :class="{ 'is-mine': isMine(row.msg) }"
          @contextmenu.prevent="openMessageMenu($event, row.msg)"
        >
          <button
            type="button"
            class="room-msg-avatar"
            :title="canModerate && !isMine(row.msg) ? '管理该用户' : ''"
            @click="onAvatarClick(row.msg)"
          >
            <img :src="avatarOf(row.msg)" :alt="nameOf(row.msg)" loading="lazy" />
          </button>

          <div class="room-msg-main">
            <div class="room-msg-meta">
              <span class="room-msg-name">{{ isMine(row.msg) ? '我' : nameOf(row.msg) }}</span>
              <time>{{ shortTime(row.msg.create_time) }}</time>
              <button
                v-if="canRevoke(row.msg)"
                type="button"
                class="room-msg-revoke"
                @click="revoke(row.msg)"
              >撤回</button>
            </div>

            <div class="room-bubble">
              <!-- 引用回复：展示被引用消息的快照（原消息撤回后依然可读） -->
              <div v-if="row.msg.quote" class="room-quote" :title="quoteTitle(row.msg.quote)">
                <span class="room-quote-name">{{ quoteName(row.msg.quote) }}</span>
                <span
                  class="room-quote-text"
                  v-html="renderEmojiWithBreaks(quotePreview(row.msg.quote), { size: 16 })"
                />
              </div>
              <img
                v-if="row.msg.msg_type === 'image'"
                class="room-image"
                :src="fullUrl(row.msg.content)"
                alt="图片消息"
                loading="lazy"
                @click="openLightbox([fullUrl(row.msg.content)])"
              />
              <span v-else class="room-text" v-html="renderEmojiWithBreaks(row.msg.content, { size: 22 })" />
            </div>
          </div>
        </div>
      </template>

      <p v-if="!loading && !messages.length" class="room-tip">还没有人说话，来打个招呼吧</p>
    </div>

    <!-- ==================== 输入 ==================== -->
    <footer class="room-composer">
      <!-- 游客：消息照常可看，发言 / 加入要先登录 -->
      <div v-if="isGuest" class="room-join">
        <p>
          <i class="bi bi-person-lock" aria-hidden="true" />
          登录后即可参与聊天
        </p>
        <button type="button" class="room-btn room-btn-primary" @click="requireLogin()">去登录</button>
      </div>

      <!-- 非成员：先加入才能看消息、发言 -->
      <div v-else-if="!joined" class="room-join">
        <p>
          <i class="bi bi-lock" aria-hidden="true" />
          {{ Number(room.join_type) === 1 ? '该群聊需要群主审批后加入' : Number(room.join_type) === 2 ? '该群聊仅限邀请加入' : '加入后即可查看消息并参与聊天' }}
        </p>
        <FormInput
          v-if="Number(room.join_type) === 1"
          v-model="joinMessage"
          maxlength="120"
          placeholder="申请留言（可选）" />
        <button
          type="button"
          class="room-btn room-btn-primary"
          :disabled="joinPending || Number(room.join_type) === 2"
          @click="join"
        >{{ joinPending ? '等待群主审批…' : Number(room.join_type) === 2 ? '仅限邀请' : '申请加入' }}</button>
      </div>

      <p v-else-if="muted" class="room-blocked">
        <i class="bi bi-mic-mute" aria-hidden="true" />{{ muteTip }}
      </p>

      <template v-else>
        <!-- 有人正在输入：只在有人打字时出现，4 秒无新上报自动消失 -->
        <p v-if="typingText" class="room-typing">
          <i class="bi bi-three-dots" aria-hidden="true" />{{ typingText }}
        </p>

        <form class="room-composer-box" @submit.prevent="send">
        <!-- @ 选人：输入 @ 或点工具栏的 @ 按钮弹出，只列在线成员 -->
        <div
          v-if="mentionOpen && mentionCandidates.length"
          class="room-mention"
          @mousedown.prevent
        >
          <p class="room-mention-title">选择要 @ 的人</p>
          <button
            v-for="item in mentionCandidates"
            :key="item.uid"
            type="button"
            class="room-mention-item"
            @mousedown.prevent
            @click="pickMention(item)"
          >
            <i class="bi bi-at" aria-hidden="true" />{{ item.name }}
          </button>
        </div>

        <!-- 引用态：显示被引用消息的摘要，发送后自动清除 -->
        <div v-if="quoteDraft" class="room-quote-draft">
          <div class="room-quote-draft-body">
            <span class="room-quote-name">{{ quoteName(quoteDraft) }}</span>
            <span
              class="room-quote-text"
              v-html="renderEmojiWithBreaks(quotePreview(quoteDraft), { size: 16 })"
            />
          </div>
          <button type="button" class="room-icon-btn" title="取消引用" @click="clearQuote">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </div>

        <div class="room-tools">
          <div class="room-emoji-wrap">
            <button type="button" class="room-icon-btn" title="插入表情" @click="emojiOpen = !emojiOpen">
              <i class="bi bi-emoji-smile" aria-hidden="true" />
            </button>
            <EmojiPicker v-model="emojiOpen" @select="insertEmoji" />
          </div>

          <button
            type="button"
            class="room-icon-btn"
            title="提到某人"
            @click="openMention"
          >
            <i class="bi bi-at" aria-hidden="true" />
          </button>

          <button
            type="button"
            class="room-icon-btn"
            :title="uploading ? '图片上传中…' : '发送图片'"
            :disabled="uploading"
            @click="pickImage"
          >
            <i :class="uploading ? 'bi bi-arrow-repeat is-spin' : 'bi bi-image'" aria-hidden="true" />
          </button>
          <input
            ref="fileRef"
            class="room-file"
            type="file"
            accept="image/*"
            multiple
            @change="onFilesPicked"
          />
        </div>

        <!-- 用 contenteditable 承载编辑：表情在输入框里直接显示成图片，
             存储格式仍是 [emoji:url] 纯文本（见 components/ChatEmojiInput.vue） -->
        <ChatEmojiInput
          ref="inputRef"
          v-model="draft"
          class="room-input"
          :maxlength="2000"
          placeholder="说点什么…"
          @keydown="onInputKeydown"
          @input="onInput"
          @send="send"
          @files="uploadAndSendImages"
        />

        <button type="submit" class="room-send" :disabled="sending || !draft.trim()" title="发送（Enter）">
          <i v-if="sending" class="bi bi-arrow-repeat is-spin" aria-hidden="true" />
          <template v-else>发送</template>
        </button>
      </form>
      </template>
    </footer>

    <!-- ==================== 右侧：群公告 / 群成员（宽屏显示，QQ NT 布局） ==================== -->
    <aside class="room-aside">
      <section class="room-aside-block">
        <header class="room-aside-head">
          <span>群公告</span>
          <button v-if="canManage" type="button" class="room-aside-link" @click="openNoticeEditor">
            编辑
          </button>
        </header>
        <p
          class="room-aside-text"
          :class="{ 'is-clickable': !!room.notice }"
          :title="room.notice ? '查看群公告' : ''"
          @click="room.notice && (noticeOpen = true)"
        >{{ room.notice || '暂无公告' }}</p>
      </section>

      <section class="room-aside-block is-grow">
        <header class="room-aside-head">
          <span>{{ isPublicRoom ? '在线成员' : '群成员' }} {{ memberRows.length }}</span>
          <button type="button" class="room-aside-link" @click="memberOpen = true">查看全部</button>
        </header>

        <ul class="room-aside-members">
          <li v-for="item in memberRows.slice(0, 30)" :key="item.uid" class="room-aside-member">
            <span class="room-aside-avatar">{{ (item.name || '?').slice(0, 1) }}</span>
            <span class="room-aside-name">{{ item.name }}</span>
            <em v-if="item.uid === me" class="room-role-tag">我</em>
            <em v-else-if="item.role" class="room-role-tag">{{ item.roleName }}</em>
          </li>
          <li v-if="!memberRows.length" class="room-aside-empty">还没有成员</li>
        </ul>
      </section>
    </aside>

    <!-- ==================== 管理面板（仅超级管理员） ==================== -->
    <div v-if="adminOpen" class="room-admin">
      <div class="room-admin-mask" @click="adminOpen = false" />

      <aside class="room-admin-panel">
        <header class="room-admin-head">
          <span>群聊管理</span>
          <button type="button" class="room-icon-btn" title="关闭" @click="adminOpen = false">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <div class="room-admin-tabs">
          <button
            v-if="canManage"
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'profile' }]"
            @click="adminTab = 'profile'"
          >房间资料</button>
          <button
            v-if="canManage"
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'notice' }]"
            @click="adminTab = 'notice'"
          >群公告</button>
          <button
            v-if="!isPublicRoom"
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'members' }]"
            @click="adminTab = 'members'; loadMembers()"
          >成员{{ members.length ? ` (${members.length})` : '' }}</button>
          <button
            v-if="!isPublicRoom && canModerate"
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'requests' }]"
            @click="adminTab = 'requests'; loadRequests()"
          >申请{{ requests.length ? ` (${requests.length})` : '' }}</button>
          <button
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'mutes' }]"
            @click="switchMutes"
          >禁言名单{{ mutes.length ? ` (${mutes.length})` : '' }}</button>
          <button
            v-if="isPublicRoom"
            type="button"
            :class="['room-admin-tab', { 'is-active': adminTab === 'clear' }]"
            @click="switchClear"
          >自动清空</button>
        </div>

        <!-- 解散是不可逆操作：单独放在 tab 下方，避免和「保存」挨着误点 -->
        <div v-if="canManage && !isPublicRoom" class="room-admin-danger">
          <button
            type="button"
            class="room-btn room-btn-danger"
            :disabled="dissolving"
            @click="dissolve"
          >
            <i class="bi bi-trash3" aria-hidden="true" />{{ dissolving ? '解散中…' : '解散群聊' }}
          </button>
          <span>解散后成员与申请都会被清空，消息不再对成员可见</span>
        </div>

        <div v-if="adminTab === 'profile'" class="room-admin-body">
          <label class="room-field">
            <span>名称</span>
            <FormInput v-model="form.name" maxlength="30" placeholder="最多 30 个字" />
          </label>

          <label class="room-field">
            <span>简介</span>
            <FormTextarea v-model="form.description" rows="3" maxlength="200" placeholder="最多 200 个字" />
          </label>

          <label class="room-field">
            <span>头像</span>
            <div class="room-avatar-row">
              <span class="room-avatar room-avatar-sm">
                <img v-if="form.avatar" :src="fullUrl(form.avatar)" alt="" />
                <i v-else class="bi bi-image" aria-hidden="true" />
              </span>
              <FormInput v-model="form.avatar" placeholder="图片地址，或直接上传" />
            </div>
          </label>

          <label v-if="!isPublicRoom" class="room-field">
            <span>入群方式</span>
            <FormSelect
              v-model="form.join_type"
              :options="joinTypeOptions"
              placeholder="请选择入群方式" />
          </label>

          <div class="room-admin-actions">
            <button type="button" class="room-btn" :disabled="avatarUploading" @click="pickAvatar">
              <i :class="avatarUploading ? 'bi bi-arrow-repeat is-spin' : 'bi bi-upload'" aria-hidden="true" />
              上传头像
            </button>
            <button type="button" class="room-btn room-btn-primary" :disabled="saving" @click="saveRoom">
              {{ saving ? '保存中…' : '保存修改' }}
            </button>
          </div>

          <input ref="avatarInput" class="room-file" type="file" accept="image/*" @change="onAvatarPicked" />
        </div>

        <div v-else-if="adminTab === 'notice'" class="room-admin-body">
          <label class="room-field">
            <span>群公告</span>
            <FormTextarea
              v-model="noticeDraft"
              rows="5"
              maxlength="500"
              placeholder="最多 500 个字，留空即清除公告" />
          </label>

          <p class="room-tip">公告会展示在房间顶部，所有成员可见（不会推送新的消息提醒）。</p>

          <div class="room-admin-actions">
            <button type="button" class="room-btn room-btn-primary" @click="saveNotice">发布公告</button>
          </div>
        </div>

        <div v-else-if="adminTab === 'members'" class="room-admin-body">
          <div v-if="canModerate" class="room-invite">
            <div class="room-invite-row">
              <FormInput v-model="inviteUid" type="number" min="1" placeholder="用户 ID" />
              <button type="button" class="room-btn room-btn-primary" :disabled="inviting" @click="invite">
                {{ inviting ? '邀请中…' : '邀请' }}
              </button>
            </div>
            <button type="button" class="room-invite-link" @click="inviteOpen = true">
              <i class="bi bi-qr-code" aria-hidden="true" />链接 / 二维码邀请
            </button>
          </div>

          <div v-for="item in members" :key="item.uid" class="room-mute-row">
            <img class="room-mute-avatar" :src="avatarOfUser({ avatar: item.avatar })" alt="" loading="lazy" />
            <span class="room-mute-name">
              {{ item.name }}
              <em v-if="item.role" class="room-role-tag">{{ item.roleName }}</em>
            </span>
            <button
              v-if="canManage && item.role !== 2 && item.uid !== me"
              type="button"
              class="room-btn room-btn-sm"
              @click="toggleRole(item)"
            >{{ item.role === 1 ? '撤销管理' : '设为管理' }}</button>
            <button
              v-if="canModerate && item.role !== 2 && item.uid !== me"
              type="button"
              class="room-btn room-btn-sm room-btn-danger"
              @click="removeMember(item)"
            >移出</button>
          </div>

          <p v-if="!members.length" class="room-tip">还没有成员</p>
        </div>

        <div v-else-if="adminTab === 'requests'" class="room-admin-body">
          <div v-for="item in requests" :key="item.id" class="room-mute-row">
            <img class="room-mute-avatar" :src="avatarOfUser({ avatar: item.avatar })" alt="" loading="lazy" />
            <span class="room-mute-name">
              {{ item.name }}
              <em>{{ item.message || '未填写留言' }}</em>
            </span>
            <button type="button" class="room-btn room-btn-sm room-btn-primary" @click="handleRequest(item, true)">通过</button>
            <button type="button" class="room-btn room-btn-sm" @click="handleRequest(item, false)">拒绝</button>
          </div>

          <p v-if="!requests.length" class="room-tip">没有待处理的入群申请</p>
        </div>

        <div v-else-if="adminTab === 'mutes'" class="room-admin-body">
          <div v-for="item in mutes" :key="item.uid" class="room-mute-row">
            <img class="room-mute-avatar" :src="avatarOfUser(item.user)" alt="" loading="lazy" />
            <span class="room-mute-name">{{ item.user?.nickname || `用户 #${item.uid}` }}</span>
            <em>{{ untilText(item.until) }}</em>
            <button type="button" class="room-btn room-btn-sm" @click="unmute(item.uid)">解除</button>
          </div>

          <p v-if="!mutes.length" class="room-tip">当前没有被禁言的用户</p>
        </div>

        <div v-else-if="adminTab === 'clear'" class="room-admin-body">
          <label class="room-field">
            <span>清空间隔</span>
            <FormSelect
              v-model="clearDays"
              :options="clearOptions"
              placeholder="请选择清空间隔" />
          </label>

          <p class="room-tip">
            到期后服务端会直接删除公共群聊的全部消息（物理删除，无法恢复）。
            默认每 3 天（72 小时）清空一次。
          </p>

          <p class="room-tip">
            上次清空：{{ clearInfo.last_time ? formatTime(clearInfo.last_time) : '尚未清空过（首次计时中）' }}<br />
            下次清空：{{ clearInfo.next_time ? formatTime(clearInfo.next_time) : '保存后按间隔计算' }}<br />
            上次清空条数：{{ clearInfo.last_count || 0 }}
          </p>

          <div class="room-admin-actions">
            <button
              type="button"
              class="room-btn room-btn-primary"
              :disabled="clearSaving"
              @click="saveAutoClear"
            >{{ clearSaving ? '保存中…' : '保存设置' }}</button>
          </div>
        </div>
      </aside>
    </div>

    <!-- ==================== 管理员点用户头像：禁言操作 ==================== -->
    <div v-if="userPanel" class="room-modal">
      <div class="room-modal-mask" @click="userPanel = null" />
      <div class="room-modal-card">
        <div class="room-modal-user">
          <img :src="avatarOfUser({ avatar: userPanel.avatar })" alt="" />
          <span>{{ userPanel.nickname }}</span>
        </div>
        <p class="room-modal-tip">禁言后该用户不能发言，已在列表中的消息保留</p>
        <div class="room-modal-actions">
          <button type="button" class="room-btn" @click="mute(userPanel.uid, 10)">禁言 10 分钟</button>
          <button type="button" class="room-btn" @click="mute(userPanel.uid, 60)">禁言 1 小时</button>
          <button type="button" class="room-btn" @click="mute(userPanel.uid, 1440)">禁言 1 天</button>
          <button type="button" class="room-btn room-btn-danger" @click="mute(userPanel.uid, 0)">永久禁言</button>
          <button
            v-if="canModerate && !isPublicRoom"
            type="button"
            class="room-btn room-btn-danger"
            @click="removeMember({ uid: userPanel.uid, name: userPanel.nickname }); userPanel = null"
          >移出群聊</button>
        </div>
      </div>
    </div>

    <!-- ==================== 在线成员 ==================== -->
    <div v-if="memberOpen" class="room-modal">
      <div class="room-modal-mask" @click="memberOpen = false" />
      <div class="room-modal-card room-member-card">
        <header class="room-member-head">
          <span>
            {{ isPublicRoom ? `在线成员（${memberRows.length}）` : `群成员（${memberRows.length}）` }}
          </span>
          <button type="button" class="room-icon-btn" title="关闭" @click="memberOpen = false">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <ul class="room-member-list">
          <li v-for="item in memberRows" :key="item.uid" class="room-member-row">
            <span class="room-member-avatar">{{ (item.name || '?').slice(0, 1) }}</span>
            <span class="room-member-name">
              {{ item.name }}
              <em v-if="item.uid === me" class="room-member-self">我</em>
              <em v-else-if="item.role" class="room-role-tag">{{ item.roleName }}</em>
            </span>
            <button
              v-if="item.uid !== me && !muted"
              type="button"
              class="room-btn room-btn-sm"
              @click="mentionMember(item)"
            >@ TA</button>
          </li>

          <li v-if="!memberRows.length" class="room-tip">
            {{ isPublicRoom ? '暂时没有其他人在线' : '还没有成员' }}
          </li>
        </ul>
      </div>
    </div>

    <!-- ==================== 群公告弹窗 ==================== -->
    <!-- 点顶部公告条 / 右侧公告栏打开：长公告在弹窗里滚动，不再挤压聊天区 -->
    <div v-if="noticeOpen" class="room-modal">
      <div class="room-modal-mask" @click="noticeOpen = false" />
      <div class="room-modal-card room-notice-card">
        <header class="room-notice-head">
          <span>
            <i class="bi bi-megaphone-fill" aria-hidden="true" />
            群公告
          </span>
          <button type="button" class="room-icon-btn" title="关闭" @click="noticeOpen = false">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <div class="room-notice-body">{{ room.notice || '暂无公告' }}</div>

        <footer v-if="canManage" class="room-notice-foot">
          <button type="button" class="room-btn" @click="editNotice">编辑公告</button>
        </footer>
      </div>
    </div>

    <!-- ==================== 邀请（链接 / 二维码） ==================== -->
    <InviteDialog
      v-if="inviteOpen"
      :room-id="roomId"
      :room-name="room.name || '群聊'"
      @close="inviteOpen = false"
      @share="shareInvite"
    />

    <!-- ==================== 消息右键菜单 ==================== -->
    <div
      v-if="messageMenu.open"
      class="room-menu-mask"
      @click="closeMessageMenu"
      @contextmenu.prevent="closeMessageMenu"
    />
    <div
      v-if="messageMenu.open"
      class="room-menu"
      role="menu"
      :style="{ left: `${messageMenu.x}px`, top: `${messageMenu.y}px` }"
    >
      <button
        v-if="!isGuest && messageMenu.msg?.is_revoked !== 1"
        type="button"
        class="room-menu-item"
        role="menuitem"
        @click="quoteMessage(messageMenu.msg)"
      >
        <i class="bi bi-quote" aria-hidden="true" />引用消息
      </button>
      <button
        v-if="canRevoke(messageMenu.msg)"
        type="button"
        class="room-menu-item"
        role="menuitem"
        @click="revokeFromMenu(messageMenu.msg)"
      >
        <i class="bi bi-arrow-counterclockwise" aria-hidden="true" />撤回消息
      </button>
      <button
        v-if="!isGuest"
        type="button"
        class="room-menu-item is-danger"
        role="menuitem"
        @click="deleteFromMenu(messageMenu.msg)"
      >
        <i class="bi bi-trash3" aria-hidden="true" />删除消息
      </button>
      <button type="button" class="room-menu-item" role="menuitem" @click="copyMessage(messageMenu.msg)">
        <i class="bi bi-clipboard" aria-hidden="true" />复制
      </button>
    </div>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InviteDialog from './InviteDialog.vue'
import {
  addRoomMember,
  announceRoom,
  deleteRoomMessage,
  dissolveRoom,
  getRoomAutoClear,
  getRoomInfo,
  handleRoomRequest,
  joinRoom,
  leaveRoom,
  listRoomMembers,
  listRoomMessages,
  listRoomMutes,
  listRoomRequests,
  markRoomRead,
  muteRoomUser,
  removeRoomMember,
  revokeRoomMessage,
  saveRoomAutoClear,
  sendRoomMessage,
  setRoomMemberRole,
  unmuteRoomUser,
  updateRoom
} from '@/api/room'
import { uploadAttachments } from '@/api/attachment'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { fromNow, formatTime } from '@/utils/time'
import { openLightbox } from '@/utils/lightbox'
import { getFullUrl, renderEmojiWithBreaks } from '@/utils/emoji'
import EmojiPicker from '@/components/EmojiPicker.vue'
import ChatEmojiInput from '@/components/ChatEmojiInput.vue'

/**
 * 公共群聊（/room/1）
 *
 * 与私信页的关系：
 *   - 复用同一套视觉与输入体验（表情 / 图片 / 粘贴上传），但群聊没有「已读 / 未读」，
 *     只 maintain 一条全局时间线，所以不复用 stores/chat 的会话逻辑；
 *   - 实时消息走同一个 WebSocket：stores/chat.js 把 room* 事件原样转给 onEvent，
 *     这里订阅后做「上屏 / 撤回 / 改资料 / 被禁言」四种处理。
 *
 * 权限：
 *   - 所有登录用户：看消息、发言、2 分钟内撤回自己的消息；
 *   - 超级管理员（后端 model.IsRootAdmin）：撤回任意消息、禁言 / 解禁、改名称 / 简介 / 头像。
 *     前端的 is_admin 只用来决定「按钮要不要显示」，真正的校验在后端。
 */
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()

const roomId = computed(() => Number(route.params.id || 1) || 1)

const room = ref({})
const messages = ref([])
const hasMore = ref(false)
const loading = ref(false)

const isAdmin = ref(false)
const isModerator = ref(false)
const muted = ref(false)
const mutedUntil = ref(0)

// 群聊身份：joined 是否成员 / myRole -1 非成员 0 成员 1 管理员 2 群主
const joined = ref(true)
const myRole = ref(-1)
const isPublicRoom = computed(() => roomId.value === 1)
const canManage = computed(() => isAdmin.value)
const canModerate = computed(() => isModerator.value)
const memberCount = computed(() => Number(room.value.member_count) || 0)

// 成员 / 申请 / 公告 / 解散
const members = ref([])
const requests = ref([])
const noticeOpen = ref(false)
const noticeDraft = ref('')
const dissolving = ref(false)
const joinMessage = ref('')
const joinPending = ref(false)
// 邀请入群：按用户 id 拉人，或生成链接 / 二维码
const inviteUid = ref('')
const inviting = ref(false)
const inviteOpen = ref(false)

const draft = ref('')
const sending = ref(false)
const uploading = ref(false)
const emojiOpen = ref(false)

// 引用态：当前正在引用的消息（null = 未引用）
const quoteDraft = ref(null)
// 消息右键菜单：open / 坐标 / 目标消息
const messageMenu = ref({ open: false, x: 0, y: 0, msg: null })

const bodyRef = ref(null)
const inputRef = ref(null)
const fileRef = ref(null)

// 管理面板
const adminOpen = ref(false)
const adminTab = ref('profile')
const mutes = ref([])
const saving = ref(false)
const avatarUploading = ref(false)
const avatarInput = ref(null)
const form = ref({ name: '', description: '', avatar: '', join_type: 0 })

/**
 * 资料表单回显：加载房间信息 / 打开管理面板都走这里
 *
 * 只有一个来源，字段才不会漏 —— 之前 openAdmin 里手写了一份 form，
 * 少了 join_type，表现就是「设置过入群方式，刷新或重开面板后下拉又变回请选择」。
 */
function fillForm() {
  form.value = {
    name: room.value.name || '',
    description: room.value.description || '',
    avatar: room.value.avatar || '',
    join_type: Number(room.value.join_type) || 0
  }
}

// 自动清空：默认 3 天（72 小时），其余选项与后端 model.RoomAutoClearOptions 对齐
const clearOptions = [
  { value: 1, label: '1 天' },
  { value: 3, label: '3 天（默认，72 小时）' },
  { value: 7, label: '1 星期' },
  { value: 30, label: '30 天' },
  { value: 182, label: '半年' },
  { value: 365, label: '一年' }
]
const clearDays = ref(3)
const clearSaving = ref(false)
const clearInfo = ref({ days: 3, last_time: 0, last_count: 0, next_time: 0 })

// 管理员点某条消息头像后弹出的用户操作
const userPanel = ref(null)

// 在线成员面板
const memberOpen = ref(false)

// @ 选人：mentionKeyword 是 @ 后面已经输入的部分，用来过滤候选
const mentionOpen = ref(false)
const mentionKeyword = ref('')
// 这条消息要提醒的人（只存 uid，正文里已经有 @昵称）
const mentionUids = ref([])

// 正在输入的人（uid → { name }）：4 秒没有新上报就移除
const typers = ref(new Map())
const typingTimers = new Map()

const me = computed(() => Number(userStore.user?.id || 0))
// 游客：可以只读浏览公共大厅（后端 room/info、room/messages 是 common），发言 / 加入要先登录
const isGuest = computed(() => !userStore.isLogged)

// 在线成员来自 chat store 的 status 广播（已按 uid 去重，不含访客）
const onlineMembers = computed(() => chatStore.presence.users)
const onlineCount = computed(() => onlineMembers.value.length)

// 成员面板：大厅看在线的（全员隐式成员），自建群看后端返回的成员表
const memberRows = computed(() =>
  isPublicRoom.value
    ? onlineMembers.value
    : members.value.map((item) => ({
        uid: item.uid,
        name: item.name,
        role: item.role,
        roleName: item.roleName
      }))
)

// 入群方式（与后端 model.ChatRoomJoin* 常量对齐）
const joinTypeOptions = [
  { value: 0, label: '自由加入' },
  { value: 1, label: '需要审批' },
  { value: 2, label: '仅限邀请' }
]

/** @ 候选：排除自己，按已输入的关键字过滤，最多 8 个 */
const mentionCandidates = computed(() => {
  const kw = mentionKeyword.value.trim().toLowerCase()
  return onlineMembers.value
    .filter((item) => Number(item.uid) !== me.value)
    .filter((item) => !kw || String(item.name || '').toLowerCase().includes(kw))
    .slice(0, 8)
})

/** 输入状态文案：一个人直接念名字，多个人只报数量，避免占满一行 */
const typingText = computed(() => {
  const names = [...typers.value.values()].map((item) => item.name).filter(Boolean)
  if (!names.length) return ''
  if (names.length === 1) return `${names[0]} 正在输入…`
  return `${names[0]} 等 ${names.length} 人正在输入…`
})

const muteTip = computed(() => {
  if (!muted.value) return ''
  if (mutedUntil.value > 0) return `你已被禁言，${formatTime(mutedUntil.value)} 解除`
  return '你已被禁言，暂时不能发言'
})

// 没有头像时的占位图：内联 SVG，省一次 404
const DEFAULT_AVATAR =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="34" fill="%238a8a82" font-family="serif">聊</text></svg>'

function fullUrl(url) {
  return getFullUrl(url || '')
}

function isMine(msg) {
  return Number(msg.uid) === me.value
}

function nameOf(msg) {
  return msg.user?.nickname || `用户 #${msg.uid}`
}

/** 头像地址：空值统一走占位图，避免 img src="" 又请求一次当前页面 */
function avatarOfUser(user) {
  const avatar = user?.avatar || ''
  return avatar ? getFullUrl(avatar) : DEFAULT_AVATAR
}

function avatarOf(msg) {
  return avatarOfUser(msg.user)
}

/** 撤回提示文案：本人撤回 / 别人撤回 / 被管理员撤回 */
function revokeText(msg) {
  const bySelf = Number(msg.revoke_uid || 0) === Number(msg.uid || 0)
  if (bySelf) return isMine(msg) ? '你撤回了一条消息' : `${nameOf(msg)} 撤回了一条消息`
  return `${nameOf(msg)} 的一条消息被管理员撤回`
}

/** 本人 2 分钟内可撤回；群主 / 群管理员 / 超级管理员对任何未撤回消息都可撤回 */
function canRevoke(msg) {
  if (!msg || msg.is_revoked === 1) return false
  if (canModerate.value) return true
  if (!isMine(msg)) return false
  return Date.now() / 1000 - Number(msg.create_time || 0) <= 120
}

// ---------- 引用回复 ----------

/**
 * 被引用消息的展示名：自己 / 对方
 *
 * quote 快照里只有 uid，没有昵称；这里从当前时间线里反查一次发送者昵称，
 * 找不到再退回「用户 #uid」。兼容引用态传来的整条消息（from_uid → uid）。
 */
function quoteName(quote) {
  if (!quote) return ''

  const uid = Number(quote.uid ?? quote.from_uid ?? 0)
  if (uid === me.value) return '我'
  if (quote.nickname) return quote.nickname

  const found = messages.value.find((item) => Number(item.uid) === uid && item.user?.nickname)
  return found?.user?.nickname || `用户 #${uid}`
}

/** 被引用消息的摘要：服务端已落快照，这里只做类型兜底 */
function quotePreview(quote) {
  if (!quote) return ''
  if (quote.msg_type === 'image') return '[图片]'
  return String(quote.content || '')
}

function quoteTitle(quote) {
  return quotePreview(quote).replace(/\[emoji:[^\]]+\]/g, '[表情]')
}

function clearQuote() {
  quoteDraft.value = null
}

// ---------- 消息右键菜单 ----------

/** 打开右键菜单：坐标按视口夹紧，避免贴边时菜单跑出屏幕 */
function openMessageMenu(event, msg) {
  if (!msg?.id) return

  const width = 148
  const height = canRevoke(msg) ? 172 : 138
  const x = Math.min(Number(event.clientX) || 0, window.innerWidth - width - 8)
  const y = Math.min(Number(event.clientY) || 0, window.innerHeight - height - 8)

  messageMenu.value = { open: true, x: Math.max(8, x), y: Math.max(8, y), msg }
}

function closeMessageMenu() {
  if (!messageMenu.value.open) return
  messageMenu.value = { ...messageMenu.value, open: false, msg: null }
}

function quoteMessage(msg) {
  closeMessageMenu()
  if (!msg?.id) return

  // 带上昵称：引用态立即可读，不必等 quoteName 去时间线里反查
  quoteDraft.value = { ...msg, nickname: nameOf(msg) }
  nextTick(() => inputRef.value?.focus())
}

function revokeFromMenu(msg) {
  closeMessageMenu()
  revoke(msg)
}

async function deleteFromMenu(msg) {
  closeMessageMenu()
  if (!msg?.id) return

  try {
    const res = await deleteRoomMessage(roomId.value, msg.id)
    if (res?.code !== 200) return

    messages.value = messages.value.filter((item) => Number(item.id) !== Number(msg.id))
    if (Number(quoteDraft.value?.id) === Number(msg.id)) clearQuote()
    toast.success('已删除')
  } catch {
    /* 业务错误已由 request.js 统一提示 */
  }
}

/** 复制：文本去掉 [emoji:url] 标记，图片复制地址 */
async function copyMessage(msg) {
  closeMessageMenu()
  if (!msg) return

  let text = ''
  if (msg.is_revoked === 1) {
    text = `该消息已被${Number(msg.revoke_uid) === Number(msg.uid) ? '' : '管理员'}撤回`
  } else if (msg.msg_type === 'image') {
    text = fullUrl(msg.content)
  } else {
    text = String(msg.content || '').replace(/\[emoji:[^\]]+\]/g, '[表情]')
  }

  if (!text) return

  try {
    await navigator.clipboard.writeText(text)
    toast.success('已复制')
  } catch {
    // 非 HTTPS / 无剪贴板权限：退回 textarea + execCommand
    const el = document.createElement('textarea')
    el.value = text
    el.setAttribute('readonly', '')
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    try {
      document.execCommand('copy')
      toast.success('已复制')
    } catch {
      toast.warning('复制失败，请手动选择文字')
    }
    document.body.removeChild(el)
  }
}

function onMenuKeydown(e) {
  if (e.key !== 'Escape') return
  closeMessageMenu()
  mentionOpen.value = false
  memberOpen.value = false
}

function onMenuScroll() {
  closeMessageMenu()
}

// ---------- 时间 ----------

function shortTime(ts) {
  if (!ts) return ''
  const diff = Date.now() / 1000 - Number(ts)
  if (diff < 60) return '刚刚'
  if (diff < 86400) return fromNow(ts)
  return formatTime(ts).slice(5, 16)
}

function untilText(until) {
  const value = Number(until) || 0
  if (value <= 0) return '永久'
  return `${formatTime(value)} 解除`
}

function dayKey(ts) {
  const value = Number(ts) || 0
  if (value <= 0) return ''
  const date = new Date(value * 1000)
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`
}

function dayLabel(ts) {
  const value = Number(ts) || 0
  if (value <= 0) return ''

  const date = new Date(value * 1000)
  const now = new Date()
  const startOf = (item) => new Date(item.getFullYear(), item.getMonth(), item.getDate()).getTime()
  const diffDays = Math.round((startOf(now) - startOf(date)) / 86400000)

  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'
  if (date.getFullYear() === now.getFullYear()) return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
  return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月 ${date.getDate()} 日`
}

/** 按天分段：跨天插日期分隔，长记录更容易定位 */
const rows = computed(() => {
  const list = []
  let lastDay = ''

  messages.value.forEach((msg) => {
    const day = dayKey(msg.create_time)
    if (day && day !== lastDay) {
      list.push({ type: 'day', key: `day-${day}-${msg.id}`, label: dayLabel(msg.create_time) })
      lastDay = day
    }
    list.push({ type: 'msg', key: `msg-${msg.id}`, msg })
  })

  return list
})

// ---------- 数据 ----------

function scrollToBottom() {
  nextTick(() => {
    const el = bodyRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

function maxSeq(list) {
  let max = 0
  ;(list || []).forEach((item) => {
    const seq = Number(item.seq || item.id) || 0
    if (seq > max) max = seq
  })
  return max
}

/** 合并去重（实时推送与历史/补拉可能先后到达同一条） */
function mergeMessages(list) {
  if (!list?.length) return 0

  const seen = new Set(messages.value.map((item) => Number(item.id)))
  const added = list.filter((item) => item && item.id && !seen.has(Number(item.id)))
  if (!added.length) return 0

  messages.value = [...messages.value, ...added].sort(
    (a, b) => Number(a.seq || a.id) - Number(b.seq || b.id)
  )

  return added.length
}

async function loadInfo() {
  try {
    const res = await getRoomInfo(roomId.value)
    if (res?.code !== 200) return

    room.value = res.data?.room || {}
    isAdmin.value = !!res.data?.is_admin
    isModerator.value = !!res.data?.is_moderator
    muted.value = !!res.data?.muted
    mutedUntil.value = Number(res.data?.muted_until || 0)
    joined.value = res.data?.room?.joined !== false
    myRole.value = Number(res.data?.room?.my_role ?? -1)

    // 面板开着时不回填：loadInfo 每 60 秒轮询一次，回填会把用户正在编辑的内容冲掉；
    // 打开面板那一刻 openAdmin 已经按最新 room 回显过了
    if (!adminOpen.value) fillForm()
    noticeDraft.value = room.value.notice || ''

    // 游客：成员 / 申请面板都要登录才有意义，跳过这些必然 401 的请求
    if (isGuest.value) return

    // 非成员：不拉历史，页面显示「加入」引导
    if (!joined.value) return

    loadMembers()
    if (canModerate.value) loadRequests()
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 成员列表：自建群走后端，公共大厅用在线成员（socket 广播） */
async function loadMembers() {
  if (isPublicRoom.value) return
  try {
    const res = await listRoomMembers(roomId.value)
    if (res?.code !== 200) return
    members.value = (res.data?.data || []).map((item) => ({
      uid: Number(item.uid),
      name: item.user?.nickname || `用户 #${item.uid}`,
      avatar: item.user?.avatar || '',
      role: Number(item.role) || 0,
      roleName: item.role_name || '成员'
    }))
  } catch {
    /* 非成员 / 权限不足，request.js 已提示 */
  }
}

/** 待审的入群申请（群主 / 管理员） */
async function loadRequests() {
  if (isPublicRoom.value) return
  try {
    const res = await listRoomRequests(roomId.value)
    if (res?.code !== 200) return
    requests.value = (res.data?.data || []).map((item) => ({
      id: Number(item.id),
      uid: Number(item.uid),
      name: item.user?.nickname || `用户 #${item.uid}`,
      avatar: item.user?.avatar || '',
      message: item.message || '',
      createTime: Number(item.create_time) || 0
    }))
  } catch {
    /* 权限不足已提示 */
  }
}

/** 游客点了需要登录的动作：提示一句并带上回跳地址去登录页 */
function requireLogin(message = '请先登录') {
  if (userStore.isLogged) return false
  // 模板里若写成 @click="requireLogin"，传进来的是 PointerEvent 而不是文案，
  // 直接透给 toast 会弹成 [object PointerEvent]；这里兜一层，只认字符串。
  toast.warning(typeof message === 'string' ? message : '请先登录')
  router.push({ name: 'login', query: { redirect: route.fullPath } })
  return true
}

/** 加入群聊：自由加入直接进；需审批的提示等待 */
async function join() {
  if (requireLogin('请先登录后再加入群聊')) return

  try {
    const res = await joinRoom(roomId.value, joinMessage.value.trim())
    if (res?.code !== 200) return

    if (res.data?.joined) {
      joined.value = true
      joinPending.value = false
      toast.success('已加入群聊')
      await loadInfo()
      await loadHistory(0)
    } else {
      joinPending.value = true
      toast.info('申请已提交，等待群主审批')
    }
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 退群：群主不能退（后端会拦），前端提前提示 */
async function leave() {
  if (myRole.value === 2) {
    toast.warning('群主不能退出，请先解散群聊')
    return
  }
  if (!window.confirm('确定退出这个群聊吗？')) return

  try {
    const res = await leaveRoom(roomId.value)
    if (res?.code !== 200) return
    toast.success('已退出群聊')
    router.push('/chat')
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 解散群聊（群主 / 超级管理员） */
async function dissolve() {
  if (dissolving.value) return
  if (!window.confirm('解散后成员与申请都会清空，确定解散吗？')) return

  dissolving.value = true
  try {
    const res = await dissolveRoom(roomId.value)
    if (res?.code !== 200) return
    toast.success('群聊已解散')
    router.push('/chat')
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    dissolving.value = false
  }
}

/** 从公告弹窗直接进编辑：先关弹窗，再打开管理面板的公告 tab */
function editNotice() {
  noticeOpen.value = false
  openNoticeEditor()
}

/** 从右侧公告栏直接进入编辑（打开管理面板并定位到公告 tab） */
function openNoticeEditor() {
  noticeDraft.value = room.value.notice || ''
  adminTab.value = 'notice'
  adminOpen.value = true
}

/** 发布 / 清除群公告 */
async function saveNotice() {
  try {
    const res = await announceRoom(roomId.value, noticeDraft.value.trim())
    if (res?.code !== 200) return
    room.value = { ...room.value, ...(res.data || {}) }
    toast.success(noticeDraft.value.trim() ? '公告已发布' : '公告已清除')
    adminTab.value = 'notice'
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 审批入群申请 */
async function handleRequest(item, approve) {
  try {
    const res = await handleRoomRequest(roomId.value, item.id, approve)
    if (res?.code !== 200) return
    requests.value = requests.value.filter((row) => row.id !== item.id)
    toast.success(approve ? '已通过申请' : '已拒绝申请')
    if (approve) loadMembers()
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 邀请成员（按用户 id）：群主 / 管理员可用，「仅邀请」的群靠它进人 */
async function invite() {
  const uid = Number(inviteUid.value)
  if (!uid || inviting.value) {
    toast.warning('请输入要邀请的用户 ID')
    return
  }

  inviting.value = true
  try {
    const res = await addRoomMember(roomId.value, uid)
    if (res?.code !== 200) return
    inviteUid.value = ''
    toast.success('已邀请加入')
    loadMembers()
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    inviting.value = false
  }
}

/** 把邀请链接填进输入框，方便直接发到群里或私聊 */
function shareInvite(url) {
  if (!url) return
  draft.value = draft.value ? `${draft.value} ${url}` : url
  nextTick(() => inputRef.value?.focus?.())
  toast.success('邀请链接已填入输入框')
}

/** 移出成员 */
async function removeMember(item) {
  if (!window.confirm(`确定把「${item.name}」移出群聊吗？`)) return
  try {
    const res = await removeRoomMember(roomId.value, item.uid)
    if (res?.code !== 200) return
    members.value = members.value.filter((row) => row.uid !== item.uid)
    toast.success('已移出')
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 设置 / 撤销群管理员（只有群主能操作） */
async function toggleRole(item) {
  const next = item.role === 1 ? 0 : 1
  try {
    const res = await setRoomMemberRole(roomId.value, item.uid, next)
    if (res?.code !== 200) return
    item.role = next
    item.roleName = next === 1 ? '管理员' : '成员'
    toast.success(next === 1 ? '已设为管理员' : '已取消管理员')
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

/** 上报已读水位（列表页未读数用；失败静默） */
function reportRead() {
  if (isPublicRoom.value || !joined.value) return
  const last = maxSeq(messages.value)
  if (last > 0) markRoomRead(roomId.value, last).catch(() => {})
}

// 历史 / 补拉的代次：自动清空发生后 +1，让在途请求的旧结果直接作废，
// 否则「清空」与「正在翻页」撞在一起时，旧的一页会把已清空的消息又贴回来。
let historyEpoch = 0

async function loadHistory(beforeId = 0) {
  if (loading.value) return

  // 游客只能读公共大厅：自建群的历史要登录 + 成员身份，别发必然 401 的请求
  if (isGuest.value && !isPublicRoom.value) {
    messages.value = []
    return
  }

  loading.value = true

  const epoch = historyEpoch

  try {
    const res = await listRoomMessages(
      roomId.value,
      beforeId ? { before_id: beforeId } : {}
    )
    if (res?.code !== 200) return
    if (epoch !== historyEpoch) return

    const list = res.data?.data || []
    if (beforeId) {
      const seen = new Set(messages.value.map((item) => Number(item.id)))
      messages.value = [...list.filter((item) => !seen.has(Number(item.id))), ...messages.value]
      const el = bodyRef.value
      if (el) el.scrollTop = 0
    } else {
      messages.value = list
      scrollToBottom()
      // 拉到最新一页就算读过了：列表页的未读数据此归零
      reportRead()
    }

    hasMore.value = !!res.data?.has_more
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    loading.value = false
  }
}

/** 断线重连补拉：按本地最大 seq 往后翻，最多 5 页 */
async function syncMessages() {
  let cursor = maxSeq(messages.value)
  const epoch = historyEpoch

  for (let page = 0; page < 5; page += 1) {
    const res = await listRoomMessages(roomId.value, { after_seq: cursor, limit: 50 })
    if (res?.code !== 200) return
    if (epoch !== historyEpoch) return

    const list = res.data?.data || []
    if (!list.length) return

    mergeMessages(list)
    cursor = Number(res.data?.last_seq) || cursor
    if (!res.data?.has_more) break
  }

  reportRead()
}

// ---------- @ 提及 ----------

/** 输入时：上报「正在输入」，并判断是否要弹出 @ 候选 */
function onInput() {
  if (muted.value) return

  chatStore.sendRoomTyping(roomId.value)

  // 输入框是 contenteditable，拿光标前的纯文本而不是 selectionStart
  const before = inputRef.value?.textBeforeCaret?.() || ''
  const match = before.match(/@([^\s@]*)$/)

  mentionOpen.value = Boolean(match)
  mentionKeyword.value = match ? match[1] : ''
}

/** 工具栏的 @ 按钮：不输入也能选人 */
function openMention() {
  if (muted.value) return
  mentionKeyword.value = ''
  mentionOpen.value = true
  nextTick(() => inputRef.value?.focus())
}

/** 选中某人：把光标处的 @关键词 替换成「@昵称 」 */
function pickMention(member) {
  const uid = Number(member?.uid) || 0
  if (uid <= 0) return

  const editor = inputRef.value
  if (!editor) return

  const before = editor.textBeforeCaret() || ''
  const at = before.lastIndexOf('@')
  // 只有「光标前紧跟着 @关键词」才替换这一段；从按钮打开的直接插在光标处
  const typed = at >= 0 && !/\s/.test(before.slice(at + 1))
  const inserted = `@${member.name} `

  if (typed) {
    editor.replaceBeforeCaret(before.length - at, inserted)
  } else {
    editor.insertText(inserted)
  }

  if (!mentionUids.value.includes(uid)) mentionUids.value.push(uid)

  mentionOpen.value = false
  mentionKeyword.value = ''

  nextTick(() => editor.focus())
}

/** 成员面板点「@ TA」：先关面板再插入，否则面板会盖住输入框 */
function mentionMember(member) {
  memberOpen.value = false
  pickMention(member)
}

/**
 * 输入框键盘处理
 *
 * @ 面板打开时，回车 / Tab 先选中第一个候选；否则回车发送。
 * e.isComposing 是中文输入法正在组字，这时回车是「选字」，不能当成发送。
 */
function onInputKeydown(e) {
  if (mentionOpen.value) {
    if ((e.key === 'Enter' || e.key === 'Tab') && mentionCandidates.value.length) {
      e.preventDefault()
      pickMention(mentionCandidates.value[0])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      mentionOpen.value = false
      return
    }
  }

  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    send()
  }
}

// ---------- 发送 ----------

async function postMessage(content, msgType = 'text', quoteId = 0) {
  const res = await sendRoomMessage(roomId.value, {
    content,
    msg_type: msgType,
    // 引用回复：只传被引用消息 id，摘要由服务端回查后落快照
    quote_id: Number(quoteId) || 0,
    // @提及只对文本生效：图片消息里没有 @昵称 可以核对
    mention_uids: msgType === 'text' ? [...mentionUids.value] : [],
    // 幂等键：连点 / 重试都只会落一条
    client_msg_id: `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  })

  if (res?.code !== 200) return null

  // 服务端已经广播了同一条，这里也合并一次：谁先到都不重复
  mergeMessages([res.data])
  scrollToBottom()

  return res.data
}

async function send() {
  const content = draft.value.trim()
  if (!content || sending.value || muted.value) return

  sending.value = true
  try {
    const message = await postMessage(content, 'text', quoteDraft.value?.id)
    if (message) {
      draft.value = ''
      clearQuote()
      // @ 名单属于这一条消息：发出去就清掉，否则下一条会被误提醒
      mentionUids.value = []
      mentionOpen.value = false
    }
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    sending.value = false
  }
}

async function revoke(msg) {
  if (!canRevoke(msg)) return

  try {
    const res = await revokeRoomMessage(roomId.value, msg.id)
    if (res?.code !== 200) return
    applyRevoke(res.data)
    toast.success('已撤回')
  } catch {
    /* 超过 2 分钟等业务错误已由 request.js 提示 */
  }
}

function applyRevoke(message) {
  if (!message?.id) return

  // 正在引用的消息被撤回：引用态一起清掉，避免发出去必然失败的引用
  if (Number(quoteDraft.value?.id) === Number(message.id)) clearQuote()

  const index = messages.value.findIndex((item) => Number(item.id) === Number(message.id))
  if (index < 0) return
  messages.value[index] = { ...messages.value[index], ...message, is_revoked: 1, content: '' }
}

/**
 * 表情面板回填
 *
 * 表情在输入框里要显示成图片而不是 [emoji:url] 短代码，
 * 所以走编辑器的 insertEmoji；颜文字之类的纯文本仍按普通文字插入。
 */
function insertEmoji(text) {
  const editor = inputRef.value
  const value = String(text || '')

  if (!editor) {
    draft.value += value
    return
  }

  if (value.startsWith('[emoji:')) {
    editor.insertEmoji(value.slice(7, -1))
    return
  }
  editor.insertText(value)
}

function pickImage() {
  if (uploading.value || muted.value) return
  fileRef.value?.click()
}

const MAX_IMAGE_SIZE = 5 * 1024 * 1024

async function uploadAndSendImages(files) {
  if (muted.value) return

  const list = Array.from(files || []).filter((file) => file && String(file.type).startsWith('image/'))
  if (!list.length) {
    toast.warning('请选择图片文件')
    return
  }

  const oversize = list.find((file) => file.size > MAX_IMAGE_SIZE)
  if (oversize) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploading.value = true
  try {
    const form_data = new FormData()
    list.forEach((file) => form_data.append('files', file))

    const res = await uploadAttachments(form_data)
    const urls = (res?.data?.results || [])
      .filter((row) => row.status !== 'fail')
      .map((row) => row.full_url || row.url || '')
      .filter(Boolean)

    if (!urls.length) {
      toast.warning('图片上传失败，请稍后重试')
      return
    }

    for (const url of urls) {
      // 引用只挂在第一张上：成功发出后即消费掉
      const message = await postMessage(url, 'image', quoteDraft.value?.id)
      if (!message) break
      clearQuote()
    }
  } catch {
    toast.error('图片发送失败，请稍后重试')
  } finally {
    uploading.value = false
  }
}

function onFilesPicked(event) {
  const input = event.target
  // 先拷贝再清空：input.value = '' 会把 input.files 一起清掉
  const files = Array.from(input.files || [])
  input.value = ''
  uploadAndSendImages(files)
}

// 粘贴图片由 ChatEmojiInput 统一识别后抛 @files，这里不再自己监听 paste

// ---------- 管理 ----------

function openAdmin() {
  fillForm()
  adminOpen.value = true
  // 群管理员没有「资料 / 公告」权限，直接落到禁言名单，避免打开是空面板
  adminTab.value = canManage.value ? 'profile' : 'mutes'
  if (!canManage.value) loadMutes()
}

function switchMutes() {
  adminTab.value = 'mutes'
  loadMutes()
}

async function loadMutes() {
  try {
    const res = await listRoomMutes(roomId.value)
    if (res?.code === 200) mutes.value = res.data?.data || []
  } catch {
    /* 权限不足等错误已由 request.js 提示 */
  }
}

function switchClear() {
  adminTab.value = 'clear'
  loadAutoClear()
}

async function loadAutoClear() {
  try {
    const res = await getRoomAutoClear()
    if (res?.code !== 200) return

    clearInfo.value = res.data || clearInfo.value
    clearDays.value = Number(res.data?.days || 3)
  } catch {
    /* 权限不足等错误已由 request.js 提示 */
  }
}

async function saveAutoClear() {
  if (clearSaving.value) return

  clearSaving.value = true
  try {
    const res = await saveRoomAutoClear(clearDays.value)
    if (res?.code !== 200) return

    clearInfo.value = res.data || clearInfo.value
    clearDays.value = Number(res.data?.days || clearDays.value)
    toast.success('已保存，下次清空按新间隔计算')
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    clearSaving.value = false
  }
}

async function saveRoom() {
  if (saving.value) return

  const name = form.value.name.trim()
  if (!name) {
    toast.warning('群聊名称不能为空')
    return
  }

  saving.value = true
  try {
    const res = await updateRoom(roomId.value, {
      name,
      description: form.value.description.trim(),
      avatar: form.value.avatar.trim(),
      // 大厅的入群方式固定为自由加入，后端会忽略这个字段
      join_type: Number(form.value.join_type ?? room.value.join_type ?? 0)
    })
    if (res?.code !== 200) return

    room.value = { ...room.value, ...(res.data || {}) }
    toast.success('已保存')
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    saving.value = false
  }
}

function pickAvatar() {
  avatarInput.value?.click()
}

async function onAvatarPicked(event) {
  const input = event.target
  const file = Array.from(input.files || [])[0]
  input.value = ''
  if (!file) return

  if (file.size > MAX_IMAGE_SIZE) {
    toast.warning('图片不能超过 5MB')
    return
  }

  avatarUploading.value = true
  try {
    const payload = new FormData()
    payload.append('files', file)

    const res = await uploadAttachments(payload)
    const row = (res?.data?.results || []).find((item) => item.status !== 'fail')
    const url = row?.full_url || row?.url || ''
    if (!url) {
      toast.warning('上传失败，请稍后重试')
      return
    }
    form.value.avatar = url
  } catch {
    toast.error('上传失败，请稍后重试')
  } finally {
    avatarUploading.value = false
  }
}

/** 群管理点消息头像：打开禁言 / 移出操作浮层 */
function onAvatarClick(msg) {
  if (!canModerate.value || isMine(msg) || msg.is_revoked === 1) return
  userPanel.value = {
    uid: Number(msg.uid),
    nickname: nameOf(msg),
    avatar: msg.user?.avatar || ''
  }
}

async function mute(uid, minutes) {
  try {
    const res = await muteRoomUser(roomId.value, uid, minutes)
    if (res?.code !== 200) return

    toast.success(minutes > 0 ? `已禁言 ${minutes} 分钟` : '已永久禁言')
    userPanel.value = null

    if (adminOpen.value && adminTab.value === 'mutes') loadMutes()
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

async function unmute(uid) {
  try {
    const res = await unmuteRoomUser(roomId.value, uid)
    if (res?.code !== 200) return

    mutes.value = mutes.value.filter((item) => Number(item.uid) !== Number(uid))
    toast.success('已解除禁言')
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

// ---------- 实时 ----------

let unsubscribeEvent = null

/** 记一条「某人正在输入」，4 秒无新上报自动移除 */
function noteTyping(uid) {
  const key = Number(uid) || 0
  if (key <= 0 || key === me.value) return

  const name = onlineMembers.value.find((item) => Number(item.uid) === key)?.name || `用户 #${key}`
  typers.value.set(key, { name })

  clearTimeout(typingTimers.get(key))
  typingTimers.set(
    key,
    setTimeout(() => {
      typers.value.delete(key)
      typingTimers.delete(key)
    }, 4000)
  )
}

function onRoomEvent(payload = {}) {
  // 断线重连：按本地最大 seq 补齐断网期间的消息，顺便刷新禁言状态
  if (payload.type === 'reconnect') {
    syncMessages()
    loadInfo()
    return
  }

  // 有人正在输入（群聊广播）：前端自己按 uid 过滤掉自己
  if (payload.type === 'typing') {
    noteTyping(payload.fromUid)
    return
  }

  // 有人 @ 我：在房间里就直接提示并把角标清掉（已经看到了）
  if (payload.type === 'mention') {
    if (Number(payload.item?.from_uid) === me.value) return
    // 别的群 @ 我：不在这里提示（列表页负责），避免进 A 群弹出 B 群的提醒
    if (Number(payload.item?.room_id || 1) !== roomId.value) return
    chatStore.clearRoomMentions()
    toast.info('有人在群聊提到了你')
    return
  }

  const event = payload.event
  const type = event?.type
  const data = event?.data
  if (!type || !data) return

  // 多群：只处理当前房间的事件（大厅事件可能不带 room_id，按 1 处理）
  const eventRoomId = Number(data.room_id || data.id || 1)
  if (eventRoomId !== roomId.value) return

  // 有人在当前群打字（服务端按 room_id 分发，前端再按 uid 过滤自己）
  if (type === 'room_typing') {
    noteTyping(Number(data.uid))
    return
  }

  if (type === 'room') {
    if (!joined.value) return
    if (mergeMessages([data])) {
      scrollToBottom()
      reportRead()
    }
    // 消息已经到了，对应的「正在输入」就该收掉
    const uid = Number(data.uid) || 0
    if (uid > 0 && typers.value.has(uid)) {
      typers.value.delete(uid)
      clearTimeout(typingTimers.get(uid))
      typingTimers.delete(uid)
    }
    return
  }

  if (type === 'room_revoke') {
    applyRevoke(data)
    return
  }

  // 本人删除：服务端只私推给同账号的其它在线设备，本机也要把气泡移除
  if (type === 'room_delete') {
    const id = Number(data.id || 0)
    if (!id) return
    messages.value = messages.value.filter((item) => Number(item.id) !== id)
    if (Number(quoteDraft.value?.id) === id) clearQuote()
    return
  }

  if (type === 'room_update') {
    room.value = { ...room.value, ...data }
    return
  }

  // 群公告：顶部公告条就地刷新
  if (type === 'room_announce') {
    room.value = { ...room.value, ...data }
    return
  }

  // 群成员变化：刷新成员 / 申请面板（群主与管理员才有的列表）
  if (type === 'room_member_join' || type === 'room_member_leave' || type === 'room_member_remove') {
    loadMembers()
    // 自己被移出：立刻退出页面
    if (type === 'room_member_remove' && Number(data.uid) === me.value) {
      toast.warning('你已被移出该群聊')
      router.push('/chat')
    }
    return
  }

  if (type === 'room_member_role') {
    loadMembers()
    return
  }

  // 新的入群申请：提醒群主 / 管理员处理
  if (type === 'room_join_request') {
    if (canModerate.value) {
      loadRequests()
      toast.info('有新的入群申请')
    }
    return
  }

  // 群被解散：成员页面统一退回列表
  if (type === 'room_dissolve') {
    toast.warning('该群聊已解散')
    router.push('/chat')
    return
  }

  // 服务端按间隔自动清空了全部消息：把本地时间线一起清掉（游标随 messages 归零）
  if (type === 'room_clear') {
    historyEpoch += 1
    messages.value = []
    hasMore.value = false
    clearQuote()
    loadHistory(0)
    toast.info('群聊记录已自动清空')
    return
  }

  if (type === 'room_mute' && Number(data.uid) === me.value) {
    muted.value = true
    mutedUntil.value = Number(data.until || 0)
    return
  }

  if (type === 'room_unmute' && Number(data.uid) === me.value) {
    muted.value = false
    mutedUntil.value = 0
  }
}

// 禁言是「到点自动失效」的：页面开着时定期刷新一次状态，
// 到期后输入框能自己恢复，不用手动刷新页面
let infoTimer = null

onMounted(async () => {
  chatStore.start()
  // 进了房间就说明已经看到 @：把私信列表入口上的红点清掉
  chatStore.clearRoomMentions()
  // 房间里被 @ 由本页 toast 提示，不再重复弹右下角浮层
  chatStore.setViewingRoom(true)
  // 先拿房间与成员身份：非成员不拉历史（后端会 403，前端也少一次无谓请求）
  await loadInfo()
  if (joined.value) await loadHistory(0)
  unsubscribeEvent = chatStore.onEvent(onRoomEvent)
  // 后台标签页不刷新：这个轮询只是为了「禁言到点自动解除」，
  // 页面不可见时刷新没有意义（用户看不到），还会占一个连接与一次 QPS 计数
  infoTimer = setInterval(() => {
    if (typeof document !== 'undefined' && document.hidden) return
    loadInfo()
  }, 60 * 1000)

  // 菜单打开时按 Esc / 滚动页面都要收起，避免浮在错误的位置
  document.addEventListener('keydown', onMenuKeydown)
  window.addEventListener('scroll', onMenuScroll, true)
})

onBeforeUnmount(() => {
  if (unsubscribeEvent) unsubscribeEvent()
  chatStore.setViewingRoom(false)
  clearInterval(infoTimer)
  typingTimers.forEach((timer) => clearTimeout(timer))
  typingTimers.clear()
  document.removeEventListener('keydown', onMenuKeydown)
  window.removeEventListener('scroll', onMenuScroll, true)
})
</script>

<style scoped>
/* 整屏页面：与私信页同一套骨架，但群聊内容居中收窄，长行更好读 */
/* 右栏：由 /chat 布局撑开，自身不再占满视口（否则会和左栏叠在一起）
   配色仿 QQ NT：浅灰蓝聊天底 + 白色气泡，局部变量，不影响站点其它页面 */
.room-chat {
  /* 映射站点语义 token：跟随主题与暗色模式，不写死颜色 */
  --chat-blue: var(--primary);
  --chat-panel: var(--bg);
  --chat-line: var(--border-soft);
  --chat-weak: var(--text-light);

  display: grid;
  grid-template-columns: minmax(0, 1fr) 264px;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  grid-template-areas:
    "head head"
    "notice notice"
    "body aside"
    "composer aside";
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  color: var(--text);
  background: var(--chat-panel);
}
.room-head {
  grid-area: head;
}
.room-notice {
  grid-area: notice;
}
.room-body {
  grid-area: body;
}
.room-composer {
  grid-area: composer;
}
.room-aside {
  grid-area: aside;
}

/* 桌面端左栏已有列表，返回按钮只在窄屏出现 */
.room-back {
  display: none;
}

/* ==================== 顶部 ==================== */
.room-head {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
  padding: 12px 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--chat-line);
}
.room-icon-btn {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  text-decoration: none;
  transition: color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.room-icon-btn:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.room-avatar {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-size: 18px;
  color: var(--primary);
  background: var(--accent-wash);
  border: 1px solid var(--primary-soft);
  border-radius: 50%;
}
.room-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.room-avatar-sm {
  width: 32px;
  height: 32px;
  font-size: 14px;
}
.room-title {
  flex: 1;
  min-width: 0;
}
.room-title h1 {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-title p {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--text-light);
  overflow: hidden;
  white-space: nowrap;
}
.room-desc {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
/* 在线人数：点开成员面板，也是 @ 选人的入口之一 */
.room-online {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 8px;
  font-family: inherit;
  font-size: 11px;
  color: var(--chat-blue);
  background: var(--accent-wash);
  border: 0;
  border-radius: 999px;
  cursor: pointer;
}
.room-online:hover {
  background: var(--accent-soft);
}
.room-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}
.room-mute-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  font-size: 11px;
  color: var(--danger);
  background: var(--accent-wash);
  border-radius: 999px;
}

/* 通用按钮 */
.room-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  font-family: inherit;
  font-size: 12.5px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.room-btn:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary-soft);
}
.room-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.room-btn-primary {
  color: #fff;
  background: var(--primary);
  border-color: var(--primary);
}
.room-btn-primary:hover:not(:disabled) {
  color: #fff;
  filter: brightness(1.06);
}
.room-btn-danger {
  color: var(--danger);
  border-color: var(--danger);
}
.room-btn-sm {
  padding: 3px 9px;
  font-size: 11.5px;
}

/* ==================== 消息区 ==================== */
.room-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 18px max(18px, calc((100% - 900px) / 2)) 10px;
  background: var(--chat-panel);
  scrollbar-width: thin;
}
.room-tip {
  padding: 32px 0;
  font-size: 12px;
  text-align: center;
  color: var(--text-light);
}
.room-more {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 0 auto 16px;
  padding: 5px 14px;
  font-family: inherit;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  cursor: pointer;
}
.room-more:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary-soft);
}
.room-day {
  display: flex;
  justify-content: center;
  margin: 18px 0 14px;
  font-size: 11px;
  color: var(--text-light);
}
.room-day span {
  padding: 0;
  background: transparent;
}

.room-msg {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 16px;
}
.room-msg.is-mine {
  flex-direction: row-reverse;
}
.room-msg-avatar {
  flex-shrink: 0;
  width: 34px;
  height: 34px;
  padding: 0;
  overflow: hidden;
  background: var(--bg-muted);
  border: 0;
  border-radius: 50%;
  cursor: pointer;
}
.room-msg-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.room-msg-main {
  display: flex;
  flex-direction: column;
  max-width: 70%;
  min-width: 0;
}
.room-msg.is-mine .room-msg-main {
  align-items: flex-end;
}
.room-msg-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  font-size: 11px;
  color: var(--text-light);
}
.room-msg.is-mine .room-msg-meta {
  flex-direction: row-reverse;
}
.room-msg-name {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-soft);
  max-width: 160px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-msg-revoke {
  padding: 0;
  font-family: inherit;
  font-size: 11px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.room-msg:hover .room-msg-revoke {
  opacity: 1;
}
.room-msg-revoke:hover {
  color: var(--danger);
}

/* QQ NT：别人白气泡（左上角小圆角），自己蓝气泡（右上角小圆角） */
.room-bubble {
  padding: 8px 12px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  background: var(--bg-card);
  border: 0;
  border-radius: 8px;
  border-top-left-radius: 2px;
  box-shadow: var(--shadow-sm);
  word-break: break-word;
  white-space: pre-wrap;
}
.room-msg.is-mine .room-bubble {
  color: #fff;
  background: var(--chat-blue);
  border-radius: 8px;
  border-top-right-radius: 2px;
  box-shadow: none;
}
.room-bubble.is-revoked {
  font-size: 12.5px;
  font-style: normal;
  color: var(--text-light);
  background: var(--bg-muted);
  border-color: transparent;
  box-shadow: none;
}
.room-msg.is-mine .room-bubble.is-revoked {
  color: var(--text-light);
  background: var(--bg-muted);
}
.room-image {
  display: block;
  max-width: 240px;
  max-height: 240px;
  border-radius: var(--radius-sm);
  cursor: zoom-in;
}
.room-text {
  display: inline-block;
  vertical-align: top;
  white-space: pre-wrap;
}

/* 引用块：气泡内的浅色摘要，左侧竖线区分「这是被引用的内容」 */
.room-quote {
  display: flex;
  flex-direction: column;
  gap: 1px;
  margin-bottom: 6px;
  padding: 5px 9px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-xs);
  white-space: normal;
}
.room-quote-name {
  font-weight: 600;
  color: var(--text-light);
}
.room-quote-text {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  line-clamp: 2;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-word;
}
.room-msg.is-mine .room-quote {
  color: rgba(255, 255, 255, 0.82);
  background: rgba(255, 255, 255, 0.14);
  border-left-color: rgba(255, 255, 255, 0.65);
}
.room-msg.is-mine .room-quote-name {
  color: #fff;
}

/* ==================== 输入区 ==================== */
.room-composer {
  flex-shrink: 0;
  padding: 10px max(18px, calc((100% - 900px) / 2)) calc(12px + env(safe-area-inset-bottom));
  background: var(--bg-card);
  border-top: 1px solid var(--chat-line);
}
.room-composer-box {
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 6px;
  padding: 6px 6px 6px 8px;
  background: var(--bg-card);
  border: 1px solid var(--chat-line);
  border-radius: 8px;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.room-composer-box:focus-within {
  border-color: var(--chat-blue);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.room-tools {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-bottom: 2px;
}
.room-emoji-wrap {
  position: relative;
}
.room-emoji-wrap :deep(.emoji-picker-panel) {
  top: auto;
  bottom: calc(100% + 8px);
  z-index: 3200;
}
/*
 * 手机端：表情按钮本身只有 30px 宽，若面板以 .room-emoji-wrap 为定位参照，
 * width:100% 就只有按钮那么窄 —— 面板会被挤成贴边的一条竖条，表情一行一个。
 * 这里让 wrap 退出定位，面板改为相对 .room-composer-box（已 position: relative）铺满输入框宽度。
 */
@media (max-width: 768px) {
  .room-emoji-wrap {
    position: static;
  }
  .room-emoji-wrap :deep(.emoji-picker-panel) {
    left: 0;
    right: 0;
    width: auto;
    min-width: 0;
    max-width: none;
  }
}
.room-input {
  flex: 1;
  min-width: 0;
  min-height: 36px;
  max-height: 132px;
  padding: 8px 2px;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: none;
  resize: none;
}
.room-input::placeholder {
  color: var(--text-light);
}
.room-send {
  min-width: 64px;
  height: 34px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 18px;
  font-size: 13px;
  font-family: inherit;
  color: #fff;
  background: var(--chat-blue);
  border: 0;
  border-radius: 6px;
  cursor: pointer;
  transition: opacity var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease);
}
.room-send:hover:not(:disabled) {
  background: var(--primary-deep);
}
.room-send:hover:not(:disabled) {
  opacity: 0.9;
}
.room-send:active:not(:disabled) {
  transform: scale(0.97);
}
.room-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.room-file {
  position: fixed;
  left: -9999px;
  top: -9999px;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.room-blocked {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0;
  padding: 10px 0;
  font-size: 12px;
  color: var(--danger);
  background: var(--accent-wash);
  border-radius: var(--radius);
}
/* 有人正在输入：一行灰字，出现 / 消失都不推挤输入框太多 */
.room-typing {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 0 0 6px;
  padding-left: 4px;
  font-size: 12px;
  color: var(--text-light);
}

/* @ 选人面板：挂在输入框上方，不遮挡输入内容 */
.room-mention {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(100% + 6px);
  max-height: 220px;
  overflow-y: auto;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  z-index: 1600;
}
.room-mention-title {
  margin: 2px 6px 6px;
  font-size: 11px;
  color: var(--text-light);
}
.room-mention-item {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 7px 8px;
  font-family: inherit;
  font-size: 13px;
  text-align: left;
  color: var(--text);
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.room-mention-item:hover {
  background: var(--bg-muted);
}
.room-mention-item i {
  color: var(--primary);
}

/* 在线成员面板 */
/* 双类选择器：.room-modal-card 的 padding 定义在文件末尾，单类规则会被它盖掉 */
.room-modal-card.room-member-card {
  display: flex;
  flex-direction: column;
  width: min(340px, 100%);
  max-height: min(70vh, 480px);
  padding: 0;
  overflow: hidden;
}
.room-member-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px 10px 14px;
  font-size: 13px;
  font-weight: 600;
  border-bottom: 1px solid var(--border-soft);
}
.room-member-head .room-icon-btn {
  width: 26px;
  height: 26px;
  font-size: 13px;
}
.room-member-list {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 4px 6px 6px;
  overflow-y: auto;
  list-style: none;
}
.room-member-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border-radius: var(--radius);
}
.room-member-row:hover {
  background: var(--bg-muted);
}
.room-member-avatar {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--accent-wash);
  border-radius: 50%;
}
.room-member-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-member-self {
  margin-left: 6px;
  font-style: normal;
  font-size: 10px;
  color: var(--primary);
}
.room-member-card .room-tip {
  padding: 8px 12px 10px;
  font-size: 12px;
}

/* ==================== 多群：公告 / 徽标 / 加入引导 ==================== */
.room-badge {
  display: inline-block;
  margin-left: 6px;
  padding: 0 5px;
  font-size: 10px;
  font-weight: 500;
  line-height: 1.6;
  color: var(--chat-blue);
  vertical-align: middle;
  background: var(--accent-soft);
  border-radius: 4px;
}
.room-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 18px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-soft);
  cursor: pointer;
  background: var(--accent-wash);
  border-bottom: 1px solid var(--border-soft);
  transition: background 0.12s;
}
/* 只是一个「预览条」：点击后到弹窗里看全文，所以给个可点的反馈 */
.room-notice:hover,
.room-notice:focus-visible {
  background: var(--accent-soft);
}
.room-notice i {
  margin-top: 2px;
  color: var(--primary);
}
.room-notice span {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-notice .room-notice-more {
  flex-shrink: 0;
  margin-top: 3px;
  font-size: 10px;
  color: var(--chat-text-weak);
}

/* ==================== 群公告弹窗 ==================== */
/* 双类选择器：.room-modal-card 的 padding 定义在文件末尾，单类规则会被它盖掉 */
.room-modal-card.room-notice-card {
  display: flex;
  flex-direction: column;
  width: min(420px, 100%);
  max-height: min(70vh, 520px);
  padding: 0;
  overflow: hidden;
}
.room-notice-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--text);
  border-bottom: 1px solid var(--border-soft);
}
.room-notice-head span {
  display: flex;
  align-items: center;
  gap: 6px;
}
.room-notice-head i {
  font-size: 12px;
  color: var(--primary);
}
.room-notice-body {
  flex: 1;
  min-height: 0;
  padding: 14px 16px 18px;
  overflow-y: auto;
  font-size: 13px;
  line-height: 1.8;
  color: var(--text-soft);
  /* 公告是用户手写的多行文本：保留换行，超长链接也要能折行 */
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.room-notice-foot {
  display: flex;
  justify-content: flex-end;
  padding: 10px 14px 12px;
  border-top: 1px solid var(--border-soft);
}
.room-join {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 14px 18px;
  font-size: 13px;
  color: var(--text-soft);
}
.room-join p {
  flex: 1 1 100%;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}
.room-join input {
  flex: 1;
  min-width: 160px;
  padding: 8px 10px;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  outline: none;
}
.room-join input:focus {
  border-color: var(--primary);
}
.room-revoked {
  margin: 12px auto;
  font-size: 12px;
  color: var(--text-light);
  text-align: center;
}

/* ==================== 右侧栏：群公告 / 群成员 ==================== */
.room-aside {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 14px 14px 10px;
  overflow: hidden;
  background: var(--bg-card);
  border-left: 1px solid var(--chat-line);
}
.room-aside-block {
  padding-bottom: 12px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--chat-line);
}
.room-aside-block.is-grow {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  margin-bottom: 0;
  padding-bottom: 0;
  border-bottom: 0;
}
.room-aside-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text);
}
.room-aside-link {
  padding: 0;
  font-family: inherit;
  font-size: 11.5px;
  color: var(--chat-blue);
  background: transparent;
  border: 0;
  cursor: pointer;
}
.room-aside-text {
  max-height: 132px;
  margin: 0;
  overflow-y: auto;
  font-size: 12px;
  line-height: 1.75;
  color: var(--text-soft);
  white-space: pre-wrap;
  word-break: break-word;
}
/* 有公告时右侧这块也是「点开看全文」的入口（与顶部公告条同一个弹窗） */
.room-aside-text.is-clickable {
  cursor: pointer;
}
.room-aside-text.is-clickable:hover {
  color: var(--text);
}
.room-aside-members {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
  scrollbar-width: thin;
}
.room-aside-member {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 4px;
  border-radius: 6px;
}
.room-aside-member:hover {
  background: var(--chat-panel);
}
.room-aside-avatar {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 50%;
}
.room-aside-name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-aside-empty {
  padding: 18px 0;
  font-size: 11.5px;
  color: var(--chat-weak);
  text-align: center;
}
@media (max-width: 1180px) {
  .room-chat {
    grid-template-columns: minmax(0, 1fr);
    grid-template-areas:
      "head"
      "notice"
      "body"
      "composer";
  }
  .room-aside {
    display: none;
  }
}
.room-role-tag {
  margin-left: 6px;
  padding: 0 5px;
  font-style: normal;
  font-size: 10px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: 4px;
}
.room-admin-danger {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-soft);
}
.room-admin-danger span {
  font-size: 11px;
  line-height: 1.5;
  color: var(--text-soft);
}
.room-mute-name em {
  display: block;
  margin-top: 2px;
  font-style: normal;
  font-size: 11px;
  color: var(--text-soft);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-invite {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-bottom: 10px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--border-soft);
}
.room-invite-row {
  display: flex;
  gap: 8px;
}
.room-invite input {
  flex: 1;
  min-width: 0;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  outline: none;
}
.room-invite input:focus {
  border-color: var(--primary);
}
.room-invite-link {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  padding: 7px 0;
  font-size: 12.5px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: var(--radius);
}
.room-invite-link:hover {
  background: var(--accent-ring);
}

/* ==================== 管理抽屉 ==================== */
.room-admin {
  position: fixed;
  inset: 0;
  z-index: 3400;
}
.room-admin-mask {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.28);
}
.room-admin-panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 360px;
  max-width: 92vw;
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-left: 1px solid var(--border);
  box-shadow: var(--shadow-md);
}
.room-admin-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 1px solid var(--border-soft);
}
.room-admin-tabs {
  display: flex;
  gap: 4px;
  padding: 10px 16px 0;
  border-bottom: 1px solid var(--border-soft);
}
.room-admin-tab {
  padding: 7px 10px;
  font-family: inherit;
  font-size: 12.5px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  border-bottom: 2px solid transparent;
  cursor: pointer;
}
.room-admin-tab.is-active {
  color: var(--primary);
  border-bottom-color: var(--primary);
}
.room-admin-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 16px calc(16px + env(safe-area-inset-bottom));
}
.room-field {
  display: block;
  margin-bottom: 14px;
}
.room-field > span {
  display: block;
  margin-bottom: 6px;
  font-size: 12px;
  color: var(--text-soft);
}
.room-field input,
.room-field textarea {
  width: 100%;
  padding: 8px 10px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-sm);
  outline: none;
  resize: vertical;
}
.room-field input:focus,
.room-field textarea:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.room-avatar-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.room-avatar-row input {
  flex: 1;
  min-width: 0;
}
.room-admin-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}
.room-mute-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-soft);
}
.room-mute-avatar {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  object-fit: cover;
  background: var(--bg-muted);
  border-radius: 50%;
}
.room-mute-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.room-mute-row em {
  font-size: 11px;
  font-style: normal;
  color: var(--text-light);
}

/* ==================== 用户操作弹窗 ==================== */
.room-modal {
  position: fixed;
  inset: 0;
  z-index: 3500;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}
.room-modal-mask {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.3);
}
.room-modal-card {
  position: relative;
  width: 300px;
  max-width: 100%;
  padding: 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.room-modal-user {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 600;
}
.room-modal-user img {
  width: 36px;
  height: 36px;
  object-fit: cover;
  background: var(--bg-muted);
  border-radius: 50%;
}
.room-modal-tip {
  margin: 10px 0 14px;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
}
.room-modal-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.is-spin {
  animation: room-spin 0.9s linear infinite;
}
@keyframes room-spin {
  to {
    transform: rotate(360deg);
  }
}

/* 窄屏 */
@media (max-width: 860px) {
  .room-back {
    display: inline-flex;
  }
  .room-head {
    padding: 10px 12px;
  }
  /* 手机端只收起简介：在线人数是成员面板 / @ 选人的入口，不能一起藏掉 */
  .room-desc {
    display: none;
  }
  .room-title p {
    margin-top: 3px;
  }
  .room-body {
    padding: 14px 12px 6px;
  }
  .room-composer {
    padding: 8px 12px calc(10px + env(safe-area-inset-bottom));
  }
  .room-msg-main {
    max-width: 80%;
  }
  .room-msg-revoke {
    opacity: 1;
  }
  .room-image {
    max-width: 200px;
    max-height: 200px;
  }
  .room-btn {
    padding: 5px 9px;
  }
  .room-admin-panel {
    width: 100%;
    max-width: none;
  }
}

/* ==================== 引用态（输入框上方） ==================== */
.room-quote-draft {
  flex-basis: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  margin-bottom: 2px;
  background: var(--bg-muted);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-xs);
}
.room-quote-draft-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 12px;
  line-height: 1.5;
}

/* ==================== 消息右键菜单 ==================== */
.room-menu-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
}
.room-menu {
  position: fixed;
  z-index: 61;
  min-width: 148px;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.room-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--text-soft);
  text-align: left;
  white-space: nowrap;
  background: none;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.room-menu-item:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.room-menu-item i {
  font-size: 14px;
  opacity: 0.9;
}
.room-menu-item.is-danger {
  color: var(--danger);
}
.room-menu-item.is-danger:hover {
  color: var(--danger);
  background: var(--accent-soft);
}
</style>
