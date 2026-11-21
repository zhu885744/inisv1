<template>
  <div class="system-config">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">系统配置</h2>
          <p class="block-desc">
            网站信息、认证、缓存、存储、短信等配置；修改后立即生效，无需重启（存储配置会自动重载）。
            密钥/限流/滑块与敏感词库、云端机审已移到
            <RouterLink class="inline-link" to="/admin/security/policy">安全中心</RouterLink>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="loading" @click="loadAll">
            <i class="bi bi-arrow-clockwise" /> 重新加载
          </button>
        </div>
      </header>

      <p v-if="loadError" class="stat-error">
        <i class="bi bi-exclamation-circle" aria-hidden="true" />
        <span>部分配置读取失败，可点击「重新加载」重试</span>
      </p>
    </section>

    <div class="config-layout">
      <!-- 模块导航 -->
      <nav class="config-nav" aria-label="配置模块">
        <button
          v-for="m in modules"
          :key="m.key"
          type="button"
          class="config-nav-item"
          :class="{ 'is-active': activeGroup.key === m.key }"
          @click="selectGroup(m)"
        >
          <i :class="m.icon" />
          <span>{{ m.label }}</span>
        </button>
      </nav>

      <div class="config-body">
        <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

        <template v-else>
          <!-- 分组内层分页：仅当该分组由多个模块合并而成时显示 -->
          <div v-if="activeTabs.length > 1" class="config-subtabs" role="tablist">
            <button
              v-for="item in activeTabs"
              :key="item.key"
              type="button"
              class="config-subtab"
              :class="{ 'is-active': tab === item.key }"
              @click="tab = item.key"
            >
              <i :class="item.icon" /> {{ item.label }}
            </button>
          </div>

          <!-- ============ 网站设置（站点信息 / 备案 / 协议 / 悬浮按钮）============ -->
          <SiteSettingsForm v-if="tab === 'site'" />
          <!-- 网站表单设置：自定义菜单 + 个性侧边栏的可视化布局编辑器 -->
          <SiteLayoutForm v-else-if="tab === 'siteLayout'" />

          <!-- ============ 内容配置（文章 / 动态 / 独立页面三份结构相同，同一页并排对比）============ -->
          <section v-if="tab === 'contentCfg'" class="content-config">
            <div class="config-head">
              <div>
                <h3 class="config-title">内容配置</h3>
                <p class="config-desc">
                  文章、动态、独立页面三份配置结构相同：编辑器、审核与评论开关放在一起对比调整；
                  审核开启后，用户新发布的内容默认进入待审核
                </p>
              </div>
            </div>

            <div class="content-grid">
              <article v-for="item in CONTENT_MODULES" :key="item.key" class="card card-pad content-card">
                <header class="content-head">
                  <span class="content-badge"><i :class="item.icon" /></span>
                  <div>
                    <h4 class="content-title">{{ item.title }}</h4>
                    <p class="content-sub">{{ item.sub }}</p>
                  </div>
                </header>

                <div class="form-item">
                  <label class="form-label">编辑器</label>
                  <FormSelect v-model="cfg.content[item.key].editor" :options="EDITOR_OPTIONS" />
                  <p class="form-hint">Mellow 后台编辑器统一使用 Markdown，该值供其它端读取</p>
                </div>

                <div class="form-item">
                  <label class="form-label">内容审核</label>
                  <FormSelect v-model="cfg.content[item.key].audit" :options="ON_OFF" />
                  <p class="form-hint">
                    开启后，用户新发布的{{ item.name }}需审核通过才对外可见；关闭则发布即通过。
                    已存在的{{ item.name }}二次编辑保存不会被打回待审核
                  </p>
                </div>

                <div class="form-item">
                  <label class="form-label">允许评论</label>
                  <FormSelect
                    v-model="cfg.content[item.key].comment.allow"
                    :options="COMMENT_ALLOW_OPTIONS" />
                  <p class="form-hint">设为「禁止」时覆盖内容自身设置，一律禁止</p>
                </div>

                <div class="form-item">
                  <label class="form-label">显示评论</label>
                  <FormSelect
                    v-model="cfg.content[item.key].comment.show"
                    :options="COMMENT_SHOW_OPTIONS" />
                  <p class="form-hint">设为「隐藏」时覆盖内容自身设置，整块不展示</p>
                </div>

                <div class="config-foot">
                  <button
                    class="btn btn-primary btn-sm"
                    :disabled="isSaving(`content_${item.tab}`)"
                    @click="saveContent(item.tab)"
                  >
                    <i class="bi bi-check2" /> 保存{{ item.title }}
                  </button>
                </div>
              </article>
            </div>
          </section>

          <!-- ============ 评论配置 ============ -->
          <section v-else-if="tab === 'comment'" class="card card-pad">
            <header class="config-head">
              <div>
                <h3 class="config-title">评论配置</h3>
                <p class="config-desc">评论开关、限流与内容校验（敏感词已独立为全站词库）</p>
              </div>
            </header>

            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">内容审核</label>
                <FormSelect v-model="cfg.comment.audit" :options="ON_OFF" />
                <p class="form-hint">
                  开启后前台新评论进入待审核（在「评论管理」里通过 / 驳回），未通过的评论前台不可见；
                  关闭则提交即通过
                </p>
              </div>
              <div class="form-item">
                <label class="form-label">允许评论</label>
                <FormSelect v-model="cfg.comment.allow" :options="ON_OFF" />
              </div>
              <div class="form-item">
                <label class="form-label">最大长度（字符）</label>
                <FormInput v-model="cfg.comment.max_length" type="number" min="0" step="1" />
              </div>
              <div class="form-item">
                <label class="form-label">限流</label>
                <FormSelect v-model="cfg.comment.rate_limit.enabled" :options="ON_OFF" />
              </div>
              <div class="form-item">
                <label class="form-label">时间窗口（秒）</label>
                <FormInput v-model="cfg.comment.rate_limit.time_window" type="number" min="1" step="1" />
              </div>
              <div class="form-item">
                <label class="form-label">窗口内最大条数</label>
                <FormInput v-model="cfg.comment.rate_limit.max_count" type="number" min="1" step="1" />
              </div>
              <div class="form-item">
                <label class="form-label">必须包含中文</label>
                <FormSelect v-model="cfg.comment.require_chinese" :options="ON_OFF" />
              </div>
            </div>

            <p class="form-hint">
              敏感词过滤已独立为「敏感词库」，评论 / 文章 / 动态 / 私信 / 群聊共用一份词表；
              邮件通知（评论 / 回复）见「邮件通知」模块中的「评论互动」分组
            </p>

            <div class="config-foot">
              <button class="btn btn-primary btn-sm" :disabled="isSaving('comment')" @click="saveComment">
                <i class="bi bi-check2" /> 保存
              </button>
            </div>
          </section>

          <!-- ============ 友链配置（LINKS）============ -->
          <section v-else-if="tab === 'links'" class="card card-pad">
            <header class="config-head">
              <div>
                <h3 class="config-title">友链配置</h3>
                <p class="config-desc">前台「申请友链」的审核策略</p>
              </div>
            </header>

            <div class="form-item">
              <label class="form-label">内容审核</label>
              <FormSelect v-model="cfg.links.audit" :options="ON_OFF" />
              <p class="form-hint">
                开启后前台提交的友链进入待审核（在「友链管理」里通过 / 驳回，申请时会通知管理员）；
                关闭则提交即通过。管理员在后台自行添加的友链不受该开关影响
              </p>
            </div>

            <div class="config-foot">
              <button class="btn btn-primary btn-sm" :disabled="isSaving('links')" @click="saveLinks">
                <i class="bi bi-check2" /> 保存
              </button>
            </div>
          </section>

          <!-- ============ 装扮（头像框 / 头衔，SYSTEM_DECORATION）============ -->
          <section v-else-if="tab === 'decoration'" class="card card-pad">
            <header class="config-head">
              <div>
                <h3 class="config-title">装扮配置</h3>
                <p class="config-desc">
                  头衔 / 头像框的全局开关；具体装扮的增删改查、价格与样式在「装扮管理」中维护
                </p>
              </div>
              <RouterLink class="btn btn-sm" to="/admin/goods/decoration">
                <i class="bi bi-box-arrow-up-right" /> 装扮管理
              </RouterLink>
            </header>

            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">装扮功能</label>
                <FormSelect v-model="cfg.decoration.enabled" :options="ON_OFF" />
                <p class="form-hint">关闭后所有用户都无法佩戴 / 卸下装扮，已佩戴的样式仍会正常展示</p>
              </div>
              <div class="form-item">
                <label class="form-label">商城显示装扮分区</label>
                <FormSelect v-model="cfg.decoration.showInGoods" :options="ON_OFF" />
                <p class="form-hint">关闭后积分商城只显示普通商品，装扮仍可通过后台发放</p>
              </div>
            </div>

            <h4 class="config-sub-title">头像框</h4>
            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">启用头像框</label>
                <FormSelect v-model="cfg.decoration.frameEnabled" :options="ON_OFF" />
              </div>
            </div>
            <!-- 缩放/对齐不再走全局默认值：每个头像框在「装扮管理」里用对齐器单独调洞大小与偏移 -->

            <h4 class="config-sub-title">头衔</h4>
            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">启用头衔</label>
                <FormSelect v-model="cfg.decoration.titleEnabled" :options="ON_OFF" />
              </div>
              <div class="form-item">
                <label class="form-label">允许用户自填文字</label>
                <FormSelect v-model="cfg.decoration.titleAllowCustom" :options="ON_OFF" />
                <p class="form-hint">
                  开启后，未内置文字且没有图片的「纯样式」头衔，可由用户佩戴时自行输入
                  （图片头衔不需要填字，在「装扮管理」里给头衔配上图片即可）
                </p>
              </div>
              <div class="form-item">
                <label class="form-label">头衔最大字数</label>
                <FormInput v-model="cfg.decoration.titleMaxLength" type="number" min="1" max="20" />
              </div>
            </div>

            <div class="config-foot">
              <button class="btn btn-primary btn-sm" :disabled="isSaving('decoration')" @click="saveDecoration">
                <i class="bi bi-check2" /> 保存
              </button>
            </div>
          </section>

          <!-- ============ 登录会话 ============ -->
          <section v-else-if="tab === 'session'" class="card card-pad">
            <header class="config-head">
              <div>
                <h3 class="config-title">登录会话</h3>
                <p class="config-desc">登录态是服务端会话（缓存里的 session:{sid}），项目已移除 JWT；令牌不携带任何声明，可在退出 / 改密 / 封禁时立即撤销</p>
              </div>
            </header>
            <div class="form-grid">
              <div class="form-item">
                <label class="form-label">会话有效期（秒，支持表达式）</label>
                <FormInput v-model="toml.session.expire" type="text" placeholder="如 15 * 24 * 60 * 60" />
                <p class="form-hint">滑动续期（最后活跃时间 + 有效期），默认 15 天；修改后只影响新登录 / 续期的会话，已签发的会话保持原时长</p>
              </div>
            </div>
            <div class="config-foot">
              <button class="btn btn-primary btn-sm" :disabled="isSaving('session')" @click="saveSession">
                <i class="bi bi-check2" /> 保存
              </button>
            </div>
          </section>

          <!-- ============ 注册 ============ -->
          <template v-else-if="tab === 'register'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">开放注册</h3>
                  <p class="config-desc">关闭后仅管理员可创建账号，注册接口会拒绝新用户提交</p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">允许用户自行注册</label>
                <FormSelect v-model="cfg.register.value" :options="ON_OFF" />
              </div>
              <div class="form-item">
                <label class="form-label">新用户注册验证</label>
                <FormSelect v-model="cfg.register.verifyMode" :options="REGISTER_VERIFY_OPTIONS" />
                <p class="form-hint">{{ registerVerifyHint }}</p>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">邮箱域名限制</h3>
                  <p class="config-desc">用于拦截一次性邮箱或限定内部邮箱注册，仅对邮箱注册方式生效</p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">限制模式</label>
                <FormSelect v-model="cfg.register.domainMode" :options="REGISTER_DOMAIN_OPTIONS" />
                <p class="form-hint">{{ registerDomainHint }}</p>
              </div>

              <div v-if="cfg.register.domainMode === 'whitelist'" class="form-item">
                <label class="form-label">注册邮箱白名单</label>
                <FormTextarea
                  v-model="cfg.register.whitelistText"
                  class="code-textarea"
                  rows="5"
                  placeholder="qq.com&#10;163.com&#10;@foxmail.com" />
                <p class="form-hint">只允许使用这些域名结尾的邮箱地址注册，每行一个域名（@ 可省略，子域自动匹配，如 qq.com 同时允许 mail.qq.com）。</p>
              </div>

              <div v-if="cfg.register.domainMode === 'blacklist'" class="form-item">
                <label class="form-label">注册邮箱黑名单</label>
                <FormTextarea
                  v-model="cfg.register.blacklistText"
                  class="code-textarea"
                  rows="5"
                  placeholder="example.com&#10;@tempmail.com" />
                <p class="form-hint">禁止使用这些域名结尾的邮箱地址注册，每行一个域名（@ 可省略，子域自动匹配）。</p>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">注册后发布欢迎消息</h3>
                  <p class="config-desc">两项可同时开启；人工审核模式下管理员通过审核时发送</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">站内消息通知</label>
                  <FormSelect v-model="cfg.register.welcomeMessage" :options="ON_OFF" />
                  <p class="form-hint">注册成功后，向用户发送站内欢迎消息</p>
                </div>
                <div class="form-item">
                  <label class="form-label">发送欢迎 Email</label>
                  <FormSelect v-model="cfg.register.welcomeEmail" :options="ON_OFF" />
                  <p class="form-hint">注册成功后，向用户注册邮箱发送欢迎邮件</p>
                </div>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">新用户默认权限组</h3>
                  <p class="config-desc">注册成功后自动入组，空选表示不带任何权限组</p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">默认权限组</label>
                <div v-if="registerGroups.loading" class="is-loading is-inline"><span class="spinner" /> 加载权限组...</div>
                <p v-else-if="!registerGroups.list.length" class="form-hint">
                  暂无可用权限组：请先到「安全 → 权限分组」创建，或确认当前账号有权访问权限组列表。
                </p>
                <div v-else class="check-grid is-block">
                  <label v-for="g in registerGroups.list" :key="g.id" class="check-line">
                    <FormCheckbox v-model="cfg.register.ids" :value="Number(g.id)" />
                    <span>
                      {{ g.name || g.key }}
                      <code v-if="g.key" class="group-key">{{ g.key }}</code>
                      <span v-if="Number(g.root) === 1" class="root-flag">超管</span>
                    </span>
                  </label>
                </div>
                <p class="form-hint">
                  注册成功后自动把新用户加入所选权限组（写入 ALLOW_REGISTER 的 text 字段，即分组 ID 列表）；
                  留空表示新用户不带任何权限组。含「超管」分组请谨慎勾选。
                </p>
              </div>
            </section>

            <section class="card card-pad">
              <div class="config-foot is-plain">
                <span class="form-hint is-inline">以上设置同属 ALLOW_REGISTER 一条记录，点保存一次即可全部生效</span>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('register')" @click="saveRegister">
                  <i class="bi bi-check2" /> 保存注册设置
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 账号修改 ============ -->
          <template v-else-if="tab === 'accountRule'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">账号修改规则</h3>
                  <p class="config-desc">
                    账号既是登录凭证又是对外身份标识，改完不可逆、还会占掉他人的登录名，是否开放交给站点决定（默认关闭）
                  </p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">允许修改账号</label>
                <FormSelect v-model="cfg.account.value" :options="ON_OFF" />
                <p class="form-hint">关闭后前台「修改账号」入口不可用，接口会直接拒绝提交</p>
              </div>
              <div class="form-item">
                <label class="form-label">冷却天数</label>
                <FormInput v-model.number="cfg.account.cooldownDays" type="number" min="0" max="365" step="1" />
                <p class="form-hint">一次修改成功后多少天内不可再次修改；填 0 表示不限制（-&gt; 最长 365 天）</p>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">修改前置条件</h3>
                  <p class="config-desc">不满足条件的用户无法修改账号，用于降低账号被盗改的风险</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">要求已绑定手机号 / 邮箱</label>
                  <FormSelect v-model="cfg.account.requireContact" :options="ON_OFF" />
                  <p class="form-hint">开启后，未绑定手机号或邮箱的账号不能修改</p>
                </div>
                <div class="form-item">
                  <label class="form-label">要求身份验证</label>
                  <FormSelect v-model="cfg.account.requireVerify" :options="ON_OFF" />
                  <p class="form-hint">开启后提交修改需先通过当前密码或邮箱 / 手机验证码校验</p>
                </div>
              </div>
            </section>

            <section class="card card-pad">
              <div class="config-foot is-plain">
                <span class="form-hint is-inline">人机校验（滑块）场景在「安全与限流 → 安全验证（滑块）」里勾选「修改账号」</span>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('accountRule')" @click="saveAccount">
                  <i class="bi bi-check2" /> 保存账号修改设置
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 缓存 ============ -->
          <template v-else-if="tab === 'cache'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">缓存设置</h3>
                  <p class="config-desc">选择缓存驱动并配置参数；切换驱动后建议执行连通性测试</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">启用缓存</label>
                  <FormSelect v-model="toml.cache.openValue" :options="ON_OFF" />
                </div>
                <div class="form-item">
                  <label class="form-label">默认驱动</label>
                  <FormSelect v-model="toml.cache.default" :options="CACHE_DRIVERS" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('cacheDefault')" @click="saveCacheDefault">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">Redis</h3>
                  <p class="config-desc">需要 Redis 服务可达</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">主机</label>
                  <FormInput v-model="toml.cache.redis.host" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">端口</label>
                  <FormInput v-model="toml.cache.redis.port" type="number" step="1" />
                </div>
                <div class="form-item">
                  <label class="form-label">数据库</label>
                  <FormInput v-model="toml.cache.redis.database" type="number" step="1" />
                </div>
                <div class="form-item">
                  <label class="form-label">密码</label>
                  <FormInput v-model="toml.cache.redis.password" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">前缀</label>
                  <FormInput v-model="toml.cache.redis.prefix" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">有效期（秒，支持表达式）</label>
                  <FormInput v-model="toml.cache.redis.expire" type="text" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-sm" :disabled="isSaving('testRedis')" @click="testRedis">
                  <i class="bi bi-plug" /> 测试连接
                </button>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('cacheRedis')" @click="saveCacheRedis">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">文件缓存</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">存储路径</label>
                  <FormInput v-model="toml.cache.file.path" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">前缀</label>
                  <FormInput v-model="toml.cache.file.prefix" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">有效期（秒，支持表达式）</label>
                  <FormInput v-model="toml.cache.file.expire" type="text" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('cacheFile')" @click="saveCacheFile">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">内存缓存</h3>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">有效期（秒，支持表达式）</label>
                <FormInput v-model="toml.cache.ram.expire" type="text" />
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('cacheRam')" @click="saveCacheRam">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 存储 ============ -->
          <template v-else-if="tab === 'storage'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">默认存储</h3>
                  <p class="config-desc">决定附件上传使用的驱动；切换后会自动重载存储配置</p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">默认驱动</label>
                <FormSelect v-model="toml.storage.default" :options="STORAGE_DRIVERS" />
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('storageDefault')" @click="saveStorageDefault">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">本地存储</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">访问域名</label>
                  <FormInput v-model="toml.storage.local.domain" type="text" placeholder="留空 = 相对路径 /storage/xxx" />
                  <p class="form-hint">
                    留空时附件地址是相对路径（跟随站点域名）；需要 CDN 时填 https://cdn.example.com。
                    不要填成与「存储路径」相同的值，否则会拼出 storage/storage/xxx
                  </p>
                </div>
                <div class="form-item">
                  <label class="form-label">存储路径</label>
                  <FormInput v-model="toml.storage.local.path" type="text" placeholder="storage" />
                  <p class="form-hint">public 下的子目录，留空则文件直接放在 public 下</p>
                </div>
                <div class="form-item">
                  <label class="form-label">目录命名规则</label>
                  <FormInput v-model="toml.storage.local.dir_rule" type="text" placeholder="{Y}-{m}/{d}" />
                  <p class="form-hint">上传目录结构，可多层（用 / 分隔）；只想放在存储路径下不要子目录就填 /</p>
                </div>
                <div class="form-item">
                  <label class="form-label">文件命名规则</label>
                  <FormInput v-model="toml.storage.local.file_rule" type="text" placeholder="{timestamp}{str-random-10}" />
                  <p class="form-hint">不含扩展名，扩展名会按原文件自动追加</p>
                </div>
              </div>
              <p class="form-hint">
                上传到：<code class="rule-preview">{{ uploadRulePreview(toml.storage.local.path, toml.storage.local.dir_rule, toml.storage.local.file_rule) }}</code>
              </p>
              <p class="form-hint">
                可用占位符：
                <span v-for="item in UPLOAD_RULE_TOKENS" :key="item.token" class="rule-token">
                  <code>{{ item.token }}</code> {{ item.desc }}
                </span>
              </p>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('storageLocal')" @click="saveStorageLocal">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">腾讯云 COS</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">SecretId</label>
                  <FormInput v-model="toml.storage.cos.secret_id" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">SecretKey</label>
                  <FormInput v-model="toml.storage.cos.secret_key" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">AppId</label>
                  <FormInput v-model="toml.storage.cos.app_id" type="text" placeholder="如 1250000000" />
                  <p class="form-hint">桶名的数字后缀，用于拼默认域名 &lt;bucket&gt;-&lt;app_id&gt;.cos.&lt;region&gt;.myqcloud.com</p>
                </div>
                <div class="form-item">
                  <label class="form-label">Bucket</label>
                  <FormInput v-model="toml.storage.cos.bucket" type="text" placeholder="如 inis-cos" />
                  <p class="form-hint">
                    可填裸桶名（自动补 -AppId），也可直接填控制台复制的全名 inis-cos-1250000000（不会再重复拼）
                  </p>
                </div>
                <div class="form-item">
                  <label class="form-label">地域</label>
                  <FormInput v-model="toml.storage.cos.region" type="text" placeholder="如 ap-guangzhou" />
                  <p class="form-hint">留空按 ap-guangzhou（广州）处理</p>
                </div>
                <div class="form-item">
                  <label class="form-label">访问域名</label>
                  <FormInput v-model="toml.storage.cos.domain" type="text" placeholder="留空 = 默认域名" />
                  <p class="form-hint">留空时用 https://&lt;bucket&gt;-&lt;app_id&gt;.cos.&lt;region&gt;.myqcloud.com，填了则作为 CDN 域名</p>
                </div>
                <div class="form-item">
                  <label class="form-label">路径前缀</label>
                  <FormInput v-model="toml.storage.cos.path" type="text" placeholder="如 inis" />
                  <p class="form-hint">对象键前缀，留空则键直接从目录命名规则开始（不带前导 /）</p>
                </div>
                <div class="form-item">
                  <label class="form-label">目录命名规则</label>
                  <FormInput v-model="toml.storage.cos.dir_rule" type="text" placeholder="{Y}-{m}/{d}" />
                  <p class="form-hint">对象键的目录结构，可多层（用 / 分隔）；不想分目录就填 /</p>
                </div>
                <div class="form-item">
                  <label class="form-label">文件命名规则</label>
                  <FormInput v-model="toml.storage.cos.file_rule" type="text" placeholder="{timestamp}{str-random-10}" />
                  <p class="form-hint">不含扩展名，扩展名会按原文件自动追加</p>
                </div>
              </div>
              <p class="form-hint">
                上传到：<code class="rule-preview">{{ uploadRulePreview(toml.storage.cos.path, toml.storage.cos.dir_rule, toml.storage.cos.file_rule) }}</code>
              </p>
              <p class="form-hint">
                可用占位符（与本地存储一致）：
                <span v-for="item in UPLOAD_RULE_TOKENS" :key="item.token" class="rule-token">
                  <code>{{ item.token }}</code> {{ item.desc }}
                </span>
              </p>
              <div class="config-foot">
                <button class="btn btn-sm" :disabled="isSaving('testCos')" @click="testStorage('cos')">
                  <i class="bi bi-plug" /> 测试连接
                </button>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('storageCos')" @click="saveStorageCos">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">附件上传限制</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">允许的后缀</label>
                  <FormInput v-model="toml.storage.attachment.allow_extensions" type="text" />
                  <p class="form-hint">
                    小写、英文逗号分隔，留空表示不限制。图片 jpg/jpeg/png/webp/gif/avif/heic/tiff…，
                    视频 mp4/mov/webm/mkv/avi…，压缩包 zip/rar/7z/tar/gz…
                  </p>
                  <button
                    type="button"
                    class="btn btn-sm"
                    @click="toml.storage.attachment.allow_extensions = RECOMMENDED_ATTACHMENT_EXTS"
                  >
                    <i class="bi bi-magic" /> 填入推荐列表
                  </button>
                </div>
                <div class="form-item">
                  <label class="form-label">单文件最大（KB）</label>
                  <FormInput v-model="toml.storage.attachment.max_file_size" type="number" min="1" step="1" />
                </div>
                <div class="form-item">
                  <label class="form-label">并发上限</label>
                  <FormInput v-model="toml.storage.attachment.concurrent_limit" type="number" min="1" step="1" />
                  <p class="form-hint">同时上传的文件数上限（跨实例用 Redis 计数，Redis 不可用时降级本机计数）</p>
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('storageAttachment')" @click="saveStorageAttachment">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 短信 ============ -->
          <template v-else-if="tab === 'sms'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">短信驱动</h3>
                  <p class="config-desc">分别指定「邮件验证码」与「短信验证码」使用的渠道</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">默认驱动</label>
                  <FormSelect v-model="toml.sms.drive.default" :options="SMS_DRIVERS" />
                </div>
                <div class="form-item">
                  <label class="form-label">邮件渠道</label>
                  <FormSelect v-model="toml.sms.drive.email" :options="SMS_DRIVERS" />
                </div>
                <div class="form-item">
                  <label class="form-label">短信渠道</label>
                  <FormSelect v-model="toml.sms.drive.sms" :options="SMS_DRIVERS" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('smsDrive')" @click="saveSmsDrive">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">邮件发送</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">SMTP 主机</label>
                  <FormInput v-model="toml.sms.email.host" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">端口</label>
                  <FormInput v-model="toml.sms.email.port" type="number" step="1" />
                </div>
                <div class="form-item">
                  <label class="form-label">账号</label>
                  <FormInput v-model="toml.sms.email.account" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">密码 / 授权码</label>
                  <FormInput v-model="toml.sms.email.password" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">发件人昵称</label>
                  <FormInput v-model="toml.sms.email.nickname" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">邮件签名</label>
                  <FormInput v-model="toml.sms.email.sign_name" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">测试收件人</label>
                  <FormInput v-model="test.email" type="text" placeholder="仅用于测试，不会保存" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-sm" :disabled="isSaving('testSmsEmail')" @click="testSmsEmail">
                  <i class="bi bi-envelope" /> 发送测试邮件
                </button>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('smsEmail')" @click="saveSmsEmail">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <!-- 发件队列：分批限流 + 失败重试（config/sms.toml 的 [email] 段） -->
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">发件队列</h3>
                  <p class="config-desc">
                    通知类邮件统一入队后分批发送（验证码入队即发，不受分批限制），
                    失败按「重试延迟」重新入队，达到「最大尝试次数」后标记失败并丢弃；
                    保存后立即生效，无需重启
                  </p>
                </div>
              </header>
              <div class="form-grid">
                <div v-for="item in MAIL_QUEUE_FIELDS" :key="item.key" class="form-item">
                  <label class="form-label">{{ item.label }}</label>
                  <FormInput
                    v-model.number="toml.sms.emailQueue[item.key]"
                    type="number"
                    step="1"
                    :min="item.min"
                    :max="item.max" />
                  <p class="form-hint">{{ item.hint }}（{{ item.min }} ~ {{ item.max }}）</p>
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('smsEmailQueue')" @click="saveSmsEmailQueue">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">阿里云短信</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">AccessKey ID</label>
                  <FormInput v-model="toml.sms.aliyun.access_key_id" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">AccessKey Secret</label>
                  <FormInput v-model="toml.sms.aliyun.access_key_secret" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">Endpoint</label>
                  <FormInput v-model="toml.sms.aliyun.endpoint" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">短信签名</label>
                  <FormInput v-model="toml.sms.aliyun.sign_name" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">模板 Code</label>
                  <FormInput v-model="toml.sms.aliyun.verify_code" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">测试手机号</label>
                  <FormInput v-model="test.phone" type="text" placeholder="仅用于测试，不会保存" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-sm" :disabled="isSaving('testSmsAliyun')" @click="testSmsAliyun">
                  <i class="bi bi-phone" /> 发送测试短信
                </button>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('smsAliyun')" @click="saveSmsAliyun">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">阿里云号码认证</h3>
                  <p class="config-desc">用于号码认证类验证码，模板参数由服务端填充</p>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">AccessKey ID</label>
                  <FormInput v-model="toml.sms.aliyunNumberVerify.access_key_id" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">AccessKey Secret</label>
                  <FormInput v-model="toml.sms.aliyunNumberVerify.access_key_secret" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">Endpoint</label>
                  <FormInput v-model="toml.sms.aliyunNumberVerify.endpoint" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">短信签名</label>
                  <FormInput v-model="toml.sms.aliyunNumberVerify.sign_name" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">模板 Code</label>
                  <FormInput v-model="toml.sms.aliyunNumberVerify.template_code" type="text" />
                </div>
              </div>
              <div class="config-foot">
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="isSaving('smsAliyunNumberVerify')"
                  @click="saveSmsAliyunNumberVerify"
                >
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">腾讯云短信</h3>
                </div>
              </header>
              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">SecretId</label>
                  <FormInput v-model="toml.sms.tencent.secret_id" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">SecretKey</label>
                  <FormInput v-model="toml.sms.tencent.secret_key" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">Endpoint</label>
                  <FormInput v-model="toml.sms.tencent.endpoint" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">SDK AppId</label>
                  <FormInput v-model="toml.sms.tencent.sms_sdk_app_id" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">短信签名</label>
                  <FormInput v-model="toml.sms.tencent.sign_name" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">模板 ID</label>
                  <FormInput v-model="toml.sms.tencent.verify_code" type="text" />
                </div>
                <div class="form-item">
                  <label class="form-label">地域</label>
                  <FormInput v-model="toml.sms.tencent.region" type="text" />
                </div>
              </div>
              <div class="config-foot">
                <button class="btn btn-sm" :disabled="isSaving('testSmsTencent')" @click="testSmsTencent">
                  <i class="bi bi-phone" /> 发送测试短信
                </button>
                <button class="btn btn-primary btn-sm" :disabled="isSaving('smsTencent')" @click="saveSmsTencent">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 邮件通知（统一开关）============ -->
          <template v-else-if="tab === 'mail'">
            <!-- 总开关 + 管理员收件人 -->
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">邮件通知总开关</h3>
                  <p class="config-desc">
                    关闭后所有场景都不再发邮件（站内消息不受影响）；
                    邮件统一走邮箱队列分批投递，失败会自动延迟重试
                    （分批 / 重试参数见「短信」标签页的「发件队列」）
                  </p>
                </div>
              </header>

              <div class="form-item">
                <label class="form-label">总开关</label>
                <FormSelect v-model="cfg.mail.enabled" :options="ON_OFF" />
              </div>

              <div class="form-item">
                <label class="form-label">管理员收件邮箱</label>
                <FormInput
                  v-model="cfg.mail.adminEmail"
                  type="text"
                  placeholder="多个用逗号分隔；留空则用「超级管理员」账号的邮箱" />
                <p class="form-hint">
                  审核 / 订单等「通知管理员」的场景发到这里；留空时自动取权限组 root=1 成员的邮箱
                </p>
              </div>
              
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('mail')" @click="saveMail">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <!-- 场景开关：按分组展示，开则正常发送、关则禁用 -->
            <section v-for="group in MAIL_GROUPS" :key="group" class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">{{ group }}</h3>
                  <p class="config-desc">按需开启；关闭后该场景不再发送邮件，站内记录不受影响</p>
                </div>
              </header>

              <div class="form-grid">
                <div v-for="scene in mailScenesOf(group)" :key="scene.key" class="form-item">
                  <label class="form-label">{{ scene.label }}</label>
                  <FormSelect v-model="cfg.mail.scenes[scene.key]" :options="ON_OFF" />
                  <p class="form-hint">{{ scene.target === 'admin' ? '发给管理员' : '发给相关用户' }}</p>
                </div>
              </div>

              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('mail')" @click="saveMail">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 通知与日志 ============ -->
          <template v-else-if="tab === 'notify'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">消息保留</h3>
                  <p class="config-desc">超过保留天数的站内消息会被清理</p>
                </div>
              </header>
              <div class="form-item">
                <label class="form-label">保留天数</label>
                <FormInput v-model="toml.notification.retention_days" type="number" min="1" step="1" />
              </div>
              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('notification')" @click="saveNotification">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">日志</h3>
                  <p class="config-desc">后端未提供日志配置的保存接口，此处仅供查看</p>
                </div>
              </header>
              <dl class="readonly-list">
                <div class="readonly-row">
                  <dt>启用</dt>
                  <dd>{{ toml.log.on ? '是' : '否' }}</dd>
                </div>
                <div class="readonly-row">
                  <dt>单文件体积（MB）</dt>
                  <dd>{{ toml.log.size }}</dd>
                </div>
                <div class="readonly-row">
                  <dt>保留天数</dt>
                  <dd>{{ toml.log.age }}</dd>
                </div>
                <div class="readonly-row">
                  <dt>备份数量</dt>
                  <dd>{{ toml.log.backups }}</dd>
                </div>
              </dl>
            </section>
          </template>

          <!-- ============ 默认值设置：默认头像 ============ -->
          <template v-else-if="tab === 'defaults'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">默认头像</h3>
                  <p class="config-desc">
                    前台「个人资料」页里供用户一键选择的预设头像。可添加 / 删除，支持粘贴图片地址或从附件库选取；
                    留空时前台使用主题内置头像
                  </p>
                </div>
              </header>

              <div class="switch-row">
                <div>
                  <div class="form-label">启用默认头像</div>
                  <p class="form-hint">
                    关闭后前台「个人资料」页不再展示预设头像，用户只能自行上传；开启时展示上面的头像列表（列表为空则使用主题内置头像）
                  </p>
                </div>
                <FormCheckbox variant="switch" v-model="cfg.defaultAvatarsEnabled" />
              </div>

              <div class="avatar-grid">
                <div
                  v-for="(url, idx) in cfg.defaultAvatars"
                  :key="idx"
                  class="avatar-item"
                >
                  <img :src="url" :alt="`默认头像 ${idx + 1}`" loading="lazy" />
                  <button
                    type="button"
                    class="avatar-del"
                    title="删除"
                    @click="removeDefaultAvatar(idx)"
                  >
                    <i class="bi bi-x-lg" />
                  </button>
                </div>
                <p v-if="!cfg.defaultAvatars.length" class="avatar-empty">
                  暂未配置默认头像，前台将使用主题内置头像
                </p>
              </div>

              <div class="avatar-add">
                <FormInput
                  v-model="newAvatarUrl"
                  placeholder="粘贴图片地址，或点「附件库」选择"
                  @keyup.enter="addDefaultAvatar" />
                <button type="button" class="btn btn-sm" @click="openAvatarLibrary">
                  <i class="bi bi-folder2-open" /> 附件库
                </button>
                <button type="button" class="btn btn-primary btn-sm" @click="addDefaultAvatar">
                  <i class="bi bi-plus-lg" /> 添加
                </button>
              </div>

              <div class="config-foot">
                <span class="save-hint">修改后需点击保存才会写入配置，前台立即读取最新值</span>
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="isSaving('defaultAvatars')"
                  @click="saveDefaultAvatars"
                >
                  {{ isSaving('defaultAvatars') ? '保存中...' : '保存' }}
                </button>
              </div>

              <AttachmentLibrary
                v-model:visible="showAvatarLibrary"
                title="选择默认头像"
                accept="image"
                @select="onAvatarLibrarySelect"
              />
            </section>

            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">图片上传压缩</h3>
                  <p class="config-desc">
                    上传图片附件前在浏览器端压缩并转换为 WebP（不改变分辨率，仅调整编码与质量），可显著减小体积；
                    仅对 jpg / png / webp / bmp / tiff 生效，gif、svg 保持原样
                  </p>
                </div>
              </header>

              <div class="form-item">
                <label class="form-label">压缩模式</label>
                <FormSelect v-model="cfg.uploadWebpMode" :options="WEBP_MODE_OPTIONS" />
                <p class="form-hint">
                  关闭：不处理，原样上传；强制压缩：所有可压缩图片（jpg / png / webp / bmp / tiff）
                  在上传前统一转换为 WebP，gif / svg 保持原样
                </p>
              </div>

              <div class="config-foot">
                <button
                  class="btn btn-primary btn-sm"
                  :disabled="isSaving('uploadWebp')"
                  @click="saveUploadWebpMode"
                >
                  {{ isSaving('uploadWebp') ? '保存中...' : '保存' }}
                </button>
              </div>
            </section>
          </template>

          <!-- ============ 第三方登录（QQ / GitHub / Gitee，SYSTEM_OAUTH）============ -->
          <template v-else-if="tab === 'oauth'">
            <section class="card card-pad">
              <header class="config-head">
                <div>
                  <h3 class="config-title">第三方登录</h3>
                  <p class="config-desc">
                    配置 QQ / GitHub / Gitee 开放平台应用。AppKey 只保存在服务端（接口不下发到前端）；
                    回调地址必须与第三方后台登记的完全一致，否则换取 token 会失败。
                  </p>
                </div>
              </header>

              <div class="form-grid">
                <div class="form-item">
                  <label class="form-label">允许第三方账号创建新账号</label>
                  <FormSelect v-model="cfg.oauth.autoRegister" :options="ON_OFF" />
                  <p class="form-hint">
                    未绑定的第三方账号登录后，回调页会让用户自己选「创建新账号」或「绑定已有账号」（输入账号密码）。
                    开启：两个选项都提供；关闭：只提供「绑定已有账号」（同时受「系统设置 → 注册」总开关约束）
                  </p>
                </div>
                <div class="form-item">
                  <label class="form-label">请求超时（秒）</label>
                  <FormInput v-model.number="cfg.oauth.timeout" type="number" min="3" max="60" step="1" />
                  <p class="form-hint">第三方接口请求超时，3 ~ 60 秒（默认 15）</p>
                </div>
                <div class="form-item">
                  <label class="form-label">HTTP 代理</label>
                  <FormInput v-model.trim="cfg.oauth.proxy" type="text" placeholder="http://127.0.0.1:7890（留空表示直连）" />
                  <p class="form-hint">
                    GitHub 在部分网络环境下直连不通（报 context deadline exceeded），此时填服务器上的代理地址即可，支持 http / socks5
                  </p>
                </div>
              </div>

              <template v-for="item in OAUTH_PLATFORMS" :key="item.key">
                <h4 class="config-sub-title">{{ item.label }}</h4>
                <div class="form-grid">
                  <div class="form-item">
                    <label class="form-label">启用</label>
                    <FormSelect v-model="cfg.oauth[item.key].enable" :options="ON_OFF" />
                    <p class="form-hint">{{ item.hint }}</p>
                  </div>
                  <div class="form-item">
                    <label class="form-label">AppID / Client ID</label>
                    <FormInput v-model.trim="cfg.oauth[item.key].appId" type="text" placeholder="开放平台应用 ID" />
                  </div>
                  <div class="form-item">
                    <label class="form-label">AppKey / Client Secret</label>
                    <FormInput v-model.trim="cfg.oauth[item.key].appKey" type="text" placeholder="开放平台应用密钥" />
                    <p class="form-hint">仅服务端使用，不会返回给前端</p>
                  </div>
                  <div class="form-item">
                    <label class="form-label">回调地址</label>
                    <FormInput v-model.trim="cfg.oauth[item.key].redirect" type="text" :placeholder="oauthRedirectPlaceholder" />
                    <p class="form-hint">建议填：{{ oauthRedirectPlaceholder }}</p>
                  </div>
                </div>
              </template>

              <div class="config-foot">
                <button class="btn btn-primary btn-sm" :disabled="isSaving('oauth')" @click="saveOauth">
                  <i class="bi bi-check2" /> 保存
                </button>
              </div>
            </section>
          </template>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
/**
 * 系统配置（/admin/system）
 *
 * 配置来源分两类：
 * 1. config 表（key / value / json / text）—— 内容/评论/友链审核开关、注册开关；
 * 2. toml 文件（经 /api/toml 读写）—— 登录会话、缓存、存储、短信、通知、日志。
 *
 * 已迁走的部分（本页不再维护，改到「安全中心」）：
 * - 密钥校验 / QPS 限流 / 自动 IP 黑名单 / 滑块验证 → 安全与限流（/admin/security/policy）；
 * - 敏感词库 / 云端内容机审 → 内容审核（/admin/security/content）。
 *
 * 后端要点（app/api/controller/toml.go、app/api/middleware/*、app/model/config.go）：
 * - QPS 自动封禁通知已并入统一邮件通知（SYSTEM_MAIL_NOTIFY 的 security.qps_ban 场景），
 *   不再有单独的 SYSTEM_QPS_NOTIFY；
 * - ALLOW_REGISTER 另用 text 存「新用户默认权限组」的 ID 列表（如 "1,2"，空表示不分组），
 *   注册成功后由 Comm.auth() 读取（utils.Unity.Ids 按数字提取），把新用户写进对应 auth-group 的 uids；
 * - SYSTEM_MAIL_NOTIFY 是「统一邮件通知」记录（见 app/model/mail-notify.go）：
 *   value / json.enabled 是总开关（0 关闭时所有场景都停发），json.admin_email 是管理员收件邮箱
 *   （留空回退「超级管理员」账号邮箱），json.scenes 存各场景开关
 *   （article.pending / comment.notify / user.banned / order.paid ...，清单见 model.MailNotifyScenes()）；
 *   评论 / 回复的邮件通知也在此管理（comment.notify / comment.reply），评论模块不再有独立邮件开关；
 *   邮件统一走邮箱队列（app/facade/mail_queue.go）分批投递，失败自动延迟重试；
 * - ALLOW_REGISTER 的 json 存注册扩展设置（见 app/model/register.go）：
 *   email_domain_mode（off/whitelist/blacklist）+ email_whitelist / email_blacklist、
 *   verify_mode（none/manual）、welcome_message / welcome_email；
 *   none 直接注册并登录（注册本身已用邮箱/短信验证码证明所有权），
 *   manual 会把 status 置为 2（待审核），需管理员通过后才能登录；
 * - 内容配置 ARTICLE / MOMENTS / PAGE 结构相同（editor / audit / comment{allow,show}），
 *   分别由 article.go、moments.go、pages.go 的 config() 读取，audit 决定新建内容的默认审核态；
 *   评论配置 COMMENT 结构独立（allow / audit / rate_limit / max_length / require_chinese），
 *   由 comment.go 的 config("comment") 读取；
 *   其余配置的 json 即配置本体，保存时整体覆盖；
 * - 密钥类字段读取时已脱敏为 `****` 串，**原样回传**即表示不修改（后端会还原真实值）；
 * - storage 不带 name 读取时不脱敏，因此这里按分组分别读取；
 *   [local] / [cos] 各支持 dir_rule（目录命名规则）与 file_rule（文件命名规则），
 *   由 app/facade/storage-rule.go 解析：最终键 = path 前缀 / 目录规则 / 文件规则 + 扩展名；
 *   占位符清单见本文件的 UPLOAD_RULE_TOKENS（与后端 StorageRulePlaceholders 同源），
 *   目录规则留空用默认 {Y}-{m}/{d}、填 / 表示不要子目录；
 * - 日志配置后端只读（没有保存接口）。
 */
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import SiteSettingsForm from '@/components/admin/SiteSettingsForm.vue'
// 自定义菜单 + 个性侧边栏的可视化布局编辑（原在网站设置里，2026-11 抽成独立分页）
import SiteLayoutForm from '@/components/admin/SiteLayoutForm.vue'
import { getConfig, getAllConfigs, saveSystemConfig, saveConfig } from '@/api/config'
import { useSiteStore } from '@/stores/site'
import AttachmentLibrary from '@/components/AttachmentLibrary.vue'
import { listAuthGroups } from '@/api/auth-group'
import {
  saveToml,
  testToml,
  getTomlLog,
  getTomlNotification,
  getTomlSms,
  getTomlCache,
  getTomlSession,
  getStorageGroup
} from '@/api/toml'
import { toast } from '@/utils/toast'

// 开关选项（config.value 是字符串）
const ON_OFF = [
  { value: '1', label: '开启' },
  { value: '0', label: '关闭' }
]

const CACHE_DRIVERS = [
  { value: 'redis', label: 'Redis' },
  { value: 'file', label: '文件' },
  { value: 'ram', label: '内存' }
]

const STORAGE_DRIVERS = [
  { value: 'local', label: '本地' },
  { value: 'cos', label: '腾讯云 COS' }
]

/**
 * 上传命名规则（config/storage.toml 的 [local] / [cos] 段：dir_rule / file_rule）
 *
 * 与后端 app/facade/storage-rule.go 的 DefaultStorageDirRule / DefaultStorageFileRule、
 * StorageRulePlaceholders 一一对应，改动默认值或占位符时两边必须同步。
 * 占位符支持 {Y} {y} {m} {d} {timestamp} {uniqid} {md5} {md5-16}
 * {str-random-16} {str-random-10} {filename} {uid}
 */
const DEFAULT_DIR_RULE = '{Y}-{m}/{d}'
const DEFAULT_FILE_RULE = '{timestamp}{str-random-10}'

/**
 * 附件后缀推荐值：图片（含 avif / heic）、视频、压缩包、文档、文本
 *
 * 与后端默认值（app/facade/bootstrap.go）保持一致；已部署站点读的是自己的
 * config/storage.toml，不会自动变，所以提供「填入推荐列表」一键应用。
 */
const RECOMMENDED_ATTACHMENT_EXTS = [
  // 图片
  'jpg', 'jpeg', 'jfif', 'png', 'apng', 'gif', 'webp', 'bmp', 'svg', 'ico', 'tif', 'tiff', 'avif', 'heic', 'heif',
  // 视频
  'mp4', 'm4v', 'mov', 'webm', 'mkv', 'avi', 'flv', 'wmv', '3gp', 'mpg', 'mpeg', 'ogv',
  // 压缩包
  'zip', 'rar', '7z', 'tar', 'gz', 'tgz', 'bz2', 'tbz2', 'xz', 'txz',
  // 文档 / 文本
  'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'csv', 'txt', 'md'
].join(',')

// 占位符说明（顺序与后端 StorageRulePlaceholders 一致）
const UPLOAD_RULE_TOKENS = [
  { token: '{Y}', desc: '年份(2026)' },
  { token: '{y}', desc: '两位数年份(26)' },
  { token: '{m}', desc: '月份(09)' },
  { token: '{d}', desc: '当月的第几号(26)' },
  { token: '{timestamp}', desc: '时间戳(秒)' },
  { token: '{uniqid}', desc: '唯一字符串' },
  { token: '{md5}', desc: '32 位随机 md5' },
  { token: '{md5-16}', desc: '16 位随机 md5' },
  { token: '{str-random-16}', desc: '16 位随机字符串' },
  { token: '{str-random-10}', desc: '10 位随机字符串' },
  { token: '{filename}', desc: '文件原始名称（不含扩展名）' },
  { token: '{uid}', desc: '上传者 ID（游客 0）' }
]

// 预览用的示例值（只影响界面上展示的示例路径，真实取值由后端决定）
const UPLOAD_RULE_SAMPLE = {
  '{Y}': '2026',
  '{y}': '26',
  '{m}': '09',
  '{d}': '26',
  '{timestamp}': '1758888888',
  '{uniqid}': '67e5d1a2f3c4b5',
  '{md5}': 'a1b2c3d4e5f60718293a4b5c6d7e8f90',
  '{md5-16}': 'e5f60718293a4b5c',
  '{str-random-16}': 'k3m9x2p7q1z8b4n6',
  '{str-random-10}': 'k3m9x2p7q1',
  '{filename}': 'photo',
  '{uid}': '1'
}

// 规则预览：把占位符替换成示例值，拼出「前缀/目录/文件名.jpg」供管理员确认结构
function uploadRulePreview(prefix, dirRule, fileRule) {
  const apply = (rule) =>
    String(rule || '').replace(/\{[^{}]*\}/g, (token) => UPLOAD_RULE_SAMPLE[token] ?? '')
  const parts = [prefix, apply(dirRule), `${apply(fileRule)}.jpg`]
    .flatMap((item) => String(item || '').split('/'))
    .map((item) => item.trim())
    .filter((item) => item && item !== '.' && item !== '..')
  return parts.join('/')
}

const SMS_DRIVERS = [
  { value: 'email', label: '邮件' },
  { value: 'aliyun', label: '阿里云短信' },
  { value: 'aliyun_number_verify', label: '阿里云号码认证' },
  { value: 'tencent', label: '腾讯云短信' }
]

// 新用户注册验证方式（对应后端 model.RegisterVerify* 常量）
// 邮箱验证已移除：注册本身就要求邮箱/手机验证码，邮箱所有权在注册时已证明
const REGISTER_VERIFY_OPTIONS = [
  { value: 'none', label: '无（注册后直接登录）' },
  { value: 'manual', label: '人工审核（管理员通过后才能登录）' }
]

// 邮箱域名限制模式
const REGISTER_DOMAIN_OPTIONS = [
  { value: 'off', label: '关闭' },
  { value: 'whitelist', label: '白名单模式' },
  { value: 'blacklist', label: '黑名单模式' }
]

// 侧边导航 = 「分组」，每项可用 tabs 声明组内分页（只有一项时分页自动隐藏）。
// 分组规则：按管理动作归类 —— 登录注册、内容发布（含评论 / 友链）、运行环境、通知渠道；
// 目的：把十几个平铺项收成 6 项，避免侧边导航过长。
// 说明：tab（当前页）始终存叶子 key，所以下面各配置区块的 v-if 判断不受分组影响。
const modules = [
  // 网站设置：基础信息留在本页；自定义菜单 / 个性侧边栏已抽成可视化布局页（网站表单设置）
  {
    key: 'site',
    label: '网站设置',
    icon: 'bi bi-globe2',
    tabs: [
      { key: 'site', label: '网站设置', icon: 'bi bi-globe2' },
      { key: 'siteLayout', label: '网站表单设置', icon: 'bi bi-layout-three-columns' }
    ]
  },
  {
    key: 'account',
    label: '账号与登录',
    icon: 'bi bi-person-badge',
    tabs: [
      { key: 'session', label: '登录会话', icon: 'bi bi-shield-check' },
      { key: 'register', label: '注册', icon: 'bi bi-person-plus' },
      { key: 'accountRule', label: '账号修改', icon: 'bi bi-person-vcard' },
      { key: 'oauth', label: '第三方登录', icon: 'bi bi-share' }
    ]
  },
  // 内容配置：文章 / 动态 / 独立页面三份结构相同，合并到同一页并排对比（不再拆成三个分页）
  // 原「互动配置」分组（评论 / 友链）并入这里：它们都是「内容发布之后」的规则，放一起更顺手
  {
    key: 'contentCfg',
    label: '内容配置',
    icon: 'bi bi-file-earmark-text',
    tabs: [
      { key: 'contentCfg', label: '内容配置', icon: 'bi bi-file-earmark-text' },
      { key: 'comment', label: '评论配置', icon: 'bi bi-chat-dots' },
      { key: 'links', label: '友链配置', icon: 'bi bi-link-45deg' }
    ]
  },
  { key: 'decoration', label: '装扮', icon: 'bi bi-palette' },
  {
    key: 'service',
    label: '服务与存储',
    icon: 'bi bi-hdd-stack',
    tabs: [
      { key: 'cache', label: '缓存', icon: 'bi bi-database' },
      { key: 'storage', label: '存储', icon: 'bi bi-hdd' }
    ]
  },
  {
    key: 'message',
    label: '通知与消息',
    icon: 'bi bi-bell',
    tabs: [
      { key: 'mail', label: '邮件通知', icon: 'bi bi-envelope-paper' },
      { key: 'sms', label: '短信', icon: 'bi bi-send' },
      { key: 'notify', label: '消息与日志', icon: 'bi bi-archive' }
    ]
  },
  { key: 'defaults', label: '默认值设置', icon: 'bi bi-sliders' }
]

// 内容配置的三份配置（PAGE / ARTICLE / MOMENTS 结构相同）
// - key   config 表的记录 key，也是 cfg.content 的下标
// - tab   保存时用的 loading 标记后缀（content_article / content_moments / content_pagesCfg）
// - name  审核开关等说明文案里的主体，避免把「文章配置」整串写进句子
const CONTENT_MODULES = [
  {
    tab: 'article',
    key: 'ARTICLE',
    title: '文章配置',
    name: '文章',
    sub: '长文 / 投稿',
    icon: 'bi bi-file-earmark-text'
  },
  {
    tab: 'moments',
    key: 'MOMENTS',
    title: '动态配置',
    name: '动态',
    sub: '短动态 / 广场',
    icon: 'bi bi-chat-square-text'
  },
  {
    tab: 'pagesCfg',
    key: 'PAGE',
    title: '独立页面配置',
    name: '独立页面',
    sub: '单页 / 关于我',
    icon: 'bi bi-window'
  }
]

// tab key → 配置定义（保存时按 key 反查）
const CONTENT_MODULE_BY_TAB = Object.fromEntries(CONTENT_MODULES.map((item) => [item.tab, item]))

const EDITOR_OPTIONS = [
  { value: 'markdown', label: 'Markdown' },
  { value: 'tinymce', label: 'TinyMCE' }
]

// 内容级评论开关与后端 / 前台口径一致：1 允许/显示、2 禁止/隐藏
// （0 是内容自身的「继承」，总开关不需要该选项）
const COMMENT_ALLOW_OPTIONS = [
  { value: '1', label: '允许' },
  { value: '2', label: '禁止' }
]
const COMMENT_SHOW_OPTIONS = [
  { value: '1', label: '显示' },
  { value: '2', label: '隐藏' }
]

// 模块定位：支持 /admin/system?tab=site 这类直达链接（如用户中心跳「网站设置」）
const route = useRoute()
const router = useRouter()

// ===== 导航分组 =====
// modules 是「导航分组」，每组用 tabs 声明组内分页（只有一项的组会自动隐藏分页行）。
// tab（当前页）存的始终是叶子 key（site / contentCfg / comment / session …），
// 因此各配置区块的 v-if="tab === 'xxx'" 判断不受分组影响，互不耦合：
//   - 点侧边导航 → selectGroup()：落到该组第一个分页（组内已选中则保持不动）；
//   - 点组内分页 → 直接改 tab。
const LEAF_GROUP = {} // 叶子 key → 所属分组
modules.forEach((group) => {
  const tabs = group.tabs && group.tabs.length ? group.tabs : [group]
  group.tabs = tabs
  tabs.forEach((item) => {
    LEAF_GROUP[item.key] = group
  })
})
const LEAF_KEYS = Object.keys(LEAF_GROUP)
// 默认落在「网站设置」：原来的默认页「安全与限流」已迁到安全中心（/admin/security/policy）
const DEFAULT_TAB = 'site'

// 旧的三个内容分页（?tab=article / moments / pagesCfg、?tab=content&sub=moments、?tab=content）
// 已合并成「内容配置」一页，这里统一折算过去，老书签 / 老链接不会失效
const LEGACY_TABS = {
  content: 'contentCfg',
  article: 'contentCfg',
  moments: 'contentCfg',
  pagesCfg: 'contentCfg'
}

function resolveTab(value) {
  const legacy = LEGACY_TABS[String(value || '')]
  if (legacy) return legacy
  return LEAF_KEYS.includes(String(value)) ? String(value) : ''
}

const initialTab = resolveTab(route.query.tab) || resolveTab(route.query.sub) || DEFAULT_TAB
const tab = ref(initialTab)
// 当前分组（侧边导航高亮）与组内分页
const activeGroup = computed(() => LEAF_GROUP[tab.value] || modules[0])
const activeTabs = computed(() => activeGroup.value.tabs || [])
function selectGroup(group) {
  // 组内已选中时保持当前分页，不跳回第一项
  if ((group.tabs || []).some((item) => item.key === tab.value)) return
  tab.value = (group.tabs && group.tabs[0] && group.tabs[0].key) || group.key
}
// 切换页面（含组内分页）时写回地址栏，便于刷新/分享后停留原位（?tab=jwt 这类直达链接照旧可用）
watch(tab, (value) => {
  if (String(route.query.tab || '') === value) return
  router.replace({ query: { ...route.query, tab: value } })
})
// 地址栏被外部改动（如侧栏再点一次直达链接）时同步
watch(
  () => route.query.tab,
  (value) => {
    const next = resolveTab(value) || DEFAULT_TAB
    if (next !== tab.value) tab.value = next
  }
)
const loading = ref(false)
const loadError = ref(false)
// 每个保存动作独立的 loading 标记，避免互相阻塞
const saving = reactive({})

// ===== config 表配置 =====
/**
 * 统一邮件通知的场景（与后端 app/model/mail-notify.go 的 MailNotifyScenes 一一对应）
 *
 * - target: admin 发给管理员（收件邮箱见上面的「管理员收件邮箱」）/ user 发给相关用户
 * - default: 默认开关（后端兜底值，前端只用于「后端还没返回该场景」时展示）
 * 新增场景时两边都要加；前端漏掉的场景后端仍按默认值处理，保存时也不会被覆盖（见 saveMail）。
 */
const MAIL_SCENES = [
  { key: 'article.pending', label: '文章待审核', group: '内容审核', target: 'admin', default: 1 },
  { key: 'article.passed', label: '文章审核通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'article.rejected', label: '文章审核未通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'page.pending', label: '独立页面待审核', group: '内容审核', target: 'admin', default: 1 },
  { key: 'page.passed', label: '独立页面审核通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'page.rejected', label: '独立页面审核未通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'links.pending', label: '友链待审核', group: '内容审核', target: 'admin', default: 1 },
  { key: 'links.passed', label: '友链审核通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'links.rejected', label: '友链审核未通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'moments.pending', label: '动态待审核', group: '内容审核', target: 'admin', default: 1 },
  { key: 'moments.passed', label: '动态审核通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'moments.rejected', label: '动态审核未通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'comment.pending', label: '评论待审核', group: '内容审核', target: 'admin', default: 1 },
  { key: 'comment.passed', label: '评论审核通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'comment.rejected', label: '评论审核未通过', group: '内容审核', target: 'user', default: 1 },
  { key: 'comment.notify', label: '评论通知（内容作者）', group: '评论互动', target: 'user', default: 1 },
  { key: 'comment.reply', label: '回复通知（被回复人）', group: '评论互动', target: 'user', default: 1 },
  // 举报治理：站内信不受开关影响，这里只控制邮件
  { key: 'report.new', label: '收到新举报', group: '举报治理', target: 'admin', default: 1 },
  { key: 'user.pending', label: '新用户待审核', group: '用户管理', target: 'admin', default: 1 },
  { key: 'user.passed', label: '用户审核通过', group: '用户管理', target: 'user', default: 1 },
  { key: 'user.frozen', label: '账号被冻结', group: '用户管理', target: 'user', default: 1 },
  { key: 'user.unfrozen', label: '账号解除冻结', group: '用户管理', target: 'user', default: 1 },
  { key: 'user.banned', label: '账号被封禁', group: '用户管理', target: 'user', default: 1 },
  { key: 'user.unbanned', label: '账号解除封禁', group: '用户管理', target: 'user', default: 1 },
  { key: 'order.paid', label: '订单支付成功', group: '积分商城', target: 'admin', default: 1 },
  { key: 'order.shipped', label: '订单已发货', group: '积分商城', target: 'user', default: 1 },
  { key: 'order.canceled', label: '订单已取消', group: '积分商城', target: 'user', default: 1 }
]

// 场景分组（展示顺序）
const MAIL_GROUPS = ['内容审核', '评论互动', '举报治理', '用户管理', '积分商城']

// 默认场景开关（"1"/"0" 字符串，与 SelectMenu 的 ON_OFF 对齐）
function defaultMailScenes() {
  const scenes = {}
  MAIL_SCENES.forEach((scene) => {
    scenes[scene.key] = String(scene.default)
  })
  return scenes
}

function mailScenesOf(group) {
  return MAIL_SCENES.filter((scene) => scene.group === group)
}

const cfg = reactive({
  // 默认头像：前台「个人资料」页供用户选择的预设头像，存于 Mellow_functions.default_avatars
  defaultAvatars: [],
  // 是否启用预设头像：关闭后前台隐藏预设、只能自行上传
  defaultAvatarsEnabled: true,
  // 图片上传压缩为 WebP：off 关闭 / optional 用户可选 / force 强制压缩
  uploadWebpMode: 'off',
  // 注册设置（同属 ALLOW_REGISTER 一条记录）
  // - value     → value 字段：是否允许自行注册
  // - ids       → text 字段：新用户默认权限组 ID 列表
  // - 其余字段  → json 字段：域名限制 / 验证方式 / 欢迎消息开关
  register: {
    value: '1',
    ids: [],
    verifyMode: 'none',
    domainMode: 'off',
    whitelistText: '',
    blacklistText: '',
    welcomeMessage: '0',
    welcomeEmail: '0'
  },
  // 账号（用户名）修改规则（SYSTEM_ACCOUNT_CHANGE 一条记录，见 app/model/account-change.go）
  // - value          → value 字段：总开关（"0" 关闭 / "1" 开启）
  // - cooldownDays   → json.cooldown_days：冷却天数（0 = 不限制）
  // - requireContact → json.require_contact：是否要求已绑定手机号或邮箱
  // - requireVerify  → json.require_verify：是否要求身份验证（当前密码 / 验证码）
  account: {
    value: '0',
    cooldownDays: 30,
    requireContact: '1',
    requireVerify: '1'
  },
  // 内容配置：ARTICLE / MOMENTS / PAGE 三份结构相同（editor / audit / comment）
  content: {
    ARTICLE: { editor: 'tinymce', audit: '1', comment: { allow: '1', show: '1' } },
    MOMENTS: { editor: 'tinymce', audit: '1', comment: { allow: '1', show: '1' } },
    PAGE: { editor: 'tinymce', audit: '1', comment: { allow: '1', show: '1' } }
  },
  // 评论配置：COMMENT（结构与内容配置不同，字段更多）
  // 注：评论 / 回复的邮件开关不在评论模块，已统一到 SYSTEM_MAIL_NOTIFY（见下方 mail）
  comment: {
    // audit：COMMENT 的 json.audit，1 开启审核（前台评论进待审核）0 直接通过
    audit: '1',
    allow: '1',
    rate_limit: { enabled: '1', max_count: 5, time_window: 60 },
    max_length: 500,
    require_chinese: '1'
  },
  // 友链配置（LINKS 一条记录）：目前只有审核开关
  links: {
    audit: '1'
  },
  // 装扮配置（SYSTEM_DECORATION 一条记录，见 app/model/decoration.go）
  // - enabled         → json.enabled：总开关，关闭后前台不可佩戴任何装扮
  // - showInGoods     → json.show_in_goods：装扮分区是否显示在商城页
  // - frame*          → json.avatar_frame：头像框开关与默认缩放
  // - title*          → json.title：头衔开关、是否允许自填文字、最大字数
  // 装扮本身的增删改查在「装扮管理」（/admin/decoration），这里只控制全局开关
  decoration: {
    enabled: '1',
    showInGoods: '1',
    frameEnabled: '1',
    titleEnabled: '1',
    titleAllowCustom: '1',
    titleMaxLength: 10
  },
  // 统一邮件通知（SYSTEM_MAIL_NOTIFY 一条记录）
  // - enabled     → json.enabled：总开关
  // - adminEmail  → json.admin_email：管理员收件邮箱（多个逗号分隔，空则用超管邮箱）
  // - scenes      → json.scenes：各场景开关（key 见 MAIL_SCENES）
  // - rawScenes   → 后端原始 scenes，保存时与表单合并（避免覆盖掉前端未知的新场景）
  mail: { enabled: '1', adminEmail: '', scenes: defaultMailScenes(), rawScenes: {} },
  // 第三方登录（SYSTEM_OAUTH 一条记录，见 app/model/oauth.go）
  // - autoRegister → json.auto_register：是否允许第三方账号创建新账号（未绑定时回调页选项之一）
  // - qq/github/gitee → json.{平台}.{enable, app_id, app_key, redirect}
  oauth: {
    autoRegister: '1',
    timeout: 15,
    proxy: '',
    qq: { enable: '0', appId: '', appKey: '', redirect: '' },
    github: { enable: '0', appId: '', appKey: '', redirect: '' },
    gitee: { enable: '0', appId: '', appKey: '', redirect: '' }
  }
})

// ===== toml 配置 =====
/**
 * 发件队列参数（config/sms.toml 的 [email] 段）
 * 与后端 app/facade/mail_queue.go 的 MailQueueDefaultValues / MailQueueLimits 一一对应，
 * 默认值与取值范围改动时两边要同步。
 */
const MAIL_QUEUE_FIELDS = [
  { key: 'batch_size', label: '每批数量', def: 10, min: 1, max: 1000, hint: '一个批次窗口内最多发送多少封通知类邮件；验证码不受此限制，入队即发' },
  { key: 'batch_interval', label: '批次间隔（秒）', def: 600, min: 1, max: 86400, hint: '一批发满后等待多久再开下一个窗口（600 = 10 分钟）' },
  { key: 'retry_delay', label: '重试延迟（秒）', def: 60, min: 1, max: 86400, hint: '发送失败后延迟多久重新入队重试' },
  { key: 'max_attempts', label: '最大尝试次数', def: 3, min: 1, max: 10, hint: '含首次发送，达到上限后标记失败并丢弃，不会无限重试' },
  { key: 'send_timeout', label: '发送超时（秒）', def: 30, min: 5, max: 600, hint: '单封邮件超过该时长按失败处理，避免 SMTP 卡住整个队列' },
  { key: 'verify_wait', label: '验证码等待（秒）', def: 10, min: 0, max: 60, hint: '等待首轮发送结果的超时，0 = 不等待（纯异步，前端只提示「已发送」）' },
  { key: 'queue_size', label: '队列上限', def: 1000, min: 10, max: 1000000, hint: '待发任务上限，超出后通知类邮件直接丢弃并记日志（验证码始终受理）' }
]

function defaultMailQueue() {
  const result = {}
  MAIL_QUEUE_FIELDS.forEach((item) => {
    result[item.key] = item.def
  })
  return result
}

const toml = reactive({
  session: { expire: '' },
  cache: {
    openValue: '0',
    default: 'file',
    redis: { host: '', port: 6379, database: 0, password: '', prefix: '', expire: '' },
    file: { path: '', prefix: '', expire: '' },
    ram: { expire: '' }
  },
  storage: {
    // 仅支持 local（本地）与 cos（腾讯云 COS）两种驱动
    default: 'local',
    // dir_rule / file_rule：上传命名规则（自定义目录结构与文件名），默认值与后端一致
    local: { domain: '', path: '', dir_rule: DEFAULT_DIR_RULE, file_rule: DEFAULT_FILE_RULE },
    cos: {
      secret_id: '',
      secret_key: '',
      app_id: '',
      bucket: '',
      region: '',
      domain: '',
      path: '',
      dir_rule: DEFAULT_DIR_RULE,
      file_rule: DEFAULT_FILE_RULE
    },
    attachment: {
      // 只有 3 项真正生效：后缀白名单 / 单文件大小 / 并发上限
      // （原先还有 limit_per_minute 等 5 个「每时段上限」，后端从未实现校验，已移除）
      allow_extensions: '',
      max_file_size: 0,
      concurrent_limit: 0
    }
  },
  sms: {
    drive: { default: 'email', email: 'email', sms: 'email' },
    email: { host: '', port: 465, account: '', password: '', nickname: '', sign_name: '' },
    // 发件队列（与 email 同属 config/sms.toml 的 [email] 段，见 app/facade/mail_queue.go）
    emailQueue: defaultMailQueue(),
    aliyun: { access_key_id: '', access_key_secret: '', endpoint: '', sign_name: '', verify_code: '' },
    aliyunNumberVerify: { access_key_id: '', access_key_secret: '', endpoint: '', sign_name: '', template_code: '' },
    tencent: { secret_id: '', secret_key: '', endpoint: '', sms_sdk_app_id: '', sign_name: '', verify_code: '', region: '' }
  },
  notification: { retention_days: 30 },
  log: { on: false, size: 0, age: 0, backups: 0 }
})

// 测试用字段（不落库）
const test = reactive({ email: '', phone: '' })

// 注册模块用到的权限组列表（后端 auth-group/all，字段同 UserList 的分配权限组弹窗）
const registerGroups = reactive({ list: [], loading: false })

async function loadAuthGroups() {
  registerGroups.loading = true
  try {
    const res = await listAuthGroups()
    registerGroups.list = res.data?.data || []
  } catch {
    // 接口受限时保持为空，页面给出提示
    registerGroups.list = []
  } finally {
    registerGroups.loading = false
  }
}

// ---------- 工具 ----------
function isSaving(key) {
  return !!saving[key]
}

async function withSaving(key, fn) {
  saving[key] = true
  try {
    await fn()
    return true
  } catch (err) {
    // 失败提示由请求拦截器统一给出（如 msg=无权限！），这里吞掉异常即可：
    // 保存按钮是 @click 直接调用，不 catch 会在控制台抛 "Uncaught (in promise)"。
    // 403 额外给一条可操作的提示：新接口的权限点需要给当前账号的权限组勾选。
    if (err?.code === 403) {
      toast.warning('当前账号没有这个权限点，请在「权限分组」中勾选后重试（新接口需重启后端补录规则）')
    }
    return false
  } finally {
    saving[key] = false
  }
}

// config 的 json 字段可能是字符串，也可能是后端解码后的对象
function parseJson(raw, fallback = {}) {
  if (!raw) return { ...fallback }
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw)
      return typeof parsed === 'object' && parsed !== null ? parsed : { ...fallback }
    } catch {
      return { ...fallback }
    }
  }
  if (typeof raw === 'object') return raw
  return { ...fallback }
}

// 把对象里的值统一成字符串，便于输入框绑定
function toText(value, fallback = '') {
  if (value === null || value === undefined) return fallback
  return String(value)
}

// 分组 ID 列表归一化：后端存的是 "|1|2|" 之类的字符串，统一取出数字（与 utils.Unity.Ids 行为一致）
function parseIdList(raw) {
  const matched = String(raw ?? '').match(/\d+/g)
  return matched ? [...new Set(matched.map(Number))] : []
}

// 邮箱域名列表 ↔ 文本：后台按「一行一个」编辑，保存时转数组（后端也会再做一次归一化）
function domainText(list) {
  return Array.isArray(list) ? list.join('\n') : String(list ?? '')
}
function parseDomainText(text) {
  const list = String(text ?? '')
    .split(/[\n\r,，;；\s]+/)
    .map((item) => item.trim().replace(/^@/, '').replace(/^\.+|\.+$/g, '').toLowerCase())
    .filter(Boolean)
  return [...new Set(list)]
}

// 注册验证方式 / 域名限制的说明文案
const registerVerifyHint = computed(() =>
  cfg.register.verifyMode === 'manual'
    ? '选择「人工审核」将由管理员在「用户管理」中逐个确定是否允许新用户登录'
    : '选择「无」用户可直接注册成功（注册本身已需要邮箱/短信验证码）'
)
const registerDomainHint = computed(() => {
  switch (cfg.register.domainMode) {
    case 'whitelist':
      return '选择「白名单模式」，只有限制名单中的邮箱域名可以注册（名单为空时任何邮箱都无法注册）'
    case 'blacklist':
      return '选择「黑名单模式」，限制名单中的邮箱域名不能注册'
    default:
      return '选择「无」直接关闭本功能；名单内每行一个域名，例如 qq.com'
  }
})

// ---------- 读取 ----------
async function readConfig(key) {
  // 配置项可能尚未初始化，失败时不弹全局提示
  const res = await getConfig(key, { silent: true })
  return res?.data && typeof res.data === 'object' ? res.data : null
}

// 内容配置（ARTICLE / MOMENTS / PAGE）结构相同：editor / audit / comment{allow,show}
function readContentConfig(raw) {
  const json = parseJson(raw?.json, {})
  const comment = json.comment && typeof json.comment === 'object' ? json.comment : {}
  return {
    editor: toText(json.editor, 'tinymce'),
    audit: toText(json.audit, '1'),
    comment: {
      allow: toText(comment.allow, '1'),
      show: toText(comment.show, '1')
    }
  }
}

// 注册配置（ALLOW_REGISTER）：value 是注册开关，text 是默认权限组，json 是扩展设置
function readRegisterConfig(raw) {
  const json = parseJson(raw?.json, {})

  cfg.register.value = toText(raw?.value, '1')
  // 新用户默认权限组：后端 Comm.auth() 读取该记录的 text 字段
  cfg.register.ids = parseIdList(raw?.text)

  // 域名限制
  cfg.register.domainMode = ['whitelist', 'blacklist'].includes(json.email_domain_mode)
    ? json.email_domain_mode
    : 'off'
  cfg.register.whitelistText = domainText(json.email_whitelist)
  cfg.register.blacklistText = domainText(json.email_blacklist)

  // 注册验证方式（历史配置里残留的 email 已废弃，不再提供，一律按「无」处理）
  cfg.register.verifyMode = json.verify_mode === 'manual' ? 'manual' : 'none'

  // 欢迎消息 / 欢迎邮件
  cfg.register.welcomeMessage = Number(json.welcome_message) ? '1' : '0'
  cfg.register.welcomeEmail = Number(json.welcome_email) ? '1' : '0'
}

// 账号修改规则（SYSTEM_ACCOUNT_CHANGE）：总开关 + 冷却天数 + 验证要求
// 缺字段按后端默认值兜底（默认关闭 / 冷却 30 天 / 要求绑定联系方式与身份验证）
function readAccountConfig(raw) {
  const json = parseJson(raw?.json, {})

  // 总开关：json.enabled 优先（与后端 AccountChangeConfig 口径一致，保存时两处同步写）
  const enabled = json.enabled === undefined || json.enabled === null ? raw?.value : json.enabled
  cfg.account.value = toText(enabled, '0')
  cfg.account.cooldownDays = Math.max(0, Math.min(365, Number(json.cooldown_days ?? 30) || 0))
  cfg.account.requireContact = toText(json.require_contact, '1')
  cfg.account.requireVerify = toText(json.require_verify, '1')
}

// 评论配置（COMMENT）：结构独立；敏感词已抽到 config/sensitive.toml
// 邮件通知（评论 / 回复）不在这里：已统一到 SYSTEM_MAIL_NOTIFY 的 comment.notify / comment.reply
function readCommentConfig(raw) {
  const json = parseJson(raw?.json, {})
  const rate = json.rate_limit && typeof json.rate_limit === 'object' ? json.rate_limit : {}
  return {
    // 缺省按「开启审核」处理：老配置里可能还没有该字段（与后端 contentAuditSwitch 口径一致）
    audit: toText(json.audit, '1'),
    allow: toText(json.allow, '1'),
    rate_limit: {
      enabled: toText(rate.enabled, '1'),
      max_count: Number(rate.max_count || 5),
      time_window: Number(rate.time_window || 60)
    },
    max_length: Number(json.max_length || 500),
    require_chinese: toText(json.require_chinese, '1')
  }
}

/**
 * 读取统一邮件通知配置（SYSTEM_MAIL_NOTIFY）
 * scenes 用默认值打底：后端新增 / 缺失的场景不会在这里丢掉（rawScenes 保存时原样合并回去）
 */
function readMailConfig(raw) {
  const json = parseJson(raw?.json, {})
  const scenes = json.scenes && typeof json.scenes === 'object' ? json.scenes : {}

  cfg.mail.enabled = toText(json.enabled, '1')
  cfg.mail.adminEmail = toText(json.admin_email, '')
  cfg.mail.rawScenes = { ...scenes }

  const form = defaultMailScenes()
  Object.keys(form).forEach((key) => {
    if (scenes[key] !== undefined) form[key] = Number(scenes[key]) === 1 ? '1' : '0'
  })
  cfg.mail.scenes = form
}

/**
 * 读取装扮配置（SYSTEM_DECORATION）
 * 用后端默认值打底：后台缺字段时不会把表单清空，保存时再补回完整结构
 */
function readDecorationConfig(raw) {
  const json = parseJson(raw?.json, {})
  const frame = json.avatar_frame && typeof json.avatar_frame === 'object' ? json.avatar_frame : {}
  const title = json.title && typeof json.title === 'object' ? json.title : {}

  cfg.decoration.enabled = toText(json.enabled, '1')
  cfg.decoration.showInGoods = toText(json.show_in_goods, '1')
  cfg.decoration.frameEnabled = toText(frame.enabled, '1')
  cfg.decoration.titleEnabled = toText(title.enabled, '1')
  cfg.decoration.titleAllowCustom = toText(title.allow_custom, '1')
  cfg.decoration.titleMaxLength = Number(title.max_length || 10)
}

// 第三方登录平台（顺序与前台按钮、后端 model/oauth.go 的平台列表一致）
const OAUTH_PLATFORMS = [
  { key: 'qq', label: 'QQ', hint: 'QQ 互联 connect.qq.com → 网站应用' },
  { key: 'github', label: 'GitHub', hint: 'GitHub → Settings → Developer settings → OAuth Apps' },
  { key: 'gitee', label: 'Gitee', hint: 'Gitee → 设置 → 第三方应用 → 创建应用' }
]

// 建议的回调地址：三个平台可共用同一个（平台信息由前端记在 sessionStorage，见 utils/oauth.js）
const oauthRedirectPlaceholder = computed(() => `${window.location.origin}/auth/oauth`)

/**
 * 读取第三方登录配置（SYSTEM_OAUTH）
 * 用默认值打底：后台缺字段时不会把表单清空，保存时再补回完整结构
 */
function readOauthConfig(raw) {
  const json = parseJson(raw?.json, {})

  cfg.oauth.autoRegister = toText(json.auto_register, '1')
  cfg.oauth.timeout = Number(json.timeout || 15)
  cfg.oauth.proxy = toText(json.proxy, '')

  OAUTH_PLATFORMS.forEach((item) => {
    const platform = json[item.key] && typeof json[item.key] === 'object' ? json[item.key] : {}
    cfg.oauth[item.key].enable = toText(platform.enable, '0')
    cfg.oauth[item.key].appId = toText(platform.app_id, '')
    cfg.oauth[item.key].appKey = toText(platform.app_key, '')
    cfg.oauth[item.key].redirect = toText(platform.redirect, '')
  })
}

async function loadConfigs() {
  // 一次拉取全部配置，再在本地按 key 分发。
  //
  // 原来这里是 17 个并发的 config/one：path 与 method 完全相同，
  // 一秒内对同一接口打十几次，会直接压满单接口 QPS 阈值（触发自动封禁），
  // 且缓存还没写入时全部穿透到数据库，每个请求都要几百毫秒。
  const res = await getAllConfigs({ silent: true }).catch(() => null)

  const configMap = new Map()
  const list = Array.isArray(res?.data?.data) ? res.data.data : []
  list.forEach((item) => {
    if (item && item.key) configMap.set(item.key, item)
  })
  const read = (key) => configMap.get(key) || null

  const register = read('ALLOW_REGISTER')
  const account = read('SYSTEM_ACCOUNT_CHANGE')
  const article = read('ARTICLE')
  const moments = read('MOMENTS')
  const pages = read('PAGE')
  const comment = read('COMMENT')
  const links = read('LINKS')
  const mailNotify = read('SYSTEM_MAIL_NOTIFY')
  const decoration = read('SYSTEM_DECORATION')
  const oauth = read('SYSTEM_OAUTH')

  if (register) {
    readRegisterConfig(register)
  }
  // 账号修改规则：配置不存在时保持默认值（关闭 / 冷却 30 天）
  if (account) {
    readAccountConfig(account)
  }

  if (article) cfg.content.ARTICLE = readContentConfig(article)
  if (moments) cfg.content.MOMENTS = readContentConfig(moments)
  if (pages) cfg.content.PAGE = readContentConfig(pages)
  if (comment) {
    const parsed = readCommentConfig(comment)
    cfg.comment.audit = parsed.audit
    cfg.comment.allow = parsed.allow
    cfg.comment.rate_limit = parsed.rate_limit
    cfg.comment.max_length = parsed.max_length
    cfg.comment.require_chinese = parsed.require_chinese
  }

  // 友链配置（LINKS）：缺省按「开启审核」处理，与后端 contentAuditSwitch 一致
  cfg.links.audit = links ? toText(parseJson(links.json, {}).audit, '1') : '1'
  if (mailNotify) {
    readMailConfig(mailNotify)
  }
  if (decoration) {
    readDecorationConfig(decoration)
  }
  if (oauth) {
    readOauthConfig(oauth)
  }
}

async function loadTomlAll() {
  // 敏感词库 / 云端机审不在本页：见「安全中心 → 内容审核」（/admin/security/content）
  const [session, cache, sms, notification, log] = await Promise.all([
    getTomlSession().catch(() => null),
    getTomlCache().catch(() => null),
    getTomlSms().catch(() => null),
    getTomlNotification().catch(() => null),
    getTomlLog().catch(() => null)
  ])

  // crypt 返回 { session: { expire } }；兼容老版本返回的内层结构
  const sessionData = session?.data?.session || session?.data?.jwt || session?.data || {}
  toml.session = { expire: toText(sessionData.expire) }

  const cacheData = cache?.data || {}
  toml.cache.openValue = cacheData.open ? '1' : '0'
  toml.cache.default = toText(cacheData.default, 'file')
  toml.cache.redis = { host: '', port: 6379, database: 0, password: '', prefix: '', expire: '', ...(cacheData.redis || {}) }
  toml.cache.file = { path: '', prefix: '', expire: '', ...(cacheData.file || {}) }
  toml.cache.ram = { expire: '', ...(cacheData.ram || {}) }

  const smsData = sms?.data || {}
  toml.sms.drive = { default: 'email', email: 'email', sms: 'email', ...(smsData.drive || {}) }

  // 邮件服务与发件队列同属 [email] 段，但表单分开：队列参数单独放 emailQueue
  const emailData = smsData.email || {}
  toml.sms.email = {
    host: toText(emailData.host, ''),
    port: Number(emailData.port || 465),
    account: toText(emailData.account, ''),
    password: toText(emailData.password, ''),
    nickname: toText(emailData.nickname, ''),
    sign_name: toText(emailData.sign_name, '')
  }
  toml.sms.emailQueue = defaultMailQueue()
  MAIL_QUEUE_FIELDS.forEach((item) => {
    const value = emailData[item.key]
    if (value !== undefined && value !== null && value !== '') {
      toml.sms.emailQueue[item.key] = Number(value)
    }
  })
  toml.sms.aliyun = { access_key_id: '', access_key_secret: '', endpoint: '', sign_name: '', verify_code: '', ...(smsData.aliyun || {}) }
  toml.sms.aliyunNumberVerify = {
    access_key_id: '',
    access_key_secret: '',
    endpoint: '',
    sign_name: '',
    template_code: '',
    ...(smsData.aliyun_number_verify || {})
  }
  toml.sms.tencent = {
    secret_id: '',
    secret_key: '',
    endpoint: '',
    sms_sdk_app_id: '',
    sign_name: '',
    verify_code: '',
    region: '',
    ...(smsData.tencent || {})
  }

  toml.notification.retention_days = Number(notification?.data?.retention_days || 30)

  const logData = log?.data || {}
  toml.log = {
    on: !!logData.on,
    size: Number(logData.size || 0),
    age: Number(logData.age || 0),
    backups: Number(logData.backups || 0)
  }

  // 存储按分组读取（不带 name 的接口不脱敏，不能直接用于表单回填）
  const names = ['local', 'cos', 'attachment']
  const groups = await Promise.all(names.map((name) => getStorageGroup(name).catch(() => null)))
  const picked = {}
  groups.forEach((res, index) => {
    picked[names[index]] = res?.data || {}
  })
  toml.storage.default = toText(picked.local?.default || picked.cos?.default || 'local', 'local')
  // 命名规则（dir_rule / file_rule）：老配置文件里没有这两项，缺项时回填与后端一致的默认值
  const localData = picked.local || {}
  const cosData = picked.cos || {}
  toml.storage.local = {
    domain: '',
    path: '',
    ...localData,
    dir_rule: toText(localData.dir_rule, DEFAULT_DIR_RULE),
    file_rule: toText(localData.file_rule, DEFAULT_FILE_RULE)
  }
  toml.storage.cos = {
    secret_id: '',
    secret_key: '',
    app_id: '',
    bucket: '',
    region: '',
    domain: '',
    path: '',
    ...cosData,
    dir_rule: toText(cosData.dir_rule, DEFAULT_DIR_RULE),
    file_rule: toText(cosData.file_rule, DEFAULT_FILE_RULE)
  }
  // 附件配置只取真正生效的 3 个字段：历史配置里已废弃的 limit_per_* 不再带回来，
  // 否则保存时会把它们又写回 config/storage.toml
  const attachmentData = picked.attachment || {}
  toml.storage.attachment = {
    allow_extensions: toText(attachmentData.allow_extensions, ''),
    max_file_size: Number(attachmentData.max_file_size || 0),
    concurrent_limit: Number(attachmentData.concurrent_limit || 0)
  }
}

async function loadAll() {
  loading.value = true
  loadError.value = false
  try {
    await Promise.all([loadConfigs(), loadTomlAll(), loadAuthGroups(), loadSiteDefaults()])
  } catch {
    loadError.value = true
    toast.error('配置加载失败')
  } finally {
    loading.value = false
  }
}

// 图片上传压缩模式选项（存 Mellow_functions.upload_webp_mode）
const WEBP_MODE_OPTIONS = [
  { value: 'off', label: '关闭' },
  { value: 'force', label: '强制压缩' }
]

// 主题内置默认头像：后台「默认值设置」首次进入时预填，前台无配置时兜底
const BUILTIN_AVATARS = [
  'https://img.zhuxu.asia/tx/1.png',
  'https://img.zhuxu.asia/tx/2.png',
  'https://img.zhuxu.asia/tx/3.png',
  'https://img.zhuxu.asia/tx/4.png',
  'https://img.zhuxu.asia/tx/5.png'
]

// 默认头像（Mellow_functions.default_avatars）：读取完整站点配置后取该字段；
// - 未配置（字段不存在）→ 预填内置头像，开箱即用
// - 已配置（含空数组）→ 尊重之，管理员删光后前台即不再展示预设
// default_avatars_enabled：开关，缺字段视为开启
const siteStore = useSiteStore()

// 读取 Mellow_functions 的完整 json
// config/one 返回的 json 可能是对象，也可能是 JSON 字符串（后端两种都兼容），这里统一解析，
// 否则遇到字符串形态会读成空对象、保存时把整份站点配置覆盖掉
async function readMellowFunctions() {
  const res = await getConfig('Mellow_functions', { silent: true })
  const raw = res?.data?.json
  if (raw && typeof raw === 'object') return { ...raw }
  if (typeof raw === 'string') {
    try {
      const decoded = JSON.parse(raw)
      return decoded && typeof decoded === 'object' ? { ...decoded } : {}
    } catch {
      return {}
    }
  }
  return {}
}

async function loadSiteDefaults() {
  try {
    const data = await readMellowFunctions()
    const list = data.default_avatars
    cfg.defaultAvatars = Array.isArray(list)
      ? list.filter((u) => typeof u === 'string' && u.trim())
      : [...BUILTIN_AVATARS]
    cfg.defaultAvatarsEnabled = data.default_avatars_enabled !== false
    // 历史值 optional 已下线，统一回落到 off
    cfg.uploadWebpMode = ['off', 'force'].includes(data.upload_webp_mode)
      ? data.upload_webp_mode
      : 'off'
  } catch {
    cfg.defaultAvatars = [...BUILTIN_AVATARS]
    cfg.defaultAvatarsEnabled = true
    cfg.uploadWebpMode = 'off'
  }
}

// ---------- 默认头像管理 ----------
const newAvatarUrl = ref('')

function addDefaultAvatar() {
  const url = newAvatarUrl.value.trim()
  if (!url) {
    toast.warning('请先输入或选择图片地址')
    return
  }
  if (cfg.defaultAvatars.includes(url)) {
    toast.warning('该头像已存在')
    return
  }
  cfg.defaultAvatars.push(url)
  newAvatarUrl.value = ''
}

function removeDefaultAvatar(index) {
  cfg.defaultAvatars.splice(index, 1)
}

// 附件库选图（复用 Mellow 通用附件库，选中的图片写入输入框，需点「添加」才入列表）
const showAvatarLibrary = ref(false)
function openAvatarLibrary() {
  showAvatarLibrary.value = true
}
function onAvatarLibrarySelect(urls = []) {
  const url = urls[0]
  if (!url) return
  newAvatarUrl.value = url
  toast.success('图片已选择，记得点「添加」')
}

// 保存：Mellow_functions 的 json 是整体覆盖，必须先拿最新完整配置再合并本字段，
// 否则会清掉 title / float_buttons 等其它配置
async function saveDefaultAvatars() {
  await withSaving('defaultAvatars', async () => {
    const full = await readMellowFunctions()
    full.default_avatars = cfg.defaultAvatars.filter((u) => u && u.trim())
    full.default_avatars_enabled = cfg.defaultAvatarsEnabled
    await saveConfig('Mellow_functions', full)
    await siteStore.invalidate()
    toast.success('已保存')
  })
}

// 保存图片上传压缩模式（与默认头像同属 Mellow_functions，先取完整配置再合并单字段）
async function saveUploadWebpMode() {
  await withSaving('uploadWebp', async () => {
    const full = await readMellowFunctions()
    full.upload_webp_mode = cfg.uploadWebpMode
    await saveConfig('Mellow_functions', full)
    await siteStore.invalidate()
    toast.success('已保存')
  })
}

// ---------- config 保存 ----------
async function saveConfigItem(key, payload, okText) {
  await withSaving(key, async () => {
    await saveSystemConfig({ key: payload.key, ...payload.data })
    toast.success(okText)
  })
}

/**
 * 保存统一邮件通知配置（SYSTEM_MAIL_NOTIFY）
 *
 * - enabled     → json.enabled：总开关（0 关闭时后端所有场景都停发）
 * - admin_email → json.admin_email：管理员收件邮箱（空则后端回退超管账号邮箱）
 * - scenes      → json.scenes：各场景开关（1 开 / 0 关）
 * 保存前与后端原始 scenes 合并，避免把前端没列出的（后端新增的）场景覆盖掉。
 */
async function saveMail() {
  const scenes = { ...cfg.mail.rawScenes }
  Object.keys(cfg.mail.scenes).forEach((key) => {
    scenes[key] = Number(cfg.mail.scenes[key]) === 1 ? 1 : 0
  })

  await saveConfigItem(
    'mail',
    {
      key: 'SYSTEM_MAIL_NOTIFY',
      data: {
        value: cfg.mail.enabled,
        json: {
          enabled: Number(cfg.mail.enabled) === 1 ? 1 : 0,
          admin_email: String(cfg.mail.adminEmail || '').trim(),
          scenes
        }
      }
    },
    '邮件通知配置已保存'
  )

  cfg.mail.rawScenes = { ...scenes }
}

/**
 * 保存装扮配置（SYSTEM_DECORATION）
 * 写入完整结构（含 currency / presets 兜底），避免后台只改一个开关就把其它字段清空；
 * 装扮本身的价格、样式等仍在「装扮管理」（/admin/decoration）里维护。
 */
async function saveDecoration() {
  await saveConfigItem(
    'decoration',
    {
      key: 'SYSTEM_DECORATION',
      data: {
        value: cfg.decoration.enabled,
        json: {
          enabled: Number(cfg.decoration.enabled) === 1 ? 1 : 0,
          show_in_goods: Number(cfg.decoration.showInGoods) === 1 ? 1 : 0,
          currency: 'integral',
          avatar_frame: {
            enabled: Number(cfg.decoration.frameEnabled) === 1 ? 1 : 0
          },
          title: {
            enabled: Number(cfg.decoration.titleEnabled) === 1 ? 1 : 0,
            allow_custom: Number(cfg.decoration.titleAllowCustom) === 1 ? 1 : 0,
            max_length: Math.max(1, Number(cfg.decoration.titleMaxLength) || 10)
          }
        }
      }
    },
    '已保存'
  )
}

/**
 * 保存第三方登录配置（SYSTEM_OAUTH）
 * 每次提交完整结构（三个平台都带上），避免只改一个平台把其它平台的密钥清空
 */
async function saveOauth() {
  const json = {
    auto_register: Number(cfg.oauth.autoRegister) === 1 ? 1 : 0,
    timeout: Math.min(60, Math.max(3, Number(cfg.oauth.timeout) || 15)),
    proxy: String(cfg.oauth.proxy || '').trim()
  }

  OAUTH_PLATFORMS.forEach((item) => {
    const platform = cfg.oauth[item.key]
    json[item.key] = {
      enable: Number(platform.enable) === 1 ? 1 : 0,
      app_id: String(platform.appId || '').trim(),
      app_key: String(platform.appKey || '').trim(),
      redirect: String(platform.redirect || '').trim()
    }
  })

  await saveConfigItem('oauth', { key: 'SYSTEM_OAUTH', data: { value: String(json.auto_register), json } }, '已保存')

  // 回读一次统一口径（后端会补默认值）
  const latest = await readConfig('SYSTEM_OAUTH')
  if (latest) readOauthConfig(latest)
}

/**
 * 保存注册配置（ALLOW_REGISTER 一条记录承载三部分数据）
 * - value：是否允许自行注册（"0"/"1"）
 * - text ：新用户默认权限组 ID 列表，后端 Comm.auth() 注册后按它把用户加进对应权限组
 * - json ：邮箱域名限制 / 注册验证方式 / 欢迎消息开关
 * 注意：只提交本页涉及的字段，避免误清空 remark 等；保存后回读，统一数字/字符串口径。
 */
async function saveRegister() {
  const setting = {
    email_domain_mode: cfg.register.domainMode,
    // 名单文本 → 数组；后端 NormalizeDomains 会再做一次归一化（去 @、小写、去重）
    email_whitelist: parseDomainText(cfg.register.whitelistText),
    email_blacklist: parseDomainText(cfg.register.blacklistText),
    verify_mode: cfg.register.verifyMode,
    welcome_message: Number(cfg.register.welcomeMessage) ? 1 : 0,
    welcome_email: Number(cfg.register.welcomeEmail) ? 1 : 0
  }

  await saveConfigItem(
    'register',
    {
      key: 'ALLOW_REGISTER',
      data: {
        value: cfg.register.value,
        text: cfg.register.ids.map(Number).join(','),
        json: setting
      }
    },
    '已保存'
  )

  const fresh = await readConfig('ALLOW_REGISTER')
  if (fresh) readRegisterConfig(fresh)
}

/**
 * 保存账号修改规则（SYSTEM_ACCOUNT_CHANGE）
 * - value / json.enabled：总开关（两处同步写，与后端 AccountChangeConfig 读取口径一致）
 * - json.cooldown_days  ：冷却天数（0 = 不限制，后端限制 0-365）
 * - json.require_contact：是否要求已绑定手机号或邮箱
 * - json.require_verify ：是否要求身份验证（当前密码 / 验证码）
 * 只提交本页字段，保存后回读统一口径。
 */
async function saveAccount() {
  const enabled = Number(cfg.account.value) === 1 ? 1 : 0
  const json = {
    enabled,
    cooldown_days: Math.max(0, Math.min(365, Number(cfg.account.cooldownDays) || 0)),
    require_contact: Number(cfg.account.requireContact) === 1 ? 1 : 0,
    require_verify: Number(cfg.account.requireVerify) === 1 ? 1 : 0
  }

  await saveConfigItem(
    'accountRule',
    { key: 'SYSTEM_ACCOUNT_CHANGE', data: { value: String(enabled), json } },
    '已保存'
  )

  const fresh = await readConfig('SYSTEM_ACCOUNT_CHANGE')
  if (fresh) readAccountConfig(fresh)
}

/**
 * 保存内容配置（ARTICLE / MOMENTS / PAGE）
 * 这三份的 json 就是配置本体且结构固定，因此直接整体覆盖；
 * 保存后重新读取，避免把数字/字符串差异留在表单里。
 */
async function saveContent(moduleKey) {
  const target = CONTENT_MODULE_BY_TAB[moduleKey]
  if (!target) return
  const item = cfg.content[target.key]

  await withSaving(`content_${moduleKey}`, async () => {
    await saveSystemConfig({
      key: target.key,
      json: {
        editor: item.editor,
        audit: Number(item.audit),
        comment: {
          allow: Number(item.comment.allow),
          show: Number(item.comment.show)
        }
      }
    })
    toast.success('已保存')
    const fresh = await readConfig(target.key)
    if (fresh) cfg.content[target.key] = readContentConfig(fresh)
  })
}

/**
 * 保存评论配置
 *
 * 注意：敏感词已迁到独立的 config/sensitive.toml，这里不再写
 * sensitive_filter / sensitive_words —— 评论提交时的校验统一走公共词库，
 * 见 app/model/sensitive.go。
 */
async function saveComment() {
  await withSaving('comment', async () => {
    await saveSystemConfig({
      key: 'COMMENT',
      json: {
        audit: Number(cfg.comment.audit),
        allow: Number(cfg.comment.allow),
        rate_limit: {
          enabled: Number(cfg.comment.rate_limit.enabled),
          max_count: Number(cfg.comment.rate_limit.max_count),
          time_window: Number(cfg.comment.rate_limit.time_window)
        },
        max_length: Number(cfg.comment.max_length),
        require_chinese: Number(cfg.comment.require_chinese)
      }
    })
    toast.success('已保存')
    const fresh = await readConfig('COMMENT')
    if (fresh) cfg.comment.audit = readCommentConfig(fresh).audit
  })
}

/** 保存友链配置（LINKS）：目前只有审核开关 */
async function saveLinks() {
  await withSaving('links', async () => {
    await saveSystemConfig({
      key: 'LINKS',
      json: { audit: Number(cfg.links.audit) }
    })
    toast.success('已保存')
    const fresh = await readConfig('LINKS')
    cfg.links.audit = fresh ? toText(parseJson(fresh.json, {}).audit, '1') : cfg.links.audit
  })
}

// ---------- toml 保存 ----------
async function saveTomlItem(key, method, data, okText = '已保存') {
  await withSaving(key, async () => {
    await saveToml(method, data)
    toast.success(okText)
    // 保存后重新读取，回填后端规范化后的值（如脱敏串）
    await loadTomlAll()
  })
}

function saveSession() {
  return saveTomlItem('session', 'crypt-session', {
    expire: toml.session.expire
  })
}

function saveCacheDefault() {
  return saveTomlItem('cacheDefault', 'cache-default', {
    value: toml.cache.default,
    open: toml.cache.openValue === '1'
  })
}

function saveCacheRedis() {
  return saveTomlItem('cacheRedis', 'cache-redis', { ...toml.cache.redis })
}

function saveCacheFile() {
  return saveTomlItem('cacheFile', 'cache-file', { ...toml.cache.file })
}

function saveCacheRam() {
  return saveTomlItem('cacheRam', 'cache-ram', { expire: toml.cache.ram.expire })
}

function saveStorageDefault() {
  return saveTomlItem('storageDefault', 'storage-default', { value: toml.storage.default })
}

function saveStorageLocal() {
  return saveTomlItem('storageLocal', 'storage-local', { ...toml.storage.local })
}

function saveStorageCos() {
  return saveTomlItem('storageCos', 'storage-cos', { ...toml.storage.cos })
}

function saveStorageAttachment() {
  return saveTomlItem('storageAttachment', 'storage-attachment', { ...toml.storage.attachment })
}

function saveSmsDrive() {
  return saveTomlItem('smsDrive', 'sms-drive', {
    default: toml.sms.drive.default,
    email: toml.sms.drive.email,
    sms: toml.sms.drive.sms
  })
}

function saveSmsEmail() {
  return saveTomlItem('smsEmail', 'sms-email', { ...toml.sms.email })
}

// 发件队列：PUT /api/toml/sms-email-queue，只提交这 7 个参数（未提交的字段后端保持原值）
function saveSmsEmailQueue() {
  const data = {}
  MAIL_QUEUE_FIELDS.forEach((item) => {
    data[item.key] = Number(toml.sms.emailQueue[item.key] ?? item.def)
  })
  return saveTomlItem('smsEmailQueue', 'sms-email-queue', data, '发件队列配置已保存')
}

function saveSmsAliyun() {
  return saveTomlItem('smsAliyun', 'sms-aliyun', { ...toml.sms.aliyun })
}

function saveSmsAliyunNumberVerify() {
  return saveTomlItem('smsAliyunNumberVerify', 'sms-aliyun-number-verify', { ...toml.sms.aliyunNumberVerify })
}

function saveSmsTencent() {
  return saveTomlItem('smsTencent', 'sms-tencent', { ...toml.sms.tencent })
}

function saveNotification() {
  return saveTomlItem('notification', 'notification', { retention_days: Number(toml.notification.retention_days) })
}

// ---------- 连通性测试 ----------
async function runTest(key, method, data, okText) {
  await withSaving(key, async () => {
    const res = await testToml(method, data)
    const detail = res?.data && typeof res.data === 'string' ? `（${res.data}）` : ''
    toast.success(`${okText}${detail}`)
  })
}

function testRedis() {
  return runTest('testRedis', 'test-redis', { ...toml.cache.redis }, 'Redis 连接成功')
}

function testStorage(kind) {
  // 仅保留腾讯云 COS 的连通性测试（本地存储无需测试连接）
  if (kind === 'cos') {
    return runTest('testCos', 'test-cos', {
      secret_id: toml.storage.cos.secret_id,
      secret_key: toml.storage.cos.secret_key,
      app_id: toml.storage.cos.app_id,
      bucket: toml.storage.cos.bucket,
      region: toml.storage.cos.region
    }, 'COS 连接成功')
  }
}

function testSmsEmail() {
  if (!test.email.trim()) {
    toast.warning('请填写测试收件人')
    return
  }
  return runTest('testSmsEmail', 'test-sms-email', { ...toml.sms.email, email: test.email.trim() }, '测试邮件已发送')
}

function testSmsAliyun() {
  if (!test.phone.trim()) {
    toast.warning('请填写测试手机号')
    return
  }
  return runTest('testSmsAliyun', 'test-sms-aliyun', { ...toml.sms.aliyun, phone: test.phone.trim() }, '测试短信已发送')
}

function testSmsTencent() {
  if (!test.phone.trim()) {
    toast.warning('请填写测试手机号')
    return
  }
  return runTest('testSmsTencent', 'test-sms-tencent', { ...toml.sms.tencent, phone: test.phone.trim() }, '测试短信已发送')
}

onMounted(loadAll)
</script>

<style scoped>
.system-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
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
.inline-link {
  color: var(--link);
  text-decoration: none;
  border-bottom: 1px dashed currentColor;
}
.inline-link:hover {
  color: var(--link-hover);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.stat-error {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 12px 0 0;
  font-size: 12px;
  color: var(--danger);
}

/* ---------- 布局 ---------- */
.config-layout {
  display: grid;
  grid-template-columns: 200px 1fr;
  gap: 16px;
  align-items: start;
}
.config-nav {
  position: sticky;
  top: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
}
.config-nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  font-size: 13px;
  text-align: left;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.config-nav-item:hover {
  background: var(--bg-muted);
  color: var(--primary);
}
.config-nav-item.is-active {
  background: var(--accent-wash);
  color: var(--primary-deep);
  font-weight: 600;
}

.config-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* ---------- 内容配置：文章 / 动态 / 独立页面并排 ---------- */
.content-config {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
/* 三份配置结构相同，宽屏并排对比，窄屏自动折行成竖排 */
.content-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(288px, 1fr));
  gap: 16px;
  align-items: stretch;
}
.content-card {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.content-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 12px;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.content-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border-radius: var(--radius-sm);
}
.content-badge .bi {
  font-size: 15px;
}
.content-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.content-sub {
  margin: 2px 0 0;
  font-size: 11px;
  color: var(--text-muted);
}
/* 保存按钮统一贴底，三张卡片高度对齐时更整齐 */
.content-card .config-foot {
  margin-top: auto;
}

/* 内容配置的内层分页（文章 / 动态 / 独立页面） */
.config-subtabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft, var(--border));
}
.config-subtab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: transparent;
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.config-subtab:hover {
  color: var(--text);
  border-color: var(--primary);
}
.config-subtab.is-active {
  color: var(--primary-deep, var(--primary));
  border-color: var(--primary);
  background: var(--primary-wash, var(--bg-muted));
}

/* ---------- 卡片 ---------- */
.config-head {
  margin-bottom: 16px;
}
.config-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
}
.config-desc {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.config-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--border-soft);
}
/* 单独的保存条（注册模块）：无上边框、按钮靠右 */
.config-foot.is-plain {
  margin-top: 0;
  padding-top: 0;
  border-top: none;
  justify-content: flex-end;
  flex-wrap: wrap;
}
.form-hint.is-inline {
  margin: 0 auto 0 0;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 12px;
}
/* 同一面板内的小节标题（如装扮页的「头像框 / 头衔」） */
.config-sub-title {
  margin: 18px 0 10px;
  padding-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  border-bottom: 1px solid var(--border-soft);
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}
.form-hint code {
  padding: 0 4px;
  font-size: 11px;
  background: var(--bg-muted);
  border-radius: 3px;
}

/* ---------- 上传命名规则（存储页的占位符说明与路径预览） ---------- */
.rule-token {
  display: inline-block;
  margin: 0 10px 2px 0;
  white-space: nowrap;
}
.rule-token code {
  color: var(--primary);
}
.rule-preview {
  word-break: break-all;
}

/* ---------- 多选（注册默认权限组） ---------- */
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
.is-loading.is-inline {
  padding: 8px 0;
  text-align: left;
  font-size: 13px;
}

/* ---------- 只读信息 ---------- */
.readonly-list {
  margin: 0;
}
.readonly-row {
  display: flex;
  gap: 12px;
  padding: 8px 0;
  font-size: 13px;
  border-bottom: 1px dashed var(--border-soft);
}
.readonly-row:last-child {
  border-bottom: none;
}
.readonly-row dt {
  width: 140px;
  flex-shrink: 0;
  color: var(--text-muted);
}
.readonly-row dd {
  margin: 0;
  color: var(--text-soft);
}

.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}

/* 默认头像管理（默认值设置 → 默认头像） */
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 12px 14px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  margin-bottom: 14px;
}
.switch-row .form-label {
  margin-bottom: 0;
}
.switch-row .form-hint {
  margin-top: 2px;
}
.switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  flex-shrink: 0;
  cursor: pointer;
}
.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}
.switch-slider {
  position: absolute;
  inset: 0;
  background: var(--border);
  border-radius: 22px;
  transition: 0.2s;
}
.switch-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 3px;
  top: 3px;
  background: var(--switch-knob, #fff);
  border-radius: 50%;
  transition: 0.2s;
}
.switch input:checked + .switch-slider {
  background: var(--primary);
}
.switch input:checked + .switch-slider::before {
  transform: translateX(18px);
}
.avatar-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.avatar-item {
  position: relative;
  width: 64px;
  height: 64px;
  padding: 4px;
  background: var(--bg-card);
  border: 2px solid var(--border);
  border-radius: var(--radius);
  overflow: hidden;
}
.avatar-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: calc(var(--radius) - 4px);
}
.avatar-del {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  cursor: pointer;
  transition: background 0.18s;
}
.avatar-del:hover {
  background: var(--danger, #e5484d);
}
.avatar-empty {
  margin: 0;
  font-size: 12px;
  color: var(--text-muted);
}
.avatar-add {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
.avatar-add .input {
  flex: 1;
  min-width: 0;
}
.save-hint {
  margin-right: auto;
  font-size: 12px;
  color: var(--text-muted);
}

/* 敏感词 TOML 原文编辑器 */
.config-code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 12px;
  line-height: 1.7;
  tab-size: 2;
}
.config-head-sub {
  margin-top: 22px;
}

/* 敏感词库：可视化规则编辑 */
.mode-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
}
.rule-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.rule-card {
  padding: 12px 14px;
  margin-bottom: 10px;
  background: var(--bg-muted);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
}
.rule-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 160px;
  gap: 0 12px;
}
.rule-grid.is-three {
  grid-template-columns: minmax(0, 1fr) 160px 160px;
}
.scene-chips {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.scene-chip {
  height: 26px;
  padding: 0 10px;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s;
}
.scene-chip:hover {
  border-color: var(--primary);
}
.scene-chip.is-active {
  color: #fff;
  background: var(--primary);
  border-color: var(--primary);
}
.rule-foot {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}
@media (max-width: 900px) {
  .rule-grid,
  .rule-grid.is-three {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (max-width: 900px) {
  .config-layout {
    grid-template-columns: 1fr;
  }
  .config-nav {
    position: static;
    flex-direction: row;
    flex-wrap: wrap;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
