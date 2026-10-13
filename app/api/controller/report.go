package controller

import (
	"errors"
	"math"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * Report - 举报（动态 / 文章 / 评论 / 用户）
 *
 * 前台：
 *   POST report/create   提交举报（登录 + 每天上限 + 同对象去重 + 内容快照）
 *   GET  report/mine     我提交的举报
 *   DELETE report/cancel 撤回（仅「待处理」可撤回）
 *
 * 后台（需要权限点，判定见 meta.permit）：
 *   GET  report/all      举报列表（可按状态筛选）
 *   PUT  report/handle   处理：属实(1) / 驳回(2)，可同时移除被举报内容
 *
 * 安全与隐私：
 *   1. 提交时必须登录，且每天有数量上限（reportMaxPerDay）——举报本身也是滥用入口
 *   2. 同一用户对同一对象只允许一条「待处理」举报，防刷屏
 *   3. 记录内容**快照**：原内容可能被作者删除，管理员仍能复核
 *   4. 只有后台权限点持有者能看别人提交的举报；举报人只看到自己的
 *   5. 处理结果只通知举报人（不含处理人 IP/UA 等管理侧信息）
 */
type Report struct {
	base
}

const (
	// reportMaxPerDay - 每人每天最多提交多少条举报（防刷）
	reportMaxPerDay = 20
	// reportSnapshotLen - 快照截断长度（按字符计，varchar(1024) 够用）
	reportSnapshotLen = 300
	// reportDefaultLimit - 后台列表默认每页条数
	reportDefaultLimit = 20
	// reportMaxPage - 后台列表最深页码
	reportMaxPage = 100
	// reportMaxImages - 举报证据图片数量上限（证据够用即可，不做相册）
	reportMaxImages = 3
	// reportImageUrlLen - 单条图片 URL 最大字符数（防超长垃圾串）
	reportImageUrlLen = 512
)

// reportAllowTargets - 允许举报的对象类型（与 model.ReportTargets 同源）
var reportAllowTargets = []any{"moment", "article", "comment", "user"}

func (this *Report) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"mine": this.mine,
		"all":  this.all,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Report) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"create": this.create,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Report) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"handle": this.handle,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Report) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"cancel": this.cancel,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Report) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// create - 提交举报
func (this *Report) create(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	targetType := strings.ToLower(strings.TrimSpace(cast.ToString(params["target_type"])))
	if !utils.In.Array(targetType, reportAllowTargets) {
		this.json(ctx, nil, facade.Lang(ctx, "举报对象类型不合法！"), 400)
		return
	}

	targetId := cast.ToInt(params["target_id"])
	if targetId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	reason := strings.TrimSpace(cast.ToString(params["reason"]))
	if _, ok := model.ReportReasons[reason]; !ok {
		this.json(ctx, nil, facade.Lang(ctx, "请选择举报原因！"), 400)
		return
	}

	detail := strings.TrimSpace(cast.ToString(params["detail"]))
	if len([]rune(detail)) > 200 {
		detail = string([]rune(detail)[:200])
	}

	reports := &model.Report{}

	// 防刷 1：每天上限
	if reports.CountTodayByUid(uid) >= reportMaxPerDay {
		this.json(ctx, nil, facade.Lang(ctx, "今天提交的举报太多了，请明天再试！"), 429)
		return
	}

	// 防刷 2：同一对象只允许一条待处理
	if reports.PendingExists(uid, targetType, targetId) {
		this.json(ctx, nil, facade.Lang(ctx, "你已经举报过了，我们正在处理中！"), 400)
		return
	}

	// 取举报对象：作者 + 内容快照
	targetUid, snapshot, err := this.reportTarget(targetType, targetId)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能举报自己！"), 400)
		return
	}

	// 证据图片：可选，最多 reportMaxImages 张；URL 逐条清洗（拒绝伪协议），只留 http(s)
	images := this.reportImages(params["images"])

	item := &model.Report{
		Uid:        uid,
		TargetType: targetType,
		TargetId:   targetId,
		TargetUid:  targetUid,
		Snapshot:   snapshot,
		Images:     images,
		Reason:     reason,
		Detail:     detail,
		Status:     model.ReportPending,
	}

	if _, err := reports.Create(item); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "提交失败，请稍后再试！"), 400)
		return
	}

	// 通知管理员（站内信始终发；邮件受「系统设置 → 邮件通知」的场景开关控制，未配置场景时静默跳过）
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "举报通知管理员时发生错误")
			}
		}()

		title := "收到一条新的举报"
		content := "类型：" + model.ReportTargetText(targetType) +
			" · 原因：" + model.ReportReasonText(reason) +
			" · 内容：" + truncateSafe(snapshot, 60) +
			" · 举报人 uid：" + cast.ToString(uid)

		imageList := this.reportImageList(images)
		if len(imageList) > 0 {
			content += " · 证据 " + cast.ToString(len(imageList)) + " 张"
		}

		// 站内信：无开关，落库即达（保证运营者不会漏看）
		model.NotifyAdmins(uid, model.NotificationTypeSystem, title, content, targetType, targetId)

		// 邮件：受「系统设置 → 邮件通知 → 举报治理 → 收到新举报」开关控制，
		// 场景未开启 / 未配置管理员邮箱时静默跳过（见 model.MailNotifyAdmin）
		lines := []string{
			"举报类型：" + model.ReportTargetText(targetType),
			"举报原因：" + model.ReportReasonText(reason),
			"内容快照：" + truncateSafe(snapshot, 80),
			"举报人：uid " + cast.ToString(uid),
			"举报对象 ID：" + cast.ToString(targetId),
			"处理入口：后台 → 举报管理",
		}
		if len(imageList) > 0 {
			// 证据图片直接给链接，管理员在邮件里就能点开看，不必先进后台
			lines = append(lines, "证据图片（"+cast.ToString(len(imageList))+" 张）："+strings.Join(imageList, " "))
		}

		model.MailNotifyAdmin("report.new", title, lines...)
	}()

	this.json(ctx, gin.H{"id": item.Id}, facade.Lang(ctx, "举报已提交，我们会尽快处理！"), 200)
}

// mine - 我提交的举报
func (this *Report) mine(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx, map[string]any{"page": 1, "limit": reportDefaultLimit})

	page := this.reportPage(params["page"])
	limit := this.reportLimit(params["limit"])

	data, count := (&model.Report{}).GetMine(uid, page, limit)

	list := make([]map[string]any, 0, len(data))
	for _, item := range data {
		list = append(list, this.reportOutput(item, false))
	}

	this.json(ctx, gin.H{
		"data":  list,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// all - 举报列表（后台）
func (this *Report) all(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx, map[string]any{"page": 1, "limit": reportDefaultLimit})

	page := this.reportPage(params["page"])
	limit := this.reportLimit(params["limit"])

	query := facade.DB.Model(&[]model.Report{})

	// 状态筛选：不传则全部（0 待处理 / 1 已处理 / 2 已驳回）
	if !utils.Is.Empty(params["status"]) {
		query = query.Where("status", cast.ToInt(params["status"]))
	}
	if !utils.Is.Empty(params["target_type"]) {
		query = query.Where("target_type", cast.ToString(params["target_type"]))
	}

	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("status asc, create_time desc").Select()

	list := make([]map[string]any, 0, len(data))
	for _, item := range data {
		list = append(list, this.reportOutput(item, true))
	}

	this.json(ctx, gin.H{
		"data":    list,
		"count":   count,
		"page":    math.Ceil(float64(count) / float64(limit)),
		"pending": (&model.Report{}).PendingCount(),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// handle - 处理举报（后台）
//
// status：1 属实 / 2 驳回；remove_content=1 时同时移除被举报内容（软删除，可恢复）。
func (this *Report) handle(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	id := cast.ToInt(params["id"])
	if id <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	status := cast.ToInt(params["status"])
	if status != model.ReportHandled && status != model.ReportRejected {
		this.json(ctx, nil, facade.Lang(ctx, "处理结果只能是「已处理」或「已驳回」！"), 400)
		return
	}

	remark := strings.TrimSpace(cast.ToString(params["remark"]))
	if len([]rune(remark)) > 200 {
		remark = string([]rune(remark)[:200])
	}

	item, _ := facade.DB.Model(&model.Report{}).Find(id)
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "举报记录不存在！"), 404)
		return
	}

	record := cast.ToStringMap(item)
	if cast.ToInt(record["status"]) != model.ReportPending {
		this.json(ctx, nil, facade.Lang(ctx, "该举报已被处理（%s），无需重复操作！", model.ReportStatusText(cast.ToInt(record["status"]))), 400)
		return
	}

	removeContent := false
	if status == model.ReportHandled && cast.ToBool(params["remove_content"]) {
		removeContent = this.removeReportedContent(cast.ToString(record["target_type"]), cast.ToInt(record["target_id"]))
	}

	// 审计留痕用 0/1（不是 bool，前端与旧数据都是数字口径）
	removeFlag := 0
	if removeContent {
		removeFlag = 1
	}

	operator := this.user(ctx)

	_, err := facade.DB.Model(&model.Report{}).Where("id", id).Update(map[string]any{
		"status":         status,
		"handler_uid":    operator.Id,
		"handle_remark":  remark,
		"handle_time":    time.Now().Unix(),
		"remove_content": removeFlag,
	})
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 通知举报人处理结果（只发必要信息，不带处理人身份与管理侧字段）
	reportUid := cast.ToInt(record["uid"])
	targetText := model.ReportTargetText(cast.ToString(record["target_type"]))

	if reportUid > 0 {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					facade.Log.Error(map[string]any{"error": r}, "举报结果通知发送失败")
				}
			}()

			title := "你的举报已处理"
			content := "类型：" + targetText + " · 结果：" + model.ReportStatusText(status)
			if removeContent {
				content += " · 相关内容已被移除"
			}
			if !utils.Is.Empty(remark) {
				content += " · 处理说明：" + remark
			}

			_, _ = (&model.Notification{}).CreateNotification(
				reportUid, 0, model.NotificationTypeSystem, title, content,
				cast.ToString(record["target_type"]), cast.ToInt(record["target_id"]),
			)
		}()
	}

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "已处理！"), 200)
}

// cancel - 撤回举报（仅举报人本人、仅待处理）
func (this *Report) cancel(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	id := cast.ToInt(params["id"])
	if id <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	item, _ := facade.DB.Model(&model.Report{}).Find(id)
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "举报记录不存在！"), 404)
		return
	}

	record := cast.ToStringMap(item)

	if cast.ToInt(record["uid"]) != uid {
		this.json(ctx, nil, facade.Lang(ctx, "无权操作他人的举报记录！"), 403)
		return
	}
	if cast.ToInt(record["status"]) != model.ReportPending {
		this.json(ctx, nil, facade.Lang(ctx, "该举报已被处理，无法撤回！"), 400)
		return
	}

	if _, err := facade.DB.Model(&model.Report{}).Where("id", id).Delete(); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "已撤回举报！"), 200)
}

// ============================== 内部方法 ==============================

// reportTarget - 取举报对象的作者与内容快照
//
// 返回 (作者uid, 快照, 错误)：对象不存在时返回错误，避免产生一条无法复核的举报。
func (this *Report) reportTarget(targetType string, targetId int) (int, string, error) {

	var item map[string]any

	switch targetType {
	case "moment":
		row, _ := facade.DB.Model(&model.Moments{}).Find(targetId)
		item = cast.ToStringMap(row)
	case "article":
		row, _ := facade.DB.Model(&model.Article{}).Find(targetId)
		item = cast.ToStringMap(row)
	case "comment":
		row, _ := facade.DB.Model(&model.Comment{}).Find(targetId)
		item = cast.ToStringMap(row)
	case "user":
		row, _ := facade.DB.Model(&model.Users{}).Find(targetId)
		item = cast.ToStringMap(row)
	default:
		return 0, "", errors.New("举报对象类型不合法！")
	}

	if utils.Is.Empty(item) {
		return 0, "", errors.New("举报的内容不存在或已被删除！")
	}

	// 用户举报：主体就是 TA 自己，快照用昵称 + 简介
	if targetType == "user" {
		targetUid := cast.ToInt(item["id"])
		snapshot := "用户：" + cast.ToString(item["nickname"]) + " · " + cast.ToString(item["description"])
		return targetUid, truncateSafe(snapshot, reportSnapshotLen), nil
	}

	snapshot := cast.ToString(item["content"])
	if targetType == "article" {
		snapshot = cast.ToString(item["title"]) + " " + snapshot
	}

	return cast.ToInt(item["uid"]), truncateSafe(snapshot, reportSnapshotLen), nil
}

// reportImages - 解析并清洗举报证据图片，返回逗号分隔的存储串
//
// 兼容前端传数组（[]any / []string）与逗号分隔字符串两种形式。
// 清洗规则：
//   - 逐条 TrimSpace，空值跳过
//   - 单条长度上限 reportImageUrlLen
//   - 走 facade.Comm.SanitizeURL 剥离危险字符，且必须仍是 http(s)（挡掉 javascript: / data: 这类伪协议）
//   - 数量上限 reportMaxImages，超出直接丢弃（不报错，避免因为多选一张图导致整单提交失败）
func (this *Report) reportImages(value any) string {

	raw := make([]string, 0)

	switch v := value.(type) {
	case []any:
		for _, item := range v {
			raw = append(raw, cast.ToString(item))
		}
	case []string:
		raw = append(raw, v...)
	case string:
		raw = append(raw, strings.Split(v, ",")...)
	}

	clean := make([]string, 0, reportMaxImages)

	for _, url := range raw {

		url = strings.TrimSpace(url)
		if utils.Is.Empty(url) || len([]rune(url)) > reportImageUrlLen {
			continue
		}

		url = strings.TrimSpace(facade.Comm.SanitizeURL(url))
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}

		clean = append(clean, url)
		if len(clean) >= reportMaxImages {
			break
		}
	}

	return strings.Join(clean, ",")
}

// reportImageList - 逗号分隔串 → 数组（输出给前端用；空值返回空数组，前端不必判空）
func (this *Report) reportImageList(value string) []string {

	value = strings.TrimSpace(value)
	if utils.Is.Empty(value) {
		return []string{}
	}

	list := make([]string, 0, reportMaxImages)
	for _, url := range strings.Split(value, ",") {
		url = strings.TrimSpace(url)
		if utils.Is.Empty(url) {
			continue
		}
		list = append(list, url)
	}

	return list
}

// removeReportedContent - 处理举报时移除被举报内容（软删除，可在回收站恢复）
//
// 只处理内容类对象；「用户」举报不在这里动账号（封禁要走用户管理的封禁流程，带申诉与留痕）。
func (this *Report) removeReportedContent(targetType string, targetId int) bool {

	if targetId <= 0 {
		return false
	}

	var err error

	switch targetType {
	case "moment":
		_, err = facade.DB.Model(&model.Moments{}).Where("id", targetId).Delete()
	case "article":
		_, err = facade.DB.Model(&model.Article{}).Where("id", targetId).Delete()
	case "comment":
		_, err = facade.DB.Model(&model.Comment{}).Where("id", targetId).Delete()
	default:
		return false
	}

	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "target_type": targetType, "target_id": targetId}, "移除被举报内容失败")
		return false
	}

	return true
}

// reportOutput - 输出结构（统一补上文案，前端不必再维护一份映射）
func (this *Report) reportOutput(item map[string]any, admin bool) map[string]any {

	item["target_type_text"] = model.ReportTargetText(cast.ToString(item["target_type"]))
	item["reason_text"] = model.ReportReasonText(cast.ToString(item["reason"]))
	item["status_text"] = model.ReportStatusText(cast.ToInt(item["status"]))
	// 证据图片：库里是逗号分隔字符串，统一转成数组再出网（前台/后台都直接渲染缩略图）
	item["images"] = this.reportImageList(cast.ToString(item["images"]))

	// 举报人与被举报人：只给公开字段（白名单 + 清洗），后台也一样
	item["reporter"] = this.reportUserBrief(cast.ToInt(item["uid"]))
	item["target_user"] = this.reportUserBrief(cast.ToInt(item["target_uid"]))

	// 处理人的身份只在后台展示（前台「我的举报」不需要，也避免暴露管理员）
	if !admin {
		delete(item, "handler_uid")
	}

	return item
}

// reportUserBrief - 用户公开信息（白名单 + 隐私清洗）
func (this *Report) reportUserBrief(uid int) map[string]any {

	if uid <= 0 {
		return map[string]any{}
	}

	item, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(item) {
		return map[string]any{"id": uid}
	}

	user := utils.Map.WithField(item, []string{"id", "nickname", "avatar", "title"})
	model.SanitizeAuthor(user)

	return user
}

// reportPage - 页码钳制
func (this *Report) reportPage(value any) int {
	page := cast.ToInt(value)
	if page <= 0 {
		return 1
	}
	if page > reportMaxPage {
		return reportMaxPage
	}
	return page
}

// reportLimit - 单页条数钳制
func (this *Report) reportLimit(value any) int {
	limit := cast.ToInt(value)
	if limit <= 0 {
		return reportDefaultLimit
	}
	if limit > 50 {
		return 50
	}
	return limit
}
