package facade

import (
	"fmt"
	"os"

	"github.com/unti-io/go-utils/utils"
)

// ============================== 启动自检：配置目录与配置文件 ==============================
//
// 场景：把编译好的二进制（前端已内嵌）单独拷到服务器运行时，config 目录可能整份都不存在。
//
// 为什么需要自己补：go-utils 的 viper.Read() 在「文件不存在」时并不会写入 Content
// （它的实现是 `if !os.IsNotExist(err) && !Is.Empty(Content)` 才写，与注释里的意图相反），
// 所以配置文件不会自动生成，只能一路报「配置初始化错误」。
//
// 这里在包内最早的 init（app.go 的 init）里补齐：
//   - 目录：config、config/i18n、runtime、runtime/cache
//   - 配置文件：app / cache / log / sms / storage（内容取自 template.go 的模板）
//
// 注意：**不**建根目录的 storage —— 见下面 ensureBootstrap 里的说明。
//
// 两个刻意不生成的文件：
//   - database.toml：由安装向导写入，提前生成空文件会让「先配好数据库才能完成安装」的校验失效；
//   - crypt.toml：JWT 密钥必须随机生成并持久化，由 crypt.go 自己读写（这里只要保证 config 目录存在）。

// 其余配置文件名（app / cache / crypt 三个已在各自文件里定义）
const (
	ConfigNameLog     = "log"
	ConfigNameSMS     = "sms"
	ConfigNameStorage = "storage"
)

// 安装状态相关文件（安装向导、中间件、启动自检共用同一口径）
const (
	// InstallLockFile 安装锁：存在表示尚未完成安装
	InstallLockFile = "install.lock"
	// DatabaseConfigFile 数据库配置：由安装向导的第一步写入
	DatabaseConfigFile = "config/database.toml"
)

// Installed - 是否已完成安装
//
// 判定口径：安装锁已解除（install.lock 不存在）**且**数据库配置已生成。
// 两个条件缺一不可：
//   - 只把二进制拷到服务器（连 install.lock 都没有）时会被判为「未安装」，引导去安装向导；
//   - 已安装的站点若 database.toml 丢失，同样回到安装向导，而不是带着空 DSN 启动后各种报错。
func Installed() bool {
	return !utils.File().Exist(InstallLockFile) && utils.File().Exist(DatabaseConfigFile)
}

// ensureBootstrap - 确保目录与配置文件就绪（幂等）
//
// 这里刻意**不建**根目录的 storage：storage.toml 里 [local] path = "storage" 指的是
// 「public 下的子目录」，本地存储驱动写文件时会自己拼上前缀 public/ ——
// 实际的上传目录是 public/storage/…（见 storage.go 的 LocalStorageStruct.Path）。
// 而且 go-utils 的 File().Save 会自动创建父目录，也不需要预建。
// 之前这里建出来的根目录 storage/ 是空的，没有任何代码读它，删除即可。
func ensureBootstrap() {
	for _, dir := range []string{
		ConfigPath,
		ConfigPath + "/i18n",
		"runtime",
		"runtime/cache", // cache.toml 里 [file] path 的默认值
	} {
		ensureDir(dir)
	}

	ensureConfigFile(ConfigNameApp, appTomlContent())
	ensureConfigFile(ConfigNameCache, cacheTomlContent())
	ensureConfigFile(ConfigNameLog, logTomlContent())
	ensureConfigFile(ConfigNameSMS, smsTomlContent())
	ensureConfigFile(ConfigNameStorage, storageTomlContent())
}

// ensureDir - 目录不存在就创建
//
// 这里只能用 fmt 输出：本函数在 ensureLogReady 之前执行，此时 Log 仍为 nil。
func ensureDir(dir string) {
	if utils.File().Exist(dir) {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("[inis] 创建目录失败：%s（%v）\n", dir, err)
	}
}

// ensureConfigFile - 配置文件不存在就用模板生成一份（已存在的不覆盖）
func ensureConfigFile(name string, content string) {
	path := fmt.Sprintf("%s/%s.%s", ConfigPath, name, ModeToml)

	if utils.File().Exist(path) {
		return
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Printf("[inis] 生成配置文件失败：%s（%v）\n", path, err)
		return
	}

	fmt.Printf("[inis] 已生成默认配置文件：%s\n", path)
}

// ---------------------------------------------------------------------------
// 以下是各配置文件的模板内容：与对应的 loader 共用同一份，避免两处维护
// ---------------------------------------------------------------------------

// appTomlContent - app.toml 内容
func appTomlContent() string {
	return utils.Replace(TempApp, nil)
}

// cacheTomlContent - cache.toml 内容
func cacheTomlContent() string {
	return utils.Replace(TempCache, map[string]any{
		"${open}":           "false",
		"${default}":        DefaultCacheDriver,
		"${local.expire}":   300,
		"${redis.host}":     "localhost",
		"${redis.port}":     "6379",
		"${redis.password}": "",
		"${redis.expire}":   "2 * 60 * 60",
		"${redis.prefix}":   "inis:",
		"${redis.database}": 0,
		"${file.expire}":    "2 * 60 * 60",
		"${file.path}":      "runtime/cache",
		"${file.prefix}":    "inis_",
		"${ram.expire}":     "2 * 60 * 60",
	})
}

// logTomlContent - log.toml 内容
func logTomlContent() string {
	return utils.Replace(TempLog, map[string]any{
		"${on}":      "true",
		"${size}":    2,
		"${age}":     7,
		"${backups}": 20,
	})
}

// storageTomlContent - storage.toml 内容
func storageTomlContent() string {
	return utils.Replace(TempStorage, map[string]any{
		"${default}": "local",
		// 本地域名留空：full_url 存相对路径 /storage/xxx，直接由站点静态服务提供；
		// 之前默认成 "storage" 会拼出 storage/storage/xxx 这种无效地址
		"${local.domain}":                "",
		"${local.path}":                  "storage",
		"${local.dir_rule}":              DefaultStorageDirRule,
		"${local.file_rule}":             DefaultStorageFileRule,
		"${cos.app_id}":                  "",
		"${cos.secret_id}":               "",
		"${cos.secret_key}":              "",
		"${cos.bucket}":                  "inis-cos",
		"${cos.region}":                  "ap-guangzhou",
		"${cos.domain}":                  "",
		"${cos.path}":                    "inis",
		"${cos.dir_rule}":                DefaultStorageDirRule,
		"${cos.file_rule}":               DefaultStorageFileRule,
		"${attachment.allow_extensions}": "jpg,png,gif,webp,bmp,svg,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar,7z,txt,md",
		"${attachment.max_file_size}":    51200,
		"${attachment.concurrent_limit}": 5,
	})
}

// smsTomlContent - sms.toml 内容
func smsTomlContent() string {
	return utils.Replace(TempSMS, smsTomlOptions())
}

// smsTomlOptions - sms.toml 的默认占位符取值
func smsTomlOptions() map[string]any {
	opts := map[string]any{
		"${drive.sms}":                              "aliyun",
		"${drive.email}":                            "email",
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

	return opts
}
