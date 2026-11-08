package facade

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

/**
 * 百度智能云 · 内容安全（图像审核）
 *
 * 两种送审方式：
 *   - imgURL：图片能被公网访问时直接传地址（百度自己拉图，最省流量）；
 *   - image：传 base64（本地存储 / 内网地址 / 未配公网域名的场景）。
 *
 * 为什么优先 URL：base64 会让请求体放大 1/3，且百度对 base64 有 4MB 上限；
 * 但本地部署时域名常常是 localhost 或内网地址，百度拉不到，只能回退 base64。
 */

const (
	baiduImageAuditURL = "https://aip.baidubce.com/rest/2.0/solution/v1/img_censor/v2/user_defined"
	// baiduImageMaxBytes 百度 base64 上传上限 4MB
	baiduImageMaxBytes = 4 << 20
)

// BaiduImageAudit - 图像审核
//
// imgURL  附件对外的访问地址（可能不可公网访问）
// savePath 附件在存储里的相对路径（imgURL 不可用时据此读本地文件）
func BaiduImageAudit(imgURL string, savePath string) (BaiduAuditResult, error) {

	result := BaiduAuditResult{Conclusion: BaiduAuditPass}

	if !AuditEnabled() {
		return result, errors.New("云端审核未启用")
	}
	if baiduBreakerOpen() {
		return result, errors.New("云端审核熔断中，稍后重试")
	}

	setting := AuditBaiduSetting()

	// 送审参数只准备一次：4MB 图片转 base64 不便宜，不该跟着零售重试反复编码。
	// 本地读不到图不是百度的问题，直接返回、不计熔断
	form, paramName, err := baiduImageForm(imgURL, savePath)
	if err != nil {
		return result, err
	}
	if setting.ImageStrategy != "" {
		form.Set("strategyId", setting.ImageStrategy)
	}

	censored, err := baiduDoWithRetry(func() (BaiduAuditResult, error) {
		return baiduImageCensorOnce(setting, form, paramName)
	})
	if err != nil {
		baiduBreakerByError(err)
		return result, err
	}
	baiduBreakerOK()

	return censored, nil
}

// baiduImageForm - 组装送审参数（imgUrl 与 image 严格二选一）
//
// 参数名必须是 imgUrl（官方文档的写法）：image 与 imgUrl 是互斥的两个入参，
// 同时传会报 282802、都不传会报 282801。
// 第二个返回值是本次走的是哪条路径，失败时带进日志便于定位。
func baiduImageForm(imgURL string, savePath string) (url.Values, string, error) {

	form := url.Values{}

	if imagePublicURL(imgURL) {
		form.Set("imgUrl", imgURL)
		return form, "imgUrl", nil
	}

	encoded, err := localImageBase64(savePath)
	if err != nil {
		return nil, "", err
	}
	form.Set("image", encoded)

	return form, "image", nil
}

// baiduImageCensorOnce - 调一次图像审核（不含重试）
func baiduImageCensorOnce(setting AuditBaiduSettingStruct, form url.Values, paramName string) (BaiduAuditResult, error) {

	result := BaiduAuditResult{Conclusion: BaiduAuditPass}

	// token 放在 once 里取：鉴权失败重试时能拿到换新后的 token
	token, err := baiduAccessToken(setting)
	if err != nil {
		return result, err
	}

	// 图片比文本慢：大图 + base64 上行，给足时间（下限 10 秒）
	timeout := setting.Timeout * 3
	if timeout < 10*time.Second {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	endpoint := baiduImageAuditURL + "?access_token=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := baiduHTTPClient.Do(req)
	if err != nil {
		return result, fmt.Errorf("百度图像审核请求失败：%w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return result, fmt.Errorf("百度图像审核读取响应失败：%w", err)
	}

	var payload baiduCensorPayload

	if err := json.Unmarshal(body, &payload); err != nil {
		return result, fmt.Errorf("百度图像审核响应解析失败：%w", err)
	}

	if payload.ErrorCode != 0 {
		// 带上本次入参类型与 log_id：282801 / 282802 都是「参数个数不对」，
		// 有这两项才能判断是代码传错、图片地址拉不到，还是文件本身不是图片
		return result, BaiduAuditError{
			Code:    payload.ErrorCode,
			Message: payload.ErrorMsg,
			Detail:  fmt.Sprintf("入参：%s，log_id：%d", paramName, payload.LogId),
		}
	}

	result.Conclusion = normalizeAuditConclusion(payload.Conclusion, payload.ConclusionType)
	result.Raw = string(body)

	for _, item := range payload.Data {
		itemConclusion := normalizeAuditConclusion(item.Conclusion, item.ConclusionType)
		rank := auditConclusionRank(itemConclusion)
		if rank > auditConclusionRank(result.Conclusion) {
			result.Conclusion = itemConclusion
			result.Messages = nil
			result.AuditType = 0
			result.AuditSubType = 0
		}
		// 图像审核没有 hits，判定说明就是 msg（如「存在女上露点不合规」）
		if rank >= auditConclusionRank(result.Conclusion) {
			result.Messages = appendAuditUnique(result.Messages, item.Msg)
			// 维度取最终结论对应的第一条（如 43 涉黄 / 430201 女上露点）
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

// imagePublicURL - 图片地址是否可能被公网拉取
//
// 本地 / 内网 / 回环地址一律走 base64：百度从公网访问不到，
// 直接传 URL 会得到一个「拉取失败」的错误码，看起来像审核故障。
func imagePublicURL(imgURL string) bool {

	value := strings.ToLower(strings.TrimSpace(imgURL))
	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		return false
	}

	blocked := []string{
		"localhost", "127.0.0.1", "0.0.0.0", "[::1]",
		"192.168.", "10.", "169.254.",
	}
	for _, item := range blocked {
		if strings.Contains(value, item) {
			return false
		}
	}

	// 172.16.0.0 ~ 172.31.255.255
	for i := 16; i <= 31; i++ {
		if strings.Contains(value, fmt.Sprintf("172.%d.", i)) {
			return false
		}
	}

	return true
}

// localImageBase64 - 读本地图片转 base64（兼容带 / 不带 public 前缀的路径）
func localImageBase64(savePath string) (string, error) {

	path := strings.TrimSpace(strings.TrimPrefix(savePath, "/"))
	if path == "" {
		return "", errors.New("图片路径为空，无法送审")
	}

	candidates := []string{path}
	if strings.HasPrefix(path, "public/") {
		candidates = append(candidates, strings.TrimPrefix(path, "public/"))
	} else {
		candidates = append(candidates, "public/"+path)
	}

	for _, item := range candidates {

		info, err := os.Stat(item)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Size() <= 0 {
			// 0 字节会编码成空串，百度那边等同「没传 image」→ 282801
			return "", errors.New("图片为空（0 字节），无法送审")
		}
		if info.Size() > baiduImageMaxBytes {
			return "", errors.New("图片超过 4MB，无法送审")
		}

		data, err := os.ReadFile(item)
		if err != nil {
			continue
		}

		return base64.StdEncoding.EncodeToString(data), nil
	}

	return "", fmt.Errorf("读取本地图片失败：%s", path)
}
