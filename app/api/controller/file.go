package controller

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"inis/app/facade"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// 预编译正则表达式
var (
	sizeRegex = regexp.MustCompile(`^(\d+)\D+(\d+)$`)
)

// 允许的文件扩展名集合
var allowedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".bmp":  true,
	".webp": true,
	".svg":  true,
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
	".txt":  true,
	".zip":  true,
	".rar":  true,
	".7z":   true,
	".tar":  true,
	".gz":   true,
	".mp3":  true,
	".mp4":  true,
	".wav":  true,
	".avi":  true,
	".mov":  true,
	".flv":  true,
}

// File - 文件管理控制器
// @Summary 文件管理API
// @Description 提供文件上传、随机图片、图片转base64等功能的API接口
// @Tags File
type File struct {
	// 继承
	base
}

// IGET - 获取文件相关信息
// @Summary 获取文件相关信息
// @Description 根据不同方法获取文件相关数据（随机图、转base64等）
// @Tags File
// @Accept json
// @Produce json
// @Param method path string true "方法名" Enums(rand, to-base64)
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/file/{method} [get]
func (this *File) IGET(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"rand":      this.rand,
		"to-base64": this.toBase64,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPOST - 上传文件
// @Summary 上传文件
// @Description 上传文件到服务器（支持图片、文档、压缩包、音视频等多种类型）
// @Tags File
// @Accept multipart/form-data
// @Produce json
// @Param method path string true "方法名" Enums(upload)
// @Param file formData file true "要上传的文件"
// @Success 200 {object} map[string]interface{} "成功响应，包含文件路径"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/file/{method} [post]
func (this *File) IPOST(ctx *gin.Context) {

	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"upload": this.upload,
	}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IPUT - 更新文件信息
// @Summary 更新文件信息
// @Description 更新文件相关数据（当前暂不支持）
// @Tags File
// @Accept json
// @Produce json
// @Param method path string true "方法名"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/file/{method} [put]
func (this *File) IPUT(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// IDEL - 删除文件
// @Summary 删除文件
// @Description 删除文件（当前暂不支持）
// @Tags File
// @Accept json
// @Produce json
// @Param method path string true "方法名"
// @Success 200 {object} map[string]interface{} "成功响应"
// @Failure 405 {object} map[string]interface{} "方法调用错误"
// @Router /api/file/{method} [delete]
func (this *File) IDEL(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{}
	err := this.call(allow, method, ctx)

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

// INDEX - 文件管理首页
// @Summary 文件管理首页
// @Description 文件管理控制器首页（没什么用）
// @Tags File
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功响应"
// @Router /api/file [get]
func (this *File) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 200)
}

// 定义常量
const (
	maxFileSize    = 10 * 1024 * 1024 // 10MB
	badRequestCode = 400
	successCode    = 200
)

// upload - 简单文件上传
func (this *File) upload(ctx *gin.Context) {
	params := this.params(ctx)

	// 上传文件
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取文件失败：%v", err.Error()), badRequestCode)
		return
	}

	// 检查文件大小
	if fileHeader.Size > maxFileSize {
		this.json(ctx, nil, facade.Lang(ctx, "文件大小超过限制（最大10MB）"), badRequestCode)
		return
	}

	// 文件数据
	file, err := fileHeader.Open()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "打开文件失败：%v", err.Error()), badRequestCode)
		return
	}
	defer func(f multipart.File) {
		if err := f.Close(); err != nil {
			// 记录错误但继续执行
			fmt.Printf("关闭文件失败: %v\n", err.Error())
		}
	}(file)

	// 安全处理文件名，防止路径遍历攻击
	fileName := strings.TrimSpace(fileHeader.Filename)
	fileName = strings.ReplaceAll(fileName, "..", "")
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")

	// 文件后缀
	suffix := ""
	if lastIndex := strings.LastIndex(fileName, "."); lastIndex > 0 {
		suffix = strings.ToLower(fileName[lastIndex:])
	}
	params["suffix"] = suffix

	// 检查文件类型是否允许
	if !allowedExtensions[suffix] {
		this.json(ctx, nil, facade.Lang(ctx, "不允许上传该类型的文件！"), badRequestCode)
		return
	}

	// 上传文件
	item := facade.Storage.Upload(facade.Storage.Path()+suffix, file)
	if item.Error != nil {
		this.json(ctx, nil, item.Error.Error(), badRequestCode)
		return
	}

	this.json(ctx, map[string]any{
		"path": item.Domain + item.Path,
	}, facade.Lang(ctx, "上传成功！"), successCode)
}

// rand - 随机图
func (this *File) rand(ctx *gin.Context) {
	params := this.params(ctx)

	root := "public"
	path := root + "/assets/rand/"

	// 分别获取目录下的文件和目录
	info := utils.File().DirInfo(path)

	if info.Error != nil {
		this.json(ctx, nil, info.Error.Error(), badRequestCode)
		return
	}

	domain := cast.ToString(this.get(ctx, "domain"))

	// 获取目录下的文件
	fnDir := func(dirPath any) (slice []string) {
		item := utils.File().Dir(dirPath).List()
		if item.Error != nil {
			return []string{}
		}
		for _, val := range item.Slice {
			// 替换 root 为域名
			val = strings.Replace(cast.ToString(val), root, domain, 1)
			slice = append(slice, cast.ToString(val))
		}
		return slice
	}

	// 读取文件内容
	fnFile := func(filePath any) (slice []string) {
		item := utils.File().Path(filePath).Byte()
		if item.Error != nil {
			return []string{}
		}
		for _, val := range strings.Split(item.Text, "\n") {
			// 过滤末尾的 /r
			slice = append(slice, strings.TrimRight(val, "\r"))
		}
		return slice
	}

	var list []string
	result := cast.ToStringMap(info.Result)
	dirs := cast.ToStringSlice(result["dirs"])
	files := cast.ToStringSlice(result["files"])

	// 读取系统内全部的图片
	fnAll := func() []string {
		var allImages []string

		if !utils.Is.Empty(dirs) {
			for _, value := range dirs {
				allImages = append(allImages, fnDir(path+value)...)
			}
		}

		if !utils.Is.Empty(files) {
			for _, value := range files {
				allImages = append(allImages, fnFile(path+value)...)
			}
		}

		return allImages
	}

	nameStr := cast.ToString(params["name"])

	// 没有指定目录或文件
	if utils.Is.Empty(params["name"]) {
		list = fnAll()
	}

	// 指定目录
	if utils.InArray[string](nameStr, dirs) {
		list = fnDir(path + nameStr)
	}
	// 指定文件
	if utils.InArray[string](nameStr, files) {
		list = fnFile(path + nameStr)
	}

	if utils.Is.Empty(list) {
		this.json(ctx, nil, facade.Lang(ctx, "无图！"), badRequestCode)
		return
	}

	// 远程图片地址
	url := list[utils.Rand.Int(len(list))]

	if cast.ToBool(params["json"]) {
		this.json(ctx, url, facade.Lang(ctx, "随机图！"), successCode)
		return
	}

	if cast.ToBool(params["redirect"]) {
		// 验证URL是否为安全域名（这里简单检查是否为预期的域名格式）
		if !strings.HasPrefix(url, "http://"+domain) && !strings.HasPrefix(url, "https://"+domain) {
			this.json(ctx, nil, facade.Lang(ctx, "不安全的重定向URL"), badRequestCode)
			return
		}
		ctx.Redirect(302, url)
		return
	}

	// 读取远程图片内容
	curlItem := utils.Curl().Url(url).Send()
	if curlItem.Error != nil {
		this.json(ctx, nil, curlItem.Error.Error(), badRequestCode)
		return
	}

	// 从 query 的 size 中获取图片尺寸
	match := sizeRegex.FindStringSubmatch(ctx.Query("size"))

	var err error
	var write int
	imgBytes := curlItem.Byte

	if match != nil {
		width := cast.ToInt(match[1])
		height := cast.ToInt(match[2])

		// 文件后缀 - 转小写
		ext := ""
		if lastIndex := strings.LastIndex(url, "."); lastIndex > 0 {
			ext = strings.ToLower(url[lastIndex+1:])
		}
		// 图片压缩
		imgBytes = compress(ctx, imgBytes, width, height, ext)
	}

	// 输出图片到页面上
	ctx.Writer.Header().Set("Content-Type", "image/jpeg")
	ctx.Writer.Header().Set("Content-Length", cast.ToString(len(imgBytes)))

	write, err = ctx.Writer.Write(imgBytes)
	if err != nil {
		this.json(ctx, nil, err.Error(), badRequestCode)
		return
	}
	if write != len(imgBytes) {
		this.json(ctx, nil, facade.Lang(ctx, "写入失败！"), badRequestCode)
		return
	}
}

// toBase64 - 网络图片转 base64
func (this *File) toBase64(ctx *gin.Context) {
	// 请求参数
	params := this.params(ctx)

	if utils.Is.Empty(params["url"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "url"), badRequestCode)
		return
	}

	url := cast.ToString(params["url"])

	// 读取远程图片内容
	item := utils.Curl().Url(url).Send()
	if item.Error != nil {
		this.json(ctx, nil, item.Error.Error(), badRequestCode)
		return
	}
	if item.StatusCode != 200 {
		this.json(ctx, nil, facade.Lang(ctx, "获取图片失败，状态码：%d", item.StatusCode), badRequestCode)
		return
	}

	// 使用 http.DetectContentType 检测内容类型
	contentType := http.DetectContentType(item.Byte)
	// 若检测到的类型是通用的，尝试从扩展名推断
	if contentType == "application/octet-stream" {
		contentType = "image/jpeg" // 默认类型
		// 尝试从URL中获取扩展名
		if lastIndex := strings.LastIndex(url, "."); lastIndex > 0 {
			ext := strings.ToLower(url[lastIndex+1:])
			switch ext {
			case "png":
				contentType = "image/png"
			case "gif":
				contentType = "image/gif"
			case "webp":
				contentType = "image/webp"
			case "svg":
				contentType = "image/svg+xml"
			case "bmp":
				contentType = "image/bmp"
			}
		}
	}

	// 转 base64
	res := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(item.Byte))
	this.json(ctx, res, facade.Lang(ctx, "成功！"), successCode)
}

// compress - 图片压缩
func compress(ctx *gin.Context, imgData []byte, width, height int, ext string) (result []byte) {
	// 图片处理模式
	mode := ctx.DefaultQuery("mode", "")
	if width == height && mode == "" {
		mode = "fill"
	} else if mode == "" {
		mode = "fit" // 默认使用fit模式
	}

	// 字节转 image
	src, err := imaging.Decode(bytes.NewReader(imgData))
	if err != nil {
		// 如果解码失败，返回原图
		return imgData
	}

	// 处理图片
	var dstImage *image.NRGBA
	switch mode {
	case "fill":
		// 填充
		dstImage = imaging.Fill(src, width, height, imaging.Center, imaging.Lanczos)
	case "resize":
		// 完全自定义大小
		dstImage = imaging.Resize(src, width, height, imaging.Lanczos)
	case "fit":
		// 等比例缩放
		dstImage = imaging.Fit(src, width, height, imaging.Lanczos)
	default:
		// 等比例缩放
		dstImage = imaging.Fit(src, width, height, imaging.Lanczos)
	}

	// 压缩的图片格式
	var format imaging.Format
	switch ext {
	case "jpg", "jpeg":
		format = imaging.JPEG
	case "png":
		format = imaging.PNG
	case "gif":
		format = imaging.GIF
	case "tif", "tiff":
		format = imaging.TIFF
	case "bmp":
		format = imaging.BMP
	default:
		format = imaging.JPEG
	}

	buffer := new(bytes.Buffer)
	// 解决GIF压缩之后不会动的问题
	if err := imaging.Encode(buffer, dstImage, format); err != nil {
		// 如果编码失败，返回原图
		return imgData
	}

	return buffer.Bytes()
}
