<template>
  <div class="author-page">
    <!-- ===== 资料卡：封面横幅 + 头像压边 + 身份 + 数据 ===== -->
    <section class="hero" :style="coverStyle">
      <div class="hero-cover">
        <!-- 更换背景：仅本人可见，点进「我的资料 → 主页封面」上传图片或选颜色 -->
        <RouterLink v-if="isSelf" to="/user/profile" class="hero-cover-edit">
          <i class="bi bi-image" aria-hidden="true" /> 更换背景
        </RouterLink>
      </div>

      <div class="hero-body">
        <div class="hero-head">
          <div class="hero-avatar">
            <AvatarFrame
              :src="profile.avatar"
              :name="profile.nickname"
              :frame="frame"
              :animated="frameRender.animated"
              :fallback="defaultAvatar"
              :size="avatarSize"
              :frame-hole="frameRender.hole"
              :frame-offset-x="frameRender.offsetX"
              :frame-offset-y="frameRender.offsetY"
              :verified="cert"
              alt="头像"
            />
          </div>

          <div class="hero-ident">
            <div class="hero-name-row">
              <h1 class="hero-name">{{ profile.nickname }}</h1>
              <span v-if="level" class="hero-level">Lv.{{ level }}</span>
              <!-- 有装扮样式用 TitleBadge 还原后台配色（文字色 / 背景 / 边框 / 发光 / 图标 / 动效） -->
              <TitleBadge
                v-if="profile.title || titlePayload"
                :payload="titlePayload || {}"
                :text="profile.title"
                size="md"
              />
              <span v-else-if="profile.title" :class="['author-title', titleClass]">{{ profile.title }}</span>
              <!-- 佩戴中的勋章：users.json.wearing_badge，佩戴时后端已把名称/图标/配色/稀有度冗余写入 -->
              <WearingBadge v-if="profile.json?.wearing_badge" :badge="profile.json.wearing_badge" />
              <!-- 封禁标识（后端 users/one 的 result.ban）：申诉中 / 申诉驳回期间也保持显示 -->
              <span v-if="isBanned" class="hero-ban-tag" :class="{ 'is-appeal': banStatus !== 0 }">
                <i class="bi" :class="banStatus === 0 ? 'bi-slash-circle' : 'bi-hourglass-split'" aria-hidden="true" />
                {{ banStatusText }}
              </span>
            </div>
            <!-- 用户 ID：仅手机端展示（参考移动端个人主页的 ID 行） -->
            <div v-if="profile.id" class="hero-id">ID {{ profile.id }}</div>
          </div>

          <div class="hero-actions">
            <template v-if="!isSelf">
              <!-- 四种按钮状态各用一套配色（后端 user-follows/relation）：
                   关注=主色 / 回关=警示色（提醒你回应）/ 互相关注=成功色 / 已关注=幽灵弱化 -->
              <button
                type="button"
                class="btn btn-sm"
                :class="followButtonClass"
                :title="followButtonTitle"
                :disabled="followBusy"
                @click="toggleFollow"
              >{{ followButtonText }}</button>
              <!-- 打赏：给这位用户送积分（自己的主页不显示，后端同样拒绝自打赏） -->
              <button
                type="button"
                class="btn btn-sm btn-ghost"
                title="用积分打赏 TA"
                @click="openReward"
              ><i class="bi bi-coin" aria-hidden="true" /> 打赏</button>
              <!-- 私信：落到用户中心的私信页并直接打开与 TA 的对话。
                   拉黑关系下不展示；未互关时只展示禁用态（后端 ChatPermission 同样会拦）。
                   未登录时保持可点，交给私信页的登录校验处理。 -->
              <RouterLink
                v-if="!blocked && canDm"
                :to="`/chat/${uid}`"
                class="btn btn-sm btn-ghost"
                title="给 TA 发私信"
              ><i class="bi bi-chat-dots" aria-hidden="true" /> 私信</RouterLink>
              <span
                v-else-if="!blocked"
                class="btn btn-sm btn-ghost is-disabled"
                title="仅互相关注的好友可以私信，关注并等 TA 回关后即可发送"
              ><i class="bi bi-chat-dots" aria-hidden="true" /> 私信</span>
              <!-- 拉黑 / 举报收进「更多」，避免低频且带破坏性的操作和关注按钮抢视线 -->
              <CardActionMenu :items="moreMenuItems" @select="onMoreMenu" />
            </template>
            <template v-else>
              <RouterLink to="/user/profile" class="btn btn-sm">
                <i class="bi bi-pencil" aria-hidden="true" /> 编辑资料
              </RouterLink>
              <button type="button" class="btn btn-sm btn-ghost" @click="userStore.logout">
                <i class="bi bi-box-arrow-right" aria-hidden="true" /> 退出登录
              </button>
            </template>
          </div>
        </div>

        <p class="hero-desc">{{ profile.description || '这个人很懒，什么都没留下' }}</p>

        <!-- 认证 / 个人网站：做成轻量胶囊，避免和签名抢行 -->
        <div v-if="cert || websiteUrl" class="hero-chips">
          <!-- 认证信息：默认只展示认证名称 + 后台填写的认证说明（认证领域不对外展示） -->
          <span v-if="cert" class="hero-chip">
            <i class="bi bi-patch-check-fill" :style="{ color: certIconColor }" aria-hidden="true" />
            <span class="hero-chip-strong">{{ cert.title }}</span>
            <span v-if="cert.description" class="hero-chip-desc">{{ cert.description }}</span>
          </span>
          <!-- 个人网站：地址经 safeUrl 过滤，非法协议不生成链接（退化为纯文本胶囊） -->
          <a
            v-if="websiteUrl"
            class="hero-chip hero-chip-link"
            :href="websiteUrl"
            target="_blank"
            rel="noopener noreferrer"
          >
            <i class="bi bi-link-45deg" aria-hidden="true" /> {{ website.name || websiteUrl }}
          </a>
        </div>

        <!--
          勋章：默认只展示昵称旁佩戴的那一枚（WearingBadge），
          全部已获得的勋章收在折叠入口后面，避免主页被勋章墙占满。
          未获得的属于「任务进度」，任何情况下都不在这里露出。
        -->
        <div v-if="badges.length" class="hero-badges">
          <template v-if="showAllBadges">
            <AchievementBadge
              v-for="item in badges"
              :key="item.id"
              :badge="item"
              size="sm"
              held
            />
          </template>
          <button type="button" class="hero-badges-toggle" @click="showAllBadges = !showAllBadges">
            <i class="bi" :class="showAllBadges ? 'bi-chevron-up' : 'bi-award'" aria-hidden="true" />
            {{ showAllBadges ? '收起勋章' : `全部勋章 ${badges.length}` }}
          </button>
        </div>

        <!-- 社交数据：文章 / 动态 / 关注 / 粉丝 / 获赞
             数字来自后端 user-follows/stats（只统计对外公开的内容：audit=1 且 status=1），
             关注 / 粉丝随关注按钮实时增减，其余为只读展示（不做跳转） -->
        <div class="hero-stats">
          <button type="button" class="hero-stat" @click="switchTab('article')">
            <strong>{{ stats.article }}</strong>
            <span>文章</span>
          </button>
          <div class="hero-stat is-static">
            <strong>{{ stats.moment }}</strong>
            <span>动态</span>
          </div>
          <button type="button" class="hero-stat" @click="switchTab('follow')">
            <strong>{{ stats.follow }}</strong>
            <span>关注</span>
          </button>
          <button type="button" class="hero-stat" @click="switchTab('fans')">
            <strong>{{ stats.fans }}</strong>
            <span>粉丝</span>
          </button>
          <div class="hero-stat is-static">
            <strong>{{ stats.likes }}</strong>
            <span>获赞</span>
          </div>
        </div>

        <!-- 封禁提示条：该账号处于封禁中，展示关键信息并提供详情 / 申诉入口 -->
        <div v-if="isBanned" class="hero-ban">
          <i class="bi bi-exclamation-octagon" aria-hidden="true" />
          <div class="hero-ban-text">
            <strong>{{ banBarTitle }}</strong>
            <span>{{ banSummary }}</span>
          </div>
          <button type="button" class="btn btn-sm btn-outline-danger" @click="openBanDialog">
            查看详情<template v-if="isSelf"> / 申诉</template>
          </button>
        </div>
      </div>
    </section>

    <!-- ===== 内容区：吸顶标签页 + 列表 ===== -->
    <section class="panel">
      <div class="tab-bar" role="tablist" :aria-label="`${who}的内容`">
        <button
          v-for="t in TABS"
          :key="t.key"
          type="button"
          role="tab"
          class="tab"
          :class="{ 'is-active': tab === t.key }"
          :aria-selected="tab === t.key"
          @click="switchTab(t.key)"
        >
          {{ t.label }}
          <span v-if="t.count !== null" class="tab-count">{{ t.count }}</span>
        </button>
      </div>

      <div class="panel-body">
        <div v-if="loading && !list.length" class="is-loading"><span class="spinner" /> 加载中…</div>
        <div v-else-if="denied" class="empty-row">
          <EmptyState :text="deniedText" />
        </div>

        <!-- 文章列表 -->
        <template v-else-if="tab === 'article'">
          <div v-if="!list.length" class="empty-row">
            <EmptyState :text="emptyText" />
          </div>
          <template v-else>
            <ArticleCard v-for="a in list" :key="a.id" :article="a" :abstract-limit="50" />
            <div v-if="!finished" class="load-more">
              <button type="button" class="btn btn-ghost btn-sm" :disabled="loading" @click="loadMore">
                <span v-if="loading" class="spinner spinner-sm" />
                {{ loading ? '加载中…' : '加载更多' }}
              </button>
            </div>
            <p v-else class="list-end">没有更多了</p>
          </template>
        </template>

        <!-- 粉丝 / 关注 列表 -->
        <template v-else-if="tab === 'fans' || tab === 'follow'">
          <!-- 粉丝管理条：只有本人在自己的粉丝列表里看得到 -->
          <div v-if="isSelf && tab === 'fans'" class="fans-bar">
            <template v-if="!fanManageMode">
              <span class="fans-bar-text">共 {{ stats.fans }} 位粉丝</span>
              <button v-if="list.length" type="button" class="btn btn-xs btn-ghost" @click="enterFanManage">
                <i class="bi bi-check2-square" aria-hidden="true" /> 管理
              </button>
            </template>
            <template v-else>
              <label class="fans-check">
                <FormCheckbox
                  :model-value="allFansSelected"
                  :indeterminate="partFansSelected"
                  @change="toggleSelectAllFans" />
                <span>全选本页</span>
              </label>
              <span class="fans-bar-text">已选 {{ selectedFanIds.length }} 人</span>
              <div class="fans-bar-actions">
                <button type="button" class="btn btn-xs btn-ghost" :disabled="fanRemoving" @click="exitFanManage">
                  取消
                </button>
                <button
                  type="button"
                  class="btn btn-xs btn-danger"
                  :disabled="!selectedFanIds.length || fanRemoving"
                  @click="confirmRemoveFansVisible = true"
                >移除</button>
              </div>
            </template>
          </div>

          <div v-if="!list.length" class="empty-row">
            <EmptyState :text="emptyText" />
          </div>
          <ul v-else class="row-list" :class="{ 'is-managing': fanManageMode && tab === 'fans' }">
            <li
              v-for="item in list"
              :key="item.id"
              class="row-item"
              @click="onFanRowClick($event, item)"
            >
              <!-- 批量模式：勾选框只在粉丝 tab 出现（关注 tab 不做批量操作） -->
              <FormCheckbox
                v-if="fanManageMode && tab === 'fans'"
                v-model="selectedFanIds"
                class="row-check"
                :value="item.id"
                :aria-label="`选择 ${item.nickname || '用户'}`" />
              <RouterLink :to="`/author/${item.uid || item.user_id || item.id}`" class="row-link">
                <img
                  :src="thumbUrl(item.avatar || defaultAvatar, 88)"
                  class="row-avatar"
                  alt=""
                  loading="lazy"
                  decoding="async"
                  @error="onImgError"
                />
                <div class="row-info">
                  <span class="row-name">
                    {{ item.nickname || item.name || '匿名用户' }}
                    <!-- 互关标识：后端 attachRelation 批量算好（未登录时不显示） -->
                    <span v-if="item.mutual" class="row-tag" title="你们互相关注">互关</span>
                    <span v-else-if="item.is_following" class="row-tag is-weak" title="你已关注 TA">已关注</span>
                  </span>
                  <span class="row-sub">{{ item.description || '这个人很懒，什么都没留下' }}</span>
                </div>
              </RouterLink>
              <div class="row-tail">
                <time class="row-time">{{ fromNow(item.create_time) }}</time>
                <!-- 行内关注：关系态由后端 attachRelation 批量给出，点完就地更新，不整页刷新
                     （粉丝管理模式里专注勾选，先不显示关注按钮） -->
                <button
                  v-if="canFollowRow(item) && !(fanManageMode && tab === 'fans')"
                  type="button"
                  class="btn btn-xs"
                  :class="item.is_following ? 'btn-ghost' : 'btn-primary'"
                  :disabled="followBusy"
                  :title="item.is_following ? '取消关注' : '关注 TA'"
                  @click="toggleFollowRow(item)"
                >{{ followRowText(item) }}</button>
              </div>
            </li>
          </ul>
        </template>

        <!-- 点赞 / 收藏 列表（含 文章/评论/动态 子分类） -->
        <template v-else>
          <div class="sub-tab-bar">
            <button
              v-for="st in currentSubTabs"
              :key="st.key"
              type="button"
              class="sub-tab"
              :class="{ 'is-active': subTab === st.key }"
              @click="switchSubTab(st.key)"
            >{{ st.label }}</button>
          </div>
          <div v-if="!list.length" class="empty-row">
            <EmptyState :text="emptyText" />
          </div>
          <ul v-else class="row-list">
            <li v-for="(i, k) in list" :key="i.id || k" class="row-item is-clickable" @click="goDetail(i)">
              <span class="like-badge" :class="badgeClass(i.target_type)">{{ typeLabel(i.target_type) }}</span>
              <span class="like-title" v-html="detailTitleHtml(i)" />
              <time class="row-time">{{ fromNow(i.create_time) }}</time>
            </li>
          </ul>
        </template>
      </div>
    </section>

    <!-- 可能认识的人：仅在自己的主页展示（基于「共同关注」的二度推荐，见后端 user-follows/suggest）。
         只展示聚合出来的「N 个共同关注」，不暴露任何具体的关注关系明细。 -->
    <section v-if="isSelf && suggestions.length" class="panel suggest-panel">
      <header class="suggest-head">
        <h2 class="suggest-title"><i class="bi bi-person-plus" aria-hidden="true" /> 可能认识的人</h2>
        <span class="suggest-tip">根据你们共同关注的人推荐</span>
      </header>
      <!-- 横向滑动卡片：人多了也只在一条轨道里滑动，不会把主页拉长 -->
      <div class="suggest-track">
        <button
          v-if="suggestScrollable"
          type="button"
          class="suggest-nav is-prev"
          :disabled="!canScrollPrev"
          aria-label="向左查看更多"
          @click="scrollSuggest(-1)"
        >
          <i class="bi bi-chevron-left" aria-hidden="true" />
        </button>
        <div
          ref="suggestScroller"
          class="suggest-scroller"
          @scroll="updateSuggestScroll"
          @wheel="onSuggestWheel"
        >
        <article v-for="u in suggestions" :key="u.id" class="suggest-card">
          <RouterLink :to="`/author/${u.id}`" class="suggest-card-link">
            <img
              :src="thumbUrl(u.avatar || defaultAvatar, 96)"
              class="suggest-avatar"
              alt=""
              loading="lazy"
              decoding="async"
              @error="onImgError"
            />
            <span class="suggest-name">{{ u.nickname || '匿名用户' }}</span>
            <span class="suggest-sub">
              <template v-if="u.mutual_count">{{ u.mutual_count }} 个共同关注</template>
              <template v-else>{{ u.description || '这个人很懒，什么都没留下' }}</template>
            </span>
          </RouterLink>
          <button
            type="button"
            class="btn btn-xs suggest-follow"
            :class="u.following ? 'btn-ghost' : 'btn-primary'"
            :disabled="followBusy"
            @click="followSuggestion(u)"
          >{{ u.following ? '已关注' : '关注' }}</button>
        </article>
        </div>
        <button
          v-if="suggestScrollable"
          type="button"
          class="suggest-nav is-next"
          :disabled="!canScrollNext"
          aria-label="向右查看更多"
          @click="scrollSuggest(1)"
        >
          <i class="bi bi-chevron-right" aria-hidden="true" />
        </button>
      </div>
    </section>

    <!-- 封禁信息弹窗：本人可提交申诉，查看他人主页时只展示信息 -->
    <BanAppealDialog ref="banDialogRef" :ban="banInfo" :appealable="isSelf" />

    <!-- 举报用户：提交后只通知管理员，被举报人不会收到任何提示 -->
    <ReportDialog
      :visible="reportVisible"
      target-type="user"
      :target-id="uid"
      :target-label="`用户「${profile.nickname || '匿名'}」`"
      :loading="reporting"
      @update:visible="(v) => (reportVisible = v)"
      @confirm="submitReport"
    />

    <!-- 打赏用户：bind_type=user 时 bind_id 就是接收者 uid -->
    <RewardDialog
      v-model:visible="rewardVisible"
      bind-type="user"
      :bind-id="uid"
      :to-uid="uid"
      :to-name="profile.nickname || ''"
    />

    <!-- 移除粉丝：二次确认，避免批量误删 -->
    <ConfirmDialog
      v-model:visible="confirmRemoveFansVisible"
      danger
      title="移除粉丝"
      :message="`确定移除已选的 ${selectedFanIds.length} 位粉丝吗？移除后对方不再出现在你的粉丝列表里，也不会收到通知。`"
      confirm-text="移除"
      :loading="fanRemoving"
      @confirm="removeSelectedFans"
    />
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ArticleCard from '@/components/ArticleCard.vue'
import AvatarFrame from '@/components/AvatarFrame.vue'
import CardActionMenu from '@/components/CardActionMenu.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import TitleBadge from '@/components/TitleBadge.vue'
import WearingBadge from '@/components/WearingBadge.vue'
import AchievementBadge from '@/components/AchievementBadge.vue'
import { getUserBadges } from '@/api/badge'
import { getUserCertification } from '@/api/user-certification'
import { certColor } from '@/utils/cert'
import BanAppealDialog from '@/components/BanAppealDialog.vue'
import ReportDialog from '@/components/ReportDialog.vue'
import RewardDialog from '@/components/RewardDialog.vue'
import { blockUser, unblockUser, blockStatus } from '@/api/user-blocks'
import { createReport } from '@/api/report'
import { getUser } from '@/api/users'
import { getAuthorArticles } from '@/api/article'
import { myLikes, myCollects } from '@/api/tags'
import { call } from '@/api/request'
import { useUserStore } from '@/stores/user'
import { fromNow, formatDate } from '@/utils/time'
import { banTypeText } from '@/utils/user'
import { pickUserTitlePayload } from '@/utils/helper'
import { resolveFrameRender } from '@/utils/frameRender'
import { toast } from '@/utils/toast'
import { renderEmoji } from '@/utils/emoji'
import { thumbUrl } from '@/utils/image'
import EmptyState from '@/components/EmptyState.vue'
import { safeUrl } from '@/utils/url'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const uid = computed(() => Number(route.params.id))

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const profile = ref({})
const level = ref(0)

// 头像尺寸：桌面 96 / 手机 76。AvatarFrame 会按 size 取 2 倍图，
// 所以这里改小不只是视觉变化，也直接减少头像的下载量。
const isMobile = ref(false)
function syncViewport() {
  isMobile.value = window.innerWidth <= 640
}
const avatarSize = computed(() => (isMobile.value ? '76px' : '96px'))
onMounted(() => {
  syncViewport()
  window.addEventListener('resize', syncViewport)
  window.addEventListener('resize', updateSuggestScroll)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', syncViewport)
  window.removeEventListener('resize', updateSuggestScroll)
})

// 个人网站（来自 json.website，用户在「个人资料」页设置）
//
// 这里必须再过一遍协议白名单：资料页只在输入框上校验了 ^https?://，
// 直接调接口提交 `javascript:` 就能绕过，而这是别人主页上的链接，
// 点一下就等于在本站域下执行脚本（token 存在可读 cookie 里）。
const website = computed(() => (profile.value?.json && profile.value.json.website) || null)
const websiteUrl = computed(() => safeUrl(website.value?.url))

// 头像框（来自 json.frame，佩戴头像框时由服务端回写；frame_animated 为动效开关）
const frame = computed(() => (profile.value?.json && profile.value.json.frame) || '')
// 头像框渲染参数：以装扮 payload 为准（users.json 上的副本只作兜底）
const frameRender = computed(() => resolveFrameRender(profile.value?.json))

// 主页封面（来自 json.cover）：用户在「我的资料」里二选一
//   { type: 'image', value: 'https://...' }  自定义图片
//   { type: 'color', value: '#c7482a' }      自定义颜色
// 兼容早期只存字符串（图片地址）的写法。
const cover = computed(() => {
  const raw = profile.value?.json && profile.value.json.cover
  if (!raw) return null
  if (typeof raw === 'string') return { type: 'image', value: raw }
  if (typeof raw === 'object' && raw.value) {
    return { type: raw.type === 'color' ? 'color' : 'image', value: raw.value }
  }
  return null
})

// 交给 CSS 变量 --author-cover：图片用 url()，纯色用「同色渐变」
// （background-image 不接受裸颜色值，必须包成渐变）。
// 未设置封面时不写该变量，由 CSS 里的主题色渐变兜底。
// 图片封面走 thumbUrl 按横幅宽度取图：主页封面是用户上传的大图，
// 原图动辄 1~3MB，横幅实际只显示 644px 宽。
const coverStyle = computed(() => {
  const item = cover.value
  if (!item) return {}
  if (item.type === 'color') {
    return { '--author-cover': `linear-gradient(0deg, ${item.value}, ${item.value})` }
  }
  return { '--author-cover': `url("${thumbUrl(item.value, 1300)}")` }
})

// 头衔（与 Profile 页一致的颜色映射）
const PRESET_TITLES = ['掌门', '长老', '护法', '内门弟子', '外门弟子', '炼气修士', '筑基修士', '结丹修士', '元婴老祖', '化神大能']
const titleColorMap = {
  '掌门': 'title-zhangmen',
  '长老': 'title-zhanglao',
  '护法': 'title-hufa',
  '内门弟子': 'title-neimen',
  '外门弟子': 'title-waimen',
  '炼气修士': 'title-lianqi',
  '筑基修士': 'title-zhuji',
  '结丹修士': 'title-jiedan',
  '元婴老祖': 'title-yuanying',
  '化神大能': 'title-huashen',
  '侠客': 'title-xiake',
  '学徒': 'title-xuetu'
}
function getTitleColorClass(title) {
  return titleColorMap[title] || 'title-default'
}
const titleClass = computed(() => getTitleColorClass(profile.value.title))
// 头衔样式：装扮系统写入的 payload；无装扮时返回 null，走上面的色板兜底
const titlePayload = computed(() => pickUserTitlePayload(profile.value))
// 社交数据卡片：文章 / 动态 / 关注 / 粉丝 / 获赞（后端 user-follows/stats 一次拿全）
const stats = ref({ article: 0, moment: 0, follow: 0, fans: 0, likes: 0 })
// 我与目标用户的关系：is_following（我关注了 TA）/ is_follower（TA 关注了我）/ mutual（互关）
const relation = ref({ is_following: false, is_follower: false, mutual: false })
// 关注 / 取关进行中：防连点（后端有幂等处理，但前端不该重复发请求）
const followBusy = ref(false)
// 我是否已拉黑 TA（拉黑是双向屏蔽：彼此内容互不进流 + 不能互相私信）
const blocked = ref(false)
// 举报弹窗
const reportVisible = ref(false)
const reporting = ref(false)

// 打赏弹窗（接收者由后端按用户 uid 解析）
const rewardVisible = ref(false)

function openReward() {
  rewardVisible.value = true
}
// 可能认识的人：仅本人的主页展示
const suggestions = ref([])
const isSelf = computed(() => userStore.isLogged && Number(userStore.user?.id) === uid.value)

// 私信门槛：私信默认只对互关好友开放（后端 model.ChatPermission 兜底）。
// 未登录时保持可点，进私信页后再走登录校验。
const canDm = computed(() => !userStore.isLogged || relation.value.mutual)

// 关注按钮文案：互相关注 > 已关注 > 回关（对方关注了我，我还没回）> 关注
const followButtonText = computed(() => {
  if (relation.value.mutual) return '互相关注'
  if (relation.value.is_following) return '已关注'
  if (relation.value.is_follower) return '回关'
  return '关注'
})

/**
 * 关注按钮配色：让「需要你回应」和「已互相关注」一眼能区分出来 ——
 *   回关      → 警示色（金色，提示有待处理的互动）
 *   互相关注  → 成功色（绿色，表示双向关系已建立）
 *   已关注    → 幽灵按钮（弱化，避免和主要操作抢视线）
 *   关注      → 主色（行动号召）
 * 颜色都取自全局变量（assets/styles.css 的 .btn-success / .btn-warning），不新造配色。
 */
const followButtonClass = computed(() => {
  if (relation.value.mutual) return 'btn-success'
  if (relation.value.is_following) return 'btn-ghost'
  if (relation.value.is_follower) return 'btn-warning'
  return 'btn-primary'
})

// 按钮悬浮说明写全，避免「已关注」这种短文案让人不知道该点会发生什么
const followButtonTitle = computed(() => {
  if (relation.value.mutual) return '你们互相关注，点击取消关注（互相关注关系会解除）'
  if (relation.value.is_following) return '你已关注 TA，点击取消关注'
  if (relation.value.is_follower) return 'TA 关注了你，点击回关（回关后你们互相关注）'
  return '关注 TA，TA 的动态会出现在你的关注流里'
})

// 「更多」菜单：拉黑 / 举报都是低频且带破坏性的操作，收进三点菜单里
const moreMenuItems = computed(() => [
  {
    key: 'block',
    label: blocked.value ? '解除拉黑' : '拉黑',
    icon: blocked.value ? 'bi bi-unlock' : 'bi bi-slash-circle',
    danger: !blocked.value
  },
  { key: 'report', label: '举报该用户', icon: 'bi bi-flag', danger: true }
])

function onMoreMenu(key) {
  if (key === 'block') toggleBlock()
  else if (key === 'report') reportVisible.value = true
}

// ---------- 封禁信息（后端 users/one 的 result.ban，见 model.Users.banInfo） ----------
const banInfo = ref(null)
const banDialogRef = ref(null)
// 自动弹窗对同一个主页只弹一次，避免切 Tab、重复加载时反复打扰
const banShown = ref(false)

const isBanned = computed(() => Boolean(banInfo.value?.is_banned))

// 封禁状态：0 封禁中 / 3 申诉中 / 5 申诉驳回 —— 申诉期间封禁继续生效，
// 所以标识要一直显示到真正解封为止（后端 BanStatusRestricted 口径）
const banStatus = computed(() => {
  const info = banInfo.value
  if (!info?.is_banned) return 0
  return Number(info.status ?? info.record?.status ?? 0) || 0
})

const banStatusText = computed(() => {
  if (banStatus.value === 3) return '申诉中'
  if (banStatus.value === 5) return '申诉驳回'
  return '封禁中'
})

const banBarTitle = computed(() => {
  if (banStatus.value === 3) return '该账号处于封禁中（申诉审核中）'
  if (banStatus.value === 5) return '该账号处于封禁中（申诉未通过）'
  return '该账号处于封禁中'
})

// 封禁摘要：限制范围 · 时长 · 原因 · 到期
const banSummary = computed(() => {
  const record = banInfo.value?.record
  if (!record) return '该账号已被限制使用'
  const days = Number(record.duration) || 0
  const parts = [banTypeText(record.ban_type), days === 0 ? '永久封禁' : `${days} 天`]
  if (record.reason) parts.push(`原因：${record.reason}`)
  if (Number(record.expires_at) > 0) parts.push(`到期：${formatDate(record.expires_at)}`)
  // 申诉中把申诉时间也带上，方便对照处理进度
  if (banStatus.value === 3 && Number(record.appeal_time) > 0) {
    parts.push(`申诉时间：${formatDate(record.appeal_time)}`)
  }
  return parts.join(' · ')
})

function openBanDialog() {
  banDialogRef.value?.show()
}

const tab = ref('article')
const list = ref([])
const loading = ref(false)
const denied = ref(false)
const page = ref(1)
const pageSize = 10
const finished = ref(false)

// 第一人称 / 第三人称：看自己的主页时文案统一用「我」，避免页面上出现「TA还没有…」
const who = computed(() => (isSelf.value ? '我' : 'TA'))

// 标签页：count 只给后端统计得到的三个（文章 / 粉丝 / 关注），
// 点赞、收藏没有对应统计接口，留 null 表示不显示徽标。
// 标签文案不再带「TA的」前缀：主页已经是这个人的主页，前缀只会让 5 个标签排不下
const TABS = computed(() => [
  { key: 'article', label: '文章', count: stats.value.article },
  { key: 'fans', label: '粉丝', count: stats.value.fans },
  { key: 'follow', label: '关注', count: stats.value.follow },
  { key: 'like', label: '点赞', count: null },
  { key: 'collect', label: '收藏', count: null }
])

const emptyText = computed(() => {
  if (tab.value === 'article') return `${who.value}还没有发布文章`
  if (tab.value === 'fans') return `${who.value}还没有粉丝`
  if (tab.value === 'follow') return `${who.value}还没有关注任何人`
  if (tab.value === 'like') return `${who.value}还没有点赞内容`
  return `${who.value}还没有收藏内容`
})

const deniedText = ref('仅本人可见')

function onImgError(e) {
  e.target.src = defaultAvatar
}

function switchTab(t) {
  if (tab.value === t) return
  tab.value = t
  subTab.value = 'all'
  page.value = 1
  finished.value = false
  denied.value = false
  // 清空旧数据：否则新标签加载期间会先渲染上一个标签的列表，看起来像串了内容
  list.value = []
  // 粉丝管理模式只在粉丝 tab 内有效，切走就退出并清空勾选
  fanManageMode.value = false
  selectedFanIds.value = []
  load()
}

// 点赞子分类（点赞支持 文章/评论/动态），默认全部
const LIKE_TABS = [
  { key: 'all', label: '全部' },
  { key: 'article', label: '文章' },
  { key: 'comment', label: '评论' },
  { key: 'moment', label: '动态' }
]
// 收藏子分类（收藏不支持评论，仅 文章/动态），默认全部
const COLLECT_TABS = [
  { key: 'all', label: '全部' },
  { key: 'article', label: '文章' },
  { key: 'moment', label: '动态' }
]
const subTab = ref('all')
// 根据当前是点赞还是收藏，返回对应的子分类
const currentSubTabs = computed(() => (tab.value === 'collect' ? COLLECT_TABS : LIKE_TABS))

function switchSubTab(t) {
  if (subTab.value === t) return
  subTab.value = t
  page.value = 1
  finished.value = false
  load()
}

function typeLabel(t) {
  if (t === 'article') return '文章'
  if (t === 'comment') return '评论'
  if (t === 'moments' || t === 'moment') return '动态'
  if (t === 'page') return '页面'
  if (t === 'users' || t === 'user') return '用户'
  return '内容'
}

function badgeClass(t) {
  if (t === 'article') return 'badge-article'
  if (t === 'comment') return 'badge-comment'
  if (t === 'moment' || t === 'moments') return 'badge-moment'
  return 'badge-other'
}

// 详情标题：优先 _detail（enrich 后），再回退 result.author.nickname（user 类型）
function detailTitle(i) {
  const d = i._detail
  if (d) {
    if (d.title) return d.title
    // 动态/评论取完整 content（不截断），避免切断 [emoji:url] 表情标记，
    // 显示截断交由 CSS 单行省略处理
    if (d.content) return d.content
  }
  if (i.target_type === 'user' && i.result?.author?.nickname) {
    return i.result.author.nickname
  }
  return '内容'
}

// 标题 HTML（含 [emoji:url] 表情渲染）
function detailTitleHtml(i) {
  return renderEmoji(detailTitle(i), { size: 18 })
}

// 点击跳转（参考 Cardify-inis goToLikedContent）
function goDetail(i) {
  const type = i.target_type
  const id = i.target_id || i.id
  if (type === 'comment') {
    const bind = i._detail?.bind_type || i.bind_type || i.result?.bind_type
    const bid = i._detail?.bind_id || i.bind_id || i.result?.bind_id
    if (bind === 'article' && bid) return router.push(`/archives/${bid}`)
    if ((bind === 'moment' || bind === 'moments') && bid) return router.push(`/moments/${bid}`)
    if (bind === 'page' && bid) return router.push(`/page/${bid}`)
    return
  }
  if (type === 'moment' || type === 'moments') return router.push(`/moments/${id}`)
  if (type === 'page') return router.push(`/page/${id}`)
  return router.push(`/archives/${id}`)
}

// 点赞/收藏列表：按 target_type 批量补充被操作对象详情（参考 Cardify-inis fetchBatchDetails）
const endpointMap = {
  article: { ctrl: 'article', field: 'id,title,abstract,content,views,create_time' },
  comment: { ctrl: 'comment', field: 'id,content,bind_id,bind_type,uid,create_time' },
  moment: { ctrl: 'moments', field: 'id,content,images,location,uid,create_time' },
  page: { ctrl: 'article', field: 'id,title,content,views,create_time' }
}

async function enrichTargets(items) {
  const grouped = {}
  for (const it of items) {
    const type = it.target_type
    const id = it.target_id
    if (!type || !id) continue
    ;(grouped[type] = grouped[type] || []).push(id)
  }
  const detailMaps = {}
  await Promise.all(
    Object.entries(grouped).map(async ([type, ids]) => {
      const meta = endpointMap[type]
      if (!meta) return
      const uniq = [...new Set(ids.map(Number).filter(Boolean))]
      try {
        const res = await call(meta.ctrl, 'all', {
          method: 'GET',
          params: {
            where: JSON.stringify({ id: { $in: uniq } }),
            limit: uniq.length,
            field: meta.field
          }
        })
        const rows = res.data?.data || res.data?.list || []
        const m = new Map()
        for (const r of rows) if (r && r.id !== undefined) m.set(String(r.id), r)
        detailMaps[type] = m
      } catch {
        // 单组失败不影响整体
      }
    })
  )
  for (const it of items) {
    const m = detailMaps[it.target_type]
    if (m && m.has(String(it.target_id))) {
      it._detail = m.get(String(it.target_id))
    }
  }
}

// 已获得的勋章（公开接口）：默认不展开，主页只显示昵称旁佩戴的那一枚
const badges = ref([])
const showAllBadges = ref(false)

// 认证摘要（公开接口，未认证 / 审核中 / 已过期时返回 null）
const cert = ref(null)

// 认证图标颜色：与 VerifiedBadge 共用同一取色函数，避免两处颜色不一致
const certIconColor = computed(() => certColor(cert.value || {}))

async function loadCertification() {
  try {
    const res = await getUserCertification(uid.value)
    cert.value = res.data || null
  } catch {
    cert.value = null
  }
}

async function loadBadges() {
  try {
    const res = await getUserBadges(uid.value)
    badges.value = Array.isArray(res.data) ? res.data : []
  } catch {
    badges.value = []
  }
}

async function loadProfile() {
  try {
    const res = await getUser(uid.value)
    const u = res.data || {}
    profile.value = {
      // 补上 id：手机端的「ID xxx」行依赖它，之前漏了字段导致这一行永远不显示
      id: u.id || uid.value,
      nickname: u.nickname || '匿名用户',
      avatar: u.avatar || '',
      description: u.description || u.remarks || '',
      title: u.title || '',
      json: u.json || {}
    }
    // 等级：兼容 result.level.current 与 users_rating.grade 两种结构
    const lv = u.result?.level?.current || u.users_rating?.users_grade
    level.value = lv?.value || u.users_rating?.grade || u.grade || 0

    // 封禁信息：被封禁时页面显示「封禁中」标识
    banInfo.value = u.result?.ban || null
    // 查看他人被封禁的主页 → 自动弹出封禁详情（只展示信息，不提供申诉）；
    // 自己被封禁时由全局弹窗（MainLayout 的 BanAppealDialog）负责提醒，这里不再重复弹
    if (isBanned.value && !isSelf.value && !banShown.value) {
      banShown.value = true
      await nextTick()
      banDialogRef.value?.show()
    }
    // 社交数据卡片（文章 / 动态 / 关注 / 粉丝 / 获赞）统一由后端 user-follows/stats 提供：
    // 口径只统计公开内容，且不会像原来那样「文章数用 countArticlesByAuthor、关注/粉丝用 counts」各算一份
    loadSocialStats()
    loadBadges()
    loadCertification()
  } catch {
    profile.value = { nickname: '用户不存在', avatar: '', description: '' }
  }
}

/**
 * 社交数据卡片（文章 / 动态 / 关注 / 粉丝 / 获赞）
 *
 * 后端 user-follows/stats 一次拿全，只统计**对外公开**的内容（audit=1 且 status=1）；
 * 获赞数需要跨表统计，后端缓存 5 分钟，关注 / 取关时会主动清掉缓存。
 */
async function loadSocialStats() {
  try {
    const res = await call('user-follows', 'stats', {
      method: 'GET',
      params: { uid: uid.value }
    })
    const d = res.data || {}
    stats.value = {
      article: Number(d.article) || 0,
      moment: Number(d.moment) || 0,
      follow: Number(d.follow) || 0,
      fans: Number(d.fans) || 0,
      likes: Number(d.likes) || 0
    }
  } catch {}
}

/**
 * 我与 TA 的关注关系（看自己主页 / 未登录时不需要）
 * 后端 user-follows/relation → 按钮显示「关注 / 已关注 / 互关 / 回关」
 */
async function loadRelation() {
  if (isSelf.value || !userStore.isLogged) {
    relation.value = { is_following: false, is_follower: false, mutual: false }
    return
  }
  try {
    const res = await call('user-follows', 'relation', {
      method: 'GET',
      params: { follow_uid: uid.value }
    })
    relation.value = {
      is_following: !!res.data?.is_following,
      is_follower: !!res.data?.is_follower,
      mutual: !!res.data?.mutual
    }
  } catch {}
}

/**
 * 关注 / 取关
 * 关注后：对方粉丝数 +1；若对方本来就关注我 → 直接变成「互关」
 */
async function toggleFollow() {
  if (!userStore.isLogged) {
    toast.info('请先登录')
    return
  }
  if (followBusy.value) return
  followBusy.value = true
  try {
    if (relation.value.is_following) {
      // 取消前先记下是不是互关：互相关注被取消要给出完整提示（不只是「已关注」四个字）
      const wasMutual = relation.value.mutual
      await call('user-follows', 'unfollow', { method: 'PUT', data: { follow_uid: uid.value } })
      relation.value.is_following = false
      relation.value.mutual = false
      stats.value.fans = Math.max(0, stats.value.fans - 1)
      toast.success(wasMutual ? '已取消关注，你们的互相关注关系已解除' : '已取消关注')
    } else {
      // 对方本来就关注我 → 这次是「回关」，提示要说清已经是互相关注
      const wasFollower = relation.value.is_follower
      await call('user-follows', 'follow', { method: 'POST', data: { follow_uid: uid.value } })
      relation.value.is_following = true
      relation.value.mutual = wasFollower
      stats.value.fans += 1
      toast.success(wasFollower ? '回关成功，你们现在是互相关注' : '关注成功，TA 的动态会出现在你的关注流里')
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    followBusy.value = false
  }
}

/**
 * 可能认识的人（仅本人主页展示）
 *
 * 后端 user-follows/suggest：基于「我关注的人也在关注谁」做二度推荐，
 * 已经关注过的不会出现在列表里；这里只维护「点击关注后」的按钮态与关注数。
 */
async function loadSuggestions() {
  if (!isSelf.value) {
    suggestions.value = []
    return
  }
  try {
    const res = await call('user-follows', 'suggest', { method: 'GET' })
    suggestions.value = (res.data?.list || []).map((u) => ({ ...u, following: false }))
  } catch {
    suggestions.value = []
  }
}

// ---------- 「可能认识的人」横滑轨道 ----------
// 桌面端：左右箭头 + 把纵向滚轮转成横向滚动；手机端保持原生滑动（箭头隐藏）
const suggestScroller = ref(null)
const suggestScrollable = ref(false)
const canScrollPrev = ref(false)
const canScrollNext = ref(false)

function updateSuggestScroll() {
  const el = suggestScroller.value
  if (!el) return
  const max = el.scrollWidth - el.clientWidth
  suggestScrollable.value = max > 4
  canScrollPrev.value = el.scrollLeft > 2
  canScrollNext.value = el.scrollLeft < max - 2
}

function scrollSuggest(dir) {
  const el = suggestScroller.value
  if (!el) return
  // 一次翻 2 张卡片的宽度，翻完停在卡片边缘（配合 scroll-snap）
  el.scrollBy({ left: dir * 284, behavior: 'smooth' })
}

/**
 * 滚轮横向滚动
 *
 * 只在「还能往这个方向滚」时接管滚轮；到达两端就放行，让页面继续纵向滚动，
 * 否则鼠标停在卡片上会把整页滚动卡死。
 */
function onSuggestWheel(event) {
  const el = suggestScroller.value
  if (!el) return
  const max = el.scrollWidth - el.clientWidth
  if (max <= 4) return
  const delta = Math.abs(event.deltaY) >= Math.abs(event.deltaX) ? event.deltaY : event.deltaX
  if (!delta) return
  const atStart = el.scrollLeft <= 0 && delta < 0
  const atEnd = el.scrollLeft >= max - 1 && delta > 0
  if (atStart || atEnd) return
  event.preventDefault()
  el.scrollLeft += delta
}



async function followSuggestion(user) {
  if (followBusy.value || user.following) return
  followBusy.value = true
  try {
    await call('user-follows', 'follow', { method: 'POST', data: { follow_uid: user.id } })
    user.following = true
    stats.value.follow += 1
    toast.success('已关注')
  } catch {
    /* 拦截器已提示 */
  } finally {
    followBusy.value = false
  }
}

// ---------- 粉丝 / 关注列表的行内关注 ----------
// 目标用户 id 只认 uid / user_id：粉丝列表里的 item.id 是关注记录 id，
// 拿它当 uid 去关注会关注到完全不相干的人。
function rowUid(item) {
  const id = Number(item.uid || item.user_id)
  return Number.isFinite(id) && id > 0 ? id : 0
}

function canFollowRow(item) {
  if (!userStore.isLogged) return false
  const id = rowUid(item)
  return id > 0 && id !== Number(userStore.user?.id)
}

function followRowText(item) {
  if (item.is_following) return '已关注'
  if (item.is_follower) return '回关'
  return '关注'
}

/**
 * 行内关注 / 取关
 *
 * 只更新这一行的关系态，不整页刷新；看自己的主页时顺带同步「关注」计数
 * （只有自己的关注数会因这次操作变化，别人的数字不受影响）。
 */
async function toggleFollowRow(item) {
  const targetUid = rowUid(item)
  if (!targetUid || followBusy.value) return
  followBusy.value = true
  try {
    if (item.is_following) {
      await call('user-follows', 'unfollow', { method: 'PUT', data: { follow_uid: targetUid } })
      item.is_following = false
      item.mutual = false
      if (isSelf.value) stats.value.follow = Math.max(0, stats.value.follow - 1)
      toast.success('已取消关注')
    } else {
      const wasFollower = item.is_follower
      await call('user-follows', 'follow', { method: 'POST', data: { follow_uid: targetUid } })
      item.is_following = true
      item.mutual = wasFollower
      if (isSelf.value) stats.value.follow += 1
      toast.success(wasFollower ? '回关成功，你们现在是互相关注' : '关注成功')
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    followBusy.value = false
  }
}

// ---------- 粉丝批量管理（仅本人可见） ----------
// 只处理「本页」勾选的粉丝：跨页累积容易误删，操作路径也更短
const fanManageMode = ref(false)
const selectedFanIds = ref([])
const fanRemoving = ref(false)
const confirmRemoveFansVisible = ref(false)

const allFansSelected = computed(
  () => list.value.length > 0 && selectedFanIds.value.length === list.value.length
)
const partFansSelected = computed(
  () => selectedFanIds.value.length > 0 && !allFansSelected.value
)

function enterFanManage() {
  fanManageMode.value = true
  selectedFanIds.value = []
}

function exitFanManage() {
  fanManageMode.value = false
  selectedFanIds.value = []
  confirmRemoveFansVisible.value = false
}

function toggleSelectAllFans() {
  selectedFanIds.value = allFansSelected.value ? [] : list.value.map((i) => i.id)
}

/**
 * 粉丝管理模式下整行可点：点哪都能勾选，比 16px 的勾选框好按
 *
 * 取消默认行为即可阻止 RouterLink 跳转（默认动作在事件冒泡结束后才执行）；
 * 点在勾选框上时直接放行，交给 v-model 处理，否则会被这里再切一次。
 */
function onFanRowClick(event, item) {
  if (!fanManageMode.value || tab.value !== 'fans') return
  if (event.target.closest('.row-check')) return
  event.preventDefault()
  const index = selectedFanIds.value.indexOf(item.id)
  if (index >= 0) selectedFanIds.value.splice(index, 1)
  else selectedFanIds.value.push(item.id)
}

async function removeSelectedFans() {
  const ids = selectedFanIds.value.slice()
  if (!ids.length || fanRemoving.value) return
  fanRemoving.value = true
  try {
    await call('user-follows', 'remove-fans', {
      method: 'DELETE',
      // DELETE 走 query：直接传数组会被序列化成 ids[]=，后端取不到，统一逗号分隔
      params: { ids: ids.join(',') }
    })
    const removed = new Set(ids)
    list.value = list.value.filter((i) => !removed.has(i.id))
    stats.value.fans = Math.max(0, stats.value.fans - ids.length)
    selectedFanIds.value = []
    fanManageMode.value = false
    confirmRemoveFansVisible.value = false
    toast.success(`已移除 ${ids.length} 位粉丝`)
  } catch {
    /* 拦截器已提示 */
  } finally {
    fanRemoving.value = false
  }
}

/** 是否已拉黑 TA（看自己主页 / 未登录时不需要） */
async function loadBlockState() {
  if (isSelf.value || !userStore.isLogged) {
    blocked.value = false
    return
  }
  try {
    const res = await blockStatus(uid.value)
    blocked.value = !!res.data?.blocked
  } catch {
    blocked.value = false
  }
}

/**
 * 拉黑 / 解除拉黑
 *
 * 拉黑是**双向**屏蔽：彼此的内容都不再出现在对方的动态流里，且不能互相私信；
 * 后端同时会解除双方的关注关系，所以这里要刷新关系与数据卡片，避免界面显示「已关注」。
 */
async function toggleBlock() {
  if (!userStore.isLogged) {
    toast.info('请先登录')
    return
  }
  if (followBusy.value) return
  followBusy.value = true
  try {
    if (blocked.value) {
      await unblockUser(uid.value)
      blocked.value = false
      toast.success('已解除拉黑，对方的内容会重新出现在你的动态流里')
    } else {
      await blockUser(uid.value)
      blocked.value = true
      // 拉黑会解除双方的关注关系：本地先同步，避免按钮仍显示「已关注 / 互相关注」
      relation.value = {
        is_following: false,
        is_follower: relation.value.is_follower,
        mutual: false
      }
      toast.success('已拉黑，彼此的内容都不会再出现在对方的动态流里')
    }
    loadSocialStats()
    loadRelation()
  } catch {
    /* 拦截器已提示 */
  } finally {
    followBusy.value = false
  }
}

/** 举报该用户（只通知管理员，被举报人不会收到任何提示） */
async function submitReport({ reason, detail, images }) {
  reporting.value = true
  try {
    await createReport({ target_type: 'user', target_id: uid.value, reason, detail, images })
    reportVisible.value = false
    toast.success('举报已提交，我们会尽快处理')
  } catch {
    /* 拦截器已提示（重复举报 / 超过每日上限等） */
  } finally {
    reporting.value = false
  }
}

// 获取目标用户的隐私设置（默认全部公开）
function getTargetPrivacy() {
  const p = profile.value?.json?.privacy
  return {
    follows: p?.follows || 'all',
    collects: p?.collects === 1 ? 1 : 0,
    likes: p?.likes === 1 ? 1 : 0
  }
}

// 粉丝 / 关注列表：受目标用户隐私设置控制（参考 Cardify-inis）
async function loadFollowList(kind) {
  if (!isSelf.value) {
    // 非本人：依据隐私设置判断是否可见
    const privacy = getTargetPrivacy()
    // 关注列表受 following 限制；粉丝列表受 followers 限制
    const visible = kind === 'follow'
      ? (privacy.follows === 'all' || privacy.follows === 'following')
      : (privacy.follows === 'all' || privacy.follows === 'followers')
    if (!visible) {
      denied.value = true
      deniedText.value = '对方设置了私密，无法查看'
      list.value = []
      finished.value = true
      return
    }
  }
  const res = await call('user-follows', kind === 'fans' ? 'followers' : 'following', {
    method: 'GET',
    params: { uid: uid.value, page: page.value, limit: pageSize }
  })
  // 后端私密拦截时返回 private:true
  if (res.data?.private) {
    denied.value = true
    deniedText.value = '对方设置了私密，无法查看'
    list.value = []
    finished.value = true
    return
  }
  // 参考 Cardify-inis：行含 result.follower（粉丝）/ result.followee（关注）
  const data = res.data?.list || res.data?.data || []
  const items = data.map((it) => {
    const o = kind === 'fans' ? it.result?.follower : it.result?.followee
    // 对方用户 id：粉丝列表用 item.uid，关注列表用 item.follow_uid（参考 Cardify-inis goToUser）
    const targetUid = kind === 'fans' ? (it.uid || o?.uid) : (it.follow_uid || o?.uid)
    return {
      id: it.id,
      uid: targetUid || o?.user_id || it.user_id,
      nickname: o?.nickname,
      avatar: o?.avatar,
      description: o?.description || o?.remarks,
      create_time: it.create_time,
      // 互关标识：后端 attachRelation 针对当前这一页批量算好（未登录时全为 false）
      is_following: !!it.is_following,
      is_follower: !!it.is_follower,
      mutual: !!it.mutual
    }
  })
  list.value = page.value === 1 ? items : [...list.value, ...items]
  finished.value = items.length < pageSize
}

async function load() {
  loading.value = true
  denied.value = false
  try {
    let res
    if (tab.value === 'article') {
      res = await getAuthorArticles(uid.value, { page: page.value, limit: pageSize })
      list.value = page.value === 1 ? (res.data?.data || []) : [...list.value, ...(res.data?.data || [])]
      finished.value = (res.data?.data || []).length < pageSize
    } else if (tab.value === 'fans' || tab.value === 'follow') {
      await loadFollowList(tab.value)
    } else if (tab.value === 'like') {
      if (!isSelf.value) {
        // 非本人：依据隐私设置判断是否可见
        const privacy = getTargetPrivacy()
        if (privacy.likes !== 1) {
          denied.value = true
          deniedText.value = '对方设置了私密，无法查看'
          list.value = []
          finished.value = true
          return
        }
        const likeParams = { page: page.value, limit: pageSize, uid: uid.value }
        if (subTab.value !== 'all') likeParams.target_type = subTab.value
        res = await myLikes(likeParams)
        if (res?.data?.private) {
          denied.value = true
          deniedText.value = '对方设置了私密，无法查看'
          list.value = []
          finished.value = true
          return
        }
        const items = res.data?.list || res.data?.data || []
        await enrichTargets(items)
        list.value = page.value === 1 ? items : [...list.value, ...items]
        finished.value = items.length < pageSize
        return
      }
      // 按子分类传 target_type（article/comment/moment），参考 Cardify-inis
      // 默认"全部"不传 target_type，查回所有类型；选具体分类再按 target_type 过滤
      const likeParams = { page: page.value, limit: pageSize }
      if (subTab.value !== 'all') likeParams.target_type = subTab.value
      res = await myLikes(likeParams)
      const items = res.data?.list || res.data?.data || []
      await enrichTargets(items)
      list.value = page.value === 1 ? items : [...list.value, ...items]
      finished.value = items.length < pageSize
    } else if (tab.value === 'collect') {
      if (!isSelf.value) {
        // 非本人：依据隐私设置判断是否可见
        const privacy = getTargetPrivacy()
        if (privacy.collects !== 1) {
          denied.value = true
          deniedText.value = '对方设置了私密，无法查看'
          list.value = []
          finished.value = true
          return
        }
        const collectParams = { page: page.value, limit: pageSize, uid: uid.value }
        if (subTab.value !== 'all') collectParams.target_type = subTab.value
        res = await myCollects(collectParams)
        if (res?.data?.private) {
          denied.value = true
          deniedText.value = '对方设置了私密，无法查看'
          list.value = []
          finished.value = true
          return
        }
        const items = res.data?.list || res.data?.data || []
        await enrichTargets(items)
        list.value = page.value === 1 ? items : [...list.value, ...items]
        finished.value = items.length < pageSize
        return
      }
      const collectParams = { page: page.value, limit: pageSize }
      if (subTab.value !== 'all') collectParams.target_type = subTab.value
      res = await myCollects(collectParams)
      const items = res.data?.list || res.data?.data || []
      await enrichTargets(items)
      list.value = page.value === 1 ? items : [...list.value, ...items]
      finished.value = items.length < pageSize
    }
  } catch (e) {
    // 接口返回了错误信息（如「请先登录！」）时，显示接口返回的提示，而非空状态文案
    if (e?.msg) {
      denied.value = true
      deniedText.value = e.msg
      list.value = []
      finished.value = true
    } else {
      list.value = page.value === 1 ? [] : list.value
    }
  } finally {
    loading.value = false
  }
}

// 文章列表加载更多（翻页）
function loadMore() {
  if (loading.value || finished.value) return
  page.value += 1
  load()
}

onMounted(() => {
  loadProfile()
  loadRelation()
  loadBlockState()
  loadSuggestions()
  load()
})

watch(() => route.params.id, () => {
  // 切换用户主页：重置封禁状态，新主页若被封禁会重新自动弹窗
  banInfo.value = null
  banShown.value = false
  suggestions.value = []
  blocked.value = false
  // 换了人，勋章墙回到「只看佩戴」的默认状态
  showAllBadges.value = false
  loadProfile()
  loadRelation()
  loadBlockState()
  loadSuggestions()
  page.value = 1
  finished.value = false
  denied.value = false
  tab.value = 'article'
  subTab.value = 'all'
  list.value = []
  fanManageMode.value = false
  selectedFanIds.value = []
  load()
})

// 推荐卡片渲染出来后再算一次箭头状态（首次进入时是空列表，数据到了才有得滚）
watch(suggestions, () => nextTick(updateSuggestScroll), { immediate: true })
</script>

<style scoped>
/* ============================================================
   用户主页（/author/:id）
   结构：资料卡（封面横幅 + 头像压边 + 身份 + 数据条）→ 内容面板 → 可能认识的人
   ============================================================ */
.author-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 资料卡 ---------- */
.hero {
  position: relative;
  overflow: hidden;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}
/* 封面横幅：优先用 users.json.cover（组件里由 :style 写成 --author-cover），
   未设置时用主色渐变兜底；顶部提亮 + 底部压暗，压边的头像与文字才不会被背景吃掉 */
.hero-cover {
  position: relative;
  height: 156px;
  background-color: var(--primary-deep);
  background-image:
    radial-gradient(120% 140% at 12% 0%, rgba(255, 255, 255, 0.2), transparent 58%),
    linear-gradient(180deg, rgba(0, 0, 0, 0) 42%, rgba(0, 0, 0, 0.34) 100%),
    var(--author-cover, linear-gradient(135deg, var(--primary) 0%, var(--primary-deep) 100%));
  background-size: cover, cover, cover;
  background-position: center;
}
/* 「更换背景」：仅本人可见，贴在横幅右下角 */
.hero-cover-edit {
  position: absolute;
  right: 12px;
  bottom: 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  background: rgba(20, 18, 15, 0.52);
  color: #fff;
  font-size: 12px;
  line-height: 1;
  text-decoration: none;
  backdrop-filter: blur(6px);
  transition: background-color var(--dur-fast) var(--ease);
}
.hero-cover-edit:hover {
  background: rgba(20, 18, 15, 0.74);
  color: #fff;
}
.hero-cover-edit .bi {
  font-size: 13px;
}

.hero-body {
  /* 底部不留 padding：数据条是卡片的「页脚」，要一直贴到卡片底边，
     否则数据条下面会多出一条 20px 的白边 */
  padding: 0 24px;
}
.hero-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 16px;
}
/* 头像上浮压住横幅下沿；描一圈卡片底色，任何封面上都清晰 */
.hero-avatar {
  position: relative;
  display: flex;
  flex-shrink: 0;
  margin-top: -42px;
}
.hero-avatar .avatar-frame {
  background: var(--bg-card);
  border-radius: 50%;
  box-shadow: 0 0 0 4px var(--bg-card);
}
.hero-ident {
  flex: 1;
  min-width: 0;
  padding-bottom: 2px;
}
.hero-name-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.hero-name {
  margin: 0;
  font-family: var(--font-serif);
  font-size: 22px;
  font-weight: 700;
  line-height: 1.3;
  letter-spacing: 0.01em;
  color: var(--text);
  overflow-wrap: anywhere;
}
.hero-level {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 11px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
/* 无装扮样式时的头衔兜底（配色见文末 .title-* 系列） */
.author-title {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  color: #fff;
}
/* 用户 ID：默认不展示，仅手机端页面显示（见 640px 适配） */
.hero-id {
  display: none;
}
.hero-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
  padding-bottom: 2px;
}

/* ---------- 封禁标识与提示条 ---------- */
.hero-ban-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 999px;
  color: var(--danger);
  background: rgba(199, 72, 42, 0.12);
  font-size: 11px;
  font-weight: 600;
}
/* 申诉中 / 申诉驳回：改用琥珀色，与「封禁中」区分开（但同样属于封禁状态） */
.hero-ban-tag.is-appeal {
  color: var(--warning);
  background: var(--gold-wash);
}
.hero-ban {
  display: flex;
  align-items: center;
  gap: 12px;
  /* 数据条是卡片页脚，这里补回资料卡底部留白 */
  margin: 16px 0 20px;
  padding: 12px 14px;
  border-radius: var(--radius);
  background: rgba(199, 72, 42, 0.06);
  border: 1px solid rgba(199, 72, 42, 0.22);
}
.hero-ban > i {
  flex-shrink: 0;
  font-size: 18px;
  color: var(--danger);
}
.hero-ban-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.hero-ban-text strong {
  font-size: 13px;
  color: var(--danger);
}
.hero-ban-text span {
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
  overflow-wrap: anywhere;
}

/* ---------- 签名 / 认证 / 网站 ---------- */
.hero-desc {
  margin: 14px 0 0;
  font-size: 13px;
  line-height: 1.75;
  color: var(--text-soft);
  overflow-wrap: anywhere;
}
.hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}
.hero-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 4px 10px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.5;
}
.hero-chip .bi {
  flex-shrink: 0;
  font-size: 13px;
}
.hero-chip-strong {
  color: var(--text-soft);
  font-weight: 600;
}
.hero-chip-desc {
  max-width: 220px;
  color: var(--text-light);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hero-chip-link {
  color: var(--primary-deep);
  text-decoration: none;
  transition: border-color var(--dur-fast) var(--ease);
}
.hero-chip-link:hover {
  border-color: var(--primary);
  color: var(--primary);
}

/* ---------- 勋章 ---------- */
.hero-badges {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}
/* 「全部勋章」的折叠入口：轻量胶囊，不和勋章本身抢视觉重量 */
.hero-badges-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12px;
  font-weight: 600;
  font-family: inherit;
  line-height: 1;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.hero-badges-toggle:hover {
  color: var(--primary);
  border-color: var(--primary);
}

/* ---------- 数据条：五等分 + 分隔线，出血到卡片两侧 ---------- */
.hero-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  margin: 18px -24px 0;
  padding: 10px 16px 12px;
  border-top: 1px solid var(--border-soft);
}
.hero-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 0;
  border: none;
  background: transparent;
  color: inherit;
  font-family: inherit;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease);
}
.hero-stat + .hero-stat {
  border-left: 1px solid var(--border-soft);
}
.hero-stat strong {
  color: var(--text);
  font-size: 17px;
  font-weight: 700;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.hero-stat span {
  color: var(--text-muted);
  font-size: 11px;
}
.hero-stat:hover strong,
.hero-stat:hover span {
  color: var(--primary);
}
/* 动态 / 获赞只是展示数字，不做跳转 → 不给手型光标与 hover 反馈，避免误导点击 */
.hero-stat.is-static {
  cursor: default;
}
.hero-stat.is-static:hover strong {
  color: var(--text);
}
.hero-stat.is-static:hover span {
  color: var(--text-muted);
}

/* ---------- 内容面板 ---------- */
.panel {
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}
/* 吸顶标签栏：列表很长时也能随时切 Tab（手机端顶栏已吸顶，这里不重复吸） */
.tab-bar {
  position: sticky;
  top: 0;
  z-index: 3;
  display: flex;
  gap: 2px;
  padding: 0 14px;
  border-bottom: 1px solid var(--border-soft);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  background: var(--bg-card);
  overflow-x: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.tab-bar::-webkit-scrollbar {
  display: none;
}
.tab {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  padding: 13px 10px;
  border: none;
  background: transparent;
  color: var(--text-soft);
  font-size: 14px;
  font-family: inherit;
  line-height: 1.4;
  white-space: nowrap;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease);
}
/* 下划线用伪元素：切 Tab 时只改背景色，不引起布局跳动 */
.tab::after {
  content: '';
  position: absolute;
  right: 8px;
  bottom: -1px;
  left: 8px;
  height: 2px;
  border-radius: 2px;
  background: transparent;
  transition: background-color var(--dur-fast) var(--ease);
}
.tab:hover {
  color: var(--primary);
}
.tab.is-active {
  color: var(--primary-deep);
  font-weight: 600;
}
.tab.is-active::after {
  background: var(--primary);
}
.tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 17px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
.tab.is-active .tab-count {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.panel-body {
  padding: 4px 18px 18px;
}

/* ---------- 子分类（点赞 / 收藏） ---------- */
.sub-tab-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 12px 0 4px;
}
.sub-tab {
  padding: 4px 12px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  font-size: 12px;
  font-family: inherit;
  line-height: 1.5;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease),
    background-color var(--dur-fast) var(--ease);
}
.sub-tab:hover {
  color: var(--primary);
  border-color: var(--primary);
}
.sub-tab.is-active {
  border-color: transparent;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-weight: 600;
}

/* ---------- 通用列表行（粉丝 / 关注 / 点赞 / 收藏 / 可能认识的人） ---------- */
.row-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 6px 0 0;
  padding: 0;
  list-style: none;
}
.row-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 8px;
  border-radius: var(--radius);
  transition: background-color var(--dur-fast) var(--ease);
}
.row-item:hover {
  background: var(--bg-muted);
}
.row-item.is-clickable {
  cursor: pointer;
}
.row-link {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--text);
  text-decoration: none;
}
.row-avatar {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border: 1px solid var(--border-soft);
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.row-info {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.row-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-link:hover .row-name {
  color: var(--primary);
}
.row-sub {
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
/* 互关 / 已关注：互关用主色，单向关注弱化 */
.row-tag {
  flex-shrink: 0;
  padding: 0 6px;
  border: 1px solid var(--primary-soft);
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 10px;
  font-weight: 600;
  line-height: 16px;
}
.row-tag.is-weak {
  border-color: var(--border-soft);
  background: var(--bg-muted);
  color: var(--text-muted);
}
.row-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
  font-variant-numeric: tabular-nums;
}
/* 行尾：关注时间 + 关注按钮 */
.row-tail {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}

/* ---------- 粉丝批量管理 ---------- */
.fans-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 12px 0 2px;
  padding: 8px 10px;
  border: 1px dashed var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-muted);
  color: var(--text-muted);
  font-size: 12px;
}
.fans-bar-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fans-bar-actions {
  display: flex;
  flex-shrink: 0;
  gap: 6px;
}
.fans-check {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  color: var(--text-soft);
  cursor: pointer;
}
.fans-check input,
.row-check {
  margin: 0;
  accent-color: var(--primary);
  cursor: pointer;
}
.row-check {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
/* 管理模式下整行可点勾选，给个手型提示 */
.row-list.is-managing .row-item {
  cursor: pointer;
}

/* ---------- 点赞 / 收藏行 ---------- */
.like-badge {
  flex-shrink: 0;
  padding: 1px 7px;
  border-radius: var(--radius-xs);
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 11px;
  font-weight: 600;
  line-height: 1.6;
}
.badge-comment {
  background: rgba(46, 122, 92, 0.14);
  color: var(--success);
}
.badge-moment {
  background: var(--gold-wash);
  color: var(--warning);
}
.badge-other {
  background: var(--bg-muted);
  color: var(--text-muted);
}
.like-title {
  flex: 1;
  min-width: 0;
  color: var(--text);
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-item.is-clickable:hover .like-title {
  color: var(--primary);
}

/* ---------- 加载 / 空态 / 更多 ---------- */
.is-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 0;
  font-size: 13px;
  color: var(--text-muted);
}
.empty-row {
  padding: 44px 0;
  text-align: center;
}
.load-more {
  display: flex;
  justify-content: center;
  padding: 18px 0 4px;
}
.spinner-sm {
  width: 12px;
  height: 12px;
  border-width: 2px;
}
.list-end {
  margin: 18px 0 4px;
  text-align: center;
  font-size: 11px;
  color: var(--text-light);
}

/* ---------- 可能认识的人 ---------- */
.suggest-panel {
  padding: 16px 18px 10px;
}
.suggest-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}
.suggest-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.suggest-title .bi {
  font-size: 14px;
  color: var(--primary);
}
.suggest-tip {
  font-size: 11px;
  color: var(--text-muted);
  white-space: nowrap;
}
/* 横向轨道：只横向滚动，滚动条隐藏；卡片对齐用 scroll-snap */
.suggest-scroller {
  display: flex;
  gap: 10px;
  margin-top: 10px;
  padding: 2px 0 4px;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x proximity;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.suggest-scroller::-webkit-scrollbar {
  display: none;
}
.suggest-card {
  display: flex;
  flex: 0 0 132px;
  /* min-width:0 才能让 flex 项真的锁死在 132px——
     否则超长昵称的 min-content 会把卡片撑宽，横滑轨道里就会出现宽窄不一 */
  min-width: 0;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 14px 10px 12px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  background: var(--bg-muted);
  scroll-snap-align: start;
  text-align: center;
  transition: border-color var(--dur-fast) var(--ease), transform var(--dur-fast) var(--ease);
}
.suggest-card:hover {
  border-color: var(--accent-ring);
  transform: translateY(-2px);
}
.suggest-card-link {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
  color: inherit;
  text-decoration: none;
}
.suggest-avatar {
  width: 48px;
  height: 48px;
  border: 1px solid var(--border-soft);
  border-radius: 50%;
  background: var(--bg-card);
  object-fit: cover;
}
.suggest-name {
  display: block;
  width: 100%;
  color: var(--text);
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.suggest-card-link:hover .suggest-name {
  color: var(--primary);
}
/* 两行截断 + 固定高度：卡片无论如何都对齐，关注按钮不会高低错落 */
.suggest-sub {
  display: -webkit-box;
  min-height: 30px;
  margin-bottom: 2px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.4;
  overflow: hidden;
  word-break: break-word;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
}
.suggest-follow {
  width: 100%;
}
/* 轨道容器：给左右箭头做定位基准 */
.suggest-track {
  position: relative;
}
/* 左右箭头：桌面端点击翻页（手机端隐藏，用手指滑） */
.suggest-nav {
  position: absolute;
  top: 50%;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  margin-top: -15px;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--bg-card);
  box-shadow: var(--shadow-sm);
  color: var(--text-soft);
  font-size: 13px;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease),
    opacity var(--dur-fast) var(--ease);
}
.suggest-nav.is-prev {
  left: -10px;
}
.suggest-nav.is-next {
  right: -10px;
}
.suggest-nav:hover:not(:disabled) {
  border-color: var(--primary);
  color: var(--primary);
}
.suggest-nav:disabled {
  opacity: 0.32;
  cursor: default;
}

/* ---------- 头衔配色（与 Profile 页 10 个头衔一致） ---------- */
.title-zhangmen { background: linear-gradient(135deg, #c8a04a, #b07d2e); }
.title-zhanglao { background: linear-gradient(135deg, #8e6f3e, #6c4f24); }
.title-hufa { background: linear-gradient(135deg, #c0392b, #a93226); }
.title-neimen { background: linear-gradient(135deg, #2980b9, #1f618d); }
.title-waimen { background: linear-gradient(135deg, #16a085, #117a65); }
.title-lianqi { background: linear-gradient(135deg, #27ae60, #1e8449); }
.title-zhuji { background: linear-gradient(135deg, #7cb342, #558b2f); }
.title-jiedan { background: linear-gradient(135deg, #e67e22, #ca6f1e); }
.title-yuanying { background: linear-gradient(135deg, #6c5ce7, #5a3fd4); }
.title-huashen { background: linear-gradient(135deg, #f6d365, #fda085); text-shadow: 0 1px 2px rgba(0, 0, 0, 0.25); }
.title-xiake { background: linear-gradient(135deg, #4a9e6f, #2f7d52); }
.title-xuetu { background: linear-gradient(135deg, #9aa0a6, #6c757d); }
.title-default { background: #6c757d; }

/* ========== 平板 ========== */
@media (max-width: 900px) {
  .hero-cover {
    height: 132px;
  }
  .hero-stats {
    margin-top: 14px;
  }
}

/* ========== 手机端 ========== */
@media (max-width: 640px) {
  .author-page {
    gap: 12px;
  }
  .hero-cover {
    height: 104px;
  }
  .hero-cover-edit {
    right: 10px;
    bottom: 10px;
    height: 26px;
    padding: 0 8px;
    font-size: 11px;
  }
  .hero-body {
    padding: 0 14px;
  }
  .hero-head {
    gap: 10px;
  }
  .hero-avatar {
    margin-top: -30px;
    order: 1;
  }
  .hero-ident {
    order: 2;
    flex: 1 1 100%;
    padding-bottom: 0;
    margin-top: 4px;
  }
  .hero-actions {
    order: 3;
    width: 100%;
    padding-bottom: 0;
  }
  .hero-actions .btn {
    flex: 1;
  }
  .hero-name {
    font-size: 19px;
  }
  /* ID 行：手机端才显示 */
  .hero-id {
    display: block;
    margin-top: 2px;
    font-size: 11px;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }
  .hero-desc {
    margin-top: 10px;
    font-size: 12.5px;
  }
  .hero-chip-desc {
    max-width: 160px;
  }
  .hero-stats {
    margin: 14px -14px 0;
    padding: 8px 8px 10px;
  }
  .hero-stat strong {
    font-size: 15px;
  }
  .hero-stat span {
    font-size: 10.5px;
  }
  .hero-ban {
    flex-wrap: wrap;
    margin-bottom: 14px;
  }
  .hero-ban .btn {
    width: 100%;
  }
  /* 手机端顶栏已吸顶，Tab 栏不再吸顶，避免两层固定元素叠在一起 */
  .tab-bar {
    position: static;
    padding: 0 8px;
  }
  .tab {
    padding: 11px 9px;
    font-size: 13px;
  }
  .panel-body {
    padding: 4px 12px 14px;
  }
  .row-item {
    padding: 9px 6px;
  }
  /* 手机端行尾只留关注按钮：时间挤在一起没有阅读价值 */
  .row-tail .row-time {
    display: none;
  }
  .suggest-panel {
    padding: 14px 14px 8px;
  }
  .suggest-card {
    flex-basis: 118px;
    padding: 12px 8px 10px;
  }
  /* 手机端用原生滑动，箭头既占地方又难点中 */
  .suggest-nav {
    display: none;
  }
  .fans-bar {
    flex-wrap: wrap;
    gap: 8px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .hero-cover-edit,
  .hero-chip-link,
  .hero-badges-toggle,
  .hero-stat,
  .tab,
  .tab::after,
  .sub-tab,
  .row-item {
    transition: none;
  }
}
</style>
