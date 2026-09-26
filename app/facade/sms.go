package facade

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// ========== SMS 日志记录 ==========
// smsLog - 记录短信/邮件发送日志到独立文件
// logType: "email" 或 "sms"
func smsLog(logType string, success bool, detail map[string]any) {
	logDir := filepath.Join("runtime", "sms")
	// 创建目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	logFile := filepath.Join(logDir, logType+".log")

	// 构建日志内容
	status := "成功"
	if !success {
		status = "失败"
	}

	// 构建详情字符串
	detailStr := ""
	for k, v := range detail {
		detailStr += fmt.Sprintf(" | %s: %v", k, v)
	}

	line := fmt.Sprintf("[%s] [%s] %s%s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		status,
		logType,
		detailStr,
	)

	// 以追加模式写入
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(line)
}

// ========== 结构体声明（必须在变量使用前） ==========
// SMSResponse - 短信响应
type SMSResponse struct {
	// 错误信息
	Error error
	// 结果
	Result any
	// 文本
	Text string
	// 验证码
	VerifyCode string
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
	// SendCommentNotify
	/**
	 * @name 发送评论通知
	 * @param recipient 收件人邮箱
	 * @param commentInfo 评论信息
	 * @return *SMSResponse
	 */
	SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse)
	// SendReplyNotify
	/**
	 * @name 发送评论回复通知
	 * @param recipient 收件人邮箱
	 * @param commentInfo 评论信息
	 * @return *SMSResponse
	 */
	SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse)
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
	SMS                   SMSInterface
	GoMail                *GoMailRequest
	SMSAliYun             *AliYunSMS
	SMSAliYunNumberVerify *AliYunNumberVerify // 阿里云号码验证实例
	SMSTencent            *TencentSMS
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
	opts := map[string]any{
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
	}

	// 发件队列参数（[email] 段）：模板里是占位符，这里补上默认值
	for key, val := range MailQueueDefaultValues() {
		opts["${email."+key+"}"] = val
	}

	item := utils.Viper(utils.ViperModel{
		Path:    "config",
		Mode:    "toml",
		Name:    "sms",
		Content: utils.Replace(TempSMS, opts),
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

	// 邮件发送队列：启动 worker 并刷新分批 / 重试参数
	// （配置文件热更新会重新执行 initSMS，因此这里同时承担 reload 的职责）
	mailQueue.reload()
	mailQueue.start()
}

// ================================== GoMail邮件服务 - 实现 ==================================
// init 初始化 邮件服务
func (this *GoMailRequest) init() {
	if SMSToml == nil {
		smsLog("email", false, map[string]any{"error": "SMS配置文件未加载"})
		return
	}
	port := cast.ToInt(SMSToml.Get("email.port"))
	host := cast.ToString(SMSToml.Get("email.host"))
	account := cast.ToString(SMSToml.Get("email.account"))
	password := cast.ToString(SMSToml.Get("email.password"))

	if utils.Is.Empty(host) || utils.Is.Empty(account) {
		smsLog("email", false, map[string]any{"error": "邮件配置缺失", "host": host, "account": account})
		return
	}

	this.Client = gomail.NewDialer(host, port, account, password)
}

// VerifyCode - 发送验证码（邮箱驱动）
//
// 流程：同步做参数 / 配置校验并生成验证码 -> 入队到「优先通道」（不占用批量窗口，来了就发）
// -> 有界等待首轮发送结果（超时按「已受理」处理，不阻塞请求）。
//
// 说明：验证码需要立即返回给调用方缓存（5 分钟有效），因此首轮发送失败时把错误同步返回，
// 调用方提示用户重试；任务本身仍留在队列里按重试规则异步重试。
func (this *GoMailRequest) VerifyCode(phone any, code ...any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(cast.ToString(phone)); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": phone, "error": err, "type": MailKindVerify})
		return
	}

	if len(code) == 0 {
		code = append(code, utils.Rand.String(6, "0123456789"))
	}
	verify := cast.ToString(code[0])

	result := mailQueue.enqueueUrgent(&MailTask{
		Kind:      MailKindVerify,
		Recipient: cast.ToString(phone),
		Code:      verify,
	})

	if result != nil && result.Error != nil {
		response.Error = result.Error
		return response
	}

	response.VerifyCode = verify
	return response
}

// check 入队前的基础校验（邮箱格式 / 邮件服务是否就绪）
//
// 这类问题属于参数或配置错误，同步返回比丢进队列反复重试更有意义，
// 因此校验放在入队之前（真正发送时再校验一次，防止运行期配置被改坏）。
func (this *GoMailRequest) check(recipient string) error {
	if !utils.Is.Email(recipient) {
		return errors.New("格式错误，请给一个正确的邮箱地址")
	}
	if this.Client == nil {
		return errors.New("邮件服务未初始化，请检查config/sms.toml配置")
	}
	return nil
}

// sendVerifyCode 渲染并发送验证码邮件（由邮件队列 worker 调用，code 已在上游生成）
func (this *GoMailRequest) sendVerifyCode(recipient, code string) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindVerify})
		return
	}

	if utils.Is.Empty(this.Template) {
		this.Template = "您的验证码是：${code}，有效期5分钟。（打死也不要把验证码告诉别人）"
	}

	item := gomail.NewMessage()
	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	item.SetHeader("From", nickname+"<"+account+">")
	// 发送给多个用户
	item.SetHeader("To", recipient)
	// 设置邮件主题
	item.SetHeader("Subject", cast.ToString(SMSToml.Get("email.sign_name")))
	// 替换验证码
	temp := utils.Replace(this.Template, map[string]any{
		"${code}": code,
	})
	// 设置邮件正文
	item.SetBody("text/html", temp)

	// 发送邮件
	err := this.Client.DialAndSend(item)
	if err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindVerify})
		return response
	}

	response.VerifyCode = code
	response.Result = "邮件发送成功"
	smsLog("email", true, map[string]any{"recipient": recipient, "type": MailKindVerify})
	return response
}

// SendCommentNotify - 发送评论通知邮件（入队：分批限流 + 失败延迟重试，非阻塞）
//
// 返回 Error != nil 表示参数 / 配置有误（同步返回，不入队）；否则任务已受理，
// 真正的发送与失败重试由邮件队列完成（见 app/facade/mail_queue.go）。
func (this *GoMailRequest) SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindComment})
		return
	}

	if !mailQueue.enqueue(&MailTask{
		Kind:      MailKindComment,
		Priority:  MailNormal,
		Recipient: recipient,
		Data:      commentInfo,
	}) {
		response.Error = errors.New("邮件队列已满，请稍后再试")
		return
	}

	response.Result = "已加入发送队列"
	return
}

// sendCommentNotify 渲染并发送评论通知邮件（由邮件队列 worker 调用）
func (this *GoMailRequest) sendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "评论通知"})
		return
	}

	template := `
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>新评论通知</title>
	<style>
	* { margin: 0; padding: 0; box-sizing: border-box; }
	body { line-height: 1.7; color: #444; background-color: #f8f9fa; padding: 20px 0; }
	.container { max-width: 720px; margin: 0 auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
	.mail-header { background: #165DFF; padding: 24px 30px; color: #fff; }
	.brand { display: flex; align-items: center; gap: 12px; }
	.brand-name { font-size: 18px; font-weight: 600; }
	.mail-content { padding: 30px; }
	.mail-title { font-size: 22px; color: #222; margin-bottom: 20px; padding-bottom: 15px; border-bottom: 1px solid #f0f0f0; }
	.subtitle { color: #666; margin-bottom: 24px; font-size: 15px; }
	.comment-card { background: #f9fafb; border-radius: 8px; padding: 20px; margin: 20px 0 30px; border-left: 4px solid #165DFF; font-size: 15px; }
	.comment-content { line-height: 1.8; color: #333; }
	.action-btn { display: inline-block; background: #165DFF; color: #fff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 500; margin: 10px 0 25px; transition: background 0.3s; }
	.action-btn:hover { background: #0E42D2; }
	.mail-footer { padding: 20px 30px; background: #f9fafb; border-top: 1px solid #f0f0f0; font-size: 14px; color: #888; }
	.footer-note { margin-bottom: 12px; }
	.unsubscribe { color: #165DFF; text-decoration: none; }
	.unsubscribe:hover { text-decoration: underline; }
	@media (max-width: 600px) {
		.container { width: 95%; margin: 0 auto; }
		.mail-header, .mail-content, .mail-footer { padding: 20px 15px; }
		.mail-title { font-size: 18px; }
		.action-btn { width: 100%; text-align: center; }
	}
	</style>
	</head>
	<body>
	<div class="container">
	<div class="mail-header">
		<div class="brand"><div class="brand-name">新评论通知</div></div>
	</div>
	<div class="mail-content">
		<p class="subtitle">您的${bind_label}《${title}》收到了一条新评论</p>
		<div class="comment-card"><div class="comment-content">${content}</div></div>
		<p><strong>评论者账号：</strong>${author_account}</p>
		<p><strong>评论者昵称：</strong>${author_name}</p>
		<p><strong>评论时间：</strong>${created_at}</p>
		<p><strong>评论者邮箱：</strong>${author_email}</p>
		<p><strong>评论IP：</strong>${ip}</p>
	</div>
	<div class="mail-footer">
		<p class="footer-note">这是自动发送的通知邮件，如有疑问可通过站点内的联系方式找到我</p>
	</div>
	</div>
	</body>
	</html>
	`

	item := gomail.NewMessage()
	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	item.SetHeader("From", nickname+"<"+account+">")
	item.SetHeader("To", recipient)
	item.SetHeader("Subject", "新评论通知 - "+cast.ToString(SMSToml.Get("email.sign_name")))

	// 转换评论信息为正确的格式
	replaceMap := make(map[string]any)
	for key, val := range commentInfo {
		replaceMap["${"+key+"}"] = val
	}

	// 替换模板变量
	temp := utils.Replace(template, replaceMap)
	item.SetBody("text/html", temp)

	// 发送邮件
	err := this.Client.DialAndSend(item)
	if err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "评论通知", "bind_type": commentInfo["bind_type"], "bind_id": commentInfo["bind_id"]})
		return response
	}

	response.Result = "邮件发送成功"
	smsLog("email", true, map[string]any{"recipient": recipient, "type": "评论通知", "bind_type": commentInfo["bind_type"], "bind_id": commentInfo["bind_id"]})
	return response
}

// SendReplyNotify - 发送评论回复通知邮件（入队：分批限流 + 失败延迟重试，非阻塞）
func (this *GoMailRequest) SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindReply})
		return
	}

	if !mailQueue.enqueue(&MailTask{
		Kind:      MailKindReply,
		Priority:  MailNormal,
		Recipient: recipient,
		Data:      commentInfo,
	}) {
		response.Error = errors.New("邮件队列已满，请稍后再试")
		return
	}

	response.Result = "已加入发送队列"
	return
}

// sendReplyNotify 渲染并发送评论回复通知邮件（由邮件队列 worker 调用）
func (this *GoMailRequest) sendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "回复通知"})
		return
	}

	template := `
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>评论回复通知</title>
	<style>
	* { margin: 0; padding: 0; box-sizing: border-box; }
	body { line-height: 1.7; color: #444; background-color: #f8f9fa; padding: 20px 0; }
	.container { max-width: 720px; margin: 0 auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
	.mail-header { background: #165DFF; padding: 24px 30px; color: #fff; }
	.brand { display: flex; align-items: center; gap: 12px; }
	.brand-name { font-size: 18px; font-weight: 600; }
	.mail-content { padding: 30px; }
	.mail-title { font-size: 22px; color: #222; margin-bottom: 20px; padding-bottom: 15px; border-bottom: 1px solid #f0f0f0; }
	.subtitle { color: #666; margin-bottom: 24px; font-size: 15px; }
	.comment-card { background: #f9fafb; border-radius: 8px; padding: 20px; margin: 20px 0 30px; border-left: 4px solid #165DFF; font-size: 15px; }
	.comment-content { line-height: 1.8; color: #333; }
	.action-btn { display: inline-block; background: #165DFF; color: #fff; padding: 12px 24px; border-radius: 6px; text-decoration: none; font-weight: 500; margin: 10px 0 25px; transition: background 0.3s; }
	.action-btn:hover { background: #0E42D2; }
	.mail-footer { padding: 20px 30px; background: #f9fafb; border-top: 1px solid #f0f0f0; font-size: 14px; color: #888; }
	.footer-note { margin-bottom: 12px; }
	.unsubscribe { color: #165DFF; text-decoration: none; }
	.unsubscribe:hover { text-decoration: underline; }
	@media (max-width: 600px) {
		.container { width: 95%; margin: 0 auto; }
		.mail-header, .mail-content, .mail-footer { padding: 20px 15px; }
		.mail-title { font-size: 18px; }
		.action-btn { width: 100%; text-align: center; }
	}
	</style>
	</head>
	<body>
	<div class="container">
	<div class="mail-header">
		<div class="brand"><div class="brand-name">评论回复通知</div></div>
	</div>
	<div class="mail-content">
		<p class="subtitle">您在${bind_label}《${title}》中的评论收到了一条回复</p>
		<div class="comment-card"><div class="comment-content">${content}</div></div>
		<p><strong>回复者账号：</strong>${author_account}</p>
		<p><strong>回复者昵称：</strong>${author_name}</p>
		<p><strong>回复时间：</strong>${created_at}</p>
		<p><strong>回复者邮箱：</strong>${author_email}</p>
		<p><strong>回复IP：</strong>${ip}</p>
	</div>
	<div class="mail-footer">
		<p class="footer-note">这是自动发送的通知邮件，如有疑问可通过站点内的联系方式找到我</p>
	</div>
	</div>
	</body>
	</html>
	`

	item := gomail.NewMessage()
	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	item.SetHeader("From", nickname+"<"+account+">")
	item.SetHeader("To", recipient)
	item.SetHeader("Subject", "评论回复通知 - "+cast.ToString(SMSToml.Get("email.sign_name")))

	// 转换评论信息为正确的格式
	replaceMap := make(map[string]any)
	for key, val := range commentInfo {
		replaceMap["${"+key+"}"] = val
	}

	// 替换模板变量
	temp := utils.Replace(template, replaceMap)
	item.SetBody("text/html", temp)

	// 发送邮件
	err := this.Client.DialAndSend(item)
	if err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "回复通知", "bind_type": commentInfo["bind_type"], "bind_id": commentInfo["bind_id"]})
		return response
	}

	response.Result = "邮件发送成功"
	smsLog("email", true, map[string]any{"recipient": recipient, "type": "回复通知", "bind_type": commentInfo["bind_type"], "bind_id": commentInfo["bind_id"]})
	return response
}

// SendMessageNotify - 发送「用户消息通知」邮件
//
// 场景：管理员在后台「消息管理」向指定用户发送系统消息时勾选「同时发送邮件通知」。
// 与评论通知的区别：正文只渲染消息标题与内容，不包含评论者 / 评论 IP 等评论字段，
// 标题、正文都按纯文本处理（转义 + 换行转 <br>），避免消息内容破坏排版。
func (this *GoMailRequest) SendMessageNotify(recipient string, messageInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindMessage})
		return
	}

	if !mailQueue.enqueue(&MailTask{
		Kind:      MailKindMessage,
		Priority:  MailNormal,
		Recipient: recipient,
		Data:      messageInfo,
	}) {
		response.Error = errors.New("邮件队列已满，请稍后再试")
		return
	}

	response.Result = "已加入发送队列"
	return
}

// sendMessageNotify 渲染并发送「用户消息通知」邮件（由邮件队列 worker 调用）
func (this *GoMailRequest) sendMessageNotify(recipient string, messageInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "消息通知"})
		return
	}

	title := strings.TrimSpace(cast.ToString(messageInfo["title"]))
	content := cast.ToString(messageInfo["content"])
	timeText := cast.ToString(messageInfo["time"])
	if utils.Is.Empty(timeText) {
		timeText = time.Now().Format("2006-01-02 15:04:05")
	}

	// 收件人身份（账号 / 昵称）：调用方没传时用「—」占位，避免模板里留下未替换的占位符
	accountText := strings.TrimSpace(cast.ToString(messageInfo["account"]))
	nicknameText := strings.TrimSpace(cast.ToString(messageInfo["nickname"]))
	if utils.Is.Empty(accountText) {
		accountText = "—"
	}
	if utils.Is.Empty(nicknameText) {
		nicknameText = "—"
	}

	site := cast.ToString(SMSToml.Get("email.sign_name"))

	// 正文按纯文本处理：先转义再换行转 <br>（顺序不能反，否则 <br> 会被一起转义）
	body := html.EscapeString(content)
	body = strings.ReplaceAll(body, "\r\n", "<br>")
	body = strings.ReplaceAll(body, "\n", "<br>")

	template := `
	<!DOCTYPE html>
	<html>
	<head>
	<meta charset="UTF-8">
	<title>用户消息通知</title>
	<style>
	* { margin: 0; padding: 0; box-sizing: border-box; }
	body { line-height: 1.7; color: #444; background-color: #f8f9fa; padding: 20px 0; }
	.container { max-width: 720px; margin: 0 auto; background: #fff; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.05); overflow: hidden; }
	.mail-header { background: #165DFF; padding: 24px 30px; color: #fff; }
	.brand { display: flex; align-items: center; gap: 12px; }
	.brand-name { font-size: 18px; font-weight: 600; }
	.mail-content { padding: 30px; }
	.subtitle { color: #666; margin-bottom: 20px; font-size: 15px; }
	.msg-title { font-size: 18px; font-weight: 600; color: #222; margin-bottom: 16px; padding-bottom: 14px; border-bottom: 1px solid #f0f0f0; }
	.msg-card { background: #f9fafb; border-radius: 8px; padding: 20px; margin-bottom: 20px; border-left: 4px solid #165DFF; font-size: 15px; line-height: 1.8; color: #333; word-break: break-word; }
	.meta { font-size: 13px; color: #888; }
	.mail-footer { padding: 20px 30px; background: #f9fafb; border-top: 1px solid #f0f0f0; font-size: 14px; color: #888; }
	.footer-note { margin-bottom: 0; }
	@media (max-width: 600px) {
		.container { width: 95%; margin: 0 auto; }
		.mail-header, .mail-content, .mail-footer { padding: 20px 15px; }
		.msg-title { font-size: 16px; }
	}
	</style>
	</head>
	<body>
	<div class="container">
	<div class="mail-header">
		<div class="brand"><div class="brand-name">用户消息通知</div></div>
	</div>
	<div class="mail-content">
		<p class="subtitle">您收到一条来自「${site}」的消息</p>
		<p class="meta"><strong>账号：</strong>${account}　|　<strong>昵称：</strong>${nickname}</p>
		<div class="msg-title">${title}</div>
		<div class="msg-card">${content}</div>
		<p class="meta"><strong>发送时间：</strong>${time}</p>
	</div>
	<div class="mail-footer">
		<p class="footer-note">这是自动发送的通知邮件，如有疑问可通过站点内的联系方式找到我</p>
	</div>
	</div>
	</body>
	</html>
	`

	item := gomail.NewMessage()
	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	item.SetHeader("From", nickname+"<"+account+">")
	item.SetHeader("To", recipient)
	item.SetHeader("Subject", title+" - "+site)

	temp := utils.Replace(template, map[string]any{
		"${site}":     site,
		"${account}":  html.EscapeString(accountText),
		"${nickname}": html.EscapeString(nicknameText),
		"${title}":    html.EscapeString(title),
		"${content}":  body,
		"${time}":     timeText,
	})

	item.SetBody("text/html", temp)

	if err := this.Client.DialAndSend(item); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "消息通知"})
		return response
	}

	response.Result = "邮件发送成功"
	smsLog("email", true, map[string]any{"recipient": recipient, "type": "消息通知"})
	return response
}

// SendMail - 发送自定义内容的邮件（主题 + 纯文本正文，正文换行会转成 HTML 换行）
//
// 用于欢迎邮件等不属于「评论通知」模板自身的场景；入队发送（分批限流 + 失败延迟重试，非阻塞）。
// 需要「调用方立刻知道发送结果」的关键邮件（如注册验证链接）请用 SendMailUrgent。
func (this *GoMailRequest) SendMail(recipient string, subject string, content string) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindCustom})
		return
	}

	if !mailQueue.enqueue(&MailTask{
		Kind:      MailKindCustom,
		Priority:  MailNormal,
		Recipient: recipient,
		Subject:   subject,
		Content:   content,
	}) {
		response.Error = errors.New("邮件队列已满，请稍后再试")
		return
	}

	response.Result = "已加入发送队列"
	return
}

// SendMailUrgent - 发送自定义邮件（优先通道 + 有界等待首轮结果）
//
// 与 SendMail 的区别：走队列的优先通道（不占批量窗口，立即发送），
// 并等待首轮发送结果（默认 10 秒，见 sms.toml 的 email.verify_wait；超时视为已受理）。
// 适用于注册验证邮件这类「调用方需要给用户明确提示」的邮件。
func (this *GoMailRequest) SendMailUrgent(recipient string, subject string, content string) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": MailKindCustom})
		return
	}

	result := mailQueue.enqueueUrgent(&MailTask{
		Kind:      MailKindCustom,
		Recipient: recipient,
		Subject:   subject,
		Content:   content,
	})

	if result != nil && result.Error != nil {
		response.Error = result.Error
		return
	}

	response.Result = "已受理"
	return
}

// sendCustomMail 渲染并发送自定义邮件（由邮件队列 worker 调用）
func (this *GoMailRequest) sendCustomMail(recipient string, subject string, content string) (response *SMSResponse) {
	response = &SMSResponse{}

	if err := this.check(recipient); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "自定义邮件"})
		return response
	}

	nickname := cast.ToString(SMSToml.Get("email.nickname"))
	account := cast.ToString(SMSToml.Get("email.account"))
	site := cast.ToString(SMSToml.Get("email.sign_name"))

	// 换行转 <br>，其余内容做最小化转义，避免正文被当成 HTML 标签
	body := html.EscapeString(cast.ToString(content))
	body = strings.ReplaceAll(body, "\r\n", "<br>")
	body = strings.ReplaceAll(body, "\n", "<br>")

	template := `<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>` + html.EscapeString(cast.ToString(subject)) + `</title></head>
<body style="margin:0;padding:24px 0;background:#f8f9fa;">
<div style="max-width:640px;margin:0 auto;background:#fff;border-radius:12px;overflow:hidden;box-shadow:0 4px 20px rgba(0,0,0,0.05);">
<div style="padding:20px 30px;background:#165DFF;color:#fff;font-size:16px;font-weight:600;">` + html.EscapeString(site) + `</div>
<div style="padding:24px 30px;line-height:1.8;color:#444;font-size:14px;">` + body + `</div>
<div style="padding:16px 30px;color:#999;font-size:12px;">这是自动发送的邮件，如有疑问可通过站点内的联系方式找到我</div>
</div>
</body>
</html>`

	item := gomail.NewMessage()
	item.SetHeader("From", nickname+"<"+account+">")
	item.SetHeader("To", recipient)
	item.SetHeader("Subject", cast.ToString(subject))
	item.SetBody("text/html", template)

	if err := this.Client.DialAndSend(item); err != nil {
		response.Error = err
		smsLog("email", false, map[string]any{"recipient": recipient, "error": err, "type": "自定义邮件"})
		return response
	}

	response.Result = "邮件发送成功"
	smsLog("email", true, map[string]any{"recipient": recipient, "type": "自定义邮件"})

	return response
}

// sendMailTask 按任务类型分发到对应的「渲染 + 发送」实现（由邮件队列 worker 调用）
func (this *GoMailRequest) sendMailTask(task *MailTask) *SMSResponse {
	switch task.Kind {
	case MailKindVerify:
		return this.sendVerifyCode(task.Recipient, task.Code)
	case MailKindComment:
		return this.sendCommentNotify(task.Recipient, task.Data)
	case MailKindReply:
		return this.sendReplyNotify(task.Recipient, task.Data)
	case MailKindMessage:
		return this.sendMessageNotify(task.Recipient, task.Data)
	default:
		return this.sendCustomMail(task.Recipient, task.Subject, task.Content)
	}
}

// SendMail - 发送自定义内容邮件（仅 email 驱动支持）
// 短信驱动不具备「任意内容邮件」能力，同样返回错误提示，由调用方决定是否忽略；
// 该入口为「入队发送」（非阻塞，失败自动延迟重试）
func SendMail(recipient string, subject string, content string) (response *SMSResponse) {

	if GoMail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	return GoMail.SendMail(recipient, subject, content)
}

// SendCommentNotify - 发送评论通知邮件（包级入口，始终走邮箱驱动）
//
// 与 SendMessageNotify 一样直接使用 GoMail，不受 sms.toml 的驱动模式影响
// （短信驱动发不了邮件）；邮件同样入队分批投递、失败延迟重试。
func SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {

	if GoMail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	return GoMail.SendCommentNotify(recipient, commentInfo)
}

// SendReplyNotify - 发送评论回复通知邮件（包级入口，始终走邮箱驱动）
func SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {

	if GoMail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	return GoMail.SendReplyNotify(recipient, commentInfo)
}

// SendMailUrgent - 发送自定义邮件（优先通道，等待首轮结果）
//
// 用于注册验证邮件这类关键邮件：不占用批量窗口、立即发送，并在超时时间内返回首轮结果。
func SendMailUrgent(recipient string, subject string, content string) (response *SMSResponse) {

	if GoMail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	return GoMail.SendMailUrgent(recipient, subject, content)
}

// SendMessageNotify - 发送「用户消息通知」邮件（包级入口）
//
// 与 SendMail 一样直接走邮箱驱动（GoMail），不受 sms.toml 的驱动模式影响：
// 短信驱动无法发邮件，若走 facade.SMS 会在站点配置短信驱动时静默发不出去。
// messageInfo 支持：title（标题，必填）、content（内容，必填）、time（发送时间，可省）。
func SendMessageNotify(recipient string, messageInfo map[string]any) (response *SMSResponse) {

	if GoMail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	return GoMail.SendMessageNotify(recipient, messageInfo)
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
		smsLog("sms", false, map[string]any{"provider": "aliyun", "phone": phone, "error": response.Error, "type": "验证码"})
		return
	}

	// 读取配置中的模板Code
	templateCode := cast.ToString(SMSToml.Get("aliyun.verify_code"))
	if utils.Is.Empty(templateCode) {
		response.Error = errors.New("阿里云短信模板Code未配置")
		smsLog("sms", false, map[string]any{"provider": "aliyun", "phone": phone, "error": response.Error, "type": "验证码"})
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
		"PhoneNumbers": tea.String(cast.ToString(phone)),
		"SignName":     tea.String(cast.ToString(SMSToml.Get("aliyun.sign_name"))),
		"TemplateCode": tea.String(templateCode),
		"TemplateParam": tea.String(utils.Json.Encode(map[string]any{
			"code": code[0],
			"min":  min,
		})),
	}

	// 签名校验
	if utils.Is.Empty(params["SignName"]) {
		response.Error = errors.New("阿里云短信签名未配置")
		smsLog("sms", false, map[string]any{"provider": "aliyun", "phone": phone, "error": response.Error, "type": "验证码"})
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
		smsLog("sms", false, map[string]any{"provider": "aliyun", "phone": phone, "error": err, "type": "验证码"})
		return response
	}

	// 响应处理
	body := cast.ToStringMap(result["body"])
	if body["Code"] != "OK" {
		response.Error = errors.New(cast.ToString(body["Message"]))
		smsLog("sms", false, map[string]any{"provider": "aliyun", "phone": phone, "error": response.Error, "type": "验证码"})
		return response
	}

	response.Result = result
	response.Text = utils.Json.Encode(result)
	response.VerifyCode = cast.ToString(code[0])
	smsLog("sms", true, map[string]any{"provider": "aliyun", "phone": phone, "type": "验证码"})
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

// SendCommentNotify - 发送评论通知邮件
func (this *AliYunSMS) SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("阿里云短信服务暂不支持评论通知")
	return response
}

// SendReplyNotify - 发送评论回复通知邮件
func (this *AliYunSMS) SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("阿里云短信服务暂不支持评论回复通知")
	return response
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
		"SchemeName":       tea.String("默认方案"),
		"CountryCode":      tea.String("86"),
		"PhoneNumber":      tea.String(phoneStr),
		"SignName":         tea.String(this.SignName),
		"TemplateCode":     tea.String(this.TemplateCode),
		"TemplateParam":    tea.String(`{"code":"##code##","min":"5"}`), // 默认系统生成验证码
		"SmsUpExtendCode":  tea.String(""),
		"OutId":            tea.String(""),
		"CodeLength":       tea.Int64(6),
		"ValidTime":        tea.Int64(300),
		"DuplicatePolicy":  tea.Int64(1),
		"Interval":         tea.Int64(60),
		"CodeType":         tea.Int64(1),
		"ReturnVerifyCode": tea.Bool(true),
		"AutoRetry":        tea.Int64(1),
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
			case "DuplicatePolicy":
				reqParams["DuplicatePolicy"] = tea.Int64(cast.ToInt64(v))
			case "Interval":
				reqParams["Interval"] = tea.Int64(cast.ToInt64(v))
			case "CodeType":
				reqParams["CodeType"] = tea.Int64(cast.ToInt64(v))
			case "ReturnVerifyCode":
				reqParams["ReturnVerifyCode"] = tea.Bool(cast.ToBool(v))
			case "AutoRetry":
				reqParams["AutoRetry"] = tea.Int64(cast.ToInt64(v))
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

// SendCommentNotify - 发送评论通知邮件
func (this *AliYunNumberVerify) SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("阿里云号码验证服务暂不支持评论通知")
	return response
}

// SendReplyNotify - 发送评论回复通知邮件
func (this *AliYunNumberVerify) SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("阿里云号码验证服务暂不支持评论回复通知")
	return response
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
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
		return
	}

	// 配置校验
	sdkAppId := cast.ToString(SMSToml.Get("tencent.sms_sdk_app_id"))
	signName := cast.ToString(SMSToml.Get("tencent.sign_name"))
	templateId := cast.ToString(SMSToml.Get("tencent.verify_code"))

	if utils.Is.Empty(sdkAppId) {
		response.Error = errors.New("腾讯云短信SDK AppID未配置")
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
		return
	}
	if utils.Is.Empty(signName) {
		response.Error = errors.New("腾讯云短信签名未配置")
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
		return
	}
	if utils.Is.Empty(templateId) {
		response.Error = errors.New("腾讯云短信模板ID未配置")
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
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
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": err, "type": "验证码"})
		return response
	}

	// 响应边界处理
	if item == nil || item.Response == nil {
		response.Error = errors.New("腾讯云短信响应为空")
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
		return response
	}

	if len(item.Response.SendStatusSet) == 0 {
		response.Error = errors.New("腾讯云短信发送状态为空")
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": response.Error, "type": "验证码"})
		return response
	}

	status := item.Response.SendStatusSet[0]
	if status == nil || *status.Code != "Ok" {
		errMsg := "未知错误"
		if status != nil && status.Message != nil {
			errMsg = *status.Message
		}
		response.Error = errors.New(errMsg)
		smsLog("sms", false, map[string]any{"provider": "tencent", "phone": phone, "error": errMsg, "type": "验证码"})
		return response
	}

	// 响应赋值
	response.VerifyCode = cast.ToString(code[0])
	response.Text = item.ToJsonString()
	response.Result = utils.Json.Decode(item.ToJsonString())
	smsLog("sms", true, map[string]any{"provider": "tencent", "phone": phone, "type": "验证码"})

	return response
}

// SendCommentNotify - 发送评论通知邮件
func (this *TencentSMS) SendCommentNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("腾讯云短信服务暂不支持评论通知")
	return response
}

// SendReplyNotify - 发送评论回复通知邮件
func (this *TencentSMS) SendReplyNotify(recipient string, commentInfo map[string]any) (response *SMSResponse) {
	response = &SMSResponse{}
	response.Error = errors.New("腾讯云短信服务暂不支持评论回复通知")
	return response
}
