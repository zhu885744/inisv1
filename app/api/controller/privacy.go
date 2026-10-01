package controller

import (
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// ============================== 用户隐私分级（请求侧） ==============================
//
// 背景：users 表 / user_ban_records 表里存有大量非公开数据 ——
// 账号(account)、邮箱(email)、手机号(phone)、管理员备注(remark)、
// 封禁记录里的操作人 IP/UA(operator_ip/operator_ua)、封禁证据(evidence)、
// 申诉往来(appeal_content/appeal_reply)、最后登录时间(login_time) 等。
//
// 而 GET /api/users/{one,all,column,rand,blackroom} 这类接口在路由规则里是
// type=common（游客可直接访问），历史上只是零星地对 email/phone 做了脱敏，
// 其余字段（remark、封禁字段、result 里内嵌的封禁记录）会被整行原样回显。
//
// 这里统一按「访客 / 登录用户 / 管理员」三档处理，所有对外返回用户数据的接口
// 一律走本文件的入口，避免各接口各写一套脱敏逻辑而出现遗漏。
//
// 分级口径（字段明细见 app/model/privacy.go）：
//   - 管理员（meta.permit）：完整数据（仍不返回 password）；
//   - 本人：自己的账号/邮箱/手机号等可见，但不可见管理侧字段（remark、封禁证据等）；
//   - 访客 / 登录用户查看他人：账号/邮箱/手机号统一隐藏为固定掩码（model.MaskHidden），
//     空值与有值返回同一掩码（不暴露「是否绑定」），也不做「保留前几位」的部分脱敏
//     （部分脱敏仍会泄露长度、号段与邮箱服务商），管理侧字段全部移除。
//
// 注意：脱敏是「原地」进行的（map 是引用类型），调用方无需接收返回值。

// 查看者隐私级别（与 model 层共用同一套枚举）
const (
	// PrivacyGuest 游客（未登录）
	PrivacyGuest = model.PrivacyGuest
	// PrivacyUser 已登录用户（查看他人数据）
	PrivacyUser = model.PrivacyUser
	// PrivacyAdmin 管理员
	PrivacyAdmin = model.PrivacyAdmin
)

// privacyLevel - 当前请求者的隐私级别
func (this meta) privacyLevel(ctx *gin.Context) int {
	if this.permit(ctx) {
		return PrivacyAdmin
	}
	if this.user(ctx).Id > 0 {
		return PrivacyUser
	}
	return PrivacyGuest
}

// privacyUser - 按当前请求者的级别处理「用户数据」（自动识别数据是否属于本人）
func (this meta) privacyUser(ctx *gin.Context, data any) {
	this.privacyUserAs(data, this.privacyLevel(ctx), this.user(ctx).Id)
}

// privacyUserList - 处理「列表」形态的用户数据
//
// 列表结果通常会被写入多用户共享的接口缓存（见 base.cache / users.all），
// 因此这里不区分「本人」——一律按他人规则脱敏，
// 避免某个登录用户的未脱敏数据进入缓存后被其他访客命中（越权泄露）。
func (this meta) privacyUserList(ctx *gin.Context, data any) {
	this.privacyUserAs(data, this.privacyLevel(ctx), 0)
}

// privacyUserAs - 按指定级别脱敏用户数据（原地处理）
//
//	level  : PrivacyGuest / PrivacyUser / PrivacyAdmin
//	selfId : 数据本人 id，>0 时该条数据按「本人」规则处理（列表场景传 0）
func (this meta) privacyUserAs(data any, level int, selfId int) {
	privacyWalk(data, func(item map[string]any) {
		model.SanitizeUser(item, level, selfId)
	})
}

// privacyBanRecord - 按当前请求者的级别处理「封禁记录」数据（单条或列表，原地处理）
func (this meta) privacyBanRecord(ctx *gin.Context, data any) {
	this.privacyBanRecordAs(data, this.privacyLevel(ctx), this.user(ctx).Id)
}

// privacyBanRecordAs - 按指定级别脱敏封禁记录（原地处理）
func (this meta) privacyBanRecordAs(data any, level int, selfId int) {
	// 管理员需要完整记录（后台封禁管理页依赖 operator_ip / evidence / 申诉往来）
	if level == PrivacyAdmin {
		return
	}
	privacyWalk(data, func(item map[string]any) {
		isSelf := selfId > 0 && cast.ToInt(item["uid"]) == selfId
		model.SanitizeBanRecord(item, isSelf)
	})
}

// privacyWalk - 遍历 map / map 列表，对每个 map 就地执行 handle
//
// 兼容 controllers 里常见的几种返回形态：
// map[string]any、[]any、[]map[string]any、gin.H（本质就是 map[string]any）。
// 由于 map 是引用类型，删除字段 / 覆盖字段都会直接反映到原数据上。
func privacyWalk(data any, handle func(item map[string]any)) {
	switch v := data.(type) {
	case map[string]any:
		handle(v)
	case []map[string]any:
		for _, item := range v {
			handle(item)
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				handle(m)
			}
		}
	}
}

// aggregateAllowFields - 非管理员可做聚合（sum/min/max）的字段白名单
//
// 聚合接口能把单列的值原样返回（例如 MAX(email) 会直接吐出某个邮箱明文），
// 因此这里只放行数值型 / 时间型字段，其余（账号、邮箱、手机号、备注、昵称…）一律拒绝。
var aggregateAllowFields = []string{
	"id", "exp", "integral", "status",
	"ban_count", "current_ban_id", "last_ban_at", "restrictions",
	"create_time", "update_time", "login_time", "delete_time",
}

// aggregateFieldForbidden - 判断非管理员是否被禁止对某字段做聚合
func aggregateFieldForbidden(field string) bool {
	for _, allow := range aggregateAllowFields {
		if field == allow {
			return false
		}
	}
	return true
}
