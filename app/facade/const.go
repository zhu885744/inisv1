package facade

import "os"

const (
	// Version - 版本号
	Version = "1.0.0"
	// Uri - 服务地址
	Uri = "https://cs.zhuxu.asia"
	// DefaultToken - 设备签名盐值默认值（可通过环境变量 INIS_DEVICE_TOKEN 覆盖）
	DefaultToken = "(unti.io)"
)

// GetToken - 获取设备签名盐值，优先使用环境变量 INIS_DEVICE_TOKEN
func GetToken() string {
	if token := os.Getenv("INIS_DEVICE_TOKEN"); token != "" {
		return token
	}
	return DefaultToken
}