package controller

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"inis/app/facade"
	"mime/multipart"
	"regexp"
	"strings"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

type File struct {
	// 继承
	base
}

// IGET - GET请求本体
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

// IPOST - POST请求本体
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

// IPUT - PUT请求本体
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

// IDEL - DELETE请求本体
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

// INDEX - GET请求本体
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
	file, err := ctx.FormFile("file")
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取文件失败：%v", err.Error()), badRequestCode)
		return
	}

	// 检查文件大小
	if file.Size > maxFileSize {
		this.json(ctx, nil, facade.Lang(ctx, "文件大小超过限制（最大10MB）"), badRequestCode)
		return
	}

	// 文件数据
	Byte, err := file.Open()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "打开文件失败：%v", err.Error()), badRequestCode)
		return
	}
	defer func(bytes multipart.File) {
		if err := bytes.Close(); err != nil {
			// 记录错误但继续执行
			fmt.Printf("关闭文件失败: %v\n", err.Error())
		}
	}(Byte)

	// 安全处理文件名，防止路径遍历攻击
	fileName := strings.TrimSpace(file.Filename)
	fileName = strings.ReplaceAll(fileName, "..", "")
	fileName = strings.ReplaceAll(fileName, "/", "")
	fileName = strings.ReplaceAll(fileName, "\\", "")

	// 文件后缀
	suffix := ""
	if lastIndex := strings.LastIndex(fileName, "."); lastIndex > 0 {
		suffix = strings.ToLower(fileName[lastIndex:])
	}
	params["suffix"] = suffix

	// 安全检查：允许的文件类型
	allowedExtensions := []string{
		// 图片文件
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg",
		// 文档文件
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt",
		// 压缩文件
		".zip", ".rar", ".7z", ".tar", ".gz",
		// 音视频文件
		".mp3", ".mp4", ".wav", ".avi", ".mov", ".flv",
	}

	// 检查文件类型是否允许
	isAllowed := false
	for _, ext := range allowedExtensions {
		if suffix == ext {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		this.json(ctx, nil, facade.Lang(ctx, "不允许上传该类型的文件！"), badRequestCode)
		return
	}

	// 上传文件
	item := facade.Storage.Upload(facade.Storage.Path()+suffix, Byte)
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

	// 获取目录下的文件
	fnDir := func(path any) (slice []string) {
		item := utils.File().Dir(path).List()
		if item.Error != nil {
			return []string{}
		}
		for _, val := range item.Slice {
			// 替换 root 为域名
			val = strings.Replace(cast.ToString(val), root, cast.ToString(this.get(ctx, "domain")), 1)
			slice = append(slice, cast.ToString(val))
		}
		return slice
	}

	// 读取文件内容
	fnFile := func(path any) (slice []string) {
		item := utils.File().Path(path).Byte()
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

	// 读取系统内全部的图片
	fnAll := func(result map[string]any) []string {
		var allImages []string

		if !utils.Is.Empty(result["dirs"]) {
			for _, value := range cast.ToStringSlice(result["dirs"]) {
				allImages = append(allImages, fnDir(path+value)...)
			}
		}

		if !utils.Is.Empty(result["files"]) {
			for _, value := range cast.ToStringSlice(result["files"]) {
				allImages = append(allImages, fnFile(path+value)...)
			}
		}

		return allImages
	}

	// 没有指定目录或文件
	if utils.Is.Empty(params["name"]) {
		list = fnAll(result)
	}

	// 指定目录
	if utils.InArray[string](cast.ToString(params["name"]), cast.ToStringSlice(result["dirs"])) {
		list = fnDir(path + cast.ToString(params["name"]))
	}
	// 指定文件
	if utils.InArray[string](cast.ToString(params["name"]), cast.ToStringSlice(result["files"])) {
		list = fnFile(path + cast.ToString(params["name"]))
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
		domain := cast.ToString(this.get(ctx, "domain"))
		if !strings.HasPrefix(url, "http://"+domain) && !strings.HasPrefix(url, "https://"+domain) {
			this.json(ctx, nil, facade.Lang(ctx, "不安全的重定向URL"), badRequestCode)
			return
		}
		ctx.Redirect(302, url)
		return
	}

	// 读取远程图片内容
	item := utils.Curl().Url(url).Send()
	if item.Error != nil {
		this.json(ctx, nil, item.Error.Error(), badRequestCode)
		return
	}

	// 正则表达式，匹配图片尺寸
	reg := regexp.MustCompile(`^(\d+)\D+(\d+)$`)
	// 从 query 的 size 中获取图片尺寸
	match := reg.FindStringSubmatch(ctx.Query("size"))

	var err error
	var write int
	img := item.Byte

	if match != nil {
		width := cast.ToInt(match[1])
		height := cast.ToInt(match[2])

		// 文件后缀 - 转小写
		ext := ""
		if lastIndex := strings.LastIndex(url, "."); lastIndex > 0 {
			ext = strings.ToLower(url[lastIndex+1:])
		}
		// 图片压缩
		img = compress(ctx, img, width, height, ext)
	}

	// 输出图片到页面上
	ctx.Writer.Header().Set("Content-Type", "image/jpeg")
	ctx.Writer.Header().Set("Content-Length", cast.ToString(len(img)))

	write, err = ctx.Writer.Write(img)
	if err != nil {
		this.json(ctx, nil, err.Error(), badRequestCode)
		return
	}
	if write != len(img) {
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

	// 由于无法直接获取Content-Type，我们根据文件扩展名或内容推断类型
	contentType := "image/jpeg" // 默认类型

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

	// 转 base64
	res := fmt.Sprintf("data:%s;base64,%s", contentType, base64.StdEncoding.EncodeToString(item.Byte))
	this.json(ctx, res, facade.Lang(ctx, "成功！"), successCode)
}

// compress - 图片压缩
func compress(ctx *gin.Context, byte []byte, width, height int, ext string) (result []byte) {
	// 图片处理模式
	mode := ctx.DefaultQuery("mode", "")
	if width == height && mode == "" {
		mode = "fill"
	} else if mode == "" {
		mode = "fit" // 默认使用fit模式
	}

	// byte 转 image
	src, err := imaging.Decode(bytes.NewReader(byte))
	if err != nil {
		// 如果解码失败，返回原图
		return byte
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
		return byte
	}

	return buffer.Bytes()
}