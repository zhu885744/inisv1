package controller

import (
	"context"
	"fmt"

	// JWT "github.com/dgrijalva/jwt-go"
	"inis/app/facade"
	"mime/multipart"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	"github.com/google/uuid"
)

// Test - 测试控制器
// @Summary 测试API控制器
// @Description 提供各种测试功能的API接口
// @Tags Test
type Test struct {
	// 继承
	base
}

// @Summary 获取测试数据
// @Description 根据不同方法获取测试相关数据
// @Tags Test
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(request, alipay)
// @Param id query int false "ID"
// @Param where query string false "查询条件"
// @Param or query string false "或条件"
// @Param like query string false "模糊查询"
// @Param cache query string false "是否使用缓存"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/test/{method} [get]
// IGET - GET请求本体
func (this *Test) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"request": this.request,
		"alipay":  this.alipay,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// @Summary 提交测试数据
// @Description 根据不同方法提交测试相关数据
// @Tags Test
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(return-url, notify-url, request, upload)
// @Param id query int false "ID"
// @Param where query string false "查询条件"
// @Param or query string false "或条件"
// @Param like query string false "模糊查询"
// @Param file formData file false "上传文件"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/test/{method} [post]
// IPOST - POST请求本体
func (this *Test) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"return-url": this.returnUrl,
		"notify-url": this.notifyUrl,
		"request":    this.request,
		"upload":     this.upload,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// @Summary 更新测试数据
// @Description 根据不同方法更新测试相关数据
// @Tags Test
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(request)
// @Param id query int false "ID"
// @Param where query string false "查询条件"
// @Param or query string false "或条件"
// @Param like query string false "模糊查询"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/test/{method} [put]
// IPUT - PUT请求本体
func (this *Test) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"request": this.request,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// @Summary 删除测试数据
// @Description 根据不同方法删除测试相关数据
// @Tags Test
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(request)
// @Param id query int false "ID"
// @Param where query string false "查询条件"
// @Param or query string false "或条件"
// @Param like query string false "模糊查询"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/test/{method} [delete]
// IDEL - DELETE请求本体
func (this *Test) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"request": this.request,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// @Summary 测试首页
// @Description 测试控制器首页接口
// @Tags Test
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功响应"
// @Router /api/test [get]
// INDEX - GET请求本体
func (this *Test) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "好的！"), 200)
}

// INDEX - GET请求本体
func (this *Test) upload(ctx *gin.Context) {

	params := this.params(ctx)

	// 上传文件
	file, err := ctx.FormFile("file")
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	// 文件数据
	bytes, err := file.Open()
	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}
	defer func(bytes multipart.File) {
		err := bytes.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(bytes)

	// 文件后缀
	suffix := file.Filename[strings.LastIndex(file.Filename, "."):]
	params["suffix"] = suffix

	item := facade.Storage.Upload(facade.Storage.Path()+suffix, bytes)
	if item.Error != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	params["item"] = item

	fmt.Println("url: ", item.Domain+item.Path)

	this.json(ctx, params, facade.Lang(ctx, "好的！"), 200)
}

func (this *Test) alipay(ctx *gin.Context) {

	// 初始化 BodyMap
	body := make(gopay.BodyMap)
	body.Set("subject", "统一收单下单并支付页面接口")
	body.Set("out_trade_no", uuid.New().String())
	body.Set("total_amount", "0.01")
	body.Set("product_code", "FAST_INSTANT_TRADE_PAY")

	payUrl, err := facade.Alipay().TradePagePay(context.Background(), body)
	if err != nil {
		if bizErr, ok := alipay.IsBizError(err); ok {
			fmt.Println(bizErr)
			return
		}
		fmt.Println(err)
		return
	}

	fmt.Println(payUrl)

	this.json(ctx, payUrl, "数据请求成功！", 200)
}

func (this *Test) returnUrl(ctx *gin.Context) {

	params := this.params(ctx)

	fmt.Println("==================== returnUrl：", params)
}

func (this *Test) notifyUrl(ctx *gin.Context) {

	params := this.params(ctx)

	fmt.Println("==================== notifyUrl：", params)
}

// 测试网络请求
func (this *Test) request(ctx *gin.Context) {

	params := this.params(ctx)

	this.json(ctx, map[string]any{
		"method":  ctx.Request.Method,
		"params":  params,
		"headers": this.headers(ctx),
	}, facade.Lang(ctx, "数据请求成功！"), 200)
}
