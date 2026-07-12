package controller

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

var allowedAttachmentExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".bmp": true, ".webp": true, ".svg": true, ".pdf": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".txt": true, ".zip": true,
	".rar": true, ".7z": true, ".mp3": true, ".mp4": true,
	".wav": true, ".avi": true, ".mov": true,
}

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true, ".webp": true,
}

var mimeWhitelist = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true,
	"image/bmp": true, "image/webp": true, "image/svg+xml": true,
	"application/pdf": true, "application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/vnd.ms-excel": true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain": true, "application/zip": true, "application/x-rar-compressed": true,
	"application/x-7z-compressed": true, "audio/mpeg": true, "video/mp4": true,
	"audio/wav": true, "video/x-msvideo": true, "video/quicktime": true,
}

type Attachment struct{ base }

const attachmentMaxFileSize = 50 * 1024 * 1024

var attachmentAllowFieldsSlice = []any{"original_name", "is_public", "target_type", "target_id", "status"}
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
		go facade.Cache.Set(cacheName, data)
	}
}

func (this *Attachment) validateFile(fileBytes []byte, suffix string) bool {
	if !allowedAttachmentExtensions[suffix] {
		return false
	}
	mimeType := http.DetectContentType(fileBytes)
	return mimeWhitelist[mimeType]
}

func (this *Attachment) getImageSize(fileBytes []byte, suffix string) (int, int) {
	if !imageExtensions[suffix] {
		return 0, 0
	}
	img, err := imaging.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return 0, 0
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
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
		"save":    this.save,
		"create":  this.create,
		"upload":  this.upload,
		"batch":   this.batch,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
		return
	}
	go this.delCache()
}

func (this *Attachment) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		"update":  this.update,
		"restore": this.restore,
		"bind":    this.bind,
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

		item := query.Where(table).Find()
		if !utils.Is.Empty(item) {
			if cast.ToInt(item["status"]) == 0 && !this.meta.root(ctx) {
				this.json(ctx, nil, facade.Lang(ctx, "附件已禁用！"), 403)
				return
			}
			if !cast.ToBool(item["is_public"]) && !this.meta.root(ctx) {
				if cast.ToInt(item["uploader_id"]) != this.meta.user(ctx).Id {
					this.json(ctx, nil, facade.Lang(ctx, "无权限访问！"), 403)
					return
				}
			}
		}
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

	query := this.withTrashOptions(facade.DB.Model(&[]model.Attachment{}), params)
	query = this.buildQuery(query, params)

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	count := query.Where(table).Count()

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		item := query.Where(table).Limit(limit).Page(page).Order(params["order"]).Select()
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
	except := utils.Unity.Ids(params["except"])
	onlyTrashed := cast.ToBool(params["onlyTrashed"])
	withTrashed := cast.ToBool(params["withTrashed"])

	query := facade.DB.Model(&model.Attachment{}).OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	if !utils.Is.Empty(except) {
		query = query.Where("id", "NOT IN", except)
	}

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id).Where("status", 1)
	}

	ids := utils.Rand.Slice(utils.Unity.Ids(query.Column("id")), limit)

	mold := facade.DB.Model(&[]model.Attachment{}).Where("id", "IN", ids)
	mold.OnlyTrashed(onlyTrashed).WithTrashed(withTrashed)
	mold = this.buildQuery(mold, params)

	data := utils.Array.MapWithField(utils.Rand.MapSlice(mold.Select()), params["field"])

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

	this.json(ctx, query.Count(), facade.Lang(ctx, "查询成功！"), 200)
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
		return query.Sum(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Attachment) min(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Min(field)
	})
	if data == nil && msg == "" {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "field"), 400)
		return
	}
	this.json(ctx, data, msg, 200)
}

func (this *Attachment) max(ctx *gin.Context) {
	data, msg := this.aggregateQuery(ctx, func(query *facade.ModelStruct, field string) any {
		return query.Max(field)
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
	query := this.withTrashOptions(facade.DB.Model(&[]model.Attachment{}), params)
	query = this.buildQuery(query, params).Order(params["order"])

	if !this.meta.root(ctx) {
		query = query.Where("uploader_id", this.meta.user(ctx).Id)
	}

	ids := utils.Unity.Keys(params["ids"])
	if !utils.Is.Empty(ids) {
		query = query.WhereIn("id", ids)
	}

	cacheName := this.cache.name(ctx)
	if cached, ok := this.getFromCache(ctx, cacheName); ok {
		msg[1] = "（来自缓存）"
		data = cached
	} else {
		data = utils.ArrayMapWithField(query.Select(), params["field"])
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
		Uuid:          (&model.Attachment{}).GenerateUUID(),
		UploaderId:    uint(userId),
		Status:        1,
		IsPublic:      true,
	}

	allowFields := append([]any{}, attachmentAllowFieldsSlice...)

	for key, val := range params {
		if utils.In.Array(key, allowFields) {
			utils.Struct.Set(&table, key, val)
		}
	}

	tx := facade.DB.Model(&table).Create(&table)

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": table.Id, "uuid": table.Uuid}, facade.Lang(ctx, "创建成功！"), 200)
}

func (this *Attachment) upload(ctx *gin.Context) {
	userId := this.meta.user(ctx).Id
	if userId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取文件失败：%v", err.Error()), 400)
		return
	}
	if fileHeader.Size > attachmentMaxFileSize {
		this.json(ctx, nil, facade.Lang(ctx, "文件大小超过限制（最大50MB）"), 400)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "打开文件失败：%v", err.Error()), 400)
		return
	}
	defer file.Close()
	fileBytes := make([]byte, fileHeader.Size)
	if _, err := file.Read(fileBytes); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "读取文件失败：%v", err.Error()), 400)
		return
	}
	fileName := strings.TrimSpace(fileHeader.Filename)
	fileName = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fileName, "..", ""), "/", ""), "\\", "")
	suffix := ""
	if lastIndex := strings.LastIndex(fileName, "."); lastIndex > 0 {
		suffix = strings.ToLower(fileName[lastIndex:])
	}
	if !this.validateFile(fileBytes, suffix) {
		this.json(ctx, nil, facade.Lang(ctx, "不允许上传该类型的文件！"), 400)
		return
	}
	fileHash := fmt.Sprintf("%x", md5.Sum(fileBytes))
	existing := (&model.Attachment{}).GetByHash(fileHash)
	if !utils.Is.Empty(existing) {
		this.json(ctx, map[string]any{
			"id": existing["id"], "uuid": existing["uuid"], "original_name": existing["original_name"],
			"full_url": existing["full_url"], "file_size": existing["file_size"],
			"mime_type": existing["mime_type"], "file_ext": existing["file_ext"],
		}, facade.Lang(ctx, "文件已存在（秒传）！"), 200)
		return
	}
	saveName := fmt.Sprintf("%d%s", time.Now().UnixNano()/1e6, suffix)
	item := facade.Storage.Upload(facade.Storage.Path()+suffix, bytes.NewReader(fileBytes))
	if item.Error != nil {
		this.json(ctx, nil, item.Error.Error(), 400)
		return
	}
	fullUrl := item.Domain + item.Path
	width, height := this.getImageSize(fileBytes, suffix)
	params := this.params(ctx)
	attachment := model.Attachment{
		Uuid: (&model.Attachment{}).GenerateUUID(), OriginalName: fileName, SaveName: saveName,
		SavePath: item.Path, FullUrl: fullUrl, FileSize: fileHeader.Size,
		MimeType: http.DetectContentType(fileBytes), FileExt: strings.TrimPrefix(suffix, "."),
		StorageDriver: cast.ToString(facade.StorageToml.Get("default")), UploaderId: uint(userId),
		TargetType: cast.ToString(params["target_type"]), TargetId: cast.ToUint(params["target_id"]),
		IsPublic: cast.ToBool(params["is_public"]), FileHash: fileHash, Status: 1,
		Width: width, Height: height,
	}
	tx := facade.DB.Model(&attachment).Create(&attachment)
	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}
	facade.Log.Info(map[string]any{"user_id": userId, "file_name": fileName, "file_size": fileHeader.Size, "storage_driver": attachment.StorageDriver}, "附件上传成功")
	this.json(ctx, map[string]any{
		"id": attachment.Id, "uuid": attachment.Uuid, "original_name": attachment.OriginalName,
		"full_url": attachment.FullUrl, "file_size": attachment.FileSize,
		"mime_type": attachment.MimeType, "file_ext": attachment.FileExt,
		"width": width, "height": height,
	}, facade.Lang(ctx, "上传成功！"), 200)
}

func (this *Attachment) batch(ctx *gin.Context) {
	userId := this.meta.user(ctx).Id
	if userId == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	form, err := ctx.MultipartForm()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "获取表单失败：%v", err.Error()), 400)
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请选择要上传的文件！"), 400)
		return
	}
	if len(files) > 10 {
		this.json(ctx, nil, facade.Lang(ctx, "单次最多上传10个文件！"), 400)
		return
	}
	var results []map[string]any
	var successCount, failCount int
	params := this.params(ctx)
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			failCount++
			continue
		}
		fileBytes := make([]byte, fileHeader.Size)
		if _, err := file.Read(fileBytes); err != nil {
			file.Close()
			failCount++
			continue
		}
		file.Close()
		if fileHeader.Size > attachmentMaxFileSize {
			failCount++
			continue
		}
		fileName := strings.TrimSpace(fileHeader.Filename)
		fileName = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fileName, "..", ""), "/", ""), "\\", "")
		suffix := ""
		if lastIndex := strings.LastIndex(fileName, "."); lastIndex > 0 {
			suffix = strings.ToLower(fileName[lastIndex:])
		}
		if !this.validateFile(fileBytes, suffix) {
			failCount++
			continue
		}
		fileHash := fmt.Sprintf("%x", md5.Sum(fileBytes))
		existing := (&model.Attachment{}).GetByHash(fileHash)
		if !utils.Is.Empty(existing) {
			results = append(results, map[string]any{"original_name": existing["original_name"], "full_url": existing["full_url"], "status": "exist"})
			successCount++
			continue
		}
		saveName := fmt.Sprintf("%d%s", time.Now().UnixNano()/1e6, suffix)
		item := facade.Storage.Upload(facade.Storage.Path()+suffix, bytes.NewReader(fileBytes))
		if item.Error != nil {
			failCount++
			continue
		}
		fullUrl := item.Domain + item.Path
		width, height := this.getImageSize(fileBytes, suffix)
		attachment := model.Attachment{
			Uuid: (&model.Attachment{}).GenerateUUID(), OriginalName: fileName, SaveName: saveName,
			SavePath: item.Path, FullUrl: fullUrl, FileSize: fileHeader.Size,
			MimeType: http.DetectContentType(fileBytes), FileExt: strings.TrimPrefix(suffix, "."),
			StorageDriver: cast.ToString(facade.StorageToml.Get("default")), UploaderId: uint(userId),
			TargetType: cast.ToString(params["target_type"]), TargetId: cast.ToUint(params["target_id"]),
			IsPublic: cast.ToBool(params["is_public"]), FileHash: fileHash, Status: 1,
			Width: width, Height: height,
		}
		tx := facade.DB.Model(&attachment).Create(&attachment)
		if tx.Error != nil {
			failCount++
			continue
		}
		results = append(results, map[string]any{"id": attachment.Id, "uuid": attachment.Uuid, "original_name": attachment.OriginalName, "full_url": attachment.FullUrl, "file_size": attachment.FileSize, "width": width, "height": height, "status": "success"})
		successCount++
	}
	facade.Log.Info(map[string]any{"user_id": userId, "success_count": successCount, "fail_count": failCount}, "批量上传附件")
	this.json(ctx, gin.H{"results": results, "success": successCount, "fail": failCount}, facade.Lang(ctx, "批量上传完成！"), 200)
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

	item := query.Find()
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

	tx := query.Update(async.Result())

	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}

	this.json(ctx, gin.H{"id": item["id"], "uuid": item["uuid"]}, facade.Lang(ctx, "更新成功！"), 200)
}

func (this *Attachment) bind(ctx *gin.Context) {
	params := this.params(ctx)
	if utils.Is.Empty(params["ids"]) && utils.Is.Empty(params["uuids"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "ids/uuids"), 400)
		return
	}
	if utils.Is.Empty(params["target_type"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_type"), 400)
		return
	}
	if utils.Is.Empty(params["target_id"]) {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_id"), 400)
		return
	}

	var ids []any
	if !utils.Is.Empty(params["uuids"]) {
		ids = utils.Unity.Keys(params["uuids"])
	} else {
		ids = utils.Unity.Ids(params["ids"])
	}

	if !this.meta.root(ctx) {
		for _, id := range ids {
			var attachment map[string]any
			if _, ok := params["uuids"]; ok {
				attachment = (&model.Attachment{}).GetByUUID(cast.ToString(id))
			} else {
				attachment = facade.DB.Model(&model.Attachment{}).Where("id", id).Find()
			}
			if !utils.Is.Empty(attachment) && cast.ToInt(attachment["uploader_id"]) != this.meta.user(ctx).Id {
				this.json(ctx, nil, facade.Lang(ctx, "无权限操作！"), 403)
				return
			}
		}
	}

	var query *facade.ModelStruct
	if _, ok := params["uuids"]; ok {
		query = facade.DB.Model(&model.Attachment{}).WhereIn("uuid", ids)
	} else {
		query = facade.DB.Model(&model.Attachment{}).WhereIn("id", ids)
	}

	tx := query.Update(map[string]any{"target_type": params["target_type"], "target_id": cast.ToUint(params["target_id"])})
	if tx.Error != nil {
		this.json(ctx, nil, tx.Error.Error(), 400)
		return
	}
	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "绑定成功！"), 200)
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

	item := facade.DB.Model(&model.Attachment{})
	if !this.meta.root(ctx) {
		item = item.Where("uploader_id", this.meta.user(ctx).Id)
	}

	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.Attachment{}).Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	facade.Log.Info(map[string]any{"user_id": this.meta.user(ctx).Id, "ids": ids}, "软删除附件")
	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
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

	item := facade.DB.Model(&model.Attachment{}).WithTrashed()
	ids = utils.Unity.Ids(item.WhereIn("id", ids).Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	items := facade.DB.Model(&model.Attachment{}).WithTrashed().WhereIn("id", ids).Select()
	for _, item := range items {
		savePath := cast.ToString(item["save_path"])
		storageDriver := cast.ToString(item["storage_driver"])
		if savePath != "" {
			go func(path, driver string) {
				defer func() {
					if r := recover(); r != nil {
						facade.Log.Error(map[string]any{"error": r, "path": path, "driver": driver}, "物理删除存储文件异常")
					}
				}()
				storage := getStorageDriver(driver)
				if storage == nil {
					facade.Log.Error(map[string]any{"path": path, "driver": driver}, "存储驱动未初始化")
					return
				}
				err := storage.Delete(path)
				if err != nil {
					facade.Log.Error(map[string]any{"error": err, "path": path, "driver": driver}, "物理删除存储文件失败")
				}
			}(savePath, storageDriver)
		}
	}

	tx := facade.DB.Model(&model.Attachment{}).WithTrashed().Force().Delete(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "删除失败！"), 400)
		return
	}

	facade.Log.Info(map[string]any{"user_id": this.meta.user(ctx).Id, "ids": ids}, "物理删除附件")
	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "删除成功！"), 200)
}

func (this *Attachment) clear(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	item := facade.DB.Model(&model.Attachment{}).OnlyTrashed()
	ids := utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	items := facade.DB.Model(&model.Attachment{}).OnlyTrashed().WhereIn("id", ids).Select()
	for _, item := range items {
		savePath := cast.ToString(item["save_path"])
		storageDriver := cast.ToString(item["storage_driver"])
		if savePath != "" {
			go func(path, driver string) {
				defer func() {
					if r := recover(); r != nil {
						facade.Log.Error(map[string]any{"error": r, "path": path, "driver": driver}, "清空回收站存储文件异常")
					}
				}()
				storage := getStorageDriver(driver)
				if storage == nil {
					facade.Log.Error(map[string]any{"path": path, "driver": driver}, "存储驱动未初始化")
					return
				}
				err := storage.Delete(path)
				if err != nil {
					facade.Log.Error(map[string]any{"error": err, "path": path, "driver": driver}, "清空回收站存储文件失败")
				}
			}(savePath, storageDriver)
		}
	}

	tx := item.Force().Delete()

	if tx.Error != nil {
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

	ids = utils.Unity.Ids(item.Column("id"))

	if utils.Is.Empty(ids) {
		this.json(ctx, nil, facade.Lang(ctx, "无可操作数据！"), 204)
		return
	}

	tx := facade.DB.Model(&model.Attachment{}).OnlyTrashed().Restore(ids)

	if tx.Error != nil {
		this.json(ctx, nil, facade.Lang(ctx, "恢复失败！"), 400)
		return
	}

	this.json(ctx, gin.H{"ids": ids}, facade.Lang(ctx, "恢复成功！"), 200)
}