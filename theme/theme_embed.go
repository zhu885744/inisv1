//go:build embed

// 打包前端产物进二进制：需要 theme/dist 目录存在。
//
// 构建方式（build.bat / build.sh 已封装）：
//
//	npm run build --prefix Mellow        # 产出 Mellow/dist
//	# 把 Mellow/dist/* 拷到 theme/dist/
//	go build -tags embed -o inis main.go
//
// 目录不存在时直接编译报错（pattern all:dist: no matching files found）——
// 这是刻意为之：避免打了 embed 标签却忘了准备产物，最后发出去一个没有主题的二进制。
package theme

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var embedded embed.FS

func init() {
	// 带上 theme/dist 前缀的子文件系统：这样 theme.Lookup("index.html") 之类
	// 用的都是「相对产物的路径」，不带内嵌目录前缀
	if sub, err := fs.Sub(embedded, "dist"); err == nil {
		dist = sub
	}
}
