# 话题 / 标签社区（#话题#）

在**动态、文章**正文里写 `#话题#`，保存时会被规范化成可点击的话题链接，点进话题聚合页
可以看到该话题下的动态与文章；话题还支持**关注、热度榜、内置顶、主持人**。

```
动态 / 文章正文：#旅行#  →  落库：#[旅行](/topic/12)  →  渲染成可点击的 #旅行#
                                    │
                                    话题聚合页 /topic/12（动态 + 文章 + 关注 + 主持人）
```

## 一条核心设计：标签即话题

**不新造一套表** —— 复用现有的 `tags` 表（管理后台的「标签管理」就是话题管理），
只加了两个扩展字段：

| 字段 | 说明 |
| :--- | :--- |
| `moderator_uid` | 话题主持人（用户 id，0 = 未设置） |
| `recommend` | 精选话题（1 = 是，话题广场可筛选） |

好处：

- 存量标签**自动变成话题**，文章上已有的标签立刻能点到话题页，零数据迁移；
- 标签与话题的口径统一，不会出现「标签 A 和话题 A 是两回事」的混乱。

> 为什么**没有** `status`（下架）字段：话题下架直接复用软删除（`delete_time`）。
> 若加 `status`，AutoMigrate 给存量标签补默认值时是 0，会被误判成「已下架」——
> 这类「零值语义」陷阱在 `moments.status` 上已经踩过一次，见 `app/model/tags.go` 的注释。

## 存储格式

保存内容时后端做两件事（`app/model/topics.go` 的 `Topic.Parse`）：

```
用户输入：#旅行# 和 #美食#
落库正文：#旅行#  →  #[旅行](/topic/12) ；#美食#  →  #[美食](/topic/13)
落库标签：content.tags = |12|13|      ← 与文章原有 tags 口径一致
```

- **正文存链接**：文章 / 独立页走 markdown-it（`html: false`，不能存 HTML 标签），
  评论 / 动态走 `Mellow/src/utils/emoji.js` 的 `renderHashtag()`，四处表现一致；
  `#[名称](/topic/12)` 里同时存了话题名与 id，**话题后来改名也不会让链接失效**。
- **标签字段存 id**：`|12|13|` 与文章原有的 `tags` 列口径完全一致，
  话题聚合页用 `LIKE '%|12|%'` 精确检索（避免 `|1|` 误匹配 `|11|`）。

规范化是**幂等**的：已规范化的链接会被占位符保护后原样保留（不会二次拆解、不会重复建话题），
所以草稿反复保存、发布后再次编辑都安全；同时它也会被计入关联列表，
`草稿 → 发布` 这一步不会丢话题。

### 匹配规则

| 规则 | 说明 |
| :--- | :--- |
| 语法 | `#话题#`（成对），话题名内不允许出现 `#` 与空白 |
| markdown 标题不误伤 | `# 标题`、`## 标题` 不会被当成话题（`#` 后跟 `#` 或空格时不匹配） |
| 单条上限 | 一条内容最多关联 **5** 个话题，超出部分按普通文本保留（防刷话题） |
| 名称清洗 | `[ ] ( ) |` 会破坏 markdown 链接语法，建话题时去掉；长度截断到 32（`tags.name` 上限） |
| 自动创建 | 话题不存在时**自动创建**，同名取 id 最小者（`tags` 表没有唯一索引，这样结果确定） |

> 建话题失败（例如数据库异常）**不会影响内容发布**：`FindOrCreate` 里兜住了 panic，
> 该 `#话题#` 退化为普通文本。这条路径在「发布动态」的关键链路上，必须容错。

## 话题关注

表 `tag_follows`（`app/model/tag-follows.go`）：

| 字段 | 说明 |
| :--- | :--- |
| `uid` + `tag_id` | 联合唯一索引 `uk_tag_follows_pair` |
| `status` | 1 = 关注中，0 = 已取消（**软开关，不删行**） |

- 关注 / 取关都是**幂等**的：取关再关注不会撞唯一索引（沿用 `user_follows` 的写法）；
- 查找已有关联时用 `Select()` 取 map，**不要**用 `Find(&struct)` + 原始 SQL 字符串
  —— facade 的 `Where` 是按列名解析的，传 `"uid = ? AND tag_id = ?"` 会被静默丢弃条件，
  最后走到 `Create` 撞唯一索引（`user_blocks` 上踩过这个坑，见 `user-blocks.go` 的注释）；
- 关注数用 `COUNT(status=1)` 实时统计 + 60 秒缓存主动失效，与项目其它计数口径一致。

## 话题内置顶

表 `topic_pins`（`app/model/topic-pins.go`）：`(tag_id, target_type, target_id)` 联合唯一索引，
`target_type` 取 `moments` / `article`。

与内容自身的 `top`（全局置顶）**是两回事**：一条内容可能同时属于多个话题，
每个话题的主持人各自决定在自己话题里置顶哪几条。

置顶排序直接并进 SQL，而不是「先查置顶、再查其余」：

```sql
ORDER BY (id IN (3,7)) DESC, create_time DESC
```

> 为什么这么做：分成两次查询会**打乱分页偏移**，翻页时出现重复项或漏项。
> 拼进 ORDER BY 的 id 全部来自数据库（int），不含用户输入。

## 主持人能力

主持人在话题页可以直接操作话题内的内容（页面上每条内容上方有一排按钮）：

| 操作 | 接口 | 效果 |
| :--- | :--- | :--- |
| 置顶 / 取消置顶 | `topics/pin` · `unpin` | 该内容在本话题内排到最前，带「话题内置顶」徽标 |
| 移出话题 | `topics/remove-content` | 只摘掉 `tags` 关联，**不删除内容**（编辑加回 `#话题#` 即可恢复） |
| 删除内容 | `topics/delete-content` | 软删除内容（后台回收站可恢复），并记一条告警日志 |
| 编辑话题资料 | `topics/moderator-update` | 只能改简介与封面；**话题名不可改**（正文链接按名称写入） |

两道防护（`app/api/controller/topics.go`）：

1. **权限**：`assertModerator()` —— 话题主持人本人或系统管理员，其余 403；
2. **归属校验**：内容必须**真的关联着该话题**（`contentInTopic()`）才允许操作，
   否则主持人能靠猜 id 去置顶 / 删除任意内容。

## 热度榜

热度分（`Topic.Heat`）：

```
heat = 文章数 × 3 + 动态数 × 2 + 关注数 × 2
```

- 内容量统计口径：文章只算 `audit=1`；动态额外要求 `status=1`（已发布，不含草稿）；
- 批量统计（一次查 `tags` 列在内存里汇总），避免「每个话题一次 COUNT」的 N+1；
- 榜单结果缓存 **60 秒**。

> ⚠️ 缓存**必须带过期时间**：关注 / 发布内容时会主动清缓存，但漏清一处也不能变成「永久缓存」。
> 系统配置缓存不带过期时间，正是「后台关掉内容审核却不生效」那类问题的根源（见 `docs/mention.md` 末尾的同类提醒）。

## 接口与权限

接口文档：`docs/API docs/topics.md`。权限在 `app/model/auth-rules.go` 的 `topics` 节点登记：

| 接口 | 方法 | 权限 |
| :--- | :--- | :--- |
| `topics/board` | GET | `type=common` —— 话题榜（含 `keyword` 搜索、`recommend=1` 只看精选） |
| `topics/detail` | GET | `type=common` —— 话题详情（`id` 或 `key`(名称) 二选一） |
| `topics/content` | GET | `type=common` —— 话题内内容聚合（`type`=moments/article、`sort`=new/hot/top） |
| `topics/candidates` | GET | `type=login` —— 编辑器 `#` 联想候选 |
| `topics/mine` | GET | `type=login` —— 我关注的话题 |
| `topics/follow` · `unfollow` | POST | `type=login` |
| `topics/moderator-update` | POST | `type=login` |
| `topics/pin` · `unpin` · `remove-content` · `delete-content` | POST | `type=login` |

管理侧的增删改仍在 `tags` 下（`/api/tags/*`，需要后台「标签管理」页面权限），
可以顺带管理 `moderator_uid` / `recommend` 两个扩展字段（见 `docs/API docs/tags.md`）。

> **升级部署后需要重启一次后端**：权限规则种子（`auth-rules`）只在启动时同步进数据库，
> 否则新接口会按「未登记」处理而返回 403。新表（`tag_follows`、`topic_pins`）与新字段
> （`tags.moderator_uid`、`tags.recommend`、`moments.tags`）同样在启动时由 AutoMigrate 自动创建。

## 前端

| 位置 | 文件 | 说明 |
| :--- | :--- | :--- |
| 话题聚合页 | `views/topic/Detail.vue` | `/topic/:key`（key 支持 id 或名称），动态 / 文章切换、最新 / 最热 / 置顶、关注、主持人操作条 |
| 话题广场 | `views/topic/Index.vue` | `/topics`：我关注的 + 热度榜 + 搜索 |
| 榜单卡片 | `components/TopicBoard.vue` | 广场与 `/feed` 的「话题」tab 共用（含关注交互） |
| 编辑器 | `components/EmojiEditor.vue` | 输入 `#` 弹候选面板；工具栏 `#` 按钮插入 `#` 并弹面板；↑/↓ 选择、回车确认、Esc 关闭 |
| 候选浮层 | `components/MentionPicker.vue` | 与 `@` 提及共用（`items` / `nameKey` 等 prop 切换数据源） |
| 正文渲染 | `utils/emoji.js` | `renderHashtag()`：`#[名称](/topic/12)` → `<a class="hashtag">`（动态、评论） |
| 文章渲染 | `utils/markdown.js` | `link_open` 规则给 `/topic/` 链接补 `class="hashtag"` |
| SPA 拦截 | `main.js` | `v-html` 里的 `<a>` 直接点击会整页刷新，这里拦成路由跳转 |
| 路径兼容 | `router/index.js` | 旧地址 `/tag/:key` → `/topic/:key`；标签云页 `/tags` → 话题广场 `/topics`（该页已删除） |

`/feed` 的「话题」tab 走 `topics/board`（榜单数据，不是动态流），因此没有分页。

## 已知边界

| 场景 | 现状 |
| :--- | :--- |
| 评论、独立页面里的 `#话题#` | **只转成可点击链接**，不参与话题内容聚合 —— 这两张表没有 `tags` 字段。要让它们也参与聚合，需要给对应表加 `tags` 字段并写入关联 id |
| 话题改名 | 正文里的链接显示的是**写入时的名称**，话题页展示的是当前名称；改名后旧内容里的显示名不会自动更新 |
| 话题删除 | 内容里的链接会指向不存在的话题（话题页给出「话题不存在」空态），不会报错 |
