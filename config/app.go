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

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// HTTP响应常量
const (
	SuccessCode      = 200
	ErrorCode        = 400
	ServerErrorCode  = 500
	StatusNotFound   = 404
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

// Gin - gin引擎
var Gin *gin.Engine

// AppToml - App配置文件
var AppToml *utils.ViperResponse

// Server - 服务
var Server *http.Server

func init() {
	initAppToml()
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

	select {}
}

// notRoute 路由不存在
func notRoute(Gin *gin.Engine) {
	Gin.NoRoute(func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				ctx.JSON(SuccessCode, gin.H{"code": ServerErrorCode, "msg": InternalErrorMsg, "data": nil})
			}
		}()

		ctx.Status(SuccessCode)

		path := ctx.Request.URL.Path
		prefix := path[:strings.LastIndex(path, "/")]
		fileName := path[strings.LastIndex(path, "/"):]
		ext := strings.ToLower(fileName[strings.LastIndex(fileName, ".")+1:])

		isExist := checkFileExist(ctx, "public"+path)
		writeErrorGif := writeGifError(ctx)
		writeImage := writeImageFile(ctx)

		switch {
		case utils.In.Array(fileName, PageFiles):
			handlePageFile(ctx, prefix, writeErrorGif)
		case utils.In.Array(ext, ImageFiles):
			handleImageFile(ctx, path, ext, writeImage, writeErrorGif, isExist)
		case strings.Contains(fileName, "."):
			handleStaticFile(ctx, path, ext, isExist)
		default:
			ctx.JSON(SuccessCode, gin.H{"code": ErrorCode, "msg": RouteNotDefined, "data": nil})
		}
	})
}

// checkFileExist 检查文件是否存在
func checkFileExist(ctx *gin.Context, path string) func(string) bool {
	return func(checkPath string) bool {
		if !strings.HasPrefix(checkPath, "public") {
			checkPath = "public/" + checkPath
		}
		exist := utils.File().Exist(checkPath)
		if !exist {
			ctx.JSON(SuccessCode, gin.H{"code": ErrorCode, "msg": ResourceNotFound, "data": nil})
		}
		return exist
	}
}

// writeGifError 写入错误GIF
func writeGifError(ctx *gin.Context) func(string) {
	return func(gifName string) {
		_, err := ctx.Writer.Write(utils.File().Byte("public/assets/images/gif/" + gifName).Byte)
		if err != nil {
			ctx.JSON(SuccessCode, gin.H{"code": ErrorCode, "msg": ResourceNotFound, "data": nil})
		}
	}
}

// writeImageFile 写入图片文件
func writeImageFile(ctx *gin.Context) func(string, string) {
	return func(path string, ext string) {
		ctx.Header("Content-Type", utils.Mime.Type(ext)+"; charset=utf-8")
		_, err := ctx.Writer.Write(utils.File().Byte("public" + path).Byte)
		if err != nil {
			ctx.JSON(SuccessCode, gin.H{"code": ErrorCode, "msg": ResourceNotFound, "data": nil})
		}
	}
}

// handlePageFile 处理页面文件
func handlePageFile(ctx *gin.Context, prefix string, writeErrorGif func(string)) {
	if check := utils.File().Exist("public/" + prefix + "/index.html"); check {
		ctx.Header("Content-Type", "text/html; charset=utf-8")
		_, err := ctx.Writer.Write(utils.File().Byte("public" + prefix + "/index.html").Byte)
		if err != nil {
			writeErrorGif("error.gif")
		}
	}
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

	_, err = ctx.Writer.Write(buffer.Bytes())
	if err != nil {
		writeErrorGif("error.gif")
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
		return
	}

	ctx.Header("Content-Type", utils.Mime.Type(ext)+"; charset=utf-8")
	_, err := ctx.Writer.Write(utils.File().Byte("public" + path).Byte)
	if err != nil {
		ctx.JSON(SuccessCode, gin.H{"code": ErrorCode, "msg": FileReadError, "data": err.Error()})
	}
}

// console 控制台
func console() {
	port := AppToml.Get("app.port", 8080)
	char := `
    ──────────────────────────────
      版本号: %-10s  端口: %-6d    
      状态: 服务已启动               
    ──────────────────────────────
    `
	fmt.Println(fmt.Sprintf(char, facade.Version, port))
}
