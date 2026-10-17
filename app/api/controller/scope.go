package controller

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 内容可见范围（公开 / 登录可见 / 付费可见）的接口层处理
//
// 约定：
//   - 文章 / 动态 / 独立页面有 scope（1 公开 / 2 登录可见 / 3 付费可见）与 scope_integral（兑换所需积分）两个字段；
//   - **评论不支持付费可见**（见 model.ScopePaySupported）：评论只有 scope（1 公开 / 2 登录可见 / 4 仅作者可见），
//     没有积分字段，历史数据里的 3 一律按「登录可见」处理；
//   - 「仅作者可见」只对评论开放（见 model.ScopeAuthorSupported）：「作者」指**被评论内容的作者**，
//     评论者本人与管理员同样可看，其他登录用户 / 访客只看到占位提示；
//   - 无权查看时**不报错**，而是清空正文 / 图片，并在返回体里补上：
//     scope（归一化后的范围）、scope_integral（所需积分）、scope_state（login 需登录 / pay 需兑换 / author 仅作者可见）；
//   - 作者本人不受限制；管理员仅在**后台管理界面**发起的请求里豁免（见 newScopeGuard），
//     这样在前台浏览时同样能看到真实的付费墙效果；付费内容的解锁记录见 model/content-unlock.go。

// scopeHideFields - 无权查看时需要清空的字段（各内容类型的正文 / 图片）
var scopeHideFields = map[string][]string{
	"article": {"content"},
	"page":    {"content"},
	"moments": {"content", "images"},
	"comment": {"content", "images"},
}

// scopeGuard - 当前请求者的可见范围判定器
type scopeGuard struct {
	uid   int
	admin bool
	// 评论「仅作者可见」需要知道被评论内容的作者 uid；
	// 一次列表请求里同一条内容会命中很多评论，用 map 缓存避免逐条查库
	authors map[string]int
}

// 后台管理界面标记（前端 utils/request 在 /admin 路径下自动附加）
//
// 为什么管理员豁免要额外看这个标记：
//   - article/one 这类接口前台和后台共用，只要用户的权限组里含该接口的权限点，
//     permit 就会通过。若仅凭 permit 豁免，付费墙对一部分登录用户会整体失效；
//   - 管理员在前台浏览时也应该看到真实的「登录可见 / 付费可见」效果，否则无法验证；
//   - 复用 X-Requested-With（已在 CORS 白名单里），避免新增自定义头导致跨域预检失败。
const (
	scopeAdminHeader = "X-Requested-With"
	scopeAdminFlag   = "inis-admin"
)

// newScopeGuard - 构造判定器
//
// 管理员豁免只对「后台管理界面发起的请求」生效，作者本人豁免由 access 里的 owner 判定负责。
//
// 这里判定的是「这个用户能不能进后台」（root 分组，或所在分组配置了后台页面权限），
// 而不是「有没有 article/one 这个接口的权限点」：前台接口（common）本来就不会出现在
// 后台权限树里，用接口权限点判定会让非 root 运营在后台编辑时看不到正文。
func newScopeGuard(ctx *gin.Context) scopeGuard {
	guard := scopeGuard{uid: meta{}.user(ctx).Id, authors: make(map[string]int)}
	if guard.uid > 0 && ctx.GetHeader(scopeAdminHeader) == scopeAdminFlag {
		guard.admin = backendUser(guard.uid)
	}
	return guard
}

// contentAuthor - 取评论所属内容的作者 uid（带请求级缓存）
func (this scopeGuard) contentAuthor(bindType string, bindId int) int {
	if bindId <= 0 || this.authors == nil {
		return 0
	}

	key := bindType + ":" + cast.ToString(bindId)
	if owner, exist := this.authors[key]; exist {
		return owner
	}

	owner := model.ScopeCommentContentOwner(bindType, bindId)
	this.authors[key] = owner
	return owner
}

// backendUser - 是否为后台用户（root 分组，或所在分组配置了后台页面权限）
//
// 只用于「后台管理界面请求」的可见范围豁免，因此不需要按接口权限点细分。
func backendUser(uid int) bool {
	if uid <= 0 {
		return false
	}

	var groups []model.AuthGroup
	facade.DB.Model(&groups).Like("uids", "%|"+cast.ToString(uid)+"|%").Select()

	for _, group := range groups {
		if group.Root == 1 || !utils.Is.Empty(group.Pages) {
			return true
		}
	}

	return false
}

// access - 判定单条内容对当前请求者的可见状态
//
// unlocked 为批量查询出来的已解锁内容集合；传 nil 时按单条查库。
// contentOwner 仅评论的「仅作者可见」需要：被评论内容的作者 uid（不传视为 0）。
func (this scopeGuard) access(owner, scope, bindId int, bindType string, unlocked map[int]bool, contentOwner ...int) string {

	// 管理员 / 作者本人（评论者 / 内容作者都算）：不受可见范围限制
	if this.admin || (this.uid > 0 && owner > 0 && owner == this.uid) {
		return model.ScopeStateAllow
	}

	// 按内容类型归一化：评论不支持付费可见，3 会降级为「登录可见」
	scope = model.ScopeForBind(bindType, scope)

	// 仅作者可见（评论）：除评论者本人外，只有被评论内容的作者能看到
	if scope == model.ScopeAuthor {
		author := 0
		if len(contentOwner) > 0 {
			author = contentOwner[0]
		}
		if this.uid > 0 && author > 0 && author == this.uid {
			return model.ScopeStateAllow
		}
		return model.ScopeStateAuthor
	}

	switch scope {
	case model.ScopeLogin:
		// 登录可见：访客只看到提示
		if this.uid == 0 {
			return model.ScopeStateLogin
		}
	case model.ScopePay:
		// 付费可见：先登录，再用积分兑换
		if this.uid == 0 {
			return model.ScopeStateLogin
		}
		if unlocked == nil {
			unlocked = model.ScopeUnlockedBatch(this.uid, bindType, []int{bindId})
		}
		if !unlocked[bindId] {
			return model.ScopeStatePay
		}
	}

	return model.ScopeStateAllow
}

// redact - 清洗单条内容（unlocked 为批量查询结果，传 nil 时单条查库）
func (this scopeGuard) redact(row map[string]any, bindType string, unlocked map[int]bool) map[string]any {

	if len(row) == 0 {
		return row
	}

	// 评论不支持付费可见：scope=3 按登录可见处理，且不再有兑换积分
	scope := model.ScopeForBind(bindType, cast.ToInt(row["scope"]))
	integral := 0
	if model.ScopePaySupported(bindType) {
		integral = cast.ToInt(row["scope_integral"])
	}
	bindId := cast.ToInt(row["id"])
	owner := cast.ToInt(row["uid"])

	// 仅作者可见：作者是「被评论内容的作者」，不是评论者本人，
	// 需要按评论的 bind_type / bind_id 反查（结果按内容缓存，不逐条查库）
	contentAuthor := 0
	if scope == model.ScopeAuthor {
		contentAuthor = this.contentAuthor(cast.ToString(row["bind_type"]), cast.ToInt(row["bind_id"]))
	}

	state := this.access(owner, scope, bindId, bindType, unlocked, contentAuthor)

	// 可查看：原样返回（不做任何写入，缓存协程可能正在序列化同一份数据）
	if state == model.ScopeStateAllow {
		return row
	}

	// 无权查看：复制一份再脱敏，避免影响缓存里的原始数据
	result := make(map[string]any, len(row)+3)
	for key, value := range row {
		result[key] = value
	}

	// 范围信息一并回传，前端据此渲染「登录后查看 / N 积分兑换」引导
	result["scope"] = scope
	result["scope_state"] = state
	if model.ScopePaySupported(bindType) {
		result["scope_integral"] = integral
	} else {
		// 评论没有兑换积分：历史列即便被 SELECT * 带出来也一并去掉
		delete(result, "scope_integral")
	}

	// 清空正文 / 图片，只保留标题、摘要、作者等占位信息
	for _, field := range scopeHideFields[bindType] {
		if _, exist := result[field]; exist {
			result[field] = ""
		}
	}

	// 评论的回复树（flat 接口）逐层处理
	if replies, exist := result["replies"]; exist {
		switch items := replies.(type) {
		case []map[string]any:
			children := make([]map[string]any, len(items))
			for index, item := range items {
				children[index] = this.redact(item, bindType, unlocked)
			}
			result["replies"] = children
		case []any:
			children := make([]any, len(items))
			for index, item := range items {
				if sub, ok := item.(map[string]any); ok {
					children[index] = this.redact(sub, bindType, unlocked)
				} else {
					children[index] = item
				}
			}
			result["replies"] = children
		}
	}

	return result
}

// redactList - 批量清洗（一次查出已解锁内容，避免逐条查库）
func (this scopeGuard) redactList(rows []map[string]any, bindType string) []map[string]any {

	if len(rows) == 0 {
		return rows
	}

	ids := make([]int, 0, len(rows))
	for _, row := range rows {
		if model.ScopeNormalize(cast.ToInt(row["scope"])) != model.ScopePay {
			continue
		}
		if id := cast.ToInt(row["id"]); id > 0 {
			ids = append(ids, id)
		}
	}

	unlocked := model.ScopeUnlockedBatch(this.uid, bindType, ids)

	// 逐条脱敏：只有无权查看的条目会被替换为副本，可查看的条目保持原引用（缓存安全）
	result := make([]map[string]any, len(rows))
	for index, row := range rows {
		result[index] = this.redact(row, bindType, unlocked)
	}

	return result
}

// applyScopeData - 统一入口：按可见范围处理接口返回的数据（单条 / 列表 / 混合切片）
//
// 返回处理后的数据，调用方需要接收返回值（列表可能被替换为新切片）。
// 可查看的条目原样返回，只有被限制的条目会生成副本，因此不会污染接口层的缓存数据。
func applyScopeData(ctx *gin.Context, data any, bindType string) any {

	if data == nil || !model.ScopeBindTypeValid(bindType) {
		return data
	}

	guard := newScopeGuard(ctx)

	switch items := data.(type) {
	case map[string]any:
		return guard.redact(items, bindType, nil)
	case []map[string]any:
		return guard.redactList(items, bindType)
	case []any:
		result := make([]any, len(items))
		for index, item := range items {
			switch value := item.(type) {
			case map[string]any:
				result[index] = guard.redact(value, bindType, nil)
			default:
				result[index] = item
			}
		}
		return result
	}

	return data
}
