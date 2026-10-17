package model

import (
	"fmt"
	"regexp"
	"strings"

	"inis/app/facade"

	"github.com/spf13/cast"
)

/**
 * 用户 @ 提及（文章 / 动态 / 独立页 / 评论 共用）
 *
 * 前端约定：输入 @ 时弹出「我关注的人」候选列表（GET /api/mention/candidates），
 * 选中后往编辑器里插入纯文本 @昵称；保存时由本文件做两件事：
 *
 *  1. 把 @昵称 规范化为 markdown 链接 @[昵称](/author/uid)：
 *     文章 / 独立页由 markdown-it 原生渲染成链接，评论 / 动态由
 *     Mellow/src/utils/emoji.js 的 renderEmojiWithBreaks 按同一格式转成 <a>，
 *     四处表现一致（显示 @昵称、点击跳作者主页）；
 *  2. 返回被 @ 到的用户列表，由控制器发站内通知（type=mention）。
 *
 * 核心限制：**只匹配当前作者已关注的人**。未关注的人一律当普通文本处理，
 * 既不转链接也不发通知；前端候选列表同样只给已关注的人，两边口径一致。
 *
 * 幂等性：Parse 会先认领内容里已存在的 @[昵称](/author/1)（加入被 @ 名单但不重写），
 * 因此草稿转发布、发布后再编辑保存都不会漏通知；"是否重复通知"由 NotifyNew 的
 * filterFrom 参数处理（只通知相比旧内容新增的人）。
 */

// Mention @ 提及
type Mention struct{}

// MentionTarget 被 @ 到的用户
type MentionTarget struct {
	Id       int    `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// 昵称长度上限（users.nickname 是 size:32），匹配时最多往后看这么多字符
const mentionNameMaxLen = 32

// 昵称里会破坏 markdown 链接语法的字符（链接文字里出现 [ ] ( ) | 会截断链接）
var mentionNameCleaner = regexp.MustCompile(`[\[\]()|]`)

// 已规范化的链接：$1 = 昵称，$2 = 用户 id
var mentionLinkPattern = regexp.MustCompile(`@\[([^\[\]]*)\]\(/author/(\d+)\)`)

// Following 取「uid 已关注的人」作为 @ 候选列表（keyword 非空时按昵称模糊过滤）
//
// 前端选择器每次打开只请求一次（limit 取较大值），之后的输入过滤在本地做，
// 所以这里不做分页，只给一个足够大的上限。
func (this *Mention) Following(uid int, keyword string, limit int) []MentionTarget {
	list := []MentionTarget{}

	if uid <= 0 {
		return list
	}
	if limit <= 0 || limit > 500 {
		limit = 500
	}

	db := facade.DB.Drive()
	var users []Users

	// 子查询限定「我关注的人」，避免先查关注表拿到一堆 ID 再拼 IN 条件
	sub := db.Model(&UserFollows{}).Select("follow_uid").Where("uid = ? AND status = ?", uid, 1)
	query := db.Model(&users).Where("id IN (?)", sub)

	if kw := strings.TrimSpace(keyword); kw != "" {
		query = query.Where("nickname LIKE ?", "%"+kw+"%")
	}

	query.Order("id desc").Limit(limit).Find(&users)

	for _, item := range users {
		nickname := strings.TrimSpace(item.Nickname)
		if nickname == "" {
			continue
		}
		list = append(list, MentionTarget{Id: item.Id, Nickname: nickname, Avatar: item.Avatar})
	}

	return list
}

// Normalize 规范化内容：把 @昵称 换成 @[昵称](/author/uid)，并返回被 @ 到的人
//
// 供控制器在写库前调用（uid 传内容作者的 id —— 决定「可以 @ 谁」）。
func (this *Mention) Normalize(content string, uid int) (string, []MentionTarget) {
	return this.Parse(content, uid)
}

// Parse 规范化 content 里的 @昵称，并返回被 @ 到的用户（去重、不含作者自己）
//
// 匹配规则：
//   - 只匹配「已关注」的昵称（大小写不敏感；昵称不唯一时取 id 小者，保证结果确定）；
//   - 从 @ 之后取**最长**的命中昵称：中文昵称后面通常不带空格，不能用右侧边界判定，
//     例如「@张三你好」在存在昵称「张三你好」时匹配整串，否则退化为匹配「张三」；
//   - @ 左侧是字母 / 数字 / . _ - + % 时不算提及（避免把邮箱 a@b.com 的 @ 当提及）；
//   - 已规范化的 @[昵称](/author/1) 原样保留，同时计入被 @ 名单（保证幂等）。
func (this *Mention) Parse(content string, uid int) (string, []MentionTarget) {
	return this.parse(content, uid, this.Following(uid, "", 500))
}

// parse 是匹配逻辑本体（候选列表由调用方给，不查库，便于单测）
func (this *Mention) parse(content string, uid int, candidates []MentionTarget) (string, []MentionTarget) {
	if !strings.Contains(content, "@") {
		return content, nil
	}

	mentioned := []MentionTarget{}
	seen := map[int]bool{}

	// 先认领已规范化的链接：这些是上一次保存写的，保持原样，
	// 但要把它们算进"被 @ 的人"，否则「草稿 → 发布」这一步会漏掉通知。
	if links := mentionLinkPattern.FindAllStringSubmatch(content, -1); len(links) > 0 {
		for _, item := range links {
			id := cast.ToInt(item[2])
			if id <= 0 || id == uid || seen[id] {
				continue
			}
			seen[id] = true
			mentioned = append(mentioned, MentionTarget{Id: id, Nickname: item[1]})
		}
	}

	if len(candidates) == 0 {
		// 没关注任何人：已存在的链接照常保留，新的 @昵称 一律当普通文本
		return content, mentioned
	}

	// 昵称 → 用户
	nameMap := make(map[string]MentionTarget, len(candidates))
	for _, item := range candidates {
		key := strings.ToLower(item.Nickname)
		if old, ok := nameMap[key]; !ok || item.Id < old.Id {
			nameMap[key] = item
		}
	}

	runes := []rune(content)
	var builder strings.Builder
	builder.Grow(len(content))

	for i := 0; i < len(runes); {
		if runes[i] != '@' || (i > 0 && isMentionBoundary(runes[i-1])) {
			builder.WriteRune(runes[i])
			i++
			continue
		}

		target := MentionTarget{}
		end := 0
		for j := i + 1; j <= len(runes) && j-i-1 <= mentionNameMaxLen; j++ {
			if item, ok := nameMap[strings.ToLower(string(runes[i+1:j]))]; ok {
				target, end = item, j
			}
		}

		if end == 0 {
			builder.WriteRune(runes[i])
			i++
			continue
		}

		name := cleanMentionName(target.Nickname)
		if name == "" {
			builder.WriteRune(runes[i])
			i++
			continue
		}

		builder.WriteString(fmt.Sprintf("@[%v](/author/%v)", name, target.Id))

		if target.Id != uid && !seen[target.Id] {
			seen[target.Id] = true
			mentioned = append(mentioned, MentionTarget{Id: target.Id, Nickname: name})
		}

		i = end
	}

	return builder.String(), mentioned
}

// NotifyNew 发站内通知
//
//	fromUid    —— 触发人（内容作者）
//	sceneName  —— 场景名（评论 / 动态 / 文章 / 独立页）
//	bindType/bindId —— 点击通知后的跳转目标（沿用站内信既有约定）
//	summary    —— 内容摘要（内部的链接会被还原成 @昵称）
//	filterFrom —— 旧内容：非空时**只通知相比它新增**的被 @ 用户，
//	              避免编辑已发布内容时把同一个人反复打扰
func (this *Mention) NotifyNew(targets []MentionTarget, fromUid int, sceneName, bindType string, bindId int, summary, filterFrom string) {
	if len(targets) == 0 || fromUid <= 0 {
		return
	}

	if strings.TrimSpace(filterFrom) != "" {
		fresh := []MentionTarget{}
		old := map[int]bool{}
		for _, item := range this.mentionsOf(filterFrom, fromUid) {
			old[item.Id] = true
		}
		for _, item := range targets {
			if !old[item.Id] {
				fresh = append(fresh, item)
			}
		}
		targets = fresh
	}

	this.Notify(targets, fromUid, sceneName, bindType, bindId, summary)
}

// NotifyMentionsOnApprove 内容「审核通过」时补发 @ 提及通知
//
// 为什么需要它：内容处于待审核状态时**故意不发** @ 通知（被 @ 的人点进去看不到内容），
// 所以审核通过的那一刻必须补上 —— 否则在开启审核的站点里（contentAuditSwitch 缺配置时
// 默认就是开启），新发的文章 / 动态 / 页面会进入待审核，@ 通知就此永久丢失。
//
// 调用点：controller 的审核回调（notifyAuditChange 的 audit=1 分支、notifyMomentsAudit 的 case 1）。
// 内容里此时已是规范化的 @[昵称](/author/uid)，Parse 会把它们原样认领出来。
func NotifyMentionsOnApprove(kind string, bindId, ownerUid int) {
	if bindId <= 0 || ownerUid <= 0 {
		return
	}

	scene := ""
	var content string

	switch kind {
	case "article":
		scene = "文章"
		row, _ := facade.DB.Model(&Article{}).Where("id", bindId).Find()
		content = cast.ToString(cast.ToStringMap(row)["content"])
	case "page":
		scene = "页面"
		row, _ := facade.DB.Model(&Pages{}).Where("id", bindId).Find()
		content = cast.ToString(cast.ToStringMap(row)["content"])
	case "moments":
		scene = "动态"
		row, _ := facade.DB.Model(&Moments{}).Where("id", bindId).Find()
		content = cast.ToString(cast.ToStringMap(row)["content"])
	default:
		return
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	mention := &Mention{}
	_, targets := mention.Parse(content, ownerUid)
	if len(targets) == 0 {
		return
	}

	mention.Notify(targets, ownerUid, scene, kind, bindId, content)
}

// NotifyMentionsOnCommentApprove 评论「审核通过」时补发 @ 提及通知
//
// 与文章 / 动态 / 页面同理：评论处于待审核时**故意不发** @ 通知（评论不展示，
// 被 @ 的人点进去看不到），所以审核通过的那一刻必须补上，否则开启评论审核的站点里
// @ 通知会永久丢失。
//
// 跳转目标沿用评论创建时的口径：指向评论**所属内容**（article / page / moments），
// 而不是评论本身 —— 前台没有「评论详情页」，跳过去也只能落在内容页。
func NotifyMentionsOnCommentApprove(commentId int) {
	if commentId <= 0 {
		return
	}

	row, _ := facade.DB.Model(&Comment{}).Where("id", commentId).Find()
	item := cast.ToStringMap(row)

	// 仅作者可见的评论：被 @ 的人看不到正文，通知只会泄露内容，直接跳过
	if cast.ToInt(item["scope"]) == ScopeAuthor {
		return
	}

	content := strings.TrimSpace(cast.ToString(item["content"]))
	uid := cast.ToInt(item["uid"])
	if content == "" || uid <= 0 {
		return
	}

	mention := &Mention{}
	_, targets := mention.Parse(content, uid)
	if len(targets) == 0 {
		return
	}

	// 评论创建时已规范化过（待审核也照样规范化），这里只需认领出被 @ 的人
	bindType := strings.TrimSpace(cast.ToString(item["bind_type"]))
	bindId := cast.ToInt(item["bind_id"])
	if bindType == "" {
		bindType = "article"
	}

	mention.Notify(targets, uid, "评论", bindType, bindId, content)
}

// mentionsOf 取一段内容里的被 @ 名单（兼容未保存过的 @昵称 与已规范化的链接）
func (this *Mention) mentionsOf(content string, uid int) []MentionTarget {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	_, targets := this.Parse(content, uid)
	return targets
}

// Notify 给被 @ 的用户发站内通知（调用方一般直接用 NotifyNew）
func (this *Mention) Notify(targets []MentionTarget, fromUid int, sceneName, bindType string, bindId int, summary string) {
	if len(targets) == 0 || fromUid <= 0 {
		return
	}

	fromName := "有人"
	if userInfo, err := facade.DB.Model(&Users{}).Where("id", fromUid).Find(); err == nil {
		if name := strings.TrimSpace(cast.ToString(cast.ToStringMap(userInfo)["nickname"])); name != "" {
			fromName = name
		}
	}

	title := fmt.Sprintf("「%v」在%v中提到了你", fromName, sceneName)
	content := summarizeMentionContent(summary)

	for _, item := range targets {
		CreateUserNotify(item.Id, fromUid, NotificationTypeMention, title, content, bindType, bindId)
	}
}

// isMentionBoundary @ 左侧字符是这些时不算提及（邮箱、网址里的 @ 都会被跳过）
func isMentionBoundary(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return r == '.' || r == '_' || r == '-' || r == '+' || r == '%'
}

// cleanMentionName 去掉会破坏 markdown 链接语法的字符
func cleanMentionName(name string) string {
	return strings.TrimSpace(mentionNameCleaner.ReplaceAllString(name, ""))
}

// summarizeMentionContent 通知摘要：链接还原成 @昵称、压掉换行、截断到 120 字
func summarizeMentionContent(content string) string {
	text := mentionLinkPattern.ReplaceAllString(content, "@$1")
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))

	runes := []rune(text)
	if len(runes) > 120 {
		return string(runes[:120]) + "…"
	}
	return text
}
