package model

import (
	"errors"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
	"inis/app/facade"
)

type ArticleGroup struct {
	Id       	int    				 `gorm:"type:int(32); comment:主键;" json:"id"`
	Pid         int    				 `gorm:"type:int(32); comment:父级ID; default:0;" json:"pid"`
	Key         string 				 `gorm:"size:256; comment:唯一键; default:Null;" json:"key"`
	Name        string 				 `gorm:"size:32; comment:名称; default:Null;" json:"name"`
	Description string 				 `gorm:"comment:描述; default:Null;" json:"description"`
	Avatar      string 				 `gorm:"size:256; comment:头像; default:Null;" json:"avatar"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

func InitArticleGroup() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&ArticleGroup{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ArticleGroup表迁移失败")
		return
	}

	// 初始化数据：必须同步完成，InitTable 的等待/超时才覆盖得到（见 base.go 的 InitTable 注释）
	initArticleGroupData()
}

// initArticleGroupData - 初始化ArticleGroup表数据
func initArticleGroupData() {

	count, _ := facade.DB.Model(&ArticleGroup{}).Count()
	if count != 0 {
		return
	}

	// 建默认分类统一走 EnsureDefaultArticleGroup（带锁 + 按 key 幂等）
	EnsureDefaultArticleGroup()
}

// defaultArticleGroupMutex - 默认分类的创建锁（见 EnsureDefaultArticleGroup）
var defaultArticleGroupMutex sync.Mutex

// EnsureDefaultArticleGroup - 确保默认分类存在，并返回它的 id（幂等 + 进程内并发安全）
//
// 默认分类原本有**两个**创建点：
//   - 本文件的 initArticleGroupData：分类表为空时创建；
//   - article.go 的 initArticleData：给默认文章准备分类时按 key 创建。
//
// 两者都在后台 goroutine 里「先查后插」，同一秒并发执行就会各插一条 ——
// 后台「文章分类」里于是出现两条内容一致、create_time 相同的「默认分类」（id 不同）。
// 现在两个调用点都收敛到这里：先按 key 查，查不到才插。
func EnsureDefaultArticleGroup() int {
	defaultArticleGroupMutex.Lock()
	defer defaultArticleGroupMutex.Unlock()

	if exist, _ := facade.DB.Model(&ArticleGroup{}).Where("key", "Default-Category").Exist(); exist {
		item, _ := facade.DB.Model(&ArticleGroup{}).Where("key", "Default-Category").Find()
		return cast.ToInt(item["id"])
	}

	item := ArticleGroup{
		Pid:         0,
		Key:         "Default-Category",
		Name:        "默认分类",
		Description: "默认分类",
	}

	if _, err := facade.DB.Model(&item).Create(&item); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "创建默认分类失败")
	}

	return item.Id
}

// AfterSave - 保存后的Hook（包括 create update）
func (this *ArticleGroup) AfterSave(tx *gorm.DB) (err error) {

	if !utils.Is.Empty(this.Key) {
		exist, _ := facade.DB.Model(&ArticleGroup{}).WithTrashed().Where("id", "!=", this.Id).Where("key", this.Key).Exist()
		if exist {
			return errors.New("key 已存在！")
		}
	}
	return
}

// AfterFind - 查询Hook
func (this *ArticleGroup) AfterFind(tx *gorm.DB) (err error) {

	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)

	return
}