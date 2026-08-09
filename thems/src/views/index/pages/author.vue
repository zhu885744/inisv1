<template>
  <div class="author-page mt-2">
    <!-- 加载状态 -->
    <div v-if="loading" class="card">
      <div class="card-body text-center py-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">加载中...</span>
        </div>
        <p class="mt-3 text-muted">加载用户信息中...</p>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="card">
      <div class="card-body text-center py-5">
        <i class="bi bi-exclamation-circle text-danger fs-1"></i>
        <p class="mt-3 text-muted">{{ error }}</p>
        <button @click="loadUserInfo" class="btn btn-sm btn-outline-secondary">
          重试
        </button>
      </div>
    </div>

    <!-- 用户主页内容 -->
    <template v-else-if="userInfo">
      <!-- 用户信息卡片 -->
      <div class="card author-profile-card mb-4">
        <div class="card-body p-4">
          <!-- 顶部：头像 + 基本信息 -->
          <div class="profile-header">
            <div class="profile-avatar">
              <i-avatar-frame
                :src="userInfo.avatar || defaultAvatar"
                :frame="userInfo.json?.frame || ''"
                :alt="userInfo.nickname || '用户头像'"
                size="100px"
                :frame-scale="1.5"
                :rounded="false"
              />
              <div
                v-if="isOnline"
                class="avatar-online-dot"
                title="在线"
              ></div>
            </div>
            <div class="profile-info">
              <div class="profile-name-row">
                <h3 class="profile-name">{{ userInfo.nickname || userInfo.account || '未知用户' }}</h3>
                <span class="profile-uid">UID:{{ userInfo.id }}</span>
                <span class="profile-role">Lv{{ userLevel }} {{ levelName }}</span>
              </div>
              <p v-if="userInfo.description" class="profile-desc">
                {{ userInfo.description }}
              </p>
              <p v-else class="profile-desc profile-desc-placeholder">
                这家伙很懒，什么都没有写...
              </p>
              <a
                v-if="userWebsite?.website?.url"
                :href="userWebsite.website.url"
                target="_blank"
                class="profile-website"
              >
                <i class="bi bi-globe"></i> {{ userWebsite.website.name || userWebsite.website.url }}
              </a>
              <p v-else class="profile-website-placeholder">&nbsp;</p>
            </div>
          </div>

          <!-- 信息标签行 -->
          <div class="profile-tags-row">
            <span
              v-if="userInfo.gender === 'boy' || userInfo.gender === 1"
              class="badge bg-primary text-white rounded-pill"
            >
              <i class="bi bi-gender-male"></i> 男
            </span>
            <span
              v-else-if="userInfo.gender === 'girl' || userInfo.gender === 2"
              class="badge bg-danger text-white rounded-pill"
            >
              <i class="bi bi-gender-female"></i> 女
            </span>
            <span v-if="userInfo.title || userInfo.json?.title" class="badge rounded-pill title-badge" :class="getTitleColorClass(userInfo.json?.title || userInfo.title)">
              <i class="bi bi-person-badge"></i> {{ userInfo.json?.title || userInfo.title }}
            </span>
            <!-- 封禁状态标签 -->
            <span v-if="banInfo.isBanned" class="badge rounded-pill bg-danger" style="cursor: pointer;" title="点击查看封禁详情" @click="showBanDetail = !showBanDetail">
              <i class="bi bi-slash-circle"></i> 小黑屋 · 第{{ banInfo.violationNum }}次
              <span v-if="banInfo.remaining"> · 剩余{{ banInfo.remaining }}</span>
            </span>
            <span class="badge bg-light text-dark rounded-pill">
              <i class="bi bi-calendar3"></i> 注册于 {{ formatDate(userInfo.create_time) }}
            </span>
          </div>

          <!-- 封禁详情卡片 -->
          <div v-if="showBanDetail && banInfo.isBanned" class="ban-detail-card mt-3 p-3 rounded border border-warning">
            <div class="d-flex justify-content-between align-items-start">
              <div>
                <h6 class="text-warning mb-2"><i class="bi bi-exclamation-triangle-fill me-1"></i>封禁详情</h6>
                <div class="row g-2 small">
                  <div class="col-sm-6">
                    <span class="text-muted">封禁原因：</span>
                    <span>{{ banInfo.reason }}</span>
                  </div>
                  <div class="col-sm-6">
                    <span class="text-muted">封禁时长：</span>
                    <span v-if="banInfo.duration === 0 && banInfo.violationNum >= 5" class="text-danger fw-bold">永久封禁</span>
                    <span v-else-if="banInfo.duration === 0" class="text-danger">永久</span>
                    <span v-else>{{ banInfo.duration }} 天</span>
                  </div>
                  <div class="col-sm-6">
                    <span class="text-muted">限制类型：</span>
                    <span v-for="bt in banInfo.banTypes" :key="bt.bit" class="badge bg-secondary me-1">{{ bt.name }}</span>
                    <span v-if="!banInfo.banTypes.length" class="badge bg-danger">全部限制</span>
                  </div>
                  <div class="col-sm-6" v-if="banInfo.expiresAt && banInfo.duration > 0">
                    <span class="text-muted">解封时间：</span>
                    <span>{{ formatDate(banInfo.expiresAt) }}</span>
                  </div>
                </div>
              </div>
              <button class="btn-close" @click="showBanDetail = false" aria-label="Close"></button>
            </div>
            <!-- 申诉按钮（仅本人才显示） -->
            <button v-if="isSelf && banInfo.violationNum < 5" class="btn btn-warning btn-sm mt-3 w-100" @click="appealDialog?.show()">
              <i class="bi bi-journal-text me-1"></i> 提交申诉
            </button>
          </div>

          <!-- 封禁申诉弹窗 -->
          <DialogAppeal ref="appealDialog" />

          <!-- 经验条 -->
          <div class="exp-section">
            <div class="exp-header">
              <span class="exp-label">{{ levelName }}</span>
              <span class="exp-info">
                <span class="exp-current">{{ currentExp }}</span>
                <span class="exp-sep">/</span>
                <span class="exp-max">{{ nextLevelExp }}</span>
                <span class="exp-unit">经验</span>
                <span class="exp-next" v-if="nextLevelName">→ {{ nextLevelName }}</span>
                <span class="exp-next" v-else>已达最高境界</span>
              </span>
            </div>
            <div class="exp-progress">
              <div
                class="exp-bar"
                :style="{ width: expPercent + '%' }"
              ></div>
            </div>
            <p class="exp-desc">{{ levelDesc }}</p>
            <button
              class="level-view-btn"
              @click="showLevelModal = true"
            >
              <i class="bi bi-eye"></i> 查看等级体系
            </button>
          </div>

          <!-- 统计数据 -->
          <div class="profile-stats">
            <div class="stat-card" :class="{ active: activeTab === 'articles' }" @click="switchTab('articles')">
              <div class="stat-num">{{ articleCount }}</div>
              <div class="stat-text">文章</div>
            </div>
            <div class="stat-card" :class="{ active: activeTab === 'followers' }" @click="switchTab('followers')">
              <div class="stat-num">{{ followersCount }}</div>
              <div class="stat-text">粉丝</div>
            </div>
            <div class="stat-card" :class="{ active: activeTab === 'following' }" @click="switchTab('following')">
              <div class="stat-num">{{ followingCount }}</div>
              <div class="stat-text">关注</div>
            </div>
            <div class="stat-card" :class="{ active: activeTab === 'likes' }" @click="switchTab('likes')">
              <div class="stat-num">{{ likeCount }}</div>
              <div class="stat-text">点赞</div>
            </div>
            <div class="stat-card" :class="{ active: activeTab === 'collects' }" @click="switchTab('collects')">
              <div class="stat-num">{{ collectCount }}</div>
              <div class="stat-text">收藏</div>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="profile-actions">
            <template v-if="!isSelf">
              <button
                v-if="!isFollowed"
                @click="toggleFollow"
                class="btn btn-primary btn-sm action-btn"
                :disabled="following"
              >
                <i class="bi bi-person-plus"></i> {{ following ? '处理中...' : '+ 关注' }}
              </button>
              <button
                v-else
                @click="toggleFollow"
                class="btn btn-outline-secondary btn-sm action-btn"
                :disabled="following"
              >
                <i class="bi bi-person-dash"></i> {{ following ? '处理中...' : '已关注' }}
              </button>
            </template>
            <template v-else>
              <button
                @click="goToEdit"
                class="btn btn-primary btn-sm action-btn"
              >
                <i class="bi bi-pencil-square"></i> 编辑资料
              </button>
            </template>
          </div>
        </div>
      </div>

      <!-- 内容区：按 tab 切换 -->
      <!-- 文章列表 -->
      <template v-if="activeTab === 'articles'">
      <div class="card shadow-sm border-0">
        <div class="card-header d-flex justify-content-between align-items-center bg-body-tertiary">
          <h5 class="card-title mb-0">
            <i class="bi bi-collection"></i> 发布的文章
          </h5>
          <span class="badge rounded-pill text-bg-primary">{{ articleCount }} 篇</span>
        </div>

        <!-- 文章加载中 -->
        <div v-if="articlesLoading" class="card-body text-center py-5">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">加载中...</span>
          </div>
          <p class="mt-3 text-muted small">正在加载文章...</p>
        </div>

        <!-- 文章列表 -->
        <template v-else-if="articleList.length > 0">
          <div class="card-body p-0">
            <div
              v-for="(article, index) in articleList"
              :key="article.id"
              class="author-article-item"
              :class="{ 'border-top': index > 0 }"
              @click="goToArticle(article.id)"
            >
              <div class="article-inner">
                <div class="article-main">
                  <h6 class="article-title">{{ article.title }}</h6>
                  <p v-if="article.abstract" class="article-desc">
                    {{ truncateText(article.abstract, 120) }}
                  </p>
                  <div class="article-footer">
                    <div class="article-meta">
                      <span class="meta-chip">
                        <i class="bi bi-eye-fill"></i>
                        {{ formatNumber(article.views) }}
                      </span>
                      <span class="meta-chip">
                        <i class="bi bi-clock-fill"></i>
                        {{ formatRelativeTime(article.create_time) }}
                      </span>
                    </div>
                  </div>
                </div>
                <div class="article-actions">
                  <button
                    v-if="isSelf"
                    class="btn btn-sm btn-outline-primary me-2"
                    @click.stop="editArticle(article.id)"
                  >
                    <i class="bi bi-pencil-square"></i>
                  </button>
                  <button
                    class="btn btn-sm btn-outline-secondary"
                    @click.stop="goToArticle(article.id)"
                  >
                    <i class="bi bi-arrow-right"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 分页 -->
          <div v-if="totalPages > 1" class="card-body text-center py-3 border-top">
            <nav aria-label="文章分页">
              <ul class="pagination pagination-sm justify-content-center mb-0">
                <li class="page-item" :class="{ disabled: currentPage <= 1 }">
                  <a class="page-link" href="#" @click.prevent="changePage(currentPage - 1)">
                    <i class="bi bi-chevron-left"></i>
                  </a>
                </li>
                <li
                  v-for="page in visiblePages"
                  :key="page"
                  class="page-item"
                  :class="{ active: page === currentPage, disabled: page === '...' }"
                >
                  <a class="page-link" href="#" @click.prevent="page !== '...' && changePage(page)">
                    {{ page }}
                  </a>
                </li>
                <li class="page-item" :class="{ disabled: currentPage >= totalPages }">
                  <a class="page-link" href="#" @click.prevent="changePage(currentPage + 1)">
                    <i class="bi bi-chevron-right"></i>
                  </a>
                </li>
              </ul>
            </nav>
          </div>
        </template>

        <!-- 无文章 -->
        <div v-else class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-inbox"></i>
            <p class="mt-3 text-muted">该用户还没有发布文章</p>
            <p class="text-muted small">发布第一篇精彩内容吧</p>
          </div>
        </div>
      </div>
      </template>

      <!-- 粉丝列表 -->
      <template v-else-if="activeTab === 'followers'">
      <div class="card shadow-sm border-0">
        <div class="card-header d-flex justify-content-between align-items-center bg-body-tertiary">
          <h5 class="card-title mb-0">
            <i class="bi bi-people-fill"></i> 粉丝
          </h5>
          <span class="badge rounded-pill text-bg-primary">{{ followersCount }} 人</span>
        </div>
        <div v-if="!isSelf" class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-lock"></i>
            <p class="mt-3 text-muted">当前粉丝隐私保护中，不可见！</p>
          </div>
        </div>
        <div v-else-if="followersLoading" class="card-body text-center py-5">
          <div class="spinner-border text-primary" role="status"></div>
          <p class="mt-3 text-muted small">正在加载粉丝列表...</p>
        </div>
        <template v-else-if="followerList.length > 0">
          <div class="card-body p-0">
            <div
              v-for="(item, index) in followerList"
              :key="item.id"
              class="author-social-item"
              :class="{ 'border-top': index > 0 }"
              @click="goToUser(item.uid)"
            >
              <i-avatar-frame
                :src="item.result?.follower?.avatar || defaultAvatar"
                :frame="item.result?.follower?.json?.frame || ''"
                size="48px"
                :frame-scale="1.7"
                class="social-avatar"
              />
              <div class="social-main">
                <div class="social-name">{{ item.result?.follower?.nickname || '未知用户' }}</div>
                <div v-if="item.result?.follower?.description" class="social-desc">
                  {{ truncateText(item.result.follower.description, 60) }}
                </div>
                <div class="social-time">{{ formatRelativeTime(item.create_time) }}</div>
              </div>
              <span class="social-arrow"><i class="bi bi-chevron-right"></i></span>
            </div>
          </div>
          <!-- 粉丝 无限加载底部状态 -->
          <div ref="followersSentinelRef" class="infinite-scroll-sentinel border-top py-4"></div>
          <div v-if="followersMoreLoading" class="text-center py-3 text-muted small">
            <span class="spinner-border spinner-border-sm me-2" role="status"></span>
            加载中...
          </div>
          <div v-else-if="!followersHasMore && followerList.length > 0" class="text-center py-3 text-muted small">
            —— 没有更多粉丝了 ——
          </div>
        </template>
        <div v-else class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-people"></i>
            <p class="mt-3 text-muted">暂无粉丝</p>
          </div>
        </div>
      </div>
      </template>

      <!-- 关注列表 -->
      <template v-else-if="activeTab === 'following'">
      <div class="card shadow-sm border-0">
        <div class="card-header d-flex justify-content-between align-items-center bg-body-tertiary">
          <h5 class="card-title mb-0">
            <i class="bi bi-heart-fill"></i> 关注
          </h5>
          <span class="badge rounded-pill text-bg-primary">{{ followingCount }} 人</span>
        </div>
        <div v-if="!isSelf" class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-lock"></i>
            <p class="mt-3 text-muted">当前关注隐私保护中，不可见！</p>
          </div>
        </div>
        <div v-else-if="followingLoading" class="card-body text-center py-5">
          <div class="spinner-border text-primary" role="status"></div>
          <p class="mt-3 text-muted small">正在加载关注列表...</p>
        </div>
        <template v-else-if="followingList.length > 0">
          <div class="card-body p-0">
            <div
              v-for="(item, index) in followingList"
              :key="item.id"
              class="author-social-item"
              :class="{ 'border-top': index > 0 }"
              @click="goToUser(item.follow_uid)"
            >
              <i-avatar-frame
                :src="item.result?.followee?.avatar || defaultAvatar"
                :frame="item.result?.followee?.json?.frame || ''"
                size="48px"
                :frame-scale="1.7"
                class="social-avatar"
              />
              <div class="social-main">
                <div class="social-name">{{ item.result?.followee?.nickname || '未知用户' }}</div>
                <div v-if="item.result?.followee?.description" class="social-desc">
                  {{ truncateText(item.result.followee.description, 60) }}
                </div>
                <div class="social-time">{{ formatRelativeTime(item.create_time) }}</div>
              </div>
              <span class="social-arrow"><i class="bi bi-chevron-right"></i></span>
            </div>
          </div>
          <!-- 关注 无限加载底部状态 -->
          <div ref="followingSentinelRef" class="infinite-scroll-sentinel border-top py-4"></div>
          <div v-if="followingMoreLoading" class="text-center py-3 text-muted small">
            <span class="spinner-border spinner-border-sm me-2" role="status"></span>
            加载中...
          </div>
          <div v-else-if="!followingHasMore && followingList.length > 0" class="text-center py-3 text-muted small">
            —— 没有更多关注了 ——
          </div>
        </template>
        <div v-else class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-heart"></i>
            <p class="mt-3 text-muted">暂无关注</p>
          </div>
        </div>
      </div>
      </template>

      <!-- 点赞列表 -->
      <template v-else-if="activeTab === 'likes'">
      <div class="card shadow-sm border-0">
        <div class="card-header d-flex justify-content-between align-items-center bg-body-tertiary">
          <h5 class="card-title mb-0">
            <i class="bi bi-hand-thumbs-up-fill"></i> 点赞
          </h5>
          <span class="badge rounded-pill text-bg-primary">{{ likeCount }} 次</span>
        </div>
        <!-- 细分 tab (仅本人可见时显示) -->
        <div v-if="isSelf" class="card-tab-bar">
          <button
            v-for="tab in LIKE_TABS"
            :key="'likes-' + tab.key"
            type="button"
            class="card-tab-item"
            :class="{ active: activeLikesSubTab === tab.key }"
            @click="switchLikesSubTab(tab.key)"
          >{{ tab.label }}</button>
        </div>
        <div v-if="!isSelf" class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-lock"></i>
            <p class="mt-3 text-muted">当前点赞隐私保护中，不可见！</p>
          </div>
        </div>
        <div v-else-if="likesLoading" class="card-body text-center py-5">
          <div class="spinner-border text-primary" role="status"></div>
          <p class="mt-3 text-muted small">正在加载点赞列表...</p>
        </div>
        <template v-else-if="likesList.length > 0">
          <div class="card-body p-0">
            <div
              v-for="(item, index) in likesList"
              :key="item.id"
              class="author-article-item"
              :class="{ 'border-top': index > 0 }"
              @click="goToLikedContent(item)"
            >
              <div class="article-inner">
                <div class="article-main">
                  <span class="badge mb-1" :class="targetTypeBadge(item.target_type)">
                    {{ targetTypeLabel(item.target_type) }}
                  </span>
                  <h6 class="article-title">
                    <template v-if="item._detail?.title">{{ item._detail.title }}</template>
                    <template
                      v-else-if="item.target_type === 'user' && item.result?.author?.nickname"
                      >{{ item.result.author.nickname }}</template
                    >
                    
                  </h6>
                  <p
                    v-if="item._detail?.abstract || item._detail?.description"
                    class="article-desc"
                  >
                    {{
                      truncateText(
                        item._detail.abstract || item._detail.description || '',
                        80
                      )
                    }}
                  </p>
                  <p
                    v-else-if="
                      item.target_type === 'comment' && item._detail?.content
                    "
                    class="article-desc"
                  >
                    <span v-html="truncateAndRenderEmoji(item._detail.content, 80)"></span>
                  </p>
                  <p
                    v-else-if="
                      item.target_type === 'moment' && item._detail?.content
                    "
                    class="article-desc"
                  >
                    <span v-html="truncateAndRenderEmoji(item._detail.content, 80)"></span>
                  </p>
                  <div class="article-footer">
                    <div class="article-meta">
                      <span class="meta-chip">
                        <i class="bi bi-clock-fill"></i>
                        {{ formatRelativeTime(item.create_time) }}
                      </span>
                      <span
                        v-if="item._detail?.views !== undefined"
                        class="meta-chip"
                      >
                        <i class="bi bi-eye-fill"></i>
                        {{ formatNumber(item._detail.views) }}
                      </span>
                    </div>
                  </div>
                </div>
                <div class="article-actions">
                  <button
                    class="btn btn-sm btn-outline-secondary"
                    @click.stop="goToLikedContent(item)"
                  >
                    <i class="bi bi-arrow-right"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>
          <!-- 点赞 无限加载底部状态 -->
          <div ref="likesSentinelRef" class="infinite-scroll-sentinel border-top py-4"></div>
          <div v-if="likesMoreLoading" class="text-center py-3 text-muted small">
            <span class="spinner-border spinner-border-sm me-2" role="status"></span>
            加载中...
          </div>
          <div v-else-if="!likesHasMore && likesList.length > 0" class="text-center py-3 text-muted small">
            —— 没有更多点赞了 ——
          </div>
        </template>
        <div v-else class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-hand-thumbs-up"></i>
            <p class="mt-3 text-muted">暂无点赞记录</p>
          </div>
        </div>
      </div>
      </template>

      <!-- 收藏列表 -->
      <template v-else-if="activeTab === 'collects'">
      <div class="card shadow-sm border-0">
        <div class="card-header d-flex justify-content-between align-items-center bg-body-tertiary">
          <h5 class="card-title mb-0">
            <i class="bi bi-bookmark-heart-fill"></i> 收藏
          </h5>
          <span class="badge rounded-pill text-bg-primary">{{ collectCount }} 个</span>
        </div>
        <!-- 细分 tab (仅本人可见时显示) -->
        <div v-if="isSelf" class="card-tab-bar">
          <button
            v-for="tab in LIKE_TABS"
            :key="'collects-' + tab.key"
            type="button"
            class="card-tab-item"
            :class="{ active: activeCollectsSubTab === tab.key }"
            @click="switchCollectsSubTab(tab.key)"
          >{{ tab.label }}</button>
        </div>
        <div v-if="!isSelf" class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-lock"></i>
            <p class="mt-3 text-muted">当前收藏隐私保护中，不可见！</p>
          </div>
        </div>
        <div v-else-if="collectsLoading" class="card-body text-center py-5">
          <div class="spinner-border text-primary" role="status"></div>
          <p class="mt-3 text-muted small">正在加载收藏列表...</p>
        </div>
        <template v-else-if="collectsList.length > 0">
          <div class="card-body p-0">
            <div
              v-for="(item, index) in collectsList"
              :key="item.id"
              class="author-article-item"
              :class="{ 'border-top': index > 0 }"
              @click="goToCollectedContent(item)"
            >
              <div class="article-inner">
                <div class="article-main">
                  <span class="badge mb-1" :class="targetTypeBadge(item.target_type)">
                    {{ targetTypeLabel(item.target_type) }}
                  </span>
                  <h6 class="article-title">
                    <template v-if="item._detail?.title">{{ item._detail.title }}</template>
                    <template
                      v-else-if="item.target_type === 'user' && item.result?.author?.nickname"
                      >{{ item.result.author.nickname }}</template
                    >
                  </h6>
                  <p
                    v-if="item._detail?.abstract || item._detail?.description"
                    class="article-desc"
                  >
                    {{
                      truncateText(
                        item._detail.abstract || item._detail.description || '',
                        80
                      )
                    }}
                  </p>
                  <p
                    v-else-if="
                      item.target_type === 'comment' && item._detail?.content
                    "
                    class="article-desc"
                  >
                    <span v-html="truncateAndRenderEmoji(item._detail.content, 80)"></span>
                  </p>
                  <p
                    v-else-if="
                      item.target_type === 'moment' && item._detail?.content
                    "
                    class="article-desc"
                  >
                    <span v-html="truncateAndRenderEmoji(item._detail.content, 80)"></span>
                  </p>
                  <div class="article-footer">
                    <div class="article-meta">
                      <span class="meta-chip">
                        <i class="bi bi-clock-fill"></i>
                        {{ formatRelativeTime(item.create_time) }}
                      </span>
                      <span
                        v-if="item._detail?.views !== undefined"
                        class="meta-chip"
                      >
                        <i class="bi bi-eye-fill"></i>
                        {{ formatNumber(item._detail.views) }}
                      </span>
                    </div>
                  </div>
                </div>
                <div class="article-actions">
                  <button
                    class="btn btn-sm btn-outline-secondary"
                    @click.stop="goToCollectedContent(item)"
                  >
                    <i class="bi bi-arrow-right"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>
          <!-- 收藏 无限加载底部状态 -->
          <div ref="collectsSentinelRef" class="infinite-scroll-sentinel border-top py-4"></div>
          <div v-if="collectsMoreLoading" class="text-center py-3 text-muted small">
            <span class="spinner-border spinner-border-sm me-2" role="status"></span>
            加载中...
          </div>
          <div v-else-if="!collectsHasMore && collectsList.length > 0" class="text-center py-3 text-muted small">
            —— 没有更多收藏了 ——
          </div>
        </template>
        <div v-else class="card-body text-center py-5">
          <div class="empty-state">
            <i class="bi bi-bookmark-heart"></i>
            <p class="mt-3 text-muted">暂无收藏记录</p>
          </div>
        </div>
      </div>
      </template>
    </template>

    <!-- 等级弹窗 -->
    <Teleport to="body">
      <div v-if="showLevelModal" class="level-modal-overlay" @click.self="showLevelModal = false">
        <div class="level-modal-content">
          <LevelDisplay
            :current-level="levelCurrent"
            :current-exp="currentExp"
            :next-level-exp="nextLevelExp"
            @close="showLevelModal = false"
          />
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { request, cache } from '@/utils/network'
import LevelDisplay from '@/comps/custom/LevelDisplay.vue'
import iAvatarFrame from '@/comps/custom/i-avatar-frame.vue'
import DialogAppeal from '@/comps/index/dialog/appeal.vue'
import { useCommStore } from '@/store/comm'
import { usePageTitle, toast, getTitleColorClass } from '@/utils/app'
import utils from '@/utils/utils'
import defaultAvatar from '@/assets/img/avatar.png'

const route = useRoute()
const router = useRouter()
const store = useCommStore()

const authorId = computed(() => route.params.id)
const { setDynamicTitle } = usePageTitle({
  staticTitle: '用户主页',
  defaultTitle: '用户主页'
})

// 响应式数据
const loading = ref(true)
const error = ref('')
const userInfo = ref(null)
const articleList = ref([])
const articleCount = ref(0)
const likeCount = ref(0)
const collectCount = ref(0)
const articlesLoading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const isFollowed = ref(false)
const following = ref(false)
const followingCount = ref(0)
const followersCount = ref(0)
const showLevelModal = ref(false)
const activeTab = ref('articles')

const followerList = ref([])
const followingList = ref([])
const likesList = ref([])
const collectsList = ref([])
const followersLoading = ref(false)
const followingLoading = ref(false)
const likesLoading = ref(false)
const collectsLoading = ref(false)
// 「加载更多」专用状态（与首次加载 Loading 分离，避免头部 spinner 反复闪烁）
const followersMoreLoading = ref(false)
const followingMoreLoading = ref(false)
const likesMoreLoading = ref(false)
const collectsMoreLoading = ref(false)
// 是否还有更多数据（false 时 IntersectionObserver 自动停手，不再触发请求）
const followersHasMore = ref(true)
const followingHasMore = ref(true)
const likesHasMore = ref(true)
const collectsHasMore = ref(true)
// 四个哨兵 DOM 引用（对应模板里 ref="xxxSentinelRef" 的 div）
const followersSentinelRef = ref(null)
const followingSentinelRef = ref(null)
const likesSentinelRef = ref(null)
const collectsSentinelRef = ref(null)
let scrollObserver = null

// 粉丝 / 关注 分页状态
const followersPage = ref(1)
const followingPage = ref(1)
const followersTotal = ref(0)
const followingTotal = ref(0)
const followersPageSize = 20
const followingPageSize = 20

// 点赞 / 收藏 分页 & sub-tab
const likesPage = ref(1)
const collectsPage = ref(1)
const likesTotal = ref(0)
const collectsTotal = ref(0)
const likesPageSize = 10
const collectsPageSize = 10
const activeLikesSubTab = ref('article')
const activeCollectsSubTab = ref('article')

// 点赞/收藏 细分 tab（文章 / 评论 / 动态 ）
const LIKE_TABS = [
  { key: 'article', label: '文章' },
  { key: 'comment', label: '评论' },
  { key: 'moment', label: '动态' },
]

// 当前登录用户
const currentUser = computed(() => store.currentUser)
const isSelf = computed(() => {
  return currentUser.value && String(currentUser.value.id) === String(authorId.value)
})

// 在线状态判断
const isOnline = computed(() => {
  if (!userInfo.value?.login_time) return false
  const loginTime = userInfo.value.login_time
  const now = Math.floor(Date.now() / 1000)
  return now - loginTime < 86400
})

const levelCurrent = computed(() => userInfo.value?.result?.level?.current || {})
const levelNext = computed(() => userInfo.value?.result?.level?.next || {})

const userLevel = computed(() => levelCurrent.value?.value || 1)
const levelName = computed(() => levelCurrent.value?.name || '凡人')
const levelDesc = computed(() => levelCurrent.value?.description || '')
const currentExp = computed(() => userInfo.value?.exp || 0)
const nextLevelName = computed(() => levelNext.value?.name || '')
const nextLevelExp = computed(() => levelNext.value?.exp || currentExp.value)

const expPercent = computed(() => {
  const current = currentExp.value
  const base = levelCurrent.value?.exp || 0
  const target = levelNext.value?.exp || base
  if (target <= base) return 100
  return Math.min(100, Math.max(0, Math.round(((current - base) / (target - base)) * 100)))
})

const userWebsite = computed(() => {
  const jsonData = userInfo.value?.json
  if (!jsonData) return null
  if (typeof jsonData === 'string') {
    try {
      return JSON.parse(jsonData)
    } catch {
      return null
    }
  }
  return jsonData
})

// 封禁状态
const showBanDetail = ref(false)
const appealDialog = ref(null)
const banTypeMap = [
  { bit: 1,  name: '限制登录' },
  { bit: 2,  name: '限制发文' },
  { bit: 4,  name: '限制评论' },
  { bit: 8,  name: '限制上传' },
  { bit: 16, name: '限制互动' },
]
const banInfo = computed(() => {
  const ban = userInfo.value?.result?.ban
  if (!ban || !ban.is_banned) {
    return { isBanned: false, reason: '', duration: 0, violationNum: 0, expiresAt: 0, remaining: '', banTypes: [] }
  }
  const record = ban.record || {}
  const expiresAt = record.expires_at || 0
  let remaining = ''
  if (record.duration > 0 && expiresAt > 0) {
    const secondsLeft = expiresAt - Math.floor(Date.now() / 1000)
    if (secondsLeft <= 0) {
      remaining = '已到期'
    } else {
      const days = Math.floor(secondsLeft / 86400)
      const hours = Math.floor((secondsLeft % 86400) / 3600)
      remaining = days > 0 ? `${days}天${hours}小时` : `${hours}小时内`
    }
  }
  const banTypes = (() => {
    const val = record.ban_type
    if (!val || val === 0 || val === 31) return []
    return banTypeMap.filter(t => val & t.bit)
  })()
  return {
    isBanned: true,
    reason: record.reason || '违反社区规定',
    duration: record.duration || 0,
    violationNum: record.violation_num || 1,
    expiresAt,
    remaining,
    banTypes,
  }
})

const switchTab = (tab) => {
  activeTab.value = tab
  if (tab === 'articles') {
    loadArticles()
  } else if (tab === 'followers') {
    followersPage.value = 1
    followerList.value = []
    followersHasMore.value = true
    if (isSelf.value) loadFollowersList()
  } else if (tab === 'following') {
    followingPage.value = 1
    followingList.value = []
    followingHasMore.value = true
    if (isSelf.value) loadFollowingList()
  } else if (tab === 'likes') {
    likesPage.value = 1
    likesList.value = []
    likesHasMore.value = true
    activeLikesSubTab.value = 'article'
    if (isSelf.value) loadLikesList()
  } else if (tab === 'collects') {
    collectsPage.value = 1
    collectsList.value = []
    collectsHasMore.value = true
    activeCollectsSubTab.value = 'article'
    if (isSelf.value) loadCollectsList()
  }
  // tab 切换后哨兵 DOM 可能刚渲染，nextTick 里重新 observe
  nextTick(() => bindInfiniteScrollObserver())
}

const goToUser = (userId) => {
  if (!userId || String(userId) === String(authorId.value)) return
  router.push(`/author/${userId}`)
}

const goToLikedContent = (item) => {
  const type = item.target_type
  const id = item.target_id
  if (!id) return
  const detail = item._detail
  if (type === 'article') {
    router.push(`/archives/${id}`)
  } else if (type === 'page') {
    router.push(`/page/${id}`)
  } else if (type === 'moment') {
    router.push(`/moments/${id}`)
  } else if (type === 'comment') {
    const bindType = detail?.bind_type
    const bindId = detail?.bind_id
    if (bindType === 'article' && bindId) {
      router.push(`/archives/${bindId}`)
    } else if (bindType === 'page' && bindId) {
      router.push(`/page/${bindId}`)
    } else if (bindType === 'moments' && bindId) {
      router.push(`/moments/${bindId}`)
    } else {
      router.push(`/moments/${id}`)
    }
  } else if (type === 'user') {
    router.push(`/author/${id}`)
  }
}

const goToCollectedContent = (item) => {
  const type = item.target_type
  const id = item.target_id
  if (!id) return
  const detail = item._detail
  if (type === 'article') {
    router.push(`/archives/${id}`)
  } else if (type === 'page') {
    router.push(`/page/${id}`)
  } else if (type === 'moment') {
    router.push(`/moments/${id}`)
  } else if (type === 'comment') {
    const bindType = detail?.bind_type
    const bindId = detail?.bind_id
    if (bindType === 'article' && bindId) {
      router.push(`/archives/${bindId}`)
    } else if (bindType === 'page' && bindId) {
      router.push(`/page/${bindId}`)
    } else if (bindType === 'moments' && bindId) {
      router.push(`/moments/${bindId}`)
    } else {
      router.push(`/moments/${id}`)
    }
  } else if (type === 'user') {
    router.push(`/author/${id}`)
  }
}

// 滚动到文章列表
const scrollToArticles = () => {
  const el = document.querySelector('.author-articles-section')
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// 分页相关
const totalPages = computed(() => Math.ceil(articleCount.value / pageSize.value) || 1)

const visiblePages = computed(() => {
  const total = totalPages.value
  const current = currentPage.value
  const pages = []

  if (total <= 7) {
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    if (current <= 4) {
      for (let i = 1; i <= 5; i++) pages.push(i)
      pages.push('...')
      pages.push(total)
    } else if (current >= total - 3) {
      pages.push(1)
      pages.push('...')
      for (let i = total - 4; i <= total; i++) pages.push(i)
    } else {
      pages.push(1)
      pages.push('...')
      for (let i = current - 1; i <= current + 1; i++) pages.push(i)
      pages.push('...')
      pages.push(total)
    }
  }

  return pages
})

const formatDate = (timestamp) => {
  if (!timestamp) return '未知'
  return utils.time.to.date(timestamp, 'Y-M-D')
}

const formatRelativeTime = (timestamp) => {
  if (!timestamp) return ''
  return utils.time.nature(timestamp, 5)
}

const formatNumber = (num) => {
  if (num === null || num === undefined || isNaN(num)) return '0'
  return utils.format.number(num)
}

const truncateText = (text, maxLength = 100) => {
  if (!text) return ''
  return text.length <= maxLength ? text : text.substring(0, maxLength) + '...'
}

// 表情渲染：将 [emoji:url] 转换为 <img> 标签
const renderEmoji = (text) => {
  if (!text) return ''
  let processed = String(text)
  // 解析表情图片 [emoji:url]
  processed = processed.replace(
    /\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g,
    '<img src="$1" class="inline-emoji" style="width: 24px; height: 24px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 1px;" loading="lazy">'
  )
  return processed
}

// 带表情渲染的文本截断：先截断再渲染表情，避免截断到表情标签中间
const truncateAndRenderEmoji = (text, maxLength = 80) => {
  if (!text) return ''
  // 先按纯文本截断（保留 [...] 表情占位符完整）
  const truncated = truncateTextPreserveEmoji(text, maxLength)
  // 再渲染表情
  return renderEmoji(truncated)
}

// 截断文本时，确保不会把 [emoji:url] 从中间切断
const truncateTextPreserveEmoji = (text, maxLength = 100) => {
  if (!text) return ''
  const str = String(text)
  if (str.length <= maxLength) return str

  // 找到所有 [emoji:...] 的位置
  const emojiRegex = /\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g
  const emojiPositions = []
  let match
  while ((match = emojiRegex.exec(str)) !== null) {
    emojiPositions.push({
      start: match.index,
      end: match.index + match[0].length
    })
  }

  // 如果 maxLength 落在某个表情标签内，则扩展截断点到该表情结束
  let cutPoint = maxLength
  for (const pos of emojiPositions) {
    if (cutPoint > pos.start && cutPoint < pos.end) {
      cutPoint = pos.end
      break
    }
  }

  return str.substring(0, cutPoint) + '...'
}

const parseTags = (tagsStr) => {
  if (!tagsStr) return []
  try {
    const tags = JSON.parse(tagsStr)
    if (Array.isArray(tags)) return tags
    return []
  } catch {
    return tagsStr.split('|').filter(Boolean)
  }
}

// 事件处理
const handleAvatarError = (event) => {
  event.target.src = defaultAvatar
}

const goToArticle = (id) => {
  router.push(`/archives/${id}`)
}

const editArticle = (id) => {
  router.push(`/article-write/${id}`)
}

const goToEdit = () => {
  router.push('/user')
}

const toggleFollow = async () => {
  if (!store.isLoggedIn) {
    store.switchAuth('login', true)
    return
  }
  if (isSelf.value) return
  
  following.value = true
  const currentUid = currentUser.value?.id
  try {
    if (isFollowed.value) {
      const res = await request.put('/api/user-follows/unfollow', {
        uid: currentUid,
        follow_uid: authorId.value
      })
      if (res.code === 200) {
        isFollowed.value = false
        followersCount.value = Math.max(0, followersCount.value - 1)
        toast.success('已取消关注')
      } else {
        toast.error(res.msg || '取消关注失败')
      }
    } else {
      const res = await request.post('/api/user-follows/follow', {
        uid: currentUid,
        follow_uid: authorId.value
      })
      if (res.code === 200) {
        isFollowed.value = true
        followersCount.value = followersCount.value + 1
        toast.success('关注成功')
      } else {
        toast.error(res.msg || '关注失败')
      }
    }
  } catch (err) {
    console.error('操作失败:', err)
    toast.error('操作失败，请稍后重试')
  } finally {
    following.value = false
  }
}

const checkIsFollowing = async () => {
  if (!store.isLoggedIn || isSelf.value) {
    isFollowed.value = false
    return
  }
  
  try {
    const res = await request.get('/api/user-follows/is-following', {
      follow_uid: authorId.value
    })
    if (res.code === 200 && res.data) {
      isFollowed.value = res.data.is_following || false
    }
  } catch (err) {
    console.error('检查关注状态失败:', err)
  }
}

const loadFollowCounts = async () => {
  try {
    const cachedKey = `author_follow_counts_${authorId.value}`
    const cached = cache.get(cachedKey)
    if (cached) {
      followingCount.value = cached.following
      followersCount.value = cached.followers
      return
    }
    
    const [followingRes, followersRes] = await Promise.all([
      request.get('/api/user-follows/counts', {
        target_type: 'following',
        target_ids: [authorId.value]
      }),
      request.get('/api/user-follows/counts', {
        target_type: 'followers',
        target_ids: [authorId.value]
      })
    ])
    
    if (followingRes.code === 200 && followingRes.data?.counts) {
      followingCount.value = followingRes.data.counts[authorId.value] || 0
    }
    if (followersRes.code === 200 && followersRes.data?.counts) {
      followersCount.value = followersRes.data.counts[authorId.value] || 0
    }
    
    cache.set(cachedKey, {
      following: followingCount.value,
      followers: followersCount.value
    }, 5)
  } catch (err) {
    console.error('获取关注统计失败:', err)
  }
}

const changePage = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  loadArticles()
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

// 数据加载
const loadUserInfo = async () => {
  loading.value = true
  error.value = ''
  try {
    const cacheKey = `author_info_${authorId.value}`
    const cacheExpire = 5

    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      userInfo.value = cachedData
      if (cachedData.nickname) {
        setDynamicTitle(`${cachedData.nickname} 的主页`)
      }
    }

    const res = await request.get('/api/users/one', {
      id: authorId.value,
      field: 'id,account,nickname,avatar,description,gender,status,exp,create_time,login_time,title,json,result'
    })

    if (res.code === 200 && res.data) {
      userInfo.value = res.data
      cache.set(cacheKey, res.data, cacheExpire)
      if (res.data.nickname) {
        setDynamicTitle(`${res.data.nickname} 的主页`)
      }
    } else if (res.code === 404 || res.code === 204) {
      error.value = '用户不存在'
      userInfo.value = null
    } else {
      error.value = res.msg || '获取用户信息失败'
      userInfo.value = null
    }
  } catch (err) {
    console.error('加载用户信息失败:', err)
    error.value = '网络错误，请稍后重试'
    userInfo.value = null
  } finally {
    loading.value = false
  }
}

const loadArticleCount = async () => {
  if (!authorId.value) return
  try {
    const cacheKey = `author_articles_count_${authorId.value}`
    const cached = cache.get(cacheKey)
    if (cached !== null && cached !== undefined) {
      articleCount.value = cached
      return
    }

    const res = await request.get('/api/article/count', {
      where: JSON.stringify({
        uid: authorId.value,
        audit: 1
      })
    })

    if (res.code === 200) {
      articleCount.value = res.data || 0
      cache.set(cacheKey, articleCount.value, 3)
    }
  } catch (err) {
    console.error('获取文章数量失败:', err)
  }
}

const loadArticles = async () => {
  if (!authorId.value) return
  articlesLoading.value = true
  try {
    const cacheKey = `author_articles_${authorId.value}_${currentPage.value}`
    const cacheExpire = 3

    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      articleList.value = cachedData.list
      articleCount.value = cachedData.count
      articlesLoading.value = false
      return
    }

    const where = JSON.stringify({
      uid: authorId.value,
      audit: 1
    })

    const [listRes, countRes] = await Promise.all([
      request.get('/api/article/all', {
        page: currentPage.value,
        limit: pageSize.value,
        order: 'create_time desc',
        where: where,
        field: 'id,title,abstract,views,create_time,tags'
      }),
      request.get('/api/article/count', {
        where: where
      })
    ])

    if (listRes.code === 200 && listRes.data) {
      articleList.value = listRes.data.data || []
    }

    if (countRes.code === 200) {
      articleCount.value = countRes.data || 0
    }

    cache.set(cacheKey, { list: articleList.value, count: articleCount.value }, cacheExpire)
  } catch (err) {
    console.error('加载文章列表失败:', err)
    articleList.value = []
  } finally {
    articlesLoading.value = false
  }
}

const loadLikeCount = async () => {
  if (!authorId.value) return
  try {
    const cacheKey = `author_likes_${authorId.value}`
    const cached = cache.get(cacheKey)
    if (cached !== null && cached !== undefined) {
      likeCount.value = cached
      return
    }

    const res = await request.get('/api/user-likes/counts', {
      target_type: 'user_likes',
      target_ids: [authorId.value]
    })

    if (res.code === 200 && res.data?.counts) {
      likeCount.value = res.data.counts[authorId.value] || 0
      cache.set(cacheKey, likeCount.value, 3)
    }
  } catch (err) {
    console.error('获取点赞数失败:', err)
  }
}

const loadCollectCount = async () => {
  if (!authorId.value) return
  try {
    const cacheKey = `author_collects_${authorId.value}`
    const cached = cache.get(cacheKey)
    if (cached !== null && cached !== undefined) {
      collectCount.value = cached
      return
    }

    const res = await request.get('/api/user-collects/counts', {
      target_type: 'user_collects',
      target_ids: [authorId.value]
    })

    if (res.code === 200 && res.data?.counts) {
      collectCount.value = res.data.counts[authorId.value] || 0
      cache.set(cacheKey, collectCount.value, 3)
    }
  } catch (err) {
    console.error('获取收藏数失败:', err)
  }
}

const loadFollowersList = async ({ append = false } = {}) => {
  if (!authorId.value) return
  // 仅本人可见：非本人直接返回，不发起请求
  if (!isSelf.value) {
    followerList.value = []
    followersHasMore.value = false
    return
  }
  // append 模式：使用更轻量的 MoreLoading；首次加载走头部 Loading
  if (append) {
    if (followersMoreLoading.value || !followersHasMore.value) return
    followersMoreLoading.value = true
  } else {
    if (followersLoading.value) return
    followersLoading.value = true
  }
  try {
    const res = await request.get('/api/user-follows/followers', {
      uid: authorId.value,
      page: followersPage.value,
      limit: followersPageSize
    })
    if (res.code === 200 && res.data) {
      const newRows = res.data.list || res.data.data || []
      followersTotal.value = Number(res.data.count) || 0
      if (append) {
        // 追加模式：去重合并，避免 ID 重复展示
        const exist = new Set(followerList.value.map((x) => String(x.id)))
        for (const r of newRows) if (!exist.has(String(r.id))) followerList.value.push(r)
      } else {
        followerList.value = newRows
      }
      // 当页返回数量 < 每页大小：说明后端没更多了，直接设为无更多
      if (newRows.length < followersPageSize) followersHasMore.value = false
      // 兜底：已展示总数 >= 后端返回 count，也视为无更多
      if (followersTotal.value > 0 && followerList.value.length >= followersTotal.value) {
        followersHasMore.value = false
      }
    }
  } catch (err) {
    console.error('加载粉丝列表失败:', err)
    if (!append) followerList.value = []
  } finally {
    if (append) followersMoreLoading.value = false
    else followersLoading.value = false
  }
}

const loadFollowingList = async ({ append = false } = {}) => {
  if (!authorId.value) return
  // 仅本人可见：非本人直接返回，不发起请求
  if (!isSelf.value) {
    followingList.value = []
    followingHasMore.value = false
    return
  }
  if (append) {
    if (followingMoreLoading.value || !followingHasMore.value) return
    followingMoreLoading.value = true
  } else {
    if (followingLoading.value) return
    followingLoading.value = true
  }
  try {
    const res = await request.get('/api/user-follows/following', {
      uid: authorId.value,
      page: followingPage.value,
      limit: followingPageSize
    })
    if (res.code === 200 && res.data) {
      const newRows = res.data.list || res.data.data || []
      followingTotal.value = Number(res.data.count) || 0
      if (append) {
        const exist = new Set(followingList.value.map((x) => String(x.id)))
        for (const r of newRows) if (!exist.has(String(r.id))) followingList.value.push(r)
      } else {
        followingList.value = newRows
      }
      if (newRows.length < followingPageSize) followingHasMore.value = false
      if (followingTotal.value > 0 && followingList.value.length >= followingTotal.value) {
        followingHasMore.value = false
      }
    }
  } catch (err) {
    console.error('加载关注列表失败:', err)
    if (!append) followingList.value = []
  } finally {
    if (append) followingMoreLoading.value = false
    else followingLoading.value = false
  }
}

const targetTypeLabel = (type) => {
  const map = {
    article: '文章',
    comment: '评论',
    moment: '动态',
    user: '用户',
    page: '页面'
  }
  return map[type] || type || '内容'
}

const targetTypeBadge = (type) => {
  const map = {
    article: 'text-bg-primary',
    comment: 'text-bg-info',
    moment: 'text-bg-success',
    user: 'text-bg-warning',
    page: 'text-bg-secondary'
  }
  return map[type] || 'text-bg-light'
}

// 根据类型把 /all 或 /one 返回的行，写入详情 Map（必须在 fetchBatchDetails 前定义，const 不提升）
const fillDetailMap = (type, map, r) => {
  if (!r || r.id === undefined) return
  const key = String(r.id)
  if (type === 'article') {
    map.set(key, {
      title: r.title,
      abstract: r.abstract,
      content: r.content,
      views: r.views,
      create_time: r.create_time
    })
  } else if (type === 'comment') {
    map.set(key, {
      content: r.content,
      bind_id: r.bind_id,
      bind_type: r.bind_type,
      uid: r.uid,
      create_time: r.create_time
    })
  } else if (type === 'moment') {
    map.set(key, {
      content: r.content,
      images: r.images,
      location: r.location,
      uid: r.uid,
      create_time: r.create_time
    })
  } else if (type === 'page') {
    map.set(key, {
      title: r.title,
      content: r.content,
      views: r.views,
      create_time: r.create_time
    })
  } else if (type === 'user') {
    map.set(key, {
      nickname: r.nickname,
      avatar: r.avatar,
      description: r.description,
      id: r.id
    })
  }
}

// 按类型批量拉取详情
// 新方案：使用 /all 接口的 MongoDB 风格 where 查询（where={"id":{"$in":[...]}}） + limit=999 一次拉全部
//        对比旧的 /one 分片并发，每种 target_type 从 N 次请求降为 1 次，彻底规避 QPS 限流
const fetchBatchDetails = async (type, ids) => {
  if (!type || !Array.isArray(ids) || ids.length === 0) return new Map()
  const idArr = [...new Set(ids.map(Number).filter(Boolean))]
  const resultMap = new Map()
  if (idArr.length === 0) return resultMap

  const endpointMap = {
    article: '/api/article',
    comment: '/api/comment',
    moment: '/api/moments',
    page: '/api/pages',
    user: '/api/users'
  }
  const endpoint = endpointMap[type]
  if (!endpoint) return resultMap

  const fieldMap = {
    article: 'id,title,abstract,content,views,create_time',
    comment: 'id,content,bind_id,bind_type,uid,create_time',
    moment: 'id,content,images,location,uid,create_time,top,views',
    page: 'id,title,content,views,create_time',
    user: 'id,nickname,avatar,description,exp'
  }
  const field = fieldMap[type] || ''

  try {
    const where = JSON.stringify({ id: { $in: idArr } })
    const params = {
      where,
      page: 1,
      // 按实际 ID 数量请求，不再用 200 / 999 这种「全量」兜底数字，
      // 避免用户在 DevTools 里看到 limit=200 误以为主分页是假的。
      limit: idArr.length,
      order: 'id desc'
    }
    if (field) params.field = field

    const res = await request.get(`${endpoint}/all`, params)
    let rows = []
    if (res.code === 200) {
      if (Array.isArray(res.data)) rows = res.data
      else if (Array.isArray(res.data?.data)) rows = res.data.data
      // 如果 all 被禁用或无结果，降级走 /one（一次并发 5 个兜底，非大流量场景）
    }
    if (rows.length === 0 && idArr.length <= 20) {
      await Promise.all(
        idArr.map(async (id) => {
          try {
            const p = field ? { id, field } : { id }
            const oneRes = await request.get(`${endpoint}/one`, p)
            if (oneRes.code === 200 && oneRes.data && oneRes.data.id !== undefined) {
              rows.push(oneRes.data)
            }
          } catch (_) { /* ignore */ }
        })
      )
    }
    rows.forEach((r) => fillDetailMap(type, resultMap, r))
  } catch (e) {
    console.warn(`[fetchBatchDetails] ${type} 批量查询失败:`, e?.message || e)
  }
  return resultMap
}

const enrichInteractionList = async (list) => {
  if (!list || list.length === 0) return []
  // 按 target_type 分组收集 target_id
  const groupedIds = {}
  list.forEach((i) => {
    if (!i.target_type || !i.target_id) return
    if (!groupedIds[i.target_type]) groupedIds[i.target_type] = []
    groupedIds[i.target_type].push(i.target_id)
  })
  // 每种类型一次批量请求（最多 5 种 → 5 次请求/每页，而不是 10+ 次 /one）
  const types = Object.keys(groupedIds)
  const detailMaps = {}
  await Promise.all(
    types.map(async (t) => {
      detailMaps[t] = await fetchBatchDetails(t, groupedIds[t])
    })
  )
  return list.map((item) => {
    const m = detailMaps[item.target_type]
    if (!m) return item
    const detail = m.get(String(item.target_id))
    if (!detail) return item
    return { ...item, _detail: detail }
  })
}

const switchLikesSubTab = (tabKey) => {
  if (activeLikesSubTab.value === tabKey) return
  activeLikesSubTab.value = tabKey
  likesPage.value = 1
  likesList.value = []
  likesHasMore.value = true
  loadLikesList()
  nextTick(() => bindInfiniteScrollObserver())
}

const switchCollectsSubTab = (tabKey) => {
  if (activeCollectsSubTab.value === tabKey) return
  activeCollectsSubTab.value = tabKey
  collectsPage.value = 1
  collectsList.value = []
  collectsHasMore.value = true
  loadCollectsList()
  nextTick(() => bindInfiniteScrollObserver())
}

const loadLikesList = async ({ append = false } = {}) => {
  if (!authorId.value) return
  // 仅本人可见：非本人直接返回，不发起请求
  if (!isSelf.value) {
    likesList.value = []
    likesHasMore.value = false
    return
  }
  if (append) {
    if (likesMoreLoading.value || !likesHasMore.value) return
    likesMoreLoading.value = true
  } else {
    if (likesLoading.value) return
    likesLoading.value = true
  }
  try {
    const params = {
      uid: authorId.value,
      page: likesPage.value,
      limit: likesPageSize
    }
    if (activeLikesSubTab.value) {
      params.target_type = activeLikesSubTab.value
    }
    const res = await request.get('/api/user-likes/likes', params)
    if (res.code === 200 && res.data) {
      const raw = res.data.list || res.data.data || []
      likesTotal.value = Number(res.data.count) || 0
      const enriched = await enrichInteractionList(raw)
      if (append) {
        const exist = new Set(likesList.value.map((x) => String(x.id)))
        for (const r of enriched) if (!exist.has(String(r.id))) likesList.value.push(r)
      } else {
        likesList.value = enriched
      }
      if (raw.length < likesPageSize) likesHasMore.value = false
      if (likesTotal.value > 0 && likesList.value.length >= likesTotal.value) {
        likesHasMore.value = false
      }
    }
  } catch (err) {
    console.error('加载点赞列表失败:', err)
    if (!append) likesList.value = []
  } finally {
    if (append) likesMoreLoading.value = false
    else likesLoading.value = false
  }
}

const loadCollectsList = async ({ append = false } = {}) => {
  if (!authorId.value) return
  if (!isSelf.value) {
    collectsList.value = []
    collectsTotal.value = 0
    collectsHasMore.value = false
    return
  }
  if (append) {
    if (collectsMoreLoading.value || !collectsHasMore.value) return
    collectsMoreLoading.value = true
  } else {
    if (collectsLoading.value) return
    collectsLoading.value = true
  }
  try {
    const params = {
      page: collectsPage.value,
      limit: collectsPageSize
    }
    if (activeCollectsSubTab.value) {
      params.target_type = activeCollectsSubTab.value
    }
    const res = await request.get('/api/user-collects/collects', params)
    if (res.code === 200 && res.data) {
      const raw = res.data.list || res.data.data || []
      collectsTotal.value = Number(res.data.count) || 0
      const enriched = await enrichInteractionList(raw)
      if (append) {
        const exist = new Set(collectsList.value.map((x) => String(x.id)))
        for (const r of enriched) if (!exist.has(String(r.id))) collectsList.value.push(r)
      } else {
        collectsList.value = enriched
      }
      if (raw.length < collectsPageSize) collectsHasMore.value = false
      if (collectsTotal.value > 0 && collectsList.value.length >= collectsTotal.value) {
        collectsHasMore.value = false
      }
    }
  } catch (err) {
    console.error('加载收藏列表失败:', err)
    if (!append) collectsList.value = []
  } finally {
    if (append) collectsMoreLoading.value = false
    else collectsLoading.value = false
  }
}

// 统一触发哨兵对应的「加载下一页」逻辑
const loadMoreByActiveTab = async () => {
  // 隐私保护：非本人访问时直接拦截，不触发任何「更多加载」
  if (!isSelf.value) return
  const tab = activeTab.value
  if (tab === 'followers') {
    if (followersMoreLoading.value || !followersHasMore.value) return
    followersPage.value += 1
    await loadFollowersList({ append: true })
  } else if (tab === 'following') {
    if (followingMoreLoading.value || !followingHasMore.value) return
    followingPage.value += 1
    await loadFollowingList({ append: true })
  } else if (tab === 'likes') {
    if (likesMoreLoading.value || !likesHasMore.value) return
    likesPage.value += 1
    await loadLikesList({ append: true })
  } else if (tab === 'collects') {
    if (collectsMoreLoading.value || !collectsHasMore.value) return
    collectsPage.value += 1
    await loadCollectsList({ append: true })
  }
}

// 根据当前 activeTab 拿到真正需要观察的哨兵节点（只观察当前展示的那个，避免四个 tab 同时触发）
const getActiveSentinel = () => {
  const tab = activeTab.value
  if (tab === 'followers') return followersSentinelRef.value
  if (tab === 'following') return followingSentinelRef.value
  if (tab === 'likes') return likesSentinelRef.value
  if (tab === 'collects') return collectsSentinelRef.value
  return null
}

const bindInfiniteScrollObserver = () => {
  if (typeof IntersectionObserver === 'undefined') return
  if (scrollObserver) {
    scrollObserver.disconnect()
  }
  scrollObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        // 只有「哨兵进入视口」且当前 tab 是它所属的 tab，才触发 loadMore
        if (entry.isIntersecting) {
          const target = entry.target
          const activeNode = getActiveSentinel()
          if (target === activeNode) {
            loadMoreByActiveTab()
          }
        }
      })
    },
    { root: null, rootMargin: '160px 0px', threshold: 0 }
  )
  // 四个哨兵都观察，但触发时会判断 activeTab，避免误触
  ;[followersSentinelRef, followingSentinelRef, likesSentinelRef, collectsSentinelRef].forEach(
    (r) => r.value && scrollObserver.observe(r.value)
  )
}

// 监听路由参数变化
watch(() => route.params.id, () => {
  if (authorId.value) {
    userInfo.value = null
    articleList.value = []
    articleCount.value = 0
    likeCount.value = 0
    collectCount.value = 0
    currentPage.value = 1
    isFollowed.value = false
    followingCount.value = 0
    followersCount.value = 0
    activeTab.value = 'articles'
    followerList.value = []
    followingList.value = []
    likesList.value = []
    collectsList.value = []
    activeLikesSubTab.value = 'article'
    activeCollectsSubTab.value = 'article'
    likesPage.value = 1
    collectsPage.value = 1
    likesTotal.value = 0
    collectsTotal.value = 0
    followersPage.value = 1
    followingPage.value = 1
    followersTotal.value = 0
    followingTotal.value = 0
    // 切用户时重置无限加载状态
    followersHasMore.value = true
    followingHasMore.value = true
    likesHasMore.value = true
    collectsHasMore.value = true
    followersMoreLoading.value = false
    followingMoreLoading.value = false
    likesMoreLoading.value = false
    collectsMoreLoading.value = false
    loadUserInfo()
    loadArticles()
    loadArticleCount()
    loadFollowCounts()
    loadLikeCount()
    loadCollectCount()
    checkIsFollowing()
  }
})

// 生命周期
onMounted(() => {
  if (authorId.value) {
    loadUserInfo()
    loadArticleCount()
    loadArticles()
    loadFollowCounts()
    loadLikeCount()
    loadCollectCount()
    checkIsFollowing()
  }
  // 首屏哨兵还没渲染（因为 activeTab 是 articles，粉丝/点赞的哨兵不在 DOM 里）
  // 等用户切到对应 tab 时，switchTab 会通过 nextTick 重新 bind
  nextTick(() => bindInfiniteScrollObserver())
})

onBeforeUnmount(() => {
  if (scrollObserver) {
    scrollObserver.disconnect()
    scrollObserver = null
  }
})
</script>

<style scoped>
/* ============ 用户信息卡片样式 ============ */
.author-profile-card {
  border: none;
  box-shadow: 0 2px 16px rgba(0, 0, 0, 0.08);
}

/* 头部 */
.profile-header {
  display: flex;
  gap: 1.5rem;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.profile-avatar {
  position: relative;
  flex-shrink: 0;
}

.avatar-online-dot {
  position: absolute;
  bottom: 1px;
  right: 1px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background-color: #10b981;
  border: 2px solid var(--bs-body-bg);
  box-shadow: 0 0 6px rgba(16, 185, 129, 0.5);
  z-index: 2;
}

.profile-info {
  flex: 1;
  min-width: 0;
}

.profile-name-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.profile-name {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--bs-heading-color);
  margin: 0;
}

.profile-uid {
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  background-color: var(--bs-secondary-bg-subtle);
  padding: 0.15rem 0.5rem;
  border-radius: 0.25rem;
}

.profile-role {
  font-size: 0.75rem;
  color: #fff;
  background-color: #10b981;
  padding: 0.15rem 0.5rem;
  border-radius: 0.25rem;
}

.profile-role-disabled {
  background-color: #6c757d;
}

.profile-desc {
  font-size: 0.9rem;
  color: var(--bs-secondary-color);
  line-height: 1.5;
  margin: 0 0 0.5rem 0;
}

.profile-desc-placeholder {
  color: var(--bs-secondary-color);
  font-style: italic;
  opacity: 0.85;
}

.profile-website {
  font-size: 0.85rem;
  color: var(--bs-primary);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
}

.profile-website:hover {
  text-decoration: underline;
}

.profile-website-placeholder {
  margin: 0;
  visibility: hidden;
}

/* 标签行 */
.profile-tags-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.tag-tag {
  font-size: 0.8rem;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--bs-secondary-color);
}

.tag-level {
  color: #7c3aed;
  font-weight: 600;
}

.tag-gender {
  color: #3b82f6;
}

.tag-gender-female {
  color: #ec4899;
}

.tag-role-icon {
  color: var(--bs-secondary-color);
}

.tag-date {
  color: var(--bs-tertiary-color);
}

/* 经验条 */
.exp-section {
  padding: 0.75rem 0;
  margin-bottom: 1rem;
}

.exp-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.375rem;
}

.exp-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: #7c3aed;
}

.exp-info {
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
  display: flex;
  align-items: center;
  gap: 0.125rem;
}

.exp-current {
  font-weight: 600;
  color: #7c3aed;
}

.exp-sep {
  margin: 0 0.125rem;
}

.exp-max {
  color: var(--bs-tertiary-color);
}

.exp-unit {
  margin-left: 0.25rem;
}

.exp-next {
  margin-left: 0.5rem;
  color: #7c3aed;
  font-weight: 500;
}

.exp-progress {
  height: 8px;
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.08));
  border-radius: 4px;
  overflow: hidden;
}

.exp-bar {
  height: 100%;
  background: linear-gradient(90deg, #7c3aed, #a855f7, #c084fc);
  border-radius: 4px;
  transition: width 0.6s ease;
}

.exp-desc {
  font-size: 0.75rem;
  color: var(--bs-tertiary-color);
  margin: 0.375rem 0 0 0;
  font-style: italic;
}

.level-view-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  padding: 4px 12px;
  font-size: 12px;
  border: 1px solid var(--bs-border-color);
  border-radius: 16px;
  background: transparent;
  color: var(--bs-secondary-color);
  cursor: pointer;
  transition: all 0.2s;
}

.level-view-btn:hover {
  background: var(--bs-secondary-bg);
  color: var(--bs-body-color);
  border-color: var(--bs-secondary-color);
}

/* 等级弹窗 */
.level-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
  animation: fadeIn 0.2s ease;
}

.level-modal-content {
  background: var(--bs-body-bg);
  border-radius: 16px;
  max-width: 900px;
  width: 90%;
  max-height: 85vh;
  overflow-y: auto;
  padding: 24px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  animation: slideUp 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from { transform: translateY(20px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

/* 统计数据 */
.profile-stats {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 0.25rem;
  padding: 1rem 0;
  border-top: 1px solid var(--bs-border-color, rgba(0, 0, 0, 0.1));
  border-bottom: 1px solid var(--bs-border-color, rgba(0, 0, 0, 0.1));
  margin-bottom: 1rem;
}

.stat-card {
  text-align: center;
  cursor: pointer;
  padding: 0.5rem;
  border-radius: 0.5rem;
  transition: background-color 0.2s ease;
}

.stat-card:hover {
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.03));
}

.stat-card.active {
  background-color: var(--bs-primary-bg-subtle, rgba(var(--bs-primary-rgb), 0.1));
  border-bottom: 2px solid var(--bs-primary);
}

.stat-card.active .stat-num {
  color: var(--bs-primary);
}

.stat-card.active .stat-text {
  color: var(--bs-primary);
}

.stat-num {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--bs-heading-color);
  line-height: 1.2;
}

.stat-text {
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
  margin-top: 0.125rem;
}

/* 操作按钮 */
.profile-actions {
  display: flex;
  justify-content: center;
}

.action-btn {
  min-width: 120px;
}

/* ============ 文章列表样式 ============ */

/* 列表卡片内的细分 tab */
.card-tab-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 0.5rem;
  padding: 0.75rem 1.25rem;
  background-color: rgba(0, 0, 0, 0.015);
  border-bottom: 1px solid var(--bs-border-color-translucent, #eee);
}

.card-tab-item {
  border: 1px solid var(--bs-border-color-translucent, #dee2e6);
  background-color: transparent;
  color: var(--bs-secondary-color, #6c757d);
  font-size: 0.82rem;
  padding: 0.35rem 0.9rem;
  border-radius: 999px;
  transition: all 0.2s ease;
  line-height: 1.2;
}

.card-tab-item:hover {
  color: var(--bs-primary);
  border-color: rgba(13, 110, 253, 0.4);
}

.card-tab-item.active {
  background-color: var(--bs-primary);
  color: #fff;
  border-color: var(--bs-primary);
  box-shadow: 0 2px 8px rgba(13, 110, 253, 0.22);
}

/* 分页栏 */
.card-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1.25rem;
  gap: 0.75rem;
}

.page-btn {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: 1px solid var(--bs-border-color-translucent, #dee2e6);
  background-color: var(--bs-body-bg);
  color: var(--bs-secondary-color);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.page-btn:hover:not(:disabled) {
  color: #fff;
  background-color: var(--bs-primary);
  border-color: var(--bs-primary);
  transform: translateY(-1px);
}

.page-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.page-info {
  font-size: 0.85rem;
  color: var(--bs-secondary-color);
  text-align: center;
  flex: 1;
}

.author-article-item {
  cursor: pointer;
  transition: background-color 0.25s ease;
  padding: 0;
}

.author-article-item:first-child {
  border-top: none;
}

.author-article-item:hover {
  background-color: var(--bs-hover-bg, rgba(0, 0, 0, 0.03));
}

.author-article-item:hover .article-title {
  color: var(--bs-primary);
}

.author-article-item:hover .article-actions .btn-outline-secondary {
  transform: translateX(4px);
  color: var(--bs-primary);
  border-color: var(--bs-primary);
}

.article-inner {
  display: flex;
  align-items: center;
  padding: 1rem 1.25rem;
  gap: 1rem;
}

.article-main {
  flex: 1;
  min-width: 0;
}

.article-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: var(--bs-heading-color);
  margin: 0 0 0.35rem 0;
  line-height: 1.4;
  transition: color 0.2s ease;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-desc {
  font-size: 0.85rem;
  color: var(--bs-secondary-color);
  margin: 0 0 0.6rem 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.article-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.article-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
  color: var(--bs-secondary-color);
  padding: 0.15rem 0.5rem;
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.04));
  border-radius: 0.375rem;
  line-height: 1.4;
}

.meta-chip i {
  font-size: 0.7rem;
}

.article-tags {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.tag-pill {
  font-size: 0.7rem;
  color: var(--bs-primary);
  background-color: var(--bs-primary-bg-subtle, rgba(13, 110, 253, 0.1));
  padding: 0.125rem 0.5rem;
  border-radius: 0.75rem;
  line-height: 1.4;
  transition: background-color 0.2s ease;
}

.author-article-item:hover .tag-pill {
  background-color: var(--bs-primary);
  color: #fff;
}

.article-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.article-actions .btn {
  width: 2rem;
  height: 2rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border-radius: 50%;
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.04));
}

.article-actions .btn i {
  font-size: 0.85rem;
}

.article-arrow {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 50%;
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.04));
}

.article-arrow i {
  color: var(--bs-secondary-color);
  font-size: 0.85rem;
  transition: transform 0.25s ease, color 0.25s ease;
}

.empty-state {
  padding: 2rem 0;
}

.empty-state i {
  font-size: 3rem;
  color: var(--bs-tertiary-color);
  opacity: 0.6;
}

/* 社交列表项 */
.author-social-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.author-social-item:hover {
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.03));
}

.social-avatar {
  flex-shrink: 0;
}

.social-main {
  flex: 1;
  min-width: 0;
}

.social-name {
  font-weight: 600;
  font-size: 0.95rem;
  color: var(--bs-heading-color);
  margin-bottom: 0.125rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.social-desc {
  font-size: 0.8rem;
  color: var(--bs-secondary-color);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.social-time {
  font-size: 0.75rem;
  color: var(--bs-tertiary-color);
  margin-top: 0.125rem;
}

.social-arrow {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 50%;
  background-color: var(--bs-secondary-bg-subtle, rgba(0, 0, 0, 0.04));
  transition: transform 0.25s ease, background-color 0.25s ease;
}

.social-arrow i {
  color: var(--bs-secondary-color);
  font-size: 0.85rem;
}

.author-social-item:hover .social-arrow {
  background-color: var(--bs-primary-bg-subtle, rgba(var(--bs-primary-rgb), 0.1));
  transform: translateX(4px);
}

.author-social-item:hover .social-arrow i {
  color: var(--bs-primary);
}

/* ============ 响应式 ============ */

@media (max-width: 768px) {
  .profile-header {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }

  .profile-name-row {
    justify-content: center;
  }

  .profile-name {
    font-size: 1.25rem;
  }

  .profile-tags-row {
    justify-content: center;
  }

  .stat-num {
    font-size: 1.1rem;
  }

  .stat-text {
    font-size: 0.7rem;
  }

  .article-inner {
    padding: 0.875rem 1rem;
    gap: 0.75rem;
  }

  .article-title {
    font-size: 0.95rem;
  }

  .article-desc {
    font-size: 0.8rem;
    margin-bottom: 0.5rem;
  }

  .article-footer {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.375rem;
  }

  .article-actions .btn {
    width: 1.75rem;
    height: 1.75rem;
  }
}

/* 头衔颜色 - 修仙体系 */
.title-badge {
  color: #fff;
  font-weight: 500;
  border: none;
}
.title-zhangmen { background: linear-gradient(135deg, #f6d365, #fda085) !important; color: #5a3e00 !important; }
.title-zhanglao { background: #8e44ad !important; }
.title-hufa { background: #c0392b !important; }
.title-neimen { background: #2980b9 !important; }
.title-waimen { background: #16a085 !important; }
.title-lianqi { background: #27ae60 !important; }
.title-zhuji { background: #8bc34a !important; color: #1a3d00 !important; }
.title-jiedan { background: #e67e22 !important; }
.title-yuanying { background: #6c5ce7 !important; }
.title-huashen { background: #00b894 !important; }
.title-default { background: #6c757d !important; }
</style>
