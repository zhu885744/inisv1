package facade

import (
	"bytes"
	"strings"
	"sync/atomic"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ConfigNameSensitive - 敏感词库配置文件（独立于 app.toml）
const ConfigNameSensitive = "sensitive"

// 词库本体与单步回滚备份的落盘位置（保存 / 回滚 / 后台读取统一走这里）
const (
	sensitiveTomlFile   = ConfigPath + "/" + ConfigNameSensitive + "." + ModeToml
	sensitiveTomlBackup = sensitiveTomlFile + ".bak"
)

// SensitiveToml - config/sensitive.toml
//
// 词库是「部署期安全配置」，独立成文件而不是塞进 app.toml：
//   - app.toml 管运行参数，改动会被频繁保存，混在一起容易误删词条；
//   - 单独文件便于备份 / 回滚 / diff，也方便运维直接看规则；
//   - 保存后只重载这一份配置，不影响数据库、通知等其它配置。
var SensitiveToml *utils.ViperResponse

// sensitiveGeneration - 配置代次
//
// 每次文件变更 / 保存后 +1。model 层的规则编译（含 AC 自动机）按代次缓存，
// 代次变了才重建，避免每条消息都重新解析 TOML / 构建匹配树。
var sensitiveGeneration atomic.Int64

func initSensitiveToml() {
	item := utils.Viper(utils.ViperModel{
		Path:    ConfigPath,
		Mode:    ModeToml,
		Name:    ConfigNameSensitive,
		Content: sensitiveTomlContent(),
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line,
		}, "敏感词配置初始化错误")
	}

	SensitiveToml = &item
}

func init() {
	// 与其它 loader 一致：本文件字典序晚于 log.go，Log 理论上已就绪，
	// 但缺文件 / 解析失败的分支都要打日志，显式确保一次更稳
	ensureLogReady()

	initSensitiveToml()
	WatchConfigChange(SensitiveToml, func() {
		// WatchConfigChange 已让 viper 重新读文件；这里刷新快照并推进代次
		refreshSensitiveResult()
		sensitiveGeneration.Add(1)
	})
}

// refreshSensitiveResult - 让 Result 快照与磁盘一致
//
// Get 走底层 viper 是实时的，但 Result 是首次 Read 的快照；后台 / API 读取
// 统一用 Get，所以 Result 只影响调试输出，仍然同步一份避免误导。
func refreshSensitiveResult() {
	if SensitiveToml == nil || SensitiveToml.Viper == nil {
		return
	}
	if err := SensitiveToml.Viper.ReadInConfig(); err == nil {
		SensitiveToml.Result = cast.ToStringMap(SensitiveToml.Viper.AllSettings())
	}
}

// SensitiveGeneration - 当前配置代次（model 层据此决定是否重建匹配器）
func SensitiveGeneration() int64 {
	return sensitiveGeneration.Load()
}

// ReloadSensitiveToml - 保存敏感词配置后立即让内存与磁盘一致
//
// 不依赖文件监听的异步时序：保存接口写完文件后同步调用，下一次校验就能读到新规则。
func ReloadSensitiveToml() {
	if SensitiveToml == nil || SensitiveToml.Viper == nil {
		initSensitiveToml()
		sensitiveGeneration.Add(1)
		return
	}

	refreshSensitiveResult()
	sensitiveGeneration.Add(1)
}

// SensitiveFileContent - 读取敏感词配置文件原文（后台编辑器用）
func SensitiveFileContent() string {
	if !utils.File().Exist(sensitiveTomlFile) {
		return sensitiveTomlContent()
	}
	return string(utils.File().Byte(sensitiveTomlFile).Byte)
}

// SensitiveBackupContent - 读取回滚备份原文
//
// 备份不存在 / 为空都视为「没有可回滚的版本」，由调用方提示管理员。
func SensitiveBackupContent() (string, bool) {
	if !utils.File().Exist(sensitiveTomlBackup) {
		return "", false
	}
	content := utils.File().Byte(sensitiveTomlBackup).Byte
	if len(content) == 0 {
		return "", false
	}
	return string(content), true
}

// BackupSensitiveToml - 把当前词库文件备份一份（文件不存在时跳过）
//
// 只保留一份备份：词库的典型事故是「保存下去发现大面积误伤」，
// 单步回滚即可覆盖；多版本备份反而让管理员不知道该退到哪一版。
func BackupSensitiveToml() error {
	if !utils.File().Exist(sensitiveTomlFile) {
		return nil
	}
	content := utils.File().Byte(sensitiveTomlFile).Byte
	if len(content) == 0 {
		return nil
	}
	return utils.File().Save(bytes.NewReader(content), sensitiveTomlBackup).Error
}

// SaveSensitiveToml - 写入词库文件并立即重载内存
//
// 保存与回滚共用：写完必须 Reload，否则在文件监听的时延窗口内，
// 新规则已落盘但线上判定的还是旧词库，管理员会以为改动没生效。
func SaveSensitiveToml(content string) error {
	body := strings.TrimRight(content, "\r\n") + "\n"
	if err := utils.File().Save(strings.NewReader(body), sensitiveTomlFile).Error; err != nil {
		return err
	}
	ReloadSensitiveToml()
	return nil
}
