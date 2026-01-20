package facade

import (
	"errors"
	"fmt"
	"strings"

	AliYunClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	AliYunUtil "github.com/alibabacloud-go/openapi-util/service"
	AliYunUtilV2 "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cast"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	TencentCloud "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sms/v20210111"
	"github.com/unti-io/go-utils/utils"
	"gopkg.in/gomail.v2"
)

// ========== 驱动模式常量 ==========
const (
	// SMSModeEmail - 邮件
	SMSModeEmail = "email"
	// SMSModeAliYun - 阿里云短信
	SMSModeAliYun = "aliyun"
	// SMSModeAliYunNumberVerify - 阿里云号码验证
	SMSModeAliYunNumberVerify = "aliyun_number_verify"
	// SMSModeTencent - 腾讯云
	SMSModeTencent = "tencent"
)

// ========== 结构体声明（必须在变量使用前） ==========
// SMSResponse - 短信响应
type SMSResponse struct {
	// 错误信息
	Error       error
	// 结果
	Result      any
	// 文本
	Text        string
	// 验证码
	VerifyCode  string
}

// SMSInterface - 短信接口
type SMSInterface interface {
	// VerifyCode
	/**
	 * @name 发送验证码
	 * @param phone 手机号（必须）
	 * @param code 验证码（可选，不传则随机生成）
	 * @return *SMSResponse
	 */
	VerifyCode(phone any, code ...any) (response *SMSResponse)
}

// GoMailRequest - GoMail邮件服务
type GoMailRequest struct {
	Client   *gomail.Dialer
	Template string
}

// AliYunSMS - 阿里云短信
type AliYunSMS struct {
	Client *AliYunClient.Client
}

// AliYunNumberVerify - 阿里云号码验证（适配SendSmsVerifyCode/CheckSmsVerifyCode接口）
type AliYunNumberVerify struct {
	Client       *AliYunClient.Client
	TemplateCode string // 保存模板Code
	SignName     string // 保存签名
	Endpoint     string // 保存endpoint，避免重复读取
}

// TencentSMS - 腾讯云短信
type TencentSMS struct {
	Client *TencentCloud.Client
}

// ========== 全局变量声明 ==========
// SMSToml - SMS配置文件
var SMSToml *utils.ViperResponse

// 全局实例变量
var (
	SMS                     SMSInterface
	GoMail                  *GoMailRequest
	SMSAliYun               *AliYunSMS
	SMSAliYunNumberVerify   *AliYunNumberVerify // 阿里云号码验证实例
	SMSTencent              *TencentSMS
)

// ========== 初始化函数 ==========
func init() {
	// 初始化配置文件
	initSMSToml()
	// 初始化短信实例
	initSMS()

	// 监听配置文件变化
	if SMSToml != nil && SMSToml.Viper != nil {
		SMSToml.Viper.WatchConfig()
		// 配置文件变化时，重新初始化短信实例
		SMSToml.Viper.OnConfigChange(func(event fsnotify.Event) {
			initSMS()
		})
	}
}

// NewSMS - 创建SMS实例
/**
 * @param mode 驱动模式
 * @return SMSInterface
 * @example：
 * 1. sms := facade.NewSMS("email")
 * 2. sms := facade.NewSMS(facade.SMSModeEmail)
 */
func NewSMS(mode any) SMSInterface {
	switch strings.ToLower(cast.ToString(mode)) {
	case SMSModeEmail:
		SMS = GoMail
	case SMSModeAliYun:
		SMS = SMSAliYun
	case SMSModeAliYunNumberVerify:
		SMS = SMSAliYunNumberVerify
	case SMSModeTencent:
		SMS = SMSTencent
	default:
		SMS = GoMail
	}
	return SMS
}

// initSMSToml - 初始化SMS配置文件
func initSMSToml() {
	item := utils.Viper(utils.ViperModel{
		Path: "config",
		Mode: "toml",
		Name: "sms",
		Content: utils.Replace(TempSMS, map[string]any{
			"${drive.sms}":                              "email",
			"${drive.email}":                            "aliyun",
			"${drive.default}":                          "email",
			"${email.host}":                             "smtp.qq.com",
			"${email.port}":                             465,
			"${email.account}":                          "xxx@qq.com",
			"${email.password}":                         "",
			"${email.nickname}":                         "inis",
			"${email.sign_name}":                        "inis",
			"${aliyun.access_key_id}":                   "",
			"${aliyun.access_key_secret}":               "",
			"${aliyun.endpoint}":                        "dysmsapi.aliyuncs.com",
			"${aliyun.sign_name}":                       "",
			"${aliyun.verify_code}":                     "",
			"${aliyun_number_verify.access_key_id}":     "",
			"${aliyun_number_verify.access_key_secret}": "",
			"${aliyun_number_verify.endpoint}":          "dypnsapi.aliyuncs.com",
			"${aliyun_number_verify.sign_name}":         "",
			"${aliyun_number_verify.template_code}":     "100001", // 号码验证专用模板
			"${tencent.secret_id}":                      "",
			"${tencent.secret_key}":                     "",
			"${tencent.endpoint}":                       "sms.tencentcloudapi.com",
			"${tencent.sms_sdk_app_id}":                 "",
			"${tencent.sign_name}":                      "",
			"${tencent.verify_code}":                    "",
			"${tencent.region}":                         "ap-guangzhou",
		}),
	}).Read()

	if item.Error != nil {
		// 替换Log为fmt.Println（避免Log未定义错误）
		fmt.Printf("SMS配置初始化错误: %v | 位置: %s:%d\n", 
			item.Error, utils.Caller().FileName, utils.Caller().Line)
		return
	}

	SMSToml = &item
}

// initSMS - 初始化所有短信实例
func initSMS() {
	// 邮件服务
	GoMail = &GoMailRequest{}
	GoMail.init()

	// 阿里云短信服务
	SMSAliYun = &AliYunSMS{}
	SMSAliYun.init()

	// 阿里云号码验证服务
	SMSAliYunNumberVerify = &AliYunNumberVerify{}
	SMSAliYunNumberVerify.init()

	// 腾讯云短信服务
	SMSTencent = &TencentSMS{}
	SMSTencent.init()

	// 设置默认驱动
	if SMSToml != nil {
		switch cast.ToString(SMSToml.Get("drive.default")) {
		case "email":
			SMS = GoMail
		case "aliyun":
			SMS = SMSAliYun
		case "aliyun_number_verify":
			SMS = SMSAliYunNumberVerify
		case "tencent":
			SMS = SMSTencent
		default:
			SMS = GoMail
		}
	} else {
		SMS = GoMail // 配置加载失败时默认使用邮件
	}
}

// ================================== GoMail邮件服务 - 实现 ==================================
// init 初始化 邮件服务
func (this *GoMailRequest) init() {
	if SMSToml == nil {
		return
	}
	port := cast.ToInt(SMSToml.Get("email.port"))
	host := cast.ToString(SMSToml.Get("email.host"))
	account := cast.ToString(SMSToml.Get("email.account"))
	password := cast.ToString(SMSToml.Get("email.password"))
	this.Client = gomail.NewDialer(host, port, account, password)
}

// VerifyCode - 发送验证码
func (this *GoMailRequest) VerifyCode(phone any, code ...any) (response *SMSResponse) {
	response = &SMSResponse{}

	if !utils.Is.Email(phone) {
		response.Error = errors.New("格式错误，请给一个正确的邮箱地址")
		return
	}

	if len(code) == 0 {
		code = append(code, utils.Rand.String(6, "0123456789"))
	}

	if utils.Is.Empty(this.Template) {
		this.Template = "您的验证码是：${code}，有效期5分钟。（打死也不要把验证码告诉别人）"
	}

	item := gomail.NewMessage()
	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	item.SetHeader("From", nickname+"<"+account+">")
	// 发送给多个用户
	item.SetHeader("To", cast.ToString(phone))
	// 设置邮件主题
	item.SetHeader("Subject", cast.ToString(SMSToml.Get("email.sign_name")))
	// 替换验证码
	temp := utils.Replace(this.Template, map[string]any{
		"${code}": code[0],
	})
	// 设置邮件正文
	item.SetBody("text/html", temp)

	// 发送邮件
	err := this.Client.DialAndSend(item)
	if err != nil {
		response.Error = err
		return response
	}

	response.VerifyCode = cast.ToString(code[0])
	return response
}

// ================================== 阿里云短信 - 实现 ==================================
// init 初始化 阿里云短信
func (this *AliYunSMS) init() {
	if SMSToml == nil {
		return
	}
	// 读取阿里云短信专用配置
	accessKeyId := cast.ToString(SMSToml.Get("aliyun.access_key_id"))
	accessKeySecret := cast.ToString(SMSToml.Get("aliyun.access_key_secret"))
	endpoint := cast.ToString(SMSToml.Get("aliyun.endpoint", "dysmsapi.aliyuncs.com"))

	// 空值校验
	//if utils.Is.Empty(accessKeyId) || utils.Is.Empty(accessKeySecret) {
	//	fmt.Printf("阿里云短信配置缺失：access_key_id/access_key_secret不能为空 | 位置: %s:%d\n",
	//		utils.Caller().FileName, utils.Caller().Line)
	//	return
	//}

	client, err := AliYunClient.NewClient(&AliYunClient.Config{
		Endpoint:        tea.String(endpoint),
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	})

	if err != nil {
		fmt.Printf("阿里云短信服务初始化错误: %v | 位置: %s:%d\n",
			err, utils.Caller().FileName, utils.Caller().Line)
		return
	}

	this.Client = client
}

// VerifyCode - 发送验证码
func (this *AliYunSMS) VerifyCode(phone any, code ...any) (response *SMSResponse) {
	response = &SMSResponse{}

	// 手机号格式校验
	if !utils.Is.Phone(phone) {
		response.Error = errors.New("格式错误，请给一个正确的手机号码")
		return
	}

	// 读取配置中的模板Code
	templateCode := cast.ToString(SMSToml.Get("aliyun.verify_code"))
	if utils.Is.Empty(templateCode) {
		response.Error = errors.New("阿里云短信模板Code未配置")
		return
	}

	// 验证码有效期，默认5分钟
	min := "5"
	if len(code) > 1 {
		min = cast.ToString(code[1])
	}

	// 生成验证码（不传则随机生成）
	if len(code) == 0 || len(code) == 1 {
		code = append(code, utils.Rand.String(6, "0123456789"))
	}

	// 组装请求参数
	params := map[string]any{
		"PhoneNumbers":  tea.String(cast.ToString(phone)),
		"SignName":      tea.String(cast.ToString(SMSToml.Get("aliyun.sign_name"))),
		"TemplateCode":  tea.String(templateCode),
		"TemplateParam": tea.String(utils.Json.Encode(map[string]any{
			"code": code[0],
			"min":  min,
		})),
	}

	// 签名校验
	if utils.Is.Empty(params["SignName"]) {
		response.Error = errors.New("阿里云短信签名未配置")
		return
	}

	// 发送请求
	runtime := &AliYunUtilV2.RuntimeOptions{}
	request := &AliYunClient.OpenApiRequest{
		Query: AliYunUtil.Query(params),
	}

	result, err := this.Client.CallApi(this.ApiInfo(), request, runtime)
	if err != nil {
		response.Error = err
		return response
	}

	// 响应处理
	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" {
		response.Error = errors.New(cast.ToString(body["Message"]))
		return response
	}

	response.Result = result
	response.Text = utils.Json.Encode(result)
	response.VerifyCode = cast.ToString(code[0])
	return response
}

// ApiInfo - 接口信息
func (this *AliYunSMS) ApiInfo() (result *AliYunClient.Params) {
	return &AliYunClient.Params{
		Action:      tea.String("SendSms"),
		Version:     tea.String("2017-05-25"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}
}

// ================================== 阿里云号码验证 ==================================
// init 初始化 阿里云号码验证
func (this *AliYunNumberVerify) init() {
	if SMSToml == nil {
		return
	}
	// 读取号码验证专用配置
	accessKeyId := cast.ToString(SMSToml.Get("aliyun_number_verify.access_key_id"))
	accessKeySecret := cast.ToString(SMSToml.Get("aliyun_number_verify.access_key_secret"))
	endpoint := cast.ToString(SMSToml.Get("aliyun_number_verify.endpoint", "dypnsapi.aliyuncs.com")) // 号码验证接口域名
	templateCode := cast.ToString(SMSToml.Get("aliyun_number_verify.template_code"))
	signName := cast.ToString(SMSToml.Get("aliyun_number_verify.sign_name"))

	// 空值校验
	if utils.Is.Empty(accessKeyId) || utils.Is.Empty(accessKeySecret) {
		fmt.Printf("阿里云号码验证配置缺失：access_key_id/access_key_secret不能为空 | 位置: %s:%d\n",
			utils.Caller().FileName, utils.Caller().Line)
		return
	}
	if utils.Is.Empty(templateCode) {
		fmt.Printf("阿里云号码验证配置缺失：template_code（模板ID）不能为空 | 位置: %s:%d\n",
			utils.Caller().FileName, utils.Caller().Line)
		return
	}
	if utils.Is.Empty(signName) {
		fmt.Printf("阿里云号码验证配置缺失：sign_name（签名）不能为空 | 位置: %s:%d\n",
			utils.Caller().FileName, utils.Caller().Line)
		return
	}

	client, err := AliYunClient.NewClient(&AliYunClient.Config{
		Endpoint:        tea.String(endpoint),
		AccessKeyId:     tea.String(accessKeyId),
		AccessKeySecret: tea.String(accessKeySecret),
	})
	if err != nil {
		fmt.Printf("阿里云号码验证服务初始化错误: %v | 位置: %s:%d\n",
			err, utils.Caller().FileName, utils.Caller().Line)
		return
	}

	// 配置保存到结构体
	this.Client = client
	this.TemplateCode = templateCode
	this.SignName = signName
	this.Endpoint = endpoint
}

// SendSmsVerifyCode - 发送号码验证验证码
func (this *AliYunNumberVerify) SendSmsVerifyCode(phone any, params ...map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	// 手机号格式校验
	if !utils.Is.Phone(phone) {
		response.Error = errors.New("格式错误，请给一个正确的手机号码")
		return
	}
	phoneStr := cast.ToString(phone)

	// 初始化默认参数
	reqParams := map[string]any{
		"SchemeName":      tea.String("默认方案"),
		"CountryCode":     tea.String("86"),
		"PhoneNumber":     tea.String(phoneStr),
		"SignName":        tea.String(this.SignName),
		"TemplateCode":    tea.String(this.TemplateCode),
		"TemplateParam":   tea.String(`{"code":"##code##","min":"5"}`), // 默认系统生成验证码
		"SmsUpExtendCode": tea.String(""),
		"OutId":           tea.String(""),
		"CodeLength":      tea.Int64(6),
		"ValidTime":       tea.Int64(300),
		"DuplicatePolicy": tea.Int64(1),
		"Interval":        tea.Int64(60),
		"CodeType":        tea.Int64(1),
		"ReturnVerifyCode": tea.Bool(true),
		"AutoRetry":       tea.Int64(1),
	}

	// 覆盖自定义参数
	if len(params) > 0 && params[0] != nil {
		for k, v := range params[0] {
			switch k {
			case "SchemeName":
				reqParams["SchemeName"] = tea.String(cast.ToString(v))
			case "TemplateParam":
				reqParams["TemplateParam"] = tea.String(cast.ToString(v))
			case "OutId":
				reqParams["OutId"] = tea.String(cast.ToString(v))
			case "CodeLength":
				reqParams["CodeLength"] = tea.Int64(cast.ToInt64(v))
			case "ValidTime":
				reqParams["ValidTime"] = tea.Int64(cast.ToInt64(v))
			case "CodeType":
				reqParams["CodeType"] = tea.Int64(cast.ToInt64(v))
			case "ReturnVerifyCode":
				reqParams["ReturnVerifyCode"] = tea.Bool(cast.ToBool(v))
			}
		}
	}

	// 发送请求
	runtime := &AliYunUtilV2.RuntimeOptions{}
	request := &AliYunClient.OpenApiRequest{
		Query: AliYunUtil.Query(reqParams),
	}

	result, err := this.Client.CallApi(this.SendSmsVerifyCodeApiInfo(), request, runtime)
	if err != nil {
		response.Error = err
		return response
	}

	// 响应处理
	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" || !cast.ToBool(body["Success"]) {
		errMsg := cast.ToString(body["Message"])
		if utils.Is.Empty(errMsg) {
			errMsg = "阿里云号码验证验证码发送失败"
		}
		response.Error = errors.New(errMsg)
		return response
	}

	// 解析返回结果
	model := cast.ToStringMap(body["Model"])
	response.VerifyCode = cast.ToString(model["VerifyCode"])
	response.Result = result
	response.Text = utils.Json.Encode(result)

	return response
}

// CheckSmsVerifyCode - 核验验证码
func (this *AliYunNumberVerify) CheckSmsVerifyCode(phone any, verifyCode string, params ...map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	// 基础校验
	if !utils.Is.Phone(phone) {
		response.Error = errors.New("格式错误，请给一个正确的手机号码")
		return
	}
	if utils.Is.Empty(verifyCode) {
		response.Error = errors.New("验证码不能为空")
		return
	}
	phoneStr := cast.ToString(phone)

	// 初始化默认参数
	reqParams := map[string]any{
		"SchemeName":     tea.String("默认方案"),
		"CountryCode":    tea.String("86"),
		"PhoneNumber":    tea.String(phoneStr),
		"OutId":          tea.String(""),
		"VerifyCode":     tea.String(verifyCode),
		"CaseAuthPolicy": tea.Int64(1), // 不区分大小写
	}

	// 覆盖自定义参数
	if len(params) > 0 && params[0] != nil {
		for k, v := range params[0] {
			switch k {
			case "SchemeName":
				reqParams["SchemeName"] = tea.String(cast.ToString(v))
			case "OutId":
				reqParams["OutId"] = tea.String(cast.ToString(v))
			case "CaseAuthPolicy":
				reqParams["CaseAuthPolicy"] = tea.Int64(cast.ToInt64(v))
			}
		}
	}

	// 发送核验请求
	runtime := &AliYunUtilV2.RuntimeOptions{}
	request := &AliYunClient.OpenApiRequest{
		Query: AliYunUtil.Query(reqParams),
	}

	result, err := this.Client.CallApi(this.CheckSmsVerifyCodeApiInfo(), request, runtime)
	if err != nil {
		response.Error = err
		return response
	}

	// 响应处理
	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" || !cast.ToBool(body["Success"]) {
		errMsg := cast.ToString(body["Message"])
		if utils.Is.Empty(errMsg) {
			errMsg = "验证码核验接口调用失败"
		}
		response.Error = errors.New(errMsg)
		return response
	}

	// 解析核验结果
	model := cast.ToStringMap(body["Model"])
	verifyResult := cast.ToString(model["VerifyResult"])
	if verifyResult != "PASS" {
		response.Error = errors.New("验证码核验失败")
		return response
	}

	// 成功返回
	response.Result = result
	response.Text = "验证码核验成功"
	return response
}

// VerifyCode - 兼容原有接口的发送方法
func (this *AliYunNumberVerify) VerifyCode(phone any, code ...any) (response *SMSResponse) {
	// 兼容原有调用方式，默认调用SendSmsVerifyCode
	var templateParam string
	if len(code) > 0 {
		// 如果传入了验证码，则使用自定义验证码模式
		templateParam = utils.Json.Encode(map[string]any{
			"code": cast.ToString(code[0]),
			"min":  "5",
		})
	} else {
		// 未传入则使用系统生成验证码
		templateParam = `{"code":"##code##","min":"5"}`
	}

	return this.SendSmsVerifyCode(phone, map[string]any{
		"TemplateParam": templateParam,
	})
}

// SendSmsVerifyCodeApiInfo - 发送验证码接口信息
func (this *AliYunNumberVerify) SendSmsVerifyCodeApiInfo() (result *AliYunClient.Params) {
	return &AliYunClient.Params{
		Action:      tea.String("SendSmsVerifyCode"),
		Version:     tea.String("2017-05-25"), // 号码验证接口版本
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}
}

// CheckSmsVerifyCodeApiInfo - 核验验证码接口信息
func (this *AliYunNumberVerify) CheckSmsVerifyCodeApiInfo() (result *AliYunClient.Params) {
	return &AliYunClient.Params{
		Action:      tea.String("CheckSmsVerifyCode"),
		Version:     tea.String("2017-05-25"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("json"),
		BodyType:    tea.String("json"),
	}
}

// ================================== 腾讯云短信 - 实现 ==================================
// init 初始化 腾讯云短信
func (this *TencentSMS) init() {
	if SMSToml == nil {
		return
	}
	secretId := cast.ToString(SMSToml.Get("tencent.secret_id"))
	secretKey := cast.ToString(SMSToml.Get("tencent.secret_key"))

	// 空值校验
	//if utils.Is.Empty(secretId) || utils.Is.Empty(secretKey) {
	//	fmt.Printf("腾讯云短信配置缺失：secret_id/secret_key不能为空 | 位置: %s:%d\n",
	//		utils.Caller().FileName, utils.Caller().Line)
	//	return
	//}

	credential := common.NewCredential(secretId, secretKey)
	clientProfile := profile.NewClientProfile()
	clientProfile.HttpProfile.Endpoint = cast.ToString(SMSToml.Get("tencent.endpoint", "sms.tencentcloudapi.com"))

	client, err := TencentCloud.NewClient(
		credential,
		cast.ToString(SMSToml.Get("tencent.region", "ap-guangzhou")),
		clientProfile,
	)

	if err != nil {
		fmt.Printf("腾讯云短信服务初始化错误: %v | 位置: %s:%d\n",
			err, utils.Caller().FileName, utils.Caller().Line)
		return
	}

	this.Client = client
}

// VerifyCode - 发送验证码
func (this *TencentSMS) VerifyCode(phone any, code ...any) (response *SMSResponse) {
	response = &SMSResponse{}

	// 手机号格式校验
	if !utils.Is.Phone(phone) {
		response.Error = errors.New("格式错误，请给一个正确的手机号码")
		return
	}

	// 配置校验
	sdkAppId := cast.ToString(SMSToml.Get("tencent.sms_sdk_app_id"))
	signName := cast.ToString(SMSToml.Get("tencent.sign_name"))
	templateId := cast.ToString(SMSToml.Get("tencent.verify_code"))

	if utils.Is.Empty(sdkAppId) {
		response.Error = errors.New("腾讯云短信SDK AppID未配置")
		return
	}
	if utils.Is.Empty(signName) {
		response.Error = errors.New("腾讯云短信签名未配置")
		return
	}
	if utils.Is.Empty(templateId) {
		response.Error = errors.New("腾讯云短信模板ID未配置")
		return
	}

	// 生成验证码
	if len(code) == 0 {
		code = append(code, utils.Rand.String(6, "0123456789"))
	}

	// 组装请求
	request := TencentCloud.NewSendSmsRequest()
	request.PhoneNumberSet = common.StringPtrs([]string{cast.ToString(phone)})
	request.SmsSdkAppId = common.StringPtr(sdkAppId)
	request.SignName = common.StringPtr(signName)
	request.TemplateId = common.StringPtr(templateId)
	request.TemplateParamSet = common.StringPtrs([]string{cast.ToString(code[0])})

	// 发送请求
	item, err := this.Client.SendSms(request)
	if err != nil {
		response.Error = err
		return response
	}

	// 响应边界处理
	if item == nil || item.Response == nil {
		response.Error = errors.New("腾讯云短信响应为空")
		return response
	}

	if len(item.Response.SendStatusSet) == 0 {
		response.Error = errors.New("腾讯云短信发送状态为空")
		return response
	}

	status := item.Response.SendStatusSet[0]
	if status == nil || *status.Code != "Ok" {
		errMsg := "未知错误"
		if status != nil && status.Message != nil {
			errMsg = *status.Message
		}
		response.Error = errors.New(errMsg)
		return response
	}

	// 响应赋值
	response.VerifyCode = cast.ToString(code[0])
	response.Text = item.ToJsonString()
	response.Result = utils.Json.Decode(item.ToJsonString())

	return response
}