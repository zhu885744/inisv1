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

// InitAuthPages - 初始化AuthPages表
func InitAuthPages() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&AuthPages{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "AuthPages表迁移失败")
		return
	}

	// 后台管理页面列表
	// Icon 统一使用 Bootstrap Icons 类名（https://icons.getbootstrap.com/），前端 <i :class="icon"> 直接可用
	pages := []AuthPages{
		{Name: "撰写文章", Icon: "bi bi-pencil-square", Path: "/admin/article/write", Size: "14px"},
		{Name: "文章管理", Icon: "bi bi-file-earmark-text", Path: "/admin/article", Size: "14px"},
		{Name: "文章分类", Icon: "bi bi-collection", Path: "/admin/article/group", Size: "14px"},
		{Name: "用户管理", Icon: "bi bi-people", Path: "/admin/users", Size: "14px"},
		{Name: "评论管理", Icon: "bi bi-chat-square-text", Path: "/admin/comment", Size: "14px"},
		{Name: "公告管理", Icon: "bi bi-megaphone", Path: "/admin/placard", Size: "14px"},
		{Name: "轮播管理", Icon: "bi bi-images", Path: "/admin/banner", Size: "14px"},
		{Name: "标签管理", Icon: "bi bi-tags", Path: "/admin/tags", Size: "14px"},
		{Name: "等级管理", Icon: "bi bi-graph-up-arrow", Path: "/admin/level", Size: "14px"},
		{Name: "经验管理", Icon: "bi bi-star-half", Path: "/admin/exp", Size: "14px"},
		{Name: "商品管理", Icon: "bi bi-bag", Path: "/admin/goods", Size: "14px"},
		{Name: "积分管理", Icon: "bi bi-coin", Path: "/admin/integral", Size: "14px"},
		{Name: "消息通知", Icon: "bi bi-bell", Path: "/admin/message", Size: "14px"},
		{Name: "友链管理", Icon: "bi bi-link-45deg", Path: "/admin/links", Size: "14px"},
		{Name: "系统配置", Icon: "bi bi-gear", Path: "/admin/system", Size: "14px"},
		{Name: "独立页面", Icon: "bi bi-window", Path: "/admin/pages", Size: "14px"},
		{Name: "撰写独立页面", Icon: "bi bi-file-earmark-plus", Path: "/admin/pages/write", Size: "14px"},
		{Name: "友链分组", Icon: "bi bi-diagram-3", Path: "/admin/links/group", Size: "14px"},
		{Name: "权限规则", Icon: "bi bi-shield-check", Path: "/admin/auth/rules", Size: "14px"},
		{Name: "权限分组", Icon: "bi bi-shield-lock", Path: "/admin/auth/group", Size: "14px"},
		{Name: "接口密钥", Icon: "bi bi-key", Path: "/admin/api/keys", Size: "14px"},
		{Name: "IP黑名单", Icon: "bi bi-slash-circle", Path: "/admin/ip/black", Size: "14px"},
		{Name: "IP白名单", Icon: "bi bi-check-circle", Path: "/admin/ip/white", Size: "14px"},
		{Name: "QPS预警", Icon: "bi bi-speedometer2", Path: "/admin/qps/warn", Size: "14px"},
		{Name: "后台页面管理", Icon: "bi bi-layout-text-window", Path: "/admin/auth/pages", Size: "14px"},
		{Name: "动态管理", Icon: "bi bi-chat-square-quote", Path: "/admin/moments", Size: "14px"},
		{Name: "附件管理", Icon: "bi bi-folder2-open", Path: "/admin/attachment", Size: "14px"},
	}

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
}
