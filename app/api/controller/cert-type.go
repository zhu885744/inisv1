package controller

import (
	"math"
	"strings"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * CertType - 认证类型配置（后台可自定义）
 *
 * 前台：
 *   GET cert-type/enable  启用中的类型（含领域候选 / 字段定义 / 认证说明），申请页据此动态渲染表单
 *
 * 后台（需要权限点，判定见 meta.permit）：
 *   GET    cert-type/all     全部类型（含停用的）
 *   POST   cert-type/create  新建类型
 *   PUT    cert-type/update  修改类型（名称 / 图标 / 说明 / 领域 / 字段 / 排序 / 启停）
 *   DELETE cert-type/remove  删除类型（类型下仍有认证记录时必须带 force=1）
 *
 * 设计口径：
 *   1. 「认证说明」由后台填写，只读展示给申请者；用户提交的任何说明都不会进这个字段；
 *   2. 字段定义完全由后台决定（类型 / 必填 / 选项 / 是否公开 / 是否仅后台可见），
 *      后端提交时按同一份定义校验，前端不需要维护第二套规则；
 *   3. 删除是软删除：历史认证记录的 cert_type 仍指向原 id，文案回退到内置枚举。
 */
type CertType struct {
	base
}

const (
	// certTypeNameLen - 类型名长度上限
	certTypeNameLen = 32
	// certTypeIntroLen - 简介长度上限
	certTypeIntroLen = 255
	// certTypeNoticeLen - 认证说明长度上限
	certTypeNoticeLen = 2000
)

func (this *CertType) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"all":    this.all,
		"enable": this.enable,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *CertType) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"create": this.create,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *CertType) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update": this.update,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *CertType) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"remove": this.remove,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *CertType) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// enable - 启用中的认证类型（公开：申请页渲染动态表单）
//
// 每个类型额外附带申请条件的判定结果：
//   - met      当前用户是否达标（无门槛 / 未登录恒为 true）
//   - missing  未达标项 [{key,label,need,current,short}]，前端直接渲染差额
//   - stats    当前用户的统计值 {fans,articles,moments,days}，未登录为空对象
//
// 未登录不做判定（met=true）：是否登录由申请时的 401 拦截，避免公开接口
// 把「未登录」显示成「条件不达标」，让用户误以为自己没资格。
func (this *CertType) enable(ctx *gin.Context) {

	list := model.CertTypeList(true)

	uid := this.user(ctx).Id
	stats := map[string]int{}
	if uid > 0 {
		stats = model.CertUserStats(uid)
	}

	for _, item := range list {
		if uid <= 0 {
			item["met"] = true
			item["missing"] = []map[string]any{}
			item["stats"] = stats
			continue
		}

		conditions, _ := item["conditions"].(model.CertConditions)
		met, missing := conditions.Check(stats)
		item["met"] = met
		item["missing"] = missing
		item["stats"] = stats
	}

	this.json(ctx, list, facade.Lang(ctx, "查询成功！"), 200)
}

// all - 全部认证类型（后台）
func (this *CertType) all(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx, map[string]any{"page": 1, "limit": 100})

	page := cast.ToInt(params["page"])
	if page <= 0 {
		page = 1
	}
	limit := cast.ToInt(params["limit"])
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	query := facade.DB.Model(&[]model.CertType{})
	count, _ := query.Count()
	items, _ := query.Order("sort asc, id asc").Limit(limit).Page(page).Select()

	list := make([]map[string]any, 0, len(items))
	for _, item := range items {
		row := model.CertTypeOutput(item)
		row["used"] = this.usedCount(cast.ToInt(item["id"]))
		list = append(list, row)
	}

	this.json(ctx, gin.H{
		"data":  list,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// create - 新建认证类型
func (this *CertType) create(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)

	item, msg := this.fill(params, nil)
	if msg != "" {
		this.json(ctx, nil, facade.Lang(ctx, msg), 400)
		return
	}

	_, err := facade.DB.Model(&model.CertType{}).Create(&item)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{"id": item.Id}, facade.Lang(ctx, "创建成功！"), 200)
}

// update - 修改认证类型
func (this *CertType) update(ctx *gin.Context) {

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

	exist := model.CertTypeFind(id)
	if exist == nil {
		this.json(ctx, nil, facade.Lang(ctx, "认证类型不存在！"), 404)
		return
	}

	item, msg := this.fill(params, exist)
	if msg != "" {
		this.json(ctx, nil, facade.Lang(ctx, msg), 400)
		return
	}

	payload := map[string]any{
		"name":       item.Name,
		"icon":       item.Icon,
		"color":      item.Color,
		"intro":      item.Intro,
		"notice":     item.Notice,
		"categories": item.Categories,
		"fields":     item.Fields,
		"conditions": item.Conditions,
		"sort":       item.Sort,
		"enable":     item.Enable,
	}

	if _, err := facade.DB.Model(&model.CertType{}).Where("id", id).Update(payload); err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{"id": id}, facade.Lang(ctx, "更新成功！"), 200)
}

// remove - 删除认证类型（软删除；类型下还有认证记录时需 force=1）
func (this *CertType) remove(ctx *gin.Context) {

	if !this.meta.permit(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：当前账号未被授予该权限点！"), 403)
		return
	}

	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])
	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	// 保护：类型被删后，历史记录的类型文案会变成「未知类型」，先让运营确认
	if !cast.ToBool(params["force"]) {
		for _, id := range ids {
			if used := this.usedCount(cast.ToInt(id)); used > 0 {
				this.json(ctx, nil, facade.Lang(ctx, "该类型下还有 %d 条认证记录，删除后这些记录将显示「%s」；确认删除请带 force=1", used, "未知类型"), 400)
				return
			}
		}
	}

	if _, err := facade.DB.Model(&model.CertType{}).Delete(ids); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	go this.delCache()

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

// fill - 组装 / 校验类型字段
//
// exist 不为 nil 时是「局部更新」：入参里没带的键沿用原值，
// 这样后台只改一个开关（如停用）不必把整份字段定义再传一次。
func (this *CertType) fill(params map[string]any, exist map[string]any) (model.CertType, string) {

	item := model.CertType{
		Name:   truncateSafe(strings.TrimSpace(cast.ToString(params["name"])), certTypeNameLen),
		Icon:   truncateSafe(strings.TrimSpace(cast.ToString(params["icon"])), 64),
		Color:  model.NormalizeCertColor(params["color"]),
		Intro:  truncateSafe(strings.TrimSpace(cast.ToString(params["intro"])), certTypeIntroLen),
		Notice: truncateSafe(strings.TrimSpace(cast.ToString(params["notice"])), certTypeNoticeLen),
		Sort:   cast.ToInt(params["sort"]),
		Enable: 1,
	}

	if exist != nil {
		item.Name = truncateSafe(utils.Ternary[string](utils.Is.Empty(item.Name), cast.ToString(exist["name"]), item.Name), certTypeNameLen)
		item.Icon = utils.Ternary[string](utils.Is.Empty(item.Icon), cast.ToString(exist["icon"]), item.Icon)
		// 颜色与图标不同：显式传空串表示「恢复默认」，不传才沿用原值，
		// 否则管理员把自定义色删掉后没法回到内置兜底色
		if _, ok := params["color"]; ok {
			item.Color = model.NormalizeCertColor(params["color"])
		} else {
			item.Color = cast.ToString(exist["color"])
		}
		item.Intro = utils.Ternary[string](utils.Is.Empty(item.Intro), cast.ToString(exist["intro"]), item.Intro)
		item.Notice = utils.Ternary[string](utils.Is.Empty(item.Notice), cast.ToString(exist["notice"]), item.Notice)
		if _, ok := params["sort"]; !ok {
			item.Sort = cast.ToInt(exist["sort"])
		}
		if _, ok := params["enable"]; ok {
			item.Enable = cast.ToInt(params["enable"])
		} else {
			item.Enable = cast.ToInt(exist["enable"])
		}
	} else if _, ok := params["enable"]; ok {
		item.Enable = cast.ToInt(params["enable"])
	}

	if item.Name == "" {
		return item, "请填写认证类型名称！"
	}
	if item.Enable != 0 {
		item.Enable = 1
	}

	// 领域
	if _, ok := params["categories"]; ok {
		item.Categories = utils.Json.Encode(model.CertCategoriesFromAny(params["categories"]))
	} else if exist != nil {
		item.Categories = utils.Json.Encode(model.CertCategoriesFromAny(exist["categories"]))
	} else {
		item.Categories = "[]"
	}

	// 字段定义
	if _, ok := params["fields"]; ok {
		fields := model.CertFieldsFromAny(params["fields"])
		if len(fields) == 0 {
			return item, "至少配置一个需要用户填写的字段！"
		}
		item.Fields = utils.Json.Encode(fields)
	} else if exist != nil {
		item.Fields = utils.Json.Encode(model.CertFieldsFromAny(exist["fields"]))
	} else {
		return item, "至少配置一个需要用户填写的字段！"
	}

	if model.CertFieldsFromAny(item.Fields) == nil {
		return item, "字段配置格式不正确（需要 JSON 数组）！"
	}

	// 申请条件：粉丝数 / 文章数 / 动态数 / 注册天数，0 表示不限
	if _, ok := params["conditions"]; ok {
		item.Conditions = utils.Json.Encode(model.CertConditionsFromAny(params["conditions"]))
	} else if exist != nil {
		item.Conditions = utils.Json.Encode(model.CertConditionsFromAny(exist["conditions"]))
	} else {
		item.Conditions = utils.Json.Encode(model.CertConditions{})
	}

	return item, ""
}

// usedCount - 该类型下的认证记录数（删除保护 / 后台展示用）
func (this *CertType) usedCount(id int) int64 {
	if id <= 0 {
		return 0
	}
	count, _ := facade.DB.Model(&model.UserCertification{}).Where("cert_type", id).Count()
	return count
}

// delCache - 清认证相关缓存（类型改动会影响申请页与后台筛选项）
func (this *CertType) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "cert-type"})
	facade.Cache.DelTags([]any{"[GET]", "user-certification"})
	// 门槛调整后要立刻反映到用户的条件进度上，否则最多 3 分钟内还显示旧的 met/missing
	model.CertStatsFlushAll()
}
