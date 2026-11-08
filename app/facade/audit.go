package facade

import (
	"strings"
	"time"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * 云端内容审核配置（独立 config/audit.toml）
 *
 * 与 config/sensitive.toml 的分工：
 *   - sensitive.toml：本地词库，同步判定，免费、即时，覆盖明显的违规词；
 *   - audit.toml：第三方机审（当前接百度内容安全），异步兜底，
 *     覆盖变体词、长文尾部、图片等本地词库覆盖不到的情况。
 *
 * 为什么单独成文件：AK/SK 属于凭据，和运行参数（app.toml）、词库（sensitive.toml）
 * 分开存放便于按权限隔离与轮换；也让「关掉机审」只需改一个开关，不用动词库。
 */

// ConfigNameAudit - 云端审核配置文件
const ConfigNameAudit = "audit"

// AuditToml - config/audit.toml
var AuditToml *utils.ViperResponse

func initAuditToml() {

	item := utils.Viper(utils.ViperModel{
		Path:    ConfigPath,
		Mode:    ModeToml,
		Name:    ConfigNameAudit,
		Content: auditTomlContent(),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "云端审核配置初始化错误")
	}

	AuditToml = &item
}

func init() {
	// 与其它 loader 一致：本文件字典序晚于 log.go，但缺文件 / 解析失败的分支要打日志，
	// 显式确保一次更稳（见 sensitive.go 的同样处理）
	ensureLogReady()

	initAuditToml()
	WatchConfigChange(AuditToml, func() {
		// 配置里只有开关和凭据，没有需要预编译的规则：
		// 热更新后下一次审核请求直接读新值，这里只需保持 Result 快照一致
		refreshAuditResult()
	})
}

// refreshAuditResult - 让 Result 快照与磁盘一致（Get 本身是实时的）
func refreshAuditResult() {
	if AuditToml == nil || AuditToml.Viper == nil {
		return
	}
	if err := AuditToml.Viper.ReadInConfig(); err == nil {
		AuditToml.Result = cast.ToStringMap(AuditToml.Viper.AllSettings())
	}
}

// ReloadAuditToml - 后台保存配置后立即让内存与磁盘一致
func ReloadAuditToml() {
	if AuditToml == nil || AuditToml.Viper == nil {
		initAuditToml()
		return
	}
	refreshAuditResult()
}

func auditTomlContent() string {
	return utils.Replace(TempAudit, nil)
}

// AuditProvider - 当前机审服务商（小写；未配置返回空串）
func AuditProvider() string {
	if AuditToml == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(cast.ToString(AuditToml.Get("provider", "baidu"))))
}

// AuditSwitch - 配置里的总开关原始值（不看凭据是否齐全）
//
// 后台表单需要区分「开关关了」和「开关开了但凭据没填」两种状态，
// 所以不能直接用 AuditEnabled。
func AuditSwitch() bool {
	if AuditToml == nil {
		return false
	}
	return cast.ToBool(AuditToml.Get("enabled", false))
}

// AuditEnabled - 云端审核是否可用
//
// 三个条件缺一不可：总开关打开、服务商受支持、该服务商的凭据已配齐。
// 凭据缺失时按「关闭」处理而不是报错：刚装好的站点默认就没有 AK/SK，
// 此时应当安静地退回本地词库，而不是每条内容都报一次配置错误。
func AuditEnabled() bool {

	if AuditToml == nil {
		return false
	}

	if !cast.ToBool(AuditToml.Get("enabled", false)) {
		return false
	}

	switch AuditProvider() {
	case AuditProviderBaidu:
		setting := AuditBaiduSetting()
		return setting.ApiKey != "" && setting.SecretKey != ""
	default:
		return false
	}
}

// AuditBaiduSetting - 百度内容安全的连接配置
type AuditBaiduSettingStruct struct {
	ApiKey        string
	SecretKey     string
	TextStrategy  string
	ImageStrategy string
	// Timeout 单次请求超时（含建连），默认 2 秒
	Timeout time.Duration
	// Workers 并发的审核协程数
	Workers int
	// QueueSize 待审队列长度（满了直接丢弃并记日志，绝不阻塞业务）
	QueueSize int
}

// AuditBaiduSetting - 读取百度审核配置（缺省值已归一化）
func AuditBaiduSetting() AuditBaiduSettingStruct {

	setting := AuditBaiduSettingStruct{
		// 官方说明「绝大部分请求模型处理时间在 2 秒内」，留出网络往返余量取 3 秒：
		// 超时按失败处理（不改内容状态），不会误伤内容
		Timeout:   3 * time.Second,
		Workers:   3,
		QueueSize: 256,
	}

	if AuditToml == nil {
		return setting
	}

	setting.ApiKey = strings.TrimSpace(cast.ToString(AuditToml.Get("baidu.api_key")))
	setting.SecretKey = strings.TrimSpace(cast.ToString(AuditToml.Get("baidu.secret_key")))
	setting.TextStrategy = strings.TrimSpace(cast.ToString(AuditToml.Get("baidu.text_strategy_id")))
	setting.ImageStrategy = strings.TrimSpace(cast.ToString(AuditToml.Get("baidu.image_strategy_id")))

	// 支持 "2" 与 "2 * 1000" 这类算式（与其它配置一致）
	if seconds := cast.ToInt64(utils.Calc(AuditToml.Get("baidu.timeout", 2))); seconds > 0 {
		setting.Timeout = time.Duration(seconds) * time.Second
	}
	if workers := cast.ToInt(AuditToml.Get("baidu.workers", 3)); workers > 0 && workers <= 32 {
		setting.Workers = workers
	}
	if size := cast.ToInt(AuditToml.Get("baidu.queue_size", 256)); size > 0 && size <= 8192 {
		setting.QueueSize = size
	}

	return setting
}
