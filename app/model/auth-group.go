package model

import (
	"errors"
	"fmt"
	"inis/app/facade"
	"strings"
	"sync"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type AuthGroup struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name    string `gorm:"comment:权限名称;" json:"name"`
	Key     string `gorm:"size:256; comment:唯一键; default:Null;" json:"key"`
	Uids    string `gorm:"type:text; comment:用户ID;" json:"uids"`
	Root    int    `gorm:"type:int(32); comment:'是否拥有越权限操作数据的能力'; default:0;" json:"root"`
	Rules   string `gorm:"type:text; comment:权限规则;" json:"rules"`
	Default int    `gorm:"type:int(32); comment:默认权限; default:0;" json:"default"`
	Pages   string `gorm:"type:text; comment:页面权限; default:Null;" json:"pages"`
	Remark  string `gorm:"comment:备注; default:Null;" json:"remark"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// OrdinaryAuthGroupKey - 系统内置「普通用户」分组的唯一键
const OrdinaryAuthGroupKey = "ordinary"

// builtinAuthGroup - 系统内置分组定义（安装时播种 / 已安装的库由 EnsureAuthGroups 补齐）
//
// 两个内置分组的定位：
//   - admin（超级管理员）：root=1 + rules/pages=all，校验一律放行，禁止删除；
//   - ordinary（普通用户）：root=0、pages 为空，只带「发布 / 管理 / 删除自己的
//     文章与动态」这几个权限点（见 ordinaryAuthPermits），不含任何后台管理能力。
//     这些接口在控制器里都有 `!meta.root → where uid = 当前用户` 的限定，
//     所以普通用户只能操作自己的数据；评论 / 友链 / 附件上传是 type=login，
//     登录即放行，不需要权限点。
type builtinAuthGroup struct {
	Id      int
	Name    string
	Key     string
	Uids    string
	Rules   string
	Pages   string
	Root    int
	Default int
	Remark  string
}

// ordinaryAuthPermits - 「普通用户」内置分组自带的权限点（"METHOD /api/模块/动作"）
//
// 只给「发布 / 编辑 / 删除自己的内容」，一律不给后台管理类接口
// （全站列表统计、审核、置顶、清空回收站等）。
// 评论区与友链的写接口是 type=login（登录即放行），不需要在这里列；
// 附件上传同样只需登录，普通人发文配图不受影响。
var ordinaryAuthPermits = []string{
	// 文章：发布 / 编辑 / 回收站恢复 / 删除（均为本人数据）
	"POST /api/article/save",
	"POST /api/article/create",
	"PUT /api/article/update",
	"PUT /api/article/restore",
	"DELETE /api/article/remove",
	"DELETE /api/article/delete",

	// 动态：同上（置顶 set_top 仍需要额外授权，不给）
	"POST /api/moments/save",
	"POST /api/moments/create",
	"PUT /api/moments/update",
	"PUT /api/moments/restore",
	"DELETE /api/moments/remove",
	"DELETE /api/moments/delete",
}

// ordinaryAuthRules - 把 ordinaryAuthPermits 换算成权限点 hash 串（逗号分隔）
//
// 通过 createAuthRules() 反查真实规则，避免手写 hash 或路由拼错；
// 只有 type=default / root 的规则才对应「需要权限点」，common / login 直接跳过。
// 计算结果与 EnsureAuthRules 写入的 hash 完全一致（同为 [METHOD]route 的 Sum32）。
func ordinaryAuthRules() string {

	permits := make(map[string]bool, len(ordinaryAuthPermits))
	for _, item := range ordinaryAuthPermits {
		// 只把请求方法转大写，路由本身区分大小写（/api/article/save）
		method, route, exist := strings.Cut(item, " ")
		if !exist {
			continue
		}
		permits[strings.ToUpper(strings.TrimSpace(method))+" "+strings.TrimSpace(route)] = true
	}

	hashes := make([]string, 0, len(ordinaryAuthPermits))
	for _, rule := range createAuthRules() {
		if utils.InArray(rule.Type, []string{"common", "login"}) {
			continue
		}
		method := strings.ToUpper(cast.ToString(rule.Method))
		if !permits[method+" "+rule.Route] {
			continue
		}
		hashes = append(hashes, utils.Hash.Sum32(fmt.Sprintf("[%s]%s", method, rule.Route)))
	}

	return strings.Join(hashes, ",")
}

var builtinAuthGroups = []builtinAuthGroup{
	{
		Id: 1, Name: "超级管理员", Key: "admin", Uids: "|1|",
		Rules: "all", Pages: "all", Root: 1, Default: 1,
		Remark: "超级管理员，拥有所有权限！",
	},
	{
		Id: 2, Name: "普通用户", Key: OrdinaryAuthGroupKey, Uids: "",
		Rules: ordinaryAuthRules(), Pages: "", Root: 0, Default: 1,
		Remark: "注册用户的默认分组：可发布 / 编辑 / 删除自己的文章与动态（只能动自己的数据）",
	},
}

// InitAuthGroup - 初始化AuthGroup表
func InitAuthGroup() {
	// 迁移表
	err := facade.DB.Drive().AutoMigrate(&AuthGroup{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "AuthGroup表迁移失败")
		return
	}
	EnsureAuthGroups()
}

// EnsureAuthGroups - 补齐系统内置分组（幂等，可重复执行）
//
// 为什么需要它：原先只在「表为空」时播种超级管理员，
// 已经装好的库永远拿不到后来新增的内置分组（比如「普通用户」）。
// 这里按 key 判断是否存在，缺哪个补哪个：
//   - **已存在的分组一律不改**（名称 / 权限 / 成员可能已被管理员调整过，覆盖等于改坏数据）；
//   - 内置 ID 被别的分组占用时（老库可能手工建过），改由自增分配，避免主键冲突。
//
// 调用时机：InitAuthGroup（安装 / 迁移）与 timer.Run（每次启动的一次性维护）。
func EnsureAuthGroups() {

	// 这里**不判断 facade.Installed()**：安装向导的 initDB 步骤会直接调 model.InitTable()，
	// 此时 install.lock 还没删除（Installed 为 false），判断的话全新安装就拿不到
	// 超级管理员分组。调用方（InitAuthGroup）已经保证「表已迁移、DB 可用」。
	// 维护任务：任何异常都不应影响服务启动
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "补齐系统内置权限分组时发生panic")
		}
	}()

	for _, builtin := range builtinAuthGroups {
		// WithTrashed：回收站里的同名分组也算「已存在」，不重复播种
		exist, _ := facade.DB.Model(&AuthGroup{}).WithTrashed().Where("key", builtin.Key).Exist()
		if exist {
			continue
		}

		table := AuthGroup{
			Name:    builtin.Name,
			Key:     builtin.Key,
			Uids:    builtin.Uids,
			Rules:   builtin.Rules,
			Pages:   builtin.Pages,
			Root:    builtin.Root,
			Default: builtin.Default,
			Remark:  builtin.Remark,
		}

		if taken, _ := facade.DB.Model(&AuthGroup{}).WithTrashed().Where("id", builtin.Id).Exist(); !taken {
			table.Id = builtin.Id
		}

		if _, err := facade.DB.Model(&AuthGroup{}).Create(&table); err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "key": builtin.Key}, "补齐系统内置权限分组失败")
			continue
		}

		// 内置分组会影响所有人的权限归属，缓存里的用户权限列表需要失效
		facade.Cache.DelTags([]any{"user["})
		facade.Log.Info(map[string]any{"id": table.Id, "key": builtin.Key, "name": builtin.Name}, "已补齐系统内置权限分组")
	}
}

// OrdinaryAuthGroupId - 取「普通用户」分组的 ID（不存在时返回 0）
//
// 用途：注册 / 第三方建号时，若「后台 → 注册设置」没有配置默认权限组，
// 兜底把新用户放进这个空权限组（见 api/controller/comm.go 的 auth()）
func OrdinaryAuthGroupId() int {
	item, err := facade.DB.Model(&AuthGroup{}).Where("key", OrdinaryAuthGroupKey).Find()
	if err != nil || utils.Is.Empty(item) {
		return 0
	}
	return cast.ToInt(item["id"])
}

// AfterSave - 保存后的Hook（包括 create update）
func (this *AuthGroup) AfterSave(tx *gorm.DB) (err error) {

	if !utils.Is.Empty(this.Key) {
		exist, _ := facade.DB.Model(&AuthGroup{}).WithTrashed().Where("id", "!=", this.Id).Where("key", this.Key).Exist()
		if exist {
			return errors.New("key 已存在！")
		}
	}
	return
}

// AfterFind - 查询Hook
func (this *AuthGroup) AfterFind(tx *gorm.DB) (err error) {

	this.Result = this.result()
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// result - 返回结果
func (this *AuthGroup) result() (result map[string]any) {

	var users any
	wg := sync.WaitGroup{}
	wg.Add(1)

	go this.users(&wg, &users)

	wg.Wait()

	return map[string]any{
		"users": users,
	}
}

func (this *AuthGroup) users(wg *sync.WaitGroup, result *any) {

	defer wg.Done()

	tags := utils.ArrayUnique(utils.ArrayEmpty(strings.Split(this.Uids, "|")))
	columns, _ := facade.DB.Model(&[]Users{}).WhereIn("id", tags).Column("id", "nickname", "avatar", "account")

	// Column() 内部走 Scan 到 []map[string]any，不会实例化 Users 结构体，
	// 因此 Users.AfterFind 不会触发，而头像域名模板（{{cos}} 等）正是在那里还原的
	// （存库时 AfterSave 用 DomainTemp2 把真实域名换成了模板）。
	// 这里手动补一次替换，保证与 users/all 等走 Select() 的接口口径一致，
	// 否则接口会把 "{{cos}}/xxx.jpg" 原样返回给前端，头像无法显示。
	*result = ReplaceDomainColumns(columns, "avatar")
}

// Auth 应用权限
func (this *AuthGroup) Auth(uid any, group any, isRemove bool) {

	for _, id := range utils.Unity.Ids(group) {

		item, _ := facade.DB.Model(&AuthGroup{}).WithTrashed().Where("id", id).Find()
		if utils.Is.Empty(item) {
			continue
		}

		uids := utils.Unity.Ids(item["uids"])

		// 移除权限
		if isRemove {
			// 判断 uid 是否在数组中
			if utils.InArray(cast.ToInt(uid), cast.ToIntSlice(uids)) {
				// 原生 Go 语言删除数组元素
				for key, val := range uids {
					if cast.ToInt(val) == cast.ToInt(uid) {
						uids = append(uids[:key], uids[key+1:]...)
					}
				}
			}
		} else {
			uids = append(uids, uid)
		}

		// 去重 - 去空
		uids = utils.Unity.Ids(uids)

		var result string
		if len(uids) > 0 {
			result = fmt.Sprintf("|%v|", strings.Join(cast.ToStringSlice(uids), "|"))
		}
		// 更新数据
		facade.DB.Model(&AuthGroup{}).WithTrashed().Where("id", id).Update(map[string]any{
			"uids": result,
		})
	}
}

// BeforeDelete - 删除前Hook（软删除/物理删除都会触发）
func (this *AuthGroup) BeforeDelete(tx *gorm.DB) (err error) {
	// 禁止删除系统管理员分组（ID=1）
	if this.Id == 1 {
		facade.Log.Warn(map[string]any{"msg": "尝试删除系统管理员分组被拦截", "groupId": this.Id})
		return errors.New("禁止删除系统管理员分组！")
	}
	return
}

// IsRootAdmin 判断用户是否为超级管理员（root 权限组且拥有全部权限 all）
func IsRootAdmin(uid int) bool {
	if uid == 0 {
		return false
	}

	var table []AuthGroup
	facade.DB.Model(&table).Where("root", 1).Like("uids", "%|"+cast.ToString(uid)+"|%").Select()

	for _, item := range table {
		if utils.InArray("all", strings.Split(cast.ToString(item.Rules), ",")) {
			return true
		}
	}

	return false
}
