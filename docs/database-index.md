# 数据库索引优化建议

## 概述

本文档基于对项目代码的分析，提供数据库索引优化建议。合理的索引设计可以显著提升查询性能，减少慢查询。

## 索引设计原则

1. **最左前缀原则**：复合索引的查询效率取决于最左边的列
2. **避免过度索引**：索引会降低写入性能，只在必要时创建
3. **数据分布原则**：基数高的列适合创建索引
4. **覆盖索引**：将常用查询字段包含在索引中，避免回表

---

## 表索引建议

### 1. articles 文章表

**常用查询模式**：
- 根据 `id` 查询单篇文章
- 根据 `audit`（审核状态）和 `status`（发布状态）筛选
- 根据 `uid`（作者）查询
- 根据 `group`（分类）筛选
- 根据 `publish_time` 排序
- 根据 `tags` 查询

```sql
-- 主键索引（已存在）
ALTER TABLE articles ADD PRIMARY KEY (id);

-- 审核状态 + 状态组合索引（首页列表查询）
ALTER TABLE articles ADD INDEX idx_articles_audit_status (audit, status);

-- 作者查询索引（用户文章列表）
ALTER TABLE articles ADD INDEX idx_articles_uid (uid);

-- 分类查询索引
ALTER TABLE articles ADD INDEX idx_articles_group (group);

-- 发布时间索引（排序）
ALTER TABLE articles ADD INDEX idx_articles_publish_time (publish_time);

-- 置顶 + 发布时间索引（首页排序）
ALTER TABLE articles ADD INDEX idx_articles_top_publish (top DESC, publish_time DESC);

-- 浏览量索引（热门文章）
ALTER TABLE articles ADD INDEX idx_articles_views (views);
```

### 2. users 用户表

**常用查询模式**：
- 根据 `id` 查询用户
- 根据 `email` 登录验证
- 根据 `account` 查询
- 根据 `status`（状态）筛选
- 根据 `create_time` 排序

```sql
-- 主键索引（已存在）
ALTER TABLE users ADD PRIMARY KEY (id);

-- 邮箱唯一索引（登录、注册验证）
ALTER TABLE users ADD UNIQUE INDEX idx_users_email (email);

-- 账号唯一索引（登录）
ALTER TABLE users ADD UNIQUE INDEX idx_users_account (account);

-- 手机号唯一索引（登录）
ALTER TABLE users ADD UNIQUE INDEX idx_users_phone (phone);

-- 状态索引（用户列表筛选）
ALTER TABLE users ADD INDEX idx_users_status (status);

-- 创建时间索引（用户列表排序）
ALTER TABLE users ADD INDEX idx_users_create_time (create_time);

-- 来源索引（第三方登录统计）
ALTER TABLE users ADD INDEX idx_users_source (source);
```

### 3. comments 评论表

**常用查询模式**：
- 根据 `bind_id` + `bind_type` 查询评论列表
- 根据 `uid`（评论者）查询
- 根据 `pid`（父评论）查询回复
- 根据 `create_time` 排序

```sql
-- 主键索引（已存在）
ALTER TABLE comments ADD PRIMARY KEY (id);

-- 绑定类型 + 绑定ID组合索引（文章/页面评论查询）
ALTER TABLE comments ADD INDEX idx_comments_bind (bind_type, bind_id);

-- 用户评论索引
ALTER TABLE comments ADD INDEX idx_comments_uid (uid);

-- 父评论索引（回复查询）
ALTER TABLE comments ADD INDEX idx_comments_pid (pid);

-- 创建时间索引（排序）
ALTER TABLE comments ADD INDEX idx_comments_create_time (create_time);

-- 复合索引：绑定类型 + 绑定ID + 创建时间（分页查询）
ALTER TABLE comments ADD INDEX idx_comments_bind_time (bind_type, bind_id, create_time DESC);
```

### 4. config 配置表

**常用查询模式**：
- 根据 `key` 查询配置项

```sql
-- 主键索引（已存在）
ALTER TABLE config ADD PRIMARY KEY (id);

-- 配置键唯一索引
ALTER TABLE config ADD UNIQUE INDEX idx_config_key (key);
```

### 5. auth_group 权限组表

**常用查询模式**：
- 根据 `uids` 查询用户所属组（LIKE 查询）
- 根据 `root` 判断超级管理员

```sql
-- 主键索引（已存在）
ALTER TABLE auth_group ADD PRIMARY KEY (id);

-- root 索引（超级管理员判断）
ALTER TABLE auth_group ADD INDEX idx_auth_group_root (root);
```

### 6. auth_rules 权限规则表

**常用查询模式**：
- 根据 `hash` 查询规则
- 根据 `pid` 查找子规则

```sql
-- 主键索引（已存在）
ALTER TABLE auth_rules ADD PRIMARY KEY (id);

-- hash 索引（权限校验）
ALTER TABLE auth_rules ADD INDEX idx_auth_rules_hash (hash);

-- 父级ID索引（规则树查询）
ALTER TABLE auth_rules ADD INDEX idx_auth_rules_pid (pid);
```

### 7. tags 标签表

**常用查询模式**：
- 根据 `id` 查询标签
- 根据 `name` 查询标签（**话题自动创建 / 按名访问话题页**都走这个查询）
- 按 `moderator_uid`（话题主持人）筛选 —— 低频，不建索引
- 按 `recommend`（精选话题）筛选 —— 基数极低，不建索引

```sql
-- 主键索引（已存在）
ALTER TABLE tags ADD PRIMARY KEY (id);

-- 标签名称索引（话题按名查找：Topic.FindOrCreate / topics/detail?key=xxx）
ALTER TABLE tags ADD INDEX idx_tags_name (name);
```

> **话题（Hashtag）导致的额外查询**：正文里的 `#话题#` 会写入内容的 `tags` 字段
> （`inis_moment.tags` / `inis_article.tags`，格式 `|1|2|3|`），话题聚合页用
> `LIKE '%|12|%'` 反查内容。这种「值包裹在字符串里」的存储**无法命中索引**，
> 属于全表扫描。当前数据量下可接受（详见 `docs/topics.md`）；如果内容量级很大，
> 需要改造成「内容-话题」关联表（`content_id + tag_id` 双列索引）再查。
> 热度榜的内容量统计（`Topic.Counts`）也依赖这一列，同样是一次全表扫描后内存汇总。

### 8. article_group 文章分类表

**常用查询模式**：
- 根据 `id` 查询分类
- 根据 `pid` 查询子分类

```sql
-- 主键索引（已存在）
ALTER TABLE article_group ADD PRIMARY KEY (id);

-- 父级ID索引（分类树查询）
ALTER TABLE article_group ADD INDEX idx_article_group_pid (pid);
```

### 9. exp 经验值表

**常用查询模式**：
- 根据 `uid` + `type` 查询经验记录
- 根据 `bind_id` + `bind_type` 查询

```sql
-- 主键索引（已存在）
ALTER TABLE exp ADD PRIMARY KEY (id);

-- 用户经验索引
ALTER TABLE exp ADD INDEX idx_exp_uid (uid);

-- 类型索引
ALTER TABLE exp ADD INDEX idx_exp_type (type);

-- 绑定索引
ALTER TABLE exp ADD INDEX idx_exp_bind (bind_type, bind_id);

-- 复合索引：用户 + 类型（统计查询）
ALTER TABLE exp ADD INDEX idx_exp_uid_type (uid, type);
```

### 10. moments 动态表

**常用查询模式**：
- 根据 `uid` 查询用户动态
- 根据 `create_time` 排序

```sql
-- 主键索引（已存在）
ALTER TABLE moments ADD PRIMARY KEY (id);

-- 用户动态索引
ALTER TABLE moments ADD INDEX idx_moments_uid (uid);

-- 创建时间索引
ALTER TABLE moments ADD INDEX idx_moments_create_time (create_time);

-- 复合索引：用户 + 时间（分页查询）
ALTER TABLE moments ADD INDEX idx_moments_uid_time (uid, create_time DESC);
```

### 11. links 友链表

**常用查询模式**：
- 根据 `state`（状态）筛选

```sql
-- 主键索引（已存在）
ALTER TABLE links ADD PRIMARY KEY (id);

-- 状态索引
ALTER TABLE links ADD INDEX idx_links_state (state);
```

### 12. pages 页面表

**常用查询模式**：
- 根据 `key` 查询页面
- 根据 `uid` 查询用户页面

```sql
-- 主键索引（已存在）
ALTER TABLE pages ADD PRIMARY KEY (id);

-- 页面标识索引
ALTER TABLE pages ADD INDEX idx_pages_key (key);

-- 用户页面索引
ALTER TABLE pages ADD INDEX idx_pages_uid (uid);
```

### 13. integral_card 积分卡密表

**常用查询模式**：
- 根据 `card` 查询卡密（兑换核心路径，需唯一）
- 根据 `status`（使用状态）筛选
- 根据 `batch`（批次）管理
- 根据 `uid` 查询某用户兑换的卡密
- 根据 `expire_time` 判断是否过期

```sql
-- 主键索引（已存在）
ALTER TABLE integral_card ADD PRIMARY KEY (id);

-- 卡密唯一索引（兑换、去重）
ALTER TABLE integral_card ADD UNIQUE INDEX uk_integral_card (card);

-- 状态索引（未使用/已使用筛选）
ALTER TABLE integral_card ADD INDEX idx_integral_card_status (status);

-- 面额索引
ALTER TABLE integral_card ADD INDEX idx_integral_card_value (value);

-- 批次索引（按批次管理）
ALTER TABLE integral_card ADD INDEX idx_integral_card_batch (batch);

-- 过期时间索引
ALTER TABLE integral_card ADD INDEX idx_integral_card_expire_time (expire_time);

-- 使用者索引
ALTER TABLE integral_card ADD INDEX idx_integral_card_uid (uid);
```

> 说明：以上索引均在模型 `IntegralCard` 中通过 GORM 结构体 tag 声明，`AutoMigrate` 会自动创建，无需手工执行。

### 14. tag_follows 话题关注表

**常用查询模式**：
- 按 `uid` 查「我关注的话题」（`uid + status`）
- 按 `tag_id` 查关注数（`tag_id + status`）
- 按 `uid + tag_id` 判断是否已关注（唯一索引直接命中）

```sql
-- 联合唯一索引：一个用户对同一话题只有一条记录（关注 / 取关复用 status，不删行）
ALTER TABLE tag_follows ADD UNIQUE INDEX uk_tag_follows_pair (uid, tag_id);

-- 我关注的话题
ALTER TABLE tag_follows ADD INDEX idx_tag_follows_uid_status (uid, status);

-- 话题关注数
ALTER TABLE tag_follows ADD INDEX idx_tag_follows_tag_status (tag_id, status);
```

### 15. topic_pins 话题内置顶表

**常用查询模式**：
- 按 `tag_id + target_type` 取该话题内置顶的内容 id 列表
- 按 `tag_id + target_type + target_id` 判断是否已置顶（唯一索引直接命中）
- 按 `target_type + target_id` 清理某条内容在所有话题里的置顶

```sql
-- 联合唯一索引：同一话题内同一条内容只能置顶一次
ALTER TABLE topic_pins ADD UNIQUE INDEX uk_topic_pins_target (tag_id, target_type, target_id);

-- 取话题内置顶列表 / 清内容置顶
ALTER TABLE topic_pins ADD INDEX idx_topic_pins_tag (tag_id, target_type);
ALTER TABLE topic_pins ADD INDEX idx_topic_pins_target (target_type, target_id);
```

> 说明：以上两张表的索引都在模型（`TagFollows` / `TopicPins`）里用 GORM tag 声明，
> `AutoMigrate` 会自动创建，无需手工执行。

---

## 索引创建脚本

> **注意**：系统使用 GORM 的 `SingularTable: true` 配置，表名前缀为 `inis_`，表名为单数形式。

```sql
-- ==================== article 表 ====================
ALTER TABLE inis_article ADD INDEX idx_article_audit_status (audit, status);
ALTER TABLE inis_article ADD INDEX idx_article_uid (uid);
ALTER TABLE inis_article ADD INDEX idx_article_group (`group`);
ALTER TABLE inis_article ADD INDEX idx_article_publish_time (publish_time);
ALTER TABLE inis_article ADD INDEX idx_article_top_publish (top DESC, publish_time DESC);
ALTER TABLE inis_article ADD INDEX idx_article_views (views);

-- ==================== user 表 ====================
ALTER TABLE inis_user ADD UNIQUE INDEX idx_user_email (email);
ALTER TABLE inis_user ADD UNIQUE INDEX idx_user_account (account);
ALTER TABLE inis_user ADD UNIQUE INDEX idx_user_phone (phone);
ALTER TABLE inis_user ADD INDEX idx_user_status (status);
ALTER TABLE inis_user ADD INDEX idx_user_create_time (create_time);
ALTER TABLE inis_user ADD INDEX idx_user_source (source);

-- ==================== comment 表 ====================
ALTER TABLE inis_comment ADD INDEX idx_comment_bind (bind_type, bind_id);
ALTER TABLE inis_comment ADD INDEX idx_comment_uid (uid);
ALTER TABLE inis_comment ADD INDEX idx_comment_pid (pid);
ALTER TABLE inis_comment ADD INDEX idx_comment_create_time (create_time);
ALTER TABLE inis_comment ADD INDEX idx_comment_bind_time (bind_type, bind_id, create_time DESC);

-- ==================== config 表 ====================
ALTER TABLE inis_config ADD UNIQUE INDEX idx_config_key (`key`);

-- ==================== auth_group 表 ====================
ALTER TABLE inis_auth_group ADD INDEX idx_auth_group_root (root);

-- ==================== auth_rules 表 ====================
ALTER TABLE inis_auth_rules ADD INDEX idx_auth_rules_hash (hash);
ALTER TABLE inis_auth_rules ADD INDEX idx_auth_rules_pid (pid);

-- ==================== tag 表 ====================
ALTER TABLE inis_tag ADD INDEX idx_tag_name (name);

-- ==================== article_group 表 ====================
ALTER TABLE inis_article_group ADD INDEX idx_article_group_pid (pid);

-- ==================== exp 表 ====================
ALTER TABLE inis_exp ADD INDEX idx_exp_uid (uid);
ALTER TABLE inis_exp ADD INDEX idx_exp_type (type);
ALTER TABLE inis_exp ADD INDEX idx_exp_bind (bind_type, bind_id);
ALTER TABLE inis_exp ADD INDEX idx_exp_uid_type (uid, type);

-- ==================== moment 表 ====================
ALTER TABLE inis_moment ADD INDEX idx_moment_uid (uid);
ALTER TABLE inis_moment ADD INDEX idx_moment_create_time (create_time);
ALTER TABLE inis_moment ADD INDEX idx_moment_uid_time (uid, create_time DESC);

-- ==================== link 表 ====================
ALTER TABLE inis_link ADD INDEX idx_link_state (state);

-- ==================== page 表 ====================
ALTER TABLE inis_page ADD INDEX idx_page_key (`key`);
ALTER TABLE inis_page ADD INDEX idx_page_uid (uid);

-- ==================== integral_card 表 ====================
ALTER TABLE inis_integral_card ADD UNIQUE INDEX uk_integral_card (card);
ALTER TABLE inis_integral_card ADD INDEX idx_integral_card_status (status);
ALTER TABLE inis_integral_card ADD INDEX idx_integral_card_value (value);
ALTER TABLE inis_integral_card ADD INDEX idx_integral_card_batch (batch);
ALTER TABLE inis_integral_card ADD INDEX idx_integral_card_expire_time (expire_time);
ALTER TABLE inis_integral_card ADD INDEX idx_integral_card_uid (uid);

-- ==================== tag_follows（话题关注）表 ====================
ALTER TABLE inis_tag_follows ADD UNIQUE INDEX uk_tag_follows_pair (uid, tag_id);
ALTER TABLE inis_tag_follows ADD INDEX idx_tag_follows_uid_status (uid, status);
ALTER TABLE inis_tag_follows ADD INDEX idx_tag_follows_tag_status (tag_id, status);

-- ==================== topic_pins（话题内置顶）表 ====================
ALTER TABLE inis_topic_pins ADD UNIQUE INDEX uk_topic_pins_target (tag_id, target_type, target_id);
ALTER TABLE inis_topic_pins ADD INDEX idx_topic_pins_tag (tag_id, target_type);
ALTER TABLE inis_topic_pins ADD INDEX idx_topic_pins_target (target_type, target_id);
```

---

## 查询缓存策略

### 1. 查询缓存配置

系统已支持查询缓存功能，配置文件路径：`config/cache.toml`

```toml
[cache]
mode = "redis"  # redis, file, ram
open = true     # 是否开启缓存
```

### 2. 缓存使用建议

| 场景 | 缓存策略 | 过期时间 |
|------|----------|----------|
| 文章列表 | 标签缓存 `[GET]article` | 默认（可配置） |
| 用户信息 | 单条缓存 `user[id]` | 默认 |
| 评论列表 | 标签缓存 `[GET]comment` | 默认 |
| 配置项 | 单条缓存 `config[key]` | 默认 |
| 权限规则 | 单条缓存 `user[id][rule-group]` | 永久 |

### 3. 缓存失效策略

- **增删改操作后**：自动删除相关标签缓存
- **配置变更后**：手动调用 `facade.Cache.DelTags()` 删除缓存
- **定时刷新**：可通过定时任务定期清理过期缓存

---

## 慢查询日志配置

### MySQL 配置

在 MySQL 配置文件中添加以下配置：

```ini
[mysqld]
slow_query_log = ON
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 2
log_queries_not_using_indexes = ON
```

### 慢查询分析工具

1. **mysqldumpslow**：MySQL 自带的慢查询分析工具
2. **pt-query-digest**：Percona Toolkit 工具集
3. **Explain**：分析查询执行计划

```sql
EXPLAIN SELECT * FROM articles WHERE audit = 1 AND status = 1 ORDER BY publish_time DESC LIMIT 10;
```

---

## 性能监控建议

### 1. 监控指标

| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| Query Cache Hit Rate | 查询缓存命中率 | < 90% |
| Slow Queries | 慢查询数量 | > 10/min |
| Connection Usage | 连接使用率 | > 80% |
| Lock Waits | 锁等待时间 | > 1s |
| Index Usage | 索引使用率 | < 90% |

### 2. 监控工具

- **Prometheus + Grafana**：系统级监控
- **MySQL Exporter**：MySQL 性能指标采集
- **Go pprof**：Go 程序性能分析

---

## 注意事项

1. **索引维护成本**：索引会增加写入操作的开销，需要权衡读写比例
2. **索引选择性**：低选择性的列（如 `status`）创建索引效果有限
3. **复合索引顺序**：将等值条件列放在前面，范围条件列放在后面
4. **定期维护**：定期使用 `ANALYZE TABLE` 更新表统计信息
5. **版本兼容**：索引操作可能影响数据库性能，建议在低峰期执行