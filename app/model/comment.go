package model

import (
	"inis/app/facade"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type Comment struct {
	Id       int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Pid      int    `gorm:"type:int(32); comment:父级ID; default:0;" json:"pid"`
	Uid      int    `gorm:"type:int(32); comment:用户ID; default:0;" json:"uid"`
	Content  string `gorm:"type:varchar(1024); comment:内容; default:Null;" json:"content"`
	Images   string `gorm:"type:text; comment:图片; default:Null;" json:"images"`
	Ip       string `gorm:"comment:IP; default:Null;" json:"ip"`
	Agent    string `gorm:"type:varchar(512); comment:浏览器信息; default:Null;" json:"agent"`
	BindId   int    `gorm:"type:int(32); comment:绑定ID; default:0;" json:"bind_id"`
	BindType string `gorm:"comment:绑定类型; default:'article';" json:"bind_type"`
	Editor   string `gorm:"comment:编辑器; default:'text';" json:"editor"`
	// Audit 审核状态（0=待审核 1=审核通过 2=审核未通过，取值见 app/api/controller/audit.go）
	//
	// 是否开启审核由 COMMENT 配置的 json.audit 决定（后台「系统设置 → 评论配置 → 内容审核」）：
	// 关闭时前台提交直接记为「已通过」，开启时记为「待审核」，非管理员在接口层看不到未通过的评论。
	// 默认值给 1（已通过）：AutoMigrate 给历史评论补这一列时按「已通过」填充，
	// 否则开启审核后老评论会全部从列表里消失。
	Audit  int    `gorm:"type:int(12); index; comment:审核; default:1;" json:"audit"`
	Reason string `gorm:"size:512; comment:审核未通过原因; default:Null;" json:"reason"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// InitComment - 初始化Comment表
func InitComment() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&Comment{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "Comment表迁移失败")
		return
	}

	// 初始化数据：必须同步完成，InitTable 的等待/超时才覆盖得到（见 base.go 的 InitTable 注释）
	initCommentData()
}

// initCommentData - 初始化Comment表数据
func initCommentData() {

	count, _ := facade.DB.Model(&Comment{}).Count()
	if count != 0 {
		return
	}

	// 确保默认文章已存在，避免异步初始化顺序问题
	article := Article{
		Uid:    1,
		Title:  "欢迎使用 inis",
		Content: "如果您看到这篇文章，表示您的 blog 已经安装成功.",
		Audit:  1,
		Status: 1,
	}
	exist, _ := facade.DB.Model(&Article{}).Where("title", "欢迎使用 inis").Exist()
	if !exist {
		facade.DB.Model(&article).Create(&article)
	} else {
		item, _ := facade.DB.Model(&Article{}).Where("title", "欢迎使用 inis").Find()
		article.Id = cast.ToInt(item["id"])
	}

	// 默认评论关联到默认文章
	comment := Comment{
		Pid:      0,
		Uid:      1,
		Content:  "欢迎来到 inis，这是一条默认评论，祝您使用愉快！",
		BindId:   article.Id,
		BindType: "article",
		Editor:   "text",
		Audit:    1, // 已通过：种子数据是站点自己的欢迎评论
	}

	facade.DB.Model(&comment).Create(&comment)
}

// AfterFind - 查询Hook
func (this *Comment) AfterFind(tx *gorm.DB) (err error) {

	// 同步获取结果，避免在批量查询时创建大量协程
	this.Result = this.syncResult()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// syncResult - 同步返回结果
func (this *Comment) syncResult() (result map[string]any) {

	var page, author, article, moments any

	// 同步调用，避免协程开销
	this.pageSync(&page)
	this.authorSync(&author)
	this.articleSync(&article)
	this.momentsSync(&moments)

	return map[string]any{
		"page":    page,
		"author":  author,
		"article": article,
		"moments": moments,
	}
}

func (this *Comment) authorSync(result *any) {
	user, _ := facade.DB.Model(&Users{}).Find(this.Uid)
	author := utils.Map.WithField(user, []string{"id", "nickname", "avatar", "title", "description", "json", "result"})
	// 公开内容里内嵌的作者：清洗 result.auth（权限组）与封禁记录里的管理侧字段
	SanitizeAuthor(author)
	*result = author
}

func (this *Comment) articleSync(result *any) {
	if this.BindType != "article" {
		return
	}

	article, _ := facade.DB.Model(&Article{}).Find(this.BindId)
	*result = utils.Map.WithField(article, []string{"id", "title"})
}

func (this *Comment) pageSync(result *any) {
	if this.BindType != "page" {
		return
	}

	page, _ := facade.DB.Model(&Pages{}).Find(this.BindId)
	*result = utils.Map.WithField(page, []string{"id", "key", "title"})
}

func (this *Comment) momentsSync(result *any) {
	if this.BindType != "moments" {
		return
	}

	moments, _ := facade.DB.Model(&Moments{}).Find(this.BindId)
	*result = utils.Map.WithField(moments, []string{"id", "content"})
}

// author - 解析作者信息（保留原有方法，兼容可能的其他调用）
func (this *Comment) author(wg *sync.WaitGroup, result *any) {
	defer wg.Done()
	this.authorSync(result)
}

// article - 解析文章信息（保留原有方法，兼容可能的其他调用）
func (this *Comment) article(wg *sync.WaitGroup, result *any) {
	defer wg.Done()
	this.articleSync(result)
}

// page - 解析页面信息（保留原有方法，兼容可能的其他调用）
func (this *Comment) page(wg *sync.WaitGroup, result *any) {
	defer wg.Done()
	this.pageSync(result)
}
