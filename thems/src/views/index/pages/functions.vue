<!-- 站点配置页面 -->
<template>
  <div class="config-page mt-2">
    <!-- 页面头部 -->
    <div class="wx-card">
      <div class="card-body">
        <div class="wx-page-header mb-0">
          <div>
            <h1 class="wx-page-title">
              <i class="bi bi-gear-wide-connected"></i>
              站点配置
            </h1>
            <p class="text-muted small mb-0 mt-1">管理网站全局设置、评论配置、自定义代码</p>
          </div>
        </div>
      </div>
    </div>

    <!-- 权限检查 -->
    <div v-if="!isAdmin" class="wx-card mt-2">
      <div class="card-body text-center py-5 px-4">
        <div class="mb-4">
          <i class="bi bi-shield-lock text-danger fs-4"></i>
        </div>
        <h3 class="h5 mb-3">权限不足</h3>
        <p class="text-muted mb-4">您没有权限访问此页面，请联系管理员</p>
        <router-link to="/" class="wx-btn-gradient">
          <i class="bi bi-house me-2"></i>
          返回首页
        </router-link>
      </div>
    </div>

    <!-- 配置内容 -->
    <div v-else-if="isAdmin">
      <!-- 导航标签与内容在同一容器 -->
      <div class="wx-card mt-2">
        <div class="card-header">
          <ul class="nav nav-pills card-header-pills" id="configTabs" role="tablist">
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link active" 
                id="global-tab" 
                data-bs-toggle="pill" 
                data-bs-target="#global" 
                type="button" 
                role="tab" 
                aria-controls="global" 
                aria-selected="true"
              >
                <i class="bi bi-globe2 me-2"></i>
                全局设置
              </button>
            </li>
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link" 
                id="comment-tab" 
                data-bs-toggle="pill" 
                data-bs-target="#comment" 
                type="button" 
                role="tab" 
                aria-controls="comment" 
                aria-selected="false"
              >
                <i class="bi bi-chat-square-text me-2"></i>
                评论设置
              </button>
            </li>
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link" 
                id="article-tab" 
                data-bs-toggle="pill" 
                data-bs-target="#article" 
                type="button" 
                role="tab" 
                aria-controls="article" 
                aria-selected="false"
              >
                <i class="bi bi-file-earmark-richtext me-2"></i>
                文章设置
              </button>
            </li>
            <li class="nav-item" role="presentation">
              <button 
                class="nav-link" 
                id="custom-code-tab" 
                data-bs-toggle="pill" 
                data-bs-target="#custom-code" 
                type="button" 
                role="tab" 
                aria-controls="custom-code" 
                aria-selected="false"
              >
                <i class="bi bi-code-square me-2"></i>
                自定义代码
              </button>
            </li>
          </ul>
        </div>
        <div class="card-body">
          <!-- 标签内容 -->
          <div class="tab-content" id="configTabsContent">
            <!-- 全局设置 -->
            <div 
              class="tab-pane fade show active" 
              id="global" 
              role="tabpanel" 
              aria-labelledby="global-tab"
            >
              <form>
                <!-- 网站基本信息 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-info-circle-fill"></i>网站基本信息</h2>
                  <div class="d-flex flex-column gap-3">
                    <div class="fs-field">
                      <label for="site-title" class="fs-label">网站标题</label>
                      <input type="text" class="wx-form-control" id="site-title" v-model="globalConfig.title" placeholder="输入您的网站标题">
                      <div class="fs-hint">显示在浏览器标签和网站头部</div>
                    </div>
                    <div class="fs-field">
                      <label for="site-description" class="fs-label">网站描述</label>
                      <input type="text" class="wx-form-control" id="site-description" v-model="globalConfig.description" placeholder="输入您的网站描述">
                      <div class="fs-hint">用于SEO和社交媒体分享</div>
                    </div>
                    <div class="fs-field">
                      <label for="site-keyword" class="fs-label">网站关键词</label>
                      <input type="text" class="wx-form-control" id="site-keyword" v-model="globalConfig.keyword" placeholder="输入关键词，用逗号分隔">
                      <div class="fs-hint">用于SEO优化，多个关键词用逗号分隔</div>
                    </div>
                    <div class="fs-field">
                      <label for="site-avatar" class="fs-label">网站LOGO</label>
                      <input type="text" class="wx-form-control" id="site-avatar" v-model="globalConfig.avatar" placeholder="输入LOGO URL">
                      <div class="fs-hint">网站的LOGO图片</div>
                    </div>
                    <div class="fs-field">
                      <label for="site-favicon" class="fs-label">网站图标</label>
                      <input type="text" class="wx-form-control" id="site-favicon" v-model="globalConfig.favicon" placeholder="输入favicon URL">
                      <div class="fs-hint">浏览器标签显示的图标</div>
                    </div>
                    <div class="fs-field">
                      <label for="site-date" class="fs-label">建站日期</label>
                      <input type="date" class="wx-form-control" id="site-date" :value="formatDate(globalConfig.date)" @change="handleDateChange">
                      <div class="fs-hint">网站的建立日期</div>
                    </div>
                  </div>
                </div>

                <!-- 显示设置 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-display-fill"></i>显示设置</h2>
                  <div class="d-flex flex-column gap-3">
                    <div class="fs-field">
                      <label class="fs-label">文章列表布局</label>
                      <select class="wx-form-control" v-model="globalConfig.display_mode">
                        <option value="grid">网格卡片模式</option>
                        <option value="list">列表模式</option>
                        <option value="horizontal">横向图文模式</option>
                      </select>
                      <div class="fs-hint">选择文章列表卡片的显示模式</div>
                    </div>
                    <div>
                      <div class="fs-switch-row">
                        <div>
                          <label class="fs-label">快速发布文章</label>
                          <div class="fs-hint">是否在首页显示快速发布文章模块</div>
                        </div>
                        <div class="form-check form-switch">
                          <input class="form-check-input" type="checkbox" id="quick_publish_switch" v-model="globalConfig.quick_publish">
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 自定义导航栏 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-bookmark-star-fill"></i>自定义导航栏</h2>
                  <div class="d-flex flex-column gap-3">
                    <div class="fs-field">
                      <label for="custom-nav-links" class="fs-label">自定义导航链接</label>
                      <textarea class="wx-form-control wx-code-textarea" id="custom-nav-links" rows="6" v-model="globalConfig.custom_nav_links" placeholder="百度一下 || https://baidu.com&#10;腾讯视频 || https://v.qq.com"></textarea>
                      <div class="fs-hint">
                        用于自定义导航栏链接。格式：<code>跳转文字 || 跳转链接</code>（中间使用两个竖杠分隔），一行一个。
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 备案信息 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-shield-check"></i>备案信息</h2>
                  <div class="d-flex flex-column gap-3">
                    <div class="fs-field">
                      <label for="copy-code" class="fs-label">ICP 备案编号</label>
                      <input type="text" class="wx-form-control" id="copy-code" v-model="globalConfig.copy.code" placeholder="输入 ICP 备案编号">
                      <div class="fs-hint">ICP 备案编号</div>
                    </div>
                    <div class="fs-field">
                      <label for="copy-link" class="fs-label">ICP 备案链接</label>
                      <input type="text" class="wx-form-control" id="copy-link" v-model="globalConfig.copy.link" placeholder="输入备案链接">
                      <div class="fs-hint">备案查询链接</div>
                    </div>
                    <div class="fs-field">
                      <label for="police-code" class="fs-label">公安联网备案编号</label>
                      <input type="text" class="wx-form-control" id="police-code" v-model="globalConfig.police.code" placeholder="输入公安联网备案编号">
                      <div class="fs-hint">公安联网备案编号</div>
                    </div>
                    <div class="fs-field">
                      <label for="police-link" class="fs-label">公安备案链接</label>
                      <input type="text" class="wx-form-control" id="police-link" v-model="globalConfig.police.link" placeholder="输入公安备案链接">
                      <div class="fs-hint">公安备案查询链接</div>
                    </div>
                  </div>
                </div>

                <!-- 登录协议提示配置 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-file-earmark-text-fill"></i>登录协议提示配置</h2>
                  <div class="d-flex flex-column gap-3">
                    <div>
                      <div class="fs-switch-row">
                        <div>
                          <label class="fs-label">启用登录协议提示</label>
                          <div class="fs-hint">在登录/注册/找回密码弹窗中显示协议同意提示</div>
                        </div>
                        <div class="form-check form-switch">
                          <input class="form-check-input" type="checkbox" id="auth_dialog_agreement_enabled_switch" v-model="globalConfig.auth_dialog_agreement.enabled">
                        </div>
                      </div>
                    </div>
                    <div class="fs-field" v-if="globalConfig.auth_dialog_agreement.enabled">
                      <label for="auth_dialog_agreement_user_content" class="fs-label">用户协议内容</label>
                      <textarea class="wx-form-control" id="auth_dialog_agreement_user_content" v-model="globalConfig.auth_dialog_agreement.user_agreement_content" rows="4" placeholder="输入用户协议内容"></textarea>
                      <div class="fs-hint">点击《用户协议》时弹窗显示此内容</div>
                    </div>
                    <div class="fs-field" v-if="globalConfig.auth_dialog_agreement.enabled">
                      <label for="auth_dialog_agreement_privacy_content" class="fs-label">隐私协议内容</label>
                      <textarea class="wx-form-control" id="auth_dialog_agreement_privacy_content" v-model="globalConfig.auth_dialog_agreement.privacy_agreement_content" rows="4" placeholder="输入隐私协议内容"></textarea>
                      <div class="fs-hint">点击《隐私协议》时弹窗显示此内容</div>
                    </div>
                  </div>
                </div>

                <!-- 悬浮按钮设置 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-arrows-fullscreen"></i>右侧悬浮按钮设置</h2>
                  <div class="d-flex flex-column gap-3">
                    <div>
                      <div class="fs-switch-row">
                        <div>
                          <label class="fs-label">启用悬浮按钮</label>
                          <div class="fs-hint">在网站右侧显示悬浮按钮</div>
                        </div>
                        <div class="form-check form-switch">
                          <input class="form-check-input" type="checkbox" id="float_buttons_enabled_switch" v-model="globalConfig.float_buttons.enabled">
                        </div>
                      </div>
                    </div>

                    <div v-if="globalConfig.float_buttons.enabled">
                      <div class="fs-switch-row fs-switch-row-compact">
                        <label for="float_buttons_show_back_to_top" class="fs-label mb-0">显示返回顶部按钮</label>
                        <div class="form-check form-switch mb-0">
                          <input class="form-check-input" type="checkbox" id="float_buttons_show_back_to_top" v-model="globalConfig.float_buttons.show_back_to_top">
                        </div>
                      </div>
                    </div>

                    <div v-if="globalConfig.float_buttons.enabled">
                      <div class="fs-switch-row fs-switch-row-compact">
                        <label for="float_buttons_show_notice" class="fs-label mb-0">显示公告按钮</label>
                        <div class="form-check form-switch mb-0">
                          <input class="form-check-input" type="checkbox" id="float_buttons_show_notice" v-model="globalConfig.float_buttons.show_notice">
                        </div>
                      </div>
                    </div>

                    <div class="fs-field" v-if="globalConfig.float_buttons.enabled">
                      <label for="float_buttons_style" class="fs-label">按钮样式</label>
                      <select class="wx-form-control" id="float_buttons_style" v-model="globalConfig.float_buttons.style">
                        <option value="rounded">圆角按钮</option>
                        <option value="square">方形按钮</option>
                      </select>
                    </div>

                    <div class="fs-field" v-if="globalConfig.float_buttons.enabled">
                      <label for="float_buttons_position" class="fs-label">显示位置</label>
                      <select class="wx-form-control" id="float_buttons_position" v-model="globalConfig.float_buttons.position">
                        <option value="center">右侧居中</option>
                        <option value="bottom">右侧底部</option>
                      </select>
                    </div>

                    <div v-if="globalConfig.float_buttons.enabled">
                      <div class="d-flex justify-content-between align-items-center mb-2">
                        <label class="fs-label mb-0">悬浮按钮管理</label>
                        <button type="button" class="wx-btn-outline" @click="addFloatButton">
                          <i class="bi bi-plus-lg me-1"></i>添加
                        </button>
                      </div>

                      <div class="d-flex flex-column gap-2">
                        <div v-for="(button, index) in globalConfig.float_buttons.buttons" :key="button.id" class="float-btn-item">
                          <div class="float-btn-header">
                            <h4>{{ button.name }}</h4>
                            <div class="float-btn-actions">
                              <button type="button" class="wx-btn-outline" @click="moveFloatButton(index, 'up')" :disabled="index === 0">
                                <i class="bi bi-arrow-up"></i>
                              </button>
                              <button type="button" class="wx-btn-outline" @click="moveFloatButton(index, 'down')" :disabled="index === globalConfig.float_buttons.buttons.length - 1">
                                <i class="bi bi-arrow-down"></i>
                              </button>
                              <button type="button" class="btn btn-sm btn-outline-danger" @click="removeFloatButton(index)">
                                <i class="bi bi-trash"></i>
                              </button>
                            </div>
                          </div>

                          <div class="d-flex flex-column gap-2">
                            <div class="fs-switch-row fs-switch-row-compact">
                              <label :for="'float-button-enabled-' + button.id" class="fs-label mb-0">启用</label>
                              <div class="form-check form-switch mb-0">
                                <input class="form-check-input" type="checkbox" :id="'float-button-enabled-' + button.id" v-model="button.enabled">
                              </div>
                            </div>
                            <div class="fs-field">
                              <label :for="'float-button-name-' + button.id" class="fs-label">按钮名称</label>
                              <input type="text" class="wx-form-control" :id="'float-button-name-' + button.id" v-model="button.name" placeholder="输入按钮名称">
                            </div>
                            <div class="fs-field">
                              <label :for="'float-button-icon-' + button.id" class="fs-label">按钮图标</label>
                              <input type="text" class="wx-form-control" :id="'float-button-icon-' + button.id" v-model="button.icon" placeholder="例如：'bi bi-qq'">
                            </div>
                            <div class="fs-field">
                              <label :for="'float-button-url-' + button.id" class="fs-label">按钮链接</label>
                              <input type="text" class="wx-form-control" :id="'float-button-url-' + button.id" v-model="button.url" placeholder="输入按钮链接">
                            </div>
                            <div class="fs-field">
                              <label :for="'float-button-tooltip-' + button.id" class="fs-label">按钮简介</label>
                              <input type="text" class="wx-form-control" :id="'float-button-tooltip-' + button.id" v-model="button.tooltip" placeholder="鼠标悬停时显示">
                            </div>
                            <div class="fs-field">
                              <label :for="'float-button-image-url-' + button.id" class="fs-label">图片链接</label>
                              <input type="text" class="wx-form-control" :id="'float-button-image-url-' + button.id" v-model="button.image_url" placeholder="鼠标移到此处的图片">
                              <div class="fs-hint">设置此处后，按钮链接将失效</div>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 保存按钮 -->
                <div class="fs-actions">
                  <button type="button" class="wx-btn-gradient" @click="saveGlobalConfig" :disabled="saving">
                    <i class="bi" :class="saving ? 'bi-arrow-clockwise spin' : 'bi-check-lg'"></i>
                    {{ saving ? '保存中..' : '保存设置' }}
                  </button>
                  <button type="button" class="wx-btn-outline" @click="resetGlobalConfig" :disabled="saving">
                    <i class="bi bi-arrow-counterclockwise me-1"></i>
                    重置
                  </button>
                </div>
              </form>
            </div>

            <!-- 评论设置 -->
            <div 
              class="tab-pane fade" 
              id="comment" 
              role="tabpanel" 
              aria-labelledby="comment-tab"
            >
              <form>
                <!-- 全局评论开关 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-power"></i>全局设置</h2>
                  <div class="form-check mb-2">
                    <input class="form-check-input" type="checkbox" id="comment_enabled" v-model="commentConfig.enabled">
                    <label class="form-check-label" for="comment_enabled">启用全局评论功能</label>
                  </div>
                  <div class="fs-hint">关闭后，所有文章的评论模块将完全不显示</div>
                </div>

                <!-- 速率限制 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-speedometer2"></i>速率限制</h2>
                  <div class="form-check mb-3">
                    <input class="form-check-input" type="checkbox" id="rate_limit_enabled" v-model="commentConfig.rate_limit.enabled">
                    <label class="form-check-label" for="rate_limit_enabled">启用速率限制，防止频繁评论</label>
                  </div>
                  <div class="d-flex flex-column gap-3" v-if="commentConfig.rate_limit.enabled">
                    <div class="fs-field">
                      <label for="rate_limit_max_count" class="fs-label">最大评论数</label>
                      <div class="input-group">
                        <input type="number" class="wx-form-control" id="rate_limit_max_count" v-model.number="commentConfig.rate_limit.max_count" min="1" max="100">
                        <span class="input-group-text">次</span>
                      </div>
                      <div class="fs-hint">时间窗口内允许的最大评论数</div>
                    </div>
                    <div class="fs-field">
                      <label for="rate_limit_time_window" class="fs-label">时间窗口</label>
                      <div class="input-group">
                        <input type="number" class="wx-form-control" id="rate_limit_time_window" v-model.number="commentConfig.rate_limit.time_window" min="1" max="3600">
                        <span class="input-group-text">秒</span>
                      </div>
                      <div class="fs-hint">速率限制的时间窗口</div>
                    </div>
                  </div>
                </div>

                <!-- 评论长度限制 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-rulers"></i>评论长度</h2>
                  <div class="d-flex flex-column gap-3">
                    <div class="fs-field">
                      <label for="max_length" class="fs-label">最大长度</label>
                      <div class="input-group">
                        <input type="number" class="wx-form-control" id="max_length" v-model.number="commentConfig.max_length" min="1" max="10000">
                        <span class="input-group-text">字</span>
                      </div>
                      <div class="fs-hint">单条评论的最大长度</div>
                    </div>
                  </div>
                </div>

                <!-- 内容要求 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-check2-square"></i>内容要求</h2>
                  <div class="form-check mb-2">
                    <input class="form-check-input" type="checkbox" id="require_chinese" v-model="commentConfig.require_chinese">
                    <label class="form-check-label" for="require_chinese">要求评论内容包含中文</label>
                  </div>
                  <div class="form-check mb-2">
                    <input class="form-check-input" type="checkbox" id="sensitive_filter" v-model="commentConfig.sensitive_filter">
                    <label class="form-check-label" for="sensitive_filter">启用敏感词过滤</label>
                  </div>
                </div>

                <!-- 敏感词 -->
                <div class="mb-4" v-if="commentConfig.sensitive_filter">
                  <h2 class="fs-section-title"><i class="bi bi-shield-exclamation"></i>敏感词设置</h2>
                  <div class="fs-field">
                    <label for="sensitive_words" class="fs-label">敏感词列表</label>
                    <textarea class="wx-form-control" id="sensitive_words" v-model="commentConfig.sensitive_words" rows="3" placeholder="多个敏感词用「，」分隔"></textarea>
                    <div class="fs-hint">评论中包含这些词将被拒绝</div>
                  </div>
                </div>

                <!-- 邮件通知 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-envelope-paper"></i>邮件通知</h2>
                  <div class="form-check mb-3">
                    <input class="form-check-input" type="checkbox" id="email_notify_enabled" v-model="commentConfig.email_notify.enabled">
                    <label class="form-check-label" for="email_notify_enabled">启用邮件通知</label>
                  </div>
                  <div class="d-flex flex-column gap-3" v-if="commentConfig.email_notify.enabled">
                    <div class="fs-field">
                      <label for="email_notify_retry_count" class="fs-label">重试次数</label>
                      <div class="input-group">
                        <input type="number" class="wx-form-control" id="email_notify_retry_count" v-model.number="commentConfig.email_notify.retry_count" min="1" max="10">
                        <span class="input-group-text">次</span>
                      </div>
                      <div class="fs-hint">发送失败时的重试次数</div>
                    </div>
                    <div class="fs-field">
                      <label for="email_notify_retry_interval" class="fs-label">重试间隔</label>
                      <div class="input-group">
                        <input type="number" class="wx-form-control" id="email_notify_retry_interval" v-model.number="commentConfig.email_notify.retry_interval" min="1" max="60">
                        <span class="input-group-text">秒</span>
                      </div>
                      <div class="fs-hint">每次重试的时间间隔</div>
                    </div>
                  </div>
                </div>

                <!-- 保存按钮 -->
                <div class="fs-actions">
                  <button type="button" class="wx-btn-gradient" @click="saveCommentConfig" :disabled="saving">
                    <i class="bi" :class="saving ? 'bi-arrow-clockwise spin' : 'bi-check-lg'"></i>
                    {{ saving ? '保存中..' : '保存设置' }}
                  </button>
                  <button type="button" class="wx-btn-outline" @click="resetCommentConfig" :disabled="saving">
                    <i class="bi bi-arrow-counterclockwise me-1"></i>
                    重置
                  </button>
                </div>
              </form>
            </div>

            <!-- 文章设置 -->
            <div class="tab-pane fade" id="article" role="tabpanel" aria-labelledby="article-tab">
              <div class="mb-4">
                <h2 class="fs-section-title"><i class="bi bi-gift-fill"></i>打赏设置</h2>
                <div class="d-flex flex-column gap-3">
                  <div>
                    <div class="fs-switch-row">
                      <div>
                        <label class="fs-label">启用打赏功能</label>
                        <div class="fs-hint">开启或关闭文章打赏功能</div>
                      </div>
                      <div class="form-check form-switch">
                        <input class="form-check-input" type="checkbox" id="reward_enabled_switch" v-model="globalConfig.reward.enabled">
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="d-flex flex-column gap-3 mb-4">
                <div class="fs-field">
                  <label for="reward_wechat" class="fs-label">微信收款码</label>
                  <input type="text" class="wx-form-control" id="reward_wechat" v-model="globalConfig.reward.wechat" placeholder="输入微信收款码图片链接">
                  <div class="fs-hint">建议使用正方形图片</div>
                </div>
                <div class="fs-field">
                  <label for="reward_alipay" class="fs-label">支付宝收款码</label>
                  <input type="text" class="wx-form-control" id="reward_alipay" v-model="globalConfig.reward.alipay" placeholder="输入支付宝收款码图片链接">
                  <div class="fs-hint">建议使用正方形图片</div>
                </div>
              </div>

              <!-- 保存按钮 -->
              <div class="fs-actions">
                <button type="button" class="wx-btn-gradient" @click="saveGlobalConfig" :disabled="saving">
                  <i class="bi" :class="saving ? 'bi-arrow-clockwise spin' : 'bi-check-lg'"></i>
                  {{ saving ? '保存中..' : '保存设置' }}
                </button>
                <button type="button" class="wx-btn-outline" @click="resetGlobalConfig" :disabled="saving">
                  <i class="bi bi-arrow-counterclockwise me-1"></i>
                  重置
                </button>
              </div>
            </div>

            <!-- 自定义代码 -->
            <div class="tab-pane fade" id="custom-code" role="tabpanel" aria-labelledby="custom-code-tab">
              <form>
                <!-- CSS代码 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-palette2"></i>CSS代码</h2>
                  <div class="fs-field">
                    <textarea class="wx-form-control wx-code-textarea" id="custom-css" v-model="customCodeConfig.css" rows="4" placeholder="在这里输入自定义CSS代码"></textarea>
                    <div class="fs-hint">自定义CSS样式，会全局生效</div>
                  </div>
                </div>

                <!-- JavaScript代码 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-braces"></i>JavaScript代码</h2>
                  <div class="fs-field">
                    <textarea class="wx-form-control wx-code-textarea" id="custom-js" v-model="customCodeConfig.js" rows="4" placeholder="在这里输入自定义JavaScript代码"></textarea>
                    <div class="fs-hint">自定义JavaScript脚本，会在页面加载时执行</div>
                  </div>
                </div>

                <!-- 头部HTML代码 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-layout-text-window-reverse"></i>头部HTML代码</h2>
                  <div class="fs-field">
                    <textarea class="wx-form-control wx-code-textarea" id="custom-header" v-model="customCodeConfig.header" rows="3" placeholder="在这里输入自定义头部HTML代码"></textarea>
                    <div class="fs-hint">会被插入到HTML的head标签中</div>
                  </div>
                </div>

                <!-- 底部HTML代码 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-layout-text-window"></i>底部HTML代码</h2>
                  <div class="fs-field">
                    <textarea class="wx-form-control wx-code-textarea" id="custom-footer" v-model="customCodeConfig.footer" rows="3" placeholder="在这里输入自定义底部HTML代码"></textarea>
                    <div class="fs-hint">会被插入到HTML的body标签末尾</div>
                  </div>
                </div>

                <!-- 网站统计代码 -->
                <div class="mb-4">
                  <h2 class="fs-section-title"><i class="bi bi-graph-up-arrow"></i>网站统计代码</h2>
                  <div class="fs-field">
                    <textarea class="wx-form-control wx-code-textarea" id="custom-analytics" v-model="customCodeConfig.analytics" rows="3" placeholder="在这里输入网站统计代码"></textarea>
                    <div class="fs-hint">会被插入到HTML的body标签末尾</div>
                  </div>
                </div>

                <!-- 保存按钮 -->
                <div class="fs-actions">
                  <button type="button" class="wx-btn-gradient" @click="saveCustomCodeConfig" :disabled="saving">
                    <i class="bi" :class="saving ? 'bi-arrow-clockwise spin' : 'bi-check-lg'"></i>
                    {{ saving ? '保存中..' : '保存设置' }}
                  </button>
                  <button type="button" class="wx-btn-outline" @click="resetCustomCodeConfig" :disabled="saving">
                    <i class="bi bi-arrow-counterclockwise me-1"></i>
                    重置
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useCommStore } from '@/store/comm'
import { toast, usePageTitle } from '@/utils/app'
import { request, cache as networkCache } from '@/utils/network'

const store = useCommStore()

// 共享的xiao_functions配置缓存
let cachedXiaoFunctionsConfig = null
let cachedXiaoFunctionsKey = ''
let cachedXiaoFunctionsTime = 0
const CACHE_EXPIRE = 5 * 60 * 1000 // 5分钟缓存

// 获取cardify_functions配置的共享方法
async function getXiaoFunctionsConfig() {
  const now = Date.now()
  // 如果缓存存在且未过期，直接返回
  if (cachedXiaoFunctionsConfig && (now - cachedXiaoFunctionsTime) < CACHE_EXPIRE) {
    return cachedXiaoFunctionsConfig
  }
  
  const response = await request.get('/api/config/one', { key: 'cardify_functions' })
  if (response.code === 200 && response.data) {
    cachedXiaoFunctionsConfig = response.data.json || {}
    cachedXiaoFunctionsTime = now
    return cachedXiaoFunctionsConfig
  }
  return {}
}

// 清除cardify_functions缓存
function clearXiaoFunctionsCache() {
  cachedXiaoFunctionsConfig = null
  cachedXiaoFunctionsTime = 0
}

// 页面标题管理
const { setDynamicTitle } = usePageTitle({
  staticTitle: '站点配置',
  defaultTitle: '站点配置'
})

// 响应式数据
const commentConfig = ref({
  enabled: true, // 全局评论开关
  rate_limit: {
    enabled: true,
    max_count: 5,
    time_window: 60
  },
  max_length: 500,
  require_chinese: true,
  sensitive_filter: true,
  sensitive_words: '广告,开发,开发',
  email_notify: {
    enabled: true,
    retry_count: 3,
    retry_interval: 5
  }
})

const globalConfig = ref({
  title: '',
  description: '',
  keyword: '',
  avatar: '',
  favicon: '',
  date: Math.floor(Date.now() / 1000).toString(),
  display_mode: 'grid', // grid 为网格卡片模式，list 为列表模式，horizontal 为横向图文模式
  custom_nav_links: '', // 自定义导航链接（每行：文字 || 链接）
  copy: {
    code: '',
    link: 'http://beian.miit.gov.cn/'
  },
  police: {
    code: '',
    link: 'https://beian.mps.gov.cn/#/query/webSearch'
  },
  auth_dialog_agreement: {
    enabled: true, // 是否启用协议提示
    user_agreement_content: '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议：\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性，并对您账户下的所有活动负责。\n\n3. 用户行为规范\n您同意不会：\n- 发布违法、有害或侵犯他人权益的内容\n- 滥用或破坏服务\n- 未经授权访问他人账户\n\n4. 知识产权\n您发布的内容的知识产权归您所有，但我们有权在服务中使用、复制和传播。\n\n5. 服务变更\n我们保留随时修改或终止服务的权利。\n\n6. 免责声明\n我们不对服务的准确性、完整性或可靠性提供保证。\n\n7. 协议变更\n我们可能会更新本协议，您继续使用服务即表示接受更新后的协议。',
    privacy_agreement_content: '隐私协议\n\n我们重视您的隐私。以下是我们的隐私政策：\n\n1. 收集的信息\n我们可能收集以下信息：\n- 您的账户信息（用户名、邮箱、手机号）\n- 使用数据（访问记录、浏览行为）\n- 设备信息（IP地址、浏览器类型）\n\n2. 信息使用\n我们使用收集的信息来：\n- 提供和改进服务\n- 个性化您的体验\n- 发送重要通知\n- 保障服务安全\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。仅在以下情况下共享：\n- 法律要求\n- 保护我们的权益\n- 经您同意\n\n4. 数据安全\n我们采取合理措施保护您的数据，但无法保证绝对安全。\n\n5. 您的权利\n您有权访问、更正或删除您的个人信息。\n\n6. Cookie使用\n我们使用Cookie来改善您的体验，您可以在浏览器中禁用。\n\n7. 政策变更\n我们可能更新隐私政策，变更将在此页面发布',
  },
  float_buttons: {
    enabled: true,
    style: 'rounded', // rounded 或 square
    position: 'center', // center 或 bottom
    show_back_to_top: true,
    show_notice: true,
    buttons: [
      {
        id: 1,
        name: 'QQ',
        icon: 'bi bi-tencent-qq',
        url: 'https://qm.qq.com/q/56SYi6MAta',
        tooltip: '联系QQ',
        enabled: true
      },
      {
        id: 2,
        name: '微信',
        icon: 'bi bi-wechat',
        url: 'https://weixin.qq.com/',
        tooltip: '扫码添加微信',
        enabled: true,
        image_url: ''
      }
    ]
  },
  reward: {
    enabled: true,
    wechat: '',
    alipay: ''
  }
})

const customCodeConfig = ref({
  css: '',
  js: '',
  header: '',
  footer: '',
  analytics: ''
})

const saving = ref(false)
const message = ref('')
const messageType = ref('')

// 计算属性
// 是否为管理员
const isAdmin = computed(() => {
  const userInfo = store.login.user
  // 检查不同可能的用户信息结构
  const userAuth = userInfo?.result?.auth || userInfo?.auth
  const userGroups = userAuth?.group?.list || userInfo?.group?.list || []
  return userAuth?.all || userGroups.some(group => group.key === 'admin')
})

// 方法
// 获取评论配置
async function getCommentConfig() {
  try {
    const response = await request.get('/api/config/one', {
      key: 'COMMENT'
    })
    if (response.code === 200 && response.data) {
      const config = response.data.json || {}
      // 填充表单
      commentConfig.value = {
        enabled: config.enabled === 1, // 全局评论开关
        rate_limit: {
          enabled: config.rate_limit?.enabled === 1,
          max_count: config.rate_limit?.max_count || 5,
          time_window: config.rate_limit?.time_window || 60
        },
        max_length: config.max_length || 500,
        require_chinese: config.require_chinese === 1,
        sensitive_filter: config.sensitive_filter === 1,
        sensitive_words: config.sensitive_words?.join(',') || '广告,开发,开发',
        email_notify: {
          enabled: config.email_notify?.enabled === 1,
          retry_count: config.email_notify?.retry_count || 3,
          retry_interval: config.email_notify?.retry_interval || 5
        }
      }
    }
  } catch (error) {
    console.error('获取评论配置失败:', error)
    toast.error('获取评论配置失败')
  }
}

// 获取全局配置
async function getGlobalConfig() {
  try {
    const config = await getXiaoFunctionsConfig()
    // 填充表单
    // 兼容旧的布尔值格式和新的字符串格式
    let displayMode = config.display_mode || 'grid'
    if (typeof displayMode === 'boolean') {
      displayMode = displayMode ? 'grid' : 'list'
    }
    globalConfig.value = {
      title: config.title || '',
      description: config.description || '',
      keyword: config.keyword || '',
      avatar: config.avatar || '',
      favicon: config.favicon || '',
      date: config.date || Math.floor(Date.now() / 1000).toString(),
      display_mode: displayMode,
      quick_publish: config.quick_publish !== false,
      custom_nav_links: config.custom_nav_links || '', // 自定义导航链接
      copy: {
        code: config.copy?.code || '',
        link: config.copy?.link || 'http://beian.mps.gov.cn/'
      },
      police: {
        code: config.police?.code || '',
        link: config.police?.link || 'https://beian.mps.gov.cn/#/query/webSearch'
      },
      auth_dialog_agreement: {
        enabled: config.auth_dialog_agreement?.enabled !== false,
        user_agreement_content: config.auth_dialog_agreement?.user_agreement_content || '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议：\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性，并对您账户下的所有活动负责。\n\n3. 用户行为规范\n您同意不会：\n- 发布违法、有害或侵犯他人权益的内容\n- 滥用或破坏服务\n- 未经授权访问他人账户\n\n4. 知识产权\n您发布的内容的知识产权归您所有，但我们有权在服务中使用、复制和传播。\n\n5. 服务变更\n我们保留随时修改或终止服务的权利。\n\n6. 免责声明\n我们不对服务的准确性、完整性或可靠性提供保证。\n\n7. 协议变更\n我们可能会更新本协议，您继续使用服务即表示接受更新后的协议。',
        privacy_agreement_content: config.auth_dialog_agreement?.privacy_agreement_content || '隐私协议\n\n我们重视您的隐私。以下是我们的隐私政策：\n\n1. 收集的信息\n我们可能收集以下信息：\n- 您的账户信息（用户名、邮箱、手机号）\n- 使用数据（访问记录、浏览行为）\n- 设备信息（IP地址、浏览器类型）\n\n2. 信息使用\n我们使用收集的信息来：\n- 提供和改进服务\n- 个性化您的体验\n- 发送重要通知\n- 保障服务安全\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。仅在以下情况下共享：\n- 法律要求\n- 保护我们的权益\n- 经您同意\n\n4. 数据安全\n我们采取合理措施保护您的数据，但无法保证绝对安全。\n\n5. 您的权利\n您有权访问、更正或删除您的个人信息。\n\n6. Cookie使用\n我们使用Cookie来改善您的体验，您可以在浏览器中禁用。\n\n7. 政策变更\n我们可能更新隐私政策，变更将在此页面发布',
      },
      float_buttons: {
        enabled: config.float_buttons?.enabled !== false,
        style: config.float_buttons?.style || 'rounded',
        position: config.float_buttons?.position || 'center',
        show_back_to_top: config.float_buttons?.show_back_to_top !== false,
        show_notice: config.float_buttons?.show_notice !== false,
        buttons: config.float_buttons?.buttons || [
          {
            id: 1,
            name: 'QQ',
            icon: 'bi bi-tencent-qq',
            url: 'https://qm.qq.com/q/56SYi6MAta',
            tooltip: '加入交流群',
            enabled: true
          },
          {
            id: 2,
            name: '微信',
            icon: 'bi bi-wechat',
            url: 'https://weixin.qq.com/',
            tooltip: '扫码添加微信',
            enabled: true,
            image_url: ''
          }
        ]
      },
      reward: {
        enabled: config.reward?.enabled !== false,
        wechat: config.reward?.wechat || '',
        alipay: config.reward?.alipay || ''
      }
    }
  } catch (error) {
    toast.error('获取全局配置失败')
  }
}

// 保存评论配置
async function saveCommentConfig() {
  saving.value = true
  message.value = ''
  messageType.value = ''
  
  try {
    // 构建配置对象
    const config = {
      enabled: commentConfig.value.enabled ? 1 : 0, // 全局评论开关
      rate_limit: {
        enabled: commentConfig.value.rate_limit.enabled ? 1 : 0,
        max_count: commentConfig.value.rate_limit.max_count || 5,
        time_window: commentConfig.value.rate_limit.time_window || 60
      },
      max_length: commentConfig.value.max_length || 500,
      require_chinese: commentConfig.value.require_chinese ? 1 : 0,
      sensitive_filter: commentConfig.value.sensitive_filter ? 1 : 0,
      sensitive_words: commentConfig.value.sensitive_words
        .split(',')
        .map(word => word.trim())
        .filter(word => word),
      email_notify: {
        enabled: commentConfig.value.email_notify.enabled ? 1 : 0,
        retry_count: commentConfig.value.email_notify.retry_count || 3,
        retry_interval: commentConfig.value.email_notify.retry_interval || 5
      }
    }

    const response = await request.post('/api/config/save', {
      key: 'COMMENT',
      json: config
    })

    if (response.code === 200) {
      message.value = '评论设置保存成功'
      messageType.value = 'success'
      toast.success('评论设置保存成功')
    } else {
      message.value = '评论设置保存失败：' + (response.msg || '未知错误')
      messageType.value = 'error'
      toast.error('评论设置保存失败')
    }
  } catch (error) {
    console.error('保存评论设置失败:', error)
    message.value = '评论设置保存失败：网络错误：'
    messageType.value = 'error'
    toast.error('评论设置保存失败：网络错误：') 
  } finally {
    saving.value = false
  }
}

// 保存全局配置
async function saveGlobalConfig() {
  saving.value = true
  message.value = ''
  messageType.value = ''
  
  try {
    const response = await request.post('/api/config/save', {
      key: 'cardify_functions',
      json: globalConfig.value
    })

    if (response.code === 200) {
      message.value = '全局配置保存成功'
      messageType.value = 'success'
      toast.success('全局配置保存成功')
      // 清除配置缓存
      clearXiaoFunctionsCache()
      // 保存成功后强制刷新站点信息
      await store.fetchSiteInfo(true)
    } else {
      message.value = '全局配置保存失败：' + (response.msg || '未知错误')
      messageType.value = 'error'
      toast.error('全局配置保存失败')
    }
  } catch (error) {
    console.error('保存全局配置失败:', error)
    message.value = '全局配置保存失败：网络错误：'
    messageType.value = 'error'
    toast.error('保存失败：网络错误：') 
  } finally {
    saving.value = false
  }
}

// 日期处理方法
function formatDate(timestamp) {
  if (!timestamp) return ''
  const date = new Date(parseInt(timestamp) * 1000)
  return date.toISOString().split('T')[0]
}

function handleDateChange(event) {
  const dateStr = event.target.value
  if (dateStr) {
    const timestamp = Math.floor(new Date(dateStr).getTime() / 1000)
    globalConfig.value.date = timestamp.toString()
  } else {
    globalConfig.value.date = Math.floor(Date.now() / 1000).toString()
  }
}

// 悬浮按钮管理方法
function addFloatButton() {
  const newId = globalConfig.value.float_buttons.buttons.length > 0 
    ? Math.max(...globalConfig.value.float_buttons.buttons.map(button => button.id)) + 1 
    : 1
  globalConfig.value.float_buttons.buttons.push({
    id: newId,
    name: '新按钮',
    icon: 'bi bi-link',
    url: '',
    tooltip: '',
    enabled: true,
    content: ''
  })
}

function removeFloatButton(index) {
  globalConfig.value.float_buttons.buttons.splice(index, 1)
}

function moveFloatButton(index, direction) {
  if (direction === 'up' && index > 0) {
    const temp = globalConfig.value.float_buttons.buttons[index]
    globalConfig.value.float_buttons.buttons[index] = globalConfig.value.float_buttons.buttons[index - 1]
    globalConfig.value.float_buttons.buttons[index - 1] = temp
  } else if (direction === 'down' && index < globalConfig.value.float_buttons.buttons.length - 1) {
    const temp = globalConfig.value.float_buttons.buttons[index]
    globalConfig.value.float_buttons.buttons[index] = globalConfig.value.float_buttons.buttons[index + 1]
    globalConfig.value.float_buttons.buttons[index + 1] = temp
  }
}

// 获取自定义代码配置
async function getCustomCodeConfig() {
  try {
    const config = await getXiaoFunctionsConfig()
    customCodeConfig.value = config.custom_code || {
      css: '',
      js: '',
      header: '',
      footer: '',
      analytics: ''
    }
  } catch (error) {
    toast.error('获取自定义代码配置失败')
  }
}

// 重置配置
function resetCommentConfig() {
  commentConfig.value = {
    rate_limit: {
      enabled: true,
      max_count: 5,
      time_window: 60
    },
    max_length: 500,
    require_chinese: true,
    sensitive_filter: true,
    sensitive_words: '敏感词1,敏感词2,敏感词3',
    email_notify: {
      enabled: true,
      retry_count: 3,
      retry_interval: 5
    }
  }
}

function resetGlobalConfig() {
  globalConfig.value = {
    title: '',
    description: '',
    keyword: '',
    avatar: '',
    favicon: '',
    date: Math.floor(Date.now() / 1000).toString(),
    display_mode: 'grid',
    custom_nav_links: '',
    copy: {
      code: '',
      link: 'http://beian.miit.gov.cn/'
    },
    police: {
      code: '',
      link: 'https://beian.mps.gov.cn/#/query/webSearch'
    },
    auth_dialog_agreement: {
      enabled: true, // 默认启用协议提示
      user_agreement_content: '用户协议\n\n欢迎使用我们的服务！请仔细阅读以下用户协议：\n\n1. 服务条款\n您必须年满13周岁才能使用本服务。\n\n2. 账户安全\n您有责任维护账户密码的安全性，并对您账户下的所有活动负责。\n\n3. 用户行为规范\n您同意不会：\n- 发布违法、有害或侵犯他人权益的内容\n- 滥用或破坏服务\n- 未经授权访问他人账户\n\n4. 知识产权\n您发布的内容的知识产权归您所有，但我们有权在服务中使用、复制和传播。\n\n5. 服务变更\n我们保留随时修改或终止服务的权利。\n\n6. 免责声明\n我们不对服务的准确性、完整性或可靠性提供保证。\n\n7. 协议变更\n我们可能会更新本协议，您继续使用服务即表示接受更新后的协议。',
      privacy_agreement_content: '隐私协议\n\n我们重视您的隐私。以下是我们的隐私政策：\n\n1. 收集的信息\n我们可能收集以下信息：\n- 您的账户信息（用户名、邮箱、手机号）\n- 使用数据（访问记录、浏览行为）\n- 设备信息（IP地址、浏览器类型）\n\n2. 信息使用\n我们使用收集的信息来：\n- 提供和改进服务\n- 个性化您的体验\n- 发送重要通知\n- 保障服务安全\n\n3. 信息共享\n我们不会向第三方出售您的个人信息。仅在以下情况下共享：\n- 法律要求\n- 保护我们的权益\n- 经您同意\n\n4. 数据安全\n我们采取合理措施保护您的数据，但无法保证绝对安全。\n\n5. 您的权利\n您有权访问、更正或删除您的个人信息。\n\n6. Cookie使用\n我们使用Cookie来改善您的体验，您可以在浏览器中禁用。\n\n7. 政策变更\n我们可能更新隐私政策，变更将在此页面发布。',
    },
    float_buttons: {
      enabled: true,
      style: 'rounded',
      position: 'center',
      show_back_to_top: true,
      show_notice: true,
      buttons: [
        {
          id: 1,
          name: 'QQ',
          icon: 'bi bi-tencent-qq',
          url: 'https://qm.qq.com/q/56SYi6MAta',
          tooltip: '加入交流群',
          enabled: true
        },
        {
          id: 2,
          name: '微信',
          icon: 'bi bi-wechat',
          url: '',
          tooltip: '扫码添加微信群',
          enabled: true,
          image_url: 'https://img1.zhuxu.asia/2026/U5PbclFMPJ.png'
        }
      ]
    },
    reward: {
      enabled: true,
      wechat: '',
      alipay: ''
    }
  }
}

// 保存自定义代码配置
async function saveCustomCodeConfig() {
  saving.value = true
  message.value = ''
  messageType.value = ''
  
  try {
    // 使用缓存的配置（如果存在且未过期）
    const currentConfig = await getXiaoFunctionsConfig()
    
    // 更新自定义代码配置
    const updatedConfig = {
      ...currentConfig,
      custom_code: customCodeConfig.value
    }
    
    // 保存到后端
    const response = await request.post('/api/config/save', {
      key: 'cardify_functions',
      json: updatedConfig
    })

    if (response.code === 200) {
      message.value = '自定义代码保存成功！'
      messageType.value = 'success'
      toast.success('自定义代码保存成功')
      // 清除配置缓存
      clearXiaoFunctionsCache()
    } else {
      message.value = '自定义代码保存失败：' + (response.msg || '未知错误')
      messageType.value = 'error'
      toast.error('自定义代码保存失败')
    }
  } catch (error) {
    message.value = '保存失败：网络错误：'
    messageType.value = 'error'
    toast.error('保存失败：网络错误：')
  } finally {
    saving.value = false
  }
}

// 重置自定义代码配置
function resetCustomCodeConfig() {
  customCodeConfig.value = {
    css: '',
    js: '',
    header: '',
    footer: '',
    analytics: ''
  }
}

// 组件挂载
onMounted(async () => {
  // 检查登录状态
  await store.checkLoginState()
  
  // 如果是管理员，获取所有配置
  if (isAdmin.value) {
    await Promise.all([
      getCommentConfig(),
      getGlobalConfig(),
      getCustomCodeConfig()
    ])
  }
})
</script>

<style scoped>
/* ==========================================
 * 导航标签
 * ========================================== */
.config-page .nav-pills .nav-link {
  border-radius: var(--wx-radius-sm);
  color: var(--bs-secondary-color);
  font-weight: 500;
  transition: var(--wx-transition);
  padding: 0.375rem 0.875rem;
  font-size: 0.875rem;
}

.config-page .nav-pills .nav-link:hover {
  color: var(--bs-primary);
  background-color: rgba(var(--bs-primary-rgb), 0.08);
}

.config-page .nav-pills .nav-link.active {
  background: var(--wx-gradient-primary);
  color: #fff;
  box-shadow: 0 2px 8px rgba(var(--bs-primary-rgb), 0.2);
}

/* ==========================================
 * 卡片头/体
 * ========================================== */
.config-page .card-header {
  background-color: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
  padding: 0.75rem 1.25rem;
}

.config-page .card-body {
  padding: 1.25rem;
}

/* ==========================================
 * 章节标题
 * ========================================== */
.fs-section-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--bs-body-color);
  margin: 0 0 0.875rem 0;
  padding-bottom: 0.5rem;
  border-bottom: 1px solid var(--bs-border-color);
}

.fs-section-title i {
  color: var(--bs-primary);
  font-size: 1rem;
}

/* ==========================================
 * 表单字段 - 统一高度与对齐
 * ========================================== */
.fs-field {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.fs-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--bs-body-color);
  margin-bottom: 0;
  line-height: 1.4;
}

.fs-hint {
  font-size: 0.78rem;
  color: var(--bs-secondary-color);
  margin-top: 0;
  line-height: 1.4;
}

/* 统一所有表单控件高度 */
.fs-field .wx-form-control,
.fs-field .form-select,
.fs-field .input-group-text {
  height: 34px;
  line-height: 1.4;
  padding: 0.4rem 0.75rem;
  font-size: 0.875rem;
}

.fs-field select.wx-form-control {
  padding-right: 2rem;
  background-image: var(--bs-form-select-bg-img, url("data:image/svg+xml,%3csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 16 16' fill='%236c757d'%3e%3cpath fill-rule='evenodd' d='M1.646 4.646a.5.5 0 0 1 .708 0L8 10.293l5.646-5.647a.5.5 0 0 1 .708.708l-6 6a.5.5 0 0 1-.708 0l-6-6a.5.5 0 0 1 0-.708z'/%3e%3c/svg%3e"));
  background-repeat: no-repeat;
  background-position: right 0.625rem center;
  background-size: 0.875em auto;
  appearance: none;
  -webkit-appearance: none;
  -moz-appearance: none;
}

.fs-field textarea.wx-form-control {
  height: auto;
  min-height: 80px;
  line-height: 1.5;
  resize: vertical;
}

.fs-field .input-group {
  height: 34px;
}

.fs-field .input-group .wx-form-control {
  height: 100%;
}

.fs-field .input-group-text {
  background: var(--bs-tertiary-bg);
  border-color: var(--bs-border-color);
  color: var(--bs-secondary-color);
  font-size: 0.85rem;
  padding: 0 0.625rem;
  min-width: 2.5rem;
  justify-content: center;
}

/* 代码文本域 */
.wx-code-textarea {
  font-family: var(--bs-font-monospace);
  resize: vertical;
}

/* ==========================================
 * 开关行 - 统一样式
 * ========================================== */
.fs-switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.625rem 0.875rem;
  background: var(--bs-tertiary-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-sm);
  transition: var(--wx-transition);
  min-height: 34px;
}

.fs-switch-row:hover {
  border-color: rgba(var(--bs-primary-rgb), 0.3);
}

.fs-switch-row .fs-label {
  margin-bottom: 0;
}

.fs-switch-row .fs-hint {
  margin-top: 2px;
}

.fs-switch-row .form-switch {
  flex-shrink: 0;
  margin: 0;
}

.fs-switch-row .form-check {
  display: flex;
  align-items: center;
}

.fs-switch-row label.form-check-label {
  display: none;
}

/* 小尺寸开关行（无描述文本） */
.fs-switch-row-compact {
  padding: 0.375rem 0.75rem;
  min-height: auto;
}

/* ==========================================
 * 保存按钮行
 * ========================================== */
.fs-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-top: 1rem;
  margin-top: 0.5rem;
  border-top: 1px solid var(--bs-border-color);
}

/* ==========================================
 * 悬浮按钮卡片
 * ========================================== */
.float-btn-item {
  background: var(--bs-body-bg);
  border: 1px solid var(--bs-border-color);
  border-radius: var(--wx-radius-md);
  padding: 0.875rem;
  transition: var(--wx-transition);
}

.float-btn-item:hover {
  border-color: rgba(var(--bs-primary-rgb), 0.3);
  box-shadow: var(--wx-shadow-sm);
}

.float-btn-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.625rem;
}

.float-btn-header h4 {
  font-size: 0.9rem;
  font-weight: 600;
  margin: 0;
}

.float-btn-actions {
  display: flex;
  gap: 0.25rem;
}

.float-btn-actions .wx-btn-outline {
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
}

/* ==========================================
 * 响应式
 * ========================================== */
@media (max-width: 768px) {
  .fs-field .wx-form-control,
  .fs-field .form-select,
  .fs-field .input-group-text {
    height: 36px;
  }
  
  .fs-field .input-group {
    height: 36px;
  }
  
  .config-page .card-body {
    padding: 1rem;
  }
}
</style>