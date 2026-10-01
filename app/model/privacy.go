package model

import (
	"regexp"

	"github.com/spf13/cast"
)

// ============================== 用户隐私分级（数据层） ==============================
//
// users 表 / user_ban_records 表里存有大量非公开数据：
// 账号(account)、邮箱(email)、手机号(phone)、管理员备注(remark)、
// 封禁记录里的操作人 IP/UA(operator_ip/operator_ua)、封禁证据(evidence)、
// 申诉往来(appeal_content/appeal_reply)、最后登录时间(login_time) 等。
//
// 本文件提供与请求无关的「数据清洗」函数；带权限判断的入口在
// app/api/controller/privacy.go，两者共用这里的字段口径。
//
// 分级口径：
//   - PrivacyAdmin：完整数据（仍不返回 password）；
//   - 本人（selfId 命中）：自己的账号/邮箱/手机号可见，但不可见管理侧字段；
//   - 访客 / 登录用户查看他人：账号/邮箱/手机号一律替换为固定掩码 MaskHidden
//     （不做「保留前几位」的部分脱敏），管理侧字段全部移除。
const (
	// PrivacyGuest 游客（未登录）
	PrivacyGuest = 0
	// PrivacyUser 已登录用户（查看他人数据）
	PrivacyUser = 1
	// PrivacyAdmin 管理员
	PrivacyAdmin = 2
)

// MaskHidden - 「他人」视图中账号 / 邮箱 / 手机号的统一占位值
//
// 为什么不再做部分脱敏：182****5257 仍会泄露号段与长度，
// 19********@qq.com 仍会泄露邮箱长度、前缀字符与邮箱服务商（@qq.com），
// 与昵称、注册时间等信息结合后足以拼出可用于撞库 / 社工的线索。
// 非本人查看时统一返回本固定值：原始内容、长度、格式、服务商均不泄露。
const MaskHidden = "********"

// HideContact - 账号 / 邮箱 / 手机号统一隐藏
//
// 无差别掩码：无论原始值是空、NULL 还是有值，一律返回 MaskHidden。
// 这样「未绑定」与「已绑定但无权查看」在他人视图中不可区分 —— 否则访客
// 只要看到空值，就能筛出「未绑定邮箱/手机」的账号，这类账号多为注册后
// 未激活的僵尸号，可被批量定位用于撞库或定向社工。
// 代价仅是前端在他人视图里无法再判断对端「是否绑定」（该类判断本身也属于
// 信息泄露）；本人视图与管理员视图完全不受影响。
func HideContact(string) string {
	return MaskHidden
}

// userPrivateFields - 非管理员查看「他人」时，必须从用户记录中移除的字段
//
//	remark                    管理员给该用户写的备注（任何时候都不应回显给用户）
//	text                      通用文本存储字段，可能含内部数据
//	source                    注册来源（内部数据）
//	ban_count/current_ban_id/last_ban_at/restrictions  封禁管理侧字段
//	login_time                最后登录时间（行为数据）
var userPrivateFields = []string{
	"password",
	"remark",
	"text",
	"source",
	"ban_count",
	"current_ban_id",
	"last_ban_at",
	"restrictions",
	"login_time",
	"delete_time",
}

// userSelfPrivateFields - 「本人」查看时仍需移除的字段（管理侧数据，本人也无需知道）
var userSelfPrivateFields = []string{
	"password",
	"remark",
	"delete_time",
}

// hideContactFields - 非本人查看时需要统一隐藏的联系方式字段
var hideContactFields = []string{"account", "email", "phone"}

// banRecordPrivateFields - 封禁记录里的管理侧/私密数据，非管理员一律移除
//
//	operator_id/operator_ip/operator_ua  操作该封禁的管理员身份与网络指纹
//	evidence                             封禁证据（内部留存）
//	appeal_content                       被封禁用户的申诉原文
//	delete_content/ban_appeal/json/text  管理项与内部存储
var banRecordPrivateFields = []string{
	"operator_id",
	"operator_ip",
	"operator_ua",
	"evidence",
	"appeal_content",
	"delete_content",
	"ban_appeal",
	"json",
	"text",
	"delete_time",
}

// banRecordSelfExtraFields - 「本人」可见、但「他人」不可见的申诉往来字段
var banRecordSelfExtraFields = []string{
	"appeal_reply",
	"appeal_reply_time",
}

// MaskNickname - 昵称脱敏：保留首尾字符
func MaskNickname(nickname string) string {
	runes := []rune(nickname)
	switch {
	case len(runes) > 2:
		return string(runes[0]) + "***" + string(runes[len(runes)-1])
	case len(runes) > 1:
		return string(runes[0]) + "*"
	case len(runes) == 1:
		// 单字昵称多为真实姓氏，保留即等于不脱敏
		return "*"
	}
	return nickname
}

// hideContact - 把 map 里的账号 / 邮箱 / 手机号统一替换为固定掩码
//
// 只区分「字段是否存在」：字段存在即写入掩码（空值 / NULL 也不例外），
// 字段不存在则跳过。users 表的这些列都会被 Select 出来，因此他人视图下
// 这几个字段恒为 MaskHidden，不会泄露「是否绑定」。
func hideContact(item map[string]any) {
	if item == nil {
		return
	}

	for _, key := range hideContactFields {
		if _, ok := item[key]; !ok {
			continue
		}
		item[key] = MaskHidden
	}
}

// derivedAvatarPattern - 由邮箱自动推导出来的头像地址
//
// users.AfterFind 在用户没设置头像、且邮箱是 QQ 邮箱时，会把头像填成
// https://q1.qlogo.cn/g?b=qq&nk=<QQ号>&s=100 —— URL 里的 nk 参数就是 QQ 号，
// 而 <QQ号>@qq.com 正是该用户的登录邮箱。邮箱在他人视图中已经统一隐藏为掩码，
// 头像却把同一个 QQ 号原样带出去，属于「绕过脱敏的关联性泄露」。
var derivedAvatarPattern = regexp.MustCompile(`(?i)^https?://[^/]*qlogo\.cn/[^?\s]*\?[^\s]*\bnk=\d+`)

// hideDerivedAvatar - 他人视图中把「由邮箱推导出来的头像」置空（等同未设置头像）
//
// 仅处理模型层自动推导的 qlogo 头像：用户自己填写/上传的头像原样保留。
func hideDerivedAvatar(item map[string]any) {
	if item == nil {
		return
	}

	avatar, ok := item["avatar"]
	if !ok {
		return
	}
	if derivedAvatarPattern.MatchString(cast.ToString(avatar)) {
		item["avatar"] = ""
	}
}

// SanitizeUser - 单条用户记录的脱敏规则（原地处理）
//
//	level  : PrivacyGuest / PrivacyUser / PrivacyAdmin
//	selfId : 数据本人 id，>0 时该条数据按「本人」规则处理（列表场景传 0）
func SanitizeUser(item map[string]any, level int, selfId int) {
	if item == nil {
		return
	}

	// 管理员：仅保证密码不外泄
	if level == PrivacyAdmin {
		delete(item, "password")
		return
	}

	isSelf := selfId > 0 && cast.ToInt(item["id"]) == selfId

	if isSelf {
		for _, key := range userSelfPrivateFields {
			delete(item, key)
		}
	} else {
		for _, key := range userPrivateFields {
			delete(item, key)
		}
		// 他人数据：账号 / 邮箱 / 手机号统一隐藏为固定掩码
		// （保留原字段结构，前端无需改判断；不做「保留前几位」的部分脱敏）
		hideContact(item)
	}

	SanitizeUserResult(item, isSelf)
}

// SanitizeUserResult - 处理 users.result（AfterFind 注入的解析结果）
//
//	result.auth 是用户的权限组/规则（授权信息，他人不可见）
//	result.ban.record 是整条封禁记录，内含管理员的 IP/UA 与封禁证据
func SanitizeUserResult(item map[string]any, isSelf bool) {
	if item == nil {
		return
	}

	// 关联性泄露：头像若由 QQ 邮箱推导而来，URL 里的 QQ 号等价于邮箱
	if !isSelf {
		hideDerivedAvatar(item)
	}

	result, ok := item["result"].(map[string]any)
	if !ok || result == nil {
		return
	}

	// 权限组：仅本人可见
	if !isSelf {
		delete(result, "auth")
	}

	ban, ok := result["ban"].(map[string]any)
	if !ok || ban == nil {
		return
	}

	if !isSelf {
		delete(ban, "ban_count")
		delete(ban, "restrictions")
	}

	if record, ok := ban["record"].(map[string]any); ok && record != nil {
		SanitizeBanRecord(record, isSelf)
	}
}

// SanitizeBanRecord - 单条封禁记录的脱敏规则（原地处理）
func SanitizeBanRecord(record map[string]any, isSelf bool) {
	if record == nil {
		return
	}

	for _, key := range banRecordPrivateFields {
		delete(record, key)
	}

	if !isSelf {
		for _, key := range banRecordSelfExtraFields {
			delete(record, key)
		}
	}

	// result.operator 是执行封禁的管理员（昵称/头像），不对非管理员暴露
	if result, ok := record["result"].(map[string]any); ok && result != nil {
		delete(result, "operator")
		// result.user 是被封禁用户的基础信息，账号 / 邮箱 / 手机号统一隐藏
		if user, ok := result["user"].(map[string]any); ok && user != nil {
			hideContact(user)
			if !isSelf {
				// 公示墙（blackroom）会把被封禁用户展示给所有人：昵称在中文语境下
				// 常含真实姓名，按「他人」规则一并脱敏（此前依赖控制器里的类型断言，
				// 一旦 ORM 返回结构变化就会静默跳过，导致完整昵称被公示）
				if nickname, ok := user["nickname"]; ok {
					user["nickname"] = MaskNickname(cast.ToString(nickname))
				}
				// 头像可能是由 QQ 邮箱推导出来的，同样会带出 QQ 号
				hideDerivedAvatar(user)
			}
		}
	}
}

// SanitizeAuthor - 处理「内嵌在公开内容里的作者信息」（原地处理）
//
// article / comment / moments / exp / user-follows 等公开接口会把作者以
// {id,nickname,avatar,description,json,result,title} 的形式内嵌进响应里（游客可直接访问），
// 其中 result.auth 是该用户的权限组/规则，result.ban.record 里含管理员的操作人 IP/UA
// （operator_ip/operator_ua）、封禁证据（evidence）、申诉原文（appeal_content）。
// 这些字段不应随公开内容一起对外提供，因此统一按「他人」规则清洗。
//
// 兼容 map[string]any、[]any、[]map[string]any 三种形态。
func SanitizeAuthor(data any) {
	switch v := data.(type) {
	case map[string]any:
		SanitizeUserResult(v, false)
	case []map[string]any:
		for _, item := range v {
			SanitizeUserResult(item, false)
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				SanitizeUserResult(m, false)
			}
		}
	}
}
