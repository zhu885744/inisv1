package facade

/**
 * 邮件发送队列（分批限流 + 优先级 + 异步重试）
 *
 * 背景：
 * 邮箱发件在「评论 / 回复通知」「群发短消息勾选邮件」「注册欢迎邮件」等场景下，
 * 可能短时间内产生大量请求；直接同步调用 SMTP 既阻塞调用方，也容易被服务商限流甚至封号。
 * 而验证码（注册 / 登录 / 找回密码 / 注册验证链接）必须优先且及时，不能被批量邮件压在后面。
 *
 * 规则：
 * 1. 非阻塞：调用方只入队（enqueue）后立即返回，真正的发送由独立 worker 协程完成；
 * 2. 优先级：MailUrgent（验证码 / 注册验证邮件）高于 MailNormal（通知 / 自定义邮件），
 *    urgent 任务不占用批量窗口额度，入队后立即发送；
 * 3. 分批：每个窗口最多发送 batch_size 封 normal 邮件，发满后等 batch_interval 秒再开新窗口
 *    （默认 10 封 / 10 分钟，可改 config/sms.toml 的 [email] 段）；
 * 4. 重试：发送失败的任务延迟 retry_delay 秒后重新入队（保留原优先级、按入队时间保持先后），
 *    累计尝试次数达到 max_attempts 后标记失败并丢弃 —— 不会无限重试；
 * 5. 超时：单封邮件发送超过 send_timeout 秒按失败处理，避免 SMTP 卡死拖住整个队列；
 * 6. 上限：pending 数超过 queue_size 时，普通任务直接丢弃并记日志（验证码类始终受理）。
 *
 * 注意：队列在内存中，进程重启后未发送的任务会丢失（邮件通知属于尽力而为，可接受）。
 * 若需要更强投递保证，应改为「任务落库 + 定时任务投递」的方案。
 */

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cast"
)

// 邮件任务优先级（数值越小越优先）
const (
	MailUrgent = 0 // 验证码 / 注册验证邮件：立即发送，不占用批量窗口
	MailNormal = 1 // 通知 / 自定义邮件：受批量窗口限制
)

// 邮件任务类型（用于日志与统计，与 runtime/sms/email.log 的 type 字段一致）
const (
	MailKindVerify  = "验证码"
	MailKindComment = "评论通知"
	MailKindReply   = "回复通知"
	MailKindMessage = "消息通知"
	MailKindCustom  = "自定义邮件"
)

// 默认参数（config/sms.toml 未配置时使用）
const (
	defaultMailBatchSize     = 10
	defaultMailBatchInterval = 10 * time.Minute
	defaultMailRetryDelay    = 60 * time.Second
	defaultMailMaxAttempts   = 3
	defaultMailSendTimeout   = 30 * time.Second
	defaultMailVerifyWait    = 10 * time.Second
	defaultMailQueueSize     = 1000
	// 队列空闲时的最长等待时间（用于兜底，避免无谓的自旋）
	defaultMailIdleWait = time.Minute
)

// MailQueueDefaultValues 发件队列参数的默认值（config/sms.toml 的 [email] 段）
//
// 后台「系统设置 → 邮件通知 → 发件队列」读写的就是这组参数
// （PUT /api/toml/sms-email-queue，字段与 reload 读取的 key 一一对应）。
func MailQueueDefaultValues() map[string]any {
	return map[string]any{
		"batch_size":     defaultMailBatchSize,
		"batch_interval": int(defaultMailBatchInterval.Seconds()),
		"retry_delay":    int(defaultMailRetryDelay.Seconds()),
		"max_attempts":   defaultMailMaxAttempts,
		"send_timeout":   int(defaultMailSendTimeout.Seconds()),
		"verify_wait":    int(defaultMailVerifyWait.Seconds()),
		"queue_size":     defaultMailQueueSize,
	}
}

// MailQueueLimits 发件队列参数的取值范围（min / max），与 reload 的边界保护一致
// 注意：verify_wait 允许 0（表示验证码也不等待首轮结果，纯异步）
func MailQueueLimits() map[string][2]int {
	return map[string][2]int{
		"batch_size":     {1, 1000},
		"batch_interval": {1, 86400},
		"retry_delay":    {1, 86400},
		"max_attempts":   {1, 10},
		"send_timeout":   {5, 600},
		"verify_wait":    {0, 60},
		"queue_size":     {10, 1000000},
	}
}

// MailTask 一封待发送的邮件
type MailTask struct {
	Kind      string         // 任务类型（MailKind*）
	Priority  int            // 优先级（MailUrgent / MailNormal）
	Recipient string         // 收件人邮箱
	Subject   string         // 主题（自定义邮件用）
	Content   string         // 正文（自定义邮件用）
	Data      map[string]any // 模板变量（评论 / 回复 / 消息通知用）
	Code      string         // 验证码（验证码邮件用）

	Attempts  int       // 已尝试次数（含首次）
	Next      time.Time // 允许发送的时间（延迟重试用）
	CreatedAt time.Time // 入队时间（同优先级按此保持先后顺序）

	// receipt 首轮发送结果的回执通道（仅 urgent 任务使用；缓冲 1，只回执一次）
	receipt chan *SMSResponse
}

// mailConfig 队列参数（配置变化时由 reload 刷新，读写都要持有 mu）
type mailConfig struct {
	batchSize     int           // 每个窗口最多发送的普通邮件数
	batchInterval time.Duration // 批次窗口长度
	retryDelay    time.Duration // 失败后的重试延迟
	maxAttempts   int           // 最大尝试次数（含首次）
	sendTimeout   time.Duration // 单封发送超时
	verifyWait    time.Duration // 验证码类等待首轮结果的超时（0 = 不等待，纯异步）
	queueSize     int           // 队列最大长度（仅限制普通任务）
}

// mailQueueStruct 邮件发送队列
type mailQueueStruct struct {
	mu     sync.Mutex
	tasks  []*MailTask
	cfg    mailConfig
	once   sync.Once
	signal chan struct{}

	// 当前批次窗口
	windowStart time.Time
	windowSent  int

	// 统计（atomic 读写）
	sent    int64 // 发送成功
	failed  int64 // 达到最大尝试次数，标记失败
	dropped int64 // 队列已满被丢弃
	retries int64 // 触发重试的次数
}

// mailQueue 全局邮件队列（facade 包内使用，外部通过 sms.go 的发送方法间接入队）
var mailQueue = &mailQueueStruct{
	cfg: mailConfig{
		batchSize:     defaultMailBatchSize,
		batchInterval: defaultMailBatchInterval,
		retryDelay:    defaultMailRetryDelay,
		maxAttempts:   defaultMailMaxAttempts,
		sendTimeout:   defaultMailSendTimeout,
		verifyWait:    defaultMailVerifyWait,
		queueSize:     defaultMailQueueSize,
	},
	signal: make(chan struct{}, 1),
}

// start 启动队列：读取配置并拉起 worker（幂等，可重复调用）
func (this *mailQueueStruct) start() {
	this.once.Do(func() {
		this.reload()
		go this.run()
	})
}

// reload 重新读取 config/sms.toml 的队列参数（配置文件热更新时由 initSMS 调用）
func (this *mailQueueStruct) reload() {
	if SMSToml == nil {
		return
	}

	batchSize := mailConfigInt("email.batch_size", defaultMailBatchSize, 1, 1000)
	batchInterval := time.Duration(mailConfigInt("email.batch_interval", int(defaultMailBatchInterval.Seconds()), 1, 86400)) * time.Second
	retryDelay := time.Duration(mailConfigInt("email.retry_delay", int(defaultMailRetryDelay.Seconds()), 1, 86400)) * time.Second
	maxAttempts := mailConfigInt("email.max_attempts", defaultMailMaxAttempts, 1, 10)
	sendTimeout := time.Duration(mailConfigInt("email.send_timeout", int(defaultMailSendTimeout.Seconds()), 5, 600)) * time.Second
	verifyWait := time.Duration(mailConfigIntAllowZero("email.verify_wait", int(defaultMailVerifyWait.Seconds()), 60)) * time.Second
	queueSize := mailConfigInt("email.queue_size", defaultMailQueueSize, 10, 1000000)

	this.mu.Lock()
	this.cfg = mailConfig{
		batchSize:     batchSize,
		batchInterval: batchInterval,
		retryDelay:    retryDelay,
		maxAttempts:   maxAttempts,
		sendTimeout:   sendTimeout,
		verifyWait:    verifyWait,
		queueSize:     queueSize,
	}
	this.mu.Unlock()
}

// mailConfigInt 读取队列相关的整数配置（缺省 / 非法值回退默认值，并做上下限保护）
func mailConfigInt(key string, def, min, max int) int {
	value := cast.ToInt(SMSToml.Get(key))
	if value <= 0 {
		value = def
	}
	if value < min {
		value = min
	}
	if max > 0 && value > max {
		value = max
	}
	return value
}

// mailConfigIntAllowZero 读取「允许配 0」的整数配置（0 表示禁用，未配置时用默认值）
// 用于 email.verify_wait：配 0 表示验证码也不等待首轮结果（纯异步）
func mailConfigIntAllowZero(key string, def, max int) int {
	raw := SMSToml.Get(key)
	if raw == nil || cast.ToString(raw) == "" {
		return def
	}

	value := cast.ToInt(raw)
	if value < 0 {
		value = 0
	}
	if max > 0 && value > max {
		value = max
	}
	return value
}

// enqueue 入队（非阻塞）
// 返回 false 表示队列已满且该任务被丢弃（只会发生在普通任务上，验证码类请用 enqueueUrgent）
func (this *mailQueueStruct) enqueue(task *MailTask) bool {
	this.start()

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	this.mu.Lock()
	// 队列长度上限只限制普通任务：验证码等时效性任务必须受理
	pending, queueSize := len(this.tasks), this.cfg.queueSize
	if task.Priority != MailUrgent && pending >= queueSize {
		this.mu.Unlock()
		atomic.AddInt64(&this.dropped, 1)
		Log.Warn(map[string]any{
			"type":      task.Kind,
			"recipient": task.Recipient,
			"queue":     pending,
			"size":      queueSize,
		}, "邮件队列已满，丢弃该邮件")
		smsLog("email", false, map[string]any{"recipient": task.Recipient, "type": task.Kind, "error": "队列已满，已丢弃", "event": "已丢弃"})
		return false
	}
	this.tasks = append(this.tasks, task)
	this.mu.Unlock()

	this.wake()
	return true
}

// enqueueUrgent 入队到优先通道（验证码 / 注册验证邮件）并等待首轮发送结果
//
// 返回值：
//   - nil：等待超时（任务仍在队列中，后续由队列重试 / 发送）
//   - response：首轮发送结果（Error == nil 表示已发出）
//
// 说明：这里只等待「首轮」，且超时受 cfg.verifyWait 限制，不会长时间阻塞调用方；
// 首轮失败的任务依旧按重试规则留在队列里（异步重试）。
func (this *mailQueueStruct) enqueueUrgent(task *MailTask) *SMSResponse {
	task.Priority = MailUrgent
	task.receipt = make(chan *SMSResponse, 1)

	if !this.enqueue(task) {
		return &SMSResponse{Error: errors.New("邮件队列已满，请稍后再试")}
	}

	this.mu.Lock()
	wait := this.cfg.verifyWait
	this.mu.Unlock()

	if wait <= 0 {
		return nil
	}

	select {
	case response := <-task.receipt:
		return response
	case <-time.After(wait):
		// 超时：任务已入队，按「已受理」处理，避免阻塞请求
		return nil
	}
}

// Stats 队列运行状态（便于排查 / 后续做后台展示）
func (this *mailQueueStruct) Stats() map[string]any {
	this.mu.Lock()
	pending := len(this.tasks)
	cfg := this.cfg
	windowSent := this.windowSent
	this.mu.Unlock()

	return map[string]any{
		"pending":       pending,
		"window_sent":   windowSent,
		"batch_size":    cfg.batchSize,
		"batch_interval": cfg.batchInterval.String(),
		"sent":          atomic.LoadInt64(&this.sent),
		"failed":        atomic.LoadInt64(&this.failed),
		"dropped":       atomic.LoadInt64(&this.dropped),
		"retries":       atomic.LoadInt64(&this.retries),
	}
}

// MailQueueStats 邮件队列运行状态（包级入口）
func MailQueueStats() map[string]any {
	return mailQueue.Stats()
}

// wake 唤醒 worker（信号通道有缓冲，重复唤醒会被合并）
func (this *mailQueueStruct) wake() {
	select {
	case this.signal <- struct{}{}:
	default:
	}
}

// run worker 主循环：取任务 -> 发送 ->（失败）延迟重试
func (this *mailQueueStruct) run() {
	for {
		task, wait := this.pick(time.Now())
		if task == nil {
			if wait <= 0 {
				wait = defaultMailIdleWait
			}
			select {
			case <-this.signal:
			case <-time.After(wait):
			}
			continue
		}

		this.deliver(task)
	}
}

// pick 取出下一个可发送的任务（返回 nil 时 wait 表示建议的等待时长）
//
// 选取顺序：urgent 任务优先（不受批次窗口限制）-> 普通任务（窗口还有额度时）
// 同优先级按入队时间先后；未到发送时间的任务（延迟重试中）只参与计算等待时长。
func (this *mailQueueStruct) pick(now time.Time) (*MailTask, time.Duration) {
	this.mu.Lock()
	defer this.mu.Unlock()

	if len(this.tasks) == 0 {
		return nil, defaultMailIdleWait
	}

	remain, windowWait := this.windowRemain(now)

	urgentIndex, normalIndex := -1, -1
	var nextDue time.Time

	for index, task := range this.tasks {
		if task.Next.After(now) {
			if nextDue.IsZero() || task.Next.Before(nextDue) {
				nextDue = task.Next
			}
			continue
		}

		if task.Priority == MailUrgent {
			if urgentIndex < 0 || task.CreatedAt.Before(this.tasks[urgentIndex].CreatedAt) {
				urgentIndex = index
			}
			continue
		}

		if remain <= 0 {
			continue
		}
		if normalIndex < 0 || task.CreatedAt.Before(this.tasks[normalIndex].CreatedAt) {
			normalIndex = index
		}
	}

	index := urgentIndex
	if index < 0 {
		index = normalIndex
	}

	if index >= 0 {
		task := this.tasks[index]
		this.tasks = append(this.tasks[:index], this.tasks[index+1:]...)
		return task, 0
	}

	// 没有可发送的任务：优先等窗口刷新，其次等最近的重试时间
	wait := windowWait
	if wait <= 0 && !nextDue.IsZero() {
		wait = nextDue.Sub(now)
	}
	if wait <= 0 {
		wait = defaultMailIdleWait
	}

	return nil, wait
}

// windowRemain 当前批次窗口剩余额度（<= 0 表示本窗口已用完，wait 为下一个窗口的时间）
// 调用方需持有 mu
func (this *mailQueueStruct) windowRemain(now time.Time) (int, time.Duration) {
	if this.windowStart.IsZero() || now.Sub(this.windowStart) >= this.cfg.batchInterval {
		// 尚未开始窗口 / 窗口已过期：满额度可用
		return this.cfg.batchSize, 0
	}

	if remain := this.cfg.batchSize - this.windowSent; remain > 0 {
		return remain, 0
	}

	return 0, this.windowStart.Add(this.cfg.batchInterval).Sub(now)
}

// markWindow 记账：普通任务占用一个批次额度（无论成败，避免失败重试变成高频轰炸）
// 调用方需持有 mu
func (this *mailQueueStruct) markWindow(now time.Time) {
	if this.windowStart.IsZero() || now.Sub(this.windowStart) >= this.cfg.batchInterval {
		this.windowStart = now
		this.windowSent = 0
	}
	this.windowSent++
}

// deliver 发送单个任务，并按结果决定「成功 / 重试 / 标记失败」
func (this *mailQueueStruct) deliver(task *MailTask) {
	task.Attempts++

	if task.Priority == MailNormal {
		this.mu.Lock()
		this.markWindow(time.Now())
		this.mu.Unlock()
	}

	response := this.dispatch(task)
	success := response != nil && response.Error == nil

	// 首轮结果回执（有界等待的调用方会读；只回执一次，避免后续重试写满缓冲通道）
	receipt := task.receipt
	task.receipt = nil
	if receipt != nil {
		receipt <- response
	}

	if success {
		atomic.AddInt64(&this.sent, 1)
		return
	}

	errText := "未知错误"
	if response != nil && response.Error != nil {
		errText = response.Error.Error()
	}

	this.mu.Lock()
	maxAttempts, retryDelay := this.cfg.maxAttempts, this.cfg.retryDelay
	this.mu.Unlock()

	// 失败次数过多：标记失败并丢弃，避免死循环
	if task.Attempts >= maxAttempts {
		atomic.AddInt64(&this.failed, 1)
		Log.Error(map[string]any{
			"type":      task.Kind,
			"recipient": task.Recipient,
			"attempts":  task.Attempts,
			"error":     errText,
		}, "邮件发送失败，已达最大尝试次数，标记失败")
		smsLog("email", false, map[string]any{"recipient": task.Recipient, "type": task.Kind, "error": errText, "attempts": task.Attempts, "event": "已放弃"})
		return
	}

	// 延迟重试：重新入队（保留优先级，Next 控制可发送时间）
	atomic.AddInt64(&this.retries, 1)
	task.Next = time.Now().Add(retryDelay)

	this.mu.Lock()
	this.tasks = append(this.tasks, task)
	this.mu.Unlock()

	Log.Warn(map[string]any{
		"type":        task.Kind,
		"recipient":   task.Recipient,
		"attempts":    task.Attempts,
		"retry_after": retryDelay.String(),
		"error":       errText,
	}, "邮件发送失败，稍后重试")
	smsLog("email", false, map[string]any{"recipient": task.Recipient, "type": task.Kind, "error": errText, "attempts": task.Attempts, "event": "待重试"})

	this.wake()
}

// dispatch 实际发送（带超时与 panic 兜底）
//
// 单独起一个协程执行发送：SMTP 卡死时 worker 不会一直被占住，
// 超时按失败处理，交给重试逻辑。
func (this *mailQueueStruct) dispatch(task *MailTask) *SMSResponse {
	// 先取一次实例：配置热更新会重建 GoMail，避免执行期间读到 nil
	mail := GoMail
	if mail == nil {
		return &SMSResponse{Error: errors.New("邮件服务未初始化，请检查config/sms.toml配置")}
	}

	this.mu.Lock()
	timeout := this.cfg.sendTimeout
	this.mu.Unlock()

	done := make(chan *SMSResponse, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- &SMSResponse{Error: fmt.Errorf("邮件发送异常：%v", r)}
			}
		}()

		done <- mail.sendMailTask(task)
	}()

	select {
	case response := <-done:
		return response
	case <-time.After(timeout):
		return &SMSResponse{Error: fmt.Errorf("邮件发送超时（%s）", timeout)}
	}
}
