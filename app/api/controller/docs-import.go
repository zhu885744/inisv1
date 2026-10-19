package controller

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"inis/app/facade"
	"inis/app/model"
)

// 导入限制
const (
	docsImportMaxSize     = 32 << 20 // 上传文件最大 32MB
	docsImportMaxFiles    = 800      // 压缩包内最多处理 800 篇
	docsImportMaxFileSize = 5 << 20  // 单篇 Markdown 最大 5MB
)

// docsImportResult - 导入结果
type docsImportResult struct {
	Groups  int      `json:"groups"`
	Created int      `json:"created"`
	Updated int      `json:"updated"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// importDocs - 导入 Markdown（单个 .md 或 .zip 压缩包）
//
// 参数（multipart/form-data）：
//   - file      必填，.md / .markdown / .zip
//   - version   可选，默认 latest
//   - gid       可选，导入到指定目录（压缩包内的子目录会在这个目录下继续建子目录）
//   - overwrite 可选，1 = 同 slug 覆盖更新，0 = 跳过（默认 1）
//
// 压缩包内的目录结构会映射成文档目录；front-matter 支持 title / slug / sort / status。
func (this *Docs) importDocs(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "请选择要导入的文件！"), 400)
		return
	}

	if file.Size <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "文件内容为空！"), 400)
		return
	}

	if file.Size > docsImportMaxSize {
		this.json(ctx, nil, facade.Lang(ctx, "文件不能超过 32MB！"), 400)
		return
	}

	version := strings.TrimSpace(ctx.PostForm("version"))
	if version == "" {
		version = "latest"
	}

	gid := cast.ToInt(ctx.PostForm("gid"))
	overwrite := true
	if raw := strings.TrimSpace(ctx.PostForm("overwrite")); raw != "" {
		overwrite = cast.ToBool(raw)
	}

	reader, err := file.Open()
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "读取文件失败！"), 400)
		return
	}
	defer reader.Close()

	name := strings.ToLower(file.Filename)
	result := &docsImportResult{Errors: make([]string, 0)}

	switch {
	case strings.HasSuffix(name, ".zip"):
		this.importZip(reader, file.Size, version, gid, overwrite, result)
	case strings.HasSuffix(name, ".md"), strings.HasSuffix(name, ".markdown"):
		content, readErr := io.ReadAll(io.LimitReader(reader, docsImportMaxFileSize))
		if readErr != nil {
			this.json(ctx, nil, facade.Lang(ctx, "读取文件失败！"), 400)
			return
		}
		meta, body := parseDocsFrontMatter(string(content))
		this.upsertImportedDoc(version, gid, file.Filename, body, meta, overwrite, result, 0)
	default:
		this.json(ctx, nil, facade.Lang(ctx, "只支持 .md 或 .zip 文件！"), 400)
		return
	}

	this.json(ctx, result, facade.Lang(ctx, "导入完成！"), 200)
}

// importZip - 解析压缩包：目录 → 文档目录，.md → 文档页面
func (this *Docs) importZip(reader io.ReaderAt, size int64, version string, gid int, overwrite bool, result *docsImportResult) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		result.Errors = append(result.Errors, "压缩包解析失败："+err.Error())
		return
	}

	entries := make([]docsZipEntry, 0, 32)

	for _, file := range archive.File {
		if file.FileInfo().IsDir() {
			continue
		}

		name := strings.ReplaceAll(file.Name, "\\", "/")
		lower := strings.ToLower(name)

		// 跳过 macOS 资源文件与隐藏文件
		if strings.HasPrefix(lower, "__macosx/") || strings.Contains(lower, "/.") || strings.HasPrefix(lower, ".") {
			continue
		}

		if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".markdown") {
			continue
		}

		if len(entries) >= docsImportMaxFiles {
			result.Errors = append(result.Errors, fmt.Sprintf("文档数量超过 %d 篇，剩余文件已忽略", docsImportMaxFiles))
			break
		}

		if file.UncompressedSize64 > docsImportMaxFileSize {
			result.Errors = append(result.Errors, "文件过大已跳过："+name)
			continue
		}

		rc, openErr := file.Open()
		if openErr != nil {
			result.Errors = append(result.Errors, "读取失败："+name)
			continue
		}

		content, readErr := io.ReadAll(io.LimitReader(rc, docsImportMaxFileSize))
		rc.Close()

		if readErr != nil {
			result.Errors = append(result.Errors, "读取失败："+name)
			continue
		}

		entries = append(entries, docsZipEntry{rel: name, content: content})
	}

	if len(entries) == 0 {
		result.Errors = append(result.Errors, "压缩包里没有找到 Markdown 文件")
		return
	}

	prefix := commonZipRoot(entries)
	groupCache := make(map[string]int)
	sort := 1

	for _, entry := range entries {
		rel := strings.TrimPrefix(entry.rel, prefix)
		rel = strings.TrimPrefix(rel, "/")

		if rel == "" || strings.Contains(rel, "..") {
			continue
		}

		segments := strings.Split(rel, "/")
		parent := gid

		// 除最后一段（文件名）外的每一段都对应一级目录
		for _, segment := range segments[:len(segments)-1] {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}

			child := this.ensureImportedGroup(version, parent, segment, groupCache)
			if child <= 0 {
				result.Errors = append(result.Errors, "创建目录失败："+segment)
				parent = 0
				break
			}
			parent = child
		}

		meta, body := parseDocsFrontMatter(string(entry.content))
		this.upsertImportedDoc(version, parent, segments[len(segments)-1], body, meta, overwrite, result, sort)
		sort++
	}

	// 目录数 = 缓存里实际创建的条目数
	for _, id := range groupCache {
		if id > 0 {
			result.Groups++
		}
	}
}

// ensureImportedGroup - 按 版本 + 父级 + 名称 找目录，没有则创建
func (this *Docs) ensureImportedGroup(version string, pid int, name string, cache map[string]int) int {
	slug := docsSlugify(name)
	key := fmt.Sprintf("%d:%s", pid, slug)

	if id, ok := cache[key]; ok {
		return id
	}

	var exist model.DocsGroup
	facade.DB.Model(&model.DocsGroup{}).
		Where("version", version).
		Where("pid", pid).
		Where("slug", slug).
		Scan(&exist)

	if exist.Id > 0 {
		cache[key] = exist.Id
		return exist.Id
	}

	var last model.DocsGroup
	facade.DB.Model(&model.DocsGroup{}).
		Where("version", version).
		Where("pid", pid).
		Order("sort desc").
		Scan(&last)

	sort := 1
	if last.Id > 0 {
		sort = last.Sort + 1
	}

	item := model.DocsGroup{
		Pid:     pid,
		Version: version,
		Slug:    slug,
		Name:    name,
		Sort:    sort,
		Status:  1,
		Visible: 1,
	}

	if _, err := facade.DB.Model(&item).Create(&item); err != nil {
		return 0
	}

	cache[key] = item.Id
	return item.Id
}

// upsertImportedDoc - 创建或更新一篇导入的文档
func (this *Docs) upsertImportedDoc(version string, gid int, filename, body string, meta map[string]string, overwrite bool, result *docsImportResult, sort int) {
	title := strings.TrimSpace(meta["title"])
	if title == "" {
		title = docsTitleFromBody(body)
	}
	if title == "" {
		title = strings.TrimSuffix(path.Base(filename), path.Ext(filename))
	}

	slug := strings.TrimSpace(meta["slug"])
	if slug == "" {
		slug = docsSlugify(path.Base(filename))
	}

	status := 1
	if value := strings.TrimSpace(meta["status"]); value != "" {
		status = cast.ToInt(value)
		if status != 0 && status != 1 {
			status = 1
		}
	}

	if sort <= 0 {
		sort = cast.ToInt(meta["sort"])
	}

	var exist model.Docs
	facade.DB.Model(&model.Docs{}).
		Where("version", version).
		Where("gid", gid).
		Where("slug", slug).
		Scan(&exist)

	if exist.Id > 0 {
		if !overwrite {
			result.Skipped++
			return
		}

		data := map[string]any{
			"title":       title,
			"content":     body,
			"status":      status,
			"sort":        docsIfZero(sort, exist.Sort),
			"update_time": time.Now().Unix(),
		}

		if _, err := facade.DB.Model(&model.Docs{}).Where("id", exist.Id).Update(data); err != nil {
			result.Errors = append(result.Errors, "更新失败："+filename+"（"+err.Error()+"）")
			return
		}

		result.Updated++
		return
	}

	item := model.Docs{
		Gid:         gid,
		Version:     version,
		Slug:        slug,
		Title:       title,
		Content:     body,
		Sort:        sort,
		Status:      status,
		Visible:     1,
	}

	if status == 1 {
		item.PublishTime = time.Now().Unix()
	}

	if _, err := facade.DB.Model(&item).Create(&item); err != nil {
		result.Errors = append(result.Errors, "创建失败："+filename+"（"+err.Error()+"）")
		return
	}

	result.Created++
}

/* ------------------------------ 解析工具 ------------------------------ */

// parseDocsFrontMatter - 解析 --- 包裹的 front-matter，返回元数据与正文
func parseDocsFrontMatter(content string) (map[string]string, string) {
	content = strings.TrimPrefix(content, "\ufeff")
	meta := make(map[string]string)

	if !strings.HasPrefix(strings.TrimLeft(content, " \t\r\n"), "---") {
		return meta, content
	}

	lines := strings.Split(content, "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			start = i
			break
		}
	}
	if start < 0 {
		return meta, content
	}

	end := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return meta, content
	}

	for _, line := range lines[start+1 : end] {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		index := strings.Index(line, ":")
		if index <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:index]))
		value := strings.Trim(strings.TrimSpace(line[index+1:]), `"'`)
		meta[key] = value
	}

	return meta, strings.Join(lines[end+1:], "\n")
}

// docsTitleFromBody - 取正文里第一个一级标题作为标题
func docsTitleFromBody(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

var docsSlugPattern = regexp.MustCompile(`[^a-z0-9]+`)

// docsSlugify - 文件名 / 目录名转 slug（中文等非 ASCII 字符用 hash 兜底）
func docsSlugify(name string) string {
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	value := strings.ToLower(strings.TrimSpace(base))
	value = docsSlugPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")

	if value != "" {
		return value
	}

	hash := utils.Hash.Sum32(base)
	if len(hash) > 8 {
		hash = hash[:8]
	}
	return "doc-" + hash
}

// docsZipEntry - 压缩包内的一篇 Markdown
type docsZipEntry struct {
	rel     string
	content []byte
}

// commonZipRoot - 判断压缩包的顶层目录是不是「包裹目录」，需要剥掉
//
// 这里不能简单地「所有文件都在同一个顶层目录下就剥掉」：
//   - my-docs/guide/a.md → my-docs 是包裹目录，应剥掉，保留 guide
//   - guide/a.md         → guide 是业务目录，不能剥，否则所有文档都会堆到根上
//
// 判断规则：
//  1. 顶层目录名是 docs / doc / documentation / markdown / md 这类常见包裹名 → 直接剥；
//  2. 否则只有「顶层目录下没有任何直接文件、全是子目录」时才视为包裹目录。
func commonZipRoot(entries []docsZipEntry) string {
	if len(entries) == 0 {
		return ""
	}

	segments := strings.Split(entries[0].rel, "/")
	if len(segments) < 2 {
		return ""
	}

	candidate := segments[0] + "/"

	// 所有条目必须都在同一个顶层目录下
	for _, item := range entries {
		if !strings.HasPrefix(item.rel, candidate) {
			return ""
		}
	}

	// 常见包裹目录名直接剥
	wrapperNames := map[string]bool{
		"docs": true, "doc": true, "documentation": true,
		"markdown": true, "md": true, "content": true,
	}
	if wrapperNames[strings.ToLower(segments[0])] {
		return candidate
	}

	// 顶层目录下如果直接放着 md 文件，说明它是业务目录（或用户没建外层），不能剥
	for _, item := range entries {
		rest := strings.TrimPrefix(item.rel, candidate)
		if rest != "" && !strings.Contains(rest, "/") {
			return ""
		}
	}

	return candidate
}

func docsIfZero(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
