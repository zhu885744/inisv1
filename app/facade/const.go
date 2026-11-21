package facade

import "os"

const (
	// Uri - 服务地址
	Uri = "https://zhuxu.asia"
	// DefaultToken - 设备签名盐值默认值（可通过环境变量 INIS_DEVICE_TOKEN 覆盖）
	DefaultToken = "(unti.io)"
)

// Version - 版本号
//
// 必须是 var 而不是 const：发布时由 GoReleaser 用
// `-ldflags "-X inis/app/facade.Version=<tag>"` 注入真实版本号（见 .goreleaser.yaml），
// const 不能被 -X 覆盖。这里的默认值只在 go run / go build 直接编译时生效。
var Version = "1.0.8"

// GetToken - 获取设备签名盐值，优先使用环境变量 INIS_DEVICE_TOKEN
func GetToken() string {
	if token := os.Getenv("INIS_DEVICE_TOKEN"); token != "" {
		return token
	}
	return DefaultToken
}