// Package theme - 内嵌主题（Mellow 的前端构建产物）
//
// 背景：主题产物默认是「拷进 public 目录、由 config/app.go 的 NoRoute 读磁盘」的方式部署的，
// 分发时要额外带一堆静态文件。要「一个二进制搞定部署」时，用 go:embed 把产物塞进可执行文件，
// 运行时只剩「二进制 + config 目录」。
//
// 两种构建方式（见 build.bat / build.sh）：
//
//	go build ./...          不带前端：dist 为 nil，Lookup 一律返回 nil，行为与以前完全一致；
//	go build -tags embed    带前端：构建脚本先把 Mellow 的前端产物拷到 theme/dist，
//	                        再由 theme_embed.go 的 //go:embed all:dist 打包进二进制。
//
// 命中优先级由调用方（config/app.go）决定：**内嵌优先、磁盘兜底** ——
// 二进制里没有的文件（public/assets 的表情包与占位图、上传附件等）照旧读磁盘，
// 所以打包后既省事，也不影响原有的动态资源。
//
// 安装向导（/install）也在主题里，因此打包后全新安装同样只需要一个二进制。
package theme

import (
	"hash/fnv"
	"inis/app/facade"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/unti-io/go-utils/utils"
)

// dist - 内嵌的主题产物文件系统
//
// 默认构建为 nil（不带任何前端文件），带 embed 构建标签时由 theme_embed.go 的 init 赋值。
var dist fs.FS

// File - 内嵌主题里的一个文件（连同响应需要的元信息）
type File struct {
	// Name 相对 theme/dist 的文件名，如 index.html、static/xxx.js
	Name string
	// Body 文件内容
	Body []byte
	// Type Content-Type
	Type string
	// ETag 内容指纹（用于 If-None-Match 校验）
	ETag string
	// Cache Cache-Control
	Cache string
}

// Ready - 当前二进制里是否打包了主题产物
func Ready() bool {
	return dist != nil
}

// HasIndex - 内嵌主题里是否有首页（对应磁盘上的 public/index.html）
//
// 首页那条路由（app/index/route/app.go）用它判断「模板是否存在」：
// 打包过的二进制里即使没有 public 目录，也该认为模板是就绪的。
func HasIndex() bool {
	return Lookup("/index.html") != nil
}

// Serve - 把内嵌主题文件写进响应，命中返回 true（调用方直接 return 即可）
//
// 未打包主题（默认构建）或路径没命中时返回 false，调用方继续走原有的磁盘逻辑。
// Content-Type / Cache-Control / ETag 与 304 都在这里统一处理。
func Serve(ctx *gin.Context, urlPath string) bool {
	file := Lookup(urlPath)
	if file == nil {
		return false
	}

	ctx.Header("Content-Type", file.Type)
	ctx.Header("Cache-Control", file.Cache)
	ctx.Header("ETag", file.ETag)

	// 只有在「响应还没开始写」时才能回 304；调用方已经写过状态码的话就只能整包返回
	if !ctx.Writer.Written() {
		if match := ctx.GetHeader("If-None-Match"); match != "" && strings.Contains(match, file.ETag) {
			ctx.Status(http.StatusNotModified)
			return true
		}
	}

	ctx.Status(http.StatusOK)

	// 已经开始写响应体了，出错只记录日志
	if _, err := ctx.Writer.Write(file.Body); err != nil {
		facade.Log.Error(map[string]any{"error": err, "path": file.Name}, "写入内嵌主题文件失败")
	}

	return true
}

// Lookup - 按 URL 路径查内嵌主题文件；没命中返回 nil（调用方继续走原有的磁盘逻辑）
//
// 支持三种写法：
//   - ""、"/"          → index.html（首页）
//   - "/static/a.js"   → static/a.js（构建产物的静态资源）
//   - "/user/profile"  → user/profile/index.html（history 模式主题的子目录；没有则返回 nil）
func Lookup(urlPath string) *File {
	if dist == nil {
		return nil
	}

	name := strings.TrimSpace(urlPath)
	// 去掉查询串 / 锚点（NoRoute 拿到的通常是纯路径，这里兼容直接调用）
	if index := strings.IndexAny(name, "?#"); index >= 0 {
		name = name[:index]
	}

	// path.Clean 会消掉 ../，再显式挡一层：避免读到 dist 之外
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || name == "." {
		name = "index.html"
	}
	if strings.Contains(name, "..") {
		return nil
	}

	if file := load(name); file != nil {
		return file
	}

	// 目录形式：/user/profile → user/profile/index.html
	if file := load(strings.TrimSuffix(name, "/") + "/index.html"); file != nil {
		return file
	}

	return nil
}

// load - 从内嵌文件系统读一个文件并补上响应元信息
func load(name string) *File {
	body, err := fs.ReadFile(dist, name)
	if err != nil {
		return nil
	}

	return &File{
		Name:  name,
		Body:  body,
		Type:  contentType(name),
		ETag:  etag(body),
		Cache: cacheControl(name),
	}
}

// mimeFallback - go-utils 的 MimeMap 没覆盖到的类型（前端产物可能出现的字体 / 图片 / 其它）
//
// 这里刻意不用标准库的 mime.TypeByExtension：它在 Windows 上会先查注册表，
// 可能把 .js 之类的类型映射成奇怪的值，导致 <script type="module"> 被 MIME 校验拦下。
var mimeFallback = map[string]string{
	"webp":        "image/webp",
	"avif":        "image/avif",
	"woff":        "font/woff",
	"woff2":       "font/woff2",
	"ttf":         "font/ttf",
	"otf":         "font/otf",
	"eot":         "application/vnd.ms-fontobject",
	"map":         "application/json",
	"wasm":        "application/wasm",
	"webmanifest": "application/manifest+json",
	"md":          "text/markdown",
	"mp4":         "video/mp4",
	"webm":        "video/webm",
	"mjs":         "application/javascript",
}

// contentType - 按扩展名给出 Content-Type
func contentType(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))

	item := strings.TrimSpace(utils.Mime.Type(ext))
	if item == "" {
		item = mimeFallback[ext]
	}
	if item == "" {
		return "application/octet-stream"
	}

	// 文本类补上编码，避免浏览器按 latin-1 猜中文
	if strings.HasPrefix(item, "text/") || item == "application/javascript" ||
		item == "application/json" || item == "application/manifest+json" {
		return item + "; charset=utf-8"
	}

	return item
}

// cacheControl - 缓存策略
func cacheControl(name string) string {
	switch {
	case strings.HasPrefix(name, "static/"):
		// Vite 产物文件名带内容 hash：改了内容文件名就变，可以放心长缓存
		return "public, max-age=31536000, immutable"
	case strings.HasSuffix(name, ".html"), strings.HasSuffix(name, "runtime-config.js"):
		// 入口 HTML 与运行时配置：必须回源校验，否则发新版后老用户还停在旧页面
		return "no-cache"
	default:
		return "public, max-age=3600"
	}
}

// etag - 内容指纹（FNV-1a，够用且不用额外依赖）
func etag(body []byte) string {
	hash := fnv.New32a()
	_, _ = hash.Write(body)
	return `"` + strconv.FormatUint(uint64(hash.Sum32()), 16) + `"`
}
