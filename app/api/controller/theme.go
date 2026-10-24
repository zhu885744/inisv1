package controller

import (
	"inis/app/facade"
	"inis/app/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// Theme - 主题配色（后台）
//
// 只提供两个动作：
//   - GET  /api/theme/config：token 表 + 内置预设 + 当前配色 + 当前 CSS（公开，前端首屏也可读）
//   - POST /api/theme/save：保存配色，保存后 /theme-custom.css 立即变为新值
//   - POST /api/theme/reset：恢复默认预设
type Theme struct {
	base
}

func (this *Theme) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"config": this.config,
		"active": this.active,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Theme) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"save":  this.save,
		"reset": this.reset,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Theme) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 405)
}

func (this *Theme) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 405)
}

func (this *Theme) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// config - 主题配色配置
func (this *Theme) config(ctx *gin.Context) {
	this.json(ctx, model.ThemePayload(), facade.Lang(ctx, "主题配色配置"), 200)
}

// active - 当前生效的样式表
//
// 前端用它把 CSS 注入 <style id="theme-runtime">，不再依赖 /theme-custom.css 这个
// 静态资源 URL：CDN / 反代对 .css 的强缓存会让主题保存后长期不生效，而接口响应
// 可以明确要求不缓存（no-store），每次回源拿最新值。
func (this *Theme) active(ctx *gin.Context) {
	css, _ := model.ThemeCSS()

	// 主题随时可能被保存修改，必须回源
	ctx.Header("Cache-Control", "no-store")

	this.json(ctx, facade.H{
		"version": model.ThemeVersion(),
		"css":     css,
	}, facade.Lang(ctx, "主题样式表"), 200)
}

// save - 保存主题配色
func (this *Theme) save(ctx *gin.Context) {
	params := this.params(ctx)

	settings := model.ThemeSettings{
		Preset: cast.ToString(params["preset"]),
		Light:  cast.ToStringMapString(params["light"]),
		Dark:   cast.ToStringMapString(params["dark"]),
		Extras: cast.ToStringMapString(params["extras"]),
	}

	// 前后端约定每次提交完整的浅色 / 深色取值；两个都空说明请求体没带颜色，
	// 此时若继续保存会静默落回预设值，直接报错更容易定位。
	if len(settings.Light) == 0 && len(settings.Dark) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "没有收到配色数据，请刷新页面后重试"), 400)
		return
	}

	result, err := model.SaveTheme(settings)
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	css, _ := model.ThemeCSS()

	// 记录收到的数据量：若日志里是 0，说明请求体没带颜色（前端问题），
	// 而不是后端把自定义值丢弃了，排查时能一眼分清。
	facade.Log.Info(map[string]any{
		"preset":     result.Preset,
		"light_keys": len(settings.Light),
		"dark_keys":  len(settings.Dark),
	}, "主题配色已保存")

	this.json(ctx, facade.H{
		"settings": result,
		"css":      css,
	}, facade.Lang(ctx, "保存成功！"), 200)
}

// reset - 恢复默认配色
func (this *Theme) reset(ctx *gin.Context) {
	result, err := model.ResetTheme()
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	css, _ := model.ThemeCSS()

	this.json(ctx, facade.H{
		"settings": result,
		"css":      css,
	}, facade.Lang(ctx, "已恢复默认配色！"), 200)
}
