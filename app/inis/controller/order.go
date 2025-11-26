package controller

import (
	"inis/app/facade"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

type Order struct {
	// 继承
	base
}

// IGET - GET请求本体
func (this *Order) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"theme":  this.theme,
		"themes": this.themes,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - POST请求本体
func (this *Order) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - PUT请求本体
func (this *Order) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - DELETE请求本体
func (this *Order) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - GET请求本体
func (this *Order) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "好的！"), 200)
}

// theme - 查询已购的指定主题
func (this *Order) theme(ctx *gin.Context) {

	// 获取请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["key"]) {
		this.json(ctx, nil, facade.Lang(ctx, "唯一识别码 key 不能为空！"), 400)
		return
	}

	// 返回简单的授权成功格式
	fakeData := map[string]interface{}{
		"id":     1,
		"is_buy": true,
		"name":   params["key"],
		"status": 1,
	}

	this.json(ctx, fakeData, facade.Lang(ctx, "已授权"), 200)
}

// themes - 查询已购的全部主题
func (this *Order) themes(ctx *gin.Context) {

	// 返回简单的授权成功格式
	fakeData := map[string]interface{}{
		"id":     1,
		"is_buy": true,
		"name":   "jue",
		"status": 1,
	}

	this.json(ctx, fakeData, facade.Lang(ctx, "已授权"), 200)
}
