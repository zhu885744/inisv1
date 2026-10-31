package controller

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

var uploadConcurrentCounter int
var uploadCounterMutex sync.Mutex

// 文件名 / SVG 清洗用的正则：包级只编译一次
//
// 原来是在 sanitizeFileName / sanitizeSVG 里 MustCompile：每上传一次文件
// 就要重新编译 1 + 9 个正则，编译产物用完即弃，纯属不必要的 CPU 与内存抖动
// （SVG 那几个正则的表达式编译结果都不小）。
var (
	fileNameIllegalPattern = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	svgScriptPattern       = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	svgScriptTagPattern    = regexp.MustCompile(`(?i)<script[^>]*\/?>`)
	svgCDataPattern        = regexp.MustCompile(`<!\[CDATA\[.*?\]\]>`)
	svgOnAttrQuotedPattern = regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`)
	svgOnAttrPattern       = regexp.MustCompile(`(?i)on\w+\s*=\s*[^\s>]+`)
	svgForeignPattern      = regexp.MustCompile(`(?i)<foreignObject[^>]*>.*?</foreignObject>`)
	svgForeignTagPattern   = regexp.MustCompile(`(?i)<foreignObject[^>]*\/?>`)
	svgJavascriptPattern   = regexp.MustCompile(`(?i)javascript:`)
	svgVbscriptPattern     = regexp.MustCompile(`(?i)vbscript:`)
)

type Attachment struct{ base }

var attachmentAllowFieldsSlice = []any{"original_name", "target_type", "target_id"}
var attachmentAllowQuerySlice = []any{"id", "uuid"}

func (this *Attachment) buildQuery(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	return query.
		IWhere(params["where"]).
		IOr(params["or"]).
		ILike(params["like"]).
		INot(params["not"]).
		INull(params["null"]).
		INotNull(params["notNull"])
}

func (this *Attachment) withTrashOptions(query *facade.ModelStruct, params map[string]any) *facade.ModelStruct {
	if cast.ToBool(params["onlyTrashed"]) {
		query = query.OnlyTrashed()
	}
	if cast.ToBool(params["withTrashed"]) {
		query = query.WithTrashed()
	}
	return query
}

func (this *Attachment) getFromCache(ctx *gin.Context, cacheName string) (any, bool) {
	if !this.cache.enable(ctx) || !facade.Cache.Has(cacheName) {
		return nil, false
	}
	return facade.Cache.Get(cacheName), true
}

func (this *Attachment) setCache(ctx *gin.Context, cacheName string, data any) {
	if this.cache.enable(ctx) {
		go facade.Cache.Set(cacheName, data, 10*time.Minute)
	}
}

func (this *Attachment) sanitizeFileName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	fileName = path.Base(fileName)
	fileName = filepath.Clean(fileName)
	fileName = fileNameIllegalPattern.ReplaceAllString(fileName, "_")
	return fileName
}

func (this *Attachment) safeDeleteFile(path string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"path": path, "panic": r}, "删除文件时发生panic")
			}
		}()
		facade.Storage.Delete(path)
	}()
}

// isFtypContainer - 是否 ISO Base Media 容器（mp4 / mov / 3gp / heic / avif 共用）
//
// 第 4~7 字节固定是 "ftyp"，第 8 字节起是品牌（isom / mp42 / qt / heic / avif …）
func isFtypContainer(headerBytes []byte) bool {
	return len(headerBytes) >= 12 &&
		headerBytes[4] == 'f' && headerBytes[5] == 't' && headerBytes[6] == 'y' && headerBytes[7] == 'p'
}

// ftypBrand - 取 ISO Base Media 的主品牌（不是 ftyp 容器时为空）
func ftypBrand(headerBytes []byte) string {
	if !isFtypContainer(headerBytes) {
		return ""
	}
	return string(headerBytes[8:12])
}

func (this *Attachment) verifyFileContent(headerBytes []byte, fileExt string) bool {
	if len(headerBytes) == 0 {
		return true
	}

	switch fileExt {
	// jfif / jpe 都是 JPEG 的另一种后缀，内容头一致
	case "jpg", "jpeg", "jfif", "jpe":
		return len(headerBytes) >= 2 && headerBytes[0] == 0xFF && headerBytes[1] == 0xD8
	// apng 只是「会动的 png」，文件头完全相同
	case "png", "apng":
		return len(headerBytes) >= 8 &&
			headerBytes[0] == 0x89 && headerBytes[1] == 0x50 && headerBytes[2] == 0x4E && headerBytes[3] == 0x47 &&
			headerBytes[4] == 0x0D && headerBytes[5] == 0x0A && headerBytes[6] == 0x1A && headerBytes[7] == 0x0A
	case "gif":
		return len(headerBytes) >= 6 &&
			(headerBytes[0] == 'G' && headerBytes[1] == 'I' && headerBytes[2] == 'F' &&
				headerBytes[3] == '8' && (headerBytes[4] == '7' || headerBytes[4] == '9') && headerBytes[5] == 'a')
	case "webp":
		return len(headerBytes) >= 12 &&
			headerBytes[0] == 'R' && headerBytes[1] == 'I' && headerBytes[2] == 'F' && headerBytes[3] == 'F' &&
			headerBytes[8] == 'W' && headerBytes[9] == 'E' && headerBytes[10] == 'B' && headerBytes[11] == 'P'
	case "bmp":
		return len(headerBytes) >= 2 && headerBytes[0] == 'B' && headerBytes[1] == 'M'
	case "ico":
		// ICO / CUR：保留字 0 + 类型 1（图标）/ 2（光标）
		return len(headerBytes) >= 4 &&
			headerBytes[0] == 0x00 && headerBytes[1] == 0x00 &&
			(headerBytes[2] == 0x01 || headerBytes[2] == 0x02) && headerBytes[3] == 0x00
	case "tif", "tiff":
		// 小端 II*\0 或大端 MM\0*
		return len(headerBytes) >= 4 &&
			((headerBytes[0] == 'I' && headerBytes[1] == 'I' && headerBytes[2] == 0x2A && headerBytes[3] == 0x00) ||
				(headerBytes[0] == 'M' && headerBytes[1] == 'M' && headerBytes[2] == 0x00 && headerBytes[3] == 0x2A))
	case "avif":
		brand := ftypBrand(headerBytes)
		return brand == "avif" || brand == "avis"
	case "heic", "heif":
		switch ftypBrand(headerBytes) {
		case "heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
			return true
		}
		return false
	case "pdf":
		return len(headerBytes) >= 4 &&
			headerBytes[0] == '%' && headerBytes[1] == 'P' && headerBytes[2] == 'D' && headerBytes[3] == 'F'
	case "doc":
		return len(headerBytes) >= 8 &&
			headerBytes[0] == 0xD0 && headerBytes[1] == 0xCF && headerBytes[2] == 0x11 && headerBytes[3] == 0xE0 &&
			headerBytes[4] == 0xA1 && headerBytes[5] == 0xB1 && headerBytes[6] == 0x1A && headerBytes[7] == 0xE1
	case "docx", "xlsx", "pptx":
		if len(headerBytes) < 4 || !(headerBytes[0] == 0x50 && headerBytes[1] == 0x4B && headerBytes[2] == 0x03 && headerBytes[3] == 0x04) {
			return false
		}
		return true
	case "xls":
		return len(headerBytes) >= 8 &&
			headerBytes[0] == 0xD0 && headerBytes[1] == 0xCF && headerBytes[2] == 0x11 && headerBytes[3] == 0xE0 &&
			headerBytes[4] == 0xA1 && headerBytes[5] == 0xB1 && headerBytes[6] == 0x1A && headerBytes[7] == 0xE1
	case "ppt":
		return len(headerBytes) >= 8 &&
			headerBytes[0] == 0xD0 && headerBytes[1] == 0xCF && headerBytes[2] == 0x11 && headerBytes[3] == 0xE0 &&
			headerBytes[4] == 0xA1 && headerBytes[5] == 0xB1 && headerBytes[6] == 0x1A && headerBytes[7] == 0xE1
	case "zip":
		return len(headerBytes) >= 4 &&
			headerBytes[0] == 0x50 && headerBytes[1] == 0x4B && headerBytes[2] == 0x03 && headerBytes[3] == 0x04
	case "rar":
		return len(headerBytes) >= 7 &&
			headerBytes[0] == 'R' && headerBytes[1] == 'a' && headerBytes[2] == 'r' && headerBytes[3] == '!' &&
			headerBytes[4] == 0x1A && headerBytes[5] == 0x07 && headerBytes[6] == 0x00
	case "7z":
		return len(headerBytes) >= 6 &&
			headerBytes[0] == 0x37 && headerBytes[1] == 0x7A && headerBytes[2] == 0xBC && headerBytes[3] == 0xAF &&
			headerBytes[4] == 0x27 && headerBytes[5] == 0x1C
	case "txt", "md", "csv":
		return true
	case "svg":
		// 原来这里少了外层括号：&& 优先于 ||，短文件会走到第二个分支导致越界 panic
		return len(headerBytes) >= 4 &&
			((headerBytes[0] == '<' && headerBytes[1] == 's' && headerBytes[2] == 'v' && headerBytes[3] == 'g') ||
				(headerBytes[0] == '<' && headerBytes[1] == '?' && headerBytes[2] == 'x' && headerBytes[3] == 'm'))
	case "mp4", "m4v", "mov", "3gp", "3g2":
		// ISO Base Media 容器：第 4~7 字节固定是 "ftyp"，第 8 字节起是品牌
		// （qt  = QuickTime/mov，其余如 isom / mp42 / heic 系列）。
		// mov 与 mp4 只认容器头，不按品牌卡死：iPhone 的实况图视频经常是
		// 「.mov 扩展名 + mp4 品牌」，反之也有，卡太严会误伤。
		return isFtypContainer(headerBytes)
	case "webm", "mkv":
		// EBML 头（WebM / MKV 同源，两者只能靠 DocType 区分，这里不再细分）
		return len(headerBytes) >= 4 &&
			headerBytes[0] == 0x1A && headerBytes[1] == 0x45 && headerBytes[2] == 0xDF && headerBytes[3] == 0xA3
	case "avi":
		return len(headerBytes) >= 12 &&
			string(headerBytes[0:4]) == "RIFF" && string(headerBytes[8:12]) == "AVI "
	case "flv":
		return len(headerBytes) >= 4 &&
			headerBytes[0] == 'F' && headerBytes[1] == 'L' && headerBytes[2] == 'V'
	case "wmv", "asf":
		// ASF 头：30 26 B2 75 8E 66 CF 11
		return len(headerBytes) >= 8 &&
			headerBytes[0] == 0x30 && headerBytes[1] == 0x26 && headerBytes[2] == 0xB2 && headerBytes[3] == 0x75 &&
			headerBytes[4] == 0x8E && headerBytes[5] == 0x66 && headerBytes[6] == 0xCF && headerBytes[7] == 0x11
	case "mpg", "mpeg", "m2v":
		// MPEG-PS 起始码 00 00 01 BA（打包）/ B3（视频序列）
		return len(headerBytes) >= 4 &&
			headerBytes[0] == 0x00 && headerBytes[1] == 0x00 && headerBytes[2] == 0x01 &&
			(headerBytes[3] == 0xBA || headerBytes[3] == 0xB3)
	case "ogv", "ogg", "oga":
		return len(headerBytes) >= 4 && string(headerBytes[0:4]) == "OggS"
	case "tar":
		// POSIX tar 在偏移 257 处有 "ustar"（读取的头部缓冲是 512 字节，够用）
		return len(headerBytes) >= 262 && string(headerBytes[257:262]) == "ustar"
	case "gz", "tgz":
		return len(headerBytes) >= 2 && headerBytes[0] == 0x1F && headerBytes[1] == 0x8B
	case "bz2", "tbz2":
		return len(headerBytes) >= 3 && headerBytes[0] == 'B' && headerBytes[1] == 'Z' && headerBytes[2] == 'h'
	case "xz", "txz":
		return len(headerBytes) >= 6 &&
			headerBytes[0] == 0xFD && headerBytes[1] == 0x37 && headerBytes[2] == 0x7A &&
			headerBytes[3] == 0x58 && headerBytes[4] == 0x5A && headerBytes[5] == 0x00
	default:
		return false
	}
}

// sniffFileExt - 按文件头（magic bytes）判断「内容实际是什么类型」
//
// 用途：内容与扩展名不符时（QQ / 微信保存的二维码、浏览器另存为的图片，经常是
// 「文件名 .jpg、内容其实是 PNG / WebP」），把扩展名按真实内容纠回来；
// 纠不了时也能给出「实际是 XX」的提示，而不是一句「不匹配」让人没头绪。
//
// 只返回**能唯一判定**的类型：doc / xls / ppt 同为 OLE2 头、docx / xlsx / pptx 与 zip
// 同为 PK 头，光看文件头分不出来，这些返回空（仍由 verifyFileContent 按扩展名判定）。
func (this *Attachment) sniffFileExt(headerBytes []byte) (ext string, label string) {
	has := func(sign ...byte) bool {
		if len(headerBytes) < len(sign) {
			return false
		}
		for index, item := range sign {
			if headerBytes[index] != item {
				return false
			}
		}
		return true
	}

	switch {
	case has(0xFF, 0xD8, 0xFF):
		return "jpg", "JPEG 图片"
	case has(0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A):
		return "png", "PNG 图片"
	case has('G', 'I', 'F', '8'):
		return "gif", "GIF 图片"
	case has('R', 'I', 'F', 'F') && len(headerBytes) >= 12 &&
		headerBytes[8] == 'W' && headerBytes[9] == 'E' && headerBytes[10] == 'B' && headerBytes[11] == 'P':
		return "webp", "WebP 图片"
	case has('B', 'M'):
		return "bmp", "BMP 图片"
	case has('%', 'P', 'D', 'F'):
		return "pdf", "PDF 文档"
	case has('R', 'a', 'r', '!', 0x1A, 0x07, 0x00):
		return "rar", "RAR 压缩包"
	case has(0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C):
		return "7z", "7z 压缩包"
	case has(0x1A, 0x45, 0xDF, 0xA3):
		// EBML 头：WebM 与 MKV 同源，靠 DocType 里的关键字区分
		if bytes.Contains(headerBytes, []byte("matroska")) {
			return "mkv", "Matroska 视频"
		}
		return "webm", "WebM 视频"
	case has('R', 'I', 'F', 'F') && len(headerBytes) >= 12 &&
		headerBytes[8] == 'A' && headerBytes[9] == 'V' && headerBytes[10] == 'I' && headerBytes[11] == ' ':
		return "avi", "AVI 视频"
	case has(0x00, 0x00, 0x01, 0x00):
		return "ico", "ICO 图标"
	case has('I', 'I', 0x2A, 0x00), has('M', 'M', 0x00, 0x2A):
		return "tiff", "TIFF 图片"
	case has('F', 'L', 'V'):
		return "flv", "FLV 视频"
	case has(0x30, 0x26, 0xB2, 0x75):
		return "wmv", "WMV 视频"
	case has(0x1F, 0x8B):
		return "gz", "GZIP 压缩包"
	case has('B', 'Z', 'h'):
		return "bz2", "BZIP2 压缩包"
	case has(0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00):
		return "xz", "XZ 压缩包"
	case has('O', 'g', 'g', 'S'):
		return "ogg", "OGG 音频 / 视频"
	case has('M', 'S', 'C', 'F'):
		return "cab", "CAB 压缩包"
	case has(0x00, 0x00, 0x01, 0xBA), has(0x00, 0x00, 0x01, 0xB3):
		return "mpg", "MPEG 视频"
	}

	// ISO Base Media（mp4 / mov / 3gp / heic / avif）：ftyp 在第 4 字节，品牌在第 8 字节
	if brand := ftypBrand(headerBytes); brand != "" {
		switch brand {
		case "qt  ":
			return "mov", "QuickTime 视频"
		case "avif", "avis":
			return "avif", "AVIF 图片"
		case "heic", "heix", "heim", "heis", "hevc", "hevx", "mif1", "msf1":
			return "heic", "HEIC 图片"
		case "3gp4", "3gp5", "3gp6", "3g2a", "3g2b", "3g2c":
			return "3gp", "3GP 视频"
		}
		return "mp4", "MP4 视频"
	}

	return "", ""
}

func (this *Attachment) sanitizeSVG(content string) string {
	content = svgScriptPattern.ReplaceAllString(content, "")
	content = svgScriptTagPattern.ReplaceAllString(content, "")
	content = svgCDataPattern.ReplaceAllString(content, "")
	content = svgOnAttrQuotedPattern.ReplaceAllString(content, "")
	content = svgOnAttrPattern.ReplaceAllString(content, "")
	content = svgForeignPattern.ReplaceAllString(content, "")
	content = svgForeignTagPattern.ReplaceAllString(content, "")
	content = svgJavascriptPattern.ReplaceAllString(content, "")
	content = svgVbscriptPattern.ReplaceAllString(content, "")
	return content
}

func (this *Attachment) checkUploadLimit(ctx *gin.Context, userId uint, count int) bool {
	config := facade.AttachmentConfigInstance
	if config == nil {
		return true
	}

	if config.ConcurrentLimit > 0 {
		if !this.tryAcquireUploadSlot(count) {
			this.json(ctx, nil, facade.Lang(ctx, "并发上传数量已达上限（%d个）！", config.ConcurrentLimit), 400)
			return false
		}
	}

	return true
}

func (this *Attachment) decrementUploadCounter(count int) {
	if facade.AttachmentConfigInstance != nil && facade.AttachmentConfigInstance.ConcurrentLimit > 0 {
		this.releaseUploadSlot(count)
	}
}

func (this *Attachment) tryAcquireUploadSlot(count int) bool {
	config := facade.AttachmentConfigInstance
	if config == nil || config.ConcurrentLimit <= 0 {
		return true
	}

	// 优先使用共享计数器（redis 下多实例共享）
	//
	// 之前直接操作 facade.Redis.Client：既绕过了缓存驱动的 prefix 与故障降级，
	// 又因为 facade.Redis 永远非 nil（initCache 无条件创建），在「没部署 redis、
	// 驱动是 file」的站点上每次上传都会先连一次 Redis 失败、打一条 Warn 再降级。
	if facade.Cache != nil {
		key := "attachment:upload_concurrent_counter"
		if current, ok := facade.Cache.Incr(key, int64(count)); ok {
			if current > int64(config.ConcurrentLimit) {
				facade.Cache.Incr(key, -int64(count))
				return false
			}
			if current == int64(count) {
				facade.Cache.Expire(key, 5*time.Minute)
			}
			return true
		}
		facade.Log.Warn(map[string]any{}, "缓存计数器不可用，上传并发计数降级为本地计数器")
	}

	// 缓存不可用时降级为本地计数器，避免因缓存服务故障导致上传被错误拒绝
	return this.tryAcquireUploadSlotLocal(count)
}

func (this *Attachment) tryAcquireUploadSlotLocal(count int) bool {
	config := facade.AttachmentConfigInstance
	if config == nil || config.ConcurrentLimit <= 0 {
		return true
	}

	uploadCounterMutex.Lock()
	defer uploadCounterMutex.Unlock()

	uploadConcurrentCounter += count
	if uploadConcurrentCounter > config.ConcurrentLimit {
		uploadConcurrentCounter -= count
		return false
	}
	return true
}

func (this *Attachment) releaseUploadSlot(count int) {
	// 优先释放共享计数器（key 已过期时不再自减，避免留下负数）
	key := "attachment:upload_concurrent_counter"
	if facade.Cache != nil && facade.Cache.Has(key) {
		if _, ok := facade.Cache.Incr(key, -int64(count)); ok {
			return
		}
	}

	// Redis 不可用时释放本地计数器
	uploadCounterMutex.Lock()
	uploadConcurrentCounter -= count
	if uploadConcurrentCounter < 0 {
		uploadConcurrentCounter = 0
	}
	uploadCounterMutex.Unlock()
}

func (this *Attachment) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"one":     this.one,
		"all":     this.all,
		"sum":     this.sum,
		"min":     this.min,
		"max":     this.max,
		"rand":    this.rand,
		"count":   this.count,
		"column":  this.column,
		"list":    this.list,
		"emoji":   this.emoji,
		"storage": this.storage,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
}

func (this *Attachment) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"save":      this.save,
		"create":    this.create,
		"batch":     this.batch,
		"checktype": this.checkType,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
	if method != "checktype" {
		go this.delCache()
	}
}

func (this *Attachment) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
	go this.delCache()
}

func (this *Attachment) IDEL(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"remove": this.remove,
		"delete": this.delete,
		"clear":  this.clear,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
	go this.delCache()
}

func (this *Attachment) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "附件管理接口"), 200)
}

func (this *Attachment) delCache() {
	facade.Cache.DelTags([]any{"[GET]", "attachment"})
}

func (this *Attachment) one(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	table := model.Attachment{}

	for key, val := range params {
		if utils.In.Array(key, attachmentAllowQuerySlice) {
			utils.Struct.Set(&table, key, val)
		}
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		query := this.withTrashOptions(facade.DB.Model(&table), params)
		query = this.buildQuery(query, params)

		item, _ := query.Where(table).Find()
		data = facade.Comm.WithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

// uploaderRows - 把列表结果统一成 []map[string]any
//
// 同一份数据可能来自数据库查询（[]map[string]any），也可能来自缓存：
// file / redis 驱动会 JSON 往返，取回来是 []any，所以这里统一转换一次。
func (this *Attachment) uploaderRows(data any) []map[string]any {
	rows := make([]map[string]any, 0)

	for _, item := range cast.ToSlice(data) {
		if row := cast.ToStringMap(item); len(row) > 0 {
			rows = append(rows, row)
		}
	}

	return rows
}

// appendUploaderName - 为附件列表补充上传者的账号 / 昵称（一次性批量查询，避免 N+1）
//
// 附件表只存 uploader_id，列表里直接显示数字没有意义，这里按 uploader_id 批量取用户，
// 给每行加两个字段：
//   - uploader_account：账号（可能为空）
//   - uploader_name：昵称；昵称为空时回退账号，两者都取不到时用「用户 #id」占位
//
// 非管理员（只能看到自己的附件）也会带上自己的昵称，展示更友好。
func (this *Attachment) appendUploaderName(items []map[string]any) []map[string]any {
	ids := make([]any, 0)
	seen := make(map[int]struct{})

	for _, item := range items {
		uid := cast.ToInt(item["uploader_id"])
		if uid <= 0 {
			item["uploader_account"] = ""
			item["uploader_name"] = ""
			continue
		}
		if _, exist := seen[uid]; exist {
			continue
		}
		seen[uid] = struct{}{}
		ids = append(ids, uid)
	}

	if len(ids) == 0 {
		return items
	}

	users, err := facade.DB.Model(&[]model.Users{}).WhereIn("id", ids).Field("id", "account", "nickname").Select()
	if err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "附件上传者信息查询失败")
		return items
	}

	accounts := make(map[int]string, len(users))
	nicknames := make(map[int]string, len(users))
	for _, user := range users {
		uid := cast.ToInt(user["id"])
		accounts[uid] = strings.TrimSpace(cast.ToString(user["account"]))
		nicknames[uid] = strings.TrimSpace(cast.ToString(user["nickname"]))
	}

	for _, item := range items {
		uid := cast.ToInt(item["uploader_id"])
		if uid <= 0 {
			continue
		}

		account, nickname := accounts[uid], nicknames[uid]
		if utils.Is.Empty(nickname) {
			nickname = account
		}
		if utils.Is.Empty(nickname) {
			nickname = fmt.Sprintf("用户 #%v", uid)
		}

		item["uploader_account"] = account
		item["uploader_name"] = nickname
	}

	return items
}

func (this *Attachment) all(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx, map[string]any{
		"page":  1,
		"order": "create_time desc",
	})

	table := model.Attachment{}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)

	if !this.meta.root(ctx) && limit > 100 {
		limit = 100
	}

	query := this.withTrashOptions(facade.DB.Model(&[]model.Attachment{}), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	count, _ := query.Where(table).Count()

	cacheName := this.cache.name(ctx)
	// 缓存名只由请求参数决定（不含调用者身份），而非管理员的列表是按 uploader_id
	// 过滤后的「自己的数据」（含 uploader_account），必须带上用户标识做隔离，
	// 否则另一个用户用相同参数就会命中同一个缓存 key 看到别人的附件与账号
	if !this.meta.root(ctx) {
		cacheName = fmt.Sprintf("%s&uid=%v", cacheName, this.meta.user(ctx).Id)
	}

	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item, _ := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	// 补上传者账号 / 昵称（uploader_account / uploader_name）：缓存存的是原始行，
	// 这里每次响应时补，用户改了昵称立即生效；field 裁剪过字段时只在调用方显式
	// 要了 uploader_* 才补，避免污染裁剪结果
	if utils.Is.Empty(params["field"]) || strings.Contains(cast.ToString(params["field"]), "uploader") {
		if rows := this.uploaderRows(data); len(rows) > 0 {
			data = this.appendUploaderName(rows)
		}
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, gin.H{
		"data":  data,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Attachment) list(ctx *gin.Context) {
	params := this.params(ctx, map[string]any{"page": 1, "limit": 10, "order": "create_time desc"})
	userId := this.meta.user(ctx).Id
	if userId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	page := cast.ToInt(params["page"])
	limit := this.meta.limit(ctx)
	data, count := (&model.Attachment{}).GetByUploader(uint(userId), page, limit)
	if utils.Is.Empty(data) {
		this.json(ctx, gin.H{"data": []any{}, "count": 0, "page": 0}, facade.Lang(ctx, "暂无附件！"), 204)
		return
	}
	this.json(ctx, gin.H{"data": utils.ArrayMapWithField(data, params["field"]), "count": count, "page": math.Ceil(float64(count) / float64(limit))}, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Attachment) rand(ctx *gin.Context) {
	params := this.params(ctx)
	limit := this.meta.limit(ctx)
	if limit > 100 {
		limit = 100
	}
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&[]model.Attachment{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	query = this.buildQuery(query, params).OrderRaw("RAND()").Limit(limit)

	items, _ := query.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
}

// storage - 当前启用的存储方式
//
// 附件表里的 storage_driver 记录的是**每个附件上传时**用的驱动（历史数据里可能还残留已下线的
// oss / kodo），这里返回的是配置（config/storage.toml 的 default）当前实际生效的驱动，
// 后台附件概览用它告诉管理员「新上传的文件会存到哪儿」。
//
// 未知值一律回退本地存储，与 facade.NewStorage / getStorageDriver 的口径保持一致。
func (this *Attachment) storage(ctx *gin.Context) {

	driver := strings.ToLower(strings.TrimSpace(cast.ToString(facade.StorageToml.Get("default"))))
	if driver != facade.StorageModeCOS {
		driver = facade.StorageModeLocal
	}

	this.json(ctx, gin.H{
		"driver": driver,
		"name":   storageDriverLabel(driver),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// storageDriverLabel - 存储驱动的展示名（与前端「系统设置 → 存储」的选项文案一致）
func storageDriverLabel(driver string) string {
	if driver == facade.StorageModeCOS {
		return "腾讯云 COS"
	}
	return "本地存储"
}

func (this *Attachment) count(ctx *gin.Context) {
	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Attachment{}), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	count, _ := query.Count()
	this.json(ctx, count, facade.Lang(ctx, "查询成功！"), 200)
}

func (this *Attachment) aggregateQuery(ctx *gin.Context, aggFunc func(query *facade.ModelStruct, field string) any) (any, string) {
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)
	query := this.withTrashOptions(facade.DB.Model(&model.Attachment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	fields := utils.Unity.Keys(params["field"])

	if utils.Is.Empty(fields) {
		return nil, ""
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		result := make(map[string]any)
		for _, val := range fields {
			result[cast.ToString(val)] = aggFunc(query, cast.ToString(val))
		}
		data = result
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		msg[0] = "数据请求成功！"
	}

	return data, facade.Lang(ctx, strings.Join(msg, ""))
}

func (this *Attachment) sum(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Sum(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Attachment) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Min(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Attachment) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		result, _ := query.Max(field)
		return result
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Attachment) column(ctx *gin.Context) {
	code := 204
	msg := []string{"无数据！", ""}
	var data any

	params := this.params(ctx)

	hasFilter := !utils.Is.Empty(params["where"]) || !utils.Is.Empty(params["or"]) ||
		!utils.Is.Empty(params["like"]) || !utils.Is.Empty(params["not"]) ||
		!utils.Is.Empty(params["null"]) || !utils.Is.Empty(params["notNull"]) ||
		!utils.Is.Empty(params["ids"])

	if !hasFilter && !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "必须携带筛选条件（where/or/like/not/null/notNull/ids）！"), 400)
		return
	}

	query := this.withTrashOptions(facade.DB.Model(&[]model.Attachment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	if !this.meta.root(ctx) {
		query = query.Limit(100)
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		items, _ := query.Select()
		data = utils.ArrayMapWithField(items, params["field"])
		this.setCache(ctx, cacheName, data)
	}

	if !utils.Is.Empty(data) {
		code = 200
		msg[0] = "数据请求成功！"
	}

	this.json(ctx, data, facade.Lang(ctx, strings.Join(msg, "")), code)
}

func (this *Attachment) save(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["id"]) && utils.Is.Empty(params["uuid"]) {
		this.create(ctx)
	} else {
		this.update(ctx)
	}
}

func (this *Attachment) create(ctx *gin.Context) {
	params := this.params(ctx)
	userId := this.meta.user(ctx).Id
	if userId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	table := model.Attachment{
		Uuid:       (&model.Attachment{}).GenerateUUID(),
		UploaderId: uint(userId),
	}

	allowFields := append([]any{}, attachmentAllowFieldsSlice...)

	for key, val := range params {
		if utils.In.Array(key, allowFields) {
			utils.Struct.Set(&table, key, val)
		}
	}

	_, err := facade.DB.Model(&table).Create(&table)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id, "uuid": table.Uuid}, facade.Lang(ctx, "创建成功！"), 200)
}

type uploadResult struct {
	Attachment *model.Attachment
	Existing   map[string]any
	Error      error
	IsExist    bool
	IsSuccess  bool
	// MotionVideo 实况图拆出的视频访问地址（安卓实况照片的 jpg 里内嵌 MP4，
	// 见 motion_photo.go）；普通图片为空
	MotionVideo string
}

func (this *Attachment) uploadSingleFile(ctx *gin.Context, fileHeader *multipart.FileHeader, userId uint, params map[string]any) *uploadResult {
	result := &uploadResult{}

	config := facade.AttachmentConfigInstance
	if config == nil {
		result.Error = fmt.Errorf("附件配置未初始化！")
		return result
	}

	if fileHeader.Size > config.GetMaxFileSizeBytes() {
		result.Error = fmt.Errorf("文件大小超过限制（最大%dKB）", config.MaxFileSize)
		return result
	}

	file, err := fileHeader.Open()
	if err != nil {
		result.Error = fmt.Errorf("打开文件失败")
		return result
	}
	defer file.Close()

	fileName := this.sanitizeFileName(fileHeader.Filename)
	// suffix 始终是「原始文件名里的后缀」，fileExt 是「最终用来存储的扩展名」，
	// 两者在下面「内容与扩展名不符」时会分叉（fileExt 按真实内容纠正，suffix 不动，
	// 这样命名规则里的 {filename} 依旧能正确去掉原后缀）
	suffix := ""
	fileExt := ""
	if lastIndex := strings.LastIndex(fileName, "."); lastIndex > 0 {
		suffix = strings.ToLower(fileName[lastIndex:])
		fileExt = strings.ToLower(fileName[lastIndex+1:])
	}
	if !config.IsExtensionAllowed(fileExt) {
		result.Error = fmt.Errorf("不允许上传该类型的文件！")
		return result
	}

	bufferSize := 512
	if fileHeader.Size < int64(bufferSize) {
		bufferSize = int(fileHeader.Size)
	}
	headerBytes := make([]byte, bufferSize)
	n, err := file.Read(headerBytes)
	if err != nil && err != io.EOF {
		result.Error = fmt.Errorf("读取文件失败")
		return result
	}
	headerBytes = headerBytes[:n]

	if !this.verifyFileContent(headerBytes, fileExt) {
		// 内容与扩展名不符：QQ / 微信保存的二维码图、浏览器「另存为」的图片常常是
		// 「文件名 .jpg，内容其实是 PNG / WebP」，直接拒掉太不友好。
		// 这里按文件头判断真实类型：
		//   - 能判定且本来就在允许列表里 → 按真实内容纠正扩展名（存储与 MIME 都用真实类型，用户无感）；
		//   - 判定不了 / 判定出的类型不被允许 → 报错时带上「实际是 XX」，方便用户改后缀后重传。
		actual, label := this.sniffFileExt(headerBytes)
		if actual == "" || !config.IsExtensionAllowed(actual) {
			if label != "" {
				result.Error = fmt.Errorf("文件内容与扩展名不匹配：内容的真实类型是 %s，请把后缀改成 .%s 后再上传", label, actual)
			} else {
				result.Error = fmt.Errorf("文件内容与扩展名不匹配！")
			}
			return result
		}

		facade.Log.Info(map[string]any{
			"file": fileName,
			"from": fileExt,
			"to":   actual,
		}, "上传文件的内容类型与扩展名不符，已按真实内容纠正扩展名")

		fileExt = actual
	}

	mimeType := http.DetectContentType(headerBytes)

	var uploadReader io.Reader
	var fileContent []byte
	// 安卓实况照片（Motion Photo）：JPEG 尾巴里追加了一段 MP4。
	// 这里先整份读出来拆分，主附件只存封面（截到视频起始处），
	// 视频随后单独存一条附件，前端按「图片::视频」使用。
	var motionVideo []byte
	uploadSize := fileHeader.Size

	if fileExt == "svg" {
		fileContent, err = io.ReadAll(file)
		if err != nil {
			result.Error = fmt.Errorf("读取文件失败")
			return result
		}
		sanitizedContent := this.sanitizeSVG(string(fileContent))
		uploadReader = strings.NewReader(sanitizedContent)
	} else if fileExt == "jpg" || fileExt == "jpeg" {
		// 图片可能有几十 MB，但附件大小在上面已经卡过上限，整读是安全的
		if seeker, ok := file.(io.Seeker); ok {
			if _, seekErr := seeker.Seek(0, io.SeekStart); seekErr == nil {
				if content, readErr := io.ReadAll(file); readErr == nil {
					if cover, video, found := splitMotionPhoto(content); found {
						motionVideo = video
						uploadReader = bytes.NewReader(cover)
						uploadSize = int64(len(cover))
					}
				}
			}
		}
	}

	if uploadReader == nil {
		if seeker, ok := file.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				result.Error = fmt.Errorf("重置文件指针失败")
				return result
			}
		}
		uploadReader = file
	}

	// 上传命名规则：目录结构（dir_rule）与文件名（file_rule）都能在
	// 「系统设置 → 存储」里自定义，见 facade/storage-rule.go；
	// {filename} 用不含扩展名的原始名称，扩展名由驱动自动追加
	ruleCtx := facade.StorageRuleContext{
		Uid:      int(userId),
		Filename: strings.TrimSuffix(fileName, suffix),
		Ext:      fileExt,
	}
	key := facade.Storage.Path(ruleCtx)
	hash := sha256.New()
	hashReader := io.TeeReader(uploadReader, hash)
	item := facade.Storage.Upload(key, hashReader)
	if item.Error != nil {
		result.Error = fmt.Errorf("上传文件失败")
		return result
	}
	// 存储文件名（与对象键一致）：命名规则生成的名字 + 扩展名
	saveName := path.Base(key)
	fileHash := fmt.Sprintf("%x", hash.Sum(nil))

	existing := (&model.Attachment{}).GetByHash(fileHash)
	if !utils.Is.Empty(existing) {
		this.safeDeleteFile(item.Path)
		existingUploaderId := cast.ToUint(existing["uploader_id"])
		if existingUploaderId != userId && !this.meta.root(ctx) {
			result.Error = fmt.Errorf("文件已存在")
			return result
		}
		result.Existing = existing
		result.IsExist = true
		// 封面命中已有附件时也要把视频挂上：同一条实况图重复上传
		// 不该出现「第一次有视频、第二次退化成静态图」
		if len(motionVideo) > 0 {
			result.MotionVideo = this.saveMotionVideo(fileName, suffix, motionVideo, userId, params)
		}
		return result
	}

	fullUrl := item.Domain + item.Path
	attachment := model.Attachment{
		Uuid: (&model.Attachment{}).GenerateUUID(), OriginalName: fileName, SaveName: saveName,
		SavePath: item.Path, FullUrl: fullUrl, FileSize: uploadSize,
		MimeType: mimeType, FileExt: fileExt,
		StorageDriver: cast.ToString(facade.StorageToml.Get("default")), UploaderId: userId,
		TargetType: cast.ToString(params["target_type"]), TargetId: cast.ToUint(params["target_id"]),
		FileHash: fileHash,
	}
	_, err = facade.DB.Model(&attachment).Create(&attachment)
	if err != nil {
		this.safeDeleteFile(item.Path)
		result.Error = fmt.Errorf("保存附件记录失败")
		return result
	}

	// 实况图：视频单独存一条附件，返回地址供前端拼「图片::视频」
	if len(motionVideo) > 0 {
		result.MotionVideo = this.saveMotionVideo(fileName, suffix, motionVideo, userId, params)
	}

	result.Attachment = &attachment
	result.IsSuccess = true
	return result
}

/**
 * saveMotionVideo - 把安卓实况图里拆出的视频单独存成一条附件
 *
 * 走与普通附件相同的存储规则（目录 / 文件名 / 驱动），并按内容 hash 去重，
 * 这样重复上传同一张实况图不会产生多份视频。
 *
 * 站点没放开 mp4 时静默跳过：图仍能正常用，只是没有实况效果。
 *
 * @return 视频的访问地址（失败 / 跳过时为空）
 */
func (this *Attachment) saveMotionVideo(fileName string, suffix string, content []byte, userId uint, params map[string]any) string {

	config := facade.AttachmentConfigInstance
	if config == nil || len(content) == 0 {
		return ""
	}

	if !config.IsExtensionAllowed("mp4") {
		facade.Log.Info(map[string]any{
			"file": fileName,
		}, "站点未允许 mp4，实况图视频已跳过（图片仍正常上传）")
		return ""
	}

	fileHash := fmt.Sprintf("%x", sha256.Sum256(content))
	if existing := (&model.Attachment{}).GetByHash(fileHash); !utils.Is.Empty(existing) {
		return utils.Replace(cast.ToString(existing["full_url"]), model.DomainTemp1())
	}

	// 与封面同一套命名规则，只把扩展名换成 mp4；
	// 名字加 -live 后缀，附件库里一眼能看出这是实况图拆出来的视频
	baseName := strings.TrimSuffix(fileName, suffix)
	key := facade.Storage.Path(facade.StorageRuleContext{
		Uid:      int(userId),
		Filename: baseName + "-live",
		Ext:      "mp4",
	})
	item := facade.Storage.Upload(key, bytes.NewReader(content))
	if item.Error != nil {
		facade.Log.Error(map[string]any{"file": fileName, "error": item.Error}, "实况图视频上传失败")
		return ""
	}

	attachment := model.Attachment{
		Uuid:          (&model.Attachment{}).GenerateUUID(),
		OriginalName:  baseName + "-live.mp4",
		SaveName:      path.Base(key),
		SavePath:      item.Path,
		FullUrl:       item.Domain + item.Path,
		FileSize:      int64(len(content)),
		MimeType:      "video/mp4",
		FileExt:       "mp4",
		StorageDriver: cast.ToString(facade.StorageToml.Get("default")),
		UploaderId:    userId,
		TargetType:    cast.ToString(params["target_type"]),
		TargetId:      cast.ToUint(params["target_id"]),
		FileHash:      fileHash,
	}

	if _, err := facade.DB.Model(&attachment).Create(&attachment); err != nil {
		this.safeDeleteFile(item.Path)
		facade.Log.Error(map[string]any{"file": fileName, "error": err.Error()}, "实况图视频记录保存失败")
		return ""
	}

	return utils.Replace(attachment.FullUrl, model.DomainTemp1())
}

func (this *Attachment) checkType(ctx *gin.Context) {
	params := this.params(ctx)
	fileNames, ok := params["file_names"].([]any)
	if !ok || len(fileNames) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请提供文件名列表！"), 400)
		return
	}

	config := facade.AttachmentConfigInstance
	if config == nil {
		this.json(ctx, nil, facade.Lang(ctx, "附件配置未初始化！"), 500)
		return
	}

	var results []map[string]any
	for _, fileName := range fileNames {
		name := cast.ToString(fileName)
		fileExt := ""
		if lastIndex := strings.LastIndex(name, "."); lastIndex > 0 {
			fileExt = strings.ToLower(name[lastIndex+1:])
		}

		isAllowed := config.IsExtensionAllowed(fileExt)
		results = append(results, map[string]any{
			"file_name":  name,
			"file_ext":   fileExt,
			"is_allowed": isAllowed,
			"message":    utils.Ternary(isAllowed, "文件类型允许上传", "不允许上传该类型的文件"),
		})
	}

	allowedCount := 0
	for _, r := range results {
		if cast.ToBool(r["is_allowed"]) {
			allowedCount++
		}
	}

	this.json(ctx, gin.H{
		"results":          results,
		"allowed_count":    allowedCount,
		"disallowed_count": len(results) - allowedCount,
		"allow_extensions": config.AllowExtensions,
		"max_file_size":    config.MaxFileSize,
		"max_file_size_kb": config.MaxFileSize,
		"max_file_size_mb": float64(config.MaxFileSize) / 1024,
	}, facade.Lang(ctx, "检查完成！"), 200)
}

func (this *Attachment) batch(ctx *gin.Context) {
	userId := this.meta.user(ctx).Id
	if userId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	config := facade.AttachmentConfigInstance
	if config == nil {
		this.json(ctx, nil, facade.Lang(ctx, "附件配置未初始化！"), 500)
		return
	}

	var files []*multipart.FileHeader

	form, err := ctx.MultipartForm()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取表单失败：%v", err.Error()), 400)
		return
	}

	if formFiles, ok := form.File["files"]; ok && len(formFiles) > 0 {
		files = formFiles
	} else {
		if fileHeader, err := ctx.FormFile("file"); err == nil {
			files = []*multipart.FileHeader{fileHeader}
		}
	}

	if len(files) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请选择要上传的文件！"), 400)
		return
	}
	if config.ConcurrentLimit > 0 && len(files) > config.ConcurrentLimit {
		this.json(ctx, nil, facade.Lang(ctx, "单次最多上传%d个文件！", config.ConcurrentLimit), 400)
		return
	}

	if !this.checkUploadLimit(ctx, uint(userId), len(files)) {
		return
	}

	defer this.decrementUploadCounter(len(files))

	var results []map[string]any
	var successCount, failCount int
	params := this.params(ctx)
	for _, fileHeader := range files {
		result := this.uploadSingleFile(ctx, fileHeader, uint(userId), params)

		if result.Error != nil {
			failCount++
			results = append(results, map[string]any{
				"original_name": fileHeader.Filename,
				"status":        "fail",
				"error":         result.Error.Error(),
			})
			continue
		}

		if result.IsExist {
			row := map[string]any{
				"original_name": result.Existing["original_name"],
				"full_url":      utils.Replace(cast.ToString(result.Existing["full_url"]), model.DomainTemp1()),
				"status":        "exist",
			}
			// 实况图（安卓 Motion Photo）：附上拆出来的视频地址，前端自动配成实况图
			if result.MotionVideo != "" {
				row["live_video"] = result.MotionVideo
			}
			results = append(results, row)
			successCount++
			continue
		}

		if result.IsSuccess && result.Attachment != nil {
			row := map[string]any{
				"id": result.Attachment.Id, "uuid": result.Attachment.Uuid,
				"original_name": result.Attachment.OriginalName,
				"full_url":      utils.Replace(result.Attachment.FullUrl, model.DomainTemp1()),
				"file_size":     result.Attachment.FileSize,
				"status":        "success",
			}
			// 实况图（安卓 Motion Photo）：附上拆出来的视频地址，前端自动配成实况图
			if result.MotionVideo != "" {
				row["live_video"] = result.MotionVideo
			}
			results = append(results, row)
			successCount++
		} else {
			failCount++
			results = append(results, map[string]any{
				"original_name": fileHeader.Filename,
				"status":        "fail",
				"error":         "上传失败",
			})
		}
	}
	facade.Log.Info(map[string]any{"user_id": userId, "success_count": successCount, "fail_count": failCount}, "批量上传附件")

	msg := "上传完成！"
	if successCount == 0 && failCount > 0 {
		msg = "所有文件上传失败！"
	} else if successCount > 0 && failCount > 0 {
		msg = "部分文件上传成功！"
	}

	code := 200
	if successCount == 0 && failCount > 0 {
		code = 400
	} else if successCount > 0 && failCount > 0 {
		code = 207
	}

	this.json(ctx, gin.H{"results": results, "success": successCount, "fail": failCount}, facade.Lang(ctx, msg), code)
}

func (this *Attachment) update(ctx *gin.Context) {
	params := this.params(ctx)

	if utils.Is.Empty(params["id"]) && utils.Is.Empty(params["uuid"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "id/uuid"), 400)
		return
	}

	var table model.Attachment
	var query *facade.ModelStruct

	if !utils.Is.Empty(params["uuid"]) {
		query = facade.DB.Model(&table).WithTrashed().Where("uuid", params["uuid"])
	} else {
		query = facade.DB.Model(&table).WithTrashed().Where("id", params["id"])
	}

	item, _ := query.Find()
	if utils.Is.Empty(item) {
		this.json(ctx, nil, facade.Lang(ctx, "附件不存在！"), 204)
		return
	}

	if !this.meta.root(ctx) && cast.ToInt(item["uploader_id"]) != this.meta.user(ctx).Id {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	async := utils.Async[map[string]any]()
	allowFields := append([]any{}, attachmentAllowFieldsSlice...)

	for key, val := range params {
		if utils.In.Array(key, allowFields) {
			async.Set(key, val)
		}
	}

	if len(async.Result()) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "没有需要更新的字段！"), 400)
		return
	}

	_, err := query.Update(async.Result())

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": item["id"], "uuid": item["uuid"]}, facade.Lang(ctx, "更新成功！"), 200)
}

// getStorageDriver 按附件记录里的 storage_driver 选择驱动
//
// 仅保留两种驱动：local（本地存储）与 cos（腾讯云 COS）；
// 历史数据里若残留 oss / kodo（旧版本上传的附件），这里回退到本地存储 ——
// 对应文件已无法通过本程序删除，需要自行在对象存储控制台清理。
func getStorageDriver(driver string) facade.StorageInterface {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case facade.StorageModeCOS:
		return facade.COS
	default:
		return facade.LocalStorage
	}
}

/**
 * purgeFiles 删除存储里的物理文件
 *
 * 入参：驱动 → 相对路径（save_path）列表，返回成功删除的路径集合与失败的路径列表。
 * - 本地存储（local）：逐个删除，计数精确（Delete 已做幂等处理，文件不存在也算成功）；
 * - 腾讯云 COS：按 500 个一批调用批量接口，某批失败则该批整体记为失败，
 *   调用方据此保留对应记录，避免「记录删了、文件还在」产生孤儿文件。
 */
func (this *Attachment) purgeFiles(filesByDriver map[string][]string) (deleted map[string]bool, failed []string) {

	const batchSize = 500

	deleted = make(map[string]bool)

	for driver, paths := range filesByDriver {
		if len(paths) == 0 {
			continue
		}

		storage := getStorageDriver(driver)
		if storage == nil {
			failed = append(failed, paths...)
			facade.Log.Error(map[string]any{"driver": driver, "count": len(paths)}, "存储驱动不可用，文件未删除")
			continue
		}

		// 本地文件逐个删，能精确统计每个文件的成败；云存储走批量接口（减少请求数）
		if driver == facade.StorageModeLocal || utils.Is.Empty(driver) {
			for _, path := range paths {
				if err := storage.Delete(path); err != nil {
					failed = append(failed, path)
					facade.Log.Error(map[string]any{"driver": driver, "path": path, "error": err.Error()}, "删除本地存储文件失败")
					continue
				}
				deleted[path] = true
			}
			continue
		}

		for start := 0; start < len(paths); start += batchSize {
			end := start + batchSize
			if end > len(paths) {
				end = len(paths)
			}
			batch := paths[start:end]

			if err := storage.DeleteMulti(batch); err != nil {
				failed = append(failed, batch...)
				facade.Log.Error(map[string]any{
					"driver": driver,
					"count":  len(batch),
					"first":  batch[0],
					"error":  err.Error(),
				}, "删除存储文件失败")
				continue
			}

			for _, path := range batch {
				deleted[path] = true
			}
		}
	}

	return
}

func (this *Attachment) remove(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	query := facade.DB.Model(&model.Attachment{}).WhereIn("id", ids)
	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	validIds := utils.Unity.Ids(query.Column("id"))
	validIdSet := make(map[any]bool)
	for _, id := range validIds {
		validIdSet[id] = true
	}

	var successIds []any
	var failedIds []any
	var errors = make(map[string]string)

	for _, id := range ids {
		if validIdSet[id] {
			successIds = append(successIds, id)
		} else {
			failedIds = append(failedIds, id)
			errors[cast.ToString(id)] = "无权删除该附件或附件不存在"
		}
	}

	if len(successIds) == 0 {
		this.json(ctx, gin.H{"success_ids": []any{}, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	_, err := facade.DB.Model(&model.Attachment{}).Delete(successIds)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	facade.Log.Info(map[string]any{"user_id": this.meta.user(ctx).Id, "ids": successIds}, "软删除附件")

	if len(failedIds) == 0 {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": []any{}, "errors": map[string]string{}}, facade.Lang(ctx, "删除成功！"), 200)
	} else {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "部分删除成功！"), 207)
	}
}

func (this *Attachment) delete(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	// 注意：必须用「切片模型」&[]model.Attachment{} 查询。
	// 传单模型（&model.Attachment{}）时 Select() 只会得到单条对象，经 JSON 转换后
	// cast.ToSlice 结果为空 —— save_path 一个都取不到，文件自然不会被删除，
	// 表现为「只删了数据库记录、存储桶里的文件还在」。
	items, _ := facade.DB.Model(&[]model.Attachment{}).WithTrashed().WhereIn("id", ids).Select()

	// 记录是否存在、每个记录对应的存储文件（Select 出来的 id 是 float64，统一转 int 比对）
	exists := make(map[int]bool)
	recordPaths := make(map[int][]string)
	filesByDriver := make(map[string][]string)

	for _, item := range items {
		id := cast.ToInt(item["id"])
		exists[id] = true

		savePath := cast.ToString(item["save_path"])
		if utils.Is.Empty(savePath) {
			continue
		}

		// 老数据可能没记录驱动：按当前默认驱动删，避免删到错误的存储
		driver := cast.ToString(item["storage_driver"])
		if utils.Is.Empty(driver) {
			driver = cast.ToString(facade.StorageToml.Get("default"))
		}

		filesByDriver[driver] = append(filesByDriver[driver], savePath)
		recordPaths[id] = append(recordPaths[id], savePath)
	}

	// 先删物理文件：文件没删成功的记录保留（否则记录没了、文件变孤儿，只能靠日志排查）
	deleted, failedFiles := this.purgeFiles(filesByDriver)

	var successIds []any
	var failedIds []any
	errors := make(map[string]string)

	for _, id := range ids {
		uid := cast.ToInt(id)

		if !exists[uid] {
			failedIds = append(failedIds, id)
			errors[cast.ToString(id)] = "附件不存在"
			continue
		}

		if !pathsAllDeleted(recordPaths[uid], deleted) {
			failedIds = append(failedIds, id)
			errors[cast.ToString(id)] = "存储文件删除失败，记录已保留（详见运行日志）"
			continue
		}

		successIds = append(successIds, id)
	}

	if len(successIds) == 0 {
		this.json(ctx, gin.H{
			"success_ids":  []any{},
			"failed_ids":   failedIds,
			"errors":       errors,
			"file_deleted": len(deleted),
			"file_failed":  len(failedFiles),
		}, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	_, err := facade.DB.Model(&model.Attachment{}).WithTrashed().Force().Delete(successIds)

	if err != nil {
		// 文件已删但记录还在：属异常情况，单独记日志便于人工核对
		facade.Log.Error(map[string]any{"error": err.Error(), "ids": successIds}, "物理文件已删除但附件记录删除失败")
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	facade.Log.Info(map[string]any{
		"user_id":      this.meta.user(ctx).Id,
		"ids":          successIds,
		"file_deleted": len(deleted),
		"file_failed":  len(failedFiles),
	}, "物理删除附件")

	if len(failedIds) == 0 {
		this.json(ctx, gin.H{
			"success_ids":  successIds,
			"failed_ids":   []any{},
			"errors":       map[string]string{},
			"file_deleted": len(deleted),
			"file_failed":  0,
		}, facade.Lang(ctx, "删除成功！"), 200)
		return
	}

	this.json(ctx, gin.H{
		"success_ids":  successIds,
		"failed_ids":   failedIds,
		"errors":       errors,
		"file_deleted": len(deleted),
		"file_failed":  len(failedFiles),
	}, facade.Lang(ctx, "部分删除成功！"), 207)
}

// pathsAllDeleted 判断某个记录的文件是否都已从存储删除（没有文件时视为成功）
func pathsAllDeleted(paths []string, deleted map[string]bool) bool {
	for _, path := range paths {
		if !deleted[path] {
			return false
		}
	}
	return true
}

func (this *Attachment) clear(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	columnData, _ := facade.DB.Model(&model.Attachment{}).OnlyTrashed().Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	// 必须用切片模型查询：单模型的 Select 拿不到多条记录，save_path 会全部丢失，
	// 结果就是「清空回收站只删了数据库记录、存储桶里的文件一个没动」
	items, _ := facade.DB.Model(&[]model.Attachment{}).OnlyTrashed().WhereIn("id", ids).Select()

	recordPaths := make(map[int][]string)
	filesByDriver := make(map[string][]string)

	for _, item := range items {
		id := cast.ToInt(item["id"])
		savePath := cast.ToString(item["save_path"])
		if utils.Is.Empty(savePath) {
			continue
		}

		driver := cast.ToString(item["storage_driver"])
		if utils.Is.Empty(driver) {
			driver = cast.ToString(facade.StorageToml.Get("default"))
		}

		filesByDriver[driver] = append(filesByDriver[driver], savePath)
		recordPaths[id] = append(recordPaths[id], savePath)
	}

	// 先删物理文件：文件没删成功的记录保留在回收站（可修复存储配置后重试，不产生孤儿文件）
	deleted, failedFiles := this.purgeFiles(filesByDriver)

	var okIds []any
	var keptIds []any
	fileErrors := make(map[string]string)

	for _, item := range items {
		id := cast.ToInt(item["id"])
		if pathsAllDeleted(recordPaths[id], deleted) {
			okIds = append(okIds, id)
			continue
		}
		keptIds = append(keptIds, id)
		fileErrors[cast.ToString(id)] = "存储文件删除失败，记录已保留（详见运行日志）"
	}

	if utils.Is.Empty(okIds) {
		facade.Log.Error(map[string]any{
			"user_id":     this.meta.user(ctx).Id,
			"file_failed": len(failedFiles),
		}, "清空回收站失败：所有文件都未能从存储删除")
		this.json(ctx, gin.H{
			"ids":          []any{},
			"kept_ids":     keptIds,
			"errors":       fileErrors,
			"file_deleted": len(deleted),
			"file_failed":  len(failedFiles),
		}, facade.Lang(ctx, "清空失败：存储文件删除失败，记录已保留！"), 400)
		return
	}

	_, err := facade.DB.Model(&model.Attachment{}).OnlyTrashed().WhereIn("id", okIds).Force().Delete()

	if err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "ids": okIds}, "物理文件已删除但清空附件记录失败")
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	facade.Log.Info(map[string]any{
		"user_id":      this.meta.user(ctx).Id,
		"ids":          okIds,
		"file_deleted": len(deleted),
		"file_failed":  len(failedFiles),
	}, "清空回收站附件")

	data := gin.H{
		"ids":          okIds,
		"kept_ids":     keptIds,
		"errors":       fileErrors,
		"file_deleted": len(deleted),
		"file_failed":  len(failedFiles),
	}

	if len(keptIds) > 0 {
		this.json(ctx, data, facade.Lang(ctx, "部分清空成功：部分记录的文件删除失败，已保留！"), 207)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "清空成功！"), 200)
}

func (this *Attachment) restore(ctx *gin.Context) {
	params := this.params(ctx)
	ids := utils.Unity.Ids(params["ids"])

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids"), 400)
		return
	}

	item := facade.DB.Model(&model.Attachment{}).OnlyTrashed().WhereIn("id", ids)
	if !this.meta.root(ctx) {
		item = item.Where("uploader_id", this.meta.user(ctx).Id)
	}

	columnData, _ := item.Column("id")
	validIds := utils.Unity.Ids(columnData)
	validIdSet := make(map[any]bool)
	for _, id := range validIds {
		validIdSet[id] = true
	}

	var successIds []any
	var failedIds []any
	var errors = make(map[string]string)

	for _, id := range ids {
		if validIdSet[id] {
			successIds = append(successIds, id)
		} else {
			failedIds = append(failedIds, id)
			errors[cast.ToString(id)] = "无权恢复该附件或附件不存在"
		}
	}

	if len(successIds) == 0 {
		this.json(ctx, gin.H{"success_ids": []any{}, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	_, err := facade.DB.Model(&model.Attachment{}).OnlyTrashed().Restore(successIds)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	if len(failedIds) == 0 {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": []any{}, "errors": map[string]string{}}, facade.Lang(ctx, "恢复成功！"), 200)
	} else {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "部分恢复成功！"), 207)
	}
}

// emoji - 获取表情列表（自动扫描 public/assets/emoji 目录及子目录）
func (this *Attachment) emoji(ctx *gin.Context) {
	emojiDir := filepath.Join("public", "assets", "emoji")

	// 支持指定分类，例如 ?category=qq
	filterCategory := strings.TrimSpace(cast.ToString(this.params(ctx)["category"]))

	categories, err := this.scanEmojiDir(emojiDir, filterCategory)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "读取表情失败: %v", err.Error()), 500)
		return
	}

	total := 0
	for _, c := range categories {
		if count, ok := c["count"].(int); ok {
			total += count
		}
	}

	this.json(ctx, gin.H{
		"categories": categories,
		"total":      total,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// scanEmojiDir - 扫描表情目录，返回各分类及其表情文件
func (this *Attachment) scanEmojiDir(rootDir, filterCategory string) ([]facade.H, error) {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	// 支持的表情图片扩展名
	allowExt := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true, ".bmp": true,
	}

	var categories []facade.H
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		categoryName := entry.Name()
		// 指定了分类时，仅返回该分类
		if filterCategory != "" && categoryName != filterCategory {
			continue
		}

		subDir := filepath.Join(rootDir, categoryName)
		files, err := this.scanEmojiFiles(subDir, categoryName, allowExt)
		if err != nil {
			continue
		}

		categories = append(categories, facade.H{
			"name":  categoryName,
			"count": len(files),
			"items": files,
		})
	}

	return categories, nil
}

// scanEmojiFiles - 扫描指定分类目录内的表情文件
func (this *Attachment) scanEmojiFiles(dir, categoryName string, allowExt map[string]bool) ([]facade.H, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var items []facade.H
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		ext := strings.ToLower(filepath.Ext(fileName))
		if !allowExt[ext] {
			continue
		}

		// 文件名（不含扩展名）作为表情名称，下划线/中划线转为可读形式
		name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		// URL 路径（public 目录映射到站点根）
		url := "/assets/emoji/" + categoryName + "/" + fileName

		info, err := entry.Info()
		size := int64(0)
		if err == nil {
			size = info.Size()
		}

		items = append(items, facade.H{
			"name": name,
			"file": fileName,
			"url":  url,
			"ext":  ext,
			"size": size,
		})
	}

	return items, nil
}
