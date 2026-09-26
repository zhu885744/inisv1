package model

/**
 * 统一邮件通知（按场景开关）
 *
 * 目的：把「审核、订单、封禁」等运营通知（发给管理员或相关用户）集中到一处：
 *   - 所有场景在同一份配置里注册（MailNotifyScenes），后台「系统设置 → 邮件通知」统一开关；
 *   - 总开关（enabled）关闭时全部不发；单个场景关闭时只禁用它自己；
 *   - 发送走邮箱队列（分批限流 + 失败延迟重试），调用方不阻塞，见 facade/mail_queue.go。
 *
 * 配置存放：config 表 SYSTEM_MAIL_NOTIFY 记录的 json 字段
 *   {
 *     "enabled": 1,            // 总开关：0 关闭（所有场景都停发）
 *     "admin_email": "",       // 管理员收件邮箱（多个用 , 或 ; 分隔；留空则用「超级管理员」账号的邮箱）
 *     "scenes": { "article.pending": 1, ... }   // 各场景开关，1 开 / 0 关
 *   }
 *
 * 用法（在业务代码里一行接入）：
 *   MailNotifyAdmin("article.pending", "有新的文章待审核", append(MailNotifyUserInfo(uid), "标题："+title)...)
 *   MailNotifyUser(uid, "article.passed", "您的文章已通过审核", "标题："+title)
 *
 * 身份信息：涉及用户的邮件统一在正文开头带上「账号 / 昵称」——
 *   - 发给用户的邮件由 MailNotifyUser 自动追加，调用方不用自己拼；
 *   - 发给管理员的邮件若描述某个用户，用 append(MailNotifyUserInfo(uid), 其它行...)... 拼在前面。
 *
 * 注意：场景未在配置里出现时按场景默认值（Default）处理，因此老库不需要先落库这条配置。
 */

import (
	"fmt"
	"strings"
	"time"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 通知目标
const (
	MailTargetAdmin = "admin" // 发给管理员（站点运营者）
	MailTargetUser  = "user"  // 发给相关用户（作者 / 买家 / 被处理的用户）
)

// MailNotifyConfigKey 邮件通知配置的 config 表 key
const MailNotifyConfigKey = "SYSTEM_MAIL_NOTIFY"

// MailNotifyScene 一个可开关的邮件通知场景
type MailNotifyScene struct {
	Key     string `json:"key"`     // 场景 key（配置里的字段名）
	Label   string `json:"label"`   // 后台展示名
	Group   string `json:"group"`   // 分组（后台按此分组展示）
	Target  string `json:"target"`  // 收件对象：admin / user
	Default int    `json:"default"` // 默认开关：1 开 / 0 关
	Desc    string `json:"desc"`    // 说明
}

// MailNotifyScenes 全部场景（新增通知时先在这里注册，再在业务处调用）
func MailNotifyScenes() []MailNotifyScene {
	return []MailNotifyScene{
		// ---------- 内容审核 ----------
		{Key: "article.pending", Label: "文章待审核", Group: "内容审核", Target: MailTargetAdmin, Default: 1, Desc: "作者提交待审核文章时，通知管理员去审核"},
		{Key: "article.passed", Label: "文章审核通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "文章审核通过时通知作者"},
		{Key: "article.rejected", Label: "文章审核未通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "文章被驳回时通知作者"},
		{Key: "page.pending", Label: "页面待审核", Group: "内容审核", Target: MailTargetAdmin, Default: 1, Desc: "提交待审核独立页面时，通知管理员"},
		{Key: "page.passed", Label: "页面审核通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "独立页面审核通过时通知作者"},
		{Key: "page.rejected", Label: "页面审核未通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "独立页面被驳回时通知作者"},
		{Key: "links.pending", Label: "友链待审核", Group: "内容审核", Target: MailTargetAdmin, Default: 1, Desc: "有新的友链申请时，通知管理员"},
		{Key: "links.passed", Label: "友链审核通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "友链通过审核时通知申请者"},
		{Key: "links.rejected", Label: "友链审核未通过", Group: "内容审核", Target: MailTargetUser, Default: 1, Desc: "友链被驳回时通知申请者"},

		// ---------- 评论互动 ----------
		{Key: "comment.notify", Label: "评论通知（内容作者）", Group: "评论互动", Target: MailTargetUser, Default: 1, Desc: "有人评论文章 / 页面 / 动态时通知内容作者"},
		{Key: "comment.reply", Label: "回复通知（被回复人）", Group: "评论互动", Target: MailTargetUser, Default: 1, Desc: "有人回复评论时通知被回复的用户"},

		// ---------- 用户管理 ----------
		{Key: "user.pending", Label: "新用户待审核", Group: "用户管理", Target: MailTargetAdmin, Default: 1, Desc: "注册需人工审核时，通知管理员审核"},
		{Key: "user.passed", Label: "用户审核通过", Group: "用户管理", Target: MailTargetUser, Default: 1, Desc: "账号通过人工审核时通知用户"},
		{Key: "user.frozen", Label: "账号被冻结", Group: "用户管理", Target: MailTargetUser, Default: 1, Desc: "管理员冻结账号时通知用户"},
		{Key: "user.unfrozen", Label: "账号解除冻结", Group: "用户管理", Target: MailTargetUser, Default: 1, Desc: "账号恢复正常时通知用户"},
		{Key: "user.banned", Label: "账号被封禁", Group: "用户管理", Target: MailTargetUser, Default: 1, Desc: "管理员封禁账号时通知用户（含原因与到期时间）"},
		{Key: "user.unbanned", Label: "账号解除封禁", Group: "用户管理", Target: MailTargetUser, Default: 1, Desc: "账号解封时通知用户（手动解封与到期自动解封都会发）"},

		// ---------- 积分商城 ----------
		{Key: "order.paid", Label: "订单支付成功（通知管理员）", Group: "积分商城", Target: MailTargetAdmin, Default: 1, Desc: "用户下单扣除积分成功后，通知管理员（实物商品需要发货）"},
		{Key: "order.shipped", Label: "订单已发货", Group: "积分商城", Target: MailTargetUser, Default: 1, Desc: "管理员把订单标记为已发货时通知买家"},
		{Key: "order.canceled", Label: "订单已取消", Group: "积分商城", Target: MailTargetUser, Default: 1, Desc: "订单取消并退还积分时通知买家"},
	}
}

// MailNotifyDefaultConfig 默认配置（未落库 / 缺字段时兜底）
func MailNotifyDefaultConfig() map[string]any {
	scenes := make(map[string]any, 0)
	for _, scene := range MailNotifyScenes() {
		scenes[scene.Key] = scene.Default
	}

	return map[string]any{
		"enabled":     1,
		"admin_email": "",
		"scenes":      scenes,
	}
}

// MailNotifyMergeConfig 用默认值补齐配置里缺失的字段
//
// 兼容三种情况：配置记录不存在、老库没有的场景 key、后台只保存了部分字段。
func MailNotifyMergeConfig(config map[string]any) map[string]any {
	scenes := cast.ToStringMap(MailNotifyDefaultConfig()["scenes"])
	for key, val := range cast.ToStringMap(config["scenes"]) {
		scenes[key] = val
	}

	result := MailNotifyDefaultConfig()
	if _, ok := config["enabled"]; ok {
		result["enabled"] = cast.ToInt(config["enabled"])
	}
	if _, ok := config["admin_email"]; ok {
		result["admin_email"] = cast.ToString(config["admin_email"])
	}
	result["scenes"] = scenes

	return result
}

// MailNotifyConfig 读取邮件通知配置（config 表 SYSTEM_MAIL_NOTIFY，带缓存）
//
// 缓存名与 config 控制器写入时清理的键一致（config[KEY]），改完配置立即生效。
func MailNotifyConfig() map[string]any {
	cacheName := "config[" + MailNotifyConfigKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return MailNotifyMergeConfig(cast.ToStringMap(facade.Cache.Get(cacheName)))
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", MailNotifyConfigKey).Find()
	config := cast.ToStringMap(cast.ToStringMap(item)["json"])

	if cacheState {
		go facade.Cache.Set(cacheName, config)
	}

	return MailNotifyMergeConfig(config)
}

// MailNotifyEnabled 场景是否开启（总开关关闭时一律为 false）
func MailNotifyEnabled(scene string) bool {
	config := MailNotifyConfig()
	if cast.ToInt(config["enabled"]) != 1 {
		return false
	}
	if utils.Is.Empty(scene) {
		return true
	}
	return cast.ToInt(cast.ToStringMap(config["scenes"])[scene]) == 1
}

// MailNotifyAdmin 发邮件给管理员（场景关闭 / 没有收件人时静默跳过）
func MailNotifyAdmin(scene, title string, lines ...string) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "scene": scene}, "发送管理员通知邮件时发生panic")
		}
	}()

	if !MailNotifyEnabled(scene) {
		return
	}

	emails := MailNotifyAdminEmails()
	if utils.Is.Empty(emails) {
		facade.Log.Warn(map[string]any{"scene": scene}, "未配置管理员收件邮箱（且没有可用的超级管理员邮箱），跳过邮件通知")
		return
	}

	for _, email := range emails {
		sendMailNotify(email, title, lines)
	}
}

// MailNotifyUser 发邮件给指定用户（场景关闭 / 用户没有邮箱时静默跳过）
func MailNotifyUser(uid int, scene, title string, lines ...string) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "scene": scene, "uid": uid}, "发送用户通知邮件时发生panic")
		}
	}()

	if !MailNotifyEnabled(scene) || uid <= 0 {
		return
	}

	email := MailNotifyUserEmail(uid)
	if utils.Is.Empty(email) {
		return
	}

	// 正文开头统一带上「账号 / 昵称」，收件人不用猜这封邮件说的是哪个账号
	body := append(MailNotifyUserInfo(uid), lines...)

	sendMailNotify(email, title, body)
}

// MailNotifyAdminEmails 管理员收件邮箱
//
// 取值顺序：配置里的 admin_email（多个用 , ; 空格分隔）→ 超级管理员（权限组 root=1）账号邮箱。
func MailNotifyAdminEmails() []string {
	config := MailNotifyConfig()
	if emails := parseEmails(cast.ToString(config["admin_email"])); !utils.Is.Empty(emails) {
		return emails
	}

	return superAdminEmails()
}

// MailNotifyUserEmail 取用户邮箱（没有 / 格式不对时返回空字符串）
func MailNotifyUserEmail(uid int) string {
	user, _ := facade.DB.Model(&Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return ""
	}

	email := cast.ToString(cast.ToStringMap(user)["email"])
	if !utils.Is.Email(email) {
		return ""
	}

	return email
}

// MailNotifyUserInfo 用户身份信息（账号 / 昵称）两行文案
//
// 涉及用户的邮件统一用它拼正文：发给用户的邮件由 MailNotifyUser 自动带上，
// 发给管理员的邮件（文章待审核、新用户待审核、订单支付……）用
// append(MailNotifyUserInfo(uid), 其它行...)... 拼在开头。
func MailNotifyUserInfo(uid int) []string {
	account, nickname := MailNotifyUserIdentity(uid)
	return []string{"账号：" + account, "昵称：" + nickname}
}

// MailNotifyUserIdentity 取用户的账号与昵称（查不到 / 为空时回退「用户 #id」或账号）
func MailNotifyUserIdentity(uid int) (account, nickname string) {
	fallback := fmt.Sprintf("用户 #%v", uid)

	user, _ := facade.DB.Model(&Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return fallback, fallback
	}

	item := cast.ToStringMap(user)
	account = strings.TrimSpace(cast.ToString(item["account"]))
	nickname = strings.TrimSpace(cast.ToString(item["nickname"]))

	if utils.Is.Empty(account) {
		account = fallback
	}
	if utils.Is.Empty(nickname) {
		nickname = account
	}

	return account, nickname
}

// MailNotifyTime 邮件正文里的时间文案
func MailNotifyTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// MailNotifySiteURL 站点地址（用于正文里给出跳转提示，取不到时返回空）
func MailNotifySiteURL() string {
	return strings.TrimRight(cast.ToString(facade.AppToml.Get("app.domain", "")), "/")
}

// superAdminEmails 超级管理员邮箱（权限组 root=1 的成员，去重后返回）
func superAdminEmails() []string {
	groups, _ := facade.DB.Model(&[]AuthGroup{}).Where("root", 1).Select()

	var uids []any
	for _, group := range groups {
		for _, uid := range utils.Unity.Ids(group["uids"]) {
			if !utils.InArray(uid, uids) {
				uids = append(uids, uid)
			}
		}
	}

	if utils.Is.Empty(uids) {
		return nil
	}

	users, _ := facade.DB.Model(&[]Users{}).WhereIn("id", uids).Select()

	var emails []string
	for _, user := range users {
		for _, email := range parseEmails(cast.ToString(user["email"])) {
			if !hasEmail(emails, email) {
				emails = append(emails, email)
			}
		}
	}

	return emails
}

// hasEmail 邮箱是否已在列表中（收件人去重）
func hasEmail(emails []string, email string) bool {
	for _, item := range emails {
		if item == email {
			return true
		}
	}
	return false
}

// parseEmails 解析邮箱串（支持 , ; 空格 / 中文逗号分隔，并过滤非法值）
func parseEmails(raw string) []string {
	raw = strings.NewReplacer(",", " ", ";", " ", "，", " ", "\n", " ").Replace(raw)

	var emails []string
	for _, item := range strings.Fields(raw) {
		if !utils.Is.Email(item) {
			continue
		}
		if !hasEmail(emails, item) {
			emails = append(emails, item)
		}
	}

	return emails
}

// MailNotifyComment 评论通知邮件（发给内容作者，沿用「新评论通知」HTML 卡片模板）
//
// 与 MailNotifyUser 的区别只在邮件模板：评论类通知带评论正文、评论者、IP 等卡片样式。
// 场景开关（comment.notify）、分批投递与失败重试与其它通知完全一致（都走邮箱队列）。
// 收件人邮箱由调用方解析（评论模块本来就查过内容作者的信息），非法邮箱直接跳过。
func MailNotifyComment(email string, info map[string]any) {
	mailNotifyCommentTpl("comment.notify", email, info, facade.SendCommentNotify)
}

// MailNotifyReply 回复通知邮件（发给被回复的人，沿用「评论回复通知」HTML 卡片模板）
// 场景开关（comment.reply）与投递策略同上。
func MailNotifyReply(email string, info map[string]any) {
	mailNotifyCommentTpl("comment.reply", email, info, facade.SendReplyNotify)
}

// mailNotifyCommentTpl 评论类通知的公共流程：开关判断 → 校验收件人 → 按指定模板入队
func mailNotifyCommentTpl(scene, email string, info map[string]any, send func(string, map[string]any) *facade.SMSResponse) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err, "scene": scene}, "发送评论通知邮件时发生panic")
		}
	}()

	if !MailNotifyEnabled(scene) || !utils.Is.Email(email) {
		return
	}

	// 入队失败（如队列已满 / 邮件服务未初始化）只记日志，不影响评论创建
	if response := send(email, info); response != nil && response.Error != nil {
		facade.Log.Warn(map[string]any{
			"scene":     scene,
			"recipient": facade.Comm.MaskEmail(email),
			"error":     response.Error,
		}, "评论通知邮件入队失败")
	}
}

// sendMailNotify 统一投递（走邮箱队列：分批限流 + 失败延迟重试，非阻塞）
func sendMailNotify(email, title string, lines []string) {
	body := []string{title, ""}
	for _, line := range lines {
		if utils.Is.Empty(strings.TrimSpace(line)) {
			continue
		}
		body = append(body, line)
	}
	body = append(body, "", "本邮件由系统自动发送，无需回复。")

	facade.SendMail(email, fmt.Sprintf("%v - %v", title, SiteTitle()), strings.Join(body, "\n"))
}
