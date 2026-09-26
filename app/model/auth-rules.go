package model

import (
	"fmt"
	"inis/app/facade"
	"net/url"
	"strings"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

type AuthRules struct {
	Id     int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name   string `gorm:"comment:规则名称;" json:"name"`
	Method string `gorm:"comment:请求类型; default:'GET';" json:"method"`
	Route  string `gorm:"comment:路由;" json:"route"`
	Type   string `gorm:"default:'default'; comment:规则类型;" json:"type"`
	Hash   string `gorm:"comment:哈希值;" json:"hash"`
	Cost   int    `gorm:"type:int(32); comment:费用; default:1;" json:"cost"`
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
func (this *AuthRules) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

// InitAuthRules - 初始化AuthRules表
func InitAuthRules() {
	facade.Log.Info(map[string]any{}, "==== InitAuthRules 开始执行 ====")

	err := facade.DB.Drive().AutoMigrate(&AuthRules{})
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "AuthRules表迁移失败")
		return
	}
	facade.Log.Info(map[string]any{}, "AuthRules AutoMigrate执行完成")

	EnsureAuthRules()

	facade.Log.Info(map[string]any{}, "==== InitAuthRules 全部执行完毕 ====")
}

// EnsureAuthRules - 补齐缺失的权限规则（幂等，可重复执行）
//
// 为什么需要它：中间件（app/api/middleware/rule.go）对「查不到规则」的接口会走默认分支
// （需要权限点），匿名请求会直接 401。而 saveAuthRules 只在 hash 不存在时插入，
// 因此新增接口的规则在**已安装**的库里不会自动出现——启动时补录一次即可解决。
//
// 同时顺带纠正历史数据里非法的规则类型（type=root → default）。
func EnsureAuthRules() {

	list := createAuthRules()
	facade.Log.Info(map[string]any{"count": len(list)}, "createAuthRules生成规则数量")

	// 一次性取出现有 hash（含回收站，避免把已删除的规则又补回来）
	rows, _ := facade.DB.Model(&AuthRules{}).WithTrashed().Column("hash")
	exist := make(map[string]bool)
	for _, hash := range cast.ToStringSlice(rows) {
		exist[hash] = true
	}

	created := 0
	for _, item := range list {
		method := strings.ToUpper(cast.ToString(item.Method))
		hash := utils.Hash.Sum32(fmt.Sprintf("[%s]%s", method, item.Route))
		if exist[hash] {
			continue
		}
		saveAuthRules(item)
		created++
	}

	if created > 0 {
		facade.Log.Info(map[string]any{"created": created}, "已补齐缺失的权限规则")
	}

	NormalizeAuthRuleTypes()
}

// NormalizeAuthRuleTypes - 纠正历史数据中非法的规则类型
//
// 早期初始化数据用过 type=root（exp/give、积分卡密、商品管理、动态置顶等），
// 但中间件只识别 common（免登录放行）/ login（登录即放行）/ 其余（需权限点）三类
// （见 app/api/middleware/rule.go），root 的运行时行为与 default 完全一致，
// 属于错误的类型标注：会导致后台「默认」统计与筛选漏掉这些规则。
// 这里统一纠正为 default，保持与种子数据（createAuthRules）口径一致。
//
// 调用时机：InitAuthRules（安装/迁移）与 timer.Run（每次启动的一次性维护任务），
// 保证「播种时已修正」的规则之外，老库在启动时也能自动纠正。
func NormalizeAuthRuleTypes() {

	// 维护任务：任何异常都不应影响服务启动
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "纠正权限规则类型时发生panic")
		}
	}()

	table := AuthRules{}
	// WithTrashed：回收站里的规则也一并纠正，避免恢复后类型又是旧值
	tx, err := facade.DB.Model(&table).WithTrashed().
		Where("type", "NOT IN", []string{"common", "login", "default"}).
		Update(map[string]any{"type": "default"})

	if err != nil {
		facade.Log.Warn(map[string]any{"error": err.Error()}, "修正权限规则类型失败")
		return
	}

	if tx == nil || tx.RowsAffected <= 0 {
		return
	}

	facade.Log.Info(map[string]any{"rows": tx.RowsAffected}, "已把非标准的权限规则类型修正为 default")

	// 规则缓存名为 rule[METHOD][path] 且无过期时间（见 middleware/rule.go 的 cacheRulePrefix），
	// 标签是模糊匹配，这里顺带清掉 auth-rules 列表缓存，避免后台仍显示旧的 root 值。
	facade.Cache.DelTags([]any{"rule"})
}

// createAuthRules - 生成规则
func createAuthRules() (result []AuthRules) {

	batch := map[string]map[string][]string{
		"proxy": {
			"GET":    {"path=&name=代理 GET 请求&type=login"},
			"PUT":    {"path=&name=代理 PUT 请求&type=login"},
			"POST":   {"path=&name=代理 POST 请求&type=login"},
			"PATCH":  {"path=&name=代理 PATCH 请求&type=login"},
			"DELETE": {"path=&name=代理 DELETE 请求&type=login"},
		},
		"comm": {
			"POST": {
				"path=login&name=传统和加密登录&type=common",
				"path=register&name=注册账户&type=common",
				"path=check-token&name=校验登录&type=common",
				"path=reset-password&name=重置密码&type=common",
				"path=logout&name=退出登录&type=common",
				"path=verify-email&name=验证注册邮箱&type=common",
				"path=send-verify-mail&name=重发注册验证邮件&type=common",
			},
			"DELETE": {"path=logout&name=退出登录&type=common"},
		},
		"toml": {
			"GET": {
				"path=sms&name=获取SMS服务配置",
				"path=cache&name=获取缓存服务配置",
				"path=crypt&name=获取加密服务配置",
				"path=log&name=获取日志服务配置",
				"path=storage&name=获取存储服务配置",
				"path=notification&name=获取通知配置",
			},
			"PUT": {
				"path=sms&name=修改SMS服务配置",
				"path=sms-email&name=修改邮件服务配置",
				"path=sms-email-queue&name=修改邮件发件队列配置",
				"path=sms-aliyun&name=修改阿里云短信服务配置",
				"path=sms-aliyun-number-verify&name=修改阿里云号码验证配置",
				"path=sms-tencent&name=修改腾讯云短信服务配置",
				"path=crypt-jwt&name=修改JWT配置",
				"path=cache-redis&name=修改Redis缓存配置",
				"path=cache-file&name=修改文件缓存配置",
				"path=cache-ram&name=修改内存缓存配置",
				"path=sms-drive&name=修改SMS驱动配置",
				"path=cache-default&name=修改缓存默认服务类型",
				"path=storage-default&name=修改存储默认服务类型",
				"path=storage-local&name=修改本地存储配置",
				"path=storage-cos&name=修改COS存储配置",
				"path=storage-attachment&name=修改附件配置",
				"path=notification&name=修改通知配置",
			},
			"POST": {
				"path=test-sms-email&name=发送测试邮件",
				"path=test-sms-aliyun&name=发送阿里云测试短信",
				"path=test-sms-aliyun-number-verify&name=发送阿里云号码验证服务测试短信",
				"path=test-sms-tencent&name=发送腾讯云测试短信",
				"path=test-redis&name=测试Redis连接",
				"path=test-cos&name=测试COS连接",
			},
		},
		"tags": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"users": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=blackroom&type=common&name=小黑屋公示",
			},
			"PUT": {
				"restore",
				"path=update&type=login",
				"path=email&type=login&name=修改邮箱",
				"path=phone&type=login&name=修改手机号",
				"path=status&type=login&name=修改用户状态",
				"path=ban&name=封禁用户",
				"path=unban&name=解封用户",
				"path=appeal-handle&name=处理申诉",
			},
			"POST": {
				"create", "save",
				"path=appeal&type=login&name=用户申诉",
				"path=appeal-public&type=common&name=封禁用户公开申诉",
			},
			"DELETE": {"remove", "delete", "clear", "path=destroy&type=login&name=注销账户"},
		},
		"links": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT": {"path=update&type=login", "path=restore&type=login"},
			"POST": {
				"path=save&type=login",
				"path=create&type=login",
			},
			"DELETE": {
				"path=remove&type=login",
				"path=delete&type=login",
				"path=clear&type=login",
			},
		},
		"pages": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"level": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"banner": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"config": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"article": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"placard": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"comment": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=flat&type=common&name=扁平化",
			},
			"PUT":    {"path=update&type=login", "path=restore&type=login"},
			"POST":   {"path=save&type=login", "path=create&type=login"},
			"DELETE": {"path=remove&type=login", "path=delete&type=login", "path=clear&type=login"},
		},
		"api-keys": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"auth-group": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore", "path=uids&name=更改用户权限"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"auth-rules": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"auth-pages": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"links-group": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"article-group": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=tree&type=common&name=树形结构",
			},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"exp": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=active&type=common&name=活跃度排行",
				"path=rules&type=common&name=经验任务规则",
				"path=check-in-status&type=login&name=签到状态",
				"path=check-in-rank&type=common&name=签到排行",
				"path=check-in-calendar&type=login&name=签到日历",
			},
			"PUT": {"update", "restore"},
			"POST": {
				"save",
				"create",
				"path=check-in&type=login&name=每日签到",
				"path=give&type=default&name=发放经验值",
				"path=share&type=login&name=分享",
			},
			"DELETE": {"remove", "delete", "clear"},
		},
		"integral": {
			"GET": {
				"path=status&type=login&name=积分余额",
				"path=all&type=login&name=积分流水",
				"path=rules&type=common&name=积分任务规则",
				"path=tasks&type=login&name=今日任务进度",
				"path=rank&type=common&name=积分排行榜",
				"path=card-all&type=default&name=卡密列表",
				"path=card-stats&type=default&name=卡密统计",
				"path=card-export&type=default&name=导出未使用卡密",
			},
			"POST": {
				"path=give&type=default&name=调整积分",
				"path=card-generate&type=default&name=生成卡密",
				"path=card-redeem&type=login&name=卡密兑换积分",
			},
			"DELETE": {
				"path=card-remove&type=default&name=删除卡密",
				"path=card-delete&type=default&name=彻底删除卡密",
			},
		},
		"goods": {
			"GET": {
				"path=one&type=common&name=商品详情",
				"path=all&type=common&name=商品列表",
				"path=categories&type=common&name=商品分类",
				"path=orders&type=login&name=我的订单",
				"path=order-one&type=login&name=订单详情",
				"path=my-stats&type=login&name=我的兑换统计",
				"path=orders-all&type=default&name=全部订单",
				"path=stats&type=default&name=商城统计",
				"path=count&type=common&name=商品数量",
			},
			"PUT": {
				"path=update&type=default&name=更新商品",
				"path=restore&type=default&name=恢复商品",
				"path=order-status&type=default&name=更新订单状态",
				"path=cancel-order&type=login&name=取消订单",
				"path=receive&type=login&name=确认收货",
			},
			"POST": {
				"path=buy&type=login&name=购买商品",
				"path=save&type=default&name=保存商品",
				"path=create&type=default&name=创建商品",
			},
			"DELETE": {
				"path=remove&type=default&name=删除商品",
				"path=delete&type=default&name=彻底删除商品",
				"path=clear&type=default&name=清空回收站",
			},
		},
		"qps-warn": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"ip-black": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"ip-white": {
			"GET":    {"one", "all", "sum", "min", "max", "count", "column", "rand"},
			"PUT":    {"update", "restore"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"search": {
			"GET": {
				"path=article&type=common&name=文章搜索",
				"path=pages&type=common&name=独立页面搜索",
				"path=tags&type=common&name=标签搜索",
				"path=users&type=common&name=用户搜索",
				"path=links&type=common&name=友链搜索",
				"path=moments&type=common&name=动态搜索",
				"path=all&type=common&name=全局搜索",
			},
		},
		"rss": {
			"GET": {
				"path=&type=common&name=RSS订阅源",
			},
		},
		"moments": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=comment&type=common&name=获取动态评论",
				"path=comment_count&type=common&name=获取动态评论数量",
			},
			"PUT":    {"update", "restore", "path=set_top&type=default&name=设置/取消置顶动态"},
			"POST":   {"save", "create"},
			"DELETE": {"remove", "delete", "clear"},
		},
		"attachment": {
			"GET": {
				"path=one&type=common&name=获取指定附件",
				"path=all&type=common&name=获取附件列表",
				"path=sum&type=common&name=求和",
				"path=min&type=common&name=最小值",
				"path=max&type=common&name=最大值",
				"path=rand&type=common&name=随机获取",
				"path=count&type=common&name=查询数量",
				"path=column&type=common&name=列查询",
				"path=list&type=login&name=获取我的附件",
				"path=emoji&type=common&name=获取表情列表",
			},
			"POST": {
				"path=save&type=login&name=保存数据",
				"path=create&type=login&name=添加数据",
				"path=upload&type=login&name=上传附件",
				"path=batch&type=login&name=批量上传附件",
				"path=checkType&type=login&name=检查文件类型",
			},
			"PUT": {
				"path=update&type=login&name=更新数据",
				"path=restore&type=login&name=恢复数据",
				"path=bind&type=login&name=绑定业务类型",
			},
			"DELETE": {
				"path=remove&type=login&name=软删除",
				"path=delete&type=login&name=彻底删除",
				"path=clear&type=login&name=清空回收站",
			},
		},
		"user-collects": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=is-collected&type=login&name=检查是否已收藏",
				"path=collects&type=common&name=获取收藏列表",
				"path=counts&type=common&name=批量查询收藏数量",
			},
			"POST": {
				"path=save&type=login",
				"path=create&type=login",
				"path=collect&type=login&name=收藏",
			},
			"PUT": {
				"path=update&type=login",
				"path=restore&type=login",
				"path=uncollect&type=login&name=取消收藏",
			},
			"DELETE": {
				"path=remove&type=login",
				"path=delete&type=login",
				"path=clear&type=login",
			},
		},
		"user-follows": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=following&type=common&name=获取关注列表",
				"path=followers&type=common&name=获取粉丝列表",
				"path=is-following&type=login&name=检查是否已关注",
				"path=counts&type=common&name=批量查询关注/粉丝数量",
			},
			"POST": {
				"path=save&type=login",
				"path=create&type=login",
				"path=follow&type=login&name=关注用户",
			},
			"PUT": {
				"path=update&type=login",
				"path=restore&type=login",
				"path=unfollow&type=login&name=取消关注",
			},
			"DELETE": {
				"path=remove&type=login",
				"path=delete&type=login",
				"path=clear&type=login",
			},
		},
		"user-likes": {
			"GET": {
				"path=one&type=common",
				"path=all&type=common",
				"path=sum&type=common",
				"path=min&type=common",
				"path=max&type=common",
				"path=rand&type=common",
				"path=count&type=common",
				"path=column&type=common",
				"path=is-liked&type=login&name=检查是否已点赞",
				"path=likes&type=common&name=获取点赞列表",
				"path=counts&type=common&name=批量查询点赞数量",
			},
			"POST": {
				"path=save&type=login",
				"path=create&type=login",
				"path=like&type=login&name=点赞",
			},
			"PUT": {
				"path=update&type=login",
				"path=restore&type=login",
				"path=unlike&type=login&name=取消点赞",
			},
			"DELETE": {
				"path=remove&type=login",
				"path=delete&type=login",
				"path=clear&type=login",
			},
		},
		"notification": {
			"GET": {
				"path=one&type=login&name=获取指定通知",
				"path=all&type=login&name=获取全部通知",
				"path=sum&type=login",
				"path=min&type=login",
				"path=max&type=login",
				"path=rand&type=login",
				"path=count&type=login&name=查询通知数量",
				"path=column&type=login&name=列查询通知",
				"path=list&type=login&name=获取通知列表",
				"path=unread-count&type=login&name=获取未读通知数",
			},
			"POST": {
				"path=save&type=login",
				"path=create&type=login",
				"path=send-system&type=default&name=发送系统消息",
			},
			"PUT": {
				"path=update&type=login",
				"path=restore&type=login",
				"path=read&type=login&name=标记已读",
				"path=read-all&type=login&name=全部标记已读",
				"path=read-batch&type=login&name=批量标记已读",
			},
			"DELETE": {
				"path=remove&type=login&name=删除通知",
				"path=delete&type=login",
				"path=clear&type=login",
				"path=remove-all&type=login&name=清空通知",
			},
		},
	}

	// 接口名称
	names := map[string]string{
		"exp":           "【经验值 API】",
		"proxy":         "【代理 API】",
		"user-follows":  "【用户关注 API】",
		"user-likes":    "【用户点赞 API】",
		"user-collects": "【用户收藏 API】",
		"comm":          "【公共 API】",
		"tags":          "【标签 API】",
		"level":         "【等级 API】",
		"pages":         "【独立页面 API】",
		"users":         "【用户 API】",
		"links":         "【友链 API】",
		"banner":        "【轮播 API】",
		"article":       "【文章 API】",
		"comment":       "【评论 API】",
		"placard":       "【公告 API】",
		"config":        "【配置 API】",
		"toml":          "【服务配置 API】",
		"ip-black":      "【IP黑名单 API】",
		"ip-white":      "【IP白名单 API】",
		"qps-warn":      "【QPS预警 API】",
		"api-keys":      "【接口密钥 API】",
		"auth-group":    "【权限分组 API】",
		"auth-pages":    "【页面权限 API】",
		"auth-rules":    "【权限规则 API】",
		"links-group":   "【友链分类 API】",
		"article-group": "【文章分类 API】",
		"search":        "【搜索 API】",
		"rss":           "【RSS订阅 API】",
		"moments":       "【动态 API】",
		"attachment":    "【附件 API】",
		"notification":  "【消息通知 API】",
		"integral":      "【积分 API】",
		"goods":         "【商品 API】",
	}

	// 基础方法
	methods := map[string]map[string]string{
		"GET": {
			"one":    "获取指定",
			"all":    "获取全部",
			"sum":    "求和",
			"min":    "最小值",
			"max":    "最大值",
			"rand":   "随机获取",
			"count":  "查询数量",
			"column": "列查询",
		},
		"POST": {
			"save":   "保存数据（推荐）",
			"create": "添加数据",
		},
		"PUT": {
			"update":  "更新数据",
			"restore": "恢复数据",
		},
		"DELETE": {
			"remove": "软删除（回收站）",
			"delete": "彻底删除",
			"clear":  "清空回收站",
		},
	}

	// 批量生成公共接口
	for key, value := range batch {
		for method, items := range value {
			for _, item := range items {

				param := map[string]string{
					"type": "default",
				}

				// 检查 item 是否包含 = 号
				if !strings.Contains(item, "=") {

					param["path"] = item

				} else {

					// 解析 "name=代理 GET 请求&path=&type=common"
					values, _ := url.ParseQuery(item)

					for name, text := range values {
						if len(text) == 1 {
							param[name] = text[0]
						} else {
							param[name] = cast.ToString(text)
						}
					}
				}

				result = append(result, AuthRules{
					Type:   param["type"],
					Method: strings.ToUpper(method),
					Route:  "/api/" + key + utils.Ternary[string](utils.Is.Empty(param["path"]), "", "/"+param["path"]),
					Name:   names[key] + utils.Default(param["name"], methods[method][param["path"]]),
					Remark: param["remark"],
				})
			}
		}
	}
	return
}

// saveAuthRules 保存权限规则
func saveAuthRules(item AuthRules) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{
				"error": err,
				"route": item.Route,
				"name":  item.Name,
			}, "保存权限规则时发生panic")
		}
	}()

	method := strings.ToUpper(cast.ToString(item.Method))
	hash := utils.Hash.Sum32(fmt.Sprintf("[%s]%s", method, item.Route))

	table := AuthRules{
		Hash:   hash,
		Type:   item.Type,
		Remark: item.Remark,
		Name:   cast.ToString(item.Name),
		Method: cast.ToString(item.Method),
		Route:  cast.ToString(item.Route),
	}

	exist, err := facade.DB.Model(&AuthRules{}).Where("hash", hash).Exist()
	if err != nil {
		// 查询异常，仅告警，不return，继续尝试写入
		facade.Log.Warn(map[string]any{"error": err.Error(), "route": item.Route, "hash": hash}, "检查hash存在性查询异常，直接尝试写入")
	}

	// 只有查询无错误并且确认存在，才跳过
	if err == nil && exist {
		return
	}

	_, err = facade.DB.Model(&AuthRules{}).Create(&table)
	if err != nil {
		// 数据库唯一索引冲突直接忽略
		if strings.Contains(err.Error(), "Duplicate entry") {
			return
		}
		facade.Log.Error(map[string]any{"error": err.Error(), "route": item.Route, "hash": hash}, "自动添加规则失败")
	}
}
