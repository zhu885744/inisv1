package controller

import (
	"math"
	"strconv"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * UserCertification - 用户认证（职业 / 兴趣 / 机构）
 *
 * 前台：
 *   GET  user-certification/one   某个用户的认证摘要（公开，用户主页展示用）
 *   GET  user-certification/mine  我的认证（含审核状态、驳回原因、已提交资料）
 *   POST user-certification/apply 提交 / 修改认证申请（登录；通过后重新提交需重新审核）
 *
 * 后台（需要权限点，判定见 meta.permit）：
 *   GET  user-certification/all    认证列表（按状态 / 类型筛选）
 *   POST user-certification/grant  直接授予认证（运营主动核验，不经过用户提交）
 *   PUT  user-certification/edit   修改用户提交的资料（不改变审核状态）
 *   PUT  user-certification/handle 审核：通过(1) / 驳回(2，必须写原因)
 *   DEL  user-certification/revoke 收回已发出的认证（记录删除，用户可重新申请）
 *
 * 安全与隐私：
 *   1. 证明材料、联系方式、公司全称**只在后台与本人接口**返回；公开的 one 只给展示字段
 *      （口径见 model.CertSummary），避免把用户证件类图片暴露给任何访客
 *   2. 提交频率有下限间隔（certSubmitInterval），防脚本刷审核队列
 *   3. 通过 / 驳回都会发站内信，驳回原因随通知一起发出
 *   4. 一人一条记录：再次提交是覆盖更新，不做多版本堆叠
 *
 * 可配置化（见 model.CertType）：
 *   - 认证类型 / 该类型下的领域 / 需要用户填写什么，全部由后台配置，本文件不再硬编码；
 *   - 「认证说明」是后台审核通过时填写的对外文案，用户提交的参数里即使带了也不会被采纳；
 *   - 用户提交的字段值统一存进 data（JSON），内置键 title / category / images 额外写回独立列，
 *     保证主页标识、后台筛选、证明材料这些既有能力不受配置改动影响。
 */
type UserCertification struct {
	base
}

const (
	// certDefaultLimit - 后台列表默认每页条数
	certDefaultLimit = 20
	// certMaxPage - 后台列表最深页码
	certMaxPage = 200
	// certMaxImages - 证明材料图片数量上限
	certMaxImages = 6
	// certImageUrlLen - 单条图片 URL 最大字符数
	certImageUrlLen = 512
	// certTitleLen / certCategoryLen / certCompanyLen / certPositionLen / certDescLen / certContactLen
	// 与模型字段长度对齐，超长直接截断（不报错，避免整单提交失败）
	// 字段本身的长度由后台配置（model.CertField.Max）决定，这里只用于写回独立列时兜底
	certTitleLen    = 32
	certCategoryLen = 32
	certCompanyLen  = 64
	certPositionLen = 32
	certDescLen     = 255
	certContactLen  = 64
	// certReasonLen - 驳回原因长度上限
	certReasonLen = 200
	// certSubmitInterval - 两次提交的最小间隔（秒），防刷审核队列
	certSubmitInterval = 60
)

func (this *UserCertification) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"one":  this.one,
		"mine": this.mine,
		"all":  this.all,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserCertification) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"apply": this.apply,
		"grant": this.grant,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserCertification) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"handle": this.handle,
		"edit":   this.edit,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserCertification) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"revoke": this.revoke,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *UserCertification) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// one - 用户认证摘要（公开）
func (this *UserCertification) one(ctx *gin.Context) {

	params := this.params(ctx)

	uid := cast.ToInt(params["uid"])
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	// 已认证返回摘要，未认证返回 null（前端按「无认证」渲染，不当作错误）
	this.json(ctx, model.CertSummary(uid), facade.Lang(ctx, "查询成功！"), 200)
}

// mine - 我的认证（含未通过 / 待审核）
func (this *UserCertification) mine(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	record := (&model.UserCertification{}).GetByUid(uid)
	if record == nil {
		this.json(ctx, nil, facade.Lang(ctx, "还没有提交认证申请。"), 200)
		return
	}

	this.json(ctx, this.certOutput(record, false, this.schemas()), facade.Lang(ctx, "查询成功！"), 200)
}

// apply - 提交 / 修改认证申请
//
// 表单完全由后台配置的类型决定（model.CertType.Fields）：
//   - 新前端把值放在 values: {字段key: 值}；
//   - 老前端是平铺传的（title / category / images …），没有 values 时按平铺解析，保持兼容；
//   - 「认证说明」不在用户可填范围内，提交时一律清空，等后台审核通过时填写。
func (this *UserCertification) apply(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	certType := cast.ToInt(params["cert_type"])
	config := model.CertTypeFind(certType)
	if config == nil || cast.ToInt(config["enable"]) != 1 {
		this.json(ctx, nil, facade.Lang(ctx, "请选择认证类型！"), 400)
		return
	}

	fields, _ := config["fields"].([]model.CertField)
	categories, _ := config["categories"].([]string)
	if len(fields) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该认证类型暂未开放申请！"), 400)
		return
	}

	// 申请门槛：粉丝数 / 文章数 / 动态数 / 注册天数（全部 0 表示不限）
	// 放在字段校验之前：门槛不达标时，报「还差多少粉丝」比报某个字段没填更有意义
	conditions := model.CertConditionsFromAny(config["conditions"])
	if !conditions.Empty() {
		if met, missing := conditions.Check(model.CertUserStats(uid)); !met {
			this.json(ctx, nil, facade.Lang(ctx, conditions.CheckMessage(missing)), 400)
			return
		}
	}

	values := cast.ToStringMap(params["values"])
	if len(values) == 0 {
		values = params
	}

	data, images, title, category, msg := this.certValues(fields, categories, values, true)
	if msg != "" {
		this.json(ctx, nil, facade.Lang(ctx, msg), 400)
		return
	}

	// 独立列有长度上限（title / category 分别是 32），字段配置写得再长也要截断，避免写库报错
	title = truncateSafe(title, certTitleLen)
	category = truncateSafe(category, certCategoryLen)

	// 认证名称是主页标识的展示文案：后台可能把 title 字段改名或删掉，
	// 这时回落到类型名，保证认证标识不会因为「title 列为空」而整个不显示
	if title == "" {
		title = truncateSafe(cast.ToString(config["name"]), certTitleLen)
	}

	// 已通过的认证重新提交后要重新审核，期间主页的认证展示会暂时下线——先把话说明白
	exist := (&model.UserCertification{}).GetByUid(uid)

	if exist != nil {
		if cast.ToInt(exist["status"]) == model.CertPending {
			this.json(ctx, nil, facade.Lang(ctx, "申请正在审核中，请耐心等待，无需重复提交！"), 400)
			return
		}
		// 频率限制：避免脚本反复提交把审核队列刷满
		if last := cast.ToInt64(exist["update_time"]); last > 0 && time.Now().Unix()-last < certSubmitInterval {
			this.json(ctx, nil, facade.Lang(ctx, "提交过于频繁，请稍后再试！"), 400)
			return
		}
	}

	if exist == nil {
		record := model.UserCertification{
			Uid:      uid,
			CertType: certType,
			Category: category,
			Title:    title,
			Images:   images,
			Data:     utils.Json.Encode(data),
			Status:   model.CertPending,
		}
		this.certLegacyFill(&record, data)

		if _, err := facade.DB.Model(&model.UserCertification{}).Create(&record); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	} else {
		// 覆盖更新：重新提交即回到待审核，清空上一次的审核结论
		payload := map[string]any{
			"cert_type":   certType,
			"category":    category,
			"title":       title,
			"company":     cast.ToString(data["company"]),
			"position":    cast.ToString(data["position"]),
			"contact":     cast.ToString(data["contact"]),
			"images":      images,
			"data":        utils.Json.Encode(data),
			"description": "",
			"status":      model.CertPending,
			"reason":      "",
			"handler_uid": 0,
			"handle_time": 0,
			"expire_time": 0,
		}

		if _, err := facade.DB.Model(&model.UserCertification{}).Where("uid", uid).Update(payload); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	}

	go this.delCache(uid)

	this.json(ctx, gin.H{"status": model.CertPending}, facade.Lang(ctx, "已提交，等待管理员审核！"), 200)
}

// all - 认证列表（后台）
func (this *UserCertification) all(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx, map[string]any{"page": 1, "limit": certDefaultLimit})

	page := this.certPage(params["page"])
	limit := this.certLimit(params["limit"])

	query := facade.DB.Model(&[]model.UserCertification{})

	// 注意：这里别用 utils.Is.Empty 判空——它对字符串 "0" 的口径容易把「待审核」筛掉，
	// 统一按「空串 = 不筛」处理
	if value := strings.TrimSpace(cast.ToString(params["status"])); value != "" {
		query = query.Where("status", cast.ToInt(value))
	}
	if value := strings.TrimSpace(cast.ToString(params["cert_type"])); value != "" {
		query = query.Where("cert_type", cast.ToInt(value))
	}
	if value := strings.TrimSpace(cast.ToString(params["uid"])); value != "" {
		query = query.Where("uid", cast.ToInt(value))
	}
	// 领域是自由文本（后台可随时改类型下的领域），按值精确匹配即可
	if value := strings.TrimSpace(cast.ToString(params["category"])); value != "" {
		query = query.Where("category", value)
	}

	count, _ := query.Count()
	data, _ := query.Limit(limit).Page(page).Order("status asc, create_time desc").Select()

	schemas := this.schemas()

	list := make([]map[string]any, 0, len(data))
	for _, item := range data {
		list = append(list, this.certOutput(item, true, schemas))
	}

	this.json(ctx, gin.H{
		"data":    list,
		"count":   count,
		"page":    math.Ceil(float64(count) / float64(limit)),
		"pending": (&model.UserCertification{}).PendingCount(),
		// 类型筛选项随列表一起返回，后台不必再单独请求一次配置
		"types": schemas.options(),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// handle - 审核认证申请（后台）
func (this *UserCertification) handle(ctx *gin.Context) {

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
	if status != model.CertPassed && status != model.CertRejected {
		this.json(ctx, nil, facade.Lang(ctx, "审核结果只能是「通过」或「驳回」！"), 400)
		return
	}

	reason := truncateSafe(strings.TrimSpace(cast.ToString(params["reason"])), certReasonLen)
	if status == model.CertRejected && utils.Is.Empty(reason) {
		this.json(ctx, nil, facade.Lang(ctx, "驳回时必须填写原因！"), 400)
		return
	}

	// 认证说明：只有后台能写，通过时随审核一起落库（对外展示的一句话）
	description := truncateSafe(strings.TrimSpace(cast.ToString(params["description"])), certDescLen)

	// 有效期：单位天；0 / 不传表示永久
	expireTime := int64(0)
	if days := cast.ToInt64(params["expire_days"]); days > 0 {
		expireTime = time.Now().AddDate(0, 0, cast.ToInt(days)).Unix()
	}

	item, _ := facade.DB.Model(&model.UserCertification{}).Find(id)
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "认证记录不存在！"), 404)
		return
	}
	record := cast.ToStringMap(item)

	operator := this.user(ctx)

	payload := map[string]any{
		"status":      status,
		"handler_uid": operator.Id,
		"handle_time": time.Now().Unix(),
		"expire_time": expireTime,
	}
	if status == model.CertPassed {
		// 通过时清掉历史驳回原因，避免主页/个人中心继续显示上一次的理由
		payload["reason"] = ""
		// 认证说明由后台填写；不传则保留原值（重新审核时不必每次重写）
		if _, ok := params["description"]; ok {
			payload["description"] = description
		}
	} else {
		payload["reason"] = reason
	}

	_, err := facade.DB.Model(&model.UserCertification{}).Where("id", id).Update(payload)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache(cast.ToInt(record["uid"]))
	go this.notify(cast.ToInt(record["uid"]), cast.ToInt(record["id"]), cast.ToString(record["title"]), status, reason)

	this.json(ctx, gin.H{"id": id, "status": status}, facade.Lang(ctx, "已处理！"), 200)
}

// grant - 后台直接授予认证（不经过用户提交 / 审核队列）
//
// 场景：知名创作者、机构账号由运营线下核验后直接授予，用户端从头到尾没有提交过申请。
// 与 apply 的差异：
//   - 跳过申请门槛（粉丝数 / 注册天数等是「用户自助申请」的门槛，运营核实过就不再拦）；
//   - 跳过 certSubmitInterval 频率限制（那是防脚本刷审核队列的，后台操作不需要）；
//   - 直接落 status=1，同时写好 handler_uid / handle_time / expire_time。
//
// 字段校验仍走 certValues（类型 / 选项范围 / 长度 / 图片张数），但必填项不强制：
// 运营手上的资料未必齐全，缺认证名称时回落到类型名，主页标识不受影响。
// 一人一条记录（uk_cert_uid），已有记录则整条覆盖。
func (this *UserCertification) grant(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	uid := cast.ToInt(params["uid"])
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	user, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	certType := cast.ToInt(params["cert_type"])
	config := model.CertTypeFind(certType)
	if config == nil {
		this.json(ctx, nil, facade.Lang(ctx, "请选择认证类型！"), 400)
		return
	}

	fields, _ := config["fields"].([]model.CertField)
	categories, _ := config["categories"].([]string)

	data, images, title, category, msg := this.certValues(fields, categories, cast.ToStringMap(params["values"]), false)
	if msg != "" {
		this.json(ctx, nil, facade.Lang(ctx, msg), 400)
		return
	}

	title = truncateSafe(title, certTitleLen)
	category = truncateSafe(category, certCategoryLen)
	if title == "" {
		title = truncateSafe(cast.ToString(config["name"]), certTitleLen)
	}

	description := truncateSafe(strings.TrimSpace(cast.ToString(params["description"])), certDescLen)

	// 有效期：单位天；0 / 不传表示永久
	expireTime := int64(0)
	if days := cast.ToInt64(params["expire_days"]); days > 0 {
		expireTime = time.Now().AddDate(0, 0, cast.ToInt(days)).Unix()
	}

	operator := this.user(ctx)
	handleTime := time.Now().Unix()

	exist := (&model.UserCertification{}).GetByUid(uid)

	if exist == nil {
		record := model.UserCertification{
			Uid:         uid,
			CertType:    certType,
			Category:    category,
			Title:       title,
			Company:     truncateSafe(cast.ToString(data["company"]), certCompanyLen),
			Position:    truncateSafe(cast.ToString(data["position"]), certPositionLen),
			Contact:     truncateSafe(cast.ToString(data["contact"]), certContactLen),
			Images:      images,
			Data:        utils.Json.Encode(data),
			Status:      model.CertPassed,
			HandlerUid:  operator.Id,
			HandleTime:  handleTime,
			ExpireTime:  expireTime,
			Description: description,
		}
		this.certLegacyFill(&record, data)

		if _, err := facade.DB.Model(&model.UserCertification{}).Create(&record); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	} else {
		payload := map[string]any{
			"cert_type":   certType,
			"category":    category,
			"title":       title,
			"company":     truncateSafe(cast.ToString(data["company"]), certCompanyLen),
			"position":    truncateSafe(cast.ToString(data["position"]), certPositionLen),
			"contact":     truncateSafe(cast.ToString(data["contact"]), certContactLen),
			"images":      images,
			"data":        utils.Json.Encode(data),
			"description": description,
			"status":      model.CertPassed,
			// 覆盖旧记录时清掉历史驳回原因，否则个人中心会继续显示上一次的理由
			"reason":      "",
			"handler_uid": operator.Id,
			"handle_time": handleTime,
			"expire_time": expireTime,
		}

		if _, err := facade.DB.Model(&model.UserCertification{}).Where("uid", uid).Update(payload); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	}

	// 取一次记录 id 给站内信做关联（Create 的返回值在这里不保证带主键）
	id := 0
	if exist != nil {
		id = cast.ToInt(exist["id"])
	} else if fresh, _ := facade.DB.Model(&model.UserCertification{}).Where("uid", uid).Find(); !utils.Is.Empty(fresh) {
		id = cast.ToInt(cast.ToStringMap(fresh)["id"])
	}

	go this.delCache(uid)
	go this.notifyGranted(uid, id, title)

	this.json(ctx, gin.H{"id": id, "uid": uid, "status": model.CertPassed}, facade.Lang(ctx, "已授予认证！"), 200)
}

// notifyGranted - 主动授予的站内信
//
// 与审核通过的通知分开写：用户没有提交过申请，说「已通过审核」会让人莫名其妙。
func (this *UserCertification) notifyGranted(uid int, id int, title string) {
	if uid <= 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{"error": r}, "认证授予通知发送失败")
		}
	}()

	_, _ = (&model.Notification{}).CreateNotification(
		uid, 0, model.NotificationTypeSystem, "你已获得认证",
		"管理员已为你的账号授予「"+title+"」认证，现在会展示在你的个人主页。",
		"user-certification", id,
	)
}

// revoke - 收回认证（后台）
//
// 与驳回的区别：驳回是「这次申请没通过」，用户改资料可以重提；
// 收回是「已经发出的认证被撤销」（冒名、资质过期、机构注销等），
// 收回后主页不再展示标识，用户端看到的就是「未认证」，可重新申请。
//
// 为什么物理删除而不是软删除：uid 上有唯一索引 uk_cert_uid，
// 软删除的行仍留在表里，用户重新提交时再插入同一 uid 会撞唯一键。
// 想保留证据的话，用 edit 改资料 / handle 驳回，而不是 revoke。
func (this *UserCertification) revoke(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	id := cast.ToInt(params["id"])
	uid := cast.ToInt(params["uid"])
	if id <= 0 && uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id"), 400)
		return
	}

	query := facade.DB.Model(&model.UserCertification{})
	if id > 0 {
		query = query.Where("id", id)
	} else {
		query = query.Where("uid", uid)
	}

	item, _ := query.Find()
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "认证记录不存在！"), 404)
		return
	}
	record := cast.ToStringMap(item)

	recordId := cast.ToInt(record["id"])
	targetUid := cast.ToInt(record["uid"])
	title := cast.ToString(record["title"])

	// 收回原因会写进站内信，可空（内部清理不一定有对外说明）
	reason := truncateSafe(strings.TrimSpace(cast.ToString(params["reason"])), certReasonLen)

	// 物理删除：Unscoped 才能绕过 soft_delete，见上面的注释
	if err := facade.DB.Drive().Unscoped().Where("id = ?", recordId).Delete(&model.UserCertification{}).Error; err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 角标缓存必须同步清掉：收回后主页不能还挂着盾牌（异步的话响应先回、页面后刷新会看到旧盾牌）
	this.delCache(targetUid)

	go this.notifyRevoked(targetUid, recordId, title, reason)

	this.json(ctx, gin.H{"id": recordId, "uid": targetUid}, facade.Lang(ctx, "已收回认证！"), 200)
}

// notifyRevoked - 收回认证的站内信
func (this *UserCertification) notifyRevoked(uid int, id int, title string, reason string) {
	if uid <= 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{"error": r}, "认证收回通知发送失败")
		}
	}()

	content := "「" + title + "」认证已被收回，主页不再展示认证标识。如资料有变可重新申请。"
	if reason != "" {
		content = "「" + title + "」认证已被收回：" + reason + "。如资料有变可重新申请。"
	}

	_, _ = (&model.Notification{}).CreateNotification(
		uid, 0, model.NotificationTypeSystem, "你的认证已被收回", content,
		"user-certification", id,
	)
}

// edit - 后台修改认证资料（不改变审核状态）
//
// 与 handle 的职责分开：这里只改「用户提交的资料」，通过 / 驳回、有效期、认证说明仍走 handle。
// 语义：
//   - 传了 values（哪怕空对象）→ 按该记录的认证类型配置整体重写字段值，缺失项视为清空；
//   - 不传 values → 字段值保持不变，只处理 description（如果传了）；
//   - 必填项不做强制校验（运营需要能清空某项），但格式校验照旧：类型 / 选项范围 / 长度 / 链接协议。
func (this *UserCertification) edit(ctx *gin.Context) {

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

	item, _ := facade.DB.Model(&model.UserCertification{}).Find(id)
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "认证记录不存在！"), 404)
		return
	}
	record := cast.ToStringMap(item)

	certType := cast.ToInt(record["cert_type"])
	config := model.CertTypeFind(certType)
	if config == nil {
		this.json(ctx, nil, facade.Lang(ctx, "该认证类型的配置已删除，无法编辑资料！"), 400)
		return
	}

	fields, _ := config["fields"].([]model.CertField)
	categories, _ := config["categories"].([]string)
	if len(fields) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "该认证类型没有配置任何申请字段！"), 400)
		return
	}

	payload := map[string]any{}
	// newData 为 nil 表示本次没有提交字段值，只改说明
	var newData map[string]any
	newTitle := cast.ToString(record["title"])

	// 传了 values 才重写字段值：不传表示「只改说明」，避免误把整份资料清空
	if rawValues, ok := params["values"]; ok {
		values := cast.ToStringMap(rawValues)

		data, images, title, category, msg := this.certValues(fields, categories, values, false)
		if msg != "" {
			this.json(ctx, nil, facade.Lang(ctx, msg), 400)
			return
		}

		title = truncateSafe(title, certTitleLen)
		category = truncateSafe(category, certCategoryLen)
		if title == "" {
			// 与提交口径一致：认证名称为空时回落类型名，主页标识不会因此消失
			title = truncateSafe(cast.ToString(config["name"]), certTitleLen)
		}

		newData = data
		newTitle = title

		payload["title"] = title
		payload["category"] = category
		payload["images"] = images
		payload["data"] = utils.Json.Encode(data)
		payload["company"] = truncateSafe(cast.ToString(data["company"]), certCompanyLen)
		payload["position"] = truncateSafe(cast.ToString(data["position"]), certPositionLen)
		payload["contact"] = truncateSafe(cast.ToString(data["contact"]), certContactLen)
	}

	// 认证说明同样是后台口径，可在编辑里顺手改（不传则保留原值）
	oldDesc := cast.ToString(record["description"])
	newDesc := oldDesc
	if _, ok := params["description"]; ok {
		newDesc = truncateSafe(strings.TrimSpace(cast.ToString(params["description"])), certDescLen)
		payload["description"] = newDesc
	}

	if len(payload) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "没有需要修改的内容！"), 400)
		return
	}

	// 先算清楚「哪些项真的变了」，用于给用户的提醒文案：
	// 原样保存（运营点开看一眼又提交）不应该产生一条「你的资料已更新」的噪音通知。
	changes := this.certChanges(fields, record, newData, oldDesc, newDesc)

	if _, err := facade.DB.Model(&model.UserCertification{}).Where("id", id).Update(payload); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache(cast.ToInt(record["uid"]))
	// 用户需要知道资料被改过（尤其是被人冒名/误改时），只异步发，失败不影响保存结果
	go this.notifyEdited(cast.ToInt(record["uid"]), id, newTitle, changes)

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "已保存！"), 200)
}

// certChanges - 对比新旧字段值，返回发生变化的字段名称（中文 label，用于提醒文案）
//
// 逐字段比对展示文本而不是原始值：图片存储的是「逗号串 vs 数组」、
// 多选存的是「数组 vs 逗号串」，直接比 raw 会误报「改了」。
// newData 为 nil 时（只改说明）跳过字段比对。
func (this *UserCertification) certChanges(fields []model.CertField, record map[string]any, newData map[string]any, oldDesc, newDesc string) []string {

	changes := make([]string, 0)

	if newData != nil {
		oldData := model.CertDataMap(record["data"])

		for _, field := range fields {
			oldValue := oldData[field.Key]
			if utils.Is.Empty(oldValue) && model.CertLegacyColumn(field.Key) {
				oldValue = record[field.Key]
			}

			if model.CertValueText(field, oldValue) == model.CertValueText(field, newData[field.Key]) {
				continue
			}
			changes = append(changes, field.Label)
		}
	}

	if oldDesc != newDesc {
		changes = append(changes, "认证说明")
	}

	return changes
}

// notifyEdited - 资料被后台修改后的站内提醒
//
// 只列字段名称、不复述字段值：
//   - 值里可能有仅后台可见的内容（联系方式等），通知正文会被前台拉取；
//   - 用户自己知道自己填了什么，只需要知道「哪几项被改」。
func (this *UserCertification) notifyEdited(uid int, id int, title string, changes []string) {

	if uid <= 0 || len(changes) == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{"error": r}, "认证资料更新通知发送失败")
		}
	}()

	// 正文上限 1024，字段太多时只报前 6 项 + 计数
	list := changes
	if len(list) > 6 {
		list = list[:6]
	}
	detail := strings.Join(list, "、")
	if len(changes) > len(list) {
		detail = detail + " 等 " + cast.ToString(len(changes)) + " 项"
	}

	content := "你的「" + title + "」认证资料已被管理员更新：" + detail + "。如信息有误，可修改资料后重新提交。"
	if utils.Is.Empty(title) {
		content = "你的认证资料已被管理员更新：" + detail + "。如信息有误，可修改资料后重新提交。"
	}

	_, _ = (&model.Notification{}).CreateNotification(
		uid, 0, model.NotificationTypeSystem, "你的认证资料已更新", content,
		"user-certification", id,
	)
}

// notify - 审核结果站内信（异步，失败不影响审核结果）
func (this *UserCertification) notify(uid int, id int, title string, status int, reason string) {
	if uid <= 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{"error": r}, "认证审核通知发送失败")
		}
	}()

	noticeTitle := "你的认证已通过"
	content := "「" + title + "」认证已通过，现在会展示在你的个人主页。"
	if status == model.CertRejected {
		noticeTitle = "你的认证未通过"
		content = "「" + title + "」认证未通过：" + reason + "。可修改资料后重新提交。"
	}

	_, _ = (&model.Notification{}).CreateNotification(
		uid, 0, model.NotificationTypeSystem, noticeTitle, content,
		"user-certification", id,
	)
}

// certSchemas - 单次请求内的类型配置快照
//
// 列表每行都要读字段定义与类型文案，逐行走 CertTypeFind 就是 N+1 查询；
// 这里一次性把类型表读进内存，行渲染只查 map。
type certSchemas struct {
	fields map[int][]model.CertField
	names  map[int]string
	colors map[int]string
	list   []map[string]any
}

func (this *UserCertification) schemas() *certSchemas {
	result := &certSchemas{
		fields: make(map[int][]model.CertField),
		names:  make(map[int]string),
		colors: make(map[int]string),
	}

	result.list = model.CertTypeList(false)
	for _, item := range result.list {
		id := cast.ToInt(item["id"])
		if fields, ok := item["fields"].([]model.CertField); ok {
			result.fields[id] = fields
		}
		if name := cast.ToString(item["name"]); name != "" {
			result.names[id] = name
		}
		if color := model.NormalizeCertColor(item["color"]); color != "" {
			result.colors[id] = color
		}
	}

	return result
}

func (this *certSchemas) fieldsOf(certType int) []model.CertField {
	return this.fields[certType]
}

// nameOf - 类型文案（类型已删除时回退到内置枚举，历史记录不至于显示空白）
func (this *certSchemas) nameOf(certType int) string {
	if name := this.names[certType]; name != "" {
		return name
	}
	return model.CertTypeText(certType)
}

// colorOf - 类型配色（空串交给前端按内置兜底色处理）
func (this *certSchemas) colorOf(certType int) string {
	return this.colors[certType]
}

// options - 类型筛选项（value/label/enable），随列表一起返回给后台
func (this *certSchemas) options() []map[string]any {
	list := make([]map[string]any, 0, len(this.list))
	for _, item := range this.list {
		list = append(list, map[string]any{
			"value":  cast.ToInt(item["id"]),
			"label":  cast.ToString(item["name"]),
			"enable": cast.ToInt(item["enable"]),
		})
	}
	return list
}

// certOutput - 认证记录输出（admin=true 时带审核信息与申请人资料）
//
// fields 是按当前类型配置顺序排好的字段值，前端不必再自己对 key：
//
//	[{key,label,type,value,text,required,public,admin_only}]
//
// 其中 value 是原始值（图片为 URL 数组），text 是可直接展示的文本。
func (this *UserCertification) certOutput(item map[string]any, admin bool, schemas *certSchemas) map[string]any {

	if utils.Is.Empty(item) {
		return map[string]any{}
	}

	if schemas == nil {
		schemas = this.schemas()
	}

	certType := cast.ToInt(item["cert_type"])
	data := model.CertDataMap(item["data"])

	rows := make([]map[string]any, 0)
	for _, field := range schemas.fieldsOf(certType) {
		value := data[field.Key]
		if utils.Is.Empty(value) && model.CertLegacyColumn(field.Key) {
			value = item[field.Key]
		}
		rows = append(rows, map[string]any{
			"key":        field.Key,
			"label":      field.Label,
			"type":       field.Type,
			"value":      value,
			"text":       model.CertValueText(field, value),
			"required":   field.Required,
			"public":     field.Public,
			"admin_only": field.AdminOnly,
		})
	}

	out := map[string]any{
		"id":             cast.ToInt(item["id"]),
		"uid":            cast.ToInt(item["uid"]),
		"cert_type":      certType,
		"cert_type_text": schemas.nameOf(certType),
		// 与公开摘要同源的配色：本人页 / 后台 与主页昵称旁的盾牌保持一色
		"color":       schemas.colorOf(certType),
		"category":    cast.ToString(item["category"]),
		"title":       cast.ToString(item["title"]),
		"company":     cast.ToString(item["company"]),
		"position":    cast.ToString(item["position"]),
		"description": cast.ToString(item["description"]),
		"contact":     cast.ToString(item["contact"]),
		"images":      this.certImageList(cast.ToString(item["images"])),
		"status":      cast.ToInt(item["status"]),
		"status_text": model.CertStatusText(cast.ToInt(item["status"])),
		"reason":      cast.ToString(item["reason"]),
		"expire_time": cast.ToInt64(item["expire_time"]),
		"handle_time": cast.ToInt64(item["handle_time"]),
		"create_time": cast.ToInt64(item["create_time"]),
		"update_time": cast.ToInt64(item["update_time"]),
		"data":        data,
		"fields":      rows,
	}

	if admin {
		out["handler_uid"] = cast.ToInt(item["handler_uid"])
		out["applicant"] = this.certUserBrief(cast.ToInt(item["uid"]))
	}

	return out
}

// certValues - 按类型配置校验并清洗字段值
//
// 返回：data（存 data JSON 的完整键值）/ images（证明材料，逗号分隔写回独立列）/
// title（认证名称）/ category（认证领域）/ msg（校验失败提示，空串表示通过）。
// 校验规则全部来自字段定义，所以后台加一个必填项，后端自动生效，不需要改代码。
//
// enforceRequired：用户自己提交时为 true（必填项必须填）；
// 后台改资料时为 false —— 运营可能需要清空某一项（例如机构改名后旧字段作废），
// 但类型 / 选项范围 / 长度 / 链接协议这些格式校验照旧执行。
func (this *UserCertification) certValues(fields []model.CertField, categories []string, values map[string]any, enforceRequired bool) (data map[string]any, images, title, category, msg string) {

	data = make(map[string]any, len(fields))
	imageList := make([]string, 0)

	required := func(field model.CertField) string {
		return "请填写「" + field.Label + "」！"
	}

	needRequired := func(field model.CertField) bool {
		return field.Required && enforceRequired
	}

	for _, field := range fields {

		raw := values[field.Key]

		switch field.Type {

		case model.CertFieldImage:
			list := this.certImageField(raw, field.MaxCount)
			if len(list) == 0 {
				if needRequired(field) {
					return nil, "", "", "", "请上传「" + field.Label + "」！"
				}
				continue
			}
			data[field.Key] = list
			if field.Key == model.CertKeyImages {
				imageList = list
			}

		case model.CertFieldSelect:
			options := model.CertFieldOptions(field, categories)
			if field.Multiple {
				list := this.certSelectList(raw, options)
				if len(list) == 0 {
					if needRequired(field) {
						return nil, "", "", "", "请选择「" + field.Label + "」！"
					}
					continue
				}
				data[field.Key] = list
				continue
			}

			value := strings.TrimSpace(cast.ToString(raw))
			if value == "" {
				if needRequired(field) {
					return nil, "", "", "", "请选择「" + field.Label + "」！"
				}
				continue
			}
			if len(options) > 0 && !utils.InArray(value, options) {
				return nil, "", "", "", "「" + field.Label + "」的取值不在候选范围内！"
			}
			data[field.Key] = value

		case model.CertFieldNumber:
			text := strings.TrimSpace(cast.ToString(raw))
			if text == "" {
				if needRequired(field) {
					return nil, "", "", "", required(field)
				}
				continue
			}
			number, err := strconv.ParseFloat(text, 64)
			if err != nil {
				return nil, "", "", "", "「" + field.Label + "」需要填写数字！"
			}
			data[field.Key] = number

		case model.CertFieldLink:
			text := truncateSafe(strings.TrimSpace(cast.ToString(raw)), model.CertFieldMax(field))
			if text == "" {
				if needRequired(field) {
					return nil, "", "", "", required(field)
				}
				continue
			}
			text = strings.TrimSpace(facade.Comm.SanitizeURL(text))
			if !strings.HasPrefix(text, "http://") && !strings.HasPrefix(text, "https://") {
				return nil, "", "", "", "「" + field.Label + "」需要填写 http(s) 链接！"
			}
			data[field.Key] = text

		case model.CertFieldDate:
			text := truncateSafe(strings.TrimSpace(cast.ToString(raw)), 10)
			if text == "" {
				if needRequired(field) {
					return nil, "", "", "", required(field)
				}
				continue
			}
			if _, err := time.Parse("2006-01-02", text); err != nil {
				return nil, "", "", "", "「" + field.Label + "」的日期格式应为 YYYY-MM-DD！"
			}
			data[field.Key] = text

		default: // text / textarea
			text := truncateSafe(strings.TrimSpace(cast.ToString(raw)), model.CertFieldMax(field))
			if text == "" {
				if needRequired(field) {
					return nil, "", "", "", required(field)
				}
				continue
			}
			data[field.Key] = text
		}
	}

	return data,
		strings.Join(imageList, ","),
		cast.ToString(data[model.CertKeyTitle]),
		cast.ToString(data[model.CertKeyCategory]),
		""
}

// certSelectList - 多选下拉：数组 / 逗号串都收，逐项校验候选范围
func (this *UserCertification) certSelectList(raw any, options []string) []string {

	list := make([]string, 0)

	switch value := raw.(type) {
	case []any:
		list = cast.ToStringSlice(value)
	case []string:
		list = value
	default:
		text := strings.TrimSpace(cast.ToString(raw))
		if text == "" {
			return []string{}
		}
		list = strings.Split(text, ",")
	}

	result := make([]string, 0, len(list))
	seen := make(map[string]bool, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		if len(options) > 0 && !utils.InArray(item, options) {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}

	return result
}

// certLegacyFill - 老列兼容：字段里恰好有 company / position / contact 的，顺手写进独立列
//
// 这样升级前就有的后台页面 / 导出脚本读到这些列时不会全是空值；
// 判断口径以字段定义为准，后台删掉这些字段后自然不再写。
func (this *UserCertification) certLegacyFill(record *model.UserCertification, data map[string]any) {
	if record == nil {
		return
	}

	record.Company = truncateSafe(cast.ToString(data["company"]), certCompanyLen)
	record.Position = truncateSafe(cast.ToString(data["position"]), certPositionLen)
	record.Contact = truncateSafe(cast.ToString(data["contact"]), certContactLen)
	// 认证说明只由后台填写，用户提交时保持为空
	record.Description = ""
}

// certUserBrief - 申请人公开信息（白名单 + 隐私清洗）
func (this *UserCertification) certUserBrief(uid int) map[string]any {

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

// certImageField - 解析并清洗图片字段（证明材料 / 后台配置的其它图片项）
//
// 兼容前端传数组与逗号分隔字符串；逐条走 SanitizeURL，且必须仍是 http(s)
// （挡掉 javascript: / data: 这类伪协议），超出该字段配置的张数上限直接丢弃。
func (this *UserCertification) certImageField(value any, max int) []string {

	if max <= 0 {
		max = certMaxImages
	}
	if max > 12 {
		max = 12
	}

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

	clean := make([]string, 0, max)

	for _, url := range raw {

		url = strings.TrimSpace(url)
		if utils.Is.Empty(url) || len([]rune(url)) > certImageUrlLen {
			continue
		}

		url = strings.TrimSpace(facade.Comm.SanitizeURL(url))
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}

		clean = append(clean, url)
		if len(clean) >= max {
			break
		}
	}

	return clean
}

// certImageList - 逗号分隔串 → 数组（空值返回空数组，前端不必判空）
func (this *UserCertification) certImageList(value string) []string {

	value = strings.TrimSpace(value)
	if utils.Is.Empty(value) {
		return []string{}
	}

	list := make([]string, 0, certMaxImages)
	for _, url := range strings.Split(value, ",") {
		url = strings.TrimSpace(url)
		if utils.Is.Empty(url) {
			continue
		}
		list = append(list, url)
	}

	return list
}

// certPage - 页码钳制
func (this *UserCertification) certPage(value any) int {
	page := cast.ToInt(value)
	if page <= 0 {
		return 1
	}
	if page > certMaxPage {
		return certMaxPage
	}
	return page
}

// certLimit - 单页条数钳制
func (this *UserCertification) certLimit(value any) int {
	limit := cast.ToInt(value)
	if limit <= 0 {
		return certDefaultLimit
	}
	if limit > 50 {
		return 50
	}
	return limit
}

// delCache - 清认证相关缓存
//
// 除了接口级缓存（[GET] 前缀模糊匹配），还可以带上受影响用户的 uid：
// 头像角标摘要是按 uid 缓存 60 秒的（见 model.CertBadgeSummary），
// 认证一旦变化必须让那份缓存立刻失效，否则主页会短暂显示旧状态。
func (this *UserCertification) delCache(uids ...int) {
	facade.Cache.DelTags([]any{"[GET]", "user-certification"})
	for _, uid := range uids {
		model.CertBadgeFlush(uid)
	}
}
