package model

import (
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

// BadgeCategory - 勋章分类（成长 / 荣誉 / 成就 / 限定……）
//
// 设计取舍：
//   - 分类是**独立表**而不是枚举字段 —— 后台要能自由增删改、排序、停用，
//     枚举一旦写死，加一类就得改代码重新发布；
//   - 勋章通过 Badge.CategoryId 关联，0 表示「未分类」（历史数据兜底）；
//   - 分类的删除不级联删勋章（勋章是资产，用户的持有记录还在），
//     只把勋章退化为「未分类」，见 controller 的 badge-category remove。
type BadgeCategory struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name        string `gorm:"size:32; comment:分类名称;" json:"name"`
	Description string `gorm:"comment:分类说明; default:Null;" json:"description"`
	Icon        string `gorm:"size:64; comment:图标（Bootstrap Icons 类名）; default:'bi bi-collection';" json:"icon"`
	Color       string `gorm:"size:32; comment:主题色; default:Null;" json:"color"`
	Cover       string `gorm:"comment:分类封面（可选，前台分组头部用）; default:Null;" json:"cover"`
	Sort        int    `gorm:"type:int(32); comment:排序（越大越前）; default:0;" json:"sort"`
	Status      int    `gorm:"tinyint; index; comment:状态（1 启用 0 停用）; default:1;" json:"status"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// AfterFind - 查询Hook
func (this *BadgeCategory) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// InitBadgeCategory - 初始化勋章分类表
//
// 必须排在 InitBadge 之后：播种时要读 Badge 表给存量勋章归类。
func InitBadgeCategory() {

	// 只在「表是这次才建出来」的时候播种：管理员把分类全删光是一种明确意图，
	// 下次重启不该又把四个默认分类塞回去
	firstRun := !facade.DB.Drive().Migrator().HasTable(&BadgeCategory{})

	if err := facade.DB.Drive().AutoMigrate(&BadgeCategory{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "BadgeCategory表迁移失败")
		return
	}

	if firstRun {
		seedBadgeCategory()
	}
}

// badgeCategoryMutex - 播种锁
//
// 分类表原本有两个写入点（InitTable 与安装完成那一刻的 task），
// 「先查数量、再插入」并发执行会插出两组一模一样的分类，和文章分类当年踩的是同一个坑。
var badgeCategoryMutex sync.Mutex

// seedBadgeCategory - 首次运行播种默认分类
//
// 只在**分类表为空**时执行一次，不覆盖管理员改过的内容（改过头像/删过分类都不会被还原）。
// 同时给「功能上线前就存在」的勋章补一个分类，避免升级后整页都是「未分类」：
// 手动发放的归到「荣誉勋章」，其余按条件自动获得的归到「成长勋章」。
// 已经分好类的（category_id > 0）不动。
func seedBadgeCategory() {

	badgeCategoryMutex.Lock()
	defer badgeCategoryMutex.Unlock()

	count, _ := facade.DB.Model(&BadgeCategory{}).Count()
	if count > 0 {
		return
	}

	defaults := []BadgeCategory{
		{
			Name: "成长勋章", Description: "靠日常活跃攒出来的勋章：发文、动态、评论、签到、经验", Icon: "bi bi-graph-up-arrow",
			Color: "#4f7cff", Sort: 100, Status: 1,
		},
		{
			Name: "荣誉勋章", Description: "官方手动授予的荣誉：活动奖励、贡献表彰", Icon: "bi bi-award",
			Color: "#d4a017", Sort: 90, Status: 1,
		},
		{
			Name: "成就勋章", Description: "达成阶段性成就才能点亮的高价值勋章", Icon: "bi bi-trophy",
			Color: "#7c5cff", Sort: 80, Status: 1,
		},
		{
			Name: "限定勋章", Description: "限时活动 / 特定版本发放，过期不再产出", Icon: "bi bi-stars",
			Color: "#e5484d", Sort: 70, Status: 1,
		},
	}

	ids := make([]int, 0, len(defaults))
	for i := range defaults {
		item := defaults[i]
		if _, err := facade.DB.Model(&BadgeCategory{}).Create(&item); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "name": item.Name}, "创建默认勋章分类失败")
			continue
		}
		ids = append(ids, item.Id)
	}

	if len(ids) == 0 {
		return
	}

	// 存量勋章归类：手动 → 荣誉（ids[1]），其余 → 成长（ids[0]）
	if len(ids) >= 2 {
		facade.DB.Drive().Model(&Badge{}).
			Where("category_id", 0).
			Where("condition", BadgeConditionManual).
			UpdateColumn("category_id", ids[1])
	}
	facade.DB.Drive().Model(&Badge{}).
		Where("category_id", 0).
		UpdateColumn("category_id", ids[0])

	facade.Log.Info(map[string]any{"categories": len(ids)}, "已初始化默认勋章分类")
}

// BadgeCategoryEnabled - 启用中的分类（排序与勋章一致：sort desc, id asc）
func BadgeCategoryEnabled() []BadgeCategory {
	var list []BadgeCategory
	facade.DB.Model(&list).Where("status", 1).Order("sort desc, id asc").Select()
	return list
}

// BadgeCategoryMap - 全部分类（含停用）的 id → 分类，用于给勋章补分类信息
func BadgeCategoryMap() map[int]BadgeCategory {
	var list []BadgeCategory
	facade.DB.Model(&list).Select()
	result := make(map[int]BadgeCategory, len(list))
	for _, item := range list {
		result[item.Id] = item
	}
	return result
}

// BadgeCategoryCounts - 每个分类下的勋章数量（key=0 是「未分类」）
func BadgeCategoryCounts() map[int]int {
	result := make(map[int]int)
	type row struct {
		CategoryId int
		Total      int
	}
	var rows []row
	facade.DB.Drive().Model(&Badge{}).
		Select("category_id, COUNT(*) as total").
		Group("category_id").
		Scan(&rows)
	for _, item := range rows {
		result[item.CategoryId] = item.Total
	}
	return result
}
