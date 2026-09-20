package config

import (
	"bytes"
	"fmt"
	"image"
	"inis/app/facade"
	"inis/app/middleware"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// HTTP响应常量
const (
	// HTTPStatusOK - 统一使用 HTTP 200 作为响应状态码
	// 业务层面的成功/失败由响应体的 code 字段表达，故不以 Success 命名，避免被误读为「业务成功」
	HTTPStatusOK = 200
	// CodeError - 业务错误码（写入响应体 code 字段）
	CodeError = 400
	// CodeServerError - 业务服务器错误码（写入响应体 code 字段）
	CodeServerError = 500

	InternalErrorMsg = "服务器内部错误！"
	ResourceNotFound = "资源不存在！"
	FileReadError    = "文件读取失败！"
	RouteNotDefined  = "路由未定义！"
)

// 文件类型常量
var (
	// 页面文件后缀
	PageFiles = []any{"/", "/index.htm", "/index.html", "/index.php", "/index.jsp"}
	// 图片文件后缀
	ImageFiles = []any{"jpg", "jpeg", "png", "gif", "tif", "tiff", "bmp"}
)

// 图片动态处理结果缓存（进程内），避免每次请求重复读盘+解码+编码
var imageCache = &sync.Map{}
var imageCacheLen atomic.Int64

// imageCacheMaxLen 图片缓存最大条目数，超过则整体清空，防止无限制增长
const imageCacheMaxLen = 1024

// Gin - gin引擎
var Gin *gin.Engine

// AppToml - App配置文件
var AppToml *utils.ViperResponse

// Server - 服务
var Server *http.Server

func init() {
	initAppToml()
	loadThemeRouteIgnore()
	InitApp()
}

// initAppToml - 初始化APP配置文件
func initAppToml() {
	item := utils.Viper(utils.ViperModel{
		Path:    "config",
		Mode:    "toml",
		Name:    "app",
		Content: utils.Replace(facade.TempApp, nil),
	}).Read()

	if item.Error != nil {
		fmt.Println("APP配置文件初始化发生错误", item.Error)
		return
	}

	AppToml = &item
}

// InitApp 初始化App
func InitApp() {
	debug := cast.ToBool(AppToml.Get("app.debug", false))

	if !debug {
		gin.SetMode(gin.ReleaseMode)
		gin.DefaultWriter = io.Discard
	}

	Gin = gin.Default()
	notRoute(Gin)
	console()

	Gin.Use(middleware.GinLogger(), middleware.GinRecovery(true))
}

// Use 注册配置
func Use(args ...func(*gin.Engine)) {
	for _, fn := range args {
		fn(Gin)
	}
}

// Run 启动服务
func Run(callback ...func()) {
	for _, fn := range callback {
		fn()
	}

	port := ":" + cast.ToString(AppToml.Get("app.port", 8080))

	Server = &http.Server{
		Addr:    port,
		Handler: Gin,
	}

	go func() {
		if err := Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Println("服务启动失败", err)
		}
	}()
}

// notRoute 路由不存在
func notRoute(Gin *gin.Engine) {
	Gin.NoRoute(func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx.JSON(HTTPStatusOK, gin.H{"code": CodeServerError, "msg": InternalErrorMsg, "data": nil})
			}
		}()

		ctx.Status(HTTPStatusOK)

		path := ctx.Request.URL.Path

		// 路径安全校验：拒绝目录穿越（../），防止读取 public 目录之外的任意文件
		if strings.Contains(path, "..") {
			ctx.JSON(HTTPStatusOK, gin.H{"code": CodeError, "msg": RouteNotDefined, "data": nil})
			return
		}

		// 使用 LastIndex 前先判断，避免 path 中无 "/" 时切片越界 panic
		lastSlash := strings.LastIndex(path, "/")
		var prefix, fileName string
		if lastSlash >= 0 {
			prefix = path[:lastSlash]
			fileName = path[lastSlash:]
		} else {
			prefix = ""
			fileName = path
		}

		ext := ""
		lastDot := strings.LastIndex(fileName, ".")
		if lastDot >= 0 && lastDot+1 < len(fileName) {
			ext = strings.ToLower(fileName[lastDot+1:])
		}

		isExist := fileExist
		writeErrorGif := writeGifError(ctx)
		writeImage := writeImageFile(ctx)

		switch {
		case utils.In.Array(fileName, PageFiles):
			handlePageFile(ctx, prefix)
		case utils.In.Array(ext, ImageFiles):
			handleImageFile(ctx, path, ext, writeImage, writeErrorGif, isExist)
		case strings.Contains(fileName, "."):
			handleStaticFile(ctx, path, ext, isExist)
		default:
			// 主题前端路由回退：history 模式的主题（如 /goods、/user/profile）在 public 下
			// 没有对应文件，统一回退到主题首页，由前端路由接管
			if handleThemeRoute(ctx, path) {
				return
			}
			ctx.JSON(HTTPStatusOK, gin.H{"code": CodeError, "msg": RouteNotDefined, "data": nil})
		}
	})
}

// fileExist 检查 public 下的文件是否存在
// 纯函数：只做判断，不产生任何响应写入，避免调用方在已写入响应体后再追加 JSON 造成响应错乱
func fileExist(path string) bool {
	if !strings.HasPrefix(path, "public") {
		path = "public/" + path
	}
	return utils.File().Exist(path)
}

// writeGifError 写入错误占位图（如 404.gif、error.gif）
func writeGifError(ctx *gin.Context) func(string) {
	return func(gifName string) {
		path := "public/assets/images/gif/" + gifName

		// 占位图本身不存在：此时尚未写入图片字节，可正常返回统一 JSON 提示
		if !utils.File().Exist(path) {
			facade.Log.Error(map[string]any{"path": path}, "错误占位图不存在")
			ctx.JSON(HTTPStatusOK, gin.H{"code": CodeError, "msg": ResourceNotFound, "data": nil})
			return
		}

		// 覆盖调用方可能已设置的 Content-Type，确保与 GIF 内容一致
		ctx.Header("Content-Type", utils.Mime.Type("gif")+"; charset=utf-8")

		if _, err := ctx.Writer.Write(utils.File().Byte(path).Byte); err != nil {
			// 响应体已开始写入，HTTP 状态码不可再修改，仅记录日志
			facade.Log.Error(map[string]any{"error": err, "path": path}, "写入错误占位图失败")
		}
	}
}

// writeImageFile 写入图片文件
func writeImageFile(ctx *gin.Context) func(string, string) {
	return func(path string, ext string) {
		ctx.Header("Content-Type", utils.Mime.Type(ext)+"; charset=utf-8")
		// 这里已经开始写响应体，HTTP 状态码无法再更改，出错只记录日志
		if _, err := ctx.Writer.Write(utils.File().Byte("public" + path).Byte); err != nil {
			facade.Log.Error(map[string]any{"error": err, "path": path}, "写入图片文件失败")
		}
	}
}

// handlePageFile 处理页面文件
func handlePageFile(ctx *gin.Context, prefix string) {
	if check := utils.File().Exist("public/" + prefix + "/index.html"); check {
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		// 这里已经开始写响应体，HTTP 状态码无法再更改，出错只记录日志
		if _, err := ctx.Writer.Write(utils.File().Byte("public" + prefix + "/index.html").Byte); err != nil {
			facade.Log.Error(map[string]any{"error": err, "path": prefix + "/index.html"}, "写入页面文件失败")
		}
		return
	}
	// 该目录下没有独立页面时，回退到主题首页（兼容 history 模式主题的 /about/ 形式）
	handleThemeRoute(ctx, prefix+"/")
}

// themeRouteIgnoreDefault - 主题回退默认忽略的路径前缀（接口与静态资源保持原有响应，便于排查问题）
var themeRouteIgnoreDefault = []string{"/api", "/dev", "/socket", "/assets"}

// themeRouteIgnore - 主题回退忽略的路径前缀
// 可通过 config/app.toml 的 app.theme_ignore_prefix 配置（多个用英文逗号分隔），无需改代码
var themeRouteIgnore = themeRouteIgnoreDefault

// loadThemeRouteIgnore - 从配置加载主题回退忽略前缀，未配置时回退默认值
func loadThemeRouteIgnore() {
	themeRouteIgnore = themeRouteIgnoreDefault

	if AppToml == nil {
		return
	}

	raw := strings.TrimSpace(cast.ToString(AppToml.Get("app.theme_ignore_prefix", "")))
	if raw == "" {
		return
	}

	items := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// 统一补齐前导斜杠，避免配置写成 "api" 时匹配不到
		if !strings.HasPrefix(item, "/") {
			item = "/" + item
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		themeRouteIgnore = items
	}
}

// handleThemeRoute 主题前端路由回退（history 模式 SPA）
// 语义对齐 nginx 的 try_files $uri $uri/ /index.html：
// 1. public/<path>/index.html（部署在子目录的应用，如 public/admin/index.html）
// 2. public/index.html（主题首页，交给前端路由接管，如 /goods、/user/profile）
// 命中忽略前缀或主题未部署时返回 false，交回调用方处理
func handleThemeRoute(ctx *gin.Context, path string) bool {
	for _, prefix := range themeRouteIgnore {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return false
		}
	}

	// 1. 目录形式：public/<path>/index.html
	if dir := strings.Trim(path, "/"); dir != "" {
		if target := "public/" + dir + "/index.html"; utils.File().Exist(target) {
			return writeThemePage(ctx, target, path)
		}
	}

	// 2. 兜底：主题首页，未部署时不回退，保持原有提示
	if !utils.File().Exist("public/index.html") {
		return false
	}
	return writeThemePage(ctx, "public/index.html", path)
}

// writeThemePage 输出主题页面
func writeThemePage(ctx *gin.Context, file, path string) bool {
	ctx.Header("Content-Type", "text/html; charset=utf-8")
	if _, err := ctx.Writer.Write(utils.File().Byte(file).Byte); err != nil {
		facade.Log.Error(map[string]any{"error": err, "path": path}, "写入主题页面失败")
	}
	return true
}

// handleImageFile 处理图片文件
func handleImageFile(ctx *gin.Context, path, ext string, writeImage func(string, string), writeErrorGif func(string), isExist func(string) bool) {
	ctx.Header("Content-Type", utils.Mime.Type(ext)+"; charset=utf-8")

	if !isExist("public" + path) {
		writeErrorGif("404.gif")
		return
	}

	reg := regexp.MustCompile(`^(\d+)\D+(\d+)$`)
	match := reg.FindStringSubmatch(ctx.Query("size"))

	if match == nil {
		writeImage(path, ext)
		return
	}

	width := cast.ToInt(match[1])
	height := cast.ToInt(match[2])
	mode := ctx.DefaultQuery("mode", utils.Ternary(width == height, "fill", ""))

	cacheKey := fmt.Sprintf("%s?size=%dx%d&mode=%s", path, width, height, mode)

	// 命中缓存则直接返回，避免重复读盘+解码+编码
	if cached, ok := imageCache.Load(cacheKey); ok {
		_, _ = ctx.Writer.Write(cached.([]byte))
		return
	}

	src, err := imaging.Open("public" + path)
	if err != nil {
		writeErrorGif("error.gif")
		return
	}

	dstImage := processImage(src, width, height, mode)
	format := getImageFormat(ext)

	buffer := new(bytes.Buffer)
	err = imaging.Encode(buffer, dstImage, format)
	if err != nil {
		writeErrorGif("error.gif")
		return
	}

	data := buffer.Bytes()

	// 写入缓存（带简单条目上限，超过则整体清空）
	if imageCacheLen.Load() < imageCacheMaxLen {
		imageCache.Store(cacheKey, data)
		imageCacheLen.Add(1)
	} else {
		imageCache.Range(func(key, _ any) bool {
			imageCache.Delete(key)
			return true
		})
		imageCacheLen.Store(1)
		imageCache.Store(cacheKey, data)
	}

	if _, err = ctx.Writer.Write(data); err != nil {
		// 响应体已开始写入，HTTP 状态码无法再更改，仅记录日志
		facade.Log.Error(map[string]any{"error": err, "path": path}, "写入处理后的图片失败")
	}
}

// processImage 处理图片
func processImage(src image.Image, width, height int, mode string) *image.NRGBA {
	switch mode {
	case "fill":
		return imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
	case "resize":
		return imaging.Resize(src, width, height, imaging.Lanczos)
	case "fit":
		return imaging.Fit(src, width, height, imaging.Lanczos)
	default:
		return imaging.Fit(src, width, height, imaging.Lanczos)
	}
}

// getImageFormat 获取图片格式
func getImageFormat(ext string) imaging.Format {
	formats := map[string]imaging.Format{
		"jpg":  imaging.JPEG,
		"jpeg": imaging.JPEG,
		"png":  imaging.PNG,
		"gif":  imaging.GIF,
		"tif":  imaging.TIFF,
		"tiff": imaging.TIFF,
		"bmp":  imaging.BMP,
	}

	if format, ok := formats[ext]; ok {
		return format
	}
	return imaging.JPEG
}

// handleStaticFile 处理静态文件
func handleStaticFile(ctx *gin.Context, path, ext string, isExist func(string) bool) {
	if !isExist("public" + path) {
		ctx.JSON(HTTPStatusOK, gin.H{"code": CodeError, "msg": ResourceNotFound, "data": nil})
		return
	}

	ctx.Header("Content-Type", utils.Mime.Type(ext)+"; charset=utf-8")
	// 这里已经开始写响应体，HTTP 状态码无法再更改，出错只记录日志
	if _, err := ctx.Writer.Write(utils.File().Byte("public" + path).Byte); err != nil {
		facade.Log.Error(map[string]any{"error": err, "path": path}, "写入静态文件失败")
	}
}

// console 控制台
func console() {
	// AppToml.Get 返回 any，直接交给 %d 格式化会 panic 或输出 0，这里显式转成 int
	port := 8080
	if AppToml != nil {
		port = cast.ToInt(AppToml.Get("app.port", 8080))
	}

	char := `
    ──────────────────────────────
      版本号: %-10s  端口: %-6d    
      状态: 服务已启动               
    ──────────────────────────────
    `
	fmt.Println(fmt.Sprintf(char, facade.Version, port))
}
