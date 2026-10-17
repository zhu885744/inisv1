package facade

import (
	"fmt"
	"os"

	"github.com/unti-io/go-utils/utils"
)

const (
	ConfigNameCrypt = "crypt"
)

// CryptToml - Crypt配置文件
var CryptToml *utils.ViperResponse

func init() {
	// 本文件按文件名字典序排在 log.go 之前，而 crypt.toml 缺失/不可读时会输出告警日志
	//（全新环境首次启动必然走到该分支），此时 Log 还是 nil，直接调用会 panic。
	// 先确保日志就绪再继续。
	ensureLogReady()

	initCryptToml()

	WatchConfigChange(CryptToml, initCrypt)
}

// initCryptToml - 初始化 Crypt配置文件
//
// 项目已彻底移除 JWT，这里只剩登录会话的有效期（session.expire）。
// 老站点的 crypt.toml 里还是 [jwt] 段，读取时由 SessionExpire() 兼容，
// 管理员在后台保存一次配置后会自动改写成 [session]。
func initCryptToml() {

	// 配置文件路径
	filePath := fmt.Sprintf("%s/%s.%s", ConfigPath, ConfigNameCrypt, ModeToml)

	// 自愈：若该路径被误建为目录（如误操作、解压事故），配置将永远无法写入
	if info, err := os.Stat(filePath); err == nil && info.IsDir() {
		if err := os.RemoveAll(filePath); err != nil {
			Log.Error(map[string]any{
				"error":     err,
				"path":      filePath,
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line}, "crypt.toml 路径被占用为目录且无法自动清理！")
		}
	}

	// 文件不存在时按模板生成（只含登录会话配置）
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		content := utils.Replace(TempCrypt, map[string]any{
			"${session.expire}": DefaultSessionExpire,
		})
		if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
			Log.Error(map[string]any{
				"error":     err,
				"func_name": utils.Caller().FuncName,
				"file_name": utils.Caller().FileName,
				"file_line": utils.Caller().Line}, "Crypt配置文件写入失败")
		}
	}

	item := utils.Viper(utils.ViperModel{
		Path: ConfigPath,
		Mode: ModeToml,
		Name: ConfigNameCrypt,
	}).Read()

	if item.Error != nil {
		Log.Error(map[string]any{
			"error":     item.Error,
			"func_name": utils.Caller().FuncName,
			"file_name": utils.Caller().FileName,
			"file_line": utils.Caller().Line}, "Crypt配置初始化错误")
	}

	CryptToml = &item
}

func initCrypt() {
	// 配置热更新：会话有效期只影响「新登录 / 续期的会话」，已签发的会话保持原 TTL
	if CryptToml == nil || CryptToml.Viper == nil {
		return
	}

	// 重新读取配置文件，否则 SessionExpire() 会一直读到启动时的旧值
	if err := CryptToml.Viper.ReadInConfig(); err == nil {
		CryptToml.Result = CryptToml.Viper.AllSettings()
	}

	Log.Info(map[string]any{
		"session.expire": SessionExpire(),
	}, "crypt 配置热更新（登录会话有效期）")
}
