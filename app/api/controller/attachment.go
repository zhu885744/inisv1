package controller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"io"
	"math"
	"mime/multipart"
	"net/http"
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
	reg := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	fileName = reg.ReplaceAllString(fileName, "_")
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

func (this *Attachment) verifyFileContent(headerBytes []byte, fileExt string) bool {
	if len(headerBytes) == 0 {
		return true
	}

	switch fileExt {
	case "jpg", "jpeg":
		return len(headerBytes) >= 2 && headerBytes[0] == 0xFF && headerBytes[1] == 0xD8
	case "png":
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
	case "txt":
		return true
	case "md":
		return true
	case "svg":
		return len(headerBytes) >= 5 &&
			(headerBytes[0] == '<' && headerBytes[1] == 's' && headerBytes[2] == 'v' && headerBytes[3] == 'g') ||
			(headerBytes[0] == '<' && headerBytes[1] == '?' && headerBytes[2] == 'x' && headerBytes[3] == 'm')
	default:
		return false
	}
}

func (this *Attachment) sanitizeSVG(content string) string {
	content = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)<script[^>]*\/?>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<!\[CDATA\[.*?\]\]>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)on\w+\s*=\s*["'][^"']*["']`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)on\w+\s*=\s*[^\s>]+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)<foreignObject[^>]*>.*?</foreignObject>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)<foreignObject[^>]*\/?>`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)vbscript:`).ReplaceAllString(content, "")
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

	if facade.Redis != nil && facade.Redis.Client != nil {
		ctx := context.Background()
		key := "inis:attachment:upload_concurrent_counter"
		current, err := facade.Redis.Client.IncrBy(ctx, key, int64(count)).Result()
		if err != nil {
			facade.Log.Error(map[string]any{"error": err, "key": key}, "Redis并发计数器操作失败，拒绝上传")
			return false
		}
		if current > int64(config.ConcurrentLimit) {
			facade.Redis.Client.DecrBy(ctx, key, int64(count))
			return false
		}
		if current == int64(count) {
			facade.Redis.Client.Expire(ctx, key, 5*time.Minute)
		}
		return true
	}

	facade.Log.Error(map[string]any{}, "Redis不可用，拒绝上传")
	return false
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
	if facade.Redis != nil && facade.Redis.Client != nil {
		ctx := context.Background()
		key := "inis:attachment:upload_concurrent_counter"
		facade.Redis.Client.DecrBy(ctx, key, int64(count))
		return
	}
}

func (this *Attachment) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"one":    this.one,
		"all":    this.all,
		"sum":    this.sum,
		"min":    this.min,
		"max":    this.max,
		"rand":   this.rand,
		"count":  this.count,
		"column": this.column,
		"list":   this.list,
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
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item, _ := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
		data = utils.ArrayMapWithField(item, params["field"])
		this.setCache(ctx, cacheName, data)
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

	query = this.buildQuery(query, params).Order("RAND()").Limit(limit)

	items, _ := query.Select()
	data := utils.Array.MapWithField(utils.Rand.MapSlice(items), params["field"])

	if utils.Is.Empty(data) {
		this.json(ctx, nil, facade.Lang(ctx, "无数据！"), 204)
		return
	}

	this.json(ctx, data, facade.Lang(ctx, "好的！"), 200)
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
		result.Error = fmt.Errorf("文件内容与扩展名不匹配！")
		return result
	}

	mimeType := http.DetectContentType(headerBytes)

	var uploadReader io.Reader
	var fileContent []byte

	if fileExt == "svg" {
		fileContent, err = io.ReadAll(file)
		if err != nil {
			result.Error = fmt.Errorf("读取文件失败")
			return result
		}
		sanitizedContent := this.sanitizeSVG(string(fileContent))
		uploadReader = strings.NewReader(sanitizedContent)
	} else {
		if seeker, ok := file.(io.Seeker); ok {
			if _, err := seeker.Seek(0, io.SeekStart); err != nil {
				result.Error = fmt.Errorf("重置文件指针失败")
				return result
			}
		}
		uploadReader = file
	}

	saveName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano()/1e6, utils.Rand.Int(1000, 9999), suffix)
	hash := sha256.New()
	hashReader := io.TeeReader(uploadReader, hash)
	item := facade.Storage.Upload(facade.Storage.Path()+suffix, hashReader)
	if item.Error != nil {
		result.Error = fmt.Errorf("上传文件失败")
		return result
	}
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
		return result
	}

	fullUrl := item.Domain + item.Path
	attachment := model.Attachment{
		Uuid: (&model.Attachment{}).GenerateUUID(), OriginalName: fileName, SaveName: saveName,
		SavePath: item.Path, FullUrl: fullUrl, FileSize: fileHeader.Size,
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

	result.Attachment = &attachment
	result.IsSuccess = true
	return result
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
			results = append(results, map[string]any{
				"original_name": result.Existing["original_name"],
				"full_url":      utils.Replace(cast.ToString(result.Existing["full_url"]), model.DomainTemp1()),
				"status":        "exist",
			})
			successCount++
			continue
		}

		if result.IsSuccess && result.Attachment != nil {
			results = append(results, map[string]any{
				"id": result.Attachment.Id, "uuid": result.Attachment.Uuid,
				"original_name": result.Attachment.OriginalName,
				"full_url":      utils.Replace(result.Attachment.FullUrl, model.DomainTemp1()),
				"file_size":     result.Attachment.FileSize,
				"status":        "success",
			})
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

func getStorageDriver(driver string) facade.StorageInterface {
	switch driver {
	case "oss":
		return facade.OSS
	case "cos":
		return facade.COS
	case "kodo":
		return facade.KODO
	default:
		return facade.LocalStorage
	}
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

	items, _ := facade.DB.Model(&model.Attachment{}).WithTrashed().WhereIn("id", ids).Select()

	validIdSet := make(map[any]bool)
	for _, item := range items {
		validIdSet[item["id"]] = true
	}

	var successIds []any
	var failedIds []any
	var errors = make(map[string]string)

	for _, id := range ids {
		if validIdSet[id] {
			successIds = append(successIds, id)
		} else {
			failedIds = append(failedIds, id)
			errors[cast.ToString(id)] = "附件不存在"
		}
	}

	if len(successIds) == 0 {
		this.json(ctx, gin.H{"success_ids": []any{}, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	filesByDriver := make(map[string][]string)
	for _, item := range items {
		savePath := cast.ToString(item["save_path"])
		storageDriver := cast.ToString(item["storage_driver"])
		if savePath != "" {
			filesByDriver[storageDriver] = append(filesByDriver[storageDriver], savePath)
		}
	}

	go func(files map[string][]string) {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "物理删除存储文件异常")
			}
		}()
		for driver, paths := range files {
			storage := getStorageDriver(driver)
			if storage == nil {
				facade.Log.Error(map[string]any{"driver": driver, "count": len(paths)}, "存储驱动未初始化")
				continue
			}
			if err := storage.DeleteMulti(paths); err != nil {
				facade.Log.Error(map[string]any{"error": err, "driver": driver, "count": len(paths)}, "批量删除存储文件失败")
			}
		}
	}(filesByDriver)

	_, err := facade.DB.Model(&model.Attachment{}).WithTrashed().Force().Delete(successIds)

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	facade.Log.Info(map[string]any{"user_id": this.meta.user(ctx).Id, "ids": successIds}, "物理删除附件")

	if len(failedIds) == 0 {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": []any{}, "errors": map[string]string{}}, facade.Lang(ctx, "删除成功！"), 200)
	} else {
		this.json(ctx, gin.H{"success_ids": successIds, "failed_ids": failedIds, "errors": errors}, facade.Lang(ctx, "部分删除成功！"), 207)
	}
}

func (this *Attachment) clear(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	item := facade.DB.Model(&model.Attachment{}).OnlyTrashed()
	columnData, _ := item.Column("id")
	ids := utils.Unity.Ids(columnData)

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	items, _ := facade.DB.Model(&model.Attachment{}).OnlyTrashed().WhereIn("id", ids).Select()

	filesByDriver := make(map[string][]string)
	for _, item := range items {
		savePath := cast.ToString(item["save_path"])
		storageDriver := cast.ToString(item["storage_driver"])
		if savePath != "" {
			filesByDriver[storageDriver] = append(filesByDriver[storageDriver], savePath)
		}
	}

	go func(files map[string][]string) {
		defer func() {
			if r := recover(); r != nil {
				facade.Log.Error(map[string]any{"error": r}, "清空回收站存储文件异常")
			}
		}()
		for driver, paths := range files {
			storage := getStorageDriver(driver)
			if storage == nil {
				facade.Log.Error(map[string]any{"driver": driver, "count": len(paths)}, "存储驱动未初始化")
				continue
			}
			if err := storage.DeleteMulti(paths); err != nil {
				facade.Log.Error(map[string]any{"error": err, "driver": driver, "count": len(paths)}, "批量清空回收站存储文件失败")
			}
		}
	}(filesByDriver)

	_, err := item.Force().Delete()

	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "清空失败！"), 400)
		return
	}

	facade.Log.Info(map[string]any{"user_id": this.meta.user(ctx).Id, "ids": ids}, "清空回收站附件")
	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "清空成功！"), 200)
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
