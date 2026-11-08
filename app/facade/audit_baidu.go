package facade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/**
 * 百度智能云 · 内容安全（文本审核）
 *
 * 接口说明：
 *   - 先用 AK/SK 换 access_token（有效期 30 天），token 存缓存并提前刷新；
 *   - 文本审核单次上限 2000 字，超长正文按段提交（最多 5 段，约 1 万字）；
 *   - 结论三态：pass / review / block，调用方据此决定放行 / 转人工 / 驳回。
 *
 * 稳定性取舍：
 *   - 所有请求都带 context 超时（默认 3 秒），绝不拖住业务协程；
 *   - 失败按错误码分类处理（见 BaiduAuditError）：
 *       · token 无效 / 过期（110 / 111）→ 清缓存换新 token 后重试；
 *       · QPS 限流（18）、5xx → 退避后重试；
 *       · 参数类 / 配额用尽 → 不重试（重试也是同一结果，只是白烧额度）；
 *     重试上限 3 次，避免把请求耗在超时上；
 *   - 只有「百度侧不可用」（限流 / 配额 / 5xx / 网络故障）才计入熔断，
 *     连续 5 次熔断 60 秒；token 失效与参数错误不计入，否则一次可自愈的
 *     token 轮换就能把审核链路整体拉闸；
 *   - 审核失败只返回 error，由调用方决定忽略 —— 本地词库已经做过同步快筛，
 *     机审只是兜底，不能因为第三方不可用而影响用户正常发布。
 */

const (
	// AuditProviderBaidu - 百度内容安全
	AuditProviderBaidu = "baidu"

	// 审核结论（与百度返回一致，调用方只需认这几个值）
	//
	// failed 对应百度的「审核失败」（conclusionType = 4）：底层服务异常，
	// 此时**不能**当成疑似处理 —— 否则第三方一抖动，正常内容就会被批量打进待审。
	BaiduAuditPass   = "pass"
	BaiduAuditReview = "review"
	BaiduAuditBlock  = "block"
	BaiduAuditFailed = "failed"

	baiduTokenURL     = "https://aip.baidubce.com/oauth/2.0/token"
	baiduTextAuditURL = "https://aip.baidubce.com/rest/2.0/solution/v1/text_censor/v2/user_defined"

	baiduTokenCacheKey = "audit:baidu:token"

	// baiduTextChunkRunes - 百度文本审核单次上限（字符数）
	baiduTextChunkRunes = 2000
	// baiduTextMaxChunks - 超长正文最多审这么多段，再长只审前面部分
	baiduTextMaxChunks = 5

	// baiduBreakerFailLimit / baiduBreakerPause - 连续失败多少次后暂停多久
	baiduBreakerFailLimit = 5
	baiduBreakerPause     = 60 * time.Second
)

// BaiduAuditResult - 一次审核的结果
type BaiduAuditResult struct {
	// Conclusion pass / review / block
	Conclusion string
	// Words 命中的违规词（百度只返回命中词，不一定完整）
	Words []string
	// Messages 百度给出的判定说明（data[].msg），如「存在女上露点不合规」
	//
	// 这是最贴近人工审核措辞的文案，直接展示给作者比「命中：xxx」更清楚，
	// 所以审核原因优先用它，拿不到才回退到命中词。
	Messages []string
	// AuditType / AuditSubType 命中的审核维度（百度的 type / subType）
	//
	// 如 33 涉黄 / 430201 女上露点。写进 sensitive_hits 后，后台可以按维度
	// 统计「近 7 天涉黄类命中多少条」，不用再解析文案。
	AuditType    int
	AuditSubType int
	// Raw 原始响应（排障用，不入库）
	Raw string
}

// 百度错误码（只列需要区别对待的几个，其余按「本端参数错误」处理）
const (
	// baiduErrTokenInvalid - access_token 无效（多半被别处重置了）
	baiduErrTokenInvalid = 110
	// baiduErrTokenExpired - access_token 过期（有效期 30 天）
	baiduErrTokenExpired = 111
	// baiduErrQPSLimit - QPS 超限，退避一下通常就能过
	baiduErrQPSLimit = 18
	// baiduErrQuotaOpenAPI / baiduErrQuotaDaily / baiduErrQuotaTotal - 配额用尽
	baiduErrQuotaOpenAPI = 4
	baiduErrQuotaDaily   = 17
	baiduErrQuotaTotal   = 19
)

// BaiduAuditError - 百度接口返回的业务错误（error_code != 0）
//
// 以前用 fmt.Errorf 拼字符串，上层拿不到错误码：token 过期、QPS 限流、
// 入参写错全都长得一样，只能一律当「服务故障」处理 —— 一次 token 失效
// 就能把熔断打开 60 秒。改成结构化错误后，重试与熔断都能按性质区分。
type BaiduAuditError struct {
	// Code 百度 error_code
	Code int
	// Message 百度 error_msg
	Message string
	// Detail 排障用的补充信息（入参类型、log_id 等），不参与分类
	Detail string
}

// Error - 实现 error 接口
func (this BaiduAuditError) Error() string {
	if this.Detail == "" {
		return fmt.Sprintf("百度审核失败：%d %s", this.Code, this.Message)
	}
	return fmt.Sprintf("百度审核失败：%d %s（%s）", this.Code, this.Message, this.Detail)
}

// Auth - 鉴权类：token 无效 / 过期，换新 token 后即可恢复
func (this BaiduAuditError) Auth() bool {
	return this.Code == baiduErrTokenInvalid || this.Code == baiduErrTokenExpired
}

// Throttled - 限流类：QPS 超限，退避后重试有意义
func (this BaiduAuditError) Throttled() bool {
	return this.Code == baiduErrQPSLimit
}

// QuotaExhausted - 配额用尽：今天再调也是同一个结果，重试纯属浪费
func (this BaiduAuditError) QuotaExhausted() bool {
	switch this.Code {
	case baiduErrQuotaOpenAPI, baiduErrQuotaDaily, baiduErrQuotaTotal:
		return true
	}
	return false
}

// ServerSide - 这次失败是否说明「百度侧不可用」
//
// 只有这类错误才该计入熔断：限流、配额用尽、5xx。
// token 失效（换新即好）和参数错误（本端 bug）都不代表服务故障。
func (this BaiduAuditError) ServerSide() bool {
	return this.Throttled() || this.QuotaExhausted() || (this.Code >= 500 && this.Code < 600)
}

// Retryable - 是否值得重试：换一下条件就可能成功的才重试
//
// 参数类错误（282xxx / 216xxx）重试一百次也是同一结果，
// 配额用尽则要等到明天，两者都不在重试范围内。
func (this BaiduAuditError) Retryable() bool {
	return this.Auth() || this.ServerSide()
}

// RetryDelay - 重试前的等待（鉴权类只是换个 token，不必等）
func (this BaiduAuditError) RetryDelay(attempt int) time.Duration {
	if this.Auth() {
		return 0
	}
	return time.Duration(attempt) * 200 * time.Millisecond
}

// baiduCensorItem - 单条命中项（文本 / 图像审核结构一致）
type baiduCensorItem struct {
	Conclusion     string `json:"conclusion"`
	ConclusionType int    `json:"conclusionType"`
	Type           int    `json:"type"`
	SubType        int    `json:"subType"`
	Msg            string `json:"msg"`
	Hits           []struct {
		Words []string `json:"words"`
	} `json:"hits"`
}

// baiduCensorPayload - 文本 / 图像审核共用的响应体
type baiduCensorPayload struct {
	Conclusion string `json:"conclusion"`
	// ConclusionType 数值结论：1 合规 / 2 不合规 / 3 疑似（比中文文案稳定）
	ConclusionType int               `json:"conclusionType"`
	Data           []baiduCensorItem `json:"data"`
	ErrorCode      int               `json:"error_code"`
	ErrorMsg       string            `json:"error_msg"`
	LogId          int64             `json:"log_id"`
}

// baiduHTTPClient - 复用连接；超时由 context 控制，这里不设 Client.Timeout
var baiduHTTPClient = &http.Client{}

// token 的进程内缓存（缓存驱动不可用时也能工作）
var (
	baiduTokenMu     sync.Mutex
	baiduToken       string
	baiduTokenExpire time.Time
)

// 熔断状态
var (
	baiduFailStreak atomic.Int64
	baiduPauseUntil atomic.Int64
)

// BaiduTextAudit - 文本审核入口（自动分段、取最严重结论）
func BaiduTextAudit(text string) (BaiduAuditResult, error) {

	result := BaiduAuditResult{Conclusion: BaiduAuditPass}

	if !AuditEnabled() {
		return result, errors.New("云端审核未启用")
	}

	if baiduBreakerOpen() {
		return result, errors.New("云端审核熔断中，稍后重试")
	}

	chunks := splitAuditText(text, baiduTextChunkRunes, baiduTextMaxChunks)
	if len(chunks) == 0 {
		// 纯空白内容没有审的必要，直接放行
		return result, nil
	}

	for _, chunk := range chunks {

		one, err := baiduTextCensor(chunk)
		if err != nil {
			baiduBreakerByError(err)
			return result, err
		}
		baiduBreakerOK()

		result.Raw = one.Raw
		if auditConclusionRank(one.Conclusion) > auditConclusionRank(result.Conclusion) {
			result.Conclusion = one.Conclusion
			result.Words = one.Words
		}

		// 已经是最高风险，后面的段不必再花钱审
		if result.Conclusion == BaiduAuditBlock {
			break
		}
	}

	return result, nil
}

// baiduRetryLimit - 一次审核最多尝试几次（含首发）
const baiduRetryLimit = 3

// baiduDoWithRetry - 按错误性质决定要不要重试
//
// 只做有限次重试：token 轮换、QPS 抖动这类问题一两次就能过去；
// 再重下去只会把请求耗在超时上，把业务协程拖住。
func baiduDoWithRetry[T any](call func() (T, error)) (T, error) {

	for attempt := 1; ; attempt++ {

		value, err := call()
		if err == nil {
			return value, nil
		}

		var auditErr BaiduAuditError
		if !errors.As(err, &auditErr) || !auditErr.Retryable() || attempt >= baiduRetryLimit {
			return value, err
		}

		// 110 / 111：缓存里的 token 已失效（被别处重置或到期），
		// 清掉重取一次即可恢复，没必要当成服务故障
		if auditErr.Auth() {
			resetBaiduToken()
		}
		if delay := auditErr.RetryDelay(attempt); delay > 0 {
			time.Sleep(delay)
		}

		Log.Warn(map[string]any{
			"error":   err,
			"code":    auditErr.Code,
			"attempt": attempt,
		}, "百度审核失败，按错误类型重试")
	}
}

// baiduTextCensor - 审一段文本（token 失效 / 限流时自动重试）
func baiduTextCensor(text string) (BaiduAuditResult, error) {
	return baiduDoWithRetry(func() (BaiduAuditResult, error) {
		return baiduTextCensorOnce(text)
	})
}

// baiduTextCensorOnce - 审一段文本（不含重试）
func baiduTextCensorOnce(text string) (BaiduAuditResult, error) {

	setting := AuditBaiduSetting()

	token, err := baiduAccessToken(setting)
	if err != nil {
		return BaiduAuditResult{}, err
	}

	form := url.Values{}
	form.Set("text", text)
	if setting.TextStrategy != "" {
		form.Set("strategyId", setting.TextStrategy)
	}

	ctx, cancel := context.WithTimeout(context.Background(), setting.Timeout)
	defer cancel()

	endpoint := baiduTextAuditURL + "?access_token=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return BaiduAuditResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := baiduHTTPClient.Do(req)
	if err != nil {
		return BaiduAuditResult{}, fmt.Errorf("百度文本审核请求失败：%w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return BaiduAuditResult{}, fmt.Errorf("百度文本审核读取响应失败：%w", err)
	}

	var payload baiduCensorPayload

	if err := json.Unmarshal(body, &payload); err != nil {
		return BaiduAuditResult{}, fmt.Errorf("百度文本审核响应解析失败：%w", err)
	}

	if payload.ErrorCode != 0 {
		return BaiduAuditResult{}, BaiduAuditError{
			Code:    payload.ErrorCode,
			Message: payload.ErrorMsg,
			Detail:  fmt.Sprintf("log_id：%d", payload.LogId),
		}
	}

	result := BaiduAuditResult{
		Conclusion: normalizeAuditConclusion(payload.Conclusion, payload.ConclusionType),
		Raw:        string(body),
	}

	// 顶层没给结论时，从各命中项里取最严重的一个；
	// msg 跟着结论走：换了更严重的结论就把之前较轻的说明丢掉
	for _, item := range payload.Data {
		itemConclusion := normalizeAuditConclusion(item.Conclusion, item.ConclusionType)
		rank := auditConclusionRank(itemConclusion)
		if rank > auditConclusionRank(result.Conclusion) {
			result.Conclusion = itemConclusion
			result.Messages = nil
			result.AuditType = 0
			result.AuditSubType = 0
		}
		if rank >= auditConclusionRank(result.Conclusion) {
			result.Messages = appendAuditUnique(result.Messages, item.Msg)
			// 维度取最终结论对应的第一条（同一结论可能有多个维度命中）
			if result.AuditType == 0 && item.Type > 0 {
				result.AuditType = item.Type
				result.AuditSubType = item.SubType
			}
		}
		for _, hit := range item.Hits {
			result.Words = append(result.Words, hit.Words...)
		}
	}

	return result, nil
}

// baiduAccessToken - 取 access_token（进程内 → 缓存 → 换新）
func baiduAccessToken(setting AuditBaiduSettingStruct) (string, error) {

	baiduTokenMu.Lock()
	defer baiduTokenMu.Unlock()

	if baiduToken != "" && time.Now().Before(baiduTokenExpire) {
		return baiduToken, nil
	}

	if Cache != nil {
		// 缓存里的是别的实例换好的 token：先信它，但进程内只记 10 分钟，
		// 万一它其实已失效，最多 10 分钟后重新换一次，不会一直失败
		if value, ok := Cache.Get(baiduTokenCacheKey).(string); ok && value != "" {
			baiduToken = value
			baiduTokenExpire = time.Now().Add(10 * time.Minute)
			return value, nil
		}
	}

	if setting.ApiKey == "" || setting.SecretKey == "" {
		return "", errors.New("百度审核 AK/SK 未配置")
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", setting.ApiKey)
	form.Set("client_secret", setting.SecretKey)

	ctx, cancel := context.WithTimeout(context.Background(), setting.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baiduTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := baiduHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("百度 access_token 请求失败：%w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", fmt.Errorf("百度 access_token 读取响应失败：%w", err)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		ExpiresIn        int64  `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("百度 access_token 响应解析失败：%w", err)
	}

	if payload.AccessToken == "" {
		return "", fmt.Errorf("百度 access_token 获取失败：%s %s", payload.Error, payload.ErrorDescription)
	}

	// 提前 5 分钟过期，避免临界点上刚好用到失效 token
	ttl := payload.ExpiresIn - 300
	if ttl < 60 {
		ttl = 60
	}

	baiduToken = payload.AccessToken
	baiduTokenExpire = time.Now().Add(time.Duration(ttl) * time.Second)

	if Cache != nil {
		Cache.Set(baiduTokenCacheKey, payload.AccessToken, time.Duration(ttl)*time.Second)
	}

	return payload.AccessToken, nil
}

// resetBaiduToken - 丢弃缓存的 access_token，下次调用重新换取
//
// 百度返回 110 / 111 时，缓存里的 token 已经不能用了；不清掉的话
// 要等进程内 TTL（或缓存里那 10 分钟）到期才会恢复，这段时间内
// 每次审核都会带着同一个坏 token 再去撞一次墙。
func resetBaiduToken() {

	baiduTokenMu.Lock()
	baiduToken = ""
	baiduTokenExpire = time.Time{}
	baiduTokenMu.Unlock()

	// 缓存里那份是别的实例也可能在读的副本，一并删掉，
	// 否则本进程重取之后，下次还是可能从缓存里捞回这个坏值
	if Cache != nil {
		Cache.Del(baiduTokenCacheKey)
	}
}

// appendAuditUnique - 收集判定说明（data[].msg）
//
// 百度可能对同一张图 / 一段文本给出多条说明，这里去重、每条截断、最多 5 条：
// 原因字段最终只留给作者看一两句，堆太多反而看不清重点。
func appendAuditUnique(list []string, value string) []string {

	value = truncateAuditText(strings.TrimSpace(value), 120)
	if value == "" {
		return list
	}

	for _, item := range list {
		if item == value {
			return list
		}
	}
	if len(list) >= 5 {
		return list
	}

	return append(list, value)
}

// truncateAuditText - 按字符截断（不切断 UTF-8）
func truncateAuditText(value string, size int) string {
	runes := []rune(value)
	if size <= 0 || len(runes) <= size {
		return value
	}
	return string(runes[:size])
}

// normalizeAuditConclusion - 归一化结论
//
// 百度固定返回 pass / review / block；未知值按 review 处理（宁可进人工，
// 也不把可能违规的内容当成通过）。
func normalizeAuditConclusion(value string, conclusionType int) string {

	// 百度 v2 接口在部分策略下返回的是中文结论（合规 / 不合规 / 疑似），
	// 只认 pass / review / block 会把「不合规」误判成疑似 → 违规内容不进驳回队列
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BaiduAuditPass, "合规":
		return BaiduAuditPass
	case BaiduAuditBlock, "不合规":
		return BaiduAuditBlock
	case BaiduAuditReview, "疑似":
		return BaiduAuditReview
	case BaiduAuditFailed, "审核失败":
		return BaiduAuditFailed
	}

	// 文案没认出来（换了策略 / 语言）时用数值兜底：
	// 1 合规 / 2 不合规 / 3 疑似 / 4 审核失败
	switch conclusionType {
	case 1:
		return BaiduAuditPass
	case 2:
		return BaiduAuditBlock
	case 3:
		return BaiduAuditReview
	case 4:
		return BaiduAuditFailed
	}

	// 两者都没有：按疑似处理（宁可进人工，也不把可能违规的内容当成通过）
	return BaiduAuditReview
}

// auditConclusionRank - 结论严重程度（用于多段结果取最严重）
func auditConclusionRank(value string) int {
	switch value {
	case BaiduAuditBlock:
		return 3
	case BaiduAuditReview:
		return 2
	case BaiduAuditPass:
		return 1
	case BaiduAuditFailed:
		// 审核失败不参与「取最严重结论」的比较：同一次响应里只要有一项判了
		// 不合规，最终结论就该是不合规，不能被一项 failed 覆盖
		return 0
	default:
		return 0
	}
}

// splitAuditText - 按字符数切段（不切断 UTF-8 字符），最多 max 段
func splitAuditText(text string, size int, max int) []string {

	runes := []rune(strings.TrimSpace(text))
	if len(runes) == 0 || size <= 0 || max <= 0 {
		return nil
	}

	chunks := make([]string, 0, max)
	for start := 0; start < len(runes) && len(chunks) < max; start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// baiduBreakerOpen - 熔断是否生效
func baiduBreakerOpen() bool {
	return time.Now().Unix() < baiduPauseUntil.Load()
}

// baiduBreakerByError - 按错误性质决定这次失败要不要计入熔断
//
// 熔断的本意是「百度整体不可用，别再往上堆请求了」，所以只该统计
// 对方侧的故障：网络不可达、5xx、限流、配额用尽。
// token 失效已经在重试里换过新 token，参数错误是我们自己传错了，
// 这两类算进熔断，只会让一次本可自愈的问题把审核链路整体拉闸 60 秒。
func baiduBreakerByError(err error) {

	var auditErr BaiduAuditError
	if errors.As(err, &auditErr) && !auditErr.ServerSide() {
		Log.Warn(map[string]any{
			"error": err,
			"code":  auditErr.Code,
		}, "百度审核失败，但属于本端可自愈问题，不计入熔断")
		return
	}

	baiduBreakerFail()
}

// baiduBreakerFail - 记一次失败，达到阈值就暂停调用
func baiduBreakerFail() {
	if baiduFailStreak.Add(1) < baiduBreakerFailLimit {
		return
	}
	baiduFailStreak.Store(0)
	baiduPauseUntil.Store(time.Now().Add(baiduBreakerPause).Unix())
	Log.Error(map[string]any{
		"pause_seconds": int(baiduBreakerPause.Seconds()),
	}, "云端审核连续失败，已暂停调用")
}

// baiduBreakerOK - 成功一次就清零失败计数
func baiduBreakerOK() {
	baiduFailStreak.Store(0)
}
