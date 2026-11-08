package model

import (
	"fmt"
	"inis/app/facade"
	"strings"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type AuthPages struct {
	Id     int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name   string `gorm:"comment:名称;" json:"name"`
	Path   string `gorm:"comment:路径;" json:"path"`
	Icon   string `gorm:"comment:图标;" json:"icon"`
	Svg    string `gorm:"type:text; comment:SVG图标;" json:"svg"`
	Size   string `gorm:"comment:图标大小; default:'16px';" json:"size"`
	Hash   string `gorm:"comment:哈希值;" json:"hash"`
	Remark string `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// AfterFind - 查询Hook
func (this *AuthPages) AfterFind(tx *gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}

// BeforeCreate - 创建前的Hook
func (this *AuthPages) BeforeCreate(tx *gorm.DB) (err error) {

	exist, _ := facade.DB.Model(&AuthPages{}).WithTrashed().Where("hash", this.Hash).Exist()
	if exist {
		return fmt.Errorf("hash: %s 已存在", this.Hash)
	}

	return
}

// defaultAuthPages - 后台页面权限点（菜单合并后的一级模块）
//
// 菜单合并（37 → 15）：侧边栏只保留这些一级模块入口，子功能（文章分类 / 用户认证 /
// 主题配色 / IP 黑白名单 …）在页内以标签页承载（前端 views/admin/AdminHub.vue）。
// 页面权限同样按一级模块授权：勾选一个模块 = 可以使用该模块下的全部标签页。
//
// Icon 统一使用 Bootstrap Icons 类名（https://icons.getbootstrap.com/），前端 <i :class="icon"> 直接可用
var defaultAuthPages = []AuthPages{
	{Name: "文章管理", Icon: "bi bi-file-earmark-text", Path: "/admin/article", Size: "14px"},
	{Name: "独立页面", Icon: "bi bi-window", Path: "/admin/pages", Size: "14px"},
	// 文档站：先「文档库」选项目，再进项目管目录 + 页面，前台 /docs/:项目 阅读
	{Name: "文档库", Icon: "bi bi-journal-bookmark", Path: "/admin/docs", Size: "14px"},
	// 内容管理：动态 / 公告 / 轮播 / 话题
	{Name: "内容管理", Icon: "bi bi-collection-play", Path: "/admin/content", Size: "14px"},
	// 用户管理：用户列表 / 用户认证 / 认证类型 / 等级 / 经验 / 勋章
	{Name: "用户管理", Icon: "bi bi-people", Path: "/admin/users", Size: "14px"},
	{Name: "评论管理", Icon: "bi bi-chat-square-text", Path: "/admin/comment", Size: "14px"},
	// 商城管理：商品 / 积分 / 装扮 / 签到
	{Name: "商城管理", Icon: "bi bi-bag", Path: "/admin/goods", Size: "14px"},
	{Name: "消息通知", Icon: "bi bi-bell", Path: "/admin/message", Size: "14px"},
	// 聊天管理：私信 / 公共聊天室的命中留证审核 + 聊天室消息、禁言名单、房间设置
	{Name: "聊天管理", Icon: "bi bi-chat-square-dots", Path: "/admin/chat", Size: "14px"},
	{Name: "举报管理", Icon: "bi bi-flag", Path: "/admin/reports", Size: "14px"},
	// 友链管理：友链列表 / 友链分组
	{Name: "友链管理", Icon: "bi bi-link-45deg", Path: "/admin/links", Size: "14px"},
	{Name: "附件管理", Icon: "bi bi-folder2-open", Path: "/admin/attachment", Size: "14px"},
	// 系统配置：网站设置 / 主题配色 / 后台页面管理
	{Name: "系统配置", Icon: "bi bi-gear", Path: "/admin/system", Size: "14px"},
	// 安全中心：权限规则 / 权限分组 / 接口密钥 / IP 黑白名单 / QPS 预警
	{Name: "安全中心", Icon: "bi bi-shield-lock", Path: "/admin/security", Size: "14px"},
	// 数据统计：实时资源 / 数据库 / 在线用户（数据来自 socket 推送，见 app/socket/controller/status.go）
	{Name: "数据统计", Icon: "bi bi-graph-up-arrow", Path: "/admin/stats", Size: "14px"},
	{Name: "查看日志", Icon: "bi bi-journal-text", Path: "/admin/logs", Size: "14px"},
}

// mergedAuthPages - 菜单合并（37 → 15）后被并入一级模块的旧页面路径
//
// key 是合并前的页面路径，value 是它归属的一级模块。
// 老库里这些记录由 mergeAuthPages 软删除（进回收站，可手动恢复），
// 权限组里对它们的勾选由 migrateAuthGroupPages 折算成一级模块。
var mergedAuthPages = map[string]string{
	// 文章管理
	"/admin/article/write": "/admin/article",
	"/admin/article/group": "/admin/article",
	// 独立页面
	"/admin/pages/write": "/admin/pages",
	// 内容管理
	"/admin/moments": "/admin/content",
	"/admin/placard": "/admin/content",
	"/admin/banner":  "/admin/content",
	"/admin/tags":    "/admin/content",
	// 用户管理
	"/admin/certifications": "/admin/users",
	"/admin/cert-types":     "/admin/users",
	"/admin/level":          "/admin/users",
	"/admin/exp":            "/admin/users",
	"/admin/badges":         "/admin/users",
	// 商城管理
	"/admin/integral":   "/admin/goods",
	"/admin/decoration": "/admin/goods",
	"/admin/checkin":    "/admin/goods",
	// 友链管理
	"/admin/links/group": "/admin/links",
	// 系统配置
	"/admin/theme": "/admin/system",
	// 安全中心（后台页面管理维护的是页面权限点，与权限规则同属安全域）
	"/admin/auth/pages": "/admin/security",
	"/admin/auth/rules": "/admin/security",
	"/admin/auth/group": "/admin/security",
	"/admin/api/keys":   "/admin/security",
	"/admin/ip/black":   "/admin/security",
	"/admin/ip/white":   "/admin/security",
	"/admin/qps/warn":   "/admin/security",
}

// InitAuthPages - 初始化AuthPages表
func InitAuthPages() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&AuthPages{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "AuthPages表迁移失败")
		return
	}

	pages := defaultAuthPages

	// legacyIcons - v1 版本的短图标名，仅用于判断「是否可以把旧值升级为 Bootstrap Icons」
	// 管理员自己在「后台页面管理」里改过的图标不会被覆盖
	legacyIcons := map[string]bool{
		"article": true, "group": true, "user": true, "comment": true, "bell": true,
		"banner": true, "tag": true, "level": true, "link": true, "system": true,
		"open": true, "rule": true, "key": true, "qps": true, "white": true,
		"black": true, "file": true,
	}

	wg := sync.WaitGroup{}

	for _, item := range pages {
		wg.Add(1)
		go func(item AuthPages, wg *sync.WaitGroup) {
			defer wg.Done()

			hash := utils.Hash.Sum32(item.Path)

			record, _ := facade.DB.Model(&AuthPages{}).Where("hash", hash).Find()
			if !utils.Is.Empty(record) {
				// 已存在的页面：仅在 icon 为空或仍是旧版短名时升级为 Bootstrap Icons，
				// 管理员自定义过的图标保持不动
				current := cast.ToString(record["icon"])
				if current != item.Icon && (utils.Is.Empty(current) || legacyIcons[current]) {
					if _, err := facade.DB.Model(&AuthPages{}).Where("hash", hash).Update(map[string]any{"icon": item.Icon}); err != nil {
						facade.Log.Error(map[string]any{"error": err.Error()}, "更新页面图标失败")
					}
				}
				return
			}

			_, err := facade.DB.Model(&AuthPages{}).Create(&AuthPages{
				Hash: hash,
				Name: cast.ToString(item.Name),
				Path: cast.ToString(item.Path),
				Icon: cast.ToString(item.Icon),
				Size: cast.ToString(item.Size),
			})

			if err != nil {
				if strings.Contains(err.Error(), "已存在") {
					return
				}
				facade.Log.Error(map[string]any{"error": err.Error()}, "自动添加页面失败")
			}
		}(item, &wg)
	}

	wg.Wait()

	// 兼容旧库：菜单合并后有两个模块换了更贴切的名字（文档多项目化 → 文档库、
	// 商品管理升级为商城管理），这里把仍是旧默认名的记录对齐一次；管理员自定义过名称的不动。
	for path, names := range map[string][2]string{
		"/admin/docs":  {"文档管理", "文档库"},
		"/admin/goods": {"商品管理", "商城管理"},
	} {
		facade.DB.Model(&AuthPages{}).
			Where("path", path).
			Where("name", names[0]).
			Update(map[string]any{"name": names[1]})
	}

	// 合并前遗留的细粒度页面记录：并入一级模块后软删除（见 mergedAuthPages）
	mergeAuthPages()
}

// mergeAuthPages - 把被合并的细粒度页面记录并入一级模块（幂等，可重复执行）
//
// 为什么需要它：seed 只补不删，老库里的「文章分类 / 用户认证 / 主题配色 …」会一直留着 ——
// 后台页面管理里就成了 37 个开关配 15 个菜单，管理员根本对不上。这里把它们软删除
// （进回收站，可在「后台页面管理」里恢复），权限折算见 migrateAuthGroupPages。
//
// 调用时机：InitAuthPages（安装 / 迁移 / 每次启动的维护）
func mergeAuthPages() {

	// 维护任务：任何异常都不应影响服务启动
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "合并后台页面权限时发生panic")
		}
	}()

	total := int64(0)
	for path := range mergedAuthPages {
		// 只处理「存在且未删除」的记录：已软删的跳过，避免每次启动重复计数
		exist, _ := facade.DB.Model(&AuthPages{}).Where("path", path).Exist()
		if !exist {
			continue
		}

		tx, err := facade.DB.Model(&AuthPages{}).Where("path", path).Delete()
		if err != nil {
			facade.Log.Warn(map[string]any{"error": err.Error(), "path": path}, "合并后台页面失败")
			continue
		}
		if tx != nil {
			total += tx.RowsAffected
		}
	}

	if total <= 0 {
		return
	}

	facade.Log.Info(map[string]any{"rows": total}, "已把旧的细粒度后台页面并入一级模块")
	if facade.Cache != nil {
		facade.Cache.DelTags([]any{"[GET]", "auth-pages"})
	}
}

// migrateAuthGroupPages - 把权限组里的旧页面 hash 折算成一级模块 hash（幂等）
//
// auth-group.pages 存的是 auth-pages.hash（Sum32(path)）的逗号串。菜单合并后旧 hash
// 对应的记录已被 mergeAuthPages 软删，不折算的话，那些只勾选过子页面（如「勋章管理」）
// 的权限组会连「用户管理」都看不到。折算规则：勾过某个模块下任意一个子页面，
// 即让该组拥有这个一级模块 —— 这正是「模块级授权」的语义。
//
// 调用时机：InitAuthGroup（EnsureAuthGroups 之后），与页面合并一起构成一次幂等迁移
func migrateAuthGroupPages() {

	// 维护任务：任何异常都不应影响服务启动
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "折算权限组页面权限时发生panic")
		}
	}()

	oldHashToNew := map[string]string{}
	for path, target := range mergedAuthPages {
		oldHashToNew[utils.Hash.Sum32(path)] = utils.Hash.Sum32(target)
	}

	groups, err := facade.DB.Model(&[]AuthGroup{}).WithTrashed().Select()
	if err != nil {
		facade.Log.Warn(map[string]any{"error": err.Error()}, "读取权限组失败")
		return
	}

	changed := 0
	for _, group := range groups {
		raw := cast.ToString(group["pages"])
		if strings.Contains(raw, "all") {
			continue
		}

		next := make([]string, 0, len(raw))
		owned := map[string]bool{}
		for _, hash := range strings.Split(raw, ",") {
			hash = strings.TrimSpace(hash)
			if hash == "" {
				continue
			}
			// 旧页面的 hash → 它归属的一级模块
			if target, ok := oldHashToNew[hash]; ok {
				hash = target
			}
			if owned[hash] {
				continue
			}
			owned[hash] = true
			next = append(next, hash)
		}

		value := strings.Join(next, ",")
		if value == raw {
			continue
		}

		if _, err := facade.DB.Model(&AuthGroup{}).WithTrashed().Where("id", group["id"]).Update(map[string]any{"pages": value}); err != nil {
			facade.Log.Warn(map[string]any{"error": err.Error(), "id": group["id"]}, "折算权限组页面权限失败")
			continue
		}
		changed++
	}

	if changed <= 0 {
		return
	}

	facade.Log.Info(map[string]any{"groups": changed}, "已把权限组的旧页面权限折算到一级模块")
	if facade.Cache != nil {
		facade.Cache.DelTags([]any{"user["})
	}
}
