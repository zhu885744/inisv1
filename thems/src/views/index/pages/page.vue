<template>
  <!-- 全局外层容器：居中、收窄宽度、统一留白 -->
  <div class="article-page-wrapper">
    <!-- 加载状态：优化尺寸、间距，更协调 -->
    <div v-if="loading" class="page-loading">
      <div class="spinner-border text-info fs-4" role="status">
        <span class="visually-hidden">Loading...</span>
      </div>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="article-content-wrap wx-card p-3 mt-2">
      <p class="mb-0 fw-normal">{{ errorMsg }}</p>
    </div>

    <!-- 独立页面主体：核心UI重写 -->
    <div v-else class="article-main">
      <!-- 归档页面统计信息 -->
      <div v-if="isArchivePage">
        <!-- 统计信息卡片区域 -->
        <main class="wx-card mt-2">
          <div class="p-3">
            <h2 class="timeline-title mb-4">数据统计</h2>
            <!-- 统计信息网格 -->
            <div class="stats-grid">
              <!-- 文章总数 -->
              <div class="stat-card">
                <div class="stat-icon article-icon">
                  <i class="bi bi-file-earmark-text"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.articleCount || 0 }}</h3>
                  <p class="stat-label">文章总数</p>
                </div>
              </div>

              <!-- 文章分类总数 -->
              <div class="stat-card">
                <div class="stat-icon category-icon">
                  <i class="bi bi-tags"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.categoryCount || 0 }}</h3>
                  <p class="stat-label">文章分类总数</p>
                </div>
              </div>

              <!-- 独立页面总数 -->
              <div class="stat-card">
                <div class="stat-icon page-icon">
                  <i class="bi bi-file-earmark"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.pageCount || 0 }}</h3>
                  <p class="stat-label">独立页面总数</p>
                </div>
              </div>

              <!-- 标签总数 -->
              <div class="stat-card">
                <div class="stat-icon tag-icon">
                  <i class="bi bi-tag"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.tagCount || 0 }}</h3>
                  <p class="stat-label">标签总数</p>
                </div>
              </div>

              <!-- 友情链接总数 -->
              <div class="stat-card">
                <div class="stat-icon link-icon">
                  <i class="bi bi-link"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.linkCount || 0 }}</h3>
                  <p class="stat-label">友情链接总数</p>
                </div>
              </div>

              <!-- 评论总数 -->
              <div class="stat-card">
                <div class="stat-icon comment-icon">
                  <i class="bi bi-chat"></i>
                </div>
                <div class="stat-content">
                  <h3 class="stat-value">{{ archiveStats.commentCount || 0 }}</h3>
                  <p class="stat-label">评论总数</p>
                </div>
              </div>
            </div>

            <!-- 文章归档时间线 -->
            <div class="archive-timeline mt-6">
              <h2 class="timeline-title mb-4">文章归档（{{ archiveStats.articleCount || 0 }}篇）</h2>
              
              <!-- 加载状态 -->
              <div v-if="articlesLoading" class="d-flex justify-content-center py-5">
                <div class="spinner-border text-info" role="status">
                  <span class="visually-hidden">Loading...</span>
                </div>
              </div>
              
              <!-- 错误状态 -->
              <div v-else-if="articlesError" class="alert alert-warning d-flex align-items-center mt-2 p-2" role="alert">
                <i class="bi bi-exclamation-triangle-fill fs-4 me-2"></i>
                <p class="mb-0">{{ articlesErrorMsg }}</p>
              </div>
              
              <!-- 无文章数据 -->
              <div v-else-if="Object.keys(groupedArticles).length === 0" class="wx-empty-state">
                <i class="bi bi-file-earmark-text"></i>
                <p>暂无文章数据，敬请期待～</p>
              </div>
              
              <div v-else class="timeline-container">
                <div v-for="(articles, yearMonth) in groupedArticles" :key="yearMonth" class="timeline-year-month">
                  <h3 class="timeline-year-month-title">{{ yearMonth }}</h3>
                  <div class="timeline-items">
                    <div v-for="article in articles" :key="article.id" class="timeline-item">
                      <div class="timeline-content">
                        <router-link :to="`/archives/${article.id}`" class="timeline-article-title">{{ article.title }}</router-link>
                        <div class="timeline-article-meta">
                          <span class="timeline-article-date d-flex align-items-center">
                            <i class="bi bi-calendar-date me-1"></i>{{ format(article.publish_time) }}
                          </span>
                          <span class="timeline-article-category d-flex align-items-center" v-if="article.result?.group && article.result.group.length > 0">
                            <i class="bi bi-folder me-1"></i>{{ article.result.group[0].name }}
                          </span>
                          <span class="timeline-article-views d-flex align-items-center">
                            <i class="bi bi-eye me-1"></i>{{ article.views || 0 }}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
                <div v-if="articlePage.value < articlePageCount.value" class="text-center py-4">
                  <button
                    @click="loadMoreArticles"
                    class="wx-btn-outline"
                    :disabled="articlesLoading.value"
                  >
                    <i class="bi bi-arrow-down-circle me-1"></i>
                    {{ articlesLoading.value ? '加载中...' : '加载更多文章' }}
                  </button>
                </div>
              </div>

              <!-- 刷新按钮 -->
              <div class="refresh-container">
                <button
                  @click="refreshArchiveStats"
                  class="wx-btn-outline"
                  :disabled="refreshingArchive"
                >
                  <i class="bi bi-arrow-clockwise" :class="{ 'spin': refreshingArchive }"></i>
                  {{ refreshingArchive ? '刷新中...' : '刷新数据' }}
                </button>
              </div>
            </div>
          </div>
        </main>
      </div>

      <!-- 友链页面 -->
      <div v-else-if="isLinksPage">
        <!-- 全局加载状态 -->
        <div v-if="loading" class="d-flex justify-content-center align-items-center py-10">
          <div class="spinner-border text-info" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
        </div>

        <!-- 错误状态 -->
        <div v-else-if="error" class="article-content-wrap wx-card p-3 mt-2">
          <p class="mb-0 fw-normal">{{ errorMsg }}</p>
        </div>

        <!-- 友链页面主体 -->
        <main class="wx-card mt-2">
          <div class="p-3">
          <h2 class="timeline-title mb-4">友情链接</h2>
          <!-- 核心内容区 -->
          <div class="pb-2">
            <!-- 友链介绍 -->
            <div class="article-content text-secondary">
              <i-markdown :model-value="pageInfo.content || '暂无友链介绍，敬请期待～'" />
            </div>

            <!-- 友链数据加载状态 -->
            <div v-if="linksLoading" class="d-flex justify-content-center py-3">
              <div class="spinner-border text-info spinner-border-sm" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
            </div>
            
            <!-- 友链加载错误 -->
            <div v-else-if="linksError" class="alert alert-warning d-flex align-items-center mt-2 p-2" role="alert">
              <i class="bi bi-exclamation-triangle-fill fs-4 me-2"></i>
              <p class="mb-0">{{ linksErrorMsg }}</p>
            </div>
            
            <!-- 无友链数据 -->
            <div v-else-if="links.length === 0" class="wx-empty-state">
              <i class="bi bi-link-45deg"></i>
              <p>暂无友链数据，敬请期待～</p>
            </div>
            
            <!-- 友链分组展示 -->
            <div v-else class="link-list-container">
              <div v-for="(groupLinks, groupName) in groupLinkMap" :key="groupName" class="mb-6">
                <h3 class="link-cate mt-3">{{ groupName }}</h3>
                <!-- 友链小卡片网格布局 -->
                <div class="row g-3">
                  <!-- 友链小卡片 -->
                  <div v-for="link in groupLinks" :key="link.id" class="col-lg-4 col-md-6 col-sm-6 col-12">
                    <a 
                      :href="link.url" 
                      :target="link.target || '_blank'" 
                      class="link-card-small text-decoration-none d-block h-100"
                    >
                      <div class="link-card-inner h-100">
                        <!-- 友链名称（带状态） -->
                        <div class="d-flex justify-content-between align-items-center mb-2">
                          <h4 class="link-name mb-0">{{ link.nickname }}</h4>
                          <!-- 状态指示器 -->
                          <span 
                            :class="[
                              'status-badge',
                              link.online ? 'status-online' : 'status-offline'
                            ]"
                            :title="link.online ? `响应时间: ${link.responseTime}ms` : '网站离线'"
                          >
                            <i class="bi" :class="link.online ? 'bi-check-circle' : 'bi-x-circle'"></i>
                            <span class="ms-1">{{ link.online ? '在线' : '离线' }}</span>
                            <span v-if="link.online" class="response-time">{{ link.responseTime }}ms</span>
                          </span>
                        </div>
                        <!-- 友链简介 -->
                        <p class="link-desc mb-2">{{ link.description || '暂无介绍' }}</p>
                        <!-- 友链链接 -->
                        <small class="link-url text-muted text-truncate d-block">
                          <i class="bi bi-link-45deg me-1"></i>{{ link.url }}
                        </small>
                      </div>
                    </a>
                  </div>
                </div>
              </div>
            </div>

            <!-- 友链申请按钮 -->
            <div class="m-4 text-center">
              <button 
                v-if="isLogin" 
                @click="openLinkApplyModal"
                :disabled="isSubmitting || isCoolingDown"
                class="btn btn-primary btn-sm"
              >
                <i class="bi bi-link-45deg me-1"></i>
                {{ isCoolingDown ? `冷却中 (${coolingDownTime}s)` : '申请友链' }}
              </button>
              <button 
                v-else 
                @click="handleLogin"
                class="btn btn-outline-primary btn-sm"
                disabled
              >
                <i class="bi bi-person me-1"></i>登录后申请
              </button>
              <p class="text-muted mt-2">请勿重复申请，每个用户只能申请一次</p>
            </div>
          </div>
          </div>
        </main>
        
        <!-- 评论区域 -->
        <div class="mt-4">
          <i-comment 
              :articleId="pageInfo.id" 
              :commentCount="commentCount" 
              :commentList="commentList" 
              :isLogin="isLogin" 
              :isDarkMode="isDarkMode"
              :currentPage="currentPage"
              :pageSize="pageSize"
              :totalComments="totalComments"
              :articleCommentConfig="articleCommentConfig"
              @publishComment="handlePublishComment"
              @replyComment="handleReplyComment"
              @pageChange="handleCommentPageChange"
            />
        </div>
      </div>
      
      <!-- 留言页面 -->
      <div v-else-if="isMessagePage">
        <!-- 留言页面主体 -->
        <main class="mt-2">
          <!-- 核心内容区 -->
          <div class="pb-2">
            <!-- 留言统计 -->
            <div class="message-stats mb-6">
              <div class="card p-4">
                <div class="row flex-nowrap">
                  <div class="col-md-3 col-sm-6 col-3">
                    <div class="text-center">
                      <div class="fs-3 font-bold">{{ commentCount || 0 }}</div>
                      <div class="text-muted">总留言数</div>
                    </div>
                  </div>
                  <div class="col-md-3 col-sm-6 col-3">
                    <div class="text-center">
                      <div class="fs-3 font-bold">{{ uniqueCommenters || 0 }}</div>
                      <div class="text-muted">留言人数</div>
                    </div>
                  </div>
                  <div class="col-md-3 col-sm-6 col-3">
                    <div class="text-center">
                      <div class="fs-3 font-bold">{{ recentMessagesCount || 0 }}</div>
                      <div class="text-muted">最近留言</div>
                    </div>
                  </div>
                  <div class="col-md-3 col-sm-6 col-3">
                    <div class="text-center">
                      <div class="fs-3 font-bold">{{ viewCount || 0 }}</div>
                      <div class="text-muted">页面浏览</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 留言显示模块 -->
            <div class="message-display mb-6 mt-2">
              <div class="wx-card">
                <div class="card-header bg-transparent">
                  <h3 class="h5 fw-bold mt-2">
                    <i class="bi bi-chat-dots me-2"></i>
                    留言列表
                  </h3>
                </div>
                <div class="card-body">
                  <!-- 评论功能关闭提示 -->
                  <div v-if="!isCommentEnabled" class="text-center py-5 text-muted">
                    <i class="bi bi-chat-x fs-3 mb-2"></i>
                    <p class="text-sm text-muted mt-2">感谢您的关注，留言功能正在维护中</p>
                  </div>
                  
                  <!-- 留言输入框：仅登录状态显示 -->
                  <div class="mb-5" v-else-if="isLogin">
                    <div
                      ref="messageEditorRef"
                      class="form-control border border-secondary-subtle bg-body wx-contenteditable w-100"
                      :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !messageInput }"
                      contenteditable="true"
                      data-placeholder="请输入你的留言..."
                      @input="onMessageInput"
                      style="min-height: 80px; max-height: 300px; overflow-y: auto;"
                    ></div>

                    <!-- 表情选择面板 -->
                    <i-emoji-picker
                      v-model="showMessageEmojiPicker"
                      :is-dark-mode="isDarkMode"
                      @select="insertMessageEmoji"
                    />
                    
                    <!-- 按钮区域：表情按钮和发布留言按钮在同一行 -->
                    <div class="d-flex gap-2 mt-3">
                      <button 
                        @click="toggleMessageEmojiPicker"
                        class="wx-btn-outline px-4 emoji-button"
                        :class="{ 'bg-dark border-dark-subtle': isDarkMode }"
                      >
                        <i class="bi bi-emoji-smile me-1"></i> 表情
                      </button>
                      <button 
                          @click="handlePublishMessage"
                          class="btn btn-primary px-4 publish-btn flex-grow-1"
                          :disabled="!messageInput.trim() || isPublishingMessage"
                        >
                        <i class="bi" :class="isPublishingMessage ? 'bi-arrow-clockwise spin' : 'bi-paper-plane-fill'"></i>
                        {{ isPublishingMessage ? ' 发布中...' : ' 发布留言' }}
                        </button>
                    </div>
                  </div>

                  <!-- 未登录引导区 -->
                  <div class="mb-5 p-4 bg-body text-center border" v-else-if="!isLogin">
                    <i class="bi bi-person-circle fs-3  mb-2"></i>
                    <p class="mb-3 text-muted">登录后即可发表留言～</p>
                    <div class="d-flex gap-2 justify-content-center">
                      <button 
                        @click="handleToLogin()"
                        class="btn btn-primary btn-sm px-4"
                      >
                        登录
                      </button>
                      <button 
                        @click="handleToRegister()"
                        class="btn btn-outline-primary btn-sm px-4"
                      >
                        注册
                      </button>
                    </div>
                  </div>

                  <!-- 留言列表 -->
                  <div v-if="isCommentEnabled && commentList.length > 0">
                    <div ref="messagesGrid" class="messages-grid">
                      <div 
                        class="message-item" 
                        v-for="(message, index) in commentList" 
                        :key="message.id || index"
                      >
                        <!-- 便利贴样式留言卡片 -->
                        <div class="sticky-note wx-card">
                          <div class="card-body">
                            <!-- 留言头部：用户信息和时间 -->
                            <div class="d-flex justify-content-between align-items-start mb-3">
                              <div class="d-flex align-items-center">
                                <img 
                                  :src="message.result?.author?.avatar || message.author?.avatar || message.avatar || 'https://picsum.photos/60/60'" 
                                  class="message-avatar rounded-circle me-3 border border-light shadow-sm" 
                                  alt="用户头像"
                                  style="width: 40px; height: 40px; object-fit: cover;"
                                  loading="lazy"
                                  decoding="async"
                                >
                                <div>
                                  <h6 class="fw-semibold mb-0">
                                    <span v-if="message.result?.author?.id || message.author?.id" class="text-decoration-none" style="cursor: pointer;" @click.prevent="navigateToAuthor(message.result?.author?.id || message.author?.id)">
                                      {{ message.result?.author?.nickname || message.author?.nickname || message.nickname || '匿名用户' }}
                                    </span>
                                    <span v-else>{{ message.result?.author?.nickname || message.author?.nickname || message.nickname || '匿名用户' }}</span>
                                    <span v-if="message.result?.author?.result?.isAuthor || message.result?.author?.isAuthor || message.author?.result?.isAuthor || message.isAuthor" class="badge bg-primary text-white ms-2 rounded-pill">作者</span>
                                  </h6>
                                  <small class="text-muted">{{ format(message.create_time || message.time || message.update_time) }}</small>
                                </div>
                              </div>
                              <!-- 回复图标：点击显示回复弹窗 -->
                              <div class="reply-icon-container">
                                <button 
                                  class="btn btn-sm btn-outline-secondary reply-icon" 
                                  @click="openReplyModal(message)"
                                  data-bs-toggle="modal"
                                  data-bs-target="#replyModal"
                                >
                                  <i class="bi bi-chat-dots"></i>
                                  <span class="ms-1">{{ message.replies?.length || 0 }}</span>
                                </button>
                              </div>
                            </div>
                            <!-- 留言内容 -->
                            <div class="message-content mb-3">
                              <p class="mb-0" v-html="processMessageContent(message.content || '')"></p>
                            </div>
                            <!-- 操作按钮 -->
                            <div class="message-actions">
                              <button 
                                class="wx-btn-outline reply-btn"
                                @click="openReplyModal(message)"
                                data-bs-toggle="modal"
                                data-bs-target="#replyModal"
                                v-if="isLogin"
                              >
                                <i class="bi bi-reply-fill me-1"></i> 回复
                              </button>
                              <button 
                                class="btn btn-sm btn-outline-secondary disabled reply-btn" 
                                v-else
                                data-bs-toggle="tooltip"
                                data-bs-title="登录后可回复"
                              >
                                <i class="bi bi-reply-fill me-1"></i> 回复
                              </button>
                            </div>

                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  
                  <!-- 无留言提示 -->
                  <div v-else-if="isCommentEnabled" class="wx-empty-state">
                    <i class="bi bi-chat-dots"></i>
                    <p>暂无留言，快来抢沙发吧～</p>
                  </div>
                  
                  <!-- 回复弹窗 -->
                  <div class="modal fade" id="replyModal" tabindex="-1" aria-labelledby="replyModalLabel" aria-hidden="true">
                    <div class="modal-dialog modal-lg">
                      <div class="modal-content">
                        <div class="modal-header">
                          <h5 class="modal-title" id="replyModalLabel">回复列表</h5>
                          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                          <!-- 评论功能关闭提示 -->
                          <div v-if="!isCommentEnabled" class="text-center py-4 text-muted">
                            <i class="bi bi-chat-x fs-3 mb-2"></i>
                            <p class="mb-0">留言功能暂时关闭</p>
                          </div>
                          
                          <!-- 回复列表 -->
                          <div v-else-if="selectedMessage && selectedMessage.replies && selectedMessage.replies.length > 0">
                            <div 
                              class="reply-item mb-3 p-3 border rounded-3" 
                              v-for="(reply, index) in selectedMessage.replies" 
                              :key="reply.id || index"
                            >
                              <div class="d-flex align-items-start mb-2">
                                <img 
                                  :src="reply.result?.author?.avatar || reply.author?.avatar || reply.avatar || 'https://picsum.photos/62/62'" 
                                  class="message-avatar rounded-circle me-3 border border-light shadow-sm" 
                                  alt="回复用户头像"
                                  style="width: 35px; height: 35px; object-fit: cover;"
                                  loading="lazy"
                                  decoding="async"
                                >
                                <div class="flex-grow-1">
                                  <h6 class="fw-semibold mb-0">
                                    <span v-if="reply.result?.author?.id || reply.author?.id" class="text-decoration-none" style="cursor: pointer;" @click.prevent="navigateToAuthor(reply.result?.author?.id || reply.author?.id)">
                                      {{ reply.result?.author?.nickname || reply.author?.nickname || reply.nickname || '匿名用户' }}
                                    </span>
                                    <span v-else>{{ reply.result?.author?.nickname || reply.author?.nickname || reply.nickname || '匿名用户' }}</span>
                                    <span v-if="reply.result?.author?.result?.isAuthor || reply.result?.author?.isAuthor || reply.author?.result?.isAuthor || reply.isAuthor" class="badge bg-primary text-white ms-2 rounded-pill">作者</span>
                                  </h6>
                                  <small class="text-muted">{{ format(reply.create_time || reply.time || reply.update_time) }}</small>
                                </div>
                              </div>
                              <p class="mb-0 px-2 py-1 bg-body-tertiary" v-html="processMessageContent(reply.content || '')"></p>
                            </div>
                          </div>
                          <div v-else-if="isCommentEnabled" class="text-center py-3 text-muted">
                            <p class="mb-0">暂无回复</p>
                          </div>
                          
                          <!-- 回复输入框 -->
                          <div class="mt-4 reply-form" v-if="isCommentEnabled && isLogin">
                            <h6 class="mb-2">回复 {{ selectedMessage?.result?.author?.nickname || selectedMessage?.author?.nickname || selectedMessage?.nickname || '用户' }}:</h6>
                            <div
                              ref="replyEditorRef"
                              class="form-control border border-secondary-subtle bg-body wx-contenteditable"
                              :class="{ 'bg-dark border-dark-subtle': isDarkMode, 'is-empty': !replyInput }"
                              contenteditable="true"
                              data-placeholder="请输入你的回复..."
                              @input="onReplyInput"
                              style="min-height: 80px; max-height: 300px; overflow-y: auto;"
                            ></div>

                            <!-- 回复表情选择面板 -->
                            <i-emoji-picker
                              v-model="showReplyEmojiPicker"
                              :is-dark-mode="isDarkMode"
                              @select="insertReplyEmoji"
                            />
                          </div>
                        </div>
                        <div class="modal-footer">
                          <div class="w-100" v-if="isCommentEnabled && isLogin">
                            <div class="d-flex gap-2">
                              <button 
                                @click="toggleReplyEmojiPicker"
                                class="btn btn-sm btn-outline-secondary px-3 emoji-button"
                                :class="{ 'bg-dark border-dark-subtle': isDarkMode }"
                              >
                                <i class="bi bi-emoji-smile me-1"></i> 表情
                              </button>
                              <button 
                                @click="handleSubmitReply(selectedMessage.id)"
                                class="wx-btn-gradient px-3 flex-grow-1"
                                :disabled="!replyInput.trim() || isPublishingMessage"
                              >
                                <i class="bi" :class="isPublishingMessage ? 'bi-arrow-clockwise spin' : ''"></i>
                                {{ isPublishingMessage ? ' 发送中...' : ' 发送回复' }}
                              </button>
                              <button type="button" class="btn btn-sm btn-secondary" data-bs-dismiss="modal">关闭</button>
                            </div>
                          </div>
                          <button type="button" class="btn btn-secondary" v-else data-bs-dismiss="modal">关闭</button>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- 分页控件 -->
                  <div v-if="isCommentEnabled && totalComments > pageSize" class="mt-4">
                    <nav aria-label="留言分页">
                      <ul class="pagination justify-content-center">
                        <li class="page-item" :class="{ disabled: currentPage === 1 }">
                          <button class="page-link" @click="handleCommentPageChange(currentPage - 1)" :disabled="currentPage === 1">
                            <span aria-hidden="true">&laquo;</span>
                          </button>
                        </li>
                        <li v-for="page in totalPages" :key="page" class="page-item" :class="{ active: currentPage === page }">
                          <button class="page-link" @click="handleCommentPageChange(page)">
                            {{ page }}
                          </button>
                        </li>
                        <li class="page-item" :class="{ disabled: currentPage === totalPages }">
                          <button class="page-link" @click="handleCommentPageChange(currentPage + 1)" :disabled="currentPage === totalPages">
                            <span aria-hidden="true">&raquo;</span>
                          </button>
                        </li>
                      </ul>
                    </nav>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </main>
      </div>
      
      <!-- 普通独立页面 -->
      <div v-else>
        <!-- 页面内容区：核心阅读区，重写样式 -->
        <main class="article-content-wrap wx-card mt-2">
          <div class="p-3">
          <!-- 页面头部：标题+元信息 -->
          <header class="article-header">
            <h1 class="article-title fw-bold mb-3 text-center">{{ pageInfo.title }}</h1>
            <!-- 文章元信息：居中布局、弱化样式 -->
            <div class="article-meta d-flex flex-wrap align-items-center justify-content-center text-muted gap-4 fs-6">
              <span class="meta-item d-flex align-items-center">
                <i class="bi bi-person-fill me-2"></i>
                {{ authorInfo.nickname || '匿名' }}
              </span>
              <span class="meta-item d-flex align-items-center">
                <i class="bi bi-calendar-fill me-2"></i>
                {{ format(pageInfo.publish_time || Date.now() / 1000) }}
              </span>
              <span class="meta-item d-flex align-items-center">
                <i class="bi bi-chat-fill me-2"></i>
                {{ commentCount || 0 }} 评论
              </span>
              <span class="meta-item d-flex align-items-center">
                <i class="bi bi-eye-fill me-2"></i>
                {{ viewCount || 0 }} 浏览
              </span>
              <span class="meta-item d-flex align-items-center">
                <i class="bi bi-clock-fill me-2"></i>
                最后更新：{{ format(pageInfo.update_time || pageInfo.last_update || pageInfo.publish || Date.now() / 1000) }}
              </span>
            </div>
          </header>
            <div class="article-content mt-4">
              <i-markdown :model-value="pageInfo.content || '暂无页面内容，敬请期待～'" />
            </div>
          </div>
        </main>
        
        <!-- 评论区域：优化间距，自然衔接 -->
        <section class="article-comment mt-2 mb-8">
          <i-comment 
            :articleId="pageInfo.id" 
            :commentCount="commentCount" 
            :commentList="commentList" 
            :isLogin="isLogin" 
            :isDarkMode="isDarkMode"
            :articleAuthor="authorInfo"
            :currentPage="currentPage"
            :pageSize="pageSize"
            :totalComments="totalComments"
            :articleCommentConfig="articleCommentConfig"
            @publishComment="handlePublishComment"
            @replyComment="handleReplyComment"
            @pageChange="handleCommentPageChange"
          />
        </section>
      </div>
    </div>
  </div>

  <!-- 友链申请弹窗 -->
  <div class="modal fade" id="linkApplyModal" tabindex="-1" aria-labelledby="linkApplyModalLabel" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title" id="linkApplyModalLabel">
            <i class="bi bi-link-45deg me-2"></i>申请友链
          </h5>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
        </div>
        <div class="modal-body">
          <form @submit.prevent="handleLinkApply">
            <div class="mb-3">
              <label for="linkNickname" class="form-label">网站名称</label>
              <input 
                type="text" 
                class="form-control" 
                id="linkNickname" 
                v-model="linkForm.nickname"
                required
                placeholder="请输入您的网站名称"
              >
            </div>
            <div class="mb-3">
              <label for="linkUrl" class="form-label">网站链接</label>
              <input 
                type="url" 
                class="form-control" 
                id="linkUrl" 
                v-model="linkForm.url"
                required
                placeholder="请输入您的网站链接"
              >
            </div>
            <div class="mb-3">
              <label for="linkAvatar" class="form-label">网站图标</label>
              <div class="input-group">
                <input 
                  type="url" 
                  class="form-control" 
                  id="linkAvatar" 
                  v-model="linkForm.avatar"
                  placeholder="请输入您的网站图标链接"
                >
                <button 
                  type="button" 
                  class="wx-btn-outline"
                  @click="handleUploadAvatar"
                >
                  上传
                </button>
              </div>
            </div>
            <div class="mb-3">
              <label for="linkDescription" class="form-label">网站描述</label>
              <textarea 
                class="form-control"
                id="linkDescription" 
                v-model="linkForm.description"
                rows="3"
                placeholder="请输入您的网站描述"
              ></textarea>
            </div>
          </form>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
          <button 
            type="button" 
            class="wx-btn-gradient"
            @click="handleLinkApply"
            :disabled="isSubmitting"
          >
            <i v-if="isSubmitting" class="bi bi-arrow-clockwise spin me-1"></i>
            {{ isSubmitting ? '提交中...' : '提交申请' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed, nextTick, shallowRef, markRaw } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useCommStore } from '@/store/comm'
import { request } from '@/utils/network'
import iMarkdown from '@/comps/custom/i-markdown.vue'
import iComment from '@/comps/custom/i-comment.vue'
import iEmojiPicker from '@/comps/custom/i-emoji-picker.vue'
import iVirtualScroll from '@/comps/custom/i-virtual-scroll.vue'
import utils from '@/utils/utils'
import { cache, uploadImage } from '@/utils/network'
import { usePageTitle, toast } from '@/utils/app'
import Sortable from 'sortablejs'

const store = useCommStore()

const { setDynamicTitle, setLoadingTitle, setErrorTitle } = usePageTitle({
  staticTitle: '页面',
  defaultTitle: '独立页面'
})

const props = defineProps({
  pageKey: {
    type: String,
    required: false,
    default: '',
  }
})

const loading = ref(true)
const error = ref(false)
const errorMsg = ref('')
const pageInfo = shallowRef({})

const linkForm = ref({
  nickname: '',
  url: '',
  description: '',
  avatar: '',
  target: '_blank',
  group: ''
})
const isSubmitting = ref(false)
const isCoolingDown = ref(false)
const coolingDownTime = ref(0)
let coolingDownTimer = null

const router = useRouter()
const route = useRoute()

const commentCount = ref(0)
const commentList = shallowRef([])
const isDarkMode = ref(false)
const authorInfo = shallowRef({})
const viewCount = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)
const totalComments = ref(0)

const parsedPageJson = computed(() => {
  const jsonData = pageInfo.value.json
  if (!jsonData) return null
  try {
    return typeof jsonData === 'string' ? JSON.parse(jsonData) : jsonData
  } catch (e) {
    console.error('解析页面JSON配置失败:', e)
    return null
  }
})

const articleCommentConfig = computed(() => {
  const parsed = parsedPageJson.value
  if (!parsed) {
    return markRaw({ allow: null, show: null })
  }
  return markRaw({
    allow: parsed?.comment?.allow ?? null,
    show: parsed?.comment?.show ?? null
  })
})

const archiveStats = ref({
  articleCount: 0,
  categoryCount: 0,
  pageCount: 0,
  tagCount: 0,
  linkCount: 0,
  commentCount: 0
})
const refreshingArchive = ref(false)
let archiveRefreshTimer = null

const articles = shallowRef([])
const articlesLoading = ref(false)
const articlesError = ref(false)
const articlesErrorMsg = ref('')
const groupedArticles = ref({})
const articlePage = ref(1)
const articlePageSize = ref(20)
const articleTotal = ref(0)

const articlePageCount = computed(() => {
  return Math.ceil(articleTotal.value / articlePageSize.value)
})

const links = shallowRef([])
const linksLoading = ref(false)
const linksError = ref(false)
const linksErrorMsg = ref('')
const linkTotal = ref(0)

const currentPageKey = computed(() => (props.pageKey || route.params.key || '').trim())

const isLogin = computed(() => store.login.finish && Object.keys(store.login.user).length > 0)

const isArchivePage = computed(() => currentPageKey.value === 'archive')

const isLinksPage = computed(() => currentPageKey.value === 'links')

const isMessagePage = computed(() => currentPageKey.value === 'message')

const uniqueCommenters = ref(0)
const recentMessagesCount = ref(0)

const messageInput = ref('')
const replyInput = ref('')
const messageEditorRef = ref(null)
const replyEditorRef = ref(null)
const showReplyIndex = ref(null)
const replyTarget = ref(null)
const selectedMessage = ref(null)
const messagesGrid = ref(null)
const showMessageEmojiPicker = ref(false)
const showReplyEmojiPicker = ref(false)
const isPublishingMessage = ref(false)
const commentConfig = shallowRef({})
const isCommentEnabled = ref(true)

const groupLinkMap = computed(() => {
  const map = {}
  links.value.forEach(link => {
    const groupName = link.result?.group?.name || '未分组'
    if (!map[groupName]) map[groupName] = []
    map[groupName].push(link)
  })
  return markRaw(map)
})

const formatTime = (timestamp) => {
  if (!timestamp || timestamp === 0) return '未知时间'
  return utils.timeToDate(timestamp, 'Y-M-D')
}

const format = (timestamp) => {
  if (!timestamp || timestamp === 0) return '未知时间'
  return utils.timeToDate(timestamp, 'Y-M-D')
}

const checkPageKey = (key) => {
  if (!key) {
    errorMsg.value = '页面标识不能为空，请检查访问地址'
    return false
  }
  const keyReg = /^[a-zA-Z0-9-_]+$/
  if (!keyReg.test(key)) {
    errorMsg.value = '页面标识不合法，仅允许字母、数字、横杠和下划线'
    return false
  }
  if (key.length > 20) {
    errorMsg.value = '页面标识过长，最大支持20个字符'
    return false
  }
  return true
}

const getCommentConfig = async () => {
  try {
    const cacheKey = 'comment_config'
    const cacheExpire = 300
    let cachedConfig = cache.get(cacheKey)
    if (cachedConfig) {
      return cachedConfig
    }
    const response = await request.get('/api/config/one', {
      key: 'COMMENT'
    })
    if (response.code === 200 && response.data) {
      const result = response.data.json || {}
      cache.set(cacheKey, result, cacheExpire)
      return result
    }
    return {}
  } catch (error) {
    console.error('获取评论配置失败:', error)
    return {}
  }
}

const getPageData = async (pageKey) => {
  loading.value = true
  error.value = false
  errorMsg.value = ''
  try {
    const cacheKey = `page_detail_${pageKey}`
    const cacheExpire = 60
    
    let cachedPage = cache.get(cacheKey)
    
    if (!cachedPage) {
      const queryParams = {
        key: String(pageKey || '').trim(),
        cache: false,
        withTrashed: false
      }
      const res = await request.get('/api/pages/one', queryParams)

      if (res.code === 200) {
        if (!res.data || Object.keys(res.data).length === 0) {
          error.value = true
          errorMsg.value = '未找到该独立页面，可能已被删除或访问地址错误'
          setErrorTitle('页面不存在')
        } else {
          cachedPage = res.data
          cache.set(cacheKey, cachedPage, cacheExpire)
          pageInfo.value = cachedPage
          viewCount.value = res.data.views || 0
          error.value = false
          setDynamicTitle(pageInfo.value.title)
        }
      } else {
        error.value = true
        errorMsg.value = res.msg || '获取独立页面数据失败'
        setErrorTitle('获取失败')
      }
    } else {
      pageInfo.value = cachedPage
      viewCount.value = cachedPage.views || 0
      error.value = false
      setDynamicTitle(pageInfo.value.title)
    }
  } catch (err) {
    error.value = true
    errorMsg.value = '网络异常，请检查网络后刷新页面'
    setErrorTitle('网络异常')
  } finally {
    loading.value = false
  }
}

const initPage = async () => {
  const key = currentPageKey.value

  if (key === 'archive') {
    setLoadingTitle()
    await getArchivePageData()
    await Promise.all([
      fetchArchiveStats(),
      fetchArticles()
    ])
    startArchiveAutoRefresh()
  } else if (key === 'links') {
    setLoadingTitle()
    await Promise.all([
      getLinksPageData(),
      fetchLinks()
    ])
    await getLinksComments(currentPage.value, pageSize.value)
  } else if (key === 'message') {
    setLoadingTitle()
    await getMessagePageData()
    await Promise.all([
      getComments(pageInfo.value.id, currentPage.value, pageSize.value),
      calculateMessageStats()
    ])
  } else if (checkPageKey(key)) {
    getPageData(key)
  } else {
    error.value = true
    loading.value = false
    setErrorTitle('页面标识不合法')
    setTimeout(() => route.goBack(), 3000)
  }
}

watch(
  currentPageKey,
  () => {
    initPage()
  },
  { immediate: false }
)

// 初始化拖拽功能
const initSortable = () => {
  if (isMessagePage.value && messagesGrid.value) {
    if (window.sortableInstance) {
      try {
        window.sortableInstance.destroy()
      } catch (e) {
        console.debug('Sortable销毁失败:', e)
      }
      window.sortableInstance = null
    }
    
    try {
      window.sortableInstance = new Sortable(messagesGrid.value, {
        animation: 150,
        ghostClass: 'sortable-ghost',
        chosenClass: 'sortable-chosen',
        dragClass: 'sortable-drag',
        handle: '.sticky-note',
        onEnd: function(evt) {
          console.log('拖拽排序完成', evt.oldIndex, evt.newIndex);
        }
      });
    } catch (error) {
      console.error('Sortable初始化失败:', error);
    }
  }
}

const getComments = async (pageId, page = 1, limit = 10) => {
  if (!pageId) return
  try {
    const cacheKey = `page_comments_${pageId}_${page}_${limit}`
    const cacheExpire = 5
    
    let cachedComments = cache.get(cacheKey)
    
    if (!cachedComments) {
      const res = await request.get('/api/comment/flat', {
        bind_id: pageId,
        bind_type: 'page',
        page: page,
        limit: limit,
        order: 'create_time desc'
      })
      
      if (res.code === 200) {
        const { count = 0, data = [] } = res.data || {}
        commentCount.value = count
        commentList.value = data
        totalComments.value = count
        currentPage.value = page
        pageSize.value = limit
        cache.set(cacheKey, {
          count,
          data,
          page,
          limit
        }, cacheExpire)
      }
    } else {
      commentCount.value = cachedComments.count || 0
      commentList.value = cachedComments.data || []
      totalComments.value = cachedComments.count || 0
      currentPage.value = cachedComments.page || page
      pageSize.value = cachedComments.limit || limit
    }
  } catch (error) {
    console.error('获取评论失败:', error)
  }
}

// 获取作者信息
const getAuthorInfo = async (authorId) => {
  try {
    const res = await request.get('/api/users/one', {
      id: authorId
    })
    
    if (res.code === 200 && res.data) {
      authorInfo.value = res.data
    }
  } catch (error) {
  }
}

// 发布评论
const handlePublishComment = async (data) => {
  try {
    const currentKey = (props.pageKey || route.params.key || '').trim()
    const isLinksPage = currentKey === 'links'
    
    const commentData = {
      content: data.content,
      bind_type: 'page',
      bind_id: pageInfo.value.id
    }
    
    const res = await request.post('/api/comment/create', commentData)
    
    if (res.code === 200) {
      const pageId = pageInfo.value.id
      const cachePrefix = isLinksPage ? 'links_comments' : 'page_comments'
      const cacheKeys = [
        `${cachePrefix}_${pageId}_1_${pageSize.value}`,
        `${cachePrefix}_${pageId}_${currentPage.value}_${pageSize.value}`
      ]
      
      cache.delMultiple(cacheKeys)
      if (isLinksPage) {
        await getLinksComments(currentPage.value, pageSize.value)
      } else {
        await getComments(pageInfo.value.id, currentPage.value, pageSize.value)
      }
      toast.success('评论发布成功！')
    } else {
      toast.error(res.msg || '评论发布失败')
    }
  } catch (error) {
    toast.error('网络异常，评论发布失败')
  }
}

// 回复评论
const handleReplyComment = async (data) => {
  try {
    const commentData = {
      content: data.content,
      bind_type: 'page',
      bind_id: pageInfo.value.id,
      pid: data.commentId
    }
    
    const res = await request.post('/api/comment/create', commentData)
    
    if (res.code === 200) {
      const pageId = pageInfo.value.id
      const currentKey = (props.pageKey || route.params.key || '').trim()
      const isLinksPage = currentKey === 'links'
      const cachePrefix = isLinksPage ? 'links_comments' : 'page_comments'
      const cacheKeys = [
        `${cachePrefix}_${pageId}_1_${pageSize.value}`,
        `${cachePrefix}_${pageId}_${currentPage.value}_${pageSize.value}`
      ]
      
      cache.delMultiple(cacheKeys)
      if (isLinksPage) {
        await getLinksComments(currentPage.value, pageSize.value)
      } else {
        await getComments(pageInfo.value.id, currentPage.value, pageSize.value)
      }
      toast.success('回复发布成功！')
    } else {
      toast.error(res.msg || '回复发布失败')
    }
  } catch (error) {
    toast.error('网络异常，回复发布失败')
  }
}

// 处理评论分页
const handleCommentPageChange = async (page) => {
  try {
    const currentKey = (props.pageKey || route.params.key || '').trim()
    const isLinksPage = currentKey === 'links'
    
    if (isLinksPage) {
      await getLinksComments(page, pageSize.value)
    } else if (pageInfo.value.id) {
      await getComments(pageInfo.value.id, page, pageSize.value)
    }
  } catch (error) {
  }
}

// 获取文章总数
const getArticleCount = async () => {
  try {
    const res = await request.get('/api/article/count')
    if (res.code === 200) {
      archiveStats.value.articleCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取文章分类总数
const getCategoryCount = async () => {
  try {
    const res = await request.get('/api/article-group/count')
    if (res.code === 200) {
      archiveStats.value.categoryCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取独立页面总数
const getPageCount = async () => {
  try {
    const res = await request.get('/api/pages/count', {
      onlyTrashed: false
    })
    if (res.code === 200) {
      archiveStats.value.pageCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取标签总数
const getTagCount = async () => {
  try {
    const res = await request.get('/api/tags/count')
    if (res.code === 200) {
      archiveStats.value.tagCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取友情链接总数
const getLinkCount = async () => {
  try {
    const res = await request.get('/api/links/count', {
      onlyTrashed: false
    })
    if (res.code === 200) {
      archiveStats.value.linkCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取评论总数
const getCommentCount = async () => {
  try {
    const res = await request.get('/api/comment/count')
    if (res.code === 200) {
      archiveStats.value.commentCount = res.data || 0
    }
  } catch (error) {
  }
}

// 获取所有归档统计数据
const fetchArchiveStats = async () => {
  try {
    const cacheKey = 'archive_stats'
    const cacheExpire = 30
    
    let cachedStats = cache.get(cacheKey)
    
    if (!cachedStats) {
      await Promise.all([
        getArticleCount(),
        getCategoryCount(),
        getPageCount(),
        getTagCount(),
        getLinkCount(),
        getCommentCount()
      ])
      cache.set(cacheKey, archiveStats.value, cacheExpire)
    } else {
      archiveStats.value = cachedStats
    }
  } catch (error) {
  } finally {
    loading.value = false
    refreshingArchive.value = false
  }
}

// 手动刷新归档统计数据
const refreshArchiveStats = async () => {
  refreshingArchive.value = true
  cache.del('archive_stats')
  await fetchArchiveStats()
  toast.success('数据刷新成功')
  refreshingArchive.value = false
}

// 启动归档页面自动刷新
const startArchiveAutoRefresh = () => {
  if (archiveRefreshTimer) {
    clearInterval(archiveRefreshTimer)
  }
  archiveRefreshTimer = setInterval(() => {
    fetchArchiveStats()
  }, 5 * 60 * 1000)
}

// 获取友链页面基础数据
const getLinksPageData = async () => {
  loading.value = true
  try {
    const cacheKey = 'page_links_data'
    const cacheExpire = 60
    
    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      pageInfo.value = { ...pageInfo.value, ...cachedData }
      viewCount.value = cachedData.views || 0
      setDynamicTitle(cachedData.title || '友链')
      loading.value = false
      const authorId = pageInfo.value.uid
      if (authorId) {
        getAuthorInfo(authorId)
      }
      return
    }
    
    const res = await request.get('/api/pages/one', {
      key: 'links',
      cache: false,
      withTrashed: false
    })
    if (res && res.code === 200 && res.data && Object.keys(res.data).length > 0) {
      pageInfo.value = { ...pageInfo.value, ...res.data }
      viewCount.value = res.data.views || 0
      setDynamicTitle(res.data.title || '友链')
      cache.set(cacheKey, res.data, cacheExpire)
    } else {
      error.value = true
      errorMsg.value = '未找到友链页面配置，请联系管理员检查后台'
    }
  } catch (err) {
    error.value = true
    errorMsg.value = '网络异常，无法加载友链页面配置'
  } finally {
    loading.value = false
    const authorId = pageInfo.value.uid
    if (authorId) {
      getAuthorInfo(authorId)
    }
  }
}

// 获取归档页面基础数据
const getArchivePageData = async () => {
  loading.value = true
  try {
    const cacheKey = 'page_archive_data'
    const cacheExpire = 60
    
    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      pageInfo.value = { ...pageInfo.value, ...cachedData }
      viewCount.value = cachedData.views || 0
      setDynamicTitle(cachedData.title || '归档')
      loading.value = false
      const authorId = pageInfo.value.uid
      if (authorId) {
        getAuthorInfo(authorId)
      }
      return
    }
    
    const res = await request.get('/api/pages/one', {
      key: 'archive',
      cache: false,
      withTrashed: false
    })
    if (res && res.code === 200 && res.data && Object.keys(res.data).length > 0) {
      pageInfo.value = { ...pageInfo.value, ...res.data }
      viewCount.value = res.data.views || 0
      setDynamicTitle(res.data.title || '归档')
      cache.set(cacheKey, res.data, cacheExpire)
    }
  } catch (err) {
    console.error('[归档页面基础数据异常]：', err)
  } finally {
    loading.value = false
    const authorId = pageInfo.value.uid
    if (authorId) {
      getAuthorInfo(authorId)
    }
  }
}

const fetchArticles = async () => {
  const cacheKey = 'archive_articles'
  const cacheExpire = 300
  const cached = cache.get(cacheKey)
  
  if (cached) {
    articles.value = cached.data
    articleTotal.value = cached.count
    groupArticlesByYearMonth(cached.data)
    return
  }
  
  articlesLoading.value = true
  articlesError.value = false
  articlesErrorMsg.value = ''
  
  try {
    const res = await request.get('/api/article/all', {
      page: 1,
      limit: 9999,
      order: 'create_time desc'
    })
    
    if (res && res.code === 200 && res.data) {
      const { data = [], count = 0 } = res.data
      
      articles.value = data
      articleTotal.value = count
      
      groupArticlesByYearMonth(articles.value)
      cache.set(cacheKey, { data, count }, cacheExpire)
    } else {
      articlesError.value = true
      articlesErrorMsg.value = res?.msg || '获取文章数据失败'
    }
  } catch (err) {
    articlesError.value = true
    articlesErrorMsg.value = '网络异常，无法加载文章数据'
  } finally {
    articlesLoading.value = false
  }
}

const loadMoreArticles = async () => {
  if (articlePage.value >= articlePageCount.value) return
  
  articlePage.value++
  try {
    const res = await request.get('/api/article/all', {
      page: articlePage.value,
      limit: articlePageSize.value,
      order: 'create_time desc'
    })
    
    if (res && res.code === 200 && res.data) {
      const { data = [] } = res.data
      articles.value = [...articles.value, ...data]
      groupArticlesByYearMonth(articles.value)
    }
  } catch (err) {
    console.error('加载更多文章失败:', err)
  }
}

const groupArticlesByYearMonth = (articlesData) => {
  const grouped = {}
  
  articlesData.forEach(article => {
    if (article.publish_time) {
      const date = new Date(article.publish_time * 1000)
      const year = date.getFullYear()
      const month = date.getMonth() + 1
      const yearMonth = `${year}年${month}月`
      
      if (!grouped[yearMonth]) {
        grouped[yearMonth] = []
      }
      
      grouped[yearMonth].push(article)
    }
  })
  
  groupedArticles.value = grouped
}

// 获取全部友链数据
const fetchLinks = async () => {
  linksLoading.value = true
  linksError.value = false
  linksErrorMsg.value = ''
  
  try {
    const cacheKey = 'links_list_with_status'
    const cacheExpire = 1440
    
    let cachedLinks = cache.get(cacheKey)
    
    if (!cachedLinks) {
      const res = await request.get('/api/links/all?status=true&limit=9999')
      if (res && res.code === 200 && res.data) {
        const { data = [], count = 0 } = res.data
        links.value = data
        linkTotal.value = count
        cache.set(cacheKey, { data, count, fetchedAt: Date.now() }, cacheExpire)
      } else {
        linksError.value = true
        linksErrorMsg.value = res?.msg || '获取友链数据失败'
      }
    } else {
      links.value = cachedLinks.data || []
      linkTotal.value = cachedLinks.count || 0
    }
  } catch (err) {
    linksError.value = true
    linksErrorMsg.value = '网络异常，无法加载友链数据'
  } finally {
    linksLoading.value = false
  }
}

// 获取友链页面评论
const getLinksComments = async (page = 1, limit = 10) => {
  try {
    const cacheKey = `links_comments_${pageInfo.value.id}_${page}_${limit}`
    const cacheExpire = 5
    
    let cachedComments = cache.get(cacheKey)
    
    if (!cachedComments) {
      const res = await request.get('/api/comment/flat', {
        bind_id: pageInfo.value.id,
        bind_type: 'page',
        page: page,
        limit: limit,
        order: 'create_time desc'
      })
      
      if (res.code === 200) {
        commentCount.value = res.data?.count || 0
        commentList.value = res.data?.data || []
        totalComments.value = res.data?.count || 0
        currentPage.value = page
        pageSize.value = limit
        cache.set(cacheKey, {
          count: res.data?.count || 0,
          data: res.data?.data || [],
          page: page,
          limit: limit
        }, cacheExpire)
      }
    } else {
      commentCount.value = cachedComments.count || 0
      commentList.value = cachedComments.data || []
      totalComments.value = cachedComments.count || 0
      currentPage.value = cachedComments.page || page
      pageSize.value = cachedComments.limit || limit
    }
  } catch (error) {
    console.error('获取评论失败：', error)
  }
}

// 获取留言页面基础数据
const getMessagePageData = async () => {
  loading.value = true
  try {
    const cacheKey = 'page_message_data'
    const cacheExpire = 60
    
    const cachedData = cache.get(cacheKey)
    if (cachedData) {
      pageInfo.value = { ...pageInfo.value, ...cachedData }
      viewCount.value = cachedData.views || 0
      setDynamicTitle(cachedData.title || '留言板')
      
      const config = await getCommentConfig()
      commentConfig.value = config
      isCommentEnabled.value = config.enabled !== 0
      
      const authorId = pageInfo.value.uid
      if (authorId) {
        getAuthorInfo(authorId)
      }
      return
    }
    
    const res = await request.get('/api/pages/one', {
      key: 'message',
      cache: false,
      withTrashed: false
    })
    if (res && res.code === 200 && res.data && Object.keys(res.data).length > 0) {
      pageInfo.value = { ...pageInfo.value, ...res.data }
      viewCount.value = res.data.views || 0
      setDynamicTitle(res.data.title || '留言板')
      cache.set(cacheKey, res.data, cacheExpire)
      
      const config = await getCommentConfig()
      commentConfig.value = config
      isCommentEnabled.value = config.enabled !== 0
    } else {
      error.value = true
      errorMsg.value = '未找到留言页面配置，请联系管理员检查后台'
    }
  } catch (err) {
    error.value = true
    errorMsg.value = '网络异常，无法加载留言页面配置'
  } finally {
    loading.value = false
    const authorId = pageInfo.value.uid
    if (authorId) {
      getAuthorInfo(authorId)
    }
  }
}

const calculateMessageStats = async () => {
  try {
    if (!pageInfo.value.id) return
    
    const cacheKey = `message_stats_${pageInfo.value.id}`
    const cacheExpire = 60
    const cachedStats = cache.get(cacheKey)
    
    if (cachedStats) {
      uniqueCommenters.value = cachedStats.uniqueCommenters || 0
      recentMessagesCount.value = cachedStats.recentMessagesCount || 0
      return
    }
    
    const res = await request.get('/api/comment/flat', {
      bind_id: pageInfo.value.id,
      bind_type: 'page',
      page: 1,
      limit: 200,
      order: 'create_time desc'
    })
    
    if (res.code === 200 && res.data?.data) {
      const allComments = res.data.data
      
      const uniqueUsers = new Set()
      allComments.forEach(comment => {
        const userId = comment.user_id || comment.result?.author?.id || comment.author?.id
        if (userId) {
          uniqueUsers.add(userId)
        }
      })
      uniqueCommenters.value = uniqueUsers.size
      
      const sevenDaysAgo = Date.now() - 7 * 24 * 60 * 60 * 1000
      const recentComments = allComments.filter(comment => {
        return comment.create_time * 1000 >= sevenDaysAgo
      })
      recentMessagesCount.value = recentComments.length
      
      cache.set(cacheKey, {
        uniqueCommenters: uniqueUsers.size,
        recentMessagesCount: recentComments.length
      }, cacheExpire)
    }
  } catch (error) {
    console.error('计算留言统计数据失败：', error)
  }
}

// 检测深色模式
const detectDarkMode = () => {
  isDarkMode.value = document.documentElement.classList.contains('dark') || 
    window.matchMedia('(prefers-color-scheme: dark)').matches
}

// 友链申请
const openLinkApplyModal = async () => {
  if (isCoolingDown.value) {
    toast.warning('请稍后再申请友链')
    return
  }
  
  linkForm.value = {
    nickname: '',
    url: '',
    description: '',
    avatar: '',
    target: '_blank'
  }
  
  if (window.bootstrap) {
    const modal = new window.bootstrap.Modal(document.getElementById('linkApplyModal'))
    modal.show()
  }
}

// 上传网站图标
const handleUploadAvatar = async () => {
  try {
    const path = await uploadImage()
    linkForm.value.avatar = path
    toast.success('网站图标上传成功')
  } catch (error) {
    toast.error(error.message || '上传失败')
  }
}

// 提交友链申请
const handleLinkApply = async () => {
  if (!linkForm.value.nickname.trim() || !linkForm.value.url.trim()) {
    toast.warning('请填写必填字段')
    return
  }
  
  try {
    new URL(linkForm.value.url)
  } catch {
    toast.warning('请输入正确的网站链接')
    return
  }
  
  isSubmitting.value = true
  
  try {
    const submitData = {
      ...linkForm.value,
      target: '_blank'
    }
    
    const res = await request.post('/api/links/create', submitData)
    
    if (res.code === 200) {
      toast.success('友链申请已提交，请等待审核')
      
      if (window.bootstrap) {
        const modal = window.bootstrap.Modal.getInstance(document.getElementById('linkApplyModal'))
        modal.hide()
      }
      
      startCoolingDown()
    } else {
      toast.error(res.msg || '提交失败，请重试')
    }
  } catch (err) {
    toast.error('网络错误，请稍后重试')
  } finally {
    isSubmitting.value = false
  }
}

// 冷却期
const startCoolingDown = () => {
  const COOLDOWN_SECONDS = 300
  isCoolingDown.value = true
  coolingDownTime.value = COOLDOWN_SECONDS
  
  if (coolingDownTimer) clearInterval(coolingDownTimer)
  
  coolingDownTimer = setInterval(() => {
    coolingDownTime.value--
    if (coolingDownTime.value <= 0) {
      clearInterval(coolingDownTimer)
      isCoolingDown.value = false
    }
  }, 1000)
}

// 处理登录
const handleLogin = () => {
  router.push('/login')
}

// 处理登录弹窗
const handleToLogin = () => {
  store.switchAuth('login', true)
}

// 处理注册弹窗
const handleToRegister = () => {
  store.switchAuth('register', true)
}

// 计算总页数
const totalPages = computed(() => {
  return Math.ceil(totalComments.value / pageSize.value)
})

// 处理留言内容
const processMessageContent = (content) => {
  if (!content) return ''
  let processedContent = content
  // 解析表情图片 [emoji:url]
  processedContent = processedContent.replace(/\[emoji:(https?:\/\/[^\]]+|\/[^\]]+)\]/g, '<img src="$1" class="inline-emoji" style="width: 24px; height: 24px; vertical-align: middle; display: inline-block; object-fit: contain;" loading="lazy">')
  // 处理换行
  processedContent = processedContent.replace(/\n/g, '<br>')
  // 处理@提及
  return processedContent.replace(/@([\u4e00-\u9fa5\w]+)/g, '<span class="at-mention">@$1</span>')
}

// 发布留言
const handlePublishMessage = async () => {
  const content = messageInput.value.trim()
  if (!content) return
  
  if (!isCommentEnabled.value) {
    toast.error('留言功能暂时关闭')
    return
  }
  
  isPublishingMessage.value = true
  
  try {
    const commentData = {
      content: content,
      bind_type: 'page',
      bind_id: pageInfo.value.id
    }
    
    const res = await request.post('/api/comment/create', commentData)
    
    if (res.code === 200) {
      const pageId = pageInfo.value.id
      const cacheKeys = [
        `page_comments_${pageId}_1_10`,
        `page_comments_${pageId}_${currentPage.value}_${pageSize.value}`
      ]
      
      cache.delMultiple(cacheKeys)
      await getComments(pageId, currentPage.value, pageSize.value)
      await calculateMessageStats()
      toast.success('留言发布成功！')
      messageInput.value = ''
      clearEditor(messageEditorRef.value)
    } else {
      toast.error(res.msg || '留言发布失败')
    }
  } catch (error) {
    toast.error('网络异常')
  } finally {
    isPublishingMessage.value = false
  }
}

// 提交回复
const handleSubmitReply = async (commentId) => {
  const content = replyInput.value.trim()
  if (!content || !commentId) return
  
  isPublishingMessage.value = true
  
  try {
    const commentData = {
      content: content,
      bind_type: 'page',
      bind_id: pageInfo.value.id,
      pid: commentId
    }
    
    const res = await request.post('/api/comment/create', commentData)
    
    if (res.code === 200) {
      const cacheKeys = [
        `page_comments_${pageInfo.value.id}_1_10`,
        `page_comments_${pageInfo.value.id}_${currentPage.value}_${pageSize.value}`
      ]
      
      cache.delMultiple(cacheKeys)
      await getComments(pageInfo.value.id, currentPage.value, pageSize.value)
      await calculateMessageStats()
      toast.success('回复成功！')
      closeReplyModal()
      replyInput.value = ''
      clearEditor(replyEditorRef.value)
    } else {
      toast.error(res.msg || '回复失败')
    }
  } catch (error) {
    toast.error('网络异常')
  } finally {
    isPublishingMessage.value = false
  }
}

// 表情功能
const toggleMessageEmojiPicker = () => {
  showMessageEmojiPicker.value = !showMessageEmojiPicker.value
  showReplyEmojiPicker.value = false
}

const toggleReplyEmojiPicker = () => {
  showReplyEmojiPicker.value = !showReplyEmojiPicker.value
  showMessageEmojiPicker.value = false
}

// ===== contenteditable 编辑器辅助函数 =====
// 从contenteditable div中提取纯文本内容，emoji图片转为[emoji:url]格式
const getEditorContent = (el) => {
  if (!el) return ''
  const clone = el.cloneNode(true)
  const imgs = clone.querySelectorAll('img[data-emoji]')
  imgs.forEach(img => {
    const url = img.getAttribute('data-emoji') || img.src || ''
    const textNode = document.createTextNode(`[emoji:${url}]`)
    img.parentNode.replaceChild(textNode, img)
  })
  return clone.innerText.trim()
}

// 清空编辑器内容
const clearEditor = (el) => {
  if (!el) return
  el.innerHTML = ''
}

// 在光标位置插入HTML
const insertHTMLAtCursor = (html, editorEl) => {
  if (!editorEl) return
  editorEl.focus()
  const selection = window.getSelection()
  if (!selection || selection.rangeCount === 0) {
    editorEl.innerHTML += html
    return
  }
  const range = selection.getRangeAt(0)
  range.deleteContents()
  const temp = document.createElement('div')
  temp.innerHTML = html
  const frag = document.createDocumentFragment()
  let lastNode = null
  while (temp.firstChild) {
    lastNode = temp.firstChild
    frag.appendChild(temp.firstChild)
  }
  range.insertNode(frag)
  if (lastNode) {
    range.setStartAfter(lastNode)
    range.collapse(true)
  }
  selection.removeAllRanges()
  selection.addRange(range)
}

// 留言输入事件
const onMessageInput = () => {
  messageInput.value = getEditorContent(messageEditorRef.value)
}

// 回复输入事件
const onReplyInput = () => {
  replyInput.value = getEditorContent(replyEditorRef.value)
}

const insertMessageEmoji = (emoji) => {
  const editor = messageEditorRef.value
  if (!editor) {
    messageInput.value += emoji
    return
  }
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  onMessageInput()
}

const insertReplyEmoji = (emoji) => {
  const editor = replyEditorRef.value
  if (!editor) {
    replyInput.value += emoji
    return
  }
  if (emoji && emoji.startsWith('[emoji:')) {
    const url = emoji.slice(7, -1)
    const imgHtml = `<img src="${url}" data-emoji="${url}" class="editor-emoji" style="width: 28px; height: 28px; vertical-align: middle; display: inline-block; object-fit: contain; margin: 0 2px;">&nbsp;`
    insertHTMLAtCursor(imgHtml, editor)
  } else {
    insertHTMLAtCursor(emoji + ' ', editor)
  }
  onReplyInput()
}

// 登录注册

// 回复弹窗
const openReplyModal = (message) => {
  selectedMessage.value = message
  replyInput.value = ''
  const nickname = message.result?.author?.nickname || message.author?.nickname || message.nickname || '用户'
  replyInput.value = `@${nickname} `
  // 等待弹窗渲染后设置编辑器内容
  nextTick(() => {
    const editor = replyEditorRef.value
    if (editor) {
      editor.innerText = `@${nickname} `
      // 将光标移到末尾
      const range = document.createRange()
      range.selectNodeContents(editor)
      range.collapse(false)
      const sel = window.getSelection()
      sel.removeAllRanges()
      sel.addRange(range)
      onReplyInput()
    }
  })
}

const cleanupModal = () => {
  document.querySelectorAll('.modal-backdrop').forEach(el => el.remove())
  document.body.classList.remove('modal-open')
  document.body.style.removeProperty('overflow')
  document.body.style.removeProperty('padding-right')
}

const closeReplyModal = () => {
  if (window.bootstrap) {
    const modal = window.bootstrap.Modal.getInstance(document.getElementById('replyModal'))
    if (modal) modal.hide()
  }
  cleanupModal()
  replyInput.value = ''
  clearEditor(replyEditorRef.value)
}

const navigateToAuthor = (authorId) => {
  const modalElement = document.getElementById('replyModal')
  if (!modalElement) {
    router.push(`/author/${authorId}`)
    return
  }
  
  const handleHidden = () => {
    modalElement.removeEventListener('hidden.bs.modal', handleHidden)
    cleanupModal()
    router.push(`/author/${authorId}`)
  }
  
  modalElement.addEventListener('hidden.bs.modal', handleHidden)
  closeReplyModal()
  
  setTimeout(() => {
    modalElement.removeEventListener('hidden.bs.modal', handleHidden)
    cleanupModal()
    router.push(`/author/${authorId}`)
  }, 500)
}

watch(
  () => pageInfo.value.id,
  (newId) => {
    if (newId) {
      getComments(newId, currentPage.value, pageSize.value)
      const authorId = pageInfo.value.uid
      if (authorId) getAuthorInfo(authorId)
    }
  },
  { immediate: false }
)

let sortableInitTimer = null
watch(
  () => commentList.value?.length,
  () => {
    if (sortableInitTimer) clearTimeout(sortableInitTimer)
    sortableInitTimer = setTimeout(() => {
      nextTick(() => {
        initSortable()
      })
    }, 100)
  }
)

onMounted(() => {
  initPage()
  detectDarkMode()
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', detectDarkMode)
})

onUnmounted(() => {
  if (archiveRefreshTimer) clearInterval(archiveRefreshTimer)
  if (coolingDownTimer) clearInterval(coolingDownTimer)
  if (sortableInitTimer) clearTimeout(sortableInitTimer)
  if (window.sortableInstance) {
    try {
      window.sortableInstance.destroy()
    } catch (e) {
      console.debug('Sortable销毁失败:', e)
    }
    window.sortableInstance = null
  }
  
  window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', detectDarkMode)
})
</script>

<style scoped>
/* 加载状态 */
.page-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 60vh;
}

/* 面包屑 */
.breadcrumb-custom {
  font-size: 0.85rem;
}

.breadcrumb-custom .breadcrumb-item a {
  color: #333;
  text-decoration: none;
}

.breadcrumb-custom .breadcrumb-item.active {
  color: #666;
}

/* 暗黑模式适配 */
[data-bs-theme=dark] {
  .breadcrumb-custom .breadcrumb-item a {
    color: #fff;
  }
  
  .breadcrumb-custom .breadcrumb-item.active {
    color: #ccc;
  }
}

/* 文章标题 */
.article-title {
  font-size: clamp(1.8rem, 5vw, 2.5rem);
  line-height: 1.3;
  font-weight: 700;
  margin-bottom: 1.5rem !important;
  color: var(--bs-heading-color);
  text-align: center;
  transition: all 0.3s ease;
}

/* 文章元信息 */
.article-meta {
  font-size: 0.9rem;
  color: var(--bs-secondary-color);
  flex-wrap: wrap;
  justify-content: center;
  gap: 1.5rem;
  padding: 1rem 0;
  border-top: 1px solid var(--bs-border-color);
  border-bottom: 1px solid var(--bs-border-color);
  margin: 1rem 0 2rem;
  transition: var(--wx-transition);
}
.article-meta .meta-item {
  white-space: nowrap;
}
.article-meta .bi {
  font-size: 1em;
  color: var(--bs-tertiary-color);
}

/* 文章内容 */
.article-content {
  line-height: 1.8;
  font-size: 1.05rem;
  color: var(--bs-body-color);
}

/* 适配Markdown渲染的内部样式（核心：保证阅读体验） */
.article-content :deep(h2),
.article-content :deep(h3),
.article-content :deep(h4) {
  margin: 1.8rem 0 0.8rem;
  font-weight: 600;
  line-height: 1.4;
  color: var(--bs-body-color);
}
.article-content :deep(h2) {
  font-size: 1.5rem;
}
.article-content :deep(h3) {
  font-size: 1.25rem;
}
.article-content :deep(img) {
  max-width: 100%;
  border-radius: 0.5rem;
  margin: 1.5rem auto;
  display: block;
  box-shadow: var(--bs-box-shadow-sm);
}
.article-content :deep(ul),
.article-content :deep(ol) {
  margin-bottom: 1.2rem;
  padding-left: 1.8rem;
  color: var(--bs-body-color);
}
.article-content :deep(li) {
  margin-bottom: 0.5rem;
  color: var(--bs-body-color);
}
.article-content :deep(a) {
  color: var(--bs-link-color);
  text-decoration: none;
}
.article-content :deep(a:hover) {
  color: var(--bs-link-hover-color);
  text-decoration: underline;
  text-underline-offset: 0.2rem;
}
.article-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1.2rem;
  color: var(--bs-body-color);
}
.article-content :deep(td),
.article-content :deep(th) {
  border: 1px solid var(--bs-border-color);
  padding: 0.7rem;
  text-align: left;
}
.article-content :deep(th) {
  background-color: var(--bs-tertiary-bg);
  font-weight: 600;
  color: var(--bs-body-color);
}

/* 评论区容器：基础间距 */
.article-comment {
  width: 100%;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .article-title {
    font-size: clamp(1.5rem, 5vw, 2rem);
  }
  .article-content {
    font-size: 1rem;
  }
  .article-meta {
    font-size: 0.65rem; /* 字号缩小，核心 */
    gap: 0.6rem !important; /* 元信息项之间的间距缩小，!important覆盖bootstrap的gap-4 */
  }
  .article-meta .meta-item .bi {
    font-size: 0.8em; /* 图标字号轻微缩小，更协调 */
    margin-right: 0.3rem !important; /* 图标与文字间距缩小 */
  }
}

/* 归档页面统计信息样式 */
/* 统计信息网格布局 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

/* 统计卡片样式 */
.stat-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.5rem;
  border-radius: var(--wx-radius-md);
  background-color: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  box-shadow: var(--wx-shadow-sm);
  transition: var(--wx-transition);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--wx-shadow-md);
  border-color: rgba(var(--bs-primary-rgb), 0.3);
}

/* 统计图标样式 */
.stat-icon {
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  color: white;
}

/* 不同统计项的图标颜色 */
.article-icon {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
}

.category-icon {
  background: linear-gradient(135deg, #10b981, #059669);
}

.page-icon {
  background: linear-gradient(135deg, #f59e0b, #d97706);
}

.tag-icon {
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
}

.link-icon {
  background: linear-gradient(135deg, #ec4899, #db2777);
}

.comment-icon {
  background: linear-gradient(135deg, #6366f1, #4f46e5);
}

/* 统计内容样式 */
.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  margin: 0 0 0.25rem 0;
  color: var(--bs-body-color);
}

.stat-label {
  font-size: 0.9rem;
  color: var(--bs-secondary-color);
  margin: 0;
}

/* 刷新按钮区域 */
.refresh-container {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--bs-border-color);
}

/* 刷新按钮已使用全局 wx-btn-outline */

.refresh-info {
  font-size: 0.75rem;
  color: var(--bs-tertiary-color);
  margin: 0;
}

/* 加载动画 */
.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 归档页面响应式设计 */
@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
  }

  .stat-card {
    padding: 1rem;
    gap: 0.75rem;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
    font-size: 1.25rem;
  }

  .stat-value {
    font-size: 1.5rem;
  }

  .stat-label {
    font-size: 0.8rem;
  }

  .refresh-container {
    flex-direction: column;
    gap: 0.5rem;
    text-align: center;
  }
}

/* 归档页面平板设备适配 */
@media (min-width: 769px) and (max-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

/* 文章归档时间线样式 */
.archive-timeline {
  margin-top: 2rem;
}

.timeline-title {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--bs-body-color);
  margin-bottom: 2rem;
  padding-bottom: 0.75rem;
  border-bottom: 2px solid var(--bs-border-color);
}

.timeline-container {
  position: relative;
}

.timeline-year-month {
  margin-bottom: 2.5rem;
}

.timeline-year-month-title {
  font-size: 1.3rem;
  font-weight: 600;
  color: var(--bs-primary);
  margin-bottom: 1.5rem;
  padding-left: 2rem;
  position: relative;
}

.timeline-year-month-title::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background-color: var(--bs-primary);
  box-shadow: 0 0 0 4px rgba(var(--bs-primary-rgb), 0.1);
}

.timeline-items {
  position: relative;
}

.timeline-items::before {
  content: '';
  position: absolute;
  left: 0.5rem;
  top: 0;
  bottom: 0;
  width: 2px;
  background-color: var(--bs-border-color);
}

.timeline-item {
  position: relative;
  margin-bottom: 1.5rem;
  padding-left: 1.5rem;
}

.timeline-content {
  background-color: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  padding: 1.25rem;
  transition: var(--wx-transition);
  position: relative;
  overflow: hidden;
}

.timeline-content::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background-color: var(--bs-primary);
  transform: scaleY(0);
  transform-origin: top;
  transition: transform 0.3s ease;
}

.timeline-content:hover {
  box-shadow: var(--wx-shadow-lg);
  transform: translateY(-2px);
  border-color: var(--bs-primary);
}

.timeline-content:hover::before {
  transform: scaleY(1);
}

.timeline-article-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--bs-body-color);
  text-decoration: none;
  margin-bottom: 0.75rem;
  display: block;
  transition: color 0.2s ease;
  line-height: 1.4;
}

.timeline-article-title:hover {
  color: var(--bs-primary);
  text-decoration: none;
}

.timeline-article-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1.25rem;
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
  margin-top: 0.5rem;
}

.timeline-article-date,
.timeline-article-category,
.timeline-article-views {
  display: flex;
  align-items: center;
  transition: color 0.2s ease;
}

.timeline-content:hover .timeline-article-date,
.timeline-content:hover .timeline-article-category,
.timeline-content:hover .timeline-article-views {
  color: var(--bs-primary);
}

.timeline-article-date i,
.timeline-article-category i,
.timeline-article-views i {
  margin-right: 0.5rem;
  font-size: 0.85rem;
}

/* 文章归档时间线响应式设计 */
@media (max-width: 768px) {
  .timeline-title {
    font-size: 1.3rem;
    margin-bottom: 1.5rem;
  }
  
  .timeline-year-month {
    margin-bottom: 2rem;
  }
  
  .timeline-year-month-title {
    font-size: 1.1rem;
    padding-left: 1.5rem;
  }
  
  .timeline-year-month-title::before {
    width: 10px;
    height: 10px;
  }
  
  .timeline-item {
    padding-left: 1.25rem;
    margin-bottom: 1.25rem;
  }
  
  .timeline-content {
    padding: 1rem;
    border-radius: 0.5rem;
  }
  
  .timeline-article-title {
    font-size: 1rem;
    margin-bottom: 0.5rem;
  }
  
  .timeline-article-meta {
    font-size: 0.8rem;
    gap: 0.75rem;
  }
  
  .timeline-article-date i,
  .timeline-article-category i,
  .timeline-article-views i {
    font-size: 0.75rem;
  }
}

/* 文章归档时间线平板设备适配 */
@media (min-width: 769px) and (max-width: 1024px) {
  .timeline-title {
    font-size: 1.4rem;
  }
  
  .timeline-year-month-title {
    font-size: 1.2rem;
  }
  
  .timeline-content {
    padding: 1.1rem;
  }
  
  .timeline-article-title {
    font-size: 1.1rem;
  }
}

/* 分页样式 */
.pagination-container {
  margin-top: 2rem;
  margin-bottom: 2rem;
}

/* 分页响应式设计 */
@media (max-width: 768px) {
  .pagination-container {
    margin-top: 1.5rem;
    margin-bottom: 1.5rem;
  }
  
  .page-link {
    padding: 0.5rem 0.75rem;
    font-size: 0.85rem;
  }
}

/* 友链页面样式 */
/* 友链头部 */
.links-header {
  margin-bottom: 2rem;
}

.links-title {
  font-size: 1.8rem;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 0.5rem;
}

.links-description {
  font-size: 1rem;
  color: #6b7280;
}

/* 友链网格布局 */
.links-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

/* 友链卡片样式 */
.link-card {
  padding: 1.5rem;
  border-radius: var(--wx-radius-md);
  box-shadow: var(--wx-shadow-sm);
  transition: var(--wx-transition);
  text-align: center;
}

.link-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--wx-shadow-md);
}

/* 友链头像 */
.link-avatar-container {
  margin-bottom: 1rem;
}

.link-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--bs-border-color);
  transition: transform 0.3s ease;
}

.link-card:hover .link-avatar {
  transform: scale(1.05);
}

/* 友链名称 */
.link-name {
  font-size: 1.1rem;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 0.5rem;
}

/* 友链描述 */
.link-description {
  font-size: 0.9rem;
  color: #6b7280;
  margin-bottom: 1rem;
  line-height: 1.4;
}

/* 友链URL */
.link-url {
  font-size: 0.85rem;
  color: var(--bs-primary);
  text-decoration: none;
  transition: color 0.2s ease;
  word-break: break-all;
}

.link-url:hover {
  color: #2563eb;
  text-decoration: underline;
}

/* 友链页面响应式设计 */
@media (max-width: 768px) {
  .links-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .link-card {
    padding: 1.25rem;
  }

  .links-title {
    font-size: 1.5rem;
  }

  .links-description {
    font-size: 0.9rem;
  }

  .link-name {
    font-size: 1rem;
  }

  .link-description {
    font-size: 0.85rem;
  }

  .link-url {
    font-size: 0.8rem;
  }
}

/* 友链页面平板设备适配 */
@media (min-width: 769px) and (max-width: 1024px) {
  .links-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* 友链页面样式 */
/* 页面标题样式 */
.article-title {
  font-size: clamp(1.8rem, 5vw, 2.5rem);
  line-height: 1.3;
  font-weight: 700;
}

/* 友链分组标题 */
.link-cate {
  font-size: 1.2rem;
  font-weight: 600;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid var(--bs-border-color);
  margin-bottom: 1rem !important;
  color: var(--bs-body-color);
}

/* 核心：现代化友链卡片样式 */
.link-card {
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-lg);
  transition: var(--wx-transition);
  background-color: var(--bs-body-bg);
  min-height: 120px;
}

/* 卡片悬浮效果 */
.link-card:hover {
  box-shadow: var(--wx-shadow-lg);
  border-color: rgba(var(--bs-primary-rgb), 0.3);
  transform: translateY(-2px);
}

/* 头像样式（现代化卡片中适当放大） */
.link-avatar {
  width: 56px;
  height: 56px;
  object-fit: cover;
  border: 2px solid var(--bs-tertiary-bg);
}

/* 名称样式 */
.link-name {
  font-weight: 600;
  color: var(--bs-body-color);
  transition: color 0.2s ease;
}

.link-name:hover {
  color: var(--bs-primary);
}

/* 描述样式 - 现代化文本截断 */
.link-desc {
  line-height: 1.5;
  font-size: 0.875rem;
  overflow: hidden;
  text-align: left;
  color: var(--bs-secondary-color);
  /* 标准属性 + 多浏览器前缀兼容 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  display: -moz-box;
  -moz-line-clamp: 2;
  -moz-box-orient: vertical;
  display: box;
  box-orient: vertical;
}

/* 通用Markdown样式 */
.article-content h2 {
  font-size: 1.5rem;
  margin: 1.5rem 0 1rem;
  font-weight: 600;
  border-bottom: 1px solid var(--bs-border-color);
  padding-bottom: 0.5rem;
  color: var(--bs-body-color);
}

.article-content a {
  color: var(--bs-link-color);
  text-decoration: none;
}

.article-content a:hover {
  color: var(--bs-link-hover-color);
  text-decoration: underline;
}

.article-content p {
  margin-bottom: 1rem;
  color: var(--bs-body-color);
}

/* 加载状态 */
.py-10 {
  padding-top: 10rem;
  padding-bottom: 10rem;
}

/* 响应式适配 - 长方形卡片专属 */
@media (max-width: 992px) {
  .article-title {
    font-size: clamp(1.5rem, 5vw, 2rem);
  }
  .article-content h2 {
    font-size: 1.25rem;
  }
  .link-cate {
    font-size: 1.1rem;
  }
  .link-avatar {
    width: 50px;
    height: 50px;
  }
  .link-card {
    min-height: 100px;
    padding: 1.5rem !important;
  }
}

@media (max-width: 768px) {
  .link-cate {
    font-size: 1rem;
  }
  .link-avatar {
    width: 45px;
    height: 45px;
  }
  .link-card {
    min-height: 90px;
    padding: 1.2rem !important;
  }
  .link-name {
    font-size: 1rem !important;
  }
  /* 响应式调整文本截断字号 */
  .link-desc {
    font-size: 0.85rem;
  }
}

@media (max-width: 576px) {
  .link-avatar {
    width: 40px;
    height: 40px;
  }
  .link-card {
    min-height: 80px;
    padding: 1rem !important;
  }
  .link-cate {
    font-size: 0.95rem;
  }
  .link-desc {
    font-size: 0.85rem;
  }
  /* 手机端单列展示 */
  .link-list-container .row > div {
    flex: 0 0 100%;
    max-width: 100%;
  }
}

/* 友链小卡片样式 */
.link-card-small {
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  padding: 1rem;
  background-color: var(--bs-body-bg);
  transition: var(--wx-transition);
}

.link-card-small:hover {
  transform: translateY(-2px);
  box-shadow: var(--wx-shadow-md);
  border-color: var(--bs-primary);
}

.link-card-small .link-name {
  font-size: 1rem;
  font-weight: 600;
  color: var(--bs-body-color);
}

/* 友链状态徽章 */
.status-badge {
  display: flex;
  align-items: center;
  font-size: 0.75rem;
  padding: 0.125rem 0.5rem;
  border-radius: 0.25rem;
}

.status-online {
  color: #10b981;
  background-color: rgba(16, 185, 129, 0.1);
}

.status-offline {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.1);
}

.response-time {
  margin-left: 0.25rem;
  opacity: 0.8;
}

.link-card-small .link-desc {
  font-size: 0.875rem;
  color: var(--bs-secondary-color);
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.link-card-small .link-url {
  font-size: 0.75rem;
  color: var(--bs-tertiary-color);
}

/* 留言板页面样式 */
/* 留言统计区域 */
.message-stats .fs-3 {
  font-weight: 700;
  color: var(--bs-body-color);
}

.message-stats .text-muted {
  font-size: 0.875rem;
}

/* 留言区域 */
.article-comment {
  margin-top: 2rem;
  margin-bottom: 3rem;
}

/* 留言板响应式设计 */
@media (max-width: 768px) {
  .message-stats .card {
    padding: 1.5rem !important;
  }
  
  .message-stats .fs-3 {
    font-size: 1.5rem !important;
  }
  
  .message-stats .text-muted {
    font-size: 0.8rem;
  }
  
  .article-comment {
    margin-top: 1.5rem;
    margin-bottom: 2rem;
  }
}

@media (max-width: 576px) {
  .message-stats .card {
    padding: 1.25rem !important;
  }
  
  .message-stats .fs-3 {
    font-size: 1.25rem !important;
  }
  
  .message-stats .text-muted {
    font-size: 0.75rem;
  }
}

/* 网格式留言板样式 */
.messages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 25px;
  padding: 20px 15px;
  min-height: 500px;
}

.message-item {
  transition: all 0.3s ease;
  transform: rotate(-1deg);
}

.message-item:nth-child(even) {
  transform: rotate(1deg);
}

.message-item:nth-child(3n) {
  transform: rotate(-0.5deg);
}

.message-item:nth-child(4n) {
  transform: rotate(0.5deg);
}

/* 便利贴样式增强 */
.sticky-note {
  border-radius: var(--wx-radius-md);
  box-shadow: var(--wx-shadow-sm);
  transition: var(--wx-transition);
  cursor: move;
  background-color: #fff9c4;
  border: none;
  min-height: 180px;
  position: relative;
  overflow: hidden;
}

.sticky-note::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 20px;
  background: linear-gradient(135deg, #ffeb3b, #fbc02d);
  border-bottom: 1px solid #f57f17;
}

.sticky-note .card-body {
  padding: 25px 20px 20px;
  position: relative;
  z-index: 1;
}

.sticky-note:hover {
  box-shadow: var(--wx-shadow-md);
  transform: translateY(-2px) rotate(0deg) !important;
}

/* 拖拽相关样式 */
.sortable-ghost {
  opacity: 0.5;
  background: #fff9c4;
  transform: rotate(0deg) !important;
}

.sortable-chosen {
  background: #fff9c4;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
}

.sortable-drag {
  opacity: 0.9;
  z-index: 1000;
  transform: rotate(5deg) scale(1.02) !important;
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.25);
}

/* 响应式网格布局 */
@media (max-width: 768px) {
  .messages-grid {
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 20px;
  }
  
  .sticky-note {
    min-height: 160px;
  }
}

@media (max-width: 576px) {
  .messages-grid {
    grid-template-columns: 1fr;
    gap: 15px;
  }
  
  .sticky-note {
    min-height: 140px;
  }
}

/* 留言板样式 */
/* 留言显示模块 */
.message-display {
  margin-bottom: 2rem;
}

.message-display .card-header {
  padding: 0.75rem 1.25rem;
}

.message-display .card-body {
  padding: 1.25rem;
}

/* 便利贴样式 */
.sticky-note {
  position: relative;
  border-radius: var(--wx-radius-md);
  background-color: #fff9c4;
  border: 1px solid var(--bs-border-color);
  box-shadow: var(--wx-shadow-sm);
}

/* 留言头像 */
.message-avatar {
  border: 2px solid rgba(var(--bs-primary-rgb), 0.1);
}

.message-avatar:hover {
  border-color: rgba(var(--bs-primary-rgb), 0.3);
}

/* 留言内容 */
.message-content {
  line-height: 1.6;
  font-size: 0.95rem;
  padding: 0.75rem;
  border-radius: var(--wx-radius-sm);
  background-color: rgba(255, 255, 255, 0.5);
}

.message-content:hover {
  background-color: rgba(255, 255, 255, 0.5);
}

/* 暗黑模式适配：颜色已由 CSS 变量自动切换，仅处理便利贴特殊背景 */
[data-bs-theme=dark] {
  .sticky-note {
    background-color: #33332a;
    border: 1px solid var(--bs-border-color);
    box-shadow: var(--wx-shadow-md);
  }

  .sticky-note:hover {
    box-shadow: var(--wx-shadow-lg);
  }

  .message-content {
    background-color: rgba(255, 255, 255, 0.08);
  }

  .message-content:hover {
    background-color: rgba(255, 255, 255, 0.12);
  }
}

/* 回复图标 */
.reply-icon-container {
  transition: all 0.3s ease;
}

.reply-icon {
  transition: all 0.3s ease;
  border-radius: 50%;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}

.reply-icon:hover {
  transform: scale(1.1);
  background-color: rgba(var(--bs-primary-rgb), 0.1);
}

/* 操作按钮 */
.message-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.reply-btn {
  transition: all 0.3s ease;
}

.reply-btn:hover {
  transform: translateY(-1px);
}

/* 回复弹窗样式 */
#replyModal .modal-content {
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
}

#replyModal .modal-header {
  border-bottom: 1px solid var(--bs-border-color);
}

#replyModal .modal-footer {
  border-top: 1px solid var(--bs-border-color);
}

#replyModal .reply-item {
  background-color: rgba(var(--bs-primary-rgb), 0.05);
}

#replyModal .reply-item:hover {
  background-color: rgba(var(--bs-primary-rgb), 0.1);
}

/* 回复输入框 */
.reply-form {
  border-radius: var(--wx-radius-sm);
  padding: 1rem;
  background-color: rgba(var(--bs-primary-rgb), 0.02);
}

/* 回复项 */
.reply-item {
  border-left: 3px solid rgba(var(--bs-primary-rgb), 0.2);
  padding-left: 1rem;
  margin-top: 0.75rem;
}

.reply-item:hover {
  border-left-color: rgba(var(--bs-primary-rgb), 0.4);
}

/* @提及样式 */
:deep(.at-mention) {
  color: var(--bs-primary);
  font-weight: 600;
  text-decoration: none;
}

/* 发布留言按钮 */
.publish-btn {
  transition: all 0.3s ease;
  font-weight: 500;
  letter-spacing: 0.5px;
}

.publish-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(var(--bs-primary-rgb), 0.3);
}

.publish-btn:disabled {
  opacity: 0.6;
  transform: none;
  box-shadow: none;
}

/* 表情功能 */
.emoji-button {
  transition: all 0.3s ease;
  z-index: 10;
}

.emoji-button:hover {
  transform: scale(1.05);
  border-color: var(--bs-primary);
}

/* 加载动画 */
.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 留言板响应式设计 */
@media (max-width: 768px) {
  .message-display .card {
    border-radius: 0.5rem;
  }
  
  .message-item {
    padding: 0.75rem;
    margin-bottom: 0.75rem;
  }
  
  .message-item p {
    padding: 0.5rem;
    font-size: 0.9rem;
  }
  
  .message-avatar {
    width: 40px !important;
    height: 40px !important;
  }
  
  .reply-item {
    margin-left: 0.75rem;
    padding-left: 0.75rem;
  }
  
  .emoji-grid {
    gap: 4px;
    padding: 4px;
  }

  .emoji-btn {
    min-width: 28px;
    min-height: 28px;
    padding: 3px 6px;
    font-size: 12px;
    border-radius: 6px;
  }
}
</style>